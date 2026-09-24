# Prompt — Sprint mi-01 · Prune guards + chart 0.3.0 knob union (MI-2, MI-3)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-01.md`](../sprints/sprint-mi-01.md)   ·   **Milestone:** MI (rollout steps MI-2, MI-3)   ·   **Prereqs:** none (MI-0 H0 reboot, Fri 2026-09-25)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-01.md`](../sprints/sprint-mi-01.md): the plan, including the knob table with each knob's later users.
- [`../rollout-plan.md`](../rollout-plan.md):
  - §2 (the MI-2 and MI-3 rows);
  - §2.1 (the chart 0.3.0 knob list: *"No later milestone reopens the chart for a knob listed here"*);
  - §2.2 (operating rules).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §4.4: the chart 0.3.0 row. Guard: a byte-identical `helm template` for every release. Rollback: revert.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - §3: D34, no alerting, so `workload: cronjob` has no v2 user;
  - §5: the memory-sum rule, which is why a fleet re-render must not happen.
- The research:
  - [t3 §8.2–8.3](../research/t3-sandbox.md): runner values, VAP requirements, `kindIs "invalid"`;
  - [t7 §6.1](../research/t7-cross-cutting-and-rollout.md): the knob union;
  - [t1 §3.3](../research/t1-content-data-model.md): the evalpack initContainer fallback.
- `../infra`, all read before editing:
  - `README.md` ("The shared `project` chart", "Secrets", host scripts);
  - `charts/project/{Chart.yaml,values.yaml,templates/*}` (today `0.2.2`);
  - `apps/*.yaml` (the 11 chart releases);
  - `clusters/vps/*.yaml` (the root `flux-system` Kustomization prunes `./clusters/vps`);
  - `infrastructure/database/cluster/{cluster.yaml,namespace.yaml,xlearn-database.yaml}`;
  - `infrastructure/messaging/{namespace.yaml,release.yaml}`;
  - `hack/host-lint.sh` and `hack/host-verify.sh`.
- The neighbour plans that will consume the knobs: [mi-03](../sprints/sprint-mi-03.md), [mi-08](../sprints/sprint-mi-08.md), [mi-10](../sprints/sprint-mi-10.md), [mi-13](../sprints/sprint-mi-13.md), [mi-14](../sprints/sprint-mi-14.md), [m3-07](../sprints/sprint-m3-07.md).

## Context

- v2 is being built infra-first. The MI track is interleaved with product work and hard-gated before M3.
- This is the first v2 sprint, and it touches **only `../infra`**, plus a docs PR here for status.
- **MI-2** matters now because nothing else guards the data. There are no backups (D12). The root
  Kustomization prunes `./clusters/vps`, so a moved or deleted file could garbage-collect the Postgres
  `Cluster` or the NATS PVC's Namespace.
- **MI-3** is the single chart bump every later MI step relies on: the runner, judge, the MI-5a fences,
  tokens off, and the coach strategy. It must be **provably a no-op** for today's 11 releases: no pod may roll.
- Live facts, measured read-only on 2026-09-24. Re-read them; don't trust them blindly:
  - 15 HelmReleases, 11 on `./charts/project`;
  - only kubescope sets `networkPolicy.enabled`;
  - `sts/nats` PVC retention is `Retain`;
  - StorageClass `longhorn` has `reclaimPolicy: Retain`;
  - infra has **no CI**, so you run the checks and paste them into the PR.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-0 H0 reboot done (Fri 2026-09-25) and `host-verify --cluster` green. *If MI-0 slipped, you may still do task 1 (MI-2 depends on nothing), then report before MI-3.*
- [ ] infra#28 merged and local `../infra` `main` synced (`git -C ../infra fetch && git -C ../infra status`).
- [ ] No open peer PR in `../infra` touches `charts/project/`, `infrastructure/database/cluster/`, `infrastructure/messaging/` or `hack/`. Check `gh pr list -R sujaykumarsuman/infra`, `git worktree list` and ListAgents.

## Do this (in order)

1. **[I] MI-2 prune guard, as its own PR, merged first.**
   - Branch `chore/mi-2-prune-guard` in `../infra`.
   - Add `metadata.annotations: {kustomize.toolkit.fluxcd.io/prune: disabled}`, with a one-line comment (MI-2, D12), to three objects:
     - the CNPG `Cluster` in `infrastructure/database/cluster/cluster.yaml`;
     - the `databases` Namespace in `infrastructure/database/cluster/namespace.yaml`;
     - the `messaging` Namespace in `infrastructure/messaging/namespace.yaml`.
   - Validate with `yq` and check that `git diff` is metadata-only.
   - Re-read over `ssh vps`, read-only:
     - `k3s kubectl get sts nats -n messaging -o jsonpath='{.spec.persistentVolumeClaimRetentionPolicy}'`;
     - `k3s kubectl get sc longhorn -o jsonpath='{.reclaimPolicy}'`.
   - Commit (conventional, with the attribution lines), push, open the PR with the checks in the body, and merge it (`gh pr merge --squash --delete-branch`).
   - Then verify read-only:
     - the annotation is on all three live objects;
     - `k3s kubectl get kustomizations -n flux-system` shows `databases` and `messaging` Ready;
     - `projects-pgstore-1` and `nats-0` kept their start times.

