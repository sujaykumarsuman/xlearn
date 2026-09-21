-- practice schema: the per-user problem lifecycle (ADR-0004/0005) — problem state,
-- attempts, stage-gating audit, server-authoritative timers, outcomes, and the
-- transactional outbox. All DDL is schema-qualified so the xlearn_practice role
-- only ever touches schema `practice`.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `practice`
-- is provisioned by the CNPG Database CR (spec.schemas, owner xlearn_practice) so
-- the least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA practice AUTHORIZATION xlearn_practice;
--   ALTER ROLE xlearn_practice SET search_path = practice;
--
-- Cross-schema references (account_id → identity.account, problem_id →
-- curriculum.problem) are SOFT (bare ids, no FK) — FKs cannot cross ownership.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.user_problem_state (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      uuid        NOT NULL,
    problem_id      text        NOT NULL,
    status          text        NOT NULL DEFAULT 'available'
                                CHECK (status IN ('locked', 'available', 'attempting', 'solved')),
    first_solved_at timestamptz,
    current_touch   integer     NOT NULL DEFAULT 0,
    last_outcome    text        CHECK (last_outcome IN ('clean', 'rough', 'assisted', 'miss')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, problem_id)
);
-- +goose StatementEnd

-- One attempt run through the stage ladder. stage_reached is the deepest CONTENT
-- stage entered (attempt → hint → solution); the terminal "log" step is recorded
-- by ended_at + the outcome row, not by stage_reached. revealed_early latches when
-- the solution is revealed before the attempt timer elapses (R-PF2).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.attempt (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_problem_state_id uuid        NOT NULL REFERENCES practice.user_problem_state (id) ON DELETE CASCADE,
    stage_reached         text        NOT NULL DEFAULT 'attempt'
                                      CHECK (stage_reached IN ('attempt', 'hint', 'solution')),
    revealed_early        boolean     NOT NULL DEFAULT false,
    started_at            timestamptz NOT NULL DEFAULT now(),
    ended_at              timestamptz
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS attempt_ups_idx
    ON practice.attempt (user_problem_state_id);
-- +goose StatementEnd

-- Audit of stage gating: one row per stage entered. The set of distinct content
-- stages here is the authoritative "unlocked stages" for a problem (R-PF1) — the
-- gateway delivers only sections whose stage appears here.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.stage_event (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id    uuid        NOT NULL REFERENCES practice.attempt (id) ON DELETE CASCADE,
    stage         text        NOT NULL CHECK (stage IN ('attempt', 'hint', 'solution')),
    unlocked_from text        CHECK (unlocked_from IN ('attempt', 'hint', 'solution')),
    entered_at    timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS stage_event_attempt_idx
    ON practice.stage_event (attempt_id);
-- +goose StatementEnd

-- Server-authoritative countdown (R-PF3): the remaining time is always computed
-- from deadline_at, so a refresh/return resumes the same clock. attempt = 15 min,
-- hint = 10 min; solution/re-implement/log are untimed (no row).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.timer (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id  uuid        NOT NULL REFERENCES practice.attempt (id) ON DELETE CASCADE,
    kind        text        NOT NULL CHECK (kind IN ('attempt', 'hint')),
    deadline_at timestamptz NOT NULL,
    expired     boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS timer_attempt_idx
    ON practice.timer (attempt_id);
-- +goose StatementEnd

-- The logged outcome (R-OL1). Below-clean opens a mistake downstream (events.md
-- flow 2). revealed_early is copied from the attempt at log time.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.outcome (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id     uuid        NOT NULL REFERENCES practice.attempt (id) ON DELETE CASCADE,
    value          text        NOT NULL CHECK (value IN ('clean', 'rough', 'assisted', 'miss')),
    revealed_early boolean     NOT NULL DEFAULT false,
    logged_at      timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS outcome_attempt_idx
    ON practice.outcome (attempt_id);
-- +goose StatementEnd

-- Transactional outbox (ADR-0004): the domain change + the event fact are written
-- in one tx; a relay publishes unsent rows to NATS JetStream (stream
-- XLEARN_PRACTICE) and stamps sent_at.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice.outbox (
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
    ON practice.outbox (created_at)
    WHERE sent_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.outbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.outcome;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.timer;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.stage_event;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.attempt;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS practice.user_problem_state;
-- +goose StatementEnd
