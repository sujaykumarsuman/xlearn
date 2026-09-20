# Sprint 09 — Progress projections & the Dashboard ★

> **Milestone:** M5 — mock + analytics complete   ·   **Design phase:** D5 Mock + analytics
> **Prereqs:** [S06](sprint-06.md), [S07](sprint-07.md), [S08](sprint-08.md)   ·   **Unblocks:** [S12](sprint-12.md)
> **Execute with:** [`../prompts/prompt-s09.md`](../prompts/prompt-s09.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Status |
|---|------|--------|
| 1 | projection consumers | ⬜ |
| 2 | progress API + Progress screen | ⬜ |
| 3 | BFF Dashboard aggregation + finalize Dashboard ★ | ⬜ |
| 4 | replay/rebuild verification | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Turn the events emitted by the practice/review/mock loop into fast read-model **projections** in
`assessment`, then assemble the two aggregate screens that read them: **Progress** and the finalized
**Dashboard ★**. This completes design phase D5 and reaches **M5** — a learner can see coverage,
retention, pattern mastery, a mock rubric trend and outcome mix without any cross-schema query, and the
Dashboard becomes the real "Today" home (replacing the S01 skeleton) with reviews prioritised over new
work.

## Scope

**In**
- `assessment` durable, idempotent consumers over `xlearn.practice.*`, `xlearn.review.*` and
  `xlearn.assessment.mock_completed`, building the `proj_coverage`, `proj_heatmap`, `proj_mastery`,
  `proj_outcome_mix` read-model tables (dedupe via the `assessment.inbox` table).
- `assessment` progress endpoints `GET /progress/summary`, `GET /progress/heatmap`,
  `GET /progress/mastery`, plus rubric trend read from the S08 mock tables and outcome mix from the
  projection.
- gateway BFF aggregation: `GET /progress` **agg** (assessment + review) and `GET /dashboard` **agg**
  (practice + review + assessment + curriculum).
- The **Progress** screen (`design-system/screens/Progress.dc.html`) wired to real data.
- The finalized **Dashboard ★** (`design-system/screens/Dashboard.dc.html`) replacing the S01 skeleton.
- A documented replay/rebuild procedure proving projections are a pure function of the event log.

**Out (later sprints)**
- Settings (profile/budget/timezone/reminders) and the coach panel — D6, [S10](sprint-10.md)/[S11](sprint-11.md).
- Perf hardening, empty/error states, a11y, e2e for the core loop, OpenAPI and the 1.0 cut — [S12](sprint-12.md).

## Tasks

### 1 · projection consumers

Inside `internal/assessment` add durable **pull** consumers on the JetStream streams for practice,
review and assessment. Subscribe to `xlearn.practice.*` (`attempt_logged`, `problem_solved`,
`solution_revealed_early`), `xlearn.review.*` (`revision_scheduled`, `revision_due`, `mistake_opened`,
`mistake_closed`) and `xlearn.assessment.mock_completed`. Every handler **dedupes on `event_id`** using
the `assessment.inbox` table (goose migration) so at-least-once delivery becomes effectively-once, and
**upserts** into the read-model tables so out-of-order arrival is safe (a `revision_scheduled` may land
before its `problem_solved` per [`../../architecture/events.md`](../../architecture/events.md)): map
`problem_solved`/`attempt_logged` -> `proj_coverage` + `proj_mastery` (by pattern) + `proj_outcome_mix`
(by outcome), `revision_scheduled`/solves -> `proj_heatmap` (reviews + solves per day), and
`mistake_opened`/`mistake_closed` -> `proj_outcome_mix`/weak-area context. Keep the handlers **pure
functions of the event log** (no external reads, no wall-clock branching on state) so a rebuild replays
cleanly. Add the `sqlc` queries; consumers run inside the assessment binary (no new deployable). Cross
schema stays soft: this service reads only the `assessment` schema
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md), [ADR-0004](../../adr/0004-inter-service-comms-and-events.md)).

