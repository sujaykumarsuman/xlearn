# Sprint spk-02 — P3 amd64 replay + image-volume spike (throwaway)

> **Milestone:** MI — rollout step **MI-10**, part 2 (spike week) · **Track:** spike · **Kind:** spike
> **Prereqs:** [spk-01](sprint-spk-01.md) (the VM, work directory, supervisor and host files) · [mi-07](sprint-mi-07.md) (MI-9: the `xlearn-evalpack:0.1.0` scaffold image, the PAT and the SOPS pull secret)
> **Unblocks:** [mi-09](sprint-mi-09.md) (the amd64 pod seccomp profile and final host files) · [m3-03](sprint-m3-03.md) (ADR-0030 acceptance) · [m3-04](sprint-m3-04.md) (per-language amd64 allowlists) · [p-01](sprint-p-01.md) (go-race TSAN verdict) · [ds-p-01](sprint-ds-p-01.md) (its entry gate: the P3 result the owner weighs for PRD Q5) · [m3-07](sprint-m3-07.md) (the pack mount: image volume or fallback) · the M3 checklist line "Spike P0–P3 **GO** and the image-volume spike **GO**" ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist))
> **Release action:** no merge (spike, throwaway). The harness, scratch repo and VMs are never kept; only the results docs PR lands, squash-merged on CI green with no owner stop (D40)
> **Calendar:** Thu 2026-10-15 → Fri 2026-10-16 (spike week, event `ev-spike-week`). The calendar is the booking window, not the budget. **Hard stop, counted as effort:** this sprint's share of the 10 h P0–P3 core is 3 h (P3 2 h + report 1 h); the image-volume spike is about 1.5 h on top, outside the core
> **Execute with:** [`../prompts/prompt-spk-02.md`](../prompts/prompt-spk-02.md) — one prompt, one session.

## D41 changes (read first; they override the text below where they conflict)

