# Sprint l-01 — L-E erase consumers (practice, review, assessment, coach) + identity ack path → v1.11.0

> **Milestone:** L — learner gate (**L-E**, consumer half)   ·   **Track:** product (identity-centred; runs beside M3)
> **Prereqs:** [mi-06](sprint-mi-06.md) (N1–N3; this sprint does the ≥ 24 h N3 re-check) · [mi-03](sprint-mi-03.md) (MI-5, MI-5a) · [m2-05](sprint-m2-05.md) (v1.10.0 live)
> **Unblocks:** [l-02](sprint-l-02.md) (the producer, one tag later) · [m3-05](sprint-m3-05.md) (judge's erase consumer reuses this pattern) · [m3-07](sprint-m3-07.md) and [mi-11](sprint-mi-11.md) (both gate on the N3 re-check record and on coach's nkey)
> **Release action:** **tag v1.11.0** (indicative: the next free minor) + **infra PR(s) first**: the re-rendered NATS ACL (coach user, new durables) and coach's seed/env, merged **before the xlearn squash-merge** (so before the tag, and before any peer tag could ship this code). Consumers ship one tag before their producer ([l-02](sprint-l-02.md), v1.12.0).
> **Calendar:** early November, after v1.10.0. No owner involvement (the agent generates coach's nkey in-session, as [mi-06](sprint-mi-06.md) task 1 allows).
> **Execute with:** [`../prompts/prompt-l-01.md`](../prompts/prompt-l-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | N3 ≥ 24 h re-check (MI-7) + record | H | ⬜ |
| 2 | Service erase consumers (practice, review, assessment, coach) on a shared `internal/platform/erase` handler + the terminal-error path in `consumer.go` | X | ⬜ |
| 3 | coach on NATS: events client, outbox + relay, `XLEARN_COACH` | X | ⬜ |
| 4 | identity side: `erase_request`, `erase_ack`, `released_username`, dead letters, `GET /internal/erasures/{id}`, ack durables, `identity admin erasures` | X | ⬜ |
| 5 | Tombstone checks in async consumers and projection replay | X | ⬜ |
| 6 | topology + golden ACL + subject registry + NATS-auth integration test + e2e | X | ⬜ |
| 7 | Infra PR(s): ACL re-render (coach user, new durables) + coach seed/env + `IDENTITY_BASE_URL` — merged before the xlearn squash-merge | I | ⬜ |
| 8 | Tag v1.11.0 (release checklist) | X | ⬜ |
| 9 | Post-tag verify on prod + record | H + X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> L milestone row, the NATS rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-7 N3 applied ≥ 24 h ago** ([mi-06](sprint-mi-06.md)): the N3 timestamp is recorded in status.md's NATS rows. The ≥ 24 h re-check itself is task 1.
- [ ] **MI-5 live** ([mi-03](sprint-mi-03.md), first PR): the `messaging` ingress policy admits NATS 4222 from practice, review, assessment, identity **and coach (forward-declared)**.
- [ ] **MI-5a live** ([mi-03](sprint-mi-03.md), second PR): identity admits same-namespace pods on :8081 (`/internal/*`), so practice, review, assessment and coach can reach `GET /internal/erasures/{id}`.
- [ ] **v1.10.0 live** ([m2-05](sprint-m2-05.md)): `/xlearn/api/v1/healthz` reports ≥ 1.10.0.
- [ ] **identity is on NATS with its nkey** ([mi-06](sprint-mi-06.md) N2 for identity): the `XLEARN_IDENTITY` stream exists and identity's relay drains. The new durables bind to that stream, and a durable can't be created on a missing stream.
- [ ] **Parallel sessions:** the next free minor is known (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents); no open peer PR adds a goose migration to identity, practice, review, assessment or coach (if one does, take the next free version at rebase), and none edits `internal/platform/events/topology.go` or `consumer.go` (serialize the golden re-render and the dispatch change).

## Goal

Ship **every erase consumer one tag before the producer** ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)):
- practice, review, assessment and coach each bind a durable on `XLEARN_IDENTITY` for `xlearn.identity.account_erasure_requested`,
  **re-verify** the request with identity, and in one transaction write a tombstone, delete the account's rows **including its outbox
  rows**, and emit their ack `xlearn.<svc>.account_erased`;
- identity gains the request/ack/cooldown tables, the re-verify endpoint, its ack durables on the four service streams, and the close
  rule (all acks that `topology.go` of the running version expects: **4 at L-E**);
- coach makes its **first NATS use** (`XLEARN_COACH`, its own nkey — N2 for coach).

Nothing can trigger an erase yet: identity's producer (`DELETE /api/me`, the CLI verb) is [l-02](sprint-l-02.md) in v1.12.0. After
this tag every consumer is bound and idle.

## Scope

**In**
- `internal/platform/erase`: the shared consumer handler (decode → re-verify → service transaction → ack), the payload types, the
  deterministic ack id and the identity re-verify client. judge reuses it at M3-1.
