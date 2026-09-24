# ADR-0014 — NATS JetStream topology & the practice outbox relay

- **Status:** Accepted. **Amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) (v2, Accepted 2026-09-24; see [Amended 2026-09-24 by ADR-0035](#amended-2026-09-24-by-adr-0035)):** streams and consumers move to a single `topology.go` table with byte budgets, poison events go to a dead-letter hook instead of vanishing, and NATS gets nkey auth with per-service ACLs.
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

### Amended 2026-09-24 by ADR-0035

[ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) §1, §2 and §6, accepted at the v2
build-plan sign-off, change the rules below. The text above stays as the v1 record. Deployment
topology, the outbox relay and the delivery semantics are unchanged.

- **One topology table.** Stream and consumer config moves out of `nats.go` into
  `internal/platform/events/topology.go`. It is the single source for every stream (owner, subjects,
  `MaxBytes`/`MaxAge`/`Discard`) and every durable consumer. Producers still self-provision with
  `CreateOrUpdateStream`, but from the table. CI tests:
  - Σ `MaxBytes` ≤ 3.75 GiB (75% of the 5 Gi store; v2 plans 3.375 GiB);
  - every publisher and subscriber has a table entry;
  - a golden rendered NATS `authorization` block;
  - the subject registry: every subject a producer emits is handled or explicitly ignored by each
    subscriber.
- **Streams in v2** add `XLEARN_JUDGE`, `XLEARN_IDENTITY` and `XLEARN_COACH`.
  - identity publishes `XLEARN_IDENTITY` through its outbox (N0, M1). The `LogPublisher` fallback is
    no longer identity's path in production.
  - Replay streams use `Discard=New`, so a full stream stalls the relay loudly while the rows stay in
    the outbox.
- **Poison events leave a row.** On the last delivery of a failing message, the consumer inserts
  `<svc>.event_dead_letter(event_id, subject, durable, err_class, at)`, calls `Term()` and logs at
  ERROR. Before, they were skipped with no record after ≈ 8 h of retries.
- **"No auth in v1" ends at N3.** NATS gets **nkey users in `$G` with fine per-service ACLs**
  rendered from `topology.go`, rolled out **server-first**:
  - N0: client options ship dark;
  - N1: users plus a `legacy` bridge (the one restart);
  - N2: seeds per service;
  - N3: `legacy` denied;
  - N4: the bridge removed.

  Each service may publish only `xlearn.<svc>.>`, manage only its own stream, and use only its own
  durables. Stream delete and purge belong to an offline `ops` identity. Public keys sit in plaintext in
  the `messaging` values, so **`messaging` needs no SOPS decryption**. Seeds are SOPS secrets in
  `apps/secrets`. This supersedes the S12 token/mTLS idea and the Consequences note on a scoped account
  per producer.
- **Standing rule.** A new stream, consumer or subject needs its **infra ACL PR (the re-rendered golden
  block) merged before the consuming service's tag**. A consumer ships one tag before its producer, or
  the producer ships dark behind a T-2 switch
  ([ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §3).

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
