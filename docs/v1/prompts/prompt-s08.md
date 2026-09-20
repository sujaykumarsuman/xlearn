# Prompt — Sprint 08 · Assessment: Mock interview

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-08.md`](../sprints/sprint-08.md)   ·   **Milestone:** none (mid-phase)   ·   **Prereqs:** [S05](../sprints/sprint-05.md), [S02](../sprints/sprint-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions (dark theme, service boundaries, ship at session end (AGENT.md land-and-sync)).
- [`../../architecture/services.md`](../../architecture/services.md) — the **assessment** service (mock aggregate): owned schema, internal API, emits/consumes.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `assessment`: `mock_session`, `rubric_score`, `inbox` (and the S09 projection tables you must **not** create yet).
- [`../../architecture/events.md`](../../architecture/events.md) — `xlearn.assessment.mock_completed`, the outbox/inbox pattern, `XLEARN_ASSESSMENT` stream, durable pull consumers.
- [`../../architecture/api.md`](../../architecture/api.md) — the BFF surface for `POST /mocks`, `GET /mocks/{id}`, `POST /mocks/{id}/score`, `GET /mocks/trend`.
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — **6.5 Mock interview** (R-MK1 phase rail, R-MK2 7-dim /35 rubric, R-MK3 targets).
- [ADR-0003](../../adr/0003-service-decomposition.md), [ADR-0004](../../adr/0004-inter-service-comms-and-events.md), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md), [ADR-0009](../../adr/0009-deployment-and-gitops.md) — decomposition, events/outbox, migrations (goose + sqlc/pgx), deploy/GitOps.
- `design-system/screens/Mock.dc.html` — the Mock artboard (setup / live phase rail / rubric radar + trend). Layout + copy intent only; **not** runnable. Reuse [`../../../design-system/README.md`](../../../design-system/README.md) tokens/components (`ds-meter`, `ds-seg`, radar `svg`).
- Sibling infra repo for deploy conventions: `infra/apps/airlift.yaml` (HelmRelease template), `infra/apps/image-automation.yaml`, `infra/infrastructure/database/README.md` (4-step DB pattern), `infra/apps/secrets/`.

## Context

Practice exists ([S05](../sprints/sprint-05.md)): the guided Problem flow logs outcomes and emits
`practice.*` events; identity/auth + the gateway ([S02](../sprints/sprint-02.md)) mint the internal
JWT and mount `/xlearn/api`. This sprint stands up the **assessment** service and ships the timed
**Mock** with its 6-phase rail and 7-dimension rubric + trend. Progress projections and the Dashboard
aggregation that read this service come **next** in [S09](../sprints/sprint-09.md) — build the schema
and consumer scaffold so they slot in without rework.

## Do this (in order)

1. **assessment service + schema + consumer bootstrap** — Scaffold `cmd/assessment/` (config, pgx
   pool, goose migrate-on-startup under a Postgres advisory lock, slog JSON logs, `/healthz`+`/readyz`,
   graceful shutdown) and `internal/assessment/` (store / service / http). Add embedded
   (`//go:embed`) goose migrations under `internal/assessment/store/migrations/` for schema
   `assessment` creating **only**: `mock_session` (`id`, `account_id`, `set_id`, `problem_id`, `date`,
   `total_35`, `notes`, plus `status` live/scored, `started_at`, `deadline_at`), `rubric_score` (`id`,
   `mock_session_id`, `dimension` 7-enum, `score` 1..5), `outbox`, `inbox` (`event_id`, `consumed_at`).
   **Do not** create the S09 projection tables — leave room for them as a later migration. Write sqlc
   queries and commit generated output (`sqlc generate`; CI runs `sqlc diff`). Add the outbox relay
   goroutine and a **durable pull-consumer scaffold** on JetStream for `practice.*` and `review.*`
   (idempotent, dedupe on `event_id` via `inbox`) with **no-op handler stubs** for S09; declare the
   `XLEARN_ASSESSMENT` stream for this service's emits. Add `deploy/assessment.Dockerfile` and a CI
   path filter for `cmd/assessment/**` + `internal/assessment/**`. Add infra:
   `infra/apps/xlearn-assessment.yaml` (copy `infra/apps/airlift.yaml`; `route.enabled: false`,
   ClusterIP), an `xlearn-assessment` ImageRepository/ImagePolicy entry in
   `infra/apps/image-automation.yaml` with the `# {"$imagepolicy": "flux-system:xlearn-assessment:tag"}`
   setter and semver range `>=0.1.0`, the `assessment` DB role/schema via the infra 4-step pattern, and
   any SOPS secret under `infra/apps/secrets/*.enc.yaml`.
