-- Readers keep the v1 response shape (m1-09 is M1a expand; m1-03 switches readers to the
-- new columns). The M1c-drop columns are nullable since 00002 but dual-written by the
-- seed, so the COALESCEs only guard a row some future writer left NULL.
--
-- Retire semantics (t1 §4): only `live` items are in the index and the counts; retired
-- and withdrawn items stay resolvable by id (GetProblem, GetProblemsByIDs).

-- name: ListProblemsByWeek :many
SELECT id, path_slug, week_n, title, difficulty, pattern,
       COALESCE(leetcode_url, '')::text AS leetcode_url,
       COALESCE(neetcode_url, '')::text AS neetcode_url,
       COALESCE(is_reinforcement, role = 'reinforcement')::boolean AS is_reinforcement
FROM curriculum.problem
WHERE path_slug = $1 AND week_n = $2 AND status = 'live'
ORDER BY sort_order, id;

-- name: ListProblemsByPath :many
-- The whole live problem index for a path (id -> week_n / pattern / difficulty /
-- reinforcement). The gateway reads this once to compose the Progress + Dashboard
-- roll-ups (by-phase completion, by-pattern mastery) without N per-problem calls.
SELECT id, path_slug, week_n, title, difficulty, pattern,
       COALESCE(leetcode_url, '')::text AS leetcode_url,
       COALESCE(neetcode_url, '')::text AS neetcode_url,
       COALESCE(is_reinforcement, role = 'reinforcement')::boolean AS is_reinforcement
FROM curriculum.problem
WHERE path_slug = $1 AND status = 'live'
ORDER BY week_n, sort_order, id;

-- name: GetProblem :one
-- Resolves any item by id, whatever its status (a retired item stays reachable).
-- contract_hash and grading_summary (m3-01) are answer-free and served on this route only.
SELECT id, path_slug, week_n, title, difficulty, pattern,
       COALESCE(leetcode_url, '')::text AS leetcode_url,
       COALESCE(neetcode_url, '')::text AS neetcode_url,
       COALESCE(is_reinforcement, role = 'reinforcement')::boolean AS is_reinforcement,
       contract_hash, grading_summary
FROM curriculum.problem
WHERE id = $1;

-- name: GetProblemsByIDs :many
-- Bulk problem-metadata read: resolve many bare problem ids in ONE round-trip so the
-- gateway can enrich the Revision due queue / mistake journal without N per-id GETs
-- (ADR-0005: the gateway composes cross-context state; this keeps it a single query).
-- Ordered by the same (week_n, sort_order, id) key as the path index for stability.
-- Any status resolves (a retired item may still be on a learner's ladder).
-- contract_hash and grading_summary are selected for internal callers; the bulk route
-- does not serve them (m3-09/m3-12 decide list exposure).
SELECT id, path_slug, week_n, title, difficulty, pattern,
       COALESCE(leetcode_url, '')::text AS leetcode_url,
       COALESCE(neetcode_url, '')::text AS neetcode_url,
       COALESCE(is_reinforcement, role = 'reinforcement')::boolean AS is_reinforcement,
       contract_hash, grading_summary
FROM curriculum.problem
WHERE id = ANY(sqlc.arg(ids)::text[])
ORDER BY week_n, sort_order, id;

-- name: CountProblemsByPath :one
SELECT COUNT(*) FROM curriculum.problem WHERE path_slug = $1 AND status = 'live';

-- name: UpsertProblem :execrows
-- The id guard (t1 §4): an id is never re-parented. The DO UPDATE applies only when the
-- stored row is in the same course, so a move affects 0 rows and the seed aborts unless
-- exactly 1 row is affected. Writes the v2 columns and dual-writes the v1 ones
-- (is_reinforcement, leetcode_url, neetcode_url) from them until M1c. contract_hash and
-- grading_summary (00004, m3-01) come from canon and the item's parts.
INSERT INTO curriculum.problem (
    id, path_slug, week_n, title, difficulty, pattern,
    leetcode_url, neetcode_url, is_reinforcement, sort_order,
    role, status, retired_at, links, content_hash, contract_hash, grading_summary
)
VALUES (
    sqlc.arg(id), sqlc.arg(path_slug), sqlc.arg(week_n), sqlc.arg(title), sqlc.arg(difficulty), sqlc.arg(pattern),
    sqlc.arg(leetcode_url), sqlc.arg(neetcode_url), sqlc.arg(is_reinforcement), sqlc.arg(sort_order),
    sqlc.arg(role), sqlc.arg(status),
    CASE WHEN sqlc.arg(status)::text = 'live' THEN NULL ELSE now() END,
    sqlc.arg(links), sqlc.arg(content_hash), sqlc.arg(contract_hash), sqlc.arg(grading_summary)
)
ON CONFLICT (id) DO UPDATE SET
    week_n           = EXCLUDED.week_n,
    title            = EXCLUDED.title,
    difficulty       = EXCLUDED.difficulty,
    pattern          = EXCLUDED.pattern,
    leetcode_url     = EXCLUDED.leetcode_url,
    neetcode_url     = EXCLUDED.neetcode_url,
    is_reinforcement = EXCLUDED.is_reinforcement,
    sort_order       = EXCLUDED.sort_order,
    role             = EXCLUDED.role,
    status           = EXCLUDED.status,
    -- The first time an item leaves `live` is kept across re-seeds.
    retired_at       = CASE WHEN EXCLUDED.status = 'live' THEN NULL
                            ELSE COALESCE(curriculum.problem.retired_at, now()) END,
    links            = EXCLUDED.links,
    content_hash     = EXCLUDED.content_hash,
    contract_hash    = EXCLUDED.contract_hash,
    grading_summary  = EXCLUDED.grading_summary
WHERE curriculum.problem.path_slug = EXCLUDED.path_slug;

-- name: RetireMissingProblems :many
-- Defensive retire (t1 §4): a live item in the DB that the seed no longer carries is
-- retired, never deleted (ids.lock.json should make this impossible; the caller logs a
-- WARN per id).
UPDATE curriculum.problem
SET status = 'retired', retired_at = now()
WHERE status = 'live' AND NOT (id = ANY(sqlc.arg(seeded_ids)::text[]))
RETURNING id, path_slug;
