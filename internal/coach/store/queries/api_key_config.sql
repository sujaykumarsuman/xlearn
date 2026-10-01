-- coach.api_key_config — one envelope-encrypted provider key per (account, provider).
--
-- EXPLICIT COLUMN LISTS, never `SELECT *` / `RETURNING *` (m1-10). Two reasons:
--   * sqlc expands a star to the column set it saw at generate time, so generated code
--     reads every column positionally. When m1-08 (M1c) drops is_default, a star query
--     would scan a column count that no longer matches and fail at runtime — the drop's
--     whole point is that nothing touches it first.
--   * is_default is DEAD from m1-10 on: coach.key_default(account_id, feature) is the
--     only default source. Naming the columns is what makes "no reader, no writer" a
--     checkable property (internal/coach/contract_test.go greps for both, and
--     hack/lint-dropped-columns.sh no longer allowlists coach).
--
-- The sealed material comes back on the read paths (the service opens it in memory for a
-- provider call); the HTTP layer only ever serialises the masked view. Two pairs are
-- stored during M1b: the LEGACY unbound pair (enc_key, enc_data_key), still readable by
-- v1.6.0 so the rollback floor holds, and the AD-BOUND pair (enc_key_ad,
-- enc_data_key_ad) under keyring entry kek_id, tied to the legacy pair it was sealed
-- beside by ad_src_digest = sha256(enc_data_key). See migration 00006 and rewrap.go.

-- name: ListApiKeyConfigs :many
-- All of an account's provider key configs (0..2), stable-ordered. Includes both sealed
-- pairs (service-only — the HTTP layer returns only the masked view).
SELECT id, account_id, provider, enc_key, enc_data_key, masked_key, default_model, name,
       enabled, created_at, updated_at, enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest
FROM coach.api_key_config
WHERE account_id = $1
ORDER BY provider;

-- name: GetApiKeyConfig :one
-- One (account, provider) key config. ErrNoRows when that provider isn't connected.
SELECT id, account_id, provider, enc_key, enc_data_key, masked_key, default_model, name,
       enabled, created_at, updated_at, enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest
FROM coach.api_key_config
WHERE account_id = $1 AND provider = $2;

-- name: CountApiKeyConfigs :one
-- How many providers the account has connected (drives "is this the first key?").
SELECT count(*) FROM coach.api_key_config
WHERE account_id = $1;

-- name: UpsertApiKeyConfig :one
-- Store or replace the (account, provider) key with pre-sealed material, re-enabling it.
-- BOTH pairs are written in this ONE statement together with kek_id and ad_src_digest, so
-- no v1.7.0+ writer can leave them divergent: the AD pair always describes the legacy
-- pair sitting next to it. (A v1.6.0 writer during a rollback rewrites only the legacy
-- pair; that is detected afterwards by the digest, not prevented — see rewrap.go.)
--
-- is_default is NOT named: the column keeps its DEFAULT false for v1.6.0's benefit, and
-- the default itself is written to coach.key_default in the same transaction by the store.
-- The raw key never reaches this layer as a column.
INSERT INTO coach.api_key_config (
    account_id, provider, enc_key, enc_data_key, masked_key, default_model, name, enabled,
    enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest
)
VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, $9, $10, $11)
ON CONFLICT (account_id, provider) DO UPDATE
SET enc_key         = EXCLUDED.enc_key,
    enc_data_key    = EXCLUDED.enc_data_key,
    masked_key      = EXCLUDED.masked_key,
    default_model   = EXCLUDED.default_model,
    name            = EXCLUDED.name,
    enabled         = true,
    enc_key_ad      = EXCLUDED.enc_key_ad,
    enc_data_key_ad = EXCLUDED.enc_data_key_ad,
    kek_id          = EXCLUDED.kek_id,
    ad_src_digest   = EXCLUDED.ad_src_digest,
    updated_at      = now()
RETURNING id, account_id, provider, enc_key, enc_data_key, masked_key, default_model, name,
          enabled, created_at, updated_at, enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest;

-- name: UpdateApiKeyMeta :one
-- Update a provider's model + name WITHOUT touching either sealed pair (switch model /
-- rename). ErrNoRows when that provider isn't connected.
UPDATE coach.api_key_config
SET default_model = $3, name = $4, updated_at = now()
WHERE account_id = $1 AND provider = $2
RETURNING id, account_id, provider, enc_key, enc_data_key, masked_key, default_model, name,
          enabled, created_at, updated_at, enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest;

-- name: SetApiKeyEnabled :execrows
-- Flip one provider's enabled flag (Settings toggle / provider-AUTH failure — never a
-- quota or rate limit, which keep the key). Returns the affected row count so a no-op
-- can 404.
UPDATE coach.api_key_config
SET enabled = $3, updated_at = now()
WHERE account_id = $1 AND provider = $2;

