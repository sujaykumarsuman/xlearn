# Sprint m3-05 — judge service: skeleton, schema, evalpack loader, erase consumer (M3-1)

> **Milestone:** M3 — judge plus code grader (**M3-1**, judge dark)   ·   **Track:** product (order 42)
> **Prereqs:** [m3-02](sprint-m3-02.md) (the synthetic `fixture` course + pack, `internal/packspec`, the compose `x-evalpack-mount` anchor) · [l-01](sprint-l-01.md) (`internal/platform/erase` + `events.Terminal`, v1.11.0) · for the compose end-to-end erase, [l-02](sprint-l-02.md) (v1.12.0)   ·   **Unblocks:** [m3-06](sprint-m3-06.md) (queue, lane, graders) → [m3-14](sprint-m3-14.md) → [m3-07](sprint-m3-07.md) (v1.13.0 + MI-13)
> **Release action:** **merge only (ships dark in v1.13.0**, tagged by [m3-07](sprint-m3-07.md)). No infra PR here: the judge ACL, DB 4-step and HelmRelease are m3-07's.
> **Calendar:** early November (after v1.11.0/v1.12.0).
> **Execute with:** [`../prompts/prompt-m3-05.md`](../prompts/prompt-m3-05.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Service skeleton: `cmd/judge`, `internal/judge`, port 8087, health, pool 8, Dockerfile, 8th `deploy.yml` job, compose (evalpack anchor + dev content overlay), sqlc | X | ⬜ |
| 2 | Schema: goose migrations for the judge tables (incl. `arena_progress`) | X | ⬜ |
| 3 | Evalpack loader (Ready with zero), `GET /internal/evaluable`, "no handler serves `EVALPACK_DIR`" tests | X | ⬜ |
| 4 | Events: `XLEARN_JUDGE` live, `XLEARN_IDENTITY` erase consumer, dead letters, topology + golden re-render | X | ⬜ |
| 5 | AuthN (`aud=judge`), IDOR scaffolding, kill switch `JUDGE_ADMISSION` | X | ⬜ |
| 6 | Verify, docs, ADR, hand-off notes for m3-07 | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> the **M3** milestone row, the flag inventory, the decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m3-02](sprint-m3-02.md) merged: the synthetic **`fixture` course (`fx-001..fx-003`)** — its public content under
      `internal/judge/testdata/content/courses/fixture/`, its source pack under `packsrc/` and the built pack at
      `internal/judge/testdata/pack` (the real `format_major`, `/manifest.json` with per-file sha256); `internal/packspec`
      (`ReadManifest`, `VerifyFiles`, the `cases.jsonl.zst` reader); and the compose **`x-evalpack-mount`** anchor
      (`source: ${EVALPACK_DIR:-./internal/judge/testdata/pack}`, `target: /evalpack`, read-only). The fixture course is
      **not** in the embedded curriculum, so without task 1's dev content overlay judge is Ready with 0 evaluable (m3-02's
      hand-off in its decisions log).
- [ ] [l-01](sprint-l-01.md) merged (v1.11.0 live): **`internal/platform/erase`** (`NewHandler`, `NewHTTPVerifier`,
      `AckEventID`, the `Store.EraseTx` interface, the handler outcome table) and **`events.Terminal`** (dead-letter + `Term()`
      on the current delivery); the four L-E consumers; identity collects the acks that `topology.go` expects.
- [ ] On `main`: [mi-05](sprint-mi-05.md) (`topology.go`, ACL renderer + goldens, dead-letter hook, `events.Dial` client options,
      the `PG_MAX_CONNS` helper); [m1-02](sprint-m1-02.md) (v2 envelope); [m1-01](sprint-m1-01.md)/[m1-09](sprint-m1-09.md)
      (`internal/course` + the embedded `all:courses` loader); [m3-01](sprint-m3-01.md) (`internal/course/canon`: `ContractHash`).
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no peer creating `cmd/judge` or `internal/judge/`
      (beyond m3-02's `testdata/`) or editing `topology.go`, `deploy.yml`, `sqlc.yaml` or `internal/course`'s loader right
      now (if one merged first, rebase and re-render the goldens).

_Informational, not a gate:_ [l-02](sprint-l-02.md) (v1.12.0: identity's erase producer, `identity admin account erase <user>`,
`identity admin erasures`). The calendar puts it before this sprint; task 6's **compose end-to-end** erase needs it. If it
hasn't merged, run the judge-side e2e only (task 4) and hand the compose end-to-end to [m3-07](sprint-m3-07.md)'s verify
(decisions log).

_Not a gate here:_ the M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) blocks the M3 UI
sprint ([m3-11](sprint-m3-11.md)). This sprint ships dark backend code only; nothing it adds is reachable by a learner
before [m3-13](sprint-m3-13.md) sets the gateway's `JUDGE_BASE_URL`.

## Goal

