# Prompt — Sprint 07 · Mistake journal & notifications

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-07.md`](../sprints/sprint-07.md)   ·   **Milestone:** M4   ·   **Prereqs:** [S06](../sprints/sprint-06.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, working rules.
- [`../../architecture/services.md`](../../architecture/services.md) — the **review** service (scheduler + mistakes + notifications + sweep) and its internal API.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `review`: the `mistake_entry`, `weak_area_snapshot`, `reminder` tables you add this sprint.
- [`../../architecture/events.md`](../../architecture/events.md) — **flow 2** (below-clean -> mistake opened), **flow 3** (re-solve fail -> reset + mistake), **flow 4** (due sweep -> reminders); the `mistake_opened` / `mistake_closed` / `revision_due` subjects and the outbox/idempotent-consumer rules.
- [`../../architecture/api.md`](../../architecture/api.md) — BFF surface: `GET /mistakes?status=`, `POST /mistakes`, `PATCH /mistakes/{id}`, `GET /weak-area`, and the `GET /dashboard` **agg**.
- [`../../prd/xlearn-prd.md#64-mistake-journal`](../../prd/xlearn-prd.md#64-mistake-journal) — R-MJ1..R-MJ4: columns, the 8 categories, weekly weak-area, close-after-2 / re-open-on-fail.
- [`../../adr/0003-service-decomposition.md`](../../adr/0003-service-decomposition.md) (why notifications lives in review for v1), [`../../adr/0004-inter-service-comms-and-events.md`](../../adr/0004-inter-service-comms-and-events.md) (NATS + outbox + idempotency), [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) (schema-per-service + goose), [`../../adr/0009-deployment-and-gitops.md`](../../adr/0009-deployment-and-gitops.md) (Flux GitOps).
- [`design-system/screens/Mistakes.dc.html`](../../../design-system/screens/Mistakes.dc.html) and `design-system/screens/Dashboard.dc.html` — layout / copy / interaction intent (design references, **not runnable**). Reuse [`../../../design-system/README.md`](../../../design-system/README.md) tokens/components + `design-system/theme.css` verbatim.

## Context

[S06](../sprints/sprint-06.md) stood up the **review** service: the five-touch scheduler (Day 1/3/7/21/45),
auto-scored re-solves, the periodic due-sweep, and the **Revision** queue — with schema `review`
(`revision_item`, `touch_result`, `inbox`, `outbox`), a durable pull-consumer on `practice.*`, and its infra
(HelmRelease + image-automation + DB role/schema). **This sprint extends that same service** (no new
service): it adds the **mistake journal**, the **weekly weak-area**, and an in-app **notifications
worker**, then wires the **Mistakes** screen and the Dashboard weak-area + due-reminders cards. Landing
all four closes design phase **D4** and milestone **M4** (the repetition + mistakes loop).

## Do this (in order)

1. **mistake journal aggregate** — Add a goose migration to schema `review` (embedded, run on startup
   under the advisory lock) creating `mistake_entry` (`id`, `account_id`, `problem_id`, `pattern`,
   `mistake`, `root_cause`, `insight`, `category` 8-enum, `revisit_date`, `status` `open`/`closed`,
   `revisit_count`). Write sqlc queries (`sqlc generate`, commit output). Extend the S06 `practice.*`
   pull-consumer (idempotent, dedupe on `event_id`): on `xlearn.practice.problem_solved` with outcome
   **below Clean** (`rough`/`assisted`/`miss`) **and** on a **failed re-solve** (flow 3), open a
   `mistake_entry` with `pattern` pre-filled (from the event payload / curriculum `GET /problems/{id}`),
   the 8-category picker seeded (**Misread · Wrong pattern · Right pattern wrong state ·
   Off-by-one/boundary · Language bug · Complexity misjudged · Communication · Time management**), empty
   `root_cause`, `status=open`, `revisit_count=0`; in the **same transaction** write an `outbox` row for
   `xlearn.review.mistake_opened` (`data`: `problem_id`, `category`). Implement the state machine: a
   **clean revisit** (auto-pass `touch_result` for that `account_id`+`problem_id`) increments
   `revisit_count`; at **2**, set `status=closed` + emit `xlearn.review.mistake_closed` (`mistake_id`,
   `problem_id`); a later **fail** on a closed entry re-opens it (`status=open`, `revisit_count=0`),
   emits `mistake_opened`, and triggers the S06 Day-1 reset. Count clean revisits only. Add review
   endpoints `GET /mistakes?status=open|closed`, `POST /mistakes`, `PATCH /mistakes/{id}` and the
   matching gateway routes.

2. **weekly weak-area** — Add `weak_area_snapshot` (`id`, `account_id`, `week_of`, `top_category`,
   `counts_json`) + sqlc queries. On the same periodic tick as the S06 due-sweep, compute one snapshot
   per account: per-category counts of open entries for the week, `top_category` = the max; **define
   `week_of` / the week boundary in the account timezone** (resolve `account.timezone` via the identity
   internal API `GET /accounts/{id}` on ClusterIP — no cross-schema read). Make the recompute idempotent
   per `week_of`. Expose review-internal `GET /weak-area/current` (top category + count + supporting
   entries) and surface it through the gateway as `GET /weak-area`.

