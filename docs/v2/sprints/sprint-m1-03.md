# Sprint m1-03 — Course resolution: gateway, BFF, SPA /:course/*, producers emit v2 (M1b)

> **Milestone:** M1 — spine (**M1b**) · **Track:** product · **Order:** 20
> **Prereqs:** [m1-02](sprint-m1-02.md) (`v1.6.0` live) · [ds-m1-01](sprint-ds-m1-01.md) (AB01–AB03 frozen)
> **Unblocks:** [m1-04](sprint-m1-04.md) (gateway router serialized m1-03 → m1-04 → m1-05 → m1-06) · [m1-10](sprint-m1-10.md) · [m1-07](sprint-m1-07.md)
> **Release action:** **merge only** (ships in `v1.7.0`, tagged by [m1-07](sprint-m1-07.md))
> **Calendar:** weeks 2–3 (≈ 2026-10-12 → 10-16, beside spike week)
> **Artboards:** AB02 (course nav) — built against the frozen board from [ds-m1-01](sprint-ds-m1-01.md)
> **Execute with:** [`../prompts/prompt-m1-03.md`](../prompts/prompt-m1-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Gateway course resolution + course-scoped routes + DSA aliases | X | ⬜ |
| 2 | curriculum: concept route, learner-safe course view, readers on new columns | X | ⬜ |
| 3 | Producers emit the v2 envelope; readers/writers leave every M1c-drop column | X | ⬜ |
| 4 | SPA `/:course/*`, `useCourse()`, manifest nav, switcher, `coming_soon`, NotFound (AB02) | X | ⬜ |
| 5 | Coach `page_context` prefix + `path_slug` | X | ⬜ |
| 6 | Enrollment slug validation (active only) | X | ⬜ |
| 7 | Tests + CI gates (route/alias parity, nav, literal grep, dropped-column queries) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] `v1.6.0` live — every consumer accepts the v2 envelope ([m1-02](sprint-m1-02.md); check `/xlearn/api/v1/healthz` and the ImagePolicies)
- [ ] AB01–AB03 frozen — [ds-m1-01](sprint-ds-m1-01.md) merged (the merge is the freeze, D40)
- [ ] No open peer PR edits `internal/gateway/bff.go`'s route table or `web/src/router.tsx` (the M1b gateway edits are serialized; `gh pr list`, ListAgents)

## Goal

Replace every `dsa` literal with **manifest-driven course resolution** end to end — gateway, BFF routes, curriculum,
SPA routes and nav, enrollment — switch every producer to the **v2 envelope**, and move every reader and writer off
the M1c-drop columns, while **DSA stays pixel- and API-identical to v1** (AB02 parity frames). After `v1.7.0` nothing
but coach `is_default` (m1-10) reads or writes a column [m1-08](sprint-m1-08.md) drops.

## Scope

**In**
- `internal/gateway/course.go`: resolve `{slug}` against the compiled manifests (`internal/course`, m1-01).
- Course-scoped BFF routes under the **existing `/api/paths/{slug}/…` prefix** ([t0 §7](../research/t0-extensibility-frame.md)),
  items by global id, and **DSA alias routes** kept ≥ 1 release after the SPA stops calling them
  ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.1).
- curriculum `GET /paths/{slug}/concepts/{c}` (+ DSA alias) and a learner-safe course view on `/paths*`.
- SPA `/:course/*`, `useCourse()`, manifest-driven `nav.ts` / `Sidebar.tsx` / `PathSwitcher.tsx`, `coming_soon` card,
  unknown course → NotFound, the shared course-slug guard list (AB02).
- Producers (practice, review, assessment, identity) emit v2; readers use `total`/`max_total`, `role`, `links`,
  `templates`, `path_slug`; writers stop writing `total_35`, `is_reinforcement`, `leetcode_url`/`neetcode_url`,
  `code_template` and write `path_slug` explicitly.
- Coach `page_context` `<course>:<ctx>` on every course-scoped context (concept, week, roadmap, dashboard, revision,
  mistakes, mock, progress) and `path_slug` writes; `problem:<id>` and account-wide contexts unchanged.
