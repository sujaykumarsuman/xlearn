-- lint-expect: 0
-- The M1b shapes: total / max_total, the per-course conflict target, near-miss names.
-- Comments are stripped, so naming total_35, is_default or
-- ON CONFLICT (account_id, week_of) here is not a hit.

-- name: ListScoredMocks :many
SELECT id, total, max_total, xtotal_35, total_350, is_default_view -- total_35 in a trailing comment
FROM assessment.mock_session
WHERE account_id = $1 AND path_slug = $2;

-- name: UpsertWeakAreaSnapshot :one
INSERT INTO review.weak_area_snapshot (account_id, week_of, path_slug)
VALUES ($1, $2, $3)
ON CONFLICT (account_id, path_slug, week_of)
DO NOTHING
RETURNING *;
