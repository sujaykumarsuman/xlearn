-- name: InsertInbox :one
-- Record a consumed event's id for idempotency. ON CONFLICT DO NOTHING so a
-- re-delivered event returns no row (pgx.ErrNoRows) → the handler no-ops instead of
-- re-applying its side effects (effectively-once, ADR-0004).
INSERT INTO review.inbox (event_id)
VALUES ($1)
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id;
