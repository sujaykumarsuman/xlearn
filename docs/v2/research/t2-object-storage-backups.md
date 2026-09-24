> **T2 research appendix.** Method: four parallel research slices (in-cluster S3 options and MinIO's 2026 status; external S3 providers and pricing; CNPG backup design; app integration) were synthesized into one draft. The draft then faced two adversarial critiques: resilience and security (6/10, 9 majors) and solo cost and YAGNI (7/10, 7 majors). This is the revised final.
>
> **Status:** settled with the owner 2026-09-24 (**D11–D13**, §10). **The owner deferred off-node backups.** The provider choices below (B2 Amsterdam, AWS Mumbai second copy, locks, the 30-day window) were **not adopted**. §3 stays as the ready-made design for when backups land, targeted at the owner's second VPS or a provider chosen then. The no-object-store findings (§1, §2, §4) **were adopted**. Decided in [ADR-0028](../../adr/0028-object-storage-and-backups.md) (Proposed).

# T2: object storage and off-node backups (research + the ready-to-use backup design)

Repo prefixes: **X/** = this repo, **I/** = the sibling `../infra` repo. "(inferred)" marks my own reasoning. Every external claim that carries weight was checked on 2026-09-24; sources are at the end. This version builds on the draft and fixes every finding from both critics (§12).

## 1. Recommendation in one paragraph

**Run no object store in the cluster, and add no app blob store in v2.0.** Canvas scenes (final and draft) and interview transcripts stay **inline in Postgres permanently** (`bytea`, lz4 TOAST). They ride the one critical backup, stay consistent with their rows at every restore point, and erasing them is SQL only.

v2.0 does need off-node storage, for three things:

1. **CNPG PITR backups** go to **Backblaze B2 EU Central (Amsterdam)**. The node is in Mumbai. They use the **Barman Cloud Plugin v0.15.0**, installed as HelmRelease chart `plugin-barman-cloud` 0.8.0.
2. **At the M3 gate**, a **separate, client-side-encrypted daily `pg_dump` copy** goes to a **second provider: AWS S3 ap-south-1 (Mumbai)**. Different account, different tool, different key.
3. **At erase step E**, the **erase ledger** goes to that same second provider. The ledger must never share a failure domain with the backup it corrects.

Backup shape:
- a daily gzip base backup and zstd WAL with `archive_timeout` 5 min, so **RPO ≤ 5 min**;
- a **30-day** recovery window; the published erase SLA for backup copies is **≤ 35 days**;
- the lineage `projects-pgstore-v1` is bumped on every restore;
- a read-only ObjectStore for recovery;
- **no Object Lock at first**; a **governance lock of 30 days** is added at the M3 gate if the lock spike passes;
- **a card on file with no usage caps.**

Rollout is tiered:

| Tier | When | Contents |
|---|---|---|
| 0 | One weekend, now | Backups MVP |
| 1 | The M3 gate | Drill 0, which includes the restart-archiving check for plugin bug #828 and a from-zero restore on the laptop; the second copy; an hourly backup check |
| 2 | Erase step E | The ledger |
| 3 | T6 | Audio, in a separate B2 account |

Every restore follows **one post-restore sequence**:
1. Hold all xLearn workloads at 0.
2. Delete the NATS streams.
3. Start the consumers with the relays paused, and check that every durable consumer exists.
4. Reseal the outboxes, then unpause the relays.
5. Replay the ledger, judged by **server timestamps**, with the operator approving the list.
6. Rotate the JWT key, revoke all sessions and suspend BYO keys.
7. Reopen.

R2 is rejected for backups: its equal-size multipart rule conflicts with barman's upload parts. MinIO is rejected because both its repos are archived. Cost: **$0** at owner scale, **about $0.3 a month** at 100 learners, **about $2–3.5 a month** at 1,000.

| Key call | Resolution | Why |
|---|---|---|
| Backup provider | **B2 EU Central** is the primary. **AWS S3 ap-south-1** holds the second copy and the ledger, and is the fallback primary. | R2 requires equal-size multipart parts; barman PR #1161 is still unmerged. B2 is $0 at this scale and restore egress is free. A second provider covers what a lock can't: B2's own doc says the only way out of an over-long lock is to *close the account*, so closing an account removes even compliance-locked data. |
| Where blobs go | Inline `bytea` for all of v2.0. Audio waits for T6. | Nothing forces a bucket before T6. |
| Lock | None at Tier 0. **Governance, 30 d (= the window)** at the M3 gate, set only through `hack/b2/lock.sh`, which asserts the value. | Compliance mode can't be undone (a typo means closing the account) and adds risk from the checksum requirement on locked buckets. With data this small, the second provider is the stronger defence. |
| Ledger home | AWS, not the B2 account that holds the backups | It must survive exactly the account or node event that forces a restore. |
| Ledger replay trust | **The server's version timestamp**, never the `erased_at` in the payload. The operator approves a dry-run list. | A leaked writer key could otherwise future-date an entry and erase a live account. |
| Window and cadence | 30 d, daily base backups. Re-decide both at T1's resize trigger. | A solo operator may notice damage late. Cost is about $0. |
| Plugin install | HelmRelease 0.8.0 with a HelmRelease-level `dependsOn` on cert-manager. No `crds:` fields. | The chart ships its CRD as a template (`crds.create`), so Flux's `crds:` policy fields do nothing for it. |
| Monitoring | An hourly `pgstore-backup-check` in `databases` with a namespaced Role, plus dead-man checks for the dump and opscheck. **healthchecks.io is both the dead-man and the push channel.** | Plugin bug #828 can silently stall archiving. Hourly costs the same as daily. |
| Drills | Drill 0 in the cluster plus a from-zero rebuild on the laptop. After that, routine drills run from the laptop (`hack/drill.sh`), with in-cluster drills only after a CNPG or plugin minor bump or a PG major bump. | Proves the backups restore without costing node RAM or a PR per drill. |

## 2. Options compared

| Option | What it is | Single-node fit | Can it be the backup? | GitOps / SOPS fit | Ops burden (solo) | Monthly cost at our scale | License / health (2026) | Verdict |
|---|---|---|---|---|---|---|---|---|
| **Backblaze B2 EU Central** | External S3. Versioning always on; Object Lock; app keys scoped to a bucket or prefix | No pods, no disk | **Yes.** Different provider and continent, versioned. But **closing the account removes locked data**, so pair it with a second provider. | Secrets use the existing `.sops.yaml` rules. Bucket, key and lifecycle state lives as code in `I/hack/b2/`. | ~0.5 day setup, no upgrades | **$0** (card on file; first 10 GB free; A/B/C calls free for pay-as-you-go); ~$0.09 at 100 learners; ~$1.5 at 1,000 (daily base) | Commercial. $6.95/TB-month, no minimum duration | **Adopt** as the primary backup target |
| **AWS S3 ap-south-1** | The reference S3, in the same metro as the node | none | Yes. barman's reference target; IAM can grant `PutObject` without delete; Object Lock | same | IAM user and budget alarm: ~2 h once | ~$0.003 now; ~$0.2 at 100; ~$1.8 at 1,000 (dumps, 14 d) | Commercial. S3 Standard ≈ $0.025/GB-month in Mumbai (approx.; confirm at signup) | **Adopt** for the second copy and the ledger (owner Q2); **fallback primary** |
| Cloudflare R2 | External S3, zero egress | none | **Not for base backups.** Parts must be equal size and barman's aren't; no versioning; no write-only key | same | low | $0 → ~$1 | Commercial. barman PR #1161 open, plugin #411 open | **Reject** for backups. Re-check only when a plugin release bundles fixed-size parts. |
| **Postgres `bytea`** | Blobs as lz4-TOASTed columns in the owner's schema | already running | It is *what* gets backed up | none | none | $0 | n/a | **Adopt** for canvas final and draft and for transcripts (≤ 512 KiB). **Not** for audio. |
| MinIO community + operator | AGPL S3 server | ~0.3–1 GiB (inferred) | No: same disk | Frozen charts | High (no patches) | $0 | **Server repo archived (last push 2026-04-24); operator archived (2026-03-20)** | **Reject** |
| MinIO AIStor Free | Proprietary edition | as above | No | Vendor charts | Licence phones home and expires | $0 | Proprietary EULA; no encryption at rest in Free | **Reject** |
| Garage v2.4 in the cluster | AGPL, light, single-node mode | Good (tens of MiB) | **No.** Same disk, same credentials; restoring it needs the cluster being restored | Chart only from the source repo; raw IngressRoute; new SOPS rule | 1–1.5 days, plus upgrades and a new internet-facing parser | $0 cash | Maintained (v2.4.1, 2026-09-08) | **Reject** for production. Use in compose/e2e **only once T6 needs presigned uploads**. |
| Garage on a second host | Self-run off-node S3 | n/a | Yes, off-node | Second machine outside Flux | A second box to patch and monitor | ~$4–6 (inferred) | as above | Only if policy rules out every external provider |
| SeaweedFS `allInOne` | Apache-2.0 multi-role store | ~150–400 MiB (inferred) | No | Chart; S3 auth off by default | 1.5–2 days, frequent releases | $0 | Active | Reject |
| RustFS, Rook/Ceph, Versity, CloudServer | Other self-hosted stores | Ceph OSD ≥ 4 GB; RustFS RAM unpublished | No | varies | high | $0 | RustFS 1.0 GA 2026-09-16 (plus a 2025 CVSS 9.8 CVE); Versity has no lifecycle; CloudServer needs Backbeat | Reject |

## 3. Backups

### Target and scope

| Asset | Off-node? | How it comes back |
|---|---|---|
| **CNPG `projects-pgstore`** (every database) | **Yes. Barman plugin → B2** (PITR, 30 d) | PITR into a new lineage. Holds the learning record, judge schema, outboxes (the durable event log), tombstones and `erase_request`. |
| **Same cluster, logical copy** (Tier 1, owner Q2) | **Yes. Daily `pg_dump -Fc` → `age` → AWS**, kept 14 d | `pg_restore` into a fresh initdb cluster. RPO ≤ 24 h. Covers the loss of the B2 account and bugs in barman or the plugin. |
| NATS JetStream PVC (5 Gi) | **No, on purpose** | Rebuilt from the restored outboxes. An old snapshot would be *harmful*: its ack floors wouldn't match the database. |
| airlift PVC (1 Gi) | No | Re-seeded (`I/apps/airlift.yaml`) |
| Longhorn backup target or recurring snapshots, Velero | **No** | Same-disk or crash-consistent copies that duplicate Barman. **Infra rule:** a PR adding a PVC with data that can't be rebuilt must name that data's off-node backup. |
| k3s datastore | No | Rebuilt from infra git. The only objects outside git are `sops-age`, the Flux deploy key, and CNPG-generated secrets and certificates; all are recreated. |
| **SOPS age key** | In the cluster as `sops-age` by design, **plus two offline copies** (password manager and one offline medium) | Root of trust. Losing it loses `COACH_MASTER_KEY` and with it every stored BYO key (ADR-0007). |
| **Backup age key** (new) | **Offline only.** Its public recipient sits in a ConfigMap. | Decrypts the dumps. Cluster compromise and AWS can't read them. |
| B2 master key, AWS root/admin | Password manager, **hardware-key 2FA**, a login email not used for GitHub or Hostinger | — |
| Eval-pack source | Private GitHub plus GHCR's last 10 versions. **Never in any bucket** (T1). | — |
| Host setup (open-iscsi, iscsid, ufw 22/80/443, pinned k3s) | Commit as `I/hack/host-bootstrap.sh` | Today it exists only as comments (`I/infrastructure/storage/release.yaml:6-9`) |
| Hostinger weekly VPS images (the KVM 4 plan includes them) | **Not a backup we use** | Whole-disk images include the database **and k3s's plaintext Secrets (including `sops-age`)**. Owner check: turn them off if hPanel allows. Otherwise treat the Hostinger login as tier-0 and fit their retention inside the ≤ 35 d SLA. Never restore one while B2 holds newer WAL. |

### Accounts, buckets and keys

Account hygiene applies to every account:
- a **card on file from day 1** (free-tier accounts are capped, and a cap stops uploads);
- **no storage, download or transaction caps**, only cap *alert* emails at 75% and 100% of generous levels. A cap hit means the archiver stalls, `pg_wal` fills, and the database goes down in about 2–4 days.

`<o>` is an owner-unique name prefix, because bucket names are global. All buckets are private and encrypted at rest by the provider.

| Account | Bucket | Layout | Versioning / lock | Lifecycle | Keys (SOPS unless noted) |
|---|---|---|---|---|---|
| B2 "backups" (EU Central) | `<o>-pgstore-backups` | `projects-pgstore-vN/{base,wals}/…` | Versioned (always on in B2). **Tier 0: no lock. M3: governance, 30 d default**, applied by `lock.sh`, which asserts `days == 30 && mode == governance` | Live lineage `projects-pgstore-v1/`: hide → delete after 1 d; cancel unfinished large files after 1 d. **Retired lineage:** upload → hide 35 d, then hide → delete 1 d | **`pgstore-archive-rw`**: listBuckets, listAllBucketNames (B2 requires it on bucket-restricted keys), listFiles, readFiles, writeFiles, **plus `deleteFiles` only if spike 1 shows barman retention needs it** (B2's S3 key doc says DeleteObject(s) should have both). Never `bypassGovernance`. **`pgstore-archive-ro`**: listBuckets, listAllBucketNames, listFiles, readFiles |
| AWS (Q2a) | `<o>-pgstore-dumps` | `<db>/<YYYY-MM-DD>.dump.age` | Versioning on, no lock | Current versions expire after 14 d; noncurrent versions after 14 d (so an overwritten original survives 14 d); abort incomplete multipart uploads after 1 d | **`pgstore-dump-w`**: `s3:PutObject` on `<o>-pgstore-dumps/*` only. No Get, List or Delete. |
| AWS (Q2a) | `<o>-xlearn-erase-ledger` (Tier 2) | `erase-ledger/<account_id>`; `cred-ledger/<account_id>/<ulid>` | Versioning + **Object Lock compliance, 40 d** (≥ SLA plus margin). Entries are never deleted anyway, so a long lock has no downside. | none (kept forever, T1) | **`identity-ledger-w`**: `s3:PutObject` on those two prefixes only. **`ledger-ro`**: ListBucketVersions and GetObjectVersion; **password manager only**, added by a restore PR and removed at teardown |
| B2 "media", a **separate account** (T6) | `<o>-xlearn-interview-media` | `u/<account_id>/audio/<id>` | Versioned, no lock | upload → hide 30 d; hide → delete 1 d; cancel unfinished 1 d | `<t6svc>-media`: listFiles, readFiles, writeFiles |

If owner Q2 is (c), there is no AWS: the ledger moves to a separate versioned B2 bucket in the backups account, and the account-level risk is accepted.

### CNPG plugin configuration shape (for T7; nothing written now)

The ObjectStores have provider-neutral names, so switching to AWS is only an endpoint and secret change.

```yaml
apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: pgstore-archive
  namespace: databases
  annotations: { kustomize.toolkit.fluxcd.io/prune: disabled }
spec:
  retentionPolicy: "30d"
  configuration:
    destinationPath: s3://<o>-pgstore-backups/
    endpointURL: https://s3.eu-central-00N.backblazeb2.com      # exact host from the bucket page
    s3Credentials:
      accessKeyId:     { name: pgstore-archive-rw, key: ACCESS_KEY_ID }
      secretAccessKey: { name: pgstore-archive-rw, key: ACCESS_SECRET_KEY }
    wal:  { compression: zstd, maxParallel: 2 }
    data: { compression: gzip, jobs: 2 }     # data accepts bzip2|gzip|lz4|snappy (chart CRD); no tags/historyTags, no encryption
  instanceSidecarConfiguration:
    retentionPolicyIntervalSeconds: 1800
    env:
      - { name: AWS_DEFAULT_REGION, value: eu-central-00N }   # (inferred; spike)
      # AWS_REQUEST_CHECKSUM_CALCULATION / AWS_RESPONSE_CHECKSUM_VALIDATION=when_required ONLY if the spike needs it
    resources: { requests: { cpu: 10m, memory: 64Mi }, limits: { memory: 256Mi } }
---
# pgstore-archive-ro: identical, pgstore-archive-ro credentials, NO retentionPolicy (recovery and drills only)
```

Changes to the Cluster (`I/infrastructure/database/cluster/cluster.yaml`):
- `metadata.annotations: { kustomize.toolkit.fluxcd.io/prune: disabled }`. The same annotation goes on `namespace.yaml`, both ObjectStores and the archive Secrets.
- `spec.plugins: [{ name: barman-cloud.cloudnative-pg.io, isWALArchiver: true, parameters: { barmanObjectName: pgstore-archive, serverName: projects-pgstore-v1 } }]`. This **restarts the only instance, about 30 s**.
- **`spec.backup` stays unset.** The scaffold's `spec.backup.method: plugin` (`I/infrastructure/database/README.md:131`) is wrong: `method` belongs on the ScheduledBackup.
- `storage.size: 20Gi` goes in its **own later PR** (Tier 1). It is one-way and expands online. `wal_compression` is dropped for now.

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata: { name: projects-pgstore-daily, namespace: databases }
spec:
  schedule: "0 30 21 * * *"      # 6-field cron; 21:30 UTC = 03:00 IST
  immediate: true
  backupOwnerReference: self
  cluster: { name: projects-pgstore }
  method: plugin                 # the default is still barmanObjectStore; must be explicit
  pluginConfiguration: { name: barman-cloud.cloudnative-pg.io }
```

**Separate logical copy** (Tier 1, Q2a). CronJob `pgstore-dump` in `databases`:
- **Schedule:** `30 22 * * *` (5-field Kubernetes cron), `concurrencyPolicy: Forbid`.
- **initContainer:** the Cluster's own image (`ghcr.io/cloudnative-pg/postgresql:18.4-system-trixie`, bumped together with `imageName`). It runs `pg_dump -Fc` for every non-template database as the new managed role **`pgstore_dump`** (`inRoles: [pg_read_all_data]`, connection limit 1) into an `emptyDir` (sizeLimit 2Gi).
- **Main container:** the opscheck binary's `dump-ship` subcommand. It uses `filippo.io/age` to encrypt to the backup recipient (ConfigMap) and `aws-sdk-go-v2` to `PutObject`. Then it pings its healthchecks.io check, or `/fail`.
- The image is pinned by tag: image automation only writes `./apps` (`I/apps/image-automation.yaml`).

### Schedules, window, RPO/RTO

| Item | Value |
|---|---|
| Base backup | Daily. Re-decide cadence and window together at T1's resize trigger (~285 learner-years; compressed base ~2 GB). |
| WAL | Continuous, `archive_timeout` 5 min, at most 288 segments a day. Idle switch segments compress to tens of KB. |
| Recovery window | **30 d** (owner Q4) |
| Logical copy | Daily, kept 14 d (Q2) |
| **RPO** | Node or disk loss: ≤ 5 min. Logical error noticed within 30 d: 0. **B2 account and node lost together:** ≤ 24 h (from the dump). |
| **RTO** (inferred; measured at rehearsal) | Pod crash: minutes, automatic. One damaged schema: 0 downtime (surgical, ~1 h of work). In-place PITR: ≤ 1.5 h (three PRs plus the post-restore sequence). **Node loss: ≤ 4 h.** B2 lost with the node alive: 1 h to re-point to AWS, no data loss. B2 and node both lost: ≤ 6 h from the dump. |
| **Published erase SLA for backup copies** | **≤ 35 d.** Worst case (inferred): the base backup taken just before an erase is obsoleted at +31 d; retention runs within 30 min; lifecycle hide → delete +1 d; lifecycle runs once a day, +1 d; so about 33 d. Dumps last ≤ 29 d (14 d current + 14 d noncurrent + cadence). Also inside the SLA: `Retain` PVs from an in-place restore, PITR copies (≤ 24 h), rehearsal hosts (same day) and Hostinger images (owner check). |

### Immutability: what each layer actually stops

| Threat | Tier 0 (owner-only data) | M3 gate and later |
|---|---|---|
| Leaked `pgstore-archive-rw` hides or deletes history | Versioning. Hides purge after 1 d. With `deleteFiles` it could delete versions. Accepted, because the live database is untouched. | **Governance lock, 30 d** (no key in the cluster has `bypassGovernance`) **plus the AWS copy** |
| The same key *overwrites* WAL or base files (poisoning; barman reads the latest version) | Caught by the version audit at each drill. Rollback works for only ~1 d. | Locked originals are kept 30 d, and **`hack/b2/rollback-to.sh <lineage> <T>`** copies each object's newest version with upload time ≤ T into a **fresh lineage** (master key, laptop). The dump is independent of this. |
| Stolen B2 master **API key** | none | Governance can be bypassed; the AWS copy survives |
| B2 **login takeover, account closure, or Backblaze suspension** | none | **The AWS copy and ledger survive.** Compliance lock would *not* help: closing the account removes locked files (Backblaze Object Lock doc). |
| Silent barman or plugin bug in the archive | Drills | Plus the `pg_dump` copy (different tool) |
| AWS account lost | n/a | B2 is intact. The ledger is re-seeded from identity's `erase_request` rows (`erase-ledger reseed`). Residual risk: erasures in the ≤ 5 min before a node loss at the same time. |

### Erase ledger (Tier 2, erase step E)

- **Home:** AWS `<o>-xlearn-erase-ledger` (Q2a), in a different account from the backups it corrects.
- **Write, intent-first:**
  1. `DELETE /api/me`.
  2. identity PUTs `erase-ledger/<account_id>` with body `{account_id, request_id, requested_at}`.
  3. **Then** identity commits the erase transaction (T1 §6.6, step 2).

  If AWS is unreachable, identity returns 503. It never says "erased" before the intent is durable off-node.
- **Credential changes** (password change, OAuth unlink) write `cred-ledger/<account_id>/<ulid>` **after commit, best-effort**. A sweeper retries rows with `ledger_written_at IS NULL`. Nothing blocks on AWS.
- **Replay** (the `xlearn-identity erase-replay` Job, `ledger-ro` key, **always a dry run first**):
  - It lists **all object versions** and takes each version's **server timestamp** (`LastModified`). The payload's own timestamps are ignored, because the writer controls them.
  - It uses the **earliest** version per account. A later version is always newer, since the key holder can't backdate, so extra versions are flagged as anomalies.
  - `T_restore` is the recovery `targetTime` if one was set; otherwise the maximum `created_at` across all outboxes (the achieved-RPO marker).

  | Account in restored DB | Server timestamp `t_w` | Class |
  |---|---|---|
  | absent | any | skip |
  | present | `t_w` in (T_restore − 1 min, T_declared] | **expected** |
  | present | older, newer than T_declared, several versions, or > 5 expected | **anomalous**: a planted entry (leaked writer key) or an intent whose commit never happened |

  The operator approves the list with one flag in the Job args (a PR). There is **no unattended auto-erase**. Approved erases re-run the normal erase transaction, and the other services erase through their own consumers (ADR-0005 holds).
  - `cred-ledger` entries with `t_w` > T_restore → force a password reset (invalidate the hash, drop OAuth links, send the reset email).
- **Drift audit at each drill:** from the laptop, compare the number of ledger objects against `erase_request` rows.

### Post-restore sequence (mandatory after *any* production restore; replaces the draft's order)

| # | Step | Mechanism |
|---|---|---|
| 1 | The restore PR sets **every xLearn HelmRelease `replicaCount: 0`**, sets `RELAY_PAUSED=true`, suspends the opscheck CronJobs, adds a **rotated JWT secret**, and adds `xlearn-erase-ledger-ro` plus the replay, revoke and verify Jobs | One PR on current `main`, from the runbook snippet |
| 2 | Wait for the Cluster to be healthy. The verify Job checks schemas, goose versions, `SHOW data_checksums`, and outbox `max(created_at)` → T_restore and the achieved RPO | Job |
| 3 | **Delete every `XLEARN_*` stream**, using the ops NATS identity once NATS auth exists | Job / NATS CLI |
| 4 | PR: backends back to 1 (**gateway stays at 0**), relays still paused. **Check that every durable exists.** `DeliverNew` durables would otherwise skip the republish (`X/internal/assessment/service.go:109`, `X/internal/review/service.go:135`). | PR + check |
| 5 | Per-producer `reseal` Jobs (`UPDATE outbox SET sent_at = NULL`), then PR `RELAY_PAUSED=false`. Inboxes dedupe. | Jobs + PR |
| 6 | **Ledger replay** (above). Wait for all 5 acks per request. | Job |
| 7 | **Security reset:** revoke all sessions (`identity revoke-all-sessions`); the JWT key is already rotated; coach sets **every BYO key to `needs_reconfirm`** (a deleted key may have come back); send users a notice to redo any security change made after T_restore | Jobs |
| 8 | ADR-0018 projection reconcile | Job |
| 9 | PR: gateway `replicaCount: 1`; remove the Jobs and `ledger-ro`; unsuspend opscheck; confirm `pgstore-backup-check` is green and the immediate base backup is in `vN+1` | PR |

Code this needs (Tier 1, before Drill 0):
- a `RELAY_PAUSED` knob in `X/internal/platform/events/relay.go`;
- a per-producer `reseal` subcommand;
- identity `revoke-all-sessions`;
- coach `suspend-byok`.

### Restore procedures (GitOps only)

- **Lineage rule:** read lineage `vN` through `pgstore-archive-ro` and archive to `vN+1` through `pgstore-archive`. The restore checklist then adds the retired-lineage B2 rule on `vN/`.
- **Recovery snippet**, kept in `I/docs/dr-runbook.md` and applied to *current* `main` at incident time. **There are no DR branches.**

  ```yaml
  bootstrap: { recovery: { source: origin, recoveryTarget: { targetTime: "<T>" } } }   # omit recoveryTarget for latest
  externalClusters:
    - name: origin
      plugin: { name: barman-cloud.cloudnative-pg.io, parameters: { barmanObjectName: pgstore-archive-ro, serverName: projects-pgstore-vN } }
  plugins:
    - { name: barman-cloud.cloudnative-pg.io, isWALArchiver: true, parameters: { barmanObjectName: pgstore-archive, serverName: projects-pgstore-vN+1 } }
  ```

- **(a) Surgical restore (the most likely case).**
  1. A PR adds a temporary `pgstore-pitr` Cluster at `targetTime`, using the RO store, with no plugins, roles or Database CRs.
  2. Run the ledger replay dry run against the copy.
  3. `pg_dump -n <schema>` → import into production in a per-service window.
  4. **Reset that service's durables to replay from T** (recreate them with `DeliverByStartTime`; the inbox dedupes), then run the ADR-0018 reconcile downstream.
  5. **Delete the copy within 24 h.** It is PII and counts toward the SLA.
- **(b) In place.**
  1. PR: drop the `prune: disabled` annotation from the Cluster.
  2. PR: remove the Cluster. The PVC goes with it via its ownerRef; the `Retain` PV stays as a forensic copy and must be deleted within 30 d.
  3. PR: re-add the Cluster with the recovery snippet, plus step 1 of the post-restore sequence.
  4. Run the post-restore sequence.
- **Accidental Cluster deletion.** Prune protection makes removing it from git a no-op; only a manual `kubectl delete` gets past it.
  - **Never revert a deleted Cluster back in.** It would `initdb` a new empty PVC, and the services would migrate it and accept signups: a split brain.
  - Either **re-bind the Released PV** (clear its `claimRef`, pre-create PVC `projects-pgstore-1` with CNPG labels, then re-add the Cluster; RPO 0; inferred, rehearsed once at Drill 0 on k3d),
  - or recover from B2 into a new lineage.
- **Hazard:** never boot a Hostinger image or a Longhorn snapshot while B2 holds newer WAL. The old instance would overwrite archived segments (inferred). If unavoidable, revoke `pgstore-archive-rw` first, then recover into a new lineage.

### Drills

| When | Drill |
|---|---|
| **Drill 0 (M3 gate)** | **(1) In the cluster:** a PR adds `drill-pgstore` (`longhorn-static`, RO store, no plugins, 100m/256Mi requests, 1 CPU / 1 Gi limit); the verify Job runs; a teardown PR follows the same day, because `databases` has `wait: true` and would stay red. **(2) Production restart check for #828:** bump `kubectl.kubernetes.io/restartedAt` on the Cluster through a PR, then assert that `pg_stat_archiver.archived_count` advances and `last_failed_time` doesn't change within 10 min. **(3) From zero on the laptop:** k3d + cert-manager + CNPG 0.29.0 + plugin 0.8.0, with secrets decrypted **using only the offline age key**, the recovery snippet, `pgstore-archive-ro`, a ledger replay dry run, the PV re-bind rehearsal, and one run of `rollback-to.sh` into a scratch lineage. |
| Monthly × 3, then quarterly | **Laptop `I/hack/drill.sh`:** `barman-cloud-restore` plus WAL replay into a local `postgres:18` with the RO key. Alternate "latest" (measures RPO) with "window edge" (T about 28 d back). Includes `pg_amcheck --heapallindexed`, the B2 version/hide audit (flags any WAL name with more than 1 version, and any hide younger than the retention cadence), the ledger drift audit, and a `hack/b2` config diff. No PRs, no node RAM. |
| CNPG or plugin *minor* bump, PG major | In-cluster drill (1) plus restart check (2). **Never bump the plugin in the same PR as a Postgres restart.** |
| Once before open signup | A full rebuild rehearsal on the laptop k3d with Flux (or a short-lived VPS). **Dummy** `COACH_MASTER_KEY`, OAuth and notification credentials, so there are no LLM calls or emails. Egress denied except B2, AWS, GHCR and GitHub. Torn down the same day. Measures RTO. |

A drill log is the PRs plus one line in `docs/v2/status.md`.

### Monitoring (smaller than the draft)

- **`pgstore-backup-check`**, hourly, in `databases`. ServiceAccount with a namespaced Role: `get` on `clusters.postgresql.cnpg.io` and `objectstores.barmancloud.cnpg.io`. Checks:
  1. `ContinuousArchiving=True`. This also covers the plugin being down or its certificate failing, and a B2 cap hit.
  2. ObjectStore `status.serverRecoveryWindow["projects-pgstore-v1"]`: `lastSuccessfulBackupTime` < 26 h, and `firstRecoverabilityPoint` ≥ now − 33 d (retention is working).

  On success it pings healthchecks.io; on failure it calls `/fail` with the reason. Grace is 2 h.
- `pgstore-dump` pings its own check (grace 26 h). T1's `xlearn-opscheck` pings a third.
- **healthchecks.io is both the dead-man and the push channel** (free tier: 20 checks). Recommend to T7 that it replace ntfy.
- **Dropped from the draft:** the exporter scrape from `xlearn`, the ClusterRole, the certificate check (cert-manager renews early), the ledger-lag check (intent-first makes it moot), and the ledger-ro key in the cluster.

### DR runbooks (`I/docs/dr-runbook.md`, one file; the xLearn commands link to `X/docs/runbooks/post-restore.md`)

**Node lost**
1. **Declare** and record `T_declared`. If compromise is possible, run *Node compromised* first.
2. **Provision** a VPS (≥ 4 vCPU / 15 GiB). Run `hack/host-bootstrap.sh`.
3. **Update the DNS A record** at the registrar (check the TTL now). The HTTP-01 certificate re-issues once DNS resolves.
4. `flux bootstrap github … --read-write-key` (image automation pushes to `main`). Create `sops-age` from the offline key. **These are the only imperative steps.**
5. Merge the restore PR: the snippet at latest from `vN` to `vN+1`, plus post-restore step 1.
6. Flux reconciles cert-manager → Longhorn → CNPG with the plugin → empty NATS → recovery (~30–45 min, inferred).
7. Run post-restore steps 2–9.
8. Add the retired-lineage rule on `vN/`. Log the measured RTO and RPO. Delete leftovers within 30 d.

**Node compromised.** The node holds `sops-age`, k3s stores Secrets in plaintext, and the Hostinger images copy both.
1. Isolate the node; never reuse it.
2. **Generate a new SOPS age key** and re-key `.sops.yaml`. Mint **new** B2 and AWS keys, the JWT key, the OAuth secret, every database role password and `COACH_MASTER_KEY`. Re-encrypt everything in **one PR**.
3. Rebuild using *Node lost*. If tampering is suspected, use PITR to before the compromise.
4. **Only then** revoke the old keys and delete old Hostinger images.
5. Purge the stored BYO keys and tell users to **rotate them at their providers**: the backups hold them under the exposed master key. Assess breach notice (owner, DPDP).

k3s `--secrets-encryption` is declined: its key sits on the same disk and in the same images.

**B2 lost** (node alive)
1. `hack/aws` creates `<o>-pgstore-backups` in ap-south-1.
2. A PR re-points `pgstore-archive`/`-ro` (endpoint + secrets) and bumps the lineage.
3. The immediate base backup runs.

If the node is lost too: fresh initdb Cluster with xLearn at 0 → a `pg_restore` Job from the latest dump, using the backup age identity as a *transient* SOPS secret that is rotated afterwards → the post-restore sequence with the AWS ledger.

**Age key lost:** re-key SOPS, re-issue every secret, force re-login, and ask users to re-enter their BYO keys.

### Spikes before T7 (laptop only; nothing touches the cluster)

1. **B2 + barman-cloud 3.20** (`postgres:18`, ~200 MB, 3 or more parts):
   - gzip base backup and zstd WAL, then restore;
   - retention with a 1-day window using **(i)** a key without `deleteFiles`, then **(ii)** a key with it: record which one works;
   - uploads into a **governance-locked** bucket (the Content-MD5/checksum requirement, with and without `when_required`);
   - hiding a locked version;
   - **a lifecycle rule deletes a hidden version once its 1-day lock lapses** (observe for 2 or more days);
   - `listAllBucketNames` on a bucket-restricted key.

   **Decided now:** if (i) fails, stay on B2 with `deleteFiles` and rely on the M3 governance lock, and §7 says so. Switch the primary to AWS **only** if base backup or restore fails. There is no R2 control run: the negative result is already documented.
2. **Ledger on AWS:** the writer can PUT but GET, LIST and DELETE are denied; an overwrite creates a new version; `ledger-ro` lists versions with `LastModified`.
3. **Dump pipeline:** `pg_dump -Fc` → age → PutObject-only key; restore with the offline key.
4. (T6) Browser presigned PUT with a signed Content-Length, plus CORS.
5. RTT and throughput from the VPS to B2 (**needs owner approval to run on the node**).

## 4. App blobs

**Is object storage needed in v2.0? Not for app blobs.** The only app-side S3 use is identity's ledger writer at step E, plus the dump shipper, which is an ops tool. Audio needs a bucket only when T6 ships.

| Blob (T1 §5) | v2.0 home | Later | Access path | Lifecycle | Backed up? | Erase |
|---|---|---|---|---|---|---|
| Canvas scene, final and counted (≤ 512 KiB) | **judge schema, inline `bytea`, permanently** | Spill to a judge-owned bucket only if a trigger fires (below) | **Proxied**, SPA → gateway → judge. The gateway uses `http.MaxBytesReader` (640 KiB → typed 413) instead of the silent `io.LimitReader` (`X/internal/gateway/bff.go:164`); judge enforces its own cap; the SPA canvas save timeout goes to ~45 s (15 s today, `X/web/src/lib/api.ts:12`) | forever | via PITR and the dump | SQL delete + tombstone |
| Canvas draft | judge inline, HOT upserts, 60 s debounce | same | proxied | 90 d sweeper (T4) | incidentally | SQL |
| Mock transcript (≤ 256 KiB) | the **T6 owner's schema**, inline | — | written server-side, read proxied | 12 months (T6) | via PITR and the dump | SQL |
| **Audio** (~8–11 MB, opt-in) | not in v2.0 | **`<o>-xlearn-interview-media`, in a separate B2 account** (never the backups account) | **Presigned PUT** (TTL 15 min; Content-Type and Content-Length signed; one key per URL) → `…/complete`: HeadObject (size, type) + a 64-byte ranged GET for magic bytes (WebM `1A45DFA3`, MP4 `ftyp`) + a tombstone check; otherwise hide and reject. Playback: presigned GET, 15 min. A sweeper hides `pending` objects after 1 h. | 30 d bucket lifecycle | **No** (disposable) | prefix hide `u/<acct>/`, purged ≤ 1 d |
| Video | not stored | — | — | — | — | — |
| Public media overflow (> 32 MB) | not needed | revisit only if the limit is hit | — | — | git | — |
| Erase and credential ledger | AWS `<o>-xlearn-erase-ledger` (Tier 2) | — | server-side from identity | kept forever | *is* the out-of-backup record | never deleted |

- **Triggers for a judge bucket:** a canvas course is live **and** judge body bytes pass 1 GiB or 25% of the Postgres volume; or any payload kind over 1 MiB appears. Until then, **defer T1's nullable `object_key`** (a column lands with its first reader).
- **Layout under ADR-0005:**
  - one writing owner per bucket, holding its own key;
  - the gateway holds **no** storage credentials;
  - no service reads another service's bucket;
  - CNPG owns the backup bucket, `pgstore-dump` the dump bucket, identity the ledger, and the T6 owner the media bucket;
  - T1's `u/<account_id>/<kind>/<id>` layout is kept inside each bucket.
- **Client:** `aws-sdk-go-v2/service/s3` behind `X/internal/platform/blob`, with `RequestChecksumCalculation` and `ResponseChecksumValidation` = `WhenRequired`, and a path-style flag.
  - Plain env: `BLOB_ENDPOINT`, `BLOB_REGION`, `BLOB_BUCKET`.
  - SOPS `envFrom`: `BLOB_ACCESS_KEY_ID`, `BLOB_SECRET_ACCESS_KEY`.
  - Rotate by bumping `podAnnotations`, as `I/apps/xlearn-identity.yaml` already does.
- **CORS** is needed only on the media bucket (T6): origin `https://projects.sujaykumar.dev`; PUT, GET, HEAD; header `Content-Type`; expose `ETag`; max-age 3600. Add the B2 host to CSP `connect-src` and `media-src` when T7 adds a CSP.
- **Encryption:**
  - provider-side encryption at rest plus TLS everywhere;
  - dumps are **client-side age-encrypted** to an offline key;
  - no SSE-C (it breaks browser presigned uploads);
  - client-side envelope encryption for media is deferred until an object holding C4 data lives longer than 30 d.
- **Local dev and CI:** unit tests use `gofakes3` in process. It enforces no signatures, which helps test the completion verifier and version-listing replay. In compose and e2e, the ledger writer uses a **`file://` driver**. **Garage (`dxflrs/garage:v2.4.x --single-node`) arrives only with T6**, for presign and CORS. No MinIO anywhere.

## 5. Infra changes (GitOps)

| Tier / PR | File (I/) | Change |
|---|---|---|
| 0 / PR 0 (by hand) | `hack/b2/` | **New.** Bucket, lifecycle and CORS JSON; key-capability lists; `b2` CLI scripts run from the laptop; `lock.sh` (asserts days and mode); `rollback-to.sh`; `audit.sh` (version/hide audit and config diff) |
| 0 / PR 1 | `infrastructure/database/operator/plugin-barman-cloud.yaml` | **New** HelmRelease `plugin-barman-cloud`, ns `cnpg-system`, chart **`"0.8.0"`** (app v0.15.0), `sourceRef` the existing HelmRepository `cloudnative-pg` (ns `flux-system`), **no `install/upgrade.crds`** (the chart templates its CRD under `crds.create: true` with `resource-policy: keep`), `install.remediation.retries: 3`, **`dependsOn: [{name: cert-manager, namespace: cert-manager}, {name: cloudnative-pg}]`**, `values.resources` requests 10m/32Mi, limit 128Mi. The chart supports only the `Recreate` strategy, so every plugin upgrade briefly interrupts archiving. |
| 0 / PR 1 | `clusters/vps/databases.yaml` | **No new Kustomization and no new `dependsOn`.** `cnpg-operator` stays independent; its `wait: true` covers the plugin. Fix the stale "3 instances bootstrap" comment (line 44). |
| 0 / PR 2a | `infrastructure/database/cluster/pgstore-archive-rw.enc.yaml`, `pgstore-archive-ro.enc.yaml` | **New** SOPS Secrets (`ACCESS_KEY_ID`, `ACCESS_SECRET_KEY`), annotated `prune: disabled`. Covered by the existing rule `.sops.yaml:6-8`. |
| 0 / PR 2a | `infrastructure/database/cluster/objectstore.yaml` | **New** `pgstore-archive` and `pgstore-archive-ro` (§3). No restart; confirm they apply cleanly. |
| 0 / PR 2b | `infrastructure/database/cluster/cluster.yaml`, `namespace.yaml`, `scheduledbackup.yaml` (new) | `spec.plugins` (lineage `projects-pgstore-v1`); `prune: disabled` on the Cluster and the Namespace; ScheduledBackup; the comment "backup: intentionally omitted" is replaced. **Quiet window (~30 s restart), revert PR prepared.** |
| 0 / PR 2b | `infrastructure/database/README.md` (Backups, 122–134), `README.md` (~147–148) | Rewrite: HelmRelease not a raw manifest; `method: plugin` on the ScheduledBackup; B2 not "e.g. R2"; lineage rule; RO store; the prune rule; pointer to the runbook |
| 1 | `infrastructure/database/cluster/cluster.yaml` | `storage.size: 20Gi` (its own PR) |
| 1 | `infrastructure/database/cluster/backup-check.yaml` | **New** SA + Role (`get` on clusters and objectstores) + RoleBinding + hourly CronJob (opscheck image, pinned tag) |
| 1 | `infrastructure/database/cluster/backup-healthchecks.enc.yaml` | **New.** Ping URLs for backup-check and dump |
| 1 (Q2a) | `infrastructure/database/cluster/pgstore-dump.yaml`, `pgstore-dump-w.enc.yaml`, `pg-pgstore-dump.enc.yaml`, `backup-age-recipient.yaml` (ConfigMap) | **New.** Dump CronJob; AWS PutObject-only key; role password; public recipient |
| 1 (Q2a) | `infrastructure/database/cluster/cluster.yaml` | `managed.roles` + `pgstore_dump` (`login`, `inRoles: [pg_read_all_data]`, `connectionLimit: 1`) |
| 1 | `hack/aws/` | **New.** Bucket, versioning, lifecycle and lock JSON; IAM policies; AWS CLI scripts (laptop) |
| 1 | `hack/host-bootstrap.sh`, `hack/drill.sh` | **New**, idempotent |
| 1 | `docs/dr-runbook.md` | **New, one file:** recovery snippet, post-restore sequence, node lost / compromised / B2 lost / age lost / accidental deletion. Links to `X/docs/runbooks/post-restore.md` for the xLearn subcommands. **No `dr/*` branches.** |
| Drill 0 onward | `infrastructure/database/cluster/drill-pgstore.yaml` (+ verify Job) | Transient: the drill PR adds it, a teardown PR removes it the same day |
| 2 (step E) | `apps/secrets/xlearn-erase-ledger.enc.yaml` (writer) | **New.** Existing rule `.sops.yaml:2-4` |
| 2 | `apps/xlearn-identity.yaml` | `envFrom` the ledger secret; `LEDGER_ENDPOINT`, `_BUCKET`, `_REGION`; `podAnnotations: xlearn.dev/ledger-rev` |
| restore only | `apps/secrets/xlearn-erase-ledger-ro.enc.yaml` + Jobs | Added by the restore PR, removed at teardown. Otherwise the key lives in the password manager. |
| T6 | `apps/secrets/xlearn-<t6svc>-media.enc.yaml` + that service's HelmRelease env | **New** |

- **No `.sops.yaml` change.** Every new path falls under an existing rule.
- **No chart change.** `env`, `envFrom` and `replicaCount` already pass through (`I/charts/project/values.yaml:29,36-37`). The restore PR's `replicaCount: 0` is a values edit in the seven `apps/xlearn-*.yaml` files.
- **NetworkPolicy (T7):** when default-deny lands, allow egress to `0.0.0.0/0:443` for the Postgres pods (Barman sidecar), the `pgstore-dump` and `pgstore-backup-check` CronJobs, identity, and later the T6 owner. NetworkPolicy can't match an FQDN. The runner namespace must **not** inherit this.
- **Secrets inventory:**

  | Where | Secrets |
  |---|---|
  | `databases` | `pgstore-archive-rw`, `pgstore-archive-ro`, `pgstore-dump-w`, `pg-pgstore-dump`, `backup-healthchecks` |
  | `xlearn` | `xlearn-erase-ledger` (Tier 2); the opscheck ping URL (T1); later `xlearn-<t6svc>-media` |
  | Transient (restore PRs only) | `xlearn-erase-ledger-ro`; the backup age identity (dump path only; rotated afterwards) |
  | In the cluster by design | `sops-age` (plus two offline copies) |
  | Never in the cluster | B2 master key, AWS root/admin, backup age private key |

## 6. Single-node fit & cost

- **Resources:**
  - plugin controller 10m / 32 Mi requested (128 Mi limit);
  - Barman sidecar 10m / 64 Mi (256 Mi limit);
  - the two CronJobs run for seconds a day (about 50m / 64 Mi, transient; the dump `emptyDir` is ≤ 2 Gi on node disk).
  - The memory-limit sum goes from about 12.5 to about 12.9 of 15 Gi (no swap).
  - Routine drills use **no node RAM** (they run on the laptop).
  - A daily gzip base takes seconds today and about 5–10 min at 1,000 learners (inferred).
- **Disk:** PVC 10 → 20 Gi (Tier 1; 176 GB free).
  - Worst-case archiver-stall growth: 16 MiB per 5 min, so ≤ 4.5 GiB/day, about 3.5 days to fill 20 Gi.
  - The hourly check plus a 2 h grace fires within about 3 h of a stall.
  - There's no object store on the node.
- **Egress:** the node matches **Hostinger KVM 4** (4 vCPU / 16 GB / 200 GB NVMe, **16 TB/month**, weekly backups included), so egress is not an issue.

  | Scale (daily base) | Outbound per month | Share of 16 TB |
  |---|---|---|
  | Owner only | < 1 GB | — |
  | 100 learners | ~25 GB | — |
  | 1,000 learners | ~220 GB (~60 GB at a weekly base) | < 1.5% |

  Drills download from B2 for free (up to 3× stored).

| Monthly | Owner only | 100 learners (1 yr) | 1,000 learners (1 yr, after the resize trigger) |
|---|---|---|---|
| B2 backups stored (30 d window) | < 1 GB | ~23 GB (base ~0.7 GB × 31 + WAL ~1.5 GB) | ~230 GB daily / ~60 GB weekly |
| **B2 cost** ($6.95/TB, first 10 GB free, A/B/C calls free with a card) | **$0** | **~$0.09** | **~$1.5 daily / ~$0.35 weekly** |
| AWS dumps (14 d; `pg_dump -Fc` ≈ ¼ of DB size, inferred) | ~0.05 GB → ~$0 | ~7 GB → **~$0.18** | ~70 GB → **~$1.75** |
| AWS ledger (Tier 2) | ~$0 | ~$0 | ~$0 |
| Audio (T6, separate B2 account, 30% opt-in, 30 d) | 0 | ~1.1 GB → $0 (free tier) | ~11 GB → ~$0.01 |
| healthchecks.io (3 of 20 free checks) | $0 | $0 | $0 |
| **Total** | **$0** | **~$0.3** | **~$3.3 daily / ~$2.1 weekly** |
| R2, for contrast | $0 | ~$0.2 | ~$1.1 |

## 7. Security / threat notes

| Credential | Lives in | If it leaks |
|---|---|---|
| `pgstore-archive-rw` | CNPG sidecar, ns `databases` | **A full read of the database:** learner data, password hashes, OAuth ids, encrypted BYO keys (not their master key). It can add, hide and **overwrite** (poisoning), and **with `deleteFiles` (likely needed; spike 1) it can delete unlocked versions.** Tier 0: B2 history is at risk, the live database is not. From M3: the governance lock covers 30 d, `rollback-to.sh` undoes poisoning, and the AWS copy is out of its reach. |
| `pgstore-archive-ro` | SOPS, `databases`; owner laptop | Full read. Adds nothing beyond the RW key. |
| `pgstore-dump-w` | `databases` | Can only add objects or versions. It can't read, list or delete, and overwritten originals survive 14 d. The dumps are age-encrypted, so it learns nothing. |
| `identity-ledger-w` | identity pod (Tier 2) | Can **plant** entries or versions. It can't read, list or delete, and it can't backdate (server timestamps). **A planted entry never erases anyone unattended:** replay is a dry run the operator approves, and anomalies are flagged. The drift audit catches plants at drill time. |
| `ledger-ro` | password manager; restore PRs only | Lists erased account UUIDs (pseudonymous) |
| `<t6svc>-media` | T6 owner pod, a separate B2 account | Read and write on voice recordings (≤ 30 d). Can't touch the backups account. |
| B2 master key and login | password manager + hardware 2FA | Can delete anything unlocked, and **can close the account, which removes even compliance-locked data**. The AWS copy survives. |
| AWS root/admin | same | Can delete the dumps and ledger. B2 survives; the ledger can be re-seeded from the database. |
| SOPS age key | `sops-age` in the cluster + offline | Decrypts every SOPS secret |
| Backup age key | offline only | Decrypts the dumps |
| Hostinger login | password manager + 2FA | Whole-disk images contain the database, **`sops-age` and every plaintext Secret**. Tier-0. |

- **Public access:** every bucket is private, with no custom domain and no CDN. The only browser access is presigned URLs to the learner's own objects (T6).
- **Presigned URL abuse (T6):**
  - short TTLs; type and length signed; one key per URL;
  - server-side verification of type, size and magic bytes;
  - quotas: ≤ 3 recordings a day, 1 pending per mock;
  - a pending sweeper and the 30 d lifecycle;
  - a **separate account, so abuse can't touch backups**;
  - `Cache-Control: no-store`; presigned URLs are never logged or sent to analytics.
- **Caps:** none on the backups account; they would turn a cost control into a database outage. Use alert emails only.
- **PII and residency:**
  - B2 copies are provider-encrypted only (barman-cloud has no client-side encryption; pgBackRest's plugin does, if that's ever needed).
  - Dumps are client-side encrypted.
  - Data leaves India for the Netherlands (B2) and stays in Mumbai for AWS. The privacy notice must name Backblaze and AWS as sub-processors with their locations. DPDP treatment isn't settled here (owner/legal).
  - Every erase copy is bounded by the ≤ 35 d SLA.
- **Restore safety:** recovery sources always use RO keys; before reviving any old image, revoke the RW key; every restore rotates the JWT key, revokes all sessions and suspends BYO keys, and users get a notice to redo security changes made after the restore point.
- **Rehearsal safety:** dummy coach, OAuth and notification credentials; egress deny-list; same-day teardown; logged in the SLA inventory.
- **In-cluster exposure:** anyone who can read Secrets in `databases` can read the backups. T7's RBAC and NetworkPolicy work treats `databases` as tier-0.

## 8. What T2 constrains downstream

- **T4 (judge contract):**
  - canvas bodies and drafts are inline `bytea`; `object_key` is deferred;
  - judge enforces its own body limit (512 KiB scene + 32 KiB export);
  - erase is SQL only in v2.0;
  - the draft sweeper runs at 90 d;
  - re-model WAL with arena submits and drafts against the ~0.5 MB per learner-day assumption;
  - events carry refs only;
  - if judge ever spills to a bucket, it needs an orphan reconcile after PITR, and a 404 means `gone`.
- **T3 (sandbox):**
  - runners get no object-store egress and no S3 credentials;
  - their default-deny must not inherit the `:443` exception;
  - runner scratch is tmpfs and never backed up;
  - the pack never goes to any bucket.
- **T5 (platform AI):**
  - AI prose (≤ 8 KiB) stays inline, covered by PITR and the dump;
  - LLM calls never get presigned URLs or bucket reads;
  - after a restore, coach runs `suspend-byok` and the rehearsal uses a dummy `COACH_MASTER_KEY`;
  - zero-retention calls are disclosed as out of reach of erase.
- **T6 (interviewer):**
  - owns the media bucket in a **separate B2 account**, the presign → complete → verify flow, the 30 d lifecycle, CORS, CSP entries and prefix-hide erase;
  - stores transcripts inline in its own schema;
  - brings Garage into compose and e2e;
  - B2 Amsterdam is ~120 ms from Mumbai (inferred): fine for upload after a session, not for live streaming;
  - still has to name the owning service.
- **T7 (rollout)**, tiered:
  - **Tier 0:**
    1. B2 account (card, 2FA, no caps) and `hack/b2`;
    2. spike 1;
    3. PR 1 (plugin);
    4. PR 2a (secrets, ObjectStores);
    5. PR 2b (plugins, ScheduledBackup, prune annotations; quiet window);
    6. one laptop restore.
  - **Tier 1 (M3 gate):**
    1. age key copies checked, `host-bootstrap.sh`, registrar TTL, Hostinger backup setting;
    2. 20 Gi PR;
    3. `pgstore-backup-check` and healthchecks.io (recommended as T1 opscheck's channel too);
    4. Q2 (AWS account, dump role and CronJob, `dump-ship`);
    5. Q3 lock via `lock.sh`, once the spike passes;
    6. the runbook, `rollback-to.sh`, `drill.sh`;
    7. `RELAY_PAUSED`, the `reseal` subcommands, `revoke-all-sessions`, `suspend-byok`;
    8. **Drill 0**, which gates M3 grading for non-owners.
  - **Tier 2 (step E; NATS auth first):** AWS ledger bucket and keys; identity's intent-first writer, best-effort `cred-ledger`, and `erase-replay`/`reseed`; the `file://` driver; ADR-0028.
  - **Tier 3 (T6):** media.
  - **Cross-cutting:** gateway `MaxBytesReader` + 413; SPA canvas timeout; NetworkPolicy egress exceptions.
  - **ADR-0028** amends:
    - ADR-0027 §3 (blob placement);
    - ADR-0027 §6 (the ledger's home and its ordering; the SLA is ≤ 35 d, not "the window");
    - T1 §3.6 (NATS is rebuilt after *any* restore, in the fixed order above).

## 9. Alternatives considered

| Alternative | Why not |
|---|---|
| MinIO (community, operator, Bitnami, forks, AIStor Free) | Server and operator repos archived; Bitnami frozen; forks are single-maintainer or `:latest`-only (no semver ImagePolicy); AIStor's licence expires and goes offline, with no encryption in Free. The owner's "Postgres + MinIO" direction (`X/docs/v1/feedbacks/README.md:90`) is superseded. |
| Garage, SeaweedFS or RustFS in the cluster | Would hold only disposable audio, which isn't in v2.0. Same failure domain. Adds a new internet-facing parser, an IngressRoute (SigV4 host and path; Traefik's 60 s read timeout), a SOPS rule and RAM. |
| Any in-cluster store as the backup or ledger target | Same disk and credentials; circular (restoring it needs the cluster) |
| Cloudflare R2 for backups | Equal-part multipart rule vs barman's variable parts (PR #1161 unmerged); no versioning; no write-only key |
| **Compliance lock from day 1** (draft) | Account closure removes locked data anyway; a typo can only be fixed by closing the account; risk from the checksum requirement on locked buckets. A second provider is stronger. Compliance stays available as Q3(b). |
| **Ledger in the same B2 account as the backups** (draft) | One account event loses both the backup and its correction record |
| Ledger dual-written to both providers | Two blocking dependencies on erase. Placing the ledger with the second provider, plus re-seeding from `erase_request`, gives the same coverage. |
| Second B2 account for the separate copy | Same vendor: billing, fraud or ToS action could hit both. Offered as Q2(b). |
| Unattended ledger auto-apply (with a count cap) | A planted entry plus an attacker-timed restore could erase ≤ N accounts. Operator approval costs one minute per rare restore. |
| A "security ledger" written by coach for BYO-key deletion | A new credential in coach. Suspending all BYO keys after any restore covers it with no infra. |
| B2 caps as a cost circuit breaker (draft) | Caps are account-wide, and a storage-cap hit blocks uploads, so archiving stalls and the database goes down |
| Long-lived `dr/restore-template` and `dr/rehearsal` branches (draft) | Go stale fast (the infra `main` gets many image-automation commits a day; new roles and Database CRs per service). One snippet applied to current `main` instead. |
| Monthly in-cluster drills; yearly throwaway-VPS rehearsal (draft) | Two PRs and ~1 Gi of node RAM per drill; a red `databases` Kustomization while a drill runs; an unbudgeted second host provider. Laptop drills test the thing that actually fails: can the archive be restored? |
| Exporter scrape + ClusterRole + cert check + ntfy (draft) | Reaches into a tier-0 namespace from `xlearn`, duplicates cert-manager, and needs two alert vendors |
| CI guard in infra against removing the Cluster | infra has no CI today; `prune: disabled` already makes removing it from git a no-op |
| k3s `--secrets-encryption` | Its key sits on the same disk and in the same Hostinger images; no gain on one node |
| Audio in Postgres | ~3.6 GB (36% of the volume) at 100 opt-in learners, plus the same again in incompressible WAL; would need chunking under 1 MiB |
| In-tree `barmanObjectStore` | Deprecated; removed in CNPG 1.31 |
| pgBackRest CNPG-I plugin | Client-side encryption and R2-friendly, but "experimental". Revisit if backups must be unreadable by the provider. |
| `pg_dump` as the *only* backup | No PITR; RPO equals the dump interval. Kept as the second copy only. |
| Longhorn backupTarget, Velero, JetStream snapshots | Duplicate Barman with crash-consistent images and the stale-archive hazard. Git is the source for Kubernetes objects; outboxes rebuild NATS. |
| Hostinger images as *the* backup | Whole-disk only, restore overwrites the VPS, reviving one corrupts the WAL archive, and they hold `sops-age` |
| Crossplane or Terraform for bucket state | A controller for 4 buckets. `hack/b2` and `hack/aws` as code plus a diff at each drill is enough. |

## 10. Owner decisions — resolved 2026-09-24 (they override the backup parts of this document)

| # | Question | Decision |
|---|----------|----------|
| D11 | Object storage in v2.0 | **None, in the cluster or externally, for app blobs.** Canvas scenes and drafts plus transcripts stay **inline in Postgres permanently**; T1's `object_key` is deferred until a trigger fires. MinIO is rejected (both repos archived). Audio is T6's call. |
| D12 | Off-node backups (Q1–Q4 above) | **Deferred: no backups for now.** Q1–Q4 were not taken. The owner's **second VPS** is the planned future target. §3 of this document stays as the **ready design**: CNPG Barman Cloud Plugin, lineage per restore, post-restore sequence, drills, #828 monitoring. It gets re-targeted at an S3 endpoint on that VPS (e.g. Garage v2.4 single-node, §2) or at a third-party provider when backups land. **No backup gate** on opening to other learners. The erase ledger is deferred with backups (erase = delete + tombstones). Until then, **Hostinger's weekly VPS images are the only safety net**: keep them on, protect the Hostinger login with 2FA, and remember they hold plaintext Secrets. |
| D13 | Opening to other learners | **Controlled, invite-only signup, for capacity reasons (not backups).** Mechanism is designed in T7. |

Carried into T7 regardless (cheap, and more important with no backups):
- `kustomize.toolkit.fluxcd.io/prune: disabled` on the CNPG Cluster and its Namespace, plus the "never revert-re-add a deleted Cluster" rule;
- the NATS rebuild from outboxes after any restore;
- disk-usage checks in opscheck;
- the gateway `MaxBytesReader` (typed 413) for canvas bodies.

## 11. Risks

| Risk | Mitigation |
|---|---|
| **Plugin #828:** "Expected empty archive" stops archiving after a restart or switchover. Open; fix PR #843 closed unmerged; reproduced on CNPG 1.30 + plugin v0.14.0 + PG 18 | Hourly `ContinuousArchiving` check with `/fail`; the Drill 0 restart assertion, repeated after every CNPG or plugin bump; a lineage per restore; **a pre-written PR adding `cnpg.io/skipEmptyWalArchiveCheck: enabled` to the Cluster** (reported to clear it within seconds). It disables the empty-archive guard, which leaves "revoke RW key and recover into a new lineage" as the only protection. |
| B2 fails with barman (multipart, checksums on locked buckets, `deleteFiles`, region signing) | Spike 1 with pre-decided fallbacks: `deleteFiles` + M3 lock; switch the primary to AWS only if backup or restore fails |
| Archiver stall → `pg_wal` fills → outage (causes: #828, credentials, cap, provider outage) | Hourly check, 20 Gi, no caps, card on file, runbook entry "cap hit = outage in ~2–4 d" |
| Account-level loss at B2 (closure, suspension, takeover) | The AWS copy and ledger (Q2); hardware 2FA; distinct email |
| Backup poisoning by overwrite | Version audit at drills; M3 lock plus `rollback-to.sh` into a fresh lineage; the separate dump |
| Wrong restore order (discarded-timeline events consumed; `DeliverNew` durables skipping the republish) | The single post-restore sequence, exercised in the pre-signup rehearsal |
| PITR revives revoked sessions, reverted passwords and deleted BYO keys | JWT rotation, revoke-all, `suspend-byok`, `cred-ledger` replay, user notice |
| A GitOps mistake deletes the Cluster; a revert re-inits an empty database | `prune: disabled` on the Cluster, Namespace, ObjectStores and Secrets; the "never revert-re-add" rule; PV re-bind rehearsed |
| Node compromise leaks every secret, including the one protecting BYO keys in the backups | The separate compromise runbook (new keys first, then revoke, then rotate everything, users rotate provider keys) |
| Planted or orphaned ledger entries | Server timestamps, operator-approved dry run, drift audit |
| Hostinger images hold `sops-age` and learner data outside the SLA | Owner check: turn them off, or apply tier-0 protection and fit retention inside the SLA |
| Barman 3.20 with PG 18.4 unproven here; silent page corruption carried into backups | Drill 0 gates M3; `pg_amcheck --heapallindexed` and a `data_checksums` assertion in every drill |
| Two providers add ops for one person | The second provider is Tier 1 and optional (Q2c). `hack/aws` is scripted. It doubles as the pre-built fallback. |
| Retired lineage never expires → SLA breach | Retired-lineage rule is a checklist item in every restore PR; config diff at every drill |
| Provider state drifts outside GitOps | `hack/b2`, `hack/aws` as code; diff at every drill |
| A plugin upgrade interrupts archiving (Recreate-only chart) | Never bump it in the same PR as a Postgres restart; drill after a minor bump |
| B2 outage blocks erase (intent-first returns 503) | Accepted: erase is rare and can be retried; correctness wins |
| Residency (India → NL) and DPDP treatment unverified | Q1(b) or Q2(a) keep a copy in India; disclose sub-processors |

## 12. Critic findings addressed

**Resilience and security critic (6/10)**

| # | Finding (severity) | Resolution |
|---|---|---|
| A1 | The compliance-lock claim was false (closing the account removes locked data); a single account and a single tool (major) | **Fixed.** Claim corrected in §3, §7 and §10. Added the age-encrypted `pg_dump` copy and moved the ledger to a second provider (Q2); hardware 2FA and a distinct email. |
| A2 | Plugin #828 archiving stall not mentioned (major) | **Fixed.** Risk added. Drill 0 and post-bump restart assertion; **hourly** check with `/fail`; pre-written `skipEmptyWalArchiveCheck` PR with its trade-off recorded. Verified: #828 open (updated 2026-09-09); #843 closed unmerged. |
| A3 | Replay trusted the payload's `erased_at`; "earliest" could be gamed; T undefined (major) | **Fixed.** Replay uses the earliest **server version timestamp**; defined T_restore (targetTime, else max outbox `created_at`); window classes; **operator-approved dry run** instead of any unattended auto-erase; drift audit; §7 reworded. |
| A4 | Restore ran in the wrong order (backends consuming stale streams; replay before the NATS rebuild) (major) | **Fixed.** One post-restore sequence: all at 0 → delete streams → consumers with `RELAY_PAUSED` → check durables → reseal → unpause → replay → reconcile → reopen. |
| A5 | A prune or revert trap re-inits an empty database (major) | **Fixed.** `prune: disabled` on the Cluster, Namespace, ObjectStores and Secrets; the "never revert-re-add" rule; PV re-bind rehearsed at Drill 0. The CI guard isn't adopted: infra has no CI, and the annotation makes removal a no-op. |
| A6 | Account-wide B2 caps can stall archiving (major) | **Fixed.** Card on file; no caps, alerts only; media in a separate account; cap-hit entry in the runbook. |
| A7 | Overwrite and hide poisoning; no rollback tool (major) | **Fixed.** `rollback-to.sh` into a fresh lineage (drilled once); version/hide audit at each drill; locked originals from M3. The audit isn't run hourly in the cluster: that would need an S3 client plus the RO key in the check, and whoever holds `pgstore-archive-rw` already holds the live database. |
| A8 | PITR rolls back security state (major) | **Fixed.** JWT rotation, revoke-all-sessions, `suspend-byok`, best-effort `cred-ledger` replay, user notice. No coach ledger: suspending BYO keys covers it. |
| A9 | Compromise response too narrow (major) | **Fixed.** Separate "node compromised" runbook: new age key and new keys first, then revoke; rotate everything; users rotate provider keys. k3s secrets-encryption declined, with the reason. |
| A10 | `deleteFiles` ambiguity (minor) | **Fixed.** B2 doc (updated 2025-12-11) says DeleteObject(s) should have both, so the plan assumes it is likely needed. Spike 1 tests both keys; the fallback is pre-decided; §7 reworded. |
| A11 | Lock shorter than the window; ledger protected more weakly than the backups (minor) | **Fixed.** Lock = 30 d = the window. Ledger on AWS with **compliance 40 d** (harmless, since entries are never deleted). Spike added for lifecycle deletion after a lock lapses. |
| A12 | Drill 0 was same-node only; no page checks (minor) | **Fixed.** Drill 0 adds a laptop k3d rebuild from zero using only the offline age key and the RO key; `pg_amcheck --heapallindexed` and a `data_checksums` assertion in every drill. |
| A13 | `ledger-ro` placement contradictory (minor) | **Fixed.** Password manager only; added by the restore PR and removed at teardown. |
| A14 | Surgical restore left durables and downstream state inconsistent; the PITR copy lived a week (minor) | **Fixed.** Durables reset to `DeliverByStartTime` T, then a downstream reconcile; the copy is deleted within 24 h and listed in the SLA inventory. |
| A15 | Rehearsal PII and side effects (minor) | **Fixed.** Laptop k3d by default; dummy coach, OAuth and notification credentials; egress deny-list; same-day teardown. |
| A16 | Hostinger images are tier-0 (minor) | **Fixed.** Stated in §3 and §7; owner check (turn off, or 2FA + SLA fit + on the rotation list). |

**Solo-dev cost and YAGNI critic (7/10)**

| # | Finding (severity) | Resolution |
|---|---|---|
| B1 | "$0" assumed the free tier's hard cap (major) | **Fixed.** Card on file from day 1; cost restated as "$0 while under 10 GB; A/B/C calls free for pay-as-you-go". |
| B2 | A compliance lock is one-way, over-built for 1 account (major) | **Fixed.** No lock at Tier 0. From M3, a **governance** lock only through `lock.sh`, which asserts the value. Compliance is Q3(b), with the account-closure consequence spelled out. |
| B3 | Too much machinery before the first backup (major) | **Fixed.** Tiers 0–3. Tier 0 is a one-weekend backups MVP. |
| B4 | Drill programme heavier than one person will sustain (major) | **Fixed.** Routine drills on the laptop (`drill.sh`); in-cluster only at Drill 0 and on CNPG/plugin minor or PG major bumps; one rehearsal before signup, not a yearly one on a throwaway VPS. |
| B5 | DR branches go stale; two runbooks (major) | **Fixed.** No branches; one runbook in I/ with a snippet applied to current `main`, checked at in-cluster drills; the X/ file holds only the subcommands. |
| B6 | Monitoring spread wide; ClusterRole into tier-0; two vendors (major) | **Fixed.** Two conditions from CRs, from inside `databases`, namespaced Role; healthchecks.io as both dead-man and push; exporter, cert check, ClusterRole and ntfy dropped. **Partly overridden:** hourly instead of daily, because of #828 (same cost). |
| B7 | Ledger over-built (lock, permanent RO key, Garage) (major) | **Mostly adopted.** `ledger-ro` out of the cluster; `file://` driver plus `gofakes3` instead of Garage (Garage waits for T6); built at step E. **Lock kept** as compliance 40 d on AWS because critic A needs the ledger to be at least as durable as the backups, and with entries never deleted it costs nothing. |
| B8 | PR 2 bundled several changes onto one restart (minor) | **Fixed.** Split into 2a (secrets, ObjectStores; no restart) and 2b (plugins, ScheduledBackup; quiet window, revert ready); 20 Gi in its own PR; `wal_compression` dropped. |
| B9 | `crds:` fields do nothing; Recreate strategy; `cnpg-operator` `dependsOn` (minor) | **Fixed.** Verified in the chart 0.8.0 tgz. `crds:` removed; Recreate noted with the rule never to bump alongside a restart; `dependsOn` on the cert-manager HelmRelease at HelmRelease level; the operator Kustomization unchanged. |
| B10 | R2 control spike pointless (minor) | **Fixed.** Dropped; the issues and doc are cited instead. |
| B11 | Egress "unverified"; egress numbers inconsistent (minor) | **Fixed.** KVM 4: 16 TB/month, weekly backups included. Egress table recomputed at the daily base (~220 GB at 1,000 learners); Hostinger backups added as an owner check. |

**Claims the critics verified as wrong, all corrected:**
- "The compliance lock means nobody can destroy the last 14 days" (§3, §7, §10).
- "A planted entry never auto-erases" (§7; replay redesigned).
- #828 was not mentioned (§11).
- Caps as a safe circuit breaker (§7, §9).
- `deleteFiles`-less retention stated as fact (§3, spike 1).
- `install/upgrade.crds` fields (§5).
- "$0" (§1, §6).
- Bandwidth "unverified" (§6).
- Egress table inconsistent (§6).
- Also found here: the draft listed the SOPS age key as "never in cluster", but it is in the cluster as `sops-age` (§5, §7).

**Kept from the draft (both critics endorsed):**
- no in-cluster store; MinIO and R2 rejections;
- the lineage-per-restore rule; the RO ObjectStore; `method: plugin` on the ScheduledBackup; the 6-field cron;
- the old-image hazard rule;
- the intent-first ledger with a 503; gateway at 0 until replay;
- the mandatory NATS rebuild and the `DeliverNew` hazard;
- inline `bytea`; bucket per owner; the presigned-upload verification design;
- the age key as root of trust; one backup mechanism, not three;
- `hack/b2` as code with a drift diff.

---

**Sources** (all seen 2026-09-24 unless noted)
- R2 equal-part multipart rule (updated 2026-07-29): https://developers.cloudflare.com/r2/objects/multipart-objects/
- GitHub API (`gh api`):
  - barman PR #1161: open, unmerged, updated 2026-06-10. https://github.com/EnterpriseDB/barman/pull/1161
  - barman#954: closed as not planned. https://github.com/EnterpriseDB/barman/issues/954
  - plugin#411: open. https://github.com/cloudnative-pg/plugin-barman-cloud/issues/411
  - **plugin#828: open, updated 2026-09-09; the `skipEmptyWalArchiveCheck` workaround is reported there.** https://github.com/cloudnative-pg/plugin-barman-cloud/issues/828
  - PR #843: closed 2026-04-15, not merged.
  - Releases v0.15.0 (2026-09-03) and v0.14.0 (2026-07-29): https://github.com/cloudnative-pg/plugin-barman-cloud/releases
- CNPG chart index (`plugin-barman-cloud` 0.8.0 → v0.15.0): https://cloudnative-pg.github.io/charts/index.yaml. I inspected the chart tgz: CRD template gated by `crds.create` with `helm.sh/resource-policy: keep`; the values.yaml warning that only Recreate is supported; ObjectStore status `serverRecoveryWindow.{firstRecoverabilityPoint,lastSuccessfulBackupTime}`; data compression enum without zstd.
- CNPG 1.30 release notes (in-tree Barman removed in 1.31): https://cloudnative-pg.io/docs/1.30/release_notes/v1.30/
- CNPG labels and annotations (`kubectl.kubernetes.io/restartedAt`): https://cloudnative-pg.io/documentation/1.26/labels_annotations/
- B2 pricing ($6.95/TB-month; first 10 GB free; 3× free egress; A/B/C free for pay-as-you-go; no minimum duration): https://www.backblaze.com/cloud-storage/pricing
- B2 S3-compatible app keys (updated 2025-12-11: DeleteObject(s) should include both `writeFiles` and `deleteFiles`; bucket-restricted keys need `listAllBucketNames`): https://www.backblaze.com/docs/cloud-storage-s3-compatible-app-keys
- B2 Object Lock ("If you locked your file for longer than you intended, you need to close your Backblaze B2 account"; can be enabled on existing buckets; lifecycle deletion fails on locked files): https://www.backblaze.com/docs/cloud-storage-object-lock
- B2 lifecycle rules (updated 2026-02-05; minimum 1 day; runs once a day; locked versions not deleted): https://www.backblaze.com/docs/cloud-storage-lifecycle-rules
- B2 app keys: https://www.backblaze.com/docs/cloud-storage-application-keys · S3 DeleteObject: https://www.backblaze.com/apidocs/s3-delete-object · hide file: https://www.backblaze.com/apidocs/b2-hide-file · regions: https://www.backblaze.com/docs/cloud-storage-data-regions
- B2 caps block uploads: https://kb.msp360.com/cloud-vendors/backblazeb2/unable-to-upload-files-to-b2-due-to-exceeded-storage-cap · https://help.backblaze.com/hc/en-us/articles/217931138-How-to-use-B2-data-caps-alerts
- R2 pricing (updated 2026-08-07): https://developers.cloudflare.com/r2/pricing/
- AWS S3 pricing, Mumbai Standard ≈ $0.023–0.025/GB-month (approximate, from search results; confirm at signup): https://aws.amazon.com/s3/pricing/
- Hostinger KVM 4 (4 vCPU / 16 GB / 200 GB NVMe / 16 TB bandwidth / weekly backups): https://www.hostinger.com/vps-hosting · https://www.whtop.com/plans/hostinger.com/40624
- healthchecks.io free tier (20 jobs): https://healthchecks.io/pricing/
- Flux prune annotation: https://fluxcd.io/flux/components/kustomize/kustomizations/ · https://fluxcd.io/flux/faq/
- MinIO repos archived (GitHub API, via the draft and the critic): https://github.com/minio/minio · https://github.com/minio/operator
- VPS location (Mumbai, AS47583): https://ipinfo.io/187.127.130.97
- Repo facts checked (read-only):
  - `X/internal/gateway/bff.go:164` (`io.LimitReader(r.Body, 1<<20)`)
  - `X/web/src/lib/api.ts:12` (15 s)
  - `X/internal/assessment/service.go:109` and `X/internal/review/service.go:135` (`WithDeliverNew`)
  - `X/internal/platform/events/relay.go`; `X/internal/identity/password.go` (password auth exists)
  - `I/infrastructure/database/cluster/cluster.yaml` (no `spec.backup`; `primaryUpdateStrategy: unsupervised`)
  - `I/infrastructure/database/README.md:122-134`
  - `I/clusters/vps/databases.yaml` (`prune: true`, `wait: true`, stale comment)
  - `I/infrastructure/storage/release.yaml:35` (`reclaimPolicy: Retain`)
  - `I/infrastructure/controllers/cert-manager.yaml` (HelmRelease `cert-manager`/`cert-manager`)
  - `I/.sops.yaml` (two rules)
  - `I/apps/image-automation.yaml` (update path `./apps` only)
  - `I/charts/project/values.yaml:29,36-37,93`
  - `I/apps/xlearn-identity.yaml` (`podAnnotations` rotation)
  - infra has no `.github/workflows`