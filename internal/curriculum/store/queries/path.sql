-- name: ListPaths :many
SELECT slug, title, status, summary, problem_total, week_total, sort_order
FROM curriculum.path
ORDER BY sort_order, slug;

-- name: GetPath :one
SELECT slug, title, status, summary, problem_total, week_total, sort_order
FROM curriculum.path
WHERE slug = $1;

-- name: UpsertPath :exec
INSERT INTO curriculum.path (slug, title, status, summary, problem_total, week_total, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (slug) DO UPDATE SET
    title         = EXCLUDED.title,
    status        = EXCLUDED.status,
    summary       = EXCLUDED.summary,
    problem_total = EXCLUDED.problem_total,
    week_total    = EXCLUDED.week_total,
    sort_order    = EXCLUDED.sort_order;
