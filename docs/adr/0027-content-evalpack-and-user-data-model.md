# ADR-0027 — Content, private eval pack & per-user data model

- **Status:** Accepted (2026-09-24, v2 build-plan sign-off). **§3 (blob placement) and §6 (erase ledger) amended by [ADR-0028](0028-object-storage-and-backups.md) (Accepted 2026-09-24; folded into §3 and §6 below):** blobs stay inline in Postgres permanently; the erase ledger is deferred with backups; the "backups gate non-owner grading" rule is withdrawn in favour of invite-only signup. **Amended by [ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) (T7, D34, no alerting in v2; Accepted 2026-09-24; folded inline in §3 and Consequences):** the `opscheck` PAT-expiry alert becomes an evalpack-CI plus manual check, and the `XLEARN_COACH` stream raises the Σ `MaxBytes` budget to 3.375 GiB (§3, Consequences). **D35 (T7):** §8's "v2.0 content scope" (D6) is the scope of the v2.x line, delivered in waves; `v2.0.0` ships the 14 pilot packs and the pilot course ([ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §1.1).
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0026](0026-per-course-extensibility-model.md) (the frame this fills in),
  [0005](0005-data-ownership-and-migrations.md), [0012](0012-curriculum-content-model-and-seeding.md),
  [0018](0018-progress-projection-grain-and-rebuild.md), [0024](0024-public-user-dashboards-and-usernames.md),
  [0025](0025-public-profiles-under-u-prefix.md). v2 topic T1: [feasibility § T1](../v2/feasibility.md#t1--content--data-model)
  and the full [research appendix](../v2/research/t1-content-data-model.md).

## Context

ADR-0026 made a course **data** and a capability **code**, with a private eval pack that only judge reads. T1 decides where every piece of authored content and per-user data lives, and who owns it.

Constraints:
- The xlearn repo and its GHCR images are **public**; infra is private (SOPS/age).
- There is one CNPG instance (10 Gi, 1 Gi memory limit), with **no backups yet**.
- NATS has a 5 Gi store and no auth.
- There is one author for about 345 items across six courses.

Problems found in v1:
- It copies LeetCode's Example 1 for Two Sum and 3Sum.
- `pattern` is ungated on several surfaces: list/bulk, the Revision card, the arena, and the coach prompt.
- The public route mints an ordinary learner token that no service checks.
- There is no account erase at all.

## Decision

### 1. The public/private rule

**If the problem statement would say it, it is public. If it encodes an answer that is not published anywhere else, it is private.**

| Tier | Contents | Enforcement |
|---|---|---|
| **Public** | Manifests, items, statements, constraints, samples, the function signature, reference code, stage-gated hints and editorial (merged only once **owner-stamped**), "solution facts", asset descriptors, rubric definitions, media | The item schema has **no field that can hold an answer**. Strict decoding, a filename denylist, an embedded-file allowlist, and a local pre-push fingerprint hook |
| **Public, withheld while live** | `pattern`/topic, hint and solution stages, solution facts, while the item has an open counted attempt or a due touch | **One gateway `withhold()` applied on every surface**: list, bulk, week, Today, the revision queue, mistakes, the arena, and coach mode and prompt. This is a nudge, not secrecy |
| **Private (judge only)** | Hidden cases, with expected outputs computed by running the reference; keys that cannot be derived from public text; SQL hidden instances and expected result sets; rubric anchors, must-cover lists and exemplars | The eval pack (§2) |

**Grades carry a trust label.**
- **`honor`**: public-key probes (e.g. DSA pattern and complexity) and in-process test profiles (go-concurrency, LLD facade tests).
- **`checked`**: everything else.
- Stats show a "judge-checked %" that excludes honor grades.

### 2. Where content lives and how it ships

- **Public content** lives in this repo at `curriculum/courses/<slug>/items/<id>/`.
  - Each item has an `item.json`, Markdown sidecars and `_code/`.
  - It is embedded with `all:courses` into every image at the release tag.
  - curriculum seeds items, sections, weeks and concepts for the read API.
  - curriculum serves manifests, rubrics, assets and media **from the embedded files**, with no tables for them.
  - practice and judge read their own embedded copy.
- **Private eval pack** lives in a fresh private repo, **`xlearn-evalpack`**.
  - Its CI validates the pack against public `main`.
  - It then pushes a **private, `FROM scratch` data image to GHCR**.
  - Flux image automation bumps it (pulled by a machine user, with the SOPS pull secret).
  - **Only judge** mounts it, read-only, as a **Kubernetes image volume** behind `EVALPACK_DIR`.
  - Fallbacks: an initContainer copy, then ORAS.
  - **Spike before M3:** confirm the mount works, and that a pod without the pull secret cannot use the cached image.
  - **Postgres holds no pack bytes.** judge keeps an in-memory index and reads cases lazily.
  - If the pack is bad, judge stays Ready with zero evaluable items.
- **Version skew is controlled by one narrow `contract_hash` per item.**
  - It covers only what hidden data depends on: part ids and types, signature, harness, checker, option ids, asset and rubric `id@v`.
  - The pack lists `accepts_contract_hashes[≤2]`.
  - On a mismatch, the item falls back to self-report with a "grading pending" badge.
  - A stale pinned hash at submit time returns `inconclusive(contract_changed)`.
  - There is no public-ref pin, bump bot or release lockstep.
- **Hashes** are sha256 over a canonical re-marshal of the typed Go structs, computed by one shared package.
  - `policy_version`: the manifest.
  - `content_hash`: the whole public item.
  - `contract_hash`: as above.
  - `item_hash` and pack digest: recorded on every evaluation.

### 3. Storage placement (no new database engine)

| Data | Owner | Storage |
|---|---|---|
| Content rows (item, section + `language`, week, phase, concept keyed `(path_slug, slug)`) | curriculum | relational + TEXT (+ `spec` JSONB ≤ 64 KB from M2a) |
| Attempt (`purpose` course \| touch, state, grade, `passed`, `graded_by`, `trust`, `criteria`, `stage_params`, `policy_version`, `contract_hash`) | practice | relational + JSONB |
| Submission (**counted** contexts course, touch and mock, **plus arena submits**) + parts; one draft per key; evaluation, including internal per-test results; feedback prose (≤ 8 KiB) | **judge** (new schema, role `xlearn_judge`) | relational; bodies inline `bytea` (lz4), with a nullable `object_key` for T2 blob kinds; bodies are validated in Go (no fallible CHECKs that would log learner code) |
| Run payload and output | judge `job` (T4) | job row only, deleted 24 h after completion |
| Touch result (+`graded_by`, `trust`, `evaluation_id`, `attempt_id`, `criteria`); mistakes, weak area | review | relational (+ `path_slug`) |
| Mock session (+`rubric_snapshot`, `total`/`max_total`, `scored_by`) + `mock_session_item`; projections | assessment | relational + JSONB (+ `path_slug`) |
| Profile and course visibility; erase request; released username (hashed, 60-day cooldown) | identity | relational |
| **Blobs (T2):** canvas scenes and drafts, interview transcripts, audio (video is not stored) | judge / T6 owner | object storage, key `u/<account>/<kind>/<id>` |

- **`path_slug` goes on every per-user row and event.**
- **Outboxes are kept forever** as the durable event log. A lost NATS volume is rebuilt with `sent_at = NULL`, so JetStream needs no snapshots.
- **Growth** is about **86 KB per active learner-day**. The resize trigger is 60% of the PG or NATS volume, about 285 learner-years; decide pruning then.
- **Stream limits must fit the 5 Gi store.** JetStream reserves each stream's `MaxBytes` against it, so the stream **`MaxBytes` sum is budgeted to ≤ 3.25 GiB** and checked by a test. Replay streams use `Discard=New`. _(T7 / ADR-0035: an `XLEARN_COACH` stream (128 MiB) for coach's erase ack makes the sum **3.375 GiB**; the test asserts the 3.75 GiB ceiling, 75% of the store.)_

