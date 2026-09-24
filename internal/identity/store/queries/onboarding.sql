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
-- The last onboarding step (Finish / Skip). Idempotent: an already-completed account keeps
-- its original completed_at (COALESCE) so re-submitting Finish never moves the timestamp.
-- The key_added column is unused: nothing ever set it, and whether a coach key is
-- connected is the coach service's to answer (GET /coach/key). It is left in place so a
-- rolling deploy's old pods (whose generated queries still name it) keep working; a later
-- migration can drop it.
UPDATE identity.onboarding
SET completed_at = COALESCE(completed_at, now())
WHERE account_id = $1
RETURNING *;
