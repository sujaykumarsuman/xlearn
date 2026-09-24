# Prompt — Sprint mi-14 · sandbox-guards: empty default-deny xlearn-runner namespace + VAP (MI-4)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the xlearn repo root. The work happens in the sibling `../infra` repo, through GitOps PRs.
> **Plan:** [`../sprints/sprint-mi-14.md`](../sprints/sprint-mi-14.md)   ·   **Milestone:** MI (rollout step MI-4, ADR-0030 A7, limit L14)   ·   **Prereqs:** [mi-01](../sprints/sprint-mi-01.md), [mi-02](../sprints/sprint-mi-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): the land-and-sync directive and the `../infra` conventions (inspect them before adding deploy config; never `kubectl apply` by hand).
- [`../sprints/sprint-mi-14.md`](../sprints/sprint-mi-14.md): the plan. Task 1 holds the object table and the pod-shape rule list; task 2 holds the corpus.
- [`../research/t3-sandbox.md`](../research/t3-sandbox.md):
  - **§8.1**: the Flux layout;
  - **§8.2**: the guard objects, the full VAP rule list and why each rule exists;
  - **§8.3**: the runner values, used to render the positive control;
  - **§8.8**: the rollout order (A7);
  - **§9** row P0: how the spike applies these objects;
  - **§2.4** row A2: the threat each rule answers.
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md) §5 (Track A: A6 host block, A7 `sandbox-guards`, A8 runner). [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §4 L14 (runner hard caps) and §5 (memory limits standing rule).
- [`../rollout-plan.md`](../rollout-plan.md):
  - **§2**: the MI-4 row, plus the MI-12 row it unblocks;
  - **§2.2**: the operating rules;
  - **§12**: the infra bullet "`sandbox-guards` before any runner object" and "runner off the `apps` wait path".
- [mi-10's plan](../sprints/sprint-mi-10.md), task 2: the runner HelmRelease values your positive control must admit.
- **`../infra` files:**
  - `clusters/vps/{messaging,databases,apps}.yaml`, the Kustomization style;
  - `infrastructure/messaging/release.yaml`, the header-comment style;
  - `charts/project/` at 0.3.0 (`values.yaml`, `templates/deployment.yaml`, the `image.digest` render);
  - `hack/{host-verify.sh,host-lint.sh,expected-netpol.tsv}`;
  - `README.md`.

## Context

The runner will execute untrusted learner code. Its namespace must exist, fenced, **before** any runner object ([mi-10](../sprints/sprint-mi-10.md), MI-12).

MI-4 is that fence:
- an empty `xlearn-runner` namespace, born default-deny with **no DNS**;
- a **ValidatingAdmissionPolicy** that pins the exact runner pod shape and denies exec/attach;
- the L14 guard objects: RuntimeClass `xlearn-judge` → handler `judge`, PriorityClass −1000 `Never`, the ResourceQuota and LimitRange;
- a judge → runner NetworkPolicy.

It lives on its **own Kustomization `sandbox-guards`**, off the `apps` wait path. It's split from [mi-03](../sprints/sprint-mi-03.md)'s fences so a VAP stall never blocks NATS N3.

Live facts (read-only, 2026-09-24):
- k3s `v1.36.4+k3s1`;
- `admissionregistration.k8s.io/v1` `ValidatingAdmissionPolicy` is available, and none exist yet;
- k3s ships RuntimeClasses `crun`, `wasm*` and others; `xlearn-judge` is new;
- no namespace carries PSA labels;
- the only PriorityClasses are `longhorn-critical` and the system ones;
- the `judge` containerd handler **doesn't exist** until the Sat 2026-10-24 host window ([mi-09](../sprints/sprint-mi-09.md)).

The infra repo has **no CI**. Validate locally with `helm lint` and `helm template`, and against the live API server with `--dry-run=server`, which persists nothing. Don't use `kubectl kustomize`: infra directories have no `kustomization.yaml` (Flux generates one), and you must not add one.

The spike ([spk-01](../sprints/sprint-spk-01.md), Mon 2026-10-12) applies these manifests on a throwaway cluster, so merge this before then.

## Entry gates: verify first

Stop and report if any gate is unmet.

- [ ] Chart 0.3.0 is merged in `../infra` ([mi-01](../sprints/sprint-mi-01.md)): `grep '^version: 0.3.0' ../infra/charts/project/Chart.yaml`. The runner knobs (`runtimeClassName`, `priorityClassName`, `hostUsers`, `automountServiceAccountToken`, `dnsPolicy`, `image.digest`) exist.
- [ ] The MI-8 `host-verify --cluster` extension is merged ([mi-02](../sprints/sprint-mi-02.md)): `hack/expected-netpol.tsv` exists and `--cluster` checks NetworkPolicy presence.
- [ ] `cd ../infra && git checkout main && git pull`. Peer check: `gh pr list --state open` in both repos, `git worktree list`, ListAgents. No open PR touches `infrastructure/sandbox/`, `clusters/vps/sandbox.yaml` or `hack/expected-netpol.tsv`. mi-03 edits the `.tsv` too, so rebase whichever lands second.
- [ ] *(Soft)* If spk-01 has run, read its results table (VAP diff, `SETPCAP`, `procMount`, exec attempt) and fold it in.

## Do this (in order)

1. **Branch in `../infra`:** `feat/mi-4-sandbox-guards`.

2. **Guard objects [I]** (plan task 1). Write `infrastructure/sandbox/`, one kind per file, each with a header comment:
   - `namespace.yaml`: `xlearn-runner`, PSA `enforce: privileged`, `warn: baseline`, `audit: baseline`. The comment says the VAP is the fence.
   - `vap-pod-shape.yaml` and `vap-pod-shape-binding.yaml`:
     - `admissionregistration.k8s.io/v1`, `failurePolicy: Fail`;
     - `namespaceSelector` `kubernetes.io/metadata.name In [xlearn-runner]` on both the policy and the binding;
     - `CREATE` and `UPDATE` on `pods` and `pods/ephemeralcontainers`;
     - the plan's 10 rules as CEL, with `has()` guards and a `variables` entry for all containers;
     - binding `validationActions: [Warn, Audit]`.
   - `vap-no-exec.yaml` and its binding: `CONNECT` on `pods/exec` and `pods/attach` only (**not** `pods/portforward`), validation `request.operation != 'CONNECT'`, binding `[Warn, Audit]`.
   - `runtimeclass.yaml` (`xlearn-judge` → `judge`), `priorityclass.yaml` (`xlearn-sandbox-lowest`, −1000, `preemptionPolicy: Never`), `resourcequota.yaml`, `limitrange.yaml` (values in the plan), and `networkpolicies.yaml` with `default-deny-all` (Ingress+Egress, no rules) and `judge-to-runner` (ns `xlearn` AND instance `xlearn-judge`, TCP 8090).
   - `clusters/vps/sandbox.yaml`: the `sandbox-guards` Kustomization only. `./infrastructure/sandbox`, interval 1m, `wait`, `prune`, timeout 5m. No `dependsOn` and no decryption. Add a comment saying `runner` joins it later in mi-10, with `dependsOn: sandbox-guards`.
   - Add two rows to `hack/expected-netpol.tsv` in mi-02's three-column format (`namespace <TAB> name <TAB> added-by`): `xlearn-runner<TAB>default-deny-all<TAB>mi-14` and `xlearn-runner<TAB>judge-to-runner<TAB>mi-14`. Paste the same rows, byte-identical, into the **embedded copy** in `hack/host-verify.sh` between its `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv` markers. host-verify runs via `bash -s` and never reads the file.
   - Add a read-only VAP-binding presence check to `host-verify.sh --cluster`, using `get` only so mi-02's read-only lint still passes.
   - Add a line to the README's infrastructure listing for `infrastructure/sandbox/` and `sandbox.yaml`.

3. **Corpus + script [I]** (plan task 2):
   - Add `hack/sandbox-vap-test.sh` (`--expect warn|deny`, server dry-run only, a pass/fail table) and `hack/sandbox-vap-test/`.
   - The corpus is G1, B1–B14, Q1, X1/X2 and C1, as in the plan. **Every object is a `kind: Pod`**: the VAP matches only pods, so a Deployment dry-run proves nothing.
   - Build **G1** as in plan task 2:
     1. `helm template xlearn-runner charts/project --namespace xlearn-runner -f <draft runner values>` from chart 0.3.0, with values that mirror mi-10 task 2 and any 64-hex digest;
     2. extract the Deployment's `.spec.template` into a `kind: Pod` named `vap-g1` in `xlearn-runner`;
     3. set `serviceAccountName: default` and keep `automountServiceAccountToken: false`. The release's own ServiceAccount doesn't exist, so the ServiceAccount admission plugin would reject every pod before the VAP runs.
   - Each B/Q case is that Pod with one change. C1 is B1 with `namespace: default`.
   - **Embed the corpus in the script**, one heredoc per file between `# >>> sandbox-vap-test/<file>.yaml` / `# <<< sandbox-vap-test/<file>.yaml` markers, piped to `apply --dry-run=server -f -`. The script writes nothing on the node (mi-02's embedded-TSV precedent).
   - **Classify every response by source**, as in plan task 2:
     - a VAP warning starts `Validation failed for ValidatingAdmissionPolicy 'xlearn-runner-pod-shape'`;
     - a VAP denial names the policy and says `denied request`;
     - a PSA warning contains `would violate PodSecurity`, and it's recorded, never counted;
     - Q1 must match the LimitRange message;
     - any other 4xx (ServiceAccount, PSA-enforce, quota, RuntimeClass, PriorityClass) is a **FAIL**.
   - Extend `hack/host-lint.sh`: shellcheck `sandbox-*.sh`, check each embedded corpus heredoc equals its file, and allow only `apply --dry-run=server` and `create --raw` on the `exec`/`attach` subresource as write verbs in this script. Run it clean.

4. **Pre-merge validation [H]:**
   - `cat infrastructure/sandbox/*.yaml | ssh vps 'sudo k3s kubectl apply --dry-run=server -f -'`. This persists nothing.
     - The cluster-scoped objects (Namespace, VAPs, bindings, RuntimeClass, PriorityClass) must pass. The API server compiles every CEL expression on write, so fix any compile error.
     - The namespaced ones return `namespaces "xlearn-runner" not found`, because the dry-run didn't persist the Namespace. Schema-check them by piping them through `sed 's/namespace: xlearn-runner/namespace: default/'` into the same dry-run.
   - `helm lint charts/project -f <draft runner values>`, plus the G1 Pod's own server dry-run once PR 1 is live (step 6).
   - `hack/host-lint.sh` clean.

5. **PR 1 [I]:** open it with the object table, the rule list and the validation output. Merge it once validated (there's no CI; you have standing merge authority).
   - Wait for Flux: `ssh vps 'sudo k3s kubectl get kustomizations -n flux-system'` shows `sandbox-guards` Ready and `apps` Ready, with no restarts elsewhere.
   - `get pods -n xlearn-runner` shows no resources.
   - Read `status.typeChecking` on both VAPs (`get validatingadmissionpolicy <name> -o jsonpath='{.status.typeChecking}'`). CEL type-checking against the Pod schema only shows on the persisted object. Any `expressionWarnings` entry gets a follow-up PR before step 7.

6. **Prove, Warn phase [H]:**
   - Run `ssh vps 'bash -s -- --expect warn' < hack/sandbox-vap-test.sh`. The corpus is embedded, so copy nothing to the node.
   - Expect:
     - every B case admitted with a **VAP** warning naming its rule;
     - **G1 with no VAP warning**. Its expected PSA baseline warnings are `SYS_ADMIN` in `capabilities.add`, and `procMount: Unmasked` unless the API server relaxes that check for `hostUsers: false`; record which;
     - C1 with no warning at all;
     - Q1 denied by the LimitRange;
     - no ServiceAccount, quota or other unexpected error.
   - If G1 draws a **VAP** warning, fix the rule or the values in a follow-up PR **before** step 7. A PSA warning is never a reason to change a VAP rule.

7. **PR 2: flip to Deny [I].** Change both bindings to `validationActions: [Deny]`, and nothing else. Merge it and wait for reconcile.

8. **Prove, Deny phase [H]:**
   - Run `--expect deny`:
     - B1–B14 denied **by `xlearn-runner-pod-shape`** (≥ 8 required shapes). A denial from anything else is a FAIL;
     - X1/X2 exec and attach CONNECT denied by `xlearn-runner-no-exec`;
     - Q1 denied by the LimitRange;
     - G1 admitted, with only the expected PSA warnings;
     - C1 unaffected.
   - If X1/X2 return NotFound before admission, record it. Real-exec proof stays with spk-01 (throwaway) and mi-10 (real runner pod).
   - Paste the table into PR 2.

9. **Verify [H]:**
   - `hack/host-lint.sh` is clean: the embedded `.tsv` and corpus equal their files.
   - `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`: green, including the NetworkPolicy presence (the two new names) and the VAP-binding check.
   - **Negative check:** `ssh vps "bash -s -- --cluster --netpol-file <(printf 'xlearn-runner\tno-such-policy\tneg\n')" < hack/host-verify.sh` must **FAIL** `cluster.netpol`, which proves the presence check isn't vacuous. Paste both runs into the PR.
   - `ssh vps 'sudo k3s kubectl get kustomizations -n flux-system -o wide'`: `apps` is unaffected, and nothing lists `sandbox-guards` in `dependsOn`.

10. **Record [X]:** a small xlearn docs PR for `docs/v2/status.md` and this sprint's file (see Update status).

## Constraints

- **GitOps only.** Every cluster change is a `../infra` PR reconciled by Flux. Never `kubectl apply` by hand. `--dry-run=server` and `create --raw` against a non-existent pod persist nothing and are allowed as validation.
- **Infra PRs stand alone** (rollout §2.2). There's no xlearn tag in this sprint.
- **Off the critical path:** `sandbox-guards` has no `dependsOn`, and `apps` must never wait on it. The runner Kustomization is mi-10's job; don't create it.
- **Selectors:** the VAP binds by `kubernetes.io/metadata.name`, never a custom label. The policy's own `matchConstraints` also carries the selector, as defence in depth.
- **Don't deny `pods/portforward`.** mi-10's acceptance suite needs it.
- **Never loosen** a t3 §8.2 rule without spike evidence. Record any change in the decisions log.
- **D34:** no Flux `Alert`, `Provider`, ping or opscheck. Presence is checked on demand by `host-verify --cluster`.
- **Memory-sum rule (ADR-0035 §5):** no pod is created. The quota's 3 Gi of limits counts only when mi-10 creates the runner; mi-10 checks it with `host-verify --cluster --with-runner`.
- **Parallel sessions:** check peers' PRs and worktrees before pushing each PR. Don't take ADR numbers; none are needed.

## Deliverables

- **In `../infra/infrastructure/sandbox/`:** the namespace, 2 VAPs and 2 bindings, the RuntimeClass, PriorityClass, ResourceQuota, LimitRange and 2 NetworkPolicies.
- **`../infra/clusters/vps/sandbox.yaml`:** the `sandbox-guards` Kustomization.
- **Proof tooling:** `../infra/hack/sandbox-vap-test.sh` with the corpus embedded, and its reviewed sources in `hack/sandbox-vap-test/` (Pod objects), with `host-lint.sh` coverage (shellcheck + embedded-copy check).
- **host-verify:** the `hack/expected-netpol.tsv` rows and the same rows in its embedded copy, plus the VAP-binding presence check.
- **Two `../infra` PRs,** Warn then Deny, with the proof tables in their descriptions.
- **An xlearn docs PR** for the status updates.

## Update status

- **This sprint's file:** set the task rows in [`../sprints/sprint-mi-14.md`](../sprints/sprint-mi-14.md) to 🔄 or ✅, and _Overall_ to ✅.
- **[`../status.md`](../status.md):**
  - **Sprint board:** mi-14 ✅.
  - **MI table, row MI-4:** ✅, with the infra PR numbers and date. mi-10's gate "MI-4 live" is now met.
  - **Decisions log:**
    - the Warn→Deny two-step;
    - the image regex allowing `:tag@sha256`;
    - `portforward` not denied;
    - the CONNECT dry-run behaviour observed;
    - any spike diff folded in, or handed to mi-10 as a pre-runner follow-up PR.
- **ADRs:** none expected. The design is ADR-0030 A7 and t3 §8.2.

## Done when (acceptance)

- [ ] `sandbox-guards` is Ready and off the `apps` path, and `xlearn-runner` is empty with PSA `privileged`/`baseline`/`baseline`.
- [ ] The VAP denies ≥ 8/8 required bad shapes (and all of B1–B14) plus exec/attach CONNECT, or records why CONNECT is handed to spk-01/mi-10. Every result is classified by source; none passes on a ServiceAccount, PSA-enforce or quota error.
- [ ] The chart-0.3.0 runner shape (G1, a Pod) is admitted with no VAP warning (its expected PSA warnings recorded), and a pod in another namespace is unaffected (C1).
- [ ] Both VAPs show no `status.typeChecking` warnings.
- [ ] The RuntimeClass, PriorityClass (−1000, `Never`), ResourceQuota and LimitRange are live, and Q1 is denied.
- [ ] `default-deny-all` (no DNS) and `judge-to-runner` are present, and `host-verify --cluster` is green, including the VAP-binding check. `host-lint.sh` is clean, and the `--netpol-file` negative check FAILs.
- [ ] status.md MI-4 and the decisions are recorded.

**Ship at session end** per AGENT.md land-and-sync, with this sprint's release action, **infra PR(s) only**:
1. `../infra` PR 1 (Warn), then PR 2 (Deny): conventional commits (`feat(sandbox): …`) with the required attribution lines. Merge each once validated and proven; there's no CI in infra.
2. Let Flux reconcile, and verify live as above.
3. Open an xlearn docs PR for the status updates. Merge it on CI green.
4. Run `git checkout main && git pull` in **both** repos. There's no tag.
