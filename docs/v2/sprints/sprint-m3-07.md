# Sprint m3-07 — M3-1 on prod: evalpack v1.0.0, judge ACL, tag v1.13.0, judge HelmRelease (MI-13)

> **Milestone:** M3 — judge plus code grader (**M3-1 on prod**; rollout step **MI-13**, sprint ids `mi-NN` ≠ rollout steps `MI-N`) · **Track:** product (release) · **Order:** 46
> **Prereqs:** [m3-14](sprint-m3-14.md) (and through it [m3-05](sprint-m3-05.md), [m3-06](sprint-m3-06.md)) · [m3-02](sprint-m3-02.md) (evalpack pipeline) · [mi-06](sprint-mi-06.md) (N1–N3) · [l-01](sprint-l-01.md) (N3 ≥ 24 h re-check; erase-consumer pattern) · [mi-03](sprint-mi-03.md) (MI-5, MI-5a) · [mi-07](sprint-mi-07.md) (MI-9) · also [mi-01](sprint-mi-01.md) (chart 0.3.0), [spk-02](sprint-spk-02.md) (image-volume result); preferred: [mi-10](sprint-mi-10.md) (runner dark)
> **Unblocks:** [m3-08](sprint-m3-08.md) · [m3-09](sprint-m3-09.md) · [mi-11](sprint-mi-11.md) · [mi-12](sprint-mi-12.md)
> **Release action:** **evalpack `v1.0.0`** → infra PR (evalpack ImagePolicy) → infra PR (judge NATS ACL, **before** the tag) → **tag `v1.13.0`** (indicative: the next free minor at tag time) → infra PRs (judge DB 4-step, then judge HelmRelease, **after** the tag). Infra PRs are their own tasks, never folded into the tag.
> **Calendar:** mid-November (after m3-14 merges and ≥ 1 pack is stamped)
> **Execute with:** [`../prompts/prompt-m3-07.md`](../prompts/prompt-m3-07.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pre-flight: gates, peers, identity-vs-`XLEARN_JUDGE` startup check, memory sum | H | ⬜ |
| 2 | Tag evalpack `v1.0.0` (stamped items so far) | E | ⬜ |
| 3 | evalpack ImagePolicy (image before policy) | I | ⬜ |
| 4 | judge nkey + NATS ACL PR (before the tag) | I | ⬜ |
| 5 | Tag `v1.13.0` (release checklist) — first `xlearn-judge` image | X | ⬜ |
| 6 | `xlearn-judge` GHCR package pullable anonymously | O | ⬜ |
| 7 | judge DB 4-step PR (role, schema, secrets) | I | ⬜ |
| 8 | judge HelmRelease PR: image automation, runner bearer, pack volume, born default-deny egress | I | ⬜ |
| 9 | Verify (Ready, evaluable, nkey, erase consumer, policies, memory sum, still dark) | H | ⬜ |
| 10 | Record (status.md: MI-13, tag/floor, evalpack stream, flags, NATS, content) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, MI row MI-13, M3 milestone row, milestone → tag → floor → snapshot, evalpack stream row).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-3** chart 0.3.0 merged ([mi-01](sprint-mi-01.md)): `networkPolicy.{from: null, extraIngress, egress}`, `probes.readinessPath`, `automountServiceAccountToken`
- [ ] **MI-5 and MI-5a** live ([mi-03](sprint-mi-03.md)): `databases/projects-pgstore-ingress` and `messaging/nats-ingress` forward-declare `xlearn-judge`; the gateway admits same-namespace pods on :8080 (the JWKS fetch)
- [ ] **MI-7 N3** applied ([mi-06](sprint-mi-06.md)) **and its ≥ 24 h re-check recorded** in status.md NATS rows ([l-01](sprint-l-01.md) task 1)
- [ ] **MI-9** done ([mi-07](sprint-mi-07.md)): private `xlearn-evalpack` + CI with the anonymous-GET probe; `xlearn-evalpack-pull` in `xlearn` and `flux-system`; ImageRepository `xlearn-evalpack` Ready (no policy yet)
- [ ] [m3-05](sprint-m3-05.md), [m3-06](sprint-m3-06.md) and [m3-14](sprint-m3-14.md) merged; the 8th `deploy.yml` job (`xlearn-judge`) is on `main`
- [ ] **≥ 1 pilot pack stamped by the owner** (ev-packs-14 in progress) and green in the evalpack CI ([m3-02](sprint-m3-02.md))
- [ ] [spk-02](sprint-spk-02.md)'s image-volume result recorded: **GO** (image volume), or the initContainer fallback recipe chosen
- [ ] [l-01](sprint-l-01.md) live (`v1.11.0`): the erase-consumer pattern and identity's ack path (identity will expect judge's ack from this tag)
- [ ] Runner dark on prod ([mi-10](sprint-mi-10.md)) — **preferred, not required** for judge dark. The judge-side runner bearer secret is created here either way (task 8)
- [ ] **Parallel sessions:** no open peer PR touches `infrastructure/messaging/release.yaml`, `apps/image-automation.yaml`, `infrastructure/database/cluster/` or `apps/xlearn-judge.yaml` (`gh pr list -R sujaykumarsuman/infra`); peers' xlearn tags checked (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Not a gate here: the M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)). This sprint **delivers its "MI-13" line**; the list gates [m3-11](sprint-m3-11.md) and [m3-13](sprint-m3-13.md)

