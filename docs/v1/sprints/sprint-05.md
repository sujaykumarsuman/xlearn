# Sprint 05 — Practice engine & the Problem workspace ★

> **Milestone:** M3 — take the guided problem flow   ·   **Design phase:** D3 Guided problem engine
> **Prereqs:** [S02](sprint-02.md), [S04](sprint-04.md)   ·   **Unblocks:** [S06](sprint-06.md), [S08](sprint-08.md), [S11](sprint-11.md)
> **Execute with:** [`../prompts/prompt-s05.md`](../prompts/prompt-s05.md) — one prompt, one session.

## Status

_Overall:_ ✅ Done — **merged & deployed to prod via Flux** (`xlearn-practice`/`xlearn-gateway`:`0.1.15`;
NATS JetStream + the `xlearn_practice` DB role/schema up). Full suite green incl. a real-Postgres
integration run of the gated flow + outbox; Problem routes verified live (session-gated →401). **M3 reached.**

| # | Task | Status |
|---|------|--------|
| 1 | practice service + schema + NATS/outbox bootstrap | ✅ |
| 2 | Stage gating + server-authoritative timers | ✅ |
| 3 | Reveal + penalty + outcome logging | ✅ |
| 4 | Problem screen ★ + BFF | ✅ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M3 milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up `practice` — the first per-user, event-producing service — and wire the hero **Problem ★**
workspace on top of it. This sprint delivers the guided, gated flow that makes xLearn enforce the
learning *method* rather than just serve content: a learner attempts blind under a server-authoritative
timer, unlocks hints then the solution (with a reveal penalty), re-implements from a blank editor, and
logs an outcome. Logging emits the domain facts (`problem_solved`, `attempt_logged`,
`solution_revealed_early`) via a transactional outbox to **NATS JetStream** — the async backbone every
D4/D5 consumer (review, assessment) is built to react to. It carries **M3**.

## Scope

**In**
- New `practice` service (`cmd/practice`, `internal/practice`) owning schema `practice`
  (`user_problem_state`, `attempt`, `stage_event`, `timer`, `outcome`, `outbox`) via `goose` + `sqlc`/`pgx`.
- Stage-gating engine: `attempt`(15m) -> `hint`(10m) -> `solution` -> `re-implement` -> `log`; locked
  stage content is **never delivered** to the client (server enforced, per R-PF1).
- Server-authoritative timers (`timer` rows with `deadline_at`); refresh/return resumes the same
  countdown (R-PF3). Client HUD is a mirror only.
- Reveal + penalty rule (R-PF2): revealing the Solution before the attempt timer elapses returns a
  penalty ack and emits `xlearn.practice.solution_revealed_early`.
- Outcome logging (R-OL1/R-OL3): Clean/Rough/Assisted/Miss updates `user_problem_state` and emits
  `xlearn.practice.problem_solved` (+ `attempt_logged`) via the outbox in one tx; below-Clean is flagged.
- Shared `internal/platform/events`: transactional-outbox writer + relay publisher to JetStream
  (stream `XLEARN_PRACTICE`).
- NATS JetStream as **new infra** (`infra/infrastructure/messaging/` + `infra/clusters/vps/messaging.yaml`).
- The **Problem ★** screen (3-pane workspace) wired to a BFF `GET /problems/{id}` aggregate plus the
  attempt/reveal/outcome endpoints; the Week five-touch dots wired to real practice state.

**Out (later sprints)**
- Scheduling the five-touch reviews and auto-scoring re-solves — that is `review` in [S06](sprint-06.md)
  (practice only *emits* the signals here).
- Opening the mistake-journal entry UI and the 20-min re-solve timer — [S07](sprint-07.md)/[S06](sprint-06.md)
  (practice only flags below-Clean in the event so review opens a mistake).