> **Amended 2026-09-24 by [ADR-0028](0028-object-storage-and-backups.md) §1 and §3 (accepted at the v2 build-plan sign-off). No object store in v2.0.**
> - The **Blobs (T2)** row above no longer applies. Canvas scenes and drafts and interview transcripts stay **inline in Postgres permanently**, as `bytea` with lz4 TOAST, in their owner's schema.
> - The nullable `object_key` is **deferred**. It lands only when a trigger fires: judge body bytes exceed 1 GiB or 25% of the Postgres volume, or a payload kind larger than 1 MiB appears.
> - Audio was left to T6 ([ADR-0032](0032-realtime-ai-mock-interviewer.md)).
> - Growth and the 60% resize trigger are unchanged.

### 4. Problems arena

- **Arena Submits are kept as a submission history.** Runs are not.
- **The arena has its own done marker, separate from course done.**
- A course attempt starts its **server** timer when the attempt starts and shows the full question plus the coding area.
- The arena has an optional **manual, client-side timer, off by default**.
- Arena submits and markers **never** count toward course progress, grades or learning signals, and are learner-private in v2.0.
- `key` and `ai_rubric` parts are never evaluated in arena or Run contexts, so the arena cannot be used to brute-force keys.

### 5. Leak and integrity controls (preconditions for M3)

