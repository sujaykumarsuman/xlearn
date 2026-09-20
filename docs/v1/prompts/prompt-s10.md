# Prompt — Sprint 10 · Settings & onboarding completion

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-10.md`](../sprints/sprint-10.md)   ·   **Milestone:** none   ·   **Prereqs:** [S02](../sprints/sprint-02.md), [S07](../sprints/sprint-07.md)

## Read first
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions (dark theme, service boundaries, GitOps, ship at session end (AGENT.md land-and-sync)).
- [`../sprints/sprint-10.md`](../sprints/sprint-10.md) — this sprint's plan, scope, acceptance, and the Status table you will keep current.
- [`../../architecture/services.md`](../../architecture/services.md) — `identity` (account/onboarding) and the `notifications` worker inside `review`.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `identity`: `account` (`display_name`, `timezone`, `study_budget_json`, `reminders_json`) + `onboarding` (`path_chosen`, `budget_set`, `key_added`, `completed_at`).
- [`../../architecture/api.md`](../../architecture/api.md) — `PATCH /me`, `POST /onboarding/step`, `GET /me`, and the Coach `GET /coach/key`.
- [`../../architecture/events.md`](../../architecture/events.md) — Flow 4 (due sweep -> `review.revision_due` -> reminders); idempotent consumers.
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — section 6.7 (Account, onboarding, settings): R-AN1/2/3.
- `design-system/screens/Settings.dc.html` and `design-system/screens/Auth.dc.html` — layout/copy/interaction intent (design references; **not runnable, do not ship**). Reuse [`design-system/theme.css`](../../../design-system/theme.css) verbatim.
- ADRs: [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (data ownership + goose/sqlc), [ADR-0006](../../adr/0006-authn-authz.md) (JWT auth), [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (coach BYO key — context for the shell), [ADR-0008](../../adr/0008-frontend-stack.md) (SPA embedded in gateway), [ADR-0009](../../adr/0009-deployment-and-gitops.md) (Flux/Helm, build-semver).
- Sibling infra repo: `infra/apps/xlearn-identity.yaml` and `infra/apps/xlearn-gateway.yaml` (existing HelmReleases + image-automation entries) — S10 ships new images of existing services; no new HelmRelease.

## Context
The core method (M5) is complete: practice, review (scheduler + mistakes + notifications), and assessment
(mock + progress) all run. S02 shipped `identity` with OAuth, sessions, JWT/JWKS, and **onboarding step 1**
(choose path). This sprint adds the D6 account surface: it finishes **Settings** (profile / budget /
timezone / reminders) and **onboarding steps 2-3**, and wires reminders to the study budget. It also builds
the Settings **API-keys section shell** so the next sprint (S11, coach) has a ready surface to make live.

## Do this (in order)
1. **profile / budget / timezone / reminders** — In `internal/identity/` + `cmd/identity/`, add `sqlc`
   queries over the existing `account` columns (`display_name`, `timezone`, `study_budget_json`,
   `reminders_json`); add a goose migration only if a column is missing. Implement `PATCH /me` behind the
   gateway (JWT-scoped to the caller's own account): update `display_name` (email read-only),
   `study_budget_json = { weekday_minutes, weekend_band in {2,"3-4",5} }`, `timezone` (IANA), and
   `reminders_json = { daily_reminder_on, daily_reminder_time (HH:MM local), revision_due_alerts_on }`.
   Wire the Profile, Study budget, and Reminders panels of `Settings.dc.html` in `web/` using `theme.css`
   classes (`ds-field`, `ds-input`, `ds-seg`, `ds-toggle`). Avatar upload is a stub (button only).
2. **onboarding completion (steps 2-3)** — Drive steps 2 and 3 through `POST /onboarding/step`. Step 2
   writes `study_budget_json` + sets `onboarding.budget_set`. Step 3 renders the provider/key form from
   `Auth.dc.html`; its key submit targets coach `PUT /coach/key` (functional only in S11), so the reliable
   completion path is **Skip for now**. **Finish**/Skip sets `onboarding.completed_at`; set `key_added`
   true only when the coach store works (S11) — do not fake it. In the gateway/SPA, read onboarding state
   from `GET /me` and route an account with `completed_at IS NULL` into the flow at the first unfinished
   step after login, away from it once complete.
3. **reminders wired to notifications** — In `internal/review/` (the S07 notifications worker), resolve
   each account's `timezone` + `study_budget_json` + `reminders_json` via identity's internal
   `GET /accounts/{id}` (JWT, ClusterIP; cache per account) — never cross-schema read `identity`. Compute
   the daily reminder at `daily_reminder_time` in the account's timezone; suppress when `daily_reminder_on`
   is false; only raise revision-due alerts past the budget window when `revision_due_alerts_on` is true.
   Keep the consumer idempotent (dedupe on `event_id`). If per-request identity reads get hot, record an
   ADR proposing an additive `xlearn.identity.account_updated` event rather than adding it silently.
4. **API-keys section shell** — Build the "API keys · your coach" panel from `Settings.dc.html`. Wire the
   read `GET /coach/key` (masked key + provider + model + enabled, never raw); when no key exists or coach
   is not yet deployed, render the empty state ("No key — coach off", `ds-badge--warn`). Render the Add /
   store / delete / enable controls to match the artboard but keep them **shell-only** — `PUT`/`DELETE
   /coach/key` become functional in S11. Do not persist a fake key.

## Constraints
- Reuse `design-system/theme.css` tokens/components **verbatim** (no Tailwind); keep the dark "landscape console" theme.
- Respect service boundaries: only `identity` touches schema `identity`; `review` reads account data via identity's internal API, never cross-schema SQL ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).
- `schema-per-service` + goose (embedded, startup + advisory lock) + `sqlc`/`pgx`; internal calls are HTTP/JSON on ClusterIP with the gateway-minted RS256 JWT.
- Server-authoritative: `PATCH /me` updates only the caller's own account; times persisted UTC, reminder times computed in the account's timezone.
- Notifications consumer stays idempotent (dedupe on `event_id`); the sweep/worker already exists from S07 — extend, don't duplicate.
- **Do not fake coach key storage** — the API-keys store/delete are shell-only until S11.
- Read-only rootfs, slog JSON logs, `/healthz`+`/readyz` unchanged; pull-based GitOps — never `kubectl apply`, deploy is Flux auto-bumping the identity + gateway image tags from `main`.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables
- `identity`: `sqlc` queries + `PATCH /me` handler (profile / budget / timezone / reminders); goose migration only if a column is missing.
- `identity`: `POST /onboarding/step` handling steps 2-3 (`budget_set`, `key_added`, `completed_at`); gateway/SPA onboarding routing off `GET /me`.
- `review`: notifications worker consuming account budget/timezone/reminder prefs (via identity internal API) to gate + localize reminders.
- `web/`: Settings screen (Profile, Study budget, Reminders live; API-keys section shell reading `GET /coach/key` + empty state) and onboarding steps 2-3, both matching the artboards.
- New identity + gateway container images auto-deployed via Flux (no new infra HelmRelease).

## Update status
- As each task lands, set its row in [`../sprints/sprint-10.md`](../sprints/sprint-10.md) to ✅ (🔄 while in progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row (this sprint carries no milestone). Add a **Decisions log** line for any notable call (e.g. how `review` sources account data for reminders).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)
- [ ] A user edits profile, study budget, timezone, and reminder preferences; changes persist via `PATCH /me`.
- [ ] First-run onboarding runs all 3 steps (path -> budget -> key, key skippable) and marks completion (`onboarding.completed_at` set); a completed account is not re-routed through the flow.
- [ ] Reminders respect the study budget + timezone (daily reminder fires at the local time; disabled prefs suppress).
- [ ] Settings screen matches the artboard, including the API-keys section shell (functional store/delete deferred to S11).
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
