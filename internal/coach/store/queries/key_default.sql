-- coach.key_default (m1-02, M1a expand): the per-feature default key that replaces
-- api_key_config.is_default. In v1.6.0 every writer of is_default dual-writes the
-- feature='coach' row here with the same meaning; m1-10 makes this the only source
-- (and adds the 'interview' feature); M1c drops is_default.

-- name: UpsertKeyDefault :exec
-- Make key_id the account's coach default, with its model (dual-write of the first key
-- and of set-default).
INSERT INTO coach.key_default (account_id, feature, key_id, model, updated_at)
VALUES ($1, 'coach', $2, $3, now())
ON CONFLICT (account_id, feature) DO UPDATE
SET key_id = EXCLUDED.key_id, model = EXCLUDED.model, updated_at = now();

-- name: SyncKeyDefaultModel :exec
-- Keep the coach default's model equal to its key's default_model when that key's model
-- changes (a key replacement or a model switch). No row when the key isn't the default.
UPDATE coach.key_default
SET model = $2, updated_at = now()
WHERE key_id = $1 AND feature = 'coach' AND model IS DISTINCT FROM $2;

-- name: PromoteEarliestKeyDefault :exec
-- After a delete: when the account has keys but no coach default left (the cascade
-- removed it with its key), make the earliest-created key the default — the same order
-- PromoteEarliestDefault uses for is_default, so the two stay equal.
INSERT INTO coach.key_default (account_id, feature, key_id, model)
SELECT c.account_id, 'coach', c.id, c.default_model
FROM coach.api_key_config AS c
WHERE c.account_id = $1
ORDER BY c.created_at, c.provider
LIMIT 1
ON CONFLICT (account_id, feature) DO NOTHING;

-- name: ListKeyDefaults :many
-- An account's default rows (tests and m1-10's Settings panel).
SELECT account_id, feature, key_id, model, updated_at
FROM coach.key_default
WHERE account_id = $1
ORDER BY feature;
