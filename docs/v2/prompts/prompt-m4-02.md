# Prompt — Sprint m4-02 · judge ai: Scorer, ledger, llm lane, breaker, caps (L17)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-02.md`](../sprints/sprint-m4-02.md)   ·   **Milestone:** M4 (ships in `v1.16.0`)   ·   **Prereqs:** [m4-01](../sprints/sprint-m4-01.md), [l-03](../sprints/sprint-l-03.md), [mi-12](../sprints/sprint-mi-12.md) (judge → identity egress)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m4-02.md`](../sprints/sprint-m4-02.md) — table columns, error mapping, verbs, acceptance. Follow it exactly.
- [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §1 (judge owns platform AI), §3 (models, pinned catalog), §4 (consent,
  pack material only in Score), §5 (four spend layers, degrade order, breaker, percentage view), §8 / the folded amendments (no push, D34;
  dogfood $15 / $12, D25/D35).
- [ADR-0029 §5](../../adr/0029-judge-contract-and-learning-signal.md#5-ai-interface-handed-to-t5) and [§3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D14 outcomes).
- [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) — cap layers, ledger, admission, headroom, pace gate, per
  call, call cap, resume, sweeper, usage semantics, truncation, degrade order, provider error mapping, abuse controls, learner view.
- [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) — where things live, schema changes,
  the Go validator, the calibration gate, independent re-grade, feedback consistency, injection defences; [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (models, effort, pinning).
- [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5) (the three schemas, quote verification, k-sampling, caps),
  [t4 §9.1–§9.2](../research/t4-judge-contract.md#91-lanes) (lanes, priorities), [t4 §2.3](../research/t4-judge-contract.md#23-admission) (row 6 exemptions),
  [t4 §11.2](../research/t4-judge-contract.md#112-t5-the-ai-interface-no-key-policy).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L10, L17), [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory sum),
  [§6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#6-amendments) (ADR-0031 §5 row: no push channel).
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §7 (owner overrides keyed by account id), §8 (judge admin verbs with
  audit rows), §12 row 13 (what judge reads from identity), §14 (ADR-0016 row: MI-5a is the fence);
  [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (T-2 kill switch, T-3 cohort) and §3 (additive migrations).
- Neighbour plans: [m4-01](../sprints/sprint-m4-01.md) (what `internal/platform/llm` gives you), [m3-06](../sprints/sprint-m3-06.md) (queue,
  lanes, graders, sweeper), [m3-14](../sprints/sprint-m3-14.md) (admission, `judge admin`, audit), [m3-05](../sprints/sprint-m3-05.md) (erase consumer),
  [l-03](../sprints/sprint-l-03.md) (`invite`, `account_consent`), [m4-03](../sprints/sprint-m4-03.md)…[m4-05](../sprints/sprint-m4-05.md) (what builds on you).
- Code: `internal/judge/**` (service, store, migrations, queries, queue, graders, admin, erase consumer), `cmd/judge/main.go`,
  `internal/platform/llm/**`, `internal/identity/{handlers,service}.go` + `store/queries/`, `sqlc.yaml`, `docker-compose.yml`.

## Context

M3 built judge (queue with lanes, `code`/`key` graders, admission, `judge admin`); P added the pilot; [m4-01](../sprints/sprint-m4-01.md)
extracted `internal/platform/llm` (adapters, typed errors, catalog, prices, usage, `RetentionPolicy`, WIF credential) and wired judge's
`LLM_*` config — all dark. This sprint gives judge the platform-AI engine itself: the `Scorer` contract and its strict schemas, the one
append-only µUSD ledger with claim-time admission (no holds), the llm lane with the L17 caps at D25's dogfood values, pace-based shedding,
a breaker that degrades to deterministic pre-fill / manual entry (an in-app badge, never an alert — D34), identity's internal read extended
with role/status/tier/consents, and the `judge admin` AI verbs. It stays behind `LLM_PLATFORM_ENABLED=false` and the owner/tester cohort;
m4-03 wires the analyzer triggers, m4-04 the provisional/dispute flow, m4-05 the routes, m4-06 the UI, and m4-07 tags `v1.16.0`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m4-01](../sprints/sprint-m4-01.md) merged: `internal/platform/llm/`, `internal/platform/llm/auth/`, `internal/judge/ai/{config,credential}.go` exist on `main`.
- [ ] [l-03](../sprints/sprint-l-03.md) merged: identity migrations on `main` create `invite` (with `tier`) and `account_consent`.
- [ ] judge → identity :8081 egress merged in `../infra` (`grep -n 8081 ../infra/apps/xlearn-judge.yaml` after `git -C ../infra pull`), per mi-12 task 4.
- [ ] No other open PR adds a judge migration or touches `internal/judge/**` / `internal/identity/handlers.go`
      (`gh pr list --state open`, `git worktree list`, ListAgents). Note the highest judge goose version on `main`.

## Do this (in order)

1. **[X] Branch** `feat/m4-02-judge-ai` off an up-to-date `main`.
2. **[X] Migration + queries (task 1)** — `internal/judge/store/migrations/000NN_m4_llm.sql` (next free version) creating `llm_call`,
   `ai_sample`, `llm_breaker`, `llm_calibration`, `llm_account_limit` exactly as the plan's table (µUSD `bigint`, enum CHECKs — incl.
   `llm_breaker.reason` `provider_quota|external_spend_or_ledger_bug|repeated_400|auth|workspace_mismatch|model_mismatch|owner` and
   `llm_calibration.thinking` `adaptive|disabled` — the four indexes; **no content column**). Queries in `internal/judge/store/queries/llm.sql`; `sqlc generate`. Extend the `XLEARN_IDENTITY` erase
   consumer: anonymize `llm_call`, delete `ai_sample` (before evaluations) and `llm_account_limit`.
3. **[X] Contract (task 2)** — `internal/judge/ai/scorer.go` (interface, `Admission`, typed errors; `Reserve` refuses non-`course`/`touch`
   contexts); package split `ai/score` (may import the pack view) vs `ai/feedback`, `ai/analyze` (must not) + an import test; embedded
   schemas `score@1`, `feedback@1`, `code_analysis@1` in the portable subset (+ the forbidden-keyword test); `validate.go` (caps, id/ref
   existence, line ranges, rune-safe truncation, path-only errors, one retry); `delimit.go` (random boundary, Unicode tag/bidi/zero-width
   strip-and-flag, role-marker escape, injection heuristic → `review_flag`, refusal → `ErrInvalidOutput` + flag); the HMAC pseudonym from
   `LLM_ENDUSER_SALT`.
4. **[X] Calls (task 3)** — model resolution only through a passed `llm_calibration` row (else `uncalibrated` → manual), its `thinking`
   mode handed to m4-01's builder; resolved-model check; **Score** (RetentionPolicy gate, k = 3 concurrent → 5 on spread ≥ 2, median/majority, quote verification + clamp ≤ 2,
   `low_confidence`, `score_regrade` on the independent Opus 5.5 `medium` config at k = 5 with no dispute-text field, one 1.5× truncation
   retry priced in); **Feedback** (Sonnet 5 `low`, 3,000, verified quotes + bands only, consistency rule, `feedback_unavailable`);
   **Analyze** (no pack, > 16 KiB skipped before the call, effort/`max_tokens` from the calibration row, truncation → unavailable **with no
   retry**, `pass_review` on passes). ≤ 8 calls per evaluation (persisted count), ≤ 60 s per call, ≤ 150 s per job. Register the
   `ai_rubric` grader in m3-06's closed registry (`internal/judge/grader`) as a `composite@1` llm-lane step; property-test that AI never flips a code/key verdict; outcome mapping to
   `inconclusive(budget_exhausted|ai_unavailable)`.
5. **[X] Ledger + admission (task 4)** — `Reserve` at claim over the four UTC scopes with inflight at estimate, worst-case job estimate,
   tier headroom (final 0; counted $0.35 account-day / $2 global-month; advisory 20 % + pace gate), count caps, `budget_blocked` rows;
   per-call inflight → settle (via `llm.Cost`); `ai_sample` resume with ≤ 2 re-claims; the sweeper rule in m3-06's ticker; limits from
   the `LLM_*` env with **D25 dogfood defaults compiled in** (`LLM_APP_CAP_USD=12`, `LLM_PROVIDER_LIMIT_USD=15`, `LLM_DAILY_GUARD_PCT=15`,
   `LLM_ACCOUNT_MONTH_USD=6`, `LLM_ACCOUNT_DAY_USD=1`, `LLM_ANALYSES_PER_DAY=20`, `LLM_FINALS_PER_DAY=6`, `LLM_INPUT_MAX_BYTES=16384`) and
   `llm_account_limit` overrides; `AllowanceFor(account)` returning percent, `resets_at`, `state`, `global_paused` — **never money**.
6. **[X] Lane (task 5)** — llm-lane workers (2; lease 30 s / heartbeat 10 s; per-account advisory lock; P0 re-grade, P3 analyze; ageing);
   queue cap 8 → 429 `queue_full` + `Retry-After` for learner-initiated jobs (re-grade and closes exempt), `analysis_skipped(queue_full)`
   for judge-initiated analyze; 429 `retry-after` / 529 → `run_after` re-queue inside the 8-call cap; the `/internal/*` route-table test
   (zero llm-lane inserts from any internal route).
7. **[X] Breaker (task 6)** — the plan's mapping table exactly (quota kinds → `provider_quota`, or `external_spend_or_ledger_bug` when
   app month-to-date is `< 70 %` of `LLM_PROVIDER_LIMIT_USD`; 3 × 400, auth, workspace mismatch, model mismatch, owner open; retry-after
   re-queue; refusal); degrade stages 1–3;
   the status read gains `ai: off|ok|shedding|paused`; the auto-throttle (≥ 3 flag/refusal/injection hits in 7 days → `ai_disabled`,
   reason `auto_flags`). ERROR/WARN log lines with reason enums — **no push, no alert, no opscheck**.
8. **[X] identity + cache (task 7)** — extend `handleInternalGetAccount` additively with `role`, `status`, `tier` (invite join, default
   `standard`), `consents` (non-withdrawn kinds → `{version, granted_at}`) + a contract test; `internal/platform/consent` kind constants
   (`ai_review_graded`, `ai_review_passing`, `ai_behavioral`; extend an existing kinds package if l-05 already made one);
   `internal/judge/ai/accounts.go` (5-minute TTL, LRU ≤ 1,000, fail closed when identity is unreachable); judge
   `POST /internal/accounts/{id}/refresh` (drops one entry, enqueues nothing). `ai_disabled` stays **judge-owned** in `llm_account_limit`
   — an amendment to Accepted ADR-0033 (§12 row 13 has judge reading it from identity), so it is an owner-confirm item (step 13).
9. **[X] Gating (task 8)** — in `Reserve`, cheapest first: `LLM_PLATFORM_ENABLED` (false → `ErrUnavailable{disabled}` before any row, read
   or credential fetch) → cohort (`platformAICohortOnly = true`: `role ∈ {owner, tester}`) → `status = active` → task consent → not
   `ai_disabled` → breaker → caps.
10. **[X] Admin (task 9)** — in m3-14's `internal/judge/admin/`: `ai status`, `ai-disable|ai-enable <user>`, `llm-limit <user> … | --clear`,
    `ledger --month YYYY-MM [--by …]`, `breaker show|set|clear [--scope runner|llm]` (default `runner` = m3-14's grammar unchanged;
    `--scope llm`: `set [--for <dur>]` opens with reason `owner`, `clear` closes, `--level` refused), `calibration list|record --from <path>|-`
    (`-` = stdin, the production form: `kubectl exec -i … judge admin calibration record --from - < row.json`, since the distroless image
    can't take `kubectl cp`; validates the tuple, refuses `passed=false`); each writes judge's admin-audit row in the same tx + one stderr
    line; `<user>` = account id or username (identity's `by-username` read). **Not** `disputes export` (m4-04).
11. **[X] Tests (task 10)** — the fake-provider suite from the plan (caps at each boundary, breaker + graceful degradation with a deterministic
    evaluation still emitting, the ledger arithmetic golden to the µUSD, Scorer behaviours, gating = zero calls and zero rows, identity
    contract, erase). Then `gofmt -l .`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e -race ./internal/e2e/...`, **`sqlc diff`**.
12. **[X] Compose check** — `docker compose up` with a fake Anthropic service via a `-f docker-compose.yml -f <override>` kept in your
    scratchpad (never committed) and a seeded passing `llm_calibration` fixture row: with `LLM_PLATFORM_ENABLED=false` nothing reaches the
    fake and `llm_call` stays empty; with it `true` for an owner-role account that has the consent rows, a scripted `Analyze` call writes one
    settled row; `judge admin ai status` and `ledger --month` print sane numbers; flip the fake to a workspace-limit 400 and watch the breaker
    open with `external_spend_or_ledger_bug` and `ai: paused`. Paste the outputs into the PR.
13. **[X] Docs** — `docs/architecture/{services,data-model,api}.md`; **create** `docs/runbooks/judge-admin.md` (m3-14 wrote no runbook; format
    of `docs/runbooks/projection-rebuild.md`): running `judge admin` via `kubectl exec` (`-i` for stdin), m3-14's `status|breaker|review-flags`,
    a pointer to m3-13's `docs/runbooks/judge-outbox-republish.md`, and this sprint's AI verbs; point the kill-switch table in
    `docs/v2/runbooks/platform-ai-provider.md` at it. Under [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2)
    row 13 and the §14 ADR-0016 row, a dated **amendment** note (the ADR-0031 §8 pattern): "**Amended by m4-02 (<date>):** `ai_disabled` is
    judge-owned (`judge.llm_account_limit`), set by `judge admin ai-disable` and the auto-throttle; identity serves role, status, tier and
    consents." Put it first under **"Owner to confirm"** in the PR body — it changes an Accepted ADR and the register.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** judge writes only schema `judge`; identity only
  `identity`. judge reads identity over its internal HTTP read (behind MI-5a's fence), never its schema. No service tokens (ADR-0016 as amended).
- **goose + sqlc:** additive migration only (ADR-0034 §3), next free judge version, generated code committed, **`sqlc diff` clean**; never
  run `Down` in production.
- **Money:** µUSD `bigint` everywhere; USD env values converted once; an unknown price refuses the call (never priced at 0). The ledger holds
  numbers and enums only — never prompts, completions, thinking, learner content or pack text (C3/C4, [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)).
- **Pack material only in `Score`**, enforced at compile time; **AI never flips a deterministic verdict**; the dispute text never reaches a model.
- **Kill switch first:** with `LLM_PLATFORM_ENABLED=false` (production until m4-07) judge makes zero provider calls and writes zero ledger rows.
- **No NATS change here** (no stream, subject or consumer) — so no ACL PR and no consumers-before-producers step in this sprint; m4-03 owns them.
- **NetworkPolicy standing rule ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)):**
  the one new in-cluster caller (judge → identity :8081) is covered by mi-12's merged infra PR; add no other caller.
- **D34:** no alert, push channel, healthchecks.io ping or opscheck — breaker reasons are log lines, a badge and `judge admin ai status`.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):**
  no new pod or container; the lane runs in judge's existing 256 Mi — bounded caches (LRU ≤ 1,000), bounded response bodies.
- **GitOps:** no `kubectl apply`, no infra PR in this sprint; read-only `ssh vps` only for the gate check.
- **Parallel sessions:** re-check peers' judge migrations and PRs right before merging; take the next free goose version at rebase. Check
  peers' ADR numbers before any ADR (none expected; the ADR-0033 note is a dated amendment, not a new ADR — and an owner-confirm item).

## Deliverables

- judge migration + `queries/llm.sql` + generated code; erase consumer extended.
- `internal/judge/ai/` (`scorer.go`, `score/`, `feedback/`, `analyze/`, `schemas/`, `validate.go`, `delimit.go`, `ledger.go`, `admission.go`,
  `allowance.go`, `breaker.go`, `gate.go`, `accounts.go`, tests); the `ai_rubric` grader in `internal/judge/grader`; llm-lane workers.
- identity internal read extended + contract test; `internal/platform/consent`.
- judge `POST /internal/accounts/{id}/refresh`; status read `ai` field.
- `judge admin` AI verbs; docs and the new `docs/runbooks/judge-admin.md`; the ADR-0033 §12/§14 amendment note (owner-confirm); compose output in the PR.

## Update status

- [`../sprints/sprint-m4-02.md`](../sprints/sprint-m4-02.md): tasks 🔄 → ✅; _Overall_ ✅ when all ten are.
- [`../status.md`](../status.md): Sprint board row for m4-02; Milestones row M4 stays 🔄; **flag inventory** — add "platform AI cohort-only
  (T-3 code default `platformAICohortOnly`; owner milestone M4; removal at GA, [ga-01](../sprints/sprint-ga-01.md))" beside the permanent kill
  switch `LLM_PLATFORM_ENABLED` (still `false`).
- **Decisions log:** `ai_disabled` is judge-owned (amends ADR-0033 §12 row 13 — **owner to confirm**); T1's `usage_ledger` = `judge.llm_call`; `judge admin disputes export` moved to m4-04 with
  its table; consent kind names; L17 defaults compiled in at D25 dogfood values; `LLM_PROVIDER_LIMIT_USD` feeds only the spend-anomaly check.
- ADRs: no new ADR; the dated "Amended by m4-02" note under ADR-0033 §12 row 13 and the §14 ADR-0016 row.

## Done when (acceptance)

- [ ] Caps enforced in tests with a fake provider: every L17 limit, tier headroom, the pace gate, overshoot ≤ one job.
- [ ] The breaker opens on spend anomaly (a cap error below 70 % of the provider limit → `external_spend_or_ledger_bug`) and on every mapped
      signal, and degrades gracefully: pre-fill / manual still complete; status `ai: paused`; no alert.
- [ ] Ledger arithmetic golden to the µUSD; rows carry key label, resolved model, `provider_request_id`, bucketed usage, `retention_class`; no content.
- [ ] Scorer: k-sampling + escalation, quote verification + clamp, independent re-grade config, `RetentionPolicy` refusal, AI never flips a
      deterministic verdict; Feedback and Analyze can't import pack material.
- [ ] Gating: flag off ⇒ zero calls and zero rows; non-cohort / suspended / no consent / `ai_disabled` ⇒ zero calls; no `/internal/*` route
      enqueues llm work.
- [ ] identity's internal read serves `role`, `status`, `tier`, `consents` additively; judge caches 5 min and can refresh an entry.
- [ ] The `judge admin` AI verbs work with audit rows; erase anonymizes the ledger and deletes samples and limits.
- [ ] CI green incl. `sqlc diff`.

Ship at session end per AGENT.md land-and-sync with **this sprint's release action — merge only**: conventional commits (`feat(judge): …`,
`feat(identity): …`) with the attribution lines, push, open the PR, wait for CI green (fix-then-merge on failure), squash-merge, then
`git checkout main && git pull`. **Do not tag** — this work ships in `v1.16.0`, which [m4-07](../sprints/sprint-m4-07.md) cuts after
[m4-03](../sprints/sprint-m4-03.md)…[m4-06](../sprints/sprint-m4-06.md); nothing deploys until then.
