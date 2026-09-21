# ADR-0018 — Progress projections: event grain, gateway roll-up & drop-and-replay rebuild

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0013](0013-bff-week-aggregation-userstate-contract.md), [0017](0017-mock-model-and-projection-consumer-scaffold.md), [events.md](../architecture/events.md), [data-model.md](../architecture/data-model.md), [runbook](../runbooks/projection-rebuild.md)

## Context

S09 turns the practice/review/mock event stream into the read-model projections in
`assessment` that back the **Progress** screen and the **Dashboard ★**
([S09](../v1/sprints/sprint-09.md)). [ADR-0017](0017-mock-model-and-projection-consumer-scaffold.md)
wired the durable consumers + the idempotent `inbox`; this ADR pins **what the
projections store**, **who composes the taxonomy**, and **how a rebuild is proven safe**.

Two constraints collide with the data-model's *indicative* column list
(`proj_coverage(week_n, total)`, `proj_mastery(pattern)`):

1. **The events carry only a bare `problem_id`.** `xlearn.practice.problem_solved` is
   `{problem_id, outcome, first_solve}`; `xlearn.review.revision_scheduled` is
   `{problem_id, touch_level, due_date}`. Neither carries the problem's **week** or
   **pattern** — those are `curriculum` facts.
2. **Projections must be a pure function of the event log** (no external reads, no
   wall-clock state branching) so a drop-and-replay rebuild is deterministic
   ([events.md](../architecture/events.md) replay note, [ADR-0005](0005-data-ownership-and-migrations.md)
   no cross-schema reads).

Resolving `problem_id → pattern/week` at projection time would require a cross-schema
read of `curriculum` inside the consumer, which breaks **both** constraints (it is an
external read, and a later curriculum edit would make a replay non-deterministic).

## Decision

### Projections are keyed at the EVENT grain; the gateway composes the taxonomy

The four `assessment` read-model tables (`00002` migration) are keyed only by data the
events actually supply:

- `proj_coverage(account_id, problem_id, solved, first_solved_at, level1_schedules)` —
  the solved set + the spaced-repetition ladder anchor count.
- `proj_mastery(account_id, problem_id, best_outcome, best_rank, clean_solves, solve_count)` —
  per-problem solve **quality** (`best_rank`: clean=4 … miss=1).
- `proj_heatmap(account_id, activity_date, solves, reviews)` — per-**UTC-day** activity.
- `proj_outcome_mix(account_id, outcome, cnt)` — first-solve outcome counts.

The **gateway** `GET /progress` **agg** composes the by-**phase** completion and the
by-**pattern** mastery by joining these per-problem rows with the `curriculum` taxonomy,
fetched once via a new bulk read `GET /paths/{slug}/problems` (id → week_n / pattern /
difficulty / reinforcement). This is the same BFF-composition role the gateway already
plays for the Week / Problem / Revision / Mistake screens ([ADR-0013](0013-bff-week-aggregation-userstate-contract.md)):
`assessment` owns the *numbers*, `curriculum` owns the *grouping*. Reinforcement problems
are excluded from the core roll-ups.

### Metric definitions (all derivable from the log, order-independent)

- **Solved / N** = `COUNT(proj_coverage WHERE solved)`; N is curriculum's live
  `path.problem_total` (the gateway overrides `assessment`'s fixed-151 default).
- **Streak** (current / longest) is derived **at read time** from `proj_heatmap` (a day
  is active with any solve or review). Read-time derivation with `now()` is fine — the
  stored projection stays pure; the query is deterministic given the rows + clock.
- **Day-7 retention** = `1 − resets / ladders`, where `ladders` = problems with a Day-1
  anchor and `resets` = `Σ GREATEST(level1_schedules − 1, 0)`. A failed re-solve
  re-schedules a Day-1 touch, so the 2nd+ `revision_scheduled(touch_level=1)` per problem
  is a reset — counted by incrementing `level1_schedules` and computing resets at read
  time (order-independent under replay).
- **Pattern mastery pct** = `Σ weight(best_rank) / total_in_pattern` (weights
  1 / 0.7 / 0.45 / 0.25); **count** = solved / total in pattern.
- **Outcome mix** counts each problem's **first** solve by outcome (sums to solved).

### Every write is an idempotent, commutative UPSERT

Accumulation is addition (`cnt + 1`), `GREATEST` (best rank) or `LEAST` (earliest solve)
— all commutative — and each runs in the **same transaction** as the `inbox` claim, so
`inbox ⇔ projected` is atomic (effectively-once). Out-of-order delivery is therefore safe
(a `revision_scheduled` may arrive before its `problem_solved`), and a **drop-and-replay
from the stream start** rebuilds a byte-identical read model. This is verified by a
real-Postgres integration test (`projections build … and rebuild on replay`) and
documented in the [rebuild runbook](../runbooks/projection-rebuild.md).

### Dashboard aggregation

`GET /dashboard` **agg** fans out **in parallel** (per-call timeouts) to `assessment`
(streak / solved / mock stats + the solved set), `review` (`/revisions/due`,
`/weak-area/current`, `/reminders`) and `curriculum` (roadmap + the bulk problem index),
then composes "Today": due reviews **lead** the daily plan (reviews-before-new-work,
R-SR5), followed by the current week's next unsolved problems. Streak + solved come from
the `assessment` projections (the aggregate of practice events); a single `practice`
`GET /state` call for the current week supplies the live per-problem status.

## Consequences

- No cross-schema read at projection time; replay is deterministic and cheap to reason
  about. The projections diverge from the data-model's *indicative* `week_n`/`pattern`
  columns **on purpose** — the taxonomy lives in the gateway, not the read model.
- A new, additive, read-only `curriculum` endpoint (`GET /paths/{slug}/problems`) lets the
  gateway build both roll-ups in **one** call instead of N per-problem calls.
- Historical events consumed as no-op stubs in the S08→S09 window are recorded in the
  `inbox`; the projections are **forward-only from S09** by design. A backfill is a
  distinct one-off replay from the stream start (the runbook), not the live inbox-gated
  path — consistent with [ADR-0017](0017-mock-model-and-projection-consumer-scaffold.md).
- Heatmap days are **UTC**; a per-account-timezone heatmap/streak is a later refinement.

## Alternatives considered

- **Enrich the event with pattern/week at emit time (in practice/review).** Rejected:
  practice/review don't own the taxonomy either (it's curriculum's), so they'd need the
  same cross-schema read, and baking a mutable fact into an immutable event breaks the
  additive-only event contract and replay determinism.
- **Resolve pattern/week in the consumer via a curriculum call.** Rejected: an external
  read in the projection path breaks the pure-function-of-the-log rule; a curriculum edit
  would then change a replay's result.
- **Per-problem curriculum calls in the gateway roll-up (no bulk endpoint).** Rejected:
  N calls per Progress/Dashboard load blows the p95; the bulk index is one cached read.
