-- Name your coach (F006): a display name for the single key config, so the account can
-- call it e.g. "Sonnet 5" or a custom label. Defaults to '' (the UI falls back to the
-- model's own name). Additive; schema `coach` is provisioned out-of-band (see 00001).

-- +goose Up
-- +goose StatementBegin
ALTER TABLE coach.api_key_config ADD COLUMN IF NOT EXISTS name text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE coach.api_key_config DROP COLUMN IF EXISTS name;
-- +goose StatementEnd
