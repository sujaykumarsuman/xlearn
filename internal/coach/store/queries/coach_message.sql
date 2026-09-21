-- name: InsertMessage :one
-- Append one message to a thread. seq (identity) orders it; created_at is the wall time.
INSERT INTO coach.coach_message (thread_id, role, content)
VALUES ($1, $2, $3)
RETURNING id, seq, role, content, created_at;

-- name: ListMessages :many
-- A thread's messages oldest-first (seq is the stable total order). Used both for
-- GET /coach/thread history and to build the provider request's prior turns.
SELECT id, role, content, created_at
FROM coach.coach_message
WHERE thread_id = $1
ORDER BY seq;
