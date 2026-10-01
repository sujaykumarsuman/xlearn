-- name: CreateAttempt :one
-- account_id / path_slug / problem_id denormalise the owning problem state (m1-02,
-- M1a): v1.6.0 writes all three on insert.
INSERT INTO practice.attempt (user_problem_state_id, account_id, path_slug, problem_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOpenAttempt :one
-- The in-progress attempt for a problem state (not yet logged), newest first.
SELECT * FROM practice.attempt
WHERE user_problem_state_id = $1 AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1;

-- name: GetLatestAttempt :one
SELECT * FROM practice.attempt
WHERE user_problem_state_id = $1
ORDER BY started_at DESC
LIMIT 1;

-- name: SetAttemptStage :exec
UPDATE practice.attempt
SET stage_reached = $2
WHERE id = $1;

-- name: SetAttemptRevealedEarly :exec
UPDATE practice.attempt
SET revealed_early = true
WHERE id = $1;

-- name: EndAttempt :exec
UPDATE practice.attempt
SET ended_at = now()
WHERE id = $1;

-- --- m1-07 (M1b, D27): the coach-assist record and the open-attempt read ---

-- name: GetOpenAttemptForUpdate :one
-- The in-progress attempt, row-locked for LogOutcome: the conclusion and a concurrent
-- MarkCoachAssist serialise on it, so the Assisted clamp sees every assist recorded
-- before the conclusion, and an assist racing the conclusion finds the attempt closed.
SELECT * FROM practice.attempt
WHERE user_problem_state_id = $1 AND ended_at IS NULL
ORDER BY started_at DESC
LIMIT 1
FOR UPDATE;

-- name: MarkCoachAssist :one
-- D27: record the first coach chat about this problem during its open counted attempt.
-- Idempotent (COALESCE keeps the first time); no row when the attempt is unknown, not
-- the account's, or already concluded (the handler tells 404 from 409 afterwards).
UPDATE practice.attempt
SET coach_assist_at = COALESCE(coach_assist_at, now())
WHERE id = $1 AND account_id = $2 AND ended_at IS NULL
RETURNING coach_assist_at;

-- name: GetAttemptOwned :one
-- An attempt by id, scoped to its account (MarkCoachAssist's 404 vs 409).
SELECT id, ended_at FROM practice.attempt
WHERE id = $1 AND account_id = $2;

-- name: ListOpenAttempts :many
-- The account's open (not yet concluded) attempts, newest first, optionally for one
-- problem. Course attempts only until M2a (m2-01 adds attempt.purpose and reports open
-- touches in the same list).
SELECT id, problem_id, path_slug, started_at, stage_reached, coach_assist_at
FROM practice.attempt
WHERE account_id = sqlc.arg(account_id)
  AND ended_at IS NULL
  AND (sqlc.narg(problem_id)::text IS NULL OR problem_id = sqlc.narg(problem_id)::text)
ORDER BY started_at DESC;
