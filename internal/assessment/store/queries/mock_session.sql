-- name: InsertMockSession :one
-- Start a live mock session with the server clock (started_at / deadline_at are
-- computed by the caller so the 45-minute window is server-authoritative). status
-- defaults to 'live', total_35 stays NULL until scoring, date defaults to today.
INSERT INTO assessment.mock_session (account_id, set_id, problem_id, difficulty, started_at, deadline_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetMockSession :one
-- Read one session scoped to its owner (soft account ownership check).
SELECT * FROM assessment.mock_session
WHERE id = $1 AND account_id = $2;

-- name: GetMockSessionForUpdate :one
-- Read + row-lock one session scoped to its owner, so concurrent score submits on the
-- same session serialise (the second waits, re-reads status='scored', and no-ops).
SELECT * FROM assessment.mock_session
WHERE id = $1 AND account_id = $2
FOR UPDATE;

-- name: MarkMockScored :one
-- Latch a live session to scored with its /35 total + notes. The status='live' guard
-- makes a double-submit idempotent at the SQL level (no row -> already scored).
UPDATE assessment.mock_session
SET status = 'scored', total_35 = $3, notes = $4, updated_at = now()
WHERE id = $1 AND account_id = $2 AND status = 'live'
RETURNING id;

-- name: ListScoredMocks :many
-- An account's scored mocks oldest-first — the trend series (R-MK3).
SELECT id, set_id, problem_id, difficulty, date, started_at, total_35
FROM assessment.mock_session
WHERE account_id = $1 AND status = 'scored'
ORDER BY started_at, created_at;
