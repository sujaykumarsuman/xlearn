-- name: UpsertWeakAreaSnapshot :one
-- Idempotent per (account, course, week_of): the weekly recompute upserts the same row
-- so a re-run of the tick never double-counts. top_category is NULL when the account has
-- no categorised open entries in the course this week. path_slug is written explicitly,
-- and the conflict target is the per-course unique weak_area_snapshot_account_path_week_uq
-- (m1-02, 00006) — m1-03 moved off the v1 per-account unique so m1-08 can drop it; until
-- then that v1 unique still refuses a second course's snapshot in the same week.
INSERT INTO review.weak_area_snapshot (account_id, week_of, top_category, counts_json, computed_at, path_slug)
VALUES ($1, $2, $3, $4, now(), $5)
ON CONFLICT (account_id, path_slug, week_of)
DO UPDATE SET top_category = EXCLUDED.top_category,
             counts_json = EXCLUDED.counts_json,
             computed_at = now()
RETURNING *;

-- name: GetLatestWeakArea :one
-- The account's most recent weekly snapshot in one course — the single signal both the
-- Mistakes banner and the Dashboard weak-area card read (GET /weak-area/current?path=).
SELECT * FROM review.weak_area_snapshot
WHERE account_id = $1 AND path_slug = $2
ORDER BY week_of DESC
LIMIT 1;
