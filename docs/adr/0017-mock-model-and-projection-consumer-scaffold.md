# ADR-0017 — Mock-interview model & the S09 progress-projection consumer scaffold

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0014](0014-nats-jetstream-topology-and-outbox-relay.md), [0015](0015-five-touch-scheduler-model.md), [events.md](../architecture/events.md), [prd §6.5](../prd/xlearn-prd.md#65-mock-interview)

## Context

S08 stands up the **assessment** service ([ADR-0003](0003-service-decomposition.md) groups
mock + progress here) and ships the timed **Mock interview** (PRD 6.5). Two things needed
pinning:

1. **The mock session model + server-authoritative timer/phase state.** The Mock is a
   45-minute session whose **phase rail** (0-5 clarify · 5-10 brute · 10-18 observation→plan ·
   18-33 code · 33-40 trace+edges · 40-45 complexity, R-MK1) and countdown must be
   server-driven (never a client clock, like the practice timers in
   [S05](../v1/sprints/sprint-05.md)/R-PF3), and it is scored on a **7-dim × 1..5 = /35**
   rubric (R-MK2) charted vs readiness targets (R-MK3).
2. **The S09 progress-projection seam.** assessment will own read-model **projections**
   (`proj_coverage`/`proj_heatmap`/`proj_mastery`/`proj_outcome_mix`) built from
   `practice.*` / `review.*` events ([S09](../v1/sprints/sprint-09.md)). S08 must wire the
   durable consumers + dedupe seam **now** so S09 slots in without rework, but the
   projection tables and handler bodies belong to S09.

## Decision

### Mock session model

- `mock_session` is **one row per mock** (`status ∈ {live, scored}`, `started_at`,
  `deadline_at = started_at + 45m`, `date`, `total_35`, `set_id`/`problem_id`/`difficulty`
  as the soft setup selection). `rubric_score` is **one row per dimension** (7-value enum,
  `score 1..5`), `UNIQUE (mock_session_id, dimension)`.
- **Server-authoritative timer + phase.** The rail state is computed on every read from
  `now - started_at`: elapsed clamps to `[0, 45:00]`, the current phase is the first whose
  end-minute boundary elapsed has not reached (the last phase once the window is exhausted),
  `overtime=true` past the deadline. A refresh/return resumes the same countdown because the
  anchor is the server-set `started_at`. The **client HUD only mirrors** this (it interpolates
  the seconds between polls and derives the phase from the same boundaries the server returns).
- **`total_35` is summed server-side** from the seven validated 1..5 scores — a client total is
  never trusted (R-MK2). Scoring transitions `live → scored` **exactly once** under a
  `SELECT … FOR UPDATE` row lock, so a double-submit / replay is an idempotent no-op that
  returns the stored result (never a duplicate `rubric_score` or a re-emitted `mock_completed`
  with a fresh `event_id`). Two integrity CHECKs back this: `score BETWEEN 1 AND 5`,
  `total_35 BETWEEN 7 AND 35`, and `(status='scored') = (total_35 IS NOT NULL)`.
- Scoring writes the `xlearn.assessment.mock_completed` fact (`mock_id`, `total_35`, `rubric`)
  to the **transactional outbox** in the same tx; the relay publishes it to
  `XLEARN_ASSESSMENT` (events.md / [ADR-0014](0014-nats-jetstream-topology-and-outbox-relay.md)).
  No consumer reacts this sprint (reserved for S09 coach nudges / trend snapshots).
- Readiness **targets are the PRD values** (`≥24 W13`, `≥28 W15`, `≥30 pre`), served from the
  API — not hard-coded UI copy.

### Projection-consumer scaffold (S09-ready, no rework)

- assessment subscribes **two durable pull consumers** — durable `assessment` on
  `XLEARN_PRACTICE` (`xlearn.practice.*`) and on `XLEARN_REVIEW` (`xlearn.review.*`). Durable
  names are per-stream, so sharing the name across streams is fine.
- Both use **`WithDeliverNew()`**. The streams already hold prior sprints' history (S05
  practice, S06/S07 review); because the projection handler bodies are **no-op stubs** this
  sprint, replaying that history has no value and would only flood the inbox — the memory-
  documented lesson from S07's notifications consumer (a new durable on a long-lived stream
  must not replay history). A restart still resumes from the committed offset (offline catch-up).
- The handler decodes the envelope, drops malformed / missing-id messages with an **ack** (they
  can never succeed on redelivery), and calls `store.RecordProjectionEvent`, which **claims the
  `inbox`** (`ON CONFLICT DO NOTHING` → effectively-once) and no-ops the projection body inside
  one transaction. A store error returns non-nil → **nak with the platform's escalating backoff**
  (never a hot loop). S09 fills the `proj_*` upserts into that **same transaction** as the inbox
  claim, so `inbox ⇔ projected` stays atomic.
- **Projections are forward-only from S09 by design.** Events consumed as no-ops in the S08→S09
  window are recorded in the inbox; this is acceptable because a progress read-model is
  derivable/rebuildable (events.md replay note) — if S09 wants historical backfill it replays
  each stream from the start via a distinct one-off consumer. The S09 projection tables land as
  an **additive later migration** (`00002`); S08's `00001` deliberately creates only the mock
  aggregate + outbox + inbox.

## Consequences

- The Mock is honest and tamper-resistant: the timer/phase and the /35 total are the server's,
  ownership is enforced per-account, and scoring is idempotent under concurrency.
- S09 adds the projection tables (additive migration) and the projection bodies (into the
  existing dedupe transaction) with no change to the wiring, the streams, or the inbox seam.
- assessment is **ClusterIP-only** (`route.enabled: false`); only the gateway calls it. It has
  no soft-ref HTTP clients this sprint (the gateway does the curriculum problem enrichment for
  the Mock view), so its infra is the minimal internal shape + `NATS_URL`.
- Difficulty is stored on `mock_session` so the setup selection round-trips; it is a small
  addition beyond the data-model's listed columns, not a new table.
