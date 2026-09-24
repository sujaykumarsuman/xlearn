# Sprint m2-03 — `public-read`, `/public/stats`, visibility toggles, profile v2 (P3, P5, P6, P7, P9; AB06, AB22)

> **Milestone:** M2 — attempt engine and projections (M2b public slice)   ·   **Track:** product
> **Prereqs:** [m2-02](sprint-m2-02.md) (v1.9.0: projections v2 bound) · boards from [ds-m2-01](sprint-ds-m2-01.md)   ·   **Unblocks:** [m2-04](sprint-m2-04.md), [m2-05](sprint-m2-05.md)
> **Release action:** merge only (ships in **v1.10.0**, tagged by [m2-05](sprint-m2-05.md))   ·   **Calendar:** late October – early November (no owner involvement)
> **Execute with:** [`../prompts/prompt-m2-03.md`](../prompts/prompt-m2-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | `public-read` mint + assessment `GET /public/stats` + own cache namespace (P3) | X | ⬜ |
| 2 | Visibility toggles: identity `PATCH /accounts/{id}/visibility`, manifest defaults, uniform 404, synchronous epoch bump (P5) | X | ⬜ |
| 3 | Header totals from visible courses only via `proj_activity` (P6) | X | ⬜ |
| 4 | Profile v2 (AB06 / P12, with P7 self provenance + P9 touch stats) + Settings visibility (AB22) | X | ⬜ |
| 5 | Tests: allowlist, `RequireRole` negatives, cache namespace + epoch, uniform 404, `preview` hidden | X | ⬜ |
| 6 | Docs: `api.md`, `openapi.yaml`, `services.md` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **v1.9.0 live** ([m2-02](sprint-m2-02.md)): assessment projections v2 (`proj_activity` dated by `anchor_at`, `proj_touch_stats`, `proj_outcome_mix_v2(account_id, path_slug, graded_by, trust, outcome, cnt)`, and `path_slug` on `proj_coverage`/`proj_mastery`) are bound in prod, and `problemTotal=151` is gone.
- [ ] **AB06 and AB22 frozen**: the [ds-m2-01](sprint-ds-m2-01.md) design PR is merged by the owner (boards under `design-system/screens/v2/`, indexed by `design-system/screens/v2/index.html` from DS-M1-01).
- [ ] **The M1b public floor is on `main` and live in v1.7.0+** ([m1-05](sprint-m1-05.md)): P1 (mock count only), P2 (the resolver returns `visible_courses[]` = enrolled ∩ visible ∩ active), P4 (limits and the 60 s negative-404 cache), P10 (the public-shape allowlist test) and P11 (suspend → 404, with no positive status cache).
- [ ] **`auth.RequireRole` exists** in `internal/platform/auth` and guards the user routes ([m1-04](sprint-m1-04.md)).
- [ ] **The identity visibility columns exist**: `account.profile_visibility` and `path_enrollment.public_visible` ([m1-02](sprint-m1-02.md)).

## Goal

Give the only unauthenticated route its **reduced authority** and give the learner **control over what it shows**.
- The gateway's public compose mints an **enforced `["public-read"]`** token. Only assessment's new `GET /public/stats` accepts it, and `/progress/*` and `/mocks/*` reject it.
- The learner gets a **profile toggle and per-course toggles**, with defaults from each course manifest (D7). A private profile is **indistinguishable from an unknown one**. A toggle bumps the gateway cache epoch **synchronously**.
- The **header totals** (solved, streak, heatmap, mocks) count **visible courses only**, read from `proj_activity`. This fixes v1's first-solve inflation.
- The public profile moves to **AB06**: per-course rows, grade provenance (self for now), touch stats and mock count only. **AB22** adds the Settings toggles.

This is the M2b public slice of [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P3, P5, P6, P7 self, P9 display, P12). It follows [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas), [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas) and [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas).

## Scope

**In**
- gateway (`internal/gateway/public.go`, `cache.go`, a new BFF route): the `public-read` mint, one `/public/stats` call instead of three `/progress/*` calls, a public cache namespace, and `PATCH /api/me/visibility` with a synchronous epoch bump and a negative-cache purge.
- assessment: `GET /public/stats?paths=` behind `auth.RequireRole(s.verifier, auth.RolePublicRead)` only. It returns only the public shape, from the v2 projections. No new migration.
- identity: `PATCH /accounts/{id}/visibility`, visibility in `GET /accounts/{id}`, manifest-default visibility at enrollment, and private and suspended profiles resolving to the uniform 404.
- web: `UserDashboard.tsx` (AB06), the shared `ProgressViews.tsx` provenance chips, and a Settings "Profile & visibility" section (AB22).

**Out (later sprints)**
- Judge-checked % values and judge stats (P8): [m3-10](sprint-m3-10.md). AB06 shows judge-checked % hidden or "—" until then.
- AI provenance on grades (P7 AI): [m4-06](sprint-m4-06.md).
- The erase → 404 half of P11 and the 60-day username hold: [l-02](sprint-l-02.md).
- The touch UI and Today in minutes: [m2-04](sprint-m2-04.md). Producers, the `touch_scored` backfill and the prod projection replay: [m2-05](sprint-m2-05.md). Until M2-05's replay, `proj_activity` holds only post-v1.9.0 history (see Risks).
- Switching the **authed** Progress and Dashboard readers to projections v2 (and so the authed provenance chips' data): [m2-05](sprint-m2-05.md), in the same tag as the replay, per M2-02's hand-off. The authed Progress screen's full v2 deltas (AB12): [m3-13](sprint-m3-13.md).

## Tasks

### 1 · `public-read` mint + `GET /public/stats` + own cache namespace (P3) [X]

**Gateway** (`internal/gateway/public.go`):
- Add a helper `mintPublicRead(accountID)` that calls `g.signer.Mint(ctx, accountID, g.audAssessment, []string{auth.RolePublicRead})` (the constant from [m1-04](sprint-m1-04.md)'s `internal/platform/auth/require.go`).
  - `sub` is the **resolved** account and `aud=assessment`, so the token is useless at every other service.
  - It is the only `public-read` mint in the codebase. A source test enforces it on two rules:
    - the only `Mint(` call site whose roles include `auth.RolePublicRead` is `mintPublicRead`;
    - the `"public-read"` literal appears in no non-test Go file except the constant itself. Assessment's route registration and every other reference use `auth.RolePublicRead`.
  - The public route never mints `learner` again (today it does: `aToken, _ := g.mintQuiet(acct.AccountID, g.audAssessment)`).
- `composePublicProfile` replaces its three `/progress/{summary,heatmap,mastery}` calls with **one** `GET /public/stats?paths=<csv>`.
  - `paths` is exactly the resolver's `visible_courses[]` (M1-05 P2: enrolled ∩ visible ∩ active). The gateway never adds a slug of its own.
  - It keeps the curriculum taxonomy fan-out (`courseTaxonomy`) for the per-course phase and pattern roll-ups ([ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md): composition at read time).

**Assessment** (new `internal/assessment/public.go` handler, registered in `service.go`):
- `mux.Handle("GET /public/stats", auth.RequireRole(s.verifier, auth.RolePublicRead)(http.HandlerFunc(s.handlePublicStats)))`, following M1-04's signature `RequireRole(v Verifier, role string)`: it verifies the bearer itself, so there's no separate `requireJWT` wrapper. **Only** `public-read` passes: a `learner` token gets 403 `forbidden`.
- `paths` is required and non-empty: comma-separated course slugs, each matching the course-slug regex `^[a-z0-9]+(-[a-z0-9]+)*$`, de-duplicated, at most 16. Anything else, **including an empty or missing `paths`**, → 400. The gateway never sends an empty list (task 3).
- Reads the projections for `(sub, path_slug ∈ paths)` only. Response (camelCase, public shape only):

  | Field | Source | Notes |
  |---|---|---|
  | `header.streak{current,longest}` | days with any activity in `proj_activity` over `paths` | UTC days |
  | `header.activity.days[{date, attempts, touches, mocks}]` | `proj_activity` over `paths`, last 16 weeks | dates only, never sub-day |
  | `header.mocks` | count of scored `mock_session` rows over `paths` | **count only (D31)** |
  | `courses[].{path, concluded, passed}` | `proj_mastery` for the path: concluded = rows with `solve_count > 0`; passed = rows with `best_rank ≥ 3` (rough or better, [T0 §4](../research/t0-extensibility-frame.md#4-universal-vs-per-course-method)) | the gateway adds `total` from curriculum |
  | `courses[].outcomeMix{clean,rough,assisted,miss}` + `provenance{self,auto,ai,override,honor}` | `proj_outcome_mix_v2` (`graded_by`, `trust`, `outcome`, `cnt`) | P7: self only until M3 |
  | `courses[].judgeCheckedPct` | `proj_outcome_mix_v2`: Σ`cnt` where `graded_by=auto ∧ trust=checked` ÷ Σ`cnt` | always 0 before M3; the UI hides it |
  | `courses[].touches{completed, day7PassRate}` | `proj_touch_stats` | P9 |
  | `courses[].mocks` | scored mock count for that path | count only |
  | `courses[].items[{id, best}]` | `proj_mastery` | **gateway-internal**: used only for the phase and pattern roll-ups and the header `solved` sum, and never forwarded (the P10 test enforces it) |

- It never returns mock `best`/`average`/`last`, `scored_by`, `notes`, rubric scores, timestamps finer than a day, or anything keyed by mock or attempt id. An assessment-side shape test pins the field list.
- **`header.solved` is computed by the gateway, not assessment**, because assessment knows no item role. It is Σ over the visible courses of passed **core** items: an item from `courses[].items[]` with `best` ≥ rough whose role in the curriculum taxonomy (already fetched for the roll-ups) is core. An item with no role counts as core, so every DSA item counts. The label follows AB06.
- **No new assessment migration.** `concluded` and `passed` come from `proj_mastery` in the read query, as the table says. Record this derivation and the gateway-side core filter in the Decisions log. A read-side `path_slug IN (…)` filter needs no new index at this data size.

**Cache namespace:**
- `internal/gateway/cache.go` gets a distinct public namespace (e.g. `publicKey(account, name) = "pub\x00" + account + "\x00" + name`, or a second map). An authed `get(account, name)` can never return a public entry, and the public path can never return an authed one.
- **Both namespaces share the per-account epoch**, so `invalidate(account)` (the account's own writes, and toggles in task 2) clears both.
- The public handler moves its `get`/`putFresh` onto the namespace; the name stays `public-profile`.

### 2 · Visibility toggles, manifest defaults, uniform 404, synchronous epoch bump (P5) [X]

**identity** (`internal/identity/`: new `visibility.go`, `store` queries; sqlc regenerate):
- `PATCH /accounts/{id}/visibility` (JWT, `RequireRole("learner")`; `sub == id`, else 403).
  - Body, partial: `{"profile": "public"|"private", "courses": {"<slug>": true|false}}`. An unknown key → 400. A slug the account isn't enrolled in → 422 `not_enrolled`.
  - One transaction updates `account.profile_visibility` and `path_enrollment.public_visible`.
  - Returns the full visibility document.
- `GET /accounts/{id}` (what `GET /api/me` serves) gains `visibility: {profile, courses: [{slug, visible, default}]}`. `default` comes from the compiled manifest's public-visibility default (`internal/course`, D7: `true` except behavioral).
- **Defaults from the manifest.** Enrollment (`handleStartEnrollment`) sets `public_visible` from the course manifest.
  - If M1-02 left the column nullable or constant-defaulted, add an expand-safe goose migration that backfills `NULL`s from the manifest default (DSA → `true`). Take the next free identity version at rebase.
  - Resolver reads use `public_visible` only, never a runtime default.
- **Private = unknown.** `GET /internal/accounts/by-username/{username}` returns the **same 404** for:
  - an unknown username;
  - `profile_visibility='private'`;
  - `status='suspended'` (M1-05 P11).

  Same status, same body, same headers, on the same code path, with no extra query only on the private branch. The gateway maps all three to its single `not_found` envelope.

**Gateway BFF:**
- `PATCH /api/me/visibility` → identity. On a 2xx, **before** writing the response:
  - `g.cache.invalidate(accountID)` (the synchronous epoch bump; ADR-0033 §13);
  - drop that account's username from the 60 s negative-404 cache (M1-05 P4), so private → public shows on the very next request.
- Add the route to `apiRoutes()` (the openapi drift test). The route-enumeration test from M1-06 lists it as "no item data".

### 3 · Header totals from visible courses only (P6) [X]

- The public header comes **only** from `/public/stats`: its `header` (streak, activity, mocks), computed over the requested (visible) `paths` with `proj_activity` as the activity source, plus `solved`, which the gateway sums from the courses' items (task 1). This replaces v1's account-wide `/progress/summary` totals and the `proj_heatmap` merge (`public.go` today reads `Totals.Solved` and `Mock` from the account-wide summary).
- **No visible course: the gateway short-circuits.** When the resolver's `visible_courses[]` is empty, the gateway makes **no** `/public/stats` call and composes a zero header (solved 0, streak 0/0, empty `activity.days`, mocks 0) with no course rows. The page renders its empty state; it does **not** 404, because only a private profile does. Assessment keeps answering 400 on an empty `paths`.
  - Tests on each side: gateway (`public_test.go`, fake assessment) — an empty `visible_courses[]` makes zero assessment calls and returns the zero header; assessment (`handlers_test.go`) — `paths=` empty or missing → 400.
- Days stay UTC ([t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas)).
- **Integration test** (real Postgres, `internal/assessment/store`): seed activity in two paths (`dsa` plus a fixture slug) and request only `dsa`. Streak, activity, per-course `concluded`/`passed` and mocks exclude the other path. A gateway test (fake assessment and curriculum) checks that header `solved` sums only the requested courses' passed core items.

### 4 · Profile v2 (AB06 / P12) + Settings visibility (AB22) [X]

Build to the frozen boards (`design-system/screens/v2/AB06-*.html`, `AB22-*.html`; index at `design-system/screens/v2/index.html`), with `theme.css` verbatim and the dark theme.

- **`web/src/lib/profile.ts` + `web/src/screens/UserDashboard.tsx`:**
  - The header shows solved, streak, the activity heatmap (attempts, touches, mocks) and the **mock count**, all from visible courses.
  - One row per visible course:
    - concluded vs passed vs total;
    - completion by phase and pattern mastery (existing roll-ups);
    - the grade mix with **provenance chips** (self now; "judge" from M3; "AI" from M4), with honor results labelled as self-reported;
    - **judge-checked %** hidden or "—" until M3 (never a 0% presented as a claim);
    - touch stats (touches completed, Day-7 pass rate);
    - mock count only.
  - **Private** renders the identical "no such profile" view as an unknown user. **Empty** (public, no visible course or no activity) renders AB06's empty state.
- **`web/src/components/ProgressViews.tsx`:** the shared outcome-mix view gains the provenance chips. They take an optional `provenance` prop and render nothing without it. The public profile feeds them now. The authed Progress screen gets its provenance when [m2-05](sprint-m2-05.md) switches the authed readers to `/progress/outcome-mix` (same tag, v1.10.0); M2-02 exposes provenance only there, and the gateway doesn't call it yet. **Don't switch the authed reader here.** The rest of AB12 is M3.
- **`web/src/screens/Settings.tsx` + `web/src/lib/settings.ts`:** a new rail tab **"Profile & visibility"** (AB22):
  - the profile public/private switch;
  - one toggle per enrolled course showing its manifest default ("visible by default" / "hidden by default");
  - a confirm dialog that names what becomes hidden: the course's stats **and its share of the header totals**, or for private, "your profile link shows *not found*";
  - it saves through `PATCH /api/me/visibility` and reads through `GET /api/me`. No optimistic update, because the server is the truth.
- Layout at 390 px; keyboard reachable; the toggles are real `<button role="switch" aria-checked>` elements.

### 5 · Tests [X]

- **P10 allowlist extended** (`internal/gateway/public_test.go`):
  - allowed: the new fields (provenance counts, `judgeCheckedPct`, touch stats, mock count);
  - still refused: item ids (so `items[]` must not be forwarded), per-item pattern, sub-day timestamps, arena or Run counts, prose, `scored_by`, `best`/`average`, transcripts.
- **`RequireRole` negative matrix** (`internal/assessment/handlers_test.go`, table-driven):
  - `public-read` → 403 on every `/progress/*` and `/mocks/*` route;
  - `learner` → 403 on `/public/stats`;
  - no token → 401;
  - the `public-read` token (`aud=assessment`) is rejected by identity, practice, review and coach verifiers (one audience test each).
- **M1-04's role tests, updated:** its per-service mux walk (every JWT route → 403 for `public-read`) exempts exactly one route, assessment's `GET /public/stats`; and its minted-role-set test now allows `public-read`, minted only by `mintPublicRead` (the source test in task 1).
- **Cache** (`cache_test.go`, `public_test.go`):
  - the same account and name in both namespaces never collide;
  - `invalidate` clears both;
  - a toggle through `PATCH /api/me/visibility` makes the next public request recompose (epoch captured before the compose is stale);
  - the negative-404 entry is purged on toggle.
- **Uniform 404:** unknown vs private vs suspended return byte-identical responses (status, body, `Content-Type`, `Cache-Control`).
- **`preview` hidden:** an owner enrolled in a `preview` fixture course, marked visible, never gets it in `paths` and never gets it in the payload. Tested like the course-slug guard ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)).
- **Web:** `UserDashboard.test.tsx` (header, per-course row, provenance chips, private = unknown, empty) and `Settings.test.tsx` (toggle, default label, confirm copy, PATCH body).
- The full suite stays green: `go test ./...`, `sqlc diff`, web tests, and the golden e2e.

### 6 · Docs [X]

- [`docs/architecture/api.md`](../../architecture/api.md) and `openapi.yaml`: `PATCH /api/me/visibility`, the `visibility` block on `GET /api/me`, and the public profile's v2 shape.
- [`docs/architecture/services.md`](../../architecture/services.md): assessment's `GET /public/stats` (`public-read` only) and identity's visibility endpoint. Note that the public cache namespace and the epoch live in the single gateway replica (L24, a scale-out blocker).
- No new ADR is expected. The design is [ADR-0033 §13/§14](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas) (amending ADR-0024). Add a Decisions-log line for anything that deviates, such as the header `solved` definition.

## Acceptance criteria

- [ ] **`public-read` cannot read `/progress/*` or `/mocks/*`.** Only `GET /public/stats` accepts it, a `learner` token can't read `/public/stats`, and the public route mints nothing else.
- [ ] **A private profile is indistinguishable from an unknown one**: byte-identical 404s, and the negative cache is purged on toggle.
- [ ] **Totals exclude hidden courses**: header solved, streak, activity and mocks come from visible courses only, via `proj_activity`, and `preview` is never shown.
- [ ] A toggle bumps the cache epoch synchronously: the next public request reflects it.
- [ ] Public and authed cache entries are namespaced apart and tested.
- [ ] UserDashboard matches AB06 (per-course rows, self provenance, touch stats, mock count only, private and empty states), and Settings matches AB22.
- [ ] CI green: `go test ./...`, `sqlc diff`, web tests, openapi drift, route enumeration, allowlist.

## Release

**Merge only: ships in v1.10.0.** [m2-05](sprint-m2-05.md) cuts the tag after [m2-04](sprint-m2-04.md) merges.
- There's no infra change: no new caller (the gateway already calls assessment and identity), so there's no NetworkPolicy PR, no env and no ACL.
- **Don't cut a tag between this merge and v1.10.0** if you can avoid it. If a peer or v1 fix does, the public header shows only post-v1.9.0 activity until M2-05's backfill and replay.
- **A recorded deviation from M2-02's invariant** ("the switch and the replay land in the same tag, so no reader sees partial history"): this sprint switches the **public** reader to `proj_activity` without a guard. It's accepted because the gap is display-only and self-healing at M2-05's replay, while a guard would have to keep the old compose (a `learner` token reading `/progress/*`) alive, which is the authority leak P3 closes. M2-04's guard differs because a touch concluded with no producer is lost for good. Record the deviation in the Decisions log. The authed readers keep M2-02's invariant: M2-05 switches them in the replay tag.

## Definition of Done

CI green · merged to `main` via a squash PR (no tag) · the screens match AB06 and AB22 · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board row, the AB06/AB22 artboard rows set to consumed, and the P3/P5/P6/P7-self/P9/P12 rows if the tracker carries them) · Decisions-log lines for the header `solved` definition (gateway-side core filter), the `concluded`/`passed` derivation, the public-reader deviation from M2-02's invariant, and any other deviation · no ADR number claimed without the parallel-sessions check.

## Risks / watch-outs

- **Cache namespace mixing public and authed entries.** A public compose served to the owner's authed dashboard, or the reverse, would leak or stale data. Separate keys are tested, and both are cleared by the shared epoch.
- **A private-profile oracle.** Any difference (status, body, header, cache behaviour, or a measurably slower branch) between private and unknown reveals that the account exists. Keep one code path in identity and one envelope in the gateway, and test byte equality.
- **Stale public data after a toggle.** The composed-payload cache *and* the 60 s negative-404 cache must both reset on toggle. The epoch alone doesn't cover the negative cache.
- **`judgeCheckedPct` = 0 before M3** reads as "nothing checked" rather than "not measured". Hide it per AB06 until M3-10 fills it.
- **The owner's own public numbers change.** `proj_activity` replaces the scheduling-inflated heatmap (+5 per first clean solve), and the header counts visible courses only. This is expected: put it in the v1.10.0 release notes (M2-05).
- **Partial history until M2-05's replay.** `proj_activity` was born empty in v1.9.0 (`DeliverNew`). The backfill and replay in M2-05 fill it, so don't judge the numbers before then. An interleaved tag would publish that partial history: the accepted deviation in *Release*.
- **Single gateway replica** (L24): the epoch and both caches are process-local. Already recorded as a scale-out blocker, so don't add a second replica.
