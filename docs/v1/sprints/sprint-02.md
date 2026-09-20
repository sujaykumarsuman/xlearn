# Sprint 02 — Identity & auth

> **Milestone:** M1 — OAuth login works end-to-end   ·   **Design phase:** D1 Foundation
> **Prereqs:** [S01](sprint-01.md)   ·   **Unblocks:** [S03](sprint-03.md), [S05](sprint-05.md), [S10](sprint-10.md)
> **Execute with:** [`../prompts/prompt-s02.md`](../prompts/prompt-s02.md) — one prompt, one session.

## Status

_Overall:_ 🔄 Merged + **deployed to prod via Flux** (`xlearn-identity`/`xlearn-gateway:0.1.7`) and
verified live: gateway JWKS served, `/me`→401 envelope, identity up (OAuth `start`→302 with PKCE +
correct callback), `xlearndb` `identity` schema migrated on startup. **Login pending only the real
GitHub OAuth client id/secret** in `xlearn-oauth.enc.yaml` (currently `CHANGE-ME`).

| # | Task | Status |
|---|------|--------|
| 1 | identity service + xlearndb wiring | ✅ |
| 2 | OAuth (GitHub + Google) + server session | ✅ |
| 3 | JWT/JWKS + gateway auth middleware | ✅ |
| 4 | Auth screen + onboarding step 1 (path) | ✅ |

> **Built & verified this session (not yet deployed):** `cmd/identity` + `internal/identity` (goose
> migrations, sqlc/pgx store, OAuth GitHub w/ state+PKCE — Google deferred, opaque sessions, outbox), the shared
> `internal/platform/auth` RS256/JWKS mint+verify (stdlib), the gateway BFF (`/me`, `/auth/logout`,
> `/onboarding/step`, OAuth proxy, JWKS publication), the standalone Auth screen + onboarding step 1,
> and all infra (`deploy/identity.Dockerfile`, CI path filter + `sqlc diff`, HelmRelease, 4-step DB
> wiring, SOPS secrets). The full mint→JWKS-verify loop and step-1 persistence were exercised live
> against a Dockerised Postgres and in the browser. **Remaining for M1:** register the GitHub/Google
> OAuth apps and fill `xlearn-oauth.enc.yaml`, then merge (xlearn + infra) so Flux deploys.

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M1 milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Make login real and stand up the first internal service plus the shared-database wiring every later
service reuses. A user signs in with GitHub or Google, gets a revocable server-side session cookie
scoped to `/xlearn`, and lands in a 3-step onboarding whose first step (choose path) is persisted.
This sprint also lays the auth propagation pattern — a gateway-minted short-TTL RS256 JWT verified by
downstream services via JWKS — carrying M1 (real login) and establishing the DB + auth conventions for
S03/S05/S10.

## Scope

**In**
- `identity` service (`cmd/identity`, `internal/identity`) owning schema `identity` with tables
  `account`, `oauth_identity`, `session`, `onboarding` (goose embedded migrations + sqlc/pgx).
- `xlearndb` wiring via the infra 4-step DB pattern: managed role `xlearn_identity`, role-password SOPS
  secret, the `xlearndb` `Database` CR (owner bootstrap role), and app credentials secret
  `infra/apps/secrets/xlearn-db.enc.yaml`; `PGHOST/PGPORT/PGDATABASE` as plain env.
- OAuth 2.0 / OIDC authorization-code flow for GitHub and Google; server-side `session` rows behind an
  HttpOnly Secure SameSite=Lax cookie (opaque id, path `/xlearn`); `session/revoke` on logout.
- RS256 signing key from SOPS secret `xlearn-jwt`; gateway validates the session cookie, mints a
  ~5-minute JWT (`sub`, `roles`, `aud`) forwarded on internal calls; JWKS at
  `/.well-known/jwks.json`; shared `internal/platform/auth` verify-via-JWKS helper.
- BFF endpoints `GET /me`, `POST /auth/{provider}/start`, `GET /auth/{provider}/callback`,
  `POST /auth/logout`, `POST /onboarding/step`; unauthenticated app routes -> 401 -> SPA redirect to OAuth.
- Wire `Auth.dc.html`: OAuth buttons, the 3-step onboarding shell, and step 1 (choose path); persist
  `onboarding.path_chosen`.
