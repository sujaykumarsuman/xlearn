# Sprint mi-14 — sandbox-guards: empty default-deny xlearn-runner namespace + VAP (MI-4)

> **Milestone:** MI — cluster safe for untrusted code (rollout step **MI-4**, ADR-0030 Track A **A7**)   ·   **Track:** infra
> **Prereqs:** [mi-01](sprint-mi-01.md) (chart 0.3.0, used to render the runner shape for the positive control), [mi-02](sprint-mi-02.md) (`host-verify --cluster` NetworkPolicy presence check + `hack/expected-netpol.tsv`)
> **Unblocks:** [mi-10](sprint-mi-10.md) (runner dark: its `runner` Kustomization `dependsOn: sandbox-guards`). Soft input to [spk-01](sprint-spk-01.md), which applies these manifests on the throwaway cluster.
> **Release action:** **infra PR(s) only.** Two `../infra` PRs, merged in order: (1) the guard objects with the VAP binding at `[Warn, Audit]`, then (2) the one-line flip to `[Deny]`. A small xlearn docs PR carries the status update. No xlearn tag.
> **Calendar:** week 2 (Mon 2026-10-05 → Fri 2026-10-09). Merge it before spike week (Mon 2026-10-12), so [spk-01](sprint-spk-01.md) applies these manifests rather than the t3 §8.2 draft.
> **Execute with:** [`../prompts/prompt-mi-14.md`](../prompts/prompt-mi-14.md). One prompt, one session.
>
> Sprint ids `mi-NN` are not rollout step ids `MI-N`. This sprint executes rollout step **MI-4**, which is also limit **L14**'s cluster half.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | MI-4 guard objects + `clusters/vps/sandbox.yaml` (`sandbox-guards`), binding at `[Warn, Audit]` | I | ⬜ |
| 2 | VAP proof corpus + `hack/sandbox-vap-test.sh` (server dry-run only) | I | ⬜ |
| 3 | Prove on the node, Warn phase: bad shapes warn, the runner shape is clean, other namespaces untouched | H | ⬜ |
| 4 | Flip the binding to `[Deny]` | I | ⬜ |
| 5 | Prove on the node, Deny phase: ≥ 8 bad shapes denied, CONNECT exec/attach denied, runner shape admitted | H | ⬜ |
| 6 | `host-verify --cluster`: expected policies + VAP bindings present; `apps` unaffected | H | ⬜ |
| 7 | Record (status.md MI-4 row, decisions, spike hand-off) | X | ⬜ |

> **Keep this current.** Set a task to 🔄 when you start it, to ✅ when its acceptance bullet passes, and to ⛔ if it's blocked (say why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, the MI table row **MI-4**, and the decisions log. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-3 chart 0.3.0 is merged ([mi-01](sprint-mi-01.md)). Its `runtimeClassName`, `priorityClassName`, `hostUsers`, `automountServiceAccountToken`, `dnsPolicy` and `image.digest` knobs render the runner shape used as the positive control.
- [ ] The MI-8 `host-verify --cluster` extension is merged ([mi-02](sprint-mi-02.md)), including `hack/expected-netpol.tsv` and the NetworkPolicy presence check.
- [ ] Local `../infra` `main` is synced. Peer check: no open PR touches `infrastructure/sandbox/`, `clusters/vps/sandbox.yaml` or `hack/expected-netpol.tsv` ([mi-03](sprint-mi-03.md) also edits the `.tsv`, so rebase whichever merges second).
- [ ] *(Soft)* If [spk-01](sprint-spk-01.md) has already run, its results table is at hand. That table is the VAP diff against the t3 §8.2 draft, and it covers `SETPCAP`, `procMount` and the real exec attempt. Fold it in.

## Goal

Stand up the fence the runner will live in before any runner object exists. That means an **empty, born-default-deny `xlearn-runner` namespace** (no DNS) with:
- its **ValidatingAdmissionPolicy**, pinning the exact runner pod shape and denying exec/attach;
- the L14 guard objects: RuntimeClass, PriorityClass −1000, ResourceQuota, LimitRange;
- the **judge → runner** NetworkPolicy.

