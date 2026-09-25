# Prompt — Sprint m4-05 · AI allowance + consents (`account_consent` AI kinds)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-05.md`](../sprints/sprint-m4-05.md)   ·   **Milestone:** M4 (platform AI)   ·   **Prereqs:** [l-03](../sprints/sprint-l-03.md) (`account_consent`) · [m4-04](../sprints/sprint-m4-04.md) (provisional grades) · built on [m4-02](../sprints/sprint-m4-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m4-05.md`](../sprints/sprint-m4-05.md) — the history rules, the refresh call, the allowance
  payload and the test tables live there.
- Decisions: [ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (two unticked
  consents + behavioral opt-in; switching off falls back to manual), [§5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25)
  (learner view is a percentage, never dollars), [§7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes)
  (UI names); [ADR-0033 §4](../../adr/0033-invite-only-admission-and-owner-admin.md#4-schema-identity-the-adr-0005-expand-rules)
  (M4 row), [§6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a)
  (l-05 later shows these consents at acceptance), [§12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2)
  row 13 (judge reads consents, 5-minute cache); [ADR-0035 §4 L17](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory);
  PRD [§5.6](../../prd/xlearn-v2-prd.md) R-AI4–R-AI6; [ADR-0019](../../adr/0019-account-settings-onboarding-and-reminder-gating.md) (onboarding).
- Research: [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (cap layers, learner view,
  owner limit override), [§8](../research/t5-platform-ai.md#8-privacy-and-residency) (consent: free, specific, withdrawable,
  effective immediately), [§9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (onboarding copy, names).
- m4-02's plan, tasks 4, 7 and 8 ([`../sprints/sprint-m4-02.md`](../sprints/sprint-m4-02.md)): `AllowanceFor`, the account
  cache + `POST /internal/accounts/{id}/refresh`, the consent gate in `Reserve`.
- The frozen board `design-system/screens/v2/AB18-ai-allowance-consents.html` (states and copy the payloads must support).
- Code: `internal/identity/{service.go,handlers.go,consent.go,store/store.go,store/queries/*.sql}`, `internal/platform/consent/`,
  `internal/judge/ai/` (`Reserve`, ledger sums, `AllowanceFor`, the account client), `internal/judge/handlers.go`,
  `internal/gateway/{bff.go,gateway.go,openapi_drift_test.go}`, `docs/architecture/openapi.yaml`,
  `web/src/screens/{Auth.tsx,Settings.tsx}`, `web/src/components/Coach.tsx`, `web/src/lib/`, `internal/e2e/`.

## Context

l-03 created identity's `account_consent` table (inert: nothing writes AI kinds yet). m4-02 gave judge its ledger, caps,
breaker, the consent kinds (`internal/platform/consent`), a gate inside `Reserve` that requires the right consent per
purpose, `AllowanceFor`, and a 5-minute cache of `/internal/accounts/{id}` with a refresh route; m4-03's analyzer and
m4-04's provisional flow fall back to manual entry when that gate refuses. This sprint adds the **write path for the AI
consents** (unticked by default, append-only history, withdrawing graded withdraws passing), **refreshes judge's cache on
every change** so a withdrawal bites on the next call, proves "no call without consent" end to end, and serves the
allowance **as percentages** at `GET /api/me/ai-allowance`. It also names the two tiers "xLearn AI (included)" / "Your AI
coach (your key)" and rewrites the onboarding coach-step copy. The meter and toggles UI is m4-06; the acceptance-step
consents are l-05. Ships in **v1.16.0** (m4-07 tags).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `account_consent` exists in identity (l-03 on `main`) with no CHECK on `kind` (the Go registry
      `internal/identity/consent.go`) and the partial unique index `(account_id, kind) WHERE withdrawn_at IS NULL`.
- [ ] m4-04 merged on `main` (`gh pr list --state merged --search "m4-04"`).
- [ ] m4-02's pieces on `main`: `internal/platform/consent`; `/internal/accounts/{id}` returning live `consents`; judge's
      account cache with `POST /internal/accounts/{id}/refresh`; the consent gate in `Reserve`; `AllowanceFor`.
- [ ] AB18 frozen: ds-m4-01 merged (the merge is the freeze), so `design-system/screens/v2/AB18-ai-allowance-consents.html` is on `main`.
- [ ] Parallel sessions: `gh pr list`, `git worktree list` and ListAgents show no peer editing identity's store, the
      gateway route table or `internal/judge/ai` right now.

## Do this (in order)

1. **[X] Branch** `feat/m4-05-ai-consents-allowance` from an up-to-date `main`.
2. **[X] identity** (plan task 1): register the three AI kinds in `internal/identity/consent.go` (no DDL); add
   `consent.Version` if missing; `SetConsents` (one tx, append-only, unchanged kinds write nothing, graded-withdraw cascades
   to passing, **enforces graded ⇒ passing itself with a typed `ErrRequiresGraded`** so l-05 inherits it, transport-free
   for l-05); `GET/PATCH /accounts/{id}/consents` (self only, three kinds writable, `not_writable`, the error mapped to 422
   `requires_ai_review_graded`, `changed_at`); the erase cascade test. `sqlc generate` for new queries.
3. **[X] gateway consents** (task 2): `internal/gateway/ai.go` with `GET/PATCH /api/me/consents`; JSON + `Sec-Fetch-Site`
   checks; after every successful PATCH, judge `POST /internal/accounts/{id}/refresh` (one retry, then log at ERROR and
   still return the PATCH result); `openapi.yaml`.
4. **[X] Consent end to end** (task 3): the `-tags e2e` table test (states set through `PATCH /api/me/consents`, the legacy
   passing-only row seeded by SQL because `SetConsents` refuses it, × purposes,
   exact `llm_call` counts, judge on an `httptest` fake provider) and the immediacy test; fix any purpose m4-02's gate
   misses and log it.
5. **[X] Allowance** (task 4): judge `GET /me/ai-allowance` over `AllowanceFor`, extended additively (`reason`,
   `day_used_pct`, `day_resets_at`, `analyses_left_today`, `finals_left_today`, the `off` state) from the same sum function as
   `Reserve`; percentages only (a test forbids currency fields); gateway `GET /api/me/ai-allowance` with presence by config.
   When absent it answers **exactly `apiNotFound`'s 404** (m3-09's rule, no distinct error code; a byte-for-byte test
   against an unknown route). Then the golden test over seeded ledger rows.
6. **[X] Web** (task 5): `web/src/lib/aiNames.ts`; swap literal names for the constants; the `StepCoach` copy with the
   conditional "xLearn AI already reviews your submissions — no key needed." line; `theme.css` verbatim; tests.
7. **[X] Docs** (task 6): `docs/architecture/{api,data-model,services}.md`, `openapi.yaml`.
8. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG integration for identity and judge via
   `XLEARN_TEST_DATABASE_URL`), `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, the migration lint, the openapi
   drift test, `npm test` + typecheck in `web/`. For a manual look, the compose stack with a scratchpad `-f` override that
   points judge at a fake provider (m4-01's convention; never committed): dev login → `PATCH /api/me/consents` →
   `GET /api/me/ai-allowance` → withdraw → the next analysis is refused.
9. **[X] Ship:** see **Ship** below.

## Constraints

- **Service boundaries (ADR-0005):** identity alone owns `account_consent`; judge reads consents only through
  `/internal/accounts/{id}` (the MI-5a fence, ADR-0016's worker-auth model, no service tokens) and owns `ai_disabled`; the
  gateway never stores consent. No HTTP inside a transaction.
- **goose + sqlc:** no migration is expected (l-03's table and index suffice); if one turns out to be needed, take the next
  free version, expand only. Generated code committed, `sqlc diff` clean; never run `Down` in production.
- **Consent is unticked and fail-closed:** no backfilled grants (owner included).
- **Money:** learners see percentages only; the allowance and `Reserve` share one sum function.
- **Frontend:** `theme.css` tokens and components verbatim (no Tailwind), dark theme; copy strings from `aiNames.ts`.
- **GitOps / infra:** none expected (gateway → judge `/internal/*` is admitted by MI-5a; no new caller); if one appears,
  it's its own `../infra` PR merged before v1.16.0 — never `kubectl apply`.
- **D34:** no alerting of any kind; a failed refresh is an ERROR log line.
- **Memory-sum rule:** no new pod.
- **Parallel sessions:** check peers' PRs, tags and worktrees before merging; rebase if a peer touched the same files.

## Deliverables

- identity: the kinds in the consent registry; `SetConsents` (with `ErrRequiresGraded`) + queries; `GET/PATCH /accounts/{id}/consents`; tests.
- gateway: `internal/gateway/ai.go` (`/api/me/consents` with the judge refresh, `/api/me/ai-allowance`); `openapi.yaml`;
  tests.
- judge: `GET /me/ai-allowance` over the extended `AllowanceFor`; the allowance golden test; gate fixes if any.
- e2e: the consent table test and the immediacy test.
- web: `web/src/lib/aiNames.ts`; the `StepCoach` copy; name swaps; tests.
- Docs (`api.md`, `data-model.md`, `services.md`).

## Update status

- [`../sprints/sprint-m4-05.md`](../sprints/sprint-m4-05.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M4** milestone row stays 🔄 (consents + allowance merged, ship
  in v1.16.0); the flag inventory unchanged; a **Decisions log** line each for: the allowance in percentages (the ADR over
  the register's "$" wording), the judge cache refresh on every consent change (bounded by the 5-minute TTL on failure),
  withdrawing graded withdraws passing, graded ⇒ passing enforced inside `SetConsents` (every writer, l-05 included), the
  allowance's absent state = the unknown-route 404, the `off` state, and the conditional onboarding line. Add to m4-07's day-1 notes:
  "owner and testers tick the consents in Settings before platform AI runs on their accounts" (a post-ship owner event; no session
  waits on it).

## Done when (acceptance)

- [ ] **Consent defaults are unticked**: no rows ⇒ every AI kind reads as not granted, on the API and on judge's side.
- [ ] `PATCH /api/me/consents` grants and withdraws with append-only history; graded-withdraw cascades to passing; only the three AI kinds; self only; `SetConsents` itself refuses passing without graded.
- [ ] **No platform-AI call for an account without the required consent** (table test through the real write path: zero `llm_call` rows); a withdrawal applies to the next call at once.
- [ ] **Allowance matches the ledger** (golden) and `paused` agrees with `Reserve`; percentages only.
- [ ] Names from shared constants; the onboarding coach step carries the t5 §9 copy and never overclaims.
- [ ] openapi drift and all identity/gateway/judge/web tests green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m4-05-ai-consents-allowance`.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships in `v1.16.0`):** Nothing deploys; it ships in `v1.16.0` (cut by [m4-07](../sprints/sprint-m4-07.md)). Don't tag. No infra PR.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
