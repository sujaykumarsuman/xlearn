# Prompt — Sprint mi-09 · October host-window prep: sandbox block, L23 kubelet args, pid limits, bumps (MI-11)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-09.md`](../sprints/sprint-mi-09.md)   ·   **Milestone:** MI (rollout step MI-11; not MI-9)   ·   **Prereqs:** [spk-01](../sprints/sprint-spk-01.md) + [spk-02](../sprints/sprint-spk-02.md) GO; window batched with [mi-08](../sprints/sprint-mi-08.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-09.md`](../sprints/sprint-mi-09.md): the plan, including the artefact table (task 1), the host-file table (task 2), the check table (task 4) and the runbook steps (task 10).
- [`../rollout-plan.md`](../rollout-plan.md):
  - §2, the MI-11 and MI-11a rows;
  - §2.2, the operating rules (verify the host, snapshots, no hand-applied changes);
  - §6, the calendar (the window is on the critical path; it can slip M3 by a month).
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md):
  - §5 Track A, step A6 (the host sandbox block);
  - §6, the patch cadence (D22);
  - §7, the T7 amendment (L23 joins the window; MI-11a before A8).
  It's still **Proposed**; [m3-03](../sprints/sprint-m3-03.md) accepts it. Don't edit it.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - §4, the L23 row;
  - §5, the memory-sum rule (MI-11a gates MI-12);
  - §6, the amendments.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):
  - §4.2, R-d as a procedure;
  - §4.3, the snapshot rule (no off-node `pg_dump`);
  - §4.4, the MI-11/MI-12 row (revert = restore the files and restart k3s).
- [t3](../research/t3-sandbox.md):
  - §8.7, host-level changes, the exact file contents;
  - §8.8, the rollout order;
  - §9, the spike;
  - §13, the risks;
  - §15, the S0 facts (runc 1.4.2, containerd 2.3.4, no `config-v3.toml.d`, no subuid, `mmap_rnd_bits=32`).
- The spike results in [t3](../research/t3-sandbox.md) §16: §16.1 (spk-01, P0–P2), §16.2 (spk-02, P3 amd64, including the amd64 pod-level seccomp profile verbatim) and §16.4 (the MI-10 verdict and the ADR-0030 deltas), landed by [spk-01](../sprints/sprint-spk-01.md) and [spk-02](../sprints/sprint-spk-02.md). **This is your source for every host file.**
- The neighbour plans:
  - [mi-02](../sprints/sprint-mi-02.md), the `host-verify --cluster` flags;
  - [mi-08](../sprints/sprint-mi-08.md), MI-11a, batched into the window;
  - [mi-10](../sprints/sprint-mi-10.md), the consumer of this window;
  - [mi-11](../sprints/sprint-mi-11.md), the monthly window and the k3s patch cadence.
- `../infra`, all read before editing:
  - `hack/host-bootstrap.sh`: blocks (0)–(8), the shared-constants block, `PIN_K3S_*`, the preflight k3s logic, the `run`/dry-run helpers;
  - `hack/host-verify.sh`: the `TODO(v2 T3)` section, the `k3s.args` and `k3s.config` checks, `check_cluster`;
  - `hack/host-lint.sh`;
  - `README.md`: "Host bootstrap & hardening", "Kernel reboot runbook", "v2 judge sandbox (planned)" and "Rebuild order";
  - `infrastructure/database/cluster/cluster.yaml`.

## Context

- The October host window (MI-11) is the one planned production restart that makes the node **safe to host
  untrusted code**. It runs on **Sat 2026-10-24** and is batched with MI-11a (BP4). It carries:
  - the host sandbox block (containerd `judge` runtime, AppArmor and pod-level seccomp profiles, the kubelet
    userns range, sysctls);
  - L23, kubelet memory reservation and eviction plus a pod pid limit. Today there's **no memory eviction**, and
    the kernel OOM killer picks the victims;
  - CNPG 18.6, and k3s v1.36.5 if it is GA.
- The cost is **one k3s restart and one PG restart**, after a manual snapshot.
- **The owner executes the window. This session only prepares it:**
  - merged, linted, dry-run scripts;
  - held **branches** (no PR) for the CNPG bump and, if GA, the k3s pin; the window opens and merges their PRs;
  - a runbook the owner has reviewed.
