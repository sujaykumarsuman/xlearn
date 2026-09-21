-- review schema: the five-touch spaced-repetition scheduler (ADR-0004/0005) — the
-- revision queue, auto-score results, the transactional outbox and the idempotent
-- inbox. All DDL is schema-qualified so the xlearn_review role only ever touches
-- schema `review`.
--
-- Schema OWNERSHIP is a platform concern, not this migration's: schema `review` is
-- provisioned by the CNPG Database CR (spec.schemas, owner xlearn_review) so the
-- least-privilege role can create tables in it without CREATE on the database.
-- For local/dev, create it once out-of-band (mirrors prod):
--   CREATE SCHEMA review AUTHORIZATION xlearn_review;
--   ALTER ROLE xlearn_review SET search_path = review;
--
-- Cross-schema references (account_id → identity.account, problem_id →
-- curriculum.problem) are SOFT (bare ids, no FK) — FKs cannot cross ownership.
-- (The mistake_entry, weak_area_snapshot, and reminder tables land in S07.)

-- +goose Up

-- The five-touch queue (R-SR1): one row per (account, problem, touch). touch_level
-- 1..5 = Day 1/3/7/21/45. status: pending (awaiting re-solve), passed (this touch
-- cleared), failed (this touch missed — a fail resets the whole problem to Day 1,
-- re-anchoring the ladder). surfaced_at latches when the periodic sweep materialises
-- the touch as "due today" (idempotent — the guard against re-emitting revision_due).
-- The UNIQUE (account_id, problem_id, touch_level) is the idempotency key: re-delivered
-- or out-of-order practice events upsert the same touch instead of double-scheduling.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.revision_item (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid        NOT NULL,
    problem_id  text        NOT NULL,
    touch_level integer     NOT NULL CHECK (touch_level BETWEEN 1 AND 5),
    due_date    timestamptz NOT NULL,
    surfaced_at timestamptz,
    status      text        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'passed', 'failed')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, problem_id, touch_level)
);
-- +goose StatementEnd

-- The sweep scans WHERE due_date <= now() AND surfaced_at IS NULL, per account; this
-- composite index keeps that ~15-min scan cheap on the single node (R-SR6).
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS revision_item_due_idx
    ON review.revision_item (account_id, due_date, surfaced_at);
-- +goose StatementEnd

-- The auto-score record for one re-solve (R-SR2). auto_pass is derived server-side:
-- named_pattern_secs < 120 AND solved_in_timer AND stated_complexity. mock_mode marks
-- the stricter Day 21 / Day 45 touches (R-SR4). It is the durable audit of a touch's
-- outcome (a fail re-anchors revision_item, so the failure lives here, not only there).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.touch_result (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    revision_item_id  uuid        NOT NULL REFERENCES review.revision_item (id) ON DELETE CASCADE,
    named_pattern_secs integer    NOT NULL,
    solved_in_timer   boolean     NOT NULL,
    stated_complexity boolean     NOT NULL,
    auto_pass         boolean     NOT NULL,
    mock_mode         boolean     NOT NULL DEFAULT false,
    scored_at         timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS touch_result_item_idx
    ON review.touch_result (revision_item_id);
-- +goose StatementEnd

-- Transactional outbox (ADR-0004): the domain change + the event fact are written in
-- one tx; the relay publishes unsent rows to NATS JetStream (stream XLEARN_REVIEW) and
-- stamps sent_at. Same shape as every producing schema's outbox.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.outbox (
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
    ON review.outbox (created_at)
    WHERE sent_at IS NULL;
-- +goose StatementEnd

-- Idempotent inbox (ADR-0004): the event_id of every CONSUMED practice event is
-- recorded here in the same tx as its side effects; a re-delivered event conflicts on
-- the PK and is a no-op → effectively-once processing. event_id is text so any
-- envelope id (or the Nats-Msg-Id fallback) can be stored without a parse.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.inbox (
    event_id    text        PRIMARY KEY,
    consumed_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS review.inbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS review.outbox;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS review.touch_result;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS review.revision_item;
-- +goose StatementEnd
