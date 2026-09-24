# Sprint mi-05 — N0: NATS topology, dead letters, identity on NATS, client options, pool pins (MI-6)

> **Milestone:** MI — cluster safe for untrusted code (rollout step **MI-6**, NATS stage **N0**)   ·   **Track:** product (xlearn code, runs beside M1)
> **Prereqs:** [m1-01](sprint-m1-01.md) (compose parity: `nats:2.14`)   ·   **Unblocks:** [m1-02](sprint-m1-02.md) (cuts v1.6.0, which carries this sprint), [mi-06](sprint-mi-06.md) (N1 needs the golden file; N2 needs v1.6.0 live with this sprint's integration test green)
> **Release action:** **merge only (ships dark in v1.6.0).** [m1-02](sprint-m1-02.md) depends on this sprint, so N0 always rides v1.6.0. No tag and no infra PR here.
> **Calendar:** week 2 (Mon 2026-10-05 → Fri 2026-10-09), merged before m1-02 cuts v1.6.0.
> **Execute with:** [`../prompts/prompt-mi-05.md`](../prompts/prompt-mi-05.md). One prompt, one session.
>
> Sprint ids `mi-NN` are not rollout step ids `MI-N`. This sprint executes rollout step **MI-6**.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | `topology.go`: the single source of truth for streams and durables, plus its tests | X | ⬜ |
| 2 | ACL renderer: golden `authorization` block and `make nats-acl-render` | X | ⬜ |
| 3 | Dead-letter hook plus `event_dead_letter` in review and assessment | X | ⬜ |
| 4 | identity publishes `XLEARN_IDENTITY` through its existing outbox | X | ⬜ |
| 5 | Client options: seed, inbox prefix, ErrorHandler, fail-closed init, 16 KiB envelope cap | X | ⬜ |
| 6 | `pgxpool` `MaxConns` pins (L21) | X | ⬜ |
| 7 | NATS-auth integration test on NATS 2.14 (`make nats-acl-test` and a CI job) | X | ⬜ |
| 8 | Verify, update the docs, record | X | ⬜ |

> **Keep this current.** Set a task to 🔄 when you start it, to ✅ when its acceptance bullet passes, and to ⛔ if it's blocked (say why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, the MI table row **MI-6**, and the decisions log. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m1-01](sprint-m1-01.md) is merged: `docker-compose.yml` runs `nats:2.14` (JetStream on) and `postgres:18`. The integration test in task 7 needs the prod NATS minor.
- [ ] Local `main` is synced, and the peer check is done (`gh pr list`, `git worktree list`, ListAgents). No open peer PR adds a goose migration to `review` or `assessment`. If one does, take the next free version at rebase; CI fails on a duplicate.
- [ ] v1.6.0 isn't tagged yet, or m1-02 has confirmed it waits for this sprint (m1-02's entry gate says so).

## Goal

Land the five small **N0** code changes from [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) and the **NATS-auth integration test** from ADR-0035 §2 "Verification before any seed is mounted". Together they make server-first nkey auth ([mi-06](sprint-mi-06.md): N1 → N3) and erase ([l-01](sprint-l-01.md)) possible. Everything ships **dark** in v1.6.0.

Once this sprint lands:
- `internal/platform/events/topology.go` owns every stream's config and limits and every live durable consumer.
- The per-service NATS `authorization` block renders from that table into a golden file, and a stale render fails CI.
- A poison event leaves a durable Postgres row instead of vanishing after about 8 h.
- identity can publish to `XLEARN_IDENTITY`.
- Services accept an nkey seed and an inbox prefix.
- Pools are pinned.

Nothing changes against today's anonymous NATS server.

## Scope

**In**
- `internal/platform/events/topology.go`: all **six** v2 streams (ADR-0035 §1 table, Σ `MaxBytes` 3.375 GiB) and the **four live durables**. `ensureStream` reads the table, which removes the hard-coded config at `internal/platform/events/nats.go:78-83`. Tests: the budget, declared entries, and the **subject registry** (L20).
- The ACL renderer, with goldens in two forms:
  - NATS conf, used by the integration test;
  - the `messaging` chart values fragment, for mi-06's N1 PR.
  
  Also `make nats-acl-render`, which takes a public-key file and `LEGACY=allow|deny|none` for the N1, N3 and N4 stages.
