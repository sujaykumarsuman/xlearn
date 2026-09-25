# Sprint ga-01 — GA PR: .release-line = 2, T-1 default flips, rc rehearsal

> **Milestone:** GA — v2.0 GA, the owner-facing default flip (**part 1 of 2**: the GA PR, the strangers check and the `-rc` rehearsal; [ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) steps 1–2) · **Track:** product · **Order:** 69
> **Prereqs:** [l-04](sprint-l-04.md) (L exit: `ev-l-rehearsal` recorded, production back to `SIGNUP_MODE=closed`, `v1.17.0`) · [p-03](sprint-p-03.md) (P exit, `v1.15.0`, the pilot `preview`) · [m4-07](sprint-m4-07.md) (M4 exit, `v1.16.0`, `LLM_PLATFORM_ENABLED` on for the cohort) · [m3-13](sprint-m3-13.md) (M3 exit, `v1.14.0`, `JUDGE_BASE_URL` set) · [mi-11](sprint-mi-11.md) (Track B finish, N4)
> **Unblocks:** [ga-02](sprint-ga-02.md) (pre-flip check, widening, snapshot, the `v2.0.0` tag)
> **Release action:** **merge only.** The GA PR **merges on CI green** (D40): launching this prompt is the owner's approval of the major bump that [ADR-0034 §1.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#13-accidental-major-guard-shipped-2026-09-24-xlearn53-infra29) makes a deliberate, reviewed PR, so there is no separate approval stop. Merging deploys nothing, because `main` is build-only. It comes **with a `v2.0.0-rc.1` prerelease tag** (§1.4 step 2): the eight fleet images build and **never deploy**, since Flux semver ranges skip prereleases. No infra PR, no range change and no `v2.0.0` here; those are [ga-02](sprint-ga-02.md).
> **Calendar:** ≈ late December 2026 – January 2027, after `ev-l-rehearsal`. Owner time is before launch only and optional (~15 min): erasing or re-roling any stranger he doesn't want suspended, and noting the last weekly-image date. The session runs the rest (`ev-strangers`, the GA PR, the rc and the rehearsal) without a review stop.
> **Execute with:** [`../prompts/prompt-ga-01.md`](../prompts/prompt-ga-01.md). One prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Entry read: milestones ✅, the M3 checklist re-read, `host-verify --cluster` green | H | ⬜ |
| 2 | Strangers: no `active` account with role `learner` (`ev-strangers`); the session suspends the rest (the owner's optional erase / re-role is before launch) | H | ⬜ |
| 3 | GA PR · `.release-line` → `2` and the T-1 default flips (judge, platform AI, go-concurrency `active`) + the test matrices | X | ⬜ |
| 4 | GA PR · flag inventory: remove the GA-removal flags; decide `COURSE_STATUS_OVERRIDE` | X | ⬜ |
| 5 | GA PR · no-contract assertion: `hack/lint-migrations.sh --no-contract-since` + a `deploy.yml` step for a major's first stable release (and its rc's) | X | ⬜ |
| 6 | GA PR · `hack/ga-preflip-check.sh` + self-test (run by ga-02) | X | ⬜ |
| 7 | GA PR · release notes (behaviour-change list) + git-strategy / ADR-0034 drift check | X | ⬜ |
| 8 | Merge the GA PR on CI green (D40; no separate approval) | X | ⬜ |
| 9 | Tag `v2.0.0-rc.1` (prerelease: builds, never deploys) | X | ⬜ |
| 10 | Compose rehearsal of the rc: forward, flip checks, R-b to the last 1.x, roll-forward; pre-flip dry run | X | ⬜ |
| 11 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md):
> - the Sprint board row;
> - the **GA milestone** (🔄 here, ✅ only in ga-02);
> - the flag inventory;
> - owner events (`ev-strangers`);
> - the CLI-use log;
> - the decisions log.
>
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

**GA checklist** ([rollout §4](../rollout-plan.md#v20-ga-the-owner-facing-default-flip), verbatim, in ADR-0034 §1.4 order). Who delivers what:
- **This sprint:** the second line (the strangers, task 2), item **1** (tasks 3–8) and item **2** (tasks 9–10).
- **Here and again in ga-02:** the third line, asserted here (task 5) and re-asserted in ga-02.
- **[ga-02](sprint-ga-02.md):** items **3–7** and the last line.
- **At entry, only the first line must be green** (task 1 re-reads it).

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

*Reading note (not part of the verbatim list):* "every `xlearn-*`" in items 3, 4 and 7 means the **8 fleet images** that `deploy.yml` builds: gateway, identity, curriculum, practice, review, assessment, coach and judge. `xlearn-runner` and `xlearn-evalpack` are **separate release streams**. They keep `>=1.0.0 <2.0.0`, and a major there is a contract break ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams); D32). Task 7 makes the docs say so.

**M3 hard entry checklist** ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist), verbatim). The GA checklist's first line needs it "still green", so task 1 re-reads every item:

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

*Key (not part of the verbatim list):* rollout step ids `MI-NN` are **not** sprint ids `mi-NN`.

| Rollout item | Sprint or event |
|---|---|
| MI-4 | [mi-14](sprint-mi-14.md) |
| MI-5, MI-5a | [mi-03](sprint-mi-03.md) |
| MI-7 (N3) | [mi-06](sprint-mi-06.md) + the ≥ 24 h re-check in [l-01](sprint-l-01.md) |
| MI-8 | [mi-02](sprint-mi-02.md) |
| MI-9 | [mi-07](sprint-mi-07.md) + the evalpack ImagePolicy in [m3-07](sprint-m3-07.md) |
| MI-10 | [spk-01](sprint-spk-01.md) + [spk-02](sprint-spk-02.md) |
| MI-11 | [mi-09](sprint-mi-09.md) + the 2026-10-24 host window |
| MI-11a | [mi-08](sprint-mi-08.md) |
| MI-12 | [mi-10](sprint-mi-10.md) |
| MI-13 | [m3-07](sprint-m3-07.md) |
| T25/T26 | [m3-01](sprint-m3-01.md) |
| 14 packs | the owner's pack event (prepared by [m3-02](sprint-m3-02.md)) |
| `account.role` | [m1-02](sprint-m1-02.md) |
| AB07–AB12 | the [ds-m3-01](sprint-ds-m3-01.md) + [ds-m3-02](sprint-ds-m3-02.md) freezes |

