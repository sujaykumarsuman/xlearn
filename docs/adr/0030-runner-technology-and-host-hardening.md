# ADR-0030 — Runner technology (sandbox) & host/cluster hardening

- **Status:** Accepted (2026-09-30, spike P0–P3 GO; ADR-0035 §6 amendments folded). Accepted as task 1 of sprint [m3-03](../v2/sprints/sprint-m3-03.md) on the sandbox-spike GO ([spk-01](../v2/sprints/sprint-spk-01.md) P0–P2 and [spk-02](../v2/sprints/sprint-spk-02.md) P3 plus the image-volume spike, run on 2026-09-25 under D41; [t3 §16](../v2/research/t3-sandbox.md#16-spike-results-mi-10)), with no owner sign-off (D40). It had stayed Proposed at the v2 build-plan sign-off (2026-09-24, BP2) until then. **§5 was amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) §6 (T7, 2026-09-24); the amendments are folded into §5 below.**
- **Date:** 2026-09-24 (accepted 2026-09-30)
- **Deciders:** @sujaykumarsuman
- **Related:** [0029](0029-judge-contract-and-learning-signal.md) (the runner contract this satisfies),
  [0027](0027-content-evalpack-and-user-data-model.md) (leak and integrity controls), [0028](0028-object-storage-and-backups.md)
  (no backups, which raises the stakes of a node compromise), [0009](0009-deployment-and-gitops.md), [0014](0014-nats-jetstream-topology-and-outbox-relay.md).
  v2 topic T3: [feasibility § T3](../v2/feasibility.md#t3--code-execution-sandbox--cluster-hardening) and the full
  [research appendix](../v2/research/t3-sandbox.md), which includes the S0 results and the spike results. The runner as built:
  [`docs/architecture/runner.md`](../architecture/runner.md).

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
- **The kernel was 6.8.0-90 while noble is at 6.8.0-142.** The Hostinger image had put the kernel meta-packages on hold, so it never auto-updated. They were unheld and 6.8.0-142 was installed on 2026-09-24 (H0); the node rebooted into it on 2026-09-25 (MI-0).
- Core dumps were piped to apport as host root (fixed by H0).
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
- Namespaced capabilities only: `SYS_ADMIN SETUID SETGID SETPCAP KILL` (SETPCAP is required; see "Spike results").
- **Localhost AppArmor** (no `userns` rule) and **Localhost seccomp**, with the new mount API, bpf, perf, keyctl, io_uring and risky socket families removed.
- No SA token, no DNS, default-deny NetworkPolicy, lowest PriorityClass, ResourceQuota.
- A **ValidatingAdmissionPolicy** pins the pod's exact shape and denies `exec` and `attach`.

**Per test case:**
- A fresh jail **without a user namespace**: its own mnt, pid, net (loopback down), ipc, uts and cgroup namespaces.
- A fresh tmpfs root, a per-job UID, zero capabilities, an empty bounding set, `NO_NEW_PRIVS`.
- A **kill-by-default seccomp allowlist** per language.
- Its own **cgroup v2 leaf**: `memory.max` + `oom.group`, `pids.max`.

**Measurement and verdicts:**
- **All verdict evidence is read outside the learner's process:** `cpu.stat`, `memory.peak/events`, pidfd, pipe counters.
- Expected outputs never leave judge.

**Process split:**
- A capability-less **front** parses every learner-controlled byte.
- A privileged **spawner** never does.
- The front restarts every 500 jobs or 6 h, and on any SIGSYS or quarantine. Each Result carries a `BootEpoch` so its evaluations can be regraded.

**Jail primitive:** go-sandbox's low-level `forkexec.Runner` (MIT), **no user namespace**, spawned into the case cgroup with `CLONE_INTO_CGROUP` (spk-01 chose it; nsjail was not needed). It is **pinned in `go.mod` at v0.13.7 (`go.sum`-verified), import-restricted** to `pkg/forkexec`, `pkg/mount` and `pkg/rlimit`, used only by `internal/runner/...`, and `go-sandbox/container` is never imported (it forces a user namespace and keeps ambient `SYS_ADMIN`); `internal/runner/imports_test.go` guards all of it. The fallbacks nsjail (`--disable_clone_newuser`), R1-U and R1b were not needed.

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
  - chronic steal (TR-STEAL, ADR-0035 §5).
- The spike could have forced R1b (a fourth trigger); it didn't: R1 passed P0–P3 (2026-09-25), so that trigger is closed.
- The runner is stateless, so a move is a network change only.

### 5. Host and cluster hardening

**Track A gates M3:**

| Step | What |
|---|---|
| A0 | **H0 host patch gate**: current noble kernel via unheld metas, `core_pattern=core`, `suid_dumpable=0`, apport off, module denylist. This protects v1 today. **Applied 2026-09-24; the node rebooted into 6.8.0-142 on 2026-09-25 (MI-0 ✅).** |
| A1 | S0 production facts: **done** (2026-09-24). The steal sample (1,305 minutes before the H0 reboot cut it short: p50 3%, p95 8%, max 20%) set the throttling thresholds; sar is the ongoing source ([t3 §15](../v2/research/t3-sandbox.md#15-s0-results-read-only-production-facts-2026-09-24-owner-approved)). |
| A2 | The **mechanism spike**: **done, GO** (2026-09-25; P0–P3 and image volume; see "Spike results" below). |
| A3 | Chart 0.3.0 knobs, default-off, with a byte-identical diff gate. The live chart is **0.2.2**; A3's knobs (the union of every topic's asks) ship as **0.3.0 in one PR**. |
| A4 | **Ingress NetworkPolicies** for `databases` and `messaging`. They forward-declare identity, coach and judge as NATS callers and judge as a PG caller (MI-5); `:8222` admits no pod (opscheck dropped, D34). |
| A5 | **NATS nkeys, server-first with fine ACLs** rendered from `internal/platform/events/topology.go`: nkey users in `$G` bridged by `no_auth_user: legacy`; N0 client code dark → N1 users + bridge (the one NATS restart) → N2 seeds per service → N3 `legacy` denied → N4 bridge removed (a reload); a compose test runs NATS with the rendered block before N2. **No SOPS decryption in `messaging`**: public keys in plaintext in its values, seeds as SOPS secrets in `apps/secrets`, the ops seed offline (break-glass over an `ssh` port-forward only). **N3 plus A4 gate M3 and erase.** Why server-first and fine: [ADR-0035 §2](0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (nats.go refuses an nkey seed when the server sends no nonce, so clients can't go first; a coarse `$JS.API.>` grant lets any service update or purge any stream). |
| A6 | The host sandbox block: containerd drop-in, AppArmor and seccomp profiles, kubelet subuid range, sysctls (`io_uring_disabled=2`, `kptr_restrict=2`, …), **plus L23** (`system-reserved` 1 GiB, `eviction-hard memory.available<500Mi`) **and the pod pid limit, in the same k3s restart** (October host window); `host-bootstrap.sh` sets them, `host-verify` asserts them. The spike's host-file diffs are below. |
| A7 | `sandbox-guards` via Flux: namespace, VAP, RuntimeClass, PriorityClass, quota, NetworkPolicies. |
| A8 | The runner deployed dark, behind an acceptance suite (isolation markers, network and syscall probes, INV-14 OOM placement, calibration), **after MI-11a** (Flux controllers 1 GiB → 512 Mi; limits for Traefik, cert-manager and metrics-server), because the runner's 3 GiB of limits would otherwise break the node's memory sum ([ADR-0035 §5](0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)). |

**Track B (non-gating):** PSA labels, tokens off on v1, xlearn egress rules, CNPG ≥ 18.6, k3s v1.36.5, Renovate. (Fine NATS ACLs are in A5; the L23 kubelet args are in A6.)

**How it's operated:**
- Host steps are manual but **scripted**: `hack/host-bootstrap.sh`, plus `host-verify.sh` run after every reboot or upgrade.
- The runner has its **own release stream** (a `runner-v*` tag and ImagePolicy) and its own Flux Kustomization **off v1's critical path**.

### 6. Patch cadence (owner)

- **Unattended security upgrades**, with the kernel meta-packages unheld. The Hostinger image held them; they were unheld on 2026-09-24.
- **No Canonical Livepatch** (owner). Kernel fixes land at the monthly reboot.
- A **monthly reboot** window.
- **Monthly k3s patch releases** pinned in git.
- **Renovate** for the runner Dockerfile and the go-sandbox pin (`go.mod`).

### 7. History

§5 was amended by ADR-0035 §6 (T7, 2026-09-24) — A5 server-first with fine ACLs, A4's forward declarations, L23 into A6, MI-11a before A8, the chart line — and those amendments were folded into §5 at acceptance (2026-09-30).

## Spike results (MI-10, spk-01/spk-02)

From [t3 §16.1–§16.4](../v2/research/t3-sandbox.md#16-spike-results-mi-10) (spk-01 on arm64 multipass, 2026-09-25; spk-02 on an amd64 KVM guest with kernel 6.8.0-142 and `mmap_rnd_bits=32`, 2026-09-25). No timing conclusions: neither environment is the production guest.

| Phase | Check | Result |
|---|---|---|
| P0 | `judge` drop-in merged verbatim; `cgroup_writable`; pod survives a k3s restart; VAP dry-run corpus | ✅ 24/24, no `.tmpl` fallback |
| P1 (positive) | `hostUsers:false` pod, non-identity uid_map; `slots/` controllers; forkexec jail **without `CLONE_NEWUSER`** (tmpfs root, `pivot_root`, lo down); both spawn paths; caps → 0 with `NoNewPrivs`; `go build` / g++ / python3 in the jail; pod recreate × 50 | ✅ all; 50/50 |
| P1 (negative) | 17 probes from the supervisor (`unshare -U`, `clone3(NEWUSER)`, `fsopen`, `open_tree`, `mount` outside `/jail`, SCTP, NETLINK, PACKET, raw inet, `ptrace(1)`, `keyctl`, `add_key`, `userfaultfd`, `io_uring_setup`, `bpf`, `perf_event_open`, `setns`) + `move_mount`/`mount_setattr` + read-write remounts of `/`, `/sys`, `/jail/**` | ✅ all denied (after block 3's remount rules) |
| P1b | `go test -race` × 20 (RACE 20/20), deadlock × 20, postgres in the jail | ✅ |
| P2 | balloon MLE × 100; container-level OOM (`memory.events.local`) and pod restarts; survivors; `slots` baseline drift; `nr_dying_descendants` after 1,000 case cgroups | ✅ **100/100**; **0 / 0**; 0; +2.0 MiB; 50 → 0 in 6 s |
| P3 | TSAN under `mmap_rnd_bits=32`; ASLR policy; amd64 exec allowlists; unexpected SIGSYS under KILL | ✅ 125/125 + 100/100; **none needed**; `go` 27, `cpp` 19, `python` 39, `go-race` 40 names; **0** |
| Image volume | read-only pack mount with the kubelet defaults; a pod without the pull secret refused | ✅ GO (ADR-0027's image-volume line stands) |

**m3-03's `runner-it` lane reproduces P2 on amd64 in CI** (a privileged `debian:trixie-slim` container, private cgroup namespace, 2 CPUs / 3 GiB): MLE 100/100 with 0 container-level OOMs, 0 survivors, `nr_dying_descendants` settling within 60 s after 1,000 case cgroups; the numbers are in `docs/v2/status.md` (decisions log, m3-03).

- **Mechanism:** go-sandbox `forkexec.Runner`, no user namespace, spawned into the case cgroup via `CLONE_INTO_CGROUP` (§1). It reproduces on amd64 6.8.0-142.
- **`SETPCAP`: required** (this corrects spk-01's "unused"). go-sandbox's `forkexec` calls `prctl(PR_SET_SECUREBITS)` whenever it sets credentials (keep-caps across `setuid`) and again before `capset(0)` (`SECURE_NOROOT` locked), and the kernel requires `CAP_SETPCAP` for `PR_SET_SECUREBITS`. The runner also uses it to **empty the bounding set** (`PR_CAPBSET_DROP`, on a locked OS thread that is then discarded) before it spawns the front and every jail, so both run with an empty bounding set as well as 0 caps and `NO_NEW_PRIVS`. The five-cap set (and so the VAP's cap list and the HelmRelease values) is unchanged.
- **Spawn path:** `CLONE_INTO_CGROUP` (clone3 + a cgroup fd) by default; the `cgroup.procs` write gated on a sync pipe works on 6.8 too and stays behind `RUNNER_SPAWN_PATH=cgroup_procs`, with a `runner_it` test.
- **Read-only bind rule:** every read-only bind into a jail is `MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY` **without `MS_PRIVATE`** (the jail's mount namespace root is already rprivate), never go-sandbox's `WithBind(…, true)`: AppArmor 4.0 can't match a remount carrying a propagation flag (t3 §16.2 block 3). A `runner_it` test proves the binds refuse writes (EROFS); proving it under the final AppArmor profile (sha256 `1d70ccd0…5775`) is m3-15's VM rehearsal and mi-10. The one read-write bind, `/dev/null`, has go-sandbox's `WithBind(…, false)` shape, which the profile's `mount options=(rw, rbind, nosuid, rprivate) -> /jail/**` rule admits.
- **Where the allowlists live:** the pod seccomp profile is a host file ([mi-09](../v2/sprints/sprint-mi-09.md)); the per-profile exec and compile filters are built inside the runner from syscall-name lists (`internal/runner/seccomp`; the real profiles are [m3-04](../v2/sprints/sprint-m3-04.md)'s), with the fixed rules: a non-x86_64 arch or an x32 number → KILL, `clone3` → ENOSYS, `clone` only with `CLONE_THREAD` (and no namespace flag), `prctl` only with `PR_SET_VMA`, `personality` in no profile.
- **go-race ASLR policy: none.** The Go 1.26 race runtime works in the jail under `mmap_rnd_bits=32` without calling `personality`; the pod profile keeps RuntimeDefault's `personality` rule, so `ADDR_NO_RANDOMIZE` stays denied. Never lower the host sysctl ([p-01](../v2/sprints/sprint-p-01.md)).
- **`GOCACHE`:** a **read-only seed used in place** (`GOCACHE` = the seed, baked into the runner image or mounted as an image volume, bound read-only into the compile jail), with `TMPDIR` on the case tmpfs (`/w`) and `GOROOT` set in the compile env (the jail has no `/proc`). **No in-pod overlay:** overlayfs EACCESes under `hostUsers:false` on arm64 and amd64 (t3 §16.1, §16.2 block 1, §16.4). The seed is built with the exec profile's exact toolchain and flags (m3-04); p-01's race seed uses the same mechanism.
- **Host-file diffs against t3 §8.7** (mi-09's host-window PR ships them):
  - the pod seccomp profile adds **`pivot_root`** (RuntimeDefault omits it; without it the jail's `pivot_root` is EPERM) and is **x86_64-only** (`architectures: [SCMP_ARCH_X86_64]`, 381 names, sha256 `730a7a55…418d`). **Ratified here:** nothing legitimate broke on the x86_64-only profile (the jail setup, the references under KILL, go-race, postgres), and it closes the ia32 `int $0x80` entry point the three-arch baseline leaves open (t3 §16.2 block 2). runc's bad-arch action is `KILL_THREAD`; inside the jail the exec filter's `KILL_PROCESS` takes precedence;
  - the AppArmor profile (sha256 `1d70ccd0…5775`) replaces the broad `remount,` with two `ro`-only rules, `remount options=(ro, nosuid, noatime, bind) /,` (the jail root) and `remount options=(ro, nosuid, nodev, rbind) /jail/**,` (read-only binds), so no read-write remount is possible anywhere;
  - the `judge` drop-in is verbatim; the subuid range is `kubelet:1073741824:7208960` with `getsubids` from `uidmap`;
  - a new host check: **no node-level registry credentials** (no `auth:` in k3s `registries.yaml`, no `/var/lib/kubelet/config.json`, no root `~/.docker/config.json`; t3 §16.3).

## Consequences

- ✅ The strongest isolation available on a KVM-less node, without a privileged pod. All measurement is outside learner processes, which makes INV-14 enforceable.
- ✅ The runner is stateless and off the critical path, so moving it to another host is cheap.
- ✅ H0 fixes live, reachable risk on v1 today.
- ✅ **R1 is proven** on this stack by the spike (P0–P3 GO) and the runner's own jail suite; what a CI container can't show — the pod's user namespace, the Localhost AppArmor and seccomp profiles and the VAP on the real node — is proven by m3-15's VM rehearsal and mi-10's dark deploy.
- ⚠️ **Shared kernel.** A kernel 0-day via an allowlisted syscall means node root, and with no backups (D12) that is critical impact. It is accepted under invite-only, and patch cadence and move triggers bound it.
- ⚠️ **A compromised runner can forge verdicts and read other jobs' data** until it restarts. `BootEpoch` makes those evaluations regradeable.
- ⚠️ **Three languages at M3** means three profiles, three harnesses and three allowlists, per-language calibration, a bigger image, and about 35–50 h more authoring (inferred).
- ⚠️ **Manual host state** (drop-in, profiles, subuid) must survive k3s upgrades. `host-verify` asserts it after each one.

## Alternatives considered

| Option | Why not |
|---|---|
| Judge0, Piston, DMOJ, go-judge as a service | Privileged containers, stored expected outputs, ptrace (breaks TSAN), or pooled containers |
| gVisor as the pod runtime | No per-process limits inside the sandbox, which breaks per-case MLE; weekly manual installs. Revisit for the execute step on open signup |
| Firecracker, Kata, gVisor-KVM | No `/dev/kvm` |
| bubblewrap, Sysbox, Landlock only | userns blocked, no cgroups, or no memory limit |
| nsjail (R1-N) as the jail | Not needed: go-sandbox's low-level Runner passed P1 without a user namespace |
| go-sandbox's `container` builder | Forces a user namespace and keeps ambient `SYS_ADMIN` (t3 §4) |
| A pod or Job per submission | 1–5 s start, and judge would need RBAC |
| Privileged pod | Any escape is host root |
| A dedicated runner VPS now | Strongest blast-radius cut, but the owner chose prod + triggers |
| Go only at launch | The owner chose Go + C++ + Python |
| Shared warm Postgres / PGlite / DuckDB / SQLite for SQL | Cross-account state and CVEs, or dialect drift |
