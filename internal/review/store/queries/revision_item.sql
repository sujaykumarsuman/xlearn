-- name: ScheduleTouch :one
-- Idempotently schedule one touch. ON CONFLICT DO NOTHING so a re-delivered or
-- out-of-order practice event never double-schedules or clobbers a touch already
-- scored: on conflict the query returns no row (pgx.ErrNoRows), which the caller
-- reads as "already scheduled — do not re-emit revision_scheduled".
INSERT INTO review.revision_item (account_id, problem_id, touch_level, due_date, status)
VALUES ($1, $2, $3, $4, 'pending')
ON CONFLICT (account_id, problem_id, touch_level) DO NOTHING
RETURNING *;

-- name: ReanchorTouch :one
-- Re-anchor one touch to a fresh schedule (the fail → reset-to-Day-1 path, R-SR3):
-- create it if missing, else overwrite due_date, re-open it to pending, and clear
-- surfaced_at so the sweep re-surfaces it when due.
INSERT INTO review.revision_item (account_id, problem_id, touch_level, due_date, status)
VALUES ($1, $2, $3, $4, 'pending')
ON CONFLICT (account_id, problem_id, touch_level)
DO UPDATE SET due_date = EXCLUDED.due_date,
             status = 'pending',
             surfaced_at = NULL,
             updated_at = now()
RETURNING *;

-- name: GetRevisionItem :one
-- Fetch one touch scoped to its owner (the account from the gateway-minted JWT).
SELECT * FROM review.revision_item
WHERE id = $1 AND account_id = $2;

-- name: GetTouch :one
SELECT * FROM review.revision_item
WHERE account_id = $1 AND problem_id = $2 AND touch_level = $3;

-- name: MarkTouchPassed :one
UPDATE review.revision_item
SET status = 'passed', updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListActiveTouches :many
-- The account's live queue: every not-yet-passed touch, soonest-due first (most
-- overdue reviews lead). The caller splits due (due_date <= now) from upcoming and
-- groups by touch_level for the Revision screen. Bounded so the queue stays light.
SELECT * FROM review.revision_item
WHERE account_id = $1 AND status <> 'passed'
ORDER BY due_date ASC, touch_level ASC
LIMIT $2;

-- name: SweepDueCandidates :many
-- The periodic sweep's scan (flow 4): touches that are due, not yet surfaced, and
-- still PENDING, across all accounts. The status guard is load-bearing: a learner can
-- pass a due touch (status -> passed, surfaced_at stays NULL) before the ~15m tick, so
-- without it the sweep would emit a spurious revision_due for an already-completed
-- touch. Cheap via the (account_id, due_date, surfaced_at) index.
SELECT * FROM review.revision_item
WHERE due_date <= now() AND surfaced_at IS NULL AND status = 'pending'
ORDER BY due_date ASC
LIMIT $1;

-- name: MarkSurfaced :one
-- Idempotently latch surfaced_at. The `surfaced_at IS NULL AND status = 'pending'`
-- guard makes a re-run (or a concurrent sweep) a no-op AND closes the TOCTOU between
-- the candidate SELECT and this UPDATE — a touch passed in between returns no row, so
-- revision_due is emitted at most once and never for a settled touch.
UPDATE review.revision_item
SET surfaced_at = now(), updated_at = now()
WHERE id = $1 AND surfaced_at IS NULL AND status = 'pending'
RETURNING account_id, problem_id, touch_level;
