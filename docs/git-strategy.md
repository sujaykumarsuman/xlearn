# Git & release strategy

**Global** (spans all versions). How code moves from a branch to `projects.sujaykumar.dev/xlearn`,
mapped onto the Flux GitOps promotion flow ([ADR-0009](adr/0009-deployment-and-gitops.md)).

## Branching — trunk-based

- **`main`** is the single long-lived branch and is **always deployable**. Merging to `main` triggers
  CI → image → Flux → prod ([release train](#release-train)).
- **Short-lived branches** off `main`, prefixed by type: `feat/…`, `fix/…`, `docs/…`, `chore/…`,
  `refactor/…`. Small, single-purpose, merged fast.
- **No** long-lived `develop`/`release` branches (solo dev + one environment → they'd only add drift).
- Merge via **PR**, squash-merge (one tidy commit per change on `main`). CI green + self-review required
  before merge. Direct pushes to `main` avoided (branch protection where practical).

## Commits — Conventional Commits

`type(scope): subject` — types: `feat` `fix` `docs` `chore` `refactor` `test` `perf` `build` `ci`.

- **scope** = the service or area: `feat(practice): server-authoritative attempt timer`,
  `fix(review): reset touch level on failed re-solve`, `docs(adr): add 0004 events`.
- Imperative subject, ≤ ~72 chars; body explains **why** when non-obvious. Reference issues/ADRs.
- Conventional history feeds changelog/versioning at 1.0.
- **Do not commit or push unless asked** ([AGENT.md](../AGENT.md)); when asked, follow this format
  and the attribution the session specifies.

## Versioning & release train

Two modes exist on the platform; xLearn adopts them in sequence.

| Phase | Mode | Image tag | ImagePolicy | Trigger |
|-------|------|-----------|-------------|---------|
| **v1 (pre-1.0)** | **Build-semver auto-deploy** (like `projects-hub`/`landscape`) | `0.<ci-run>.x` | `>=0.1.0` | **push/merge to `main`** |
| **1.0+** | **Release-semver tags** (like `airlift`) | `vX.Y.Z` → `X.Y.Z` | `>=1.0.0` | **git tag `vX.Y.Z`** |

- **v1:** every merge to `main` builds the changed service images and Flux deploys them — fast solo
  iteration, no ceremony. Roll back by reverting the commit (Flux redeploys the prior image) or by
  pinning a tag in `infra`.
- **1.0+:** cut a release by tagging `vX.Y.Z` (semver: breaking→major, feature→minor, fix→patch).
  Tags are annotated; GitHub release notes are generated. `infra` image-automation switches its range to
  `>=1.0.0`. This ADR-0009 switch is a deliberate milestone, recorded in `docs/vN/status.md`.

### Release train

```
branch (feat/…) ──PR──▶ main ──CI(ci.yml)──▶ green
                                   │
                          push→deploy.yml builds changed xlearn-<svc> images ▶ GHCR
                                   │
                    Flux image-automation bumps infra/apps/xlearn-<svc>.yaml ▶ commit
                                   │
                         Flux helm-controller upgrades HelmRelease ▶ prod
```

## Environments & promotion

- **v1: one environment — prod** (single node). "Promotion" = merge to `main`. No staging.
- Later, a `xlearn-staging` namespace (separate schemas, `/xlearn-staging` route) can front `main`
  while tagged releases go to prod — additive, recorded as a new ADR if adopted.
- **Never** `kubectl apply` by hand; all cluster change flows through `infra` + Flux.

## Tags, hotfixes, rollback

- **Tags:** `vX.Y.Z` (from 1.0). Pre-1.0 uses CI-run build numbers, not tags.
- **Hotfix (post-1.0):** branch from the tag, `fix:`, tag `vX.Y.(Z+1)`.
- **Rollback:** revert on `main` (auto-redeploys) or pin the previous image tag in `infra/apps/…`
  and let Flux reconcile. Prefer revert for traceability.

## CI gates (`.github/workflows/ci.yml`)

Runs on PR + push to `main` (mirrors the sibling repos):

- **go:** `gofmt -l` (must be empty) · `go vet ./...` · `go test -race ./...` · `sqlc diff` (generated code current).
- **web:** `npm ci` · typecheck · lint · test · build.
- Path filters skip a job when its tree is untouched, but the **merge to `main`** is what deploys —
  see `deploy.yml` ([ADR-0009](adr/0009-deployment-and-gitops.md)).

## Repo hygiene

- `docs/adr/` append-only; notable decisions get an ADR before/at merge.
- `docs/vN/status.md` updated when a chunk of build work lands.
- Conventional-commit history is the changelog source; keep it clean (squash).
