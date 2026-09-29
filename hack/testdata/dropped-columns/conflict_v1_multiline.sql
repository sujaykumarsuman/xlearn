-- lint-expect: 1
-- The retired v1 weak-area conflict target, split over lines and oddly cased: still a
-- hit (m1-08 drops the UNIQUE (account_id, week_of) it names).

-- name: UpsertWeakAreaSnapshot :one
INSERT INTO review.weak_area_snapshot (account_id, week_of, path_slug)
VALUES ($1, $2, $3)
On
  CONFLICT(
    Account_ID ,
	week_of
  )
DO UPDATE SET computed_at = now()
RETURNING *;
