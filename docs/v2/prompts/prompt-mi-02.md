# Prompt — Sprint mi-02 · host-verify --cluster extension (MI-8)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-02.md`](../sprints/sprint-mi-02.md)   ·   **Milestone:** MI (rollout step MI-8 + the MI-0 follow-up)   ·   **Prereqs:** none in code; MI-0 (H0 reboot, Fri 2026-09-25)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] If the H0 reboot (`ev-mi0`) came before the S0 window ended (2026-09-27 ≈ 07:52 UTC): say in `status.md`'s S0 row, or in the launch message, whether `/tmp/xlearn-s0-vmstat.log` was copied off before the reboot, and where the copy is. With no note, the session records the log as lost at reboot.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-02.md`](../sprints/sprint-mi-02.md): the plan, with the check-by-check spec, the seeded TSV rows and today's expected numbers.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - §3: the MI-8 check table, and "No alerting in v2 (D34)";
  - §5: the memory-sum rule, the v2 resource model, and the TR-* triggers and responses;
  - §2: NATS stages N0–N4, and why `:8222` admits no pod.
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6: the release checklist, which calls `host-verify --cluster` before contract, erase and GA tags.
- [`../rollout-plan.md`](../rollout-plan.md):
  - §2: the MI-0 and MI-8 rows;
  - §2.2: "Verify the host";
  - §5: the M3 hard checklist's MI-8 and TR-STEAL items.
- [t7 §3.4 and §4](../research/t7-cross-cutting-and-rollout.md): the original K and H checks and the headroom baselines. The opscheck parts are superseded by D34.
- `../infra`:
  - `hack/host-verify.sh`: read all of it. It's read-only today, with `kc_get`, `pass/warn/fail/info` and `check_cluster`.
  - `hack/host-lint.sh` (the shared-constants pattern) and `hack/host-bootstrap.sh`.
  - `README.md`: the host-scripts section, and how the scripts are run (`scp` to `/root`, or `bash -s` over ssh).
- Neighbours that consume this: [mi-03](../sprints/sprint-mi-03.md), [mi-06](../sprints/sprint-mi-06.md), [mi-08](../sprints/sprint-mi-08.md), [mi-09](../sprints/sprint-mi-09.md), [mi-14](../sprints/sprint-mi-14.md), [m1-08](../sprints/sprint-m1-08.md).

## Context

- v2 has **no alerting** (D34). The owner looks at landscape and kubescope, and runs `host-verify --cluster`
  after every host change and before contract, erase and GA tags. This sprint makes that run answer the
  questions v2 depends on:
  - Does the memory sum still fit? It's the gate for the runner, MI-12.
  - Is anything OOMKilled or restarting?
  - Are the PG and NATS volumes filling?
  - Is NATS at the expected auth stage (N3: no `legacy` connection)?
  - Are the expected NetworkPolicies present?
  - Is steal firing TR-STEAL?
- Live facts (read-only, 2026-09-24):
  - node capacity 15.62 GiB;
  - Σ memory limits 9,354 Mi;
  - 23 containers with no memory limit (about 1.16 GiB working set);
  - NATS 2.14.6 with 7 anonymous connections;
  - `/varz` reachable at `k3s kubectl get --raw /api/v1/namespaces/messaging/pods/nats-0:8222/proxy/varz`;
  - the kubelet stats summary reachable through the node proxy;
  - `jq`, `python3` and `sar` present;
  - 17 sar files.
- The script is piped over ssh or copied to `/root`, so the tables it reads **must be embedded in it**.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-0 done: the H0 reboot into 6.8.0-142 (Fri 2026-09-25), and `host-verify --cluster` green after it. *If it slipped, build tasks 1–8 now and leave task 9's H0 check ⛔ (MI-0 pending); its S0 branch still runs if the window has ended.*
- [ ] No open peer PR in `../infra` touches `hack/`. Check `gh pr list -R sujaykumarsuman/infra`, `git worktree list` and ListAgents. If [mi-01](../sprints/sprint-mi-01.md) is open and adds `chart-diff.sh` to `host-lint.sh`, rebase onto it and keep both changes.

## Do this (in order)