- **NATS auth plus a `messaging` NetworkPolicy before M3 and before erase.**
  - Per-service credentials; publish only on `xlearn.<svc>.>`.
  - Stream purge and delete only for an ops identity.
- **Learner-facing Evaluation DTO allowlist.**
  - Hidden cases report only the **passed count and the first failure class** (WA/TLE/RE/MLE), with bucketed usage.
  - No case ids, output or per-case timing.
- **Pack material enters only platform-key scoring calls**, with retention off.
  - BYO-key scoring (e.g. the T6 interviewer) sees public rubric descriptors only, labelled `ai-byo`.
- **The public profile gets an enforced `public-read` role.**
  - A new assessment `GET /public/stats` accepts only that role.
  - `/progress/*` rejects it.

### 6. Account erase (v2.0)

`DELETE /api/me` (re-auth + typed confirm) runs as follows:
1. identity deletes the account in one transaction and emits `account_erasure_requested`.
2. Each owning service re-verifies with identity, then deletes its rows **and its outbox rows** and writes a tombstone.
3. judge also deletes the account's `u/<account>/` objects.
4. An **erase ledger** stored outside the CNPG backup is replayed after any restore.

Notes:
- Pseudonymous NATS events stay; a re-seal runbook (purge, then republish from the outbox) is available on request.
- The backup PITR window is the erase SLA.

> **Amended 2026-09-24 by [ADR-0028](0028-object-storage-and-backups.md) §3 (accepted at the v2 build-plan sign-off). The erase ledger is deferred with backups (D12).**
> - Until backups exist, erase is **delete plus outbox delete plus tombstones in every service**. Steps 1 and 2 stand.
> - Step 3 has nothing to delete, because there is no object store (§3 as amended).
> - Step 4 (the ledger and its replay) and the "backup PITR window is the erase SLA" note arrive with the backup ADR.
> - **T1's "CNPG backups gate grading for non-owner learners" is withdrawn.** Invite-only signup (D13, [ADR-0033](0033-invite-only-admission-and-owner-admin.md)) replaces it.
> - With no ledger, a Hostinger snapshot restore brings back accounts erased after the snapshot. [ADR-0034](0034-v2-release-labelling-gating-and-rollback.md) §4.2 therefore lists those erases before a restore and re-runs them through the CLI afterwards.
> - The re-seal runbook uses the offline NATS `ops` identity ([ADR-0035](0035-v2-operations-nats-auth-limits-capacity.md) §2).

### 7. Public profile deltas

- **Scope:** per course, only for **enrolled, visible, active** courses.
- **What each course shows:**
  - concluded vs passed vs total;
  - difficulty split and topic mastery;
  - first-attempt grade mix **with provenance** (self/auto/ai), plus judge-checked %;
  - counted submits, first-submit acceptance and languages;
  - touch pass rates;
  - mocks as a percentage.
- **Defaults:** profile public; each course per its manifest default (**behavioral hidden**).
- **Header totals** (solved, streak, heatmap, mocks) come **from visible courses only**. A toggle bumps the cache epoch.
- **The heatmap becomes `proj_activity`** (attempts, touches, mocks). This fixes v1's +6 spike on a first clean solve. It is backfilled via `touch_scored`, then the projections are replayed at M2b.
- **Never public:** code, answers, feedback prose, mistakes, notes, transcripts, per-item timelines.

