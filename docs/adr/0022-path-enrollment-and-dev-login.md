# ADR-0022 — Per-user path enrollment + the local dev login

- **Status:** Accepted
- **Date:** 2026-09-22
- **Deciders:** @sujaykumarsuman
- **Related:** [0006](0006-authn-authz.md) (authn/authz — this adds a strictly-gated local login path), [0005](0005-data-ownership-and-migrations.md) (identity owns the new table), [0013](0013-bff-week-aggregation-userstate-contract.md) (BFF agg pattern). Tracks the post-1.0 UI/UX feedback batch in [`docs/v1/feedbacks/`](../v1/feedbacks/).

## Context

Two post-1.0 changes need a decision recorded:

1. **Starting a path (F002).** v1 shipped DSA as globally `status:'active'` — effectively "started for
   everyone". The feedback is to make starting a path an explicit, per-user action, and to show a
   learner's **current day**, **streak** and **what's scheduled today** once started.
2. **Reviewing locally (F002 support).** The app is OAuth-only ([ADR-0006](0006-authn-authz.md)); there
   is no way to reach an authenticated session from a local `docker-compose` without registering a
   `localhost` OAuth callback. A reviewer needs to log in to see the UI.

## Decision

### 1. Enrollment is per-user state owned by identity

- A new `identity.path_enrollment(account_id, path_slug, status, started_at)` table (one row per
  account+path). `POST /paths/{slug}/start` is **JWT-scoped and idempotent** (`ON CONFLICT DO UPDATE
  SET status='active'` — a repeat start never moves `started_at`, so "current day" can't reset).
- Enrollments ride **`GET /me`** so every screen reads started-state from the one `me` fetch. The
  gateway proxies the start endpoint and mints the identity-audience JWT, like the other identity BFF
  calls.
- **Current day** is derived (`floor(days since started_at) + 1`, learner-local). **Streak** and
  **today's scheduled** are NOT re-derived — the Catalog + Dashboard reuse the existing `/dashboard`
  agg. No new aggregation, no new event.
- Onboarding's `path_chosen` is left as an interest signal; it is **not** auto-promoted to an
  enrollment (keeping "choose in onboarding" and "start on the Catalog" as distinct, explicit steps).
  Revisit if the double step feels redundant.
- The global seed `status:'active'` now reads as *"available to start"*, not *"pre-started"*.

### 2. Dev login is a strictly env-gated, local-only endpoint

- `identity` exposes `POST /auth/dev/login` and `GET /auth/dev/enabled`, both **404 unless `DEV_AUTH`
  is truthy**. `DEV_AUTH` is set only in the `docker-compose` stack and **must never** be set in a
  prod image. The prod route table still registers them (they just refuse), so there is no
  build-time divergence — the gate is the single env var.
- Dev login mints a session for a **fixed local account** (reusing `FindOrCreateAccount` with a
  `github`/`dev-local` identity — no new provider enum), completes its onboarding so the reviewer
  lands in the app, and sets the same session cookie the OAuth callback does. The SPA shows a "Dev
  sign in (local)" button only when `/auth/dev/enabled` says so.
- This adds no new trust surface in prod: the endpoints are inert without `DEV_AUTH`, they only ever
  mint the one fixed local identity, and they never touch OAuth data or real accounts.

## Consequences

- **Positive:** durable, cross-device, multi-path-ready enrollment; a frictionless, prod-safe local
  review login; no new events/aggregations (reuses `/me` + `/dashboard`).
- **Trade-offs:** a real (if gated) auth code path exists in the binary — mitigated by the single
  env gate defaulting off, the fixed identity, and the SPA button being server-gated. The
  `/dashboard` agg stays DSA-pinned in the gateway (fine while DSA is the only real path).
- **Local review** is documented in [`deploy/local/README.md`](../../deploy/local/README.md); the
  compose stack is a review aid only — production remains GitOps via `../infra` + Flux.
