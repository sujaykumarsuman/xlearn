# Prompt — Sprint spk-02 · P3 amd64 replay + image-volume spike (throwaway)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the xlearn repo root.
> **Plan:** [`../sprints/sprint-spk-02.md`](../sprints/sprint-spk-02.md) · **Milestone:** MI (rollout step MI-10, part 2) · **Prereqs:** [spk-01](../sprints/sprint-spk-01.md), [mi-07](../sprints/sprint-mi-07.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] **Optional, preferred: environment A.** If your second VPS exists, is amd64 and is empty, add an SSH alias for it on this Mac (for example `spike-a`) and name the alias in your launch message. Launching approves its full-upgrade and reboot now, and you reimage it afterwards (the session records that follow-up). A is also the only place the image-volume check can run if the pack image is single-arch amd64. Without an alias, the session uses environment B (GitHub Actions, P3 only).

## Read first

- [`../sprints/sprint-spk-02.md`](../sprints/sprint-spk-02.md): the plan. It holds the image-volume table (i)–(iii), the verdict rules, the report spec and the teardown this prompt follows.
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and land-and-sync.
- **spk-01's results:** [`../research/t3-sandbox.md`](../research/t3-sandbox.md) §16.1 (the mechanism chosen, the host files, the VAP diff), and the hand-off notes in [`../status.md`](../status.md).
- **The P3 script:** [t3 §9](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead) (the Q-C questions, "Part 2 (P3)", the P3 row, and what's thrown away).
- **Profiles and ASLR:** [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them), including the "ASLR policy (corrected)" note right below the table.
- **Production facts:** [t3 §15](../research/t3-sandbox.md#15-s0-results-read-only-production-facts-2026-09-24-owner-approved) (`mmap_rnd_bits=32`, the kernel) and [t3 §12](../research/t3-sandbox.md#12-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D20 Go/C++/Python; D21 never on the backup VPS).
- **The image volume:** [t1 §3.3 Private eval pack](../research/t1-content-data-model.md#33-private-eval-pack) (the flow and the "Platform check") and [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (the eval pack: judge-only image volume, fallbacks).
- **Release streams:** [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams) (the evalpack stream, for context only; this sprint tags nothing).
- **In the infra repo:**
  - `../infra/charts/project/`: chart 0.3.0's `initContainers` knob and `extraVolumes`;
  - `../infra/apps/secrets/xlearn-evalpack-pull.enc.yaml`: the pull secret, which is **never** printed;
  - `../infra/hack/host-bootstrap.sh`: the pinned k3s install.

## Context

**Where things stand.** spk-01 answered Q-A and Q-B on an arm64 multipass VM, `xl-spike`, and left behind:
- the VM itself (stopped);
- a throwaway supervisor, profiles and corpus in `~/xl-spike/`, outside every repo.

**This session: Q-C on amd64.**
- Does the Go race runtime (TSAN) work in the jail under production's `vm.mmap_rnd_bits=32`, and what ASLR policy does it need?
- What are the per-language **amd64 exec allowlists** for Go, C++ and Python (D20)?

**Plus the eval-pack image-volume check.** Does judge's read-only image volume of the private pack mount with the pull secret, does a pod *without* the secret get refused the cached image, and does the initContainer fallback work?

**Who uses the answers.**
- [m3-03](../sprints/sprint-m3-03.md) accepts ADR-0030.
- [mi-09](../sprints/sprint-mi-09.md) scripts the amd64 pod seccomp profile for the Sat 2026-10-24 host window.
- [m3-04](../sprints/sprint-m3-04.md) builds the runner profiles on the allowlists.
- [p-01](../sprints/sprint-p-01.md) gates go-race on the TSAN result.
- [m3-07](../sprints/sprint-m3-07.md) mounts the pack.
- The M3 hard checklist's first line is "Spike P0–P3 **GO** and the image-volume spike **GO**" ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] t3 §16.1 exists (spk-01 merged). `multipass list` shows `xl-spike` (stopped), and `~/xl-spike/` holds the supervisor, profiles and corpus.
- [ ] MI-9 is done:
  - `status.md` records mi-07 ✅ and the PAT expiry;
  - `../infra/apps/secrets/xlearn-evalpack-pull.enc.yaml` exists on `main`;
  - an **unauthenticated** manifest request for `ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0` returns 401/403. Don't try an authenticated request from the laptop.
- [ ] The environment: **A** if the launch message names an SSH alias for the second VPS (the before-launch block), else **B**, a private GitHub Actions scratch repo you create yourself (the launch approves it, D40). B serves **P3 only**; it never runs the image-volume check.
- [ ] **The image-volume check has a compliant environment:** the pack image includes `linux/arm64` (so the arm64 `xl-spike` can pull it), **or** A exists. mi-07 builds `0.1.0` with no platform list, which gives a single-arch amd64 image, so expect arm64 to be missing unless the build was made multi-arch.
  - Check without an authenticated registry request from the laptop: `git -C ../xlearn-evalpack show v0.1.0:.github/workflows/build.yml | grep -n platforms`, and the build run's log (`gh run view -R sujaykumarsuman/xlearn-evalpack <run-id> --log`). If both are inconclusive, treat the image as single-arch amd64.
  - If neither holds, run P3 anyway, but mark step 3 ⛔ (the image-volume line stays red) and report that the fix is a multi-arch rebuild in the evalpack repo (a new `0.1.x` tag from mi-07's workflow with `platforms: linux/amd64,linux/arm64`). Never move the pull secret into CI to work around it.
- [ ] Chart 0.3.0 is merged ([mi-01](../sprints/sprint-mi-01.md)), or its PR branch is available for `helm template`.

## Do this (in order)

1. **[H] P3 environment.**
   - **A (preferred), the second VPS:**
     - Over the SSH alias from the launch message, run `apt full-upgrade` and reboot. Mirror H0: `core_pattern=core`, `suid_dumpable=0`, apport masked.
     - Add the sandbox sysctls, and load spk-01's final AppArmor profile.
     - Record `uname -r`, `lscpu`, `apparmor_restrict_unprivileged_userns` and `vm.mmap_rnd_bits`. Set `mmap_rnd_bits=32` if it's lower.
     - Install go1.26.8 (checksummed), `g++`, `python3`, PGDG PG 18, `auditd` (for `ausearch`/`ausyscall`) and the `seccomp` package (for `scmp_sys_resolver`).
     - If the pack image is single-arch amd64, install the pinned k3s as `host-bootstrap.sh` does: step 3 runs here.
   - **B, GitHub Actions:**
     - Create the private repo yourself: `gh repo create sujaykumarsuman/xlearn-spike-scratch --private` (approved by the launch, D40). Push only throwaway code: the supervisor, the references and the workflow.
     - Add a `workflow_dispatch` job on `ubuntu-24.04` that first asserts or sets `apparmor_restrict_unprivileged_userns=1` and `mmap_rnd_bits=32`, then installs the same toolchain, `auditd` and `seccomp` included.
     - B is for P3 only. No pull secret, PAT, evalpack content or production fact ever goes into the scratch repo or its runner.
     - Upload the logs as an artifact with `retention-days: 1`.
     - The kernel is linux-azure, so the results are functional only.
   - **On either environment,** the supervisor runs as root in `systemd-run --scope -p Delegate=yes`, under `aa-exec -p xlearn-runner`, with the amd64 pod seccomp profile (next step) loaded through libseccomp. If time allows on A, run it instead inside spk-01's positive pod on a pinned k3s. That's closest to production.
2. **[H] P3 replay (2 h).**
   - **Build.** Cross-compile the spk-01 supervisor with `GOARCH=amd64`.
   - **The amd64 pod seccomp profile,** which is the file mi-09 ships. Re-target spk-01's pod profile to `architectures: [SCMP_ARCH_X86_64, SCMP_ARCH_X86, SCMP_ARCH_X32]` (the baseline inherited from RuntimeDefault) and load it on the supervisor for the whole replay.
     - The supervisor's jail setup must pass.
     - spk-01's negative probes must still fail: `unshare -U`, `clone3(CLONE_NEWUSER)`, `fsopen`/`open_tree` + `move_mount`, `socket(SCTP)`, `socket(AF_NETLINK)`.
     - **Also test an x86_64-only variant** (`architectures: [SCMP_ARCH_X86_64]`). The baseline leaves the ia32 and x32 compat entry points open to every process in the runner pod, learner code included, and the static Go/C++/Python targets shouldn't need them. Re-run the jail setup, the negative probes and the references under it, and record whether anything breaks.
     - Record both files verbatim. Don't pick the architectures yourself: §16.4 proposes the choice to m3-03 as an ADR-0030 delta.
   - **References.** Write about 20 throwaway reference programs per language in `~/xl-spike/refs/{go,cpp,py}/`: hash maps, two pointers, sorting, BFS/DFS, heaps, DP, deep recursion, big input and output, string building, and Python's common stdlib. **Never copy any from `xlearn-evalpack`.**
   - **TSAN.** Run a `go test -c -race` binary in the jail under `mmap_rnd_bits=32`, and record what it does:
     - fails with "unexpected memory mapping";
     - or re-execs itself with `personality(ADDR_NO_RANDOMIZE)`, and which syscalls that needs.

     If it needs a policy, allow `personality` for **go-race processes only**. If that isn't enough, try a per-process `setarch -R` launcher. **Never lower the host sysctl.** If it still fails, go-race is out of the pilot.
   - **Postgres.** Boot it in the jail and run the SQL balloon once.
   - **Allowlists.**
     - Run each profile's exec jail with default action `SECCOMP_RET_LOG` over the P2 corpus and all references.
     - **Collect through auditd.** While auditd runs, seccomp `type=1326` records go to `/var/log/audit/audit.log`, not to `journalctl -k` / `dmesg`, and without it the printk path is rate-limited and drops records under a RET_LOG flood. So first run `auditctl -b 8192 -r 0` (a bigger backlog, no rate limit), then collect with `ausearch -m SECCOMP -i --start <run start>` (`-i` maps numbers to names for the record's arch). `auditctl -s` must show `lost 0` afterwards; if it doesn't, raise the backlog and re-run.
     - If auditd can't run, keep it stopped, read `type=1326` from `journalctl -k`, map names with `ausyscall` or `scmp_sys_resolver`, and treat that data as possibly incomplete.
     - Take the union per profile: `go`, `cpp` (`g++ -std=gnu++20 -O2`, static), `python`, and `go-race` if it works.
     - Record the compile-jail sets too.
   - **KILL re-run.** Switch to KILL-default and re-run the P2 balloon/fork subset plus every reference.
     - The target is **0 unexpected SIGSYS**, with MLE still classified and the outputs correct.
     - Allowlist a syscall only if it's safe, and justify it. Never allowlist the new mount API, `bpf`, `io_uring`, `userfaultfd` or the keyring calls.
3. **[H] Image-volume spike (≈ 1.5 h).**
   - **Where it runs** (settled by the entry gate). If the pack image includes `linux/arm64`, start the VM (`multipass start xl-spike`) and run every `kubectl` through `multipass exec xl-spike -- sudo k3s kubectl`, because the laptop's context tunnels to production. If it's single-arch amd64, the arm64 VM can't pull it ("no match for platform"): run this step on **environment A** with the pinned k3s, using `ssh <env-A> 'sudo k3s kubectl …'` in place of the `multipass exec` form, and note the multi-arch request for m3-02 and mi-07. **Never in GitHub Actions.** With neither, this step is ⛔.
   - **Pull secret.** Run it yourself (approved by the launch, D40): `sops -d ../infra/apps/secrets/xlearn-evalpack-pull.enc.yaml | multipass exec xl-spike -- sudo k3s kubectl apply -f -` (on A: `| ssh <env-A> 'sudo k3s kubectl apply -f -'`). It creates namespace `xlearn`'s secret; create the namespace first. **Never print, `cat` or save the decrypted secret.**
   - **Record the kubelet settings** from `configz` (`k3s kubectl get --raw /api/v1/nodes/<node>/proxy/configz`): the `KubeletEnsureSecretPulledImages` gate and `imagePullCredentialsVerificationPolicy`.
   - **(i)** A judge-like pod in `xlearn`, with the namespace labelled `pod-security.kubernetes.io/enforce=baseline`:
     - `imagePullSecrets: [xlearn-evalpack-pull]`;
     - volume `image: {reference: ghcr.io/sujaykumarsuman/xlearn-evalpack:0.1.0, pullPolicy: IfNotPresent}`, mounted `readOnly` at `/evalpack`, with `EVALPACK_DIR=/evalpack`;
     - a small busybox container that lists `/evalpack`, cats `/evalpack/manifest.json`, and fails `touch /evalpack/x`. The image holds only `build/` (`/manifest.json`); `pack.json` is a repo source file, so don't expect it.

     **The kubelet must do the pull. Never `ctr pull` or import the pack,** because a preloaded image skips verification.
   - **(ii)** In namespace `probe`, a pod **without** the secret references the same image volume, once with `IfNotPresent` and once with `Never`. It must not start. Record the event text.
   - **(ii-b)** Run `sudo systemctl restart k3s` and repeat (ii).
   - **If (ii) passes the pod through:** set `imagePullCredentialsVerificationPolicy: AlwaysVerify` through a kubelet config drop-in on the VM, and retest. If that fixes it, it goes to mi-09's L23 batch.
   - **(iii)** Render chart 0.3.0 with `helm template ../infra/charts/project` using the `initContainers` knob, an emptyDir in `extraVolumes`, and the mount at `/evalpack`.
     - **Variant A:** a locally built image (never pushed) = the pack layer plus a static copier as ENTRYPOINT, which copies into the emptyDir.
     - **Variant B:** an `oras`/`crane` initContainer pulling with the pull secret mounted.
     - Record both. Recommend A: no credential inside judge's pod.
   - **Verdict:**
     - (i) and (ii) pass → **GO**;
     - (ii) fixed only by `AlwaysVerify` → **GO with setting**;
     - (i) fails → **fallback: A, else B, else ORAS**.
   - **A fallback and the M3 line.** The rollout §5 line reads "the image-volume spike **GO** (MI-10)". A fallback from the **pre-decided chain** (A, else B, else ORAS) is your call under D40 and satisfies it, worded in `status.md` as, for example, "image-volume: fallback A (pre-decided chain, D40, date)". If every variant fails, that's outside the pre-decided paths: record the options and a recommendation in §16.3, mark the line ⛔ "needs owner decision" in `status.md`, and still land the results PR. Don't ask and don't wait.
4. **[X] Report (1 h, docs PR).**
   - Branch `docs/spk-02-results`.
   - Under `## 16. Spike results (MI-10)` in `docs/v2/research/t3-sandbox.md`, add:
     - **§16.2 P3 amd64 replay (spk-02):** the environment and kernel; `mmap_rnd_bits`; TSAN and the ASLR policy; PG; **the amd64 pod seccomp profile verbatim**, with its negative-probe results (mi-09 ships it); **the per-profile allowlists as sorted syscall-name lists**, with sizes and justified additions; the compile sets; the KILL re-run numbers.
     - **§16.3 Image volume (spk-02):** `configz`; the environment used (VM or A); the (i), (ii), (ii-b) and (iii) rows with their exact events; the pack's platforms; the verdict (a fallback noted as the pre-decided chain, D40; if every variant failed, the options and a recommendation).
     - **§16.4 MI-10 verdict and proposed ADR-0030 deltas:** the line "Spike P0–P3 GO/NO-GO; image volume GO / fallback <variant> (pre-decided chain, D40) / needs owner decision", then a bullet list for m3-03 (mechanism, SETPCAP, spawn path, ASLR policy, where the allowlists live, host-file diffs, and the pod profile's architectures: x86_64-only if nothing broke, else the three-arch baseline). If the fallback was chosen, add "ADR-0027's image-volume line needs amending (m3-07)".
   - Add one "Result (spk-02, date) … see t3 §16.3" line under t1 §3.3's "Platform check".
   - Update `docs/v2/status.md` (below).
   - Commit `docs(v2): sandbox spike P3 + image-volume results (MI-10)`, open the PR, and merge it once CI is green (see Ship).
5. **[H] Teardown** (after the docs PR merges).
   - **The VM:** if spk-03 is booked on `xl-spike` this week, hand it over and record that. Otherwise run `multipass delete --purge xl-spike`.
   - **The work directory:** `rm -rf ~/xl-spike`, plus any local pack-variant images.
   - **Environment A:** remove the SSH alias you were given, and record the reimage in `status.md` → Open owner items (hPanel work: an owner follow-up, not a wait).
   - **Environment B:** record the deletion of `xlearn-spike-scratch` in `status.md` → Open owner items. Deleting a repo is permanent, so it stays the owner's action; don't wait for it.
   - Record what was deleted, and when, in `status.md`.

## Constraints

- **Throwaway.**
  - No harness, reference, manifest or log is committed to xlearn, infra or evalpack.
  - The scratch repo holds throwaway code only, and the owner deletes it.
  - The only commit is the results docs PR.
- **Production is off-limits.** Read-only `ssh vps` facts only. No infra PR, no GitOps change, no evalpack tag, and **no `kubectl` on the laptop** (its context tunnels to production). Inside the VM or scratch environment, `kubectl apply` is fine: it's throwaway.
- **Secrets and private content.**
  - The pull secret (the PAT) exists only inside a throwaway k3s: the `xl-spike` VM, or environment A's k3s. It's piped there from SOPS, never echoed, never written to disk outside that k3s, and never put in the scratch repo, a GitHub Actions secret or any CI runner. It goes when the VM is deleted or A is reimaged.
  - Never copy `xlearn-evalpack` content into the scratch repo, the references or the results. Record only structure and sizes.
- **Hard stop, counted as effort, not elapsed days.** This session's share of the 10 h P0–P3 core is 3 h (P3 2 h + report 1 h); the image-volume check is about 1.5 h more, outside the core. The Thu→Fri calendar is only the booking window. At the stop, mark the unfinished rows "not run (time box)". Unknown never counts as GO.
- **No timing conclusions** on GitHub Actions or the second VPS.
- **ASLR.** A per-process policy only. Never lower the host `mmap_rnd_bits`.
- **Leave the ADRs untouched.** ADR-0030 (m3-03) and ADR-0027 (m3-07) are changed by their consumers. No new ADR is expected. If one is needed, check peers' numbers first (`gh pr list --state all`, `git worktree list`, `ListAgents`).
- **No alerting** tooling (D34).
- **D21.** The runner never runs on the backup VPS, so the second VPS must be reimaged before it becomes one.

## Deliverables

- `docs/v2/research/t3-sandbox.md` §16.2–§16.4: the P3 results, the per-profile amd64 allowlists, the image-volume results, the MI-10 verdict, and the proposed ADR-0030 deltas.
- The t1 §3.3 result pointer.
- `docs/v2/status.md`: the MI-10 verdict, the Sprint board row, the Decisions log, the hand-offs and the teardown record.
- One merged docs PR.
- The throwaway environments torn down, or the VM handed to spk-03.

## Update status

- In [`../sprints/sprint-spk-02.md`](../sprints/sprint-spk-02.md), move each task ⬜ → 🔄 → ✅, or ⛔ with a reason. Set _Overall_ ✅ at the end.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for spk-02;
  - the **MI rows**: MI-10 ✅ / ⛔, with the verdict worded as the M3 checklist reads it ("Spike P0–P3 GO and image-volume GO", or "… image-volume: fallback A (pre-decided chain, D40, date)", or ⛔ "needs owner decision"). Tick the M3 checklist line, if `status.md` tracks it, **only** for a GO or a pre-decided fallback;
  - the owner calendar event `ev-spike-week` done, or continuing into spk-03;
  - **Decisions log** lines: the go-race ASLR policy; where the allowlists live (pod seccomp → mi-09, per-case → m3-04); the pod profile's architectures (proposed ADR-0030 delta → m3-03); image volume GO or fallback (pre-decided chain, D40); any kubelet policy change (→ mi-09); any multi-arch pack request (→ m3-02, mi-07);
  - hand-off notes to m3-03, mi-09, m3-04, p-01, m3-07 and m3-02;
  - the teardown record, and the owner follow-up (reimage the second VPS, or delete the scratch repo) under Open owner items.

## Done when (acceptance)

- [ ] **Q-C answered:** the TSAN / go-race result and its ASLR policy (or "out of the pilot"), and postgres in the jail on amd64.
- [ ] **amd64 allowlists** are generated for `go`, `cpp` and `python` (and `go-race` if it works), with 0 unexpected SIGSYS under KILL.
- [ ] **The amd64 pod seccomp profile** (the x86_64/x86/x32 baseline) is validated under the supervisor, the negative probes still fail with it, the x86_64-only variant is tested, and both are recorded verbatim for mi-09, with the architecture choice proposed to m3-03.
- [ ] **Image volume:** (i) mounts read-only; (ii) and (ii-b) refuse a pod without the secret, or the `AlwaysVerify` setting goes to mi-09; (iii) the fallback is tested. The verdict is **GO or a named fallback from the pre-decided chain** (D40), or ⛔ "needs owner decision" with the options if every variant failed.
- [ ] The results are in t3 §16.2–16.4 and the t1 §3.3 pointer, the MI-10 verdict is in `status.md`, and the hand-offs are noted.
- [ ] The VM is deleted or handed to spk-03, `~/xl-spike/` is gone and the SSH alias is removed. The owner follow-up (reimage the second VPS, **or** delete the scratch repo) is recorded in `status.md`.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `docs/spk-02-results`, then the conventional commit `docs(v2): sandbox spike P3 + image-volume results (MI-10)` with the attribution lines, then push, then the PR. This repo only: no `../infra` PR and no evalpack change.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — spike (throwaway):** the harness, references, manifests, logs and scratch repo contents are never committed; only this results docs PR lands. No tag, no evalpack tag, no deploy. A verdict outside the pre-decided paths still lands, with its options, a recommendation and ⛔ "needs owner decision" in `status.md`. Then the teardown (step 5).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (the teardown record usually needs the follow-up).
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
