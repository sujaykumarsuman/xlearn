# Sprint m3-14 — judge admission (L9–L15, L6), learner API + DTO allowlist, drafts, arena history/progress, telemetry, judge admin

> **Milestone:** M3 — judge plus code grader (**M3-1**, judge dark) · **Track:** product · **Order:** 44
> **Prereqs:** [m3-06](sprint-m3-06.md) (queue, runner lane, graders, contexts, internal context endpoints) · through it [m3-05](sprint-m3-05.md) (judge skeleton, schema, pack loader, erase consumer)
> **Unblocks:** [m3-07](sprint-m3-07.md) (M3-1 on prod: evalpack `v1.0.0`, ACL, tag `v1.13.0`, judge HelmRelease)
> **Release action:** **merge only (ships dark in `v1.13.0`)**, cut by [m3-07](sprint-m3-07.md). Nothing here is reachable by a learner until the gateway BFF ([m3-09](sprint-m3-09.md)) and `JUDGE_BASE_URL` ([m3-13](sprint-m3-13.md)) land in `v1.14.0`.
> **Calendar:** mid-November
> **Execute with:** [`../prompts/prompt-m3-14.md`](../prompts/prompt-m3-14.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | judge migration: idempotency keys, `job_telemetry`, `admission_reject`, `breaker_override`, `ce_cache`, `admin_audit` | X | ⬜ |
| 2 | Admission control (L9–L13, L15, L6) + per-account idempotency | X | ⬜ |
| 3 | Learner API + DTO allowlist (submissions, Runs, drafts, arena history, status) | X | ⬜ |
| 4 | Arena progress (D10): auto upsert, Mark studied, read | X | ⬜ |
| 5 | Ops surface: telemetry rows, CE cache, `judge admin status\|breaker\|review-flags` | X | ⬜ |
| 6 | Tests + verify (typed 429/413/503, kill-switch drain, IDOR, erase, compose e2e, `sqlc diff`) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row; M3 milestone row stays 🔄).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m3-06](sprint-m3-06.md) merged: the `job` queue with `SKIP LOCKED` claims and one runner lane, the `code`/`key`/`composite@1` graders, the four contexts, `evaluation_completed` through the outbox, and the internal `evaluations`/`watermark` endpoints are on `main`
- [ ] [m3-05](sprint-m3-05.md)'s judge schema, pack loader (`GET /internal/evaluable`), `aud=judge` JWT verification, IDOR scaffold and kill-switch env are on `main` (m3-06 depends on them)
- [ ] The shared body helper with the typed 413 (`internal/platform/httpx/body.go`: `ReadBody`, `http.MaxBytesReader`, L6) is on `main` ([m1-05](sprint-m1-05.md))
- [ ] **Migration numbering:** judge migrations are serialized m3-05 → m3-06 → **m3-14**. No open peer PR adds a judge migration (`gh pr list`, `git worktree list`, ListAgents); take the next free goose version at rebase (CI fails on a duplicate)
- [ ] The M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) is **not** a gate here: it gates the M3 UI sprint ([m3-11](sprint-m3-11.md)) and [m3-13](sprint-m3-13.md). This sprint is backend and ships dark

## Goal

Put the judge's **learner-facing surface and its overload protection** on top of m3-06's evaluation core:
- **admission control** in one transaction, with typed errors: daily quotas (L9), lane caps (L10), per-account pending and pacing (L11), runner slot budgets (L12), the duty-cycle breaker (L13), the kill switch (L15) and judge's body caps (L6);
- **per-account idempotency** on every learner action;
- the **learner API** behind `aud=judge` with a **DTO allowlist** (no hidden inputs, expected outputs, per-case timings or pack paths), drafts and arena history;
- **`arena_progress`** (the D10 arena done marker), upserted in the passing arena submit's result transaction, plus "Mark studied";
- **telemetry rows** and the **`judge admin`** CLI, the on-demand reads that replace opscheck's J checks under D34.

## Scope

