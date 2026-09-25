# Sprint mi-11 — Track B finish: xlearn egress, PG connection limits, Renovate, N4 (MI-15)

> **Milestone:** MI — infra-first track (rollout step **MI-15**, second half; [mi-08](sprint-mi-08.md) landed the PSA labels and SA tokens; sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** infra (Track B hygiene) · **Order:** 52
> **Prereqs:** [m3-07](sprint-m3-07.md) (judge live dark with its nkey and its own default-deny egress) · [l-01](sprint-l-01.md) (coach on NATS with its nkey; the N3 ≥ 24 h re-check) · also [mi-03](sprint-mi-03.md) (MI-5a ingress) and [mi-06](sprint-mi-06.md) (N1–N3)
> **Unblocks:** [ga-01](sprint-ga-01.md) (MI complete before the GA PR) · [mi-13](sprint-mi-13.md) (MI-16 extends coach's egress from this policy)
> **Release action:** **infra PR(s) only**, plus one xlearn PR (`renovate.json`, the monthly-window runbook, status). The xlearn PR merges only; there's no tag, and nothing ships, because `main` is build-only.
> **Calendar:** November–December (after m3-07 and l-01). N4's fallback restart rides the next monthly window (ev-monthly-window, D22).
> **Owner time:** before launch only: ~5 min to install the Renovate GitHub App (task 6), and a look at the last weekly image date in hPanel. The session runs task 2's erase smoke itself (mint two throwaway testers with the CLI, then erase one after PR a and one after PR b): `identity admin` through `kubectl exec` is pre-approved by launching the prompt (D40).
> **Execute with:** [`../prompts/prompt-mi-11.md`](../prompts/prompt-mi-11.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Caller matrix from live config + `main` | H | ⬜ |
| 2 | xlearn egress PRs (3 batches, smoke after each, incl. the erase smoke) | I + H | ⬜ |
| 3 | PG role `connectionLimit: 20` (L21) | I | ⬜ |
| 4 | Renovate config — xlearn | X | ⬜ |
| 5 | Renovate config — infra | I | ⬜ |
| 6 | Install the Renovate app (owner) — before launch | O | ⬜ |
| 7 | N4: remove `legacy` + `no_auth_user` together | I | ⬜ |
| 8 | Monthly-window runbook | X | ⬜ |
| 9 | Verify | H | ⬜ |
| 10 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + MI track row MI-15 + the NATS rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **N3 ≥ 24 h re-check recorded** ([l-01](sprint-l-01.md) task 1): `host-verify --cluster --nats-stage=n3` showed no `legacy` connection ≥ 24 h after N3
- [ ] **judge ([m3-07](sprint-m3-07.md)) and coach ([l-01](sprint-l-01.md)) connect with their nkeys.** `/connz?auth=true` shows every client on its own nkey and **zero** on `legacy`. N4 would cut any anonymous client
- [ ] **MI-5a ingress live** ([mi-03](sprint-mi-03.md)). Every egress rule below has a matching ingress rule on its callee. Chart 0.3.0's `networkPolicy.egress` template is available ([mi-01](sprint-mi-01.md))
- [ ] **The last Hostinger weekly image is ≤ 7 days old** (the owner reads hPanel before launch; launching attests it, D40). N4 may fall back to a NATS restart
- [ ] **Parallel sessions:** no open peer PR touches `apps/xlearn-*.yaml`, `infrastructure/database/cluster/cluster.yaml` or `infrastructure/messaging/release.yaml` (`gh pr list -R sujaykumarsuman/infra`, `git worktree list`, ListAgents). In particular, [mi-12](sprint-mi-12.md) edits `apps/xlearn-judge.yaml`; **this sprint doesn't touch judge**

## Goal

Finish the Track B hygiene that doesn't gate M3 but belongs in v2.0
([ADR-0030 §5](../../adr/0030-runner-technology-and-host-hardening.md#5-host-and-cluster-hardening) Track B,
[ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md)):
- **egress NetworkPolicies** for the seven v1 `xlearn` services, built from an explicit per-release caller
  matrix, so a compromised service can no longer dial anywhere;
- **per-role Postgres connection limits** (L21);
- **Renovate** for the pins D22 promises: the runner toolchain, base images, charts, k3s and CNPG. PRs only;
- **N4**, which removes the NATS `legacy` bridge user and `no_auth_user` together, so `auth_required: true`
  and no anonymous connection is possible;
- the **monthly-window runbook** (D22).

## Scope

**In**
- `xlearn` egress from an explicit caller matrix: kube-dns everywhere, PG and NATS where used, in-namespace
  HTTP from live config plus forward-declared callers, TCP 443 only for identity (GitHub OAuth) and coach
  (LLM providers). judge keeps its own policy.
- CNPG per-role `connectionLimit: 20` on every xlearn login role (L21).
- Renovate in both repos (`renovate.json`), PRs only, no automerge, and scoped so it never fights Flux image
  automation.
- N4 (`legacy` + `no_auth_user` removed in one reload; a restart fallback in the monthly window).
- `docs/v2/runbooks/monthly-window.md` (D22, ev-monthly-window).

**Out**
- **judge's egress** (born in [m3-07](sprint-m3-07.md), already including judge → identity :8081 for the
  erase re-verify; M4's TCP 443 is [mi-12](sprint-mi-12.md)'s). This sprint doesn't edit `apps/xlearn-judge.yaml`.
- **coach WSS egress for voice** (MI-16, [mi-13](sprint-mi-13.md)). coach's 443 rule here is CIDR-wide and
  probably already covers the provider WebSocket; mi-13 confirms it and adds nothing it doesn't need.
- **PSA labels and SA tokens off** (the first MI-15 slice, [mi-08](sprint-mi-08.md)).
- **The `xlearn-runner` namespace** (`sandbox-guards`, [mi-14](sprint-mi-14.md)): default-deny already.
- **L22** CNPG memory (trigger-only). A metrics/logs stack or any alerting (D34).

## Tasks

### 1 · Caller matrix from live config + `main` [H]

A caller missing from the matrix is cut **silently**, so build the matrix from evidence, not memory:
- live env in `../infra/apps/xlearn-*.yaml`: `*_BASE_URL`, `JWKS_URL`, `NATS_URL`, `PG*`;
- `main`'s `internal/*/config.go` env reads and outbound clients. Anything merged but **not yet tagged** is
  forward-declared, the way MI-5 did it;
- `/connz?auth=true` (NATS clients, via `host-verify --cluster`);
- the MI-5a ingress values (every egress rule needs its callee's ingress rule);
- the **caller hand-offs earlier sprints recorded in status.md for this matrix**:
  - [m2-01](sprint-m2-01.md) task 8: practice → review :8084 (`REVIEW_BASE_URL`, the touch-start read of
    `GET /internal/revisions/{id}`, live since v1.9.0);
  - [l-01](sprint-l-01.md): practice, assessment and coach → identity :8081 (`IDENTITY_BASE_URL`, the erase
    re-verify `GET /internal/erasures/{id}`; review already had it) and coach → NATS :4222 (v1.11.0);
  - [m3-05](sprint-m3-05.md) / [m3-07](sprint-m3-07.md): judge → identity :8081, already in judge's own
    policy (nothing to add here).
- external hosts in code. On 2026-09-24 only `internal/identity/oauth.go` (github.com, api.github.com) and
  `internal/coach/config.go` (api.openai.com, api.anthropic.com) dial out.

The expected shape, on 2026-09-24 plus the v2 callers landed or planned before this sprint. **Verify every cell:**

| Caller | DNS :53 | PG :5432 | NATS :4222 | In-namespace HTTP | TCP 443 |
|---|---|---|---|---|---|
| gateway | ✓ | — (not a DB owner) | — | identity 8081, curriculum 8082, practice 8083, review 8084, assessment 8085, coach 8086; **judge 8087 forward-declared** ([m3-09](sprint-m3-09.md) BFF, `JUDGE_BASE_URL` set in [m3-13](sprint-m3-13.md)) | — |
| identity | ✓ | ✓ | ✓ (N2) | gateway 8080 (JWKS) | ✓ GitHub OAuth |
| curriculum | ✓ | ✓ | — | — (no `JWKS_URL` today; re-check) | — |
| practice | ✓ | ✓ | ✓ | gateway 8080 (JWKS); **review 8084** (touch start, [m2-01](sprint-m2-01.md)); **identity 8081** (erase re-verify, [l-01](sprint-l-01.md)); **judge 8087 forward-declared** ([m3-08](sprint-m3-08.md) reconciler, v1.14.0) | — |
| review | ✓ | ✓ | ✓ | gateway 8080; identity 8081 (ADR-0016 workers and the erase re-verify); curriculum 8082 (ADR-0016 workers) | — |
| assessment | ✓ | ✓ | ✓ | gateway 8080; **identity 8081** (erase re-verify, [l-01](sprint-l-01.md)) | — |
| coach | ✓ | ✓ | ✓ ([l-01](sprint-l-01.md)) | gateway 8080; **identity 8081** (erase re-verify, [l-01](sprint-l-01.md)) | ✓ LLM providers (BYO keys) |
| judge | unchanged: [m3-07](sprint-m3-07.md)'s policy (DNS, PG, NATS, runner, gateway 8080, **identity 8081** for the erase re-verify, born there); M4 adds only TCP 443, in [mi-12](sprint-mi-12.md) | | | | |

Destinations, **read live** (labels checked 2026-09-24):

| Destination | Selector / range |
|---|---|
| DNS | ns `kube-system`, pods `k8s-app: kube-dns`, UDP **and** TCP 53 |
| PG | ns `databases`, pods `cnpg.io/cluster: projects-pgstore`, TCP 5432 |
| NATS | ns `messaging`, pods `app.kubernetes.io/name: nats`, TCP 4222 |
| In-namespace | pods `app.kubernetes.io/instance: xlearn-<svc>` + the port |
| TCP 443 | `ipBlock 0.0.0.0/0` **except** `10.0.0.0/8` (covers pod `10.42.0.0/16` and service `10.43.0.0/16`), `172.16.0.0/12`, `192.168.0.0/16`, `100.64.0.0/10`, `169.254.0.0/16` and the **node IP `/32`** (read live: `k3s kubectl get node -o wide`). The same except-list [mi-12](sprint-mi-12.md) task 4 uses for judge |

Paste the matrix, with its evidence column, into each egress PR.

### 2 · xlearn egress PRs (3 batches, smoke after each) [I + H]

Set `networkPolicy.egress` (chart 0.3.0; a list adds `Egress` to `policyTypes`) in each
`apps/xlearn-<svc>.yaml`, next to MI-5a's ingress values. There are three PRs, merged in order. After each:
smoke, then `host-verify --cluster`. Revert with `git revert`; it fails open.

| PR | Releases | Smoke after merge |
|---|---|---|
| **a** | curriculum, assessment, review, practice | dashboard and week views; a review background worker run (identity + curriculum calls, ADR-0016); the outbox `unsent` count drains on practice, review and assessment; **forced JWKS re-fetch**: the same PR bumps a `podAnnotations` key (e.g. `xlearn.dev/restart`) on review, so Flux rolls it and it fetches JWKS under the new policy; then an authed call through review (mistakes page); **a touch start** (practice → review :8084): start a due touch if the owner has one, else `POST …/touches/<random UUID>/start` must answer 404, never 503 `touches_unavailable` (review unreachable); **erase round trip #1** (below) |
| **b** | identity, coach | a **GitHub OAuth login round trip** (the token exchange is 443 to github.com); email login; a coach chat that streams (the owner's BYO key); identity's outbox drains to `XLEARN_IDENTITY`; coach shows on `/connz` with its nkey; **erase round trip #2** (below) |
| **c** | gateway | login, the dashboard, every screen's BFF call, the public profile `/xlearn/u/<owner>`, coach SSE through the gateway |

- **The erase round trip** is the path [l-01](sprint-l-01.md) warned breaks silently: a consumer that can't
  reach identity :8081 naks and retries, so the request just never closes, and nothing alerts (D34). The
  session runs it itself: the CLI over `ssh sujaykumar-vps 'k3s kubectl exec …'` is pre-approved by launching the prompt (D40).
  - **Before PR a**, the session mints two throwaway testers with the CLI (`identity admin account create --role
    tester --email …`, as in ev-first-tester; no data needed).
  - **After PR a and again after PR b**, the session erases one of them with l-02's CLI verb
    (`identity admin account erase <email> --confirm <email>`, via `kubectl exec`).
  - **Pass:** within a few minutes, `identity admin erasures --since <today>` (a read-only `kubectl exec`) shows the
    request **closed with every expected ack**: 5 once judge's consumer is live (practice, review, assessment,
    coach, judge). There are no new `event_dead_letter` rows in those services.
  - **A request still open** means a consumer is cut. Read its logs, then `git revert` the batch.
  - Log the `kubectl exec` uses in status.md (the sanctioned manual path, as in l-02).

- **The practice reconciler** (a judged submit in the cohort): if v1.14.0 is live, smoke it after PR a.
  If it isn't, the practice → judge rule is **forward-declared**. It's harmless, and m3-13's post-tag smoke
  exercises it. Record which in the decisions log.
- **From now on, the standing NetworkPolicy rule covers ingress AND egress**, as every tag sprint's release
  checklist already words it (ADR-0035 §2's text says only "updates the policy"). Every later tag that adds an
  in-cluster caller needs its egress rule too, in its own infra PR merged before the tag. Name the known
  ones in the PR: gateway → judge (m3-13), practice → judge (m3-08/v1.14.0), judge TCP 443 (mi-12, in judge's
  own policy), coach WSS (mi-13).
- If the chart emits any new object name, update `hack/expected-netpol.tsv` ([mi-02](sprint-mi-02.md)) **and**
  its byte-identical embedded copy between the `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv`
  markers in `hack/host-verify.sh`, in the same PR, and run `hack/host-lint.sh`. The 0.3.0 template puts egress
  in the same per-release policy, so the names should be unchanged.

### 3 · PG role `connectionLimit: 20` (L21) [I]

- In `infrastructure/database/cluster/cluster.yaml` `managed.roles`, set `connectionLimit: -1` → **`20`** for
  `xlearn_identity`, `xlearn_curriculum`, `xlearn_practice`, `xlearn_review`, `xlearn_assessment`,
  `xlearn_coach`, `xlearn_judge` and the bootstrap owner `xlearn`. No app logs in as `xlearn`; 20 leaves room
  for the owner's ad-hoc use. Leave any non-xlearn role alone.
- **Why 20:** `MaxConns` is pinned at 4 per pod (judge 8) since N0 (L21). A rolling update holds two pods:
  8 connections, 16 for judge. Both fit under 20, and the sum stays well under Postgres' 100.
- CNPG applies it with `ALTER ROLE`: no PG restart, no pod roll. Verify through the Cluster status
  (`.status.managedRolesStatus`), a read-only `get`.

### 4 · Renovate config — xlearn [X]

Add `renovate.json` at the repo root. Validate it with `npx --yes --package renovate renovate-config-validator`
(use the current schema's key names).
- **Managers, deliberately narrow:**
  - `dockerfile` for `deploy/*.Dockerfile` base images, with `pinDigests: true` and one group;
  - `gomod` **only** for the go-sandbox pin (`github.com/criyle/go-sandbox`, or wherever m3-03 pinned or
    vendored it);
  - a custom regex manager for the runner toolchain `ARG *_VERSION` pins m3-15 declared in
    `deploy/runner.Dockerfile` (Go via the `golang-version` datasource; the others per their pins).
  - Everything else is disabled, including npm, other go modules and GitHub Actions. Widening the scope is a
    later, deliberate change.
- **Policy:** `automerge: false`, `platformAutomerge: false`, label `deps` (plus `runner` on the runner
  group), a weekly schedule, a low `prConcurrentLimit`.
- **No vulnerability-alert PRs** (D34): set `vulnerabilityAlerts: {enabled: false}` and leave
  `osvVulnerabilityAlerts` at `false`. Renovate's vulnerability PRs are on by default and bypass the
  schedule; bumps arrive on the weekly schedule like everything else.
- **Runner pins are a runner release.** A toolchain bump is a runner patch (`runner-v1.x.y`) with the ±5%
  speed check ([t3 §6.1](../research/t3-sandbox.md#61-image-and-release)). Renovate can't recompute a
  tarball checksum, so its PR fails CI until a human updates the `*_SHA256`. That's intended: the PR is the
  reminder.

### 5 · Renovate config — infra [I]

`renovate.json` in `../infra`, validated the same way:
- the **`flux` manager** for HelmRelease chart versions under `infrastructure/**` (Longhorn, cert-manager,
  the CNPG operator, NATS). By default it matches only `gotk-components.yaml`, so set
  `flux: {managerFilePatterns: ["infrastructure/**/*.yaml"]}`. The HelmRepository sources it resolves
  against live in the same tree (`infrastructure/*/repository.yaml`, `controllers/jetstack.yaml`). One PR per
  chart, labelled `deps` and `restart`: each bump restarts its workload, so merge only in a window;
- a **regex manager for `PIN_K3S_VERSION`** in `hack/host-bootstrap.sh` **and** `hack/host-verify.sh` as
  one PR (the shared constants must stay identical; `host-lint.sh` checks it). Datasource: GitHub releases
  `k3s-io/k3s`, patch updates within the pinned minor only, no prereleases. This is D22's monthly k3s patch;
- a **regex manager for the CNPG `imageName`** in `infrastructure/database/cluster/cluster.yaml`
  (`<major>.<minor>-system-trixie`, minor only, a PG restart);
- **`ignorePaths`:**
  - `apps/**` and `runner/**`: Flux image automation owns those tags and digests; two bots on one line
    would fight;
  - `clusters/vps/flux-system/**`: bootstrap-managed; a Flux upgrade is a deliberate `flux bootstrap`;
  - `charts/project/**`: the local chart;
- no automerge, label `deps`, a weekly schedule, a low `prConcurrentLimit`;
- `vulnerabilityAlerts: {enabled: false}`, `osvVulnerabilityAlerts: false`, as in task 4 (D34).
- After the app runs, check that the Dependency Dashboard or the first PRs list the four charts. If the flux
  manager found nothing, the file patterns are wrong.

### 6 · Install the Renovate app [O, before launch]

**Before launch**, the owner installs the Mend Renovate GitHub App on **`sujaykumarsuman/xlearn` and
`sujaykumarsuman/infra` only** (selected repositories), about 5 minutes. Granting app permissions is the
owner's action; launching the prompt attests it's done (D40). The session verifies the installation, then
confirms that Renovate picked up each repo's in-repo config once it's merged, opened at least one PR, and
merged none. If Renovate opened its onboarding PR before a repo's `renovate.json` landed, the merged config
supersedes it: close it unmerged unless Renovate already has. If the app turns out to be missing, the
configs stay dormant: record ⛔ "Renovate app not installed" in status.md and land the rest. A self-hosted
Renovate run (for example a GitHub Actions workflow on a schedule) is the alternative, but it would add a PAT
and a timer, so it isn't built here.

### 7 · N4: remove `legacy` + `no_auth_user` together [I]

- **Pre-check:** `host-verify --cluster --nats-stage=n3` shows zero `legacy` connections, and
  `/connz?auth=true` shows all six service nkeys (practice, review, assessment, identity, judge, coach).
- **Rehearse locally first** on the exact server version `/varz` reports (2.14.6):
  - run `nats-server` in a container with the live N3 config (the golden block + `legacy` deny +
    `no_auth_user`) and connect a client;
  - apply the N4 edit, then `nats-server --signal reload`. The log must show the reload applied, and `/varz`
    must show `auth_required: true`;
  - then rehearse the **revert**, which re-adds `no_auth_user`.

  [t7 C3](../research/t7-cross-cutting-and-rollout.md): on 2.10.22 a reload refuses to *add* `no_auth_user`,
  so a revert needs a restart. Record both results.
- **PR:** in `infrastructure/messaging/release.yaml`, remove the `legacy` user **and** the top-level
  `no_auth_user` in **one commit** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first),
  N4). The chart's config reloader signals the server.
  - The same PR bumps `hack/host-verify.sh`'s `PIN_NATS_STAGE` to `n4` ([mi-02](sprint-mi-02.md)), so a
    plain `host-verify --cluster` checks the live stage. A revert rolls both back together.
- **Verify:**
  - `host-verify --cluster --nats-stage=n4` shows `/varz` `auth_required: true`;
  - `/connz?auth=true` shows all six nkeys connected;
  - the outbox `unsent` count is 0 everywhere and consumer pending is 0;
  - no permission-violation ERROR lines.
- **If the reload is refused** (`config_load_time` unchanged, `auth_required` still false, a reload error in
  the NATS log): **revert at once**, so git matches the running config again. Then land N4 **plus a
  pod-template annotation bump** as one PR in the next monthly window (task 8). That's one restart, and the
  outboxes buffer. **Never leave git ≠ the running server.**
- **Reverting a successful N4** also needs a restart, because `no_auth_user` can't be re-added by reload.
  Do it in a window, not ad hoc.

### 8 · Monthly-window runbook [X]

`docs/v2/runbooks/monthly-window.md`: the D22 cadence, executed by the owner (ev-monthly-window).
- **When:** monthly, a quiet hour. No tag or deploy in flight; parallel sessions paused.
- **Before:**
  - the last Hostinger weekly image is ≤ 7 days old;
  - **a manual Hostinger snapshot before every monthly reboot**, kernel-only windows included, and wait until
    it completes. This is the infra README "Kernel reboot runbook" step 1, which this runbook keeps;
  - the Hostinger VNC console open (the same step 1): the way back in if SSH doesn't return;
  - **no off-node `pg_dumpall`** ([ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule)).
    That dump is the only part of README step 1 this runbook skips. [mi-09](sprint-mi-09.md) flagged the
    conflict; this sprint resolves it per ADR-0034 §4.3: a small infra PR drops the dump from that README
    step (keeping the snapshot and VNC parts), pre-approved by launching the prompt (D40). Record the
    decision in the Decisions log.
- **Kernel:**
  - `host-bootstrap.sh --with-sandbox --dry-run`, then apply. From the October window on, every bootstrap run
    passes `--with-sandbox`, so drift in the sandbox block gets corrected too;
  - `host-verify --pre-reboot --cluster` must say GO;
  - the one-shot GRUB reboot, the confirm step and the rollback: link the infra README "Kernel reboot runbook"
    steps 3–5, don't copy them (its step 1 is covered under **Before**, its step 2 by the two bullets above);
  - after the reboot: `host-verify --cluster --expect-sandbox --nats-stage=<live>`, the runner Ready (its
    `/readyz` canaries), and a smoke test of login, the dashboard and coach.
- **k3s patch (D22):** if Renovate opened a `PIN_K3S_VERSION` PR, merge it here and run
  `host-bootstrap.sh --with-sandbox --with-k3s --replace-k3s` (one restart). Then `host-verify --expect-sandbox`:
  `sandbox.containerd` must pass, so the drop-in import (or, if the spike recorded the `.tmpl` fallback, the
  template and its rendered runc settings) survived the upgrade.
- **Restart-inducing Renovate PRs** (charts, CNPG image): merge them only here, one at a time, with
  `host-verify --cluster` after each.
- **N4 fallback:** if task 7's reload was refused, open a PR from the N4 + annotation-bump branch that
  status.md records, and merge it here (the one NATS restart). Then run `host-verify --cluster --nats-stage=n4`.
- **On-demand reads while you're there** (D34: checks, not alerts):
  - the evalpack PAT expiry date in status.md (ev-pat-expiry);
  - the TR-STEAL, TR-CPU, TR-MEM and TR-DISK lines from `host-verify --cluster`;
  - Renovate's open PRs;
  - dead-letter rows, if an admin read exists.
- **Record** a monthly-window log line in status.md.

### 9 · Verify [H]

- `host-verify --cluster --nats-stage=n4` (or `n3` if N4 was deferred) is green against the expected-policy
  list. If task 2 changed any name, the `.tsv` and its embedded copy in `host-verify.sh` changed together and
  `hack/host-lint.sh` is green.
- The memory sum is unchanged (no pods added).
- Every xlearn pod is Running with no restarts caused by this sprint (only the annotation-bumped review
  rolled).
- Renovate PRs are open and none is merged.

### 10 · Record [X]

In `docs/v2/status.md`:
- the MI track MI-15 row ✅ (with mi-08's slice), with the PR numbers;
- the NATS rows: the N4 date, or "deferred to the monthly window of <date>" plus the reverted PR;
- the Sprint board row;
- Decisions log lines: the egress matrix (a link to the PR), which callers were forward-declared,
  `connectionLimit` 20 including the owner role, Renovate's scope and ignored paths, the N4 reload outcome on
  2.14.6, and the off-node `pg_dumpall` dropped from the infra README per ADR-0034 §4.3 (mi-09's flag);
- the erase log (the two throwaway-tester erases: request id, date, via `cli`, acks) and the `kubectl exec`
  uses (the manual-path log);
- the MI milestone row: set it ✅ when its exit criteria hold (Track A green; the runner acceptance suite
  passes dark; `host-verify --cluster` extended green). They may already hold since mi-10. MI-14 and MI-16
  keep their own rows.

## Acceptance criteria

- [ ] Egress policies present on all seven v1 releases; OAuth, coach, the JWKS re-fetch, review workers and the practice reconciler are unaffected (the reconciler if v1.14.0 is live, else forward-declared and recorded)
- [ ] Both erase round trips close with every expected ack and no new dead-letter rows (after PR a and after PR b); a touch start reaches review (never 503 `touches_unavailable`)
- [ ] `auth_required: true`; no anonymous connection possible (`host-verify --cluster --nats-stage=n4`), or N4 reverted cleanly and scheduled for the monthly window
- [ ] Renovate opens PRs and merges none automatically, in both repos
- [ ] Every xlearn login role at `connectionLimit: 20`, with no PG restart
- [ ] `docs/v2/runbooks/monthly-window.md` merged
- [ ] `host-verify --cluster` green with the updated expected-policy list

## Release

**Infra PR(s) only**, merged in this order:
1. egress a, then b, then c (smoke between);
2. the PG role limits;
3. Renovate (infra);
4. the README alignment (docs only: the off-node dump dropped, ADR-0034 §4.3);
5. **N4 last**, so any breakage is attributable.

The xlearn PR (`renovate.json`, the runbook, status) merges only. There's no tag; `main` is build-only, so
nothing ships.

If N4's reload is refused, the revert merges in this sprint. The N4 + annotation-bump change is pushed as
a **branch without a PR** (e.g. `feat/nats-n4-restart`), and status.md records its name. The monthly window
opens and merges that PR, so this session leaves no open PR behind.

## Definition of Done

All infra PRs merged, or N4 reverted and scheduled · smoke evidence in each PR · Renovate live and
PR-only · runbook merged · statuses updated (this file + [`../status.md`](../status.md)) · no `kubectl apply`
or `kubectl rollout restart` (restarts come from annotation bumps in git) · no alert, timer or CronJob (D34).

## Risks / watch-outs

- **N4's reload may be refused on 2.14.6.** Revert at once, then restart in the monthly window. Rehearse
  locally first.
- **Reverting N4 needs a restart** (`no_auth_user` can't be re-added by reload).
- **An egress policy missing a port breaks OAuth callback exchanges** (identity → github.com:443) or the DNS
  TCP fallback. Allow DNS on UDP **and** TCP 53.
- **A caller missing from the matrix is cut silently.** Build the matrix from live config plus `main` and
  the earlier sprints' hand-offs (m2-01, l-01, m3-05/m3-07), and forward-declare. The erase path is the
  quiet one: a cut consumer naks and retries, so the request just stays open. That's why the smoke includes
  two erase round trips. [Rollout §12](../rollout-plan.md#12-downstream-constraints-for-the-build-plan-session)'s
  "M4 adds only 443" stands: judge → identity :8081 moved forward to [m3-07](sprint-m3-07.md), so
  [mi-12](sprint-mi-12.md) adds only judge's TCP 443.
- **From here on, every new in-cluster caller needs an egress rule as well as an ingress rule** before its
  tag. That is the standing rule as every tag sprint's release checklist carries it ("ingress and egress");
  ADR-0035 §2's own sentence says only "updates the policy". Name the known upcoming callers in the PR so
  their sprints don't miss them.
- **443 is CIDR-wide.** Vanilla NetworkPolicy has no FQDN rules, so identity and coach can reach any public
  443 host. That's accepted (the same except-list as judge's in mi-12 task 4). DNS tunnelling through CoreDNS remains.
- **Renovate vs Flux:** without `ignorePaths` on `apps/**` and `runner/**`, Renovate would fight the image
  automation over tags and digests.
- **Restart-inducing Renovate PRs** (charts, CNPG image, the k3s pin) merge only in a window.
- **`too many connections for role`** during a migration plus a rollout means the limit is too tight. Raise
  it; don't remove it.
- **D34:** Renovate's PRs and dependency dashboard are views, not alerts. Vulnerability-alert PRs are
  disabled explicitly (`vulnerabilityAlerts.enabled: false`), and no push channel is added.
- **Renovate's `flux` manager finds nothing** without `managerFilePatterns`: by default it reads only
  `gotk-components.yaml`, which is ignored anyway. Check that the first run lists the four charts.
