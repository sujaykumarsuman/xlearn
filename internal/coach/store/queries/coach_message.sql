-- name: InsertMessage :one
-- Append one message to a thread, labelled with the thread's course (path_slug, copied
-- from the thread so the two never disagree; NULL for an account-wide thread). seq
-- (identity) orders it; created_at is the wall time. No row when the thread is missing.
INSERT INTO coach.coach_message (thread_id, role, content, path_slug)
SELECT t.id, sqlc.arg(role), sqlc.arg(content), t.path_slug
FROM coach.coach_thread t
WHERE t.id = sqlc.arg(thread_id)
RETURNING id, seq, role, content, created_at;

-- name: ListMessages :many
-- A thread's messages oldest-first (seq is the stable total order). Used both for
-- GET /coach/thread history and to build the provider request's prior turns.
SELECT id, role, content, created_at
FROM coach.coach_message
WHERE thread_id = $1
ORDER BY seq;