- Each step below is tagged with the plan task(s) it ticks in the plan's Status table (the order differs from
  the plan's numbering).
- `ssh vps` is **read-only** for you. You never change the live host.
- Read-only facts from 2026-09-24, to re-check:
  - kernel 6.8.0-90 (H0 reboots into ≥ 6.8.0-142 on 09-25); k3s `v1.36.4+k3s1`;
  - the rendered `config.toml` already imports `config-v3.toml.d/*.toml`, and that directory is absent;
  - `/var/lib/kubelet/seccomp` is absent; `/etc/subuid` holds only `ubuntu:100000:65536`; `getsubids` is absent;
  - `io_uring_disabled=0`, `kptr_restrict=1`;
  - `configz`: `evictionHard` {imagefs 5%, nodefs 5%}, no `systemReserved`, `podPidsLimit` −1; capacity 16,376,040 Ki;
  - no `/etc/rancher/k3s/config.yaml` (and `host-verify` WARNs if one appears).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] spk-01 and spk-02 report **GO** (P0–P3 + image volume), and t3 §16.1–16.4 exist. If the spike forced **R1b**, stop: that's a D21 move trigger (R2), and the window's sandbox block moves to the runner VPS.
- [ ] MI-0 done: `ssh vps uname -r` shows ≥ 6.8.0-142, and `host-verify --cluster` was green after it.
- [ ] mi-02 merged: `host-verify.sh --help` lists `--cluster`, `--with-runner` and `--nats-stage`.
- [ ] mi-08's MI-11a is merged, or its PRs are open and ready to batch into the window. Record which; it gates the window, not this session.
- [ ] The owner has booked Sat 2026-10-24 (check status.md or ask).
- [ ] Parallel sessions: `gh pr list -R sujaykumarsuman/infra` shows no open peer PR touching `hack/`, `cluster.yaml` or the README host sections. Check `git worktree list` and ListAgents too.

## Do this (in order)

1. **[H] Take the spike's final host artefacts** (plan task 1). From t3 §16.1–16.2, collect the final
   content, or the diffs against t3 §8.7, of:
   - the containerd drop-in (or, if P0 recorded the fallback, the `config-v3.toml.tmpl`) and the AppArmor profile;
   - the **amd64** pod-level seccomp profile (P3's file, not the arm64 P0 file);
   - the userns range and the `getsubids` package;
   - any sysctl change.

   If you only have diffs, rebuild each file from t3 §8.7 plus the diffs. Hand the final caps list (5 caps,
   or 4 without SETPCAP) to mi-10 via status.md; it's a runner value, not a host file.

2. **[H] Sandbox block** (plan task 2). Replace block (8) `FUTURE (v2 T3)` in `host-bootstrap.sh` with a real block, moved
   **before** block (6) k3s, behind a new `--with-sandbox` flag.
   - It's idempotent and dry-run aware, and each artefact is embedded as a heredoc.
   - It writes: `/etc/sysctl.d/60-xlearn-sandbox.conf` (+ `sysctl --system`); `…/containerd/config-v3.toml.d/20-judge.toml`
     (or `…/containerd/config-v3.toml.tmpl` if the spike recorded the `tmpl` fallback);
     `/etc/apparmor.d/xlearn-runner` (+ `apparmor_parser -r`); `/var/lib/kubelet/seccomp/profiles/xlearn-runner.json`;
     the system user `kubelet` with `kubelet:<start>:7208960` in `/etc/subuid` and `/etc/subgid`; and the
     `getsubids` package.
   - It refuses a range that overlaps `ubuntu:100000:65536`, isn't a multiple of 65,536, or ends near
     UINT32_MAX.
   - It **never restarts k3s**. It prints `k3s restart required to apply: …` instead.

3. **[H] L23** (plan task 3). Under the same flag, write `/etc/rancher/k3s/config.yaml.d/60-xlearn-kubelet.yaml` with
   `kubelet-arg`:
   - `system-reserved=cpu=250m,memory=1Gi`;
   - `eviction-hard=memory.available<500Mi,nodefs.available<5%,imagefs.available<5%`;
   - `pod-max-pids=4096`.

   Leave `enforce-node-allocatable` at its default.

4. **[H] Asserts + BOM + lint** (plan task 4). In `host-verify.sh`, replace the `TODO(v2 T3)` section with
   the plan's task 4 checks (`sandbox.sysctl`, `sandbox.containerd`, `sandbox.apparmor`, `sandbox.seccomp`,
   `sandbox.userns`, `kubelet.config`).
   - **Presence rule:** a partial install FAILs; a full absence is INFO unless `--expect-sandbox` is set.
   - `sandbox.containerd` branches on the shared constant `SANDBOX_CONTAINERD_MODE`, set from the spike
     record. With `dropin`, fail loudly if the rendered `config.toml` no longer imports
     `config-v3.toml.d/*.toml`, and check the drop-in hash. With `tmpl`, check the template hash, and skip
     the import assertion. Both modes check that runtime `judge` is listed, runc parity, and that the
     rendered runc has `SystemdCgroup = true`.
   - Read `configz` with `k3s kubectl get --raw`, print the whole `evictionHard` map, and keep every call a
     `get`.
   - Add `SANDBOX_SHA_*` constants and `SANDBOX_CONTAINERD_MODE` to the shared block in both scripts, and
     write `hack/host-bom.txt`.
   - Extend `host-lint.sh`: heredoc sha256 = constant = BOM line for every artefact.
   - Shellcheck must be clean.

