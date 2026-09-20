# ADR-0005 — Data ownership & migrations

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0004](0004-inter-service-comms-and-events.md), [0009](0009-deployment-and-gitops.md)

## Context

The infra platform runs **one shared** CloudNativePG cluster (`projects-pgstore` in the `databases`
namespace), and documents a clean pattern for giving a service its own database + role. xLearn has
several services that each own a slice of the domain. We must decide the isolation granularity
(database-per-service vs schema-per-service vs shared tables), how services access data, and the
migration + query tooling.

## Decision

### Topology: one database, **schema-per-service**

- xLearn uses **one Postgres database `xlearndb`** on the shared cluster, borrowed via the infra
  4-step pattern (role password SOPS secret → CNPG managed role → `Database` CR → app credentials
  secret). See [0009](0009-deployment-and-gitops.md) for the concrete infra files.
- **Each service owns exactly one schema** (`identity`, `curriculum`, `practice`, `review`,
  `assessment`, `coach`). A service's DB role has privileges **only** on its own schema.
- **No service reads or writes another service's schema.** Cross-service data is fetched via the
  owner's API (sync) or derived from its events (async, [0004](0004-inter-service-comms-and-events.md)).
- The `outbox` table lives **inside each owning schema**.

Schema-per-service (not database-per-service) because: one shared cluster, solo ops, and the infra
pattern would otherwise mean N databases × N roles × 2N SOPS secrets to hand-manage. Per-schema roles
still hard-enforce the "no cross-service tables" boundary. **Path to DB-per-service** is a later ADR
if a service needs independent backup/restore, its own cluster, or a non-Postgres store.

### Migrations: `goose`, per service, applied on startup

- **[goose](https://github.com/pressly/goose)** with plain SQL migrations under
  `internal/<service>/store/migrations/`, **embedded** in the binary (`//go:embed`).
- Applied on service **startup** inside a **Postgres advisory lock** (so parallel replicas don't race);
  the service refuses to serve if migration fails. Simple for a solo dev; no separate migration Job to sequence.
- Each service migrates **only its own schema**; `search_path`/schema-qualified DDL keeps them isolated.

### Query layer: `sqlc` + `pgx`

- **[sqlc](https://sqlc.dev)** generates type-safe Go from hand-written SQL; **[pgx](https://github.com/jackc/pgx)** is the driver/pool. No ORM.
- Go-first, explicit SQL, compile-time-checked queries — fits the "Go-first" ethos and keeps the hot
  paths (dashboard aggregation, revision sweep) transparent and tunable.
- `sqlc generate` output is committed; CI verifies it is up to date (`sqlc diff`).

## Consequences

- ✅ Strong logical ownership (per-schema grants) with light ops (one DB, one connection target).
- ✅ Migrations and queries are versioned in-repo, embedded, and reproducible; no runtime migration tool image.
- ✅ `sqlc` catches query/schema drift at build time.
- ⚠️ Cross-service joins are impossible by design — screen aggregation happens in the **gateway** via
  multiple calls (accepted; it's the BFF's job). Read-heavy screens (Progress) rely on `assessment`
  projections rather than live cross-schema joins.
- ⚠️ Startup migrations + advisory lock must be correct for multi-replica rollouts (documented in `internal/store`).

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Database-per-service** | Cleanest isolation, but N databases/roles/secrets to hand-manage on one shared cluster for a solo dev; deferred until a service truly needs it. |
| **Shared tables / one schema, no ownership** | Fastest to write, but silently recreates a monolith's coupling; forbidden by [0003](0003-service-decomposition.md). |
| **ORM (GORM/ent)** | Hides SQL on the exact hot paths we need to tune; `sqlc` keeps SQL explicit and type-safe. |
| **golang-migrate** | Fine alternative; `goose` chosen for simpler embedded Go usage + Go-function migrations if ever needed. |
| **Separate migration Kubernetes Job** | More moving parts to sequence in Flux; startup-with-advisory-lock is simpler at this scale. |
