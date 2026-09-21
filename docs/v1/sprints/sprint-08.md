# Sprint 08 — Assessment: Mock interview

> **Milestone:** none (mid-phase)   ·   **Design phase:** D5 Mock + analytics
> **Prereqs:** [S05](sprint-05.md), [S02](sprint-02.md)   ·   **Unblocks:** [S09](sprint-09.md)
> **Execute with:** [`../prompts/prompt-s08.md`](../prompts/prompt-s08.md) — one prompt, one session.

## Status

_Overall:_ ✅ Done — merged & deployed to prod via Flux (`0.1.21`)

| # | Task | Status |
|---|------|--------|
| 1 | assessment service + schema + consumer bootstrap | ✅ |
| 2 | mock session lifecycle | ✅ |
| 3 | 7-dimension rubric + trend | ✅ |
| 4 | Mock screen | ✅ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up the **assessment** service and ship the timed **Mock interview**. A user runs a
45-minute, server-timed session whose **phase rail** walks the six interview phases, then scores
themselves on the **7-dimension rubric** (/35) and sees the score charted against the readiness
targets. This is the first half of design phase D5 (the mock loop); the progress **projections** and
the Dashboard aggregation it feeds are built next in [S09](sprint-09.md). It closes goal G4 (honest
interview simulation) from [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md).

## Scope

**In**
- New `assessment` service: `cmd/assessment/` + `internal/assessment/`, following the new-service
  checklist (`/healthz`+`/readyz`, slog JSON, goose migrations, sqlc/pgx, outbox, `deploy/assessment.Dockerfile`, CI path filter).
- Schema `assessment`: `mock_session`, `rubric_score`, `outbox`, and an `inbox` (dedupe) table.
- NATS: durable **pull-consumer scaffold** subscribing to `practice.*` / `review.*` and the
  `XLEARN_ASSESSMENT` stream for the service's own emits (projection handlers are wired in S09).
- Mock lifecycle API: `POST /mocks` (start 45-min timed session, setup -> live), `GET /mocks/{id}`
  (live session + phase-rail state), server-authoritative 45-min timer + phase state machine.
- Rubric API: `POST /mocks/{id}/score` (7 dims x 1-5 -> /35), `GET /mocks/trend` (trend vs targets),
  and the `xlearn.assessment.mock_completed` emit via outbox.
- Gateway routes for `POST /mocks`, `GET /mocks/{id}`, `POST /mocks/{id}/score`, `GET /mocks/trend`.
- The **Mock** screen (`/xlearn/dsa/mock`): setup -> live phase rail -> rubric radar + trend chart.
- Infra: `apps/xlearn-assessment.yaml` HelmRelease + image-automation entry + DB role/schema + SOPS secret.

**Out (later sprints)**
- Progress projections (`proj_coverage`/`proj_heatmap`/`proj_mastery`/`proj_outcome_mix`) and the
  consumer handlers that build them, plus `GET /progress` + the Dashboard `A` aggregation — [S09](sprint-09.md).
- Pairing mocks with a real interviewer AI (live LLM interviewer) — out of v1 scope.

## Tasks

### 1 · assessment service + schema + consumer bootstrap

Create `cmd/assessment/` (main: config, pgx pool, goose migrate-on-startup under a Postgres advisory
lock, slog JSON to stdout, `/healthz`+`/readyz`, graceful shutdown) and `internal/assessment/`
(store, service, http). Add the schema `assessment` goose migrations under
`internal/assessment/store/migrations/` (embedded via `//go:embed`) creating **only** this sprint's
tables — see [`../../architecture/data-model.md`](../../architecture/data-model.md):

- `mock_session` (`id`, `account_id`, `set_id`, `problem_id`, `date`, `total_35`, `notes`) plus
  session-state columns needed by the lifecycle: `status` (`live`/`scored`), `started_at`, `deadline_at`.
- `rubric_score` (`id`, `mock_session_id`, `dimension` (7-enum), `score` 1..5).
- `outbox` (`event_id`, `subject`, `payload_json`, `created_at`, `sent_at`) for the transactional-outbox relay.
- `inbox` (`event_id`, `consumed_at`) for idempotent consume/dedupe.

Design the migrations so the S09 projection tables (`proj_coverage`, `proj_heatmap`, `proj_mastery`,
`proj_outcome_mix`) **slot in as a later migration** without touching these — do NOT create them here.
Write sqlc queries (`sqlc generate`, commit output; CI runs `sqlc diff`). Add the outbox relay
goroutine and a **durable pull-consumer scaffold** on JetStream subscribing to `practice.*` and
`review.*` per [`../../architecture/events.md`](../../architecture/events.md) and
[ADR-0004](../../adr/0004-inter-service-comms-and-events.md): idempotent (dedupe on `event_id` via
`inbox`), but the handler bodies that update projections are **left as no-op stubs for S09**. Declare
the `XLEARN_ASSESSMENT` stream for this service's own emits. Add `deploy/assessment.Dockerfile`, a CI
path filter for `internal/assessment/**` + `cmd/assessment/**`, and the infra additions
(`infra/apps/xlearn-assessment.yaml` HelmRelease copied from `apps/airlift.yaml` with `route.enabled: false`;
an `xlearn-assessment` entry in `infra/apps/image-automation.yaml`; the `assessment` DB role/schema via
the infra 4-step DB pattern; secrets under `infra/apps/secrets/*.enc.yaml` as needed). See
[ADR-0003](../../adr/0003-service-decomposition.md), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md),
[ADR-0009](../../adr/0009-deployment-and-gitops.md).