### 8. Authoring and rights

- **Package format** (Kattis/Polygon semantics adapted to function-signature problems):
  - **public item:** statement, constraints, samples, signature, reference Go, solution facts, 2–3 revision probes;
  - **private pack:** generated cases, `tests.lock`, and wrong solutions with expected verdicts.
- **CI gates:** the reference agrees with a brute oracle; wrong solutions get their declared verdicts; inputs are validated from `constraints[]`; `tests.lock` reproduces; leak lints; the stamp gate on hints and editorial.
- **AI may draft statements (from the brief only), generators, wrong solutions and anchors. It never writes expected outputs, and keys must be owner-confirmed.**
- **v2.0 content scope** _(T7, D35: the scope of the v2.x line, delivered in waves; `v2.0.0` ships the 14 pilot packs and the pilot course)_:
  - all **151 DSA items at the self-path tier**;
  - **full packs for weeks 1–4** (about 35 items);
  - a ~10-item second-course pilot after M3.
  - That is about 180–285 h. M5 waits until every live DSA item has a pack.
- **Rights:** statements are written from scratch; LeetCode appears only as an outbound link and is never fetched; `provenance` is required on every item; the two copied v1 examples are replaced in M1a.
- **Licence:** **MIT for everything** (content included). The pack is proprietary.

## Consequences

- ✅ A slip into the public repo is the only way hidden data leaks, and it is guarded at four layers. There is no always-public ciphertext.
- ✅ No new database engine, no pack bytes in Postgres, and no artefact GC. T2's blob scope is small: canvas, transcripts, audio.
- ✅ The self-path tier lets every DSA item work in v2.0 while test packs catch up. Upgrading an item from self-report to judge grading needs no id or route change.
- ⚠️ **A second repo, a machine user and a PAT to rotate.** A PAT expiry combined with image GC blocks judge from starting. The kill switch keeps self-report working. _(T7 / ADR-0035, D34: there is no `opscheck` in v2, so no alert. The PAT expiry is a manual owner check (its date is recorded in `docs/v2/status.md`), and the evalpack repo's CI runs the anonymous-GET probe.)_
- ⚠️ **Honor-grade results exist** (public-key probes, in-process tests). They are labelled everywhere, so they cannot inflate "checked".
- ⚠️ **NATS auth becomes a hard M3 precondition** (owned by T7).
- ⚠️ **Content is the critical path:** about 180–285 h (±40%) on top of engineering.
- ⚠️ **Integrity moves from DB CHECKs to loader and CI validation.** The row-snapshot test and the registry lint are mandatory.
- ⚠️ **The arena history adds growth** beyond the 86 KB/learner-day model. Re-model in T4; it stays well inside the resize budget.

## Alternatives considered

| Option | Why not |
|---|---|
| SOPS-encrypted pack in the public repo | The ciphertext is public forever; one key leak exposes every version; diffs cannot be reviewed. |
| Pack from T2 object storage | Couples M3 to T2 and gives CI production write credentials. |
| Pack rows in judge's Postgres, or in a ConfigMap/Secret | Uses the unbacked 10 Gi instance and needs a privileged loader; a ConfigMap is capped at 1 MiB and mixes content into infra. |
| All content private (two images) | Rejected in T0; only the eval pack needs privacy. |
| Broad `spec_hash` plus a pinned public ref and a bump bot | Every typo would desync the pack. `contract_hash` changes only when hidden data must change anyway. |
| Content-addressed artefact table with GC | GC for ≤ 64 KiB payloads breaks HOT updates. Inline bodies plus a nullable `object_key` instead. |
| Pruning outbox/inbox plus JetStream snapshots | Removes the only durable event copy. Keep the outbox; revisit at the resize trigger. |
| AI-predicted expected outputs or keys | Weak tests pass wrong code (EvalPlus). Outputs are computed from the reference; keys are owner-confirmed. |
| CC BY-NC-SA for content | The owner chose MIT for everything. |
| Arena submits not persisted (drafts only) | The owner wants a LeetCode-style arena history with its own done marker. |
