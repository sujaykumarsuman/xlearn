# Sprint spk-01 — Sandbox mechanism spike P0–P2 (multipass arm64, throwaway)

> **Milestone:** MI — rollout step **MI-10**, part 1 (spike week) · **Track:** spike · **Kind:** spike
> **Prereqs:** MI-0 done (event `ev-mi0`) · soft: [mi-14](sprint-mi-14.md) (the MI-4 guard manifests). The owner's go-ahead (D23) is the launch of this prompt (D40); the session records `ev-spike-goahead`
> **Unblocks:** [spk-02](sprint-spk-02.md) (P3 + image volume, on this VM and harness) · [spk-03](sprint-spk-03.md) (WIF, on this VM) · [mi-09](sprint-mi-09.md) (host files) · [m3-03](sprint-m3-03.md) (ADR-0030 acceptance) · the M3 checklist line "Spike P0–P3 **GO**" (together with spk-02)
> **Release action:** no merge (spike, throwaway). The harness, VM and manifests are never committed; only the results docs PR lands, squash-merged on CI green with no owner stop (D40)
> **Calendar:** Mon 2026-10-12 → Wed 2026-10-14 (spike week, event `ev-spike-week`). The calendar is the booking window, not the budget. **Hard stop, counted as effort:** 10 h core for P0–P3 together ([t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead)). This sprint's core share is 7 h (P0 1.5 + P1 3 + P2 2.5), plus P1b's 0.75 h, which sits outside the core
> **Execute with:** [`../prompts/prompt-spk-01.md`](../prompts/prompt-spk-01.md) — one prompt, one session.

## D41 changes (read first; they override the text below where they conflict)

