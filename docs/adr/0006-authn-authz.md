# ADR-0006 — AuthN/AuthZ

- **Status:** Accepted. **Refined by [ADR-0033](0033-invite-only-admission-and-owner-admin.md) (v2, Proposed):** roles are enforced (`RequireRole`, `public-read`), and `account.role` (learner, tester, owner) lives in the DB, never in the JWT.
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0007](0007-ai-coach-byo-key-and-secrets.md)

## Context

The Auth screen offers **OAuth (GitHub / Google)** and a 3-step onboarding. xLearn stores no
passwords (a safety win and less to secure). We need: a way to authenticate users at the edge,
propagate identity to internal services, and let services trust calls without a heavy mesh — all on
one node, solo-operated.

## Decision

### End users: OAuth → server session cookie

- **`identity`** service implements the **OAuth 2.0 / OIDC Authorization Code flow** with GitHub and
  Google. No local passwords.
- On success it creates a **server-side session** and sets an **HttpOnly, Secure, SameSite=Lax**
  cookie scoped to `projects.sujaykumar.dev` path `/xlearn` (opaque session id; session record in the
  `identity` schema). The **gateway** validates the cookie on every request.
- The cookie is opaque (not a JWT) so sessions are **revocable** server-side and no user token lives in JS.

### Internal propagation: gateway-minted short-lived JWT

- The **gateway** exchanges the validated session for a **short-TTL JWT** (RS256, ~5 min) carrying
  `sub` (user id), `roles`, and an audience per downstream service, and forwards it on internal calls.
- Downstream services **verify** the JWT against the gateway's **JWKS** (published at an internal
  endpoint / mounted public key). No service calls `identity` on the hot path.
- Signing key: an RSA keypair from a **SOPS-managed secret** (`xlearn-jwt`), mounted to the gateway;
  public JWKS distributed to services. Rotation = new kid, overlap window.

### Authorization

- v1 is **single-role** (`learner`) + ownership checks: a service only ever operates on rows scoped
  to the `sub` in the JWT. No admin UI in v1 (curriculum is seeded, not edited).
- Service-to-service: the JWT `aud` + NetworkPolicy (only gateway may call most services; only
  `review`/`assessment` consume NATS) provide defence in depth. Broker auth via NATS credentials
  (SOPS secret) later; v1 relies on cluster-internal isolation.

## Consequences

- ✅ No passwords stored; sessions revocable; no tokens in browser JS.
- ✅ Services are stateless verifiers (JWKS) — no auth hot-path call to `identity`.
- ✅ Consistent with the platform's SOPS secret handling.
- ⚠️ JWKS distribution + key rotation must be implemented carefully (shared `internal/platform/auth`).
- ⚠️ Clock skew across pods matters for a 5-min TTL — rely on node NTP; allow small leeway.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Stateless JWT session cookie (no server session)** | Not revocable without a denylist; puts a longer-lived token where XSS could reach it. |
| **Passwords + local accounts** | More attack surface (hashing, reset flows, breaches); OAuth-only is simpler and safer, and matches the design. |
| **Full service mesh (mTLS, e.g. Linkerd)** | Real per-call identity, but a heavy control plane for one node + solo ops. |
| **Opaque token introspection on every internal call** | Adds an `identity` round-trip to every request; JWKS verification avoids it. |
| **Third-party IdP (Auth0/Clerk)** | External dependency + cost for a personal project; GitHub/Google OAuth direct is enough. |