5. **[H] Baseline, read-only, on the live node** (plan task 5). Run `crictl info`, read the rendered
   `config.toml`, run `configz`, and read capacity/allocatable, `/etc/subuid|subgid` and the sysctls.
   - Find the maximum `pids.current` per pod under `kubepods.slice`. If any pod is above 2,048, raise
     `pod-max-pids` or stop and ask.
   - Run `host-verify --cluster --with-runner`.
   - Dry-run both `main`'s and your branch's `host-bootstrap.sh` on the node, **without** `--with-sandbox`,
     and diff the two outputs. The diff must be empty: your branch plans nothing beyond what `main` already
     plans. Every run plans kernel-meta upgrades and any pending security updates, so don't expect an empty
     plan. The script warns against piping it into `bash -s`, because apt children can eat stdin. So stream
     each copy into a `mktemp` file, run it with `--dry-run </dev/null`, and delete the file. Those temp
     files are the only things you write on the node, and they change no host state.
   - Paste everything into the PR.

6. **[H] Throwaway-VM dry run** (plan task 6).
   - `multipass launch 24.04 --cpus 4 --memory 8G --disk 30G`, then `apt full-upgrade`, then reboot.
   - Run `host-bootstrap.sh --upgrade --with-k3s`, then `host-verify` once as the VM's baseline, then
     `--with-sandbox`, then `systemctl restart k3s`, then **`host-verify --expect-sandbox`** (host section).
     Every `sandbox.*` line and `kubelet.config` (through `configz`) must **PASS**, and no host line that
     passed in the baseline may regress.
   - **Don't gate on `--cluster` on the VM.** A bare multipass k3s has no Longhorn, CNPG or Flux, and none of
     mi-02's inputs (NATS, the expected NetworkPolicies, the memory budget), so `check_cluster` FAILs by
     design. If you run `--expect-sandbox --cluster` anyway, record the expected VM-only FAILs
     (`cluster.longhorn`, `cluster.cnpg`, `cluster.flux-ks`, `cluster.flux-hr`, NATS, NetworkPolicy).
     `--expect-sandbox --cluster` green is the live-node gate in runbook step 6.
   - Record `configz`, `crictl info` (`judge`), allocatable, the restart time and whether pods survived.
   - If step 7 bumps k3s, repeat on a fresh VM with `--with-sandbox --with-k3s --replace-k3s`, since one
     installer restart must apply everything.
   - Then `multipass delete --purge`. Commit nothing from the VM.

7. **[H] k3s pin decision** (plan task 8). Check whether `v1.36.5+k3s1` is a non-prerelease release on the
   v1.36 channel.
   - **If it is,** prepare the branch `chore/k3s-v1.36.5-host-window` bumping `PIN_K3S_VERSION` in both
     shared-constant copies, and dry-run it (step 6).
   - **If it isn't,** stay on v1.36.4 and write that into the runbook and status.

