# Runbook — rebuilding the `assessment` progress projections

**When:** a projection is suspected wrong (Progress / Dashboard numbers disagree with the
source services), after a projection **schema/logic change**, or to **backfill** history
that was consumed as a no-op in the S08→S09 window.

**Why it is safe:** the projections are a **pure function of the event log**
([ADR-0018](../adr/0018-progress-projection-grain-and-rebuild.md)). Every write is an
idempotent, commutative UPSERT applied in the same transaction as the `inbox` claim, and
JetStream retains the `XLEARN_PRACTICE` / `XLEARN_REVIEW` streams, so dropping a projection
and replaying from the stream start rebuilds an **identical** result.

> **Scope:** touch only schema `assessment`. Never `kubectl apply` — this is a data
> operation run against the database + NATS, not an infra change.

## What feeds what

| Event (subject) | Projection effect |
|---|---|
| `xlearn.practice.problem_solved` | `proj_coverage` (solved + earliest solve), `proj_mastery` (quality), `proj_heatmap.solves`, `proj_outcome_mix` (first solve) |
| `xlearn.review.revision_scheduled` | `proj_heatmap.reviews`; `touch_level=1` → `proj_coverage.level1_schedules` (ladder anchor / reset) |
| others (`attempt_logged`, `solution_revealed_early`, `revision_due`, `mistake_opened/closed`) | recorded in `inbox` (dedupe), no projection change |

## A · Verified dry-run (proves replayability)

Reproduced by the store integration test
`internal/assessment/store/store_integration_test.go` →
`projections build coverage/mastery/heatmap/outcome-mix and rebuild on replay`:

1. Apply a fixed event log; snapshot the read model (solved, ladders/resets, outcome mix,
   mastery, heatmap totals).
2. Truncate `proj_coverage` / `proj_mastery` / `proj_heatmap` / `proj_outcome_mix` and
   clear the replayed `inbox` rows.
3. Re-apply the **same** log; assert the read model is identical.

Run it locally against a throwaway Postgres:

```bash
XLEARN_TEST_DATABASE_URL='postgres://xlearn_assessment:PW@localhost:5599/xlearndb?sslmode=disable' \
  go test ./internal/assessment/store/ -run TestStoreIntegration -count=1 -v
```

## B · Full rebuild in an environment

1. **Quiesce writers (optional).** A rebuild is safe concurrently (upserts are
   commutative), but for a clean reconcile, pause the live consumers by scaling the
   `assessment` Deployment to 0 replicas via the GitOps flow, or accept eventual
   convergence and skip this.
2. **Truncate the projection tables** (NOT `mock_session` / `rubric_score` / `outbox`):
   ```sql
   TRUNCATE assessment.proj_coverage, assessment.proj_mastery,
            assessment.proj_heatmap, assessment.proj_outcome_mix;
   ```
3. **Reset the durable consumers** so they re-deliver from the stream start. The live
   consumers are `DeliverNew` and keyed on the `inbox` for dedupe, so a rebuild uses a
   **distinct one-off replay consumer** per stream (durable name e.g. `assessment-rebuild`,
   `DeliverAllPolicy`) rather than the live path — and the `inbox` must be cleared for the
   events being replayed (or the replay consumer must not share the live `inbox`):
   ```sql
   TRUNCATE assessment.inbox;   -- only during a full rebuild while writers are paused
   ```
   Then run the one-off replay (a short script that Subscribes with
   `events.NewNatsConsumer` + `DeliverAllPolicy`, no `WithDeliverNew()`, over
   `xlearn.practice.*` and `xlearn.review.*`, applying `store.ApplyProjection`).
4. **Resume** the live consumers (scale back to 1). They pick up from their committed
   offset; any events that arrived during the rebuild re-apply idempotently.

## C · Reconcile

After a rebuild, the projection numbers must match the source services:

| Progress / Dashboard number | Source of truth to compare against |
|---|---|
| Solved / N | `practice.user_problem_state` count of `status='solved'` |
| Current / longest streak | `practice` solve dates + `review.revision_item` schedule dates |
| Day-7 retention (resets) | `review.revision_item` Day-1 re-anchors (a reset resets the ladder) |
| Revisions due | `review` `GET /revisions/due` `dueCount` |
| Mock average / best / trend | `assessment.mock_session` (status `scored`) |

If a number still disagrees after a clean rebuild, suspect a **missed event** (an outbox
row never relayed) or a **non-idempotent handler** before touching the UI
([sprint-09 risks](../v1/sprints/sprint-09.md#risks--watch-outs)).
