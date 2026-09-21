-- review schema, S07: the mistake journal, the weekly weak-area rollup and the
-- in-app notifications reminder table (ADR-0005/0015/0016). All DDL is
-- schema-qualified so the xlearn_review role only ever touches schema `review`.
--
-- Cross-schema references stay SOFT (bare ids, no FK): account_id → identity.account,
-- problem_id → curriculum.problem. The problem `pattern` and the account
-- `timezone`/`study_budget` are resolved via internal APIs, never a cross-schema read.

-- +goose Up

-- The mistake journal (R-MJ1/R-MJ2/R-MJ4): one row per opened mistake. An entry is
-- opened on a below-Clean outcome or a failed re-solve (pattern pre-filled from
-- curriculum), closed after two clean revisits, and re-opened on a later fail.
-- `category` is the 8-value enum (R-MJ2), NULL until the learner picks one from the
-- seeded picker (an auto-opened entry starts uncategorised). `revisit_count` counts
-- CLEAN revisits only; at 2 the entry closes. `revisit_date` mirrors the problem's
-- soonest pending revision touch (a soft, denormalised convenience for the journal
-- "Revisit" column). status is open|closed.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.mistake_entry (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid        NOT NULL,
    problem_id    text        NOT NULL,
    pattern       text        NOT NULL DEFAULT '',
    mistake       text        NOT NULL DEFAULT '',
    root_cause    text        NOT NULL DEFAULT '',
    insight       text        NOT NULL DEFAULT '',
    category      text        CHECK (category IN (
                      'misread', 'wrong_pattern', 'right_pattern_wrong_state', 'off_by_one',
                      'language_bug', 'complexity_misjudged', 'communication', 'time_management')),
    revisit_date  timestamptz,
    status        text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    revisit_count integer     NOT NULL DEFAULT 0 CHECK (revisit_count >= 0),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- At most one OPEN entry per (account, problem): the idempotency key for below-clean
-- opens (a repeat miss on an already-open problem is a no-op) and the load-bearing
-- invariant for the close/re-open state machine (the clean-revisit increment targets
-- "the open entry", so there must be exactly one). A closed entry does not conflict,
-- so a problem can recur (a new open row) after an earlier one was resolved.
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS mistake_entry_one_open_idx
    ON review.mistake_entry (account_id, problem_id)
    WHERE status = 'open';
-- +goose StatementEnd

-- The journal list scan: an account's entries, optionally filtered by status,
-- newest-first (GET /mistakes).
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS mistake_entry_account_idx
    ON review.mistake_entry (account_id, status, created_at DESC);
-- +goose StatementEnd

-- The weekly weak-area rollup (R-MJ3): one snapshot per (account, week_of). week_of
-- is the Monday of the week in the ACCOUNT timezone (computed by the recompute job,
-- ADR-0016), so snapshots don't straddle the wrong day for users far from UTC.
-- top_category is the max per-category count of this week's open entries (NULL when
-- there are no categorised open entries this week); counts_json is the full
-- {category: count} map. UNIQUE (account_id, week_of) makes the weekly recompute
-- idempotent — re-running the tick upserts the same row.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.weak_area_snapshot (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   uuid        NOT NULL,
    week_of      date        NOT NULL,
    top_category text,
    counts_json  jsonb       NOT NULL DEFAULT '{}'::jsonb,
    computed_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, week_of)
);
-- +goose StatementEnd

-- The read path picks an account's latest snapshot (most recent week_of first).
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS weak_area_snapshot_account_idx
    ON review.weak_area_snapshot (account_id, week_of DESC);
-- +goose StatementEnd

-- In-app notification reminders (v1 single channel). The notifications worker
-- (internal/review/notifications, kept cleanly separable per ADR-0003/0016) writes
-- one row per revision_due event, scheduled inside the account study-budget window.
-- delivered_at latches when the reminder is surfaced/acknowledged (unused in v1's
-- read-only Dashboard surface, reserved for a later channel).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS review.reminder (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   uuid        NOT NULL,
    kind         text        NOT NULL,
    due_at       timestamptz NOT NULL,
    delivered_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- The Dashboard "revisions due" surface reads an account's pending reminders,
-- soonest-due first.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS reminder_account_idx
    ON review.reminder (account_id, due_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS review.reminder;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS review.weak_area_snapshot;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS review.mistake_entry;
-- +goose StatementEnd
