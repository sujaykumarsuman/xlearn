# Sprint m4-07 — Canary log test, caps sizing, ledger check → v1.16.0 (+ LLM_PLATFORM_ENABLED for the cohort)

> **Milestone:** M4 — platform AI (the **M4 exit** and its tag) · **Track:** product · **Order:** 66
> **Prereqs:** [m4-06](sprint-m4-06.md) (and through it [m4-01](sprint-m4-01.md) … [m4-05](sprint-m4-05.md)) · [mi-12](sprint-mi-12.md) (MI-14: judge 443 + identity egress, `xlearn-judge-llm`, projected token, runbook, ADR-0031 Accepted) · owner events `ev-acceptance-set` (labelled) and `ev-provider-runbook` (done)
> **Unblocks:** [l-05](sprint-l-05.md) (rides `v1.17.0`, not an M4 tag) · [l-04](sprint-l-04.md) (`v1.16.0` live) · [ga-01](sprint-ga-01.md) (M4 complete)
> **Release action:** **tag `v1.16.0`** (indicative: the next free minor at tag time, [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline); gate state after: `LLM_PLATFORM_ENABLED` + cohort; rollback floor unchanged at `1.13.0`) · **then the infra PR `LLM_PLATFORM_ENABLED=true`** (its own PR, after the tag is verified) · an `xlearn-evalpack` PR for the acceptance report (no evalpack tag: the acceptance set is not in the pack image) · a pre-tag ACL/NetworkPolicy infra PR only if task 6 finds a gap
> **Calendar:** December 2026 · owner, before launch: export a fresh `xlearn-calib` key as `LLM_CALIB_API_KEY` in the shell that launches the session, on his machine (the session runs the key-bearing acceptance commands there and never prints the key; [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used)) · owner, after ship (never waited on; `ev-m4-followup`, recorded by task 9): tick the two AI consents, ≈ 15 min in the Console (the ledger reads and the rate limits), optional ≈ 1 h day-1 dogfood (gates neither this sprint's ✅ nor M4 ✅). `ev-m4-day1` itself is only the date the enable PR merges.
> **Execute with:** [`../prompts/prompt-m4-07.md`](../prompts/prompt-m4-07.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Canary log test: unit (public CI) + the `-tags e2e` harness | X | ⬜ |
| 2 | M4 exit suite (`-tags e2e` harness): pre-fill within caps, exhaustion → manual, breaker, kill switch, consent off, re-grade | X | ⬜ |
| 3 | Analyzer acceptance run on the owner's before-launch `xlearn-calib` key: agent dry run → dev sweep (or m4-03 task 10's recorded results) → one frozen test-split run → report | X + E | ⬜ |
| 4 | Size the L17 caps (+ the Console rate-limit values for the owner, after ship); record the acceptance row (`judge admin calibration record --from -`) | X | ⬜ |
| 5 | Ledger: extend m4-02's `judge admin ledger --month` (days, tokens) if missing; the monthly procedure; the calib-run check (its Console read is the owner's, after ship: ⛔ pending in status.md, non-gating) | X | ⬜ |
| 6 | Pre-tag infra check: ACL, NetworkPolicy, secret/values, JWKS, memory | I | ⬜ |
| 7 | Tag `v1.16.0` (release checklist) + m4-01's pod smoke as the first post-tag step | X | ⬜ |
| 8 | Enable for the cohort: sized values + `LLM_PLATFORM_ENABLED=true` (own PR, after the tag); record `ev-m4-day1` | I | ⬜ |
| 9 | Record the owner's post-ship items as the owner event `ev-m4-followup` in status.md: the consents, the **optional** day-1 dogfood checklist, the Console follow-ups (the event gates nothing) | X | ⬜ |
| 10 | M4 exit recorded; production ledger vs Console (±5%): the owner's Console read after the first production spend, ⛔ pending in status.md (non-gating) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, **M4 → ✅**, tag → floor, flag inventory, events, manual checks, the caps table).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **Acceptance set labelled** (`ev-acceptance-set`, already an [m4-01](sprint-m4-01.md) entry gate): `../xlearn-evalpack/acceptance/` holds **≥ 70** labelled examples, **test ≥ 40** (15 optimal, 15 suboptimal, 10 failing) + **dev ≥ 30**. They validate against mi-12's label schema, and the **test split is untouched** (its `configs_tried.log` is empty).
- [ ] **Provider runbook executed** (`ev-provider-runbook`, [mi-12](sprint-mi-12.md)):
  - `xlearn-platform-prod` has a **$15** hard limit, Console alerts at 50%/80% and auto-reload **off**;
  - the WIF service account and rule exist, **plus** the separate **`xlearn-calib`** workspace with its own limit. The owner's key for it (7–30-day expiry) is a **before-launch** item: exported as `LLM_CALIB_API_KEY` in the shell that launches the session, on the owner's machine (a missing key isn't a gate failure: task 3 says what lands). That key is **never in the cluster, never pasted into or printed by the session**.
- [ ] **MI-14 live** ([mi-12](sprint-mi-12.md)):
  - [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) Accepted;
  - judge egress = TCP 443 to non-cluster addresses + `xlearn-identity` :8081;
  - `xlearn-judge-llm` loaded, `LLM_PLATFORM_ENABLED=false`, the projected token mounted.
- [ ] **All M4 code merged:** [m4-01](sprint-m4-01.md) (its compose `llm-smoke` output recorded in its PR; the pod smoke can only run after this sprint's tag, in task 7), [m4-02](sprint-m4-02.md), [m4-03](sprint-m4-03.md), [m4-04](sprint-m4-04.md), [m4-05](sprint-m4-05.md), [m4-06](sprint-m4-06.md) (`git log origin/main`). `git log v1.15.0..main` shows nothing that must not ship in `v1.16.0`.
- [ ] **m4-03's ACL PR merged** in `../infra`: judge's durable on `XLEARN_PRACTICE` and review's durable on `XLEARN_JUDGE` (`evaluation_analyzed`). Task 6 re-checks the golden.
- [ ] **`v1.15.0` live** ([p-03](sprint-p-03.md)), so P and M4 never share a tag, and no open peer PR edits `internal/judge`.
- [ ] **Parallel sessions:** no peer tag or open PR claims `v1.16.0`, and no open peer PR edits `../infra/apps/xlearn-judge.yaml` (`git ls-remote --tags origin`, `gh pr list` in xlearn and `../infra`, `git worktree list`, ListAgents).

## Goal

**Close M4.** Prove platform AI is **safe** (the canary test: no prompt, completion, pack material or credential reaches a log, event, ledger row or learner-visible field, and learner content goes back only to its owner, on allowlisted fields) and **within budget** (caps sized from the owner's labelled acceptance run, and a ledger that matches the Anthropic Console). Cut **`v1.16.0`** with platform AI dark, then turn it on for the **owner/tester cohort** in its own infra PR. That merge is **M4 day 1**, which starts the ≥ 2-week data window that re-sizes `SEAT_CAP`. The window is a v3 **opening** gate, not a GA gate ([rollout §3](../rollout-plan.md#3-milestone-map), [§11](../rollout-plan.md#11-opening-gates-v3)).

**M4 exit** ([rollout §3](../rollout-plan.md#3-milestone-map)): pre-fill within caps; exhaustion → manual; ledger within ±5% of the Console.

Sources:
- [rollout §3](../rollout-plan.md#3-milestone-map) (M4 exit), [§4 M4](../rollout-plan.md#m4-platform-ai--68-sprints) (gating, content), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.16.0`);
- [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md): §3 pinned catalog, §4 canary, §5 caps and sizing;
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md): §2 kill switches, §3 dark producer, §4.4 M4 row, §6;
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md): §2 standing rules, §3 provider side (no alerting), §4 L17;
- [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (acceptance gate), [§5](../research/t5-platform-ai.md#5-cost-model) (unit costs, activity model), [§6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (caps, sizing rule, ledger, error mapping), [§7 item 8](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) (canary), [§8](../research/t5-platform-ai.md#8-privacy-and-residency) (logging rules).

## Scope

**In**
- The canary test (unit + the `-tags e2e` harness) and the log-redaction fixes it forces.
- The M4 exit suite in the `-tags e2e` harness.
- The acceptance run: effort, `max_tokens` and the D26 fingerprint threshold picked on dev (or taken from [m4-03](sprint-m4-03.md) task 10's recorded dev results); one frozen test run.
- Cap sizing within D25's dogfood limits.
- Extending [m4-02](sprint-m4-02.md)'s audited `judge admin ledger --month` with a per-day breakdown and token sums, **if missing**. The acceptance row is written with m4-02's existing `judge admin calibration record --from -`, fed m4-03's `--calibration-out` row; there is no new verb.
- The monthly ledger procedure, appended to `docs/v2/runbooks/platform-ai-provider.md` (mi-12's runbook reserves that section for this sprint).
- The pre-tag infra check, the tag, m4-01's pod smoke, the enable PR, the M4 exit record, and the owner's post-ship items recorded as an owner event (the consents, the optional day-1 dogfood, the Console reads for the production ledger check).

**Out**
- Raising the limits to **$100 / $80** and re-sizing `SEAT_CAP` from ≥ 2 weeks of M4 data → the **v3 opening** ([rollout §11](../rollout-plan.md#11-opening-gates-v3)), not v2.
- Platform AI on for **every** account (the T-1 default flip) → GA ([ga-01](sprint-ga-01.md), [ga-02](sprint-ga-02.md)).
- The consents at the acceptance step (L-C) → [l-05](sprint-l-05.md); `v1.17.0` → [l-04](sprint-l-04.md).
- Rubric calibration (≥ 50-example test split per rubric, QWK gate) → the SD-course milestone (v2.2+). No v2.0 item calls `Score`.
- A second processor (the `gpt-6-luna` lever, an OpenAI challenger) → not v2.0; after a failed gate it is only an option recorded for the owner's decision (task 3), by ADR, never added here.
- Any alert, push ping, opscheck digest or Flux Alert (D34). `judge admin` reads and the provider's own Console alerts replace them.

## Tasks

### 1 · Canary log test [X]

[ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) ("a canary test enforces this"), [t5 §7 item 8](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (allowed log fields; `slog.LogValuer` → `[redacted N bytes]`; provider error bodies parsed ≤ 8 KiB and never echoed).

**Canaries:** a fresh random token per class per run, e.g. `XLCANARY-PACK-<16 hex>`.

| Class | Planted in |
|---|---|
| `PACK` | the synthetic fixture pack: rubric anchors and exemplars of the synthetic rubric item, and hidden-case tags |
| `CODE` | the learner's code, as a comment and a string literal, in a course submission, a touch and an arena submit |
| `TEXT` | rubric text parts |
| `DISPUTE` | the dispute text |
| `CRED` | a fake `LLM_ANTHROPIC_API_KEY`, the projected-token file (a JWT-shaped string with the canary in a claim), the access token the fake exchange returns, and `LLM_ENDUSER_SALT` |
| `PROVIDER` | the fake provider's error bodies, and a field of a completion that judge must drop (e.g. thinking text) |

**Unit** (public CI, no network): `internal/platform/llm/canary_test.go` + `internal/judge/ai/canary_test.go`, with a capturing `slog.Handler` at DEBUG and a fake transport. It runs Score, Feedback, Analyze, the re-grade, the WIF exchange, the breaker and the validator failure paths, then asserts:
- no canary in any log record (message, attrs, `LogValuer` output), any `error.Error()`, any validator error (field paths and ids only), any `llm_call` value, `ai_sample` after finalize (deleted), or any outbox payload (`evaluation_completed`, `evaluation_analyzed`);
- **request shape** (the fake transport records every outbound body):
  - `PACK` appears **only** in `Score` requests, never in Feedback or Analyze (the compile-time `PackView` guard plus this contract test);
  - `DISPUTE` appears in **no** request (the re-grade never reads the appeal);
  - `CRED`'s salt never appears raw (only the HMAC pseudonym in `metadata.user_id`);
  - no raw `account_id`, email or username in any request;
- **panic path:** a forced panic whose value contains `CODE` is recovered without logging it. Today `internal/platform/httpx/httpx.go:65-68` logs `"panic", rec` verbatim. If that is still there, change it to log the type and `[redacted N bytes]` (a small fix riding this tag; it covers every service).

**e2e** — `internal/e2e/m4_canary_test.go` (`-tags e2e`), on the in-process harness m4-03/m4-04/m4-05 extended (real Postgres and JetStream, services and the gateway in-process; CI's e2e lane runs it). Judge has `LLM_PLATFORM_ENABLED=true` and the fake credential env in the harness config only, and runs on m4-02's `httptest` fake provider. That is m4-01's convention: no fake provider is a committed compose service.
- **Setup:**
  - **two cohort accounts** (owner and tester), each with its **own** `CODE`/`TEXT`/`DISPUTE` canaries, plus a `learner` account; the consents are set through `PATCH /api/me/consents`;
  - **passed `llm_calibration` rows** for the synthetic configurations: `analyze`, and m4-04's `score` + `score_regrade`. Seed them through `judge admin calibration record --from -` (m3-14's dispatcher called in-process) or m4-04's SQL fixture. Without them the analyzer makes no call (`no_acceptance`) and the rubric item answers 503, so the canaries would never reach a provider request.
- **Run** (per cohort account): a failed and a passed course attempt (analyzer and pass review), a touch, the synthetic rubric item's final → dispute → re-grade; then one breaker trip.
- **Collect:**
  - **logs:** a capturing `slog.Handler` at DEBUG, installed with `slog.SetDefault` **and** passed to every in-process service, so nothing logs around it; plus the stderr of every `judge admin` call;
  - **data:** a data-only dump of every schema (`pg_dump --data-only` against `XLEARN_TEST_DATABASE_URL`, or every table read through the harness pool); every outbox row; `event_dead_letter`;
  - **events:** the messages on every `XLEARN_*` stream, read directly from the harness's JetStream;
  - **BFF:** every BFF response each account received, keyed by (route, account);
  - **admin:** the output of `judge admin ai status`, `ledger --month` and `disputes export`.
- **Assert:**
  - `PACK` and `CRED` appear **nowhere**: not in logs, rows, events, admin output or any BFF response. The pack lives in the image volume, never in a DB row, log, event or DTO.
  - **DB:** `CODE`, `TEXT` and `DISPUTE` appear only in an explicit **allowlist of (schema, table, column)** that legitimately stores learner content (judge submissions and drafts, `evaluation_dispute.text`, practice drafts, `ai_sample` before finalize).
  - **BFF:** a second explicit **allowlist of (BFF route, JSON path)** where the **caller's own** `CODE`/`TEXT` may come back. That covers drafts, the submission view, AB17's excerpt of the learner's own code and AB16 F1's verified quote ("You wrote: '…'"). `DISPUTE` is in **no** BFF response (m4-04's DTO denylist). An account's canaries never appear in another account's responses, and `PACK`/`CRED`/`PROVIDER` appear in no BFF response at all.
  - Keep both allowlists in the test with a comment, so a new content column or field is added deliberately.
  - `DISPUTE` may also appear in the owner's `judge admin disputes export`, which exists to show it; `PACK`/`CRED` may not.
  - `PROVIDER` appears nowhere: error bodies aren't echoed and dropped fields aren't stored.
  - Events carry enums and refs only: **no canary of any class** in any outbox row or stream message.

**Manual compose check (belt and braces for the real binaries' log wiring; nothing committed):** run the same scenario once on `docker compose up --build`. Point judge's `LLM_ANTHROPIC_BASE_URL` at a fake through a `-f docker-compose.yml -f <override>` kept in the session scratchpad (m4-01's convention). Then grep `docker compose logs --no-color` for the run's canaries, expecting nothing, and paste the result into the PR.

### 2 · M4 exit suite [X]

`internal/e2e/m4_exit_test.go` (`-tags e2e`), on the same in-process harness as task 1, with m4-02's `httptest` fake provider scripted per case. Owner, tester and `learner` accounts are seeded as the harness seeds accounts (roles per m1-04); the cohort consents are set through `PATCH /api/me/consents`. **Seed first**, as in task 1: passed `llm_calibration` rows for `analyze` and m4-04's `score` + `score_regrade` configurations. Without the `analyze` row nothing is pre-filled (`no_acceptance`); without the score rows the rubric cases answer 503. The `judge admin` verbs run through m3-14's dispatcher, in-process, against the harness DB.

1. **Pre-fill within caps.**
   - Owner and tester conclude below-Clean DSA fixture attempts in Go, C++ and Python → `evaluation_analyzed` → review holds `category_source=analyzer` and concepts. A clean pass gets pointer notes and "correct, with improvements".
   - Every analysis's `llm_call` rows are settled with cost > 0, ≤ 8 calls per evaluation, and ≤ `max_tokens`. An input > 16 KiB is `analysis_skipped(too_large)` with no call.
   - The **learner** (non-cohort) gets no `llm_call` row, 404s from the presence-gated routes (`/api/me/ai-allowance`, analysis, pointer notes, optional revisit, dispute), and the self path unchanged. `GET /api/me/consents` still answers 200: m4-05 doesn't gate it by cohort.
2. **Caps and the degrade order** (ADR-0031 §5):
   - with tiny limits via `judge admin llm-limit`, advisory pass notes shed first when spend runs ahead of pace;
   - counted analyses continue to the counted headroom;
   - a rubric final keeps its `final` headroom;
   - past the account cap → `ErrBudgetExhausted` → deterministic pre-fill + manual category, and the allowance shows `paused`.
3. **Exhaustion → manual (provider side).** The fake returns the 400 "workspace API usage limits" → the breaker opens → `global_paused` shows → **no further provider requests** (count them) → grading continues: deterministic verdicts, a manual mistake, `self_grade_pending` with "Wait / Grade it myself" on the rubric item. The breaker's reason follows m4-02's rule: `external_spend_or_ledger_bug` while the seeded month-to-date is under 70% of `LLM_PROVIDER_LIMIT_USD`, else `provider_quota`; assert whichever the seed gives. `judge admin breaker clear --scope llm` resumes.
4. **Kill switch; nothing re-graded** (ADR-0034 §4.4, M4 row).
   - Snapshot `practice.outcome`, the conclusion outbox rows and `GET /api/progress`.
   - Restart judge (rebuild it in the harness) with `LLM_PLATFORM_ENABLED=false`: the owner's allowance reads `off` (`platform_disabled`), so the UI treats it as absent; no llm-lane job is enqueued; open provisional attempts still settle through practice's ticker, and the loop never blocks. The snapshots are unchanged apart from new self-path work.
   - Re-enable: attempts concluded while AI was off are **not** analyzed retroactively. That's the expected m4-03 rule: the durable acks and skips while off. If m4-03 chose otherwise, assert its rule and name it in the decisions log.
5. **Consent off.** With the first consent off, the next conclusion has no `llm_call` row and the UI goes manual. With only the second off, failures are analyzed and passes get no notes.
6. **Per-account disable.** `judge admin ai-disable <tester>` → no calls for the tester, AB18 F6, an audit row.
7. **Re-grade.** Exactly one (`409 dispute_used`, `UNIQUE(regrade_of)`). Its ledger rows carry the independent configuration: the seeded `score_regrade` row's tuple (another `prompt@v`, or Opus 5.5 `medium`), not the first grade's.

Record the run (date, commit, pass counts) in this file.

### 3 · Analyzer acceptance run [X + E]

The harness is `cmd/judge-eval` from [m4-03](sprint-m4-03.md). Acceptance runs are billed to the owner's **`xlearn-calib`** workspace, **never** `xlearn-platform-prod` and never from the cluster ([ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25), [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used)). Budget ≈ $10–30 ([t5 §5](../research/t5-platform-ai.md#5-cost-model)).

**Before launch (owner):** a fresh `xlearn-calib` key (7–30-day expiry) exported as `LLM_CALIB_API_KEY` in the shell that launches the session, on the owner's machine. The session runs the key-bearing commands in that shell — a live run on an already-provisioned key within the stated budget, so launching the prompt pre-approves it (D40) — and never prints, copies or stores the key. **Key missing:** do step 1, land everything that doesn't depend on the run (the xlearn PR, task 6's read-only check), **don't tag** (no passed acceptance row), and record ⛔ "pending `LLM_CALIB_API_KEY` (owner, before launch)" in status.md; a re-run of the prompt picks up at step 2.

1. **Agent dry run (no key).** `go run ./cmd/judge-eval --set ../xlearn-evalpack/acceptance --split dev --provider fake --out dry-run.json` (m4-03's flags; the output stays in the session scratchpad, never committed; `make judge-eval SET=../xlearn-evalpack/acceptance SPLIT=dev PROVIDER=fake` is the same):
   - validates the labels against mi-12's `label.schema.json` and checks the split counts and mix. Check the test split's labels against the schema too, but **never run the test split, even with `--provider fake`**: every test-split run appends its config hash to `configs_tried.log`;
   - prints the planned configurations and a token/cost estimate from the dev set's sizes;
   - confirms the test split's `configs_tried.log` is empty.

   Fix harness bugs here (X). m4-03's report already carries p95 µUSD per analysis from usage × `platform/llm` prices; make sure it also prints the run's **total µUSD** with the **same** usage parser and price table judge uses (task 5 compares it with the Console), and add that output if missing.
2. **Dev results.** First check [`../status.md`](../status.md): if it already records [m4-03](sprint-m4-03.md) task 10's dev-split tuning (recommended effort, `max_tokens`, τ) for the current `prompt@v`/`schema@v`, **reuse it** and go straight to step 3. Otherwise the session runs the dev sweep in its shell, where the owner exported the key before launch (never pasted, printed or stored):

   ```sh
   # LLM_CALIB_API_KEY: exported by the owner before launch (the xlearn-calib key); never print it
   make judge-eval SET=../xlearn-evalpack/acceptance SPLIT=dev PROVIDER=anthropic SWEEP=effort:low-nothink,low,medium
   ```

   That sweeps `claude-sonnet-5` × {`low` with thinking disabled, `low`, `medium`} ([t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)). Pick:
   - the **cheapest** configuration that passes the thresholds on dev;
   - `max_tokens` = p99(thinking + visible) × 1.5 (provisional 4,000);
   - τ = the **D26 "materially different" fingerprint threshold** (`ANALYZER_FP_THRESHOLD`), from the report's τ sweep over the dev `pair_of` examples ([ADR-0031 §6](../../adr/0031-platform-ai-and-two-tier-keys.md#6-scope-of-ai-review-of-passing-solutions-owner-d26-refines-d16)).
3. **One test run.** Freeze the tuple (`prompt@v`, `schema@v`, model, effort, `max_tokens`, `anthropic-version`, threshold) and run the **test split once**, in the same shell with `LLM_CALIB_API_KEY` still exported:

   ```sh
   ANALYZER_FP_THRESHOLD=<τ> go run ./cmd/judge-eval --set ../xlearn-evalpack/acceptance --split test \
     --provider anthropic --effort <effort> --max-tokens <max_tokens> \
     --out report.json --calibration-out row.json
   ```

   The harness appends the config hash to `configs_tried.log` and writes `row.json`, the `llm_calibration` row task 4 records. It shares `internal/judge/ai`, so it reads `ANALYZER_FP_THRESHOLD` as judge does; confirm with `--help` in step 1.
   - **Gate** ([t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)):
     - category accuracy ≥ 80% with a bootstrap 95% lower bound ≥ 70%;
     - false-pointer rate ≤ 10%;
     - line ranges 100% valid;
     - schema-valid after ≤ 1 retry 100%;
     - truncation ≤ 2%;
     - p95 $/analysis recorded.
   - **If it fails:** apply the levers in order. `low` with thinking disabled was already swept, so next cut D26's scope to course passes only (a pre-decided lever: launching the prompt pre-approves it, D40; record it in the decisions log, and the owner may restore D26 later). A second test run counts as a new configuration in the log. After **two** failed test runs, **don't tag**: land the xlearn PR and the report PR (both runs logged), record the options (another configuration, a second processor by ADR) and a recommendation in status.md, and mark M4 ⛔ "needs owner decision". Never wait. Without a passed row the analyzer resolves to manual, and the M4 exit can't be met.
4. **Report** → an `xlearn-evalpack` PR: `acceptance/reports/<date>-<config>.json` (the test run's `report.json`) + `configs_tried.log`. **Numbers and the config tuple only:**
   - split sizes; per-category precision and recall; accuracy + lower bound; false-pointer rate; validity, truncation;
   - p50/p95 tokens (input, output, thinking); p50/p95 $/analysis; p95 latency; the total µUSD.
   
   No learner artefact, label text or pack text. Only the aggregate numbers go into the public status.md.

### 4 · Size the L17 caps; record the acceptance row [X]

**Sizing rule** ([ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)): app cap ≥ 1.2 × the P90 projection for N active accounts at the measured unit cost; provider limit = app cap ÷ 0.8; above the ceiling, pull the levers. In owner-only v2 (D35), N = the owner plus active testers, and D25's dogfood limits apply: **provider $15, app $12** ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L17).

- **Compute** from the report: the per-account month projection = 3.05 analyses per active day ([t5 §5](../research/t5-platform-ai.md#5-cost-model) activity model; re-measured after the data window) × ≈ 21 active days × p95 $/analysis. Check:
  - 1.2 × the owner's projection ≤ the per-account **$6/month**;
  - 20 analyses × p95 $/analysis ≤ **$1/day** (else lower the daily count cap, never the headroom rules);
  - N × 1.2 × P90 ≤ the **$12** app cap;
  - the daily guard stays at 15%; ≤ 6 AI finals/day, ≤ 8 calls/evaluation and the 16 KiB input skip are unchanged.
  
  Expect the L17 defaults to hold. Keep the owner at the account default unless the projection needs more, and any owner override must leave the testers their headroom under the app cap.
- **Values** go into task 8's infra PR: only those that differ from m4-02's compiled defaults, plus the analyzer route (effort, `max_tokens`) and the D26 threshold, **`ANALYZER_FP_THRESHOLD`** (m4-03; default 0.30). Use the env names judge actually reads (`grep -rn 'LLM_\|ANALYZER_' internal/judge internal/platform/llm cmd/judge`).
- **Console rate limits (owner, after ship):** compute the `xlearn-platform-prod` per-model RPM/OTPM at ≈ 2 × what the llm lane can use at the measured p95 tokens (lane cap 8, 2 workers) and record the values in `ev-m4-followup` (task 9). Setting them is Console work, so the owner lowers or raises them after ship; nothing waits on it. mi-12's runbook left this for "after the acceptance run".
- **Status table:** the measured unit cost, the projections and the chosen values go into [`../status.md`](../status.md).
- **The acceptance row.** The analyzer class resolves to a pinned model **only through a passed row** ([ADR-0031 §3](../../adr/0031-platform-ai-and-two-tier-keys.md#3-models) "pinned catalog"). Without one, every analysis goes manual.
  - The verb is [m4-02](sprint-m4-02.md)'s **`judge admin calibration record --from <path>|-`**. It validates the tuple against the compiled catalog and the binary's `prompt@v`/`schema@v`, refuses a row whose gate failed (`passed=false`), and writes the row plus its audit row. Its input is **`row.json`**, the `llm_calibration` row [m4-03](sprint-m4-03.md)'s `--calibration-out` wrote in task 3, **not** the report. The judge image is distroless (no shell or `tar`, so no `kubectl cp`), so the row goes in on stdin: `ssh vps 'k3s kubectl exec -i -n xlearn deploy/xlearn-judge -- judge admin calibration record --from -' < row.json`. If `--from -` is missing, add stdin support here (X, with a test); add no new verb.
  - Before the tag, dry-check it in the e2e harness: `calibration record --from -` accepts the real `row.json` shape and refuses a copy with `passed=false`.
  - Run it on production yourself **after the tag and before task 8**. It's a sanctioned admin-CLI write via `kubectl exec` ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)) that this plan specifies, so launching the prompt pre-approves it (D40). Log the use in status.md, and check `judge admin calibration list` shows the row.

### 5 · Ledger check [X]

- **Verb.** [m4-02](sprint-m4-02.md) ships `judge admin ledger --month YYYY-MM [--by purpose|model|account]`: USD totals from `llm_call` (settled rows plus `usage_unknown` at estimate), with an audit row. **Extend it additively if missing** (X, with tests): a per-day breakdown (`--by day`), status counts (`usage_unknown` at estimate, `inflight` flagged), the **token sums** (input, cache read/write, output) and the prices version. Task 10's cents-level check compares tokens, so the sums are required. Numbers only; read-only apart from its audit row.
- **Procedure.** Append a "Monthly ledger check" section to `docs/v2/runbooks/platform-ai-provider.md`:
  - after the UTC month closes and the Console has settled (≥ 24 h), compare the ledger total with the Console cost for `xlearn-platform-prod`, pre-tax USD (GST and FX excluded). The tolerance is **±5%**;
  - **ledger > Console:** the price table is stale or estimates stuck at `usage_unknown`. Fix `platform/llm/prices.go` (effective-dated) in a patch;
  - **Console > ledger:** external spend or a ledger bug. **Disable the credential first**, using the runbook's kill switches in order (Console rule/key, then `judge admin breaker set --scope llm`, then `LLM_PLATFORM_ENABLED=false`), then investigate ([t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls));
  - log each monthly check in status.md as a **manual check** (D34: nothing reminds you).
- **Validation on the calib run.** Compare the harness's total µUSD from task 3 with the `xlearn-calib` Console cost for that day. Reading the Console is owner-only, so record the harness total and ⛔ "pending Console read (owner)" in status.md (listed in `ev-m4-followup`, task 9) and carry on: neither the tag nor the flip waits for it (D40; meanwhile the $15 provider limit and the breaker bound any mispricing). Within ±5% validates the usage parser and price table; outside it, fix `platform/llm/prices.go` in a `v1.16.x` patch, or for Console > ledger disable the credential first (the procedure above).

### 6 · Pre-tag infra check [I]

Read-only first. A PR only for a gap: each gap is its own `../infra` PR, merged **before** the tag.

- **ACL** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)): the `authorization` block in `../infra/infrastructure/messaging/release.yaml` equals the golden rendered from `topology.go` at the commit you will tag. That covers judge's new durable on `XLEARN_PRACTICE` (`problem_solved`, `touch_concluded`), its publish on `xlearn.judge.evaluation_analyzed`, and review's durable on `XLEARN_JUDGE`.
- **NetworkPolicy** (standing rule):
  - `v1.16.0`'s new in-cluster caller is **judge → `xlearn-identity` :8081** (m4-02); the new external egress is **judge → TCP 443**. Both are mi-12's PR. Read `ssh vps 'k3s kubectl get networkpolicy -n xlearn xlearn-judge -o yaml'`.
  - The dispute path (gateway → practice → judge) uses existing edges; confirm them.
  - If [mi-11](sprint-mi-11.md)'s `xlearn` egress matrix is live, it includes all of these.
- **Secret and values:** `xlearn-judge-llm` is present; `LLM_PLATFORM_ENABLED=false`; the projected token is mounted with automount off.
- **JWKS:** `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` (read-only; the script header's documented invocation; it includes mi-12's kid check) is green. A k3s upgrade since mi-12 may have rotated the SA key, and a stale inline JWKS means every exchange fails 401.
- **Memory:** no new pod; judge's limit is unchanged, so the memory sum is unchanged ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).

### 7 · Tag `v1.16.0` [X]

- One xlearn PR (branch `feat/m4-07-canary-caps-ledger`) with the canary and exit tests, the redaction fix, any verb extensions and harness output added, and the runbook section. Conventional commits with the attribution lines; CI green; squash-merge.
- Run the **release checklist** (below). GitHub release title: **`v1.16.0 — v2 build · M4 platform AI`**.
- **Release notes:**
  - platform AI (analyzer and mistake suggestions, improvement notes on passes, honor-probe claims, the provisional/dispute contract, allowance, consents) ships **dark behind `LLM_PLATFORM_ENABLED=false`**;
  - a separate infra PR turns it on **for the owner and tester cohort only**;
  - flag inventory: `LLM_PLATFORM_ENABLED` is a permanent kill switch;
  - once on, nothing changes a grade except the learner's own choices (D14), and pass review never touches grade or ladder (D26).
- **Rollback floor after:** unchanged, **1.13.0** (expand-only migrations; the contract lint is silent). Not a contract, erase or GA tag, so no snapshot.
- **First post-tag step: [m4-01](sprint-m4-01.md)'s pod smoke** (its task 10; it can't run before the code is deployed). Run `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin llm-smoke'`. It must pass: WIF mode, exchange latency recorded, Models 200, the `anthropic-workspace-id` header matching when present, and `claude-sonnet-5` and `claude-opus-5-5` listed. It only calls `GET /v1/models`, so it **spends nothing and writes no ledger row**, only its admin-audit row. A failure **blocks task 8 (the flag PR), not the tag**: breaker/manual is the safe state. Fix forward (e.g. the runbook's JWKS re-paste) and re-run.
- **After the tag, by looking (D34):** the reads below that need a login (the owner's allowance and consents, the checklist's login/dashboard/coach smoke) use an already-signed-in browser session if the session has one; otherwise run the credential-free checks and record "owner login smoke pending" as a pending-smoke note in status.md. Never enter credentials.
  - still dark (`LLM_PLATFORM_ENABLED=false`): the owner's `GET /api/me/ai-allowance` reads `state: off`, `reason: platform_disabled`, so the SPA shows no AI surface; a `learner` account (if one exists) gets 404 from the presence-gated routes. `GET /api/me/consents` answers for every account (m4-05: not cohort-gated), and its Settings section stays hidden;
  - judge is Ready;
  - review's durable on `XLEARN_JUDGE` and judge's on `XLEARN_PRACTICE` are bound, with no new `event_dead_letter` rows.
- Then record task 4's acceptance row on production (`judge admin calibration record --from -`, fed `row.json` on stdin).

### 8 · Enable for the cohort [I]

One `../infra` PR, **its own task, never folded into the tag**, opened only after task 7's verification (the pod smoke green) and task 4's recorded acceptance row:
- `apps/xlearn-judge.yaml` env: the sized values (task 4) that differ from the compiled defaults, the analyzer route (effort, `max_tokens`), the D26 threshold, and **`LLM_PLATFORM_ENABLED: "true"`** on a line of its own. R-a is then a one-line PR back to `"false"`, not a revert of the sizing.
- **Leave `LLM_ACCEPT_STD_RETENTION` unset.** No v2.0 item calls `Score`. Set it, recording D24's attestation, with the first rubric course.
- **Consumers before producers:** judge's `evaluation_analyzed` producer is dark until this PR (the dark-producer pattern, [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)). Merge only once review's durable is bound (task 7).
- **Cohort:** every AI feature stays behind the T-3 cohort until GA; a `learner` account gets nothing.
- **After the merge:**
  - judge restarts once; `judge admin ai status` (m4-02's AI digest; plain `judge admin status` is m3-14's runner status) shows the flag on, cohort mode, the breaker closed and the calibration row in use;
  - `GET /api/me/ai-allowance` → 200 for the owner, reading `state: off`, `reason: no_consent` until he ticks the consents (404 for a `learner` account, if one exists); login-dependent, so handle it as in task 7;
  - `k3s kubectl top pod -n xlearn` shows judge well under its limit.
- **The owner ticks the two AI consents** in Settings (AB18) after ship: "xLearn AI reviews my graded work" and "…and also reviews my passing solutions". It takes a minute, and it's his own use of the product, so it's a post-ship owner event (`ev-m4-followup`, task 9) that nothing waits on. They're unticked by default ([ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24)), so no platform AI runs on his account until he does. They take effect on the next call: m4-05's gateway refreshes judge's account cache on every consent PATCH.
- **Record M4 day 1** = this merge's date (`ev-m4-day1`). It starts the ≥ 2-week data window; the earliest `SEAT_CAP` re-size is day 1 + 14. That's an opening gate (v3), not GA.

### 9 · Owner post-ship items → `ev-m4-followup`, incl. the optional day-1 dogfood [X]

The session's part is to record one owner event, **`ev-m4-followup`**, in status.md's owner events (with a pointer in the enable PR body). The event **gates nothing**: neither this sprint's _Overall_ ✅ nor M4 ✅, and no session waits on it. The e2e suites (tasks 1–2) prove the exit behaviour, and task 10's production ledger check can use any production spend, including the owner's ordinary use. The event lists:
- **Consents** (task 8): the owner ticks both AI consents in Settings; testers do the same.
- **Console** (owner-only): set the `xlearn-platform-prod` per-model RPM/OTPM to task 4's values; read the `xlearn-calib` cost for the calib-run day (task 5) and the `xlearn-platform-prod` cost and tokens for the first production-spend day(s) (task 10).
- **Optional day-1 dogfood** (about 1 h, after ticking the consents; they apply at once, with no wait):
  - Solve 2–4 DSA items:
    - one below Clean (a WA then a pass, or a hinted solve) → the AI mistake suggestion (AB16 F2) within about a minute; use or change it;
    - one clean pass → improvement notes or "no improvement notes" (AB17);
    - a due touch, if any (a synonym pattern answer exercises the claim).
  - Check the allowance meter (a percentage and a reset date, **no dollars**) and Today.
  - **Reads (read-only):** `judge admin ledger --month` (calls, cost, statuses), `judge admin ai status`, breaker closed, no `usage_unknown` or `inflight` rows older than their lease, no new dead letters.
  - If the first conclusion after ticking the consents goes unanalyzed, suspect a **failed cache refresh**, not an expected delay. m4-05's gateway logs an ERROR line when judge's `POST /internal/accounts/{id}/refresh` fails, and only then does the 5-minute TTL apply. Check `judge admin ai status` and the gateway/judge logs.
  - If a tester exists (after MI-5b), the tester ticks the consents and repeats one item. Otherwise the tester leg rests on task 2's e2e run.
  - The optional exhaustion drill (task 10).
- File issues with the label `m4-dogfood`. Blockers are fixed forward in a `v1.16.x` patch (R-c), never by moving the tag. A follow-up records what the owner reports (or "skipped (optional)"); the tag, the flip and M4 ✅ stand either way.

### 10 · Production ledger vs Console; M4 exit recorded [X]

- **Ledger check** on the first production spend, from the owner's dogfood or ordinary use after ticking the consents, so it can only come after ship, and the Console read is owner-only: record ⛔ "pending first production spend / Console read (owner)" in status.md (listed in `ev-m4-followup`) with the procedure, and carry on. Once the Console has settled (the same or the next day), the owner reads the `xlearn-platform-prod` cost for those day(s), and `judge admin ledger --month YYYY-MM --by day` (task 5) gives the ledger side. The totals must be within **±5%**. At cents-level totals (< $0.20), rounding swamps 5%, so **also compare token counts** (Console input/output tokens vs the ledger's token sums). A short follow-up records both. It gates neither the tag, the flip, _Overall_ ✅ nor M4 ✅ (D40); a miss is a follow-up (the monthly procedure in task 5).
- **Optional production drill** (it needs the owner's own conclusion, so it rides his post-ship dogfood in `ev-m4-followup` and is never waited on; it costs nothing because admission blocks before any call): `judge admin llm-limit <owner> --day-usd 0` → the next conclusion shows the paused allowance and manual entry → restore the owner's previous values (`llm-limit <owner> --clear` if he had no override) → log both writes in status.md. Otherwise exhaustion → manual rests on task 2.
- **Record in [`../status.md`](../status.md):**
  - **Milestones:** M4 ✅ with `v1.16.0`, on the session's own evidence. Exit: "pre-fill within caps (acceptance run + e2e suite [+ day-1 dogfood]), exhaustion → manual (e2e suite [+ prod drill]), ledger within ±5% (calib run + production: ⛔ pending the owner's Console reads, non-gating)".
  - **m4-01's pod smoke** (task 7): paste the output into [sprint-m4-01.md](sprint-m4-01.md), set its task 10 ✅, tick its pod-smoke acceptance item and set its _Overall_ ✅; add a line to the decisions log.
  - **Tag → floor → snapshot:** `v1.16.0 → 1.13.0 → n/a`.
  - **Flag inventory:** `LLM_PLATFORM_ENABLED` = **true** (cohort; permanent kill switch; PR #). `LLM_ACCEPT_STD_RETENTION` unset (with the reason).
  - **The caps table** from task 4, and the analyzer tuple (model, effort, `max_tokens`, threshold, `prompt@v`) with the report's aggregate numbers.
  - **Events:** `ev-m4-day1` ✅ (date; "SEAT_CAP re-size not before day 1 + 14, an opening gate"); `ev-m4-followup` ⬜ (task 9: consents, Console reads and rate limits, the optional dogfood; gates nothing).
  - **Manual checks:** the next monthly ledger check; a `retire_not_before` check before any model change; the JWKS kid check after k3s upgrades.
  - **Content:** acceptance set used (report version).
  - **Decisions log:** the chosen configuration and threshold (and whether the dev results came from m4-03 task 10); the caps; any verb extension (the ledger's per-day and token sums; `calibration record --from -` stdin if it was missing); the backlog rule (task 2); the httpx redaction; any admin-CLI write on production; m4-01's pod smoke result.
  - **Sprint board:** m4-07 ✅; note l-04, l-05 and ga-01 are unblocked on the M4 side.

## Acceptance criteria

- [ ] **Canary:** no `PACK`, `CRED` or `PROVIDER` canary anywhere (BFF responses included); `CODE`/`TEXT`/`DISPUTE` only in the allowlisted content columns; the caller's own `CODE`/`TEXT` only on the allowlisted (BFF route, JSON path) fields, `DISPUTE` in no BFF response, and no account's canary in another account's responses; no canary in any event; `PACK` only in `Score` requests; `DISPUTE` in no provider request (unit + `-tags e2e` green).
- [ ] **M4 exit (`-tags e2e`, CI's e2e lane):** pre-fill within caps; the degrade order holds; exhaustion → breaker → manual with no further provider calls; kill switch → nothing re-graded and the loop never blocks; consent off → no calls; one re-grade on an independent configuration.
- [ ] **Acceptance run:** the frozen test split was run once (or twice, logged) and passed the gate. The report is in `xlearn-evalpack`; the `row.json` row is recorded on production with `judge admin calibration record --from -`.
- [ ] **Caps:** sized per the rule within $15 / $12, recorded in status.md, and set in the enable PR.
- [ ] **Ledger:** `ledger --month` prints per-day totals and token sums; the monthly procedure is merged; the calib run's harness total is recorded, and each Console comparison (calib run, first production spend; ±5%, tokens compared at cents-level totals) is the owner's Console read after ship: ⛔ "pending Console read (owner)" in status.md until it lands, gating neither the tag, the flip, _Overall_ ✅ nor M4 ✅ (D40).
- [ ] `v1.16.0` is live and verified per the checklist; m4-01's pod smoke passed after it (spent nothing, output recorded in sprint-m4-01.md); the `LLM_PLATFORM_ENABLED=true` PR merged after both; AI is on for the cohort only; M4 day 1 is recorded.
- [ ] `ev-m4-followup` is in status.md with the consents, the Console items and the optional day-1 dogfood checklist (the owner's items gate nothing; findings come later as `m4-dogfood` issues).

## Release

**Tag `v1.16.0`** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): M4 platform AI; gate state after: `LLM_PLATFORM_ENABLED` + cohort; rollback floor unchanged at **1.13.0**), with the `xlearn-evalpack` report PR (task 3; no evalpack tag) and any pre-tag gap infra PR (task 6) merged before it, followed by the `LLM_PLATFORM_ENABLED=true` infra PR (task 8). No step waits on the owner (D40): the owner's post-ship items are `ev-m4-followup` (task 9).

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule):
- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:**
- **ACL:** m4-03's PR (judge on `XLEARN_PRACTICE`, review on `XLEARN_JUDGE` `evaluation_analyzed`), re-checked in task 6.
- **New service:** n/a (no new image or policy).
- **Contract / erase / GA:** n/a. Expand-only migrations, so no snapshot is required; `host-verify --cluster` is still read in task 6 for the JWKS kids.
- **From M6:** n/a.
- **New callers:** judge → identity :8081 and judge → 443, both in mi-12's merged PR.
- **Memory:** no new pod.
- **Extra after-tag reads:** m4-01's pod smoke first (`judge admin llm-smoke`; spends nothing; a failure blocks task 8, not the tag); still dark (the owner's allowance `off`/`platform_disabled`, a `learner` gets 404 from the presence-gated routes; `/api/me/consents` answers for everyone); both new durables bound; no dead letters.
- **Right before pushing the tag,** re-check for peer and owner messages (a hold overrides the ship).

**Record:** `M4 → v1.16.0 → floor 1.13.0 → snapshot n/a`; flags: `LLM_PLATFORM_ENABLED` true for the cohort (task 8's PR #).

**Rollback** ([ADR-0034 §4.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first), [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) M4 row), fastest first:
1. Console: disable the WIF rule, or drop the workspace limit → 401/400 → breaker → manual (seconds, no git).
2. `judge admin breaker set --scope llm` (`clear --scope llm` undoes it).
3. **R-a:** `LLM_PLATFORM_ENABLED: "false"` (a one-line infra PR).
4. Per-account `judge admin ai-disable`.
5. **R-b:** narrow the ranges, never below 1.13.0.
6. **R-c:** revert + patch tag (the default for code bugs).

**Residue: money already spent.** D25's dogfood limits bound it: provider $15, app cap $12.

## Definition of Done

CI green (including the canary unit tests, `sqlc diff` and the OpenAPI drift test) · the canary and M4 exit suites green in the `-tags e2e` harness (CI's e2e lane) · the acceptance report merged in `xlearn-evalpack` · any pre-tag infra gap merged · `v1.16.0` tagged, deployed by Flux (no hand `kubectl apply`) and verified by the checklist · m4-01's pod smoke green after the tag and recorded in sprint-m4-01.md (its task 10 ✅) · the acceptance row recorded on production (`calibration record --from -`) · the enable PR merged after the tag · the calib-run harness total recorded, with both Console comparisons recorded or ⛔ "pending Console read (owner)" · `ev-m4-followup` recorded (consents, Console items, the optional day-1 dogfood; never waited on) · statuses updated (this file + [`../status.md`](../status.md): board row, **M4 → ✅**, tag/floor row, flag inventory, the caps table, events, manual checks, decisions log) · local `main` synced in xlearn, `../infra` and `../xlearn-evalpack`.

## Risks / watch-outs

- **Spend anomaly.** The provider's **$15 hard limit** is the backstop, with the app cap at $12, the breaker, and the Console alerts at 50%/80% (external to the cluster). No alert reaches the owner otherwise (D34): read `judge admin ledger` and `judge admin ai status` on the first production days.
- **Overfitting the acceptance set.** The test split is touched once per configuration, and `configs_tried` is logged. Tuning happens on dev only. A third test run is never made in-session: after two failures, no tag and ⛔ "needs owner decision" (not a retry).
- **Keys in the session.** The agent never sees the `xlearn-calib` key, the salt or any token: the owner exports the calib key in the session's shell before launch, the harness reads it from the environment, the agent never prints or stores it, and reports carry numbers only.
- **Privacy of the acceptance set.** Owner-written artefacts stay in the private repo. Only aggregate numbers are public.
- **Cents-level ledger checks.** 5% of a few cents is rounding noise, so compare tokens too. The real ±5% test is the monthly check once spend accumulates.
- **Console lag.** The cost view can trail by hours. The Console reads are the owner's, after ship, and the tag and flip don't wait for them.
- **Stale inline JWKS** after a k3s upgrade → 401 → breaker → manual. Task 6 reads the kid check; the fix is the runbook's re-paste.
- **Consent refresh.** Consents apply to the next call: m4-05's gateway calls judge's `POST /internal/accounts/{id}/refresh` after every consent PATCH. The 5-minute cache TTL only bounds a **failed** refresh, which logs an ERROR line. So an unanalyzed first conclusion after ticking the consents is a refresh failure to check (`judge admin ai status`, the logs), not an expected delay.
- **A backlog on re-enable.** If the analyzer durable replayed attempts concluded while AI was off, re-enabling would spend on stale work. Task 2 pins the rule.
- **`LLM_ACCEPT_STD_RETENTION` unset** means `Score` refuses. That's correct for v2.0 content, but it must be set, with D24's attestation, before the first rubric course.
- **The data window isn't a GA gate.** ≥ 2 weeks of M4 data re-sizes `SEAT_CAP` for the v3 opening. GA ([ga-01](sprint-ga-01.md)) doesn't wait for it (rollout §3: no dogfood gate).
- **Don't raise limits** to $100 / $80 in v2 (D25, D35). That's the opening's job.
- **Tag hygiene.** Never move or re-push `v1.16.0`; a dogfood blocker is a `v1.16.x` patch.
