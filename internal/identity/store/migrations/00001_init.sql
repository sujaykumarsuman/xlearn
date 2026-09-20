-- identity schema: accounts, OAuth links, server-side sessions, onboarding state
-- and the transactional outbox (ADR-0004/0005). All DDL is schema-qualified so the
-- xlearn_identity role only ever touches schema `identity`.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `identity`
-- is provisioned by the CNPG Database CR (spec.schemas, owner xlearn_identity) so
-- the least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA identity AUTHORIZATION xlearn_identity;
--   ALTER ROLE xlearn_identity SET search_path = identity;

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.account (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name      text        NOT NULL,
    email             text,
    timezone          text        NOT NULL DEFAULT 'UTC',
    study_budget_json jsonb       NOT NULL DEFAULT '{}'::jsonb,
    reminders_json    jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at        timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.oauth_identity (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id       uuid        NOT NULL REFERENCES identity.account (id) ON DELETE CASCADE,
    provider         text        NOT NULL CHECK (provider IN ('github', 'google')),
    provider_user_id text        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_user_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS oauth_identity_account_id_idx
    ON identity.oauth_identity (account_id);
-- +goose StatementEnd

-- Opaque, revocable server-side session behind the HttpOnly cookie. The id is a
-- high-entropy random token (not a uuid) minted by the service, so text not uuid.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.session (
    id         text        PRIMARY KEY,
    account_id uuid        NOT NULL REFERENCES identity.account (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS session_account_id_idx
    ON identity.session (account_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.onboarding (
    account_id   uuid        PRIMARY KEY REFERENCES identity.account (id) ON DELETE CASCADE,
    path_chosen  text,
    budget_set   boolean     NOT NULL DEFAULT false,
    key_added    boolean     NOT NULL DEFAULT false,
    completed_at timestamptz
);
-- +goose StatementEnd

-- Transactional outbox (ADR-0004): the account_created fact is written in the same
-- tx as the account row; a relay publishes unsent rows to NATS JetStream and stamps
-- sent_at. Columns mirror practice.outbox in data-model.md.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.outbox (
    event_id     uuid        PRIMARY KEY,
    subject      text        NOT NULL,
    payload_json jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    sent_at      timestamptz
);
-- +goose StatementEnd

-- Partial index over the relay's hot path: the oldest unsent rows.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS outbox_unsent_idx
    ON identity.outbox (created_at)
    WHERE sent_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.outbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.onboarding;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.session;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.oauth_identity;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.account;
-- +goose StatementEnd
