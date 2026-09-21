-- name: UpsertWeakAreaSnapshot :one
-- Idempotent per (account, week_of): the weekly recompute upserts the same row so a
-- re-run of the tick never double-counts. top_category is NULL when the account has no
-- categorised open entries this week.
INSERT INTO review.weak_area_snapshot (account_id, week_of, top_category, counts_json, computed_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (account_id, week_of)
DO UPDATE SET top_category = EXCLUDED.top_category,
             counts_json = EXCLUDED.counts_json,
             computed_at = now()
RETURNING *;

-- name: GetLatestWeakArea :one
-- The account's most recent weekly snapshot — the single signal both the Mistakes
-- banner and the Dashboard weak-area card read (GET /weak-area).
SELECT * FROM review.weak_area_snapshot
WHERE account_id = $1
ORDER BY week_of DESC
LIMIT 1;