1. **[H] Scaffold** on branch `feat/mi-8-host-verify-cluster` in `../infra`:
   - Add the `kc_top` and `kc_raw` wrappers beside `kc_get`, with the same 30 s timeout and stdout-only capture.
   - Add the flags `--with-runner`, `--nats-stage=open|n1|n3|n4`, `--json`, `--budget-file FILE` and `--netpol-file FILE`, plus usage text.
   - Add the constants `PIN_NATS_STAGE=open` and `HOST_RESERVE_MIB=2355`.
   - The JSON checks need `jq`. If it's missing, WARN and skip them.
   - In `host-bootstrap.sh` (no apply): add `jq` **and `sysstat`** to `PKGS` (line 65), and ensure `sysstat.service` + `sysstat-collect.timer` are enabled and active, the way it handles `iscsid`.

2. **[H] Memory sum** (plan task 1):
   - Create `hack/memory-budget.tsv`, seeded from the 23 limitless containers at their current `top` × 1.2 and marked `provisional`. Read it read-only with `ssh vps 'k3s kubectl top pods -A --containers --no-headers'`.
   - Embed it between marker lines.
   - Implement the terms:
     - Σ limits (incl. native sidecars);
     - limitless budgets (WARN when an entry is missing);
     - the surge: the **xlearn fleet roll only**, as in ADR-0035 §5 (7 × 128 Mi today). Print the largest other RollingUpdate pod limit (1,024 Mi Flux controllers today) as INFO, not in the sum;
     - host;
     - capacity − 512 Mi.
   - Add `--with-runner` (+3,072 Mi while `xlearn-runner` has no pod).
   - FAIL tagged `TR-MEM`. Print every term.
   - **Check the numbers:** hand-calculate each printed term from the same live reads; they must match. The totals must be within ±0.3 GiB of the plan's expected values: ≈ 13.7 GiB PASS (≈ +1.4) today, and ≈ 16.7 GiB FAIL (≈ −1.6) with `--with-runner`. ADR-0035 §5's row is context only, since its basis differs.

3. **[H] Flux, pods, CNPG** (plan task 2):
   - Extend the Flux Ready loop to GitRepository, HelmRepository, ImageRepository, ImagePolicy and ImageUpdateAutomation.
   - OOMKilled within 24 h outside `xlearn-runner` → FAIL (TR-MEM).
   - `restartCount > 3` with the last termination inside 24 h in `xlearn`, `messaging`, `databases` or `xlearn-runner` → FAIL. Label it as an approximation.
   - Keep the existing CNPG check.

4. **[H] Volumes and disk** (plan task 3):
   - PVC usage from `kc_raw /api/v1/nodes/<node>/proxy/stats/summary`: PG or NATS PVC ≥ 60% → WARN TR-DISK.
   - `df -P` on `/`, `/var/lib/rancher` and `/var/lib/longhorn`: ≥ 70% → WARN.
   - No `exec`.

5. **[H] NATS stage** (plan task 4): read `/varz` and `/connz?auth=true&limit=1024` through the `nats-0:8222` pod proxy, and apply the stage table: `open`, `n1`, `n3` (any `legacy` or anonymous connection → FAIL), `n4` (the n3 rule, plus a strict `.auth_required == true`; `false`, `null` or absent → FAIL). nats-server **omits** `auth_required` when auth is off. **Prove that both `--nats-stage=n3` and `--nats-stage=n4` FAIL today.**

6. **[H] NetworkPolicy presence** (plan task 5): create `hack/expected-netpol.tsv`, seeded with `kubescope/kubescope`, and embed it. Missing → FAIL. Unlisted in `xlearn`, `databases`, `messaging` or `xlearn-runner` → INFO.

7. **[H] Steal and CPU** (plan task 6):
   - Parse `sar -u -f` for every `/var/log/sysstat/sa??` with `LC_ALL=C`, locating columns by header.
   - Report: 24 h p95; per-day p95 over 7 days; the worst 1 h average; per-day maximum 60-minute user + system; full-history p95 (INFO).
   - WARN `TR-STEAL`: > 10% on ≥ 3 of 7 days, any 1 h > 25%, or the latest 24 h > 10%.
   - WARN `TR-CPU`: > 50% on ≥ 3 of 7 days.
   - If `sar` (`SAR_BIN`, default `sar`) or the sa files are missing, or none covers the last 24 h, WARN `host.steal`/`host.cpu` "sar unavailable — TR-STEAL unmeasured". **Never PASS.** Test it with `SAR_BIN=/nonexistent`.

