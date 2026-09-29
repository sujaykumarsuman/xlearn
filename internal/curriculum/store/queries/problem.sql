-- Readers select the v2 columns only (m1-03): role and links. The v1 response fields
-- (the reinforcement flag and the two outbound URLs) are derived from them in Go
-- (internal/curriculum/handlers.go), so no query here reads a column M1c (m1-08) drops.
-- The seed stops writing those columns too (UpsertProblem): nullable since 00002, they
-- get their 00001 defaults ('' and false) on a new row and keep what an older seed wrote
-- on an existing one. A v1.6.0 reader (the R-b target, or a pod mid-rollout) COALESCEs
-- each of them, so it never fails on them, and a v1.6.0 boot's own seed rewrites them for
-- every item that image carries. M1c drops the columns.
--
-- Retire semantics (t1 §4): only `live` items are in the index and the counts; retired
-- and withdrawn items stay resolvable by id (GetProblem, GetProblemsByIDs).

-- name: ListProblemsByWeek :many
SELECT id, path_slug, week_n, title, difficulty, pattern, role, links
FROM curriculum.problem
WHERE path_slug = $1 AND week_n = $2 AND status = 'live'
ORDER BY sort_order, id;

-- name: ListProblemsByPath :many
-- The whole live problem index for a path (id -> week_n / pattern / difficulty /
-- role). The gateway reads this once to compose the Progress + Dashboard roll-ups
-- (by-phase completion, by-pattern mastery) without N per-problem calls.
SELECT id, path_slug, week_n, title, difficulty, pattern, role, links
FROM curriculum.problem
WHERE path_slug = $1 AND status = 'live'
ORDER BY week_n, sort_order, id;

-- name: GetProblem :one
-- Resolves any item by id, whatever its status (a retired item stays reachable).
-- contract_hash and grading_summary (m3-01) are answer-free and served on this route only.
SELECT id, path_slug, week_n, title, difficulty, pattern, role, links,
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
SELECT id, path_slug, week_n, title, difficulty, pattern, role, links,
       contract_hash, grading_summary
FROM curriculum.problem
WHERE id = ANY(sqlc.arg(ids)::text[])
ORDER BY week_n, sort_order, id;

-- name: CountProblemsByPath :one
SELECT COUNT(*) FROM curriculum.problem WHERE path_slug = $1 AND status = 'live';

-- name: UpsertProblem :execrows
-- The id guard (t1 §4): an id is never re-parented. The DO UPDATE applies only when the
-- stored row is in the same course, so a move affects 0 rows and the seed aborts unless
-- exactly 1 row is affected. Writes the v2 columns only (m1-03): the v1 flag and URL
-- columns are never written (see the header).
-- contract_hash and grading_summary (00004, m3-01) come from canon and the item's parts.
INSERT INTO curriculum.problem (
    id, path_slug, week_n, title, difficulty, pattern, sort_order,
    role, status, retired_at, links, content_hash, contract_hash, grading_summary
)
VALUES (
    sqlc.arg(id), sqlc.arg(path_slug), sqlc.arg(week_n), sqlc.arg(title), sqlc.arg(difficulty), sqlc.arg(pattern),
    sqlc.arg(sort_order), sqlc.arg(role), sqlc.arg(status),
    CASE WHEN sqlc.arg(status)::text = 'live' THEN NULL ELSE now() END,
    sqlc.arg(links), sqlc.arg(content_hash), sqlc.arg(contract_hash), sqlc.arg(grading_summary)
)
ON CONFLICT (id) DO UPDATE SET
    week_n           = EXCLUDED.week_n,
    title            = EXCLUDED.title,
    difficulty       = EXCLUDED.difficulty,
    pattern          = EXCLUDED.pattern,
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
