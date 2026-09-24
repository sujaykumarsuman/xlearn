# Sprint m4-04 — Provisional grades, dispute, re-grade, honor claims (D14)

> **Milestone:** M4 — platform AI (owner cohort)   ·   **Track:** product (order 63)
> **Prereqs:** [m4-03](sprint-m4-03.md) (judge/review migrations + gateway routes serialized) · [m3-08](sprint-m3-08.md) (practice attempt states, fact table, strategies, ticker; live in v1.14.0)   ·   **Unblocks:** [m4-05](sprint-m4-05.md) · [m4-06](sprint-m4-06.md) (AB16★ UI)
> **Release action:** merge only (ships in **v1.16.0**, tagged by [m4-07](sprint-m4-07.md)). No infra PR: no new subject, stream, durable or in-cluster caller.
> **Calendar:** December 2026
> **Execute with:** [`../prompts/prompt-m4-04.md`](../prompts/prompt-m4-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | practice: the deferred provisional columns + the lock → provisional rules (AI suggestion, honor-key could-gain; mocks excluded) | X | ⬜ |
| 2 | Accept / edit (bounded by ceilings) + 24 h auto-accept in the ticker (none when flagged) | X | ⬜ |
| 3 | Honor-key claims on touches ("[I meant this]", once per attempt) | X | ⬜ |
| 4 | Dispute → blind re-grade: gateway → practice authorize → judge `POST /evaluations/{id}/regrade` | X | ⬜ |
| 5 | Fallbacks: budget / breaker / kill switch / no consent → `self_grade_pending` or attestation; never blocks | X | ⬜ |
| 6 | Gateway routes, poll payload (camelCase), `gradesWaiting` in the dashboard agg, openapi | X | ⬜ |
| 7 | Event values, replay goldens, lifecycle tests, e2e | X | ⬜ |
| 8 | Docs + ADR | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M4 milestone row + decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m4-03](sprint-m4-03.md) merged (judge/review migrations and gateway route files are serialized behind it)
- [ ] practice attempt states from M3 live ([m3-08](sprint-m3-08.md), v1.14.0): `provisional` and `self_grade_pending` in the
      `attempt.state` CHECK (declared in [m2-01](sprint-m2-01.md)), the `attempt_evaluation` fact table, the closed strategies
      (`verdict_timer@1`, `weighted_gate@1`, `self@1`), the guarded `Conclude()`, the 30 s ticker, `retry_used`, and the v1
      outcome route accepted in `self_grade_pending` (INV-9)
- [ ] [m4-02](sprint-m4-02.md)'s `ai_rubric` grader is registered on the llm lane and covered by its fixture-item tests, and
      `llm_calibration` rows gate which configurations may run (the re-grade builds on both). The end-to-end proof is this
      sprint's own e2e ([task 7](#7--events-replay-tests-x)), not a gate
- [ ] AB16★ frozen ([ds-m4-01](sprint-ds-m4-01.md)): `design-system/screens/v2/AB16-ai-suggestion-dispute.html` — its frames
      (provisional card, dispute modal, re-grade compare, override confirm, honor claim) define the endpoint contract
- [ ] Parallel sessions: no peer PR open on practice or judge migrations, or on `internal/gateway` route tables

## Goal

Make every non-authoritative grade a **suggestion the learner finishes** (D14). A lock that an AI step contributed to,
or a touch that only fails because an **honor-grade public key** missed, goes to `provisional` instead of concluding:
the learner **accepts**, **edits** (bounded by the deterministic ceilings, labelled `override`, `trust=honor`), asks for
**one blind re-grade** (AI) or makes **one "[I meant this]" claim** (honor keys); an untouched suggestion **auto-accepts
after 24 h** unless it was flagged. Submitting triggers the downstream actions — the single conclusion, mistake entry and
revision schedule — anchored at the lock time, so Day 1 never slips. When AI is off, over budget or not consented,
practice falls back to `self_grade_pending` (manual entry) or attestation, and **the loop never blocks**. Today shows
"N grades waiting".

