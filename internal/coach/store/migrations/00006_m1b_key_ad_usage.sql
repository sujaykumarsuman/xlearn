-- v2 M1b EXPAND (sprint m1-10; ADR-0031 §7, t5 §9, ADR-0034 §3): a SECOND sealed pair
-- bound to the account and provider with AEAD associated data, under a named KEK from
-- the COACH_MASTER_KEYS keyring — plus per-message usage and cost.
--
-- Why a second pair rather than re-sealing in place: v1.6.0 opens the sealed material
-- with nil associated data, so a key re-sealed WITH ad would be undecryptable by it.
-- Re-sealing would silently raise coach's rollback floor to 1.7.0 while v1.7.0 declares
-- 1.6.0 (ADR-0034 §3). So this is an expand: the legacy pair (enc_key, enc_data_key)
-- stays, is dual-written by every v1.7.0+ writer, and is contracted in two later steps
-- (l-01 stops reading/writing it at v1.11.0; l-02 drops it at v1.12.0, floor v1.11.0).
--
--   * enc_key_ad / enc_data_key_ad — the AD-bound pair, sealed with associated data
--     `xlearn/coach/key/v1|<account_id>|<provider>` on BOTH AEAD layers.
--   * kek_id — which keyring entry wrapped enc_data_key_ad (k0 is today's
--     COACH_MASTER_KEY, so production needs no secret change this sprint).
--   * ad_src_digest — sha256 of the LEGACY enc_data_key the AD pair was sealed beside.
--     v1.6.0 stays a valid R-b target, and its UpsertApiKeyConfig rewrites only the
--     legacy pair, so without this digest a key replaced during an R-b would still be
--     answered from the STALE AD pair after rolling forward (a revoked old key would
--     401 and disable the key the owner just pasted; a live one would bill silently).
--     An AD pair is CURRENT ⇔ enc_key_ad IS NOT NULL AND (enc_data_key IS NULL OR
--     ad_src_digest = sha256(enc_data_key)). The re-wrap job repairs a stale row.
--
--   * coach_message gains the provider/model and token usage of the turn that produced
--     it, plus est_cost_micros (catalog price × tokens; NULL for a custom model id or
--     when the stream reported no usage) and the raw stop_reason. Display-only; it
--     feeds "This month on your keys" (AB01 F13).
--
-- Additive only, every column nullable, no backfill: the re-wrap goroutine fills the AD
-- columns in batches after startup, and v1.6.0 never reads any of them. This is an
-- EXPAND, so it deliberately carries no contract marker and no rollback floor — only the
-- later drop migrations (l-01, l-02) declare one, which hack/lint-migrations.sh enforces
-- in both directions.

-- +goose Up

-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS enc_key_ad bytea;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS enc_data_key_ad bytea;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS kek_id text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS ad_src_digest bytea;
-- +goose StatementEnd

-- The re-wrap selector scans for rows needing work (AD pair missing, wrapped under a
-- retired KEK, or stale against the legacy pair). A partial index on the cheap arm keeps
-- the steady state — every row current — off a sequential scan.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS api_key_config_rewrap_pending_idx
    ON coach.api_key_config (id)
    WHERE enc_key_ad IS NULL;
-- +goose StatementEnd

-- Usage of the provider turn that produced an assistant message (task 4). All nullable:
-- v1 rows and any turn whose stream reported no usage keep NULL.
-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS provider text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS model text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS input_tokens integer;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS output_tokens integer;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS est_cost_micros bigint;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE coach.coach_message ADD COLUMN IF NOT EXISTS stop_reason text;
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS stop_reason;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS est_cost_micros;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS output_tokens;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS input_tokens;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS model;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.coach_message DROP COLUMN IF EXISTS provider;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS coach.api_key_config_rewrap_pending_idx;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS ad_src_digest;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS kek_id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS enc_data_key_ad;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS enc_key_ad;
-- +goose StatementEnd
