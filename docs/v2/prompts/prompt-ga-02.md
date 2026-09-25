# Prompt — Sprint ga-02 · Cut v2.0.0: pre-flip check, widen ranges, snapshot, tag, verify

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root **on GA day, right after the owner's snapshot** (below).
> **Plan:** [`../sprints/sprint-ga-02.md`](../sprints/sprint-ga-02.md)   ·   **Milestone:** GA (part 2 of 2)   ·   **Prereqs:** [ga-01](../sprints/sprint-ga-01.md) (GA PR merged; `v2.0.0-rc.N` rehearsed; no active learner)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] The last Hostinger weekly image is ≤ 7 days old (hPanel). If it's older, wait for the next one before launching.
- [ ] The host has settled: no host change, reboot or restart-inducing infra PR in the last 24 h (`status.md`).
- [ ] **`ev-ga`:** take a Hostinger manual snapshot in hPanel (one at a time, 1-day retention) and launch this prompt right after it: the tag must land the same day. Put the snapshot's time or id and the weekly-image date in your launch message.
- [ ] Your go-concurrency public visibility (AB22) is set as you want it: after the tag, the pilot's rows are public when visible.
- [ ] For the GA smoke's owner leg: a browser profile signed in to xLearn as you, that the session can drive. Optional: without it, that leg is recorded as your follow-up.
- [ ] Optional: say in your launch message if you want the tester-as-learner check (plan task 7), and have that tester's signed-in browser profile available to the session.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, land-and-sync.
- **The plan:** [`../sprints/sprint-ga-02.md`](../sprints/sprint-ga-02.md). It holds:
  - the verbatim GA checklist and release checklist;
  - the stray-tag branches;
  - the widening diff rule;
  - the GA smoke;
  - the rollback.