**What is live on prod in v2.0.** No v2.0 course carries an `ai_rubric` step (system design is v2.2+), and the
independent re-grade configuration is calibrated with the SD course ([t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream)).
So on prod the **AI-provisional and re-grade paths are dormant**, proven in the e2e against a fixture rubric item; the
live path is the **DSA touch honor-key claim** (pattern and complexity probes, [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course)).

## Scope

**In**
- practice: provisional columns and rules, accept/edit, auto-accept, claims, dispute authorization, re-grade facts,
  `self_grade_pending` for AI/budget/consent failures, `GET /attempts/waiting`
  ([t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers),
  [§3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) as overridden by
  D14; [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer)).
- judge: `POST /evaluations/{id}/regrade` (learner route, aud=judge) with a synchronous pre-check, the `evaluation_dispute`
  table and `judge admin disputes export` (both moved here from m4-02), the `regrade` job calling m4-02's
  `Score(purpose = score_regrade)` on the independent configuration, never reading the appeal text
  ([t5 §3 spending paths](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets),
  [§7.5](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)).
- gateway: accept, claim and dispute routes; the M4 fields on the submission poll and touch view; `gradesWaiting` in
  `GET /api/dashboard`; the `{"honor_claims": true}` cohort bit on touch start.
- The `resolution` values `accepted | regraded | override | claimed | auto_accepted` on `problem_solved` v2 and
  `touch_concluded`.

**Out**
- All UI — the AB16★ card, dispute modal, compare, override confirm, claim card, the Today "grades waiting" render
  → [m4-06](sprint-m4-06.md).
- Consent write path and the allowance → [m4-05](sprint-m4-05.md) (this sprint treats "no consent" like "AI unavailable").
- Mock AI proposals and `ScoreMock` → M6a ([m6a-03](sprint-m6a-03.md)); **mocks never enter this path** (T6 amendment).
- Calibrating a real rubric and the independent re-grade configuration → the SD-course milestone (v2.2+).
- Per-problem time budgets (PRD Q7) — future research, not a gate.

## Tasks

### 1 · practice provisional columns + rules [X]

