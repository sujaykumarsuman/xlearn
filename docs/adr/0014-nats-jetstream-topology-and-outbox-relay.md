# ADR-0014 — NATS JetStream topology & the practice outbox relay

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0009](0009-deployment-and-gitops.md), [events.md](../architecture/events.md)

## Context

[ADR-0004](0004-inter-service-comms-and-events.md) chose **NATS JetStream** as the event
backbone and the **transactional outbox** as the producer pattern, but left the concrete
topology open ("NATS is new infra to stand up"). S05 stands `practice` up as the **first
event producer**, so the broker deployment, the stream/subject layout, and the relay's
delivery semantics must be pinned now — every D4/D5 consumer (review, assessment) is built to
react to what practice emits here. The [status.md](../v1/status.md) open assumption *"NATS
placement: its own `messaging` namespace vs inside `xlearn`"* is resolved here too.

## Decision

### Deployment topology (infra)

- NATS runs as a **HelmRelease in its own `messaging` namespace** (`infra/infrastructure/messaging/`:
  namespace + `HelmRepository` + `HelmRelease`), reconciled by a dedicated Flux Kustomization
  `clusters/vps/messaging.yaml`. It is a **cluster-infrastructure concern**, not an app, so it lives
  beside storage/databases rather than under `apps/` — and services reach it cross-namespace by FQDN
  `nats://nats.messaging.svc.cluster.local:4222`.
- **Single, non-clustered server** (`config.cluster.enabled: false`) on the one node, with
  **JetStream file storage on a small Longhorn PVC** (5Gi) so streams survive a pod restart. The
  Kustomization therefore `dependsOn: infra-storage` (the `longhorn` StorageClass), mirroring
  `databases`. It sits **off the apps critical path**: a broker hiccup never stalls the running apps.
- **No auth in v1.** NATS is ClusterIP-only and only `practice` connects; a token/mTLS is deferred to
  the S12 hardening pass (tracked alongside the cluster-wide `sslmode=prefer` → verified-TLS item).
  The in-cluster `nats-box` CLI pod is disabled to keep the node lean.

### Stream & subject layout (events.md)

- **One JetStream stream per producing context.** `practice` owns **`XLEARN_PRACTICE`** capturing
  subjects **`xlearn.practice.*`** (`file` storage). The producer provisions the stream on startup
  (`CreateOrUpdateStream`, idempotent), so no manual broker setup is needed and a fresh cluster
  self-heals. Review/assessment add `XLEARN_REVIEW` / `XLEARN_ASSESSMENT` the same way.
- Events carry the [events.md](../architecture/events.md) envelope
  (`event_id`, `subject`, `occurred_at`, `version`, `account_id`, `data`).

### Outbox relay & delivery semantics (`internal/platform/events`)

- Producers write the **domain row + an `outbox` row in one Postgres transaction** (never publish
  inline). The shared **`Relay`** goroutine polls unsent rows (oldest first), publishes each, and
  stamps `sent_at`; a crash between publish and stamp simply re-publishes — **at-least-once**.
- The JetStream **`NatsPublisher`** publishes with the **`event_id` as the NATS `Msg-Id`**, so a
  re-published row is **deduplicated at the broker** within the stream's duplicate window (5 min).
  Downstream consumers still dedupe on `event_id` (an inbox/offset) for effectively-once end to end.
- The NATS connection is **resilient** (`RetryOnFailedConnect`, infinite reconnect): practice starts
  and serves even if NATS is briefly unreachable — the outbox buffers and the relay retries — so the
  "NATS up **before** practice depends on it" requirement holds without a hard start-order coupling.
- A **`LogPublisher`** fallback (no `NATS_URL`) logs-and-drains for local dev and for producers whose
  stream isn't provisioned yet (identity's reserved `account_created`), so the outbox never grows
  unbounded off-cluster.

### The three practice events

`xlearn.practice.problem_solved` (`data: problem_id, outcome, first_solve, below_clean`),
`xlearn.practice.attempt_logged` (`data: problem_id, stage_reached, duration_s`), and
`xlearn.practice.solution_revealed_early` (`data: problem_id`) — all on `XLEARN_PRACTICE`. The
additive `below_clean` flag lets `review` open a mistake without re-deriving it (events are
facts, additive-only).

## Consequences

- ✅ One broker, one file-backed stream, self-provisioned by the producer; consumers (S06/S08) attach
  durable pull subscriptions without touching practice.
- ✅ Outbox + broker dedupe + consumer dedupe = no lost or double-applied events across a crash or a
  brief NATS outage.
- ✅ `messaging` is isolated and off the apps path; a JetStream problem degrades to "events queue in
  the outbox", not "the app is down".
- ⚠️ Single non-clustered NATS on one node is **not HA** (like the DB): a node loss loses the broker
  until reschedule; the PVC preserves the streams. Clustered NATS is a multi-node, later concern.
- ⚠️ No broker auth in v1 — acceptable inside a single-tenant ClusterIP cluster; hardened in S12.
- ⚠️ The producer needs `CreateOrUpdateStream` rights on JetStream; fine with no auth, revisit when
  auth lands (a scoped account per producer).

## Alternatives considered

| Option | Why not |
|--------|---------|
| **NATS inside the `xlearn` namespace** | Couples broker lifecycle to the app namespace; a dedicated `messaging` namespace matches the storage/databases infra pattern and keeps RBAC/quotas separable. |
| **Provision streams via a separate manifest / job** | The producer creating its own stream (idempotent) is self-healing and needs no extra ordering; one less thing to drift from code. |
| **Publish inline from the handler (no outbox)** | Loses atomicity — a crash after commit but before publish drops the event; the outbox is the whole point of ADR-0004. |
| **No broker-side dedupe (rely only on consumers)** | Consumers must dedupe anyway, but the `Msg-Id` window cheaply collapses the common re-publish case and shrinks consumer inbox churn. |
| **Clustered/HA NATS now** | Overkill for one node and this event volume; the file-store PVC covers restart durability. Revisit with more nodes. |
