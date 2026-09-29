-- name: CreateSession :one
INSERT INTO identity.session (id, account_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetValidSession :one
-- A live session of an ACTIVE account (m1-04, ADR-0033 §7): the join on
-- status = 'active' kills every session of a suspended account at once, with no
-- revocation step. It returns what session-validate reports (§6): the account's role,
-- status and accepted_at, and the SESSION's created_at (L8's fresh-session check). It
-- runs on every API call and reads the account by primary key.
SELECT s.id, s.account_id, s.created_at, s.expires_at,
       a.role, a.status, a.accepted_at
FROM identity.session s
JOIN identity.account a ON a.id = s.account_id AND a.status = 'active'
WHERE s.id = $1 AND s.revoked_at IS NULL AND s.expires_at > now();

-- name: RevokeSession :execrows
UPDATE identity.session
SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllSessions :execrows
-- Revoke every live session of an account (m1-04): a password change (the caller's
-- own session included), the admin CLI's suspend and revoke-sessions.
UPDATE identity.session
SET revoked_at = now()
WHERE account_id = $1 AND revoked_at IS NULL;
