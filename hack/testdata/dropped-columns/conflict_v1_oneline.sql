-- lint-expect: 1

-- name: UpsertWeakAreaSnapshot :one
INSERT INTO review.weak_area_snapshot (account_id, week_of) VALUES ($1, $2) ON CONFLICT (account_id,week_of) DO NOTHING RETURNING *;
