# ADR-0024 — Public user dashboards + usernames (the first public route)

- **Status:** Accepted — the bare `/xlearn/<username>` URL shape is **superseded by
  [ADR-0025](0025-public-profiles-under-u-prefix.md)** (profiles now live at `/xlearn/u/<username>`), and
  so are the reserved-word rules (§2 and the 2026-09-23 update below): the list is now impersonation-only
  and course slugs are not reserved. **v2:** [ADR-0033](0033-invite-only-admission-and-owner-admin.md) (Accepted
  2026-09-24; see [Amended 2026-09-24 by ADR-0033](#amended-2026-09-24-by-adr-0033)) enforces `public-read`, shows
  enrolled ∩ visible ∩ active courses only, and shows the mock count only (D31);
  [ADR-0032](0032-realtime-ai-mock-interviewer.md) (v2, Proposed) records the same mock-count-only change (D31).
- **Date:** 2026-09-23
- **Deciders:** @sujaykumarsuman
- **Related:** [0006](0006-authn-authz.md) (session gating — this adds the first UNauthenticated
  route), [0010](0010-spa-base-path-and-gateway-serving.md) (SPA base path + fallback), [0018](0018-progress-projection-grain-and-rebuild.md)
  (the projections + gateway roll-up this reuses), [0023](0023-email-password-auth-and-account-linking.md)
  (the identity/auth surface it extends). Tracks the post-1.0 UI/UX batch in [`docs/v1/feedbacks/`](../v1/feedbacks/) (F009).

## Context

Two feedback items land together: (1) surface per-course **progress** inside the course rather than
from the avatar menu, and (2) give each learner a **public, LeetCode-style dashboard** at the bare URL
`projects.sujaykumar.dev/xlearn/<username>`. (2) forces two firsts for xLearn:

- A **username** identity — a URL-safe public handle that appears bare in the URL and doubles as a
  sign-in identifier (email OR username).
- The **first public (unauthenticated) route**, both in the SPA and at the gateway. Until now every
  route except `/auth` sat behind `AuthedShell`, and every `/api` handler began with `authAccount`
  ([ADR-0006](0006-authn-authz.md)).

Owner decisions (sign-off): username is set in **Settings** with a **claim-gate** when the avatar →
Dashboard link is hit without one; profiles are **public by default** (non-PII only); the heatmap
reuses the existing solves+reviews projection; shipped as one batch (**v1.4.0**).

## Decision

### 1. Usernames on the account (identity, migration `00004`)

`identity.account` gains a nullable **`username`** (unclaimed until set). It is a lowercase URL-safe
slug — `^[a-z0-9]+(?:-[a-z0-9]+)*$`, **3–30 chars**, no leading/trailing/repeated hyphens — stored
normalised (lower-case) with a partial **`UNIQUE(lower(username)) WHERE username IS NOT NULL`** index,
mirroring the email-unique index from [ADR-0023](0023-email-password-auth-and-account-linking.md). Shape
+ reserved-word validation lives in the **service** (`internal/identity/username.go`), not the DB.

- `POST /accounts/{id}/username` (JWT) claims/changes it; `409` on a taken name, `422` on invalid/reserved.
- `GET /username/available?u=` (JWT) powers the live claim UI.
- `POST /auth/login` now resolves **email OR username** (an `@` means email, else username) — a uniform
  `401` on failure is preserved (no enumeration).
- `GET /me` exposes `username` (omitted until claimed).

### 2. Reserved words are a single source of truth

A username shares the `/xlearn` path namespace, so it must never shadow a current or future top-level
route, a gateway prefix, a static asset, or a confusable/system word. `reservedUsernames`
(`internal/identity/username.go`) is that list — current routes (`auth`, `settings`, `dsa`,
`claim-username`, …), gateway prefixes (`api`, `assets`, `.well-known`, probes), the profile namespace
(`u`, `user`, `profile`, `me`, …) and safety words (`admin`, `login`, …). **Whenever a new top-level
route or a new curriculum path slug is added, it must be added here too.** The React router keeps these
as real static routes; React Router ranks static segments above the dynamic `:username`, so a reserved
name resolves to its route, and the reserved list stops anyone from claiming one at write time.

### 3. The public dashboard is composed in the gateway from non-PII sources only

- **SPA:** a public React route `/:username` renders **outside** `AuthedShell` (like `/auth`), with its
  own slim chrome (brand + Sign in). The gateway already serves `index.html` for any non-`/api`,
  non-asset path ([ADR-0010](0010-spa-base-path-and-gateway-serving.md)), so no gateway change is needed
  to serve the shell — only a public API.
- **API:** `GET /api/u/{username}` is the **only** handler that does not call `authAccount`. It:
  1. resolves the username via identity's ClusterIP-only `GET /internal/accounts/by-username/{username}`,
     which returns **only** non-PII fields (id, username, display name, join date) — never email, OAuth
     identities, timezone, budget or sessions;
  2. mints read-scoped service tokens **for the resolved account** and fans out to `assessment`
     (projections) + `curriculum` (taxonomy), exactly as the authed Progress aggregation does
     ([ADR-0018](0018-progress-projection-grain-and-rebuild.md));
  3. composes **only** public aggregates: per-course completion + pattern mastery, account-wide solved
     count + streak + mock stats, and one activity heatmap **merged across courses** (the `proj_heatmap`
     projection is already account-wide). The response is cached per resolved account (public → shared
     across viewers; short TTL + the account's own write-invalidation bound staleness and blunt abuse).

**Why this is safe.** PII lives only in identity, and the public path touches identity solely through
the non-PII resolver. `assessment` and `curriculum` hold no PII. So no matter what the gateway composes,
nothing private can surface. Minting a service token for an arbitrary account on an anonymous request is
contained: the token only reaches read-only projection/taxonomy endpoints, and the handler filters the
output to the public shape. An unknown (or unclaimed) username is a uniform `404` — no existence detail.

### 4. Progress relocation (frontend only)

"Progress & stats" leaves the avatar menu; per-course **Progress** moves into the curriculum sidebar
(`navForPath`). The avatar menu gains **Dashboard** → `/xlearn/<username>`, or the **claim-username**
gate when the account has none yet. The authed Progress screen and the public dashboard share the same
presentational views (`components/ProgressViews.tsx`) so they never drift.

## Consequences

- Public-by-default means any learner who claims a username has a public page; it exposes only non-PII
  stats. A private/opt-in toggle can be added later as an account flag without changing the URL shape.
- Usernames are **editable** in Settings (changing one moves the public URL; v1 keeps no redirect from
  the old name). Email verification remains out of scope ([ADR-0023](0023-email-password-auth-and-account-linking.md)).
- A single-segment unknown path (`/xlearn/typo`) now resolves to the public profile route (a friendly
  "no such profile"), not the in-shell 404; multi-segment unknowns still hit the in-shell NotFound.
- Per-course stats intersect the account-wide projections with each path's problem set; streak/mock are
  account-wide today (one shared streak, LeetCode-style). Path-scoped mocks/streak are a later change
  when a second course ships.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **`/u/<username>` prefix** (avoid the collision) | Owner wants the bare `/xlearn/<username>` shape; the reserved-word list handles the collision cleanly. |
| **Dedicated public projection service / `/internal/public-stats` endpoints** | Stronger isolation, but more surface to build; mint-for-account + a non-PII resolver + output filtering already contains exposure, since assessment/curriculum hold no PII. Revisit if a public endpoint ever needs data from a PII-holding service. |
| **Opt-in (private by default)** | Safer, but the brief is LeetCode-style public profiles; only non-PII is exposed and unknown/unclaimed is a uniform 404. A visibility flag can be added later without a URL change. |
| **Fold mock attempts into the heatmap** | Needs an event/projection change + replay; the existing solves+reviews `proj_heatmap` is already account-wide and deterministic. Deferred. |

## Update — 2026-09-23: all path slugs + likely v2 routes reserved, slug coverage test-enforced

> **Superseded** by [ADR-0025's 2026-09-23 update](0025-public-profiles-under-u-prefix.md): course slugs and
> route words are no longer reserved, and `TestReservedCoversCurriculumPathSlugs` was removed.

The original list reserved only `dsa` among the curriculum path slugs; the coming-soon paths in
`curriculum/paths.json` (`system-design`, `go-concurrency`, `lld-ood`, `sql`, `behavioral`) are now
reserved too, plus likely v2 top-level route words (`judge`, `submissions`, `interview(s)`, `arena`,
`problems`, `course(s)`, `path(s)`). A read-only prod check on 2026-09-23 found **0 accounts** holding
any of the five slugs, so nothing was orphaned.

The "add new slugs here too" rule from §2 is now **enforced by a test**
(`TestReservedCoversCurriculumPathSlugs` reads the embedded `paths.json`), so adding a path without
reserving its slug fails CI. Reserve a route word **before** the route ships: the public-profile
resolver (`/internal/accounts/by-username/{username}`) re-validates against the list, so reserving a
name someone already holds turns their public profile into a 404. There is no web copy of the list;
the claim UI shows the reason returned by `GET /username/available`.

## Amended 2026-09-24 by ADR-0033

[ADR-0033](0033-invite-only-admission-and-owner-admin.md) §13 and §14, accepted at the v2 build-plan
sign-off, change §3's composition and the Consequences above. The text above stays as the v1 record. The
URL shape is ADR-0025's.

| Change | Milestone |
|---|---|
| **Mock count only (D31).** `best` and `average` leave the public payload and tile; the authed Progress keeps them. [ADR-0032](0032-realtime-ai-mock-interviewer.md) records the same rule for interviewer mocks | M1b |
| **Scope.** Only **enrolled ∩ visible ∩ active** courses show (D7); `preview` is hidden everywhere, public stats included. The resolver returns `visible_courses[]` | M1a columns → M1b filter |
| **Abuse limits.** 60/min per IP (burst 20); 404s cached negatively for 60 s; ≤ 8 concurrent cold composes | M1b |
| **Shape.** A public-shape allowlist test: no item ids, per-item pattern, sub-day timestamps, arena or Run counts, prose, `scored_by` or transcripts | M1b |
| **Suspend and erase.** A uniform 404 at once; the username is held for 60 days after erase | M1b (suspend) / L-E (erase) |
| **An enforced `public-read` role** replaces §3's ordinary learner mint. Only assessment's `GET /public/stats` accepts it; `/progress/*` and `/mocks/*` reject it; it gets its own cache namespace | M2b |
| **Visibility.** Profile and per-course toggles; this is the "private toggle later" from Consequences. Private returns the same 404 as unknown. A toggle bumps the cache epoch synchronously. Header totals come from visible courses only | M2b |

§3's safety argument still holds, and the mint is now narrower: a `public-read` token reaches only
`/public/stats`.