8. **[I] README** (plan task 9; goes into the host-script PR):
   - host-table rows for the sandbox block and L23;
   - the new flags;
   - the real "v2 judge sandbox" section;
   - **Rebuild order:** `--with-sandbox` on the bootstrap line and `--expect-sandbox` on both verify lines,
     plus a note that an R-d restore rewinds the host files;
   - a PR question to the owner about the Kernel reboot runbook's off-node `pg_dumpall`, which conflicts
     with ADR-0034 §4.3.

9. **[I] Open and merge the host-script PR** (steps 2–4 + 8; closes plan tasks 2–4 and 9). Paste in the lint output, the baseline, the
   dry-run evidence and the task 1 divergence table (artefact · t3 §8.7 · final · spike row).
   - infra has no CI, so the local check output is the evidence.
   - Merge it once the checks pass. This changes nothing on the cluster: Flux doesn't apply `hack/`.
   - The k3s pin stays **out** of this PR. Merging it early would make every pre-window `host-verify` FAIL
     `k3s.version`.

10. **[I] Push the held branches, with no PR** (plan tasks 7 and 8). AGENT.md allows no open PR at session
    end, and neither change may merge before the window, so push branches only (the same pattern as mi-11's
    N4 fallback). Don't merge either.
    - **CNPG** (`chore/cnpg-18.6-host-window`): in `infrastructure/database/cluster/cluster.yaml`, change
      `imageName` to `ghcr.io/cloudnative-pg/postgresql:18.6-system-trixie` and record the digest. Say in the
      commit message that the merge causes one PG restart (`unsupervised`, 1 instance), and that runbook
      step 7 opens and merges its PR.
    - **k3s pin** (`chore/k3s-v1.36.5-host-window`, only if GA, from step 7): runbook step 5b opens and
      merges its PR.
    - Put each branch's compare link in the runbook, so the owner reviews the diffs in step 12.

11. **[X] Runbook** (plan task 10) `docs/v2/runbooks/host-window-2026-10.md`. Write steps 0–9 plus the
    rollback from the plan's task 10. Give each step its command, expected output and stop condition. Include:
    - steps 5b and 7 opening the held branches' PRs (`gh pr create -R sujaykumarsuman/infra --head <branch> …`)
      and merging them;
    - the batched MI-11a step, and **step 3b always**: `host-verify --cluster --with-runner` shows the memory
      sum inside the rule before any host file goes in, whether or not mi-08's PRs were still open;
    - the k3s restart doubling as the stress test for mi-08's 512 Mi Flux controller limits;
    - the CNPG step's PSA note: if the new pod is refused under mi-08's `databases` `restricted` label,
      revert that label PR first;
    - the no-off-node-dump note;
    - the exact rollback commands (`rm` + `apparmor_parser -R` + `sysctl --system` + restart k3s);
    - the R-d order.

12. **[O] Owner review** (plan task 11). Ask the owner to read the runbook (~15 min) before 10-24. Record
    their approval, their k3s decision and their `pg_dumpall` answer.

13. **[X] Record** (plan task 12) in status.md (see Update status). Open the xlearn docs PR and merge it.

## Constraints

- **You never change the live host.** `ssh vps` is read-only. The window is an owner event.
- **GitOps:** never `kubectl apply`. Host state is the one sanctioned manual path, and it stays scripted,
  BOM-hashed and asserted by `host-verify` after every reboot, k3s upgrade or rebuild
  ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).
- **Host scripts:**
  - idempotent and `--dry-run` aware;
  - the shared constants byte-identical in both scripts;
  - `host-verify` read-only (get verbs only, no temp files);
  - shellcheck clean;
  - bootstrap never restarts k3s itself and never downgrades it.
- **D34, no alerting:** no timer, CronJob, Flux Alert, push channel or healthchecks.io. `host-verify` stays
  on demand.
- **The memory-sum rule** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):
  MI-11a before MI-12. The runbook checks the sum with `--with-runner` before the host block goes in.
