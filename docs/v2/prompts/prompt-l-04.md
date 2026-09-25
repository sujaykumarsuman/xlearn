# Prompt — Sprint l-04 · Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root. The session runs the whole sprint, the L-exit rehearsal (`ev-l-rehearsal`) included (D40).
> **Plan:** [`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md)   ·   **Milestone:** L (L exit)   ·   **Prereqs:** [l-05](../sprints/sprint-l-05.md) + [l-03](../sprints/sprint-l-03.md) (merged), [m4-07](../sprints/sprint-m4-07.md) (`v1.16.0` live), [l-02](../sprints/sprint-l-02.md), [mi-04](../sprints/sprint-mi-04.md) (MI-5b live)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] **`ev-snap-v1.17.0`:** note the date of the last Hostinger weekly image, take a manual snapshot in hPanel (one at a time, 1-day retention) right before you launch, and put its time or id in your launch message. The tag lands the same day.
- [ ] **The rehearsal's GitHub identity:** a second GitHub account you own, **not linked to xLearn and with no verified email that belongs to an xLearn account** (else GitHub auto-links into that account, or a password account refuses it). Sign in to github.com with it in a separate browser profile the session can drive, and put its verified email in your launch message (it binds the GitHub-path invite; never committed).
- [ ] **Your xLearn sign-in,** in a browser profile the session can drive: after the tag the session passes your one-time acceptance step there (18+ and the notice ticked, the pre-filled region kept, your two AI consents left exactly as they are) and runs the smoke. Optional: without it, the step greets you at your next sign-in (recorded as your follow-up).

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md).
- The plan: [`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md). It has the runbook steps 0–8, the abort path, the PR A/B recipe and the release notes.
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md):
  - **§1** (L's exit test: a tester-operated round-trip on production, both paths, then `closed`; the optional `set-role tester` for a judge grade). Here you operate tester T1 yourself (D40);
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
2. cuts **`v1.17.0`**, which labels L-A and L-C. It is treated as an erase tag, so `host-verify` and the owner's before-launch snapshot come first;
3. passes the owner's one-time acceptance step, in his signed-in profile;
4. writes the **L-exit rehearsal runbook**;
5. then runs **`ev-l-rehearsal`** itself (D40): infra PR A (`SIGNUP_MODE=invite`, explicit `SEAT_CAP`) opens the door, you play tester T1 through one invite round-trip on production on both create paths, and PR B returns production to `closed`. The result is recorded in status.md. That record is [ga-01](../sprints/sprint-ga-01.md)'s entry gate.

v2 stays owner-only (D35), and the opening is v3.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.16.0` is live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`).
- [ ] l-03 and l-05 are merged on `main` (l-05's notice landed as drafted; its owner review is a v3 opening gate, D40). `git log --oneline v1.16.0..main` holds only what `v1.17.0` should carry.
- [ ] MI-5b is live: no admin console answers on `projects.sujaykumar.dev` (status.md MI-5b ✅).
- [ ] Tester web erase works end to end, and identity expects the `topology.go` ack set, judge included (`v1.13.0`).
- [ ] `host-verify --cluster` is available (mi-02).
- [ ] The launch message carries the snapshot's time or id and the second GitHub account's verified email (the before-launch block). **If the rehearsal identity is missing,** ship everything through the tag, record `ev-l-rehearsal` ⛔ "rehearsal identity missing" in status.md, and never open PR A.

## Do this (in order)

1. **[X] Branch** `feat/l-exit-erase-all` from an up-to-date `main`.
2. **[X] Web erase for every non-owner** (plan task 1):
   - identity: add `learner` to l-02's web-erase **allow-list constant** in `internal/identity/erase.go` (checked by `store.EraseAccount`'s `via=web` guard), so it reads `{learner, tester}`; the owner still gets `403 erase_owner_cli_only` (CLI only); L8 is unchanged. `GET /api/me`'s `erase_web` follows the same constant and must report `allowed` for a learner.
   - `Settings.tsx`: the erase section shows for learners without the tester-only note.
   - Tests: the handler matrix; `/api/me` `erase_web` per role; a `-tags e2e` learner erase created through an invite, with every `topology.go` ack (practice, review, assessment, coach, judge); an unaccepted learner's `DELETE /api/me` passes the gateway's acceptance gate (API level only: the SPA has no erase path while acceptance is required, per l-05); the Settings component tests.
   - Update `api.md`.