All of it sits on its **own Flux Kustomization `sandbox-guards`**, off the `apps` wait path. Nothing depends on it until [mi-10](sprint-mi-10.md) adds `runner`.

It's split from the other fences ([mi-03](sprint-mi-03.md)) so a VAP problem never blocks NATS N3 or the xlearn ingress.

## Scope

**In**
- **`../infra/infrastructure/sandbox/`**, raw manifests, one kind per file:
  - Namespace `xlearn-runner` with its PSA labels;
  - two VAPs and their two bindings: pod shape, and no exec/attach;
  - RuntimeClass `xlearn-judge` → handler `judge`;
  - PriorityClass `xlearn-sandbox-lowest`;
  - ResourceQuota and LimitRange;
  - NetworkPolicies `default-deny-all` and `judge-to-runner`.
- **`../infra/clusters/vps/sandbox.yaml`**, holding **only** the `sandbox-guards` Kustomization.
- **`../infra/hack/sandbox-vap-test.sh`** and the corpus `hack/sandbox-vap-test/*.yaml` (Pod objects), embedded in the script between markers. It's server dry-run only, re-runnable by [spk-01](sprint-spk-01.md) and [mi-10](sprint-mi-10.md). Add it to `hack/host-lint.sh` coverage: shellcheck plus the embedded-copy check.
- **`hack/expected-netpol.tsv`** and its embedded copy in `host-verify.sh` gain the two `xlearn-runner` rows. `host-verify --cluster` gains a read-only presence check for the two VAP bindings.

**Out (later sprints)**
- **Runner objects** go to [mi-10](sprint-mi-10.md) (MI-12): the `runner` Kustomization, the runner HelmRelease/Service, the 2nd ImageUpdateAutomation, the runner bearer secret, and any runner pod.
- **The host sandbox block** goes to [mi-09](sprint-mi-09.md) and the Sat 2026-10-24 host window: the containerd `judge` handler, AppArmor and seccomp profiles, the subuid range, sysctls, L23 kubelet args. Until then the `judge` handler doesn't exist on the node, so the RuntimeClass is inert.
- **judge's own policies and egress** go to [m3-07](sprint-m3-07.md) (MI-13).
- **PSA labels on other namespaces** go to [mi-08](sprint-mi-08.md).
- **Other fences:** xlearn ingress and `databases`/`messaging` go to [mi-03](sprint-mi-03.md); xlearn egress goes to [mi-11](sprint-mi-11.md).
- **A real exec attempt against a running runner pod.** Nothing can run here before MI-11, so that happens on the spike's throwaway cluster ([spk-01](sprint-spk-01.md)) and again at [mi-10](sprint-mi-10.md).

## Tasks

### 1 · MI-4 guard objects [I]

These are raw manifests under `infrastructure/sandbox/`, from [t3 §8.2](../research/t3-sandbox.md) and [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) L14. Give each file a short header comment in the house style of `infrastructure/messaging/release.yaml`.

