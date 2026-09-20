# Prompt — Sprint 05 · Practice engine & the Problem workspace ★

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-05.md`](../sprints/sprint-05.md)   ·   **Milestone:** M3   ·   **Prereqs:** [S02](../sprints/sprint-02.md) (identity/JWT), [S04](../sprints/sprint-04.md) (Week/Concept + BFF week agg)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) + [`../../../AGENT.md`](../../../AGENT.md) — repo conventions; start here.
- [`../../adr/0004-inter-service-comms-and-events.md`](../../adr/0004-inter-service-comms-and-events.md) — NATS JetStream, transactional outbox, idempotent consumers; the async contract.
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — schema-per-service, `goose` (embedded, startup + advisory lock), `sqlc`/`pgx`.
- [`../../adr/0009-deployment-and-gitops.md`](../../adr/0009-deployment-and-gitops.md) — infra layout: `charts/project`, GHCR image, image-automation, SOPS, NATS under `infrastructure/messaging/`.
- [`../../adr/0006-authn-authz.md`](../../adr/0006-authn-authz.md) — the gateway-minted RS256 internal JWT that practice validates.
- [`../../architecture/services.md`](../../architecture/services.md) — the **practice** service section (API + owned schema + events).
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — **schema `practice`** table shapes.
- [`../../architecture/events.md`](../../architecture/events.md) — practice events + **flows 1-2** (solve -> schedule; below-clean -> mistake).
- [`../../architecture/api.md`](../../architecture/api.md) — the **Problem workspace** BFF endpoints.
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — **6.1 Guided problem flow** (gated stages, R-PF1..R-PF4) + **6.2 Outcome logging**.
- `design-system/screens/Problem.dc.html` — the Problem ★ artboard (layout/copy/interaction intent; not runnable). Reuse [`../../../design-system/README.md`](../../../design-system/README.md) tokens/components + `design-system/theme.css` verbatim.
- Sibling infra repo: `infra/apps/airlift.yaml` and `infra/apps/landscape.yaml` (HelmRelease templates), `infra/apps/image-automation.yaml`, `infra/infrastructure/database/README.md` (the DB 4-step pattern), `infra/clusters/vps/` (Kustomizations). Inspect before adding deploy config; never `kubectl apply` by hand.

## Context

Browsing works through **M2**: Catalog -> Roadmap -> Week -> Concept render real seeded curriculum, and
the BFF week aggregation ([S04](../sprints/sprint-04.md)) already returns five-touch state *placeholders*.
Identity + the gateway internal JWT exist from [S02](../sprints/sprint-02.md). This is the **hero sprint**:
it adds the guided Problem engine (`practice`), the **first event producer**, and the **NATS JetStream
backbone**. Everything in D4/D5 (review scheduler, mistakes, assessment projections) consumes what
practice emits here, so the schema, events, and outbox must be right.

## Do this (in order)

1. **practice service + schema + NATS/outbox bootstrap** — Scaffold `cmd/practice` + `internal/practice`.
   Create schema `practice` with `goose` SQL migrations (`internal/practice/store/migrations/`, `//go:embed`,
   applied on startup under a Postgres advisory lock): `user_problem_state` (`status`
   locked/available/attempting/solved, `first_solved_at`, `current_touch`, `last_outcome`,
   `unique(account_id,problem_id)`), `attempt`, `stage_event`, `timer`, `outcome`, `outbox` — shapes per
   data-model.md. Generate `sqlc`/`pgx` queries. Add `/healthz`+`/readyz` and `slog` JSON logging. Build the
   shared `internal/platform/events` package: an outbox **writer** (append an `outbox` row in the caller's tx)
   and a relay **publisher** goroutine (publish unsent rows to JetStream stream `XLEARN_PRACTICE`, subjects
   `xlearn.practice.*`, mark `sent_at` on ack, backoff-retry). Stand up NATS as new infra:
   `infra/infrastructure/messaging/` (HelmRelease + Helm repo + namespace + a small Longhorn PVC for the
   JetStream file store) and `infra/clusters/vps/messaging.yaml` (Kustomization). Add
   `deploy/practice.Dockerfile`, a CI path filter, the per-service DB role `xlearn_practice` (4-step pattern,
   scoped to schema `practice`) with its SOPS secret, `infra/apps/xlearn-practice.yaml` (copy
   `infra/apps/airlift.yaml`; ClusterIP, `route.enabled: false`), and the `xlearn-practice` entry in
   `infra/apps/image-automation.yaml`. Bring NATS up healthy **before** practice depends on it.

2. **Stage gating + server-authoritative timers** — Implement the stage ladder `attempt`(15m) -> `hint`(10m)
   -> `solution` -> `re-implement` -> `log`. Enforce gating **server-side**: a locked stage's content is not
   delivered (R-PF1). `POST /problems/{id}/attempt/start` creates/resumes the `attempt`, writes a
   `stage_event`, sets `user_problem_state` to `attempting`, and starts the 15-min attempt `timer`
   (`deadline_at = now + 15m`); the `hint` stage carries a 10-min timer; solution/re-implement/log are
   untimed. Timers are **server-authoritative** (R-PF3): always compute remaining time from `deadline_at` so
   refresh/return resumes the same countdown. Implement `GET /state?week=` (per-problem states for a week ->
   Week dots + daily plan) and `GET /state/{problemId}` (one problem's state + active timer -> Problem HUD).

3. **Reveal + penalty + outcome logging** — `POST /problems/{id}/reveal` unlocks the next content stage
   (hint, then solution) and writes a `stage_event` (`unlocked_from`). Revealing the **solution before the
   attempt timer elapses** returns a **penalty ack** (owes another attempt in 3 days), flags
   `revealed_early`, and emits `xlearn.practice.solution_revealed_early` (`data:{ problem_id }`).
   `POST /problems/{id}/outcome` logs Clean/Rough/Assisted/Miss: write the `outcome` row, set
   `user_problem_state.status=solved` (+ `first_solved_at`, `last_outcome`), and in the **same transaction**
   append outbox rows for `xlearn.practice.problem_solved` (`data:{ problem_id, outcome, first_solve }`) and
   `xlearn.practice.attempt_logged` (`data:{ problem_id, stage_reached, duration_s }`). Flag any **below-Clean**
   outcome in the `problem_solved` payload so `review` opens a mistake (S06/S07 consume it). Emit only through
   the outbox — never publish inline.

4. **Problem screen ★ + BFF** — Wire `design-system/screens/Problem.dc.html` into the SPA (theme.css verbatim,
   no Tailwind): 3-pane workspace (statement | editor | timer HUD), stage stepper + gated reveal buttons that
   show the penalty note **before** the click, a **blank** re-implement editor at stage 3 (no prefill, R-PF4),
   and the outcome picker. Difficulty tokens: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err` (3Sum ->
   Medium/amber). Add gateway BFF `GET /problems/{id}` **agg** (curriculum content **limited to unlocked
   stages** + practice state + active timer) and proxy `POST /problems/{id}/attempt/start|reveal|outcome` to
   practice with the minted internal JWT. Drive the timer HUD from the server timer (poll/SSE, not a local
   clock). Wire the **Week** five-touch dots (the `GET /paths/{slug}/weeks/{n}` agg from S04) to real practice
   state. Validate end-to-end on hero problem **#16 3Sum** (Week 2, Two pointers).

## Constraints

- Reuse `design-system/theme.css` tokens/components **verbatim**; dark theme; no Tailwind. Difficulty
  green/amber/red.
- Schema-per-service: practice touches only schema `practice`; no cross-schema reads/FKs. `goose` embedded
  migrations on startup under an advisory lock; `sqlc`/`pgx`, no ORM.
- Timers **server-authoritative**; locked-stage content enforced server-side (never just hidden in the UI).
- Emit events **only** via the transactional outbox (domain row + outbox row in ONE tx); relay is
  at-least-once; downstream consumers dedupe on `event_id`.
- Internal calls are HTTP/JSON on ClusterIP with the gateway-minted RS256 JWT (JWKS).
- Match infra conventions exactly: copy `infra/apps/airlift.yaml`/`infra/apps/landscape.yaml` as the
  HelmRelease template; GHCR image `ghcr.io/sujaykumarsuman/xlearn-practice`; Flux image-automation with the
  `# {"$imagepolicy": "flux-system:xlearn-practice:tag"}` setter; ImagePolicy semver `>=0.1.0`; SOPS/age
  secrets under `infra/apps/secrets/*.enc.yaml`; only gateway routes (Traefik `/xlearn`); read-only rootfs
  from chart defaults. **Pull-based GitOps — never `kubectl apply` by hand.**
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables

- `cmd/practice`, `internal/practice` (schema `practice`, `goose` migrations, `sqlc`/`pgx` queries,
  gating engine, server timers, reveal/penalty, outcome logging), `internal/platform/events` (outbox writer
  + JetStream relay), `deploy/practice.Dockerfile`, a CI path filter.
- practice API: `GET /state?week=`, `GET /state/{problemId}`, `POST /problems/{id}/attempt/start`,
  `POST /problems/{id}/reveal`, `POST /problems/{id}/outcome`; events `xlearn.practice.attempt_logged`,
  `problem_solved`, `solution_revealed_early` on stream `XLEARN_PRACTICE`.
- gateway: `GET /problems/{id}` **agg** + the attempt/reveal/outcome proxies; Week dots wired to real state.
- Problem ★ screen wired to real data.
- infra: `infra/infrastructure/messaging/` (NATS JetStream + PVC), `infra/clusters/vps/messaging.yaml`,
  `infra/apps/xlearn-practice.yaml`, `xlearn-practice` in `infra/apps/image-automation.yaml`, the
  `xlearn_practice` DB role + SOPS secret.

## Update status

- As each task lands, set its row in [`../sprints/sprint-05.md`](../sprints/sprint-05.md) to ✅ (🔄 while in
  progress); set _Overall_ to ✅ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the
  **Milestones** table (M3). Add a **Decisions log** line for any notable call (e.g. the NATS namespace
  placement).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style) — e.g. the NATS
  JetStream stream/consumer layout or the outbox-relay design if it hardens beyond ADR-0004.

## Done when (acceptance)

- [ ] A user takes the full gated flow on #16 3Sum: `attempt`(15m) -> `hint`(10m) -> `solution` ->
      `re-implement` -> log outcome; locked content is not delivered before its stage.
- [ ] The attempt and hint timers are server-authoritative and resume the same countdown across refresh.
- [ ] Revealing the solution early shows the penalty and emits `xlearn.practice.solution_revealed_early`.
- [ ] Logging an outcome updates `user_problem_state` and emits `problem_solved` + `attempt_logged` via the
      outbox in one tx; below-Clean is flagged in the event.
- [ ] NATS JetStream is deployed and healthy; practice deployed to prod via Flux; Problem ★ matches the
      artboard. **M3 met.**
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