2. **[I] `hack/chart-diff.sh`,** on branch `feat/mi-3-chart-0.3.0`. Build it exactly as the plan's task 2 specifies:
   - discover the `./charts/project` HelmReleases with `yq` and assert 11 (`--expect`);
   - render base (`git archive "${BASE_REF:-origin/main}" charts/project`) and head with identical values, using `--kube-version v1.36.4`;
   - mask only `helm.sh/chart: project-<ver>`;
   - `diff -u`, exit 1 on any difference;
   - add `--self-test` and `--knobs` modes;
   - put temp files in `mktemp -d` with a trap;
   - for a release with `valuesFrom`, overlay the **same placeholder on both sides** and print that it was used. For projects-hub that's `--set sharedSecret.data.ADMIN_PASSWORD=chart-diff-placeholder`. Without it, the chart's `sharedsecret.yaml` `fail`s and projects-hub exits 1 offline on both base and head. A `valuesFrom` release with no placeholder entry fails the run.

   Extend `hack/host-lint.sh` to shellcheck `chart-diff.sh`. Install `shellcheck` locally if it's missing.
   **Run it against the untouched chart first: 11/11 identical.**

3. **[I] Chart 0.3.0.** Implement the plan's knob table in these `charts/project/` files:
   - `templates/deployment.yaml`, `templates/networkpolicy.yaml`, and a new `templates/cronjob.yaml`;
   - `templates/_helpers.tpl`, for the shared pod-spec `define`;
   - `templates/service.yaml` and `templates/ingressroute.yaml` (its Middlewares included): the `workload` guard;
   - `values.yaml`.

   The details:
   - The knobs: `automountServiceAccountToken`; `runtimeClassName`, `priorityClassName`, `hostUsers`; `dnsPolicy`/`dnsConfig`; `terminationGracePeriodSeconds`; split `probes.readinessPath`/`probes.livenessPath`; `image.digest`; `initContainers`; the NetworkPolicy `extraIngress` + nullable `from` + `egress` template; `strategy.rollingUpdate`; `workload: cronjob` with a `cronjob:` block.
   - Rendering rules:
     - every default is off;
     - booleans and integers are guarded with `kindIs "invalid"`;
     - strings, lists and maps render only when non-empty;
     - a digest renders `repository@digest`;
     - the pod spec is shared between Deployment and CronJob through one `define`;
     - a CronJob gets no Service, IngressRoute or Middleware;
     - `egress: null` leaves the 0.2.2 output untouched;
     - `networkPolicy.from: null` drops the Traefik rule, so mi-03's gateway can be written as `from: null` plus **one** `extraIngress` rule with both peers (Traefik and `podSelector: {}`). Never select on `part-of`: it isn't a pod label.
   - Document every key in `values.yaml` in the repo's comment style.
   - Bump `Chart.yaml` to `0.3.0`, and update the README chart section.
   - **Re-run `hack/chart-diff.sh` after every template edit** until it shows 11/11 identical.

4. **[I] Prove it and merge MI-3.**
   - Add `charts/project/ci/knob-<name>.yaml` for each knob, plus `knob-all-on.yaml`. Make `--knobs` assert each field's path. Include these:
     - `knob-cronjob.yaml`: assert a CronJob and **no** Deployment, Service, IngressRoute or Middleware;
     - `knob-netpol-gateway.yaml` and `knob-netpol-internal.yaml`: mi-03's exact shapes, per the plan's task 4.
   - Run `helm lint --strict` with each file, and `kubeconform -strict` if you have it.
   - Before merging, snapshot the pods over `ssh vps`: `k3s kubectl get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,START:.status.startTime`.
   - Open the PR with this output in the body:
     - chart-diff (11/11);
     - `--self-test` (exit 1);
     - `--knobs`;
     - `helm lint`;
     - `hack/host-lint.sh`.
   - Merge it.
   - Watch read-only until `k3s kubectl get hr -A` shows all 15 Ready and the 11 chart Deployments carry `helm.sh/chart: project-0.3.0`.
   - Diff the pod snapshot: **it must be identical**. If any pod rolled, revert immediately (`git revert`, PR, merge) and report.