> **D41 (owner, 2026-09-25): spikes first.** All four spikes run **before any build sprint**, so every design yes/no is answered before M1 starts: spk-01 and spk-02 on Fri 2026-09-25 (agent-only, after the MI-0 reboot), spk-03 and spk-04 on Sat 2026-09-26 (spk-03 after the owner's Console step; spk-04 with the owner present). For this sprint: the calendar line below (Oct 12–14) is superseded; everything else stands. mi-14 hasn't run, so use the soft-gate fallback (write the draft guard objects yourself and report the diff back to mi-14).

## Status

_Overall:_ ✅ Done — **Q-A GO, Q-B GO**; mechanism: go-sandbox `forkexec.Runner` (no userns, `CLONE_INTO_CGROUP`); nsjail not needed (2026-09-25)

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | P0 setup: VM, k3s, host files, guards, VAP proof (1.5 h) | H | ✅ drop-in merges; VAP 24/24 dry-run + X1/X2/E1 CONNECT denied |
| 2 | P1 viability: R1 positive and negative checks (3 h) | H | ✅ jail works both cgroup modes; 17/17 negative probes denied; ×50 recreate 50/50 |
| 3 | P1b pilot check: go-race and postgres in the jail (0.75 h, doesn't gate M3) | H | ✅ RACE 20/20, DEADLOCK 20/20, postgres OK |
| 4 | P2 INV-14 and cleanup corpus (2.5 h) | H | ✅ MLE 100/100, 0 container OOMs, baseline +2 MiB, nr_dying→0 in 6 s |
| 5 | Results table → `t3-sandbox.md` §16.1 and `status.md` (docs PR); hand the VM to spk-02 | X | ✅ §16.1 written; VM stopped for spk-02 |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, and ⛔ if it's blocked (say why).
> A NO-GO row is still ✅ once it's recorded with numbers; the verdict goes in the results. Update the _Overall_ line to match, and mirror the sprint's state into
> [`../status.md`](../status.md) (the Sprint board row and the MI rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **The go-ahead for P0–P3 and the image-volume spike (D23) is this launch** (D40). Record `ev-spike-goahead` ✅ in `docs/v2/status.md` with the results.
- [ ] **MI-0 is done.** Production runs kernel `6.8.0-142` (`ssh sujaykumar-vps uname -r`, read-only). The VM mirrors that noble kernel line.
- [ ] **multipass ≥ 1.16** is on the owner's M3 Max with 8 GiB of RAM and 30 GB of disk free for the VM. At planning time, 1.16.3 was installed on a 36 GiB machine.
- [ ] **Soft:** the MI-4 manifests from [mi-14](sprint-mi-14.md) are merged in `../infra/infrastructure/sandbox/` or open on its PR branch. If they aren't ready, apply the [t3 §8.2](../research/t3-sandbox.md#82-guard-objects-infrastructuresandbox) draft and report the diff back to mi-14.
- [ ] **Soft:** mi-14's VAP corpus (`../infra/hack/sandbox-vap-test/`) and `hack/sandbox-vap-test.sh` exist on `main` or its PR branch. If they're absent, write G1, the required B-shapes, Q1 and C1 yourself in `~/xl-spike/manifests/`, following [mi-14](sprint-mi-14.md) task 2's definitions, and report them back to mi-14.

## Goal

Answer the two go/no-go questions that the runner design (R1, [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md), Proposed) depends on, on a throwaway arm64 VM that mirrors production's software stack:
- **Q-A: R1 viability.** Does each of these work on Ubuntu 24.04 with k3s v1.36.4 (containerd 2.3.4, runc 1.4.2) and the AppArmor userns restriction?
  - the `judge` containerd **drop-in** (`cgroup_writable`, systemd);
  - `hostUsers:false` with namespaced capabilities;
  - Localhost AppArmor (no `userns` rule) and Localhost seccomp;
  - a **userns-less `forkexec.Runner` jail** with per-case cgroups;
  - user-namespace creation and the new mount API **denied** to the supervisor.
- **Q-B: INV-14.** Does every learner OOM land in a case cgroup, SQL backends included, with **zero container OOMs**, and does cgroup cleanup converge?

This is MI-10 part 1 ([rollout §2](../rollout-plan.md#2-mi-infra-track)). Its answers feed ADR-0030's acceptance ([m3-03](sprint-m3-03.md) task 1), the host files for the October window ([mi-09](sprint-mi-09.md)), and any VAP diff for [mi-14](sprint-mi-14.md).

## Scope

**In**
- **P0 setup, P1 viability, P1b pilot check and P2 INV-14 + cleanup**, exactly as [t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead) specifies, inside the time box.
- **The fallback chain, if the default path fails:** go-sandbox `forkexec.Runner` → nsjail `--disable_clone_newuser` (R1-N) → R1-U → R1b ([t3 §3](../research/t3-sandbox.md#3-isolation-options-on-this-host)).
- **A functional smoke test for the three launch languages** (D20): `go build`, `g++`, `python3` inside the jail. The allowlists themselves come from P3.
- **Recording the results.** The results table, the final host files and the VAP diff go into `docs/v2/research/t3-sandbox.md` §16.1 and `docs/v2/status.md`.

**Out**
- **P3, the amd64 replay and allowlists, and the image-volume spike:** [spk-02](sprint-spk-02.md).
- **The WIF spike:** [spk-03](sprint-spk-03.md).
- **Latency, CV, canary thresholds, the `GOCACHE` mechanism, the cross-account marker suite and gVisor:** A8 on production ([m3-15](sprint-m3-15.md) acceptance suite, [mi-10](sprint-mi-10.md)).
- **Timing conclusions of any kind.** arm64 under HVF doesn't carry over to the EPYC guest.
- **Anything on the VPS or the production cluster.** Only read-only `ssh sujaykumar-vps` facts are allowed.
- **Accepting ADR-0030** ([m3-03](sprint-m3-03.md)), **writing `host-bootstrap.sh`** ([mi-09](sprint-mi-09.md)), and **committing any harness, manifest or VM artefact**.

## Tasks

**Shared setup.**
- **VM:** `xl-spike`.
- **Work directory:** `~/xl-spike/` on the laptop, **outside every git repo** and never committed. Mount it into the VM with `multipass mount ~/xl-spike xl-spike:/spike`. It holds the throwaway supervisor, the corpus, the pod manifests and the raw logs. spk-02 reuses it, and spk-02 deletes it.
- **Run every `kubectl` inside the VM** (`multipass exec xl-spike -- sudo k3s kubectl …`). The laptop's kubectl context tunnels to production.
- **Running commands inside the positive pod.** The VAP denies CONNECT on `pods/exec` and `pods/attach` in `xlearn-runner` by design, so `kubectl exec` can't reach the pod. Use one of these channels:
  - **CRI exec (default):** `multipass exec xl-spike -- sudo k3s crictl exec <container-id> …`, with the id from `k3s crictl ps --name <container> -q`. CRI calls bypass API-server admission, but runc still applies the container's seccomp, AppArmor, capabilities and user namespace to the exec'd process, so the checks stay representative.
  - **Baked command (alternative):** put the checklist and the supervisor corpus in the container's command, and read the results with `kubectl logs`. That's a GET on `pods/log`, not a CONNECT.
  - **Never relax the VAP to get exec.** That would invalidate the X1/X2 proof.

### 1 · P0 setup (1.5 h) [H]

- **VM.**
  - `multipass launch 24.04 --name xl-spike --cpus 4 --memory 8G --disk 30G`, then `sudo apt-get update && sudo apt-get -y full-upgrade && sudo reboot` to mirror H0.
  - Mirror the rest of H0 that the probes depend on: `kernel.core_pattern=core`, `fs.suid_dumpable=0`, and apport masked. Without these, P1's "no host core helper spawned" check proves nothing.
  - Add the sandbox sysctls from [t3 §8.7](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded) as `/etc/sysctl.d/60-xlearn-sandbox.conf`:
    - `io_uring_disabled=2`, `unprivileged_bpf_disabled=2`, `vm.unprivileged_userfaultfd=0`, `dmesg_restrict=1`, `kptr_restrict=2`;
    - assert `perf_event_paranoid ≥ 3`.
- **Assert and record.**
  - `uname -r` (arm64 noble; record production's `6.8.0-142` alongside it).
  - `sysctl kernel.apparmor_restrict_unprivileged_userns` = 1.
  - `stat -fc %T /sys/fs/cgroup` = `cgroup2fs`.
  - `/dev/kvm` absent. If HVF nested virtualisation exposes it, record that; R1 doesn't use it.
- **Install inside the VM only.**
  - k3s **`v1.36.4+k3s1`**, exactly as `../infra/hack/host-bootstrap.sh` does it: the installer from the pinned tag's `install.sh`, `INSTALL_K3S_VERSION`, and args `server --write-kubeconfig-mode 0644`.
  - **go1.26.8**, from the checksummed go.dev tarball.
  - **PGDG PostgreSQL 18**, `g++` and `python3`.
- **Host files, written from the [t3 §8.7](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded) drafts.**
  - **containerd drop-in** `/var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d/20-judge.toml`: runtime `judge`, `io.containerd.runc.v2`, `cgroup_writable = true`, `SystemdCgroup = true`. It's a drop-in: no `.tmpl` and no `BinaryName`.
  - **AppArmor** `/etc/apparmor.d/xlearn-runner`: `abi <abi/4.0>`, **no `userns` rule**, mounts limited to the jail paths, `pivot_root`, the caps `sys_admin setuid setgid setpcap kill`, and `deny ptrace`. Load it with `apparmor_parser -r`.
  - **Seccomp** `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json`: RuntimeDefault-with-SYS_ADMIN, **minus** the new mount API, bpf/perf/fanotify, the keyring calls, io_uring, userfaultfd, `lookup_dcookie` and `syslog`. `socket()` is limited to AF_UNIX and AF_INET/INET6 `SOCK_STREAM`.
  - **User-namespace range:** a system user `kubelet` with `/etc/subuid` and `/etc/subgid` entries `kubelet:<start>:7208960` (110 × 65,536), ending well below UINT32_MAX. Install `getsubids`.
- **Restart k3s with a test pod running.** Record whether running pods survive (container restart counts).
- **Assert:**
  - `k3s crictl info` lists `judge`;
  - the rendered `…/containerd/config.toml` runc section still says `SystemdCgroup = true`;
  - the `judge` and default runc report the same `runc --version`.
- **Guards.** Apply the MI-4 objects inside the VM:
  - from `../infra/infrastructure/sandbox/` (`main` or mi-14's PR branch), or else the t3 §8.2 draft;
  - Namespace `xlearn-runner`, the two VAPs + bindings (pod shape; no exec/attach), RuntimeClass `xlearn-judge` → `judge`, PriorityClass −1000, Quota, LimitRange, and the `default-deny-all` / `judge-to-runner` NetworkPolicies;
  - the bindings go in at **`validationActions: [Deny]`**. mi-14 ships `[Warn, Audit]` first and flips to `[Deny]` in its second PR. If only the first PR has merged, change the VM copy only.
- **VAP proof.** Reuse mi-14's corpus, `../infra/hack/sandbox-vap-test/*.yaml`, and its `sandbox-vap-test.sh --expect deny` logic, pointed at the VM's API server (`multipass exec … k3s kubectl apply --dry-run=server`). Don't run it over `ssh sujaykumar-vps`: that targets production. If mi-14's corpus is absent, use the cases you wrote in `~/xl-spike/manifests/` (entry gates).
  - **G1** must be admitted.
  - **B1–B14** must all be denied. The **required 8** match mi-14's list: privileged (B1), hostPID (B2), token automount (B3), wrong image (B4), Unconfined seccomp (B5), extra caps (B6) and hostPath (B7) as dry-runs, plus **exec**, proven by the real X1/X2 CONNECT below.
  - **Q1** must be denied.
  - **C1**, the same bad shape in `default`, must be **admitted**, which proves the `namespaceSelector` scoping (mi-14's risk).
  - **X1/X2:** a **real** `kubectl exec` / `attach` into the P1 positive pod once it exists. CONNECT must be denied. mi-14 can't prove this on production, where no pod runs yet, so this spike owns the proof.
  - **E1 (extra row, not one of the 8):** a real `kubectl debug` ephemeral-container attempt on the P1 positive pod (no `-it`, so it only writes `pods/ephemeralcontainers`) must be denied, since the t3 §8.2 VAP covers that subresource. A dry-run of a pod manifest can't test this. Report E1 to mi-14 as a corpus addition.
- **Pass:** `judge` is listed, `SystemdCgroup=true` still renders, the runc is the same, **the VAP denies 8/8 (B1–B7 plus exec CONNECT)**, and E1 is denied.
- **Fail:** fix the drop-in. If containerd doesn't merge it, fall back to a minimal `config-v3.toml.tmpl` and record that. Carry this into mi-09 and `host-verify`.

### 2 · P1 viability (3 h) [H]

**The positive pod.** Build a throwaway image, `debian:trixie-slim` + go1.26.8 + the throwaway supervisor + `g++`/`python3`/PG 18, for arm64.
- Import it with `k3s ctr images import` under the name `ghcr.io/sujaykumarsuman/xlearn-runner` and reference it **by digest** with `imagePullPolicy: Never`. The VAP's image rule pins `ghcr.io/sujaykumarsuman/xlearn-runner@sha256:*`.
- If CRI can't resolve a digest reference for an imported image, run the positive pod against a spike-only copy of the VAP with only the image rule widened, and record that. The dry-run set (G1, B1–B14, Q1, C1) always runs against the unmodified VAP. X1/X2 and E1 then run against the widened copy, which differs only in the image rule, so they still prove the CONNECT and ephemeral-container denials; record which VAP they ran against.
- The pod shape follows [t3 §8.2–8.3](../research/t3-sandbox.md#82-guard-objects-infrastructuresandbox):
  - `hostUsers: false` and `runAsUser: 0`;
  - caps `SYS_ADMIN SETUID SETGID SETPCAP KILL`, drop `ALL`;
  - AppArmor `Localhost/xlearn-runner` and seccomp `Localhost/profiles/xlearn-runner.json`, at pod and container level;
  - `procMount: Unmasked`, a read-only rootfs, `automountServiceAccountToken: false`, `enableServiceLinks: false`;
  - `priorityClassName: xlearn-sandbox-lowest`, `dnsPolicy: None` with nameserver `127.0.0.1`, emptyDir only, and requests/limits inside the Quota.

**Checklist.** Each item is one results row with its errno ([t3 §9 P1 checklist](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead)). Run the in-pod rows through CRI exec or the baked command (shared setup), never through `kubectl exec`.

| Kind | Check | Pass |
|---|---|---|
| Positive | pod Running with the shape above; the UID map is not the identity map; the AppArmor label is `xlearn-runner` | ✓ |
| Positive | `mkdir /sys/fs/cgroup/slots`; enable `+cpu +memory +pids` | ✓ |
| Positive | `forkexec.Runner` jail **without CLONE_NEWUSER** (mnt/pid/net/ipc/uts/cgroup): tmpfs root, `nosuid,nodev` read-only binds, one overlay, `pivot_root`, procfs `hidepid=invisible,subset=pid`, loopback down | ✓ |
| Positive | spawn into a case cgroup via **`CLONE_INTO_CGROUP`** and via a **`cgroup.procs` write + sync pipe** (kernel 6.8) | record both |
| Positive | caps → 0 and the bounding set dropped in the child. **Is `SETPCAP` needed?** | record |
| Positive | `go build` in the jail; **D20 smoke:** `g++ -std=gnu++20 -O2` compile + run, and a `python3` run, in the jail | ✓ (functional only) |
| Negative, from the supervisor | `unshare -U`, `clone3(CLONE_NEWUSER)`, `fsopen`+`move_mount`, `open_tree`+`move_mount`, `socket(SCTP)`, `socket(AF_NETLINK)` | all **fail** |
| Negative, from the jail | a disallowed syscall → SIGSYS; then **no host core helper spawned** (`core_pattern` untouched, no new process on the VM) | ✓ |
| Negative, API | X1/X2: a real `kubectl exec` / `attach` CONNECT into this pod, and E1: `kubectl debug` (ephemeral container) | all **denied** |
| Stability | delete/recreate the pod × 50 (the subuid range; kubernetes #139916) | 50/50 Running |

**Fail path, inside the time box:**
- forkexec fails → **nsjail `--disable_clone_newuser`** (R1-N), spawned into our case cgroup.
- nsjail fails → the result is outside every pre-decided path. Record whether **R1-U** (the profile gains `userns,`) or **R1b** (`hostUsers:true`) would work, with a recommendation, but **don't choose**: R1-U/R1b are the owner's call, and R1b is a D21 trigger to move to a dedicated runner VPS. Mark the M3 checklist line ⛔ "needs owner decision" in `status.md` and still land the results PR (D40).
- Any fallback changes the host files. Record the final ones.

### 3 · P1b pilot check (0.75 h, doesn't gate M3) [H]

- Run `go test -c -race` on a small race fixture, then run the test binary in the jail. With glibc, `clone3` should fall back to `clone` on ENOSYS. Record the errno trail.
- Boot `postgres` (PGDG 18) in the jail on a Unix socket: `listen_addresses=''`, socket in `/job/sock`, loopback down. Run one query.
- **Pass:** both run.
- **Fail:** record it for [p-01](sprint-p-01.md). The pilot profile is redesigned before the pilot, and this doesn't block M3.

### 4 · P2 INV-14 and cleanup (2.5 h) [H]

**Supervisor.** A throwaway supervisor of about 400 lines in `~/xl-spike/`, using the cgroup layout from [t3 §7.1](../research/t3-sandbox.md#71-cgroup-layout):
- `runner/` and `slots/s0|s1`;
- per case, `case-K` with `memory.max`, `memory.oom.group=1`, `memory.swap.max=0` and `pids.max`;
- evidence read outside the process: `memory.events`, `memory.peak`, `cpu.stat`, the pidfd, and pipe byte counts;
- `cgroup.kill` and `rmdir` after each case.

**Corpus:** a 1 GiB balloon, a fork bomb, a thread bomb, a tmpfs fill, an inode fill, a stdout flood, an orphan double-fork, sleep/idle, a spin, a **SQL balloon** (`array_agg` / recursive CTE, with the backend moved into `case-K`), a **SQL `set_config` timeout bypass** (the wall kill and cgroup must still end it), a lock-order deadlock, and a race fixture.

**Pass** ([t3 §9 P2](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead), [§5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8)):

| Measure | Pass |
|---|---|
| MLE classification over 100 balloon runs (SQL included) | **100/100** |
| Container-level OOM kills (`memory.events` `oom_kill` of the container cgroup; pod `restartCount`) | **0**. Any container OOM is a **blocker** |
| Surviving processes after each job | 0 |
| `runner/` + slot memory back to baseline | ± 5 MiB |
| `nr_dying_descendants` after 1,000 case cgroups | → ~0 within 60 s |
| RACE detected | ≥ 19/20 |
| DEADLOCK classified | 20/20 |

**If a row fails,** tune the limits and re-run within the time box. Otherwise record NO-GO for Q-B with the numbers.

### 5 · Results [X] — docs PR `docs(v2): sandbox spike P0–P2 results (MI-10)`

- **Append `## 16. Spike results (MI-10)` to [`../research/t3-sandbox.md`](../research/t3-sandbox.md), with `### 16.1 P0–P2 (spk-01, arm64 multipass)`.** spk-02 adds §16.2–§16.4. It contains:
  - **the environment:** VM kernel, k3s, containerd, runc, go, PG and g++/python versions, with production's kernel for contrast. Also the **`github.com/criyle/go-sandbox` module version and commit** the supervisor built with (`go list -m github.com/criyle/go-sandbox`), and the **nsjail version and commit** if R1-N was used. [m3-03](sprint-m3-03.md) pins go-sandbox at exactly this version;
  - **the in-pod channel used** (CRI exec or baked command);
  - **the per-phase results table:** step, measure, number, pass/fail, errno;
  - **Q-A and Q-B verdicts:** GO or NO-GO;
  - **the jail mechanism chosen:** forkexec or R1-N; or, past R1-N, "needs owner decision" with the R1-U/R1b options and a recommendation;
  - **the open points answered:** whether `SETPCAP` is needed; `CLONE_INTO_CGROUP` vs `cgroup.procs` on 6.8; pod survival on a k3s restart; the subuid range used and the × 50 recreate result; which noble package ships `getsubids`;
  - **the final host files, verbatim:** the drop-in TOML (merged, or the `.tmpl` fallback), the AppArmor profile, and the arm64 pod seccomp profile as a sorted syscall list (or the diff from t3 §8.7). The arm64 pod profile is **not** the one mi-09 ships: spk-02 replays it on amd64;
  - **the VAP diff** against mi-14's manifests, if any, plus the E1 corpus addition (and the self-written corpus, if mi-14's was absent);
  - **"No timing conclusions (arm64 / HVF)."**
- **Leave ADR-0030 untouched.** [m3-03](sprint-m3-03.md) task 1 accepts it from these results.
- **`docs/v2/status.md`:**
  - the MI-10 row: part 1 result, GO or NO-GO;
  - the Sprint board row for spk-01;
  - `ev-spike-goahead` ✅ (the launch, D40);
  - Decisions log lines: the mechanism chosen, and any fallback that needs an owner decision (marked ⛔ "needs owner decision");
  - hand-off notes to spk-02 (the VM and work directory), mi-09 (host files), mi-14 (VAP diff: comment on its PR, or open an infra issue if it has merged) and m3-03.
- **The VM:** `multipass stop xl-spike`. **Don't delete it.** spk-02's image-volume check (Thu) and spk-03's WIF spike (Fri) reuse its k3s. The last spike of the week deletes it with `multipass delete --purge`.

## Acceptance criteria

- [ ] Every P0, P1, P1b and P2 row is filled with GO/NO-GO, numbers and an errno where relevant, and Q-A and Q-B each have a verdict.
- [ ] The VAP denies 8/8 required shapes (B1–B7 by dry-run, plus exec by the real X1/X2 CONNECT), denies the rest of B1–B14, Q1 and the E1 ephemeral-container attempt, and admits the same shape outside `xlearn-runner` (C1). The VAP was never relaxed to obtain exec.
- [ ] Zero container OOMs across P2. Any container OOM is recorded as a NO-GO blocker.
- [ ] The final host files and the chosen jail mechanism are recorded verbatim in t3 §16.1, and the hand-offs to mi-09, mi-14, m3-03 and spk-02 are noted in `status.md`.
- [ ] Nothing but the results is committed. The VM is stopped and handed to spk-02 (deleted by the last spike of the week), and `~/xl-spike/` is outside every repo.

## Release

**No merge: a throwaway spike.** The harness, corpus, manifests and VM are never committed. The only merge is the **docs PR** carrying the results (t3 §16.1 and `status.md`), squash-merged on CI green with no owner stop (D40); it doesn't ship in any tag. A result outside the pre-decided paths still lands, with its options, a recommendation and ⛔ "needs owner decision" in `status.md`. There's no infra PR, and nothing touches production.

## Definition of Done

- The results are recorded and the docs PR is merged.
- No production change was made: read-only `ssh sujaykumar-vps` facts only.
- Statuses are updated in this file and in [`../status.md`](../status.md).
- The VM and work directory are handed over per task 5.
- The hard stop was respected. If it was hit, every unfinished row is marked "not run (time box)".

## Risks / watch-outs

- **Hard stop: 10 h core effort for P0–P3**, counted as effort, not elapsed days; the Mon→Wed calendar is only the booking window. This sprint's core share is 7 h, and P1b's 0.75 h sits outside the core. When you reach it, stop and report what you know. Unknown rows stay explicit and never count as a GO.
- **The VAP blocks `kubectl exec` into the positive pod by design.** Use CRI exec or a baked command. Relaxing the VAP to get a shell would invalidate the X1/X2 proof.
- **arm64 timings don't carry to the EPYC guest.** Draw no timing conclusions. Mechanism semantics (userns, cgroup delegation, AppArmor mediation, seccomp, procfs) don't depend on the architecture ([t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead)).
- **The laptop's kubectl context tunnels to production** (`127.0.0.1:6443`). Every `kubectl` goes through `multipass exec … k3s kubectl`, with `KUBECONFIG` unset on the laptop.
- **A fallback past R1-N is an owner decision.** R1b is a D21 trigger (move to a dedicated runner VPS), so don't pick it: record the options with a recommendation, mark ⛔ "needs owner decision" and still land the results (D40).
- **kubernetes #139916** (`hostUsers:false` sandbox failures) can make the ×50 recreate loop flaky. The bounded subuid range is the mitigation, so record every failure.
- **The VM's AppArmor/kernel build differs from production's** (arm64 vs amd64 noble). spk-02 replays the syscall-sensitive parts on amd64.