At GA, N4 has run ([mi-11](sprint-mi-11.md)), so the MI-8 NATS item is read at `--nats-stage=n4` (`auth_required: true`).

**Sprint gates:**

- [ ] **MI complete for GA:** every MI step is ✅ in status.md **except MI-16**, which is an M6b gate ([mi-13](sprint-mi-13.md)). In particular:
  - MI-5b live ([mi-04](sprint-mi-04.md));
  - MI-14 ([mi-12](sprint-mi-12.md));
  - MI-15 finished ([mi-08](sprint-mi-08.md) + [mi-11](sprint-mi-11.md): the `xlearn` egress policies, PG `connectionLimit`, Renovate, N4).
- [ ] **M1–M4 + P + L complete:** the Milestones table shows each ✅ with its exit recorded:
  - M1 `v1.8.0`;
  - M2 `v1.10.0`;
  - M3 `v1.14.0` (kill switch tested, denylist green);
  - P `v1.15.0` (manifest `preview`);
  - M4 `v1.16.0` (ledger within ±5% of the Console);
  - L `v1.17.0` plus the rehearsal record.
- [ ] **L exit recorded:** `ev-l-rehearsal` done and `SIGNUP_MODE=closed` live again ([l-04](sprint-l-04.md)). `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/auth/config` → `{"signup":"closed"}`.
- [ ] **The T-2 switches the flip relies on are live.** Otherwise the T-1 flip turns nothing on:
  - `JUDGE_BASE_URL` is set on the gateway and practice ([m3-13](sprint-m3-13.md));
  - `LLM_PLATFORM_ENABLED=true` is set on judge ([m4-07](sprint-m4-07.md)).

  Check the HelmRelease env in `../infra/apps/xlearn-{gateway,practice,judge}.yaml`.
