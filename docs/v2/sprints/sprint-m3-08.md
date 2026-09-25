# Sprint m3-08 — practice: judge consumer, reconciler, D15/D16/D18 grading strategy

> **Milestone:** M3 — judge plus code grader (**M3-2**, Run/Submit) · **Track:** product · **Order:** 48
> **Prereqs:** [m3-07](sprint-m3-07.md) (`v1.13.0` live, judge dark) · through it [m3-06](sprint-m3-06.md) (internal `evaluations`/`watermark` endpoints) and [m3-05](sprint-m3-05.md) (`/internal/evaluable`) · on `main`: [m2-01](sprint-m2-01.md) (M2a attempt engine + ticker), [m1-07](sprint-m1-07.md) (`coach_assist_at`), [l-01](sprint-l-01.md) (practice erase consumer, `event_dead_letter`), [m3-01](sprint-m3-01.md) (`contract_hash`), [m1-01](sprint-m1-01.md) (manifest D18 params), [m2-05](sprint-m2-05.md) (`problem_solved` v2 `anchor_at`/`revisable`)
> **Unblocks:** [m3-10](sprint-m3-10.md) (review + assessment on the new event fields) · [m4-04](sprint-m4-04.md) (provisional grades build on these attempt states) · [m3-09](sprint-m3-09.md) (its entry gate: the gateway BFF builds against task 7's internal contract)
> **Release action:** **merge only (ships in `v1.14.0`)**, tagged by [m3-13](sprint-m3-13.md) — plus **one infra PR** (practice's `XLEARN_JUDGE` durable in the NATS ACL) merged **in this sprint**, before `v1.14.0`.
> **Calendar:** mid-November (in parallel with [m3-09](sprint-m3-09.md))
> **Execute with:** [`../prompts/prompt-m3-08.md`](../prompts/prompt-m3-08.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Schema: attempt M3 columns, `attempt_evaluation` facts, `inbox`, `arena_revealed_at` | X | ⬜ |
| 2 | Grading strategies `verdict_timer@1` (D18), `weighted_gate@1`, `self@1` + invariants | X | ⬜ |
| 3 | Judged start, hint gate, sync close + give-up, INV-9, arena-reveal record | X | ⬜ |
| 4 | `evaluation_completed` consumer + gap-free conclusion | X | ⬜ |
| 5 | Ticker: gap reconciler, touch + course watermark drain, attempt deadline Miss, kill-switch sweep | X | ⬜ |
| 6 | Events: `problem_solved` v2, `touch_concluded`, `attempt_logged` | X | ⬜ |
| 7 | Gating (`JUDGE_BASE_URL`, cohort, grading override) + the internal contract for m3-09 | X | ⬜ |
| 8 | NATS ACL PR: practice durable on `XLEARN_JUDGE` (+ egress check) | I | ⬜ |
| 9 | Tests + verify: lifecycle goldens, reconciler heal, drain, self path = v1 | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row; M3 milestone stays 🔄; flag inventory).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **`v1.13.0` live** ([m3-07](sprint-m3-07.md)): judge Ready dark, `XLEARN_JUDGE` exists. judge produces nothing while dark (the gateway has no `JUDGE_BASE_URL`), so this consumer ships one tag before any event can be produced
- [ ] On `main`: m3-06's `GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T` and its watermark endpoint — **preferred:** `GET /internal/contexts/{kind}/{id}/watermark` for `course` and `touch` with `judgeapi.Client.Watermark(kind, id)` (the generalization belongs in m3-06 task 8); **else** the touch-only `…/touch/{id}/watermark`, and task 5 generalizes it here — plus m3-05's `GET /internal/evaluable?path=` and the compose fixture pack ([m3-02](sprint-m3-02.md))
- [ ] On `main`: the M2a engine ([m2-01](sprint-m2-01.md): `attempt.{purpose,state,grade,passed,graded_by,trust,criteria,policy_version,stage_params,probes_shown_at,anchor_at,resolution}`, guarded `Conclude()`, `internal/practice/ticker.go`); `attempt.coach_assist_at` ([m1-07](sprint-m1-07.md)); practice's erase consumer, `erased_account` and `event_dead_letter` ([l-01](sprint-l-01.md)); `internal/course/canon` ([m3-01](sprint-m3-01.md)); the D18 values under the manifest's `grading.params["verdict_timer@1"]` ([m1-01](sprint-m1-01.md))
- [ ] **Migration numbering:** no open peer PR adds a practice migration (`gh pr list`, `git worktree list`, ListAgents); take the next free goose version at rebase
- [ ] **Parallel with [m3-09](sprint-m3-09.md):** the internal practice contract in task 7 is agreed (both sprints depend only on m3-07; pin it in `docs/architecture/api.md` first if both run at once)
- [ ] Not a gate here: the M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) — it gates [m3-11](sprint-m3-11.md) and [m3-13](sprint-m3-13.md)

## Goal

Make **practice the single writer of learning signals from judge evidence** ([ADR-0026](../../adr/0026-per-course-extensibility-model.md):
judge produces evidence, practice decides; [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md)):
- practice's first judge consumer (`evaluation_completed`, always ack) with an inbox, the **gap-free `attempt_seq`**
  rule, a **pull reconciler** and the extended ticker;
- the **synchronous close and give-up**;
- the **D15/D16/D18 method** for judged DSA attempts: a **45:00** hard limit, the **hint unlocks at 15:00** and caps
  Assisted, coach help caps Assisted, **a pass finishes the attempt** (no re-implement), **reveal = give-up = Miss**, and
  **timeout = Miss dated at the deadline**;
- while the **self path stays exactly as it is today** (golden = v1).

## Scope

**In**
- practice `inbox`; a durable on `XLEARN_JUDGE` filtered to `xlearn.judge.evaluation_completed`; the fact table; the
  gap-free check; the pull reconciler ([t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)).
- Attempt M3 columns (`within_timer`, `failed_submits`, `counted_submits`, `first_submit_passed`, `language`,
  `contract_hash`, plus the close/lock/fact bookkeeping) and `user_problem_state.arena_revealed_at` ([t4 §11.4](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag) #9).
- The judged start (pins), the hint gate, the sync close and give-up, INV-9, the arena-reveal record (D17: recorded, never caps).
- The closed strategy set `verdict_timer@1`, `weighted_gate@1`, `self@1`, table-tested against v1 and D18, with the
  universal invariants as property tests.
- `problem_solved` v2 + `touch_concluded` trust fields; `attempt_logged` course-only.
- Cohort + `JUDGE_BASE_URL` gating and the grading override (kill switches).
- The infra ACL PR for practice's new durable.

**Out**
- AI provisional grades, dispute, re-grade, claims, the 24 h settle → M4 ([m4-04](sprint-m4-04.md)).
- **Per-problem time budgets** (PRD Q7) — a future research session, not a gate. The 45:00 / 15:00 / 20:00 defaults come from the manifest.
- Gateway routes, poll composition, late-submit refusal at the gateway, typed 413, `withhold()` for arena → [m3-09](sprint-m3-09.md).
- review/assessment reading the new fields, and the DSA `mistakes.prefill` table → [m3-10](sprint-m3-10.md).
- UI (Workspace, results dock) → [m3-11](sprint-m3-11.md), [m3-12](sprint-m3-12.md).
- Setting `JUDGE_BASE_URL` in production → [m3-13](sprint-m3-13.md) (gateway **and** practice; hand-off in Release).
- The re-implement stage and the `reimplementing` state (dropped by D16); mock evaluations (silent in v2.0).
- practice's `xlearn` egress policy → [mi-11](sprint-mi-11.md) (task 8 only checks it if already live).

## Tasks

### 1 · Schema [X]

`internal/practice/store/migrations/0000N_m3_judge_facts.sql` (next free version; expand only, no `xlearn:contract`
marker; any `CREATE INDEX CONCURRENTLY` in its own `-- +goose NO TRANSACTION` file). Sources: [t1 §4 practice](../research/t1-content-data-model.md),
[t4 §3.1](../research/t4-judge-contract.md#3-attempt-and-conclusion-lifecycle) (as overridden by D14–D18).

`practice.attempt` gains (M2a already has `state` incl. `self_grade_pending`/`voided`, `grade`, `passed`, `graded_by`,
`trust`, `criteria`, `policy_version`, `stage_params`, `probes_shown_at`, `anchor_at`, `resolution`; m1-07 has
`coach_assist_at`):

| Column | Type | Meaning |
|---|---|---|
| `grading_mode` | `text NOT NULL DEFAULT 'self' CHECK (grading_mode IN ('self','judge'))` | pinned at start; `self` = today's path |
| `contract_hash` | `text` | pinned at start (canon over the embedded item) |
| `language` | `text` | language of the deciding (else last counted) submit |
| `within_timer` | `boolean` | the deciding pass was submitted ≤ deadline (2 s grace) |
| `failed_submits` | `int NOT NULL DEFAULT 0` | counted `failed` facts, excluding `free_classes` (CE, REJECTED) |
| `counted_submits` | `int NOT NULL DEFAULT 0` | counted facts (course/touch context, action ≠ run) |
| `first_submit_passed` | `boolean` | seq 1 passed |
| `grade_locked_at` | `timestamptz` | when the strategy locked |
| `close_submission_id`, `close_seq`, `close_action` | `uuid`, `int`, `text` (Go enum `final\|give_up`) | the recorded sync close |
| `gave_up_at` | `timestamptz` | the give-up's `submitted_at` |
| `released_closes` | `int NOT NULL DEFAULT 0` | `final` closes judge released (infra) |
| `retry_used` | `boolean NOT NULL DEFAULT false` | the one "Retry grading" from `self_grade_pending` |
| `facts_through_seq` | `int NOT NULL DEFAULT 0` | the gap-free prefix |
| `facts_gap_since` | `timestamptz` | first time a gap was seen (reconciler input) |
| `ceiling` | `text` | the grade ceiling (the `self_grade_pending` picker bound) |

- **Not added (D16):** `reimplementing`, `reimpl_context_id`. **Deferred to M4 ([m4-04](sprint-m4-04.md)):** `candidate_grade`,
  `settle_deadline`, `dispute_used`, `claim_used`.
- `practice.user_problem_state.arena_revealed_at timestamptz` (D17: recorded as `arena_prior`, **never** caps).
- **Fact table** `practice.attempt_evaluation`: `evaluation_id uuid PK`, `attempt_id uuid NOT NULL REFERENCES practice.attempt(id) ON DELETE CASCADE`,
  `account_id uuid NOT NULL`, `context_kind`, `context_id uuid`, `seq int NOT NULL`, `submission_id uuid`, `action`,
  `closes_context bool`, `released bool`, `status` (CHECK passed|failed|inconclusive|error), `reason`, `class`, `checks jsonb`,
  `score jsonb`, `hidden jsonb`, `perf`, `signals text[]`, `trust`, `provisional_eligible bool`, `language`, `contract_hash`,
  `submitted_at`, `completed_at`, `regrade_of uuid`, `source text CHECK (source IN ('event','pull','invalid'))`, `created_at`;
  unique `(context_id, seq) WHERE regrade_of IS NULL`; index `(account_id, created_at)`. Events carry only enums, numbers
  and refs (INV-8), so no learner code lands here.
- `practice.inbox(event_id text PK, consumed_at timestamptz NOT NULL DEFAULT now())` — review's shape.
  `practice.event_dead_letter` exists since [l-01](sprint-l-01.md) (create it only if it doesn't).
- **Erase:** add `attempt_evaluation` to practice's erase delete list ([l-01](sprint-l-01.md)) and its fixture test.
- Queries: `internal/practice/store/queries/{attempt_evaluation,inbox}.sql` + additions to `attempt.sql`; `sqlc generate`; **`sqlc diff` clean**.

### 2 · Grading strategies [X]

`internal/practice/grading/` — pure functions over facts (no DB, no clock), the closed set from
[t4 §5.4](../research/t4-judge-contract.md#54-grading-strategy-practice) / [§6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds)
**with [t4 §13](../research/t4-judge-contract.md) D15–D18 overriding the body** (the 15 + 10 golden rows in §6.2 are
superseded):

```go
type Strategy interface { ID() string; ValidateParams(json.RawMessage) error; Grade(f Facts, p Params) Result }
type Facts struct {
    Purpose string; StartedAt, Deadline time.Time
    HintAt, CoachAt, SolutionAt, GaveUpAt, ArenaRevealedAt *time.Time // ArenaRevealedAt is recorded, never a cap (D17)
    CloseSeq *int; Evals []EvalFact /* gap-free prefix, seq order */; Probes []ProbeFact
}
type Result struct {
    LockSeq *int; LockedAt *time.Time; Grade string; Passed bool; Score *Score; Criteria []Criterion
    GradedBy, Trust, Resolution, Ceiling string; Signals []string
}
```

**`verdict_timer@1`** (archetype A). Params = the manifest's `grading.params["verdict_timer@1"]`:
`time_limit_s 2700`, `hint_at_s 900`, `clean_within_s 1200`, `max_failed_for_clean 3`, `free_classes [CE, REJECTED]`.
- **Lock trigger:** the earliest-by-seq passing counted submit inside the gap-free prefix; or a give-up (the earliest
  pass below `close_seq`, else Miss at `gave_up_at`, [t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution));
  or the deadline (after the watermark drain, task 5).
- **Grade at the deciding pass** (`t` = its `submitted_at − started_at`; every timer applies to `submitted_at`, INV-6):
  **Assisted** if the hint was revealed or coach help recorded before `t` (D18, D27); else **Clean** if
  `t ≤ 20:00` and failed-before ≤ 3; else **Rough** (`t ≤ 45:00` + 2 s grace); **Miss** if no pass by the limit or a
  give-up with no earlier pass. `inconclusive`/`error` never count as failed (I3).
- **Golden table** (`internal/practice/grading/verdict_timer_golden_test.go`, one row each, all D18):

| # | Facts (time from start) | Grade · resolution · `anchor_at` |
|---|---|---|
| 1 | pass 09:00, 0 failed | clean · immediate · the pass's `submitted_at` |
| 2 | pass 09:00 after WA, WA, TLE | clean |
| 3 | pass 09:00 after WA, WA, TLE, RE | rough |
| 4 | pass 09:00 after 5 × CE + 1 × REJECTED | clean (free classes) |
| 5 | pass 19:59 | clean |
| 6 | submitted 19:59, evaluated 20:30 | clean (`submitted_at` decides) |
| 7 | pass 20:01, 0 failed | rough |
| 8 | hint revealed 15:10, pass 15:40 | assisted |
| 9 | coach assist 05:00, pass 09:00 | assisted |
| 10 | pass 09:00, coach assist 09:30 (after the lock) | clean |
| 11 | 2 × inconclusive, then pass 10:00 | clean |
| 12 | submitted 44:59, evaluated 45:40 | rough (drained before the deadline conclusion) |
| 13 | no pass by 45:00 | miss · timeout · the deadline |
| 14 | pass seq 4 at 14:59 still in flight; give-up seq 5 at 15:00 | clean (earliest pass below `close_seq`) · immediate · 14:59 |
| 15 | give-up seq 5; only failures before | miss · immediate · `gave_up_at` (event `gave_up=true`; `revealed_early`, `solution_revealed_early` outbox row) |
| 16 | `arena_revealed_at` before start; pass 09:00 | clean, `arena_prior=revealed` (D17: no cap) |
| 17 | stale `contract_hash` with counted evals | → `self_grade_pending` |
| 18 | `contract_hash` changed, no counted evals | silent re-pin; the timer continues |
| 19 | hint revealed 16:00, pass 44:59 | assisted |

**`weighted_gate@1`** (composite B, C quizzes — used from P; fixtures only now): lock on the closing `final` or
`give_up`; Miss if a required gate failed, status ≠ passed, or `s < pass_pct`; else band by `s` (`clean_pct 1.0`,
`pass_pct 0.8`) with the same hint/coach/late ceilings.

**`self@1`**: `{cap: none}` = today's self path (v1 `LogOutcome` as amended by m1-07's D27 Assisted cap), `{cap: stage}`
= a pick ≤ `ceiling` in `self_grade_pending`. **Its table is copied from the existing practice store tests and must equal them.**

**Universal invariants** ([t4 §6.1](../research/t4-judge-contract.md#61-universal-invariants-practice-enforced-property-tested),
D17 applied) as property tests over generated fact sequences (random orderings, gaps, classes, hint/coach/give-up times;
stdlib `testing/quick` or a seeded generator): **I1** `passed ⇔ grade ≠ miss`; **I2** ceilings (hint or coach before the
lock → ≤ assisted; solution seen before the deciding evidence → miss; late → ≤ rough; **no arena cap**); **I3**;
**I4** give-up without an earlier pass → miss; **I5** same facts + params → same result; **I7** `graded_by=auto` for
judge strategies; **I8** `revisable = role ∈ {core, reinforcement} ∨ (drill ∧ manifest.revision.drills)`.

**Registry + manifest:** a closed `map[string]Strategy`; boot validates the manifest's `grading.strategy` and params
against it. DSA's `grading.strategy` becomes **`verdict_timer@1`** (the params already sit there, m1-01); update
`internal/course/golden_test.go` in the same PR, citing D18. Non-evaluable items and non-cohort accounts use
`self@1{cap:none}`, the universal fallback (not a manifest field). `stages.reimplement` is read **only** by the self path.

### 3 · Judged start, hint gate, sync close + give-up [X]

**Start** — `POST /problems/{id}/attempt/start` accepts an optional body `{judge: true}` that **only the gateway sets**
(the BFF builds the body after its T-3 cohort check; role is never in the JWT, [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md)).
practice pins `grading_mode=judge` only when **all** hold: `judge: true`; practice's `JUDGE_BASE_URL` is set;
`GRADING_OVERRIDE` ≠ `self`; and judge `GET /internal/evaluable?path=<slug>` reports the item `ok` with
`submit_available` for practice's own `contract_hash` (the shared canon package over the embedded item). The judge call
runs **before** the transaction (5 s timeout); judge down → `self` (WARN log). Then one transaction: the attempt
(`purpose=course`, `state=attempting`, `grading_mode=judge`, `contract_hash`, `policy_version`, `stage_params` = the D18
snapshot) and **one** `timer(kind='attempt', deadline_at = started_at + 2700 s)` (no hint timer). Resume returns the same
attempt until it concludes. Otherwise the v1 start runs unchanged.

**Hint gate** — `POST /problems/{id}/reveal` on a judged attempt: to `hint` only once `now ≥ started_at + hint_at_s`
(else **409** `hint_locked{available_at}`), recorded in `stage_event` (the `HintAt` fact); to `solution` → **409**
`use_give_up` (the gateway turns "Reveal solution" into a give-up close, D15). Self attempts unchanged.

**Close** — new `POST /attempts/{id}/close {action: final|give_up, submission_id, close_seq, submitted_at}`
([t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers) step 4): called by
the gateway **after** judge accepted the closing submission; synchronous, authoritative, **idempotent on
`submission_id`**; `sub` must own the attempt; the attempt must be judged and open (or `self_grade_pending` with
`retry_used = false`, which it then sets — the one "Retry grading").
- **`give_up`:** `gave_up_at = submitted_at`, `close_*` set, the stage moves to `solution` (**the solution unlocks now,
  whatever judge's state**), `revealed_early = true`, and the v1 `solution_revealed_early` outbox row. The lock applies as
  soon as the facts for every `seq < close_seq` are present (earliest pass, else Miss at `gave_up_at`); until then the
  consumer/reconciler completes it. A give-up close is **never released** (INV-13).
- **`final`** (quiz/B items via `weighted_gate@1`): `state=awaiting_evaluation`; the closing fact locks. A **released**
  close (judge sets `released=true` on `inconclusive`/`error`): infra reason and `released_closes < 3` → back to
  `attempting` (`released_closes++`); `contract_changed`, `budget_exhausted` or the 3rd release → `self_grade_pending`.
- **D16:** a pass finishes a judged attempt. There is no re-implement stage on the judged path, whatever
  `stages.reimplement` says; the Day-1 revision is the from-memory re-attempt.

**INV-9** — `POST /problems/{id}/outcome` → **409** `evaluated_item` unless `grading_mode=self` or
`state=self_grade_pending`; in `self_grade_pending` the pick is bounded by `ceiling` (`self@1{cap:stage}`, `graded_by=self`,
`trust=honor`).

**Arena reveal (D17)** — `POST /problems/{id}/arena-reveal` (the gateway calls it when the learner confirms the arena
spoiler, [m3-09](sprint-m3-09.md)): `arena_revealed_at = COALESCE(arena_revealed_at, now())`. It sets
`arena_prior=revealed` on the next course `problem_solved` and **never caps** a grade.

### 4 · `evaluation_completed` consumer [X]

`internal/practice/judgeconsumer.go`: a durable on `XLEARN_JUDGE`, filter `xlearn.judge.evaluation_completed`,
`DeliverAll`, with mi-05's `WithDeadLetter` sink → `practice.event_dead_letter`. Declare it in
`internal/platform/events/topology.go` (mi-05 lists "practice on `XLEARN_JUDGE`" as this sprint's) with its
subject-registry entry; re-render the golden ACL. **Start it in the background** from `cmd/practice/main.go` so
readiness never waits on a stream.

Per message, **one transaction** ([t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)):
1. Claim the inbox row (`ON CONFLICT DO NOTHING` → a duplicate is acked); tombstoned account → ack, store nothing.
2. Resolve `context_id` → the attempt `FOR UPDATE` (course: the attempt id; touch: the touch attempt id). Validate
   account, item, `path_slug`, purpose and `grading_mode=judge`. A mismatch → ack, WARN `forged_or_stale` with ids, no fact.
3. Insert the fact `ON CONFLICT DO NOTHING` (`source=event`); a `contract_hash` ≠ the pin is stored as
   `inconclusive(contract_changed)`.
4. A fact with `closes_context` and no recorded close records the close (the backstop, t4 §2.4 step 5).
5. Attempt already concluded or voided → commit (a late fact).
6. Recompute `facts_through_seq`, `counted_submits`, `failed_submits`, `first_submit_passed`, `language`; run the strategy
   on the gap-free prefix. Lock only if its trigger is at seq T **and** every `seq < T` is present; otherwise
   `facts_gap_since = COALESCE(facts_gap_since, now())` and commit.
7. On a lock: extend m2-01's guarded `Conclude()` (`UPDATE … WHERE concluded_at IS NULL RETURNING` + `UNIQUE(outcome.attempt_id)`)
   with `grade`, `passed`, `graded_by=auto`, `trust`, `within_timer`, `anchor_at` (the deciding fact's `submitted_at`, or
   `gave_up_at`), `resolution=immediate`, `grade_locked_at`, and write the outbox rows (task 6) in the same transaction.

**Always ack** unless the database errors (then nak → the backoff → mi-05's dead-letter row at `maxDeliver`).
**Touch facts** (`context_kind=touch`): for an item with a pack, the band criterion `correct_in_timer` becomes checked
evidence — met iff a passing touch submit has `submitted_at ≤` the touch deadline — and m2-01's `concludeTouch()` runs
when every criterion is decided (`trust=checked` for that criterion). Items without a pack keep m2-01's attestation.
**Relay:** practice reads `RELAY_INTERVAL` (default **1 s**) into `events.WithInterval` (`internal/practice/config.go`); judge's is set in m3-06.

### 5 · Ticker [X]

Extend `internal/practice/ticker.go` (m2-01: every 30 s, `FOR UPDATE SKIP LOCKED LIMIT 50`, one transaction per row).
**Never hold a row lock across an HTTP call:** select candidate ids in a short transaction, call judge, then lock the row,
re-check its state and write. Judge client: **m3-06's `internal/platform/judgeapi.Client`** (`EvaluationsBefore`,
`Watermark`, `Evaluable`; the pulled facts arrive in the exact `evaluation_completed` payload shape, so they go through
the consumer's validation code), wrapped nil-safe with a 5 s timeout for when `JUDGE_BASE_URL` is unset; tests use m3-06's
exported practice-side fake and `internal/platform/events/testdata/evaluation_completed.v2.json`. practice → judge `/internal/*` is fenced by MI-5a and
judge's own ingress; no service token ([ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 10).

| Job | Selects | Does |
|---|---|---|
| **Gap reconciler** | judged attempts with `facts_gap_since < now() − 30 s` | `GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T` (T = max seen seq + 1, or `close_seq`); insert missing facts (`source=pull`, the consumer's validation); a pulled fact that fails validation inserts a **non-passing filler** (`source=invalid`, never counted as failed) and logs ERROR; then steps 6–7 of task 4. An evaluation still running simply isn't returned yet — keep waiting (INV-1) |
| **Touch deadline / End review drain** | judged touches ended, or past deadline + 5 s | `GET /internal/contexts/touch/{id}/watermark → {max_seq, open_jobs}`; once the facts are complete through `max_seq` and `open_jobs = 0`, conclude per m2-01's rules; only infra-inconclusive evidence → `voided` (stays due) |
| **Attempt deadline (D15)** | `grading_mode=judge` course attempts, not concluded, past deadline + 5 s | drain the **course** watermark the same way; then the strategy with the deadline: a pass submitted ≤ deadline (+2 s grace) grades normally, else **Miss dated at the deadline** (`resolution=timeout`, `anchor_at = deadline`) |
| **Kill-switch sweep** | open judged attempts while practice's `JUDGE_BASE_URL` is unset or `GRADING_OVERRIDE=self` | course → `self_grade_pending` (picker ≤ `ceiling`; **never** an automatic Miss); a touch with an undecided checked criterion → `voided` (stays due) |
| **Stale reads (D34: no alert)** | gaps > 10 min, `awaiting_evaluation` > 10 min, `self_grade_pending` > 7 d | one WARN line per attempt per crossing; the three SQL reads go in `docs/v2/runbooks/practice-judge-lifecycle.md` for on-demand use |

- **Course watermark:** the course deadline drain needs `/internal/contexts/{kind}/{id}/watermark` (`kind ∈ course|touch`).
  - **If m3-06 shipped it for both kinds** (preferred): consume `judgeapi.Client.Watermark(kind, id)` and its fixtures;
    this sprint touches no judge code.
  - **If m3-06 shipped the touch route only:** generalize it here — read-only, no migration, the same query keyed by
    context — and treat the session as touching **judge** too (record it on the status board): `internal/judge/handlers.go`
    (route `/internal/contexts/{kind}/{id}/watermark`, unknown `kind` → 404; keep the touch path's response unchanged),
    the watermark query under `internal/judge/store/queries/` only if it filters by kind (then `sqlc generate`),
    `internal/platform/judgeapi` (`Client.Watermark(ctx, kind, id)`, `testdata/watermark.json` gains a course case, the
    exported practice-side fake) and m3-06's judge-side contract test. First check no open peer PR touches those files
    ([m3-09](sprint-m3-09.md) runs in parallel).
- **Why the drain matters for courses:** a lost **last** evaluation leaves no gap for the reconciler to see ([m3-13](sprint-m3-13.md)'s
  outbox-republish rationale). Draining to `max_seq` before the deadline conclusion turns that loss into a visible gap, so
  the pass is pulled and graded instead of the attempt being concluded as a wrong Miss.
- **Judge unreachable:** the deadline and drain jobs **wait** (conclusion is delayed, never guessed). After a kill switch
  the sweep takes over.
- **Self-mode attempts get no deadline Miss** — the self path is v1 until M5 makes DSA evaluator-only.

### 6 · Events [X]

Append-only envelope fields ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md); decoders forever;
[t4 §2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs), §11.4 #12):
- **`problem_solved` v2** adds whatever isn't there yet ([m2-05](sprint-m2-05.md) may have added `anchor_at`, `revisable`;
  [m1-07](sprint-m1-07.md) added `assist{hint, coach}`): `anchor_at`, `revisable`, `arena_prior (none|revealed)`,
  `gave_up`, `reimplemented` (always `false`, D16), `resolution (immediate|timeout|self)` (a `self_grade_pending` pick is
  `self`; M4 adds `accepted|regraded|override|claimed`), `mistake_hint{category, strength, signal}`, `concepts_hint[≤3]`, `language`,
  `counted_submits`, `first_submit_passed`, `trust`, and `graded_by`, `within_timer`, `failed_submits`,
  `evaluation_ids[]` if absent.
- **`mistake_hint`:** judge `signals[]` of the deciding/last facts ∪ practice's attempt signals (`late`, `hint_used`,
  `solution_seen`, `gave_up`, `blank_draft`, `many_failed`), mapped through the manifest's `mistakes.prefill` (first match
  wins; for a rough-from-failures grade, the most frequent failure class) — [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only).
  The DSA table is empty until [m3-10](sprint-m3-10.md) fills it, so the hint is null until then; tests use a fixture table.
- **`concepts_hint[]`:** the failed checks' `concept` refs (≤ 3, validated against `item.concepts`).
- **`touch_concluded`** adds `evaluation_ids[]` (and `trust` if absent), `mistake_hint`, `concepts_hint[]` for judged re-solves.
- **`attempt_logged`** stays **course-only** (never touches), with `path_slug` and `anchor_at`.
- These are existing subjects with additive fields: consumers ignore unknown fields, and m3-10's readers ship in the same
  `v1.14.0`, so no two-tag split is needed. Update [`../../architecture/events.md`](../../architecture/events.md).

### 7 · Gating + the internal contract for m3-09 [X]

**Presence by config** ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md)): the judged path
exists only when practice's **`JUDGE_BASE_URL`** is set (`http://xlearn-judge.xlearn.svc.cluster.local:8087`), the
grading override is off, and the gateway passes `judge: true` at start (T-3: role ∈ {owner, tester}). Items with a pack
show Run/Submit; the others keep the picker (`grading_mode` in the state response).

**Grading override** — ADR-0034 §2 lists "the grading override" as a permanent kill switch; this plan names it
`GRADING_OVERRIDE` (unset | `self`) on practice. If a peer already named it in status.md's flag inventory, use that name.
`self` → new attempts pin `self`, and the sweep moves open judged attempts to `self_grade_pending`. Nothing is re-graded.

**Internal routes m3-09 consumes** (document in [`../../architecture/api.md`](../../architecture/api.md) first):

| Route | Change |
|---|---|
| `POST /problems/{id}/attempt/start` | + optional `{judge: bool}` |
| `GET /state/{problemId}` (and the batched `GET /state?ids=`) | + `grading_mode, stage_params, state, started_at, deadline_at, hint_available_at, hint_opened_at?, coach_used_at?, grade?, graded_by?, trust?, resolution?, failed_submits, facts_through_seq, concluded, ceiling?, mistake_hint?, concepts_hint?, arena_revealed` — exactly the fields [m3-09](sprint-m3-09.md)'s poll composition allowlists (t4 §2.7: the SPA renders a conclusion only once `facts_through_seq ≥ attempt_seq`); additive keys only |
| `POST /attempts/{id}/close` | new (task 3) |
| `POST /problems/{id}/reveal` | 409 `hint_locked` / `use_give_up` on judged attempts |
| `POST /problems/{id}/outcome` | INV-9 409 `evaluated_item` |
| `POST /problems/{id}/arena-reveal` | new (D17) |

The gateway refuses counted submits after `deadline_at` + grace (t4 §2.3 gateway checks) — that check is m3-09's; practice
exposes `deadline_at`. Compose sets practice's `JUDGE_BASE_URL`.

### 8 · NATS ACL PR [I]

- `make nats-acl-render` on this branch: the diff against the live `../infra/infrastructure/messaging/release.yaml` block
  must be **only** practice's user gaining `$JS.API.CONSUMER.{CREATE,INFO,MSG.NEXT}` and `$JS.ACK` for its
  `XLEARN_JUDGE` durable. Paste it; **no** pod-template annotation bump (a reload). Merge **in this sprint** — it only
  grants, so it is harmless before `v1.14.0`, and it must be merged before that tag ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).
- Verify: `config_load_time` moved, `nats-0` not restarted, no `legacy`, no permission-violation logs,
  `host-verify --cluster --nats-stage=n3` (or `n4`) green.
- **NetworkPolicy standing rule:** practice → judge :8087 is a new in-cluster HTTP caller in `v1.14.0`. judge's ingress
  already admits practice ([m3-07](sprint-m3-07.md)). If [mi-11](sprint-mi-11.md)'s `xlearn` egress policies are live,
  confirm practice's egress lists `xlearn-judge:8087` (mi-11 plans it) — if missing, add it in a separate infra PR merged
  before `v1.14.0`. If mi-11 hasn't run, there is no egress policy on practice and nothing to change.

### 9 · Tests + verify [X]

- Strategy goldens (task 2) and invariant property tests.
- **Lifecycle goldens** (store level, real Postgres): pass → immediate conclusion; give-up × an in-flight earlier pass
  (both orderings); give-up with only failures → Miss + `solution_revealed_early`; **timeout → Miss at the deadline after
  the watermark drain**; released `final` ×2 → back to `attempting`, 3rd → `self_grade_pending`; stale contract →
  `self_grade_pending`; a `self_grade_pending` pick bounded by `ceiling`; INV-9 409; the kill-switch sweep.
- **Replay golden:** the `problem_solved` v2 / `touch_concluded` emitted by the timeout, immediate-accept and
  `self_grade_pending` conclusions, replayed through review + assessment in compose, give identical projections on a
  drop-and-replay (the m2-02 runbook path).
- **Consumer:** duplicate (inbox), forged/stale (ack, no fact), tombstone, late fact after conclusion, DB error → nak →
  dead-letter row.
- **Reconciler heal** (compose, `-tags e2e`, against m3-06's real internal endpoints): a test hook acks the next
  `evaluation_completed` without storing it → a gap → the ticker pulls it → the attempt concludes exactly once.
- **Touch drain:** a judged touch past its deadline with a queued submit waits for `open_jobs = 0`, then concludes on
  the evidence; infra-only evidence → voided.
- `-race`: two tickers + the consumer + a close on one attempt → exactly one conclusion and one outbox signal.
- **Self path = v1:** every existing practice test and v1 e2e unchanged; `self@1{cap:none}` equals today's results.
- `gofmt`, `go vet`, `go test -race ./...`, `npm --prefix web run test`, `-tags e2e`, the migration lint, `sqlc diff`;
  the topology golden, budget and subject-registry tests; the NATS-auth integration test covers the new durable.

## Acceptance criteria

- [ ] **Replay/golden tests** for **timeout**, **accept** and **override** conclusions — in M3: the timeout Miss, the immediate accept, and the `self_grade_pending` pick bounded by the ceiling (the AI override lands in [m4-04](sprint-m4-04.md)) — green.
- [ ] The **reconciler heals a dropped event** in compose, against m3-06's internal endpoints.
- [ ] The **touch deadline drains to the watermark**; expired judged course attempts **conclude as Miss at the deadline**.
- [ ] The **strategy table equals v1 on the self path**; the D18 golden table and the invariant property tests are green.
- [ ] Give-up is synchronous: the solution unlocks at once; the lock is the earliest pass below `close_seq`, else Miss.
- [ ] `problem_solved` v2, `touch_concluded` and `attempt_logged` carry the new fields; the subject-registry and golden ACL tests are green.
- [ ] The ACL PR is merged; `sqlc diff` clean; CI green.

## Release

**Merge only — ships in `v1.14.0`**, tagged by [m3-13](sprint-m3-13.md) (indicative: the next free minor at tag time).
**Plus one infra PR in this sprint** (task 8), merged before `v1.14.0`.
- **Consumers before producers:** judge (the producer) has been dark since `v1.13.0`; it emits nothing until `v1.14.0`
  and m3-13's `JUDGE_BASE_URL` PR, so this consumer is bound before the first event.
- **Hand-offs to [m3-13](sprint-m3-13.md)** (record them in the decisions log): its `JUDGE_BASE_URL` infra PR sets the
  variable on **practice and the gateway** (practice first, or in the same PR); its kill-switch exit test covers both
  unsetting `JUDGE_BASE_URL` and `GRADING_OVERRIDE=self` (open judged attempts → `self_grade_pending`, nothing re-graded).

## Definition of Done

CI green (incl. `sqlc diff`, the migration lint and the topology/registry/golden-ACL tests) · merged to `main` · the ACL
PR merged and verified (no hand `kubectl`) · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md): board row, flag inventory: `GRADING_OVERRIDE` and practice's `JUDGE_BASE_URL`) ·
decisions logged (deadline Miss scoped to judged attempts, the course watermark generalization (m3-06 or here), kill switch →
`self_grade_pending`, the m3-13 hand-offs).

## Risks / watch-outs

- **A producer live before this consumer would lose events** — judge stays dark until `v1.14.0` + `JUDGE_BASE_URL`.
  Beyond that, a practice outage longer than `XLEARN_JUDGE`'s 14 d MaxAge (Discard Old) is covered by the reconciler
  plus m3-13's outbox republish runbook.
- **Row locks held across HTTP** would stall the ticker and the consumer — the select → call → lock → re-check pattern.
- **Double conclusion** (consumer vs ticker vs close) — the guarded `Conclude()`, `UNIQUE(outcome.attempt_id)` and the `-race` test.
- **A deadline Miss while an earlier submit is still evaluating** would be wrong — the watermark drain comes first, and
  `submitted_at` (not evaluation time) decides.
- **A kill switch must never Miss a learner** — the sweep moves attempts to `self_grade_pending` and voids touches.
- **Self-path regressions** — golden = v1; the self path never reads the D18 params and still reads `stages.reimplement`.
- **Consumer start blocking readiness** — start it in the background.
- **`mistake_hint` is null until m3-10** fills `mistakes.prefill` — expected, not a bug.
- **Migration races** with peers touching practice — rebase last, next free version, CI's duplicate check.
