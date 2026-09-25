# Prompt — Sprint p-02 · Multi-course: go-concurrency manifest (preview), catalog, agenda, nav

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-p-02.md`](../sprints/sprint-p-02.md)   ·   **Milestone:** P   ·   **Prereqs:** [p-01](../sprints/sprint-p-01.md) (runner-v1.1.0, `gotest@1`), [ds-p-01](../sprints/sprint-ds-p-01.md) (Q5 + frozen boards)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, land-and-sync.
- The plan: [`../sprints/sprint-p-02.md`](../sprints/sprint-p-02.md) — the manifest table (task 1), the item list (task 2)
  and the `preview` matrix (task 3) are the brief.
- Frozen boards: `design-system/screens/v2/AB02-course-nav.html` and `AB05-catalog-agenda.html` (the **"P · full
  fidelity"** sections), `AB15-go-concurrency-race.html` (F1–F2, F16); [`../../../design-system/theme.css`](../../../design-system/theme.css) verbatim.
- Course model: [ADR-0026](../../adr/0026-per-course-extensibility-model.md) (§1, §4, §5),
  [t0 §6](../research/t0-extensibility-frame.md#6-six-course-fit), [PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses),
  [PRD §7 Q5/Q7](../../prd/xlearn-v2-prd.md#7-open-questions-routed-to-topics).
- Content: [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery),
  [§7.1](../research/t1-content-data-model.md#71-package-format), [§7.2](../research/t1-content-data-model.md#72-ci-validation),
  [§8](../research/t1-content-data-model.md#8-content-rights-stance).
- Grading and touches: [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide), [§4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key),
  [§5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts), [§5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start),
  [§6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds),
  [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only), [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course);
  D3, D4, D7, D16, D18, D27 in the [decisions log](../feasibility.md#decisions-log-newest-first).
- Gating: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
  (`preview` everywhere, T-2 envs, the flag inventory rule) and [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules);
  [ADR-0033 §12–§13](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2); [rollout §10](../rollout-plan.md#10-public-dashboard-tasks).
- The sprints you build on: [m1-01](../sprints/sprint-m1-01.md) (manifest blocks + validator, coming-soon stubs),
  [m1-09](../sprints/sprint-m1-09.md) (loader, id guard, the `path.status` hand-off), [m1-02](../sprints/sprint-m1-02.md)
  (migration lint + relax marker), [m1-03](../sprints/sprint-m1-03.md) (`courseVisible`, `useCourse()`, routes),
  [m1-04](../sprints/sprint-m1-04.md) task 7 (cohort `preview`), [m1-05](../sprints/sprint-m1-05.md) / [m2-03](../sprints/sprint-m2-03.md)
  (public surfaces), [m2-04](../sprints/sprint-m2-04.md) (catalog, agenda, planner), [m3-01](../sprints/sprint-m3-01.md)
  (authoring guide), [m3-08](../sprints/sprint-m3-08.md) (strategies), [m3-11](../sprints/sprint-m3-11.md) (Workspace-Code),
  [p-01](../sprints/sprint-p-01.md) (the `module` block, `gotest@1`).
- Code: `curriculum/courses/`, `curriculum/ids.lock.json`, `curriculum/paths.json`, `internal/course/`,
  `internal/curriculum/store/migrations/`, `internal/gateway/` (course resolution, public, agenda), `internal/identity/`
  (enrollment), `web/src/screens/{Catalog,Dashboard,Roadmap,Week,Problems,Revision,Workspace}.tsx`,
  `web/src/components/{PathSwitcher,Sidebar,Topbar}.tsx`, `web/src/nav.ts`, m3-11's
  `web/src/screens/workspace/{CodeEditor,EditorPane}.tsx` and `web/src/lib/judge.ts` (`serialize()`, drafts), and
  `web/src/parts/` (**new in this sprint**: no part registry exists before it), `docs/v2/authoring.md`.

## Context

Milestone **P** proves the extensibility model: a second course that ships **as manifest + content, with widget/profile
code only**. [p-01](../sprints/sprint-p-01.md) gave the runner the `go-race@1.26` profile and the `gotest@1` harness
(`runner-v1.1.0`, live dark) and judge the honor-grade mapping; the item schema gained the `module` block. This sprint
turns go-concurrency's `coming_soon` stub into a full **`preview`** course — visible and enrollable only for the
owner/tester cohort (`account.role` from session-validate, never the JWT) and absent from every other surface, public
ones included — adds ~10 item scaffolds for the owner to finish, a T-2 runtime override for course status, the
multi-course catalog/agenda/nav at full fidelity, and file tabs for multi-file Go modules in Workspace-Code. It merges
dark into `v1.15.0`, which [p-03](../sprints/sprint-p-03.md) tags after adding the quiz widget, the race verdict views
and the pilot pack. At GA the manifest flips to `active` ([ga-01](../sprints/sprint-ga-01.md)).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB14–AB15 (and AB02/AB05 full-fidelity) frozen — the ds-p-01 board PR is merged (the merge is the freeze)
- [ ] PRD Q5 confirmed (ds-p-01) — go-concurrency
- [ ] p-01 merged and `runner-v1.1.0` live dark (the `module` schema + judge `gotest@1` mapping on `main`)
- [ ] The M1/M2 `preview` plumbing is on `main` (m1-03 `courseVisible`, m1-04 task 7, m1-05/m2-03 public hiding, m2-04 catalog/agenda)
- [ ] practice grades key-only items with `weighted_gate@1` in a course whose strategy is `verdict_timer@1` (m3-08) — if not, **stop and report** (an M3 gap, not pilot code)
- [ ] Parallel sessions: no open peer PR on `curriculum/courses/`, `internal/course/`, the Catalog/Dashboard/Workspace screens, `web/src/screens/workspace/`, `web/src/lib/judge.ts`, `web/src/parts/` or PathSwitcher/Sidebar (`gh pr list`, `git worktree list`, ListAgents)

## Do this (in order)

1. **[X] Branch** `feat/p-02-go-concurrency-preview` from an up-to-date `main`.
2. **[X] Manifest** (plan task 1): replace the stub `curriculum/courses/go-concurrency/course.json` with the full
   `preview` manifest per the plan's table (nav without Mock, D18 timers, `verdict_timer@1` + `weighted_gate@1` params,
   `self_report {outcome, touch, mock}` all `allowed` (m1-01's shape), L1–3 recall band with criterion key
   `recall_correct` (pass = s ≥ `pass_pct` ∧ every required probe correct, t4 §6.6) / L4–5 re-solve band with
   `correct_in_timer`, categories + prefill, no mock, `est_minutes`, D7 visibility,
   coach persona ≤ 600 chars). Run `course.Load`; add the method-values table test. Add `phases.json`, `weeks.json`,
   `concepts.json` + ≈ 8 original `concepts/*.md`; append `gc-001…gc-010` to `ids.lock.json`; update the `paths.json`
   row if it still carries status (`preview`, 10 items, 2 weeks).
3. **[X] Widen `path.status`** (curriculum, next free goose version): drop + re-add the CHECK as
   `('active','preview','coming_soon','retired')` in one statement; confirm the constraint name with `\d+ curriculum.path`
   in compose; put m1-02's relaxation marker on the `DROP CONSTRAINT` line (never a contract marker); store test for
   every old value + `preview`; `sqlc generate`, `sqlc diff` clean; `hack/lint-migrations.sh` green.
4. **[X] Scaffolds** (plan task 2): the ten items in the plan's table. Code items: `item.json` (code part with
   `harness: "gotest@1"` and `module{path, go, files[], api[]}`, the `tests` step, `concepts`, `provenance original /
   ai-assisted`, two pack-keyed L1–3 probes with `criterion: recall_correct`), `_starter/module/{go.mod.tmpl, *.go, visible_test.go}` (visible tests fail on
   the starter), `_code/module/…` (reference passes the visible tests in the runner image), `sections/attempt/01-statement.md`
   (original, lists the exported API). Quiz items: `choice`/`blank` parts, `cadence final`, a `key` step, role `drill`,
   **no correctness in public files**. No hints/editorial (stamp gate). Nothing copied from Go docs or examples.
5. **[X] `preview` everywhere** (plan task 3): write the matrix tests with the **real** embedded manifest — catalog,
   every `/api/paths/go-concurrency/…` route (route enumeration), `GET /api/problems/gc-001` and every
   `/api/problems/{id}/…` sub-route, identity enrollment, agenda/dashboards/revision all-courses, `/u/<name>` +
   `/public/stats`, coach contexts, the SPA route — for anonymous, `learner`, `tester`, `owner`. Non-cohort 404 bodies
   must equal an unknown slug's/id's. Fix any surface that leaks (it's a platform bug from M1–M3: note it in the Decisions log).
6. **[X] `COURSE_STATUS_OVERRIDE`** in `internal/course` (applied in `Load`): `slug=status[,…]`, statuses `active |
   preview | coming_soon`; unknown slug/status, `retired` and `dsa` ignored with one ERROR log; effective overrides
   logged at start; tests. Don't set it anywhere in infra.
7. **[X] Multi-course UI** (plan task 4) to AB02-P1–P7 and AB05-P1–P7: Catalog + agenda (Preview badge, honor note,
   minutes, "nothing fits", per-course R-SR5, 7-day toggle), nav/Sidebar/PathSwitcher/Topbar from the go-concurrency
   manifest (no Mock), item-noun labels and crumbs, Revision badges for go touches (the recall-quiz body is p-03's).
8. **[X] Part registry + multi-file workspace** (plan task 5).
   - **Create** `web/src/parts/registry.ts`: a closed `PartType → PartWidget` map in one file, no self-registration,
     t4 §5.1. p-03 adds `choice`/`blank` to it.
   - Create `web/src/parts/code/`: the `code` widget implementing t4 §5.5's `PartWidget` by **wrapping** m3-11's editor.
     `load()` is the existing lazy `CodeEditor` chunk. `empty()` keeps m3-11's single-file starter and builds
     `{language: "go", files[]}` for `gotest@1`. `serialize`/`validate` take over the pre-send size checks from
     `judge.ts` `serialize()`. `draft` is 15 s / 25 s.
   - `EditorPane` renders the code part through the registry. DSA single-file behaviour is unchanged (m3-11's tests stay green).
   - For `gotest@1`: file tabs + tree, editable vs read-only (`xl-lock`), no add/rename/delete, per-file CodeMirror.
   - Drafts: the draft body is the whole `{language, files[]}` value under m3-05's existing (context, part, language)
     key, with no server change.
   - The 64 KiB guard with AB08's 413 copy, **Run visible tests** (⌘↵) / **Submit** (⌘⇧↵), `module.api[]` in the
     statement panel, and the < 1024 px Statement / Files / Results tabs with a file `<select>`.
9. **[X] Tests** (plan task 6): Go (manifest, id guard, content CI, CHECK widening, matrix, override), web (one vitest per
   AB02-P*/AB05-P* frame; the multi-file widget), compose e2e (owner enrolls, opens `gc-001`, sees locked read-only
   files, Runs visible tests where the compose runner exists; a `learner` gets 404 everywhere and `/u/<owner>` has no pilot row).