| Object | Spec |
|---|---|
| **Namespace** `xlearn-runner` | Labels `pod-security.kubernetes.io/enforce: privileged`, `warn: baseline`, `audit: baseline`. The comment should say why: baseline forbids adding `SYS_ADMIN`; the pod is never `privileged: true`; **the VAP is the fence**. |
| **VAP** `xlearn-runner-pod-shape` + binding | `failurePolicy: Fail`. `matchConstraints.namespaceSelector`: `kubernetes.io/metadata.name In [xlearn-runner]` (the label the API server sets). `resourceRules`: `CREATE`, `UPDATE` on `pods` and `pods/ephemeralcontainers`. Validations: the rule list below. |
| **VAP** `xlearn-runner-no-exec` + binding | Same selector and `failurePolicy`. `CONNECT` on `pods/exec` and `pods/attach` only. Validation: `request.operation != 'CONNECT'`, message "exec/attach into xlearn-runner is denied; delete the pod instead". **Don't match `pods/portforward`**: mi-10's acceptance suite reaches the runner through a node-local port-forward. |
| **RuntimeClass** `xlearn-judge` | `handler: judge`. |
| **PriorityClass** `xlearn-sandbox-lowest` | `value: -1000`, `preemptionPolicy: Never`, `globalDefault: false`. |
| **ResourceQuota** `sandbox` | `pods: 2`; `requests.cpu: 500m`, `requests.memory: 1Gi`; `limits.cpu: "2"`, `limits.memory: 3Gi`; `persistentvolumeclaims: 0`; `services: 1`; `services.nodeports: 0`; `services.loadbalancers: 0`. |
| **LimitRange** `sandbox` | `Container`: `defaultRequest` 50m/64Mi, `default` 500m/512Mi, `max` 2/3Gi. `Pod`: `max` 2/3Gi. Every container then carries a memory limit (ADR-0035 §5 standing rule), and the quota admits pods that don't set their own. |
| **NetworkPolicy** `default-deny-all` | `podSelector: {}`, `policyTypes: [Ingress, Egress]`, **no rules** (no DNS). |
| **NetworkPolicy** `judge-to-runner` | `podSelector: {}`. Ingress from **one** `from` element holding both `namespaceSelector` (`kubernetes.io/metadata.name: xlearn`) **and** `podSelector` (`app.kubernetes.io/instance: xlearn-judge`), so the two are ANDed. Port TCP 8090. |

**Pod-shape rules.** Use CEL, with `has()` guards on every optional field: an unguarded missing field is a runtime error, and under `failurePolicy: Fail` that means deny. Use `variables` for `allContainers = containers + initContainers`.

1. `object.spec.runtimeClassName == 'xlearn-judge'`, and `hostUsers == false` (present and false).
2. `automountServiceAccountToken == false` and `enableServiceLinks == false`, both present. `priorityClassName == 'xlearn-sandbox-lowest'`.
3. No `hostNetwork`, `hostPID`, `hostIPC`, `shareProcessNamespace`, `securityContext.sysctls`, or container `hostPort`.
4. Volumes ⊆ {`emptyDir`, `secret` with `secretName == 'runner-auth'`}. No `imagePullSecrets`.
5. Every image matches `^ghcr\.io/sujaykumarsuman/xlearn-runner(:[\w][\w.-]{0,127})?@sha256:[a-f0-9]{64}$`. The optional `:tag` is there because chart 0.3.0's `image.digest` may render `repo:tag@sha256:…`; check the mi-01 template.
6. Per container:
   - `privileged` absent or false;
   - `allowPrivilegeEscalation == false`;
   - `capabilities.add` ⊆ {`SYS_ADMIN`, `SETUID`, `SETGID`, `SETPCAP`, `KILL`}, spelled without `CAP_`;
   - `capabilities.drop` contains `ALL`;
   - `procMount` ∈ {`Default`, `Unmasked`}.
7. Seccomp is `Localhost` with `localhostProfile == 'profiles/xlearn-runner.json'` at **pod and container** level. It's never `Unconfined` or `RuntimeDefault`. AppArmor is `Localhost` with profile `xlearn-runner`.
8. `runAsUser: 0` is allowed only with `hostUsers: false`, which rule 1 already implies. Keep an explicit check anyway.
9. Probes use `httpGet` only. There are no `exec` handlers in `lifecycle`.
10. No `ephemeralContainers`, on CREATE and on the `pods/ephemeralcontainers` UPDATE.

**Flux Kustomization.** Add `clusters/vps/sandbox.yaml` with **only** the `sandbox-guards` Kustomization (t3 §8.1): `path: ./infrastructure/sandbox`, `interval: 1m`, `wait: true`, `prune: true`, `timeout: 5m`.
- It has **no `dependsOn`**, and **nothing depends on it yet**. `apps` stays untouched.
- There's no SOPS decryption: the guards hold no secrets.