**In**
- Admission rows 1–6 of [t4 §2.3](../research/t4-judge-contract.md#23-admission) in judge (row 7, the insert, is m3-06's), with L9–L13, L15 and L6 from [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) and the breaker/budget table of [t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments).
- `Idempotency-Key` on `POST /submissions` and `POST /runs`: HTTP mapping of m3-06's `UNIQUE(account_id, idem_key)` + `body_sha256`, and the Run-side key ([t4 §2.2](../research/t4-judge-contract.md#22-submission-request-post-apiproblemsidsubmissions--judge-post-submissions)).
- Learner routes (judge-internal, called only by the gateway): submissions, Runs, drafts, arena history, arena progress, degradation status.
- The learner DTO types and an allowlist test over every judge response ([t4 §2.6](../research/t4-judge-contract.md#26-evaluation), INV-7).
- `arena_progress` upsert inside m3-06's fenced result transaction ([t4 §2.5](../research/t4-judge-contract.md#25-job-judgejob-claim-mechanics-in-9) step 5, §3.7) and `POST /arena/{item}/studied`.
- `job_telemetry` per runner job (steal, canary, slot-ms, `profile_sha`, re-runs, queue waits), the `admission_reject` counter, the per-account CE cache, and `judge admin status|breaker|review-flags` with `admin_audit` rows.

**Out**
- The queue, runner client, graders, contexts, `evaluation_completed`, internal context endpoints → [m3-06](sprint-m3-06.md) (done).
- The gateway BFF routes, the gateway-side denylist test, the typed 413 at 64 KiB for the SPA, quota headers, L16 classic-mock caps, `GET /api/judge/status` → [m3-09](sprint-m3-09.md).
- practice's consumer, reconciler, close/give-up and strategies → [m3-08](sprint-m3-08.md).
- The llm lane, `ai_rubric`, analyzer, disputes/re-grade and the AI `judge admin` verbs (`ai-disable`, `llm-limit`, `disputes export`) → M4 ([m4-01](sprint-m4-01.md) … [m4-05](sprint-m4-05.md)). `review-flags` ships here and lists nothing until M4 writes `review_flag`.
- Any infra change (judge's HelmRelease, policies, NATS user) → [m3-07](sprint-m3-07.md).
- UI → [m3-11](sprint-m3-11.md), [m3-12](sprint-m3-12.md).

## Tasks

### 1 · judge migration [X]

`internal/judge/store/migrations/0000N_m3_admission_ops.sql` (next free judge version after m3-06; expand only, no
`xlearn:contract` marker). Tables that can hold learner code or compiler output carry **PK/FK only** (no fallible CHECKs
that would log a row, [t1 §4 judge](../research/t1-content-data-model.md)); every per-user table gets an
`(account_id, created_at)` index.

| Object | Shape | Why |
|---|---|---|
| *(already there)* `submission.idem_key`, `body_sha256`, `UNIQUE(account_id, idem_key)` | from [m3-06](sprint-m3-06.md) task 5 (`ErrReplay` / `ErrIdempotencyMismatch`) — **don't re-add**; this sprint maps them to HTTP | per-account idempotency for evaluation-bearing actions, closes and give-ups included |
| `job.idem_key uuid`, `job.body_sha256 bytea` (only if m3-06 didn't add Run keys) | partial unique `(account_id, idem_key) WHERE kind = 'run'`, `CREATE UNIQUE INDEX CONCURRENTLY` in its own `-- +goose NO TRANSACTION` file | Runs are jobs, never `submission` rows ([t4 §2.1](../research/t4-judge-contract.md#21-vocabulary)); the key dies with the job 24 h after completion (m3-06's sweeper) |
| `job_telemetry` | `job_id uuid PK`, `account_id`, `lane`, `kind`, `context_kind`, `priority`, `enqueued_at`, `claimed_at`, `finished_at`, `queue_wait_ms`, `busy_ms` (slot-ms: compile + cases + quiet re-runs = the runner's `Telemetry.SlotMs`), `runner_busy_ms` (`Telemetry.BusyMs`, diagnostic only), `other_slot_busy bool`, `steal_pct`, `canary_ratio`, `quiet_reruns`, `tests_cap_hit bool`, `throttled`, `infra_kind`, `runner_version`, `profile`, `profile_sha`, `image_digest`, `toolchain`, `cpu_model`, `boot_epoch`, `sigsys bool`, `quarantined bool`; index `(finished_at)` and `(account_id, finished_at)` (task 5 maps each column to its runner field) | [t3 §10](../research/t3-sandbox.md) "judge persists per-job runner telemetry"; the L12/L13 inputs; ADR-0035 §3 rule "anything needed after the next rollout is a Postgres row" |
| `admission_reject` | `(day date, lane text, reason text, counted bool) PK`, `n int NOT NULL DEFAULT 0` — **no account id** (an aggregate counter, so nothing to erase) | a rejected request creates no job and so no `job_telemetry` row; this is the only record of `queue_full` and the other typed 429/503s, the TR-QUEUE "any counted `queue_full`" input ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)) |
| `breaker_override` | `id`, `level text` (`auto\|0\|1\|2`), `until timestamptz`, `set_at`, `note`; rows are **kept** (`clear` sets `until = now()`), so `status` can count past level-2 pins | an operator pin for L13 via `judge admin breaker` |
| `ce_cache` | `(account_id, key bytea) PK`, `diagnostics jsonb`, `expires_at`; `key` = sha256(`profile_sha256` ‖ `harness@v` ‖ sha256 of the files) | [t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments) CE cache; **per account** so erase stays exact |
| `admin_audit` | `id`, `at`, `verb`, `target`, `detail jsonb` | the `judge admin` trail (same shape as identity's `admin_audit`, [m1-04](sprint-m1-04.md)) |

- sqlc queries in `internal/judge/store/queries/{admission,telemetry,arena,admin}.sql`; commit `sqlc generate`; **`sqlc diff` clean**.
- **Erase:** add `job_telemetry` and `ce_cache` to judge's `XLEARN_IDENTITY` erase delete list (m3-05's consumer) and
  extend its erase fixture test. `admin_audit` and `admission_reject` hold no account data.

### 2 · Admission control + idempotency [X]

`internal/judge/admission/` — one `Admit(ctx, tx, req) (Decision, error)` that m3-06's `Service.Submit` / `Service.Run`
call first inside their transaction (add the hook to m3-06's core if it isn't there), plus the HTTP mapping of m3-06's
typed errors. Order ([t4 §2.3](../research/t4-judge-contract.md#23-admission) judge checks; every error is a typed JSON
body `{error, …}`, and every 429 carries `Retry-After`):

| # | Check | Response |
|---|---|---|
| 1 | **L15 kill switch** `JUDGE_ADMISSION` (m3-05: unset/`open` → open, `closed` or anything else → closed; its `requireAdmission` middleware goes on every new-work route here) or the item not evaluable (m3-06's `ErrNotEvaluable{fallback}`: `spec_mismatch`, `unsupported`, no pack, runner lane off) | **503** `evaluation_unavailable{fallback: self\|none}`. Queued and running jobs are **not** cancelled: workers keep draining. (A request whose gateway-sourced **pin** differs from the live `contract_hash` is *not* refused: m3-06 evaluates it inline as `inconclusive(contract_changed)`, which practice turns into `self_grade_pending`, [t4 §2.2](../research/t4-judge-contract.md#22-submission-request-post-apiproblemsidsubmissions--judge-post-submissions).) |
| 2 | Idempotent replay: same `(account_id, idem_key)` | m3-06's `ErrReplay` → the original response (200/202 with the same ids); `ErrIdempotencyMismatch` → **422** `idempotency_key_reused`; header missing on `POST /submissions`/`/runs` → **400** `idempotency_key_required`. Runs get the same semantics through `job.idem_key` |
| 3 | **L6 body caps**: the request via `httpx.ReadBody(w, r, 1<<20)`; the decoded part caps ([t1 §6.3](../research/t1-content-data-model.md): source 64 KiB; multi-file 128 KiB ≤ 16 files; text 16 KiB; choice/blank 2 KiB; canvas scene 512 KiB; export 32 KiB; custom Run input 64 KiB) are enforced in m3-06's `Submit`/`Run` — map its cap error | **413** `body_too_large{max}` / **413** `part_too_large{part_id,max}`; unknown or mistyped part → **422** `invalid_part` |
| 4 | Close state (m3-06's `ErrContextClosed`, `ErrPartLocked`, the existing-close replay) | **409** `context_closed` / **409** `part_locked` / **200 with the existing close** (replay by context, INV-13) |
| 5 | `key`/`ai_rubric` in arena or Run; `ai_rubric` in mock | step → `skipped(not_evaluated_here)`; refuse (**422** `nothing_to_evaluate`) only if **every** required step would be skipped, **except in mock** (stored as the mock answer) |
| 6a | **L9 daily quotas** per account per UTC day: 200 Runs, 100 counted (course/touch/mock, not Run), 100 arena | **429** `quota_exceeded{kind, reset_at}` |
| 6b | **L10 lane cap**: runner queue ≥ 32 queued | **429** `queue_full` |
| 6c | **L11 pending**: ≤ 2 queued Runs **across contexts**, ≤ 4 queued non-Run jobs, ≥ 2 s between submits in one context; (1 running per lane per account is m3-06's claim serialization) | **429** `too_many_pending` / `too_fast` |
| 6d | **L12 runner budget** for Run and arena work: Σ `busy_ms` ≤ 10 slot-min over the trailing hour and ≤ 60 slot-min over the trailing 24 h per account (compile, CE and quiet re-runs count) | **429** `budget_exhausted{window, reset_at}`. Counted submits, mock and P0 are **never** budget-blocked ([t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments)) |
| 6e | **L13 duty-cycle breaker**: busy = Σ `busy_ms` / (2 slots × 60 min) over the trailing 60 min, or the `breaker_override` level while unexpired | **> 40 % (level 1):** arena (P2) jobs get `run_after = now() + 5 min`; Runs ≤ 1 per 30 s per account (**429** `too_fast{reason: breaker}`). **> 60 % (level 2):** only P0 closes and P1 counted submits and mock run; Runs and arena → **429** `shed{reason: breaker}` |

- **Exemptions** ([t4 §2.3](../research/t4-judge-contract.md#23-admission) row 6, [§9.2](../research/t4-judge-contract.md#92-priorities-and-caps)):
  closes (`final`, `give_up`) and the one re-grade are exempt from **all** of row 6 (quota, pending, pacing, queue-full,
  budget, breaker). They are bounded by one unreleased close per context.
- A new Run **cancels the account's queued Runs in that context** (m3-06 task 1 does this); the cancelled ones still
  count toward the daily Run quota.
- Every 2xx on `POST /submissions` and `POST /runs` carries `quota{kind: runs|counted|arena, remaining, limit, reset_at}`
  (L9), computed from the counts admission already read in the same transaction. **This is the only implementation of
  the field:** [m3-09](sprint-m3-09.md) mirrors it into `X-Quota-*` headers and AB08's "8 submits left today" and adds
  none of its own. It is part of judge's DTO golden (task 3).
- Counters are Postgres counts over indexed columns (`submission`, `job`, `job_telemetry`), inside the same transaction;
  `pg_advisory_xact_lock(hashtext(account_id))` serializes one account's admissions so two tabs can't both take the last
  slot. No in-process counters (judge may restart at any time).
- **Rejection counter:** every typed 429 and the 503 of row 1 bump `admission_reject(day, lane, reason, counted)` with an
  `INSERT … ON CONFLICT DO UPDATE SET n = n + 1` in its **own short statement after the admission transaction has
  rolled back** (best effort: a lost increment is acceptable, a failed one never changes the response). `counted` =
  the request was a course/touch/mock submit (not a Run or arena).
- Store each decision's inputs in the typed error (never a stack trace, never learner code) so the SPA copy in AB08
  ("8 submits left today", "busy, try again in 30 s") can be derived by the gateway ([m3-09](sprint-m3-09.md)).
- **Thresholds are provisional** (tuned on the laptop and S0 data): keep them in one `admission.Limits` struct with env
  overrides (`JUDGE_LIMITS_*`), defaults = the values above, so R0 tuning (ADR-0035 §5) is an infra PR, not a release.

### 3 · Learner API + DTO allowlist [X]

Routes on judge (`aud=judge` JWT, `sub` = account; **every read by id is scoped to `sub`** → another account's id is
**404**, never 403). Only the gateway calls them (m3-07's ingress). Context ids for course/touch/mock are
**gateway-resolved** and passed in the body; judge derives the **arena** id itself as `uuidv5(sub, item_id)` and
ignores any client value ([t4 §2.1](../research/t4-judge-contract.md#21-vocabulary)).

| Route | Behaviour |
|---|---|
| `POST /submissions` | `Idempotency-Key` header; body `{item_id, context{kind, id?, band?}, action, parts[], contract_hash?}` (`contract_hash` is the gateway-sourced pin for course/touch/mock; closes are `action: final\|give_up`). Admission (task 2), then m3-06's `Service.Submit`: **200** `{evaluation, quota}` (inline key-only) or **202** `{submission_id, attempt_seq?, state, poll_after_ms: 500, queue_position, quota}` |
| `GET /submissions/{id}` | learner DTO + `poll_after_ms` backing off 500 ms → 1 s → 2 s → 5 s, `ETag` → **304** ([t4 §2.7](../research/t4-judge-contract.md#27-transport)) |
| `POST /runs`, `GET /runs/{id}` | Run DTO: compile `{ok, diagnostics ≤ 8 KiB}` (learner files only), per sample/custom input `{id, status, class, got ≤ 8 KiB, sample_expected, stdout ≤ 8 KiB, stderr ≤ 2 KiB}`, ≤ 64 KiB per Run |
| `GET /drafts?context_kind=&context_id=&item_id=` · `PUT /drafts/{context_kind}/{context_id}/{part_id}` | upsert on m3-05's `draft` (key columns `NOT NULL DEFAULT ''`, `language` in the key); arena drafts keyed by item id; caps as task 2 row 3; `PUT` is idempotent (last write wins, `updated_at` server-set) |
| `GET /arena/{item}/history?cursor=` | the account's arena submissions, newest first: DTO + `submitted_at` + language + a code ref |
| `GET /arena/{item}/submissions/{id}/parts/{part_id}` | the learner's **own** code for "Copy to editor" / diff (the gateway's `withhold()` decides when it's shown, [m3-09](sprint-m3-09.md)) |
| `GET /arena/progress?path=[&item=]` · `POST /arena/{item}/studied` | task 4 (bulk for the Problems markers, or one item) |
| `GET /internal/status?path=` | **no JWT** (not per-account; ClusterIP behind MI-5a and judge's ingress, like m3-06's `/internal/*`): degradation flags for badges — `saturated` (runner queue ≥ 75 % of L10, or the runner lane unreachable), `breaker{level}`, `evaluable` count, per-item `contract_mismatch`, `admission: open\|closed` (the L15 kill switch; there is no `kill_switch` field). Feeds m3-09's `GET /api/judge/status`, which maps `admission: closed` (or judge unreachable) to `off` |

- m3-06 has a route-table test asserting judge registers only `/healthz`, `/readyz` and `/internal/*`: update it to the
  new JWT-guarded learner routes (every non-`/internal` route must sit behind `requireJWT`; new-work routes also behind
  `requireAdmission`).

**DTO allowlist** — `internal/judge/dto/` holds **typed structs only** (no `map[string]any`, no embedding of internal
records). Per [t4 §2.6](../research/t4-judge-contract.md#26-evaluation):
- all: `submission_id, status, reason, action, context_kind, submitted_at, completed_at, attempt_seq, checks[{key,label,required,met}], score, trust, provisional, evaluated_with`;
- `code` Submit: samples in full on failure; hidden = `{passed, total}` over correctness cases, `perf: passed|failed|not_run`,
  `first_failure.class`, bucketed usage — **no** stderr, case ids, ordinals, per-case timing, test names or pack paths;
  diagnostics only at learner-file positions; a compile error located in a pack/hidden file becomes CE with the fixed
  text "Hidden tests don't compile against your code; check the required signature."; check keys are generic (`tests`, `race`, `leak`);
- `key` marks and `answer_reveal` only when the caller passes `concluded=true` (the gateway composes practice state,
  [m3-09](sprint-m3-09.md)); DSA code items have no key parts in M3, so this path is covered by a fixture only.
- **Allowlist test** (`internal/judge/dto/allowlist_test.go`): reflect over every response type reachable from the route
  table and assert the JSON field set equals a golden list; a second test marshals fixtures for every evaluation shape
  (AC, WA 37/40, perf TLE, CE incl. the hidden-file case, inconclusive, `contract_changed`, `not_evaluated_here`) and
  asserts none of `expected, args, input, case_id, anchors, key, aliases, seed, gen, test_name, per_test, evalpack, /evalpack`
  appears anywhere. The gateway repeats it per endpoint in m3-09.
- Document the routes, the `quota{}` block, the `/internal/status` shape and the error codes in
  [`../../architecture/api.md`](../../architecture/api.md) (judge section). **That section is the single source
  [m3-09](sprint-m3-09.md) reads**: where a peer plan paraphrases this contract differently (a path, a JWT, a field
  name), `api.md` as merged here wins.

### 4 · Arena progress (D10) [X]

`arena_progress(account_id, item_id, first_passed_at, submits, source auto|manual)` exists from m3-05
([ADR-0027 §4](../../adr/0027-content-evalpack-and-user-data-model.md), [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)).
- **Fill m3-06's `onArenaPass` hook** — step (6) of its fenced result transaction, after the claim-token fence, the
  tombstone check and the submission lock, before the outbox step. On a `passed` arena submit, one
  `INSERT … ON CONFLICT (account_id, item_id) DO UPDATE … WHERE arena_progress.first_passed_at IS NULL`:
  `first_passed_at = submitted_at` is set **once** (a later pass changes nothing), `source` becomes `auto`, and `submits`
  records **the number of arena submits up to and including that first pass** (counted from `submission` for
  `(account, item, arena)`) — a stable "passed after N" figure; the live total comes from the history read. The fence
  guarantees a re-claimed job never double-writes, so **a passing arena submit upserts exactly once**.
- `POST /arena/{item}/studied` ("Mark studied", study mode): insert `(…, first_passed_at NULL, submits 0, source 'manual')`,
  or set `source = 'manual'` on an existing row **only while `first_passed_at IS NULL`** (an auto pass is the stronger
  marker); idempotent.
- `GET /arena/progress?path=[&item=]` → `{item_id: {done, source, first_passed_at, submits, total_submits}}` with
  `done = first_passed_at IS NOT NULL OR source = 'manual'`, for the Problems dual markers ([m3-12](sprint-m3-12.md), AB09).
- Arena never emits an event and never touches course state (D10): assert that no outbox row is written for any arena
  evaluation.

### 5 · Ops surface: telemetry, CE cache, `judge admin` [X]

- **Telemetry.** In the runner-lane result path (m3-06), write one `job_telemetry` row per runner job from the runner
  `Result` ([m3-03](sprint-m3-03.md)'s `runnerapi` types) and the claim timings (`queue_wait_ms = claimed_at − enqueued_at`).
  Field → column, explicitly:
  - `Telemetry.SlotMs` → **`busy_ms`** (slot-ms incl. compile and quiet re-runs; **the only L12/L13 input**) ·
    `Telemetry.BusyMs` → `runner_busy_ms` (diagnostic, never a budget input) · `Telemetry.OtherSlotBusy` → `other_slot_busy` ·
    `Telemetry.StealPct` → `steal_pct` · `Telemetry.CanaryRatio` → `canary_ratio` · `Telemetry.QuietReruns` → `quiet_reruns` ·
    `Telemetry.TestsCapHit` → `tests_cap_hit` (false until [p-01](sprint-p-01.md));
  - `Result.Throttled` → `throttled` · `Result.Infra.Kind` → `infra_kind` (NULL when no infra error);
  - `Versions.Runner` → `runner_version` · `.ImageDigest` → `image_digest` · `.Profile` → `profile` · `.ProfileSHA` →
    `profile_sha` · `.Toolchain` → `toolchain` · `.CPUModel` → `cpu_model` · `.BootEpoch` → `boot_epoch`
    (`CanaryMedian` is not persisted; `canary_ratio` carries it).

  Rows are kept (tiny); erase deletes them.
- **CE cache.** Before dispatching a `code` job, look up `(account_id, key)`; on a hit within 24 h, write the cached CE
  evaluation without calling the runner (no slot time spent). Store only on a CE result. m3-06's sweeper also deletes
  expired rows.
- **`judge admin`** (D34: the on-demand read that replaces opscheck's J1–J5): `cmd/judge/main.go` dispatches
  `judge admin …` before `run()`; code in `internal/judge/admin/`; static Go, `flag` parsing, distroless-safe (the
  [m1-04](sprint-m1-04.md) identity pattern). Run as
  `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin <verb>'`. Every verb, reads included, writes an
  `admin_audit` row; output never contains learner code, hidden data or secrets. `--json` on every verb.

| Verb | Shows / does | Replaces (t7 J checks) |
|---|---|---|
| `status` | pack `{version, digest, format_major, validated_against}`; evaluable counts by course and status (`ok\|spec_mismatch\|unsupported\|invalid`); kill switch; lanes (queued, running, oldest wait); breaker (level, busy % 60 min, override); last 24 h: evaluations by status/reason, `inconclusive` and `error` counts, steal p95, quiet re-run rate, SIGSYS/quarantine count, counted-submit wait p95 (`job_telemetry.queue_wait_ms`), rejections by reason with the **counted `queue_full`** count called out (`admission_reject`); last 7 d: **hours at breaker level ≥ 2**, computed on demand from `job_telemetry` (5-min steps of the trailing-60-min `busy_ms` %, `generate_series` + a range join on `finished_at`) plus any level-2 `breaker_override` intervals; runner lane last success / last failure | J1, J3, J4, J5; all three TR-QUEUE legs and the TR-STEAL inputs ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)) |
| `breaker [show\|set --level auto\|0\|1\|2 --for 2h\|clear]` | reads or pins L13 via `breaker_override` (expiry required, ≤ 24 h) | R0 tuning |
| `review-flags [--since 7d]` | evaluation ids, item, time and reason code where `review_flag` is set — **ids only, no prose**; empty until M4 | the AI `review_flag` audit list ([t4 §11.3](../research/t4-judge-contract.md#113-t6-and-t7)) |

- J2 (evalpack anonymous GET, PAT expiry) is not here: it is the evalpack CI probe plus the status.md expiry row (D34).

### 6 · Tests + verify [X]

- **Admission table tests** (`internal/judge/admission/admission_test.go`, real Postgres via the existing store-test
  harness): one case per row and per L-limit, asserting the status code, the typed error, `Retry-After`/`reset_at`, and
  that exempt actions (close, give-up) pass every row-6 limit. Include the idempotency trio (replay, 422 reuse, 400 missing),
  and assert a forced counted `queue_full` bumps `admission_reject` although the admission transaction rolled back.
- **Kill switch drain** (compose or store-level): with jobs queued, set `JUDGE_ADMISSION=closed`; new `POST /submissions` and
  `/runs` → **503** `evaluation_unavailable`; the queued jobs still reach terminal evaluations and `evaluation_completed`
  rows for counted contexts are still written.
- **Breaker**: seed `job_telemetry` to 45 % and 65 % busy → the level-1 and level-2 behaviours; an override pins and expires.
- **IDOR**: every `{id}` route with another account's id → 404 (extend m3-05's scaffold to the new routes).
- **Arena**: a passing arena submit upserts once even when the job is re-claimed (fence test); Mark studied → `manual`;
  a later auto pass → `auto`; no outbox row for arena.
- **DTO allowlist + denylist** (task 3).
- **Erase**: the extended fixture removes `job_telemetry` and `ce_cache` rows and still acks.
- **Compose e2e** (`internal/e2e`, fixture pack from m3-02): submit a Go fixture through `POST /submissions` with an
  Idempotency-Key, poll to terminal, replay the same key → same ids; exhaust a small overridden quota → typed 429.
- `gofmt`, `go vet`, `go test -race ./...`, `sqlc generate` + **`sqlc diff`**, the migration lint, `-tags e2e`.

## Acceptance criteria

- [ ] Quota and cap tests return **typed 429s** (`quota_exceeded`, `queue_full`, `too_many_pending`, `too_fast`, `budget_exhausted`, `shed`) with `Retry-After`; closes and give-ups pass every row-6 limit.
- [ ] The **kill switch** answers new work with **503** `evaluation_unavailable` while the queue **drains** to terminal evaluations.
- [ ] Body caps answer **413** `body_too_large` / `part_too_large{part_id,max}`; idempotent replay returns the original result; a reused key with a different body → 422.
- [ ] A passing arena submit upserts `arena_progress` **once** (`source=auto`); **Mark studied** sets `manual`; arena writes no outbox row.
- [ ] The DTO allowlist/denylist tests are green over every judge response type; IDOR tests → 404.
- [ ] `job_telemetry` rows are written per runner job (`busy_ms` = `Telemetry.SlotMs`); rejections land in `admission_reject`; `judge admin status|breaker|review-flags` run distroless-safe, `status` reports all three TR-QUEUE legs, and every verb writes `admin_audit`.
- [ ] Erase covers the new tables; `sqlc diff` clean; CI green.

## Release

**Merge only — ships dark in `v1.13.0`**, which [m3-07](sprint-m3-07.md) tags (indicative: the next free minor at tag
time, [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md)). No infra change, no new NATS subject
or durable, no ACL PR. Nothing is reachable: judge has no HelmRelease until m3-07, and the gateway routes and
`JUDGE_BASE_URL` arrive in `v1.14.0`.

## Definition of Done

CI green (incl. `sqlc diff` and the migration lint) · merged to `main` (no tag) · acceptance criteria met · statuses
updated (this file + [`../status.md`](../status.md) board row) · the provisional L12/L13 thresholds and the per-account
CE cache recorded in the decisions log · judge routes and error codes documented in `docs/architecture/api.md`.

## Risks / watch-outs

- **Duty-cycle breaker tuned on the laptop.** 40 %/60 % and the 10/60 slot-min budgets are inferred; revisit with prod
  telemetry (TR-QUEUE, TR-STEAL). Env overrides keep R0 tuning to an infra PR.
- **Two tabs, one last slot.** Without the per-account advisory lock two admissions can both pass a count check; keep the
  lock inside the enqueue transaction.
- **Closes must never be refused for load.** A refused give-up would leave the learner unable to stop; the exemption
  tests are the guard.
- **Allowlist drift.** A new field added to an internal record must not reach a DTO by embedding; typed DTO structs plus
  the reflection golden catch it.
- **Kill switch vs drain.** Cancelling queued jobs would break INV-1 (every accepted submission ends in one terminal
  evaluation); the switch only refuses new work.
- **Telemetry is erase-scoped** because it carries `account_id`; forgetting a table in the erase list leaves personal
  data behind — the erase fixture enumerates every judge table with an `account_id` column.
