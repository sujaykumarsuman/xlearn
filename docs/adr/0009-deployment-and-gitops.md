# ADR-0009 — Deployment & GitOps

- **Status:** Accepted. **Refined by [ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) (v2, Accepted 2026-09-24; see [Amended 2026-09-24 by ADR-0034](#amended-2026-09-24-by-adr-0034)):** bounded ImagePolicy ranges and the GA range flip, `runner-v*` and evalpack release streams, and rollback via kill switch → narrowed ImagePolicy → revert + patch tag → snapshot restore (pinning a tag in `infra` doesn't roll back under image automation).
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0002](0002-monorepo-vs-multi-repo.md), [0004](0004-inter-service-comms-and-events.md), [0005](0005-data-ownership-and-migrations.md), [0006](0006-authn-authz.md), [0007](0007-ai-coach-byo-key-and-secrets.md)
- **Source of truth for infra conventions:** the sibling repo `github.com/sujaykumarsuman/infra`.

## Context

xLearn deploys onto the existing `projects.sujaykumar.dev` single-node **k3s** cluster, driven by
**Flux + Helm** GitOps from the `infra` repo. The platform already defines: a shared `charts/project`
Helm chart, GHCR images auto-bumped by Flux image-automation, Traefik `IngressRoute`s under
`projects.sujaykumar.dev/<app>`, a shared CloudNativePG cluster, and SOPS/age secrets. xLearn must
**match these conventions**, not invent its own. It also introduces multiple services (not the
platform's usual one-image-per-app) and a new dependency (NATS).

## Decision

**Mirror the platform exactly; add only what xLearn needs.**

### Images & CI (in the xlearn repo)

- One image per service: `ghcr.io/sujaykumarsuman/xlearn-<service>` (e.g. `xlearn-gateway`,
  `xlearn-identity`, …), each from `deploy/<service>.Dockerfile`.
- **`ci.yml`** (PR + push to main): `go` job (`gofmt -l`, `go vet`, `go test -race ./...`, `sqlc diff`)
  and `web` job (typecheck, lint, test, build) — same shape as the airlift/landscape workflows.
- **`deploy.yml`** (push to main): for each changed service, call the reusable workflow
  `sujaykumarsuman/.github/.github/workflows/build-push.yml@main` with `image:` and `dockerfile:`,
  gated by **path filters** so only changed services rebuild.
- **Versioning:** **build-semver auto-deploy from `main`** for v1 (`0.<run>.x`, like `projects-hub`
  and `landscape`), ImagePolicy range `>=0.1.0`. Switch to **release tags** (`v1.x`, range `>=1.0.0`)
  at the 1.0 milestone. See [`../git-strategy.md`](../git-strategy.md).

### `infra` repo additions (the only deploy config; no `kubectl` by hand)

```
infra/
  infrastructure/messaging/           # NEW — NATS JetStream (HelmRelease + repo + namespace), small Longhorn PVC
  infrastructure/database/cluster/
    pg-xlearn.enc.yaml                # NEW — SOPS role password for the xlearn DB role
    xlearn-database.yaml              # NEW — CNPG Database CR: xlearndb, owner xlearn
    cluster.yaml                      # EDIT — add managed role `xlearn`
  clusters/vps/messaging.yaml         # NEW — Kustomization for infrastructure/messaging
  apps/
    xlearn-gateway.yaml               # NEW — HelmRelease (charts/project): route /xlearn, exposed
    xlearn-identity.yaml              # NEW — HelmRelease: ClusterIP only (route.enabled: false)
    xlearn-curriculum.yaml            # NEW
    xlearn-practice.yaml              # NEW
    xlearn-review.yaml                # NEW
    xlearn-assessment.yaml            # NEW
    xlearn-coach.yaml                 # NEW
    image-automation.yaml             # EDIT — ImageRepository/ImagePolicy per xlearn-<svc>
    secrets/
      xlearn-db.enc.yaml              # NEW — PGUSER/PGPASSWORD for xlearndb (matches pg-xlearn)
      xlearn-jwt.enc.yaml             # NEW — RSA signing key for gateway (ADR-0006)
      xlearn-coach.enc.yaml           # NEW — coach master key + OAuth client secrets (ADR-0007)
      xlearn-oauth.enc.yaml           # NEW — GitHub/Google OAuth client id/secret (identity)
```

- **Namespace:** a single **`xlearn`** namespace holds all service HelmReleases. Each service is a
  `HelmRelease` rendering `charts/project` with ~10 values (copy `apps/airlift.yaml` as the template).
- **Routing:** only **gateway** sets a route (`route.pathPrefix: /xlearn`, `stripPrefix: true`,
  `redirectSlash: true`, a suitable `priority`). All other services set `route.enabled: false`.
- **Database:** wire `xlearndb` per the infra 4-step pattern
  ([`infrastructure/database/README.md`](../../../infra/infrastructure/database/README.md));
  services get `PGHOST/PGPORT/PGDATABASE` as plain env + `PGUSER/PGPASSWORD` from the SOPS secret, and
  set their own `search_path`/schema (per-service role, [0005](0005-data-ownership-and-migrations.md)).
- **Secrets:** all via SOPS/age under `apps/secrets/*.enc.yaml`, `encrypted_regex ^(data|stringData)$`,
  decrypted in-cluster by Flux (age key only in the `sops-age` secret). Never committed in plaintext.
- **Pod hardening:** inherit the chart defaults (non-root, read-only rootfs, dropped caps).

### Environments & promotion

- **One environment (prod)** on the single node in v1; **trunk-based** — merge to `main` → CI image →
  Flux reconciles → live. No staging cluster ([PRD Q4](../prd/xlearn-prd.md#10-open-questions)).
- A `xlearn-staging` namespace (same chart, `/xlearn-staging` route, separate schema set) can be added
  later if pre-prod verification is needed — an additive infra change, not a re-architecture.

### Observability (v1, deliberately light)

- **Structured logs** (Go `slog`, JSON, to stdout) — collected via `kubectl logs` / Flux; request-id
  propagated gateway→services.
- **Health/readiness** via the chart's `probes.path` per service.
- **Metrics/tracing:** no Prometheus/OTel stack exists on the platform yet → **out of scope for v1**;
  recorded as a known gap. Add an OTel collector + metrics later if the node grows (would be a new ADR
  + infra addition).

### Amended 2026-09-24 by ADR-0034

[ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §1, §4 and §5, accepted at the v2
build-plan sign-off, refine **Versioning** and **Environments & promotion** above. The text above stays
as the v1 record; [`../git-strategy.md`](../git-strategy.md) is the operational version.

- **Versioning (the fleet).** Release tags `vX.Y.Z` (ADR-0021) build every fleet image. That is
  8 images once `xlearn-judge` joins as the 8th `deploy.yml` job (M3-1).
  - Every fleet ImagePolicy is **bounded to the live major**: `>=1.0.0 <2.0.0` until the v2.0 GA
    (infra#29), then `>=1.0.0 <3.0.0`.
  - A `.release-line` check in `deploy.yml` refuses to build a stable tag whose major differs
    (xlearn#53).
  - `-rc.N` prereleases build but never deploy.
- **Two more release streams**, each with its own range `>=1.0.0 <2.0.0`, where a major is a
  contract break:
  - **`runner-v*`** tags, built by a separate `runner-release.yml`. The runner has its own Flux
    Kustomization after `sandbox-guards` and a 2nd ImageUpdateAutomation (`update.path: ./runner`).
  - **Evalpack `v1.x`** tags in the private `xlearn-evalpack` repo. Its marker sits on judge's
    image-volume `reference:`.
- **Range changes have one fixed order each** (ADR-0034 §1.4):
  - a new policy or a raised floor: tag first, then merge (image before policy);
  - a superset widening: pre-flip check, merge, then tag;
  - a narrowing is only ever an R-b rollback.
- **Rollback, fastest first** (ADR-0034 §4). **Pinning a tag in `infra/apps/…` doesn't roll back:**
  image automation rewrites the marked line within about a minute. The ladder is:
  - **R-a:** the env kill switch;
  - **R-b:** narrow the ImagePolicy, never below the rollback floor;
  - **R-c:** revert plus a patch tag (the default);
  - **R-d:** the Hostinger snapshot, restored only by the §4.2 procedure (pin git back first; re-run
    later erases).
- **Environments & promotion.** **No `xlearn-staging` in v2** (ADR-0034 §5): it would break the
  memory-sum rule and couldn't host the runner. The substitutes are:
  - compose at prod parity;
  - a laptop k3d rehearsal;
  - `-rc` images;
  - cohort dark launch on prod (T-2 env and the T-3 `account.role` cohort).

  Revisit when a second node exists, or after the v3 opening once there are more than about 10 active
  learners and a contract migration touches their data.

## Consequences

- ✅ Deploys land the same way as every other project on the platform — no new ops model to learn.
- ✅ Path-filtered CI keeps the monorepo's multi-image builds fast.
- ✅ Secrets, DB, and routing all reuse existing, proven infra patterns.
- ⚠️ NATS is a **new stateful infra component** to introduce and babysit (small, but real).
- ⚠️ Several `xlearn-*` HelmReleases + image-automation entries are a bit more infra YAML than a
  single-app project; scripted from the `airlift.yaml` template to stay consistent.
- ⚠️ No metrics/tracing in v1 — debugging leans on structured logs until an observability stack lands.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Push-based deploy (CI `kubectl apply` / Helm)** | Violates the platform's pull-based GitOps rule ("nothing `kubectl apply`-ed by hand"); exposes the cluster to CI. |
| **Release-tag deploy from day one** (`>=1.0.0`) | Slower iteration for a pre-1.0 build; build-semver auto-deploy matches hub/landscape and suits rapid solo iteration. Adopt tags at 1.0. |
| **One combined `xlearn` image** | Contradicts [0003](0003-service-decomposition.md) (per-service deploy/scale). |
| **Separate DB cluster for xlearn** | Wasteful on one node; the shared CNPG cluster + schema-per-service ([0005](0005-data-ownership-and-migrations.md)) is the platform norm. |
| **Managed cloud (Vercel/Render/Fly)** | Off-platform cost + a second deploy model; the k3s/Flux platform already exists and is the stated target. |
