# Sprint ga-02 — Cut v2.0.0: pre-flip check, widen ranges, snapshot, tag, verify

> **Milestone:** GA — v2.0 GA, the owner-facing default flip (**part 2 of 2**: [ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) steps 3–7, exactly in order) · **Track:** product · **Order:** 70
> **Prereqs:** [ga-01](sprint-ga-01.md):
> - the GA PR merged (on CI green, D40: `.release-line = 2`, T-1 flips);
> - `v2.0.0-rc.N` rehearsed in compose, including R-b;
> - no active learner;
> - `hack/ga-preflip-check.sh` on `main`.
>
> **Unblocks:**
> - [m6a-01](sprint-m6a-01.md) (its gate "`v2.0.0` live") and so M6a/M6b ([m6a-06](sprint-m6a-06.md), [mi-13](sprint-mi-13.md) and [m6b-03](sprint-m6b-03.md) cut `v2.0.x` patches; [m6b-04](sprint-m6b-04.md) cuts `v2.1.0`);
> - [m5-01](sprint-m5-01.md);
> - the D6 content waves as `v2.0.x` patches (`ev-d6-waves`).
>
> **Release action:** **tag `v2.0.0`**, the owner-facing GA ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): "GA (§1.4): the default flip for the owner; no learner opening". Gate state after: kill switches stay. Rollback floor: GA has no contract, and R-b to `<2.0.0` returns to the last 1.x). Two things come first:
> - **one infra PR** in `../infra`: the superset widening of the 8 fleet ImagePolicies. It is **merged before the tag**, is its own task, and is never folded into the tag;
> - the owner's **manual Hostinger snapshot** (`ev-ga`), taken **before launch** (D40).
>
> The session runs the rest with no owner stop.
>
> **Calendar:** **GA day** (≈ Dec 2026 – Jan 2027). Owner time is before launch only (~10 min: the weekly-image check and the snapshot); nobody needs to watch the verify. Hostinger keeps a manual snapshot about 1 day, so the owner launches this prompt right after taking it, and the tag (task 6) lands **the same day**.
> **Execute with:** [`../prompts/prompt-ga-02.md`](../prompts/prompt-ga-02.md). One prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pre-flight: gates re-read, the tag freeze announced, `main` = the rc'd commit (else `-rc.N+1`) | X | ⬜ |
| 2 | Pre-flip check: `hack/ga-preflip-check.sh --major 2 --cluster` all PASS (step 3) | X | ⬜ |
| 3 | Widen: infra PR, 8 fleet ImagePolicies → `>=1.0.0 <3.0.0` (runner and evalpack untouched), merged **before** the tag; nothing moves (step 4) | I | ⬜ |
| 4 | `host-verify --cluster` green on a settled host (step 5, first half) | H | ⬜ |
| 5 | Snapshot `ev-ga`: weekly image ≤ 7 d, then the manual snapshot (step 5), taken by the owner before launch | O (before launch) | ⬜ |
| 6 | Tag `v2.0.0` + GitHub release (step 6) | X | ⬜ |
| 7 | Verify (ADR-0034 §6) + the GA smoke (step 7) | X | ⬜ |
| 8 | Env clean-up (conditional): infra PR dropping env lines for flags ga-01 removed, after the verify | I | ⬜ |
| 9 | Record; end the tag freeze; land and sync | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md):
> - the Sprint board row;
> - **GA ✅** in Milestones;
> - milestone → tag → floor → snapshot;
> - the range table and the release line;
> - the flag inventory;
> - owner events (`ev-ga`);
> - the CLI-use log;
> - the decisions log.
>
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

