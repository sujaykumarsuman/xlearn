# Prompt — Sprint m4-07 · Canary log test, caps sizing, ledger check → v1.16.0 (+ LLM_PLATFORM_ENABLED for the cohort)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-07.md`](../sprints/sprint-m4-07.md)   ·   **Milestone:** M4 (exit + tag)   ·   **Prereqs:** [m4-06](../sprints/sprint-m4-06.md) (and m4-01…m4-05) merged, [mi-12](../sprints/sprint-mi-12.md) live, owner events `ev-acceptance-set` and `ev-provider-runbook` done

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and land-and-sync. This sprint **tags**, opens an infra PR and an `xlearn-evalpack` PR.
- The plan: [`../sprints/sprint-m4-07.md`](../sprints/sprint-m4-07.md). Its canary classes, exit-suite cases, gate thresholds, sizing rule, release checklist and rollback are authoritative.
- **M4:** [rollout §3](../rollout-plan.md#3-milestone-map) (M4 exit), [§4 M4](../rollout-plan.md#m4-platform-ai--68-sprints), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.16.0`), [§11](../rollout-plan.md#11-opening-gates-v3) (the data window and the $100/$80 raise belong to the v3 opening).
- **Decisions:**
  - [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md): §2 credential, egress, kill switches; §3 pinned catalog; §4 logging + canary; §5 caps, sizing, degrade order; §6 D26;
  - [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md): §2 kill switches, §3 dark producer, [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) M4 row, [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist);
  - [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md): §2 ACL/NetworkPolicy standing rules, §3 no alerting (provider side only), §4 L17, §5 memory sum.
- **Research:**
  - [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used) (acceptance runs billed to `xlearn-calib`, never the cluster);
  - [§4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (the acceptance gate and effort sweep), [§5](../research/t5-platform-ai.md#5-cost-model) (unit costs, 3.05 analyses/day), [§6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (cap layers, sizing rule, ledger, error mapping, monthly Console reconciliation);
  - [§7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 8 (canary), [§8](../research/t5-platform-ai.md#8-privacy-and-residency) (logging rules).
- **Runbook:** `docs/v2/runbooks/platform-ai-provider.md` (mi-12: workspaces, WIF, kill switches; the "Monthly ledger check" section is yours to add).
- **Code:**
  - `internal/judge/ai` (Scorer, ledger, caps, breaker), `internal/platform/llm` (adapters, `auth`, `prices.go`, usage parsing), `internal/judge/admin` (existing verbs), `cmd/judge-eval` (m4-03's harness);
  - `internal/platform/httpx/httpx.go:65-68` (the panic recoverer logs `rec`);
  - `internal/e2e/` (the in-process `-tags e2e` harness, real Postgres + JetStream; judge on m4-02's `httptest` fake provider, never a committed compose service), `internal/judge/testdata/pack` (m4-04's synthetic rubric item) and m4-04's calibration-row fixture;
  - `../infra/apps/xlearn-judge.yaml`, `../infra/infrastructure/messaging/release.yaml`, `../infra/hack/host-verify.sh`;
  - `../xlearn-evalpack/acceptance/` (private).

## Context

- **What is already merged.** All of M4's code is on `main`, and it ships in **`v1.16.0`**:
  - [m4-01](../sprints/sprint-m4-01.md): `platform/llm`, WIF, `RetentionPolicy`;
  - [m4-02](../sprints/sprint-m4-02.md): Scorer, ledger, llm lane, caps, breaker, and `judge admin ai status | ai-disable | ai-enable | llm-limit | ledger --month | breaker show|set|clear --scope llm | calibration list|record --from <path>|-`;
  - [m4-03](../sprints/sprint-m4-03.md): analyzer, `evaluation_analyzed`, pointer notes, `cmd/judge-eval` (report + `--calibration-out row.json`), `ANALYZER_FP_THRESHOLD`;
  - [m4-04](../sprints/sprint-m4-04.md): provisional, dispute, re-grade, claims, `judge admin disputes export`;
  - [m4-05](../sprints/sprint-m4-05.md): allowance, consents;
  - [m4-06](../sprints/sprint-m4-06.md): the UI.
- **What mi-12 left live.** Judge's 443 + identity egress, the `xlearn-judge-llm` secret, the projected token, `LLM_PLATFORM_ENABLED=false` and ADR-0031 Accepted.
- **The owner has** labelled the acceptance set (≥ 70 examples) and configured the Anthropic Console ($15 hard limit, 50/80% alerts, auto-reload off, WIF, and the separate `xlearn-calib` workspace).
- **This sprint closes M4:**
  - prove it's safe (canary) and inside the budget (acceptance-sized caps, ledger vs Console);
  - tag `v1.16.0` with AI dark;
  - flip `LLM_PLATFORM_ENABLED=true` for the owner/tester cohort in its own infra PR. That merge is **M4 day 1**, which starts the ≥ 2-week data window for `SEAT_CAP`: an opening gate, not GA.
- **Floor and limits:** the floor stays **1.13.0** (no contract). D25's dogfood limits ($15 / $12) hold for all of v2 (D35). There is no alerting anywhere (D34).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `../xlearn-evalpack/acceptance/`: ≥ 70 labelled examples (test ≥ 40 = 15 optimal / 15 suboptimal / 10 failing; dev ≥ 30), schema-valid; the test split's `configs_tried.log` is **empty**.
- [ ] `ev-provider-runbook` ✅ in status.md, and the owner confirms the `xlearn-calib` workspace plus a personal key on their machine. **You never receive that key.**
- [ ] mi-12 live: ADR-0031 Accepted; read-only `ssh vps 'k3s kubectl get networkpolicy -n xlearn xlearn-judge -o yaml'` shows TCP 443 + identity :8081; `xlearn-judge-llm` mounted; `LLM_PLATFORM_ENABLED=false`.
- [ ] m4-01 … m4-06 merged (`git log origin/main`), and m4-01's compose `llm-smoke` output is in its PR. (Its pod smoke can't run before this sprint's tag; it's step 9's first post-tag step.) `git log v1.15.0..main` has nothing that must not ship.
- [ ] m4-03's ACL PR merged in `../infra`.
- [ ] `v1.15.0` live; no open peer PR edits `internal/judge`.
- [ ] Parallel sessions: no peer tag or PR claims `v1.16.0`; no open peer PR edits `../infra/apps/xlearn-judge.yaml` (`git ls-remote --tags origin`, `gh pr list` in xlearn and `../infra`, `git worktree list`, ListAgents). **Find the next free minor** and use it instead of 1.16.0 everywhere if it's taken.

## Do this (in order)

1. **[X] Branch** `feat/m4-07-canary-caps-ledger` from an up-to-date `main`.
2. **[X] Canary tests** (plan task 1).
   - Random canary tokens per class: `PACK`, `CODE`, `TEXT`, `DISPUTE`, `CRED`, `PROVIDER`.
   - **Unit** (`internal/platform/llm/canary_test.go`, `internal/judge/ai/canary_test.go`): a capturing `slog.Handler` at DEBUG plus a fake transport over Score, Feedback, Analyze, the re-grade, the WIF exchange, the breaker and the validator errors. Assert no canary reaches logs, errors, `llm_call`, `ai_sample` after finalize, or outbox payloads. `PACK` appears only in `Score` requests; `DISPUTE` in no request; the salt appears only as the HMAC pseudonym. A panic carrying `CODE` is recovered without echoing it; if `httpx.go` still logs `rec` verbatim, log the type and `[redacted N bytes]` instead.
   - **e2e** (`internal/e2e/m4_canary_test.go`, `-tags e2e`, on the in-process harness; judge on m4-02's `httptest` fake provider with `LLM_PLATFORM_ENABLED=true` and the fake credential env in the harness config only):
     - **setup:** two cohort accounts (owner, tester), each with its own `CODE`/`TEXT`/`DISPUTE` canaries, plus a `learner`; **seed passed `llm_calibration` rows** for `analyze` and m4-04's `score` + `score_regrade` (`judge admin calibration record --from -` in-process, or m4-04's SQL fixture). Without them nothing reaches a provider;
     - **run** the plan's scenario, then collect: logs through a capturing DEBUG `slog.Handler` (set as `slog.SetDefault` and passed to every in-process service), a data-only dump of every schema, outbox rows, the `XLEARN_*` stream messages read from the harness's JetStream, `event_dead_letter`, every BFF response per (route, account), and the `judge admin ai status` / `ledger --month` / `disputes export` outputs;
     - **assert:**
       - no `PACK`/`CRED`/`PROVIDER` anywhere, BFF responses included;
       - `CODE`/`TEXT`/`DISPUTE` only in the test's explicit (schema, table, column) allowlist;
       - the caller's own `CODE`/`TEXT` only on a second explicit allowlist of (BFF route, JSON path): drafts, the submission view, AB17's own-code excerpt, AB16 F1's verified quote;
       - `DISPUTE` in no BFF response; no account's canary in another account's responses;
       - `DISPUTE` may also appear in `disputes export`; no canary appears in any event.
   - **Manual compose check** (nothing committed): the same scenario once on `docker compose up --build`, with a fake-provider override kept in your scratchpad; grep `docker compose logs` for the canaries and paste the (empty) result into the PR.
3. **[X] M4 exit suite** (plan task 2): `internal/e2e/m4_exit_test.go` on the same harness, with the same seeded calibration rows, covering:
   - pre-fill within caps (owner and tester in Go/C++/Python; ≤ 8 calls; the 16 KiB skip; the learner gets no `llm_call` row and 404s from the presence-gated routes, while `/api/me/consents` still answers 200);
   - the degrade order (advisory → counted → final headroom);
   - provider limit 400 → breaker → no more calls → manual; `judge admin breaker clear --scope llm` resumes;
   - the kill switch (the owner's allowance reads `off`/`platform_disabled`; nothing re-graded, the loop never blocks, no retroactive analysis on re-enable, per m4-03's rule);
   - consent off → no calls;
   - `ai-disable`;
   - exactly one re-grade, on the seeded `score_regrade` row's independent configuration.

   Record the run in the plan.
4. **[X] Harness + verbs.** Use m4-02/m4-03's verbs; add **no** new verb.
   - Make `cmd/judge-eval` print the run's total µUSD and p50/p95 $/analysis with judge's own usage parser and price table, if it doesn't already.
   - **Ledger:** m4-02 ships `judge admin ledger --month YYYY-MM [--by purpose|model|account]`. Extend it additively, with tests, if it's missing a per-day breakdown (`--by day`), status counts, the **token sums** (input, cache read/write, output) or the prices version. The cents-level Console check compares tokens.
   - **Acceptance row:** m4-02's `judge admin calibration record --from <path>|-` takes m4-03's `--calibration-out` **`row.json`** (not the report). The judge image is distroless, so production feeds it on stdin (`--from -`). If stdin support is missing, add it with a test. In the harness, check it accepts a real `row.json` shape and refuses `passed=false`.
   - Append the **"Monthly ledger check"** section to `docs/v2/runbooks/platform-ai-provider.md`: ±5% pre-tax USD after the month settles; ledger > Console → fix prices; Console > ledger → disable the credential first, then investigate; log each check in status.md.
5. **[E + O] Acceptance run** (plan task 3).
   - **You (dry run, no key):** `go run ./cmd/judge-eval --set ../xlearn-evalpack/acceptance --split dev --provider fake --out dry-run.json` (output stays in your scratchpad). It validates labels and counts, prints the planned configurations and a cost estimate, and confirms the test log is empty. **Never run `--split test`, even with the fake provider:** every test-split run appends to `configs_tried.log`. Check `--help` for how the harness reads `ANALYZER_FP_THRESHOLD`.
   - **Dev results:** if status.md already records m4-03 task 10's dev-split tuning (effort, `max_tokens`, τ) for the current `prompt@v`/`schema@v`, reuse it and skip the sweep.
   - **Hand the owner the exact commands** to run in **his own shell**, never in this session:
     1. the dev sweep (skip it if reused):
        ```sh
        export LLM_CALIB_API_KEY=…   # the xlearn-calib key
        make judge-eval SET=../xlearn-evalpack/acceptance SPLIT=dev PROVIDER=anthropic SWEEP=effort:low-nothink,low,medium
        ```
        It sweeps `claude-sonnet-5` × {`low` + thinking disabled, `low`, `medium`}. Pick the cheapest passing configuration, `max_tokens` = p99 × 1.5, and τ from the report's sweep over the dev `pair_of` examples;
     2. one frozen **test** run, with the key still exported:
        ```sh
        ANALYZER_FP_THRESHOLD=<τ> go run ./cmd/judge-eval --set ../xlearn-evalpack/acceptance --split test \
          --provider anthropic --effort <effort> --max-tokens <max_tokens> \
          --out report.json --calibration-out row.json
        ```
   - **Gate:** accuracy ≥ 80% with lower bound ≥ 70%; false pointers ≤ 10%; line ranges 100%; schema-valid ≤ 1 retry 100%; truncation ≤ 2%; p95 $ recorded.
   - **On failure:** cut D26 to course passes (an owner call) and log a second configuration. **After two failed test runs, stop and report; don't tag.**
   - Open an `xlearn-evalpack` PR with `acceptance/reports/<date>-<config>.json` (the test run's `report.json`) + `configs_tried.log` (**numbers only**) → CI green → squash-merge. No evalpack tag. Keep `row.json` for step 10.
6. **[X + O] Size caps + pre-tag ledger validation** (plan tasks 4–5).
   - **Caps:**
     - compute the per-account month projection (3.05 × ≈ 21 × p95 $/analysis) and check it against $6/month, $1/day (20 × p95), and N × 1.2 × P90 ≤ $12;
     - write the caps table for status.md; list only the env values that differ from m4-02's defaults, plus the analyzer route and the threshold, `ANALYZER_FP_THRESHOLD` (`grep -rn 'LLM_\|ANALYZER_' internal/judge internal/platform/llm cmd/judge`);
     - ask the owner to set the Console per-model RPM/OTPM to ≈ 2 × the lane's use.
   - **Ledger:** the owner reads the `xlearn-calib` Console cost for the run day. It must match the harness's total within ±5%; fix and re-check if not.
7. **[X] Verify and merge.** `gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` clean (only if a verb added a query) · the contract-header lint silent · the OpenAPI drift test · `npm --prefix web run {typecheck,lint,test,build}` · `XLEARN_TEST_DATABASE_URL=… go test -tags e2e -race ./internal/e2e/...` (CI's e2e lane). Then the PR with conventional commits and the attribution lines → CI green → squash-merge.
8. **[I] Pre-tag infra check** (plan task 6). All read-only:
   - the ACL golden rendered at the tag commit equals `messaging/release.yaml` (judge on `XLEARN_PRACTICE`, the `evaluation_analyzed` publish, review on `XLEARN_JUDGE`);
   - judge's NetworkPolicy (443 + identity :8081) and the existing dispute-path edges;
   - the `xlearn-judge-llm` secret and the projected token;
   - `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` (the script header's documented invocation) green, including the JWKS kid check;
   - no new pod.

   Open and merge an infra PR **only for a gap**, before the tag.
9. **[X] Tag `v1.16.0`** with the plan's **release checklist**. Title **`v1.16.0 — v2 build · M4 platform AI`**; release notes per the plan (dark behind `LLM_PLATFORM_ENABLED=false`; the cohort flip follows; the D14/D26 promises). **Right before pushing the tag, re-check for peer and owner messages.**

   **First post-tag step: m4-01's pod smoke** (its task 10). Run `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin llm-smoke'`. It must pass: WIF mode, Models 200, the workspace header matching when present, and `claude-sonnet-5` and `claude-opus-5-5` listed. It only calls `GET /v1/models`, so it **spends nothing and writes no ledger row**, only an admin-audit row. A failure **blocks step 11 (the flag PR), not the tag**: fix forward and re-run.

   Then, by looking:
   - healthz reports the version; `get deploy` shows the images; ImagePolicy latest = the tag; HelmReleases Ready;
   - the smoke test (login, dashboard, coach);
   - still dark: the owner's `GET /api/me/ai-allowance` reads `off`/`platform_disabled`, so no AI surface shows; a `learner` account (if any) gets 404 from the presence-gated routes. `/api/me/consents` answers for every account (not cohort-gated) with its section hidden;
   - judge's and review's new durables bound; no dead letters.
10. **[O] Record the acceptance row on production.** `ssh vps 'k3s kubectl exec -i -n xlearn deploy/xlearn-judge -- judge admin calibration record --from -' < row.json`, using step 5's `row.json` (m4-03's `--calibration-out`), not the report. It's an admin-CLI write, so the owner runs it, or you run it only on the owner's explicit go-ahead in chat. Check `judge admin calibration list` shows it, and log it in status.md.
11. **[I] Enable for the cohort** (plan task 8). One `../infra` PR on `apps/xlearn-judge.yaml`:
    - the sized values, the analyzer route and threshold, and `LLM_PLATFORM_ENABLED: "true"` on its own line;
    - leave `LLM_ACCEPT_STD_RETENTION` **unset**;
    - merge only after step 9's verification (pod smoke green) and step 10's recorded row.

    After Flux applies it:
    - judge restarts; `judge admin ai status` (m4-02's AI digest; plain `judge admin status` is m3-14's runner status) shows the flag on, the breaker closed and the calibration row in use;
    - the owner's `/api/me/ai-allowance` → 200, reading `off`/`no_consent` until he ticks the consents;
    - `k3s kubectl top pod -n xlearn` shows judge under its limit.

    **Ask the owner to tick both AI consents** in Settings (a minute; they're unticked by default, so nothing runs on his account until then). They take effect on the next call: the gateway refreshes judge's cache on every consent PATCH, so there's no wait.

    **Record M4 day 1** (`ev-m4-day1`) = the merge date.
12. **[O] Optional owner day-1 dogfood** (plan task 9; gates neither _Overall_ ✅ nor M4 ✅).
    - If the owner has about 1 h, hand him the checklist:
      - a below-Clean solve → the AI mistake suggestion;
      - a clean pass → notes or "no improvement notes";
      - a due touch if any;
      - the allowance shows a percentage and no dollars.
    - Meanwhile, read-only: `judge admin ledger --month`, `judge admin ai status`, dead letters.
    - An unanalyzed first conclusion means a failed cache refresh, not an expected delay: the gateway logs an ERROR line, and only then does the 5-minute TTL apply. Check `judge admin ai status` and the logs.
    - Issues go under the label `m4-dogfood`, fixed forward as `v1.16.x`.
    - If the owner skips it, mark it ⛔ "skipped (optional)".
13. **[X + O] Production ledger vs Console + M4 exit** (plan task 10).
    - On the first production spend (the dogfood, or the owner's ordinary use), once the Console has settled: compare the owner's Console reading with `judge admin ledger --month YYYY-MM --by day`, and **tokens too** if the total is < $0.20. It must be within ±5%. If there's no settled spend yet, mark it ⛔ "pending first production spend / Console" and close it in a short follow-up; until then _Overall_ and M4 stay 🔄.
    - Optional exhaustion drill (`judge admin llm-limit <owner> --day-usd 0` → manual → restore the previous values, or `--clear` if there were none), **only on the owner's go-ahead**, logged.
    - Write m4-01's pod-smoke output into `sprint-m4-01.md`: its task 10 ✅, its pod-smoke acceptance item ticked, its _Overall_ ✅.
    - Then *Update status*.

## Constraints

- **Keys and secrets:** you never see or type the `xlearn-calib` key, the salt, a token or the break-glass key. The owner runs key-bearing commands in their own shell, and reports and PRs carry numbers only. Never set `ANTHROPIC_API_KEY` anywhere.
- **Private content:** nothing from `../xlearn-evalpack` (labels, artefacts, pack text) is copied into this repo, its PRs or CI logs. Test fixtures are synthetic and public.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** judge alone holds the platform credential and the ledger; verbs live in `internal/judge/admin`; no cross-schema reads.
- **goose + sqlc:** a new verb's queries are read-only or target judge's own tables. If a migration is ever needed, it's expand-only (the contract lint stays silent); `sqlc generate` is committed and `sqlc diff` is clean. Never run `Down` in prod.
- **Outbox/inbox + consumers before producers:** `evaluation_analyzed` stays dark until the enable PR, and that PR merges only once review's durable is bound. The ACL PR is merged before the tag.
- **GitOps:** never `kubectl apply`. The enable PR and any gap PR are **their own infra PRs, never folded into the tag**. Never move or re-push a tag; don't suspend the shared IUA. Production writes happen only through the admin CLI, by the owner or on the owner's explicit go-ahead, and each is logged in status.md.
- **D34:** no alerting, opscheck, healthchecks.io, Flux Alert or push channel. Spend is watched through the provider's Console limit and alerts and `judge admin ledger`, by looking.
- **D25 / D35:** limits stay at $15 provider / $12 app for all of v2; never raise them here.
- **No UI work here** ([m4-06](../sprints/sprint-m4-06.md) shipped it). A dogfood UI fix is a `v1.16.x` patch built to the frozen AB16–AB18 boards with `theme.css` verbatim.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod and no limit change; confirm judge's working set after the flip.
- **Parallel sessions:** check peers' tags, PRs and worktrees (+ ListAgents) in xlearn, `../infra` and `../xlearn-evalpack` before tagging and before claiming any ADR number, and again right before the tag push.

## Deliverables

- `internal/platform/llm/canary_test.go`, `internal/judge/ai/canary_test.go`, `internal/e2e/m4_canary_test.go`, `internal/e2e/m4_exit_test.go`; the `httpx` redaction fix if needed.
- Extensions to m4-02's verbs only if missing (the ledger's per-day and token sums; `calibration record --from -` stdin) + tests; the harness cost output. No new verb.
- `docs/v2/runbooks/platform-ai-provider.md` "Monthly ledger check" section.
- `xlearn-evalpack` PR: the acceptance report + `configs_tried.log`.
- **`v1.16.0`** tagged and verified; m4-01's pod smoke recorded; the acceptance row on production; the `LLM_PLATFORM_ENABLED=true` infra PR; M4 day 1; both ledger checks (production may be ⛔ pending); the optional day-1 dogfood.

## Update status

- [`../sprints/sprint-m4-07.md`](../sprints/sprint-m4-07.md): task rows ✅ (or ⛔ with the reason: "pending first production spend / Console", "skipped (optional)"); _Overall_ ✅ once tasks 1–8 and 10 are done. Task 9 is optional and never blocks it.
- [`../sprints/sprint-m4-01.md`](../sprints/sprint-m4-01.md): the pod-smoke output pasted in, task 10 ✅, the pod-smoke acceptance item ticked, _Overall_ ✅.
- [`../status.md`](../status.md):
  - **Sprint board:** m4-07 ✅; note l-04, l-05 and ga-01 are unblocked on the M4 side.
  - **Milestones:** **M4 ✅** with `v1.16.0` and the exit evidence.
  - **Tag → floor → snapshot:** `v1.16.0 → 1.13.0 → n/a`.
  - **Flag inventory:** `LLM_PLATFORM_ENABLED` **true** (cohort, permanent kill switch, PR #); `LLM_ACCEPT_STD_RETENTION` unset (reason).
  - **The caps table** and the analyzer tuple, with the report's aggregate numbers.
  - **Events:** `ev-m4-day1` ✅ (date; "`SEAT_CAP` re-size not before day 1 + 14, an opening gate").
  - **Manual checks:** the next monthly ledger check; `retire_not_before` before any model change; the JWKS kid check after k3s upgrades.
  - **Content:** acceptance set used (report version).
  - **Admin-CLI writes on production:** the `calibration record`, and the drill if run.
  - **Decisions log:** the chosen configuration and threshold (and whether the dev results came from m4-03 task 10), caps, any verb extension, the backlog rule, the redaction fix, m4-01's pod-smoke result.
- ADR only if you depart from ADR-0031/0034/0035 (e.g. a different acceptance-row mechanism); check peers' ADR numbers first.

## Done when (acceptance)

- [ ] **Canary:** unit + `-tags e2e` green. No `PACK`/`CRED`/`PROVIDER` anywhere, BFF responses included; content canaries only in allowlisted columns and, for the caller's own content, allowlisted (BFF route, JSON path) fields; `DISPUTE` in no BFF response; no cross-account leak; none in events; `PACK` only in `Score`; `DISPUTE` in no provider request.
- [ ] **M4 exit (`-tags e2e`):** pre-fill within caps; the degrade order; exhaustion → breaker → manual; the kill switch re-grades nothing; consent off → no calls; one independent re-grade.
- [ ] The acceptance gate passed on the frozen test split; the report is in `xlearn-evalpack`; `row.json` is recorded on production (`calibration record --from -`).
- [ ] Caps sized within $15 / $12 and recorded; the ledger is within ±5% of the Console on the calib run and on the first production spend (tokens compared at cents-level totals; production may close in a short follow-up).
- [ ] `v1.16.0` is live and verified; m4-01's pod smoke green after it and recorded in sprint-m4-01.md; `LLM_PLATFORM_ENABLED=true` merged after both, for the cohort only; M4 day 1 recorded; status.md shows M4 ✅. (The day-1 dogfood is optional.)

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: tag `v1.16.0` (the next free minor), then the `LLM_PLATFORM_ENABLED=true` infra PR**. In order:
1. the `xlearn-evalpack` report PR → CI green → squash-merge;
2. the xlearn PR → CI green → squash-merge;
3. any pre-tag gap infra PR → merged;
4. the tag → Flux deploys → verify per the checklist;
5. m4-01's pod smoke, then the production `calibration record --from -` (owner, or on the owner's go-ahead);
6. the enable infra PR → CI green → merge → verify → the owner ticks the consents → M4 day 1 recorded;
7. the production ledger check (or ⛔ "pending first production spend / Console") and the optional dogfood;
8. `git checkout main && git pull` in every repo touched (xlearn, `../infra`, `../xlearn-evalpack`).

Never leave an open PR or merged work unpulled. Don't enable PR auto-merge.