8. **[H] Read-only proof, `--json`, lint, README; PR and merge** (plan task 7):
   - Extend `hack/host-lint.sh`:
     - the verb allowlist (`get|top` via the wrappers only; no apply, create, delete, edit, patch, replace, scale, exec, cp, label, annotate, drain, cordon, rollout or port-forward);
     - no **output** redirection to a path (`>`, `>>`, `>|`, `&>` to anything but `/dev/null` or `&N`), ignoring `[[ ]]`/`(( ))` comparisons, single-quoted strings and comments. Input redirection (`<`, `<<<`, `<(…)`) is allowed. Use the plan's exact `redir_hits` function (task 7), which has 0 hits on today's script;
     - the embedded TSVs equal the files;
     - shellcheck on `sample-top.sh`.
   - Implement `--json`: one escaped object per check plus a summary. Every line must parse with `jq -c .`.
   - Update the script header, the usage text, and the README host-scripts section.
   - Run the script on the node **only** in the piped form, `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`, which writes nothing. Also run it with `--with-runner`, `--nats-stage=n3`, `--nats-stage=n4` and `--json`.
   - Open the PR with that output and `host-lint.sh` in the body, and merge it. infra has no CI.
   - **The `/root` copy (pre-approved, D40).** After the merge, run `scp hack/host-verify.sh vps:/root/` once: it's a node write this plan names, so launching this prompt pre-approves it. Then run `ssh vps 'bash /root/host-verify.sh --cluster'` once; it must be green. Record it. If the refresh fails, record "`/root/host-verify.sh` stale" as a pending item.

9. **[H] Sampler** (plan task 8):
   - It writes on the node: `/root/sample-top.sh`, `/var/tmp/xlearn-top.tsv` and `/var/tmp/xlearn-top.pid`. The plan names these writes, so launching this prompt pre-approves them (D40). Don't ask; build and start it:
     - `hack/sample-top.sh` samples every 5 minutes with a timestamp, stops itself after `SAMPLE_HOURS=168`, and writes a PID file;
     - start it with `scp hack/sample-top.sh vps:/root/ && ssh vps 'nohup setsid bash /root/sample-top.sh </dev/null >/dev/null 2>&1 &'`;
     - record the start time and PID.
   - It ships in the same infra PR, or a follow-up.

10. **[H → X] MI-0 follow-up** (plan task 9):
    - Verify the H0 result yourself, read-only (event `ev-mi0`): `ssh vps uname -r` shows kernel 6.8.0-142, and the piped `--cluster` run is green after the reboot. On 2026-09-25 the node was still on 6.8.0-90.
    - The S0 log `/tmp/xlearn-s0-vmstat.log` has a 72 h window that ends **2026-09-27 ≈ 07:52 UTC**. Take the branch that applies:
      - **`ev-mi0` came first:** the reboot emptied `/tmp`. Use the copy the before-launch note names; if it names none, the log is lost.
      - **The window ended first:** copy the log off yourself (`scp vps:/tmp/xlearn-s0-vmstat.log <scratchpad>/`, a read). Once the copy is verified, delete it from the node yourself (`ssh vps rm /tmp/xlearn-s0-vmstat.log`): a node write the plan names, so launching this prompt pre-approves it (D40). Record the deletion.
    - If a copy exists, compute steal p50, p95 and max from the `st` column and append an S0 row to [t3 §15](../research/t3-sandbox.md). Otherwise record "lost at reboot; sar is the source".

11. **[X] Record** (branch `docs/mi-02-status`):
    - the Status table in [`../sprints/sprint-mi-02.md`](../sprints/sprint-mi-02.md);
    - in [`../status.md`](../status.md): MI-0 ✅ and MI-8 ✅ (PR, dates) and the Sprint board row;
    - **Decisions log:** the memory-sum terms and margins (plain and `--with-runner`), the steal baselines, the sampler start/PID or "declined", the S0 result, and the `PIN_NATS_STAGE` convention (mi-06 bumps it at N1 and N3, mi-11 at N4).

## Constraints

