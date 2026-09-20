# Prompt — Sprint 06 · Review scheduler & the Revision queue

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-06.md`](../sprints/sprint-06.md)   ·   **Milestone:** none   ·   **Prereqs:** [S05](../sprints/sprint-05.md) (practice emits `xlearn.practice.*` over NATS)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions (dark theme, service boundaries, GitOps, ship at session end (AGENT.md land-and-sync)).
- [`../../adr/0004-inter-service-comms-and-events.md`](../../adr/0004-inter-service-comms-and-events.md) — NATS JetStream, durable pull consumers, transactional outbox, idempotent inbox, and the **event-driven + periodic-sweep hybrid** scheduler. The contract for this whole sprint.
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — schema-per-service, no cross-schema FKs, goose migrations.
- [`../../architecture/services.md`](../../architecture/services.md) — the **review** service: what it owns, its API, what it emits/consumes.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `review` table shapes (`revision_item`, `touch_result`, `mistake_entry`, `weak_area_snapshot`, `reminder`, `outbox`, `inbox`).
- [`../../architecture/events.md`](../../architecture/events.md) — the envelope + subjects, and **flows 1, 3, 4** (schedule / auto-score / sweep). Ground every handler in these.
- [`../../architecture/api.md`](../../architecture/api.md) — the **Revision & mistakes** gateway surface (`GET /revision/due`, `POST /revision/{itemId}/score`).
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — **6.3 Five-touch spaced repetition** (R-SR1..R-SR6); the auto-score thresholds are load-bearing.
- `design-system/screens/Revision.dc.html` — the Revision artboard (design intent only, **not runnable** — read for layout/copy/interaction), and [`../../../design-system/README.md`](../../../design-system/README.md) + `design-system/theme.css` for tokens/components.
- Sibling `../infra` repo: inspect `apps/airlift.yaml` / `apps/landscape.yaml` (HelmRelease template), the image-automation config, and `apps/secrets/*.enc.yaml` before adding any deploy config.

## Context

The build is at [S05](../sprints/sprint-05.md): the **practice** service owns the guided Problem flow,
server timers, and reveal penalty, and publishes `xlearn.practice.problem_solved`,
`xlearn.practice.attempt_logged`, and `xlearn.practice.solution_revealed_early` to NATS JetStream via a
transactional outbox. NATS/JetStream is already running on the cluster. **This sprint builds the first
consumer** — the **review** service: the five-touch scheduler and the Revision queue, the heart of the
spaced-repetition method. No milestone gate; it unblocks [S07](../sprints/sprint-07.md) (mistakes) and
[S09](../sprints/sprint-09.md) (progress projections).

## Do this (in order)

1. **review service + schema + durable consumers** — Scaffold `cmd/review/` (main + wiring) and
   `internal/review/` (domain, store, HTTP, consumers), reusing `internal/platform` (config, slog JSON
   logging, `httpx`) and the shared `internal/platform/events` helpers. Own schema `review`; write embedded
   **goose** migrations (run at startup under an advisory lock) for `revision_item` (`id`, `account_id`,
   `problem_id`, `touch_level` 1..5 = Day 1/3/7/21/45, `due_date`, `surfaced_at`, `status`
   `pending`/`passed`/`failed`), `touch_result` (`revision_item_id`, `named_pattern_secs`, `solved_in_timer`,
   `stated_complexity`, `auto_pass`, `scored_at`, mock-mode flag), plus `outbox` and `inbox`. (The
   `mistake_entry`, `weak_area_snapshot`, and `reminder` tables are added in [S07](../sprints/sprint-07.md).)
   Generate queries with **sqlc/pgx**. Declare the `XLEARN_REVIEW` JetStream stream and subscribe **durable pull
   consumers** to `XLEARN_PRACTICE` (`xlearn.practice.*`); every handler dedupes on `event_id` via `inbox`.
   Add `/healthz` + `/readyz`, `deploy/review.Dockerfile` (read-only rootfs), and a CI path filter. In
   `../infra`, add `infra/apps/xlearn-review.yaml` by copying `apps/airlift.yaml` / `apps/landscape.yaml`
   (shared `charts/project`); image `ghcr.io/sujaykumarsuman/xlearn-review`; an image-automation entry with
   the `# {"$imagepolicy": "flux-system:xlearn-review:tag"}` setter and ImagePolicy semver `>=0.1.0`; the
   per-service DB role/schema via the infra 4-step pattern; a SOPS/age secret under
   `infra/apps/secrets/*.enc.yaml`. Internal only — no Traefik route.

2. **five-touch scheduler (event-driven)** — On `xlearn.practice.problem_solved` with `first_solve` true,
   schedule five `revision_item` rows at **Day 1/3/7/21/45** from the solve date (`touch_level` 1..5,
   `status=pending`). On `xlearn.practice.solution_revealed_early`, schedule the **owed attempt in 3 days**
   as a follow-up touch (R-PF2). Emit `xlearn.review.revision_scheduled` (`problem_id`, `touch_level`,
   `due_date`) per touch through the transactional **outbox** (domain rows + outbox row in one tx; the relay
   publishes to `XLEARN_REVIEW`). **Upsert** on (`account_id`, `problem_id`, `touch_level`) and tolerate
   out-of-order delivery so re-delivered events never double-schedule.

3. **auto-scoring (advance or reset)** — Implement the review endpoint the gateway surfaces as
   `POST /xlearn/api/revision/{itemId}/score` (internal `POST /revisions/{id}/score`). Record a
   `touch_result` from `named_pattern_secs`, `solved_in_timer`, `stated_complexity`. **Auto-pass iff**
   pattern named **< 2 min** (`named_pattern_secs < 120`) **AND** solved within the **20-min** timer **AND**
   complexity stated (R-SR2) — any one failing is a fail. **Pass** -> set `status=passed`, advance
   `touch_level`, compute the next `due_date`, emit `xlearn.review.revision_scheduled` for the next touch.
   **Fail** -> reset to **Day 1** (`touch_level=1`, new `due_date`, `status=failed`) per R-SR3 (opening the
   mistake entry is [S07](../sprints/sprint-07.md)). **Day 21 and Day 45** run under **mock conditions**
   (stricter) per R-SR4 — set the `touch_result` mock-mode flag.

4. **periodic sweep + Revision screen** — Add a cron-like sweep (~15m) that selects
   `revision_item WHERE due_date <= now() AND surfaced_at IS NULL`, marks `surfaced_at` (**idempotent**),
   and emits `xlearn.review.revision_due` per item (events.md flow 4). Index (`account_id`, `due_date`,
   `surfaced_at`) so the scan stays cheap. Implement `GET /revision/due` returning the **prioritised** queue
   (reviews before new work, R-SR5), grouped by touch day. Then build the **Revision** screen in `web/`
   (React + TS) from `design-system/screens/Revision.dc.html`: the "re-solves from a blank editor, not
   re-reads" eyebrow, the Day-grouped queue with D1/D3/D7/D21/D45 touch dots, "Re-solve from blank" on a
   **20:00** timer, and the auto-score result panel (pattern named / solved in-timer / complexity stated ->
   advances to Day 3, or reset to Day 1). Reuse `design-system/theme.css` verbatim.

## Constraints

- **Design:** reuse `design-system/theme.css` verbatim (no Tailwind); keep the dark theme; difficulty tokens
  Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. The `.dc.html` artboard is reference only — do not ship it.
- **Boundaries:** schema-per-service (`review` only), no cross-schema reads/writes/FKs; cross-service refs are
  bare ids. goose migrations (embedded, startup + advisory lock) + sqlc/pgx.
- **Async correctness:** durable pull consumers on `XLEARN_PRACTICE`; **idempotent** handlers (dedupe on
  `event_id` in `inbox`); **upsert** and assume no cross-subject ordering; emit via the transactional
  **outbox** only. The sweep must be idempotent (`surfaced_at`) and cheap (indexed).
- **Thresholds:** auto-pass exactly matches R-SR2 (pattern < 2 min, solved in-timer, complexity stated);
  Day 21/45 mock conditions.
- **Ops:** `/healthz`+`/readyz`, slog JSON logs, read-only rootfs; internal ClusterIP (no route). Mirror
  `../infra` conventions exactly (copy `apps/airlift.yaml`/`apps/landscape.yaml`; GHCR image; Flux
  image-automation setter; ImagePolicy `>=0.1.0`; SOPS secret). **Pull-based GitOps — never `kubectl apply`.**
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables

- `cmd/review/` + `internal/review/` (domain, store, HTTP handlers, JetStream consumers, sweep worker).
- Schema `review` goose migrations (`revision_item`, `touch_result`, `outbox`, `inbox`) + sqlc queries.
  (The `mistake_entry`, `weak_area_snapshot`, and `reminder` tables are added in [S07](../sprints/sprint-07.md).)
- Consumers on `xlearn.practice.problem_solved` / `solution_revealed_early`; emissions
  `xlearn.review.revision_scheduled` + `xlearn.review.revision_due`.
- `GET /revision/due` + `POST /revision/{itemId}/score` (via the gateway BFF) and the periodic sweep.
- The **Revision** screen in `web/` matching the artboard.
- `deploy/review.Dockerfile`, a CI path filter, and the `infra` additions
  (`infra/apps/xlearn-review.yaml` HelmRelease + image-automation entry + DB role/schema + SOPS secret).

## Update status

- As each task lands, set its row in [`../sprints/sprint-06.md`](../sprints/sprint-06.md) to ✅ (🔄 while in
  progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row for S06. Add a
  **Decisions log** line for any notable call.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] Solving a problem (first clean) schedules **5 `revision_item` rows** at Day 1/3/7/21/45; a
      `xlearn.review.revision_scheduled` event is emitted per touch.
- [ ] A re-solve is **auto-scored**: pass advances the touch (next `due_date` + event); fail resets to
      Day 1; Day 21/45 use mock conditions.
- [ ] The sweep surfaces due-today **reliably even after days offline** (idempotent `surfaced_at`), and
      `GET /revision/due` prioritises reviews over new work.
- [ ] **review** is deployed to prod on ClusterIP via Flux; the **Revision** screen matches the artboard.
- [ ] Re-delivered practice events do **not** double-schedule (dedupe on `event_id` + upsert).
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
