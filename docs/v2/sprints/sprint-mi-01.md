# Sprint mi-01 — Prune guards + chart 0.3.0 knob union (MI-2, MI-3)

> **Milestone:** MI — infra-first track (rollout steps **MI-2**, **MI-3**; sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** infra · **Order:** 1
> **Prereqs:** none (first v2 sprint) · calendar event MI-0 (H0 reboot, Fri 2026-09-25)
> **Unblocks:** [m1-01](sprint-m1-01.md) (MI-2 is M1's entry gate) · [mi-14](sprint-mi-14.md) (MI-4) · [mi-03](sprint-mi-03.md) (MI-5/5a) · [mi-08](sprint-mi-08.md) (MI-11a, tokens off)
> **Release action:** **infra PR(s) only** — two `../infra` PRs, **MI-2 merged first**, then MI-3 (one PR: chart 0.3.0 + `hack/chart-diff.sh`). Plus an xlearn docs PR for the status rows. No tag.
> **Calendar:** week 1 (2026-09-25 → 10-02), **first**
> **Execute with:** [`../prompts/prompt-mi-01.md`](../prompts/prompt-mi-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | MI-2 prune-guard PR (merged first) | I | ⬜ |
| 2 | `hack/chart-diff.sh` | I | ⬜ |
| 3 | Chart 0.3.0 knob union (§2.1), all default-off | I | ⬜ |
| 4 | Knob renders + byte-identical proof, merge MI-3 | I | ⬜ |
| 5 | Verify the host | H | ⬜ |
| 6 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track rows MI-2, MI-3).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-0 H0 reboot done (Fri 2026-09-25) and `host-verify --cluster` green. *MI-2 may go first if MI-0 slips: it depends on nothing ([rollout §2](../rollout-plan.md)).*
- [ ] infra#28 merged, local `../infra` `main` synced (✅ 2026-09-24)
- [ ] No open peer PR in `../infra` touches `charts/project/`, `infrastructure/database/cluster/`, `infrastructure/messaging/` or `hack/` (parallel sessions: `gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents)

## Goal

Land the cheapest data-loss guard first, then the one chart bump every later MI step needs. **MI-2** puts
`kustomize.toolkit.fluxcd.io/prune: disabled` on the three objects whose deletion loses data (the CNPG
`Cluster`, the `databases` Namespace, and the `messaging` Namespace that holds the NATS PVC). With no
backups (D12), a Flux prune is the one cheap way to lose everything. **MI-3** ships chart
`charts/project` **0.2.2 → 0.3.0** with the full [rollout §2.1](../rollout-plan.md) knob union, all
default-off. A new `hack/chart-diff.sh` proves that all 11 chart releases render byte-identically, so
the merge rolls **no pod**. After this sprint, no later milestone reopens the chart for a §2.1 knob.

## Scope

**In**
- MI-2: prune annotations on the CNPG `Cluster` `projects-pgstore`, Namespace `databases`, Namespace `messaging`. Own PR, merged first.
- MI-3, one PR: chart 0.3.0 with every §2.1 knob, `charts/project/ci/knob-*.yaml` render fixtures, `hack/chart-diff.sh`, and the infra README chart section.
- Read-only confirmation of the neighbouring data guards (NATS StatefulSet PVC retention, Longhorn StorageClass reclaim policy, CNPG `Database` reclaim policy), recorded.

**Out**
- Every consumer of the new knobs: `sandbox-guards` → [mi-14](sprint-mi-14.md); MI-5a ingress values → [mi-03](sprint-mi-03.md); tokens off + PSA → [mi-08](sprint-mi-08.md); runner values → [mi-10](sprint-mi-10.md); judge HelmRelease + default-deny egress → [m3-07](sprint-m3-07.md); judge 443 → [mi-12](sprint-mi-12.md); coach `rollingUpdate` 1/0 + grace 60 s → [mi-13](sprint-mi-13.md).
- A `workload: cronjob` user. After D34 there is none in v2. The knob stays default-off for the D34 revisit ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3).
- Knobs that already exist: `imagePullSecrets`, `extraVolumes`, `podAnnotations`, `strategy.type`, `enableServiceLinks`, `podSecurityContext`/`containerSecurityContext` (full pass-through).
- The `host-verify --cluster` extension → [mi-02](sprint-mi-02.md).

## Tasks

### 1 · MI-2 prune-guard PR (merged first) [I]

In `../infra`, add the annotation, with a short comment in the repo's explanatory style, to:

| Object | File | Owning Kustomization |
|---|---|---|
| CNPG `Cluster` `databases/projects-pgstore` | `infrastructure/database/cluster/cluster.yaml` | `databases` |
| Namespace `databases` | `infrastructure/database/cluster/namespace.yaml` | `databases` |
| Namespace `messaging` (holds the NATS PVC `nats-js-nats-0`) | `infrastructure/messaging/namespace.yaml` | `messaging` |

```yaml
metadata:
  annotations:
    kustomize.toolkit.fluxcd.io/prune: disabled   # MI-2: never garbage-collected by Flux (no backups, D12)
```

- **Why these three.** The root `flux-system` Kustomization prunes `./clusters/vps`. Deleting or renaming
  `clusters/vps/databases.yaml` or `messaging.yaml`, or moving these files, would garbage-collect the
  Cluster and the Namespaces, and the PVCs with them. With the annotation, Flux leaves the object in place
  (orphaned) instead.
- **Read-only facts to confirm and record** (measured 2026-09-24; re-read over `ssh vps`):
  - `sts/nats` `persistentVolumeClaimRetentionPolicy` = `Retain/Retain`;
  - StorageClass `longhorn` `reclaimPolicy` = `Retain`;
  - `Database` `xlearn` `databaseReclaimPolicy: retain` (already in git).

  These change nothing in this PR.
- Validate offline: `yq` parses every file, and `git diff` shows metadata only.
- After merge (read-only over `ssh vps`):
  - `k3s kubectl get cluster.postgresql.cnpg.io projects-pgstore -n databases -o jsonpath='{.metadata.annotations}'` and the two Namespaces show the annotation;
  - `k3s kubectl get kustomizations -n flux-system` shows `databases` and `messaging` Ready;
  - `projects-pgstore-1` and `nats-0` keep their start times (an annotation on the Cluster CR isn't propagated to pods).
- Commit `chore(data): guard the CNPG cluster and data namespaces from Flux prune (MI-2)`. Merge under the standing authority. infra has no CI, so paste the checks into the PR body.

### 2 · `hack/chart-diff.sh` [I]

A laptop script (never run on the node) that proves a chart change renders identically for every release.
- **Discover** each HelmRelease in `apps/*.yaml` whose `.spec.chart.spec.chart == "./charts/project"`, using `yq` (mikefarah v4). **Assert the count is 11** (`--expect N`, default 11): the 7 `xlearn-*`, airlift, kubescope, landscape and projects-hub. A release that silently drops out fails the run.
- For each release, write `.spec.values` to a temp values file, and render with the same values on both sides:
  - **base:** `git archive "$BASE_REF" charts/project`, where `BASE_REF` defaults to `origin/main`;
  - **head:** the working tree;
  - both with `helm template <releaseName> <chart> -n <namespace> -f values.yaml --kube-version v1.36.4`.
- **Normalise exactly one line:** `helm.sh/chart: project-<version>`, becoming `project-X`.
  - That label comes from `project.labels`. It sits on object metadata (Deployment, Service, SA, NetworkPolicy, IngressRoute, ConfigMap, ClusterRole, the shared Secret) and **never on the pod template**, which uses `project.selectorLabels`.
  - So the version bump updates metadata only and rolls nothing. Print how many lines were masked.
- `diff -u` per release. **Exit 1 on any byte difference**, and print `N/11 identical (helm.sh/chart masked)`.
- **`--self-test`:** apply a one-byte template change in a temp copy and require exit 1, so the script can't pass vacuously.
- **`--knobs`:** render each `charts/project/ci/knob-*.yaml` and assert the expected path with `yq` (task 4).
- **`valuesFrom` placeholder rule.** A release with `.spec.valuesFrom` can't resolve it offline, and it can't
  render without it either. projects-hub sets `sharedSecret.name` and `.namespaces` in `.spec.values`, and
  `charts/project/templates/sharedsecret.yaml` calls `fail "sharedSecret.data is empty…"` when `.data` is
  missing. So an offline `helm template` of projects-hub exits 1 on base **and** head (checked 2026-09-25).
  - For every release with `valuesFrom`, overlay the **same placeholder on both sides**. For projects-hub
    that's `--set sharedSecret.data.ADMIN_PASSWORD=chart-diff-placeholder`, or a stub values file with that
    key. Identical input on both sides keeps the diff valid.
  - Keep the placeholders in a per-release map in the script. A release that has `valuesFrom` but no map
    entry **fails the run** and names the release. It's never skipped silently.
  - Print `projects-hub: valuesFrom placeholder used (sharedSecret.data)` for each overlay. Never use a real value.
- **Temp files** live under `mktemp -d` with a `trap` cleanup.
- Add the script to `hack/host-lint.sh`'s shellcheck set (today it globs `host-*.sh` only). Install `shellcheck` locally if it's missing.

### 3 · Chart 0.3.0 knob union ([rollout §2.1](../rollout-plan.md)) [I]

Edit these files in `charts/project/`:
- `templates/deployment.yaml`, `templates/networkpolicy.yaml`, and a new `templates/cronjob.yaml`;
- `templates/_helpers.tpl`: the shared pod-spec `define` for the Deployment and the CronJob;
- `templates/service.yaml` and `templates/ingressroute.yaml` (its three Middlewares included): a `workload` guard, so a CronJob renders no Service, IngressRoute or Middleware;
- `values.yaml`: documented defaults, **all off**;
- `Chart.yaml`: `version: 0.3.0`, `appVersion` unchanged.

Every one of these template edits can cause a whitespace diff, not just the Deployment. `chart-diff.sh` covers them all.

Two rendering rules:
- **Booleans and integers** are emitted only when set, guarded by `kindIs "invalid"`. `false` and `0` then still render when set on purpose.
- **Strings, lists and maps** render only when non-empty.

A knob belongs here only if its shape serves every later user. The chart isn't reopened.

| Knob | `values.yaml` key (default) | Renders when set | Later users (what the shape must serve) |
|---|---|---|---|
| SA token | `automountServiceAccountToken: null` | pod `automountServiceAccountToken` | mi-08 (`false` on the 7 v1 releases), runner (VAP requires `false`) |
| Runtime / priority / userns | `runtimeClassName: ""`, `priorityClassName: ""`, `hostUsers: null` | pod fields | runner: `xlearn-judge`, `xlearn-sandbox-lowest`, `hostUsers: false` ([t3 §8.2–8.3](../research/t3-sandbox.md)) |
| DNS | `dnsPolicy: ""`, `dnsConfig: {}` | pod `dnsPolicy`, `dnsConfig` (toYaml) | runner: `None` + nameserver `127.0.0.1` |
| Grace period | `terminationGracePeriodSeconds: null` | pod field | runner 70 s; coach 60 s (MI-16) |
| **Split probes** | `probes.readinessPath: ""`, `probes.livenessPath: ""` (empty falls back to `probes.path`) | readiness and liveness on separate paths; readiness on `/readyz` is a per-release opt-in | every service as it opts in (ADR-0034 §3 "readiness gates cutover"); runner, judge |
| Image digest | `image.digest: ""` | image reference becomes `repository@digest` (the tag drops out of the reference) | runner: the VAP admits only `…/xlearn-runner@sha256:*` |
| Init containers | `initContainers: []` | pod `initContainers` (toYaml) | the evalpack initContainer fallback, if spk-02's image-volume spike fails ([t1 §3.3](../research/t1-content-data-model.md)) |
| **NetworkPolicy: multi-source ingress, same-namespace source, egress template** | keep `networkPolicy.enabled`/`from` **exactly as 0.2.2**. `from` may be set to `null` to drop the Traefik rule. Add `networkPolicy.extraIngress: []` (raw ingress rules; a peer with only `podSelector` is the same-namespace source) and `networkPolicy.egress: null` (`null` leaves egress untouched as today; a list, `[]` included, adds `Egress` to `policyTypes`, and `[]` means deny all) | extra ingress rules; the egress rules | [mi-03](sprint-mi-03.md) MI-5a. **Gateway:** `from: null` plus **one** `extraIngress` rule whose peers are [kube-system/traefik, `podSelector: {}`], on :8080. **Internal services:** `from: null` plus `extraIngress: [{from: [{podSelector: {}}], ports: [<containerPort>]}]`. Select by namespace or `instance`, **never `part-of`**: it's on object metadata only (`project.groupingLabels`), not a pod label. m3-07: judge is born default-deny egress (DNS, PG, NATS, runner). mi-12 adds 443 |
| Rolling update | `strategy.rollingUpdate: {}` | `rollingUpdate` under `strategy` when the type is `RollingUpdate` | mi-13 coach `maxSurge: 1`, `maxUnavailable: 0` |
| **`workload: cronjob`** | `workload: deployment`; `cronjob: {schedule: "", concurrencyPolicy: Forbid, activeDeadlineSeconds: 120, backoffLimit: 0, successfulJobsHistoryLimit: 1, failedJobsHistoryLimit: 1}` | a `CronJob` replaces the `Deployment`, with no Service or route. The pod spec is shared through one `define` helper | none in v2 (D34). It's kept for the healthchecks.io/opscheck revisit ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3) |

- Extracting the pod spec into a helper and wrapping `deployment.yaml` in a `workload` condition are the likeliest sources of a whitespace diff. `chart-diff.sh` is the guard.
- kubescope's policy (the only `networkPolicy.enabled: true` today) **must** render byte-identically. It guards a cluster-admin console.
- Update the **infra README "The shared `project` chart"** section with one line per new knob.

### 4 · Knob renders + byte-identical proof, merge MI-3 [I]

- **Fixtures.** Add one `charts/project/ci/knob-<name>.yaml` per knob, plus `knob-all-on.yaml`.
  - `hack/chart-diff.sh --knobs` asserts each field at its expected path. Examples:
    - `.spec.template.spec.hostUsers == false`;
    - an image ending `@sha256:…` with no `:tag`;
    - `policyTypes` containing `Egress` only when `egress` is set;
    - `knob-cronjob.yaml`: `kind: CronJob`, and **no** `Deployment`, `Service`, `IngressRoute` or `Middleware` in the render;
    - `knob-netpol-gateway.yaml`, which is mi-03's exact gateway shape (`from: null` plus one `extraIngress` rule with peers [kube-system/traefik, `podSelector: {}`] on :8080): `.spec.ingress | length == 1`, `.spec.ingress[0].from | length == 2`, and no `part-of` anywhere in the policy;
    - `knob-netpol-internal.yaml` (`from: null`, `extraIngress: [{from: [{podSelector: {}}], ports: [{port: 8081}]}]`): one rule, a single `podSelector: {}` peer, and no Traefik peer.
  - `helm lint --strict charts/project -f <file>` passes for each. Run `kubeconform -strict` too if it's installed.
- **PR body:**
  - `hack/chart-diff.sh` output (**11/11 identical**, `helm.sh/chart` masked);
  - `--self-test` exit 1;
  - the `--knobs` assertions;
  - `helm lint`;
  - `hack/host-lint.sh`.
- **Why the version bump matters:** HelmReleases use the default `reconcileStrategy: ChartVersion`, so a template change deploys only with the bump.
- **Merge, then watch** (read-only):
  - before the merge, snapshot `k3s kubectl get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,START:.status.startTime`;
  - after the merge, all 15 HelmReleases are Ready, and the 11 chart releases are upgraded (`helm.sh/chart: project-0.3.0` on their Deployments);
  - the pod snapshot is **identical**: same names, same start times.
- **Rollback:** `git revert` re-renders the 11 releases and leaves no residue ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §4.4).
- Commit `feat(chart): project 0.3.0 knob union, default-off, with chart-diff (MI-3)`.

### 5 · Verify the host [H]

After both reconciles, run `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`, which writes nothing on
the node. Don't `scp` to `/root`: that's a node write, and refreshing the node copy is the owner's call. Expect
no FAIL. If [mi-02](sprint-mi-02.md) has merged by then, the MI-8 checks run too.

### 6 · Record [X]

- In [`../status.md`](../status.md):
  - the **MI track** rows: MI-2 ✅ and MI-3 ✅ (infra PR numbers, dates, chart `0.3.0`);
  - the **Sprint board** row.
- **Decisions log:**
  - the masked-label rule (`helm.sh/chart` is the only expected chart-bump diff; it never reaches a pod template);
  - the `valuesFrom` placeholder rule (identical placeholder on both sides; a `valuesFrom` release with no placeholder entry fails the run);
  - the confirmed data guards (NATS PVC `Retain`, Longhorn `Retain`, `Database` `retain`).
- Update this file's Status table. Ship it as an xlearn docs PR.

## Acceptance criteria

- [ ] The prune annotation is live on `Cluster databases/projects-pgstore`, `Namespace databases` and `Namespace messaging`. The `databases` and `messaging` Kustomizations are Ready, and `projects-pgstore-1` and `nats-0` weren't restarted.
- [ ] `hack/chart-diff.sh` reports **11/11 identical** against `origin/main` (only `helm.sh/chart` masked; projects-hub rendered with its printed `valuesFrom` placeholder), and `--self-test` exits 1.
- [ ] Every §2.1 knob renders **only when set**: each `ci/knob-*.yaml` puts its field at the asserted path, the default render omits it, and `helm lint --strict` is clean.
- [ ] After the MI-3 merge, all 15 HelmReleases are Ready, the 11 chart releases are on `project-0.3.0`, and **no pod restarted** (pod start times unchanged).
- [ ] `host-verify --cluster` shows no FAIL after both merges.
- [ ] `docs/v2/status.md` shows MI-2 and MI-3 ✅ with their PR numbers.

## Release

**infra PR(s) only**, with no xlearn tag:
- The MI-2 PR merges first.
- The MI-3 PR follows. It carries the chart, fixtures, `chart-diff.sh`, the host-lint change and the README.
- The status rows ship as an xlearn docs PR.
- Nothing here rides a release tag. Infra PRs are never folded into a tag ([rollout §2.2](../rollout-plan.md)).
- **Rollback:** `git revert` of either PR. MI-2 is metadata only; MI-3 re-renders with no residue.

## Definition of Done

- Both infra PRs are merged, MI-2 before MI-3, with the check output in each PR body (infra has no CI).
- Flux is Ready and no pod restarted.
- `host-verify --cluster` is green.
- Statuses are updated (this file + [`../status.md`](../status.md)), and the xlearn docs PR is merged.
- Local `main` is synced in `xlearn` and `../infra`.

## Risks / watch-outs

- **Whitespace re-renders.** The pod-spec helper and the `workload` wrapper can shift a newline in every release's Deployment, which rolls every pod: a fleet surge of about 0.9 GiB on a node with a thin memory margin ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5). `chart-diff.sh` must be green before the merge, never after.
- **The chart label is the only allowed diff.** If a future template puts `project.labels` on the pod template, every chart bump rolls the fleet. Keep `project.selectorLabels` there.
- **A prune guard orphans; it doesn't forbid.** Removing a guarded file from git leaves the object running. To delete one on purpose, first remove the annotation in a PR, then delete. A manual `kubectl delete` isn't guarded (and is never sanctioned).
- **The NetworkPolicy defaults** must keep kubescope's policy identical. A regression there exposes a cluster-admin console pod-to-pod.
- **Offline renders need `valuesFrom` placeholders.** projects-hub's chart run `fail`s without `sharedSecret.data`. The placeholder is identical on both sides, so the diff stays valid, but its rendered Secret isn't the live one. A new `valuesFrom` release needs its own map entry, or the run fails (by design).
- **Local vs controller Helm.** Local Helm is v4 and the controller renders with its own SDK. `chart-diff.sh` compares like with like, so avoid Helm-4-only template functions.
- **No `workload: cronjob` user** may appear in v2 (D34). No Flux `Provider`/`Alert` and no opscheck either.
