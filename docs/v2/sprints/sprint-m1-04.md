# Sprint m1-04 — Identity security floor: roles, sessions, admin CLI, DEV_AUTH guard, L3, CSP (M1b)

> **Milestone:** M1 — spine (**M1b**) · **Track:** product · **Order:** 21
> **Prereqs:** [m1-03](sprint-m1-03.md) (merged; gateway router serialized) · [m1-02](sprint-m1-02.md) (`v1.6.0` live: the M1a identity columns)
> **Unblocks:** [m1-05](sprint-m1-05.md) · [m1-07](sprint-m1-07.md) · [l-02](sprint-l-02.md)
> **Release action:** **merge only** (ships in `v1.7.0`, tagged by [m1-07](sprint-m1-07.md))
> **Calendar:** weeks 2–3 (≈ 2026-10-12 → 10-16). **Prepares owner events:** `ev-owner-role` (after `v1.7.0`), `ev-first-tester` (after MI-5b is live)
> **Execute with:** [`../prompts/prompt-m1-04.md`](../prompts/prompt-m1-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Sessions: status join, validate returns role/status/accepted/created_at, revoke-all | X | ⬜ |
| 2 | `auth.RequireRole` on every user route (JWTs stay `["learner"]`) + gateway cohort helper | X | ⬜ |
| 3 | L7 guard: `SIGNUP_MODE=open` honoured only with `DEV_AUTH` | X | ⬜ |
| 4 | L3: bcrypt semaphore (≤ 2 in flight → 429) + dummy hash | X | ⬜ |
| 5 | `identity admin` CLI (`account …`, `seats`) + `admin_audit` | X | ⬜ |
| 6 | CSP + JSON / `Sec-Fetch-Site` checks on mutating calls | X | ⬜ |
| 7 | Cohort `preview`: enrollment + gateway course visibility | X | ⬜ |
| 8 | Runbook `docs/runbooks/identity-admin.md` (+ owner events) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] `v1.6.0` live — `account.role`, `status`, `admitted_via`, `accepted_at`, `admin_audit` exist in production ([m1-02](sprint-m1-02.md))
- [ ] [m1-03](sprint-m1-03.md) merged (the M1b gateway-router edits are serialized m1-03 → m1-04 → m1-05 → m1-06)
- [ ] No open peer PR edits `internal/identity/` or `internal/gateway/gateway.go`/`bff.go` (`gh pr list`, `git worktree list`, ListAgents)

## Goal