5. **[H] Verify the host:** `ssh vps 'bash -s -- --cluster' < ../infra/hack/host-verify.sh`, which writes nothing on the node. Don't `scp` to `/root`: refreshing the node copy is the owner's call. Expect no FAIL.

6. **[X] Record** (branch `docs/mi-01-status` in this repo):
   - [`../sprints/sprint-mi-01.md`](../sprints/sprint-mi-01.md): task rows ✅ and _Overall_ ✅;
   - [`../status.md`](../status.md): the **MI track** rows MI-2 ✅ and MI-3 ✅ (infra PR numbers, dates, chart `0.3.0`) and the **Sprint board** row;
   - **Decisions log:** the masked-label rule and the confirmed data guards.

## Constraints

- **GitOps only.** Never `kubectl apply`, `edit`, `annotate` or `delete` by hand. `ssh vps` is **read-only** here (`get`, `top` and the `host-verify` reads). Everything reaches the cluster through a merged `../infra` PR ([ADR-0009](../../adr/0009-deployment-and-gitops.md)).
- **Infra PRs are their own tasks.** They're never folded into a tag. MI-2 and MI-3 are separate PRs, and MI-2 merges first.
- **No pod may roll.** Today's memory margin is thin ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5 memory-sum rule), and this sprint adds no pod. `chart-diff.sh` must be green **before** the merge.
- **D34, no alerting:** no Flux `Provider`/`Alert`, no opscheck, no CronJob user, no healthchecks.io. `workload: cronjob` stays default-off and unused.
- **Don't change existing keys' meaning or defaults**, including `strategy.type`, `networkPolicy.from`, `probes.path` and `enableServiceLinks`. Anything else would break the byte-identical guarantee.
- **Not applicable here, since there's no xlearn code:** goose + sqlc (`sqlc diff`), outbox/inbox, service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css` verbatim, consumers-before-producers, the ACL-PR-before-consuming-tag rule. If you find yourself editing `internal/`, `cmd/` or `web/`, stop.
- **Parallel sessions:** re-check peers' open infra PRs right before each merge, and rebase if `main` moved. The Flux image-automation bot commits to infra `main` every few minutes, so rebase before you push.
- Conventional commits with the required attribution lines, and squash merges. Don't enable auto-merge.

## Deliverables

- infra PR, MI-2: prune annotations on the CNPG `Cluster` and the `databases` and `messaging` Namespaces.
- infra PR, MI-3:
  - `charts/project` 0.3.0 (the knob union, default-off);
  - `templates/cronjob.yaml`;
  - `charts/project/ci/knob-*.yaml`;
  - `hack/chart-diff.sh` (with `--self-test` and `--knobs`);
  - the `hack/host-lint.sh` shellcheck set;
  - the README chart section.
- xlearn docs PR: the status rows and this sprint's Status table.

## Update status

- Set each task row in [`../sprints/sprint-mi-01.md`](../sprints/sprint-mi-01.md) to 🔄 or ✅ as you go, and _Overall_ to ✅ when all six are done.
- Mirror the state in [`../status.md`](../status.md): the **Sprint board** row, and the **MI track** rows MI-2 and MI-3 (PR numbers, dates).
- Add **Decisions log** lines for:
  - the masked `helm.sh/chart` label;
  - the `valuesFrom` placeholder rule (projects-hub);
  - the recorded data guards (NATS PVC `Retain`, Longhorn `Retain`, `Database` `retain`);
  - any knob whose shape you had to change from the plan's table, and why.
- No ADR is expected. If you deviate from rollout §2.1 (a knob dropped or reshaped), record an ADR, and check peers' ADR numbers first.

## Done when (acceptance)

- [ ] The prune annotation is live on `Cluster databases/projects-pgstore`, `Namespace databases` and `Namespace messaging`. `databases` and `messaging` are Ready, and PG and NATS weren't restarted.
- [ ] `hack/chart-diff.sh` shows **11/11 identical** (`helm.sh/chart` masked, projects-hub's `valuesFrom` placeholder printed), and `--self-test` exits 1.
- [ ] Every §2.1 knob renders only when set (the `--knobs` asserts pass), and `helm lint --strict` is clean.
- [ ] After the MI-3 merge, all 15 HelmReleases are Ready, the 11 chart releases are on `project-0.3.0`, and **no pod restarted**.
- [ ] `host-verify --cluster` shows no FAIL.
- [ ] `docs/v2/status.md` shows MI-2 and MI-3 ✅ with their PR numbers.
- Ship at session end per AGENT.md land-and-sync, with this sprint's release action: **infra PR(s) only**.
  - Merge MI-2, then MI-3, yourself, with the local check output in each PR body (infra has no CI).
  - Merge the xlearn status docs PR.
  - No tag.
  - Sync local `main` in both repos.
