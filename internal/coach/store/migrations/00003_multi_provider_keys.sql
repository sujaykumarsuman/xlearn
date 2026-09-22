-- Multi-provider coach keys (F006 round 2): move from ONE key per account to one key per
-- (account, provider), plus a per-account DEFAULT pointer — the provider whose model the
-- coach answers with. A learner can connect both Anthropic and OpenAI and flip the default
-- from Settings or the header quick-switch without re-pasting a key.
--
-- `is_default` is app-maintained as exactly one true row per account (the store sets it on
-- the first key, moves it on set-default, and promotes a survivor on delete). Schema `coach`
-- is provisioned out-of-band (see 00001).

-- +goose Up
-- +goose StatementBegin
-- Drop the single-key-per-account uniqueness; the key is now per (account, provider).
ALTER TABLE coach.api_key_config DROP CONSTRAINT IF EXISTS api_key_config_account_id_key;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS is_default boolean NOT NULL DEFAULT false;
-- +goose StatementEnd
-- +goose StatementBegin
-- Existing rows are one-per-account (old UNIQUE), so each becomes its account's default.
UPDATE coach.api_key_config SET is_default = true;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD CONSTRAINT api_key_config_account_provider_key UNIQUE (account_id, provider);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP CONSTRAINT IF EXISTS api_key_config_account_provider_key;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS is_default;
-- +goose StatementEnd
-- +goose StatementBegin
-- Best-effort restore of single-key uniqueness (fails if an account already has >1 key).
ALTER TABLE coach.api_key_config ADD CONSTRAINT api_key_config_account_id_key UNIQUE (account_id);
-- +goose StatementEnd
