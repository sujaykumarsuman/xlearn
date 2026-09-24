# Sprint m4-02 — judge ai: Scorer, ledger, llm lane, breaker, caps (L17)

> **Milestone:** M4 — platform AI (owner cohort) · **Track:** product · **Order:** 61
> **Prereqs:** [m4-01](sprint-m4-01.md) (`internal/platform/llm`, `llm/auth`, `RetentionPolicy`, judge AI config) · [l-03](sprint-l-03.md) (`invite.tier`, `account_consent`) · [mi-12](sprint-mi-12.md) (judge → identity :8081 egress) · built on [m3-06](sprint-m3-06.md) (job queue, lanes, graders) and [m3-14](sprint-m3-14.md) (admission, `judge admin`)
> **Unblocks:** [m4-03](sprint-m4-03.md) (analyzer wiring on `Scorer.Analyze` and the ledger) · and through the chain [m4-04](sprint-m4-04.md) (re-grade on `Score`), [m4-05](sprint-m4-05.md) (allowance + consent kinds), [m4-07](sprint-m4-07.md) (`judge admin ledger`, `calibration record`, cap sizing)
> **Release action:** **merge only (ships in `v1.16.0`)**, tagged by [m4-07](sprint-m4-07.md). No infra PR here (the judge → identity egress is mi-12's, already merged), no NATS subject, no new pod.
> **Calendar:** December 2026 (agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m4-02.md`](../prompts/prompt-m4-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Schema: `llm_call` ledger, `ai_sample`, `llm_breaker`, `llm_calibration`, `llm_account_limit` + erase | X | ⬜ |
| 2 | Scorer contract, strict schemas, Go validator, injection defences | X | ⬜ |
| 3 | Score / Feedback / Analyze + the `ai_rubric` grader | X | ⬜ |
| 4 | Ledger + admission (`Reserve`), per-call accounting, resume, sweeper, allowance | X | ⬜ |
| 5 | llm lane + L17 caps | X | ⬜ |
| 6 | Breaker, provider-error mapping, degradation status, auto-throttle | X | ⬜ |
| 7 | identity `/internal/accounts/{id}` fields + judge account cache | X | ⬜ |
| 8 | Gating: `LLM_PLATFORM_ENABLED` + cohort + status + consent | X | ⬜ |
| 9 | `judge admin` AI verbs | X | ⬜ |
| 10 | Tests (fake provider) + docs | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row;
> M4 stays 🔄; flag inventory; decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **[m4-01](sprint-m4-01.md) merged:** `internal/platform/llm` (adapters, typed errors, catalog, prices, usage, `RetentionPolicy`,
      platform builder, workspace pin), `internal/platform/llm/auth` and `internal/judge/ai/{config,credential}.go` on `main`
- [ ] **[l-03](sprint-l-03.md) merged:** identity `invite` (with `tier DEFAULT 'standard'`) and
      `account_consent(account_id, kind, version, granted_at, withdrawn_at)` exist on `main`
- [ ] **judge → identity :8081 egress merged** ([mi-12](sprint-mi-12.md) task 4), so `/internal/accounts/{id}` is reachable from the
      judge pod once `v1.16.0` deploys (ADR-0035 §2 standing rule: the NetworkPolicy lands before the tag that introduces the caller)
- [ ] **Migration numbering:** judge migrations are serialized m4-02 → [m4-03](sprint-m4-03.md) → [m4-04](sprint-m4-04.md) →
      [m4-05](sprint-m4-05.md); this sprint takes the next free goose version after [p-03](sprint-p-03.md)'s
- [ ] **Parallel sessions:** no open peer PR or worktree touches `internal/judge/**`, `internal/identity/handlers.go` or adds a judge
      migration (`gh pr list --state open`, `git worktree list`, ListAgents)

## Goal

Give judge its **platform-AI engine** ([ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §1, §3, §5;
[ADR-0029 §5](../../adr/0029-judge-contract-and-learning-signal.md#5-ai-interface-handed-to-t5);
[t5 §6–§7](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)):
- the **`Scorer{Reserve, Score, Feedback, Analyze}`** contract with strict portable schemas, a Go validator and the injection
  defences — pack material only ever in `Score`, and AI never flips a deterministic verdict;
- **one append-only money ledger** (`judge.llm_call`) with a claim-time admission check and no holds;
- the **llm lane** with the **L17 caps** at D25's dogfood values (provider $15 in the Console, app cap $12 here; $6/month and $1/day
  per account; ≤ 20 analyses and ≤ 6 AI finals a day; > 16 KiB input skipped), pace-based shedding of advisory work and a
  **breaker** that degrades to deterministic pre-fill / manual entry with an in-app badge — never an alert (D34);
- identity's internal account read gains **role, status, tier and consents** (MI-5a is the fence), cached 5 minutes in judge;
- the **`judge admin`** AI verbs that replace the opscheck AI digest (D34);
- everything behind the permanent kill switch **`LLM_PLATFORM_ENABLED`** (default `false`) and the owner/tester **cohort** until GA.

Nothing is learner-visible yet: the analyzer triggers are [m4-03](sprint-m4-03.md)'s, the provisional/dispute flow
[m4-04](sprint-m4-04.md)'s, the routes [m4-05](sprint-m4-05.md)'s and the UI [m4-06](sprint-m4-06.md)'s.

## Scope

**In**
- `internal/judge/ai/**`: Scorer, schemas, validator, delimiting, Score/Feedback/Analyze, ledger, admission, allowance, breaker,
  gating, account client + cache; the `ai_rubric` grader kind on the llm lane (in m3-06's closed registry, `internal/judge/grader`).
- judge goose migration + sqlc queries: `llm_call`, `ai_sample`, `llm_breaker`, `llm_calibration`, `llm_account_limit`; the erase
  consumer extended.
- llm-lane workers on m3-06's `judge.job` queue; L10's llm queue cap 8; L17 caps.
- identity: `handleInternalGetAccount` gains `role`, `status`, `tier`, `consents`; `internal/platform/consent` kind constants.
- judge: `POST /internal/accounts/{id}/refresh` (cache drop; enqueues nothing); the status read gains `ai`.
- `judge admin ai status | ai-disable | ai-enable | llm-limit | ledger | breaker show|set|clear --scope llm | calibration list|record`.

**Out**
- Analyzer triggers (judge durables on `XLEARN_PRACTICE`), the D26 fingerprint, skip rules, `evaluation_analyzed`, pointer-note storage,
  `cmd/judge-eval` and its ACL PRs → [m4-03](sprint-m4-03.md).
- practice `provisional`, 24 h auto-accept, the re-grade route, `evaluation_dispute`, claims and **`judge admin disputes export`** →
  [m4-04](sprint-m4-04.md) (the verb moves with the table it exports; the register listed it here — recorded in the decisions log).
- `PATCH /api/me/consents`, `GET /api/me/ai-allowance` and the onboarding copy → [m4-05](sprint-m4-05.md) (this sprint provides the
  judge-side `AllowanceFor` and the consent kind constants); acceptance-step consents → [l-05](sprint-l-05.md).
- Badges, suggestion cards, Settings → [m4-06](sprint-m4-06.md).
- The canary log test, the acceptance run and cap sizing, ledger vs Console ±5 %, the tag and `LLM_PLATFORM_ENABLED=true` →
  [m4-07](sprint-m4-07.md).
- Any push alert, opscheck digest or healthchecks.io ping — **dropped (D34)**; `judge admin ai status` is the on-demand read.
- Raising limits to $100 / $80 → the v3 opening ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- Score/Feedback as separate jobs, sample-1-first caching, rubric calibration sets → the SD-course milestone (post-v2.0,
  [t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream)).

## Tasks

### 1 · Schema: ledger, samples, breaker, calibration, per-account limits [X]

Sources: [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (ledger),
[t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 4 (calibration),
[ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt)
("judge keys the owner's overrides by account id"), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (additive only).

One goose migration `internal/judge/store/migrations/000NN_m4_llm.sql` (next free version; additive, no `xlearn:contract` marker):

| Table | Columns (money in µUSD `bigint`; numbers and enums only — never prompts, completions, learner content or pack text) |
|---|---|
| **`llm_call`** — the one append-only ledger (T1 §4's placeholder name `usage_ledger` is this table, per t5 §6) | `id uuid` (v7) · `account_id uuid NULL` · `evaluation_id uuid NULL` · `job_id uuid NULL` · `path_slug` · `purpose` (`analyze\|score\|score_regrade\|feedback`) · `tier` (`final\|counted\|advisory`) · `provider` · `model_requested` · `model_resolved NULL` · `effort` · `prompt_v` · `schema_v` · `prices_v` · `key_label` · `retention_class` (`provider_std\|zdr`) · `est_micros` · `in_tok`, `cache_write_tok`, `cache_write_1h_tok`, `cache_read_tok`, `out_tok`, `reasoning_tok` (bucketed usage; reasoning informational) · `cost_micros NULL` · `status` (`inflight\|ok\|invalid_output\|refusal\|truncated\|rate_limited\|budget_blocked\|provider_error\|timeout\|usage_unknown`) · `provider_request_id NULL` · `workspace_id_seen NULL` · `latency_ms NULL` · `created_at` · `settled_at NULL`. CHECKs on every enum. Indexes `(created_at)`, `(account_id, created_at)`, `(evaluation_id)`, partial `(created_at) WHERE status = 'inflight'`. |
| `ai_sample` | `evaluation_id`, `call_id → llm_call`, `parsed jsonb`, `created_at`; PK `(evaluation_id, call_id)`. Class C3; deleted when the evaluation finalizes. |
| `llm_breaker` | `scope text PK` (`global`), `open_until NULL`, `reason` (`provider_quota\|external_spend_or_ledger_bug\|repeated_400\|auth\|workspace_mismatch\|model_mismatch\|owner`; task 6 maps each signal), `opened_at`, `opened_by` (`auto\|owner`). |
| `llm_calibration` | `id`, `task` (`analyze\|score\|feedback`), `rubric_v NULL`, `prompt_v`, `schema_v`, `model`, `effort`, `thinking` (`adaptive\|disabled`, default `adaptive`; m4-03's `low-nothink` = effort `low` + `disabled`), `max_tokens`, `api_version`, `split`, `n_test`, `qwk NULL`, `qwk_lb NULL`, `within1 NULL`, `smd NULL`, `passfail NULL`, `passfail_lb NULL`, `twins_ok NULL`, `category_acc NULL`, `category_acc_lb NULL`, `false_pointer_rate NULL`, `schema_ok`, `trunc_rate`, `p95_ms`, `p95_cost_micros`, `configs_tried`, `passed bool`, `recorded_at`, `recorded_by`. |
| `llm_account_limit` | `account_id PK`, `month_micros NULL`, `day_micros NULL`, `analyses_per_day NULL`, `finals_per_day NULL` (NULL = default), `ai_disabled_at NULL`, `ai_disabled_reason NULL` (`owner\|auto_flags`), `updated_at`. |

- sqlc queries in `internal/judge/store/queries/llm.sql`; `sqlc generate`; commit the output.
- **Erase** — extend judge's `XLEARN_IDENTITY` erase consumer ([m3-05](sprint-m3-05.md)) in the same transaction as its other deletes:
  `UPDATE llm_call SET account_id = NULL, evaluation_id = NULL WHERE account_id = $1`, delete the account's `ai_sample` rows before its
  evaluations go, delete its `llm_account_limit` row. Global totals survive ([t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) "Learner view").

### 2 · Scorer contract, strict schemas, Go validator, injection defences [X]

Sources: [ADR-0029 §5](../../adr/0029-judge-contract-and-learning-signal.md#5-ai-interface-handed-to-t5),
[t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5), [t4 §11.2](../research/t4-judge-contract.md#112-t5-the-ai-interface-no-key-policy),
[t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) items 2, 3 and 8.
- **`internal/judge/ai/scorer.go`** — `Scorer{Reserve(ctx, Admission) (Reservation, error); Score(ctx, ScoreRequest) (ScoreResponse, error);
  Feedback(ctx, FeedbackRequest) (FeedbackResponse, error); Analyze(ctx, AnalyzeRequest) (AnalyzeResponse, error)}`. `Admission{AccountID,
  Context, Purpose, Tier, EvaluationID, JobID}`; **`Reserve` refuses any context other than `course` or `touch`** (mock, arena, run → refused).
  Typed errors `ErrBudgetExhausted{Scope, ResetsAt}`, `ErrUnavailable{Reason}`, `ErrInvalidOutput`. Every response returns the
  resolved `model`, `prompt@v`, `schema@v` and the ledger call ids.
- **Pack isolation at compile time** — Score lives in `internal/judge/ai/score` (the only one that may import the pack view);
  `ai/feedback` and `ai/analyze` must not import it, and their request types have no pack-typed field. An import test enforces it
  ([ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24)).
- **Schemas** `internal/judge/ai/schemas/{score,feedback,code_analysis}@1.json` (embedded) in the portable subset: types, `enum`,
  `const`, `$defs`/`$ref`; optional values as `anyOf:[T,null]`; every field required; `additionalProperties:false`; `band` =
  `enum:[1,2,3,4,5]`; `confidence` = `enum:["low","medium","high"]` (the analyzer precedence "≥ 0.6" becomes "≥ medium"); must-cover and
  red-flag ids opaque (`m1…`, `r1…`); `code_analysis@1` gains `pass_review:{pointer_notes[{note, lines|null}], revisit{suggested,
  reason|null}}|null`. A test fails if `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`, `maxItems` or `minItems > 1` appears.
- **Validator** (`validate.go`) — what the subset can't express: string caps 480 / 240 / 200 / 160; findings ≤ 5, concepts ≤ 3,
  pointer notes ≤ 3; complexity `^O\([^)]{1,24}\)$`; every id/ref exists in the manifest or rubric snapshot (unknown → `""`); line
  ranges inside the submission; no URLs or images; over-long prose cut at a rune boundary; quotes verified or dropped, never cut.
  **Errors carry field paths and ids only, never values.** One retry on invalid output, then `*_unavailable`.
- **Delimiting and injection** (`delimit.go`) — random-boundary delimiting of every learner part; strip **and flag** Unicode tag
  characters (U+E0000–E007F), bidi controls and zero-width characters; escape the boundary prefix and role markers; the heuristic
  ("ignore previous", "score 5", role markers, a band ≥ 4 resting on a single quote under 20 chars) sets `review_flag` (→ `trust=honor`);
  `stop_reason: refusal` → `ErrInvalidOutput` + `review_flag`.
- **Pseudonym** — `metadata.user_id` = base64url(HMAC-SHA256(`LLM_ENDUSER_SALT`, account id)), never the raw id, email or name.

### 3 · Score / Feedback / Analyze + the `ai_rubric` grader [X]

Sources: [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5) (a)–(c),
[t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)
items 4–6, [ADR-0031 §3](../../adr/0031-platform-ai-and-two-tier-keys.md#3-models), D24.
- **Model resolution** — a class resolves to a pinned id **only through a passed `llm_calibration` row** for (task, rubric@v, prompt@v,
  schema@v, model, effort, thinking); none → `ErrUnavailable{uncalibrated}` (→ manual). The row's `thinking` mode is passed to m4-01's
  platform builder. `claude-sonnet-5` is the default everywhere;
  `claude-opus-5-5` at `medium` is the per-rubric escalation and the independent re-grade configuration. A response whose `model` differs
  from the calibrated id stops the call (→ manual) and opens the breaker (`model_mismatch`). Effort is always explicit and part of `prompt@v`.
- **Score** — refuses unless `RetentionPolicy.AllowsPackMaterial()` (m4-01; `zdr` or the owner's `LLM_ACCEPT_STD_RETENTION=true`, D24);
  k = 3 samples issued concurrently, median band per criterion, majority vote on must-cover; a spread ≥ 2 on a required criterion
  escalates to k = 5, a persisting spread sets `low_confidence`. Evidence before band; quote verification (NFKC, strip zero-width,
  collapse whitespace, fold quotes and dashes on both sides, substring match) — unverified quotes dropped, a required criterion with no
  verified evidence clamped to ≤ 2. `purpose = score_regrade` uses the independent configuration at k = 5 and has **no field** for
  dispute text. One retry at 1.5 × `max_tokens` on truncation, priced into admission.
- **Feedback** — Sonnet 5 at `low`, `max_tokens` 3,000; sees the learner parts, public statement, descriptors, Score's bands and
  **verified** quotes, and the public reference after conclusion — never anchors, must-cover lists, exemplars or hidden cases.
  Consistency (no "strength" at band ≤ 2; an "improve" for every criterion ≤ 3) in the validator. Failure → `feedback_unavailable`
  (never blocks conclusion).
- **Analyze** — one call; input = the learner's final submission (> 16 KiB → `analysis_skipped(too_large)` before any call), the public
  statement, and the deterministic verdict class and signals; **no pack material**; effort and `max_tokens` from the passed calibration
  row (provisional `low` / 4,000 until m4-07 records the acceptance run); truncation → `analysis_unavailable(truncated)`, **never
  retried** (learner-controlled input); `pass_review` only on a pass. *Which* conclusions are analyzed (D16/D26, skip rules) is m4-03's.
- **Per evaluation:** ≤ 8 calls (a persisted `COUNT` over `llm_call`, checked before every call), ≤ 60 s per call, ≤ 150 s per job,
  retry grading ≤ 1 per attempt.
- **`ai_rubric` grader** — register the kind in m3-06's grader registry as a `composite@1` step on the **llm** lane (the composite
  re-queues onto it and never holds a runner slot); bands → the step result per the item's thresholds; `graded_by=ai` iff it ran with
  `required` or `weight > 0`; `review_flag` → `trust=honor`. **AI never flips a deterministic verdict**: code and key results are inputs to
  the AI step, never outputs (property test). No v2.0 item carries `ai_rubric`; cover it with an in-memory fixture item.
- **Outcome mapping** — `ErrBudgetExhausted` → `inconclusive(budget_exhausted)`, no retry; provider failure after the lane's re-queues →
  `inconclusive(ai_unavailable)`; practice then moves the attempt to `self_grade_pending` ([m4-04](sprint-m4-04.md)) and touches fall back to
  v1 self-scoring. The loop never blocks.

### 4 · Ledger + admission (`Reserve`), per-call accounting, resume, sweeper, allowance [X]

Sources: [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (admission, headroom, pace gate, per call,
call cap, resume, sweeper, truncation), [ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25),
[ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L17.
- **`Reserve` at job claim** — sum `cost_micros` (inflight rows at `est_micros`) over **global-month, global-day, account-month and
  account-day** (UTC); admit iff sum + worst case(job) ≤ limit − headroom(tier) on every scope, else `ErrBudgetExhausted{scope,
  resets_at}`. **No holds**, so nothing leaks. Worst case = the job's maximum calls × (input estimate + `max_tokens` output) at the
  model's price (+ the 1.5× retry for Score/Feedback). With two workers the overshoot is at most one job (test).
- **Headroom** — `final` 0; `counted` $0.35 at account-day and $2 at global-month; `advisory` 20 % **and** the pace gate: allowed while
  global month-to-date ≤ app cap × min(1, 1.15 × elapsed fraction + 0.05).
- **Count caps** — ≤ 20 analyses and ≤ 6 AI-scored finals per account per UTC day (counted from `llm_call` by purpose, distinct
  evaluations); refusals are written as `budget_blocked` rows at zero cost.
- **Per call** — insert `inflight` at `est_micros` → call → settle from the provider usage: `cost_micros` (always recorded, even above the
  estimate, via m4-01's `llm.Cost`), `model_resolved`, `provider_request_id`, `workspace_id_seen`, `latency_ms`, status, `key_label`,
  `retention_class`, `prices_v`.
- **Resume** — parsed samples persist in `ai_sample`; a re-claimed job resumes from them; at most 2 re-claims per job, then
  `ai_unavailable`; `ai_sample` rows deleted when the evaluation finalizes.
- **Sweeper** (m3-06's single ticker) — an `inflight` row older than its lease + 60 s → `usage_unknown` at the estimate (fails safe).
- **Limits config** (defaults = D25's dogfood L17 values, so no infra PR is needed now; m4-07 sets sized values if they change):
  `LLM_APP_CAP_USD=12`, `LLM_PROVIDER_LIMIT_USD=15` (the Console limit, used only by the anomaly check), `LLM_DAILY_GUARD_PCT=15`,
  `LLM_ACCOUNT_MONTH_USD=6`, `LLM_ACCOUNT_DAY_USD=1`, `LLM_ANALYSES_PER_DAY=20`, `LLM_FINALS_PER_DAY=6`, `LLM_INPUT_MAX_BYTES=16384`;
  per-account overrides from `llm_account_limit`.
- **`AllowanceFor(account)`** → `{used_pct, resets_at, state: ok|low|paused, global_paused}` — percent of the account's month cap
  (`low` ≥ 80 %, `paused` at the cap or when disabled), **never money** (ADR-0031 §5). [m4-05](sprint-m4-05.md) serves it as
  `GET /api/me/ai-allowance`.

### 5 · llm lane + L17 caps [X]

Sources: [t4 §9.1–§9.2](../research/t4-judge-contract.md#91-lanes), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L10, L17),
[t4 §2.3](../research/t4-judge-contract.md#23-admission) row 6.
- **Workers** on m3-06's `judge.job` queue, lane `llm`: 2 workers, lease 30 s / heartbeat 10 s, `FOR UPDATE SKIP LOCKED` claim with the
  per-(account, lane) advisory lock (one running per account), priorities with ageing (P0 re-grade, P3 analyze); `Reserve` runs in the claim.
- **Queue cap 8 (L10)** — a learner-initiated llm job over the cap → 429 `queue_full` + `Retry-After`; the one re-grade and closes are
  exempt (t4 §9.2); judge-initiated analyze jobs over the cap are dropped as `analysis_skipped(queue_full)`, never blocking anything.
- **Provider throttling** — 429 with `retry-after` or 529 → re-queue with `run_after`, still inside the 8-call cap.
- **L17** enforced by task 4 (app cap, daily guard, per-account month/day, analyses/finals per day) and task 3 (16 KiB input, 8 calls).
- **No `/internal/*` route enqueues llm work** — a route-table test drives every `/internal/*` handler against a spy queue and asserts zero
  llm-lane inserts; the only enqueuers are gateway-JWT learner routes ([m4-04](sprint-m4-04.md)'s re-grade, rubric finals) and judge's own
  NATS durables ([m4-03](sprint-m4-03.md)) ([t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) invariant tests).

### 6 · Breaker, provider-error mapping, degradation status, auto-throttle [X]

Sources: [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (error mapping, degrade order),
[ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25) and §8 / [ADR-0035 §6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#6-amendments)
(no push channel, D34).

| Signal (m4-01 typed error) | Action — an ERROR log line with the reason enum + an `llm_breaker` row + the in-app badge; **never a push** |
|---|---|
| Quota: workspace-limit 400, org-limit 400, 429 `enforced_spend_limit_reached`, 402 `billing_error`, credit exhausted | Open the global breaker (`provider_quota`) until the UTC month resets or the owner closes it; `ErrBudgetExhausted`. **Spend anomaly:** if the ledger's app month-to-date is under 70 % of `LLM_PROVIDER_LIMIT_USD`, the reason is `external_spend_or_ledger_bug` instead (someone else is spending on the workspace, or the ledger is wrong) |
| Any other 400 on a locally validated request, 3 in a row | Open (`repeated_400`) |
| 401/403, token-exchange 401, `no_credential`, a persisting `jti_reused` | Open (`auth`); queued jobs stop retrying |
| `ErrWorkspaceMismatch` | Open (`workspace_mismatch`) |
| Resolved model ≠ calibrated id | Stop Score/Analyze (→ manual); open (`model_mismatch`) |
| 429 with `retry-after`, 529 | Re-queue with `run_after` (no breaker) |
| `stop_reason: refusal` | `ErrInvalidOutput` + `review_flag` (no breaker) |
| Owner: `judge admin breaker set --scope llm` | Open (`owner`, `opened_by=owner`) until `--for` expires or `breaker clear --scope llm` |

- **Degrade order** (utilisation = max over the four scopes): stage 1 sheds `advisory` (ahead of pace or > 80 %); stage 2 stops `counted`
  analyzer and Feedback at limit − headroom (deterministic pre-fill and bands/quotes/descriptors remain); stage 3 stops `final` Score at
  the limit ("Wait for xLearn AI" or D14 manual; v1 self-scoring for touches). **Nothing blocks the learning loop** (test).
- **Degradation status** — the judge status read that [m3-09](sprint-m3-09.md)'s degradation badge polls gains `ai: off | ok | shedding |
  paused` (`off` when the flag is off; `paused` when the breaker is open or the app cap is reached).
- **Auto-throttle** — ≥ 3 `review_flag` / refusal / injection hits on one account within 7 days → `llm_account_limit.ai_disabled_at = now()`,
  `ai_disabled_reason = auto_flags`, a WARN line; the owner reviews and re-enables with `judge admin ai-enable` (AB18 F6 is what the
  learner sees).

### 7 · identity `/internal/accounts/{id}` fields + judge account cache [X]

Sources: [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 13,
[§4](../../adr/0033-invite-only-admission-and-owner-admin.md#4-schema-identity-the-adr-0005-expand-rules), §14 (the ADR-0016 row: MI-5a is the fence).
- **identity** — `handleInternalGetAccount` (`internal/identity/handlers.go`) adds, without changing existing fields: `role`, `status`,
  `tier` (`account.invite_id → invite.tier`; `standard` when there is no invite — the owner and CLI testers), and `consents`
  (`{kind: {version, granted_at}}` for rows with `withdrawn_at IS NULL`). Still no email, OAuth or session data. A contract test pins
  the JSON; the review workers' existing tests stay green. No identity migration (the columns and tables exist since M1a and l-03).
- **Consent kinds** — a tiny shared package `internal/platform/consent`: `AIReviewGraded = "ai_review_graded"`,
  `AIReviewPassing = "ai_review_passing"`, `AIBehavioral = "ai_behavioral"` ([ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24)).
  [m4-05](sprint-m4-05.md) writes them and [l-05](sprint-l-05.md) uses them at acceptance (if l-05 already defined a kinds package, extend it).
- **judge client** (`internal/judge/ai/accounts.go`) — the read over judge → identity :8081 (mi-12's egress), an in-process TTL cache of
  **5 minutes** (ADR-0033 §12 #13), LRU-bounded (≤ 1,000 entries, inside judge's 256 Mi). `POST /internal/accounts/{id}/refresh` on judge
  drops one entry and enqueues nothing; [m4-05](sprint-m4-05.md) calls it after `PATCH /api/me/consents` so a withdrawal applies to new
  work at once. identity unreachable → **fail closed** for that job (`ErrUnavailable{identity}`), not the breaker.
- **`ai_disabled` is judge-owned** (`llm_account_limit`), immediate, set by `judge admin ai-disable` and the auto-throttle. **This departs
  from Accepted [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md)** (§12 row 13 and the §14 ADR-0016 row list
  `ai_disabled` among the fields judge reads from identity) and from the M4 register (identity serves it). The reasons: the per-account
  kill switch must be immediate ([t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) kill switch 4; a
  5-minute cache isn't), the auto-throttle is judge-side, judge can't write identity's schema
  ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), and ADR-0033 §8 already puts `ai-disable` in `judge admin`. Record it as
  an **amendment, not a clarification**: a dated "**Amended by m4-02 (<date>)** — `ai_disabled` is judge-owned …" note under ADR-0033 §12
  row 13 and the §14 ADR-0016 row (the pattern of [ADR-0031 §8](../../adr/0031-platform-ai-and-two-tier-keys.md#8-amended-by-t7-2026-09-24)),
  an entry in the status.md decisions log, and an **owner-confirm item** at the top of the PR body. If the owner rejects it, identity
  gains the column in a follow-up and judge reads it through the 5-minute cache plus the refresh hook.

### 8 · Gating: `LLM_PLATFORM_ENABLED` + cohort + status + consent [X]

Sources: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service),
[ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24), [rollout §4 M4](../rollout-plan.md#4-per-milestone-detail) (gating).

Platform AI runs for an account only if **all** hold, checked cheapest first inside `Reserve`:
1. **`LLM_PLATFORM_ENABLED=true`** — the permanent T-2 kill switch, default `false`; `false` returns `ErrUnavailable{disabled}` before
   any ledger row, identity read or credential fetch.
2. **Cohort** — `role ∈ {owner, tester}` while the code default `platformAICohortOnly = true` (T-3); [ga-01](sprint-ga-01.md) flips it at GA.
3. `status = active`.
4. **Consent** for the task — `ai_review_graded` for Score, Feedback and Analyze on a failure; plus `ai_review_passing` for Analyze on a pass;
   plus `ai_behavioral` for behavioral items.
5. Not `ai_disabled`; breaker closed; caps (task 4).

**Day-1 note:** nothing writes consent rows until m4-05/m4-06's Settings toggles (and l-05's acceptance step), so on `v1.16.0` every AI path
is off until the owner ticks the consents — expected; [m4-07](sprint-m4-07.md)'s enable step includes it.

### 9 · `judge admin` AI verbs [X]

Sources: [ADR-0033 §8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec)
("judge adds `breaker`, `ai-disable`, `llm-limit`, `review-flags` and `disputes export` … with its own audit rows"),
[rollout §4 M4](../rollout-plan.md#4-per-milestone-detail) (the on-demand read that replaces the opscheck AI digest, D34),
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) (kill switches).

In m3-14's dispatcher (`internal/judge/admin/`), run as `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin …'`. Every
verb — reads included — writes judge's admin-audit row in the same transaction plus one stderr line; nothing prints a secret or learner
content. `<user>` is an account id or a username (resolved through identity's existing `GET /internal/accounts/by-username/{username}`);
for an email, use `identity admin account list` first.

| Verb | Does |
|---|---|
| `ai status` | The on-demand digest: flag and cohort mode; credential mode and label; breaker state, reason and since; month-to-date vs app cap with a linear forecast; today vs the daily guard; 7-day `ai_unavailable`, invalid-output and truncation rates; `review_flag` count; accounts above 80 % of their month; `usage_unknown` rows; calibration rows in use; `retire_not_before` of the models in use when within 90 days |
| `ai-disable <user> [--reason …]` · `ai-enable <user>` | Set / clear `llm_account_limit.ai_disabled_*` (immediate; kill switch 4) |
| `llm-limit <user> [--month-usd N] [--day-usd N] [--analyses N] [--finals N]` · `llm-limit <user> --clear` | Per-account overrides (e.g. the owner at 2 × a learner) |
| `ledger --month YYYY-MM [--by purpose\|model\|account]` | Totals in USD from `llm_call` (settled rows + `usage_unknown` at estimate) — the number [m4-07](sprint-m4-07.md) compares with the Console (±5 %) |
| `breaker show\|set\|clear [--scope runner\|llm]` | Extends m3-14's `breaker` verb with a scope; `runner` is the default and keeps m3-14's grammar (`set --level auto\|0\|1\|2 --for …`) unchanged. `--scope llm` (kill switch 2): `show` = state, reason, `opened_by`, since, until; `set [--for <dur>]` opens it with reason `owner` (no `--for` = until `clear`; the owner's note goes into the audit row's detail, never the enum); `clear` closes it. `--level` is refused with `--scope llm` |
| `calibration list` · `calibration record --from <path>\|-` | List rows; record a row produced by `cmd/judge-eval` ([m4-03](sprint-m4-03.md)). `-` reads the row JSON on **stdin** — the production form, because the distroless judge image has no shell or `tar` for `kubectl cp`: `ssh vps 'k3s kubectl exec -i -n xlearn deploy/xlearn-judge -- judge admin calibration record --from -' < row.json`. Validates the tuple against the compiled catalog and the binary's `prompt@v`/`schema@v`, refuses a row whose gate failed (`passed=false`), writes the row + audit row. [m4-07](sprint-m4-07.md) uses this verb for the analyzer acceptance row (no separate `calibration import`) |

`disputes export` → [m4-04](sprint-m4-04.md), with the `evaluation_dispute` table it reads.

### 10 · Tests (fake provider) + docs [X]

- **Fake provider** — an `httptest` Anthropic Messages server scripted per test: JSON bodies, usage (plain, cached prefix, 1 h write,
  thinking-heavy, truncated), headers (`request-id`, `anthropic-workspace-id`, resolved `model`) and every error class from task 6.
- **Caps** — each L17 limit refuses exactly at its boundary with the right `scope`/`resets_at`; headroom per tier; the pace gate sheds
  advisory work only when ahead of pace; two workers racing `Reserve` overshoot by at most one job.
- **Breaker** — a workspace-limit 400 while app month-to-date sits at 40 % of the provider limit opens it with
  `external_spend_or_ledger_bug` (at 90 %: `provider_quota`); 402, 3 × 400, 401, workspace and model mismatch each open it with their
  reason; `breaker set --scope llm` opens it as `owner` and `clear` closes it; afterwards Analyze →
  `analysis_unavailable(breaker)`, Score → `inconclusive(ai_unavailable)`, status `ai: paused`, and a deterministic evaluation still
  completes and emits `evaluation_completed` (the loop never blocks).
- **Ledger arithmetic golden** — a table of calls → per-row `cost_micros` and month totals to the µUSD; reasoning never priced; inflight
  at estimate; `usage_unknown` at estimate; every row carries key label, resolved model, `provider_request_id`, bucketed usage and
  `retention_class`; no content column exists (schema test).
- **Scorer** — k = 3 median, escalation to 5, quote verification + clamp, `low_confidence`, the independent re-grade config, the
  `RetentionPolicy` refusal, AI never flips a deterministic verdict (property test), validator caps with path-only errors, Unicode
  strip + `review_flag`, refusal handling, pack-isolation import test, resume from `ai_sample`, the persisted 8-call cap, the sweeper.
- **Gating** — flag off ⇒ zero provider calls **and** zero ledger rows; non-cohort, suspended, no consent, `ai_disabled` ⇒ zero calls;
  the `/internal/*` route-table test; identity down ⇒ fail closed.
- **identity** — the internal read's contract test; review workers unaffected. **Erase** — `llm_call` anonymized, `ai_sample` and
  `llm_account_limit` deleted.
- **Docs** — `docs/architecture/services.md` (judge `ai`, identity's internal read fields), `docs/architecture/data-model.md` (the five
  judge tables), `docs/architecture/api.md` (judge `/internal/accounts/{id}/refresh`); **create** `docs/runbooks/judge-admin.md` (format
  of [`projection-rebuild.md`](../../runbooks/projection-rebuild.md); m3-14 documented its verbs only in `docs/architecture/api.md` and wrote
  no runbook): how to run `judge admin` via `kubectl exec` (`-i` for stdin), m3-14's `status`, `breaker`, `review-flags`, a pointer to
  m3-13's `docs/runbooks/judge-outbox-republish.md`, and this sprint's AI verbs; point the kill-switch table in
  `docs/v2/runbooks/platform-ai-provider.md` at it.

## Acceptance criteria

- [ ] **Caps enforced in tests with a fake provider:** every L17 limit (app cap $12, daily guard 15 %, $6/month and $1/day per account,
      ≤ 20 analyses and ≤ 6 AI finals a day, > 16 KiB skipped, ≤ 8 calls per evaluation), tier headroom, the pace gate, overshoot ≤ one job.
- [ ] **Breaker opens on spend anomaly and degrades gracefully:** a provider cap error below 70 % of the provider limit opens it as
      `external_spend_or_ledger_bug`; every mapped signal opens it; deterministic pre-fill / manual paths still complete; status `ai: paused`;
      no alert of any kind.
- [ ] **Ledger arithmetic golden** to the µUSD; rows carry key label, resolved model, `provider_request_id`, bucketed usage and
      `retention_class`; no content is stored.
- [ ] Scorer contract: k-sampling + escalation, quote verification + clamp, independent re-grade configuration, `RetentionPolicy` refusal,
      AI never flips a deterministic verdict; Feedback and Analyze cannot import pack material.
- [ ] Gating: `LLM_PLATFORM_ENABLED=false` ⇒ zero provider calls and zero ledger rows; non-cohort / suspended / no consent / `ai_disabled`
      ⇒ zero calls; no `/internal/*` route enqueues llm work.
- [ ] identity's `/internal/accounts/{id}` serves `role`, `status`, `tier`, `consents` additively; judge caches it 5 minutes and can refresh one entry.
- [ ] `judge admin ai status | ai-disable | ai-enable | llm-limit | ledger | breaker show|set|clear --scope llm | calibration list|record`
      work with audit rows and print no secret or content; `calibration record --from -` reads stdin; `breaker` without `--scope` behaves
      exactly as m3-14's.
- [ ] Erase anonymizes `llm_call` and deletes the account's `ai_sample` and `llm_account_limit` rows.
- [ ] CI green (gofmt, vet, `go test -race ./...`, e2e lane, **`sqlc diff`**, the migration lint).

## Release

**Merge only — ships in `v1.16.0`**, cut by [m4-07](sprint-m4-07.md). Do **not** tag. What `v1.16.0` needs from outside this PR, checked
by m4-07's release checklist: the judge → identity :8081 egress ([mi-12](sprint-mi-12.md), merged — the ADR-0035 §2 standing rule for the one
new in-cluster caller this sprint introduces). No NATS stream or subject here (so no ACL PR; [m4-03](sprint-m4-03.md) has them). No infra value
change: `LLM_PLATFORM_ENABLED` already sits at `false`, and the L17 defaults are compiled in at D25's dogfood values (m4-07's infra PR sets sized
values only if the acceptance run changes them). The cohort gate is a code default (T-3) flipped by [ga-01](sprint-ga-01.md). No new pod: the llm
lane is two goroutines in the existing judge pod, so the memory sum ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses))
is unchanged — keep bodies bounded to stay inside 256 Mi. Migrations are additive; the rollback floor is unchanged. Rollback: R-a
`LLM_PLATFORM_ENABLED=false` (money already spent is the only irreversible residue, [ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step)).

## Definition of Done

CI green (incl. `sqlc diff`) · PR squash-merged to `main` (no tag; ships in `v1.16.0`) · acceptance criteria met · statuses updated (this
file + [`../status.md`](../status.md): Sprint board row, M4 stays 🔄, flag inventory adds "platform AI cohort-only (T-3 code default; owner
milestone M4; removed at GA by ga-01)" next to the permanent `LLM_PLATFORM_ENABLED`) · decisions log: `ai_disabled` judge-owned
(**owner-confirm**), `usage_ledger` = `llm_call`, `disputes export` moved to m4-04, consent kind names, L17 defaults compiled in · dated
"Amended by m4-02" note under ADR-0033 §12 row 13 and the §14 ADR-0016 row, listed as an owner-confirm item in the PR body.

## Risks / watch-outs

- **Cost-model drift** — tokenizer, thinking or price changes skew the ledger; m4-01's dated prices, `usage_unknown` at estimate, and
  [m4-07](sprint-m4-07.md)'s ledger vs Console ±5 % check are the controls. Never price an unknown model at 0.
- **Tiny dogfood caps** — $12/month with a $2 counted headroom means the pace gate can shed advisory notes early in a bursty month; that
  is by design, visible in `ai status`, and re-sized by m4-07 from the acceptance run.
- **No consent on day 1** — AI looks "off" after `v1.16.0` until the owner ticks the Settings consents; don't "fix" it by defaulting consent on
  (ADR-0031 §4: unticked).
- **Stale consent** — the 5-minute cache; the refresh hook makes withdrawal immediate for new work, and identity-unreachable fails closed.
- **No calibration row, no AI** — Analyze refuses until m4-07 records the acceptance run; seed a fixture row in tests and compose only.
- **Two breakers** — m3-14's runner duty-cycle breaker and the llm breaker must stay distinct scopes in `judge admin breaker`.
- **Money arithmetic** — µUSD `bigint` everywhere; convert USD env values once; round per call, never on totals.
- **Migration collisions** — judge migrations are serialized m4-02 → m4-05; take the next free goose version at rebase (CI fails on a duplicate).
- **Sprint size** — if the session runs long, land tasks 1–8 and 10 first and move task 9 (admin verbs) into a follow-up PR **before**
  [m4-03](sprint-m4-03.md) starts; both are merge-only into `v1.16.0`.
