-- coach.key_default(account_id, feature) — the PER-FEATURE default key.
--
-- m1-02 (M1a) created it beside api_key_config.is_default and dual-wrote the
-- feature='coach' row. m1-10 (here) makes it the ONLY default source: is_default is no
-- longer read or written by coach, and `interview` joins `coach` as a feature. m1-08
-- (M1c) then drops the column.
--
-- Two features, deliberately asymmetric:
--   * coach     — always present once the account has any key. The first key becomes it;
--                 deleting it promotes the earliest surviving key. Chat resolves through it.
--   * interview — only ever set EXPLICITLY (t6 §11 / m6a-02). Never auto-assigned and
--                 never auto-promoted: a learner who has not chosen an interview brain
--                 has none, and "Not set" is a real state the UI shows. Its model may be
--                 a catalog id with the interview_brain capability or a custom id.
--
-- A row dies with its key (key_id … ON DELETE CASCADE, migration 00004).

-- name: GetKeyDefault :one
-- The account's default key for one feature, with everything a provider call needs: the
-- key's sealed material (both pairs) and enabled flag, plus the FEATURE's chosen model.
-- ErrNoRows when that feature has no default (no keys at all, or `interview` never set).
--
-- d.model is the feature's model and may differ from the key's own default_model — the
-- caller prefers d.model and falls back to k.default_model, then to the catalog default.
SELECT k.id, k.account_id, k.provider, k.enc_key, k.enc_data_key, k.masked_key,
       k.default_model, k.name, k.enabled, k.created_at, k.updated_at,
       k.enc_key_ad, k.enc_data_key_ad, k.kek_id, k.ad_src_digest,
       d.model AS feature_model
FROM coach.key_default AS d
JOIN coach.api_key_config AS k ON k.id = d.key_id
WHERE d.account_id = $1 AND d.feature = $2;

-- name: GetKeyDefaultKeyID :one
-- The key id backing one feature's default, or ErrNoRows. The store marks
-- KeyConfig.IsDefault from the feature='coach' answer, so GET /keys's v1 JSON
-- (keys[].is_default, default_provider) and the coach itself read the same source — an
-- open v1.6.0 tab and this release agree about which provider answers.
SELECT key_id FROM coach.key_default
WHERE account_id = $1 AND feature = $2;

-- name: UpsertKeyDefault :exec
-- Set one feature's default key and model (the first key for `coach`, an explicit
-- set-default for either feature).
INSERT INTO coach.key_default (account_id, feature, key_id, model, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (account_id, feature) DO UPDATE
SET key_id = EXCLUDED.key_id, model = EXCLUDED.model, updated_at = now();

-- name: SyncKeyDefaultModel :exec
-- Keep the COACH default's model equal to its key's default_model when that key's model
-- changes (a key replacement or a model switch in Settings). Scoped to feature='coach' on
-- purpose: an `interview` default pointing at the same key keeps its own chosen brain,
-- which a coach-side model switch must not clobber.
UPDATE coach.key_default
SET model = $2, updated_at = now()
WHERE key_id = $1 AND feature = 'coach' AND model IS DISTINCT FROM $2;

-- name: PromoteEarliestKeyDefault :exec
-- After a delete: when the account still has keys but the named feature has no default
-- left (the cascade removed it with its key), make the earliest-created key the default,
-- carrying that key's own model. The store calls this for `coach` ONLY — `interview` is
-- never auto-promoted.
INSERT INTO coach.key_default (account_id, feature, key_id, model)
SELECT c.account_id, sqlc.arg(feature), c.id, c.default_model
FROM coach.api_key_config AS c
WHERE c.account_id = sqlc.arg(account_id)
ORDER BY c.created_at, c.provider
LIMIT 1
ON CONFLICT (account_id, feature) DO NOTHING;

-- name: ListKeyDefaults :many
-- Every default the account has, with the provider of the key behind it — what the
-- Settings panel renders as per-feature defaults (AB01 F13) and what GET /keys returns as
-- `defaults`. A feature with no default is simply absent (the UI shows "Not set").
SELECT d.feature, d.model, d.updated_at, k.provider, k.id AS key_id
FROM coach.key_default AS d
JOIN coach.api_key_config AS k ON k.id = d.key_id
WHERE d.account_id = $1
ORDER BY d.feature;

-- name: DeleteKeyDefault :exec
-- Clear one feature's default without touching the key (used to unset an `interview`
-- brain). `coach` is never cleared this way — it is promoted, not emptied.
DELETE FROM coach.key_default
WHERE account_id = $1 AND feature = $2;