- The dead-letter hook in `internal/platform/events/consumer.go`, plus `event_dead_letter` expand migrations in **review** and **assessment** (the services that consume today).
- identity's relay publishes to `XLEARN_IDENTITY` when `NATS_URL` is set. The outbox table and relay **already exist** (`identity.outbox` in `00001_init.sql`, `Service.NewOutboxRelay`), so **no identity migration** is needed. docker-compose gives identity `NATS_URL`.
- Client options from optional env: `NATS_NKEY_SEED_FILE` and `NATS_INBOX_PREFIX` (default `_INBOX_<svc>` when a seed is set), an `ErrorHandler` that logs permission violations at ERROR, per-service connection names, **fail-closed** init when `NATS_URL` is set, and a relay envelope cap of 16 KiB.
- `pgxpool` `MaxConns` 4 in all six DB-owning services (L21), with an optional `PG_MAX_CONNS` override.
- The NATS-auth integration test on NATS **2.14**, run by `make nats-acl-test` and a new CI job `nats-acl`.

**Out (later sprints)**
- **Server-side auth.** N1 users plus the `legacy` bridge, N2 seed mounts (**including identity's prod `NATS_URL` + seed**), and N3 `legacy` deny all go to [mi-06](sprint-mi-06.md). N4 goes to [mi-11](sprint-mi-11.md).
- **The v2 envelope type and v2 decoding** in consumers, the `path_slug` expand, and tagging v1.6.0 all go to [m1-02](sprint-m1-02.md).
- **Erase:**
  - erase consumers, identity's own `event_dead_letter`, and identity becoming a consumer go to [l-01](sprint-l-01.md);
  - coach's events client and the runtime use of `XLEARN_COACH` go to [l-01](sprint-l-01.md);
  - judge's erase consumer and the runtime use of `XLEARN_JUDGE` go to [m3-05](sprint-m3-05.md).
- **Later durables and ACL re-renders.** Each is added by the sprint that adds its `Subscribe` call, and that sprint merges the ACL PR before its tag:
  - practice on `XLEARN_JUDGE`: [m3-08](sprint-m3-08.md);
  - judge's analyzer on `XLEARN_PRACTICE` and review on `evaluation_analyzed`: [m4-03](sprint-m4-03.md).
- CNPG per-role `connectionLimit` 20 goes to [mi-11](sprint-mi-11.md). A NetworkPolicy for NATS callers goes to [mi-03](sprint-mi-03.md) (MI-5).

## Tasks

### 1 · `topology.go`: single source of truth [X]

Add `internal/platform/events/topology.go`, plain Go data with no I/O:

```go
type Stream struct {
    Name     string                  // XLEARN_PRACTICE
    Owner    string                  // practice (the only publisher; ACL + registry key)
    Subjects []string                // captured subjects, e.g. xlearn.practice.*
    Emits    []string                // every concrete subject Owner can publish (registry)
    MaxBytes int64
    MaxAge   time.Duration           // 0 = unlimited
    Discard  jetstream.DiscardPolicy // New | Old
}
type Durable struct {
    Stream, Name, Filter, Service string
    Handles, Ignores []string        // subject registry: Handles ∪ Ignores ⊇ Emits ∩ Filter
}
```

**Streams.** Six streams, values from ADR-0035 §1. The dedupe window stays 5 min for all of them.

| Stream | Owner | `MaxBytes` | `MaxAge` | Discard |
|---|---|---|---|---|
| `XLEARN_PRACTICE` | practice | 1 GiB | 0 | New |
| `XLEARN_REVIEW` | review | 1.5 GiB | 0 | New |
| `XLEARN_ASSESSMENT` | assessment | 128 MiB | 0 | New |
| `XLEARN_JUDGE` | judge | 512 MiB | 14 d | Old |
| `XLEARN_IDENTITY` | identity | 128 MiB | 0 | New |
| `XLEARN_COACH` | coach | 128 MiB | 0 | New |

`XLEARN_JUDGE` and `XLEARN_COACH` are **declared only**. They're budgeted and rendered into the ACL, but nothing publishes to them until [m3-05](sprint-m3-05.md) and [l-01](sprint-l-01.md).

**Live durables.** Four durables, matching today's code. Verify every value against the code before you commit.

| Stream | Durable | Filter | Service | Handles | Ignores |
|---|---|---|---|---|---|
| `XLEARN_PRACTICE` | `review` | `xlearn.practice.*` | review | `problem_solved`, `solution_revealed_early` | `attempt_logged` |
| `XLEARN_PRACTICE` | `assessment` | `xlearn.practice.*` | assessment | all three practice subjects | — |
| `XLEARN_REVIEW` | `notifications` | `xlearn.review.revision_due` | review | `revision_due` | — |
| `XLEARN_REVIEW` | `assessment` | `xlearn.review.*` | assessment | the four review subjects (check `internal/assessment/consumers.go`) | — |

The rest of the ADR-0035 §1 durable table is the **target shape**. Put it in a comment that points to the sprints that add each durable. Don't declare those durables now: no code subscribes to them, and they don't have names yet.

**Wiring.**
- `NewNatsPublisher` takes the stream by name, looks it up in the table (an undeclared stream is an error), and `ensureStream` builds `jetstream.StreamConfig` from it.
- `NatsConsumer.Subscribe` refuses a `(stream, durable, filter)` that isn't declared.
- The per-service `StreamSubjects` vars in `internal/{practice,review,assessment}/service.go` either derive from the table or are asserted equal to it by a test.
- Updating a **live, unlimited** stream applies the limits on the v1.6.0 rollout (`CreateOrUpdateStream`). Test that path: create the v1 config first, then call `ensureStream`, then assert the limits.

**Tests** (default `go test` lane):
- **Budget:** Σ `MaxBytes` ≤ 3.75 GiB, which is 75% of `max_file_store` 5 Gi in `../infra/infrastructure/messaging/release.yaml`. Every stream has `MaxBytes` > 0.
- **Declared entries:** every stream constant and durable constant used in `internal/*` and `cmd/*` resolves in the table. That includes each service's `DurableName`, `NotificationsDurable` and subject filters.
- **Subject registry** (ADR-0035 §1.1): for every durable, `Emits(stream) ∩ Filter ⊆ Handles ∪ Ignores`, and `Handles ∩ Ignores = ∅`.
  - Each consumer's `switch` default branch calls a `topology.Ignored(durable, subject)` check and **logs ERROR** for an unlisted subject. Today, `internal/review/consumers.go:58-61` acks silently.
  - A per-consumer test drives every `Handles` subject through the handler and asserts that none reaches the default branch.
  - Each producer's package asserts that its `Subject*` constants ⊆ its stream's `Emits`. identity's `SubjectAccountCreated` is one of them.

### 2 · ACL renderer and golden files [X]

Add `RenderAuthorization(keys map[string]string, legacy LegacyMode)` in `internal/platform/events/acl.go`. It turns the table into the per-service rules of the ADR-0035 §2 ACL table:

| Direction | Rule |
|---|---|
| Publish, events | `xlearn.<svc>.>` |
| Publish, own stream | `$JS.API.INFO`; `$JS.API.STREAM.{CREATE,UPDATE,INFO}.<OWN_STREAM>` |
| Publish, per consumed `(S, d)` | `$JS.API.CONSUMER.CREATE.<S>.<d>` and `….<d>.>` (a filtered create carries the filter); `$JS.API.CONSUMER.INFO.<S>.<d>`; `$JS.API.CONSUMER.MSG.NEXT.<S>.<d>`; `$JS.ACK.<S>.<d>.>` |
| Subscribe | `_INBOX_<svc>.>` only |
| Never | `STREAM.DELETE`, `STREAM.PURGE`, `STREAM.MSG.DELETE`, any other service's stream or durable |

**Identities rendered:**
- the six services: practice, review, assessment, identity, judge, coach. judge and coach get their own-stream and publish rules only, until their durables exist;
- **ops**: publish `$JS.API.>` (purge included) and subscribe `_INBOX.>`, for the `nats` CLI during break-glass;
- optionally **`legacy`** plus top-level `no_auth_user: legacy`:
  - `LEGACY=allow` renders it with allow `>` (N1). Its password is a placeholder in the golden, and `make nats-acl-render` generates a random one that no client ever uses, since `no_auth_user` maps anonymous connections to `legacy`;
  - `deny` renders `deny ">"` (N3);
  - `none` omits both (N4).

**Goldens** are rendered with placeholder public keys (`<NKEY_PUB:practice>` and so on):
- `internal/platform/events/testdata/nats-authorization.golden.conf`: NATS server conf, the form the integration test boots.
- `internal/platform/events/testdata/nats-authorization.golden.yaml`: the values fragment for the `nats` chart pinned in `../infra/infrastructure/messaging/release.yaml`. Confirm the chart's key (for example `config.merge`) by reading the chart's values schema; this is read-only.

The golden test fails when the files are stale, and `go test ./internal/platform/events -run Golden -update` rewrites them. Every `$`-prefixed subject stays **quoted** in both forms, because NATS conf reads an unquoted `$x` as a variable. Assert the quoting.

`make nats-acl-render NKEYS=<pubkeys.env> LEGACY=allow FORMAT=yaml` (via `go run ./internal/platform/events/natsacl`) prints the block with real public keys for mi-06's N1 PR. The input file holds `svc=U…` lines, public keys only.

### 3 · Dead-letter hook [X]

**Consumer side**, in `internal/platform/events/consumer.go`:
- When a handler fails on the delivery where `NumDelivered >= maxDeliver` (100, `consumer.go:30-40`), `dispatch` calls an injected sink, then `msg.Term()`, then logs at **ERROR** with **ids only**: `event_id`, subject, durable, `err_class` and `stream_seq`.
- The sink:
  ```go
  type DeadLetter struct { EventID, Subject, Durable, ErrClass string; StreamSeq uint64; At time.Time }
  type DeadLetterSink interface{ RecordDeadLetter(context.Context, DeadLetter) error }
  ```
  It's passed as `events.WithDeadLetter(sink)`.
- With no sink, the hook still calls `Term()` and logs ERROR.
- If the sink insert itself fails, log ERROR with the ids, then `Term()` anyway. The server won't redeliver past `MaxDeliver`.
- `ErrClass(err)` maps errors to `timeout`, `db`, `decode` or `other`.
- `handleTimeout` (25 s) stays below `AckWait` (30 s), so the last delivery returns an error before redelivery. Keep it that way.

**Service side:**
- Expand migrations, taking the **next free goose version** in each service. Today that's review `00004` and assessment `00003`. [m1-02](sprint-m1-02.md) numbers its migrations after these.
  ```sql
  CREATE TABLE IF NOT EXISTS <svc>.event_dead_letter (
      event_id text NOT NULL, subject text NOT NULL, durable text NOT NULL,
      err_class text NOT NULL, stream_seq bigint, at timestamptz NOT NULL DEFAULT now(),
      PRIMARY KEY (durable, event_id));
  ```
  - The table stores ids only, so it's erase-safe (ADR-0035 §1.2).
  - `stream_seq` goes beyond the ADR's column list, for break-glass inspection. Record that in the decisions log.
- sqlc queries: `InsertDeadLetter :exec` with `ON CONFLICT DO NOTHING`, and `ListDeadLetters :many`, read on demand.
- The review and assessment stores implement the sink, and every `Subscribe` in review (2 durables) and assessment (2) passes `WithDeadLetter`.
- practice, identity, coach and judge add their tables when they first consume ([m3-08](sprint-m3-08.md), [l-01](sprint-l-01.md), [m3-05](sprint-m3-05.md)).

**Tests:**
- A unit test with a fake `jetstream.Msg` covers three cases: the last-delivery failure records the row, calls `Term` and doesn't Nak; an earlier failure Naks with backoff; a sink error still ends in `Term`.
- A test-only `WithMaxDeliver(n)` option covers the integration path. NATS rejects a `BackOff` longer than `MaxDeliver`, so the option shortens `BackOff` too.

### 4 · identity on NATS [X]

identity already writes `account_created` to `identity.outbox` (`internal/identity/store/migrations/00001_init.sql`), and its relay drains through a `LogPublisher` (`internal/identity/service.go:101-105`). Change these pieces:
- **`cmd/identity/main.go`:** build the publisher exactly as `cmd/practice/main.go` `newPublisher` does, on stream `XLEARN_IDENTITY` from the table. Use `NatsPublisher` when `NATS_URL` is set and `LogPublisher` when it's unset (local dev).
- **`NewOutboxRelay`:** takes a `Publisher`.
- **identity's config:** gains `NATS_URL`.
- **docker-compose:** sets `NATS_URL: nats://nats:4222` on identity and adds `depends_on` for nats.
- **Prod:** identity has **no `NATS_URL`** (`../infra/apps/xlearn-identity.yaml`), so it stays on the `LogPublisher` in v1.6.0 (dark). Its `NATS_URL` and seed arrive together in [mi-06](sprint-mi-06.md)'s identity N2 PR, so identity's first prod connection already uses its own nkey. This follows [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md)'s MI-5 table (identity joins 4222 at "N2, after N0") and the rollout §2 MI-5 row, so no v1.6.0-era infra PR sets identity's `NATS_URL`. MI-5 ([mi-03](sprint-mi-03.md)) forward-declares identity on NATS 4222, so that order is safe.

Rows the `LogPublisher` already marked sent never reach the stream. That's fine: `account_created` has no consumer (`docs/architecture/events.md`: reserved).

### 5 · Client options [X]

Add one `events.Dial(ctx, svc, url string, log *slog.Logger) (*nats.Conn, error)` and use it in `NewNatsPublisher` and `NewNatsConsumer`. Both constructors gain the service name. Behaviour:
- **Connection name** `xlearn-<svc>:<stream>:pub|cons`, so `/connz` shows which service owns each connection. Today it shows `xlearn-XLEARN_PRACTICE-consumer` for both review and assessment.
- **`NATS_NKEY_SEED_FILE`:** when set, `nats.NkeyOptionFromSeed(path)`.
- **`NATS_INBOX_PREFIX`:** `nats.CustomInboxPrefix`. When a seed is set and the prefix isn't, it **defaults to `_INBOX_<svc>`** (and logs that), because under the ACL the default `_INBOX.` would be denied.
- **`nats.ErrorHandler`:** logs `nats.ErrPermissionViolation` at **ERROR** with the subject, and anything else at WARN.
- **Fail closed:** when `NATS_URL` is set, a publisher or consumer init error (a bad or missing seed, bad options) **exits the process** with a clear error.
  - Today, `newPublisher` in `cmd/{practice,review,assessment}/main.go` falls back to the `LogPublisher`. That would mark outbox rows **sent without delivering them**, which is silent loss once seeds exist.
  - `newConsumer` disables consumers.
  - Keep both fallbacks only for `NATS_URL` unset. Against today's server, init never fails (`RetryOnFailedConnect`), so nothing changes.
- **Envelope cap:** `const MaxEnvelopeBytes = 16 << 10` (L20).
  - In `Relay.drain`, a row larger than the cap isn't published. The relay logs ERROR **once per event id per process** (`event_id`, subject, size), leaves the row unsent, and **continues** with the batch instead of stalling.
  - Add `events.CheckEnvelope([]byte) error` (`ErrEnvelopeTooLarge`) for producers. m1-02's v2 envelope type should call it.
  - **Measure the v1 maximum first:** the payload structs, plus a read-only `max(octet_length(payload_json::text))` on each prod outbox. Prod `/jsz` on 2026-09-24: 21 messages, 8,830 B, about 0.4 KiB each. Record the result.

### 6 · Pool pins (L21) [X]

Add a small shared helper, for example `config.PGMaxConns(def int32)`, which reads an optional `PG_MAX_CONNS` and defaults to **4**. Apply it in every `newPool` (`cmd/{identity,curriculum,practice,review,assessment,coach}/main.go`) as `poolCfg.MaxConns = …`. judge will use 8 ([m3-05](sprint-m3-05.md)). The gateway has no pool.

pgx's default is max(4, NumCPU), which is already 4 on the 4-vCPU node, so this is a no-op in prod today. It stops the pool from doubling on an R1 upgrade to KVM 8. Unit-test the helper, and add a test per service that asserts the pinned value is applied.

### 7 · NATS-auth integration test [X]

A build-tagged (`natsacl`) Go test, run by `make nats-acl-test` and a new `nats-acl` job in `.github/workflows/ci.yml`.

**Setup:**
- Generate throwaway nkeys for the 7 identities with `github.com/nats-io/nkeys`, which is already in the module graph.
- Render the conf through task 2's renderer, into a temp dir.
- Boot NATS **2.14**, the same image m1-01 pinned in `docker-compose.yml`, from a small `deploy/local/nats-acl.compose.yml` that mounts the conf with JetStream on.
- Run the test against it. Drive the real `events.Dial`, `NewNatsPublisher` and `NewNatsConsumer` with the client options; don't use raw nats.go only.

**Allowed, for every service:**
- `ensureStream` on its own stream (all six, judge and coach included);
- publish `xlearn.<svc>.probe` and get the PubAck;
- for each of its declared durables: a filtered consumer create, a fetch, an ack, and a `NakWithDelay`.

**Denied, for every service.** The `ErrorHandler` must fire with `ErrPermissionViolation` for each of these:
- publishing on another service's subject;
- `UpdateStream` on another service's stream;
- `PurgeStream` and `DeleteStream` on **its own** stream;
- creating or fetching a durable it doesn't own;
- subscribing to another service's inbox or to `_INBOX.>`.

**ops** can purge a scratch stream.

**Stages**, each a restart of the test server with a different render:
- `LEGACY=allow`: an anonymous client connects as `legacy` and works end to end, as v1.5.2-era clients will during N1.
- `LEGACY=deny`: an anonymous client connects but every publish is denied (N3).
- `LEGACY=none`: an anonymous connect is refused and `/varz` shows `auth_required: true` (N4).

The ADR-0035 §2 ACL table is authoritative. If the test shows that a real client call needs a subject the table lacks (a JetStream API variant, for example), add it to the renderer **narrowly**. Record it in the decisions log, and flag it for an ADR-0035 note.

### 8 · Verify, update the docs, record [X]

**Checks:**
- `gofmt`, `go vet`, `go test -race ./...`;
- the `e2e` lane, which uses anonymous embedded NATS and must be unchanged;
- `sqlc generate`, then `sqlc diff`;
- web tests;
- `make nats-acl-test`.

**Compose smoke:**
1. Run `docker compose up --build`.
2. Dev-login a fresh account.
3. Confirm `account_created` is in `XLEARN_IDENTITY`: `docker compose exec nats wget -qO- 'localhost:8222/jsz?streams=true'`.
4. Confirm the other streams show their limits.
5. Solve a problem and see the Revision queue update, which shows the event flow is unchanged.

**Docs:**
- `docs/architecture/events.md`: `topology.go` is the source of truth; add the stream limits table, the dead-letter rule, identity's stream and the ACL render.
- `docs/architecture/data-model.md`: the `event_dead_letter` tables.

**Record:** see Update status in the prompt.

## Acceptance criteria

- [ ] `topology.go` is the only stream config: `ensureStream` reads it, and `nats.go`'s hard-coded config is gone. The budget test (Σ 3.375 GiB ≤ 3.75 GiB) and the declared-entry test are green.
- [ ] The subject-registry test is green: every v1 subject is handled or explicitly ignored per durable, and no consumer acks an unlisted subject silently.
- [ ] Golden `authorization` renders (conf and chart values) are committed, and a stale render fails CI. `make nats-acl-render` prints the block from a public-key file for `LEGACY=allow|deny|none`.
- [ ] A message that fails its last delivery leaves a row in `review.event_dead_letter` or `assessment.event_dead_letter`, gets `Term()`, and logs ERROR with ids only (unit and integration).
- [ ] With `NATS_URL` set, identity's `account_created` lands in `XLEARN_IDENTITY` (compose). Without it, behaviour is as before.
- [ ] Seed, inbox prefix, `ErrorHandler` and connection names work. An init failure with `NATS_URL` set is fatal, with no `LogPublisher` fallback. The relay refuses envelopes over 16 KiB without stalling, and the v1 maximum is measured and recorded.
- [ ] `MaxConns` is 4 in all six DB-owning services (`PG_MAX_CONNS` overrides).
- [ ] `make nats-acl-test` and the CI `nats-acl` job are green on NATS 2.14:
  - allowed operations pass and denied operations are denied, for all 6 services plus ops;
  - the three `legacy` stages behave as N1/N3/N4 expect.
- [ ] No behaviour change: `go test -race ./...`, the e2e lane, web tests and `sqlc diff` are green, and nothing changes against an anonymous server.

## Release

**Merge only. This ships dark in v1.6.0.** The tag is cut by [m1-02](sprint-m1-02.md), which lists this sprint as an entry gate and runs the ADR-0034 §6 release checklist.

This sprint opens no infra PR:
- the golden block is pasted by [mi-06](sprint-mi-06.md) N1;
- identity's prod `NATS_URL` and seed land in mi-06's identity N2 PR.

After v1.6.0 is live, mi-06's N2 gate reads "v1.6.0 live, with the NATS-auth integration test green". The status.md MI-6 row records both facts.

## Definition of Done

- CI is green: `go`, `e2e`, the new `nats-acl` job, and `sqlc diff`.
- The work is squash-merged to `main`, with no tag.
- Compose shows `XLEARN_IDENTITY` populated.
- The docs are updated (`events.md`, `data-model.md`).
- Statuses are updated (this file and [`../status.md`](../status.md)).
- Notable calls are in the decisions log. Any change to the ADR-0035 ACL table is flagged for an ADR note.

## Risks / watch-outs

- **The subject-registry test will surface v1 subjects that are acked silently today.** review ignores `attempt_logged` (`internal/review/consumers.go:58-61`). Fix it by listing each one explicitly under `Ignores`, not by widening `Handles`.
- **The envelope cap could refuse an existing large payload.** Measure the v1 maximum first (task 5). Prod is at about 0.4 KiB per event.
- **The `LogPublisher` fallback on init failure is silent loss** (it marks rows sent) once seeds exist. That's why init fails closed when `NATS_URL` is set (task 5). Don't keep the fallback "for safety".
- **Stream limits change on a live stream at the v1.6.0 rollout.** The owning service's `CreateOrUpdateStream` applies `MaxBytes` and `Discard` to today's unlimited streams. That's covered by task 1's update-path test. Usage is KBs, so nothing is discarded.
- **Conf and chart-values renders can drift**, for example in `$` quoting or the chart key. The integration test proves the conf form. mi-06's N1 check (`/connz?auth=true`, every connection `legacy`) proves the chart form.
- **The ACL table may be missing a subject a real client needs.** Only the integration test finds that. Add the subject narrowly and record it; never widen to `$JS.API.>`, which is the nats-server#3202 exposure.
- **NATS version:** the e2e lane embeds `nats-server/v2 v2.10.22`. The ACL test must run the **prod minor 2.14**, which is why it uses the compose image.
- **Goose numbering:** review and assessment take their next free version here. m1-02, and any peer, numbers after at rebase.
