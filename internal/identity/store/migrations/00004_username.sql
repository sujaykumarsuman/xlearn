-- Usernames (F009 / ADR-0024): a URL-safe public handle powering the LeetCode-style public
-- dashboard at /xlearn/<username>, and an alternative login identifier (email OR username).
-- Nullable — existing + OAuth-only accounts have none until they claim one from Settings —
-- and case-insensitively unique so the public URL and username sign-in are unambiguous.
-- The allowed shape (slug, length, reserved words) is enforced in the service, not the DB.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE identity.account ADD COLUMN IF NOT EXISTS username text;
-- +goose StatementEnd
-- +goose StatementBegin
-- Case-insensitive uniqueness over the non-NULL usernames (accounts without one keep NULL
-- and are exempt). Mirrors the email unique index from migration 00003.
CREATE UNIQUE INDEX IF NOT EXISTS account_username_lower_unique
    ON identity.account (lower(username))
    WHERE username IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS identity.account_username_lower_unique;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE identity.account DROP COLUMN IF EXISTS username;
-- +goose StatementEnd
