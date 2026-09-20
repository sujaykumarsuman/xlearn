# Prompt — Sprint 09 · Progress projections & the Dashboard ★

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-09.md`](../sprints/sprint-09.md)   ·   **Milestone:** M5 — mock + analytics complete   ·   **Prereqs:** [S06](../sprints/sprint-06.md), [S07](../sprints/sprint-07.md), [S08](../sprints/sprint-08.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions: match `theme.css`, respect service boundaries, ship at session end (AGENT.md land-and-sync).
- [`../sprints/sprint-09.md`](../sprints/sprint-09.md) — this sprint's scope, tasks and acceptance criteria (source of the checklist below).
- [`../../architecture/services.md`](../../architecture/services.md) — `assessment` (mock + **progress projections**) and the gateway's screen-aggregation role.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — the `proj_coverage`/`proj_heatmap`/`proj_mastery`/`proj_outcome_mix` tables and the `inbox` dedupe table in schema `assessment`.
- [`../../architecture/events.md`](../../architecture/events.md) — which event feeds which projection, at-least-once + dedupe, ordering caveats, and the **replay** note.
- [`../../architecture/api.md`](../../architecture/api.md) — the external contracts `GET /progress` **agg** and `GET /dashboard` **agg**.
- [`../../adr/0004-inter-service-comms-and-events.md`](../../adr/0004-inter-service-comms-and-events.md) — outbox + idempotent consumers + replay (the projection reliability model).
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — no cross-schema joins; compose in gateway or read `assessment` projections.
- [`../../adr/0003-service-decomposition.md`](../../adr/0003-service-decomposition.md) — why mock + progress live in one `assessment` service.
- [`../../../design-system/screens/Progress.dc.html`](../../../design-system/screens/Progress.dc.html) + [`../../../design-system/screens/Dashboard.dc.html`](../../../design-system/screens/Dashboard.dc.html) — layout / copy / interaction intent (design references only — **not runnable, do not ship them**).
- [`../../../design-system/README.md`](../../../design-system/README.md) — tokens + component classes to reuse verbatim.
- Sibling infra: `infra/apps/xlearn-assessment.yaml` and `infra/apps/xlearn-gateway.yaml` already exist (HelmRelease + Flux image-automation from S08/S01) — confirm the `# {"$imagepolicy": "flux-system:xlearn-<svc>:tag"}` setter is present; this sprint needs **no new** infra file.

## Context

By now practice (S05) emits `attempt_logged`/`problem_solved`/`solution_revealed_early`, review (S06/S07)
emits `revision_scheduled`/`revision_due`/`mistake_opened`/`mistake_closed`, and assessment (S08) runs
the 45-min Mock and emits `mock_completed`. The `assessment` service and gateway are already deployed.
**This sprint** turns those events into read-model projections in `assessment`, adds the progress read
API + the Progress screen, and finalizes the Dashboard ★ via BFF aggregation — completing D5 and reaching M5.

## Do this (in order)

