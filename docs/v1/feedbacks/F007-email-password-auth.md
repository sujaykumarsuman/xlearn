# F007 — Email/password sign-in + GitHub↔email account linking

## Feedback

The login screen only offered **GitHub OAuth** (the email/password fields were inert). The ask:

- A **Sign in / Sign up pill** just below the "or" divider, above the email field, with the
  email + password form made functional.
- A **GitHub** sign-up sets the account email to the GitHub one; the user can **set a password**
  (first time) or reset it in **Settings**.
- An **email** sign-up can **connect GitHub** from **Settings**.
- **Skip email verification** for now (none exists — confirmed) — verification + real reminders
  come later.

## Decision

Recorded in **[ADR-0023](../../adr/0023-email-password-auth-and-account-linking.md)**. Highlights:

- `account.password_hash` (bcrypt, nullable) + a **case-insensitive unique email** index.
- Email endpoints: `POST /auth/signup`, `POST /auth/login` (uniform 401), `POST /me/password`
  (set/change), `DELETE /me/oauth/{provider}` (never the last sign-in method), and OAuth
  **link mode** (`?link=1`) for Settings → Connect GitHub.
- **Collision → auto-link by verified email** (owner's pick): an OAuth sign-in whose email matches
  an existing account attaches to it instead of creating a duplicate.
- Password policy **8–72 chars**; `GET /me` exposes only `has_password` + `linked_providers`.

## Scope / changes

- **identity**: migration `00003` (`password_hash` + unique email); queries (`CreateEmailAccount`,
  `GetAccountByEmail`, `SetAccountPassword`, `DeleteOauthIdentity`, `ListOauthProviders`); store
  (`CreateEmailAccount`/`GetAccountByEmail`/`SetAccountPassword`/`LinkOAuth`/`UnlinkOAuth`/
  `ListOAuthProviders` + auto-link-by-email in `FindOrCreateAccount`); `password.go` (bcrypt +
  validation); `authemail.go` (signup/login/set-password/unlink handlers); OAuth start/callback
  **link mode**; `/me` gains `has_password` + `linked_providers`. Tests for all of it.
- **gateway**: proxy `POST /api/auth/signup|login` (Set-Cookie passthrough), `POST /api/me/password`,
  `DELETE /api/me/oauth/{provider}`; `?link=1` flows through the existing OAuth proxy.
- **docs**: `openapi.yaml` (4 routes + `EmailAuthRequest`/`SetPasswordRequest` + Account fields);
  ADR-0023.
- **web**: `lib/auth.ts` (`useSignup`/`useLogin`/`useSetPassword`/`useUnlinkOAuth`, `oauthLinkAction`,
  Account fields); `Auth.tsx` (Sign in / Sign up pill + live email form); `Settings.tsx` (a
  **Sign-in & security** card: set/change password · connect/disconnect GitHub, with the link
  redirect banner). Tests updated.

### Review-round refinements (post-build, in local review)

- **Auth pill**: the Sign in / Sign up selector is now full-width (field-equal) with circular ends
  (`ds-seg--block` + `ds-seg--pill` in `theme.css`).
- **Settings nav → tabs**: the section rail (F006) was a scroll-spy that couldn't reliably reach or
  highlight the last item (Sign-in & security). It's now a **tablist** — one section shown at a time,
  a deterministic click, no scroll math. Deep links pick the tab: `?tab=<id>` (the coach "open
  settings" prompts now use `?tab=coach`) and the GitHub link-return (`?linked=/?error=`) opens the
  account tab so its banner shows.
- **OAuth wording**: the "Set a password above to disconnect GitHub" line (which read as *setting a
  password disconnects GitHub*) was reworded — adding a password never disconnects anything; both
  methods coexist, and the only-sign-in-method guard is just so you can't lock yourself out.

Shell polish requested in the same review round (logo home-link + live Practice-loop badges) is
tracked separately in [F008](F008-shell-logo-and-live-badges.md) and ships in the same release.

## Status

🔄 **In progress** — code-complete and green (Go build/vet/gofmt/test incl. new identity + gateway
tests, `sqlc diff`, OpenAPI drift; web typecheck/lint + 88 tests). Bringing up the local
docker-compose stack for review at `http://localhost:8080/xlearn/auth`. Land-and-sync only after an
explicit go-ahead (ships as `v1.3.0` — identity schema migration included).

## Notes

- **No email verification / no email is sent** — a sign-up trusts the address as an identifier + login
  handle; verification + reminders are a later change.
- Auto-link trusts the provider's verified email (GitHub's is).
- Disconnecting a provider is blocked when it's the account's only sign-in method (set a password first).
