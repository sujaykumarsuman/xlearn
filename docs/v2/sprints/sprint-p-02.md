# Sprint p-02 — Multi-course: go-concurrency manifest (preview), catalog, agenda, nav

> **Milestone:** P — pilot course (go-concurrency)   ·   **Track:** product (order 58)
> **Prereqs:** [p-01](sprint-p-01.md) (`gotest@1` + the `module` item schema on `main`; `runner-v1.1.0` live dark) · boards from [ds-p-01](sprint-ds-p-01.md) (AB02/AB05 full fidelity, AB15 F1–F2)
> **Unblocks:** [p-03](sprint-p-03.md) (quiz widget, race verdict UI, pilot pack, `v1.15.0`)
> **Release action:** merge only (ships in **v1.15.0**, tagged by [p-03](sprint-p-03.md)). No infra PR: `COURSE_STATUS_OVERRIDE` stays unset.
> **Calendar:** December. Prepares owner event `ev-pilot-content` (December, ~10 items, 10–20 owner h).
> **Execute with:** [`../prompts/prompt-p-02.md`](../prompts/prompt-p-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Course manifest `preview` + pilot structure + `path.status` CHECK widening | X | ⬜ |
| 2 | Item scaffolds `gc-001…gc-010` with public starter modules | X | ⬜ |
| 3 | `preview` hidden everywhere outside the cohort (real-manifest test matrix) + `COURSE_STATUS_OVERRIDE` (T-2) | X | ⬜ |
| 4 | Multi-course UI: catalog, agenda, nav at full fidelity (AB02/AB05 full) | X | ⬜ |
| 5 | Part registry + `code` widget (wrapping m3-11's editor); multi-file workspace: file tabs + editable list for `gotest@1` items (AB15 F1–F2) | X | ⬜ |
| 6 | Tests + compose e2e | X | ⬜ |
| 7 | Prepare `ev-pilot-content` (the owner authors after ship; doesn't gate _Overall_ ✅): authoring guide section, content-status rows | X | ⬜ |
| 8 | Record (status.md: flag inventory, content, decisions) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> milestone P, flag inventory, content status). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] AB14–AB15 (and AB02/AB05 full-fidelity) frozen: [ds-p-01](sprint-ds-p-01.md) merged (the merge is the freeze)
- [ ] PRD Q5 confirmed (ds-p-01)
- [ ] [p-01](sprint-p-01.md) merged and `runner-v1.1.0` live dark: judge maps `gotest@1`; the `module` block is in `curriculum/_schema/item.schema.json` and `internal/course`
- [ ] The M1/M2 `preview` plumbing is on `main`: m1-03's `courseVisible(m, cohort)`, m1-04 task 7 (cohort enrollment + visibility), m1-05 / m2-03 (public profile and `/public/stats` never show `preview`), m2-04's multi-course catalog/agenda
- [ ] **Strategy per item archetype:** on `main`, practice grades key-only (quiz) items with `weighted_gate@1` and code items with the course strategy ([m3-08](sprint-m3-08.md)). If practice can run only one strategy per course, stop and report: that's practice code (an M3 follow-up with an ADR), not pilot content — P ships widget/profile code only
- [ ] Parallel sessions: no open peer PR on `curriculum/courses/`, `internal/course/`, `web/src/screens/{Catalog,Dashboard,Workspace}.tsx`, `web/src/screens/workspace/`, `web/src/lib/judge.ts`, `web/src/parts/` or `web/src/components/{PathSwitcher,Sidebar}.tsx`

## Goal

Ship the second course **as manifest + content with widget/profile code only**: `curriculum/courses/go-concurrency/`
flips from its `coming_soon` stub to a full **`preview`** manifest with ~10 item scaffolds (public starter modules,
visible tests, statement drafts), hidden from everyone outside the owner/tester cohort **everywhere** — catalog,
enrollment, course and item routes, Today/agenda, public profile and `/public/stats` — with a runtime override
(`COURSE_STATUS_OVERRIDE`, T-2) recorded in the flag inventory. Bring the catalog, agenda and nav to AB02/AB05 full
fidelity with two real courses, and give Workspace-Code file tabs for multi-file `gotest@1` items. The owner then authors
and stamps the content (`ev-pilot-content`); [p-03](sprint-p-03.md) adds the quiz widget, the race verdict views and the
pack, and tags `v1.15.0` with the course still `preview`.

## Scope

**In**
- `curriculum/courses/go-concurrency/course.json` (status `preview`, `id_prefix` `gc`) with every method block;
  `phases.json`, `weeks.json`, `concepts.json` + `concepts/*.md`; `ids.lock.json` entries; the `paths.json` row if it
  still carries status (m1-01's consistency test).
- The curriculum migration widening `path.status` to `preview`/`retired` (handed here by [m1-09](sprint-m1-09.md)).
- Item scaffolds `gc-001…gc-010` (7 code, 3 quiz), public halves only.
- The real-manifest `preview` test matrix and `COURSE_STATUS_OVERRIDE` in `internal/course`.
- `web/`: Catalog, agenda, course Dashboard, PathSwitcher/Sidebar/Topbar/nav, Roadmap/Problems labels (AB02-P*, AB05-P*);
  the new part registry (`web/src/parts/registry.ts`) with the `code` widget (`web/src/parts/code/`) wrapping m3-11's
  editor, and the multi-file editor in Workspace-Code (AB15 F1–F2).
- The go-concurrency section of the authoring guide and the content-status rows.

**Out**
- Quiz widget A7 (course items and recall touches, AB14) and the race / deadlock / leak / dump result views (AB15 F3–F16) → [p-03](sprint-p-03.md).
- Hidden tests, pack keys for the probes and quizzes, go-race params → the pilot pack (E) in [p-03](sprint-p-03.md).
- Content hours: final statements, starters, visible tests, stamps → the owner (`ev-pilot-content`).
- Any runner or judge change → done in [p-01](sprint-p-01.md); a missing piece there is a p-01 follow-up, not pilot code.
- Flipping the course to `active` → GA ([ga-01](sprint-ga-01.md)); setting `COURSE_STATUS_OVERRIDE` on prod → an infra PR only if ever needed.
- A mock for the pilot (the manifest has none).

## Tasks

### 1 · Course manifest + pilot structure [X]

Sources: [ADR-0026 §1, §4](../../adr/0026-per-course-extensibility-model.md#4-universal-vs-per-course-method),
[m1-01 task 2](sprint-m1-01.md) (manifest blocks, validation), [t4 §6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds),
[§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only), [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course),
[t0 §6](../research/t0-extensibility-frame.md#6-six-course-fit), D3, D4, D7, D16, D18, [PRD §7 Q7](../../prd/xlearn-v2-prd.md#7-open-questions-routed-to-topics) (45 min / hint at 15 as the default).

`curriculum/courses/go-concurrency/course.json` — proposed values; they land as drafted (D40), and the owner can revise
them later with a content PR (course design is data, and `course.Load`'s validator is the arbiter of shape):

| Block | Value |
|---|---|
| identity | `format 1`, `slug go-concurrency`, `title "Go Concurrency Patterns"`, **`status preview`**, `id_prefix gc` (m1-01's proposed prefix, locked by `path.id_prefix`) |
| `nav` | `item_noun "exercise"`; Today · Roadmap · Exercises · Progress / cap "Practice loop": Revision · Mistakes — **no Mock** |
| `stages` / timers | the D18 defaults (PRD Q7): attempt 45:00, hint at 15:00, `reimplement: off` (D16); same `verdict_timer@1` params as DSA (`clean_within_s 1200`, `max_failed_for_clean 3`, `free_classes [CE, REJECTED]`) |
| `grading` | `strategy verdict_timer@1` (code items); `params["weighted_gate@1"] = {clean_pct 1.0, pass_pct 0.8}` (quiz items and the L1–3 recall band, t4 §6.2); `self_report {outcome: allowed, touch: allowed, mock: allowed}` (m1-01's shape; T-1: items without a stamped pack stay on the self path; `mock` is moot, since the course has no mock) |
| `revision.bands` | **L1–3** `format recall`, label "Recall", `timer_s 300`, `parts []`, `criteria [{key: recall_correct}]` (every recall probe of a gc item sets `criterion: recall_correct`; pass per [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course): s ≥ `pass_pct` (the `weighted_gate@1` param, 0.8) **and** every required probe correct; `checked`, pack keys), `mock_mode false`; **L4–5** `format resolve`, label "Re-solve", `timer_s 1500`, `parts ["solution"]`, `criteria [{key: correct_in_timer}]` (a passing touch submit ≤ timer, honor), `mock_mode true`; `drills false` (quiz drills don't schedule, and per I8 they open no mistake entries either) |
| `mistakes` | core `{misread, time_management, communication}` + `data_race`, `deadlock`, `goroutine_leak`, `wrong_primitive`, `missed_cancellation`, `unbounded_concurrency`; `prefill`: `race → data_race` (strong), `deadlock → deadlock` (strong), `leak → goroutine_leak` (strong), `misconception:<cat>` → `<cat>` (strong), `late → time_management` (strong), `sample_failed → misread` (weak) |
| `mock` | absent (a course may have no mock, ADR-0026 §4) |
| `plan.est_minutes` | `course_attempt 45`, `touch_recall 5`, `touch_resolve 25` (D4) |
| `public_stats` | `default_visible true` (D7) — moot while `preview` (never public), effective after GA |
| `coach` | `persona` ≤ 600 chars (Socratic Go-concurrency coach: asks about happens-before, ownership of channels, who closes what, cancellation paths; never writes the fix; reads race-detector output with the learner); `primary_language go`; `off_during [touch, mock]` |

- Every criterion key maps to a step check (lint #14 of [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start));
  every `prefill.category` is declared. A table test pins the manifest's method values so changes are deliberate.
- **Structure:** `phases.json` (one phase "Pilot"), `weeks.json` (week 1 "Goroutines, channels, races"; week 2
  "Cancellation, leaks, limits"), `concepts.json` + `concepts/<slug>.md` (≈ 8 original concept pages: goroutines and
  WaitGroups, channel ownership and closing, select and timeouts, context cancellation, mutexes vs atomics, data races
  and happens-before, deadlocks and lock ordering, goroutine leaks), Markdown profile per m1-09.
- `curriculum/ids.lock.json`: append `gc-001…gc-010` (append-only; later removals are retirements).
- `curriculum/paths.json` (only if m1-09's loader hasn't taken over status): `status preview`, `problem_total 10`,
  `week_total 2` so the catalog doesn't promise the full 60-item course.
- **`path.status` CHECK widening** (curriculum, next free goose version at rebase): drop and re-add the status CHECK as
  `status IN ('active','preview','coming_soon','retired')` in one statement, the `DROP CONSTRAINT` line carrying
  m1-02's reviewed-relaxation marker, `-- xlearn:relax widen path.status for preview/retired (m1-09 hand-off)` (m1-02's
  grammar defines only `xlearn:relax` and `xlearn:contract`). It is **not** a contract marker: a widening sets no
  floor. Confirm the
  constraint name with `\d+ curriculum.path` in compose; a store test inserts every pre-existing value and `preview`;
  `sqlc diff` clean.

### 2 · Item scaffolds `gc-001…gc-010` [X]

Sources: [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (item.json, go-concurrency row),
[§7.2](../research/t1-content-data-model.md#72-ci-validation) (stamp gate, leak lints), [§8](../research/t1-content-data-model.md#8-content-rights-stance),
[t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key), [p-01 task 3](sprint-p-01.md) (the `module` block).

Proposed list (the owner edits titles, order and mix in `ev-pilot-content`):

| Id | Title | Kind | Role | Week | Concepts (primary first) |
|---|---|---|---|---|---|
| gc-001 | Bounded worker pool | code (implement) | core | 1 | worker-pools, goroutines-and-waitgroups |
| gc-002 | Fix the racy cache | code (fix) | core | 1 | data-races, mutexes-vs-atomics |
| gc-003 | Pipeline stages that stop cleanly | code (implement) | core | 1 | channel-ownership, context-cancellation |
| gc-004 | Merge channels without leaks | code (implement) | core | 1 | goroutine-leaks, channel-ownership |
| gc-005 | Fix the transfer deadlock | code (fix) | core | 1 | deadlocks-and-lock-ordering |
| gc-006 | A token-bucket limiter | code (implement, `testing/synctest`) | core | 2 | select-and-timeouts |
| gc-007 | Cancel the slow fan-out | code (implement) | core | 2 | context-cancellation |
| gc-008 | Spot the race | quiz (`choice.single`/`multi`) | drill | 2 | data-races |
| gc-009 | Predict the output | quiz (`blank.predict_output`) | drill | 2 | select-and-timeouts |
| gc-010 | Pick the primitive | quiz (`choice.single`, `choice.order`) | drill | 2 | mutexes-vs-atomics |

- **Code items:** `item.json` with `parts: [{id: solution, type: code, cadence: iterate, grading: auto, config:
  {harness: "gotest@1", languages: ["go"], module: {path, go: "1.26", files: [...], api: [...]}, limits}}]`,
  `grader: [{step: tests, kind: code, inputs: [solution], required: true}]`, `concepts[]`, `provenance {origin:
  original, authored_by: ai-assisted}`, `status live`, and **2 recall probes** (`revision.probes[]`: `band` L1–3,
  `grading: key`, `key_source: pack`, **`criterion: recall_correct`**, e.g. `p-race` choice "Which line races?",
  `p-output` `blank.predict_output`). Their keys live in the pack (p-03). Lint #14 checks that every criterion key maps
  to a step check. Files:
  `items/<id>/_starter/module/{go.mod.tmpl, <editable>.go, visible_test.go}` (2–3 visible tests that compile and fail
  on the starter), `items/<id>/_code/module/…` (a reference that passes the visible tests in the runner image — p-01's
  content-CI gate), `sections/attempt/01-statement.md` (original draft listing the exported API from `module.api`).
- **Quiz items:** `choice` / `blank` parts (`cadence: final`, `grading: auto`, **no correctness in the public file** —
  the answer-free schema test enforces it), `grader: [{step: answers, kind: key, inputs: [...], required: true}]`,
  role `drill`; option ids stable (a label edit on a key-graded part is flagged by content CI).
- **No hints or editorial sections** (the stamp gate): until the owner stamps them, the hint stage shows the templated
  generic hint. Everything is written from scratch; Go blog / tour / "Go by Example" appear only as outbound `links`.
- The scaffolds are structurally complete (content CI green, seeded in compose); the owner turns drafts into final
  content and stamps `review.statement` during `ev-pilot-content`.

### 3 · `preview` hidden everywhere + `COURSE_STATUS_OVERRIDE` [X]

Sources: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
("`preview` hides a course everywhere outside the cohort … tested like the course-slug guard"; T-2 list; ≤ 6 non-kill
flags with owning and removal milestones), [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2)
(cohort from session-validate, never the JWT), [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P2, P3, P10).

- **Real-manifest test matrix** (the M1/M2 tests used fixture manifests; these load the embedded go-concurrency
  manifest). Accounts: anonymous, `learner`, `tester`, `owner`:

  | Surface | anonymous / `learner` | `tester` / `owner` |
  |---|---|---|
  | `GET /api/paths` (catalog) | no go-concurrency entry | entry with `status: preview` |
  | every `/api/paths/go-concurrency/…` route (route enumeration) | 404 `course_not_found`, same body as an unknown slug | 200 |
  | `GET /api/problems/gc-001` and **every** `/api/problems/{id}/…` sub-route (sections, attempts, submissions, runs, drafts, arena) | 404, same body as an unknown id | 200 |
  | enrollment (identity `handleStartEnrollment`, `POST /paths/{slug}/start`) | 404 `course_not_found` | **200** with the `enrollment` body (idempotent; the status the handler returns on `main`, which m1-04 task 7 doesn't change) |
  | `GET /api/agenda`, course dashboards, revision all-courses | no gc item, no minutes | included |
  | public profile `/u/<name>` header totals + course rows; `/public/stats?paths=go-concurrency` | never shown | **never shown either** (public surfaces hide `preview` for every viewer) |
  | coach with context `problem:gc-001` or `go-concurrency:concept:*` | 404 | allowed (D27 applies) |
  | SPA `/xlearn/go-concurrency` | NotFound identical to an unknown course | the course |

- **`COURSE_STATUS_OVERRIDE`** (T-2) in `internal/course`: `slug=status[,slug=status]`, statuses `active | preview |
  coming_soon` only; applied inside `course.Load` so every service that embeds the manifests (all of them) agrees;
  unknown slugs, unknown statuses, `retired` and `dsa` are ignored with one ERROR log line (a typo never hides DSA or
  crash-loops a release); the effective overrides are logged at start. Parser and application tests. Use cases: hide the
  pilot without a tag (`go-concurrency=coming_soon`), or flip a course early for everyone.
- **Flag inventory** row (status.md, [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)):
  `COURSE_STATUS_OVERRIDE` · tier T-2 · owning milestone **P** · default unset (manifest status wins) · set, if ever, by
  an infra PR on **every** `xlearn-*` HelmRelease · **removal milestone: GA** — at [ga-01](sprint-ga-01.md) it is either
  removed (the GA flip made it redundant) or promoted to a permanent operating mode by amending ADR-0034 §2's list (and
  leaving the ≤ 6 non-kill budget). Record which is recommended: keep it, since every future course launch needs the
  same "hide without a tag" path.

### 4 · Multi-course UI at full fidelity (AB02/AB05) [X]

Build to the frozen `design-system/screens/v2/AB02-course-nav.html` and `AB05-catalog-agenda.html` **"P · full
fidelity"** sections (AB02-P1–P7, AB05-P1–P7), `theme.css` verbatim.
- `web/src/screens/Catalog.tsx` (+ `lib/dashboard.ts` `useAgenda()`): catalog cards for DSA and Go Concurrency Patterns
  (`ds-badge--violet` **Preview**, "Pilot · 10 exercises · 2 weeks", the honor note, [Enroll]); the two-course agenda in
  minutes, "Reviews first", per-course new work, "nothing fits" and the R-SR5 per-course pause; the 7-day agenda toggle.
- `web/src/nav.ts`, `components/Sidebar.tsx`, `PathSwitcher.tsx`, `Topbar.tsx`: the go-concurrency nav from its manifest
  (no Mock), the Preview badge in the switcher, crumbs with the item noun (`xlearn / go-concurrency / exercises / gc-003`).
- `Roadmap.tsx`, `Week.tsx`, `Problems.tsx`: labels from `item_noun`; exercise rows with a code or quiz glyph; no topic
  chip while an item is live (`withhold()` unchanged).
- `Revision.tsx`: go-concurrency touches listed with their manifest badges ("Recall · ~5 min", "Re-solve · 25:00 · Mock
  conditions"); the recall-quiz touch body is p-03's (AB14 F12) — this sprint renders the badge and entry only.
- The SPA never filters `preview` itself: the server already omits it; a non-cohort deep link lands on NotFound.

### 5 · Multi-file workspace (AB15 F1–F2) [X]

Sources: [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (value `{language, files[]}`, read-only
`visible_test.go`), [t4 §5.1](../research/t4-judge-contract.md#51-registries) (closed registries, no self-registration),
[t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (the `PartWidget` contract),
AB15 F1–F2 and F16, [m3-11](sprint-m3-11.md) (Workspace-Code: `web/src/screens/workspace/{CodeEditor,EditorPane}.tsx`,
the lazy CodeMirror chunk, `serialize()` and drafts in `web/src/lib/judge.ts`), [m3-05](sprint-m3-05.md) (the `draft`
table, keyed by context, part and language).

- **The part registry starts here.** No sprint before P builds t4 §5.5's `PartWidget` registry: m3-11 wired the
  single-file editor straight into Workspace-Code, and `web/src/parts/` doesn't exist. This sprint creates:
  - **`web/src/parts/registry.ts`**: a closed map `PartType → PartWidget` in one file (no `init()`-style
    self-registration, t4 §5.1), with one entry, `code`. [p-03](sprint-p-03.md) adds `choice` and `blank` to this file.
  - **`web/src/parts/code/`**: the `code` widget, implementing `PartWidget<CodeConfig, CodeValue>` by **wrapping** m3-11's
    editor, not rewriting it:
    - `load()` returns the existing lazy `web/src/screens/workspace/CodeEditor.tsx` chunk, so non-judged screens still
      never download CodeMirror (m3-11's lazy-chunk rule);
    - `empty(config, {language, starter})`: a single-file config gives m3-11's starter behaviour, and
      `harness: gotest@1` builds `{language: "go", files[]}` from the starter module;
    - `serialize` / `deserialize` / `validate` take over m3-11's pre-send size checks from `web/src/lib/judge.ts`
      `serialize()` (≤ 64 KiB decoded code, ≤ 1 MiB encoded request; `judge.ts` calls the widget), and `judge.ts`
      keeps transport, polling and the draft PUTs;
    - `draft {debounceMs: 15000, timeoutMs: 25000}` (m3-11's code policy).
  - `EditorPane` renders the code part through the registry. **DSA single-file behaviour is unchanged:** m3-11's
    `Workspace.test.tsx` and `judge.test.ts` stay green without edits beyond imports.
  - This is widget code, inside the P exit rule. Record the registry's creation in the Decisions log.
- **Multi-file UI for `gotest@1`** (in `web/src/parts/code/`): file tabs (`role=tablist`, arrow keys) and a small file
  tree; editable files open in CodeMirror, `visible_test.go` and `go.mod` are read-only with `xl-lock` "read-only"; no
  add, rename or delete.
- **Drafts:** m3-11's autosave and m3-05's `draft` key (context, part, language) are unchanged. For `gotest@1` the draft
  body is the whole part value `{language, files[]}`, holding every editable file (read-only files come from the starter
  and are never stored). One PUT per debounce, flushed on Run, Submit, blur, `hidden` and route leave. No server change.
  The widget's `serialize` computes the encoded size and shows the AB08 413 copy before sending anything over the 64 KiB
  cap (L6).
- Actions: **Run visible tests** (⌘↵) and **Submit** (⌘⇧↵); the statement panel lists `module.api[]`.
- Server side stays authoritative: judge rejects any path outside the editable list (p-01's A5 lint, REJECTED, uncounted).
- Results use m3-12's generic dock states until [p-03](sprint-p-03.md) adds the `gotest@1` result view.
- < 1024 px: Statement / Files / Results tabs; the file tabs become a `<select>` (AB15 F16).

### 6 · Tests + compose e2e [X]

- Go: the manifest validates and its method table test passes; the id guard accepts `gc-001…gc-010`; content CI green
  (schema, answer-free, leak lints, embed allowlist, Markdown profile, reference passes visible tests); the CHECK
  widening store test; the §3 matrix (gateway route enumeration + identity + public-profile tests) with the real
  manifest; the override parser; `sqlc diff` clean.
- Web (vitest + `fetchMock`): one test per AB02-P* and AB05-P* frame; the registry (its keys equal the supported part
  types; `code` resolves to the lazy editor); the multi-file widget (tabs, read-only locks, the `{language, files[]}`
  draft body, size guard, keyboard); m3-11's DSA Workspace and `judge.ts` tests unchanged and green.
- e2e in compose (`-tags e2e`): the owner enrolls in go-concurrency, opens `gc-001`, sees the module's files with the
  read-only ones locked, and — where the compose stack runs the runner (m3-06's e2e) — Runs the visible tests against the
  reference; a `learner` gets 404 on the catalog entry, the course, `gc-001` and enrollment, and `/u/<owner>` shows no pilot row.

### 7 · Prepare `ev-pilot-content` [X]

- `docs/v2/authoring.md` ([m3-01](sprint-m3-01.md)'s guide) gains a **go-concurrency (`gotest@1`)** section: the module
  layout and file roles, `api[]` and `contract_hash`, visible vs hidden tests, `goleak`, `testing/synctest` for time,
  `-count` and the detection-rate expectation, per-test deadlines ≥ 10 × the go-race baseline (p-01), Σ TL ≤ 40 CPU-s,
  `mem_mb` ≥ 2 × peak, honor labelling, probe keys in the pack, quiz `distractors` tagged with declared categories, stamps.
- `docs/v2/status.md` content status: go-concurrency rows per item (scaffolded ✅ · statement stamped · pack stamped);
  owner event `ev-pilot-content` marked ready (December, 10–20 h).
- The authoring itself is owner content (`ev-pilot-content`), done after this sprint ships: it doesn't gate this
  sprint's _Overall_ ✅, and [p-03](sprint-p-03.md)'s entry gate waits for it.

### 8 · Record [X]

[`../status.md`](../status.md): Sprint board row; milestone P 🔄; the **flag inventory** row (task 3); the content
rows (task 7); the Artboards rows AB02/AB05 (full) → consumed by p-02; Decisions-log lines (manifest values as drafted,
revisable by the owner with a content PR, the status-CHECK relaxation, override semantics and its GA decision, no mock for the pilot, the part registry
`web/src/parts/registry.ts` created here with the `code` widget wrapping m3-11's editor).

## Acceptance criteria

- [ ] A non-cohort account sees no trace of the pilot (catalog, profile, stats)
- [ ] Owner can enroll and open a gc item
- [ ] The go-concurrency manifest validates as `preview` with every method block; `gc-001…gc-010` seed; content CI green
- [ ] Every surface in the §3 matrix returns the non-cohort result for anonymous/`learner` and the cohort result for `tester`/`owner`, with the real manifest
- [ ] `COURSE_STATUS_OVERRIDE` parses, applies in every service via `course.Load`, ignores bad input with an ERROR log, and is in the flag inventory with owning milestone P and removal milestone GA
- [ ] `path.status` accepts `preview`/`retired` (relaxation marker, no contract marker, store test, `sqlc diff` clean)
- [ ] Catalog, agenda and nav match AB02-P*/AB05-P*; the multi-file workspace matches AB15 F1–F2 (read-only files locked)
- [ ] CI green (Go, sqlc, content, web, openapi drift, route enumeration, e2e)

## Release

**Merge only (ships in v1.15.0).** [p-03](sprint-p-03.md) tags `v1.15.0` with the course still `preview` (cohort-only).
No infra PR (the new env var stays unset), no new pod, subject, stream or in-cluster caller: the memory sum, NATS ACLs
and NetworkPolicies are unchanged.
The first non-DSA `path_slug` events need consumers ≥ 1.7.0 ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)) — satisfied.

## Definition of Done

CI green · squash-merged · no tag · the owner can use the course in compose · statuses updated (this file +
[`../status.md`](../status.md)) · the authoring guide section merged · `main` synced.

## Risks / watch-outs

- **Consumer floor ≥ 1.7.0 for non-DSA `path_slug` events** — satisfied (production runs ≥ 1.14.0).
- **A leak through an item route.** Items are addressed by global id, so `gc-*` must 404 for non-cohort on every
  `/api/problems/{id}/…` route, not just the course routes — the route-enumeration test is the control.
- **A hotfix tag cut from `main` between p-02 and p-03** would carry the preview course (without the quiz widget) to the
  cohort. Cut hotfixes from the last tag, or set `COURSE_STATUS_OVERRIDE=go-concurrency=coming_soon` for that window.
- **Strategy per archetype.** If practice can't grade quiz items with `weighted_gate@1` in a `verdict_timer@1` course,
  it's an M3 gap: stop and report instead of adding pilot-specific practice code (P exit: widget/profile code only).
- **Honor labelling:** the catalog card and CTA say results count on the learner's honour; nothing implies judge-checked.
- **Content rights:** statements, starters, tests and concepts are original; no copied Go examples.
- **Manifest values are course design:** they land as drafted (D40); the owner can revise them later with a content PR,
  which is a deliberate, tested change (the method table test).