## Goal

Put judge on production **dark**, in the only valid order ([ADR-0034 §1.4–§1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)):
**pack image → pack policy → judge ACL → tag (the first `xlearn-judge` image) → judge infra PRs**. At the end, `xlearn-judge`
is Ready in `xlearn` with the pack mounted read-only, authenticated to NATS with its own nkey, its `XLEARN_IDENTITY`
erase consumer bound, **born default-deny egress** (DNS, PG, NATS, runner, gateway :8080 for JWKS, identity :8081 for
the erase re-verify), admitting ingress
only from the gateway and practice — and **zero learner exposure**: the gateway's `JUDGE_BASE_URL` stays unset until
[m3-13](sprint-m3-13.md).

## Scope

**In**
- evalpack `v1.0.0` (the first stamped items) and its ImagePolicy `>=1.0.0 <2.0.0`, digest-pinned.
- judge's nkey user + fine ACL (and identity's ack durable on `XLEARN_JUDGE`) in the `messaging` values, with its seed.
- Tag `v1.13.0` — M3-1 judge dark, born with its erase consumer ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline)).
- **MI-13:** the [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) 4-step (role `xlearn_judge`, schema `judge`,
  secrets), `apps/xlearn-judge.yaml` (port 8087, the evalpack image volume, kill-switch env, NATS seed, runner bearer,
  readiness on `/readyz`, default-deny egress, gateway/practice-only ingress), ImageRepository + ImagePolicy
  `xlearn-judge`, and the `host-verify` expected-NetworkPolicy list.

**Out**
- `JUDGE_BASE_URL` on the gateway (and practice) — stays **unset**; set by [m3-13](sprint-m3-13.md) after `v1.14.0`.
- judge TCP 443 egress (M4) → [mi-12](sprint-mi-12.md). (judge → identity :8081 is **born here** for the erase re-verify;
  mi-12 then adds **only** TCP 443 and verifies :8081 is already present.)
- The `xlearn` egress policies for every other release, including practice → judge :8087 → [mi-11](sprint-mi-11.md).
- practice's `XLEARN_JUDGE` durable and its ACL PR → [m3-08](sprint-m3-08.md).
- The definitive TL re-gate of the packs against `runner-v1.0.0` × the prod multipliers → [m3-13](sprint-m3-13.md).
- The runner itself → [mi-10](sprint-mi-10.md).

## Tasks

### 1 · Pre-flight [H]

- Confirm every entry gate; paste the evidence in the first infra PR's description.
- **identity must not block on `XLEARN_JUDGE`.** From this tag identity's topology declares an ack durable on
  `XLEARN_JUDGE` (`xlearn.judge.account_erased`, [m3-05](sprint-m3-05.md)), but that stream only exists once judge's
  pod first starts (task 8, after the tag). `NatsConsumer.Subscribe` retries until the stream exists
  (`internal/platform/events/consumer.go`), and v1 mains call `Subscribe` **before** `ListenAndServe`
  (`cmd/review/main.go`). Prove it in compose at the candidate commit: bring the stack up **without judge** → identity
  must be Ready and serving (a log line such as "waiting for stream XLEARN_JUDGE" is fine); then start judge → identity
  binds the durable within its backoff. **If identity blocks**, stop: fix it first in a small merge-only X PR (start that
  consumer in the background) — otherwise the `v1.13.0` identity rollout stalls until task 8 lands (the old pod keeps
  serving and `apps` goes not-Ready under `wait: true`).
