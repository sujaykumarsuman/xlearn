# Sprint mi-03 — Fences: databases/messaging ingress, xlearn ingress (MI-5, MI-5a)

> **Milestone:** MI — cluster safe for untrusted code (rollout steps **MI-5** and **MI-5a**, ADR-0030 Track A **A4**)   ·   **Track:** infra
> **Prereqs:** [mi-01](sprint-mi-01.md) (chart 0.3.0 NetworkPolicy knobs, needed by MI-5a only), [mi-02](sprint-mi-02.md) (`host-verify --cluster` presence check, `--nats-stage`)
> **Unblocks:**
> - [mi-06](sprint-mi-06.md): the MI-5 PR merged is the **N3 precondition**;
> - [l-01](sprint-l-01.md): L-E needs MI-5 and MI-5a live;
> - [m3-07](sprint-m3-07.md): MI-13 judge needs MI-5a. MI-5 and MI-5a are also M3 hard-checklist items (rollout §5).
>
> **Release action:** **infra PR(s) only**. Three `../infra` PRs, merged in order:
> 1. MI-5, the `databases` + `messaging` ingress;
> 2. the MI-5 caller restart, one `podAnnotations` bump;
> 3. MI-5a, the xlearn ingress.
>
> An xlearn docs PR carries the architecture note and status. No xlearn tag.
> **Calendar:** week 2 (Mon 2026-10-05 → Fri 2026-10-09). **MI-5 must merge before mi-06's N3** (week 3).
> **Owner:** before launch only: read the Hostinger weekly image date in hPanel and record it in status.md (entry gate). The login smokes never wait for the owner: they run in an already-signed-in browser session if the session has one, and otherwise become an "owner login smoke pending" note in status.md (D40); an agent never enters credentials.
> **Execute with:** [`../prompts/prompt-mi-03.md`](../prompts/prompt-mi-03.md). One prompt, one session.
>
> Sprint ids `mi-NN` are not rollout step ids `MI-N`. This sprint executes **MI-5** and **MI-5a**. MI-4 is [mi-14](sprint-mi-14.md) and MI-5b is [mi-04](sprint-mi-04.md).

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Caller matrix from live config and live connections (both PRs) | H | ⬜ |
| 2 | MI-5 PR: `databases` + `messaging` ingress (callers forward-declared) | I | ⬜ |
| 3 | MI-5 smoke: selector proof, caller restart PR, CNPG/NATS/outbox checks; login smoke (signed-in session, else a pending-smoke note) | H + I | ⬜ |
| 4 | MI-5a PR: xlearn ingress from the chart 0.3.0 knobs (Traefik + same-namespace JWKS) | I | ⬜ |
| 5 | MI-5a smoke: JWKS through the fence from a fresh pod, internal calls; login smoke (signed-in session, else a pending-smoke note) | H | ⬜ |
| 6 | Verify: `host-lint` clean, `host-verify --cluster` green with the 9 new policy names, negative check FAILs | H | ⬜ |
| 7 | Record: status.md MI rows, architecture note, ADR-0033 row 7 clarification | X | ⬜ |

