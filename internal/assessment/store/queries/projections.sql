-- S09 progress-projection read model (ADR-0017/0018). Every write is an UPSERT with
-- additive or GREATEST(...) accumulation so out-of-order delivery is safe and a
-- drop-and-replay rebuild converges to an identical result (a pure function of the
-- event log). Each runs inside the SAME transaction as the inbox claim (00001), so
-- inbox <-> projected stays atomic (effectively-once).

-- name: UpsertCoverageSolve :exec
-- problem_solved -> mark the problem solved and keep the EARLIEST solve time. solved
-- latches true on any solve (out-of-order safe); level1_schedules is left untouched.
INSERT INTO assessment.proj_coverage (account_id, problem_id, solved, first_solved_at, updated_at)
VALUES ($1, $2, true, $3, now())
ON CONFLICT (account_id, problem_id) DO UPDATE SET
    solved          = true,
    first_solved_at = LEAST(
        COALESCE(assessment.proj_coverage.first_solved_at, EXCLUDED.first_solved_at),
        EXCLUDED.first_solved_at),
    updated_at      = now();

-- name: UpsertCoverageLevel1 :exec
-- revision_scheduled(touch_level=1) -> count Day-1 ladder anchors. The first is the
-- initial ladder; each subsequent one is a reset (a failed re-solve reset the ladder).
-- Inserts a not-yet-solved row if the schedule outran its problem_solved (out of order).
INSERT INTO assessment.proj_coverage (account_id, problem_id, level1_schedules, updated_at)
VALUES ($1, $2, 1, now())
ON CONFLICT (account_id, problem_id) DO UPDATE SET
    level1_schedules = assessment.proj_coverage.level1_schedules + 1,
    updated_at       = now();

-- name: UpsertMastery :exec
-- problem_solved -> accumulate per-problem solve quality. best_rank is the best outcome
-- ever (clean=4 > rough=3 > assisted=2 > miss=1), taken by GREATEST so it is a
-- commutative max under replay; best_outcome tracks the label of that best rank.
INSERT INTO assessment.proj_mastery (
    account_id, problem_id, best_outcome, best_rank, clean_solves, solve_count, updated_at)
VALUES ($1, $2, $3, $4, $5, 1, now())
ON CONFLICT (account_id, problem_id) DO UPDATE SET
    best_rank    = GREATEST(assessment.proj_mastery.best_rank, EXCLUDED.best_rank),
    best_outcome = CASE
        WHEN EXCLUDED.best_rank > assessment.proj_mastery.best_rank
            THEN EXCLUDED.best_outcome
        ELSE assessment.proj_mastery.best_outcome END,
    clean_solves = assessment.proj_mastery.clean_solves + EXCLUDED.clean_solves,
    solve_count  = assessment.proj_mastery.solve_count + 1,
    updated_at   = now();

-- name: UpsertHeatmapSolve :exec
-- problem_solved -> +1 solve on the event's UTC day.
INSERT INTO assessment.proj_heatmap (account_id, activity_date, solves, updated_at)
VALUES ($1, $2, 1, now())
ON CONFLICT (account_id, activity_date) DO UPDATE SET
    solves     = assessment.proj_heatmap.solves + 1,
    updated_at = now();

-- name: UpsertHeatmapReview :exec
-- revision_scheduled -> +1 review on the event's UTC day (each advance / reset /
-- initial-ladder schedule is same-day review work).
INSERT INTO assessment.proj_heatmap (account_id, activity_date, reviews, updated_at)
VALUES ($1, $2, 1, now())
ON CONFLICT (account_id, activity_date) DO UPDATE SET
    reviews    = assessment.proj_heatmap.reviews + 1,
    updated_at = now();

-- name: UpsertOutcomeMix :exec
-- problem_solved(first_solve) -> +1 for the first-solve outcome (Clean/Rough/Assisted/Miss).
INSERT INTO assessment.proj_outcome_mix (account_id, outcome, cnt, updated_at)
VALUES ($1, $2, 1, now())
ON CONFLICT (account_id, outcome) DO UPDATE SET
    cnt        = assessment.proj_outcome_mix.cnt + 1,
    updated_at = now();

-- name: CountSolvedProblems :one
-- The "solved / 151" numerator: distinct problems the account has solved.
SELECT COUNT(*) FROM assessment.proj_coverage
WHERE account_id = $1 AND solved;

-- name: RetentionStats :one
-- Day-7 retention inputs: `ladders` is problems that started a spaced-repetition ladder
-- (a Day-1 anchor), `resets` is how many times a ladder was reset by a failed re-solve.
SELECT
    COALESCE(SUM(GREATEST(level1_schedules - 1, 0)), 0)::bigint AS resets,
    COUNT(*) FILTER (WHERE level1_schedules > 0)               AS ladders
FROM assessment.proj_coverage
WHERE account_id = $1;

-- name: ListHeatmap :many
-- The per-day revision-activity rows on/after `since` (the heatmap window + the streak).
SELECT activity_date, solves, reviews
FROM assessment.proj_heatmap
WHERE account_id = $1 AND activity_date >= $2
ORDER BY activity_date;

-- name: ListSolvedMastery :many
-- Every solved problem with its solve quality — the gateway groups these by curriculum
-- pattern (mastery bars) and by week -> phase (completion table). Coverage is the solved
-- authority; mastery supplies the quality (absent -> zeros for a not-yet-scored row).
SELECT
    c.problem_id,
    c.first_solved_at,
    COALESCE(m.best_outcome, '')  AS best_outcome,
    COALESCE(m.best_rank, 0)::int AS best_rank,
    COALESCE(m.clean_solves, 0)   AS clean_solves,
    COALESCE(m.solve_count, 0)    AS solve_count
FROM assessment.proj_coverage c
LEFT JOIN assessment.proj_mastery m
    ON m.account_id = c.account_id AND m.problem_id = c.problem_id
WHERE c.account_id = $1 AND c.solved
ORDER BY c.problem_id;

-- name: ListOutcomeMix :many
-- The first-solve outcome mix (Clean/Rough/Assisted/Miss counts).
SELECT outcome, cnt FROM assessment.proj_outcome_mix
WHERE account_id = $1;

-- name: MockAggregate :one
-- Scored-mock roll-up for the Progress + Dashboard tiles: how many, the average /35, and
-- the best /35. `last`/`delta` come from the ordered trend in Go.
SELECT
    COUNT(*)                               AS scored_count,
    COALESCE(ROUND(AVG(total_35)), 0)::int AS average_35,
    COALESCE(MAX(total_35), 0)::int        AS best_35
FROM assessment.mock_session
WHERE account_id = $1 AND status = 'scored';