**First binding and pre-merge check.** The **first PR** ships both bindings at `validationActions: [Warn, Audit]`. Before merging:
- **Don't run `kubectl kustomize`, and don't add a `kustomization.yaml`.** No infra directory except `clusters/vps/flux-system` has one: Flux generates it, and `kubectl kustomize` on a plain directory errors. Keep the house convention.
- Server dry-run the new files only: `cat infrastructure/sandbox/*.yaml | ssh vps 'sudo k3s kubectl apply --dry-run=server -f -'`. That's validation only and persists nothing.
  - The cluster-scoped objects (Namespace, VAPs, bindings, RuntimeClass, PriorityClass) must pass. The API server **compiles** every CEL expression on write, so a syntax or compile error fails here.
  - The namespaced objects (ResourceQuota, LimitRange, NetworkPolicies) get `namespaces "xlearn-runner" not found`, because a dry-run doesn't persist the Namespace. Schema-check them by piping them through `sed 's/namespace: xlearn-runner/namespace: default/'` into the same dry-run.
- **After PR 1 reconciles**, read `status.typeChecking` on both VAPs (`k3s kubectl get validatingadmissionpolicy xlearn-runner-pod-shape -o jsonpath='{.status.typeChecking}'`, and the same for `xlearn-runner-no-exec`). Type-checking against the Pod schema happens only on the persisted object. Any `expressionWarnings` entry is a bug: fix it in a follow-up PR before the Deny flip.

### 2 · VAP proof corpus + script [I]

Add `hack/sandbox-vap-test.sh`, run over `ssh vps`. It only calls `kubectl apply --dry-run=server` and one raw `create --raw` POST, so it persists nothing. Give it `--expect warn|deny`, and have it print a pass/fail table. It must be shellcheck clean and covered by `host-lint.sh`.

**The corpus travels inside the script**, following [mi-02](sprint-mi-02.md)'s embedded-TSV precedent. The script runs as `ssh vps 'bash -s …' < hack/sandbox-vap-test.sh`, so it never sees `hack/`, and copying files to the node is a host write this sprint avoids.
- The reviewed sources are `hack/sandbox-vap-test/*.yaml`.
- Each is embedded byte-identically as a heredoc between `# >>> sandbox-vap-test/<file>.yaml` and `# <<< sandbox-vap-test/<file>.yaml` markers.
- The script pipes each heredoc into `k3s kubectl apply --dry-run=server -f -`. It writes no file on the node.
- `host-lint.sh` fails when an embedded copy differs from its file, as it does for the TSVs.

**Every corpus object is a `kind: Pod`.** The VAP matches only `pods` and `pods/ephemeralcontainers`, and the LimitRange and quota also act on Pods. A dry-run of the chart's Deployment never creates a Pod, so it would be trivially admitted and prove nothing. Build G1 like this:
1. `helm template xlearn-runner charts/project --namespace xlearn-runner -f <draft runner values>` on chart 0.3.0, with the values from [t3 §8.3](../research/t3-sandbox.md) and [mi-10](sprint-mi-10.md) task 2 and any 64-hex sha256 digest.
2. Take the Deployment's `.spec.template` (for example with `yq`) and wrap it as `apiVersion: v1`, `kind: Pod`, `metadata: {name: vap-g1, namespace: xlearn-runner, labels: <the template's labels>}`, `spec: <the template's spec>`.
3. **Set `serviceAccountName: default`** and keep `automountServiceAccountToken: false`. The chart renders `serviceAccountName: <release>` (`charts/project/templates/deployment.yaml`, when `serviceAccount.create`), and that ServiceAccount doesn't exist in `xlearn-runner`: runner objects belong to mi-10, and dry-run objects aren't persisted. The ServiceAccount admission plugin would reject every corpus pod before the VAP runs. The `default` ServiceAccount exists in every namespace, and the VAP doesn't check the SA name.

Commit the extracted Pod, not the Deployment. Every B, Q and C case is a copy of that Pod with one change.

