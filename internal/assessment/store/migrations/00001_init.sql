-- assessment schema: the timed mock-interview aggregate (ADR-0004/0005) — mock
-- sessions with a server-authoritative 45-minute timer + phase rail, the 7-dimension
-- rubric (/35), the transactional outbox and the idempotent inbox. All DDL is
-- schema-qualified so the xlearn_assessment role only ever touches schema `assessment`.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `assessment`
-- is provisioned by the CNPG Database CR (spec.schemas, owner xlearn_assessment) so
-- the least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA assessment AUTHORIZATION xlearn_assessment;
--   ALTER ROLE xlearn_assessment SET search_path = assessment;
--
-- Cross-schema references (account_id -> identity.account, problem_id ->
-- curriculum.problem) are SOFT (bare ids, no FK) — FKs cannot cross ownership.
--
-- The S09 progress-projection tables (proj_coverage, proj_heatmap, proj_mastery,
-- proj_outcome_mix) are DELIBERATELY not created here: they land as an additive later
-- migration (00002) so this sprint owns only the mock aggregate + the consume seam.

-- +goose Up

-- One timed mock-interview session (R-MK1). Lifecycle: created status='live' with a
-- server clock (started_at=now, deadline_at=now+45m); the phase rail + remaining time
-- are computed server-side from started_at (never a client clock). Scoring latches
-- status='scored' + total_35 (R-MK2). date is the calendar day the mock was taken
-- (trend x-axis). set_id / problem_id / difficulty are the setup selection; problem_id
-- is a soft curriculum ref the gateway enriches (may be '' for a mixed set).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.mock_session (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid        NOT NULL,
    set_id      text        NOT NULL,
    problem_id  text        NOT NULL DEFAULT '',
    difficulty  text        NOT NULL DEFAULT 'med'
                            CHECK (difficulty IN ('easy', 'med', 'hard')),
    date        date        NOT NULL DEFAULT current_date,
    status      text        NOT NULL DEFAULT 'live'
                            CHECK (status IN ('live', 'scored')),
    total_35    integer     CHECK (total_35 IS NULL OR total_35 BETWEEN 7 AND 35),
    notes       text        NOT NULL DEFAULT '',
    started_at  timestamptz NOT NULL DEFAULT now(),
    deadline_at timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- A scored session must carry a total; a live one must not (integrity guard).
    CHECK ((status = 'scored') = (total_35 IS NOT NULL))
);
-- +goose StatementEnd

-- The trend query reads an account's scored sessions oldest-first; this index keeps
-- that scan cheap and ordered.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS mock_session_account_idx
    ON assessment.mock_session (account_id, started_at);
-- +goose StatementEnd

-- One rubric score per dimension per session (R-MK2): 7 dimensions x 1..5 = /35. The
-- dimension is a fixed 7-value enum (stored as text + CHECK, not free text). The
-- UNIQUE (mock_session_id, dimension) makes a re-submit a no-op at the row level and
-- is the integrity backstop for "exactly one score per dimension".
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.rubric_score (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    mock_session_id uuid        NOT NULL REFERENCES assessment.mock_session (id) ON DELETE CASCADE,
    dimension       text        NOT NULL CHECK (dimension IN (
                        'communication', 'problem_understanding', 'brute_force',
                        'optimisation', 'code_quality', 'edge_cases', 'complexity')),
    score           integer     NOT NULL CHECK (score BETWEEN 1 AND 5),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (mock_session_id, dimension)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS rubric_score_session_idx
    ON assessment.rubric_score (mock_session_id);
-- +goose StatementEnd

-- Transactional outbox (ADR-0004): the mock scoring + the mock_completed event fact
-- are written in one tx; the relay publishes unsent rows to NATS JetStream (stream
-- XLEARN_ASSESSMENT) and stamps sent_at. Same shape as every producing schema's outbox.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.outbox (
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
    ON assessment.outbox (created_at)
    WHERE sent_at IS NULL;
-- +goose StatementEnd

-- Idempotent inbox (ADR-0004): the event_id of every CONSUMED practice/review event is
-- recorded here in the same tx as its side effects; a re-delivered event conflicts on
-- the PK and is a no-op -> effectively-once processing. The projection HANDLER BODIES
-- that update the S09 read-model tables are no-op stubs this sprint (S08 wires only the
-- durable consumers + this dedupe seam); S09 fills the projections into the same tx.
-- event_id is text so any envelope id (or the Nats-Msg-Id fallback) stores without a parse.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.inbox (
    event_id    text        PRIMARY KEY,
    consumed_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.inbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.outbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.rubric_score;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.mock_session;
-- +goose StatementEnd
