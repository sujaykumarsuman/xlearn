# Prompt — Sprint m1-05 · Gateway limits + public-dashboard floor

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-05.md`](../sprints/sprint-m1-05.md)   ·   **Milestone:** M1 (M1b slice, ships in `v1.7.0`)   ·   **Prereqs:** [m1-04](../sprints/sprint-m1-04.md) (and through it [m1-03](../sprints/sprint-m1-03.md), `v1.6.0`)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m1-05.md`](../sprints/sprint-m1-05.md).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) — the limits inventory (L1, L2, L4, L5, L6, L24 rows; "limits fail loudly"; the `X-Real-Ip` note under the table).
- [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) — public-dashboard authz deltas (D31, scope, abuse, shape, suspend).
- [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) — P1–P12 (this sprint: P1, P2, P4, P10, P11 suspend part) and [§4 M1](../rollout-plan.md#4-per-milestone-detail) (scope T7).
- [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) — what is public and what is never public.
- [ADR-0024](../../adr/0024-public-user-dashboards-and-usernames.md) / [ADR-0025](../../adr/0025-public-profiles-under-u-prefix.md) — the v1 public profile you are narrowing.
- Code: `internal/gateway/{gateway,bff,public,cache,coach,assessment,mistakes}.go`, `internal/gateway/public_test.go`,
  `internal/platform/httpx/httpx.go`, `internal/identity/{usernames,service}.go` + `store/queries/`,
  `web/src/lib/{api,profile}.ts`, `web/src/screens/UserDashboard.tsx`, `.github/workflows/ci.yml`, `Makefile`,
  `../infra/infrastructure/configs/traefik-config.yaml` (read only), `../infra/apps/xlearn-gateway.yaml` (read only: 1 replica, 128 Mi).

## Context

The gateway has no request limits today: overload is silent, bodies are truncated at 1 MiB by `io.LimitReader` without
an error, and `GET /api/u/{username}` — the only unauthenticated route — shows `best`/`average` mock scores and every
active path, with no rate limit. v2 decides (ADR-0035 §4) that per-IP and per-account limits live **in the gateway
process** (Traefik can't key by account), fail with typed 429/413, and that the public route shrinks (ADR-0033 §13).
m1-04 just gave session-validate `role`/`status` and added CSP and cross-site write checks; this sprint is next on the
serialized gateway router (m1-03 → m1-04 → **m1-05** → m1-06). [m1-10](../sprints/sprint-m1-10.md) may merge in parallel with
one additive coach route row. Everything ships in `v1.7.0` (m1-07's tag).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m1-04](../sprints/sprint-m1-04.md) merged (`validateSession` returns role/status; CSP middleware present; `identity admin account suspend` exists)
- [ ] `v1.6.0` live (`curl -s https://projects.sujaykumar.dev/xlearn/api/healthz` ≥ 1.6.0) — identity `status`,
      `profile_visibility`, `path_enrollment.public_visible` columns on `main`
- [ ] [m1-03](../sprints/sprint-m1-03.md) merged (`internal/course` registry used for course status in identity and the gateway)
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no peer PR edits
      `internal/gateway/{bff,public,gateway,cache}.go` other than m1-10's single route row; m1-06 has not started

## Do this (in order)

1. **[X] Branch** `feat/m1-05-gateway-limits` off an up-to-date `main`.
2. **[X] Measure the SPA fan-out**: count requests on cold Dashboard, Week, Problem, Progress and `/u/<name>` loads
   in the **browser network log**, on `docker compose up` with dev login (or vite dev with the mock config). The e2e
   harness can't do this: it drives the service HTTP surfaces and runs neither the gateway nor the SPA. Keep the table
   for the PR; bursts must be ≥ 2× the worst.
3. **[X] Limiter package (task 1)** — `internal/gateway/limit`: `golang.org/x/time/rate` promoted to a direct
   dependency; bounded per-key maps (sweep 60 s, idle ≥ 10 min evicted, cap 16 384); injectable clock; `ClientIP(r)`
   (`X-Real-Ip` else `RemoteAddr`, never `X-Forwarded-For`); typed 429 helper (`rate_limited` / `busy`, `Retry-After`,
   `retry_after`). Unit + `clientip_test.go` trust tests.
4. **[X] Wire L1, L2, L5 (task 1)** — L1 on `POST /api/auth/login`: 10/min/IP burst 5 + 5 failures / 15 min per
   identifier (buffer the body, SHA-256 of the normalised `email` field, count identity 401s via a status recorder,
   short-circuit 429 at 5, clear on success). L2: 5/min/IP on `POST /api/auth/signup` and `POST /api/auth/{provider}/start`.
   L5 at the shared post-validation point of `authAccount` / `authSession` (m1-04): 20 rps burst 40 for all `/api`, plus 5 rps burst 10 for mutating methods.
   Exempt health/JWKS/probes/static.
5. **[X] L6 (task 2)**
   - `internal/platform/httpx/body.go`: `ReadBody` + the limit registry.
   - Replace the 11 request-body sites: `assessment.go:82,143`, `bff.go:164,189,261,310,664,763`, `coach.go:153,238`,
     `mistakes.go:102`. Re-grep `LimitReader(r.Body` after rebasing.
   - **Cap `identityClient.forward`** (`bff.go:1087-1092`): pass `http.MaxBytesReader(w, r.Body, BodyLimitDefault)`,
     never raw `r.Body`. An over-cap `Content-Length` → 413 before dialing; an overrun whose client error unwraps to
     `*http.MaxBytesError` → typed 413, not 502. Add a test that an oversize signup → 413 `body_too_large`.
   - `hack/lint-bodies.sh` in `ci.yml` + `make lint`. It flags `LimitReader(r.Body` / `ReadAll(r.Body`, and under
     `internal/gateway/` also raw `r.Body` passed to `http.NewRequest*` and `json.NewDecoder(r.Body)`.
   - `web/src/lib/api.ts`: **keep the envelope's `code` and `reason`**, and only add `retryAfter` (from `Retry-After`
     or `error.retry_after`). Test that a 429 `provider_limited` and a 429 `busy` keep their codes. Retry copy goes
     on existing error states, with no react-query retry sooner than `Retry-After`.
6. **[X] P1 (task 3)** — `publicMock{count}`; `lib/profile.ts`, `UserDashboard.tsx` tile "Mocks · N"; tests.
7. **[X] P2 (task 4)** — identity resolver returns `visible_courses` (enrolled ∩ `public_visible` ∩ manifest `active`;
   new sqlc query, `sqlc generate`); `composePublicProfile` iterates `visible_courses` ∩ the gateway's active list;
   tests incl. `preview` hidden.
8. **[X] P4/L4 (task 5)** — per-IP 60/min burst 20 on `/api/u/{username}`; 60 s negative 404 cache (bounded); ≤ 8
   concurrent cold composes (semaphore → 429 `busy`, `Retry-After: 1`); tests.
9. **[X] P11 (task 7)** — identity resolver answers only `status='active'`; keep per-request resolution in the gateway
   (comment "P11: never cache a positive resolve"); test: warm cache → suspend → next request 404; reactivate → ≤ 60 s.
10. **[X] P10 (task 6)** — recursive key-path allowlist + value checks in `public_test.go` with sentinel-planting fakes;
    `joinedAt` → `YYYY-MM-DD` (identity or gateway), `formatJoined` still renders "Joined <Mon YYYY>".
11. **[X] Docs (task 8)** — L24 in `docs/architecture/services.md`; 429/413 envelopes and the public shape in
    `docs/architecture/api.md` + `openapi.yaml` (the drift test must stay green).
12. **[X] Verify** — `gofmt`, `go vet`, `go test -race ./...`, `sqlc diff`, e2e (`go test -tags e2e -race ./internal/e2e/...`;
    golden = v1 except D31, 429s and the date-only `joinedAt` on `/api/u/{username}`), `npm --prefix web run` `typecheck`, `lint`, `test`, `build`; the read-only check
    `ssh vps 'k3s kubectl -n kube-system get svc traefik -o jsonpath={.spec.externalTrafficPolicy}'` → `Local` (record in the PR).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the gateway owns no schema; the
  resolver change is an identity query in schema `identity` via sqlc (`sqlc diff` clean). No migration is expected; if
  one is needed it is additive only (ADR-0034 §3).
- **Limits live in the gateway process** (ADR-0035 §4 placement); no Traefik `RateLimit`, no new component, no Redis.
- **No new pod; memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** every map bounded; stay well inside the gateway's 128 Mi limit.
- **No events, no NATS:** nothing here publishes or consumes (the gateway is not a NATS client), so no ACL PR and no
  consumers-before-producers step. Suspend visibility relies on per-request resolution, not on an event.
- **Serialized router:** touch only what the plan lists; m1-06 rebases on you. Don't touch `internal/coach/**` (m1-10).
- **Frontend:** `theme.css` verbatim, dark theme; only the tile copy and error copy change (no new screen).
- **GitOps:** no `kubectl apply`; the only cluster access is the read-only `ssh vps` check. **D34:** no alerting; a 429 is
  a typed response, not a signal anyone is paged on.
- **Parallel sessions:** if you record an ADR (e.g. a limit value departing from ADR-0035), re-check peers' PRs and
  ADR numbers first.

## Deliverables

- `internal/gateway/limit` (+ tests); L1/L2/L5 wiring; `X-Real-Ip` trust test.
- `internal/platform/httpx/body.go` + the 11 replaced sites + the capped identity `forward` + `hack/lint-bodies.sh` in CI; SPA 429/413 with the server code kept and `retryAfter` added.
- Public profile: `mock.count` only, `visible_courses` filtering, per-IP limit, negative cache, compose semaphore,
  suspend → 404, date-only `joinedAt`; identity resolver query; allowlist test.
- `services.md` L24; API docs for 429/413 and the public shape; fan-out table in the PR.

## Update status

- Set each task in [`../sprints/sprint-m1-05.md`](../sprints/sprint-m1-05.md) 🔄 → ✅; _Overall_ ✅ when all eight are.
- [`../status.md`](../status.md): **Sprint board** row for m1-05; **Milestones** M1 stays 🔄.
- **Decisions log:** the measured fan-out and final limit values; `joinedAt` is date-only on the public route; the
  accepted L1 per-identifier lockout; L24 recorded. No flag change.
- ADR only for a call beyond ADR-0035/0033 (numbered after the parallel-sessions check).

## Done when (acceptance)

- [ ] Typed 429 + `Retry-After` under load tests at the L1/L2/L4/L5 thresholds; no 429 on measured cold loads.
- [ ] Typed 413 `body_too_large` at all 11 sites and through the identity `forward` proxy. The grep gate, including
      the proxy and decoder patterns, is in CI. SPA 429s keep the server's `code`.
- [ ] `X-Real-Ip` trust test green.
- [ ] Public payload passes the allowlist test; no best/avg; date-only `joinedAt`.
- [ ] Only enrolled ∩ visible ∩ active courses on the public profile; `preview` hidden.
- [ ] Suspended profile 404s at once.
- [ ] L24 recorded; every v1 e2e green (golden = v1 except D31, 429s, date-only `joinedAt`); CI green.

Ship at session end per AGENT.md land-and-sync with **this sprint's release action — merge only**: conventional
commits (`feat(gateway): …`) with the attribution lines, push, open the PR, wait for CI green (fix-then-merge on failure),
squash-merge, then `git checkout main && git pull`. **Do not tag** — this work ships in `v1.7.0`, which
[m1-07](../sprints/sprint-m1-07.md) cuts; Flux deploys nothing until then, so there is no live verification in this session.
