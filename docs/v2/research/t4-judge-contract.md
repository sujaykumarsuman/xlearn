> **T4 research appendix.** Method: four parallel research slices (contract and lifecycle; archetypes and plug-ins; evidence to learning signal; learner flows) were synthesized into one draft. The draft then faced two adversarial critiques: six-course loop correctness (6/10, 1 blocker) and integrity and ops (6/10, 2 blockers). This is the revised final. Decided in [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) (Accepted 2026-09-24).
>
> **Status:** settled with the owner 2026-09-24 (**D14–D19**, §13). **Where §13 conflicts with the body, §13 wins.** In particular:
> - the hard timer limit (D15, D18) replaces "resumable forever" and the 15 + 10 golden rows in §6.2;
> - there is no mandatory re-implement stage (D16);
> - the arena is unrestricted (D17);
> - AI results are suggestions the learner can edit before submitting (D14);
> - the canvas is Excalidraw (D19).

# T4: judge types beyond code, and the common contract

> Research and design only. No repo files were changed. This builds on D0–D13, ADR-0026/0027/0028 and the T0/T1/T2 appendices without re-arguing them.
> - **FLAG** marks a spot that refines or conflicts with a settled text. All of them are collected in §11.4.
> - **(inferred)** marks my own reasoning, where no file or source states it.
> - v1 line refs are `main@2681862`.
>
> **Re-verified this pass:**
> - projections date activity from the envelope `occurred_at` (`assessment/store/projections.go:176,198,219`)
> - `LogOutcome` also writes `attempt_logged` (`practice/store/store.go:420`)
> - `ReanchorTouch` has no pending filter (`review/store/queries/revision_item.sql:11-22`)
> - `AutoPass` is strict `< 120` s, and L4–5 are mock touches (`review/store/store.go:35,707,712`)
> - the review fail path re-anchors at `time.Now()` (`store.go:515`)
> - `openMistakeTx(…, pgtype.Timestamptz{})` is at `review/store/store.go:342`
> - the consumer uses `maxDeliver=100` with backoff capped at 5 min, about 8 h in total (`consumer.go:30,39,121`)
> - the server `ReadTimeout` is 30 s (`httpx.go:30`)
> - **11** request-body `io.LimitReader` sites exist (listed in §2.3)
> - T0 §4 says "more than N failed submits" (`t0…:111`)
> - PRD R-SR1 says "from its conclusion" (`xlearn-v2-prd.md:72`)
> - T1's `awaiting_evaluation` guard is at `t1…:784`
> - ADR-0026 has the revisable-item guard at `:78`
> - `scratchpad/t4/vol_t4.py` re-runs and reproduces §9.5

---

## 1. Recommendation in one paragraph

**judge** is a course-blind evidence engine. It has two closed registries: five **part types** (`code`, `text`, `choice`, `blank`, `canvas`) and three authored **grader kinds** (`code`, `key`, `ai_rubric`). Every item has one non-nested `composite@1` root, plus an `analyzer` that runs after the fact. Three **archetypes** are named compositions, not registry keys:
- **A: Code IDE.** dsa, go-concurrency, sql, and the LLD facade gate.
- **B: Structured response.** Text, blank, choice and a typed canvas, graded by `key` steps plus one `ai_rubric`. Used for system-design, lld-ood and behavioral.
- **C: Quiz/key.** Drills everywhere, and the universal recall-probe engine.

**Submissions.** Every action is an idempotency-keyed Submission. Its context, pin, `submitted_at` and `attempt_seq` all come from the server, and only evaluation-bearing submissions get a `seq`. Key-only work is evaluated **inline**, in the enqueue transaction. Runner and LLM work goes through a Postgres `SKIP LOCKED` job table with two lanes. Each submission ends in exactly one terminal Evaluation: an internal record, plus an allowlisted learner DTO. Counted course and touch evaluations emit `evaluation_completed`.

**Conclusion.** practice stores evaluations as facts. It always acks and **pulls gaps from judge**. It locks the grade with the course's closed strategy once the facts below the trigger are gap-free. Give-up and close are **synchronous practice writes**. practice concludes **exactly once**, emitting `problem_solved` v2 or `touch_concluded` with `anchor_at`, `revisable`, `mistake_hint` and `concepts_hint[]`. review anchors L1–L5 for every concluded counted attempt on a **revisable** item, of any grade, at `anchor_at`. It pre-fills mistakes by precedence.

**Parked Q1: who finalizes an AI-graded outcome.** A grade is provisional only when a non-authoritative grader could have cost the learner something. That means an AI rubric, or an honor-grade public key. The learner gets one **blind** AI re-grade, then an override bounded by a ceiling, or one "I meant this" claim for honor keys. It auto-accepts after 24 h and is anchored at lock time.

**Parked Q2: abandoned attempts.** An attempt is resumable forever before the lock, with no sweeper. After the lock, a 24 h settle window applies. A touch is voided only if it was never shown, or if its only evidence is platform-inconclusive.

**Downstream.**
- Performance cases become seeded specs for a closed, public generator registry, and the 256 KiB literal cap stays.
- T3 gets a runner contract: typed infra and throttle signals, measurements taken outside the process, and diagnostics mapped to file positions.
- T5 gets a `Scorer` interface with three strict schemas and hard caps on the number of calls.

---

## 2. The common contract

### 2.1 Vocabulary

