-- name: ListApiKeyConfigs :many
-- All of an account's provider key configs (0..2), stable-ordered. Includes the sealed
-- material (service-only — the HTTP layer returns only the masked view).
SELECT * FROM coach.api_key_config
WHERE account_id = $1
ORDER BY provider;

-- name: GetApiKeyConfig :one
-- One (account, provider) key config. ErrNoRows when that provider isn't connected.
SELECT * FROM coach.api_key_config
WHERE account_id = $1 AND provider = $2;

-- name: GetDefaultApiKeyConfig :one
-- The account's DEFAULT provider key — the one the coach answers with. ErrNoRows when the
-- account has no keys at all.
SELECT * FROM coach.api_key_config
WHERE account_id = $1 AND is_default
LIMIT 1;

-- name: CountApiKeyConfigs :one
-- How many providers the account has connected (drives "is this the first key?").
SELECT count(*) FROM coach.api_key_config
WHERE account_id = $1;

-- name: UpsertApiKeyConfig :one
-- Store or replace the (account, provider) key with pre-sealed material, re-enabling it.
-- $8 is_default: the store passes true only when this is the account's first key. On
-- conflict the row keeps its default flag unless $8 promotes it. The raw key never reaches
-- this layer as a column.
INSERT INTO coach.api_key_config (account_id, provider, enc_key, enc_data_key, masked_key, default_model, name, enabled, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8)
ON CONFLICT (account_id, provider) DO UPDATE
SET enc_key       = EXCLUDED.enc_key,
    enc_data_key  = EXCLUDED.enc_data_key,
    masked_key    = EXCLUDED.masked_key,
    default_model = EXCLUDED.default_model,
    name          = EXCLUDED.name,
    enabled       = true,
    is_default    = coach.api_key_config.is_default OR EXCLUDED.is_default,
    updated_at    = now()
RETURNING *;

-- name: UpdateApiKeyMeta :one
-- Update a provider's model + name WITHOUT touching the sealed key (switch model / rename).
-- ErrNoRows when that provider isn't connected.
UPDATE coach.api_key_config
SET default_model = $3, name = $4, updated_at = now()
WHERE account_id = $1 AND provider = $2
RETURNING *;

-- name: SetApiKeyEnabled :execrows
-- Flip one provider's enabled flag (Settings toggle / provider-auth failure). Returns the
-- affected row count so a no-op can 404.
UPDATE coach.api_key_config
SET enabled = $3, updated_at = now()
WHERE account_id = $1 AND provider = $2;

-- name: SetDefaultProvider :execrows
-- Make one provider the account's default and clear the others, in a single statement
-- (exactly one row matches $2 → exactly one default). The store verifies the target
-- provider exists first, so an unknown provider can't blank the default.
UPDATE coach.api_key_config
SET is_default = (provider = $2), updated_at = now()
WHERE account_id = $1;

-- name: PromoteEarliestDefault :one
-- After deleting the default, make the earliest-created remaining key the default. Does
-- nothing (ErrNoRows) when a default already exists or no keys remain. Keeps exactly one
-- default per account.
UPDATE coach.api_key_config AS k
SET is_default = true, updated_at = now()
WHERE k.id = (
    SELECT c.id FROM coach.api_key_config AS c
    WHERE c.account_id = $1
      AND NOT EXISTS (SELECT 1 FROM coach.api_key_config AS d WHERE d.account_id = $1 AND d.is_default)
    ORDER BY c.created_at, c.provider
    LIMIT 1
)
RETURNING k.provider;

-- name: DeleteApiKeyConfig :execrows
-- Remove one provider's key. Returns the affected row count so DELETE can 404 a no-op.
DELETE FROM coach.api_key_config
WHERE account_id = $1 AND provider = $2;
