# Sprint 07 — Mistake journal & notifications

> **Milestone:** M4 — repetition + mistakes loop closed   ·   **Design phase:** D4 Spaced repetition
> **Prereqs:** [S06](sprint-06.md)   ·   **Unblocks:** [S09](sprint-09.md), [S10](sprint-10.md)
> **Execute with:** [`../prompts/prompt-s07.md`](../prompts/prompt-s07.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Status |
|---|------|--------|
| 1 | mistake journal aggregate | ⬜ |
| 2 | weekly weak-area | ⬜ |
| 3 | notifications worker (in review, v1) | ⬜ |
| 4 | Mistakes screen + Dashboard weak-area | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Close the repetition + mistakes loop inside the existing **review** service. Every below-Clean outcome
and every failed re-solve opens a **mistake_entry** (pattern pre-filled, 8-category picker, root-cause
field); an entry **closes after 2 successful revisits** and **re-opens on a later fail**. A **weekly
weak-area** rolls up the top category, and a **notifications worker** turns due reviews into in-app
reminders. Wiring the **Mistakes** screen and the Dashboard weak-area + due-reminders cards lands
milestone **M4** — the five-touch scheduler ([S06](sprint-06.md)) plus this sprint make the full
repetition + mistakes loop real.

## Scope

**In**
- Extend the **review** service (created in [S06](sprint-06.md)) — no new service, no new HelmRelease.
  New goose migrations add `mistake_entry`, `weak_area_snapshot`, and `reminder` to schema `review`.
- **Mistake journal aggregate:** open on below-Clean `problem_solved` and on failed re-solves
  (flow 2 / flow 3), pattern pre-filled + 8-category picker + root-cause; close after 2 clean
  revisits, re-open on later fail. Emit `xlearn.review.mistake_opened` / `mistake_closed` via outbox.
- **Weekly weak-area:** compute `weak_area_snapshot` (top category + `counts_json`) per account at the
  week boundary in the account timezone; internal `GET /weak-area/current`.
- **Notifications worker:** a separable worker in the review binary consuming `xlearn.review.revision_due`
  plus the account study-budget window, writing `reminder` rows (in-app channel only, v1).
- Gateway routes for `GET /mistakes?status=`, `POST /mistakes`, `PATCH /mistakes/{id}`, `GET /weak-area`;
  reminders surface through the existing `GET /dashboard` **agg**.
- The **Mistakes** screen (`/xlearn/dsa/mistakes`) and the Dashboard **weak-area** + **due-reminders**
  cards, wired to real APIs with `theme.css` verbatim.

**Out (later sprints)**
- Email / push notifications — deliberately deferred; a second channel is what would justify splitting
  notifications into its own service later.
- Mock + progress analytics (D5) — [S08](sprint-08.md) / [S09](sprint-09.md). The `mistake_opened` /
  `mistake_closed` events are consumed by assessment's weak-area / outcome-mix projections in
  [S09](sprint-09.md); no consumer reaction is required this sprint.

## Tasks

### 1 · mistake journal aggregate

Add the `mistake_entry` table to schema `review` via a new goose migration (embedded, run on startup
under the advisory lock — see [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) and
[`../../architecture/data-model.md`](../../architecture/data-model.md)): `id`, `account_id`,
`problem_id`, `pattern`, `mistake`, `root_cause`, `insight`, `category` (8-enum), `revisit_date`,
`status` (`open`/`closed`), `revisit_count`. Extend the durable pull-consumer that S06 already runs on
`practice.*` (idempotent, dedupe on `event_id` via the review `inbox`/offset) so it also reacts to
mistake triggers per [`../../architecture/events.md`](../../architecture/events.md) flow 2 and flow 3:

- On `xlearn.practice.problem_solved` with an outcome **below Clean** (`rough`/`assisted`/`miss`),
  and on a **failed re-solve** in the scheduler (S06 flow 3), **open** a `mistake_entry`: `pattern`
  pre-filled from the problem (soft reference by `problem_id`, resolved from the event payload / the
  curriculum `GET /problems/{id}` `pattern` field), `category` seeded to the 8-category picker, empty
  `root_cause`, `status=open`, `revisit_count=0`. Write the domain row **and** an `outbox` row for
  `xlearn.review.mistake_opened` (`data`: `problem_id`, `category`) in **one transaction**.
- The 8 categories ([R-MJ2](../../prd/xlearn-prd.md#64-mistake-journal)) are exactly: **Misread ·
  Wrong pattern · Right pattern wrong state · Off-by-one/boundary · Language bug · Complexity
  misjudged · Communication · Time management** — an 8-value enum, stored not as free text.
- **Close/re-open state machine** ([R-MJ4](../../prd/xlearn-prd.md#64-mistake-journal)): a clean
  revisit (an auto-pass `touch_result` from the S06 scheduler for the same `account_id`+`problem_id`)
  increments `revisit_count`; at **2** clean revisits set `status=closed` and emit
  `xlearn.review.mistake_closed` (`data`: `mistake_id`, `problem_id`). A later **fail** on a closed
  entry **re-opens** it (`status=open`, reset `revisit_count=0`), emits `mistake_opened`, and the
  scheduler re-enters the problem at Day 1 (the S06 reset path). Count clean revisits **only** — a fail
  never counts toward the 2.
- API (review internal, exposed through the gateway per [`../../architecture/api.md`](../../architecture/api.md)):
  `GET /mistakes?status=open|closed` (journal list), `POST /mistakes` (create/edit root cause + insight
  + category), `PATCH /mistakes/{id}` (update status / revisit). Add the matching gateway routes.

### 2 · weekly weak-area

Add the `weak_area_snapshot` table (`id`, `account_id`, `week_of`, `top_category`, `counts_json`).
Compute one snapshot per account **weekly**: per-category counts of open entries over the week, the
`top_category` = the max ([R-MJ3](../../prd/xlearn-prd.md#64-mistake-journal)). Run it on the same
periodic tick as the S06 due-sweep (a review-internal scheduled job; idempotent per `week_of`).
The **week boundary is defined in the account timezone** (soft reference to the identity
`account.timezone`, resolved via the identity internal API `GET /accounts/{id}` on ClusterIP — no
cross-schema read, per [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). Expose the current
banner as review-internal `GET /weak-area/current`, surfaced through the gateway as `GET /weak-area`:
returns `top_category`, its count, and the supporting entries. This is the single signal both the
Mistakes weekly banner and the Dashboard weak-area card read.

### 3 · notifications worker (in review, v1)

Add the `reminder` table (`id`, `account_id`, `kind`, `due_at`, `delivered_at`). Build a
**notifications worker** as a **cleanly separable package** (e.g. `internal/review/notifications/`)
inside the review binary — not a new service — that consumes `xlearn.review.revision_due` (emitted by
the S06 due-sweep, flow 4 in [`../../architecture/events.md`](../../architecture/events.md)) and,
combined with the account **study-budget window** (identity `account.study_budget_json` + `timezone`,
via the identity internal API), writes `reminder` rows scheduled inside that window. **In-app channel
only** in v1 — no email/push. The worker is idempotent (dedupe on `event_id`). Keep its code path
isolated so a later split into a standalone notifications service (per
[ADR-0003](../../adr/0003-service-decomposition.md) and [`../../architecture/services.md`](../../architecture/services.md))
is a lift-and-shift, not a rewrite. Reminders are read by the gateway `GET /dashboard` **agg** for the
"Revisions due today" surface.

### 4 · Mistakes screen + Dashboard weak-area

Build the **Mistakes** screen at `/xlearn/dsa/mistakes`, wiring
[`design-system/screens/Mistakes.dc.html`](../../../design-system/screens/Mistakes.dc.html) with real API calls and
`theme.css` verbatim (no Tailwind):

- The journal **table** with the 8 columns from [R-MJ1](../../prd/xlearn-prd.md#64-mistake-journal):
  **Problem · Pattern · My mistake -> root cause -> correct insight · Category · Revisit · Status**
  (`GET /mistakes`). The **Status** cell reads `Open · n/2` for open entries and `Closed` for closed;
  category badges follow the artboard (Off-by-one/boundary uses `--ds-warn` amber, Communication `info`,
  Right-pattern-wrong-state `violet`, others a plain chip).
- The **open/closed filter** (segmented All / Open / Closed) and the **category chips** with per-category
  counts + the `N open` / `N closed` header, matching the artboard interactions.
- The **weekly weak-area banner** ("This week's weak area" -> top category + count + supporting copy +
  "Show these N") from `GET /weak-area`, plus the closed-entry re-open note.

Wire the **Dashboard** cards ([`design-system/screens/Dashboard.dc.html`](../../../design-system/screens/Dashboard.dc.html)):
the **"Weak area this week"** card (top category + entry count + copy, from `GET /weak-area`) and the
**"Revisions due today"** / reminders surface (from `GET /dashboard`), reflecting that due reviews take
priority over new problems ([R-SR5](../../prd/xlearn-prd.md#63-five-touch-spaced-repetition)).

## Acceptance criteria

- [ ] A below-Clean outcome or a failed re-solve **opens** a mistake with the **pattern pre-filled** and
      the **8-category picker**; `xlearn.review.mistake_opened` is emitted via the outbox.
- [ ] An entry **closes after 2 successful revisits** (`mistake_closed` emitted) and **re-opens on a
      later fail** (`mistake_opened`, revision reset to Day 1) — clean revisits only count toward the 2.
- [ ] The **weekly weak-area** (top category) is computed per account with the **week boundary in the
      account timezone** and surfaces on the **Dashboard** banner via `GET /weak-area`.
- [ ] Due reviews generate **in-app reminders** (no email/push); the **Mistakes** screen matches the
      artboard (columns, filters, weak-area banner) — **M4: repetition + mistakes loop closed**.

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- **Close/re-open counting must be exact:** 2 **clean** revisits close an entry; a later fail re-opens
  it and resets the count. A fail never counts toward the 2, and re-open must also re-enter the problem
  at Day 1 (the S06 scheduler reset) — keep the two aggregates' state changes in one transaction path.
- **Weak-area is weekly:** define `week_of` and the week boundary in the **account timezone**, not UTC,
  or snapshots straddle the wrong day for users far from UTC. The recompute must be idempotent per
  `week_of` so re-running the tick does not double-count.
- **The notifications worker shares the review deployment in v1** — keep its code path cleanly
  separable (own package, own consumer) so a later split into its own service is mechanical.
- **Background workers have no user JWT:** the weak-area recompute and reminder worker resolve account
  timezone / study-budget via the identity internal API without a request context — decide and record
  the service-to-service auth path for workers as an ADR ([ADR-0006](../../adr/0006-authn-authz.md)).
