# Sprint mi-08 — Limit hygiene (MI-11a) + Track B slice 1: PSA labels, SA tokens off (MI-15 part)

> **Milestone:** MI — rollout step **MI-11a** (it **gates MI-12**, the runner) and the first slice of **MI-15** (Track B) · **Track:** infra · **Order:** 25
> **Prereqs:** [mi-01](sprint-mi-01.md) (MI-3 chart 0.3.0: the `automountServiceAccountToken` knob, `hack/chart-diff.sh`) · [mi-02](sprint-mi-02.md) (MI-8 memory-sum check, `hack/memory-budget.tsv`, the top sampler)
> **Unblocks:** [mi-10](sprint-mi-10.md) (runner dark — ADR-0035: MI-11a before MI-12) · batched with [mi-09](sprint-mi-09.md)'s host window
> **Release action:** **infra PR(s) only** — merged before, or batched into, the **Sat 2026-10-24** host window (BP4)
> **Calendar:** week 4 (2026-10-17 → 10-23), right before [mi-09](sprint-mi-09.md). Preferred: merge the MI-11a PRs by **Thu 2026-10-22**, so the 24 h controller watch ends before the window. Fallback: leave them for the window's batched step (ev-host-window).
> **Execute with:** [`../prompts/prompt-mi-08.md`](../prompts/prompt-mi-08.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Read ≥ 48 h of top samples → p95 per container; stop the sampler if alive, delete its files | H | ⬜ |
| 2 | Flux controllers 1 GiB → 512 Mi (`flux-system/kustomization.yaml` patch) | I | ⬜ |
| 3 | Traefik + cert-manager limits; metrics-server budget entry | I | ⬜ |
| 4 | Longhorn (and other limitless) budget rows; drop now-limited rows; re-embed in `host-verify.sh` (host-lint); UI limit if exposed | I | ⬜ |
| 5 | PSA labels on `xlearn`, `databases`, `messaging` (server dry-run first) | I | ⬜ |
| 6 | `automountServiceAccountToken: false` on the 7 `xlearn-*` releases | I | ⬜ |
| 7 | Merge timing + re-verify (`host-verify --cluster --with-runner`) | H | ⬜ |
| 8 | Record (MI-11a, MI-15 slice, memory-sum numbers, decisions) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the MI table's MI-11a / MI-15 rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-3 merged ([mi-01](sprint-mi-01.md)): chart `0.3.0` with the `automountServiceAccountToken` knob, and `hack/chart-diff.sh`
- [ ] MI-8 memory-sum check available ([mi-02](sprint-mi-02.md)): `host-verify --cluster [--with-runner]`, `hack/memory-budget.tsv` and its embedded copy in `hack/host-verify.sh` (checked by `hack/host-lint.sh`)
- [ ] ≥ 48 h of samples exist in `/var/tmp/xlearn-top.tsv` (mi-02 started the sampler after the H0 reboot; it self-stops after `SAMPLE_HOURS`, default 168 h, so by now it has normally finished) — **or** mi-02 recorded the sampler as declined (its Decisions-log line) → keep the provisional `top × 1.2` budgets and say so (task 1)
- [ ] Hostinger weekly image date checked (≤ 7 days): the owner reads hPanel **before launch** (the prompt's before-launch item; launching attests it, D40) — the controller restarts are a restart-inducing step ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag))
- [ ] Parallel sessions: no `xlearn` tag rolling out and no peer PR open on the files below (a fleet rollout during the 24 h watch muddles it)

## Goal

Free **≈ 3 GiB of memory limits**, and give every limitless platform container **a limit or a budget
entry**, so the runner's 3 GiB (MI-12) fits the **memory-sum rule**
([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):

> Σ limits + Σ p95 working set of limitless containers + the largest rollout surge + host ≤ node capacity − 0.5 GiB

Before MI-11a the v2 projection is ≈ 0 margin steady and −1.1 to −1.5 GiB during a fleet rollout. After it,
the projection is ≈ 14.2 GiB during a fleet rollout, **≈ 0.9 GiB inside the rule**. Batch it with the Sat
2026-10-24 host window (BP4). Also land the first **Track B** hygiene, which needs only MI-3: **PSA labels**
and **SA tokens off** on the v1 releases ([t3 §8.6](../research/t3-sandbox.md#86-psa-tokens-and-pg-all-track-b)).

Live baseline (read-only, 2026-09-24): Σ limits **9,354 Mi**, of which the six Flux controllers hold
**6 × 1 Gi** while using 70–177 Mi. **23 containers** have no memory limit, ≈ 1.2 GiB working set: Longhorn ×14,
cert-manager ×3, Traefik, metrics-server, local-path, svclb ×2 and the NATS reloader.

## Scope

**In**
- MI-11a:
  - the 6 Flux controllers 1 GiB → 512 Mi;
  - limits on Traefik and cert-manager (controller, cainjector, webhook) through their values;
  - metrics-server as a documented budget entry;
  - Longhorn budgeted at measured p95 × 1.5 (a UI limit only if the chart exposes it; **never** a cap on
    instance-manager).
- MI-15 slice:
  - PSA labels — `xlearn` enforce `baseline`, `databases` enforce `restricted`, `messaging` enforce
    `baseline`; warn/audit `restricted` on all three;
  - `automountServiceAccountToken: false` on the 7 `xlearn-*` releases.

**Out**
- L22 CNPG memory (1 Gi today; 2 Gi only on TR-MEM) → trigger-only, no sprint ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)).
- L23 kubelet `system-reserved` / `eviction-hard` / `pod-max-pids` → the host window, [mi-09](sprint-mi-09.md).
- Runner objects → [mi-10](sprint-mi-10.md); the `xlearn-runner` namespace (born with its own PSA labels) → [mi-14](sprint-mi-14.md).
- `xlearn` egress, PG `connectionLimit` 20, Renovate, N4 → [mi-11](sprint-mi-11.md) (rest of MI-15).
- PSA on `kube-system`, `longhorn-system`, `cert-manager`, `flux-system` (flux-system already warns
  `restricted`; Longhorn needs privileged) — not in v2.
- Tokens off on airlift, kubescope, landscape, projects-hub — **never** (kubescope and landscape use their
  ServiceAccount's RBAC).

## Tasks

### 1 · Read the samples [H]

- Copy `/var/tmp/xlearn-top.tsv` off the node (`ssh vps 'cat /var/tmp/xlearn-top.tsv' > <scratch>/top.tsv`) and
  compute p50 / p95 / max per **(namespace, owner, container)**. Group by the pod's owner prefix: pod names
  change on every rollout. Cover every limitless container, the six Flux controllers (to confirm 512 Mi is
  ≥ 2.5 × their p95), Traefik, the three cert-manager containers and metrics-server.
- Record the sample window (first and last timestamp). The samples are about 2–3 weeks old by now, so take one
  fresh `ssh vps 'k3s kubectl top pods -A --containers'` and, wherever a container's current use exceeds its
  sampled p95, use the current value instead and say so.
- Put the table (container → p50, p95, max, proposed limit or budget) in the PR descriptions. It is the only
  record once the file is gone.
- **Clean up** (a node write this plan names: pre-approved by launching the prompt, D40, so the session runs it): stop the loop only if it is still alive, by its PID
  file, then delete all three files:
  `ssh vps 'pid=$(cat /var/tmp/xlearn-top.pid 2>/dev/null); [ -n "$pid" ] && ps -p "$pid" -o args= | grep -q sample-top.sh && kill "$pid"; rm -f /var/tmp/xlearn-top.tsv /var/tmp/xlearn-top.pid /root/sample-top.sh'`.
  The sampler was throwaway, never a timer (D34).
- **If mi-02 recorded the sampler as declined**: there is nothing to read or delete. Keep mi-02's provisional
  `top × 1.2` budget rows, and size the Traefik/cert-manager limits (task 3) and the Flux check (task 2) from the
  fresh `top` snapshot, treated as both p50 and p95. Mark them provisional in the PR and the Decisions log.
- If the sampler ran but there are < 48 h of samples (the loop died at an unplanned reboot), restart it per
  mi-02 (a node write, pre-approved by launching the prompt, D40) and wait: the budget rows outlive this sprint.

### 2 · Flux controllers → 512 Mi [I]

`../infra/clusters/vps/flux-system/kustomization.yaml` — patch here, **never** in `gotk-components.yaml` (which
`flux bootstrap` regenerates):

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- gotk-components.yaml
- gotk-sync.yaml
patches:
- target:
    kind: Deployment
    labelSelector: app.kubernetes.io/part-of=flux
  patch: |
    - op: replace
      path: /spec/template/spec/containers/0/resources/limits/memory
      value: 512Mi
```

- Requests stay 64 Mi and CPU is unchanged. `kubectl kustomize clusters/vps/flux-system | grep -c 'memory: 512Mi'`
  must print **6** before the PR opens.
- On merge, kustomize-controller applies the patch and all six controllers restart: a short reconcile pause,
  no app impact.
- **24 h watch:** `k3s kubectl -n flux-system get pods` shows 0 restarts and no OOMKilled;
  `flux get all -A` (or `get kustomizations,helmreleases -A`) all Ready.
- Revert = `git revert`.

### 3 · Traefik, cert-manager, metrics-server [I]

**Limit rule** (record it in the PR): memory limit = max(2 × p95, p95 + 64 Mi), rounded up to 32 Mi. **Always
set a request with the limit.** A limit alone makes the request default to the limit, which inflates Σ
requests. Request ≈ p50.
- **Traefik** (k3s-bundled chart `traefik-40.1.4`): `../infra/infrastructure/configs/traefik-config.yaml`
  (HelmChartConfig `valuesContent`) gains `resources: {requests: {cpu: 25m, memory: <p50>}, limits: {memory: <rule>}}`.
  k3s' helm-controller upgrades Traefik. The live strategy is RollingUpdate `maxSurge 1 / maxUnavailable 0`
  (verified 2026-09-24), so no ingress gap is expected. Smoke `projects.sujaykumar.dev/xlearn` and the sibling
  apps after the upgrade. A values typo fails the `helm-install-traefik` job while the old Traefik keeps
  serving; read that job's log.
- **cert-manager** (`../infra/infrastructure/controllers/cert-manager.yaml`, chart `v1.21.2`): `resources.limits`
  (controller; keep its 32 Mi request) plus `cainjector.resources` and `webhook.resources` (requests + limits),
  and `startupapicheck.resources` for the hook Job. Confirm the keys with
  `helm show values jetstack/cert-manager --version v1.21.2`. Verify: the three pods roll;
  `k3s kubectl get certificate -A` all Ready.
- **metrics-server:** a k3s-packaged addon (`/var/lib/rancher/k3s/server/manifests/metrics-server/`, reapplied
  by k3s at start). A Flux-side patch would fight k3s' deploy controller, and a `.skip` file plus an edited
  manifest is an unscripted host change. **Decision: a budget entry** in `hack/memory-budget.tsv` at p95 × 1.5,
  with that reason in the row's `source` column. Taking metrics-server over (`--disable metrics-server` plus a
  Flux HelmRelease) is not done in v2: it would add a component and a k3s restart.
  - This is the sprint's sanctioned fallback ("a limit through the k3s manifest override if supported, else a
    budget entry"), and it **satisfies ADR-0035's standing rule**: every container carries a memory limit **or**
    a budget entry, and the memory-sum check fails only on neither. ADR-0035 §5's MI-11a bullet lists
    metrics-server among the "limits through their values", but a k3s addon has no values, so **no new ADR** is
    needed. Record it as a Decisions-log line.

### 4 · Longhorn and the other limitless containers [I]

- `../infra/hack/memory-budget.tsv`, in mi-02's format
  (`namespace<TAB>container-glob<TAB>budget_mib<TAB>source`, budgeted per container instance): replace each
  provisional row with **p95 × 1.5** (rounded up to 16 Mi); the `source` column carries the p95 and the sample
  dates (e.g. `p95×1.5 of 41 Mi, samples 2026-09-26..10-03`).
  - **Longhorn:** instance-manager, longhorn-manager (+ its pre-pull sidecar), the csi-plugin's 3 containers,
    csi-attacher / provisioner / resizer / snapshotter, engine-image, driver-deployer, longhorn-ui ×2.
  - **Others:** local-path-provisioner, svclb ×2, the NATS `reloader`, metrics-server.
  - **Delete** the rows for containers that now carry a limit (task 3): `traefik`, `cert-manager-controller`,
    `cert-manager-cainjector`, `cert-manager-webhook`, and `longhorn-ui` if it gets one. Otherwise they show as
    "unused budget" INFO.
- **Re-embed** the edited file byte-identically between the `# >>> memory-budget.tsv` / `# <<< memory-budget.tsv`
  markers in `../infra/hack/host-verify.sh`. The script runs as `ssh vps 'bash -s' < host-verify.sh` (or from
  `/root/host-verify.sh`) with no TSV beside it, so **only the embedded copy counts**. Run `hack/host-lint.sh`
  (it fails when the copies differ) and paste its output into the PR.
- **Limits** only where the Longhorn `1.12.1` chart exposes `resources` for a part outside the data path. That
  means **longhorn-ui**, if exposed (check `helm show values longhorn/longhorn --version 1.12.1`).
  - **Never** cap instance-manager: it serves the PG and NATS volumes, so an OOM takes both down.
  - Keep longhorn-manager and the CSI parts budget-only: an OOM stalls attach/detach, which the window's PG
    restart needs.
- After merge, `host-verify --cluster` must report **no** "no limit and no budget entry" WARN, **zero**
  provisional budgets (unless the sampler was declined) and no "unused budget" INFO. Then refresh the node copy
  so on-demand `/root/host-verify.sh` runs use the new budgets: the session runs `scp ../infra/hack/host-verify.sh vps:/root/` once itself (a node write this plan names; pre-approved by launching the prompt, D40).

### 5 · PSA labels (MI-15 slice) [I]

| Namespace | File | enforce | warn / audit | Why ([t3 §8.6](../research/t3-sandbox.md#86-psa-tokens-and-pg-all-track-b)) |
|---|---|---|---|---|
| `xlearn` | `../infra/infrastructure/configs/namespaces.yaml` | `baseline` | `restricted` | judge's D5 `image` volume isn't in restricted's volume list |
| `databases` | `../infra/infrastructure/database/cluster/namespace.yaml` | `restricted` | `restricted` | CNPG is restricted-clean |
| `messaging` | `../infra/infrastructure/messaging/namespace.yaml` | `baseline` | `restricted` | the NATS chart sets no securityContext |

- Labels: `pod-security.kubernetes.io/enforce: <level>`, `enforce-version: v1.36` (pinned, so a k3s minor bump
  never tightens enforcement silently; bump it with k3s), `warn: restricted`, `warn-version: latest`,
  `audit: restricted`, `audit-version: latest`. Keep the MI-2 prune annotations on the two Namespaces.
- **Dry-run first** (server side; nothing persists), for each namespace at its level:
  `ssh vps 'k3s kubectl label --dry-run=server --overwrite ns databases pod-security.kubernetes.io/enforce=restricted pod-security.kubernetes.io/enforce-version=v1.36'`.
  Any "existing pods … violate" warning blocks that label. Paste the output into the PR.
- Labels restart nothing. `databases` meets its first real admission at the **window's PG restart** (CNPG 18.6,
  mi-09). If the CNPG pod is refused (a `FailedCreate` event), revert the `databases` label first. `xlearn` meets
  its first admission in task 6's rollout. The `warn: restricted` warning on the future judge pod (image volume)
  is expected; document it in the PR.

### 6 · SA tokens off on the 7 `xlearn-*` releases [I]

- `automountServiceAccountToken: false` in the values of `../infra/apps/xlearn-{gateway,identity,curriculum,practice,review,assessment,coach}.yaml`
  (the chart 0.3.0 knob). None of the seven talks to the Kubernetes API: no `k8s.io` module in `go.mod`.
- `hack/chart-diff.sh`: the **only** diff per xlearn release is that one pod-spec field. **No** diff on airlift,
  kubescope, landscape or projects-hub (kubescope and landscape need their tokens).
- Each xlearn pod rolls once (maxSurge 1; the transient surge is the ≈ 0.9 GiB the memory sum already counts).
- Verify:
  - the pods show `automountServiceAccountToken: false` and **no** `kube-api-access-*` volume;
  - login, dashboard and coach smoke OK;
  - NATS consumers re-bound (`/connz`);
  - every outbox's unsent count 0.

### 7 · Merge timing + re-verify [H]

- **Order:** task 2 (Flux) → tasks 3–4 (Traefik, cert-manager, budget file) → task 5 (PSA) → task 6 (tokens off).
  Tasks 5–6 are MI-15 and not window-bound: merge them when ready.
- **MI-11a (tasks 2–4):**
  - **preferred:** merged by Thu 2026-10-22, so the 24 h controller watch is clean before the Sat 2026-10-24
    window, whose k3s restart then doubles as the 512 Mi stress test;
  - **fallback (BP4 batch, when the session runs after Thu 2026-10-22):** push the two changes as branches
    without PRs (no PR stays open across sessions, D40) and record the branch names in status.md. The window
    session runs [mi-09](sprint-mi-09.md)'s runbook on its date: it opens their PRs, merges them, then runs
    `host-verify --cluster --with-runner`.
- **After merge:** `ssh vps 'bash -s -- --cluster --with-runner' < ../infra/hack/host-verify.sh`. Record
  Σ limits, Σ limitless p95 (budget file), the largest surge, the host share, the total vs capacity − 0.5 GiB,
  and the margin.
- **Also project the full v2 state by hand:** + judge 256 Mi (+ its surge) + coach P1 128 Mi. Compare with
  ADR-0035 §5 (≈ 14.2 GiB during a fleet rollout, ≈ 0.9 GiB inside the rule).

### 8 · Record [X]

- `docs/v2/status.md`:
  - the MI table's **MI-11a** row ✅ (PR numbers, merge date or "batched into the 10-24 window");
  - the **MI-15** row: PSA ✅, tokens ✅; egress, PG limits, Renovate and N4 pending [mi-11](sprint-mi-11.md);
  - the Sprint board row;
  - **Decisions log:** the memory-sum numbers (before → after, with and without the runner; mi-02 logged the
    baseline the same way), the sample window, the limit rule, the metrics-server budget-only decision (the
    sanctioned fallback; satisfies ADR-0035's limit-or-budget rule, no ADR), the pinned PSA enforce-version,
    Longhorn budget-only apart from the UI, and "sampler declined → provisional budgets kept" if that applied.

## Acceptance criteria

- [ ] Σ limits drop by ≈ 3 GiB from the Flux controllers (6 × 1 Gi → 6 × 512 Mi). The net Σ limits change and
      the memory-sum delta are recorded (the new Traefik/cert-manager limits add back roughly their limit − p95).
- [ ] No limitless container without a budget entry (`host-verify --cluster` has no such WARN).
- [ ] `host-verify --cluster --with-runner`: the memory sum is **inside the rule** (≤ capacity − 0.5 GiB), with
      the numbers recorded.
- [ ] PSA labels applied, with **no admission warnings** for running pods (server dry-run output in the PR).
- [ ] The 7 `xlearn-*` pods run with `automountServiceAccountToken: false`; chart-diff showed that field only,
      and nothing on the 4 non-xlearn releases.
- [ ] 24 h after the Flux PR: 0 controller restarts / OOMKills; all Flux objects Ready.

## Release

**Infra PR(s) only** — four small `../infra` PRs (Flux limits; Traefik + cert-manager + budget file with its
embedded copy in `hack/host-verify.sh`; PSA labels; tokens off), each its own task and never folded into a tag. `../infra` has no CI: merge under the standing
authority after each PR's verify step. The MI-11a PRs merge **before** the Sat 2026-10-24 window (preferred) or in
its batched step (BP4). The xlearn docs PR (`status.md`, this file) merges at session end. No xlearn tag; nothing
here ships in an image.

## Definition of Done

All four infra PRs merged through GitOps (or, for MI-11a, handed to the window's batched step per BP4 when the session runs after Thu 2026-10-22) ·
`host-verify --cluster --with-runner` inside the memory-sum rule · no unbudgeted limitless container · PSA and
tokens verified · statuses updated (this file + [`../status.md`](../status.md)) · decisions logged.

## Risks / watch-outs

- **A controller OOMs at 512 Mi under a large reconcile** (the window's k3s restart, or a chart bump
  re-rendering 11 releases). Watch restarts for 24 h; the fix is a revert or a per-controller 768 Mi override.
  Today's peak use is 177 Mi (kustomize-controller).
- **metrics-server is a k3s addon.** An override needs the host window or a component takeover, hence the
  budget entry (task 3).
- **A PSA label can reject an existing pod on its next restart.** Dry-run first; the `databases` label's
  real test is the window's PG restart. Revert the label before anything else if CNPG is refused.
- **A limit without a request becomes the request.** Always set both.
- **Traefik is a single replica.** RollingUpdate 1/0 is verified live, but do it at a quiet time and smoke-test
  every site on the node.
- **The cert-manager webhook restart** briefly fails cert-manager CR applies; Flux retries.
- **Tokens off on the wrong release** would break kubescope and landscape. chart-diff must show no change on
  them.
- **Stale or missing samples.** The sampler ran for 7 days after mi-02, so its data is 2–3 weeks old: cross-check
  against a fresh `top` (task 1). If an unplanned reboot left < 48 h, restart it and wait rather than fall back:
  the budget rows outlive this sprint. The only fallback to `top × 1.2` is mi-02 having recorded the
  sampler as declined.
- **Editing only `hack/memory-budget.tsv` changes nothing live.** `host-verify` reads the embedded copy;
  `host-lint.sh` catches the drift, so run it before the PR.
