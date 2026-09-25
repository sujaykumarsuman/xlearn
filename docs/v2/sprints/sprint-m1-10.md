# Sprint m1-10 — Coach keys + provider hygiene: key_default(feature), catalog, AEAD AD, keyring re-wrap, store:false, classifier, usage

> **Milestone:** M1 — spine (**M1b** slice, rides `v1.7.0`) · **Track:** product (runs beside [m1-04](sprint-m1-04.md) … [m1-06](sprint-m1-06.md)) · **Order:** 22
> **Prereqs:** [m1-02](sprint-m1-02.md) (`v1.6.0`: `coach.key_default` created and dual-written) · [m1-03](sprint-m1-03.md) (coach `path_slug` / `page_context` edits) · [ds-m1-01](sprint-ds-m1-01.md) (AB01 frozen)
> **Unblocks:** [m1-07](sprint-m1-07.md) (D27, mode gate, L18 and the `v1.7.0` tag)
> **Release action:** **merge only** (ships in `v1.7.0`, tagged by [m1-07](sprint-m1-07.md)) · no infra PR · no flag
> **Calendar:** week 3 (2026-10-12 → 10-16; agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m1-10.md`](../prompts/prompt-m1-10.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Per-feature default keys: `key_default(feature)` the only source; stop writing `is_default` | X | ⬜ |
| 2 | Server model catalog + `GET /api/coach/models`; custom ids; no base URLs | X | ⬜ |
| 3 | Key crypto: AEAD associated data, `kek_id` keyring, background re-wrap | X | ⬜ |
| 4 | Provider hygiene: OpenAI `store:false`, typed errors (`ErrModelAccess`…), usage + `est_cost_micros` | X | ⬜ |
| 5 | Settings keys UI + model switcher from the catalog (AB01 F11–F15) | X | ⬜ |
| 6 | Tests, gateway transit test, doc fixes | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] `v1.6.0` live ([m1-02](sprint-m1-02.md)): `coach.key_default(account_id, feature, key_id, model, updated_at)` exists
      (PK `(account_id, feature)`, `feature ∈ {coach, interview}`, `key_id … ON DELETE CASCADE`), is backfilled from
      `is_default` and dual-written; `api_key_config.is_default` is nullable; readers prefer `key_default`
- [ ] [m1-03](sprint-m1-03.md) merged (coach `path_slug` writes and the `<course>:` `page_context` prefix land first; rebase on them)
- [ ] AB01 frozen: [ds-m1-01](sprint-ds-m1-01.md) merged (the merge is the freeze, D40). This sprint's frames are **F11–F15**: F11 model
      access error, F12 catalog switcher, F13 Settings keys (per-feature defaults, "This month on your keys"), F14
      onboarding step, F15 no-usable-key empty state. F1–F10 belong to [m1-07](sprint-m1-07.md).
- [ ] **File ownership** (so this runs beside m1-04 … m1-06): this sprint owns `internal/coach/**`,
      `internal/platform/secrets/**` (coach is its only importer), `cmd/coach/**` and the coach/settings web files
      (`web/src/components/Coach.tsx`, `CoachModelSwitcher.tsx`, `web/src/lib/settings.ts`, the keys section of
      `web/src/screens/Settings.tsx`, the onboarding coach step in `web/src/screens/Auth.tsx`). Its gateway change is
      **additive only**: a new file `internal/gateway/coach_models.go`, **one** `apiRoutes()` row, its
      `docs/architecture/openapi.yaml` path, and a new test file — rebase over whichever of m1-04 … m1-06 merged first
- [ ] Parallel sessions: no open peer PR touches `internal/coach/**` or adds a coach goose migration (`gh pr list`,
      `git worktree list`, ListAgents)

## Goal

Bring coach's key handling and provider calls to the v2 contract
([ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §7, [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2)):
per-feature default keys (`coach`, `interview`) from `key_default` only, a **server-side model catalog**, AEAD
associated data bound to account and provider under a **`kek_id` keyring** with a background re-wrap, OpenAI
**`store:false`**, **typed provider errors** (quota never disables a key), and **usage / cost capture** — so
[m1-07](sprint-m1-07.md) only adds the D27 assist capture, the mode gate, L18 and the tag. Nothing the owner sees
changes except the model list, the Settings key panel (AB01) and clearer provider-error copy; the rollback floor stays
**1.6.0**.

## Scope

**In**
- `key_default(feature)` becomes the only default source; coach stops reading **and** writing `api_key_config.is_default`
  (dropped by [m1-08](sprint-m1-08.md) in M1c); explicit column lists replace `SELECT *` on `coach.api_key_config`.
- Provider registry + dated model catalog (`GET /models` in coach, `GET /api/coach/models` through the gateway);
  custom ids; strict request decoding (no user base URLs).
- AEAD associated data `xlearn/coach/key/v1|<account_id>|<provider>` on a new sealed pair, `kek_id`, the
  `COACH_MASTER_KEYS` keyring, a background re-wrap job. The **legacy pair is dual-written** so an R-b to `v1.6.0`
  still decrypts every key, and an `ad_src_digest` ties each AD pair to the legacy pair it was sealed beside, so a
  key replaced by `v1.6.0` during an R-b is never answered from a stale AD pair after rolling forward.
- OpenAI `store:false`; finer provider error kinds incl. `ErrModelAccess`; usage + `stop_reason` parsed from both
  streams; tokens and `est_cost_micros` stored per assistant message; a month-to-date usage summary.
- AB01 **F11–F15**: the Settings key panel (per-feature defaults, catalog-driven switcher, custom id "cost unknown",
  "This month on your keys"), the onboarding step, the provider-error `reason` copy and the no-usable-key empty state
  in the coach panel.
- Doc fixes named by t5 §9 (ADR-0007 "gateway transits the PUT", ADR-0020 stale `UNIQUE(account_id)`).

**Out**
- D27 assist capture, the mode gate (`locked`/`attempt`/`review`/`general`, `coach-prompt@2`), L18 caps, the AB01
  coach-panel states (assist confirm, locked, cap reached, cut-short marker) and the `v1.7.0` tag → [m1-07](sprint-m1-07.md).
- Anything that *uses* the `interview` default (the interviewer, derived realtime credentials) → M6a/M6b
  ([m6a-02](sprint-m6a-02.md), [m6b-01](sprint-m6b-01.md)); this sprint only stores and validates it.
- Platform AI, `internal/platform/llm`, the judge ledger → M4 ([m4-01](sprint-m4-01.md)).
- Dropping `is_default` → [m1-08](sprint-m1-08.md). Retiring the legacy sealed pair → two later tag sprints
  ([l-01](sprint-l-01.md) stops reading and writing it, [l-02](sprint-l-02.md) drops it; see Release).
- Rotating the production master key (adding `COACH_MASTER_KEYS` to the SOPS secret) — supported by code, not done here.
- Gateway rate limits and typed 413 (incl. the `LimitReader` at `gateway/coach.go:153,238`) → [m1-05](sprint-m1-05.md).

## Tasks

### 1 · Per-feature default keys [X]

Sources: [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (*Per-feature keys*),
[ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §7, [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3.
- **Queries** (`internal/coach/store/queries/`): new `key_default.sql` — `GetKeyDefault(account, feature)` (join to
  `api_key_config` for the sealed material and `enabled`), `UpsertKeyDefault(account, feature, key_id, model)`,
  `ListKeyDefaults(account)`, `PromoteEarliestKeyDefault(account, feature)`. Rewrite `api_key_config.sql`: every
  `SELECT *` becomes an **explicit column list without `is_default`** (a `SELECT *` would scan a column count that no
  longer matches once M1c drops the column); `UpsertApiKeyConfig` no longer names `is_default` (the column default
  fills it); delete `GetDefaultApiKeyConfig`, `SetDefaultProvider`, `PromoteEarliestDefault`.
- **Store** (`internal/coach/store/store.go`): `GetDefaultKey(ctx, account, feature)`; `SetDefault(ctx, account,
  provider, feature, model)`; `PutKey` makes the account's **first** key its `coach` default (v1 behaviour) in the same
  transaction; `DeleteKey` relies on `ON DELETE CASCADE` and then promotes the earliest remaining key to the `coach`
  default. The `interview` default is **never auto-assigned or auto-promoted**.
- **Handlers** (`internal/coach/handlers.go`): `PUT /keys` body gains optional `feature` (`coach` default | `interview`)
  used with `default: true`. An `interview` default accepts either a catalog model with the `interview_brain`
  capability or a **valid custom id** (task 2's regex; "custom" means absent from the catalog at read time, not a stored flag; shown "cost unknown"). A custom id is allowed
  for the text interviewer but never gets an AI proposal ([t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream),
  T5 §9 key/catalog row; [m6a-02](sprint-m6a-02.md) relies on this). A **known** catalog id without `interview_brain` →
  422 `model_not_interview_capable`; a malformed id → 422 `unknown_model`. `GET /keys` keeps its v1 JSON (`keys[].is_default`,
  `default_provider`) **derived from `key_default(feature='coach')`** (open-tab skew), and adds
  `defaults: {coach: {provider, model} | null, interview: {provider, model} | null}` and `usage_month` (task 4).
  Chat (`handleChat`) resolves the key via `GetDefaultKey(account, "coach")` and the model via `key_default.model`
  (falling back to the key's `default_model`, then the catalog default).
- **Grep test** (`internal/coach/contract_test.go`): fails if `is_default` / `IsDefault` appears anywhere under
  `internal/coach/` except `store/migrations/`, `store/schema.sql` and the sqlc-generated `store/gen/models.go`, or if
  any query in `store/queries/` selects `*` from `coach.api_key_config`. Also **remove coach `is_default` from the
  allowlist in m1-03's `hack/lint-dropped-columns.sh`**, so that gate now covers coach too. Together they are
  [m1-08](sprint-m1-08.md)'s "no reader or writer left" precondition for coach.

### 2 · Server model catalog + `GET /api/coach/models` [X]

Sources: [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (*Providers and catalog*), [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §7.
- **Provider registry** (`internal/coach/providers.go`): `map[string]Provider` built in `NewService`. The handlers
  validate a provider against the Service's registry (`s.providers[p]`, replacing `store.ValidProvider` at
  `handlers.go:114,205`). `store` cannot import `coach` (coach imports store; a cycle), so `store.ValidProvider` is kept
  only as a static mirror of the DB `CHECK (provider IN ('openai','anthropic'))`. A test asserts that the registry's
  keys, `store.ValidProvider` and the CHECK agree. The CHECK stays; widening it is a later expand.
- **Catalog** (`internal/coach/catalog.go`, data in code): per entry `id, provider, label, capabilities
  ⊆ {chat, interview_brain, voice_shell}, price{input_micros_per_mtok, output_micros_per_mtok}, as_of, recommended,
  covered_model` (Covered Models are allowed on BYO, labelled "your provider keeps these chats 30 days"). Entries:
  the v1 list still served by the providers plus **`claude-opus-5-5`, `gpt-6-sol`, `gpt-6-luna`, `gpt-6-astra`**;
  defaults `claude-sonnet-5` (Anthropic) and `gpt-6-sol` (OpenAI, **only after a smoke call succeeds on the owner's
  key**, which he exports as `OPENAI_KEY_SMOKE` before launch (an optional before-launch item); without the key or on a
  failed call, keep `gpt-5.6-sol` and log the decision). Prices are copied from the providers' pricing pages on the day,
  with `as_of` set to that date — never guessed. `voice_shell` entries are provisional until S6 ([spk-04](sprint-spk-04.md);
  ADR-0032 stays Proposed).
- **Validation:** a known id → accepted; an unknown id matching `^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$` → accepted as
  `custom: true`, shown "cost unknown" (no cost estimate); anything else → 422 `unknown_model`. Request bodies are
  decoded with `DisallowUnknownFields`, so a `base_url` (or any other field) → 400 — **no user-supplied base URLs** (SSRF).
- **Endpoint:** coach `GET /models` (JWT-scoped like every coach route) →
  `{as_of, providers: [...], models: [...], defaults: {anthropic, openai}}`.
- **Gateway:** new `internal/gateway/coach_models.go` — `handleCoachModels` (session-gated via `authAccount`, mints the
  coach JWT, passthrough; 503 `unavailable` when coach isn't configured); **one** `apiRoutes()` row
  `{"GET", "/api/coach/models", g.handleCoachModels, true}` in the coach group; the path in
  `docs/architecture/openapi.yaml` (the drift test enforces it). It stays session-gated: the public profile remains the
  only unauthenticated `/api` route. If [m1-06](sprint-m1-06.md)'s route-enumeration test is already on `main`, classify the
  row as withhold-**exempt** with reason "model catalog; no item data".
- **Onboarding** (`web/src/screens/Auth.tsx` coach step, **AB01 F14**): Google is already gone (v1.5.1 / F010,
  pinned by `Auth.test.tsx:308`); keep that test. The provider list now comes from the catalog. Use the pre-M4 copy
  from F14 verbatim (the from-M4 variant is M4's). Also add the t5 §9 spend-limit tip in the step's existing help slot:
  "Tip: set a monthly spend limit on this key at your provider". This is a **deliberate board delta** (see task 5).

### 3 · Key crypto: AEAD associated data, keyring, background re-wrap [X]

Sources: [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (*ADR-0007 fixes* P1),
[ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md), [ADR-0020](../../adr/0020-coach-service-realization-and-behaviour-gate.md) (XChaCha20-Poly1305),
[ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3 (rollback floor).
- **Why a second sealed pair:** a key re-sealed with associated data cannot be opened by `v1.6.0` (it passes `nil`
  AD, `internal/platform/secrets/secrets.go:118,133`). Re-sealing in place would silently raise coach's rollback
  floor to 1.7.0 while `v1.7.0` declares **1.6.0**. So this is an expand: new columns, dual-write, backfill; the
  legacy pair is contracted later.
- **Migration** (`internal/coach/store/migrations/0000N_m1b_key_ad_usage.sql`, next free goose version after m1-02's
  and m1-03's coach files; additive only, no `xlearn:contract` marker):
  `api_key_config.enc_key_ad bytea NULL, enc_data_key_ad bytea NULL, kek_id text NULL, ad_src_digest bytea NULL`;
  the usage columns of task 4.
- **`internal/platform/secrets`:** add a `Keyring` (`active` id + `map[id]*Cipher`), `ParseKeyring(spec)` for
  `COACH_MASTER_KEYS="k1:<base64>,k0:<base64>"` (first entry active; ids `^k[0-9]+$`; each key via `ParseMasterKey`),
  `SealAD(plaintext, ad) (encKey, encDataKey, kekID)` and `OpenAD(encKey, encDataKey, kekID, ad)` binding the same AD
  on **both** AEAD layers (secret under the data key; data key under the KEK). Keep `Seal`/`Open` for the legacy pair.
  Errors stay static (`ErrDecrypt`, new `ErrUnknownKEK`); nothing logs key material.
- **`cmd/coach/main.go`:** `loadCipher` → `loadKeys`: the legacy cipher from `COACH_MASTER_KEY`/`_FILE` (still
  required: it opens the legacy pair); the keyring from `COACH_MASTER_KEYS`/`COACH_MASTER_KEYS_FILE` when set, else
  `{k0: COACH_MASTER_KEY}` active `k0`. **No infra change:** production runs with `k0` = today's key.
- **AD** = `xlearn/coach/key/v1|<account_id>|<provider>` ([ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §7).
- **Binding the pairs (`ad_src_digest`):** `ad_src_digest` = SHA-256 of the legacy `enc_data_key` bytes the AD pair
  was sealed beside. The digest is needed because `v1.6.0` stays a valid R-b target. Its `UpsertApiKeyConfig`
  (`api_key_config.sql:25-41`) rewrites only the legacy pair on a key replace and never touches the AD columns, so
  after rolling forward the AD pair would still hold the **old** key. A revoked old key would then 401, `KindAuth`
  would disable the key the owner just pasted, and a still-valid old key would be billed silently. The envelope seal
  draws a fresh data key every time, so the digest names exactly one legacy seal.
  An AD pair is **current** ⇔ `enc_key_ad IS NOT NULL` **and** (`enc_data_key IS NULL` **or**
  `ad_src_digest = sha256(enc_data_key)`). The `enc_data_key IS NULL` arm is for the later stop-writing tag
  ([l-01](sprint-l-01.md)): every image from `v1.7.0` on then keeps reading the AD pair. Go computes the digest with
  `crypto/sha256`; the re-wrap selector uses PG's built-in `sha256(bytea)`, and a test asserts the two agree.
- **Write (PUT):** seal twice: the legacy pair with `Seal` (unchanged bytes format) **and** the AD pair with `SealAD`
  under the active KEK. Write both pairs, `kek_id` and `ad_src_digest = sha256(new enc_data_key)` in **one**
  `UpsertApiKeyConfig`, so no `v1.7.0`+ write can make them diverge. A `v1.6.0` write during an R-b is detected by the
  digest, not prevented.
- **Read (chat):** current AD pair → `OpenAD` with the row's `kek_id`. Absent (not yet backfilled) or **stale** (digest
  mismatch after a `v1.6.0` write) → legacy `Open`; the next re-wrap pass repairs a stale row. Never fall back from a
  *current but failing* AD pair to the legacy pair, which would defeat the binding: that is `ErrDecrypt`, handled
  exactly as v1 handles a decrypt failure today. Zero plaintext after the provider call.
- **Residual risk until the contract:** while the legacy pair exists, chat still falls back to the unbound legacy
  pair when the AD columns are NULL or the digest mismatches. Anyone with DB write access could force that path and
  swap legacy pairs between accounts. The AEAD binding is **enforced only once the legacy pair is contracted**
  ([l-01](sprint-l-01.md) / [l-02](sprint-l-02.md), Release). This is accepted: v2 is owner-only (D35), and the
  database is not reachable from outside the cluster (MI-5).
- **Background re-wrap** (`internal/coach/rewrap.go`, started from `cmd/coach/main.go` with the service context):
  on start, then every 10 min while work remains, then hourly.
  - **Lock:** each **batch runs in one transaction** that first takes `pg_try_advisory_xact_lock(hashtext('coach.rewrap'))`.
    If it returns false, another runner holds it and the pass ends. Commit or rollback releases the lock on the same
    connection, which makes it safe across a rolling update. Never use a session-level `pg_try_advisory_lock` on the
    pgxpool: the unlock can run on a different pooled connection and leak the lock. The transaction holds one of
    coach's 4 pooled connections ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) L21) only for
    the batch.
  - **Selector:** batches of ≤ 50 rows `WHERE enc_key_ad IS NULL OR kek_id IS DISTINCT FROM $active OR
    ad_src_digest IS DISTINCT FROM sha256(enc_data_key)`.
  - **Source of the key:** a stale row decrypts from the **legacy** pair, which is the newer one; never read a
    stale AD pair. A current row being rotated decrypts from its AD pair (its `kek_id`). A legacy-only row decrypts
    from the legacy pair.
  - **Re-seal and update:** re-seal the AD pair under the active KEK, then
    `UPDATE … SET enc_key_ad, enc_data_key_ad, kek_id, ad_src_digest = sha256($seen) WHERE id = $1 AND enc_data_key = $seen`,
    so a racing PUT wins.
  - **Undecryptable rows:** a row it **cannot decrypt is skipped and reported, never deleted or disabled**. Log a WARN
    `coach rewrap: skipped key config` with `key_config_id` only, and an INFO summary
    `coach rewrap: pass rewrapped=N skipped=M pending=K` (D34: a log line read on demand, not an alert).
- The legacy pair's contract is **not** in M1c. It is recorded for [l-01](sprint-l-01.md) / [l-02](sprint-l-02.md)
  (Release).

### 4 · Provider hygiene: `store:false`, typed errors, usage + cost [X]

Sources: [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (*Usage display and limits*, *ADR-0007 fixes*),
[t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream) (shared typed errors for T6).
- **`store:false`** on every OpenAI request payload (`OpenAIProvider.Stream`), plus
  `stream_options: {include_usage: true}`.
- **Typed errors** (`internal/coach/providers.go`): extend `ProviderErrorKind` beyond v1.5.1's
  `Auth / Limited / Unavailable` to `KindAuth`, `KindQuota` (out of credit, billing, spend/usage limits),
  `KindRateLimited`, `KindModelAccess` (403 `permission_error` on a model, 404 `not_found_error`, `model_not_found`),
  `KindRegion` (`unsupported_country_region_territory`), `KindUnavailable`; sentinels `ErrProviderAuth`, `ErrQuota`,
  `ErrRateLimited`, **`ErrModelAccess`**, `ErrRegion`, with `ErrProviderLimited` kept as an umbrella that matches every
  non-auth account-side kind (v1 callers keep working). Classify by `error.type` / `error.code` first, status second,
  message last (Anthropic's spend-limit 400 is only distinguishable by message — keep `anthropicSpendLimitMessage`).
  **Only `KindAuth` sets `enabled=false`** — quota ≠ disable.
- The chat SSE `error` event keeps its v1 `error` value (`provider_auth` / `provider_limited` / `provider_error`) and
  `message`, and gains `reason ∈ {auth, quota, rate_limit, model_access, region, unavailable}` (the pre-stream 429
  `provider_limited` JSON envelope gains the same `reason`). `Coach.tsx` maps each reason to copy taken **verbatim
  from the frozen AB01**:
  - `quota` and `rate_limit` → F10's single provider-limited line. F10's frame (the key toggle stays on) is reviewed
    by [m1-07](sprint-m1-07.md), which keeps this copy as is.
  - `model_access` → **F11** ("This key can't use `<model>`. Pick another model.") with F11's affordance, which opens
    the catalog switcher.
  - `region` → no frame shows it. Use "Your provider doesn't serve this region." and list it as a board delta (task 5).
- **F15** (no usable key, parity) is this sprint's: `CoachEmptyState` (`Coach.tsx:217`) keeps v1's behaviour and
  routes to Settings, with the board's copy under the new name. No other `Coach.tsx` change: the panel header and FAB
  naming, and F1–F10, are [m1-07](sprint-m1-07.md)'s.
- **Usage:** Anthropic — `message_start.message.usage.input_tokens`, `message_delta.usage.output_tokens`,
  `stop_reason`; OpenAI — the final chunk's `usage.prompt_tokens` / `completion_tokens` and `finish_reason`.
  `StreamResult` gains `Usage{InputTokens, OutputTokens}`.
- **Storage:** the migration of task 3 adds `coach_message.provider text, model text, input_tokens integer,
  output_tokens integer, est_cost_micros bigint, stop_reason text` (all NULL). The assistant row stores them;
  `est_cost_micros` = catalog price × tokens, NULL for custom ids or missing usage.
- **Month to date:** `UsageMonth(account)` (UTC calendar month; `coach_message` ⋈ `coach_thread` on account) →
  `{messages, input_tokens, output_tokens, est_cost_micros, has_unknown_cost}`, returned as `usage_month` in `GET /keys`
  (no extra gateway route). Display only; an estimate.

### 5 · Settings keys UI + model switcher (AB01 F11–F15) [X]

Sources: AB01 (`design-system/screens/v2/AB01-coach-states.html`, frozen by [ds-m1-01](sprint-ds-m1-01.md); this
sprint's frames are **F11–F15**), [rollout §9](../rollout-plan.md#9-artboards-by-milestone),
[t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (*Names in the UI*),
[t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 key/catalog row).
**Copy is taken verbatim from the frozen board.** The plan does not restate it.
- `web/src/lib/settings.ts`: delete the static `COACH_MODELS` list; `useCoachModels()` (react-query, `staleTime` 1 h)
  on `GET /api/coach/models`; types for `defaults` and `usage_month`; `usePutCoachKey` accepts `feature`.
- `web/src/components/CoachModelSwitcher.tsx` (**F12**): models of the active provider from the catalog (name,
  capability tags, price per MTok with "as of <date>", **Recommended**, Covered-Model note); a "Custom model id…" row
  with inline validation and the "cost unknown" chip; no base-URL field. On catalog failure show only the currently
  set model.
- `web/src/screens/Settings.tsx` (keys section, **F13**): the "Your AI coach (your key)" heading, masked keys per
  provider, and per-feature defaults. **Coach** shows the provider and model. **Interview** lists the catalog's
  `interview_brain` models **plus** the "Custom model id…" row, which is shown "cost unknown"
  (task 1). The "This month on your keys" line comes from `usage_month`, and `has_unknown_cost` adds a
  "custom models not estimated" note. `theme.css` tokens and components verbatim; match F11–F15 at 1440 px and 390 px.
- **Deliberate board deltas.** The plan wins on these; list each in the PR and in the decisions log. ds-m1-01's merge
  already froze the board (D40), so bringing the board in line is a follow-up design PR, not a wait.
  1. F13's interview parenthetical becomes `interview_brain` ("needs an interview-capable model"), not
     "realtime-capable". t6 §11 dropped the realtime rule.
  2. The unset interview default reads "Not set", if the board shows no unset state.
  3. F14 gains the t5 §9 spend-limit tip.
  4. The `region` error line (task 4) is added.
- Tests: `Settings.test.tsx` (incl. a custom interview id), a new `CoachModelSwitcher.test.tsx`, `Auth.test.tsx`
  (providers from the catalog; no Google), `Coach.test.tsx` (reason → F10/F11 copy; F15 empty state).

### 6 · Tests, gateway transit test, doc fixes [X]

- **Provider fixtures** (`internal/coach/providers_test.go`): table tests from recorded error bodies per provider and
  class — 401 / `invalid_api_key` / `authentication_error`; `insufficient_quota`, `credit_balance_exhausted`,
  402 `billing_error`, Anthropic spend-limit 400 message, OpenAI spend-limit 429 codes; 429 rate limit; 403
  `permission_error`; 404 `model_not_found` / `not_found_error`; region 403; 500 / overloaded / in-stream error
  frame — assert kind, sentinel and that **only auth disables** the key (handler test).
- **`store:false`:** the fake OpenAI server asserts `"store": false` and `include_usage` on **every** request the
  provider makes (chat, retries, any future path through `Stream`).
- **Crypto:** `SealAD`/`OpenAD` round trip; swapping sealed pairs between two accounts or two providers fails
  (AD binding); unknown `kek_id` → `ErrUnknownKEK`; keyring parse errors.
- **Re-wrap** (`internal/coach/rewrap_test.go` + store integration test on PG 18):
  - legacy-only rows → AD pairs under `k0`;
  - rotation `k0 → k1` re-wraps every row;
  - a corrupted row is skipped, reported and left byte-identical;
  - a PUT racing the job is not clobbered;
  - **stale-pair repair:** rewrite only the legacy pair, as `v1.6.0` does. Chat must open the legacy pair, and the
    next pass must re-derive the AD pair from it (digest matches again);
  - a row with a NULL legacy pair and a current AD pair reads the AD pair;
  - Go's SHA-256 and PG's `sha256(bytea)` agree;
  - **single runner:** two concurrent passes, each batch under `pg_try_advisory_xact_lock`. Exactly one batch runs,
    and afterwards `pg_locks` shows no advisory lock held on any pooled connection.
- **Provider registry:** its keys, `store.ValidProvider` and the migration's `CHECK` list agree.
- **Rollback rehearsal (compose):**
  1. After migrating and re-wrapping, run the **`v1.6.0` coach image** against the schema. It decrypts every key
     through the legacy pair and chat works (floor 1.6.0 holds).
  2. **Then**, still on `v1.6.0`, **replace a key** (PUT a new key for the same provider) and roll forward to this
     branch. Chat must use the **new** key (the fake provider sees it), nothing is disabled, and the next re-wrap pass
     repairs the row.
- **Gateway transit** (`internal/gateway/coach_key_transit_test.go`, new file): `PUT /api/coach/key` with a sentinel key
  through the gateway to a fake coach — the fake receives the sentinel (transit), and it appears in **no** gateway log
  line (capture the `slog` handler at DEBUG, access log included), no response body and no error path (fake coach down
  → 502 without the body). `GET /api/coach/models` passthrough test.
- **e2e:** existing coach handler tests and the SSE chat path unchanged; `go test -race ./...`, `sqlc diff`, web tests.
- **Docs:** [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) gets a dated amendment note ("the gateway transits the
  PUT body in memory, never logging or storing it — test-backed"; AEAD AD + keyring per ADR-0031 §7);
  [ADR-0020](../../adr/0020-coach-service-realization-and-behaviour-gate.md) gets a dated note that keys are per
  (account, provider) since `00003` and defaults per feature since this sprint; `docs/architecture/services.md` coach row
  and `docs/architecture/data-model.md` coach tables updated; `openapi.yaml` `CoachKeyList` gains `defaults` and `usage_month`.

## Acceptance criteria

- [ ] After one re-wrap pass every stored key has a current AD-bound pair under the active KEK. In compose, seeded
      legacy rows are 100 % re-wrapped and a corrupted row is skipped, reported and untouched. Swapped pairs fail to
      decrypt: **AD pairs are written and verified**. The binding is **enforced once the legacy pair is contracted**
      ([l-01](sprint-l-01.md) / [l-02](sprint-l-02.md)); the residual risk is accepted until then.
- [ ] The `v1.6.0` coach image still decrypts every key and serves chat against this sprint's schema (R-b rehearsal).
      A key replaced on `v1.6.0` is the one chat uses after rolling forward, and no key is disabled (digest check).
- [ ] **`store:false` asserted on every OpenAI call**; provider fixture tests green for every error class; only auth
      errors disable a key.
- [ ] `GET /api/coach/models` serves the catalog (id, capabilities, price, `as_of`, recommended default, incl.
      `claude-opus-5-5`, `gpt-6-sol/luna/astra`); custom ids accepted as "cost unknown"; malformed ids → 422; a
      `base_url` field → 400.
- [ ] **No writer or reader of `is_default` left** (grep test in CI); no `SELECT *` on `coach.api_key_config`;
      `key_default(feature)` is the only default source. The `interview` default accepts `interview_brain` catalog
      models or a valid custom id, and rejects known catalog ids without `interview_brain`.
- [ ] Usage + `est_cost_micros` stored on assistant messages. Settings, the switcher, onboarding and the coach empty
      and error states match **AB01 F11–F15** (copy verbatim; deliberate deltas listed in the PR). The coach chat flow,
      including v1.5.1's F010 fixes, is unchanged.
- [ ] The gateway transit test proves the key PUT is forwarded but never logged or echoed.
- [ ] CI green (`sqlc diff`, migration lint, gofmt/vet, `go test -race`, web typecheck/lint/test/build).

## Release

**Merge only — ships in `v1.7.0`**, cut by [m1-07](sprint-m1-07.md) (M1b). Do **not** tag. No infra PR (production keeps
`COACH_MASTER_KEY` as keyring entry `k0`), no new flag, no new pod (the re-wrap is a goroutine in the existing coach
pod; memory-sum unchanged). Rollback floor unchanged by this sprint: **1.6.0** (the legacy pair is dual-written).

**Carried forward** (record in the [`../status.md`](../status.md) decisions log; the register and those sprints must
carry it). The legacy sealed pair (`api_key_config.enc_key` / `enc_data_key`, plus `ad_src_digest`) is contracted in
two steps once the floor is ≥ 1.7.0, which holds after `v1.8.0`. It is **not** on m1-08's M1c list.
1. **[l-01](sprint-l-01.md) → `v1.11.0`** (it already touches coach for the erase consumer): stop **reading and
   writing** the legacy pair. `DROP NOT NULL` on it as an expand; queries stop selecting it; chat and re-wrap use the AD
   pair only. **Precondition:** the re-wrap log shows `pending=0 skipped=0`, so any skipped key is re-pasted by the
   owner first (a before-launch item for the sprint that does this step, D40). Every image from `v1.7.0` on already reads a row with a NULL legacy pair through its AD pair (task 3),
   so R-b inside the floor stays safe.
2. **[l-02](sprint-l-02.md) → `v1.12.0`** (already snapshot-first): drop `enc_key`, `enc_data_key` and
   `ad_src_digest` with `-- xlearn:contract floor=v1.11.0`. This matches the tag timeline's "floor after 1.11.0". From
   here the AEAD binding is enforced.

If either tag moves, the pair moves with the next coach-touching tags that satisfy the same floors.

## Definition of Done

CI green (incl. `sqlc diff`) · PR squash-merged to `main` (no tag; ships in `v1.7.0`) · acceptance criteria met ·
AB01 F11–F15 matched (deltas listed) · statuses updated (this file + [`../status.md`](../status.md): board row, M1 row
stays 🔄) · the legacy-pair contract (l-01 / l-02), the residual-risk note, the AB01 deltas and the `gpt-6-sol`
default decision recorded in the decisions log · ADR-0007/0020 notes added.

## Risks / watch-outs

- **Re-wrap on a key it can't decrypt** — skip and report (`key_config_id` only), never delete, never disable; the
  owner re-pastes the key if it matters.
- **Silent floor raise.** Re-sealing the existing columns with AD would strand an R-b to `v1.6.0`. The dual pair and
  the compose R-b rehearsal are the guard. Never "simplify" it away before the floor passes 1.7.0.
- **Pair drift on the R-b path.** A `v1.6.0` key replace rewrites only the legacy pair. Without `ad_src_digest`,
  chat would answer from the stale AD pair after rolling forward, which disables the new key on a 401 or bills the old
  one. The digest rule, the stale-pair test and rehearsal step 2 are the guard.
- **Advisory lock on a pool.** Use only the transaction-scoped `pg_try_advisory_xact_lock`. A session lock taken
  through pgxpool can be released on a different connection and leak.
- **`SELECT *` after M1c** — sqlc code built against a `SELECT *` breaks when a column disappears; the grep test and
  explicit column lists are m1-08's precondition.
- **Catalog drift** — model ids and prices change (GPT-6 ids are days old; Haiku 4.5 retires ≥ 2026-10-15); `as_of`
  on every entry, display-only estimates, and a model 404 is `ErrModelAccess` (pick another model), never a disable.
- **Gateway rebase** — the one `apiRoutes()` row can conflict with m1-05/m1-06 edits nearby; keep the row in the coach
  group and let the later-merging PR rebase. m1-05 also rewrites `gateway/coach.go`'s body reads — this sprint never
  touches that file.
- **Covered Models on BYO** are allowed (the learner's own org) — label them, don't block (t5 §9 correction).
- **m1-07 builds on this** (`coach_message` columns, error reasons, `GetDefaultKey`): keep names stable once merged.
- **m6a-02 builds on the interview default** (catalog `interview_brain` model or custom id): keep "custom ⇔ not in
  the catalog" derivable from `key_default.model`.
