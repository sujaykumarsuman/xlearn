# ADR-0023 — Email/password sign-in + GitHub↔email account linking

- **Status:** Accepted
- **Date:** 2026-09-23
- **Deciders:** @sujaykumarsuman
- **Related:** [0006](0006-authn-authz.md) (authn/authz — this adds a first-party credential path alongside OAuth), [0005](0005-data-ownership-and-migrations.md) (identity owns the schema change). Tracks the post-1.0 UI/UX feedback batch in [`docs/v1/feedbacks/`](../v1/feedbacks/) (F007).

## Context

v1 was **OAuth-only** ([ADR-0006](0006-authn-authz.md)); the sign-in screen carried an inert
email/password form. The feedback: let a new user **sign up / sign in with email + password**, let a
**GitHub** user set a password from Settings, and let an **email** user connect GitHub from Settings —
so the two identities converge on one account. There is **no email-verification or email-sending**
infrastructure yet, and it is explicitly out of scope for now.

## Decision

### 1. Passwords live on the account; email becomes unique

- `identity.account` gains a nullable **`password_hash`** (bcrypt, `DefaultCost`). OAuth-only accounts
  keep it `NULL` until they set one. A partial **`UNIQUE(lower(email)) WHERE email IS NOT NULL`** index
  makes email case-insensitively unique so email login + linking are unambiguous (OAuth accounts
  without an email stay exempt).
- Password policy: **8–72 chars** (the 72-byte bcrypt ceiling is enforced, not silently truncated). No
  composition rules yet. The hash is **never serialised** — `GET /me` exposes only a derived
  `has_password` boolean and the `linked_providers` list.

### 2. Endpoints (identity, proxied by the gateway)

- `POST /auth/signup` `{email, password}` → create account (display name defaults to the email
  local-part) + session; `409` on a taken email.
- `POST /auth/login` `{email, password}` → session; a **uniform `401`** on any failure (no
  user-enumeration).
- `POST /accounts/{id}/password` (JWT) → set/change. An account with a hash must supply the correct
  `current_password`; an OAuth-only account sets the first one without it.
- `DELETE /accounts/{id}/oauth/{provider}` (JWT) → disconnect, but **never the last sign-in method**
  (`409` unless a password or another provider remains).
- OAuth `POST /auth/{provider}/start?link=1` → **link mode**: the callback attaches the provider to the
  session's account and returns to `/settings`, instead of signing in.

The email flows are **fetch-based** (JSON), so identity sets the session cookie on the response and the
gateway passes `Set-Cookie` through — the browser still only ever talks to the gateway origin.

### 3. Collision = auto-link by verified email

When an OAuth sign-in's email already belongs to an account (and that identity isn't linked elsewhere),
`FindOrCreateAccount` **attaches the identity to the existing account** rather than creating a duplicate.
Provider emails are verified by the provider, so this safely merges an email sign-up and a later "Continue
with GitHub" for the same person. Explicit **Connect GitHub** in Settings remains for linking when the
emails differ.

## Consequences

- One account can now hold both a password and OAuth identities; the sign-in screen has a Sign in / Sign
  up pill, and Settings gains a **Sign-in & security** card (set/change password · connect/disconnect
  GitHub).
- **No verification** means an email sign-up trusts the address as given — acceptable for now because the
  address is only an identifier + login handle (no mail is sent); a verification + real reminders flow is
  deferred to a later change. Auto-link relies on the provider's email being verified (GitHub's is).
- The unique-email index fails the migration if pre-existing duplicate emails exist (none expected in
  prod — OAuth emails are unique).
