-- v2 M1b EXPAND (sprint m1-07 task 3; L18 in ADR-0035 §4, t5 §9 "Usage display and
-- limits"): the durable half of the BYO coach caps — at most 300 coach messages per
-- account per UTC day. It bounds what a stolen xLearn session can spend from a learner's
-- own provider key.
--
-- One row per (account, UTC day); n is the number of chat turns admitted that day. The
-- coach claims a message with ONE atomic conditional upsert (queries/message_quota_day.sql)
--
--   INSERT … VALUES (account, day, 1)
--   ON CONFLICT (account_id, day) DO UPDATE SET n = n + 1 WHERE n < 300 RETURNING n
--
-- and no row back means the cap is reached. Concurrent requests serialize on the row lock,
-- so they cannot overshoot the cap (internal/coach/store's race test pins this).
--
-- `day` is the UTC date computed in Go and passed as a parameter, never the database's
-- current_date (one clock, no session TimeZone dependency). The in-process halves of L18
-- (2 concurrent streams, 20 messages a minute) hold no database state.
--
-- Per-user data, erased by account_id (data-model.md; l-01's coach erase list must name
-- this table). Old days are never read again; nothing prunes them yet (one small row per
-- account per active day).
--
-- A new table only: v1.6.0 (the R-b floor) never touches it. This is an EXPAND, so it
-- deliberately carries no contract marker and no rollback floor.

-- +goose Up

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS coach.message_quota_day (
    account_id uuid NOT NULL,
    day        date NOT NULL,
    n          int  NOT NULL,
    PRIMARY KEY (account_id, day)
);
-- +goose StatementEnd

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
-- +goose StatementBegin
DROP TABLE IF EXISTS coach.message_quota_day;
-- +goose StatementEnd
