# Sprint m4-03 — Analyzer (D16/D26), pointer notes, `evaluation_analyzed` + acceptance-set harness

> **Milestone:** M4 — platform AI (owner cohort)   ·   **Track:** product (order 62)
> **Prereqs:** [m4-02](sprint-m4-02.md) (Scorer, ledger, llm lane, caps, gate) · [m3-10](sprint-m3-10.md) (review `category_source` + `last_refreshed_by_attempt_id`, live in v1.14.0)   ·   **Unblocks:** [m4-04](sprint-m4-04.md) · [m4-06](sprint-m4-06.md) (AB17 pointer notes UI)
> **Release action:** merge only (ships **dark** in **v1.16.0**, tagged by [m4-07](sprint-m4-07.md)) + **one infra ACL PR** of its own, merged before v1.16.0. The `evaluation_analyzed` producer stays dark behind `LLM_PLATFORM_ENABLED=false` until m4-07's cohort flip.
> **Calendar:** December 2026 (after v1.15.0). The owner labels the acceptance set (`ev-acceptance-set`) before [m4-01](sprint-m4-01.md); this sprint only needs its schema.
> **Execute with:** [`../prompts/prompt-m4-03.md`](../prompts/prompt-m4-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | judge + review migrations (`analysis`, `pointer_note`, `review_fingerprint`; `revision_item.improvements_*`, `optional_revisit`) | X | ⬜ |
| 2 | Normalized-code fingerprint `fp@1` (D26 "materially different") | X | ⬜ |
| 3 | Analyzer durables, eligibility and skip rules | X | ⬜ |
| 4 | `analyze` jobs on the llm lane: inputs, analyzer-side checks, results, pointer notes, `judge admin analyses` | X | ⬜ |
| 5 | `evaluation_analyzed` event + review `review-analysis` consumer | X | ⬜ |
| 6 | Learner reads (pointer notes, attempt analysis, "correct, with improvements", optional revisit), coach review context, erase coverage | X | ⬜ |
| 7 | Acceptance harness `cmd/judge-eval` + synthetic set | X | ⬜ |
| 8 | Topology/registry, e2e, docs, ADR | X | ⬜ |
| 9 | Infra ACL PR (re-rendered golden), merged before v1.16.0 | I | ⬜ |
| 10 | Dev-split tuning run on the owner's machine (optional here; required before m4-07's test run) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M4 milestone row + the infra PR list + decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m4-02](sprint-m4-02.md) merged: `internal/judge/ai` `Scorer{Reserve, Score, Feedback, Analyze}` (with `Analyze`'s schema, validator and `pass_review`); the `llm_call` ledger; the llm lane (queue cap 8); L17 caps, pace gate and breaker; the gate inside `Reserve` (`LLM_PLATFORM_ENABLED`, owner/tester cohort, `status`, consent per purpose, `ai_disabled`, breaker, caps); the `internal/platform/consent` kind constants; the calibration/acceptance-row catalog gate and `judge admin calibration record --from <file.json>`; `httptest` fake providers for tests ([m4-01](sprint-m4-01.md)'s convention: fakes are never committed as compose services); identity `/internal/accounts/{id}` returning `role`, `tier`, `status`, `consents` (`ai_disabled` is judge-owned, `llm_account_limit`)
- [ ] AB17 frozen ([ds-m4-01](sprint-ds-m4-01.md)): `design-system/screens/v2/AB17-pointer-notes.html` and AB16's F2 (the analyzer suggestion on a deterministic grade) — their frames define the read endpoints below
- [ ] review `mistake_entry.{category_source, category_suggested, concepts, concepts_source, last_refreshed_by_attempt_id}` live ([m3-10](sprint-m3-10.md), v1.14.0)
- [ ] `problem_solved` v2 carries `attempt_id`, the grade, `anchor_at`, `revisable`, `language`, `gave_up`; `touch_concluded` carries `attempt_id`, `revision_item_id`, `touch_passed`, `evaluation_ids[]` ([m2-05](sprint-m2-05.md), [m3-08](sprint-m3-08.md)). If either lacks `attempt_id`, stop and report
- [ ] Migration numbering: judge/review migrations are serialized m4-02 → **m4-03** → m4-04 → m4-05; no peer PR is open on `internal/judge/store/migrations` or `internal/review/store/migrations`
- [ ] The acceptance-set template is merged in `xlearn-evalpack` ([mi-12](sprint-mi-12.md) task 7): `acceptance/schema/label.schema.json`, `acceptance/{dev,test}/`

## Goal

Give judge its **analyzer**: on every concluded counted attempt it listens to (judge's own durables on `problem_solved`
and `touch_concluded`, never an HTTP trigger), decide whether and how to analyze; run one Sonnet 5 call on the llm lane
under the L17 caps; validate the output; store it; and publish **`evaluation_analyzed`** so review can pre-fill the
mistake category and concepts and mark a revision item **"correct, with improvements"**. The analyzer runs on
**passes too** (D16) under D26's scope: every **course pass**, and a **touch pass only when its normalized code
materially differs** from the last reviewed solution. Pass reviews produce learner-private **pointer notes** per
(account, item). Also build **`cmd/judge-eval`**, the harness that measures the analyzer against the owner's labelled
acceptance set (dev/test split); m4-07 uses it to size caps. Everything ships dark in v1.16.0: with
`LLM_PLATFORM_ENABLED=false` the durables ack and skip, so no LLM call is made and no `evaluation_analyzed` is emitted.