Stand up **judge — the 8th service** — as a course-blind evidence engine that is safe to deploy before it does anything:
its schema, a pack loader that **can never crash-loop** (a bad or missing pack means **Ready with zero evaluable items**),
the `GET /internal/evaluable` read practice pins attempts from, and its **`XLEARN_IDENTITY` erase consumer from birth**
([rollout §4 M3](../rollout-plan.md#4-per-milestone-detail) "cohort and erase"). judge verifies `aud=judge` JWTs, scopes
every read to the token subject, and honours a permanent kill switch. The queue, runner lane and graders come next
([m3-06](sprint-m3-06.md)); the learner API and admission after that ([m3-14](sprint-m3-14.md)); production (DB 4-step, ACL,
HelmRelease) in [m3-07](sprint-m3-07.md).

## Scope

**In**
- `cmd/judge/main.go`, `internal/judge/{config,service,handlers}.go`, `internal/judge/store/` (goose + sqlc),
  `internal/judge/pack/`, `deploy/judge.Dockerfile`, the 8th `deploy.yml` job, a compose `judge` service, the local
  `judge` schema, the `sqlc.yaml` block.
- The **dev content overlay** m3-02 handed over (`FIXTURE_CONTENT_DIR`, honoured only with `DEV_AUTH`): a helper in
  `internal/course`, wired into judge and the other services the compose e2es cross (curriculum, practice, and the
  gateway if it loads `internal/course`), so compose and the e2es see the `fixture` course.
- The judge schema ([t1 §4 judge](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual) + [t4 §11.4 #11](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag)), `arena_progress` (D10) included.
- Pack loader + `GET /internal/evaluable?path=`; the "no handler serves `EVALPACK_DIR`" tests.
- `XLEARN_JUDGE` publisher + outbox relay; the erase consumer on `XLEARN_IDENTITY`; `judge.event_dead_letter`;
  `topology.go` entries and re-rendered ACL goldens.
- `aud=judge` verification, IDOR scaffolding, the `JUDGE_ADMISSION` kill switch.

**Out**
- Job queue, runner lane client, graders, contexts, `evaluation_completed`, internal context endpoints, the sweeper,
  `RELAY_INTERVAL` → [m3-06](sprint-m3-06.md).
- Admission L9–L13/L15 enforcement, L6 body caps, idempotency replay semantics, the learner API + DTO allowlist, drafts,
  arena history, the `arena_progress` upsert and read, telemetry, `judge admin` → [m3-14](sprint-m3-14.md).
- Infra: the NATS ACL + seed, the ADR-0005 4-step, `apps/xlearn-judge.yaml`, the evalpack ImagePolicy, `v1.13.0` →
  [m3-07](sprint-m3-07.md). Gateway `aud=judge` minting and BFF routes → [m3-09](sprint-m3-09.md).
- `ai_rubric`, `usage_ledger`, `evaluation_dispute` → M4 ([m4-02](sprint-m4-02.md), [m4-04](sprint-m4-04.md)).

## Tasks

### 1 · Service skeleton [X]

Mirror `cmd/review/main.go` and `internal/review/config.go`, the newest v1 services:
- **`cmd/judge/main.go`:** `-version` (ldflags `-X main.version`), migrate under the advisory lock before serving (refuse
  to serve on failure), `pgxpool` with **`MaxConns` 8** through mi-05's helper (`PG_MAX_CONNS` override; L21), the pack
  load (task 3), the publisher + relay and the erase consumer (task 4), the HTTP server, signal handling. Shutdown order:
  stop the consumer → stop the relay → `srv.Shutdown` → close the pool.
- **`internal/judge/config.go`** (12-factor, `LoadConfig()`):

  | Env | Default | Notes |
  |---|---|---|
  | `PORT` | `8087` | ADR-0035 / rollout MI-13 |
  | `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`, `PGSEARCHPATH` | …, `xlearn_judge`, …, `judge` | creds from the SOPS `xlearn-judge-db` secret in prod (m3-07) |
  | `JWKS_URL`, `JWT_AUDIENCE`, `JWT_ISSUER` | gateway JWKS, **`judge`**, `xlearn-gateway` | ADR-0006 |
  | `NATS_URL`, `NATS_NKEY_SEED_FILE`, `NATS_INBOX_PREFIX` | — | mi-05 client options; fail closed when `NATS_URL` is set |
  | `IDENTITY_BASE_URL` | — | erase re-verify (task 4). **When `NATS_URL` is set and `IDENTITY_BASE_URL` is not, judge refuses to start** — l-01's rule: an erase consumer never runs without its re-verify path. With no `NATS_URL` there is no consumer at all. [m3-07](sprint-m3-07.md)'s HelmRelease sets it |
  | `EVALPACK_DIR` | — | unset or empty → 0 evaluable (task 3) |
  | `FIXTURE_CONTENT_DIR` | — | the dev content overlay (below); **ignored unless `DEV_AUTH` is set** (then an ERROR log) |
  | `DEV_AUTH` | unset | read only to gate `FIXTURE_CONTENT_DIR` (the ADR-0033 §3 guard pattern); never set in prod |
  | `JUDGE_ADMISSION` | `open` | the L15 kill switch (task 5) |
  | `PG_MAX_CONNS` | `8` | mi-05 helper |
- **`internal/judge/service.go` / `handlers.go`:** `GET /healthz` (live), `GET /readyz` (**DB ping only** — never the pack,
  never NATS), `GET /internal/evaluable` (task 3). No learner route yet.
- **`deploy/judge.Dockerfile`:** a copy of `deploy/review.Dockerfile` for `./cmd/judge` (static, CGO-free, distroless
  nonroot, `-X main.version=${VERSION}`, `PORT=8087`, `EXPOSE 8087`). `deploy/version_test.go` globs `*.Dockerfile`, so it
  covers judge; confirm `judge -version` echoes the probe.
- **`.github/workflows/deploy.yml`:** an 8th job `judge` (`needs: release-line`, `packages: write`, image
  `ghcr.io/sujaykumarsuman/xlearn-judge`, `dockerfile: deploy/judge.Dockerfile`); update the header comment ("all eight").
- **Dev content overlay** (m3-02's hand-off): the `fixture` course lives in `internal/judge/testdata/content`, not in the
  embedded `all:courses`, so a pack item from the fixture has no live public item and would be `invalid` (task 3).
  - `internal/course` gains a small helper beside the embedded loader: given `FIXTURE_CONTENT_DIR`, it adds the directory's
    `courses/<slug>/` to the loaded course set as if embedded. It **only adds** courses: a slug that already exists in the
    embed is an error. It is honoured **only when `DEV_AUTH` is set**; a set `FIXTURE_CONTENT_DIR` without `DEV_AUTH` is
    ignored and logged at ERROR (fail-safe, like `SIGNUP_MODE`). In judge, an overlay error is logged at ERROR and judge
    continues without the overlay (Ready with zero, never a crash).
  - Wire it into **judge** (its item set for task 3) and into every other service whose content loader the compose e2es
    of [m3-08](sprint-m3-08.md), [m3-11](sprint-m3-11.md) and [m3-12](sprint-m3-12.md) cross — **curriculum and practice**
    (m3-02's hand-off), and the **gateway** if it loads `internal/course` itself (m3-12 names gateway, practice and judge):
    a config field plus the loader call each, nothing else. A service that doesn't load content through `internal/course`
    yet gets no wire; write that gap into the decisions log as a hand-off to m3-08.
  - Tests: overlay ignored without `DEV_AUTH`; `fixture` present with it; a slug collision is rejected; the embedded set is
    unchanged either way.
- **Compose:** a `judge` service (build `deploy/judge.Dockerfile`, `PORT: "8087"`, `PGSEARCHPATH: judge`,
  `JWT_AUDIENCE: judge`, the `x-pg-env` / `x-jwt-env` anchors, `NATS_URL`, `IDENTITY_BASE_URL: http://identity:8081`,
  `EVALPACK_DIR: /evalpack`, `depends_on` postgres/nats/identity). Its volumes attach **m3-02's anchor** —
  `volumes: [*evalpack-mount, …]`, not a second hand-written line — plus the overlay bind
  `./internal/judge/testdata/content:/fixture-content:ro`, with `FIXTURE_CONTENT_DIR: /fixture-content` and `DEV_AUTH: "1"`
  (the value identity already uses in compose). Every other service wired above gets the same bind and the same two env vars.
  `docker compose config` stays valid. **The gateway gets no `JUDGE_BASE_URL`** ([m3-09](sprint-m3-09.md)).
  `deploy/local/initdb.sql` gains `CREATE SCHEMA IF NOT EXISTS judge;`; the e2e harness creates it too, and in-process e2es
  load the overlay directory directly.
- **`sqlc.yaml`:** a judge block (`internal/judge/store/{schema.sql,migrations,queries}` → `internal/judge/store/gen`).

### 2 · Schema [X]

`internal/judge/store/migrations/00001_init.sql` onward (a new schema, so its own goose table `judge.goose_db_version`;
all additive). Ids are **`uuidv7()`** (PG 18 built-in; compose and CI run PG 18 since m1-01, prod runs 18.4). Rules from
[t1 §4 judge](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual): **tables holding bodies
(`submission_part`, `draft`, `job`, `evaluation_feedback`) carry only PK and FK constraints** (Go validates before insert,
so a failed CHECK can never log learner code); every per-user table gets `path_slug` and an `(account_id, created_at)` index.

| Table | Shape |
|---|---|
| `submission` | `id uuid PK DEFAULT uuidv7()`, `account_id uuid NOT NULL`, `path_slug text NOT NULL`, `item_id text NOT NULL`, `context_kind text NOT NULL CHECK (context_kind IN ('course','touch','mock','arena'))`, `context_id uuid NOT NULL`, `action text NOT NULL CHECK (action IN ('submit','final','give_up'))`, `idem_key uuid NOT NULL`, `body_sha256 bytea NOT NULL`, `contract_hash text NOT NULL`, `band text`, `closes_context boolean NOT NULL DEFAULT false`, `released boolean NOT NULL DEFAULT false`, `attempt_seq integer` (evaluation-bearing rows only, INV-12), `submitted_at timestamptz NOT NULL DEFAULT now()`, `created_at timestamptz NOT NULL DEFAULT now()`. `UNIQUE (account_id, idem_key)`; partial unique `(context_kind, context_id) WHERE closes_context AND NOT released` (INV-13); partial unique `(context_kind, context_id, attempt_seq) WHERE attempt_seq IS NOT NULL`. Runs are **never** rows here (T4 §2.1) |
| `submission_part` | `(submission_id uuid REFERENCES submission ON DELETE CASCADE, part_id text, language text NOT NULL DEFAULT '', sha256 bytea, bytes integer, body bytea COMPRESSION lz4)`, PK `(submission_id, part_id)`; PK/FK only |
| `draft` | `(account_id uuid, context_kind text, context_id text, part_id text, language text, body bytea COMPRESSION lz4, path_slug text, updated_at timestamptz, created_at timestamptz)`, **all key columns `NOT NULL DEFAULT ''`** (uuid `account_id` NOT NULL), PK over the five keys, `WITH (fillfactor = 70)`, **no index on `updated_at`** (the m3-06 sweeper seq-scans); PK/FK only |
| `evaluation` | `id uuid PK DEFAULT uuidv7()`, `submission_id uuid NOT NULL REFERENCES submission ON DELETE CASCADE`, `job_id uuid`, `regrade_of uuid UNIQUE`, `reused_from uuid`, `account_id`, `path_slug`, `status text NOT NULL CHECK (status IN ('passed','failed','inconclusive','error'))`, `reason text NOT NULL` (closed Go enum, no CHECK), `class text`, `checks jsonb`, `score numeric`, `max numeric`, `hidden jsonb`, `perf text`, `first_failure jsonb`, `parts jsonb`, `trust text NOT NULL CHECK (trust IN ('checked','honor'))`, `provisional_eligible boolean NOT NULL DEFAULT false`, `review_flag boolean NOT NULL DEFAULT false`, `signals jsonb`, `per_test jsonb` (internal only), `usage jsonb`, `versions jsonb`, `completed_at timestamptz`, `created_at`. Partial unique `(submission_id) WHERE regrade_of IS NULL` — one primary terminal evaluation per submission (INV-1) |
| `evaluation_feedback` | `(evaluation_id uuid PK REFERENCES evaluation ON DELETE CASCADE, body bytea COMPRESSION lz4, created_at)`; PK/FK only (M4 fills it) |
| `job` | `id uuid PK DEFAULT uuidv7()`, `kind`, `submission_id uuid REFERENCES submission ON DELETE CASCADE` (NULL for Runs), `evaluation_id uuid`, `account_id uuid NOT NULL`, `path_slug`, `item_id`, `context_kind`, `context_id`, `seq`, `priority smallint`, `lane`, `step_cursor smallint`, `state`, `claim_token bigint NOT NULL DEFAULT 0`, `infra_retries smallint NOT NULL DEFAULT 0`, `poison_count smallint NOT NULL DEFAULT 0`, `run_after timestamptz`, `lease_until timestamptz`, `worker_id text`, `pack_digest text`, `versions jsonb`, `payload bytea`, `output bytea`, `created_at`, `finished_at`. **PK/FK only** (kind/lane/state validated in Go). Indexes: partial `(lane, priority, created_at) WHERE state = 'queued'`; partial `(account_id, lane) WHERE state = 'running'`; partial `(context_kind, context_id) WHERE state IN ('queued','running')`; `(account_id, created_at)` |
| `context_counter` | `(context_kind text, context_id uuid, last_seq integer NOT NULL DEFAULT 0, PK (context_kind, context_id))` — `INSERT … ON CONFLICT DO UPDATE SET last_seq = context_counter.last_seq + 1 RETURNING last_seq` in m3-06 |
| `arena_progress` | `(account_id uuid, item_id text, path_slug text NOT NULL, first_passed_at timestamptz, submits integer NOT NULL DEFAULT 0, source text NOT NULL CHECK (source IN ('auto','manual')), updated_at timestamptz NOT NULL DEFAULT now(), created_at …, PK (account_id, item_id))` — D10's arena done marker ([t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)); written by m3-14 |
| `outbox` | the v1 shape (`internal/review/store/migrations/00001_init.sql`) **plus `account_id uuid`** (M1a convention) |
| `inbox` | `(event_id uuid PK, received_at)` |
| `erased_account` | `(account_id uuid PK, erase_request_id uuid NOT NULL, erased_at timestamptz NOT NULL DEFAULT now())` — l-01's shape; kept forever ([t1 §6.4](../research/t1-content-data-model.md#64-retention-v20-caps-and-quotas-only-no-pruners)) |
| `event_dead_letter` | mi-05's shape: `(event_id, subject, durable, err_class, stream_seq, at)`, PK `(durable, event_id)` |

- **Bodies stay inline** (`bytea COMPRESSION lz4`) — no `object_key` column. `object_key` is a future expand migration when
  [ADR-0028 §1](../../adr/0028-object-storage-and-backups.md#1-no-object-store-in-v20)'s trigger fires (judge body bytes > 1 GiB or 25% of the
  PG volume, or a payload > 1 MiB); v2 has no object store (D11).
- `COMPRESSION lz4` needs a Postgres built with lz4 (the official 18 images and CNPG's are); a failure surfaces as a
  migration error in compose and CI, never silently.
- Not here: `usage_ledger` (M4), `evaluation_dispute` (M4), telemetry and `admin_audit` ([m3-14](sprint-m3-14.md)).
- `internal/judge/store/schema.sql` declares the infra-provisioned schema for sqlc (the v1 pattern); queries for this
  sprint: tombstone insert/check, the erase deletes, outbox insert/list/mark-sent, dead-letter insert/list. No inbox query
  yet: the erase path is idempotent without one (tombstone `DO NOTHING`, deterministic ack id — l-01); `inbox` is for later consumers.
  `sqlc generate`; commit the generated code.

### 3 · Evalpack loader + `GET /internal/evaluable` [X]

`internal/judge/pack` ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack),
[§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning), [ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships)).
**Build on m3-02's `internal/packspec`, one definition:** parse with `packspec.ReadManifest` and verify with
`packspec.VerifyFiles`; the statuses, the index and `OpenCases` stay judge's. If `VerifyFiles` doesn't stream (or reports
only a pack-level error instead of per-item results), add streaming and per-item results **there** (with its tests), so
packlint and judge keep one reader — never a fork under `internal/judge`.

- **At start, never fatal.** `Load(dir, items)` returns an `*Index` and never an error that stops the process; panics are
  recovered into a pack-level status. Pack-level status: `ok` · `missing` (`EVALPACK_DIR` unset, empty or absent) ·
  `unsupported_format` (`format_major` ≠ the one `packspec` defines — `0` is mi-07's scaffold and is never loaded) ·
  `invalid_manifest` (unparseable `/manifest.json`). Anything but `ok` ⇒ **Ready with 0 evaluable**, logged once at ERROR
  (a crash-looping judge would wedge `apps`, which Flux applies with `wait: true`).
- **Verify every file** listed in `manifest.items[id].files{path: sha256}` (via `VerifyFiles`) by **streaming** sha256 (never holding a file
  in memory — judge's limit is 256 Mi; the pack may approach 150 MB). A missing file or a mismatch marks that **item**
  `invalid`, not the pack.
- **Per-item status** against judge's public content (the embedded `all:courses` loader, `internal/course`, plus the dev
  overlay of task 1 when it is honoured):
  `ok` (live public item; its `canon.ContractHash` ∈ `accepts_contract_hashes`; every grader kind in the build's supported
  set `{code, key}`) · `spec_mismatch` (contract hash not accepted → self path with the "grading pending" badge) ·
  `unsupported` (a grader kind this build lacks, e.g. `ai_rubric` before M4, or a harness not in the closed registry) ·
  `invalid` (hash failure, a pack item with no live public item, or a manifest `course` that differs from the public
  item's course) · `no_pack` (a public item absent from the pack). So with the default compose mount and **no** overlay,
  every `fx-*` pack item is `invalid`, every DSA item `no_pack`, and judge is Ready with 0 evaluable — by design.
- **In memory:** keys, anchors and exemplars (small); **lazily:** `cases.jsonl.zst` — expose `OpenCases(itemID)`
  returning a streaming reader over `packspec`'s `cases.jsonl.zst` reader (`github.com/klauspost/compress` is already a
  direct dependency through m3-02); m3-06 decodes lines with `packspec`'s case type.
- **Pack digest** = sha256 of the raw `/manifest.json` bytes (the manifest pins every file hash, so it identifies the
  whole pack; judge cannot see the OCI digest from inside the volume). Logged with `version`, `format_major` and
  `validated_against` at load; recorded on every evaluation by m3-06. Record this in the ADR (task 6).
- **`GET /internal/evaluable?path=<slug>`** (ClusterIP-only, no user JWT — the ADR-0016 internal model behind MI-5a;
  practice pins `grading_mode` and `contract_hash` from it at attempt start, [m3-08](sprint-m3-08.md)) answers from memory:
  `{admission, pack: {status, version, digest, format_major, validated_against}, items: [{item_id, status, contract_hash,
  grader_kinds, run_available, submit_available}]}`. In this sprint `run_available`/`submit_available` are always `false`
  (m3-06 wires the runner and graders); the shape is final. Put the wire types in **`internal/platform/judgeapi`**
  (`Evaluable`, `EvaluableItem`) so practice and the gateway import them without importing judge.
- **No handler serves `EVALPACK_DIR`** (T1 §3.3 step 4), two tests: (a) a sentinel string planted in a pack file (in a
  `t.TempDir()` copy of the fixture pack) must not appear in any response from any route (enumerate the mux registrations; include traversal attempts such as
  `/internal/evaluable?path=../../evalpack`); (b) a `go/ast` test fails if non-test code under `internal/judge/` outside
  `pack/` calls `http.FileServer`, `http.ServeFile`, `http.ServeContent`, `http.Dir` or `os.DirFS`.
- **Tests** (`internal/judge/pack/*_test.go`, loading the fixture content through the overlay helper): good pack (3 `ok`),
  a file hash mismatch (that item `invalid`, others `ok`), `format_major: 0` (0 evaluable), unparseable manifest (0),
  missing dir (0), a contract-hash mismatch (`spec_mismatch`), an `ai_rubric` item (`unsupported`), a pack item with no
  public item (`invalid`), and the fixture pack **without** the overlay (0 evaluable). **Variants are generated at test
  time in `t.TempDir()`** from the committed `pack/` (copy, then corrupt a file, rewrite the manifest or add an item) —
  never committed: m3-01's repo-wide leak lint fails CI on `cases.jsonl*` / `*.jsonl.zst` anywhere except exactly
  `internal/judge/testdata/{pack,packsrc}`.

### 4 · Events + erase [X]

- **`topology.go`** ([ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes);
  mi-05 declared `XLEARN_JUDGE` budget-only): `XLEARN_JUDGE` (owner judge, `xlearn.judge.*`, 512 MiB, MaxAge 14 d,
  `Discard=Old`) now **Emits `xlearn.judge.account_erased`**; add the durable **judge on `XLEARN_IDENTITY`** (`judge-erase`,
  l-01's convention; filter `xlearn.identity.account_erasure_requested`; Handles it) and identity's
  **ack durable on `XLEARN_JUDGE`** (filter `xlearn.judge.account_erased`). l-01 made identity's ack durables and its
  expected-ack set data-driven from `topology.go`; if it hard-coded them, make the **minimal** identity change here and say
  so in the PR. Identity then expects **5** acks (practice, review, assessment, coach, judge).
- **Golden re-render:** `go test ./internal/platform/events -run Golden -update`; commit both goldens (`.conf`, `.yaml`).
  The budget and subject-registry tests stay green. The **infra ACL PR** that pastes this render is
  [m3-07](sprint-m3-07.md)'s (merged before v1.13.0).
- **Publisher + relay:** `NewNatsPublisher` on `XLEARN_JUDGE` (the stream is created by judge's `ensureStream` on first
  start), the outbox relay at the platform default interval (m3-06 adds `RELAY_INTERVAL`, default 1 s). No `NATS_URL` → the
  log publisher (local only; fail closed when set, mi-05).
- **Erase consumer** ([ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20),
  [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo)) — **l-01's shared
  contract, not a hand-rolled copy.** Durable `judge-erase` (l-01's `<svc>-erase` convention) on `XLEARN_IDENTITY`,
  **`DeliverAll`** (on its first start it replays every past erasure request and acks each — tombstones only), subscribed
  with `events.WithDeadLetter(<judge store>)`:
  - **Wire** `erase.NewHandler("judge", erase.NewHTTPVerifier(IDENTITY_BASE_URL, httpClient), store, log)`. The handler
    decodes l-01's payload (`erase_request_id`, `account_id`; reuse its fixture), re-verifies with identity
    `GET /internal/erasures/{id}` **before any tx** (no HTTP inside a tx) and maps outcomes exactly as l-01's table:
    undecodable → `events.Terminal("decode")`; identity 404 / `account_gone=false` / account mismatch (**forged or stale**)
    → **nothing deleted**, `events.Terminal("erase_unverified")` → a dead-letter row (`err_class=erase_unverified`) +
    `Term()` on this delivery, never retried; identity unreachable, timeout, 5xx or a non-identity 404 → a plain error →
    nak with backoff (the last delivery → mi-05's hook); verified → `EraseTx` → ack; `EraseTx` fails → nak.
  - **Implement judge's `erase.Store.EraseTx(ctx, r, ackEventID)`**, one tx in schema `judge`:
    1. `INSERT INTO judge.erased_account (account_id, erase_request_id, erased_at) … ON CONFLICT (account_id) DO NOTHING`;
    2. collect the account's `(context_kind, context_id)` pairs from `submission`, then delete `context_counter` for those
       contexts, `job`, `submission` (cascading parts, evaluations, feedback), `draft`, `arena_progress` and **the account's
       `outbox` rows**, all `WHERE account_id = $1`;
    3. insert the ack outbox row `xlearn.judge.account_erased` (l-01's payload) with
       **`event_id = erase.AckEventID("judge", r.EraseRequestID)`** `ON CONFLICT (event_id) DO NOTHING` and
       **`outbox.account_id = NULL`** — otherwise a redelivered erase would delete its own unsent ack in step 2.
  - Idempotent by construction (tombstone `DO NOTHING`, deterministic ack id): no inbox claim is needed.
- **Dead letters:** the subscription's `WithDeadLetter(store)` sink (mi-05 hook) writes `judge.event_dead_letter`, ids only;
  the handler never writes the row itself.
- **Tombstone helper** `store.IsErased(ctx, account)` for m3-06's worker and m3-14's handlers.
- **Tests:** real-PG store tests for `EraseTx` (every table emptied for A, B untouched, the ack row written with
  `account_id IS NULL` and the deterministic id, a second run is a no-op with exactly one ack row); l-01's **coverage test**
  for judge (every `judge` table with an `account_id` column is in the erase list, the cascade allow-list — `evaluation`, which
  cascades from `submission` — or the exempt list — `erased_account`); handler tests with a fake verifier for every outcome row,
  asserting `errors.As(err, *events.TerminalError)` and its `Class` for forged/stale and decode, and a plain error for
  identity-down; an e2e (`-tags e2e`, embedded NATS + real PG + a **fake identity** `/internal/erasures/{id}`) publishing
  l-01's erasure fixture on `XLEARN_IDENTITY` → judge rows gone, `xlearn.judge.account_erased` published once, a replay is
  a no-op, and a forged event (the fake answers 404) leaves every row, writes the dead-letter row and is `Term`ed; the
  subject-registry test covering both new durables. The fake identity closes nothing — identity's close after the 5th ack
  is task 6's compose check on the real path.

### 5 · AuthN, IDOR scaffolding, kill switch [X]

- **`aud=judge`:** `auth.NewJWKSVerifier(JWKS_URL, "judge", JWT_ISSUER)` and a `requireJWT` middleware (the v1 pattern in
  `internal/review/service.go`); `accountFrom(ctx)` returns `claims.Subject`. In prod the JWKS fetch goes to
  `xlearn-gateway:8080` (MI-5a's same-namespace source, judge's egress in m3-07). Tests: wrong `aud` → 401, expired → 401.
- **IDOR scaffolding** (every read by id scoped to `sub`, [t4 §10](../research/t4-judge-contract.md#10-security-and-integrity-notes)):
  (a) a query-lint test over `internal/judge/store/queries/*.sql`: every query touching a per-user table (`submission`,
  `submission_part`, `draft`, `evaluation`, `evaluation_feedback`, `job`, `arena_progress`) either filters
  `account_id = $n` or carries a `-- scope: internal` comment (workers, erase, internal endpoints); (b) a reusable test
  helper `idorCase(t, route, owner, other)` — the other account's token on the owner's resource id must get **404** (never
  403: no existence oracle). m3-14 adds one case per learner route.
- **Kill switch `JUDGE_ADMISSION`** (L15, [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory);
  a **permanent** T-2 kill switch, [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)):
  unset or `open` → open; `closed` → closed; **any other value → closed**, logged at ERROR (fail-safe both ways, like
  `SIGNUP_MODE`). Closed means: new-work routes answer **503** `{"error":"evaluation_unavailable","fallback":"self"}`
  (T4 §2.3 row 1) through a `requireAdmission` middleware m3-14 applies to its routes; `/internal/evaluable` reports
  `admission: "closed"` with every `submit_available=false`; queued jobs still drain (m3-06). Table-test the parser and the
  middleware.

### 6 · Verify, docs, ADR, hand-off [X]

- **Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`,
  the migration lint, `make nats-acl-test`, web tests (unchanged), `docker compose config`. Compose (`docker compose up --build`):
  - judge Ready; `curl localhost:8087/internal/evaluable?path=fixture` (via `docker compose exec`) shows **fx-001..fx-003
    `ok`**; `?path=dsa` shows every DSA item `no_pack` (the fixture pack has none);
  - **without the overlay** (`FIXTURE_CONTENT_DIR` unset on judge) → Ready, 0 evaluable (every `fx-*` pack item `invalid`);
  - a broken pack from a temp dir, never committed (`d=$(mktemp -d); cp -R internal/judge/testdata/pack/. "$d"`, then corrupt
    `$d/manifest.json`), `EVALPACK_DIR=$d docker compose up -d judge` → still Ready, 0 evaluable; the same with an empty
    dir (no pack) and with `format_major: 0` written into the copy's manifest;
  - **erase end to end on the real path** (needs [l-02](sprint-l-02.md); see the informational gate): take a seeded
    non-owner test account, create judge rows for it (a store-level seed script or the e2e helper — no learner route exists
    yet), run `docker compose exec identity identity admin account erase <user>`, then check judge's rows for the account are gone,
    one `xlearn.judge.account_erased` was published, and `identity admin erasures` shows the request **closed with 5 acks**.
- **Docs:** `docs/architecture/services.md` (judge: port 8087, schema `judge`, role `xlearn_judge`, pack volume, callers),
  `data-model.md` (judge schema), `deploy/local/README.md` (the dev content overlay beside m3-02's "Eval pack in compose"), `events.md` (`XLEARN_JUDGE` live; judge's erase durable; the `account_erased` ack; 5
  expected acks), `api.md` (`/internal/evaluable`).
- **ADR** "judge service realization (M3-1)": Ready-with-zero rules and the item statuses, pack digest = sha256(manifest),
  the loader on `internal/packspec`, the dev content overlay (`FIXTURE_CONTENT_DIR` + `DEV_AUTH`), `JUDGE_ADMISSION`
  semantics, the erase durable on l-01's `internal/platform/erase`, `internal/platform/judgeapi`. Take the **next free number** after
  checking peers' PRs and worktrees.
- **Hand-off to [m3-07](sprint-m3-07.md)** (write it into the PR description and the status.md decisions log):
  the env table above; secrets `xlearn-judge-db` and the NATS seed; **judge's default-deny egress must also admit
  `xlearn-identity:8081`** (the erase re-verify) in addition to DNS, PG, NATS, the runner and `xlearn-gateway:8080` —
  without it every erasure naks with backoff until its last delivery dead-letters it. [m3-07](sprint-m3-07.md) already
  carries identity :8081 ("born here") and sets `IDENTITY_BASE_URL`; confirm both in the hand-off. The dev content
  overlay is **compose-only**: m3-07 sets neither `FIXTURE_CONTENT_DIR` nor `DEV_AUTH`.

## Acceptance criteria

- [ ] judge is **Ready in compose** with the fixture pack and the dev overlay (fx-001..fx-003 `ok` in `/internal/evaluable?path=fixture`), and with a broken pack (0 evaluable, **no crash**), with no pack, and with `format_major: 0`.
- [ ] **Without the overlay**, judge is Ready with 0 evaluable; the overlay is ignored (ERROR log) when `DEV_AUTH` is unset.
- [ ] The judge-side e2e (fake identity): an erasure **removes every judge row** for the account, writes the tombstone, publishes `xlearn.judge.account_erased` once, and a replay is a no-op; a **forged or stale** event deletes nothing, writes a dead-letter row (`err_class=erase_unverified`) and is `Term`ed; the coverage test is green.
- [ ] Compose end to end on the real path (`identity admin account erase`, l-02): judge's rows are gone and `identity admin erasures` shows the request closed with **5 acks** — or, if l-02 hasn't merged, the check is handed to m3-07 in the decisions log.
- [ ] With `NATS_URL` set and `IDENTITY_BASE_URL` unset, judge refuses to start (l-01's rule).
- [ ] No handler serves `EVALPACK_DIR` (sentinel + AST tests green).
- [ ] `aud=judge` enforced; the IDOR query-lint test green; `JUDGE_ADMISSION` parses fail-safe and `closed` yields the typed 503.
- [ ] `deploy.yml` has 8 image jobs, each `needs: release-line`; `deploy/version_test.go` covers `judge.Dockerfile`.
- [ ] `topology.go` budget, registry and golden tests green with the new durables; `sqlc diff` clean; every v1 test and e2e green.

## Release

**Merge only — ships dark in v1.13.0**, the tag [m3-07](sprint-m3-07.md) cuts after [m3-06](sprint-m3-06.md) and
[m3-14](sprint-m3-14.md) merge, with its ACL PR merged **before** the tag and the MI-13 judge HelmRelease **after** it (image
before policy, [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)).
From this merge, **any** `v*` tag builds `xlearn-judge` too; that is harmless before m3-07 (no ImageRepository or
HelmRelease exists yet).

## Definition of Done

CI green (`go`, `e2e`, `nats-acl`, `sqlc diff`, the migration lint) · compose shows judge Ready with and without a valid
pack, and Ready with 0 evaluable without the dev overlay · squash-merged to `main`, **no tag** · docs + ADR written ·
statuses updated (this file + [`../status.md`](../status.md): Sprint board, M3 🔄, flag inventory `JUDGE_ADMISSION`, decisions
log incl. the m3-07 egress hand-off and the dev content overlay).

## Risks / watch-outs

- **A crash-looping judge wedges `apps`** (`wait: true`): Ready-with-zero is mandatory — recover panics in the loader,
  never `log.Fatal` on pack problems, keep `/readyz` off the pack and off NATS.
- **Missing identity egress on prod** turns every erasure into a dead letter (fail closed). [m3-07](sprint-m3-07.md)
  carries identity :8081 and `IDENTITY_BASE_URL`; confirm it merged with both before v1.13.0 deploys judge.
- **The dev overlay reaching prod** would publish synthetic items: it is honoured only with `DEV_AUTH` (never set in prod),
  ignored with an ERROR otherwise, and it can only add a course, never shadow a real one.
- **Identity expects judge's ack from v1.13.0**, but judge's pod only starts after the MI-13 infra PR. Erasures in that
  window stay open until judge starts and replays (DeliverAll). D34: nothing alerts — m3-07's verify reads identity's open
  requests. identity's `Subscribe` on `XLEARN_JUDGE` must tolerate the stream not existing yet (judge creates it).
- **Startup cost and memory:** hashing a large pack streams at start (≈ 1 s per 150 MB); never buffer files; cases stay lazy.
- **Public repo:** only m3-02's **synthetic** fixture pack lives here — never copy anything from `../xlearn-evalpack`
  (AGENT.md rule from m3-01).
- **Schema churn:** m3-06 and m3-14 add columns and indexes on top; keep every later migration expand-only and take the
  next free goose version at rebase.
- **Shared files** (`topology.go`, the goldens, `deploy.yml`, `sqlc.yaml`, `docker-compose.yml`, `internal/course`'s loader,
  curriculum/practice config): rebase late and re-render.
