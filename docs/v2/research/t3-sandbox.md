> **T3 research appendix.** Method: five parallel research slices (isolation tech on this host; OSS judges vs build; language images and the SQL sandbox; cluster hardening and threat model; performance and limits on 4 vCPU) were synthesized into one draft. The draft then faced two adversarial critiques: escape/red team (6/10, 1 blocker) and solo-ops, performance and GitOps (6/10). This is the revised final, plus the S0 results (§15). Decided in [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) (Proposed).
>
> **Status:** settled with the owner 2026-09-24 (**D20–D23**, §12). §12 overrides the body where they conflict, notably:
> - **Go + C++ + Python at M3** (not Go only);
> - the mechanism **spike (P0–P3) is deferred** to a build-plan gate; only S0 ran.

# T3: code-execution sandbox, online-judge engine and cluster-hardening prerequisites

**Scope.** Research and design only, dated 2026-09-24. No repo files were changed, the VPS and cluster were not touched, and nothing was installed.

**Inputs.** The T3 draft and two adversarial critiques. I re-checked the critiques' load-bearing claims:
- read-only against local copies of the k3s v1.36.4 source, the go-sandbox/go-judge source and containerd's seccomp source, plus the infra and xlearn repos;
- on the web, all seen 2026-09-24. Sources are in the appendix ([W1]–[W33]).

**(inferred)** marks my own estimates or reasoning.

---

## 1. Recommendation in one paragraph

**Build a thin Go runner; don't adopt an existing judge.** The service is `xlearn-runner`, about 3–4k lines plus an isolation test suite **(inferred)**.

**Jail primitive.** Use go-sandbox's low-level `forkexec.Runner` (MIT, v0.13.7). Do not use its `container` package: that package always creates a user namespace and keeps ambient `SYS_ADMIN`.

**Pod.**
- One Deployment in a new `xlearn-runner` namespace, on the existing node.
- A runc pod with its own user namespace (`hostUsers:false`).
- It runs under a containerd handler called `judge` (`cgroup_writable`, systemd cgroup driver), shipped as a k3s **drop-in** file, not a full template.
- Namespaced capabilities; a Localhost AppArmor profile with **no `userns` rule**; a Localhost seccomp profile that removes the new mount API, bpf/perf/keyctl/io_uring and risky socket families.
- A ValidatingAdmissionPolicy (VAP) pins that exact pod shape.

**Per case.** Each case gets:
- a **fresh jail with no user namespace**: its own mount, PID, network (loopback down), IPC, UTS and cgroup namespaces;
- a fresh tmpfs, a per-job UID, zero capabilities and `NO_NEW_PRIVS`;
- a seccomp filter that kills by default;
- its own cgroup v2 leaf.

All verdict evidence is read outside the learner's process. Expected outputs never leave judge.

**Privilege split.** A capability-free *front* process parses every learner-supplied byte. A privileged *spawner* never parses learner bytes and never touches a path learners can write to.

**Scheduling and timing.**
- Two equal slots of 1 CPU each, one job per slot.
- `throttled` comes only from steal or a runner-owned canary.
- A failing time verdict gets one **in-runner quiet re-run**.
- Per-account runner-second budgets plus a duty-cycle breaker keep the node below Hostinger's CPU throttle.

**Languages.** M3 ships `go` only. `go-race` or `sql-pg` arrives with the second-course pilot (D6). C++ comes after M5 (owner Q1).

**gVisor is out of v2.0.** It does not enforce per-process limits inside a sandbox [W1], which breaks the per-case memory-limit verdict (MLE) and INV-14.

**Before any runner exists, the node needs host gate H0, which also protects v1 today.** Today's 6.8.0-90 kernel is behind known privilege-escalation fixes [W21–W23]. H0 means:
- current noble kernel plus Livepatch;
- a module denylist;
- no piped core-dump helper.

**Hardening** is split into Track A, which gates M3, and Track B, which does not.

**Spike.** About 10 hours of core work, hard stop at 1.5 days, throwaway:
- an arm64 multipass VM for the mechanisms;
- an amd64 replay for the syscall allowlist and the race detector's (TSAN's) behaviour.

---

## 2. Threat model

### 2.1 Attacker
- **Primary:** an invited learner (D13), or someone who has stolen an account.
  - Submits Go now; `go test` packages and SQL with the pilot; C++ later.
  - Controls the source, compile-time inputs and all runtime behaviour.
  - Knows everything public: the image, harnesses, generators and reference solutions.
  - Is bounded by T4: 200 Runs, 100 counted and 100 arena submits a day; **one running runner job per account** (T4 §9.1 per-account claim serialization); ≥ 2 s between submits.
- **Secondary:** remote code execution in an internet-facing v1 pod (the gateway).
- **Trusted:** the owner, GitHub CI, GHCR and Flux. The host OS becomes trusted **only once H0 lands and the Q3 cadence holds**. Today it is a known gap: 6.8.0-90 predates the CrackArmor fix (6.8.0-106) and the Copy Fail fix (6.8.0-117), and noble is still listed as vulnerable to CVE-2026-74469 [W21–W23].
- **Out of scope:**
  - Hostinger and the hypervisor.
  - Micro-architectural *confidentiality* side channels between slots.
  - A determined cheat holding the public reference solution.
  - *Integrity* interference between slots is in scope (A16).

