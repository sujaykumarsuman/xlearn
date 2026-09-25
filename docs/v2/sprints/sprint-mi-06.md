# Sprint mi-06 — NATS auth server-first: N1 → N2 → N3 (MI-7)

> **Milestone:** MI — rollout step **MI-7**, a **hard precondition for M3 and for erase (L-E)** · **Track:** infra · **Order:** 19
> **Prereqs:** [mi-05](sprint-mi-05.md) (N0: `topology.go`, golden ACL, client options) · [m1-02](sprint-m1-02.md) (`v1.6.0` carries N0) · [mi-03](sprint-mi-03.md) (its first PR, MI-5, before N3) · [mi-02](sprint-mi-02.md) (`host-verify --nats-stage`)
> **Unblocks:** [l-01](sprint-l-01.md) (its task 1 is the ≥ 24 h N3 re-check) · [m3-07](sprint-m3-07.md) (judge's ACL rides on this model) — and through l-01, [mi-11](sprint-mi-11.md) (N4)
> **Release action:** **infra PR(s) only** — six `../infra` PRs (N1, four N2, N3), never folded into a tag · plus one xlearn **docs** PR (runbook + status)
> **Calendar:** week 3 (2026-10-12 → 10-16), after `v1.6.0` is live; runs beside the spike week. N1 restarts NATS: check the Hostinger weekly image first.
> **Execute with:** [`../prompts/prompt-mi-06.md`](../prompts/prompt-mi-06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Generate the nkeys (four service seeds → SOPS, agent in-session; the ops public key comes from the owner, generated offline before launch) | I | ⬜ |
| 2 | Local rehearsal of the N1 / N3 server config (NATS 2.14.6) | H | ⬜ |
| 3 | **N1** PR: nkey users + fine ACLs + `legacy` + `no_auth_user` + `PIN_NATS_STAGE=n1` — the one restart | I | ⬜ |
| 4 | **N2** PRs: seed + env per service — practice → review → assessment → identity (identity also gets `NATS_URL`) | I | ⬜ |
| 5 | **N3** PR: `legacy` → `deny ">"` (reload) + `PIN_NATS_STAGE=n3` + verify | I | ⬜ |
| 6 | NATS break-glass runbook; tunnel proved by the N3 probe (the owner's first ops-seed use is recorded as a pending owner event) | X | ⬜ |
| 7 | Record (status NATS rows, MI-7, events.md note) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the MI table's MI-7 row + the NATS rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

Each gate guards one stage; a later stage may wait while an earlier one proceeds.

- [ ] **Before N1:** MI-6 merged ([mi-05](sprint-mi-05.md)) — `internal/platform/events/testdata/nats-authorization.golden.conf` and `make nats-acl-render` on `main`
- [ ] **Before N1:** MI-8 host-verify NATS stage check available ([mi-02](sprint-mi-02.md)) — `hack/host-verify.sh --cluster --nats-stage=open|n1|n3|n4`, with the script constant `PIN_NATS_STAGE=open` that this sprint bumps
- [ ] **Before N1:** Hostinger weekly image date checked (≤ 7 days; the owner reads hPanel before launch and records it in status.md, and the session verifies it) — N1 is a restart-inducing step ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)). If it isn't recorded, don't wait: set N1 and the stages after it ⛔ in status.md, naming the owner item, and land the rest (the runbook and status docs PR)
- [ ] **Before N2:** `v1.6.0` — the tag carrying N0 ([m1-02](sprint-m1-02.md)) — live on all seven `xlearn-*` Deployments, with the NATS-auth integration test (`make nats-acl-test`, its CI job) green at that tag
- [ ] **Before identity's N2:** `v1.6.0` live on `xlearn-identity` (the NATS publisher code), and either no `messaging` ingress NetworkPolicy exists yet or MI-5's policy lists `xlearn-identity` as a 4222 caller (forward-declared, [mi-03](sprint-mi-03.md)) — `ssh sujaykumar-vps 'k3s kubectl -n messaging get networkpolicy -o yaml'`
- [ ] **Before N3:** MI-5 PR merged ([mi-03](sprint-mi-03.md)'s first PR: `databases` + `messaging` ingress). MI-5a and MI-4 ([mi-14](sprint-mi-14.md)) do **not** gate this sprint
- [ ] **Before N3:** all four N2 PRs verified — `/connz?auth=true` shows **no** `legacy` connection

## Goal

Turn NATS from an open bus into **per-service nkey users with fine ACLs** rendered from `topology.go`, with
**one restart and zero event loss**, and close the `legacy` fallback (**N3**) — the hard precondition for M3
and erase ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).
Server first: the server learns the users while every client still connects anonymously (mapped to `legacy`);
then each service presents its own seed; then `legacy` loses every permission. Clients-first can't work:
nats.go refuses nkey auth against a server that sends no nonce (ADR-0035 context).

## Scope

**In**
- **N1:** nkey users with the golden fine ACLs for **practice, review, assessment, identity** plus the offline
  **ops** identity; a `legacy` user (allow `>`, never-used password) and top-level `no_auth_user: legacy`;
  one restart via a pod-template annotation bump.
- **N2:** seeds for practice, review, assessment and identity — SOPS secret, `extraVolumes` mount at
  `/var/run/secrets/nats/seed`, `NATS_NKEY_SEED_FILE` + `NATS_INBOX_PREFIX` — one PR per service. identity has
  no `NATS_URL` in prod today (it stays on its `LogPublisher` in `v1.6.0`, [mi-05](sprint-mi-05.md) task 4), so
  **identity joins NATS in its N2 PR**: `NATS_URL`, seed and prefix arrive together.
- **N3:** `legacy` → `deny ">"` (a reload, not a restart), verified right after.
- `../infra/hack/host-verify.sh`'s `PIN_NATS_STAGE` constant: `open` → `n1` in the N1 PR, → `n3` in the N3 PR
  ([mi-02](sprint-mi-02.md)'s convention), so a plain `host-verify --cluster` always checks the live stage.
- The NATS break-glass runbook, proved once read-only.

**Out**
- **N4** (remove `legacy` and `no_auth_user` together) → [mi-11](sprint-mi-11.md).
- **judge** and **coach** keys, users and seeds → with their tags: [m3-07](sprint-m3-07.md) (judge ACL PR before
  `v1.13.0`) and [l-01](sprint-l-01.md) (coach ACL PR before `v1.11.0`). MI-5 already forward-declares both.
- The ≥ 24 h N3 re-check → [l-01](sprint-l-01.md) task 1 (the 2026-10-24 host-window runbook,
  [mi-09](sprint-mi-09.md), also runs `--nats-stage=n3`).
- **Any SOPS decryption in `messaging`** — never: public keys are plaintext values, seeds live in `apps/secrets`.
- Stream limits and the ACL renderer itself → [mi-05](sprint-mi-05.md); a monitoring identity → none exists
  (`host-verify` reads `:8222` through the API-server proxy).

## Tasks

### 1 · Generate the nkeys [I; the ops key: O, before launch]

On the owner's machine (never the VPS, never CI). **O, before launch** = the ops key, the owner alone; **I** = the four service
seed files in `../infra/apps/secrets/`, which the agent generates in-session on that machine and which land in the
N2 PRs. Tools: `go install github.com/nats-io/nkeys/nk@latest` and
`go install github.com/nats-io/natscli/nats@latest` (the CLI is for task 6).
- **Service users — practice, review, assessment, identity.** For each: `nk -gen user > <scratch>/<svc>.nk`
  (mode 600; the seed is never printed), `nk -inkey <scratch>/<svc>.nk -pubout` → the public key `U…` (goes in
  the N1 PR's key table), then write `../infra/apps/secrets/xlearn-nats-<svc>.enc.yaml` and encrypt it in place
  with `sops` (the existing `apps/secrets/.*\.enc\.ya?ml$` rule):

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: xlearn-nats-<svc>
    namespace: xlearn
  type: Opaque
  stringData:
    seed: SUA…            # the nkey seed — encrypted by sops before any commit
  ```

  Then delete the plaintext `<svc>.nk`. The agent may run these four in-session (seed piped straight into the
  file, never echoed); each encrypted file lands in its own N2 PR (task 4).
- **ops (owner, before launch).** The **owner alone** runs `nk -gen user -pubout` offline before launching the
  prompt and gives the agent only the public key. The seed goes offline with the two age-key copies (MI-1) — never
  in git, the cluster or a transcript
  ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first), "Other identities").
  The session checks it has the public key before N1. If it's missing, N1 can't render the ops user: don't wait;
  set N1 ⛔ in status.md, naming the owner item, and land what doesn't depend on it.
- **judge, coach: not now.** m3-07 and l-01 generate theirs with their ACL PRs, so no seed waits unencrypted
  for weeks.

### 2 · Local rehearsal of the N1 / N3 config [H]

Before touching production, prove the exact config the chart will render:
- In an xlearn worktree at the **`v1.6.0`** tag (the code the cluster runs), `make nats-acl-render`. The
  `messaging` values need the block as YAML under `config.merge` (the nats chart renders `nats.conf` as JSON from
  values). Use the renderer's YAML output if it has one; else transcribe **without changing any subject**. Keep
  exactly the users for practice, review, assessment, identity and ops. If the golden also declares judge or
  coach, leave them out and say so in the PR: m3-07 and l-01 re-render and add them. mi-05 always renders
  **ops** as publish `$JS.API.>` (purge included) and subscribe `_INBOX.>` (the `nats` CLI's default inbox;
  [mi-05](sprint-mi-05.md) task 2). If the golden has no ops user, **stop and report**: that is a renderer
  defect to fix in xlearn first, never a hand-edited subject.
- `helm template nats nats/nats --version 2.14.6 -f <values excerpt>` → extract `nats.conf` from the
  `nats-config` ConfigMap → `docker run --rm -v …:/etc/nats nats:2.14.6-alpine -c /etc/nats/nats.conf -t`
  (config test) → then run it for real (JetStream on) and check with the `nats` CLI: an **anonymous** client is
  mapped to `legacy` and can publish and subscribe. Swap in the N3 variant → the anonymous `pub` and `sub` are
  **denied**, and the server logs the violation. Per-service seed connections and ACL semantics are already
  proven by mi-05's integration test (test keys); don't copy production seeds into the rehearsal.
- Confirm the values keys against `helm show values nats/nats --version 2.14.6` (`config.merge`,
  `podTemplate.merge`), and that the live StatefulSet's pod template carries **no config-checksum annotation**
  (verified 2026-09-24: none). That is what makes N3 a reload, not a restart.

### 3 · N1: users + `legacy` + `no_auth_user` — the one restart [I]

`../infra/infrastructure/messaging/release.yaml` values (and rewrite the stale header comment "No auth in v1 …
tighten in the S12 hardening pass" to point at ADR-0035):

```yaml
    config:
      merge:
        no_auth_user: legacy
        authorization:
          users:
            # --- rendered by `make nats-acl-render` at xlearn v1.6.0 — do not hand-edit subjects ---
            - nkey: U…PRACTICE
              permissions:
                publish:   { allow: ["xlearn.practice.>", "$JS.API.INFO", "$JS.API.STREAM.CREATE.XLEARN_PRACTICE", …] }
                subscribe: { allow: ["_INBOX_practice.>"] }
            # … review, assessment, identity, ops (ops: publish $JS.API.> incl. purge; subscribe _INBOX.>) …
            # --- end rendered ---
            - user: legacy
              password: "$2a$11$…"   # bcrypt (`nats server passwd`) of a random, never-used password
              permissions:
                publish:   { allow: [">"] }
                subscribe: { allow: [">"] }
    podTemplate:
      merge:
        metadata:
          annotations:
            xlearn.dev/nats-auth-rev: "1"   # the one restart (N1); later config changes are reloads
```

- Public keys are **plaintext** (no SOPS in `messaging`). Paste the public-key table (service → `U…`) and the
  rehearsal output into the PR.
- **The same PR** bumps `PIN_NATS_STAGE` in `../infra/hack/host-verify.sh` from `open` to `n1`
  ([mi-02](sprint-mi-02.md)'s convention; a revert rolls both back together). Run `hack/host-lint.sh` and paste
  its output into the PR.
- Right before merging, record the live connection count from `/connz` (7 on 2026-09-24: practice, review and
  assessment only; identity has no `NATS_URL` until its N2 PR).
- Merge → the `messaging` Kustomization reconciles (1 m) → `nats-0` restarts (~30–60 s plus the 30 s lame duck).
  The outboxes buffer; clients reconnect on their own (`MaxReconnects(-1)`, 2 s wait —
  `internal/platform/events/nats.go`, `consumer.go`). Watch
  `ssh sujaykumar-vps 'k3s kubectl -n messaging rollout status sts/nats'`.
- **Verify:** a **plain** `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` (no `--nats-stage`
  override, so it proves the bumped constant) green; then refresh the node copy `/root/host-verify.sh` the mi-02
  way: the agent runs `scp ../infra/hack/host-verify.sh sujaykumar-vps:/root/` once (a node write this plan names, so
  launching the prompt pre-approves it, D40);
  `/connz?auth=true` (`ssh sujaykumar-vps 'k3s kubectl get --raw "/api/v1/namespaces/messaging/pods/nats-0:8222/proxy/connz?auth=true"'`)
  shows every connection authorised as **`legacy`**, and the count is back to the pre-restart count recorded
  above; `/jsz?consumers=true` shows every durable with pending
  draining to 0; every service's outbox unsent → 0
  (`k3s kubectl -n databases exec projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select count(*) from <svc>.outbox where sent_at is null"`);
  no permission-violation ERROR in any `xlearn-*` log.
- **Revert:** `git revert` plus a second annotation bump (restart) — [ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step), "NATS N1–N3".

### 4 · N2: seed + env per service [I]

**One PR per service, in the order practice → review → assessment → identity, each verified before the next.**
identity goes last, once its entry gate is green (`v1.6.0` on `xlearn-identity`; MI-5's policy, if merged,
admits `xlearn-identity` on 4222). In `../infra/apps/xlearn-<svc>.yaml`, add the seed secret file from task 1
and:

```yaml
    env:
      # …existing…
      - name: NATS_URL                     # identity's PR ONLY — practice/review/assessment already have it
        value: nats://nats.messaging.svc.cluster.local:4222
      - name: NATS_NKEY_SEED_FILE          # N0 client option (mi-05); no-op if unset
        value: /var/run/secrets/nats/seed
      - name: NATS_INBOX_PREFIX            # must match the ACL's _INBOX_<svc>.>
        value: _INBOX_<svc>
    extraVolumes:
      - name: nats-seed
        secret:
          secretName: xlearn-nats-<svc>
          defaultMode: 0440                # root:65532 (chart fsGroup): group-read for the non-root user
          items: [{ key: seed, path: seed }]
    extraVolumeMounts:
      - name: nats-seed
        mountPath: /var/run/secrets/nats
        readOnly: true
```

- The Deployment rolls once. A missing secret stops the new pod at `CreateContainerConfigError` while the old pod
  keeps serving, so the rollout stalls safely. With `NATS_URL` set, a bad or unreadable seed makes the new pod
  **exit** (mi-05's fail-closed init) and crash-loop while the old pod keeps serving; for identity that old pod
  is still on its `LogPublisher`, so nothing is lost either way. Reverting identity's PR takes it back to the
  `LogPublisher`.
- **Verify per service:** every connection of that service (publisher and consumer) shows **its public key**,
  not `legacy`; no permission-violation ERROR in `k3s kubectl -n xlearn logs deploy/xlearn-<svc> --since=15m`
  (the N0 `ErrorHandler`); outbox unsent = 0; its durables' pending = 0. Then drive one real flow: for practice,
  log an attempt → review and assessment consume it. **identity** has no earlier connection: its first
  connection must appear **directly with its own nkey** (never as `legacy`; a `legacy` identity connection means
  the seed isn't in use: revert). Its `XLEARN_IDENTITY` stream now exists (`/jsz?streams=true`) and its relay
  drains (`identity.outbox` unsent → 0).
- **Revert:** that release only.

### 5 · N3: `legacy` → `deny ">"` (reload) [I]

Only after task 4 is verified for all four services **and** MI-5 is merged. First confirm **zero** `legacy`
connections; if one remains, find the straggler before going on. Change only `legacy`'s permissions:

```yaml
            - user: legacy
              password: "$2a$11$…"
              permissions:
                publish:   { deny: [">"] }
                subscribe: { deny: [">"] }
```

- **The same PR** bumps `PIN_NATS_STAGE` in `../infra/hack/host-verify.sh` from `n1` to `n3` (a revert rolls
  both back together); run `hack/host-lint.sh` and paste its output. [mi-11](sprint-mi-11.md) bumps it to `n4`.
- **No** annotation bump: the reloader sidecar SIGHUPs `nats-server` once the ConfigMap lands in the pod
  (≈ 1–2 min). Confirm `/varz` `config_load_time` moved and `start` did not.
- **Verify right after:** a **plain** `ssh sujaykumar-vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh` (no
  override; it now checks `n3`) green — no `legacy` or anonymous connection; the node copy refreshed the mi-02
  way (the pre-approved `scp`); all services reconnected with their nkeys; outbox unsent 0, pending 0; login,
  dashboard and coach smoke OK (login in an already-signed-in browser session if the session has one; otherwise
  the credential-free checks, plus "owner login smoke pending (N3)" as a pending-smoke note in status.md).
- **Denied → logged (on prod, harmless):** the agent opens the break-glass tunnel (task 6) and, **without** a
  seed, runs `nats pub probe.n3 x` and `nats sub probe.n3` (a subject no stream captures) → both refused. The
  `nats-0` log shows the violation for user `legacy`. Close the tunnel, log the probe in the NATS break-glass log,
  and re-run the plain `host-verify --cluster`.
- The **≥ 24 h re-check** is [l-01](sprint-l-01.md)'s first task; record the N3 timestamp so l-01 can gate on it.
- **Revert:** `git revert` (a reload).

### 6 · Break-glass runbook [X]

Write **`docs/v2/runbooks/nats-break-glass.md`** — the only sanctioned manual NATS path
([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag), "No hand-applied changes"):
- **When:** the re-seal runbook ([ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md): purge,
  then republish from the outbox), stream surgery (a poison message, a consumer reset), a NATS emergency.
  Never routine.
- **How:** on the owner's machine,
  `ssh -L 14222:127.0.0.1:14222 vps 'k3s kubectl -n messaging port-forward svc/nats 14222:4222'`, then
  `nats --server nats://127.0.0.1:14222 --nkey <offline ops seed> …` (no `--inbox-prefix`: ops subscribes on the
  default `_INBOX.>`, as mi-05 renders it).
  Read first (`stream ls`, `stream info`, `consumer info`); mutate only against a written plan; close the
  tunnel; leave no seed copy in the working directory.
- **Why it passes MI-5:** port-forward enters the pod's network namespace through the kubelet, so the
  `messaging` NetworkPolicy doesn't apply (proved by task 5's N3 probe through the tunnel, after MI-5).
- **Rotation / revocation** (ADR-0035 §2): add the new public key → roll the seed (SOPS + a `podAnnotations`
  bump, e.g. `xlearn.dev/nats-seed-rev`) → remove the old key. Every step is a reload; removing a key alone
  revokes it. Same for ops.
- **Never:** a Job or pod carrying the ops seed, `kubectl apply`, SOPS in `messaging`.
- **Log every use** in `status.md` → NATS break-glass log (date, who, why, commands, outcome).
- **First use (a pending owner item, not a wait):** the owner runs `nats stream ls` through the runbook
  (read-only) with the offline ops seed and logs it in the break-glass log. That proves the ops key and its ACL
  before anyone needs them; task 5's N3 probe already proves the tunnel. Task 7 records it in status.md as the
  post-ship owner event `ev-nats-ops-first-use`; it doesn't gate _Overall_ ✅.

### 7 · Record [X]

- `docs/v2/status.md`: the MI table's **MI-7** row (N1 / N2 × 4 / N3 dates and `../infra` PR numbers); the
  **NATS rows** (stage = n3, the N3 timestamp for l-01's ≥ 24 h re-check); the break-glass log (the N3 probe);
  the owner event `ev-nats-ops-first-use` (post-ship, prepared by mi-06: the owner's first ops-seed
  `nats stream ls`); a pending-smoke note if the N3 login smoke couldn't run; the Sprint board row; Decisions-log lines (judge/coach keys deferred to their ACL PRs, m3-07 / l-01; `legacy`
  password bcrypt; the N2 order; identity joined NATS in its N2 PR; `PIN_NATS_STAGE` now `n3`).
- `docs/architecture/events.md`: a short "NATS auth (v2)" note — nkey users per service, fine ACLs rendered from
  `topology.go`, a new stream/consumer/subject needs its infra ACL PR merged before the consuming tag; link
  ADR-0035 §2.

## Acceptance criteria

- [ ] Every live NATS connection authenticates with **its own nkey** (practice, review, assessment, identity);
      `/connz?auth=true` shows no `legacy` connection.
- [ ] Denied operations are **logged, not silently dropped**: the N3 anonymous probe is refused and the
      violation appears in the `nats-0` log; services show no violation ERRORs in normal flow.
- [ ] **N3 applied** (a reload, `start` unchanged); a plain `host-verify --cluster` (`PIN_NATS_STAGE=n3`) green
      right after the reload, with the timestamp recorded for l-01's ≥ 24 h re-check.
- [ ] identity is on NATS with its own nkey: `XLEARN_IDENTITY` exists and its relay drains (l-01's entry gate).
- [ ] **No event lost:** every outbox's unsent count is 0 and every durable's pending is 0 after N1, after each
      N2 and after N3; a real flow (attempt → review + assessment) works end to end.
- [ ] The break-glass runbook exists, and its tunnel is proved by the N3 probe (logged in `status.md`). The
      owner's first ops-seed use is recorded as the post-ship owner event `ev-nats-ops-first-use`; it doesn't gate
      _Overall_ ✅.

## Release

**Infra PR(s) only** — six `../infra` PRs, each its own task and **never folded into a tag**
([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)): N1, four N2 (one per
service), N3. `../infra` has no CI; merge each under the standing merge authority once its verify step passes,
then the next. The xlearn **docs PR** (runbook, `status.md`, `events.md`, this file) merges at session end. No
xlearn tag: N0's client code already shipped in `v1.6.0`. The tag-timeline row "— (infra) NATS N1 → N2 → N3"
([rollout §7](../rollout-plan.md#7-indicative-tag-timeline)) sets the client-image floor at `v1.6.0`: never
narrow an `xlearn-*` ImagePolicy below it (R-b) while seeds are mounted.

## Definition of Done

Six infra PRs merged through GitOps (no hand `kubectl apply`) · a plain `host-verify --cluster` green at
`PIN_NATS_STAGE=n3` (node copy refreshed) ·
zero event loss shown (outbox and pending at 0) · runbook written and its tunnel proved (the N3 probe) · acceptance met · statuses updated
(this file + [`../status.md`](../status.md)) · notable calls in the Decisions log.

## Risks / watch-outs

- **The one restart blips event flow for about a minute.** The outboxes buffer and the relays retry, so nothing
  is lost. Do it at a quiet time, with the weekly image checked.
- **A client on an image older than the N0 tag can't use a seed** — the floor is `v1.6.0`. An R-b below it
  after N2 would cut that service off after N3.
- **A straggler still on `legacy` at N3 loses NATS instantly.** Hence the explicit zero-`legacy` check before
  the N3 merge. Revert N3 if anything drops.
- **A stale `PIN_NATS_STAGE` makes every plain `--cluster` run vacuous** (`open` always PASSes). The release
  checklist (contract, erase, GA) and the M3 checklist rely on the plain run, so the bump rides in the N1 and N3
  PRs themselves, and the node copy in `/root` is refreshed after each (per mi-02; pre-approved by launching the
  prompt, D40).
- **Golden vs live drift.** Render at the tag the cluster runs (`v1.6.0`), not at `main`. A newer `main` may
  declare durables the live code doesn't use yet (harmless) or rename one (not harmless).
- **Chart keys.** `config.merge` / `podTemplate.merge` are confirmed against `helm show values` in task 2. If a
  config checksum ever appears on the pod template, N3 becomes a restart: still safe, but note it.
- **Seed file permissions.** A seed the non-root user can't read fails the connect loudly (the relay logs and
  retries). `defaultMode: 0440` with the chart's `fsGroup: 65532` is the fix.
- **The `legacy` password is in git** (bcrypt only). It grants nothing an anonymous client doesn't already get
  before N3, and nothing after.
