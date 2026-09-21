-- assessment schema: the S09 progress-projection read-model tables (ADR-0004/0005,
-- ADR-0017 scaffold, ADR-0018 grain). These are the durable read models the
-- projection consumers (durable pull consumers on XLEARN_PRACTICE / XLEARN_REVIEW)
-- upsert from practice/review events, so the Progress + Dashboard reads never touch
-- another schema. Every projection is a PURE FUNCTION OF THE EVENT LOG: handlers do no
-- external reads and no wall-clock state branching, and every mutation is an UPSERT
-- with commutative (additive / max) accumulation, so out-of-order delivery is safe and
-- a drop-and-replay from the stream start rebuilds an identical result.
--
-- GRAIN (ADR-0018): the practice/review events carry only a bare problem_id (never a
-- pattern or week_n — those are curriculum facts). So the projections are keyed at the
-- grain the events actually supply — per (account, problem), per (account, date), per
-- (account, outcome) — and the gateway composes the by-week / by-phase / by-pattern
-- roll-ups from curriculum. Enriching at projection time would need a cross-schema read
-- and would break replay determinism, so it is deliberately not done here.
--
-- All DDL is schema-qualified so the xlearn_assessment role only ever touches schema
-- `assessment`. The idempotency inbox (00001) claims each event_id in the SAME
-- transaction as these upserts, keeping inbox <-> projected atomic.

-- +goose Up

-- Per-(account, problem) coverage + spaced-repetition-ladder state. `solved` latches
-- true on the first problem_solved (any outcome) and is the source of the "solved /
-- 151" tile and the by-phase completion (mapped problem -> week -> phase in the
-- gateway). `level1_schedules` counts Day-1 (touch_level=1) revision_scheduled events:
-- the FIRST is the ladder's initial anchor, so a later Day-1 re-schedule means a reset
-- (a failed re-solve reset the ladder). resets = GREATEST(level1_schedules - 1, 0),
-- computed at read time, which keeps the count order-independent under replay.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.proj_coverage (
    account_id       uuid        NOT NULL,
    problem_id       text        NOT NULL,
    solved           boolean     NOT NULL DEFAULT false,
    first_solved_at  timestamptz,
    level1_schedules integer     NOT NULL DEFAULT 0,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, problem_id)
);
-- +goose StatementEnd

-- Per-(account, problem) solve quality — the pattern-mastery source. best_rank is the
-- best outcome ever achieved for the problem (clean=4 > rough=3 > assisted=2 > miss=1),
-- accumulated by GREATEST so it is a commutative max under replay; best_outcome is its
-- text label. clean_solves / solve_count let the gateway weight a pattern's mastery.
-- The gateway groups these rows by curriculum pattern to draw the mastery bars.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.proj_mastery (
    account_id   uuid        NOT NULL,
    problem_id   text        NOT NULL,
    best_outcome text        NOT NULL DEFAULT '',
    best_rank    smallint    NOT NULL DEFAULT 0,
    clean_solves integer     NOT NULL DEFAULT 0,
    solve_count  integer     NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, problem_id)
);
-- +goose StatementEnd

-- Per-(account, day) revision-activity heatmap: solves counts problem_solved by the
-- event's occurred_at (UTC) day; reviews counts revision_scheduled the same way (each
-- advance / reset / initial-ladder schedule is same-day review work). Both are additive
-- counters (commutative under replay). The current/longest streak is derived at read
-- time from these rows. Dates are UTC for determinism (a per-account-timezone heatmap
-- is a later refinement).
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.proj_heatmap (
    account_id    uuid        NOT NULL,
    activity_date date        NOT NULL,
    solves        integer     NOT NULL DEFAULT 0,
    reviews       integer     NOT NULL DEFAULT 0,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, activity_date)
);
-- +goose StatementEnd

-- Per-(account, outcome) first-solve outcome mix (Clean / Rough / Assisted / Miss). cnt
-- increments once per problem's FIRST solve (problem_solved.first_solve = true), so the
-- totals sum to the solved count — the "N first-solves" the outcome-mix bar shows.
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS assessment.proj_outcome_mix (
    account_id uuid        NOT NULL,
    outcome    text        NOT NULL
                           CHECK (outcome IN ('clean', 'rough', 'assisted', 'miss')),
    cnt        integer     NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, outcome)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.proj_outcome_mix;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.proj_heatmap;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.proj_mastery;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS assessment.proj_coverage;
-- +goose StatementEnd
