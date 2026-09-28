-- +goose NO TRANSACTION
-- v2 M1a EXPAND, part 2 (sprint m1-02; ADR-0034 §3): the per-course weak-area unique,
-- built CONCURRENTLY (no write lock on the table) and therefore outside a transaction.
-- IF NOT EXISTS makes a re-run a no-op; a failed CONCURRENTLY build leaves an INVALID
-- index that IF NOT EXISTS would then skip, so the store test asserts every review
-- index is valid (pg_index.indisvalid).
--
-- The v1 UNIQUE (account_id, week_of) stays: v1.5.2's upsert targets it. It would block
-- a second course's snapshot in the same week, so it joins m1-08's M1c drop list and
-- must be gone before a second course writes snapshots.

-- +goose Up
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS weak_area_snapshot_account_path_week_uq
    ON review.weak_area_snapshot (account_id, path_slug, week_of);

-- +goose Down
-- Never run in production (ADR-0034 §3: roll forward). Local/dev only.
DROP INDEX CONCURRENTLY IF EXISTS review.weak_area_snapshot_account_path_week_uq;
