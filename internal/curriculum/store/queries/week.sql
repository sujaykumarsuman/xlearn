-- name: ListWeeksWithCounts :many
-- Weeks for a path with the difficulty mix computed from the SEEDED live problems, so
-- the Roadmap's per-week counts agree with the index (retired and withdrawn items are
-- out of both).
SELECT
    w.n,
    w.title,
    w.thesis,
    COUNT(p.id) FILTER (WHERE p.difficulty = 'easy') AS easy,
    COUNT(p.id) FILTER (WHERE p.difficulty = 'med')  AS med,
    COUNT(p.id) FILTER (WHERE p.difficulty = 'hard') AS hard,
    COUNT(p.id)                                       AS total
FROM curriculum.week w
LEFT JOIN curriculum.problem p
    ON p.path_slug = w.path_slug AND p.week_n = w.n AND p.status = 'live'
WHERE w.path_slug = $1
GROUP BY w.n, w.title, w.thesis
ORDER BY w.n;

-- name: GetWeek :one
SELECT n, title, thesis
FROM curriculum.week
WHERE path_slug = $1 AND n = $2;

-- name: UpsertWeek :one
INSERT INTO curriculum.week (path_slug, n, title, thesis)
VALUES ($1, $2, $3, $4)
ON CONFLICT (path_slug, n) DO UPDATE SET
    title  = EXCLUDED.title,
    thesis = EXCLUDED.thesis
RETURNING id;

-- name: DeleteMissingWeeks :exec
-- Delete-missing per course (content-only table; week_concept rows cascade).
DELETE FROM curriculum.week
WHERE path_slug = sqlc.arg(path_slug) AND NOT (n = ANY(sqlc.arg(keep)::int[]));