- Enrollment accepts only an `active` course ([ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 8).

**Out**
- `preview` for the owner/tester cohort (enrollment + resolver) → [m1-04](sprint-m1-04.md) (needs the role from session-validate).
- CSP, `RequireRole`, sessions → [m1-04](sprint-m1-04.md); limits and the public floor → [m1-05](sprint-m1-05.md); `withhold()` → [m1-06](sprint-m1-06.md).
- coach keys/catalog and stopping `is_default` writes → [m1-10](sprint-m1-10.md); coach mode gate / D27 → [m1-07](sprint-m1-07.md).
- Multi-course fidelity (catalog/agenda, Today in minutes D4, a second course) → [m2-04](sprint-m2-04.md), [p-02](sprint-p-02.md).
- `problem_solved` v2's M2a/M3 data fields (`attempt_id`, `graded_by`, …) → [m2-01](sprint-m2-01.md) and later; this sprint only
  moves the envelope to v2 with `path_slug`.
- Removing the DSA aliases (a later tag, recorded in status.md).

## Tasks

### 1 · Gateway course resolution + course-scoped routes + DSA aliases [X]

Sources: [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §5; [t0 §7](../research/t0-extensibility-frame.md) (URL model, API);
[ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.1 (aliases), §2 (`preview` hidden outside the cohort).
- **Route prefix decision:** course-scoped aggregates live under the existing `/api/paths/{slug}/…` (t0 §7), not a new
  `/api/courses/…` namespace — one prefix, no parallel tree. Record it in the decisions log.
- `internal/gateway/course.go`: `resolveCourse(slug) (course.Manifest, error)` over the embedded manifests, plus
  `courseVisible(m, cohort bool)`: `active` → yes; `coming_soon` → listed in the catalog only (data routes 404);
  `retired` → 404; `preview` → **404 for everyone in this sprint** (m1-04 passes the cohort bit). Unknown → 404
  `course_not_found` (uniform envelope).
- New routes in `apiRoutes()` (`internal/gateway/bff.go`) **and** [`../../architecture/openapi.yaml`](../../architecture/openapi.yaml)
  (the drift test enforces it): `GET /api/paths/{slug}/dashboard`, `…/progress`, `…/revision/due`, `GET|POST …/mistakes`,
  `…/weak-area`, `POST …/mocks`, `…/mocks/trend`, `…/concepts/{c}`. Existing `/api/paths/{slug}`, `…/problems`,
  `…/weeks/{n}` gain resolution. Items stay by global id (`/api/problems/{id}`, `/api/revision/{itemId}/score`,
  `/api/mistakes/{id}`, `/api/mocks/{id}`): the course comes from the item's `path_slug` (curriculum already returns it)
  or the row's.
- **DSA aliases** — `/api/dashboard`, `/api/progress`, `/api/revision/due`, `GET|POST /api/mistakes`, `/api/weak-area`,
  `POST /api/mocks`, `/api/mocks/trend`, `/api/concepts/{slug}` — call the course-scoped handler with `course.DefaultSlug`. `apiRoute`
  gains an `Alias` field; OpenAPI marks them `deprecated: true`. The SPA stops calling them in `v1.7.0`; they stay at
  least through `v1.8.0` — record the earliest removal tag in status.md.
- Replace the literal call sites: `bff.go` (`problemGateFor(…, "dsa", …)` ≈ l.571; `requireEnrolled(…, "dsa")` ≈ l.647),
  `scheduling.go:74`, `dashboard.go:166,174`, `progress.go:153,161,214`. Downstream calls pass the course:
  review `GET /revisions/due|/mistakes|/weak-area/current?path=<slug>`, assessment `/mocks/trend|/progress/*?path=<slug>`
  and `POST /mocks {path_slug}`, practice writes carry `path_slug` of the resolved item (review/assessment/practice add
  the `path` filter/param in `internal/{review,assessment,practice}/handlers.go`, defaulting to `course.DefaultSlug` when
  absent for mixed-version safety).
- **One default, one file:** `internal/course/default.go` holds `const DefaultSlug = "dsa"`, the course a v1 caller means.
  Every mixed-version default and alias site in this sprint uses it (the gateway aliases, the internal `?path=` defaults,
  curriculum's `GET /concepts/{slug}`, coach's legacy-context parser) — never a bare `"dsa"`. The literal gate (task 7)
  allowlists only that file for this purpose.
- Cache names include the slug (`dashboard:<slug>`, `progress:<slug>`, …; `week:` already does).

### 2 · curriculum: concept route, learner-safe course view, readers on new columns [X]

Sources: [t1 §4 curriculum](../research/t1-content-data-model.md); [t0 §8](../research/t0-extensibility-frame.md) (curriculum serves a learner-safe manifest view).
- `internal/curriculum/service.go` + `handlers.go`: `GET /paths/{slug}/concepts/{c}` keyed on `(path_slug, slug)` (m1-09's composite
  unique); `GET /concepts/{slug}` stays as the DSA alias, resolving with `course.DefaultSlug` (task 1).
- `GET /paths` / `GET /paths/{slug}` add a `course` block from a new `internal/course/view.go` (`LearnerView()`: status, nav
  labels and order, item noun, stage labels and timers, `est_minutes`, mock rail labels, short code) — nothing
  answer-bearing. `GET /paths` returns every non-retired course with its status; the gateway filters `preview` (task 1).
- Readers switch to `role`, `status`, `links`, `templates`, `language` only; the JSON keeps the v1 fields **derived** from
  them (`is_reinforcement = role == 'reinforcement'`, `leetcode_url`/`neetcode_url` from `links`, `code_template` from
  `templates`) plus the new `role`/`links` — API additive only.
- The seed/loader stops writing `is_reinforcement`, `leetcode_url`, `neetcode_url`, `code_template` (nullable since m1-09).

### 3 · Producers emit v2; readers/writers leave every M1c-drop column [X]

Sources: [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3; [t0 §5](../research/t0-extensibility-frame.md); [t1 §9 Events and replay](../research/t1-content-data-model.md).
- Replace the four hand-rolled envelope marshallers (`marshalEnvelope` in `internal/{practice,review,assessment}/store/store.go`,
  `marshalAccountCreated` in `internal/identity/store/store.go`) with m1-02's `events.NewEnvelope(2, …)`:
  - practice (`problem_solved`, `attempt_logged`, `solution_revealed_early`) — `path_slug` of the attempt;
  - review (`revision_scheduled`, `revision_due`, `mistake_opened`, `mistake_closed`) — from the row;
  - assessment (`mock_completed`) — `path_slug`; `data` gains `rubric_id`, `total`, `max_total`, `scored_by` and **keeps
    `total_35`** for a 35-point rubric (the payload is append-only; the DB column is a separate matter);
  - identity (`account_created`) — v2, account-scoped, no `path_slug`.
- Writers write `path_slug` **explicitly** everywhere (m1-08 drops the `'dsa'` defaults). assessment **stops writing
  `total_35`** (m1-02 relaxed the CHECK) and reads only `total`/`max_total` and `mock_session_item`.
- **review's weekly weak-area upsert leaves the v1 unique:** `UpsertWeakAreaSnapshot`
  (`internal/review/store/queries/weak_area_snapshot.sql`, today `ON CONFLICT (account_id, week_of)`) becomes
  `ON CONFLICT (account_id, path_slug, week_of)` (m1-02's `weak_area_snapshot_account_path_week_uq`) and inserts `path_slug`
  explicitly. The weekly recompute runs per `(account_id, path_slug)` over `mistake_entry.path_slug`, and
  `GetLatestWeakArea` filters on the `?path=` course (task 1). This is what lets [m1-08](sprint-m1-08.md) drop the v1
  `UNIQUE (account_id, week_of)` ([m1-07](sprint-m1-07.md) task 5 checks it).
- The rollback floor after `v1.7.0` becomes **1.6.0** (v2 envelopes in the log) — m1-07 records it.
- Gate: `hack/lint-dropped-columns.sh` fails if any `internal/*/store/queries/*.sql` references `total_35`,
  `is_reinforcement`, `leetcode_url`, `neetcode_url`, `code_template`, **or** uses the retired conflict target
  `ON CONFLICT (account_id, week_of)` (whitespace-tolerant match); coach `is_default` is allowlisted until m1-10
  removes it (m1-08 reuses this script against the `v1.7.0` tree).

### 4 · SPA `/:course/*` + manifest nav (AB02) [X]

Sources: [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §5; [t0 §7](../research/t0-extensibility-frame.md); AB02 (frozen board
`design-system/screens/v2/AB02-*.html` from [ds-m1-01](sprint-ds-m1-01.md)); [`../../../design-system/theme.css`](../../../design-system/theme.css).
- `web/src/router.tsx`: the ten `dsa/*` children become one `:course` subtree inside `CurriculumShell` (`""` Roadmap,
  `problems`, `dashboard`, `week/:n`, `concept/:slug`, `problem/:id`, `revision`, `mistakes`, `mock`, `progress`); the static
  `settings`, `auth`, `u/:username` keep winning by rank (tested).
- `web/src/lib/course.ts`: `useCourse()` → `{slug, view, status}` from the session-gated catalog; unknown → `NotFound`;
  `coming_soon` → the AB02 coming-soon card; a `coursePath(slug, …)` helper replaces every `/dsa` link.
- `web/src/nav.ts` renders the manifest `nav` block; `crumbsFor`/`titleFor` go generic over `/:course/*`. DSA output is
  **identical** to v1 (snapshot tests).
- `Sidebar.tsx`, `PathSwitcher.tsx` (short code from the view, current fallback kept), `AppShell.tsx`, `NotFound.tsx`, and the
  literal sites in `Revision`, `Catalog`, `Roadmap`, `Auth` (onboarding path picker reads the active courses), `Dashboard`,
  `Progress`, `Week`, `Concept`, `Problems`, `Mock`, `Problem`, `Mistakes`; `web/src/lib/*.ts` call the course-scoped endpoints.
- Course-slug guard: `web/src/lib/courseSlugGuard.ts` exports the reserved segments (`u, auth, settings, api, assets, healthz,
  readyz, privacy`); a parity test asserts it equals the list m1-09's Go guard enforces.
- **DSA parity:** compose at `v1.6.0` vs this branch, every DSA screen at 1440 px and 390 px — screenshots in the PR; any
  difference beyond AB02's intended frames is a bug. `theme.css` verbatim, dark theme, no new tokens.

### 5 · Coach `page_context` prefix + `path_slug` [X]

Sources: [t0 §7](../research/t0-extensibility-frame.md) (coach contexts: "Contexts that are not problems become
`<course>:<ctx>`"); [t1 §4 coach](../research/t1-content-data-model.md). The thread key is
`coach_thread UNIQUE (account_id, page_context)`, so **every course-scoped context** needs the prefix, not only concepts:
without it, DSA week 3 and a second course's week 3 (or their Revision, Mock, … screens) would share one thread once
[p-02](sprint-p-02.md) adds a course. This goes beyond t1 §4's concept-only line and follows t0 §7; record that in the
decisions log.
- `web/src/components/Coach.tsx` (`useCoachContext` and `genericContext`, ≈ l.258–340), with `course` from `useCourse()`:

  | v1 context | v1.7.0 context | Scope |
  |---|---|---|
  | `problem:<id>` | unchanged | global item id |
  | `catalog` (`/`), `settings`, `general` | unchanged | account-wide |
  | `concept:<slug>` | `<course>:concept:<slug>` | course |
  | `week:<n>` | `<course>:week:<n>` | course |
  | `roadmap`, `dashboard`, `revision`, `mistakes`, `mock`, `progress` | `<course>:roadmap`, `<course>:dashboard`, … | course |

  The route matchers go generic over `/:course/*` (no `/dsa` literal).
- coach service (`internal/coach/handlers.go`, thread read + chat) and `internal/gateway/coach.go`: one shared **dual
  parser** normalizes a context before it becomes a key. A context is new-form only when its first segment resolves to a
  course and its second is a known kind. Otherwise a legacy course-scoped form (`concept:<slug>`, `week:<n>`, the six bare
  words, from open `v1.6.0` tabs) maps to `course.DefaultSlug` + `:` + the context; `problem:*` and the account-wide ones
  pass through. Write `coach_thread.path_slug` / `coach_message.path_slug` (problem → the item's course; course-scoped →
  the prefix; account-wide → `NULL`).
- One idempotent coach data migration (next free version, no contract statement): `path_slug = 'dsa'` on existing
  course-scoped threads/messages, and rewrite `page_context` for `concept:%`, `week:%` and the six bare words to the
  `dsa:` form, guarded (`WHERE page_context NOT LIKE 'dsa:%'`) so a re-run is a no-op. Production has 0 coach rows
  (read-only check, 2026-09-25), and an R-b tab simply opens a fresh thread.
- Tests: the parser table (every v1 form → its v1.7.0 key; new forms unchanged; `problem:*`/account-wide untouched), the
  migration on seeded v1 contexts (and a second run), and a two-course test (`dsa` and `zz-fixture` week 3 get separate
  threads).

### 6 · Enrollment slug validation [X]

Source: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 8.
- `internal/identity/handlers.go` `handleStartEnrollment` (+ the gateway's `handleStartPath`): `course.Lookup(slug)` —
  unknown or `preview` → 404 `course_not_found`; `coming_soon`/`retired` → 409 `course_not_available`; `active` → enroll
  (writes `public_visible` from the manifest, m1-02). The cohort branch for `preview` is [m1-04](sprint-m1-04.md) task 7.

### 7 · Tests + CI gates [X]

- **Fixture manifests** (`internal/course/testdata/`: an `active` `zz-fixture`, a `coming_soon`, a `preview`, a `retired`),
  injected through a test-only registry — never embedded in an image.
- Route tests for every course-scoped route with `dsa` and `zz-fixture`; 404 for unknown/preview/retired; **alias parity**:
  `/api/X` and `/api/paths/dsa/X` return byte-identical bodies against the same fakes.
- Producer tests: every outbox row practice/review/assessment/identity writes is version 2, course-scoped ones carry
  `path_slug`; mi-05's subject-registry test green; the 19-event replay (m1-02 fixture) still equals golden with v2 input.
- Nav/crumb/title unit tests (DSA snapshot = v1; `zz-fixture` renders its labels); router ranking tests; `useCourse` states.
- e2e unchanged (**golden = v1**).
- **Literal gate** `hack/lint-course-literals.sh` (CI): no `"dsa"`, `'dsa'`, `` `/dsa ``, `/dsa/` in `internal/` or `web/src/`
  outside an allowlist: `internal/course/default.go` (`DefaultSlug`, task 1), `curriculum/dsa/**`, the frozen v1-envelope
  meaning in `internal/platform/events/envelope.go` (a platform package; it doesn't import `internal/course`), migrations,
  `testdata/`, tests. The alias rows and `?path=` defaults are **not** allowlisted: they use `course.DefaultSlug`. The SPA
  should need no default (a v1 `/dsa/…` URL is simply `/:course` = `dsa`; `PathSwitcher`'s and `Catalog`'s `dsa:` maps give
  way to the manifest view). If one does turn out to be needed, it goes in one exported constant whose file is allowlisted
  with a reason. Plus `hack/lint-dropped-columns.sh` (task 3).

## Acceptance criteria

- [ ] DSA UI and API identical to v1 through the new resolution (parity screenshots, alias parity test, e2e golden = v1).
- [ ] Unknown course → 404 (API `course_not_found`, SPA NotFound); a `coming_soon` course renders the AB02 card; `preview` and
      `retired` are invisible.
- [ ] All producers emit the v2 envelope; the `v1.6.0` consumers process them (replay golden equal).
- [ ] No query reads or writes an M1c-drop column except coach `is_default` (allowlisted for m1-10), and none upserts on
      `ON CONFLICT (account_id, week_of)`; both lint gates green in CI.
- [ ] Every course-scoped coach context is `<course>:<ctx>`; v1 forms from open tabs map to `dsa:<ctx>`; two courses' week 3
      get separate threads.
- [ ] Enrollment refuses any non-`active` course.
- [ ] `sqlc diff` clean; CI green; merged to `main`.

## Release

**Merge only — ships in `v1.7.0`**, which [m1-07](sprint-m1-07.md) tags once m1-03, m1-04, m1-05, m1-06 and m1-10 are merged
(rollback floor after `v1.7.0`: **1.6.0**). Nothing deploys at merge (`main` is build-only). Keep `main` taggable: every
merge is green and behaviour-identical for DSA.

## Definition of Done

CI green (incl. `sqlc diff`, OpenAPI drift, literal and dropped-column gates) · merged to `main` · parity screenshots in
the PR · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): board row, artboards
AB01–AB03 confirmed **frozen (ds-m1-01 PR #, date)** — ds-m1-01's merge records them under D40; repair if missing —
decisions log).

## Risks / watch-outs

- **A forgotten literal keeps a DSA-only path** — the literal gate is the guard; extend its allowlist only with a reason.
- **Open-tab skew:** `v1.6.0` tabs keep calling the unscoped routes and sending unprefixed contexts (`concept:<slug>`,
  `week:<n>`, `dashboard`, …) — aliases and the dual context parser must stay until the recorded removal tag.
- **One IUA commit bumps every service at once** — the `v1.6.0` consumers already decode v2, and the new internal `?path=`
  params default to `dsa`, so any mixed-version minute is safe.
- **Router ranking:** a dynamic `:course` segment must never shadow `settings`, `auth` or `u/…` — explicit tests.
- **Parity is visual:** compare screenshots side by side; AB02 must not introduce a DSA-visible change.
- **Serialized gateway edits:** m1-04 → m1-05 → m1-06 wait on this merge; land it in one PR and promptly.
