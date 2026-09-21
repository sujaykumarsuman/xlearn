-- name: UpsertApiKeyConfig :one
-- Store or replace an account's provider key (ADR-0007). v1 is single-key per account
-- (account_id UNIQUE), so PUT /coach/key upserts: a second key REPLACES the first,
-- re-enabling the config. Only the sealed material + the display mask are written; the
-- raw key never reaches this layer as a column.
INSERT INTO coach.api_key_config (account_id, provider, enc_key, enc_data_key, masked_key, default_model, enabled)
VALUES ($1, $2, $3, $4, $5, $6, true)
ON CONFLICT (account_id) DO UPDATE
SET provider      = EXCLUDED.provider,
    enc_key       = EXCLUDED.enc_key,
    enc_data_key  = EXCLUDED.enc_data_key,
    masked_key    = EXCLUDED.masked_key,
    default_model = EXCLUDED.default_model,
    enabled       = true,
    updated_at    = now()
RETURNING *;

-- name: GetApiKeyConfig :one
-- Read an account's key config (all columns incl. the sealed material — the service
-- decrypts in memory only for a provider call and never returns it). ErrNoRows when the
-- account has no key.
SELECT * FROM coach.api_key_config
WHERE account_id = $1;

-- name: DeleteApiKeyConfig :execrows
-- Remove an account's key. Returns the affected row count so DELETE can 404 a no-op.
DELETE FROM coach.api_key_config
WHERE account_id = $1;

-- name: SetApiKeyEnabled :execrows
-- Flip enabled without touching the sealed key. Used both by the Settings toggle and by
-- the chat path when a provider auth failure disables a bad key (ADR-0007). Returns the
-- affected row count.
UPDATE coach.api_key_config
SET enabled = $2, updated_at = now()
WHERE account_id = $1;