- **`host-verify.sh` is read-only, provably.** It uses `kubectl get` and `top` only; JSON goes through `get --raw` via the API-server proxy. No `exec`, `port-forward`, temp files or network calls outside the node. host-lint enforces this.
- **`ssh vps` stays read-only** except the three node writes this prompt names, which launching it pre-approves (D40): the `/root/host-verify.sh` refresh, the sampler's files, and deleting the S0 log. Verification always uses the piped form, `bash -s`, which writes nothing.
- **D34, no alerting:**
  - no timer, CronJob, push channel, Flux `Provider`/`Alert`, healthchecks.io or opscheck;
  - the sampler is a self-terminating one-off, never a monitor;
  - nothing notifies anyone.
- **GitOps:** no cluster object changes in this sprint. If a check tempts you to "fix" something live, record it instead, and plan a PR.
- **Memory-sum rule:** this sprint adds no pod. The check itself is the rule's enforcement point for every later always-on pod ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5).
- **Not applicable here, since there's no xlearn code:** goose + sqlc (`sqlc diff`), outbox/inbox, service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css` verbatim, consumers-before-producers, the ACL-PR-before-consuming-tag rule.
- **Parallel sessions:** re-check peers' infra PRs touching `hack/` before merging, and rebase. The image-automation bot commits to infra `main` often.
- Conventional commits with the attribution lines, and squash merges. Don't enable auto-merge.

## Deliverables

- infra PR:
  - `hack/host-verify.sh` (the MI-8 checks and flags; the embedded tables);
  - `hack/memory-budget.tsv` and `hack/expected-netpol.tsv`;
  - `hack/sample-top.sh`;
  - `hack/host-lint.sh` (the read-only proof, the TSV equality check, shellcheck);
  - `hack/host-bootstrap.sh` (`jq` and `sysstat` in `PKGS`; sysstat collection enabled);
  - the README host-scripts section.
- `/root/host-verify.sh` refreshed (step 8), or recorded as pending if that failed. The sampler running, or its in-session decline recorded.
- `hack/host-bootstrap.sh`: `jq` and `sysstat` in `PKGS`, sysstat collection enabled.
- xlearn docs PR: the status rows, the Decisions log lines, and the t3 §15 S0 row if the log exists.

## Update status

- Set each task row in [`../sprints/sprint-mi-02.md`](../sprints/sprint-mi-02.md) to 🔄 or ✅ (or ⛔ with a reason), and _Overall_ to ✅ when all ten are done.
- Mirror the state in [`../status.md`](../status.md): the **Sprint board** row, and the **MI track** rows MI-0 and MI-8.
- Add the **Decisions log** lines listed in step 11.
- No ADR is expected: this implements ADR-0035 §3 as written. If a check has to differ from the ADR table (a threshold, a data source), record it in the Decisions log. If it changes a decision, write an ADR, and check peers' ADR numbers first.

## Done when (acceptance)

- [ ] `host-verify --cluster` is green on the live node and prints every ADR-0035 §3 check.
- [ ] Each memory-sum term matches a hand calculation from live data. The totals are within ±0.3 GiB of ≈ 13.7 GiB PASS, and ≈ 16.7 GiB FAIL with `--with-runner`.
- [ ] All 23 limitless containers have a budget entry (no `cluster.mem-budget` WARN), and a trimmed `--budget-file` triggers the WARN.
- [ ] `--nats-stage=open` PASSes, and both `--nats-stage=n3` and `--nats-stage=n4` FAIL today.
- [ ] With `SAR_BIN=/nonexistent`, `host.steal`/`host.cpu` WARN "unmeasured" and never PASS.
- [ ] Read-only proven: the host-lint verb and output-redirection (`redir_hits`) checks pass, shellcheck is clean, and the embedded TSVs equal the files.
- [ ] `--json` parses line by line.
- [ ] The sampler is running (start recorded), or its in-session decline is recorded.
- [ ] `docs/v2/status.md` records MI-0, MI-8 ✅, the baselines and the S0 result.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: `feat/mi-8-host-verify-cluster` in `../infra`, then `docs/mi-02-status` in xlearn.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. infra has no CI: paste the node-run output and `host-lint.sh` into the PR body and merge on them.
3. **Release action — infra PR(s) only (host script):** Merge the infra PR (plus a sampler follow-up PR if `sample-top.sh` didn't ride it; each its own PR, never folded into a tag), then the xlearn docs/status PR. No tag. The script ships on merge (the piped form always runs the merged version); refresh the `/root` copy once afterwards (step 8).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
