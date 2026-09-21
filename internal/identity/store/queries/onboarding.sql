-- name: CreateOnboarding :one
INSERT INTO identity.onboarding (account_id)
VALUES ($1)
RETURNING *;

-- name: GetOnboarding :one
SELECT * FROM identity.onboarding
WHERE account_id = $1;

-- name: SetOnboardingPath :one
UPDATE identity.onboarding
SET path_chosen = $2
WHERE account_id = $1
RETURNING *;

-- name: SetOnboardingBudgetSet :one
-- Onboarding step 2 flag. The study budget itself is written to account.study_budget_json
-- (via UpdateAccount) in the SAME transaction as this flag.
UPDATE identity.onboarding
SET budget_set = true
WHERE account_id = $1
RETURNING *;

-- name: CompleteOnboarding :one
-- Onboarding step 3 (Finish / Skip). Idempotent: an already-completed account keeps
-- its original completed_at (COALESCE) so re-submitting Finish never moves the timestamp.
-- key_added is NOT set here — it flips true only once the coach key store works (S11).
UPDATE identity.onboarding
SET completed_at = COALESCE(completed_at, now())
WHERE account_id = $1
RETURNING *;