- [ ] **ADRs:** [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (accepted at the sandbox-spike GO) and [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) (accepted in [mi-12](sprint-mi-12.md)) are Accepted. ADR-0032 may still be Proposed; [ds-m6a-01](sprint-ds-m6a-01.md) accepts it.
- [ ] **No contract since the last 1.x tag, so far.** Call the last stable 1.x tag `<last-1.x>` (expected `v1.17.0`):

  ```sh
  git tag -l 'v1.*' --sort=-v:refname | grep -v -- - | head -1
  ```

  No migration added since it carries `-- xlearn:contract`:

  ```sh
  git diff --name-only --diff-filter=AM <last-1.x>..origin/main -- 'internal/*/store/migrations/*.sql' | xargs -r grep -l 'xlearn:contract'
  ```

  The output must be empty. Task 5 makes this check mechanical.
- [ ] **The flag inventory** in status.md names every T-3 cohort gate whose removal milestone is GA. At least:
  - m3-09's `judgeCohortOnly`;
  - [m4-02](sprint-m4-02.md)'s `platformAICohortOnly` (the "platform AI cohort-only" row), which also covers the cohort terms in [m4-04](sprint-m4-04.md)'s dispute and [m4-05](sprint-m4-05.md)'s allowance presence checks.

  It also carries [p-02](sprint-p-02.md)'s `COURSE_STATUS_OVERRIDE` row and its recommendation.
- [ ] **Parallel sessions:**
  - no open peer PR edits `.release-line`, `.github/workflows/deploy.yml`, `hack/lint-migrations.sh`, `internal/gateway/`, `internal/judge/`, `curriculum/courses/go-concurrency/course.json` or `docs/git-strategy.md`;
  - no peer plans a tag from `main`.

  Check with `gh pr list --state open`, `git ls-remote --tags origin`, `git worktree list` and ListAgents. Tell the peers now: **from the GA PR's merge, `main` is on the 2 line**, and a 1.x hotfix branches from `<last-1.x>`.

## Goal

Prepare the **owner-facing v2.0 GA** ([D32](../feasibility.md#decisions-log-newest-first), [D35](../feasibility.md#decisions-log-newest-first); [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme), [§1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure)) as **one deliberate xlearn PR**, merged on CI green (D40):
- **`.release-line = 2`**;
- the **T-1 default flips**. Judge features and platform AI are on for **every** account; until now only the owner and tester cohort (T-3) had them. go-concurrency goes from `preview` to **`active`**;
- the flags whose removal milestone is GA are removed. The kill switches stay;
- two mechanical guards that ga-02 relies on: the **no-contract assertion** and the **pre-flip check script**;
- release notes that carry the **behaviour-change list**.

Before it merges, the sprint makes sure **no stranger from v1's open signup is still an `active` learner** ([ADR-0033 §2](../../adr/0033-invite-only-admission-and-owner-admin.md#2-close-v1-signup-now-v15x-stopgap-shipped-in-v152)). After it merges on CI green (D40), it tags **`v2.0.0-rc.1`**. The images build but never deploy. The sprint rehearses them in compose, **including the R-b rollback to the last 1.x**, so [ga-02](sprint-ga-02.md) can tag `v2.0.0` on exactly the rehearsed commit.

Nothing is visible on production after this sprint. In v2 the flip reaches the owner and testers only. `SIGNUP_MODE` stays `closed`, and the opening is v3 ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).

## Scope

**In**
- The strangers triage on production (the session suspends every remaining active learner, the plan's reversible default, D40; the owner's optional erase or re-role happens before launch; CLI via `kubectl exec`; counts only in the record).
- One xlearn GA PR (branch `feat/ga-v2-default-flip`):
  - `.release-line`;
  - the judge and platform-AI cohort gates removed from the code defaults;
  - `curriculum/courses/go-concurrency/course.json` → `active`;
  - the test matrices flipped, and `internal/e2e/ga_flip_test.go` + `internal/e2e/ga_rehearse_test.go` added;
  - the flag clean-up and the `COURSE_STATUS_OVERRIDE` decision;
  - the `hack/lint-migrations.sh --no-contract-since` mode, and a `deploy.yml` step that runs it for a major's first stable release and its rc's (`vN.0.0`, or the version above a stray that ga-02 may have to cut);
  - `hack/ga-preflip-check.sh` + a self-test in `ci.yml`;
  - the docs drift fixes (`docs/git-strategy.md`, the `deploy.yml` header, `docs/architecture/api.md`, and a dated clarification in ADR-0034 if needed);
  - the release notes (the PR body).
- The merge on CI green (D40: the launch is the owner's approval; he may review afterwards).
- `v2.0.0-rc.1` (more rc's if the rehearsal finds a bug), its GitHub **prerelease**, and the four-phase compose rehearsal.

**Out**
- The pre-flip run, the widening infra PR, `host-verify` + the snapshot, the `v2.0.0` tag and the verify → [ga-02](sprint-ga-02.md).
- Any `SIGNUP_MODE` change, any invite, re-sizing `SEAT_CAP`, raising the platform-AI limits to $100 / $80, the alerting revisit. These are v3 opening gates ([rollout §11](../rollout-plan.md#11-opening-gates-v3)). **GA has no dogfood gate:** the ≥ 2-week M4 data window is an opening gate.
- Keeping judge off the self path for DSA (`evaluator_only`) → M5, [m5-01](sprint-m5-01.md).
- The interviewer → M6a/M6b ([m6a-01](sprint-m6a-01.md) … [m6b-04](sprint-m6b-04.md), `v2.1.0`).
- The D6 content waves → `v2.0.x` patches after GA (`ev-d6-waves`).
- Contracting any column a removed flag leaves unused. It comes at least one release after its last reader ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)), so a `2.0.x`, **never `2.0.0`**.
- Removing the `preview` machinery or `inCohort`. Future courses still launch as `preview` ([ADR-0026 §1](../../adr/0026-per-course-extensibility-model.md#1-a-course-is-data-a-capability-is-code), ADR-0034 §2).

## Tasks

### 1 · Entry read [H]

Re-read the GA checklist's first line and the M3 checklist on the live system. **Stop if anything is red.**
- **Host and cluster.** Run `ssh vps 'bash /root/host-verify.sh --cluster --with-runner --nats-stage=n4'` ([mi-02](sprint-mi-02.md); the node copy refreshed per its README). It must show no FAIL:
  - the memory sum with the runner's 3 GiB counted;
  - Flux Ready;
  - no OOMKills;
  - PVCs < 60%;
  - `auth_required: true`;
  - the expected NetworkPolicies present, including the mi-11 egress set;
  - sar steal p95 below TR-STEAL (or R2 planned).
- **status.md:**
  - the 14 packs are stamped (content-status rows);
  - AB07–AB12 are frozen (artboard rows);
  - the MI, M1–M4, P and L rows are ✅.
- **The date of the last Hostinger weekly image,** from the owner's launch message (a before-launch item). ga-02 needs it again on GA day.

Record the read in the decisions log: "GA entry: M3 checklist re-read green on <date>".

### 2 · Strangers [H] (`ev-strangers`)

Rule: [ADR-0033 §2](../../adr/0033-invite-only-admission-and-owner-admin.md#2-close-v1-signup-now-v15x-stopgap-shipped-in-v152). Before the `v2.0.0` flip there is **no `active` account with role `learner`**. Accounts from v1's open signup (M1a stamped them `admitted_via='grandfathered'`) would otherwise get judge, platform AI and the pilot at GA. The CLI is [ADR-0033 §8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec) ([m1-04](sprint-m1-04.md), with erase from [l-01](sprint-l-01.md)/[l-02](sprint-l-02.md)).

1. **Read (the agent runs it; read-only, but it writes `admin_audit`):**

   ```sh
   ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list --role learner --status active'
   ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin seats'
   ```

   Keep the output **in the session's terminal only**. It carries emails and usernames. **Never** paste it into a file, a PR, a commit or status.md, because the repo is public.
2. **Triage** (D40: launching approves the reversible default; the irreversible erase stays the owner's, before launch):
   - **Suspend (default).** Reversible. The data stays. Sessions are revoked in the same transaction, the public profile returns the uniform 404 (P11), and the account can't sign in. The session runs `identity admin account suspend <account-id>` for **every account still listed** after the owner's before-launch erases and the tester re-roles below.
   - **Erase.** Irreversible, and there are no backups (D12). **The owner runs it himself, before launch** (optional):

     ```sh
     ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account erase <account-id> --confirm <account-id>'
     ```

     `--confirm` must repeat the same account ([l-02](sprint-l-02.md) task 3: non-interactive `kubectl exec` has no prompt, so the verb refuses without it). It needs every ack `topology.go` expects (practice, review, assessment, coach, judge); confirm with `… identity admin erasures --open` → empty.
   - An account the owner recognises as a real tester: `account set-role <id> tester`. It leaves the seat count. The owner names such accounts in his launch message (or re-roles them before launch), and the session runs the verb for each named account.
3. **Verify:**
   - the `account list … --role learner --status active` read → **empty**;
   - `seats` → **0** active learners out of 15, **0** outstanding invites.
4. **Record, counts only:**
   - "`ev-strangers` ✅ <date>: N suspended, M erased, K re-roled; 0 active learners; seats 0/15";
   - one CLI-use log line per verb.

The GA PR merges only once this read is empty (task 8): a production-state gate the session itself clears, not an owner wait. ga-02 re-reads it on GA day.

### 3 · GA PR: `.release-line` and the T-1 flips [X]

Branch `feat/ga-v2-default-flip` from an up-to-date `main`. Sources:
- [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): "`2.0.0` flips the D32 defaults (judge and platform AI on, pilot `active`) for every account";
- [§2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): the dark-launch path "code off (T-2) → cohort (T-3) → the **default flip in code at the labelled tag** → the flag is removed by its removal milestone"; presence by config.

**a. `.release-line`.** `1` → `2`. The `deploy.yml` guard reads the file **at the tagged commit**, so:
- from this merge, a `v1.*` tag cut from `main` builds nothing;
- a 1.x hotfix still works from a branch off `<last-1.x>` (ADR-0034 §1.3).

**b. Judge for every account.**
- **gateway:** delete `const judgeCohortOnly` ([m3-09](sprint-m3-09.md) task 1). `g.judgeFor` becomes `authSession` → `g.judge != nil`. **`JUDGE_BASE_URL` stays the kill switch:** unset means every judge route answers exactly `apiNotFound`'s 404 for everyone.
- **practice:** its attempt start takes the gateway's forwarded presence decision (m3-09 task 3). If [m3-08](sprint-m3-08.md) also added a practice-side role check ("judge path only when … role ∈ {owner, tester}"), remove it. The grading override stays.
- **judge:** if any learner-path handler checks the role, remove that check (judge's own admission L9–L15 is unchanged).

**c. Platform AI for every account.** Remove every AI cohort gate M4 added. Cross-check against the names the flag inventory records; there are three places:
- **judge:** delete `platformAICohortOnly` in `internal/judge/ai` ([m4-02](sprint-m4-02.md) task 8: gate 2 of `Reserve`, "`role ∈ {owner, tester}` while `platformAICohortOnly = true`"). `Reserve` keeps gates 1 and 3–5: the kill switch, `status = active`, consent per purpose, `ai_disabled`, the breaker and the caps.
- **gateway, dispute:** drop the cohort term from [m4-04](sprint-m4-04.md)'s presence check on `POST /api/evaluations/{id}/dispute` ("404s unless `JUDGE_BASE_URL` is set and the account is in the cohort"). It becomes `JUDGE_BASE_URL` set.
- **gateway, allowance:** drop the cohort term from [m4-05](sprint-m4-05.md)'s presence check on `GET /api/me/ai-allowance` (`internal/gateway/ai.go`), and from its `state = off` rule ("outside the cohort"). **Retire the `not_cohort` reason:** nothing emits it from `v2.0.0`. Remove it from the `reason` enum in `docs/architecture/openapi.yaml` and from [m4-06](sprint-m4-06.md)'s web handling, if any (a response-enum narrowing; the SPA ships inside the gateway image, so an R-b rolls both back together), and update the golden test.

What stays:
- **`LLM_PLATFORM_ENABLED` stays the permanent kill switch, and its code default stays `false`.** That is presence by config: production has set it to `true` since m4-07 through infra. Tests set it **in the test harness only**, next to an `httptest` fake provider ([m4-03](sprint-m4-03.md)'s e2e convention; [m4-01](sprint-m4-01.md): fakes are never committed as compose services). Committed compose leaves it unset.
- **Consent still gates every call** ([ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24): the two unticked opt-ins, plus the behavioural opt-in). The flip removes only the role check.
- The caps (L17) and the $12 app / $15 provider limits are unchanged ([mi-12](sprint-mi-12.md)).

**d. Pilot `active`.** In `curriculum/courses/go-concurrency/course.json`, `"status": "preview"` → `"active"`. **Nothing else** in the manifest changes. The `content` CI job must pass with `active`; if it enforces stricter rules for active courses, fix the content, not the rule. The `preview` machinery (`courseVisible`, `inCohort`, m1-04 task 7) stays for future courses.

**e. Sweep.** Every hit of the following is either removed (judge, AI) or justified in the PR description (`preview`):

```sh
git grep -nE 'CohortOnly|cohortOnly|inCohort\(|isCohort|dogfood' -- internal web/src
```

In `web/src`, any copy that assumes the cohort for judge or AI surfaces is updated. The pilot's `preview` badge disappears through data, because the status changed.

**f. Tests.** Flip the expectations; don't delete coverage:
- m3-09's presence-gate tests become: a `learner` with `JUDGE_BASE_URL` set → 200; with it unset → the identical 404 for **every** role.
- `internal/e2e/m3_exit_test.go`'s learner leg: "self path, every judge route 404" becomes an **execution-graded** conclusion `graded_by=auto, trust=checked` with provenance.
- The M4 AI gate tests: a learner with the consents gets a provisional AI suggestion; without them, manual entry (`self_grade_pending`). That covers m4-02's `Reserve` gating tests ("non-cohort ⇒ zero calls" becomes "`learner` with consent ⇒ a call"), m4-04's dispute presence test and m4-05's allowance presence and golden tests.
- p-02's real-manifest matrix now expects go-concurrency for anonymous and `learner` viewers under `active` rules:
  - listed in the catalog;
  - enrollment 201;
  - `/api/problems/gc-*` 200 once enrolled, as for DSA;
  - public profile rows and `/public/stats?paths=go-concurrency` only for **enrolled ∩ visible ∩ active** (P2).
- m1-04's `preview` matrix moves to the **fixture** `preview` manifest, so `preview` stays covered.
- **New `internal/e2e/ga_flip_test.go`** (`-tags e2e`). It runs in CI's in-process **`e2e`** job, so it uses that job's harness, not compose:
  - **services in-process:** the gateway, identity, curriculum, practice and judge handlers wired as the existing `internal/e2e` tests do, on the job's Postgres service (`XLEARN_TEST_DATABASE_URL`) and embedded JetStream;
  - **judge:** the fixture pack `internal/judge/testdata/pack`, and [m3-06](sprint-m3-06.md)'s `internal/judge/runnerfake` scripted to an AC Result for the fixture item. Real execution for a `learner` is proven by the flipped `m3_exit_test.go` (`-tags e2e,runner`, the `judge-runner-e2e` job) and by task 10;
  - **platform AI:** [m4-02](sprint-m4-02.md)'s `httptest` fake provider, with `LLM_PLATFORM_ENABLED=true` set in the harness only; consent rows seeded by SQL fixture, as [m4-03](sprint-m4-03.md)'s e2e does;
  - **accounts:** created in-process with identity's store fixtures (or by calling [m1-04](sprint-m1-04.md)'s `internal/identity/admin` package directly): a `learner`, plus an `owner` for contrast. There is no compose CLI in CI.

  Cases, all for the `learner`:
  - executes a packed fixture item (Submit → `auto · checked`);
  - with consent, gets an AI suggestion; without it, manual entry;
  - the dispute route and `/api/me/ai-allowance` answer (no `not_cohort` anywhere);
  - sees and enrolls go-concurrency.

  **Kill-switch drills** (harness config per sub-test): `JUDGE_BASE_URL` unset → the self path for everyone, and nothing is re-graded; `LLM_PLATFORM_ENABLED=false` → manual. The compose-only checks (published images, the real runner, R-b) stay in task 10's rehearsal.
- **New `internal/e2e/ga_rehearse_test.go`** (`//go:build rehearse`; black-box over `XLEARN_REHEARSE_BASE_URL`, like [m1-08](sprint-m1-08.md)'s `rehearse_test.go`). The same learner assertions, with `XLEARN_REHEARSE_EXPECT=v2` (flipped) or `v1` (the 1.x behaviour: judge 404, no gc). Task 10 points it at published images. It lands **in this PR** so nothing needs committing after the rc.

**g. Docs.**
- `docs/architecture/api.md`: presence is "configured" instead of "configured and cohort" for the judge and AI routes.
- `docs/architecture/services.md`, if it mentions the cohort for judge or AI.
- No route is added or removed. The only `openapi.yaml` change is the retired `not_cohort` enum value (3c), and the route drift test is unchanged.

**No migration in this PR.** If one is unavoidable, it is expand only, and task 5's lint proves it.

Before opening the PR, sanity-check from source: `docker compose up --build`, a learner signs up (compose runs `SIGNUP_MODE=open`), sees Run/Submit and sees go-concurrency.

### 4 · Flag inventory [X]

Per [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): ≤ 6 live non-kill flags, each with an owning and a removal milestone. Kill switches and operating modes are permanent.
- **Remove** every flag whose removal milestone in status.md is **GA**. That means the T-3 cohort gates the T-1 flip makes redundant (task 3 b–c), plus anything else the inventory lists for GA. Remove the code, its config parsing, its tests and its HelmRelease env docs. If a flag is set in `../infra` (none is expected: the cohort gates are code constants), its env line goes in a **separate infra PR merged after `v2.0.0`**. Record it (flag, service, HelmRelease file) in the status.md flag inventory; [ga-02](sprint-ga-02.md) task 8 removes it after the verify.
- **`COURSE_STATUS_OVERRIDE`** ([p-02](sprint-p-02.md) task 3: removal milestone GA, "either removed … or promoted to a permanent operating mode"). p-02 recommended **keeping it**, because every future course launch needs "hide without a tag". The session decides by that recommendation (D40), records the decision in the PR and the decisions log, and the owner may reverse it later with a follow-up PR:
  - **keep (recommended; the default):**
    - a dated amendment line in ADR-0034 §2 adds it to the permanent "kill switches and operating modes" list. The Accepted ADR is amended in place with a dated note, as the build-plan session folded the T7 amendments;
    - status.md moves it out of the non-kill budget into the operating modes;
    - the `COURSE_STATUS_OVERRIDE=go-concurrency=preview` drill (task 10) stays in the rehearsal. It is the pilot's R-a ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step), P row);
  - **remove** (the alternative, not taken unless the owner has already said so in status.md): delete the parser in `internal/course`, its tests and the inventory row.
- **Kept, unchanged:** the grading override, `JUDGE_BASE_URL`, `LLM_PLATFORM_ENABLED`, `REVISION_ENTRY_RULE`, `SIGNUP_MODE`.
- **status.md inventory after GA:** each removed flag reads "removed in GA PR #N, ships in `v2.0.0`". Count the remaining live non-kill flags (≤ 6).

### 5 · No-contract assertion [X]

The GA checklist's third line ("`2.0.0` carries **no contract migration**") is what makes GA's rollback safe. R-b to `<2.0.0` returns the last 1.x, and that only works if 1.x runs on the 2.0.0 schema ([ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) "Rollback of GA", [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) GA row). Make it mechanical:
- **`hack/lint-migrations.sh --no-contract-since <ref>`** (extend [m1-02](sprint-m1-02.md)'s lint; don't fork it). For every migration file **added** in `<ref>..HEAD` (`git diff --name-only --diff-filter=A`), it fails if the file carries `-- xlearn:contract` **or** an unrelaxed contract statement (the lint's own classifier). It also fails if any migration file that existed at `<ref>` was **modified**: goose files are immutable once shipped. Self-tests use a throwaway git repo built in a temp dir by `hack/testdata/migrations/no-contract-since.sh`, with these cases:
  - expand-only → pass;
  - contract marker → fail;
  - a bare `DROP COLUMN` → fail;
  - `-- xlearn:relax` → pass;
  - an old file edited → fail.

  Run them in `ci.yml`'s `go` job and in `make lint`.
- **`deploy.yml`, in the `release-line` job:**
  - Replace the sparse checkout with `actions/checkout@v4` `fetch-depth: 0`, so the tags are present. The job stays first, and every image job `needs` it.
  - Add a step after the major check. It fires for **a major's first stable release and its rc's**: the tag's major is ≥ 2 and **no other non-prerelease `v<major>.*` git tag exists**. That is `vN.0.0` and its rc's in the normal case, and also the version ga-02 would cut above an undeletable GHCR stray (for example `v2.0.1`; ga-02 task 2 deletes the stray git tag, which is always possible). A pattern on `vN.0.0` alone would skip that tag.

    ```sh
    major=${GITHUB_REF_NAME#v}; major=${major%%.*}
    others=$(git tag -l "v${major}.*" | grep -v -- '-' | grep -vxF "$GITHUB_REF_NAME" || true)
    if [ "$major" -ge 2 ] && [ -z "$others" ]; then
      prev=$(git tag -l "v$((major-1)).*" | grep -v -- '-' | sort -V | tail -1)
      hack/lint-migrations.sh --no-contract-since "$prev"
    fi
    ```

    A failure stops every image build, so no first-of-major image (`2.0.0`, or GA cut above a stray) can exist with a contract. Once `v2.0.0` exists, `v2.0.x` patches skip the step, as they should: a contract may land in a `2.0.x`.
  - Update the header comment: the range is "bounded to the live major (`.release-line`)" rather than a hard-coded `<2.0.0`.
- This covers `v2.0.0`, its rc's, a GA cut above a stray, and every later major's first release. `v2.0.0-rc.1` (task 9) is the step's first real run.

### 6 · `hack/ga-preflip-check.sh` [X]

ADR-0034 §1.4 step 3, as a read-only script. **Why it's written here and not in ga-02:** ga-02 must tag the exact commit this sprint's rc rehearsed. A script merged on GA day would move `main` and force a new rc. The rc dry run (task 10) also proves the script ignores prerelease tags.

Bash with `curl` and `jq` only. No new toolchain: `crane` isn't installed and isn't needed.
1. **Line:** `.release-line` at HEAD equals `--major` (default: the file's content). Refuse `--major 1`.
2. **Git tags:** `git ls-remote --tags origin "refs/tags/v${major}*"` → **FAIL** on any ref that isn't a prerelease (`vM.m.p-…`; peeled `^{}` refs are folded). rc refs are listed as OK.
3. **GHCR tags on the fleet:**
   - Parse the fleet from `.github/workflows/deploy.yml` (`image: ghcr.io/sujaykumarsuman/xlearn-*`) and **assert exactly these 8**: gateway, identity, curriculum, practice, review, assessment, coach, judge. `xlearn-runner` and `xlearn-evalpack` are other streams, and the evalpack package is private.
   - For each, fetch an anonymous pull token: `curl -s "https://ghcr.io/token?scope=repository:sujaykumarsuman/xlearn-<svc>:pull"`.
   - List `https://ghcr.io/v2/sujaykumarsuman/xlearn-<svc>/tags/list?n=1000`, following `Link` pagination.
   - **FAIL** on any tag that Flux's semver (Masterminds, lenient) would read as a stable `major.*`: `^v?2(\.[0-9]+){0,2}(\+[0-9A-Za-z.-]+)?$`. So `2`, `2.0`, `v2.0.0` and `2.0.0+x` all count; `2.0.0-rc.1`, `sha-…`, `latest` and `0.1.N` don't.
4. **No contract:** `hack/lint-migrations.sh --no-contract-since <last stable (major−1).x tag>`.
5. **`--cluster`** (optional, read-only). It prints `name`, `.spec.policy.semver.range` and `.status.latestRef.tag` for every `xlearn-*` ImagePolicy, through `ssh vps 'k3s kubectl get imagepolicy -n flux-system -o json'`. ga-02 reads it before and after the widening.

Behaviour:
- Exit 0 only when every check passes.
- On a FAIL it prints the ADR remedy: "delete the stray tag and image version, or cut GA above it and tag **before** widening. See ga-02 task 2."
- Overrides for the self-test: `--tags-file`, `--ghcr-dir`, `--deploy-yml`.
- Self-test `hack/testdata/preflip/`:
  - a stray `v2.0.0` git tag → fail;
  - a stray `2.0.0` GHCR tag → fail;
  - a lenient `v2.0` tag → fail;
  - only rc tags → pass;
  - a 9th `xlearn-*` image in the workflow → fail.

  It runs in `ci.yml`'s `go` job without network.

### 7 · Release notes + drift check [X]

**Release notes.** The GA PR body is the **behaviour-change list**. ga-02 reuses it as the `v2.0.0` GitHub release body. For every account (in v2 the owner and the testers; learners from the v3 opening):
- **Judge** ([ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md#consequences) Consequences):
  - Run/Submit on packed DSA items, execution-graded with provenance;
  - the full statement at start; a hard **45:00** limit; the hint at 15:00 caps the grade at Assisted; using the coach caps at Assisted; revealing the solution = give-up = **Miss**; "complexity stated **correctly**"; **no re-implement** stage (D15/D16/D18);
  - ahead-of-schedule items open the **arena**; the arena is honor-system and never counts;
  - Week, Mistakes and Progress show provenance and the judge-checked % (AB12).
- **Platform AI** ([ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md)):
  - AI-suggested grades are **provisional**, auto-accept after 24 h, and can be disputed with one blind re-grade;
  - pointer notes and "correct, with improvements";
  - the allowance meter, shown as a percentage;
  - all of it **only with the consents**; manual entry always works; the breaker degrades to deterministic pre-fill.
- **go-concurrency `active`:**
  - listed in the catalog, enrollable, and in the agenda;
  - public-profile rows and `/public/stats` follow the visibility toggles (P2, P5);
  - race and goroutine results are labelled honor (`trust=honor`).
- **Unchanged:**
  - `SIGNUP_MODE=closed`; no invite;
  - `/api/v1` (no route removed);
  - every kill switch;
  - the D2 ladder rule, D27 and D31 (since their 1.x minors).
- **Rollback:** R-b to `<2.0.0` returns `<last-1.x>`. There is no contract.
- **Ops:**
  - `main` is on release line 2, and a 1.x hotfix branches from `<last-1.x>`;
  - the fleet ranges widen to `<3.0.0` right before the tag (ga-02); runner and evalpack stay `<2.0.0`.
- **Owner note:** once the pilot is `active`, **your go-concurrency progress shows on your public profile** if the course's visibility is on (AB22 toggles). Choose before ga-02.

**Drift check.** Compare `docs/git-strategy.md` (rewritten in the build-plan session) step by step with ADR-0034 §1.3–§1.5 and with this sprint + ga-02. Sections: major-line guard, GA procedure, `-rc`, the range-change order, the runner and evalpack streams, R-a…R-d. Fix drift in this PR:
- "every `xlearn-*`" = the 8 fleet policies; runner and evalpack are separate streams;
- the pre-flip script and the no-contract `deploy.yml` step;
- the GA freeze and hotfix-from-`<last-1.x>` note;
- `.release-line` reads "`2` from the GA PR".

Also grep for stale mentions (`git grep -n -e '<2.0.0' -e '1.x until'`) outside `docs/v1/` (historical) and the ADRs:
- the `deploy.yml` header (task 5);
- `docs/git-strategy.md`.

Add a **dated note** under ADR-0034 §1.4, approved by the launch (D40) and landing with the GA PR. It carries up to two clarifications:
- step 4's "every `xlearn-*`" means the 8 fleet policies (if the text still needs it);
- step 3's "an anonymous `crane ls` works" is done by `hack/ga-preflip-check.sh` with anonymous `curl` against the GHCR registry API (token + `tags/list`), because `crane` isn't installed. It is the same check, lenient forms included. The infra comment in `apps/image-automation.yaml` changes in ga-02's widening PR, not here.

### 8 · Merge the GA PR on CI green [X]

Open the PR:
- title `feat!: v2.0 GA default flip — .release-line 2, judge + platform AI for every account, go-concurrency active`;
- the body carries the release notes, the flag decisions, the sweep's justified `inCohort` hits and the `COURSE_STATUS_OVERRIDE` decision;
- the attribution lines.

**CI green:** `go` (gofmt, vet, `go test -race`, **`sqlc diff`**, the migration lint + its new self-tests, the pre-flip self-test, the OpenAPI drift), `e2e` (including `ga_flip_test.go`), `judge-runner-e2e` (the flipped `m3_exit_test.go`, `-tags e2e,runner`), `web`, `content`.

**Then merge, with no separate approval (D40).** The major bump is still a deliberate, self-reviewed PR (ADR-0034 §1.3), and launching this prompt is the owner's approval of it; he may review after the merge. An owner "hold" in the session still overrides. Then:
1. Confirm the task 2 gate (no active learner) is still true. Re-run the read.
2. Squash-merge. Merging deploys nothing.
3. Tell the peers: "`main` is on release line 2 from <sha>; no tags from `main` until ga-02 reports `v2.0.0` verified; a 1.x hotfix branches from `<last-1.x>`".

### 9 · `v2.0.0-rc.1` [X]

ADR-0034 §1.4 step 2; the [git-strategy](../../git-strategy.md) `-rc` convention.
1. **Parallel-sessions check:**
   - `git ls-remote --tags origin 'refs/tags/v2*'` shows nothing, or only earlier rc's (take the next N);
   - `gh pr list`, `git worktree list`, ListAgents.
2. **Tag and release:**

   ```sh
   git tag -a v2.0.0-rc.1 <merge sha> -m "v2.0.0-rc.1 — v2.0 GA rehearsal (prerelease; never deploys)"
   git push origin v2.0.0-rc.1
   gh release create v2.0.0-rc.1 --verify-tag --prerelease --latest=false --title "v2.0.0-rc.1 — v2.0 GA rehearsal" --notes "Prerelease: builds, never deploys (ADR-0034 §1.4 step 2)."
   ```
3. **Watch `deploy.yml`** (`gh run watch`):
   - the `release-line` job prints the prerelease notice;
   - the **no-contract step runs and passes** (it's the first real run);
   - 8 images `2.0.0-rc.1` are pushed.
4. **Prove it never deploys** (read-only):
   - `ssh vps 'k3s kubectl get imagepolicy -n flux-system'` shows the fleet `latest` still at `<last-1.x>`;
   - `git -C ../infra log origin/main --oneline -5` shows no `chore(images)` commit for `xlearn-*`.
5. **Record the rc'd commit SHA.** ga-02 tags `v2.0.0` on it, or on a later `main` whose diff from it touches only `docs/`, `*.md` or `design-system/`. Those paths are in `.dockerignore`, and `.github/` is **not** exempt, because the tag runs the workflow at the tagged commit.

### 10 · Compose rehearsal [X]

Use [m1-08](sprint-m1-08.md)'s override `deploy/local/compose.rehearse.yml`, which pins each service to `ghcr.io/sujaykumarsuman/xlearn-<svc>:${XLEARN_IMAGE_TAG}`. Now 8 services, judge included (check m3-05 added it; if not, add it **in the GA PR**, never after the rc). Also:
- a separate compose project `xlearn-rehearse` with its own volumes;
- the runner at its live `runner-v1.x`, with `RUNNER_MODE=dev`;
- the fixture pack;
- a throwaway fake LLM provider in the scratchpad, wired through a scratchpad compose override that also sets `LLM_PLATFORM_ENABLED=true` on judge ([m4-01](sprint-m4-01.md)'s convention: fakes are never committed as compose services); **never committed**.

Any orchestration script stays in the scratchpad: nothing is committed after the rc.

| Phase | Images | Do | Check |
|---|---|---|---|
| 0 | `<last-1.x>` (e.g. `1.17.0`) | Fresh volumes. Owner (`identity admin account set-role … owner`) + a learner (email signup; compose runs `open`). The owner does a judged submit on a fixture item; the learner does a self-graded solve. `ga_rehearse_test.go` with `EXPECT=v1` | learner: judge routes 404, no go-concurrency anywhere; owner: judge + go-concurrency with the `preview` badge |
| 1 | `2.0.0-rc.1` | Start: any expand migration applies, and the lint already proved no contract. `EXPECT=v2`, then the drills | learner: Run/Submit → `auto · checked`; AI suggestion with consent, manual without; go-concurrency `active`, enrollable; the public profile shows gc only when visible. Owner unchanged. **Drills:** unset `JUDGE_BASE_URL` on gateway + practice → self path for everyone, nothing re-graded; `LLM_PLATFORM_ENABLED=false` → manual; if kept, `COURSE_STATUS_OVERRIDE=go-concurrency=preview` → gc hidden again outside the cohort |
| 2 (R-b) | `<last-1.x>` on phase 1's schema | Start; restart each service once; `EXPECT=v1` | goose no-ops on the newer versions (no error); all Ready; the learner is back on the 1.x behaviour; phase 1's evaluations, AI rows and gc enrollment are tolerated (no crash, no 500). **This proves GA's R-b** |
| 3 | `2.0.0-rc.1` | Start; `EXPECT=v2` | roll-forward is a no-op; green |

Also run:
- `go test -tags e2e ./internal/e2e/...` at the merge commit;
- **`hack/ga-preflip-check.sh --cluster` now:**
  - it must **PASS with the rc tags present** (git `v2.0.0-rc.1` and GHCR `2.0.0-rc.1` → OK), which proves the prerelease filter;
  - `--cluster` shows 8 fleet ranges `>=1.0.0 <2.0.0` at `<last-1.x>`, and `xlearn-runner` / `xlearn-evalpack` at their own versions.

**On a failure:**
1. Fix PR, merged on CI green (D40).
2. `v2.0.0-rc.2`.
3. Repeat phases 1–3.

Record the final rc and its SHA.

### 11 · Record [X]

In [`../status.md`](../status.md):
- **Sprint board:** ga-01.
- **Milestones:** GA 🔄 (GA PR #, `v2.0.0-rc.N` = <sha>).
- **Release line:** `main` on **2** from <date>; 1.x hotfixes from `<last-1.x>`.
- **Flag inventory:** task 4's removals and the `COURSE_STATUS_OVERRIDE` decision.
- **Owner events:** `ev-strangers` ✅ (counts only).
- **CLI-use log:** one line per verb.
- **Decisions log:**
  1. the widening covers the 8 fleet policies; runner and evalpack are separate streams;
  2. the `COURSE_STATUS_OVERRIDE` choice;
  3. the pre-flip script was written in ga-01, so `v2.0.0` tags the rehearsed commit;
  4. the `deploy.yml` no-contract step for a major's first stable release and its rc's;
  5. the rc rehearsal result, including R-b.

Land it through a small **docs-only** PR. It may merge after the rc, because `docs/` is not an image input.

## Acceptance criteria

- [ ] `identity admin account list --role learner --status active` is **empty**, and `seats` shows 0/15 learners and 0 outstanding invites. It is recorded with counts only.
- [ ] The GA PR is **merged on CI green** (D40; no separate approval):
  - `.release-line` = `2`;
  - judge and platform AI have no role gate in the code defaults (kill switches intact; consents intact);
  - go-concurrency is `active`;
  - the GA-removal flags are gone, and `COURSE_STATUS_OVERRIDE` is decided (kept, per the recommendation, unless status.md already records the owner's call) and recorded;
  - the release notes carry the behaviour-change list.
- [ ] The `--no-contract-since` lint and the `deploy.yml` first-of-major step are in place with self-tests. `v2.0.0-rc.1`'s run shows the step green.
- [ ] `hack/ga-preflip-check.sh` is on `main` with a green self-test, and it PASSES on the live state with the rc tags present.
- [ ] `v2.0.0-rc.N` is published as a **prerelease**; its 8 images exist; **nothing deployed** (fleet latest still `<last-1.x>`, no IUA commit).
- [ ] The compose rehearsal is green on all four phases, **including R-b to `<last-1.x>` on the 2.0.0 schema** and the kill-switch drills.
- [ ] `docs/git-strategy.md`, the `deploy.yml` header and `api.md` match what ga-01 and ga-02 do.

## Release

**Merge only**, plus the **`v2.0.0-rc.N` prerelease**. No image deploys, because Flux ranges skip prereleases. `main` is build-only.
- **The rc is required here**, although GA checklist item 2 says "Optional rehearsal": [ga-02](sprint-ga-02.md) tags the rc'd commit, and task 10's R-b phase is the only proof that GA's rollback works.
- The GA PR merges **on CI green** (D40). ADR-0034 §1.3's "the major bump becomes a reviewed PR" is this deliberate PR; launching the prompt is the owner's approval, and there is no separate approval stop.
- `v2.0.0` itself, the widening infra PR and the snapshot are [ga-02](sprint-ga-02.md). **This sprint never touches `../infra`.**
- **Before the rc tag:** the parallel-sessions check (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents). The rc is `v2.0.0-rc.N` with the next free N. It passes the `.release-line` guard as a prerelease, and the no-contract step as an rc of the major's first release.
- **What ships the content of this PR:** `v2.0.0`, tagged in [ga-02](sprint-ga-02.md) on the rc'd commit.

## Definition of Done

- CI green, including `sqlc diff`, the new self-tests and the flipped e2e.
- The strangers are triaged (counts recorded).
- The GA PR is merged on CI green (D40).
- `v2.0.0-rc.N` is built and never deployed.
- The four-phase compose rehearsal is green, and the pre-flip check passes on the live state.
- The rc'd SHA is recorded for ga-02.
- Statuses are updated: this file, and [`../status.md`](../status.md) (board, GA 🔄, release line, flag inventory, `ev-strangers`, the CLI-use log, decisions).
- `main` is synced locally. The peers know `main` is on the 2 line.

## Risks / watch-outs

- **A stray non-prerelease `2.*` would deploy at the widening.** Any image `Masterminds` reads as stable 2.x deploys the moment ga-02 widens the ranges. Examples: a peer's `v2.0.0` from a commit that predates xlearn#53, or a `2.0` tag. The pre-flip script checks the lenient forms. Until ga-02 is done, **no peer tags from `main`**. After `.release-line=2`, a peer's `v2.0.1` would build.
- **`.release-line = 2` freezes 1.x from `main`.** Between this merge and GA, a production fix is a hotfix branch from `<last-1.x>`, tagged `v1.x.(y+1)`, and cherry-picked to `main`. Then cut a new rc, because `main` changed.
- **The flip is invisible on production.** Every active account is the owner or a tester, all in the cohort. The compose rehearsal is the proof for `learner`, and ga-02 has an optional tester-as-learner check.
- **A leftover cohort check.** A role check hidden in practice or judge would leave learners on the self path after GA; one left in `platformAICohortOnly`, the dispute route or `/api/me/ai-allowance` would leave them without AI. The sweep (3e), the flipped m3-exit learner leg and `ga_flip_test.go` catch it.
- **`LLM_PLATFORM_ENABLED`'s code default.** Flipping it to `true` in code would enable AI wherever the env is unset: committed compose, and tests without the `httptest` fake. Keep the default `false`. It's a presence-by-config kill switch, and production sets it.
- **The pilot's public visibility.** `active` makes gc rows public for the owner if his visibility is on. The release notes tell him to choose (AB22).
- **The strangers list is PII.** Keep it in the terminal only. Erase is irreversible (D12), so the owner runs it himself, before launch; the session only suspends.
- **Committing after the rc moves the tagged commit.** Only `docs/`, `*.md` and `design-system/` changes may follow the rc. Anything else, `.github/` included, needs `-rc.N+1`.
- **The rehearsal's fake LLM provider** must never be committed, and never pointed at a real key.
