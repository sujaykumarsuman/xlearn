# Prompt — Sprint m3-06 · judge queue, runner lane, graders, contexts, internal context endpoints

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-06.md`](../sprints/sprint-m3-06.md)   ·   **Milestone:** M3 (M3-1, judge dark)   ·   **Prereqs:** [m3-05](../sprints/sprint-m3-05.md) · [m3-03](../sprints/sprint-m3-03.md) · [m3-04](../sprints/sprint-m3-04.md) (profiles, harness, lint, `tl_multiplier`) · [m3-02](../sprints/sprint-m3-02.md) (generators, checkers, validator)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m3-06.md`](../sprints/sprint-m3-06.md) — the claim SQL, the retry table, the grader rules, the
  submit-core steps, the result-transaction order, the endpoint contracts and the hand-offs live there. Follow them exactly.
- **What earlier sprints hand this one by name — import, never re-implement:**
  [m3-02](../sprints/sprint-m3-02.md) tasks 2–4 (`internal/packspec` case lines + the `constraints[]` validator,
  `internal/packspec/gen`, `internal/platform/checker`); [m3-03](../sprints/sprint-m3-03.md)'s hand-off line (`runnerapi` fixtures,
  `ValidateResult`, judge infers `runner_oom`, the 45 s cap → TLE, **refuse a `dev` runner in production**);
  [m3-04](../sprints/sprint-m3-04.md)'s hand-off line (`harness.Generate`/`EncodeInput`/`DecodeOutput`,
  `CanonicalizableOutput`, `lint.Check`, `tl_multiplier`, the term→class fixtures, the fixed CE text);
  [m3-05](../sprints/sprint-m3-05.md)'s dev content overlay (`FIXTURE_CONTENT_DIR` + `DEV_AUTH`).
