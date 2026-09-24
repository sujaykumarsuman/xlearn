# Prompt — Sprint m4-01 · platform/llm extraction + WIF auth + retention policy

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-01.md`](../sprints/sprint-m4-01.md)   ·   **Milestone:** M4 (ships in `v1.16.0`)   ·   **Prereqs:** [mi-12](../sprints/sprint-mi-12.md) (MI-14), [m3-13](../sprints/sprint-m3-13.md) (`v1.14.0`), [p-03](../sprints/sprint-p-03.md) (`v1.15.0`), [ds-m4-01](../sprints/sprint-ds-m4-01.md) (AB16–AB18 frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m4-01.md`](../sprints/sprint-m4-01.md) — tasks, file layout, lints, acceptance.
- [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) (Accepted by mi-12) — §1 who holds what, §2 credential and egress, §3 models,
  §4 retention (D24), §7 coach; the spike result folded into §2.
- [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) (WIF, `RetentionPolicy`, workspace check, invariant tests),
  [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (request parameters, pinning),
  [t5 §5](../research/t5-platform-ai.md#5-cost-model) (prices), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)
  (usage semantics, provider error mapping), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)
  item 1 (`platform/llm` shape), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (logging), and **t5 §15** (the spk-03
  WIF result: endpoint and field names, `check_jti`, lifetime vs rotation) in [`../research/t5-platform-ai.md`](../research/t5-platform-ai.md).
- The Anthropic [WIF reference](https://platform.claude.com/docs/en/manage-claude/wif-reference) and
  [WIF with Kubernetes](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes) — the exchange request shape.
- [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory sum),
  [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (`LLM_PLATFORM_ENABLED` is a permanent kill switch).
- [m1-10's plan](../sprints/sprint-m1-10.md) tasks 2 and 4 — the catalog, typed errors and usage you are moving; [mi-12's plan](../sprints/sprint-mi-12.md)
  tasks 3 and 5 — the env names, secret and token mount judge already has.
- Code: `internal/coach/{providers,catalog,handlers,config,service}.go`, `internal/coach/providers_test.go`, `cmd/coach/main.go`,
  `internal/judge/**` and `cmd/judge/main.go` (the judge admin dispatcher from [m3-14](../sprints/sprint-m3-14.md)),
  `internal/platform/{httpx,slogx}`, `docker-compose.yml`, `.github/workflows/ci.yml`; `../infra/apps/xlearn-judge.yaml` and
  `../infra/apps/secrets/xlearn-judge-llm.enc.yaml` (read only: names, never values).

## Context

Platform AI (M4) is owner-paid and judge-owned: `internal/judge/ai` will hold the Scorer, ledger and policy ([m4-02](../sprints/sprint-m4-02.md));
`internal/platform/llm` holds the policy-free adapters, catalog and prices that both judge and the BYO coach use. Today those
adapters live inside coach (`internal/coach/providers.go`, `catalog.go`, hardened by m1-10: typed errors, `store:false`, usage and
cost). This sprint lifts them into `internal/platform/llm` without changing coach's behaviour, adds the **WIF** credential
(exchange the k3s projected token for a short-lived `workspace:inference` token) with the break-glass key as the only fallback,
replaces "zero retention" with **`RetentionPolicy`**, and adds the CI guards ADR-0031 requires. mi-12 already put the secret, the
non-secret values (`LLM_PLATFORM_ENABLED=false`), the projected token and the 443 egress on the judge pod; this sprint changes no
infra. Everything is dark and lazy until [m4-07](../sprints/sprint-m4-07.md) tags `v1.16.0` and flips the flag for the cohort.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-14 done: [mi-12](../sprints/sprint-mi-12.md) ✅ in `docs/v2/status.md`; `docs/adr/0031-*` says **Accepted**; read-only
      `ssh vps 'sudo k3s kubectl get pod -n xlearn -l app.kubernetes.io/instance=xlearn-judge -o jsonpath="{.items[0].spec.volumes[*].name}"'`
      lists `anthropic-token`; `ev-provider-runbook` ✅.
- [ ] The latest tag is ≥ `v1.15.0` and < `v1.16.0` (a `v1.15.x` patch is fine; `git ls-remote --tags origin | sort -V | tail`),
      `/xlearn/api/v1/healthz` reports it, and `v1.14.0` exists.
- [ ] `ev-acceptance-set` ✅ in status.md (≥ 70 labelled: ≥ 40 test + 30 dev).
- [ ] AB16–AB18 frozen: [ds-m4-01](../sprints/sprint-ds-m4-01.md)'s PR merged (the three board files are on `main`).
- [ ] t5 §15 (spk-03) exists and states WIF GO + `check_jti` (or the fallback); note the exchange field names and the
      access-token lifetime vs rotation interval.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no peer PR or worktree touches
      `internal/coach/**`, `internal/platform/**` or `internal/judge/**`.

## Do this (in order)

1. **[X] Branch** `feat/m4-01-platform-llm` off an up-to-date `main`.
2. **[X] Goldens first (task 1)** — on the unmodified code, add `internal/coach/testdata/golden/` tests capturing, from the existing
   `httptest` fakes: request method/path/headers (header **names** for credentials, never values) and JSON bodies for both providers
   (effort and no-effort models; history; OpenAI `store:false` + `include_usage`); the SSE output for a normal reply, a cut-short reply
   and every error class (`auth`, `quota`, `rate_limit`, `model_access`, `region`, `unavailable`) plus the pre-stream 429 envelope;
   `GET /models` JSON; stored usage columns for a catalog model, a **custom model id** and a reply with no usage (the last two store
   `est_cost_micros` NULL and succeed). Commit green. **Never regenerate these later** — a diff means behaviour changed.
3. **[X] Core package (task 2)** — create `internal/platform/llm/`: `llm.go` (`Provider`, `Req` with **no** tools/tool-choice/MCP/
   container/temperature/top_p/top_k field, `Resp`, `Cred`, `Block`, `Message`), `anthropic.go` (Messages only, streaming +
   non-streaming, `output_config.format`, `request-id` and `anthropic-workspace-id` into `Resp`, `ListModels`), `openai.go` (chat
   streaming for coach; `Complete` → `ErrUnsupported`), `errors.go` (move m1-10's kinds and sentinels; add `InvalidRequest`,
   `Refusal`, `ErrWorkspaceMismatch`, `ErrUnknownPrice`, `ErrUnsupported`, `RetryAfter`; map the workspace/org-limit 400s, 429
   `enforced_spend_limit_reached`, 402 `billing_error` and credit-exhausted to Quota), `http.go` (platform client with
   `ResponseHeaderTimeout` 30 s; no retries anywhere), `redact.go` (`slog.LogValuer` → `[redacted N bytes]`; error bodies read
   ≤ 8 KiB, parsed, never echoed).
4. **[X] Catalog, prices, usage (task 3)** — move `internal/coach/catalog.go` to `internal/platform/llm/catalog.go` with `class`,
   `caps` (incl. `thinking_off`: Sonnet 5 yes, Opus 5.5 no), `retention`, `retire_not_before`, `byo_visible`, `platform_allowed` (Sonnet 5 and Opus 5.5 platform-allowed; Haiku 4.5 and
   Fable 5.1 not). `prices.go` with effective dates, `prices_v`, µUSD per MTok for input / 5 m write / 1 h write (2×) / cache read /
   output — **copy today's values from the providers' pricing pages and set `as_of`; never invent a number**. `usage.go` per the plan.
   Golden usage → cost tests per adapter; a test that every catalog model has a price effective today.
5. **[X] Coach onto the package (task 4)** — `internal/coach/providers.go` delegates to `llm` `Stream`; coach error names become
   aliases of the `llm` ones; `internal/coach/catalog.go` becomes the `byo_visible` view; coach maps `llm.ErrUnknownPrice` to a
   NULL `est_cost_micros` (never refuses a chat); coach gains no `LLM_*` config. Run the
   step-2 goldens, `go test ./internal/coach/... ./internal/gateway/...` and the e2e lane: **no test edits except import paths**.
6. **[X] WIF + key sources (task 5)** — `internal/platform/llm/auth/{source,wif,key}.go` exactly per the plan: decode the projected
   token's payload without verifying it (only `iat`/`exp`); cached access token if > 5 min left; exchange (raw HTTP `POST /v1/oauth/token`,
   the field names from t5 §15 / the WIF reference) **only when the file's `iat` is newer**; otherwise run the cached token to expiry,
   then `ErrUnavailable{token_not_rotated}`; `jti_reused` → unavailable until rotation, WARN once; 401/403 → `ErrAuth`; `singleflight`.
   Key mode when `LLM_ANTHROPIC_API_KEY` is present (WARN every start), else WIF, else `no_credential`. Tests against a fake token
   endpoint, including a DEBUG-level log capture proving no token/access-token/key bytes appear anywhere.
7. **[X] Retention + builder + pin (task 6)** — `retention.go` (`LLM_RETENTION_CLASS` default `provider_std`, `LLM_ACCEPT_STD_RETENTION`
   default `false`, `AllowsPackMaterial()`), `request_platform.go` (explicit effort; `thinking` from the caller's mode —
   `{adaptive, display: omitted}` by default or `{disabled}` for the `low-nothink` analyzer candidate, refused on a model without
   `caps.thinking_off`; `metadata.user_id`, `inference_geo: global`, `cache_control` on the shared prefix; refuses
   non-`platform_allowed` or `covered` models), and the
   `anthropic-workspace-id` pin on every platform 2xx → `ErrWorkspaceMismatch`. Verify each parameter name against the current API
   reference and record it in the golden.
8. **[X] Guards (task 7)** — `request_golden_test.go` (+ `testdata/`) with goldens for analyze (both thinking modes), score and
   feedback shapes and the top-level key allowlist; a reflect test on `Req`;
   `boundary_test.go` using `go list -deps -json ./...` for the five lints in the plan (only `internal/judge/ai` imports `llm/auth`;
   `cmd/coach` closure excludes `llm/auth` and `internal/judge/...`; no bare `ANTHROPIC_API_KEY`; the coach binary cannot load `LLM_*`;
   no provider SDK and no auto-retry). They run in CI's existing `go test -race ./...` step — no workflow change needed.
9. **[X] judge wiring (task 8)** — `internal/judge/ai/config.go`, `internal/judge/ai/credential.go` (lazy source; no exchange at boot),
   `cmd/judge/main.go` wiring; `judge admin llm-smoke` in m3-14's dispatcher: credential (timed) → `GET /v1/models` (timed) → workspace
   header when present → both models listed; prints mode, label, `iat`/`exp`, `expires_in`, latencies, ✓/✗; non-zero on ✗; one
   admin-audit row. Unit-test it against `httptest` fakes.
10. **[X] Compose check** — with a `-f docker-compose.yml -f <override>` **kept in your scratchpad** (never committed) that points
    `LLM_ANTHROPIC_BASE_URL` at a tiny fake and mounts a fake token file: `docker compose run --rm judge admin llm-smoke` passes; with
    `LLM_PLATFORM_ENABLED=false` and no admin verb, judge makes zero provider requests (the fake's access log is empty). Paste both
    outputs into the PR.
11. **[X] Docs (task 9)** — `docs/architecture/services.md` (judge, coach rows); append the three sections to
    `docs/v2/runbooks/platform-ai-provider.md` (pod smoke, break-glass switch, exchange-401 triage). A dated note in ADR-0031 §2 only if
    the re-exchange rule had to differ from t5 §3.
12. **[X] Verify locally** — `gofmt -l .` empty, `go vet ./...`, `go test -race ./...`, `go test -tags e2e -race ./internal/e2e/...`,
    `sqlc diff` (clean; no migration here), and `npm --prefix web run build` (the embed step CI runs first).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** no schema, no migration, no event, no NATS
  subject in this sprint — so no ACL PR and no consumers-before-producers step. `internal/platform/llm` is shared library code, not a
  service; it holds **no policy** (caps, breaker, consent are m4-02's).
- **judge owns platform AI (ADR-0031 §1):** only `internal/judge/ai` may import `llm/auth`; coach never sees `LLM_*`; BYO keys never do
  platform work. **No SDK, no auto-retries, no `tools` field** — by construction and by lint.
- **Secrets:** never log, store or error-wrap the projected token, the access token, the salt or any key; ids, labels, `iat`/`exp`
  and enums only. Never set or read `ANTHROPIC_API_KEY`. Agents never read the SOPS secret's values.
- **Dark and lazy:** with `LLM_PLATFORM_ENABLED=false` (the production value until m4-07) judge makes **zero** provider calls; the
  credential is fetched on first use only; `judge admin llm-smoke` spends nothing (`GET /v1/models` only).
- **GitOps:** no `kubectl apply`, no infra PR (mi-12 already landed everything); read-only `ssh vps` for the gate checks only. **D34:**
  no alert, push channel or opscheck — the smoke and the runbook are read on demand.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):**
  no new pod or container; nothing here adds steady memory to judge (256 Mi limit) or coach (128 Mi).
- **Coach goldens are the contract:** never regenerate them to make a test pass.
- **Parallel sessions:** before merging, re-check peers' open PRs touching `internal/coach/**`, `internal/platform/**` or `internal/judge/**`;
  rebase last. Record an ADR only after the peers' ADR-number check.

## Deliverables

- `internal/platform/llm/` (`llm.go`, `anthropic.go`, `openai.go`, `errors.go`, `http.go`, `redact.go`, `catalog.go`, `prices.go`,
  `usage.go`, `retention.go`, `request_platform.go`, tests, `testdata/`) and `internal/platform/llm/auth/` (`source.go`, `wif.go`, `key.go`, tests).
- Coach on the shared package (`providers.go`, `catalog.go` as thin views) with the pre-move goldens under `internal/coach/testdata/golden/`.
- `internal/judge/ai/{config,credential}.go`, `cmd/judge/main.go` wiring, `judge admin llm-smoke`.
- Golden request-shape test and boundary lints; docs and runbook additions; compose smoke output in the PR.

## Update status

- [`../sprints/sprint-m4-01.md`](../sprints/sprint-m4-01.md): tasks 1–9 🔄 → ✅ as they land; task 10 🔄 "runs in m4-07 task 7
  (after-tag reads)"; _Overall_ 🔄 until m4-07 records the smoke and ticks task 10.
- [`../status.md`](../status.md): Sprint board row for m4-01; **Milestones** row M4 → 🔄; **Artboards** rows AB16–AB18 → "frozen (PR #, date)"
  and [ds-m4-01](../sprints/sprint-ds-m4-01.md) ✅ on the board (set its tasks 6–7 and _Overall_ ✅ in its sprint file); events —
  `ev-freeze-ds-m4-01` ✅. Flag inventory unchanged (`LLM_PLATFORM_ENABLED` is a permanent kill switch, still `false`).
- **Decisions log:** credential mode shipped (WIF with `check_jti=<value>` or break-glass key), the re-exchange rule (new `iat` only;
  cached token to expiry; never a used `jti`), `platform_allowed` models (Sonnet 5, Opus 5.5), the price table's `as_of`.
- ADRs: none expected; a dated note in ADR-0031 §2 only if the re-exchange rule departs from t5 §3.

## Done when (acceptance)

- [ ] Coach behaviour unchanged: pre-move goldens byte-identical; coach, gateway coach and e2e tests green with no test edits.
- [ ] Usage → cost goldens per adapter; every catalog model priced today; unknown model or price refused.
- [ ] Golden request shape + key allowlist green (no `tools`/`mcp_servers`/`container`/sampling params; explicit effort; thinking
      `{adaptive, display: omitted}` or `{disabled}`, both golden).
- [ ] Coach's custom-id / no-usage exchanges still store `est_cost_micros` NULL and succeed.
- [ ] Boundary lints green: only `internal/judge/ai` imports `llm/auth`; the coach binary cannot load `LLM_*`; no `ANTHROPIC_API_KEY`; no SDK; no auto-retry.
- [ ] WIF source: exchange on `iat` change only, single-flight, `jti_reused` and 401 without loops, no token material in logs/errors; key mode WARNs.
- [ ] `RetentionPolicy` refuses pack material unless `zdr` or `LLM_ACCEPT_STD_RETENTION=true`; covered / non-platform models refused; workspace mismatch → `ErrWorkspaceMismatch`.
- [ ] `judge admin llm-smoke` passes in compose (spends nothing, writes no ledger row); the real-workspace pod smoke is scheduled
      in m4-07 task 7's after-tag reads, before its task 8 (task 10 🔄).
- [ ] CI green incl. `sqlc diff`.

Ship at session end per AGENT.md land-and-sync with **this sprint's release action — merge only**: conventional commits
(`feat(llm): …`, `refactor(coach): …`) with the attribution lines, push, open the PR, wait for CI green (fix-then-merge on failure),
squash-merge, then `git checkout main && git pull`. **Do not tag** — this work ships in `v1.16.0`, which [m4-07](../sprints/sprint-m4-07.md)
cuts; Flux deploys nothing new until then, so the pod smoke (task 10) runs there, in task 7's after-tag reads before task 8.