1. **projection consumers** — In `internal/assessment`, add durable JetStream **pull** consumers on the practice, review and assessment streams, subscribing to `xlearn.practice.*`, `xlearn.review.*` and `xlearn.assessment.mock_completed`. Add a goose migration for the `assessment.inbox` table and **dedupe every handler on `event_id`**. **Upsert** into the read-model tables so out-of-order delivery is safe: `problem_solved`/`attempt_logged` -> `proj_coverage` + `proj_mastery` (by pattern) + `proj_outcome_mix` (by outcome); `revision_scheduled` + solves -> `proj_heatmap` (reviews + solves per day); `mistake_opened`/`mistake_closed` -> `proj_outcome_mix` / weak-area context. Keep handlers **pure functions of the event log** (no external reads, no wall-clock state branching) so a rebuild replays deterministically. Add the `sqlc` queries; consumers live inside the assessment binary (no new deployable). Touch only the `assessment` schema.
2. **progress API + Progress screen** — Add assessment endpoints `GET /progress/summary` (solved/151, streak, Day-7 retention, mock average), `GET /progress/heatmap` (`proj_heatmap`), `GET /progress/mastery` (`proj_mastery`), rubric trend via `GET /mocks/trend` (S08 `mock_session` + `rubric_score`), and outcome mix from `proj_outcome_mix`. Build the Progress screen from `Progress.dc.html`: four stat tiles, revision-activity heatmap (teal ramp), completion-by-phase table, pattern-mastery bars (mastery ramp, not difficulty tokens), mock rubric-trend line with W13/W15/pre-interview targets, outcome-mix bar + legend (Clean `--ds-ok`, Rough `--ds-warn`, Assisted `--ds-info`, Miss `--ds-err`). SPA reads it through gateway `GET /progress` **agg** (assessment + review).
3. **BFF Dashboard aggregation + finalize Dashboard ★** — Implement gateway `GET /dashboard` **agg** fanning out to curriculum (this week's next problems), practice (state + streak + solved), review (`GET /revision/due`, `GET /weak-area`) and assessment (`proj_coverage` stats + mock best). Compose "Today": daily plan, due reviews, weak-area callout, streak/stats — ordering **due reviews ahead of new problems** (reviews-before-new-work). Finalize `Dashboard.dc.html` ★ with real data, replacing the S01 skeleton: quick-stats row, Today's plan cards (difficulty chip uses green/amber/red difficulty tokens), week-progress bar, Revisions-due panel with five-touch dots, Weak-area card. Parallelize the fan-out, set per-call timeouts, watch p95, cache only where safe.
4. **replay/rebuild verification** — Document and dry-run the rebuild: truncate a projection table (e.g. `proj_mastery`), reset the durable consumer/offset, and **replay the stream from the start** so the projection upserts back to an identical result. Then reconcile Progress + Dashboard numbers against the source services (practice solved/streak, review due/weak-area, mock trend). Write the runbook and capture the rebuild approach as an ADR if notable.

## Constraints

- Reuse `design-system/theme.css` tokens/components **verbatim** (dark theme, no Tailwind); difficulty chips stay Easy=`--ds-ok` / Medium=`--ds-warn` / Hard=`--ds-err`.
- Schema-per-service: touch only the `assessment` schema; goose migrations (embedded, startup + advisory lock) + `sqlc`/`pgx`. **No cross-schema SQL** — screens compose in the gateway or read `assessment` projections.
- Idempotent consumers: dedupe on `event_id` via `inbox`; **upsert** (never assume per-subject order); keep projections a **pure function of the event log** so replay is safe.
- `GET /dashboard` is a 4-service fan-out — parallelize, time out, watch p95; the `assessment` and `gateway` services keep `/healthz`+`/readyz` and slog JSON logs, read-only rootfs.
- Pull-based GitOps: **never** `kubectl apply`. `assessment` + `gateway` already have HelmRelease + Flux image-automation; images auto-bump on merge to `main` (build-semver). No new infra file this sprint.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables

- `internal/assessment` projection consumers + `assessment.inbox` goose migration + `sqlc` queries; upsert logic for `proj_coverage`/`proj_heatmap`/`proj_mastery`/`proj_outcome_mix`.
- assessment endpoints `GET /progress/summary`, `GET /progress/heatmap`, `GET /progress/mastery` (+ rubric trend / outcome mix).
- gateway aggregation handlers `GET /progress` **agg** and `GET /dashboard` **agg**.
- The wired **Progress** screen and the finalized **Dashboard ★** (replacing the S01 skeleton) in `web/`.
- A replay/rebuild runbook (and an ADR if the approach is notable).

## Update status

- As each task lands, set its row in [`../sprints/sprint-09.md`](../sprints/sprint-09.md) to ✅ (🔄 while in progress); set _Overall_ when all tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the **Milestones** table (M5). Add a **Decisions log** line for any notable call.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] Progress shows coverage, revision heatmap, pattern mastery, rubric trend and outcome mix, all from projections.
- [ ] The Dashboard ★ aggregates real daily plan + due reviews + weak area + streak/stats, with reviews prioritised over new work.
- [ ] Projections are idempotent (dedupe on `event_id`) and can be rebuilt by replaying JetStream from the start (verified).
- [ ] Deployed to prod via Flux — **M5 (mock + analytics complete)** — and Progress + Dashboard match the artboards.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
