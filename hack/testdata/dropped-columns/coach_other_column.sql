-- lint-expect: 1
-- lint-as: internal/coach/store/queries/coach_thread.sql
-- The coach allowlist covers is_default only, not the rest of the drop list.

-- name: Bad :one
SELECT total_35 FROM coach.coach_thread WHERE account_id = $1;
