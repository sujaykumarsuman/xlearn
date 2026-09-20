# ADR-0002 — Monorepo vs multi-repo

- **Status:** Accepted
- **Date:** 2026-09-20
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0009](0009-deployment-and-gitops.md)

## Context

xLearn is service-oriented (multiple Go services + a React web app + curriculum content), but is
built and operated by a solo developer on a single-node k3s cluster via Flux GitOps. The sibling
`infra` repo already assumes **one image per app repo**, published to `ghcr.io/sujaykumarsuman/<app>`
and auto-deployed by Flux image-automation. We must choose how xLearn's code is laid out across git
repositories.

## Decision

**A single application monorepo:** `github.com/sujaykumarsuman/xlearn`, containing all services,
the web app, shared Go packages, and the curriculum seed. Deployment config stays in the sibling
`infra` repo (unchanged convention).

- Go **workspace** (`go.work`) with one module per service is avoided; instead **one Go module**
  with `cmd/<service>/` entrypoints and shared `internal/` packages (simplest for a solo dev; matches
  the airlift/landscape layout). Split into multiple modules only if build times or dependency
  isolation demand it.
- **Multiple images** are produced from the one repo — one per deployable service — each with its
  own `deploy/<service>.Dockerfile` and a matching CI build job (see [0009](0009-deployment-and-gitops.md)).
- The web app lives at `web/` and is embedded into / served by the gateway service.

See [repo layout](../architecture/overview.md#repository-layout) for the tree.

## Consequences

- ✅ One PR can change a service + the web + shared contracts atomically — no cross-repo version dance.
- ✅ Shared types (event schemas, domain enums, API DTOs) live in `internal/` and can't drift.
- ✅ One CLAUDE.md / AGENT.md / docs tree governs everything; one place to clone.
- ⚠️ CI must build only the images that changed (path filters) to stay fast — addressed in [0009](0009-deployment-and-gitops.md).
- ⚠️ "One repo, many images" is a small deviation from the infra norm of one-image-per-repo; the
  `infra/apps/` side is unaffected (it just gains several `xlearn-*` HelmReleases and image-automation entries).

## Alternatives considered

| Option | Pros | Why not |
|--------|------|---------|
| **Multi-repo** (one repo per service) | Maximal isolation; matches infra's one-image-per-repo. | Cross-cutting changes span N PRs; shared contracts drift; N× the CI/CLAUDE.md/branch overhead for a solo dev. |
| **Go multi-module workspace in one repo** | Dependency isolation per service. | Extra `go.work` friction, versioning of internal modules — premature for one small codebase. |
| **Monorepo, single image (modular monolith binary)** | Simplest deploy. | Contradicts the repo's explicit "distributed, not a monolith" mandate; loses independent scaling/deploy. |
