-- Every read of assessment.mock_session names its columns (never `*`, which sqlc expands
-- to every column): after v1.7.0 no query may read the v1 /35 column m1-08 drops (M1c),
-- so the rollback floor can move to 1.7.0. The three session reads share one column list
-- so their generated row types convert into each other.

-- name: InsertMockSession :one
-- Start a live mock session with the server clock (started_at / deadline_at are
-- computed by the caller so the 45-minute window is server-authoritative). status
-- defaults to 'live', total stays NULL until scoring, date defaults to today. The
-- course (path_slug, written explicitly), the rubric the session is scored against
-- (id + snapshot) and its max total are written too (m1-02, M1a); the ordinal-1
-- mock_session_item row goes in the same tx.
INSERT INTO assessment.mock_session (
    account_id, set_id, problem_id, difficulty, started_at, deadline_at,
    path_slug, rubric_id, rubric_snapshot, max_total
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, account_id, set_id, problem_id, difficulty, date, status, notes, started_at,
    deadline_at, created_at, updated_at, path_slug, rubric_id, rubric_snapshot, total,
    max_total, scored_by;

-- name: InsertMockSessionItem :exec
-- One ordered item of a session (m1-02, M1a; supersedes problem_id / set_id at M1c).
-- v1 sessions have exactly one row, ordinal 1; item_id is NULL for a mixed set
-- (problem_id ''). path_slug is the session's course, written explicitly.
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
SELECT id, account_id, set_id, problem_id, difficulty, date, status, notes, started_at,
    deadline_at, created_at, updated_at, path_slug, rubric_id, rubric_snapshot, total,
    max_total, scored_by
FROM assessment.mock_session
WHERE id = $1 AND account_id = $2;

-- name: GetMockSessionForUpdate :one
-- Read + row-lock one session scoped to its owner, so concurrent score submits on the
-- same session serialise (the second waits, re-reads status='scored', and no-ops).
SELECT id, account_id, set_id, problem_id, difficulty, date, status, notes, started_at,
    deadline_at, created_at, updated_at, path_slug, rubric_id, rubric_snapshot, total,
    max_total, scored_by
FROM assessment.mock_session
WHERE id = $1 AND account_id = $2
FOR UPDATE;

-- name: MarkMockScored :one
-- Latch a live session to scored with its rubric total + notes. The status='live' guard
-- makes a double-submit idempotent at the SQL level (no row -> already scored). Since
-- m1-03 only total / max_total / scored_by are written (m1-02 relaxed the scored CHECK
-- to accept a row with total alone).
UPDATE assessment.mock_session
SET status = 'scored', notes = $4,
    total = $3, max_total = COALESCE(max_total, $5), scored_by = 'self',
    updated_at = now()
WHERE id = $1 AND account_id = $2 AND status = 'live'
RETURNING id;

-- name: ListScoredMocks :many
-- An account's scored mocks in one course, oldest-first — the trend series (R-MK3;
-- GET /mocks/trend?path=, m1-03). total is backfilled for every v1 row (m1-02) and
-- dual-written since v1.6.0, the rollback floor after v1.7.0.
SELECT id, set_id, problem_id, difficulty, date, started_at, total, max_total
FROM assessment.mock_session
WHERE account_id = $1 AND path_slug = $2 AND status = 'scored'
ORDER BY started_at, created_at;

-- name: GetLiveMockSession :one
-- The account's live mock (m1-07): status 'live' and its 45-minute window still open,
-- newest first. It locks the coach (D27, ADR-0031 §7: coach_paused during a live mock);
-- ErrNoRows when there is none. Same column list as GetMockSession.
SELECT id, account_id, set_id, problem_id, difficulty, date, status, notes, started_at,
    deadline_at, created_at, updated_at, path_slug, rubric_id, rubric_snapshot, total,
    max_total, scored_by
FROM assessment.mock_session
WHERE account_id = $1 AND status = 'live' AND deadline_at > now()
ORDER BY started_at DESC
LIMIT 1;