### 2.2 Assets (node root means all of them)
| Asset | Why it is critical |
|---|---|
| CNPG single instance, **no backups** (D12) | The whole learning record. Hostinger's weekly images are the only fallback, so D12 raises the impact of any node compromise. |
| `flux-system/sops-age`; Flux deploy key with write access to `infra` | Decrypts every secret; gives persistence that survives a rebuild. |
| kubescope cluster-admin service account | The full cluster. |
| JWT signing key, `COACH_MASTER_KEY`, OAuth secret, GHCR PAT | Session forgery; learners' LLM spend. |
| Eval pack (judge's image volume and the containerd store) | Integrity of checked grades. |
| Code and hidden inputs of in-flight jobs **and of every later job until the runner restarts** | Privacy and verdict integrity. |
| INV-14 (a learner never causes `inconclusive`) | Correctness of the learning signal. |
| Node availability and IP reputation | v1 outage; abuse reports. |

### 2.3 Trust boundaries
```
learner process
 ──B0 per-case jail: new mnt/pid/net(lo down)/ipc/uts/cgroup ns, NO userns, job UID, 0 caps, NNP,
     RLIMIT_CORE=0, KILL-default seccomp, own cgroup leaf, nosuid/nodev binds──▶
front (capless Go: HTTP, spool, ALL parsing of learner bytes, builds Result)
   ◀── fixed-schema socketpair + SCM_RIGHTS fds ──▶
spawner (PID 1; namespaced caps; never parses learner bytes; rule R-FS)
 ──B2 pod: hostUsers:false · AppArmor Localhost (no userns) · seccomp Localhost (trimmed) ·
     VAP-pinned shape · no SA token · no exec/attach──▶
node kernel (H0-patched, module denylist) + containerd/runc ──▶ everything in §2.2
B3 network: empty netns per jail (primary) + namespace default-deny + no DNS + bearer token on :8090
B4 judge: holds the pack and expected outputs, compares outside, validates every Result
B5 host helpers: core_pattern is a plain file (never a pipe); apport off
```
**Rule R-FS (extended):** once learner code has run, **no process holding any capability** opens, chowns, copies or traverses a path learners can write to. Only the capless front reads learner bytes, and only from pipes.

### 2.4 Attacks and the control that stops each one
| # | Attack | Stopped by (primary) | Backup layers | Residual |
|---|---|---|---|---|
| A0 | **Known, unpatched kernel privilege escalation (LPE)** reachable from pods: Copy Fail (AF_ALG), CrackArmor (bypasses the AppArmor userns restriction), DiagSpill (SCTP sock_diag, noble still vulnerable) [W21–W23] | **H0 gate:** current noble kernel (≥ 6.8.0-117), Livepatch, and `install <mod> /bin/false` for sctp, sctp_diag, dccp, tipc, rds, n_hdlc, af_alg/algif_*, pppoe, l2tp_*, can*, atm, ax25, x25, netrom, rose, rxrpc | Jail seccomp denies `socket()` except AF_UNIX (SQL only); runner seccomp denies AF_ALG/NETLINK/PACKET/VSOCK/SCTP; RuntimeDefault already blocks AF_ALG and AF_VSOCK [W33] | A new 0-day in the same class. See Q3. |
| A1 | Kernel LPE through allowlisted syscalls | Exec seccomp kills by default (~40–70 syscalls **(inferred)**). Denied: io_uring, bpf, perf, userfaultfd, keyctl, ptrace, `process_vm_*`, mount, unshare, setns, namespace flags on `clone`, splice/vmsplice, sockets. `clone3` returns ENOSYS. Zero capabilities, NNP, **no user-namespace creation anywhere in the pod**. | Sysctls: `io_uring_disabled=2`, `unprivileged_bpf_disabled=2`, `unprivileged_userfaultfd=0`; Q3 patch cadence | A 0-day in core syscalls (futex, mm). Critical impact; Q2 and Q3. |
| A2 | Escape through runtime misconfiguration or a runtime CVE | The VAP (§8.2) pins: no privileged mode, host namespaces, hostPath, hostPort, sysctls, exec probes/hooks or Unconfined seccomp; `hostUsers:false`; the digest-pinned image; **CONNECT on pods/exec and pods/attach denied** | judge has zero Kubernetes RBAC. runc 1.4.3 fixed CVE-2026-41579 (malicious image; low risk here because the image is ours) [W24]. containerd 2.3.5 fixes CVE-2026-53495 (exec-probe DoS; the VAP forbids exec probes) [W25]. Track k3s v1.36.5 (rc1 out 2026-09-24) [W31]. | runc or containerd 0-day |
| A3 | Lateral network access (NATS without auth, PG :5432, `/internal/*`, kube API, kubelet, metadata, DNS exfiltration) | **Empty network namespace per jail, loopback down** | Runner namespace default-deny; `dnsPolicy: None`; bearer token; runner seccomp socket filter; NATS nkeys; PG/NATS ingress lists | Node-local traffic bypasses NetworkPolicy; this affects the front only (A11) |
| A4 | Reading expected outputs or the pack | Never sent: judge compares | Run materialises public samples only | Honor-grade profiles (T1 accepts this) |
| A5 | Compile-time exfiltration or code execution (`//go:embed`, cgo, `.s`/`.syso`, `//line`, compiler CVEs) | **Lint:** only judge-named `*.go` files, with no separators, `..` or leading dots. Rejects `.s .S .c .h .cc .syso .swig .swigcxx .m .f`, `go.mod`, `go.work`, `vendor/`, `import "C"`, `//go:embed`, `//line`, `/*line*/`, `//export`, and every `//go:` directive except `//go:build`. The runner re-checks filenames. | Compile jail: ENOSYS by default, KILL for dangerous calls, **no `socket()`**, limits; toolchain patched through its own release stream (§6.1) | About 1 bit per compile, bounded by budget |
| A6 | Leakage or persistence across jobs and accounts (files, shm, mqueue, abstract sockets, keyrings, daemons, caches) | **Fresh per-case namespaces and tmpfs.** The job's artifact tmpfs is written only by the spawner, before any learner code runs. `cgroup.kill`; keyring calls denied; UID pool not reused within 1,000 jobs; caches read-only and built from public content. | T1's cross-account marker test is a release gate (A8 in §8.8) | — |
| A7 | Forging a verdict from inside the jail | Evidence comes from cgroup files, pidfd and pipe counters. Harness frames travel on fd 4 and are compared **in judge**. cgroupfs is not mounted in the jail. | judge validates the Result against the Job | Honor profiles |
| A8 | Denial of service inside a job | Per-case `pids.max`, `memory.max` + `oom.group`, tmpfs size and inode caps, `RLIMIT_FSIZE`, output caps, CPU time-limit kill, idle kill | Pod capped at 2 CPU / 3 GiB; lowest priority | — |
| A9 | Learner-induced `inconclusive` (INV-14) | Memory arithmetic (§7.1), **including SQL backends moved into case cgroups**. `throttled` = steal or canary only. A wall-clock kill with CPU below the limit triggers a quiet re-run, and inside the quiet re-run it becomes TLE. | Poison-pill rules (§5.9) | A runner bug a learner can trigger: quarantined and alerted |
| A10 | Compromising the SQL server (PG 18.6 fixed SQL-reachable code-execution bugs [W18]) | Postgres **runs per job inside the jail** as the job UID, under exec seccomp, with no network; it dies with the job | SELECT-only role, read-only transaction, extended protocol | The learner can read its own job's hidden rows; Submit output is verdict-only |
| A11 | Compromising the supervisor through hostile output | **Privilege split:** all parsing happens in the capless front; the spawner never parses and has no long-lived in-jail init. Memory-safe Go with capped, strict decoders. | Runner seccomp strips the new mount API [W33], so AppArmor's fstype mount rules can't be bypassed through `fsmount`; no userns creation | A front compromise can forge Results; see the residuals |
| A12 | Poison-pill submission | Only an **unexplained** supervisor death, a container OOM or `job_timeout` counts: ≤ 1 retry, then quarantine if the same submission kills the runner twice (§5.9) | Lane suspension after 2 quarantines a day | — |
| A13 | Supply chain | Digest-pinned base image, checksummed Go tarball, pinned PGDG version, `go.sum`; setuid bits stripped; reproducible build; Renovate (Q3) | Image digest recorded per evaluation | — |
| A14 | Compromised v1 pod pivots | Track A ingress NetworkPolicies for `databases` and `messaging`; NATS nkeys | Track B: egress rules, fine-grained ACLs, per-role `connectionLimit` | — |
| A15 | **Core-dump helper as a host-root deputy.** A SIGSYS or crash in a jail runs apport as host root on a core image the learner controls [critic; CVE-2025-5054] | `kernel.core_pattern=core` (never a pipe), `fs.suid_dumpable=0`, apport disabled, systemd-coredump masked; jails set `RLIMIT_CORE=0` | host-verify asserts it; readyz reads `core_pattern` | — |
| A16 | **Cross-slot integrity attack.** A memory-bandwidth hog in one slot pushes a near-limit victim in the other slot into TLE | **Any failing time verdict while the other slot was busy gets an automatic quiet re-run**; the canary includes a memory-bound kernel | — | A pass that was slowed down stays a pass (no harm) |
| A17 | **Saturation leading to Hostinger CPU throttling**, which shows up as steal and mass `inconclusive`; free compile bombs | Per-account runner-second budgets **that include compile and CE**; CE cache; duty-cycle breaker (§7.4); go-race Runs use `-count=1` and a 20 s deadline | Runner hard cap of 2 vCPU (50% of the VPS) | Hostinger doesn't publish its trigger [W28]; S0 measures steal |
| A18 | **An uncounted oracle over hidden data** via a runtime REJECTED | Only rejections decided **before learner code runs** (lint, SQL static classification) are uncounted. SIGSYS at runtime and SQLSTATE 25006 map to **RE, counted**, plus an ops alert. | — | — |
| A19 | go-race dump or `test2json` leaks or forgeries | Dumps are stripped of arguments and PC offsets; `TestEvent.Output` is dropped in Submit; results come from a harness-owned `TestMain` on fd 4; a timeout or non-zero exit fails, whatever the pass events say | Profile stays honor-grade | Honor |

**Accepted residuals (owner confirms through Q2 and Q3):**
1. A shared-kernel 0-day means node root (critical impact, low likelihood under invite-only). D12 makes this worse.
2. **A runner compromise means verdict forgery and loss of privacy for *all* jobs until the container restarts.** Reference solutions are public, so a compromised runner can compute correct answers. This is bounded as follows:
   - the container restarts every 500 jobs or 6 h, and immediately after any SIGSYS or quarantine;
   - `Result.Versions.BootEpoch` lets evaluations from a suspect epoch be found and regraded.
3. Honor profiles expose hidden tests to the runner.
4. A side channel of ≤ 5.4 bits per arena submit (T4).
5. A learner can force one quiet re-run of their own job. That at most doubles the job's cost, which is charged to the learner's budget.

---

## 3. Isolation options on this host

Host: k3s 1.36.4, containerd 2.3.4, runc 1.4.2, kernel 6.8 (patched to the current noble kernel under H0), Ubuntu's AppArmor userns restriction on, no `/dev/kvm`.

| Option | Works without KVM? | Works with the AppArmor userns restriction? | Host changes (GitOps vs manual) | Isolation strength | Per-job overhead | Verdict |
|---|---|---|---|---|---|---|
| **R1:** `judge` handler (`cgroup_writable`, systemd) + `hostUsers:false` + namespaced caps + per-case jail without a user namespace, via go-sandbox `forkexec.Runner` | ✅ | ✅ Nothing unprivileged creates a user namespace: runc creates the pod's as host root and the jail creates none. The profile has no `userns` rule; P1 asserts that `unshare -U` and `clone3(CLONE_NEWUSER)` both fail. | **Manual, scripted (H0 + H8):** containerd drop-in, AppArmor profile, Localhost seccomp file, subuid range, sysctls, core/modprobe settings. **GitOps:** namespace, VAP, RuntimeClass, NetworkPolicy, quota, PriorityClass. | Host kernel behind a kill-by-default per-case allowlist; 0 caps; pod user namespace; exact per-case cgroups; supervisor behind a trimmed seccomp profile | ~2–5 ms per case jail, ~10–30 ms per job **(inferred)** | **ADOPT** (spike P1 gate) |
| R1-N: R1 with nsjail `--disable_clone_newuser` spawned into our case cgroup | ✅ | ✅ | + the nsjail binary in the image | ≈ R1; accounting stays ours | < 20 ms | **Plan B** (mature, Google-maintained) |
| R1-U: the jail creates a nested user namespace (go-sandbox `container` package) | ✅ | ⚠️ needs an AppArmor `userns,` rule | same as R1 | Same surface for the learner, but a compromised supervisor reaches userns-gated kernel code (nf_tables) | same | Fallback, only if no path without a user namespace works |
| R1b: R1 with `hostUsers:true` | ✅ | ✅ | same as R1 | Jail unchanged; a compromised supervisor is close to host root | same | Last resort; switches Q2 to (b) |
| gVisor `runsc` (systrap) as the pod runtime | ✅ [W2] | ✅ | Manual install, weekly releases | Strongest without KVM | 50–400 ms per sandbox | **REJECT for v2.0.** No per-process limits inside the sandbox [W1] breaks per-case MLE and INV-14. |
| R1 + `runsc` for the execute step only | ✅ | ✅ | the gVisor install on top of R1 | Strongest for learner code | +50–400 ms; needs a watchdog for per-case memory | **DEFER to v2.x.** Trigger: open signup or a kernel-CVE burden. |
| isolate 2.7 (setuid, GPL-2+) inside R1 | ✅ | ✅ | none beyond R1 | ≈ R1 | ms | Plan C |
| bubblewrap | ✅ | ❌ | a profile | No cgroups | ms | Reject |
| Sysbox | ✅ | ✅ | a source build plus daemons | Not a per-case sandbox | — | Reject |
| Pod or Job per submission | ✅ | ✅ | none | Freshest state | 1–5 s start; judge would need RBAC | Reject |
| Privileged pod (the Judge0/Piston shape) | ✅ | ✅ | none | Host root on any escape [W15] | — | **Never** |
| Landlock only | ✅ | ✅ | none | No fresh filesystem view, no memory limit | ~0 | Optional inner layer only |
| Firecracker, Kata, gVisor-KVM | ❌ | — | — | — | — | Impossible |
| A dedicated runner VPS, with R1 there | ✅ | ✅ | second host bootstrap plus WireGuard/mTLS | Blast radius limited to that VPS | + network round-trip | Owner Q2(b); **never the backup VPS** |

**Why R1 and not gVisor:**
- **Per-case contract.** gVisor enforces limits only at the sandbox level [W1]. That means either a sandbox per case (40 × 50–400 ms) or a memory balloon that becomes an infra error.
- **Signals.** Steal and cgroup signals are synthetic or missing inside gVisor.
- **Ops.** Weekly manual installs on a node managed through GitOps.

**Where R1's risk lives now.** containerd #12182 is not a blocker: it works with the systemd driver [W3][W4]. k3s v1.36.4 picks the systemd driver when it is **not in a user namespace, the `cpuset` controller is present and `INVOCATION_ID` is set** (`config_linux.go` L71 [W32]). This host has cpuset and runs k3s as a systemd service, so the condition should hold **(inferred)**; S0 confirms it.

The remaining unknowns are all P1 gates:
- whether go-sandbox's `forkexec.Runner` works without a user namespace;
- AppArmor behaviour when the process holds `SYS_ADMIN`;
- whether containerd merges the drop-in.

---

## 4. Adopt vs build

**Legend.** "Split": compile runs separately and its artifact is reused. "Outside": per-case evidence is measured outside the learner's process. "Pack stays": expected outputs stay in judge and inputs are streamed. "Fresh": a new sandbox per job. "Node": runs here without privilege or KVM, under the AppArmor restriction. "Thr/Dump": provides the `throttled` flag and the goroutine dump.

| Candidate | Licence, status (seen 2026-09-24) | Split | Outside | Pack stays | Fresh | Node | Thr/Dump | Verdict |
|---|---|---|---|---|---|---|---|---|
| Judge0 | GPL-3.0; v1.13.1, 2024-04-18 [W15] | ✗ | ✓ | ✗ stores `expected_output` | ✓ | ✗ privileged; cgroup v1 | ✗ | Reject |
| Piston | MIT; no release since 2021 | ✗ | partial | ✗ | ✓ | ✗ privileged | ✗ | Reject |
| DMOJ judge-server | AGPL-3.0; v5.0.0, 2026-06-14 [W16] | ✓ | ptrace + `RLIMIT_AS` (breaks TSAN) | ✗ | ~ | `SYS_PTRACE` | ✗ | Reject |
| go-judge as a service | MIT; v1.12.3, 2026-08-31 [W11] | ✓ | ✓ | wrapper needed | **✗ pooled containers are only `Reset()`; IPC namespace, network namespace and init survive** | its docs use `--privileged` | ✗ | Borrow patterns only |
| **go-sandbox `forkexec.Runner`** (library) | MIT; v0.13.7, pushed 2026-08-30 [W12] | ours | ours | ours | ✓ per case | ✓ in R1. **Only the low-level Runner:** the `container` builder forces `CLONE_NEWUSER` and UID maps and keeps ambient `SYS_ADMIN`/`SYS_RESOURCE` in its init (verified in source). `CgroupFD` passes through (go-judge's call site). | ours | **ADOPT as the primitive** (vendored, pinned) |
| nsjail | Apache-2.0; 3.6, 2026-03-18 [W14] | ✓ | ✗ (we add cgroups) | ✓ | ✓ | ✓ with `--disable_clone_newuser` | ✗ | **Plan B** |
| isolate | GPL-2+; 2.7, 2026-08-24 [W13] | ✓ | ✓ | ✓ | ✓ | "not recommended" in containers | ✗ | Plan C |
| DOMjudge, CMS | GPL-2 / AGPL-3 | ✓ | ✓ | whole contest system | ✓ | host root | ✗ | Reference only |
| agent-sandbox, e2b, microsandbox | Apache-2.0 | — | — | — | pod or VM | KVM or pod-per-session | ✗ | Reject |
| Hosted (E2B, Modal, Cloudflare) | — | — | — | ✗ inputs leave the platform | ✓ | needs egress | ✗ | Reject |

**Why build, and what we own and borrow:**
- **Why build.** Every service candidate would need a wrapper as large as the runner itself to meet the contract: streaming, typed infra errors, `throttled`, the dump, verdict terms. The only close candidate, go-judge, reuses containers across jobs as shipped.
- **We own:**
  - the job and case lifecycle (created and destroyed, never pooled);
  - the privilege split;
  - cgroups and measurement;
  - profiles and harnesses;
  - `throttled` and the quiet re-run;
  - the dump;
  - typed errors;
  - rule R-FS.
- **We borrow:** go-sandbox's `pkg/forkexec`, its seccomp building and its mount setup — the syscall sequences between clone and exec that are hard to get right. **No fork is needed:** the user-namespace coupling sits in the `container` builder, which we don't use.
- **Fallbacks, in order:** nsjail (R1-N), then R1-U, then R1b. The learner-facing surface is the same for the first two.
- **This is not an owner decision.**

---

## 5. Runner architecture

### 5.1 Placement
```
xlearn ns                                    xlearn-runner ns (PSA privileged + VAP; default-deny; no DNS; no token; no exec)
┌─────────────┐  HTTP :8090 + bearer token   ┌───────────────────────────────────────────────────────────────────┐
│ judge       │ ───────────────────────────▶ │ runner pod (1 replica, Recreate, RuntimeClass xlearn-judge,       │
│ SKIP LOCKED │   job header+files+inputs    │  hostUsers:false, UID 0 in userns, caps SYS_ADMIN/SETUID/SETGID/   │
│ 2 workers   │ ◀─────────────────────────── │  SETPCAP/KILL, AppArmor+seccomp Localhost, RO rootfs, 2 CPU/3 GiB) │
│ pack volume │   Result JSON (sync)         │  spawner(PID1) ⇄ front ── cgroups: runner/ · slots/s0 · slots/s1   │
└─────────────┘                              └───────────────────────────────────────────────────────────────────┘
```
The runner never initiates a connection and never shares judge's pod or namespaces (T1 §10).

### 5.2 Process model (privilege split)
| Process | Privileges | Does | Never does |
|---|---|---|---|
| **spawner** (PID 1) | Namespaced caps; no listener | Creates cgroups, mounts, per-case jails and per-job artifact tmpfs; receives the compiled artifact as raw bytes; reads cgroup evidence and wait status; `cgroup.kill` | Parse learner bytes (it only reads fixed-schema requests from the front); open learner-writable paths after learner code ran |
| **front** | `capset(0)`, bounding set cleared, NNP, right after spawning | HTTP, auth, spooling inputs, all decoding of fd 1/2/4, `test2json` and dumps; assembles the Result | Hold capabilities; touch cgroupfs |

The front receives each case's pipe fds from the spawner via `SCM_RIGHTS`.

**Restart policy.** The front drains and exits every 500 jobs or 6 h, and at once after any SIGSYS or quarantine. Kubelet then restarts the container (~2–5 s, image cached). `BootEpoch` (pod UID + container start time) goes into every Result.

**Ops rule:** never exec into the runner; delete the pod instead. The VAP denies exec and attach.

### 5.3 API (judge is the only caller)
- **`POST /v1/jobs`** (HTTP/1.1, ClusterIP, `Authorization: Bearer`, token from SOPS).
  - The body is a length-prefixed stream: `job.json`, then the case inputs, ≤ 8 MiB per case and ≤ 16 MiB per job, enforced as bytes arrive.
  - `job.json` holds: id, `profile@v`, `harness@v`, mode, limits, `StopGroupOn`, `Count`, `Tests[]`, `OutputMode` (bytes or sha256), and files with `hidden` flags.
- **Responses:**
  - `200`: the Result;
  - `503`: both slots busy, draining, or a quiet re-run holds exclusivity;
  - `400`: contract error.
  - A 503 is never a verdict. judge re-queues with `run_after`.
- **Cancellation:** if judge closes the connection, the job subtree is killed with `cgroup.kill` and no Result is sent.
- **`GET /v1/profiles`** returns `{profile@v, toolchain, profile_sha256, image_digest, harness majors, baseline@v, boot_epoch}`.
- **`/readyz`** re-checks these canaries every 60 s:
  - egress fails (kube-dns, apiserver VIP, `1.1.1.1`);
  - the AppArmor label is `xlearn-runner`;
  - the UID map is not the identity map;
  - cgroupfs is writable;
  - `unshare(CLONE_NEWUSER)` is denied;
  - `fsopen("tmpfs")` is denied;
  - `socket(AF_INET, …, IPPROTO_SCTP)` is denied;
  - `/proc/sys/kernel/core_pattern` does not start with `|`.
- **judge validates each Result:** case ids and counts, enums, size caps, and profile/boot epoch. A mismatch is `infra: setup`.
- Token-aware claiming and `Job.Cost` are **dropped** (§14).

### 5.4 Lifecycle of one job
1. **Admit.** Take a free slot, or return 503.
2. **Spool.** Inputs go into front memory, charged to the `runner/` cgroup.
3. **Lint.** judge runs it (T4 §5.6 #15 with the §2.4 A5 rules); the front re-checks names and types. A violation is REJECTED, **uncounted, and only ever before any learner code runs**.
4. **Compile jail** (`slots/sN/job/compile`):
   - new namespaces, **no user namespace**, loopback down;
   - tmpfs root; read-only binds of the toolchain and `GOMODCACHE`, all `nosuid,nodev`;
   - overlay with the read-only `GOCACHE` seed as the lower layer and tmpfs as the upper;
   - `/job/src` read-only;
   - compile UID, 0 caps, NNP, rlimits;
   - **compile seccomp:** ENOSYS by default (logged), KILL for dangerous calls, no `socket()`;
   - runs `go build -json …`;
   - an in-jail exporter running as the compile UID opens the artifact with `O_NOFOLLOW` and streams it over a pipe (≤ 64 MiB);
   - the spawner writes those raw bytes into a new per-job **artifact tmpfs**, mode `0111`, owned by pod root;
   - the jail is destroyed.
   - A compile that hits its limits is **CE**, and its CPU counts toward the learner's budget.
5. **Case loop.** Samples first (stop on the first failure), then **all** correctness cases, then the perf block (stop at the first TLE). For each case:
   - `mkdir case-K`; set `memory.max`, `memory.oom.group=1`, `memory.swap.max=0`, `pids.max`;
   - fork into the cgroup. Prefer `CLONE_INTO_CGROUP`; the fallback is a `cgroup.procs` write gated on a sync pipe (go-judge enables clone3 only on kernel ≥ 6.9; P1 tests both on 6.8);
   - fresh namespaces; a fresh tmpfs root with the artifact bound read-only and a per-case writable `/w`;
   - job UID, 0 caps, NNP, `RLIMIT_CORE=0`, exec seccomp (KILL by default);
   - fd 3 is the input pipe; fd 4 the result pipe; fds 1 and 2 are capped pipes;
   - poll every 10 ms;
   - read the evidence, `cgroup.kill`, `rmdir`.
6. **Teardown.**
   - Kill the job subtree and `rmdir`; unmount the artifact tmpfs.
   - Assert that the slot's `memory.current` is back to baseline ±5 MiB, `pids.current` is 0, and `nr_dying_descendants` is falling.
   - Free the slot and respond.
7. **Startup.** Wipe the slot cgroups; move the spawner and front into `runner/`; pre-read the toolchain files so their page cache is charged to `runner/`.

### 5.5 Measurement (all outside the learner's process)
| Field | Source | Notes |
|---|---|---|
| `CPUms` (the verdict clock) | Δ `cpu.stat usage_usec` of the case cgroup | Cross-checked against `wait4`. S0 checks whether steal is excluded (`PARAVIRT_TIME_ACCOUNTING`). |
| `WallMs` | Monotonic clock, pidfd | — |
| `PeakKB` | Case `memory.peak` minus the calibrated baseline | A fresh cgroup per case |
| MLE | `memory.events oom_kill > 0` | For SQL, the backend's case cgroup (§6.3) |
| Fork cap | `pids.events max` | RE (fork limit) |
| OLE | Front byte counters on fds 1, 2 and 4 | — |
| Exit or signal | pidfd | **SIGSYS → RE (counted)** plus an ops alert; `BootEpoch` rotated |
| Harness result | Length-prefixed frames on fd 4 | Bytes, or `{sha256, bytes}` |
| Telemetry, internal | Steal over the window, canary ratio, busy ms, whether the other slot was busy | **judge persists it per job** (§10) |

### 5.6 Streaming inputs
- judge generates perf inputs from seeded specs onto its `emptyDir` and streams them.
- The front checks the caps as bytes arrive.
- A crashing case does not stop later correctness cases.

### 5.7 `throttled` and the quiet re-run (learner-proof, bounded)

**Signals.**
- **s1:** node steal fraction from `/proc/stat` over the case window.
- **s2:** a runner-owned canary: ~150 ms of mixed ALU work plus a memory stream whose working set exceeds the LLC. It runs **every 5 min when idle** (to keep a rolling median) and **on demand** after a failing time verdict, when no process from that job is alive.

s3 (`nr_throttled`) and s4 (PSI) are dropped.

**When a time verdict is suspect.** A failing CPU-TLE, wall kill or go-race deadline is suspect if any of these holds:
- (a) **the other slot was busy during the case**;
- (b) s1 ≥ 0.10;
- (c) s2 ≥ 1.2 × median.

A wall kill with CPU below the limit and a CPU rate above 5% over the last second is also suspect. If the rate was below 5%, it is **TLE (idle)**.

**The in-runner quiet re-run** is the same synchronous call:
- Stop admitting (503) and wait **≤ 60 s** for the other slot to drain.
- Require steal < 5% over 10 s and a pre-canary ≤ 1.1× median.
- Re-run the failing case and the rest of its group. **The quiet verdict is final;** a wall kill in quiet mode is TLE.
- If quiet conditions can't be met within 60 s, or s1/s2 fire during the re-run, set `Result.Throttled=true`. judge then re-queues **once** with `run_after = 5 min`. If the result is still throttled, it is `inconclusive(runner_throttled)`.
- Total delay is bounded at about 10 min, well inside D15's 45-minute limit. Queue latency never penalises the learner (`submitted_at`).

**Why a learner can't produce it.** A learner can't produce steal, and the canary runs when the learner's processes are dead. Another learner's hog only triggers a quiet re-run, which runs with the other slot empty. Driving the host into Hostinger's throttle is what the duty-cycle breaker prevents (§7.4).

### 5.8 Goroutine dump (go-race)
- The per-test `-test.timeout` is below the external deadline. Backstop: SIGQUIT (`GOTRACEBACK=all`), 500 ms, then `cgroup.kill`.
- The front parses goroutine headers and frames. A goroutine blocked in a **learner-package** frame gives DEADLOCK; otherwise TLE.
- The Result carries learner-package **function names and file:line only**: no argument words, no PC offsets.
- In Submit, `TestEvent.Output` is dropped. Results come from a harness-owned `TestMain` on fd 4. A timeout or non-zero exit is a failure whatever the pass lines say.

### 5.9 Typed infra errors and poison pills
| Kind | Evidence | Retry policy |
|---|---|---|
| `setup` | A spawner syscall failed before learner code ran, or a Result mismatch | ≤ 1 retry |
| `runner_oom` | Container-level `oom_kill`, or an unexplained restart | Counts toward poison pill |
| `killed` | SIGTERM, drain or rollout (the front answers 503 for jobs it can't finish within the grace period) | **Re-queued as `saturated`; spends no retry and no poison count** |
| `job_timeout` | Supervisor deadline passed (Σ case walls + compile + quiet re-run + slack) while per-case budgets held | Counts toward poison pill |
| `saturated` | 503 at admission, or connection refused during a known rollout | Re-queue with `run_after` |

**Poison pill.** Only `runner_oom`, an unexplained supervisor death, or `job_timeout` on the **same** submission counts:
- ≤ 1 retry after the restart;
- if the second attempt also kills the runner: `inconclusive(runner_unavailable)`, quarantine, an ops alert, and a `BootEpoch` rotation;
- lane suspension after 2 quarantines per account per day.

### 5.10 Cleanup invariants (release gate A8)
After 100 hostile jobs and 1,000 case cgroups:
- no leftover cgroups, mounts or processes;
- `nr_dying_descendants` settles near 0;
- `runner/` memory is back to baseline ±5 MiB;
- markers written by job N in `/w`, `/tmp`, `/dev/shm`, SysV shm, POSIX mq, an abstract socket and a keyring are invisible to job N+1 under another account;
- double-forked daemons are gone;
- a SIGSYS or SIGSEGV in a jail spawns no host helper.

### 5.11 T4 §11.1 checklist
| Requirement | How it is met |
|---|---|
| Compile separate from execution; artifact reused | Compile jail → pipe → per-job artifact tmpfs; hidden files exist only in compile; one compile per job |
| Per-case terminal state measured outside the process | Per-case cgroup + pidfd + front counters, mapped to `ok\|tle\|mle\|ole\|signal\|exit_nonzero\|not_run` |
| `CPUms`/`WallMs`/`PeakKB` | §5.5 |
| Expected outputs never enter (honor excepted) | judge compares; `OutputMode: sha256` for large outputs |
| Harness channel separate; stdout/stderr in Run only; caps | fd 4; fds 1/2 capped (8 KiB / 2 KiB kept in Run; hard cap 1 MiB → OLE) |
| Positioned diagnostics | `go build -json`, filtered to learner files |
| Typed infra errors | §5.9 |
| `throttled` | §5.7 (semantics amended: "no clean quiet re-run was possible") |
| Goroutine dump | §5.8 |
| Synchronous call honouring ctx cancellation | Connection close → `cgroup.kill` |
| ≤ 2 concurrent jobs; 429/503 when saturated | 2 equal slots; 503 |
| Fresh tmpfs per job | Fresh per case (stronger) |
| No network | Empty netns per case with loopback down, plus default-deny and no DNS |
| Streamed input ≤ 8 MiB per case, ≤ 16 MiB per job | Enforced at receive |
| Deterministic environment and locale | `env -i` + a fixed env hashed into `profile_sha256` |
| TL baseline, 3× margin | `baseline@v` keyed to `profile_sha256` (§7.3) |
| No KVM; AppArmor userns restriction | R1 |
| Run p50 < 5 s | ~1.5–2.5 s warm **(inferred; measured at A8)** |
| `Versions` | Runner, ImageDigest, Profile + Toolchain, ProfileSHA, CPUModel, CanaryMedian, **BootEpoch** (amendment) |
| T1 §10 (not in judge's pod or namespace; read-only public caches; cross-account test) | Separate namespace; seeds built from public content; §5.10 gate |

---

## 6. Language profiles

### 6.1 Image and release
- **One public image, `ghcr.io/sujaykumarsuman/xlearn-runner`**, with no secrets and no pack.
- **Base:** `debian:trixie-slim@sha256:…` (glibc). **All setuid and setgid bits are stripped.**
- **At M3 it contains:**
  - the supervisor binaries;
  - **go1.26.8**, from a checksummed tarball, pruned;
  - the `GOCACHE` seed. Its cache files carry a **fixed future mtime** and each job gets a fresh `trim.txt`, so they are never copied up or trimmed;
  - profile JSON files.
- **The pilot adds** `gcc`/`libc6-dev` (go-race) or PostgreSQL 18.x ≥ 18.6 from PGDG (sql-pg). C++ adds g++ and a precompiled header after M5.
- **Size:** about 200–250 MB compressed at M3 **(inferred)**.
- **Own release stream.** The runner is **not** rebuilt on every fleet `v*` tag (deploy.yml does that for all images today). It gets:
  - a `runner-v*` tag and a path-filtered workflow;
  - its own ImagePolicy;
  - a reproducible build (`SOURCE_DATE_EPOCH`, fixed mtimes).
- **Calibration keys on `profile_sha256`,** the content hash of toolchain, flags, harnesses, seccomp profiles and seeds, **not on the image digest**.
- **Patch bumps.** A Go 1.26.x or PG 18.x bump runs the speed index at startup:
  - within ±5% of the previous baseline, TLs carry forward;
  - otherwise TLs scale **up** by the ratio and are flagged.
- **Full recalibration** happens only on a minor-version bump or measured drift.
- **Architecture:** built with `TARGETARCH`, so the arm64 spike uses the same Dockerfile.

### 6.2 Profile table (limits are proposed; A8 tunes them)
| | **`go@1.26` (M3)** | `go-race@1.26` (pilot, honor) | `sql-pg@18` (pilot) | `cpp` (after M5) |
|---|---|---|---|---|
| Compile | `CGO_ENABLED=0 go build -json -trimpath -buildvcs=false -vet=off`; 15 s, 768 MiB, pids 256 | `go test -c -race -vet=off -trimpath`; 15 s, 1.25 GiB | Static classification only (§6.3) | `g++ -std=gnu++20 -O2 -static` + PCH |
| Slot | 1 | **1** (`cpu.max` 1 CPU, `GOMAXPROCS=2` to get interleavings) | 1 | 1 |
| Exec unit | A process per case; no `/proc` | A process per declared test: `-test.run '^Name$' -test.count=N -test.timeout`; fresh procfs with **`hidepid=invisible,subset=pid`** | Per-job postmaster + trusted `sqlharness`; backends moved into case cgroups | A process per case |
| Env | `GOMAXPROCS=1`, **`GOMEMLIMIT` unset**, `TZ=UTC`, `LANG=C.UTF-8`, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOFLAGS=-mod=readonly`, `GOTELEMETRY=off` | + `GORACE="halt_on_error=1 atexit_sleep_ms=0 exitcode=66"`, `GOTRACEBACK=all` | GUCs in §6.3 | — |
| Seccomp (exec) | KILL by default; no sockets; `clone3` → ENOSYS | + threads; `clone3` → ENOSYS (glibc ≥ 2.34 falls back to `clone`); **`personality(ADDR_NO_RANDOMIZE)` + re-exec of self if the amd64 replay shows TSAN needs it** | + AF_UNIX, shm | — |
| Case limits | CPU TL from the pack (≥ 1 s); wall 1.5·TL + 0.5 s; mem `mem_mb` (default 256) + baseline; pids 32; FSIZE 1 MiB; NOFILE 64; `/w` tmpfs 64 MiB / 4k inodes | per-test deadline ≥ 10× baseline; 1 GiB; pids 128; job ≤ 45 s; **Run: `-count=1`, job ≤ 20 s** | `statement_timeout` = TL; supervisor wall 3·TL + 2 s; case 256 MiB; `pg/` 256 MiB | — |
| Verdicts | T4 mapping; panic class on fd 4 | RACE > DEADLOCK > TLE > LEAK > RE > WA; a missing pass fails | Result set → WA; runtime SQL error, **including 25006** → RE; timeout → TLE; backend OOM → MLE | ASan/UBSan in Run only |
| Lint allowlist (public) | `fmt sort slices maps strings strconv math math/bits math/rand/v2 container/* unicode/* bytes errors cmp iter`; denies `os/* syscall unsafe net/* plugin embed runtime/* reflect C` | + `sync sync/atomic time context`, limited `runtime` | A single statement; static SELECT check | — |
| Warm timings **(inferred)** | compile 0.5–1.5 CPU-s; Run end-to-end ≈ 1.5–2.5 s; Submit ≈ 2–6 CPU-s | race compile 3–6 s | PG boot 0.3–1 s | — |

**Harnesses.**
- **`func-json@1` / `class-ops@1`:** a judge-generated `zz_xl_harness.go`; frames in on fd 3, canonical JSON out on fd 4; no reflection.
- **go-race** uses `goleak` through the harness-owned `TestMain`. Pack guidance: use `testing/synctest` for time-based tests.

**ASLR policy (corrected).** ASLR stays on by default. On amd64, noble's `vm.mmap_rnd_bits=32` breaks TSAN with "unexpected memory mapping"; arm64 is unaffected [W29]. TSAN from LLVM ≥ 18.1 re-executes itself with ASLR off when that happens [W30]. Whether Go 1.26's race runtime does the same is **(inferred)**, and the amd64 replay decides it. If needed, allow `personality(ADDR_NO_RANDOMIZE)` **for go-race processes only**. Never lower the host sysctl.

### 6.3 The SQL sandbox (`sql-pg@18`, pilot)
- **Placement:** a fresh PostgreSQL 18 per job, inside the jail. The baked `PGDATA` is copied into the job tmpfs by a trusted step running as the PG UID. `listen_addresses=''`, socket in `/job/sock`, loopback down.
- **Never** CNPG (T0/D12), a shared warm server, PGlite, DuckDB or SQLite. T0's always-on sandbox Postgres is **removed**.
- **Static classification before any hidden data exists:**
  1. Apply the public `schema.sql` into `schema_tpl`.
  2. `PREPARE` the learner statement there. A syntax or name error is CE. A plan whose top node is not a plain SELECT (no `ModifyTable`, no data-modifying CTE) is REJECTED. Both are pre-run and uncounted.
- **Run:**
  1. Load the samples (Run) or the streamed hidden instances (Submit, via `sql_rows@1`).
  2. `ANALYZE`.
  3. For each instance: `CREATE DATABASE inst_k TEMPLATE schema_tpl`; the harness connects; **the spawner moves `pg_backend_pid()` into `case-K`** (with `memory.max` and `oom.group`); the statement runs.
  4. Result sets are capped at ≤ 10,000 rows or 8 MiB, else OLE.
- **Mapping:**
  - backend OOM → MLE, after which the postmaster's crash recovery runs and the harness reconnects;
  - wall kill → TLE;
  - a postmaster death caused by a case → that case's verdict, **never infra**.
- **Guards:** SELECT-only role, `BEGIN READ ONLY`, extended protocol, `statement_timeout` / `transaction_timeout`, `temp_file_limit=64MB`, `work_mem=4MB`, no extensions or superuser, `log_statement=none`. **The real backstop is the wall kill plus the cgroup,** because `set_config` can bypass the in-server timeouts.
- **Determinism:**
  - `max_parallel_workers_per_gather=0`, `jit=off`, `autovacuum=off`, `io_method=sync`;
  - fsync, `full_page_writes` and `synchronous_commit` off; `wal_level=minimal`;
  - UTC, `DateStyle=ISO,YMD`, `lc_messages=C`;
  - hidden tables ≤ 30,000 rows;
  - EXPLAIN checks compare node types and relation names only.

### 6.4 C++
Deferred past M5 (owner Q1). The spec is ready: each language multiplies 151 references, calibration and codecs.

---

## 7. Limits, timing fairness and the single-node budget

### 7.1 Cgroup layout
```
<runner container>  cpu.max 2 · memory.max 3 GiB
├─ runner/          spawner + front (spool ≤ 2×16 MiB, artifact tmpfs ≤ 2×64 MiB, toolchain page cache)
└─ slots/  (+cpu +memory +pids)
   ├─ s0/  cpu.max 1 CPU · memory.max 1.25 GiB · pids.max 512
   │   └─ job/ ├─ compile/  ├─ pg/ (sql only, 256 MiB)  └─ case-K/ (memory.max item+baseline · oom.group 1 · swap.max 0)
   └─ s1/  same
```
- 2 × 1.25 GiB + `runner/` (≤ ~300 MiB of non-reclaimable memory **(inferred)**) < 3 GiB. **So a learner OOM always lands in a case, compile or `pg` cgroup and is never a container OOM** (INV-14; P2 gate).
- No `memory.high`.
- No `io.max`: every writable path is tmpfs.

### 7.2 Time-limit policy
- **The TL is CPU time.** TL = max(3 × the reference max on the production profile, 1 s).
- **Wall** = 1.5·TL + 0.5 s.
- **Idle kill:** CPU rate < 5% for ≥ 1 s gives TLE (idle).
- **Kill point:** CPU > TL + min(0.5 s, TL/2); `RLIMIT_CPU` backstop at TL + 2 s.
- Suspect time verdicts go to §5.7.
- **Pack lints (T1 amendments):** Σ TL ≤ 40 CPU-s; `mem_mb` ≥ 2 × the reference peak + baseline.

### 7.3 Calibration
- **`baseline@v`:** 5 kernels (integer loop, sort 1e6, map-heavy, alloc/GC-heavy, BFS on 2e5 edges) × 30 runs, measured **in the production runner at A8** (owner go-ahead) and keyed to `profile_sha256`.
- CI references are scaled by `speed_index(prod) / speed_index(CI)`.
- **Drift triggers:** a CPU model change; the canary median drifting > 15% for 24 h; a minor-version bump.
- TLs only scale up.

### 7.4 Per-account budgets and the duty-cycle breaker (T4 amendments)
**Budgets (inferred numbers, tuned with S0 data).**
- Runner time is measured in **slot-seconds**: compile, CE and quiet re-runs all count.
- Run and arena work: ≤ 10 slot-min per hour and ≤ 60 per day per account. Past that, "budget exhausted" (T4's DTO decides the wording).
- Counted submits, mock and P0 are **never** budget-blocked; T4 quotas and D18's timer already bound them.

**CE cache.** judge caches CE diagnostics by (`profile_sha256`, `harness@v`, sha256 of the files) for 24 h, so an identical resubmit never reaches the runner.

**Fairness.** T4's per-account claim serialization already means one running job per account. With equal slots there is no head-of-line blocking.

**Duty-cycle breaker (judge).** "Busy" is Σ slot-seconds / (2 × window).

| Busy over 60 min | Action |
|---|---|
| > 40% | Arena (P2) gets `run_after` +5 min; Runs ≤ 1 per 30 s per account |
| > 60% | Only P0/P1 counted submits and mock run |

The runner can never exceed 2 vCPU (50% of the VPS). Hostinger cuts CPU by 25% an hour under "sustained high CPU" but **does not publish the trigger** (page updated 2026-09-15 [W28]). S0's 72 h steal sampling sets the thresholds.

### 7.5 Contention with CNPG and the gateway
- **Runner resources:** requests 100m / 512Mi, limits 2 / 3Gi. The cpu.weight is about 11 against CNPG's 25.
- **Priority:** PriorityClass −1000 with `preemptionPolicy: Never`, which makes the runner **the first *kubelet-eviction* victim**.
  - *Correction:* PriorityClass does not set `oom_score_adj`. For the kernel OOM killer, Burstable pods get 1000 − 1000·request/capacity: the runner ≈ 968 and v1 pods at 32Mi ≈ 998. The runner is picked by the kernel only when its RSS is the largest.
- **CPU pinning:** none (no static CPU manager).
- **judge's CPU limit:** 500m.
- **Node limits sum:** about 12.3 Gi, not 15.5 (corrected; T0's 12.5 Gi already included the runner, and the always-on sandbox PG is removed).

### 7.6 Throughput and latency (inferred; measured at A8)
| Workload | Estimate |
|---|---|
| Go Submit | 2–6 CPU-s → **≈ 20–60 a minute** with 2 slots |
| go-race | 1 slot, ≈ 4–10 a minute |
| SQL | ≈ 30 a minute |
| Run, warm | enqueue ~20 ms + claim (**T4 amendment: in-process nudge**, ~20 ms) + jail ~20 ms + compile 0.5–1.5 s + samples ~50 ms + SPA poll (800 ms then 400 ms with ETag) → **p50 ≈ 1.5–2.5 s, p95 ≈ 3–4 s** |
| Quiet re-run | +≤ 60 s wait + the failing group; failing time verdicts only |

---

## 8. Cluster hardening plan

### 8.1 Flux layout (keep the runner off v1's critical path)
- **New `clusters/vps/sandbox.yaml`** with two Kustomizations, following the precedent of `messaging.yaml` / `databases.yaml` being "deliberately OFF the critical path":
  - `sandbox-guards` (`./infrastructure/sandbox`; interval 1m; `wait`; `prune`);
  - `runner` (`./runner`; `dependsOn: sandbox-guards`; SOPS decryption for the bearer token; `wait`). **Nothing depends on `runner`.**
- **judge** in `./apps` never waits on the runner. An unreachable runner means `saturated` plus an alert.
- **Image automation:** a **second ImageUpdateAutomation** with `update.path: ./runner`. The existing one updates only `./apps` (verified).
- **T2's laptop k3d rehearsal:** the runner is **expected to be NotReady** there, because the host prerequisites are missing.

### 8.2 Guard objects (`infrastructure/sandbox/`)
**Namespace `xlearn-runner`.**
- PSA `enforce: privileged`, needed only because baseline forbids adding `SYS_ADMIN`; the pod is never `privileged: true`.
- `warn`/`audit: baseline`.

**VAP.** Bound by `namespaceSelector kubernetes.io/metadata.name In [xlearn-runner]`, a label the API server sets, not a mutable custom one. `Deny`, `failurePolicy: Fail`. Operations: CREATE/UPDATE on pods and `pods/ephemeralcontainers`, and **CONNECT on `pods/exec` and `pods/attach`**. It requires:
- `runtimeClassName == xlearn-judge` and `hostUsers == false`;
- `runAsUser: 0` only with `hostUsers:false`;
- `automountServiceAccountToken == false`, `enableServiceLinks == false`, `priorityClassName == xlearn-sandbox-lowest`;
- no `hostNetwork`/`hostPID`/`hostIPC`/`shareProcessNamespace`/`hostPort`/`sysctls`;
- volumes ⊆ {emptyDir, secret `runner-auth`}; no `imagePullSecrets`; images match `ghcr.io/sujaykumarsuman/xlearn-runner@sha256:*`;
- `privileged` absent or false; `allowPrivilegeEscalation: false`;
- `capabilities.add ⊆ {SYS_ADMIN, SETUID, SETGID, SETPCAP, KILL}` (spelled without `CAP_` [W10]) and `drop ⊇ {ALL}`;
- AppArmor `Localhost/xlearn-runner`; seccomp `Localhost/profiles/xlearn-runner.json` at pod **and** container level (never Unconfined or RuntimeDefault);
- `procMount ∈ {Default, Unmasked}`;
- probes `httpGet` only; no `exec` lifecycle hooks;
- no ephemeral containers.

`SETPCAP` is added **(inferred)** to drop the bounding set in children; P1 records whether go-sandbox's `DropCaps` needs it.

**Other objects:**
- **RuntimeClass** `xlearn-judge` → handler `judge`.
- **PriorityClass** `xlearn-sandbox-lowest`: −1000, `preemptionPolicy: Never`.
- **ResourceQuota:** pods 2, requests 500m / 1Gi, limits 2 / 3Gi, PVCs 0, services 1, NodePorts 0, LoadBalancers 0.
- **LimitRange.**
- **NetworkPolicies:** `default-deny-all` (Ingress + Egress, **no DNS**) and `judge-to-runner` (namespace `xlearn` AND pod `app.kubernetes.io/instance: xlearn-judge` → TCP 8090).
- The draft's `ServiceAccount default` object is **dropped**: chart pods use their own service account, and the pod-level knob plus the VAP enforce no token.
- A proposed rule forbidding the `privileged` PSA label elsewhere is **dropped**. Unlabelled namespaces are already privileged by default on this cluster (no PSA defaults configured **(inferred)**); Track B labels fix that.

### 8.3 Chart 0.2.2 → 0.3.0 and the runner values

_(Fixed 2026-09-24 at the v2 build-plan sign-off: the live `charts/project` chart is **0.2.2**, not 0.2.1, as ADR-0030 §7 records. The heading used to say 0.2.1.)_

- **New knobs, all default-off:**
  - `automountServiceAccountToken`, `runtimeClassName`, `priorityClassName`, `hostUsers`;
  - `dnsPolicy`/`dnsConfig`, `terminationGracePeriodSeconds`;
  - split readiness and liveness paths;
  - `image.digest`;
  - `workload: cronjob`.
- **Gate:** a byte-identical `helm template` for every existing release (`hack/chart-diff.sh`). Booleans are emitted with `kindIs "invalid"`.
- **Runner values:**
  - `podSecurityContext: {runAsUser: 0, runAsNonRoot: false}`, overriding the chart default of 65532 (verified in `values.yaml`); UID 0 exists **inside the user namespace only**;
  - container `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, caps as in §8.2, `procMount: Unmasked`, AppArmor and seccomp Localhost;
  - `dnsPolicy: None` with nameserver `127.0.0.1`;
  - `Recreate`; `terminationGracePeriodSeconds: 70`; `/readyz` probe.
- ***Rationale corrected:*** `Unmasked` is needed because a fresh procfs cannot be mounted in a nested PID namespace while masked paths exist (KEP-4265). It is **not** about `/proc/self/maps`. Jail procfs uses `hidepid=invisible,subset=pid`.

### 8.4 NetworkPolicies for v1 (raw manifests)
- **Track A (hard M3 precondition, T4 §10), ingress:**
  - PG 5432 admits an explicit xlearn list; the gateway is excluded;
  - CNPG :8000 admits `cnpg-system` only;
  - NATS 4222 admits practice, review and assessment (+ judge, identity and coach in v2);
  - NATS :8222 admits opscheck only.
- **Track B:** xlearn egress (kube-dns everywhere; :443 only for identity and coach).
- Selection uses `app.kubernetes.io/instance`.
- **Known limit:** node-local traffic is always admitted.

### 8.5 NATS authentication
- **Track A:** nkeys in the global account, with coarse permissions (each service publishes on `xlearn.<svc>.>` plus JetStream API access). The server config holds public keys only, so **`messaging` needs no SOPS decryption**. Seeds go in `apps/secrets/`. Clients ship first (`NATS_NKEY_SEED_FILE` is optional); ≤ ~2 min of event lag, no loss **(inferred)**.
- **Track B:** fine-grained per-service `$JS.API`/ack/inbox ACLs; purge and delete only for an offline ops identity.

### 8.6 PSA, tokens and PG (all Track B)
| Namespace | Enforce | Warn / audit | Note |
|---|---|---|---|
| `xlearn` | baseline | restricted | judge's D5 `image` volume isn't in restricted's list |
| `databases` | restricted | restricted | CNPG is restricted-clean |
| `messaging` | baseline | restricted | The NATS chart sets no securityContext |
| `xlearn-runner` | privileged | baseline | Plus the VAP |

Also in Track B:
- tokens off on all v1 releases;
- CNPG per-role `connectionLimit: 20`;
- **CNPG 18.4 → ≥ 18.6** [W18], early in Track B.

### 8.7 Host-level changes (manual, scripted, recorded)
**Extend T2's planned `I/hack/host-bootstrap.sh`** (open-iscsi, ufw, pinned k3s) rather than writing a parallel script. Add a read-only `host-verify.sh`, `host-bom.txt` and a line in `dr-runbook.md`. On a rebuild, the sandbox files are written **before** the pinned k3s install, so no extra restart is needed.

| Block | Contents | When |
|---|---|---|
| **H0 patch gate** (protects v1 now) | See the list below | Now (owner go-ahead); reboot blip |
| **Sandbox** | See the list below | H8/A6, in a window; k3s restart |
| Track B | `kubelet-arg`: `pod-max-pids=4096`, kube- and system-reserved (200m / 512Mi / pid 2000), `eviction-hard` **(inferred)** | Any time |

**H0 patch gate:**
- `apt full-upgrade` to the current noble GA kernel (≥ 6.8.0-117), then reboot;
- unattended-upgrades and Livepatch (Q3);
- `/etc/modprobe.d/60-xlearn-deny.conf` with `install <mod> /bin/false` for the §2.4 A0 list, reviewed against S0's `lsmod`;
- `kernel.core_pattern=core`, `fs.suid_dumpable=0`, apport disabled, systemd-coredump masked.

**Sandbox block:**
- **Sysctls** (`60-xlearn-sandbox.conf`): `io_uring_disabled=2`, `unprivileged_bpf_disabled=2`, `vm.unprivileged_userfaultfd=0`, `dmesg_restrict=1`, `kptr_restrict=2`; assert `perf_event_paranoid ≥ 3`; pin `apparmor_restrict_unprivileged_userns=1`.
- **containerd drop-in** `/var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d/20-judge.toml`. The k3s v1.36.4 base template imports `config-v3.toml.d/*.toml` (L184 [W32]). **No `.tmpl` and no `BinaryName`**: runc is found through k3s's PATH, and the rendered runc section has no BinaryName (L242–246).
  ```toml
  version = 3
  [plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.judge]
    runtime_type = "io.containerd.runc.v2"
    cgroup_writable = true
  [plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.judge.options]
    SystemdCgroup = true
  ```
- **AppArmor** `/etc/apparmor.d/xlearn-runner`:
  - `abi <abi/4.0>`, **no `userns` rule**;
  - `mount fstype=tmpfs|overlay|proc` and `bind` under the jail paths only; `pivot_root`, `umount`;
  - writes to its own `/sys/fs/cgroup/**` only;
  - capabilities `sys_admin setuid setgid setpcap kill`;
  - `deny ptrace`.
- **Seccomp** `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json`: RuntimeDefault-with-SYS_ADMIN **minus**:
  - `fsopen fsconfig fsmount fspick move_mount open_tree mount_setattr`;
  - `bpf perf_event_open fanotify_*`;
  - `keyctl add_key request_key io_uring_* userfaultfd lookup_dcookie syslog`;
  - with `socket()` limited to AF_UNIX and AF_INET/INET6 `SOCK_STREAM` (no SCTP, NETLINK, PACKET, ALG or VSOCK).
- **User-namespace range:** a system user `kubelet`, and `/etc/subuid` + `/etc/subgid` `kubelet:<start>:7208960` (110 × 65,536) chosen to end well below UINT32_MAX. This avoids kubernetes #139916 (open) [W26][W27]. Install `getsubids`.
- Restart k3s, then run `host-verify`.

**`host-verify` runs after every reboot, k3s upgrade and rebuild.** It asserts:
- `uname -r` ≥ the BOM floor; `modprobe -n` fails for each denied module;
- `core_pattern` and the sysctls;
- `crictl info` lists `judge`, the `judge` runc version equals the default runc, and the rendered runc still has `SystemdCgroup = true`;
- the drop-in matches the BOM, and it fails loudly if a future k3s renames the drop-in directory;
- `aa-status` shows the profile; the seccomp file hash; the subuid entry.

### 8.8 Rollout order that keeps v1 up
| # | Track | Change | v1 risk | Gate before the next step |
|---|---|---|---|---|
| A0 | A | **H0 host patch gate** | Reboot blip | `host-verify --h0` green; all pods Running |
| A1 | A | **S0** read-only production facts + 72 h steal sampling (owner go-ahead) | none | Facts table; thresholds set |
| A2 | A | Spike (§9) | none | Results table → ADR-0030 |
| A3 | A | Chart 0.3.0 | none | Empty chart diff |
| A4 | A | `databases` + `messaging` ingress NetworkPolicies | medium (a missed caller) | CNPG Ready; consumers bound; login, dashboard and coach work |
| A5 | A | NATS nkeys, coarse (clients first, then server) | ≤ ~2 min event lag | No permission errors; outbox drained |
| A6 | A | Host sandbox block, in a window | k3s restart blip | `host-verify` green |
| A7 | A | `sandbox-guards` | none | Flux Ready; VAP denies the bad shapes (`--dry-run=server`) |
| A8 | A | `runner`, dark: acceptance suite (§5.10, network and syscall probes), timing thresholds from S0, `baseline@1` | none | All gates pass |
| A9 | A | judge M3 wiring behind a flag (T7) | per T4 | Kill switch |
| B* | B | PSA labels; tokens off; xlearn egress; fine NATS ACLs; kubelet reserved/eviction; PG role limits; CNPG 18.6; k3s v1.36.5 (containerd 2.3.5) when GA; Renovate | low–medium | Per item; none gates M3 |

Every step reverts with `git revert`. Host steps revert by restoring the files and restarting k3s or rebooting.

---

## 9. The smallest spike (local and throwaway; needs the owner's go-ahead)

**Open questions it answers (the build choice depends on these alone):**
- **Q-A (go/no-go):** Does R1 work on Ubuntu 24.04 with the current noble kernel, k3s v1.36.4 (containerd 2.3.4, runc 1.4.2) and the AppArmor restriction? Specifically:
  - the drop-in handler with `cgroup_writable` and systemd;
  - `hostUsers:false`, namespaced caps, Localhost AppArmor (no `userns`) and Localhost seccomp;
  - per-case cgroups;
  - **userns-less jails through go-sandbox `forkexec.Runner`**;
  - user-namespace creation and the new mount API denied to the supervisor.
- **Q-B (INV-14):** Does every learner OOM land in a case cgroup, including a **SQL backend moved into a case cgroup**, and does cgroup cleanup converge?
- **Q-C (amd64):**
  - Does `go test -race` work in the jail with noble amd64's `mmap_rnd_bits=32`, and which syscalls does TSAN's ASLR re-exec need?
  - What is the amd64 exec allowlist?

**Not in the spike (moved to A8 on production, informed by S0):**
- latency, CV, canary thresholds and cpuset A/B;
- choosing the `GOCACHE` mechanism;
- the cross-account marker suite;
- gVisor (P5 dropped).

Arm64 timings under macOS/HVF don't carry over to a Hostinger EPYC guest.

**Environment:**
- **Part 1 (P0–P2): a multipass Ubuntu 24.04 arm64 VM on the M3 Max.** `multipass launch 24.04 --cpus 4 --memory 8G --disk 30G`, then `apt full-upgrade` + reboot to mirror H0.
  - Assert `apparmor_restrict_unprivileged_userns=1`, cgroup2 and no `/dev/kvm`.
  - Install inside the VM only: k3s `INSTALL_K3S_VERSION=v1.36.4+k3s1`, go1.26.8, PGDG PG 18.
  - *Why:* user namespaces, cgroup delegation, AppArmor mediation, seccomp semantics and procfs rules don't depend on the architecture. Docker Desktop's linuxkit lacks Ubuntu's AppArmor restriction.
- **Part 2 (P3): amd64 functional replay, with no k3s.** The throwaway supervisor runs as root in a delegated systemd scope with the same AppArmor and seccomp profiles.
  - **Use the owner's second VPS if it already exists, is amd64 and is empty.** It is closest to production: a generic noble kernel on a KVM guest. **Reimage it afterwards,** before it becomes the backup target.
  - **Otherwise,** a throwaway GitHub Actions `ubuntu-24.04` job (amd64) in a private scratch repo, with the userns restriction asserted first. Its kernel is linux-azure, so no timing conclusions.
- **S0 is a separate ask** (read-only on production, before A6):
  - `grep -A8 runtimes.runc …/config.toml`;
  - `/proc/pressure/cpu`;
  - `grep PARAVIRT_TIME_ACCOUNTING /boot/config-$(uname -r)`;
  - `lscpu`;
  - `/proc/sys/kernel/core_pattern` and `systemctl is-enabled apport`;
  - `sysctl vm.mmap_rnd_bits`;
  - `lsmod` for the denylist modules;
  - `uname -r`;
  - 72 h of `vmstat 60` steal sampling to `/tmp`.

**Steps.** Time box: **10 h core, hard stop at 1.5 working days.**

| Phase (h) | Steps | Measure | Pass | Fail → |
|---|---|---|---|---|
| **P0 setup (1.5)** | VM + kernel upgrade; k3s; write the **drop-in** (not a template), AppArmor, seccomp file and subuid range; restart k3s (record whether running pods survive); apply guard objects | `crictl info`; the rendered runc section; runc versions; VAP `--dry-run=server` with ≥ 8 bad shapes; a real `kubectl exec` attempt | `judge` listed; `SystemdCgroup=true` still rendered; same runc; **VAP denies 8/8 and CONNECT exec is denied** | Fix the drop-in; if it isn't merged, fall back to a minimal `.tmpl` and record it |
| **P1 viability (3)** | See the checklist below | errno per step | **All positive checks succeed; all negative probes fail** | forkexec fails → nsjail `--disable_clone_newuser` (R1-N) → R1-U → R1b (Q2 switches to b) |
| **P1b pilot check (0.75, not gating M3)** | `go test -c -race` + run in the jail (`clone3` → ENOSYS with glibc); `postgres` boot in the jail on a Unix socket | works or errno | Both run | The pilot profile is redesigned before the pilot |
| **P2 INV-14 and cleanup (2.5)** | Throwaway ~400-line supervisor; corpus below | Class per run; container OOM count; `memory.current`; `nr_dying_descendants` after 1,000 case cgroups | **MLE 100/100 (SQL included) and 0 container OOMs**; 0 survivors; baseline ±5 MiB; `nr_dying_descendants` → ~0 within 60 s; RACE ≥ 19/20; DEADLOCK 20/20 | Tune limits; **any container OOM is a blocker** |
| **P3 amd64 replay (2)** | Same supervisor and profiles; record `mmap_rnd_bits` and set it to 32 if lower; `go test -race` in the jail: does it re-exec with ADDR_NO_RANDOMIZE, and which syscalls does it need; postgres; generate amd64 allowlists in `SECCOMP_RET_LOG` mode over the corpus + ~20 reference solutions; re-run the P2 balloon/fork subset under KILL | Syscall sets; race result; classes | Race works with a **per-process** ASLR policy; **0 unexpected SIGSYS** under KILL after the allowlist | Per-process `setarch -R` launcher; if still failing, go-race is out of the pilot |
| **Report (1)** | Results table → ADR-0030 draft | — | — | — |

**P1 checklist.**
- *Positive:* runner pod up with `hostUsers:false`, 5 caps, Localhost AppArmor and seccomp, `Unmasked`.
- *Positive:* `mkdir /sys/fs/cgroup/slots` and enable `+cpu +memory +pids`.
- *Positive:* a `forkexec.Runner` jail **without CLONE_NEWUSER** (mnt/pid/net/ipc/uts/cgroup), with tmpfs root, `nosuid,nodev` read-only binds, one overlay, `pivot_root` and procfs `hidepid=invisible,subset=pid`.
- *Positive:* spawn into a case cgroup both with `CLONE_INTO_CGROUP` and with a `cgroup.procs` write + sync pipe on 6.8.
- *Positive:* caps to 0 and bounding-set drop (is SETPCAP needed?); `go build` in the jail.
- *Negative probes from the supervisor:* `unshare -U`, `clone3(CLONE_NEWUSER)`, `fsopen`+`move_mount`, `open_tree`+`move_mount`, `socket(SCTP)`, `socket(AF_NETLINK)`.
- *Negative probes from the jail:* a syscall probe (SIGSYS); then confirm **no host core helper spawned**.
- *Stability:* pod recreate loop × 50 (subuid).

**P2 corpus.** 1 GiB balloon, fork bomb, thread bomb, tmpfs fill, inode fill, stdout flood, orphan double-fork, sleep/idle, spin, **SQL balloon** (`array_agg`/recursive CTE, backend moved into `case-K`), **SQL `set_config` timeout bypass**, lock-order deadlock, race fixture.

**What is thrown away:**
- the VM (`multipass delete --purge`);
- the second VPS, reimaged, or the scratch repo, deleted;
- k3s, the manifests, the throwaway supervisor and the corpus, which live only in the scratchpad and are never committed.

Only the results table and the numbers survive, into ADR-0030 "Runner technology and host hardening" and T7.

---

## 10. What T3 constrains downstream

**T5 (platform AI)**
- The runner has no egress, no secrets and no LLM path. AI never calls the runner.
- The analyzer sees code, the verdict class, learner-file diagnostics, bucketed usage, and in Run only the learner's own stdout.
- It **never** sees hidden inputs, expected outputs, per-case timings or ordinals, hidden test sources or dump arguments. Those are allowed only in platform-key `Score` calls (T1).
- Anything the AI wants executed goes through judge as a Run on public samples, under the learner's budget.

**T6 (mock interviewer)**
- Coding rounds reuse the runner through judge (context `mock`, P1). The 2 slots are shared.
- Mock is **never shed** by the breaker but is budget-counted; T6 caps Run frequency (≤ 1 per 10 s).
- No media path reaches `xlearn-runner`. Run p50 < 5 s holds.

**T7 (rollout)** owns:
- §8.8 Track A/B, S0, H0 and ADR-0030;
- the runner image and its **separate release stream**, the `sandbox.yaml` Kustomizations and the second ImageUpdateAutomation;
- chart 0.3.0; extending T2's host scripts and BOM; the DR-runbook step;
- NATS nkeys, NetworkPolicies, PSA, CNPG 18.6, Renovate, and the production calibration.

**Amendments to settled docs:**
- **T4 §9.1:**
  - in-process claim nudge;
  - runner-lane retry classification (§5.9);
  - **per-account runner-second budgets, the CE cache and the duty-cycle breaker (§7.4)**;
  - `throttled` → one re-queue after 5 min, then `inconclusive(runner_throttled)`.
- **T4 §2.7:** Run poll cadence.
- **T4 §11.1:**
  - `Job` gains `HiddenFiles`, `Tests[]` and `OutputMode`;
  - `Result.Versions` gains `Toolchain`, `ProfileSHA`, `CPUModel`, `CanaryMedian` and `BootEpoch`;
  - `Cases` gains `OutputSHA256`;
  - `throttled` semantics (§5.7);
  - **runtime SIGSYS → RE (counted)**; SQL 25006 → RE;
  - wall = 1.5·TL + 0.5 s; idle kill;
  - go-race Run `-count=1`, ≤ 20 s.
  - **Dropped from the draft:** `Job.Cost` and token-aware claiming.
- **T4 §5.6 #15 (lint):** the A5 file-type and directive rules.
- **T1:**
  - `sql_rows@1` replaces `gen-hidden.sql`; SQL static classification; hidden tables ≤ 30k rows;
  - lints Σ TL ≤ 40 CPU-s and `mem_mb` ≥ 2 × the reference peak;
  - `time_ms` is learner CPU time.
- **T0:** no always-on sandbox PG; NetworkPolicies are raw manifests; runner requests 100m / 512Mi; limits total ≈ 12.3 Gi.
- **T2:** `host-bootstrap.sh` gains the H0 and sandbox blocks; the k3d rehearsal expects the runner NotReady.
- **feasibility.md:**
  - `messaging` needs **no** SOPS decryption;
  - dependency-matrix rows: runner image and stream, containerd drop-in, AppArmor and seccomp files, subuid, VAP, H0, S0, NATS nkeys;
  - `xlearn` PSA is baseline because of D5.

**Telemetry.** judge persists per-job runner telemetry in its own schema: steal, canary, busy ms, `profile_sha`, digest, `BootEpoch`, quiet re-runs. opscheck (a daily CronJob; there is no metrics stack, per ADR-0009) queries it for:
- steal p95 > 10% over 24 h;
- re-run rate > 5%;
- inconclusive > 0;
- busy > 40% for 2 h;
- SIGSYS or quarantine events;
- `host-verify` drift.

judge also logs and alerts when the **runner lane is unavailable or returning 503 for > 10 min**; T7 picks the channel.

**Scale-out.** The runner is stateless, so moving it to another host is a network change only.

---

## 11. Alternatives considered

| Alternative | Why not (and when to revisit) |
|---|---|
| gVisor as the pod runtime | No limits inside the sandbox [W1] breaks per-case MLE and INV-14; manual weekly upgrades. |
| gVisor for the execute step only | 50–400 ms per sandbox plus a memory watchdog. **Revisit** on open signup or a kernel-CVE burden. |
| Firecracker, Kata, gVisor-KVM | No `/dev/kvm`. |
| Judge0, Piston, DMOJ, go-judge as a service | Privileged containers, stored expected outputs, ptrace/`RLIMIT_AS`, or pooled containers (§4). |
| go-sandbox's `container` package with a long-lived in-jail init | Forces a user namespace; the init keeps ambient `SYS_ADMIN` and runs path operations next to learner code (a TOCTOU risk). |
| isolate (Plan C) or nsjail (Plan B) as the first choice | Both are viable in R1. nsjail reports no resource usage; isolate is setuid, GPL-2 and "not recommended" in containers. Kept as fallbacks. |
| bubblewrap, Sysbox, Landlock only | User namespaces blocked, no cgroups, or no memory limit. |
| A pod or Job per submission | 1–5 s start; judge would need RBAC. |
| Privileged pod | Judge0 CVE-2024-28185 class. Never. |
| Hosted sandboxes | Hidden inputs leave the platform. |
| A shared warm Postgres | State and CVEs cross accounts; an OOM hits everyone. |
| PGlite, DuckDB, SQLite | Dialect drift from PG 18. |
| A full `config-v3.toml.tmpl` with a pinned `BinaryName` | Goes stale on k3s upgrades; the drop-in is lighter. |
| Guards in `infra-configs`, runner in `./apps` | Couples runner failures to every v1 reconcile. |
| Runner built on every fleet `v*` tag; baseline keyed to the digest | Downtime and recalibration on each release; patching would stall. |
| 2-token go-race + token-aware claiming + PSI/`nr_throttled` signals | Over-built; head-of-line blocking; lets a learner flag itself. Replaced by equal slots, steal/canary, in-runner re-runs. |
| Runtime SIGSYS or SQL-write errors as REJECTED (uncounted) | Creates an uncounted oracle over hidden data (A18). |
| Wall-clock TLs; `RLIMIT_AS`; `GOMEMLIMIT` near the limit | Noisy; breaks TSAN and ASan; turns MLE into TLE. |
| Host `vm.mmap_rnd_bits=28` for TSAN | Weakens ASLR for the whole host; a per-process setting suffices. |
| Kubelet static CPU manager | Takes 2 of 4 vCPUs from everything else. |
| Pre-warmed sandbox pool | Saves 10–30 ms against a ~1 s compile; reuse risk. |
| A dedicated runner VPS | The strongest blast-radius cut. Owner Q2(b). |

---

## 12. Owner decisions — resolved 2026-09-24 (they OVERRIDE the body where they conflict)

| # | Question | Decision |
|---|----------|----------|
| **D20** | Q1 / PRD Q6: launch languages | **Go, C++ and Python at M3**, not "Go only". Consequences the build plan must absorb:<br>• three runner profiles (`go@1.26`, `cpp` with g++ `-std=gnu++20 -O2`, `python@3.x`) and three `func-json@1` / `class-ops@1` harness codecs;<br>• three exec seccomp allowlists (Python needs the widest), all generated on **amd64**;<br>• per-language time limits: a Go reference plus per-language multipliers calibrated at A8, with per-language references for fully packed items in CI to catch harness and signature bugs;<br>• C++ compile-bomb limits;<br>• a larger runner image;<br>• roughly +35–50 authoring hours for C++/Python references on the ~35 fully packed items **(inferred)**.<br>§6.4 "C++ deferred" is superseded. go-race and sql-pg stay with the second-course pilot. |
| **D21** | Q2: where untrusted code runs | **On the production node** with R1, H0 and the patch cadence, while signup is invite-only (D13). **Move it to a dedicated runner VPS on any trigger:** open signup; a reachable, unpatched LPE older than 7 days; P1 forcing R1b; chronic steal. Never the backup VPS. |
| **D22** | Q3: patch cadence | **Unattended security upgrades** (with the kernel meta-packages **unheld**; the Hostinger image had held them), **no Livepatch** (owner, 2026-09-24), a **monthly reboot window**, **monthly k3s patch releases pinned in git**, and **Renovate** for the runner image and go-sandbox pins. **H0 is not optional**: it is a separate task on the live node. |
| **D23** | The spike go-ahead | **Only S0 (read-only production facts) was approved and run** (§15). The mechanism spike **P0–P3 is deferred.** It becomes a **build-plan gate before M3** (T7). Until then R1's go/no-go questions (Q-A, Q-B, Q-C) are open, with nsjail → R1-U → R1b as the fallbacks. The spike must now also generate the **C++ and Python** amd64 allowlists (D20). |

## 13. Risks

| Risk | Likelihood / Impact | Mitigation |
|---|---|---|
| **Known-unpatched kernel LPEs on the live node today** (6.8.0-90; DiagSpill still unfixed on noble) | **H / Critical** | H0 now: kernel ≥ 6.8.0-117, Livepatch, module denylist; kernel floor in `host-verify` |
| Kernel 0-day through allowlisted syscalls on a node holding everything (D12) | L / Critical | KILL-default allowlist; no userns, io_uring, bpf or userfaultfd; Q3; Q2 triggers |
| Runner compromise → verdict forgery and privacy loss across jobs | L / High | Privilege split; restart every 500 jobs / 6 h and on events; `BootEpoch` regrade; no exec |
| go-sandbox `forkexec.Runner` can't run without a user namespace on this stack | M / Med | P1 gate; nsjail Plan B; R1-U; R1b |
| The drop-in isn't merged, or a future k3s renames it | L / High | P0 assert; `host-verify` diff after every upgrade |
| Systemd driver not selected (reopens #12182) | L / High | S0; bootstrap preflight hard-fails |
| AppArmor without `userns` doesn't bind a `SYS_ADMIN` holder [W19] | M / Med | P1 asserts `unshare -U` and `clone3(NEWUSER)` fail; seccomp denies namespace flags in jails |
| amd64 TSAN vs `mmap_rnd_bits=32` [W29] | M / Med (pilot only) | P3 replay; per-process ASLR policy; go-race out of the pilot if it fails |
| An amd64 allowlist gap → SIGSYS → wrongly counted RE | M / Med | Allowlists generated in `SECCOMP_RET_LOG` over all 151 references in CI; ops alert on every SIGSYS; regrade path |
| Hostinger throttling or chronic steal | M / Med | Duty-cycle breaker; budgets; S0 thresholds; bounded quiet path; Q2(b) if chronic |
| #139916 `hostUsers:false` sandbox failures [W26] | M / Med | Bounded subuid range; P0 recreate loop × 50; alert on runner lane down > 10 min |
| containerd 2.3.4 CVE-2026-53495 (exec-probe DoS) [W25] | L / Med | VAP bans exec probes and hooks; k3s v1.36.5 when GA |
| A VAP bug under the PSA `privileged` label | L / High | VAP tested in P0 and in CI (`--dry-run=server`); binding by `metadata.name` |
| Manual host drift or loss on rebuild | M / Med | T2's script extended; BOM; `host-verify`; readyz canaries; DR ordering |
| Honor profiles leak hidden tests | certain / Low | `trust=honor`; excluded from "judge-checked %" |
| Node-local NetworkPolicy exemption | M / Low | Jail netns is primary; bearer token; front is capless |
| Poison pills; the owner's own deploys quarantining learners | L / Med | §5.9 classification (a rollout is `saturated`) |
| Supply chain | L / Med | Pinned digests and checksums; setuid stripped; reproducible build; Renovate; digest per evaluation |

---

## 14. Critic findings addressed

### 14.1 Red-team critique (score 6)
| # | Finding (severity) | Resolution | Where |
|---|---|---|---|
| 1 | Unpatched kernel; "patched host OS" trusted (**blocker**) | **Fixed.** H0 gate before anything else, and now for v1: kernel ≥ 6.8.0-117, Livepatch, `install … /bin/false` module denylist; kernel floor in BOM and `host-verify`; spike runs the current noble kernel. Verified: CrackArmor fixed in 6.8.0-106 [W22]; Copy Fail fixed in 6.8.0-117 [W23]; CVE-2026-74469 still "Vulnerable" on noble [W21]. | §2.1, A0, §8.7, §8.8 A0, §9 |
| 2 | `SYS_ADMIN` opens the new mount API, bpf and more in RuntimeDefault (major) | **Fixed.** Localhost pod seccomp (RuntimeDefault minus the listed calls, socket families restricted), pinned by the VAP; P1 probes. The `SYS_ADMIN` branch was verified in containerd's source [W33]. | A11, §8.2, §8.7, §9 P1 |
| 3 | go-sandbox `container` package forces a user namespace; its init keeps caps and does path operations (major) | **Fixed.** `forkexec.Runner` only; no long-lived init; R-FS extended to all capable processes; privilege split. Verified in source. | §4, §5.2, §5.4 |
| 4 | Apport core helper as a root deputy (major) | **Fixed.** `core_pattern=core`, `suid_dumpable=0`, apport off, `RLIMIT_CORE=0`; S0 reads the current value; P1 checks no helper spawns. | A15, §8.7, §9 |
| 5 | Runtime REJECTED is an uncounted oracle; lint gaps (major) | **Fixed.** Only pre-run rejections are uncounted; SIGSYS and 25006 → RE (counted); file-type allowlist; directive denylist including `//line`; SQL static classification on the schema-only database. | A5, A18, §5.5, §6.3 |
| 6 | Supervisor-compromise residual understated (major) | **Fixed.** Restated as forgery plus privacy loss for all jobs until restart; privilege split; restarts every 500 jobs / 6 h and on events; `BootEpoch`; VAP denies exec. | §2.4 residuals, §5.2 |
| 7 | Quotas count jobs, not CPU; Hostinger throttle (major) | **Fixed.** Slot-second budgets including CE; CE cache; duty-cycle breaker; go-race Run `-count=1`, 20 s. *Correction:* Hostinger's page states 25% per hour but **no duration** (the "~180 min" is not on the page as of 2026-09-15 [W28]). | A17, §7.4 |
| 8 | Cross-slot integrity attack (major) | **Fixed.** Automatic quiet re-run whenever the other slot was busy; memory-bound canary. | A16, §5.7 |
| 9 | SQL memory not attributed per case (major) | **Fixed.** Backends moved into case cgroups; MLE/TLE mapping; never infra; SQL balloon and `set_config` programs in P2. | §6.3, §9 P2 |
| 10 | `clone3` under KILL breaks glibc threads (minor) | **Fixed.** `clone3` → ENOSYS; P1b checks. | §6.2 |
| 11 | Wrong reason for `procMount: Unmasked` (minor) | **Fixed.** Rationale corrected; `hidepid=invisible,subset=pid`. | §8.3, §6.2 |
| 12 | `BinaryName` pins a stale runc (minor) | **Fixed.** Dropped; drop-in; `host-verify` compares runc versions. runc 1.4.3 / CVE-2026-41579 verified [W24]. | §8.7 |
| 13 | VAP gaps (minor) | **Fixed.** Seccomp Localhost, no hostPort/`shareProcessNamespace`/sysctls, `httpGet`-only probes, no exec hooks, UID 0 only with `hostUsers:false`, CONNECT exec denied. CVE-2026-53495 verified [W25]. | §8.2 |
| 14 | `hostUsers:false` UID-range bug (minor) | **Fixed.** Bounded `kubelet` subuid range + P0 loop. #139916 verified open [W26]. | §8.7, §9 P0 |
| 15 | go-race dump and `test2json` leakage or forgery (minor) | **Fixed.** Arguments and PC offsets stripped; Output dropped in Submit; fd-4 `TestMain`. | §5.8 |
| 16 | setuid binaries in the image; unauthenticated API (minor) | **Fixed.** Bits stripped; `nosuid,nodev` binds; bearer token from SOPS. | §6.1, §5.3 |
| 17 | Compile jail too lax; compile-limit verdict undefined (minor) | **Fixed.** ENOSYS by default, KILL for dangerous calls, no sockets; limit → CE with CPU counted. | §5.4 |
| — | Claim "containerd 2.3.4 affected by CVE-2026-95838" | **Not confirmed.** The 2.3.5 release lists only CVE-2026-53495 and GHSA-rp3h-jf77-q9p4 [W25]. | Appendix |

### 14.2 Ops, performance and GitOps critique (score 6)
| # | Finding (severity) | Resolution | Where |
|---|---|---|---|
| 1 | `BinaryName` doesn't exist; a full template is heavy (major) | **Fixed.** Drop-in `config-v3.toml.d/20-judge.toml`. Verified: `templates.go` L184 imports and L242–246 have no BinaryName [W32]. | §8.7 |
| 2 | Guards and runner on v1's critical path (major) | **Fixed.** `clusters/vps/sandbox.yaml` with two Kustomizations; nothing depends on the runner; a second ImageUpdateAutomation. Verified: `apps` has `wait: true`, `infra-configs` 30m, automation path `./apps`. | §8.1 |
| 3 | Fleet-wide release rebuilds the runner; calibration tied to the digest (major) | **Fixed.** `runner-v*` stream; reproducible build; `profile_sha256` keying; automatic ±5% speed check on patch bumps. Verified: deploy.yml builds all images on `v*`. | §6.1 |
| 4 | `killed` counts toward poison pills (major) | **Fixed.** Rollout, drain and SIGTERM → `saturated`, no retry spent; quarantine only on a repeat kill by the same submission. | §5.9 |
| 5 | Quiet re-run unbounded; S0 optional (major) | **Fixed.** ≤ 60 s in the runner, then ≤ 1 re-queue after 5 min → bounded ~10 min; S0 required before A6 (still a separate owner ask, read-only). | §5.7, §8.8, §9 |
| 6 | Arm64 timing gates meaningless; amd64 TSAN risk (major) | **Fixed.** Timing moved to A8; amd64 replay P3. LP #2056762 verified [W29]; TSAN re-exec behaviour [W30]. | §9 |
| 7 | go-sandbox userns coupling and `CLONE_INTO_CGROUP` on 6.8 (major) | **Fixed.** The coupling is in the `container` builder, not `forkexec.Runner`, so no fork is needed. `CgroupFD` passes through (go-judge call site). P1 tests both clone3 and `cgroup.procs` + sync. Plan B nsjail. | §4, §5.4, §9 |
| 8 | Spike not minimal (minor) | **Fixed.** 10 h core: P3 markers and P4 timing moved to A8; P5 dropped; VAP via `--dry-run=server`. | §9 |
| 9 | Hardening sequenced before the runner (minor) | **Fixed.** Track A (gates M3) / Track B. | §8.8 |
| 10 | `throttled` over-built (minor) | **Fixed.** Equal slots; steal or canary only; in-runner re-run; `Job.Cost`, token-aware claiming and PSI admission dropped; ~150 ms mixed canary run on demand. | §5.7 |
| 11 | Telemetry has nowhere to go (minor) | **Fixed.** judge persists it; opscheck queries it; judge alerts on the runner lane being down > 10 min. | §10 |
| 12 | A parallel host-bootstrap script (minor) | **Fixed.** Extend T2's script; files written before the k3s install on DR. | §8.7 |
| 13 | k3s, runc and pins never get patched (minor) | **Fixed.** Q3(a) monthly k3s + Renovate. | §12 |
| 14 | `runAsUser` default; SA object has no effect (minor) | **Fixed.** Runner values UID 0 (userns only); SA object dropped; pod-level knob plus VAP. Verified `runAsUser: 65532` in chart values. | §8.2, §8.3 |
| 15 | VAP bound by a mutable label (minor) | **Fixed.** `kubernetes.io/metadata.name`. The rule forbidding the privileged label elsewhere was not adopted: unlabelled namespaces are already privileged by default **(inferred)**; Track B labels handle that. | §8.2 |
| 16 | go-race and sql-pg at launch (minor) | **Fixed.** Q1(a): M3 is `go` only, consistent with D6's pilot after M3. | §6, §12 |

### 14.3 Draft claims corrected
| Claim | Correction | Source |
|---|---|---|
| "k3s picks systemd under systemd" [W6] | The rule is `!userns && cpuset && INVOCATION_ID != ""` (`config_linux.go` L71). It holds for this host **(inferred)**. | [W32] |
| Runner has "the highest `oom_score_adj`" | ≈ 968 vs v1's ≈ 998; the runner is only the first *eviction* victim | kubelet formula |
| Limits sum ≈ 15.5 Gi | ≈ 12.3 Gi | T0 |
| "A jail break cannot fabricate a hidden AC" | It can, via the public references; restated | §2.4 |
| "SIGSYS is only reachable through allowlisted imports" | `.s`/`.syso` files and compiler bugs reach it too; now RE | A18 |
| "A learner cannot move s1" | Indirectly, through the host throttle; breaker added | §7.4 |
| "runc 1.4.2 is past the escapes" | runc 1.4.3 / 1.5.x fix CVE-2026-41579 (low risk here) | [W24] |
| "ASLR on; TSAN fails with it off" (self-found) | TSAN turns ASLR off itself (LLVM ≥ 18.1); per-process policy decided by P3 | [W29][W30] |
| "Network: loopback only" (self-found) | Loopback **down**; SQL uses a Unix socket | §5.4 |
| "SQL write → REJECTED" (self-found) | Static pre-run classification only; runtime → RE | §6.3 |

---

### Appendix: sources

**Kept from the draft (all seen 2026-09-24):**
- [W1] gVisor compatibility, limits not enforced within the sandbox: https://gvisor.dev/docs/user_guide/compatibility/
- [W2] gVisor releases: https://github.com/google/gvisor/releases
- [W3] containerd #12182, "It does work with the systemd driver": https://github.com/containerd/containerd/issues/12182
- [W4] tkhq/valet PR #277: https://github.com/tkhq/valet/pull/277
- [W5] containerd CRI config: https://github.com/containerd/containerd/blob/main/docs/cri/config.md
- [W6] k3s #5454 — **superseded by [W32]**
- [W7] k3s v1.36.4+k3s1: https://github.com/k3s-io/k3s/releases/tag/v1.36.4+k3s1
- [W8] User namespaces GA:
  - https://kubernetes.io/blog/2026/04/23/kubernetes-v1-36-userns-ga/
  - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/127-user-namespaces
- [W9] ProcMount KEP-4265: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4265-proc-mount
- [W10] Kubernetes validation (`CAP_SYS_ADMIN` literal): https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation.go
- [W11] go-judge: https://github.com/criyle/go-judge
- [W12] go-sandbox: https://github.com/criyle/go-sandbox
- [W13] isolate NEWS: https://github.com/ioi/isolate/blob/master/NEWS
- [W14] nsjail: https://github.com/google/nsjail/releases
- [W15] Judge0 and its CVEs:
  - https://github.com/judge0/judge0/releases
  - https://tantosec.com/blog/judge0/
- [W16] DMOJ: https://github.com/DMOJ/judge-server/releases
- [W17] Go release history: https://go.dev/doc/devel/release
- [W18] PostgreSQL 18.6: https://www.postgresql.org/about/news/postgresql-186-1711-1615-1519-1424-and-19-beta-3-released-3365/
- [W19] Ubuntu userns restriction: https://ubuntu.com/blog/ubuntu-23-10-restricted-unprivileged-user-namespaces
- [W20] PSI (no longer load-bearing)

**New, verified 2026-09-24:**
- [W21] CVE-2026-74469, SCTP; noble "Vulnerable"; published 2026-08-15: https://ubuntu.com/security/CVE-2026-74469
- [W22] CrackArmor, userns-restriction bypass; noble fixed in 6.8.0-106.106: https://ubuntu.com/security/vulnerabilities/crackarmor
- [W23] CVE-2026-31431 "Copy Fail" (algif_aead); noble fixed in 6.8.0-117.117: https://ubuntu.com/security/CVE-2026-31431
- [W24] runc releases: v1.4.3 on 2026-06-13 and v1.5.1 on 2026-07-14, fixing CVE-2026-41579: https://github.com/opencontainers/runc/releases
- [W25] containerd v2.3.5, 2026-09-04: fixes CVE-2026-53495 (ExecSync exec-probe/hook DoS) and GHSA-rp3h-jf77-q9p4
  - https://github.com/containerd/containerd/releases/tag/v2.3.5
  - https://github.com/containerd/containerd/security/advisories/GHSA-7jxh-36q5-gcqv
- [W26] kubernetes #139916, open, fix PR #140190 open: https://github.com/kubernetes/kubernetes/issues/139916
- [W27] Kubernetes user-namespaces docs (the `kubelet` subuid range): https://kubernetes.io/docs/concepts/workloads/pods/user-namespaces/
- [W28] Hostinger CPU limit: 25% an hour; trigger duration not stated; updated 2026-09-15: https://www.hostinger.com/support/6899741-what-is-the-cpu-use-limit-for-vps-at-hostinger/
- [W29] LP #2056762, TSAN on amd64 with `mmap_rnd_bits=32`: https://bugs.launchpad.net/bugs/2056762
- [W30] TSAN in LLVM 18.1 re-executes with ASLR off: https://sysdev.me/2025/12/05/dealing-with-threadsanitizer-fails-on-startup/
- [W31] k3s releases: v1.36.4 on 2026-09-14; v1.36.5-rc1 on 2026-09-24: https://github.com/k3s-io/k3s/releases
- [W32] k3s v1.36.4 `pkg/agent/templates/templates.go` (L184, L242–246) and `pkg/agent/containerd/config_linux.go` (L71), read from local copies in `scratchpad/t3crit/`
- [W33] containerd `contrib/seccomp/seccomp_default.go` (the `CAP_SYS_ADMIN` branch; AF_ALG/AF_VSOCK blocked), read from `scratchpad/t3src/`

**Taken from the critiques without re-fetching:** the DiagSpill PoC date (heyitsas.im), lxc#4723, the lore regression link, the Go compiler CVEs (oss-sec 2026/q2/53), moby#42680, CVE-2025-5054, core(5), and the Coolify steal thread.

**Repo files read (read-only):**
- `docs/v2/feasibility.md`
- `docs/v2/research/t4-judge-contract.md` (§9, §10, §11.1, §13)
- `docs/v2/research/t2-object-storage-backups.md` (host-bootstrap, D12)
- `.github/workflows/deploy.yml`
- `../infra/clusters/vps/{infrastructure,apps,messaging}.yaml`
- `../infra/apps/image-automation.yaml`
- `../infra/charts/project/{values.yaml,templates/deployment.yaml}`
- `scratchpad/t3src/` and `scratchpad/t3crit/`

**Needs the owner:** 17 MCP servers in this session are waiting for authorization (GitHub, Slack, Notion, Linear and others). This research didn't need them. claude.ai connectors are authorized in their connector settings; other servers through `claude mcp` or `/mcp` in an interactive session.

---

## 15. S0 results (read-only production facts, 2026-09-24, owner-approved)

| Check | Result | Implication |
|---|---|---|
| runc cgroup driver | `SystemdCgroup = true` in the rendered `config.toml` | ✅ The systemd driver is in use, so containerd #12182 does not apply |
| Drop-in dir `config-v3.toml.d/` | **absent** (only `config.toml`, rendered 2026-09-16) | Created by the host sandbox block. Whether it merges is checked at spike P0 / A6 |
| runc / containerd | runc **1.4.2**, containerd **2.3.4-k3s1.36** | Matches the design; runc 1.4.3 and containerd 2.3.5 fixes arrive with k3s v1.36.5 (Track B) |
| CPU PSI | `some avg10≈2.0, avg300≈2.8` | Low contention today |
| Steal accounting | `CONFIG_PARAVIRT=y`, **`CONFIG_PARAVIRT_TIME_ACCOUNTING` not set** | **cgroup CPU time includes steal**, so CPU-time TLs inflate under steal. The steal signal (s1) and the quiet re-run (§5.7) are **mandatory** |
| Steal since boot | ≈ **2.4 %** of CPU time (`/proc/stat`) | A noisy neighbour exists. The **72 h `vmstat` sample** (running until ~2026-09-27 07:52 UTC, `/tmp/xlearn-s0-vmstat.log`) sets the thresholds |
| CPU | 4 × EPYC 9355P, 1 thread per core, **L3 64 MiB in 4 instances**, 1 NUMA node, AVX-512 | Each vCPU appears to have its own L3 slice, so cross-slot cache interference is lower (inferred) |
| `vm.mmap_rnd_bits` | **32** | The amd64 TSAN issue applies to go-race (§6.2); per-process ASLR policy |
| Sysctls | `io_uring_disabled=0` ✗ · `unprivileged_bpf_disabled=2` ✓ · `unprivileged_userfaultfd=0` ✓ · `perf_event_paranoid=4` ✓ · `dmesg_restrict=1` ✓ · `kptr_restrict=1` (→ 2) · **`fs.suid_dumpable=2` ✗** (with the apport pipe) · `apparmor_restrict_unprivileged_userns=1` ✓ | The sandbox block sets `io_uring_disabled=2` and `kptr_restrict=2`. **H0 sets `suid_dumpable=0` and `core_pattern=core`** |
| AppArmor features | `namespaces/userns_create` present | userns mediation is available, consistent with R1's "no `userns` rule" design |
| subuid / getsubids | no `kubelet` range; `getsubids` absent | Added by the host sandbox block |
| Denylist modules | **none loaded** | The module denylist is safe to apply |
| k3s invocation | systemd unit `k3s server --write-kubeconfig-mode 0644`; no `config.yaml` | Matches T2's host-bootstrap assumptions |
| Kernel (verified separately) | running **6.8.0-90**; noble candidate **6.8.0-142**; **`linux-image-generic` meta not installed**, so the kernel never auto-updates; Livepatch off; `core_pattern` pipes to apport as host root | **H0 is urgent for v1 today.** It is a separate task (patch the kernel, install the meta-package, fix core dumps) |

---

## 16. Spike results (MI-10)

### 16.1 P0–P2 (spk-01, arm64 multipass)

Run 2026-09-25 on a throwaway `multipass` Ubuntu 24.04 **arm64** VM `xl-spike` on the owner's Mac
(D41: spikes first; the D23 go-ahead is the launch, D40). Everything below is thrown away except this
table; the VM, harness, corpus and manifests are never committed. **No timing conclusions (arm64 / HVF).**

**Environment**

| Component | Version | Production (for contrast) |
|---|---|---|
| Kernel | `6.8.0-142-generic` aarch64 | `6.8.0-142-generic` amd64 (MI-0 done) |
| k3s | `v1.36.4+k3s1` | same (pinned) |
| containerd | `2.3.4-k3s1.36` | `2.3.4-k3s1.36` |
| runc | `1.4.2` (judge == default runtime) | `1.4.2` |
| go | `go1.26.8` (checksummed tarball) | — |
| PostgreSQL | PGDG `18.6` | — |
| g++ / python3 | `13.3.0` / `3.12.3` | — |
| AppArmor parser | `4.0.1` | — |
| **`github.com/criyle/go-sandbox`** | **`v0.13.7`**, commit **`6a60e40be9d0cefb656c4ae12415c5fd040df954`** (tag `v0.13.7`) | m3-03 pins go-sandbox at exactly this version |
| nsjail (R1-N) | not used — forkexec passed | — |

`apparmor_restrict_unprivileged_userns=1`, cgroup2 (`cgroup2fs`), `/dev/kvm` absent, `vm.mmap_rnd_bits=33`.

**In-pod channel: CRI exec** (`k3s crictl exec <id> /opt/spike/sup …`). `kubectl exec` is denied by the VAP
by design; CRI bypasses API admission but runc still applies the container's seccomp, AppArmor, caps and
user namespace to the exec'd process, so the checks stay representative. The VAP was **never relaxed to
obtain exec**.

**Per-phase results**

| Phase | Step | Measure | Number | Pass | errno / note |
|---|---|---|---|---|---|
| P0 | drop-in `judge` handler | `crictl info` lists `judge`; `cgroupWritable:true`; `SystemdCgroup:true` | yes | ✅ | drop-in merges verbatim; **no `.tmpl` fallback needed** |
| P0 | rendered runc section | `SystemdCgroup = true` still rendered; judge==default runc `1.4.2`; no `BinaryName` | yes | ✅ | — |
| P0 | k3s restart with a pod running | pod survives (same container id, restartCount 0) | yes | ✅ | restart ~6 s |
| P0 | VAP dry-run corpus (unmodified VAP) | G1 admit; B1–B14(+variants) deny; Q1 LimitRange; C1(default) admit | 24/24 | ✅ | see VAP diff below |
| P1 | positive pod Running | `hostUsers:false`; AppArmor label `xlearn-runner`; uid_map `0 1073807360 65536` (non-identity, in the subuid range) | yes | ✅ | — |
| P1 | `mkdir /sys/fs/cgroup/slots` + `+cpu +memory +pids` | slots controllers `cpu memory pids` | yes | ✅ | cgroupWritable delegation works |
| P1 | forkexec jail **without CLONE_NEWUSER** (mnt/pid/net/ipc/uts/cgroup) | tmpfs root, nosuid binds, `pivot_root`, procfs `hidepid=invisible,subset=pid`, lo down | hello OK | ✅ | **needs `pivot_root` in seccomp — see below** |
| P1 | spawn via `CLONE_INTO_CGROUP` | jail OK, accounted in case cgroup | OK | ✅ | clone3 + cgroup fd |
| P1 | spawn via `cgroup.procs` write + sync pipe | jail OK (kernel 6.8) | OK | ✅ | both paths work on 6.8 |
| P1 | caps → 0 in the child | `CapEff/CapPrm/CapInh = 0`, `NoNewPrivs=1`; privileged op → EPERM | yes | ✅ | **`CapBnd` stays `0x2001e0` — see SETPCAP** |
| P1 | `go build` in the jail (+ run) | compiled and ran | OK | ✅ | GOCACHE on the `/work` tmpfs |
| P1 | D20 smoke: `g++ -std=gnu++20 -O2` + run; `python3` run | both ran | OK | ✅ | functional only |
| P1 | negative probes from the supervisor | `unshare -U`, `clone3(NEWUSER)`, `fsopen`, `open_tree`, `mount`(outside jail), SCTP, NETLINK, PACKET, raw-inet, ptrace(pid1), keyctl, add_key, userfaultfd, io_uring_setup, bpf, perf_event_open, setns(pid1) | 17/17 denied | ✅ | userns EINVAL(22); clone3(NEWUSER)/mount EACCES(13); rest EPERM(1) |
| P1 | SIGSYS from the jail on a disallowed syscall | signal `SIGSYS`; `core_pattern` unchanged (`core`, no `|`); no host helper | yes | ✅ | `RLIMIT_CORE=0` |
| P1 | procfs `hidepid=invisible,subset=pid` | jail sees only its own pids (`[1]`, self is pid 1 in the fresh pidns) | yes | ✅ | host pid1/others not visible (fresh pidns) |
| P1 | X1/X2 real CONNECT `kubectl exec`/`attach` | denied by `xlearn-runner-no-exec` | denied | ✅ | ran against the widened VAP copy (see below) |
| P1 | E1 real `kubectl debug` ephemeral container | denied by `xlearn-runner-pod-shape` (R10) | denied | ✅ | report to mi-14 as a corpus row |
| P1 | delete/recreate the pod × 50 | Running | **50/50** | ✅ | bounded `kubelet` subuid range holds (#139916) |
| P1b | `go test -c -race` fixture in the jail × 20 | RACE detected | **20/20** | ✅ | glibc `clone3`→ENOSYS→`clone` fallback; threads work |
| P1b | deadlock fixture in the jail × 20 | classified (timeout/abort) | **20/20** | ✅ | — |
| P1b | postgres in the jail (unix socket, `listen_addresses=''`, uid 999) | `select 42` returned | OK | ✅ | postmaster boots, query runs |
| P2 | balloon (1 GiB), case `memory.max` 256Mi, `oom.group=1` | MLE (case `oom_kill`) | ✅ | — | killed by SIGKILL; oom_group_kill=1 |
| P2 | fork bomb, `pids.max` 32 | RE(pids) / EAGAIN | ✅ | — | `fork: resource temporarily unavailable` at 27 |
| P2 | thread bomb, `pids.max` 64 | RE(pids) | ✅ | — | go runtime `newosproc errno=11` |
| P2 | tmpfs fill (256Mi) / inode fill (8192) | ENOSPC, bounded | ✅ | — | `no space left on device` |
| P2 | stdout flood / orphan double-fork / sleep / spin | bounded, cleaned up | ✅ | — | orphan killed by `cgroup.kill` |
| P2 | **MLE classification × 100 (balloon)** | 100/100 | **100/100** | ✅ | SQL balloon variant deferred to spk-02 (image-volume harness); classic balloon 100/100 |
| P2 | **container-level OOM** | `memory.events.local oom_kill` (non-hierarchical) + pod `restartCount` | **0 / 0** | ✅ | hierarchical `memory.events oom_kill=202` = the sum of case-cgroup OOMs, i.e. every OOM landed in a case cgroup (INV-14) |
| P2 | survivors after each job / leftover case cgroups | 0 / 0 | **0** | ✅ | — |
| P2 | `slots` memory back to baseline | +2.0 MiB | ✅ (±5 MiB) | — | — |
| P2 | `nr_dying_descendants` after 1,000 case cgroups | 50 at 1 s → **0 at 6 s** | ✅ (< 60 s) | — | 1,000 jails in 1.49 s |

**Q-A (R1 viability): GO.** The `judge` drop-in, `hostUsers:false` + namespaced caps, Localhost AppArmor
(no `userns` rule) + Localhost seccomp, per-case cgroups, and a **userns-less `forkexec.Runner` jail** all
work on this stack, and user-namespace creation and the new mount API are denied to the supervisor.

**Q-B (INV-14): GO.** Every learner OOM lands in a case cgroup, with **0 container-level OOMs** and
`restartCount 0`; cleanup converges (`nr_dying_descendants`→0 in 6 s, baseline +2 MiB, 0 survivors).

**Jail mechanism chosen: go-sandbox `forkexec.Runner`** (the low-level Runner, not the `container`
builder), no user namespace, spawned into the case cgroup via `CLONE_INTO_CGROUP`. **nsjail (R1-N) was not
needed.** No fallback past R1-N; no owner decision required.

**Open points answered**

- **`pivot_root` is NOT in containerd's RuntimeDefault allowlist** (nor its `CAP_SYS_ADMIN` block — only
  `chroot`, gated on `CAP_SYS_CHROOT`, which the runner does not hold). Without adding it the jail's
  `pivot_root` returns **EPERM** and R1 fails. **The runner seccomp profile must add `pivot_root`.** This
  is the single most important host-file delta from t3 §8.7; it reproduced identically under both
  `hostUsers:false` and `hostUsers:true`, and was not an AppArmor or userns effect (complain mode still
  EPERM'd). mi-09 and spk-02 must carry it.
- **`SETPCAP`:** go-sandbox's `DropCaps` zeroes effective/permitted/inheritable (which needs **no**
  `SETPCAP`); it does **not** drop the bounding set (`CapBnd` stays `0x2001e0` in the child). Dropping the
  bounding set would need `CAP_SETPCAP` (via `PR_CAPBSET_DROP`), which go-sandbox does not call. So
  `SETPCAP` is currently **unused** by the jail; `NoNewPrivs=1` plus `CapEff=0` already prevent regaining
  privilege. Keep `SETPCAP` in the set only if a future bounding-set drop is wanted (defense in depth);
  otherwise it can be removed. Recorded for m3-03/mi-14.
- **`CLONE_INTO_CGROUP` vs `cgroup.procs`:** both work on kernel 6.8. `CLONE_INTO_CGROUP` (clone3 with the
  case cgroup fd) is clean; the `cgroup.procs` write + sync-pipe path also works. Prefer `CLONE_INTO_CGROUP`.
- **Pod survival on k3s restart:** yes (same container id, `restartCount 0`).
- **subuid range:** `kubelet:1073741824:7208960` (110 × 65 536, ends 1 080 950 783, well below UINT32_MAX).
  The × 50 recreate loop was **50/50 Running**; the pod's uid_map is `0 1073807360 65536` (non-identity,
  inside the range).
- **`getsubids`:** shipped by noble's **`uidmap`** package.
- **overlayfs mount under `hostUsers:false` returns EPERM.** An overlay assembled in the pod on the
  idmapped `/work`/`/jail` tmpfs (lower+upper+work) fails with EACCES/EPERM(13). The GOCACHE-seed overlay
  therefore cannot be an in-pod overlay mount; the seed must be delivered another way (baked into the
  image, or a plain read-only bind + a writable tmpfs GOCACHE). The `GOCACHE` mechanism is an A8 item
  anyway; flagged for spk-02 (amd64 replay) and p-01.

**Final host files**

*containerd drop-in* `/var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d/20-judge.toml` (verbatim,
merges cleanly — no `.tmpl` fallback):

```toml
version = 3
[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.judge]
  runtime_type = "io.containerd.runc.v2"
  cgroup_writable = true
[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.judge.options]
  SystemdCgroup = true
```

*Seccomp* `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json` (arm64; **`SCMP_ARCH_ARM,SCMP_ARCH_AARCH64`**;
`defaultAction SCMP_ACT_ERRNO`; 385 syscall names). Generated from **containerd v2.3.4 `DefaultProfile`**
with the 5 runner caps, then:
- **removed** (from RuntimeDefault): `fsopen fsconfig fsmount fspick move_mount open_tree mount_setattr`,
  `bpf perf_event_open fanotify_init fanotify_mark`, `lookup_dcookie syslog`, and containerd's broad
  `socket` rule;
- `socket()` restricted to **AF_UNIX**, and **AF_INET/AF_INET6 `SOCK_STREAM`** with protocol 0 or TCP
  (no SCTP/NETLINK/PACKET/ALG/VSOCK/raw);
- **added `pivot_root`** (the §8.7 delta above — RuntimeDefault omits it);
- `keyctl add_key request_key io_uring_* userfaultfd` are not in RuntimeDefault at all, so they fall to the
  default `ERRNO` (EPERM) with no explicit rule — consistent with §8.7's intent.
- **The arm64 profile above is NOT the one mi-09 ships:** spk-02 replays and regenerates it on amd64. The
  `pivot_root` addition and the socket restriction carry over unchanged.

*AppArmor* `/etc/apparmor.d/xlearn-runner` (final spike copy; abi 4.0, **no `userns` rule**):

```
abi <abi/4.0>,
include <tunables/global>
profile xlearn-runner flags=(attach_disconnected,mediate_deleted) {
  include <abstractions/base>
  capability sys_admin, setuid, setgid, setpcap, kill,
  network unix, network inet stream, network inet6 stream, unix,
  file,
  signal (receive) peer=unconfined, signal (receive) peer=runc,
  signal (send,receive) peer=xlearn-runner,
  mount fstype=tmpfs -> /jail/**,
  mount fstype=overlay -> /jail/**,
  mount fstype=proc -> /jail/**,
  mount options=(rw, rbind, nosuid, rprivate) -> /jail/**,
  mount options=(rw, rprivate) -> /,
  mount options=(rw, rslave) -> /,
  remount,           # a remount only re-flags an already path-scoped mount
  pivot_root,
  umount,
  deny ptrace,
  deny mount fstype=sysfs, deny mount fstype=cgroup, deny mount fstype=cgroup2,
  deny @{PROC}/sysrq-trigger rwklx, deny @{PROC}/kcore rwklx,
  deny /sys/firmware/** rwklx, deny /sys/kernel/security/** rwklx,
}
```

**AppArmor caveat for mi-09/spk-02.** go-sandbox's read-only bind uses a recursive-bind **remount**
(`ro, nosuid, remount, rbind, rprivate`) that AppArmor 4.0 mount-flag matching rejects ("failed flags
match"), even with an exact `options=(…)` rule. For the spike the toolchain binds were made **rw** (they
sit on the container's **read-only** rootfs, so they stay read-only; `nosuid` is kept via the bind flag and
`NoNewPrivs` neutralises setuid), and `remount` is allowed broadly (a remount cannot create a new mount).
The exact ro-bind-remount rule (or making the binds ro another way) must be finalised in spk-02's amd64
replay before mi-09 ships. **Resolved in spk-02 re-run block 3 (§16.2):** read-only binds made without
`MS_PRIVATE` match an exact, `ro`-only remount rule scoped to `/jail/**`, and the broad `remount,` is gone. Fresh-mount **path confinement to `/jail` is enforced** and proven: the
`mount_tmpfs_outside` probe (a fresh tmpfs at `/mnt`) is denied EACCES.

**VAP diff (against the t3 §8.2 draft; mi-14 had not run — corpus and guards written here)**

- The spike wrote the guard objects and a 24-case corpus (`G1`, `B1–B14` + b-variants, `Q1`, `C1`) in
  `~/xl-spike/`. The bindings were applied at **`[Deny]`** on the VM only. Result: G1 admitted (one PSA
  `baseline` warning for `SYS_ADMIN`; `procMount:Unmasked` drew no PSA warning — the
  `UserNamespacesPodSecurityStandards` relaxation applies), C1(default) admitted, Q1 denied by the
  LimitRange, all B-shapes denied by `xlearn-runner-pod-shape`. Both VAPs' `status.typeChecking` empty.
- **Corpus/order deltas to fold into mi-14** (kube-apiserver rejects some single-violation shapes *before*
  admission, so those bad shapes never reach the VAP; the corpus needs companion tweaks and rule order):
  - `privileged:true` is rejected by API validation when `allowPrivilegeEscalation:false` — B1 must also
    set `allowPrivilegeEscalation:true` to reach the VAP.
  - `hostPID:true` is rejected by API validation when `hostUsers:false`; and `procMount:Unmasked` is
    rejected when `hostUsers` is not false — B2 must set `hostUsers:true`+`procMount:Default`, and B9/B9b
    must set `procMount:Default`.
  - Put the `privileged` and host-namespace rules **before** the `hostUsers`/`allowPrivilegeEscalation`
    rules so the denial names the intended rule.
  - PSA warnings carry `"baseline:latest"` — a classifier that stops at the first `:` misses them.
- **E1 (extra corpus row for mi-14):** a real `kubectl debug` ephemeral-container attempt (no `-it`) on the
  positive pod is denied by the pod-shape VAP (R10 / `pods/ephemeralcontainers`). A pod-manifest dry-run
  can't test this; add it to the corpus.
- **Image rule / CRI digest:** CRI could not resolve the imported image by digest (an image imported via
  `ctr images import` has **no RepoDigests**), so `imagePullPolicy: Never` with `@sha256:` gave
  `ErrImageNeverPull`. The **dry-run set (G1, B1–B14, Q1, C1) ran against the unmodified VAP** (image rule
  `…xlearn-runner@sha256:…`). The **positive pod, X1/X2 and E1** ran against a spike-only VAP copy
  (`xlearn-runner-pod-shape-spike`) that differs **only** in the image rule (a tag ref allowed); the
  `xlearn-runner-no-exec` VAP was unchanged, so the CONNECT proof stands. On production the image is pulled
  by digest from GHCR (has a RepoDigest), so the unmodified rule is correct there — no change to mi-14's
  regex is required for the real runner.

**No timing conclusions (arm64 / HVF).**

### 16.2 P3 amd64 replay (spk-02)

Run 2026-09-25 on **environment A**, the owner's spare `skriptvalley-vps` (D41; launch = go-ahead, D40).
**The first session was interrupted after the jail-setup and probe rows.** Under D40 the partial results land here and
every row not run is ⛔. Everything else is thrown away: the harness, image, registry and k3s were removed at
teardown, and nothing was committed. **No timing conclusions** (1-vCPU KVM guest, not the production guest).

**Re-run (second session, 2026-09-25), in three blocks, each landed as its own results PR.** The same env A was set
up again from its baseline: kernel `6.8.0-142`, `mmap_rnd_bits=32`, k3s `v1.36.4+k3s1 --disable traefik`, the
`judge` drop-in, spk-01's host files, go1.26.8 (checksummed), the private registry, and the same generator (the
four pod-profile variants hash identically to the first session's). Block 1 is the image volume (§16.3) plus the
GOCACHE seed (below) ✅. Block 2 (TSAN, PG, allowlists, KILL, architectures, below) ✅. Block 3 (AppArmor
remount narrowing, below) ✅. Env A was then torn down to its baseline (status.md spike record).

**Environment**

| Component | Value | Production (for contrast) |
|---|---|---|
| Host | Hostinger KVM 1: 1 vCPU AMD EPYC 9354P, 3.9 GB RAM, KVM guest | 4 × EPYC 9355P |
| Kernel | `6.8.0-142-generic` x86_64 (`apt full-upgrade` from 6.8.0-90, then a reboot) | `6.8.0-142-generic` (MI-0) |
| `vm.mmap_rnd_bits` | **32** (left as is, never lowered) | 32 |
| `apparmor_restrict_unprivileged_userns` | 1 | 1 |
| H0 mirror + sandbox sysctls | `core_pattern=core`, `suid_dumpable=0`, apport masked; the §8.7 sysctls | H0 done; sandbox block at the host window |
| k3s / containerd / runc | `v1.36.4+k3s1` (`--disable traefik`) / `2.3.4-k3s1.36` / `1.4.2` | same |
| `judge` drop-in | merged verbatim (`crictl info`: `cgroupWritable:true`, `SystemdCgroup:true`) | — |
| Toolchain | go1.26.8 (checksummed); runner image `debian:trixie-slim` + g++ + python3 + PGDG PG 18, pulled by the kubelet | — |

**Path used:** the supervisor ran inside spk-01's positive pod shape on the pinned k3s (the "closest to
production" option): `hostUsers:false`, 5 caps, Localhost AppArmor + Localhost seccomp, `runtimeClassName`
→ `judge`, driven over CRI exec. The pod's uid_map was `0 1074397184 65536` (non-identity, inside the
`kubelet` subuid range). The guard VAPs were not re-applied, because P0 was already proven in §16.1.

**The amd64 pod seccomp profile (generated; validation partial).** It was built with §16.1's recipe from
containerd **v2.3.4 `DefaultProfile`** on amd64 with the 5 runner caps:
- the same 14 removals (the new mount API, `bpf`, `perf_event_open`, `fanotify_*`, `lookup_dcookie`, `syslog`, and containerd's broad `socket` rule);
- `socket()` limited to AF_UNIX and AF_INET/AF_INET6 `SOCK_STREAM` (proto 0/TCP);
- `pivot_root` **added**.

`architectures: [SCMP_ARCH_X86_64, SCMP_ARCH_X86, SCMP_ARCH_X32]`, `defaultAction: SCMP_ACT_ERRNO`, **381 names**,
sha256 `40412b25…a5ca570`. Against the arm64 file (385 names) it adds `arch_prctl` and `modify_ldt`, and drops
`arm_fadvise64_64`, `arm_sync_file_range`, `breakpoint`, `cacheflush`, `set_tls` and `sync_file_range2`. The
x86_64-only variant (sha256 `8817e742…e22445`) and a variant with an exact-argument
`personality(0x0040000)` rule were generated but not run in the first session. Re-run block 2 ran the x86_64-only
variant, found the `personality` variant **unnecessary** (TSAN needs no ASLR change), and records the final JSON
verbatim (see "Re-run block 2" below).

**Jail setup on amd64 (three-arch profile, spk-01's AppArmor profile)**

| Step | Result | Pass |
|---|---|---|
| jail via `CLONE_INTO_CGROUP` / via `cgroup.procs` + sync | OK / OK | ✅ |
| caps in the child | `CapEff/CapPrm/CapInh = 0`, `NoNewPrivs=1`, `CapBnd 0x2001e0`; a mount after the drop → EACCES(13) | ✅ |
| procfs `hidepid=invisible,subset=pid` | only the jail's own pid visible | ✅ |
| `go build` / `g++ -std=gnu++20 -O2 -static` / `python3` in the jail | ran | ✅ (functional only) |
| in-pod overlayfs under `hostUsers:false` | **EACCES(13)**, the same as arm64 | ✅ confirms the GOCACHE seed can't be an in-pod overlay |
| a disallowed syscall in the jail | `SIGSYS`; `core_pattern` unchanged (`core`) | ✅ |

**Negative probes from the supervisor (three-arch profile, spk-01's AppArmor profile)**

| Probe | Result | Pass |
|---|---|---|
| spk-01's 17 (`unshare -U`, `clone3(NEWUSER)`, `fsopen`, `open_tree`, `mount` outside `/jail`, SCTP, NETLINK, PACKET, raw inet, `ptrace(1)`, `keyctl`, `add_key`, `userfaultfd`, `io_uring_setup`, `bpf`, `perf_event_open`, `setns`) | all denied with the same errnos as arm64 | ✅ 17/17 |
| `move_mount` / `mount_setattr` | EPERM / EPERM | ✅ |
| x32-ABI `getpid` | ENOSYS(38) | ✅ |
| ia32-ABI `getpid` (`int $0x80`) | **allowed** (returned the pid) | expected under the three-arch baseline. It is the evidence for the x86_64-only proposal, which closes it (block 2 below) |
| **remount of `/` read-write, remount of `/sys` read-write** | **unexpected success** | ❌ **finding.** spk-01's broad `remount,` AppArmor rule lets the supervisor context remount the container's read-only `/` and `/sys` read-write. The probe restored `/` to read-only. The line was stopped here, as the probe rules require, and not investigated further |
| `personality(ADDR_NO_RANDOMIZE)` (information only) | EPERM: RuntimeDefault's `personality` rule allows only a fixed set of argument values, and this isn't one of them. The `0xffffffff` query is allowed | if go-race needs ASLR off, the pod profile needs the exact-argument rule |

**Consequence of the finding (mi-09, before the host window).** The broad `remount,` rule **must not ship.**
It must be replaced by remount rules scoped to the jail tree, plus the jail's own read-only root remount after
`pivot_root`. Both remount probes join the must-deny set, and the jail setup has to be re-proven with the
scoped rules. This is also the ro-bind-remount rule that §16.1 left open. **Resolved in re-run block 3 (below):**
two `ro`-only remount rules replace `remount,`, and the read-write remounts of `/`, `/sys` and `/jail/**` are now refused
with EACCES.

**Rows not run in the first session (⛔ until the re-run blocks land).** None of these count as GO:
- ~~TSAN / go-race under `mmap_rnd_bits=32` and its ASLR policy~~ → done in block 2 (below);
- ~~postgres in the jail on amd64, and the SQL balloon~~ → done in block 2;
- ~~the RET_LOG allowlists for `go`, `cpp`, `python` and `go-race` (through auditd), and the compile-jail sets~~ → done in block 2;
- ~~the KILL re-run~~ → done in block 2;
- ~~the x86_64-only pod-profile variant~~ → done in block 2;
- ~~the scoped AppArmor remount rule~~ → done in block 3 (below);
- ~~the GOCACHE seed without an overlay~~ → done in block 1 (below).

Prepared for the re-run, throwaway and outside every repo: 22 reference programs per language (Go, C++,
Python; synthetic inputs). Their outputs were cross-checked natively against the Python references: Go 22/22,
C++ 22/22.

**GOCACHE seed without an overlay (re-run block 1) ✅.** The in-pod overlay is out (EACCES under
`hostUsers:false`, §16.1 and above), so the seed was delivered read-only instead. The seed is the runner image's
`/opt/gocache`: a GOCACHE filled at image-build time by building every Go reference with the exec profile's flags
(`CGO_ENABLED=0`, `-trimpath`, go1.26.8). It is **34 MiB in 439 files**. It was delivered two ways: baked into the
runner image layer, and as a separate `FROM scratch` image mounted as a read-only **image volume** at
`/opt/gocache-iv`. The image volume mounted fine in the `hostUsers:false` runner pod on the `judge` runtime. The jail
bound the seed read-only at `/seed` and ran `go build -trimpath` of one reference (`15_topo`), with `TMPDIR` on the
case tmpfs, once unchanged and once with a learner edit to `main.go` (its package is **not** in the seed). All 12
builds succeeded. Numbers are from the second of two rounds (1-vCPU spare VPS: **relative only**; A8 re-measures on
production):

| Seed delivery (GOCACHE) | Build, unchanged | Build, learner-edited | Case tmpfs used | Case memory peak |
|---|---|---|---|---|
| cold: empty tmpfs GOCACHE | 12.7 s | 11.9 s | 30 MiB | 280–289 MiB |
| image-layer seed, `cp -a` → tmpfs | 0.57 s | 0.44 s | 37 MiB | 71 MiB |
| image-volume seed, `cp -a` → tmpfs | 0.39 s | 0.43 s | 37 MiB | 70–71 MiB |
| image-volume seed, symlink farm (`cp -rs`) → tmpfs | 0.37 s | 0.36 s | < 1 MiB (439 symlinks) | 34–35 MiB |
| **image-volume seed read-only in place** (`GOCACHE=/seed`) | **0.31 s** | **0.36 s** | 0 | 33–34 MiB |
| read-only seed + `GOFLAGS=-trimpath` | 0.29 s | 0.33 s | 0 | 34–35 MiB |

- **Recommendation (p-01, m3-04):** point `GOCACHE` at the read-only seed **in place** and keep `TMPDIR` on the case
  tmpfs. The go command reads the dependencies' compiled packages from the read-only cache (the `-x` link step's
  `packagefile` lines point into the seed) and silently ignores its failed cache writes for the learner's own
  package. The compile drops from ~12 s to ~0.35 s with **no per-case copy** and about 250 MiB less peak memory
  per case. `TMPDIR` must be writable: with it on the read-only root, the build fails with
  `go: creating work dir: mkdir /tmp/go-build…: read-only file system`.
- The seed only hits if it was built by the **same toolchain with the same flags** (`CGO_ENABLED`, `-trimpath`,
  GOOS/GOARCH). A mismatch just means cache misses (cold speed), never a wrong build. Build it in the runner-image
  pipeline, either baked into the image or as a sibling image volume. Both measured the same; the image volume lets
  the seed update without rebuilding the runner.

**Re-run block 2 (2026-09-25): TSAN, postgres, amd64 allowlists, KILL re-run, pod-profile architectures ✅.**
Everything ran inside spk-01's positive pod shape on env A: `hostUsers:false`, the 5 caps, the `judge` runtime,
Localhost AppArmor (spk-01's profile, still with the broad `remount,` rule that block 3 replaces) and a Localhost pod
seccomp profile. The work was driven over CRI exec, with the same supervisor cross-built by go1.26.8. Two pods
were used: one on the three-arch profile, and one on the x86_64-only variant. Host `vm.mmap_rnd_bits` stayed
**32** throughout, and nothing lowered it.

*go-race / TSAN (Q-C) ✅: no ASLR policy needed.*
- The fixtures were P1b's `go test -c -race` binary: `TestRace`, `TestNoRace`, `TestChannels`, `TestContext` and
  `TestDeadlock`. It is a non-PIE `EXEC`, dynamically linked to glibc; glibc landed at randomized addresses near
  `0x7abe…`.
- Every run passed: the race was detected (exit 66), the deadlock was classified (exit 2, test timeout), and the
  clean tests passed.
  - Three-arch pod, no exec filter: 20 rounds × 5 = **100/100**.
  - Three-arch pod, go-race KILL filter: **25/25**.
  - x86_64-only pod, go-race KILL filter: **125/125** (5 + 20 rounds).
- **TSAN never called `personality`.** In the RET_LOG run of the go-race profile, where the pod profile allows the
  `0xffffffff` query, no `personality` record appeared at all. The pod profile keeps RuntimeDefault's `personality`
  rule, which denies `ADDR_NO_RANDOMIZE` (§16.2 probes). The generated `personality(0x0040000)` variants are
  **not needed** and don't ship; `setarch -R` isn't needed either.
- go-race is **in** the pilot as far as the sandbox goes (p-01).

*postgres in the jail (amd64) ✅.*
- PG 18 ran as uid 999 on a Unix socket, with `listen_addresses=''`, and returned `select 42`.
- The SQL balloon (`array_agg` over 60 M rows) ran with its backend moved into a case cgroup (`memory.max`
  128 MiB, `oom.group=1`). The backend was OOM-killed there (`oom_kill 2`, `oom_group_kill 1`) → **MLE**.
- The postmaster recovered, and `select 43` succeeded on the 2nd try.
- The container's own `memory.events.local` stayed at `oom_kill 0`.
- The result was the same on both pods.

*Allowlist method.*
- Each profile's exec jail ran with a classic-BPF filter whose default action was `SECCOMP_RET_LOG`. Records were
  collected through **auditd**: `auditctl -b 8192 -r 0`, then the `type=SECCOMP` records were parsed from
  `/var/log/audit/audit.log` and mapped with `ausyscall x86_64`.
- It ran in two passes, to keep the flood small:
  1. one program with an empty allowlist, so every syscall is logged;
  2. every program with pass 1's names allowed, so only syscalls not yet seen are logged.

  The result is the union.
- `auditctl -s` showed **`lost 0`** after every run.
- The inputs were the 22 references per language (3 rounds in the KILL re-run), plus P1b's race fixtures for
  go-race. The P2 corpus ran separately under the `go` profile.
- The filter's fixed rules, which m3-04 should keep:
  - a non-x86_64 arch, or an x32 number (`nr ≥ 0x40000000`) → `KILL_PROCESS`;
  - `clone3` → `ERRNO(ENOSYS)`, so glibc falls back to `clone`;
  - **`clone` only with `CLONE_THREAD` set**: threads yes, `fork` no;
  - `prctl` only with `PR_SET_VMA` (where listed);
  - `personality` in **no** profile.

**Per-profile amd64 exec allowlists.** Kernel names, sorted; `ausyscall` prints `pread64` as `pread`.
**0 unexpected SIGSYS** under KILL.

| Profile | Size | Allowlist |
|---|---|---|
| `go` (static, `CGO_ENABLED=0`) | 27 | `arch_prctl clone`† `epoll_create1`‡ `epoll_ctl`‡ `epoll_pwait`‡ `eventfd2`‡ `execve exit_group fcntl futex getpid gettid madvise mmap nanosleep openat prctl`§ `prlimit64 read rt_sigaction rt_sigprocmask rt_sigreturn sched_getaffinity sched_yield sigaltstack tgkill write` |
| `cpp` (`g++ -std=gnu++20 -O2 -static`) | 19 | `arch_prctl brk execve exit_group fstat futex getrandom lseek mmap mprotect munmap prlimit64 read readlinkat rseq set_robust_list set_tid_address write writev` |
| `python` (`python3` 3.13, dynamic) | 39 | `access arch_prctl brk clone`† `close execve exit exit_group fcntl fstat futex getcwd`¶ `getdents64 getegid geteuid getgid getrandom gettid getuid ioctl lseek madvise mmap mprotect mremap munmap newfstatat open openat pread64 prlimit64 read readlink rseq rt_sigaction rt_sigprocmask set_robust_list set_tid_address write` |
| `go-race` (`go test -c -race`, dynamic) | 40 | `access arch_prctl brk clone`† `close epoll_create1 epoll_ctl epoll_pwait eventfd2 execve exit_group fcntl fstat futex getpid getrandom gettid gettimeofday madvise mmap mprotect munmap nanosleep openat prctl`§ `pread64 prlimit64 read readlinkat rseq rt_sigaction rt_sigprocmask rt_sigreturn sched_getaffinity sched_yield set_robust_list set_tid_address sigaltstack tgkill write` |

Justified additions beyond what the references logged:
- † `clone` is allowed only with `CLONE_THREAD`. Python's threading (a deep-recursion reference runs in a thread)
  and the Go and TSAN runtimes all pass it. A plain `os.fork()` in Python then dies with SIGSYS; it succeeded before
  the argument check was added.
- ‡ The `go` profile gets `epoll_create1`, `epoll_ctl`, `epoll_pwait` and `eventfd2`. The Go runtime's netpoller
  backs every timer (`time.Sleep`, `time.After`, context deadlines): it was seen in go-race's `TestContext` and the
  corpus `sleep` row. The DSA references have no timers, but the go-concurrency pilot needs them. They create only
  process-local fds.
- § `prctl` is allowed with **arg0 = `PR_SET_VMA`** only. Go ≥ 1.25 names its anonymous mappings when the main
  module declares `go ≥ 1.25`. The references were built in GOPATH mode and never called it, but a module-mode
  binary (`go 1.26`) SIGSYS'd at `prctl` without this rule. It only labels the process's own mappings.
- ¶ Python gets `getcwd` (seen under `python3 -c`). It is read-only and reveals `/work`.
- Never allowlisted, and never seen in any LOG set: the new mount API, `bpf`, `io_uring_*`, `userfaultfd`, the
  keyring calls, `socket`, `ptrace` and `personality`.
- For m3-04: `python` needs `ioctl` (`TCGETS` on stdin) and legacy `open`. Consider narrowing `ioctl` to `TCGETS`.

**Compile-jail sets** (informational: compile seccomp is ENOSYS-default, t3 §6.2; all builds cold):
- `go build` (54): `arch_prctl chdir clone close copy_file_range dup3 epoll_create1 epoll_ctl epoll_pwait eventfd2 execve exit_group faccessat2 fallocate fchmodat fcntl flock fstat ftruncate futex getcwd getdents64 getpid gettid lseek madvise mkdirat mmap munmap nanosleep newfstatat openat pidfd_open pidfd_send_signal pipe2 prctl pread64 prlimit64 pwrite64 read readlinkat renameat rt_sigaction rt_sigprocmask rt_sigreturn sched_getaffinity sched_yield sigaltstack tgkill uname unlinkat utimensat waitid write`
- `g++ -static` (38): `access arch_prctl brk chmod clone close dup execve exit_group faccessat2 fcntl fstat futex getcwd getrandom getrusage ioctl lseek mmap mprotect mremap munmap newfstatat openat pread64 prlimit64 read readlink rseq rt_sigaction rt_sigprocmask set_robust_list set_tid_address sysinfo umask unlink wait4 write`
- `python3 -m py_compile` (38): `access arch_prctl brk close execve exit_group fcntl fstat futex getcwd getdents64 getegid geteuid getgid getrandom gettid getuid ioctl lseek mkdir mmap mprotect mremap munmap newfstatat open openat pread64 prlimit64 read readlink rename rseq rt_sigaction rt_sigprocmask set_robust_list set_tid_address write`
- `go test -c -race` (78, cgo): the `go build` set plus `brk chmod dup fadvise64 getegid geteuid getgid getppid getrandom getrusage getuid ioctl mprotect readlink rseq set_robust_list set_tid_address statfs sysinfo umask unlink vfork wait4`, all logged.
- The compile jail's process spawning (`posix_spawn` → `clone(CLONE_VM|CLONE_VFORK)`, `vfork`) is **not**
  compatible with the exec profiles' `CLONE_THREAD`-only rule, so compile filters must not reuse it.
- Without `/proc` in the jail, `go build` needs `GOROOT=/usr/local/go` set (else `go: cannot find GOROOT directory:
  'go' binary is trimmed`). `GOTELEMETRY=off` avoids the telemetry sidecar.

**KILL re-run (KILL-default, the allowlists above) ✅**

| Run | Three-arch pod | x86_64-only pod |
|---|---|---|
| references `go` / `cpp` / `python`, 3 rounds × 22 each | **66/66 / 66/66 / 66/66 correct, 0 SIGSYS** | **66/66 / 66/66 / 66/66, 0 SIGSYS** |
| go-race fixtures (`go-race` profile) | 25/25 | 125/125 |
| P2 corpus under `go`: balloon / MLE × 20 / threadbomb | MLE / **20/20** / RE(pids) | the same |
| P2 corpus under `go`: `sleep`, `spin`, `stdoutflood` | OK | OK |
| P2 corpus under `go`: `forkbomb`, `orphan` / `tmpfsfill` / `inodefill` | SIGSYS (`pidfd_open`) / SIGSYS (`close`) / SIGSYS (`newfstatat`), **expected**: the learner `go` profile can't spawn processes or manage files | the same |
| container-level OOM (`memory.events.local oom_kill`) | 0 (the hierarchical delta of 42 = the case-cgroup OOMs) | 0 |
| disallowed-syscall cases | `socket(AF_INET)` from Go → SIGSYS after `pre-socket`; ia32 `int $0x80` → SIGSYS (arch `0x40000003`); x32 `getpid` → SIGSYS; Python `socket.socket()` → SIGSYS; Python `os.fork()` → SIGSYS (`clone` without `CLONE_THREAD`) | the same |

The corpus workloads ran as a standalone module-mode Go binary, so they met the same profile a submission would.
Two early KILL attempts failed for harness reasons, not policy ones, and are excluded: a garbage-collected stdin fd
in the throwaway driver, and the supervisor binary's own `go-sandbox` init calling `statfs`.

**Pod-profile architectures: x86_64-only vs the three-arch baseline**

| Check | Three-arch (`X86_64, X86, X32`) | x86_64-only |
|---|---|---|
| jail setup (both spawn paths, caps → 0, procfs `hidepid`, `go build`, `g++ -static`, `python3`, SIGSYS row) | ✅ | ✅ identical |
| spk-01's 17 must-deny probes + `move_mount`/`mount_setattr` | denied | denied, same errnos |
| ia32 `int $0x80` `getpid` from the supervisor | **allowed** (returned the pid) | **blocked**: the thread gets SIGSYS (audit `arch=40000003 syscall=20 code=0x0`, i.e. `KILL_THREAD`); the syscall never runs |
| x32 `getpid` from the supervisor | ENOSYS(38) | blocked the same way (`code=0x0`) |
| references (KILL), go-race, postgres + SQL balloon, C++/Python compiles | ✅ | ✅, nothing broke |

- **Proposal to m3-03 (ADR-0030 delta): x86_64-only.** Nothing legitimate broke, and it closes the ia32 compat entry
  point for every process in the runner pod.
- Caveat: runc's bad-arch action is `KILL_THREAD`. In the multi-threaded supervisor, a stray compat syscall would
  leave a hung zombie leader, not a clean exit; the probe child had to be killed by hand. Inside the jail, the exec
  filter's `KILL_PROCESS` takes precedence, so a learner process dies cleanly.

**The final amd64 pod seccomp profile (mi-09 ships this file)** `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json`:
- It is §16.1's recipe on containerd v2.3.4 `DefaultProfile` (5 caps; 14 removals; `socket` limited; **`pivot_root`
  added**) with `architectures: [SCMP_ARCH_X86_64]`.
- 15 rules, 381 names; 7,035 bytes; **sha256 `730a7a5535897636ac69de4c17547ae2c4d1169bedcc44486525faa8ae5f418d`**.
- The rules are one per line, with names sorted. It is equivalent, as JSON, to the generator's indented output
  (sha256 `8817e742…e22445`).
- The names list is containerd's, including i386-only names that libseccomp skips on x86_64.
- **Validated as shipped.** A fresh pod loaded it as its Localhost profile; the runtime spec shows `[SCMP_ARCH_X86_64]`, 15 rules and `pivot_root`. The jail setup passed, and the references passed under KILL 22/22 per language.

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {"names": ["_llseek", "_newselect", "accept", "accept4", "access", "adjtimex", "alarm", "bind", "brk", "cachestat", "capget", "capset", "chdir", "chmod", "chown", "chown32", "clock_adjtime", "clock_adjtime64", "clock_getres", "clock_getres_time64", "clock_gettime", "clock_gettime64", "clock_nanosleep", "clock_nanosleep_time64", "close", "close_range", "connect", "copy_file_range", "creat", "dup", "dup2", "dup3", "epoll_create", "epoll_create1", "epoll_ctl", "epoll_ctl_old", "epoll_pwait", "epoll_pwait2", "epoll_wait", "epoll_wait_old", "eventfd", "eventfd2", "execve", "execveat", "exit", "exit_group", "faccessat", "faccessat2", "fadvise64", "fadvise64_64", "fallocate", "fchdir", "fchmod", "fchmodat", "fchmodat2", "fchown", "fchown32", "fchownat", "fcntl", "fcntl64", "fdatasync", "fgetxattr", "flistxattr", "flock", "fork", "fremovexattr", "fsetxattr", "fstat", "fstat64", "fstatat64", "fstatfs", "fstatfs64", "fsync", "ftruncate", "ftruncate64", "futex", "futex_requeue", "futex_time64", "futex_wait", "futex_waitv", "futex_wake", "futimesat", "get_robust_list", "get_thread_area", "getcpu", "getcwd", "getdents", "getdents64", "getegid", "getegid32", "geteuid", "geteuid32", "getgid", "getgid32", "getgroups", "getgroups32", "getitimer", "getpeername", "getpgid", "getpgrp", "getpid", "getppid", "getpriority", "getrandom", "getresgid", "getresgid32", "getresuid", "getresuid32", "getrlimit", "getrusage", "getsid", "getsockname", "getsockopt", "gettid", "gettimeofday", "getuid", "getuid32", "getxattr", "getxattrat", "inotify_add_watch", "inotify_init", "inotify_init1", "inotify_rm_watch", "io_cancel", "io_destroy", "io_getevents", "io_pgetevents", "io_pgetevents_time64", "io_setup", "io_submit", "ioctl", "ioprio_get", "ioprio_set", "ipc", "kill", "landlock_add_rule", "landlock_create_ruleset", "landlock_restrict_self", "lchown", "lchown32", "lgetxattr", "link", "linkat", "listen", "listmount", "listxattr", "listxattrat", "llistxattr", "lremovexattr", "lseek", "lsetxattr", "lsm_get_self_attr", "lsm_list_modules", "lsm_set_self_attr", "lstat", "lstat64", "madvise", "map_shadow_stack", "membarrier", "memfd_create", "memfd_secret", "mincore", "mkdir", "mkdirat", "mknod", "mknodat", "mlock", "mlock2", "mlockall", "mmap", "mmap2", "mprotect", "mq_getsetattr", "mq_notify", "mq_open", "mq_timedreceive", "mq_timedreceive_time64", "mq_timedsend", "mq_timedsend_time64", "mq_unlink", "mremap", "mseal", "msgctl", "msgget", "msgrcv", "msgsnd", "msync", "munlock", "munlockall", "munmap", "name_to_handle_at", "nanosleep", "newfstatat", "open", "openat", "openat2", "pause", "pidfd_open", "pidfd_send_signal", "pipe", "pipe2", "pkey_alloc", "pkey_free", "pkey_mprotect", "poll", "ppoll", "ppoll_time64", "prctl", "pread64", "preadv", "preadv2", "prlimit64", "process_mrelease", "pselect6", "pselect6_time64", "pwrite64", "pwritev", "pwritev2", "read", "readahead", "readlink", "readlinkat", "readv", "recv", "recvfrom", "recvmmsg", "recvmmsg_time64", "recvmsg", "remap_file_pages", "removexattr", "removexattrat", "rename", "renameat", "renameat2", "restart_syscall", "rmdir", "rseq", "rt_sigaction", "rt_sigpending", "rt_sigprocmask", "rt_sigqueueinfo", "rt_sigreturn", "rt_sigsuspend", "rt_sigtimedwait", "rt_sigtimedwait_time64", "rt_tgsigqueueinfo", "sched_get_priority_max", "sched_get_priority_min", "sched_getaffinity", "sched_getattr", "sched_getparam", "sched_getscheduler", "sched_rr_get_interval", "sched_rr_get_interval_time64", "sched_setaffinity", "sched_setattr", "sched_setparam", "sched_setscheduler", "sched_yield", "seccomp", "select", "semctl", "semget", "semop", "semtimedop", "semtimedop_time64", "send", "sendfile", "sendfile64", "sendmmsg", "sendmsg", "sendto", "set_robust_list", "set_thread_area", "set_tid_address", "setfsgid", "setfsgid32", "setfsuid", "setfsuid32", "setgid", "setgid32", "setgroups", "setgroups32", "setitimer", "setpgid", "setpriority", "setregid", "setregid32", "setresgid", "setresgid32", "setresuid", "setresuid32", "setreuid", "setreuid32", "setrlimit", "setsid", "setsockopt", "setuid", "setuid32", "setxattr", "setxattrat", "shmat", "shmctl", "shmdt", "shmget", "shutdown", "sigaltstack", "signalfd", "signalfd4", "sigprocmask", "sigreturn", "socketcall", "socketpair", "splice", "stat", "stat64", "statfs", "statfs64", "statmount", "statx", "symlink", "symlinkat", "sync", "sync_file_range", "syncfs", "sysinfo", "tee", "tgkill", "time", "timer_create", "timer_delete", "timer_getoverrun", "timer_gettime", "timer_gettime64", "timer_settime", "timer_settime64", "timerfd_create", "timerfd_gettime", "timerfd_gettime64", "timerfd_settime", "timerfd_settime64", "times", "tkill", "truncate", "truncate64", "ugetrlimit", "umask", "uname", "unlink", "unlinkat", "uretprobe", "utime", "utimensat", "utimensat_time64", "utimes", "vfork", "vmsplice", "wait4", "waitid", "waitpid", "write", "writev"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["personality"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 0, "op": "SCMP_CMP_EQ"}]},
    {"names": ["personality"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 8, "op": "SCMP_CMP_EQ"}]},
    {"names": ["personality"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 131072, "op": "SCMP_CMP_EQ"}]},
    {"names": ["personality"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 131080, "op": "SCMP_CMP_EQ"}]},
    {"names": ["personality"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 4294967295, "op": "SCMP_CMP_EQ"}]},
    {"names": ["process_vm_readv", "process_vm_writev", "ptrace"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["arch_prctl", "modify_ldt"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["clone", "clone3", "mount", "quotactl", "quotactl_fd", "setdomainname", "sethostname", "setns", "umount", "umount2", "unshare"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["pivot_root"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["socket"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 1, "op": "SCMP_CMP_EQ"}]},
    {"names": ["socket"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 2, "op": "SCMP_CMP_EQ"}, {"index": 1, "value": 15, "valueTwo": 1, "op": "SCMP_CMP_MASKED_EQ"}, {"index": 2, "value": 0, "op": "SCMP_CMP_EQ"}]},
    {"names": ["socket"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 2, "op": "SCMP_CMP_EQ"}, {"index": 1, "value": 15, "valueTwo": 1, "op": "SCMP_CMP_MASKED_EQ"}, {"index": 2, "value": 6, "op": "SCMP_CMP_EQ"}]},
    {"names": ["socket"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 10, "op": "SCMP_CMP_EQ"}, {"index": 1, "value": 15, "valueTwo": 1, "op": "SCMP_CMP_MASKED_EQ"}, {"index": 2, "value": 0, "op": "SCMP_CMP_EQ"}]},
    {"names": ["socket"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 0, "value": 10, "op": "SCMP_CMP_EQ"}, {"index": 1, "value": 15, "valueTwo": 1, "op": "SCMP_CMP_MASKED_EQ"}, {"index": 2, "value": 6, "op": "SCMP_CMP_EQ"}]}
  ]
}
```

**Re-run block 3 (2026-09-25): AppArmor remount narrowing ✅.** The broad `remount,` rule is replaced. The
experiments ran on the final pod shape: the x86_64-only pod seccomp file above, plus the `xlearn-runner` profile
reloaded with `apparmor_parser -r`. Each candidate was scored from the `apparmor="DENIED"` records in auditd's log.
- **Discovery (no remount rule at all).** Every jail failed at `pivot_root`, with
  `DENIED op=mount name=/ flags="ro, nosuid, remount, noatime, bind"`. That is go-sandbox's read-only remount of the
  pivoted jail root, seen as `/` inside the jail's mount namespace. The read-only bind remounts showed up as:
  - `name=/jail/jr-*/usr/ flags="ro, nosuid, remount, rbind, rprivate"` for go-sandbox's `WithBind(…, true)`;
  - `flags="ro, nosuid, nodev, remount, rbind"` for a bind made as `MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY`,
    without `MS_PRIVATE`.
- **The `rprivate` form can't be matched.** An exact `remount options=(ro, nosuid, rbind, rprivate) /jail/**` rule
  is still denied. This is spk-01's "failed flags match", reproduced; AppArmor 4.0 can't match a remount that
  carries a propagation flag.
- **The form without `MS_PRIVATE` matches exactly.** It needs nothing else, because go-sandbox has already made the
  jail's mount namespace root `rprivate` before any bind.
- A non-recursive `MS_BIND` isn't covered by the existing bind rule (`flags="rw, bind"`), so it isn't used.

**The two remount rules that replace `remount,`:**

```
  remount options=(ro, nosuid, noatime, bind) /,
  remount options=(ro, nosuid, nodev, rbind) /jail/**,
```

Both require `ro`, so **no read-write remount is possible anywhere**. The first rule also lets the supervisor
re-flag the container's own `/` read-only, which is harmless (it's already read-only); the kernel refuses even that
with EPERM, because the rootfs mount's atime flags are locked in the pod's user namespace. **Supervisor-code rule
(m3-04):** build every read-only bind as `mount.Mount{Flags: MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY}`, never
go-sandbox's `WithBind(…, true)`. A read-write bind of a writable source is writable from the jail: the `robind`
check wrote into `/work/robind` through an rw bind.

**Verification (final pod: x86_64-only pod seccomp + the final AppArmor profile; every jail bind read-only)**

| Check | Result | Pass |
|---|---|---|
| jail setup: `CLONE_INTO_CGROUP` and `cgroup.procs` spawn paths, caps → 0 (a mount after the drop → EACCES), procfs `hidepid`, `go build`, `g++ -std=gnu++20 -O2 -static`, `python3`, SIGSYS row (`core_pattern` unchanged) | all OK | ✅ |
| read-only binds (`MS_REC`, no `MS_PRIVATE`) of the read-only rootfs (`/usr`) and of a **writable** emptyDir source | both mount; `touch` inside → `Read-only file system` | ✅ |
| go-sandbox `WithBind(…, true)` (`rprivate` remount) | `mount: permission denied` | ✅ expected (not used) |
| references under KILL with read-only binds (`go` / `cpp` / `python`) | 22/22 / 22/22 / 22/22, 0 SIGSYS | ✅ |
| go-race fixtures (KILL) · GOCACHE seed (image volume, read-only bind; 12 builds) · postgres + SQL balloon | 10/10 · 12/12 OK · `select 42`, MLE in the case cgroup, recovered | ✅ |
| **remount `/` read-write** (supervisor) | **EACCES(13)**: `DENIED op=mount name=/ flags="rw, remount, bind"` | ✅ refused |
| **remount `/sys` read-write** (supervisor) | **EACCES(13)**: `DENIED op=mount name=/sys/ flags="rw, remount, bind"` | ✅ refused |
| remount a read-only tmpfs under `/jail` read-write (new must-deny) | **EACCES(13)**: `DENIED name=/jail/xl-roprobe/ flags="rw, nosuid, nodev, remount"` | ✅ refused |
| spk-01's 17 must-deny probes | `unshare -U` EINVAL(22); `clone3(NEWUSER)` and `mount` outside `/jail` EACCES(13); the rest EPERM(1); `setns` blocked at `open(/proc/1/ns/user)` EACCES | ✅ 17/17 |
| `move_mount` / `mount_setattr` | EPERM / EPERM | ✅ |
| ia32 / x32 `getpid` (x86_64-only pod profile) | the thread gets SIGSYS (`KILL_THREAD`, audit `code=0x0`) | ✅ blocked |

**The final AppArmor profile (mi-09 ships this file)** `/etc/apparmor.d/xlearn-runner` (abi 4.0, no `userns`
rule; sha256 `1d70ccd07e453f1a169cdeeb3efb638cf6c3beb8d18fe0e14581ff8ecd295775`). Its only change from §16.1 is
the remount block. The `mount fstype=overlay` rule is inert under `hostUsers:false` (EACCES either way), so mi-09
may drop it.

```
abi <abi/4.0>,
include <tunables/global>

# xLearn runner pod profile (t3 §8.7), spk-02 amd64 env-A copy (starts as the spk-01 final).
# NO userns rule: under abi 4.0 a confined task may not create a user namespace,
# even holding CAP_SYS_ADMIN. Mounts are limited to the jail paths under /jail.
profile xlearn-runner flags=(attach_disconnected,mediate_deleted) {
  include <abstractions/base>

  # exactly the runner's namespaced capabilities (pod drops ALL, adds these 5)
  capability sys_admin,
  capability setuid,
  capability setgid,
  capability setpcap,
  capability kill,

  network unix,
  network inet stream,
  network inet6 stream,
  unix,

  file,

  signal (receive) peer=unconfined,
  signal (receive) peer=runc,
  signal (receive) peer=crun,
  signal (send,receive) peer=xlearn-runner,

  # jail construction, confined to /jail (an emptyDir): tmpfs/overlay/proc/bind/remount,
  # under the jail tree only (mounts elsewhere are denied, see the /mnt probe).
  # Fresh mounts are confined to /jail (an emptyDir); a fresh mount elsewhere is denied
  # (the mount_tmpfs_outside probe proves this).
  mount fstype=tmpfs -> /jail/**,
  mount fstype=overlay -> /jail/**,
  mount fstype=proc -> /jail/**,
  mount options=(rw, rbind, nosuid, rprivate) -> /jail/**,
  mount options=(rw, rprivate) -> /,
  mount options=(rw, rslave) -> /,
  # Remounts (spk-02 block 3): NO broad `remount,`. Only read-only re-flags, only where the jail needs them:
  #  - the jail root after pivot_root (go-sandbox: MS_BIND|MS_REMOUNT|MS_RDONLY|MS_NOATIME|MS_NOSUID on "/"),
  #  - read-only bind mounts under the jail tree, made as MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY
  #    (no MS_PRIVATE: the jail mount ns root is already rprivate; AppArmor cannot match a remount carrying it).
  # A read-write remount anywhere (/, /sys, /jail/**) is denied.
  remount options=(ro, nosuid, noatime, bind) /,
  remount options=(ro, nosuid, nodev, rbind) /jail/**,
  pivot_root,
  umount,

  deny ptrace,
  deny mount fstype=sysfs,
  deny mount fstype=cgroup,
  deny mount fstype=cgroup2,
  deny @{PROC}/sysrq-trigger rwklx,
  deny @{PROC}/kcore rwklx,
  deny /sys/firmware/** rwklx,
  deny /sys/kernel/security/** rwklx,
}
```

### 16.3 Image volume (spk-02)

**Verdict (re-run block 1, 2026-09-25): image-volume GO.** (i) and (ii)/(ii-b) pass with the kubelet **defaults**,
so no `AlwaysVerify` setting is needed. The (iii) fallback also works (variant A recommended). Two node-credential
rules go to mi-09 (below). The first session was interrupted before any pod; the settings recorded then are
unchanged.

**Environment:** A's pinned k3s (as above). The private registry was `registry:2` + htpasswd over TLS with a
throwaway CA. An anonymous `GET /v2/` returned **401**. k3s `registries.yaml` carried the endpoint CA only, **no
credentials**, so credentials could come only from `imagePullSecrets`. The image was a synthetic `FROM scratch`
pack: one layer per course plus `/manifest.json` (3 layers), random bytes, 160 KiB (62 KB compressed),
**linux/amd64 only**. It holds no evalpack content. The judge stand-in is `busybox:1.37.0`. The pull secret
`xlearn-evalpack-pull` (a `dockerconfigjson` for the throwaway registry) was piped from a root-only file straight
into `kubectl apply` and never printed. **The kubelet did every pull**: the pack was absent from k3s's containerd
before (i), and nothing was pulled with `ctr` or imported.

**Kubelet settings:**
- `configz`: `imagePullCredentialsVerificationPolicy: NeverVerifyPreloadedImages`; `featureGates` not overridden;
- kubelet metrics: `KubeletEnsureSecretPulledImages` **BETA, enabled (1)**; `ImageVolume` enabled (1).

| # | Check | Result (exact events) | Pass |
|---|---|---|---|
| i | Pod `judge-i` in `xlearn` (PSA `enforce=baseline`), `imagePullSecrets: [xlearn-evalpack-pull]`, volume `image: {reference: …/xlearn-evalpack:0.1.0, pullPolicy: IfNotPresent}` mounted `readOnly` at `/evalpack`, `EVALPACK_DIR=/evalpack` | Admitted with no PSA warning; Running. `Pulling image "…/xlearn-evalpack:0.1.0"` → `Successfully pulled image … in 220ms … Image size: 62341 bytes`, then `Container image "…" already present on machine and can be accessed by the pod`. `/evalpack/manifest.json` readable (9 items; both course trees listed). `touch /evalpack/x` → `Read-only file system`. Mount: `overlay /evalpack overlay ro,relatime,…`. The kubelet's pull record (`/var/lib/kubelet/image_manager/pulled/`) maps the image to `kubernetesSecrets: [xlearn/xlearn-evalpack-pull]` | ✅ |
| ii | Pod in `probe`, **no** pull secret, `pullPolicy: IfNotPresent` | **Refused.** The kubelet re-pulls without credentials instead of reusing the cached image: `Failed to pull image "…/xlearn-evalpack:0.1.0": failed to pull and unpack image …: failed to resolve reference …: pull access denied, repository does not exist or may require authorization: authorization failed: no basic auth credentials` → `ErrImagePull` / `ImagePullBackOff` | ✅ |
| ii | The same, `pullPolicy: Never` | **Refused:** `ErrImageNeverPull`, `Container image "…/xlearn-evalpack:0.1.0" is not present with pull policy of Never` (it *is* on the node; for this pod the kubelet treats it as absent) | ✅ |
| ii+ | Pod in the **same** namespace `xlearn`, without `imagePullSecrets` (extra row) | Refused, with the same `ErrImagePull` event as (ii) | ✅ |
| ii-b | `systemctl restart k3s` (`judge-i` survived with `restartCount 0`), then (ii) and (ii+) again | Identical events: still refused. The pull records persist on disk across the restart | ✅ |
| iii-A | **Variant A:** initContainer = the pack layers + a static copier `ENTRYPOINT`, built locally and pushed only to the throwaway registry. Pulled by the kubelet through `imagePullSecrets`; runs as uid 65534 with a read-only rootfs and all caps dropped; copies into an `emptyDir`. Judge mounts the `emptyDir` `readOnly` at `/evalpack` (raw manifests; D41) | init `copier: copied 2 paths into /evalpack`, exit 0. Judge reads the manifest (9 items), and `touch` gets `Read-only file system`. **No credential file inside the pod** | ✅ works |
| iii-B | **Variant B:** a `crane:debug` initContainer runs `crane export …/xlearn-evalpack:0.1.0 - \| tar -x -C /evalpack`, with the pull secret mounted as `DOCKER_CONFIG` and the CA from a ConfigMap | init `crane-export-ok`; judge reads it read-only | ✅ works, but the **credential is mounted inside the pod** (in the init container) |

**Findings for mi-09 (node-level credentials).**
1. **Node-level registry credentials defeat (ii).** The first (ii) attempt ran while the image build's `docker login`
   had left credentials in root's `~/.docker/config.json`. The kubelet reads that file as **node-wide** credentials.
   The secret-less pods pulled successfully, and the kubelet rewrote the image's pull record to
   `nodePodsAccessible: true`, after which every pod could mount the pack. That run was discarded: the image and its
   record were removed with k3s stopped, and (i)/(ii) above were re-run clean. The effect was then reproduced on
   purpose with containerd-level credentials (`auth:` for the registry in k3s `registries.yaml`): all three
   secret-less pods Running, record `nodePodsAccessible: true`. **The record is sticky:** after removing the `auth:`
   and restarting k3s, secret-less pods still started (until the image itself is removed).
   **Rule:** the GHCR pack credential exists **only** as the `xlearn-evalpack-pull` imagePullSecret. Never put it in
   k3s `registries.yaml` `auth:`, `/var/lib/kubelet/config.json`, or a root `~/.docker/config.json` on the node
   (no `docker login` on the host). Add that to mi-09's host checks.
2. **`NeverVerifyPreloadedImages` exempts images that have no pull record.** It's safe while the pack is only ever
   pulled by the kubelet (never `k3s ctr` pull or import, no airgap tarball). If `/var/lib/kubelet/image_manager/`
   were lost while the image stayed on the node, the pack would count as preloaded and be exempt from verification.
   `AlwaysVerify` would close that, but it wasn't needed for GO and wasn't tested here: optional hardening for mi-09,
   not a gate.

**Pack platforms.** The synthetic pack is `linux/amd64` only, which is all that production (amd64) needs. A
multi-arch pack now matters only for local arm64 development (m3-02, optional).

**Verdict:** (i) and (ii) pass → **image-volume GO** (kubelet defaults; no `AlwaysVerify`). The fallback isn't
needed. If production ever differs, variant A works and is the recommended fallback (no credential inside judge's
pod), ahead of variant B. ADR-0027's image-volume line stands.

### 16.4 MI-10 verdict and proposed ADR-0030 deltas

**Spike P0–P3 GO; image volume GO** (spk-01 + spk-02 re-run, 2026-09-25). This is the line the M3 checklist reads:
- P0–P2: spk-01 (Q-A GO, Q-B GO).
- P3 (Q-C GO): TSAN needs no ASLR policy, and the amd64 allowlists pass KILL with 0 unexpected SIGSYS.
- The first session's AppArmor finding is closed: block 3's `ro`-only remount rules refuse the read-write remounts of
  `/` and `/sys`.
- Image volume GO with the kubelet defaults.
- Nothing needs an owner decision.

Proposed ADR-0030 deltas (m3-03 folds these in when it accepts the ADR; this section doesn't edit it):
- **Mechanism:** unchanged. go-sandbox `forkexec.Runner` with no user namespace, spawned via `CLONE_INTO_CGROUP`; the jail setup reproduces on amd64 kernel 6.8.0-142.
- **Pod seccomp:** §16.1's recipe on amd64 (381 names; `pivot_root` added), with **`architectures: [SCMP_ARCH_X86_64]` proposed** (x86_64-only). Nothing legitimate broke, and it closes the ia32 `int $0x80` entry point that the three-arch baseline leaves open (§16.2, block 2). The file is recorded verbatim in §16.2 (sha256 `730a7a55…418d`), and mi-09 ships it. If m3-03 prefers the baseline, the only change is the `architectures` line.
- **AppArmor:** the profile is recorded verbatim in §16.2 block 3 (sha256 `1d70ccd0…5775`), and mi-09 ships it. The broad `remount,` is replaced by `remount options=(ro, nosuid, noatime, bind) /,` (the jail root) and `remount options=(ro, nosuid, nodev, rbind) /jail/**,` (read-only binds), so no read-write remount is possible. Supervisor rule: read-only binds are `MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY` without `MS_PRIVATE`, never go-sandbox's `WithBind(…, true)`.
- **ASLR policy: none.** The Go 1.26 race runtime works in the jail under `mmap_rnd_bits=32` without calling `personality` (125/125 + 100/100 runs). The pod profile keeps RuntimeDefault's `personality` rule, so `ADDR_NO_RANDOMIZE` stays denied, and no exec profile allows `personality`. Never lower the host sysctl.
- **GOCACHE:** not an in-pod overlay (EACCES on amd64 too). Use a **read-only seed in place** (`GOCACHE` = the seed, baked into the image or mounted as an image volume; `TMPDIR` on the case tmpfs), built with the exec profile's exact toolchain and flags: about 12 s → 0.35 s per compile on env A, with no per-case copy (§16.2). A8 re-measures the timing.
- **Eval pack (not ADR-0030; for m3-07/mi-09):** image volume **GO** with the kubelet defaults (§16.3), so ADR-0027's image-volume line stands. The pack credential must stay pod-level only (never node-level).
- **Where the allowlists live:** pod seccomp → mi-09 (the file above); per-profile exec allowlists → m3-04 (`go` 27, `cpp` 19, `python` 39, `go-race` 40; §16.2 block 2). They carry the fixed rules: non-x86_64/x32 → KILL, `clone3` → ENOSYS, `clone` with `CLONE_THREAD` only, `prctl` with `PR_SET_VMA` only.
- **SETPCAP:** as §16.1.
- **Host-file diffs vs t3 §8.7 (mi-09's host-window PR):** the `judge` drop-in verbatim (§16.1). The pod seccomp file with `pivot_root` added, x86_64-only (§16.2 block 2). The AppArmor profile with the two `ro`-only remount rules in place of `remount,` (§16.2 block 3). The subuid `kubelet:1073741824:7208960`, with `getsubids` from `uidmap`. A new host check: **no node-level registry credentials** (no `auth:` in k3s `registries.yaml`, no `/var/lib/kubelet/config.json`, no root `~/.docker/config.json`; §16.3). No kubelet `imagePullCredentialsVerificationPolicy` change is needed.
