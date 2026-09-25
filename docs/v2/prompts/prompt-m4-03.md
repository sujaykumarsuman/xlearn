# Prompt — Sprint m4-03 · Analyzer (D16/D26), pointer notes, `evaluation_analyzed` + acceptance-set harness

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-03.md`](../sprints/sprint-m4-03.md)   ·   **Milestone:** M4 (platform AI)   ·   **Prereqs:** [m4-02](../sprints/sprint-m4-02.md) (judge ai) · [m3-10](../sprints/sprint-m3-10.md) (review on judge signals)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Optional here (only step 15's dev-split run needs it; [m4-07](../sprints/sprint-m4-07.md) needs it anyway): a fresh `xlearn-calib` personal key (7–30-day expiry, from the Anthropic Console) exported as `LLM_CALIB_API_KEY` in the shell that launches this session — never pasted into the session.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m4-03.md`](../sprints/sprint-m4-03.md) — tables, the eligibility table, the event shape and the harness report live there.
- Decisions: [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §1 (judge owns platform AI), §4 (retention, consents,
  pack material only in `Score`), §5 (spend), **§6 (D26 pass-review scope)**; [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md)
  §3–§5 (D16: the analyzer reviews passes; mistake pre-fill precedence); PRD [§5.6](../../prd/xlearn-v2-prd.md) R-AI3, R-AI6.
- Research: [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used) (task table, calibration workspace),
  [§4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (analyzer model/effort, acceptance thresholds,
  portable schema subset), [§6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (tiers, pace gate, skip
  rules, one analysis per attempt), [§7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)
  (`pass_review` schema, Go validator, injection defences, canary), [§8](../research/t5-platform-ai.md#8-privacy-and-residency)
  (pointer notes are C3, withheld during live touches); **t5 §12 (D24–D27) overrides the body.**
  [t4 §2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs) (`evaluation_analyzed`,
  the two durables), [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (precedence,
  `last_refreshed_by_attempt_id`), [§7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5) (`xlearn.code_analysis@1`).
- Ops rules: [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)
  (dark producer behind a T-2 switch), [ADR-0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes)
  (topology, registry test, ACL standing rule), [§4 L17](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory).
- The frozen boards `design-system/screens/v2/AB17-pointer-notes.html` (collapsed notes, "correct, with improvements",
  optional revisit, the unavailable variants) and AB16's F2 in `AB16-ai-suggestion-dispute.html` (the analyzer suggestion on a
  deterministic grade) — their frames define what the read endpoints must return.
- Code: `internal/judge/{consumers.go,service.go,handlers.go,store/}` and `internal/judge/ai/` (m4-02),
  `internal/platform/llm/` (m4-01), `internal/platform/events/{topology.go,consumer.go}` and
  `testdata/nats-authorization.golden.conf`, `internal/review/{service.go,handlers.go,consumers.go,store/store.go,store/mistakes.go,store/queries/*.sql,store/migrations/}`,
  `internal/gateway/{judge.go,review.go,coach.go,public_test.go,openapi_drift_test.go}`, `docs/architecture/openapi.yaml`,
  `internal/e2e/`, `docker-compose.yml`, `Makefile`; infra: `../infra/infrastructure/messaging/release.yaml`;
  the evalpack template `xlearn-evalpack/acceptance/schema/label.schema.json` (read-only here).

## Context

M3 made judge the evidence engine (v1.13.0/v1.14.0), P added the pilot course (v1.15.0), and m4-01/m4-02 gave judge a
shared LLM layer, WIF auth, the Scorer, the money ledger, the llm lane, L17 caps and a breaker — all behind
`LLM_PLATFORM_ENABLED=false` and the owner/tester cohort. This sprint adds the **analyzer**: judge's own `DeliverNew`
durables on `problem_solved` and `touch_concluded` decide whether to analyze (below-clean conclusions and failed code
touches → **diagnose**, counted tier; every course pass and **materially different** touch passes → **pass review**,
advisory tier, D26), feed m4-02's `Scorer.Analyze` (one Sonnet 5 call), check the result against the submission, store pointer notes per (account, item), and publish
`evaluation_analyzed` so review pre-fills the mistake category/concepts and marks revision items "correct, with
improvements". It also builds `cmd/judge-eval`, which m4-07 runs on the owner's frozen test split to size caps. Nothing
calls the model on prod yet: the flag is off, no AI consent row exists until m4-05, and no acceptance row until m4-07
(all fail closed). Everything ships in
**v1.16.0** (m4-07 tags).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m4-02 merged on `main`: `internal/judge/ai` Scorer + `llm_call` ledger + llm lane + caps/breaker + account gate;
      the consent kinds + gate, the acceptance-row catalog gate and `judge admin calibration record`; `httptest` fake providers
      in tests (m4-01's convention); identity `/internal/accounts/{id}` returns `role`, `tier`, `status`, `consents`
      (`gh pr list --state merged --search "m4-02"`; read the code).
- [ ] AB17 frozen and AB16 on `main`: ds-m4-01 merged, and the merge is the freeze (`design-system/screens/v2/AB17-pointer-notes.html`,
      `AB16-ai-suggestion-dispute.html`); note whether the merged AB17 keeps F3 (optional revisit) — if not, skip `optional_revisit` and its routes.
- [ ] review `mistake_entry.{category_source, category_suggested, concepts, concepts_source, last_refreshed_by_attempt_id}`
      exist (m3-10; `internal/review/store/migrations/`), and v1.14.0 is live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`).
- [ ] `problem_solved` v2 and `touch_concluded` carry `attempt_id` (+ `revision_item_id`, `touch_passed`, `evaluation_ids[]`
      on the touch): check the fixtures in `internal/platform/events/testdata/`.
- [ ] The acceptance-set template is merged in `xlearn-evalpack` (`acceptance/schema/label.schema.json`).
- [ ] Parallel sessions: `gh pr list` (xlearn, `../infra`), `git worktree list` and ListAgents show no peer editing judge or
      review migrations, `topology.go`/the ACL golden, or `infrastructure/messaging/release.yaml` right now.

## Do this (in order)

1. **[X] Branch** `feat/m4-03-analyzer` from an up-to-date `main`.
2. **[X] Migrations** (plan task 1): judge `analysis` (UNIQUE `attempt_id`, validated `body jsonb`), `pointer_note` (PK
   account+item), `review_fingerprint`; review `revision_item.improvements_at` + `improvements_analysis_id` and
   `optional_revisit` (off-ladder). Next free goose version per service; prose tables PK/FK only (no CHECKs); `sqlc generate`.
3. **[X] Consent + model gates** (read, don't rebuild; the one additive change is step 6's `Admission.PassReview`): m4-02's
   `internal/platform/consent` kinds and its gate inside `Reserve` (consent per purpose); m4-02's catalog resolves effort/`max_tokens` only from a passed acceptance row (none ⇒
   `no_acceptance`, no call). Add `ANALYZER_FP_THRESHOLD` (judge env, default 0.30) to judge's config.
4. **[X] `internal/judge/ai/fingerprint`** (task 2): lexers (Go `go/scanner`; C++/Python hand lexers), normalization,
   5-token shingles + winnowing (w=4), Jaccard, `fp@1`, τ = 0.30 provisional; golden tests + a fuzz target.
5. **[X] Analyzer durables** (task 3): `judge-analyzer-solved` / `judge-analyzer-touch` with `WithDeliverNew()`; gate read
   before the tx (cached client); in one tx: inbox, tombstone, evidence lookup, the eligibility table, skip rules, insert
   `analysis` (+ `job` kind `analyze`, lane llm, P3) `ON CONFLICT (attempt_id) DO NOTHING`. When the llm lane is at its
   L10 cap (8 queued), write `skipped(queue_full)` with no job (decided at enqueue). A test asserts both durables are
   `DeliverNew`.
6. **[X] Analyze jobs** (task 4): `Reserve(Admission{AccountID, Context, Purpose: analyze, Tier, EvaluationID, JobID,
   PassReview})`, with m4-02's field names. `PassReview` is new and additive (true for `pass_review`/`both`). m4-02's gate
   now keys on it: graded consent is always required, passing consent only when `PassReview` is true. The refusal names
   the missing kind. At claim, a `both` job refused only for passing consent downgrades to `diagnose` and runs. Table-test
   rows: a rough pass with graded-only consent; passing consent withdrawn between enqueue and claim. Then the inputs to
   m4-02's `Scorer.Analyze`
   (public statement + public solution, DTO-level verdict, attempt facts, category/concept enums; extend `AnalyzeRequest`
   additively if a field is missing; it must not be able to import `PackView` — add the contract test); the analyzer-side
   checks (line ranges inside the submission, ids exist, category only at ≥ medium); the fenced result write that
   replaces `pointer_note`, upserts `review_fingerprint` and writes the `evaluation_analyzed` outbox row;
   `judge admin analyses`.
7. **[X] Event + review consumer** (task 5): the payload and `testdata/evaluation_analyzed.v2.json`; review's
   `review-analysis` durable and `HandleEvaluationAnalyzed`. It has two independent parts:
   - the **mistake pre-fill** (precedence, concepts), only for `diagnose`/`both` events that carry a category or concepts,
     with the 10-minute retryable window when no mistake matches yet, then a stale skip;
   - **`improvements_at`**, keyed on `revision_item_id` + account, applied at once whether or not a mistake matches.
     Pass-review events never look up a mistake.

   Test: a touch-pass event with no mistake entry sets `improvements_at` on the first delivery.
8. **[X] Reads + erase** (task 6):
   - judge `GET /items/{item_id}/pointer-notes` (its upstream JSON carries the note's `analysis_id`) and
     `GET /attempts/{attempt_id}/analysis`;
   - gateway `GET /api/problems/{id}/pointer-notes` and `GET /api/attempts/{id}/analysis`, with presence-by-config and
     `withhold()` (`{state:"withheld"}`);
   - review `GET /revisions/due` gains `improvements` + `optional[]`, served on the shared
     `GET /api/paths/{slug}/revision/due` handler (so `/api/revision/due` too);
   - the optional-revisit routes: review `POST|DELETE /problems/{id}/optional-revisit` and
     `POST /problems/{id}/optional-revisit/done` (`internal/review/optional_revisit.go`,
     `store/queries/optional_revisit.sql`), fronted by `internal/gateway/review.go`. The gateway checks judge's
     `revisit_suggested` and passes `analysis_id`, else 409 `no_revisit_suggested`;
   - the coach's `review`-mode context gets the pointer notes plus the done analysis's `summary`/`next_step`, never when
     withheld. m1-07's final-submission slot stays unowned: report it, don't build it;
   - public denylist additions; judge's and review's erase consumers cover the new tables; `openapi.yaml`.
9. **[X] Harness** (task 7): `cmd/judge-eval` + `internal/judge/ai/eval` (metrics, bootstrap LB, per-category P/R,
   τ sweep; `configs_tried.log`, appended and checked **only** on `--split test --provider anthropic`: dev and fake runs
   never write it, tests use a `t.TempDir()` copy, and a test proves both; the repeat-config refusal;
   `--calibration-out` writing an `llm_calibration` row for
   `judge admin calibration record`; the `llm.Cred` built inside `internal/judge/ai/eval` so m4-01's boundary lints stay green), the synthetic set, `make judge-eval`, and `go test ./cmd/judge-eval/...` with the fake provider.
10. **[X] Topology + ACL golden** (task 8): the three durables, `XLEARN_JUDGE.Emits`, registry ignore lists; re-render the
    golden (`-update`); run `make nats-acl-test`.
11. **[X] e2e** (task 8): the five scenarios in the plan (clean pass → notes; miss → analyzer category; renamed touch pass →
    `fingerprint_same`; different improvable touch pass → `improvements_at` on the first delivery; flag off → nothing), with judge pointed at an
    `httptest` fake provider and consent rows seeded by SQL fixture. For a manual look, use a `-f docker-compose.yml -f
    <override>` in the session scratchpad (never committed, m4-01's convention).
12. **[X] Docs + ADR:** `docs/architecture/{events,data-model,services,api}.md`; the ADR "Analyzer and pass review (M4
    realization)" — check peers' PRs/worktrees for ADR numbers first, then take the next free number (MADR, append-only).
13. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG integration tests via
    `XLEARN_TEST_DATABASE_URL` for judge and review), `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, the
    migration lint, `make nats-acl-test`, the openapi drift test, web tests (unchanged).
14. **[I] Infra ACL PR** (task 9) in `../infra`: paste the re-rendered judge and review blocks into
    `infrastructure/messaging/release.yaml`; state "no NetworkPolicy change: no new in-cluster HTTP caller"; merge before
    v1.16.0 (a reload, not a restart). After it reconciles, read-only check the NATS pod logs for config-reload errors.
15. **[X] Dev-split tuning** (task 10, optional here): verify the before-launch key is set without printing it
    (`[ -n "$LLM_CALIB_API_KEY" ] && echo set || echo missing`). If set, run
    `make judge-eval SET=../xlearn-evalpack/acceptance SPLIT=dev PROVIDER=anthropic SWEEP=effort:low-nothink,low,medium`
    yourself (pre-approved by launching this prompt, D40; billed to `xlearn-calib` within the ≈ $10–30 acceptance budget;
    dev runs never write `configs_tried.log`) and record the results (effort, `max_tokens`, τ) in status.md. If missing,
    set task 10 ⛔ "no calib key in the session env: runs at m4-07 start" in status.md and carry on.
16. **[X] Ship:** see **Ship** below.

## Constraints

- **Service boundaries (ADR-0005):** judge writes only schema `judge`, review only `review`. Judge never writes review or
  assessment (INV-2); review learns about analyses only from the event. The account gate and consents are read over
  identity's internal API, never its tables. No HTTP inside a transaction.
- **goose + sqlc:** migrations embedded, run on startup under the advisory lock, next free version, expand only,
  generated code committed, `sqlc diff` clean. Never run `Down` in production.
- **Outbox / inbox:** the analysis result, the pointer note and the `evaluation_analyzed` outbox row commit together in
  the fenced result transaction; consumers claim the inbox in the same tx as their writes; `NakWithDelay` backoff; stop
  consumers before `srv.Shutdown()`.
- **Consumers before producers / ACL before tag:** the producer ships **dark** behind `LLM_PLATFORM_ENABLED=false`
  (T-2); the infra ACL PR merges **before** v1.16.0. The durables are `DeliverNew`.
- **Money and privacy:** no LLM call outside `Reserve`; no pack material in `Analyze` (compile-time); no `tools`; no prose
  in events, logs, errors or the harness output; pointer notes never on public routes; `LLM_CALIB_API_KEY` stays on the
  owner's machine, exported by the owner before launch — never read, echoed or stored by the agent (only the harness reads
  it from the environment); never set `ANTHROPIC_API_KEY`.
- **Consent is fail-closed:** no consent row ⇒ no analysis. Never default a consent to granted.
- **GitOps:** infra changes only through `../infra` PRs, each its own task, never folded into a tag; never `kubectl apply`.
- **D34:** no alerting, digests, pings, Flux Alerts or opscheck; `judge admin analyses` is the on-demand read.
- **Memory-sum rule (ADR-0035 §5):** no new pod; judge keeps its 256 Mi limit.
- **Parallel sessions:** check peers' PRs, tags and worktrees before claiming the ADR number and before merging
  migrations; rebase and take the next free goose version if a peer merged first.

## Deliverables

- judge: the migration; `internal/judge/ai/{analyzer.go,fingerprint/,eval/}`; `Admission.PassReview` + the gate keying on it;
  the two durables; the result write + outbox;
  `GET /items/{item_id}/pointer-notes`, `GET /attempts/{attempt_id}/analysis`; erase coverage; `judge admin analyses`; tests
  (unit, fuzz, real-PG, `-race`).
- review: the `revision_item` + `optional_revisit` migration; `HandleEvaluationAnalyzed` + the `review-analysis` durable;
  `improvements` and `optional[]` on `GET /revisions/due`; the optional-revisit routes (`optional_revisit.go`,
  `store/queries/optional_revisit.sql`); erase coverage; tests.
- gateway: `GET /api/problems/{id}/pointer-notes`, `GET /api/attempts/{id}/analysis`, the optional-revisit routes;
  `improvements` + `optional[]` on `GET /api/revision/due`; notes and the analysis `summary`/`next_step` in the coach review
  context; public denylist additions; `openapi.yaml`.
- `cmd/judge-eval` + synthetic set + `make judge-eval`.
- `topology.go` entries, re-rendered ACL golden, `testdata/evaluation_analyzed.v2.json`, e2e cases.
- Docs (`events.md`, `data-model.md`, `services.md`, `api.md`) and the new ADR.
- `../infra` PR: the re-rendered NATS ACL for judge and review.

## Update status

- [`../sprints/sprint-m4-03.md`](../sprints/sprint-m4-03.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when tasks
  1–9 are ✅ (task 10 is optional: ✅ if run, else ⛔ with its note; it never blocks _Overall_).
- [`../status.md`](../status.md): the Sprint board row; the **M4** milestone row stays 🔄 (analyzer merged, ships dark in
  v1.16.0); the infra PR list (ACL PR #); the **flag inventory** unchanged (`LLM_PLATFORM_ENABLED` = false, kill switch);
  the content/acceptance-set row (template → harness ready; dev-split result if run); a **Decisions log** line each for:
  `fp@1` and τ (`ANALYZER_FP_THRESHOLD`), the eligibility table (diagnose vs pass review, tiers), `Admission.PassReview`
  (m4-02's gate keyed on it; `both` → `diagnose` downgrade at claim), `queue_full` decided at enqueue, events only on done
  analyses, `DeliverNew`, pointer notes withheld under `withhold()`, the optional revisit off-ladder, fail-closed consent and
  the acceptance-row gate, and the ADR number.

## Done when (acceptance)

- [ ] e2e (fake provider): a course **pass and a fail are both analyzed**; review shows the analyzer category and concepts on the fail's mistake; the pass has pointer notes via the gateway, and the attempt-analysis read explains every skip.
- [ ] Touch passes are reviewed **only when materially different**; an improvable one marks the revision item "correct, with improvements"; a renamed/reformatted resubmission is skipped and existing notes still show.
- [ ] Every skip rule holds (table test, incl. `queue_full`) with **no `llm_call` row** for any skipped case; a `both` job that lost only passing consent downgrades to `diagnose`; a pass-review event never waits on a mistake entry.
- [ ] With `LLM_PLATFORM_ENABLED=false` nothing is enqueued or emitted; both durables are `DeliverNew`.
- [ ] Analyze requests carry **no pack material** and no tools; no call without a passed acceptance row; validator caps hold; no prose in events or logs.
- [ ] Notes and analyses are withheld under `withhold()` and absent from public routes; an optional revisit never enters the ladder, R-SR5 or the Today budget; erase removes the new rows.
- [ ] `cmd/judge-eval` runs **end to end on the synthetic set** with every metric in the plan and writes a calibration row; a repeated test-split config is refused.
- [ ] Registry, stream-budget, ACL golden and NATS-auth integration tests green; the infra ACL PR merged (before v1.16.0).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: the xlearn PR on `feat/m4-03-analyzer` and the `../infra` ACL PR (infra has no CI: paste the local checks, e.g. `make nats-acl-test` and the golden render, into its PR body and merge on them).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in `v1.16.0`) + the ACL PR:** Nothing deploys; it ships in `v1.16.0` (cut by [m4-07](../sprints/sprint-m4-07.md)). Don't tag, and don't flip `LLM_PLATFORM_ENABLED`. The `../infra` ACL PR (task 9) is merged on its own, before `v1.16.0` (a config reload, not a restart); after it reconciles, read-only check the NATS pod logs for config-reload errors.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
