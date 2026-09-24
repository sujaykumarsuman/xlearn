# ADR-0026 — Per-course extensibility: course manifests, capability registries, one learning-signal waist

- **Status:** Accepted (2026-09-24, v2 build-plan sign-off)
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0005](0005-data-ownership-and-migrations.md),
  [0012](0012-curriculum-content-model-and-seeding.md), [0015](0015-five-touch-scheduler-model.md),
  [0016](0016-mistake-journal-and-worker-service-auth.md), [0017](0017-mock-model-and-projection-consumer-scaffold.md),
  [0018](0018-progress-projection-grain-and-rebuild.md), [0025](0025-public-profiles-under-u-prefix.md).
  v2 topic T0: [feasibility § T0](../v2/feasibility.md#t0--guiding-frame) and the full
  [research appendix](../v2/research/t0-extensibility-frame.md).

## Context

v2 turns xLearn's self-reported loop into an evaluated one (a code judge, AI evaluation, a live
interviewer) and opens the other five catalog courses: `system-design`, `go-concurrency`, `lld-ood`,
`sql`, `behavioral`. Their navigation, solving ground and evaluation differ. Today, Revision, Mistakes,
Mock and Progress stay common.

v1 hardcodes the DSA course throughout:
- **Global ids that re-parent.** `problem.id` and `concept.slug` are global, and the seed upserts move rows between paths on conflict.
- **`"dsa"` literals** in 8 gateway call sites and the static `dsa/*` SPA routes.
- **The DSA method as DB `CHECK` enums:** stages, the 4 outcomes, the 8 mistake categories, and the 7-dimension `/35` rubric.
- **No course on events.** No event or review/assessment row carries a course.

v1 also has a method hole. The ladder is scheduled only on a **first clean** solve (`review/store/store.go:307-309`), and `solved` is terminal. So a rough, assisted or miss first solve never gets revision touches, and its mistake entry can never close (closing needs two clean revisits).

## Decision

### 1. A course is data; a capability is code

- **Course manifest.** Each course has a settings-only file, `curriculum/<slug>/course.json`, holding:
  - nav labels and visibility, and the item noun;
  - stage timers, labels and penalty days, and the re-implement policy;
  - the grading strategy and its thresholds;
  - the revision touch format per level band;
  - the mistake taxonomy;
  - the mock rail, rubric, targets and item pools;
  - `est_minutes` per format;
  - public-stats picks, the coach persona, and the self-report policy.
- **No logic in the manifest.** It carries no expressions or scripts.
- **Compiled in.** It is embedded in every service image at the release tag and validated in CI. It is content-hashed and **snapshotted onto each row** that acts on it (attempt, mock session). It is never fetched at runtime.
- **Closed code registries, keyed by type, never by course slug:**
  - SPA widgets by **part type**: `code`, `text`, `choice`/`blank`, `canvas`; `audio` reserved.
  - Result views by **grader kind**.
  - judge graders by **kind**: `code` (with a T3 runner profile), `key`, `ai_rubric`, `composite`, plus an optional `analyzer` step.
  - practice **grading strategies**: a closed set written in Go (e.g. `verdict_timer@1`, `rubric_pct@1`, `weighted_gate@1`, `self`). A course picks one and sets its thresholds.
- **Cost of a new course:** content plus a manifest if it reuses existing kinds. A new part type, grader kind or runner profile needs code and a release, plus an ADR if it adds a new trust surface.

### 2. Concepts

| Concept | Definition |
|---|---|
| **Course** | The `curriculum.path` slug. It is the enrollment unit and the URL segment, and it owns one manifest. |
| **Item** | The v1 `problem`, generalized. Its id is **globally unique**, never re-parented, and retired rather than deleted. DSA keeps its bare ids; new courses use prefixed ids (`sd-001`). An item has a `role ∈ {core, reinforcement, drill}`, a topic, stage-gated public sections, and one or more **parts**. |
| **Part** | A typed answer slot. It selects a widget, a grader step and a cadence: `iterate` (run or submit many times) or `final` (evaluated once, at conclusion). |
| **Private eval pack** | Hidden tests, answer keys, expected result sets and rubric anchors. **Only judge reads it.** It is never a `problem_section` row, and never in the public repo, the images, curriculum's API or the BFF. |

### 3. The narrow waist: practice is the only writer of learning signals

- **judge (new service; its details are T4 and T3)** produces **evidence only**: an Evaluation with `passed | failed | inconclusive` plus checks and a score. It never decides grades or ladders.
- **practice owns every timed attempt**, both `purpose ∈ {course, touch}`, so the attempt runtime and the grading mapper exist once. practice concludes an attempt exactly once and writes one signal in the same transaction:
  - `xlearn.practice.problem_solved` **v2** for course attempts. This is additive only: `path_slug`, `attempt_id`, `graded_by`, `passed`, `score`, `criteria[]`, evaluation refs and `policy_version` are added.
  - `xlearn.practice.touch_concluded` (new) for revision touches.
- **Mocks** still score only through `ScoreMock`.
- **Downstream is course-blind.** review, assessment and the gateway's Today/Roadmap/Progress never read parts, widgets or grader kinds. Events carry categories, numbers and refs, never prose or code.
- **Every practice and review event gains `path_slug`.** Consumers treat a missing path as `dsa` for version-1 envelopes only.

### 4. Universal vs per-course method