Give identity the v2 **authz floor** ([ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md)): roles and status
live in identity's DB and are returned by session-validate (never minted into a JWT); sessions die at once on suspend and
are revocable wholesale; services enforce `RequireRole("learner")`; the owner runs a distroless-safe `identity admin` CLI
whose every verb is audited; identity refuses `SIGNUP_MODE=open` without `DEV_AUTH` (the L7 guard `v1.5.2` didn't carry);
bcrypt can't be used to stall `/sessions/validate` or to time-probe identifiers (L3); and xLearn serves a strict CSP and
rejects cross-site writes from sibling `*.sujaykumar.dev` hosts.

## Scope

**In**
- Session-validate returns `role`, `status`, `accepted`, `created_at`; `status='active'` join; revoke-all on suspend,
  password change and the `revoke-sessions` verb.
- `auth.RequireRole` in `internal/platform/auth`, used by every service's user routes; the gateway cohort helper (T-3).
- L7 in `internal/identity/config.go`; L3 in `internal/identity/password.go` + `authemail.go`.
- `identity admin account list|suspend|reactivate|revoke-sessions|set-role|create --role tester` and `seats`, each writing `admin_audit`.
- Gateway CSP + `Content-Type: application/json` + `Sec-Fetch-Site` checks; the SPA sends JSON on every mutating call.
- `preview` enrollment and visibility for the owner/tester cohort (moved here from m1-03).
- `docs/runbooks/identity-admin.md`.

**Out**
- Invites, redeem, `SIGNUP_MODE=invite` behaviour, invite counts in `seats` → [l-03](sprint-l-03.md).
- The acceptance step and `403 acceptance_required` enforcement → [l-05](sprint-l-05.md) (`accepted` is returned here, not enforced).
- `account erase` and web erase (L8) → [l-02](sprint-l-02.md).
- Gateway rate limits L1/L2/L4/L5, typed 413 (L6), P11 suspend → public 404 → [m1-05](sprint-m1-05.md).
- `public-read` mint and `/public/stats` → [m2-03](sprint-m2-03.md); Permissions-Policy (MI-16) → [mi-13](sprint-mi-13.md).
- `judge admin` verbs → M3/M4 sprints.

## Tasks

### 1 · Sessions [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §6 (validate fields), §7 (`status=suspended`), §12 row 4.
- `internal/identity/store/queries/session.sql`: `GetValidSession` joins `identity.account` on `status = 'active'` and returns
  `role`, `status`, `accepted_at`, `session.created_at`; new `RevokeAllSessions(account_id) :execrows`. `sqlc generate`.
- `handleValidateSession` (`internal/identity/handlers.go`) adds `role`, `status`, `accepted` (`accepted_at IS NOT NULL`;
  always false until L-A, not enforced here) and `created_at` (the **session's**, for L8's fresh-session check) — additive.
- Revoke-all on: CLI `suspend` and `revoke-sessions`; `POST /accounts/{id}/password` (`handleSetPassword` in `authemail.go`) —
  every session including the caller's, response `{ok:true, reauth:true}`, the SPA goes to `/auth` with a "password changed —
  sign in again" notice (erase joins in l-02).
- A suspended account cannot start a session: email login returns the uniform 401 (after the dummy compare, task 4); the
  OAuth callback redirects to `/auth?error=account_unavailable`.
- Gateway (`internal/gateway/bff.go`): `identityClient.validateSession` returns `sessionInfo{AccountID, Role, Status, Accepted,
  CreatedAt}`; `authAccount` keeps its signature for existing callers and a new `authSession` exposes the info. The gateway
  **never caches** status or role (m1-05's P11 relies on it).

### 2 · RequireRole + cohort helper [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §7, §12 row 5; [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (T-3).
- `internal/platform/auth/require.go`: `RoleLearner = "learner"`, `RolePublicRead = "public-read"` (used from M2b);
  `RequireRole(v Verifier, role string) func(http.Handler) http.Handler` — bearer → verify → role present in `Claims.Roles`,
  else 401 `unauthenticated` / 403 `forbidden`; claims stored under a shared context key (`auth.ClaimsFrom(ctx)`).
- Every service's `requireJWT` (`internal/{identity,practice,review,assessment,coach}/handlers.go`) becomes
  `auth.RequireRole(s.verifier, auth.RoleLearner)` on every user route.
- **JWTs stay `["learner"]`** for all accounts (`bff.go` `mintFor`/`mintQuiet`); no `owner`/`tester` role is ever minted —
  a unit test asserts the gateway's minted role set is exactly `{learner}` today.
- `internal/gateway/cohort.go`: `inCohort(sessionInfo) = role ∈ {owner, tester}`, the single T-3 read, used by task 7.
- Tests: per service, walk the mux and call every JWT route with a `["public-read"]` token → 403; `["learner"]` → not 401/403.

### 3 · L7 guard [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §3; [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §4 (L7).
- `internal/identity/config.go`: `SignupMode` gains `invite`; `resolveSignupMode(raw string, devAuth bool) (SignupMode, string)`:
  `open` + `DEV_AUTH` → `open`; `open` without `DEV_AUTH` → **`closed`** plus the reason, logged at **ERROR** at startup
  ("SIGNUP_MODE=open ignored without DEV_AUTH; running closed"); `invite` → `closed` until L-A (INFO); anything else → `closed`.
  Logged from `NewService`/`cmd/identity/main.go` (D34: a log line, no alert).
- Table test over `SIGNUP_MODE ∈ {unset, "", open, OPEN, " open ", closed, invite, junk}` × `DEV_AUTH ∈ {unset, true, false, junk}`.
- Compose keeps `SIGNUP_MODE=open` + `DEV_AUTH=true`, so local sign-up is unchanged.

### 4 · L3 bcrypt [X]

Sources: [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §4 (L3); [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §9 (enumeration, bcrypt DoS).
- `internal/identity/password.go`: a process-wide gate of **2** slots around every bcrypt hash and compare; a non-blocking
  acquire that fails → `errBcryptBusy` → handlers return **429** `too_many_requests` with `Retry-After: 1` (protects
  `/sessions/validate`, which every API call needs, on a 250m CPU limit).
- A **dummy hash** generated once at startup at the real cost; `handleLogin` (`authemail.go`) compares against it when the
  identifier is unknown, the account has no password hash, or it is suspended — so the uniform 401 is uniform in time.
- Tests: an injected hasher counts compares (unknown identifier → exactly one compare); a 3rd concurrent compare → 429.

### 5 · `identity admin` CLI [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §7, **§8**, §3 (seats); [t7 §2.3–§2.4](../research/t7-cross-cutting-and-rollout.md).
- `cmd/identity/main.go` dispatches `identity admin …` before `run()` (like `-version`); code in `internal/identity/admin/`,
  queries in `internal/identity/store/queries/admin.sql`. Distroless-safe: static Go, no shell, `flag` parsing. It uses the
  pod's own DB env (`identity.LoadConfig().DB`), a 1-conn pool, and **never runs migrations**.
- `<user>` = email (contains `@`, case-insensitive) | account id (UUID) | username (case-insensitive).

| Verb | Behaviour |
|---|---|
| `account list [--dormant 30d] [--role r] [--status s] [--json]` | id, email, username, role, status, `admitted_via`, created, last session; `--dormant` = no session created in the window |
| `account suspend <user>` | `status='suspended'` + revoke-all, one tx; refuses the **last active owner** |
| `account reactivate <user>` | `status='active'`; for a `learner`, takes `pg_advisory_xact_lock(hashtext('identity.seats'))` and checks the cap |
| `account revoke-sessions <user>` | revoke-all |
| `account set-role <user> learner\|tester\|owner` | refuses to demote the **last active owner** (lock `identity.owners`); becoming `learner` takes the seat lock and checks the cap; promoting is always allowed (the first `owner` is set this way) |
| `account create --role tester --email <e>` | only `tester` in M1b; `admitted_via='cli'`; a random 24-char password (crypto/rand, base64url) printed **once** to stdout; bcrypt via task 4's gate; onboarding row as email signup does; outbox `account_created` (v2) in the same tx |
| `seats` | active `learner` accounts / `SEAT_CAP` (env, default 15); invites "n/a until L-A" |

- **Every verb** (reads included) writes `admin_audit(verb, target = account id or '-', detail jsonb)` in the same
  transaction as its change, plus one log line to stderr. **No secret** (password, invite code) in `admin_audit`, logs or
  `detail`. Note: an exec'd process's output reaches the owner's terminal, not `kubectl logs` — `admin_audit` is the trail.
- Store integration tests (PG 18) per verb: effects, audit row, last-owner and seat guards, password absent from the captured log.

### 6 · CSP + cross-site write checks [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §9 (threat table), §12 row 15; [t7 §2.5](../research/t7-cross-cutting-and-rollout.md).
- `internal/gateway/security.go`, applied in `Handler()` (`internal/gateway/gateway.go`):
  - SPA shell responses: `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com;
    font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self';
    frame-ancestors 'none'; form-action 'self' https://github.com` + `X-Content-Type-Options: nosniff` + `Referrer-Policy: same-origin`.
    Every response gets `nosniff`. No `'unsafe-inline'` in `script-src`. `connect-src 'self'` covers the coach SSE.
  - `form-action` must list `https://github.com`: Chrome applies it to the redirect after the OAuth start form POST.
  - React `style={{…}}` props are set through the CSSOM and are **not** blocked; a `<style>` element or `style=""` attribute in
    `web/index.html` would be — keep it free of both.
  - Mutating `/api/*` (POST/PUT/PATCH/DELETE): `Sec-Fetch-Site`, when present, must be `same-origin` or `none` → else **403
    `cross_site_request`**; `Content-Type` must be `application/json` → else **415 `unsupported_media_type`** — except the form
    POST `/api/auth/{provider}/start` (sign-in and Settings' "Connect GitHub"), which keeps only the `Sec-Fetch-Site` check.
- SPA: `web/src/lib/api.ts` `apiFetch` sets `Content-Type: application/json` on every non-GET (with or without a body). The
  coach chat's raw SSE `fetch` is `streamCoachChat` in `web/src/lib/settings.ts` (not `Coach.tsx`, which has no `fetch`); it
  already sends `Content-Type: application/json` — keep it. A vitest covers both (and fails if either drops the header).
- Tests: CSP present on `/`, `/dsa`, an unknown SPA route; script-src has no `'unsafe-inline'`; a cross-site POST → 403; a
  same-origin form POST to `/api/auth/github/start` → allowed; missing JSON content type → 415.

### 7 · Cohort `preview` [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 8; [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (`preview` hidden outside the cohort).
- identity `handleStartEnrollment`: a `preview` course is enrollable when the caller's **DB** role is `owner` or `tester`
  (identity reads its own row; never a JWT claim); everyone else keeps m1-03's 404.
- Gateway: `courseVisible(m, inCohort(info))` (m1-03's hook) — `preview` courses appear in `/api/paths` and resolve on course
  routes only for the cohort. Tests with m1-03's `preview` fixture manifest: owner/tester see and enroll; learner gets 404.
  (Public profile and `/public/stats` hiding: m1-05 / m2-03.)

### 8 · Runbook [X]

`docs/runbooks/identity-admin.md` (next to the v1 `projection-rebuild.md`):
- The call: `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin <verb> …'` — one example per verb, the
  output shape, exit codes, and that each use is logged in `docs/v2/status.md` (a sanctioned manual path, [rollout §2.2](../rollout-plan.md)).
- **`ev-owner-role`** (after `v1.7.0`, once): `identity admin account set-role <owner-email> owner`, then `account list --role owner`.
- **`ev-first-tester`** — only after **MI-5b is live** ([mi-04](sprint-mi-04.md); ADR-0033 §11): `account create --role tester --email …`,
  hand over the one-time password, the tester changes it in Settings.
- Stranger triage for the GA rule (ADR-0033 §2): `account list --role learner --status active` → `suspend` (erase from L-E).

## Acceptance criteria

- [ ] `suspend` kills every session of the account at once (the next API call 401s); `revoke-sessions` and a password change
      revoke all; the last-owner guard refuses demote/suspend.
- [ ] `SIGNUP_MODE=open` without `DEV_AUTH` runs `closed` and logs ERROR; `invite` runs closed until L-A; compose unchanged.
- [ ] Login timing is uniform for unknown identifiers (one dummy compare); a 3rd concurrent bcrypt → 429 with `Retry-After`.
- [ ] Every CLI verb writes `admin_audit`; no secret in audit or logs; `account create --role tester` prints the password once.
- [ ] A `public-read` token gets 403 on every user route of every service; JWTs are still `["learner"]` only.
- [ ] CSP present on the SPA; cross-site writes 403, non-JSON writes 415; the full web suite and a compose smoke of every
      screen show no CSP violation.
- [ ] `preview` fixture: visible and enrollable for owner/tester, 404 for a learner.
- [ ] `sqlc diff` clean; CI green; merged to `main`.

## Release

**Merge only — ships in `v1.7.0`** ([m1-07](sprint-m1-07.md) tags it; floor after: 1.6.0). No infra change: `SIGNUP_MODE: closed`
stays on the identity HelmRelease, and `SEAT_CAP` uses its code default (15) until L-A sets it explicitly.
**Owner, after `v1.7.0` is live:** `ev-owner-role` — run `set-role <owner> owner` once (runbook); do it before [l-02](sprint-l-02.md)
relies on the role to refuse web erase for the owner.

## Definition of Done

CI green (incl. `sqlc diff`) · merged to `main` · acceptance criteria met · runbook written · statuses updated (this file +
[`../status.md`](../status.md): board row; owner events `ev-owner-role` and `ev-first-tester` marked "prepared (runbook)";
flag inventory lists `SIGNUP_MODE` as an operating mode with the L7 guard) · notable calls in the decisions log.

## Risks / watch-outs

- **CSP breaks an inline style or script** in the SPA or a third-party chunk — run the full web suite plus a manual compose
  smoke of every screen with the console open; never add `'unsafe-inline'` to `script-src` (a `style-src` exception needs a decisions-log entry).
- **Revoke-all on password change logs the owner out** — expected; the SPA explains it.
- **Owner role not yet set:** until `ev-owner-role` runs, there is no owner — the guard allows the first promotion; schedule it
  right after `v1.7.0`.
- **The session join runs on every API call** — it hits the account PK; measure p95 of `/sessions/validate` in compose before/after.
- **Non-browser clients have no `Sec-Fetch-Site`** — allowed by design; the JSON check still applies.
- **Minting a tester before MI-5b** would put a non-owner on the shared origin (ADR-0033 §11) — the CLI can't know; the runbook gates it.
- **`admin_audit` is an honest-operator trail, not tamper-proof** (anyone with exec is root) — accepted in ADR-0033.