- Code execution / judging — **not in v1** (re-implementation is self/auto-assessed, [PRD Q1](../../prd/xlearn-prd.md#10-open-questions)).

## Tasks

### 1 · practice service + schema + NATS/outbox bootstrap

Create `cmd/practice` and `internal/practice` following the new-service checklist in
[`../build-plan.md`](../build-plan.md). Own schema `practice` exactly as in
[data-model.md](../../architecture/data-model.md#schema-practice): `user_problem_state`
(`status` locked/available/attempting/solved, `first_solved_at`, `current_touch`, `last_outcome`,
`unique(account_id,problem_id)`), `attempt`, `stage_event`, `timer`, `outcome`, `outbox`. Migrations are
plain-SQL `goose` under `internal/practice/store/migrations/`, `//go:embed`-ed and applied on startup
inside a Postgres advisory lock; queries are `sqlc`/`pgx` (per [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).
Add `/healthz`+`/readyz`, `slog` JSON logs, and a per-service DB role `xlearn_practice` scoped to the
`practice` schema via the infra 4-step pattern (`infra/infrastructure/database/README.md`).

Stand up NATS JetStream as new infra: `infra/infrastructure/messaging/` (HelmRelease + Helm repo +
namespace, small Longhorn PVC for the JetStream file store) and register it with
`infra/clusters/vps/messaging.yaml` Kustomization — mirroring how existing infra Kustomizations are
wired ([ADR-0004](../../adr/0004-inter-service-comms-and-events.md), [ADR-0009](../../adr/0009-deployment-and-gitops.md)).
Implement the shared `internal/platform/events`: an outbox **writer** (append an `outbox` row inside the
caller's tx) and a relay **publisher** goroutine that reads unsent rows and publishes to JetStream
stream `XLEARN_PRACTICE` (subjects `xlearn.practice.*`), marking `sent_at` on ack — at-least-once,
retried with backoff. Deploy: copy `infra/apps/airlift.yaml` as the template for
`infra/apps/xlearn-practice.yaml` (HelmRelease on `charts/project`, ClusterIP only, `route.enabled: false`),
add the `xlearn-practice` ImageRepository/ImagePolicy to `infra/apps/image-automation.yaml`, and the
SOPS DB secret. Add `deploy/practice.Dockerfile` and a CI path filter. NATS must come up **before**
practice depends on it.

### 2 · Stage gating + server-authoritative timers

Implement the gating engine over the stage ladder `attempt` -> `hint` -> `solution` -> `re-implement` ->
`log` ([PRD 6.1](../../prd/xlearn-prd.md#61-guided-problem-flow-gated-stages)). The server is the source of
truth for which stages are unlocked; a locked stage's content is **not delivered** to the client (R-PF1)
— gating is enforced in practice/gateway, never merely hidden in the UI. `POST /problems/{id}/attempt/start`
creates/resumes the `attempt`, writes a `stage_event`, moves `user_problem_state` to `attempting`, and
starts the 15-min `attempt` timer (a `timer` row with `deadline_at = now + 15m`). The `hint` stage carries
a 10-min timer; `solution`/`re-implement`/`log` are untimed. Timers are **server-authoritative** (R-PF3):
the remaining countdown is always computed from `deadline_at`, so refresh/return resumes the same clock and
the client HUD is a mirror. `GET /state?week=` returns per-problem states for a week (backs the Week
five-touch dots and the daily plan) and `GET /state/{problemId}` returns one problem's state + active
timer (backs the Problem HUD). Reference [services.md (practice)](../../architecture/services.md#practice--internal)
and [api.md (Problem workspace)](../../architecture/api.md#problem-workspace).

### 3 · Reveal + penalty + outcome logging

`POST /problems/{id}/reveal` unlocks the next content stage (`hint`, then `solution`), writing a
`stage_event` with `unlocked_from`. Per R-PF2, revealing the **Solution before the attempt timer has
elapsed** returns a **penalty ack** (the learner owes another attempt in 3 days), sets
`outcome.revealed_early`/the attempt's early-reveal flag, and emits `xlearn.practice.solution_revealed_early`
(`data: { problem_id }`) — `review` consumes this to schedule the owed re-attempt (the artboard shows it
as "+1 re-attempt queued for Day 3"). `POST /problems/{id}/outcome` logs one of Clean/Rough/Assisted/Miss:
it writes the `outcome` row, sets `user_problem_state.status = solved` (+ `first_solved_at`,
`last_outcome`), and, in the **same transaction**, appends outbox rows for
`xlearn.practice.problem_solved` (`data: { problem_id, outcome, first_solve }`) and
`xlearn.practice.attempt_logged` (`data: { problem_id, stage_reached, duration_s }`). Any outcome **below
Clean** is flagged in the `problem_solved` payload so `review` opens a mistake and shortens/resets the
interval (R-OL2/R-OL3, [events.md flows 1-2](../../architecture/events.md#flow-1--attempt-logged--revision-scheduled-the-core-loop)).
Emitting only through the outbox keeps producer and broker decoupled; consumers dedupe on `event_id`.

### 4 · Problem screen ★ + BFF

Wire `design-system/screens/Problem.dc.html` into the real SPA, reusing `design-system/theme.css`
tokens/components verbatim. The 3-pane workspace is: **left** reading pane (statement + only the
unlocked-stage sections), **center** editor (with a **blank** re-implement editor at stage 3 — no
prefill, R-PF4), **right** method rail (stage stepper, reveal buttons, outcome picker), plus the top
timer HUD (stage tabs + countdown ring driven by the server timer). The reveal buttons show the penalty
note **before the click** ("Revealing the full solution now means you owe #16 another attempt in 3 days").
Difficulty chips use the tokens: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err` (3Sum is Medium ->
amber). Add the BFF `GET /problems/{id}` **agg** endpoint (gateway) that composes curriculum content
**limited to unlocked stages** (`curriculum` `GET /problems/{id}` with stage scope) + practice state +
active timer, and proxy `POST /problems/{id}/attempt/start|reveal|outcome` to practice (gateway mints the
RS256 internal JWT, [ADR-0006](../../adr/0006-authn-authz.md)). Wire the five-touch dots on the **Week**
screen (the `GET /paths/{slug}/weeks/{n}` agg from [S04](sprint-04.md)) to real practice state. Hero path:
problem **#16 3Sum**, Week 2, Two pointers.

## Acceptance criteria

- [ ] A user takes the full gated flow on the hero problem (#16 3Sum): `attempt`(15m) -> `hint`(10m) ->
      `solution` -> `re-implement` -> log outcome; locked content is not delivered before its stage.
- [ ] The attempt and hint timers are server-authoritative and resume the same countdown across refresh.
- [ ] Revealing the solution early shows the penalty and creates the owed-attempt signal
      (`xlearn.practice.solution_revealed_early` emitted).
- [ ] Logging an outcome updates `user_problem_state` and emits `problem_solved` + `attempt_logged` via
      the outbox in one tx; below-Clean is flagged in the event.
- [ ] NATS JetStream is deployed and healthy; `practice` is deployed to prod via Flux; the Problem ★
      screen matches the artboard. **M3 met.**

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- **Timer drift:** the HUD must recompute from the server `deadline_at` (poll or SSE) and never run a
  naive standalone local timer — the server is authoritative.
- **Locked-stage leakage:** unlocked-stage scoping must be enforced server-side (practice/gateway); the
  client must not receive hint/solution content before its stage unlocks, not just hide it.
- **Outbox correctness:** write the domain row **and** the outbox row in ONE transaction; the relay is
  at-least-once, so consumers dedupe on `event_id`. A crash between commit and publish must be safe (row
  stays unsent and is re-relayed).
- **NATS is new stateful infra:** the JetStream file store needs a small PVC; validate the messaging
  namespace/HelmRelease comes up healthy **before** practice starts depending on it. NATS placement
  (own `messaging` namespace vs `xlearn`) is an open assumption in [`../status.md`](../status.md) — record
  the resolution as an ADR / Decisions-log line.
