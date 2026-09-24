# ADR-0028 — Object storage & off-node backups for v2.0 (none yet; design ready)

- **Status:** Proposed **§4's opscheck watch amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) (D34: no alerting in v2).**
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** amends [0027](0027-content-evalpack-and-user-data-model.md) §3 (blob placement) and §6
  (erase ledger); [0005](0005-data-ownership-and-migrations.md) (single-instance note, no backups).
  v2 topic T2: [feasibility § T2](../v2/feasibility.md#t2--object-storage--off-node-backups) and the
  [research appendix](../v2/research/t2-object-storage-backups.md), which holds the ready backup design.

## Context

T1 (ADR-0027) reduced xLearn's blob needs to three things:
- canvas scenes and drafts (≤ 512 KiB);
- interview transcripts (≤ 256 KiB);
- opt-in audio (~8 MB, 30 days; T6).

T1 also assumed CNPG point-in-time backups and an erase ledger stored outside them.

Today nothing leaves the node:
- CNPG `spec.backup` is unset.
- Longhorn has no backup target.

Findings (2026):
- **MinIO:**
  - The community server repo was archived on 2026-04-25 and ships source only.
  - The operator was archived on 2026-03-20.
  - Bitnami's charts are frozen.
  - AIStor Free is proprietary: its licence expires, and it has no encryption at rest.
- **In-cluster stores (Garage, SeaweedFS) can never be the backup here.** They share the node's single disk.
- **Cloudflare R2 is unfit for Barman base backups.** It requires equal-size multipart parts.
- **Backblaze B2 plus the CNPG Barman Cloud Plugin would cost about $0** at this scale.
- **Hostinger's weekly VPS images** (included in the KVM 4 plan) copy the whole disk, including k3s's plaintext Secrets.

## Decision

### 1. No object store in v2.0

- Run **no object store in the cluster** and add **no external bucket for app data** in v2.0.
- **Canvas scenes and drafts and interview transcripts stay inline in Postgres permanently.** They are stored as `bytea` with lz4 TOAST, in their owner's schema.
- ADR-0027's nullable `object_key` is **deferred**. It lands only when a trigger fires: judge body bytes exceed 1 GiB or 25% of the Postgres volume, or a payload kind larger than 1 MiB appears.
- **Audio** is the only real object-storage case. Where it lives, if it is stored at all, is decided by **T6**.
- **MinIO is rejected** in every form.

### 2. Off-node backups deferred

- **No off-node backups for now (owner decision).** The planned future target is the owner's **second VPS**.
- The T2 appendix keeps a **ready design**, to be re-targeted at an S3 endpoint on that VPS (e.g. Garage single-node) or at a provider chosen then:
  - CNPG Barman Cloud Plugin (HelmRelease), a read/write and a read-only ObjectStore, and a new lineage per restore;
  - a ScheduledBackup with `method: plugin`, a 5-minute RPO, and a window to be chosen then;
  - the mandatory post-restore sequence: scale to zero, rebuild NATS from outboxes, replay the ledger, reset security state, reopen;
  - drills and #828 monitoring.
- **No gate** on opening xLearn to other learners is tied to backups. **Opening is controlled by invite-only signup for capacity reasons** (mechanism: T7).

### 3. Amendments to ADR-0027

- **§3:** blobs stay inline permanently; `object_key` is deferred (above).
- **§6:** the **erase ledger is deferred with backups.** Until backups exist, erase is delete plus outbox delete plus tombstones in every service. The ledger, its replay and the backup-erase SLA arrive with the backup ADR.
- **T1's "CNPG backups gate grading for non-owner learners" is withdrawn.** Controlled signup replaces it.

### 4. Kept regardless (cheap, and more important with no backups)

- `kustomize.toolkit.fluxcd.io/prune: disabled` on the CNPG `Cluster` and its Namespace, and never revert a deleted Cluster back in, because it would `initdb` an empty database.
- Outboxes remain the durable event log. NATS is rebuilt from them after any restore.
- **Hostinger's weekly images stay enabled** as the only safety net. The Hostinger login gets 2FA and is treated as top-tier sensitive, because the images hold `sops-age` and every Secret.
- `opscheck` watches disk and volume usage. Postgres and NATS PVCs are resized at 60%.

## Consequences

- ✅ Zero new infrastructure, zero cost, and no new stateful service or credentials for v2.0.
- ✅ Erase stays pure SQL, and every blob is consistent with its row.
- ✅ When backups land, the design is ready and only needs a target and window.
- ⚠️ **A node or disk loss loses all learner data since the last Hostinger weekly image**, up to about 7 days. Restoring an image overwrites the whole VPS and carries every Secret. The owner accepts this risk for v2.0.
- ⚠️ No point-in-time recovery, so a bad migration or operator mistake cannot be rewound except to the last weekly image.
- ⚠️ If audio is stored, T6 must pick a home without a bucket (options: Postgres chunks, the second VPS, or not storing audio).

## Alternatives considered

| Option | Why not (now) |
|---|---|
| B2 Amsterdam primary + AWS Mumbai second copy + governance lock (the T2 recommendation) | Owner deferred backups. Kept as the ready design. |
| MinIO (community, operator, Bitnami, AIStor Free) | Archived or frozen, or proprietary with a licence that expires. |
| Garage or SeaweedFS in the cluster | Same disk as the data; would hold only disposable audio, which is not in v2.0. |
| Cloudflare R2 for backups | Its equal-part multipart rule conflicts with Barman's uploads (fix unmerged). |
| Audio or large blobs in Postgres | About 36% of the volume at 100 opt-in learners, plus WAL. T6 decides. |
| Longhorn backup target, Velero, JetStream snapshots | Same-disk or crash-consistent copies. NATS is rebuilt from outboxes anyway. |
