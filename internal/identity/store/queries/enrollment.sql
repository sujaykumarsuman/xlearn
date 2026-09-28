-- name: StartEnrollment :one
-- Idempotent: the first call records started_at (default now()); re-starting only
-- re-activates the row and keeps the ORIGINAL started_at (it is not in the SET),
-- so a learner's "current day" never resets on a repeat Start. public_visible is the
-- course manifest's public_stats.default_visible (m1-02, M1a; D7), written on insert
-- only: a re-start never overwrites the learner's own choice (m2-03's toggle).
INSERT INTO identity.path_enrollment (account_id, path_slug, public_visible)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, path_slug)
DO UPDATE SET status = 'active'
RETURNING *;

-- name: ListEnrollments :many
SELECT * FROM identity.path_enrollment
WHERE account_id = $1
ORDER BY started_at;
