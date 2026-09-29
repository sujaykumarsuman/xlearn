-- The owner admin CLI (m1-04, ADR-0033 §8): `identity admin …`, run via kubectl exec
-- with the pod's own credentials. Every verb runs in one transaction that also writes
-- identity.admin_audit (admin_audit.sql). Guards serialize on transaction-scoped
-- advisory locks: 'identity.owners' (the last-owner guard) before 'identity.seats'
-- (SEAT_CAP), always in that order.

-- name: AdminXactLock :exec
SELECT pg_advisory_xact_lock(hashtext(sqlc.arg(lock_name)::text));

-- name: AdminLockAccountByID :one
SELECT * FROM identity.account
WHERE id = $1
FOR UPDATE;

-- name: AdminLockAccountByEmail :one
SELECT * FROM identity.account
WHERE email IS NOT NULL AND lower(email) = lower($1)
FOR UPDATE;

-- name: AdminLockAccountByUsername :one
SELECT * FROM identity.account
WHERE username IS NOT NULL AND lower(username) = lower($1)
FOR UPDATE;

-- name: AdminCountActiveOwners :one
SELECT count(*) FROM identity.account
WHERE role = 'owner' AND status = 'active';

-- name: AdminCountSeatsUsed :one
-- A seat (ADR-0033 §3) is an active learner. Invites join the count at L-A.
SELECT count(*) FROM identity.account
WHERE role = 'learner' AND status = 'active';

-- name: AdminSetStatus :one
UPDATE identity.account
SET status = $2
WHERE id = $1
RETURNING *;

-- name: AdminSetRole :one
-- Role only: admitted_via never changes (ADR-0033 §2).
UPDATE identity.account
SET role = $2
WHERE id = $1
RETURNING *;

-- name: AdminCreateAccount :one
-- `account create --role tester`: a password account provisioned by the CLI.
INSERT INTO identity.account (display_name, email, password_hash, admitted_via, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AdminListAccounts :many
-- `account list`: optional role/status filters; dormant_before keeps only accounts with
-- no session created at or after it. last_session_at is NULL for an account that never
-- signed in.
SELECT a.id, a.email, a.username, a.role, a.status, a.admitted_via, a.created_at,
       ls.last_session_at::timestamptz AS last_session_at
FROM identity.account a
LEFT JOIN LATERAL (
    SELECT max(s.created_at) AS last_session_at
    FROM identity.session s
    WHERE s.account_id = a.id
) ls ON true
WHERE (sqlc.narg('role')::text IS NULL OR a.role = sqlc.narg('role')::text)
  AND (sqlc.narg('status')::text IS NULL OR a.status = sqlc.narg('status')::text)
  AND (sqlc.narg('dormant_before')::timestamptz IS NULL
       OR ls.last_session_at IS NULL
       OR ls.last_session_at < sqlc.narg('dormant_before')::timestamptz)
ORDER BY a.created_at, a.id;
