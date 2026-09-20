-- name: ListWeeksWithCounts :many
-- Weeks for a path with the difficulty mix computed from the SEEDED problems
-- (real content — most weeks are 0 until the seed is expanded past the sample set).
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
    ON p.path_slug = w.path_slug AND p.week_n = w.n
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