> **Keep this current.** Set a task to 🔄 when you start it, to ✅ when its acceptance bullet passes, and to ⛔ if it's blocked (say why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, the MI table rows **MI-5** and **MI-5a**, the decisions log, and the pending-smoke notes if a login smoke is pending. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] The MI-8 `host-verify --cluster` extension is merged ([mi-02](sprint-mi-02.md)), with the NetworkPolicy presence check, `hack/expected-netpol.tsv` and `--nats-stage=open`. Both PRs need it.
- [ ] MI-3 chart 0.3.0 is merged ([mi-01](sprint-mi-01.md)), with multi-source ingress and a same-namespace source. **MI-5a only.** MI-5 is raw manifests and may go first if mi-01 slips, since it's the N3 precondition.
- [ ] Local `../infra` `main` is synced. Peer check: no open PR touches `apps/xlearn-*.yaml`, `infrastructure/{database/cluster,messaging}/` or `hack/expected-netpol.tsv`. [mi-14](sprint-mi-14.md) also edits the `.tsv`, so rebase whichever lands second.
- [ ] **[O, before launch]** The date of the last Hostinger weekly image is checked and recorded in status.md ([rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag): check it before any restart-inducing step). The owner reads hPanel before launch; the session verifies the date is recorded. This sprint's two `podAnnotations` bumps each restart one stateless pod. If the date is missing, don't wait: land MI-5 (task 2 restarts nothing) and the docs PR, and set the restart-inducing work (task 3's restart PR, tasks 4–5) ⛔ in status.md, naming the owner item.
- [ ] No xlearn release is rolling out: every `xlearn-*` Deployment is available and no tag was pushed in the last 15 min. Otherwise a blocked caller gets confused with a rollout, and a fleet surge stacks on this sprint's restarts. The memory margin is about 0 before MI-11a (ADR-0035 §5).

## Goal

Stand up the two caller fences v2 depends on, without breaking any live in-cluster call:
- **MI-5:** ingress policies for `databases` (PG 5432, CNPG :8000) and `messaging` (NATS 4222; 8222 admits no pod). They admit today's callers **and forward-declare** tomorrow's. This is the N3 precondition and an M3/L-E gate.
- **MI-5a:** the **`xlearn` ingress fence**, which makes the unauthenticated internal HTTP routes safe (`/sessions/*`, `/internal/*`). The gateway admits Traefik **plus same-namespace pods**, because every service fetches **JWKS from `xlearn-gateway:8080`**. Internal services admit only the `xlearn` namespace, and the runner namespace is admitted nowhere.

## Scope

**In**
- MI-5 raw NetworkPolicies:
  - `databases/projects-pgstore-ingress` in `infrastructure/database/cluster/`;
  - `messaging/nats-ingress` in `infrastructure/messaging/`.
  
  Callers are selected by namespace `xlearn` AND `app.kubernetes.io/instance`.
- One caller-restart PR (a GitOps `podAnnotations` bump on `xlearn-review`), so a fresh pod proves the MI-5 path end to end.
- MI-5a per-release `networkPolicy` values in all 7 `apps/xlearn-*.yaml` through the chart 0.3.0 knobs. The same PR bumps `podAnnotations` on `xlearn-assessment` for the fresh-pod JWKS proof.
- `hack/expected-netpol.tsv` gains the 9 names (2 + 7) in [mi-02](sprint-mi-02.md)'s three-column format, and so does its **embedded copy** in `hack/host-verify.sh`.
- The caller matrix goes into both PR descriptions and `docs/architecture/services.md`.

**Out (later sprints)**
- MI-4 `sandbox-guards` and the runner namespace's own policies: [mi-14](sprint-mi-14.md).
- **xlearn egress** (MI-15): [mi-11](sprint-mi-11.md). That sprint re-derives the egress matrix from the same live config.
- **judge's own policies** (born with MI-13; default-deny egress): [m3-07](sprint-m3-07.md). This sprint only forward-declares judge as a caller.
- **NATS auth** (MI-7, N1–N3): [mi-06](sprint-mi-06.md).
- **Admin-console isolation** (MI-5b): [mi-04](sprint-mi-04.md).
- **PSA labels / SA tokens:** [mi-08](sprint-mi-08.md).

## Tasks

### 1 · Caller matrix from live config and live connections [H]

Build it from `../infra/apps/xlearn-*.yaml` env (`*_BASE_URL`, `JWKS_URL`, `NATS_URL`, `PG*`), then **confirm it against live state**. All of these reads are read-only:
- **NATS:** `/connz` through the API-server proxy (`k3s kubectl get --raw /api/v1/namespaces/messaging/services/nats-headless:8222/proxy/connz`). Map client IPs to pods with `get pods -n xlearn -o wide`.
- **PG:** `pg_stat_activity` via `k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select usename, client_addr, count(*) from pg_stat_activity where client_addr is not null group by 1,2"`.

Any caller that isn't in the matrix stops the sprint until it's explained.

Matrix as planned (config read 2026-09-24):

| Caller (`app.kubernetes.io/instance`) | PG 5432 | NATS 4222 | HTTP it calls (in `xlearn`) |
|---|---|---|---|
| `xlearn-gateway` | **no** | no | identity :8081, curriculum :8082, practice :8083, review :8084, assessment :8085, coach :8086 |
| `xlearn-identity` | yes | **forward** (N2, mi-06) | gateway :8080 (JWKS) |
| `xlearn-curriculum` | yes | no | none (no `JWKS_URL`) |
| `xlearn-practice` | yes | yes | gateway :8080 (JWKS) |
| `xlearn-review` | yes | yes | gateway :8080 (JWKS); identity :8081 and curriculum :8082 (ADR-0016 workers) |
| `xlearn-assessment` | yes | yes | gateway :8080 (JWKS) |
| `xlearn-coach` | yes | **forward** (L-E ack, l-01) | gateway :8080 (JWKS) |
| `xlearn-judge` (M3-1, m3-07) | **forward** | **forward** | gateway :8080 (JWKS); called by the gateway and practice |
| Traefik (`kube-system`, `app.kubernetes.io/name: traefik`) | — | — | gateway :8080 |
| CNPG operator (`cnpg-system`) | :8000 status only | — | — |
| Node / host (kubelet probes, API-server proxy used by `host-verify`) | node-local, always admitted | 8222 node-local | probes |

Live facts that shape the selectors (read 2026-09-24):
- **xlearn pods carry only `app.kubernetes.io/name: project` and `app.kubernetes.io/instance: <release>`.** The chart puts `app.kubernetes.io/part-of` on the **Deployment only**, never on the pod template (`charts/project/templates/deployment.yaml`). So a `part-of` selector **matches nothing**. Select by namespace, or by `instance`.
- The Traefik pod is labelled `app.kubernetes.io/name=traefik` in `kube-system`, as kubescope's working policy already relies on.
- The NATS pod `nats-0` is `app.kubernetes.io/{name,instance}=nats`. Its monitor port 8222 is only on the `nats-headless` Service.
- The PG pod `projects-pgstore-1` is `cnpg.io/cluster=projects-pgstore`.
- The CNPG operator runs in `cnpg-system`.
- `/connz` shows 7 connections, from practice, review and assessment only. identity joins NATS in [mi-06](sprint-mi-06.md)'s N2 PR (ADR-0035 §2 MI-5 table), so it's forward-declared here. If an identity connection already shows, it's a declared caller, not a stop: record it.

### 2 · MI-5 PR: `databases` + `messaging` ingress [I] (first)

These follow [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) (MI-5 table) and [t3 §8.4](../research/t3-sandbox.md).

`infrastructure/database/cluster/networkpolicy.yaml`:
```yaml
kind: NetworkPolicy        # databases/projects-pgstore-ingress
spec:
  podSelector: {matchLabels: {cnpg.io/cluster: projects-pgstore}}
  policyTypes: [Ingress]
  ingress:
    - from:   # the 6 DB-owning services + judge (forward-declared); NOT the gateway
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: xlearn}}
          podSelector:
            matchExpressions:
              - {key: app.kubernetes.io/instance, operator: In,
                 values: [xlearn-identity, xlearn-curriculum, xlearn-practice, xlearn-review,
                          xlearn-assessment, xlearn-coach, xlearn-judge]}
      ports: [{port: 5432, protocol: TCP}]
    - from: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: cnpg-system}}}]
      ports: [{port: 8000, protocol: TCP}]
    - from: [{podSelector: {matchLabels: {cnpg.io/cluster: projects-pgstore}}}]  # instance↔instance, for when `instances` > 1
      ports: [{port: 5432, protocol: TCP}, {port: 8000, protocol: TCP}]
```

`infrastructure/messaging/networkpolicy.yaml`:
```yaml
kind: NetworkPolicy        # messaging/nats-ingress
spec:
  podSelector: {matchLabels: {app.kubernetes.io/name: nats, app.kubernetes.io/instance: nats}}
  policyTypes: [Ingress]
  ingress:
    - from:   # practice, review, assessment today; identity (N2), judge (MI-13), coach (L-E) forward-declared
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: xlearn}}
          podSelector:
            matchExpressions:
              - {key: app.kubernetes.io/instance, operator: In,
                 values: [xlearn-practice, xlearn-review, xlearn-assessment,
                          xlearn-identity, xlearn-judge, xlearn-coach]}
      ports: [{port: 4222, protocol: TCP}]
    # no rule for 8222: no pod reads the monitor; host-verify reads it node-locally via the API-server proxy
```

**Comments.** Give each file a header in the house style. It should say:
- the list is the caller matrix;
- forward-declared selectors are harmless, and a missing one blocks its caller **silently** (identity's relay would only log "not ensured");
- **any new caller updates this policy in its own infra PR, merged before its tag** (the ADR-0035 §2 standing rule).

**Also in this PR:** `hack/expected-netpol.tsv` gets two rows in mi-02's three-column format (`namespace <TAB> name <TAB> added-by`): `databases<TAB>projects-pgstore-ingress<TAB>mi-03` and `messaging<TAB>nats-ingress<TAB>mi-03`.
- **Update the embedded copy too.** host-verify runs via `ssh vps 'bash -s' < host-verify.sh` and never sees the file; it reads the table between its `# >>> expected-netpol.tsv` / `# <<< expected-netpol.tsv` markers. Paste the same rows there, byte-identical.
- Run `hack/host-lint.sh` clean before opening the PR. It fails when the embedded copy differs from the file.

**Before merge:**
- server dry-run **the two new files only**: `ssh vps 'sudo k3s kubectl apply --dry-run=server -f -' < infrastructure/database/cluster/networkpolicy.yaml`, and the same for `infrastructure/messaging/networkpolicy.yaml`. That persists nothing. Don't run `kubectl kustomize` on the directories: they have no `kustomization.yaml` (Flux generates one; don't add one), and `infrastructure/database/cluster/` holds SOPS-encrypted `pg-*.enc.yaml` files that can't be dry-run;
- run the **selector proof**, which evaluates each policy's selectors against live pods:
  - `get pods -n xlearn -l 'app.kubernetes.io/instance in (…PG list…)' -o name` lists exactly the 6 live DB services;
  - the NATS list lists the 6 live instances (identity and coach pods exist but don't connect yet);
  - the target selectors match `projects-pgstore-1` and `nats-0`.

**Merge.** This PR alone is the N3 precondition. Record its merge date: [mi-06](sprint-mi-06.md) gates on it.

### 3 · MI-5 smoke [H + I]

Existing TCP connections (pgx pools, NATS clients) can survive a new policy through conntrack, so a wrong selector may only show on the next reconnect. Prove it with a fresh pod and with read-only checks:

1. **Restart PR [I].** Once `get networkpolicy -n databases,messaging` shows both policies, merge a one-line `../infra` PR that bumps `podAnnotations` on `apps/xlearn-review.yaml` (for example `sujaykumar.dev/restarted-for: mi-5`).
   - It must be a **separate** PR: `databases`, `messaging` and `apps` are separate Kustomizations with no apply order between them.
   - review is the right pod to restart: it touches PG, NATS (1 publisher + 2 consumers), identity, curriculum and JWKS.
   - Only one pod surges (+128 Mi), which is safe before MI-11a.
2. **Checks [H]** after the new review pod is Ready:
   - **NATS:** `/connz` shows 7 connections again, review's reconnected. `host-verify --cluster --nats-stage=open` still reads `/varz` and `/connz` through the API-server proxy. That proves node-local access to 8222 survives the policy.
   - **Consumers:** `/jsz?consumers=true` shows every durable with `num_pending` 0 and `num_ack_pending` 0.
   - **Outboxes:** unsent rows are 0 in practice, review and assessment (read-only `select count(*) … where sent_at is null`).
   - **CNPG:** `get clusters.postgresql.cnpg.io -n databases` shows "Cluster in healthy state". The operator logs show no connection errors for 2 min.
   - **App:** `curl -sf https://projects.sujaykumar.dev/xlearn/api/healthz`, and `GET /xlearn/api/u/<owner-username>`. The public dashboard runs gateway → identity → PG, and gateway → assessment and curriculum → PG, and needs no credentials.
     - **Find the username once, read-only:** `ssh vps 'sudo k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select username from identity.account where username is not null order by created_at limit 3"'`. Prod had one account on 2026-09-24, the owner's. Once m1-02's `role` column is live, filter on `role = 'owner'` instead (mind the shell quoting). Record the username in status.md so later sessions skip the lookup.
   - **Login smoke:** login, dashboard and coach. Login needs credentials, which an agent never enters. Use the owner's already-signed-in browser session if this session has one; otherwise the credential-free checks above stand, record "owner login smoke pending (MI-5)" as a pending-smoke note in status.md, and carry on (D40).
3. **Revert** is `git revert` of the MI-5 PR, which fails open: Flux prunes the policies within about 1 min. Revert on any caller error.

### 4 · MI-5a PR: xlearn ingress [I]

These are per-release values in `apps/xlearn-*.yaml`, using the **chart 0.3.0 NetworkPolicy knobs** from [mi-01](sprint-mi-01.md). Read the knob names in `charts/project/values.yaml`. mi-01's design (its knob table and `knob-netpol-*` fixtures) is:
- `networkPolicy.enabled: true`;
- `networkPolicy.from: null`, which drops 0.2.2's single Traefik rule;
- `networkPolicy.extraIngress:` raw ingress rules. A peer with only `podSelector` is the same-namespace source.

With those values the **rendered** policies are as follows. Selectors use namespace and `instance` only, **never `part-of`** (see task 1).

gateway (`xlearn/xlearn-gateway`):
```yaml
spec:
  podSelector: {matchLabels: {app.kubernetes.io/name: project, app.kubernetes.io/instance: xlearn-gateway}}
  policyTypes: [Ingress]
  ingress:
    - from:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: kube-system}}
          podSelector: {matchLabels: {app.kubernetes.io/name: traefik}}
        - podSelector: {}     # same namespace (xlearn): the in-cluster JWKS fetch of every service, future xlearn-judge included
      ports: [{port: 8080, protocol: TCP}]
```

identity :8081, curriculum :8082, practice :8083, review :8084, assessment :8085 and coach :8086 each get **same namespace only**:
```yaml
  ingress:
    - from: [{podSelector: {}}]      # xlearn namespace only: /sessions/*, /internal/* and user routes via the gateway
      ports: [{port: <containerPort>, protocol: TCP}]
```

- **No Traefik source on internal services.** Chart 0.2.2's default `networkPolicy.from` is Traefik (kubescope uses it). Set `from: null` so the 0.3.0 render of each internal release doesn't inherit it.
- `xlearn-runner` is admitted nowhere: no source selects it.
- The gateway's values are `from: null` plus **one** `extraIngress` rule whose `from` holds both peers (Traefik and `podSelector: {}`) on :8080. Internal services get `from: null` plus one `extraIngress` rule with a single `podSelector: {}` peer on their `containerPort`.
- The same PR adds 7 rows to `hack/expected-netpol.tsv` in the three-column format, `xlearn<TAB>xlearn-<svc><TAB>mi-03` (the chart names each policy after its release), and pastes the same rows into the embedded copy in `hack/host-verify.sh`. Run `hack/host-lint.sh` clean before opening the PR.
- It also bumps `podAnnotations` on `apps/xlearn-assessment.yaml` (`restarted-for: mi-5a`); that's one pod, which helm applies after the policy object exists.

**Before merge:**
- render every release with `helm template` at the base values and the new values, then diff. The **only** changes allowed are one added NetworkPolicy per release and the assessment annotation;
- check each rendered policy against the shapes above. **Pass:** the gateway admits exactly a Traefik peer and a same-namespace peer on :8080, and each internal service admits exactly a same-namespace peer on its port, with no other peers or ports and no `part-of`. If the merged chart splits the gateway's two peers into two rules, that's semantically equal: accept it and record it, but never accept an extra peer;
- `--dry-run=server` the rendered policies;
- run the selector proof: `get pods -n xlearn -l app.kubernetes.io/name=project` lists all 7 xlearn pods.

### 5 · MI-5a smoke: the JWKS path [H]

`JWKSVerifier` (`internal/platform/auth/jwks.go`) caches keys for 1 h and **tolerates a failed refresh while it holds stale keys**. Running pods would therefore look fine for up to an hour, and only a **fresh** pod proves the fence admits JWKS.

1. When the new `xlearn-assessment` pod is Ready, `GET https://projects.sujaykumar.dev/xlearn/api/u/<owner-username>`. Wait past the 15 s aggregation cache, or use a fresh request.
   - The gateway mints an assessment JWT, and the fresh assessment pod must fetch JWKS from `xlearn-gateway:8080` **through the new policy** to verify it.
   - Pass: the response includes the assessment-backed sections, and the assessment logs show **no** JWKS or 401 errors.
2. **review → identity and review → curriculum** (ADR-0016 workers): after the next sweep tick, the review logs show no identity/curriculum timeouts. After about 1 h, the hourly JWKS refetches in every service log no refresh failures.
3. **Login smoke:** login (Traefik → gateway → identity), dashboard, mistakes (review with a JWT), coach, and the SPA loads. As in task 3: use an already-signed-in browser session if this session has one; otherwise check that the SPA loads, add MI-5a to the "owner login smoke pending" note, and carry on. An agent never enters credentials.
4. **Landscape:** its xlearn cards still show healthy. If landscape probes xlearn pods in-cluster, that shows up here: record it and add it as a source in a follow-up, or accept it.
5. **On any 401 or timeout, revert** the MI-5a PR (fail-open).

### 6 · Verify [H]

- `hack/host-lint.sh` is clean: the embedded `expected-netpol.tsv` equals the file, now with the 9 new rows.
- `ssh vps 'bash -s -- --cluster --nats-stage=open' < hack/host-verify.sh` is green:
  - NetworkPolicy presence covers the 9 new names;
  - pods show no OOMKills and no restarts beyond the two annotation bumps;
  - the NATS stage read works.
- **Negative check:** `ssh vps "bash -s -- --cluster --nats-stage=open --netpol-file <(printf 'databases\tno-such-policy\tneg\n')" < hack/host-verify.sh` must **FAIL** `cluster.netpol`. The node's bash evaluates the `<(…)`, and mi-02's `--netpol-file` override reads it. That proves the presence check isn't vacuous. Paste both runs into the MI-5a PR.
- Repeat the login smoke once more (login, dashboard, coach) the same way: in a signed-in session, else it stays on the pending-smoke note.
- **The next fleet rollout is the full proof:** the next xlearn tag, likely v1.6.0, restarts every caller under both fences. That tag's release checklist already runs `host-verify` and the smoke. Note that in status.md.

### 7 · Record [X]

In an xlearn docs PR:
- **`docs/v2/status.md`:**
  - MI table rows **MI-5** ✅ and **MI-5a** ✅, with the PR numbers and dates. The MI-5 date is mi-06's N3 gate.
  - Sprint board mi-03 ✅.
  - Decisions log:
    - `part-of` isn't on pod labels, so selectors use namespace and `instance`;
    - the gateway admits same-namespace for JWKS;
    - the restart-PR proof method;
    - any caller found live that wasn't in config;
    - the Hostinger weekly image date read before the first restart, and the owner's username for the public-dashboard smoke.
  - **Pending-smoke notes:** "owner login smoke pending (MI-5, MI-5a)" if no signed-in session was available.
- **`docs/architecture/services.md`:** a "Network fences (v2)" subsection with the caller matrix. It should say:
  - **MI-5a is the internal-HTTP fence**: ADR-0006 and ADR-0016 as amended by ADR-0033 §12 row 7 / §14, still with no service tokens;
  - the gateway also serves **in-namespace JWKS**;
  - a regression fails open, and `host-verify --cluster` presence is the only detector (D34).
- **ADR-0033:** the BP2 sign-off (2026-09-24) should already have folded "plus same-namespace pods on :8080 for in-cluster JWKS" into §12 row 7 when it accepted the ADR. **Verify it.** Only if row 7 still reads "the gateway admits only Traefik", append a **dated one-line clarification** with that text. Don't rewrite the decision; accepted ADRs are append-only.

## Acceptance criteria

- [ ] MI-5 live: `databases/projects-pgstore-ingress` and `messaging/nats-ingress` exist, and the selector proof matches exactly the intended live callers.
- [ ] After the restart PR, review is back on PG and NATS, and nothing is lost: 7 NATS connections, consumers pending 0, outbox unsent 0, CNPG healthy.
- [ ] `host-verify`'s NATS stage read still works (node-local API-server proxy to 8222).
- [ ] MI-5a live: 7 `xlearn/xlearn-*` policies render as specified: the gateway admits exactly the Traefik and same-namespace peers on :8080, each internal service exactly the same-namespace peer, with no other peers and no `part-of`. The diff shows only NetworkPolicy additions plus the one annotation.
- [ ] A restarted service re-fetches JWKS through MI-5a and serves an authed call: fresh assessment serves the public dashboard with its sections and no 401s.
- [ ] No caller is blocked: the public dashboard, the review workers and the SPA through Traefik all work; login, the signed-in dashboard, mistakes and coach work in a signed-in session, or "owner login smoke pending" is recorded as a pending-smoke note.
- [ ] `host-lint.sh` is clean (the embedded `expected-netpol.tsv` carries the 9 new three-column rows), `host-verify --cluster` is green with NetworkPolicy presence covering them, and the `--netpol-file` negative check FAILs.
- [ ] The caller matrix is in both PR descriptions and `docs/architecture/services.md`. status.md has the MI-5 and MI-5a rows with dates.

## Release

**Infra PR(s) only, with no xlearn tag.** The `../infra` PRs merge in this order:
1. **MI-5.** This is the N3 precondition.
2. **The MI-5 restart** (a review `podAnnotations` bump). It's a separate PR because `databases`, `messaging` and `apps` are independent Kustomizations; this is why there are three PRs where the register listed two.
3. **MI-5a**, with the assessment `podAnnotations` bump.

Each stands alone, and none is folded into a tag (rollout §2.2). An xlearn docs PR records the status and the architecture note.

No step restarts CNPG or NATS. Each PR restarts at most one stateless xlearn pod. Those `podAnnotations` bumps are still restart-inducing steps, so the last Hostinger weekly image date is checked and recorded first (entry gate, rollout §2.2). No manual snapshot is needed: that's only for contract, erase or GA tags.

## Definition of Done

- All three infra PRs are merged and reconciled by Flux, with no hand `kubectl apply`.
- Both smokes pass on a fresh pod (their login parts in a signed-in session, or recorded as a pending-smoke note), and `host-verify --cluster` is green.
- The architecture note and ADR-0033 clarification are merged in xlearn.
- The status is updated (this file and [`../status.md`](../status.md)), and both repos' `main` is synced.

## Risks / watch-outs

- **A missed caller is blocked silently.** identity's relay would only log "not ensured" while its outbox grows. Mitigations:
  - build the list from live config **and** live connections (`/connz`, `pg_stat_activity`);
  - forward-declare identity, judge and coach;
  - run the selector proof before merging.
- **"Gateway: Traefik only" would break every service's JWKS fetch on its next pod restart.** Running pods hide it for up to 1 h (stale-key tolerance). Hence the same-namespace source and the fresh-pod smoke.
- **`part-of` isn't a pod label.** A policy selecting `app.kubernetes.io/part-of: xlearn` matches nothing and blocks every caller. Select by namespace or `instance` only (verified live 2026-09-24).
- **Established connections outlive a new policy.** A wrong selector shows on the next reconnect or rollout, which is why the fresh-pod restarts and the note to re-run `host-verify` plus the smoke after the next fleet rollout.
- **Traefik labels might differ from the assumption.** They were read live: `app.kubernetes.io/name=traefik` in `kube-system`. Re-read them before merging in case k3s's Traefik chart changed.
- **The CNPG operator needs `:8000` from `cnpg-system`.** If the Cluster status degrades or the operator logs connection errors, revert. The instance↔instance rule is forward-declared for when `instances` goes above 1.
- **The API-server proxy to 8222 relies on node-local admission.** It's verified in task 3. If it breaks, the fallback is an `ipBlock` for the node's `cni0` address on 8222 only. Never admit a pod.
- **Memory:** restart one pod at a time. The margin before MI-11a is about 0 (ADR-0035 §5), so never trigger a fleet restart for a smoke.
- **The chart 0.3.0 default source** (Traefik) must not leak into internal services' policies. That would admit the internet-facing proxy to `/sessions/*` and `/internal/*`, leaving them one IngressRoute mistake away from exposure. Check the render diff.
