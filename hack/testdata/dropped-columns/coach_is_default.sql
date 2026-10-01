-- lint-expect: 1
-- lint-as: internal/coach/store/queries/api_key_config.sql
-- m1-03 allowlisted coach's is_default (it still dual-wrote the column then). m1-10 moved
-- coach onto coach.key_default as the only default source and removed the allowlist, so a
-- coach query naming is_default is now a failure like any other service's.

-- name: GetDefaultKey :one
SELECT id FROM coach.api_key_config WHERE account_id = $1 AND is_default;
