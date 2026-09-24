# ADR-0030 — Runner technology (sandbox) & host/cluster hardening

- **Status:** Proposed. The mechanism spike (P0–P3) is **pending** and is a build-plan gate before M3. **Kept Proposed at the v2 build-plan sign-off (2026-09-24, BP2):** it is accepted at the **sandbox-spike GO** (spike week Mon 2026-10-12 → Fri 2026-10-16: sprints [spk-01](../v2/sprints/sprint-spk-01.md), P0–P2, and [spk-02](../v2/sprints/sprint-spk-02.md), P3 plus the image-volume spike), as task 1 of sprint [m3-03](../v2/sprints/sprint-m3-03.md), before M3. **§5 amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) (T7, 2026-09-24; ADR-0035 Accepted 2026-09-24):** NATS nkeys roll out server-first with fine ACLs in the same step, and the kubelet reservation joins the October host window (§7).
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0029](0029-judge-contract-and-learning-signal.md) (the runner contract this satisfies),
  [0027](0027-content-evalpack-and-user-data-model.md) (leak and integrity controls), [0028](0028-object-storage-and-backups.md)
  (no backups, which raises the stakes of a node compromise), [0009](0009-deployment-and-gitops.md), [0014](0014-nats-jetstream-topology-and-outbox-relay.md).
  v2 topic T3: [feasibility § T3](../v2/feasibility.md#t3--code-execution-sandbox--cluster-hardening) and the full
  [research appendix](../v2/research/t3-sandbox.md), which includes the S0 results.

## Context

xLearn v2 runs **untrusted learner code** (Go, C++, Python; later go-race and SQL) on the **single production node**. That node holds everything:
- CNPG, with **no backups**;
- the SOPS age key and the Flux deploy key;
- kubescope's cluster-admin;
- the JWT and coach master keys.

**Host facts (verified):**
- No `/dev/kvm`, so Firecracker, Kata and gVisor-KVM are impossible.
- Ubuntu 24.04's AppArmor restriction on unprivileged user namespaces is on.
- cgroup v2, with the systemd driver.
- k3s v1.36.4, containerd 2.3.4, runc 1.4.2.
- **The kernel was 6.8.0-90 while noble is at 6.8.0-142.** The Hostinger image had put the kernel meta-packages on hold, so it never auto-updated. They were unheld and 6.8.0-142 was installed on 2026-09-24 (H0; the reboot is due 2026-09-25).
- Core dumps are piped to apport as host root.
- Cgroup CPU time **includes steal** (`PARAVIRT_TIME_ACCOUNTING` is unset). Average steal is about 2.4%.

**Existing judges don't fit:**
- Judge0 and Piston need privileged containers and store expected outputs.
- DMOJ uses ptrace, which breaks TSAN, and is AGPL.
- go-judge pools containers across jobs.
- gVisor doesn't enforce per-process limits inside a sandbox, which breaks per-case MLE.

## Decision

### 1. Build a thin Go runner (`xlearn-runner`) — "R1"

**The pod:**
- One Deployment in a new `xlearn-runner` namespace on the production node.
- The pod runs in **its own user namespace** (`hostUsers:false`) under a containerd handler `judge` (`cgroup_writable`, systemd), installed as a **k3s drop-in**.
- Namespaced capabilities only.
- **Localhost AppArmor** (no `userns` rule) and **Localhost seccomp**, with the new mount API, bpf, perf, keyctl, io_uring and risky socket families removed.
- No SA token, no DNS, default-deny NetworkPolicy, lowest PriorityClass, ResourceQuota.
- A **ValidatingAdmissionPolicy** pins the pod's exact shape and denies `exec` and `attach`.

**Per test case:**
- A fresh jail **without a user namespace**: its own mnt, pid, net (loopback down), ipc, uts and cgroup namespaces.
- A fresh tmpfs root, a per-job UID, zero capabilities, `NO_NEW_PRIVS`.
- A **kill-by-default seccomp allowlist** per language.
- Its own **cgroup v2 leaf**: `memory.max` + `oom.group`, `pids.max`.

**Measurement and verdicts:**
- **All verdict evidence is read outside the learner's process:** `cpu.stat`, `memory.peak/events`, pidfd, pipe counters.
- Expected outputs never leave judge.

**Process split:**
- A capability-less **front** parses every learner-controlled byte.
- A privileged **spawner** never does.
- The front restarts every 500 jobs or 6 h, and on any SIGSYS or quarantine. Each Result carries a `BootEpoch` so its evaluations can be regraded.

**Jail primitive:** go-sandbox's low-level `forkexec.Runner` (MIT, vendored and pinned). Fallbacks, in order: **nsjail** (`--disable_clone_newuser`), R1-U, then R1b.

### 2. Languages at M3 (owner)

**Go, C++ and Python.** Each gets:
- a runner profile;
- `func-json@1` and `class-ops@1` harness codecs;
- an amd64 exec allowlist;
- a time limit from the Go reference × a per-language multiplier, calibrated at A8.

CI also runs a per-language reference for each fully packed item.

go-race and sql-pg arrive with the second-course pilot. SQL runs a **fresh Postgres per job inside the jail**, never CNPG.

### 3. Timing on a noisy 4-vCPU VM

- **Slots:** 2 equal slots of 1 CPU each.
- **Time limit:** CPU time, TL = max(3 × the reference on the production profile, 1 s).
- **Throttling signal:** `throttled` comes only from **steal** or a **runner-owned canary**, because S0 showed cgroup CPU time includes steal.
- **Quiet re-run:** a failing time verdict while the other slot was busy or steal was high gets one in-runner quiet re-run. If that still fails, the job is re-queued once. Only after that is it `inconclusive(runner_throttled)`.
- **Guarding Hostinger's CPU throttle:** per-account runner-second budgets (compile and CE included), a CE cache, and a duty-cycle breaker.

### 4. Where it runs (owner)

- On the **production node** while signup is invite-only (D13).
- **Move to a dedicated runner VPS on any trigger** (never the backup VPS):
  - open signup;
  - a reachable unpatched LPE left for more than 7 days;
  - the spike forcing R1b;
  - chronic steal.
- The runner is stateless, so a move is a network change only.

### 5. Host and cluster hardening

**Track A gates M3:**

| Step | What |
|---|---|
| A0 | **H0 host patch gate**: current noble kernel via unheld metas, `core_pattern=core`, `suid_dumpable=0`, apport off, module denylist. This protects v1 today. **Applied 2026-09-24; the reboot into 6.8.0-142 is scheduled for 2026-09-25.** |
| A1 | S0 production facts (done; 72 h steal sample running). |
| A2 | The **mechanism spike** (deferred; gate). |
| A3 | Chart 0.3.0 knobs, default-off, with a byte-identical diff gate. |
| A4 | **Ingress NetworkPolicies** for `databases` and `messaging`. |
| A5 | **NATS nkeys**, server-first with fine ACLs in the same step (amended by T7, §7; was "coarse"). |
| A6 | The host sandbox block: containerd drop-in, AppArmor and seccomp profiles, kubelet subuid range, sysctls (`io_uring_disabled=2`, `kptr_restrict=2`, …). |
| A7 | `sandbox-guards` via Flux: namespace, VAP, RuntimeClass, PriorityClass, quota, NetworkPolicies. |
| A8 | The runner deployed dark, behind an acceptance suite (isolation markers, network and syscall probes, INV-14 OOM placement, calibration). |

**Track B (non-gating):** PSA labels, tokens off on v1, xlearn egress rules, CNPG ≥ 18.6, k3s v1.36.5, Renovate. (Fine NATS ACLs moved into A5 by T7, §7.)

**How it's operated:**
- Host steps are manual but **scripted**: `hack/host-bootstrap.sh`, plus `host-verify.sh` run after every reboot or upgrade.
- The runner has its **own release stream** (a `runner-v*` tag and ImagePolicy) and its own Flux Kustomization **off v1's critical path**.

### 6. Patch cadence (owner)

- **Unattended security upgrades**, with the kernel meta-packages unheld. The Hostinger image held them; they were unheld on 2026-09-24.
- **No Canonical Livepatch** (owner). Kernel fixes land at the monthly reboot.
- A **monthly reboot** window.
- **Monthly k3s patch releases** pinned in git.
- **Renovate** for the runner Dockerfile and the go-sandbox pin.

### 7. Amended by T7 / ADR-0035 (2026-09-24)

- **A5 rolls out server-first, with fine ACLs in the same step.** T3 said "clients first" and "coarse". Neither holds:
  - nats.go refuses an nkey seed when the server sends no nonce, and the server sends one only once nkeys are configured, so clients can't go first;
  - a coarse `$JS.API.>` grant lets any service update or purge any stream.
- **The rollout:**
  - nkey users in `$G`, bridged by a `no_auth_user: legacy` user;
  - N0 client code ships dark → N1 users + bridge (the **one** NATS restart) → N2 seeds per service → N3 `legacy` denied → N4 bridge removed (a reload);
  - ACLs are rendered from `internal/platform/events/topology.go`, and a compose test runs NATS with the rendered block before N2;
  - **N3 plus the `databases`/`messaging` NetworkPolicies (A4) gate M3 and erase.**
- **Secrets:** **no SOPS decryption in `messaging`**. Public keys are plaintext in its values; seeds are SOPS secrets in `apps/secrets`; the ops seed stays offline, used only for break-glass through an `ssh` port-forward.
- **Kubelet reservation (L23):** `system-reserved` 1 GiB plus `eviction-hard memory.available<500Mi` join A6 in the **October host window**, in the same k3s restart. `host-bootstrap.sh` sets them and `host-verify` asserts them. **MI-11a** (Flux controller limits 1 GiB → 512 Mi; limits for Traefik, cert-manager and metrics-server) lands before A8, because the runner's 3 GiB of limits would otherwise break the node's memory sum.
- **Chart:** the live chart is **0.2.2**. A3's knobs ship as **0.3.0 in one PR** with the union of every topic's asks.

## Consequences

- ✅ The strongest isolation available on a KVM-less node, without a privileged pod. All measurement is outside learner processes, which makes INV-14 enforceable.
- ✅ The runner is stateless and off the critical path, so moving it to another host is cheap.
- ✅ H0 fixes live, reachable risk on v1 today.
- ⚠️ **Shared kernel.** A kernel 0-day via an allowlisted syscall means node root, and with no backups (D12) that is critical impact. It is accepted under invite-only, and patch cadence and move triggers bound it.
- ⚠️ **A compromised runner can forge verdicts and read other jobs' data** until it restarts. `BootEpoch` makes those evaluations regradeable.
- ⚠️ **Three languages at M3** means three profiles, three harnesses and three allowlists, per-language calibration, a bigger image, and about 35–50 h more authoring (inferred).
- ⚠️ **R1 is unproven until the spike runs.** nsjail is the mature Plan B.
- ⚠️ **Manual host state** (drop-in, profiles, subuid) must survive k3s upgrades. `host-verify` asserts it after each one.

## Alternatives considered

| Option | Why not |
|---|---|
| Judge0, Piston, DMOJ, go-judge as a service | Privileged containers, stored expected outputs, ptrace (breaks TSAN), or pooled containers |
| gVisor as the pod runtime | No per-process limits inside the sandbox, which breaks per-case MLE; weekly manual installs. Revisit for the execute step on open signup |
| Firecracker, Kata, gVisor-KVM | No `/dev/kvm` |
| bubblewrap, Sysbox, Landlock only | userns blocked, no cgroups, or no memory limit |
| A pod or Job per submission | 1–5 s start, and judge would need RBAC |
| Privileged pod | Any escape is host root |
| A dedicated runner VPS now | Strongest blast-radius cut, but the owner chose prod + triggers |
| Go only at launch | The owner chose Go + C++ + Python |
| Shared warm Postgres / PGlite / DuckDB / SQLite for SQL | Cross-account state and CVEs, or dialect drift |
