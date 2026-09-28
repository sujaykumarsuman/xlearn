-- assessment schema, mi-05 (N0, ADR-0035 §1.2): the dead-letter table for assessment's
-- two durable consumers (XLEARN_PRACTICE/assessment, XLEARN_REVIEW/assessment).
--
-- When a handler fails on the LAST delivery (NumDelivered >= MaxDeliver, ~8 h of
-- backoff), the consumer records one row here, terminates the message and logs ERROR —
-- replacing v1's silent skip. Rows are durable (pod logs don't survive a rollout) and
-- read on demand (D34: no alerting). IDs ONLY — no payload, no error text, no account
-- id — so the table is erase-safe. stream_seq (beyond the ADR's column list) locates
-- the raw message on the stream for break-glass inspection or replay.
--
-- Expand-only (a new table); INSERT ... ON CONFLICT (durable, event_id) DO NOTHING
-- keeps a re-dead-lettered event (a replay) to one row. Both assessment durables are
-- named `assessment`; event ids are globally unique, so (durable, event_id) still
-- identifies one event (subject says which stream).

-- +goose Up

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.event_dead_letter (
    event_id   text        NOT NULL,
    subject    text        NOT NULL,
    durable    text        NOT NULL,
    err_class  text        NOT NULL,
    stream_seq bigint,
    at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (durable, event_id)
);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.event_dead_letter;
-- +goose StatementEnd
