# Sprint mi-10 — Runner dark on production (MI-12)

> **Milestone:** MI — infra-first track (rollout step **MI-12**; sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** infra · **Order:** 41
> **Prereqs:** [mi-14](sprint-mi-14.md) (MI-4 `sandbox-guards`) · [mi-08](sprint-mi-08.md) (MI-11a) · [mi-09](sprint-mi-09.md) + the Oct host window (MI-11) · [m3-15](sprint-m3-15.md) (the `runner-v1.0.0` image)
> **Unblocks:** [m3-11](sprint-m3-11.md) (M3 hard checklist item "MI-12 done"). This is preferred, not required, before [m3-07](sprint-m3-07.md) (judge dark).
> **Release action:** **infra PR(s) only** (`runner-v1.0.0` was already cut in m3-15). Status goes in an xlearn docs PR (merge only, no tag).
> **Calendar:** early November (after the 2026-10-24 window and m3-15)
> **Execute with:** [`../prompts/prompt-mi-10.md`](../prompts/prompt-mi-10.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pre-flight: gates, host, memory sum, render + VAP dry run | H | ⬜ |
| 2 | SOPS rule + runner bearer token | I | ⬜ |
| 3 | Image automation (image before policy): ImageRepository/ImagePolicy + 2nd IUA | I | ⬜ |
| 4 | `runner` Kustomization + HelmRelease | I | ⬜ |
| 5 | Acceptance suite on prod + TL calibration | H | ⬜ |
| 6 | Verify | H | ⬜ |
| 7 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track row MI-12 + the runner stream row).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-4 live** ([mi-14](sprint-mi-14.md)). This means:
  - the `sandbox-guards` Kustomization is Ready;
  - the `xlearn-runner` namespace is empty;
  - the VAP bindings are at **`[Deny]`** (mi-14's PR 2), and the VAP denies 8/8 bad shapes and CONNECT exec;
  - any spike VAP diff that reached mi-14 after it merged is recorded in status.md as handed to mi-10. Task 1 lands it;
  - the RuntimeClass `xlearn-judge`, the PriorityClass `xlearn-sandbox-lowest`, the Quota and the LimitRange exist;
  - `default-deny-all` and `judge-to-runner` are present.
- [ ] **MI-11 host window executed** (ev-host-window, prepared by [mi-09](sprint-mi-09.md)) and `host-verify --expect-sandbox --cluster` green:
  - the `judge` runtime is listed;
  - AppArmor `xlearn-runner` is in enforce mode;
  - the seccomp profile and the subuid range are present;
  - L23 shows in `configz`.
- [ ] **MI-11a merged** ([mi-08](sprint-mi-08.md)). [ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) requires MI-11a before MI-12.
- [ ] **`runner-v1.0.0` image exists** ([m3-15](sprint-m3-15.md)): `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0`, with its digest recorded and a reproducible build, and it's **anonymously pullable** (m3-15 task 8 ✅; if status.md still shows its ⛔ owner item for the package visibility, this gate is unmet).
- [ ] **ADR-0030 Accepted** ([m3-03](sprint-m3-03.md) task 1). The image implements the mechanism the spike chose. The caps list (5 caps, or 4 without SETPCAP) is recorded in status.md ([mi-09](sprint-mi-09.md) task 1).
- [ ] **Parallel sessions:** no open peer PR touches `clusters/vps/sandbox.yaml`, `runner/`, `apps/image-automation.yaml` or `.sops.yaml` (`gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents).

## Goal

Deploy **`runner-v1.0.0` dark** in the `xlearn-runner` namespace:
- reachable only from judge's selector (judge itself arrives with [m3-07](sprint-m3-07.md), so until then
  nothing in the cluster can reach it);
- off the `apps` wait path;
- on its own release stream ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)).

Then pass the **acceptance suite on the production node**: step A8 of
[ADR-0030 §5](../../adr/0030-runner-technology-and-host-hardening.md#5-host-and-cluster-hardening), with
isolation markers, network and syscall probes, INV-14 OOM placement, cleanup invariants, and calibration of
`baseline@1` plus the per-language time-limit multipliers.

After this sprint, M3's hard checklist can tick MI-12, and judge has a runner to call.

## Scope

**In**
- `.sops.yaml` creation rule for `runner/secrets/`. Today the rules cover only `apps/secrets/` and `infrastructure/database/cluster/`.
- The runner bearer token, `runner/secrets/runner-auth.enc.yaml` (Secret `runner-auth`, the name the VAP's volume allowlist expects).
- ImageRepository + ImagePolicy `xlearn-runner` (`>=1.0.0 <2.0.0`, digest reflection) and a **second ImageUpdateAutomation** with `update.path: ./runner`, merged after the image exists.
- The `runner` Kustomization in `clusters/vps/sandbox.yaml`: `dependsOn: sandbox-guards`, SOPS decryption, `wait`. **Nothing depends on it.**
- The runner HelmRelease `runner/xlearn-runner.yaml` on chart 0.3.0's sandbox knobs: `strategy: Recreate`, grace 70 s, digest-pinned image.
- The acceptance suite (`make runner-acceptance` from [m3-15](sprint-m3-15.md)) against the in-cluster runner, plus the calibration.

**Out**
- **judge** (MI-13, [m3-07](sprint-m3-07.md)): its HelmRelease, its runner egress, and its copy of the bearer secret (`apps/secrets/xlearn-judge-runner-auth.enc.yaml`).
- **The go-race profile** (`runner-v1.1.0`, [p-01](sprint-p-01.md)).
- **Any change to `sandbox-guards` or the VAP** ([mi-14](sprint-mi-14.md)). If the pod is rejected, fix the values, not the VAP.
- **Per-language exec allowlists and profiles.** They're inside the image ([m3-04](sprint-m3-04.md)); changing them is a runner patch release.
- **Host files** ([mi-09](sprint-mi-09.md)).

## Tasks

### 1 · Pre-flight: gates, host, memory sum, render + VAP dry run [H]

- Confirm every entry gate.
  - `flux get kustomizations sandbox-guards` is Ready.
  - `ssh vps 'bash -s -- --expect-sandbox --cluster --with-runner' < hack/host-verify.sh` is **green**.
    The memory sum with the runner's 3 GiB of limits counted must be ≤ capacity − 0.5 GiB
    (≈ 0.9 GiB inside the rule after MI-11a).
- **A pending spike VAP diff first.** If status.md hands mi-10 a spike diff that arrived after mi-14 merged
  (`SETPCAP`, `procMount`, CONNECT behaviour), land it as a small infra PR **before** the first runner pod.
  mi-14 anticipated this.
- **Render check before anything merges.** Re-run mi-14's `hack/sandbox-vap-test.sh` with its **G1**
  positive control rebuilt from **the real task 4 values**: `helm template charts/project -f <values>`, the
  pod template wrapped as a `Pod` in `xlearn-runner`, then a server-side dry run over `ssh vps`. It must be
  **admitted with no warning**, and B1–B14 must still be denied. A server dry run persists nothing.
  Three points need special attention:
  - The image renders as `ghcr.io/sujaykumarsuman/xlearn-runner@sha256:<digest>`. Chart 0.3.0's
    `image.digest` knob drops the tag ([mi-01](sprint-mi-01.md)); the VAP regex also tolerates `:tag@sha256:`.
  - The capabilities, AppArmor/seccomp at pod **and** container level, `procMount`, the volumes and the
    probe types must all match the VAP rule list ([t3 §8.2](../research/t3-sandbox.md#82-guard-objects-infrastructuresandbox)).
  - The pod must fit the Quota and LimitRange.

### 2 · SOPS rule + runner bearer token [I]

- **`.sops.yaml`:** add a creation rule `path_regex: runner/secrets/.*\.enc\.ya?ml$`,
  `encrypted_regex: ^(data|stringData)$`, with the same age recipient as the existing rules.
- **Generate the token offline**, e.g. `openssl rand -base64 48`.
  - Pipe it straight into the Secret manifest and `sops -e -i`. It is never echoed, never in shell history,
    and never written unencrypted outside a pipe.
  - The file is `runner/secrets/runner-auth.enc.yaml`: a `v1` Secret **`runner-auth`** in `xlearn-runner`,
    `type: Opaque`, written with **`stringData.token`** (not `data`), so the `sops -d --extract
    '["stringData"]["token"]'` paths below work. The key `token` is the one runner-v1.0.0 reads: its
    `RUNNER_TOKEN_FILE` defaults to `/var/run/secrets/runner-auth/token` ([m3-03](sprint-m3-03.md)), which
    task 4 mounts.
  - The name must be exactly `runner-auth`, because the VAP's volume allowlist is {emptyDir, secret `runner-auth`}.
- **One value, two files.** judge's `apps/secrets/xlearn-judge-runner-auth.enc.yaml` carries the **same
  value**. Whichever of mi-10 and [m3-07](sprint-m3-07.md) runs first generates it. The other decrypts
  (`sops -d --extract '["stringData"]["token"]' …`) and re-encrypts into its own path. If m3-07 already ran,
  decrypt its file instead of generating a new one.
- Record in status.md who generated it and when, never the value, plus one rotation line:
  - new value in both files, as a pair of PRs, runner first;
  - judge gets 401 until its PR lands, which it treats as `saturated` and re-queues.

### 3 · Image automation (image before policy) [I]

In `apps/image-automation.yaml`, next to the fleet entries:

| Object | Settings |
|---|---|
| `ImageRepository xlearn-runner` | `image: ghcr.io/sujaykumarsuman/xlearn-runner` (public, no `secretRef`), `interval: 1m` |
| `ImagePolicy xlearn-runner` | `filterTags.pattern: '^\d+\.\d+\.\d+$'`; `policy.semver.range: ">=1.0.0 <2.0.0"`; `digestReflectionPolicy: IfNotPresent` |
| `ImageUpdateAutomation runner` (**2nd IUA**) | same `sourceRef` and `git.checkout`/`push` as `flux-system`; `update.path: ./runner`; `strategy: Setters`; `messageTemplate: "chore(runner): automated image update"` |

- `IfNotPresent` is enough because tags are never re-pushed ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).
  Per the Flux ImagePolicy docs, `Always` would also need `spec.interval`. Check the field against the
  cluster's Flux (v2.9.5) CRD before writing it.
- The first IUA keeps `update.path: ./apps`. **Never suspend it**: that would also freeze airlift,
  landscape, the hub and kubescope.
- **Image before policy:** the image exists (entry gate), so this PR merges first. Then check that
  `flux get images policy xlearn-runner` shows the latest as `1.0.0`, and that the reflected digest equals
  m3-15's recorded digest.
- `deploy.yml`'s `v*` filter never builds the runner (m3-15 tested this). A `runner-v*` tag builds through
  `runner-release.yml` only.

### 4 · `runner` Kustomization + HelmRelease [I]

**Kustomization.** Add it to `clusters/vps/sandbox.yaml`, which [mi-14](sprint-mi-14.md) created with
`sandbox-guards` only ([t3 §8.1](../research/t3-sandbox.md#81-flux-layout-keep-the-runner-off-v1s-critical-path)):

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: runner
  namespace: flux-system
spec:
  interval: 1m
  dependsOn:
    - name: sandbox-guards
  sourceRef:
    kind: GitRepository
    name: flux-system
  path: ./runner
  prune: true
  wait: true
  timeout: 5m
  decryption:
    provider: sops
    secretRef:
      name: sops-age
```

No Kustomization lists `runner` in its `dependsOn`. `apps` never waits on it, so a NotReady runner fails
only this Kustomization.

**HelmRelease.** `runner/xlearn-runner.yaml`: `metadata.name: xlearn-runner` and **`releaseName: xlearn-runner`**
(the fleet's pattern), namespace `xlearn-runner`, chart `./charts/project` (0.3.0). The chart's fullname is
`.Release.Name`, so the Service is `xlearn-runner.xlearn-runner.svc.cluster.local:8090` (judge's
`RUNNER_BASE_URL`, hard-coded by [m3-06](sprint-m3-06.md)/[m3-07](sprint-m3-07.md)), and the pods carry
`app.kubernetes.io/instance: xlearn-runner` (judge's egress selector in m3-07). Knob names are as
[mi-01](sprint-mi-01.md) defined them. Values:

| Value | Setting | Why |
|---|---|---|
| `image.repository` / `image.tag` / `image.digest` | `ghcr.io/sujaykumarsuman/xlearn-runner`; tag `1.0.0` with `# {"$imagepolicy": "flux-system:xlearn-runner:tag"}`; digest `sha256:…` with `# {"$imagepolicy": "flux-system:xlearn-runner:digest"}` | the VAP admits `…@sha256:*` only; the 2nd IUA bumps both |
| `env` | `RUNNER_IMAGE_DIGEST: sha256:…` with the **same** marker `# {"$imagepolicy": "flux-system:xlearn-runner:digest"}` on its value line; `RUNNER_MODE: prod` (explicit, though it's the default). `RUNNER_TOKEN_FILE` stays at its default `/var/run/secrets/runner-auth/token` | [m3-03](sprint-m3-03.md)'s hand-off: the runner reports it as `Versions.ImageDigest` (empty → `unknown`), which judge stores in `job_telemetry.image_digest` ([m3-14](sprint-m3-14.md)). Written with the marker from the first commit, so the 2nd IUA keeps it equal to `image.digest` |
| `runtimeClassName` / `hostUsers` / `priorityClassName` | `xlearn-judge` / `false` / `xlearn-sandbox-lowest` | [ADR-0030 §1](../../adr/0030-runner-technology-and-host-hardening.md#1-build-a-thin-go-runner-xlearn-runner--r1); the lowest priority makes it the first kubelet-eviction victim (L23) |
| `automountServiceAccountToken` / `enableServiceLinks` | `false` / `false` | VAP |
| `dnsPolicy` / `dnsConfig` | `None` / `nameservers: [127.0.0.1]` | no DNS |
| `podSecurityContext` | `runAsUser: 0`, `runAsGroup: 0`, `runAsNonRoot: false`; `seccompProfile: {type: Localhost, localhostProfile: profiles/xlearn-runner.json}`; `appArmorProfile: {type: Localhost, localhostProfile: xlearn-runner}`. Helm deep-merges the chart default, so null out `fsGroup` if the render check or the spike needs it | UID 0 exists only inside the pod's user namespace |
| `containerSecurityContext` | `capabilities: {drop: [ALL], add: [SYS_ADMIN, SETUID, SETGID, SETPCAP, KILL]}` (drop `SETPCAP` if the accepted ADR-0030 / P1 says it's not needed); `procMount: Unmasked`; `readOnlyRootFilesystem: true`; `allowPrivilegeEscalation: false`; seccomp + AppArmor Localhost repeated here | VAP; `Unmasked` is needed for the jail's fresh procfs (t3 §8.3) |
| `strategy.type` / `terminationGracePeriodSeconds` | `Recreate` / `70` | the Quota (limits 2 CPU / 3 GiB) would stall a surge; 70 s drains in-flight jobs |
| `containerPort` / `service.port` / `route.enabled` | `8090` / `8090` / `false` | ClusterIP only; `judge-to-runner` admits TCP 8090 |
| `networkPolicy.enabled` | `false` (**no chart policy**) | policies are additive, so any chart policy would widen `sandbox-guards`' default-deny |
| `probes` | `readinessPath: /readyz` (its canaries: egress fails, AppArmor label, non-identity UID map, writable cgroupfs, userns/`fsopen`/SCTP denied, `core_pattern` not a pipe); **`livenessPath: /healthz`** (the front is up and the IPC pair answers; [m3-03](sprint-m3-03.md)) | the VAP allows `httpGet` probes only |
| `resources` | requests `100m` / `512Mi`; limits `2` / `3Gi` | [t3 §7.5](../research/t3-sandbox.md#75-contention-with-cnpg-and-the-gateway); the memory sum counts the 3 GiB |
| `extraVolumes` / mounts | Secret `runner-auth` mounted **read-only at `/var/run/secrets/runner-auth`** (so the file is `…/token`); any `emptyDir` scratch the image needs (m3-03's hand-off in status.md) | the VAP allows {emptyDir, secret `runner-auth`} only; no `imagePullSecrets` (public image) |
| `replicaCount` | `1` | 2 slots per pod (L14) |

- **Pre-merge:** repeat the task 1 render + dry run with these exact values. Both digests in the values
  (`image.digest` and `RUNNER_IMAGE_DIGEST`) equal the policy's reflected digest, so the 2nd IUA has nothing
  to commit.
- **After merge,** check:
  - `flux get kustomizations runner` is Ready;
  - the pod is Running and Ready (every `/readyz` canary passes);
  - there's no `FailedCreate` or VAP denial in the ReplicaSet events;
  - the Quota shows 1 pod with 2 CPU / 3 GiB limits;
  - through task 5's port-forward, `GET /v1/profiles` (bearer) reports `mode: prod` and `image_digest` equal
    to m3-15's recorded digest, not `unknown`.
- **The real exec/attach proof** (mi-14 X1/X2, handed here because it needs a running runner pod):
  `ssh vps 'k3s kubectl -n xlearn-runner exec deploy/xlearn-runner -- true'` must be **denied by the VAP**
  ("exec/attach into xlearn-runner is denied; delete the pod instead"). Do the same for `attach`. Nothing
  runs, because the request is refused at admission. Record the message.

### 5 · Acceptance suite on prod + TL calibration [H]

- **Reach the runner without exec.** The VAP denies exec and attach, and nothing in-cluster can reach the
  runner yet. Use a throwaway ssh tunnel to a port-forward:
  ```
  ssh -L 18090:127.0.0.1:18090 vps 'k3s kubectl -n xlearn-runner port-forward deploy/xlearn-runner 18090:8090'
  # 1 · the release gate (m3-15's prod run: the prod subset plus section I)
  make runner-acceptance RUNNER_URL=http://127.0.0.1:18090 \
    RUNNER_TOKEN_FILE=<(sops -d --extract '["stringData"]["token"]' ../infra/runner/secrets/runner-auth.enc.yaml) \
    SUBSET=prod REQUIRE_PROD=1
  # 2 · calibration, at a quiet hour (or add CALIBRATE=1 to run 1 if it's already quiet)
  make runner-acceptance RUNNER_URL=http://127.0.0.1:18090 \
    RUNNER_TOKEN_FILE=<(sops -d --extract '["stringData"]["token"]' ../infra/runner/secrets/runner-auth.enc.yaml) \
    SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1
  ```
  These are the flags [m3-15](sprint-m3-15.md) task 4 names for production. Without `REQUIRE_PROD=1`,
  section I never runs; without `CALIBRATE=1`, `baseline@1` and `CanaryMedian` aren't produced. Both are
  A8 gates. Port-forward traffic enters the pod's own network namespace over loopback. Every job still runs
  in an empty per-case netns, so the network probes stay meaningful.
- **Suite ([m3-15](sprint-m3-15.md); [t3 §5.10](../research/t3-sandbox.md#510-cleanup-invariants-release-gate-a8)):**
  - **section I, prod-only canaries** (`REQUIRE_PROD=1`): `/v1/profiles` `mode=prod`; `/readyz` green with
    the AppArmor label `xlearn-runner`, a non-identity UID map, and userns/`fsopen`/SCTP denied. `mode=dev`
    fails the run;
  - network probes: no egress, no DNS;
  - syscall probes: SIGSYS, and no host core helper spawned;
  - the P2 corpus subset: 1 GiB balloon, fork bomb, thread bomb, tmpfs and inode fill, stdout flood, orphan
    double-fork, spin, sleep;
  - INV-14: every OOM lands in a case cgroup, with **0 container OOMs**;
  - cross-account markers: `/w`, `/tmp`, `/dev/shm`, SysV shm, POSIX mq, abstract socket, keyring;
  - cleanup invariants: 0 survivors, `runner/` memory back to baseline ±5 MiB, `nr_dying_descendants`
    → ~0 within 60 s.
- **Calibration** ([t3 §7.3](../research/t3-sandbox.md#73-calibration), A8): `baseline@1` is 5 kernels × 30
  runs in the production runner, keyed to `profile_sha256`. Record `CanaryMedian`.
  - Derive the per-language TL multipliers (Go, C++, Python) against the canary.
  - Cgroup CPU time **includes steal** (S0). Run at a quiet hour, record sar steal for the run window, and
    re-run if steal was > 5%.
  - Use the S0 steal thresholds for the quiet-re-run and `throttled` settings.
- **Stop conditions** while the suite runs: an OOMKill outside `xlearn-runner`, node memory pressure, or any
  `Evicted` pod → stop at once. The kill switch is to delete the runner pod (the [t3 §5.2](../research/t3-sandbox.md#52-process-model-privilege-split)
  ops rule), then a revert PR for the `runner` path if needed.
- **Record** the numbers in status.md. Put `baseline@1`, `CanaryMedian`, the multipliers and the steal
  during the run in **`docs/architecture/runner-tl-baselines.md`**, filling each profile table's production
  column that m3-15 marked "pending mi-10" (an xlearn docs PR). If they're compiled into runner profiles
  instead, open an xlearn PR that rides the next `runner-v*` tag; it isn't tagged here. They feed m3-07's TL
  gate and m3-13's re-gate.

### 6 · Verify [H]

- `host-verify --cluster --expect-sandbox`. The runner's limits are now counted live. `--with-runner` is a
  no-op once a pod exists in `xlearn-runner` ([mi-02](sprint-mi-02.md)), so drop it from runbooks from here
  on. Expect:
  - the memory sum ≤ capacity − 0.5 GiB;
  - the `xlearn-runner` policies present (`hack/expected-netpol.tsv` unchanged: `default-deny-all`, `judge-to-runner`);
  - no OOMKills;
  - Flux `sandbox-guards` and `runner` Ready.
- **Reachability** is judged at the policy level. `k3s kubectl get networkpolicy -n xlearn-runner` shows
  exactly the two mi-14 policies. An optional read-only deep check: kube-router's rules for the runner pod
  (`iptables-save` over ssh) admit only the judge selector's ipset, which stays empty until judge exists.
  Node-local traffic bypasses NetworkPolicy by design (t3 §8.4), so a probe from the node proves nothing.
- The pod has no DNS (`dnsPolicy: None`), no SA token and no service links; `kubectl describe` shows
  RuntimeClass `xlearn-judge`.

### 7 · Record [X]

In `docs/v2/status.md`:
- the MI track MI-12 row ✅, with the PRs and date;
- the **runner stream** row: `runner-v1.0.0` live dark, with its digest;
- the calibration numbers and steal during the run;
- who generated the bearer, and when;
- the Sprint board row;
- the M3 hard-checklist key: MI-12 done;
- Decisions log lines: `IfNotPresent` digest reflection, the liveness path (`/healthz`), `SETPCAP` kept or
  dropped, and where the multipliers live.

## Acceptance criteria

- [ ] Runner Ready on prod and admitted by the VAP. It is reachable only from judge's selector: no policy in `xlearn-runner` besides `default-deny-all` + `judge-to-runner`, and no chart policy
- [ ] A real `exec`/`attach` against the running runner pod is denied by the VAP (mi-14's X1/X2, proven live)
- [ ] Acceptance suite passes dark with `SUBSET=prod REQUIRE_PROD=1`: section I's prod-only canaries (`mode=prod`, AppArmor label, non-identity UID map, userns/`fsopen`/SCTP denied), network, syscall, the P2 subset (0 container OOMs), cross-account markers, cleanup invariants
- [ ] `GET /v1/profiles` reports `image_digest` equal to m3-15's recorded digest (`RUNNER_IMAGE_DIGEST` set with the `:digest` `$imagepolicy` marker)
- [ ] The memory-sum rule holds with the runner counted (`host-verify --cluster`)
- [ ] `baseline@1`, `CanaryMedian` and the Go/C++/Python TL multipliers (the `CALIBRATE=1` run) are recorded, with steal during the run, in `docs/architecture/runner-tl-baselines.md`'s production column
- [ ] `flux get images policy xlearn-runner` shows `1.0.0` with its digest; the 2nd IUA is Ready; the first IUA is untouched
- [ ] `runner` is off `apps`' wait path: no Kustomization depends on it

## Release

**Infra PR(s) only.** `runner-v1.0.0` was cut in [m3-15](sprint-m3-15.md); this sprint deploys it. Order:
1. the `.sops.yaml` rule + `runner-auth` secret, and the image automation (ImageRepository/Policy + 2nd IUA).
   The image exists, so image-before-policy holds;
2. the `runner` Kustomization + HelmRelease;
3. the xlearn docs PR (status, calibration). Merge only, no tag.

A later runner patch or minor (`runner-v1.x.y`) deploys through the 2nd IUA with no infra PR. A runner
**major** is a judge↔runner contract break: GA in miniature ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)).

## Definition of Done

Infra PRs merged in order · runner Ready and admitted · acceptance suite green on prod · calibration
recorded · `host-verify --cluster` green with the runner counted · statuses updated (this file +
[`../status.md`](../status.md)) · no `kubectl apply` or exec · no alert, timer or CronJob (D34).

## Risks / watch-outs

- **The VAP rejects the pod over a knob mismatch.** Fix the values, not the VAP. The task 1 render + dry run
  catches it before merge.
- **A surge rollout would stall on the Quota**, hence `Recreate`. During an image bump, judge sees
  `saturated` for about 70 s + start-up, and re-queues.
- **Policies are additive.** Leaving `networkPolicy.enabled` on (a Traefik ingress rule) would widen the
  sandbox's default-deny.
- **The missing `.sops.yaml` rule.** Without it, `sops -e` doesn't know the recipient. Flux would still
  decrypt a file encrypted by hand, so the rule is the reviewable path.
- **Bearer drift between runner and judge** means 401, which judge treats as `saturated`. It's harmless
  while dark, and m3-07 verifies it.
- **The suite runs hostile code on the production node.** It's bounded by the pod caps (2 vCPU / 3 GiB,
  pids), the host block and L23 eviction. Run at a quiet hour and honour the stop conditions.
- **Steal inflates CPU-time calibration.** Re-run on a noisy hour. TLs only scale up (t3 §7.3).
- **A failing `/readyz` canary after host drift** (e.g., a k3s upgrade that dropped the drop-in import)
  keeps the runner NotReady. Run `host-verify --expect-sandbox` first; a NotReady `runner` Kustomization
  also shows in host-verify's Flux check. That's the signal, not an alert.
- **Never exec into the runner** (the VAP denies it anyway). Delete the pod instead.
- **The two IUAs both push to infra `main`.** A push race retries on the next interval. Rebase your own PRs
  before merging.