- The judge contract: [t4 §2](../research/t4-judge-contract.md#2-the-common-contract) (vocabulary, submission, admission,
  closes, job, evaluation, transport, events, invariants INV-1…INV-14), [§3.2](../research/t4-judge-contract.md#32-practice-consumer-and-ticker)
  (what practice's reconciler and ticker call), [§4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (code verdicts,
  perf specs), [§4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key) (key modes), [§4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context),
  [§5.1–§5.3](../research/t4-judge-contract.md#52-grader-contract-go) (registries, grader contract, composite),
  [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (signals), [§9](../research/t4-judge-contract.md#9-queue-admission-and-single-node-budget)
  (lanes, priorities, sweeper), [§11.1](../research/t4-judge-contract.md#111-t3-the-runner-contract-not-the-technology),
  [§11.4 #11](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag). **t4 §13's owner decisions override
  the body:** D16 (no re-implement, so no child context), D17 (arena unrestricted), D14 (AI is M4).
- The runner side: [t3 §5.3](../research/t3-sandbox.md#53-api-judge-is-the-only-caller) (API, profiles, Result validation),
  [§5.7](../research/t3-sandbox.md#57-throttled-and-the-quiet-re-run-learner-proof-bounded) (`throttled`),
  [§5.9](../research/t3-sandbox.md#59-typed-infra-errors-and-poison-pills) (typed infra errors, poison pills);
  [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (still Proposed until m3-03 accepts it — read m3-03's
  accepted text on `main`).
- Content: [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (item format, checker registry),
  [t1 §6.3](../research/t1-content-data-model.md#63-caps-and-quotas-enforced-by-judge-t3t4-own-rate-limiting) (part caps;
  [t4 §5.1](../research/t4-judge-contract.md#51-registries) places the part registry at `internal/platform/item/parts`),
  [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning) (`contract_hash` at submit).
- ADRs: [0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract), [0027 §4–§5](../../adr/0027-content-evalpack-and-user-data-model.md#4-problems-arena),
  [0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (consumers before
  producers), [0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes)
  (topology, envelope cap), [0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L14 runner caps),
  [0005](../../adr/0005-data-ownership-and-migrations.md), [0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (internal endpoints).
- Rollout: [§4 M3](../rollout-plan.md#4-per-milestone-detail), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag).
- Code: `internal/judge/**` (m3-05: config, store, pack, `store.IsErased`, `JUDGE_ADMISSION`, the overlay wire), `internal/platform/judgeapi`,
  `internal/platform/runnerapi` (m3-03: `ValidateResult`, `testdata/` fixtures), **`internal/platform/harness`** and
  **`internal/platform/runnerapi/lint`** (m3-04; the term→class fixtures), **`internal/packspec`**, **`internal/packspec/gen`**
  and **`internal/platform/checker`** (m3-02), `internal/platform/item/parts` (if present), `internal/course`'s overlay helper and
  `internal/judge/testdata/content` (m3-05/m3-02), the runner's Linux CI job and compose profile (m3-03/m3-04/m3-15),
  `internal/course/keys` (m2-01), `internal/course` + `canon`, `internal/judge/testdata/pack`,
  `internal/platform/events/{topology.go,relay.go,consumer.go}`, `internal/e2e/`, `docker-compose.yml`, `.github/workflows/ci.yml`.

## Context

m3-05 stood judge up dark: schema, a pack loader that is Ready with zero on any pack problem, `/internal/evaluable`, the erase
consumer, `aud=judge`, the `JUDGE_ADMISSION` kill switch. This sprint builds the **evaluation core** on it: the Postgres
`SKIP LOCKED` queue with one runner lane of 2 workers and fenced claims; the runner client with T3's typed retry classes and
the single 5-minute `throttled` re-queue; the closed grader registry (`code@1` with typed checker verdicts and count-only
hidden feedback, `key@1` inline on the shared `internal/course/keys` evaluator, the `composite@1` root); the course, touch,
mock and arena contexts plus the `run` action; `evaluation_completed` for counted course and touch evaluations; and the two
read-only internal endpoints practice's reconciler and its touch and course deadline drains call. **There is still no learner route** — m3-14 wraps
this core with admission, idempotency semantics and the DTO allowlist, and m3-07 ships it all dark in v1.13.0.
`evaluation_completed` has no consumer until v1.14.0 (m3-08), which is safe only because nothing can reach judge yet.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m3-05 merged (judge skeleton, schema, pack loader, `judgeapi`, `IsErased`, `JUDGE_ADMISSION`, the dev content overlay on `main`).
- [ ] m3-03 merged: `internal/platform/runnerapi` with the T3 amendments and `ValidateResult`; note the exact job endpoint path and that `GET /v1/profiles` reports `mode`.
- [ ] m3-04 merged: `go`, `cpp`, `python` profiles (each with `tl_multiplier`), **`internal/platform/harness`** (`Generate`, `EncodeInput`, `DecodeOutput`, `CanonicalizableOutput`, `func-json@1` / `class-ops@1`), **`internal/platform/runnerapi/lint`**, the term→class fixtures (the acceptance needs all three languages).
- [ ] On `main`: `internal/course/keys`, `internal/course/canon`, the fixture pack, and m3-02's **`internal/packspec/gen`**, **`internal/platform/checker`** and the `internal/packspec` `constraints[]` validator.
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no peer editing `internal/judge/`, `topology.go`, judge migrations, `internal/packspec/gen`, `internal/platform/checker` or `internal/platform/item/parts`.

## Do this (in order)

1. **[X] Branch** `feat/m3-06-judge-core` from an up-to-date `main`.
2. **[X] Queue** (plan task 1): `internal/judge/queue` — enqueue in the caller's tx with P0–P2 priorities; the claim tx
   (ageing order, `FOR UPDATE SKIP LOCKED`, `pg_try_advisory_xact_lock(hashtext(account_id::text || lane))`, `NOT EXISTS`
   running for the account and lane, `claim_token++`, versions at claim); 2 workers polling at 1 s; the 10 s heartbeat that
   cancels on lease loss; the fenced result write; the 3-expiry re-claim cap; Run cancellation. Any new index or column goes
   in a new judge migration (next free goose version).
3. **[X] Runner client** (task 2): `internal/judge/runner` — `RUNNER_BASE_URL` (unset = lane off), `RUNNER_TOKEN_FILE` /
   `RUNNER_TOKEN`, the streamed length-prefixed call with caps enforced before sending, `GET /v1/profiles` every 5 min
   (per-profile `tl_multiplier`; **`mode=dev` refused — lane off + ERROR `runner_dev_mode` — unless `DEV_AUTH` is set**),
   `runnerapi.ValidateResult`, `runner_oom` inferred from a dropped connection + a changed `BootEpoch`, and the plan's retry table (setup ≤ 1; poison ×2 → `runner_unavailable`; `killed` = `saturated`;
   saturated backoff with the 30-min liveness bound; 401/403 ×3 → `runner_unavailable`; `throttled` once at +5 min →
   `runner_throttled`; 400 → `error(internal)`).
4. **[X] Case sources** (task 3): samples from the public item; `OpenCases` lines decoded with `internal/packspec`'s case
   type; seeded perf lines streamed from **m3-02's `internal/packspec/gen`** (`int_array@1`, `string@1`, `permutation@1`,
   `tree@1`, `graph@1`, `op_sequence@1` — import it; a new generator is a new `@v` there, never a judge copy); per-job files
   under `JUDGE_SCRATCH_DIR`, deleted after the job; expected outputs stay in judge; `OutputMode: sha256` only when
   `harness.CanonicalizableOutput`.
5. **[X] Graders** (task 4): the closed registry and the invariant wrapper; `code@1` (job build with `harness.Generate` /
   `EncodeInput` / `DecodeOutput` and TL = `time_ms × tl_multiplier`; the seven checkers **from m3-02's `internal/platform/checker`**;
   the production term→class mapper tested on m3-04's fixtures; verdict precedence, count-only hidden feedback + one perf
   bit + first-failure class, learner-file diagnostics, the fixed CE text, generic check keys, the code-family signals); `key@1` (counted contexts only, `final` locks once, `internal/course/keys`
   for text/big-O, the quiz modes, honor vs checked, misconception signals); `composite@1` (status order, score, provenance).
   Keep DSA touch probe lock-ins in practice.
6. **[X] Submit core** (task 5): `Service.Submit` (admission + evaluable check, arena `uuidv5(account, item)`, part
   validation against `internal/platform/item/parts` — create it with the t1 §6.3 caps if absent — then **`lint.Check`
   before any enqueue → an inline `failed(REJECTED)` evaluation**, idempotency errors, closes, `context_counter` seq,
   pin-mismatch → `inconclusive(contract_changed)`, steps per context with **row 5's `ErrNothingToEvaluate`** (not in
   mock), inline key-only vs job, **the outbox row iff course/touch**), `Service.Run` (custom input checked with
   `internal/packspec`'s `constraints[]` validator; lint → `REJECTED` without a runner call), `GetSubmission`, `GetRun` —
   all `sub`-scoped, **no HTTP learner route**.
7. **[X] Result tx + event** (task 6): the seven-step fenced tx (tombstone, `FOR UPDATE` erase guard, evaluation + versions,
   release rule, the `onArenaPass` no-op hook, the outbox row for course/touch only); the payload + the shared golden
   `internal/platform/events/testdata/evaluation_completed.v2.json`; `RELAY_INTERVAL` (default 1 s); `topology.go` Emits
   `evaluation_completed`; re-render the goldens (expect no ACL diff) and keep the registry test green.
8. **[X] Sweeper** (task 7): one ticker for lease re-queue, Run payload deletion after 24 h, 90-day draft deletion in
   batches, scratch cleanup; stopped before the pool closes.
9. **[X] Internal endpoints** (task 8): `GET /internal/contexts/{kind}/{id}/evaluations?seq_lt=T` (course/touch only, facts in
   the `evaluation_completed` shape) and **`GET /internal/contexts/{kind}/{id}/watermark`** (`kind ∈ course|touch`,
   `{max_seq, open_jobs}`), read-only; `judgeapi.Client` (`Watermark(kind, id)`), the fixtures (a course and a touch case),
   the exported practice-side fake, the contract test on real PG. Tell m3-08 the course watermark exists.
10. **[X] Tests** (task 9): the unit/property set (claim concurrency on real PG under `-race`, fencing, lease expiry, retry
    table, throttled, checkers, precedence, composite, key modes, INV-1/4/12/13/14, payload golden, route table); the
    fake-runner e2e (`internal/judge/runnerfake`); the real-runner lane (compose profile `runner`, `privileged` local only,
    `RUNNER_MODE=dev` accepted because judge has `DEV_AUTH`; `internal/e2e/judge_runner_test.go` with `-tags e2e,runner`,
    judge's content loaded **through m3-05's dev overlay** so fx-001..fx-003 are `ok`; synthetic solutions per language) and
    the CI job `judge-runner-e2e` mirroring m3-03/m3-04's Linux job.
11. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e -race ./internal/e2e/...`, the
    real-runner lane on Linux (CI, or the multipass VM from the spike), `sqlc diff`, the migration lint, `make nats-acl-test`,
    web tests (unchanged).
12. **[X] Docs + ADR + hand-offs** (task 10): `docs/architecture/{events,services,data-model,api}.md`; ADR "judge evaluation
    core (M3)" after checking peers for the next free number; the m3-07 / m3-08 / m3-14 hand-offs in the PR description
    and the status.md decisions log — especially m3-07's **scratch `emptyDir`** and the runner env names.
13. **[X] Ship** per AGENT.md land-and-sync with this sprint's release action (below).

## Constraints

- **Service boundaries (ADR-0005):** judge writes only schema `judge`; it never computes grades and never writes to
  practice, review or assessment (INV-2). practice reads judge only through the internal endpoints; shared wire types live
  in `internal/platform/judgeapi` and `internal/platform/runnerapi`, never in `internal/judge`.
- **goose + sqlc:** embedded migrations under the advisory lock, next free judge version, generated code committed,
  `sqlc diff` clean, expand only. Never run `Down` in production.
- **Outbox:** the evaluation and its `evaluation_completed` row in one tx; no HTTP inside a tx (the runner call happens
  outside the result tx); stop the workers, the sweeper and the relay before `srv.Shutdown()` and before the pool closes.
- **Consumers before producers (ADR-0034 §3):** no learner-reachable route in judge, no gateway change, no temporary test
  route — the producer stays dark behind the gateway's unset `JUDGE_BASE_URL` until v1.14.0's consumer exists.
- **Leak rules:** expected outputs never reach the runner (honor in-process profiles excepted, P); hidden results are
  count + perf bit + class; no case ids, ordinals, timings, stderr or test names leave judge; diagnostics at learner-file
  positions only.
- **Closed registries, one definition each:** graders, checkers, normalizers and generators are explicit lists in one file
  each — no `init()` self-registration, plugins or WASM. Checkers (`internal/platform/checker`), generators
  (`internal/packspec/gen`), the harness codec (`internal/platform/harness`) and the source lint (`runnerapi/lint`) are
  **imported** from the sprints that own them; extend them in place, never copy them under `internal/judge`.
- **No `dev` runner in production:** accepted only with `DEV_AUTH` (compose/CI); m3-07 never sets `DEV_AUTH` on judge.
- **Public repo hygiene:** synthetic fixtures and solutions only; never copy from `../xlearn-evalpack`.
- **GitOps:** no infra PR in this sprint (m3-07 owns them); never `kubectl apply`. The compose `runner` profile is
  **local only** (`privileged`), never a production pattern.
- **ACL standing rule (ADR-0035 §2):** this sprint adds no stream or durable; if the golden render changes anyway, say so in
  the PR so m3-07's ACL PR carries it **before** v1.13.0. judge → runner and practice → judge are already covered by
  m3-07's NetworkPolicy PR.
- **D34:** no alerting; "quarantine" and saturation are ERROR logs (telemetry rows arrive in m3-14).
- **Memory-sum rule (ADR-0035 §5):** judge stays inside its 256 Mi limit — stream inputs and compares, 2 workers only.
- **Parallel sessions:** check peers' PRs, tags and worktrees before claiming the ADR number and before merging shared files.

## Deliverables

- `internal/judge/{queue,runner,grader,…}` and `submit.go`, the result tx, the sweeper, the internal endpoints; any new
  judge migration + sqlc output; `internal/platform/item/parts` if it wasn't on `main` (no generator, checker, codec or lint
  copy — those are imported).
- `internal/platform/judgeapi` (`EvaluationFact`, `Watermark`, `Client`, fixtures, the practice-side fake).
- `internal/platform/events/testdata/evaluation_completed.v2.json`; `topology.go` + goldens.
- `internal/judge/runnerfake`; `internal/e2e/judge_runner_test.go`; the compose `runner` profile (if m3-15 didn't add it);
  the `judge-runner-e2e` CI job; synthetic solutions under `internal/judge/testdata/solutions/`.
- Docs, the ADR, the hand-offs.

## Update status

- [`../sprints/sprint-m3-06.md`](../sprints/sprint-m3-06.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M3** milestone row stays 🔄 ("judge core merged, ships dark in
  v1.13.0"); **Decisions log** lines for: the claim SQL / fencing, the retry table incl. the 401 rule and the 30-min
  liveness bound, the shared packages imported (`internal/packspec/gen` — a move from t4 §5.1's `internal/judge/gen`, one
  registry — `internal/platform/checker`, `internal/platform/harness`, `runnerapi/lint`, `internal/platform/item/parts`), the
  `dev`-runner refusal, `tl_multiplier` in versions, the `{kind}` watermark (m3-08 hand-off), `key@1` on
  `internal/course/keys` with DSA probe lock-ins kept in practice, mock silent in v2.0, `RELAY_INTERVAL` 1 s,
  **the m3-07 hand-off (scratch `emptyDir`, `RUNNER_BASE_URL`, `RUNNER_TOKEN_FILE`)**, the ADR number, and the quiz-mode
  split if you had to make it.

## Done when (acceptance)

- [ ] Compose e2e: submit → runner → evaluation for Go, C++ and Python fixtures (AC, WA with a hidden count, perf TLE as one bit, CE with learner-file diagnostics only) — the real-runner lane is green on Linux.
- [ ] The internal watermark (course and touch) and evaluations endpoints answer per T4 §3.2 — contract tests green against the fixtures and the practice-side fake.
- [ ] No second generator, checker, codec or lint under `internal/judge`; lint violations → `failed(REJECTED)` without a runner call; a `dev` runner is refused unless `DEV_AUTH`; TL = `time_ms × tl_multiplier`; row 5 → `ErrNothingToEvaluate` (never in mock).
- [ ] The fake-runner e2e proves INV-1, INV-4, INV-12, INV-13 and INV-14; stale fenced writes are dropped.
- [ ] `evaluation_completed` only for course and touch, via the outbox at 1 s; the golden payload is stable and under the envelope cap.
- [ ] The retry table and `throttled` path are tested; the sweeper re-queues expired leases and deletes old Run payloads and idle drafts.
- [ ] No learner route in judge; registry and golden tests, `sqlc diff` and every v1 test and e2e are green.

**Shipping:** per AGENT.md land-and-sync with **this sprint's release action — merge only (ships dark in v1.13.0)**:
branch → conventional commits with the attribution lines → push → PR → CI green → squash-merge → `git checkout main && git pull`.
**Do not tag** (m3-07 cuts v1.13.0 after m3-14) and open **no infra PR** (m3-07 carries the hand-offs); nothing deploys yet.
