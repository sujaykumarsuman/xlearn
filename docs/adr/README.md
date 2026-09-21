# Architecture Decision Records

Numbered, [MADR](https://adr.github.io/madr/)-style, **append-only**. An ADR is never edited to
reverse a decision — instead add a new ADR that **supersedes** it and update the status line of the
old one. ADRs are **global** (they span all versions of xLearn); per-version build execution lives
under [`../v1/`](../v1/).

## Conventions

- Filename: `NNNN-kebab-title.md` (zero-padded, monotonic).
- Each ADR has: **Status · Context · Decision · Consequences · Alternatives considered**.
- Status is one of: `Proposed` · `Accepted` · `Superseded by ADR-XXXX` · `Deprecated`.
- Keep them decisive and skimmable. Record trade-offs, not essays.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted |
| [0002](0002-monorepo-vs-multi-repo.md) | Monorepo vs multi-repo | Accepted |
| [0003](0003-service-decomposition.md) | Service decomposition | Accepted |
| [0004](0004-inter-service-comms-and-events.md) | Inter-service comms & events (NATS JetStream) | Accepted |
| [0005](0005-data-ownership-and-migrations.md) | Data ownership & migrations | Accepted |
| [0006](0006-authn-authz.md) | AuthN/AuthZ (OAuth + session + internal JWT) | Accepted |
| [0007](0007-ai-coach-byo-key-and-secrets.md) | AI coach BYO-key & secret handling | Accepted |
| [0008](0008-frontend-stack.md) | Frontend stack | Accepted |
| [0009](0009-deployment-and-gitops.md) | Deployment & GitOps (Flux/Helm on k3s) | Accepted |
| [0010](0010-spa-base-path-and-gateway-serving.md) | SPA base path & gateway serving model | Accepted |
| [0011](0011-jwks-publication-and-shared-db-schema-ownership.md) | JWKS publication & shared-DB schema ownership | Accepted |
| [0012](0012-curriculum-content-model-and-seeding.md) | Curriculum content model & seeding | Accepted |
| [0013](0013-bff-week-aggregation-userstate-contract.md) | BFF week-aggregation `userState` contract | Accepted |
| [0014](0014-nats-jetstream-topology-and-outbox-relay.md) | NATS JetStream topology & the practice outbox relay | Accepted |
| [0015](0015-five-touch-scheduler-model.md) | Five-touch scheduler model & the first durable consumer | Accepted |
| [0016](0016-mistake-journal-and-worker-service-auth.md) | Mistake journal, weekly weak-area & the worker service-auth path | Accepted |
| [0017](0017-mock-model-and-projection-consumer-scaffold.md) | Mock-interview model & the S09 projection-consumer scaffold | Accepted |
| [0018](0018-progress-projection-grain-and-rebuild.md) | Progress projections: event grain, gateway roll-up & drop-and-replay rebuild | Accepted |
| [0019](0019-account-settings-onboarding-and-reminder-gating.md) | Account settings, onboarding completion & reminder gating | Accepted |
| [0020](0020-coach-service-realization-and-behaviour-gate.md) | Coach service realization & the server-authoritative behaviour gate | Accepted |
| [0021](0021-release-tagging-and-api-versioning.md) | Release tagging, API v1 & the 1.0 hardening cut | Accepted |

## Adding an ADR

1. Copy the structure of [0001](0001-record-architecture-decisions.md).
2. Take the next number; add a row above.
3. Open as `Proposed`; move to `Accepted` when the decision is made.
4. Superseding: add the new ADR, set the old one's status to `Superseded by ADR-NNNN`, link both ways.
