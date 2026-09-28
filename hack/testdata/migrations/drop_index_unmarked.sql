-- lint-expect: 1
-- EVERY DROP INDEX is contract (a grep can't tell whether the index was unique), even a
-- CONCURRENTLY one in a NO TRANSACTION file, until a reviewer marks it xlearn:relax.

-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS demo.thing_account_week_idx;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS thing_account_week_idx ON demo.thing (account_id, week_of);
