# Sprint m4-01 — platform/llm extraction + WIF auth + retention policy

> **Milestone:** M4 — platform AI (owner cohort) · **Track:** product · **Order:** 60 (the first M4 build sprint)
> **Prereqs:** [mi-12](sprint-mi-12.md) (MI-14: judge 443 + identity egress, `xlearn-judge-llm` secret, projected token, provider runbook, ADR-0031 Accepted) · [m3-13](sprint-m3-13.md) (M3 exit, `v1.14.0`) · [p-03](sprint-p-03.md) (`v1.15.0` cut) · [ds-m4-01](sprint-ds-m4-01.md) (AB16–AB18 frozen) · input: [spk-03](sprint-spk-03.md) (WIF result in t5 §15)
> **Unblocks:** [m4-02](sprint-m4-02.md) (the Scorer, ledger and caps build on these adapters, credential and policy) · the pod smoke (task 10) runs in [m4-07](sprint-m4-07.md) task 7's after-tag reads, before its task 8 (enable)
> **Release action:** **merge only (ships in `v1.16.0`)**, tagged by [m4-07](sprint-m4-07.md). No infra PR (mi-12 delivered the secret, values, token and egress), no new flag, no migration.
> **Calendar:** December 2026 (after the `v1.15.0` tag; agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m4-01.md`](../prompts/prompt-m4-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Capture coach goldens (before any move) | X | ⬜ |
| 2 | `internal/platform/llm` core: types, raw-HTTP adapters, typed errors, redaction | X | ⬜ |
| 3 | Catalog, dated prices, usage → cost (golden per adapter) | X | ⬜ |
| 4 | Coach refactored onto `platform/llm` — no behaviour change | X | ⬜ |
| 5 | `platform/llm/auth`: WIF token exchange + break-glass key | X | ⬜ |
| 6 | `RetentionPolicy`, platform request builder, workspace pin | X | ⬜ |
| 7 | Guards: golden request shape + credential/config boundary lints | X | ⬜ |
| 8 | judge wiring (dark) + `judge admin llm-smoke` | X | ⬜ |
| 9 | Docs + runbook additions | X | ⬜ |
| 10 | Real-workspace smoke from the judge pod (after `v1.16.0` deploys; runs in m4-07 task 7, before task 8) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> Milestones row M4 → 🔄, Artboards rows AB16–AB18 → frozen). Task 10 can only run after the tag: leave it 🔄
> "runs in m4-07 task 7 (after-tag reads)" when this PR merges; [m4-07](sprint-m4-07.md) records the output here, ticks it and
> sets _Overall_ ✅. (m4-07's entry gate cannot require the smoke: it runs only after m4-07's own tag.)
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-14 done** ([mi-12](sprint-mi-12.md)): judge egress TCP 443 to non-cluster addresses **and** `xlearn-identity` :8081 merged;
      `xlearn-judge-llm` loaded via `envFrom`; `LLM_PLATFORM_ENABLED=false` and the non-secret `LLM_ANTHROPIC_*` / `LLM_KEY_LABEL`
      values set; the projected token (audience `https://api.anthropic.com`, 3600 s) mounted at `/var/run/secrets/anthropic.com/token`;
      provider runbook executed (`ev-provider-runbook`); **[ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) Accepted**
- [ ] **M3 exit live** (`v1.14.0`, [m3-13](sprint-m3-13.md)) **and `v1.15.0` cut** ([p-03](sprint-p-03.md)), so all M4 code ships in
      `v1.16.0` and P/M4 never edit `internal/judge` concurrently
- [ ] **Acceptance set labelled:** ≥ 70 (≥ 40 test + 30 dev) in `xlearn-evalpack` `acceptance/` (`ev-acceptance-set`) —
      [rollout §3](../rollout-plan.md#3-milestone-map) M4 entry
- [ ] **AB16–AB18 frozen** ([ds-m4-01](sprint-ds-m4-01.md) merged by the owner) — [rollout §3](../rollout-plan.md#3-milestone-map) M4 entry
- [ ] **WIF result recorded** ([spk-03](sprint-spk-03.md) → t5 §15): WIF GO with the `check_jti` value, the exchange endpoint and
      field names, access-token lifetime vs rotation interval — **or** the fallback (single-workspace key, 90 days) chosen
- [ ] **Parallel sessions:** no open peer PR or worktree touches `internal/coach/**`, `internal/platform/**` or `internal/judge/**`
      (`gh pr list --state open`, `git worktree list`, ListAgents) — e.g. a `v1.15.x` judge fix; sequence with it

## Goal

Give xLearn **one audited LLM layer** that judge and coach share, with the credential and retention rules platform AI needs
([ADR-0031 §1–§4](../../adr/0031-platform-ai-and-two-tier-keys.md#1-who-holds-what),
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets),
[t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 1):
- **`internal/platform/llm`** — policy-free raw-HTTP adapters (Anthropic Messages, OpenAI chat), typed errors, a dated model
  catalog and price table, usage parsing with golden tests. No SDK, no auto-retries, no `tools` field — by construction.
- **Coach moves onto it with no behaviour change** (the BYO path m1-10 built stays byte-identical on the wire).
- **`platform/llm/auth`** — Workload Identity Federation: exchange the k3s projected token for a short-lived
  `workspace:inference` token, re-exchanging only when the token file's `iat` changes; the break-glass key only if WIF failed.
- **`RetentionPolicy`** replaces "zero retention": Messages API only, no Batch/Files/tools/Covered Models, `retention_class` for
  the ledger, pack material refused unless ZDR or the owner's D24 attestation; the `anthropic-workspace-id` pin.
- **Guards in CI:** a golden request-shape test and a lint that only `internal/judge/ai` may touch the platform credential; the
  coach binary cannot load `LLM_*` configuration.

Everything ships dark in `v1.16.0`: the credential is lazy and `LLM_PLATFORM_ENABLED` stays `false`, so no provider call happens
until [m4-07](sprint-m4-07.md) flips it for the cohort.

## Scope

**In**
- New `internal/platform/llm/` (adapters, errors, redaction, catalog, prices, usage, retention, platform request builder) and
  `internal/platform/llm/auth/` (WIF source, key source).
- Coach refactor: `internal/coach/providers.go` and `internal/coach/catalog.go` become thin views over `platform/llm`; handlers,
  SSE framing, error reasons, `GET /models` and stored usage unchanged.
- judge: `internal/judge/ai/{config,credential}.go` (the **only** importer of `llm/auth`), config wiring in `cmd/judge/main.go`,
  and one admin verb `judge admin llm-smoke`.
- Boundary lints as Go tests (run by CI's `go test ./...`).
- Docs: `docs/architecture/services.md` (judge, coach), additions to `docs/v2/runbooks/platform-ai-provider.md` (mi-12's runbook).

**Out**
- `internal/judge/ai` Scorer, ledger tables, llm lane, L17 caps, breaker, identity internals, the AI admin verbs →
  [m4-02](sprint-m4-02.md). (This sprint returns typed errors; m4-02 decides what opens the breaker.)
- Analyzer wiring, `evaluation_analyzed`, pointer notes, `cmd/judge-eval` → [m4-03](sprint-m4-03.md).
- Provisional/dispute/re-grade → [m4-04](sprint-m4-04.md); consents + allowance → [m4-05](sprint-m4-05.md); UI → [m4-06](sprint-m4-06.md).
- The canary log test, cap sizing, ledger vs Console, the `v1.16.0` tag and `LLM_PLATFORM_ENABLED=true` → [m4-07](sprint-m4-07.md).
- Any infra change (secret, values, projected token, egress) — done in [mi-12](sprint-mi-12.md). A new env var must default to the
  value mi-12's values already imply.
- An OpenAI platform credential or `Complete` on OpenAI (only once a challenger passes calibration, [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)),
  the credential-injecting `xlearn-llm-egress` proxy (trigger-based, [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)),
  GitHub-OIDC WIF for `xlearn-calib` calibration CI — not v2.0.

## Tasks

### 1 · Capture coach goldens (before any move) [X]

The refactor must be provably behaviour-neutral, so record the wire behaviour **first**, on `main`, as testdata:
- `internal/coach/testdata/golden/` — for Anthropic and OpenAI, via the existing `httptest` fake provider servers in
  `internal/coach/providers_test.go`: the exact request method, path, headers (`anthropic-version`, content type; the key header
  **name** only, never a value) and JSON body for: a model that takes effort and one that doesn't; a thread with history; the
  OpenAI payload with `store:false` + `stream_options.include_usage` (m1-10).
- The SSE output the handler writes for a normal reply, a cut-short reply (`stop_reason` max tokens) and **each error class**
  m1-10 defined (`auth`, `quota`, `rate_limit`, `model_access`, `region`, `unavailable`), plus the pre-stream 429 envelope.
- The `GET /models` JSON and the stored `coach_message` usage columns for one exchange on a catalog model, **one on a custom model
  id** and one reply with no usage block — the last two store `est_cost_micros` NULL and the chat succeeds (m1-10's rule).
Commit these tests green on the unmodified code in the first commit of the branch; every later commit must keep them green.

### 2 · `internal/platform/llm` core [X]

Sources: [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 1,
[t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (request parameters),
[t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (provider error mapping),
[t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (logging), [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5).

| File | Content |
|---|---|
| `llm.go` | `Provider{Name(); Complete(ctx, Cred, Req) (Resp, error); Stream(ctx, Cred, Req, sink func(delta string) error) (Resp, error)}`. `Req{Model, System []Block{Text, Cache bool}, Messages []Message, Schema json.RawMessage /* nil = text */, MaxOutputTokens, Effort /* "" = omit */, Thinking *Thinking /* nil = omit */, EndUser /* already pseudonymous */, NoStore}` — **no field for tools, tool choice, MCP servers, containers, temperature, top_p or top_k**. `Resp{Text, JSON, Stop, Usage, ModelResolved, RequestID, WorkspaceID}`. `Cred` carries either an API key (BYO or break-glass) or a bearer from an `auth.Source`, plus a `Label` for the ledger. |
| `anthropic.go` | Messages API only: `POST /v1/messages` (streaming and non-streaming), `anthropic-version: 2023-06-01`, structured output via `output_config.format`; reads the `request-id` and `anthropic-workspace-id` response headers into `Resp`; `ListModels` (`GET /v1/models`) for the smoke. No other endpoint exists in the adapter (no Batch, Files, Skills, Agents). |
| `openai.go` | Chat Completions **streaming** for coach BYO (m1-10's `store:false`, `include_usage`, reasoning-effort gating moved as is). `Complete` returns `ErrUnsupported` (no OpenAI platform use in v2.0). |
| `errors.go` | m1-10's typed errors moved here — `ProviderError{Provider, Kind, Status, Code}`, kinds `Auth`, `Quota`, `RateLimited`, `ModelAccess`, `Region`, `Unavailable`, the sentinels and the `ErrProviderLimited` umbrella — plus the platform kinds judge needs: `InvalidRequest` (any other 400), `Refusal` (`stop_reason: refusal`), `ErrWorkspaceMismatch`, `ErrUnknownPrice`, `ErrUnsupported`; `RetryAfter` parsed from 429/529. Quota includes the workspace-limit 400, the org-limit 400 ("You have reached your specified API usage limits"), 429 `enforced_spend_limit_reached`, 402 `billing_error` and the credit-exhausted message (t5 §6). Classify by type/code, then status, then message; keep m1-10's Anthropic spend-limit matcher. |
| `http.go` | Client constructors: platform clients get `ResponseHeaderTimeout` 30 s and **no retry anywhere in the package** (callers own retries so the 8-call cap and the ledger stay exact); coach keeps its streaming client (no overall timeout, per-call context). |
| `redact.go` | `Redacted` implements `slog.LogValuer` → `[redacted N bytes]`; provider error bodies are read with a ≤ 8 KiB cap, parsed for type/code and **never** echoed into errors or logs. Allowed log fields only: request id, provider `request-id`, workspace id, model, effort, usage, µUSD, latency, outcome enum, label. |

### 3 · Catalog, dated prices, usage → cost [X]

Sources: [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (pinning and deprecation),
[t5 §5](../research/t5-platform-ai.md#5-cost-model) (prices), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)
(usage semantics), [ADR-0031 §3](../../adr/0031-platform-ai-and-two-tier-keys.md#3-models).
- **`catalog.go`** — m1-10's `internal/coach/catalog.go` moved here and extended: `{provider, id, class, label, caps{chat,
  interview_brain, voice_shell, effort, structured, thinking_off}, retention: standard|covered, retire_not_before, as_of,
  recommended, byo_visible, platform_allowed}` (`thinking_off`: Sonnet 5 true; Opus 5.5 false — thinking can't be disabled there,
  [t5 sources](../research/t5-platform-ai.md#sources)). `claude-sonnet-5` (class `sonnet`, `retire_not_before` 2027-06-30) and `claude-opus-5-5`
  (class `opus`, 2027-09-22) are `platform_allowed`; Haiku 4.5 and Fable 5.1 are **not** (ADR-0031 §3); Covered Models stay
  BYO-visible with m1-10's label. Class → pinned id resolution through a *passed calibration row* is m4-02's job; this file only
  holds the facts.
- **`prices.go`** — `PriceTable` with effective dates and a `prices_v` string; µUSD per MTok for input, 5-minute cache write,
  1-hour cache write (2 × input), cache read and output. Values copied from the providers' pricing pages on the day with `as_of`
  set — never guessed (t5 §5 as of 2026-09-24, to re-verify: Sonnet 5 input $2, 5 m write $2.50, cache read $0.20, output $10;
  Opus 5.5 input $4, 5 m write $5, cache read $0.20, output $20).
  `Cost(model, usage, at) (micros, pricesV, error)`; an unknown model or date → `ErrUnknownPrice` (never priced at 0). What a
  caller does with it is the caller's policy: judge refuses the call (m4-02); **coach stores a NULL `est_cost_micros` and never
  refuses a chat** (m1-10's rule for custom model ids and missing usage).
- **`usage.go`** — `Usage{In /* uncached input */, CacheWrite5m, CacheWrite1h, CacheRead, Out /* all billed output incl. thinking */,
  Reasoning /* informational, never priced */}`. Anthropic: `input_tokens`, `cache_creation_input_tokens` (split 5m/1h when the
  response breaks it down), `cache_read_input_tokens`, `output_tokens`. OpenAI: `prompt_tokens − cached_tokens`, `cached_tokens`,
  `completion_tokens` (reasoning tokens informational).
- **Tests** — golden usage → cost tables per adapter from recorded response bodies (plain, cached prefix, 1 h cache write, a
  thinking-heavy reply, a truncated reply); a CI test that **every catalog model has a price** effective today; `ErrUnknownPrice`
  path. Coach's `est_cost_micros` now comes from `llm.Cost` with `ErrUnknownPrice` → NULL (the task 1 goldens, including the
  custom-id exchange, prove the stored numbers — and the NULLs — are unchanged).

### 4 · Coach refactored onto `platform/llm` — no behaviour change [X]

- `internal/coach/providers.go` keeps coach's `Provider` interface for the handlers, implemented by delegating to
  `llm.Anthropic` / `llm.OpenAI` `Stream` with a `Cred` built from the decrypted BYO key (zeroed after the call as today). Coach's
  error names become aliases of the `llm` ones (`type ProviderError = llm.ProviderError`, `ErrProviderAuth = llm.ErrAuth`, …) so
  `handlers.go` compiles unchanged; **only auth disables a key** still holds. Coach maps `llm.ErrUnknownPrice` to a NULL
  `est_cost_micros` — a custom model id or a reply without usage is stored unpriced, never refused.
- `internal/coach/catalog.go` becomes a filtered view (`byo_visible`) over `llm.Catalog`; `GET /models` output unchanged.
- Coach config keeps its own env names (`ANTHROPIC_BASE_URL`, `OPENAI_BASE_URL`, `COACH_MASTER_KEY*`); it gains **no** `LLM_*` field.
- Pass bar: the task 1 goldens, `internal/coach` unit tests, `internal/gateway` coach tests and the e2e lane all green with **no test
  edits** other than import paths.

### 5 · `platform/llm/auth`: WIF token exchange + break-glass key [X]

Sources: [ADR-0031 §2](../../adr/0031-platform-ai-and-two-tier-keys.md#2-credential-and-egress),
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets), the spk-03 result (t5 §15), the Anthropic
[WIF reference](https://platform.claude.com/docs/en/manage-claude/wif-reference) and
[WIF with Kubernetes](https://platform.claude.com/docs/en/manage-claude/wif-providers/kubernetes).
- **`source.go`** — `Source{Cred(ctx) (llm.Cred, error); Mode() string /* wif|key */; Label() string}`.
- **`wif.go`** — config: `LLM_ANTHROPIC_ORG_ID`, `…_WORKSPACE_ID`, `…_SERVICE_ACCOUNT_ID`, `…_FEDERATION_RULE_ID`,
  `LLM_WIF_TOKEN_FILE` (default `/var/run/secrets/anthropic.com/token`, mi-12's mount), `LLM_ANTHROPIC_BASE_URL` (default
  `https://api.anthropic.com`; tests override), `LLM_KEY_LABEL`. Behaviour:
  - Read the projected file (≤ 16 KiB) and decode the JWT payload **without verifying it**, only for `iat`/`exp`/`jti` presence;
    the token itself is never logged, stored or put in an error.
  - Cached access token valid for > 5 min → use it. Otherwise, if the file's `iat` is **newer** than the one last exchanged →
    raw-HTTP `POST /v1/oauth/token` with exactly the request shape spk-03 recorded from the WIF reference (grant type, assertion,
    org / service-account / federation-rule identifiers); cache `{access_token, expires_at, iat}`. If the file has **not**
    rotated, keep the cached token to its expiry and then return `ErrUnavailable{reason: token_not_rotated}` — **never re-present a
    used `jti`**. spk-03's invariant (access-token lifetime > rotation interval, kubelet refresh ≈ 80% of 3600 s) makes that
    state transient; a unit test pins the measured numbers.
  - Boot after an in-place restart: if the exchange answers `jti_reused` (only possible with `check_jti=true`), return
    `ErrUnavailable{reason: jti_reused}` until the file rotates, logged once at WARN — no retry loop. Dormant but tested when
    spk-03 set `check_jti=false`.
  - An exchange 401/403 → `llm.ErrAuth` (m4-02 opens the breaker; the runbook's first suspect is JWKS drift after a k3s
    SA-signing-key rotation).
  - Concurrent callers share one exchange (`singleflight`).
- **`key.go`** — break-glass only: `LLM_ANTHROPIC_API_KEY` (from the SOPS secret, present only on the fallback or a WIF outage) →
  `x-api-key`. **Selection:** key present → key mode with a WARN on every start `platform AI: break-glass key in use (label=…)`;
  else complete WIF config + token file → WIF; else `ErrUnavailable{reason: no_credential}` (judge still starts; AI stays off).
- **Never set or read `ANTHROPIC_API_KEY`** (it would shadow federation in any SDK) — enforced by task 7.
- **Tests** against a fake token endpoint: exchange only on `iat` change; cache reuse; single-flight; `jti_reused`; 401 → `ErrAuth`;
  token and access-token bytes absent from every log line and error (capture the `slog` handler at DEBUG); key-mode selection and
  WARN; missing file → `no_credential`.

### 6 · `RetentionPolicy`, platform request builder, workspace pin [X]

Sources: [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (D24),
[t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) (`RetentionPolicy`, workspace check),
[t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) (request parameters).
- **`retention.go`** — `RetentionPolicy{Class /* provider_std | zdr */, AcceptStd bool}` from `LLM_RETENTION_CLASS` (default
  `provider_std`; `zdr` only once the org has a ZDR agreement) and `LLM_ACCEPT_STD_RETENTION` (default `false`; the owner's D24
  attestation). `AllowsPackMaterial()` is true iff `zdr` or `AcceptStd` — m4-02's `Score` refuses otherwise. `Class` is what m4-02
  writes to every ledger row as `retention_class`.
- **Platform builder** (`request_platform.go`) — Anthropic Messages only: resolved model id, `max_tokens`, `system` blocks with
  `cache_control` on the shared prefix, `messages`, `output_config{effort: <always explicit>, format: <json schema when set>}`,
  `thinking` from the caller's mode — `adaptive` (default) → `{type: "adaptive", display: "omitted"}`, `disabled` →
  `{type: "disabled"}` (the `low` + thinking-off analyzer candidate [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task)
  sweeps and m4-03's harness calls `low-nothink`; m4-02 carries the mode with effort in the calibration tuple / `prompt@v`) —
  `metadata.user_id` (the pseudonym m4-02 computes from `LLM_ENDUSER_SALT`), `inference_geo: "global"`. **Never**
  `temperature`/`top_p`/`top_k`, `tools`, `tool_choice`, `mcp_servers` or `container`, and never adaptive thinking with a visible
  display. Refuses a catalog entry that is not `platform_allowed` or is `retention: covered`, and `disabled` on a model without
  `caps.thinking_off`.
- **Workspace pin** — on every platform 2xx, `anthropic-workspace-id` must equal `LLM_ANTHROPIC_WORKSPACE_ID`, else
  `ErrWorkspaceMismatch` (fail closed; m4-02 opens the breaker). `ModelResolved` and `RequestID` are always returned for the ledger.

### 7 · Guards: golden request shape + boundary lints [X]

Sources: [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) ("Invariant tests").
- **Golden request shape** (`internal/platform/llm/request_golden_test.go`, `testdata/`): the builder's JSON for an analyze-shaped
  request in **both thinking modes** (`adaptive` + `display: "omitted"`, and `disabled`), a score-shaped and a feedback-shaped
  request is byte-compared to goldens (so m4-07 can pick the thinking-off candidate without touching a golden), a `disabled`
  request for Opus 5.5 is refused, and a **top-level key allowlist**
  (`model, max_tokens, system, messages, output_config, thinking, metadata, inference_geo, stream`) rejects anything else —
  `tools`, `tool_choice`, `mcp_servers`, `container`, `temperature`, `top_p`, `top_k` fail by name. A reflect test asserts `Req`
  has no such field.
- **Boundary lints** (`internal/platform/llm/boundary_test.go`, using `go list -deps -json ./...`; they run in CI's `go test`):
  1. only `internal/judge/ai` imports `internal/platform/llm/auth` (the "credential use only from `internal/judge/ai`" lint);
  2. `cmd/coach`'s dependency closure contains neither `internal/platform/llm/auth` nor any `internal/judge/...` package;
  3. no Go source outside `internal/platform/llm/auth` names `LLM_ANTHROPIC_API_KEY` or the token path, and none anywhere uses
     the bare `ANTHROPIC_API_KEY` env var (regex `(^|[^_A-Z])ANTHROPIC_API_KEY`);
  4. **the coach binary cannot load `LLM_*` config:** no `LLM_` getenv under `internal/coach/**` or `cmd/coach/**`, and
     `coach.LoadConfig()` with every `LLM_*` variable set equals the config without them;
  5. no provider SDK in `go.mod` (`github.com/anthropics/…`, `github.com/openai/…`) and no request-reissuing loop in
     `internal/platform/llm` (no SDK-style auto-retries).

### 8 · judge wiring (dark) + `judge admin llm-smoke` [X]

- **`internal/judge/ai/config.go`** — `LoadConfig()` reads every `LLM_*` value mi-12 set (`LLM_PLATFORM_ENABLED` default `false`,
  the ids, label, base URL, token file, retention); non-secret values logged once at start (never the salt or key).
- **`internal/judge/ai/credential.go`** — `NewSource(cfg)`; the package is the only importer of `llm/auth` (task 7 lint). The
  source is **lazy**: nothing exchanges a token at boot, so a judge with `LLM_PLATFORM_ENABLED=false` makes **zero** provider calls.
- **`cmd/judge/main.go`** — builds the AI config and source and hands them to the service (m4-02 attaches the Scorer).
- **`judge admin llm-smoke`** (in the `judge admin` dispatcher from [m3-14](sprint-m3-14.md)) — works regardless of
  `LLM_PLATFORM_ENABLED` and **spends nothing**: pick the source, get a credential (timed), call `GET /v1/models` (timed; allowed
  by `workspace:inference`), check the `anthropic-workspace-id` header when present, confirm `claude-sonnet-5` and
  `claude-opus-5-5` are listed. Print only: mode (`wif|key`), label, token `iat`/`exp`, `expires_in`, latencies, statuses, ✓/✗
  per check; exit non-zero on any ✗; one admin-audit row (verb, result enum; no secret). The definitive workspace pin on Messages
  runs with the first ledgered call (m4-02 path).
- **Tests + compose:** the committed tests use `httptest` fakes (token endpoint, Models API, a fake projected-token file). For the
  compose check, point judge's `LLM_ANTHROPIC_BASE_URL` at a tiny fake service through a `-f docker-compose.yml -f <override>`
  kept in the session scratchpad (the repo's convention for fake providers — never committed), then
  `docker compose run --rm judge admin llm-smoke` must pass; paste the output into the PR.

### 9 · Docs + runbook additions [X]

- `docs/architecture/services.md`: judge row — "holds the platform-AI credential (WIF; break-glass key), `internal/judge/ai`";
  coach row — "BYO keys only, over the shared `internal/platform/llm` adapters".
- `docs/v2/runbooks/platform-ai-provider.md` (created by [mi-12](sprint-mi-12.md)) gains: **Smoke from the judge pod**
  (`ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin llm-smoke'`, expected output); **Break-glass switch**
  (owner writes the key into the SOPS file and bumps `xlearn.dev/llm-rev` → judge logs the key-mode WARN; removing it returns to
  WIF); **An exchange 401** (check `host-verify --cluster` for the JWKS kid drift WARN → re-paste the JWKS per the runbook).
- No new ADR expected (this implements ADR-0031 §1–§4). If spk-03's numbers force a different re-exchange rule than t5 §3, add a
  dated note to [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §2.

### 10 · Real-workspace smoke from the judge pod [X]

Cannot run before the code is deployed, and M4 ships in one tag, so it is **not** an m4-07 entry gate. It runs in
**[m4-07](sprint-m4-07.md) task 7's after-tag reads**, after `v1.16.0` rolls out and **before** task 8 (the
`LLM_PLATFORM_ENABLED=true` infra PR): `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin llm-smoke'` must
pass (WIF mode, exchange latency recorded, Models 200, workspace header matching when present, both models listed). Not a host
script: it is the xlearn verb from task 8 run through `kubectl exec`. **Semantics are this sprint's:** `GET /v1/models` only,
**spends nothing and writes no ledger row** (only its admin-audit row) — m4-02's `llm_call.purpose` has no `smoke` value; the first
ledgered call is the first real analysis after task 8, which also runs the definitive workspace pin on Messages. The output goes
into this sprint file and the status.md decisions log. A failure blocks the flag flip (breaker/manual is the safe state), not the tag.

## Acceptance criteria

- [ ] **Coach behaviour unchanged:** the task 1 goldens (request bytes, SSE output, every error class, `GET /models`, stored usage)
      pass byte-identical after the move; `internal/coach`, gateway coach tests and the e2e lane green with no test edits.
- [ ] Usage → cost goldens per adapter (cache 5m/1h, thinking in output, reasoning never priced); every catalog model has a price
      effective today; unknown model or price refused.
- [ ] Golden request shape and the key allowlist green: no `tools`/`mcp_servers`/`container`/sampling params; explicit effort;
      `thinking` is either `{adaptive, display: "omitted"}` or `{disabled}` (both golden; `disabled` refused where the model can't).
- [ ] Coach's unpriced cases unchanged: a custom model id or a reply without usage stores `est_cost_micros` NULL and the chat succeeds.
- [ ] Boundary lints green in CI: only `internal/judge/ai` imports `llm/auth`; the coach binary cannot load `LLM_*`; no
      `ANTHROPIC_API_KEY`; no provider SDK; no auto-retry.
- [ ] WIF source: exchange only on `iat` change, single-flight, `jti_reused` and 401 handled without loops, no token material in
      any log or error; break-glass key selection with its WARN.
- [ ] `RetentionPolicy.AllowsPackMaterial()` false unless `zdr` or `LLM_ACCEPT_STD_RETENTION=true`; covered / non-platform models
      refused; a workspace-header mismatch returns `ErrWorkspaceMismatch`.
- [ ] `judge admin llm-smoke` passes in compose; **the from-the-pod run against the real workspace passes after `v1.16.0` deploys**
      (dark; no spend, no ledger row; executed in m4-07 task 7's after-tag reads, before task 8, and recorded here).
- [ ] CI green (gofmt, vet, `go test -race ./...`, e2e lane, `sqlc diff` — this sprint adds no migration).

## Release

**Merge only — ships in `v1.16.0`**, cut by [m4-07](sprint-m4-07.md) ("M4 platform AI"). Do **not** tag. No infra PR (the secret,
values, projected token and egress are mi-12's), no new flag (`LLM_PLATFORM_ENABLED` already sits in the values at `false`; this
sprint only reads it), no migration, no new pod or container — the memory sum ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses))
is unchanged. Rollback floor unchanged. Until m4-07, the deployed images run the old code; after it, the new code is dark.

## Definition of Done

CI green (incl. `sqlc diff`) · PR squash-merged to `main` (no tag; ships in `v1.16.0`) · tasks 1–9 ✅ and task 10 🔄 "runs in m4-07
task 7 (after-tag reads)" · acceptance criteria met except the pod smoke, which m4-07 ticks · statuses updated (this file + [`../status.md`](../status.md):
Sprint board row, M4 row 🔄, AB16–AB18 "frozen (PR #, date)", `ev-freeze-ds-m4-01` ✅, [ds-m4-01](sprint-ds-m4-01.md) set ✅) · the
decisions log carries the credential mode and the re-exchange rule.

## Risks / watch-outs

- **Inline JWKS rotation at k3s upgrades.** Inline JWKS has no auto-refresh: a k3s SA-signing-key rotation (upgrade, rebuild,
  `certificate rotate`) makes every exchange 401 → breaker → manual grading. Documented in the runbook; `host-verify --cluster`
  WARNs on kid drift (mi-12); the owner re-pastes. Never "fix" it by switching to the key silently.
- **Coach regression.** This moves the most-used AI surface; the byte-level goldens are captured before the move for that reason.
  Don't regenerate them to make a failing test pass — a diff is a behaviour change. The easy one to miss: `llm.Cost` refuses
  unknown prices, coach must not (NULL cost, chat succeeds).
- **WIF request-shape drift.** Use exactly the shape spk-03 recorded from the WIF reference; the golden and the pod smoke catch a
  change. Don't guess field names.
- **Re-presenting a used `jti`.** Re-exchange only on a newer `iat`; if the file hasn't rotated, run the cached token to expiry and
  go unavailable. A loop that retries the same file is a bug even with `check_jti=false`.
- **Break-glass drift.** Key mode is loud on every start and the key's 90-day expiry is a status.md manual check (D34). Never name it
  `ANTHROPIC_API_KEY`.
- **"No auto-retries" creeping back.** The ledger and the 8-call cap (m4-02) are only exact if the adapters never retry; the lint and
  review enforce it.
- **Cost of a wrong price.** Prices are copied on the day with `as_of`; an unknown price refuses the call rather than recording $0.
- **Parallel names.** [m4-02](sprint-m4-02.md) builds directly on `llm.Req`, the typed errors and `RetentionPolicy`; keep the names stable
  once merged.
