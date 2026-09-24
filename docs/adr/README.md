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
| [0006](0006-authn-authz.md) | AuthN/AuthZ (OAuth + session + internal JWT) | Accepted (refined by 0033) |
| [0007](0007-ai-coach-byo-key-and-secrets.md) | AI coach BYO-key & secret handling | Accepted (narrowed by 0031; realtime credential + key lifetime amended by 0032) |
| [0008](0008-frontend-stack.md) | Frontend stack | Accepted |
| [0009](0009-deployment-and-gitops.md) | Deployment & GitOps (Flux/Helm on k3s) | Accepted (refined by 0034) |
| [0010](0010-spa-base-path-and-gateway-serving.md) | SPA base path & gateway serving model | Accepted |
| [0011](0011-jwks-publication-and-shared-db-schema-ownership.md) | JWKS publication & shared-DB schema ownership | Accepted |
| [0012](0012-curriculum-content-model-and-seeding.md) | Curriculum content model & seeding | Accepted |
| [0013](0013-bff-week-aggregation-userstate-contract.md) | BFF week-aggregation `userState` contract | Accepted |
| [0014](0014-nats-jetstream-topology-and-outbox-relay.md) | NATS JetStream topology & the practice outbox relay | Accepted (amended by 0035) |
| [0015](0015-five-touch-scheduler-model.md) | Five-touch scheduler model & the first durable consumer | Accepted |
| [0016](0016-mistake-journal-and-worker-service-auth.md) | Mistake journal, weekly weak-area & the worker service-auth path | Accepted (refined by 0033) |
| [0017](0017-mock-model-and-projection-consumer-scaffold.md) | Mock-interview model & the S09 projection-consumer scaffold | Accepted |
| [0018](0018-progress-projection-grain-and-rebuild.md) | Progress projections: event grain, gateway roll-up & drop-and-replay rebuild | Accepted |
| [0019](0019-account-settings-onboarding-and-reminder-gating.md) | Account settings, onboarding completion & reminder gating | Accepted |
| [0020](0020-coach-service-realization-and-behaviour-gate.md) | Coach service realization & the server-authoritative behaviour gate | Accepted |
| [0021](0021-release-tagging-and-api-versioning.md) | Release tagging, API v1 & the 1.0 hardening cut | Accepted (refined by 0034) |
| [0022](0022-path-enrollment-and-dev-login.md) | Per-user path enrollment & the local dev login | Accepted |
| [0023](0023-email-password-auth-and-account-linking.md) | Email/password sign-in + GitHub↔email account linking | Accepted |
| [0024](0024-public-user-dashboards-and-usernames.md) | Public user dashboards + usernames (the first public route) | Accepted (URL shape + reserved-word rules superseded by 0025; public mock stats → count only by 0032; public-read + per-course visibility by 0033) |
| [0025](0025-public-profiles-under-u-prefix.md) | Public profiles move under `/xlearn/u/<username>` | Accepted |
| [0026](0026-per-course-extensibility-model.md) | Per-course extensibility: course manifests, capability registries, one learning-signal waist (v2 · T0) | Proposed |
| [0027](0027-content-evalpack-and-user-data-model.md) | Content, private eval pack & per-user data model (v2 · T1) | Proposed (§3, §6 amended by 0028; PAT alert and stream budget amended by 0035; D6 scope → the v2.x line, D35) |
| [0028](0028-object-storage-and-backups.md) | Object storage & off-node backups for v2.0: none yet, design ready (v2 · T2) | Proposed (§4 opscheck watch amended by 0035) |
| [0029](0029-judge-contract-and-learning-signal.md) | Judge contract, judge archetypes & learning-signal v2 (v2 · T4) | Proposed |
| [0030](0030-runner-technology-and-host-hardening.md) | Runner technology (sandbox) & host/cluster hardening (v2 · T3) | Proposed (spike pending; §5 A5 and host window amended by 0035) |
| [0031](0031-platform-ai-and-two-tier-keys.md) | Platform AI & two-tier keys (v2 · T5) | Proposed (WIF spike + egress gate before M4; push alert dropped by 0035; "before learners" → v3 opening by 0033) |
| [0032](0032-realtime-ai-mock-interviewer.md) | Realtime AI mock interviewer — M6 / v2.1 (v2 · T6) | Proposed (spike S6 approved, pending) |
| [0033](0033-invite-only-admission-and-owner-admin.md) | Invite-only admission, account roles and the owner admin CLI (v2 · T7) | Proposed (stopgap shipped in v1.5.2: #51 auto-link fix, #52 `SIGNUP_MODE` closed; `DEV_AUTH` guard at M1b) |
| [0034](0034-v2-release-labelling-gating-and-rollback.md) | v2 release labelling, feature gating and rollback (v2 · T7) | Proposed (guard shipped: #53, infra#29) |
| [0035](0035-v2-operations-nats-auth-limits-capacity.md) | v2 operations: NATS auth, limits, capacity triggers (no alerting) (v2 · T7) | Proposed (NATS auth before M3) |

## Adding an ADR

1. Copy the structure of [0001](0001-record-architecture-decisions.md).
2. Take the next number; add a row above.
3. Open as `Proposed`; move to `Accepted` when the decision is made.
4. Superseding: add the new ADR, set the old one's status to `Superseded by ADR-NNNN`, link both ways.
