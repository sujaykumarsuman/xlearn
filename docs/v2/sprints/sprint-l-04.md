# Sprint l-04 — Web erase for every non-owner → v1.17.0 (L-A/L-C) + L-exit rehearsal prep

> **Milestone:** L — learner gate (**L exit**) · **Track:** product · **Order:** 68
> **Prereqs:** [l-05](sprint-l-05.md) (merged; notice approved) · [l-03](sprint-l-03.md) (merged; the invite backend has shipped dark) · [m4-07](sprint-m4-07.md) (`v1.16.0` live) · [l-02](sprint-l-02.md) (web erase for testers, `v1.12.0`) · [mi-04](sprint-mi-04.md) (MI-5b live)
> **Unblocks:** [ga-01](sprint-ga-01.md) (its entry gate: "L exit recorded: `ev-l-rehearsal` done and `SIGNUP_MODE=closed` live again")
> **Release action:** **tag `v1.17.0`** (indicative: the next free minor, [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)). It is **treated as an erase tag**, since it widens web erase to every non-owner, so `host-verify --cluster` and a snapshot come first. It comes with **infra PR(s) for the rehearsal** in `../infra`. PR A (`SIGNUP_MODE=invite`) is opened as a **draft** and merged only inside `ev-l-rehearsal`. PR B (`SIGNUP_MODE=closed`) is opened right after PR A merges. Neither is ever folded into the tag.
> **Calendar:** December, right after [l-05](sprint-l-05.md) merges. Owner time:
> - a manual snapshot before the tag (~5 min; owner event **`ev-snap-v1.17.0`**, prepared by this sprint, like l-02's `ev-snap-v1.12.0`);
> - the acceptance step once after it (~1 min);
> - **`ev-l-rehearsal`** with a tester (~1 h, any day after the tag).
>
> **Execute with:** [`../prompts/prompt-l-04.md`](../prompts/prompt-l-04.md). One prompt, one session. The rehearsal may be a later sitting of the same sprint.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Web erase for every non-owner (identity + Settings) | X | ⬜ |
| 2 | Pre-tag verification + a compose dry run of the rehearsal (email path) | X | ⬜ |
| 3 | `host-verify --cluster` green (erase-tag line) | H | ⬜ |
| 4 | Manual Hostinger snapshot right before the tag (`ev-snap-v1.17.0`) | O | ⬜ |
| 5 | Tag `v1.17.0` + verify + record | X | ⬜ |
| 6 | Owner passes the acceptance step once | O | ⬜ |
| 7 | Rehearsal runbook `docs/v2/runbooks/l-exit-rehearsal.md` | X | ⬜ |
| 8 | Infra PR A (draft): `SIGNUP_MODE=invite` + explicit `SEAT_CAP: "15"` | I | ⬜ |
| 9 | `ev-l-rehearsal`: owner + tester invite round-trip on production | O | ⬜ |
| 10 | Infra PR B: `SIGNUP_MODE=closed`, opened and merged inside the event | I | ⬜ |
| 11 | Record the L rehearsal; L exit ✅ | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row; the **L milestone** (✅ only after task 11); the tag → floor → snapshot row; the flag inventory; the **L rehearsal record**; owner events.
> If the tag ships but the rehearsal hasn't happened yet, leave _Overall_ at 🔄 "awaiting `ev-l-rehearsal`".
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **`v1.16.0` live** ([m4-07](sprint-m4-07.md)), with `LLM_PLATFORM_ENABLED` on for the cohort.
- [ ] **[l-03](sprint-l-03.md) and [l-05](sprint-l-05.md) merged on `main`**, and `ev-notice-text` ✅. `git log v1.16.0..main` shows nothing that must not ship in `v1.17.0`.
- [ ] **MI-5b live** ([mi-04](sprint-mi-04.md)). The rehearsal creates learner accounts on production, and the first non-owner account needs the consoles off the xLearn origin ([ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b)).
- [ ] **Erase works for testers end to end on production:**
  - `v1.12.0`'s web erase ([l-02](sprint-l-02.md));
  - `v1.13.0`'s judge erase consumer ([m3-07](sprint-m3-07.md)), so identity expects 5 acks: practice, review, assessment, coach and judge (`topology.go`).
- [ ] **`host-verify --cluster` runnable** ([mi-02](sprint-mi-02.md)).
- [ ] **For the rehearsal (task 9), not for the tag:**
  - the owner is present (~1 h);
  - a **tester** is available with a browser, an **email address that is not an xLearn account**, and a **GitHub account that isn't linked to xLearn and whose verified emails don't belong to any xLearn account**. Otherwise GitHub auto-links into the existing account and the invite stays unused, or a password account refuses it.

## Goal

Finish the learner gate. Open **web erase to every non-owner account** ([ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt): testers from L-E, learners from the L exit; the owner never). Then cut **`v1.17.0`**, which labels L-A and L-C: the invite flow that l-03 shipped dark, and l-05's acceptance step, notice, 18+, region and consents.

Then **prepare and run the L exit** ([rollout §4](../rollout-plan.md#4-per-milestone-detail) L "Exit"; [ADR-0033 §1](../../adr/0033-invite-only-admission-and-owner-admin.md#1-audience-owner-only-v2-invite-flow-built-owner-d35)): an **invite round-trip on production with a tester, on both create paths**, after which production goes **back to `SIGNUP_MODE=closed`**, and the result is recorded in [`../status.md`](../status.md). That record is [ga-01](sprint-ga-01.md)'s entry gate. The opening itself stays v3 (D35).

## Scope

**In**
- identity + web: `DELETE /api/me` allowed for `learner` and `tester`; the owner is still refused (CLI only). Settings shows the erase section to learners.
- Tag `v1.17.0` (the erase-tag lines of the checklist included) and the owner's one-time acceptance.
- `docs/v2/runbooks/l-exit-rehearsal.md`.
- `../infra` PR A (draft) and PR B (opened in the event) on `apps/xlearn-identity.yaml`.
- `ev-l-rehearsal` (owner + tester) and its record in [`../status.md`](../status.md).

**Out**
- **Keeping `SIGNUP_MODE=invite` beyond the rehearsal.** That is the v3 opening ([rollout §11](../rollout-plan.md#11-opening-gates-v3); `ev-v3-opening`).
- Minting any invite other than the two rehearsal invites. Re-sizing `SEAT_CAP` (it needs ≥ 2 weeks of M4 data and is an opening gate).
- Stranger triage (no `active` learner before GA) → [ga-01](sprint-ga-01.md) (`ev-strangers`). The rehearsal only compares seat counts with their baseline.
- The GA flip → [ga-01](sprint-ga-01.md) / [ga-02](sprint-ga-02.md).
- Any alerting on the rehearsal (D34). It is watched live and verified with the CLI.

## Tasks

### 1 · Web erase for every non-owner [X]

Sources: [ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md#7-roles-and-status-live-in-identitys-database-never-in-the-jwt) (the web-erase column), [§9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls), §12 row 12; [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L8; [ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) ("testers only until the L exit").
- **identity:** l-02 defines the web-erase eligibility as a **code constant allow-list** in `internal/identity/erase.go`, checked by `store.EraseAccount`'s `via=web` guard (`{tester}` at L-E; not an env flag). Add `learner` to it, so it reads `{learner, tester}`. `owner` → the same `403 erase_owner_cli_only` as before (the CLI `identity admin account erase` stays the only owner path), and any role outside the list keeps `403 erase_not_available`. A suspended account has no session, so it is CLI only as before.
  - L8 is unchanged: typed confirmation, a session under 5 minutes old, and one open request per account.
  - **`GET /api/me`'s `erase_web`** (l-02) is derived from the same constant, so it must now report `allowed` for a learner. That field drives which AB21 frame Settings renders.
- **web:** `web/src/screens/Settings.tsx`'s erase section (AB21) renders for learners too, without AB21's "testers only for now" note. The owner keeps AB21's owner-refused state.
- **Acceptance interplay (API level only):** `DELETE /api/me` is in [l-05](sprint-l-05.md)'s exempt set as a safeguard, so the acceptance gate never blocks an erase request. The SPA has no path to it while acceptance is required (AuthedShell redirects to `/auth`; AB19 offers only Sign out), as l-05's decisions log records, so test it at the API only, not as a user flow.
- **Tests:**
  - identity handler matrix: learner ✅, tester ✅, owner `403 erase_owner_cli_only`, a stale session → `403 reauth_required`; flip l-02's `EraseAccount` guard test so `learner` via web is now allowed;
  - `GET /api/me` `erase_web`: `allowed` for learner and tester, `owner_cli_only` for the owner;
  - gateway: an unaccepted learner session's `DELETE /api/me` passes the acceptance gate (not 403 `acceptance_required`);
  - a `-tags e2e` learner erase: the account is created via an invite in `invite` mode, erased on the web, **every ack expected by `topology.go` arrives** (practice, review, assessment, coach, judge) and identity closes the request; the profile 404s; the username is held;
  - `Settings.test.tsx`: the learner (`erase_web: "allowed"`) sees the section, the owner sees the refusal.
- **Docs:** `docs/architecture/api.md` (the `DELETE /api/me` eligibility row).

### 2 · Pre-tag verification [X]

- `git log --oneline v1.16.0..main`: write the release notes from it. They cover:
  - the invite flow (dark since the tag that carried l-03);
  - **the acceptance step, which every account passes once, the owner included**;
  - the privacy notice, 18+, region and the two consents;
  - web erase for every non-owner;
  - "production stays `SIGNUP_MODE=closed`".
- **Notice truth check:** l-05's privacy notice says erase is "in Settings, for every non-owner". Confirm task 1 is merged on the commit you will tag and that a learner's `/api/me` reports `erase_web: "allowed"` in compose. If task 1 isn't in the tag, the notice would claim something unbuilt: don't tag.
- Full verify: `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e ./internal/e2e/...`, `sqlc diff`, the migration lint (no `-- xlearn:contract` since `v1.16.0`), the OpenAPI drift test, and web typecheck/lint/test/build.
- **Compose dry run of the rehearsal, email path.** GitHub needs the real OAuth app; l-03's fake-provider tests cover that path. With a `SIGNUP_MODE: invite` override:
  1. `docker compose exec identity identity admin invite create --ttl 1d --note "dry run"`;
  2. open the link, sign up, accept, pick DSA, and log a self-graded outcome on a problem;
  3. Settings → erase;
  4. `invite list` shows `erased` with an empty note, and `seats` is back to baseline.

  Fold anything learned into the runbook (task 7).

### 3 · `host-verify --cluster` [H]

`v1.17.0` opens web erase to every non-owner, and the rehearsal's learner erases follow it. So it is **treated as an erase tag** ([ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule), §6). Record the call in the decisions log.

Run the MI-8 `host-verify --cluster` over `ssh vps`, as [mi-02](sprint-mi-02.md) documents. It must be green: memory sum, Flux Ready, no OOMKills, PVCs < 60%, NATS `auth_required` with no `legacy`, and the NetworkPolicies present. The host must have settled, with no restart-inducing change in flight.

### 4 · Snapshot [O]

Owner event **`ev-snap-v1.17.0`** (~5 min). Right before the tag, the owner:
- notes the date of the last Hostinger weekly image;
- takes a **manual snapshot** in hPanel;
- gives the agent the snapshot id or time for status.md, and `ev-snap-v1.17.0` is marked ✅.

**R-d note:** the rehearsal's erases (task 9) happen **after** this snapshot. An R-d restore would resurrect those accounts, so the runbook's R-d steps list them (identity's `erase_request` rows) and re-run them through the CLI ([ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button) step 3).

### 5 · Tag `v1.17.0` [X]

Run the release checklist (§Release). The GitHub release title is **`v1.17.0 — v2 build · L-A/L-C`**, with the task 2 notes. After the tag, run the verification checklist and the extra smoke:
- `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/auth/config` → `{"signup":"closed"}`;
- the auth page shows the invite-only state with **no Sign up tab** and the `mailto:` link;
- `/xlearn/privacy` loads logged out;
- `identity admin seats` → `outstanding invites 0`. This runs on production through `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin seats'`, and m1-04 audits every verb, reads included, so it writes an `admin_audit` row. It is a **sanctioned manual path** ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)), not a read-only check: the **owner** runs it (or the agent, only with the owner's go-ahead in the session), and it is logged in status.md's CLI-use log.

Record in [`../status.md`](../status.md):
- milestone L-A/L-C → tag `v1.17.0` → rollback floor **unchanged** (copy the current floor; no contract) → snapshot id;
- gate state `closed`;
- the flag inventory: `SIGNUP_MODE` is an operating mode with values `closed | invite | open (dev only)`, `closed` in production; `SEAT_CAP` = 15 (code default).

### 6 · Owner acceptance [O]

After the tag is verified, the owner signs in and lands on the acceptance step:
- ticks 18+ and the privacy-notice agreement (AB19-F9's two required boxes);
- confirms the region;
- sets the two AI consents as he wishes. They show his live M4 Settings grants, and an untick withdraws (l-05's flagged deviation from AB19-F9/F11).

Then the smoke continues: login, dashboard and coach (the checklist line). Testers pass it on their next sign-in. Record "owner accepted `privacy-notice@1` on <date>" in status.md.

### 7 · Rehearsal runbook [X]

Write `docs/v2/runbooks/l-exit-rehearsal.md`. Every CLI call has the form `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin …'` and is logged in status.md as a sanctioned manual path ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).

0. **Preconditions.**
   - `v1.17.0` is live and the owner has accepted.
   - Record the baseline, as **counts only**: `identity admin seats` and `identity admin account list --role learner --status active`. The baseline is 0 if the strangers have already been triaged; otherwise note N. `ga-01` owns the triage.
   - `invite list --status outstanding` is empty.
   - The tester is ready (entry gate). Note the time. No snapshot is needed, since no tag is involved; task 4's covers it.
1. **Open the door.** Mark PR A ready and merge it. Wait for Flux, and check the env with `ssh vps 'k3s kubectl -n xlearn get deploy xlearn-identity -o jsonpath=…'`. Confirm `GET /xlearn/api/v1/auth/config` → `{"signup":"invite"}`.
2. **Mint two invites** with a short TTL:
   - `invite create --ttl 1d --note "L rehearsal · email path"`;
   - `invite create --ttl 1d --email <the tester's GitHub-verified email> --note "L rehearsal · GitHub path"` (this also rehearses the `intended_email` binding);
   - optionally `--region` on one of them, to see the pre-fill.

   `seats` = baseline + 2. The owner sends both links over his own channel (xLearn sends no email).
3. **Email path** (the tester, in a private window):
   1. Open link 1. The address bar shows `/xlearn/auth` without `#invite=`, with Sign up open and "valid until …".
   2. Sign up with the non-xLearn email.
   3. The acceptance step: 18+, the privacy-notice agreement, region, **leave both AI boxes unticked**, [Continue].
   4. Onboarding: pick DSA.
   5. Open a problem, attempt it, and log a **self-graded** outcome. judge stays cohort-gated for a `learner` before GA.
   6. Settings → Delete account: type the confirmation, and "sign in again" if the session is over 5 minutes old.
   7. The browser is signed out, and `/xlearn/u/<username>` 404s if a username was claimed.
4. **GitHub path** (a new private window):
   1. Open link 2 → Continue with GitHub (the tester's account). The account is created.
   2. Acceptance, then onboarding, then a self-graded solve.
   3. **Optional:** the owner runs `account set-role <that user> tester`. That frees the seat, and the tester re-submits a problem to rehearse a **judge grade** (tester cohort).
   4. Delete the account on the web.
5. **Negative checks** (a minute; skip any the tester can't do):
   - reopen link 1 → `invite_invalid`;
   - a spare GitHub account with no invite → `/auth?error=invite_required`.
6. **Verify**, all with the CLI:
   - `invite list` → both `redeemed`, the redeemer `erased`, **`note` empty**;
   - `seats` back to **baseline** (0/15 if the strangers are triaged);
   - `account list --role learner --status active` equals the baseline;
   - both erase requests are closed with all `topology.go` acks: `identity admin erasures --since <the event's start>` (l-01) lists both as closed with every expected ack, and `identity admin erasures --open` is empty.
7. **Close the door.** Open PR B from a fresh `../infra` `main` (task 10), merge it, and wait for Flux. Confirm `GET /xlearn/api/v1/auth/config` → `{"signup":"closed"}` and that the auth page shows invite-only. `invite revoke` any invite left outstanding.
8. **Record** (task 11).

**Abort path.** On any failure or surprise:
- merge PR B **at once**;
- `invite revoke` every outstanding invite;
- CLI-erase any half-created rehearsal account (`identity admin account erase <user> --confirm <user>`, l-02; the `--confirm` value repeats `<user>`);
- record what happened;
- fix the problem (a patch tag if the code is at fault);
- reschedule.

Production must never be left in `invite` unattended.

**Privacy of the record.** The record carries no tester name, email, GitHub handle or account id. Write "tester T1". The repo is public.

**R-d note.** After the rehearsal, an R-d restore to task 4's snapshot resurrects the two rehearsal accounts. Before any restore, list the erases from identity's `erase_request`, and re-run them through the CLI afterwards (ADR-0034 §4.2).

### 8 · Infra PR A (draft) [I]

In `../infra`:
- branch `chore/xlearn-signup-invite-rehearsal`;
- edit only `apps/xlearn-identity.yaml`'s `env`:
  - `SIGNUP_MODE` → `invite`;
  - add `SEAT_CAP: "15"` with a comment. It is ADR-0033 §3's "capacity as an auditable number in git"; the R0 seat freeze lowers it.
  - Update the existing comment to point at `ev-l-rehearsal`.
- **Never touch the image tag line**, which the IUA rewrites.
- Commit `chore(xlearn): SIGNUP_MODE=invite for the L-exit rehearsal (ev-l-rehearsal)`.
- Open it as a **draft PR**. The body says "merge only inside `ev-l-rehearsal`; PR B follows immediately" and links the runbook.
- Rebase it just before the event: the IUA keeps committing to this file's tag line.
- No `kubectl apply`: the rollout is the env change through Flux.

### 9 · `ev-l-rehearsal` [O]

The owner and the tester run runbook steps 0–7. The agent, if present, reads the CLI output back and opens PR B.

### 10 · Infra PR B [I]

**Opened only after PR A has merged**, from a fresh `../infra` `main`:
- branch `chore/xlearn-signup-closed-after-rehearsal`;
- `SIGNUP_MODE` → `closed`, keeping `SEAT_CAP: "15"`;
- commit `chore(xlearn): SIGNUP_MODE=closed after the L-exit rehearsal`.

The runbook holds the exact commands, and the owner merges it at once.

It is **not** prepared as a PR stacked on A. With squash merges, a revert stacked on A's branch nets to zero against `main` once A is squashed, and merging it would leave production at `invite`.

### 11 · Record + L exit [X]

Add an **L rehearsal record** row to [`../status.md`](../status.md):
- the date and the tag (`v1.17.0`); PR A # and PR B #;
- email path ✅ and GitHub path ✅: each redeemed → accepted → self-graded solve → erased on the web;
- the optional judge grade;
- seats baseline → +2 → baseline;
- the invite notes empty ✅;
- the erase acks complete ✅;
- the final mode `closed`, verified by the config endpoint;
- the snapshot the erases followed.

Set **L exit ✅** and **milestone L ✅**, which unblocks [ga-01](sprint-ga-01.md). Mark owner event `ev-l-rehearsal` ✅. Log each CLI use.

## Acceptance criteria

- [ ] **Web erase:** a learner and a tester can erase on the web (typed confirmation, fresh session). The owner is refused (403) and the CLI still works. Every ack that `topology.go` expects arrives.
- [ ] **`v1.17.0` tagged and verified** by the checklist. `host-verify --cluster` was green and the snapshot id is recorded. `auth/config` = `closed`, there is no Sign up tab on production, and `/xlearn/privacy` loads.
- [ ] **The owner passed the acceptance step once** (recorded).
- [ ] **The runbook is written; PR A is open as a draft**, unmerged until the event.
- [ ] **L exit:**
  - an invite round-trip was rehearsed on production with a tester on **both** create paths, and each account accepted, solved on the self path and erased itself on the web;
  - seats are back to baseline, and the redeemed invites show `note` empty;
  - **production is back to `SIGNUP_MODE=closed`** (PR B merged, config verified);
  - it is **recorded in status.md**.

## Release

**Tag `v1.17.0`** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): "L-A: invite flow, acceptance, notice, 18+, region, consents; web erase opens to every non-owner account. An invite round-trip is rehearsed on prod with a tester, then `SIGNUP_MODE` returns to `closed`". Gate state after: `closed`; rollback floor: unchanged). Take the next free minor at tag time.
Release checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) §6, verbatim, plus the ADR-0035 §2 standing rule):

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
- **ACL PRs:** none. There is no new stream, consumer or subject; `account_created.tier` is a field.
- **New service:** none.
- **Contract:** none. l-03's migration is expand only, and the lint confirms no `-- xlearn:contract` since `v1.16.0`.
- **Erase:** **yes, treated as an erase tag** (tasks 3–4).
- **M6:** n/a.
- **New in-cluster callers:** none. gateway → identity and judge → identity already exist, so no NetworkPolicy PR is needed. There is no new pod, so the memory sum ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)) is unchanged.
- **Smoke:** the smoke runs after the owner's acceptance (task 6), plus task 5's extra checks.
- **Rollback:** R-b to `v1.16.0` stays above the floor (no contract).
- **Infra PRs:** A and B are their own tasks (8, 10), **never folded into the tag**, and neither is merged outside `ev-l-rehearsal`.

## Definition of Done

`v1.17.0` shipped by Flux (no hand `kubectl`) and verified by the checklist · CI green · the owner's acceptance recorded · the runbook merged (in the tag's PR or a docs PR) · PR A open as a draft · **`ev-l-rehearsal` done, PR B merged, production `closed` and verified, and the L rehearsal record written**. Until then, the sprint stays 🔄 "awaiting `ev-l-rehearsal`" · statuses updated:
- this file;
- [`../status.md`](../status.md): the board, **L milestone ✅**, tag → floor → snapshot, the flag inventory, the L rehearsal record, owner events, and the CLI-use log.

Decisions log: `v1.17.0` treated as an erase tag (with `ev-snap-v1.17.0`), PR B not stacked, and the counts-only record.

## Risks / watch-outs

- **Rehearsal accounts must erase themselves.** Verify the seat count is back to baseline with `identity admin seats`, and check `account list`. If a web erase fails, CLI-erase the account and record why.
- **Production at `invite` is a live door.** Use one sitting, `--ttl 1d` invites, revoke any leftovers, and merge PR B right after. The abort path puts production back to `closed` first and investigates second.
- **The GitHub auto-link trap.** If the tester's GitHub email belongs to an OAuth-only xLearn account, GitHub signs into that account and the invite stays unused. If it belongs to a password account, it is refused (`account_exists_password`). Check before the event (entry gate).
- **The baseline isn't 0 while strangers remain.** Pre-v1.5.2 `learner` accounts count as seats. Compare with the baseline, not with 0. Triage belongs to [ga-01](sprint-ga-01.md).
- **The acceptance step greets the owner after the tag.** It is expected and in the release notes. The smoke can't pass until he accepts.
- **The stacked-revert trap** (PR B). Always open PR B from a fresh `main` after A merges.
- **An R-d restore after the rehearsal resurrects the erased accounts.** Follow the runbook's R-d note.
- **The IUA edits `apps/xlearn-identity.yaml`** (the tag line). Rebase PR A before merging it, and never edit the tag line.
- **No tester PII in `docs/v2/status.md`**; the repo is public.