### 2 · mock session lifecycle

Implement the internal API `POST /mocks`, `GET /mocks/{id}` (see the internal surface in
[`../../architecture/services.md`](../../architecture/services.md) and the BFF surface in
[`../../architecture/api.md`](../../architecture/api.md)):

- `POST /mocks` accepts the setup selection (problem set + difficulty), inserts a `mock_session`
  (`status=live`, `started_at=now`, `deadline_at=now+45m`), and returns the live session. This is the
  `setup -> live` transition.
- `GET /mocks/{id}` returns the live session plus **server-computed phase-rail state**: which of the
  six phases is current given `now - started_at`, and the remaining time. The phases (per
  [PRD 6.5 R-MK1](../../prd/xlearn-prd.md#65-mock-interview)) are: **0-5 clarify · 5-10 brute force ·
  10-18 observation->plan · 18-33 code · 33-40 trace+edges · 40-45 complexity+follow-ups**.
- The 45-min timer and phase index are **server-authoritative** (like the practice attempt/hint
  timers in [S05](sprint-05.md) / [R-PF3](../../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages));
  the client HUD is a mirror, and refresh/return resumes the same countdown. A session past its
  deadline reports elapsed clamped at 45:00.

### 3 · 7-dimension rubric + trend

Implement `POST /mocks/{id}/score` and `GET /mocks/trend`:

- `POST /mocks/{id}/score` records one `rubric_score` per dimension for the seven dims
  (**Communication, Problem understanding, Brute force, Optimisation, Code quality, Edge cases,
  Complexity**), each an integer **1-5**; the server computes `total_35 = sum` (exactly /35, per
  [R-MK2](../../prd/xlearn-prd.md#65-mock-interview)), sets `mock_session.status=scored`, and in the
  **same transaction** writes an `outbox` row for `xlearn.assessment.mock_completed`
  (payload: `mock_id`, `total_35`, `rubric`) per [`../../architecture/events.md`](../../architecture/events.md).
- `GET /mocks/trend` returns the ordered `total_35` history for the account against the readiness
  targets from [R-MK3](../../prd/xlearn-prd.md#65-mock-interview): **>=24 by W13, >=28 by W15, >=30
  pre-interview** (used to draw the three dashed target lines on the trend chart).
- The relay publishes `mock_completed` to the `XLEARN_ASSESSMENT` stream (reserved for S09 coach
  nudges / trend snapshots; no consumer reaction is required this sprint).

### 4 · Mock screen

Build the **Mock** screen at `/xlearn/dsa/mock`, wiring
[`design-system/screens/Mock.dc.html`](../../../design-system/screens/Mock.dc.html) with real API calls and
`theme.css` verbatim (no Tailwind). Three states from the artboard:

- **Setup:** problem-set + difficulty (`ds-seg`) picker, the "45-minute rail" reference, and the
  "Start 45-minute mock" primary button -> `POST /mocks` -> live.
- **Live:** the phase rail (six segments; the current phase highlighted `--ds-teal`, past phases
  `--ds-ok`), the server timer HUD (color shifts amber >=40:00, red >=43:00), the interview-mode
  editor pane, and the current-phase prompt card; "Finish & score" transitions to results. State
  polls `GET /mocks/{id}` so the rail/timer stay server-driven.
- **Results:** the 7-dim **radar** (DS `svg` polygon) + the `total_35` headline, the **trend chart**
  vs the W13/W15/pre-interview target lines, and the per-dimension `ds-meter` rows colored by score
  (>=4 green `--ds-ok`, 3 amber `--ds-warn`, <=2 red `--ds-err`). "New mock" resets to setup.

## Acceptance criteria

- [ ] A user runs a 45-min mock with the phase rail advancing through the 6 phases on a **server timer**
      (refresh/return resumes the same countdown; elapsed clamps at 45:00).
- [ ] Scoring the 7 dimensions (each 1-5) yields a **/35** total, and the trend renders against the
      W13 (>=24) / W15 (>=28) / pre-interview (>=30) targets.
- [ ] `xlearn.assessment.mock_completed` is emitted via the outbox on scoring; assessment is deployed
      to prod as a **ClusterIP** service (no Traefik route).
- [ ] The **Mock** screen matches the artboard across setup / live / results.

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- The phase rail is a **timed state machine** — keep it server-authoritative like the practice timers
  ([S05](sprint-05.md)); never trust a client-reported elapsed time for the current phase.
- The rubric is exactly **7 dims x 1-5 = /35** — validate each score is an integer in [1,5] and reject
  partial submissions; compute `total_35` server-side (never trust a client total). Targets come from
  the PRD, not hard-coded UI copy.
- Keep the **projection tables out of this sprint** (they belong to [S09](sprint-09.md)), but design
  the schema + consumer scaffold so they slot in as an additive migration + handler bodies later.
- Emit-order is best-effort per-subject; the S09 consumers must be idempotent (dedupe on `event_id`) —
  wire the `inbox` now even though handlers are stubs.
