# Prompt — Sprint m3-05 · judge service: skeleton, schema, evalpack loader, erase consumer (M3-1)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-05.md`](../sprints/sprint-m3-05.md)   ·   **Milestone:** M3 (M3-1, judge dark)   ·   **Prereqs:** [m3-02](../sprints/sprint-m3-02.md) (`fixture` course + pack, `internal/packspec`, compose anchor) · [l-01](../sprints/sprint-l-01.md) (`internal/platform/erase`, `events.Terminal`) · for the compose erase check, [l-02](../sprints/sprint-l-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m3-05.md`](../sprints/sprint-m3-05.md) — the env table, the dev content overlay, the schema
  table, the pack statuses, the erase wiring and the m3-07 hand-off live there. Follow them exactly.
- m3-02's plan [task 7](../sprints/sprint-m3-02.md#7--synthetic-fixture-pack--pack-fixture-ci-job--compose-anchor-x) (the
  `fixture` course, the `x-evalpack-mount` anchor, **the dev-overlay hand-off to this sprint**) and its `internal/packspec`
  table (`ReadManifest` + `VerifyFiles` are the reader this loader imports).
- Pack and hashes: [t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack) (image layout, judge-at-start rules,
  "no handler serves `EVALPACK_DIR`"), [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning)
  (`contract_hash`, mismatch rules), [t1 §3.7](../research/t1-content-data-model.md#37-local-dev-and-public-ci) (compose mount),
  [ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships) and
  [§3](../../adr/0027-content-evalpack-and-user-data-model.md#3-storage-placement-no-new-database-engine).
- Schema: [t1 §4 judge](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual),
  [t4 §2.4–§2.6](../research/t4-judge-contract.md#2-the-common-contract) (closes, job, evaluation fields),
  [t4 §11.4 #11](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag),
  [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena) (`arena_progress`); **t4 §13's owner decisions override
  the body** (D16: no reimpl child context).
- Erase: [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20),
  [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo), and **l-01's
  [task 2](../sprints/sprint-l-01.md#2--service-erase-consumers-x)** — the `internal/platform/erase` API
  (`NewHandler`, `NewHTTPVerifier`, `AckEventID`, `Store.EraseTx`), the handler outcome table, `events.Terminal`, the
  `EraseTx` steps (tombstone with `erase_request_id`, deletes, ack row with `account_id = NULL`), the config rule and the
  coverage test. Judge **uses** that package; it does not re-implement it.
- Events and limits: [ADR-0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes)
  (topology, golden, registry, dead letters, ACL standing rule), [§4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)
  (L15 kill switch, L21 pool 8), [ADR-0034 §1.5](../../adr/0034-v2-release-labelling-gating-and-rollback.md#15-other-release-streams)
  (the 8th `deploy.yml` job; image before policy), [§2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
  (permanent kill switches), [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)
  (expand rules), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md), [ADR-0006](../../adr/0006-authn-authz.md),
  [ADR-0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (internal endpoints, no HTTP in a tx),
  [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md).
- Rollout: [§4 M3](../rollout-plan.md#4-per-milestone-detail) ("judge is born with its `XLEARN_IDENTITY` erase consumer"),
  [§2 MI-13](../rollout-plan.md#2-mi-infra-track), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag).
- Code: `cmd/review/main.go`, `internal/review/{config,service,consumers}.go`, `internal/review/store/{migrate.go,schema.sql,migrations,queries}`
  (the newest v1 service template), the l-01 erase consumers, `internal/platform/{auth,events,health,httpx,config}`,
  `internal/course` + `internal/course/canon`, the `curriculum` embed package, `internal/packspec` (m3-02),
  `internal/judge/testdata/{content,pack}` (m3-02), `internal/platform/erase` + `internal/platform/events/consumer.go` (l-01),
  curriculum's and practice's config and content loading (for the overlay wire),
  `deploy/review.Dockerfile`, `deploy/version_test.go`, `.github/workflows/{deploy,ci}.yml`, `docker-compose.yml`,
  `deploy/local/initdb.sql`, `sqlc.yaml`, `internal/e2e/`. Infra (read-only, for the hand-off): `../infra/apps/xlearn-review.yaml`.

## Context

judge is the 8th service: a course-blind evidence engine that owns submissions, drafts, evaluations, the private eval pack
and (next sprint) the Postgres job queue. M3-1 ships it **dark** in v1.13.0: nothing reaches it until m3-13 sets the
gateway's `JUDGE_BASE_URL`. This sprint builds the part that must be right before anything else: a schema, a pack loader
that turns every pack problem into **"Ready with zero evaluable items"** (a crash-loop would wedge Flux's `apps` apply,
which waits), `GET /internal/evaluable`, `aud=judge` verification with every read scoped to the subject, a permanent kill
switch, and the **`XLEARN_IDENTITY` erase consumer from birth** (the rollout's "cohort and erase" rule; L-E shipped the
pattern in v1.11.0). Production wiring — ACL, the ADR-0005 4-step, the HelmRelease with the pack image volume — is m3-07's.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m3-02 merged: the **`fixture` course (fx-001..fx-003)** — content under `internal/judge/testdata/content`, the built
      pack at `internal/judge/testdata/pack` (a real `format_major`, per-file sha256 in `/manifest.json`); `internal/packspec`
      (`ReadManifest`, `VerifyFiles`, the cases reader); the compose `x-evalpack-mount` anchor.
- [ ] l-01 merged and v1.11.0 live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz` ≥ 1.11.0): identity serves
      `GET /internal/erasures/{id}`; `internal/platform/erase` and `events.Terminal` are on `main`.
- [ ] On `main`: mi-05's `topology.go` + ACL renderer + goldens + dead-letter hook + `events.Dial` + the `PG_MAX_CONNS` helper;
      m1-02's v2 envelope; `internal/course` with the embedded `all:courses` loader; m3-01's `internal/course/canon.ContractHash`.
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no peer creating `cmd/judge` / `internal/judge/`
      (beyond `testdata/`) or editing `topology.go`, `deploy.yml`, `sqlc.yaml` or `internal/course`'s loader right now.
- _Informational:_ l-02 merged (v1.12.0: `identity admin account erase`, `identity admin erasures`) for step 7's compose erase
  check. If not, run the judge-side e2e only and hand the compose check to m3-07 (decisions log).

## Do this (in order)

1. **[X] Branch** `feat/m3-05-judge-skeleton` from an up-to-date `main`.
2. **[X] Skeleton** (plan task 1): `cmd/judge/main.go` (version flag, migrate-then-serve, pool `MaxConns` 8 via the mi-05
   helper, shutdown order consumer → relay → HTTP → pool), `internal/judge/config.go` with the plan's env table,
   `service.go`/`handlers.go` with `/healthz`, `/readyz` (DB ping only). `deploy/judge.Dockerfile` (port 8087,
   `-X main.version`), the 8th `judge` job in `deploy.yml` (`needs: release-line`), the compose `judge` service attaching
   **m3-02's `*evalpack-mount` anchor** (not a hand-written volume line) with `IDENTITY_BASE_URL` (no gateway
   `JUDGE_BASE_URL`), `CREATE SCHEMA IF NOT EXISTS judge;` in `deploy/local/initdb.sql` and the e2e harness, the `sqlc.yaml`
   block. Config refuses to start when `NATS_URL` is set without `IDENTITY_BASE_URL` (l-01's rule).
   **Dev content overlay** (m3-02's hand-off): an `internal/course` helper that adds `FIXTURE_CONTENT_DIR`'s
   `courses/<slug>/` to the loaded set (add-only; a slug collision is an error), honoured **only with `DEV_AUTH`** (else
   ignored + ERROR); wire it into judge, curriculum and practice — and the gateway if it loads `internal/course` itself —
   (config field + loader call each); compose binds `./internal/judge/testdata/content:/fixture-content:ro` with
   `FIXTURE_CONTENT_DIR: /fixture-content` and `DEV_AUTH: "1"` on each wired service. Tests for ignored-without-`DEV_AUTH`, present-with-it and the collision.
3. **[X] Schema** (task 2): `00001_init.sql` (+ separate files if any index needs `CONCURRENTLY`) with every table in the plan —
   `submission` (idempotency unique, the unreleased-close partial unique, the seq partial unique), `submission_part`,
   `draft` (keys `NOT NULL DEFAULT ''`, fillfactor 70), `evaluation` (one primary per submission), `evaluation_feedback`,
   `job` (PK/FK only + the three partial indexes), `context_counter`, **`arena_progress`**, `outbox` (+`account_id`),
   `inbox`, `erased_account` (**with `erase_request_id`**, l-01's shape), `event_dead_letter`. Body tables PK/FK only, bodies
   inline `bytea COMPRESSION lz4` with **no `object_key`** (deferred until ADR-0028's trigger, D11); `path_slug` and
   `(account_id, created_at)` on every per-user table. Queries for this sprint's needs; `sqlc generate`.
4. **[X] Pack loader** (task 3): `internal/judge/pack` on **`packspec.ReadManifest` + `packspec.VerifyFiles`** (streaming;
   if `VerifyFiles` isn't streaming or per-item, extend it in `internal/packspec` — never fork it), pack-level and per-item
   statuses, in-memory keys, lazy `OpenCases` over `packspec`'s cases reader, pack digest = sha256(manifest bytes), recover
   panics; `internal/platform/judgeapi` wire types; `GET /internal/evaluable?path=`; the sentinel and AST "no handler serves
   `EVALPACK_DIR`" tests; the variant table tests, **generating every variant at test time in `t.TempDir()`** from the
   committed `pack/` (copy, then corrupt / rewrite the manifest / add an item). Commit no new pack artefacts: m3-01's leak
   lint allows `cases.jsonl*` / `*.jsonl.zst` only under `internal/judge/testdata/{pack,packsrc}`.
5. **[X] Events + erase** (task 4): `topology.go` — `XLEARN_JUDGE` emits `xlearn.judge.account_erased`; judge's erase durable
   `judge-erase` on `XLEARN_IDENTITY`; identity's ack durable on `XLEARN_JUDGE` (data-driven from l-01; minimal identity
   change only if l-01 hard-coded it); re-render the goldens (`-update`). Publisher + relay. The erase consumer is
   **`erase.NewHandler("judge", erase.NewHTTPVerifier(IDENTITY_BASE_URL, …), store, log)`** (DeliverAll,
   `WithDeadLetter`) over judge's **`EraseTx`**: tombstone with `erase_request_id` `DO NOTHING`; delete `context_counter`
   (contexts found via `submission`), `job`, `submission` (cascades), `draft`, `arena_progress` and the account's `outbox`
   rows; the ack row with `event_id = erase.AckEventID("judge", id)` and `account_id = NULL`. Forged or stale → nothing
   deleted, dead-letter row `erase_unverified` + `Term()` (l-01's table — never log-and-ack). `store.IsErased`. Tests:
   real-PG `EraseTx` (+ replay, one ack row), l-01's coverage test for judge, handler cases per outcome row, the e2e on
   embedded NATS with a fake identity (incl. a forged event).
6. **[X] AuthN + IDOR + kill switch** (task 5): the `aud=judge` verifier and `requireJWT`; the query-lint test and the
   `idorCase` helper; `JUDGE_ADMISSION` (unset/`open` → open, `closed` → closed, anything else → closed + ERROR) with
   `requireAdmission` returning 503 `evaluation_unavailable{fallback: self}` and `/internal/evaluable` reflecting it.
7. **[X] Verify** (task 6): `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG tests via `XLEARN_TEST_DATABASE_URL`),
   `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, the migration lint, `make nats-acl-test`, web tests. Compose:
   judge Ready with `curl …/internal/evaluable?path=fixture` showing **fx-001..fx-003 `ok`**; without the overlay → Ready,
   0 evaluable; a broken pack copied into a `mktemp -d` dir and mounted via `EVALPACK_DIR=$d` → Ready, 0 evaluable; no
   pack → the same. **Erase on the real path** (l-02): seed judge rows for a non-owner test account, run
   `docker compose exec identity identity admin account erase <user>`, check judge's rows are gone and
   `identity admin erasures` shows the request closed with **5 acks**.
8. **[X] Docs + ADR:** `docs/architecture/{services,data-model,events,api}.md`; ADR "judge service realization (M3-1)" —
   check peers' PRs and worktrees, then take the next free number (MADR, append-only).
9. **[X] Hand-off:** put the m3-07 hand-off (env table, secrets, **egress to `xlearn-identity:8081`** besides DNS, PG, NATS,
   runner and `xlearn-gateway:8080`) in the PR description and the status.md decisions log.
10. **[X] Ship** per AGENT.md land-and-sync with this sprint's release action (below).

## Constraints

- **Service boundaries (ADR-0005):** judge writes only schema `judge`; identity is read through `GET /internal/erasures/{id}`,
  never its tables. judge never computes grades and never writes to review or assessment (INV-2). The only non-judge
  changes allowed are `topology.go` + goldens (and, only if l-01 hard-coded it, identity's ack durable list), the dev
  overlay helper in `internal/course`, and its wire in curriculum's and practice's (and, if it loads `internal/course`, the
  gateway's) config — a field + the loader call.
- **goose + sqlc:** embedded migrations under the advisory lock; `judge.goose_db_version`; commit generated code;
  `sqlc diff` clean. New schema = expand only; never run `Down` in production.
- **Outbox / erase:** `EraseTx` writes the tombstone, the deletes and the ack row in one transaction (idempotent without an
  inbox claim); re-verification happens **before** `pool.Begin` (inside l-01's handler); transient errors nak with backoff;
  forged/stale events are `events.Terminal` (dead-letter + `Term()`); stop the consumer before `srv.Shutdown()`.
- **Dev overlay is compose-only:** honoured only with `DEV_AUTH`; add-only; m3-07 sets neither variable in prod.
- **Ready with zero, never a crash-loop:** no pack problem may stop the process; `/readyz` never checks the pack or NATS.
- **No learner route, no gateway change:** the learner API is m3-14's, the BFF m3-09's; the gateway keeps no `JUDGE_BASE_URL`.
- **Public repo hygiene:** synthetic fixtures only; never copy from `../xlearn-evalpack`; no new committed pack artefacts
  (test variants live in temp dirs).
- **GitOps:** no infra PR in this sprint (m3-07 owns the ACL, the 4-step and the HelmRelease); never `kubectl apply`.
  The ACL standing rule still applies — m3-07 merges this sprint's re-rendered golden **before** v1.13.0.
- **D34:** no alerting; log at ERROR where the design says "alert".
- **Memory-sum rule (ADR-0035 §5):** judge's pod (requests 50m/128 Mi, limits 500m/256 Mi) is budgeted in the rule and
  lands in m3-07; keep the loader streaming so 256 Mi holds.
- **Parallel sessions:** check peers' PRs, tags and worktrees before claiming the ADR number and before merging shared files.

## Deliverables

- `cmd/judge/main.go`; `internal/judge/{config,service,handlers}.go`; `internal/judge/store/` (migrations, queries, gen,
  migrate, schema.sql); `internal/judge/pack/`; `internal/platform/judgeapi/`.
- The dev overlay helper in `internal/course` + its wire in judge and the other services the e2es cross; `deploy/local/README.md` note.
- `deploy/judge.Dockerfile`; the 8th `deploy.yml` job; compose `judge` (anchor + overlay) and the overlay env on the other
  wired services; `initdb.sql`; `sqlc.yaml`.
- `topology.go` entries + re-rendered goldens; the erase consumer + dead letters; tests (unit, real-PG, e2e).
- Docs + the ADR; the m3-07 hand-off in the PR description.

## Update status

- [`../sprints/sprint-m3-05.md`](../sprints/sprint-m3-05.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M3** milestone row → 🔄 ("M3-1 judge skeleton merged, ships
  dark in v1.13.0"); the **flag inventory** gains `JUDGE_ADMISSION` as a **permanent kill switch** (L15; R-a path = an infra
  PR on `apps/xlearn-judge.yaml`; owner milestone M3; never removed); **Decisions log** lines for: pack digest =
  sha256(manifest), the Ready-with-zero statuses, the loader on `internal/packspec`, **the dev content overlay**
  (`FIXTURE_CONTENT_DIR` + `DEV_AUTH`, which services are wired — or the gap handed to m3-08),
  `JUDGE_ADMISSION`'s fail-safe parse, the erase durable `judge-erase` on `internal/platform/erase` and the expected-ack
  count 4 → 5, the `internal/platform/judgeapi` package, **the m3-07 egress hand-off (identity :8081)**, the compose erase
  check (done, or handed to m3-07 if l-02 wasn't merged), and the ADR number.

## Done when (acceptance)

- [ ] judge is Ready in compose with the fixture pack + overlay (fx-001..fx-003 `ok` under `?path=fixture`), a broken pack, no pack and `format_major: 0` — never a crash; without the overlay, Ready with 0 evaluable.
- [ ] Judge-side e2e: an erasure removes every judge row for the account, writes the tombstone, publishes `xlearn.judge.account_erased` once; replays are no-ops; forged/stale events delete nothing, dead-letter (`erase_unverified`) and are `Term`ed. Compose on the real path: `identity admin erasures` shows the request closed with 5 acks (or handed to m3-07 if l-02 isn't merged).
- [ ] `NATS_URL` set without `IDENTITY_BASE_URL` → judge refuses to start.
- [ ] No handler serves `EVALPACK_DIR` (sentinel + AST tests).
- [ ] `aud=judge` enforced; IDOR query-lint green; `JUDGE_ADMISSION` fail-safe with the typed 503.
- [ ] 8 image jobs in `deploy.yml`, each `needs: release-line`; the version test covers judge.
- [ ] Topology budget, registry and golden tests green; `sqlc diff` clean; every v1 test and e2e green.

**Shipping:** per AGENT.md land-and-sync with **this sprint's release action — merge only (ships dark in v1.13.0)**:
branch → conventional commits with the attribution lines → push → PR → CI green → squash-merge → `git checkout main && git pull`.
**Do not tag** (m3-07 cuts v1.13.0 after m3-06 and m3-14) and open **no infra PR** (m3-07 owns them); nothing deploys yet.