- **Memory sum** ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):
  `ssh vps 'bash -s -- --cluster --nats-stage=n3' < ../infra/hack/host-verify.sh` green (N4 is [mi-11](sprint-mi-11.md)'s,
  which runs after this sprint). Add judge's **256 Mi limit** and its **256 Mi rollout surge** by hand: the sum
  must stay ≤ capacity − 0.5 GiB (≈ 0.9 GiB of margin after MI-11a with the runner counted). Record the numbers.

### 2 · Tag evalpack `v1.0.0` [E]

In `../xlearn-evalpack` (private; never copy anything from it into xlearn):
- The stamped items so far (**≥ 1**; the 14 pilot packs join as they land in later `v1.x` tags). Unstamped items stay
  out (the stamp gate, [m3-02](sprint-m3-02.md)).
- With the sibling `../xlearn` checked out **at the commit you will tag `v1.13.0`**, run `make packcheck` (it builds
  packlint from that checkout, `go -C ../xlearn run ./cmd/packlint`, [m3-02](sprint-m3-02.md)) and
  `(cd ../xlearn && go run ./cmd/packlint check --public . --pack ../xlearn-evalpack)`
  ([m3-01](sprint-m3-01.md)): every item's `accepts_contract_hashes` holds the live `contract_hash`. If a content change merges before
  the tag, re-run; a mismatch leaves that item `spec_mismatch` (judge stays Ready, the item is just not evaluable).
- **TLs:** stay flagged **provisional** unless the `runner-v1.0.0` baselines ([m3-15](sprint-m3-15.md)) × the
  [mi-10](sprint-mi-10.md) prod multipliers already exist; if they do, re-run the TL gate first. The definitive re-gate is
  [m3-13](sprint-m3-13.md)'s.
- Tag **`v1.0.0`** → CI builds `ghcr.io/sujaykumarsuman/xlearn-evalpack:1.0.0` and runs the **anonymous manifest GET
  probe** (401/403 or the build fails). Record the **digest**.
- Confirm: pullable with the machine-user PAT, not anonymously; `ImageRepository xlearn-evalpack` lists `1.0.0`
  (`k3s kubectl get imagerepository xlearn-evalpack -n flux-system -o jsonpath='{.status.lastScanResult.latestTags}'`).

### 3 · evalpack ImagePolicy [I] (after `v1.0.0` exists)

`../infra/apps/image-automation.yaml`, beside the ImageRepository from [mi-07](sprint-mi-07.md):

```yaml
apiVersion: image.toolkit.fluxcd.io/v1
kind: ImagePolicy
metadata:
  name: xlearn-evalpack
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: xlearn-evalpack
  filterTags:
    pattern: '^\d+\.\d+\.\d+$'
  policy:
    semver:
      range: ">=1.0.0 <2.0.0"     # a pack major is a contract break (ADR-0034 §1.5)
  digestReflectionPolicy: IfNotPresent   # digest-pinned; tags are never re-pushed
```

- Check `digestReflectionPolicy` against the cluster's Flux CRD before writing it (as [mi-10](sprint-mi-10.md) does).
- Extend the file's header comment: the evalpack marker sits on judge's image-volume `reference:` (task 8), so the shared
  IUA bumps it and **each pack tag restarts judge**.
- Verify after merge: the policy is Ready with latest `1.0.0@sha256:<digest>`. Nothing references the marker yet, so
  nothing moves.

### 4 · judge nkey + NATS ACL PR [I] (merged **before** the tag)

[ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) standing rule:
a new stream or durable needs its ACL PR merged before the consuming service's tag.
- **Key** (the [mi-06](sprint-mi-06.md) procedure): `nk -gen user > <scratch>/judge.nk` (mode 600, never printed),
  `nk -inkey <scratch>/judge.nk -pubout` → `U…`. Write `../infra/apps/secrets/xlearn-nats-judge.enc.yaml` (Secret
  `xlearn-nats-judge`, ns `xlearn`, key `seed`), `sops -e -i` it, delete the plaintext. A Secret with no consumer yet is
  harmless.