> **D41 (owner, 2026-09-25): spikes first.** All four spikes run **before any build sprint**, so every design yes/no is answered before M1 starts: spk-01 and spk-02 on Fri 2026-09-25 (agent-only, after the MI-0 reboot), spk-03 and spk-04 on Sat 2026-09-26 (spk-03 after the owner's Console step; spk-04 with the owner present). For this sprint, overriding the text below:
> - **Environment A is the owner's `skriptvalley-vps`** (Hostinger KVM 1: 1 vCPU, 3.9 GB RAM, 45 GB free, amd64, Ubuntu 24.04, kernel 6.8.0-90, `vm.mmap_rnd_bits=32`, AppArmor userns restriction on — the same as production). The owner declared it free for any PoC and cleanup. It runs two throwaway landing containers on 80/443 (stop them if they get in the way). Install k3s `v1.36.4+k3s1` with `--disable traefik`. Launching approves its full-upgrade and reboot. **Teardown:** `k3s-uninstall.sh` and remove everything the spike added (no reimage needed); leave the landing containers as found. This is not the D12/D21 "backup VPS" concern, which is about hosting the production runner.
> - **The MI-9 gate is dropped.** The image-volume questions (i)–(iii) are about kubelet/containerd behaviour, not GHCR, so run them against a **password-protected private registry inside environment A** (`registry:2` with htpasswd; k3s `registries.yaml`) holding a synthetic `FROM scratch` pack image. GHCR specifics (package private, anonymous GET 401/403) stay mi-07's CI probe.
> - **The chart-0.3.0 gate is dropped.** Use raw manifests for (iii); mi-01 verifies the `initContainers` knob renders the same shape.
> - Environment B (GitHub Actions) is not needed.

## Status

_Overall:_ ✅ **Done 2026-09-25: Spike P0–P3 GO, image volume GO (MI-10 ✅).** The first session was interrupted after the P3 jail setup and probes. The re-run landed in three blocks: block 1 gave image volume **GO** and the GOCACHE seed. Block 2 gave Q-C GO (TSAN with no ASLR policy, PG, allowlists at 0 unexpected SIGSYS) and the x86_64-only proposal. Block 3 closed the AppArmor `remount,` finding with `ro`-only scoped rules. Env A was torn down. See [t3 §16.2–16.4](../research/t3-sandbox.md#162-p3-amd64-replay-spk-02).

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | P3 amd64 environment: the second VPS (SSH access given by the owner before launch), or a GitHub Actions scratch job | H (+ O before launch) | ✅ env A = `skriptvalley-vps` (D41): kernel 6.8.0-142, `mmap_rnd_bits=32`, k3s v1.36.4, a private registry |
| 2 | P3 replay: the amd64 pod seccomp profile, TSAN under `mmap_rnd_bits=32`, amd64 allowlists (Go, C++, Python), KILL re-run (2 h) | H | ✅ (re-run block 2). **Q-C GO:** TSAN works under `mmap_rnd_bits=32` with no ASLR policy (125/125 + 100/100), and PG + SQL balloon → MLE. The allowlists are `go` 27, `cpp` 19, `python` 39 and `go-race` 40, and the KILL re-run gives 66/66 per language with 0 unexpected SIGSYS. The x86_64-only pod profile is tested (nothing broke, ia32 closed) and proposed; the final JSON is recorded verbatim. Block 1 measured the GOCACHE seed without an overlay ✅. The first session's finding (the broad AppArmor `remount,` rule) is **closed in block 3**: two `ro`-only remount rules; read-write remounts of `/`, `/sys` and `/jail/**` → EACCES; the final profile is recorded ([t3 §16.2](../research/t3-sandbox.md#162-p3-amd64-replay-spk-02)) |
| 3 | Image-volume spike (i) mount, (ii) cached-image credential check, (iii) initContainer fallback (≈ 1.5 h) | H | ✅ **GO** (re-run block 1): (i) mounts read-only under PSA baseline. (ii) and (ii-b) are refused without the secret (`ErrImagePull … no basic auth credentials`, `ErrImageNeverPull`), with the kubelet defaults and no `AlwaysVerify`. (iii) variants A and B both work, and A is recommended. **Finding → mi-09:** node-level registry credentials make the cached pack readable by every pod ([t3 §16.3](../research/t3-sandbox.md#163-image-volume-spk-02)) |
| 4 | Report → `t3-sandbox.md` §16.2–16.4, t1 §3.3 pointer, `status.md` (docs PR, 1 h) | X | ✅ first session's partial results (#61); re-run blocks 1–3 in #62, #63 and the block-3 PR (D40) |
| 5 | Teardown: VM, work directory, SSH alias; the second VPS reimage or scratch repo deletion recorded as an owner follow-up (not a wait) | H | ✅ env A restored to baseline after both sessions (2026-09-25; D41: no reimage; the alias is the owner's, so it stays). `xl-spike` purged. `~/xl-spike/` is kept on the Mac for the orchestrator (throwaway, no secrets); `rm -rf ~/xl-spike` when done |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, and ⛔ if it's blocked (say why).
> A NO-GO row is still ✅ once it's recorded with numbers. Update the _Overall_ line to match, and mirror the sprint's state into
> [`../status.md`](../status.md) (the Sprint board row and the MI rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **spk-01 is done.** t3 §16.1 is recorded, and the `xl-spike` VM (stopped) and `~/xl-spike/` (supervisor, profiles, corpus) are available.
- [ ] **MI-9 is done** ([mi-07](sprint-mi-07.md)):
  - `ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` exists and is private (the CI anonymous-GET probe returns 401/403);
  - `../infra/apps/secrets/xlearn-evalpack-pull.enc.yaml` is merged;
  - the PAT's expiry is in `status.md`.
- [ ] **The environment is settled before launch** (D40). Environment A, the owner's **second VPS**, is used only if the owner gave SSH access to it before launch (the prompt's before-launch block) because it exists, is **amd64** and is **empty**; launching approves its full-upgrade, reboot and later reimage. **Otherwise** the session uses the GitHub Actions path and creates the private scratch repo itself (approved by the launch). That path serves **P3 only**; it never runs the image-volume check (task 3).
- [ ] **The image-volume check has a compliant environment.** Either the pack image includes **`linux/arm64`**, so the arm64 `xl-spike` VM can pull it, **or** environment A exists. mi-07 builds `0.1.0` with no platform list, which gives a single-arch amd64 image, so arm64 is only there if the build was made multi-arch (`platforms: linux/amd64,linux/arm64`; free for `FROM scratch` data).
  - **Check without an authenticated registry request from the laptop:** `git -C ../xlearn-evalpack show v0.1.0:.github/workflows/build.yml | grep -n platforms`, plus the build run's log (`gh run view -R sujaykumarsuman/xlearn-evalpack <run-id> --log`). If both are inconclusive, treat the image as single-arch amd64 (the likely case).
  - **If neither holds,** P3 can still run, but task 3 is ⛔ and the image-volume line stays red. Report it: the fix is a multi-arch rebuild in the evalpack repo (mi-07's build job with the platform list, as a new `0.1.x` tag), then use that tag in place of `0.1.0`.
- [ ] **Chart 0.3.0 is merged** ([mi-01](sprint-mi-01.md)), because it carries the `initContainers` knob for (iii). If it's still a PR, render from its branch and note that.

## Goal

This is MI-10 part 2 ([rollout §2](../rollout-plan.md#2-mi-infra-track)); with spk-01 it produces the M3 checklist's first line. It answers two sets of questions.

**Q-C** ([t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead)), on amd64:
- Does `go test -race` work in the jail under noble amd64's `vm.mmap_rnd_bits=32` (production's value, [t3 §15](../research/t3-sandbox.md#15-s0-results-read-only-production-facts-2026-09-24-owner-approved))? Which syscalls does TSAN's ASLR re-exec need?
- What are the **amd64 exec allowlists** for the three launch languages, Go, C++ and Python (D20)?

**The eval-pack image-volume questions** ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack), [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md)):
- Does judge's read-only **image volume** of the private pack mount with the pull secret?
- Can a pod **without** the secret mount the cached image?
- Does the **initContainer + emptyDir fallback** work with chart 0.3.0's knob?

## Scope

**In**
- **P3 amd64 replay (2 h) and report (1 h),** as [t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead) specifies, reusing spk-01's supervisor and profiles cross-compiled for amd64.
- **Allowlist generation in `SECCOMP_RET_LOG` mode** over the P2 corpus and about 20 reference programs per language, then a KILL-mode re-run.
- **The image-volume spike (i)–(iii)** on the `xl-spike` k3s if the pack image includes `linux/arm64`, otherwise on environment A's pinned k3s (see task 3). Never on GitHub Actions.
- **The combined MI-10 verdict** and a "proposed ADR-0030 deltas" list for m3-03.

**Out**
- **Anything on production.** Only read-only `ssh sujaykumar-vps` facts are allowed.
- **Committing any harness.** Code goes only into the throwaway scratch repo, and the owner deletes that repo.
- **Timing conclusions.** GitHub Actions runs linux-azure; the second VPS is a different guest from production.
- **Latency, CV, canary thresholds and calibration:** A8 on production ([mi-10](sprint-mi-10.md), [m3-15](sprint-m3-15.md)).
- **The WIF spike:** [spk-03](sprint-spk-03.md).
- **The evalpack ImagePolicy and the judge HelmRelease:** [m3-07](sprint-m3-07.md).
- **Changing the pack image or its CI:** [m3-02](sprint-m3-02.md) (E) acts on this sprint's findings.
- **Accepting ADR-0030** ([m3-03](sprint-m3-03.md)) and **amending ADR-0027** (m3-07 does that if the fallback is chosen).

## Tasks

### 1 · P3 amd64 environment [H (+ O before launch)]

Pick, in this order ([t3 §9 Environment](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead)):

- **A. The owner's second VPS** if it exists, is amd64 and is empty. It's closest to production: a generic noble kernel on a KVM guest.
  - The owner gives SSH access before launch (an SSH alias named in the launch message).
  - Run `apt full-upgrade` and reboot onto the current noble kernel, mirroring H0: `core_pattern=core`, `suid_dumpable=0`, apport masked.
  - Add the sandbox sysctls and load spk-01's AppArmor profile.
  - Install go1.26.8 (checksummed), `g++`, `python3`, PGDG PG 18, `auditd` (for `ausearch`/`ausyscall`) and the `seccomp` package (for `scmp_sys_resolver`).
  - Record `uname -r`, `lscpu`, `apparmor_restrict_unprivileged_userns` and `vm.mmap_rnd_bits`. Set `mmap_rnd_bits` to **32** if it's lower, to mirror production.
  - **k3s is optional for P3.** The default is the no-k3s path: the supervisor runs as root in a delegated scope (`systemd-run --scope -p Delegate=yes`), under `aa-exec -p xlearn-runner`, with the **amd64 pod seccomp profile** (task 2) loaded on it by the spike launcher through libseccomp. If time allows, install the pinned k3s the way `host-bootstrap.sh` does and run the supervisor in spk-01's positive pod instead. That's the closest match to production, and it can host task 3 too. If the pack image is single-arch amd64 (entry gates), k3s on A is **required**, because task 3 then runs there.
  - The owner **reimages it afterwards**, before it becomes the backup target: an owner follow-up the session records, not a wait. The runner never runs on the backup VPS (D21).
- **B. Otherwise, GitHub Actions** `ubuntu-24.04` (amd64).
  - Use a **private** scratch repo, `sujaykumarsuman/xlearn-spike-scratch`, which the session creates (`gh repo create … --private`; approved by the launch, D40), holding only throwaway code.
  - Use a `workflow_dispatch` job. Its first steps **assert** `apparmor_restrict_unprivileged_userns=1` (set it if it isn't) and set `mmap_rnd_bits=32`.
  - Install go1.26.8, `g++`, `python3` and PGDG PG 18, load the AppArmor profile, and run the supervisor under `sudo systemd-run --scope -p Delegate=yes`.
  - Install `auditd` (for `ausearch`/`ausyscall`) and the `seccomp` package (for `scmp_sys_resolver`) too, as on A.
  - Upload the logs as an artifact with `retention-days: 1`.
  - The kernel is linux-azure, so the results are functional only, with no timing.
  - **Environment B serves P3 only.** It never runs the image-volume check, because that needs the pull secret, and the secret never goes to CI.
  - **Never push evalpack content, xlearn secrets, the PAT or production facts to it.**

### 2 · P3 replay (2 h) [H]

- **Supervisor and binaries.** Cross-compile the spk-01 supervisor (`GOARCH=amd64`) and bring the same profiles and corpus.
- **The amd64 pod-level seccomp profile** (the file [mi-09](sprint-mi-09.md) ships). Re-target spk-01's pod profile to `architectures: [SCMP_ARCH_X86_64, SCMP_ARCH_X86, SCMP_ARCH_X32]`, the baseline inherited from RuntimeDefault, and load it on the supervisor for the whole replay.
  - The supervisor's own needs must pass: jail setup, `pivot_root`, the cgroup writes, `clone3`/`CLONE_INTO_CGROUP`.
  - spk-01's negative probes must still fail on amd64: `unshare -U`, `clone3(CLONE_NEWUSER)`, `fsopen`/`open_tree` + `move_mount`, `socket(SCTP)`, `socket(AF_NETLINK)`.
  - **Also test an x86_64-only variant** (`architectures: [SCMP_ARCH_X86_64]`). The three-arch baseline keeps the ia32 and x32 compat syscall entry points open to every process in the runner pod, learner code included. The static Go, C++ and Python targets shouldn't need them. Re-run the supervisor's needs, the negative probes and the reference set under the variant, and record whether anything breaks.
  - Record both files verbatim. Don't fix the architectures here: §16.4 hands the choice to [m3-03](sprint-m3-03.md) as a proposed ADR-0030 delta, and the mi-09 hand-off names both files and the recommendation.
- **Reference programs.** Write about **20 per language** in `~/xl-spike/refs/{go,cpp,py}/`, covering typical DSA shapes:
  - hash maps; two pointers; sorting;
  - BFS/DFS on grids; heaps; DP tables;
  - deep recursion (Python's `sys.setrecursionlimit`); big input and big output; string building;
  - Python's common stdlib (`collections`, `heapq`, `bisect`, `itertools`, `functools`, `math`).

  These are throwaway. **Never copy them from `xlearn-evalpack`**, because its content is private.
- **go-race / TSAN.**
  - Build with `go test -c -race` and run the binary in the jail under `mmap_rnd_bits=32`.
  - Does the Go 1.26 race runtime fail with "unexpected memory mapping", or re-exec itself with `personality(ADDR_NO_RANDOMIZE)`?
  - Which syscalls does it need (`personality`, `execve` of itself)?
  - **Pass:** race detection works with a **per-process** ASLR policy. The host sysctl is never lowered.
  - **Fail:** try a per-process `setarch -R` launcher. If it still fails, go-race is out of the pilot ([p-01](sprint-p-01.md) gate).
- **postgres.** Boot PG 18 in the jail on a Unix socket and run the SQL balloon once. That's the amd64 check of spk-01's P1b/P2 rows.
- **Allowlists.**
  - Run each profile's exec jail with the default action **`SECCOMP_RET_LOG`** over the P2 corpus plus the references.
  - **Collect the logged syscalls through auditd,** so no record is dropped. While auditd runs, seccomp `type=1326` records go to `/var/log/audit/audit.log`, **not** to `journalctl -k` / `dmesg`. Without auditd they fall back to a rate-limited printk path, which drops records under a RET_LOG flood.
    - Before the runs: `auditctl -b 8192 -r 0` (a bigger backlog, no rate limit).
    - Collect with `ausearch -m SECCOMP -i --start <run start>`. `-i` maps syscall numbers to names for the record's arch.
    - Afterwards, `auditctl -s` must show `lost 0`. If it doesn't, raise the backlog and re-run.
    - If you can't run auditd, keep it stopped, read `type=1326` from `journalctl -k`, and map names with `ausyscall` or `scmp_sys_resolver`. Treat that data as possibly incomplete.
    - The KILL re-run still catches gaps, but it shouldn't be the first line of defence.
  - Take the union per profile: `go`, `cpp` (`g++ -std=gnu++20 -O2`, static, per [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them)), `python`, and `go-race` if it works.
  - Also record the compile-jail syscall sets. Compile seccomp is ENOSYS-default, so they inform the profiles but don't gate.
- **KILL re-run.** Switch to KILL-default with the generated allowlists and re-run the P2 balloon/fork subset plus every reference.
  - **Pass:** **0 unexpected SIGSYS**, MLE still classified 100%, and the references produce the expected output.
  - A SIGSYS becomes an allowlist entry only if the syscall is safe, and every addition is justified in the results.

### 3 · Image-volume spike (≈ 1.5 h) [H]

- **Where it runs.** Image volumes are on by default since Kubernetes 1.36 ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack)). The entry gate settled the platforms:
  - **The pack image includes `linux/arm64`:** start `xl-spike` and use its k3s (v1.36.4).
  - **Single-arch amd64:** the arm64 VM can't pull it ("no match for platform"). Run (i)–(iii) on **environment A** with the pinned k3s installed as `host-bootstrap.sh` does it, and flag to [m3-02](sprint-m3-02.md) that the pack should be published as a multi-arch index. It's `FROM scratch` data, so the extra platform is free.
  - **Never on GitHub Actions.** The pull secret never goes into a CI runner or the scratch repo. With neither the VM nor A available, task 3 is ⛔ (entry gates).
- **The pull secret.** It gets into the throwaway k3s **without ever being printed or written to disk outside it**. The session runs (approved by the launch, D40) `sops -d ../infra/apps/secrets/xlearn-evalpack-pull.enc.yaml | multipass exec xl-spike -- sudo k3s kubectl apply -f -` (on A: `| ssh <env-A> 'sudo k3s kubectl apply -f -'`), which creates it in namespace `xlearn`. The VM is deleted in task 5, and A is reimaged.
- **Record the kubelet settings** from `configz`: `featureGates.KubeletEnsureSecretPulledImages` (beta, on by default since 1.35) and `imagePullCredentialsVerificationPolicy` (default `NeverVerifyPreloadedImages`). **The image must be pulled by the kubelet with the secret, never with `ctr pull` or import.** A preloaded image is exempt from verification, which would make (ii) meaningless.

| # | Check | Pass |
|---|---|---|
| i | A judge-like pod in namespace `xlearn`, labelled PSA `enforce=baseline` as production will be, with `imagePullSecrets: [xlearn-evalpack-pull]` and volume `image: {reference: ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0, pullPolicy: IfNotPresent}` mounted read-only at `/evalpack`, with `EVALPACK_DIR=/evalpack` | pod Running; `/evalpack/manifest.json` readable (the image holds only `build/`; `pack.json` is a repo source file, not in the image); `touch /evalpack/x` fails; baseline PSA admits it |
| ii | A pod in another namespace (`probe`), **without** the pull secret, referencing the same image volume, once with `pullPolicy: IfNotPresent` and once with `Never` | the pod **can't start** (credential verification refuses the cached image); record the exact event |
| ii-b | Restart k3s (`systemctl restart k3s`), because the Oct 24 host window restarts it, then repeat (ii) | still refused (the kubelet's pull records persist) |
| iii | The fallback with chart 0.3.0: `helm template ../infra/charts/project` with the `initContainers` knob, an emptyDir in `extraVolumes`, and judge's mount at `/evalpack`. The pack is `FROM scratch`, so the initContainer can't run inside it. Test **variant A**: an image built locally in the VM (never pushed) = the pack layer plus a static copier as ENTRYPOINT, which copies into the emptyDir. Test **variant B**: an `oras`/`crane` initContainer that pulls with the pull secret mounted | A and B each work or fail, with their errors; recommend A (no credential inside judge's pod) |

- **Verdict:**
  - **(i) and (ii) pass:** image-volume **GO**.
  - **(ii) fails:** test once more with `imagePullCredentialsVerificationPolicy: AlwaysVerify`, set through the kubelet config. If that fixes it, hand the setting to [mi-09](sprint-mi-09.md) to join the L23 kubelet args in the host window, and record **GO with that setting**.
  - **(i) fails:** **fallback chosen**: variant A, else B, else ORAS as t1's last resort.
- **How a fallback maps onto the M3 line.** The [rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist) line reads "the image-volume spike **GO** (MI-10)". A fallback from the **pre-decided chain** (variant A, else B, else ORAS; [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack), ADR-0027) is the session's call under D40 and satisfies it; `status.md` words it that way, for example "image-volume: fallback A (pre-decided chain, D40, date)". If every variant fails, that's outside the pre-decided paths: record the options and a recommendation in §16.3, mark the line ⛔ "needs owner decision" in `status.md`, and still land the results PR.

### 4 · Report (1 h) [X] — docs PR `docs(v2): sandbox spike P3 + image-volume results (MI-10)`

- **In [`../research/t3-sandbox.md`](../research/t3-sandbox.md), under spk-01's `## 16. Spike results (MI-10)`, add three sections:**
  - **§16.2 P3 amd64 replay (spk-02):**
    - the environment, A or B, with its kernel and `mmap_rnd_bits`;
    - the TSAN result and the ASLR policy needed;
    - PG on amd64;
    - **the amd64 pod-level seccomp profile, verbatim**, with its architectures and the negative-probe results. mi-09 ships this file;
    - **per-profile allowlists as sorted syscall-name lists** (go, cpp, python, and go-race if it works), with sizes and the justified additions;
    - the compile-jail sets;
    - the KILL re-run numbers (unexpected SIGSYS = 0?).
  - **§16.3 Image volume (spk-02):**
    - the kubelet `configz` values;
    - the (i), (ii), (ii-b) and (iii) rows with their exact events;
    - the pack image's platforms;
    - the verdict: GO, GO with `AlwaysVerify`, the fallback variant (from the pre-decided chain, D40), or "needs owner decision" with the options and a recommendation if every variant failed.
  - **§16.4 MI-10 verdict and proposed ADR-0030 deltas:**
    - **"Spike P0–P3 GO / NO-GO; image volume GO / fallback <variant> (pre-decided chain, D40) / needs owner decision"**, the line the M3 checklist reads;
    - the chosen jail mechanism;
    - the bullet list of what m3-03 folds into ADR-0030 when accepting it: mechanism, SETPCAP, the spawn path, the ASLR policy, the allowlists' home, the host-file diffs, and **the pod profile's architectures** (x86_64-only if nothing broke, else the three-arch baseline, with the evidence);
    - if the fallback was chosen, "ADR-0027's image-volume line needs amending in m3-07".
- **In [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack),** add one line to the "Platform check" list: *Result (spk-02, date): … — see t3 §16.3.*
- **Leave ADR-0030 and ADR-0027 untouched.** [m3-03](sprint-m3-03.md) and [m3-07](sprint-m3-07.md) own those changes.
- **`docs/v2/status.md`:**
  - the MI-10 row ✅ (or ⛔) with the verdict worded as the M3 checklist reads it: "Spike P0–P3 GO and image-volume GO", or "… image-volume: fallback A (pre-decided chain, D40, date)", or ⛔ "needs owner decision" with the reason. Tick the M3 checklist line in `status.md` (if it tracks it) **only** for a GO or a fallback from the pre-decided chain;
  - the Sprint board row for spk-02;
  - Decisions log lines: the ASLR policy; the allowlist home (pod seccomp → mi-09, per-case → m3-04); the pod profile's architectures (proposed ADR-0030 delta); image volume GO or the fallback (pre-decided chain, D40); the kubelet policy if changed; any multi-arch pack request to m3-02 and mi-07;
  - hand-off notes to m3-03, mi-09, m3-04, p-01, m3-07 and m3-02.

### 5 · Teardown [H]

- **`xl-spike`:** if [spk-03](sprint-spk-03.md) runs on it this week (Fri), hand it over and spk-03 deletes it. Otherwise run `multipass delete --purge xl-spike`.
- **`~/xl-spike/`:** `rm -rf`, after the results PR has merged. Also delete any locally built pack variant images.
- **Environment A:** the agent removes the SSH alias it was given. Reimaging the second VPS is hPanel work, so it's an **owner follow-up**: record it in `status.md` → Open owner items; don't wait for it.
- **Environment B:** deleting `xlearn-spike-scratch` is permanent, so it stays the owner's action: record it in `status.md` → Open owner items as an **owner follow-up**; don't wait for it.
- Record in `status.md` what was deleted and when.

## Acceptance criteria

- [ ] **Q-C answered:**
  - the TSAN / go-race result under `mmap_rnd_bits=32`, with the ASLR policy it needs (or "out of the pilot");
  - postgres runs in the jail on amd64.
- [ ] **amd64 allowlists** are generated for `go`, `cpp` and `python` (and `go-race` if it works), with **0 unexpected SIGSYS** under KILL over the corpus subset and every reference.
- [ ] **The amd64 pod-level seccomp profile** (the x86_64/x86/x32 baseline) is validated under the supervisor, spk-01's negative probes still fail with it, the x86_64-only variant is tested too, and both files are recorded verbatim for mi-09, with the architecture choice proposed to m3-03.
- [ ] **Image volume:**
  - (i) mounts read-only;
  - (ii) and (ii-b) refuse a pod without the secret, or the `AlwaysVerify` setting is handed to mi-09;
  - (iii) the fallback variant is tested and recommended;
  - the verdict is **image-volume GO or a named fallback from the pre-decided chain** (D40); if every variant failed, ⛔ "needs owner decision" with the options and a recommendation.
- [ ] The results are in t3 §16.2–16.4 and the t1 §3.3 pointer, the MI-10 verdict (worded as the M3 checklist reads it) is in `status.md`, and the hand-offs are noted.
- [ ] **Teardown:** the VM is deleted or handed to spk-03, `~/xl-spike/` is gone and the SSH alias is removed. The owner follow-up (reimage the second VPS, **or** delete the scratch repo) is recorded in `status.md`.

## Release

**No merge: a throwaway spike.** Nothing but results is kept. The only merge is the **docs PR** (t3 §16.2–16.4, the t1 §3.3 pointer and `status.md`), squash-merged on CI green with no owner stop (D40), and it doesn't ship in any tag. A result outside the pre-decided paths still lands, marked ⛔ "needs owner decision". There's no infra PR, no evalpack tag, and nothing touches production.

## Definition of Done

- The results are recorded and the docs PR is merged.
- No production change was made.
- The throwaway environments are torn down per task 5 (the owner follow-up recorded).
- Statuses are updated in this file and in [`../status.md`](../status.md).
- The hard stop was respected. Unfinished rows are marked "not run (time box)", and unknown never counts as GO.

## Risks / watch-outs

- **No amd64 box means GitHub Actions only for P3.** That's functional, not timing: linux-azure isn't the production guest. Say so in §16.2.
- **A single-arch pack image can't be pulled on the arm64 VM, and it's the likely case** (mi-07 sets no platform list). The image-volume check then needs environment A. With neither a multi-arch image nor A, task 3 is blocked: ask for a multi-arch rebuild (mi-07's workflow, m3-02 for the real pipeline). Never move the pull secret into CI to work around it.
- **Seccomp log loss.** RET_LOG floods can drop records on the printk path. Collect through auditd with a raised backlog and no rate limit, and check `lost 0`.
- **Preloaded images skip credential verification** (`NeverVerifyPreloadedImages`). Only a kubelet-driven pull makes (ii) meaningful.
- **Secret handling.** The pull secret (the PAT) exists only inside a throwaway k3s: the `xl-spike` VM, or environment A's k3s. It's piped there from SOPS, never echoed, never written to disk outside that k3s, never committed, and never put in the scratch repo, a GitHub Actions secret or a CI runner. It goes when the VM is deleted or A is reimaged.
- **Private content.** Never copy `xlearn-evalpack` content into the scratch repo or the reference set.
- **Allowlist over-fitting.** A syscall added only because a reference hit it must still be safe. Unsafe additions (new mount API, `bpf`, `io_uring`, `userfaultfd`, keyring) are never allowlisted; the reference changes instead.
- **The second VPS must be reimaged** before it becomes the backup target. The runner never runs on the backup VPS (D21).
