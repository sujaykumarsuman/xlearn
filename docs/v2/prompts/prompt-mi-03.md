# Prompt — Sprint mi-03 · Fences: databases/messaging ingress, xlearn ingress (MI-5, MI-5a)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the xlearn repo root. The cluster work happens in the sibling `../infra` repo, through GitOps PRs.
> **Plan:** [`../sprints/sprint-mi-03.md`](../sprints/sprint-mi-03.md)   ·   **Milestone:** MI (rollout steps MI-5 and MI-5a, ADR-0030 A4)   ·   **Prereqs:** [mi-01](../sprints/sprint-mi-01.md) (MI-5a only), [mi-02](../sprints/sprint-mi-02.md)

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Read the date of the last Hostinger weekly image in hPanel and record it in `status.md` (or give it in the launch message). Rollout §2.2 requires it before any restart-inducing step, and this sprint's two `podAnnotations` bumps each restart a pod.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): the land-and-sync directive and the `../infra` conventions (never `kubectl apply` by hand).
- [`../sprints/sprint-mi-03.md`](../sprints/sprint-mi-03.md): the plan. It holds the **caller matrix**, the exact MI-5 manifests, the MI-5a rendered policies and the smoke steps.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - **§2**, the "MI-5 ingress policies" table and the standing rule;
  - the Context row "NetworkPolicy: none in `xlearn`, `databases` or `messaging`";
  - **§3**, `host-verify` NetworkPolicy presence;
  - **§5**, the memory margin before MI-11a.
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 7 (internal HTTP: "the NetworkPolicy is the fence"), §14 (the amendments to ADR-0006 and ADR-0016), and its Context row "Internal HTTP".
- [ADR-0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (review workers call identity over ClusterIP with no JWT) and [ADR-0006](../../adr/0006-authn-authz.md) (JWKS verification).
- [`../research/t3-sandbox.md`](../research/t3-sandbox.md) §8.4 (the v1 NetworkPolicies, and the known limit that node-local traffic is always admitted) and §8.8 row A4.
- [`../rollout-plan.md`](../rollout-plan.md) §2 (the MI-5 and MI-5a rows), §2.2 (the operating rules), §5 (the M3 checklist names MI-5 and MI-5a).
- **Code and config:**
  - `internal/platform/auth/jwks.go`: lazy fetch, 1 h cache, stale tolerated on a failed refresh;
  - `internal/identity/service.go:70-84`: the unauthenticated `/sessions/*` and `/internal/*`;
  - `internal/gateway/public.go`: `GET /api/u/{username}` fans out to identity, assessment (with a minted JWT) and curriculum;
  - `../infra/apps/xlearn-*.yaml` (env);
  - `../infra/charts/project/{values.yaml,templates/networkpolicy.yaml,templates/deployment.yaml,templates/_helpers.tpl}` at 0.3.0;
  - `../infra/infrastructure/{database/cluster,messaging}/`, `../infra/clusters/vps/{databases,messaging,apps}.yaml`;
  - `../infra/hack/{host-verify.sh,expected-netpol.tsv}`.

## Context

`xlearn`, `databases` and `messaging` have **no NetworkPolicy today**. Several things rely on an isolation that doesn't exist:
- PG admits anything in the cluster, the internet-facing gateway included;
- NATS is an open bus until N3;
- identity's `/sessions/*` and `/internal/*`, and review's workers' calls, rely on isolation that isn't there.

This sprint lands the two fences v2 depends on:
- **MI-5** is PG, CNPG and NATS ingress with forward-declared callers. It's the **N3 precondition** for [mi-06](../sprints/sprint-mi-06.md) and a gate for M3 and L-E.
- **MI-5a** is the xlearn ingress fence that makes internal HTTP safe, for M3 and L.

The trap: every JWT-verifying service (identity, practice, review, assessment, coach, and judge later) fetches **JWKS from `xlearn-gateway:8080` in-cluster**. So "gateway admits only Traefik" would break them all on their next restart. Running pods hide the breakage for up to 1 h, because stale keys are tolerated.

Live facts (read-only, 2026-09-24):
- **xlearn pods carry only `app.kubernetes.io/name=project` and `app.kubernetes.io/instance=<release>`.** `part-of` is on the Deployment, not the pod, so never select on it.
- **Traefik** is `app.kubernetes.io/name=traefik` in `kube-system`.
- **NATS** is `nats-0`, `app.kubernetes.io/{name,instance}=nats`. Port 8222 is only on the `nats-headless` Service.
- **PG** is `projects-pgstore-1`, `cnpg.io/cluster=projects-pgstore`. The CNPG operator runs in `cnpg-system`.
- **`/connz`** shows 7 connections, from practice, review and assessment.

The infra repo has **no CI**. Validate with `helm template` diffs, `hack/host-lint.sh`, and `--dry-run=server` on the node, which persists nothing. Don't use `kubectl kustomize`: infra directories have no `kustomization.yaml` (Flux generates one; don't add one), and `infrastructure/database/cluster/` holds SOPS-encrypted files.

