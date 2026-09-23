-- Email/password sign-in (ADR-0023). Adds an optional bcrypt password hash to the account
-- (OAuth-only accounts keep it NULL until they set one from Settings), and makes email
-- case-insensitively unique so email login + GitHub↔email account-linking are unambiguous.
-- No email verification yet (deferred) — a fresh email sign-up is usable immediately.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE identity.account ADD COLUMN IF NOT EXISTS password_hash text;
-- +goose StatementEnd
-- +goose StatementBegin
-- Case-insensitive uniqueness over the non-NULL emails (OAuth accounts without an email
-- keep NULL and are exempt). Fails the migration if duplicate emails already exist.
CREATE UNIQUE INDEX IF NOT EXISTS account_email_lower_unique
    ON identity.account (lower(email))
    WHERE email IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS identity.account_email_lower_unique;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE identity.account DROP COLUMN IF EXISTS password_hash;
-- +goose StatementEnd
