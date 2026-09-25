# Sprint m1-05 — Gateway limits + public-dashboard floor (L1, L2, L4–L6, L24; P1, P2, P4, P10, P11)

> **Milestone:** M1 — spine (**M1b** slice, rides `v1.7.0`) · **Track:** product (gateway router serialized [m1-03](sprint-m1-03.md) → [m1-04](sprint-m1-04.md) → **m1-05** → [m1-06](sprint-m1-06.md)) · **Order:** 23
> **Prereqs:** [m1-04](sprint-m1-04.md) (session-validate returns role/status; CSP + cross-site write checks in the chain)
> **Unblocks:** [m1-06](sprint-m1-06.md) · [m1-07](sprint-m1-07.md)
> **Release action:** **merge only** (ships in `v1.7.0`, tagged by [m1-07](sprint-m1-07.md)) · no infra PR · no flag
> **Calendar:** week 3 (2026-10-12 → 10-16; agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m1-05.md`](../prompts/prompt-m1-05.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Limiter package `internal/gateway/limit` + L1, L2, L5 + `X-Real-Ip` trust test | X | ⬜ |
| 2 | L6 typed 413: `httpx.ReadBody` (`MaxBytesReader`) at the 11 request-body sites + the identity `forward` proxy + CI grep gate + SPA 429/413 handling | X | ⬜ |
| 3 | P1 mock count only (D31) | X | ⬜ |
| 4 | P2 visible courses: resolver `visible_courses[]` = enrolled ∩ visible ∩ active | X | ⬜ |
| 5 | P4 + L4 public abuse limits (per-IP bucket, 60 s negative 404 cache, ≤ 8 cold composes) | X | ⬜ |
| 6 | P10 public-shape allowlist test (+ date-only `joinedAt`) | X | ⬜ |
| 7 | P11 suspended → uniform 404 on the next request | X | ⬜ |
| 8 | L24 recorded: single gateway replica is a scale-out blocker | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m1-04](sprint-m1-04.md) merged — `validateSession` returns `role`/`status`/`accepted`/`created_at`; the CSP and
      JSON/`Sec-Fetch-Site` middleware is in the gateway chain; the `identity admin account suspend` verb exists (P11 test)
- [ ] `v1.6.0` live ([m1-02](sprint-m1-02.md)): identity `account.status`, `account.profile_visibility`,
      `path_enrollment.public_visible` exist
- [ ] [m1-03](sprint-m1-03.md) merged: the compiled `internal/course` registry answers "is course X `active`?" in identity
      (enrollment validation) and the gateway (course resolution)
- [ ] Gateway router serialization respected: no open peer PR edits `internal/gateway/{bff,public,gateway,cache}.go`
      except [m1-10](sprint-m1-10.md)'s one additive coach route row (`gh pr list`, `git worktree list`, ListAgents)

## Goal

Make overload **fail loudly** and shrink the only unauthenticated route to its safe v2 shape
([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) §4,
[ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) §13,
[rollout §10](../rollout-plan.md#10-public-dashboard-tasks)): in-process token buckets give typed **429 + `Retry-After`**
for login, signup/GitHub start, the public profile and every account; oversize bodies get a typed **413**; the public
profile shows **mock count only** (D31), only **enrolled ∩ visible ∩ active** courses, is **rate-limited**, passes an
**allowlist test**, and a **suspended** account 404s at once. The single gateway replica is recorded as a scale-out
blocker (L24). Visible changes are D31 and 429s only. The one other payload change is the date-only `joinedAt` on
`/api/u/{username}` (P10), which renders the same. Golden = v1 otherwise.

## Scope

**In**
- `internal/gateway/limit` (token buckets keyed by `X-Real-Ip` or account) and its use for **L1** login, **L2** signup +
  GitHub start, **L4** public profile, **L5** per-account buckets; the `X-Real-Ip` trust test.
- **L6:** `internal/platform/httpx` body helper → typed 413 `body_too_large` (1 MiB default; registry entries for canvas
  640 KiB and interview snapshot 64 KiB). It replaces the **11** gateway request-body `LimitReader` sites and caps the
  streaming identity proxy (`identityClient.forward`: signup, OAuth start/callback, dev login). A CI grep gate keeps it
  that way.
- SPA: `web/src/lib/api.ts` surfaces 429 / 413 as typed errors with retry copy on existing error states.
- Public dashboard **P1** (D31), **P2**, **P4**, **P10**, **P11** (suspend part).
- **L24** written into `docs/architecture/services.md`.

**Out**
- L3 (bcrypt semaphore, dummy hash) and L7 (`DEV_AUTH` guard) → [m1-04](sprint-m1-04.md).
- The `public-read` role and `GET /public/stats` (P3), visibility toggles and "private → 404" (P5), header totals from
  visible courses (P6), touch stats (P9), profile v2 (AB06) → [m2-03](sprint-m2-03.md) (M2b).
- The invite-check limit (L2's third route) → [l-03](sprint-l-03.md) (L-A).
- judge body caps (scene/export) and the 64 KiB code 413 → [m3-09](sprint-m3-09.md) / [m3-14](sprint-m3-14.md) (M3).
- L18 BYO coach caps → [m1-07](sprint-m1-07.md); erased accounts' 404 + 60-day username hold (P11 erase part) →
  [l-02](sprint-l-02.md) (L-E).
- `withhold()` → [m1-06](sprint-m1-06.md). Coach's own internal decoder (`internal/coach/handlers.go:584,590`) sits behind
  the gateway cap and belongs to [m1-10](sprint-m1-10.md)'s files, so it is not touched here. identity's `decodeJSON`
  (`internal/identity/handlers.go:468-471`) likewise sits behind the capped `forward` (task 2) and is not touched.

## Tasks

### 1 · Limiter package + L1, L2, L5 [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L1, L2, L5, L24 rows; "limits fail loudly"),
[t7](../research/t7-cross-cutting-and-rollout.md) (limits table; `X-Real-Ip` rationale).
- **Package** `internal/gateway/limit`: `golang.org/x/time/rate` (already in `go.sum` as indirect → promote to direct);
  `Limiter{rate, burst}` over a `map[key]*bucket` with an injectable clock, idle eviction (sweep every 60 s, evict
  buckets idle ≥ 10 min) and a hard cap of 16 384 keys per limiter (on overflow evict idle first, else reset — a cold
  bucket is always safe). Worst case ≈ a few MiB inside the gateway's 128 Mi limit (memory-sum unchanged).
- **Client IP:** `ClientIP(r)` = `X-Real-Ip` (set by Traefik; `externalTrafficPolicy: Local` in
  `../infra/infrastructure/configs/traefik-config.yaml`, no CDN) else the `RemoteAddr` host. **Never** `X-Forwarded-For`
  or any other client-supplied header. MI-5a ([mi-03](sprint-mi-03.md)) closes in-cluster spoofing.
- **Typed 429:** `writeError(w, 429, "rate_limited", …)` with `Retry-After: <ceil seconds>` and
  `{"error":{"code":"rate_limited","message":…,"retry_after":N}}`; `busy` for the compose semaphore (task 5).
- **L1 login** (`POST /api/auth/login`, `handleAuthLogin`): 10/min per IP (burst 5) **and** 5 failures / 15 min per
  identifier. Buffer the body with task 2's helper (small cap, e.g. 8 KiB), read `email` (email **or** username, F009),
  normalise (trim, lowercase), key the failure window by SHA-256 of it (no raw identifier in memory or logs), re-attach
  the body and forward to identity; count a failure when identity answers 401 (status recorder around `forward`); at 5
  failures in the window, answer 429 **without calling identity** until the oldest ages out; a success clears the
  identifier's window. The uniform 401 is untouched (L3's timing work is m1-04's).
- **L2:** `POST /api/auth/signup` and `POST /api/auth/{provider}/start` — 5/min per IP.
- **L5 per account:** applied at the shared post-validation point of `authAccount` / `authSession` (m1-04 added
  `authSession` beside `authAccount`), right after session validation — every authed route covered, no per-route
  wiring: all `/api` 20 req/s (burst 40); mutating methods (POST/PUT/PATCH/DELETE) additionally 5 req/s (burst 10).
  The coach SSE chat counts once at request start (L18's message caps are m1-07's).
- **Exempt:** `/api/healthz`, `/.well-known/jwks.json`, `/healthz`, `/readyz`, SPA static assets.
- **Measure first:** count the SPA's requests on cold Dashboard, Week, Problem, Progress and public-profile loads.
  Measure through the **browser network log** on `docker compose up` with dev login (preferred), or on vite dev with the
  mock config. Don't use the e2e harness: `internal/e2e/coreloop_test.go` drives the practice, review and assessment
  HTTP surfaces directly and runs neither the gateway nor the SPA. Put the table in the PR; every burst must be ≥ 2×
  the worst cold load.
- **`X-Real-Ip` trust test** (`internal/gateway/limit/clientip_test.go`): the key is `X-Real-Ip`; adding
  `X-Forwarded-For`, `Forwarded` or `True-Client-Ip` never changes it; missing header → `RemoteAddr` host (IPv4 and IPv6).
  Plus a read-only live check recorded in the PR: `ssh sujaykumar-vps 'k3s kubectl -n kube-system get svc traefik -o jsonpath={.spec.externalTrafficPolicy}'` = `Local`.

### 2 · L6 typed 413 [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6).
- `internal/platform/httpx/body.go`: `ReadBody(w, r, limit int64) ([]byte, bool)` — wraps `http.MaxBytesReader`,
  detects `*http.MaxBytesError` and writes **413** `{"error":{"code":"body_too_large","limit":N}}`; other read errors →
  400 `bad_request` as today. Registry: `BodyLimitDefault = 1 << 20`, `BodyLimitCanvas = 640 << 10`,
  `BodyLimitInterviewSnapshot = 64 << 10` (M3 adds judge scene 512 KiB / export 32 KiB; m3-09 adds code 64 KiB).
- Replace the **11** silent request-body sites: `internal/gateway/assessment.go:82,143`,
  `internal/gateway/bff.go:164,189,261,310,664,763`, `internal/gateway/coach.go:153,238`, `internal/gateway/mistakes.go:102`
  (line numbers as of `f5694ab`; re-grep after rebase). Upstream **response** reads (`resp.Body`) keep `LimitReader`:
  they bound replies from our own services, not client input.
- **The streaming proxy.** `identityClient.forward` (`bff.go:1087-1092`, called at `bff.go:228,239,247,353`) passes the
  raw `r.Body` to `http.NewRequestWithContext` with no cap, which leaves signup, OAuth start/callback and dev-login
  bodies unbounded.
  - Wrap it as `body := http.MaxBytesReader(w, r.Body, httpx.BodyLimitDefault)` and pass `body`, never `r.Body`.
  - A declared `Content-Length` over the cap → 413 `body_too_large` before dialing identity.
  - A chunked body that overruns → the client error unwraps (`errors.As`) to `*http.MaxBytesError` → the same typed
    413, not a 502.
  - L1's login path keeps its 8 KiB buffer and re-attaches the buffered body.
- **CI grep gate:** `hack/lint-bodies.sh`, run in `ci.yml`'s `go` job and in `make lint`. It fails on these patterns:
  - `LimitReader(r.Body` or `ReadAll(r.Body` under `internal/gateway/` and `internal/platform/`;
  - under `internal/gateway/`, any raw `r.Body` passed to `http.NewRequest` / `http.NewRequestWithContext`;
  - under `internal/gateway/`, `json.NewDecoder(r.Body)`.
- **SPA:** `web/src/lib/api.ts` **keeps the server envelope's `code` (and `reason`) as is** and only adds `retryAfter`,
  parsed from the `Retry-After` header or `error.retry_after`. Every 429 carries its own code:
  - `rate_limited`, the gateway limits;
  - `busy`, the compose semaphore;
  - `provider_limited` plus m1-10's `reason`, the coach;
  - [m1-07](sprint-m1-07.md)'s L18 codes.

  None of them may collapse into `rate_limited`, because `settings.ts` routes on `code === "provider_limited"`. A 413
  keeps `body_too_large`. Existing error states render "Too many requests — try again in N s" or "That's too large to
  send"; there is no new screen. react-query must not auto-retry a 429 faster than `Retry-After`.
- **Tests:** in `web/src/lib/`, a 429 `provider_limited` and a 429 `busy` keep their codes and gain `retryAfter`. In
  `internal/gateway/`, an oversize `POST /api/auth/signup` → 413 `body_too_large`, and the fake identity never sees
  the overrun.

### 3 · P1 mock count only (D31) [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P1, [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) (D31).
- `internal/gateway/public.go`: `publicMock` becomes `{count}`; `best`/`average` never leave the gateway (the summary
  projection still returns them; the authed Progress route keeps them).
- `web/src/lib/profile.ts` (`PublicProfileMock = {count}`), `web/src/screens/UserDashboard.tsx` tile: "Mocks · N"
  (and "No mocks yet" at 0) — the "Mock best / Avg" copy goes; `UserDashboard.test.tsx` updated.
- `internal/gateway/public_test.go`: replace the `mock.best == 24` assertion with "no `best`/`average` anywhere".

### 4 · P2 visible courses [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P2, [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas), D7.
- identity `handleInternalGetAccountByUsername` (`internal/identity/usernames.go`) returns
  `visible_courses: ["dsa", …]` = enrolled (`path_enrollment`) ∩ `public_visible = true` ∩ manifest `status = active`
  (the embedded `internal/course` registry m1-03's enrollment validation uses). `preview`, `coming_soon` and `retired`
  never appear — including the owner's cohort `preview` enrollments. New sqlc query; `sqlc diff` clean.
- `internal/gateway/public.go`: `composePublicProfile` iterates `visible_courses` ∩ the gateway's own active list
  (defence in depth) instead of walking every active path (`activePaths`, `public.go:144`). The account-wide header
  totals are unchanged here (P6 is M2b).
- Tests (identity + gateway): enrolled-and-visible active course shown; enrolled-but-hidden excluded; active-but-not-
  enrolled excluded; `preview` excluded even when enrolled.

### 5 · P4 + L4 public abuse limits [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L4, [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P4.
- `GET /api/u/{username}`: 60/min per IP (burst 20) from task 1.
- **Negative cache:** a 404 from the resolver is cached for 60 s per normalised username (bounded map, drop-all on
  overflow like `aggCache`); it is the **only** resolver-side cache (task 7).
- **≤ 8 concurrent cold composes:** a buffered-channel semaphore around `composePublicProfile` on a composed-cache miss;
  full → 429 `busy` with `Retry-After: 1`. Warm hits skip the semaphore.
- Tests with an injected clock: the 21st burst request 429s; a 404 is served from the negative cache without calling
  identity; the 9th concurrent cold compose 429s.

### 6 · P10 public-shape allowlist test [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P10, [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) ("never public").
- `internal/gateway/public_test.go`: walk the serialized payload recursively and compare its **key paths** against an
  explicit allowlist (the current shape minus `mock.best/average`: `user.{username,displayName,joinedAt,region}`,
  `totals.{solved,streak.{current,longest}}`, `mock.count`, `heatmap.days[].*`, `courses[].{slug,title,solved,total,pct,
  phases[].*,patterns[].{name,solved,total,pct}}` — enumerate the `*` members exactly); any other key fails, so a new
  field needs a reviewed allowlist edit.
- Value checks: no item ids (no `problemId`/`id`/`items` key, no fixture id value), no per-item `pattern`, **no sub-day
  timestamps** (no string matching `T\d{2}:\d{2}`), no `scored_by`, `transcript`, `notes`, `feedback`, `prose`, arena or
  Run counts. Upstream fakes plant a sentinel in every field that must not pass.
- **`joinedAt` becomes a date** (`YYYY-MM-DD`, from identity's `created_at`) to satisfy "no sub-day timestamps";
  `formatJoined` in `UserDashboard.tsx` keeps rendering "Joined <Mon YYYY>".

### 7 · P11 suspended → 404 at once [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P11, [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) (Suspend / erase).
- Mechanism: suspend runs in the identity pod (`identity admin` via `kubectl exec`) and the gateway is not a NATS client,
  so the gateway **never positively caches account resolution**: every public request re-resolves username →
  `{id, status, visible_courses}` from identity (it already does — keep it, with a comment naming P11). identity's
  resolver answers only `status = 'active'`; a suspended account gets the same uniform 404 as an unknown one. The
  composed-payload cache (keyed by account, reached only **after** a successful resolve) and the 60 s negative 404 cache
  stay; the in-process epoch bump stays for gateway-routed writes.
- Test: warm the composed cache → suspend the account (identity store/CLI in the harness) → the **very next** request is
  404; reactivate → visible again within ≤ 60 s (negative-cache TTL).

### 8 · L24 record [X]

- `docs/architecture/services.md` (gateway section): the **single gateway replica** holds the aggregation cache epoch
  **and** all limiter state (L1/L2/L4/L5 buckets, the login failure windows, the negative 404 cache, the compose
  semaphore) — a **scale-out blocker**; scaling out needs shared state (or sticky routing) first. Link
  [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L24 and
  [rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session).
- `docs/architecture/api.md` / `openapi.yaml`: document the 429 (`rate_limited`, `busy`) and 413 (`body_too_large`)
  envelopes and the public profile's `mock.count` + date-only `joinedAt`.

## Acceptance criteria

- [ ] **Typed 429 + `Retry-After`** under the load tests at exactly the L1/L2/L4/L5 thresholds (injected clock); below
      them, the measured cold-load fan-out never 429s (table in the PR).
- [ ] **Typed 413** `body_too_large` on oversize bodies at all 11 sites **and** through the identity `forward` proxy
      (oversize signup test). The CI grep gate fails on a new raw `r.Body` read, proxy or decoder. SPA 429s keep the
      server's `code`.
- [ ] `X-Real-Ip` trust test green; `externalTrafficPolicy: Local` confirmed read-only.
- [ ] **Public payload passes the allowlist test; no best/avg**; `joinedAt` date-only; the SPA tile shows the count only.
- [ ] Public profile lists only enrolled ∩ visible ∩ active courses; `preview` never shown.
- [ ] **Suspended profile 404s at once** (next request, warm cache).
- [ ] L24 recorded in `services.md`; 429/413 documented in the API docs.
- [ ] Every v1 e2e green. Golden = v1 except D31, 429s and the **date-only `joinedAt` on `/api/u/{username}`**
      (m1-07's tag checklist uses this exception list). CI green (incl. `sqlc diff`).

## Release

**Merge only — ships in `v1.7.0`**, cut by [m1-07](sprint-m1-07.md) (M1b). Do **not** tag. No infra PR, no flag, no new
pod (limiter state lives in the existing gateway process). Rollback floor unaffected (no schema change beyond an
identity query). The limit values are code constants; changing one is a normal PR in a later tag.

## Definition of Done

CI green (incl. `sqlc diff` and the new grep gate) · PR squash-merged to `main` (no tag) · acceptance criteria met ·
statuses updated (this file + [`../status.md`](../status.md): board row; M1 stays 🔄) · measured fan-out and any value
change recorded in the decisions log · L24 in `services.md`.

## Risks / watch-outs

- **Limits too tight for the SPA's burst on dashboard load** — measure the request fan-out first; bursts ≥ 2× the worst
  cold load; react-query retries must honour `Retry-After`.
- **A `LimitReader` site missed** — the grep gate in CI, plus re-grepping after rebasing on m1-04 (which may add
  handlers).
- **L1 per-identifier lockout** can be triggered by anyone who knows the owner's email or username (5 bad passwords →
  15 min of 429 on password login). Accepted by ADR-0035 L1; GitHub sign-in is unaffected. Note it in the decisions log.
- **Memory** — every limiter map is bounded and swept; a flood of distinct IPs resets buckets rather than growing.
- **Negative cache vs a new username** — a just-claimed username can 404 for ≤ 60 s; acceptable.
- **m1-06 rebases on this** (same files: `bff.go`, `public.go`, `coach.go`); merge promptly and keep the diff focused.
- **Time-dependent tests** — inject the clock; never sleep in tests.