- **Render:** `make nats-acl-render` on `main` (m3-05/m3-06/m3-14 merged = the `v1.13.0` candidate). The diff against
  the live `infrastructure/messaging/release.yaml` block must be **exactly**:
  - a new **judge** user: publish `xlearn.judge.>`, `$JS.API.INFO`, `$JS.API.STREAM.{CREATE,UPDATE,INFO}.XLEARN_JUDGE`;
    its erase durable on `XLEARN_IDENTITY` (`CONSUMER.CREATE/INFO/MSG.NEXT` + `$JS.ACK`); subscribe `_INBOX_judge.>`;
    never DELETE/PURGE;
  - **identity**'s block gaining its ack durable on `XLEARN_JUDGE` (`xlearn.judge.account_erased`).

  Anything else in the diff means another sprint's ACL change is unmerged: stop and explain.
- Paste under `config.merge.authorization.users` (public key in plaintext — **no SOPS in `messaging`**). Do **not** bump
  the pod-template annotation: since N1 an authorization change is a **reload**, not a restart.
- **NetworkPolicy standing rule:** confirm read-only that `messaging/nats-ingress` and `databases/projects-pgstore-ingress`
  already list `xlearn-judge` (MI-5 forward-declared it): `ssh vps 'k3s kubectl get networkpolicy -n messaging -o yaml'`.
  If not, add it in this PR.
- **Verify after merge:** `/varz` `config_load_time` moved while `nats-0`'s start time did not; `/connz?auth=true`
  shows every existing connection on its own nkey and none on `legacy`; `host-verify --cluster --nats-stage=n3`
  green; no permission-violation ERROR in `k3s kubectl -n xlearn logs deploy/xlearn-<svc> --since=10m` for the six services.

### 5 · Tag `v1.13.0` [X]

Run the release checklist below. GitHub release title **`v1.13.0 — v2 build · M3-1 judge dark`**; notes: "first
`xlearn-judge` image (8th `deploy.yml` job); judge is deployed by the infra PR that follows the tag; identity expects
judge's erase ack from this tag; no learner-visible change". Anything else merged since the last tag rides along (e.g.
[l-03](sprint-l-03.md), inert while `SIGNUP_MODE=closed`).
- After the tag: the 7 existing releases roll to `1.13.0`; `ghcr.io/sujaykumarsuman/xlearn-judge:1.13.0` exists; judge
  is **not** deployed yet (no HelmRelease, no policy).
- **Gate state after:** judge dark. **Rollback floor after:** unchanged (judge's schema is new and additive).
  **Snapshot:** not required (not a contract, erase or GA tag).

### 6 · `xlearn-judge` GHCR package pullable [O]

A first push creates a new GHCR package. The cluster pulls every `xlearn-*` image without a pull secret, so confirm an
**anonymous** read works like the other seven: `crane ls ghcr.io/sujaykumarsuman/xlearn-judge` lists `1.13.0`. If it
returns 401/403, the **owner** sets the package to public in GitHub (it holds only public code, [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md);
the pack is a separate private image) and links it to the repo. **Never** do this to `xlearn-evalpack`. Task 8 waits for
this, or judge would sit in ImagePullBackOff and its ImageRepository would fail to scan.

### 7 · judge DB 4-step PR [I] (after the tag)

The infra database README pattern ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), as its own PR **before**
the HelmRelease: `databases` and `apps` are separate Kustomizations with no apply order, and a judge pod that starts
before its schema exists would crash-loop and hold `apps` not-Ready under `wait: true`.

| File | Change |
|---|---|
| `infrastructure/database/cluster/pg-xlearn-judge.enc.yaml` | Secret `pg-xlearn-judge` (ns `databases`, label `cnpg.io/reload: "true"`, `kubernetes.io/basic-auth`, `username: xlearn_judge`, a generated password) — SOPS-encrypted in place |
| `infrastructure/database/cluster/cluster.yaml` | managed role `xlearn_judge` (`login: true`, `connectionLimit: -1` like its neighbours; [mi-11](sprint-mi-11.md) later raises every xlearn role, judge included, to `20` for L21) |
| `infrastructure/database/cluster/xlearn-database.yaml` | schema `judge`, owner `xlearn_judge` |
| `apps/secrets/xlearn-judge-db.enc.yaml` | Secret `xlearn-judge-db` (ns `xlearn`): `PGUSER: xlearn_judge`, `PGPASSWORD` = the same password |

