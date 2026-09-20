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
