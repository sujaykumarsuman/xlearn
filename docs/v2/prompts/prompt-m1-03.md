# Prompt — Sprint m1-03 · Course resolution: gateway, BFF, SPA /:course/*, producers emit v2 (M1b)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-03.md`](../sprints/sprint-m1-03.md)   ·   **Milestone:** M1 (M1b)   ·   **Prereqs:** [m1-02](../sprints/sprint-m1-02.md) (`v1.6.0` live), [ds-m1-01](../sprints/sprint-ds-m1-01.md) (AB01–AB03 frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-m1-03.md`](../sprints/sprint-m1-03.md).
- [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §1 (manifest compiled in), §3 (every event gains `path_slug`), **§5 (routes, course-slug guard)**.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) **§1.1 (DSA aliases ≥ 1 release after the SPA stops calling them)**, §2 (`preview` hidden outside the cohort), §3 (envelope, consumers before producers).
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 8 (enrollment validation).
- [t0 §5 (envelope rule), §7 (screens & routing, API), §8 (curriculum's learner-safe view), §9 (M1b)](../research/t0-extensibility-frame.md);
  [t1 §4 curriculum/coach, *Expand → backfill → contract*, §9 Events and replay](../research/t1-content-data-model.md).
- [`../rollout-plan.md`](../rollout-plan.md) §4 M1 (scope T0), §9 (AB02, freeze rule).
- AB02: the frozen board `design-system/screens/v2/AB02-*.html` (see `design-system/screens/v2/index.html`) — static preview,
  never shipped; [`../../../design-system/README.md`](../../../design-system/README.md) + [`../../../design-system/theme.css`](../../../design-system/theme.css).
- [`../../architecture/api.md`](../../architecture/api.md), [`../../architecture/openapi.yaml`](../../architecture/openapi.yaml) (drift-tested), [`../../architecture/events.md`](../../architecture/events.md).
- Code: `internal/course/` (m1-01/m1-09), `internal/gateway/{bff.go,dashboard.go,progress.go,scheduling.go,coach.go,cache.go}`,
  `internal/curriculum/{service.go,handlers.go,seed.go}`, `internal/{practice,review,assessment}/handlers.go` (the internal
  routes gaining `?path=` / `path_slug`), `internal/{practice,review,assessment,identity}/store/store.go` (outbox writers),
  `internal/review/store/queries/weak_area_snapshot.sql` (`UpsertWeakAreaSnapshot`, `GetLatestWeakArea`),
  `internal/coach/handlers.go` (thread read + chat: context keys), `internal/coach/store/migrations/00001_init.sql`
  (`coach_thread UNIQUE (account_id, page_context)`), `internal/platform/events/envelope.go` (m1-02),
  `internal/identity/handlers.go` (`handleStartEnrollment`), `web/src/{router.tsx,nav.ts}`,
  `web/src/components/{Sidebar,PathSwitcher,AppShell,Coach}.tsx` (Coach: `useCoachContext` + `genericContext`),
  `web/src/screens/*.tsx`, `web/src/lib/*.ts`.

## Context

`v1.6.0` (m1-02) shipped the M1a expand: `path_slug` columns everywhere, `total`/`max_total`, `mock_session_item`,
coach `key_default`, identity role/status columns, and consumers that decode v1 and v2 envelopes. This sprint is the
bulk of **M1b**: DSA stops being special. The gateway resolves `{slug}` against the compiled manifests, BFF routes become
course-scoped under `/api/paths/{slug}/…` with DSA aliases for open tabs, the SPA routes `/:course/*` with a manifest nav,
producers emit v2, and readers/writers leave the columns M1c drops. DSA must look and behave **exactly** like v1. It
merges only; `v1.7.0` is cut by m1-07 after m1-04, m1-05, m1-06 and m1-10 also land. The gateway router is edited by
m1-03 → m1-04 → m1-05 → m1-06 in that order, so land this in one PR.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.6.0` live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz` reports it; ImagePolicies at `1.6.0`)
- [ ] AB01–AB03 frozen: ds-m1-01's design PR is merged (the merge is the freeze, D40)
- [ ] No open peer PR edits `internal/gateway/bff.go`'s route table or `web/src/router.tsx` (`gh pr list`, `git worktree list`, ListAgents)

## Do this (in order)

1. **[X] Branch** `feat/m1b-course-resolution` from an up-to-date `main`.
2. **[X] Course view + curriculum** (plan task 2): `internal/course/view.go` `LearnerView()`; curriculum `GET /paths/{slug}/concepts/{c}`
   (keep `GET /concepts/{slug}` as the DSA alias); `course` block on `GET /paths` and `GET /paths/{slug}`; readers on `role`,
   `status`, `links`, `templates`, `language` with the v1 JSON fields derived; the loader stops writing the four M1c-drop columns.
3. **[X] Gateway resolution** (plan task 1): `internal/gateway/course.go` (`resolveCourse`, `courseVisible` — `preview` 404 for
   everyone until m1-04); new `/api/paths/{slug}/{dashboard,progress,revision/due,mistakes,weak-area,mocks,mocks/trend,concepts/{c}}`
   in `apiRoutes()` **and** `openapi.yaml`; DSA aliases through the same handlers (`Alias` field, `deprecated: true`); replace the
   `"dsa"` sites in `bff.go`, `scheduling.go`, `dashboard.go`, `progress.go`; pass `?path=`/`path_slug` to review, assessment and
   practice (their internal routes in `handlers.go` accept it, defaulting to `course.DefaultSlug`); slug in every cache name.
   Add `internal/course/default.go` (`const DefaultSlug = "dsa"`) and use it at **every** mixed-version default and alias
   site (gateway aliases, internal `?path=` defaults, curriculum's `/concepts/{slug}` alias, coach's legacy-context parser),
   never a bare `"dsa"`. Record the `/api/paths/{slug}` prefix decision.
4. **[X] Producers v2 + drop-column exit** (plan task 3): every outbox writer uses `events.NewEnvelope(2, …)` (course-scoped
   subjects carry `path_slug`; `account_created` doesn't); `mock_completed` data adds `rubric_id`, `total`, `max_total`, `scored_by`
   and keeps `total_35`; every writer sets `path_slug` explicitly; assessment stops writing/reading `total_35`. review's
   `UpsertWeakAreaSnapshot` moves to `ON CONFLICT (account_id, path_slug, week_of)` (m1-02's `weak_area_snapshot_account_path_week_uq`)
   and inserts `path_slug`; the weekly recompute runs per `(account_id, path_slug)`; `GetLatestWeakArea` filters by course. Add
   `hack/lint-dropped-columns.sh` to CI: the five drop-list columns **and** `ON CONFLICT (account_id, week_of)`; coach
   `is_default` allowlisted for m1-10.
5. **[X] SPA** (plan task 4, AB02): `/:course/*` router subtree, `useCourse()` + `coursePath()`, manifest `nav.ts`, generic crumbs
   and titles, `Sidebar`/`PathSwitcher`/`AppShell`/`NotFound`, the AB02 coming-soon card, every `/dsa` literal in screens and
   `lib/*.ts` replaced, onboarding path picker from the catalog, `courseSlugGuard.ts` + parity test. `theme.css` verbatim.
6. **[X] Coach** (plan task 5): **every course-scoped context** gets the prefix (t0 §7): `${course}:concept:${slug}`,
   `${course}:week:${n}`, `${course}:{roadmap,dashboard,revision,mistakes,mock,progress}`; `problem:<id>`, `catalog`, `settings`,
   `general` unchanged. One shared dual parser in coach + gateway maps the v1 forms (open `v1.6.0` tabs) to
   `course.DefaultSlug` + `:` + context. Write `path_slug` on threads/messages (`NULL` for account-wide). The idempotent coach
   data migration: `'dsa'` backfill and the `dsa:` rewrite for `concept:%`, `week:%` and the six bare words (guarded so a
   re-run is a no-op). Tests: parser table, migration twice, `dsa` vs `zz-fixture` week 3 → separate threads.
7. **[X] Enrollment** (plan task 6): identity + gateway start-path refuse unknown/`preview` (404) and `coming_soon`/`retired` (409).
8. **[X] Tests + gates** (plan task 7): fixture manifests (`zz-fixture` active, plus `coming_soon`/`preview`/`retired`) via a
   test-only registry; per-course route tests; alias byte-parity; producer v2 tests + replay golden; nav/router/`useCourse`
   tests; `hack/lint-course-literals.sh` in CI (allowlist: `internal/course/default.go`, `curriculum/dsa/**`, the v1 meaning in
   `envelope.go`, migrations, `testdata/`, tests — alias rows and `?path=` defaults use `course.DefaultSlug`, not a literal). Then `gofmt`, `go vet`, `go test -race ./...`, `sqlc generate` + **`sqlc diff`**,
   web typecheck/lint/test/build, `-tags e2e` (golden = v1).
9. **[X] DSA parity:** `docker compose up` on `v1.6.0` images vs this branch; screenshot every DSA screen at 1440 px and 390 px;
   attach to the PR; fix any unintended diff.
10. **[X] PR** → conventional commits (`feat(m1b): …`) with the attribution lines → CI green → squash-merge (see Ship). **No tag.**

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the gateway composes and gates only;
  curriculum owns content and the learner view; each service writes its own `path_slug`. No cross-schema reads.
- **goose + sqlc:** the coach data migration is its own file at the next free version, idempotent, no contract statement
  (the m1-02 migration lint must pass); **`sqlc diff`** clean.
- **Envelope append-only, decoders forever**; payload fields are only ever added (`total_35` stays in `mock_completed` data).
- **API `/api/v1` additive only;** aliases stay until the recorded removal tag (≥ 1 release after the SPA stops calling them).
- **Frontend:** `theme.css` tokens/components verbatim (no Tailwind, no new tokens), dark theme, match AB02; difficulty tokens
  Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`.
- **No behaviour change for DSA;** golden = v1.
- **GitOps:** nothing to deploy here (merge only); never `kubectl apply`. **D34:** no alerting.
- **Parallel sessions:** check peers' PRs/worktrees before editing the serialized gateway files; claim ADR numbers only after
  checking peers.

## Deliverables

- `internal/gateway/course.go`, `internal/course/default.go` (`DefaultSlug`), course-scoped routes + aliases (+ `openapi.yaml`),
  literal sites replaced, slugged cache names.
- `internal/course/view.go`; curriculum concept route + course block; readers on new columns; loader writes only new columns.
- Producers on v2 envelopes; review's weak-area upsert on the per-course unique; `hack/lint-dropped-columns.sh`.
- SPA `/:course/*`, `useCourse`, manifest nav, AB02 states, `courseSlugGuard.ts`.
- Coach context prefix on every course-scoped context + dual parser + `path_slug` + data migration.
- Enrollment validation.
- Fixture manifests + tests; `hack/lint-course-literals.sh`; parity screenshots in the PR.

## Update status

- This plan's Status table ([`../sprints/sprint-m1-03.md`](../sprints/sprint-m1-03.md)): tasks 🔄 → ✅; _Overall_ ✅.
- [`../status.md`](../status.md): Sprint board row (m1-03 ✅, "merged, ships in v1.7.0"); **confirm artboards AB01–AB03 read
  "frozen (ds-m1-01 PR #, date)"** (ds-m1-01's merge records them under D40; repair if missing; this is the first build sprint
  consuming them); decisions log: the `/api/paths/{slug}` prefix, the alias
  list and its earliest removal tag, `course.DefaultSlug` as the one mixed-version default, the coach context rewrite for every
  course-scoped context (t0 §7, beyond t1 §4's concept-only line), the weak-area upsert on the new unique (m1-08 may drop the v1
  one), `mock_completed` keeping `total_35` in data.
- Record an ADR only for a call beyond ADR-0026/0034 (check peers' ADR numbers first).

## Done when (acceptance)

- [ ] DSA UI and API identical to v1 through the new resolution (screenshots, alias parity, e2e golden = v1).
- [ ] Unknown course → 404; `coming_soon` renders the AB02 card; `preview`/`retired` invisible.
- [ ] All producers emit v2; the `v1.6.0` consumers process them (replay golden equal).
- [ ] No query touches an M1c-drop column except coach `is_default`, and none upserts on `(account_id, week_of)`; literal and
      dropped-column gates green.
- [ ] Every course-scoped coach context is `<course>:<ctx>`; v1 forms map to `dsa:<ctx>`.
- [ ] Enrollment refuses any non-`active` course.
- [ ] CI green (`sqlc diff`, OpenAPI drift); merged to `main`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1b-course-resolution`, then conventional commit(s) with the attribution lines, then push, then the PR (one PR: the gateway router is serialized). This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships in **`v1.7.0`**, which [m1-07](../sprints/sprint-m1-07.md) tags once m1-04, m1-05, m1-06 and m1-10 have merged too. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