-- name: DeleteApiKeyConfig :execrows
-- Remove one provider's key. Returns the affected row count so DELETE can 404 a no-op.
-- coach.key_default rows pointing at it go with it (ON DELETE CASCADE, migration 00004);
-- the store then promotes the earliest surviving key to the account's `coach` default.
DELETE FROM coach.api_key_config
WHERE account_id = $1 AND provider = $2;

-- --- background re-wrap (m1-10 task 3; internal/coach/rewrap.go) ---

-- name: TryLockRewrap :one
-- Take the re-wrap lock for the CURRENT TRANSACTION. False means another runner (another
-- replica, or a pass still in flight during a rolling update) holds it and this pass ends.
--
-- It must be the TRANSACTION-scoped lock, never pg_try_advisory_lock: on a pgxpool the
-- matching unlock can be issued on a different pooled connection, which leaks the lock
-- for the lifetime of the pod. COMMIT or ROLLBACK releases this one on the connection
-- that took it, which is what makes it safe across a rolling update.
SELECT pg_try_advisory_xact_lock(hashtext('coach.rewrap')) AS locked;

-- name: ListRewrapPending :many
-- One batch of key configs whose AD pair needs (re)sealing: never written, wrapped under
-- a KEK that is no longer active, or STALE — the legacy pair was rewritten underneath it
-- (a v1.6.0 key replace during a rollback), which the digest detects.
--
-- This WHERE is the exact COMPLEMENT of the currency rule (KeyConfig.ADPairCurrent), and
-- the `enc_data_key IS NOT NULL AND` guard is what makes it so. `sha256(NULL)` is NULL and
-- `<non-null digest> IS DISTINCT FROM NULL` is TRUE, so without the guard a row with NO
-- LEGACY PAIR — the state l-01 creates when it stops writing one — would read as pending
-- forever while ADPairCurrent() called it current. The job would re-seal and discard it on
-- every pass, and l-01's own precondition (`pending=0 skipped=0`) could never be met.
-- Not reachable today (enc_key/enc_data_key are still NOT NULL), but this is the half of
-- that forward-compat path that has to work when l-01 relaxes them.
--
-- Ordered by id for a stable, resumable scan. $2 is the batch size (≤ 50: the pass holds
-- one of coach's 4 pooled connections and must stay inside the pod's 128 Mi, ADR-0035 §5).
SELECT id, account_id, provider, enc_key, enc_data_key, enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest
FROM coach.api_key_config
WHERE enc_key_ad IS NULL
   OR kek_id IS DISTINCT FROM sqlc.arg(active_kek_id)::text
   OR (enc_data_key IS NOT NULL AND ad_src_digest IS DISTINCT FROM sha256(enc_data_key))
ORDER BY id
LIMIT sqlc.arg(batch_size);

-- name: CountRewrapPending :one
-- How many rows still need a re-wrap, for the pass summary log line (D34: a log line read
-- on demand, not an alert) and for l-01's `pending=0` precondition.
SELECT count(*) FROM coach.api_key_config
WHERE enc_key_ad IS NULL
   OR kek_id IS DISTINCT FROM sqlc.arg(active_kek_id)::text
   OR (enc_data_key IS NOT NULL AND ad_src_digest IS DISTINCT FROM sha256(enc_data_key));

-- name: UpdateApiKeyAD :execrows
-- Install a freshly sealed AD pair. OPTIMISTIC on the legacy pair: the WHERE pins
-- enc_data_key to the bytes the job decrypted from, so a PUT that replaced the key
-- mid-pass wins (0 rows affected) instead of being clobbered with a pair bound to the old
-- key. ad_src_digest is computed by PG from those same bytes, so the digest and the pair
-- are always written from one consistent observation.
--
-- IS NOT DISTINCT FROM, not `=`: with a NULL legacy pair (l-01 onwards) `NULL = NULL` is
-- NULL, so a plain `=` could never match such a row and every pass would re-seal it and
-- throw the result away. Against non-NULL bytes the two are identical, so this does not
-- weaken the race check — a racing PUT always writes a non-NULL enc_data_key, which still
-- fails to match the NULL the job observed.
UPDATE coach.api_key_config
SET enc_key_ad      = sqlc.arg(enc_key_ad),
    enc_data_key_ad = sqlc.arg(enc_data_key_ad),
    kek_id          = sqlc.arg(kek_id),
    ad_src_digest   = sha256(sqlc.arg(seen_enc_data_key)),
    updated_at      = now()
WHERE id = sqlc.arg(id) AND enc_data_key IS NOT DISTINCT FROM sqlc.arg(seen_enc_data_key);

-- name: Sha256OfLegacyPair :one
-- PG's sha256 of a row's legacy wrapped data key — the exact value the selector compares
-- ad_src_digest against. Exists so a test can assert Go's crypto/sha256 and PG's
-- sha256(bytea) agree on real stored bytes (they must, or the selector would see every
-- row as stale forever and the job would spin).
SELECT sha256(enc_data_key) AS digest FROM coach.api_key_config WHERE id = $1;
