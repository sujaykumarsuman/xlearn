-- lint-expect: 0
-- lint-as: internal/coach/store/queries/api_key_config.sql
-- coach's is_default is allowlisted until m1-10: a note, not a failure.

-- name: GetDefaultKey :one
SELECT id FROM coach.api_key_config WHERE account_id = $1 AND is_default;
