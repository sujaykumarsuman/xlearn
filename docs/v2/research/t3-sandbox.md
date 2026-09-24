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

### 8.3 Chart 0.2.1 → 0.3.0 and the runner values
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