**Classify every response by its source.** The script must never count a warning or a denial just because one happened:
- **VAP warning** (Warn phase): the line starts with `Validation failed for ValidatingAdmissionPolicy 'xlearn-runner-pod-shape'`.
- **VAP denial** (Deny phase): the error names `ValidatingAdmissionPolicy 'xlearn-runner-pod-shape'` (or `'xlearn-runner-no-exec'` for X1/X2) and says `denied request`.
- **PSA warning:** contains `would violate PodSecurity`. The script records it and never counts it as a VAP result.
- **LimitRange denial:** `maximum memory usage per Container is 3Gi` (or the `Pod` form). Only Q1 may produce it.
- **Anything else is a FAIL, not a PASS.** That includes a ServiceAccount, PSA-enforce, quota, RuntimeClass or PriorityClass error, or any other 4xx.

Record the exact strings from the first node run in the PR, and tighten the patterns to match them.

- **G1 (good).** The extracted runner Pod above. Expect it **admitted** with **no VAP warning**. It must also fit the quota and LimitRange. It **does** draw PSA `warn: baseline` warnings, because the namespace warns at baseline and the runner shape needs `privileged` enforcement (t3 §8.2):
  - `SYS_ADMIN` in `capabilities.add` always warns;
  - `procMount: Unmasked` warns unless the API server relaxes the baseline check for `hostUsers: false` pods (the `UserNamespacesPodSecurityStandards` relaxation). Record which case you see.
  
  These PSA warnings are expected. They don't mean a VAP rule or the runner values are wrong.
- **B1–B14 (bad).** Each is G1 plus exactly one violation. Expect a VAP warning naming its rule (Warn phase) or a VAP denial (Deny phase). Every B case inherits G1's PSA warnings, and some add their own (B1, B2, B5, B6, B7, B13). The script records them and ignores them.

  | Id | Violation |
  |---|---|
  | B1 | `privileged: true` |
  | B2 | `hostPID: true` |
  | B3 | `automountServiceAccountToken` true, or unset |
  | B4 | image by tag, or from another repo |
  | B5 | seccomp `Unconfined` (B5b: `RuntimeDefault`) |
  | B6 | cap add `NET_ADMIN` |
  | B7 | a `hostPath` volume |
  | B8 | no `runtimeClassName` |
  | B9 | `hostUsers` true, or unset |
  | B10 | `imagePullSecrets` |
  | B11 | an `exec` liveness probe or lifecycle hook |
  | B12 | `shareProcessNamespace` |
  | B13 | a `hostPort` |
  | B14 | a PVC volume |

- **Q1 (quota/L14).** G1 with a 4Gi memory limit. The LimitRange **denies** it in both phases, and the denial must match the LimitRange message above; a quota or any other error is a FAIL.
- **X1 (exec CONNECT).** `k3s kubectl create --raw "/api/v1/namespaces/xlearn-runner/pods/vap-probe/exec?command=true&stdout=true" -f /dev/null`. Admission runs in the connect handler before the pod lookup, so expect a VAP deny (Deny phase). **X2** does the same for `attach`.
  - If the API server answers NotFound before admission, record that. The real-exec proof then stays with [spk-01](sprint-spk-01.md) and [mi-10](sprint-mi-10.md).
- **C1 (control).** B1 with `metadata.namespace: default`, dry-run there. Expect it **admitted with no warning of any kind**: the VAP's selector doesn't match `default`, and `default` carries no PSA labels, so it's PSA-privileged with no `warn` level. `serviceAccountName: default` exists there too. That proves the selector's scope.

### 3 · Prove on the node, Warn phase [H]

After PR 1 merges, check `k3s kubectl get kustomizations -n flux-system`: `sandbox-guards` Ready and `apps` Ready and unchanged. Check both VAPs' `status.typeChecking` (task 1). Then run `ssh vps 'bash -s -- --expect warn' < hack/sandbox-vap-test.sh`:
- every B case is admitted with a **VAP** warning naming its rule;
- G1 is admitted with **no VAP warning**. It shows only the expected PSA baseline warnings from task 2 (`SYS_ADMIN`, and `procMount` unless relaxed);
- C1 is admitted with no warning at all;
- Q1 is denied by the LimitRange;
- no case hits an unexpected error (ServiceAccount, quota, RuntimeClass, PriorityClass).

