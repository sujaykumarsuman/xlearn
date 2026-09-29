-- lint-expect: 1
-- lint-as: internal/identity/store/queries/account.sql
-- The is_default allowlist is coach's alone.

-- name: GetDefault :one
SELECT id FROM identity.account WHERE is_default;