2. **mock session lifecycle** — Implement `POST /mocks` (accept setup: problem set + difficulty; insert
   `mock_session` with `status=live`, `started_at=now`, `deadline_at=now+45m`; return the live session
   = `setup -> live`) and `GET /mocks/{id}` (return the live session + **server-computed** phase-rail
   state and remaining time). Phases: **0-5 clarify · 5-10 brute force · 10-18 observation->plan · 18-33
   code · 33-40 trace+edges · 40-45 complexity+follow-ups**. Make the 45-min timer and phase index
   **server-authoritative** (client HUD mirrors it; refresh/return resumes the same countdown; elapsed
   clamps at 45:00). Route these through the gateway BFF.
3. **7-dimension rubric + trend** — Implement `POST /mocks/{id}/score`: record one `rubric_score` per
   dimension for **Communication, Problem understanding, Brute force, Optimisation, Code quality, Edge
   cases, Complexity** (each integer 1-5); compute `total_35 = sum` server-side; set
   `status=scored`; and in the **same transaction** write an `outbox` row for
   `xlearn.assessment.mock_completed` (`mock_id`, `total_35`, `rubric`). Implement `GET /mocks/trend`:
   ordered `total_35` history vs the targets **>=24 by W13, >=28 by W15, >=30 pre-interview**. Wire both
   through the gateway.
4. **Mock screen** — Build `/xlearn/dsa/mock` from `Mock.dc.html` with `theme.css` verbatim (no
   Tailwind): **setup** (problem-set + `ds-seg` difficulty picker + "Start 45-minute mock" -> `POST /mocks`),
   **live** (six-segment phase rail with the current phase `--ds-teal` / past `--ds-ok`; server timer HUD
   turning amber >=40:00, red >=43:00; interview-mode editor; current-phase prompt card; "Finish & score"
   polling `GET /mocks/{id}`), and **results** (7-dim radar `svg` + `total_35` headline; trend chart vs
   the W13/W15/pre target lines; per-dim `ds-meter` rows colored >=4 `--ds-ok` / 3 `--ds-warn` / <=2
   `--ds-err`; "New mock" resets).

## Constraints

- Reuse `design-system/theme.css` verbatim (tokens + `ds-*` components; **no Tailwind**); keep the dark
  "landscape console" theme. Difficulty tokens: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`.
- Schema-per-service: assessment touches **only** its own `assessment` schema; goose migrations
  (embedded, applied on startup under an advisory lock) + sqlc/pgx (no ORM). Cross-service references
  are **soft ids** (no FKs across schemas).
- **Server-authoritative** timer + phase state (never trust a client clock); validate every rubric
  score is an integer in [1,5] and compute `total_35` on the server.
- Async via NATS JetStream + **transactional outbox**; consumers **idempotent** (dedupe on `event_id`).
  Consumer handler bodies for projections are stubs this sprint (built in [S09](../sprints/sprint-09.md)).
- assessment is **ClusterIP only** (`route.enabled: false`); only the gateway has a Traefik route.
  Inherit chart pod hardening (non-root, **read-only rootfs**, dropped caps).
- **Pull-based GitOps**: all deploy config goes through the `infra` repo and Flux — never `kubectl apply`
  by hand. Mirror `infra` conventions exactly (copy `airlift.yaml`; GHCR image
  `ghcr.io/sujaykumarsuman/xlearn-assessment`; build-semver auto-deploy from `main`).
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables

- `cmd/assessment/` + `internal/assessment/` (store/service/http), embedded goose migrations for the
  `assessment` schema (`mock_session`, `rubric_score`, `outbox`, `inbox`), committed sqlc output.
- Outbox relay + durable pull-consumer scaffold (idempotent, stub handlers) + `XLEARN_ASSESSMENT` stream.
- Internal + BFF endpoints: `POST /mocks`, `GET /mocks/{id}`, `POST /mocks/{id}/score`, `GET /mocks/trend`,
  plus the `xlearn.assessment.mock_completed` emit.
- The **Mock** screen (`/xlearn/dsa/mock`) across setup / live / results.
- `deploy/assessment.Dockerfile`, CI path filter, and infra additions (`infra/apps/xlearn-assessment.yaml`
  HelmRelease, `infra/apps/image-automation.yaml` entry, `assessment` DB role/schema, SOPS secret).

## Update status

- As each task lands, set its row in [`../sprints/sprint-08.md`](../sprints/sprint-08.md) to ✅ (🔄 while
  in progress); set _Overall_ to ✅ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row. Add a
  **Decisions log** line for any notable call.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] A user runs a 45-min mock with the phase rail advancing through the 6 phases on a **server timer**
      (refresh/return resumes; elapsed clamps at 45:00).
- [ ] Scoring the 7 dimensions (each 1-5) yields a **/35** total; the trend renders vs the W13 (>=24) /
      W15 (>=28) / pre-interview (>=30) targets.
- [ ] `xlearn.assessment.mock_completed` is emitted via the outbox; assessment is deployed to prod as a
      **ClusterIP** service.
- [ ] The **Mock** screen matches the artboard across setup / live / results.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
