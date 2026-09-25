# Prompt — Sprint m1-10 · Coach keys + provider hygiene

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-10.md`](../sprints/sprint-m1-10.md)   ·   **Milestone:** M1 (M1b slice, ships in `v1.7.0`)   ·   **Prereqs:** [m1-02](../sprints/sprint-m1-02.md) (`v1.6.0` live), [m1-03](../sprints/sprint-m1-03.md), [ds-m1-01](../sprints/sprint-ds-m1-01.md) (AB01 frozen)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Optional: for the `gpt-6-sol` smoke call (step 4), export your OpenAI key in the shell you launch this session from (`read -s OPENAI_KEY_SMOKE; export OPENAI_KEY_SMOKE`). The session never prints or stores it. Without it, the session keeps `gpt-5.6-sol` as the OpenAI default and logs why (the plan's fallback).

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m1-10.md`](../sprints/sprint-m1-10.md) — tasks, file ownership, acceptance.
- [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §7 *Coach (BYO) changes* (Proposed; its §7 is the coach contract this sprint implements).
- [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) — per-feature keys, providers and catalog, usage and limits, the ADR-0007 fixes table (P1 rows are this sprint; P0 rows shipped in v1.5.1); [t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream) — shared typed errors.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3 — expand → backfill → contract, rollback floor (why the legacy sealed pair is dual-written).
- [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (envelope encryption) and [ADR-0020](../../adr/0020-coach-service-realization-and-behaviour-gate.md) (XChaCha20-Poly1305, `X-Coach-Mode`).
- [rollout §4 M1](../rollout-plan.md#4-per-milestone-detail) (scope T1/T5/T6) and [§9](../rollout-plan.md#9-artboards-by-milestone) (AB01).
- [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream), the amendment row "T5 §9 key and catalog". It governs the `interview` default: text needs `interview_brain`, the "realtime-capable" rule is dropped, and custom ids are allowed for the text interviewer but never get an AI proposal. [m6a-02](../sprints/sprint-m6a-02.md) builds on it.
- AB01, the frozen board `design-system/screens/v2/AB01-coach-states.html`. **Your frames are F11–F15:** F11 model-access error, F12 catalog switcher, F13 Settings keys, F14 onboarding step, F15 no-usable-key empty state; F1–F10 are m1-07's. Take copy **verbatim** from the board. Tokens and components come from [`../../../design-system/theme.css`](../../../design-system/theme.css) and [`../../../design-system/README.md`](../../../design-system/README.md).
- [m1-02's plan](../sprints/sprint-m1-02.md) task 1 (coach) — the exact `key_default` shape you build on.
- Code: `internal/coach/{handlers,providers,prompt,service,config}.go`, `internal/coach/store/{store.go,queries/,migrations/}`,
  `internal/platform/secrets/secrets.go`, `cmd/coach/main.go`, `internal/gateway/{coach.go,bff.go}` (read only, except
  the one route row), `docs/architecture/openapi.yaml`, `web/src/lib/settings.ts`, `web/src/components/{Coach,CoachModelSwitcher}.tsx`,
  `web/src/screens/{Settings,Auth}.tsx`, `../infra/apps/xlearn-coach.yaml` (read only: how `COACH_MASTER_KEY` arrives).

## Context

v1 coach stores one sealed key per (account, provider) with an `is_default` flag, seals with **no associated data**
under one master key, calls OpenAI with retention on by default, and serves a stale model list baked into the SPA.
v1.5.1 already fixed the P0s (typed auth vs limited errors, `max_tokens` 4096 + low effort, the onboarding key step).
`v1.6.0` ([m1-02](../sprints/sprint-m1-02.md)) created `coach.key_default(account_id, feature, key_id, model)` and
dual-writes it. This sprint makes `key_default` the only default source, adds the catalog, the AD-bound key format with
a keyring and re-wrap, `store:false`, finer typed errors and usage capture. It runs in parallel with m1-04 … m1-06
(gateway work) and must stay inside its file ownership. [m1-07](../sprints/sprint-m1-07.md) then adds D27, the mode gate,
L18 and cuts `v1.7.0`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.6.0` live: `curl -s https://projects.sujaykumar.dev/xlearn/api/healthz` reports ≥ 1.6.0, and
      `internal/coach/store/migrations/` on `main` has m1-02's `key_default` migration (PK `(account_id, feature)`,
      `ON DELETE CASCADE`, `is_default` nullable)
- [ ] [m1-03](../sprints/sprint-m1-03.md) merged (`git log origin/main --oneline | grep -i m1-03` or its PR merged)
- [ ] AB01 frozen: [ds-m1-01](../sprints/sprint-ds-m1-01.md) merged (the merge is the freeze, D40; the board file exists on `main`)
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no peer PR or worktree touches
      `internal/coach/**`, `internal/platform/secrets/**`, `cmd/coach/**` or adds a coach migration

## Do this (in order)

1. **[X] Branch** `feat/m1-10-coach-keys` off an up-to-date `main`.
2. **[X] Migration** — `internal/coach/store/migrations/0000N_m1b_key_ad_usage.sql` (next free version after m1-02's
   and m1-03's coach files): `api_key_config.enc_key_ad bytea, enc_data_key_ad bytea, kek_id text, ad_src_digest bytea`;
   `coach_message.provider text, model text, input_tokens integer, output_tokens integer, est_cost_micros bigint,
   stop_reason text` — all nullable, additive only, **no** `xlearn:contract` marker (m1-02's lint enforces it).
3. **[X] Queries + store (task 1)** — new `queries/key_default.sql` (get/upsert/list/promote per feature); rewrite
   `queries/api_key_config.sql` with **explicit column lists** that never name `is_default`; delete the three
   `is_default` queries; `sqlc generate`. Store: `GetDefaultKey(account, feature)`, `SetDefault(account, provider,
   feature, model)`; first key → `coach` default in the same tx; delete relies on the cascade then promotes the
   earliest remaining key for `coach` only; `interview` never auto-assigned. Remove coach `is_default` from the
   allowlist in m1-03's `hack/lint-dropped-columns.sh` (the gate must pass without it).
4. **[X] Catalog (task 2)** — `internal/coach/catalog.go` (id, provider, label, capabilities ⊆ {chat, interview_brain,
   voice_shell}, price per MTok in micros, `as_of`, recommended, `covered_model`) with the v1 models still served plus
   `claude-opus-5-5`, `gpt-6-sol`, `gpt-6-luna`, `gpt-6-astra`. Copy prices from the providers' pricing pages today and
   set `as_of` to today — never invent a number. Default `claude-sonnet-5`; make `gpt-6-sol` the OpenAI default only if
   a smoke call on the owner's key (`OPENAI_KEY_SMOKE`, set before launch; never echoed) succeeds — without the key or on a
   failure, keep `gpt-5.6-sol` and log it for the decisions log. Provider registry map in `coach`.
   The handlers validate against `s.providers[p]`. `store` cannot import `coach` (a cycle), so keep
   `store.ValidProvider` only as a static mirror of the DB CHECK, plus a test that the registry, the mirror and the
   CHECK agree. Custom-id regex `^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$` → `custom:true`; else 422
   `unknown_model`; decode bodies with `DisallowUnknownFields` (a `base_url` → 400). Coach `GET /models`.
5. **[X] Handlers (task 1/2)** — `PUT /keys` optional `feature` with `default:true`. An `interview` default accepts
   either a catalog model with `interview_brain` or a valid custom id ("custom" = not in the catalog; "cost unknown").
   A **known** catalog id without `interview_brain` → 422 `model_not_interview_capable`; a malformed id → 422
   `unknown_model`. `GET /keys` keeps `keys[].is_default` / `default_provider`
   derived from `key_default(coach)`, adds `defaults` and `usage_month`; chat resolves key + model from
   `key_default(coach)`.
6. **[X] Crypto (task 3)** — `internal/platform/secrets`: `Keyring`, `ParseKeyring` (`COACH_MASTER_KEYS="k1:<b64>,k0:<b64>"`,
   first active), `SealAD`/`OpenAD` binding AD `xlearn/coach/key/v1|<account_id>|<provider>` on both layers,
   `ErrUnknownKEK`. `cmd/coach/main.go`: legacy cipher from `COACH_MASTER_KEY` (still required) + keyring from
   `COACH_MASTER_KEYS[_FILE]` or `{k0: COACH_MASTER_KEY}`. `ad_src_digest` (step 2) is the SHA-256 of the legacy
   `enc_data_key` the AD pair was sealed beside.
   - **Why:** `v1.6.0`'s upsert, a valid R-b target, replaces only the legacy pair, which leaves the AD pair stale.
   - **Write:** PUT writes **both** pairs + `kek_id` + `ad_src_digest` in one upsert.
   - **Current AD pair** ⇔ `enc_key_ad` present **and** (legacy `enc_data_key` NULL **or** the digest matches it).
   - **Read:** chat opens a current AD pair. Absent or stale → legacy pair. A current but failing AD pair is
     `ErrDecrypt`, with no fallback.
7. **[X] Re-wrap job (task 3)** — `internal/coach/rewrap.go`: start + every 10 min while pending, then hourly.
   - **Lock:** each batch is one transaction that first takes `pg_try_advisory_xact_lock(hashtext('coach.rewrap'))`
     (false → end the pass). **Never** use a session-level `pg_try_advisory_lock` on the pgxpool: its unlock can land
     on another pooled connection and leak.
   - **Selector:** batches of 50 where `enc_key_ad IS NULL OR kek_id IS DISTINCT FROM $active OR ad_src_digest IS
     DISTINCT FROM sha256(enc_data_key)`.
   - **Source:** stale rows decrypt from the **legacy** pair.
   - **Update:** optimistic `UPDATE … SET …, ad_src_digest = sha256($seen) WHERE id=$1 AND enc_data_key=$seen`.
   - **Failures:** on a decrypt failure, skip and WARN (`key_config_id` only); **never delete or disable**. INFO
     summary per pass; zero plaintext.
8. **[X] Provider hygiene (task 4)** — OpenAI payload `store:false` + `stream_options.include_usage`; kinds
   `Auth/Quota/RateLimited/ModelAccess/Region/Unavailable`, sentinels incl. **`ErrModelAccess`**, `ErrProviderLimited`
   as umbrella; only auth disables; the SSE `error` event (and the pre-stream 429 envelope) keeps its v1 `error`/`message` and adds `reason`; parse usage + stop reason from both streams;
   store provider/model/tokens/`est_cost_micros`/`stop_reason` on the assistant row; `UsageMonth(account)` (UTC month).
9. **[X] Gateway (additive only)** — new `internal/gateway/coach_models.go` (`handleCoachModels`, session-gated,
   passthrough, 503 when coach unset); **one** `apiRoutes()` row in the coach group; the `/coach/models` path +
   `CoachKeyList.defaults/usage_month` in `docs/architecture/openapi.yaml`. If m1-06's withhold policy field exists on
   `apiRoute`, mark this row exempt: "model catalog; no item data". Touch no other gateway file.
10. **[X] Web (task 5)**, AB01 **F11–F15** with copy verbatim from the frozen board:
    - `lib/settings.ts`: `useCoachModels`, drop `COACH_MODELS`, `feature` on PUT, new types.
    - `CoachModelSwitcher.tsx` (F12): catalog, custom id + "cost unknown", failure fallback.
    - `Settings.tsx` keys section (F13): naming, per-feature defaults, where Interview lists the `interview_brain`
      models **and** the "Custom model id…" row, and "This month on your keys".
    - `Coach.tsx`: the error `reason` → copy map (`quota`/`rate_limit` → F10's line, `model_access` → F11) **and**
      F15's no-usable-key empty state (`CoachEmptyState`). The panel header, FAB and F1–F10 stay m1-07's.
    - `Auth.tsx` coach step (F14): providers from the catalog, F14's pre-M4 copy + the t5 §9 spend-limit tip.
    - **Deliberate deltas** go in the PR and the decisions log: `interview_brain` ("needs an interview-capable model")
      instead of F13's "realtime-capable", the unset "Not set" state if the board lacks it, the F14 spend-limit tip,
      and the `region` line.
    - Screenshot 1440 px and 390 px next to AB01.
11. **[X] Tests (task 6)** — provider fixture table per error class; `store:false` on every OpenAI request; AD swap test;
    keyring tests. Re-wrap tests on PG 18:
    - legacy → k0;
    - rotation k0 → k1;
    - corrupted row untouched;
    - racing PUT;
    - **stale pair**: a legacy-only rewrite as `v1.6.0` does it. Chat uses the legacy key, and the next pass repairs
      the row;
    - NULL legacy + current AD → AD;
    - Go and PG `sha256` agree;
    - **single runner** under `pg_try_advisory_xact_lock`, and no advisory lock is left in `pg_locks` after the pass.

    Then the provider registry / `ValidProvider` / CHECK agreement test; `internal/coach/contract_test.go` grep test (no `is_default` outside migrations/schema/`gen/models.go`, no
    `SELECT *` on `api_key_config`); `internal/gateway/coach_key_transit_test.go` (sentinel forwarded, never logged at
    DEBUG, never echoed, 502 path clean); web tests. Run `sqlc diff`, `go vet`, gofmt, `go test -race ./...`, the e2e
    suite (`go test -tags e2e -race ./internal/e2e/...`), and `npm --prefix web run` `typecheck`, `lint`, `test`, `build`.
12. **[X] Compose rehearsal** — `docker compose up` on this branch with seeded keys (at least one legacy row, one
    corrupted row):
    1. Confirm the re-wrap log lines and 100 % AD pairs.
    2. Run the **`v1.6.0` coach image** against the same DB and confirm chat still works (R-b floor 1.6.0).
    3. **Still on `v1.6.0`, replace a key**, then roll forward to this branch. Chat must use the **new** key (the fake
       provider sees it), no key is disabled, and the next re-wrap pass repairs the row.

    Paste all three outputs into the PR.
13. **[X] Docs** — dated amendment notes on ADR-0007 (gateway transits the PUT in memory; AD + keyring per ADR-0031 §7)
    and ADR-0020 (per-(account, provider) keys since `00003`, per-feature defaults now); `docs/architecture/services.md`
    and `data-model.md` coach entries.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** coach touches only schema `coach`;
  the gateway owns no schema and only proxies. Coach emits and consumes no events here (no outbox/inbox change, no
  NATS subject, so no ACL PR and no consumers-before-producers step).
- **goose + sqlc:** additive migration only (ADR-0034 §3); commit `sqlc generate` output; **`sqlc diff` clean**. Never
  run `Down` in production.
- **Rollback floor stays 1.6.0:** the legacy sealed pair is dual-written and still readable by `v1.6.0`; do not re-seal
  it in place, do not drop it (a later two-step contract, recorded in status.md).
- **File ownership:** stay inside the plan's list; the gateway gets one new file, one route row, a test file and the
  OpenAPI path — nothing else in `internal/gateway/`.
- **Secrets:** no key material, plaintext or AD-bound ciphertext in logs, errors or test output; ids only. Nothing is
  pasted into infra; production keeps `COACH_MASTER_KEY` as `k0` (no SOPS change this sprint).
- **Frontend:** `theme.css` tokens/components verbatim (no Tailwind), dark theme, match AB01; the public profile stays the
  only unauthenticated `/api` route.
- **GitOps:** no `kubectl apply`; nothing deploys until m1-07's tag. **D34:** the re-wrap "report" is a log line read on
  demand — no alert, no push channel, no opscheck.
- **Memory-sum rule ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5):** no new pod or container;
  the re-wrap goroutine must stay within coach's existing 128 Mi limit (batches of 50).
- **Parallel sessions:** before claiming an ADR number (if you record one) re-check peers' PRs, tags and worktrees;
  rebase last and take the next free goose version.

## Deliverables

- Coach migration (AD pair, `kek_id`, usage columns); rewritten key queries without `is_default`; `key_default` queries.
- `internal/coach/catalog.go`, coach `GET /models`; gateway `coach_models.go` + route row + OpenAPI path.
- `internal/platform/secrets` keyring + AD seal/open; `cmd/coach` key loading; `internal/coach/rewrap.go`.
- OpenAI `store:false` + usage; finer typed errors with `ErrModelAccess`; usage/cost stored; `usage_month`.
- Settings key panel, catalog-driven switcher, onboarding provider list, coach error copy.
- Tests listed above, incl. the grep test and the gateway transit test; compose rehearsal output in the PR.
- ADR-0007 / ADR-0020 dated notes; architecture docs.

## Update status

- Set each task in [`../sprints/sprint-m1-10.md`](../sprints/sprint-m1-10.md) 🔄 → ✅ as it lands; _Overall_ ✅ when all six are.
- In [`../status.md`](../status.md): the **Sprint board** row for m1-10; the **Milestones** row M1 stays 🔄 (M1b ships
  with m1-07); if the AB01 artboard row isn't already "frozen (PR #, date)", set it.
- **Decisions log:** the legacy sealed pair is dual-written. Its two-step contract is owed to
  [l-01](../sprints/sprint-l-01.md) (`v1.11.0`: stop reading/writing, precondition `pending=0 skipped=0`) and
  [l-02](../sprints/sprint-l-02.md) (`v1.12.0`: drop, `floor=v1.11.0`), not M1c. The AEAD binding is enforced only
  after that; the residual risk is accepted until then. List the AB01 deltas;
  the OpenAI default (`gpt-6-sol` or kept `gpt-5.6-sol`) and the catalog `as_of`; `interview_brain` / `voice_shell`
  capability entries provisional until S6. No flag-inventory change (no new flag).
- Record an ADR only for a call beyond ADR-0031 §7 (e.g. if you depart from the dual-pair design); number it after the
  parallel-sessions check.

## Done when (acceptance)

- [ ] After one re-wrap pass every key has a current AD-bound pair under the active KEK. A corrupted row is skipped,
      reported and untouched. Swapped pairs fail to decrypt: AD pairs are written and verified, and the binding is
      enforced once the legacy pair is contracted (l-01/l-02).
- [ ] The `v1.6.0` coach image still decrypts every key and serves chat against this schema. A key replaced on
      `v1.6.0` is the one chat uses after rolling forward, and nothing is disabled.
- [ ] `store:false` asserted on every OpenAI call; provider fixture tests green per error class; only auth disables a key.
- [ ] `GET /api/coach/models` serves the dated catalog; custom ids "cost unknown"; malformed ids 422; `base_url` 400.
- [ ] No reader or writer of `is_default` left (grep test in CI); no `SELECT *` on `coach.api_key_config`. The interview
      default takes `interview_brain` catalog models or a valid custom id.
- [ ] Usage + `est_cost_micros` stored; F11–F15 match AB01 (copy verbatim; deltas listed); chat unchanged.
- [ ] Gateway transit test green (the key PUT is forwarded, never logged or echoed).
- [ ] CI green incl. `sqlc diff`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1-10-coach-keys`, then conventional commit(s) (`feat(coach): …`) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. The AB01 deltas are listed in the PR and the decisions log; a board update is a later design PR, not a wait.
3. **Release action — merge only:** nothing deploys (`main` is build-only), so there is no live verification in this session. It ships in **`v1.7.0`**, which [m1-07](../sprints/sprint-m1-07.md) cuts. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
