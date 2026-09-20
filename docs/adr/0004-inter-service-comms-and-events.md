# ADR-0004 — Inter-service comms & events

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0005](0005-data-ownership-and-migrations.md), [0009](0009-deployment-and-gitops.md)

## Context

Services need two interaction styles: **synchronous** request/response (the gateway aggregating a
screen; `review` fetching a problem's metadata) and **asynchronous** reactions to domain facts
(a solved problem must schedule five revisions; a below-Clean outcome must open a mistake). The
five-touch scheduler ([R-SR1](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)) needs both
event-driven scheduling *and* a reliable "due today" sweep that survives the learner being offline.

Constraints: single small node, solo ops, Go everywhere.

## Decision

### Synchronous: HTTP/JSON over ClusterIP

- Internal calls are **HTTP/JSON** on ClusterIP services (`<svc>.xlearn.svc.cluster.local`).
- A small shared client in `internal/httpx` (timeouts, retries with jitter, request-id + JWT propagation).
- **Not gRPC in v1** — the proto toolchain/codegen overhead isn't justified for a handful of
  endpoints and one consumer (the gateway). Revisit for hot internal paths later.

### Asynchronous: NATS JetStream

- **NATS JetStream** is the event backbone — one lightweight binary, runs happily on k3s, durable
  streams + per-consumer acks, first-class Go client. Added to `infra` as a HelmRelease
  (see [0009](0009-deployment-and-gitops.md)).
- Subjects: `xlearn.<context>.<event>` (e.g. `xlearn.practice.problem_solved`). One stream per
  producing context; **durable pull consumers** per subscribing service.
- **Transactional outbox**: a service writes the domain change **and** an `outbox` row in the same
  Postgres transaction; a relay goroutine publishes unacked outbox rows to JetStream and marks them
  sent. This gives at-least-once delivery without 2-phase commit. Consumers are **idempotent**
  (dedupe on `event_id`).
- Events are **facts, versioned, additive-only** (see [`../architecture/events.md`](../architecture/events.md)).

### Revision scheduler: event-driven + periodic sweep (hybrid)

- **Event-driven:** on `xlearn.practice.problem_solved`, `review` computes the next touch's due date
  and persists a `RevisionItem`. On a fail it resets to Day 1 and opens a mistake — all reactions to events.
- **Periodic sweep:** a cron-like worker in `review` runs on an interval (e.g. every 15 min) to
  **materialise "due today"** and enqueue reminders — this catches items whose due date passed while
  the learner was away ([R-SR6](../prd/xlearn-prd.md#63-five-touch-spaced-repetition)). The sweep is
  idempotent and cheap (indexed `due_date <= now AND not surfaced`).

## Consequences

- ✅ One broker to run; durable, replayable event log; consumers can be added without touching producers.
- ✅ Outbox → no lost events even if NATS is briefly down; idempotent consumers → safe redelivery.
- ✅ Hybrid scheduler is correct whether the user is active or offline for a week.
- ⚠️ NATS is **new infra** to stand up (namespace/HelmRelease, small PVC for JetStream file store).
- ⚠️ Every consumer must implement idempotency + an outbox migration — codified in a shared `internal/platform/events` package.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Kafka / Redpanda** | Operationally heavy (ZK/KRaft, JVM-scale memory) for one node; overkill for this event volume. |
| **Redis Streams** | Would add Redis just for this; weaker durability story than JetStream file storage; another stateful component. |
| **Postgres `LISTEN/NOTIFY` + polling only** | No durable backlog/replay; NOTIFY is fire-and-forget (lost if no listener); couples consumers to the producer's DB. |
| **gRPC for sync calls (v1)** | Codegen + proto maintenance cost unjustified for the current surface; HTTP/JSON is debuggable with curl. |
| **Pure cron scheduler (no events)** | Loses immediacy (reviews only appear on the next tick) and couples the scheduler to practice's tables. |