- `xlearn.identity.account_created` emitted via the transactional outbox (stream reserved; no consumer yet).
- Infra: `infra/apps/xlearn-identity.yaml` HelmRelease (ClusterIP, `route.enabled: false`) + an
  image-automation entry; Dockerfile `deploy/identity.Dockerfile`; a CI path filter for identity.

**Out (later sprints)**
- Profile / study budget / timezone editing and full onboarding steps 2-3 — Settings in [S10](sprint-10.md).
- Coach API-key storage (onboarding step 3 "power up your coach") — coach in [S11](sprint-11.md).
- The `Auth.dc.html` email/password form is not wired: xLearn stores no passwords (OAuth-only, per
  [ADR-0006](../../adr/0006-authn-authz.md)); those inputs stay inert in v1.

## Tasks

### 1 · identity service + xlearndb wiring
Scaffold `cmd/identity` (main + config from env) and `internal/identity` (handlers, store, migrations).
Create schema `identity` with `account` (`id`, `display_name`, `email`, `timezone`,
`study_budget_json`, `reminders_json`, `created_at`), `oauth_identity` (`id`, `account_id`, `provider`
`github`/`google`, `provider_user_id`, `unique(provider, provider_user_id)`), `session`
(`id` opaque, `account_id`, `created_at`, `expires_at`, `revoked_at`), and `onboarding` (`account_id`,
`path_chosen`, `budget_set`, `key_added`, `completed_at`) plus an `outbox` table — all under
[data-model.md](../../architecture/data-model.md) schema `identity`. Use goose plain-SQL migrations
under `internal/identity/store/migrations/`, embedded via `//go:embed`, applied on startup inside a
Postgres advisory lock ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)); the service
refuses to serve if migration fails. Generate queries with sqlc + pgx (committed output; CI `sqlc diff`).
Add `/healthz` (liveness) and `/readyz` (readiness — pings the DB). Log JSON via slog to stdout.
Wire `xlearndb` through the infra 4-step pattern (`infra/infrastructure/database/README.md`): a
role-password SOPS secret `infra/infrastructure/database/cluster/pg-xlearn-identity.enc.yaml`, the
managed role `xlearn_identity` added to `infra/infrastructure/database/cluster/cluster.yaml`, the
`xlearndb` `Database` CR (owner = the bootstrap `xlearn` role) if not already present, and the app
credentials SOPS secret `infra/apps/secrets/xlearn-db.enc.yaml` (`PGUSER`/`PGPASSWORD` for the identity
role). Set the role `search_path` to schema `identity` so it touches only its own schema. Add the
HelmRelease `infra/apps/xlearn-identity.yaml` (copy `infra/apps/airlift.yaml`: `charts/project`,
ClusterIP, `route.enabled: false`, `PGHOST=projects-pgstore-rw.databases.svc.cluster.local`,
`PGPORT=5432`, `PGDATABASE=xlearndb`, `envFrom` the db secret) and an ImageRepository/ImagePolicy pair
(`>=0.1.0`) in `infra/apps/image-automation.yaml` with the `# {"$imagepolicy": "flux-system:xlearn-identity:tag"}`
setter on the tag. Add `deploy/identity.Dockerfile` (read-only rootfs, non-root) and a CI path filter
so only identity rebuilds on identity changes.

