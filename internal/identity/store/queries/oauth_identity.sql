-- name: CreateOauthIdentity :one
INSERT INTO identity.oauth_identity (account_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteOauthIdentity :execrows
-- Unlink a provider from an account (Settings: disconnect GitHub). Returns the affected
-- row count so a no-op can 404.
DELETE FROM identity.oauth_identity
WHERE account_id = $1 AND provider = $2;

-- name: ListOauthProviders :many
-- The providers linked to an account (Settings shows what's connected).
SELECT provider FROM identity.oauth_identity
WHERE account_id = $1
ORDER BY provider;
