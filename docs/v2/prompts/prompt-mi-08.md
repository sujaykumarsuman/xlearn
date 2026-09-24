# Prompt — Sprint mi-08 · Limit hygiene (MI-11a) + Track B slice 1: PSA labels, SA tokens off (MI-15 part)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-08.md`](../sprints/sprint-mi-08.md) · **Milestone:** MI (MI-11a gates MI-12; first slice of MI-15) · **Prereqs:** [mi-01](../sprints/sprint-mi-01.md) (chart 0.3.0), [mi-02](../sprints/sprint-mi-02.md) (memory-sum check + sampler)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, GitOps, land-and-sync.
- [`../sprints/sprint-mi-08.md`](../sprints/sprint-mi-08.md) — the plan, with the Flux patch, the limit rule, the PSA table and the merge order.
- [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (the memory-sum rule, the v2 resource model, MI-11a, the standing rules) and [§4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L22, L23).
- [t3 §8.6](../research/t3-sandbox.md#86-psa-tokens-and-pg-all-track-b) (PSA levels and why), [§8.8](../research/t3-sandbox.md#88-rollout-order-that-keeps-v1-up) (Track B items).
- [Rollout §2](../rollout-plan.md#2-mi-infra-track) (MI-11, MI-11a, MI-12, MI-15), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (verify the host, snapshots, infra PRs stand alone), [§12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session) ("Before MI-12: MI-11a"; the memory-sum standing gate).
- [`../sprints/sprint-mi-09.md`](../sprints/sprint-mi-09.md) — the 2026-10-24 window runbook, which carries the batched "merge the mi-08 PRs" step.
- Infra:
  - `../infra/clusters/vps/flux-system/{kustomization.yaml,gotk-components.yaml}` (six controllers, 1 Gi limits);
  - `../infra/infrastructure/configs/traefik-config.yaml` (HelmChartConfig);
  - `../infra/infrastructure/controllers/cert-manager.yaml` (chart v1.21.2);
  - `../infra/infrastructure/storage/release.yaml` (Longhorn 1.12.1);
  - `../infra/infrastructure/configs/namespaces.yaml`, `../infra/infrastructure/database/cluster/namespace.yaml`, `../infra/infrastructure/messaging/namespace.yaml`;
  - `../infra/apps/xlearn-*.yaml`;
  - `../infra/hack/{host-verify.sh,memory-budget.tsv,host-lint.sh,chart-diff.sh}` — `memory-budget.tsv` is embedded byte-identically in `host-verify.sh` between `# >>> memory-budget.tsv` / `# <<< memory-budget.tsv` markers, and `host-lint.sh` fails when the copies differ.
- [mi-02](../sprints/sprint-mi-02.md): the Tasks preamble (the embedding), task 1 (the TSV format) and task 8 (the sampler: `/root/sample-top.sh`, `/var/tmp/xlearn-top.{tsv,pid}`, `SAMPLE_HOURS` 168, the owner-declined fallback).

## Context

The node has 15.6 GiB and no swap. The kubelet has no memory eviction until L23 lands in the host window, so the
kernel OOM killer picks victims. On 2026-09-24:
- Σ limits was 9,354 Mi, and 6 GiB of it was the six Flux controllers (using 70–177 Mi).
- 23 containers had no memory limit at all (≈ 1.2 GiB working set).

ADR-0035's **memory-sum rule** puts v2 at ≈ 0 margin, and negative during a fleet rollout, until **MI-11a**
trims the Flux limits and budgets the limitless containers. The runner's 3 GiB (MI-12, [mi-10](../sprints/sprint-mi-10.md))
can't land before that. The owner batched MI-11a with the Sat 2026-10-24 host window (BP4). This sprint also
lands the cheapest Track B hygiene, which needs only chart 0.3.0: PSA labels on the three app namespaces, and no
auto-mounted SA token on the seven xlearn pods (none of them talks to the Kubernetes API).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] mi-01 merged: chart `0.3.0` in `../infra/charts/project/Chart.yaml` with the `automountServiceAccountToken` knob; `hack/chart-diff.sh` exists.
- [ ] mi-02 merged: `host-verify.sh --cluster --with-runner` works, `hack/memory-budget.tsv` exists and is embedded in `host-verify.sh`.
- [ ] ≥ 48 h of samples: `ssh vps 'head -1 /var/tmp/xlearn-top.tsv; tail -1 /var/tmp/xlearn-top.tsv'` spans ≥ 48 h (the sampler self-stops after 168 h, so it has normally finished) — **or** the owner declined the sampler in mi-02 (its Decisions-log line): then keep the provisional `top × 1.2` budgets and say so. If it was approved but an unplanned reboot left < 48 h, restart it per mi-02 and wait.
- [ ] The owner confirms the last Hostinger weekly image is ≤ 7 days old (controller restarts).
- [ ] No xlearn tag rolling out and no peer PR on these files (`gh pr list` in both repos, `git worktree list`, ListAgents).

## Do this (in order)

1. **[H] Samples → p95.** Copy `/var/tmp/xlearn-top.tsv` into the scratchpad. Compute p50/p95/max per
   (namespace, owner prefix, container) for every limitless container, the six Flux controllers, Traefik,
   cert-manager ×3 and metrics-server, and record the sample window. The data is 2–3 weeks old: take one fresh
   `k3s kubectl top pods -A --containers` and use the current value wherever it exceeds the sampled p95. Keep the
   table for the PR bodies. Then clean up (a node write: get the owner's OK first): stop the loop only if still alive,
   by its PID file, and delete all three files —
   `ssh vps 'pid=$(cat /var/tmp/xlearn-top.pid 2>/dev/null); [ -n "$pid" ] && ps -p "$pid" -o args= | grep -q sample-top.sh && kill "$pid"; rm -f /var/tmp/xlearn-top.tsv /var/tmp/xlearn-top.pid /root/sample-top.sh'`.
   **Sampler declined in mi-02:** keep the provisional budget rows; size step 2's check and step 3's limits
   from the fresh `top` snapshot (as p50 and p95), marked provisional.
2. **[I] Flux PR** (`chore/flux-limits-512mi` in `../infra`): add the `patches:` block from the plan to
   `clusters/vps/flux-system/kustomization.yaml` (target `kind: Deployment`,
   `labelSelector: app.kubernetes.io/part-of=flux`, replace `…/limits/memory` with `512Mi`). Check that
   `kubectl kustomize clusters/vps/flux-system | grep -c 'memory: 512Mi'` = 6. Merge per the timing in step 7,
   then start the 24 h watch (restarts, OOMKilled, Ready).
3. **[I] Traefik + cert-manager + budget PR** (`chore/limit-hygiene`):
   - **Limit rule:** limit = max(2 × p95, p95 + 64 Mi), rounded up to 32 Mi; request ≈ p50. Always set both.
   - **Traefik:** `resources` in `infrastructure/configs/traefik-config.yaml` `valuesContent`.
   - **cert-manager:** in `infrastructure/controllers/cert-manager.yaml`, the controller limit (keep its request)
     plus `cainjector`, `webhook` and `startupapicheck` resources; confirm the keys with `helm show values`.
   - **`hack/memory-budget.tsv`** in mi-02's format (`namespace<TAB>container-glob<TAB>budget_mib<TAB>source`):
     replace the provisional rows with p95 × 1.5 (rounded up to 16 Mi; `source` = the p95 and the sample dates)
     for every Longhorn container, local-path, svclb ×2, the NATS reloader and metrics-server (budget-only; the
     k3s-addon reason in `source`). **Delete** the rows for containers that now carry a limit (`traefik`,
     `cert-manager-*`, and `longhorn-ui` if limited), or they show as "unused budget" INFO.
   - **Re-embed** the file byte-identically between the `# >>> memory-budget.tsv` / `# <<< memory-budget.tsv`
     markers in `hack/host-verify.sh` (only the embedded copy is read at run time), run `hack/host-lint.sh`, and
     paste its output into the PR.
   - **metrics-server budget-only** is the planned fallback and satisfies ADR-0035's standing rule (a limit **or**
     a budget entry): no ADR, a Decisions-log line only.
   - **Longhorn:** a `longhorn-ui` limit only if the 1.12.1 chart exposes it. Never instance-manager; manager
     and CSI parts stay budget-only.
   - After merge: Traefik rolled with no gap (smoke every site on the node); `helm-install-traefik` job OK;
     `get certificate -A` Ready; `host-verify --cluster` shows no unbudgeted container, no provisional budget
     (unless the sampler was declined) and no "unused budget" INFO; the node copy refreshed the mi-02 way (gated
     [O]: the owner does it, or you run `scp ../infra/hack/host-verify.sh vps:/root/` once with the owner's
     explicit OK; otherwise record "`/root/host-verify.sh` stale: owner to refresh").
4. **[I] PSA PR** (`chore/psa-labels`):
   - Server dry-run each label first and paste the output:
     `ssh vps 'k3s kubectl label --dry-run=server --overwrite ns <ns> pod-security.kubernetes.io/enforce=<level> pod-security.kubernetes.io/enforce-version=v1.36'`.
     Any violation warning → stop and report that namespace.
   - Then set `enforce` (xlearn `baseline`, databases `restricted`, messaging `baseline`), `enforce-version:
     v1.36`, `warn`/`audit: restricted`, `warn-version`/`audit-version: latest` in the three Namespace
     manifests, keeping the MI-2 prune annotations.
   - Merge; `get ns --show-labels`.
5. **[I] Tokens-off PR** (`chore/xlearn-no-sa-token`): `automountServiceAccountToken: false` in the 7
   `apps/xlearn-*.yaml`. `hack/chart-diff.sh` must show only that field on the xlearn releases and nothing on
   airlift, kubescope, landscape or projects-hub; paste its output. Merge. Then verify:
   - no `kube-api-access-*` volume on any xlearn pod;
   - login, dashboard and coach smoke OK;
   - `/connz` consumers re-bound;
   - outbox unsent 0.
6. **[H] Verify.** `ssh vps 'bash -s -- --cluster --with-runner' < ../infra/hack/host-verify.sh`: green, no
   "no limit and no budget entry" WARN, memory sum ≤ capacity − 0.5 GiB. Record Σ limits, Σ limitless budget,
   surge, host, total and margin. Add the hand projection for judge (256 Mi + surge) and coach P1 (128 Mi), and
   compare with ADR-0035 §5 (≈ 14.2 GiB, ≈ 0.9 GiB inside the rule).
7. **[H] Timing.**
   - **Preferred:** merge the MI-11a PRs (steps 2–3) by **Thu 2026-10-22**, so the 24 h watch ends before the
     Sat 2026-10-24 window.
   - **If the session runs later**, or the owner prefers the batch (BP4): leave those two PRs open, labelled
     for the window, and tell the owner. [mi-09](../sprints/sprint-mi-09.md)'s runbook merges them and then
     runs `--with-runner`.
   - Steps 4–5 are not window-bound: merge them in this session.
8. **[X] Record** on a `docs/…` branch in xlearn: `docs/v2/status.md` (below) and this sprint's Status table.

## Constraints

- **GitOps only.** Every change is a `../infra` PR reconciled by Flux. Never `kubectl apply`, `edit`, `patch` or
  `label` for real by hand. `--dry-run=server` persists nothing and is the only "write-shaped" command allowed.
  Never edit `gotk-components.yaml`.
- **Infra PRs stand alone,** never folded into a tag. No xlearn code or migration changes here: goose/sqlc,
  outbox/inbox semantics, `theme.css` and service boundaries don't apply.
- **Memory-sum rule** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):
  every container ends this sprint with a memory limit **or** a budget entry. Never cap Longhorn's
  instance-manager. Always set a request with a limit.
- **D34:** no alerting, timer, CronJob or Flux Alert. The sampler was throwaway: stop it if alive and delete its
  three files. `host-verify` is on-demand only.
- **Host writes:** only the sampler cleanup (step 1) and mi-02's node-copy refresh
  (`scp ../infra/hack/host-verify.sh vps:/root/`), each only with the owner's explicit OK. Every other
  `ssh vps` command is read-only.
- **Snapshots:** check the weekly image before the restart-inducing merges. The host window's own snapshot is
  mi-09's runbook, not this sprint's.
- **kubescope and landscape keep their SA tokens** (their RBAC console depends on it). The chart-diff output
  proves it.
- **Parallel sessions:** check peers' PRs, tags and worktrees before each merge. Don't merge the Flux PR while
  an xlearn tag is rolling out.
- n/a: consumers-before-producers and ACL-before-tag (no NATS or topology change).

## Deliverables

- `../infra` PRs:
  - Flux controller limits (`clusters/vps/flux-system/kustomization.yaml`);
  - Traefik + cert-manager resources + `hack/memory-budget.tsv` rows (limited containers' rows removed) and its
    re-embedded copy in `hack/host-verify.sh`, with `host-lint.sh` output (+ a Longhorn UI limit if exposed);
  - PSA labels on three Namespaces;
  - `automountServiceAccountToken: false` on 7 releases.
- The p95 table and sample window in the PR bodies; the sampler stopped (if it was alive) and its three files
  deleted; `/root/host-verify.sh` refreshed (owner-gated) or recorded as a pending owner item.
- `host-verify --cluster --with-runner` numbers recorded; the xlearn docs PR (`status.md`, this sprint's statuses).

## Update status

- [`../sprints/sprint-mi-08.md`](../sprints/sprint-mi-08.md): each task 🔄 / ✅ / ⛔; _Overall_ ✅ when all eight
  are ✅. If the MI-11a PRs were handed to the window, mark task 7 ⛔ "batched into 2026-10-24 (mi-09)" until
  merged.
- [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **MI** table: MI-11a ✅ (PR numbers, merged date or "batched into the 10-24 window"), and MI-15
    partial (PSA ✅, tokens ✅; egress, PG limits, Renovate and N4 → [mi-11](../sprints/sprint-mi-11.md));
  - the **Decisions log:** the memory-sum numbers (before → after; with `--with-runner`; the v2 projection —
    logged the way mi-02 logged the baseline), the sample window, the limit rule, metrics-server budget-only
    (the planned fallback; it satisfies ADR-0035's limit-or-budget rule), the pinned `enforce-version`, Longhorn
    budget-only apart from the UI, and "sampler declined → provisional budgets kept" if that applied.
- No ADR expected (ADR-0035 governs; metrics-server budget-only is not a reversal). If a decision does reverse
  ADR-0035 §5, write one after checking peers for the next free ADR number.

## Done when (acceptance)

- [ ] Flux limits 6 × 1 Gi → 6 × 512 Mi (Σ −3 GiB); the net Σ limits change and the memory-sum delta are
      recorded.
- [ ] No limitless container without a budget entry.
- [ ] `host-verify --cluster --with-runner` memory sum inside the rule, with the numbers recorded.
- [ ] PSA labels applied with no admission warnings for running pods.
- [ ] 7 xlearn pods without an SA token; no change on the 4 non-xlearn releases.
- [ ] 24 h after the Flux merge: no controller restart or OOMKill; Flux Ready.

**Shipping:** release action = **infra PR(s) only**. Per AGENT.md land-and-sync:
- merge each `../infra` PR after its verify step (no CI in `../infra`; standing merge authority), let Flux
  reconcile, and verify live;
- the two MI-11a PRs may instead stay open **only** as the owner-agreed BP4 batch for the 2026-10-24 window;
  record that in `status.md` so mi-09's runbook merges them;
- then branch → commit (with the attribution lines) → PR → merge the xlearn docs PR;
- `git checkout main && git pull` in **both** repos (the peer-worktree route if `main` is held elsewhere).

No tag.
