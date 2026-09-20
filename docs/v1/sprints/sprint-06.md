# Sprint 06 — Review scheduler & the Revision queue

> **Milestone:** none (mid-phase)   ·   **Design phase:** D4 Spaced repetition
> **Prereqs:** [S05](sprint-05.md)   ·   **Unblocks:** [S07](sprint-07.md), [S09](sprint-09.md)
> **Execute with:** [`../prompts/prompt-s06.md`](../prompts/prompt-s06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Status |
|---|------|--------|
| 1 | review service + schema + durable consumers | ⬜ |
| 2 | five-touch scheduler (event-driven) | ⬜ |
| 3 | auto-scoring (advance or reset) | ⬜ |
| 4 | periodic sweep + Revision screen | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up the **review** service and build the heart of the spaced-repetition method: the five-touch
scheduler and the Revision queue. This is the **first async consumer** in the system — it reacts to the
practice events shipped in [S05](sprint-05.md) to schedule Day 1/3/7/21/45 re-solves, auto-scores each
re-solve (advance or reset-to-Day-1), and materialises "due today" reliably even after the learner is
offline for days. It unblocks the mistake journal ([S07](sprint-07.md)) and progress projections
([S09](sprint-09.md)).

## Scope

**In**
- New **review** service (`cmd/review` + `internal/review`), schema `review`, deployed internal (ClusterIP).
- Schema `review` via goose migrations: `revision_item`, `touch_result`, `inbox`, `outbox`. (The mistake
  journal tables — `mistake_entry`, `weak_area_snapshot`, `reminder` — are created in [S07](sprint-07.md).)
- **Durable pull consumers** on `XLEARN_PRACTICE` (subjects `xlearn.practice.*`) with an idempotent
  `inbox` (dedupe on `event_id`).
- **Event-driven five-touch scheduler**: `problem_solved` (first clean) -> five `revision_item` rows;
  `solution_revealed_early` -> the owed 3-day attempt. Emits `xlearn.review.revision_scheduled` per touch.
- **Auto-scoring** endpoint: `POST /revision/{itemId}/score` records `touch_result`, decides pass/fail,
  advances the touch or resets to Day 1; Day 21/45 under mock conditions.
- **Periodic sweep** (~15m) materialising due-today (`surfaced_at`) and emitting `xlearn.review.revision_due`;
  `GET /revision/due` returns the prioritised queue.
- Wire the **Revision** screen (`design-system/screens/Revision.dc.html`): queue groups, re-solve on a
  20-min timer, auto-score result panel.
- Infra: `infra/apps/xlearn-review.yaml` HelmRelease + image-automation entry + DB role/schema/SOPS secret.

**Out (later sprints)**
- The mistake journal, weekly weak-area, and notifications: the `mistake_entry`, `weak_area_snapshot`, and
  `reminder` tables **and** their behaviour all land in [S07](sprint-07.md).
- Progress projections consuming `xlearn.review.*` (heatmap/mastery/outcome-mix) -> [S09](sprint-09.md).
- Mock scoring and the assessment service -> [S08](sprint-08.md).

## Tasks

### 1 · review service + schema + durable consumers

Scaffold the seventh Go service against the new-service checklist ([build-plan.md](../build-plan.md)):
`cmd/review/` (main + wiring) and `internal/review/` (domain, store, HTTP, consumers), reusing
`internal/platform` for config, slog JSON logging, `httpx`, and the shared `internal/platform/events` helpers.

- **Schema + migrations.** Own schema `review` per [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)
  and [data-model.md](../../architecture/data-model.md#schema-review). Embedded **goose** migrations run at
  startup under an advisory lock; queries via **sqlc/pgx**. Create: `revision_item`
  (`id`, `account_id`, `problem_id`, `touch_level` 1..5, `due_date`, `surfaced_at`, `status`
  `pending`/`passed`/`failed`), `touch_result` (`revision_item_id`, `named_pattern_secs`, `solved_in_timer`,
  `stated_complexity`, `auto_pass`, `scored_at`, plus a mock-mode flag), plus `outbox` (emits) and `inbox`
  (dedupe). (The `mistake_entry`, `weak_area_snapshot`, and `reminder` tables are added in [S07](sprint-07.md).)
- **Durable consumers.** Subscribe as **durable pull consumers** to the `XLEARN_PRACTICE` stream
  (`xlearn.practice.*`) per [ADR-0004](../../adr/0004-inter-service-comms-and-events.md) and
  [events.md](../../architecture/events.md). Every handler is **idempotent**: persist `event_id` in `inbox`
  and no-op on duplicates; never assume cross-subject ordering (upsert). Declare the `XLEARN_REVIEW`
  stream for this service's own emissions.
- **Platform surface.** `/healthz` + `/readyz`, structured slog JSON logs, `deploy/review.Dockerfile`
  (read-only rootfs), a CI path filter for `review`.
- **Infra (sibling repo).** Add `infra/apps/xlearn-review.yaml` by copying the `apps/airlift.yaml` /
  `apps/landscape.yaml` HelmRelease template (shared `charts/project`); image
  `ghcr.io/sujaykumarsuman/xlearn-review`; a Flux image-automation entry with the
  `# {"$imagepolicy": "flux-system:xlearn-review:tag"}` setter and an ImagePolicy semver `>=0.1.0`;
  the per-service DB role/schema via the infra 4-step pattern; SOPS/age secret under
  `infra/apps/secrets/*.enc.yaml`. No Traefik route (internal only). Never `kubectl apply` by hand.

### 2 · five-touch scheduler (event-driven)

Implement the core loop ([R-SR1](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition),
[events.md flow 1](../../architecture/events.md#flow-1--attempt-logged--revision-scheduled-the-core-loop)):

- On `xlearn.practice.problem_solved` with `first_solve` true (first clean solve), schedule the five
  touches at **Day 1/3/7/21/45** from the solve date -> five `revision_item` rows (`touch_level` 1..5,
  `status=pending`). Handlers **upsert** so re-delivery does not double-schedule.
- On `xlearn.practice.solution_revealed_early`, schedule the **owed attempt in 3 days**
  ([R-PF2](../../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages)) as its own follow-up touch.
- Emit `xlearn.review.revision_scheduled` (`problem_id`, `touch_level`, `due_date`) per scheduled touch via
  the transactional **outbox** (domain row + outbox row in one tx; relay publishes to `XLEARN_REVIEW`).
- Tolerate out-of-order delivery: a `problem_solved` may arrive before/after other practice events; the
  scheduler keys on `account_id`+`problem_id` and upserts idempotently.

### 3 · auto-scoring (advance or reset)

Expose scoring via the review API (surfaced by the gateway BFF as `POST /xlearn/api/revision/{itemId}/score`,
mapped from the internal `POST /revisions/{id}/score`; see [api.md](../../architecture/api.md#revision--mistakes)):

- Record a `touch_result` from the submitted `named_pattern_secs`, `solved_in_timer`, `stated_complexity`.
- **Auto-pass iff** pattern named **< 2 min** (`named_pattern_secs < 120`) **AND** solved within the
  **20-min** timer **AND** complexity stated ([R-SR2](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition)).
  Any one check failing is a fail.
- **Pass** -> `revision_item.status=passed`, advance `touch_level`, compute the next `due_date`, and emit
  `xlearn.review.revision_scheduled` for the next touch.
- **Fail** -> reset to **Day 1** (`touch_level=1`, new `due_date`), `status=failed`
  ([R-SR3](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition)). Opening the mistake entry from a fail
  is wired in [S07](sprint-07.md) (this sprint only creates the resettable state; the `mistake_entry` table
  is added in [S07](sprint-07.md)).
- **Day 21 and Day 45** run under **mock conditions** (stricter, closer to interview) per
  [R-SR4](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition) — flag the `touch_result` mock-mode.

### 4 · periodic sweep + Revision screen

Build the offline-safe sweep and the screen it feeds
([ADR-0004 hybrid](../../adr/0004-inter-service-comms-and-events.md#revision-scheduler-event-driven--periodic-sweep-hybrid),
[events.md flow 4](../../architecture/events.md#flow-4--periodic-due-sweep--reminders)):

- A cron-like sweep (~15m) selects `revision_item WHERE due_date <= now() AND surfaced_at IS NULL`, marks
  `surfaced_at` (**idempotent**, safe on re-run), and emits `xlearn.review.revision_due` per item. Keep it
  cheap: index on (`account_id`, `due_date`, `surfaced_at`).
- `GET /revision/due` returns the **prioritised** queue (due reviews before new work,
  [R-SR5](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition)), grouped by touch day.
- Wire `design-system/screens/Revision.dc.html` in `web/` (React + TS): the eyebrow + "re-solves from a
  blank editor, not re-reads" copy, the Day-grouped queue with the D1/D3/D7/D21/D45 touch dots, "Re-solve
  from blank" on a **20:00** timer, and the auto-score result panel ("pattern named / solved in-timer /
  complexity stated -> advances to Day 3" or reset-to-Day-1). Reuse `design-system/theme.css` verbatim
  (Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`).

## Acceptance criteria

- [ ] Solving a problem (first clean) schedules **5 `revision_item` rows** at Day 1/3/7/21/45; a
      `xlearn.review.revision_scheduled` event is emitted per touch.
- [ ] A re-solve is **auto-scored**: pass advances the touch (next `due_date` + event); fail resets to
      Day 1; Day 21/45 use mock conditions.
- [ ] The sweep surfaces due-today **reliably even after days offline** (idempotent `surfaced_at`), and
      `GET /revision/due` returns the queue with reviews prioritised over new work.
- [ ] **review** is deployed to prod on ClusterIP via Flux; the **Revision** screen matches the artboard.
- [ ] Re-delivered practice events do **not** double-schedule (dedupe on `event_id` + upsert).

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- **Idempotency:** re-delivered `xlearn.practice.*` events must not double-schedule — dedupe on `event_id`
  in `inbox` and upsert `revision_item` on (`account_id`, `problem_id`, `touch_level`).
- **No assumed ordering:** a `revision_scheduled` may reach assessment ([S09](sprint-09.md)) before its
  projection exists; consumers upsert. Likewise this service must not assume `problem_solved` arrives before
  other practice events.
- **Sweep must be idempotent and cheap:** `surfaced_at` guards re-emission; the `due_date`/`surfaced_at`
  index keeps the ~15m scan light on the single node.
- **Thresholds must match the PRD exactly:** pattern named **< 2 min**, solved **within the 20-min timer**,
  complexity **stated** — all three, or it is a fail (Day 21/45 stricter mock conditions).
- **New infra:** the `XLEARN_REVIEW` JetStream stream + a new DB role/schema/SOPS secret must land in
  `infra` before the pod can start; verify the durable pull consumer binds to `XLEARN_PRACTICE`.
