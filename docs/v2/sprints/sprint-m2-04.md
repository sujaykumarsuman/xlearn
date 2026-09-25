# Sprint m2-04 — Touch UI (AB04★) + catalog/agenda + Today in minutes (D4) (AB05)

> **Milestone:** M2 — attempt engine and projections (M2a UI + D4)   ·   **Track:** product
> **Prereqs:** [m2-01](sprint-m2-01.md) (practice touch engine), [m2-02](sprint-m2-02.md) (v1.9.0), [m2-03](sprint-m2-03.md) (gateway/web serialized) · boards from [ds-m2-01](sprint-ds-m2-01.md)   ·   **Unblocks:** [m2-05](sprint-m2-05.md)
> **Release action:** merge only (ships in **v1.10.0**, tagged by [m2-05](sprint-m2-05.md)). The touch start stays behind a build-time guard that M2-05 removes with the producer.   ·   **Calendar:** late October – early November (no owner involvement)
> **Execute with:** [`../prompts/prompt-m2-04.md`](../prompts/prompt-m2-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Touch screen (AB04★): `web/src/screens/Touch.tsx` + Revision entry + real format badges | X | ⬜ |
| 2 | BFF touch routes (the only gateway touch routes) + the `touchesEnabled` build-time guard | X | ⬜ |
| 3 | Today in minutes (D4): the pure `plan` package, course dashboard + `GET /api/agenda` | X | ⬜ |
| 4 | Catalog + cross-course agenda + course Dashboard in minutes + "grades waiting" slot (AB05) | X | ⬜ |
| 5 | Tests: screen frames, BFF, planner table, in-process e2e touch loop | X | ⬜ |
| 6 | Docs: `api.md`, `openapi.yaml`, route-enumeration list | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB04 and AB05 frozen**: [ds-m2-01](sprint-ds-m2-01.md) merged (the merge is the freeze, D40; boards under `design-system/screens/v2/`, indexed by `design-system/screens/v2/index.html` from DS-M1-01).
- [ ] **v1.9.0 live** ([m2-02](sprint-m2-02.md)), so this merges into v1.10.0 together with the producers.
- [ ] **M2-03 merged**: the gateway BFF and web edits are serialized M2-03 → M2-04.
- [ ] **M2-01's practice touch endpoints are on `main`** as [M2-01 task 4](sprint-m2-01.md#4--touch-endpoints--practice--review-read-x) specifies. If `main` drifted, follow `main` and note it in task 2:
  - `POST /touches/{revision_item_id}/start`: server-timed. Practice checks ownership, pending, due and the lowest level through review's internal read, and resumes an open touch.
  - `GET /touches/{attempt_id}`: the first call stamps `probes_shown_at`; after conclusion it adds the result with `next{level, day}`.
  - `POST /attempts/{id}/probes/{probe_id}`: lock-in.
  - `POST /attempts/{id}/attest`: `{criterion, claim}`, the M2 self re-solve.
  - `POST /attempts/{id}/end`.
  - The touch-deadline ticker (D15).

## Goal

Make revision a **timed, server-scored experience** and budget the day **in minutes**.
- **The Touch screen (A8 / AB04★)** runs a DSA touch end to end on M2-01's practice engine:
  1. name the pattern (lock-in, under a 2:00 clock);
  2. re-solve (self path until M3);
  3. state the complexity (lock-in).

  Correctness stays hidden until the server's criteria result, which reads "advances to Day 21" or "resets to Day 1". Levels 4–5 run under **mock conditions**.
- **Today** stops planning "3 new problems". It becomes a deterministic **minutes budget** (D4):
  - due touches from **every** course come first;
  - R-SR5 blocks new work **per course**;
  - the remainder splits across active enrollments.
- **AB05** gives the home a **cross-course agenda** and a "grades waiting" slot that M4 fills.

This follows [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today), [PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses), [t4 §3.6–§3.7, §6.6 and §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed), and [t0 §7](../research/t0-extensibility-frame.md#7-screens--routing).

## Scope

**In**
- web: the `Touch.tsx` screen (every AB04 DSA frame), the Revision queue's entry into it with real format badges from the manifest bands, `Catalog.tsx` (agenda + catalog, AB05), `Dashboard.tsx` in minutes, and `web/src/lib/touch.ts` plus the `dashboard.ts` types.
- gateway:
  - the BFF touch routes. They exist **only here**: M2-01 added none, so nothing was reachable before the v1.10.0 producers;
  - a `touchesEnabled` build-time guard (a T-1 code default, no env), default **off**;
  - the pure `internal/gateway/plan` package;
  - the course dashboard in minutes and `GET /api/agenda`.
- practice: only what the BFF strictly needs from M2-01's engine (for example a missing field on the view). No new rules.

**Out (later sprints)**
- **Emitting `touch_concluded`** and **removing the guard**: [m2-05](sprint-m2-05.md). This keeps the UI and the producer in the same tag.
- The honor-probe **"I meant this" claim** and AI-provisional touches: M4 ([m4-04](sprint-m4-04.md), [m4-06](sprint-m4-06.md)). In M2 an honor-probe miss shows the accepted answers and fails the criterion.
- **Judge-graded re-solve** (`correct_in_timer` checked by a passing touch submit ≤ 20:00): M3 ([m3-08](sprint-m3-08.md), [m3-11](sprint-m3-11.md)). In M2 it's self-reported (`graded_by=self`, `trust=honor`).
- Non-DSA touch formats (go/sql/lld recall quizzes) and full-fidelity AB05 for a second course: P ([p-02](sprint-p-02.md), [p-03](sprint-p-03.md)).
- "Unfinished: resume or give up" on Today, for course attempts under the hard limit: M3.

## Tasks

### 1 · Touch screen (AB04★) [X]

Build to the frozen `design-system/screens/v2/AB04-*.html`, with `theme.css` verbatim.
- **Route.** `/:course/revision/:itemId` inside `CurriculumShell` (`web/src/router.tsx`, course from `useCourse()`). The Revision queue's "Start" opens it. `web/src/screens/Revision.tsx` shows the **real format badges** from the manifest band of each level (AB03's placeholders from M1-06 become data), served per item by the shared `GET /api/paths/{slug}/revision/due` handler (task 2):
  - "Recall · ~5 min" or "Re-solve · 20:00";
  - "Mock conditions" at L4–5.
- **Frames (DSA):**

  | Frame | Behaviour |
  |---|---|
  | Cover | format badge, band, est minutes, the L4–5 "Mock conditions" note ("coach off · statement only · talk aloud, no notes"); [Start] |
  | Name the pattern | free text + [Lock in] (Enter). A visible **2:00 recall clock** from the server start. The input freezes on lock-in. **No correctness shown** |
  | Re-solve (self path) | statement only, the band's re-solve timer (20:00) as a server-time ring with no pause; the learner attests "Solved within the timer" / "Not solved" (`POST …/attest`, honor) |
  | State the complexity | time / space Big-O inputs + [Lock in] (`p-complexity`, `blank.big_o`); no correctness shown |
  | Result | server criteria rows (e.g. "Pattern named 0:48 · solved in timer (self) · O(n²) / O(1)"), pass or fail, and **"advances to Day 21"** or **"resets to Day 1"** (from `next{level, day}`). On a probe miss, the accepted canonical answer is shown (no claim button until M4) |
  | End review | a confirm dialog: "Unmet criteria count as a fail" → [End review] / [Keep going] |
  | Abandoned | shown then left past the deadline → the result frame as a fail (`resolution=abandoned`), computed by M2-01's ticker, never the client |
  | Inconclusive / voided | "Couldn't verify, not counted. [Retry]"; Retry starts a fresh touch attempt |
  | Resume | reloading mid-touch returns to the same step with server time; there's no client timer state |

- **Timers are server-authoritative.** Render from `startedAt`/`deadline` with drift correction. `aria-live` announces at minute boundaries, not every second.
- **Coach.** The panel shows the locked state (M1-07's `coach_paused`) during a live touch. M2-01 reports open touches as `purpose: "touch"` on `GET /attempts/open`, so add a gateway test that the coach relay returns 409 `coach_paused` during one.
- **Cache.** Every touch write invalidates the account's cached views. On return to Revision or Dashboard, refetch. The ladder change appears once review consumes `touch_concluded`: the producer is M2-05, and it takes relay tick plus consumer lag.
- **Fallback.** While `touchesEnabled` is false (task 2), Revision keeps the v1 inline self-score form (unchanged `POST /api/revision/{itemId}/score`) and hides the Touch entry. M2-05 deletes the guard and the fallback branch.
- **A11y and layout:**
  - focus moves to each step's primary input;
  - Enter locks in, Esc closes the End-review dialog;
  - contrast follows the tokens;
  - below 1024 px, the steps stack and the timer ring stays pinned.

### 2 · BFF touch routes + the build-time guard [X]

New `internal/gateway/touch.go`. Every route is session-gated (`RequireRole("learner")`) and proxies to M2-01's practice endpoints with a practice-audience JWT. Practice does the ownership, pending, due, lowest-level and one-live-touch checks ([t4 §3.3](../research/t4-judge-contract.md#33-transitions)). The BFF passes its typed errors through unchanged: 404, 409 `touch_not_pending` / `touch_not_due` / `touch_not_lowest` / `item_withdrawn` / `probe_locked` / `touch_expired`, and 503 `touches_unavailable`.

| BFF route | Upstream (M2-01) | Rules |
|---|---|---|
| `POST /api/touches/{itemId}/start` | `POST /touches/{revision_item_id}/start` | 503 **`touches_disabled`** while `touchesEnabled` is false. Otherwise it returns `{attemptId}`. An open touch for the same problem is **resumed**: practice returns the existing attempt |
| `GET /api/touches/{attemptId}` | `GET /touches/{attempt_id}` | Format, band, `mockMode`, server timer, answer-free probes with `locked`/`lockedAtSecs`. **The first call stamps `probes_shown_at`** ([t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)), so the SPA calls it only when it renders the probes, never to prefetch. After conclusion: `passed`, per-criterion `met` + value, the accepted canonical answer for unmet probes (never the alias table), and `next{level, day}`. Runs `withhold()` (M1-06): no pattern, concepts or solution facts before conclusion |
| `POST /api/touches/{attemptId}/probes/{probeId}` | `POST /attempts/{id}/probes/{probe_id}` | Lock-in. The response carries the lock state **only**, never correctness |
| `POST /api/touches/{attemptId}/attest` | `POST /attempts/{id}/attest` | `{criterion: "correct_in_timer", claim: bool}`: the M2 self re-solve (honor), stamped before the deadline |
| `POST /api/touches/{attemptId}/end` | `POST /attempts/{id}/end` | End review: undecided criteria count as unmet |

- **The guard.** An unexported package-level `var touchesEnabled = false` in `touch.go`. No env var or flag reads it, so it's a **T-1 build-time default**, not a T-2 env flag ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)). It uses its own code `touches_disabled`, distinct from practice's fail-closed `touches_unavailable`.
  - **Tests** switch it on through a helper in the gateway package's `_test.go` files (`enableTouches(t)` sets it and restores it with `t.Cleanup`), so both branches are unit-tested.
  - **Local eyeballing** uses a build-tag file `internal/gateway/touches_dev.go` (`//go:build touchesdev`, an `init()` that sets it true) and a local `go build -tags touchesdev` of the gateway (or a scratchpad compose override that passes the tag, never committed). No committed Dockerfile, compose file, CI job or image ever passes it; a test asserts the default build has it false.
  - The shared course-scoped handler `GET /api/paths/{slug}/revision/due` (so its DSA alias `/api/revision/due` too) exposes `touchesEnabled` so the SPA picks the Touch entry or the v1 form, and each item's real **format** from `internal/course` bands (`kind`, `label`, `estMin`, `timerSecs`, `mockMode`). Both go on that one handler, not only on the alias: the SPA has called the course-scoped route since v1.7.0 ([m1-03](sprint-m1-03.md)).
  - **Flag inventory:** list it in status.md as a **T-1 build-time gate**, owner M2-04, removal M2-05 ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): every non-kill gate has an owning and a removal milestone). It spans at least one merge-to-tag window.
  - **Why:** a peer or v1-fix tag cut between this merge and v1.10.0 would otherwise let touches conclude with no `touch_concluded` producer, so review would never hear of them. M2-05 deletes the guard in the producer PR.
