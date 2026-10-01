-- +goose NO TRANSACTION
-- v2 M1b EXPAND (sprint m1-07; D27, ADR-0031 §7, t5 §9 "D18 capture"): an attempt
-- records when the learner first used the coach ON THAT PROBLEM while the attempt was
-- open. The gateway writes it (POST /attempts/{id}/assist, idempotent: COALESCE keeps
-- the first value) BEFORE it forwards the chat, so a failed write sends nothing (fails
-- closed). LogOutcome caps a self-reported clean/rough at assisted when the coach was
-- used on the attempt before its conclusion.
--
-- Additive only: one nullable column (a metadata-only change) and an index built
-- CONCURRENTLY (no write lock), hence outside a transaction (data-model.md, expand rules).
-- IF NOT EXISTS makes a re-run a no-op; the store test asserts the index is valid
-- (pg_index.indisvalid), since a failed CONCURRENTLY build would leave an INVALID one
-- that IF NOT EXISTS then skips. v1.6.0 (the rollback floor) names its attempt columns
-- explicitly (sqlc expands `*`), so it never reads the column. An EXPAND: no contract
-- marker.
--
-- The partial index serves GET /attempts/open (an account's open attempts, newest
-- first): open attempts are few per account, ended ones are the bulk of the table.

-- +goose Up
ALTER TABLE practice.attempt ADD COLUMN IF NOT EXISTS coach_assist_at timestamptz;

CREATE INDEX CONCURRENTLY IF NOT EXISTS attempt_open_by_account_idx
    ON practice.attempt (account_id, started_at DESC)
    WHERE ended_at IS NULL;

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
DROP INDEX CONCURRENTLY IF EXISTS practice.attempt_open_by_account_idx;
ALTER TABLE practice.attempt DROP COLUMN IF EXISTS coach_assist_at;