**GA checklist** ([rollout §4](../rollout-plan.md#v20-ga-the-owner-facing-default-flip), verbatim, in ADR-0034 §1.4 order):
- **Green at entry:** the first three lines and items 1–2, delivered by [ga-01](sprint-ga-01.md) and re-read in task 1.
- **This sprint's tasks:** items 3–7 and the last line.

- [ ] MI + M1–M4 + P + L complete; the M3 checklist (§5) is still green
- [ ] `identity admin account list` shows **no `active` account with role `learner`**: every stranger from v1's open signup has been suspended or erased (ADR-0033 §2)
- [ ] `2.0.0` carries **no contract migration**
- [ ] **1. GA PR (xlearn):** set **`.release-line = 2`** and flip the T-1 defaults (judge and platform AI on, pilot `active`). The release notes carry the behaviour-change list.
- [ ] **2. Optional rehearsal:** `v2.0.0-rc.N`, run in compose. Prereleases never deploy.
- [ ] **3. Pre-flip check:** no non-prerelease `2.*` git tag, and no such GHCR tag on any `xlearn-*`
- [ ] **4. Widen (infra PR):** every `xlearn-*` range becomes `>=1.0.0 <3.0.0`, **merged before the tag**. It's a superset, so nothing moves.
- [ ] **5. Snapshot:** the last Hostinger weekly image is ≤ 7 days old, the host has settled, and `host-verify --cluster` is green. Then take the **manual snapshot**.
- [ ] **6. Tag `v2.0.0`.**
- [ ] **7. Verify and record** (ADR-0034 §6): the `/xlearn/api/v1/healthz` version, the images in `get deploy -n xlearn`, the ImagePolicies' latest, HelmReleases Ready, and a smoke test; record milestone → tag → floor → snapshot
- [ ] `SIGNUP_MODE` stays `closed`, and no invite is minted (the opening is v3)

*Reading note (not part of the verbatim list):* "every `xlearn-*`" in items 3, 4 and 7 means the **8 fleet images** that `deploy.yml` builds: `xlearn-gateway`, `-identity`, `-curriculum`, `-practice`, `-review`, `-assessment`, `-coach` and `-judge`. **`xlearn-runner` and `xlearn-evalpack` are separate streams.** They stay `>=1.0.0 <2.0.0`, and a major there is a contract break, GA in miniature ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams); D32). ga-01 aligned the docs.

**Sprint gates:**

- [ ] **[ga-01](sprint-ga-01.md) ✅:**
  - the GA PR # is merged, and `cat .release-line` on `origin/main` → `2`;
  - `v2.0.0-rc.N` is published as a prerelease, and its four-phase compose rehearsal is green (R-b included);
  - the **rc'd commit SHA** is recorded in status.md;
  - `hack/ga-preflip-check.sh` and its self-test are green on `main`.
- [ ] **Still no active learner** (things may have changed since ga-01): `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list --role learner --status active'` → empty. Show the output in the terminal only.
- [ ] **Production healthy and settled:**
  - no host change, reboot, k3s/CNPG bump or restart-inducing infra PR in the last 24 h (status.md, `git -C ../infra log --since=24.hours origin/main`);
  - no failing Flux object (`ssh sujaykumar-vps 'k3s kubectl get kustomizations,helmreleases -A'` all Ready).
- [ ] **Before launch (owner), attested by the launch (D40):**
  - the last Hostinger weekly image is ≤ 7 days old, and the manual snapshot is taken on the settled host (task 5); the launch message gives the snapshot's time or id and the weekly-image date;
  - he has set his go-concurrency public visibility (AB22) as he wants it. After the tag, the pilot's rows are public when visible ([ga-01](sprint-ga-01.md) release notes).
- [ ] **Parallel sessions, and the tag freeze:**
  - no peer tag or open PR claims `v2.0.0`, edits `../infra/apps/image-automation.yaml`, or plans a tag from `main` (`git ls-remote --tags origin`, `gh pr list --state open` in both repos, `git worktree list`, ListAgents);
  - **every peer has been told the freeze** (task 1): no tag and no merge to xlearn `main` or to `../infra/apps/image-automation.yaml` until this sprint reports `v2.0.0` verified.
- [ ] **No live interviews:** n/a. The interviewer isn't built until M6a, which this tag unblocks.

## Goal

