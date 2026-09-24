# Prompt — Sprint l-04 · Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root. The owner-and-tester rehearsal (`ev-l-rehearsal`) may run as a later sitting of this same sprint.
> **Plan:** [`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md)   ·   **Milestone:** L (L exit)   ·   **Prereqs:** [l-05](../sprints/sprint-l-05.md) + [l-03](../sprints/sprint-l-03.md) (merged), [m4-07](../sprints/sprint-m4-07.md) (`v1.16.0` live), [l-02](../sprints/sprint-l-02.md), [mi-04](../sprints/sprint-mi-04.md) (MI-5b live)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md). It has the runbook steps 0–8, the abort path, the PR A/B recipe and the release notes.
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md):
  - **§1** (L's exit test: a tester-operated round-trip on production, both paths, then `closed`; the optional `set-role tester` for a judge grade);
  - **§7** (the web-erase column: learners from the L exit, the owner never);
  - **§9** (erase; clearing `note`);
  - **§11** (MI-5b before any non-owner account);
  - §3 (`SIGNUP_MODE`, `SEAT_CAP` as HelmRelease env; the R0 seat freeze).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):
  - §1.6 (the `v1.17.0` row);
  - **§4.2** (the R-d procedure, erases since the snapshot);
  - **§4.3** (the snapshot rule);
  - §4.4 (the L-E and L-A rows);
  - **§6** (the release checklist).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L7, L8) and §2 (the NetworkPolicy standing rule).
- [rollout §4](../rollout-plan.md#4-per-milestone-detail) (L "Exit" steps 1–5), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.17.0`), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (operating rules; sanctioned manual paths), [§11](../rollout-plan.md#11-opening-gates-v3) (what stays for v3).
- The plans of [l-02](../sprints/sprint-l-02.md) (the web erase rules), [l-03](../sprints/sprint-l-03.md) (the invite CLI, error codes, seats), [l-05](../sprints/sprint-l-05.md) (acceptance, the exempt set) and [ga-01](../sprints/sprint-ga-01.md) (which gates on this sprint's record).
- Code and infra:
  - identity's `DELETE /me` handler and erase transaction (l-02), `web/src/screens/Settings.tsx`, `internal/e2e/`, `docs/runbooks/identity-admin.md`;
  - `../infra/apps/xlearn-identity.yaml` (env block; **never** the image tag line); `../infra/hack/host-verify.sh` (`--cluster`, per [mi-02](../sprints/sprint-mi-02.md)).

## Context

`v1.16.0` (M4) is live. [l-03](../sprints/sprint-l-03.md)'s invite backend shipped dark in an M3 tag, and [l-05](../sprints/sprint-l-05.md) merged the front door: the invite-aware auth page, the acceptance step, `403 acceptance_required`, the privacy notice and the consents. Production is `SIGNUP_MODE=closed`. Web erase is open to testers only ([l-02](../sprints/sprint-l-02.md)).

This sprint:
1. opens web erase to every non-owner;
2. cuts **`v1.17.0`**, which labels L-A and L-C. It is treated as an erase tag, so `host-verify` and a snapshot come first;
3. has the owner pass the acceptance step once;
4. writes the **L-exit rehearsal runbook** and a **draft** infra PR A (`SIGNUP_MODE=invite`, explicit `SEAT_CAP`);
5. then, in the owner-and-tester event **`ev-l-rehearsal`**, runs one invite round-trip on production on both create paths. PR B returns production to `closed`, and the result is recorded in status.md. That record is [ga-01](../sprints/sprint-ga-01.md)'s entry gate.

v2 stays owner-only (D35), and the opening is v3.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.16.0` is live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`).
- [ ] l-03 and l-05 are merged on `main`, and `ev-notice-text` is ✅ in `docs/v2/status.md`. `git log --oneline v1.16.0..main` holds only what `v1.17.0` should carry.
- [ ] MI-5b is live: no admin console answers on `projects.sujaykumar.dev` (status.md MI-5b ✅).
- [ ] Tester web erase works end to end, and identity expects the `topology.go` ack set, judge included (`v1.13.0`).
- [ ] `host-verify --cluster` is available (mi-02).
- [ ] **Before task 9 only:** the owner is present, and a tester has a browser, an email that isn't an xLearn account, and a GitHub account that isn't linked to xLearn and whose verified emails aren't xLearn accounts. If they aren't available, do tasks 1–8 and 7's runbook, ship the tag, and leave the sprint 🔄 "awaiting `ev-l-rehearsal`".

## Do this (in order)

1. **[X] Branch** `feat/l-exit-erase-all` from an up-to-date `main`.
2. **[X] Web erase for every non-owner** (plan task 1):
   - identity: add `learner` to l-02's web-erase **allow-list constant** in `internal/identity/erase.go` (checked by `store.EraseAccount`'s `via=web` guard), so it reads `{learner, tester}`; the owner still gets `403 erase_owner_cli_only` (CLI only); L8 is unchanged. `GET /api/me`'s `erase_web` follows the same constant and must report `allowed` for a learner.
   - `Settings.tsx`: the erase section shows for learners without the tester-only note.
   - Tests: the handler matrix; `/api/me` `erase_web` per role; a `-tags e2e` learner erase created through an invite, with every `topology.go` ack (practice, review, assessment, coach, judge); an unaccepted learner's `DELETE /api/me` passes the gateway's acceptance gate (API level only: the SPA has no erase path while acceptance is required, per l-05); the Settings component tests.
   - Update `api.md`.
