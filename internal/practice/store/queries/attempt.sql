-- name: CreateAttempt :one
INSERT INTO practice.attempt (user_problem_state_id)
VALUES ($1)
RETURNING *;

-- name: GetOpenAttempt :one
-- The in-progress attempt for a problem state (not yet logged), newest first.
SELECT * FROM practice.attempt
WHERE user_problem_state_id = $1 AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1;

-- name: GetLatestAttempt :one
SELECT * FROM practice.attempt
WHERE user_problem_state_id = $1
ORDER BY started_at DESC
LIMIT 1;

-- name: SetAttemptStage :exec
UPDATE practice.attempt
SET stage_reached = $2
WHERE id = $1;

-- name: SetAttemptRevealedEarly :exec
UPDATE practice.attempt
SET revealed_early = true
WHERE id = $1;

-- name: EndAttempt :exec
UPDATE practice.attempt
SET ended_at = now()
WHERE id = $1;
