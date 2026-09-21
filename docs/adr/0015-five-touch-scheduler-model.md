# ADR-0015 — Five-touch scheduler model & the first durable consumer

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0014](0014-nats-jetstream-topology-and-outbox-relay.md), [events.md](../architecture/events.md), [prd §6.3](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)

## Context

S06 stands up `review` as the **first async consumer** in xLearn: the five-touch
spaced-repetition scheduler and the Revision queue. [ADR-0004](0004-inter-service-comms-and-events.md)
/ [ADR-0014](0014-nats-jetstream-topology-and-outbox-relay.md) fixed the async contract
(durable pull consumers, transactional outbox, dedupe on `event_id`); the PRD fixed the
mechanics (R-SR1..R-SR6, R-PF2). Two things were left open and are pinned here:

1. **The `revision_item` model.** The acceptance criterion says a first clean solve
   *"schedules **5 `revision_item` rows** at Day 1/3/7/21/45"* (a row-per-touch), while the
   auto-score prose describes *"advance `touch_level`"* / *"reset to `touch_level=1`"* (a
   single advancing row). These two framings need reconciling into one coherent model.
2. **The durable-consumer seam.** `internal/platform/events` shipped the producer half
   (Relay + `NatsPublisher`) in S05 with the `Consumer` interface as a placeholder; the
   concrete durable pull consumer lands now.

## Decision

### `revision_item` is row-per-touch

Each `(account_id, problem_id, touch_level)` is a distinct row — `UNIQUE
(account_id, problem_id, touch_level)`, `touch_level ∈ 1..5 = Day 1/3/7/21/45`. This
satisfies the "5 rows on first clean solve" acceptance criterion directly and makes the
Revision queue a plain `SELECT` (each due touch is its own queue entry). The unique key is
also the **idempotency key** for scheduling.

- **Schedule (on `problem_solved`, first solve **and** `outcome=clean`, R-SR1):** insert
  levels 1..5 with `due_date = solve_date + [1,3,7,21,45]d`, `status=pending`, via
  `INSERT … ON CONFLICT DO NOTHING`. Emit `revision_scheduled` per **newly-inserted** row.
  A below-clean or repeat solve is consumed (recorded in the inbox) but schedules nothing —
  opening the mistake entry is S07.
- **Owed attempt (on `solution_revealed_early`, R-PF2):** upsert the **Day-3 (level 2)**
  touch, `due_date = reveal + 3d`, `DO NOTHING`. Modelled inside the 1..5 ladder (Day 3 == the
  3-day owed re-solve) rather than adding a sixth level, so it coexists with the normal ladder
  and never double-schedules.
- **Auto-score (`POST /revisions/{id}/score`, R-SR2):** `auto_pass = named_pattern_secs < 120
  AND solved_in_timer AND stated_complexity` — recorded on a `touch_result` row (the durable
  audit). `mock_mode = touch_level ∈ {4,5}` (Day 21/45, R-SR4). **Idempotent:** only a `pending`
  touch is scored; re-scoring a settled (`passed`) touch is a no-op that returns the prior
  outcome, so a double-submit/replay never writes a duplicate `touch_result` or re-emits
  `revision_scheduled` with a fresh `event_id`.
  - **Pass:** mark the scored touch `passed`; emit `revision_scheduled` for the next touch
    (which already exists from the initial schedule — created lazily if absent). Day-45 pass
    completes the ladder.
  - **Fail (R-SR3):** **re-anchor the whole ladder to Day 1** — every touch for the problem is
    reset to `pending` with `due_date = now + [1,3,7,21,45]d` and `surfaced_at` cleared, so the
    problem restarts. The failure is durably recorded in `touch_result.auto_pass=false`
    (`revision_item.status` `failed` is retained in the schema for the S07 mistake linkage).
- **Sweep (~15m, R-SR6):** `SELECT … WHERE due_date <= now() AND surfaced_at IS NULL AND
  status = 'pending'`, latch `surfaced_at` under the same guard (idempotent, and TOCTOU-safe),
  emit `revision_due` per item. The `status = 'pending'` predicate is load-bearing: a learner
  can pass a due touch before the tick (status → passed, `surfaced_at` still NULL), and without
  it the sweep would emit a spurious `revision_due` for an already-completed touch. Indexed on
  `(account_id, due_date, surfaced_at)`. Catches up every missed date after the learner is
  offline for days.

### The durable pull consumer

- `events.NatsConsumer` (shared `internal/platform/events`) creates a **durable pull consumer**
  (`AckExplicit`, `DeliverAll`, `MaxDeliver` cap) via `CreateOrUpdateConsumer`, retrying until
  the producer-owned stream exists. It acks on handler success and, on error, **naks with an
  escalating backoff** (`NakWithDelay` + a `BackOff` schedule, 1s→5m) so `MaxDeliver` spans hours
  — a transient DB/broker outage is a bounded, recoverable retry, never an instant budget burn
  that would silently drop a `problem_solved`. The handler runs on a context **decoupled from the
  shutdown signal** (bounded `Background` context) and the subscription is **stopped before the
  HTTP drain**, so a rolling deploy neither cancels an in-flight schedule nor dispatches onto a
  cancelled context.
- **Effectively-once** is the inbox: every consumed event's `event_id` is written to
  `review.inbox` in the **same transaction** as its side effects; a re-delivery conflicts on the
  PK and no-ops. Layered on the broker's 5-minute `Msg-Id` dedupe window (ADR-0014).
- review **owns `XLEARN_REVIEW`** (its emissions) and **subscribes to `XLEARN_PRACTICE`**
  (durable `review`, filter `xlearn.practice.*`). `attempt_logged` matches the filter but is a
  no-op (consumed by assessment). No cross-subject ordering is assumed — everything upserts.

## Consequences

- The "5 rows" acceptance and the "advance/reset" prose are both satisfied by one model; a
  fail genuinely restarts the problem (all touches re-pend) rather than stranding a mid-ladder
  `failed` row that could never re-surface.
- Re-delivered or out-of-order practice events never double-schedule (inbox dedupe **and**
  `ON CONFLICT DO NOTHING`).
- `revision_item.status` at rest is effectively `pending`/`passed` (a fail re-anchors to
  pending in the same tx); the durable fail signal lives in `touch_result`. S07 reads
  `touch_result.auto_pass=false` to open/re-open mistakes.
- The consumer is offline-safe: the durable's server-side offset + the sweep together mean a
  pod restart or a days-long outage loses no scheduling.

## Alternatives considered

- **Single advancing `revision_item` row per problem** (mutate `touch_level`). Cleaner
  reset, but fails the literal "5 rows on first clean solve" acceptance and makes the
  Day-grouped queue a derived projection. Rejected.
- **A sixth `touch_level` (or a `kind` column) for the R-PF2 owed attempt.** More faithful to
  "a separate follow-up touch", but complicates the 1..5 domain and the queue grouping for no
  user-visible gain — the owed re-solve *is* a Day-3 re-solve. Rejected; folded into level 2.
- **Persisting `failed` and resetting only the Day-1 touch.** Leaves levels 2..5 with stale
  past due-dates (all overdue → surfaced at once, jumping ahead) or a stranded failed level.
  Rejected in favour of re-anchoring the whole ladder.
- **JetStream durable offset as the only dedupe** (no inbox). The offset dedupes redelivery of
  the *same* consumer, but the transactional inbox is what makes a handler's DB side effects
  exactly-once across crashes. Kept the inbox per ADR-0004.
