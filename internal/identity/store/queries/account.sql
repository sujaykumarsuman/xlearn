-- name: CreateAccount :one
INSERT INTO identity.account (display_name, email)
VALUES ($1, $2)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM identity.account
WHERE id = $1;

-- name: GetAccountByProviderIdentity :one
SELECT a.*
FROM identity.account a
JOIN identity.oauth_identity oi ON oi.account_id = a.id
WHERE oi.provider = $1 AND oi.provider_user_id = $2;
