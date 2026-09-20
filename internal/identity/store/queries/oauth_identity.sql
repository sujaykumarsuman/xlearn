-- name: CreateOauthIdentity :one
INSERT INTO identity.oauth_identity (account_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING *;