A new goose migration in `internal/practice/store/migrations/` (next free version; expand only). [m3-08](sprint-m3-08.md)
already added `ceiling`, `grade_locked_at`, `retry_used` and the fact table `attempt_evaluation` with `regrade_of` (unique
`(context_id, seq) WHERE regrade_of IS NULL`), and **deferred** `candidate_grade`, `settle_deadline`, `dispute_used` and
`claim_used` to this sprint ([t4 §3.1](../research/t4-judge-contract.md#3-attempt-and-conclusion-lifecycle)). Add:

| Column (`attempt`) | Meaning |
|---|---|
| `candidate_grade text` | what an accept concludes with (the AI suggestion, or the re-grade's) |
| `settle_deadline timestamptz` | the 24 h auto-accept deadline, exposed by practice as **`accept_deadline_at`** (AB16's field name; the BFF serves it as `acceptDeadlineAt`, task 6); **NULL = no auto-accept** (flagged, low confidence, dispute pending) |
| `provisional_reason text` | closed Go enum `ai \| honor_key` (no CHECK) |
| `suggestion jsonb` | enums and refs only: `{evaluation_id, grade, category, concept_keys[], feedback_ref, review_flag, low_confidence, regrade_missing}` — no prose (the BFF composes prose from judge's DTO) |
| `dispute_used boolean NOT NULL DEFAULT false`, `dispute_authorized_at timestamptz`, `dispute_evaluation_id uuid`, `regrade_evaluation_id uuid` | one dispute per attempt; when it was authorized (the backstop reads it); which evaluation it targeted; its re-grade |
| `claim_used boolean NOT NULL DEFAULT false` | one claim per attempt |

judge's `evaluation.regrade_of uuid UNIQUE` already exists ([m3-05](sprint-m3-05.md)).

**Lock → provisional** (in the strategy-apply step of the consumer and in `concludeTouch()`; one guarded path):

| Situation | Rule | `provisional_reason` |
|---|---|---|
| Course or touch lock where an **`ai_rubric` step contributed** | **provisional** — D14 makes every AI-graded result a suggestion (this overrides t4 §3.5's "only when the learner could gain" for AI) | `ai` |
| … but the grade is fixed by a **failed deterministic gate** (I6: AI never lifts a failed gate) | **final** at once — the grade isn't editable (D14); mistake and notes stay editable in the journal | — |
| Touch that **fails only because honor-grade public keys missed** (`key_source: public:*`; key mismatch, locked within its timer) and flipping those criteria would pass it | **provisional** (t4 §3.5 could-gain rule), **cohort only until GA** (below) | `honor_key` |
| Honor-key miss where the touch would still fail (e.g. the checked re-solve failed, or the pattern was locked after 120 s) | **final** fail | — |
| `context_kind = mock` or arena | **never provisional** (mocks keep `ScoreMock`; excluded from the 24 h auto-submit) | — |

On entering `provisional`: `grade_locked_at`, `candidate_grade` (AI: the evaluation's grade under the strategy; honor-key
touch: fail), `suggestion`, `settle_deadline = now() + 24h` — **or NULL when the evaluation has `review_flag` or
`low_confidence`** ([t5 §7.5](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)).
`anchor_at` = the lock time and **never changes** afterwards. No event is written until conclusion. `sqlc generate`.

**Cohort gate for the honor-key hold.** The hold is a learner-visible change that doesn't depend on
`LLM_PLATFORM_ENABLED`, so it rides the T-3 cohort, following [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)'s
dark-launch path and [rollout §4 M4](../rollout-plan.md#4-per-milestone-detail)'s gating ("owner (and tester) cohort
until the GA flip"). It adds **no new flag**:
- The gateway's touch-start proxy (`POST /api/touches/{itemId}/start` → practice `POST /touches/{revision_item_id}/start`,
  [m2-04](sprint-m2-04.md)) adds `{"honor_claims": true}` only when `!judgeCohortOnly || inCohort(info)`. Only the gateway
  sets it, like m3-09's `{"judge": true}`.
- `judgeCohortOnly` is m3-09's constant, which [ga-01](sprint-ga-01.md) deletes, so the hold opens to every account at GA.
- practice snapshots the bit into the touch's `stage_params` at start. m2-01 already snapshots there, so there is no
  migration, and a resumed touch keeps its snapshot.
- `concludeTouch()` takes the `honor_key` row only when `stage_params.honor_claims` is true. Otherwise M2's rule holds: a
  final fail, with `claim_allowed=false`.
- Test: a non-cohort touch with an honor-key-only miss concludes at once as a final fail.

AI-provisional rows need no extra gate: an `ai_rubric` step runs only through `Reserve`, which already applies the flag
and the cohort.

### 2 · Accept / edit + auto-accept [X]

- practice `POST /attempts/{id}/accept {grade?, category?}` (JWT, owner only; idempotent through the guarded `Conclude()`):
  - no `grade`, or `grade = candidate_grade` → conclude with the evaluation's `graded_by` (`ai`) and `trust`;
    `resolution = regraded` if a re-grade exists, else `accepted`;
  - 409 `not_provisional` when the attempt isn't provisional;
  - `grade ≠ candidate_grade` (**edit**, D14) → must lie in `[miss, ceiling]` (else 422 `above_ceiling`); concludes
    `graded_by=override`, `trust=honor`, `resolution=override` (excluded from judge-checked %, P7);
  - the attempt's grade is fixed by a deterministic gate → 409 `grade_not_editable`;
  - `category` (optional, a manifest mistake category) → `problem_solved.mistake_hint = {category, strength:"learner",
    signal:"suggestion"}` so review's precedence keeps the learner's pick. **Notes and concepts** edited in AB16 F5 are
    saved through review's mistakes API **after** the conclusion opens the entry (a learner `PATCH` locks the field,
    [m3-10](sprint-m3-10.md)); m4-06 sequences the two calls, so no prose ever rides an event;
  - honor-key touches accept only `{}` (accepting the fail); claims are task 3.
- **Ticker** (m2-01/m3-08's 30 s ticker, `FOR UPDATE SKIP LOCKED`): `state='provisional' AND settle_deadline < now()` →
  conclude with `candidate_grade`, **`resolution=auto_accepted`** (for an honor-key touch: the fail, dated at `anchor_at`).
  Not `timeout`: m3-08 already uses `timeout` for the D15 deadline Miss, and one value must not mean two things.
  Flagged suggestions (`settle_deadline IS NULL`) are never auto-accepted; they stay in "grades waiting".
- The 24 h window is a constant; `PRACTICE_SETTLE_WINDOW` may shorten it **only when `DEV_AUTH` is set** (the e2e and
  m4-06's compose run reach AB16 F13's auto-accept in seconds); production never sets it.
- A `-race` test: an accept racing the ticker's auto-accept → exactly one conclusion, one event.

### 3 · Honor-key claims [X]

- practice `POST /attempts/{id}/claim {probe_ids[]}`: `purpose=touch`, `state=provisional`, `provisional_reason=honor_key`,
  `claim_used = false` (else 409 `claim_used`); every id must be a missed honor-key probe of this attempt (else 422
  `not_claimable`). **One claim action per attempt, naming every missed honor-key probe the learner means** (a single action keeps "once" meaningful and
  restores v1's self-affirmation, `Revision.tsx`'s old behaviour; checked pack keys stay strict).
- The named criteria become met with `source=claim`; `claim_used` is set; `concludeTouch()` concludes — a pass when
  every required criterion is now met — with `resolution=claimed`, `trust=honor`, `graded_by` unchanged (the CHECK has no
  `claimed` value, and none is needed), `anchor_at` unchanged.
- The touch view after lock already shows the accepted canonical answer for unmet probes ([m2-01](sprint-m2-01.md));
  confirm it and add `claim_allowed`.
- Claims are countable per probe from `attempt.criteria` (`source=claim`) so the owner can grow the alias tables
  ([t4 §14](../research/t4-judge-contract.md#14-risks)); no new table.

### 4 · Dispute → blind re-grade [X]

The spending path is authenticated and ordered by the gateway ([t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)
amends [t4 §2.4 step 8](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers)):

1. SPA → gateway `POST /api/evaluations/{id}/dispute {attempt_id, criteria[], reason_codes[], text?}` with an
   Idempotency-Key. `text` ≤ 500 chars (else 422); `reason_codes` a closed enum matching AB16 F6 (`missed_evidence`,
   `misread_answer`, `unfair_to_approach`, `other`).
2. gateway → practice `POST /attempts/{attempt_id}/dispute {evaluation_id}` (JWT): owner; `state=provisional` (else 409
   `not_provisional`); `provisional_reason=ai` and the evaluation is the attempt's deciding AI evaluation (else 409
   `not_disputable`); `dispute_used = false` (else 409 `dispute_used`) → set `dispute_used`, `dispute_authorized_at`,
   `dispute_evaluation_id`, and pause the window (`settle_deadline = NULL`). **Replay** (same evaluation, already authorized,
   no re-grade fact yet) → 200, so a gateway retry heals. `DELETE /attempts/{attempt_id}/dispute` **releases** an
   authorization that has no re-grade fact yet (`dispute_used = false`, a fresh 24 h window) — the gateway calls it when
   judge refuses synchronously (step 3).
3. gateway → judge `POST /evaluations/{id}/regrade {criteria[], reason_codes[], text}` with the learner's aud=judge JWT
   ([m3-09](sprint-m3-09.md)'s minting). judge: owner by `sub`; a counted course/touch evaluation with a contributing
   `ai_rubric` step; not itself a re-grade. **Synchronous pre-check:** kill switch off, breaker open, m4-02's catalog finds no
   passed `llm_calibration` row with **`task = score_regrade`** for this `rubric@v` (`ErrUnavailable{uncalibrated}`), or the
   admission check would refuse → **503 `ai_unavailable`** (or `budget_exhausted` with `resets_at`) without writing anything; the
   gateway then releases practice's authorization and returns the 503, so the SPA offers AB16 F11/F12 and the dispute is
   not spent. Otherwise it inserts a `judge.evaluation_dispute(id, evaluation_id UNIQUE, account_id, criteria, reason_codes,
   text, created_at)` row — **this sprint creates the table** (m4-02 moved it here; a body table, PK/FK only) — and a
   `regrade` job (P0, llm lane); the result is guarded by **`UNIQUE(evaluation.regrade_of)`** (m3-05); a second request
   returns the existing one. Exempt from the L9–L11 quotas ([t4 §2.3](../research/t4-judge-contract.md#23-admission) row 6),
   **not** from `Reserve` (tier `final`).
4. The job calls m4-02's `Score` with **`purpose = score_regrade`**: the **independent configuration** (a second `prompt@v`
   with reordered anchors and different exemplars, or Opus 5.5 `medium`) at **k = 5**. It is resolved only through a passed
   `llm_calibration` row with `task = score_regrade` for the item's `rubric@v`, never through the primary `task = score`
   row. **The independent configuration is its own row**: m4-02's `llm_calibration.task` enum (`analyze|score|feedback`)
   gains `score_regrade`, additively, in this sprint's judge migration (the one that creates `evaluation_dispute`). If
   m4-02 put a CHECK on `task`, the migration replaces it with the wider one: a superset, so it is expand-safe
   ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
   m4-02's catalog lookup and `judge admin calibration record --from` accept the new value, and `llm_call.purpose`
   already has it. In v2.0 no such row exists, so step 3's pre-check answers 503 and no job is created (the rubric
   course brings it). A job whose configuration or budget disappears between the pre-check and the claim ends
   `inconclusive(ai_unavailable | budget_exhausted)` without a call. The dispute `text`, `criteria` and `reason_codes`
   **never reach the scorer**: `ScoreRequest` has no field for them (m4-02), plus a canary-style test that plants a unique
   string in the dispute text and asserts it appears in no request, log line, ledger row or event.
5. The result is a new evaluation with `regrade_of` set, **reusing the original's `attempt_seq`**, emitted as
   `evaluation_completed` through the outbox. practice's consumer stores the fact (its gap check ignores `regrade_of`
   rows), re-applies the strategy with the re-grade in place of the original, sets `candidate_grade`,
   `regrade_evaluation_id`, and a **new** `settle_deadline = now() + 24h` (NULL if the re-grade is flagged). An
   `inconclusive` re-grade leaves the candidate unchanged, sets `suggestion.regrade_missing`, opens the new window, and the
   learner may still edit.
6. **Backstop:** the ticker gives a dispute authorized > 30 min ago with no re-grade fact a fresh 24 h window and
   `suggestion.regrade_missing = true` (the UI offers accept or edit).
- judge's erase consumer ([m3-05](sprint-m3-05.md)) deletes `evaluation_dispute` rows (test).
- **`judge admin disputes export [--since 30d]`** (moved here from m4-02 with its table; m3-14's dispatcher, `kubectl exec`,
  an audit row): evaluation id, item, criteria, reason codes, the re-grade's direction and the dispute text — the one place
  the text is read, by the owner as course author, for calibration (D34's on-demand read; never a digest).
- **Ordering is self-protecting:** a re-grade can only be requested through the new practice authorization route, so
  judge cannot produce re-grade facts for a practice that doesn't understand them — both ship in v1.16.0.

### 5 · Fallbacks: the loop never blocks [X]

| Failure | Course attempt | Touch |
|---|---|---|
| A required `ai_rubric` step ends `inconclusive(budget_exhausted \| ai_unavailable)` — breaker open, `LLM_PLATFORM_ENABLED=false`, provider down, no passed calibration, no consent (m4-02's gate maps it to `ai_unavailable`) | `self_grade_pending` (existing M3 state): the evaluation DTO carries `resets_at` for `budget_exhausted` (from `ErrBudgetExhausted{scope, resets_at}`); the SPA offers **"Wait for xLearn AI (resets <date>)"** (stay pending; **Retry grading** once, `retry_used`) or **"Grade it myself"** (the picker ≤ ceiling through the v1 outcome route, INV-9). Never auto-concluded. | the AI criterion opens for **attestation** (v1 self-scoring: `graded_by=self`, `trust=honor`, [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used)); runner-only `inconclusive` keeps D15's void rule |
| Honor-key probes | not applicable | claims need no AI and work with the flag off |

- `GET /attempts/waiting` (practice, JWT) → `[{attempt_id, problem_id, path_slug, purpose, state, provisional_reason,
  candidate_grade, accept_deadline_at|null}]` for `provisional` and `self_grade_pending` attempts: the source of "N grades
  waiting" (AB16 F14: item, grade, countdown).
- A test runs the whole course and touch loop with AI off (flag false, breaker open, budget exhausted) and asserts every
  attempt reaches a conclusion through a learner action and no path waits on judge.

### 6 · Gateway [X]

`internal/gateway/grades.go` (new), routes in the `apiRoutes()` table and `docs/architecture/openapi.yaml`:

| Route | Upstream |
|---|---|
| `POST /api/attempts/{id}/accept` | practice `POST /attempts/{id}/accept` |
| `POST /api/attempts/{id}/claim` | practice `POST /attempts/{id}/claim` |
| `POST /api/evaluations/{id}/dispute` | practice authorize → judge `POST /evaluations/{id}/regrade` (task 4); 202, or judge's 503 `ai_unavailable` / `budget_exhausted{resets_at}` after releasing the authorization (`DELETE /attempts/{id}/dispute`) |
| `GET /api/submissions/{id}` (existing poll, [m3-09](sprint-m3-09.md)) and the touch view `GET /api/touches/{attemptId}` ([m2-04](sprint-m2-04.md)) | add `{candidateGrade, provisionalReason, acceptDeadlineAt, serverNow, disputeAllowed, claimAllowed, flagged, regradeMissing}`, mapped from practice's snake_case fields (`candidate_grade`, `accept_deadline_at`, …) on m3-08's `GET /state/{problemId}` and m2-01's `GET /touches/{attempt_id}`, extended here ([t4 §2.7](../research/t4-judge-contract.md#27-transport)); **server-provided deadline only** |
| `GET /api/dashboard` (agg) | adds `gradesWaiting: {count, firstDeadlineAt, items[≤5]}` from `GET /attempts/waiting`, item titles from curriculum as the agg already does |

- **Casing:** both composed responses are camelCase, like the rest of m3-09's poll (`pollAfterMs`, `factsThroughSeq`,
  `gradingMode`), m2-04's touch view (`mockMode`, `lockedAtSecs`) and the dashboard (`revisionsDue`). practice's upstream
  JSON stays snake_case, and the BFF maps it. So AB16's `accept_deadline_at` reaches the SPA as `acceptDeadlineAt`,
  `server_now` as `serverNow`, and `grades_waiting` as `gradesWaiting`. `openapi.yaml` records the camelCase names, and a
  gateway test asserts that no snake_case key appears in either response.
- Presence by config: the dispute route 404s unless `JUDGE_BASE_URL` is set and the account is in the cohort; accept and
  claim follow the attempt (practice refuses non-provisional attempts).
- Mutating routes go through M1b's JSON + `Sec-Fetch-Site` checks, `httpx.ReadBody` (typed 413) and the L5 bucket.
- Cache epoch: invalidate the agg on the gateway's own accept, claim and dispute calls; ticker-driven auto-accepts show
  after the agg's TTL (accepted; single replica).
- The DTO denylist test covers the new responses (no `aliases`, `key`, `anchors` or dispute text).

### 7 · Events, replay, tests [X]

- `problem_solved` v2 and `touch_concluded`: `resolution` gains `accepted | regraded | override | claimed |
  auto_accepted` (closed Go enum, no CHECK — [m2-01](sprint-m2-01.md)'s decision; `timeout` stays m3-08's D15 deadline
  Miss). `graded_by` stays within the existing CHECK (`self, auto, ai, override`). **No new subject, stream or durable** → no ACL PR; no new in-cluster caller → no
  NetworkPolicy PR.
- Consumers need no change: review schedules on pass/fail from `anchor_at`; assessment's provenance treats `override` and
  claims as `trust=honor`, so judge-checked % (`auto ∧ checked`) excludes them. Add **replay golden rows** (ADR-0018) for
  accepted, regraded, override, claimed and auto-accepted conclusions.
- **Lifecycle goldens** (practice): AI lock → provisional → accept; → 24 h auto-accept; → edit within / above the ceiling;
  → dispute → re-grade (up, down, inconclusive) → accept / edit; judge 503 pre-check → authorization released, dispute not
  spent; second dispute → 409 `dispute_used`; flagged → no auto-accept; deterministic gate failed → final + 409 on edit;
  honor-key touch → claim → pass; → accept → fail; → auto-accept → fail at `anchor_at`; second claim → 409; honor miss +
  failed re-solve → final fail; mock context → never provisional; the dispute backstop.
- **Property tests:** `passed ⇔ grade ≠ miss`; `anchor_at` unchanged across every window; an edit never exceeds the
  ceiling; exactly one conclusion per attempt.
- **e2e** (`internal/e2e`, `-tags e2e`; judge on an `httptest` fake provider — [m4-01](sprint-m4-01.md)'s convention, never a
  committed compose service): the synthetic rubric fixture item — one `text` part, one `ai_rubric` step — under
  `internal/judge/testdata/pack` (reuse m4-02's if it's there, else add it as test data only; [m4-06](sprint-m4-06.md) reuses
  it), seeded passed `llm_calibration` rows for its rubric with `task = score` (primary) and `task = score_regrade` (the
  independent configuration), and the AI consent rows. Cases:
  final close → provisional → dispute → re-grade → accept → **exactly one** `problem_solved` (`resolution=regraded`); the
  same without the `task = score_regrade` row → 503 and the dispute is not spent; `PRACTICE_SETTLE_WINDOW` shortened → auto-accept
  (`resolution=auto_accepted`); a DSA touch with a pattern alias miss → claim → pass (`resolution=claimed`); the flag off →
  the fixture item lands in `self_grade_pending` → picker → concluded.

### 8 · Docs + ADR [X]

- `docs/architecture/events.md` (the `resolution` values; "Flow — provisional, dispute and claim"), `api.md` (the routes),
  `data-model.md` (practice columns; `judge.evaluation_dispute`), `services.md` (gateway → practice authorize → judge
  re-grade; the ticker's settle job).
- **ADR** "Provisional grades, disputes and claims (M4 realization)": D14 applied to every AI-contributing lock; the
  honor-key could-gain rule, cohort-only until GA through `judgeCohortOnly`; one claim naming every missed honor probe;
  the independent configuration as its own `task = score_regrade` calibration row; flagged suggestions never auto-accept; re-grade
  facts reuse the seq; the synchronous 503 pre-check that releases the authorization (no independent configuration in
  v2.0); `auto_accepted` vs m3-08's `timeout`; the touch AI fallback is attestation. Take the **next free ADR number**
  after checking peers.

## Acceptance criteria

- [ ] **Auto-accept at 24 h** (untouched, unflagged; `resolution=auto_accepted`), anchored at the lock; **no auto-accept
      when flagged or low-confidence**.
- [ ] **One re-grade max** per attempt (`dispute_used` + `UNIQUE(regrade_of)`); the re-grade never sees the appeal text
      (canary test); with no passed independent configuration judge answers 503 `ai_unavailable`, the authorization is
      released (the dispute isn't spent) and the learner can accept or edit.
- [ ] Edits are bounded by the ceiling and **labelled `override`, `trust=honor`**; deterministic verdicts aren't editable.
- [ ] One honor claim per touch attempt flips only missed honor-key criteria (`resolution=claimed`, `trust=honor`); the
      honor-key hold is cohort-only until GA (a non-cohort honor-key miss is a final fail).
- [ ] **The loop never blocks when AI is off**: flag off, breaker open, budget exhausted or no consent → `self_grade_pending`
      (course) or attestation (touch), concluded by a learner action; claims still work.
- [ ] Mocks never enter the provisional path; exactly one conclusion per attempt (`-race`, property tests).
- [ ] `gradesWaiting` appears in `GET /api/dashboard`, and the poll and touch view carry the M4 fields in camelCase; the new
      routes are in `openapi.yaml` (drift test green).
- [ ] Replay goldens equal for the new resolutions; e2e green.

## Release

**Merge only — ships in v1.16.0**, tagged by [m4-07](sprint-m4-07.md). No infra PR. Two behaviour changes reach the
**cohort** (owner and testers) in v1.16.0 and belong in its release notes. First, a DSA touch that fails **only** on an
honor-key miss is now held `provisional` (claim or accept, 24 h auto-accept) instead of failing at once. Day 1 is still
dated from the lock, and non-cohort accounts keep M2's immediate fail until [ga-01](sprint-ga-01.md) deletes
`judgeCohortOnly`. Second, AI-grading failures surface "Wait for xLearn AI / Grade it myself". The AI-provisional and re-grade paths stay dormant on prod
(no `ai_rubric` item in v2.0).

## Definition of Done

CI green (`go build`, `go vet`, `go test -race ./...`, `sqlc diff`, the migration lint, the subject-registry test, the
openapi drift and DTO denylist tests, web tests untouched) · e2e green · merged to `main` via PR (squash) ·
statuses updated (this file + [`../status.md`](../status.md)) · the ADR written · local `main` synced.

## Risks / watch-outs

- **Mocks are excluded from the 24 h auto-submit** (T6 amendment): keep the provisional path `course | touch` only; a test
  asserts a mock context never enters it.
- **An e2e-green re-grade is not a prod path in v2.0** — no `ai_rubric` item and no independent calibration exist until
  the SD course. Don't claim otherwise in the release notes.
- **Honor-key holds change the DSA touch feel** (a 24 h provisional window where M2/M3 failed at once): cohort-only until
  GA through m3-09's `judgeCohortOnly`. The ladder is unaffected because `anchor_at` is the lock time; the release notes
  say so.
- **Accept vs auto-accept vs claim races** — one guarded `Conclude()`, `FOR UPDATE SKIP LOCKED`, the `-race` test.
- **Gateway crash between authorize and re-grade** — idempotent authorize + judge `UNIQUE(regrade_of)` heal a retry; the
  30-minute ticker backstop covers an abandoned retry.
- **Flagged suggestions can wait indefinitely** — by design (no auto-accept); Today's "grades waiting" surfaces them, and
  `self_grade_pending` likewise never auto-concludes (t4 §3.3).
- **Dispute text is an injection channel** — stored for calibration only, never in any request (test), erased with the
  account, exported only via `judge admin disputes export`.
- **Overrides inflating rigour** — `trust=honor` everywhere they appear; judge-checked % excludes them.
- **Countdown drift** — the SPA renders only the server's `acceptDeadlineAt` (m4-06).