3. **notifications worker (in review, v1)** — Add `reminder` (`id`, `account_id`, `kind`, `due_at`,
   `delivered_at`) + sqlc queries. Build a **separable** package (e.g. `internal/review/notifications/`)
   inside the review binary — **not** a new service — that consumes `xlearn.review.revision_due` (from
   the S06 sweep, flow 4) and, combined with the account **study-budget window** (identity
   `study_budget_json` + `timezone`), writes `reminder` rows inside that window. **In-app channel only**
   (no email/push). Idempotent (dedupe on `event_id`). Keep the code path isolated so a later split into
   a standalone service is lift-and-shift. Surface reminders via the gateway `GET /dashboard` **agg**.

4. **Mistakes screen + Dashboard weak-area** — Build `/xlearn/dsa/mistakes` from
   `design-system/screens/Mistakes.dc.html` with `theme.css` verbatim: the 8-column journal table
   (**Problem · Pattern · My mistake -> root cause -> correct insight · Category · Revisit · Status**;
   Status cell `Open · n/2` or `Closed`; Off-by-one badge `--ds-warn`, Communication `info`,
   Right-pattern-wrong-state `violet`), the **All / Open / Closed** segmented filter + category chips
   with counts + the `N open` / `N closed` header, and the **weekly weak-area banner** ("This week's
   weak area") from `GET /weak-area`, plus the closed-entry re-open note. Then wire the **Dashboard**
   cards from `Dashboard.dc.html`: the **"Weak area this week"** card (`GET /weak-area`) and the
   **"Revisions due today"** / reminders surface (`GET /dashboard`), reflecting that due reviews take
   priority over new problems.

## Constraints

- **Extend the existing review service** — do **not** create a new service, schema, HelmRelease, image
  policy, or DB role. This sprint is additive goose migrations to schema `review` + new handlers/workers;
  the existing review Deployment auto-deploys the new image tag via Flux image-automation.
- **Schema-per-service + goose + sqlc/pgx:** touch only schema `review`; migrations embedded and run on
  startup under the advisory lock; commit generated sqlc output (CI runs `sqlc diff`).
- **Outbox + idempotent consumers:** write the domain row and the `outbox` row in one transaction; the
  consumer/worker dedupe on `event_id`. Cross-service data (problem `pattern`, account `timezone` /
  study-budget) is a **soft reference by id**, resolved via event payload or an internal API — never a
  cross-schema read.
- **Server-authoritative state:** the close/re-open counting, the weekly snapshot, and reminder timing
  are computed server-side; the client only renders.
- **Frontend:** reuse `theme.css` tokens/components verbatim (no Tailwind); match the artboards; keep the
  dark theme. Difficulty tokens where shown are Easy=`--ds-ok` / Medium=`--ds-warn` / Hard=`--ds-err`.
- **GitOps:** deploy is **pull-based via Flux** — never `kubectl apply` by hand; mirror `../infra`
  conventions. Read-only rootfs containers. **Do not commit or push unless asked.**

## Deliverables

- New goose migrations + sqlc queries in schema `review`: `mistake_entry`, `weak_area_snapshot`, `reminder`.
- Mistake journal aggregate (open/close/re-open state machine) emitting `xlearn.review.mistake_opened` /
  `mistake_closed` via the outbox; endpoints `GET /mistakes?status=`, `POST /mistakes`, `PATCH /mistakes/{id}`.
- Weekly weak-area computation (timezone-correct `week_of`) + `GET /weak-area/current` -> gateway `GET /weak-area`.
- In-app notifications worker (`internal/review/notifications/`) consuming `revision_due` -> `reminder` rows.
- Gateway routes for the above; reminders threaded into `GET /dashboard`.
- The **Mistakes** screen (`/xlearn/dsa/mistakes`) + the Dashboard **weak-area** and **due-reminders** cards.

## Update status

- As each task lands, set its row in [`../sprints/sprint-07.md`](../sprints/sprint-07.md) to ✅ (🔄 while
  in progress); set _Overall_ to ✅ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row **and** the
  **Milestones** table (**M4**). Add a **Decisions log** line for any notable call (e.g. the
  worker service-auth path, the timezone week boundary).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] A below-Clean outcome or a failed re-solve opens a mistake with the **pattern pre-filled** + the
      **8-category picker**; `xlearn.review.mistake_opened` is emitted via the outbox.
- [ ] An entry **closes after 2 successful revisits** (`mistake_closed`) and **re-opens on a later fail**
      (`mistake_opened`, revision reset to Day 1); only clean revisits count toward the 2.
- [ ] The **weekly weak-area** (top category) is computed with the **week boundary in the account
      timezone** and surfaces on the **Dashboard** banner via `GET /weak-area`.
- [ ] Due reviews generate **in-app reminders** (no email/push); the **Mistakes** screen matches the
      artboard — **M4: repetition + mistakes loop closed**.
- Do not commit or push unless asked.
