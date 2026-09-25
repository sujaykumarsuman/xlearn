# Prompt — Sprint mi-06 · NATS auth server-first: N1 → N2 → N3 (MI-7)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-06.md`](../sprints/sprint-mi-06.md) · **Milestone:** MI (step MI-7; hard precondition for M3 and L-E) · **Prereqs:** [mi-05](../sprints/sprint-mi-05.md), [m1-02](../sprints/sprint-m1-02.md) (`v1.6.0`), [mi-03](../sprints/sprint-mi-03.md) (MI-5), [mi-02](../sprints/sprint-mi-02.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Generate the NATS **ops** nkey offline (`nk -gen user -pubout`). Keep the seed with the two offline age-key copies from MI-1 (`ev-mi1`; make them first if they don't exist yet), never in git, the cluster or a transcript, and give the session only its public key (`U…`) in the launch message.
- [ ] Read the date of the last Hostinger weekly image in hPanel and record it in `status.md` (or give it in the launch message). N1 restarts NATS, so it must be ≤ 7 days old.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, GitOps, land-and-sync.
- [`../sprints/sprint-mi-06.md`](../sprints/sprint-mi-06.md) — the plan, with the exact values snippets for N1, N2 and N3.
- [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) — the model (nkey users in `$G`, public keys plaintext, seeds in `apps/secrets`), the ACL table, the ops identity and break-glass, the N0–N4 rollout table, the standing rule. §1 has the v2 topology that drives the ACLs.
- [ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) (NATS N1–N3 row) and [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist).
- [Rollout §2](../rollout-plan.md#2-mi-infra-track) (MI-6, MI-7, MI-8), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (server before clients, verify the host, snapshots, break-glass), [§12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session) (infra NATS bullets).
- [t7 §1.5](../research/t7-cross-cutting-and-rollout.md) — NATS auth rollout (the research detail behind ADR-0035 §2).
- Code: `internal/platform/events/{topology.go,nats.go,consumer.go}` (N0 client options, reconnect settings), `internal/platform/events/testdata/nats-authorization.golden.conf`, the `Makefile` targets `nats-acl-render` / `nats-acl-test` (all from [mi-05](../sprints/sprint-mi-05.md)).
- Infra: `../infra/infrastructure/messaging/release.yaml` (NATS chart `2.14.6`), `../infra/clusters/vps/messaging.yaml`, `../infra/apps/xlearn-{practice,review,assessment,identity}.yaml`, `../infra/charts/project/values.yaml` (`extraVolumes`, `extraVolumeMounts`, `podSecurityContext.fsGroup: 65532`), `../infra/.sops.yaml`, `../infra/hack/host-verify.sh` (`--cluster --nats-stage`, the `PIN_NATS_STAGE` constant), `../infra/hack/host-lint.sh`.
- [mi-05 task 4](../sprints/sprint-mi-05.md) (identity has no `NATS_URL` in prod; it arrives with its seed in this sprint's identity N2 PR) and [mi-02 task 4](../sprints/sprint-mi-02.md) (the `PIN_NATS_STAGE` convention this sprint bumps).

## Context

NATS 2.14.6 runs one server in `messaging` with **no auth**. Today practice, review and assessment connect
(7 connections, 2026-09-24). identity has **no `NATS_URL`** in prod, so in `v1.6.0` it still drains its outbox
through the `LogPublisher`; it **joins NATS in its N2 PR**, with `NATS_URL`, seed and inbox prefix together, and
its first connection is already on its own nkey. Any pod can publish on any subject and read,
UPDATE or purge any stream. [mi-05](../sprints/sprint-mi-05.md) shipped N0 dark in `v1.6.0`: `topology.go`, the
golden `authorization` block, and the optional client env (`NATS_NKEY_SEED_FILE`, `NATS_INBOX_PREFIX`, a
permission-violation `ErrorHandler`), with a compose test proving allowed and denied ops. This sprint turns it
on **server first**:
- **N1** teaches the server every user plus a catch-all `legacy` user that anonymous clients map to — the
  **one restart**.
- **N2** gives each service its seed.
- **N3** takes every permission away from `legacy` — a reload.

The ConfigMap-change reload is done by the chart's reloader sidecar; the live pod template has no config
checksum (verified 2026-09-24). N4 (removing `legacy`) is [mi-11](../sprints/sprint-mi-11.md). judge and coach
join through their own ACL PRs ([m3-07](../sprints/sprint-m3-07.md), [l-01](../sprints/sprint-l-01.md)).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **Before N1:** mi-05 merged — the golden file and `make nats-acl-render` exist on `main`.
- [ ] **Before N1:** mi-02 merged — `host-verify.sh --cluster --nats-stage=…` and its `PIN_NATS_STAGE=open` constant exist in `../infra/hack/`.
- [ ] **Before N1:** the last Hostinger weekly image date is recorded (a before-launch item) and ≤ 7 days old (N1 restarts NATS). If it isn't recorded, don't wait: set N1 and the stages after it ⛔ in status.md, naming the owner item, and land the rest (the runbook and status docs PR).
- [ ] **Before N2:** `v1.6.0` live on all seven `xlearn-*` Deployments (`ssh vps 'k3s kubectl -n xlearn get deploy -o wide'`) and the NATS-auth integration test green at that tag.
- [ ] **Before identity's N2:** `v1.6.0` on `xlearn-identity`, and either no `messaging` NetworkPolicy yet or MI-5's policy lists `xlearn-identity` as a 4222 caller (`ssh vps 'k3s kubectl -n messaging get networkpolicy -o yaml'`).
- [ ] **Before N3:** the MI-5 PR (mi-03's first) merged; all four N2 PRs verified; zero `legacy` connections.
- [ ] Parallel sessions: no peer PR is editing `infrastructure/messaging/release.yaml` or the four `apps/xlearn-*.yaml` (`gh pr list --state open` in both repos, `git worktree list`, ListAgents).

N1 may proceed while the N2 or N3 gates are still open; stop at the first stage whose gate is red and report it.

## Do this (in order)

1. **[I] Keys** (on the owner's machine; you generate the four service seeds in-session and their SOPS
   files land in the N2 PRs; the owner generated ops offline before launch). Install `nk` and `nats`
   (`go install github.com/nats-io/nkeys/nk@latest`, `…/natscli/nats@latest`). For practice, review,
   assessment and identity: generate the seed into a mode-600
   scratch file (never printed), derive the public key with `nk -inkey … -pubout`, write
   `../infra/apps/secrets/xlearn-nats-<svc>.enc.yaml` (Secret `xlearn-nats-<svc>`, ns `xlearn`,
   `stringData.seed`), `sops --encrypt --in-place`, delete the plaintext. The **ops** key is a before-launch
   owner item: use the public key from the launch message (its seed stays offline with the age-key copies). If
   you don't have it, don't wait: N1 can't render the ops user, so set N1 ⛔ in status.md, naming the owner item,
   and land what doesn't depend on it. Don't generate judge or coach keys.
2. **[H] Rehearse locally.** In an xlearn worktree at tag `v1.6.0`, `make nats-acl-render`; build the N1 values
   excerpt (the golden users for practice, review, assessment, identity and ops as YAML under `config.merge`,
   plus `legacy` and `no_auth_user`; drop judge/coach entries if the golden has them and note it in the PR).
   mi-05 renders ops as publish `$JS.API.>`, subscribe `_INBOX.>`; if the golden has no ops user, stop and
   report (a renderer defect to fix in xlearn, never a hand-edited subject). Confirm the
   keys with `helm show values nats/nats --version 2.14.6`. `helm template` → extract `nats.conf` →
   `nats:2.14.6-alpine -t`. Run it with the `nats` CLI: an anonymous client works as `legacy`. With the N3
   variant, anonymous pub/sub is denied and logged. Test keys only; no production seed in the rehearsal.
3. **[I] N1 PR** on a `feat/nats-auth-n1` branch in `../infra`: `infrastructure/messaging/release.yaml` →
   `config.merge` (rendered users + `legacy` with a bcrypt'd never-used password + `no_auth_user: legacy`) and
   `podTemplate.merge.metadata.annotations.xlearn.dev/nats-auth-rev: "1"`; refresh the header comment to cite
   ADR-0035; **and** bump `PIN_NATS_STAGE` in `hack/host-verify.sh` from `open` to `n1`, then run
   `hack/host-lint.sh`. Put the public-key table, the rehearsal output and the host-lint output in the PR body.
   Record the live `/connz` connection count (7 on 2026-09-24: practice, review, assessment). Merge → watch
   `rollout status sts/nats` → verify: a **plain** `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh`
   (no `--nats-stage` override) green, then the mi-02 node-copy refresh: run
   `scp ../infra/hack/host-verify.sh vps:/root/` once (pre-approved by launching this prompt, D40);
   `/connz?auth=true` all `legacy` with the recorded count restored; `/jsz?consumers=true` pending
   → 0; each outbox's unsent → 0 (psql via `kubectl exec` on `projects-pgstore-1`, read-only); no
   permission ERRORs in `xlearn-*` logs. Anything off → revert + a second annotation bump.
4. **[X] Break-glass runbook, ready for the owner's first use.** Write the runbook (step 7; plan task 6) far
   enough for the owner to run `nats stream ls` through the `ssh -L … port-forward svc/nats` tunnel with the
   offline ops seed (default inbox, no `--inbox-prefix`: ops subscribes on `_INBOX.>`). Don't wait for that run:
   it's a pending owner item, recorded in status.md as the post-ship owner event `ev-nats-ops-first-use`
   (step 7), and it doesn't gate _Overall_ ✅.
5. **[I] N2 PRs, one per service: practice → review → assessment → identity.** Each PR adds its encrypted seed
   file plus `NATS_NKEY_SEED_FILE=/var/run/secrets/nats/seed`, `NATS_INBOX_PREFIX=_INBOX_<svc>`, and the
   `extraVolumes` (secret `xlearn-nats-<svc>`, `defaultMode: 0440`, item `seed`) / `extraVolumeMounts`
   (`/var/run/secrets/nats`, read-only) from the plan. **identity's PR also adds
   `NATS_URL=nats://nats.messaging.svc.cluster.local:4222`** (it has none today). Merge one, verify it: every
   connection of that service shows its public key; no violation ERROR in its log; outbox unsent 0; its
   durables' pending 0; one real flow. Then open the next. identity only once its gate is green (`v1.6.0` on
   `xlearn-identity`; MI-5's policy, if merged, admits it on 4222). Its first connection must show its own
   nkey (never `legacy`), `XLEARN_IDENTITY` must exist (`/jsz?streams=true`) and `identity.outbox` unsent → 0.
6. **[I] N3 PR** (`feat/nats-auth-n3`): only `legacy`'s permissions → `deny ">"` for publish and subscribe, plus
   `PIN_NATS_STAGE` `n1` → `n3` in `hack/host-verify.sh` (run `hack/host-lint.sh`); **no** annotation bump.
   Before merging: zero `legacy` connections and MI-5 merged. After: `/varz` `config_load_time` moved and
   `start` unchanged; a **plain** `host-verify --cluster` (now checking `n3`) green, then refresh the `/root`
   copy the same way; all services back on nkeys; outbox and pending 0; login, dashboard and coach smoke (login
   in an already-signed-in browser session if you have one; otherwise the credential-free checks, plus "owner
   login smoke pending (N3)" as a pending-smoke note). On this machine you open the tunnel and, with no seed,
   run an anonymous `nats pub probe.n3 x` / `nats sub probe.n3`: both are refused, and the `nats-0` log shows
   it. Close the tunnel, log the probe in the break-glass log, and re-run the plain `--cluster`. Record the N3
   timestamp.
7. **[X] Runbook + record** on a `docs/…` branch in xlearn:
   - finish `docs/v2/runbooks/nats-break-glass.md` (when, how, why port-forward passes MI-5, rotation and
     revocation, never-list, the log rule);
   - add a short "NATS auth (v2)" note to `docs/architecture/events.md`;
   - update `docs/v2/status.md` (below) and this sprint's Status table.

## Constraints

- **GitOps only.** Every cluster change is a `../infra` PR reconciled by Flux; never `kubectl apply`, `edit`,
  `patch` or `delete` by hand. Reads (`get`, `get --raw` proxy, `logs`, a read-only `psql` via `exec`) are
  fine. The break-glass path is the only manual NATS path, and this sprint uses it read-only. The only host
  write is mi-02's node-copy refresh (`scp ../infra/hack/host-verify.sh vps:/root/`) after the N1 and N3
  merges, which launching this prompt pre-approves (D40).
- **Server before clients:** N1 before any seed; each N2 verified before the next; N3 only at zero `legacy`.
  **Infra PRs stand alone** — never folded into a tag. No SOPS decryption in `messaging`: public keys are
  plaintext values.
- **Secrets hygiene:** seeds never printed, committed unencrypted, or copied to the VPS by hand; the ops seed
  never leaves the owner's offline storage.
- **Outbox / inbox:** the outbox is the buffer. "No event lost" means every `<svc>.outbox` unsent count and
  every durable's pending return to 0 after each stage. Never purge a stream or delete a consumer here.
- **ACL before tag (standing rule):** this sprint *is* the ACL PR for the four live services. Any later new
  stream, consumer or subject needs a re-rendered golden merged in `../infra` before its consuming tag.
- **D34:** no alerting, Flux Alert, CronJob, timer or monitoring identity — `host-verify --cluster` is the
  only check, run on demand.
- **Memory-sum rule:** no new pod or container (seed mounts only); nothing to re-check.
- **Rollback floor:** client images must stay ≥ `v1.6.0` while seeds are mounted (no R-b below it).
- **Parallel sessions:** check peers' PRs, tags and worktrees before each infra merge (a peer tag rolling
  `xlearn-*` during an N2 verify muddles the check). Re-check for owner or peer messages before N3.
- n/a here: goose/sqlc (no migration), `theme.css` (no UI), service boundaries (no code).

## Deliverables

- `../infra`: N1 PR (`messaging` values: users, `legacy`, `no_auth_user`, restart annotation; `hack/host-verify.sh`
  `PIN_NATS_STAGE=n1`); four N2 PRs (seed SOPS file + mount + env per service; identity's also adds `NATS_URL`);
  N3 PR (`legacy` deny; `PIN_NATS_STAGE=n3`). The node copy `/root/host-verify.sh` refreshed after N1 and N3 (pre-approved, D40).
- xlearn docs PR: `docs/v2/runbooks/nats-break-glass.md`, the `events.md` note, `status.md` updates, this
  sprint's statuses.
- Live: every NATS connection on its own nkey; `legacy` denied; the break-glass tunnel proved (the N3 probe).

## Update status

- [`../sprints/sprint-mi-06.md`](../sprints/sprint-mi-06.md): each task 🔄 / ✅ / ⛔; _Overall_ ✅ when all seven are ✅.
- [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **MI** table's MI-7 row (N1, N2 × 4 and N3 dates with `../infra` PR numbers);
  - the **NATS** rows: stage `n3`, and the **N3 timestamp** that [l-01](../sprints/sprint-l-01.md)'s ≥ 24 h
    re-check and [mi-11](../sprints/sprint-mi-11.md) gate on;
  - the **NATS break-glass log** (the N3 probe);
  - the **Owner calendar events**: add `ev-nats-ops-first-use` (after this sprint; the owner's first ops-seed
    `nats stream ls` through the runbook, logged in the break-glass log; prepared by mi-06);
  - the **Pending-smoke notes**, if the N3 login smoke couldn't run in a signed-in session;
  - the **Decisions log** (judge/coach keys deferred to m3-07/l-01; bcrypt'd `legacy`; N2 order; identity
    joined NATS in its N2 PR; `PIN_NATS_STAGE` now `n3`; anything the rehearsal changed).
- No new ADR expected (ADR-0035 governs). If you must deviate from ADR-0035 §2, write an ADR after checking
  peers for the next free number.

## Done when (acceptance)

- [ ] Every live NATS connection authenticates with its own nkey; no `legacy` connection.
- [ ] Denied operations are logged, not silently dropped (the N3 probe is refused and logged).
- [ ] N3 applied as a reload; a plain `host-verify --cluster` (`PIN_NATS_STAGE=n3`) green right after; timestamp recorded for l-01.
- [ ] identity on NATS with its own nkey: `XLEARN_IDENTITY` exists and its relay drains.
- [ ] No event lost: outbox unsent 0 and pending 0 after every stage; a real flow works end to end.
- [ ] Break-glass runbook written; its tunnel proved by the N3 probe and logged; `ev-nats-ops-first-use` recorded for the owner's first ops-seed use (post-ship; it doesn't gate _Overall_ ✅).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: `feat/nats-auth-n1`, one branch per N2 service and `feat/nats-auth-n3` in `../infra`, then a `docs/…` branch in xlearn.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. infra has no CI: paste each PR's verify output into its body and merge on it.
3. **Release action — infra PR(s) only:** Merge the infra PRs in the plan's order (each its own PR, never folded into a tag): N1 → N2 practice → review → assessment → identity → N3, each only after the previous one's verify step passes and Flux has reconciled; then the xlearn docs/status PR. No tag. Run a plain `host-verify --cluster` after N1 and after N3, each followed by the `/root` copy refresh.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
