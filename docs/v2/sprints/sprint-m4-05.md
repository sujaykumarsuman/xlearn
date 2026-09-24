# Sprint m4-05 — AI allowance + consents (`account_consent` AI kinds)

> **Milestone:** M4 — platform AI (owner cohort)   ·   **Track:** product (order 64)
> **Prereqs:** [l-03](sprint-l-03.md) (`account_consent` table) · [m4-04](sprint-m4-04.md) (identity/judge/gateway edits serialized) · built on [m4-02](sprint-m4-02.md) (consent kinds, the gate in `Reserve`, `AllowanceFor`, the account cache + refresh)   ·   **Unblocks:** [m4-06](sprint-m4-06.md) (AB18 allowance meter + Settings consents) · [l-05](sprint-l-05.md) (acceptance-step consents reuse the write path)
> **Release action:** merge only (ships in **v1.16.0**, tagged by [m4-07](sprint-m4-07.md)). No infra PR: every caller already exists (gateway → identity; gateway → judge, including judge's `/internal/*` under MI-5a; judge → identity :8081 from [mi-12](sprint-mi-12.md)).
> **Calendar:** December 2026
> **Execute with:** [`../prompts/prompt-m4-05.md`](../prompts/prompt-m4-05.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | identity: register the AI kinds (+ behavioral opt-in), append-only `SetConsents`, `GET/PATCH /accounts/{id}/consents` | X | ⬜ |
| 2 | gateway: `GET/PATCH /api/me/consents` + judge account-cache refresh after every change | X | ⬜ |
| 3 | judge honours consents end to end: verify m4-02's per-purpose gate, the "no call without consent" table test through the real write path | X | ⬜ |
| 4 | `GET /api/me/ai-allowance` (m4-02's `AllowanceFor`, extended additively) | X | ⬜ |
| 5 | Naming + onboarding copy ("xLearn AI (included)" / "Your AI coach (your key)") | X | ⬜ |
| 6 | Docs, openapi, tests | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M4 milestone row + decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] `account_consent(id, account_id, kind, version, granted_at, withdrawn_at)` exists in identity ([l-03](sprint-l-03.md)) with
      **no CHECK on `kind`** (the Go registry `internal/identity/consent.go`), the partial unique index
      `(account_id, kind) WHERE withdrawn_at IS NULL`, and `ON DELETE CASCADE` from `account`
- [ ] [m4-04](sprint-m4-04.md) merged (identity, judge and gateway edits are serialized behind it)
- [ ] [m4-02](sprint-m4-02.md) on `main`: `internal/platform/consent` (`AIReviewGraded`, `AIReviewPassing`, `AIBehavioral`);
      identity's `/internal/accounts/{id}` returning `consents` as `{kind: {version, granted_at}}` for live rows; judge's
      5-minute account cache with **`POST /internal/accounts/{id}/refresh`**; the gate inside `Reserve` (consent per purpose);
      `AllowanceFor(account)`; judge-owned `ai_disabled` (`llm_account_limit`)
- [ ] AB18 frozen ([ds-m4-01](sprint-ds-m4-01.md)): `design-system/screens/v2/AB18-ai-allowance-consents.html` — its states
      (F1–F6), the consent copy (F7–F8) and the two names (F9) define the payloads
- [ ] Parallel sessions: no peer PR open on identity's store, `internal/gateway` route tables or `internal/judge/ai`

## Goal

Give every account control over, and a view of, the platform AI it uses. identity stores the **two AI consents**
("xLearn AI reviews my graded work"; "…and also reviews my passing solutions") and the **separate behavioral opt-in** as
unticked-by-default, append-only `account_consent` rows, editable at `PATCH /api/me/consents`; every change **drops
judge's cached copy at once**, so m4-02's gate in `Reserve` applies a withdrawal to the very next call; and
`GET /api/me/ai-allowance` shows the account's allowance **as percentages, never dollars**, from the same ledger sums
admission uses. The UI names the two tiers **"xLearn AI (included)"** and **"Your AI coach (your key)"**, and the
onboarding coach step says a key is optional.

## Scope

**In**
- identity: the three AI kinds in the consent registry, `SetConsents`, JWT self routes
  ([ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24),
  [ADR-0033 §4](../../adr/0033-invite-only-admission-and-owner-admin.md#4-schema-identity-the-adr-0005-expand-rules) M4 row).
- gateway: `GET/PATCH /api/me/consents` (+ the judge refresh call), `GET /api/me/ai-allowance`.
- judge: the learner route `GET /me/ai-allowance` over an additively extended `AllowanceFor`; the end-to-end consent test
  ([t5 §6 learner view](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls),
  [§8 consent](../research/t5-platform-ai.md#8-privacy-and-residency),
  [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 13).
- web: `web/src/lib/aiNames.ts` and the onboarding coach-step copy ([t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2)).

**Out**
- The per-purpose consent gate, the ledger, caps and `AllowanceFor` itself → already [m4-02](sprint-m4-02.md).
- The consents on the acceptance step at invite (L-C), the 18+ attestation and the notice version → [l-05](sprint-l-05.md)
  (v1.17.0), which calls this sprint's `SetConsents`.
- The allowance meter, paused states, Settings AI consent toggles and badges → [m4-06](sprint-m4-06.md) (AB18).
- Raising limits to $100 / $80 → the v3 opening ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- Any BYO use for platform work (never in v2.0).

## Tasks

### 1 · identity consents [X]

- **Kinds:** register m4-02's `consent.AIReviewGraded` (Analyze diagnosis, `Score`, `Feedback`), `consent.AIReviewPassing`
  (D16/D26 pass review; m4-02's gate needs it **together with** graded) and `consent.AIBehavioral` (behavioral C4 answers; no
  v2.0 course uses it, but the toggle exists) in l-03's registry `internal/identity/consent.go`. No CHECK on `kind`, so **no
  DDL**. Add `consent.Version = "ai-consent@1"` to m4-02's package if it isn't there (AB18 F7 shows "consent v1").
- **History** (append-only): a grant inserts `(kind, version = consent.Version, granted_at = now())`; a withdrawal sets
  `withdrawn_at = now()` on the live row; a re-grant inserts a new row. l-03's partial unique index keeps one live row per
  kind (a concurrent double grant hits it and is retried as a no-op). **No row ⇒ not granted** — defaults are unticked (D24,
  PRD R-AI6). **Withdrawing `ai_review_graded` also withdraws `ai_review_passing`** (AB18 F8), so a later re-grant of graded
  doesn't silently revive pass review.
- **Store function** `SetConsents(ctx, accountID, changes)` in `internal/identity/store`: one transaction; unchanged kinds
  write nothing; transport-free, so [l-05](sprint-l-05.md)'s acceptance step calls it too (it sends only the two review kinds).
  **`SetConsents` itself enforces graded ⇒ passing**, so every writer inherits the rule. Inside the transaction, a result
  that would leave `ai_review_passing` live without a live `ai_review_graded` fails with the typed error
  `store.ErrRequiresGraded` and writes nothing. That covers granting passing alone, and withdrawing graded while granting
  passing in one call. The route maps the error to 422, and l-05 gets the same error. Judge's own graded ⇒ passing check
  stays as defence in depth.
- **Routes** (`internal/identity/consents.go`, JWT, the token subject must equal `{id}`):
  - `GET /accounts/{id}/consents` → `{ai_review_graded: {granted, version, changed_at|null}, ai_review_passing: {…},
    ai_behavioral: {…}}` (`changed_at` = the latest grant or withdrawal, for AB18 F7's "Last changed … · consent v1");
  - `PATCH /accounts/{id}/consents {ai_review_graded?: bool, ai_review_passing?: bool, ai_behavioral?: bool}` → the same
    shape. Only these three kinds are writable here (l-05's `privacy_notice` / `age_18` → 422 `not_writable`). Granting
    passing without graded (already live or in the same request) → `SetConsents`' `ErrRequiresGraded`, mapped to 422
    `requires_ai_review_graded`.
- **Internal read unchanged:** m4-02's `/internal/accounts/{id}` already returns live rows as `{kind: {version,
  granted_at}}`; judge's gate applies the graded ⇒ passing rule. Erase: rows go with the account (l-03's cascade; add a
  test).

### 2 · gateway consents + judge cache refresh [X]

- `GET /api/me/consents` and `PATCH /api/me/consents` in `apiRoutes()` (`internal/gateway/ai.go`, new), proxying to identity
  with the minted JWT like `handlePatchMe`. The PATCH goes through M1b's JSON + `Sec-Fetch-Site` checks, `httpx.ReadBody`
  (typed 413) and the L5 per-account bucket. Not cohort-gated: every account may record its choices.
- **After every successful PATCH**, the gateway calls judge `POST /internal/accounts/{id}/refresh` (m4-02; drops the cached
  entry, enqueues nothing) when `JUDGE_BASE_URL` is set — so "switching one off … effective immediately"
  ([t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)) holds for the next call. One retry; if it still fails,
  log at ERROR (D34: no alert) and return the PATCH result anyway — the consent is stored, and the 5-minute TTL bounds the
  staleness. A test with a fake judge asserts the call and the failure path.
- `docs/architecture/openapi.yaml` gains both routes.

### 3 · judge honours consents, end to end [X]

m4-02's gate (inside `Reserve`) already requires, per purpose: `ai_review_graded` for `Score`, `Feedback`, `score_regrade`
and Analyze diagnoses; plus `ai_review_passing` for Analyze pass reviews (keyed on [m4-03](sprint-m4-03.md)'s
`Admission.PassReview`; a `both` job that lost only passing consent downgrades to `diagnose`); plus `ai_behavioral` for
behavioral items. This
sprint proves it **through the real write path**:
- **Table test** (`internal/e2e`, `-tags e2e`, judge on an `httptest` fake provider — m4-01's convention): consent states
  {no rows, graded only, graded + passing, a legacy passing-only row, withdrawn, re-granted} × purposes {diagnose,
  pass_review, score, feedback, score_regrade, behavioral}.
  - Every state is set via `PATCH /api/me/consents`, except the legacy passing-only row: `SetConsents` refuses to write
    it, so that row is seeded by SQL.
  - Assert the exact number of `llm_call` rows (**zero wherever consent is missing**) and the fallback reached (m4-03's
    `skipped(no_consent)`, m4-04's `self_grade_pending` / attestation).
- **Immediacy test:** grant → a pass review runs; withdraw → the next conclusion makes no call, without waiting 5 minutes.
- If the gate misses a purpose (e.g. `score_regrade`), fix it in `internal/judge/ai` and record it in the decisions log.

### 4 · AI allowance [X]

- judge learner route **`GET /me/ai-allowance`** (aud=judge JWT, scoped to `sub`) over m4-02's `AllowanceFor(account)`,
  **extended additively** (the existing `used_pct`, `resets_at`, `state`, `global_paused` keep their meaning) for AB18
  F1–F6:

  ```json
  {"state": "ok|low|paused|off",
   "reason": null,
   "used_pct": 37, "resets_at": "2027-01-01T00:00:00Z",
   "day_used_pct": 12, "day_resets_at": "2026-12-16T00:00:00Z",
   "analyses_left_today": 17, "finals_left_today": 6,
   "global_paused": false}
  ```

  - `reason ∈ {null, account_month, account_day, global, breaker, ai_disabled, platform_disabled, not_cohort, no_consent}`.
  - `used_pct` / `day_used_pct` = `min(100, floor(100 × used / limit))` against the account's effective limits (L17 $6/month,
    $1/day, or the owner's `llm_account_limit` override); the counts against ≤ 20 analyses and ≤ 6 AI finals a day
    ([ADR-0035 §4 L17](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)). All from the **same
    sum function `Reserve` uses** (inflight rows at their estimate; UTC periods).
  - `paused` **iff `Reserve` would refuse a `counted` job for this account now** (month or day cap, global cap, daily guard,
    breaker) — AB18 F3/F4/F5; `low` when either account scope is ≥ 80% (F2); `off` when platform AI isn't in use for this
    account: `ai_disabled` (F6), no `ai_review_graded`, the kill switch false or outside the cohort (m4-06 treats the last
    two as "absent").
  - **Percentages only, never dollars** ([ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25),
    PRD R-AI5). The register's "remaining $" wording is superseded by the ADR (ds-m4-01 recorded the same reading); dollars
    stay owner-only in `judge admin llm-limit` / `ledger`. A test asserts no response field is a currency amount.
- gateway **`GET /api/me/ai-allowance`** (`internal/gateway/ai.go`) is **present by config**, following
  [m3-09](sprint-m3-09.md)'s presence rule. Unless `JUDGE_BASE_URL` is set and the account is in the cohort, it answers
  **exactly `apiNotFound`'s 404**, the same status, body and headers as an unknown `/api` route, so there is nothing to
  feature-detect. m4-06 hides the section on any 404. No BFF caching. A presence test compares the response to an
  unknown-route 404 byte for byte.
- **Golden test** with seeded ledger rows: ok; low at 80%; month exhausted; day exhausted; global cap; breaker open;
  `ai_disabled`; owner override limit; an `inflight` row counted at its estimate; a `usage_unknown` row — each asserting the
  payload and that `Reserve` agrees with `state`.

### 5 · Naming + onboarding copy [X]

- `web/src/lib/aiNames.ts`: `XLEARN_AI_NAME = "xLearn AI (included)"`, `BYO_COACH_NAME = "Your AI coach (your key)"`;
  replace literal names where they already appear (coach panel header, Settings coach section — m1-10's "your key" naming)
  with the constants. m4-06 uses them for AB18 F9.
- **Onboarding coach step** (`StepCoach` in `web/src/screens/Auth.tsx` — step 4 of 4 today; t5 §9 calls it "3/4" from before
  the username step): heading **"Optional: bring your own AI coach"**; the line **"xLearn AI already reviews your
  submissions — no key needed."** only when `GET /api/me/ai-allowance` answers 200 with `state ≠ off` (otherwise "Your key
  powers the coach chat on every screen." — the copy must never claim a review that won't happen); buttons
  **[Connect & finish] [Skip for now]**. No consent toggle here: consents live in Settings (m4-06) and the acceptance step
  (l-05), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2).
- `theme.css` classes verbatim; tests in `Auth.test.tsx` (both copy variants, the skip path) and `Settings.test.tsx` (names).

### 6 · Docs, openapi, tests [X]

- `docs/architecture/api.md` (the routes, the allowance shape, the error codes), `openapi.yaml` (drift test),
  `data-model.md` (consent kinds, history, the graded ⇒ passing rule), `services.md` (gateway → judge cache refresh).
- identity store integration tests: grant / withdraw / re-grant history, concurrent PATCHes against the partial unique
  index, the graded-withdraw cascade, and `SetConsents` refusing passing without graded (`ErrRequiresGraded`, nothing
  written) when called directly, as l-05 does. Handler tests (self only, closed kinds, `not_writable`, `requires_ai_review_graded`),
  gateway route tests (incl. the refresh call), the judge tests from tasks 3–4, the web tests from task 5.
- No new ADR expected (ADR-0031 §4/§5 cover it); record the calls in the decisions log. A deviation from ADR-0031 would
  need a new ADR.

## Acceptance criteria

- [ ] **Consent defaults are unticked**: an account with no rows reads every AI kind as not granted, on the API and on
      judge's side; no surface pre-ticks.
- [ ] `PATCH /api/me/consents` grants and withdraws with append-only history; withdrawing graded withdraws passing; only
      the three AI kinds are writable; only for the caller's own account. `SetConsents` itself refuses passing without
      graded, so l-05's writer can't store a passing-only row.
- [ ] **No platform-AI call for an account without the required consent** (table test through the real write path: zero
      `llm_call` rows); a withdrawal applies to the next call at once (judge cache refreshed on every change).
- [ ] **Allowance matches the ledger** (golden) and `paused` agrees with `Reserve`; percentages only, no currency field.
- [ ] "xLearn AI (included)" / "Your AI coach (your key)" come from shared constants; the onboarding coach step carries the
      t5 §9 copy and never claims AI review when it's off.
- [ ] openapi drift and identity/gateway/judge/web tests green.

## Release

**Merge only — ships in v1.16.0**, tagged by [m4-07](sprint-m4-07.md). No infra PR and no new caller. On M4 day 1 (m4-07's
flip) **the owner ticks the consents in Settings** (m4-06) before any platform AI runs on his account — consent is
fail-closed for everyone, the owner included; testers do the same. l-05 later asks every account at the acceptance step.

## Definition of Done

CI green (`go build`, `go vet`, `go test -race ./...`, `sqlc diff`, the migration lint, the openapi drift test, web tests
and typecheck) · e2e green · merged to `main` via PR (squash) · statuses updated (this file + [`../status.md`](../status.md))
· local `main` synced.

## Risks / watch-outs

- **Consent defaults must be unticked** — never backfill grants, not even for the owner; the table test and the web tests
  guard it.
- **A failed cache refresh** leaves a withdrawn consent live in judge for ≤ 5 minutes — bounded, logged at ERROR, and the
  write itself succeeded; retry once. Don't make the PATCH fail on it (the learner's choice is recorded either way).
- **Two sources of truth for spend** — the allowance must call admission's own sum function, never re-derive it.
- **Leaking prices** — percentages only for learners; dollars only in `judge admin`.
- **Onboarding copy overclaiming** AI review for an account without consent or outside the cohort — the conditional line.
- **l-05 reuse** — keep `SetConsents` transport-free so the acceptance step has no second writer.
