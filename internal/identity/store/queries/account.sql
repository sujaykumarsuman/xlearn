-- name: CreateAccount :one
INSERT INTO identity.account (display_name, email)
VALUES ($1, $2)
RETURNING *;

-- name: CreateEmailAccount :one
-- Create an account from an email sign-up (ADR-0023): email is required + case-insensitively
-- unique (partial index), and password_hash is the pre-computed bcrypt hash.
INSERT INTO identity.account (display_name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM identity.account
WHERE id = $1;

-- name: GetAccountByEmail :one
-- Look up an account by email, case-insensitively (email sign-in + link-by-email). Returns
-- the row incl. password_hash — never serialised to a client.
SELECT * FROM identity.account
WHERE lower(email) = lower($1);

-- name: SetAccountPassword :one
-- Set or replace the account's bcrypt password hash (Settings: set/change password).
UPDATE identity.account
SET password_hash = $2
WHERE id = $1
RETURNING *;

-- name: GetAccountByUsername :one
-- Look up an account by username, case-insensitively (username sign-in + the public
-- profile at /xlearn/<username>). ErrNotFound when no account has claimed that username.
SELECT * FROM identity.account
WHERE username IS NOT NULL AND lower(username) = lower($1);

-- name: SetUsername :one
-- Claim or change the account's username (F009). The partial unique index on
-- lower(username) enforces case-insensitive uniqueness; a collision surfaces as a unique
-- violation the store maps to ErrUsernameTaken. The value is validated + normalised (lower,
-- reserved-word check) in the service before it reaches here.
UPDATE identity.account
SET username = $2
WHERE id = $1
RETURNING *;

-- name: GetAccountByProviderIdentity :one
SELECT a.*
FROM identity.account a
JOIN identity.oauth_identity oi ON oi.account_id = a.id
WHERE oi.provider = $1 AND oi.provider_user_id = $2;

-- name: UpdateAccount :one
-- Partial update of the caller's own account (PATCH /me, S10). A NULL narg leaves
-- the column unchanged (COALESCE), so profile / budget / timezone / reminders can be
-- saved independently. The jsonb blobs are validated + canonicalised in the handler.
UPDATE identity.account
SET display_name      = COALESCE(sqlc.narg('display_name'), display_name),
    timezone          = COALESCE(sqlc.narg('timezone'), timezone),
    study_budget_json = COALESCE(sqlc.narg('study_budget_json')::jsonb, study_budget_json),
    reminders_json    = COALESCE(sqlc.narg('reminders_json')::jsonb, reminders_json)
WHERE id = $1
RETURNING *;
