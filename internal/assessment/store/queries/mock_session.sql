-- name: InsertMockSession :one
-- Start a live mock session with the server clock (started_at / deadline_at are
-- computed by the caller so the 45-minute window is server-authoritative). status
-- defaults to 'live', total_35 stays NULL until scoring, date defaults to today.
-- m1-02 (M1a) also writes the course, the rubric the session is scored against (id +
-- snapshot) and its max total; the ordinal-1 mock_session_item row goes in the same tx.
INSERT INTO assessment.mock_session (
    account_id, set_id, problem_id, difficulty, started_at, deadline_at,
    path_slug, rubric_id, rubric_snapshot, max_total
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: InsertMockSessionItem :exec
-- One ordered item of a session (m1-02, M1a; supersedes problem_id / set_id at M1c).
-- v1 sessions have exactly one row, ordinal 1; item_id is NULL for a mixed set
-- (problem_id '').
INSERT INTO assessment.mock_session_item (session_id, ordinal, item_id, path_slug)
VALUES ($1, $2, $3, $4);

-- name: ListMockSessionItems :many
-- A session's ordered items (tests and the M1b readers).
SELECT session_id, ordinal, item_id, path_slug, contract_hash
FROM assessment.mock_session_item
WHERE session_id = $1
ORDER BY ordinal;

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
-- makes a double-submit idempotent at the SQL level (no row -> already scored). m1-02
-- (M1a) dual-writes total / max_total / scored_by beside total_35.
UPDATE assessment.mock_session
SET status = 'scored', total_35 = $3, notes = $4,
    total = $3, max_total = COALESCE(max_total, $5), scored_by = 'self',
    updated_at = now()
WHERE id = $1 AND account_id = $2 AND status = 'live'
RETURNING id;

-- name: ListScoredMocks :many
-- An account's scored mocks oldest-first — the trend series (R-MK3). The total is read
-- as COALESCE(total, total_35) (m1-02, M1a).
SELECT id, set_id, problem_id, difficulty, date, started_at, COALESCE(total, total_35) AS total_35
FROM assessment.mock_session
WHERE account_id = $1 AND status = 'scored'
ORDER BY started_at, created_at;