| Term | Values / rule |
|---|---|
| `context.kind` | `course \| touch \| mock \| arena` (matches T1's CHECK). **FLAG:** T0 §10 listed `run` as a context. Here it is an **action**, and a Run never becomes a `submission` row. |
| `context.id` (gateway-resolved) | **course:** the attempt id, or the **child** `attempt.reimpl_context_id = uuidv5(attempt_id,"reimpl")`, which has its own seq. **touch:** the touch attempt id. **mock:** `mock_session_id` (must be live and contain the item). **arena:** **`uuidv5(account_id, item_id)`**, which is account-scoped. |
| `action` | `run \| submit \| final \| give_up`, plus internal `regrade` |
| **counted** | `kind ∈ {course, touch, mock} ∧ action ≠ run`. Arena submits are persisted but never counted (D10). |
| **evaluation-bearing** | A submission that will produce an evaluation. **Only these take `attempt_seq`.** Runs use their own `run_id`. |
| **cadence** | `iterate`: any number of counted submits, and Run is allowed for `code`. `final`: frozen once per (context, part), by a **lock-in** or by the close. |
| **lock-in** | A `submit` whose parts are all `final`. It freezes those parts, runs their `key` steps inline, and gets its own server `submitted_at`, which timed probes need. In touches, correctness is withheld until the touch concludes. |
| **close** | `final` or `give_up`: a snapshot of every part, with missing parts filled from drafts. **At most one unreleased close per context** (§2.4). |
| **step runs when** | Its `inputs[]` are present. `ai_rubric` runs only on `final`. `code` runs on each submit, and on a close only if the code hash changed since the last evaluated submit; otherwise the close records `reused_from`. |
| **grade lock / conclusion** | The lock is when the grade becomes determined (`anchor_at`). The conclusion is when practice emits the single signal. The two differ only when a re-implement is owed or a provisional window is open (§3). |

### 2.2 Submission request (`POST /api/problems/{id}/submissions` → judge `POST /submissions`)

| Field | Source | Rule |
|---|---|---|
| `Idempotency-Key` | The SPA mints a UUIDv7 **per user action**, including Close, Give up and Dispute | judge enforces `UNIQUE(account_id, idem_key)` plus `body_sha256`. Same key and body replays the result; same key with a different body returns **422** ([IETF draft-07](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header-07)). |
| `account_id` | JWT `sub`, minted by the gateway for audience `judge` | Never read from the body. Every read by id is scoped to `sub`. |
| `item_id` | URL | Must be `live` in judge's embedded content. judge derives `path_slug` itself. |
| `context{kind,id,band?}` | kind from the body; id and band **gateway-resolved** | See §2.1. `band` (touch only) comes from practice's touch-start response. |
| `action` | body | See §2.1. |
| `parts[] {part_id, language?, payload}` | body | `payload` is a **raw JSON value, not an escaped string**. Caps are on the decoded payload (T1 §6.3). The SPA's `serialize()` measures the **encoded request** against 1 MiB. Once stored, `payload_ref = (submission_id, part_id)`. |
| `run_input?` | body, Run only | ≤ 64 KiB, validated against the public `constraints[]` |
| `contract_hash` | **server-sourced**: the attempt pin (course, touch), the `mock_session_item` pin (mock), or live (arena, run) | If it doesn't match the pack: `inconclusive(contract_changed)` |
| **Stamped by judge** | — | `submission_id` (uuidv7) and `submitted_at = now()` (single CNPG, so one clock, inferred). `attempt_seq` comes from `context_counter(kind,id) … RETURNING`, **for evaluation-bearing rows only**. |

### 2.3 Admission

**Gateway checks, first:**
- the session is valid, and the learner is enrolled (counted kinds);
- the attempt or mock is owned and open;
- **touch:** `now ≤ deadline + 5 s`;
- **mock:** the session is live, and the rate limit holds (≤ 1 live session per course, ≤ 3 started per day, inferred).

Request bodies go through one shared `httpx.ReadBody(w, r, 1<<20)` helper using `http.MaxBytesReader`, with `*http.MaxBytesError` → a typed **413**. The helper is applied at **all 11 request-body sites**: `bff.go:164,189,261,310,664,763`, `assessment.go:82,143`, `coach.go:153,238` and `mistakes.go:102`. New routes go into `openapi.yaml`, or the drift test fails.

**judge checks, in one transaction:**

| # | Check | Response |
|---|---|---|
| 1 | Kill switch, or the item is not evaluable (`spec_mismatch`, no pack) | **503** `evaluation_unavailable{fallback: self\|none}` |
| 2 | Idempotent replay | the original result |
| 3 | Unknown or mistyped part / over the cap | **422** `invalid_part` / **413** `part_too_large{part_id,max}` |
| 4 | An unreleased close exists and this is not a close / a `final` part is already locked / a close while an unreleased close exists | **409** `context_closed` / **409** `part_locked` / **200 with the existing close** (replay by context) |
| 5 | `key` or `ai_rubric` in arena or run; `ai_rubric` in mock | The step becomes `skipped(not_evaluated_here)` and the others run. The request is refused if **every** required step would be skipped, **except in mock**, where the submission is stored as the mock answer. |
| 6 | Daily quota (200 Runs, 100 counted, 100 arena; T1) · lane queue full (runner 32, llm 8) · per account: ≤ 2 queued Runs **across contexts**, ≤ 4 queued non-Run jobs, ≥ 2 s between submits in a context | **429** `quota_exceeded{kind,reset_at}` / `queue_full` + `Retry-After` / `too_many_pending` / `too_fast`. **Closes, give-ups and the one re-grade are exempt from all of row 6.** They are already bounded by one unreleased close per context and one dispute per attempt. |
| 7 | Insert the submission, parts and seq. **Key-only:** evaluate inline and write the evaluation plus the outbox row in the same transaction. **Otherwise:** insert a job. **Run:** a job only; it cancels the account's queued Runs in that context. | **200** `{evaluation}` (inline) / **202** `{submission_id\|run_id, state, poll_after_ms: 500, queue_position}` |

### 2.4 Close, give-up and dispute (fixes both critics' blockers)

1. The SPA sends `POST /api/attempts/{id}/close {action: final|give_up, parts[]}` with an Idempotency-Key.
2. The gateway reads practice's attempt: owner, open, pin, purpose and `grading_mode`.
3. The gateway calls judge `POST /submissions` with `closes_context=true`, guarded by a **partial `UNIQUE(context_kind, context_id) WHERE closes_context AND NOT released`**. judge cancels the context's queued Runs, then either enqueues at P0 or evaluates inline. It returns `{submission_id, seq, submitted_at}`.
4. The gateway calls practice `POST /attempts/{id}/close {action, submission_id, close_seq, submitted_at}`. This call is **synchronous, authoritative, and idempotent on `submission_id`**.
   - `give_up`: practice sets `gave_up_at = submitted_at`, moves the stage to solution (the solution unlocks now), sets `revealed_early`, and writes the `solution_revealed_early` outbox row. It locks as soon as the facts for every `seq < close_seq` are present (§3.4).
   - `final`: the state becomes `awaiting_evaluation`.
5. **Backstop.** If step 4 never happens (the gateway crashed), practice's consumer records the close from the fact with `closes_context=true`.
6. **Release.** In the result transaction, judge sets `released=true` on a `final` close whose evaluation is `inconclusive` or `error`. practice then decides:
   - an infra reason and fewer than 3 releases: back to `attempting` ("Couldn't grade: not counted. [Submit again]");
   - `contract_changed`, `budget_exhausted`, or the 3rd release: `self_grade_pending`.

   A **`give_up` close is never released.** It is evidence only, and the lock never waits for it.
7. **Re-implement** runs in the child context (§2.1). It has its own seq and its own drafts (the draft key includes `context_id`, so no `slot` column is needed). It has an optional **Finish** (a `final` close on the child context). Its facts only ever set `Reimplemented`, and never trigger a lock.
8. **Dispute.** `POST /api/evaluations/{id}/dispute` goes to practice, which checks that the attempt is provisional and the dispute is unused. practice then calls judge `POST /evaluations/{id}/regrade`, guarded by `UNIQUE(regrade_of)`.

### 2.5 Job (`judge.job`; claim mechanics in §9)

| Field | Meaning |
|---|---|
| `id` uuidv7; `kind` `evaluate\|run\|regrade\|analyze` | |
| `submission_id?`, `evaluation_id?`, `account_id`, `path_slug`, `item_id`, `seq` | `seq` is informational. **Judge has no per-context FIFO**; practice orders by seq (§3.2). |
| `priority`, `lane` (`runner\|llm`), `step_cursor` | Key steps run in-process in whichever worker holds the job. A composite re-queues onto the next step's lane and never holds a runner slot while waiting on an LLM. |
| `state` `queued\|running\|done\|cancelled`; **`claim_token`** (fencing, incremented on every claim); **`infra_retries`** (a separate budget); `run_after`; `lease_until`; `worker_id` | |
| `payload bytea?`, `output bytea?` | Run only; deleted 24 h after completion |

**The result write is fenced:** `UPDATE job SET state='done' WHERE id=$1 AND state='running' AND claim_token=$t`. If zero rows change, the result is dropped. Then, in the same transaction:
1. check the tombstone;
2. `SELECT submission … FOR UPDATE` (the erase guard);
3. insert the evaluation;
4. release the close if §2.4 step 6 applies;
5. upsert `arena_progress` on a passing arena submit;
6. write the outbox row if the context is counted.

### 2.6 Evaluation

**Status, reason and class (closed enums).**

| status | Meaning | Counts as a failed submit? | Can fail a touch? | `reason` |
|---|---|---|---|---|
| `passed` | Every required step passed | — | — | `ok` |
| `failed` | Learner-attributable | yes, unless `class ∈ free_classes` (default `CE, REJECTED`) | yes | `tests_failed, compile_error, rejected, checks_unmet, empty` |
| `inconclusive` | A **platform-side** non-verdict (INV-14) | never | never | `contract_changed, spec_mismatch, runner_throttled, runner_unavailable, infra_exhausted, ai_unavailable, ai_invalid_output, budget_exhausted` |
| `error` | A judge fault (opscheck alert) | never | never | `internal, pack_invalid` |

- **Step-level only:** `skipped(not_evaluated_here | gate_failed | give_up)` and `reused`.
- **Code `class`:** `AC WA TLE MLE OLE RE CE REJECTED RACE DEADLOCK LEAK`. **FLAG:** this extends T1's hidden set.

**Internal record (`judge.evaluation`):**
- `id, submission_id, job_id?, regrade_of?, reused_from?`
- `status, reason, class?`
- `checks[{key, kind, required, met, part_id, step, concept?, locked_at?}]`
- `score?`
- `hidden?{passed, total}` (correctness cases) and `perf?{passed|failed|not_run}`
- `first_failure?{class, on}`
- `parts[{part_id, status, reason, grader_kind}]`
- `trust` (`honor` if any contributing step is honor, or if `review_flag` is set)
- `provisional_eligible`
- `review_flag?` (§7)
- **`signals[]`**, internal only
- `per_test` (internal)
- `usage` (bucketed)
- `versions` recorded **at claim**: grader, harness, checker and generator `@v`; profile and image digest; rubric `@v`; model; `prompt@v`; pack version and digest; `item_hash`, `contract_hash`, `content_hash`
- `feedback_ref`

**Learner DTO (an allowlist):**

| Grader kind | Shown |
|---|---|
| all | `submission_id, status, reason, action, context_kind, submitted_at, completed_at, attempt_seq, checks[{key,label,required,met}], score, trust, provisional`, and a public `evaluated_with` label |
| `code` Run | compile `{ok, diagnostics ≤ 8 KiB}` (learner files only). For each sample or custom input: `{id, status, class, got ≤ 8 KiB, sample_expected (public), stdout ≤ 8 KiB, stderr ≤ 2 KiB}`. ≤ 64 KiB per Run. |
| `code` Submit | **Samples:** full detail on a failure. **Hidden:** `{passed, total}` over correctness cases, `perf: passed\|failed\|not_run`, and `first_failure.class`, with bucketed usage. No stderr, case ids, ordinals or timing. **Diagnostics appear only at learner-file positions.** A compile error located in a pack or hidden file becomes CE with fixed text: "Hidden tests don't compile against your code; check the required signature." Code check keys are generic (`tests`, `race`, `leak`), never test names. |
| `key` | Per-question correct/partial marks appear **after the context closes**; in a touch, only **after the touch concludes**. `answer_reveal` (the key and rationale) appears only after a **counted conclusion**. **In mock:** an aggregate score only, unless the learner has already concluded that item. |
| `ai_rubric` | Criterion bands with public descriptors and met flags. Must-cover and red flags as **counts**. After conclusion, author `reveal_label`s for missed must-cover items. `feedback_ref`. |
| `composite` | A header, then each step's `Present()`. Deterministic steps are marked "not disputable". |

**Denylist.** The gateway test runs **per endpoint and per concluded state**. It rejects `expected, args, input, case_id, anchors, key, aliases, seed, gen, test_name` everywhere. `sample_expected` is allowed only in Run and sample views. `answer_reveal` is allowed only when `Visibility.Concluded` is true.

**Hidden-case execution.**
1. **Samples** run first. They're public, so they exit early; on a failure, hidden cases are `not_run`.
2. **Every correctness case** (edge, then random) runs in canonical order. `passed` is a count, and `first_failure` is the earliest failing class.
3. **The perf block** runs last and stops at the first TLE. It is reported as **one bit**, so no ordinal leaks. This fixes critic B's point about the perf skip.
4. The job's wall budget is Σ case walls + compile + slack, so truncation can only be runner-side, which gives `inconclusive(infra_exhausted)`.

**FLAG:** Kattis-style early exit on hidden cases would leak ordinals, which ADR-0027 §5 forbids.

### 2.7 Transport

- **Enqueue** takes milliseconds, well inside the 10 s upstream and 1 MiB limits. A canvas (512 KiB plus a 32 KiB export) fits.
- **Poll** `GET /api/submissions/{id}` or `/api/runs/{id}`.
  - `poll_after_ms` backs off 500 ms → 1 s → 2 s → 5 s, with `ETag` → 304. The SPA stops at 5 min: "still grading, we'll show it on Today".
  - For counted contexts, the gateway also returns practice's `{state, grade?, graded_by?, facts_through_seq, concluded, accept_deadline?, dispute_allowed, claim_allowed, ceiling, reimpl_owed}`.
  - **The SPA keeps polling until `facts_through_seq ≥ attempt_seq`**, up to 60 s after the evaluation is terminal. Only then does it render the conclusion.
- **Cache epoch.** The gateway calls `invalidateAgg` (`bff.go:684-688`) when the **practice state it observes changes** (lock, provisional, conclusion) and when the arena done marker changes, not when judge reports a terminal result. This fixes critic B's re-caching race. The epoch is valid only while the gateway has a single replica.
- **SSE is deferred.** The upgrade path is `GET …/events` on the same resource.
- **Knobs:**
  - `RELAY_INTERVAL` = 1 s for judge and practice (5 s today, `relay.go:40`).
  - SPA `REQUEST_TIMEOUT_MS` = **25 s** for submissions and draft PUTs. That is ≤ the server `ReadTimeout` of 30 s (`httpx.go:30`); today's value is 15 s (`api.ts:12`).

### 2.8 Events (payloads carry only enums, numbers and refs)

| Subject | When | Fields (T4 additions in **bold**) |
|---|---|---|
| `xlearn.judge.evaluation_completed` (stream `XLEARN_JUDGE`, 512 MiB, MaxAge 14 d) | Counted **course** (including reimpl) and **touch** evaluations, lock-ins included. **Mock is silent in v2.0.** | `evaluation_id, submission_id, path_slug, item_id, context_kind, context_id, action, closes_context, `**`released`**`, attempt_seq, submitted_at, completed_at, status, reason, class?, checks[{key,required,met,locked_at?}], score?, hidden?, `**`perf?`**`, parts[], signals[], trust, provisional_eligible, regrade_of?, `**`reused_from?`**`, contract_hash`; ≤ ~1.5 KiB |
| `xlearn.practice.problem_solved` **v2** | once per course attempt | v1 fields, the T0/T1 fields, plus **`anchor_at`, `revisable`, `arena_prior (none\|revealed)`, `gave_up`, `reimplemented`, `resolution (immediate\|accepted\|regraded\|override\|claimed\|timeout\|self)`, `mistake_hint{category,strength,signal}`, `concepts_hint[≤3]`** |
| `xlearn.practice.attempt_logged` | **kept, for `purpose=course` only, never for touches** | v1 fields plus **`path_slug`, `anchor_at`** |
| `xlearn.practice.touch_concluded` | once per non-voided touch | T0 fields plus **`trust`, `evaluation_ids[]`, `anchor_at`, `score?`, `resolution (…\|abandoned)`, `mock_mode`, `mistake_hint`, `concepts_hint[]`** |
| `xlearn.judge.evaluation_analyzed` (M4) | one per analyze job | `attempt_id, evaluation_ids[], path_slug, problem_id, category\|"", confidence_bucket, concept_keys[≤3], source: ai, feedback_ref, analyzer@v` |

**Durable consumers:**

| Durable | Filter | Notes |
|---|---|---|
| `practice` (its first inbox) | `xlearn.judge.evaluation_completed` | **Always acks** after the inbox transaction. Gaps are pulled, not nakked (§3.2). Ships one release before judge emits. |
| `judge-analyzer-solved`, `judge-analyzer-touch` (M4) | `…problem_solved`, `…touch_concluded` | **Two durables**, because v1 `Subscribe` takes one filter subject and `{a,b}` is not NATS syntax. Both use `WithDeliverNew`. |
| `review-analysis` (M4) | `…evaluation_analyzed` | Applied iff `attempt_id = mistake_entry.last_refreshed_by_attempt_id` |

Every handler is DB-only and takes about 100 ms or less, inside the 25 s handler timeout and 30 s AckWait (`consumer.go:17,23`).

### 2.9 Contract invariants

| # | Invariant |
|---|---|
| INV-1 | **Liveness:** every accepted evaluation-bearing submission ends in exactly one terminal evaluation. |
| INV-2 | judge never computes grades and never writes to review or assessment. |
| INV-3 | Context id, `contract_hash`, `submitted_at` and `attempt_seq` come from the server. |
| INV-4 | `key` and `ai_rubric` never run in arena or Run. `ai_rubric` never runs in mock. |
| INV-5 | Status derives from **required** steps only. AI can't lift a failed deterministic gate. |
| INV-6 | Graders are time-blind. Every timer is practice's, applied to `submitted_at` or `locked_at`. |
| INV-7 | Raw `StepResult` never leaves judge. The DTO is an allowlist, backed by a per-endpoint denylist test. |
| INV-8 | Events carry no prose and no code. |
| INV-9 | practice concludes **once**: `UPDATE … WHERE concluded_at IS NULL RETURNING` plus `UNIQUE(outcome.attempt_id)`, and only on a gap-free fact set. **Self outcomes are accepted only when `grading_mode=self` or `state=self_grade_pending`**; otherwise **409** `evaluated_item`. |
| INV-10 | `inconclusive` and `error` never count as failed and never fail a touch. |
| INV-11 | Result writes are fenced by `claim_token` and serialized against erase. |
| INV-12 | **Only evaluation-bearing submissions consume `attempt_seq`.** Runs never do. |
| INV-13 | **At most one unreleased close per context.** A `give_up` close is never released. |
| INV-14 | **`inconclusive` needs platform-side evidence**: a typed runner infra error, throttle telemetry, a contract change, or an AI or budget failure. Nothing the learner's process does yields it. |

**FLAG (T0 §5):** "FIFO ⇒ the trigger is the last evaluation seen" fails under JetStream nak-with-delay ([NATS](https://docs.nats.io/learn/jetstream/delivery-and-acknowledgment), [Synadia](https://www.synadia.com/blog/process-jetstream-messages-strict-order)). The v1 library has no `MaxAckPending` knob (`consumer.go:118-126`). The replacement is **`attempt_seq` plus a gap-free check in practice plus a pull reconciler**, and judge drops per-context ordering entirely.

---

## 3. Attempt and conclusion lifecycle

### 3.1 States (practice `attempt`, `purpose ∈ {course, touch}`)

```
          resume any time before lock; server timers keep running; no sweeper
start ─► attempting ──facts (event or pull)── strategy LOCKS at seq T, facts < T gap-free ──┬─ nothing owed ─────────────► concluded ─► ONE signal
          │  │                                                                               ├─ reimpl owed ─► reimplementing ┤   (Conclude tx + outbox)
          │  ├─ give_up (sync, §2.4) ── lock: earliest pass < close_seq, else miss ──────────┤    [reimpl pass | Finish | 24 h idle]
          │  └─ final close (sync) ─► awaiting_evaluation ── closing fact ── lock ───────────┤
          │                              ├─ released (infra, < 3) ─► attempting              └─ could gain ∧ non-authoritative grader ─► provisional
          │                              └─ contract_changed | budget | 3rd release ─► self_grade_pending    [accept | re-grade→accept/override | claim | 24 h]
          │                                                  [pick ≤ ceiling | Retry grading ×1] ─► concluded
touch:  attempting ─[all criteria met | End review | deadline+5 s, drained to judge watermark]─► concluded (pass | fail | abandoned)
                   └─ never shown, or only platform-inconclusive evidence ─► voided (stays due; nothing emitted)
```

**FLAG (T1 §4 schema, additive):**
- `attempt.state` CHECK gains `reimplementing`, `self_grade_pending` and `voided`.
- New columns: `grade_locked_at, anchor_at, locked_grade, ceiling, candidate_grade, settle_deadline, dispute_used, claim_used, retry_used, close_submission_id, close_seq, released_closes, gave_up_at, reimpl_context_id, grading_mode, facts_through_seq, facts_gap_since, probes_shown_at, ended_at`.
- `user_problem_state.arena_revealed_at`.
- A new fact table: `practice.attempt_evaluation(evaluation_id PK, attempt_id, context_id, seq, action, closes, released, status, reason, class, checks, score, signals, trust, provisional_eligible, submitted_at, regrade_of, source: event|pull|invalid)`.
- `UNIQUE(outcome.attempt_id)`.

### 3.2 practice consumer and ticker

**Consumer (one transaction per message; always ack unless the DB errors):**
1. Claim the inbox row and check the tombstone.
2. Resolve `context_id` to the attempt (main or reimpl context) `FOR UPDATE`. Validate the account, item, path and purpose. On a mismatch: ack, count it as `forged_or_stale`, and store no fact.
3. Insert the fact `ON CONFLICT DO NOTHING`. If the `contract_hash` differs from the pin, store it as `inconclusive(contract_changed)`.
4. If the fact has `closes_context` and no close is recorded, record the close (the backstop).
5. If the attempt is already concluded or voided, commit (a late fact).
6. Run `strategy.Grade(facts, params)`. If it has locked at seq T and every `seq < T` is present, apply the lock. Otherwise set `facts_gap_since` and commit.
7. Apply the lock: `Conclude()` now, or move to `reimplementing`/`provisional` with `settle_deadline = now() + 24h`.
8. Timing: the stage *as of* `submitted_at` comes from `stage_event`. `within_timer` compares `submitted_at` with the deadline, with 2 s grace (inferred).

**`Conclude()`** replaces `LogOutcome` (`store.go:352-433`). v1 reads the state without `FOR UPDATE`, has no `ended_at`/status guard on its updates, and no unique outcome, so concurrent calls can double-emit. The new shape:
- `UPDATE attempt SET concluded_at=now(), state='concluded' WHERE id=$1 AND concluded_at IS NULL RETURNING`; zero rows means a no-op;
- the outcome row (unique);
- `solved`;
- the outbox: `problem_solved` v2 plus `attempt_logged` (course), or `touch_concluded`.

**Ticker (practice, every 30 s, `FOR UPDATE SKIP LOCKED`).** This replaces relying on 100 naks, which cap out at about 8 h (`consumer.go:30`).

| Job | Rule |
|---|---|
| **Gap reconciler** | When `facts_gap_since` is more than 30 s old, call judge `GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T` and insert the results (`source=pull`, same validation). If a pulled fact fails validation, insert an `invalid` filler that is non-passing and never failed, and raise an alert. |
| **Touch deadline / End review** | Call judge `GET /internal/contexts/touch/{id}/watermark → {max_seq, open_jobs}`. Once the facts are complete through `max_seq` and `open_jobs = 0`, conclude or void (§3.6). |
| **Settle** | When `settle_deadline` passes: accept the provisional grade, or finish the re-implement (`reimplemented=false`). |
| **Alerts** | Gaps older than 10 min; `awaiting_evaluation` for more than 10 min; `self_grade_pending` for more than 7 d. |

### 3.3 Transitions

| From → to | Trigger / guard | Effect |
|---|---|---|
| start → attempting | Enrolled; no open attempt (otherwise resume, `store.go:229-238`); **touch:** one live touch per (account, problem), at the lowest due level | Pins `policy_version`, `stage_params`, `contract_hash` and `grading_mode` from judge `/internal/evaluable`. Starts the server timer, then shows the statement and the workspace (D10). |
| attempting → lock | The strategy's trigger (§6.2) on a gap-free prefix | `grade_locked_at`; `anchor_at` = the `submitted_at` of the deciding fact, or `gave_up_at` |
| lock → concluded | Nothing owed and not provisional | `resolution=immediate` |
| lock → reimplementing | Locked by `give_up` (which includes reveal) and `reimplement: required` | Blank editor in the child context. Concludes on a reimpl pass, on **Finish** (enabled after ≥ 1 reimpl submit), or after 24 h idle (`timeout`). |
| lock → provisional | `provisional_eligible` and the learner could gain (§3.5) | Stores the candidate and ceiling; `accept_deadline = now()+24h`. No signal yet. |
| provisional → concluded | Accept · 24 h · re-grade then accept or override · honor claim | Signal. `anchor_at` is unchanged. |
| awaiting → attempting | A `final` close released for an infra reason, fewer than 3 times | `released_closes++` |
| pre-lock → self_grade_pending | Kill switch; `contract_changed` with counted evals; `budget_exhausted`; 3rd release | Picker capped at the server ceiling (`self@1{cap:stage}`), plus **Retry grading, once** (a new close). Never auto-concluded. |
| `contract_changed`, no counted evals | — | Silently re-pin; the timer continues (inferred). |

### 3.4 Give-up (which, for evaluated items, includes "reveal full solution")

- Reveal without a pass is `give_up`: both lock at miss, and give-up unlocks the solution (`t0…:111,167`).
- **It is a synchronous practice write (§2.4 step 4).** The solution unlocks and `revealed_early` is set at that moment, whether or not the runner is up.
- **Lock rule:** once the facts are complete for `seq < close_seq`:

  | Facts below `close_seq` | Grade |
  |---|---|
  | a passing counted submit exists | `verdict_timer@1` at the **earliest** pass (the in-flight pass wins) |
  | none | miss at `gave_up_at` |

  Both orderings are golden rows (§6.2).
- The closing snapshot is **evidence only**. judge runs the deterministic steps, which produce signals such as `blank_draft` and `sample_failed`, and skips `ai_rubric` (`skipped(give_up)`).
- The self path (items with no evaluator) keeps v1's reveal semantics.

### 3.5 Parked Q1 resolved: who finalizes a non-authoritative grade

1. **Provisional only when the learner could gain.** `provisional_eligible` is true when (a) an `ai_rubric` step contributed, or (b) a criterion decided by an **honor-grade public key** (`key_source: public:*`, e.g. the DSA pattern and complexity probes) was missed.
   - **Course attempt:** provisional iff `candidate < ceiling`. The ceiling comes from the timer, hint, reveal and `arena_prior` facts plus the required deterministic gates.
   - **Touch:** provisional iff the touch would fail **and** flipping the non-authoritative criteria would pass it.
   - Anything else is final immediately.
2. **AI: one blind dispute.** The learner ticks the misjudged criteria (reason codes) and may add ≤ 500 chars of text. That text is **stored for calibration and never reaches the scorer**, because it is an injection channel ([Raina et al.](https://arxiv.org/html/2504.18333v1), [JudgeDeceiver](https://arxiv.org/pdf/2403.17710)). It lives in `judge.evaluation_dispute`, which the erase path covers.
   - The re-grade is a fresh k=5 with reshuffled exemplars, as a new evaluation with `regrade_of`. It may move the grade up or down, and it opens a new 24 h window.
   - Precedent: [Gradescope](https://guides.gradescope.com/hc/en-us/articles/21854736042253-Submitting-a-Regrade-Request).
3. **Override** is available only **after** the re-grade. It is bounded to `[miss, ceiling]`, recorded as `graded_by=override, trust=honor`, excluded from "judge-checked %", and labelled everywhere.
4. **Honor-key claim.** For a missed public-key probe, the accepted answers are shown with one **"I meant this"** claim per attempt. The criterion is then met, with `resolution=claimed` and `trust=honor`. The probe was honor-grade already; this restores v1's self-affirmation (`Revision.tsx:340-354`) for synonyms or notation missing from the alias table. Checked (pack) keys stay strict.
5. **Timeout:** 24 h after the latest candidate, the ticker accepts it.
6. **Anchoring at `anchor_at`** (the lock time). Day 1 never slips during a window, and replay stays deterministic (ADR-0018). **FLAG:** this amends PRD R-SR1's "from its conclusion" to "from the moment the grade was determined". That moment is at most about 48 h before the signal.

Why: LLM rubric scores vary between runs (α of 0.27–0.56, [Rating Roulette](https://aclanthology.org/2025.findings-emnlp.1361/)) and carry position and verbosity bias ([Zheng et al.](https://arxiv.org/pdf/2306.05685)). Alias tables also miss synonyms. So one non-authoritative call shouldn't silently reset a ladder, yet PRD V3 wants "evaluated", not self-picked.

### 3.6 Parked Q2 resolved: abandoned attempts

| Situation | Behaviour |
|---|---|
| Course attempt before lock | **Resumable forever**, and the server timers keep running ("Resuming: best grade now Rough"). **No sweeper** (T0: a sweeper fabricates activity and races evaluations, `t0…:351`). After 24 h idle, Today shows "Unfinished: resume or give up". It blocks new work in that course only through the frontier (`dashboard.go:301-326`). |
| After lock (reimpl, provisional) | A 24 h settle window, then auto-finish. `reimplemented=false` / `resolution=timeout` record what didn't happen. Projections date the activity at `anchor_at`, not at the conclusion (§11.4 #13), so **no activity is fabricated**. |
| Touch: all criteria met | Concludes at once (pass). |
| Touch: End review, or deadline + 5 s | Drain to judge's watermark (§3.2), then apply the criteria. |
| Touch: learner evidence exists | Conclude, pass or fail, on criteria as they stand. |
| Touch: probes shown (`probes_shown_at` set), no evidence | **Fail** (`resolution=abandoned`). The learner saw the test; resume within the timer covers accidents. This closes critic B's read-abandon-research loop. |
| Touch: never shown, **or** the only evidence is `inconclusive` | **Voided.** No signal; the touch stays due and R-SR5 keeps blocking new work in that course. A fresh touch attempt starts next time. |
| Drafts | Kept for 90 days idle (§9.6). |

**FLAG (T0 §5 triggers):** the grade locks at give-up, and conclusion follows the re-implement settle rule. Whether a re-implement is owed after a **passing** attempt is **O3**.

### 3.7 Touches, mocks, arena

- **Touch** (`purpose=touch`, M2a):
  - `POST /touches/start` creates the attempt with a server start, the band's timer, and `mock_mode` for L4–5 bands.
  - The gateway sets `probes_shown_at` when it serves the touch view.
  - Probes are **lock-ins** (inline key evaluation, correctness withheld). `recall_secs` = the probe's `locked_at − started_at`.
  - `touch_passed` = every required criterion met (T0).
  - review keeps `Score`'s pending guard. It gains `UNIQUE(touch_result.attempt_id)`, and **`Score` and `ReanchorTouch` take `anchor_at`**; the fail path uses `time.Now()` today (`store.go:515`).
- **Mock** (evidence only):
  - Non-AI steps run. The composite is `inconclusive(not_evaluated_here)` when a required AI step is skipped, and the submission is still stored as the mock answer.
  - `key` shows only an aggregate score unless the item is already concluded (§2.6).
  - Rate limits are in §2.3.
  - `ScoreMock` (`assessment/store/store.go:265-351`) stays the only mock signal. There is no `problem_solved`, no mistakes and no schedule.
- **Arena** (D10):
  > **Superseded by D17 (§13):** arena is unrestricted — no locks during live attempts or touches, and an early reveal is recorded but doesn't cap the course grade. The lock and cap bullets below are the pre-decision design.
  - `context_id = uuidv5(account, item)`. Submits are persisted; Runs are not.
  - A passing submit upserts `arena_progress(first_passed_at, submits, source=auto)` in the result transaction.
  - B and C items are **study mode**: editable parts, a revealable reference, and **"Mark studied"** (`source=manual`). LLD's `code` facade step is allowed in the arena; its AI step is not.
  - **Withholding** uses T0 §7's predicate, "open counted attempt **or due or live** touch". **Arena Submit is disabled** (sample Run stays, since it equals course Run), and **Solution, History and Copy-to-editor are withheld**. The route-enumeration test covers them.
  - An arena **Solution reveal** before the item's course attempt has concluded requires a spoiler confirm. It records `arena_revealed_at` through `POST /api/arena/{item}/reveal`, which caps the later course attempt at `assisted` (**O4**).
  - No events. The timer is manual, client-side and off by default. The v1 no-op branch (`bff.go:636-645`) is replaced.

---

## 4. Archetypes

### 4.0 Overview

| | **A. Code IDE** | **B. Structured response** | **C. Quiz / key** |
|---|---|---|---|
| Courses | dsa, go-concurrency, sql, LLD facade gate | system-design, lld-ood, behavioral (not a pilot) | drills in every course; recall probes for every touch |
| Parts | `code` iterate | `text`, `blank`, `choice`, `canvas` final; LLD `code` iterate | `choice`, `blank`, short `text`, all final |
| Graders (lane) | `code` (runner) [+ `key` inline] | `key` (inline) → [`code` facade gate (runner)] → **one** `ai_rubric` (llm) | `key` (**inline, synchronous**) |
| Actions | run, submit, give_up | lock-in, final, give_up | lock-in (probes), final |
| Run | samples plus custom input | LLD code only | none |
| Arena | Run plus Submit | study mode (+ LLD facade Submit) | study mode |
| Mock | hidden count, first failure class | key aggregate; AI not run | aggregate only (unless concluded) |
| Trust | checked; **honor** for in-process profiles | checked; honor if a facade gate, override or `review_flag` contributed | checked (pack key) / honor (`public:*`) |
| Provisional | never (except honor-probe criteria in touches) | yes (§3.5) | honor keys only (claim) |
| Strategy | `verdict_timer@1` | `weighted_gate@1` / `rubric_pct@1` | `weighted_gate@1` |

### 4.1 Archetype A: Code IDE

- **Widget:** CodeMirror 6 (MIT). The value is `{language, files[]}`; go-concurrency's `visible_test.go` is read-only. Tab doesn't trap focus by default ([CodeMirror](https://codemirror.net/examples/tab/)). Monaco is rejected: 2–5 MB and no mobile support ([Sourcegraph](https://sourcegraph.com/blog/migrating-monaco-codemirror), [Replit](https://blog.replit.com/codemirror)).
- **Run:** public `samples[]` from the embedded `item.json`. No pack is read, so Run survives `spec_mismatch`. Custom input shows only the learner's own output; there is **no reference oracle**.
- **Submit:** samples, then correctness cases, then the perf block (§2.6).
- **Harnesses:** public and versioned: `func-json@1`, `class-ops@1`, `gotest@1`, `sql-result@1`, and C++ later. The result channel is separate from stdout, so there is no presentation error.

**Verdict mapping:**

| External | xLearn | Counts as failed? |
|---|---|---|
| AC / OK | passed / AC | — |
| WA, PE | failed / WA | yes |
| TLE (CPU or wall **of the learner's process**) | failed / TLE | yes |
| MLE (including an OOM kill of the learner's process) / OLE | failed / MLE / OLE | yes |
| RTE, NZEC, signal | failed / RE (class only for hidden cases) | yes |
| CE | failed / CE | **no** (the default `compile_penalty=false` in [DOMjudge](https://www.domjudge.org/docs/manual/main/configuration-reference.html)) |
| disallowed construct | failed / REJECTED | **no** |
| IE, JE | error | never |
| typed runner infra error, or `throttled` persisting after one re-run | inconclusive(`runner_unavailable` / `runner_throttled`) | never |

Within a case, precedence is error > TLE > MLE > OLE > RE > WA ([DMOJ](https://dmoj.readthedocs.io/en/latest/judge/status_codes/)).

**Performance cases (decided; this amends T1 §3.3):**
- Literal cases stay capped at **256 KiB**.
- Seeded specs: `{gen: "int_array@1", params, seed, expected: literal | {sha256, bytes}}`. The generators form a **closed, public Go registry** (`int_array, string, permutation, tree, graph, op_sequence`; ~400 LOC, inferred), golden-hash tested per `gen@v`.
- judge generates each input to a per-job file on `emptyDir` and **streams** it to the runner.
- Digest comparison is used only for canonicalizable checkers. CI derives the digests from the reference solution and cross-checks the brute oracle on small seeds.
- A per-item `large_case_exception` of ≤ 2 MiB needs an owner stamp.
- The pack holds specs and seeds, **never generator code**.
- Rejected: DMOJ runtime generators, which put per-item code on the hot path ([DMOJ](https://dmoj.readthedocs.io/en/latest/problem_format/generator/)).

**Variants:**

| Variant | Profile / harness | Checks → classes | Trust | Specifics |
|---|---|---|---|---|
| **SQL** | `sql-pg`, `sql-result@1` | `resultset{ordered, columns, float_eps, null_equal}` → WA · SQL error → RE · `statement_timeout` → TLE · a write in a query item → REJECTED (read-only transaction plus SELECT-only role) · **non-required** `explain:*` checks (required only if the item declares them) | checked | Run: public sample dataset, grid ≤ 100 rows. Hidden instances are built only for submits. |
| **go-concurrency** | `go-race`, `gotest@1` (in-process), `-race -count=N`, goleak | Every declared test must pass (`test2json`) → RACE / DEADLOCK / LEAK / WA / TLE / RE / CE | **honor** | **Rule:** a race, or an assertion failure, in **any** run fails ([Go](https://go.dev/doc/articles/race_detector)). Per-test deadline ≥ 10× the published baseline. **On a timeout, the runner captures a goroutine dump.** Goroutines blocked in learner packages → **failed/DEADLOCK**, even if other runs passed; otherwise TLE. The result is `inconclusive(runner_throttled)` only when T3 reports `throttled` and one re-run in a quiet slot is still throttled (INV-14). Spot-the-race and predict drills are archetype C. |
| **LLD facade** | `gotest@1` / `func-json@1` as a **required gate** inside B | tests only; generic check keys | honor | Tests never fix the design (T0). |

**T3 must provide** (§11.1): compile kept separate from execution; per-case terminal state measured outside the untrusted process; positioned diagnostics; typed infra errors; `throttled`; a goroutine dump; streamed per-case input ≤ 8 MiB.

### 4.2 Archetype B: Structured response (canvas + questionnaire + AI)

```jsonc
{"id":"sd-001","course":"system-design","role":"core","concepts":["system-design:concept:sharding","system-design:concept:caching"],
 "parts":[
  {"id":"reqs","type":"text","cadence":"final","config":{"fields":[{"id":"functional","max":2000},{"id":"nonfunctional","max":2000}]}},
  {"id":"est","type":"blank","cadence":"final","config":{"fields":[{"id":"write_qps","kind":"numeric","units":["req/s"]},{"id":"storage_5y","kind":"numeric","units":["GB","TB","PB"]}]}},
  {"id":"hld","type":"canvas","cadence":"final","config":{"palette":"system-design/palette@1","max_nodes":40}},
  {"id":"deep","type":"text","cadence":"final","config":{"fields":[{"id":"tradeoffs","max":4000}]}}],
 "grader":[
  {"step":"estimates","kind":"key","inputs":["est"],"required":false,"weight":0.2},
  {"step":"design","kind":"ai_rubric","inputs":["reqs","hld","deep","est"],"rubric":"system-design/hld@1","required":true,"weight":0.8}],
 "analyzer":{"inputs":["reqs","hld","deep"]}}
// public rubrics/hld@1: criteria[{id,label,concept,weight,bands{1..5}}], dims[{id,category,concepts[]}], pass_rule{min_pct:60, required_criteria:["scalability"]}
// private pack: keys{est.write_qps:{value:1160,tol:{factor:3}}}, rubric{must_cover[], anchors{1,3,5}, red_flags[], exemplars[synthetic]}
```

**Widgets:**
- `text`: named fields with a counter (16 KiB total).
- `blank`: a number plus a unit `<select>`.
- `choice`: options shuffled per attempt.
- `canvas`: recommended **React Flow** (O5).

**Canvas storage.** The stored payload is library-neutral: `canvas-graph@1 {palette, nodes[{id,kind,label,x,y,group?,fields?}], edges[{id,from,to,kind,label?,mult?}], groups[], notes[]}`, ≤ 512 KiB and ≤ 2,000 elements.
- The palette is public and versioned. UML multiplicities are dropdowns: they were the weakest LLM criterion in the UML grading study ([SciTePress](https://www.scitepress.org/Papers/2025/134819/134819.pdf)), and dropdowns let `key` check them.
- judge derives **`canvas-export@1`** in Go: layout stripped, ids canonical, labels ≤ 80 chars, ≤ 32 KiB. A **"What the grader sees"** drawer runs the same algorithm client-side.
- An **Outline view** (a table editor) covers keyboard, screen-reader and mobile use ([React Flow a11y](https://reactflow.dev/learn/advanced-use/accessibility)).
- Both libraries are MIT and React 19-compatible (npm, 2026-09-24): `@xyflow/react` 12.11.6 and `@excalidraw/excalidraw` 0.18.1. tldraw needs a licence key ([tldraw](https://tldraw.dev/sdk-features/license-key)).

**Grading and actions:**
- **The LLM grades the text export, never a PNG.** Models score 14.9–28.7 points lower from images than from text ([IsoBench](https://arxiv.org/abs/2404.01266)).
- **Run:** none, except LLD code. The primary action is **Finish & submit for grading**, after a pre-flight check.
- **Verdicts:** `key` steps give `checks_unmet` or a partial score. `ai_rubric` passes when every required criterion is at band ≥ 3 **and** the percentage is ≥ `pass_rule.min_pct`.
- **Give-up:** the key steps and the facade gate run, and AI is skipped. The feedback call still runs as B's analyzer.
- **Mock:** key steps only, with an aggregate score (§3.7).
- **Behavioral:** `rubric_pct@1` on STAR text. It needs consent before any platform-key call (T5/T7), is hidden by default, and is not a pilot.

**Cost (inferred; T5 prices it):** about 7–10k input tokens × k=3, plus a feedback call, ≈ $0.05–0.15 per final.

### 4.3 Archetype C: Quiz / key

| Mode (inside `choice`/`blank`) | Value | Key | Scoring |
|---|---|---|---|
| `choice.single` | `{selected:[id]}` | `answer`, `distractors{id: misconception}`, `rationale_md` | exact |
| `choice.multi` | `{selected:[ids]}` | `answer[]` | `all_or_nothing \| right_minus_wrong (≥0) \| per_option` ([QTI 3](https://www.imsglobal.org/spec/qti/v3p0/impl)) |
| `choice.order` | `{order:[ids]}` | `order[]` | `exact \| adjacent_pairs \| kendall` |
| `blank.text` | `{fields:{id:{raw}}}` | `accept[]` plus closed normalizers `trim, casefold, collapse_ws, strip_punct, nfkc`; `typo:0\|1` | exact after normalization |
| `blank.numeric` | `{raw, unit}` | `value, unit, tol{abs\|rel\|factor}` | tolerance after a closed unit-family conversion ([Moodle](https://docs.moodle.org/dev/Question_Engine_2:Numerical_tolerances)) |
| **`blank.big_o`** (new) | `{raw}` | `accept[]` | A closed normalizer: `O(n^2)` ≡ `O(n²)` ≡ `O(N*N)` ≡ `O(n·n)`; `log n` ≡ `logn` ≡ `lg n`; variable case and whitespace folded; the `time/space` pair split |
| `blank.predict_output` | `{raw}` | `accept[]`; `trim_lines, crlf` | exact, no typo tolerance |

- **Pass:** `pass.min_pct` (100 for a single question). **Score:** correct/total.
- **Reveal:** the key and rationale appear only after a **counted conclusion**, even for a miss ([Butler & Roediger 2008](https://link.springer.com/article/10.3758/MC.36.3.604)). The key is never shown in the arena before the item is concluded, in Run, or in mock for an unconcluded item.
- **Diagnosis:** a chosen distractor's tag becomes `misconception:<category>`, a deterministic category ([Briggs et al.](https://www.tandfonline.com/doi/abs/10.1207/s15326977ea1101_2)).
- **Widgets:** native radio and checkbox groups in `fieldset`/`legend`. Ordering uses move buttons and Alt+Arrow. Numbers use `inputmode=decimal`.
- **Guessing:** a 4-option guess is right 25% of the time, so recall bands prefer `blank` and text probes, and MCQ is drawn from a fresh permutation or pool sample (inferred).
- **T3/T5:** nothing needed.

### 4.4 Which steps run in which context

| Context | Steps |
|---|---|
| course | the item's `grader[]` |
| touch | the band's `revision.bands[L].criteria[]` → probe `key` steps (`probe:<id>`), plus the `grader[]` steps whose parts the band lists |
| mock | `grader[]` minus `ai_rubric` |
| arena | `code` steps only |
| run | `code`, samples only |

---

## 5. Plug-in interface

### 5.1 Registries

Closed: an explicit list in one file, with no `init()` self-registration, plugins or WASM (T0).

| Registry | Where | Keyed by |
|---|---|---|
| Parts | `internal/platform/item/parts` (curriculum, practice, judge) | part type: config and payload decoders, caps, cadences, draft policy, `contract_hash` fields |
| Graders | `internal/judge/grader` | `code \| key \| ai_rubric`; `composite` root; `analyzer` |
| Normalizers | `internal/judge/grader/key/norm` | `trim, casefold, collapse_ws, strip_punct, nfkc, big_o`, unit families |
| Generators | `internal/judge/gen` | `name@v` |
| Strategies | `internal/practice/grading` | `id@v` |
| Widgets / result views | `web/src/parts/registry.ts`, `web/src/results/registry.ts` | part type / grader kind |

`cmd/partsgen` emits `registry.gen.json`, and a vitest check fails CI if the TS registries drift from it.

### 5.2 Grader contract (Go)

```go
type Caps struct {
    PartTypes []parts.Type; Contexts ContextSet // code: all; key: {course,touch,mock}; ai_rubric: {course,touch}
    SupportsRun bool; Cadences []parts.Cadence; Deterministic bool
    GradedBy GradedBy                           // auto | ai
    Lane Lane                                   // inline | runner | llm
    NeedsPack bool
}
type Grader interface {
    Kind() Kind; Version() string; Caps() Caps
    ValidateSpec(s StepSpec, it ItemView) []Issue
    ValidatePack(s StepSpec, it ItemView, p PackItem) []Issue
    Estimate(in Input) Cost                                                    // admission + T5 Reserve()
    Evaluate(ctx context.Context, in Input, pack PackView) (StepResult, error) // error = infrastructure only
    Present(sr StepResult, v Visibility) json.RawMessage                      // v={Context, Closed, Concluded, SolutionUnlocked, PriorConcluded}
    TrustOf(s StepSpec) Trust
}
type Input struct { Step StepSpec; Item ItemView; Parts map[string]parts.Payload; Action Action
    Context ContextKind; Prior []StepResult; Deps Deps /* Runner(T3), LLM(T5), Gen: fakeable */ }
type StepResult struct { Step, Version string; Kind Kind; Status StepStatus; Reason Reason; Class Class
    Checks []Check; Score *Score; Hidden *Count; Perf PerfBlock; FirstFailure *Failure
    Signals []Signal; Usage Usage; ReviewFlag bool; Internal json.RawMessage /* never presented */ }
type Analyzer interface { Analyze(ctx context.Context, in AnalyzeInput) (Analysis, error) } // M4
```

**The wrapper enforces the settled invariants:**
- A context outside `Caps.Contexts` → `skipped(not_evaluated_here)`.
- In `run` and `arena`, `PackView` is a samples-only type **with no `Hidden()` method**.
- `Present()` is followed by the per-endpoint denylist.
- Code diagnostics are filtered to learner-file positions.
- Versions are recorded at claim.
- The tombstone is checked before the write.
- `Estimate()` is checked against quotas and T5's `Reserve()`.
- `inconclusive` is accepted only with a platform-side reason (INV-14).

### 5.3 `composite@1` (the root of every item; never nested)

- **Steps:** `{kind, inputs[], required, weight ≥ 0, after: always|gates_passed}` run in order. `ai_rubric` defaults to `after: gates_passed`.
- **Status:**
  1. any required step `failed` → failed;
  2. else any required `error` → error;
  3. else any required `inconclusive`/`skipped(not_evaluated_here)` → inconclusive (first reason);
  4. else passed.

  `give_up` skips are excluded. Non-required steps never affect status, so INV-5 is structural. **In mock** there is no refusal: the composite stays `inconclusive(not_evaluated_here)`, and the non-AI step results are the evidence.
- **Score:** `Σ w·v/max ÷ Σ w` over non-context-skipped steps.
- **Provenance:** `graded_by=ai` iff an `ai_rubric` step ran with `required` or `weight > 0`. `trust=honor` iff any contributing step is honor, or a `review_flag` is set.

### 5.4 Grading strategy (practice)

```go
type Strategy interface { ID() string; ValidateParams(json.RawMessage) error; Grade(f Facts, p Params) Result } // pure
type Facts struct {
    Purpose string; StartedAt, Deadline time.Time; HintAt, SolutionAt, GaveUpAt, ArenaRevealedAt *time.Time
    CloseSeq *int; Evals []EvalFact /* seq order */; Probes []ProbeFact; Claims []Claim; Reimplemented bool
}
type Result struct {
    LockSeq *int; LockedAt *time.Time; Grade string; Passed bool; Score *Score; Criteria []Criterion
    GradedBy, Trust string; ProvisionalEligible bool; Ceiling string; Signals []string
}
```

I1–I8 (§6.1) are property-tested over every registered strategy.

### 5.5 Widget and result-view contracts (TS)

```ts
export interface PartWidget<C, V> {
  type: PartType; version: string;                      // == Go parts.Def.Version (generated test)
  load(): Promise<{ default: React.ComponentType<WidgetProps<C, V>> }>;  // lazy CM6 / React Flow
  empty(config: C, ctx: { language?: string; starter?: File[] }): V;
  validate(config: C, v: V): Issue[];                   // UX mirror; server authoritative
  serialize(config: C, v: V): { value: unknown; encodedBytes: number }; // raw JSON value; request size checked BEFORE send
  deserialize(config: C, value: unknown): V;
  draft: { debounceMs: number; timeoutMs: number };     // code/text 15s/25s; choice/blank 5s/25s; canvas 60s/25s
}
export interface WidgetProps<C, V> {
  partId: string; config: C; value: V; onChange(v: V): void;
  mode: 'edit' | 'readOnly' | 'review' | 'locked'; review?: PartReviewDTO;
  labelId: string; describedById?: string; disabled?: boolean;
  actions?: { run?(): void; submit?(): void; lockIn?(): void };
}
export interface ResultView { kind: 'code'|'key'|'ai_rubric'|'composite'|'analyzer';
  load(): Promise<{ default: React.ComponentType<ResultProps> }>; }
export interface ResultProps { evaluation: EvaluationDTO | null; practice: PracticeViewDTO | null;
  phase: 'idle'|'queued'|'running'|'settling'|'done';   // settling = waiting for practice facts_through_seq
  context: 'run'|'arena'|'course'|'touch'|'mock'; parts: PartSpecView[]; onFocusPart(id: string): void; }
```

### 5.6 CI lints (public CI, and again at judge start)

1. Part types are registered, and their configs decode strictly.
2. Step kinds are in the closed set.
3. `inputs ⊆` part ids, and their types match `Caps`.
4. `key` and `ai_rubric` inputs are `final`; `code` inputs are `iterate`.
5. At most one `ai_rubric` per item, with a `pass_rule`.
6. At least one step is required.
7. Every part feeds a step or is `ungraded: true`.
8. `public:*` sources resolve.
9. `contract_hash` covers ids, kinds, inputs, harness, checker, generator, rubric and palette `@v`.
10. Golden fixtures for each deterministic grader.
11. The per-endpoint denylist over every `Present()`.
12. `concepts[]` resolve.
13. `mistakes.prefill` signals and categories are valid.
14. **Every `revision.bands[*].criteria[]` key maps to a step check.**
15. **A compile-leak fixture per in-process profile:** a hidden test calls a missing learner symbol, and the DTO must show the fixed CE text only.

### 5.7 Cost of adding capability (inferred)

| Change | Code | Effort | ADR? |
|---|---|---|---|
| A course reusing kinds | content plus manifest | 0 code | no |
| A new `choice`/`blank` mode or normalizer | Go scorer, TS mode, `Present()`, fixtures | 1–3 d | no |
| A new perf generator | Go plus golden hash | ~0.5 d | no |
| A new strategy | Go plus golden tables | 2–4 d | yes |
| A new part type (audio, table) | `parts.Def`, widget, caps, draft policy, contract fields, artboard | 1–2 wk | if it adds a trust or PII surface |
| A new grader kind | grader, pack schema and CI, result view, allowlist, lane | 1–3 wk | yes |
| A new runner profile | T3 profile, codec, baseline | 2–4 wk | yes |
| **A new archetype** | none if it composes existing kinds; otherwise the sum of the rows above | — | only via its new kinds |

---

## 6. From evidence to learning signal

### 6.1 Universal invariants (practice-enforced, property-tested)

| # | Invariant |
|---|---|
| I1 | `passed ⇔ grade ≠ miss` for the **signal** field. **FLAG:** T0 §4's "passed ≥ rough" becomes a projection-level coverage choice. |
| I2 | **Ceilings:** a hint before lock caps at assisted. The solution seen in-attempt before the deciding evidence gives miss. **An arena reveal before the attempt caps at assisted (O4).** Late caps at rough. `self@1{cap:none}` is exempt. |
| I3 | `inconclusive` and `error` never enter the failed count and never fail a criterion. |
| I4 | `give_up` with no earlier pass (by seq) gives miss. |
| I5 | Same facts and params give the same result. A rule change means `@2`, and concluded attempts are never re-graded. |
| I6 | AI never rescues a failed gate. |
| I7 | `graded_by=ai` iff an `ai_rubric` step contributed; else `auto`, `self`, `override` or `claimed`. |
| I8 | **`revisable = role ∈ {core, reinforcement} ∨ (role = drill ∧ manifest.revision.drills)`**, carried on the signal. This restores ADR-0026 §4 and T0 §4. |

### 6.2 Strategies (a closed Go set; the manifest picks one and sets thresholds)

| Strategy | Used by | Lock trigger | Grade | `score` |
|---|---|---|---|---|
| `verdict_timer@1` | A | the earliest-by-seq passing counted submit (attempt or hint stage at its `submitted_at`); or a give-up / reveal (the §3.4 rule) | **clean:** attempt stage, ≤ deadline, failed ≤ N. **rough:** late, **or failed > N** (T0 `:111`). **assisted:** passed at the hint stage, or `arena_prior=revealed`. **miss:** no pass below `close_seq`. | hidden passed/total of the deciding submit |
| `rubric_pct@1` | AI-only B | closing `final` or `give_up` | median-of-k % → clean ≥ `clean_pct` ∧ no hint ∧ on time; rough ≥ `pass_pct` (late caps here); assisted ≥ `pass_pct` with a hint; else miss | s |
| `weighted_gate@1` | composite B, C quizzes | closing `final` or `give_up` | miss if a required gate failed, or status ≠ passed, or s < `pass_pct`; else band by s with the I2 ceilings. Quiz: `clean_pct 1.0, pass_pct 0.8`. | Σ w·s / Σ w |
| `self@1` | no evaluator; degraded fallback | the learner's pick | `cap:none` = v1 (`POST /problems/{id}/outcome`); `cap:stage` = pick ≤ ceiling | — |

**`verdict_timer@1` golden rows.** DSA params: 15 min attempt timer, 10 min hint timer, 3 penalty days, **`max_failed_for_clean: 3`** (rough when failed > 3), `free_classes: [CE, REJECTED]`. v1's copy (`Problem.tsx:21-26`) is the reference.

| Facts | Grade | Concludes |
|---|---|---|
| Pass at 9 m, 0–3 failed | clean | at the pass (O3 a) |
| Pass at 9 m, 4 failed (WA, WA, TLE, RE) | rough | pass |
| Pass at 9 m, 5 CE | clean (CE is free) | pass |
| Pass at 16 m, 0 failed | rough | pass |
| Pass at the hint stage | assisted | pass |
| `submitted_at` 14:59, evaluation done 15:30 | clean | pass |
| 2 × inconclusive, then pass at 10 m | clean (I3) | pass |
| **Pass at seq 4 (14:59) still in flight; give-up at seq 5 (15:00)** | **clean** (earliest pass below `close_seq`) | pass (no reimpl owed) |
| **Give-up at seq 5; the only earlier facts failed** | **miss**, `revealed_early` | reimpl pass, Finish, or 24 h |
| `arena_revealed_at` < start; pass at 9 m | assisted | pass |
| Stale `contract_hash` with evals | — | `self_grade_pending` |

### 6.3 Mistake pre-fill (two tiers, data only)

**Tier 1: judge `signals[]`.** A closed, course-blind vocabulary, computed in judge outside the sandbox by joining runner verdicts with pack case tags. It travels **only** in `evaluation_completed`.

| Family | Signals |
|---|---|
| code | `ce_only, sample_failed, wa, wa_edge_only, tle_perf_only, tle_small, re_index, re_nil, re_stack, re_other, mle, ole, race, deadlock, leak` |
| key | `key_wrong:<field>`, `misconception:<category>` |
| rubric | `dim_low:<dim>`, `must_cover_missed`, `red_flag_hit` |
| attempt (practice) | `late, hint_used, solution_seen, arena_revealed, gave_up, blank_draft, many_failed, crit_unmet:<criterion>` |

**Tier 2: the manifest's `mistakes.prefill: [{signal, category, strength}]`.** The first match wins. For a rough grade from failed submits, the most frequent failure class is used. DSA proposal:

| Order | Signal | Category | Strength |
|---|---|---|---|
| 1 | `tle_perf_only` / `crit_unmet:complexity_stated` | complexity_misjudged | strong |
| 2 | `crit_unmet:pattern_named_fast` | wrong_pattern | strong |
| 3 | `wa_edge_only` | off_by_one | strong |
| 4 | `late` | time_management | strong |
| 5 | `sample_failed` | misread | weak |
| 6 | `mle` | complexity_misjudged | weak |
| 7 | `re_index` / `re_nil`, `ce_only` | off_by_one / language_bug | weak |
| 8 | `hint_used`; `gave_up` + `blank_draft` | wrong_pattern | weak |
| — | `wa`, `tle_small`, `re_other` | analyzer (M4) | — |

- SD and LLD map `dim_low` through `dims[].category`. Quizzes map `misconception:*` (strong).
- **Precedence:** learner > strong rule > analyzer (≥ 0.6) > weak rule > empty. A learner `PATCH` locks the field.
- **Where it applies:** mistakes open **only on revisable items** (I8), so a mistake can always close. The weak area counts learner, strong-rule and analyzer categories; weak rules show as "suggested". This switches the weak area on in **M3**.
- **Analyzer scope (M4):** below-clean archetype-A course conclusions and failed code touches, at most once per conclusion. Results are applied iff `attempt_id = mistake_entry.last_refreshed_by_attempt_id`, which both `problem_solved` and `touch_concluded` set. That fixes the dropped touch analyses.
- **Bounded leak, accepted:** `wa_edge_only` reveals one bit after conclusion.

### 6.4 Concepts to revise

- **Shape:** ≤ 3 ordered refs `<course>:concept:<slug>`. `item.json` gains `concepts[]` (the primary concept first), checked in CI.
- **Base:** the item's `concepts[]`, joined by the gateway at read time. That is composition, which ADR-0018 allows.
- **Per attempt:** `concepts_hint[]` comes from the failed checks' `concept`, plus the analyzer's `concept_keys[]`, validated by judge and review. It is stored on `mistake_entry.concepts` with its source.
- **Surfaces:** the result panel **after conclusion**, the Mistakes journal, a "Weak concepts" card, and the coach's `WeakConcepts` field (`prompt.go:93-94`).
- **Withheld** for the primary concept of an item with a live attempt or a due or live touch. This extends `withhold()` and fixes the chip at `Revision.tsx:235-237`.

### 6.5 Unsolved attempts enter revision (D2, with the revisable guard)

- Every concluded counted attempt on a **revisable** item, of any grade, anchors L1–L5 at `anchor_at`. This replaces first-clean-only (`review/store/store.go:307-309`).
- Below clean also opens or refreshes the mistake, with `revisit_date` = Day 1. v1 passes an empty timestamp at `review/store/store.go:342`.
- A miss's Day-1 touch is a full re-solve from blank (DSA).
- **Scheduling (FLAG, fixes a T0 §4/§5 inconsistency):**
  - A **new** query, `ReanchorPendingTouches`, re-anchors pending levels only on conclusion. v1's `ReanchorTouch` (`revision_item.sql:11-22`) resets any status, so it stays only for the fail → Day-1 path, now with `anchor_at`.
  - `ScheduleTouch … DO NOTHING` stays for `solution_revealed_early`. Give-up = the reveal = `anchor_at`, so the owed Day-3 touch *is* L2.
- Non-revisable drills, pre-lock attempts, mock, arena and Run never schedule.

### 6.6 Touch formats and pass criteria per course

Criteria are **manifest-declared keys** (`revision.bands[L].criteria[]`); DSA's three are just one instance. practice applies every timer to `locked_at` / `submitted_at`.

| Course · band | Format and criteria | Trust | Pass | Provisional |
|---|---|---|---|---|
| **dsa L1–3** | `pattern_named_fast`: probe `p-pattern` (text, `public:pattern` aliases) with `locked_at − start` **< 120 s**, strict as v1 `AutoPass` (`store.go:707`). `correct_in_timer`: a passing touch submit ≤ 20:00 (checked). `complexity_stated`: `p-complexity` (`blank.big_o`, `public:solution_facts`). | honor (min over criteria) | all three; concludes the moment all are met | honor-probe misses → claim (§3.5) |
| **dsa L4–5** | Same criteria, **`mock_mode=true`**: coach off, statement only, "talk aloud, no notes" (v1 `isMockTouch`, `store.go:712`) | honor | all three | same |
| **go-concurrency L1–3** | C: spot-the-race / predict-output (pack key), ~5 min | checked | s ≥ `pass_pct` ∧ required questions correct | never |
| **go-concurrency L4–5** | A re-solve (`go-race`), every declared test passes ≤ timer | honor | pass | never |
| **sql L1–3** | C: predict the result set / pick the index (pack key) | checked | s ≥ `pass_pct` | never |
| **sql L4–5** | A re-solve (`sql-pg`) ≤ timer, plus any `explain:*` checks the item marks required | checked | pass | never |
| **lld-ood L1–3** | C: responsibilities / pattern quiz (pack key) | checked | s ≥ `pass_pct` | never |
| **lld-ood L4–5** | B timed redesign plus the facade gate, `weighted_gate@1` | honor (facade) | s ≥ `pass_pct` ∧ gate | on AI fail |
| **system-design L1–3** | C recall: components (text aliases), bottleneck (choice, pack key), one estimate (numeric, tolerance) | checked | every required probe | never |
| **system-design L4–5** | B timed redesign, `weighted_gate@1` | checked | s ≥ **`pass_pct`** (not `clean_pct`, `t0…:117`) | on AI fail |
| **behavioral** (not a pilot) | STAR re-tell (text). AI with consent, else `self@1` | — | s ≥ `pass_pct` | on AI fail |

**Behaviour change:** DSA "stated" becomes "stated **correctly**", softened by the claim. The v1 self endpoint keeps the old meaning for items with no evaluator.

### 6.7 Mock evidence → `ScoreMock`

At **Finish & score**, the gateway composes judge's DTO for each item: hidden passed/total, first failure, the key aggregate, "design: score it yourself".
- An optional `mock.evidence_hints` maps signals to **suggested** upper bounds on dimensions (e.g. `tle_perf_only → optimisation ≤ 2`). These are shown only.
- `scored_by=self` in v2.0. AI mock scoring is T6's: `ai-byo`, public descriptors, a proposal held outside assessment. `ScoreMock`'s score-once path is untouched.

---

## 7. Structured feedback shape (for T5)

```go
type Scorer interface {                                                         // internal/platform/llm; T5 implements
    Reserve(ctx context.Context, account uuid.UUID, est grader.Cost) (Reservation, error) // ErrBudgetExhausted
    Score(ctx context.Context, r ScoreRequest) (ScoreResponse, error)          // pack material ONLY here
    Feedback(ctx context.Context, r FeedbackRequest) (FeedbackResponse, error) // no pack material, ever
    Analyze(ctx context.Context, r AnalyzeRequest) (AnalyzeResponse, error)    // code analyzer (M4)
}
type ScoreRequest struct {
    Purpose string /* ai_rubric.score | ai_rubric.regrade */; ModelClass string
    System, Rubric string; Anchors, Exemplars []string; Learner []DelimitedPart // random boundary
    Schema json.RawMessage; Samples, MaxOutputTokens int; ZeroRetention bool   // must be true
}
// typed errors: ErrBudgetExhausted, ErrUnavailable, ErrInvalidOutput; ≤ 60 s per call; model + prompt@v returned
```

**Hard caps:**
- **≤ 8 LLM calls per evaluation**, counting escalation, infra retries and feedback.
- **Retry grading ≤ 1 per attempt.**
- **Dispute ≤ 1 per attempt.**

**(a) Scoring, `xlearn.score@1`.** A strict JSON Schema in the portable subset ([Anthropic](https://platform.claude.com/docs/en/build-with-claude/structured-outputs), [OpenAI](https://developers.openai.com/api/docs/guides/structured-outputs)).

```json
{"criteria":[{"id":"<enum>","evidence":["<≤200 chars, verbatim learner quote>"],"band":3}],
 "must_cover":[{"id":"<enum>","met":true}],"red_flags":[{"id":"<enum>","present":false}]}
```

- **Evidence comes before the band** ([Wang et al.](https://arxiv.org/abs/2305.17926)).
- **Quote verification:** apply NFKC, strip zero-width characters, collapse whitespace, and fold quotes and dashes on **both** sides, then substring-match.
  - An unverified quote is **dropped**; the sample is kept.
  - A required criterion with **no verified evidence is clamped to band ≤ 2**. It fails closed, and the learner can still dispute.
  - Quotes stay in `Internal`. **FLAG:** T0's "strict integer or enum" gains verified learner quotes, which are internal only.
- **Sampling:** k=3, issued concurrently, taking the median band and a majority vote on must-cover. A spread of ≥ 2 on a required criterion escalates to k=5; if it persists, set `low_confidence`, which shows a dispute nudge.
- **Rubric design:** analytic criteria with binary must-cover checks ([CheckEval](https://aclanthology.org/2025.emnlp-main.796/)) and anchors at bands 1/3/5 ([Prometheus](https://arxiv.org/abs/2310.08491)). The template is frozen per `prompt@v`, and the rubric states "length is not merit".
- **Injection heuristic** ("ignore previous", "score 5", role markers), or a band ≥ 4 whose evidence came from a single quote under 20 chars: set **`review_flag`**. That makes the evaluation `trust=honor` (excluded from judge-checked %) and puts it on an owner audit list in opscheck. The grade itself still goes through the clamp.

**(b) Feedback, `xlearn.feedback@1`** (≤ 8 KiB). It sees the learner's parts, the public statement, the descriptors, the numeric bands, and the **public** reference unlocked at conclusion. It never sees anchors, must-cover lists, exemplars or hidden cases.

```json
{"summary":"≤480","dims":[{"id":"<dim>","evidence":[{"part":"<id>","quote":"≤160 verbatim"}],"improve":"≤240"}],
 "strengths":["≤160"],"suggested_category":{"id":"<manifest key>","confidence":0.0},
 "concepts":[{"ref":"<course>:concept:<slug>","why":"≤160"}],
 "next_step":{"kind":"reread_concept|retry_part|practice_item|none","ref":"<id>","text":"≤200"}}
```

**(c) Code analyzer, `xlearn.code_analysis@1`** (M4). `summary`, `suggested_category`, `concepts[≤3]`, `next_step`, `findings[≤5]{kind, lines[from,to] within the submission, note ≤240}`, and `complexity{time,space}` matching `^O\([^)]{1,24}\)$`. No code block longer than 8 lines.

**Validation (judge, before storing):**
- the schema and every cap hold;
- every id and ref exists in the manifest or rubric snapshot (unknown → `""`);
- quotes are verified the same way (bad quotes dropped);
- no URLs or images.

On failure, retry once, then `feedback_unavailable`. That never blocks conclusion.

**Error mapping:**
- `ErrBudgetExhausted` → `inconclusive(budget_exhausted)`, with no retry.
- Provider failure after retries → `ai_unavailable`.
- A course attempt then goes to `self_grade_pending`, and a touch falls back to v1 self scoring. The loop never blocks.

**Calibration gate (per rubric `@v` × model × `prompt@v`):**
- ≥ 30 owner-labelled synthetic responses spanning the bands, **including length-padded twins and injection twins**.
- QWK ≥ 0.70, within-1 ≥ 90%, SMD ≤ 0.15 ([Williamson et al.](https://onlinelibrary.wiley.com/doi/abs/10.1111/j.1745-3992.2011.00223.x), [ACT](https://www.act.org/content/dam/act/unsecured/documents/R2100-auto-scoring-standards-2021-07.pdf)).
- An owner re-label two weeks later stands in for intra-rater agreement (inferred).
- The set doubles as the model-change regression suite.

---

## 8. Learner flows and v2 artboards needed

**One shell for every archetype.**
- **HUD:** crumb, difficulty, role, stage tabs, a server timer ring (no pause), a provenance chip.
- **Left:** the statement (hints and solution only once unlocked) and a part navigator.
- **Centre:** the widget plus a **results dock** (Samples · Submission · History · Feedback), which evolves `Problem.dc.html:289-302`.
- **Right:** stage stepper, a live **grade outlook** ("Clean possible · failed 1/3"), the action card, the conclusion card.
- **Below 1024 px:** Statement / Work / Results tabs.
- The pattern chip appears only from the hint stage. `Problem.dc.html:104` shows it during the attempt, which leaks.

**Course attempt (A)** _(fixed 2026-09-24 at the v2 build-plan sign-off to match D15, D16 and D18 in §13. The draft had a "Start attempt · 15:00" cover, a re-implement step and a hint "available any time".)_:
1. **Cover** ("Start attempt · 45:00", the D18 manifest default; per-item overrides allowed). The statement is shown on Start (D10), and the server timer starts then with no pause (D15).
2. Solving: Run (⌘↵) and Submit (⌘⇧↵).
3. **Hint unlocks at 15:00** (D18); taking it caps the attempt at Assisted. Past 20:00 Clean is gone, and the outlook drops.
4. The **Give up · Miss** modal. Revealing the solution is a give-up (§3.4): "your draft is sent for feedback; the solution unlocks now; graded Miss unless a submit already in flight passes; enters revision tomorrow; opens a mistake entry".
5. **Time's up at 45:00** with no pass: concluded as Miss at the deadline (D15), and the solution unlocks for study. **There is no re-implement step** (D16): a pass finishes the attempt, and after a Miss the Day-1 revision is the from-memory re-attempt.
6. Grading: "Queued · 1 ahead" → "Running hidden tests…" → "Recording…" (the settling phase).
7. Concluded: grade + provenance, facts, touch dots, and a mistake block with a pre-filled chip.

Copy fix: `Problem.tsx:566` becomes "Hint unlocks at 15:00; using it caps this attempt at Assisted."

**Arena:**

> **Superseded by D17 (§13):** no arena locks and no grade cap on reveal; the spoiler confirm only records `arena_revealed_at`.
- "Nothing here counts toward your course." Dual markers. A manual timer popover. A History drawer (read-only, Copy to editor, Diff).
- **While the item has an open attempt or a due or live touch:** Submit, Solution and History are locked ("Finish your review first"), and sample Run stays.
- The spoiler confirm on reveal says "caps your course attempt at Assisted".
- B and C items are in study mode with **Mark studied**.
- Ahead-of-schedule items send the learner to the arena instead of v1's `counted:false` attempt (`bff.go:655`), a behaviour change (inferred).

**Touches:**
- Format badges ("Recall · ~5 min" / "Re-solve · 20:00", with "Mock conditions" at L4–5).
- DSA: name the pattern (lock-in, clock under 2:00) → re-solve → state the complexity (lock-in). Correctness is hidden until the end.
- Result: server criteria ("Pattern named 0:48 · passed hidden at 13:20 · O(n²)/O(1)") and "advances to Day 21" or "resets to Day 1".
- An honor-probe miss shows the accepted answers and **[I meant this]** once.
- "End review" confirms "unmet criteria count as a fail".
- On `inconclusive`: "Couldn't verify, not counted. [Retry]".

**SD (B):** part navigator; canvas + palette, Outline view and the grader-sees drawer; estimates with units; Finish pre-flight; step progress ("Estimates done · Design rubric: AI, usually under 90 s"); composite result; provisional.

**Quiz (C):** question cards, "4 / 6 answered", Submit answers; per-question marks, key and rationale after conclusion.

**Provisional and claim:**
- "**Rough** · AI · provisional · auto-accepts in 23:41" with [Accept] [Dispute…].
- The dispute modal lists the criteria (deterministic steps shown locked) and a helper: "shown to the course author for calibration; the re-grade doesn't read it".
- A re-grade compare view, then [Accept] [Set my own grade…] with a warning that names the label and the judge-checked exclusion.
- The claim card for honor probes.
- Today shows "1 grade waiting".

**Mock:** server-picked items (replacing `PROBLEM_SETS`, `Mock.tsx:19-22`); N rubric dimensions; widgets in mock mode; two-column scoring (evidence | sliders), "Scores lock when saved".

**Coach:** off during live touches and mocks; Socratic during course attempts (inferred).

**Results-dock states:**

| State | Copy |
|---|---|
| queued / running / settling | "Queued · 1 ahead of you" / "Running hidden tests…" / "Recording your result…" |
| WA | "**Wrong answer:** 37/40 hidden passed. Hidden inputs aren't shown; try edge cases with a custom Run." |
| perf TLE | "**Too slow on large inputs:** correctness 40/40, performance failed. Check your complexity against n ≤ 1e5." |
| CE (hidden API) | "Hidden tests don't compile against your code. Check the required signature. (Not counted.)" |
| `inconclusive` / close released | "We couldn't get a reliable result. Not counted. [Submit again]" |
| `contract_changed` | "This problem was updated while you worked. Not graded." |
| `not_evaluated_here` | "Checked only in a course attempt or review." |
| 413 / quota | "Too large to submit (code is capped at 64 KiB)" / "8 submits left today" |

**New artboards** (T7 owns production):

| # | Artboard | Frames |
|---|---|---|
| A1 | `Workspace-Code` | Cover ("Start attempt · 45:00") · Solving + Run · Submit WA · perf TLE · Hint unlocked at 15:00 (caps Assisted) / past 20:00 · Give-up modal (reveal = Miss) · Time's up at 45:00 → Miss · solution (study) · Concluded Clean · Concluded Miss + pre-fill · Resume · `self_grade_pending`. **No re-implement frame** (D16; fixed 2026-09-24) |
| A2 | `Results` | every dock state above, including settling |
| A3 | `Grade-Dispute` | provisional card · dispute · re-grade compare · override confirm · honor claim |
| A4 | `Problems` | dual markers, filters |
| A5 | `Arena` | timer, history, locked-while-due, spoiler confirm, study mode + Mark studied |
| A6 | `Workspace-Canvas` | navigator, canvas, Outline, grader-sees, estimates, pre-flight, progress, result |
| A7 | `Workspace-Quiz` | list, lock-in, results with rationale |
| A8 | `Touch` | DSA recall + re-solve (L1–3, L4–5 mock) · SD recall · go/sql/lld recall quizzes · pass / fail / abandoned · claim · AI provisional fail |
| A9 | `Mock-v2` | server-picked setup · multi-item · evidence + rubric |

**Changed v1 artboards:**
- `Problem.dc.html` is superseded by A1: remove `:104`, `:126` and `:332`; the picker at `:349-383` remains only on the self path.
- `Revision` (badges, withheld pattern, server criteria), `Mock` (N dimensions, evidence), `Dashboard` (Resume, grades waiting).
- `Week` (provisional state; wire the touch dots, `aggregate.go:120`), `Mistakes` (source chip, concepts), `Progress` (provenance, judge-checked %).

---

## 9. Queue, admission and single-node budget

### 9.1 Lanes

One `judge.job` table, claimed with `FOR UPDATE SKIP LOCKED`, with a **1 s poll**. There is no LISTEN/NOTIFY, since it would need a dedicated connection and reconnect logic.

| Lane | Workers | Budget | Lease / heartbeat | Infra retries | On exhaustion |
|---|---|---|---|---|---|
| inline (`key`-only submissions) | the request's transaction | < 50 ms | — | — | `error(internal)` |
| `runner` (`code`, plus in-process key steps) | **2** (T0 envelope) | compile ≤ 15 s + tests ≤ 45 s wall (T3 confirms) | 30 s / 10 s | 3 (2 s, 10 s, 30 s) | `inconclusive(infra_exhausted)` |
| `llm` (`ai_rubric`, analyze) | 2 | ≤ 60 s per call, ≤ 150 s per job, ≤ 8 calls | 30 s / 10 s | inside the 8-call cap | `ai_unavailable`; a budget error ends at once |

**Claim, in one transaction:**
1. Pick candidates by `priority − floor(wait/30 s)`.
2. For each, `pg_try_advisory_xact_lock(hashtext(account_id‖lane))`, then `NOT EXISTS (running job for account, lane)`. This serializes per-account claims under READ COMMITTED.
3. `claim_token++`, `state=running`.

**There is no per-context FIFO**, because practice orders by seq. Learner-caused verdicts never retry. Runner 429 or 503 re-queues with `run_after`; it is not a verdict.

Rejected: River's per-key sequences, which are Pro-only ([River Pro](https://riverqueue.com/docs/pro/sequences)), and NATS, which T0 rejected.

### 9.2 Priorities and caps

| P | Jobs | Caps |
|---|---|---|
| 0 | closes (`final`, `give_up`), `regrade` | quota-, pending- and `too_fast`-exempt; queue-full-exempt |
| 1 | counted `submit`, `run` | a new Run cancels the account's queued Runs in that context; ≤ 2 queued Runs per account across contexts |
| 2 | arena `submit` | ≤ 1 running arena job while P0/P1 work is queued (inferred) |
| 3 | `analyze` | never blocks |

- Lane queues: runner ≤ 32, llm ≤ 8. Per account: ≤ 4 queued non-Run jobs, ≥ 2 s between submits in a context.
- Daily quotas: 200 Runs, 100 counted, 100 arena (T1). The mock session limits are in §2.3.
- Kill switch: 503 for new work; queued jobs drain.
- Queue latency never penalizes the learner (`submitted_at`).

### 9.3 Throughput and memory (inferred; T3 measures)

- A Go submit costs ≈ 2–6 CPU-s. Two workers give ≈ 20–60 submits a minute, ample under invite-only signup (D13).
- An SD final costs ≈ 30–90 s of llm-lane time and ≈ $0.05–0.15.
- **judge memory:** 128 Mi request, 256 Mi limit (the T0 envelope had about 64 Mi). Generated inputs go to `emptyDir` files and are streamed, never held twice in memory. The node envelope grows by about 64 Mi requested.

### 9.4 The per-case input cap (T1 flag) resolved

- Literals stay ≤ 256 KiB.
- Perf cases are generated from specs and seeds (§4.1).
- A stamped `large_case_exception` allows ≤ 2 MiB.
- **T3 accepts streamed per-case input ≤ 8 MiB and ≤ 16 MiB per job.** n=1e5 ints in JSON ≈ 700 KB; 2e5 edges ≈ 2.8 MB.
- The pack stays far under its 150 MB cap.

### 9.5 Volume re-model (`scratchpad/t4/vol_t4.py`, re-run this pass; parameters inferred)

It adds persisted arena submits (2 a day), probe lock-ins (8 a day), and practice fact and inbox rows.

| Metric | T1 | T4 |
|---|---|---|
| Postgres per active learner-day | 86 KB | **~124 KB** (+44%) |
| 100 learners / year | 2.1 GiB | **~3.0 GiB** (30% of 10 Gi) |
| Learner-years to the 60% resize trigger | ~285 | **~198** |
| `XLEARN_JUDGE` | — | ~2 MB per learner-year (MaxAge 14 d bounds it) |
| Code draft WAL | 0.5 MB/day assumed | ~0.37 MB/learner-day |

- The outbox and inbox are 38% of growth, which confirms T1's "revisit pruning at the resize trigger".
- **Not modelled:** SD finals (≈ 50–100 KB each) and canvas autosave WAL (≈ 3 MB/day for an active SD learner, ~6× T2's assumption). Neither matters until backups (D12) and the SD course land; flagged to T7.
- The T4 fixes (pull reconciler, released closes, no Run seq) add a negligible number of rows.

### 9.6 Sweeper (one judge ticker)

- Delete drafts idle for 90 days (batched).
- Delete Run jobs finished more than 24 h ago.
- Re-queue running jobs whose lease expired (`claim_token++`).

---

## 10. Security and integrity notes

| Path | Control |
|---|---|
| Forged `evaluation_completed` | NATS auth plus the `messaging` NetworkPolicy are a **hard M3 precondition**. practice re-validates account, item, path, purpose, pin and context. Gaps are filled only by **pulling from judge's authenticated internal API**. **FLAG:** this replaces T1 §10's "concludes only for `awaiting_evaluation`", because an iterate pass legitimately concludes from `attempting`. A `final` lock still needs a recorded close. |
| Client-chosen context or pin | The gateway resolves the context; the server stamps pin, time and seq. Arena ids are account-scoped. |
| Key brute force | `key`/`ai_rubric` never in arena or Run; a `final` part locks once; **mock shows key aggregates only for unconcluded items, plus mock session limits**; quotas. |
| Arena as an uncounted debugger | Arena Submit, History and Solution locked while an attempt is open or a touch is due or live; an arena reveal caps the later course attempt. |
| Hidden data in feedback | DTO allowlist; per-endpoint denylist (renamed `sample_expected`/`answer_reveal`); run-all correctness cases plus a one-bit perf block; class-only RE; no stderr for hidden cases; **diagnostics limited to learner files, with fixed CE text for pack-file errors; generic check keys**. |
| Residual side channel | `passed` over correctness cases gives up to log2(total+1) ≈ 5.4 bits per arena submit at 40 cases, plus 1 perf bit (**correcting the draft's "≈ 2 bits"**). `wa_edge_only` gives 1 bit at conclusion. Accepted: the reference is public, and secrecy protects signal integrity, not against a determined cheat. |
| Learner-induced `inconclusive` | INV-14: only runner-side evidence (typed infra errors, `throttled` telemetry) yields it. Timeouts and OOM of the learner's own process are TLE or MLE; an intermittent deadlock is DEADLOCK. |
| Generator drift | Golden hash per `gen@v`; the version is recorded on each evaluation. |
| In-process profiles | T1's controls; `trust=honor`; the compile-leak lint (§5.6 #15). |
| AI prompt injection | Delimited learner content; normalized quote verification; drop-quote clamp; `review_flag` → honor plus an audit list; AI can't lift a gate; provisional finality; a **blind** re-grade; call caps. |
| Pack material in AI calls | Only in `Score` (platform key, zero retention). Never in `Feedback`, `Analyze` or `ai-byo`. |
| Canvas payload | Palette-typed; labels ≤ 80 chars; ≤ 2,000 elements; no images or links; validated in Go, with no fallible CHECKs. |
| Early exposure | `withhold()` extended to concepts, arena Submit, History, Copy-to-editor and the pattern chip while a touch is due or live; probe correctness hidden until the touch concludes; the key only after a counted conclusion. |
| Self-report bypass | Self outcomes are accepted only in `self` mode or `self_grade_pending` (INV-9). |
| Double conclusion | `concluded_at IS NULL` guarded `UPDATE … RETURNING` plus `UNIQUE(outcome.attempt_id)`. |
| IDOR | Every read by id is scoped to `sub`; IDOR tests. |
| Erase races | Tombstone at claim and at write; the result write locks the parent `FOR UPDATE`; erase cancels jobs; dispute text is in the erase path; SQL-only delete (T2). |
| Spend | `Reserve()` before any LLM call; Retry grading ≤ 1; ≤ 8 calls per evaluation; no unauthenticated endpoint spends. |
| Oversized or escaped bodies | Raw-JSON payloads; the SPA measures encoded size; `MaxBytesReader` + typed 413 at all 11 body sites. |

---

## 11. What T4 constrains downstream

### 11.1 T3: the runner contract (not the technology)

```go
type Job struct {
    ID, Profile, Harness string        // go|go-race|sql-pg|cpp ; func-json@1|class-ops@1|gotest@1|sql-result@1
    Files  []File                      // learner files + public harness (read-only)
    Cases  []CaseInput                 // {OpaqueID, Group: sample|edge|random|perf, Input: streamed file}; expected never enters (except honor in-process)
    Limits Limits                      // per case cpu_ms, wall_ms, mem_mb, output_kb; compile; job wall = Σ + slack
    Mode   string                      // run | submit
    StopGroupOn map[string]string      // {"sample":"any_fail","perf":"tle"}
    Count  int                         // -count=N (go-race)
}
type Result struct {
    Compile struct{ OK bool; Diags []Diag /* {File, Line, Col, Msg}; judge drops non-learner files */ }
    Cases   []CaseResult  // Term: ok|tle|mle|ole|signal|exit_nonzero|not_run — LEARNER-attributable only;
                          // CPUms/WallMs/PeakKB measured OUTSIDE the untrusted process; PanicClass; Output on harness channel;
                          // Stdout/Stderr in Run mode only
    Tests   []TestEvent   // test2json
    Race    bool
    Dump    *GoroutineDump // on per-test timeout: blocked frames with package paths
    Throttled bool         // cgroup cpu.stat throttling / steal above threshold during the job
    Infra   *InfraError    // {Kind: setup|runner_oom|killed|job_timeout|saturated} — never a learner verdict
    Versions struct{ Runner, ImageDigest, Profile string }
}
```

Requirements:
- A synchronous call inside the lease that honours ctx cancellation (kill the process group).
- ≤ 2 concurrent jobs, with 429/503 when saturated.
- A fresh tmpfs per job, no network.
- Streamed input ≤ 8 MiB per case and ≤ 16 MiB per job.
- A deterministic environment and locale.
- A published time-limit baseline per profile (3× margin).
- Must work without KVM and under AppArmor's userns restriction (T0).
- UX target (inferred): Run p50 < 5 s.

### 11.2 T5: the AI interface (no key policy)

- Implement `Scorer{Reserve, Score, Feedback, Analyze}` with the three strict schemas.
- Typed errors, zero retention on `Score`, ≤ 60 s per call, k-sampling (3, escalating to 5), and **≤ 8 calls per evaluation**.
- Return `model` and `prompt@v`, with bucketed usage.
- Re-grade uses fresh samples and never the appeal text.
- Budget exhaustion degrades (→ `self_grade_pending`) and never blocks.
- Decide the key holder (candidate: judge) and per-account budgets.
- Confirm that platform-key mock scoring is out of scope for v2.0.

### 11.3 T6 and T7

**T6:**
- Mock AI scoring is a proposal held outside assessment, then `ScoreMock` once (`ai-byo`, public descriptors).
- Mock `evaluation_completed` is switched on consumer-first, if T6 needs it.
- The key aggregate rule applies to mock evidence.

**T7:**
- Write **ADR-0029, "Judge contract and learning-signal v2"**, from this document.
- **M3 minimum cut:**
  - judge with the `code` and `key` graders (inline key, one runner lane);
  - `verdict_timer@1`, `weighted_gate@1` (quiz) and `self@1`;
  - practice's first consumer with the pull reconciler and ticker;
  - the sync close and give-up;
  - drafts; arena submits with per-account contexts;
  - strong-rule `mistake_hint`, `concepts_hint`, `revisable`;
  - the DTO allowlist and denylist;
  - the `MaxBytesReader` helper;
  - NATS auth.
- **M4:** `ai_rubric`, the llm lane, analyzer, provisional/dispute, claims, calibration.
- The second course after M3 is go-concurrency or SQL (A + C). SD needs M4 and the canvas.
- Knobs: `RELAY_INTERVAL` 1 s; SPA timeouts 25 s; judge 128/256 Mi.
- practice → judge internal reads (`watermark`, `evaluations`): a service identity and a NetworkPolicy.
- opscheck alerts (§3.2) and the AI `review_flag` audit list.
- The outbox republish runbook for a practice outage longer than the 14 d MaxAge.
- The single-replica cache epoch is documented as a scale-out blocker.
- Artboards A1–A9.

### 11.4 Amendments to settled docs (every FLAG)

1. **T0 §10:** `run` is an action, not a context.
2. **T0 §5 "FIFO ⇒ last seen":** replaced by `attempt_seq`, a gap-free check and the pull reconciler. judge drops per-context ordering; practice always acks.
3. **T0 §5 triggers:** the grade lock is separate. A pre-solution pass locks and concludes (O3). Give-up is a **sync practice write** that locks on the earliest pass below `close_seq`, else miss. Conclusion follows the settle rule.
4. **T0 §4 "passed ≥ rough":** the signal uses `passed ⇔ grade ≠ miss`. Coverage is left to projections.
5. **T0 §4/§5 scheduling queries:** a new `ReanchorPendingTouches` on conclusion; `ScheduleTouch DO NOTHING` for the early reveal; `ReanchorTouch` kept for the fail reset. `Score` and `ReanchorTouch` take `anchor_at`.
6. **T0 §10 T5 "integer/enum" scoring:** plus verified learner quotes, internal only.
7. **T1 §3.3 "no generators":** the pack carries `gen` specs and seeds; the generator code is public and closed.
8. **T1 hidden class set:** plus OLE, RACE, DEADLOCK, LEAK. Hidden results are a correctness count, a perf block, and the first failure class.
9. **T1 §4 practice schema:** the states and columns in §3.1, the fact table, `UNIQUE(outcome.attempt_id)`, `user_problem_state.arena_revealed_at`. Reimpl uses a child context, so drafts need **no `slot`**.
10. **T1 §10 guard:** "concludes only for `awaiting_evaluation`" becomes the context/pin/purpose/account match plus a recorded close for `final` locks.
11. **T1 §4 judge schema:** `submission.{closes_context, released, attempt_seq}` with the partial unique index; `context_counter` (evaluation-bearing rows only); `arena_progress.source`; `evaluation_dispute`; `evaluation.review_flag`; internal `watermark` and `evaluations` endpoints. The arena context id is `uuidv5(account, item)`.
12. **Events:**
    - `problem_solved` v2 adds `anchor_at, revisable, arena_prior, gave_up, reimplemented, resolution, mistake_hint, concepts_hint[]`.
    - `touch_concluded` adds `trust, evaluation_ids[], anchor_at, score?, resolution, mock_mode, mistake_hint, concepts_hint[]`.
    - **`attempt_logged` is kept, course-only, adding `path_slug` and `anchor_at`.**
    - `evaluation_analyzed` adds `source` and `concept_keys[]`.
    - T1's to-confirm fields are confirmed.
13. **Projections** (v1 `projections.go:176,198,219` and T1's `proj_activity`): v2 envelopes date activity and coverage by **`anchor_at`**, not `occurred_at`. A replay golden test covers timeout, accept and override conclusions.
14. **PRD R-SR1 "from its conclusion":** scheduled from grade lock (`anchor_at`).
15. **Review schema:** `mistake_entry.{last_refreshed_by_attempt_id, category_source (learner\|rule_strong\|rule_weak\|analyzer), category_suggested, concepts, concepts_source}`; `touch_result.{anchor_at}` with `UNIQUE(attempt_id)`.
16. Mock evaluations don't emit in v2.0.
17. `override`, `claimed` and `review_flag` all imply `trust=honor`.
18. **PRD V2 "see per-test results":** per-**sample** results; hidden = count + perf bit + class.
19. **Content and manifest:** `item.concepts[]`; `mistakes.prefill`, `mock.evidence_hints`, `revision.drills`, `revision.bands[*].{parts, criteria, mock_mode}`.
20. **T0 §7 withholding:** extended to arena Submit, History, Copy-to-editor and the concept chips; plus the arena reveal record.
21. **T0 §8 budget:** judge 128 Mi request / 256 Mi limit.

---

## 12. Alternatives considered

| Option | Why not |
|---|---|
| Archetype as a registry key | It re-creates course-as-code (ADR-0026). |
| Nested or authored composites | Ambiguous status. One root plus `required` makes INV-5 structural. |
| NATS work queue for jobs | The 25 s / 30 s handler limits; T0 rejected it. |
| River OSS | Per-key sequences and concurrency limits are Pro-only. |
| **Judge per-context FIFO** | Redundant once practice orders by seq; it is the most complex claim SQL. |
| **A `local` job lane for key steps** | They take under a millisecond. Inline evaluation in the enqueue transaction is simpler. |
| **Server-derived close keys + `close_epoch`** | The epoch had to be coordinated across services and broke retries. Client keys plus an "unreleased close" partial unique index need no coordination. |
| **Nak-and-wait on gaps** | `maxDeliver=100` exhausts in about 8 h and drops the trigger. Ack-always plus a pull reconciler can't stall. |
| **Advisory `/awaiting` call** | practice couldn't lock a give-up or see in-flight submits. The sync close plus the watermark fix both. |
| **practice stamps probe times itself (critic B)** | Values must be frozen where they're evaluated (judge owns drafts, T1). Inline key lock-ins cost one small event each and keep one evaluator. |
| SSE now | 1–10 s results and one gateway replica; polling with ETag is enough. |
| Broker ordering (`MaxAckPending=1`) | It serializes every learner, and the library can't set it. |
| Kattis early exit / a per-case perf count | Leaks ordinals. A one-bit perf block doesn't. |
| Raise the literal cap to 2–4 MiB | The pack would exceed 150 MB. |
| DMOJ runtime generators | Per-item code and the reference on the hot path. |
| Reference oracle for custom Run | Hands out answers during a timed attempt. |
| AI final at once / AI advisory only / k-median with no dispute | No recourse / fails PRD V3 / cost without recourse. |
| Override without a re-grade; re-grade reading the appeal | Makes the AI a formality / an injection channel. |
| Strict key match with no claim for honor probes | A missing synonym resets the ladder; worse than v1's self-affirm. |
| `flaky_timing` → inconclusive for mixed timeouts | The learner can induce it, and it hides go-concurrency's central bug class. |
| Auto-miss sweeper | Fabricates activity and races evaluations (T0). |
| Void every abandoned touch | Lets a learner read, abandon, research and retry. |
| Fail an unshown or infra-only touch | Punishes accidents and outages with a Day-1 reset. |
| Anchor at conclusion time | Provisional and reimpl windows would push Day 1 back by up to 48 h. |
| Analyzer on every failing evaluation | N× the cost. |
| Canvas graded as an image | 15–29 points worse (IsoBench). |
| tldraw / Monaco | A licence key / 2–5 MB with no mobile support. |
| Platform-key mock scoring | T0 limits the platform key to course and touch. |

---

## 13. Owner decisions — resolved 2026-09-24 (they OVERRIDE the body where they conflict)

| # | Question | Decision (as the owner stated it, applied) |
|---|----------|---------------------------------------------|
| **D14** | O1: who finalizes a non-authoritative (AI or honor-key) grade | **The AI produces a *suggested result with its reasoning*.** That covers the grade, mistake category, notes and concepts. The learner can **accept**, **edit**, or ask for **one blind re-grade** (the dispute), and **submitting** triggers the downstream actions: conclusion, mistake entry, revision schedule. **Manual entry is always available as the fallback** when AI is unreachable, unavailable or over budget. Defaults, which the owner may overrule:<br>• an edited grade is bounded by the deterministic ceilings (no Clean after a hint, coach help, or a late pass);<br>• edits are labelled `graded_by=override`, `trust=honor`;<br>• an untouched suggestion auto-submits after 24 h;<br>• deterministic code verdicts are not editable, but the mistake and notes always are. |
| **D15** | O2: abandoned attempts | **The timer is a hard limit.** Starting a problem starts the server timer. **No passing submit by the limit means the attempt concludes as failed (Miss) at the deadline**, and it is re-attempted through the revision schedule (Day 1, D2). An explicit give-up is also a Miss. practice's ticker concludes expired attempts, dated at the deadline. This **replaces §3.6's "resumable forever before lock"**. Touches follow the same rule: a started touch that reaches its deadline fails; a touch that was never shown, or has only infra-inconclusive evidence, is voided. |
| **D16** | O3: when an attempt finishes, and re-implement | **A pass finishes the attempt.** There is no mandatory re-implement. After a failure (timeout or give-up) the solution unlocks for study, and **the Day-1 revision is the from-memory re-attempt**, so there is no immediate re-implement stage. **The AI assessment also reviews passing solutions.** If the logic can be improved, it attaches **pointer notes to that problem** (persistent, learner-visible, shown on later attempts and touches) and suggests an **optional revisit**, which does not change the grade or the mandatory ladder. This widens the analyzer's scope to passes (cost goes to T5). |
| **D17** | O4: the arena on enrolled items before the course attempt | **No restriction.** The arena is not locked during a live attempt or a due touch. An early solution reveal is recorded (`arena_prior`) but **does not cap** the course grade. Accepted trade-off: the arena works on the honor system. This **replaces** §3.7's locks and §6.1 I2's arena cap. |
| **D18** | DSA time budget (with the hard limit) | **A 45-minute default per problem; the hint unlocks at 15 minutes.** Grades:<br>• **Clean:** pass ≤ 20 min, no hint, no coach, ≤ 3 failed submits.<br>• **Rough:** pass ≤ 45 min without hint or coach, but after 20 min or with more than 3 failed submits.<br>• **Assisted:** pass ≤ 45 min after the hint **or with AI-coach help during the attempt** (any coach message on that problem while the attempt is open; the UI warns first).<br>• **Miss:** no pass by 45 min, or give-up.<br>Manifest defaults with per-item overrides. **Per-problem time budgets are a separate future research session.** |
| **D19** | O5: canvas library | **Excalidraw** (MIT, `@excalidraw/excalidraw`), chosen to minimise complexity. Typed shapes come from a constrained library via `customData`; freehand strokes are ignored by grading. The stored payload is the library-neutral `canvas-graph@1` derived from the scene, and the AI grades its **text export**, never an image. The Outline (table) view is optional. |

Everything else in this document is **adopted as recommended**: the contract, the three archetypes, the Postgres job queue, seq plus gap-free conclusion, the DTO allowlist, strategies, mistake pre-fill, concepts-to-revise, seeded perf generators, the M3/M4 split, and the §11.4 amendments.

## 14. Risks

| Risk | Mitigation |
|---|---|
| practice's first consumer and reconciler are the critical path | Ack-always plus pull; gap and `awaiting` alerts; lifecycle golden tests (give-up × in-flight pass, released close → re-close, reimpl → conclude, touch drain). |
| practice outage longer than `XLEARN_JUDGE`'s 14 d MaxAge | The pull reconciler reads judge's tables directly; the outbox republish runbook. |
| AI variance still drives scheduling | Calibration gate, k-sampling, clamp, provisional plus a blind re-grade, touch **pass** threshold. Reduced, not removed. |
| Alias tables and the big-O normalizer miss answers | The claim; the owner reviews claim frequency per probe to grow the alias tables. |
| Mis-tuned `max_failed_for_clean` or prefill tables | Manifest data, golden-tested; the learner wins on the category. |
| Generator drift silently marks correct code WA | Golden hash per `gen@v`; version recorded on each evaluation. |
| Honor surfaces inflate perceived rigour | `trust` everywhere; judge-checked % excludes honor, override, claimed and flagged. |
| Run-all correctness costs CPU on bad submits | Sample early exit, perf-block stop, job wall, per-account lane cap, quotas. |
| go-race false DEADLOCK (slow runner, not a real deadlock) | Deadline ≥ 10× baseline; the dump must show learner-package frames; the `throttled` re-run. |
| practice ↔ judge internal coupling (watermark, pull) | Read-only, idempotent, used only at deadlines and on gaps; a judge outage just delays conclusion. |
| Canvas UX, a11y and SD WAL volume | Grader-sees drawer, Outline view, a canvas spike before SD; re-check WAL when backups land. |
| Behaviour changes vs v1 (statement at Start, ahead-of-schedule → arena, "stated correctly", reveal = give-up, arena reveal cap) | Release notes; the self path keeps v1 semantics. |
| Single-replica cache epoch | Documented as a scale-out blocker (T7). |
| Canvas library cadence | Library-neutral payload; the widget is swappable. |

---

## 15. Critic findings addressed

### 15.1 Critic 1: six-course loop correctness

| Finding (severity) | Resolution | § |
|---|---|---|
| Close uniqueness blocks reimpl submits, close retries and Retry grading (**blocker**) | Partial unique on **unreleased** closes; judge releases a `final` close on inconclusive/error itself (no epoch coordination); reimpl in a **child context** with its own seq; cap of 3 releases, then `self_grade_pending`; Retry ×1; golden lifecycle tests | 2.4, 3.3, 14 |
| Give-up waits on judge; pass-vs-give-up race undefined (major) | **Sync practice close write** unlocks the solution at once; lock = earliest pass below `close_seq`, else miss; both orderings are golden rows | 2.4, 3.4, 6.2 |
| Touch deadline conclusion: no actor, no drain visibility, inconclusive-only fails (major) | practice ticker plus judge **watermark**; the gateway refuses late touch submits; infra-only → void; shown-then-abandoned → fail | 3.2, 3.6 |
| Projections date by `occurred_at` (major) | Projections v2 use `anchor_at`; replay golden; amendment #13 | 2.8, 11.4 |
| Arena as an uncounted debugger during a live attempt or due touch (major) | T0's "open or due or live" predicate locks arena Submit, History and Solution; sample Run stays | 3.7, 10 |
| Revisable guard dropped (major) | I8 `revisable` on the signal; review schedules and opens mistakes only when revisable | 6.1, 6.5 |
| Touch formats only for dsa and SD (major) | Per-course band table; criteria are manifest keys | 6.6 |
| go-concurrency mixed timeouts → inconclusive (major) | Goroutine dump → DEADLOCK; inconclusive only with `throttled` after a re-run; T3 `Dump`/`Throttled` | 4.1, 11.1 |
| Honor-probe key misses reset the ladder (major) | `blank.big_o` normalizer; one "I meant this" claim via the provisional machinery | 3.5, 4.3 |
| Every SD mock submission refused (minor) | Mock exempt from refusal; non-AI evidence; composite `inconclusive(not_evaluated_here)` | 2.3, 5.3 |
| Arena B/C done marker; `item_id` context shared (minor) | "Mark studied" (`source=manual`); LLD facade in arena; `uuidv5(account,item)` | 2.1, 3.7 |
| Self outcome can pre-empt judge (minor) | INV-9: 409 unless `self` mode or `self_grade_pending` | 2.9 |
| Touch analyses dropped (minor) | Keyed on `last_refreshed_by_attempt_id`; columns listed | 6.3, 11.4 #15 |
| Acked invalid fact → permanent gap (minor) | Pull reconciler plus an `invalid` filler and alert | 3.2 |
| L4–5 mock conditions dropped (minor) | DSA L4–5 band with `mock_mode` | 6.6 |
| `attempt_logged` unspecified (minor) | Kept course-only, with `path_slug` and `anchor_at` | 2.8 |
| Six unflagged discrepancies (minor) | R-SR1 → #14; T1 §10 → #10; "> N" restored; new `ReanchorPendingTouches` → #5; two analyzer durables; one live touch per problem | 2.8, 3.3, 6.2, 11.4 |

### 15.2 Critic 2: integrity, security and single-node ops

| Finding (severity) | Resolution | § |
|---|---|---|
| Close epoch vs partial unique (**blocker**) | Same as critic 1's blocker | 2.4 |
| practice can't see accepted seqs; `/awaiting` advisory; T1 guard dropped silently (**blocker**) | Sync close with `close_seq`; judge `watermark` and `evaluations` internal APIs; T1 §10 amendment flagged with a replacement guard | 2.4, 3.2, 10, 11.4 #10 |
| Arena context shared across accounts (major) | `uuidv5(account, item)`; no cross-account FIFO (FIFO dropped entirely) | 2.1, 9.1 |
| Runs creating seq gaps (major) | INV-12: only evaluation-bearing rows take seq; gap alert | 2.9, 3.2 |
| Admission: Runs across contexts; quotas blocking close and give-up (major) | ≤ 2 queued Runs per account across contexts; per-lane queue caps; closes, give-ups and the re-grade exempt | 2.3, 9.2 |
| Learner-inducible inconclusive; ambiguous `killed` (major) | INV-14; learner OOM → MLE; typed `Infra.Kind`; `throttled` | 2.9, 4.1, 11.1 |
| AI fail-open: injection at ceiling, fragile quote rule, unbounded retries (major) | Normalized verification; drop quote, not sample; clamp unsupported bands ≤ 2; `review_flag` → honor + audit; Retry ≤ 1; ≤ 8 calls | 7 |
| Mock leaks per-question keys (major) | Aggregate only for unconcluded items; mock session limits | 2.3, 2.6, 3.7 |
| Hidden-test compile diagnostics leak (major) | Learner-file positions only; fixed CE text; generic check keys; CI fixture | 2.6, 5.6 #15, 11.1 |
| Cache re-caches pre-conclusion state (major) | `facts_through_seq` in the poll; SPA "settling" phase; invalidate on practice state change | 2.7 |
| `Conclude` copies v1's unguarded shape (major) | Guarded `UPDATE … RETURNING` plus `UNIQUE(outcome.attempt_id)`; self endpoint 409 | 3.2, 2.9 |
| Perf skip leaks an ordinal (minor) | One-bit perf block | 2.6 |
| Denylist vs `expected`/key fields (minor) | Renamed to `sample_expected`/`answer_reveal`; denylist per endpoint and state | 2.6 |
| `attempts` as both fence and retry budget; racy NOT EXISTS; LISTEN/NOTIFY (minor) | `claim_token` vs `infra_retries`; advisory xact lock; 1 s poll | 2.5, 9.1 |
| judge 64 Mi vs 16 MiB inputs (minor) | 128/256 Mi; inputs generated to file and streamed | 9.3 |
| Solo-dev over-build (minor) | FIFO dropped; inline key lane; M3 minimum cut published. Probe lock-ins kept in judge (§12 explains why) | 9.1, 11.3, 12 |
| Void abuse after reading probes (minor) | `probes_shown_at`: shown + no evidence → fail | 3.6 |
| 1 MiB vs escaped JSON; SPA timeout > ReadTimeout; single-site fix (minor) | Raw JSON values plus encoded-size check; 25 s timeouts; shared helper at 11 sites | 2.2, 2.3, 2.7 |
| AutoPass `< 120`; `Score` anchors at `now()`; missing columns (minor) | Strict `< 120` s; `Score`/`ReanchorTouch` take `anchor_at`; columns in #9, #11, #15 | 6.6, 3.7, 11.4 |

### 15.3 Verified-wrong claims in the draft, corrected

| Draft claim | Correction |
|---|---|
| Timeout or auto-accept "fabricates nothing" | Projections dated by `occurred_at` would. Fixed by `anchor_at` (#13). |
| "revisit_date NULL at `mistakes.go:107`" | It is at `review/store/store.go:342` (`openMistakeTx(…, pgtype.Timestamptz{})`). |
| Re-anchor "reuses the upsert behind `ReanchorTouch`" | `revision_item.sql:11-22` resets any status; the new `ReanchorPendingTouches` is needed. |
| DSA touch "every level (v1 parity)" | v1 runs L4–5 under mock conditions (`store.go:712`); the bands are now split. |
| "≥ N failed submits → rough" | T0 says "more than N" (`t0…:111`); the golden rows are fixed. |
| Anchoring at lock time needs no flag | It conflicts with PRD R-SR1 "from its conclusion"; flagged (#14). |
| `awaiting_evaluation` advisory needs no flag | It conflicts with T1 §10 (`t1…:784`); flagged (#10). |
| `Conclude` has "the same transaction shape" as `LogOutcome` | v1 also emits `attempt_logged` (`store.go:420`) and is race-unsafe (unlocked read, unguarded updates, no unique outcome). |
| Partial `UNIQUE(context)` for closes | Contradicted the draft's own retry paths; replaced by the unreleased-close index. |
| "Lock at give-up doesn't wait for the evaluation" | Impossible with an advisory `/awaiting`; now a sync practice write. |
| Side channel "≈ 2 bits per arena submit" | log2(total+1) ≈ 5.4 bits at 40 cases, plus 1 perf bit. |
| "Locked in ≤ 120 s" | v1 `AutoPass` is strict `< 120` (`store.go:35,707`). |
| `MaxBytesReader` fixes "the" `io.LimitReader` at `bff.go:664` | There are **11** request-body sites (critic B counted 10; `mistakes.go:102` is the 11th). |

### Sources

- **Queues and protocols:** [River Pro sequences](https://riverqueue.com/docs/pro/sequences) · [PostgreSQL SELECT / SKIP LOCKED](https://www.postgresql.org/docs/current/sql-select.html) · [IETF Idempotency-Key draft-07](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header-07) · [NATS delivery and acknowledgment](https://docs.nats.io/learn/jetstream/delivery-and-acknowledgment) · [Synadia: strict order](https://www.synadia.com/blog/process-jetstream-messages-strict-order)
- **Judges and verdicts:** [Judge0 CE](https://ce.judge0.com/) · [DMOJ status codes](https://dmoj.readthedocs.io/en/latest/judge/status_codes/) · [DMOJ generators](https://dmoj.readthedocs.io/en/latest/problem_format/generator/) · [CMS contest configuration](https://cms.readthedocs.io/en/latest/Configuring%20a%20contest.html) · [DOMjudge configuration](https://www.domjudge.org/docs/manual/main/configuration-reference.html) · [CLICS contest API](https://ccs-specs.icpc.io/2023-06/contest_api) · [Kattis judgements](https://support.kattis.com/support/solutions/articles/79000141261-list-of-possible-judgements) · [Go race detector](https://go.dev/doc/articles/race_detector)
- **Assessment design:** [QTI 3](https://www.imsglobal.org/spec/qti/v3p0/impl) · [Moodle numerical tolerances](https://docs.moodle.org/dev/Question_Engine_2:Numerical_tolerances) · [Butler & Roediger 2008](https://link.springer.com/article/10.3758/MC.36.3.604) · [Briggs et al. 2006](https://www.tandfonline.com/doi/abs/10.1207/s15326977ea1101_2) · [Gradescope regrade requests](https://guides.gradescope.com/hc/en-us/articles/21854736042253-Submitting-a-Regrade-Request)
- **LLM grading:** [IsoBench](https://arxiv.org/abs/2404.01266) · [UML grading via text](https://www.scitepress.org/Papers/2025/134819/134819.pdf) · [CheckEval](https://aclanthology.org/2025.emnlp-main.796/) · [Prometheus](https://arxiv.org/abs/2310.08491) · [Wang et al.](https://arxiv.org/abs/2305.17926) · [Zheng et al.](https://arxiv.org/pdf/2306.05685) · [Rating Roulette](https://aclanthology.org/2025.findings-emnlp.1361/) · [Raina et al.](https://arxiv.org/html/2504.18333v1) · [JudgeDeceiver](https://arxiv.org/pdf/2403.17710) · [Anthropic structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) · [OpenAI structured outputs](https://developers.openai.com/api/docs/guides/structured-outputs)
- **Automated-scoring standards:** [Williamson, Xi & Breyer 2012](https://onlinelibrary.wiley.com/doi/abs/10.1111/j.1745-3992.2011.00223.x) · [ACT standards](https://www.act.org/content/dam/act/unsecured/documents/R2100-auto-scoring-standards-2021-07.pdf)
- **Editors and canvas:** [React Flow accessibility](https://reactflow.dev/learn/advanced-use/accessibility) · [xyflow #5229](https://github.com/xyflow/xyflow/issues/5229) · [Excalidraw #9186](https://github.com/excalidraw/excalidraw/issues/9186) · [tldraw licence key](https://tldraw.dev/sdk-features/license-key) · [CodeMirror Tab handling](https://codemirror.net/examples/tab/) · [Sourcegraph: Monaco → CodeMirror](https://sourcegraph.com/blog/migrating-monaco-codemirror) · [Replit: CodeMirror](https://blog.replit.com/codemirror) · npm registry (`@xyflow/react` 12.11.6, `@excalidraw/excalidraw` 0.18.1, queried 2026-09-24)

**Files relied on:**
- `docs/v2/feasibility.md`
- `docs/adr/0026-per-course-extensibility-model.md`
- `docs/adr/0027-content-evalpack-and-user-data-model.md`
- `docs/v2/research/t0-extensibility-frame.md`
- `docs/v2/research/t1-content-data-model.md`
- `docs/prd/xlearn-v2-prd.md`
- `internal/assessment/store/projections.go`
- `internal/practice/store/store.go`
- `internal/review/store/store.go`
- `internal/review/store/queries/revision_item.sql`
- `internal/platform/events/consumer.go`
- `internal/platform/httpx/httpx.go`
- `internal/gateway/{bff,assessment,coach,mistakes}.go`
- `scratchpad/t4/vol_t4.py`