# ADR-0003 — Service decomposition

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0002](0002-monorepo-vs-multi-repo.md), [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md)

## Context

The domain model (see the [PRD](../prd/xlearn-prd.md#7-domain-model-product-view)) has clear seams:
content, per-user practice state, a revision scheduler, a mistake journal, mock sessions, analytics,
an AI coach, identity, and the web edge. The repo mandates a service-oriented design ("don't collapse
back to a monolith"). But the runtime is **one small VPS node** and the team is **one person** — so
the count and granularity of *deployables* must stay honest.

The tension: honour clean service boundaries **without** paying the operational cost of a dozen
always-on processes on a single node.

## Decision

Define **eight bounded contexts + the web app** as the target architecture, and deploy them as
**separate services** (own process, own image, own schema, own API). Bring them online
**incrementally** across v1 sprints; where a context is tiny, it may run as a *worker within its
sibling's deployment* until it earns its own (noted below) — but the **code boundary and event
contract exist from day one**.

| # | Service (`cmd/…`) | Responsibility | Owns (schema) | Emits / consumes | v1 deploy |
|---|-------------------|----------------|---------------|------------------|-----------|
| 1 | **gateway** (BFF) | Web edge: serves the SPA + the single external API; session auth; screen aggregation; coach proxy pass-through. | *(none — read-through)* | calls all services (sync) | **Sprint 01** — exposed at `/xlearn` |
| 2 | **identity** | OAuth (GitHub/Google), accounts, sessions, JWT issuance, onboarding state. | `identity` | emits `account.created` | **Sprint 02** |
| 3 | **curriculum** | Paths, phases, weeks, problems + content sections. Read-heavy, seeded. | `curriculum` | — (read API) | **Sprint 03** |
| 4 | **practice** | UserProblemState, stage gating, server-authoritative timers, outcome logging. | `practice` | emits `attempt.logged`, `problem.solved`, `solution.revealed_early` | **Sprint 05** |
| 5 | **review** | Five-touch scheduler **and** mistake journal (two aggregates, one service). | `review` | consumes `problem.solved`, `attempt.logged`; emits `revision.scheduled`, `revision.due`, `mistake.opened/closed` | **Sprint 06** |
| 6 | **assessment** | Mock sessions + rubric scoring, **and** progress/analytics read-models (projections). | `assessment` | consumes practice/review events; emits `mock.completed` | **Sprint 08** |
| 7 | **coach** | AI-coach gateway: BYO-key storage (encrypted), provider fan-out, page-context prompts. | `coach` | consumes context via API; no domain events | **Sprint 09** |
| 8 | **notifications** | Reminders (study budget, due reviews). Runs as a **worker inside `review`** in v1; splits out when a second channel (email/push) lands. | *(uses `review` read-models)* | consumes `revision.due` | **folded into `review`** (Sprint 06) |
| — | **web** | React SPA (all 12 screens). Built to static assets, embedded in / served by **gateway**. | — | — | **Sprint 01+** |

**Consolidation rationale.** `review` groups the scheduler and the journal because they share the
outcome-event stream and always change together (a fail schedules *and* opens a mistake).
`assessment` groups mock + progress because progress is largely a **read projection** over mock and
practice events — a separate always-on analytics process is not worth a node slot in v1. Both can be
split later without changing their public contracts.

### Boundary rules

- A service owns exactly one Postgres **schema**; **no service reads another's tables** ([0005](0005-data-ownership-and-migrations.md)).
- Cross-service reads go through the owner's **sync API**; cross-service reactions go through **events** ([0004](0004-inter-service-comms-and-events.md)).
- Only **gateway** is internet-facing (has a Traefik route). All others are ClusterIP-only.

## Consequences

- ✅ Clean seams that match the domain and the screens; each service is independently testable/deployable.
- ✅ Incremental delivery: the app is demoable after Sprint 01–03 (browse curriculum) long before practice/mocks land.
- ✅ Node-frugal: ~6 always-on processes in v1, not 10.
- ⚠️ Grouping (`review`, `assessment`) is a judgement call; if either aggregate grows, splitting is an explicit later ADR.
- ⚠️ Event contracts must be defined early even for consolidated services, or the later split gets expensive.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Full 10-service split** (identity, curriculum, practice, revision, mistakes, mock, progress, coach, notifications, gateway) | Faithful to the enumerated list, but ~10 always-on processes on one node for a solo dev — over-provisioned; scheduler+journal and mock+progress churn together anyway. |
| **Modular monolith** (one binary, packages as "services") | Lowest ops, but contradicts the explicit distributed mandate and loses independent deploy/scale. |
| **Split by screen** | Screens cut across contexts (Dashboard reads five services); would produce chatty, incohesive services. |
