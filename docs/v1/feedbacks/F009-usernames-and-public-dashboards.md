# F009 — Progress relocation + usernames + public user dashboards

## Feedback

A three-part batch (see [ADR-0024](../../adr/0024-public-user-dashboards-and-usernames.md)):

1. **Relocate the progress tracker.** Drop "Progress & stats" from the avatar menu; surface per-course
   progress from *within* the course instead. Add a **Dashboard** item to the avatar menu that goes to
   the user's public profile.
2. **Public user dashboard** at `projects.sujaykumar.dev/xlearn/<username>` — a LeetCode-style public
   profile: per-course stats + one activity heatmap merged across courses. Public (no login).
3. **Usernames** — add a `username` to the account (URL-safe, case-insensitively unique), a flow to
   claim it, and let sign-in accept **email OR username**.

## Decision

Settled with the owner up front (the five open calls):

1. **Username flow** — set in **Settings** (canonical), with a **claim-gate** when the avatar → Dashboard
   is used without one. Editable (a change moves the public URL; no old-URL redirect in v1).
2. **Visibility** — **public by default**, non-PII only. Unknown/unclaimed → a uniform 404.
3. **Routing** — the bare `/xlearn/<username>` shape, guarded by a **reserved-word list** (single source
   in `internal/identity/username.go`; the SPA keeps reserved names as real static routes).
4. **Heatmap** — reuse the existing `proj_heatmap` (solves + reviews), already account-wide/merged.
5. **Release** — one batch, built for local review, shipped as **v1.4.0** on the go-ahead.

Architecture + the security model for the first public route are in
[ADR-0024](../../adr/0024-public-user-dashboards-and-usernames.md). The public API resolves the username
via identity's **non-PII** ClusterIP resolver, then composes only public aggregates from `assessment` +
`curriculum` (which hold no PII) — so nothing private can surface.

## Scope / changes

- **identity**: migration `00004` (`username` + partial unique index); `account.sql` queries
  (`GetAccountByUsername`, `SetUsername`) + sqlc; `store.go` (`Username` field, `ErrUsernameTaken`);
  `username.go` (normalise/validate + reserved words); `usernames.go` (set / availability /
  internal-by-username handlers); `authemail.go` (login resolves email OR username);
  `handlers.go` (`username` in `accountJSON`); `service.go` (route mounts). Tests: `usernames_test.go`.
- **gateway**: `bff.go` (identity-client `setUsername` / `usernameAvailable` / `publicAccountByUsername`;
  `handleSetUsername` + `handleUsernameAvailable` proxies; three routes in `apiRoutes`); `public.go`
  (the public `GET /u/{username}` aggregation — the only unauthenticated handler). Tests: `public_test.go`
  + the shared agg harness gains the resolver + `/paths` catalog.
- **web**: new public `screens/UserDashboard.tsx` + authed `screens/ClaimUsername.tsx`;
  `components/ProgressViews.tsx` (shared Heatmap / CompletionByPhase / PatternMastery, extracted from
  Progress); `lib/profile.ts` (`usePublicProfile`); `lib/auth.ts` (`username` on Account,
  `useSetUsername`, `useUsernameAvailability`); `router.tsx` (public `/:username` + `/claim-username`);
  `nav.ts` (Progress into the curriculum nav; `routeTitle`); `Topbar.tsx` (avatar Dashboard link);
  `Auth.tsx` ("Email or username"); `Settings.tsx` (username field in Sign-in & security). Tests:
  `UserDashboard.test.tsx`, `ClaimUsername.test.tsx`, updated `nav`/`Auth`/`AppShell` tests.
- **docs**: [ADR-0024](../../adr/0024-public-user-dashboards-and-usernames.md); this entry;
  `openapi.yaml` (`/me/username`, `/username/available`, `/u/{username}`, `username` on Account +
  `PublicProfile` schema — the route-drift test enforces it).

## Status

🚀 **Shipped (`v1.4.0`)** — merged (xlearn#39) → `v1.4.0` tag → deploy built all seven `1.4.0` images →
Flux deployed → **verified live**: gateway `v1.4.0`; the public `GET /api/v1/u/{username}` is live
**unauthenticated** (unknown user → `404 "no such user"`, vs `404 "no such endpoint"` for a bogus path —
route discrimination); `POST /api/v1/me/username` and `GET /api/v1/username/available` are present and
session-gated (→401). Built + reviewed on the local docker-compose stack first (the F001–F008 review-then-
ship convention); shipped on the owner's explicit go-ahead. Green throughout: Go build/vet/`sqlc diff` +
`go test ./...`; web typecheck/lint/build + 98 tests. No infra change (the additive `00004` migration runs
on identity startup).

## Review round (public dashboard)

Owner review of the first local build asked for two changes to `/xlearn/<username>`:

- **Header** — when the *viewer* is signed in, the public page now shows the normal authenticated
  header (brand + notifications + account menu) by reusing `Topbar variant="plain"`; the "Sign in"
  button only appears for anonymous visitors (brand-only while `/me` is still loading, so it doesn't
  flash). Also fixed a latent bug: the standalone page wasn't rendering `IconSprite`, so its icons were
  invisible.
- **Layout → 20:80** — a left identity column (silhouette avatar placeholder, name, @username, **Region**,
  16-week activity heatmap) and a right column (Problems-solved / Current-streak / Mock-best tiles, then
  **collapsible per-course rows** — collapsed shows the overview, expanding reveals completion-by-phase +
  pattern mastery). Stacks to one column on tablet/mobile.
- **Region** — a **coarse UTC-offset band** (e.g. `UTC+05:30`), derived server-side from the account
  timezone in identity's non-PII resolver; the raw IANA zone is never exposed (owner chose the offset
  granularity over continent / full-zone / omit).

A 5-dimension adversarial review workflow (frontend/a11y · responsive-CSS · privacy-region · regression ·
design-intent) ran over the change: 3 dimensions clean, **1 confirmed finding fixed** — the collapsed
course-row progress meter used `<span class="ds-meter">`, but `.ds-meter` needs block layout, so the bar
never drew; since the meter sits inside the clickable `<button>` header (phrasing-content only), it's kept
as spans with an explicit `display:block` rather than switched to `<div>`.

## Notes

- **First public route.** Every other `/api` handler is session-gated; `GET /u/{username}` is the sole
  exception, and it never reads a PII source (see ADR-0024's security model). DEV_AUTH stays local-only.
- Per-course stats intersect account-wide projections with each path's problem set; streak/mock are
  account-wide (one shared streak). Path-scoped mocks/streak arrive when a second course ships.
- Reserved-word upkeep: adding a new top-level route or curriculum slug means adding it to
  `reservedUsernames` too (called out in ADR-0024).
