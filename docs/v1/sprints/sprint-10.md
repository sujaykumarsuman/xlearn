# Sprint 10 — Settings & onboarding completion

> **Milestone:** none (mid-phase)   ·   **Design phase:** D6 Account + coach
> **Prereqs:** [S02](sprint-02.md), [S07](sprint-07.md)   ·   **Unblocks:** [S11](sprint-11.md)
> **Execute with:** [`../prompts/prompt-s10.md`](../prompts/prompt-s10.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Status |
|---|------|--------|
| 1 | profile / budget / timezone / reminders | ⬜ |
| 2 | onboarding completion (steps 2-3) | ⬜ |
| 3 | reminders wired to notifications | ⬜ |
| 4 | API-keys section shell | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row — this
> sprint carries no milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Build the account surface. After M5 the core learning method is complete; D6 gives the learner a place
to own their account. This sprint finishes **Settings** (profile, study budget, timezone, reminders) and
completes the **3-step onboarding** started in S02 so new users are routed through path -> budget -> key
on first login. It also stands up the Settings **API-keys section shell** that the coach sprint (S11)
turns into a working key manager — so S11 has a ready surface to wire against.

## Scope

**In**
- Extend the `identity` service: `PATCH /me` updates profile (`account.display_name`), study budget
  (`account.study_budget_json`), timezone (`account.timezone`) and reminder prefs (`account.reminders_json`).
- Onboarding steps 2 and 3 on `POST /onboarding/step`: persist `onboarding.budget_set`,
  `onboarding.key_added`, `onboarding.completed_at`; gateway/SPA routes an incomplete account through the flow.
- Feed study-budget windows + reminder prefs + timezone into the `review` notifications worker (from S07)
  so in-app reminders respect the user's budget and timezone.
- Settings screen wired to the artboard: Profile, Study budget, Reminders sections functional; API-keys
  section shell reads `GET /coach/key` (masked) and renders the empty state.

**Out (later sprints)**
- Coach key encryption/storage backend, provider fan-out, and the coach panel — [S11](sprint-11.md).
  The API-keys **store/delete** actions become fully functional there; here they stay shell-only.

## Tasks

### 1 · profile / budget / timezone / reminders

Extend `identity` (schema `identity`, [data-model](../../architecture/data-model.md#schema-identity)):
the `account` row already carries `display_name`, `timezone`, `study_budget_json`, `reminders_json`, so
no new table is needed — add a goose migration only if a column is actually missing
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). Add `sqlc` queries and implement the
gateway-fronted `PATCH /me` ([api.md Auth & account](../../architecture/api.md#auth--account)):

- **Profile:** editable `display_name`; `email` is read-only (owned by the OAuth identity). Avatar upload
  is a stub in v1 (button present per artboard; no storage).
- **Study budget** -> `study_budget_json`: `{ weekday_minutes, weekend_band }` where `weekend_band` is one
  of `2` / `3-4` / `5` (matches the stepper + segmented control in `Settings.dc.html`).
- **Timezone** -> `account.timezone` (IANA name, e.g. `Asia/Kolkata`).
- **Reminders** -> `reminders_json`: `{ daily_reminder_on, daily_reminder_time (HH:MM local), revision_due_alerts_on }`.

Requests are authenticated by the gateway-minted JWT ([ADR-0006](../../adr/0006-authn-authz.md)); the
handler updates only the caller's own account. Reuse `design-system/theme.css` component classes
(`ds-field`, `ds-input`, `ds-seg`, `ds-toggle`) verbatim — no Tailwind. Wire the **Profile**,
**Study budget** and **Reminders** panels of `Settings.dc.html`.

### 2 · onboarding completion (steps 2-3)

Complete the 3-step first-run flow whose **step 1 (choose path -> `onboarding.path_chosen`)** shipped in
S02. Drive steps 2 and 3 through `POST /onboarding/step`
([api.md](../../architecture/api.md#auth--account)), persisting to the `onboarding` table:

- **Step 2 — set study budget:** writes `study_budget_json` (same shape as task 1) and sets
  `onboarding.budget_set = true`.
- **Step 3 — add API key (optional/skippable):** renders the provider + key form from the `Auth.dc.html`
  onboarding artboard. Submitting a key targets coach `PUT /coach/key`, which only persists from
  [S11](sprint-11.md); until then the reliable completion path is **Skip for now**. **Finish** (or Skip)
  sets `onboarding.completed_at`; `onboarding.key_added` is set true only once the coach store works (S11).
  **Do not fake key storage.**
- **Routing:** the gateway/SPA reads onboarding state from `GET /me`; an account with
  `completed_at IS NULL` is routed into the flow at the first unfinished step after login, and away from it
  once complete. SPA is embedded in gateway ([ADR-0008](../../adr/0008-frontend-stack.md)).

### 3 · reminders wired to notifications

Make the S07 **notifications worker** (inside `review`, see
[services.md](../../architecture/services.md#notifications-worker-inside-review-in-v1)) respect the user's
budget and timezone. The worker turns `xlearn.review.revision_due` + study-budget windows into
`review.reminder` rows ([events.md Flow 4](../../architecture/events.md#flow-4--periodic-due-sweep--reminders)).
Because `review` cannot cross-schema read `identity`
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):

- The worker resolves each account's `timezone` + `study_budget_json` + `reminders_json` via identity's
  internal `GET /accounts/{id}` (JWT-authenticated, ClusterIP), caching per account.
- Compute the daily reminder at `daily_reminder_time` **in the account's timezone**; suppress reminders
  when `daily_reminder_on = false`; only raise revision-due alerts when reviews exceed the budget window
  and `revision_due_alerts_on = true`.
- If per-request identity reads become hot, capture the alternative (an additive
  `xlearn.identity.account_updated` event consumed by `review`) as an ADR
  ([ADR-0004](../../adr/0004-inter-service-comms-and-events.md)) rather than inventing it silently. Any
  consumer stays idempotent (dedupe on `event_id`).

### 4 · API-keys section shell

Build the Settings **API keys · your coach** panel from `Settings.dc.html` (provider, masked key, default
model, enabled toggle, "Test connection", "Add a provider"). Coach backend lands in [S11](sprint-11.md)
([ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md)), so in S10:

- Wire the **read**: `GET /coach/key` ([api.md Coach](../../architecture/api.md#coach)) returns the
  masked key + provider + model + enabled, never the raw key. When no key exists or the coach service is
  not yet deployed, render the **empty state** ("No key — coach off", `ds-badge--warn`).
- The **Add a provider** form and the **store/delete/enable** controls render (matching the artboard) but
  are shell-only: `PUT /coach/key` and `DELETE /coach/key` become functional in S11. Keep the UI ready;
  **do not persist a fake key**.
- Reuse `theme.css` (`ds-badge`, `ds-chip`, `ds-seg`, `ds-toggle`, `ds-input--mono`) verbatim.

## Acceptance criteria
- [ ] A user edits profile, study budget, timezone, and reminder preferences; changes persist via `PATCH /me`.
- [ ] First-run onboarding runs all 3 steps (path -> budget -> key, key skippable) and marks completion
      (`onboarding.completed_at` set); a completed account is no longer routed through the flow.
- [ ] Reminders respect the study budget + timezone (daily reminder fires at the local time; disabled prefs suppress).
- [ ] Settings screen matches the artboard, including the API-keys section shell (functional store/delete deferred to S11).

## Definition of Done
CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs
- **Timezone correctness** for reminders and weekly weak-area boundaries: store IANA names, compute
  reminder times in the account's timezone (not server-local), and keep all persisted times UTC.
- The API-keys section depends on coach (S11) for store/delete — keep the UI ready but **do not fake key
  storage**; exercise only the masked `GET /coach/key` read + empty state in S10.