- Generate the password offline and pipe it into both files; never echo it.
- Verify (read-only): the role exists, `judge` is owned by `xlearn_judge`
  (`k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select nspowner::regrole from pg_namespace where nspname='judge'"`),
  "Cluster in healthy state".
- *Fallback:* one combined PR with task 8; the brief crash-loop self-heals ([ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)). Accepted, not preferred.

### 8 · judge HelmRelease PR [I] (after tasks 6 and 7)

**Runner bearer (owned here).** `apps/secrets/xlearn-judge-runner-auth.enc.yaml`: Secret `xlearn-judge-runner-auth`,
key `token`, the **same value** as the runner's `runner/secrets/runner-auth.enc.yaml`:
- if [mi-10](sprint-mi-10.md) ran, `sops -d --extract '["stringData"]["token"]' runner/secrets/runner-auth.enc.yaml` piped
  straight into the new Secret and `sops -e -i`;
- otherwise generate offline (`openssl rand -base64 48`) and note in status.md that mi-10 must re-encrypt **this** value.

Never echoed. A missing Secret would stop judge at `CreateContainerConfigError` on `apps`' wait path.

**Image automation** (image before policy — the image exists since task 5): in `apps/image-automation.yaml`,
ImageRepository `xlearn-judge` (`ghcr.io/sujaykumarsuman/xlearn-judge`, 1 m) and ImagePolicy `xlearn-judge`
`range: ">=1.0.0 <2.0.0"`, exactly like the fleet's.

**`apps/xlearn-judge.yaml`** (chart `./charts/project` 0.3.0, same `chart`/`sourceRef` block as the other releases).
Env **names must match what m3-05/m3-06 merged** — read `internal/judge/config.go` first; the table uses the planned names.