Also check `k3s kubectl get pods -n xlearn-runner`: **no resources**.

If G1 draws a **VAP** warning, the rule, the runner values, or the chart 0.3.0 render is wrong. Fix it in a follow-up PR before the flip. PSA warnings on G1 are expected and are **not** a reason to change a VAP rule.

### 4 · Flip to Deny [I]

PR 2 changes both bindings to `validationActions: [Deny]` and nothing else.

### 5 · Prove on the node, Deny phase [H]

Run `ssh vps 'bash -s -- --expect deny' < hack/sandbox-vap-test.sh`:
- **B1–B14 denied by the VAP**, with the denial naming `xlearn-runner-pod-shape`. At least 8 of them are the rollout's required shapes (privileged, hostPID, token automount, wrong image, Unconfined seccomp, extra caps, hostPath, exec). A B case denied by anything else is a FAIL;
- X1 and X2 denied by `xlearn-runner-no-exec`, or recorded as above;
- Q1 denied by the LimitRange;
- G1 admitted, with only the expected PSA warnings;
- C1 unaffected: admitted with no warning.

The namespace stays empty. Paste the table into PR 2 and status.md.

If [spk-01](sprint-spk-01.md) already reported a diff against the t3 draft (whether `SETPCAP` is needed, `procMount`, CONNECT behaviour), it lands in PR 1 or PR 2. If the spike reports **after** this sprint merges, its diff is a small follow-up infra PR taken by [mi-10](sprint-mi-10.md) before the first runner pod. Note that in status.md.

### 6 · Verify [H]

