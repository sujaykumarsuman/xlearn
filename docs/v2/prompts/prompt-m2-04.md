# Prompt — Sprint m2-04 · Touch UI (AB04★) + catalog/agenda + Today in minutes (D4) (AB05)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m2-04.md`](../sprints/sprint-m2-04.md)   ·   **Milestone:** M2 (M2a UI + D4)   ·   **Prereqs:** [m2-01](../sprints/sprint-m2-01.md), [m2-02](../sprints/sprint-m2-02.md) (v1.9.0 live), [m2-03](../sprints/sprint-m2-03.md) merged, boards from [ds-m2-01](../sprints/sprint-ds-m2-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and land-and-sync.
- The plan: [`../sprints/sprint-m2-04.md`](../sprints/sprint-m2-04.md). The frame table, route table, planner rules and acceptance are authoritative.
- Touches:
  - [t4 §3.1 and §3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts): the touch lifecycle (conclude, abandoned, voided);
  - [§3.7](../research/t4-judge-contract.md#37-touches-mocks-arena): start, `probes_shown_at`, lock-ins;
  - [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course): DSA criteria `pattern_named_fast` < 120 s, `correct_in_timer`, `complexity_stated`, and L4–5 `mock_mode`;
  - [§8 "Touches"](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed): the copy.
- [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (touches share the hard limit; never-shown or infra-only touches are voided) and [ADR-0026 §4–§5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today) (per-course touch formats; Today and budget).
- D4: [PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses), [t0 §7 "Cross-course Today and budget"](../research/t0-extensibility-frame.md#7-screens--routing), and the [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D4).
- [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): T-1 build-time defaults vs T-2 env flags, which is why the guard is a code default with no env var; every non-kill gate still gets a flag-inventory row with an owning and a removal milestone. [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules): consumers before producers.
- [Rollout §4 M2 and §9](../rollout-plan.md#9-artboards-by-milestone) (AB04★, AB05; the freeze rule).
- Boards: `design-system/screens/v2/AB04-*.html` (Touch★) and `AB05-*.html` (catalog + agenda), via `design-system/screens/v2/index.html`. Also the v1 references `design-system/screens/Revision.dc.html`, `Dashboard.dc.html` and `Catalog.dc.html` (not runnable). [`theme.css`](../../../design-system/theme.css) verbatim.
- Code:
  - M2-01's practice touch endpoints and ticker (`internal/practice/`);
  - `internal/gateway/dashboard.go` (`newWorkLimit`, `buildPlan`), `review.go` (the course-scoped `/api/paths/{slug}/revision/due` handler and its alias), `bff.go` (`apiRoutes`), `withhold.go` and the route-enumeration test (M1-06);
  - `internal/course` (`est_minutes`, `revision.bands`);
  - `web/src/router.tsx`, `screens/Revision.tsx`, `Catalog.tsx`, `Dashboard.tsx`, `lib/revision.ts`, `lib/dashboard.ts`, `lib/budget.ts`, `components/Coach.tsx`;
  - `internal/e2e/coreloop_test.go` (the in-process e2e: practice, review and assessment handlers, real Postgres, embedded JetStream, **no gateway**).
- [m1-03](../sprints/sprint-m1-03.md) task 1: course-scoped aggregates live under `/api/paths/{slug}/…` (no `/api/courses/…` tree), with the DSA aliases calling the same handlers.

## Context

- **M2-01** built practice's touch engine and **no gateway routes**:
  - `purpose=touch` attempts with a server-timed start and probe lock-ins;
  - the touch-deadline ticker (D15: shown then abandoned → fail; never shown or infra-only → void);
  - `problem.spec` probes.
- **v1.9.0** (M2-02) bound review's `touch_concluded` consumer and the projections v2, with producers idle.
- **This sprint** adds:
  - the only BFF touch routes and the **Touch screen (AB04★)**;
  - real format badges on Revision;
  - the deterministic **minutes planner (D4)** for the course dashboard and a new account-wide **agenda**;
  - the AB05 home.
- The producer (`touch_concluded`) lands in **M2-05**, which also tags v1.10.0.
- So the touch start ships **dark behind a build-time guard** (`var touchesEnabled = false`, no env var), and M2-05 deletes it in the producer PR. An interleaved tag can never let touches conclude unheard.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **AB04 and AB05 frozen**: the DS-M2-01 PR is merged by the owner, and the board files exist.
- [ ] **v1.9.0 live**: healthz and `k3s kubectl get deploy -n xlearn` over read-only `ssh vps`.
- [ ] **M2-03 merged** on `main` (the gateway and web serialization).
- [ ] **M2-01's touch endpoints are on `main`** ([M2-01 task 4](../sprints/sprint-m2-01.md#4--touch-endpoints--practice--review-read-x)):
  - `POST /touches/{revision_item_id}/start` (practice checks ownership, pending, due and the lowest level; it resumes an open touch);
  - `GET /touches/{attempt_id}` (the first call stamps `probes_shown_at`; after conclusion it adds the result with `next{level, day}`);
  - `POST /attempts/{id}/probes/{probe_id}`;
  - `POST /attempts/{id}/attest`;
  - `POST /attempts/{id}/end`;
  - the D15 ticker.

  If anything drifted, follow `main` and note it in the plan's task 2. If an endpoint is missing, stop and report; practice rules are M2-01's.
- [ ] **Parallel sessions**: `gh pr list`, `git worktree list` and ListAgents. Check that no peer is editing `internal/gateway/dashboard.go`, `bff.go`, `web/src/router.tsx` or `Catalog.tsx`.

## Do this (in order)

1. **[X] BFF touch routes.** New `internal/gateway/touch.go`. Every route is session-gated, proxied to M2-01's practice endpoint with a practice-audience JWT, and invalidates the account cache on writes. Practice does the ownership, pending, due, lowest-level and one-live-touch checks. Pass its typed errors through unchanged: 404; 409 `touch_not_pending`/`touch_not_due`/`touch_not_lowest`/`item_withdrawn`/`probe_locked`/`touch_expired`; 503 `touches_unavailable`.
   - `POST /api/touches/{itemId}/start` → `POST /touches/{revision_item_id}/start`:
     - returns `{attemptId}`, and resumes an open touch (same attempt);
     - returns 503 **`touches_disabled`** while `touchesEnabled` is false.
   - `GET /api/touches/{attemptId}` → `GET /touches/{attempt_id}`:
     - the first call stamps `probes_shown_at`, so the SPA calls it only when it renders the probes, never to prefetch;
     - after conclusion: `passed`, per-criterion `met` + value, the accepted canonical answer for unmet probes, and `next{level, day}`;
     - runs `withhold()`.
   - `POST /api/touches/{attemptId}/probes/{probeId}` → `POST /attempts/{id}/probes/{probe_id}`: lock state only, never correctness.
   - `POST /api/touches/{attemptId}/attest` → `POST /attempts/{id}/attest` `{criterion: "correct_in_timer", claim}` (self re-solve, honor).
   - `POST /api/touches/{attemptId}/end` → `POST /attempts/{id}/end`.
   - **The guard:** an unexported package-level `var touchesEnabled = false` in `touch.go` (no env var reads it).
     - Tests turn it on with a `_test.go` helper `enableTouches(t)` that restores it via `t.Cleanup`.
     - For local eyeballing only, `internal/gateway/touches_dev.go` (`//go:build touchesdev`, an `init()` setting it true). No committed Dockerfile, compose file or CI job passes the tag; a test asserts the default build has it false.
   - The **shared course-scoped handler** `GET /api/paths/{slug}/revision/due` (and so its DSA alias `/api/revision/due`) exposes `touchesEnabled` and each item's real **format** from `internal/course` bands (`kind`, `label`, `estMin`, `timerSecs`, `mockMode`). Put them on that handler, not only on the alias: the SPA has called the course-scoped route since v1.7.0.
   - Add the routes to `apiRoutes()`, `openapi.yaml`, and the route-enumeration test as `withhold`-applied.

2. **[X] Planner.** New pure package `internal/gateway/plan` (`Build(Input) Plan`), replacing `newWorkLimit`/`buildPlan`:
   - **budget:** weekday `weekday_minutes`; weekend band lower bound (2h → 120, 3–4h → 180, 5h+ → 300); the day in the account tz;
   - **minutes** from manifest `est_minutes`;
   - **due touches from all courses first**, oldest first, never dropped (`overBudgetMin`);
   - **R-SR5 per course:** a touch due before today's start (account tz) blocks that course's new work (`blocked: "reviews_overdue"`);
   - **the remainder** split evenly across active, unblocked enrollments (leftover minute to the earliest enrollment), each filled with frontier items while they fit;
   - deterministic.

3. **[X] Dashboard + agenda.**
   - The course dashboard (`GET /api/paths/{slug}/dashboard` + the DSA alias `/api/dashboard`; M1-03's prefix, no `/api/courses/…` route) uses `plan` for its course, with the header "Today · N min of M".
   - New `GET /api/agenda`, account-wide and cached per account: all courses' due touches, then per-course shares, blocked states, `budgetMin`, `plannedMin`, `overBudgetMin`, and `gradesWaiting: 0`.
   - Read the budget and timezone from identity `GET /accounts/{id}`, in the existing parallel fan-out with per-call timeouts.

4. **[X] Touch screen (AB04★).** `web/src/screens/Touch.tsx` at `/:course/revision/:itemId`, plus `web/src/lib/touch.ts`. Frames, in order (see the plan's table):
   - the cover (format badge; L4–5 "Mock conditions: coach off · statement only · talk aloud, no notes");
   - name the pattern (lock-in, 2:00 clock);
   - re-solve (self report, server-time ring);
   - state the complexity (lock-in);
   - the result (server criteria rows, pass or fail, "advances to Day N" / "resets to Day 1" from `next`, the accepted canonical answer on a probe miss, **no claim button**);
   - the End-review confirm ("Unmet criteria count as a fail");
   - abandoned; inconclusive/voided "Couldn't verify, not counted. [Retry]"; resume.

   Correctness is **never** shown before the result. The coach panel is locked during the touch. Also:
   - focus management, Enter to lock in, `aria-live` at minute boundaries;
   - the < 1024 px stack;
   - `Revision.tsx`: real format badges; "Start" opens Touch when `touchesEnabled`, else the v1 inline form, unchanged.

5. **[X] Catalog + Dashboard (AB05).**
   - `Catalog.tsx`: the agenda on top (with the blocked note and over-budget line), then the multi-course catalog (enrolled, active, `coming_soon`, `preview` for the cohort only), then the "grades waiting" slot, hidden at 0.
   - `Dashboard.tsx`: plan cards in minutes, due first, and the blocked note.
   - Types in `web/src/lib/dashboard.ts` + `useAgenda()`.

6. **[X] Tests.**
   - **Screen tests** for every AB04 frame and for Catalog, Dashboard and Revision (badges, guard fallback).
   - **Gateway `touch_test.go`** (fake practice): practice's typed errors pass through; a resumed start returns the same id; no correctness in lock-in responses; guard off (default) → 503 `touches_disabled` and `touchesEnabled: false` on `/api/paths/{slug}/revision/due` and its alias, guard on (`enableTouches(t)`) → proxied; no pattern pre-conclusion; cache invalidation; the coach relay returns 409 `coach_paused` during a live touch.
   - **`plan_test.go`**, table-driven: weekday/weekend × two timezones across a UTC boundary; cross-course due order; per-course R-SR5; over-budget; split and leftover; nothing fits; determinism.
   - **The in-process e2e** (`internal/e2e`, CI `e2e` job, `go test -tags e2e`). It has no gateway, so drive **practice's touch handlers directly** (the BFF and the guard are covered by `touch_test.go`):
     1. start → view (`probes_shown_at` stamped) → lock-ins → self re-solve → result;
     2. End review → fail;
     3. abandoned: after a shown view, set `timer.deadline_at` in the past through practice's store and trigger **one ticker pass** directly (M2-01's tick function, or an injectable clock/interval), with no 30 s wait.

     Assert the review leg by feeding the concluded attempt's `touch_concluded` fixture to review. The real producer is M2-05.
   - Then `go test ./...`, `go test -tags e2e ./internal/e2e/...` (needs `XLEARN_TEST_DATABASE_URL`), `sqlc diff`, `npm test`, openapi drift and route enumeration.

7. **[X] Docs.** Update [`api.md`](../../architecture/api.md) and `openapi.yaml` (the touch routes, `/api/agenda`, the minute fields). Add Decisions-log entries: the weekend-band mapping, the "overdue" definition, the `touchesEnabled` guard (removed in M2-05) and the route `/:course/revision/:itemId`.

8. **Eyeball** Touch, Revision, Catalog and Dashboard against AB04/AB05 in the browser at 1440 px and 390 px: compose with a local `-tags touchesdev` gateway build (a scratchpad override, never committed), or the temporary vite mock config.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).**
  - practice owns touch attempts and their timers, and review owns the ladder. The gateway only composes, and the planner is a pure function inside it.
  - No cross-schema reads. Manifest data comes from the compiled `internal/course`.
- **Server-authoritative:** timers, criteria, pass or fail, and abandoned or voided outcomes are decided by practice. The client renders, never decides.
- **goose + sqlc:** only if a practice view field is missing. Expand-only, embedded migrations, generated code committed, `sqlc diff` clean.
- **No producer in this sprint.** Do **not** add the `touch_concluded` outbox write, and do **not** change the `touchesEnabled` default. Both are M2-05's, in one PR, so the UI and producer ship in the same tag one tag after the consumer ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
- **Honesty gating:** every touch route goes through `withhold()` and the enumeration test. Lock-ins never echo correctness.
- **Frontend:** `theme.css` verbatim, no Tailwind, the dark theme, a match to the frozen boards. Difficulty tokens are Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`.
- **No new pod, env var, subject or in-cluster caller**, so there's no NetworkPolicy, ACL or infra PR and the memory sum is unchanged.
- **GitOps:** no `kubectl apply`. `ssh vps` is read-only here.
- **No alerting (D34).**
- **Parallel sessions:** check peers' PRs, tags and worktrees before merging, and before any ADR number.
- **No tag in this sprint.**

## Deliverables

- `internal/gateway/touch.go` (five BFF routes + the `touchesEnabled` guard var), `touches_dev.go` (local-only build tag), and the format badges plus the guard flag on the shared `/api/paths/{slug}/revision/due` handler.
- `internal/gateway/plan` (pure planner + table tests), the course dashboard in minutes, and `GET /api/agenda`.
- `web/src/screens/Touch.tsx` (+ `lib/touch.ts`), Revision badges and entry, `Catalog.tsx` agenda + catalog, and `Dashboard.tsx` in minutes.
- Screen, gateway, planner and e2e tests; `api.md` and `openapi.yaml` updates; Decisions-log lines.

## Update status

- In [`../sprints/sprint-m2-04.md`](../sprints/sprint-m2-04.md): set task rows 🔄 → ✅ (⛔ with a reason), set _Overall_, and fill task 2's upstream paths.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for M2-04;
  - the **Artboards** rows AB04 and AB05 → consumed by M2-04 (AB05 full fidelity is at P);
  - **Decisions log** lines for the budget mapping, the overdue definition, the build-time guard (with "removed in M2-05") and the touch route;
  - the **Flag inventory**: `touchesEnabled` as a **T-1 build-time gate** (no env var; owner M2-04; removal M2-05). ADR-0034 §2 gives every non-kill gate an owning and a removal milestone, and this one spans at least one merge-to-tag window.
- Record an ADR only for a departure from ADR-0026 §5 / D4. Check peers first.

## Done when (acceptance)

- [ ] The Touch flow matches the AB04 frames, and correctness is hidden until the server's criteria result.
- [ ] Today shows minutes and due-first ordering; R-SR5 blocks per course; over-budget is reported, never dropped.
- [ ] Catalog and the agenda match AB05 (grades-waiting slot hidden at 0).
- [ ] The touch start is dark (the default build has `touchesEnabled` false → 503, Revision falls back to the v1 form) in any tag built from this merge alone.
- [ ] No touch response leaks the pattern, concepts or correctness before conclusion.
- [ ] CI green (Go, sqlc, web, openapi drift, route enumeration, the in-process e2e).

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: merge only (ships in v1.10.0)**. That means branch `feat/m2-04-touch-ui-today-minutes`, conventional commits with the attribution lines, a PR, CI green, and a squash-merge, then `git checkout main && git pull`. **Do not tag and do not change the `touchesEnabled` default**; [m2-05](../sprints/sprint-m2-05.md) tags and deletes the guard. There's no infra PR.
