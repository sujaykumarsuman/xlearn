# Sprint mi-09 — October host-window prep: sandbox block, L23 kubelet args, pid limits, bumps (MI-11)

> **Milestone:** MI — infra-first track (rollout step **MI-11**; sprint ids `mi-NN` ≠ rollout steps `MI-N`, and this is **not** MI-9) · **Track:** infra (host) · **Order:** 26
> **Prereqs:** [spk-01](sprint-spk-01.md) + [spk-02](sprint-spk-02.md) (spike **GO**: P0–P3 + image volume) · batched with [mi-08](sprint-mi-08.md) (MI-11a) in the window · reads [mi-02](sprint-mi-02.md)'s `host-verify --cluster` flags
> **Unblocks:** [mi-10](sprint-mi-10.md) (runner dark, MI-12) once the owner has run the window. ADR-0030's acceptance is [m3-03](sprint-m3-03.md)'s first task, so runner code never waits on this sprint.
> **Release action:** **infra PR(s) only.** The host-script PR merges before the window (the owner applies the scripts by hand in it). The CNPG 18.6 change, and the k3s pin change if v1.36.5 is GA, are pushed as **branches without a PR** (`chore/cnpg-18.6-host-window`, `chore/k3s-v1.36.5-host-window`); the window's runbook opens and merges their PRs (the same pattern as [mi-11](sprint-mi-11.md)'s N4 fallback), so this session leaves no open PR behind. The runbook and status ride an xlearn docs PR (merge only, no tag).
> **Calendar:** week 4 (Sat 2026-10-17 → Fri 10-23), right after mi-08. Prepares the owner event **ev-host-window, Sat 2026-10-24** (BP4).
> **Execute with:** [`../prompts/prompt-mi-09.md`](../prompts/prompt-mi-09.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Take the spike's final host artefacts | H | ⬜ |
| 2 | Sandbox block in `host-bootstrap.sh` (opt-in `--with-sandbox`) | H | ⬜ |
| 3 | L23 kubelet reservations + pod pid limit | H | ⬜ |
| 4 | `host-verify` asserts + `hack/host-bom.txt` + lint | H | ⬜ |
| 5 | Read-only pre-window baseline on the live node | H | ⬜ |
| 6 | Throwaway-VM dry run | H | ⬜ |
| 7 | CNPG 18.6 branch, held for the window | I | ⬜ |
| 8 | k3s pin: v1.36.5 only if GA, held branch for the window | H | ⬜ |
| 9 | infra README: host table, flags, Rebuild order (the DR runbook) | I | ⬜ |
| 10 | Window runbook `docs/v2/runbooks/host-window-2026-10.md` | X | ⬜ |
| 11 | Owner reviews the runbook (~15 min) | O | ⬜ |
| 12 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track row MI-11).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [spk-01](sprint-spk-01.md) and [spk-02](sprint-spk-02.md) report **GO** (P0–P3 + image volume), and their results are in [t3](../research/t3-sandbox.md) §16.1–16.4 (the final host files verbatim, plus the MI-10 verdict)
- [ ] The spike did **not** force R1b. R1b is a D21 move trigger: the runner goes to its own VPS (R2), and this window's sandbox block moves with it. If R1b, stop and report to the owner
- [ ] MI-0 done: the node runs the H0 kernel (≥ 6.8.0-142; it was still 6.8.0-90 on 2026-09-24) and `host-verify --cluster` was green after the reboot
- [ ] [mi-02](sprint-mi-02.md) merged: `host-verify --cluster`, `--with-runner` and `--nats-stage=…` exist (the runbook calls them)
- [ ] MI-11a merged, or batched into the same window ([mi-08](sprint-mi-08.md)). This gates the **window**, not this sprint
- [ ] The window date is booked with the owner (**Sat 2026-10-24**), together with the spike week (ev-spike-goahead)
- [ ] Parallel sessions: no open peer PR in `../infra` touches `hack/`, `infrastructure/database/cluster/cluster.yaml` or the README host sections (`gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents)

## Goal

Turn the spike results into **scripted host changes** and a **runbook the owner executes** on Sat
2026-10-24. The window is batched with MI-11a (BP4) and costs **one k3s restart and one PG restart**.
It carries:
- the **host sandbox block** ([ADR-0030 §5](../../adr/0030-runner-technology-and-host-hardening.md#5-host-and-cluster-hardening) step A6): sysctls, the containerd `judge` runtime drop-in, the `xlearn-runner` AppArmor and pod-level seccomp profiles, and the kubelet subuid/subgid range;
- **L23** ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)): kubelet `system-reserved` 1 GiB, `eviction-hard memory.available<500Mi`, and a pod pid limit. The kubelet then evicts the lowest-priority pod (the runner, once it exists) before the kernel OOM killer picks a victim;
- **CNPG 18.4 → 18.6**, and **k3s v1.36.5 only if it is GA** (Track B bumps folded into the same restarts).

This sprint changes nothing on the live host. `ssh vps` stays read-only. It delivers merged scripts, held
branches, a dry-run proof and a reviewed runbook.

## Scope

**In**
- The spike's final host artefacts ([spk-01](sprint-spk-01.md) P0/P1, [spk-02](sprint-spk-02.md) P3 amd64) as the source of truth for every host file.
- `../infra/hack/host-bootstrap.sh`: the sandbox block plus the L23 kubelet args and `pod-max-pids`, opt-in behind `--with-sandbox`, and ordered **before** the pinned k3s install so a rebuild needs no extra restart ([t3 §8.7](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded)).
- `../infra/hack/host-verify.sh`: asserts for all of it (`--expect-sandbox`); a new `hack/host-bom.txt`; the `host-lint.sh` extension.
- CNPG `18.6-system-trixie` branch, and the k3s v1.36.5 pin branch only if GA. Both are pushed without a PR
  and held for the window.
- The infra README host sections, including **Rebuild order** (the DR runbook; the infra repo has no separate `dr-runbook.md`).
- The window runbook `docs/v2/runbooks/host-window-2026-10.md` (the owner executes it).

**Out**
- **Executing the window**: owner event ev-host-window, Sat 2026-10-24.
- **Accepting ADR-0030**: [m3-03](sprint-m3-03.md) task 1. This sprint notes divergences from t3 §8.7 in its PR only. The window intentionally applies the §5 host block on the spike GO while ADR-0030 is still Proposed ([status.md decisions log](../status.md#decisions-log)).
- **MI-11a limit hygiene**: [mi-08](sprint-mi-08.md). The runbook merges mi-08's PRs in the window if they are still open; this sprint doesn't redo them.
- **Runner deployment** (MI-12): [mi-10](sprint-mi-10.md). Runner caps and values live there, not on the host.
- **Per-language exec seccomp allowlists** (Go, C++, Python, amd64): they ship **inside the runner image** ([m3-04](sprint-m3-04.md)). The host holds only the pod-level profile.
- **`sandbox-guards`** (namespace, VAP, RuntimeClass, Quota, NetworkPolicies): [mi-14](sprint-mi-14.md).
- **Track B finish** (xlearn egress, PG role limits, Renovate, N4, the monthly-window runbook): [mi-11](sprint-mi-11.md).
- **L22** CNPG memory 1 → 2 GiB: trigger-only (TR-MEM), not in this window.
- Any alert, timer or CronJob (D34). `host-verify` stays on demand.

## Tasks

### 1 · Take the spike's final host artefacts [H]

Read the spike results in [t3](../research/t3-sandbox.md) §16: §16.1 (spk-01, P0–P2 on arm64), §16.2
(spk-02, the P3 amd64 replay, including the **amd64 pod-level seccomp profile verbatim**) and §16.4 (the
MI-10 verdict and the proposed ADR-0030 deltas). They name the chosen jail mechanism: go-sandbox `forkexec.Runner`
without CLONE_NEWUSER (R1), or the nsjail `--disable_clone_newuser` fallback. For each host artefact,
they also give its final content or its diff against [t3 §8.7](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded):

| Artefact | t3 §8.7 baseline | Take from the spike |
|---|---|---|
| containerd drop-in | runtime `judge`, runc v2, `cgroup_writable`, `SystemdCgroup = true`; no `.tmpl`, no `BinaryName` | P0: merged into the rendered config? (P0 falls back to a minimal `.tmpl` if not, and records it) |
| AppArmor `xlearn-runner` | abi 4.0, **no `userns` rule**, jail-path mounts, `pivot_root`, own cgroupfs writes, 5 caps, `deny ptrace` | P1: every rule the positive checks needed; whether `setpcap` stayed |
| Pod-level seccomp `xlearn-runner.json` | RuntimeDefault + SYS_ADMIN, minus the new mount API, bpf/perf/fanotify, keyctl, io_uring, userfaultfd, lookup_dcookie, syslog; `socket()` limited to AF_UNIX and AF_INET/INET6 `SOCK_STREAM` | **P3 (amd64)**, the file replayed on amd64, with `architectures` x86_64/x86/x32. The arm64 P0 file is **not** the one to ship |
| kubelet userns range | `kubelet:<start>:7208960` (110 × 65,536) in `/etc/subuid` + `/etc/subgid`; `getsubids` | P0: the range used and the recreate × 50 result; the package that ships `getsubids` on noble |
| Sysctls | `io_uring_disabled=2`, `unprivileged_bpf_disabled=2`, `vm.unprivileged_userfaultfd=0`, `dmesg_restrict=1`, `kptr_restrict=2`; assert `perf_event_paranoid ≥ 3`; pin `apparmor_restrict_unprivileged_userns=1` | P0/P1: unchanged unless recorded |

- **If §16 carries diffs only** for an artefact, rebuild that file from t3 §8.7 plus every recorded diff.
  The spike sessions never commit their VM files. Task 6 re-validates the rebuilt files.
- **If the spike landed on R1-U** (a userns variant), the AppArmor profile gains exactly the `userns` rule the
  spike validated. Follow the record; don't improvise.
- **PR description:** a table of artefact · t3 §8.7 · final · spike row that forced the change. ADR-0030 is
  **not** edited here; m3-03 accepts it with the same table.
- The caps list (5 caps, or 4 if P1 showed `SETPCAP` unneeded) is a runner **value**. Hand it to
  [mi-10](sprint-mi-10.md) and record it in status.md. Only the AppArmor `capability` lines are a host concern.

### 2 · Sandbox block in `host-bootstrap.sh` [H]

Replace block (8) `FUTURE (v2 T3)` with a real block, **moved before block (6) k3s**. On a rebuild the
files then exist before the pinned k3s install, so no second restart is needed (t3 §8.7). Renumber the
comments.

- **Opt-in `--with-sandbox`.** A routine bootstrap run before the window writes nothing new, so it leaves no
  "pending k3s restart" state behind. The runbook, the DR Rebuild order and the monthly window pass the flag.
- **Idempotent and dry-run aware** like blocks (0)–(7): `run …`, `[dry-run]` output, compare-then-write.
- **Never restarts k3s.** When a file that k3s reads changed (the drop-in or the kubelet config, task 3),
  print `k3s restart required to apply: <files>`, the way the summary prints REBOOT REQUIRED. The restart is a
  runbook step.
- **Embed each artefact as a heredoc**. `host-verify` must still run over `ssh vps 'bash -s' < hack/host-verify.sh`
  with nothing copied. Task 4 puts each artefact's sha256 in the shared constants.

| Piece | Path on the node | Takes effect | Today (read-only, 2026-09-24) |
|---|---|---|---|
| Sysctls | `/etc/sysctl.d/60-xlearn-sandbox.conf`, applied with `sysctl --system` | live | `io_uring_disabled=0` → **2**; `kptr_restrict=1` → **2**; the rest are already at target (`perf_event_paranoid=4`, `apparmor_restrict_unprivileged_userns=1`) |
| containerd drop-in | `/var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d/20-judge.toml` | k3s restart | the rendered `config.toml` already `imports` `config-v3.toml.d/*.toml`; the directory doesn't exist yet |
| AppArmor | `/etc/apparmor.d/xlearn-runner`, loaded with `apparmor_parser -r` | live; reloaded at boot | not present |
| Seccomp (pod-level, amd64) | `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` (create the dirs `0755 root`) | read at pod create | `/var/lib/kubelet/seccomp` doesn't exist |
| userns range | system user `kubelet`; `kubelet:<start>:7208960` in `/etc/subuid` **and** `/etc/subgid`; the `getsubids` package | kubelet start (k3s restart) | only `ubuntu:100000:65536`; no `getsubids` |

The `<start>` value comes from the spike. It must be a multiple of 65,536, must not overlap
`ubuntu:100000:65536` (so ≥ 196,608), and must end well below UINT32_MAX (kubernetes #139916).
Bootstrap refuses an overlapping range.

### 3 · L23 kubelet reservations + pod pid limit [H]

Under the same `--with-sandbox` flag and the same restart, write a k3s config drop-in
`/etc/rancher/k3s/config.yaml.d/60-xlearn-kubelet.yaml`:

```yaml
# L23 (ADR-0035 §4). Applied at the next k3s restart. Owner: hack/host-bootstrap.sh.
kubelet-arg:
  - system-reserved=cpu=250m,memory=1Gi
  - eviction-hard=memory.available<500Mi,nodefs.available<5%,imagefs.available<5%
  - pod-max-pids=4096
```

- **Why a `config.yaml.d` drop-in** and not the systemd unit: the unit's args are the `PIN_K3S_ARGS` constant
  that `k3s.args` compares, and changing them means re-running the installer. `host-verify` already WARNs
  when `/etc/rancher/k3s/config.yaml` exists. Keep that check; the drop-in joins the BOM instead. If the
  task 6 dry run shows k3s v1.36.x ignoring `config.yaml.d`, fall back to `/etc/rancher/k3s/config.yaml`,
  change the `k3s.config` check to match, and record the call.
- **Live today** (`configz`, 2026-09-24): `evictionHard` {`imagefs.available 5%`, `nodefs.available 5%`},
  `systemReserved` unset, `podPidsLimit` −1, capacity 16,376,040 Ki (15.62 GiB), allocatable = capacity.
  **After:** allocatable memory ≈ 15.62 − 1 − 0.49 ≈ **14.1 GiB**, allocatable CPU 3,750m.
- **Keep `enforce-node-allocatable` at its default (`pods`).** `system-reserved` only lowers allocatable.
  Enforcing it on a system cgroup could OOM k3s itself.
- **`eviction-hard` replaces the kubelet's set.** The task 4 check prints the whole map. If a signal the
  kubelet used to default (such as `nodefs.inodesFree`) disappears, add it explicitly and record the call.
- `pod-max-pids=4096` counts threads. Task 5 measures the live maximum first. Longhorn instance-manager and
  Postgres are the likely heavy ones.

### 4 · `host-verify` asserts + `hack/host-bom.txt` + lint [H]

Replace the `TODO(v2 T3)` section of `host-verify.sh`:

- **Presence rule.** If any sandbox artefact exists, assert all of them; a partial install is a **FAIL**. If
  none exists, print INFO `sandbox block not installed`, unless `--expect-sandbox` is set, which makes it a
  FAIL. From the window on, every runbook and the DR Rebuild order pass `--expect-sandbox`. Pre-window runs
  on the live node therefore stay exactly as green as today.
- **Checks** (one PASS/WARN/FAIL line each, as the script does today):

| Check | Asserts |
|---|---|
| `sandbox.sysctl` | the five values live **and** persisted in `60-xlearn-sandbox.conf`; `perf_event_paranoid ≥ 3`; `apparmor_restrict_unprivileged_userns = 1` |
| `sandbox.containerd` | Always: `k3s crictl info` lists runtime `judge` (runc v2, `SystemdCgroup`, `cgroup_writable`); the `judge` runc is the default runc (same binary and version); the rendered `config.toml` still has runc `SystemdCgroup = true`. Then **branch on the spike record** (a shared constant, e.g. `SANDBOX_CONTAINERD_MODE=dropin\|tmpl`): **`dropin`** (the t3 §8.7 path): the rendered `config.toml` still imports the `config-v3.toml.d/*.toml` glob, and the drop-in sha256 = BOM. **FAIL loudly** if a k3s upgrade renamed the directory or dropped the import. **`tmpl`** (spk-01's recorded fallback): `config-v3.toml.tmpl` sha256 = BOM, and no import assertion; the rendered runc and `judge` checks above carry the weight |
| `sandbox.apparmor` | `xlearn-runner (enforce)` in `/sys/kernel/security/apparmor/profiles`; the file's sha256 = BOM |
| `sandbox.seccomp` | the file's sha256 = BOM |
| `sandbox.userns` | the `kubelet` user exists; its range is in both `/etc/subuid` and `/etc/subgid`, a multiple of 65,536, and overlaps no other entry; `getsubids kubelet` resolves it |
| `kubelet.config` | the drop-in's sha256 = BOM; `k3s kubectl get --raw /api/v1/nodes/<node>/proxy/configz` shows `systemReserved` {cpu 250m, memory 1Gi}, `evictionHard` including `memory.available: 500Mi` (print the full map), `podPidsLimit: 4096`; node allocatable memory ≈ capacity − 1 GiB − 500 Mi |

- **Still read-only.** Every call is a `get` verb (`get --raw` included), so [mi-02](sprint-mi-02.md)'s
  read-only lint in `host-lint.sh` must still pass. No temp files.
- **`hack/host-bom.txt`**, one line per artefact: `path · sha256 · source (spike record row or t3 §8.7) ·
  written by (block) · takes effect (live | k3s restart)`. Plus the pins observed after the window: kernel
  floor, k3s, containerd, runc.
- **Shared constants:** add one `SANDBOX_SHA_*` per artefact, plus `SANDBOX_CONTAINERD_MODE` (`dropin`, or
  `tmpl` if the spike recorded the fallback; bootstrap then writes `config-v3.toml.tmpl` instead of the
  drop-in), byte-identical in both scripts.
  `host-lint.sh` gains one check: heredoc content sha256 = constant = BOM line, for every artefact.
- Shellcheck clean (`-S style`).

### 5 · Read-only pre-window baseline on the live node [H]

Over `ssh vps`, reads only. Paste the results into the host-script PR. This is the "before" picture the
runbook compares against:
- `k3s crictl info` runtimes; the rendered `config.toml`; `configz`; node capacity and allocatable;
  `/etc/subuid` and `/etc/subgid`; the task 2 sysctls.
- **The pid ceiling:** the maximum `pids.current` per pod under `/sys/fs/cgroup/kubepods.slice/`. If any pod
  is above 2,048, raise `pod-max-pids` (or stop and ask) before the window.
- **The memory picture:** `host-verify --cluster --with-runner` (from mi-02, after mi-08 if merged). Record
  `memory.available` today: after L23, eviction starts 500 Mi above the kernel OOM.
- `host-bootstrap.sh --dry-run` **without** `--with-sandbox` plans nothing beyond what `main`'s copy plans.
  Every run plans kernel-meta upgrades and any pending security updates (infra README), so compare, don't
  expect silence: dry-run `main`'s copy and the PR branch's copy back to back and diff the two outputs. The
  diff must be empty. The script warns against `bash -s` piping, because apt children can eat stdin. So
  stream each copy into a `mktemp` file on the node, run it with `--dry-run </dev/null`, and delete the file.
  Those temp files are the only writes, and they change no host state.

### 6 · Throwaway-VM dry run [H]

- `multipass launch 24.04 --cpus 4 --memory 8G --disk 30G` (as spk-01 P0), then `apt full-upgrade` and a
  reboot.
- `host-bootstrap.sh --upgrade --with-k3s` (the pinned v1.36.4), then `host-verify` once as the VM's
  baseline, then `--with-sandbox`, then `systemctl restart k3s`, then **`host-verify --expect-sandbox`**
  (host section only). The gate: every `sandbox.*` line and `kubelet.config` (read through `configz`) **PASS**,
  and no host line that passed in the baseline regresses.
- **Don't gate on `--cluster` here.** A bare multipass k3s has no Longhorn, CNPG or Flux, and none of
  [mi-02](sprint-mi-02.md)'s inputs (NATS `/varz` through `nats-0`, the expected NetworkPolicies, the memory
  budget), so `check_cluster` FAILs by design. Optionally run `--expect-sandbox --cluster` too and record
  the expected VM-only FAILs (`cluster.longhorn`, `cluster.cnpg`, `cluster.flux-ks`, `cluster.flux-hr`, the
  NATS and NetworkPolicy lines). `--expect-sandbox --cluster` green is the **live-node** gate, at runbook step 6.
- Record: `configz` (system-reserved, eviction-hard, podPidsLimit), `crictl info` showing `judge`,
  allocatable, how long the k3s restart took, and whether running pods survived it.
- **If task 8 bumps k3s:** repeat on a fresh VM along the window's path: `--with-sandbox --with-k3s
  --replace-k3s`, where one installer restart applies everything. The drop-in merge was spike-validated on
  v1.36.4 only.
- **Arch caveat:** multipass on the M3 Max is arm64, so no pod loads the amd64 seccomp file here (P3 proved
  that file on amd64). This run proves the scripts, the drop-in merge, the kubelet args and the asserts.
- `multipass delete --purge` afterwards. Nothing from the VM is committed.

### 7 · CNPG 18.6 branch, held for the window [I]

- `infrastructure/database/cluster/cluster.yaml`: `imageName` `ghcr.io/cloudnative-pg/postgresql:18.4-system-trixie`
  → `18.6-system-trixie` (verified to exist in [rollout §2](../rollout-plan.md#2-mi-infra-track) MI-11).
  Record the image digest in the commit message and the runbook.
- The effect is `primaryUpdateStrategy: unsupervised` with `instances: 1`, so the merge causes **one PG
  restart**: seconds to a minute. pgx pools reconnect and the outboxes buffer.
- MI-2's prune guard on the Cluster ([mi-01](sprint-mi-01.md)) is untouched.
- Push it as the branch **`chore/cnpg-18.6-host-window`**, one conventional commit, **with no PR**. AGENT.md
  allows no open PR at session end, and a PR opened now couldn't be merged by this session anyway (the change
  restarts PG, so it belongs in the window). Put the branch's compare link in the runbook so the owner reviews
  the diff in task 11. The runbook's step 7 opens the PR from the branch and merges it (the owner, or an
  agent session assisting in the window with the owner's go-ahead).

### 8 · k3s pin: v1.36.5 only if GA, held branch for the window [H]

- **On the day,** check whether `v1.36.5+k3s1` is a non-prerelease GitHub release (not `-rc`) and on the
  v1.36 channel. It brings containerd 2.3.5 and runc 1.4.3 (t3 §15; CVE-2026-53495).
- **If GA:** push a separate branch **`chore/k3s-v1.36.5-host-window`**, with **no PR**, that bumps
  `PIN_K3S_VERSION` in **both** shared-constant copies (`host-lint.sh` enforces this). Merging early would
  make every pre-window `host-verify` FAIL `k3s.version`. The runbook's step 5b opens its PR and merges it.
  Re-run task 6 on v1.36.5 before calling it GO for the window.
- **If not GA:** stay on v1.36.4 and say so in the runbook and status.md. The fixes wait for a monthly window
  (D22, [mi-11](sprint-mi-11.md)).
- **No way back but the snapshot.** Bootstrap never downgrades k3s, so the rollback for this step is R-d.
  That's one more reason to take the bump only when GA and dry-run.

### 9 · infra README: host table, flags, Rebuild order [I]

Same PR as tasks 2–4:
- "Host bootstrap & hardening": add rows for the sandbox block and the kubelet reservations, add
  `--with-sandbox` and `--expect-sandbox` to the flag tables, and replace "v2 judge sandbox (planned)" with the
  real section.
- **Rebuild order** (the DR runbook): add `--with-sandbox` to the bootstrap line (sandbox files before the
  pinned k3s install) and `--expect-sandbox` to both verify lines. Add one line saying that a Hostinger
  snapshot restore rewinds the host files too, so after any R-d restore you re-run bootstrap and verify.
- **Kernel reboot runbook**, step 1: flag that its off-node `pg_dumpall` conflicts with
  [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule) (no off-node
  dump in v2: PII off the node, bends D12). Leave the step as it is and ask the owner in the PR.
  [mi-11](sprint-mi-11.md)'s monthly-window runbook records the owner's call.

### 10 · Window runbook `docs/v2/runbooks/host-window-2026-10.md` [X]

The owner executes it (an agent session may assist). Every step gives the command, the expected output
and a stop condition:

0. **T−1 day:** runbook reviewed (task 11). Ready: mi-08's MI-11a PRs if still open, and the held branches
   `chore/cnpg-18.6-host-window` and (if GA) `chore/k3s-v1.36.5-host-window`, rebased on `main` and still
   lint-green. Copy scripts from `main`: `scp hack/host-bootstrap.sh hack/host-verify.sh vps:/root/`.
1. **Pre-checks.** The last Hostinger weekly image is ≤ 7 days old. `host-verify --cluster` gives the baseline:
   anything red now isn't the window's fault. No deploy or tag in flight (`flux get kustomizations`; peer
   sessions paused).
2. **Manual Hostinger snapshot**; wait until it completes. Keep the Hostinger VNC console open. **No
   off-node `pg_dumpall`** ([ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule)).
3. **Batched MI-11a** (only if mi-08's PRs are still open): merge them, then wait for Flux and the
   controller restarts.
   - **3b, always** (whether or not step 3 merged anything): `host-verify --cluster --with-runner` must show
     the memory sum inside the rule, about 0.9 GiB inside
     ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)),
     before any host file goes in. If it doesn't, stop. Record the numbers.
4. **Host files:** `bash /root/host-bootstrap.sh --with-sandbox --dry-run`, then apply it. Sysctls and
   AppArmor are live now. The drop-in, the kubelet config and the subuid range wait for the restart.
5. **One k3s restart:** (a) on v1.36.4, `systemctl restart k3s`; or (b) open the k3s pin PR from
   `chore/k3s-v1.36.5-host-window` (`gh pr create -R sujaykumarsuman/infra --head chore/k3s-v1.36.5-host-window …`),
   merge it, re-`scp` the scripts, and run `host-bootstrap.sh --with-sandbox --with-k3s --replace-k3s` (the installer restarts k3s
   once). Expect 1–2 minutes of API unavailability. Pods keep running; record any that restarted.
   - This restart doubles as the stress test for mi-08's 512 Mi Flux controller limits. Watch their
     restarts and OOMKills.
6. **`host-verify --expect-sandbox --cluster`** must be green: `judge` runtime listed, runc parity, drop-in
   hash, AppArmor enforce, seccomp hash, subuid, sysctls, `configz` (system-reserved, eviction-hard,
   `podPidsLimit` 4096), allocatable ≈ 14.1 GiB, and **no pod `Evicted`**.
7. **CNPG:** open the CNPG PR from `chore/cnpg-18.6-host-window` and merge it. Flux applies it and CNPG
   restarts the single primary (the one PG restart).
   Wait until the Cluster is healthy. Services reconnect and the outboxes drain (unsent → 0).
   - This restart is the `databases` namespace's first admission under [mi-08](sprint-mi-08.md)'s PSA
     `restricted` label. If the new CNPG pod is refused (a `FailedCreate` event), revert mi-08's `databases`
     label PR first, then let CNPG retry.
8. **`host-verify --cluster --expect-sandbox --nats-stage=<live>`** must be green. The live stage is `n3` if
   [mi-06](sprint-mi-06.md) has run N3, else `n1` or `open`. Smoke-test login, the dashboard and coach.
9. **Record** in `docs/v2/status.md`: MI-11 ✅; a window log line with the snapshot time, k3s version, CNPG
   image, memory-sum numbers, anything that restarted or was evicted, and the `configz` values.

**Rollback**
- **Host files:** the runbook lists the exact commands: `rm` the drop-ins and profiles, `apparmor_parser -R`,
  restore the sysctl file, `sysctl --system`. Bootstrap never deletes. Then restart k3s and run
  `host-verify --cluster`.
- **CNPG:** a revert PR (another PG restart; a same-major minor downgrade keeps PGDATA).
- **k3s bump:** R-d only.
- **Worst case:** R-d, the snapshot restore, in the [ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button)
  order. The restore rewinds the host files too.

### 11 · Owner reviews the runbook [O]

The owner reads the runbook (~15 min) before 10-24 and confirms the snapshot and VNC plan, the k3s
decision (task 8) and the `pg_dumpall` question (task 9). Approve, or request changes.

### 12 · Record [X]

In `docs/v2/status.md`:
- the MI track MI-11 row: "prepared", with the PR numbers, and the window booked for 2026-10-24;
- the Sprint board row;
- Decisions log lines: `config.yaml.d` vs `config.yaml`, `--with-sandbox`/`--expect-sandbox`,
  `SANDBOX_CONTAINERD_MODE` (drop-in or `.tmpl`), the k3s decision, the caps list handed to mi-10, and any divergence from t3 §8.7;
- the held branches (`chore/cnpg-18.6-host-window`, and `chore/k3s-v1.36.5-host-window` if GA) with their
  head commits and "opened and merged in ev-host-window", so the next session's peer checks don't take them
  for stragglers or delete them.

## Acceptance criteria

- [ ] On a throwaway VM with the new block (dry run), on the k3s pin the window will use: `host-verify --expect-sandbox` is green for every `sandbox.*` check and `kubelet.config`, with no host line regressed from the VM's baseline (`--cluster` is not gated on the VM; its expected VM-only FAILs are recorded if it was run)
- [ ] `hack/host-lint.sh` green: shared constants identical; heredoc = constant = BOM hash for every artefact; shellcheck clean; mi-02's read-only lint still passes
- [ ] On the live node, **without** `--with-sandbox`, the PR branch's `host-bootstrap.sh --dry-run` plans nothing beyond `main`'s copy (the two outputs diff empty), and `host-verify --cluster` is unchanged (sandbox: INFO only)
- [ ] The CNPG 18.6 branch (and the k3s pin branch, if GA) is pushed with no PR, lint-green, and linked from the runbook
- [ ] Runbook reviewed by the owner before 10-24
- [ ] The infra README host table, flags and Rebuild order are updated

## Release

**Infra PR(s) only.**
1. The host-script PR (tasks 2–4 + 9) **merges before the window**. Flux doesn't apply `hack/`, so merging
   changes nothing on the cluster. The owner copies the scripts from `main` in the window.
2. The CNPG 18.6 change and the k3s pin change (if GA) are pushed as **branches without a PR**
   (`chore/cnpg-18.6-host-window`, `chore/k3s-v1.36.5-host-window`). The window opens and merges their PRs at
   runbook steps 7 and 5b, so this session leaves no open PR behind (AGENT.md), the same pattern as
   [mi-11](sprint-mi-11.md)'s N4 fallback branch.
3. The xlearn docs PR (runbook + status) merges; `main` is build-only, so no tag and no deploy.

## Definition of Done

Host-script PR merged with lint green · the dry run recorded in the PR · held branches pushed (no PR) and recorded in status.md ·
runbook merged and owner-reviewed · statuses updated (this file + [`../status.md`](../status.md)) ·
nothing applied to the live host by the agent · no `kubectl apply` · no alert, timer or CronJob added (D34).

## Risks / watch-outs

- **If the spike ended on nsjail or R1-U**, the profile and drop-in differ from t3 §8.7. mi-09 must use the
  spike's final files (task 1), not the research draft.
- **One k3s restart and one PG restart:** snapshot first. A restore rewinds the host too.
- **The drop-in could stop merging.** A k3s upgrade that renames `config-v3.toml.d` or drops the `imports`
  line would leave the `judge` runtime silently missing. The verify check fails loudly, and task 6
  dry-runs the exact pin.
- **If P0 needed the `.tmpl` fallback**, the template replaces k3s's whole containerd config. It then has to
  be re-diffed against the upstream template on every k3s bump, including the monthly patches. Set
  `SANDBOX_CONTAINERD_MODE=tmpl`, hash the template in the BOM, and let `sandbox.containerd` assert the
  rendered runc `SystemdCgroup = true` and runtime `judge` without the import check (task 4), which would
  otherwise FAIL on a correct host.
- **Eviction becomes real.** Below 500 Mi available, the kubelet evicts. Before the runner exists, the
  victims are BestEffort pods and Burstable pods over their requests. That is why MI-11a comes first (step 3).
  Look for `Evicted` pods after the restart.
- **`pod-max-pids=4096` could starve a pid-heavy pod** (Longhorn instance-manager, Postgres backends).
  Task 5 measures first.
- **Seccomp architecture:** the host file must be the amd64 file from P3. The arm64 syscall set differs,
  and a wrong file breaks the runner pod at create time (in mi-10, not here).
- **The subuid range must not overlap** the node's existing `ubuntu:100000:65536` entry.
- **Merging a held branch early** breaks pre-window `host-verify` (k3s) or restarts PG outside the window (CNPG).
  The branches carry no PR, so a PR sweep won't merge them by accident; status.md names them.
- **Hand-applied host state** is the one sanctioned exception to GitOps. It stays scripted, BOM-hashed and
  asserted by `host-verify` after every reboot, k3s upgrade or rebuild.
- **Parallel sessions:** no tag and no fleet rollout may overlap the k3s restart. The runbook pauses them
  (step 1).
