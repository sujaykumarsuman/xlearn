# Sprint mi-02 — host-verify --cluster extension (MI-8)

> **Milestone:** MI — infra-first track (rollout step **MI-8**, plus the MI-0 follow-up; sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** infra (host scripts) · **Order:** 3
> **Prereqs:** none in code · calendar event MI-0 (H0 reboot, Fri 2026-09-25) · can run beside [mi-01](sprint-mi-01.md)
> **Unblocks:** [mi-14](sprint-mi-14.md) · [mi-03](sprint-mi-03.md) (NetworkPolicy presence) · [mi-06](sprint-mi-06.md) (NATS stage check) · [mi-08](sprint-mi-08.md) (memory sum + samples) · [m1-08](sprint-m1-08.md) (the first contract tag's checklist) · [mi-09](sprint-mi-09.md) (its entry gate: the window runbook calls `--cluster`, `--with-runner` and `--nats-stage=…`). It also feeds every contract, erase and GA tag checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6) and the M3 hard checklist.
> **Release action:** **infra PR(s) only** (a host script, run on demand over `ssh vps`). Plus an xlearn docs PR for status. No tag.
> **Calendar:** weeks 1–2 (after 2026-09-25)
> **Execute with:** [`../prompts/prompt-mi-02.md`](../prompts/prompt-mi-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Memory-sum check (+ embedded `memory-budget.tsv`) | H | ⬜ |
| 2 | Flux objects, pod health, CNPG | H | ⬜ |
| 3 | Volumes and disk | H | ⬜ |
| 4 | NATS auth stage | H | ⬜ |
| 5 | NetworkPolicy presence (+ embedded `expected-netpol.tsv`) | H | ⬜ |
| 6 | Steal and CPU (TR-*) | H | ⬜ |
| 7 | Read-only proof, lint, `--json`, README; PR and merge | H | ⬜ |
| 8 | Throwaway top sampler for MI-11a (owner OK) | H | ⬜ |
| 9 | MI-0 follow-up (H0 result, S0 vmstat) | O | ⬜ |
| 10 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track rows MI-0, MI-8).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-0 done: the H0 reboot into 6.8.0-142 (Fri 2026-09-25) and `host-verify --cluster` green after it. *Tasks 1–8 don't need the reboot. If it slipped, ask the owner whether to build them now and leave task 9 ⛔.*
- [ ] No open peer PR in `../infra` touches `hack/` (parallel sessions: `gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents)

## Goal

Give the owner the only ops tool v2 has (D34): cheap, **read-only**, on-demand cluster reads in
`hack/host-verify.sh --cluster`, implementing the [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3 check table:
- the memory-sum rule;
- Flux readiness;
- pod health;
- volumes and disk;
- the NATS auth stage;
- NetworkPolicy presence;
- sar steal and CPU (the TR-* triggers).

It's run after every host change, and in the release checklist before contract, erase and GA tags. It's
**never an alert**. The sprint also starts the throwaway `top` sampler that [mi-08](sprint-mi-08.md) needs
for Longhorn's p95 budget, and records the MI-0 outcome.

## Scope

**In**
- The seven ADR-0035 §3 checks, as new sections of `check_cluster`, with new flags:
  - `--with-runner`;
  - `--nats-stage=open|n1|n3|n4`;
  - `--json`;
  - `--budget-file` / `--netpol-file` (test overrides).
- `hack/memory-budget.tsv` and `hack/expected-netpol.tsv` as the reviewed sources, each **embedded byte-identically** in `host-verify.sh`, with `host-lint.sh` checking the copies.
- `hack/host-lint.sh`: the read-only verb assertion; shellcheck for the new scripts.
- `hack/sample-top.sh`: a throwaway, self-terminating sampler writing `/var/tmp/xlearn-top.tsv`.
- The MI-0 follow-up record: the H0 result and the S0 vmstat collection.
- The infra README (host scripts section): the new flags and where each is used.

**Out**
- Any alert, timer, CronJob, push channel, Flux `Provider`/`Alert`, healthchecks.io or opscheck (D34, [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3). The sampler is a one-off loop, not a monitor.
- Writes of any kind by `host-verify.sh`.
- ADR-0035 §3's "cheap later additions": stream usage from `/jsz`, consumer pending, the live `SIGNUP_MODE` and seat count, deployed tag vs ImagePolicy latest. Add them later if wanted, in their own PR.
- Reading the samples and setting real budgets (p95 × 1.5) → [mi-08](sprint-mi-08.md). Growing the expected-policy list → [mi-14](sprint-mi-14.md), [mi-03](sprint-mi-03.md), [m3-07](sprint-m3-07.md). Bumping the NATS stage → [mi-06](sprint-mi-06.md) (N1, N3) and [mi-11](sprint-mi-11.md) (N4).
- The host sandbox block and L23 assertions (`host-verify` without `--cluster`) → [mi-09](sprint-mi-09.md).

## Tasks

The script already has `pass/warn/fail/info`, `kc_get` (a `kubectl get` wrapper with a 30 s timeout) and a
`check_cluster` section. Add each check as its own function called from `check_cluster`, with stable ids
(`cluster.memsum`, `cluster.mem-budget`, `cluster.oom`, `cluster.restarts`, `cluster.pvc`, `host.disk`,
`cluster.nats-auth`, `cluster.nats-legacy`, `cluster.netpol`, `host.steal`, `host.cpu`). Add two
wrappers: `kc_top` (`kubectl top`) and `kc_raw` (`kubectl get --raw`, for the API-server proxy). Parse
JSON with `jq`.

**Rebuild-proof the dependencies** in `host-bootstrap.sh` (no apply now):
- Add `jq` and `sysstat` to `PKGS` (today `linux-virtual open-iscsi nfs-common unattended-upgrades ufw curl`, line 65).
- Ensure sysstat collection is on, the same way the script handles `iscsid`: `sysstat.service` and `sysstat-collect.timer` enabled and active. On noble the systemd timer runs `sa1`, so `/etc/default/sysstat`'s `ENABLED` doesn't matter. Both units were `enabled` on 2026-09-25, and the node has 17 sa files.

Both tools are present on the node today. If `jq` is missing, WARN and skip the JSON checks. If `sar` or the sa files are missing, see task 6.

**The tables must travel with the script.** The owner runs it as `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`
or from `/root/host-verify.sh` (README), and neither has the TSV files beside it. So:
- each TSV is embedded between `# >>> memory-budget.tsv` / `# <<< memory-budget.tsv` markers (same for `expected-netpol.tsv`);
- `host-lint.sh` fails when an embedded copy differs from the file, following the existing shared-constants pattern.

### 1 · Memory-sum check [H]

Compute [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5's rule:

> **Σ limits + Σ p95 working set of limitless containers + the largest rollout surge + host ≤ node capacity − 0.5 GiB**

- **Σ limits.** The memory limits of every container in every Running pod, across all namespaces. This
  includes native sidecars (init containers with `restartPolicy: Always`). Parse Ki, Mi, Gi, K, M, G and
  plain bytes.
- **Limitless containers.** Each one's budget comes from the embedded `memory-budget.tsv`:
  - Format: `namespace <TAB> container-glob <TAB> budget_mib <TAB> source`, budgeted per container instance, so two `longhorn-ui` replicas count twice.
  - A limitless container with **no entry** gets a **WARN** (`cluster.mem-budget`) and counts as its current `top` × 1.2.
  - A budget entry for a container that now has a limit gets INFO ("unused budget").
- **Seed the TSV from today's 23 limitless containers**, as current `top` × 1.2, with the source marked `provisional top×1.2 <date>`. The check prints how many budgets are provisional. [mi-08](sprint-mi-08.md) replaces them with measured p95 × 1.5.

  | Namespace | Containers (globs) |
  |---|---|
  | `cert-manager` | `cert-manager-controller`, `cert-manager-cainjector`, `cert-manager-webhook` |
  | `kube-system` | `traefik`, `metrics-server`, `local-path-provisioner`, `lb-tcp-*` (svclb) |
  | `longhorn-system` | `instance-manager`, `longhorn-manager`, `pre-pull-share-manager-image`, `longhorn-csi-plugin`, `node-driver-registrar`, `longhorn-liveness-probe`, `csi-attacher`, `csi-provisioner`, `csi-resizer`, `csi-snapshotter`, `engine-image-ei-*`, `longhorn-driver-deployer`, `longhorn-ui` |
  | `messaging` | `reloader` |

- **Largest rollout surge.** Use [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5's definition: the xlearn fleet roll from one image-automation commit.
  - Take the Σ over `xlearn` Deployments with `RollingUpdate` of the pod memory limit × effective `maxSurge`. 25% rounds up to 1 at one replica.
  - Today that's 7 × 128 Mi = 896 Mi. judge's 256 Mi joins automatically once it exists (m3-07).
  - `Recreate` workloads (the runner) add 0.
  - **Print as INFO, not in the sum:** the largest pod limit among the other `RollingUpdate` Deployments. Today that's 1,024 Mi (the five RollingUpdate Flux controllers; source-controller is `Recreate`), or 512 Mi after MI-11a. A Flux or vendor upgrade is an owner-attended host change, and `host-verify` is re-run after it.
- **Host.** `HOST_RESERVE_MIB`, default **2355** (2.3 GiB, the upper end of ADR-0035 §5's 1.8–2.3 GiB). Print the k3s-server RSS and `top node` working set as INFO for context. `MemAvailable` can't separate host from page cache, so don't derive the host term from it.
- **Capacity** is the node's `.status.capacity.memory`, minus 512 MiB.
- **`--with-runner`.** If no pod exists in `xlearn-runner`, add the runner's 3,072 Mi of limits (MI-12 not yet deployed), and print both totals.
- **Output.** INFO lines for each term, then PASS with the margin, or **FAIL** tagged `TR-MEM`.
- **Expected, from live reads on 2026-09-25:**

  | Term | Value |
  |---|---|
  | Σ limits | 9,354 Mi |
  | Limitless budgets | 23 containers, 1,201 Mi working set × 1.2 ≈ 1,441 Mi |
  | Surge | 896 Mi |
  | Host | 2,355 Mi |
  | Capacity | 16,376,040 Ki ≈ 15,992 Mi, so capacity − 512 = 15,480 Mi |

  - The sum is ≈ 14,046 Mi ≈ **13.7 GiB, PASS by ≈ +1.4 GiB**.
  - `--with-runner` (+3,072 Mi): ≈ 17,118 Mi ≈ **16.7 GiB, FAIL by ≈ −1.6 GiB**. That's why MI-11a gates MI-12.
  - **Context only:** ADR-0035 §5's "before MI-11a, fleet rollout: −1.1 to −1.5 GiB" row uses a different basis (raw capacity; judge 256 Mi and coach +128 Mi counted). Its numbers are close to these, but they aren't the acceptance target.

### 2 · Flux objects, pod health, CNPG [H]

- **Flux.** The existing Kustomization and HelmRelease loop extends to every Flux object that has a Ready condition:
  - GitRepository and HelmRepository;
  - ImageRepository, ImagePolicy and ImageUpdateAutomation.

  Any object not Ready → FAIL.
- **OOMKills** (TR-MEM), in any namespace except `xlearn-runner`: a container whose current or last termination reason is `OOMKilled`.
  - If it finished within the last 24 h → **FAIL**. Older → WARN.
  - In `xlearn-runner` it's a WARN.
- **Restarts** in `xlearn`, `messaging`, `databases` and `xlearn-runner`: `restartCount > 3` with the last termination inside 24 h → **FAIL**. `> 3` but older → INFO.

  *Approximation:* the API keeps only the count and the last termination, so the check can't count restarts within 24 h exactly. Say so in the line.
- **CNPG.** The existing phase and ready-instances check stays as it is.

### 3 · Volumes and disk [H]

- **PVC usage** comes from the kubelet stats summary through the API-server node proxy (read-only; verified 2026-09-24): `k3s kubectl get --raw /api/v1/nodes/<node>/proxy/stats/summary`, reading `.pods[].volume[] | select(.pvcRef)` for `usedBytes` and `capacityBytes`.
  - The PG PVC (`databases/projects-pgstore-1`) or the NATS PVC (`messaging/nats-js-nats-0`) at **≥ 60% → WARN (TR-DISK)**.
  - Other PVCs → INFO.
  - `df` inside a pod would need `exec`. It's not a read verb, so it's forbidden here.
- **Node disk:** `df -P` on `/`, `/var/lib/rancher` and `/var/lib/longhorn`. **≥ 70% → WARN (TR-DISK)**.

### 4 · NATS auth stage [H]

- **Reads.** `/varz` and `/connz?auth=true&limit=1024`, through the API-server **pod** proxy:
  `k3s kubectl get --raw /api/v1/namespaces/messaging/pods/nats-0:8222/proxy/varz`.
  - This is verified 2026-09-24.
  - The `nats` ClusterIP Service exposes only 4222. Port 8222 is `monitor` on `nats-headless`.
  - The proxy is node-local, so it needs no NetworkPolicy change ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2: `:8222` admits no pod).
- **The expected stage** is a script constant, `PIN_NATS_STAGE=open`. `--nats-stage=` overrides it. [mi-06](sprint-mi-06.md) bumps the constant in its N1 and N3 PRs, and [mi-11](sprint-mi-11.md) bumps it at N4, so a plain `--cluster` run always checks the live stage.

| Stage | PASS when | FAIL when |
|---|---|---|
| `open` (today) | always; INFO lists the connections by `name` (7 on 2026-09-24: practice, review, assessment) | — |
| `n1` | every connection shows `authorized_user` = `legacy` or an `nkey`; INFO gives the counts | — |
| `n3` | every connection is authenticated with an **nkey** | any connection that is `legacy` or anonymous (`cluster.nats-legacy`) |
| `n4` | the n3 rule, plus `/varz` `.auth_required == true` (a strict `jq` test) | **anything else** in `auth_required` (`false`, `null` or absent all count as false: `cluster.nats-auth`), or the n3 rule |

Two negative tests, both of which **must FAIL today**:
- `--nats-stage=n3`. Today's `/connz?auth=true` shows `authorized_user` and `nkey` as `null` on all 7 connections.
- `--nats-stage=n4`. With auth off, nats-server **omits** `auth_required` from `/varz` (read 2026-09-25: `{"version":"2.14.6","auth_required":null}` through `jq`). An `== false` test would pass vacuously.

Together they prove neither check is vacuous. Re-verify the field names on 2.14.6 when N1 lands (mi-06).

### 5 · NetworkPolicy presence [H]

- **Reads.** Compare `get networkpolicy -A` against the embedded `expected-netpol.tsv` (`namespace <TAB> name <TAB> added-by`).
  - A missing entry → **FAIL** (`cluster.netpol`).
  - A policy in `xlearn`, `databases`, `messaging` or `xlearn-runner` that isn't listed → INFO.
- **Seed.** `kubescope/kubescope`: chart-rendered, and it guards a cluster-admin console. Today there are no policies in the four ADR namespaces.
- **Growth.** [mi-14](sprint-mi-14.md) adds the `xlearn-runner` policies, [mi-03](sprint-mi-03.md) adds the MI-5/5a ones, and [m3-07](sprint-m3-07.md) adds judge's. Each sprint adds its own rows.
- **Skip vendor-managed policies:** `flux-system`, and `longhorn-system`, whose names change with chart upgrades.

### 6 · Steal and CPU (TR-*) [H]

- **Source.** Parse `sar -u -f` over every `/var/log/sysstat/sa??` file, with `LC_ALL=C` and columns located by header name. There were 17 files on 2026-09-24.
- **Missing data is never a PASS.** If `sar` isn't installed, or there are no `sa??` files, or none covers the last 24 h, emit **WARN** `host.steal` and `host.cpu`: *"sar unavailable — TR-STEAL unmeasured"* (or *"TR-CPU unmeasured"*). The M3 hard checklist's "TR-STEAL not firing" item can't be ticked from that run. A rebuilt node gets sysstat back through `host-bootstrap.sh` (see the intro above). The binary is `SAR_BIN` (default `sar`), so the missing case can be tested with `SAR_BIN=/nonexistent`.
- **Report:**
  - the last 24 h steal p95;
  - the per-day steal p95 for the last 7 days;
  - the worst 1 h rolling average of steal;
  - the per-day maximum 60-minute user + system average;
  - the **full-history steal p95**, printed as INFO. The M3 hard checklist's "TR-STEAL not firing" item reads this.
- **WARN with the trigger id** ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5):

  | Trigger | Condition |
  |---|---|
  | `TR-STEAL` | 24 h p95 > 10% on ≥ 3 of the last 7 days |
  | `TR-STEAL` | any 1 h average > 25% |
  | `TR-STEAL` (early signal) | the latest 24 h p95 > 10% |
  | `TR-CPU` | user + system 60-minute average > 50% on ≥ 3 of 7 days |

- **Baseline** (t7 §4): 9-day p95 5.2%, and 7.1% on 9/24.

### 7 · Read-only proof, lint, `--json`, README; PR and merge [H]

- **`hack/host-lint.sh`:**
  - Assert that `host-verify.sh` reaches `kubectl` only through `kc_get`/`kc_top`/`kc_raw`.
  - Assert that no `kubectl` verb outside `get|top` appears: no `apply`, `create`, `delete`, `edit`, `patch`, `replace`, `scale`, `exec`, `cp`, `label`, `annotate`, `drain`, `cordon`, `rollout` or `port-forward`.
  - Assert that it has **no output redirection to a path**: `>`, `>>`, `>|` or `&>`, followed by anything other than `/dev/null` or an fd duplication (`>&N`, `N>&M`, `N>&-`).
    - Ignore comparisons inside `[[ ]]` and `(( ))`, single-quoted strings (awk programs), comment lines, and the embedded TSV blocks.
    - **Input redirection is allowed:** `<`, `<<`, `<<<` and `<(…)`. The `--budget-file`/`--netpol-file` reads need it.
    - Use exactly this. It's tested on 2026-09-25's `host-verify.sh`, which has 0 hits (including lines 154/511/957/1077 and the `2>&1 >/dev/null` in `kc_get`). It catches `> /tmp/x`, `>>"$f"`, `2>err.log`, `&>out`, `$(cmd >/tmp/y)`, `>| f`, and a second target after `>/dev/null`.

      ```bash
      redir_hits() {  # prints offending lines with their real line numbers; empty = clean
        sed -E -e '/^# >>> .*\.tsv$/,/^# <<< .*\.tsv$/s/.*//' -e 's/^[[:space:]]*#.*//' \
               -e "s/'[^']*'//g" -e 's/\[\[[^]]*\]\]//g' -e 's/\(\([^)]*\)\)//g' \
               -e 's#([0-9]*|&)>>?[[:space:]]*/dev/null##g' -e 's/[0-9]*>&[0-9]+-?//g' "$1" \
        | grep -nE '(^|[[:space:];&|(])([0-9]*|&)>>?\|?[[:space:]]*[^&=[:space:]]' || true
      }
      hits=$(redir_hits host-verify.sh)
      if [[ -n $hits ]]; then echo "host-lint: host-verify.sh writes to a path:" >&2; echo "$hits" >&2; rc=1; fi
      ```

    - It's a line-level heuristic. A `>` inside a double-quoted string that's followed by a space and a word would be flagged. Reword the message rather than weaken the regex.
  - Check that the embedded TSVs equal the files.
  - Run `shellcheck -S style` on `host-*.sh`, `sample-top.sh` (and `chart-diff.sh` once mi-01 lands).
- **Header.** Update the script's header: *"`kubectl get`/`top` only; JSON reads via `get --raw` through the API-server proxy"*.
- **`--json`.** One JSON object per check line (`{"status","id","msg"}`, correctly escaped) plus a summary object, for pasting into status.md or a PR. Every line must parse with `jq -c .`.
- **Usage and README.** Update the usage text and the infra README host-scripts section: each new flag, and where each one is used (after host changes; the release checklist before contract, erase and GA tags; `--with-runner` in mi-08/mi-09; `--nats-stage` in mi-06/mi-11).
- **PR.** Open it with this output in the body: `host-lint.sh`, a piped node run (`--cluster` green), the `--with-runner` FAIL, the `--nats-stage=n3` and `--nats-stage=n4` FAILs, the `SAR_BIN=/nonexistent` WARN, and the `--json` parse check. Merge it. infra has no CI.
- **Agents verify by piping only:** `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`, which writes nothing on the node. Every run in this sprint uses that form.
- **Refreshing the node copy is gated [O].** `scp hack/host-verify.sh vps:/root/` is a node write, and `ssh vps` stays read-only unless the owner explicitly asks. So after the merge, **ask the owner**:
  - either the owner refreshes `/root/host-verify.sh` (the README way);
  - or the owner gives an explicit OK and the agent runs the `scp` once, then `ssh vps 'bash /root/host-verify.sh --cluster'` once.

  Record which happened. If neither happens this session, record *"`/root/host-verify.sh` stale: owner to refresh"* as a pending owner item. The piped run is the sprint's proof either way.

### 8 · Throwaway top sampler for MI-11a (owner OK) [H]

- **Approval.** Ask the owner before starting it. It writes on the node (`/root/sample-top.sh`, `/var/tmp/xlearn-top.tsv` and `/var/tmp/xlearn-top.pid`), and `ssh vps` is otherwise read-only.
- **The script:** `hack/sample-top.sh`.
  - It appends `k3s kubectl top pods -A --containers --no-headers`, timestamp-prefixed, to `/var/tmp/xlearn-top.tsv` every 5 minutes.
  - It stops itself after `SAMPLE_HOURS` (default 168).
  - It writes its PID to `/var/tmp/xlearn-top.pid`.
- **Start it once:** `scp hack/sample-top.sh vps:/root/ && ssh vps 'nohup setsid bash /root/sample-top.sh </dev/null >/dev/null 2>&1 &'`.
- **It's not a timer, CronJob or alert (D34).** The file survives a reboot; the loop doesn't. Restart it after one.
- **Expected size:** about 9 MB over 7 days.
- **Hand-off:** [mi-08](sprint-mi-08.md) reads ≥ 48 h of samples, sets the p95 × 1.5 budgets, stops the loop and deletes the file.
- If the owner declines, record that. mi-08 then keeps the `top` × 1.2 budgets.

### 9 · MI-0 follow-up [O]

The owner confirms the H0 result (event `ev-mi0`): `uname -r` = 6.8.0-142, and `host-verify --cluster` is
green after the reboot. On 2026-09-25 the node was still on 6.8.0-90, so the reboot hadn't happened yet.

The S0 vmstat log is `/tmp/xlearn-s0-vmstat.log`, written by a self-terminating `vmstat -t -w 60 4320` that
started 2026-09-24 07:52 UTC. Its 72 h window ends **2026-09-27 ≈ 07:52 UTC**. There are two branches:
- **`ev-mi0` happened before the window ended.** The reboot ended the sampler, and tmpfiles empties `/tmp`
  at boot. The owner says whether the log was copied off before the reboot.
- **`ev-mi0` hadn't happened by the window's end.** The log is complete and still on the node, so it has to
  be collected and then removed. Do this before `ev-mi0` if it's still pending.
  - Copying it off is a read. The agent may do it: `scp vps:/tmp/xlearn-s0-vmstat.log <scratchpad>/`.
  - **Deleting it is a node write [O]:** the owner runs `ssh vps rm /tmp/xlearn-s0-vmstat.log`, or explicitly OKs the agent doing it.
  - Record the deletion. It closes the pending S0 owner item.

If a copy exists (either branch), the agent computes steal p50, p95 and max from its `st` column, and appends an
S0 row to [t3 §15](../research/t3-sandbox.md). If the log was lost, record "lost at reboot; sar (9+ days) is
the source". sar already holds the history (rollout §2, MI-0).

### 10 · Record [X]

- In [`../status.md`](../status.md):
  - **MI track:** MI-0 ✅ (date, kernel) and MI-8 ✅ (infra PR, date);
  - the **Sprint board** row.
- **Decisions log:**
  - the baseline memory sum (each term, margin, `--with-runner` margin);
  - the steal baselines (24 h, 7-day, full-history p95);
  - the sampler start time and PID (mi-08 may read from start + 48 h), or "declined";
  - the S0 result;
  - the `PIN_NATS_STAGE` convention.
- Update this file's Status table. Ship it as an xlearn docs PR.

## Acceptance criteria

- [ ] `host-verify --cluster` runs green (no FAIL) on the live node and prints a line for every [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3 check: memory sum, Flux, pod health, volumes and disk, NATS, NetworkPolicy, steal and CPU.
- [ ] Each printed memory-sum term (Σ limits, limitless budgets, surge, host, capacity − 512 Mi) matches a hand calculation from the same live reads. The totals are within ±0.3 GiB of task 1's expected ≈ 13.7 GiB PASS (≈ +1.4 GiB) and, with `--with-runner`, ≈ 16.7 GiB **FAIL** (≈ −1.6 GiB). ADR-0035 §5 is context, not the target.
- [ ] All 23 limitless containers match a budget entry (no `cluster.mem-budget` WARN), and a `--budget-file` without one entry produces that WARN.
- [ ] `--nats-stage=open` PASSes today. `--nats-stage=n3` **FAILs today**, listing the anonymous connections, and `--nats-stage=n4` **FAILs today** (`auth_required` absent).
- [ ] With `sar` unavailable (`SAR_BIN=/nonexistent`), `host.steal` and `host.cpu` WARN "unmeasured" and never PASS.
- [ ] The script is provably read-only: the `host-lint.sh` verb and output-redirection assertions pass (task 7's `redir_hits`), shellcheck is clean, and the embedded TSVs equal the files.
- [ ] `--json` output parses line by line with `jq -c .`.
- [ ] The sampler is running with the owner's OK (start time recorded), or it was declined and that's recorded.
- [ ] `docs/v2/status.md` records MI-0, MI-8 ✅, the memory-sum and steal baselines, and the S0 result.

## Release

**infra PR(s) only**, with no tag:
- One `../infra` PR carries `hack/host-verify.sh`, `memory-budget.tsv`, `expected-netpol.tsv`, `sample-top.sh`, `host-lint.sh`, the `host-bootstrap.sh` changes (`jq` + `sysstat` in `PKGS`, sysstat collection enabled) and the README.
- The script ships the moment it's merged: the piped form (`ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`) always runs the merged version. The `/root` copy is refreshed by the owner, or with the owner's explicit OK (task 7). It's used on demand, and in every contract, erase and GA tag checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6).
- The status rows ship as an xlearn docs PR.
- **Rollback:** `git revert`. The script is read-only, so it leaves no residue.

## Definition of Done

- The infra PR is merged with its check output in the body.
- A green piped `--cluster` run of the merged script is recorded. `/root/host-verify.sh` is either refreshed (by the owner or with the owner's OK) or recorded as a pending owner item.
- The sampler is running, or its decline is recorded.
- Statuses are updated (this file + [`../status.md`](../status.md)), and the xlearn docs PR is merged.
- Local `main` is synced in both repos.

## Risks / watch-outs

- **Working-set p95 needs samples.** Until mi-08 has ≥ 48 h of them, limitless budgets are `top` × 1.2 and the line says so. Don't present them as p95.
- **Tables must be embedded.** A TSV read from `hack/` works on the laptop and fails silently on the node. host-lint is the guard.
- **NATS monitor path.** Use the pod proxy (`nats-0:8222`). The `nats` Service has no 8222 port. If the StatefulSet is renamed or scaled, find the pod by label.
- **`/connz` field names** (`authorized_user`, `nkey`) must be re-verified at N1 on nats-server 2.14.6, or the n3 check could pass vacuously. Its negative test today, which must FAIL, guards the anonymous case.
- **Absent fields read as "false".** nats-server omits `auth_required` when auth is off, so only a strict `== true` makes n4 meaningful. The n4 negative test guards that, just as the n3 test guards the anonymous case.
- **The surge term is the fleet roll only** (ADR-0035 §5). A Flux or vendor upgrade can surge a 1 Gi controller (512 Mi after MI-11a). The script prints that figure as INFO, and the owner re-runs `host-verify` after such an upgrade.
- **No sar means no steal verdict.** A rebuilt node without sysstat would otherwise pass TR-STEAL silently. The check WARNs "unmeasured", and `host-bootstrap.sh` now installs and enables sysstat.
- **Restart counting is approximate** (no per-restart history). A flapping pod shows up as `restartCount` with a recent termination, which is enough to look.
- **`top` depends on metrics-server,** itself a limitless container. If it's down, the unbudgeted fallback can't be computed: WARN, don't FAIL.
- **The sampler must not become a monitor (D34).** It's self-terminating, has no notifications, and is deleted by mi-08.
- **Steal is elevated and noisy.** TR-STEAL could fire before M3 and pull R2 ahead of it ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5). Record the numbers as they are.
