# Prompt — Sprint mi-05 · N0: NATS topology, dead letters, identity on NATS, client options, pool pins (MI-6)

> **One self-contained prompt = one sprint = one session.** Paste it into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-mi-05.md`](../sprints/sprint-mi-05.md)   ·   **Milestone:** MI (rollout step MI-6, NATS stage N0)   ·   **Prereqs:** [m1-01](../sprints/sprint-m1-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, the land-and-sync directive, service boundaries.
- [`../sprints/sprint-mi-05.md`](../sprints/sprint-mi-05.md): the plan. Its task sections hold the tables and code shapes you implement.
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md):
  - **§1**: the five code changes and the v2 topology table;
  - **§2**: the NATS auth model, the **ACL table**, "Verification before any seed is mounted", the N0–N4 rollout, the standing rule;
  - **§4**: L20 (NATS limits) and L21 (pool pins).
- [ADR-0014](../../adr/0014-nats-jetstream-topology-and-outbox-relay.md) (today's topology and relay), [ADR-0004](../../adr/0004-inter-service-comms-and-events.md) (outbox and idempotent consumers), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (schema-per-service, goose), [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3 (expand rules, consumers before producers).
- [`../research/t7-cross-cutting-and-rollout.md`](../research/t7-cross-cutting-and-rollout.md) §1.1–§1.5: findings with file:line evidence, N0, topology, auth rollout.
- [`../rollout-plan.md`](../rollout-plan.md) §2 (the MI-6 and MI-7 rows), §2.2 (operating rules), §12 (the xlearn M1 bullets).
- [`../../architecture/events.md`](../../architecture/events.md) (subjects and flows) and [`../../architecture/data-model.md`](../../architecture/data-model.md).
- **Code:**
  - `internal/platform/events/{nats.go,consumer.go,relay.go,events.go,consumer_test.go}`;
  - `internal/{practice,review,assessment,identity}/service.go`;
  - `internal/review/consumers.go`, `internal/review/notifications/worker.go`, `internal/assessment/consumers.go`;
  - `cmd/*/main.go` (`newPublisher`, `newConsumer`, `newPool`);
  - `internal/identity/store/{migrations/00001_init.sql,queries/outbox.sql}`;
  - `internal/e2e/coreloop_test.go`, `docker-compose.yml`, `.github/workflows/ci.yml`, `Makefile`, `sqlc.yaml`.
- **Read-only infra context:** `../infra/infrastructure/messaging/release.yaml` (NATS 2.14.6, `max_file_store` 5 Gi, the chart's values keys) and `../infra/apps/xlearn-*.yaml` (who has `NATS_URL`; identity doesn't).

## Context

NATS today (ADR-0035 Context):
- v2.14.6, one server, the `$G` account, **no auth**;
- 3 streams and 4 consumers with **no limits**, config hard-coded at `internal/platform/events/nats.go:78-83`;
- poison events are skipped after about 8 h with **no record** (`consumer.go:30-40,176-192`);
- identity publishes to a `LogPublisher`;
- review acks unknown subjects silently.

Server-first nkey auth ([mi-06](../sprints/sprint-mi-06.md): N1 → N2 → N3) needs client code that can use a seed and an inbox prefix, plus a golden ACL rendered from one topology table. Erase ([l-01](../sprints/sprint-l-01.md)) needs `XLEARN_IDENTITY` and dead-letter rows.

This sprint is **N0**: five dark changes and the integration test that proves the ACLs before any seed is mounted.

It merges to `main` and **rides v1.6.0**, which [m1-02](../sprints/sprint-m1-02.md) tags. m1-02 lists this sprint as an entry gate. Against today's anonymous server, nothing may change.

Facts checked while planning (2026-09-24):
- **identity already has `identity.outbox` and a relay** (`00001_init.sql`, `Service.NewOutboxRelay`). Only the publisher changes, so there's **no identity migration**.
- **Prod connection names** don't say which service owns them. Both review and assessment show `xlearn-XLEARN_PRACTICE-consumer`.
- **Prod `/jsz`:** 21 messages, 8,830 B.
- **The CI e2e lane** embeds `nats-server/v2 v2.10.22`.

## Entry gates: verify first

Stop and report if any gate is unmet.

- [ ] [m1-01](../sprints/sprint-m1-01.md) is merged. `docker-compose.yml` runs `nats:2.14` (JetStream on) and `postgres:18`.
- [ ] `git checkout main && git pull`. Then run the peer check: `gh pr list --state open`, `git worktree list`, ListAgents. No open peer PR adds a goose migration to `review` or `assessment`; if one does, plan to take the next free number at rebase.
- [ ] v1.6.0 isn't tagged (`git ls-remote --tags origin 'v1.6.*'`), or m1-02 is waiting on this sprint.

## Do this (in order)

1. **Branch.** Create `feat/mi-05-nats-n0` from `main`.

2. **`topology.go` [X]** (plan task 1):
   - Add `internal/platform/events/topology.go` with `Stream` and `Durable`.
   - Declare the **six streams** exactly as in the plan's stream table (ADR-0035 §1): `XLEARN_JUDGE` is 512 MiB, `MaxAge` 14 d, `Discard` Old; the others are Discard New.
   - Declare the **four live durables**. **Verify each** against the code: `review` and `assessment` on `XLEARN_PRACTICE`; `notifications` and `assessment` on `XLEARN_REVIEW`.
   - Put the remaining ADR-0035 durables in a comment only, naming the sprint that adds each: l-01, m3-05, m3-08, m4-03.
   - Make `NewNatsPublisher` look up its stream, and build `StreamConfig` from it in `ensureStream`. Delete the hard-coded config.
   - Make `NatsConsumer.Subscribe` refuse an undeclared `(stream, durable, filter)`.
   - Derive the services' `StreamSubjects` from the table, or assert them equal to it.
   - Add the tests: the **budget** (Σ ≤ 3.75 GiB, each > 0), the **declared entries**, and the **subject registry**:
     - `Emits ∩ Filter ⊆ Handles ∪ Ignores`, with `Handles ∩ Ignores = ∅`;
     - each consumer's default branch uses `topology.Ignored` and logs ERROR for an unlisted subject;
     - a per-consumer test drives every `Handles` subject;
     - each producer's `Subject*` constants ⊆ its stream's `Emits`.
   - Add the **live-stream update test**: a v1-shaped unlimited stream, then `ensureStream`, then assert the limits apply.

3. **ACL renderer [X]** (plan task 2):
   - Add `internal/platform/events/acl.go` with `RenderAuthorization(keys, legacy)`, which renders the ADR-0035 §2 ACL table for:
     - practice, review, assessment and identity;
     - judge and coach (own stream and publish only);
     - **ops** (`$JS.API.>` and `_INBOX.>`);
     - an optional `legacy` user plus `no_auth_user` for `LEGACY=allow|deny|none`.
   - Write both goldens with placeholder keys:
     - `testdata/nats-authorization.golden.conf`;
     - `testdata/nats-authorization.golden.yaml`, the values fragment for the pinned `nats` chart. Confirm the chart's key read-only.
   - Add the `-update` flag, and assert that `$` subjects are quoted.
   - Add `internal/platform/events/natsacl` (package main) and `make nats-acl-render NKEYS=… LEGACY=… FORMAT=conf|yaml`.

4. **Dead-letter hook [X]** (plan task 3):
   - **Hook:** add `DeadLetter`, `DeadLetterSink`, `WithDeadLetter` and `ErrClass`. On the failing delivery where `NumDelivered >= maxDeliver`: call the sink, then `Term()`, then log ERROR with ids only. A sink error still ends in `Term()`.
   - **Test option:** add `WithMaxDeliver(n)`, test-only, which also shortens `BackOff`.
   - **Migrations:** add the goose expand migration `event_dead_letter` (the columns and PK are in the plan) in **review** and **assessment**, at the next free version: today review `00004`, assessment `00003`.
   - **Queries:** add the sqlc queries `InsertDeadLetter` (`ON CONFLICT DO NOTHING`) and `ListDeadLetters`.
   - **Sinks:** make the stores implement the sink, and pass it on all four `Subscribe` calls.
   - **Tests:** unit-test with a fake `jetstream.Msg`.

5. **identity on NATS [X]** (plan task 4):
   - In `cmd/identity/main.go`, build the publisher like practice's `newPublisher`, on `XLEARN_IDENTITY` from the table.
   - `NewOutboxRelay(pub)`.
   - Add identity's `NATS_URL` config.
   - In docker-compose, set `NATS_URL: nats://nats:4222` on identity and add `depends_on` for nats.
   - **Don't** touch `../infra`. Prod identity stays on the `LogPublisher` until mi-06's identity N2 PR adds `NATS_URL` and the seed together ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) MI-5 table: identity joins 4222 at "N2, after N0"). Note that hand-off in status.md. If another sprint doc (for example m1-02) still plans an earlier identity `NATS_URL` PR, record the conflict in the decisions log and follow ADR-0035.

6. **Client options [X]** (plan task 5):
   - Add `events.Dial(ctx, svc, url, log)` and use it in both constructors, which now take the service name:
     - connection name `xlearn-<svc>:<stream>:pub|cons`;
     - `NATS_NKEY_SEED_FILE` → `nats.NkeyOptionFromSeed`;
     - `NATS_INBOX_PREFIX` → `nats.CustomInboxPrefix`, defaulting to `_INBOX_<svc>` when a seed is set;
     - an `ErrorHandler` that logs `ErrPermissionViolation` at ERROR.
   - **Fail closed:** in `cmd/{practice,review,assessment,identity}/main.go`, an init error with `NATS_URL` set exits non-zero. Remove the `LogPublisher` and "consumer disabled" fallbacks for that case; keep them only when `NATS_URL` is unset.
   - **Measure the v1 envelope maximum:**
     - the payload structs;
     - a read-only query on each prod outbox, which touches nothing (`ssh vps 'sudo k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select max(octet_length(payload_json::text)) from practice.outbox"'`), and the same for review, assessment and identity.
   - **Then add the cap:** `MaxEnvelopeBytes = 16 KiB` and `CheckEnvelope`. In `Relay.drain`, skip an oversize row: log ERROR once per id, leave it unsent, and continue the batch.

7. **Pool pins [X]** (plan task 6):
   - Add a shared `PG_MAX_CONNS` helper, default 4, and apply `poolCfg.MaxConns` in all six `newPool`s.
   - Add the tests.

8. **NATS-auth integration test [X]** (plan task 7):
   - **Harness:**
     - add `deploy/local/nats-acl.compose.yml`, using the `nats:2.14` image from `docker-compose.yml` with a mounted conf and JetStream;
     - write a `natsacl`-tagged Go test that generates throwaway nkeys (`nkeys.CreateUser()`) for the 7 identities, renders the conf, and boots or reloads the server per stage;
     - add `make nats-acl-test`;
     - add a `nats-acl` job in `.github/workflows/ci.yml` (ubuntu-latest has docker).
   - **Assert, through `events.Dial` and the real constructors,** every allowed and denied case in plan task 7:
     - all 6 services plus ops;
     - the denied cases fire the `ErrorHandler` with `ErrPermissionViolation`;
     - the stages `LEGACY=allow` (anonymous works), `deny` (anonymous connects, publish denied) and `none` (anonymous refused, `auth_required`).
   - If a real call needs a subject the ADR-0035 table lacks, add it **narrowly** and record it.

9. **Verify [X]:**
   - **Checks:** `gofmt -l .` (empty), `go vet ./...`, `go test -race ./...`, the e2e lane (`go test -tags e2e -race -count=1 ./internal/e2e/...` with the DB URL), `sqlc generate`, `sqlc diff`, web tests, `make nats-acl-test`.
   - **Compose:**
     1. `docker compose up --build`.
     2. Dev-login a fresh account, then `docker compose exec nats wget -qO- 'localhost:8222/jsz?streams=true'`: `XLEARN_IDENTITY` holds `account_created`, and the streams show their limits.
     3. Solve a problem, and the Revision queue updates.

10. **Docs [X]:**
    - `docs/architecture/events.md`: `topology.go` is the source of truth, the stream limits table, the dead-letter rule, identity's stream (dark in prod until mi-06 N2), the ACL render and `make nats-acl-render`.
    - `docs/architecture/data-model.md`: `review.event_dead_letter` and `assessment.event_dead_letter`.

## Constraints

- **Service boundaries (ADR-0003, ADR-0005):** each service touches only its own schema. `topology.go` is shared platform code, not a cross-service data path.
- **goose + sqlc:** expand-only DDL (new tables), embedded migrations run under the advisory lock, and committed sqlc output. **`sqlc diff` must pass.** Never `Down` in prod.
- **Outbox / inbox:** don't change the at-least-once semantics. The dead-letter hook only replaces "silently skip after `MaxDeliver`".
- **Dark:** against today's anonymous server, v1.6.0 must behave exactly like v1.5.2 for users. Every new option is optional env.
- **No infra change** in this sprint, and no `kubectl apply` ever. Prod reads are read-only (`get`, `exec … psql -c select`).
- **Consumers before producers, and the ACL PR before a consuming tag** (ADR-0035 §2 standing rule). No new durable is added here; the four live ones are only declared. Later sprints that add durables re-render and merge the ACL PR before their tag.
- **D34:** no alerting, pings or push channels. Dead letters are Postgres rows, read on demand.
- **Memory-sum rule (ADR-0035 §5):** no new pod, so there's nothing to check.
- **Parallel sessions:** check peers' PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents) before claiming goose numbers and before pushing. **Don't tag**: v1.6.0 belongs to m1-02.
- ADR numbers: you shouldn't need a new ADR. If you do, check peers' ADR numbers first. Record ACL-table deviations in the decisions log.

## Deliverables

- **Platform code:** `internal/platform/events/{topology.go,acl.go,natsacl/}`, the goldens under `testdata/`, and the dead-letter hook and client options in `consumer.go`, `nats.go` and `relay.go`.
- **Migrations and sqlc:** `event_dead_letter` in review and assessment.
- **Services:** identity publishes to `XLEARN_IDENTITY` when `NATS_URL` is set; fail-closed NATS init; `MaxConns` pins.
- **Tests:** budget, declared entries, subject registry, golden, dead letter, live-stream update, pool pins; the `natsacl`-tagged integration test.
- **Tooling:** `make nats-acl-render`, `make nats-acl-test`, the CI job `nats-acl`, `deploy/local/nats-acl.compose.yml`, and the docker-compose identity `NATS_URL`.
- **Docs:** `docs/architecture/events.md` and `docs/architecture/data-model.md`.

## Update status

- **This sprint's file:** set each task row in [`../sprints/sprint-mi-05.md`](../sprints/sprint-mi-05.md) to 🔄 or ✅ as it goes, and _Overall_ to ✅ when all 8 are done.
- **[`../status.md`](../status.md):**
  - **Sprint board:** mi-05 ✅, with the PR number.
  - **MI table, row MI-6:** "merged <date> (#PR); ships dark in v1.6.0". m1-02 updates it to "live in v1.6.0".
  - **NATS rows:** "N0 ready: golden at `internal/platform/events/testdata/`, integration test green. Identity prod `NATS_URL` + seed pending mi-06 N2."
  - **Decisions log:**
    - identity needed no outbox migration;
    - later durables are declared by the sprints that add them;
    - init fails closed when `NATS_URL` is set;
    - the extra `stream_seq` dead-letter column;
    - the measured v1 envelope maximum;
    - any ACL subject added beyond the ADR-0035 table (flag it for an ADR-0035 note).
- **ADRs:** none expected. If you make a notable call that isn't in ADR-0035, add an ADR after the peer ADR-number check.

## Done when (acceptance)

- [ ] `topology.go` is the only stream config, and the budget test (Σ 3.375 ≤ 3.75 GiB) and declared-entry test are green.
- [ ] The subject-registry test is green, and no consumer acks an unlisted subject silently.
- [ ] Golden `authorization` renders (conf and chart values) are committed, and stale fails CI. `make nats-acl-render` works for `LEGACY=allow|deny|none`.
- [ ] A last-delivery failure writes `event_dead_letter` (review, assessment), calls `Term()` and logs ERROR with ids only.
- [ ] identity's `account_created` lands in `XLEARN_IDENTITY` in compose, and prod stays dark (no `NATS_URL`).
- [ ] Client options work. Init fails closed when `NATS_URL` is set. The 16 KiB cap is enforced without stalling, and the v1 maximum is recorded.
- [ ] `MaxConns` is 4 in all six DB services.
- [ ] `make nats-acl-test` and CI `nats-acl` are green on NATS 2.14: the allowed and denied matrix for 6 services plus ops, and the N1/N3/N4 `legacy` stages.
- [ ] No behaviour change: unit, e2e, web and `sqlc diff` are green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: `feat/mi-05-nats-n0` in xlearn only (for example `feat(events): N0 topology, ACL render, dead letters, nkey client options (MI-6)`).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. CI includes the new `nats-acl` job.
3. **Release action — merge only (ships dark in v1.6.0):** Nothing deploys; it ships in `v1.6.0` (cut by [m1-02](../sprints/sprint-m1-02.md)). Don't tag. No infra PR here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn only). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
