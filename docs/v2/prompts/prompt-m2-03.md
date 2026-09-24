# Prompt — Sprint m2-03 · `public-read`, `/public/stats`, visibility toggles, profile v2

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m2-03.md`](../sprints/sprint-m2-03.md)   ·   **Milestone:** M2 (M2b public slice)   ·   **Prereqs:** [m2-02](../sprints/sprint-m2-02.md) (v1.9.0 live), boards from [ds-m2-01](../sprints/sprint-ds-m2-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, stack, and the land-and-sync rule.
- The plan: [`../sprints/sprint-m2-03.md`](../sprints/sprint-m2-03.md). Its task list, table shapes and acceptance are authoritative for this session.
- [ADR-0033 §7, §13, §14](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas): `public-read`, visibility, and the ADR-0024 amendment. [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas): what each course shows and the defaults.
- [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): `preview` is hidden everywhere, public stats included. [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md): projections are composed at read time.
- [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas): what becomes public, defaults, enforcement, events and replay. [t0 §4](../research/t0-extensibility-frame.md#4-universal-vs-per-course-method): "passed" = grade ≥ rough.
- [Rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P3, P5, P6, P7, P9, P12) and [§4 M2](../rollout-plan.md#4-per-milestone-detail). [PRD §5.5 R-PP1](../../prd/xlearn-v2-prd.md).
- Boards: `design-system/screens/v2/AB06-*.html` (public profile v2) and `AB22-*.html` (visibility toggles), via `design-system/screens/v2/index.html`. Reuse [`design-system/theme.css`](../../../design-system/theme.css) verbatim.
- Code:
  - `internal/gateway/public.go`, `public_test.go`, `cache.go`, and `bff.go` (`apiRoutes`, the `mint*` helpers);
  - `internal/assessment/service.go`, `progress.go`, `store/projections.go`, and `store/queries/projections.sql`;
  - `internal/identity/usernames.go`, `service.go`, `handlers.go` (enrollment), `store/`;
  - `internal/platform/auth/` (M1-04's `RequireRole(v Verifier, role string)`, `RoleLearner`, `RolePublicRead`, and M1-04's role tests);
  - `internal/course/` (manifest visibility default);
  - `web/src/screens/UserDashboard.tsx`, `Settings.tsx`, `web/src/lib/profile.ts`, `settings.ts`, `web/src/components/ProgressViews.tsx`.

## Context

- **v1.9.0** (M2-02) shipped the projections v2, with producers idle:
  - `proj_activity`, dated by `anchor_at`;
  - `proj_touch_stats`;
  - `proj_outcome_mix_v2(account_id, path_slug, graded_by, trust, outcome, cnt)`;
  - `path_slug` on `proj_coverage` and `proj_mastery`.
- M2-02 kept the authed readers on v1. Its invariant is that a reader switches in the same tag as M2-05's replay. The authed switch is M2-05's; this sprint switches only the public reader (a recorded deviation, see step 3).
- **M1b** (M1-04/M1-05) already:
  - enforces `RequireRole("learner")` on user routes;
  - shrank the public payload: mock count only (P1), enrolled ∩ visible ∩ active courses via the resolver's `visible_courses[]` (P2), rate limits and the 60 s negative-404 cache (P4), the allowlist test (P10), and suspend → 404 (P11).
- **The gap this sprint closes:** the public route still mints an ordinary `learner` token and reads the same `/progress/*` endpoints as the authed Progress screen. There's no learner control over visibility, and the header totals are account-wide.
- This sprint makes `public-read` real, adds the toggles and moves the profile to AB06 and AB22.
- It merges dark into **v1.10.0**. [m2-05](../sprints/sprint-m2-05.md) cuts that tag after [m2-04](../sprints/sprint-m2-04.md), and runs the `touch_scored` backfill and projection replay that fill `proj_activity`'s history.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **v1.9.0 live**: `ssh vps 'k3s kubectl get deploy -n xlearn -o wide'` shows `1.9.0` (or later) images, and `/xlearn/api/v1/healthz` reports it. The projection v2 tables exist on `main`.
- [ ] **AB06 and AB22 frozen**: the DS-M2-01 PR is merged by the owner (`gh pr list --state merged --search "AB06"`), and the board files exist under `design-system/screens/v2/`.
- [ ] **The M1b public floor is on `main`**: `visible_courses[]` in the resolver, the P10 allowlist test, the negative-404 cache, and no positive status cache.
- [ ] **`auth.RequireRole` exists**, and `account.profile_visibility` / `path_enrollment.public_visible` exist in identity's schema.
- [ ] **Parallel sessions**: `gh pr list`, `git worktree list` and ListAgents. Check that no peer is editing `internal/gateway/public.go`, `cache.go` or `Settings.tsx`, and that no open PR touches identity migrations. If one does, take the next free goose version at rebase.

## Do this (in order)

1. **[X] `public-read` mint.** In `internal/gateway/public.go`:
   - add `mintPublicRead(accountID)` → `g.signer.Mint(ctx, accountID, g.audAssessment, []string{auth.RolePublicRead})`;
   - remove the public route's `mintQuiet(…, g.audAssessment)` (`learner`) call;
   - add a source test: the only `Mint(` call site whose roles include `auth.RolePublicRead` is `mintPublicRead`, and the `"public-read"` literal appears in no non-test Go file except the constant itself (assessment and everything else use `auth.RolePublicRead`).

2. **[X] assessment `GET /public/stats`.** New `internal/assessment/public.go`, registered in `service.go` as `mux.Handle("GET /public/stats", auth.RequireRole(s.verifier, auth.RolePublicRead)(http.HandlerFunc(s.handlePublicStats)))` (M1-04's signature verifies the bearer itself; no separate `requireJWT`). **Only** `public-read` passes.
   - `paths` is required and non-empty: course-slug regex, de-duplicated, ≤ 16. Anything else, including empty or missing, → 400.
   - Read the projections for `(sub, paths)` and return the plan's task-1 shape:
     - `header{streak, activity.days[{date,attempts,touches,mocks}], mocks}` (no `solved`: the gateway computes it);
     - `courses[{path, concluded, passed, outcomeMix, provenance, judgeCheckedPct, touches{completed, day7PassRate}, mocks, items[{id,best}]}]`.
   - Sources: `concluded`/`passed` from `proj_mastery` (`solve_count > 0`; `best_rank ≥ 3`, rough or better); `outcomeMix`, `provenance` and `judgeCheckedPct` from `proj_outcome_mix_v2`; touches from `proj_touch_stats`; activity and streak from `proj_activity`. **No new migration.**
   - **No mock best/average/last, `scored_by`, notes, rubric scores, or sub-day timestamps.**
   - Add sqlc queries, run `sqlc generate`, commit the output, and add an assessment-side shape test.

3. **[X] Gateway compose on `/public/stats`.** In `composePublicProfile`:
   - replace the three `/progress/*` calls with one `/public/stats?paths=<visible_courses csv>`;
   - keep the curriculum taxonomy fan-out for phases, patterns and `total`;
   - fold `items[]` into the roll-ups and **drop it from the response**;
   - compute header `solved` = Σ over the visible courses of `items[]` with `best` ≥ rough whose curriculum role is core (no role → core, so every DSA item counts); take the rest of the header from `header` only;
   - **empty `visible_courses[]` → no `/public/stats` call at all**: a zero header, no course rows, the empty state (not a 404). Test both sides: the gateway makes zero assessment calls; assessment answers 400 on empty `paths`;
   - this switches the **public** reader before M2-05's replay, a deliberate deviation from M2-02's "switch and replay in the same tag" invariant (display-only and self-healing; a guard would keep the `learner`-token compose that P3 removes). Record it in the Decisions log.

4. **[X] Public cache namespace.** In `internal/gateway/cache.go`:
   - add a public namespace (`"pub\x00"` prefix or a second map) sharing the per-account epoch, so `invalidate(account)` clears both;
   - move the public handler's `get`/`putFresh` onto it.

5. **[X] identity visibility.**
   - `PATCH /accounts/{id}/visibility`:
     - JWT, `RequireRole("learner")`, `sub == id`;
     - partial body `{profile?, courses?{slug: bool}}`: unknown key → 400, not enrolled → 422 `not_enrolled`;
     - one transaction; returns the full doc.
   - `GET /accounts/{id}` gains `visibility{profile, courses[{slug, visible, default}]}`, with `default` from `internal/course`.
   - Enrollment sets `public_visible` from the manifest default (D7). If M1-02 left `NULL`s or a constant default, add an expand-safe goose migration backfilling from the manifest (next free identity version).
   - `GET /internal/accounts/by-username/{username}` returns the **same 404** for unknown, private and suspended, on one code path.

6. **[X] Gateway `PATCH /api/me/visibility`.**
   - Proxy to identity. On a 2xx, call `g.cache.invalidate(accountID)` **and** purge that username's negative-404 entry **before** responding.
   - Add the route to `apiRoutes()` and `docs/architecture/openapi.yaml`. List it in the M1-06 route-enumeration allowlist ("no item data").

7. **[X] Web: profile v2 (AB06).** In `web/src/lib/profile.ts` and `UserDashboard.tsx`:
   - the header shows solved, streak, the activity heatmap (attempts, touches, mocks) and the mock count;
   - each visible course gets a row: concluded/passed/total, phases, patterns, grade mix with **provenance chips** (self; honor labelled self-reported), judge-checked % hidden or "—" until M3, touch stats and mock count;
   - private renders the same view as unknown; empty renders AB06's empty state;
   - add the provenance chips to the shared `web/src/components/ProgressViews.tsx`, with an optional `provenance` prop that renders nothing when absent. The authed Progress screen gets its provenance data from M2-05's reader switch (`/progress/outcome-mix`); **don't switch the authed reader here**.

8. **[X] Web: Settings visibility (AB22).** In `Settings.tsx` and `web/src/lib/settings.ts`:
   - a "Profile & visibility" rail tab with the profile switch and per-course switches showing the manifest default;
   - a confirm dialog naming what becomes hidden (the course's stats and its share of the header totals; private → "your profile link shows *not found*");
   - `PATCH /api/me/visibility`, no optimistic update;
   - `role="switch"`, keyboard reachable, a 390 px layout.

9. **[X] Tests.**
   - Extend the P10 allowlist (new fields allowed; no `items`, ids, sub-day timestamps, `scored_by`, `best`/`average`).
   - A `RequireRole` matrix: `public-read` → 403 on every `/progress/*` and `/mocks/*`; `learner` → 403 on `/public/stats`; no token → 401; the `aud=assessment` token rejected by identity, practice, review and coach.
   - Update M1-04's role tests: the per-service mux walk exempts exactly assessment's `GET /public/stats`, and the minted-role-set test allows `public-read` only via `mintPublicRead`.
   - Cache namespace separation, plus the epoch invalidation of both.
   - Toggle → the next public request recomposes and the negative cache is purged.
   - Byte-identical 404s for unknown, private and suspended.
   - `preview` never appears.
   - A real-Postgres integration test that the totals exclude a hidden path, plus a gateway test that header `solved` sums only the requested courses.
   - Web tests for UserDashboard and Settings.
   - Then: `go test ./...`, `sqlc diff`, `npm test` in `web/`, and the golden e2e in compose.

10. **[X] Docs.** Update [`docs/architecture/api.md`](../../architecture/api.md), `openapi.yaml` and [`services.md`](../../architecture/services.md) (`/public/stats` is `public-read` only; the public cache namespace and the epoch live in the single gateway replica, L24).

11. **Eyeball the screens** against AB06 and AB22 in the browser: compose, or the temporary vite mock config used for v1 visual checks. Use the 1440 px and 390 px layouts.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).**
  - assessment reads only its own projections, and identity only its own tables.
  - The gateway composes and never reads a schema; the manifest default is read from the compiled `internal/course` package.
  - No cross-schema reads.
- **goose + sqlc.**
  - Migrations are embedded and run on startup under the advisory lock. Expand only here: nullable or constant-default, backfill by `UPDATE`, no drops.
  - Commit the generated sqlc output. CI runs `sqlc diff`, and the contract-header lint must stay silent.
- **Authz:**
  - roles never go in the JWT beyond `learner`/`public-read`;
  - no `owner`/`tester` role is minted (ADR-0033 §7);
  - the `public-read` token is `aud=assessment` only.
- **Frontend:** `theme.css` tokens and components verbatim (no Tailwind), the dark theme, and a match to the frozen boards.
- **No new always-on pod, env var, NATS subject or in-cluster caller**, so there's no NetworkPolicy or ACL PR and the memory sum is unchanged. If you find you need one, stop: it's a plan change.
- **GitOps:** never `kubectl apply`. Read-only `ssh vps` only for the entry-gate checks.
- **No alerting of any kind (D34).**
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before merging, and before claiming an ADR number if one turns out to be needed.
- This sprint **does not tag**. v1.10.0 is cut by M2-05.

## Deliverables

- The gateway `public-read` mint, the `/public/stats`-based compose and the public cache namespace.
- `PATCH /api/me/visibility` with a synchronous epoch bump and a negative-cache purge.
- assessment `GET /public/stats` (`public-read` only) with a pinned public shape.
- identity `PATCH /accounts/{id}/visibility`, visibility on `GET /accounts/{id}`, manifest-default enrollment (+ a backfill migration if needed), and uniform 404s.
- UserDashboard v2 (AB06), provenance chips in `ProgressViews.tsx`, and the Settings "Profile & visibility" tab (AB22).
- Tests (allowlist, the role matrix, cache, uniform 404, `preview`, totals) and docs (`api.md`, `openapi.yaml`, `services.md`).

## Update status

- In [`../sprints/sprint-m2-03.md`](../sprints/sprint-m2-03.md): set each task 🔄 → ✅ (⛔ with a reason), and set _Overall_.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for M2-03;
  - the **Artboards** rows AB06 and AB22 → consumed by M2-03 (set "frozen (PR #, date)" if M2-01 didn't);
  - the public-dashboard rows P3, P5, P6, P7 (self), P9 and P12 → done (ships in v1.10.0), if the tracker carries them;
  - **Decisions log** lines for the header `solved` definition (gateway-side core filter), the `concluded`/`passed` derivation from `proj_mastery`, the public-reader deviation from M2-02's same-tag invariant (with its reason), and any other deviation from the plan.
- Record an ADR only if you depart from ADR-0033 §13. Run the parallel-sessions check before numbering it.

## Done when (acceptance)

- [ ] `public-read` cannot read `/progress/*` or `/mocks/*`, only `/public/stats` accepts it, and the public route mints nothing else.
- [ ] A private profile is indistinguishable from an unknown one (byte-identical 404), and a toggle purges the negative cache.
- [ ] Totals exclude hidden courses (header from visible courses via `proj_activity`), and `preview` is never shown.
- [ ] A toggle bumps the cache epoch synchronously, and the public and authed cache namespaces are separated and tested.
- [ ] UserDashboard matches AB06 and Settings matches AB22.
- [ ] CI green (`go test ./...`, `sqlc diff`, web tests, openapi drift, route enumeration, allowlist).

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: merge only (ships in v1.10.0)**. That means branch `feat/m2-03-public-read-visibility`, conventional commits with the attribution lines, a PR, CI green, and a squash-merge, then `git checkout main && git pull`. **Do not tag.** v1.10.0 is cut by [m2-05](../sprints/sprint-m2-05.md) after M2-04 merges. There's no infra PR.