### 2 · OAuth (GitHub + Google) + server session
Implement the OAuth 2.0 / OIDC authorization-code flow for both providers. `POST /auth/{provider}/start`
generates state (CSRF) + PKCE and returns/redirects (302) to the provider authorize URL; the callback
URL must match `projects.sujaykumar.dev/xlearn` and each provider app's registered redirect.
`GET /auth/{provider}/callback` exchanges the code, reads the provider user id + profile, upserts
`account` (match on `oauth_identity(provider, provider_user_id)`; create the `account` + `onboarding`
rows on first sign-in), creates a `session` row (opaque id, `expires_at`), and sets the HttpOnly Secure
SameSite=Lax cookie scoped to path `/xlearn`. On first account creation, write the
`xlearn.identity.account_created` outbox row in the same transaction (task 4 finalizes emission).
`session/revoke` (invoked by `POST /auth/logout`) sets `revoked_at` and clears the cookie. No passwords
are stored anywhere. OAuth client id/secret for both providers come from the SOPS secret
`infra/apps/secrets/xlearn-oauth.enc.yaml`, mounted to identity via `envFrom`. See
[ADR-0006](../../adr/0006-authn-authz.md) and the Auth surface in
[api.md](../../architecture/api.md#auth--account).

### 3 · JWT/JWKS + gateway auth middleware
Load an RSA signing keypair from the SOPS secret `xlearn-jwt` (`infra/apps/secrets/xlearn-jwt.enc.yaml`),
mounted to the gateway. On every `/xlearn/api/*` request the gateway validates the session cookie
(calling `POST /sessions/validate` on identity, or a shared verify) and, on success, mints a short-TTL
(~5 min) RS256 JWT with `sub` (account id), `roles` (`["learner"]`), and per-downstream `aud`, then
forwards it on internal HTTP/JSON calls. identity publishes JWKS at `/.well-known/jwks.json`
(surfaced through the gateway internally). Implement the verifier in shared `internal/platform/auth`:
fetch + cache JWKS by `kid`, verify signature/`exp`/`aud`, tolerate small clock skew (leeway ~30s), and
support `kid` overlap during rotation. Unauthenticated -> `401 { "error": { "code": "unauthenticated" } }`
so the SPA redirects to OAuth ([api.md conventions](../../architecture/api.md#conventions)).
Demonstrate end-to-end verification on identity itself or a stub protected route. Auth model:
[ADR-0006](../../adr/0006-authn-authz.md); deploy/secret conventions:
[ADR-0009](../../adr/0009-deployment-and-gitops.md).

### 4 · Auth screen + onboarding step 1 (path)
Wire `design-system/screens/Auth.dc.html` into the SPA using `design-system/theme.css` verbatim (dark
"landscape console" look; no Tailwind): the "Continue with GitHub" / "Continue with Google" buttons hit
`POST /auth/{provider}/start`; the 3-step onboarding shell renders with step 1 ("Pick your path")
active — DSA Interview Mastery selectable, the other paths shown as "Coming soon". "Continue" persists
`onboarding.path_chosen` via `POST /onboarding/step`. Implement the BFF `GET /me` (current account +
onboarding state, used to gate routing and drive the redirect), `POST /auth/logout`, and
`POST /onboarding/step`. Finalize emission of `xlearn.identity.account_created` through the outbox relay
(stream `XLEARN_IDENTITY` reserved; no consumer yet). Steps 2-3 of onboarding render as inert shell only
(deferred to [S10](sprint-10.md)/[S11](sprint-11.md)). The email/password inputs in the artboard stay
inert (OAuth-only). Screen intent: [Auth.dc.html]; account/onboarding fields:
[data-model.md](../../architecture/data-model.md#schema-identity).

## Acceptance criteria
- 🔶 A user signs in with GitHub (Google deferred); a session cookie is set; refreshing keeps them
      logged in; logout revokes the session server-side. **Verified end-to-end locally** (real Postgres
      + browser); in prod it needs only the real GitHub OAuth client id/secret (placeholders today).
- [x] Unauthenticated requests to app routes redirect to OAuth; `GET /me` returns the account plus
      onboarding state (including `path_chosen` after step 1). (`/me`→401 envelope verified live.)
- [x] Downstream calls carry a gateway-minted JWT; a service verifies it via JWKS (demonstrated on
      identity), tolerating small clock skew and `kid` overlap. (JWKS served live; overlap-publish added.)
- [x] identity is deployed to prod via Flux as a ClusterIP service (`route.enabled: false`); the
      `xlearndb` `identity` schema is migrated on startup; all secrets are via SOPS. **(M1)**

## Definition of Done
CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs
- JWKS distribution and key rotation (`kid` overlap during a rotation window) must be correct in the
  shared `internal/platform/auth` — a stale or missing `kid` silently 401s every internal call.
- Clock skew across pods vs the ~5-minute JWT TTL: rely on node NTP and allow a small verification leeway.
- Cross-namespace CNPG secret: the identity role password appears in two SOPS files (the `databases`
  namespace role-password secret and the `xlearn`-namespace app secret) and both must hold the same value.
- OAuth callback URL must match the `/xlearn` base and each provider app's registered redirect exactly,
  or the authorization-code exchange fails.
- Cookie scope/attributes: HttpOnly + Secure + SameSite=Lax + path `/xlearn` must be exact, or sessions
  leak across paths or fail to send on same-origin SPA calls.
