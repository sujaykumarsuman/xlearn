# ADR-0019 — Account settings, onboarding completion & reminder localization/gating

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0005](0005-data-ownership-and-migrations.md) (schema-per-service, soft refs),
  [0006](0006-authn-authz.md) (gateway-minted JWT; cluster-internal isolation),
  [0007](0007-ai-coach-byo-key-and-secrets.md) (coach BYO key — the shell this preps),
  [0008](0008-frontend-stack.md) (SPA embedded in gateway),
  [0016](0016-mistake-journal-and-worker-service-auth.md) (the worker service-auth path this reuses)

## Context

S10 builds the D6 account surface: **Settings** (profile / study budget / timezone /
reminders), the completion of the **3-step onboarding** started in S02 (path → budget →
key), the **reminder** wiring so the S07 notifications worker respects the learner's
budget + timezone, and the Settings **API-keys section shell** the coach sprint (S11)
turns into a working key manager. Four calls needed to be durable.

## Decision

### 1 · `PATCH /me` is a partial update on the caller's own account; email is read-only

`identity` gains `PATCH /accounts/{id}` (behind the gateway's `PATCH /me`), JWT-scoped to
the token subject — the handler 403s if `sub != id`, exactly like `GET /accounts/{id}`.
The update is **partial**: `display_name`, `timezone`, `study_budget_json`,
`reminders_json` are each optional, applied with a single `UPDATE … SET col =
COALESCE(narg, col)` so an omitted field leaves its column untouched (no read-modify-write
race). The handler **validates + canonicalises** before persisting: `display_name` trimmed
non-empty (≤120), `timezone` a resolvable IANA zone (rejects `""`/`Local`),
`study_budget` = `{ weekday_minutes 30..240, weekend_band ∈ {"2","3-4","5"} }`,
`reminders` = `{ daily_reminder_on, daily_reminder_time HH:MM 24h, revision_due_alerts_on }`.
`email` is owned by the OAuth identity — it is **accepted-but-ignored** in the body (never
writable). Times are stored UTC; reminder times are local wall-clock strings.

`study_budget` + `reminders` now ride on **`GET /me`** (they were internal-only in S07) so
the Settings form can round-trip its current values — they are the **owner's own prefs**,
returned only on the JWT-gated `/me`, never anyone else's.

### 2 · Onboarding completion is keyed on `completed_at`, and routing gates on it

`POST /onboarding/step` gains two steps beyond S02's `path`:

- **`budget`** — writes `study_budget_json` to the account **and** sets
  `onboarding.budget_set` in **one transaction**.
- **`finish`** — stamps `onboarding.completed_at = COALESCE(completed_at, now())`
  (idempotent; a re-submit never moves the timestamp). It **does not** set `key_added` —
  that flips true only once the coach key store works (S11). **We never fake key storage.**

The SPA reads onboarding state from `GET /me` and routes an account with **onboarding not
complete** (`completed_at IS NULL`) into the flow at the **first unfinished step**
(`path_chosen` → `budget_set` → else key), and away from it once complete. The `RequireAuth`
gate therefore keys on `onboarding.completed` (not `path_chosen`, the S02 stopgap), so a
learner who chose a path but never finished budget/key resumes the flow instead of landing
in a half-configured app. The initial step is computed **once** from the first `/me` so a
background refetch (the invalidate after saving a step) can't reset the wizard.

### 3 · Reminders are localized + gated per event; `review` sources prefs from identity's existing internal endpoint

The S07 notifications worker (a durable consumer on `xlearn.review.revision_due`) resolves
each account's **timezone + study_budget + reminders** via identity's existing
**`GET /internal/accounts/{id}`** (ClusterIP, no user JWT — ADR-0016), which already
returned all three; the review `IdentityClient.ResolveAccount` signature widened to carry
`reminders` too (nil-safe → default prefs). **No new endpoint, event, or infra** — the
per-request read was already in place from S07 and is not yet hot, so the additive
`xlearn.identity.account_updated` event the prompt reserves is **not** introduced this
sprint (recorded here rather than added silently).

Each `revision_due` event maps to at most one in-app reminder, decided in the account's
**local** time from `now`:

- **Within today's budget window** `[daily_reminder_time, daily_reminder_time + today's
  budget]` → the **daily reminder** channel, gated by `daily_reminder_on`: scheduled at the
  daily time (or immediately if that time has already passed but the window is still open).
- **Past the window** (reviews piled up beyond the day's budget) → the **revision-due
  alert** channel, gated by `revision_due_alerts_on`: surfaced immediately.
- Either channel off → **suppressed** (no reminder row written; the event is still acked so
  it isn't redelivered).

Today's budget minutes come from `weekday_minutes` on weekdays and the weekend band
(`"2"`→120, `"3-4"`→210, `"5"`→300 min) on Sat/Sun in the account's zone. An empty/malformed
`reminders_json` **defaults to ON** (daily at 20:00, alerts on) via pointer-bool decoding —
matching the Settings artboard's initial state, so "disabled prefs" is an explicit opt-out.
The consumer stays idempotent (dedupe on `event_id` in the inbox, inside the write tx) and
uses `DeliverNew` (unchanged from S07).

### 4 · The Settings API-keys section is a masked-read shell; the gateway degrades to an empty state until coach exists

The gateway gains `GET /coach/key` (external) → coach's internal `GET /keys` with a
**coach-audience** minted JWT, returning **only** the masked key(s) + provider + model +
enabled (`{ keys[], connected }`), never a raw key. Because coach lands in S11,
`COACH_BASE_URL` **defaults to empty** (not localhost): a nil coach client makes the handler
return the **empty state** `{keys:[],connected:false}` so the Settings section renders
"No key — coach off" (`ds-badge--warn`) instead of dialling a non-existent service. When a
coach *is* configured, a 404 (no key) also normalizes to the empty state, and a transport
error degrades to it rather than failing the whole screen. The **Add / store / delete /
enable** controls render to match the artboard but are **shell-only** — `PUT`/`DELETE
/coach/key` become functional in S11. **No fake key is ever persisted.**

## Consequences

- **No infra change this sprint.** S10 ships new **identity + gateway + review** images that
  Flux auto-bumps from `main`; there is no new HelmRelease, role, schema, or secret. `review`
  already had `IDENTITY_BASE_URL` (S07); the gateway's `COACH_BASE_URL`/`JWT_AUD_COACH` have
  safe defaults (empty / `"coach"`) and are set when S11 deploys coach.
- **No new migration.** The `account` columns (`display_name`, `timezone`,
  `study_budget_json`, `reminders_json`) and the `onboarding` flags already exist from S02;
  S10 only adds `sqlc` queries + handlers over them.
- **S11 has a ready surface.** The coach vertical (config, client, masked read, coach-audience
  JWT, empty-state contract) exists; S11 flips on `COACH_BASE_URL` and makes store/delete live.
- The reminder gating is a **pure function** of `(now, tz, budget, prefs)` — unit-tested at the
  window boundaries and across timezones — so it stays deterministic and replay-safe.