3. **[X] Runbook** (plan task 7): write `docs/v2/runbooks/l-exit-rehearsal.md` with preconditions, steps 0–8, verification (`identity admin erasures --since <event start>` shows both requests closed with every ack; `erasures --open` is empty), the abort path (CLI erase is `identity admin account erase <user> --confirm <user>`), privacy ("tester T1", counts only) and the R-d note. Every CLI line has the form `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin …'`.
4. **[X] Verify + compose dry run** (plan task 2):
   - **Notice truth check:** l-05's notice says erase is "in Settings, for every non-owner". Confirm step 2's change is on the commit you will tag (a learner's `/api/me` reports `erase_web: "allowed"`). If not, don't tag.
   - The full suite: `gofmt -l`, `go vet ./...`, `go test -race ./...`, `-tags e2e`, `sqlc diff`, the migration lint (no contract since `v1.16.0`), OpenAPI drift, and web typecheck/lint/test/build.
   - Compose with `SIGNUP_MODE: invite`: mint → sign up → accept → solve (self) → erase → `invite list` shows `erased` with an empty note, and `seats` is back. Fold the findings into the runbook.
   - **PR** (`feat(identity): web erase for every non-owner`, `docs: L-exit rehearsal runbook`) with the attribution lines → CI green → squash-merge.
5. **[H] `host-verify --cluster`** (plan task 3) over `ssh sujaykumar-vps`, per mi-02. It must be green and the host settled. Log the run in status.md.
6. **[O] Snapshot `ev-snap-v1.17.0`** (plan task 4) — **taken before launch**. Record the id or time and the weekly-image date from the launch message, and mark the event ✅. The tag lands the same day.
7. **[X] Tag `v1.17.0`** (plan task 5):
   - Run the §Release checklist from the plan. **Parallel-sessions check first:** `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents; the next free minor; major = `.release-line`.
   - Tag and create the release **`v1.17.0 — v2 build · L-A/L-C`** with the notes (including "every account passes the acceptance step once").
   - Verify by looking: healthz version, `get deploy -n xlearn` images, ImagePolicies' latest, HelmReleases Ready.
   - The extra smoke: `auth/config` = `closed`, no Sign up tab, `/xlearn/privacy` logged out, `seats` shows 0 invites. `identity admin seats` on production is a `kubectl exec` that writes `admin_audit` (m1-04 audits reads too), so it is a sanctioned manual path (rollout §2.2): you run it (approved by the launch, D40), and it goes in the CLI-use log.
8. **[H] The owner's acceptance** (plan task 6): in the owner's signed-in profile, pass the step for him (D40): 18+ and the notice agreement ticked, the pre-filled region kept, both AI consents left exactly as they show (his live grants). Then smoke login, the dashboard and coach, and record it. **No such profile →** record task 6 ⛔ "owner follow-up: pass the acceptance step at your next sign-in", and run the smoke's login and dashboard through T1 in step 10.
9. **[I] PR A** (plan task 8), opened at the start of the rehearsal window:
   - In `../infra` (up-to-date `main`), branch `chore/xlearn-signup-invite-rehearsal`, and edit **only the env block** of `apps/xlearn-identity.yaml`: `SIGNUP_MODE: invite` and `SEAT_CAP: "15"`, with comments.
   - Commit `chore(xlearn): SIGNUP_MODE=invite for the L-exit rehearsal (ev-l-rehearsal)` with the attribution lines.
   - `gh pr create` (not a draft). The body says it opens the L-exit rehearsal window and that PR B follows at once, and links the runbook. It merges in step 10.
10. **[H] `ev-l-rehearsal`** (plan tasks 9–10), run by you (D40):
    - rebase PR A if the IUA moved `main`, then merge it (the door opens);
    - run the runbook steps 1–6 as tester T1, reading back each CLI output: the email path with a throwaway, unused `@example.test` address and a random password you never print or record; the GitHub path in the owner's second-GitHub-account profile (the launch approves the OAuth authorization); the invite links stay in the browser and terminal only;
    - then **[I] PR B** from a fresh `../infra` `main` (`SIGNUP_MODE: closed`, keep `SEAT_CAP`), commit `chore(xlearn): SIGNUP_MODE=closed after the L-exit rehearsal`; merge it at once;
    - verify `auth/config` = `closed` and revoke leftovers.

    On any surprise, take the **abort path** (PR B first). **A tester step you can't perform yourself** (a tool or rule stops you, e.g. typing the password) is a failure of that path, never a wait: abort it, record it ⛔ "needs a human" with the exact steps in status.md → Open owner items, and carry on.
11. **[X] Record** (plan task 11):
    - the L rehearsal record in status.md (counts only; "tester T1"; PR A/B numbers; both paths ✅; seats baseline → +2 → baseline; notes empty; acks complete; the final mode `closed`; the snapshot the erases followed);
    - L exit ✅, milestone L ✅, `ev-l-rehearsal` ✅ (or the ⛔ path with its steps);
    - the CLI-use log.

    Land it through a small docs PR (see Ship).

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
  - `invite` **only** between PR A and PR B inside `ev-l-rehearsal`, attended by you, in this one session;
  - never `open`;
  - no invite other than the two rehearsal invites.
- **D34:** no alerting, opscheck or Flux Alert. Verification is by looking and by the CLI.
- **Memory-sum rule:** no new pod or container, so it is unaffected.
- **Snapshot:** the owner's, taken right before launch on a settled host (erase-tag line, D40); the tag lands the same day.
- **Parallel sessions:** peers' tags, PRs, worktrees and ListAgents are checked before the tag and before claiming an ADR number. No ADR is expected.
- **Privacy:** no tester name, email, handle or account id in any committed file (the repo is public).

## Deliverables

- identity + web: web erase for learners and testers, the owner refused; tests including the e2e learner erase.
- `docs/v2/runbooks/l-exit-rehearsal.md`.
- Tag `v1.17.0` (release notes), verified; snapshot id recorded; owner acceptance recorded (or ⛔ as his follow-up).
- `../infra` PR A and PR B, both opened and merged inside the rehearsal window.
- The L rehearsal record, L exit ✅ and milestone L ✅ in `docs/v2/status.md`.

## Update status

- The plan's Status table ([`../sprints/sprint-l-04.md`](../sprints/sprint-l-04.md)): tasks 🔄 → ✅; _Overall_ ✅ only after task 11 (🔄 while any rehearsal path is ⛔).
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
- [ ] The owner's acceptance step passed once (by you in his signed-in profile, or recorded ⛔ as his follow-up).
- [ ] The runbook is written; PR A and PR B were opened and merged only inside the rehearsal window.
- [ ] **L exit:** an invite round-trip rehearsed on production with tester T1 (you) on both create paths (redeem → accept → self-graded solve → web erase); seats back to baseline; the redeemed invites' `note` empty; production back to `SIGNUP_MODE=closed` (PR B merged, verified); recorded in status.md.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/l-exit-erase-all`, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched: the xlearn code + runbook PR (step 4), then — only after the tag is verified — the two `../infra` rehearsal PRs (steps 9–10), and a small docs PR for the record (step 11). Infra PRs are never folded into the tag.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. `../infra` has no CI: each rehearsal PR's check is its env-only diff and the post-merge `auth/config` read. PR A merges only to open the attended rehearsal window, and PR B right after it.
3. **Release action — tag `v1.17.0`, treated as an erase tag** (the next free minor): walk the release checklist (ADR-0034 §6, in the plan; the erase line is `host-verify --cluster` green plus the owner's before-launch snapshot), push the tag the same day as the snapshot, let Flux deploy, verify live by looking (step 7), then pass the owner's acceptance step (step 8) and run `ev-l-rehearsal` (step 10). Production ends at `SIGNUP_MODE=closed`.
4. Update status: the sprint file and `docs/v2/status.md` (step 11), in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in xlearn and `../infra`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
