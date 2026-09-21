-- review schema, S12 hardening: an index for the periodic due-sweep's scan.
--
-- The five-touch schedule reads (GET /revisions/due, ListActiveTouches) are already
-- covered by revision_item_due_idx (account_id, due_date, surfaced_at) — the
-- (account_id, due_date) prefix seeks the account and orders by due date. The one
-- un-served hot query is the ~15-min offline sweep (events.md Flow 4): it scans
-- ACROSS all accounts (no account_id predicate), so the account_id-leading index
-- can't seek it. A partial index on the exact predicate makes it an index-only walk
-- of the small "due, not surfaced, still pending" set instead of a filtered scan of
-- the whole table.

-- +goose Up

-- Serves SweepDueCandidates: WHERE due_date <= now() AND surfaced_at IS NULL AND
-- status = 'pending' ORDER BY due_date ASC. Partial (the predicate columns are
-- constant in the index) so it stays tiny — only rows awaiting their first surfacing.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS revision_item_sweep_idx
    ON review.revision_item (due_date)
    WHERE surfaced_at IS NULL AND status = 'pending';
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP INDEX IF EXISTS review.revision_item_sweep_idx;
-- +goose StatementEnd
