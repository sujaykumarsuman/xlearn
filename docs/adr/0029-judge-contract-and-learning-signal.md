# ADR-0029 — Judge contract, judge archetypes & learning-signal v2

- **Status:** Proposed
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0026](0026-per-course-extensibility-model.md) (frame), [0027](0027-content-evalpack-and-user-data-model.md)
  (content, eval pack, schemas), [0028](0028-object-storage-and-backups.md) (inline blobs),
  [0015](0015-five-touch-scheduler-model.md), [0016](0016-mistake-journal-and-worker-service-auth.md),
  [0017](0017-mock-model-and-projection-consumer-scaffold.md), [0018](0018-progress-projection-grain-and-rebuild.md),
  [0020](0020-coach-service-realization-and-behaviour-gate.md). v2 topic T4:
  [feasibility § T4](../v2/feasibility.md#t4--judge-types--the-common-contract) and the full
  [research appendix](../v2/research/t4-judge-contract.md).

## Context

ADR-0026 settled that **judge produces evidence and practice decides**. T4 defines three things:
- the common Submission → Evaluation → learning-signal contract;
- the judge types beyond code (system-design canvas plus questionnaire plus AI; quizzes);
- the plug-in seam.

Every v1 grade is self-reported, and v1's `LogOutcome` is race-unsafe: an unlocked read, unguarded updates, and no unique outcome.

T0's ordering assumption ("FIFO per attempt ⇒ the last evaluation seen triggers conclusion") breaks under JetStream nak-with-delay. The v1 events library has no `MaxAckPending` knob.

## Decision

### 1. Registries and archetypes

- **Closed registries, keyed by type, never by course:**
  - 5 part types: `code`, `text`, `choice`, `blank`, `canvas` (`audio` later);
  - 3 authored grader kinds: `code` (sandbox runner), `key` (answer key), `ai_rubric`;
  - every item has one non-nested `composite@1` root;
  - an `analyzer` runs after the fact.
- **Three archetypes are named compositions, not plug-ins:**
  - **A. Code IDE:** dsa, go-concurrency (`go-race`, honor-grade), sql (`sql-pg`).
  - **B. Structured response:** system-design, lld-ood, behavioral. `key` steps plus **one** `ai_rubric` over text, blanks, choices and an **Excalidraw** canvas (typed shapes via `customData`). Freehand is ignored. It is graded from the text export, never an image.
  - **C. Quiz/key:** single, multi and ordering choice; text, numeric-with-units, **Big-O** and predict-output blanks. Also the recall-probe engine for touches.
- **Adding a course that reuses these costs no code.** A new mode or normalizer takes days. A new part type, grader kind or runner profile takes weeks and needs an ADR.

### 2. The contract

- **Submission.**
  - Every action is an **idempotency-keyed submission**.
  - The **server** sets the context id, `contract_hash` pin, `submitted_at` and a per-context **`attempt_seq`**. Only evaluation-bearing submissions take a seq; Runs never do.
  - Actions are `run | submit | final | give_up`. Contexts are `course | touch | mock | arena`.
  - **Counted** means course, touch or mock, excluding Run. Arena submits are persisted but never counted.
- **Execution.**
  - Key-only work is evaluated **inline** in the enqueue transaction.
  - Runner and LLM work goes to a Postgres `FOR UPDATE SKIP LOCKED` **job table**. It has two lanes (runner 2, llm 2), fenced claims, lease and heartbeat, priorities with ageing, per-account caps and daily quotas.
  - There is no NATS work queue and no River.
- **Every accepted submission ends in exactly one terminal evaluation**: `passed | failed | inconclusive | error`, plus a reason enum and, for code, a class.
  - `inconclusive` needs platform-side evidence and never counts as a failure.
  - Code classes: AC WA TLE MLE OLE RE CE REJECTED RACE DEADLOCK LEAK. CE and REJECTED are free.
- **Learner-facing DTO (allowlist).**
  - **Hidden tests:** the correctness passed/total, one perf bit and the first failure class. No case ids, stderr or per-case timing. Diagnostics are shown only at learner-file positions.
  - **Samples:** full detail.
  - **Keys and rationale:** only after a counted conclusion.
  - A per-endpoint denylist test backs this.
- **Transport.** Synchronous enqueue (202), then polling with `poll_after_ms` and an ETag. SSE is deferred. Body limits use a shared `http.MaxBytesReader` helper returning a typed 413 at all 11 request-body sites.
- **Events.**
  - `xlearn.judge.evaluation_completed` fires for counted **course and touch** evaluations only (enums and refs, ≤ ~1.5 KiB). Mock evaluations stay silent in v2.0.
  - `evaluation_analyzed` comes later (M4).
- **Large performance inputs** come from a **closed, public generator registry** driven by seed and spec. The 256 KiB literal cap stays; a stamped exception allows ≤ 2 MiB. The runner accepts streamed input of ≤ 8 MiB per case.

### 3. Conclusion (practice, the single writer)

- practice stores each evaluation as a **fact**, always acks, and **pulls gaps from judge**.
- The course's closed **grading strategy** (`verdict_timer@1`, `rubric_pct@1`, `weighted_gate@1`, `self@1`) **locks** the grade only when the facts below the trigger seq are **gap-free**.
- `Conclude()` is guarded (`UPDATE … WHERE concluded_at IS NULL RETURNING` plus `UNIQUE(outcome.attempt_id)`). It emits **one** `problem_solved` v2 or `touch_concluded`. New additive fields:
  - `anchor_at` (the moment the grade was determined; scheduling and projections date from it);
  - `revisable`;
  - `resolution`;
  - `mistake_hint`;
  - `concepts_hint[≤3]`.
- **Hard time limit (owner).**
  - Starting a problem starts the server timer.
  - **No pass by the limit concludes the attempt as Miss** at the deadline, and it is re-attempted per the revision schedule (Day 1).
  - An explicit **give-up** is a synchronous practice write: the solution unlocks, and the grade is Miss unless an in-flight earlier submit passes.
  - Touches use the same hard limit. A touch that was never shown, or has only infra-inconclusive evidence, is voided.
- **DSA defaults (manifest; per-item overrides later).**

  | Setting | Value |
  |---|---|
  | Limit per problem | **45 min** |
  | Hint unlocks at | **15 min** |
  | **Clean** | pass ≤ 20 min, no hint, no coach, ≤ 3 failed submits |
  | **Rough** | pass ≤ 45 min without hint or coach |
  | **Assisted** | pass after the hint, **or with AI-coach help during the attempt** |
  | **Miss** | timeout or give-up |

  **Per-problem time budgets are a separate research session.**
- **A pass finishes the attempt.** There is no mandatory re-implement stage. After a failure the solution unlocks for study, and the Day-1 revision is the from-memory re-attempt.
- **AI-graded results are suggestions** (grade, mistake category, notes and concepts, with reasoning).
  - The learner **accepts, edits, or requests one blind re-grade**, and **submitting** triggers the downstream actions.
  - **Manual entry is the fallback** whenever AI is unavailable or over budget.
  - Edits are bounded by the deterministic ceilings and labelled `override`, `trust=honor`.
  - An untouched suggestion auto-submits after 24 h.
  - Deterministic code verdicts are not editable; mistake and notes always are.
- **The AI assessment also reviews passing solutions.** If the logic can be improved, it attaches **pointer notes to the problem** and suggests an **optional revisit**, which does not affect the grade or the ladder.

### 4. Learning loop

- **Revision.** Every concluded, counted attempt on a **revisable** item (role core or reinforcement, or drills where the manifest allows) anchors L1–L5 at `anchor_at` (D2). A new `ReanchorPendingTouches` query handles the ladder; v1's `ReanchorTouch` stays only for the fail → Day-1 reset.
- **Mistake pre-fill.**
  - judge emits closed `signals[]`: e.g. `tle_perf_only`, `wa_edge_only`, `ce_only`, `dim_low:<dim>`, `misconception:<cat>`.
  - The manifest's `mistakes.prefill` maps them to course categories.
  - Precedence: learner > strong rule > analyzer (≥ 0.6) > weak rule.
  - This switches on the weak area in M3.
- **Concepts to revise.** `item.concepts[]` plus failed-check concepts plus analyzer concepts (≤ 3). They are shown after conclusion (Mistakes, coach) and withheld while the item is live.
- **Mocks.** Evaluations are evidence only. `ScoreMock` stays the only mock signal, with `scored_by=self` in v2.0. `ai_rubric` never runs in mock.
- **Arena (owner).**
  - **Unrestricted**: no locks during live attempts or touches.
  - An early solution reveal is recorded but does not cap the course grade.
  - It has its own done marker (D10).
  - `key` and `ai_rubric` never run in arena or Run.

### 5. AI interface handed to T5

- A `Scorer{Reserve, Score, Feedback, Analyze}` interface with strict JSON schemas (`xlearn.score@1`, `xlearn.feedback@1`, `xlearn.code_analysis@1`).
- **Scoring:**
  - evidence quotes are verified against the learner's text; a required criterion without verified evidence is clamped to ≤ 2;
  - k = 3 samples, escalating to 5; pass bar ≤ 8 calls per evaluation;
  - zero retention;
  - pack material is used **only** in `Score`, never in `Feedback` or `Analyze`;
  - an injection heuristic sets `review_flag` → honor.
- **Calibration gate** per rubric × model × prompt: QWK ≥ 0.70, within-1 ≥ 90%.

### 6. Milestones

| Milestone | Scope |
|---|---|
| **M3** | judge with `code` and `key`; the code, quiz and self strategies; practice's first consumer with the pull reconciler and ticker (including deadline expiry); sync close and give-up; drafts; arena history; strong-rule pre-fill; the DTO allowlist; `MaxBytesReader`; **NATS auth (precondition)** |
| **M4** | `ai_rubric`, the llm lane, the analyzer (including on passes), suggestion review/edit and re-grade, calibration |
| **SD course** | Needs M4 and the Excalidraw widget |

The second course after M3 is go-concurrency or SQL.

## Consequences

- ✅ **One evidence engine and one learning signal.** Courses never add code unless they need a new capability.
- ✅ **Correct under redelivery, restarts and outages.** The seq plus gap-free lock plus pull reconciler make this hold, and there is exactly one conclusion.
- ✅ **v1's timing model becomes enforceable.** The server timer is a hard limit, and hint or coach use is visible to the grade.
- ✅ **AI noise can't silently reset ladders.** The learner reviews and edits every AI suggestion, and manual entry always works.
- ⚠️ **Behaviour changes vs v1:**
  - the full statement is shown at start;
  - the hard 45-minute limit;
  - revealing the solution means Miss;
  - "complexity stated **correctly**";
  - ahead-of-schedule items go to the arena;
  - using the coach caps the grade at Assisted;
  - no re-implement stage.
- ⚠️ **The arena is honor-system** (owner). It could be used as an uncounted debugger during a timed attempt.
- ⚠️ **practice gains its first consumer and a ticker.** Lifecycle golden tests are mandatory.
- ⚠️ **Volume re-modelled** to about 124 KB per active learner-day. The 60% resize trigger now comes at about 198 learner-years.
- ⚠️ **The analyzer on passes adds platform-AI cost.** T5 budgets it.
- ⚠️ **Excalidraw is less accessible than a graph editor.** The Outline view is optional.

## Alternatives considered

| Option | Why not |
|---|---|
| Archetypes as registry keys (course-as-code) | They would re-create per-course logic; archetypes are compositions instead |
| NATS work queue / River OSS | 25 s handler and 30 s AckWait limits; River's per-key sequencing is Pro-only |
| Judge per-context FIFO / broker `MaxAckPending=1` | Redundant with seq ordering, or it serializes every learner |
| Nak-and-wait on gaps | `maxDeliver=100` exhausts in about 8 h; ack-always plus pull cannot stall |
| AI grade final, or AI advisory only | No recourse, or not an evaluated loop; the owner chose suggestion plus edit plus one re-grade |
| Resumable-forever abandoned attempts | Replaced by the owner's hard time limit |
| Mandatory re-implement after every attempt | The owner: a pass finishes it; re-attempt via the schedule |
| Arena locks and a reveal cap | The owner: no restriction |
| React Flow canvas | The owner chose Excalidraw (open source) to minimise complexity |
| Raising the literal case cap to 2–4 MiB | The pack would exceed 150 MB; seeded generators instead |
| Canvas graded as an image | 15–29 points worse than text (IsoBench) |