Cut **`v2.0.0`**, the **owner-facing v2.0 GA** ([D32](../feasibility.md#decisions-log-newest-first), [D35](../feasibility.md#decisions-log-newest-first); [rollout §4](../rollout-plan.md#v20-ga-the-owner-facing-default-flip)), by executing [ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) steps 3–7 **exactly in order**:
1. **pre-flip check:** no stable 2.x exists anywhere;
2. **superset widening:** merged first, and nothing moves;
3. **`host-verify --cluster` + the manual snapshot** (the owner's, taken before launch);
4. **tag `v2.0.0`**, on the commit ga-01 rehearsed;
5. **verify by looking** (D34) **and record.**

After it:
- judge and platform AI are on, and go-concurrency is `active`, **for every account**. In v2 that's the owner and testers;
- `SIGNUP_MODE` stays `closed`, and no invite is minted. The opening is v3 ([rollout §11](../rollout-plan.md#11-opening-gates-v3));
- **there is no dogfood gate:** ≥ 2 weeks of M4 data is an opening gate, not a GA gate.

## Scope

**In**
- A tag freeze across peer sessions for the sitting.
- Running `hack/ga-preflip-check.sh` (written and self-tested in [ga-01](sprint-ga-01.md) so that `main` doesn't move after the rc), including the `--cluster` reads before and after the widening.
- **One `../infra` PR:** 8 fleet `range` lines in `apps/image-automation.yaml` → `>=1.0.0 <3.0.0`, plus its header comment.
- `host-verify --cluster --with-runner --nats-stage=n4`; recording the owner's before-launch snapshot.
- The `v2.0.0` annotated tag and its GitHub release (body = ga-01's release notes).
- The ADR-0034 §6 verify, the GA smoke, and the record.
- **Conditional, after the verify:** one `../infra` PR dropping the HelmRelease env lines of flags ga-01 removed, if ga-01 recorded any (task 8). None is expected.

**Out**
- Changing `xlearn-runner` or `xlearn-evalpack` ranges. They are separate streams, and a runner or pack major is its own procedure (ADR-0034 §1.5).
- Any `SIGNUP_MODE` change or invite; `SEAT_CAP`; the platform-AI limit raise; the alerting revisit. These are v3 opening gates ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- Code changes. The GA PR is [ga-01](sprint-ga-01.md). A bug found here is fixed forward (Rollback below), never by moving the tag.
- `v2.0.x` patches: M6a dark ([m6a-06](sprint-m6a-06.md)), MI-16 ([mi-13](sprint-mi-13.md)), M6b dark ([m6b-03](sprint-m6b-03.md)), the D6 content waves; `v2.1.0` ([m6b-04](sprint-m6b-04.md)); M5 ([m5-01](sprint-m5-01.md)).

## Tasks

### 1 · Pre-flight [X]

- **Tag freeze.** Tell every active peer session (ListAgents → SendMessage): "v2.0 GA in progress: no tags, and no merges to xlearn `main` or `../infra/apps/image-automation.yaml`, until I report `v2.0.0` verified." From the widening merge (task 3) until the tag, **any** stable 2.x image would deploy: Flux takes the highest version in range. This freeze covers that window.
- **Re-read the gates** (above). If one fails, stop.
- **`main` = the rehearsed commit:**

  ```sh
  git fetch --tags origin
  RC=$(git rev-parse v2.0.0-rc.N^{commit})
  git merge-base --is-ancestor "$RC" origin/main && echo "rc is on main"
  git diff --stat "$RC" origin/main -- . ':!docs' ':!*.md' ':!design-system'
  ```

  **One rule:** the diff must be **empty**, and then the tag commit is **`origin/main`** (the rc'd commit or a docs-only descendant of it). Only `.dockerignore`d paths may have changed since the rc, and `.github/` is **not** exempt, because the tag runs `deploy.yml` at the tagged commit. **If the diff is not empty, stop:** don't tag `$RC` behind a `main` that has moved on with code (it could drop a cherry-picked 1.x hotfix). Cut `v2.0.0-rc.N+1` from `origin/main`, re-run [ga-01](sprint-ga-01.md) task 10's phases 1–3 on it, record its SHA, and repeat this check against the new rc **before** continuing.
- `.release-line` at the tag commit = `2`. `hack/lint-migrations.sh --no-contract-since <last-1.x>` is green. Here `<last-1.x>` = `git tag -l 'v1.*' --sort=-v:refname | grep -v -- - | head -1`, expected `v1.17.0`.

### 2 · Pre-flip check [X] (ADR-0034 §1.4 step 3)

Run `hack/ga-preflip-check.sh --major 2 --cluster` at the tag commit. **Every line must PASS:**
- **git:** no stable `refs/tags/v2*`. `v2.0.0-rc.*` is listed as OK;
- **GHCR:** no tag on the 8 fleet packages that Flux would parse as a stable 2.x (`2`, `2.0`, `v2.0.0`, `2.0.0+…` all count; `2.0.0-rc.*` doesn't). The fleet is parsed from `deploy.yml` and asserted as exactly 8;
- **no contract** since `<last-1.x>`;
- **`--cluster`:** the 8 fleet policies are `>=1.0.0 <2.0.0` with `latestRef.tag` = `<last-1.x>`. `xlearn-runner` and `xlearn-evalpack` are `>=1.0.0 <2.0.0` at their own latest. Paste this table into the widening PR.

**If a stray exists**, the ADR gives two branches. The session takes them in this order (D40: its call, approved by the launch):
- **Delete it (preferred).**
  - The git tag: `git push origin :refs/tags/<tag>`.
  - The GHCR version: `gh api -X DELETE /users/sujaykumarsuman/packages/container/xlearn-<svc>/versions/<id>` for each affected package. That needs a token with `delete:packages`. The ids come from `gh api /users/sujaykumarsuman/packages/container/xlearn-<svc>/versions`.

  Both deletions are **irreversible**; launching this prompt approves them (D40). A stray git tag can always be deleted; if the token lacks `delete:packages`, the GHCR version can't be, so take the next branch. Then re-run the check until it passes.
- **Cut GA above it** if it can't be deleted:
  1. tag a version higher than the stray (for example `v2.0.1` if `2.0.0` exists);
  2. **tag before widening:** steps 4 and 6 swap. The order becomes check → `host-verify` → tag → images exist → widen → verify (the snapshot was already taken, before launch), so the policy's first 2.x pick is the real GA.
  3. **Keep the no-contract guard.** Only a GHCR version can resist deletion; a stray **git** tag can always be deleted, so delete it anyway. `deploy.yml`'s no-contract step ([ga-01](sprint-ga-01.md) task 5) fires for a major's first stable release, meaning no other non-prerelease `v2.*` git tag exists, so it still runs for the chosen tag. Also run `hack/lint-migrations.sh --no-contract-since <last-1.x>` at the chosen tag commit (task 1) and paste the result into the record.
  4. Record the deviation in the decisions log. The release title still says "v2.0 GA".

Re-run the check **immediately before** merging task 3's PR if more than ~10 minutes have passed.

### 3 · Widen [I] (ADR-0034 §1.4 step 4)

In `../infra`, from an up-to-date `main`, branch `chore/xlearn-fleet-ranges-v2-ga`:
- **`apps/image-automation.yaml`.** For **exactly** these 8 ImagePolicies, change `range: ">=1.0.0 <2.0.0"` → `range: ">=1.0.0 <3.0.0"`:

  `xlearn-gateway`, `xlearn-identity`, `xlearn-curriculum`, `xlearn-practice`, `xlearn-review`, `xlearn-assessment`, `xlearn-coach`, `xlearn-judge`

  - **Don't** touch `xlearn-runner` or `xlearn-evalpack`. They sit in the same file with the same range string, so **no global `sed`**: edit per document.
  - Don't touch any ImageRepository, the IUAs, or any `apps/xlearn-*.yaml` tag line (the IUA owns those).
- **Header comment:** "xlearn-* fleet: release semver bounded to the live major (`<3.0.0` since the v2.0 GA; widen to `<4.0.0` before a `v3.0.0`, xlearn `docs/git-strategy.md`). `xlearn-runner` and `xlearn-evalpack` are separate streams at `<2.0.0`."
- **Self-check** in the PR body: a before/after table of `metadata.name → spec.policy.semver.range` for every ImagePolicy in the file, for example

  ```sh
  yq -r 'select(.kind=="ImagePolicy") | .metadata.name + " " + (.spec.policy.semver.range // "-")' apps/image-automation.yaml
  ```

  **Exactly 8 rows differ.** Paste in task 2's `--cluster` table too.
- **Commit:** `chore(xlearn): widen fleet ImagePolicies to <3.0.0 for the v2.0 GA (ADR-0034 §1.4)`, with the attribution lines.
- **PR body:** "Superset widening: merge first, then tag. Nothing moves, because the highest stable fleet image is still `<last-1.x>`. Pre-flip check green at <time>."
- **Merge** (squash). `../infra` has no CI: the before/after table and the pre-flip output in the PR body are its checks. No `kubectl apply`: Flux reconciles `apps`.
- **Verify that nothing moved** (read-only):
  - `ssh sujaykumar-vps 'k3s kubectl get kustomization apps -n flux-system -o jsonpath={.status.lastAppliedRevision}'` shows the merge SHA;
  - `hack/ga-preflip-check.sh --major 2 --cluster` shows the 8 fleet ranges `>=1.0.0 <3.0.0` with `latestRef.tag` still `<last-1.x>`, and runner and evalpack unchanged;
  - `git -C ../infra fetch && git -C ../infra log origin/main --oneline -3` shows no `chore(images)` commit touching `apps/xlearn-*.yaml`;
  - `ssh sujaykumar-vps 'k3s kubectl get pods -n xlearn'` shows no new restarts or ages.

  If anything moved, **stop**: a stray exists that the check missed. Narrow back (revert the PR, R-b), then investigate.

From here until task 6, keep the sitting short and the freeze held.

### 4 · `host-verify --cluster` [H] (step 5, first half)

`ssh sujaykumar-vps 'bash /root/host-verify.sh --cluster --with-runner --nats-stage=n4'` ([mi-02](sprint-mi-02.md); [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34)) must show **no FAIL**:
- the memory sum with the runner's 3 GiB counted ≤ capacity − 0.5 GiB ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)). GA adds no pod and no container, so the sum is unchanged;
- Flux Ready;
- no OOMKills and ≤ 3 restarts in 24 h; CNPG healthy;
- PVCs < 60%, node disk < 70%;
- NATS `auth_required: true` with no `legacy` user;
- the expected NetworkPolicies present;
- sar steal and CPU below the TR-* thresholds.

**Settled** means no host change and no restart-inducing change in the last 24 h (gate above). Paste the summary into the record.

### 5 · Snapshot [O, before launch] (`ev-ga`; step 5)

[ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule): a manual snapshot right before any GA tag, plus the weekly image ≤ 7 days. Under D40 the owner does both **before launching** this prompt, and the launch attests them:
1. The owner reads the date of the **last Hostinger weekly image** in hPanel. If it is more than 7 days old, he waits for the next weekly image before launching.
2. The owner takes the **manual snapshot** on a settled host (no host change, reboot or restart-inducing infra PR in the last 24 h). It's one at a time, replaces any earlier manual one, and is auto-deleted after about 1 day. The launch message carries its time or id and the weekly-image date; the session records them.
3. The tag follows **the same day**. The widening (task 3) moves nothing, so the snapshot still holds the fleet at `<last-1.x>` and the pre-GA data. If the tag can't land inside the snapshot's life, don't tag on a lapsed snapshot: revert task 3's PR (the ranges go back to `<2.0.0`), end the freeze, and record ⛔ "snapshot lapsed; re-take it and relaunch" in `status.md`.

R-d relevance: the snapshot holds the fleet at `<last-1.x>` and the pre-GA data. It stays useful for about 1 day.

### 6 · Tag `v2.0.0` [X] (step 6)

Run the release checklist (§Release). This is an irreversible step, so first check again for peer and owner messages, and re-run `git ls-remote --tags origin 'refs/tags/v2*'` (only rc's).

```sh
git tag -a v2.0.0 <tag commit from task 1> -m "v2.0.0 — v2.0 GA (owner-facing)"
git push origin v2.0.0
gh release create v2.0.0 --verify-tag --latest --title "v2.0.0 — v2.0 GA (owner-facing)" --notes-file <scratchpad>/v2.0.0-notes.md
```

The notes file is ga-01's GA PR body (the behaviour-change list), plus:
- "rollback: R-b to `<2.0.0` returns `<last-1.x>`; no contract";
- the widening PR link.

Watch `gh run watch` on `deploy.yml`:
- the `release-line` job: `v2.0.0 is on release line 2`, and the **no-contract step green**;
- **8** image jobs push `2.0.0`, judge included.

Then Flux picks it up. The ImageRepositories scan (1 m), the 8 fleet policies resolve `2.0.0`, and the IUA writes **one** commit to `../infra` bumping `apps/xlearn-*.yaml`. The HelmReleases then upgrade, all in one IUA commit with no set order. That is fine: GA adds no subject, stream or consumer.

**Never move or re-push the tag.** A problem is fixed forward (Rollback below).

### 7 · Verify + GA smoke [X] (step 7)

**ADR-0034 §6, after the tag (by looking, D34):**
- `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz` → `"version":"v2.0.0"`.
- `ssh sujaykumar-vps 'k3s kubectl get deploy -n xlearn -o custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image,READY:.status.readyReplicas'` → **8** deployments on `:2.0.0`, all ready.
- `ssh sujaykumar-vps 'k3s kubectl get imagepolicy -n flux-system'`:
  - the 8 fleet policies show `2.0.0`;
  - **`xlearn-runner` and `xlearn-evalpack` are unchanged** (their own versions);
  - the runner's namespace `xlearn-runner` shows no restart.
- `ssh sujaykumar-vps 'k3s kubectl get helmreleases -A'`: every `xlearn-*` is Ready.
- **Smoke:** login, the dashboard, coach.

**GA smoke:**
- **Owner** (in a browser profile signed in as the owner that the session can drive, a before-launch item; without it, record this leg ⛔ as an owner follow-up in `status.md`, lean on ga-01's compose proof, and carry on):
  - a packed DSA item → Run → Submit → `auto · checked`, with provenance on Week and Progress;
  - an AI suggestion on a newly concluded attempt (consents on), or the allowance meter reading;
  - go-concurrency in the catalog **without** the `preview` badge, and a gc item opens.
- **Anonymous** (no cookie): `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/u/<owner-username>` (the gateway's public route `GET /api/u/{username}`, `internal/gateway/bff.go`) includes the go-concurrency row if the owner's gc visibility is on, and omits it if it is off. The SPA page `/xlearn/u/<owner-username>` matches the JSON. Don't curl `/public/stats`: it is assessment's internal route, accepted only with the gateway-minted `public-read` token ([m2-03](sprint-m2-03.md) task 1), and the public API answers `not_found` for it whatever the toggle. ga-01's test matrices cover it directly.
- **The gate stays closed:**
  - `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/auth/config` → `{"signup":"closed"}`; the auth page shows the invite-only state;
  - `… identity admin seats` → 0 active learners out of 15, **0 outstanding invites**;
  - `… account list --role learner --status active` → empty;
  - **no invite minted**.
- **Optional, only if the owner's launch message asks for it** (and names a tester whose signed-in browser profile the session can drive). This is the only way to see the flip as a non-cohort account on production:
  1. `… identity admin account set-role <a tester> learner`. It briefly takes a seat and is audited.
  2. The tester sees Run/Submit on a packed item and go-concurrency in the catalog.
  3. `… set-role <that account> tester` within the sitting.
  4. Record it: counts and roles only, no identity.
- **Host after the rollout:** `ssh sujaykumar-vps 'bash /root/host-verify.sh --cluster --with-runner --nats-stage=n4'` is green. The memory sum is unchanged, with no restarts beyond the rollout.

If anything fails, go to Rollback. **R-b is the GA rollback.**

### 8 · Env clean-up [I] (conditional)

Only if ga-01's record (status.md flag inventory; [ga-01](sprint-ga-01.md) task 4) lists an env line in `../infra` for a flag it removed. None is expected, because the cohort gates were code constants. If there is none, set the task ✅ with "n/a: none recorded" and go on.

Otherwise, **after task 7 is fully green** (never before or with the tag):
- In `../infra`, from an up-to-date `main`, branch `chore/xlearn-drop-removed-flag-env`. Remove exactly the recorded `env` entries from the named `apps/xlearn-<svc>.yaml` HelmReleases. **Don't touch** the image tag lines (the IUA owns them), the kill switches (`JUDGE_BASE_URL`, `LLM_PLATFORM_ENABLED`, the grading override, `REVISION_ENTRY_RULE`, `SIGNUP_MODE`) or `COURSE_STATUS_OVERRIDE` if ga-01 kept it.
- Commit `chore(xlearn): drop env for flags removed in v2.0.0`, with the attribution lines. PR body: the flag names, "removed in GA PR #N, shipped in `v2.0.0`", and "a values change restarts the named services".
- Squash-merge (`../infra` has no CI: the PR body's flag list is its check). No `kubectl apply`: Flux reconciles `apps`.
- Verify by looking: `ssh sujaykumar-vps 'k3s kubectl get helmreleases -A'` all Ready; healthz still `"version":"v2.0.0"`; the restarted pods Ready on `:2.0.0`.

It is its own PR, never folded into the tag or the widening.

### 9 · Record + end the freeze + land and sync [X]

In [`../status.md`](../status.md):
- **Milestones:** **GA ✅** <date>, `v2.0.0`: judge and platform AI on and go-concurrency `active` for every account; `SIGNUP_MODE=closed`; no dogfood gate.
- **Milestone → tag → floor → snapshot:** `v2.0 GA → v2.0.0 → floor unchanged (<copy the current floor; expected 1.13.0>). GA has no contract; R-b to <2.0.0 returns <last-1.x> → snapshot <time/id>, taken with the fleet at <last-1.x>`.
- **Gate state after:** the kill switches stay (`JUDGE_BASE_URL`, the grading override, `LLM_PLATFORM_ENABLED`, `REVISION_ENTRY_RULE`, `SIGNUP_MODE=closed`, plus `COURSE_STATUS_OVERRIDE` if ga-01 kept it).
- **Release line and ranges:** `.release-line` = 2. The 8 fleet policies are `>=1.0.0 <3.0.0` (infra PR #). `xlearn-runner` and `xlearn-evalpack` are `>=1.0.0 <2.0.0`. From `v2.0.0` on, **the minor moves only at a GA flip**: `v2.1.0` is reserved for the interviewer, and everything else is a `v2.0.x` patch.
- **Flag inventory:** the flags ga-01 removed are now shipped ("gone in `v2.0.0`"), plus task 8's env PR # if one was needed.
- **Owner events:** `ev-ga` ✅ (the weekly-image date, the snapshot time).
- **CLI-use log:** each `identity admin` call.
- **Accepted risks** unchanged (D12, D34).
- **Decisions log:** "`v2.0.0` cut <date>, on the rc'd commit <sha>; pre-flip green; widening PR #; the snapshot preceded the tag", plus any deviation (task 2's "cut above a stray").

Then:
- **End the freeze** (SendMessage to peers): "`v2.0.0` verified. Tags resume, and the next is a `v2.0.x` patch."
- **Land and sync:** a docs-only PR with this record → squash-merge. Then `git checkout main && git pull` in **both** `xlearn` and `../infra`. If `main` is checked out in a clean peer worktree, `git -C <worktree> merge --ff-only origin/main`.

## Acceptance criteria

- [ ] The pre-flip check passed immediately before the widening: no stable `2.*` git tag, no stable 2.x GHCR tag on the 8 fleet packages, and no contract since `<last-1.x>`.
- [ ] The widening PR changed **exactly 8** fleet ranges to `>=1.0.0 <3.0.0` and **merged before the tag**. Nothing moved (latest still `<last-1.x>`, no IUA commit, no restarts). Runner and evalpack are untouched.
- [ ] `host-verify --cluster --with-runner --nats-stage=n4` was green on a settled host. The weekly image was ≤ 7 days old. The manual snapshot was taken before launch (D40), the same day as the tag, and its time is recorded.
- [ ] `v2.0.0` is tagged on the rehearsed commit (or a docs-only descendant), and `deploy.yml` was green, including the no-contract step.
- [ ] Verified by looking:
  - healthz `v2.0.0`;
  - 8 deployments on `2.0.0`;
  - 8 fleet ImagePolicies at `2.0.0`, runner and evalpack unchanged;
  - HelmReleases Ready;
  - smoke (login, dashboard, coach) plus the GA smoke.
- [ ] **v2.0 GA:** judge and platform AI on and the pilot `active` for every account (owner and testers); `SIGNUP_MODE` `closed`; no invite minted; no dogfood gate.
- [ ] Recorded in status.md (milestone → tag → floor → snapshot, ranges, release line, flags, `ev-ga`). The freeze is lifted, and `main` is synced in both repos.

## Release

**Tag `v2.0.0`** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): "(`v2.0.0-rc.N`) → **v2.0.0** · GA (§1.4): the default flip for the owner; no learner opening · kill switches stay · GA has no contract; R-b to `<2.0.0` returns to the last 1.x"). Title **`v2.0.0 — v2.0 GA (owner-facing)`**.

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:**
- **Version:** `v2.0.0`, the GA label. Major 2 = `.release-line`, set by ga-01.
- **ACL PRs:** none. There is no new stream, consumer or subject.
- **New service:** none. judge's image and policy have existed since `v1.13.0`.
- **Contract:** none. The `deploy.yml` step (it fires for the major's first stable release, so also for a "cut above a stray" tag once the stray git tag is deleted) and the pre-flip check both assert it. The floor is unchanged.
- **GA tag:** **yes.** `host-verify --cluster` green, the host settled, the snapshot taken (tasks 4–5; the snapshot by the owner before launch, D40).
- **Range change:** a **superset widening**, merged **first**, then the tag (ADR-0034 §1.4 table).
- **M6:** n/a. There are no interviews yet.
- **New in-cluster callers:** none. The flip removes role checks and adds no call path. No NetworkPolicy PR is needed, and the memory sum is unchanged ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- **"Every `xlearn-*` ImagePolicy's latest equals the tag"** is read for the **8 fleet** policies. `xlearn-runner` and `xlearn-evalpack` keep their own versions.
- **Infra PRs:** the widening is its own task (3), merged before the tag and **never folded into it**. Env removals for flags, if any, are a separate PR after the verify (task 8).

## Rollback

[ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first), fastest first. The session picks the fastest mechanism that fixes the failure (approved by the launch, D40) and records it; an owner "hold" in the session still overrides.
- **R-a (1–2 min), per capability, one infra PR each:**
  - judge off for everyone: unset `JUDGE_BASE_URL` on the gateway **and** practice. That gives the self path, and nothing is re-graded;
  - platform AI off: `LLM_PLATFORM_ENABLED=false` on judge. That gives manual entry;
  - the pilot hidden again: `COURSE_STATUS_OVERRIDE=go-concurrency=preview` on every `xlearn-*` HelmRelease (if ga-01 kept it).
- **R-b (the GA rollback, 2–3 min, no build):** one infra PR narrowing the **8 fleet** policies back to `>=1.0.0 <2.0.0`. The merge *is* the rollback. The IUA writes `<last-1.x>` back, and the 1.x images run on the 2.0.0 schema, as ga-01's phase 2 proved. **Undo it after the fix, tag first:**
  1. **tag `v2.0.1`** with the fix. It builds, and it can't deploy while the ranges are `<2.0.0`;
  2. **then re-widen** the 8 fleet policies to `>=1.0.0 <3.0.0`. Flux's first 2.x pick is `2.0.1`, the highest in range, and the widening merge is the deploy moment.

  **Never re-widen first.** The bad `2.0.0` images are still in GHCR, so a widening before `v2.0.1` exists redeploys `2.0.0` at the merge: the stray hazard of [ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) step 3. The ADR's "superset widening: merge first, then tag" holds only while **no stable 2.x exists**; after GA it no longer does, and the pre-flip check would rightly fail here. Alternative: make R-b's range `>=1.0.0 <3.0.0, !=2.0.0` instead. It also returns `<last-1.x>`, and `v2.0.1` then deploys directly on its tag; dropping the exclusion afterwards is optional. Take the plain narrowing by default, and record which one was used in the decisions log.
- **R-c (6–8 min):** revert on `main` + `v2.0.1`. The ranges are already `<3.0.0`, so nothing else changes.
- **R-d (the snapshot, within ~1 day only):** the [§4.2 procedure](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button). The restore (step 4) is the owner's hPanel action, so the session does steps 1–3, records ⛔ "R-d restore needed (owner)" with steps 4–6 in `status.md`, and doesn't wait:
  1. pin git back first (R-b);
  2. confirm the `xlearn-*` tags equal the snapshot's (`<last-1.x>`);
  3. list the erases since the snapshot (`identity admin erasures --since <snapshot time>`; none expected at GA);
  4. restore;
  5. run `host-verify --cluster`;
  6. re-run the erases, and record the lost-writes window.

## Definition of Done

- Pre-flip green.
- The widening PR merged before the tag, and nothing moved.
- `host-verify` green; the owner's before-launch snapshot recorded.
- `v2.0.0` shipped by Flux (no hand `kubectl`) and verified by the checklist and the GA smoke.
- `SIGNUP_MODE` still `closed`, with no invite.
- Statuses updated: this file, and [`../status.md`](../status.md) (board, **GA ✅**, tag → floor → snapshot, ranges and release line, flags, `ev-ga`, the CLI-use log, decisions).
- Task 8's env PR merged after the verify, or recorded n/a.
- The peers told the freeze is over.
- The docs PR merged, and `main` synced in `xlearn` and `../infra`.

## Risks / watch-outs

- **Getting the order wrong deploys a stray or stalls.** Follow the checklist verbatim:
  - widening after the tag → the tag waits on an infra merge, and the deploy moment drifts away from the snapshot;
  - widening before the pre-flip check → a stray deploys at the merge.
- **A lenient-semver stray** (`2`, `2.0`, `v2.0.0` as a GHCR tag). Flux parses it as stable 2.x. The script's regex covers these, so don't hand-roll a narrower check.
- **A global `sed` on `<2.0.0`** would also widen `xlearn-runner` and `xlearn-evalpack`. A future runner-v2 or pack-v2 image would then deploy without its GA-in-miniature steps. Edit the 8 documents by name and check the 8-row diff.
- **A peer tags during the window.** With `.release-line = 2` and the ranges widened, a peer's `v2.0.1` would build **and deploy**. Hold the freeze from task 1 until task 9's message.
- **The snapshot expires in ~1 day.** The owner launches right after taking it, and the tag lands the same day. If the tag slips past it, narrow back and record ⛔ (task 5).
- **The flip is invisible on production** (all accounts are in the cohort). Rely on ga-01's compose proof, and optionally the tester-as-learner check (task 7). Don't mint a learner or an invite for it.
- **A re-pushed tag doesn't roll.** GHCR is overwritten, and Flux doesn't redeploy the same tag string. Never move `v2.0.0`; fix forward with `v2.0.1`.
- **Public visibility of the pilot.** The owner's gc rows become public if visible. He sets his choice before launch.
- **No alerting (D34).** A failed rollout is seen only by task 7's reads, so don't end the sitting before they're all green.
