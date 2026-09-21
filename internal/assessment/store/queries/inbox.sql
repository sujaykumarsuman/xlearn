-- name: InsertInbox :one
-- Record a consumed event's id for idempotency. ON CONFLICT DO NOTHING so a
-- re-delivered event returns no row (pgx.ErrNoRows) -> the handler no-ops instead of
-- re-applying its side effects (effectively-once, ADR-0004). The S09 projection writes
-- slot into the SAME transaction as this claim so inbox <-> projected stays atomic.
INSERT INTO assessment.inbox (event_id)
VALUES ($1)
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id;
