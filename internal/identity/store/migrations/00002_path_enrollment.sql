-- Per-user path enrollment (F002): "starting" a learning path is an explicit,
-- durable, per-account action (not the global seed `status:'active'`). started_at
-- anchors the learner's "current day" on that path; status leaves room for a later
-- pause/leave without dropping the start date. One row per (account, path).
--
-- Schema `identity` is provisioned out-of-band (CNPG in prod; a CREATE SCHEMA in
-- local/dev) — see 00001_init.sql. All DDL stays schema-qualified.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS identity.path_enrollment (
    account_id uuid        NOT NULL REFERENCES identity.account (id) ON DELETE CASCADE,
    path_slug  text        NOT NULL,
    status     text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused')),
    started_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, path_slug)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS identity.path_enrollment;
-- +goose StatementEnd
