> **T0 research appendix** — the full synthesized frame behind [ADR-0026](../../adr/0026-per-course-extensibility-model.md)
> and [feasibility § T0](../feasibility.md#t0--guiding-frame).
>
> **Method:** three independent candidate models (evolve-v1, course-diversity, arch-ops-security),
> each reviewed by two adversarial critics (a six-course end-to-end stress test, and solo-dev cost,
> single-node ops, security and migration), then synthesized. Critic scores were 6–7/10 per candidate;
> this frame fixes their blocker and major findings (§13).
>
> **Status:** settled with the owner 2026-09-24 (decisions D1–D4, §12). Edited after synthesis for the
> owner's routing rule: profiles under `/u/`, and course slugs **not** reserved as usernames.

# T0 guiding frame: per-course extensibility model

## 1. Recommendation in one paragraph

A course is data and a capability is code. The loop after the learning signal is one universal engine that never branches on course.

The frame starts from the **evolve-v1** candidate ("Course Manifest on the v1 Spine"):
- Each course gets a manifest of parameters only, with no logic.
- Screen widgets and evaluators are chosen by part type or evaluator kind, never by course slug.
- One new service, `judge`, is added.
- practice stays the only writer of learning signals.
- DSA keeps its bare ids, and a per-course self-report policy lets the manual loop coexist during migration.

Three ideas are taken from **course-diversity** ("Course-as-Data over a Universal Learning Loop"):
- Grading strategies (rules that turn evaluation facts into a grade) are a closed set in Go. The manifest selects one and sets its thresholds.
- An evaluator step can read several parts of an item.
- Items get a role (core, reinforcement or drill).

Five ideas are taken from **arch-ops-security** ("Course bundles on a capability registry"):
- Evaluator kinds are grouped by how trusted they are, not by course.
- The grading mode is set per part.
- Events carry categories, numbers and references, never prose.
- The judge runs its jobs from a Postgres queue.
- Each consuming service re-checks the context it is handed.

The critics' fixes change the plan in six places:
- **Practice owns every timed attempt**, both course attempts and revision touches, so the attempt runtime and the grading mapper exist once.
- **The manifest is compiled into every image** from the same tag, not fetched at runtime.
- **Answer keys are private** as well as hidden tests.
- **Routing builds on ADR-0025** (profiles under `/u/`, shipped in v1.5.0). Course slugs are guarded on the course side, not reserved as usernames.
- **The grading rule reproduces v1's semantics exactly**, and any behaviour change (for example unsolved attempts entering the review ladder) is an explicit step the owner approves after DSA is proven unchanged.
- **AI never blocks finalizing an attempt**, and no background worker calls an unauthenticated endpoint that spends money or returns personal data.

## 2. Conceptual model

| # | Concept | One-line definition |
|---|---|---|
| 1 | **Course** | The existing `curriculum.path` slug: the unit of enrollment (`identity.path_enrollment`), the URL namespace, and the owner of exactly one manifest. |
| 2 | **Course manifest** | Declarative method parameters, compiled into every service, versioned by content hash, and holding no logic. It covers nav labels, stage timers and labels, the grading strategy and its thresholds, the revision touch format, the mistake taxonomy, mock formats, estimated minutes, coach persona, public-stats picks and self-report policy. |
| 3 | **Item** | The v1 `problem`. Its id is globally unique and never re-parented. It has a role (core, reinforcement or drill), a topic, stage-gated public sections and one or more parts. It may reference **course assets** (a SQL dataset, a canvas palette): the public descriptor lives in curriculum and any private companion lives in the eval pack. |
| 4 | **Part** | A typed answer slot (`code`, `text`, `choice`/`blank`, `canvas`; `audio` reserved). It names a widget, an evaluator step and a **cadence**: `iterate` (run or submit many times) or `final` (evaluated once, at conclusion). |
| 5 | **Evaluator** | A judge adapter chosen by kind: `code` (with a runner profile), `key` (answer match), `ai_rubric`, or `composite`, plus an optional `analyzer` post-step. Its inputs are one or more parts plus the **private eval pack**. It returns an **Evaluation**: `passed \| failed \| inconclusive`, item-scoped `checks[]`, a score out of a maximum, and references. Evaluations are evidence only. |
| 6 | **Attempt** | practice's timed record, with `purpose ∈ {course, touch}`. It holds stages, server timers, reveals and the evaluation facts it has consumed, and it **concludes exactly once**. |
| 7 | **Learning signal** | The narrow waist: facts from a concluded attempt, written only by practice. That is `problem_solved` v2 for course attempts and `touch_concluded` for revision touches. `ScoreMock` remains the only mock signal. |
| 8 | **Common loop** | review (five-touch ladder, mistake journal, weak area), assessment (mock lifecycle, projections) and the Today/Roadmap/Progress composition in the gateway. None of these branch on course. |

```
Course ──(manifest: params only)───────────────────────────────────────────┐
  └─ Items (role, topic, public sections, course assets)                   │
       └─ Parts ──▶ Widget (SPA registry, by part type) ─▶ solving ground  │
             │                                                             │
             └─ submit/run ─▶ judge: Evaluator(kind) + private eval pack   │
                                   + runner (T3) / platform LLM (T5)       │
                                   │ evaluation_completed (counted only)   │
                                   ▼                                       ▼
                 practice Attempt(purpose=course|touch) ── grading strategy(manifest)
                                   │  grade · passed · criteria · graded_by
                                   ▼
        LEARNING SIGNAL  problem_solved v2 (course) · touch_concluded (touch)
                                   │                ▲ evaluation_analyzed (category + ref,
                                   │                │ never blocks, never prose)
          ┌────────────────────────┼────────────────┴───────────────┐
          ▼                        ▼                                ▼
  review: ladder, journal,   assessment: projections,       gateway: Today (planner),
  weak area (course-blind)   mock lifecycle/ScoreMock        Roadmap, Progress (labels only)
```

## 3. Plug-in boundaries

Plug-in forms:
- **Data** means a file in `curriculum/<slug>/`, validated in CI.
- **Code** means a closed registry keyed by type, never by course.
- **Service** means a new process. Only `judge` is one.

| Seam | Common or per-course | Plug-in form | Owning service | How a new course adds it |
|---|---|---|---|---|
| Catalog, enrollment, slug | Common | Data: `paths.json` row. Code: the **course-slug guard**, which rejects slugs that clash with a static top-level SPA segment (`u`, `auth`, `settings`) or a gateway-owned path (`api`, `assets`, `healthz`, `readyz`, `.well-known`) at seed time and in a test | curriculum | Add a row. The guard fails CI on a clash. Course slugs are **not** reserved as usernames (ADR-0025 update). |
| Nav and routes | Common screen set and route segments. Per course: labels, visibility, item noun | Data: manifest `nav` over closed screen keys. Code: one `/:course/*` route | SPA (embedded in gateway) | Set labels. No code. |
| Content (phases, weeks, concepts, items, sections) | Common shape, per-course content | Data: `curriculum/<slug>/*.json`, globbed. Replaces the fixed list at `internal/curriculum/seed.go:18-24` and `curriculum/embed.go:15` | curriculum | Author the files, with course-prefixed ids (`sd-001`). CI refuses id collisions and re-parenting. |
| Course assets | Per course | Data: public descriptor plus an optional private companion | curriculum (public), judge (private) | Declare the asset. Items reference it by id. |
| Private eval pack (hidden tests, answer keys, expected result sets, rubric anchors) | Per item | Data: a private artefact that only judge reads | judge | Author it in the private source. CI checks that every auto-graded part has assets. |
| Solving ground | Common workspace shell (statement, parts, timer display, Run/Submit). Widget per part type | Code: SPA registry `grounds/<partType>` and `results/<evaluatorKind>` | SPA | Reuse the existing widgets (data only), or add a widget (code plus release). |
| Evaluator and runner | Common Submission→Evaluation contract. Adapter per kind; config per item | Code: judge adapter registry plus a runner profile (T3) | judge (plus the runner) | Reuse the kinds (data). A new kind or profile needs code, a release, and an ADR if it adds a new trust surface. |
| Grading strategy | Universal 4-grade scale. Strategy chosen from a closed set | Code: `verdict_timer@1`, `rubric_pct@1`, `weighted_gate@1`, `self`. Data: selection and thresholds | practice | Pick a strategy and set its thresholds. |
| Stage ladder and timers | Universal roles. Per-course durations, labels, penalty days, re-implement policy | Data: manifest `stages`, copied onto the attempt when it starts | practice (gateway does the gating) | Data only. |
| Revision touch format | Universal ladder and pass rule. Per-course format per level band. Per-item probes | Data: manifest `revision` plus the item's `revision.probes[]` | practice (touch attempts), review (ladder) | Data, plus authoring the probes. |
| Mistake taxonomy | Universal core `{misread, time_management, communication}` plus per-course categories | Data: manifest `mistakes`. Ids are append-only | review | Data only. |
| Mock format | Lifecycle is universal. Rail, rubric, targets and item pools are per course. A course may have no mock | Data: manifest `mock`. The rubric is copied onto the session | assessment | Data only. |
| Progress and public stats | Universal providers. Per course: metrics picked from a closed set, and visibility | Data: manifest `public_stats` | assessment (numbers), gateway (composition) | Data only. |
| Coach persona | Universal mode gate. Per-course persona | Data: manifest `coach` | coach | Data only. |
| Today inputs | Universal planner. Per-course `est_minutes` per format | Data: manifest `plan` | gateway (a pure `plan` package) | Data only. |
| Degrade overrides | Common | Env in the infra HelmRelease (Flux applies it in about 1 minute) | practice, judge, gateway | None. |

## 4. Universal vs per-course method

| Aspect | Choice | Why |
|---|---|---|
| Five-touch ladder | **Universal** Day 1/3/7/21/45, kept as a Go constant (`review/store/store.go:27`). A fail resets to Day 1 (R-SR3). **Per course:** the touch format for each level band (L1–3 a cheap recall format; L4–5 the full format under mock conditions) and its timers. | The ladder is the product's thesis, and review is already generic over an opaque id. Varying the format rather than the intervals handles the difference in cost between courses (a 45-minute SD redesign on Day 1 would be waste). |
| Outcome scale | **Universal** four grades (clean/rough/assisted/miss), plus a separate `passed` flag and an optional normalized score. Each grade is derived by practice using the course's **closed** grading strategy. | Projections, mastery weights and the mistake allow-list all key on these four values (`assessment/store/projections.go:44-57`). A strategy is code, so rules never become a DSL. |
| Grading rule for code (`verdict_timer@1`, DSA) | **Matches v1 exactly** (`Problem.tsx:21-26`). Grade comes from the *first passing submit*: before the hint and within the attempt timer is clean. Before the hint but late, or more than N failed submits, is rough. After the hint but before the solution is assisted. No pass before the solution is revealed, or giving up, is miss. Re-implement (R-PF4, PRD G1) must happen before the attempt can conclude when the course sets `reimplement: required`, but it does not change the grade. | This fixes the blocker. A table-driven golden test pins the rule against the v1 copy. |
| Stage ladder and timers | **Universal** roles attempt → hint → solution (reference) → re-implement (required, optional or off per course) → conclude. R-PF2's owed Day-3 touch is universal. Durations, labels and penalty days are per course and copied onto `practice.attempt`. The stage CHECK stays. | Every course has a timed first try, a scaffold, an exemplar and a redo. Gateway gating is already agnostic to stage names (`aggregate.go:154-180`). |
| Mistake taxonomy | Universal core plus per-course categories, validated against the manifest rather than the CHECK (`review 00002:28-30`). Ids are append-only. Mechanics stay as v1: one open entry per item, close after 2 clean revisits, weak area per course. Mocks open no mistakes in v2.0. | `off_by_one` means nothing for behavioral. Because every concluded attempt now schedules touches, every open entry has touches that can close it. |
| Rubric | Per course: N dimensions (1–10) each scored 1–5, stored as `total`/`max_total` and copied onto each session. Displayed normalized and never averaged across rubrics. | `/35` (`assessment 00001:39`) is meaningless for SD or behavioral. Score-once stays universal. |
| Mock rail and targets | Per course (optional): duration, phases, rubric, and a server-chosen **ordered item list** per session. The lifecycle (server timer, `FOR UPDATE` score-once, outbox) is universal. | Formats differ between courses; the lifecycle does not. SQL and behavioral mocks need several items, and `mock_session` holds one `problem_id` (`assessment 00001:33`). |
| What enters spaced revision | **Universal end state:** every concluded, counted attempt on a revisable core or reinforcement item anchors L1–L5 at conclusion, **whatever the grade, including miss or give-up**, and a below-clean grade also opens a mistake. Drills are revisable per course. Arena, ahead-of-schedule and mock attempts never schedule. **Migration:** DSA stays on v1's first-clean-only rule (`store.go:307-309`) until the step the owner approves (§9 M2c). New courses start on the universal rule. | This delivers the vision's "revision even for UNSOLVED attempts" and PRD R-OL3. It also closes the real v1 hole: below-clean outcomes without an early reveal never get a ladder. `ScheduleTouch ON CONFLICT DO NOTHING` combines it with R-PF2. |
| Revision touch pass | **Universal:** a touch passes when `passed` holds and every required criterion is met. DSA's criteria are `pattern_named_fast` (under 120 s), `correct_in_timer` and `complexity_stated`, so `AutoPass` (`store.go:706-708`) is unchanged. Touches graded by AI pass at the course pass threshold, not the clean threshold. | One rule for every course. review keeps its scoring core. |
| Readiness targets | Per course, as a normalized score at a week relative to the course. | W13/W15 (`assessment/mock.go:147`) only mean something for a 16-week course scored out of 35. |
| Item role and frontier | `role ∈ {core, reinforcement, drill}` generalizes `is_reinforcement`. The frontier counts **concluded** core items (any grade, as v1 does today, `scheduling.go:69-100`). Coverage and "passed" count grade ≥ rough. | Otherwise drills would block weeks, and one noisy AI "miss" could freeze a course's frontier. |
| Grading mode and self-report | Per part: auto, AI, or none (none means practice's v1 self path). Per course, and per signal (outcome, touch, mock): `allowed \| evaluator_only`, with an env override as the runtime kill switch. | Lets items migrate one at a time and gives an honest signal later. The override is the switch to fall back to when the judge or AI is down. |

## 5. The learning-signal contract (narrow waist)

**Writers.** practice is the only writer of learning signals, through its outbox and in the same transaction that concludes the attempt.
- judge writes evidence only: `evaluation_completed`, and later `evaluation_analyzed`.
- review writes `touch_scored` and the mistake events.
- assessment writes `mock_completed`.
- Nothing downstream reads parts, widgets or evaluator kinds.

**(a) Course attempt → `xlearn.practice.problem_solved`, envelope v2, additive only**
- Kept from v1: `problem_id`, `outcome` (the grade), `first_solve`, `below_clean`. It already fires for a miss, so "solved" already means "concluded".
- Added: `path_slug`, `attempt_id`, `graded_by` (self, auto, ai or override), `passed?`, `score?{value,max}`, `criteria[]{key,met}`, `stage_reached`, `revealed_early`, `within_timer`, `failed_submits`, `evaluation_ids[]`, and `policy_version` (manifest hash).
- It carries no prose, code or diagnosis text.
- Triggers:
  1. v1 `POST /problems/{id}/outcome` (`graded_by=self`, subject to the course's self-report policy). This stays the path for any item without an evaluator.
  2. practice's **new first consumer** (inbox plus a durable consumer on `XLEARN_JUDGE`) concludes the attempt when a *conclusion-triggering* evaluation arrives. Those are: the re-implement submit passes, the final submit, or give-up.
- The first conclusion wins; conclusion is idempotent on the attempt row.

**(b) Revision touch → `xlearn.practice.touch_concluded` (new subject)**
- Fields: `path_slug`, `problem_id`, `revision_item_id`, `touch_level`, `attempt_id`, `touch_passed`, `criteria[]`, `recall_secs?`, `graded_by`, `evaluation_id?`, `policy_version`.
- practice computes `touch_passed` from the course's required criteria, using server timing (a new server-side touch start). review therefore never interprets parts.
- review's consumer calls the **same** internal scoring function as v1's `Score(ScoreInput)` (`review/store/store.go:404`). The v1 self endpoint `POST /revisions/{id}/score` stays and still maps to `AutoPass`.
- `touch_result` gains `graded_by` and `evaluation_id`.

**(c) Mistakes**
- review opens entries on `below_clean`, as today (`store.go:338-345`).
- judge's `evaluation_analyzed{evaluation_id, attempt_id, path_slug, problem_id, category (a taxonomy key or ""), confidence, feedback_ref}` pre-fills the category idempotently, keyed on `evaluation_id`. It arrives later and **never blocks conclusion**.
- The prose stays in judge, and the gateway composes it at read time with the user's JWT.
- `PATCH /mistakes/{id}` is unchanged, so the learner can always override. It validates against the manifest.

**(d) Mock → `ScoreMock`** (`assessment/store/store.go:265`)
- Same shape. It validates against the rubric snapshot on the session and adds `scored_by` (self or ai).
- `mock_completed` gains `path_slug`, `rubric_id`, `total` and `max_total`.
- Mock evaluations are evidence shown to the scorer. A mock emits no `problem_solved`.

**(e) Projections input: `xlearn.review.touch_scored{path_slug, problem_id, level, passed, graded_by}` (new)**
- Heatmap "reviews" count completed touches rather than scheduling events. This fixes the +5 inflation per first clean solve (`projections.go:215-220`).
- Mastery becomes max(best first grade, level derived from touches), so passing touches can raise a noisy miss.

**Envelope rule**
- Every practice and review subject gains `path_slug`, including `solution_revealed_early`, which today is `{problem_id}` only (`practice/store/store.go:332-334`).
- Consumers treat a missing path as `dsa` **only for version-1 envelopes**. A v2 envelope with no path is parked and alerted on.
- The owner's 19 v1 events replay deterministically, which satisfies ADR-0018's ban on enrichment.

**How unsolved attempts schedule revision**
- "Give up" submits the current draft with `action=give_up`. judge evaluates and analyzes it, so an unsolved attempt still has evidence.
- The solution then unlocks, and the learner re-implements if the course requires it. practice concludes the attempt as `miss` with `first_solve=true, below_clean=true`.
- Under the universal rule, review anchors L1–L5 at `occurred_at`, reusing the upsert behind `ReanchorTouch`, and opens a pre-filled mistake.
- Abandoned attempts stay resumable (v1 behaviour). There is no auto-miss sweeper in v2.0 (candidate; decided in T4).

**Lifecycle invariants (fixes the blockers on conclusion and cadence)**
- **States:** `attempting → awaiting_evaluation → (provisional, AI-graded only) → concluded`.
- **Evaluation order:** judge processes each attempt's jobs **first in, first out**, so the evaluation that triggers conclusion is the last one practice sees.
- **Facts at conclusion:** the grading strategy reads the **latest** submission for each part. `within_timer` uses judge's server-stamped `submitted_at` against the attempt's deadline.
- **Cadence:** `iterate` parts show Run/Submit. `final` parts (AI rubric) are evaluated once, at conclusion. `inconclusive` never counts as a failed submit and never fails a touch.
- **AI verdicts:** an AI-graded verdict is provisional until the learner accepts or disputes it (one re-grade, then a tagged override), or until a timeout accepts it. This keeps exactly one signal per attempt.

## 6. Six-course fit

| Course | Item kinds | Solving ground archetype(s) | Evaluator archetype(s) | Fits? | Notes |
|---|---|---|---|---|---|
| dsa | Coding problem: one `code` part; per-language starters, samples and hidden tests. Pattern-recall probes for touches | `code` (Go now; C++ once its runner exists) | `code` (runner `go`, profile chosen per language), `analyzer`. No evaluator means the v1 self path | Yes | Manifest is v1's constants, pinned by a golden test: 15/10 min, 3 days, 8 categories, 7×1–5 /35, 45-min rail, W13 24 / W15 28 / Pre 30. Recall becomes free-text with an alias list, and the pattern chip is hidden during a live touch (`Revision.tsx:235-237`). The real gap is content: 14 of 151 problems are seeded and none has tests. |
| system-design | Multi-part round: requirements (`text`), estimates (`blank`, numeric with tolerance), HLD (`canvas`), deep dive and trade-offs (`text`), concept drills | `text`, `blank`, `choice`, `canvas` (typed component and edge palette; semantic export) | `composite`: `key` for estimates and drills, plus one `ai_rubric` step over canvas and text against private anchors. `analyzer` | Yes | New primitives are the canvas widget and `ai_rubric`. Freehand shapes are ignored by grading. Timers are about 35–45 min. Touches: L1–3 are a 5-min recall of components, bottleneck and one estimate; L4–5 are a timed redesign. The reference design is public once gated. |
| go-concurrency | Implement or fix concurrent code (multi-file); predict and spot-the-race drills | `code` (Go, file tree), `choice` | `code` (runner `go-race`: `-race`, `-count=N`, timeouts, leak check), `key` | Yes | The three-state verdict is needed: a flaky or infrastructure failure is `inconclusive`. `-race` is CPU-heavy and falls under the T3 concurrency cap. Verdicts from an in-process `go test` harness have integrity only against the learner, not against tampering. |
| lld-ood | Design and implement: class diagram (`canvas`), code against a **facade interface fixed by the item**, rationale (`text`) | `canvas` (UML palette), `code`, `text` | `composite`: `code` tests gate on the facade, then `ai_rubric` over the diagram, code and rationale | Yes | Each item's evaluator block says either "facade tests + AI design" or "AI only". Tests never fix the design behind the facade. Language set is Go only unless the owner says otherwise (T3). |
| sql | Query, optimisation and schema-design items; a large drill set | `code` (SQL mode) plus a schema explorer over a **course asset** dataset; `choice`/`blank` | `code` (runner `sql-pg`: ephemeral seeded database, result-set comparison, EXPLAIN checks as `checks[]`), `key`, `ai_rubric` for design rationale | Yes | Datasets are course assets: a public descriptor plus a private companion holding hidden rows and expected results. Learner SQL never runs on the platform's CNPG. Plans must be deterministic (fixed data volume, ANALYZE). Drills (role=drill) use short timers and never block the frontier. |
| behavioral | STAR prompt (`text` with structured fields); `audio` later | `text` (STAR fields) | `ai_rubric` only (structure, ownership, metrics, reflection); self-rubric as fallback | **Partial** | v2 revises *prompts*, not the learner's stories. A story bank needs an entity owned by the learner, which is a spine change because review keys on `problem_id` (`review 00001:27-39`). That is a later ADR. Answers are personal data: they need consent before any platform-key LLM call, public stats are off by default, and the cross-service erase path must exist first. Its mock depends on T6. Not a pilot course. |

## 7. Screens & routing

**URL model**
- This builds on ADR-0025, shipped in v1.5.0: profiles live at `/xlearn/u/:username`, and course slugs are **not** reserved as usernames. The only guard is the course-slug guard (§3), which keeps slugs clear of static top-level segments.
- Course routes: `/xlearn/:course` (Roadmap) plus `/xlearn/:course/{dashboard, problems, week/:n, concept/:slug, problem/:id, revision, mistakes, mock, progress}`. The segments are the same for every course; only labels differ.
- Account routes: `/xlearn` (Catalog plus the cross-course agenda), `/xlearn/settings`, `/xlearn/auth`, `/xlearn/u/:username`.
- There is one `/:course/*` route **inside AuthedShell**, checked against the session-gated catalog.
  - Unknown slug: NotFound. `coming_soon`: the catalog teaser. Signed-out visitor: `/auth`, as today.
  - This drops boot-time route generation, a public paths endpoint and any slug-versus-username resolver.
- A `useCourse()` hook replaces the literal `"dsa"` and `/dsa/` sites listed in the audit (a8, plus `nav.ts:90-135`). `navForPath` renders the manifest `nav` block.

**API**
- Course-scoped aggregates go under the existing prefix: `/api/paths/{slug}/{dashboard, progress, revision/due, mistakes, weak-area, mocks, mocks/trend}`.
- Items are addressed by global id: `/api/problems/{id}`, `/api/problems/{id}/submissions`, `/api/submissions/{id}` (poll or SSE). The gateway resolves the course from the item's `path_slug`, replacing `bff.go:571,647`, `scheduling.go:74`, `progress.go:153,161,214` and `dashboard.go:166,174`.
- The old routes without a path stay as DSA aliases for one release.
- Cache and query keys include the slug.

**Scope of each screen**

| Account-wide | Course-scoped |
|---|---|
| Catalog and agenda, Settings, onboarding (any active course), Auth, public profile header (streak and heatmap as account activity), coach thread index | Roadmap, Week, Concept, Problems, Problem workspace, Revision (with an "all courses" toggle), Mistakes and weak area, Mock and trend, Progress |

- The public profile shows one row per **enrolled or started** course (v1 walks every active path, `public.go:144-232`).
  - Mock stats are per course and normalized.
  - Grades show where they came from (self, auto or AI).
  - Each course has its own visibility flag; behavioral is off by default.
- The coach keeps `problem:<id>` keys, since ids are global. Contexts that are not problems become `<course>:<ctx>` (`Coach.tsx:258-336`; the key is `coach_thread UNIQUE(account_id,page_context)`).

**Arena and honesty gating**
- While an item has an open counted attempt or a due or live touch, both the arena view (`?practice=1`, which today returns every stage, `bff.go:526-540`) and every item surface **withhold the solution stages and any field that gives away the answer** (topic or pattern) for that item.
- Arena Run/Submit calls never count and never use the platform AI key.

**Cross-course Today and budget** (decision D4)
- There is one study budget per account, **in minutes**, with `est_minutes` for each format taken from the manifest. v1 currently plans by a count of 3 (`dashboard.go:23`).
- Due touches from every course come first, oldest first.
- R-SR5 blocks new work **per course**: a course with overdue reviews does not advance.
- The rest of the budget goes to new work. A course dashboard uses its own course; the home agenda splits evenly across active enrollments.
- The planner is a deterministic pure function of (account, date) in a `plan` package that the gateway calls. There is no cross-course optimizer in v2.

## 8. Service boundaries & single-node budget

There are **8 services** (today's 7 plus `judge`) and one runner deployment. Nothing is merged, so this stays service-oriented.

| Service | Owns / changes |
|---|---|
| curriculum | Globbed per-course content; a seed guard against id collisions and re-parenting (`queries/problem.sql:40-41`, `concept.sql:17-18`); ids are **retired, never deleted**, and delete-missing applies only to sections, links and parts; manifest validation; serves a learner-safe manifest view. No events. |
| identity | No slug reservations (ADR-0025 update). Enrollment is checked against active slugs, in the gateway's start handler (`identity/handlers.go:292-299` accepts any slug today). |
| practice | The attempt engine for `purpose ∈ {course, touch}`: stages, server timers, R-PF2, copy-on-create snapshots of method parameters and `policy_version`, grading strategies, self-report policy. Its **first inbox and consumer** (`XLEARN_JUDGE`, counted evaluations only). The only writer of `problem_solved` v2 and `touch_concluded`. `solved` stays final for course credit (`store.go:220-227`). |
| review | `path_slug` on `revision_item`, `mistake_entry` and `weak_area_snapshot`. Consumes `touch_concluded` and `evaluation_analyzed`. Emits `touch_scored`. Anchors the ladder on any conclusion (behind the M2c toggle). Validates taxonomy against the manifest. Never fetches prose. |
| assessment | Per-course mock formats (rubric snapshot on the session, `total`/`max_total`, ordered `mock_session_item` list, `scored_by`). Projections gain `path_slug`; heatmap reviews and mastery come from `touch_scored`; `problemTotal=151` is removed (`assessment/progress.go:18`). Stores no transcripts or code. |
| coach | BYO-key chat only, with the persona from the manifest. Provider code moves to a shared `internal/platform/llm` package (candidate; decided in T5). |
| gateway | Composition and gating only: resolves the course, applies per-course enrollment and frontier gates, the arena withholding rule, the Today planner, and proxies submissions (a synchronous enqueue to judge, then poll or SSE). When a poll reports a terminal result it bumps the account's cache epoch, fixing the 15 s stale window (`bff.go:676-687`). |
| **judge (new; needs a new ADR)** | Schema `judge`, port 8087, stream `XLEARN_JUDGE` with retention limits (this needs a publisher option; `nats.go:78-83` hard-codes the stream settings today). Owns submissions and drafts, evaluations, the private eval pack, the adapter registry and the Postgres `SKIP LOCKED` job queue (runs exceed the 25 s handler timeout, `consumer.go:23`). Holds the platform LLM key. Emits events only for counted contexts; arena and Run results are served over its API only. Knows nothing about grades or ladders. |
| runner (a worker, not a bounded context) | Its own namespace: default-deny, ingress only from judge, no service-account token, secrets, database, NATS or egress. T3 decides the technology. |

**Budget (live today: 1520m CPU / 1548Mi requested, limits 8350m / 8970Mi):**
- judge adds about 50m CPU / 64Mi.
- The runner envelope for T3 is **2 or fewer concurrent jobs, about 0.5 vCPU / 1 GiB requested, and at most 2 vCPU / 3 GiB of limits**, plus a sandbox Postgres for SQL of about 100m / 256Mi.
- The total comes to about **2.2 vCPU / 2.9 GiB requested out of 4 / 15**.
- The memory-limit sum rises to about 12.5 Gi with no swap, so runners need hard cgroup limits and the lowest eviction priority. PriorityClass affects scheduling, not CPU share.
- The manifest adds no processes.

## 9. How live DSA migrates

Each step ships on its own, and the v1 manual loop keeps working until M5. Schema changes use expand → backfill → contract across releases. Consumers of a new subject ship **one release before** its producer, because review acks unknown subjects silently (`review/consumers.go:58-61`).

- **M0 (done, v1.5.0, 2026-09-24):** profiles moved under `/u/`; the reserved list is impersonation-only (#46, #47).
- **M1: spine with no behaviour change.**
  - **M1a (expand).** Includes:
    - the `course` package plus `curriculum/dsa/course.json`, with a golden test that the manifest equals the v1 constants and grading rule
    - the glob loader, id guard and retire semantics
    - `path_slug` on every event (v2 envelope) and column (default `'dsa'`)
    - `total`/`max_total` added next to `total_35`
  - **M1b.** The gateway literals become course resolution. The SPA gets `/:course/*`, manifest nav and `useCourse()`. The BFF gets course-scoped routes with DSA aliases. Coach contexts are prefixed.
  - **M1c (contract).** Drop the category and dimension CHECKs in favour of manifest validation, and drop `total_35`.
  - Every DSA item has no evaluator, so it uses the self path, and the loop is unchanged.
- **M2: attempt engine and projections.**
  - **M2a.** practice attempt `purpose=touch` with a server-timed touch start. The self endpoints are unchanged. review consumes `touch_concluded` (consumer first, then producer).
  - **M2b.** `touch_scored`, plus a projection replay that redefines heatmap reviews and mastery.
  - **M2c.** DSA switches to "every concluded attempt anchors the ladder" (approved by the owner 2026-09-24, decision D2). It lands in a later release than M1, and only after M1 ships with no behaviour change. It ships in the same release as M2b. The PRD R-SR1 amendment is recorded in the v2 PRD.
- **M3: judge plus the code evaluator.** It starts only after the sandbox namespace's default-deny policy, admission control (a bounded global queue plus per-account run and submit limits) and the runner are live.
  - DSA items **that have hidden tests** show Run/Submit.
  - Every other item keeps the picker.
  - The arena gets Run/Submit that never counts.
  - The grading strategy is table-tested against the v1 copy.
- **M4: platform AI.** `analyzer` pre-fills mistakes; the dispute window infrastructure lands. It degrades to "no diagnosis" when unavailable.
- **M5:** when every seeded DSA item has tests, set DSA to `evaluator_only` for outcomes. Give-up means miss.

The env override is the kill switch. The owner's account and 19 events survive every step: no ids are renamed and the version-1 path default covers replay. Because Flux image automation re-applies the newest tag within about a minute, releases **roll forward**; reverting a bump is not a rollback.

## 10. What T0 constrains downstream

- **T1 (content and data)**
  - Must model: the manifest (content-hashed); items with role, topic, parts (widget, evaluator step with `inputs[]`, cadence, weight), per-language starters and harness references, and `revision.probes[]`; course assets split into a public descriptor and a private companion; and a **private eval pack** (hidden tests, *answer keys*, expected result sets, rubric anchors) that is never `problem_section` rows and never readable from the public repo, images, curriculum GETs or the BFF.
  - Ids are global, never re-parented, and retired rather than deleted. DSA keeps its bare ids and new courses use prefixes.
  - `path_slug` goes on every per-user row and every event.
  - Must define the attempt `purpose` and `mock_session_item`.
  - Public-profile changes: per-course visibility, where each grade came from, and passed shown separately from attempted.
  - Decides the delivery mechanism for the private eval pack (decided in T1), and whether a spec hash per item lets judge downgrade mismatched items to self.
  - Needs a sanitizing Markdown renderer with images, tables and fenced code.
- **T2 (storage)**
  - Stores judge-owned artefacts by key (drafts, submissions, canvas scenes, large test inputs and datasets, audio later).
  - Events carry references only; the NATS payload limit is 1 MiB.
  - Payloads over 1 MiB need an upload path that bypasses the gateway.
  - Off-node backup must cover the judge schema and the eval pack's source. If that source is a private git repo, git is its backup.
  - Needs a per-account erase path across practice, judge, review, assessment and coach.
- **T4 (judge contract)**
  - Request: `{submission_id (idempotency), account, item, path_slug, context{kind: course|touch|mock|arena|run, id}, action: run|submit|final|give_up, parts[{part_id, language?, payload_ref}]}`.
  - Result: `{status: passed|failed|inconclusive|error, checks[{key,required,met}], score{value,max}?, per-part results, submitted_at, feedback_ref, evaluator/runner/rubric/model versions}`.
  - Enqueue is synchronous and the client polls or uses SSE. Evaluations are first-in-first-out per attempt.
  - Only counted contexts emit events, with the mode in the subject or payload for filtering.
  - AI never flips a deterministic verdict.
  - Run is declared per kind: `code` runs on samples; `key` and `ai_rubric` have no run.
  - `analyzer` is a separate, later fact.
  - Must specify draft ownership (judge) and the provisional/dispute state.
- **T3 (sandbox)**
  - Profiles: `go`, `go-race`, `sql-pg`, then `cpp`, chosen by evaluator config and never by course.
  - Hidden-case feedback is pass/fail and resource usage only. Verdicts are computed outside the untrusted process where the harness allows it.
  - Must report inconclusive and flaky runs.
  - Runs in its own namespace with default-deny policy, admitted only from judge.
  - Budget: 2 or fewer concurrent jobs, at most 2 vCPU / 3 GiB of limits.
  - Must work with no KVM and with AppArmor restricting user namespaces.
  - Learner SQL never touches CNPG.
- **T5 (platform AI)**
  - The platform key is used only for guided course and touch evaluation and analysis, never in the arena or Run.
  - Split the calls: the scoring call sees private anchors and returns strict integer or enum JSON; the feedback call sees only the learner's work plus public content they are entitled to after the gate.
  - Output is clamped, input is capped, and learner content is delimited as data.
  - Budget exhaustion degrades to self or to no diagnosis and never blocks the loop.
  - Behavioral answers need consent.
  - The key holder is decided in T5 (candidate: judge). No unauthenticated internal endpoint may spend money.
- **T6 (interviewer)**
  - Reads the rail and rubric from the manifest and scores through `ScoreMock` with `scored_by=ai` (AI first, then one learner dispute).
  - Runs on the BYO key.
  - The session transcript is the primary artefact, kept out of assessment and off the public route.
  - Media never passes through gateway body or timeout limits.
- **T7 (rollout)**
  - Order: M0 → M5, and the second course only after M3.
  - Required first: NetworkPolicy for the sandbox namespace, then the xlearn, databases and messaging namespaces as a separate step with egress exceptions for identity and coach. Admission control and rate limits must exist before any evaluator reaches open signup.
  - Chart changes, off by default: initContainers or pull secrets, `automountServiceAccountToken`, `runtimeClassName`, `priorityClassName`, NetworkPolicy, and separate readiness and liveness paths.
  - ADRs:
    - course manifest and plug-in model
    - judge context and learning signal v2
    - sandbox namespace
    - LLM key holder
    - PRD amendment to R-SR1
  - Pick a release label other than "v2" (commit `d531967` already uses it). A `v2.0.0` tag auto-deploys under `>=1.0.0`.
  - Needs v2 artboards for the workspace widgets and the course nav.

## 11. Alternatives considered

| Option | Why not |
|---|---|
| Per-course Go and TS plugin modules (course-as-code) | Common screens drift per course, and logic ends up per course. Variation belongs to part type and evaluator kind, not course. |
| Manifest fetched at runtime from curriculum by 5–6 services (evolve-v1's `course.Client`, course-diversity's `GET /definition`) | Puts curriculum on the write path and needs version history, caches and fakes. Every tag already rebuilds all 7 images (`deploy.yml`), so embedding costs nothing. |
| Private content repo, two OCI images, capability majors, `requires:`, snapshots, cross-repo CI (arch-ops-security) | For one author, it mainly manages the version skew it creates. The revert gets re-bumped by Flux. Only the eval pack needs to be private. |
| Each context owner maps its own signals: practice, review, assessment (arch-ops-security) | Builds the timed-attempt runtime and the mapper three times. |
| judge sends revision verdicts straight to review (evolve-v1) | review would become an interpreter of part roles that knows about courses. |
| New `attempt_evaluated` subject with dual emission during migration (course-diversity) | Not atomic, since review acks unknown subjects. `problem_solved` v2 additive does the job. |
| Renaming DSA ids to `dsa:16`, or a composite `(path_slug,id)` key | Replay reads bare ids from an immutable log; a composite key would ripple through 4 schemas and every event. |
| Boot-time route generation, a public `/api/paths`, a `/:seg/*` resolver alongside `/:username` | Superseded by #46. Anonymous visitors cannot read the session-gated catalog. |
| Judge inside practice; AI grading inside coach; one shared "platform-ai" service | Would mix untrusted-code orchestration and AI egress with learning state, or blur the BYO and platform key paths. |
| Auto-conclude sweeper that turns abandoned attempts into misses (all three candidates) | Changes the live loop, fabricates heatmap and streak activity, and races in-flight evaluations. |
| Per-course ladder intervals and step-back failure policy (course-diversity critique) | Weakens the product thesis. Formats per level band plus pass thresholds per course handle cost and AI noise. |
| A banding "policy" expressed as data, or expressions or scripts in the manifest | Becomes a DSL. The closed Go strategies are parameterized by data instead. |
| Quarantining bad bundles at runtime; JSON-Schema capability manifest | CI validation of embedded content is enough. Quarantine hides errors and is a code path that is hard to test. |
| Judge work as a NATS work queue | 25 s handler and 30 s AckWait limits, and the broker would sit within the sandbox's reach. |
| Go plugins or WASM loaded at runtime | Fragile, and a new attack surface. |

## 12. Owner decisions — resolved 2026-09-24

| # | Question | Decision |
|---|----------|----------|
| D1 | How does a course plug in? | **Settings-only course file compiled into every image, plus closed code registries keyed by part type or grader kind** (recommended). |
| D2 | What gets spaced-revision touches? | **Every concluded, counted attempt (any grade, including give-up) anchors L1–L5, and below-clean also opens a mistake.** This is universal. New courses start on this rule. DSA switches in a later release, after M1 ships with no behaviour change (M2c). |
| D3 | How much of the method is universal? | **The Day 1/3/7/21/45 ladder and the 4-grade scale are universal.** The touch format per level band, timers, grading strategy and thresholds, mistake taxonomy (a universal core plus per-course categories) and mock rubric and rail are per course. |
| D4 | Today and budget across courses? | **One budget per account, in minutes.** Due touches from all courses come first. R-SR5 blocks new work per course. The remainder splits across enrollments. |

**Parked to later topics** (candidate recommendations, not yet decided):
- Who finalizes an AI-graded outcome (provisional plus one dispute), and what happens to abandoned attempts (they stay resumable; only an explicit give-up concludes): **T4**.
- Where the platform LLM key lives (candidate: judge, with provider code in a shared `platform/llm`): **T5**.
- Where public content and the private eval pack live (candidate: public in this repo, eval pack private and delivered only to judge): **T1**.

## 13. Critic findings addressed

- **The outcome rule contradicted v1 and skipped re-implement** (blocker). The `verdict_timer@1` strategy matches `Problem.tsx:21-26`: hint means assisted. Re-implement gates conclusion. A golden table test pins it (§4).
- **Answers were exposed during counted contexts** (blocker). The arena and item surfaces withhold solutions and topic for live items. Recall is free text, not a multiple-choice question over a field already on screen (§7).
- **Conclusion, cadence and races were undefined** (blocker). The attempt state machine, part cadence, latest submission per part, first-in-first-out evaluation per attempt, the three-state verdict and no sweeper (§5).
- **Routing conflicted with #46** (blocker). Builds on ADR-0025 (`/u/` profiles). One `/:course/*` route inside AuthedShell, plus the course-slug guard. Course slugs are not reserved as usernames (§7).
- **The every-conclusion ladder broke parity** (blocker). An owner-approved toggle at M2c, shipped with the heatmap and mastery redefinition (§9).
- **Async callers had no service auth** (blocker). judge holds the platform key; review never fetches prose; runner ingress is from judge only (§8, candidate; decided in T5).
- **The runtime manifest fetch** (fatal and major). Manifests are embedded at compile time and snapshotted onto rows when created.
- **Answer keys sat in learner-facing config** (fatal). Every field that carries an answer is private eval-pack data.
- **Renaming ids with a replay, and dual emission** (fatal). Bare DSA ids are kept, `problem_solved` v2 is additive, and consumers ship before producers.
- **Diagnosis sat in the finalize event, and prose was in events.** Split into `evaluation_analyzed` carrying a category plus a reference.
- **Revision mapping made review course-aware.** practice emits a normalized `touch_concluded`.
- **Touch probes per item.** Item `revision.probes[]`, counted in the authoring budget.
- **Mock multi-item sets and mock evaluation.** `mock_session_item`; mock evaluations are evidence only; mocks open no mistakes.
- **Shared datasets and assets.** Course asset concept (public descriptor plus private companion).
- **LLD's fixed API.** Each item's evaluator block chooses facade tests or AI only.
- **Mastery ignored touches; heatmap inflation.** `touch_scored` feeds both.
- **Early-reveal events lacked a path.** Every subject gains `path_slug`; the default only applies to version-1 envelopes.
- **Flaky verdicts.** `inconclusive` never counts.
- **Behavioral story bank and personal data.** Scoped honestly as a partial fit: a later ADR, consent, not a pilot.
- **The kill switch was not runtime.** Env override in the HelmRelease.
- **AI prompt injection and leaks.** Split scoring and feedback calls, clamped output, AI grades show where they came from on public stats.
- **Hidden tests leaking via the sandbox.** Verdict-only hidden feedback; verdicts computed outside the process (T3 and T4).
- **Drift between content, spec and runner.** Spec hash cross-check that downgrades mismatched items to self (T1).
- **Event fan-out and retention.** Only counted contexts emit; a publisher retention option.
- **Open signup could exhaust CPU or budget.** Admission control is a prerequisite for M3.
- **A big-bang release.** M1a/b/c expand, backfill, contract.
- **Evaluator steps spanning several parts.** Step `inputs[]`.
- **`is_reinforcement` was dropped; frontier semantics.** Item `role`; the frontier counts concluded core items.
- **Budget in minutes versus planning by count.** `est_minutes` in the manifest (decision D4).
- **AI noise resetting the ladder.** Touch pass threshold per course, plus the dispute window.
- **Disputes arriving after emission.** The dispute happens before conclusion.
- **Mistakes that could never close.** Every concluded attempt now schedules touches.
- **Grading logic dressed up as data.** Closed Go strategies.
- **Delete-missing orphaned history.** Retire, don't delete.
- **Language chosen at submit.** Runner profile keyed per language.
- **Missing chart knobs; NetworkPolicy ordering; PriorityClass assumptions.** Listed under T7; sandbox default-deny first; hard limits.
- **Relay latency for Run.** Synchronous enqueue plus polling; events only for finalization.

## 14. Risks

- **Content is the bottleneck, not code.** DSA has 14 of 151 problems seeded and none has tests; about 345 items across six courses need tests, keys, anchors and touch probes. Build each widget or adapter only when its first course's content is being authored.
- **Manifest creep.** Pressure to add conditionals. Hold the line: closed strategies and a new Go primitive (plus ADR) for any new logic.
- **AI grading variance and cost** on SD and behavioral grades that drive scheduling. Dispute windows and calibration help but do not remove it.
- **Leaks of hidden or private material** through a bundling mistake. The repo and GHCR images are public. Needs a CI lint that the public pack contains no answer-bearing fields.
- **Honesty is protected against accident, not against a determined cheat.** Solutions are public on GitHub. Public stats must show where each grade came from.
- **Single-node contention.** `-race` and SQL sandboxes compete with CNPG (no CPU limit, 1 Gi) on 4 vCPU with no swap. Nothing is backed up today.
- **More review load.** With every conclusion scheduling five touches, due reviews could crowd out new work on Today even with minute budgets.
- **practice's first consumer.** Races between self and judge conclusions depend on the first-wins idempotency on the attempt row.
- **Behavioral fits only partially,** and the story-bank ADR may later force a spine change in review.
- **Relaxing CHECK enums** moves integrity into manifest validation, and a manifest bug could write unknown categories.
- **Released images always move forward,** so an operational mistake is fixed forward, not reverted.
- **judge could grow into a god-service** (queue, submissions, packs, AI). Adapters sit behind interfaces so it can be split later by ADR.