### 2 · progress API + Progress screen

Expose the assessment-internal read endpoints from the projections: `GET /progress/summary`
(solved/151, streak, Day-7 retention, mock average), `GET /progress/heatmap` (`proj_heatmap`) and
`GET /progress/mastery` (`proj_mastery`), plus **rubric trend** read from the S08 `mock_session` +
`rubric_score` tables (`GET /mocks/trend`) and **outcome mix** from `proj_outcome_mix`. Wire
[`design-system/screens/Progress.dc.html`](../../../design-system/screens/Progress.dc.html) section for section: the
four stat tiles, the revision-activity heatmap (teal ramp), completion-by-phase table, pattern-mastery
bars (bar color from a mastery ramp, not the difficulty tokens), the mock rubric-trend line with the
W13/W15/pre-interview targets, and the outcome-mix bar + legend (Clean = `--ds-ok`, Rough = `--ds-warn`,
Assisted = `--ds-info`, Miss = `--ds-err`). The SPA reads the screen through the gateway `GET /progress`
**agg** ([`../../architecture/api.md`](../../architecture/api.md)); reuse `theme.css` verbatim.

### 3 · BFF Dashboard aggregation + finalize Dashboard ★

Implement gateway `GET /dashboard` **agg** fanning out server-side to: **curriculum** (this week's next
problems), **practice** (per-user state, streak, solved count), **review** (`GET /revision/due`,
`GET /weak-area`) and **assessment** (`proj_coverage` stats, mock best). Compose "Today": the daily plan,
due reviews, the weak-area callout, streak and stats. The **reviews-before-new-work** rule is reflected
in the plan ordering — due revisions are surfaced ahead of new problems (per the Dashboard artboard's
plan list and "Revisions due today" panel). Finalize
[`design-system/screens/Dashboard.dc.html`](../../../design-system/screens/Dashboard.dc.html) ★ with this real data,
replacing the S01 stub: the quick-stats row, Today's plan cards (difficulty chip uses the difficulty
tokens green/amber/red), week-progress bar, the Revisions-due panel with five-touch dots, and the
Weak-area card. Fan-out is 4 services — parallelize the calls and watch p95; cache only where safe.

### 4 · replay/rebuild verification

Document (and dry-run) the procedure to prove replayability: truncate a projection table (e.g.
`proj_mastery`), reset the durable consumer/offset, and **replay the JetStream stream from the start** so
the projection rebuilds by upsert to the identical result — this is safe precisely because handlers are
pure functions of the log ([`../../architecture/events.md`](../../architecture/events.md) reliability
notes). Then **reconcile** the numbers: Progress + Dashboard figures must match their source services
(practice solved count and streak, review due/weak-area, mock trend from the mock tables). Record the
rebuild runbook and any notable call as an ADR under `docs/adr/`.

## Acceptance criteria

- [ ] Progress shows coverage, revision heatmap, pattern mastery, rubric trend and outcome mix, all read from projections.
- [ ] The Dashboard ★ aggregates real daily plan + due reviews + weak area + streak/stats, with reviews prioritised over new work.
- [ ] Projections are idempotent (dedupe on `event_id`) and can be rebuilt by replaying JetStream from the start (verified).
- [ ] Deployed to prod via Flux — **M5 (mock + analytics complete)** — and Progress + Dashboard match the artboards.

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- **Projection consistency:** always upsert and dedupe on `event_id`; never assume per-subject order (a `revision_scheduled` can arrive before its `problem_solved`).
- **Replayability:** keep every projection a pure function of the event log — no external reads or wall-clock state — so a drop-and-replay rebuild is safe and deterministic.
- **Dashboard fan-out:** `GET /dashboard` hits 4 services; parallelize, set timeouts, and watch p95 — cache only read-through data that is safe to stale.
- **Reconciliation drift:** if Progress/Dashboard numbers disagree with the source services, suspect a missed event or non-idempotent handler before touching the UI.
