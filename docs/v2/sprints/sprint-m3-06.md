# Sprint m3-06 — judge queue, runner lane, graders, contexts, internal context endpoints

> **Milestone:** M3 — judge plus code grader (**M3-1**, judge dark)   ·   **Track:** product (order 43)
> **Prereqs:** [m3-05](sprint-m3-05.md) (judge skeleton, schema, pack loader, dev content overlay) · [m3-03](sprint-m3-03.md) (runner contract types) · [m3-04](sprint-m3-04.md) (Go/C++/Python profiles, `internal/platform/harness`, `runnerapi/lint`, `tl_multiplier`) · [m3-02](sprint-m3-02.md) (`internal/packspec/gen`, `internal/platform/checker`, the `constraints[]` validator)
> **Unblocks:** [m3-14](sprint-m3-14.md) (admission + learner API on this core) → [m3-07](sprint-m3-07.md) (v1.13.0) · consumed by [m3-08](sprint-m3-08.md) (practice consumer, reconciler, touch drain) and [m3-09](sprint-m3-09.md) (BFF types)
> **Release action:** **merge only (ships dark in v1.13.0**, tagged by [m3-07](sprint-m3-07.md)). No infra PR here (hand-offs to m3-07 below).
> **Calendar:** early–mid November.
> **Execute with:** [`../prompts/prompt-m3-06.md`](../prompts/prompt-m3-06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Job queue: enqueue, fenced claim, lease + heartbeat, priorities with ageing | X | ⬜ |
| 2 | Runner lane client: bearer, streaming, Result validation, retry classification, `throttled` | X | ⬜ |
| 3 | Case sources: lazy pack cases, m3-02's generator registry, scratch files | X | ⬜ |
| 4 | Graders: closed registry + invariant wrapper; `code@1` (harness, checkers, term→class); `key@1`; `composite@1` | X | ⬜ |
| 5 | Contexts and the submit core (course, touch, mock, arena; `run` action; closes) — no learner route | X | ⬜ |
| 6 | Result transaction + `evaluation_completed` (counted contexts only) + `RELAY_INTERVAL` | X | ⬜ |
| 7 | Sweeper ticker (leases, Run payloads, drafts) | X | ⬜ |
| 8 | Internal context endpoints (`evaluations?seq_lt`, `{kind}` `watermark`) + `judgeapi` client + contract fixtures | X | ⬜ |
| 9 | Tests: unit/property, fake-runner e2e, real-runner compose lane (Go/C++/Python) | X | ⬜ |
| 10 | Docs, ADR, hand-offs (m3-07, m3-08, m3-14) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> the **M3** milestone row, the decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m3-05](sprint-m3-05.md) merged: `cmd/judge`, the judge schema (`job`, `context_counter`, `submission`, `evaluation`,
      `draft`, `outbox`, `erased_account`), `internal/judge/pack` with `OpenCases`, `internal/platform/judgeapi`,
      `JUDGE_ADMISSION`, `store.IsErased`, and the **dev content overlay** (`FIXTURE_CONTENT_DIR` + `DEV_AUTH`, the
      `internal/course` helper) that makes the `fixture` course (fx-001..fx-003) evaluable.
- [ ] [m3-03](sprint-m3-03.md) merged: `internal/platform/runnerapi` (`Job`/`Result` with the T3 amendments — `HiddenFiles`,
      `Tests[]`, `OutputMode`, `Cases.OutputSHA256`, `Versions{…, ProfileSHA, CPUModel, CanaryMedian, BootEpoch}`;
      `ValidateResult`; the golden `job.json` / `result.json` fixtures), the runner's job endpoint path and `GET /v1/profiles`
      (which reports `mode`).
- [ ] [m3-04](sprint-m3-04.md) merged: the `go`, `cpp` and `python` profiles (each serving `tl_multiplier` on
      `GET /v1/profiles`); **`internal/platform/harness`** (`Generate`, `EncodeInput`, `DecodeOutput`,
      `CanonicalizableOutput`, the `func-json@1` / `class-ops@1` codecs); **`internal/platform/runnerapi/lint`**
      (`lint.Check`); and its term→class mapping fixtures. The acceptance ("Go/C++/Python fixtures") and these imports need
      it, so it is a hard gate here; in the global order m3-04 lands first. If it hasn't, stop and report.
- [ ] On `main`: `internal/course/keys` ([m2-01](sprint-m2-01.md)), `internal/course/canon` ([m3-01](sprint-m3-01.md)); from
      [m3-02](sprint-m3-02.md): the synthetic fixture pack, **`internal/packspec/gen`** (the seeded perf generator registry,
      golden hash per `gen@v`), **`internal/platform/checker`** (the closed checker registry) and **`internal/packspec`** (the
      canonical case line and the generic `constraints[]` validator). This sprint imports all three — one definition each.
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no peer editing `internal/judge/`,
      `topology.go`, judge migrations or the shared packages this sprint may extend (`internal/packspec/gen`,
      `internal/platform/checker`, `internal/platform/item/parts`) (m3-14 follows this sprint; take the next free goose version at rebase).

