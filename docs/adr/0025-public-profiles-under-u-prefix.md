# ADR-0025 — Public profiles move under `/xlearn/u/<username>`

- **Status:** Accepted
- **Date:** 2026-09-23
- **Deciders:** @sujaykumarsuman
- **Related:** supersedes the **URL shape** of [0024](0024-public-user-dashboards-and-usernames.md)
  (the bare `/xlearn/<username>` route); the rest of 0024 — usernames, email-OR-username login, the
  non-PII public composition and its security model — stands. [0010](0010-spa-base-path-and-gateway-serving.md)
  (SPA fallback). Tracks [`docs/v1/feedbacks/F009`](../v1/feedbacks/F009-usernames-and-public-dashboards.md).

## Context

ADR-0024 put the public dashboard at the bare `/xlearn/<username>`, so usernames shared one namespace
with every top-level route and curriculum slug. React Router ranks static segments first, so a claimed
name that later became a route would turn into an unreachable profile. The mitigation was a reserved-word
list that had to **predict** future routes, and it cut both ways: the by-username resolver re-validates
against the list, so reserving a word someone already held would 404 their profile. That upkeep was
already slipping (only `dsa` of the six seeded path slugs was reserved), and a mistyped single-segment
URL (`/xlearn/setings`) showed a "no such profile" page instead of the in-shell 404.

## Decision

- The public profile route is **`/xlearn/u/<username>`** (SPA route `/u/:username`, still outside
  `AuthedShell`). This matches the public API, which was already `GET /api/v1/u/{username}`.
- The bare `/xlearn/<username>` route is **dropped, with no redirect**. Profiles went live the same day
  (`v1.4.0`), so few links exist; a redirect would keep the shared namespace (and the typo → "no such
  profile" behaviour) alive. An unknown `/xlearn/<x>` is now the in-shell NotFound.
- In-app links follow: the avatar menu's **Dashboard**, the Settings "Live at …" link, and the
  onboarding URL preview all use `/xlearn/u/<username>`.
- The **reserved-word list stays**, re-purposed: it no longer protects routes (a username can't shadow
  one now) but keeps official-looking or confusing handles (`@dsa`, `@admin`, `@settings`, …) out of
  public URLs and the username login. Every curriculum path slug is still required
  (`TestReservedCoversCurriculumPathSlugs`). `u` stays reserved.

No backend, API, or infra change: the gateway already serves `index.html` for any non-`/api`, non-asset
path (ADR-0010).

## Consequences

- Adding a course or top-level route can never break a profile, so route words no longer need to be
  reserved ahead of time. Growing the reserved list is now rare, which also makes the
  "reserving a held name 404s its profile" hazard rare (check prod before adding a word).
- **Breaking for any bare profile link shared since `v1.4.0`**: it now shows the in-shell NotFound
  (or the sign-in page for a signed-out visitor). Accepted given the one-day exposure.
- Profile URLs are two characters longer.
- Supersedes ADR-0024's note that a single-segment unknown path resolves to the profile route.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Keep bare `/xlearn/<username>`** (ADR-0024) | Needs the reserved list to predict every future route; reserving late 404s a live profile; typos render "no such profile". |
| **`/u/` plus a redirect from the bare path** | Keeps usernames in the shared namespace for the redirect's lifetime; not worth it for links that existed for under a day. |
| **`/@<username>`** | `@` in a path segment is legal but awkward to type/share and easy to mangle in chat/Markdown autolinkers; `/u/` matches the existing API and LeetCode's own `/u/<username>`. |

## Update — 2026-09-23: course slugs and route words are not reserved; the list is impersonation-only

Owner direction: usernames live at `/xlearn/u/<username>`, courses at `/xlearn/<course-id>`, and
**usernames are not reserved for courses**. This replaces the Decision bullet above that kept the list
"re-purposed" with every curriculum path slug required:

- `reservedUsernames` drops every course slug (`dsa`, `system-design`, `go-concurrency`, `lld-ood`,
  `sql`, `behavioral`), every route-collision word (`auth`, `settings`, `api`, `assets`, probes, the
  profile namespace `u`/`user`/`profile`/…, `dashboard`, `progress`, …) and the likely-v2 route words
  (`judge` … `path`) that the reservation hardening added. None of them can shadow anything under `/u/`.
- `TestReservedCoversCurriculumPathSlugs` is deleted. `TestValidateUsername` now asserts that course
  slugs and route words **are** claimable.
- What remains is an **impersonation/system-only** list: `xlearn`, `admin`, `root`, `support`, `help`,
  `about`, `system`, the auth-flow words (`login`, `logout`, `signin`, `signup`, `register`), and
  `terms`, `privacy`, `legal`, `new`, `edit`, `index`, `null`, `undefined`, `none`, `anonymous`. This
  is the default; the owner can veto or trim it.
- The course-side guard moves to the curriculum: a course slug must avoid the static top-level SPA
  segments (`u`, `auth`, `settings`) and the gateway-owned `api`, `assets`, `healthz`, `readyz` and
  `.well-known`. It is designed with v2's dynamic `/:course/*` routes (`docs/v2/feasibility.md`, T0) as
  a seed/test check, not as a username rule.

Removing reserved words cannot orphan a profile; only **adding** a word can, because the by-username
resolver re-validates. The shipped `v1.4.2` list is a superset of the new one, so no prod check was
needed.