3. **[X] Runbook** (plan task 7): write `docs/v2/runbooks/l-exit-rehearsal.md` with preconditions, steps 0–8, verification (`identity admin erasures --since <event start>` shows both requests closed with every ack; `erasures --open` is empty), the abort path (CLI erase is `identity admin account erase <user> --confirm <user>`), privacy ("tester T1", counts only) and the R-d note. Every CLI line has the form `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin …'`.
4. **[X] Verify + compose dry run** (plan task 2):
   - **Notice truth check:** l-05's notice says erase is "in Settings, for every non-owner". Confirm step 2's change is on the commit you will tag (a learner's `/api/me` reports `erase_web: "allowed"`). If not, don't tag.
   - The full suite: `gofmt -l`, `go vet ./...`, `go test -race ./...`, `-tags e2e`, `sqlc diff`, the migration lint (no contract since `v1.16.0`), OpenAPI drift, and web typecheck/lint/test/build.
   - Compose with `SIGNUP_MODE: invite`: mint → sign up → accept → solve (self) → erase → `invite list` shows `erased` with an empty note, and `seats` is back. Fold the findings into the runbook.
   - **PR** (`feat(identity): web erase for every non-owner`, `docs: L-exit rehearsal runbook`) with the attribution lines → CI green → squash-merge.
5. **[H] `host-verify --cluster`** (plan task 3) over `ssh vps`, per mi-02. It must be green and the host settled. Log the run in status.md.
6. **[O] Snapshot `ev-snap-v1.17.0`** (plan task 4): ask the owner to note the last weekly image date and take the manual Hostinger snapshot **right before** the tag. Record the id or time and mark the event ✅.
7. **[X] Tag `v1.17.0`** (plan task 5):
   - Run the §Release checklist from the plan. **Parallel-sessions check first:** `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents; the next free minor; major = `.release-line`.
   - Tag and create the release **`v1.17.0 — v2 build · L-A/L-C`** with the notes (including "every account passes the acceptance step once").
   - Verify by looking: healthz version, `get deploy -n xlearn` images, ImagePolicies' latest, HelmReleases Ready.
   - The extra smoke: `auth/config` = `closed`, no Sign up tab, `/xlearn/privacy` logged out, `seats` shows 0 invites. `identity admin seats` on production is a `kubectl exec` that writes `admin_audit` (m1-04 audits reads too), so it is a sanctioned manual path (rollout §2.2): the **owner** runs it, or you do only with the owner's go-ahead, and it goes in the CLI-use log.
8. **[O] Owner acceptance** (plan task 6): the owner signs in and passes the acceptance step once. Then smoke login, the dashboard and coach, and record it.
9. **[I] PR A (draft)** (plan task 8):
   - In `../infra` (up-to-date `main`), branch `chore/xlearn-signup-invite-rehearsal`, and edit **only the env block** of `apps/xlearn-identity.yaml`: `SIGNUP_MODE: invite` and `SEAT_CAP: "15"`, with comments.
   - Commit `chore(xlearn): SIGNUP_MODE=invite for the L-exit rehearsal (ev-l-rehearsal)` with the attribution lines.
   - `gh pr create --draft`. The body says to merge only inside `ev-l-rehearsal` and that PR B follows at once, and links the runbook.
   - **Do not merge it.**
10. **[O] `ev-l-rehearsal`** (plan tasks 9–10), with the owner and tester present:
    - rebase PR A, mark it ready, and the owner merges it;
    - run the runbook steps 1–6, reading back each CLI output;
    - then **[I] PR B** from a fresh `../infra` `main` (`SIGNUP_MODE: closed`, keep `SEAT_CAP`), commit `chore(xlearn): SIGNUP_MODE=closed after the L-exit rehearsal`; the owner merges it at once;
    - verify `auth/config` = `closed` and revoke leftovers.

    On any surprise, take the **abort path** (PR B first).
11. **[X] Record** (plan task 11):
    - the L rehearsal record in status.md (counts only; "tester T1"; PR A/B numbers; both paths ✅; seats baseline → +2 → baseline; notes empty; acks complete; the final mode `closed`; the snapshot the erases followed);
    - L exit ✅, milestone L ✅, `ev-l-rehearsal` ✅;
    - the CLI-use log.

    Land it through a small docs PR (squash-merge).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the erase eligibility lives in identity. The web only reflects it, and the gateway never decides roles from a JWT (ADR-0033 §7).
- **goose + sqlc:** no migration is expected. If one is needed, it is expand only; **`sqlc diff` clean**. No contract: this is not a contract tag.
- **Outbox / erase:** erase keeps l-02's transaction (the outbox `account_erasure_requested`) and l-03's note clearing. Every consumer acks per `topology.go`, and nothing changes in the event flow, so **no NATS ACL PR** is needed and consumers-before-producers is already satisfied (the consumers shipped in `v1.11.0`/`v1.13.0`).
- **Frontend:** `theme.css` verbatim; AB21 as frozen.
- **GitOps:**
  - infra changes go only through PR A and PR B, each its own task, **never folded into the tag**;
  - never `kubectl apply` or `kubectl edit`;
  - **never edit the image tag line** in `apps/xlearn-identity.yaml`;
  - never suspend the shared IUA.

  The `identity admin` CLI through `kubectl exec` is the sanctioned manual path; log each use.
- **Production `SIGNUP_MODE`:**
  - `invite` **only** between PR A and PR B inside `ev-l-rehearsal`, attended, one sitting;
  - never `open`;
  - no invite other than the two rehearsal invites.
- **D34:** no alerting, opscheck or Flux Alert. Verification is by looking and by the CLI.
- **Memory-sum rule:** no new pod or container, so it is unaffected.
- **Snapshot:** right before the tag (erase-tag line), only once the host has settled.
- **Parallel sessions:** peers' tags, PRs, worktrees and ListAgents are checked before the tag and before claiming an ADR number. No ADR is expected.
- **Privacy:** no tester name, email, handle or account id in any committed file (the repo is public).

## Deliverables

- identity + web: web erase for learners and testers, the owner refused; tests including the e2e learner erase.
- `docs/v2/runbooks/l-exit-rehearsal.md`.
- Tag `v1.17.0` (release notes), verified; snapshot id recorded; owner acceptance recorded.
- `../infra` PR A (draft → merged in the event) and PR B (opened and merged in the event).
- The L rehearsal record, L exit ✅ and milestone L ✅ in `docs/v2/status.md`.

## Update status

- The plan's Status table ([`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md)): tasks 🔄 → ✅; _Overall_ ✅ only after task 11. Until then it is 🔄 "awaiting `ev-l-rehearsal`".
- [`../status.md`](../status.md):
  - the Sprint board row;
  - **Milestones: L ✅** (the exit criterion met, with the date);
  - **milestone → tag → floor → snapshot:** L-A/L-C → `v1.17.0` → floor unchanged → the snapshot id;
  - the **flag inventory:** `SIGNUP_MODE` is an operating mode (`closed` in production; `invite` only during the rehearsal, with its window's timestamps); `SEAT_CAP` 15 is now explicit in git;
  - the **L rehearsal record**;
  - owner events: `ev-snap-v1.17.0` ✅ (with the snapshot id), `ev-l-rehearsal` ✅; the owner's acceptance of `privacy-notice@1`;
  - the CLI-use log;
  - the gate state after `v1.17.0`: `closed`.
- Decisions log: `v1.17.0` treated as an erase tag (with `ev-snap-v1.17.0`); PR B not stacked (the squash-merge trap); the counts-only record; the rehearsal invites minted with `--ttl 1d`.

## Done when (acceptance)

- [ ] Learners and testers can erase on the web and the owner is refused; every `topology.go` ack arrives.
- [ ] `v1.17.0` is tagged and verified; `host-verify --cluster` was green and the snapshot is recorded; `auth/config` = `closed` with no Sign up tab; `/xlearn/privacy` loads.
- [ ] The owner passed the acceptance step once.
- [ ] The runbook is written; PR A was a draft until the event.
- [ ] **L exit:** an invite round-trip rehearsed on production with a tester on both create paths (redeem → accept → self-graded solve → web erase); seats back to baseline; the redeemed invites' `note` empty; production back to `SIGNUP_MODE=closed` (PR B merged, verified); recorded in status.md.

Ship per AGENT.md land-and-sync with **this sprint's release action: tag `v1.17.0`** (erase-tag lines: `host-verify --cluster` green and a snapshot first), **plus infra PR(s) for the rehearsal**. PR A stays a draft until `ev-l-rehearsal`, and PR B is opened and merged inside it. Neither is folded into the tag, and neither is ever merged outside the attended event. After merging the code PR, tagging and verifying, run `git checkout main && git pull` in **both** `xlearn` and `../infra`.