**Owner time:** none mid-run. The weekly image date is a before-launch item. The login smokes use an already-signed-in browser session if you have one; otherwise record "owner login smoke pending" as a pending-smoke note in status.md and carry on (an agent never enters credentials).

## Entry gates: verify first

Stop and report if any gate is unmet.

- [ ] The MI-8 extension is merged ([mi-02](../sprints/sprint-mi-02.md)): `../infra/hack/expected-netpol.tsv` exists, and `host-verify.sh --cluster` checks NetworkPolicy presence and supports `--nats-stage=open`.
- [ ] *(For MI-5a only)* Chart 0.3.0 is merged ([mi-01](../sprints/sprint-mi-01.md)), with multi-source ingress and a same-namespace source in `charts/project/values.yaml`. If it isn't, **do MI-5 alone**, record mi-03 as 🔄 (MI-5a waiting on mi-01), and stop.
- [ ] `cd ../infra && git checkout main && git pull`. Peer check in both repos: `gh pr list --state open`, `git worktree list`, ListAgents. No open PR touches `apps/xlearn-*.yaml`, `infrastructure/{database/cluster,messaging}/` or `hack/expected-netpol.tsv` (mi-14 edits the `.tsv` too).
- [ ] The last Hostinger weekly image date is recorded in status.md (the owner reads hPanel before launch; see above). [Rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) requires it before any restart-inducing step, and this sprint's two `podAnnotations` bumps each restart a pod. **If it's missing, don't wait:** land MI-5 (step 2 restarts nothing) and the docs PR, and set the restart-inducing work (step 3's restart PR, steps 4–5) ⛔ in status.md, naming the owner item; a re-run picks up at step 3.
- [ ] No xlearn release is mid-rollout (`ssh sujaykumar-vps 'sudo k3s kubectl get deploy -n xlearn'`: all available). No `v*` tag was pushed in the last 15 min (`git ls-remote --tags origin`).

## Do this (in order)

1. **Caller matrix [H]** (plan task 1):
   - Re-derive the matrix from `../infra/apps/xlearn-*.yaml` env.
   - Confirm it against live state, read-only:
     - NATS: `ssh sujaykumar-vps 'sudo k3s kubectl get --raw /api/v1/namespaces/messaging/services/nats-headless:8222/proxy/connz'`, with client IPs mapped via `get pods -n xlearn -o wide`;
     - PG: `ssh sujaykumar-vps 'sudo k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select usename, client_addr, count(*) from pg_stat_activity where client_addr is not null group by 1,2"'`, with IPs mapped to pods.
   - **Stop** on any caller that isn't in the plan's matrix.
   - Re-read the Traefik, NATS, PG and xlearn pod labels (`--show-labels`).