_Not a gate here:_ the M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) blocks the M3 UI
sprint ([m3-11](sprint-m3-11.md)); this sprint is dark backend code.

## Goal

Build judge's **evaluation core**: a Postgres `FOR UPDATE SKIP LOCKED` job queue with **one runner lane** (2 workers),
fenced claims and a sweeper; the runner client with T3's retry classification and the bounded `throttled` re-queue; the
closed grader registry — `code@1` with typed checker verdicts, `key@1` inline, and the `composite@1` root; the four
contexts (course, touch, mock, arena) plus the `run` action; `xlearn.judge.evaluation_completed` for **counted course and
touch evaluations only**; and the two read-only internal endpoints practice's reconciler and its touch and course (D15) deadlines need
([t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)). Everything is callable in-process and over
the internal endpoints; **no learner HTTP route** exists until [m3-14](sprint-m3-14.md) wraps this core with admission,
idempotency semantics and the DTO allowlist.

## Scope

**In**
- Queue + lane + sweeper ([t4 §2.5](../research/t4-judge-contract.md#25-job-judgejob-claim-mechanics-in-9),
  [§9.1–§9.2](../research/t4-judge-contract.md#91-lanes), [§9.6](../research/t4-judge-contract.md#96-sweeper-one-judge-ticker)).
- Runner client ([t3 §5.3](../research/t3-sandbox.md#53-api-judge-is-the-only-caller), [§5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded),
  [§5.9](../research/t3-sandbox.md#59-typed-infra-errors-and-poison-pills); [t4 §11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology)).
- Graders `code@1`, `key@1`, `composite@1` ([t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide),
  [§4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key), [§5.1–§5.3](../research/t4-judge-contract.md#52-grader-contract-go);
  [t1 §7.1](../research/t1-content-data-model.md#71-package-format) checker registry); results `passed | failed |
  inconclusive | error` with reasons (incl. `contract_changed`, `not_evaluated_here`).
- Contexts course / touch / **mock** (the context M6a builds on) / arena (never counts, D10; id `uuidv5(account, item)`);
  `run` is an action ([t4 §2.1](../research/t4-judge-contract.md#2-the-common-contract), [§4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context)).
- `evaluation_completed` via the outbox with `RELAY_INTERVAL=1s` ([t4 §2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs)).
- `GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T` and `GET /internal/contexts/{kind}/{id}/watermark`
  (`kind ∈ course | touch`; [t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker), [§11.4 #11](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag)).
- **Runtime use of the shared packages — imported, never copied:** m3-02's `internal/packspec/gen` (seeded perf generators),
  `internal/platform/checker` (checkers) and `internal/packspec` (case lines, the `constraints[]` validator); m3-04's
  `internal/platform/harness` (harness files, input encoding, output decoding) and `internal/platform/runnerapi/lint`
  (pre-enqueue source lint). t4 §5.1 names `internal/judge/gen` for the generators; they live in `internal/packspec/gen`
  because packlint and judge import one registry — record that move in the ADR, never a second copy.
- The **part registry** `internal/platform/item/parts` ([t4 §5.1](../research/t4-judge-contract.md#51-registries)): create it
  here if it isn't on `main`, with the M3 part types and the [t1 §6.3](../research/t1-content-data-model.md#63-caps-and-quotas-enforced-by-judge-t3t4-own-rate-limiting)
  caps as a closed list; P extends it.

**Out**
- Admission L9–L13 and L15 enforcement, L6 body caps, HTTP idempotency replay/422, the learner API + DTO allowlist, drafts
  API, arena history, the **`arena_progress` upsert** (this sprint leaves the named hook), telemetry rows, `judge admin`
  → [m3-14](sprint-m3-14.md).
- The `llm` lane, `ai_rubric`, analyzer, regrade, provisional → M4 ([m4-02](sprint-m4-02.md), [m4-03](sprint-m4-03.md), [m4-04](sprint-m4-04.md)).
- practice's consumer, inbox, reconciler, touch drain and strategies → [m3-08](sprint-m3-08.md). BFF routes, `aud=judge`
  minting, the denylist test → [m3-09](sprint-m3-09.md).
- Mock Runs/finals exposed to learners and mock `evaluation_completed` (silent in v2.0) → M6a ([m6a-04](sprint-m6a-04.md)).
- go-race / RACE / DEADLOCK / LEAK verdicts → P ([p-01](sprint-p-01.md)). Infra (scratch volume, runner env, practice →
  judge ingress) → [m3-07](sprint-m3-07.md).

## Tasks

### 1 · Job queue [X]

`internal/judge/queue` over m3-05's `job` table ([t4 §2.5](../research/t4-judge-contract.md#25-job-judgejob-claim-mechanics-in-9), [§9.1–§9.2](../research/t4-judge-contract.md#91-lanes)):
- **Enqueue** inside the caller's tx. Priorities: **P0** closes (`final`, `give_up`) · **P1** counted `submit` and `run` ·
  **P2** arena `submit` · P3 `analyze` (M4, reserved). Lane `runner` only; enqueuing an `llm` job is an error until M4.
  `key` steps inside a code job run in-process in whichever worker holds it.
- **Claim, one tx:** candidates `WHERE lane = $1 AND state = 'queued' AND (run_after IS NULL OR run_after <= now())`,
  ordered by `priority − floor(wait / 30 s)` (ageing), `FOR UPDATE SKIP LOCKED LIMIT 8`; for each,
  `pg_try_advisory_xact_lock(hashtext(account_id::text || lane))` then `NOT EXISTS (running job for that account and lane)`
  (serializes per-account claims under READ COMMITTED: **1 running per lane per account**); the first eligible gets
  `claim_token = claim_token + 1`, `state = 'running'`, `lease_until = now() + 30 s`, `worker_id`, and its **versions at
  claim** (grader, harness, checker and generator `@v`; `pack_digest` and pack version; `item_hash`, `contract_hash`,
  `content_hash`). **No per-context FIFO** (practice orders by seq).
- **Workers:** `JUDGE_RUNNER_WORKERS` (default 2, clamped to 2 — the runner has 2 slots); poll every 1 s (no LISTEN/NOTIFY);
  a heartbeat every 10 s extends `lease_until … WHERE id = $1 AND claim_token = $t AND state = 'running'` — zero rows means
  the lease is lost, so cancel the runner call's context.
- **Fenced result write:** `UPDATE job SET state = 'done', finished_at = now(), … WHERE id = $1 AND state = 'running' AND
  claim_token = $t`; zero rows → drop the result (task 6 continues only on one row).
- **Re-claim cap:** a job whose lease expired 3 times ends `inconclusive(infra_exhausted)` through the normal result path (INV-1).
- **Run cancellation:** a new Run cancels the account's queued Runs in the same context (`state = 'cancelled'`); a close
  cancels the context's queued Runs.
- Queue latency never penalizes the learner: every timer is practice's, applied to `submitted_at` (INV-6).

### 2 · Runner lane client [X]

`internal/judge/runner` over `internal/platform/runnerapi`:
- **Config:** `RUNNER_BASE_URL` (unset → the runner lane is off: code items report `run_available = submit_available =
  false` in `/internal/evaluable`, "presence by config", ADR-0034 §2); bearer from `RUNNER_TOKEN_FILE` (the mounted secret,
  key `token`, prod) or `RUNNER_TOKEN` (compose).
- **Call:** the job endpoint exactly as m3-03 shipped it (t3 §5.3: `POST /v1/jobs`), HTTP/1.1, a length-prefixed streamed
  body (`job.json`, then case inputs), caps enforced **before** sending (≤ 8 MiB per case, ≤ 16 MiB per job); synchronous
  within the lease; cancelling the context closes the connection, which kills the job on the runner.
- **Profiles:** `GET /v1/profiles` at start and every 5 min (non-fatal) → the offered `profile@v` / harness majors feed
  item availability and the recorded versions; each profile's **`tl_multiplier`** (and its `calibrated` flag, m3-04) feeds
  the time limit in task 4 and is recorded in the evaluation's versions.
- **Refuse a `dev` runner in production** (m3-03's hand-off): `GET /v1/profiles` reports `mode`. judge accepts
  `mode=dev` only when `DEV_AUTH` is set (compose and CI — the same guard as m3-05's overlay); otherwise the runner lane
  is **off** (code items `run_available = submit_available = false`) with an ERROR log `runner_dev_mode`, re-checked on
  every profiles poll. Test both ways.
- **Validate every Result** with m3-03's **`runnerapi.ValidateResult(job, res)`** (case ids and counts match the job, enums
  valid, size caps, profile and boot epoch consistent); a mismatch is infra `setup`. judge **infers `runner_oom`** from a
  dropped connection plus a changed `BootEpoch` (m3-03's `errors.go`); the L14 45 s tests cap (`Telemetry.TestsCapHit`)
  is **TLE** through the ordinary term mapping, never infra.
- **Retry classification** (t3 §5.9 + t4 §9.1; infra-retry backoff 2 s, 10 s, 30 s):

  | Runner signal | Handling | Budget / terminal |
  |---|---|---|
  | `Infra.setup`, or a Result mismatch | re-queue with `run_after` | ≤ 1 retry → `inconclusive(infra_exhausted)` |
  | `Infra.runner_oom`, `Infra.job_timeout`, an unexplained drop mid-job | re-queue, `poison_count++` | the 2nd poison on the same submission → `inconclusive(runner_unavailable)`, ERROR log "quarantine" (D34: no alert) |
  | `Infra.killed` (drain / rollout) | re-queue as `saturated` | spends no retry and no poison |
  | 503 / 429 `saturated`, connection refused | re-queue with backoff | no poison; **liveness bound:** a job still unanswered 30 min after `submitted_at` → `inconclusive(runner_unavailable)` (INV-1; record the bound) |
  | 401 / 403 (bearer drift between judge and runner, see [mi-10](sprint-mi-10.md)) | like `saturated`, plus ERROR `runner_auth_rejected` | after 3 → `inconclusive(runner_unavailable)` |
  | `Throttled = true` | re-queue **once** with `run_after = now() + 5 min` | still throttled → `inconclusive(runner_throttled)` (INV-14) |
  | 400 contract error | `error(internal)` — a judge bug | ERROR log |

  Learner-caused verdicts (TLE, MLE, RE of the learner's process) are never retried and never `inconclusive` (INV-14).

### 3 · Case sources [X]

- **Samples** come from the embedded public item (`samples[]`), never the pack — so Run works on `spec_mismatch` items.
- **Hidden cases:** `pack.OpenCases(item)` (m3-05) streams `cases.jsonl.zst`; decode each line with **`internal/packspec`'s
  canonical case type** (m3-02: `{id, args | ctor+ops+args | gen+params+seed, expected, tags}`) — never a judge-local
  struct. Case ids are content hashes, internal to judge only.
- **Seeded perf specs** (a case line in `{gen, params, seed}` form, `expected: literal | {sha256, bytes}`,
  [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)) stream from **m3-02's `internal/packspec/gen`** —
  `Generate(w, params, seed)`, the closed public registry `int_array@1`, `string@1`, `permutation@1`, `tree@1`, `graph@1`,
  `op_sequence@1`, each with its golden hash. judge **imports** it; the names are exactly t4 §4.1's and m3-02's. A new
  generator or a behaviour change is a new `@v` added **there** (with its golden hash), never a judge-local registry — the
  evalpack CI imports the same package at `validated_against`, so two registries would silently turn correct code into WA.
- **Materialize** each input to a per-job file under `JUDGE_SCRATCH_DIR` (default `os.TempDir()`; prod needs an `emptyDir`
  because the root FS is read-only — hand-off to m3-07), stream it, delete it after the job; never hold an input twice in
  memory (judge's limit is 256 Mi).
- **Expected outputs never go to the runner** (honor in-process profiles excepted, P); judge compares. Use
  `OutputMode: sha256` for large outputs only when `harness.CanonicalizableOutput` (m3-04) holds for the item's checker
  (`exact`).

### 4 · Graders [X]

`internal/judge/grader` — a **closed registry** in one file ([t4 §5.1](../research/t4-judge-contract.md#51-registries)):
`code@1`, `key@1` and the `composite@1` root, implementing t4 §5.2's `Grader` interface (`Kind`, `Version`, `Caps`,
`ValidateSpec`, `ValidatePack`, `Estimate`, `Evaluate`, `Present`, `TrustOf`). The build's supported set is asserted equal
to m3-05's `{code, key}` constant.

- **Invariant wrapper** (t4 §5.2): a context outside `Caps.Contexts` → `skipped(not_evaluated_here)`; in `run` and `arena`
  the `PackView` is a samples-only type **with no `Hidden()` method** (compile-time); `inconclusive` without a platform-side
  reason is rewritten to `error(internal)` and logged (INV-14); `Present()` output is what m3-14 exposes and m3-09's
  denylist checks.
- **`code@1`:** build the `runnerapi.Job` — profile per `language` (item `languages[]` ∩ offered profiles), harness from the
  item (`func-json@1` / `class-ops@1`); the learner files plus the harness files from **`harness.Generate(lang, harness,
  sig)`** (`internal/platform/harness`: created by m3-02, extended by m3-04 — one package); case inputs encoded with **`harness.EncodeInput`**; limits = the item's
  Go-based `time_ms` × the profile's **`tl_multiplier` from `GET /v1/profiles`** (task 2; the runner enforces only the CPU
  TL judge sends), `mode` run|submit, `StopGroupOn {"sample": "any_fail", "perf": "tle"}`; order: samples → every
  correctness case (edge, then random) → the perf block. Outputs are decoded with **`harness.DecodeOutput`** (a harness
  `{"error": …}` frame → WA, m3-04).
  - **Checkers** — **m3-02's `internal/platform/checker`**, the closed registry packlint's gates already use (typed verdicts, never
    formatted values): `exact`, `unordered`, `unordered_deep`, `float_abs(eps)`, `float_rel(eps)`, `set_equal`, `any_of`.
    Import it; a missing checker is added there, never in `internal/judge`.
  - **Verdicts:** the **production term → class mapper** lives here (m3-04 left it to this sprint) and is tested against
    **m3-04's term→class fixtures**; within a case error > TLE > MLE > OLE > RE > WA; classes AC WA TLE MLE OLE RE CE
    REJECTED (RACE, DEADLOCK, LEAK reserved for P); `CE` and `REJECTED` are `failed` with a free class (practice decides
    counting). `REJECTED` comes only from the pre-enqueue source lint (task 5).
  - **Hidden feedback:** samples in full on failure (Run and sample views); hidden = `{passed, total}` over correctness
    cases, **one perf bit** (`passed | failed | not_run`), `first_failure{class, on}`; no case ids, ordinals, timings or
    stderr for hidden cases. **Diagnostics only at learner-file positions**; a compile error in a pack or hidden file
    becomes CE with the fixed text "Hidden tests don't compile against your code; check the required signature."
    Check keys are generic (`tests`), never test names.
  - **Signals** (code family, [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)):
    `ce_only, sample_failed, wa, wa_edge_only, tle_perf_only, tle_small, re_index, re_nil, re_stack, re_other, mle, ole`,
    joined from verdicts and pack case tags, outside the sandbox; they travel only in `evaluation_completed`.
- **`key@1`:** inline and synchronous (< 50 ms), **counted contexts only** (course, touch, mock); arena and Run →
  `skipped(not_evaluated_here)` (INV-4). A `final` part locks once per (context, part) → `ErrPartLocked`. **One evaluator:**
  text / alias and `blank.big_o` reuse `internal/course/keys` (m2-01); add the quiz modes P's pilot needs (t4 §4.3):
  `choice.single`, `choice.multi` (`all_or_nothing | right_minus_wrong | per_option`), `choice.order`
  (`exact | adjacent_pairs | kendall`), `blank.text` (closed normalizers `trim, casefold, collapse_ws, strip_punct, nfkc`;
  `typo: 0|1`), `blank.numeric` (tolerance after unit-family conversion), `blank.predict_output`. `key_source: public:*` →
  `trust = honor`; `pack` keys → `checked`. A chosen distractor → `misconception:<category>`; `key_wrong:<field>`.
  **DSA touch probe lock-ins stay in practice through M3** (m2-01's path, no behaviour change on a live M2 surface); the
  package is the single implementation both call (T4 §12 "one evaluator"). Record it in the ADR.
- **`composite@1`** ([t4 §5.3](../research/t4-judge-contract.md#53-composite1-the-root-of-every-item-never-nested)): steps
  `{kind, inputs[], required, weight, after: always | gates_passed}` in order; status = any required `failed` → failed,
  else any required `error` → error, else any required `inconclusive` / `skipped(not_evaluated_here)` → inconclusive (first
  reason), else passed; give-up skips excluded; score `Σ w·v/max ÷ Σ w` over steps not context-skipped; `graded_by = auto`
  (no `ai_rubric` in M3); `trust = honor` iff any contributing step is honor or `review_flag` is set.

If the session runs long, the quiz key modes beyond `internal/course/keys` may move to a follow-up before P
([p-02](sprint-p-02.md)); record the split in the decisions log.

### 5 · Contexts and the submit core [X]

`internal/judge/submit.go` — `Service.Submit(ctx, SubmitRequest) (SubmitResult, error)`, the transactional core **with no
learner route** (m3-14 wraps it). The caller passes the account (the JWT `sub`), `item_id`, `context{kind, id, band?}`,
`action`, `parts[]` (raw JSON values), `idem_key` and the **server-sourced** `contract_hash` pin (course / touch: the attempt
pin; mock: the `mock_session_item` pin; arena: live). One tx ([t4 §2.2–§2.4](../research/t4-judge-contract.md#2-the-common-contract)):
1. Admission open (`JUDGE_ADMISSION`) and the item `ok` in the pack index, else `ErrNotEvaluable{fallback}`.
2. **Arena:** the context id is recomputed as `uuidv5(account, item)` (a namespace constant in `judgeapi`); a client value
   is ignored. Kinds and actions validated.
3. Parts decoded and capped in Go against the part registry **`internal/platform/item/parts`** (Scope; t1 §6.3: code 64 KiB;
   multi-file 128 KiB / 16 files; …) → `ErrPartTooLarge{part_id, max}` / `ErrInvalidPart{part_id}` (m3-14: 413 / 422).
   **Source lint:** each code part runs through **`lint.Check(profile, files)`** (m3-04's `internal/platform/runnerapi/lint`)
   before anything is enqueued; a violation becomes an inline `failed` evaluation with class **`REJECTED`** (a free class)
   and a generic message, written like a key-only evaluation in step 9 — never a job, never rule detail.
4. Insert the submission; `UNIQUE(account_id, idem_key)` → same `body_sha256` = `ErrReplay{submission_id}`, different =
   `ErrIdempotencyMismatch` (m3-14 maps to 200 / 422).
5. **Closes** (`final`, `give_up`) set `closes_context`: a second close while one is unreleased returns the existing close;
   a non-close submit then gets `ErrContextClosed`; a locked `final` part gets `ErrPartLocked`; the close cancels the
   context's queued Runs and enqueues at P0.
6. **Seq:** evaluation-bearing rows take `attempt_seq` from `context_counter` (INV-12).
7. **Pin mismatch** (pin ≠ live `contract_hash`) → evaluate inline as `inconclusive(contract_changed)`, no runner.
8. **Steps per context** ([t4 §4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context)): course → the item's
   `grader[]`; touch → the `grader[]` steps whose parts the band lists (DSA re-solve: the code step); mock → `grader[]` minus
   `ai_rubric`, stored as the mock answer even when every step is skipped, **evidence only**; arena → `code` steps only.
   **Nothing to evaluate** ([t4 §2.3](../research/t4-judge-contract.md#23-admission) row 5): if every required step
   would be skipped here (e.g. an arena Submit on a key-only item) → **`ErrNothingToEvaluate`** and the tx rolls back
   (m3-14 maps it to 422 `nothing_to_evaluate`) — **except in mock**, where the answer is stored as evidence.
9. Key-only (or a lint `REJECTED`) → evaluate inline and write the evaluation (**+ the outbox row iff `context_kind ∈
   {course, touch}`**, task 6 — never mock or arena) in the same tx → `200 {evaluation}` shape; otherwise insert a job →
   `202 {submission_id, state, poll_after_ms: 500, queue_position}` shape.
- `Service.Run(ctx, RunRequest) (run_id)` — a job only (`payload` = parts + custom input ≤ 64 KiB), samples + custom input,
  no seq and no submission row; custom input decoded against the item signature's closed type registry and checked
  against the item's public `constraints[]` with **m3-02's generic validator in `internal/packspec`** (the one packlint
  applies to pack inputs) → `ErrInvalidPart{part_id: "input"}` on a violation; a key-only item → `ErrNothingToEvaluate`
  (row 5); the same source lint as Submit, where a violation finishes the Run at once with class `REJECTED` (no runner
  call). Results via `Service.GetRun`.
- Reads for m3-14: `GetSubmission(account, id)`, `GetRun(account, id)` — `sub`-scoped (m3-05's IDOR lint).

### 6 · Result transaction + `evaluation_completed` [X]

In the fenced tx ([t4 §2.5](../research/t4-judge-contract.md#25-job-judgejob-claim-mechanics-in-9)): (1) the fenced `job`
update; (2) the tombstone check (`IsErased` → drop); (3) `SELECT submission … FOR UPDATE` (gone → drop: erase won);
(4) insert the evaluation with versions (claim-time versions + the runner `Result.Versions`); (5) **release**: a `final`
close whose evaluation is `inconclusive` or `error` gets `released = true` — a `give_up` close is never released (INV-13);
(6) **the `onArenaPass` hook** — an explicit, tested no-op extension point that [m3-14](sprint-m3-14.md) fills with the
`arena_progress` upsert; (7) the outbox row `xlearn.judge.evaluation_completed` **iff** `context_kind ∈ {course, touch}`
(never mock — silent in v2.0 — and never arena).
- **Payload** (v2 envelope, m1-02; [t4 §2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs)):
  `account_id, evaluation_id, submission_id, path_slug, item_id, context_kind, context_id, action, closes_context, released,
  attempt_seq, submitted_at, completed_at, status, reason, class?, checks[{key, required, met, locked_at?}], score?, hidden?,
  perf?, parts[], signals[], trust, provisional_eligible, regrade_of?, reused_from?, contract_hash` — enums, numbers and refs
  only, target ≤ 1.5 KiB, hard cap the 16 KiB envelope (mi-05). Golden fixture
  `internal/platform/events/testdata/evaluation_completed.v2.json`, shared with [m3-08](sprint-m3-08.md).
- **`RELAY_INTERVAL`** env on judge's relay (default **1 s**, [t4 §2.7](../research/t4-judge-contract.md#2-the-common-contract)).
- **`topology.go`:** `XLEARN_JUDGE` Emits `evaluation_completed`; identity's ack durable (filter `xlearn.judge.account_erased`)
  is unaffected; the registry test stays green; re-render the goldens (expected: no ACL diff — judge already publishes
  `xlearn.judge.>`). practice's durable arrives in m3-08.
- **Consumers before producers** ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)):
  nothing consumes `evaluation_completed` until v1.14.0; the producer is **dark behind a T-2 switch** — the gateway's
  `JUDGE_BASE_URL` stays unset until [m3-13](sprint-m3-13.md), and judge has no learner route here — so no counted
  submission can exist before the consumer. `XLEARN_JUDGE` keeps 14 days, and practice's reconciler pulls any gap.

### 7 · Sweeper ticker [X]

One judge ticker ([t4 §9.6](../research/t4-judge-contract.md#96-sweeper-one-judge-ticker)), `JUDGE_SWEEP_INTERVAL`
(default 60 s), stopped before the pool closes:
- re-queue `running` jobs with `lease_until < now()` (`claim_token++`, `state = 'queued'`, re-claim count +1; past the cap
  → task 1's terminal path);
- delete Run jobs (payload and output) finished more than 24 h ago;
- delete drafts with `updated_at < now() − 90 days`, batched (500 per statement; a seq scan is fine at this scale);
- remove scratch files older than 1 h (crash leftovers).

### 8 · Internal context endpoints [X]

ClusterIP-only, no user JWT (the ADR-0016 internal model behind MI-5a; m3-07's judge ingress admits the gateway and
practice), read-only, idempotent, **never enqueue** ([t4 §3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)):
- **`GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T`** — `kind ∈ {course, touch}` (else 404); `seq_lt` required,
  positive (else 400); returns `{context_kind, context_id, max_seq, evaluations: [...]}` — the primary evaluations of the
  evaluation-bearing submissions with `attempt_seq < T`, in seq order, **each in the exact `evaluation_completed` payload
  shape**, so practice validates pulled facts with the same code (the gap reconciler).
- **`GET /internal/contexts/{kind}/{id}/watermark`** — `kind ∈ {course, touch}` (else 404) → `{max_seq, open_jobs}`:
  `max_seq` = `context_counter.last_seq` (0 if none); `open_jobs` = jobs in `queued | running` for that context's
  evaluation-bearing submissions (Runs excluded) — the touch deadline / End review drain **and** the D15 course attempt
  deadline drain ([m3-08](sprint-m3-08.md)). Shipping both kinds here keeps m3-08 free of judge-side edits.
- **`internal/platform/judgeapi`:** `EvaluationFact`, `Watermark`, a small `Client` (`EvaluationsBefore`,
  `Watermark(kind, id)`, `Evaluable`) and fixtures `testdata/{evaluations,watermark}.json` (a course and a touch case); a contract test runs the real handlers on real PG and
  compares with the fixtures, and a practice-side fake server built from the same fixtures is exported for m3-08.

### 9 · Tests [X]

- **Unit / property:** claim concurrency under `-race` (4 workers, 50 accounts: ≤ 1 running per account per lane, one
  winner per `claim_token`); fencing (a stale token's write is dropped); lease expiry → re-claim → one evaluation; the
  retry-classification table and the throttled path (fake clock); the `dev`-runner refusal both ways; `runner_oom`
  inference; the job build (harness files from `harness.Generate`, TL = `time_ms × tl_multiplier`, `OutputMode: sha256`
  only when `CanonicalizableOutput`); the term→class mapper on **m3-04's fixtures** and verdict precedence; checker
  integration through `internal/platform/checker` (its own tables stay m3-02's); the lint `REJECTED` path (no runner call); row 5
  (`ErrNothingToEvaluate`, not in mock); part caps; the composite status table; key modes; INV-1, INV-4, INV-12, INV-13, INV-14 as property tests; the payload golden and the size cap; a
  route-table test that judge registers only `/healthz`, `/readyz` and `/internal/*`.
- **Fake-runner e2e** (`internal/judge/runnerfake` speaks the m3-03 API over HTTP with scripted Results; default `go test`
  and the `e2e` lane with real PG + embedded NATS): submit → claim → Result → evaluation → outbox → JetStream
  `xlearn.judge.evaluation_completed` for course/touch, nothing for mock/arena; a worker killed mid-job still yields exactly
  one evaluation.
- **Real-runner lane (the acceptance):** a compose profile `runner` (reuse m3-15's if present; otherwise build
  `deploy/runner.Dockerfile`, `privileged: true` **local only**, a dev `RUNNER_TOKEN`, port 8090) and
  `internal/e2e/judge_runner_test.go` (`-tags e2e,runner`) driving `Service.Submit` in-process against it. judge's
  content comes through **m3-05's dev content overlay** (`internal/judge/testdata/content`, the `fixture` course), without
  which every fixture item is `invalid` and nothing is evaluable; the compose runner runs `RUNNER_MODE=dev`, which judge
  accepts only because `DEV_AUTH` is set (task 2). For each of
  **Go, C++ and Python** on the fixture items: AC (passed, all hidden, perf passed), WA (failed, `hidden{passed<total}`),
  perf TLE (correctness all passed, perf failed), CE (class CE, learner-file diagnostics only), plus a Run with custom input.
  Synthetic learner solutions in `internal/judge/testdata/solutions/<item>/`. The jail needs a privileged Linux host:
  mirror m3-03/m3-04's Linux CI job as `judge-runner-e2e`; on macOS use the spike's multipass VM or rely on CI.

### 10 · Docs, ADR, hand-offs [X]

- **Docs:** `docs/architecture/events.md` (the `evaluation_completed` row: producer dark until the gateway's
  `JUDGE_BASE_URL`, consumer practice from v1.14.0), `services.md` (queue, lane, sweeper, runner caller, practice → judge
  internal reads), `data-model.md` (job lifecycle), `api.md` (the internal context endpoints).
- **ADR** "judge evaluation core (M3)": the claim SQL and fencing; the retry table incl. the 401 rule and the 30-min
  liveness bound; the shared packages judge imports — `internal/packspec/gen` (recorded as a **move** from t4 §5.1's
  `internal/judge/gen`: one registry for packlint and judge), `internal/platform/checker`, `internal/platform/harness`,
  `runnerapi/lint` and `internal/platform/item/parts`; the `dev`-runner refusal; `tl_multiplier`; `key@1` reusing
  `internal/course/keys` with DSA probe lock-ins kept in practice through M3; mock silent; `RELAY_INTERVAL` 1 s. Take the **next free number** after checking peers.
- **Hand-offs** (PR description + status.md decisions log):
  - **[m3-07](sprint-m3-07.md):** judge env `RUNNER_BASE_URL=http://xlearn-runner.xlearn-runner.svc.cluster.local:8090`,
    `RUNNER_TOKEN_FILE` → the `xlearn-judge-runner-auth` secret mount (key `token`, same value as the runner's),
    `RELAY_INTERVAL=1s`, and **`JUDGE_SCRATCH_DIR` on an `emptyDir`** (the root FS is read-only; without it every code job
    fails as infra `setup`).
  - **[m3-08](sprint-m3-08.md):** `judgeapi.Client`, the fixtures and the practice-side fake; `evaluation_completed.v2.json`;
    **the watermark already serves `kind ∈ course|touch`** (`Client.Watermark(kind, id)`), so m3-08's "generalize the
    watermark" step reduces to using it — no judge-side edit there.
  - **[m3-14](sprint-m3-14.md):** `Service.Submit/Run/GetSubmission/GetRun`, the error types to map (`ErrNotEvaluable`,
    `ErrReplay`, `ErrIdempotencyMismatch`, `ErrContextClosed`, `ErrPartLocked`, `ErrNothingToEvaluate` → 422
    `nothing_to_evaluate`, `ErrPartTooLarge` → 413 `part_too_large`, `ErrInvalidPart` → 422 `invalid_part`), the lint
    `REJECTED` evaluation (already an evaluation — nothing to map), and the `onArenaPass` hook.

## Acceptance criteria

- [ ] **Compose e2e: submit → runner → evaluation for Go, C++ and Python fixtures** — AC, WA with a hidden count, perf TLE as one bit, CE with learner-file diagnostics only (the real-runner lane, green on Linux).
- [ ] **The internal watermark and evaluations endpoints answer per T4 §3.2** — for `course` and `touch`; contract tests green against the fixtures and the practice-side fake.
- [ ] judge imports m3-02's `internal/packspec/gen`, `internal/platform/checker` and `constraints[]` validator and m3-04's `internal/platform/harness` and `runnerapi/lint` — no second generator, checker, codec or lint copy under `internal/judge`; the verdict tests reuse m3-04's term→class fixtures.
- [ ] A lint violation yields `failed(REJECTED)` without a runner call; a `dev` runner is refused (lane off, ERROR) unless `DEV_AUTH` is set; the TL sent to the runner is `time_ms × tl_multiplier` and is recorded in versions; row 5 refuses with `ErrNothingToEvaluate` (never in mock).
- [ ] The fake-runner e2e proves INV-1 (a killed worker still yields exactly one evaluation), INV-4, INV-12, INV-13 and INV-14; stale fenced writes are dropped.
- [ ] `evaluation_completed` is written only for course and touch evaluations, via the outbox at 1 s; the golden payload is stable and under the envelope cap.
- [ ] The retry table and the `throttled` path are tested; the sweeper re-queues expired leases, deletes Run payloads older than 24 h and drafts idle for 90 days.
- [ ] judge registers no learner route; the registry and golden tests, `sqlc diff` and every v1 test and e2e are green.

## Release

**Merge only — ships dark in v1.13.0**, cut by [m3-07](sprint-m3-07.md) once [m3-14](sprint-m3-14.md) has merged. m3-07
also carries this sprint's hand-offs into the MI-13 HelmRelease (runner env and token, the scratch `emptyDir`,
`RELAY_INTERVAL`). No new stream, durable or in-cluster caller is added here beyond what m3-07's ACL and NetworkPolicy PRs
already cover (judge → runner, practice → judge).

## Definition of Done

CI green (`go`, `e2e`, `nats-acl`, `judge-runner-e2e`, `sqlc diff`, the migration lint) · squash-merged to `main`, **no
tag** · docs + ADR written · hand-offs recorded · statuses updated (this file + [`../status.md`](../status.md): Sprint board,
M3 🔄, decisions log).

## Risks / watch-outs

- **Queue timings tuned on a laptop** (lease, heartbeat, backoff, the 30-min bound): revisit with prod telemetry (TR-QUEUE;
  m3-14 adds the telemetry rows).
- **A producer before its consumer:** `evaluation_completed` has no consumer until v1.14.0. Keep judge unreachable from
  learners (no route here; the gateway's `JUDGE_BASE_URL` unset) — do not add a "temporary" test route.
- **Hidden data leaking** through diagnostics, signals or `Present()`: the fixed CE text, the learner-file filter, generic
  check keys and count-only hidden results; m3-09's denylist test is the backstop.
- **Generator or checker drift** silently marks correct code WA: one registry each (m3-02's `internal/packspec/gen` and
  `internal/platform/checker`, imported by packlint and judge alike), a golden hash per `gen@v`, a new `@v` for every change, the
  version on every evaluation. A judge-local copy is the failure mode — never add one.
- **A `dev` runner in production** would grade with security canaries only warning: judge refuses `mode=dev` unless
  `DEV_AUTH` is set, and m3-07 never sets `DEV_AUTH` on judge.
- **The claim SQL under READ COMMITTED:** the advisory xact lock plus `NOT EXISTS` is what serializes per-account claims;
  the `-race` concurrency test must use real Postgres, not a fake.
- **The runner can't run on macOS**; the real-runner lane is Linux-only (CI or the multipass VM).
- **Memory (256 Mi):** stream cases and compares, never buffer a whole case file, 2 workers only.
- **Read-only root FS in prod:** the scratch `emptyDir` hand-off must reach m3-07.
- **Scope:** this is a heavy session; the quiz key modes are the pre-agreed split point if needed.
