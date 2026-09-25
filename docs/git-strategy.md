# Git & release strategy

**Global** (spans all versions). How code moves from a branch to `projects.sujaykumar.dev/xlearn`,
mapped onto the Flux GitOps promotion flow ([ADR-0009](adr/0009-deployment-and-gitops.md)). Release
labels, gating and rollback for the v2 build are decided in
[ADR-0034](adr/0034-v2-release-labelling-gating-and-rollback.md); this page is its operational version.
**If the two disagree, the ADR wins** and this page gets fixed.

## Branching — trunk-based

- **`main`** is the single long-lived branch and is **always deployable**. Merging to `main` runs CI
  only; since 1.0, **a release tag** is what deploys ([release train](#release-train)).
- **Short-lived branches** off `main`, prefixed by type: `feat/…`, `fix/…`, `docs/…`, `chore/…`,
  `refactor/…`. Small, single-purpose, merged fast.
- **No** long-lived `develop`/`release` branches (solo dev + one environment → they'd only add drift).
- Merge via **PR**, squash-merge (one tidy commit per change on `main`). CI green + self-review required
  before merge. Direct pushes to `main` avoided (branch protection where practical).
- **Parallel sessions share this repo, its tags and `../infra`.** Before numbering an ADR, opening a PR
  on a shared file, or tagging, check peers' open PRs, tags and worktrees.

## Commits — Conventional Commits

`type(scope): subject` — types: `feat` `fix` `docs` `chore` `refactor` `test` `perf` `build` `ci`.

- **scope** = the service or area: `feat(practice): server-authoritative attempt timer`,
  `fix(review): reset touch level on failed re-solve`, `docs(adr): add 0004 events`.
- Imperative subject, ≤ ~72 chars; body explains **why** when non-obvious. Reference issues/ADRs.
- Conventional history feeds the generated release notes.
- Don't commit or push **mid-task**; the **end-of-session ship is a standing directive** and is its own
  authorization (see [AGENT.md](../AGENT.md) "land and sync") — no separate ask needed. Follow this
  commit format and the attribution the session specifies.
- **v2 sprints (D40): every sprint lands and syncs.** Launching a sprint prompt pre-approves every change
  it makes: merges, tags (contract, erase and GA included), infra PRs, board freezes, ADR acceptances and
  the production steps it specifies. Nothing stops for owner review. The sprint's **release action** only
  shapes the ending, which is always the prompt's `## Ship (land-and-sync — owner approval pre-granted)`:
  - a **tag** sprint tags and verifies live;
  - a **merge-only** sprint deploys nothing and names the tag that ships it;
  - a **design** sprint's merge is the board freeze;
  - a **spike** commits no code and lands only its results docs.

  The owner's in-session "hold / don't ship" still overrides.

## Versioning & release train

xLearn has used three deploy modes in sequence.

| Phase | Mode | Image tag | Fleet ImagePolicy | Trigger |
|-------|------|-----------|-------------------|---------|
| **v1 (pre-1.0)** | Build-semver auto-deploy (like `projects-hub`/`landscape`) | `0.<ci-run>.x` | `>=0.1.0` | push/merge to `main` |
| **1.0 → the v2.0 GA ← current** | Release-semver tags (like `airlift`), **bounded to the live major** | `vX.Y.Z` → `X.Y.Z` | `>=1.0.0 <2.0.0` | git tag `vX.Y.Z` |
| **v2.0 GA onward** | The same, one major up | `vX.Y.Z` → `X.Y.Z` | `>=1.0.0 <3.0.0` | git tag `vX.Y.Z` |

**Status:** 1.x since `v1.0.0` (S12). The live release is **v1.5.2**. `deploy.yml` triggers only on a
release tag, and all 7 fleet ImagePolicies are `>=1.0.0 <2.0.0` (infra#29). See
[ADR-0021](adr/0021-release-tagging-and-api-versioning.md) and the
[major-line guard](#major-line-guard-release-line).

### Release labels in v2 (D32)

| Label | Meaning |
|---|---|
| **`v1.x` minors** | Every v2 milestone while it's built. Take the **next free minor at tag time**: v1 fixes and peer sessions interleave, and semver sorts 1.10 above 1.9. New behaviour lands **dark** (a T-2 env switch or the T-3 owner/tester cohort) |
| **`v2.0.0`** | The **v2.0 GA**: flips the v2 defaults (judge and platform AI on, pilot course `active`) for the owner, once MI + M1–M4 + pilot + L are complete. No learner opening (that's v3) |
| **`v2.0.x`** | After GA: fixes, D6 content waves, and dark M6a/M6b code |
| **`v2.1.0`** | The interviewer GA (M6a + M6b) |
| a later **2.x minor** | M5, DSA `evaluator_only` |

- **Labels mark GA flips, not code landings.** From `v2.0.0` on, the minor moves only at a GA flip, and
  everything else is a patch.
- **Release titles name the milestone**, e.g. `v1.9.0 — v2 build · M2a`. `docs/v2/status.md` maps
  milestone → tag → rollback floor → the snapshot that preceded it.
- **The API stays `/api/v1`** for all of 2.x, additive only. The product version never forces an API
  major.
- The indicative tag timeline is [ADR-0034 §1.6](adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline).

### Major-line guard (`.release-line`)

xLearn stays on **1.x until the v2 GA** (D32). Flux always deploys the highest version in range, so
a stray major tag would ship the whole fleet, and no older-major tag could deploy after it. For
example, parallel sessions share tags, so one could push a `v2.0.0`. Two guards prevent this:

- **`.release-line`** (repo root) holds the live major, currently `1`. The first `deploy.yml` job
  fails the run **before any image is built** in two cases:
  - a stable tag's major differs from `.release-line`;
  - the tag isn't `vX.Y.Z[-prerelease]`.

  Prereleases (`vX.Y.Z-rc.N`) skip the major check and still build. No `ImagePolicy` range selects a
  prerelease, so they never auto-deploy. The check reads the file **at the tagged commit**, so a 1.x
  hotfix branched from the last 1.x tag still passes after `main` moves to `2`.
- **Bounded range:** every fleet `xlearn-*` `ImagePolicy` is `>=1.0.0 <2.0.0`. A tag runs the
  `deploy.yml` of the commit it points at, so this is the backstop for tags on commits that predate the
  guard.

The move to a new major is the [GA procedure](#ga-procedure-1x--20) below. After GA,
`.release-line = 2` blocks a stray `v3.*` the same way.

### Prereleases (`-rc`)

- **Form:** `vX.Y.Z-rc.N` from `main` (`N` from 1, never reused). The runner uses
  `runner-vX.Y.Z-rc.N`.
- **What happens:** the images build and push to GHCR as `X.Y.Z-rc.N`, and **nothing deploys**. Flux's
  semver ranges skip prereleases unless the range carries `-0`, and **no range ever does**. Mark the
  GitHub release as a prerelease.
- **What they're for:** a rehearsal in compose (or on the laptop k3d) of exactly the images a stable
  tag would build. The main uses are the GA (`v2.0.0-rc.N`), a contract migration, and a runner or
  evalpack major.
- **Prereleases don't count** in the GA pre-flip check. Only a non-prerelease `2.*` is a stray.

### Release streams

| Stream | Tag / workflow | Images | ImagePolicy range | A major means |
|---|---|---|---|---|
| **Fleet** | `vX.Y.Z` in this repo → `deploy.yml` (the `.release-line` guard, then one job per service) | every `xlearn-<svc>` at `X.Y.Z`: 7 today, **8 once `xlearn-judge` joins** as the 8th job (M3-1) | `>=1.0.0 <2.0.0`; `<3.0.0` from the GA | a GA default flip (D32) |
| **Runner** | `runner-vX.Y.Z` in this repo → a self-contained `runner-release.yml` (lands with `runner-v1.0.0`; `docker/metadata-action` `type=match` strips the prefix; reproducible build; digest output). `deploy.yml`'s `v*` filter never matches it, and vice versa | `xlearn-runner` only | `>=1.0.0 <2.0.0`; **not widened at GA** | a judge↔runner contract break |
| **Evalpack** | `vX.Y.Z` in the private `xlearn-evalpack` repo → its own CI (with the anonymous-GET probe) → a private GHCR data image | `xlearn-evalpack` only | `>=1.0.0 <2.0.0` (ImageRepository with a pull `secretRef`); **not widened at GA** | a pack contract break (`contract_hash` covers ordinary skew) |

- **Runner:** it has its own Flux Kustomization (`runner`, `dependsOn: sandbox-guards`, off the `apps`
  `wait` path) and a 2nd ImageUpdateAutomation (`update.path: ./runner`). It uses `strategy: Recreate`,
  because its Quota would stall a surge. A runner patch is a toolchain patch and needs the ±5% speed
  check.
- **Evalpack:** its tag marker sits on judge's image-volume `reference:`, so the shared IUA bumps it,
  and **each pack bump restarts judge**. The pull PAT's expiry date is a manual check recorded in
  `docs/v2/status.md` (no alerting, D34).
- **A runner or pack major is GA in miniature:**
  1. judge learns the new contract in an ordinary fleet tag;
  2. run the pre-flip check for that package, then merge the superset widening (`<2.0.0` → `<3.0.0` on
     that one policy);
  3. tag the major.

### ImagePolicy range changes

Ordinary releases **never touch a range**; a release is only a tag. Each kind of range change has one
fixed order ([ADR-0034 §1.4](adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure)):

| Range change | When | Order | Why |
|---|---|---|---|
| **None** | every ordinary release | — | ranges are set once |
| **A new policy, or a raised floor** | a new service (judge at M3-1, the runner at MI-12); ADR-0021's 1.0 flip | **tag first, then merge** (image before policy) | the policy matches nothing until the image exists, and the deploy stalls |
| **A superset widening** (`<2.0.0` → `<3.0.0`) | the GA; a runner or pack major | **pre-flip check, merge, then tag** | nothing moves at the merge, so the tag stays the single deploy act |
| **A narrowing** (`!=x.y.z`, or back to `<2.0.0`) | an R-b rollback only | **the merge is the rollback** | undo it after the fix |

**Pre-flip check** (before any widening): no non-prerelease tag of the new major exists, either as a
git tag (`git ls-remote --tags origin 'refs/tags/v2*'`) or on any affected GHCR package (the fleet
packages are public, so an anonymous `crane ls` works). A stray would **deploy the moment the range
widens**:
- delete it first (the git tag and the image version);
- if it can't be deleted, cut the release as a version above it and tag **before** widening. The
  snapshot then precedes the widening.

### GA procedure (1.x → 2.0)

Copied into the GA sprint; the full checklist is in the rollout plan (§4) and
[ADR-0034 §1.4](adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure).

1. **GA PR (xlearn):** set `.release-line` to `2`, and flip the T-1 defaults (judge and platform AI on,
   pilot `active`). The release notes carry the behaviour-change list. Merging deploys nothing.
2. **Optional rehearsal:** tag `v2.0.0-rc.N` and run the images in compose. They never deploy.
3. **Pre-flip check** (above): no non-prerelease `2.*` git tag or GHCR tag on any fleet package.
4. **Widen (infra PR):** every **fleet** ImagePolicy becomes `>=1.0.0 <3.0.0`, **merged before the
   tag**. It's a superset, so nothing moves. The runner and evalpack policies stay `<2.0.0`.
5. **Snapshot:** the last Hostinger weekly image is ≤ 7 days old, the host has settled, and
   `host-verify --cluster` is green. Then take the manual snapshot.
6. **Tag `v2.0.0`.** `2.0.0` carries **no contract migration**, so R-b back to `<2.0.0` returns to the
   last 1.x.
7. **Verify and record** ([release checklist](#release-checklist)).

### Ordering rules for a release

- **Image before HelmRelease or policy.** For a new service, tag the release that first builds its image,
  **then** merge the infra PR: the ADR-0005 4-step schema and role, the HelmRelease, and the
  ImageRepository and ImagePolicy. A brief crash-loop until the schema exists self-heals, as v1's
  services did.
- **Infra ACL PR before a new consumer.** A new NATS stream, consumer or subject needs its **infra ACL
  PR** (the re-rendered golden `authorization` block from `topology.go`) **merged before the consuming
  service's tag**. Otherwise the service is permission-denied at runtime
  ([ADR-0035 §2](adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).
  The same holds for a new caller of a NetworkPolicy-guarded port.
- **Consumers before producers.** Use two consecutive tags, or ship the producer dark behind a T-2
  switch that an infra PR flips once the consumers are bound. A single tag has no ordering, because one
  IUA commit bumps every service, and unknown subjects are acked silently. The subject-registry test
  enforces this in CI.
- **Expand → backfill → contract.** Expand releases hold additive DDL only. A contract comes at least
  one release after the last reader or writer of the old shape. Mark it with
  `-- xlearn:contract floor=vX.Y.Z` (CI lint), rehearse it in compose on seeded data, record the floor
  in `docs/v2/status.md`, and snapshot right before the tag. **Never run goose `Down` in production**;
  fix forward.
- **Infra PRs stand alone.** They are their own tasks, never folded into a tag.
- **Memory.** A new always-on pod is checked against the memory-sum rule
  ([ADR-0035 §5](adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)),
  and every new container carries a memory limit.

### Release train

```
branch (feat/…) ──PR──▶ main ──CI(ci.yml)──▶ green   (merging deploys nothing: main is build-only)
                                   │
  git tag vX.Y.Z ──deploy.yml──▶ .release-line guard ▶ build every fleet image xlearn-<svc> @ X.Y.Z ▶ GHCR
  (vX.Y.Z-rc.N builds too, but no range selects a prerelease, so it never deploys)
                                   │
  Flux image-automation (fleet range >=1.0.0 <2.0.0; <3.0.0 from GA) bumps infra/apps/xlearn-<svc>.yaml ▶ commit
                                   │
                         Flux helm-controller upgrades HelmRelease ▶ prod

  git tag runner-vX.Y.Z ──runner-release.yml──▶ xlearn-runner @ X.Y.Z ▶ 2nd IUA (./runner) ▶ runner Kustomization
  evalpack repo tag vX.Y.Z ──its CI──▶ private xlearn-evalpack @ X.Y.Z ▶ IUA bumps judge's image volume ▶ judge restarts
```

### Release checklist

The authoritative list is
[ADR-0034 §6](adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist); every tag
sprint copies it. In short:

- **Before the tag:**
  - peers' tags and PRs are checked;
  - it's the next free version, and its major equals `.release-line`;
  - ACL PRs for new streams and consumers are merged;
  - a new service's image comes before its policy;
  - for a contract: rehearsed in compose, floor marked;
  - for a contract, erase or GA tag: `host-verify --cluster` green, the host settled, then the manual
    snapshot;
  - from M6: no live interviews.
- **After the tag, by looking** (there is no alerting, D34):
  - `/xlearn/api/v1/healthz` reports the version;
  - `k3s kubectl get deploy -n xlearn` shows the new images;
  - every fleet ImagePolicy's latest equals the tag, and the HelmReleases are Ready;
  - smoke-test login, the dashboard and coach.
- **Record** milestone → tag → floor → snapshot, and any flag change, in `docs/v2/status.md`.

## Environments & promotion

- **One environment: prod** (single node). "Promotion" is a release tag on `main`; merging alone
  deploys nothing.
- **No staging in v2** ([ADR-0034 §5](adr/0034-v2-release-labelling-gating-and-rollback.md#5-no-staging-environment-in-v2)).
  An `xlearn-staging` namespace would break the memory-sum rule, needs a second NATS, and can't host
  the runner. The substitutes are:
  - compose at prod parity (PG 18, NATS 2.14);
  - the laptop k3d rehearsal;
  - `-rc` images, which build and never deploy;
  - **cohort dark launch on prod**: a T-2 env switch plus the T-3 `account.role` owner/tester cohort.
    v2 is owner-only (D35), so a regression reaches only the owner and testers.

  Revisit when a second node exists, or after the v3 opening once there are more than about 10 active
  learners and a contract migration touches their data. Adopting it would be a new ADR.
- **Never `kubectl apply` by hand**; all cluster change flows through `infra` + Flux. The sanctioned
  manual paths are:
  - the admin CLIs via `kubectl exec` (`identity admin`, `judge admin`);
  - the NATS ops break-glass (an `ssh vps` port-forward and the offline ops seed);
  - the host scripts (`hack/host-bootstrap.sh`, `hack/host-verify.sh`).

  Log each break-glass use in `docs/v2/status.md`.

## Tags, hotfixes, rollback

- **Tags:**
  - `vX.Y.Z` for the fleet (since 1.0);
  - `runner-vX.Y.Z` for the runner;
  - `vX.Y.Z` in `xlearn-evalpack` for the pack;
  - a `-rc.N` suffix for prereleases.

  Tags are annotated, and GitHub release notes are generated. Pre-1.0 used CI-run build numbers, not
  tags.
- **Never move or re-push a tag.** Re-pushing overwrites the GHCR tag, and Flux doesn't roll, because
  the tag string is unchanged. Fix forward with a patch tag.
- **Hotfix:** branch from the tag, `fix:`, and tag `vX.Y.(Z+1)`. After the GA, a hotfix is a `v2.0.x`
  patch. A 1.x hotfix branched from the last 1.x tag still passes the guard, but it deploys only while
  the fleet range is narrowed back to `<2.0.0` (R-b).

### Rollback

**Pinning the previous tag in `infra/apps/…` does not roll back.** The tag line keeps its
`$imagepolicy` marker, and image automation rewrites it within about a minute. Use the ladder instead,
fastest first ([ADR-0034 §4](adr/0034-v2-release-labelling-gating-and-rollback.md#4-rollback)):

| | Mechanism | Time | Limits |
|---|---|---|---|
| **R-a** | **Env kill switch** (T-2): an infra PR on the HelmRelease env, e.g. `JUDGE_BASE_URL` unset, the grading override, `LLM_PLATFORM_ENABLED=false`, `REVISION_ENTRY_RULE`, `SIGNUP_MODE=closed` | 1–2 min | gated capabilities only |
| **R-b** | **Narrow the ImagePolicy** per service, e.g. `>=1.0.0 <2.0.0, !=1.7.0`. The IUA writes the previous tag back, with no build | 2–3 min | never below the **rollback floor**, and a consumer never below its consumer tag. Undo it after the fix |
| **R-c** | **Revert on `main` plus a patch tag.** **The default**: best for traceability | 6–8 min | never below the rollback floor |
| **R-d** | **Hostinger snapshot restore** (whole VM), **only as the procedure below** | slowest | only within ~1 day of the tag; **loses every write since the snapshot** |

The **rollback floor** is the lowest tag every service can run against the current schema and event
log. `docs/v2/status.md` records it per tag. R-b and R-c never go below it; only R-d can.

**R-d is a procedure, not a button.** A restore rewinds the cluster **but not git**. The IUA has
already committed the bad tags to `infra/main`, so on first boot Flux would redeploy the bad release
and re-run its migration on the restored database. In order:

1. **Pin git back first.** Do R-b (narrow to the pre-tag version) or R-c, and wait until the IUA has
   written the old tags to `infra/main`. The old pods may fail against the new schema until the
   restore; that's expected.
2. **Confirm** that git HEAD's `xlearn-*` tags equal the snapshot's. `docs/v2/status.md` records which
   tag each snapshot preceded.
3. **List the erases completed since the snapshot** (account ids only) from the live database. There
   is no erase ledger (D12), so the restore would bring those accounts back.
4. **Restore** the snapshot in hPanel.
5. **Run `host-verify --cluster`**, then the release checklist's verify step.
6. **Re-run the listed erases** through the CLI, and record the lost-writes window in
   `docs/v2/status.md`.

A restore also rewinds the host (the kernel, for example), so snapshot only after host changes have
settled.

**Snapshot rule.** Take a Hostinger manual snapshot right before any **contract, erase or GA** tag.
There is one at a time, and it is kept for 1 day. The GA also needs the last weekly image to be
≤ 7 days old. There is no off-node `pg_dump` (D12). The snapshot is an owner-only hPanel step, so in v2
it sits in the tag prompt's `## Before you launch (owner)` block and is taken right before launch (D40).

**Never:**
- edit the tag line (the IUA rewrites it);
- suspend the shared IUA (it also freezes airlift, landscape, hub and kubescope);
- run goose `Down`;
- move a tag.

Detection is manual (D34): the release checklist's verify step runs within minutes of every tag.

## CI gates (`.github/workflows/ci.yml`)

Runs on PR + push to `main` (mirrors the sibling repos):

- **go:** `gofmt -l` (must be empty) · `go vet ./...` · `go test -race ./...` · `sqlc diff` (generated code current).
- **web:** `npm ci` · typecheck · lint · test · build.
- Path filters skip a job when its tree is untouched. **Nothing deploys from CI or from a merge**: a
  release tag runs `deploy.yml` ([ADR-0021](adr/0021-release-tagging-and-api-versioning.md)).
- **v2 adds, as each lands:**
  - the NATS topology tests (the golden ACL block, the stream-budget sum, the subject registry);
  - the contract-header lint (`-- xlearn:contract floor=…`);
  - the course-manifest golden and slug-guard tests;
  - the runner's reproducible-build check.

## Repo hygiene

- `docs/adr/` append-only; notable decisions get an ADR before/at merge.
- `docs/vN/status.md` updated when a chunk of build work lands. For v2 it also carries
  milestone → tag → rollback floor → snapshot, the flag inventory, and the break-glass log.
- Conventional-commit history is the changelog source; keep it clean (squash).
