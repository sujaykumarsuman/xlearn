-- coach schema: the AI-coach state (ADR-0007). It holds each user's provider API key
-- ENVELOPE-ENCRYPTED at rest (never plaintext), plus the per-page-context chat threads
-- and their messages. All DDL is schema-qualified so the xlearn_coach role only ever
-- touches schema `coach`.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `coach` is
-- provisioned by the CNPG Database CR (spec.schemas, owner xlearn_coach) so the
-- least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA coach AUTHORIZATION xlearn_coach;
--   ALTER ROLE xlearn_coach SET search_path = coach;
--
-- Cross-schema references (account_id -> identity.account) are SOFT (bare ids, no FK) —
-- FKs cannot cross ownership.
--
-- coach EMITS NO EVENTS (no outbox) and CONSUMES NONE via NATS (page context arrives on
-- the request from the gateway) — so this schema has no outbox/inbox tables.

-- +goose Up

-- One provider-API-key configuration per account (ADR-0007). v1 is single-key: a user
-- has at most one active provider key, so account_id is UNIQUE and PUT /coach/key
-- upserts it. The raw provider key is NEVER stored — instead:
--   enc_key      = the raw key sealed with a per-record 32-byte data key
--                  (XChaCha20-Poly1305, nonce-prefixed ciphertext),
--   enc_data_key = that data key wrapped by the service master key (same AEAD),
--   masked_key   = a display-only redaction (e.g. sk-...3f2a) — the only value that
--                  ever leaves the service.
-- The master key lives outside the DB (SOPS-managed secret), so a DB/backup leak alone
-- cannot decrypt the keys. enabled flips to false when the key fails provider auth
-- (the client then routes the user back to Settings).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS coach.api_key_config (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid        NOT NULL UNIQUE,
    provider      text        NOT NULL CHECK (provider IN ('openai', 'anthropic')),
    enc_key       bytea       NOT NULL,
    enc_data_key  bytea       NOT NULL,
    masked_key    text        NOT NULL,
    default_model text        NOT NULL DEFAULT '',
    enabled       boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- One chat thread per (account, page context). page_context is a stable key the client
-- sends (e.g. "problem:pb-016", "concept:sliding-window", "dashboard") so the coach
-- panel resumes the right conversation per screen. GET /coach/thread?context= reads it.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS coach.coach_thread (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   uuid        NOT NULL,
    page_context text        NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, page_context)
);
-- +goose StatementEnd

-- One chat message in a thread. role is user|assistant (the system prompt is built
-- server-side per request and never persisted). seq is a monotonic per-schema identity
-- so history has a stable total order even when two messages share a created_at
-- millisecond (a bare created_at sort is ambiguous for the user->assistant pair).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS coach.coach_message (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id  uuid        NOT NULL REFERENCES coach.coach_thread (id) ON DELETE CASCADE,
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    role       text        NOT NULL CHECK (role IN ('user', 'assistant')),
    content    text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- History reads scan a thread oldest-first; this index keeps that ordered scan cheap.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS coach_message_thread_idx
    ON coach.coach_message (thread_id, seq);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS coach.coach_message;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS coach.coach_thread;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS coach.api_key_config;
-- +goose StatementEnd
