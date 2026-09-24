# ADR-0021 — Release tagging, API v1, and the 1.0 hardening cut

- **Status:** Accepted. **Refined by [ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) (v2, Accepted 2026-09-24; see [Amended 2026-09-24 by ADR-0034](#amended-2026-09-24-by-adr-0034)):** bounded ImagePolicy ranges (`<2.0.0` until GA), a `.release-line` CI guard, `-rc` prereleases and the GA range flip; the API stays `/api/v1`.
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0009](0009-deployment-and-gitops.md) (deploy/GitOps — this refines its versioning section), [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0006](0006-authn-authz.md), [0018](0018-progress-projection-grain-and-rebuild.md)
- **Source of truth for infra conventions:** the sibling repo `github.com/sujaykumarsuman/infra`.

## Context

xLearn is feature-complete (M6). S12 hardens it and cuts **1.0**: it flips the deploy mode from
build-semver auto-deploy to release-semver tags ([ADR-0009](0009-deployment-and-gitops.md) always
planned this switch "at the 1.0 milestone"), introduces API versioning, and lands a set of hardening
decisions (performance, a generated API contract, a real-broker e2e) that are each notable enough to
record. This ADR captures the deliberate calls.

## Decision

### 1. Deploy mode: build-semver → release-semver tags

- The xlearn `deploy.yml` trigger moves from **push to `main`** (`VERSION=0.1.<run>`) to **push a
  release tag `v*`** (`VERSION=${{ github.ref_name }}`), mirroring the platform's `airlift`. The reusable
  `build-push.yml@main` tags the image with the clean semver (`vX.Y.Z` → `X.Y.Z`). All seven services
  rebuild on a tag: a release is a coordinated version bump, and `dorny/paths-filter` has no reliable
  base on a tag push, so per-service gating is dropped.
- In `infra/apps/image-automation.yaml`, every `xlearn-<svc>` `ImagePolicy` range flips `>=0.1.0` →
  `>=1.0.0`. Flux's `ImageUpdateAutomation` (Setters) then bumps each HelmRelease `tag:` marker to the
  new `1.0.0` automatically — the markers are **not** hand-edited.
- **Coordination (the one ordering that avoids a Flux stall):** (a) land the repo changes on `main`
  (nothing deploys — no tag, and the old `>=0.1.0` range still governs); (b) `git tag v1.0.0` → CI builds
  and pushes all seven images as `1.0.0`; (c) **only then** flip the infra ranges to `>=1.0.0` and merge.
  Flipping the range before a `1.0.0` image exists would leave every policy matching nothing (deploy
  stalls at `0.1.27`).

#### Amended 2026-09-24 by ADR-0034

[ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §1, accepted at the v2 build-plan
sign-off, refines §1. It is not superseded, and the text above stays as the v1 record.
[`../git-strategy.md`](../git-strategy.md) holds the operational version.

- **Bounded ranges.** Every fleet `xlearn-*` ImagePolicy is `>=1.0.0 <2.0.0` (infra#29, 2026-09-24)
  until the v2.0 GA widens it to `>=1.0.0 <3.0.0`. A `.release-line` file (`1` until the GA PR sets
  `2`) and a first `deploy.yml` job refuse any stable tag whose major differs, before any image is
  built (xlearn#53). The check reads the file at the tagged commit.
- **"Tag, then flip" becomes a table** (ADR-0034 §1.4). Ordinary releases never touch a range:
  - **a new policy or a raised floor** (this section's 1.0 flip; a new service such as judge): tag
    first, then merge;
  - **a superset widening** (the GA; a runner or evalpack major): run the pre-flip check, merge, then
    tag;
  - **a narrowing**: only as an R-b rollback.
- **`-rc` prereleases.** `vX.Y.Z-rc.N` tags build images that never auto-deploy, because Flux's
  semver ranges skip prereleases.
- **Labels are GA flips.** v2 milestones ship dark as `1.x` minors. `v2.0.0` flips the v2 defaults for
  the owner; `v2.1.0` is the interviewer GA. From `v2.0.0` on, the minor moves only at a GA flip.
- **Every fleet service still rebuilds on a tag.** That is seven today, and judge becomes the 8th
  `deploy.yml` job at M3-1. The runner (`runner-v*`) and the eval pack have their own tag streams.
- **The API stays `/api/v1`** for all of 2.x, additive only (§2 unchanged). DSA alias routes stay for
  at least one release after the SPA stops calling them. `/api/v2` comes only with an external client.

### 2. API versioning: `/xlearn/api/v1` as an alias

The gateway introduces **`/xlearn/api/v1`** (the [api.md](../architecture/api.md) versioning note). It is a
one-line prefix **alias** in `gateway.Handler`: `/api/v1/…` is rewritten to `/api/…` before dispatch, so a
single route table serves both. The SPA calls `/xlearn/api/v1` (`web/src/lib/api.ts`); the unversioned
`/xlearn/api` stays a same-origin compat alias. Chosen over rewriting all ~30 route patterns because the
app is same-origin (the SPA ships with its gateway — there are no external legacy clients to break), and
the alias keeps the change to two touch points instead of the whole mux. JWKS and the k8s probes are
outside `/api` and stay unversioned.

### 3. Generated API contract + CI drift check

`docs/architecture/openapi.yaml` (OpenAPI 3.1) is the machine-readable contract. It is **authored** (not
tool-generated) because the gateway composes several response bodies from `map[string]json.RawMessage`
aggregations that no struct-reflection generator can introspect. To keep it honest, the gateway route
table is a first-class value (`apiRoutes` in `bff.go`) and a Go test (`openapi_drift_test.go`) asserts a
bijection between it and the spec's paths — a route added/removed without a spec update fails CI. This
runs in the default `go test` lane (no network). Adds one dep: `gopkg.in/yaml.v3`.

### 4. Core-loop e2e over real NATS JetStream

`internal/e2e` boots the real practice/review/assessment handlers in-process against a real Postgres and a
real JetStream, wired exactly as `cmd/*` (outbox → relay → JetStream → durable idempotent consumers), and
drives events.md Flows 1-4. NATS runs **in-process** (an embedded `nats-server/v2`, JetStream file store in
a temp dir), so the test is self-contained — it needs only a Postgres — and CI provides just a Postgres
service container. It is behind the `e2e` build tag + gated on `XLEARN_TEST_DATABASE_URL`, so the default
lane never runs it. Adds one test-scoped dep: `github.com/nats-io/nats-server/v2`.

### 5. Performance: N+1 removal + per-account BFF cache; index audit

- The p95 lever was a **gateway N+1**: the Revision-due and mistake-journal enrichers fetched curriculum
  one problem-id at a time. Fixed with a bulk `curriculum GET /problems?ids=` (`GetProblemsByIDs`) called
  once per fan-out.
- The BFF caches the **Dashboard** and **Week** aggregations **per account**, short TTL (env
  `AGG_CACHE_TTL`, default 15s), invalidated on that account's own mutating writes (outcome / revision
  score / mock score / mistake edit). Keyed by account id so one user never sees another's data; the short
  TTL bounds staleness from async events the gateway doesn't observe (the sweep, the projections).
- **Index audit:** the example indexes the sprint named are already covered — `user_problem_state` has no
  `week` column (the param is log-only); `revision_item_due_idx (account_id, due_date, surfaced_at)` already
  leads with `(account_id, due_date)`; each `assessment.proj_*` PK **is** its read key. The one genuine gap
  was the cross-account due-sweep (no `account_id` predicate), fixed by a partial index
  `revision_item_sweep_idx (due_date) WHERE surfaced_at IS NULL AND status = 'pending'` (review `00003`).
- **Fit-on-one-node:** NATS (req 25m/64Mi, limit 256Mi; JetStream on a 5Gi PVC) and CNPG (req 250m/512Mi,
  limit 1Gi per instance × 3) resource requests/limits are **set and fit** the single VPS node — verified,
  no change needed. CNPG `postgresql.parameters` tuning + a PgBouncer pooler are recorded as a later,
  cross-project follow-up (they restart the shared cluster).

## Consequences

- ✅ Releases are deliberate: a tag ships prod, with generated notes and a clean semver history.
- ✅ The API surface is versioned without breaking the same-origin SPA, and the contract can't silently
  drift from the handlers.
- ✅ The async core loop has a real, fast, repeatable regression test — a broken outbox/consumer path fails
  CI instead of shipping.
- ⚠️ Two new deps (`yaml.v3`, test-only `nats-server/v2`) enter `go.mod`; both are justified and neither is
  compiled into a production binary's request path.
- ⚠️ The tag↔range coordination is a manual two-repo discipline (called out above and in the release
  checklist); getting the order wrong stalls auto-deploy until corrected.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Rewrite every route to `/api/v1/…`** | More churn and drops the unversioned alias for no benefit — the SPA is same-origin, so there are no external clients pinned to `/api`. |
| **Generate OpenAPI from handler annotations** (swaggo / oapi-codegen) | Heavier, and still can't introspect the gateway's composed `map[string]json.RawMessage` bodies. Author + drift-check is lighter and stays honest. |
| **e2e against a mocked broker** | Would pass while the real async path is broken — the whole point is to exercise JetStream + the outbox. |
| **Docker NATS service container in CI** | Works, but `services:` can't pass the `-js` flag and it adds an external dependency; embedded `nats-server` is self-contained and needs only Postgres. |
| **Add the named example indexes anyway** | They're redundant or reference a non-existent column — noise, not a speed-up. Honesty over box-ticking. |
| **Keep build-semver at 1.0** | Contradicts ADR-0009's plan; release tags give traceable, deliberate prod cuts. |