- `internal/platform/events/consumer.go`: a **terminal-error path** (`events.Terminal(class, err)` → dead-letter row + `Term()` on
  **any** delivery), so forged or undecodable events are dead-lettered at once instead of being retried ~100 times by mi-05's
  last-delivery hook.
- Expand migrations: `erased_account` tombstones in practice, review, assessment, coach; `event_dead_letter` where missing (practice,
  coach, identity); coach's `outbox`; identity's `erase_request`, `erase_ack`, `released_username` (+ `outbox.account_id` if M1a didn't add it).
- The four service consumers; tombstone checks in every async consumer and in projection replay.
- coach: events client, outbox + relay, `XLEARN_COACH` (128 MiB, New; already budgeted in `topology.go`).
- identity: `GET /internal/erasures/{id}`, the ack durables and the close rule, `identity admin erasures [--open]` (the on-demand read that replaces t1's "open > 24 h" ops check under D34). **Producer path not wired.**
- topology, golden ACL, subject registry, the NATS-auth integration test and an e2e over the real broker.
- Infra: the ACL re-render with coach's user, coach's seed + env, `IDENTITY_BASE_URL` for practice, assessment and coach.
- Tag v1.11.0.

**Out (later sprints)**
- The producer: identity's erase transaction, `DELETE /api/me`, `identity admin account erase`, the Settings UI (AB21), the username
  hold on claim, P11's erase part and the erase-tag snapshot → [l-02](sprint-l-02.md) (v1.12.0).
- judge's erase consumer (born at M3-1) and identity's ack durable on `XLEARN_JUDGE` → [m3-05](sprint-m3-05.md) / [m3-07](sprint-m3-07.md).
- Clearing redeemed invites' `note` on erase → [l-03](sprint-l-03.md) (it creates `invite`).
- Web erase for every non-owner → [l-04](sprint-l-04.md).
- `xlearn` egress policies (MI-15) → [mi-11](sprint-mi-11.md): it must include the callers this sprint adds (practice, assessment, coach → identity :8081; coach → NATS :4222), recorded in task 9.
- N4 (remove `legacy`) → [mi-11](sprint-mi-11.md). The re-seal runbook for pseudonymous stream copies stays on request (ADR-0027 §6).

## Tasks

### 1 · N3 ≥ 24 h re-check (MI-7) [H]

Sources: [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (N3: "no `legacy` connection right after the reload and again ≥ 24 h later").
- Before touching coach's key: `ssh vps 'bash -s -- --cluster --nats-stage=n3' < ../infra/hack/host-verify.sh` at least 24 h after the recorded N3 time.
  It must show **no `legacy` connection** and every other check green.
- Record the run (date, result) in status.md's NATS rows: "N3 ≥ 24 h re-check ✅ <date>". [m3-07](sprint-m3-07.md) and [mi-11](sprint-mi-11.md) gate on this record.
- A `legacy` connection means some client still connects anonymously: stop, find it (`/connz?auth=true`), and fix it through its own infra PR before continuing.

### 2 · Service erase consumers [X]

Sources: [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo) steps 3–4,
[ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) as amended by [ADR-0028 §3](../../adr/0028-object-storage-and-backups.md#3-amendments-to-adr-0027)
(no erase ledger: delete + outbox delete + tombstones), [ADR-0033 §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls), [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes).

**The contract (fixed here, consumed by l-02's producer).** In `internal/platform/events`:
- `SubjectAccountErasureRequested = "xlearn.identity.account_erasure_requested"`, data `{erase_request_id, account_id, requested_at}`;
- `SubjectAccountErased(svc) = "xlearn.<svc>.account_erased"`, data `{erase_request_id, account_id, service}`;
- both as v2 envelopes (account-scoped: no `path_slug`, per m1-02's envelope rules; registered as **account-scoped** in
  `topology.go`'s subject registry even though `xlearn.practice.*` / `xlearn.review.*` are otherwise course-scoped, task 6), with shared fixtures
  `internal/platform/events/testdata/account_erasure_requested.v2.json` and `account_erased.v2.json`. l-02's producer must match the
  first byte for byte.

**The terminal-error path (`internal/platform/events/consumer.go`).** `events.Handler` is `Handle(ctx, Event) error`: a handler
never sees the `jetstream.Msg`, so it can't `Term()` itself, and mi-05's dead-letter hook fires only when `NumDelivered >= maxDeliver`
(100 deliveries, about 8 h of backoff). Returning a plain error for a forged event would retry it ~100 times. So this sprint adds, in
`internal/platform/events`:
```go
// TerminalError marks a failure that must never be retried: dispatch dead-letters it on the current delivery.
type TerminalError struct{ Class string; Err error }
func (e *TerminalError) Error() string
func (e *TerminalError) Unwrap() error
func Terminal(class string, err error) error // → &TerminalError{Class: class, Err: err}
```
- `dispatch`: when `errors.As(err, &te)` finds a `*TerminalError` — on **any** delivery — it calls the injected `DeadLetterSink`
  (mi-05's `DeadLetter{EventID, Subject, Durable, ErrClass: te.Class, StreamSeq, At}`), then `msg.Term()`, then logs ERROR with
  ids only; it never Naks. With no sink, `Term()` + ERROR log; a sink error is logged and `Term()` runs anyway (mi-05's rules).
  Every other error keeps mi-05's path unchanged (Nak with backoff; the last delivery → sink + `Term()`).
- `ErrClass(err)` returns `te.Class` for a `*TerminalError` (even wrapped with `%w`), so the column carries `decode`,
  `erase_unverified` or `erase_unknown_request`; other errors keep `timeout|db|decode|other`.
- Unit tests in `internal/platform/events/consumer_test.go` with mi-05's fake `jetstream.Msg`: a `Terminal` error on delivery **1**
  records the row with its class, calls `Term` and never `Nak`; a `fmt.Errorf("…: %w", Terminal(…))` behaves the same; a sink
  failure still ends in `Term`; a plain error on delivery 1 still Naks (regression).
- Hand-off: every later consumer with a "never retry" case uses `events.Terminal` — in particular judge's erase consumer
  ([m3-05](sprint-m3-05.md)), which reuses `erase.NewHandler` and so gets this path, not a log-and-ack.

**`internal/platform/erase`** (new, no I/O of its own beyond the injected pieces):
```go
type Request struct{ EraseRequestID, AccountID string; RequestedAt time.Time }
type Verifier interface{ Verify(ctx context.Context, r Request) error } // nil | ErrUnverified | transient error
type Store interface{ EraseTx(ctx context.Context, r Request, ackEventID string) error } // one tx in the service's own schema
func NewHandler(svc string, v Verifier, st Store, log *slog.Logger) events.Handler
func AckEventID(svc, eraseRequestID string) string // uuidv5(ns, "account_erased:"+svc+":"+id)
func NewHTTPVerifier(identityBaseURL string, c *http.Client) Verifier // GET /internal/erasures/{id}, 5 s timeout
```
The handler records no dead-letter row itself: it returns `events.Terminal(class, err)` and the subscription's
`WithDeadLetter(<svc store>)` sink writes the row. Handler outcomes:

| Case | Handler returns → consumer action |
|---|---|
| Payload doesn't decode | `events.Terminal("decode", err)` → dead-letter row (`err_class=decode`) + `Term()` on this delivery; ERROR log with ids only |
| identity says 404, `account_gone=false`, or the request's `account_id` ≠ the event's | **nothing is deleted**; `events.Terminal("erase_unverified", ErrUnverified)` → dead-letter row (`err_class=erase_unverified`) + `Term()` on this delivery, never retried; ERROR log with ids only. This is the forged-event path |
| identity unreachable, timeout, 5xx, or a 404 whose body isn't identity's `writeError` JSON with `error.code = "not_found"` (a wrong `IDENTITY_BASE_URL` must never look like a forged event) | a plain error → nak with backoff (the platform consumer's retry; the last delivery → mi-05's hook) |
| Verified | `EraseTx` → `nil` → ack |
| `EraseTx` fails | a plain error → nak with backoff |

**`EraseTx`, per service, one transaction in its own schema** (no cross-schema reads, [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):
1. `INSERT INTO <svc>.erased_account (account_id, erase_request_id, erased_at) … ON CONFLICT (account_id) DO NOTHING`.
2. `DELETE … WHERE account_id = $1` on every per-user table (child tables go through their `ON DELETE CASCADE` FKs), and
   `DELETE FROM <svc>.outbox WHERE account_id = $1` (the column m1-02 added in M1a).
3. Insert the ack outbox row: subject `xlearn.<svc>.account_erased`, `event_id = AckEventID(svc, id)` with `ON CONFLICT (event_id) DO NOTHING`,
   and **`outbox.account_id = NULL`** — otherwise a redelivered erase would delete its own unsent ack in step 2.

Starting table lists (verify against the schema at the time — M2 and M3 add tables):

| Service | Deletes (children cascade) |
|---|---|
| practice | `user_problem_state` (→ `attempt` → `stage_event`, `timer`, `outcome`), `attempt` rows by `account_id` (denormalized at M1a), any M2 touch rows, `outbox` |
| review | `revision_item` (→ `touch_result`), `mistake_entry`, `reminder`, `weak_area_snapshot`, `outbox` |
| assessment | `mock_session` (→ `rubric_score`, `mock_session_item`), every `proj_*` table (`coverage`, `mastery`, `heatmap`, `outcome_mix`, and M2's `activity`, `touch_stats`, …), `outbox` |
| coach | `api_key_config` (the encrypted BYO keys; → `key_default`, m1-02's M1a table, which carries `account_id` and cascades from `api_key_config` — list it in the coverage test's cascade allow-list, or delete it by `account_id` first), `coach_thread` (→ `coach_message`), `message_quota_day` (m1-07's daily cap rows, handed to this sprint by name), `outbox` |

`inbox` and `event_dead_letter` hold ids only and stay; `erased_account` is kept forever
([t1 §2](../research/t1-content-data-model.md#per-user-data)). assessment's `EraseTx` takes the projection replay lock **shared**, the same
way the live projection handler does ([m2-05](sprint-m2-05.md) task 4), so a running replay can't resurrect rows mid-erase.

**Durables** (all on `XLEARN_IDENTITY`, filter `xlearn.identity.account_erasure_requested`, `DeliverAll` so no erase is ever missed):
`practice-erase`, `review-erase`, `assessment-erase`, `coach-erase`. Each `Subscribe` passes `events.WithDeadLetter(<svc store>)`.
Idempotency comes from the handler itself (tombstone `DO NOTHING`, deletes, deterministic ack id); services with an `inbox` may also
claim the event there, but practice and coach don't need one for this handler.

**Config.** practice and assessment gain `IDENTITY_BASE_URL` (review already has it; coach gains it in task 3). **When `NATS_URL` is set
and `IDENTITY_BASE_URL` is not, the service refuses to start** with a clear error: an erase consumer must never run without its
re-verify path. The infra PR (task 7) adds the env **before this code is merged** (see task 7's order). docker-compose sets it for all four.

**Tests** (real Postgres 18, per service store): seed two accounts across every per-user table and the outbox; erase one; assert zero rows
for it and untouched rows for the other; the ack row exists with `account_id IS NULL`; a second run is a no-op with exactly one ack row.
**Coverage test:** `information_schema.columns` lists every table in the service's schema with an `account_id` column; each must be in
the erase list, the cascade allow-list (children deleted through an `ON DELETE CASCADE` FK, e.g. coach's `key_default`) or an explicit
exempt list (`erased_account`) — a table added later without an erase path turns CI red. Handler unit tests with a fake verifier cover
every row of the outcome table, asserting `errors.As(err, *events.TerminalError)` and its `Class` for the two terminal rows and a
plain (non-terminal) error for the two retry rows.

### 3 · coach on NATS [X]

Sources: [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes) (`XLEARN_COACH`, chosen over an HTTP ack), [t1 §4 coach](../research/t1-content-data-model.md#coach).
- Expand migration (next free coach version): `coach.outbox (event_id uuid PK, subject, payload_json jsonb, created_at, sent_at, account_id uuid NULL)`
  + the unsent index, `coach.erased_account`, `coach.event_dead_letter` (mi-05's shape). sqlc queries in `internal/coach/store/queries/{outbox,erase}.sql`;
  `sqlc generate`, `sqlc diff` clean.
- `internal/coach/config.go`: `NATS_URL`, `IDENTITY_BASE_URL` (the N0 client env `NATS_NKEY_SEED_FILE` / `NATS_INBOX_PREFIX` is read by `events.Dial`).
  Update the package comment ("coach emits no events …") — it now emits its erase ack and consumes one subject.
- `cmd/coach/main.go`: the publisher on `XLEARN_COACH` from the table (`NatsPublisher` when `NATS_URL` is set, `LogPublisher` when unset,
  fail-closed init per mi-05), the outbox relay goroutine, and the `coach-erase` consumer; shutdown order per the existing services
  (stop consumers → drain relay → close pool).
- docker-compose: `NATS_URL`, `IDENTITY_BASE_URL` and `depends_on: nats` on coach.
- coach keeps its user paths unchanged: no event on chat, no other subscription.

### 4 · identity side [X]

Sources: [ADR-0033 §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls), [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes) (identity consumes the 4 service streams), [t1 §4 identity](../research/t1-content-data-model.md#identity), [t7 §2.3](../research/t7-cross-cutting-and-rollout.md#23-admission-design-identity-owns-it).

**Expand migration** (next free identity version; constant defaults only):
- `identity.erase_request (id uuid PK, account_id uuid NOT NULL /* no FK: the account row is deleted in the same tx */, requested_at timestamptz NOT NULL DEFAULT now(), requested_via text NOT NULL CHECK (requested_via IN ('web','cli')), expected_acks text[] NOT NULL, closed_at timestamptz)`
  + `CREATE UNIQUE INDEX erase_request_one_open ON identity.erase_request (account_id) WHERE closed_at IS NULL` (L8: 1 open per account).
- `identity.erase_ack (erase_request_id uuid REFERENCES identity.erase_request ON DELETE CASCADE, service text, acked_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (erase_request_id, service))`.
- `identity.released_username (username_sha256 bytea PRIMARY KEY, released_at timestamptz NOT NULL, available_at timestamptz NOT NULL)` —
  sha256 of the normalized lower-case username; 60-day cooldown. Nothing writes it until [l-02](sprint-l-02.md).
- `identity.event_dead_letter` (mi-05's shape): **identity becomes a consumer here** and wires the dead-letter hook (ADR-0035 §1).
- `identity.outbox.account_id uuid` **if M1a didn't add it** (backfilled from `payload_json->>'account_id'`; the writer sets it) — l-02's
  erase transaction deletes identity's own rows for the account by it.

**`GET /internal/erasures/{id}`** in `internal/identity/service.go` beside `/internal/accounts/{id}` (same ClusterIP + MI-5a trust model; no
user JWT). 200 → `{id, account_id, requested_at, account_gone, state: "open"|"closed"}` with `account_gone` computed live
(`NOT EXISTS (SELECT 1 FROM identity.account WHERE id = account_id)`); unknown id → 404 `not_found`. No PII.

**Ack durables** — `identity-erase-acks` on `XLEARN_PRACTICE`, `XLEARN_REVIEW`, `XLEARN_ASSESSMENT` and `XLEARN_COACH`, each filtered to that
stream's `xlearn.<svc>.account_erased`, `DeliverAll`. The handler: decode → `INSERT INTO erase_ack … ON CONFLICT DO NOTHING` → if every
service in the request's `expected_acks` has acked, `UPDATE erase_request SET closed_at = now() WHERE id = $1 AND closed_at IS NULL`; log INFO
with ids only. A payload that doesn't decode → `events.Terminal("decode", err)`; an ack for an unknown request →
`events.Terminal("erase_unknown_request", err)`: both become a dead-letter row in `identity.event_dead_letter` + `Term()` on that
delivery via task 2's terminal-error path (each `Subscribe` passes `events.WithDeadLetter(<identity store>)`); a DB error is a plain
error → nak with backoff. An ack from a service outside `expected_acks` (e.g. judge replaying old requests at M3-1) is recorded and
doesn't block anything.

**The expected set.** `events.ErasureAckServices()` returns every service that declares a durable filtered on
`xlearn.identity.account_erasure_requested` in `topology.go` — **{practice, review, assessment, coach} at L-E**. l-02's erase transaction
snapshots it into `expected_acks` when a request opens, so a request opened before judge exists never waits for judge (a test pins
the L-E set).

**`identity admin erasures [--open] [--since <RFC3339>]`** in `internal/identity/admin/` (m1-04's CLI): id, account id, requested at,
via, acks received / expected (missing services named), closed at. A read verb that writes its `admin_audit` row like every verb.
It is the on-demand check that replaces t1's "open > 24 h" ops alert (D34) and the list R-d step 3 needs
([ADR-0034 §4.2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#42-r-d-is-a-procedure-not-a-button)).

Tests: store integration for the tables and the one-open index; handler tests (close on the last expected ack, duplicate acks, unexpected
service, unknown request → a `*events.TerminalError` with class `erase_unknown_request`, undecodable → class `decode`);
`GET /internal/erasures/{id}` shapes; the CLI verb's output and audit row.

### 5 · Tombstone checks [X]

Sources: [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo) step 4 (async consumers and projection replay; not user write handlers).
The stream keeps pseudonymous copies of an erased account's events, and a consumer may lag the erase, so every **async** path that
could recreate rows skips a tombstoned account:
- review's consumers on `XLEARN_PRACTICE` (`problem_solved`, `solution_revealed_early`, M2's `touch_concluded`) and review/notifications;
- assessment's consumers on `XLEARN_PRACTICE` and `XLEARN_REVIEW`, and the projection replay (`assessment admin replay-projections`, [m2-05](sprint-m2-05.md));
- any background worker that resolves an account through identity treats a 404 as "skip".

Mechanism: inside the consumer's transaction, `SELECT 1 FROM <svc>.erased_account WHERE account_id = $1` → claim the inbox (where one
exists), ack, no effect, DEBUG log with ids only. Tests: an event for a tombstoned account creates no row; the replay digest excludes it.
Hand-off note for later consumers (practice's judge consumer [m3-08](sprint-m3-08.md), judge's worker [m3-06](sprint-m3-06.md)): same check.

### 6 · topology + tests [X]

Sources: [ADR-0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first); mi-05's `topology.go` and ACL renderer.
- `internal/platform/events/topology.go`: the 4 `*-erase` durables on `XLEARN_IDENTITY` and the 4 `identity-erase-acks` durables;
  `Emits` gains `account_erasure_requested` for `XLEARN_IDENTITY` (declared now, produced in l-02) and `account_erased` for the 4 service streams.
- **Subject registry:** every existing durable whose filter matches `xlearn.<svc>.account_erased` (e.g. review's and assessment's
  `xlearn.practice.*`, assessment's `xlearn.review.*`) lists it in `Ignores`; the new durables list their one subject in `Handles`.
- **Account-scoped flag:** m1-02 records a course-scoped flag per subject in `topology.go`, and a v2 envelope on a course-scoped subject
  with an empty `path_slug` is dead-lettered. Register all four `xlearn.<svc>.account_erased` subjects (and
  `xlearn.identity.account_erasure_requested`) as **account-scoped (course-scoped = false)** — the `practice.*` / `review.*` prefix
  rule must not apply to them. Also make sure a subject in a durable's `Ignores` is **acked before any `path_slug` check** (fix the
  decode order in the review/assessment consumers if it isn't). **Consumer test:** review's durable on `XLEARN_PRACTICE` and assessment's
  durables on `XLEARN_PRACTICE` and `XLEARN_REVIEW` receive the `account_erased.v2.json` fixture (no `path_slug`) → acked, no error, no
  dead-letter row. A registry test pins the flag for the five erase subjects.
- **Golden ACL** re-rendered (`go test ./internal/platform/events -run Golden -update`, `make nats-acl-render`): coach's user (publish
  `xlearn.coach.>`, own-stream `XLEARN_COACH`, consumer grants for `coach-erase`, subscribe `_INBOX_coach.>`); consumer grants for the
  three other `*-erase` durables and identity's four ack durables. Budget test unchanged (`XLEARN_COACH` was already budgeted: Σ 3.375 GiB).
- **NATS-auth integration test** (`make nats-acl-test`, NATS 2.14): coach ensures `XLEARN_COACH`, binds `coach-erase`, fetches, acks,
  publishes its ack; identity binds its four ack durables; **denied**: coach publishing on `xlearn.identity.>`, practice creating a durable
  named `review-erase`, coach updating `XLEARN_IDENTITY`.
- **e2e** `internal/e2e/erase_test.go` (`//go:build e2e`, the core-loop harness: real Postgres, embedded JetStream): boot identity,
  practice, review, assessment and coach in-process; seed one account's activity through their HTTP surfaces (an attempt, a revision
  item and mistake, a scored mock, a coach key and thread); a **test helper** plays the not-yet-built producer (insert `erase_request`
  with the L-E `expected_acks`, delete the account, write the outbox row from the fixture); assert zero rows in all four services, four
  acks, `closed_at` set, the ack rows' `account_id` NULL; replay the same event → no change; a **forged** event (unknown request id) →
  nothing deleted, a dead-letter row (`err_class=erase_unverified`) written on the **first** delivery (no redelivery), no ack.
- Docs: [`docs/architecture/events.md`](../../architecture/events.md) (the two subjects, their durables), [`services.md`](../../architecture/services.md)
  (identity's `/internal/erasures/{id}`; coach is a NATS client; the new in-namespace callers), [`data-model.md`](../../architecture/data-model.md) (the new tables).

### 7 · Infra PR(s): ACL + coach seed, before the xlearn merge [I]

Sources: [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (standing rule: the ACL PR merges before the consuming tag), [mi-06](sprint-mi-06.md) tasks 1 and 4 (key generation, the N2 values shape), [mi-05](sprint-mi-05.md) task 2 (`LEGACY=allow|deny|none`).

**Order (so no peer tag can ship this code into a cluster that can't run it).** Task 2 adds a fail-closed start guard (`NATS_URL` set
without `IDENTITY_BASE_URL` → refuse to start), and the new `*-erase` / `identity-erase-acks` durables need their ACL grants. If the
xlearn PR were squash-merged first, any peer tag cut before these PRs (a v1.10.x patch, a parallel M3 sprint) would deploy practice and
assessment into a crash-loop and get every new durable denied. So: the xlearn PR is open and CI-green but **not merged**; both infra PRs
(each additive and harmless on the running 1.10.x images) merge first, each verified before the next; then the xlearn PR is
squash-merged and tagged back to back (task 8).
1. **ACL re-render** — `infrastructure/messaging/release.yaml`: paste the re-rendered `authorization` block, adding coach's user with
   its public key and the new durable grants. Coach's key: `nk -gen user > <scratch>/coach.nk` (mode 600, never printed),
   `nk -inkey … -pubout` → the public key; the seed goes straight into task 7.2's SOPS file; delete the plaintext.
   - **Render from the xlearn PR's head commit** (CI-green, final): `make nats-acl-render NKEYS=<pubkeys.env> LEGACY=deny FORMAT=yaml`.
     `LEGACY` is **the stage recorded in status.md's NATS rows** — `deny` at N3 (production's stage at l-01; [mi-11](sprint-mi-11.md)
     applies N4 later), `none` only if N4 is already recorded. **Never `allow`**: that would silently undo N3 and re-admit anonymous
     publishing, the control the forged-event threat model depends on. `<pubkeys.env>` holds public keys only: practice, review,
     assessment, identity and ops copied from the live `release.yaml`, plus coach's new one; judge stays out ([m3-07](sprint-m3-07.md) adds it).
   - **Diff check before the PR:** diff the rendered block against the live `release.yaml` block. The only allowed changes are **coach's
     user** (publish `xlearn.coach.>`, own-stream `XLEARN_COACH`, the `coach-erase` grants, subscribe `_INBOX_coach.>`), the **new
     consumer grants** (`practice-erase`, `review-erase`, `assessment-erase` on `XLEARN_IDENTITY`; identity's four `identity-erase-acks`),
     and the regenerated `legacy` placeholder password. The `legacy` user must still read **`deny ">"`** (or be absent at N4) with
     `no_auth_user: legacy` unchanged. Anything else means another sprint's ACL change is unmerged, or the wrong stage: stop and explain.
     Paste the diff summary into the PR body.
   - If the xlearn PR's `topology.go` changes after the render (a rebase), re-render and update this PR before the xlearn merge.
   - A ConfigMap change → **reload, no restart**. Verify: `host-verify --cluster --nats-stage=n3` green (zero `legacy` connections in
     `/connz?auth=true`); the reloaded `nats-config` ConfigMap (`k3s kubectl get cm -n messaging -o yaml`, read-only) still has `legacy`
     with `deny ">"`; no permission-violation ERROR in any `xlearn-*` log for 15 min; outbox unsent = 0 everywhere.
2. **coach seed + env** — `apps/secrets/xlearn-nats-coach.enc.yaml` (SOPS, `stringData.seed`); `apps/xlearn-coach.yaml`: `NATS_URL`,
   `NATS_NKEY_SEED_FILE=/var/run/secrets/nats/seed`, `NATS_INBOX_PREFIX=_INBOX_coach`, `IDENTITY_BASE_URL`, and the `extraVolumes` /
   `extraVolumeMounts` seed mount exactly as mi-06 task 4; `apps/xlearn-practice.yaml` and `apps/xlearn-assessment.yaml`: `IDENTITY_BASE_URL`
   (`http://xlearn-identity.xlearn.svc.cluster.local:8081`). The running 1.10.x images ignore these env vars (coach has no events client
   yet), so each Deployment rolls once, harmlessly. Verify: every `xlearn-*` pod Ready on its 1.10.x image; the env is present
   (`k3s kubectl get deploy -n xlearn -o yaml`, read-only).
3. **Then** re-confirm the xlearn PR's golden still equals what PR 1 pasted, squash-merge it, and go straight to task 8. If the tag
   can't follow at once, record in status.md "l-01 merged, untagged — infra PRs already merged; the next tag ships the erase consumers"
   so peers know what their tag carries.

**NetworkPolicy standing rule** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)): the new
in-cluster callers are coach → NATS 4222 and practice/assessment/coach → identity 8081. Read the live policies
(`k3s kubectl get networkpolicy -n messaging -o yaml`, `-n xlearn -o yaml`): MI-5 forward-declared coach on 4222 and MI-5a admits the
whole `xlearn` namespace on identity, and there is no `xlearn` egress policy yet ([mi-11](sprint-mi-11.md) comes after this sprint). If both
hold, record "no NetworkPolicy PR needed" with that evidence; if either is narrower, open that PR first and merge it before the xlearn merge.

### 8 · Tag v1.11.0 [X]

Right after the xlearn squash-merge (task 7 step 3; the infra PRs are already merged), run the **release checklist** (see *Release*).
GitHub release title **`v1.11.0 — v2 build · L-E consumers`**. Release notes: every erase
consumer bound and idle (practice, review, assessment, coach); identity's ack path and `identity admin erasures`; coach on NATS; no
user-visible change. **Floor after this tag: unchanged (1.9.0)** — expand-only and no producer; it rises to 1.11.0 with v1.12.0.

### 9 · Post-tag verify on prod + record [H + X]

By looking (D34), read-only over `ssh vps`:
- `healthz` reports 1.11.0; `k3s kubectl get deploy -n xlearn -o wide` shows the new images; ImagePolicies' latest = the tag; HelmReleases Ready.
- JetStream (via `host-verify --cluster` or the API-server proxy to `:8222`): `XLEARN_COACH` exists; `XLEARN_IDENTITY` shows the 4
  `*-erase` durables; each service stream shows `identity-erase-acks`; pending 0.
- `/connz?auth=true`: coach's connections use **coach's public key**; zero `legacy`; no permission-violation ERROR in coach, identity,
  practice, review or assessment logs.
- `host-verify --cluster --nats-stage=n3` green; memory sum inside the rule; coach's working set vs its 128 Mi limit (`k3s kubectl top pod -n xlearn`):
  if it sits above ~75%, open a separate infra PR raising it, checked against the memory sum ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- Smoke: login, the dashboard and coach (a chat streams).
- Record in status.md (see the prompt's *Update status*): the tag → floor row, the NATS rows (coach N2 = this PR, the ACL PR #), the new
  callers for mi-11's egress matrix, and the Decisions-log lines.

## Acceptance criteria

- [ ] A synthetic erase event (e2e harness over real JetStream + Postgres) removes the account's rows in all 4 services **including their outbox rows**, each service acks, and identity closes the request with 4 acks; a replay changes nothing.
- [ ] A forged event (unknown request, or account still present, or mismatched account) deletes nothing and leaves a dead-letter row.
- [ ] The coverage test pins every `account_id` table in each schema to the erase list; the subject-registry, golden and budget tests are green; the NATS-auth integration test covers coach and the ack subjects.
- [ ] Async consumers and the projection replay skip tombstoned accounts (tests).
- [ ] N3 ≥ 24 h re-check recorded; the ACL PR (rendered with the live `LEGACY` stage, diff-checked) and coach's seed/env PR merged before the xlearn squash-merge; the NetworkPolicy check recorded.
- [ ] **coach connects with its nkey on prod**; v1.11.0 is live and verified per the checklist; status.md updated.

## Release

**Tag `v1.11.0`** (indicative: the next free minor, major = `.release-line` = 1). **Consumers first:** the producer ships in
[l-02](sprint-l-02.md)'s v1.12.0 ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline), [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)).

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule):
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

For this tag:
- **ACL line applies** (task 7.1), plus coach's seed PR (task 7.2).
- "A new service's image" is **n/a** (no new service; coach is an existing image gaining a client).
- Contract line **n/a** (expand only). Snapshot line **n/a**: v1.11.0 is not the erase tag — nothing can trigger an erase until
  v1.12.0 — but `host-verify --cluster` runs anyway (tasks 1, 7, 9). M6 line n/a.
- NetworkPolicy line: verified per task 7 (forward-declared by MI-5/MI-5a; no egress policy yet) or its PR merged first.
- Flags: none added. Gate state after: erase consumers bound and idle; no erase path exists.

**Rollback** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step)): R-c (revert + patch tag) is
the default. R-b may narrow to 1.10.x (no producer exists yet); coach then simply stops connecting. The ACL PR reverts as a reload. No
data changes until v1.12.0.

## Definition of Done

CI green (`go test ./...`, `sqlc diff`, the contract-header lint, golden/budget/registry tests, `nats-acl`, `e2e`) · the two infra PRs
merged before the xlearn squash-merge (GitOps; no hand `kubectl apply`) · v1.11.0 cut, deployed by Flux and verified by looking · acceptance criteria met ·
statuses updated (this file + [`../status.md`](../status.md): Sprint board, L row, tag → floor, NATS rows, Decisions log) · local `main`
synced in xlearn and `../infra`.

## Risks / watch-outs

- **A consumer acting without re-verification could be tricked by a forged event.** N3 closes anonymous publishing and the fine ACL
  lets only identity publish `xlearn.identity.>`; the re-verify (request exists, account gone, ids match) closes the rest. Never
  shortcut the verifier in tests of the production handler.
- **A terminal dead letter is never redelivered.** `events.Terminal` is for genuinely forged or malformed events only; a transient
  or misconfiguration failure (unreachable identity, 5xx, a non-identity 404) must stay a plain error. A real erase that ends in a
  dead letter (terminal, or after mi-05's 100 deliveries) shows as a missing ack in `identity admin erasures --open`, and today no
  verb re-emits it: say so in the Decisions log for [l-02](sprint-l-02.md)'s runbook.
- **The ack deleted by its own erase.** An ack outbox row stamped with the account id would be deleted by a redelivered `EraseTx`
  before the relay sends it. Keep `outbox.account_id = NULL` on acks (and on identity's `account_erasure_requested` row in l-02); test it.
- **A new per-user table without an erase path.** The `information_schema` coverage test is the guard; M3 and M4 sprints that add
  tables must extend the erase list in the same PR.
- **Subject or payload drift between consumer (here) and producer (l-02).** The constants and the fixture are shared; l-02 asserts
  byte equality. An unknown subject is acked silently ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
- **Existing durables start receiving `account_erased`.** List it in their `Ignores`; the registry test and the consumers' default
  branch ERROR log catch a miss.
- **coach at 128 Mi.** A NATS client and a relay goroutine add a few MiB; measure after the tag (task 9). MI-16 raises coach later.
- **Egress matrix drift.** [mi-11](sprint-mi-11.md)'s xlearn egress must admit practice, assessment and coach → identity :8081 and coach →
  NATS :4222, or the erase path breaks silently when egress lands. Recorded in status.md for it; judge's re-verify at M3-1 needs
  identity :8081 in [m3-07](sprint-m3-07.md)'s default-deny egress too.
- **Migration numbering races** with M3 sprints running in parallel: take the next free goose version per service at rebase; CI fails on a duplicate.
- **Pseudonymous copies stay in JetStream** (ADR-0027 §6 note). identity's historic `account_created` events carry `display_name`;
  l-02 records this in the erase runbook with the re-seal option (ops break-glass). Don't change the payload (the envelope is append-only).