| Aspect | Universal | Per course (manifest) |
|---|---|---|
| Revision ladder | Day **1·3·7·21·45**; a fail resets to Day 1 | Touch **format** per level band (e.g. L1–3 recall, L4–5 the full format under mock conditions) and its timers |
| Outcome | The **4-grade scale** (clean/rough/assisted/miss), plus a `passed` flag and an optional normalized score | The grading strategy (from the closed set) and its thresholds. DSA's strategy reproduces v1 exactly, pinned by a golden test |
| Stages | Roles attempt → hint → solution → re-implement → conclude; R-PF2's owed Day-3 touch | Durations, labels, penalty days, re-implement required/optional/off |
| Mistakes | Mechanics (one open entry per item, close after 2 clean revisits); a core set `{misread, time_management, communication}` | Additional categories, validated against the manifest (replaces the DB `CHECK`); ids are append-only |
| Mock | Lifecycle (server timer, score-once, outbox) | Rail, rubric (N dimensions × 1–5, stored as `total`/`max_total`, snapshotted on the session), targets, item pools; a course may have no mock |
| **Revision entry** | **Every concluded, counted attempt on a revisable item, of any grade including miss or give-up, anchors L1–L5. A below-clean grade also opens a mistake.** Arena, ahead-of-schedule and mock attempts never schedule | Whether drills are revisable |

**Revision entry amends PRD R-SR1** (first clean solve only). New courses start on the new rule. **DSA switches in a later release**, once the M1 restructure has shipped with no behaviour change. This also closes the v1 hole described under Context.

### 5. Screens, routing, Today

- **Routes.** Courses live at `/xlearn/:course/…`: one dynamic route inside `AuthedShell`, resolved against the session-gated catalog, with the same segments for every course and only labels differing. Profiles stay at `/xlearn/u/:username` (ADR-0025). **Course slugs are not reserved as usernames.** Instead a **course-slug guard** rejects, at seed time and in a test, any slug equal to a static top-level SPA segment (`u`, `auth`, `settings`) or a gateway-owned path (`api`, `assets`, `healthz`, `readyz`, `.well-known`).
- **Scope of each screen.**
  - Course-scoped: Roadmap, Week, Concept, Problems, the workspace, Revision (with an all-courses toggle), Mistakes and weak area, Mock, Progress.
  - Account-wide: Catalog and the agenda, Settings, onboarding, and the public profile header (streak and heatmap as account activity).
- **Today and budget.**
  - **One budget per account, in minutes**, using each manifest's `est_minutes`.
  - Due touches from **all** courses come first.
  - R-SR5 blocks new work **per course**.
  - The remainder splits across active enrollments.
  - The planner is a deterministic pure function in the gateway.

### 6. Migration (the v1 manual loop coexists throughout)

| Step | What lands |
|---|---|
| **M1** | The course spine with **no behaviour change**: manifest plus golden test, the per-course seed loader and id guard, `path_slug` everywhere, course-scoped routes and API with DSA aliases. Schema changes follow expand → backfill → contract. |
| **M2** | practice touch attempts, and the heatmap and mastery rebuilt from completed touches. **DSA switches to the new revision-entry rule in this release.** |
| **M3** | judge plus the code grader, but only after the sandbox isolation and admission limits are live. Items without tests keep the self path. |
| **M4** | The platform-AI analyzer. |
| **M5** | DSA becomes evaluator-only. |

- **Consumers ship before producers.** Consumers of a new subject ship one release before its producer, because review acks unknown subjects.
- **The second course follows M3.**

## Consequences

- ✅ A course that reuses existing kinds is content and a manifest only. The common loop never branches on course.
- ✅ Replay stays deterministic: bare DSA ids are kept, and the v2 envelope carries its own `path_slug`, so no enrichment is needed (ADR-0018 holds).
- ✅ Delivers "revision even for unsolved attempts" and closes v1's stuck-mistake hole.
- ✅ No new infra for the frame itself: manifests ship in the images. The only new process is `judge` (plus the T3 runner). The estimate is about 2.2 vCPU / 2.9 GiB requested out of 4 / 15.
- ⚠️ **Integrity moves from DB `CHECK`s to manifest validation.** A manifest bug could write unknown categories, so CI validation and golden tests are mandatory.
- ⚠️ **Manifest creep.** New logic needs a new Go primitive plus an ADR, never a manifest conditional.
- ⚠️ **practice gains its first consumer.** Conclusion must be first-wins and idempotent on the attempt row.
- ⚠️ **More review load.** Every concluded attempt now schedules five touches, so Today's minute budget has to absorb it.
- ⚠️ **`behavioral` fits only partially.** A learner-owned story bank does not fit the item model and needs a later ADR, along with consent and erase paths. It is not a pilot course.
- ⚠️ **Content is the real bottleneck.** 14 of 151 DSA items are seeded and none has tests. Build each widget or grader when its first course's content is authored.

## Alternatives considered

| Option | Why not |
|---|---|
| **Per-course Go/TS plugin modules (course-as-code)** | Common screens drift per course and logic duplicates. Variation belongs to part type and grader kind, not course. |
| **Manifest fetched at runtime from curriculum** | Puts curriculum on every write path and needs history, caches and fakes. Every tag already rebuilds all images, so embedding is free. |
| **Per-course ladder intervals or outcome scales** | Weakens the method thesis. Projections, mastery weights and the mistake allow-list all key on the 4 grades. Formats per level band and per-course thresholds absorb the real differences. |
| **Rules or expressions in the manifest (a DSL)** | Becomes a programming language to test and secure. Closed Go strategies parameterized by data instead. |
| **judge writes review or assessment directly; each context maps its own signals** | review would have to interpret parts, and the attempt runtime would exist three times. |
| **Composite `(path_slug, id)` keys or renaming DSA ids** | Would ripple through 4 schemas and the immutable event log. |
| **Keep v1's first-clean-only revision entry, or make it per course** | Doesn't meet the "even unsolved" goal, leaves the stuck-mistake hole, and makes the core method differ per course. |
| **Reserve course slugs as usernames** | Rejected by the owner (ADR-0025 update). Profiles live under `/u/`, so only the course side needs a guard. |
