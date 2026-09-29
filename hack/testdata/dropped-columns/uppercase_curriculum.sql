-- lint-expect: 1
-- Case-insensitive: SQL identifiers fold, so an upper-case drop-list column is a hit
-- (and each of the curriculum columns is on the list).

-- name: ListProblems :many
SELECT id, IS_REINFORCEMENT, "leetcode_url", p.neetcode_url, sqlc.arg(code_template)
FROM curriculum.problem AS p;
