-- name: GetUserProblemState :one
SELECT * FROM practice.user_problem_state
WHERE account_id = $1 AND problem_id = $2;

-- name: ListUserProblemStates :many
SELECT * FROM practice.user_problem_state
WHERE account_id = sqlc.arg(account_id)
  AND problem_id = ANY(sqlc.arg(problem_ids)::text[]);

-- name: UpsertUserProblemState :one
-- Get-or-create the (account, problem) state row, touching updated_at so the row is
-- always returned (ON CONFLICT DO NOTHING would return nothing on the resume path).
INSERT INTO practice.user_problem_state (account_id, problem_id, status)
VALUES ($1, $2, 'available')
ON CONFLICT (account_id, problem_id)
DO UPDATE SET updated_at = now()
RETURNING *;

-- name: SetStateAttempting :one
UPDATE practice.user_problem_state
SET status = 'attempting', updated_at = now()
WHERE account_id = $1 AND problem_id = $2
RETURNING *;

-- name: SetStateSolved :one
-- Mark solved, latching first_solved_at on the first solve and recording the last
-- outcome. current_touch stays as-is (review owns the five-touch progression).
UPDATE practice.user_problem_state
SET status = 'solved',
    last_outcome = $3,
    first_solved_at = COALESCE(first_solved_at, now()),
    updated_at = now()
WHERE account_id = $1 AND problem_id = $2
RETURNING *;
