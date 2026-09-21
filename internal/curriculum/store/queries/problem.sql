-- name: ListProblemsByWeek :many
SELECT id, path_slug, week_n, title, difficulty, pattern, leetcode_url, neetcode_url, is_reinforcement
FROM curriculum.problem
WHERE path_slug = $1 AND week_n = $2
ORDER BY sort_order, id;

-- name: ListProblemsByPath :many
-- The whole problem index for a path (id -> week_n / pattern / difficulty /
-- reinforcement). The gateway reads this once to compose the Progress + Dashboard
-- roll-ups (by-phase completion, by-pattern mastery) without N per-problem calls.
SELECT id, path_slug, week_n, title, difficulty, pattern, leetcode_url, neetcode_url, is_reinforcement
FROM curriculum.problem
WHERE path_slug = $1
ORDER BY week_n, sort_order, id;

-- name: GetProblem :one
SELECT id, path_slug, week_n, title, difficulty, pattern, leetcode_url, neetcode_url, is_reinforcement
FROM curriculum.problem
WHERE id = $1;

-- name: CountProblemsByPath :one
SELECT COUNT(*) FROM curriculum.problem WHERE path_slug = $1;

-- name: UpsertProblem :exec
INSERT INTO curriculum.problem (
    id, path_slug, week_n, title, difficulty, pattern,
    leetcode_url, neetcode_url, is_reinforcement, sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET
    path_slug        = EXCLUDED.path_slug,
    week_n           = EXCLUDED.week_n,
    title            = EXCLUDED.title,
    difficulty       = EXCLUDED.difficulty,
    pattern          = EXCLUDED.pattern,
    leetcode_url     = EXCLUDED.leetcode_url,
    neetcode_url     = EXCLUDED.neetcode_url,
    is_reinforcement = EXCLUDED.is_reinforcement,
    sort_order       = EXCLUDED.sort_order;
