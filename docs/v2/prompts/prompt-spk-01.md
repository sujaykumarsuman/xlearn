# Prompt — Sprint spk-01 · Sandbox mechanism spike P0–P2 (multipass arm64, throwaway)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the xlearn repo root.
> **Plan:** [`../sprints/sprint-spk-01.md`](../sprints/sprint-spk-01.md) · **Milestone:** MI (rollout step MI-10, part 1) · **Prereqs:** MI-0; soft: [mi-14](../sprints/sprint-mi-14.md). Launching this prompt is the D23 go-ahead (`ev-spike-goahead`, D40)

## Read first

- [`../sprints/sprint-spk-01.md`](../sprints/sprint-spk-01.md): the plan. It holds the P1 checklist and P2 pass tables this prompt measures against, the in-pod channel, and the results spec.
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and land-and-sync.
- **[t3 §9 The smallest spike](../research/t3-sandbox.md#9-the-smallest-spike-local-and-throwaway-needs-the-owners-go-ahead): the script for this session.** Q-A/Q-B, the P0/P1/P1b/P2 table, the P1 checklist, the P2 corpus, and what is thrown away.
- **Mechanism background:**
  - t3 [§3 isolation options](../research/t3-sandbox.md#3-isolation-options-on-this-host) (R1, R1-N, R1-U, R1b);
  - [§5.2–5.4 process model and job lifecycle](../research/t3-sandbox.md#52-process-model-privilege-split);
  - [§5.10 cleanup invariants](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8);
  - [§6.3 SQL sandbox](../research/t3-sandbox.md#63-the-sql-sandbox-sql-pg18-pilot);
  - [§7.1 cgroup layout](../research/t3-sandbox.md#71-cgroup-layout).
- **Host files and guards:** t3 [§8.2 guard objects](../research/t3-sandbox.md#82-guard-objects-infrastructuresandbox), §8.3 (the runner values, right below §8.2), [§8.7 host-level changes](../research/t3-sandbox.md#87-host-level-changes-manual-scripted-recorded), and [§15 S0 production facts](../research/t3-sandbox.md#15-s0-results-read-only-production-facts-2026-09-24-owner-approved).
- **[ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md), which stays Proposed.** Its acceptance is [m3-03](../sprints/sprint-m3-03.md) task 1, not this session. Also [t3 §12 owner decisions](../research/t3-sandbox.md#12-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D20 three languages, D21 move triggers, D23 the spike go-ahead).
- **In the infra repo:**
  - `../infra/infrastructure/sandbox/`: mi-14's MI-4 manifests, on `main` or its PR branch.
  - `../infra/hack/sandbox-vap-test.sh` and its corpus `hack/sandbox-vap-test/`: mi-14's cases G1, B1–B14, Q1, X1/X2 and C1. See [mi-14](../sprints/sprint-mi-14.md) task 2.
  - `../infra/hack/host-bootstrap.sh`: the pinned k3s install and the H0 block to mirror.

## Context

**What the spike decides.** R1 is the runner design: a thin Go runner in a `hostUsers:false` pod under a `judge` containerd handler, with userns-less per-case jails via go-sandbox `forkexec.Runner`. It's **unproven until this spike runs** (D23). This session answers two questions on a throwaway **arm64 multipass VM** that mirrors production's stack (Ubuntu 24.04, k3s v1.36.4+k3s1, containerd 2.3.4, runc 1.4.2, the AppArmor userns restriction):
- **Q-A:** is R1 viable?
- **Q-B:** does every OOM land in a case cgroup, and does cleanup converge (INV-14)?

**What happens next.** spk-02 replays the syscall-sensitive parts on amd64 (Q-C) and runs the image-volume check on this same VM. spk-03 (WIF) may also use it. The results feed:
- [m3-03](../sprints/sprint-m3-03.md), which accepts ADR-0030 and builds the runner core;
- [mi-09](../sprints/sprint-mi-09.md), which scripts the host files for the Sat 2026-10-24 host window;
- [mi-14](../sprints/sprint-mi-14.md), for any VAP fix.

The M3 hard checklist needs "Spike P0–P3 **GO**" ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)).

**Time box.** The spike is time-boxed at **10 h of core effort across P0–P3** (t3 §9), counted as effort, not elapsed days. The Mon→Wed calendar is only the booking window. This session's core share is **7 h** (P0 1.5 + P1 3 + P2 2.5), plus **P1b's 0.75 h, which sits outside the core**.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] The go-ahead for P0–P3 and the image-volume spike (D23) is this launch (D40). Record `ev-spike-goahead` ✅ in `docs/v2/status.md` with the results; don't ask for another one.
- [ ] `ssh vps uname -r` (read-only) shows `6.8.0-142` or later, which means MI-0 is done.
- [ ] `multipass version` is ≥ 1.16, and the Mac has ≥ 8 GiB of RAM and 30 GB of disk free. There's no existing `xl-spike` VM, or it's one you may reuse and it's clean.
- [ ] Soft: find mi-14's manifests (`git -C ../infra log -- infrastructure/sandbox`, or `gh pr list -R sujaykumarsuman/infra`). If they're absent, use the t3 §8.2 draft and plan to report the diff.
- [ ] Soft: find mi-14's VAP corpus, `../infra/hack/sandbox-vap-test/` and `hack/sandbox-vap-test.sh`, on `main` or its PR branch. If it's absent, write G1, the 7 dry-run B-shapes (B1–B7, plus B8–B14 if time allows), Q1 and C1 yourself in `~/xl-spike/manifests/`, following [mi-14](../sprints/sprint-mi-14.md) task 2's definitions, and report them back to mi-14.

## Do this (in order)

0. **Set up the workspace.**
   - `mkdir -p ~/xl-spike/{sup,corpus,manifests,logs}`. This is outside every git repo and is never committed.
   - `unset KUBECONFIG`. The laptop's kubectl context tunnels to production, so from here on **every** `kubectl` is `multipass exec xl-spike -- sudo k3s kubectl …`.
   - Log each step's command and output into `~/xl-spike/logs/` as you go, so the results table can quote numbers.
   - **How you'll run commands in the positive pod.** The VAP denies CONNECT on `pods/exec` and `pods/attach` in `xlearn-runner` by design, so `kubectl exec` won't work there. Use **CRI exec**: `multipass exec xl-spike -- sudo k3s crictl exec <container-id> …`, with the id from `k3s crictl ps --name <container> -q`. CRI bypasses API-server admission, and runc still applies the container's seccomp, AppArmor, capabilities and user namespace to the exec'd process, so the checks stay representative. The alternative is to bake the checklist and corpus into the container's command and read the results with `kubectl logs` (a GET on `pods/log`, not a CONNECT). **Never relax the VAP to get exec.**
1. **[H] P0 setup (1.5 h).**
   - **Create the VM.** `multipass launch 24.04 --name xl-spike --cpus 4 --memory 8G --disk 30G`, then `multipass mount ~/xl-spike xl-spike:/spike`. Inside it: `apt-get update && apt-get -y full-upgrade && reboot`.
   - **Mirror H0.** Set `kernel.core_pattern=core` and `fs.suid_dumpable=0`, and mask apport.
   - **Add the sandbox sysctls** from t3 §8.7 as `/etc/sysctl.d/60-xlearn-sandbox.conf`.
   - **Assert and record:** `uname -r`, `apparmor_restrict_unprivileged_userns=1`, cgroup2, and whether `/dev/kvm` is absent.
   - **Install:**
     - k3s pinned as `../infra/hack/host-bootstrap.sh` does it: `install.sh` from the `v1.36.4+k3s1` tag, `INSTALL_K3S_VERSION=v1.36.4+k3s1`, args `server --write-kubeconfig-mode 0644`;
     - go1.26.8 from the checksummed tarball;
     - PGDG PostgreSQL 18, `g++` and `python3`.
   - **Write the host files** from t3 §8.7:
     - the drop-in `config-v3.toml.d/20-judge.toml`;
     - the AppArmor `xlearn-runner` profile, loaded with `apparmor_parser -r`;
     - the seccomp `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json`;
     - the `kubelet` system user with `subuid`/`subgid` `kubelet:<start>:7208960`, plus `getsubids`.
   - **Restart k3s** with a test pod running, and record whether it survives.
   - **Assert:** `crictl info` lists `judge`; the rendered `config.toml` runc section still says `SystemdCgroup = true`; the runc versions are the same.
   - **Apply the guards** inside the VM: mi-14's `infrastructure/sandbox/*.yaml`, or else the t3 §8.2 draft. The bindings go in at `validationActions: [Deny]`; if mi-14's flip hasn't merged, change the VM copy only.
   - **Prove the VAP** with mi-14's corpus `../infra/hack/sandbox-vap-test/*.yaml` (or your own from `~/xl-spike/manifests/`), running its `--expect deny` checks against the **VM** (`multipass exec xl-spike -- sudo k3s kubectl apply --dry-run=server -f -`). Never run it over `ssh vps`: that targets production.
     - G1 must be admitted.
     - B1–B14 must be denied. The **required 8** match mi-14's list: privileged (B1), hostPID (B2), token automount (B3), wrong image (B4), Unconfined seccomp (B5), extra caps (B6) and hostPath (B7) as dry-runs, plus **exec**, proven by the real X1/X2 CONNECT.
     - Q1 must be denied.
     - C1 (the same bad shape in `default`) must be **admitted**.
     - X1/X2, the **real** exec/attach CONNECT, run after step 2 creates the positive pod. This spike owns that proof, because mi-14 can't make it on production.
     - **E1 (extra, not one of the 8):** after step 2, a real `kubectl debug` ephemeral-container attempt on the positive pod (no `-it`) must be denied; the t3 §8.2 VAP covers `pods/ephemeralcontainers`. A pod-manifest dry-run can't test this. Report E1 to mi-14 as a corpus addition.
   - **If the drop-in isn't merged,** fix it, or fall back to a minimal `.tmpl` and record that.
2. **[H] P1 viability (3 h).**
   - **Image.** Build a throwaway arm64 image (debian:trixie-slim + go1.26.8 + your supervisor + g++/python3/PG 18). Import it with `k3s ctr images import` as `ghcr.io/sujaykumarsuman/xlearn-runner` and reference it by **digest** with `imagePullPolicy: Never`. If CRI won't resolve the digest, use a spike-only VAP copy with only the image rule widened, and record that. The dry-run set (G1, B1–B14, Q1, C1) still runs against the unmodified VAP; X1/X2 and E1 run against the copy, and you record which one.
   - **Positive pod** in `xlearn-runner`, per t3 §8.2–8.3:
     - `hostUsers:false`, UID 0, caps `SYS_ADMIN SETUID SETGID SETPCAP KILL` (drop ALL);
     - Localhost AppArmor and seccomp at pod and container level;
     - `procMount: Unmasked`, a read-only rootfs, no token, no service links;
     - the lowest PriorityClass, `dnsPolicy: None`, and requests/limits inside the Quota.
   - **Then run, recording the errno for each** (the plan's P1 table). Run the in-pod rows through CRI exec (or the baked command), never through `kubectl exec`:
     - cgroup `slots` with `+cpu +memory +pids`;
     - a `forkexec.Runner` jail **without CLONE_NEWUSER**: tmpfs root, `nosuid,nodev` read-only binds, an overlay, `pivot_root`, `hidepid=invisible,subset=pid`, loopback down;
     - spawn into a case cgroup via `CLONE_INTO_CGROUP` **and** via `cgroup.procs` + a sync pipe;
     - caps → 0 plus the bounding-set drop. Is `SETPCAP` needed?
     - `go build`, a `g++ -std=gnu++20 -O2` compile+run, and a `python3` run, all in the jail;
     - the negative probes from the supervisor (`unshare -U`, `clone3(CLONE_NEWUSER)`, `fsopen`+`move_mount`, `open_tree`+`move_mount`, `socket(SCTP)`, `socket(AF_NETLINK)`). All must fail.
     - from the jail: SIGSYS on a disallowed syscall, and no host core helper spawned;
     - CONNECT `kubectl exec`/`attach` into the pod (X1/X2), and `kubectl debug` (E1), which must all be denied;
     - delete/recreate the pod × 50.
   - **Fallbacks, still inside the time box:** try **nsjail `--disable_clone_newuser`** (the pre-decided Plan B). Past that, the result is outside the pre-decided paths: only *record* whether R1-U or R1b would work, with a recommendation, and mark the M3 line ⛔ "needs owner decision" in `status.md`. They are owner decisions, and R1b is a D21 trigger. The results PR still lands.
3. **[H] P1b (0.75 h).** Build a race fixture with `go test -c -race` and run it in the jail (clone3 → ENOSYS → clone). Boot `postgres` in the jail on `/job/sock` with `listen_addresses=''`, and run one query. Record the results. They go to p-01 and don't gate M3.
4. **[H] P2 (2.5 h).**
   - **Write the throwaway supervisor** of about 400 lines in `~/xl-spike/sup/`:
     - the t3 §7.1 layout;
     - `case-K` with `memory.max`, `oom.group=1`, `swap.max=0` and `pids.max`;
     - evidence read outside the process;
     - `cgroup.kill` + `rmdir`.
   - **Run the corpus:** balloon, fork bomb, thread bomb, tmpfs fill, inode fill, stdout flood, orphan double-fork, sleep, spin, the SQL balloon (backend moved into `case-K`), the SQL `set_config` timeout bypass, a lock-order deadlock, and a race fixture.
   - **Measure against the plan's P2 table:**
     - MLE 100/100, SQL included;
     - **0 container OOMs**;
     - 0 survivors;
     - baseline ± 5 MiB;
     - `nr_dying_descendants` → ~0 within 60 s after 1,000 case cgroups;
     - RACE ≥ 19/20;
     - DEADLOCK 20/20.
   - **Tune the limits and re-run** if a row fails. Any container OOM is a blocker.
5. **[X] Results (docs PR).**
   - Branch `docs/spk-01-results`.
   - Append **`## 16. Spike results (MI-10)` / `### 16.1 P0–P2 (spk-01, arm64 multipass)`** to `docs/v2/research/t3-sandbox.md`, with:
     - the environment versions, including the **`github.com/criyle/go-sandbox` module version and commit** (`go list -m github.com/criyle/go-sandbox`) and the **nsjail version and commit** if R1-N was used. m3-03 pins go-sandbox at exactly this version;
     - the in-pod channel used (CRI exec or baked command);
     - the per-phase table (step · measure · number · pass/fail · errno);
     - the Q-A and Q-B verdicts;
     - the jail mechanism;
     - the SETPCAP answer;
     - CLONE_INTO_CGROUP vs `cgroup.procs`;
     - pod survival on a k3s restart;
     - the subuid range and the × 50 recreate result;
     - which noble package ships `getsubids`;
     - **the final host files verbatim** (the drop-in, merged or `.tmpl`; the AppArmor profile; the arm64 pod seccomp profile as a sorted syscall list, or diffs from §8.7). Note that the pod profile mi-09 ships comes from spk-02's amd64 replay;
     - the VAP diff, plus E1 as a corpus addition (and your own corpus, if mi-14's was absent);
     - the line "no timing conclusions".
   - Update `docs/v2/status.md` (below).
   - Commit `docs(v2): sandbox spike P0–P2 results (MI-10)`, open the PR, and merge it once CI is green (see Ship).
   - Send the VAP diff, if there is one, to mi-14: a comment on its infra PR, or an infra issue if it has merged.
   - **Stop the VM** with `multipass stop xl-spike`. **Don't delete it**; spk-02 and spk-03 reuse it.

## Constraints

- **Throwaway.**
  - Never commit the supervisor, corpus, manifests, VM images or logs to any repo.
  - They live in `~/xl-spike/` and the VM only, and spk-02 deletes both at the end of the week.
  - The only commit is the results docs PR.
- **Production is off-limits.**
  - Read-only `ssh vps` facts only (`uname -r`, `sysctl`). No infra PR, no GitOps change, no `kubectl` on the laptop.
  - Inside the VM, `kubectl apply` is fine: it's a throwaway cluster. The GitOps rule is about production.
- **Hard stop.** 10 h of core effort for P0–P3, counted as effort, not elapsed days. This session's core share is 7 h, and P1b's 0.75 h is outside the core. At the stop, record the unfinished rows as "not run (time box)". Unknown never counts as GO.
- **Never relax the VAP to get exec** into the positive pod. Use CRI exec or a baked command. A loosened VAP invalidates the X1/X2 proof.
- **No timing conclusions** from arm64/HVF.
- **Don't decide past R1-N.** R1-U and R1b go to the owner as options with a recommendation: in t3 §16.1, the Decisions log and a ⛔ "needs owner decision" in status.md. Never wait for the answer (D40).
- **Don't edit ADR-0030.** m3-03 accepts it. No new ADR is expected. If a finding needs one, check peers' ADR numbers first (`gh pr list --state all`, `git worktree list`, `ListAgents`), because parallel sessions claim numbers.
- **No alerting** tooling of any kind (D34).
- **Credentials.** The spike needs none. Don't copy any production secret or kubeconfig into the VM.

## Deliverables

- `docs/v2/research/t3-sandbox.md` §16.1: the P0–P2 results table, the verdicts, the mechanism, the final host files and the VAP diff.
- `docs/v2/status.md`: the MI-10 part 1 result, the Sprint board row, the Decisions log and the hand-offs.
- One merged docs PR.
- A stopped `xl-spike` VM and the `~/xl-spike/` work directory, handed to spk-02. Neither is committed.

## Update status

- In [`../sprints/sprint-spk-01.md`](../sprints/sprint-spk-01.md), move each task ⬜ → 🔄 → ✅, or ⛔ with a reason. A NO-GO row is ✅ once it's recorded with its numbers. Set _Overall_ ✅ at the end.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for spk-01;
  - the **MI rows**, MI-10 part 1: "P0–P2: GO/NO-GO (mechanism: …)";
  - the owner calendar events: `ev-spike-goahead` ✅ (the launch, D40) and `ev-spike-week` in progress;
  - **Decisions log** lines: the mechanism chosen; any fallback that needs an owner decision (R1-U/R1b, marked ⛔ "needs owner decision"); the SETPCAP answer.
  - hand-off notes to spk-02, mi-09, mi-14 and m3-03.

## Done when (acceptance)

- [ ] Every P0, P1, P1b and P2 row has GO/NO-GO, numbers and an errno where relevant, and Q-A and Q-B each have a verdict.
- [ ] The VAP denies 8/8 required shapes (B1–B7 by dry-run, plus exec by the real X1/X2 CONNECT), the rest of B1–B14, Q1 and E1, and admits the same shape outside `xlearn-runner` (C1). It was never relaxed to obtain exec.
- [ ] Zero container OOMs, or a recorded NO-GO blocker.
- [ ] The final host files and the mechanism are in t3 §16.1, and the hand-offs are noted.
- [ ] Nothing but results is committed. The VM is stopped and handed to spk-02, and `~/xl-spike/` is outside every repo.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `docs/spk-01-results`, then the conventional commit `docs(v2): sandbox spike P0–P2 results (MI-10)` with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — spike (throwaway):** the supervisor, corpus, manifests, VM and logs are never committed; only this results docs PR lands. No tag, no deploy. A result outside the pre-decided paths (R1-U/R1b) still lands, with its options and recommendation in t3 §16.1 and ⛔ "needs owner decision" in `status.md`. The VM stays stopped for spk-02.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