| Values key | Setting |
|---|---|
| `partOf` / `component` | `xlearn` / `judge` |
| `image` | `repository: ghcr.io/sujaykumarsuman/xlearn-judge`, `tag: 1.13.0 # {"$imagepolicy": "flux-system:xlearn-judge:tag"}` |
| `imagePullSecrets` | `[{name: xlearn-evalpack-pull}]` (for the pack volume; the judge image itself is public) |
| `containerPort` / `service.port` / `route.enabled` | `8087` / `8087` / `false` (ClusterIP only) |
| `automountServiceAccountToken` | `false` (M4 adds a projected token with automount still off, [mi-12](sprint-mi-12.md)) |
| `probes` | `path: /healthz`, `readinessPath: /readyz`, `initialDelaySeconds: 10` (migrations + pack verify) |
| `resources` | requests `50m`/`128Mi`, limits `500m`/`256Mi` ([t4 §9.3](../research/t4-judge-contract.md#93-throughput-and-memory-inferred-t3-measures); ADR-0035 §5) |
| `envFrom` | `[{secretRef: {name: xlearn-judge-db}}]` |
| `env` | `LOG_LEVEL=info`; `PGHOST=projects-pgstore-rw.databases.svc.cluster.local`, `PGPORT=5432`, `PGDATABASE=xlearndb`, `PGSEARCHPATH=judge`; `PG_MAX_CONNS=8` (L21); `JWKS_URL=http://xlearn-gateway.xlearn.svc.cluster.local:8080/.well-known/jwks.json`, `JWT_AUDIENCE=judge`, `JWT_ISSUER=xlearn-gateway`; `NATS_URL=nats://nats.messaging.svc.cluster.local:4222`, `NATS_NKEY_SEED_FILE=/var/run/secrets/nats/seed`, `NATS_INBOX_PREFIX=_INBOX_judge`; `RELAY_INTERVAL=1s`; `EVALPACK_DIR=/evalpack`; `JUDGE_SCRATCH_DIR=/scratch`; `IDENTITY_BASE_URL=http://xlearn-identity.xlearn.svc.cluster.local:8081` (the erase re-verify, m3-05); `RUNNER_BASE_URL=http://xlearn-runner.xlearn-runner.svc.cluster.local:8090`, `RUNNER_TOKEN_FILE=/var/run/secrets/runner/token` (m3-06); **`JUDGE_ADMISSION=open` set explicitly** (the L15 kill switch, m3-05: `closed` → 503 for new work while the queue drains; R-a) |
| `extraVolumes` | `evalpack` **image volume**: `image: {reference: ghcr.io/sujaykumarsuman/xlearn-evalpack:1.0.0@sha256:<digest> # {"$imagepolicy": "flux-system:xlearn-evalpack"}, pullPolicy: IfNotPresent}` ([t1 §3.3](../research/t1-content-data-model.md)); `nats-seed` (Secret `xlearn-nats-judge`, `defaultMode: 0440`, `items: [{key: seed, path: seed}]`); `runner-auth` (Secret `xlearn-judge-runner-auth`, `0440`, `items: [{key: token, path: token}]`); `scratch` (`emptyDir: {sizeLimit: 256Mi}` — the root FS is read-only and m3-06 materializes each case input under `JUDGE_SCRATCH_DIR`; without it every code job fails as infra `setup`, [t4 §9.3](../research/t4-judge-contract.md#93-throughput-and-memory-inferred-t3-measures)) |
| `extraVolumeMounts` | `/evalpack` (ro), `/var/run/secrets/nats` (ro), `/var/run/secrets/runner` (ro), `/scratch` (rw) |
| `networkPolicy` | below |

If [spk-02](sprint-spk-02.md) recorded the **fallback**, replace the image volume with its initContainer recipe (chart
0.3.0 `initContainers` into an `emptyDir` at `/evalpack`), keeping the `$imagepolicy` marker on the pack reference;
`EVALPACK_DIR` hides the difference from judge's code.

**NetworkPolicy — born default-deny egress** (chart 0.3.0 knobs, [mi-01](sprint-mi-01.md); selectors by namespace and
`app.kubernetes.io/instance` only, never `part-of`, [mi-03](sprint-mi-03.md) task 1):

```yaml
    networkPolicy:
      enabled: true
      from: null                         # no Traefik source: judge has no route
      extraIngress:                      # ADR-0033 §12 row 10: gateway (learner API) and practice (/internal/*) only
        - from:
            - podSelector: {matchLabels: {app.kubernetes.io/instance: xlearn-gateway}}
            - podSelector: {matchLabels: {app.kubernetes.io/instance: xlearn-practice}}
          ports: [{port: 8087, protocol: TCP}]
      egress:                            # a list ⇒ Egress in policyTypes; everything not listed is denied
        - to: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: kube-system}},
                podSelector: {matchLabels: {k8s-app: kube-dns}}}]          # read CoreDNS's labels live first
          ports: [{port: 53, protocol: UDP}, {port: 53, protocol: TCP}]
        - to: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: databases}},
                podSelector: {matchLabels: {cnpg.io/cluster: projects-pgstore}}}]
          ports: [{port: 5432, protocol: TCP}]
        - to: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: messaging}},
                podSelector: {matchLabels: {app.kubernetes.io/name: nats, app.kubernetes.io/instance: nats}}}]
          ports: [{port: 4222, protocol: TCP}]
        - to: [{namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: xlearn-runner}},
                podSelector: {matchLabels: {app.kubernetes.io/instance: xlearn-runner}}}]
          ports: [{port: 8090, protocol: TCP}]
        - to: [{podSelector: {matchLabels: {app.kubernetes.io/instance: xlearn-gateway}}}]   # JWKS for aud=judge
          ports: [{port: 8080, protocol: TCP}]
        - to: [{podSelector: {matchLabels: {app.kubernetes.io/instance: xlearn-identity}}}]  # erase re-verify (m3-05)
          ports: [{port: 8081, protocol: TCP}]
```

- The runner side already admits exactly this pod (`judge-to-runner`: ns `xlearn` AND instance `xlearn-judge`, TCP 8090,
  [mi-14](sprint-mi-14.md)). The gateway's MI-5a same-namespace source admits judge's JWKS fetch. PG and NATS ingress
  were forward-declared by MI-5. Kubelet probes and the image-volume pull are node-local and unaffected.
- **Why the gateway and identity are on the list:** rollout §2 MI-13 says "DNS, PG, NATS, runner only", but
  - judge verifies `aud=judge` JWTs against the gateway's JWKS ([m3-05](sprint-m3-05.md) task 5): without :8080 every
    learner call would 401 once `JUDGE_BASE_URL` is set;
  - judge's erase consumer re-verifies each request with identity `GET /internal/erasures/{id}` before acting
    ([m3-05](sprint-m3-05.md) task 4 hand-off): without :8081 it fails closed and dead-letters every erasure.
    identity's MI-5a ingress already admits the `xlearn` namespace. This moves "judge → identity :8081" forward from
    [mi-12](sprint-mi-12.md) (which planned it for M4's account read, m4-02's `/internal/accounts/{id}`, served by the
    same rule) to M3-1. From here on: [mi-11](sprint-mi-11.md) leaves judge's policy as set here (identity :8081
    included); mi-12 adds **only** TCP 443 and **verifies** :8081 is present rather than adding it; rollout §12's
    "M4 adds 443 + judge → identity :8081" reads "M4 adds 443". Record this in the decisions log (task 10).
- **`host-verify` expected policies:** add `xlearn	xlearn-judge` to `hack/expected-netpol.tsv` and its embedded copy
  in `hack/host-verify.sh` (`host-lint.sh` checks the copies match).

**Before merge:** `helm template` the release with these values and read the Deployment and NetworkPolicy; pipe both to
`ssh vps 'k3s kubectl apply --dry-run=server -f -'` (persists nothing); selector proof —
`k3s kubectl get pods -n xlearn -l 'app.kubernetes.io/instance in (xlearn-gateway,xlearn-practice)' -o name` lists both,
and the egress targets match `projects-pgstore-1`, `nats-0` and (if [mi-10](sprint-mi-10.md) ran) the runner pod.
**Merge →** Flux applies; judge migrates, verifies the pack against `/manifest.json`, and turns Ready.

### 9 · Verify [H]

Read-only, over `ssh vps`:
- `k3s kubectl -n xlearn get deploy xlearn-judge` 1/1 Ready on `1.13.0`; ≤ 1 restart; `apps` Kustomization Ready.
- `k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin status`: pack `1.0.0` + the digest, **N evaluable =
  the stamped items** (status `ok`), kill switch off, runner lane reachable (or "unreachable" if mi-10 hasn't run — expected).
- **NATS:** `/connz?auth=true` shows judge's public key and still no `legacy`; `/jsz?streams=true&consumers=true`:
  `XLEARN_JUDGE` exists (512 MiB, 14 d, Old), judge's erase durable on `XLEARN_IDENTITY` is bound with pending 0, and
  identity's ack durable on `XLEARN_JUDGE` is bound.
- **Erase path:** judge's erase durable is `DeliverAll`, so on first start it replays every past erasure request, re-verifies
  each with identity (:8081 through the new egress) and acks with tombstones only; `judge.event_dead_letter` is empty and
  identity shows judge's acks on those requests (read-only).
- **PG:** `pg_stat_activity` shows `xlearn_judge` sessions ≤ 8.
- **JWKS:** judge logs show the gateway JWKS fetched, no errors.
- **Memory sum:** `host-verify --cluster` green with judge counted (and the runner, if live); no OOMKill.
- **Policies:** `host-verify` NetworkPolicy check green including `xlearn/xlearn-judge`; the rendered egress list matches
  the plan.
- **Still dark:** `JUDGE_BASE_URL` absent from `apps/xlearn-gateway.yaml` and from the live gateway env
  (`k3s kubectl -n xlearn get deploy xlearn-gateway -o jsonpath='{..env}'`); the SPA shows no judge surface.
- Smoke: login, the dashboard and coach.

### 10 · Record [X]

In an xlearn docs PR (merge only): [`../status.md`](../status.md) — MI row **MI-13 ✅** (PR links); milestone → tag →
floor → snapshot row **`M3-1 → v1.13.0 → floor unchanged → snapshot n/a`**; **evalpack stream** row (`1.0.0`, digest,
`validated_against`, item count, TL provisional flag); **content status** (items stamped per tier); NATS rows (judge nkey
live, ACL PR, reload date); the runner-auth secret's provenance (who generated it, when — never the value); **flag
inventory**: judge's kill-switch env (a permanent kill switch, L15) and `JUDGE_BASE_URL` unset; the memory-sum numbers;
decisions log: the gateway :8080 (JWKS) and identity :8081 (erase re-verify) egress additions to MI-13 (mi-11 leaves judge's policy as is; mi-12 then adds only TCP 443 and verifies :8081 is present; rollout §12's M4 line reads "443" only), and the two-PR DB/HelmRelease order.

## Acceptance criteria

- [ ] judge **Ready on prod with the pack mounted** (`judge admin status` shows N evaluable); **0 learner exposure** (gateway `JUDGE_BASE_URL` unset; ingress from gateway and practice only).
- [ ] **Order respected** and recorded: pack image → pack policy → ACL → tag → DB 4-step → HelmRelease.
- [ ] judge connects with its **own nkey**; no `legacy` connection; its erase consumer and identity's ack durable are bound; `XLEARN_JUDGE` exists.
- [ ] judge is **born default-deny egress** (DNS, PG, NATS, runner, gateway :8080, identity :8081); `hack/expected-netpol.tsv` updated; `host-verify --cluster` green incl. the memory sum.
- [ ] `v1.13.0` tagged and verified per the release checklist; evalpack `v1.0.0` and its ImagePolicy live.

## Release

**Tag `v1.13.0`** (indicative; next free minor at tag time, [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md)) +
**evalpack `v1.0.0`** + **infra PRs** (ImagePolicy after the pack; ACL before the tag; DB 4-step and HelmRelease after it).
Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:**
- *ACL PRs:* judge's user and identity's new durable on `XLEARN_JUDGE` → task 4, merged before the tag.
- *New service, image before policy:* `xlearn-judge`'s ImageRepository/ImagePolicy land in task 8, **after** the tag;
  the "every `xlearn-*` ImagePolicy's latest equals the tag" line is read twice — for the 7 existing policies right after
  the tag, and for `xlearn-judge` after task 8.
- *Contract / erase / GA:* none (expand only; judge's schema is new) → no snapshot required.
- *M6:* n/a.
- *NetworkPolicy standing rule:* the only new callers are judge's own (→ PG, NATS, runner, gateway, identity), and judge's pod does
  not exist until task 8, which carries its policy (born default-deny); the callee side was forward-declared by MI-5,
  MI-5a and MI-4. The gateway → judge and practice → judge calls start only in `v1.14.0`, and their ingress is already
  in task 8's policy.
- *Extra after-tag reads:* identity logs show its `XLEARN_JUDGE` ack consumer waiting (not failing) until task 8, then bound.

## Definition of Done

evalpack `v1.0.0` tagged with the probe green · all infra PRs merged in order and verified · `v1.13.0` tagged, deployed by
Flux (no hand `kubectl`) and verified by the checklist · judge Ready dark with the pack mounted · acceptance criteria met ·
statuses updated (this file + [`../status.md`](../status.md): board, MI-13, M3, tag → floor, evalpack stream, flags,
NATS, content) · notable calls in the decisions log.

## Risks / watch-outs

- **Brief crash-loop until the schema exists** (self-heals, as v1) — avoided by merging the DB 4-step first (task 7).
- **Pack pull failure:** if the pack image can't be pulled, the old pod keeps serving but a **first** start fails —
  check the pull secret (`xlearn-evalpack-pull`, PAT expiry in status.md) before task 8. A bad or unsupported pack never
  crash-loops judge: it is Ready with 0 evaluable (m3-05).
- **identity blocking on the missing `XLEARN_JUDGE` stream** would stall the identity rollout at the tag — task 1's
  compose check; if it can't be fixed in time, shorten the window by merging task 7 and task 8 within minutes of the tag.
- **An erase requested between the tag and judge Ready** waits for judge's ack; judge's durable is `DeliverAll`, so it
  completes once judge starts. Erase is tester-only and rare; just don't rehearse one in that window.
- **`xlearn-judge` GHCR package created private** → ImagePullBackOff and a failed ImageRepository scan (task 6).
- **`contract_hash` drift** between the pack's `validated_against` and the tagged commit leaves items `spec_mismatch`
  (safe, not evaluable); re-run packlint at the tag commit and cut a pack patch if needed.
- **Each pack tag restarts judge** (its marker is on the pod spec) — harmless while dark; batch pack tags later.
- **Runner bearer mismatch** between the two Secret copies → the runner answers 401, judge re-queues as `saturated`; dark,
  so invisible until [m3-13](sprint-m3-13.md)'s dogfood. Record provenance; rotate as a pair of PRs, runner first.
- **Rollback:** R-a the kill-switch env; remove judge by reverting the HelmRelease PR (Flux prunes it; the schema and
  NATS user stay, harmless); R-b of the fleet below `1.13.0` only drops identity's expectation of judge's ack.
</content>
</invoke>
