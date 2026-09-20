# Prompt — Sprint 02 · Identity & auth

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-02.md`](../sprints/sprint-02.md)   ·   **Milestone:** M1 — OAuth login works end-to-end   ·   **Prereqs:** [S01](../sprints/sprint-01.md)

## Read first
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions: match `theme.css`, respect service boundaries, conventional commits, do not commit/push unless asked.
- [`../../adr/0006-authn-authz.md`](../../adr/0006-authn-authz.md) — the auth model: OAuth -> opaque server-session cookie; gateway-minted RS256 JWT; JWKS verify. No passwords.
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — one `xlearndb`, schema-per-service, per-service role; goose (embedded, startup + advisory lock) + sqlc/pgx.
- [`../../adr/0009-deployment-and-gitops.md`](../../adr/0009-deployment-and-gitops.md) — mirror `../infra`: `charts/project` HelmRelease, GHCR image, Flux image-automation, SOPS secrets, pull-based (no `kubectl`).
- [`../../architecture/services.md`](../../architecture/services.md) — identity + gateway responsibilities, endpoints, owned schema, events.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `identity` tables (`account`, `oauth_identity`, `session`, `onboarding`, `outbox`).
- [`../../architecture/api.md`](../../architecture/api.md) — the BFF Auth & account surface + the 401 envelope convention.
- `design-system/screens/Auth.dc.html` — layout/copy/interaction intent for the Auth screen + 3-step onboarding (design reference only, not runnable).
- [`../../../design-system/README.md`](../../../design-system/README.md) + `design-system/theme.css` — tokens/components to reuse verbatim (dark "landscape console" look).
- Sibling `infra/infrastructure/database/README.md` — the 4-step DB pattern (role-password secret -> managed role -> `Database` CR -> app credentials secret). Copy `infra/apps/airlift.yaml` as the HelmRelease template; `infra/apps/image-automation.yaml` for the image-automation entry.

## Context
S01 delivered a deployable gateway serving the React shell at `/xlearn` (M0), with all 12 routes as
stubs and the GitOps path proven. Nothing authenticates yet and no internal service or database is
wired. This sprint stands up the first internal service (`identity`) and the shared `xlearndb`,
makes OAuth login real end-to-end, and lays the JWT/JWKS propagation pattern every later service reuses
(carries M1).

## Do this (in order)
1. **identity service + xlearndb wiring** — Scaffold `cmd/identity` + `internal/identity`. Create schema
   `identity` with `account`, `oauth_identity` (`unique(provider, provider_user_id)`), `session`
   (opaque id, `expires_at`, `revoked_at`), `onboarding` (`path_chosen`, `budget_set`, `key_added`,
   `completed_at`), and an `outbox` table — see `data-model.md` schema `identity`. Migrations: goose
   plain SQL under `internal/identity/store/migrations/`, embedded (`//go:embed`), applied on startup
   inside a Postgres advisory lock; refuse to serve on migration failure. Queries: sqlc + pgx (commit
   generated output; CI runs `sqlc diff`). Add `/healthz` and `/readyz` (readyz pings the DB) and slog
   JSON logs to stdout. Wire `xlearndb` via the infra 4-step pattern: `pg-xlearn-identity.enc.yaml`
   role-password SOPS secret + managed role `xlearn_identity` in
   `infra/infrastructure/database/cluster/cluster.yaml`; the `xlearndb` `Database` CR (owner = the
   bootstrap `xlearn` role) if absent; app secret `infra/apps/secrets/xlearn-db.enc.yaml`
   (`PGUSER`/`PGPASSWORD` for the identity role); set the role `search_path` to `identity`. Add
   `infra/apps/xlearn-identity.yaml` (copy `airlift.yaml`; ClusterIP, `route.enabled: false`;
   `PGHOST=projects-pgstore-rw.databases.svc.cluster.local`, `PGPORT=5432`, `PGDATABASE=xlearndb` plain
   env; `envFrom` the db secret) + an ImageRepository/ImagePolicy (`>=0.1.0`) with the
   `# {"$imagepolicy": "flux-system:xlearn-identity:tag"}` setter in `infra/apps/image-automation.yaml`.
   Add `deploy/identity.Dockerfile` (non-root, read-only rootfs) and a CI path filter for identity.
2. **OAuth (GitHub + Google) + server session** — Implement the OAuth2/OIDC authorization-code flow for
   both providers with state (CSRF) + PKCE. `POST /auth/{provider}/start` -> 302 to the provider;
   `GET /auth/{provider}/callback` exchanges the code, upserts `account` (match
   `oauth_identity(provider, provider_user_id)`; create `account` + `onboarding` on first sign-in),
   creates a `session` row, and sets an HttpOnly Secure SameSite=Lax cookie (opaque session id) scoped
   to path `/xlearn`. On first account creation, write the `xlearn.identity.account_created` outbox row
   in the same transaction. `POST /sessions/revoke` (via `POST /auth/logout`) sets `revoked_at` and
   clears the cookie. Store no passwords. OAuth client id/secret from SOPS secret
   `infra/apps/secrets/xlearn-oauth.enc.yaml` (mounted via `envFrom`); the callback URL must match the
   `/xlearn` base + each provider app config.