- **[ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):**
  - [**§1.4**](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) (the range-change table: a superset widening merges **first**, then the tag; GA steps 3–7; "Rollback of GA");
  - [§1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) (runner and evalpack are separate streams);
  - [§1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline) (the `v2.0.0` row);
  - [§4.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first) / [§4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button) / [§4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (rollback, the R-d procedure, the snapshot rule);
  - [**§6**](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) (the release checklist).
- [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34) (the `host-verify --cluster` checks; D34) and [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (the memory sum).
- [rollout §4 "v2.0 GA"](../rollout-plan.md#v20-ga-the-owner-facing-default-flip) (the GA checklist), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (the range-order rule, snapshots, "never move a tag"), [§6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) (GA day), [§11](../rollout-plan.md#11-opening-gates-v3) (what stays for v3).
- [`../../git-strategy.md`](../../git-strategy.md) (the GA procedure as ga-01 left it).
- ga-01's record in [`../status.md`](../status.md):
  - the GA PR #;
  - the rc'd SHA;
  - `<last-1.x>`;
  - the flag decisions;
  - the current rollback floor.
- **Code and infra:**
  - `hack/ga-preflip-check.sh` (flags `--major`, `--cluster`) and `hack/lint-migrations.sh --no-contract-since`, both written in ga-01;
  - `.github/workflows/deploy.yml`;
  - `../infra/apps/image-automation.yaml` (8 fleet ImagePolicies + `xlearn-runner` + `xlearn-evalpack`, **same range string**);
  - `../infra/hack/host-verify.sh` (run as `/root/host-verify.sh` on the node).

## Context

ga-01 merged the **GA PR** (on CI green, D40): `.release-line = 2`, and judge, platform AI and go-concurrency `active` for every account as code defaults. It rehearsed **`v2.0.0-rc.N`** in compose, including R-b to the last 1.x, and cleared the strangers. Production still runs `<last-1.x>`: `main` is build-only, and the fleet ImagePolicies are `>=1.0.0 <2.0.0`.

This session executes **ADR-0034 §1.4 steps 3–7 in order**:
- **3.** pre-flip check (no stable 2.x anywhere);
- **4.** the **superset widening** of the 8 fleet policies to `<3.0.0`, **merged before the tag** (nothing moves);
- **5.** `host-verify --cluster` + the owner's **manual snapshot** (taken before launch, D40);
- **6.** **tag `v2.0.0`** on the rehearsed commit;
- **7.** verify by looking, then record.

**Getting the order wrong deploys a stray or stalls the deploy**, so follow the checklist verbatim. `SIGNUP_MODE` stays `closed`, no invite is minted, and the opening is v3.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **ga-01 ✅:**
  - `git show origin/main:.release-line` → `2`;
  - the GA PR # is merged;
  - `v2.0.0-rc.N` is a GitHub prerelease, and its four-phase rehearsal is green;
  - the **rc'd SHA** is in status.md;
  - `hack/ga-preflip-check.sh` and its self-test are green on `main`.
- [ ] **No active learner:** `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list --role learner --status active'` → empty. Terminal only (PII).
- [ ] **Production settled and healthy:**
  - no host change, reboot or restart-inducing infra PR in the last 24 h (status.md; `git -C ../infra log --since=24.hours origin/main`);
  - `ssh vps 'k3s kubectl get kustomizations,helmreleases -A'` is all Ready.
- [ ] **The launch message carries** the snapshot's time or id and the weekly-image date (≤ 7 days). The owner's go-concurrency visibility is set (a before-launch item).
- [ ] **Parallel sessions:**
  - no peer tag or open PR claims `v2.0.0` or edits `../infra/apps/image-automation.yaml`;
  - no peer plans a tag (`git ls-remote --tags origin`, `gh pr list --state open` in both repos, `git worktree list`, ListAgents).

## Do this (in order)

1. **[X] Pre-flight** (plan task 1):
   - **Freeze.** SendMessage every active peer: "v2.0 GA in progress: no tags and no merges to xlearn `main` or `../infra/apps/image-automation.yaml` until I report `v2.0.0` verified."
   - **Find the tag commit.** `git fetch --tags origin`; `git merge-base --is-ancestor <rc sha> origin/main`; then `git diff --stat <rc sha> origin/main -- . ':!docs' ':!*.md' ':!design-system'` must be **empty** (`.github/` is not exempt). **One rule:** if it is empty, the tag commit is `origin/main`. If it isn't, stop and never tag the old rc: cut `-rc.N+1` from `origin/main`, redo ga-01 task 10's phases 1–3 on it, record its SHA, and re-run this check against it.
   - **Check the line and the contract rule.** `.release-line` at the tag commit = `2`, and `hack/lint-migrations.sh --no-contract-since <last-1.x>` is green.
2. **[X] Pre-flip check** (plan task 2; ADR-0034 §1.4 step 3). Run `hack/ga-preflip-check.sh --major 2 --cluster` at the tag commit. **All PASS:**
   - no stable `v2*` git tag (rc's are OK);
   - no stable 2.x GHCR tag (lenient forms included) on the **8** fleet packages;
   - no contract;
   - the live table: 8 fleet `>=1.0.0 <2.0.0` at `<last-1.x>`; runner and evalpack at their own versions.

   Save the table for the PR. **If a stray exists**, take the ADR's two branches in this order (D40: your call, approved by the launch):
   - **delete it:** `git push origin :refs/tags/<tag>` and `gh api -X DELETE …/packages/container/xlearn-<svc>/versions/<id>`. **Irreversible**, and approved by the launch. The GHCR delete needs a token with `delete:packages`; without one, take the next branch;
   - **cut GA above it:** tag a version above the stray, **before widening**. ADR-0034 §1.4 steps 4 and 6 swap (this prompt's steps 3 and 6), so the order is check → `host-verify` → tag → widen → verify (the snapshot was taken before launch). Delete the stray **git** tag anyway (only a GHCR version can resist deletion), so `deploy.yml`'s no-contract step still sees the chosen tag as the major's first stable release; the step 1 lint at the chosen commit is the second guard. Record the deviation.
3. **[I] Widen** (plan task 3; step 4). In `../infra`, from a fresh `main`, branch `chore/xlearn-fleet-ranges-v2-ga`:
   - In `apps/image-automation.yaml`, set `range: ">=1.0.0 <3.0.0"` on **exactly** `xlearn-{gateway,identity,curriculum,practice,review,assessment,coach,judge}`. **Not `xlearn-runner` or `xlearn-evalpack`, and no global `sed`.**
   - Update the header comment ("`<3.0.0` since the v2.0 GA; runner and evalpack are separate streams at `<2.0.0`").
   - Check the `yq` before/after `name → range` table: **exactly 8 rows differ**.
   - Commit `chore(xlearn): widen fleet ImagePolicies to <3.0.0 for the v2.0 GA (ADR-0034 §1.4)` with the attribution lines. The PR body carries the pre-flip output and "merge first, then tag; nothing moves".
   - **Re-run step 2 if more than ~10 min have passed**, then squash-merge. `../infra` has no CI: the `yq` table and the pre-flip output in the PR body are its checks.
   - **Verify nothing moved** (read-only):
     - `apps` `lastAppliedRevision` = the merge SHA;
     - `hack/ga-preflip-check.sh --major 2 --cluster` shows 8 × `<3.0.0` still at `<last-1.x>`, with runner and evalpack unchanged;
     - no `chore(images)` commit in `../infra`;
     - no pod restarts in `xlearn`.

     If anything moved, **stop**: revert the PR (R-b) and investigate.
4. **[H] `host-verify`** (plan task 4). `ssh vps 'bash /root/host-verify.sh --cluster --with-runner --nats-stage=n4'` → no FAIL: the memory sum with the runner counted, Flux, pods, CNPG, PVCs/disk, NATS `auth_required`, NetworkPolicies, steal. Save the summary.
5. **[O] Snapshot** (plan task 5, `ev-ga`) — **taken before launch**:
   - Record the snapshot's time or id and the **last weekly image date** from the launch message.
   - Go straight on to the tag: it must land the same day (the snapshot has a ~1-day life). If it can't, don't tag on a lapsed snapshot: revert step 3's PR, end the freeze, and record ⛔ "snapshot lapsed; re-take it and relaunch".
6. **[X] Tag `v2.0.0`** (plan task 6; step 6).
   - Run the §Release checklist from the plan. **This is an irreversible step, so check again first:**
     - peer and owner messages;
     - `git ls-remote --tags origin 'refs/tags/v2*'` (rc's only);
     - `gh pr list`.
   - Tag and release:

     ```sh
     git tag -a v2.0.0 <tag commit> -m "v2.0.0 — v2.0 GA (owner-facing)"
     git push origin v2.0.0
     gh release create v2.0.0 --verify-tag --latest --title "v2.0.0 — v2.0 GA (owner-facing)" --notes-file <scratchpad>/v2.0.0-notes.md
     ```

     The notes are ga-01's GA PR body, plus "rollback: R-b to `<2.0.0` returns `<last-1.x>`; no contract" and the widening PR link.
   - `gh run watch`: `release-line` (line 2, **no-contract step green**) → 8 image jobs → the IUA's single `chore(images)` commit in `../infra` → the HelmReleases upgrade.
   - **Never move or re-push the tag.**
7. **[X] Verify + GA smoke** (plan task 7; step 7).
   - **The ADR-0034 §6 after-tag lines:**
     - healthz → `"version":"v2.0.0"`;
     - `ssh vps 'k3s kubectl get deploy -n xlearn -o custom-columns=…'` → 8 on `:2.0.0`, ready;
     - `get imagepolicy -n flux-system` → the 8 fleet at `2.0.0`, **runner and evalpack unchanged**; the runner namespace has no restarts;
     - `get helmreleases -A` → every `xlearn-*` Ready;
     - smoke: login, the dashboard, coach.
   - **The GA smoke:**
     - owner, in his signed-in browser profile (a before-launch item; without it, record this leg ⛔ as his follow-up and carry on): packed DSA Run → Submit → `auto · checked`; an AI suggestion or the allowance meter; go-concurrency without the `preview` badge;
     - anonymous (no cookie): `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/u/<owner>` (the gateway's only public route, `GET /api/u/{username}`) includes the gc row if his visibility is on and omits it if off; the SPA `/xlearn/u/<owner>` matches. **Not** `/public/stats`: that is assessment's internal `public-read` route, so the public API always answers `not_found` for it;
     - the gate: `auth/config` → `{"signup":"closed"}`; `identity admin seats` → 0/15 learners, **0 outstanding invites**; `account list --role learner --status active` → empty; **no invite minted**.
   - **Optional, only if the launch message asks for it:** a tester → `set-role learner`, check Run/Submit and gc as a non-cohort account in that tester's signed-in profile, then `set-role tester` back within the session. Record it with counts and roles only.
   - Post-rollout: `host-verify --cluster --with-runner --nats-stage=n4` green.
   - On any failure, go to the plan's **Rollback** and pick the fastest mechanism that fixes it yourself (approved by the launch, D40): R-a per capability; **R-b** (narrow the 8 fleet policies back to `<2.0.0`) is the GA rollback; R-c = `v2.0.1`; R-d only within the snapshot's day, in the §4.2 order — its restore is the owner's hPanel action, so do steps 1–3, record ⛔ "R-d restore needed (owner)" with steps 4–6, and don't wait.
   - **Undoing R-b is tag first:** tag `v2.0.1` with the fix (it builds and can't deploy under `<2.0.0`), **then** re-widen to `<3.0.0`. Re-widening first would redeploy the bad `2.0.0`, which is still in GHCR. The alternative is R-b as `>=1.0.0 <3.0.0, !=2.0.0`, after which `v2.0.1` deploys directly.
8. **[I] Env clean-up, conditional** (plan task 8). Only if ga-01's record lists an `../infra` env line for a removed flag (none expected; otherwise mark it ✅ "n/a: none recorded"). After step 7 is fully green: one infra PR, branch `chore/xlearn-drop-removed-flag-env`, removing exactly those `env` entries from the named `apps/xlearn-<svc>.yaml` HelmReleases (never a tag line, a kill switch or a kept `COURSE_STATUS_OVERRIDE`). Commit `chore(xlearn): drop env for flags removed in v2.0.0` with the attribution lines, squash-merge, then check HelmReleases Ready and healthz still `v2.0.0`.
9. **[X] Record, end the freeze, land and sync** (plan task 9).
   - Update status.md (below).
   - Tell the peers: "`v2.0.0` verified; tags resume; next is a `v2.0.x` patch".
   - Land a docs-only PR and sync (see Ship).

## Constraints

- **The ADR-0034 §1.4 order is law:** pre-flip → widen (merged **before** the tag) → `host-verify` + snapshot (the owner's, taken before launch, D40; the widening moves nothing, so it still holds the pre-GA state) → tag → verify. The only exceptions are the ADR's own "stray can't be deleted" branch, recorded as a deviation, and undoing an R-b (tag `v2.0.1` first, then re-widen): "merge first" holds only while no stable 2.x exists.
- **Superset widening of the 8 fleet policies only.** `xlearn-runner` and `xlearn-evalpack` stay `>=1.0.0 <2.0.0` (ADR-0034 §1.5; a major there is a contract break).
- **GitOps:**
  - the widening is **its own infra PR**, never folded into the tag;
  - never `kubectl apply`/`edit`, never edit an `apps/xlearn-*.yaml` tag line, never suspend the shared IUA;
  - `ssh vps` stays read-only except the sanctioned `identity admin` CLI (logged).
- **Tags:** `v2.0.0` exactly once, on the rehearsed commit (or a docs-only descendant). **Never move or re-push a tag.** From `v2.0.0` on, the minor moves only at a GA flip (`v2.1.0` = interviewer).
- **No contract:** asserted by `deploy.yml` and the pre-flip check. The floor is unchanged. **No migration or code change in this sprint.**
- **No ACL PR, no NetworkPolicy PR** (no new caller); **memory-sum** unchanged (no new pod).
- **D34:** no alerting, opscheck, healthchecks.io, Flux Alert or push channel. Everything is verified by looking.
- **Snapshot:** the owner's, taken before launch on a settled host; the tag lands the same day (1-day life). No off-node `pg_dump` (D12).
- **Parallel sessions:** the freeze message at the start, a re-check before the widening merge and before the tag push, and the release message at the end.
- **Privacy:** no stranger, tester or learner identity in any committed file or PR. The GHCR/git outputs are fine.
- **`SIGNUP_MODE` stays `closed`; no invite; no `SEAT_CAP` or AI-limit change** (v3).

## Deliverables

- The pre-flip check output (PASS), pasted into the widening PR.
- `../infra` PR: 8 fleet ranges → `>=1.0.0 <3.0.0`, merged before the tag, nothing moved.
- `host-verify --cluster` green; the owner's before-launch manual snapshot (time recorded).
- Tag `v2.0.0` + GitHub release "`v2.0.0 — v2.0 GA (owner-facing)`", deployed by Flux and verified (ADR-0034 §6 + the GA smoke).
- If ga-01 recorded any: the `../infra` env clean-up PR, merged after the verify.
- A docs PR with the GA record; peers released from the freeze; both repos synced.

## Update status

- **The plan's Status table** ([`../sprints/sprint-ga-02.md`](../sprints/sprint-ga-02.md)): tasks 🔄 → ✅; _Overall_ ✅ after task 9.
- **[`../status.md`](../status.md):**
  - the Sprint board row;
  - **Milestones: GA ✅** <date>, `v2.0.0` (judge and platform AI on and gc `active` for every account; `SIGNUP_MODE=closed`; no dogfood gate);
  - **milestone → tag → floor → snapshot:** `v2.0 GA → v2.0.0 → floor unchanged (<current>); R-b to <2.0.0 returns <last-1.x> → snapshot <time/id>`;
  - **gate state:** kill switches stay;
  - **release line and ranges:** `.release-line` 2; 8 fleet `<3.0.0` (infra PR #); runner and evalpack `<2.0.0`; "from `v2.0.0` the minor moves only at a GA flip";
  - the **flag inventory:** ga-01's removals shipped;
  - **owner events:** `ev-ga` ✅;
  - the **CLI-use log**;
  - accepted risks unchanged (D12, D34).
- **Decisions log:** "`v2.0.0` cut <date> on <sha>; pre-flip green; widening PR #; snapshot before the tag", plus any deviation.
- **ADRs:** none expected. If a stray forced "cut GA above it", note it in the decisions log and in ADR-0034 §1.4 as a dated line.

## Done when (acceptance)

- [ ] Pre-flip PASS immediately before the widening (git, GHCR on the 8 fleet packages, no contract).
- [ ] The widening changed **exactly 8** ranges, **merged before the tag**, and nothing moved. Runner and evalpack are untouched.
- [ ] `host-verify --cluster --with-runner --nats-stage=n4` green on a settled host; weekly image ≤ 7 days; the owner's before-launch snapshot recorded, the same day as the tag.
- [ ] `v2.0.0` is tagged on the rehearsed commit, and `deploy.yml` is green (no-contract step included).
- [ ] Verified:
  - healthz `v2.0.0`;
  - 8 deployments on `2.0.0`;
  - 8 fleet ImagePolicies at `2.0.0`, runner and evalpack unchanged;
  - HelmReleases Ready;
  - smoke + the GA smoke.
- [ ] **v2.0 GA:** judge and platform AI on, pilot `active`, for every account (owner + testers); `SIGNUP_MODE` `closed`; no invite minted; no dogfood gate.
- [ ] Recorded in status.md; the freeze is lifted; `main` is synced in both repos.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched: the `../infra` widening PR first (step 3: its own PR, merged **before** the tag, never folded into it), then, after the verify, the conditional env clean-up PR (step 8) and the xlearn docs PR with the record (step 9).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. `../infra` has no CI: the before/after `name → range` table and the pre-flip output in the widening PR's body (and the flag list in the clean-up PR's) are its checks.
3. **Release action — tag `v2.0.0`:** walk the release checklist (ADR-0034 §6, in the plan) in the ADR-0034 §1.4 order: the pre-flip check → the widening merged → `host-verify --cluster` green → (the owner's snapshot, taken before launch) → push the tag `v2.0.0` on the rehearsed commit → let Flux deploy → verify live by looking (step 7, plus the GA smoke). Never move or re-push the tag.
4. Update status: the sprint file and `docs/v2/status.md` (step 9), in the docs PR merged the same way; then end the freeze.
5. Run `git checkout main && git pull` in xlearn and `../infra`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
