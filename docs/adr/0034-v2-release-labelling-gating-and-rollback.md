# ADR-0034 — v2 release labelling, feature gating and rollback

- **Status:** Proposed. **The accidental-major guard (§1.3) shipped on 2026-09-24:** xlearn#53 (`.release-line` = `1` plus the `deploy.yml` check) and infra#29 (every `xlearn-*` ImagePolicy `>=1.0.0 <2.0.0`). Everything else applies from M1.
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0021](0021-release-tagging-and-api-versioning.md) (Release tagging, API v1 and the 1.0 hardening cut; **§1 refined here**), [0009](0009-deployment-and-gitops.md) (Deployment & GitOps; **Versioning and Environments & promotion refined here**), [0033](0033-invite-only-admission-and-owner-admin.md) (`account.role`, `SIGNUP_MODE`, testers, the v2 audience), [0035](0035-v2-operations-nats-auth-limits-capacity.md) (NATS ACL ordering, memory sum, no alerting), [0026](0026-per-course-extensibility-model.md) (course manifest `status`), [0027](0027-content-evalpack-and-user-data-model.md) (evalpack stream, `contract_hash`), [0028](0028-object-storage-and-backups.md) (D12: no backups), [0029](0029-judge-contract-and-learning-signal.md), [0030](0030-runner-technology-and-host-hardening.md) (runner stream), [0031](0031-platform-ai-and-two-tier-keys.md) (`LLM_PLATFORM_ENABLED`), [0005](0005-data-ownership-and-migrations.md) (migrations, the 4-step), [0018](0018-progress-projection-grain-and-rebuild.md) (drop and replay).
- **Links:** v2 topic T7 (D32, D35), [feasibility § T7](../v2/feasibility.md#t7--cross-cutting-and-infra-first-rollout), the [rollout plan](../v2/rollout-plan.md) and the [research appendix](../v2/research/t7-cross-cutting-and-rollout.md).

## Context

**How a release reaches production today:**

| Fact | Consequence | Evidence |
|---|---|---|
| `deploy.yml` fires on **any** `v*` tag and rebuilds all 7 images at that version (**fixed 2026-09-24 by xlearn#53:** a tag off `.release-line` builds nothing) | whoever pushes a tag ships the fleet | `.github/workflows/deploy.yml:13-15` |
| All 7 `xlearn-*` ImagePolicies are `>=1.0.0` with **no upper bound** (**fixed 2026-09-24 by infra#29:** `>=1.0.0 <2.0.0`) | **a stray `v2.0.0` deploys the whole fleet in about 4 minutes, and no 1.x tag can ever deploy again** (2.0.0 stays the highest) | `infra/apps/image-automation.yaml` |
| **Parallel agent sessions share this repo and its tags**, and "v2" is the planning name everywhere | a `v2.0.0` tag is an easy mistake | `git worktree list` (5 agent worktrees on 2026-09-24); AGENT.md "End of session — land and sync" |
| The documented rollback, "pin the previous tag in `infra/apps/…`", **doesn't work**: the tag line keeps its `$imagepolicy` marker and image automation rewrites it within about a minute | there is no working documented rollback | `docs/git-strategy.md` (Tags, hotfixes, rollback) |
| Flux semver ranges (Masterminds) **skip prereleases** unless the range carries `-0` | `-rc.N` tags build images that never auto-deploy | [Flux ImagePolicy](https://fluxcd.io/flux/components/image/imagepolicies/), [Masterminds/semver](https://github.com/Masterminds/semver/blob/master/README.md) |
| **One ImageUpdateAutomation** (1 m) writes every bump in **one commit** | services in one tag upgrade in no set order, so "same release" gives no consumer-before-producer ordering | `infra/apps/image-automation.yaml` (IUA) |
| Migrations run at startup (goose v3.28.0, advisory lock), and goose **ignores DB versions missing from the files** | an older image boots against a newer schema without error, so **image rollback works only if the old code tolerates the newer schema**. `Down` blocks exist; nothing runs them in prod | goose `internal/gooseutil/resolve.go` |
| Readiness and liveness both probe `/healthz`; `/readyz` exists but nothing uses it | a pod that fails its dependencies still reports Ready | `infra/charts/project/templates/deployment.yaml:58-66` (one `probes.path` for both); `infra/apps/xlearn-*.yaml` (`probes.path: /healthz`) |
| Tag to live ≈ 4–6 min; revert plus a patch tag ≈ 6–8 min | rollback by image is minutes, not seconds | `gh run list`; Flux intervals |
| **One environment, one node, no backups (D12).** A Hostinger manual snapshot is one at a time, **auto-deleted after 1 day**, and restores the whole VM | the only undo for data is a same-day snapshot | [Hostinger backups](https://support.hostinger.com/en/articles/1583232-how-to-back-up-or-restore-a-vps) |
| The HTTP API `/api/v1` is independent of the product version | the API never forces a major (v1.5.1 removed a field in a patch) | ADR-0021 §2 |

**v2 decisions this ADR builds on:**
- **D35:** v2 is **used by the owner only**, plus CLI-minted testers. The learner gate L still ships and is exercised, but **inviting real learners is the owner's call, planned for v3**.
- **D34:** **no alerting in v2.** A bad release is seen by the release checklist's verify step or when the owner looks (landscape, kubescope).
- **D28:** the interviewer is **v2.1**.
- **D6:** its content scope is delivered across the **v2.x line** in waves (D35); `v2.0.0` doesn't wait for it.

## Decision

### 1. Release labels (D32)

#### 1.1 Scheme

| Label | Meaning | Expected (inferred) |
|---|---|---|
| **`v1.x` minors** | Every v2 milestone while it is built: the next free minor at tag time. New behaviour lands **dark** (§2) | Oct–Dec 2026 |
| **`v2.0.0`** | **v2.0 GA**: flips the v2 defaults (judge and platform AI on, pilot course `active`) **for the owner**, once MI + M1–M4 + pilot + L are complete. **No learner opening** (D35) | Dec 2026–Jan 2027 |
| **`v2.0.x`** | After GA: fixes, **D6 content waves** and dark M6a/M6b code | from GA |
| **`v2.1.0`** | **Interviewer GA** (M6a + M6b, D28) | Q1 2027 |
| a later **2.x minor** | **M5**, DSA `evaluator_only`, after the last DSA pack is stamped. It rides v2.1.0 if it's ready by then | H2 2027 |

- **Labels mark GA flips, not code landings.** Code lands dark in earlier tags. The labelled tag **changes the build-time defaults** (T-1, §2). Before `2.0.0`, the judge- and AI-dependent method (D15/D16/D18, evaluator grading, platform AI) and `preview` courses reach only the T-3 cohort. D27 (M1b), D31 (M1b; the v1.5.2 stopgap didn't carry it) and D2 (M2, kill switch `REVISION_ENTRY_RULE`) ship to every account in their 1.x minors. `2.0.0` flips the D32 defaults (judge and platform AI on, pilot `active`) for every account, which in v2 means the owner and testers. That flip is the breaking change the major marks.
- **From `v2.0.0` on, the minor moves only at a GA flip.** Everything else is a patch. That keeps `2.1.0` for the interviewer.
- **The opening isn't a label.** Inviting real learners is `SIGNUP_MODE=invite` (T-2) plus ADR-0033's opening gates, and it is planned for v3. How v3 is labelled is v3 planning's call; nothing here needs a major for it.
- **API:** `/api/v1` for all of 2.x, additive only. Keep the DSA alias routes for **at least one release after the SPA stops calling them** (open-tab skew). `/api/v2` comes only with an external client.

#### 1.2 Hygiene

- GitHub release titles name the milestone, e.g. `v1.9.0 — v2 build · M2a`.
- `docs/v2/status.md` carries **milestone → tag → rollback floor**, the snapshot each risky tag was preceded by, and the flag inventory (§2).
- **Before tagging, check peers' tags and open PRs** (parallel sessions).
- **Never move or re-push a tag.** Re-pushing overwrites the GHCR tag, and Flux doesn't roll because the tag string is unchanged. Fix forward with a patch tag.

#### 1.3 Accidental-major guard (shipped 2026-09-24: xlearn#53, infra#29)

- **Infra:** bound every `xlearn-*` ImagePolicy to **`>=1.0.0 <2.0.0`**. judge's policy is born with the same bound.
- **xlearn:** add a one-line **`.release-line`** file (`1` until GA), plus a first `deploy.yml` step, which every image job needs. The step **fails when the tag's major ≠ `.release-line` at the tagged commit**, so no image is built.
- **Result:**
  - a stray `v2.*` tag builds nothing and would deploy nothing;
  - the major bump becomes a reviewed PR;
  - after GA, `.release-line = 2` blocks a stray `v3.*` the same way;
  - the check reads the file at the tagged commit, so a 1.x hotfix branched from the last 1.x tag still passes after `main` moves to `2`.

#### 1.4 Range changes and the GA procedure

**Resolves "tag first, then flip" (ADR-0021) against "widen before the tag":** ordinary releases never touch a range. Each kind of range change has one fixed order:

| Range change | When | Order | Why |
|---|---|---|---|
| **None** | every ordinary release | — | ranges are set once; a release is only a tag |
| **New policy, or a raised floor** (it matches nothing until the image exists) | a new service (judge), a new stream, ADR-0021's 1.0 flip | **tag first, then merge** (image before policy) | otherwise the policy matches nothing and the deploy stalls |
| **Superset widening** (`<2.0.0` → `<3.0.0`) | GA; a runner or pack major | **merge first, then tag**, after the pre-flip check | nothing moves at the merge, so the tag stays the single deploy act |
| **Narrowing** (`!=x.y.z`, back to `<2.0.0`) | R-b rollback only (§4) | the merge *is* the rollback | undone after the fix |

**GA procedure (ordered):**
1. **GA PR (xlearn).** Set `.release-line` to `2`, and flip the T-1 defaults: judge and platform AI on, pilot course `active`. The release notes carry the behaviour-change list. Merging deploys nothing, because `main` is build-only.
2. **Optional rehearsal.** Tag `v2.0.0-rc.N` from `main`. The images build and never deploy; run them in compose.
3. **Pre-flip check.** No non-prerelease `2.*` may exist, neither a git tag (`git ls-remote --tags origin 'refs/tags/v2*'`) nor a tag on any `xlearn-*` GHCR package (public, so an anonymous `crane ls` works). A stray would **deploy the moment the range widens**.
   - If one exists, delete it (the image version and the git tag) first.
   - If it can't be deleted, cut GA as a version above it and tag it **before** widening (steps 4 and 6 swap; the snapshot then precedes the widening), so the policy's first 2.x pick is the real GA.
4. **Widen (infra PR).** Change every `xlearn-*` ImagePolicy to `>=1.0.0 <3.0.0` and merge. Nothing moves, since the highest non-prerelease is still 1.x.
5. **Snapshot.** Check that the last Hostinger weekly image is ≤ 7 days old and that host changes have settled. Then take the manual snapshot (§4.3).
6. **Tag `v2.0.0`.**
7. **Verify and record** (§6).
- **Rollback of GA:** R-b back to `>=1.0.0 <2.0.0` redeploys the last 1.x. GA carries **no contract migration**, so 1.x runs on its schema.

#### 1.5 Other release streams

| Stream | Tags / workflow | Range | Notes |
|---|---|---|---|
| **judge** (fleet) | the **8th `deploy.yml` job**; every `v*` tag builds it | `>=1.0.0 <2.0.0`, flips at GA with the rest | **Image before policy:** tag the release that first builds `xlearn-judge` (M3-1), *then* merge the infra PR: the ADR-0005 4-step schema and role, the HelmRelease, and the ImageRepository and ImagePolicy. A brief crash-loop until the schema exists self-heals, as v1's services did |
| **runner** | `runner-v*` tags; a **self-contained** `runner-release.yml` (`docker/metadata-action` `type=match,pattern=runner-v(\d+\.\d+\.\d+.*),group=1`; reproducible build; digest output). `deploy.yml`'s `v*` filter doesn't match them | `>=1.0.0 <2.0.0` | `clusters/vps/sandbox.yaml` (`sandbox-guards`, then `runner`), a 2nd IUA with `update.path: ./runner`, and **`strategy: Recreate`** (its Quota would stall a surge). A **major is a judge↔runner contract break**; a patch is a toolchain patch with the ±5% speed check |
| **evalpack** | `v1.x` tags in `xlearn-evalpack` | `>=1.0.0 <2.0.0` | ImageRepository with `secretRef`. Its marker sits on judge's image-volume `reference:`, so the existing IUA bumps it, and each pack bump restarts judge. `contract_hash` covers skew (ADR-0027). A **major is a contract break** |

**A runner or pack major is GA in miniature:**
1. judge learns the new contract in an ordinary xlearn tag;
2. the pre-flip check and the superset widening are merged;
3. the major is tagged.

#### 1.6 Indicative tag timeline

Take the next free minor at tag time: v1 fixes and peer sessions may interleave, and semver sorts 1.10 above 1.9. L's tags **stay in v2** (D35).

| Tag | Content | Gate state after | Rollback floor after |
|---|---|---|---|
| v1.5.2 | the ADR-0033 §2 stopgap: #51 auto-link fix + #52 `SIGNUP_MODE` closed (infra#30 sets `closed`). `open` isn't yet tied to `DEV_AUTH` (M1b) | signup closed (since 2026-09-24) | — |
| — (infra + CI) | ✅ **§1.3 guard**: every range `<2.0.0` (infra#29); the `.release-line` check (xlearn#53) runs from the next tag | — | — |
| v1.6.0 | M1a expand: identity `role`/`status` columns; consumers accept the v2 envelope; N0 if ready | no behaviour change | none (expand only) |
| v1.7.0 | M1b: producers emit the v2 envelope; course resolution; `withhold()`; limits; `RequireRole`; `identity admin` verbs | golden = v1 (D27 confirm, D31 and 429s aside) | **1.6.0** (v2 envelopes in the log) |
| v1.8.0 | M1c **contract** (snapshot first) | — | **1.7.0, hard** |
| — (infra) | NATS N1 → N2 → N3 (ADR-0035) | — | client images ≥ the N0 tag |
| v1.9.0 | M2a `touch_concluded` consumer **and the M2b consumers**; producers idle | producers off | — |
| v1.10.0 | M2a/M2b producers on; projections v2 replay; M2c D2 rule | D2 on (`REVISION_ENTRY_RULE` = kill switch) | 1.9.0 |
| v1.11.0 → v1.12.0 | L-E erase consumers (practice, review, assessment, coach) → `DELETE /api/me` and its UI (snapshot before 1.12.0) | web erase: **testers only**; owner refused (CLI only) | 1.11.0 |
| runner-v1.0.0 · evalpack v1.0.0 | runner dark (MI-12); first pack | runner reachable only from judge | — |
| v1.13.0 → v1.14.0 | M3-1 judge dark, born with its `XLEARN_IDENTITY` erase consumer (8th image; infra PR **after** the tag) → M3-2 Run/Submit, arena, the D15/D16/D18 flow | `JUDGE_BASE_URL` set; judge features cohort-only | 1.13.0 |
| v1.15.0 (+ runner-v1.1.0) | P: pilot course (go-concurrency) as `preview`; the go-race profile | `preview` = cohort only | consumers ≥ 1.7.0 |
| v1.16.0 | M4 platform AI | `LLM_PLATFORM_ENABLED` + cohort | — |
| v1.17.0 | L-A: invite flow, acceptance, notice, 18+, region, consents; web erase opens to every non-owner account. **An invite round-trip is rehearsed on prod with a tester**, then `SIGNUP_MODE` returns to `closed` | `closed` | — |
| (`v2.0.0-rc.N`) → **v2.0.0** | **GA** (§1.4): the default flip for the owner; no learner opening | kill switches stay | GA has no contract; R-b to `<2.0.0` returns to the last 1.x |
| v2.0.x | D6 content waves (self-tier items are compiled into images; packs ride evalpack `v1.x`); M6a/M6b dark | cohort | — |
| **v2.1.0** | **Interviewer GA** (M6a + M6b) | — | — |
| a later 2.x minor | M5 `evaluator_only` | env override = kill switch | reversible |

### 2. Feature gating: three tiers, no flag service

| Tier | Mechanism | Change path | Used for |
|---|---|---|---|
| **T-1 build-time** | course manifest `status: active \| preview \| coming_soon \| retired`; per-course grading `allowed \| evaluator_only`; code defaults | a tag (4–6 min) | **GA flips**, which *are* the labelled releases |
| **T-2 runtime env** | HelmRelease env: `JUDGE_BASE_URL` (unset = off), the grading override, `LLM_PLATFORM_ENABLED`, `REVISION_ENTRY_RULE`, `COURSE_STATUS_OVERRIDE`, `SIGNUP_MODE` | an infra PR (1–2 min, pod restart) | kill switches, dark launch, operating modes |
| **T-3 cohort** | `account.role ∈ {owner, tester}` from session-validate; the gateway gates `preview` features on it. **Never in the JWT** (ADR-0033) | `identity admin` CLI, immediate | dogfood before GA; non-owner paths through testers |

**Rules:**
- **Presence by config.** A capability is present only if its upstream is configured **and** its gate passes (the `DEV_AUTH` precedent).
- **Dark-launch path:**
  1. code off (T-2);
  2. the cohort (T-3);
  3. the **default flip in code at the labelled tag**;
  4. the flag is removed by its removal milestone.
- **Kill switches and operating modes are permanent:**
  - the grading override;
  - `JUDGE_BASE_URL`;
  - `LLM_PLATFORM_ENABLED`;
  - `REVISION_ENTRY_RULE`;
  - `SIGNUP_MODE`.
- **≤ 6 live non-kill flags.** Each has an owning milestone and a removal milestone in `docs/v2/status.md`.
- **`preview` hides a course everywhere outside the cohort:** the catalog, enrollment, the public profile and `/public/stats`. It is tested like the course-slug guard. Anonymous profile viewers exist even in an owner-only v2.
- **Lazy chunks.** Add a `vite:preloadError` → reload handler with the first lazy import (M3). Hashed assets are `immutable`, so a tab open across a deploy would otherwise 404.

### 3. Migration and event-compatibility rules

- **Expand** releases hold only additive DDL:
  - nullable or constant-default columns;
  - new tables;
  - `CREATE INDEX CONCURRENTLY` under `-- +goose NO TRANSACTION`.

  Backfill in the same or the next release.
- **Contract** at least one release after the last reader or writer of the old shape.
  - Mark the file with `-- xlearn:contract floor=vX.Y.Z` (CI lint) and record the floor in status.md.
  - **Rehearse it in compose on seeded data**, and take a snapshot right before the tag.
- **Rollback floor:** the lowest tag every service can run against the current schema and event log. R-b and R-c never go below it; only R-d can.
- **Never run `Down` in production.** Fix forward with a new Up migration.
- **Readiness gates cutover.** Chart 0.3.0 splits the probes, and each release opts in to readiness on `/readyz`, so a failed migration never takes traffic.
- **Consumers before producers.** Two consecutive tags, **or** the producer ships dark behind a T-2 switch that an infra PR flips once consumers are bound. A single tag has no ordering (one IUA commit), and unknown subjects are acked silently (`internal/review/consumers.go:58-61`). The subject-registry test (ADR-0035) enforces it in CI.
- **A new stream or consumer** needs its NATS ACL PR merged **before** the service tag (ADR-0035).
- **Envelope:** append-only. Decoders keep v1 and v2 **forever**. The first non-DSA event sets the consumer floor at the path-aware release (1.7.0, indicative).

### 4. Rollback

#### 4.1 Mechanisms, fastest first

| | Mechanism | Time | Limits |
|---|---|---|---|
| **R-a** | Env kill switch (T-2) | 1–2 min | gated capabilities only |
| **R-b** | **Narrow the ImagePolicy** per service, e.g. `>=1.0.0 <2.0.0, !=1.7.0`. The IUA writes the previous tag back; no build | 2–3 min | never below the rollback floor, and a consumer never below its consumer tag. Undo it after the fix |
| **R-c** | Revert on `main` plus a patch tag | 6–8 min | the default; best for traceability |
| **R-d** | Hostinger snapshot restore (whole VM), **as the procedure in §4.2** | the slowest | only within ~1 day of the tag; **loses every write since the snapshot** |

**Never:**
- edit the tag line (the IUA rewrites it);
- suspend the shared IUA (it also freezes airlift, landscape, hub and kubescope);
- run goose `Down`;
- move a tag.

**Detection is manual (D34).** The §6 verify step runs within minutes of every tag.

#### 4.2 R-d is a procedure, not a button

A restore rewinds the cluster **but not git**. The IUA has already committed the bad tags to `infra/main`, so on first boot Flux re-deploys the bad release, and its migration re-runs on the restored database.

1. **Pin git back first.** Do R-b (narrow to the pre-tag version) or R-c, and wait until the IUA has written the old tags to `infra/main`. The old pods may fail against the new schema until the restore; that's expected.
2. **Confirm** that git HEAD's `xlearn-*` tags equal the snapshot's (status.md records which tag each snapshot preceded).
3. **List the erases completed since the snapshot** (account ids only) from the live database. With no erase ledger (D12), the restore would bring those accounts back.
4. **Restore** the snapshot in hPanel.
5. **Run `host-verify --cluster`**, then the §6 verify step.
6. **Re-run the listed erases** through the CLI, and record the lost-writes window in status.md.

**A restore also rewinds the host** (for example the kernel), so only snapshot after host changes have settled.

#### 4.3 Snapshot rule

- **Take a Hostinger manual snapshot right before any contract, erase or GA tag.** It is one at a time, kept for 1 day, and inside D12.
- The GA checklist also requires the last **weekly** image to be ≤ 7 days old.
- **No off-node `pg_dump`.** It puts PII off the node and bends D12.

#### 4.4 Reversibility by step

| Step | Reversible? | Irreversible residue | Guard |
|---|---|---|---|
| Chart 0.3.0 (MI-3) | revert `Chart.yaml` (re-renders all 11 releases) | none | byte-identical `helm template` for every release (a new `hack/chart-diff.sh`, written in MI-3) |
| NetworkPolicies (MI-5, MI-5a) | git revert (fail-open) | none | caller list from live pods; smoke-test login, dashboard and coach |
| M1a/M1b v2 envelope | yes, while DSA-only | v2 envelopes stay in the outboxes and JetStream for good | decoders forever |
| **M1c contract** | **no, below 1.7.0** | dropped columns and CHECKs | floor marker; compose rehearsal; snapshot |
| NATS N1–N3 | the server config reverts | none (the outbox buffers) | server-first bridge (ADR-0035) |
| M2a/M2b new subjects | yes | events acked silently if a consumer is rolled below its consumer tag while the producer is on | two tags or a dark producer; registry test |
| M2b projections v2 | drop and replay (ADR-0018) | none while the outbox is kept | never trim the outbox |
| **M2c D2 rule** | `REVISION_ENTRY_RULE` stops new anchors | D2-created items stay scheduled | stamp `revision_item.anchor_rule` |
| **L-E erase** | code, yes | **deleted data is gone** (D12, no backups); an R-d restore resurrects later erases (§4.2 step 3) | testers only until the L exit; owner never on the web; typed confirmation; snapshot |
| MI-11/MI-12 host block, runner | restore the files and restart k3s; remove the `runner` Kustomization path | none | `host-verify`; runner off `apps`' `wait` path |
| M3 judge | `JUDGE_BASE_URL` unset, or the grading override | evaluations and `graded_by=auto` history stay | the kill switch is an M3 exit test; nothing is re-graded |
| P pilot | manifest back to `preview` (tag) or `COURSE_STATUS_OVERRIDE` | enrollments and events with `path_slug ≠ dsa` | consumer floor ≥ 1.7.0 |
| **M4 platform AI** | `LLM_PLATFORM_ENABLED=false` | **money spent** | D25's dogfood limits in owner-only v2: provider $15, app cap $12 ($100 / $80 from the v3 opening) |
| L-A admission | `SIGNUP_MODE=closed` | the invite-created rehearsal accounts (role `learner`; each erases itself on the web at the L exit) | only tester-operated round-trips in v2 (D35) |
| **v2.0.0 GA** | R-b back to `<2.0.0` | owner and tester data written under the new defaults | 2.0.0 carries **no** contract migration |
| M5 evaluator-only | the env override brings the picker back | none | — |

### 5. No staging environment in v2

**No `xlearn-staging`.** On this node it would cost:
- **≈ +1.3 GiB of memory limits.** The ADR-0035 memory sum already has ≈ 0 margin, and this breaks its rule even after MI-11a.
- **A second NATS.** Stream names and subjects are hard-coded, so staging events would feed prod consumers.
- A second database with 8 roles and secrets, a second OAuth app, and about 25 infra objects. Every release would deploy twice.
- **No runner.** Its Quota and the host block can't be duplicated, so the riskiest v2 part couldn't be staged anyway.

**Substitutes:**
- compose at prod parity (PG 18, NATS 2.14);
- the laptop k3d rehearsal;
- **`-rc` images**, which build and never deploy;
- **cohort dark launch on prod** (T-2/T-3).

D35 makes the last one cheap: throughout v2 (until the v3 opening), v2's only users are the owner and testers.

**Revisit** when a second node exists, or after the v3 opening, once there are more than about 10 active learners and a contract migration touches their data.

### 6. Release checklist

- **Before the tag:**
  - peers' tags and PRs are checked;
  - it is the next free version, and its major equals `.release-line`;
  - ACL PRs for new streams and consumers are merged;
  - a new service's image comes before its policy;
  - for a contract: rehearsed in compose, floor marked;
  - for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken;
  - from M6: no live interviews.
- **After the tag (by looking, D34):**
  - `/xlearn/api/v1/healthz` reports the version;
  - `k3s kubectl get deploy -n xlearn` shows the new images;
  - every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready;
  - smoke-test login, the dashboard and coach.
- **Record** milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`.

### 7. What this changes elsewhere (not done in this ADR)

- **ADR-0021 §1 is refined, not superseded:**
  - the ranges are bounded;
  - its "tag, then flip" becomes the §1.4 table;
  - `-rc` tags are added;
  - labels are GA flips;
  - the API stays `/api/v1`.
- **ADR-0009 is refined:**
  - Versioning gets bounded ranges and 8 images;
  - Environments & promotion: staging is declined for v2 (§5).

  Their status lines point back here. 0021's and 0009's notes are in place.
- **`docs/git-strategy.md` must be rewritten in the build** (a downstream task):
  - the rollback section becomes R-a to R-d, and "pin the tag" is dropped;
  - add the `-rc` convention, the runner and evalpack streams, `.release-line` (✅ landed with xlearn#53, with the bounded ranges and the GA steps), the §1.4 range rules, "image before HelmRelease/policy" and "infra ACL PR before a new consumer";
  - the staging note is updated;
  - the versioning table shows the bounded ranges.

## Consequences

- ✅ **No session can ship a stray major.** A mistaken `v2.*` builds nothing and deploys nothing, and 1.x can't be frozen.
- ✅ **Labels mean something.** `2.0.0` is the D32 default flip (judge, platform AI, pilot `active`) and `2.1.0` the interviewer, matching D28. The API never forces a major.
- ✅ **Rollback has a working, ordered path.** The broken "pin the tag" instruction is retired, and the snapshot restore can't roll itself forward.
- ✅ **Nothing new to run.** There is no flag service and no staging on a node with ≈ 0 memory margin, and gating reuses env, the manifest and `account.role`.
- ✅ **D35 makes prod dark-launch safe.** Until the v3 opening, a v2 regression reaches the owner and testers only.
- ⚠️ **GA is a two-repo sequence** (the pre-flip check, then widen, snapshot and tag). Getting it wrong deploys a stray image or stalls. §1.4 is the checklist.
- ⚠️ **1.x minors carry dark v2 code, and post-GA patches carry dark M6 code.** Release titles and status.md must map tags to milestones.
- ⚠️ **A contract sets a hard floor.** Below it only R-d works: within ~1 day, losing every write since the snapshot, and re-running erases.
- ⚠️ **The snapshot is manual** (hPanel, one at a time, 1 day). Forgetting it removes R-d for that release.
- ⚠️ **No staging.** The first run of a migration on real data is production. The mitigations are the compose rehearsal and expand/contract discipline; the runner is rehearsed only on k3d.
- ⚠️ **Rollback needs someone to look (D34).** A failed release is found by the verify step or by chance.

## Alternatives considered

| Option | Why not |
|---|---|
| `v2.0.0` only at M5 (every DSA item judge-graded) | GA hangs on +230–340 h of authoring and inverts D28's v2.1 |
| `v2.0.0` at M1 | M1 changes no behaviour, and v2.1 would become M2, which contradicts D28 |
| Product names decoupled from semver (1.x forever) | A permanent "v1.23 = v2.1" confusion |
| ImagePolicy bound only, no `.release-line` check | A stray tag still builds a `2.0.0` image and tag that must be deleted before GA, and it would deploy at the widening |
| Discipline only | Parallel sessions share tags; one mistake freezes 1.x |
| At GA, tag `v2.0.0` first and widen after | Works, but the deploy moment moves to an infra merge, away from the snapshot. Kept only as the fallback when a stray 2.x can't be deleted |
| Dark code in post-GA minors | `2.1.0` would be taken before the interviewer GA (D28) |
| A flag service (Unleash, flagd, OpenFeature) | A 7th stateful thing to run, for one user on one node |
| Account-id allowlist in HelmRelease env | Every cohort change is a restart; `account.role` through the CLI is immediate |
| A role claim in the JWT | ADR-0033: roles live in the DB, never in the JWT |
| An `xlearn-staging` namespace | ≈ +1.3 GiB of limits, a second NATS, ~25 objects, and it can't host the runner |
| An off-node `pg_dump` before risky tags | PII leaves the node; bends D12 |
| "Pin the previous tag in infra" (today's docs) | The IUA rewrites the marker within a minute |
| goose `Down` for rollback | No GitOps path, and `kubectl exec` is off-limits; fix forward |
| One tag with both consumer and producer | One IUA commit gives no order; early events are acked silently and lost |
