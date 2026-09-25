# Prompt — Sprint mi-10 · Runner dark on production (MI-12)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-10.md`](../sprints/sprint-mi-10.md)   ·   **Milestone:** MI (rollout step MI-12)   ·   **Prereqs:** [mi-14](../sprints/sprint-mi-14.md), [mi-08](../sprints/sprint-mi-08.md), [mi-09](../sprints/sprint-mi-09.md) + the host window, [m3-15](../sprints/sprint-m3-15.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions and the land-and-sync rule.
- [`../sprints/sprint-mi-10.md`](../sprints/sprint-mi-10.md): the plan, including the image-automation table (task 3), the full values table (task 4) and the suite list (task 5).
- [`../rollout-plan.md`](../rollout-plan.md):
  - §2, the MI-12 row (Recreate, grace 70 s, off `apps`' wait path, 2nd IUA, image before policy);
  - §2.2, the operating rules;
  - §5, the M3 hard checklist this sprint feeds.
- [ADR-0030](../../adr/0030-runner-technology-and-host-hardening.md): §1, the pod and process model; §5, A7/A8 (now Accepted by m3-03).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md):
  - §1.4, range changes (a new policy goes in after the image exists);
  - §1.5, the runner stream;
  - §4.4, the MI-11/MI-12 rollback row.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md): §4, L14 (the runner's hard caps); §5, the memory-sum rule and MI-11a before MI-12.
- [t3](../research/t3-sandbox.md):
  - §5.1–5.3, placement, the privilege split, and the API/`/readyz` canaries;
  - §5.10, the cleanup invariants;
  - §7.1–7.5, the cgroup layout, TL policy, calibration and contention;
  - §8.1, the Flux layout;
  - §8.2, the VAP rule list;
  - §8.3, the runner values.
- The neighbour plans:
  - [mi-01](../sprints/sprint-mi-01.md), the chart 0.3.0 knob names (`image.digest` renders `repo@digest`; `probes.readinessPath`/`livenessPath`; …);
  - [mi-14](../sprints/sprint-mi-14.md), the VAP and guard objects;
  - [mi-09](../sprints/sprint-mi-09.md), the host block and the caps list;
  - [m3-03](../sprints/sprint-m3-03.md), [m3-04](../sprints/sprint-m3-04.md), [m3-15](../sprints/sprint-m3-15.md): the runner code, profiles, image, acceptance suite and TL baselines. In particular: m3-03 task 5's config and task 8's hand-off to mi-10 (`/healthz`, `RUNNER_TOKEN_FILE`, `RUNNER_IMAGE_DIGEST`, `RUNNER_MODE`); m3-15 task 4 (the `SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1` flags, section I) and task 7 (`docs/architecture/runner-tl-baselines.md`);
  - [m3-07](../sprints/sprint-m3-07.md), judge's side of the bearer secret.
- `../infra`, all read before editing:
  - `.sops.yaml`;
  - `clusters/vps/sandbox.yaml`;
  - `infrastructure/sandbox/*` (VAP, Quota, policies);
  - `apps/image-automation.yaml` (the existing IUA, markers);
  - `charts/project/{values.yaml,templates/deployment.yaml}` (0.3.0);
  - `hack/host-verify.sh`, `hack/expected-netpol.tsv`;
  - README "Secrets".

## Context

- The runner (`xlearn-runner`) is xLearn v2's code-execution sandbox. It's a thin Go supervisor running
  untrusted learner code in userns-less jails, inside a user-namespaced pod on the production node.
- Its guard objects are live (MI-4), and so are its host prerequisites (MI-11, the 2026-10-24 window).
  MI-11a freed the memory it needs.
- `runner-v1.0.0` exists on its own release stream. This sprint deploys it **dark**:
  - nothing in the cluster calls it until judge ships ([m3-07](../sprints/sprint-m3-07.md));
  - only judge's selector can ever reach it;
  - it sits off `apps`' wait path.

  Then it runs the **acceptance suite on production** (A8) and calibrates time limits there.
- The runner's pod shape is pinned by a ValidatingAdmissionPolicy. Every value must match it, and **you fix
  values, never the VAP**.
- Each step below is tagged with the plan task it ticks in the plan's Status table (steps 5 and 6 both tick
  task 5).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] MI-4 live: `flux get kustomizations sandbox-guards` is Ready; `k3s kubectl get pods -n xlearn-runner` is empty; the VAP bindings are at `[Deny]` (mi-14's PR 2); the RuntimeClass `xlearn-judge`, PriorityClass `xlearn-sandbox-lowest`, Quota, LimitRange, `default-deny-all` and `judge-to-runner` exist. Check status.md for a spike VAP diff handed to mi-10.
- [ ] The MI-11 window has been executed: `ssh vps 'bash -s -- --expect-sandbox --cluster' < ../infra/hack/host-verify.sh` is green (`judge` runtime, AppArmor enforce, seccomp, subuid, L23 in `configz`).
- [ ] MI-11a merged (mi-08), and `host-verify --cluster --with-runner` shows the memory sum inside the rule.
- [ ] `ghcr.io/sujaykumarsuman/xlearn-runner:1.0.0` exists (an anonymous `crane digest` works), and its digest matches m3-15's record. If m3-15 left a ⛔ owner item for the package visibility in status.md and the anonymous read still fails, this gate is unmet.
- [ ] ADR-0030 is Accepted (m3-03), and status.md records the caps list (mi-09).
- [ ] Parallel sessions: no open peer PR touches `clusters/vps/sandbox.yaml`, `runner/`, `apps/image-automation.yaml` or `.sops.yaml` (`gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents). Also check whether m3-07 already generated the runner bearer.

## Do this (in order)

1. **[H] Render + VAP dry run** (plan task 1).
   - **Pending spike VAP diff first.** If status.md hands mi-10 one (it arrived after mi-14 merged), land it
     as a small infra PR before any runner pod exists.
   - Write the task 4 values.
   - Re-run mi-14's `hack/sandbox-vap-test.sh` with its **G1** positive control rebuilt from those values:
     `helm template ../infra/charts/project -f <values>`, the Deployment's pod template wrapped as a `Pod`
     in `xlearn-runner`, then a server-side dry run over `ssh vps`. G1 must be **admitted with no warning**,
     and B1–B14 must still be denied. A server-side dry run persists nothing.
   - The image renders as `ghcr.io/sujaykumarsuman/xlearn-runner@sha256:<digest>`, and the pod fits the
     Quota and LimitRange.
   - If it's denied, fix the **values** and repeat.

2. **[I] SOPS rule + bearer** (plan task 2). In `.sops.yaml`, add `path_regex: runner/secrets/.*\.enc\.ya?ml$` (same
   recipient, `encrypted_regex: ^(data|stringData)$`). Then create `runner/secrets/runner-auth.enc.yaml`:
   a `v1` Secret **`runner-auth`** in ns `xlearn-runner`, `type: Opaque`, written as **`stringData.token`**
   (not `data`; the `sops -d --extract '["stringData"]["token"]'` paths depend on it). `token` is the key
   runner-v1.0.0 reads: its `RUNNER_TOKEN_FILE` defaults to `/var/run/secrets/runner-auth/token` (m3-03).
   - **If m3-07 already created** `apps/secrets/xlearn-judge-runner-auth.enc.yaml`, decrypt that value with
     `sops -d --extract` and re-encrypt it here.
   - **Otherwise** generate it offline (`openssl rand -base64 48`), piped, never echoed.

3. **[I] Image automation PR** (plan task 3). In `apps/image-automation.yaml`, add:
   - `ImageRepository xlearn-runner` (public, 1m);
   - `ImagePolicy xlearn-runner` (`filterTags '^\d+\.\d+\.\d+$'`, `semver ">=1.0.0 <2.0.0"`,
     `digestReflectionPolicy: IfNotPresent`);
   - `ImageUpdateAutomation runner` (`update.path: ./runner`, `Setters`, same git settings,
     `messageTemplate: "chore(runner): automated image update"`).

   Merge it (with step 2's `.sops.yaml` change and secret, or on its own). Check that
   `flux get images policy xlearn-runner` shows `1.0.0` and the reflected digest equals m3-15's.

4. **[I] runner PR** (plan task 4).
   - Add the `runner` Kustomization to `clusters/vps/sandbox.yaml`: `./runner`, `dependsOn: sandbox-guards`,
     SOPS `sops-age`, `prune`, `wait`, `timeout: 5m`. No `dependsOn` elsewhere points at it.
   - Write `runner/xlearn-runner.yaml` with the plan's values table. In short:
     - `metadata.name` and **`releaseName: xlearn-runner`**, so the Service is
       `xlearn-runner.xlearn-runner.svc.cluster.local:8090` (judge's hard-coded `RUNNER_BASE_URL`, m3-06/m3-07)
       and the pods carry `app.kubernetes.io/instance: xlearn-runner` (judge's egress selector);
     - image repository + tag + digest, with the `:tag`/`:digest` `$imagepolicy` markers;
     - `env`: **`RUNNER_IMAGE_DIGEST: sha256:…`** with the same `# {"$imagepolicy": "flux-system:xlearn-runner:digest"}`
       marker as `image.digest`, from the first commit (m3-03's hand-off; the runner reports it as
       `Versions.ImageDigest`, judge stores it in `job_telemetry.image_digest`), and `RUNNER_MODE: prod`;
     - RuntimeClass `xlearn-judge`, `hostUsers: false`, PriorityClass `xlearn-sandbox-lowest`;
     - token automount off, service links off;
     - `dnsPolicy: None` with nameserver `127.0.0.1`;
     - pod `runAsUser: 0`, Localhost seccomp `profiles/xlearn-runner.json` and AppArmor `xlearn-runner`
       at pod **and** container level;
     - caps drop ALL, add the recorded list;
     - `procMount: Unmasked`, read-only rootfs, no privilege escalation;
     - `Recreate`, grace 70 s;
     - port 8090 ClusterIP, no route, `networkPolicy.enabled: false`;
     - readiness `/readyz`, liveness **`/healthz`**;
     - requests 100m/512Mi, limits 2/3Gi;
     - the Secret `runner-auth` volume, mounted read-only at `/var/run/secrets/runner-auth`;
     - `replicaCount: 1`.
   - Re-run step 1 with the final values, then merge.
   - Watch that the `runner` Kustomization goes Ready and the pod goes Ready (the `/readyz` canaries), with
     no VAP denial in the events.
   - Through step 5's tunnel, `GET /v1/profiles` must report `mode: prod` and `image_digest` equal to m3-15's
     recorded digest (not `unknown`).
   - **Real exec/attach proof** (mi-14's X1/X2 needs a running pod):
     `ssh vps 'k3s kubectl -n xlearn-runner exec deploy/xlearn-runner -- true'` must be **denied by the
     VAP**; do the same for `attach`. Record the message. Nothing runs, because the request is refused at
     admission.

5. **[H] Acceptance suite** (plan task 5). Tunnel in:
   `ssh -L 18090:127.0.0.1:18090 vps 'k3s kubectl -n xlearn-runner port-forward deploy/xlearn-runner 18090:8090'`.
   Then run the prod gate with m3-15's production flags:
   ```
   make runner-acceptance RUNNER_URL=http://127.0.0.1:18090 \
     RUNNER_TOKEN_FILE=<(sops -d --extract '["stringData"]["token"]' ../infra/runner/secrets/runner-auth.enc.yaml) \
     SUBSET=prod REQUIRE_PROD=1
   ```
   - It covers: **section I, the prod-only canaries** (`/v1/profiles` `mode=prod`; `/readyz` green with the
     AppArmor label `xlearn-runner`, a non-identity UID map, userns/`fsopen`/SCTP denied; `mode=dev` fails the
     run); no egress/DNS; syscall probes; the P2 subset; INV-14 with 0 container OOMs; cross-account
     markers; cleanup invariants. Without `REQUIRE_PROD=1`, section I never runs.
   - **Stop at once** on an OOMKill outside `xlearn-runner`, node memory pressure or an `Evicted` pod. Then
     delete the runner pod, and open a revert PR if needed.

6. **[H] Calibration** (plan task 5). At a quiet hour, run the same command with **`CALIBRATE=1`** added
   (`SUBSET=prod REQUIRE_PROD=1 CALIBRATE=1`; or add it to step 5's run if that hour was already quiet).
   Without it, `baseline@1` and `CanaryMedian` aren't produced. It measures `baseline@1` (5 kernels × 30
   runs), `CanaryMedian`, and the Go, C++ and Python TL multipliers against the canary.
   - Record sar steal for the window. Re-run if steal > 5%.
   - Write the numbers into **`docs/architecture/runner-tl-baselines.md`**, filling each profile table's
     production column that m3-15 marked "pending mi-10" (xlearn docs PR). If they live in runner profiles
     instead, open an xlearn PR for the next `runner-v*` tag, and don't tag it here.

7. **[H] Verify** (plan task 6). Run `host-verify --cluster --expect-sandbox`. `--with-runner` is a no-op once a pod
   exists in `xlearn-runner` (mi-02), so drop it from here on. Expect:
   - the memory sum ≤ capacity − 0.5 GiB;
   - no OOMKills;
   - `sandbox-guards` and `runner` Ready;
   - `expected-netpol.tsv` unchanged, with exactly the two mi-14 policies in `xlearn-runner`.

   Also confirm `describe pod` shows RuntimeClass `xlearn-judge`, no SA token and no DNS.

8. **[X] Record** (plan task 7). Update status.md (see Update status), then open the xlearn docs PR and merge it.

## Constraints

- **GitOps:** never `kubectl apply` (a `--dry-run=server` check persists nothing and is the sanctioned VAP
  proof). **Never exec** into the runner; delete the pod instead. Never suspend the shared IUA.
- **Image before policy** ([ADR-0034 §1.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#14-range-changes-and-the-ga-procedure)):
  the runner policy merges only because `runner-v1.0.0` already exists. **Never move or re-push a tag.**
- **The VAP is the contract.** Fix values, not the VAP. **Don't add a chart NetworkPolicy**: policies are
  additive and would widen the sandbox.
- **Keep the runner off the critical path.** No Kustomization depends on `runner`, and `apps` never waits on it.
- **The memory-sum rule** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):
  a new always-on pod is checked against the sum, not Σ limits alone. The runner carries a 3 GiB memory limit.
- **Secrets:** the bearer is generated offline and never echoed or committed in plaintext. The same value
  goes into judge's file (m3-07).
- **D34, no alerting:** no Flux Alert, CronJob or push channel. `host-verify` stays on demand.
- **Not applicable, since there's no xlearn service code here:** goose + sqlc (`sqlc diff`), outbox/inbox,
  service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), `theme.css`,
  consumers-before-producers, and the ACL-PR-before-consuming-tag rule (the runner has no NATS). The only
  xlearn changes are docs, plus the multipliers PR if they live in code.
- **Parallel sessions:** re-check peers' infra PRs right before each merge, and rebase first. The Flux bots
  commit to infra `main` often.
- Conventional commits with the required attribution lines, and squash merges. Don't enable auto-merge.

## Deliverables

- infra PR: the `.sops.yaml` rule and `runner/secrets/runner-auth.enc.yaml`; ImageRepository + ImagePolicy
  `xlearn-runner` and ImageUpdateAutomation `runner` in `apps/image-automation.yaml`.
- infra PR: the `runner` Kustomization in `clusters/vps/sandbox.yaml`, and `runner/xlearn-runner.yaml`.
- The acceptance-suite and calibration evidence, pasted into the runner PR and recorded in status.md.
- xlearn docs PR: status rows, `baseline@1`/multipliers in `docs/architecture/runner-tl-baselines.md` (m3-15's TL-baseline doc), and this sprint's Status
  table (plus an xlearn PR for the multipliers only if they live in code).

## Update status

- Set each task row in [`../sprints/sprint-mi-10.md`](../sprints/sprint-mi-10.md) to 🔄 or ✅ as you go (each step above names its plan task), and _Overall_ to ✅ when all seven plan tasks are done.
- Mirror it in [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **MI track** MI-12 row ✅ (PRs, date);
  - the **runner stream** row: `runner-v1.0.0` live dark, with its digest;
  - the calibration numbers;
  - who generated the bearer (and when);
  - the M3 hard-checklist key: MI-12 done.
- Add **Decisions log** lines for `IfNotPresent`, the liveness path (`/healthz`), the caps (SETPCAP kept or dropped), and where the multipliers live.
- No ADR is expected. If a VAP-vs-chart conflict can't be fixed in the values (the only pre-decided path), don't change the VAP or ADR-0030 in this sprint. Land the PRs that don't depend on it (steps 2–3, the docs PR), push the runner change as a branch without a PR, and record the conflict, the options and a recommendation in status.md, with MI-12 marked ⛔ "needs owner decision (ADR-0030 amendment)". Don't wait.

## Done when (acceptance)

- [ ] Runner Ready on prod and admitted by the VAP. It is reachable only from judge's selector: `xlearn-runner` holds only `default-deny-all` + `judge-to-runner`, and no chart policy
- [ ] A real `exec`/`attach` against the running runner pod is denied by the VAP
- [ ] Acceptance suite passes dark with `SUBSET=prod REQUIRE_PROD=1`: section I's prod-only canaries, network, syscall, P2 subset with 0 container OOMs, cross-account markers, cleanup invariants
- [ ] `GET /v1/profiles` reports `image_digest` equal to m3-15's recorded digest (`RUNNER_IMAGE_DIGEST` carries the `:digest` marker)
- [ ] The memory-sum rule holds with the runner counted (`host-verify --cluster`)
- [ ] `baseline@1`, `CanaryMedian` and the Go/C++/Python TL multipliers (the `CALIBRATE=1` run) are recorded, with steal during the run, in `docs/architecture/runner-tl-baselines.md`'s production column
- [ ] `flux get images policy xlearn-runner` shows `1.0.0` with its digest; the 2nd IUA is Ready; the first IUA is untouched
- [ ] `runner` is off `apps`' wait path

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: the `../infra` PRs of steps 1–4 first, then the xlearn docs PR (plus the multipliers PR only if they live in code).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. `../infra` has no CI: put the local check output (render + VAP dry run, suite results) in each infra PR body and merge on it.
3. **Release action — infra PR(s) only (`runner-v1.0.0` was already cut in m3-15):** merge the infra PRs in the plan's order, each its own PR and never folded into a tag: a pending spike VAP diff first, if status.md handed one over (before any runner pod exists); the `.sops.yaml` rule + `runner-auth` secret and the image automation (image before policy); then the `runner` Kustomization + HelmRelease. Verify live with step 7's `host-verify --cluster --expect-sandbox`. Then merge the xlearn docs PR (status, calibration). A multipliers PR, if they live in runner profiles, merges only and rides the next `runner-v*` tag. No tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
