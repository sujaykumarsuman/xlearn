-- name: CreateSession :one
INSERT INTO identity.session (id, account_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetValidSession :one
SELECT * FROM identity.session
WHERE id = $1 AND revoked_at IS NULL AND expires_at > now();

-- name: RevokeSession :execrows
UPDATE identity.session
SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;