- Add each route to `apiRoutes()` and `docs/architecture/openapi.yaml` (drift test), and to M1-06's route-enumeration test as `withhold`-applied.
- Every 2xx write calls `g.cache.invalidate(accountID)`.

### 3 · Today in minutes (D4) [X]

A new pure package, `internal/gateway/plan`: `Build(in Input) Plan`, deterministic for (account, date) ([t0 §7](../research/t0-extensibility-frame.md#7-screens--routing)). It replaces `newWorkLimit = 3` and `buildPlan` in `internal/gateway/dashboard.go`.
- **Budget (minutes).**
  - From identity `study_budget_json` + `timezone` via the existing `GET /accounts/{id}`.
  - Mon–Fri use `weekday_minutes`. Sat–Sun use the weekend band's **lower bound**: 2h → 120, 3–4h → 180, 5h+ → 300. Record this mapping in the Decisions log.
  - The date is taken in the account timezone.
- **Minutes per item** come from the compiled manifest (`internal/course` `est_minutes` per format): the touch band's format by level, and the course-attempt format for new work.
- **Rules, in order** ([PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses)):
  1. **Due touches from every enrolled course first**, oldest due first, ties by level and then enrollment order. **Never dropped**, even over budget: the plan reports `overBudgetMin`.
  2. **R-SR5 per course.** A course with an **overdue** touch (due before the start of today, account tz, not done) gets **no new work**. The plan carries `blocked: "reviews_overdue"` for it; other courses are unaffected. Record the overdue definition in the Decisions log.
  3. **The remainder** (budget − due minutes, floored at 0) splits evenly across active, unblocked enrollments, in integer minutes, with the leftover to the earliest enrollment.
  4. Each course fills its share with frontier items, in order, while they fit. An item that doesn't fit isn't shown.
- **Course dashboard** (`GET /api/paths/{slug}/dashboard`, plus M1-03's DSA alias `GET /api/dashboard`; M1-03 put every course-scoped aggregate under `/api/paths/{slug}/…`, with no `/api/courses/…` tree): the plan for that course only. Its due touches first, then the whole remainder for that course. The cards carry `estMin`, and the header reads "Today · N min of M".
- **Agenda** (new `GET /api/agenda`, account-wide, cached per account as `agenda`): due touches from all courses (with course chip and format), then each course's share, the per-course blocked state, `budgetMin`, `plannedMin`, `overBudgetMin`, and `gradesWaiting: 0`. M4 fills that last field; the key is present now so the UI slot exists.
- **Parallel fan-out** with the per-call timeouts, as today. Every section degrades independently.

### 4 · Catalog + agenda + course Dashboard (AB05) [X]

Build to the frozen `design-system/screens/v2/AB05-*.html`.
- **`web/src/screens/Catalog.tsx`** (home `/`):
  - the **agenda** on top: "Today · N min of M"; due touches from all courses with course chip and format badge; then per-course new work; a "Reviews first — new work paused in *Course*" note for a blocked course;
  - then the **multi-course catalog** (enrolled, active, `coming_soon`, and `preview` only for the cohort per M1-04);
  - a **"grades waiting"** card slot, hidden at 0.
  - Low fidelity for non-DSA courses; full fidelity at P.
- **`web/src/screens/Dashboard.tsx`** (course dashboard): the plan cards in minutes, due touches first, the R-SR5 blocked note, and an over-budget line.
- **`web/src/lib/dashboard.ts`** gets the new types (`estMin`, `budgetMin`, `plannedMin`, `overBudgetMin`, `blocked`, `gradesWaiting`) and a `useAgenda()` hook. **`web/src/lib/touch.ts`** gets the touch hooks.

### 5 · Tests [X]

- **Screens** (vitest + `fetchMock`):
  - `Touch.test.tsx`: one test per AB04 frame; lock-ins never render correctness; End-review confirm copy; inconclusive Retry; resume at the right step; L4–5 mock-conditions badge; the coach locked;
  - `Revision.test.tsx`: format badges; the Touch entry vs the v1 form by `touchesEnabled`;
  - `Catalog.test.tsx`, `Dashboard.test.tsx`: minutes, due-first, blocked note, over-budget, grades-waiting hidden at 0.
- **Gateway** (`touch_test.go`, fake practice):
  - practice's typed errors pass through unchanged (404; 409 `touch_not_pending`/`touch_not_due`/`touch_not_lowest`/`item_withdrawn`/`probe_locked`/`touch_expired`; 503 `touches_unavailable`);
  - a resumed start returns the existing attempt id;
  - lock-in response carries no correctness;
  - the guard off (default) → 503 `touches_disabled` and `touchesEnabled: false` in `/api/paths/{slug}/revision/due` and its alias; on (via `enableTouches(t)`) → the upstream call is made;
  - no pattern in any pre-conclusion response (withhold);
  - cache invalidated on writes;
  - route enumeration and openapi drift green.
- **Planner** (`plan_test.go`, table-driven): weekday vs weekend budget in two timezones across a UTC day boundary; due-first ordering across two courses; R-SR5 blocks only its own course; over-budget reported, never dropped; even split with a leftover minute; nothing fits → no new work; same input → same plan.
- **The in-process e2e** (`internal/e2e`, CI `e2e` job, `go test -tags e2e`). It boots practice, review and assessment in-process against real Postgres and embedded JetStream, with **no gateway**, so it drives **practice's touch handlers directly** (the BFF layer and the guard are covered by `touch_test.go`):
  1. start → view (assert `probes_shown_at` stamped) → pattern lock-in → self re-solve → complexity lock-in → the concluded result;
  2. End review → fail;
  3. abandoned: after a shown view, set the touch's `timer.deadline_at` in the past through practice's store, then trigger **one ticker pass** directly (M2-01's ticker tick function, or an injectable clock/interval), with no 30 s wait → the fail with `resolution=abandoned`.

  The **review leg** (ladder advance or reset via `touch_concluded`) is asserted by feeding the concluded attempt's `touch_concluded` fixture to review in the harness. The real producer lands in M2-05, whose e2e closes the loop end to end.

### 6 · Docs [X]

- [`docs/architecture/api.md`](../../architecture/api.md) + `openapi.yaml`: the five touch routes, `GET /api/agenda`, and the dashboard's minute fields.
- [`docs/architecture/events.md`](../../architecture/events.md): no change here. The `touch_concluded` producer row is M2-05's.
- A Decisions-log line each for: the weekend-band mapping, the "overdue" definition, the `touchesEnabled` guard (removed in M2-05), and the route shape `/:course/revision/:itemId`.
- The status.md **flag inventory** row for `touchesEnabled` (T-1 build-time gate; owner M2-04; removal M2-05).

## Acceptance criteria

- [ ] **The Touch flow matches the AB04 frames**: pattern lock-in under the 2:00 clock, self re-solve, complexity lock-in, correctness hidden until the server's criteria result, "advances to Day N" / "resets to Day 1", End-review confirm, abandoned, inconclusive Retry, resume, and L4–5 mock conditions.
- [ ] **Today shows minutes and due-first ordering**: due touches from all courses first, R-SR5 blocks new work per course, the remainder is split across enrollments, and over-budget is reported, never dropped.
- [ ] Catalog and the agenda match AB05, including the "grades waiting" slot (hidden at 0).
- [ ] The touch start is unreachable in any tag built from this merge alone (the default build has `touchesEnabled` false → 503; Revision falls back to the v1 form).
- [ ] No touch response leaks the pattern, concepts or correctness before conclusion (withhold, plus the enumeration test).
- [ ] CI green: `go test ./...`, `sqlc diff` (if practice changed), web tests, openapi drift, route enumeration, the in-process e2e (`-tags e2e`).

## Release

**Merge only: ships in v1.10.0.** [m2-05](sprint-m2-05.md) removes `touchesEnabled` in the producer PR and then cuts the tag, so the Touch UI and `touch_concluded` go live in the **same** tag, one tag after the consumer (v1.9.0).
- There's no infra change: the gateway already calls practice, review and identity, so there's no NetworkPolicy, env or ACL PR, and no new pod.

## Definition of Done

CI green · merged to `main` via a squash PR (no tag) · the screens match AB04 and AB05 (checked in the browser at 1440 px and 390 px) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board row, the AB04/AB05 artboard rows set to consumed, the flag-inventory row for `touchesEnabled`) · Decisions-log lines for the budget mapping, the overdue definition and the guard.

## Risks / watch-outs

- **The UI must not ship before the producers.** The M2-02 dependency only guarantees this merges after v1.9.0. The `touchesEnabled` guard covers an interleaved tag: the Touch UI stays dark until M2-05 deletes it next to the producer.
- **Alias gaps fail honest recalls.** Without M4's claim, a synonym missing from `public:pattern` fails the criterion and resets the ladder. Show the accepted answers, and flag thin alias lists on the seeded items as content work. Honor probes are `trust=honor` either way.
- **More review load (D2 from M2-05).** Every conclusion now schedules five touches. The minute budget and the "over budget" line make this visible rather than hidden; reviews still come first by design.
- **Timezone edges.** The weekday/weekend and overdue boundaries use the account timezone and are table-tested across a UTC day boundary.
- **Timer drift and resume.** The server's `startedAt`/`deadline` are the only truth. The client ring is cosmetic and resyncs on every fetch.
- **Owner-visible change on Today.** A count of 3 becomes minutes. That is expected (D4); list it in the v1.10.0 release notes (M2-05).