10. **[X] Authoring prep** (plan task 7): the go-concurrency (`gotest@1`) section in `docs/v2/authoring.md`; the content
    rows and `ev-pilot-content` "ready" in `docs/v2/status.md`.
11. **[X] Verify:** `gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` · migration lint · content CI ·
    `npm --prefix web run typecheck && npm --prefix web run lint && npm --prefix web test && npm --prefix web run build` ·
    openapi drift · route enumeration · e2e · `docker compose up --build` click-through as owner and as a learner.
12. **[X] Record** (plan task 8), then ship: see **Ship** below.

## Constraints

- **Widget/profile code only (P exit).** Pilot content is data; the only code is the part registry and the multi-file
  `code` widget, the multi-course UI, the override and tests. Don't add course-specific branches (`if slug == "go-concurrency"`) anywhere — the literal
  lint from m1-03 applies; don't add practice/review/judge rules for the pilot (a gap there is an M3 follow-up).
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** curriculum owns content and
  `path`; identity owns enrollment and roles; the gateway composes and filters; no cross-schema reads; the cohort bit
  comes from session-validate, never the JWT.
- **goose + sqlc:** one expand-safe curriculum migration with the relaxation marker (no contract marker, no floor);
  embedded; generated code committed; `sqlc diff` clean.
