-- name: CreateTimer :one
INSERT INTO practice.timer (attempt_id, kind, deadline_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTimer :one
-- The latest timer of a kind for an attempt (attempt = 15m, hint = 10m).
SELECT * FROM practice.timer
WHERE attempt_id = $1 AND kind = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: ListTimers :many
SELECT * FROM practice.timer
WHERE attempt_id = $1
ORDER BY created_at;
