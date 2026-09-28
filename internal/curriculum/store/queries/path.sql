-- name: ListPaths :many
SELECT slug, title, status, summary, problem_total, week_total, sort_order
FROM curriculum.path
ORDER BY sort_order, slug;

-- name: GetPath :one
SELECT slug, title, status, summary, problem_total, week_total, sort_order
FROM curriculum.path
WHERE slug = $1;

-- name: UpsertPath :exec
-- id_prefix comes from the course manifest (00002; unique since 00003).
INSERT INTO curriculum.path (slug, title, status, summary, problem_total, week_total, sort_order, id_prefix)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (slug) DO UPDATE SET
    title         = EXCLUDED.title,
    status        = EXCLUDED.status,
    summary       = EXCLUDED.summary,
    problem_total = EXCLUDED.problem_total,
    week_total    = EXCLUDED.week_total,
    sort_order    = EXCLUDED.sort_order,
    id_prefix     = EXCLUDED.id_prefix;
