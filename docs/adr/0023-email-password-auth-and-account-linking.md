# ADR-0023 — Email/password sign-in + GitHub↔email account linking

- **Status:** Accepted. **§2 and §3 amended 2026-09-24 by v1.5.2**: sign-up closed by default (xlearn#52) and auto-link only into OAuth-only accounts (xlearn#51); both are recorded in place below. **§2 further amended by [ADR-0033](0033-invite-only-admission-and-owner-admin.md) (v2, Accepted 2026-09-24):** the `invite` mode, the `DEV_AUTH` guard on `open`, and a login that is uniform in time.
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

> **Amended 2026-09-24: sign-up is closed by default (`SIGNUP_MODE`).** The owner closed v1 sign-up
> (v2 planning, T7.2). Identity reads `SIGNUP_MODE` = `open | closed`. Any value other than `open`,
> including unset, means **closed**, so a missing env fails safe; `invite` is reserved for v2. While
> closed:
>
> - `POST /auth/signup` returns **`403 signup_closed`** before the body is read, so it can't reveal
>   which emails are registered.
> - An OAuth sign-in that matches no account redirects to `/auth?error=signup_closed`
>   (`FindOrCreateAccount` with `NoCreate` returns `ErrSignupClosed` instead of creating).
>
> Existing accounts sign in as before: by password, by a linked provider, or by the email auto-link
> in §3 (as amended below). `docker-compose` sets `SIGNUP_MODE=open` for local review. The local-only
> dev login (`DEV_AUTH`, [ADR-0022](0022-path-enrollment-and-dev-login.md)) is not gated.

> **Amended 2026-09-24 by [ADR-0033](0033-invite-only-admission-and-owner-admin.md) §3, §5 and §14
> (accepted at the v2 build-plan sign-off).**
>
> - **`SIGNUP_MODE ∈ {closed, invite, open}`.** In `invite` mode, built at L, `POST /auth/signup` and
>   a new GitHub account both need a valid invite. It is redeemed inside the account-create transaction
>   under the `identity.seats` lock, and it is checked **before** the email. Every bad invite gets the
>   same `invite_invalid`; a new GitHub user with no invite gets `/auth?error=invite_required`.
>   `closed` refuses both, as above.
> - **`open` is local and dev only.** From M1b, identity honours `open` only when `DEV_AUTH` is also
>   set. Otherwise it runs as `closed` and logs at ERROR. v1.5.2 honours `open` without `DEV_AUTH`.
> - **Login is uniform in time as well as in status** (M1b). An unknown identifier runs a dummy bcrypt,
>   and bcrypt concurrency is capped (≤ 2 in flight, then 429), so the uniform `401` no longer leaks
>   through timing.
> - In `invite` mode, signup is no email oracle for anyone without a valid invite, because the invite
>   check comes before the email check. With a valid invite, a taken email rolls back the transaction,
>   so the invite is not consumed.

### 3. Collision = auto-link by verified email

When an OAuth sign-in's email already belongs to an account (and that identity isn't linked elsewhere),
`FindOrCreateAccount` **attaches the identity to the existing account** rather than creating a duplicate.
Provider emails are verified by the provider, so this safely merges an email sign-up and a later "Continue
with GitHub" for the same person. Explicit **Connect GitHub** in Settings remains for linking when the
emails differ.

> **Amended 2026-09-24: auto-link only into OAuth-only accounts.** The reasoning above checked that the
> *incoming* provider email was verified, but not the *existing* account's email. Email sign-up doesn't
> verify the address. So an attacker could sign up as `victim@…` with their own password, and the
> victim's first "Continue with GitHub" would sign them into the attacker's account, which the attacker
> can still open. This is pre-account hijacking
> ([USENIX Security 2022](https://arxiv.org/abs/2205.10174)).
>
> `FindOrCreateAccount` now auto-links only when the matched account has **no password**. Such an
> account is OAuth-only, so its email also came from a provider. When the account has a password, it
> links nothing, creates no duplicate, and returns `ErrPasswordAccountExists`. The callback then
> redirects to `/auth?error=account_exists_password`, and the SPA tells the user to sign in with their
> password and connect GitHub from Settings. Unchanged: a returning identity (matched before any email
> check), Settings link mode (an authenticated session), and links made before this fix.

## Consequences

- One account can now hold both a password and OAuth identities; the sign-in screen has a Sign in / Sign
  up pill, and Settings gains a **Sign-in & security** card (set/change password · connect/disconnect
  GitHub).
- **No verification** means an email sign-up trusts the address as given — acceptable for now because the
  address is only an identifier + login handle (no mail is sent); a verification + real reminders flow is
  deferred to a later change. Auto-link relies on the provider's email being verified (GitHub's is).
- The unique-email index fails the migration if pre-existing duplicate emails exist (none expected in
  prod — OAuth emails are unique).
