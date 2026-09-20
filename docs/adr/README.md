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

## Adding an ADR

1. Copy the structure of [0001](0001-record-architecture-decisions.md).
2. Take the next number; add a row above.
3. Open as `Proposed`; move to `Accepted` when the decision is made.
4. Superseding: add the new ADR, set the old one's status to `Superseded by ADR-NNNN`, link both ways.