- **Outbox/inbox and events:** unchanged; the first non-DSA `path_slug` events need consumers ≥ 1.7.0 (satisfied).
- **Private data:** public files carry no answers (the answer-free schema test); probe and quiz keys and hidden tests
  belong in the pack (p-03). Never copy content from `../xlearn-evalpack`.
- **Frontend:** `theme.css` verbatim, dark theme, no Tailwind, match the frozen boards; difficulty Easy=`--ds-ok`,
  Medium=`--ds-warn`, Hard=`--ds-err`; the SPA never filters `preview` itself.
- **GitOps / infra:** no infra PR, no `kubectl`; `COURSE_STATUS_OVERRIDE` stays unset. No new pod, stream, subject or
  in-cluster caller, so the memory-sum rule, ACL-before-consumer and consumers-before-producers aren't triggered.
- **No alerting (D34).**
- **Parallel sessions:** check peers' PRs, tags and worktrees before merging and before claiming any ADR number.
- **No tag in this sprint** (`v1.15.0` is p-03's).

## Deliverables

- `curriculum/courses/go-concurrency/` (manifest `preview`, phases, weeks, concepts, `items/gc-001…gc-010`), `ids.lock.json`,
  the `paths.json` row if applicable.
- The curriculum `path.status` widening migration + store test.
- The real-manifest `preview` matrix tests; `COURSE_STATUS_OVERRIDE` in `internal/course` + tests.
- Catalog/agenda/nav at AB02/AB05 full fidelity; `web/src/parts/registry.ts` + the `code` widget (`web/src/parts/code/`,
  wrapping m3-11's editor, multi-file for `gotest@1`); vitest + e2e.
- The go-concurrency section of `docs/v2/authoring.md`; status.md rows.

## Update status

- [`../sprints/sprint-p-02.md`](../sprints/sprint-p-02.md): task rows 🔄 → ✅ (⛔ with a reason), _Overall_.
- [`../status.md`](../status.md): the **Sprint board** row; **Milestones** P 🔄; the **flag inventory** row for
  `COURSE_STATUS_OVERRIDE` (T-2 · owning P · default unset · every `xlearn-*` release · removal GA, with the recommendation
  to promote it to a permanent operating mode at ga-01); **content status** go-concurrency rows (10 scaffolded) and
  `ev-pilot-content` ready; **Artboards** AB02/AB05 (full) → consumed by p-02; **Decisions log** (manifest values as
  drafted, revisable by the owner with a content PR, the status-CHECK relaxation, override semantics, no mock, any `preview` leak fixed, the part registry
  created here with the `code` widget wrapping m3-11's editor).
- ADR only for a departure from ADR-0026/ADR-0034 (none expected). Check peers before numbering.

## Done when (acceptance)

- [ ] A non-cohort account sees no trace of the pilot (catalog, profile, stats)
- [ ] Owner can enroll and open a gc item
- [ ] The manifest validates as `preview` with every method block; `gc-001…gc-010` seed; content CI green
- [ ] The §3 matrix passes for anonymous/`learner` vs `tester`/`owner` with the real manifest
- [ ] `COURSE_STATUS_OVERRIDE` works via `course.Load`, ignores bad input with an ERROR log, and is in the flag inventory (owning P, removal GA)
- [ ] `path.status` accepts `preview`/`retired` (relax marker, no contract marker, store test, `sqlc diff` clean)
- [ ] Catalog, agenda and nav match AB02-P*/AB05-P*; the multi-file workspace matches AB15 F1–F2
- [ ] CI green (Go, sqlc, content, web, openapi drift, route enumeration, e2e)

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). (Branch `feat/p-02-go-concurrency-preview`; this repo only.)
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships in `v1.15.0`):** Nothing deploys; it ships in `v1.15.0` (cut by [p-03](../sprints/sprint-p-03.md)). Don't tag. No infra PR: `COURSE_STATUS_OVERRIDE` stays unset.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