- **Held branches:** the CNPG and k3s pin changes are pushed as branches with **no PR** and aren't yours to
  merge. The window's runbook opens and merges their PRs (steps 7 and 5b). Never leave an open PR at session
  end.
- **Don't edit ADR-0030.** It's accepted in m3-03. Divergences from t3 §8.7 go in your PR description and the
  status decisions log.
- **Not applicable, since there's no xlearn code:** goose + sqlc (`sqlc diff`), outbox/inbox, service
  boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css` verbatim,
  consumers-before-producers, and the ACL-PR-before-consuming-tag rule. If you find yourself editing
  `internal/`, `cmd/` or `web/`, stop.
- **Parallel sessions:** re-check peers' infra PRs and worktrees right before each merge, and rebase first.
  The Flux image bot commits to infra `main` often. No ADR number is needed here.
- Conventional commits with the required attribution lines, and squash merges. Don't enable auto-merge.

## Deliverables

- infra PR (merged):
  - `hack/host-bootstrap.sh`: the sandbox block + L23, `--with-sandbox`;
  - `hack/host-verify.sh`: the asserts, `--expect-sandbox`;
  - `hack/host-bom.txt`;
  - `hack/host-lint.sh`: the hash check;
  - the README host sections and Rebuild order.
- infra branch (pushed, **no PR**): `chore/cnpg-18.6-host-window`, CNPG `18.6-system-trixie`.
- infra branch (pushed, **no PR**, **only if v1.36.5 is GA**): `chore/k3s-v1.36.5-host-window`, `PIN_K3S_VERSION` in both scripts.
- xlearn docs PR (merged): `docs/v2/runbooks/host-window-2026-10.md`, the status rows and this sprint's Status table.

## Update status

- Set each task row in [`../sprints/sprint-mi-09.md`](../sprints/sprint-mi-09.md) to 🔄 or ✅ as you go (each step above names its plan task), and _Overall_ to ✅ when all 12 plan tasks are done.
- Mirror it in [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **MI track** MI-11 row: "prepared", the PR numbers, the window booked for 2026-10-24. The owner sets it ✅ in the window;
  - the held branches (names and head commits, "opened and merged in ev-host-window"), so the next session's peer checks don't take them for stragglers.
- Add **Decisions log** lines for:
  - `config.yaml.d` (or the fallback to `config.yaml`);
  - `--with-sandbox`/`--expect-sandbox`;
  - `SANDBOX_CONTAINERD_MODE` (drop-in or `.tmpl`, from the spike record);
  - the k3s v1.36.5 decision;
  - the caps list for mi-10;
  - any divergence from t3 §8.7;
  - the owner's `pg_dumpall` answer.
- No ADR is expected. If you must deviate from ADR-0030 §5 or ADR-0035 L23 values, write an ADR (check peers' ADR numbers first) rather than editing ADR-0030.

## Done when (acceptance)

- [ ] On a throwaway VM with the new block (dry run), on the k3s pin the window will use: `host-verify --expect-sandbox` green for every `sandbox.*` check and `kubelet.config`, with no host line regressed from the VM's baseline (`--cluster` not gated on the VM; any expected VM-only FAILs recorded)
- [ ] `hack/host-lint.sh` green: shared constants identical; heredoc = constant = BOM for every artefact; shellcheck clean; mi-02's read-only lint still passes
- [ ] On the live node, without `--with-sandbox`, your branch's `host-bootstrap.sh --dry-run` plans nothing beyond `main`'s copy (the two outputs diff empty), and `host-verify --cluster` is unchanged
- [ ] The CNPG 18.6 branch (and the k3s pin branch, if GA) is pushed with no PR, lint-green locally, and linked from the runbook
- [ ] Runbook reviewed by the owner before 10-24
- [ ] The infra README host table, flags and Rebuild order are updated
- Ship at session end per AGENT.md land-and-sync, with this sprint's release action: **infra PR(s) only**.
  - Merge the host-script PR and the xlearn docs PR once their checks pass.
  - Leave the CNPG and k3s pin changes as **pushed branches with no PR**, recorded in status.md. The window opens and merges their PRs, so no PR is left hanging.
  - No tag.
  - Sync local `main` in both repos.