3. **JWT/JWKS + gateway auth middleware** — Load an RSA signing keypair from SOPS secret `xlearn-jwt`
   (`infra/apps/secrets/xlearn-jwt.enc.yaml`) mounted to the gateway. Gateway middleware on
   `/xlearn/api/*`: validate the session cookie (`POST /sessions/validate` on identity or shared
   verify), then mint a ~5-minute RS256 JWT (`sub` = account id, `roles` = `["learner"]`, per-downstream
   `aud`) and forward it on internal HTTP/JSON calls. identity serves JWKS at
   `/.well-known/jwks.json`. Implement `internal/platform/auth`: JWKS fetch + cache by `kid`, verify
   signature/`exp`/`aud`, ~30s clock-skew leeway, `kid`-overlap tolerance for rotation. Unauthenticated
   -> `401 { "error": { "code": "unauthenticated" } }`. Demonstrate verify end-to-end on identity or a
   stub protected route.
4. **Auth screen + onboarding step 1 (path)** — Wire `design-system/screens/Auth.dc.html` into the SPA
   with `theme.css` verbatim (no Tailwind): OAuth buttons call `POST /auth/{provider}/start`; render the
   3-step onboarding shell with step 1 ("Pick your path") active (DSA selectable; other paths "Coming
   soon"); "Continue" persists `onboarding.path_chosen` via `POST /onboarding/step`. Implement BFF
   `GET /me` (account + onboarding state; drives route gating + the unauth redirect), `POST /auth/logout`,
   and `POST /onboarding/step`. Finalize `xlearn.identity.account_created` emission through the outbox
   relay (stream `XLEARN_IDENTITY` reserved; no consumer yet). Leave onboarding steps 2-3 and the
   email/password inputs inert (OAuth-only; deferred to S10/S11).

## Constraints
- Reuse `design-system/theme.css` verbatim (tokens/components; dark theme); no Tailwind or ad-hoc CSS frameworks.
- Match `../infra` conventions exactly: copy `apps/airlift.yaml` as the HelmRelease template; GHCR image `ghcr.io/sujaykumarsuman/xlearn-identity`; Flux image-automation with the `# {"$imagepolicy": "flux-system:xlearn-identity:tag"}` setter; ImagePolicy semver `>=0.1.0`; SOPS/age secrets under `apps/secrets/*.enc.yaml`. Pull-based GitOps — never `kubectl apply` by hand.
- One `xlearndb`, schema-per-service, per-service DB role via the 4-step pattern; goose embedded migrations (startup + advisory lock); sqlc/pgx (no ORM); the identity role touches only schema `identity`.
- Store no passwords anywhere; the session cookie is opaque + revocable (HttpOnly Secure SameSite=Lax, path `/xlearn`). Internal auth is the gateway-minted short-TTL RS256 JWT, verified via JWKS — no `identity` call on the hot path.
- Emit `xlearn.identity.account_created` via the transactional outbox (write domain row + outbox row in one tx; relay publishes). Route only on gateway; identity is ClusterIP (`route.enabled: false`).
- Pod hardening: inherit chart defaults (non-root, read-only rootfs, dropped caps). Structured slog JSON logs to stdout.
- Do not commit or push unless asked.

## Deliverables
- `cmd/identity` + `internal/identity` (handlers, sqlc store, goose migrations for schema `identity`), with `/healthz` + `/readyz`.
- OAuth authorization-code flow (GitHub + Google), server-side `session` rows, and the `/xlearn`-scoped session cookie; logout revocation.
- Gateway auth middleware (cookie -> validate -> mint JWT -> forward) + identity JWKS endpoint + shared `internal/platform/auth` verifier.
- Wired Auth screen + onboarding step 1; BFF `GET /me`, `POST /auth/logout`, `POST /onboarding/step`; `xlearn.identity.account_created` outbox emission.
- `deploy/identity.Dockerfile` + CI path filter; infra additions: `infra/apps/xlearn-identity.yaml`, image-automation entry, and the 4-step DB wiring + SOPS secrets `xlearn-db` / `xlearn-oauth` / `xlearn-jwt`.

## Update status
- As each task lands, set its row in [`../sprints/sprint-02.md`](../sprints/sprint-02.md) to ✅ (🔄 while in progress); set _Overall_ when all tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the **Milestones** table (M1). Add a **Decisions log** line for any notable call.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)
- [ ] A user signs in with GitHub and with Google; a session cookie is set; refreshing keeps them logged in; logout revokes the session server-side.
- [ ] Unauthenticated requests to app routes redirect to OAuth; `GET /me` returns the account + onboarding state (including `path_chosen` after step 1).
- [ ] Downstream calls carry a gateway-minted JWT; a service verifies it via JWKS (demonstrated on identity or a stub protected route).
- [ ] identity is deployed to prod via Flux as ClusterIP; the `xlearndb` `identity` schema is migrated on startup; secrets are via SOPS. **(M1)**
- Do not commit or push unless asked.