- Add two rows to `hack/expected-netpol.tsv` (in PR 1), in [mi-02](sprint-mi-02.md)'s three-column format `namespace <TAB> name <TAB> added-by`: `xlearn-runner<TAB>default-deny-all<TAB>mi-14` and `xlearn-runner<TAB>judge-to-runner<TAB>mi-14`.
- **Update the embedded copy too.** `host-verify.sh` never reads the `.tsv`: it runs via `bash -s` and carries the table between its `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv` markers. Paste the same rows there, byte-identical.
- Add a read-only presence check to `host-verify.sh --cluster` for the two VAP bindings (`get validatingadmissionpolicybinding`). A missing binding is a FAIL.
- Run `hack/host-lint.sh` **clean before each PR**. It fails when an embedded copy differs from its file (the `.tsv` and this sprint's corpus), and it runs shellcheck and the read-only verb check.
- Run `ssh vps 'bash -s -- --cluster' < hack/host-verify.sh`: it's green, and the NetworkPolicy and VAP checks pass.
- **Negative check.** Prove the new rows are really checked: `ssh vps "bash -s -- --cluster --netpol-file <(printf 'xlearn-runner\tno-such-policy\tneg\n')" < hack/host-verify.sh` must **FAIL** `cluster.netpol`. The node's bash evaluates the `<(…)`, and mi-02's `--netpol-file` override reads it. Paste both runs into the PR.
- `k3s kubectl get kustomizations -n flux-system -o wide`: `apps` is Ready, and nothing lists `sandbox-guards` in `dependsOn`.

### 7 · Record [X]

In xlearn's `docs/v2/status.md`:
- MI table row **MI-4** ✅, with the two infra PR numbers and the date;
- Sprint board mi-14 ✅;
- the corpus table (or a link to PR 2);
- decisions log:
  - the Warn→Deny two-step;
  - the image regex allowing `:tag@sha256`;
  - `portforward` deliberately not denied;
  - the CONNECT test outcome;
  - any spike diff folded in, or handed to mi-10.

## Acceptance criteria

- [ ] `sandbox-guards` is Ready and off the `apps` path. `xlearn-runner` exists with PSA `privileged`/`baseline`/`baseline` and **no pods**.
- [ ] The VAP denies **≥ 8/8 required bad shapes** (and the rest of B1–B14) and exec/attach CONNECT, or records why X1/X2 can't be proven here and hands them to spk-01/mi-10.
- [ ] The chart-rendered runner shape (G1, extracted as a Pod) is **admitted** with no VAP warning (expected PSA baseline warnings recorded), and a pod in another namespace is **unaffected**. Every corpus result is classified by source, and no case passes on a ServiceAccount, PSA-enforce or quota error.
- [ ] Both VAPs show no `status.typeChecking` warnings.
- [ ] RuntimeClass `xlearn-judge`, PriorityClass `xlearn-sandbox-lowest` (−1000, `Never`), the ResourceQuota (pods 2, 500m/1Gi req, 2/3Gi lim, PVCs 0, services 1, NodePorts 0, LBs 0) and the LimitRange are live. Q1 is denied.
- [ ] `default-deny-all` (no DNS) and `judge-to-runner` (ns `xlearn` AND `xlearn-judge` → TCP 8090) are present, and the `host-verify --cluster` NetworkPolicy and VAP-binding checks are green. `host-lint.sh` is clean (embedded `.tsv` and corpus equal their files), and the `--netpol-file` negative check FAILs as it should.
- [ ] Status and decisions are recorded, and any spike diff is folded in or handed to mi-10.

## Release

**Infra PR(s) only, with no xlearn tag.**
1. `../infra` PR 1: the guard objects, `clusters/vps/sandbox.yaml`, the corpus and script, `expected-netpol.tsv`, and the host-verify VAP check, with the bindings at `[Warn, Audit]`.
2. `../infra` PR 2: the `[Deny]` flip.
3. An xlearn docs PR for the status updates.

Each infra PR stands alone and is never folded into a tag (rollout §2.2). None of them restarts anything: no pod exists or changes, so there's no Hostinger snapshot or image check to do.

## Definition of Done

- Both infra PRs are merged and reconciled by Flux; nothing was applied by hand, since dry-runs aren't applies.
- The corpus is proven in both phases on the live node.
- `host-verify --cluster` is green.
- The status is updated (this file and [`../status.md`](../status.md)).
- Local `../infra` and xlearn `main` are synced.

## Risks / watch-outs

- **A VAP with `failurePolicy: Fail` and a wrong selector could block pods elsewhere.**
  - The selector keys on `kubernetes.io/metadata.name`, a label the API server sets.
  - PR 1 ships **Warn**, and C1 proves another namespace is untouched before the Deny flip.
  - CEL runtime errors fail closed only for matched requests, so guard every optional field with `has()`.
- **The positive control is the real safety net.** A rule that denies the legit runner shape would only show at [mi-10](sprint-mi-10.md). G1 must come from chart 0.3.0 with the real runner values, digest format included, extracted as a **Pod** (a Deployment dry-run never reaches the VAP) with `serviceAccountName: default` (the release's SA doesn't exist yet).
- **PSA warnings look like VAP failures.** The namespace warns at baseline, so G1 always draws PSA warnings. Classify by source (task 2); never loosen a VAP rule because of a PSA warning.
- **`pods/portforward` must stay allowed.** mi-10's acceptance suite uses a node-local port-forward. The VAP denies only `exec` and `attach`.
- **The CONNECT dry-run may not reach admission** (NotFound first). Record the observed behaviour. A real exec attempt is proven on spk-01's throwaway cluster and re-checked at mi-10 once a runner pod exists.
- **RuntimeClass `judge` has no handler on the node until the host window (MI-11).** That's harmless for an empty namespace, and mi-10 is gated on MI-11.
- **The PSA `privileged` label reads as alarming.** The namespace comment and the README must say the VAP is the fence.
- **The spike may change the rule list** (`SETPCAP`, `procMount`). Fold it in, or hand it to mi-10. Never loosen a rule without the spike's evidence.
