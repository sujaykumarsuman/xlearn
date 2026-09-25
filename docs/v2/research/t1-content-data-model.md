> **T1 research appendix.** Method: four parallel research slices (content model; private content and rights; per-user data, volumes and public-dashboard deltas; authoring pipeline) were synthesized into one draft. The draft then faced two adversarial critiques (leaks and threat model; solo-dev cost, migration and single-node ops). This is the revised final.
>
> **Status:** settled with the owner 2026-09-24 (D5–D10, §14). Two owner decisions change the synthesized text, and **where they conflict, §14 overrides the body**:
> - **D9:** licence is MIT for everything, not CC BY-NC-SA.
> - **D10:** arena Submits are **kept** as history with a separate arena done marker. The body was edited in the places this touches, but the volume model in §6.2 still assumes arena submits are not persisted; re-model in T4.

# T1: content and data model (authored content vs per-user data, course-pluggable)

> Builds on ADR-0026 and the T0 frame, and on ADRs 0005, 0012, 0018, 0024 and 0025. Decided in [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (Accepted 2026-09-24).
>
> **(inferred)** marks reasoning that I did not read in a file or doc. This document replaces the T1 draft. §16 lists every change made in response to the two critiques. The critics' v1 file:line claims were re-checked in this session (see the appendix).

---

## 1. Recommendation in one paragraph

The dividing rule is: **if the problem statement would say it, it is public; if it encodes an answer that is not published anywhere else, it is private.**

**Public content lives in this repo.**
- **What it covers:** manifests, items, answer-free specs, statements, samples, stamped hints and editorials, reference code, stage-gated "solution facts", public asset descriptors, rubric definitions and media.
- **Where:** under `curriculum/courses/<slug>/`, compiled into every image from the release tag.
- **Who reads it:**
  - curriculum seeds items, sections, weeks and concepts for the read API (ADR-0012);
  - curriculum serves manifests, rubrics and asset descriptors straight from the embedded files, not from tables;
  - practice and judge read their own embedded copy.

**The private eval pack lives in a new private repo, `xlearn-evalpack`.**
- **What it covers:**
  - hidden cases, with expected outputs computed by running the reference solution;
  - keys that cannot be derived from public text;
  - SQL hidden instances and their expected result sets;
  - rubric anchors and exemplars.
- **Delivery:**
  1. The pack repo's CI validates it against the public `main`.
  2. CI pushes a private, data-only OCI image.
  3. **Only judge** mounts that image, read-only, as a Kubernetes image volume behind `EVALPACK_DIR`.
- Postgres holds no pack bytes.

**Skew between the two repos is controlled by one narrow `contract_hash`.** It covers only what hidden data depends on: part ids and types, signature, checker, harness, option ids, and asset and rubric `id@v`. If an item's live contract is not in the pack's accepted set, the item falls back to self-grading and shows a badge.

**Per-user data stays in Postgres,** one schema per service, with `path_slug` on every row and event.
- **judge stores:** only **counted** submissions (inline `bytea`, with a nullable `object_key` for T2 blob kinds), one inline draft per key, and evaluations.
- **Runs** live only on the T4 job row. **Arena submits are kept as history** (decision D10) but never count.
- **Outboxes are kept forever** as the durable copy of every event. So there are no pruners and no JetStream snapshots.
- **Growth:** about 86 KB per active learner-day. That keeps the 10 Gi volume under its 60% resize trigger for about 285 learner-years.

**Security preconditions this topic adds:**
- NATS authentication before M3 and before the erase path.
- Learner-facing evaluation responses carry verdicts only.
- `key` parts are never evaluated outside counted contexts.
- In-process profiles and keys that can be derived from public text are labelled **honor**, never "judge-checked".
- An erase path: identity deletes first; each service re-verifies, then deletes and writes a tombstone; an erase ledger is replayed after any restore.

**Authoring:** AI drafts, machines verify, the owner signs off.
- v2.0 scope is all 151 DSA items at the self-path tier, plus full packs for weeks 1–4. That is about 180–285 h, including tooling.
- Statements are original. LeetCode and NeetCode appear only as outbound links.

**Public profile:** it gains per-course aggregates that the learner can hide, each showing where its grades came from. They are served through a `public-read` role that is actually enforced.

**No new database engine.**

---

## 2. Entity map

**Visibility classes:**

| Class | Meaning |
|---|---|
| **C0** | Public, by the learner's choice |
| **C1** | Identifiers and secrets |
| **C2** | Pseudonymous learning facts |
| **C3** | User content |
| **C4** | Sensitive user content |

### Authored content

| Entity | Owner service | Public/private | Storage | Versioning | Notes |
|---|---|---|---|---|---|
| Course catalog row (`paths.json`) | curriculum `path` | public (API needs a session) | relational | release tag | adds `id_prefix` (unique); `problem_total` stays the declared target |
| Course manifest (`course.json`) | embedded in **every** image; curriculum serves it from the embedded files | public (a learner-safe view) | embedded JSON; **no DB copy** | `policy_version` = sha256 of the canonical bytes | practice copies the operative parameters to `attempt.stage_params` plus `policy_version`; assessment copies `rubric_snapshot` onto the session |
| Phase, week, week↔concept | curriculum | public | relational | delete-missing per course | no per-user references |
| Concept reading | curriculum `concept`, key **(path_slug, slug)** | public | TEXT + `templates` JSONB `{lang: code}` | `content_hash` | coach key becomes `<course>:concept:<slug>` |
| Item | curriculum `problem` | public | relational + `spec` JSONB (added when its first reader lands) | `content_hash`, `contract_hash` (both computed at build) | Global id, never re-parented. Status `live \| retired \| withdrawn`; drafts are never seeded. |
| Section (statement, hints, editorial, reference code per language) | curriculum `problem_section` + `language` | public, stage-gated by the gateway | TEXT | rewritten when `content_hash` changes | **hints and editorial are merged only once owner-stamped** (§7.2) |
| **Solution facts** (e.g. `complexity {time[], space[]}`) | item file; judge reads its embedded copy | public, **served only inside the solution stage** | part of `spec` | `content_hash` | the deterministic key for public-derivable probes, which are honor-grade (§7.1) |
| **Item spec:** parts (type, cadence, grading, config), grader steps (kind, `inputs[]`, checker, harness), probes | curriculum `problem.spec`; **embedded** in practice and judge | public; the schema has no field that can hold an answer | JSONB (≤ 64 KB) | `contract_hash` | config covers signature, languages, limits, structured `constraints[]`, samples with expected outputs, option and field ids and labels, asset refs |
| Course asset, public descriptor (SQL schema and small sample generator; canvas palette) | curriculum, **from the embedded files**, by `id@v` | public | embedded file | immutable `<course>/<name>@<v>` + sha256 in `ids.lock.json` | referenced from item specs; no `course_asset` or `problem_asset` tables |
| Media (SVG, PNG, WebP) | curriculum only, via a separate `curriculum/media` embed package | public | embedded file, served by sha256 | sha256 | ≤ 512 KB each, ≤ 32 MB in total; above that, T2 |
| Rubric definition (dimensions, public band descriptors) | curriculum from the embedded files; also embedded in assessment and judge | public | embedded JSON | immutable `id@v` + sha256 | snapshotted onto `mock_session` and onto each evaluation's `versions` |
| Public alias tables (e.g. `dsa/keys/patterns.json`) | embedded in judge and curriculum | public | embedded JSON | `content_hash` | used by `key` grading of public-derivable probes |
| Mock pool | manifest (explicit id lists) | public | manifest JSON | `policy_version` | `mock_session_item` pins `contract_hash` |
| Hidden cases + computed expected outputs | judge (pack) | **private** | pack-image file (JSONL, zstd) | pack semver + digest; per-item `item_hash`; case ids are internal only | expected outputs never enter the sandbox (§10 lists the in-process exceptions) |
| **Private-only** keys (MCQ answers, SD estimates with tolerance, predict-the-output) and per-option rationale | judge (pack) | **private** | pack file | same | rationale reaches the learner only in a counted evaluation result |
| SQL hidden-instance generators + expected result sets | judge (pack) | **private** | pack file | `for_asset` `id@v` | hidden instances are built only for counted or arena-submit jobs; output is verdict-only |
| Rubric anchors, must-cover lists, red flags, exemplars | judge (pack) | **private** | pack file | same | used only in a platform-key scoring call (§10) |
| Pack index | judge, **in memory** | internal | none | pack digest | evaluations record digest and `item_hash`; no `pack_*` tables |
| CI-only artefacts (generators, validators, invalid inputs, brute oracles, wrong solutions, calibration set) | none at runtime | **private** | private git only | commit | never in the built image; the calibration set is **synthetic or owner-authored** |
| Id ledger `curriculum/ids.lock.json` | git | public | JSON | append-only | holds **only** id → course/status, and asset/rubric `id@v` → sha256 |

### Per-user data

| Entity | Owner service | Class | Storage | Pins / versioning | Notes |
|---|---|---|---|---|---|
| Profile visibility | identity `account.profile_visibility` | C0 | relational | — | default `public` (ADR-0024) |
| Enrollment + course visibility | identity `path_enrollment` + `public_visible` | C0/C2 | relational | — | default from the manifest's `public_stats.default_visible` |
| Erase request | identity `erase_request` | C1 | relational | — | no FK, so it outlives the account row |
| Released username | identity `released_username(sha256(lower(username)), until)` | C1 (hashed) | relational | — | 60-day cooldown after an erase; stores no plaintext |
| Item state | practice `user_problem_state` + `path_slug` | C2 | relational | — | `solved` stays terminal for course credit |
| Attempt (`purpose` course \| touch), stages, timers, outcome | practice | C2 | relational + `criteria`, `stage_params` JSONB | `policy_version`, `contract_hash` at start | single writer of learning signals; no `policy_snapshot` table |
| Submission (**counted contexts** course, touch and mock, **plus arena submits**) | judge | C2 metadata | relational, `uuidv7` PK | `contract_hash` | Runs are not persisted. Arena submits are kept as history (D10) and never count; `counted` is derived from `context_kind`. |
| Submission part (code, text, choice JSON, canvas scene) | judge `submission_part` | C3 | inline `body bytea` (lz4), **or** `object_key` for blob kinds; `sha256` not unique | — | T4's `payload_ref` = (submission_id, part_id); no content-addressed GC |
| Draft (one per account, context, part and language) | judge | C3 | one inline row per key, HOT upsert, `fillfactor=70`, key columns `NOT NULL DEFAULT ''` | — | deleted 90 days after `updated_at` (one DELETE in the T4 sweeper) |
| Run job payload and output | judge `job` (T4) | C3 | on the job row only | — | deleted 24 h after completion |
| **Arena done marker** (D10) | judge (derived from a passing arena submit), shown on the Problems arena list | C2 | relational | — | **separate from course done**; never a learning signal; learner-private in v2.0 (not in public stats or the heatmap) |
| Evaluation (checks, internal per-test verdicts, versions, `trust`) | judge | C2 | relational + JSONB | pack digest, `item_hash`, grader, harness, runner, rubric and model versions | the learner-facing shape comes from an allowlisted DTO (§10) |
| Evaluation feedback / analyzer prose | judge `evaluation_feedback` | C3 | TEXT ≤ 8 KiB | — | never in events |
| Revision item; touch result | review + `path_slug`; `touch_result` gains `graded_by`, `trust`, `evaluation_id`, `attempt_id`, `criteria` | C2 | relational | — | the ladder stays universal |
| Mistake entry | review + `path_slug` | C3 prose | relational | — | 2 KiB per field, checked in the handler |
| Weak-area snapshot | review + `path_slug` | C2 | relational + JSONB | — | kept (tiny) |
| Mock session, `mock_session_item`, rubric score | assessment + `path_slug`, `rubric_snapshot`, `total`/`max_total`, `scored_by` | C2 (+ notes C3) | relational + JSONB | `policy_version`, rubric `id@v`, item `contract_hash` | — |
| Projections | assessment `proj_*` + `path_slug` | C2 | relational | rebuilt by replay | §9 |
| Coach thread and message | coach, key `<course>:<ctx>` | C3 | TEXT | — | kept; revisit at the resize trigger |
| Transcript, audio | T6 owner (**not** assessment) | C4 | object storage | — | T6 decides |
| outbox (+ `account_id`), inbox, `erased_account` | every producer/consumer | C2 | relational | — | **outbox and inbox are kept forever** (the durable event copy); tombstones are kept forever |
| Events | NATS | C2 only | stream | envelope v2 | never prose or code; publishers authenticated (§10) |

---

## 3. Where content lives and how it ships

### 3.1 Visibility tiers

| Tier | Contents | Enforcement |
|---|---|---|
| Public, ungated | catalog, manifest view, item metadata, `spec` (parts, samples, options, signature), attempt-stage sections | the schema has no answer fields; strict decoding |
| Public, withheld while an item is live | `pattern`/topic, hint and solution sections, **solution facts**. "Live" means there is an open counted attempt or a due or live touch (T0 §7). | **one gateway `withhold()` function applied on every surface** (table in §10), not just `aggregateProblem` |
| Private, judge-only | the eval pack | a separate repo, a private image, mounted only in judge |

- **Withholding is a nudge, not secrecy.** The repo is public. Any probe whose key can be derived from public content is therefore **honor-grade** (§7.1, §9).
- **There is no fourth "private but shown to the learner" channel.** Rationale for a private key reaches the learner only through a counted evaluation result.

### 3.2 Public content: layout and delivery

```
curriculum/                      package curriculum:  //go:embed paths.json ids.lock.json all:courses
  paths.json  ids.lock.json  _schema/*.schema.json
  courses/<slug>/
    course.json  phases.json  weeks.json  concepts.json  concepts/<slug>.md
    rubrics/<id>@<v>.json   keys/*.json (public alias tables)
    assets/<name>@<v>/{descriptor.json, schema.sql, sample.sql}   # generators, not dumps
    items/<id>/                                                   # dir = stable id
      item.json                                                   # strict; no field can hold a private answer
      sections/<stage>/<NN>-<kind>.md                             # hints/editorial only once stamped
      _code/solution.go                                           # solution-stage reference (Go only until PRD Q6)
      _starter/…                                                  # optional; else generated from signature
curriculum/media/                package media: //go:embed all:courses   (imported by curriculum ONLY)
```

**Why this layout** (checked by experiment in the scratchpad; the critic confirmed it):

| Finding | Consequence |
|---|---|
| A single `all:courses` embed makes a new course data only | no code edit per course |
| A plain `.cpp` file in a Go package directory breaks `go vet ./...` | reference code goes in `_`-prefixed directories |
| Plain `//go:embed courses` silently drops `_` directories | the directive must be `all:courses` |
| `all:` also embeds dotfiles | the loader ignores dotfiles, and CI diffs the **embedded-file allowlist** |
| A nested `go.mod` is silently dropped by both embed forms | ship `go.mod.tmpl` instead |
| `.dockerignore`'s `*.md` covers only root-level Markdown, and a nested private clone lands in the build context | `.gitignore` and `.dockerignore` list both `evalpack/` and `xlearn-evalpack/`; the allowlist check catches anything that slips through |

This refines ADR-0026's `curriculum/<slug>/` to `curriculum/courses/<slug>/`. The substance of ADR-0026 is unchanged.

**Delivery path:**
1. A tag triggers `deploy.yml`, which rebuilds every image from the same commit.
2. **curriculum** seeds in one transaction. The id guard, slug guard and retire rules apply, and an item's children are rewritten only when its `content_hash` changes. It serves:
   - items, sections, weeks and concepts from the DB;
   - `GET /paths/{slug}/manifest`, `GET /rubrics/{id}@{v}`, `GET /course-assets/{id}@{v}` and `GET /media/{sha256}` from the embedded files.
3. **practice** and **judge** parse their embedded copy at start: an in-memory index of spec, role, status and `contract_hash`.
4. The **gateway** proxies curriculum under `/api/…` (never `/assets/`, which is the SPA's static prefix) and applies `withhold()`.

**Sizes:** about 5–8 MB of text at 345 items (inferred). Media stays out of the practice and judge binaries. SQL datasets ship as seeded generators plus a small sample, ≤ 256 KB.

### 3.3 Private eval pack

**Source repo:** `xlearn-evalpack`.
- It is **created fresh**, never as a fork or template of xlearn, and is **never made public**.
- It is checked out as a sibling of the public repo, never nested inside it.

```
pack.json                                   # {format_major, version}
courses/<slug>/items/<id>/
  pack.json        # accepts_contract_hashes[≤2], private keys, wrong-solution expectations, review stamps
  tests/edge.jsonl # hand-picked inputs; generated-case hashes pinned in tests.lock
  keys/ anchors/ exemplars/                 # only where the answer is not public-derivable
  gen/ validate/ invalid/ submissions/{brute.*,wrong/*}   # CI-only
courses/<slug>/assets/<name>@<v>/{gen-hidden.sql, instances.json}
tests.lock                                  # case id → sha256 of materialized cases
```

**Built image:** `ghcr.io/sujaykumarsuman/xlearn-evalpack:<semver>`.
- `FROM scratch`, with **one layer per course**.
- `/manifest.json` = `{format_major, version, validated_against: <public commit>, items{id: {item_hash, accepts_contract_hashes, grader_kinds, files{path: sha256}}}}`.
- Each item directory holds `cases.jsonl.zst` and its keys, anchors and exemplars.
- **No generators, oracles, wrong solutions or calibration data.**

**Flow:**

1. **Private CI**
   - Validates against the public repo at `main` (or a ref named in the PR) and records the commit as `validated_against`.
   - There is no pinned `PUBLIC_REF` file and no bump bot.
   - Runs the gates in §7.2, then pushes to private GHCR.
   - After every push, **an anonymous manifest GET must return 401 or 403** (making a package public is irreversible).
   - Pulls use a **machine user** (required), with a classic `read:packages` PAT scoped to that account.
2. **Flux**
   - ImageRepository with `secretRef`, and ImagePolicy `>=1.0.0 <2.0.0` with the digest pinned.
   - The judge HelmRelease declares `extraVolumes: [{name: evalpack, image: {reference: …, pullPolicy: IfNotPresent}}]`, a read-only mount at `/evalpack`, `EVALPACK_DIR=/evalpack`, and `imagePullSecrets`.
   - The chart already passes these through (`infra/charts/project/values.yaml:27,107-108`; `templates/deployment.yaml:24-27,88,95-106`), so **no chart change is needed**.
   - The pull secret goes in `xlearn` and `flux-system`, under the existing SOPS rule `apps/secrets/*.enc.yaml`.
3. **judge at start**
   - Verifies every file hash against `manifest.json` and builds the in-memory index of keys and anchors. It reads cases lazily, per job.
   - Each item gets a status: `ok | spec_mismatch | unsupported | invalid`.
   - If the format major is unsupported or the manifest is invalid, judge stays **Ready with zero evaluable items**; it never crash-loops, which would wedge `apps` with `wait: true`.
   - If the image cannot be pulled, the pod cannot start, and the rolling update keeps the old pod serving (§15).
4. **Serving**
   - `GET /internal/evaluable?path=` returns item id → `run_available`, `submit_available` and grader kinds, from memory.
   - **No handler ever serves `EVALPACK_DIR`**; a test asserts this.
   - The runner receives only the current job's inputs; comparison happens in judge (§10 has the exceptions).

**Platform check:**
- Image volumes are stable and on by default since Kubernetes 1.36. k3s `v1.36.4+k3s1` bundles containerd v2.3.4.
- **Spike before M3:**
  - (i) mount works;
  - (ii) a pod in another namespace **without** the pull secret cannot mount the cached image (KEP-2535 behaviour; confirm it covers image volumes).
- **Fallback:** an initContainer that copies the pack into an emptyDir (needs an `initContainers` chart knob, off by default). `EVALPACK_DIR` hides the difference from judge's code.
- *Result (spk-02 re-run, 2026-09-25):* **image-volume GO**, with the kubelet defaults (`NeverVerifyPreloadedImages`; `KubeletEnsureSecretPulledImages` beta, on), on env A's k3s v1.36.4 against a password-protected private registry. (i) The read-only image-volume mount works under PSA baseline. (ii) A pod without the pull secret is refused the cached image: `ErrImagePull … no basic auth credentials` with `IfNotPresent`, `ErrImageNeverPull` with `Never`. It stays refused after a k3s restart. (iii) The initContainer fallback (variant A) also works. Caveat for mi-09: the pack credential must exist only as the imagePullSecret, because node-level credentials make the cached image readable by every pod. See [t3 §16.3](t3-sandbox.md#163-image-volume-spk-02).

### 3.4 Versioning and pinning

All hashes are sha256 over **canonical bytes from re-marshalling the typed Go struct** (fixed field order, sorted map keys). One public Go package does this, shared by the curriculum seed, practice, judge and `packlint`. There is no JCS dependency.

| Hash | Scope | Used for |
|---|---|---|
| `policy_version` | the course manifest | copied onto attempts and mock sessions |
| `content_hash` | the whole resolved public item, sidecars inlined | seed skips unchanged children; cache busting |
| **`contract_hash`** | **only what hidden data depends on:** part ids and types; signature (mode, name, param and return types, class ops); `harness@v`; checker name and contract parameters (unordered, eps); choice and blank option/field **ids**; probe ids and types; asset and rubric `id@v`. **Excludes** prompts, labels, samples, limits, languages, sections and metadata. | the grading contract the pack was written against |
| `item_hash` / pack digest | the item's private files / the whole image | provenance on every evaluation (recorded when a job is **claimed**) |
| asset / rubric `id@v` | immutable bytes (CI fails if an `id@v` changes) | part of `contract_hash` |

There is no `spec_rev`. Humans see a `contract_hash` prefix.

**Mismatch rules:**
1. **At judge start:** an item is `ok` only if its embedded `contract_hash` is in the pack's `accepts_contract_hashes`. Otherwise:
   - it grades as self (or blocks counted attempts if the course is `evaluator_only`);
   - the SPA shows a **"grading pending: self-report" badge**;
   - the ops check counts mismatches (§11).
2. **At submit:** if the attempt's pinned `contract_hash` is not the live one, the result is `inconclusive(contract_changed)`. It never counts as a failed submit; the SPA reloads and keeps the draft.
3. **Edits to prompts, samples, limits or languages** change only `content_hash`, so grading is unaffected. `packlint` re-checks them against the pack as an advisory on the next pack PR, e.g. that a time-limit change keeps the TLE expectations valid.
4. **Label edits on `key`-graded parts** are flagged by public CI. The owner either mints new option ids (a contract change) or confirms "typo" in the PR.
5. **Contract changes need new hidden data anyway.** Author them with the pack change, and list both hashes. Either repo can then ship first; the worst case is a short self-graded window for that item.

**Pack semver:**
- **MAJOR** is a format break: ship judge first, then widen the range.
- **MINOR** means items were added or changed; **PATCH** means test fixes.
- Concluded attempts are never re-graded.

### 3.5 Rollback
- **Everything rolls forward.** Revert the commit and tag a new patch; Flux re-selects the highest tag (T0 §9).
- **Emergency options:**
  - pin a digest or narrow the ImagePolicy in an infra PR;
  - flip the T0 env kill switch (self-report only), which Flux applies in about a minute.
- A pack change restarts judge; with one replica, the old pod serves until the new one is Ready (inferred).

### 3.6 Backup hooks for T2

| What | Backup |
|---|---|
| Public content | public git |
| Pack source | private GitHub repo plus a local clone. **No bundle mirror** in an in-cluster bucket. GHCR keeps the last 10 versions. |
| Postgres | **CNPG PITR is the one critical backup.** It now holds the learning record (judge submissions and evaluations) **and the outboxes, which are the durable event log.** A lost NATS volume is rebuilt with `UPDATE outbox SET sent_at = NULL` per producer; consumers dedupe through the inbox, and projections are commutative (ADR-0018). |
| JetStream | **No snapshots needed.** |
| Erase ledger | an append-only object `erase-ledger/<account_id>` `{account_id, erased_at}` **outside** the CNPG backup, written by identity with a credential scoped to that prefix. The restore runbook must replay it (§6.6). |
| Gate | **CNPG backups must exist before M3 grading opens to anyone other than the owner.** M3's code does not depend on T2. |

### 3.7 Local dev and public CI
- Compose mounts `${EVALPACK_DIR:-./internal/judge/testdata/pack}:/evalpack:ro`, a synthetic fixture. With no pack, every item is self-graded.
- Public CI and e2e need no secrets.
- Compose parity: move to `postgres:18` and `nats:2.14` (today `docker-compose.yml:28,45` pins 16 and 2.10).
- `AGENT.md` gains a rule: **never copy content from `../xlearn-evalpack` into this repo.** AI agents work next to the sibling checkout.

---

## 4. Schema deltas vs v1 (conceptual)

**The rule: a column lands with its first reader.** Spec and contract columns wait for M2a and M3, which keeps M1's "no behaviour change" promise provable.

### curriculum

**`path`**
- Add `id_prefix` (unique). There is no manifest column (it is served from the embedded files).
- **Course-slug guard** at seed time and in a test: `^[a-z0-9]+(-[a-z0-9]+)*$`, and not in `{u, auth, settings, api, assets, healthz, readyz}`. The regex already excludes `.well-known`.

**`concept`**
- `UNIQUE(path_slug, slug)` replaces `UNIQUE(slug)`. The upsert is keyed on both, which ends re-parenting.
- `code_template` becomes `templates jsonb`.
- The API becomes `GET /paths/{slug}/concepts/{c}`, with a DSA alias for one release.

**`problem`**

| Column(s) | Change | Lands |
|---|---|---|
| `role` | CHECK `(core, reinforcement, drill)`, backfilled from `is_reinforcement` | M1a |
| `status` + `retired_at` | CHECK `(live, retired, withdrawn)` | M1a |
| `links jsonb` | backfilled from `leetcode_url`/`neetcode_url` | M1a |
| `content_hash` | new | M1a |
| `spec jsonb` (probes + solution facts) | new | **M2a** |
| `contract_hash`, derived `grading_summary` | new | **M3** |

**Id guard**
- DSA ids match `^[1-9][0-9]{0,2}$`; other courses match `^<id_prefix>-[0-9]{3}$`.
- `UpsertProblem … WHERE problem.path_slug = EXCLUDED.path_slug` becomes `:execrows`. The seed aborts if the row count isn't 1 (fixes `problem.sql:40-49`).

**Retire semantics**
- **Retired:** hidden from the index, week counts, new counted attempts and mock pools; still resolvable by id; existing ladders continue.
- **Withdrawn:** a takedown. Prose is blanked and due touches are hidden.
- An item in the DB but missing from the seed is retired defensively; `ids.lock.json` should prevent that case anyway.

**`problem_section`**
- Add `language NOT NULL DEFAULT ''`, backfilled to `'go'` for `kind=code`.
- The unique key becomes `(problem_id, stage, "order", language)`.
- Delete and re-insert when `content_hash` changes. This fixes v1's lingering rows (there is no DELETE in `queries/*.sql`).

**Other changes**
- **No new tables.** Assets, rubrics and manifests come from the embedded files.
- **Delete-missing** per course for phases, weeks, `week_concept` and concepts.
- **CHECKs:** keep `difficulty`, `stage` and `path.status`; add `role` and `problem.status`. Sets backed by a code registry (part type, grader kind, section kind, asset kind, harness, checker) are validated by the loader only.
- **API:**
  - `GET /problems/{id}` adds `spec` (answer-free), `role`, `status`, `links` and asset refs;
  - `solution_facts` is returned only with the solution stage;
  - `maxBulkProblemIDs` goes from 256 to 1024 (`handlers.go:238`).

### practice

| Change | Lands |
|---|---|
| `user_problem_state.path_slug` | M1a |
| `attempt`: `account_id`, `path_slug`, `problem_id` (denormalized) | M1a |
| `attempt.purpose` CHECK `(course, touch)`; `revision_item_id`, `touch_level`, `band` | M2a |
| `attempt.state` CHECK `(attempting, awaiting_evaluation, provisional, concluded)`; `concluded_at` set once (`WHERE concluded_at IS NULL`) | M2a |
| `grade`, `passed`, `score_value`/`score_max`, `graded_by` CHECK `(self, auto, ai, override)`, `trust` CHECK `(checked, honor)`, `criteria jsonb` | M2a |
| `policy_version`, `stage_params jsonb` | M2a |
| `within_timer`, `failed_submits`, `counted_submits`, `first_submit_passed`, `language`, `contract_hash` | M3 |

- `timer.kind` adds `touch`. The stage CHECK and the 4-grade CHECK stay.
- New tables: `inbox` (M3, practice's first consumer) and `erased_account`. `outbox.account_id` lands in M1a.

### judge (new schema, owner role `xlearn_judge`; queue and state machine belong to T4)

| Table | Shape |
|---|---|
| `submission` | `(id uuidv7, account_id, path_slug, item_id, context_kind, context_id, action, contract_hash, submitted_at)`. `context_kind` CHECK `(course, touch, mock, arena)`; `action` CHECK `(submit, final, give_up)`. Arena rows never count (D10), so there is no separate `counted` column. |
| `submission_part` | `(submission_id, part_id, language NOT NULL DEFAULT '', sha256, bytes, body bytea COMPRESSION lz4 NULL, object_key NULL)` |
| `draft` | `(account_id, context_kind, context_id, part_id, language, body, object_key, updated_at)`; all key columns `NOT NULL DEFAULT ''`; `fillfactor=70`; no index on `updated_at`. Arena drafts are keyed by item id. |
| `evaluation` | `(…, status CHECK (passed, failed, inconclusive, error), reason enum, score, max, trust, checks jsonb, per_test jsonb (internal), versions jsonb)` |
| other | `evaluation_feedback`, `job` (T4; holds the Run payload and output), `usage_ledger` (T5), `outbox` (+`account_id`), `inbox`, `erased_account` |

- **Tables holding bodies (`submission_part`, `draft`, `job`, `evaluation_feedback`) carry only PK and FK constraints.** Validation happens in Go before insert, so a failed CHECK never writes `DETAIL: Failing row contains (…)` with learner code into the Postgres logs.
- Every per-user table gets an `(account_id, created_at)` index.

### review
- `path_slug` on `revision_item`, `mistake_entry` and `weak_area_snapshot` (unique `(account, path, week_of)`); `reminder.path_slug` is nullable.
- **`touch_result`** (M2a):
  - adds `graded_by`, `trust`, `evaluation_id`, `attempt_id` and `criteria jsonb`;
  - the v1 DSA columns (`named_pattern_secs`, `solved_in_timer`, `stated_complexity`) become **nullable**;
  - the v1 self endpoint keeps filling them.
- The `mistake_entry.category` CHECK is dropped in M1c (`00002:28-30`); the manifest validates instead.
- 2 KiB per prose field, enforced in the handler.
- `outbox.account_id` and `erased_account` are added.

### assessment
- **`mock_session`:**
  - adds `path_slug`, `rubric_id`, `rubric_snapshot jsonb`, `total`/`max_total` and `scored_by`;
  - `total_35` is backfilled into `total` with `max_total=35`, then dropped in M1c;
  - `difficulty` becomes nullable.
- **`mock_session_item(session_id, ordinal, item_id, path_slug, contract_hash)`** supersedes `problem_id`/`set_id`, with one backfilled row per existing session.
- The `rubric_score.dimension` CHECK is dropped; scores are validated against the snapshot.
- Projections are redefined (§9), then dropped and replayed at M2b.
- Remove `problemTotal=151` (`progress.go:18`).
- Add `GET /public/stats`, which accepts **only** `public-read`.
- `outbox.account_id` and `erased_account` are added.

### identity
- `account.profile_visibility` CHECK `(public, private)`, default `public`.
- `path_enrollment.public_visible`, defaulted from the manifest.
- Enrollment validates the slug against active catalog paths.
- New: `erase_request`, `released_username`, and an erase-ledger writer (lands with T2).
- Provision `XLEARN_IDENTITY` (today identity uses a `LogPublisher`).
- The non-PII resolver adds `visible_courses[]`.
- **No slug reservations** (ADR-0025).

### coach
- `page_context`: `problem:<id>` stays; `concept:<slug>` becomes `<course>:concept:<slug>` (`Coach.tsx:287`). Add a nullable `path_slug`. Prod has 0 coach rows.
- **Mode is `review` only when the item has no live attempt or touch.** Today it is keyed on `solved` (`coach.go:294-302`). `pattern` is left out of the prompt during a live attempt or touch.
- Add an erase consumer (coach's first NATS use).

### Expand → backfill → contract

| Step | What lands |
|---|---|
| **M1a** (expand) | `path_slug DEFAULT 'dsa'` on per-user rows. `outbox.account_id`, backfilled from `payload_json`. `role`, `status`, `links`, `language`, `templates`, `id_prefix`, the concept composite key, `content_hash`. `total`/`max_total` + `mock_session_item`. identity visibility columns. The **new layout via a one-shot converter**, plus a snapshot of v1 seeded rows asserted by the new loader's test; the old loader is deleted in the same PR. Id and slug guards with `:execrows`; section rewrite. **Envelope v2 with `path_slug` on unchanged subjects**, consumers accepting v2 before producers emit it. Stream `MaxBytes` budget and options knob (§6.5). The two copied LeetCode examples are replaced (§8). |
| **M1b** | Readers move to the new columns. Course-scoped APIs with DSA aliases. Coach prefixes. **The gateway `withhold()` function on every surface** (§10). |
| **M1c** (contract) | Drop `is_reinforcement`, `leetcode_url`/`neetcode_url`, `code_template`, `total_35`, the category and dimension CHECKs, `UNIQUE(concept.slug)`, and the `'dsa'` defaults. |
| **M2a** | Touch attempts, `problem.spec` (probes and solution facts), the `touch_result` columns. The `touch_concluded` consumer ships before its producer. |
| **M2b** | The `touch_scored` backfill, **then** projections v2 plus replay. The `public-read` role, `/public/stats`, visibility toggles. |
| **E** (erase) | **After NATS authentication.** `XLEARN_IDENTITY`, erase tables, consumers before producer, `DELETE /api/me`. Required before behavioral and before signup opens broadly. |
| **M3** | The judge schema, the `contract_hash` columns, attempt evaluation columns, the practice inbox, pack delivery. **Preconditions:** NATS authentication, sandbox default-deny, admission limits; CNPG backups before grading for non-owner learners. |

Production has 1 account and 19 events, so every step is cheap.

---

## 5. Blob list for T2

**Not blobs:**
- code, text and choice answers (inline `bytea`);
- hidden tests (the pack image);
- public SQL data (generator scripts);
- media (embedded, up to 32 MB).

The object-key layout is `u/<account_id>/<kind>/<id>`, so an erase is a prefix delete.

| Artefact | Owner | Typical / max | Access pattern | Retention | Backup? |
|---|---|---|---|---|---|
| Canvas scene (final, counted) | judge | 30–80 KB / **512 KiB** minified, no embedded images | Written once at submit, **proxied** through gateway → judge (fits the 1 MiB body cap, `bff.go:664`). Read back through judge under the learner's JWT. The grader reads only the semantic export (JSONB ≤ 32 KiB). **Stored inline until T2 exists.** | forever (counted) | yes |
| Canvas draft | judge | same | Autosave on change with a ≥ 60 s debounce, proxied; read on resume | 90 d idle | no |
| Mock transcript | T6 owner | about 40 KB per 45 min / 256 KiB | Written server-side by T6; read by the learner (proxied) and by the scoring call | 12 months (T6 to confirm) | yes |
| Audio recording | T6 owner | about 8 MB per 45 min at 24 kbps / about 11 MB | **Presigned PUT** from the browser (bypasses the 1 MiB cap and Traefik timeouts); short-TTL presigned GET for playback | opt-in, 30 d | no |
| Video | — | about 400 MB per 45 min | **not stored** | — | — |
| CNPG base backup + WAL archive | platform | about 0.9 GB live today; draft churn adds WAL | Barman writes; read only on restore | PITR 14–30 d; **this window is the backup erase SLA** | this is the backup |
| Erase ledger | identity | < 1 KB per erase | Append-only; read only by the restore runbook | forever | its own prefix; not inside the CNPG backup |
| Public media overflow (only above 32 MB) | curriculum | ≤ 512 KB per image | public, cacheable by sha256 | life of the content | git |

**Removed from the draft:**
- JetStream snapshots (the outbox is the copy);
- the eval-pack git bundle (it would add exposure; git and GHCR suffice).

---

## 6. Volumes and retention

### 6.1 Content volumes (inferred)

| Store | Size |
|---|---|
| `curriculum` schema at 345 items | < 10 MB with indexes |
| Embedded text | ~5–8 MB per binary |
| Media | ≤ 32 MB, curriculum only |
| Pack image | v2.0 (~35 DSA packs): a few MB. Full programme: about 30–40 MB zstd, capped at ≤ 150 MB. Hard caps: ≤ 4 MiB compressed per item, ≤ 256 KiB per case input. It sits on node disk in the containerd store; zero Postgres. |
| GHCR, last 10 versions with per-course layers | ≲ 400 MB stored; ≲ 200 MB/month transfer (only changed layers are pulled). That is inside the 500 MB storage and 1 GB/month transfer fallback if private-package billing applies. |

### 6.2 Per-user growth (model: scratchpad `t1/vol_final.py`)

**Assumptions** (unchanged from the draft):
- 250 active days a year;
- each day: 2 concluded attempts with 2 counted submits each, 4 touches, 3 arena Runs, 0.7 mistakes, 0.15 mocks;
- about 38 events a day.

**Final design choices modelled:**
- Runs and arena submits are not persisted. _(Superseded by D10: arena submits are kept. The growth figures below are therefore slightly low; re-model in T4.)_
- Counted parts are stored inline without dedupe.
- outbox and inbox are kept.
- Drafts expire after 90 days idle.

**Growth is about 86 KB per active learner-day, including indexes.** Where it goes:

| Share | Source |
|---|---|
| 45% | outbox + inbox (the durable event log) |
| 31% | counted submissions |
| 8% | AI feedback |
| small | everything else |

| 1-year projection | Postgres | NATS (all streams) |
|---|---|---|
| Owner alone | 0.03 GiB | 0.01 GiB |
| 100 learners | 2.1 GiB (21% of 10 Gi) | 0.49 GiB |
| 300 learners | 6.3 GiB (**past the 60% trigger**) | 1.46 GiB |
| 1,000 learners | 21 GiB (needs a resize to ~40 Gi) | 4.9 GiB (needs a PVC resize) |

- **Resize trigger:** 60% of either volume, or about 285 learner-years. That is the point to decide retention (pruners, outbox archiving).
- CNPG, Longhorn and the NATS PVC all expand online, and the node has 176 GB free.
- Revisit the 1 Gi CNPG memory limit at the same trigger (inferred).

### 6.3 Caps and quotas (enforced by judge; T3/T4 own rate limiting)

| Payload | Cap |
|---|---|
| Source part | 64 KiB |
| Multi-file part | 128 KiB, ≤ 16 files |
| Text part | 16 KiB (STAR fields 4 KiB each) |
| Choice/blank part | 2 KiB |
| Canvas scene | 512 KiB |
| Semantic export | 32 KiB |
| Custom Run input | 64 KiB |
| Captured output | 8 KiB per case, 64 KiB per Run (**samples only**) |
| AI prose | 8 KiB |
| Mistake fields | 2 KiB each |

- **Per account per day:** at most 200 Runs, 100 counted submits and 100 arena submits.
- Everything fits the 1 MiB gateway body cap and NATS's 1 MiB `max_payload`. Events carry refs only.

### 6.4 Retention (v2.0: caps and quotas only; no pruners)

| Data | Rule | Mechanism |
|---|---|---|
| Counted submissions, evaluations, AI prose | keep (the learning record) | erase only |
| Run payloads and outputs | never persisted beyond the job row; deleted 24 h after completion | T4 job sweeper |
| Arena submits (D10) | kept (history); erase only | — |
| Drafts | 90 d after `updated_at` | one DELETE in the same sweeper (a seq scan is fine at this scale) |
| outbox, inbox | **kept** | revisit at the resize trigger |
| Coach, reminders, weak-area snapshots | kept | revisit at the resize trigger |
| `erased_account` tombstones | **forever**; the rebuild runbook must not truncate them | — |
| NATS per-message TTL | **not adopted on 2.14.6** (nats-server#8594) | stream limits instead |

### 6.5 NATS stream budget (fixes the uncreatable 4 GiB limits)

JetStream reserves each stream's `MaxBytes` against the server's `max_file_store` when the stream is created. Live, `max_file_store` is **5 Gi**. So the **sum of all `MaxBytes` must stay ≤ 75% of it (3.75 GiB).**

| Stream | MaxBytes | MaxAge | Discard | Growth (per learner-year) |
|---|---|---|---|---|
| `XLEARN_PRACTICE` | 1 GiB | 0 | New | ~1.1 MB |
| `XLEARN_REVIEW` | 1.5 GiB | 0 | New | ~3.0 MB |
| `XLEARN_JUDGE` | 512 MiB | 14 d | Old | ~0.9 MB |
| `XLEARN_ASSESSMENT` | 128 MiB | 0 | New | tiny |
| `XLEARN_IDENTITY` | 128 MiB | 0 | New | tiny |
| **Sum** | **3.25 GiB** | | | |

- `Discard=New` on replay streams makes the relay stall **loudly** rather than drop events. Unsent rows stay in the outbox, so nothing is lost.
- The ops check alerts at 70%.
- A unit test sums every stream's `MaxBytes` against a declared store budget.
- The stream options move out of the hard-coded `nats.go:78-83`.
- Grow the NATS PVC on the same resize trigger.

### 6.6 Erase path (none exists today; no `DELETE /api/me` in `bff.go`)

**Precondition:** NATS authentication, with per-service publish ACLs, is live (§10).

1. **Request.** `DELETE /api/me` with a typed confirmation and a re-auth within the last 5 minutes.
2. **identity**, in one transaction:
   - deletes the `account` row, cascading to OAuth links, sessions, onboarding and enrollments;
   - inserts `erase_request` and `released_username` (hashed, 60-day cooldown);
   - writes outbox `xlearn.identity.account_erasure_requested{account_id}`;
   - bumps the gateway cache epoch.

   The profile returns the uniform 404 at once.
3. **Each owner (practice, judge, review, assessment, coach) erases through a durable consumer:**
   - it **re-verifies** with identity (`GET /internal/erasures/{id}`: the request exists and the account row is gone) before acting;
   - then, in one transaction: insert the tombstone, and `DELETE … WHERE account_id = $1` **including its outbox rows**;
   - judge also deletes `u/<account_id>/` objects (via T2);
   - it acks with `xlearn.<svc>.account_erased`.
4. **Tombstones** are checked by async consumers, judge's worker before it writes a result, and projection replay. They are not needed in user write handlers, because deleted sessions stop those writes.
5. **NATS:** the default (owner Q4) leaves the pseudonymous C2 events in place; they are unlinkable once identity and the backups are gone. The optional **re-seal runbook** removes them with no subject change:
   - purge the stream (using the ops identity);
   - `UPDATE outbox SET sent_at = NULL`;
   - the relays republish without the erased rows, and inboxes dedupe.
6. **Close** the request when all 5 acks arrive. The ops check flags anything open for more than 24 h.
7. **Backups:** erased rows age out within the PITR window, which is the published SLA. **After any restore**, the runbook replays the erase ledger before traffic resumes: tombstones are rewritten and the deletes re-run.
8. **Out of reach, and disclosed:** platform-LLM providers. Calls use `store:false` or zero-retention where offered. Calibration and exemplar sets never contain learner answers.

---

## 7. Authoring pipeline

### 7.1 Package format

**Public `item.json`** (abridged; a fresh example, since v1's 3Sum example is replaced in §8):

```jsonc
{ "id": "16", "course": "dsa", "week_n": 2, "sort_order": 2, "title": "3Sum", "difficulty": "med",
  "pattern": "two-pointers", "role": "core", "status": "live",
  "provenance": {"origin": "original", "inspired_by": ["leetcode:15"], "authored_by": "ai-assisted"},
  "links": [{"kind": "leetcode", "url": "https://leetcode.com/problems/3sum/"}],
  "review": {"statement": "2026-10-02", "hints": "2026-10-09", "editorial": "2026-10-09"},
  "parts": [{"id": "solution", "type": "code", "cadence": "iterate", "grading": "auto", "required": true,
    "config": {"signature": {"mode": "function", "name": "threeSum",
                             "params": [{"name": "nums", "type": "int[]"}], "returns": "int[][]"},
               "languages": ["go"], "harness": "func-json@1", "checker": "unordered_deep",
               "constraints": [{"arg": "nums", "len": [3, 3000]}, {"arg": "nums[*]", "range": [-100000, 100000]}],
               "samples": [{"id": "s1", "args": [[4, -2, -2, 0, 1]], "expected": [[-2, -2, 4]]}],
               "limits": {"time_ms": 2000, "memory_mb": 256}}}],
  "grader": [{"step": "tests", "kind": "code", "inputs": ["solution"], "required": true}],
  "solution_facts": {"complexity": {"time": ["O(n^2)"], "space": ["O(1)", "O(log n)", "O(n)"]}},  // solution-stage only
  "revision": {"probes": [
    {"id": "p-pattern", "type": "text", "band": "recall", "grading": "key", "key_source": "public:pattern",
     "criterion": "pattern_named_fast", "timer_s": 120, "prompt_md": "Name the pattern."},
    {"id": "p-complexity", "type": "blank", "band": "recall", "grading": "key", "key_source": "public:solution_facts.complexity",
     "criterion": "complexity_stated",
     "config": {"fields": [{"id": "time", "label": "Time"}, {"id": "space", "label": "Space"}]}}]} }
```

**Private `pack.json`.** For DSA, there are no keys, because both probe answers are public:

```jsonc
{ "item": "16", "accepts_contract_hashes": ["sha256:…"],
  "wrong": [{"file": "submissions/wrong/no-dedupe.go", "expect": "WA", "category": "right_pattern_wrong_state"},
            {"file": "submissions/wrong/cubic.go", "expect": "TLE", "category": "complexity_misjudged"}],
  "review": {"tests": "2026-10-09"} }
```

**Format rules:**
- **Cases:** canonical JSONL `{args | ctor+ops+args, expected, tags}`. The case id is a content hash, **internal to judge only**, so a fix creates a new case. Expected outputs are filled in by CI; only samples carry a public `expected`.
- **Closed type registry:** `int`, `int64`, `float64`, `bool`, `string`, `T[]`, `T[][]`, `ListNode`, `TreeNode` (level order), `GraphNode`, and `class` mode with op sequences. A new type is code plus a release.
- **Closed checker registry:** `exact`, `unordered`, `unordered_deep`, `float_abs|rel(eps)`, `set_equal`, `any_of`, plus a few semantic ones. Checkers return **typed verdicts, never formatted values**. No per-item checker code runs at runtime.
- **Key sources:**
  - `key_source: public:<field>` reads the public spec or a public alias table; the probe is **honor-grade** (`trust=honor`).
  - `key_source: pack` reads a private key (MCQ answers, SD estimates, predict-the-output); the probe is `trust=checked`.
- **Key parts:** a `key`-graded part must be `cadence: final`, and judge evaluates it **only in counted contexts** (course, touch, mock), once per context (§10).
- **Probes:** target 2–3 per item. DSA pattern and complexity probes are **templated from public fields**, so they cost no private key files.

**Per-course variations:**

| Course | Public half | Private half | Trust |
|---|---|---|---|
| go-concurrency | `_starter/module/` with `go.mod.tmpl`, editable file list, `visible_test.go` | hidden `*_test.go` (stress, goleak, deadlines), `go-race` params, mutants | **honor** (the hidden tests run in-process) |
| sql | asset descriptor + `schema.sql` + a small sample generator; result spec (columns, ordered, eps) | seeded `gen-hidden.sql`, expected result sets, EXPLAIN checks, an alternative reference query | checked (comparison happens in judge) |
| SD / LLD / behavioral | `text`, `blank` and `canvas` parts; composite grader; public rubric; public reference design | must-cover list, anchors at bands 1/3/5, red flags, synthetic exemplars; LLD adds facade tests | AI (`ai_rubric`); LLD facade tests are **honor** |

### 7.2 CI validation

**Public repo (new `content` job, free):**
- JSON-Schema validation and strict decoding of every file.
- Id and slug guards against `ids.lock.json` and the previous tag: no disappearing ids, no course change, no change to an `id@v`'s bytes.
- Markdown profile: CommonMark + GFM tables and fenced code; images only as `asset:` refs; https links; no raw HTML. An SVG lint rejects `<script>`, `on*=` and external refs.
- Starters are generated and compiled (Go).
- The **reference passes the samples** in the public production runner image, with `--network none`.
- **Stamp gate:** a `hints` or `editorial` section file fails CI unless its `review` stamp is set. Unstamped drafts stay local or in the private repo's `drafts/`. Until a stamp exists, the hint stage shows the pattern name plus a templated generic hint, and the solution stage shows machine-verified reference code and `solution_facts`.
- **Leak lints:**
  - a filename denylist (`*.ans`, `hidden*`, `secret*`, `expected*`, `anchors*`, `exemplar*`, `calibration*`, `submissions/`, `wrong/`);
  - after the build, the embedded-file list is **diffed against an allowlist**.
- Label-edit flag on `key`-graded parts (§3.4).
- The M1a row-snapshot test.

**Local pre-push hook in the public clone** (the only check that sees both halves **before** anything is public):
- If `../xlearn-evalpack` exists, scan the outgoing diff for hidden-case payloads above a minimum length and for anchor and exemplar shingles.
- Never publish hidden-input hashes to public CI.

**Private repo** (runs against public `main`, which needs no credentials):

| Gate | Rule |
|---|---|
| Reproducibility | regenerated cases equal `tests.lock` (seed = `hash(item_id, cmd)`) |
| Validators | **one generic validator derived from public `constraints[]`**; custom code only for structural invariants (BST, connected graph); every `invalid/*` case must be rejected |
| Correctness | the reference agrees with the brute oracle on small cases |
| Wrong solutions | each gets its declared verdict |
| Time limits | TL ≥ 3 × the reference's max, with a floor of 1 s, **measured against a baseline for the production runner profile** (T3), not raw GitHub runners |
| Cross-repo fingerprint scan | again, as a backstop |
| Review-stamp gate | unstamped items are left out of the built pack (so they fall back to self) |
| After each push | an anonymous GET must return 401 or 403 |

- **Deferred until T7 picks that course:** go-race detection rates, SQL mutants and EXPLAIN, and `ai_rubric` calibration.
- **Budget:** changed items only. A full rerun only when the harness, runner or checker version changes. Heavy gates can run locally (`make packcheck`) with stamps committed. That is well under 2,000 min/month (inferred).
- Every author-supplied or AI-written program runs in runner containers with `--network none`.

### 7.3 Where AI may help

| Artefact | AI drafts? | Checked by | Owner must |
|---|---|---|---|
| **Expected outputs** | **Never** | computed from the reference, then checked against the oracle | — |
| **Private keys** | proposes only | blind solve (advisory) | **confirm** |
| Inputs, generators, custom validators | yes | the generic validator, the lock, wrong solutions | review the edge list |
| Go reference | yes | the oracle; samples in public CI | approve the idiom |
| Brute oracle | yes | must be a *different* algorithm | skim |
| Wrong solutions | yes | must fail as declared | glance |
| Statement, constraints, samples | yes, **from the brief only** | templated samples and constraints | **correctness, clarity, originality** (before the owner's own attempt) |
| Hints, editorial | yes | stamp gate | teaching quality and originality (**after** the owner's own attempt) |
| Anchors, must-cover, exemplars | yes (exemplars are synthetic) | calibration (when that course starts) | **label the calibration set** |

- Pack material is drafted **only on API or no-training plans**.
- The platform key is used only for the calibration job (T5).

### 7.4 Effort: v2.0 scope (inferred, bottom-up, with AI; ±40%)

| Scope | Items | h/item | Hours |
|---|---|---|---|
| DSA **self-path tier**: statement, samples, reference Go passing the samples, solution facts, 2 templated probes, hints and editorial stamped after the owner's attempt | 151 (137 new + 14 reworked) | 0.5–0.8 | 75–120 |
| DSA **full pack** upgrade: generators, brute oracle, ≥ 2 wrong solutions, time limit, lock | ~35 (weeks 1–4, including the 14 seeded) | 1.0–1.5 | 35–55 |
| Authoring tooling: packlint, contract hash, generic validator, private CI, pre-push hook and fingerprint scan, embed allowlist, anonymous-GET probe | — | — | 60–90 |
| Second-course pilot (after M3; T7 picks the course) | ~10 | 1–2 | 10–20 |
| **v2.0 total** | | | **~180–285 h** |

- **Excluded:**
  - codecs, checkers and harnesses (T3/T4 engineering);
  - C++ references (until PRD Q6);
  - go-race, SQL and calibration gates;
  - the full 345-item programme, which is about 450–650 h (inferred) and not committed.
- At 10 h a week, the v2.0 content is about 18–28 weeks, **on top of** M1–M5 engineering.
- **Measure the speedup on the first 10 items.** METR found experienced developers 19% slower with AI on their own repos.
- The self-path tier upgrades to auto-grading later with no id or route change.

### 7.5 What comes first
1. Tooling, then packs for the **14 seeded DSA items** as the M3 pilot. They cover function mode, the unordered checkers and LRU's class mode.
2. DSA weeks 1–4 at full-pack tier; the rest of DSA at the self-path tier, in week-sized waves **ahead of the owner's own frontier**.
3. Other courses only once their widget or grader ships, after M3. go-concurrency is the cheapest by reuse, but its tests are in-process (honor); SQL is the cheapest *checked* course. This is input to T7.
4. **M5 (DSA `evaluator_only`) waits** until every live DSA item has a pack (owner Q5).

---

## 8. Content rights stance

These are research findings, not legal advice.

- **Original expression, shared ideas, link out.**
  - Statements, examples, tests and editorials are written from scratch in xLearn's own template.
  - Titles are fine: short phrases aren't protected (Copyright Office Circular 33), and ideas are free (17 U.S.C. §102(b)).
- **LeetCode:**
  - Its terms claim its questions and solutions and forbid scraping, and its DMCA notices targeted wholesale Premium copying.
  - **Never fetch LeetCode programmatically.** Keep outbound links and a "not affiliated" line; no logos.
  - For Premium-only ideas, write original statements and never link the Premium page as the practice target.
- **The 151-item list:** keep xLearn's own 16-week, 4-phase and role arrangement. Don't market it as "NeetCode 150" or reuse its category order verbatim. Compilation risk is low (inferred).
- **SQL:** own schemas, or permissive datasets (e.g. Chinook, MIT) with a `license` field on the asset and a NOTICE file.
- **AI:**
  - Prompt from xLearn's brief and spec only. LLMs memorize popular problems, so the reviewer compares against the reference page once.
  - The **stamp gate** (§7.2) means no unreviewed AI editorial is ever published.
  - Record `authored_by`.
- **Provenance** is required on every item: `{origin: original|adapted|licensed, inspired_by[], license?, attribution?, authored_by}`. CI refuses a missing field and `origin: copied`.
- **Fix now:** v1 copies LeetCode's Example 1 inputs for Two Sum (`problems.json:88`) and 3Sum (`problems.json:249`). Replace them in M1a.
- **Takedown:**
  - `status: withdrawn` blanks the prose;
  - remove the text from HEAD;
  - rewrite history only if a notice demands it;
  - publish a contact address.
- **Licence (decided D9):** **MIT for everything**, content included (the root `LICENSE` already covers it). The private eval pack is proprietary.

---

## 9. Public-dashboard data deltas

### What becomes public
Per course, **only for courses that are enrolled, `public_visible` and active** (v1 walks every active path, `public.go:144`).

| Fact (non-PII aggregates) | Source |
|---|---|
| Concluded vs **passed** (grade ≥ rough) vs total core items | `proj_coverage(+path_slug, concluded, passed)` ← `problem_solved` v2 |
| Difficulty split; topic mastery = max(best first grade, highest passed touch level) | coverage, mastery, curriculum index (gateway joins at read time, per ADR-0018) |
| First-attempt grade mix **with provenance**, plus **"judge-checked %"** = `graded_by=auto AND trust=checked` | `proj_outcome_mix(account, path, graded_by, trust, outcome)` |
| Counted submits, first-submit acceptance %, languages (code courses) | `proj_judge_stats(account, path, language)` ← additive `problem_solved` v2 fields `language`, `counted_submits`, `first_submit_passed`, `trust` (T4 to confirm) |
| Touches completed; Day-7 pass rate | `proj_touch_stats(account, path, level)` ← `touch_scored` (+`trust`) |
| Mocks: count, best %, average % (`total/max_total`), `scored_by` (`ai` vs `ai-byo` vs `self`) | `mock_session` + `path_slug`; never averaged across rubrics |
| **Header totals: solved, streak, heatmap and mocks, computed from visible courses only** | `proj_activity(account, path, date: attempts, touches, mocks)` replaces `proj_heatmap`. This fixes the +5 inflation per first clean solve and the Day-45 gap. v1's header `Totals.Solved` and `Mock` come from the account-wide summary (`public.go:186-200`). Days stay UTC. |

**Never public:**
- code, answers, canvas, drafts;
- feedback prose, mistakes, weak-area categories;
- notes, coach messages, transcripts;
- per-item lists or timelines, sub-day timestamps, Run and arena counts.

### Defaults (owner Q3)
- Profile `public` (ADR-0024).
- Course visibility from the manifest: `true`, except **behavioral** (`false`).
- A private profile returns the same 404 as an unknown user.
- **Any visibility toggle bumps the per-account public cache epoch synchronously.**
- After an erase, the username stays unclaimable for 60 days, so old résumé links don't resolve to a newcomer.

### Enforcement (fixes a v1 gap)
- **Today:** the public handler mints an ordinary `["learner"]` token (`bff.go:703`), and no service checks roles. `mock_session.notes` (C3) sits in assessment, so "assessment holds no PII" isn't an argument.
- **v2:**
  - the gateway mints `roles: ["public-read"]` with `sub` = the profile's account;
  - a shared `RequireRole` middleware;
  - `GET /public/stats?paths=` accepts only that role and returns only the public shape;
  - `/progress/*` rejects it;
  - a separate cache namespace and rate limit (T7).

### Events and replay
- `path_slug` comes from v2 envelopes; version-1 envelopes default to `dsa`.
- All projections are dropped and replayed at M2b, **after** review emits a one-off `touch_scored` backfill (`occurred_at = scored_at`).
- `mock_completed` gains `path_slug`, `rubric_id`, `total`, `max_total` and `scored_by`.

---

## 10. Security and threat notes

**Framing:** hidden-data secrecy protects **signal integrity, not against a determined cheat.** Solutions are public, so a "checked" grade means *a correct program was submitted under the timer*, not that the learner wrote it. For private `key` parts, the key *is* the evaluation.

| Leak or integrity path | How it's closed |
|---|---|
| **NATS has no auth** (`infra/…/messaging/release.yaml:9-10`; `nats.go:44-50` connects with no credentials; single `$G` account; no NetworkPolicies; other projects share the cluster). Anyone reaching `nats:4222` could forge `account_erasure_requested` or `evaluation_completed`, purge the replay streams, or fill them. | **Precondition for M3 and for E**, owned by T7: one NATS user or nkey per service; **publish only on `xlearn.<svc>.>`**; subscribe only to its consumed streams; `$JS.API` create/update/info only for its own stream and consumers; **purge and delete only for one ops identity**; a NetworkPolicy on `messaging` admitting only the `xlearn` namespace. Erase consumers also re-verify with identity (§6.6). practice concludes only for an attempt of that account that is `awaiting_evaluation`, with a matching context. |
| Private data committed to the public repo (permanent on GitHub: PR refs, SHA reachability across the fork network, forks) | Answer-free schema, strict decoding, filename denylist. **Local pre-push fingerprint hook** against the sibling checkout. `.gitignore` and `.dockerignore` list `evalpack/` and `xlearn-evalpack/`. The AGENT.md rule. Private-CI scan as a backstop. **Leak runbook:** treat any slip as disclosed, author new cases and keys, ask GitHub Support to purge cached views and PR refs. |
| Nested private clone lands in an image | Verified: it enters the build context (`curriculum.Dockerfile:14`). `.dockerignore` entries, the **embedded-file allowlist**, and CI building only from git close it. Only the binary reaches the final stage. |
| GHCR package made public (irreversible) | Private by default; created fresh, never a fork or template. Only private CI pushes. Anonymous GET must return 401/403 after every push and on a schedule. Pulls use a machine user. |
| Pod on the node mounts the cached pack image without credentials | Spike check (§3.3); judge-only `imagePullSecrets`. |
| Curriculum API or BFF passthrough (the gateway forwards unknown keys, `aggregate.go:154-184`) | The pack is never seeded into curriculum. A gateway test runs a denylist (`expected`, `args`, `input`, `case_id`, `anchors`, `key`, `aliases`) against `/problems/*` **and every judge response**. |
| **Stage gating was incomplete in v1.** `pattern` is ungated in list, bulk and detail (`curriculum/handlers.go:55-65,238-266`), on the Revision card (`Revision.tsx:235-238`), in the arena (`bff.go:530-538`) and in the coach prompt (`coach.go:~313`), and the coach enters review mode on `solved` (`coach.go:294-302`). | One gateway `withhold(item, live)` applied to: `/problems?ids=`, `/paths/{slug}/problems`, the week view, Today, the revision due queue, mistakes enrichment, the arena GET and the coach (mode and prompt). A route-enumeration test covers them. It is a nudge, so **probes with public keys are honor-grade**. |
| Brute-forcing `key` parts through the arena (unlimited contexts) | judge **never evaluates `key` or `ai_rubric` parts in `arena` or `run` contexts**; it returns `not_evaluated_here`. Counted contexts only, once per context. A judge test covers this. |
| **In-process profiles** (go-concurrency hidden `_test.go`, LLD facade tests): the learner's `init()` can read the test sources or binary strings, or forge `--- PASS` / test2json lines (the SWE-bench #667 class) | Stated exception to "expected data never enters the sandbox". Compile the test binary in a step the learner's code doesn't run in, and execute with no hidden sources on disk. **Every test name the pack declares must report pass**, so a missing result is a failure; never trust exit 0 or "ok". `-count=1`. Evaluation `trust=honor`, left out of "judge-checked %". Prefer serialized-output harnesses wherever the problem allows. |
| State persisting across jobs (GOCACHE, the test cache, a reused filesystem; a runner inside the judge pod) | T3 hard constraints: a fresh tmpfs per job; caches read-only and prebuilt from public content only; `GOFLAGS=-count=1`; the runner **never** shares judge's pod, PID or mount namespace. Isolation test: job N writes a marker, and job N+1 from another account must not see it. |
| Hidden data reaching a visible channel | **Run** contexts materialize only public samples and generators. Hidden generators, instances and tests are used only for counted or arena **submits**, whose output is **verdict-only**. |
| Feedback-channel extraction | The learner-facing Evaluation DTO is an **allowlist**. For hidden cases it carries the **passed count plus the first failure class** (WA/TLE/RE/MLE), with no ordinals, case ids, tags, stdout/stderr, EXPLAIN detail or per-case timings. Resource usage is aggregate (max, bucketed). This **overrides T0 §10's per-case resource usage** for hidden cases. `reason` is an enum. Per-account quotas apply. Rationale is released only after a counted evaluation. |
| Checker errors, 500s, panics or logs carrying want/got values (the Recoverer logs panic values, `httpx.go:59-68`) | Checkers return typed verdicts. judge never logs case data or learner code. A log lint in CI. |
| Pack material in LLM calls | **Invariant:** anchors, must-cover lists and exemplars enter **only a platform-key scoring call**, with `store:false` or zero retention. **No pack material in feedback calls** (T5). BYO-key evaluations (T6 mocks, arena) get only public rubric descriptors and are labelled `ai-byo`. |
| judge read APIs (IDOR) | There is **no service-auth mechanism**: `/internal/*` is unauthenticated and ClusterIP-only (ADR-0016 §3). Trust model: payloads there are low-sensitivity, and the sandbox must not reach them (T7 NetworkPolicy first). Every learner read by id is scoped to the JWT `sub`; ids are never relied on to be unguessable. IDOR tests. |
| Postgres logs capturing learner content | Tables that hold bodies have only PK and FK constraints; validation happens in Go before insert (§4). |
| Backup restore resurrects erased learners | Erase ledger outside the CNPG backup, replayed as a mandatory restore step (§6.6). |
| Cluster secrets | k3s Secrets aren't encrypted at rest (opt-in), and node root can read the containerd store. Accepted, since ops is trusted; optional k3s secrets encryption (T7). |
| Public media and Markdown | React-node renderer, never `dangerouslySetInnerHTML` (`Markdown.tsx:4-10`). Images only via `asset:`. SVG served with `image/svg+xml`, `nosniff` and a sandbox CSP, and rendered only through `<img>`. |

**PII rules:**
- C3/C4 never appears in events, logs, the public route, or the coach context without the gate.
- C4 needs consent before any platform-key call and is private by default.
- Events carry only C2 **and are authenticated**.
- Calibration and exemplar data is synthetic or owner-authored. If real answers are ever used, that needs explicit consent and a removal procedure.

---

## 11. Single-node and ops fit

- **Postgres volume:** content adds < 10 MB and the pack adds 0. Growth is about 86 KB per active learner-day (§6.2). The owner alone uses about 0.03 GiB a year. Resize at 60%, which is about 285 learner-years.
- **Memory:** the content working set is tiny, and TOAST handles long prose. judge's index of keys and anchors is a few MB; cases are read lazily from the mount. The 1 Gi CNPG limit holds; revisit at the resize trigger.
- **WAL:** drafts are single-row inline upserts (15 s debounce, 60 s for canvas) with `fillfactor=70`, no index on `updated_at` and unchanged key columns, so updates stay **HOT**. There is no artefact GC churn. Size T2's WAL archive for draft churn (inferred: ~0.5 MB per learner-day).
- **Backups:** nothing is backed up today. **T2's CNPG PITR is the single critical backup** (learning record plus outbox event log) and a gate before grading opens to non-owner learners. JetStream needs no snapshots. The pack and public content are in git.
- **NATS:** stream budget of 3.25 of 5 GiB with a sum test (§6.5). `XLEARN_JUDGE` MaxAge 14 d. Replay streams use `Discard=New`. No per-message TTL on 2.14.6.
- **Node disk:** pack images of 30–40 MB each at most, GC'd by containerd. No tmpfs or memory cost.
- **CPU:** all content validation runs in GitHub Actions or locally, never on the node. judge's startup hash check takes about 1 s (inferred).
- **Alerting (none exists; only metrics-server):**
  - T1 names **one `xlearn-opscheck` CronJob**: daily, running a subcommand in an existing image.
  - It checks: `spec_mismatch > 0`, evaluable = 0, PG and NATS usage ≥ 60/70%, erase requests open > 24 h, GHCR anonymous GET ≠ 401/403, PAT expiry within 14 d.
  - It pushes to an ntfy topic or email (T7 picks).
  - It is backed by **UI degradation badges** ("grading pending: self-report") so nothing depends on someone reading logs.
- **No new database engine.** Nothing needs full-text search, a document store or a key-value store:
  - the pack is an OCI artefact;
  - per-user payloads fit Postgres with lz4 TOAST;
  - only T2's blob kinds need object storage.

---

## 12. What T1 constrains downstream

**T2 (object storage and backups)**
- Blob kinds are only: canvas scenes and drafts, transcripts, audio. Video is not stored.
- The key layout is `u/<account_id>/<kind>/<id>`, so erase is a prefix delete.
- Presigned direct upload is for audio only.
- CNPG PITR 14–30 d, which is also the backup erase SLA, plus a WAL archive sized for draft churn.
- **No JetStream snapshots.** The outbox is the rebuild source (runbook: `sent_at = NULL`).
- The erase ledger lives outside the CNPG backup, and the **restore runbook replays it**.
- No pack bundle in any in-cluster bucket.
- CNPG backups must exist **before grading opens to non-owner learners**. M3's code does not depend on T2.

**T4 (judge contract)**
- `payload_ref` = (submission_id, part_id). Run and arena payloads and outputs live on the job row, deleted 24 h after completion.
- Counted contexts (course, touch, mock) **and arena submits** persist submissions (D10). **Only counted contexts emit events.**
- **Learner-facing Evaluation DTO allowlist** (§10).
- `key` and `ai_rubric` parts return `not_evaluated_here` in run and arena contexts, and are evaluated once per counted context.
- Defined results: `inconclusive(contract_changed)` and the item-level `spec_mismatch` capability.
- Evaluations record pack version and digest, `item_hash`, grader, harness, runner, rubric and model versions, and **`trust`**; the digest is recorded when the job is **claimed**.
- Additive `problem_solved` v2 fields to confirm: `language`, `counted_submits`, `first_submit_passed`, `trust`. `touch_concluded` also gains `trust`.
- The worker checks tombstones before writing a result.
- The draft sweeper deletes drafts after 90 d idle.
- Every read by id is scoped to the JWT `sub`.

**T3 (sandbox)**
- Runner images are public and identical in CI and production. Versioned `harness@v` codecs; closed type registry.
- Comparison happens outside the sandbox for function mode and SQL. **In-process profiles (go-race, facade) are honor-grade:** a separate compile step, no hidden sources on disk at execution, all declared tests must report pass, `-count=1`.
- A fresh tmpfs per job; read-only caches built from public content only; never inside judge's pod or namespaces; a cross-account isolation test.
- Run uses public samples only. Hidden feedback is the passed count, the first failure class and aggregate bucketed usage (**overrides T0 §10**).
- Publish a time-limit baseline for the production runner profile for pack calibration.
- `sql-pg` builds hidden instances only for submits, with deterministic EXPLAIN.
- The launch language set (PRD Q6) sets how many reference solutions are needed.

**T5 (platform AI)**
- Pack material only in **platform-key scoring calls** with `store:false` or zero retention. **Never in feedback calls.**
- BYO-key scoring sees only public descriptors.
- The calibration set is synthetic or owner-authored and held out; it doubles as the model-change regression set, gated when the first AI-graded course starts.
- Prefer `key` over AI wherever alias or set matching works.
- AI prose is capped at 8 KiB and stored in judge. C4 needs consent.

**T6 (interviewer)**
- It runs on the BYO key (T0), so AI mock scoring uses **only public rubric descriptors**, labelled `scored_by=ai-byo`. Pack anchors never leave judge except in a platform-key call.
- Transcripts and audio have a non-assessment owner.

**T7 (rollout)**
- **NATS auth plus the messaging NetworkPolicy before M3 and before the erase step E.**
- The stream `MaxBytes` budget, with a sum test and an options knob.
- Private repo (created fresh), private CI, private GHCR package, **machine user**. Pull secret in `xlearn` and `flux-system`; ImageRepository `secretRef`; ImagePolicy `>=1.0.0 <2.0.0`; judge `extraVolumes`.
- Image-volume spike, including the cached-image credential check; the initContainer knob off by default.
- The `opscheck` CronJob and a push channel. UI degradation badges.
- Pre-push hook, `.gitignore`/`.dockerignore` entries, the AGENT.md rule.
- Erase endpoint and UI; visibility toggles; `public-read` role and rate limit.
- Compose parity: Postgres 18, NATS 2.14.
- Resize trigger at 60% for both PG and NATS.
- ADR-0027.
- Markdown renderer upgrade: tables, fenced code, images, links.
- Choose the second course: go-concurrency (cheap, honor) vs SQL (cheap, checked).

---

## 13. Alternatives considered

| Option | Why not |
|---|---|
| SOPS/age-encrypted pack in the public repo, baked into judge | The ciphertext is public forever: one key leak exposes every past version, and metadata leaks anyway. Diffs can't be reviewed. It contradicts ADR-0026 §2. |
| ConfigMap or Secret from the private infra repo | 1 MiB per object, tmpfs counts against pod memory, Secrets are unencrypted in kine, and it mixes content with infra. |
| Pack rows in judge's Postgres | Uses the unbacked 10 Gi / 1 Gi instance and needs a privileged load endpoint. Even the index isn't worth a table: it lives in memory. |
| judge clones or ORAS-pulls at runtime | A GitHub token and egress in production, and no CI gate or GitOps version. ORAS remains fallback #2 behind `EVALPACK_DIR`. |
| CI uploads a tarball to T2 object storage | Couples M3 to T2 and gives CI production write credentials. |
| All content private (two images, capability majors) | Rejected by T0 §11. Only the pack is private. |
| **Broad `spec_hash` + pinned `PUBLIC_REF` + daily bump bot + pack-first release order** | Rebuilds the version-skew machinery T0 §11 rejected. Every typo would desync the pack. A narrow `contract_hash` changes only when hidden data must change anyway. |
| **Content-addressed `artefact` table with refcount GC** | GC for payloads ≤ 64 KiB, with autosave churn that breaks HOT updates. Inline `body` plus a nullable `object_key` gives the same later move to T2 with no GC. |
| Persisting Runs | Most of the modelled 40% growth, for data that never counts. Runs stay on the job row only. (Arena **submits** are kept per D10.) |
| **Pruning outbox and inbox + JetStream snapshots in T2** | ADR-0018 already replays from NATS, so pruning removes the only backed-up copy and creates a new backup duty. Keep the outbox; revisit at the resize trigger. |
| `MaxBytes 4 GiB` per replay stream | Can't be created: reservations would exceed `max_file_store` 5 Gi. Budgeted limits instead. |
| **Account-suffixed subjects in M1a** | A single-token `*` filter and exact-match `switch e.Subject` (`review/consumers.go:53-63`, `projections.go:156-166`) would silently drop and ack the events. Needs a two-release migration; offered as owner Q4 (b). |
| Private keys for DSA pattern and complexity | The answers are public already (`pattern`, the editorial). Templated honor-grade probes instead. |
| `course_asset`, `problem_asset`, `rubric` and `path.manifest` tables; a `policy_snapshot` table | Copies of embedded data that nothing queries relationally. Served from the embedded files; `stage_params` plus `policy_version` on the attempt. |
| JCS canonicalization; `spec_rev` counter; hashes in `ids.lock.json` | Every reader shares one Go package: typed re-marshal. Hashes are computed at build; the lock holds only id→course and `id@v`→sha256. |
| Golden test that keeps both the v1 and v2 loaders | A one-shot converter plus a snapshot of v1 seeded rows; the old loader is deleted in the same PR. |
| Kattis `.in`/`.ans` stdin cases | Don't fit function-signature problems. Kattis *semantics* are adopted instead: expected verdicts, invalid inputs, time-limit margins, CI-materialized data. |
| Per-item checker programs at runtime | Untrusted, un-audited code in judge. Closed registry instead. |
| AI-predicted expected outputs or keys | Weak tests pass wrong code (EvalPlus). Outputs are computed; keys are owner-confirmed. |
| Unreviewed AI editorials published at once | Rights and quality exposure is highest ahead of the owner's frontier. Stamp gate instead. |
| Per-course validators and C++ references for all items | Cost without a v2.0 need. Generic validator from `constraints[]`; Go only until PRD Q6. |
| Eval-pack bundle mirror in the backup bucket | Puts generators and oracles where in-cluster credentials can reach them, for no gain over GitHub plus a local clone. |
| Relying on "alert" without an alert stack | Log lines nobody reads. `opscheck` plus UI badges. |
| NATS per-message TTL | Leaks on 2.14.6 (nats-server#8594). |
| Crypto-shredding for erase | Events carry no prose. Delete plus tombstone plus ledger gives the same result and replays deterministically. |
| Mock pools as selection rules | A DSL. Explicit id lists instead. |

---

## 14. Owner decisions — resolved 2026-09-24

| # | Question | Decision |
|---|----------|----------|
| D5 | Where the private eval pack lives and how it reaches judge | **Private sibling repo `xlearn-evalpack`** (created fresh; pulled by a machine user) → a **private GHCR data image** → **judge-only read-only image volume** behind `EVALPACK_DIR`. Fallbacks: initContainer copy, then ORAS. Spike before M3. |
| D6 | v2.0 content scope | **All 151 DSA items at the self-path tier, plus full packs for weeks 1–4 (~35), plus a ~10-item second-course pilot after M3.** M5 (DSA evaluator-only) waits until every live DSA item has a pack. About 180–285 h. |
| D7 | Public-profile defaults | **Profile public. Each course visible by its manifest default (behavioral hidden). Every header aggregate is computed from visible courses only.** The learner can hide any course or the whole profile. |
| D8 | Account erase at v2.0 | **Delete rows and outbox rows, write tombstones, and keep an erase ledger** that is replayed after any restore. Pseudonymous NATS events are left in place, with a re-seal runbook on request. Requires NATS auth first. |
| D9 | Licence | **MIT for everything**, content included. The eval pack is proprietary. The rights stance is unchanged: original statements, outbound links only, provenance on every item. |
| D10 | Problems-arena history and timers | **Arena Submits are kept as a submission history** (Runs are not). **The arena and the course have separate done markers.** A course attempt starts its server timer when the attempt starts and shows the full question plus the coding area. **The arena has a manual timer, off by default** (client-side, no grading effect). Arena submits and markers never count toward course progress, grades or learning signals, and are learner-private in v2.0. |

## 15. Risks

- **Content is the critical path.** v2.0 is about 180–285 h on top of M1–M5 engineering, with ±40% and a plausible 2× tooling overrun. Mitigations: the self-path tier by default, measuring the AI speedup on the first 10 items, deferring heavy gates.
- **The author is also the learner.** Mitigations: statement review before the attempt; hints, editorial and test stamps after it; keys for SD and LLD can come later.
- **Silent downgrade on a contract mismatch.** Mitigations: UI badge, `opscheck`, and authoring contract changes together with the pack.
- **NATS auth slips past M3.** The erase path and judge-driven conclusion would then be forgeable. It is a hard precondition in T7; M3 does not start without it.
- **Image volume can't start the pod.** If the image is GC'd *and* the PAT has expired, judge's pod can't start and grading is down (the kill switch keeps self-report working). Mitigations: machine-user PAT, expiry check in `opscheck`, spike before M3.
- **A GHCR public flip is irreversible,** and a public-repo slip is permanent. Mitigations: pre-push hook, CI gates, leak runbook. Recovery means authoring new hidden tests.
- **Honor-grade inflation.** In-process profiles and public-key probes could be mistaken for checked results. `trust` is stored and shown, and "judge-checked %" excludes them.
- **Hidden-test extraction through feedback.** Mitigations: count plus class only, bucketed usage, quotas, key parts only in counted contexts.
- **Rights exposure from memorized AI text.** Mitigations: brief-only prompts, stamp gate, provenance lint, the two copied examples fixed in M1a.
- **Weak tests pass wrong code.** Mitigations: oracle agreement, wrong-solution gates, the generic validator. Go-race flakiness is deferred with its course and maps to `inconclusive`.
- **Integrity moves from DB CHECKs to loader validation.** Mitigations: the row-snapshot test and registry validation in CI.
- **Postgres and NATS grow without pruners.** Mitigation: the resize trigger at 60%, about 285 learner-years; decide retention then.
- **No backups until T2.** Mitigation: T2 gates grading for non-owner learners. The outbox is the NATS rebuild source.
- **Go embed traps** (dotfiles, dropped nested `go.mod`, `_` directories). Mitigation: the embedded-file allowlist.
- **GHCR private quotas** (500 MB storage, 1 GB/month transfer) if billing applies. Mitigations: per-course layers, keep the last 10 versions.
- **The heatmap redefinition changes the owner's history view.** The `touch_scored` backfill must run before the M2b replay.

---

## 16. Critic findings addressed

**Leaks and threat model (critic 1)**
- **NATS has no auth (blocker):** made a hard precondition for M3 and for erase, with per-service publish ACLs, ops-only purge/delete and a messaging NetworkPolicy. Erase consumers re-verify with identity. NATS rows added to §10 and T7.
- **In-process profiles contradicted "hidden outputs never enter the sandbox" (major):** stated as an explicit exception, honor-grade, with a separate compile step, all declared tests required to pass, `-count=1`.
- **Cross-job state (major):** T3 constraints: fresh tmpfs, read-only public caches, no shared pod or namespace, isolation test.
- **Public "private" keys and incomplete stage gating (major):** DSA probes are templated from public fields and honor-grade. The full `withhold()` surface list includes the coach mode and prompt. Gating is labelled a nudge.
- **Arena brute-forcing of keys (major):** `key` and `ai_rubric` are never evaluated in run or arena contexts.
- **BYO-key exposure of anchors (major):** pack material only under the platform key with zero retention; BYO scoring uses public descriptors, labelled `ai-byo`.
- **Judge response leaks (major):** allowlisted learner DTO; case ids internal; no pack material in feedback calls; typed checker verdicts; denylist test on judge responses.
- **Git permanence (major):** local pre-push fingerprint hook, leak runbook, repo created fresh, both names ignored, AGENT.md rule.
- **Erase gaps (major):** erase ledger replayed after restore; synthetic calibration and exemplars; `outbox.account_id` deleted in the erase transaction.
- **Per-case usage and failing index (minor):** passed count plus failure class, aggregate usage; explicitly overrides T0.
- **Run/SQL visible output (minor):** Run uses public samples only; hidden data only on submits, verdict-only.
- **Service-auth claim (minor):** corrected. The real trust model is stated, with JWT-`sub`-scoped reads and IDOR tests.
- **Public deltas (minor):** all header aggregates from visible courses; cache epoch bumped on toggle; "judge-checked" excludes honor grades; 60-day username cooldown (hashed).
- **Pack exposure (minor):** bundle mirror dropped; machine user required; no-training plans for authoring; credential check in the spike.
- **PII in Postgres logs (minor):** body tables have no fallible constraints; `store:false` on LLM calls; disclosure.

**Solo-dev cost and single-node ops (critic 2)**
- **Stream MaxBytes uncreatable (blocker):** a 3.25 GiB budget against the 5 Gi store, with a sum test.
- **Cross-repo skew machinery (major):** narrow `contract_hash`; no `PUBLIC_REF` pin, no bump bot, no release dance.
- **Premature retention (major):** caps and quotas only; the outbox kept as the durable copy; no JetStream snapshots.
- **Artefact GC (major):** inline draft and part bodies; Runs only on job rows; `NOT NULL DEFAULT ''` keys.
- **Account-suffixed subjects in M1a (major):** moved to owner Q4 (b) as a two-release migration; spec and hash columns moved to M2a/M3; tombstone checks only on async paths.
- **Understated effort (major):** v2.0 scope re-based (self-path tier by default, ~35 full packs, Go only, generic validator, gates deferred): **~180–285 h**; owner Q5 added.
- **Public DSA keys (major):** templated from public fields, with no private key files.
- **No alert stack (major):** `opscheck` CronJob plus UI badges.
- **Unreviewed public editorials (minor):** stamp gate on hints and editorials.
- **Redundant tables (minor):** manifests, rubrics and assets served from the embedded files; pack index in memory; `policy_snapshot` dropped.
- **Hash machinery (minor):** typed re-marshal, no JCS, no `spec_rev`, a slimmer lock.
- **CI minutes and time-limit calibration (minor):** changed items only; local heavy gates; production runner baseline.
- **GHCR transfer and backups (minor):** transfer quota noted; CNPG backups gate grading for non-owner learners.
- **Golden test keeping the v1 loader (minor):** one-shot converter plus a row snapshot.

**Claims corrected:**
- (1) NATS is *already* the replay source (ADR-0018:75-78).
- (2) The 4 GiB `MaxBytes` couldn't be created.
- (3) The HOT claim contradicted the artefact design.
- (4) The DSA "private" keys were public.
- (5) T0 §11 also rejected skew machinery.
- (6) Stage gating was incomplete.
- (7) The in-process exception to "hidden outputs never enter the sandbox".
- (8) No service auth exists.
- (9) The outbox had no `account_id`.
- (10) Tombstones don't survive a restore.
- (11) "C2-only events" is confidentiality, not integrity.

**Strengths kept:**
- private repo → private OCI image → image volume, judge only;
- answer-free schema, strict decoding, embed allowlist, `.dockerignore` entries;
- comparison in judge, verdict-only feedback;
- enforced `public-read`;
- C2-only events;
- the erase path with tombstones; retire/withdraw as takedown;
- schema-per-service roles (plus the new `xlearn_judge` role);
- `cadence: final` for keys;
- no new DB engine;
- the embed and Docker experiments;
- computed expected outputs and closed checkers;
- the self-path tier and "judge Ready with zero evaluable items";
- size caps that fit the 1 MiB limits;
- the `touch_scored` backfill before replay;
- the rights stance.

---

### Appendix: verification and sources

**Re-checked this session (repo and infra):**

| Claim | Evidence |
|---|---|
| NATS has no auth | `infra/infrastructure/messaging/release.yaml:9-10` ("No auth in v1 … tighten in the S12 hardening pass"); `internal/platform/events/nats.go:44-50` connects with no credentials |
| Stream config is hard-coded | `nats.go:78-83` |
| Live store limit | critic.md §5 (`max_file_store: 5Gi`) |
| NATS is already the replay source | `docs/adr/0018:75-78` |
| Outbox tables have no `account_id` | identity `00001:72-78`, review `00001:75-81`, assessment `00001:83-89` |
| Exact-match dispatch | `review/consumers.go:53-63`; `assessment/store/projections.go:156-166` |
| Single-token stream subjects | `practice/service.go:20`, `review/service.go:27,44` |
| Public header totals are account-wide | `gateway/public.go:186-200` |
| Coach mode keyed on solved; pattern in the prompt | `gateway/coach.go:294-302`, `~313` |
| Recoverer logs the panic value | `platform/httpx/httpx.go:59-68` |
| Per-service schema owner roles | `infra/infrastructure/database/cluster/xlearn-database.yaml` |

- The draft's own file:line claims were accepted as re-verified by critic 2.
- The volume model was recomputed with the final design: scratchpad `t1/vol_final.py`.

**Sources:**
- Draft sources are retained: Kubernetes image volumes, KEP-4639, k3s v1.36 notes, GHCR visibility and billing, Flux ImagePolicy, Go embed, PG TOAST and 18 release notes, CNPG and Longhorn, NATS TTL and nats-server#8594, Kattis/testlib/DMOJ/CMS, EvalPlus, METR, LeetCode ToS and DMCA notices, Circular 33, 17 U.S.C. §102, Project Euler licence, Chinook, GDPR Art. 17 and Recital 26, DPDP §12.
- Added from the critiques:
  - NATS chart values (`fileStore.maxSize` defaults to the PVC size): https://raw.githubusercontent.com/nats-io/k8s/main/helm/charts/nats/values.yaml
  - JetStream storage reservation (error 10047): https://github.com/nats-io/nats-server/issues/4281 and https://github.com/nats-io/nats-server/issues/3321
  - GitHub cross-fork object reference: https://trufflesecurity.com/blog/anyone-can-access-deleted-and-private-repo-data-github
  - Test-marker spoofing: https://github.com/SWE-bench/SWE-bench/issues/667
  - KEP-2535 (ensure secret-pulled images): https://github.com/kubernetes/enhancements/issues/2535
  - GitHub Packages billing (storage and transfer): https://docs.github.com/en/billing/concepts/product-billing/github-packages

**Files read:**
- `docs/adr/0026-per-course-extensibility-model.md`
- `docs/v2/research/t0-extensibility-frame.md`
- `docs/adr/0018-progress-projection-grain-and-rebuild.md`
- `scratchpad/orient/critic.md`
- `scratchpad/t1/vol.py`

**Model script written:**
- `scratchpad/t1/vol_final.py`