2. **MI-5 PR [I]** (plan task 2):
   - On branch `feat/mi-5-db-msg-ingress` in `../infra`, add:
     - `infrastructure/database/cluster/networkpolicy.yaml` (`projects-pgstore-ingress`: PG 5432 ← the 6 DB services + `xlearn-judge`, **not the gateway**; :8000 ← `cnpg-system`; instance↔instance 5432/8000);
     - `infrastructure/messaging/networkpolicy.yaml` (`nats-ingress`: 4222 ← practice, review, assessment + identity, judge, coach; **no 8222 rule**).
   - Use the manifests from the plan verbatim, with header comments: the matrix, why callers are forward-declared, and the standing rule.
   - Add two rows to `hack/expected-netpol.tsv` in mi-02's three-column format (`namespace <TAB> name <TAB> added-by`): `databases<TAB>projects-pgstore-ingress<TAB>mi-03` and `messaging<TAB>nats-ingress<TAB>mi-03`. Paste the same rows, byte-identical, into the **embedded copy** in `hack/host-verify.sh` between its `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv` markers. host-verify runs via `bash -s` and never reads the file.
   - **Validate:**
     - `hack/host-lint.sh` clean (the embedded copy equals the file);
     - server dry-run **the two new files only**: `ssh sujaykumar-vps 'sudo k3s kubectl apply --dry-run=server -f -' < infrastructure/database/cluster/networkpolicy.yaml`, and the same for `infrastructure/messaging/networkpolicy.yaml`;
     - the **selector proof**: run each `from` selector as a `-l` query against live pods, and each target selector must match exactly `projects-pgstore-1` and `nats-0`.
   - Open the PR with the matrix and the proofs, then merge it. **Record the merge date** (mi-06's N3 gate).
   - Wait until `get networkpolicy -n databases,messaging` shows both policies.

3. **MI-5 restart PR and smoke [I + H]** (plan task 3):
   - Merge a separate one-line `../infra` PR that bumps `podAnnotations` on `apps/xlearn-review.yaml` (`sujaykumar.dev/restarted-for: mi-5`).
   - When the new review pod is Ready, check (read-only):
     - `/connz`: 7 connections, review's new;
     - `/jsz?consumers=true`: `num_pending` and `num_ack_pending` both 0;
     - outbox unsent is 0 in practice, review and assessment;
     - CNPG Cluster healthy, and the operator logs are clean for 2 min;
     - `ssh sujaykumar-vps 'bash -s -- --cluster --nats-stage=open' < ../infra/hack/host-verify.sh` still reads NATS, which proves the node-local proxy path;
     - `curl -sf https://projects.sujaykumar.dev/xlearn/api/healthz` and `GET /xlearn/api/u/<owner-username>` return 200 with data. Find the username once, read-only: `ssh sujaykumar-vps 'sudo k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select username from identity.account where username is not null order by created_at limit 3"'` (prod had one account on 2026-09-24, the owner's; filter on `role = 'owner'` once m1-02's column is live). Record it in status.md;
     - the login smoke (login, dashboard, coach). Login needs credentials, which you never enter: use an already-signed-in browser session if you have one; otherwise record "owner login smoke pending (MI-5)" as a pending-smoke note in status.md and carry on (D40).
   - **Any error: `git revert` the MI-5 PR** (fail-open) and stop.

4. **MI-5a PR [I]** (plan task 4):
   - On branch `feat/mi-5a-xlearn-ingress`, set the chart 0.3.0 NetworkPolicy values in all 7 `apps/xlearn-*.yaml` (mi-01's knobs: `enabled: true`, `from: null`, and `extraIngress` raw rules), so that:
     - `xlearn-gateway` admits Traefik (`kube-system` + `app.kubernetes.io/name: traefik`) **plus same-namespace pods** on 8080: one `extraIngress` rule with both peers;
     - identity, curriculum, practice, review, assessment and coach each admit **same-namespace only** on their `containerPort`: one `extraIngress` rule with a single `podSelector: {}` peer;
     - there's **no Traefik source** on the internal services (`from: null`) and **no `part-of` selector** anywhere.
   - Add 7 rows to `hack/expected-netpol.tsv`, `xlearn<TAB>xlearn-<svc><TAB>mi-03` (each policy is named after its release), and the same rows to the embedded copy in `hack/host-verify.sh`. Run `hack/host-lint.sh` clean.
   - Bump `podAnnotations` on `apps/xlearn-assessment.yaml` (`restarted-for: mi-5a`).
   - **Validate:**
     - `helm template` each release at the base values and the new values, then diff. The only changes allowed are one NetworkPolicy per release and the assessment annotation;
     - the rendered policies match the plan: the gateway admits exactly a Traefik peer and a same-namespace peer on :8080, each internal service exactly a same-namespace peer on its port, with no other peers or ports. If the merged chart renders the gateway's two peers as two rules, that's semantically equal: accept and record it, but never accept an extra peer;
     - `--dry-run=server` them;
     - `get pods -n xlearn -l app.kubernetes.io/name=project` lists all 7 pods.
   - Open the PR with the matrix and the diffs, then merge it.

5. **MI-5a smoke: the JWKS path [H]** (plan task 5):
   - When the fresh `xlearn-assessment` pod is Ready, request `GET https://projects.sujaykumar.dev/xlearn/api/u/<owner-username>` after the 15 s cache. The assessment-backed sections must be present, and `k3s kubectl logs -n xlearn deploy/xlearn-assessment` must show **no** JWKS or 401 errors.
   - After a sweep tick, the review logs show no identity or curriculum timeouts.
   - The login smoke: login, dashboard, mistakes, coach, and the SPA loads, in an already-signed-in session if you have one; otherwise check that the SPA loads, add MI-5a to the "owner login smoke pending" note, and carry on.
   - Landscape's xlearn cards are still healthy.
   - After about 1 h, no service logs a JWKS refresh failure.
   - **Any 401 or timeout: `git revert` the MI-5a PR.**

6. **Verify [H]:**
   - `hack/host-lint.sh` is clean: the embedded `expected-netpol.tsv` equals the file, with all 9 new rows.
   - `ssh sujaykumar-vps 'bash -s -- --cluster --nats-stage=open' < ../infra/hack/host-verify.sh` is green, with NetworkPolicy presence covering all 9 new names and no restarts beyond the two annotation bumps.
   - **Negative check:** `ssh sujaykumar-vps "bash -s -- --cluster --nats-stage=open --netpol-file <(printf 'databases\tno-such-policy\tneg\n')" < ../infra/hack/host-verify.sh` must **FAIL** `cluster.netpol`, which proves the presence check isn't vacuous. Paste both runs into the MI-5a PR.
   - Note in status.md that the next xlearn tag's fleet rollout is the full re-proof; its release checklist runs `host-verify` and the smoke.

7. **Record [X]** (plan task 7): open an xlearn docs PR on `docs/mi-03-fences` with:
   - `docs/v2/status.md` (see Update status);
   - `docs/architecture/services.md`: a "Network fences (v2)" subsection with the matrix, "MI-5a is the internal-HTTP fence (ADR-0006/0016 as amended by ADR-0033)", the gateway's in-namespace JWKS, and "a regression fails open; `host-verify --cluster` presence is the detector";
   - ADR-0033 §12 row 7: the BP2 sign-off should already say "plus same-namespace pods on :8080 for in-cluster JWKS". **Verify it.** Only if it still says "the gateway admits only Traefik", append a **dated one-line clarification** (append-only; don't rewrite the decision).

## Constraints

- **GitOps only.** Every change is a `../infra` PR reconciled by Flux. Never `kubectl apply`, `rollout restart` or `delete pod` by hand. Restarts are `podAnnotations` bumps in git. `--dry-run=server`, `get`, `get --raw` and read-only `psql` are allowed.
- **Infra PRs stand alone**, never folded into a tag (rollout §2.2). The MI-5 PR merges first, alone, and it's the N3 precondition.
- **Selectors:** namespaces by `kubernetes.io/metadata.name`; callers by `app.kubernetes.io/instance`; the same namespace by `podSelector: {}`. **Never `app.kubernetes.io/part-of`**, which isn't on pods.
- **Forward-declare** identity, judge and coach (NATS) and judge (PG) now. Any **other** new caller later updates the policy in its own infra PR, merged before its tag (the ADR-0035 §2 standing rule).
- **Service boundaries:** the gateway never reaches PG or NATS.
- **Memory-sum rule (ADR-0035 §5):** no new pod. Restarts go one at a time, never a fleet restart, because the margin is about 0 before MI-11a.
- **D34:** no alerting, Flux `Alert` or ping. Detection is `host-verify --cluster`, run on demand.
- **Parallel sessions:** check peers' PRs and worktrees before each push. Don't tag. You shouldn't need an ADR number; if you do, check peers' ADR numbers first.

## Deliverables

- **`../infra` PR 1 (MI-5):** `infrastructure/database/cluster/networkpolicy.yaml`, `infrastructure/messaging/networkpolicy.yaml`, and 2 lines in `hack/expected-netpol.tsv`.
- **`../infra` PR 2 (MI-5 restart):** the review `podAnnotations` bump.
- **`../infra` PR 3 (MI-5a):** `networkPolicy` values in the 7 `apps/xlearn-*.yaml`, 7 lines in `hack/expected-netpol.tsv`, and the assessment `podAnnotations` bump.
- **An xlearn docs PR:** `docs/v2/status.md`, `docs/architecture/services.md` (the fences and the matrix), and the ADR-0033 row 7 clarification if needed.

## Update status

- **This sprint's file:** set the task rows in [`../sprints/sprint-mi-03.md`](../sprints/sprint-mi-03.md) to 🔄 or ✅, and _Overall_ to ✅. If MI-5a waits on mi-01, set 🔄 with the reason.
- **[`../status.md`](../status.md):**
  - **Sprint board:** mi-03 ✅.
  - **MI table:** row **MI-5** ✅ with the PR number and merge date (mi-06's N3 gate), and row **MI-5a** ✅ with its PR number and date (gates for m3-07 and l-01).
  - **Decisions log:**
    - the `part-of` finding and selectors by `instance` and namespace;
    - the gateway admits same-namespace for JWKS;
    - fresh-pod restart PRs as the proof method;
    - any live caller that wasn't in config;
    - the next fleet rollout is the full re-proof;
    - the Hostinger weekly image date read before the first restart, and the owner's username for the smoke.
  - **Pending-smoke notes:** "owner login smoke pending (MI-5, MI-5a)" if no signed-in session was available (the owner runs it).
- **ADRs:** none new. At most the dated ADR-0033 row 7 clarification.

## Done when (acceptance)

- [ ] MI-5 policies are live, and the selector proof matches exactly the intended callers.
- [ ] The fresh review pod is back on PG and NATS: 7 connections, consumers pending 0, outbox unsent 0, CNPG healthy.
- [ ] `host-verify`'s NATS stage read still works after MI-5.
- [ ] MI-5a's 7 policies render as specified (exact peers, no extras, no `part-of`), and the diff shows only policy additions plus the one annotation.
- [ ] A restarted service (assessment) re-fetches JWKS through MI-5a and serves an authed call: the public dashboard returns its assessment sections with no 401s.
- [ ] No caller is blocked: the public dashboard, the review workers and the SPA via Traefik work; login, the signed-in dashboard, mistakes and coach pass in a signed-in session, or "owner login smoke pending" is recorded.
- [ ] `host-lint.sh` is clean, `host-verify --cluster` is green with the 9 new three-column rows present, and the `--netpol-file` negative check FAILs.
- [ ] The matrix is recorded in the PRs and `docs/architecture/services.md`, and status.md has the MI-5 and MI-5a rows with dates.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: three `../infra` PRs (`feat(netpol): …`), then `docs/mi-03-fences` in xlearn.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. infra has no CI: paste the validation and smoke output into each PR body and merge each once validated and smoked.
3. **Release action — infra PR(s) only:** Merge the infra PRs in the plan's order (each its own PR, never folded into a tag): MI-5 → the MI-5 restart → MI-5a, letting Flux reconcile and verifying live after each (steps 2–5); then the xlearn docs/status PR. No tag. Run `host-verify --cluster` after the restart-inducing PRs (step 6).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