## Scope

**In**
- judge: two durables on `XLEARN_PRACTICE` (`judge-analyzer-solved`, `judge-analyzer-touch`, `DeliverNew`), eligibility
  and skip rules, `analyze` jobs on the llm lane feeding m4-02's `Scorer.Analyze` (inputs, analyzer-side checks),
  `judge.analysis`, `judge.pointer_note`, `judge.review_fingerprint`, the `fp@1` fingerprint, the `evaluation_analyzed`
  outbox event, learner reads for pointer notes and per-attempt analyses, erase coverage, `judge admin analyses`
  ([t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls),
  [§7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences),
  [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only),
  [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5),
  [ADR-0031 §6](../../adr/0031-platform-ai-and-two-tier-keys.md#6-scope-of-ai-review-of-passing-solutions-owner-d26-refines-d16)).
- review: the `review-analysis` durable on `XLEARN_JUDGE`, the category/concepts precedence, `revision_item.improvements_*`,
  the `improvements` flag on the revision read, the off-ladder `optional_revisit` (AB17 F3).
- gateway: `GET /api/problems/{id}/pointer-notes` and `GET /api/attempts/{id}/analysis` (withheld under `withhold()`),
  `improvements` + `optional[]` on `GET /api/revision/due`, the optional-revisit routes, and in the coach's review-mode
  context the pointer notes plus the xLearn AI feedback (the analysis `summary` and `next_step`).
- `cmd/judge-eval` + `internal/judge/ai/eval` + a synthetic set; the ACL golden re-render + its infra PR.
- judge `internal/judge/ai` (m4-02's package, additive): `Admission.PassReview` and the gate keying on it; the consent
  refusal naming the missing kind.

**Out**
- Labelling the ≥ 70 examples → owner event `ev-acceptance-set` (before [m4-01](sprint-m4-01.md)).
- Running the **test** split, choosing the analyzer effort/`max_tokens` for production, sizing L17 caps, the canary log
  test, `LLM_PLATFORM_ENABLED=true` → [m4-07](sprint-m4-07.md).
- The write path for the AI consents (`PATCH /api/me/consents`), judge's account-cache refresh on it, the allowance route
  → [m4-05](sprint-m4-05.md). Until then no consent row of these kinds exists on prod, so the analyzer is **fail-closed**
  (runs for nobody); the e2e seeds rows directly.
- Provisional grades, dispute, re-grade, claims → [m4-04](sprint-m4-04.md).
- The AB16 F2 / AB17 UI (analyzer suggestion card, collapsed notes, "correct, with improvements" chip, optional revisit)
  → [m4-06](sprint-m4-06.md).
- Analysis of mocks and arena (they emit no signal), of archetype B/C items (the analyzer is code-only in v2.0), and
  BYO-key analysis (never in v2.0, [ADR-0031 §1](../../adr/0031-platform-ai-and-two-tier-keys.md#1-who-holds-what)).

## Tasks

### 1 · Migrations [X]

Next free goose version in each service (serialized after m4-02; CI fails on a duplicate). Expand only
([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
Tables that hold AI prose are **body tables: PK/FK constraints only**, validated in Go, so a failed CHECK never echoes
learner-derived text into the Postgres log ([t1 §4 judge](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual)).

| Schema · table | Shape |
|---|---|
| `judge.analysis` | `id uuid PK` (v7) · `attempt_id uuid NOT NULL UNIQUE` (one analysis per attempt, [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)) · `trigger_event_id` · `account_id` · `path_slug` · `item_id` · `language` · `context_kind` (course\|touch) · `revision_item_id` (touch) · `kind` (diagnose\|pass_review\|both) · `tier` (counted\|advisory) · `status` (queued\|done\|skipped\|unavailable) · `reason` (closed Go enum, below) · `category` · `confidence` (low\|medium\|high) · `concept_keys text[]` · `body jsonb` (the validated `summary`, `next_step`, `findings`, `complexity`, `pass_review`; C3 learner-private) · `improvements bool` · `revisit_suggested bool` · `baseline_analysis_id` (the review a `fingerprint_same` skip deferred to) · `analyzer_v`, `prompt_v`, `fp_v` · `llm_call_id` · `job_id` · `created_at` · `completed_at`. Enums validated in Go (no CHECKs: prose table). Index `(account_id, created_at)`. |
| `judge.pointer_note` | `PK (account_id, item_id)` · `path_slug` · `notes jsonb` (≤ 3 × `{note ≤ 240 runes, lines [from,to] \| null}`, ≤ 8 KiB total, C3 learner-private) · `revisit_suggested` · `revisit_reason` (≤ 200) · `analysis_id` · `updated_at`. The latest done pass review **replaces** the row; a review with no pointers deletes it (the improved solution no longer needs them). |
| `judge.review_fingerprint` | `PK (account_id, item_id)` · `language` · `fp_v` · `shingles bigint[]` · `analysis_id` · `reviewed_at` — the last **reviewed** passing solution (D26's baseline). |
| `review.revision_item` | `+ improvements_at timestamptz NULL`, `+ improvements_analysis_id uuid NULL` |
| `review.optional_revisit` | AB17 F2–F3: `id` · `account_id` · `path_slug` · `problem_id` · `due_on date` · `analysis_id` · `created_at` · `done_at` · `removed_at`; partial unique `(account_id, problem_id) WHERE done_at IS NULL AND removed_at IS NULL`. **Off-ladder** (D16/D26): never a five-touch level, never counted by R-SR5 or the Today minutes budget. Drop this row (and task 6's routes) if the owner cut AB17 F3 at the freeze. |

`reason` values (Go enum, `internal/judge/ai/analyzer.go`): `platform_disabled`, `not_cohort`, `account_inactive`,
`ai_disabled`, `no_consent`, `not_code`, `no_evidence`, `blank_draft`, `starter_only`, `too_quick`, `too_large`,
`fingerprint_same`, `nothing_to_analyze`, `queue_full`, `budget_exhausted`, `no_acceptance`, `truncated`, `invalid_output`,
`provider_error`, `refusal`, `erased`. `queue_full` is m4-02's `analysis_skipped(queue_full)` (L10): it is decided **at
enqueue** in task 3's transaction and writes a `skipped` row with no job and no `llm_call`. `sqlc generate` in both
services; `sqlc diff` clean.

### 2 · Normalized-code fingerprint `fp@1` [X]

`internal/judge/ai/fingerprint` — pure, no I/O, versioned `fp@1` (any change to the lexers, `k`, `w` or τ bumps it):
1. **Lex** per language: Go via `go/scanner`; C++ and Python via a small hand lexer (comments, string/char literals,
   raw strings, numbers, identifiers, operators).
2. **Normalize:** drop comments and whitespace; every identifier that is not a keyword or in a short per-language
   builtin allowlist (`len`, `append`, `make`, `std::sort`, `vector`, `range`, `enumerate`, …) → `ID`; string literals →
   `STR`; numbers → `NUM`. Whitespace, comments and renames therefore vanish (D26).
3. **Shingle + winnow:** hash every 5-token window (FNV-64), keep window minima over `w = 4` → the fingerprint set.
4. `similarity = Jaccard(a, b)`; **materially different ⇔ `1 − similarity > τ`**, or a different language, or no
   baseline. **τ = `ANALYZER_FP_THRESHOLD`, 0.30 provisional**; the dev split tunes it
   ([task 10](#10--dev-split-tuning-run-o)); m4-07 sets the tuned value and records it with `fp@1` in status.md.

Golden tests: reformat / comment / rename-only variants of a reference solution → similarity ≥ 0.95; a genuinely
different algorithm for the same problem (e.g. sort + two pointers vs hash map) → < 0.5; per-language fixtures for Go,
C++ and Python; the function never panics on garbage input (fuzz target).

### 3 · Analyzer durables, eligibility and skip rules [X]

`internal/judge/consumers.go` binds two durables with `events.WithDeliverNew()` — **never `DeliverAll`**, which would
analyze every historical conclusion on the first flip ([t4 §2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs)):
`judge-analyzer-solved` (filter `xlearn.practice.problem_solved`) and `judge-analyzer-touch`
(filter `xlearn.practice.touch_concluded`) — two durables because v1 `Subscribe` takes one filter subject.

Per message (handler DB-only, ≤ ~100 ms): **before** the transaction read the account gate through m4-02's cached
identity client (no HTTP inside a tx); then in **one transaction**: claim judge's `inbox`, check the `erased_account`
tombstone, locate the evidence in judge's own tables, apply the table below, and insert `analysis` (+ the `job` when
enqueued) `ON CONFLICT (attempt_id) DO NOTHING`. The handler always acks; a DB error returns an error
(`NakWithDelay` → the mi-05 dead-letter hook).

| Trigger | Condition | `kind` | `tier` | Consent (m4-02's `internal/platform/consent` kinds) |
|---|---|---|---|---|
| `problem_solved`, grade ∈ {rough, assisted} | code evidence | `both` (diagnose + pass review) | counted | `ai_review_graded`; the pass-review part only if `ai_review_passing` is also effective (else `diagnose`) |
| `problem_solved`, grade = miss | code evidence, not blank/starter | `diagnose` | counted | `ai_review_graded` |
| `problem_solved`, grade = clean | code evidence | `pass_review` | advisory | `ai_review_graded` ∧ `ai_review_passing` |
| `touch_concluded`, failed, its code evaluation failed | a failed code evaluation among `evaluation_ids[]` | `diagnose` | counted | `ai_review_graded` |
| `touch_concluded`, passed, its code evaluation passed | fingerprint **materially differs** from `review_fingerprint` (task 2) | `pass_review` | advisory | `ai_review_graded` ∧ `ai_review_passing` |
| anything else | recall-only touch, failed touch whose code passed, quiz/key items, self path with no judge evidence | skip (`not_code` / `no_evidence` / `nothing_to_analyze`) | — | — |

- **Evidence:** course → the earliest passing counted submission in context `(course, attempt_id)` (the lock trigger of
  `verdict_timer@1`); a miss → the latest counted submission or close snapshot, else the latest draft. Touch → the
  submissions behind `evaluation_ids[]` (the passing one on a pass, the last failed one on a fail).
- **Skip rules** ([t5 §6 abuse controls](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)): blank draft or
  equal to the starter; conclusion < 60 s after start; learner code > **16 KiB** (`too_large`); account not in the
  owner/tester cohort (T-3), not `active`, or `ai_disabled`; consent not effective; `LLM_PLATFORM_ENABLED=false`
  (`platform_disabled`: ack, skip row, no job); the llm lane already holds **8 queued jobs** (L10: m4-02's enqueue refuses
  judge-initiated analyze jobs over the cap) → `queue_full`, decided here at enqueue, never at claim. Every skip writes a
  `status='skipped'` row (ids and enums only) with **no job and no `llm_call`**, so `judge admin analyses` and
  `GET /api/attempts/{id}/analysis` can explain why nothing ran; a `fingerprint_same` skip also records
  `baseline_analysis_id` (AB17 F6: "its notes still apply").
- **Consent** uses m4-02's gate rule (task 8 there: `ai_review_graded` for a diagnosis; plus `ai_review_passing` for a pass
  review, [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24)), made explicit
  by task 4's `Admission.PassReview`. The consumer reads it through the cached account client only to decide `kind` (a rough
  or assisted pass without passing consent is enqueued as `diagnose`) and to record a `no_consent` skip; `Reserve`
  re-applies the gate at claim, so a withdrawal between enqueue and claim still stops the call (task 4 says how a `both`
  job downgrades).

### 4 · `analyze` jobs on the llm lane [X]

`internal/judge/ai/analyzer.go`, run by m4-02's llm lane worker (P3, "never blocks", [t4 §9.1](../research/t4-judge-contract.md#91-lanes)).
m4-02 built `Scorer.Analyze` (the call, `xlearn.code_analysis@1` + `pass_review`, its validator, no truncation retry); this
task decides *what* goes in and *what happens* with the result:
- **Admission:** `Reserve(ctx, Admission{AccountID, Context, Purpose: analyze, Tier, EvaluationID, JobID, PassReview})` at
  claim, using m4-02's field names. `EvaluationID` is the evidence submission's evaluation (nil for a draft).
  **`PassReview bool` is new and additive:** it is true for `kind ∈ {pass_review, both}`. m4-02's gate
  (`internal/judge/ai`, task 8 rule 4 there) keys on it: `ai_review_graded` is always required for `analyze`, and
  `ai_review_passing` only when `PassReview` is true. A diagnose-only job is therefore never refused for a missing passing
  consent. `Reserve`'s consent refusal names the missing kind (`ErrUnavailable{Reason: no_consent, Consent: <kind>}`,
  additive). At claim:
  - a `both` job refused **only** for `ai_review_passing` downgrades: it sets `analysis.kind = diagnose` and calls
    `Reserve` again with `PassReview=false`. The job is not skipped, so the diagnosis still runs;
  - a `pass_review` job, or any job missing `ai_review_graded`, → `skipped(no_consent)`;
  - any other gate refusal → `skipped(<reason>)`;
  - `ErrBudgetExhausted` → `unavailable(budget_exhausted)`.

  Advisory work is shed first by m4-02's pace gate. Table-test rows: a rough pass with graded-only consent (enqueued as
  `diagnose` and runs); passing consent withdrawn between enqueue and claim of a `both` job (downgraded, one call, no
  `pass_review` in the request); graded consent withdrawn (skipped, no `llm_call`). ≤ 20 analyses/day and the 16 KiB input
  cap are L17 ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)).
- **Model tuple:** effort and `max_tokens` come from the **passed acceptance row** m4-02's catalog resolves (provisional
  `low` / 4,000 in tests); with no passed row → `ErrUnavailable{uncalibrated}` → `unavailable(no_acceptance)` and **no call**,
  so nothing runs on prod before [m4-07](sprint-m4-07.md) records the acceptance run. The one analyzer tunable outside
  the row is **`ANALYZER_FP_THRESHOLD`** (judge env, default 0.30 = task 2's τ), which m4-07 sets from the dev split.
- **Inputs** (built from judge's own tables and embedded public content): the learner's code (task 3's evidence), its
  language, the item's **public** statement and **public** solution section (unlocked at conclusion), the deterministic
  verdict **as the learner DTO shows it** (status, class, hidden passed/total, perf bit, first failure class, signals — no
  hidden inputs), attempt facts (late, hint used, failed submits), the manifest's mistake categories and the item's
  `concepts[]` as enums, and whether `pass_review` is wanted. **Never pack material or pack reference solutions**
  (`AnalyzeRequest` cannot import `PackView`, compile-time, plus a contract test). If m4-02's `AnalyzeRequest` lacks one of
  these fields, add it additively; the random-boundary delimiting and the character filter (U+E0000–E007F, bidi, zero-width)
  stay m4-02's.
- **Analyzer-side checks** on top of m4-02's validator: every `lines` range inside the submission's line count (only the
  analyzer knows the submission); category and concept ids exist in the manifest / the item (unknown → `""`); the category
  applies only at confidence ≥ `medium`; errors carry field paths and ids only.
- **Result write** (fenced by `claim_token`, [t4 §2.5](../research/t4-judge-contract.md#25-job-judgejob-claim-mechanics-in-9)):
  update `analysis` (enums + the validated `body`); on a done pass review, replace `pointer_note` and upsert
  `review_fingerprint`; `improvements` = at least one pointer note; write the `evaluation_analyzed` outbox row (task 5) — all in
  one transaction, after the tombstone check.
- **`judge admin analyses [--since 7d]`** (`kubectl exec`, audit row, D34's on-demand read): counts by status, reason and
  tier, p95 latency and µUSD from the ledger — never prose.

### 5 · `evaluation_analyzed` + review consumer [X]

- **Event** `xlearn.judge.evaluation_analyzed` (v2 envelope; enums, numbers and refs only, INV-8), one per **done**
  analysis (skips and failures change nothing downstream, so they emit nothing):
  `{analysis_id, attempt_id, context_kind, revision_item_id|null, evaluation_ids[], path_slug, problem_id, kind,
  category|"", confidence, concept_keys[≤3], improvements, revisit_suggested, source:"ai", analyzer_v}`. Shared fixture
  `internal/platform/events/testdata/evaluation_analyzed.v2.json` (judge's golden marshal test and review's consumer
  tests both use it).
- **review** (`internal/review/consumers.go` gains a `judgeHandler`; `store.HandleEvaluationAnalyzed`) on a new durable
  `review-analysis` (`XLEARN_JUDGE`, filter `xlearn.judge.evaluation_analyzed`), in one transaction. After the inbox
  claim, the event is applied in two independent parts:
  - claim the inbox;
  - **mistake pre-fill: only for `kind ∈ {diagnose, both}` that carries a `category` or `concept_keys`.** Only these
    conclusions open or refresh a mistake. m3-10 does it for below-clean conclusions; clean course passes and touch passes
    never do. A `pass_review` event, or a diagnosis with neither field, **skips this step**: no lookup, no retry window.
    Otherwise find the account's `mistake_entry` for `problem_id` with `last_refreshed_by_attempt_id = attempt_id`
    ([t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)).
    **No match:** if the event is < 10 min old, return a retryable error, because review may not have processed that
    attempt's `problem_solved` yet. After 10 minutes, skip the pre-fill as stale, log at INFO and continue to the next
    step. Then:
    - **category**, precedence learner > strong rule > analyzer (≥ medium) > weak rule > empty: overwrite only when
      `category_source ∈ {rule_weak, NULL}` (m3-10: weak rules fill only `category_suggested`), setting `category`,
      `category_suggested` and `category_source='analyzer'`, so analyzer categories count in the weak area;
    - **concepts**: replace only when `concepts_source ∈ {NULL, rule}` ([m3-10](sprint-m3-10.md)'s enum), never over
      `learner`; `concepts_source='analyzer'`; ≤ 3, format `<course>:concept:<slug>`;
  - **"correct, with improvements"**: keyed on `revision_item_id` + account, **whether or not a mistake matches**, and
    applied at once. `context_kind=touch ∧ improvements` → `UPDATE revision_item SET improvements_at = now(),
    improvements_analysis_id = $1 WHERE id = $revision_item_id AND account_id = $acct` (idempotent). No ladder, grade or
    pass/fail change (D26).
  - A malformed payload is logged at ERROR and acked; transient DB errors return an error (`NakWithDelay`).
  - Unit tests (`store.HandleEvaluationAnalyzed`, real PG):
    - a touch-pass `pass_review` event with no mistake entry sets `improvements_at` on the first delivery (no retry);
    - a clean course-pass event never looks up a mistake;
    - a `diagnose` event with no matching entry retries inside 10 min, then skips the pre-fill as stale.

### 6 · Learner reads + erase coverage [X]

The reads AB16 F2 and AB17 F1–F9 need. Every judge route is aud=judge, scoped to `sub` (IDOR tests); every gateway route
(`internal/gateway/judge.go`) is **presence by config** (404 unless `JUDGE_BASE_URL` is set and the account is in the cohort)
and applies `withhold()` with the same T0 §7 predicate as the pattern chip (an open counted attempt, or a due or live
touch, on that item) → `{state:"withheld"}` with no prose (AB17 F5: "Notes are hidden during this review").

| Route (gateway → judge) | Returns | Frames |
|---|---|---|
| `GET /api/problems/{id}/pointer-notes` → `GET /items/{item_id}/pointer-notes` | `{state: available\|none\|withheld, notes[≤3]{note, lines}, revisit_suggested, revisit_reason, updated_at}` | AB17 F1–F2, F5, F9 |
| `GET /api/attempts/{id}/analysis` → `GET /attempts/{attempt_id}/analysis` | `{status, reason, kind, category, confidence, concept_keys[], summary, next_step, findings[], improvements, baseline_reviewed_at}` — `reason` explains a skip or an unavailable result (`no_consent` → "notes are off", `budget_exhausted` → "paused for now", `fingerprint_same` + `baseline_reviewed_at` → "same approach as your Dec 3 solution") | AB16 F2; AB17 F4, F6–F8 |

- **Revision read:** review's due read **`GET /revisions/due`** (`handleDueQueue`, `internal/review/service.go` /
  `handlers.go`; its query in `internal/review/store/queries/revision_item.sql`) gains `improvements: bool` per item (from
  `improvements_at`) and a separate `optional[]` list of open `optional_revisit` rows (AB17 F3), both additive. The gateway
  serves them on the shared course-scoped handler `GET /api/paths/{slug}/revision/due`, so its DSA alias
  `/api/revision/due` gets them too ([m2-04](sprint-m2-04.md)'s rule: both on the one handler).
- **Optional revisit** (review, AB17 F2–F3). The review upstream routes are JWT, owner by `sub`, registered in
  `internal/review/service.go` with handlers in `internal/review/optional_revisit.go` (new) and queries in
  `internal/review/store/queries/optional_revisit.sql` (new):

  | Gateway (`internal/gateway/review.go`) | Review upstream | Does |
  |---|---|---|
  | `POST /api/problems/{id}/optional-revisit` | `POST /problems/{id}/optional-revisit {analysis_id}` | adds the revisit (idempotent on the partial unique index); `due_on` = today + 7 days in the account timezone |
  | `DELETE /api/problems/{id}/optional-revisit` | `DELETE /problems/{id}/optional-revisit` | "Remove": sets `removed_at` |
  | `POST /api/problems/{id}/optional-revisit/done` | `POST /problems/{id}/optional-revisit/done` | sets `done_at` |

  The POST needs a done pass review with `revisit_suggested`. The gateway checks it on judge's pointer-notes read
  (`state=available ∧ revisit_suggested`, else 409 `no_revisit_suggested`). Judge's upstream response carries the note's
  `analysis_id`; the gateway passes it to review as an opaque ref and strips it from the SPA DTO. Off-ladder: never
  touches `revision_item`, R-SR5 or the Today budget.
- **Coach review mode** (m1-07's mode gate in `internal/gateway/coach.go`). The `review`-mode context gains two things, and
  the gateway reads both from judge only when their state isn't withheld:
  - the pointer notes, when their state is `available`;
  - the **xLearn AI feedback**: `summary` and `next_step` of the done analysis of the item's concluded course attempt
    (attempt id from practice's `GET /state/{problemId}`, then judge `GET /attempts/{attempt_id}/analysis`), only when
    `status=done`.

  Both are the learner's own data going to the learner's own key. There is no new caller: the gateway already calls judge
  and practice. This closes m1-07's hand-off of the "AI feedback and pointer notes" review slots
  ([t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) review row). m1-07's **final submission
  (≤ 16 KiB)** slot stays unowned; it is flagged to the register, not built here.
- **Public:** the public-shape allowlist test (P10) adds `pointer`, `notes`, `findings` and `improvements` to the denylist
  for `/api/u/{username}` and `/public/stats`.
- **Erase:** judge's `XLEARN_IDENTITY` erase consumer ([m3-05](sprint-m3-05.md)) also deletes `analysis`, `pointer_note` and
  `review_fingerprint` rows for the account; review's erase consumer ([l-01](sprint-l-01.md)) adds `optional_revisit` to its
  delete list (`revision_item` columns go with their rows). Tests for both.
- `docs/architecture/openapi.yaml` gains the new routes (the drift test fails otherwise). The gateway's optional-revisit
  routes live in `internal/gateway/review.go` (review upstream, table above) under the same presence-by-config rule.

### 7 · Acceptance harness `cmd/judge-eval` [X]

A standalone binary (no image, no `deploy.yml` job; `make judge-eval`), sharing `internal/judge/ai` (request builder,
validator, fingerprint) so it measures exactly what production runs:
- **Input:** `--set <path to xlearn-evalpack checkout>/acceptance`, `--split dev|test`, `--provider fake|anthropic`,
  `--effort low-nothink|low|medium` (sweepable), `--max-tokens`, `--out report.{json,md}`. Labels follow mi-12's
  `label.schema.json` (`split`, `item`, `language`, `artefact`, `verdict`, `kind` optimal\|suboptimal\|failing,
  `expected.category`, `expected.concepts[]`, `expected.pointers[]`, `pair_of`); the item statement and public solution come
  from the embedded public content.
- **Credential:** `--provider anthropic` reads a key for the **`xlearn-calib`** workspace from `LLM_CALIB_API_KEY` on the
  owner's machine only — never the cluster, never the prod workspace, never `ANTHROPIC_API_KEY`
  ([t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used)). The `llm.Cred` is built inside
  `internal/judge/ai/eval`, so m4-01's boundary lints (credential use only from `internal/judge/ai`) stay green; if one trips,
  widen it narrowly for that package and record it.
- **Report** (`judge-eval-report@1`): category accuracy with a **bootstrap 95% lower bound** (10,000 resamples, fixed seed);
  **per-category precision and recall**; **false-pointer rate** on `optimal` examples; line-range validity; schema-valid
  after ≤ 1 retry; truncation rate; p50/p95 latency; **p95 µUSD per analysis** from usage × `platform/llm` prices; on dev,
  a **τ sweep** over `pair_of` pairs (recommended τ); pass/fail against the thresholds of
  [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (accuracy ≥ 80%, LB ≥ 70%, false pointers
  ≤ 10%, lines 100%, schema-valid 100%, truncation ≤ 2%).
- **Calibration row:** `--calibration-out row.json` writes the run as an `llm_calibration` row (m4-02's shape: purpose
  `analyze`, the config tuple, split, `n_test`, the metrics, `configs_tried`, `passed`) that m4-07 records on production with
  `judge admin calibration record --from row.json`; the harness never writes a database itself.
- **Frozen test split:** only a **`--split test --provider anthropic`** run appends
  `config_hash = sha256(model, effort, max_tokens, prompt_v, schema_v, analyzer_v, fp_v)` to
  `acceptance/test/configs_tried.log`. Such a run with a hash already in the log is refused without `--force-rerun`; the
  check runs before any credential is read. Dev runs (including [task 10](#10--dev-split-tuning-run-o)'s) and every
  `--provider fake` run never write the log, so m4-07's fake dry run can confirm it is still empty.
  `go test ./cmd/judge-eval/...` copies the synthetic set to `t.TempDir()`. A harness test asserts two things: dev and fake
  runs leave the log byte-identical, and a pre-seeded hash refuses an anthropic test run with no key needed.
- Never prints artefact code or model prose to stdout/stderr (the report holds ids, metrics and enums).
- **Synthetic set** `internal/judge/ai/eval/testdata/acceptance-synthetic/` (≈ 8 examples covering every `kind`, both
  splits, one `pair_of`) + `go test ./cmd/judge-eval/...` end to end against the fake provider.

### 8 · Topology, registry, e2e, docs, ADR [X]

- `internal/platform/events/topology.go`: durables `judge-analyzer-solved` and `judge-analyzer-touch` (stream
  `XLEARN_PRACTICE`, service judge) and `review-analysis` (stream `XLEARN_JUDGE`, service review); `XLEARN_JUDGE.Emits` +=
  `xlearn.judge.evaluation_analyzed`; registry: handled by review, listed as ignored by every other `XLEARN_JUDGE`
  subscriber whose filter captures it (identity's erase-ack durable, if its filter is `xlearn.judge.>`). Stream budget
  unchanged. Re-render `internal/platform/events/testdata/nats-authorization.golden.conf` (`-update`); `make nats-acl-test`
  (NATS 2.14, mi-05) passes with the new durables.
- **e2e** (`internal/e2e`, `-tags e2e`, real JetStream + PG; judge pointed at an `httptest` fake provider with
  `LLM_PLATFORM_ENABLED=true` in the test harness only; AI consent rows seeded by SQL fixture). A manual compose check uses a
  `-f docker-compose.yml -f <override>` kept in the session scratchpad ([m4-01](sprint-m4-01.md)'s convention; never
  committed). Cases: a **clean course pass** → pass review → a pointer
  note readable through the gateway; a **miss** → diagnose → review's mistake shows `category_source=analyzer` and concepts;
  a **renamed-only touch pass** → `skipped(fingerprint_same)` and the old notes still show; a **materially different
  improvable touch pass** → `improvements_at` set on the first delivery (no mistake entry exists, and none is waited for);
  the same flow with the flag off → no job, no `llm_call`, no event (`TestNoAnalyzedWhenDisabled`).
- Docs: `docs/architecture/events.md` (catalogue row; "Flow — conclusion → analyzer → `evaluation_analyzed` → review",
  producer dark until m4-07), `data-model.md` (the three judge tables; review's columns and `optional_revisit`),
  `services.md` (judge's analyzer durables; the harness; notes in the coach's review context), `api.md` (pointer-notes,
  attempt-analysis and optional-revisit routes; `improvements`, `optional[]`).
- **ADR** "Analyzer and pass review (M4 realization)": the eligibility table, `fp@1` and τ, one call per attempt with
  `pass_review` riding the diagnose call (`Admission.PassReview`; a `both` job downgrades to `diagnose` when only passing
  consent is gone), events only on done, `DeliverNew`, notes withheld under `withhold()`, the
  off-ladder optional revisit, `ANALYZER_FP_THRESHOLD` + the acceptance-row gate, fail-closed until m4-05. Take the
  **next free ADR number** after checking peers' PRs and worktrees.

### 9 · Infra ACL PR [I]

One `../infra` PR: paste the re-rendered judge and review user blocks from `make nats-acl-render` into
`infrastructure/messaging/release.yaml` (public keys unchanged; only permissions grow: judge's CONSUMER.CREATE / INFO /
MSG.NEXT / `$JS.ACK` on `XLEARN_PRACTICE.judge-analyzer-{solved,touch}`, review's on `XLEARN_JUDGE.review-analysis`).
A permissions change is a config reload, not a restart. **Merged before v1.16.0**
([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)
standing rule). The PR states that **no NetworkPolicy change is needed**: no new in-cluster HTTP caller (gateway → judge
and judge/review → NATS already exist). GitOps only; never `kubectl apply`.

### 10 · Dev-split tuning run [O]

Optional in this sprint, required before m4-07's test-split run. The agent prepares the command; the **owner** runs it on
his machine with the `xlearn-calib` key in the environment (never pasted into a session):
`make judge-eval SET=../xlearn-evalpack/acceptance SPLIT=dev PROVIDER=anthropic SWEEP=effort:low-nothink,low,medium`.
Output: the recommended effort, `max_tokens` (p99 thinking + visible × 1.5) and τ. Recorded in status.md; the dev split
may be re-run freely (it is not frozen). Billed to `xlearn-calib` (≈ $10–30 for the whole acceptance effort).

## Acceptance criteria

- [ ] e2e (fake provider): a course **pass and a fail are both analyzed**; review shows the analyzer category
      (`category_source=analyzer`) and concepts on the fail's mistake; the pass has pointer notes via
      `GET /api/problems/{id}/pointer-notes`, and `GET /api/attempts/{id}/analysis` explains every skip by its reason.
- [ ] Touch passes are reviewed **only when materially different** (`fp@1`); an improvable one marks the revision item
      "correct, with improvements"; a renamed/reformatted resubmission is skipped and the existing notes still show.
- [ ] Every skip rule holds (blank/starter, < 60 s, > 16 KiB, non-code, no consent, `ai_disabled`, not cohort, flag off,
      `queue_full`): table test, and **no `llm_call` row** for any skipped case. A `both` job whose passing consent was
      withdrawn before claim downgrades to `diagnose` and still runs; a pass-review event never waits on a mistake entry.
- [ ] The coach's `review`-mode context carries the pointer notes and the analysis `summary`/`next_step`, never when
      withheld.
- [ ] With `LLM_PLATFORM_ENABLED=false` the durables ack and nothing is enqueued or emitted; both durables are `DeliverNew`.
- [ ] Analyze requests carry **no pack material** (compile-time + contract test) and no tools; without a passed acceptance
      row the analyzer makes **no call** (`no_acceptance`); validator caps hold; events and logs carry no prose.
- [ ] Pointer notes and analyses are withheld under the `withhold()` predicate and absent from every public route (allowlist
      test); an optional revisit never enters the ladder, R-SR5 or the Today budget; erase removes `analysis`, `pointer_note`,
      `review_fingerprint` and `optional_revisit` rows.
- [ ] `cmd/judge-eval` runs **end to end on the synthetic set** (fake provider) and reports per-category precision/recall,
      accuracy + bootstrap LB, false-pointer rate, line validity, schema-valid, truncation, p95 µUSD and the τ sweep; it
      writes an `llm_calibration`-shaped row for `judge admin calibration record`; a repeated test-split config is refused.
- [ ] Registry, stream-budget, ACL golden and NATS-auth integration tests green; the infra ACL PR merged (or approved to
      merge before v1.16.0).

## Release

**Merge only — ships dark in v1.16.0**, the tag cut by [m4-07](sprint-m4-07.md). The review consumer and the judge
producer land in the same tag, which is safe because the producer is dark behind the T-2 kill switch
`LLM_PLATFORM_ENABLED=false` ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)):
m4-07 flips it by infra PR only after the tag, so review's durable is bound before the first event exists. Before the
tag: the task 9 ACL PR merged. No new pod, so the memory-sum rule is unaffected (judge keeps 128 Mi / 256 Mi).

## Definition of Done

CI green (`go build`, `go vet`, `go test -race ./...`, `sqlc diff`, the migration lint, the subject-registry, stream-budget
and ACL golden tests, `make nats-acl-test`, the openapi drift test, web tests untouched) · e2e green · merged to
`main` via PR (squash) · the ACL infra PR merged · statuses updated (this file + [`../status.md`](../status.md)) · the ADR
written · local `main` synced in xlearn and `../infra`.

## Risks / watch-outs

- **The analyzer on passes roughly doubles analyzer volume** — pass reviews are `advisory` and shed first by the pace
  gate, need the separate passing consent, and touch passes are fingerprint-gated (D26). Counted work keeps its headroom.
- **`DeliverAll` by mistake** would analyze every historical conclusion on the flip → a cost spike. Both durables are
  `DeliverNew`, asserted in a test on the `Subscribe` options.
- **Fail-closed until m4-05:** nothing writes the AI consent kinds before m4-05, so on prod the analyzer runs for nobody —
  expected. Do not "fix" it by defaulting consent to granted (consents must be unticked, D24/R-AI6).
- **`fp@1` too coarse or too fine** (alpha-renaming conflates similar structures) — τ is tuned on dev pairs; any change
  bumps `fp_v`, so old baselines are compared only within a version (a version change counts as "different").
- **False pointers on optimal code** (t5 §13) — gated at ≤ 10% in m4-07; notes are advisory, withheld during touches, and
  switchable off by consent.
- **Injection through code comments** — random-boundary delimiting, the character filter, the analyzer can't touch
  grade or ladder (I6), its category applies only at ≥ medium and never over learner or strong-rule categories.
- **Race: `evaluation_analyzed` before review processed `problem_solved`**: the 10-minute retryable window, then stale.
  This applies only to the mistake pre-fill of `diagnose`/`both` events; pass reviews and `improvements_at` never wait on
  it.
- **Prose leaking to logs** — validator errors are paths-only; the harness prints no code; m4-07's canary test proves it.
- **judge memory** (256 Mi limit): inputs are ≤ 16 KiB and fingerprints are small; check the working set read-only after
  v1.16.0 (m4-07).
