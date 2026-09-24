# Prompt — Sprint ga-01 · GA PR: .release-line = 2, T-1 default flips, rc rehearsal

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root. The session may pause at the owner's review of the GA PR (step 9) and resume in a later sitting.
> **Plan:** [`../sprints/sprint-ga-01.md`](../sprints/sprint-ga-01.md)   ·   **Milestone:** GA (part 1 of 2)   ·   **Prereqs:** [l-04](../sprints/sprint-l-04.md) (L exit), [p-03](../sprints/sprint-p-03.md), [m4-07](../sprints/sprint-m4-07.md), [m3-13](../sprints/sprint-m3-13.md), [mi-11](../sprints/sprint-mi-11.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, land-and-sync, status updates.
- **The plan:** [`../sprints/sprint-ga-01.md`](../sprints/sprint-ga-01.md). It holds:
  - the verbatim GA and M3 checklists;
  - the flip list;
  - the script specs;
  - the rehearsal phases;
  - the release-notes outline.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):
  - [§1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme) (labels are GA flips; what `2.0.0` flips);
  - [§1.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#13-accidental-major-guard-shipped-2026-09-24-xlearn53-infra29) (`.release-line`: the major bump is a reviewed PR; the check reads the file at the tagged commit);
  - [**§1.4**](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure) (the GA procedure, steps 1–2 here; "Rollback of GA");
  - [§1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) (runner and evalpack are separate streams);
  - [**§2**](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (T-1/T-2/T-3, presence by config, kill switches, ≤ 6 non-kill flags);
  - [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand/contract, floors);
  - [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) (the GA and P rows).
- [ADR-0033 §2](../../adr/0033-invite-only-admission-and-owner-admin.md#2-close-v1-signup-now-v15x-stopgap-shipped-in-v152) (the strangers rule) and [§8](../../adr/0033-invite-only-admission-and-owner-admin.md#8-the-owner-admin-cli-identity-admin-run-via-kubectl-exec) (the `identity admin` CLI).
- [ADR-0029 Consequences](../../adr/0029-judge-contract-and-learning-signal.md#consequences) (the judge behaviour changes vs v1) and [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §4–§5 (consents, spend). Together they make up the release-notes list.
- [rollout §4 "v2.0 GA"](../rollout-plan.md#v20-ga-the-owner-facing-default-flip), [§5](../rollout-plan.md#5-m3-hard-entry-checklist), [§7](../rollout-plan.md#7-indicative-tag-timeline), [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading), [§11](../rollout-plan.md#11-opening-gates-v3) (what is *not* GA). [t7 §5.2](../research/t7-cross-cutting-and-rollout.md#52-scheme-recommended-owner-q1) (why the pre-flip check exists).
- [`../../git-strategy.md`](../../git-strategy.md): the major-line guard, the GA procedure, `-rc`, range order, R-a…R-d. You check it for drift.
- **What the flip removes:**
  - [m3-09](../sprints/sprint-m3-09.md) task 1 (`judgeCohortOnly`, `judgeFor`);
  - [m3-08](../sprints/sprint-m3-08.md) (practice gating);
  - [m4-02](../sprints/sprint-m4-02.md), [m4-04](../sprints/sprint-m4-04.md), [m4-05](../sprints/sprint-m4-05.md) (the AI cohort gate);
  - [p-02](../sprints/sprint-p-02.md) task 3 (the real-manifest matrix, `COURSE_STATUS_OVERRIDE`);
  - [m1-04](../sprints/sprint-m1-04.md) task 7 (the `preview` cohort);
  - [m3-13](../sprints/sprint-m3-13.md) (`internal/e2e/m3_exit_test.go`);
  - [m1-02](../sprints/sprint-m1-02.md) task 4 (`hack/lint-migrations.sh`);
  - [m1-08](../sprints/sprint-m1-08.md) task 3 (`deploy/local/compose.rehearse.yml`, `internal/e2e/rehearse_test.go`).
- **Code:**
  - `.release-line`, `.github/workflows/{deploy,ci}.yml`, `.dockerignore`;
  - `hack/`;
  - `internal/gateway/` (the judge presence gate, the AI routes, `cohort.go`);
  - `internal/practice/`, `internal/judge/` (the AI gate), `internal/course/`;
  - `curriculum/courses/go-concurrency/course.json`;
  - `internal/e2e/`, `docker-compose.yml`, `deploy/local/`;
  - `web/src/`;
  - `docs/architecture/{api,services}.md`.
- **The live state:** [`../status.md`](../status.md) (Milestones, the MI rows, the flag inventory, content status, artboards, the decisions log).

## Context

Every v2.0 milestone has shipped as a 1.x minor:
- M1 `v1.8.0`, M2 `v1.10.0`, M3 `v1.14.0`, P `v1.15.0`, M4 `v1.16.0`, L `v1.17.0`;
- the L-exit invite round-trip is rehearsed, and production is back at `SIGNUP_MODE=closed`.

Judge and platform AI run on production **for the owner and tester cohort only** (T-3), with `JUDGE_BASE_URL` and `LLM_PLATFORM_ENABLED` set (T-2). go-concurrency is `preview`.

**v2.0 GA** (`v2.0.0`, D32/D35) flips the defaults for every account: judge and platform AI on, the pilot `active`. In v2 "every account" means the owner and testers. Signup stays closed, and the opening is v3.

ADR-0034 §1.4 makes GA a **two-repo sequence**. This sprint does steps 1–2:
- the **GA PR** (`.release-line = 2` + the T-1 flips + the release notes). It is reviewed and approved by the owner, and merging it deploys nothing;
- a **`v2.0.0-rc.1`** rehearsed in compose.

It also clears the **strangers** (no `active` learner) and lands the two guards that [ga-02](../sprints/sprint-ga-02.md) runs:
- the **no-contract assertion** (lint mode + a `deploy.yml` step);
- **`hack/ga-preflip-check.sh`**.

Both go in the GA PR, so `v2.0.0` can later be tagged on the exact commit you rehearse.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **MI complete for GA** (every MI step ✅ except MI-16) and **M1–M4 + P + L ✅** in status.md, with exits recorded.
- [ ] **The M3 checklist is still green.** Plan task 1:
  - `ssh vps 'bash /root/host-verify.sh --cluster --with-runner --nats-stage=n4'` has no FAIL;
  - 14 packs stamped; AB07–AB12 frozen; TR-STEAL quiet;
  - the owner has the weekly-image date.
- [ ] **L exit recorded,** and `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/auth/config` → `{"signup":"closed"}`.
- [ ] **The T-2 switches are live:**
  - `JUDGE_BASE_URL` is set on `../infra/apps/xlearn-{gateway,practice}.yaml`;
  - `LLM_PLATFORM_ENABLED: "true"` is set on `apps/xlearn-judge.yaml`.
- [ ] **ADRs:** ADR-0030 and ADR-0031 are Accepted.
- [ ] **No contract so far.** Find the last 1.x tag, `<last-1.x>`:

  ```sh
  git tag -l 'v1.*' --sort=-v:refname | grep -v -- - | head -1
  ```

  Then check that no migration since it carries the marker:

  ```sh
  git diff --name-only --diff-filter=AM <last-1.x>..origin/main -- 'internal/*/store/migrations/*.sql' | xargs -r grep -l 'xlearn:contract'
  ```

  The output must be empty.
- [ ] **The flag inventory** lists the judge and AI T-3 cohort gates (removal: GA) and p-02's `COURSE_STATUS_OVERRIDE` row.
- [ ] **Parallel sessions:**
  - no open peer PR touches `.release-line`, `deploy.yml`, `hack/lint-migrations.sh`, `internal/gateway/`, `internal/judge/`, the gc manifest or `docs/git-strategy.md`;
  - no peer plans a tag.

  Check with `gh pr list --state open`, `git ls-remote --tags origin`, `git worktree list` and ListAgents.
- [ ] **The owner is reachable** for the strangers triage (~15 min) and the GA PR review.

## Do this (in order)

1. **[H] Entry read** (plan task 1). Record "GA entry: M3 checklist re-read green <date>" in the decisions log draft.
2. **[O] Strangers** (plan task 2, `ev-strangers`):
   - Run the read-only lists:
     - `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account list --role learner --status active'`;
     - `… identity admin seats`.
   - Show them to the owner **in the terminal only**. They are PII, and the repo is public: never put them in a file, PR or status.md.
   - The owner decides per account:
     - **suspend** (default, reversible): you may run `account suspend <id>` only on his explicit per-account instruction;
     - **erase**: the owner runs it himself (irreversible, D12): `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account erase <id> --confirm <id>'`. `--confirm` must repeat the account (l-02's verb; `kubectl exec` has no prompt). Then check `… identity admin erasures --open` → empty;
     - **re-role** a real tester: `account set-role <id> tester`.
   - Verify: the list is **empty**; `seats` shows 0/15 and 0 outstanding invites.
   - Log each CLI use, with counts only.
3. **[X] Branch** `feat/ga-v2-default-flip` from an up-to-date `main`.
4. **[X] Flips** (plan task 3):
   - `.release-line` → `2`.
   - Delete `judgeCohortOnly`, so `judgeFor` = session + `g.judge != nil`. Remove any practice or judge role check for the judge path.
   - Remove the platform-AI cohort gates (all three; cross-check the inventory names):
     - `platformAICohortOnly` in `internal/judge/ai` (m4-02 task 8, gate 2 of `Reserve`);
     - the cohort term in m4-04's dispute-route presence check;
     - the cohort term in m4-05's `/api/me/ai-allowance` presence check and `state = off` rule. Retire the `not_cohort` reason (the enum in `docs/architecture/openapi.yaml`, any web handling, the golden test).

     Keep `LLM_PLATFORM_ENABLED` with code default `false` (tests set it in the harness only, next to an `httptest` fake; committed compose leaves it unset), and keep the consents.
   - In `curriculum/courses/go-concurrency/course.json`, `status` → `active` (nothing else).
   - Run the `git grep -nE 'CohortOnly|cohortOnly|inCohort\(|isCohort|dogfood' -- internal web/src` sweep. Remove each hit or justify it (`preview` stays).
   - Update `api.md` and `services.md`.
   - **Flip the tests:**
     - m3-09's presence tests;
     - `m3_exit_test.go`'s learner leg → `auto · checked`;
     - the M4 AI gate tests (m4-02's `Reserve` gating, m4-04's dispute presence, m4-05's allowance presence and golden);
     - p-02's real-manifest matrix (active rules, P2);
     - m1-04's `preview` matrix → the fixture manifest.
   - **Add** `internal/e2e/ga_flip_test.go` (`-tags e2e`, CI's in-process `e2e` job, **not compose**): the service handlers in-process on the job's Postgres and embedded JetStream; the fixture pack `internal/judge/testdata/pack` with m3-06's `internal/judge/runnerfake`; m4-02's `httptest` fake provider with `LLM_PLATFORM_ENABLED=true` set in the harness; consents seeded by SQL fixture; accounts from identity's store fixtures or m1-04's `internal/identity/admin` package called in-process. A `learner` gets `auto · checked`, an AI suggestion with consent (manual without), the dispute and allowance routes, and gc; plus the two kill-switch drills. Real execution stays in the flipped `m3_exit_test.go` (`judge-runner-e2e`) and in step 11.
   - **Add** `internal/e2e/ga_rehearse_test.go` (`//go:build rehearse`, `XLEARN_REHEARSE_EXPECT=v1|v2`).
   - **No migration.**
5. **[X] Flag inventory** (plan task 4):
   - Remove every GA-removal flag: code, config, tests, docs.
   - Prepare the `COURSE_STATUS_OVERRIDE` question for the owner. **Keep (recommended):** a dated amendment line in ADR-0034 §2's permanent list, and status.md moves it to the operating modes. **Or remove:** the parser, its tests and its row.
   - Kill switches untouched.
6. **[X] No-contract assertion** (plan task 5):
   - `hack/lint-migrations.sh --no-contract-since <ref>`: fail on an added contract file (marker or unrelaxed statement), and on any edited old migration. Self-tests via `hack/testdata/migrations/no-contract-since.sh`, run in `ci.yml` and `make lint`.
   - In `deploy.yml`'s `release-line` job:
     - use `fetch-depth: 0`;
     - add a step that fires for a major's first stable release and its rc's (major ≥ 2 and **no other non-prerelease `v<major>.*` git tag exists**, so `vN.0.0` and also a GA cut above an undeletable stray), and runs the lint against the last stable `v(major−1).*` tag;
     - make the header comment range-generic.
7. **[X] `hack/ga-preflip-check.sh`** (plan task 6). Bash + curl + jq:
   - the release line;
   - stable `v2*` git tags;
   - the fleet parsed from `deploy.yml` and **asserted as exactly 8**;
   - GHCR anonymous tag listing with pagination, and the lenient stable-2.x regex `^v?2(\.[0-9]+){0,2}(\+[0-9A-Za-z.-]+)?$`;
   - no-contract;
   - `--cluster` (read-only `ssh vps` ImagePolicy range/latest table).

   Overrides for the offline self-test in `hack/testdata/preflip/`, run in `ci.yml`.
8. **[X] Release notes + drift check** (plan task 7):
   - The PR body is the behaviour-change list: judge (D15/D16/D18, arena), platform AI (provisional, dispute, consents), gc `active` (visibility, honor labels), unchanged items, rollback, ops, and the owner's gc-visibility note.
   - Fix drift in `docs/git-strategy.md`: the 8 fleet policies vs the runner/evalpack streams; the pre-flip script; the no-contract step; the hotfix-from-`<last-1.x>` rule; `.release-line` "2 from the GA PR".
   - Grep for stale `<2.0.0` mentions.
   - Add a dated note under ADR-0034 §1.4: "every `xlearn-*`" = the 8 fleet policies (if still needed), and the pre-flip check's anonymous `curl` against the GHCR API stands in for step 3's `crane ls` (`crane` isn't installed).
9. **[X→O] Open the PR:**
   - title `feat!: v2.0 GA default flip — .release-line 2, judge + platform AI for every account, go-concurrency active`;
   - the attribution lines;
   - compose sanity from source: a learner sees Run/Submit and gc.

   Get **CI green**: gofmt, vet, `go test -race`, **`sqlc diff`**, the migration and pre-flip self-tests, OpenAPI drift, `e2e` (with `ga_flip_test.go`), `judge-runner-e2e` (the flipped `m3_exit_test.go`), `web`, `content`.

   **Then STOP and ask the owner to review and approve.** Standing merge authority does not cover this PR. On approval:
   - re-run step 2's read (still empty);
   - squash-merge;
   - tell the peers: "`main` is on release line 2 from <sha>; no tags from `main` until ga-02 reports `v2.0.0` verified; a 1.x hotfix branches from `<last-1.x>`".
10. **[X] Tag `v2.0.0-rc.1`** (plan task 9):
    - Parallel-sessions check; `git ls-remote --tags origin 'refs/tags/v2*'` → nothing, or earlier rc's (take the next N).
    - Tag and release:

      ```sh
      git tag -a v2.0.0-rc.1 <merge sha> -m "v2.0.0-rc.1 — v2.0 GA rehearsal (prerelease; never deploys)"
      git push origin v2.0.0-rc.1
      gh release create v2.0.0-rc.1 --verify-tag --prerelease --latest=false …
      ```
    - `gh run watch`: the prerelease notice, the **no-contract step green**, 8 × `2.0.0-rc.1` images.
    - Prove it never deployed (read-only): the fleet ImagePolicies are still at `<last-1.x>`, and there is no `chore(images)` commit in `../infra`.
11. **[X] Compose rehearsal** (plan task 10). Use `deploy/local/compose.rehearse.yml` with `XLEARN_IMAGE_TAG` (8 services), project `xlearn-rehearse`, the live runner in `RUNNER_MODE=dev`, the fixture pack, and a throwaway fake LLM provider in the scratchpad, wired by a scratchpad compose override that also sets `LLM_PLATFORM_ENABLED=true` on judge (m4-01: fakes are never committed as compose services). The orchestration stays in the scratchpad.

    | Phase | Images | Expect |
    |---|---|---|
    | 0 | `<last-1.x>` | `v1` |
    | 1 | `2.0.0-rc.1` | `v2`, plus the drills: `JUDGE_BASE_URL` unset; `LLM_PLATFORM_ENABLED=false`; `COURSE_STATUS_OVERRIDE=go-concurrency=preview` if kept |
    | 2 (R-b) | `<last-1.x>` on the 2.0.0 schema | `v1`; restart every service; no errors |
    | 3 | `2.0.0-rc.1` | `v2` |

    Then:
    - `go test -tags e2e ./internal/e2e/...`;
    - **`hack/ga-preflip-check.sh --cluster` must PASS with the rc tags present.**

    On a failure: fix PR (owner approval if it touches the flips) → `-rc.2` → phases 1–3 again. Record the final rc's SHA.
12. **[X] Record** (plan task 11), through a **docs-only** PR. It may merge after the rc, because `docs/` is `.dockerignore`d. Then `git checkout main && git pull`.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):**
  - the flip only deletes role checks where they live;
  - roles come from session-validate, never the JWT ([ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt));
  - `preview` and `inCohort` stay.
- **goose + sqlc:** **no migration** in the GA PR (if forced: expand only, proven by the new lint); **`sqlc diff` clean**. `2.0.0` carries **no contract**. Never edit a shipped migration.
- **Outbox/inbox and events:** unchanged. There is no new subject, stream or consumer, so **no NATS ACL PR**, and consumers-before-producers is not in play.
- **Frontend:** `theme.css` verbatim; the frozen boards (AB07–AB18, AB22) stay as they are. Only cohort-specific copy changes.
- **GitOps:**
  - **no `../infra` change in this sprint** (the widening is ga-02);
  - never `kubectl apply`/`edit`;
  - the `identity admin` CLI via `kubectl exec` is the sanctioned path, and every use is logged.
- **D34:** no alerting, opscheck, healthchecks.io, Flux Alert or push channel. Verification is by looking and by `host-verify --cluster`.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod or container, so it is unaffected.
- **Kill switches stay:** `JUDGE_BASE_URL`, the grading override, `LLM_PLATFORM_ENABLED` (code default `false`), `REVISION_ENTRY_RULE`, `SIGNUP_MODE`. **No `SIGNUP_MODE` change, no invite.**
- **Tags:**
  - only `v2.0.0-rc.N` (prerelease) in this sprint, **never `v2.0.0`**;
  - never move or re-push a tag;
  - after the rc, only `docs/`, `*.md` or `design-system/` may change on `main`; anything else (`.github/` included) needs a new rc.
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before the rc tag and before claiming an ADR number. No new ADR is expected; ADR-0034 gets at most dated amendment lines.
- **Privacy:** no stranger, tester or learner identity in any committed file. Counts only.

## Deliverables

- The strangers triaged on production; `ev-strangers` recorded with counts.
- The GA PR, merged with the owner's approval:
  - `.release-line = 2`;
  - the judge and AI cohort gates removed; go-concurrency `active`;
  - the GA-removal flags gone, and `COURSE_STATUS_OVERRIDE` decided;
  - `ga_flip_test.go` + `ga_rehearse_test.go`;
  - `hack/lint-migrations.sh --no-contract-since` + the `deploy.yml` first-of-major step;
  - `hack/ga-preflip-check.sh` + the offline self-test;
  - the docs drift fixes;
  - the release notes.
- `v2.0.0-rc.N`: a GitHub prerelease with 8 images, never deployed; the four-phase rehearsal green; the rc'd SHA recorded.
- A docs PR with the status record.

## Update status

- **The plan's Status table** ([`../sprints/sprint-ga-01.md`](../sprints/sprint-ga-01.md)): 🔄 → ✅ per task. While the PR waits for the owner, _Overall_ is 🔄 "awaiting owner review"; ✅ after task 11.
- **[`../status.md`](../status.md):**
  - the Sprint board row;
  - **Milestones: GA 🔄** (GA PR #, `v2.0.0-rc.N` = <sha>);
  - **release line:** `main` on 2 from <date>, with the hotfix rule;
  - the **flag inventory** (removals "ships in `v2.0.0`"; the `COURSE_STATUS_OVERRIDE` decision);
  - **owner events:** `ev-strangers` ✅ (counts);
  - the **CLI-use log**.
- **Decisions log:**
  - the widening covers the 8 fleet policies; runner and evalpack are separate streams;
  - the `COURSE_STATUS_OVERRIDE` choice;
  - the pre-flip script was written in ga-01 so the tag is on the rehearsed commit;
  - the `deploy.yml` no-contract step for a major's first stable release and its rc's;
  - the rc rehearsal result, R-b included.
- **ADRs:** at most a dated amendment line in ADR-0034 (§2 and/or §1.4). No new ADR number unless a notable call arises (check peers first).

## Done when (acceptance)

- [ ] No `active` `learner` account; `seats` 0/15 and 0 outstanding invites (recorded, counts only).
- [ ] The GA PR is merged **with the owner's explicit approval**:
  - `.release-line = 2`;
  - no role gate on judge or platform AI (kill switches and consents intact);
  - go-concurrency `active`;
  - the GA-removal flags gone;
  - the release notes carry the behaviour-change list.
- [ ] The no-contract lint mode and the `deploy.yml` first-of-major step are in place with self-tests. The rc's run showed the step green.
- [ ] `hack/ga-preflip-check.sh` is on `main`, its self-test is green, and it PASSES on the live state with the rc tags present.
- [ ] `v2.0.0-rc.N` is a prerelease with 8 images, and **nothing deployed**.
- [ ] The compose rehearsal is green on all four phases, **including R-b to `<last-1.x>`** and the kill-switch drills.
- [ ] git-strategy, the `deploy.yml` header and `api.md` match the GA procedure.

Ship per AGENT.md land-and-sync with **this sprint's release action: merge only**. The GA PR merges **only after the owner's explicit approval**; merging deploys nothing. Add the **`v2.0.0-rc.N` prerelease tag** (it builds and never deploys; **required here**, although GA checklist item 2 calls the rehearsal optional, because ga-02 tags the rc'd commit) and a docs-only status PR. **No `../infra` change and no `v2.0.0` tag** (those are [ga-02](../sprints/sprint-ga-02.md)). After merging, run `git checkout main && git pull` in `xlearn`, or `git -C <peer worktree> merge --ff-only origin/main` if a clean peer worktree holds `main`.
