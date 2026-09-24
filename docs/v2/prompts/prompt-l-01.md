# Prompt — Sprint l-01 · L-E erase consumers (practice, review, assessment, coach) + identity ack path → v1.11.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-l-01.md`](../sprints/sprint-l-01.md)   ·   **Milestone:** L (L-E, consumer half)   ·   **Prereqs:** [mi-06](../sprints/sprint-mi-06.md) (N3 ≥ 24 h ago), [mi-03](../sprints/sprint-mi-03.md) (MI-5, MI-5a), [m2-05](../sprints/sprint-m2-05.md) (v1.10.0 live)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and the land-and-sync rule. This sprint **tags**.
- The plan: [`../sprints/sprint-l-01.md`](../sprints/sprint-l-01.md). Its tasks, handler outcome table, task 7's merge order, release checklist and rollback are authoritative.
- [`../status.md`](../status.md): the NATS rows (the N3 timestamp and the current `LEGACY` stage), the tag → floor rows, the Decisions log (m1-07's coach erase hand-off).
- **Erase design:**
  - [t1 §6.6](../research/t1-content-data-model.md#66-erase-path-none-exists-today-no-delete-apime-in-bffgo) (the flow; read "erase ledger" and "ops check" as dropped: D12, D34);
  - [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20) as amended by [ADR-0028 §3](../../adr/0028-object-storage-and-backups.md#3-amendments-to-adr-0027) (no ledger: delete + outbox delete + tombstones);
  - [ADR-0033 §9](../../adr/0033-invite-only-admission-and-owner-admin.md#9-erase-and-abuse-controls) (acks from `topology.go` of the running version; 4 at L-E; gates N3 + MI-5, plus MI-5a) and §12 row 12 (consumers re-verify via `/internal/erasures/{id}`).
- **NATS:** [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes) (topology table: `XLEARN_IDENTITY` consumed by practice, review, assessment, judge, coach; identity consumes every service stream; `XLEARN_COACH`), [§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (fine ACLs, N3 ≥ 24 h, "ACL PR before the consuming tag", MI-5 forward declarations), [§3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34) (no alerting; durable state is a Postgres row), [§5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses) (memory sum).
- **Releases:** [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (consumers before producers; expand rules), [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step) (L-E row), [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist); [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag), [§7](../rollout-plan.md#7-indicative-tag-timeline) (v1.11.0 → v1.12.0).
- **Sibling plans:** [mi-05](../sprints/sprint-mi-05.md) (the `topology.go` types, `Handles`/`Ignores`, the ACL renderer, the dead-letter hook and its table shape, `events.Dial`, fail-closed init), [mi-06](../sprints/sprint-mi-06.md) tasks 1 and 4 (nkey generation, the seed secret and values shape), [m1-02](../sprints/sprint-m1-02.md) (`outbox.account_id` in practice, review, assessment; the course-scoped flag per subject; coach's `key_default`), [m1-07](../sprints/sprint-m1-07.md) task 3 (coach's `message_quota_day`, handed to you), [m1-04](../sprints/sprint-m1-04.md) task 5 (`identity admin` CLI layout), [m2-05](../sprints/sprint-m2-05.md) task 4 (the projection replay and its advisory lock), [l-02](../sprints/sprint-l-02.md) (the producer that consumes your contract).
- **Code:**
  - `internal/platform/events/{topology.go,acl.go,consumer.go,nats.go,events.go}`, `testdata/nats-authorization.golden.*`; `Makefile` (`nats-acl-render`, `nats-acl-test`);
  - `internal/{practice,review,assessment,coach,identity}/store/{migrations,queries}`, `sqlc.yaml`;
  - `internal/review/consumers.go`, `internal/assessment/consumers.go`, `internal/{practice,assessment,coach}/config.go`, `internal/review/config.go` (`IdentityBaseURL` precedent);
  - `internal/identity/service.go` (the `/internal/*` routes, `NewOutboxRelay`), `internal/identity/admin/`;
  - `cmd/{practice,review,assessment,coach,identity}/main.go`; `docker-compose.yml`; `internal/e2e/coreloop_test.go` (the harness to copy).
- **Infra (read, then PR):** `../infra/infrastructure/messaging/release.yaml` (the `authorization` block), `../infra/apps/xlearn-{coach,practice,assessment}.yaml`, `../infra/apps/secrets/`, `../infra/.sops.yaml`, `../infra/charts/project/values.yaml` (`extraVolumes`, `extraVolumeMounts`), `../infra/hack/host-verify.sh` (`--cluster --nats-stage=n3`).

## Context

- Nothing in xLearn can erase an account today (no `DELETE /api/me`, no CLI verb). L-E builds it in two tags so that **consumers ship
  one tag before the producer**:
  - **v1.11.0 (this sprint):** practice, review, assessment and coach bind their erase consumers; identity gains the request/ack tables,
    the re-verify endpoint and its ack durables; coach makes its first NATS connection with its own nkey.
  - **v1.12.0 ([l-02](../sprints/sprint-l-02.md)):** identity's erase transaction, `DELETE /api/me` for testers, the CLI verb and the UI.
- NATS is already server-first authenticated (N1–N3, [mi-06](../sprints/sprint-mi-06.md)); coach and judge were left out of N2 on purpose —
  coach joins here, judge at M3-1 ([m3-05](../sprints/sprint-m3-05.md)/[m3-07](../sprints/sprint-m3-07.md), reusing your `internal/platform/erase`).
- There are no backups (D12) and no alerting (D34): a wrong delete is permanent, and a stuck erase is found only by looking
  (`identity admin erasures --open`). That is why every consumer **re-verifies** before deleting and why this tag deletes nothing.
- The floor after v1.11.0 stays **1.9.0**; it rises to 1.11.0 with v1.12.0.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **N3 applied ≥ 24 h ago:** status.md's NATS rows carry the N3 timestamp, and now − that ≥ 24 h.
- [ ] **MI-5 live:** `ssh vps 'k3s kubectl get networkpolicy -n messaging -o yaml'` admits 4222 from `xlearn-coach` (forward-declared) as well as practice, review, assessment, identity.
- [ ] **MI-5a live:** `ssh vps 'k3s kubectl get networkpolicy -n xlearn -o yaml'` — identity admits same-namespace pods on :8081.
- [ ] **v1.10.0 live:** `ssh vps 'k3s kubectl get deploy -n xlearn -o wide'` ≥ 1.10.0, and healthz agrees.
- [ ] **identity on NATS with its nkey:** `XLEARN_IDENTITY` exists (JetStream info via `host-verify --cluster` or the API-server proxy) and identity's outbox has 0 unsent.
- [ ] **Parallel sessions:** `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents → the **next free minor** (use it instead of 1.11.0 everywhere if taken); no open peer PR adds goose migrations to identity, practice, review, assessment or coach, or edits `topology.go` or `consumer.go` (if one does, rebase after it and take the next free versions).

## Do this (in order)

1. **[H] N3 ≥ 24 h re-check.** `ssh vps 'bash -s -- --cluster --nats-stage=n3' < ../infra/hack/host-verify.sh` → no `legacy` connection,
   all checks green. Note the result for status.md. A `legacy` connection → stop and report.

2. **[X] Branch** `feat/l-01-erase-consumers` from an up-to-date `main`.

3. **[X] The contract + `internal/platform/erase`.**
   - `internal/platform/events`: `SubjectAccountErasureRequested = "xlearn.identity.account_erasure_requested"` (data `{erase_request_id, account_id, requested_at}`)
     and `SubjectAccountErased(svc)` (data `{erase_request_id, account_id, service}`), v2 envelopes (account-scoped), fixtures
     `testdata/account_erasure_requested.v2.json` and `account_erased.v2.json`.
   - **Terminal-error path** in `internal/platform/events` (`consumer.go`): `TerminalError{Class, Err}` + `Terminal(class, err)`; `dispatch`
     finds it with `errors.As` on **any** delivery → the `WithDeadLetter` sink (class passed through) → `msg.Term()` → ERROR log with ids
     only, never a Nak; `ErrClass` returns the terminal class. Plain errors keep mi-05's path (Nak with backoff; last delivery → sink +
     Term). Tests in `consumer_test.go` with mi-05's fake `jetstream.Msg`: terminal on delivery 1 → row + Term, no Nak; wrapped with
     `%w` → same; sink failure → still Term; plain error on delivery 1 → Nak (regression). A handler can't Term by itself
     (`Handle(ctx, Event) error`), and without this a forged event would be retried ~100 times (about 8 h).
   - `internal/platform/erase`: `Request`, `Verifier` (`NewHTTPVerifier`: `GET {IDENTITY_BASE_URL}/internal/erasures/{id}`, 5 s timeout),
     `Store.EraseTx`, `NewHandler(svc, …)` with exactly the plan's outcome table: decode error → `events.Terminal("decode", …)`;
     identity's JSON 404 / account present / ids differ → **no delete**, `events.Terminal("erase_unverified", …)`; transport, 5xx or a
     404 whose body isn't identity's JSON `error.code = "not_found"` → plain error (nak); `EraseTx` error → plain error (nak). The handler writes no
     dead-letter row itself — the subscription's `WithDeadLetter` sink does. `AckEventID(svc, id) = uuidv5(ns, "account_erased:"+svc+":"+id)`.
     Unit tests for every outcome (assert the `*events.TerminalError` class, or a plain error).

4. **[X] Migrations + stores (practice, review, assessment).** Next free goose version per service, expand-only:
   `<svc>.erased_account (account_id uuid PRIMARY KEY, erase_request_id uuid NOT NULL, erased_at timestamptz NOT NULL DEFAULT now())`;
   practice also gets `event_dead_letter` (mi-05's shape). sqlc queries in `store/queries/erase.sql`; `sqlc generate`; commit.
   Implement `EraseTx` per the plan: tombstone `ON CONFLICT DO NOTHING` → `DELETE … WHERE account_id = $1` on every per-user table
   (cascades cover children) and on `outbox` → insert the ack outbox row with the deterministic id, `ON CONFLICT (event_id) DO NOTHING`,
   **`account_id = NULL`**. assessment takes the projection replay lock shared inside `EraseTx`. Add the **coverage test**
   (`information_schema.columns` → every `account_id` table is in the erase list or exempt) and the two-account store test.

5. **[X] Wire the consumers.** In `cmd/{practice,review,assessment}/main.go`: `Subscribe` the durables `practice-erase`, `review-erase`,
   `assessment-erase` on `XLEARN_IDENTITY`, filter `xlearn.identity.account_erasure_requested`, `DeliverAll`, `WithDeadLetter(store)`.
   practice and assessment config gain `IDENTITY_BASE_URL`; **refuse to start when `NATS_URL` is set and `IDENTITY_BASE_URL` is not.**
   docker-compose sets it on all four services.

6. **[X] coach on NATS.** Migration: `coach.outbox` (with `account_id uuid NULL` and the unsent index), `coach.erased_account`,
   `coach.event_dead_letter`. Config: `NATS_URL`, `IDENTITY_BASE_URL` (+ the N0 client env via `events.Dial`); update the package comment.
   `cmd/coach/main.go`: publisher on `XLEARN_COACH` from the table (fail-closed when `NATS_URL` is set), relay goroutine, `coach-erase`
   consumer, clean shutdown. Coach's `EraseTx`: `api_key_config` (→ `key_default` by cascade; in the coverage test's cascade allow-list),
   `coach_thread` (→ `coach_message`), `message_quota_day`, `outbox`. docker-compose: coach gets
   `NATS_URL`, `IDENTITY_BASE_URL`, `depends_on: nats`.

7. **[X] identity side.**
   - Migration (next free identity version): `erase_request` (+ the one-open partial unique index), `erase_ack`, `released_username`,
     `event_dead_letter`, and `outbox.account_id` if M1a didn't add it (backfill from `payload_json->>'account_id'`; the writer sets it).
   - `GET /internal/erasures/{id}` in `service.go` → `{id, account_id, requested_at, account_gone, state}` or 404 `not_found`; no PII.
   - `events.ErasureAckServices()` from `topology.go` (a test pins `{assessment, coach, practice, review}`).
   - Ack consumer `identity-erase-acks` on `XLEARN_PRACTICE`, `XLEARN_REVIEW`, `XLEARN_ASSESSMENT`, `XLEARN_COACH`: record the ack
     idempotently, close the request when every `expected_acks` service has acked; an unknown request →
     `events.Terminal("erase_unknown_request", …)`, undecodable → `events.Terminal("decode", …)` (dead letter + Term on that delivery);
     tolerate unexpected services; `WithDeadLetter(<identity store>)`. Wire it in `cmd/identity/main.go`.
   - `identity admin erasures [--open] [--since <RFC3339>]` in `internal/identity/admin/` (read verb; writes its `admin_audit` row).
   - **Do not** add any route or verb that creates an erase request: that is l-02.
   - Tests: tables, the one-open index, the close rule, duplicates, unknown request, the endpoint, the CLI verb.

8. **[X] Tombstone checks.** In review's consumers (`XLEARN_PRACTICE` subjects incl. `touch_concluded`, review/notifications),
   assessment's consumers and `assessment admin replay-projections`: skip a tombstoned account inside the transaction (claim the inbox
   where one exists, ack, no effect). Workers that get a 404 from identity skip the account. Tests per path; the replay digest excludes erased accounts.

9. **[X] topology + tests.**
   - `topology.go`: the 4 `*-erase` durables and the 4 `identity-erase-acks` durables; `Emits` for `account_erasure_requested` and each `account_erased`;
     every existing durable matching `xlearn.<svc>.account_erased` lists it in `Ignores`.
   - Register the four `xlearn.<svc>.account_erased` subjects and `xlearn.identity.account_erasure_requested` as **account-scoped**
     (course-scoped = false) in the subject registry — m1-02 otherwise dead-letters a `practice.*`/`review.*` v2 envelope with no
     `path_slug`. `Ignores` subjects are acked before any `path_slug` check. Test: review's and assessment's existing durables receive the
     `account_erased.v2.json` fixture → acked, no error, no dead-letter row.
   - Re-render the golden ACL (`go test ./internal/platform/events -run Golden -update`; `make nats-acl-render`) with coach's user and the new grants.
   - Extend the NATS-auth integration test (`make nats-acl-test`, NATS 2.14): coach's stream, `coach-erase`, coach's ack publish; identity's
     four ack durables; **denied**: coach → `xlearn.identity.>`, practice creating `review-erase`, coach updating `XLEARN_IDENTITY`.
   - `internal/e2e/erase_test.go` (`//go:build e2e`): the plan's scenario — seed via HTTP, a test helper plays the producer, assert zero rows
     in 4 services, 4 acks, `closed_at` set, ack rows' `account_id` NULL, idempotent replay, and the forged-event path.
   - Docs: `docs/architecture/events.md`, `services.md`, `data-model.md` per the plan.

10. **[X] PR, CI green — do not merge yet.** One PR (or a short serialized series): conventional commits with the attribution lines;
    CI green (`go test ./...`, `sqlc diff`, the contract-header lint, golden/budget/registry, `nats-acl`, `e2e`, web tests). **Hold the
    squash-merge until both infra PRs are merged** (the plan's task 7 order): the new start guard (`NATS_URL` set without
    `IDENTITY_BASE_URL` → refuse to start) and the new durables would crash-loop or be denied on prod if a peer tag shipped this code
    before the env and the ACL land.

11. **[I] Infra PR 1 — ACL re-render.** Generate coach's nkey on your machine (never the VPS or CI): `nk -gen user > <scratch>/coach.nk`
    (mode 600, never printed), `nk -inkey <scratch>/coach.nk -pubout` → public key. Build `<pubkeys.env>` (public keys only): practice,
    review, assessment, identity and ops copied from the live `../infra/infrastructure/messaging/release.yaml`, plus coach; judge stays
    out (m3-07 adds it). From the **xlearn PR's head commit** run
    `make nats-acl-render NKEYS=<pubkeys.env> LEGACY=deny FORMAT=yaml` — `LEGACY` is **the stage recorded in status.md's NATS rows**
    (`deny` at N3; `none` only if N4 is already recorded; **never `allow`**, which would undo N3 and re-admit anonymous publishing).
    **Diff** the rendered block against the live one: the only allowed changes are coach's user, the new consumer grants
    (`practice-erase`, `review-erase`, `assessment-erase`, identity's four `identity-erase-acks`) and the regenerated `legacy` placeholder
    password; `legacy` must still read `deny ">"` with `no_auth_user: legacy`. Anything else → stop and explain. Paste the block into
    `release.yaml` (same `config.merge` shape mi-06 used; don't hand-edit subjects) and the diff summary into the PR body. PR → merge →
    **reload** (no restart). Verify: `host-verify --cluster --nats-stage=n3` green; the reloaded `nats-config` ConfigMap still has
    `legacy` with `deny ">"`; no permission-violation ERROR in any `xlearn-*` log for 15 min; every service's outbox unsent = 0.

12. **[I] Infra PR 2 — coach seed + env.** `../infra/apps/secrets/xlearn-nats-coach.enc.yaml` (`stringData.seed`, encrypted in place with
    `sops`; delete `<scratch>/coach.nk`). `apps/xlearn-coach.yaml`: `NATS_URL`, `NATS_NKEY_SEED_FILE=/var/run/secrets/nats/seed`,
    `NATS_INBOX_PREFIX=_INBOX_coach`, `IDENTITY_BASE_URL`, and the seed `extraVolumes`/`extraVolumeMounts` exactly as mi-06 task 4.
    `apps/xlearn-{practice,assessment}.yaml`: `IDENTITY_BASE_URL=http://xlearn-identity.xlearn.svc.cluster.local:8081`. PR → merge →
    each Deployment rolls once on its current 1.10.x image (harmless; verify all pods Ready and the env present). **NetworkPolicy check:**
    confirm from the live policies that coach is admitted to NATS 4222 and identity admits the namespace on 8081, and that no `xlearn`
    egress policy exists; record "no NetworkPolicy PR needed" with that evidence — or open and merge that PR first.

13. **[X] Merge, then tag at once.** If the PR's `topology.go` changed since step 11 (a rebase), re-render and update infra PR 1 first.
    Squash-merge the xlearn PR, then immediately run the plan's *Release* checklist (re-check peers' tags right before the push). Tag
    the merge commit; GitHub release title **`v1.11.0 — v2 build · L-E consumers`**; notes per the plan. If the tag can't follow the
    merge at once, record in status.md "l-01 merged, untagged — infra PRs already merged; the next tag ships the erase consumers".

14. **[H] Post-tag verify** (read-only): healthz 1.11.0; images; ImagePolicy latest = tag; HelmReleases Ready; `XLEARN_COACH` exists;
    the 4 `*-erase` durables on `XLEARN_IDENTITY` and `identity-erase-acks` on each service stream, pending 0; `/connz?auth=true` shows
    coach on **its** public key and zero `legacy`; no permission-violation ERROR; `host-verify --cluster --nats-stage=n3` green; coach's
    working set vs 128 Mi (a raise, if needed, is its own infra PR checked against the memory sum); smoke login, dashboard, a coach chat.

15. **[X] Record** — see *Update status*.

## Constraints

- **Consumers before producers** ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)). No path that creates an erase request ships in this tag: no route, no CLI verb, no test-only endpoint in a prod binary. The e2e helper lives in the test package.
- **Re-verify before any delete.** No consumer deletes on the event alone; the verifier is never stubbed in production wiring.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).** Each `EraseTx` touches only its own schema; identity's re-verify is an HTTP call to `/internal/erasures/{id}`, never a cross-schema read.
- **Outbox / inbox.** Tombstone, deletes and the ack row in **one** transaction; deterministic ack ids; ack rows carry `outbox.account_id = NULL`. Never trim outboxes otherwise.
- **goose + sqlc.** Expand-only migrations (new tables, nullable columns); next free version per service; embedded, run on startup under the advisory lock; generated code committed; `sqlc diff` clean. **Never run `Down` in prod.**
- **NATS.** The ACL PR merges **before** the tag; seeds live only in `apps/secrets` (SOPS); public keys in plaintext in `messaging`; no SOPS decryption in `messaging`; never `STREAM.DELETE`/`PURGE`; the ops seed stays offline and isn't needed here.
- **GitOps:** never `kubectl apply`; never hand-edit a HelmRelease; never move or re-push a tag; don't suspend the shared IUA. Infra PRs are their own tasks, never folded into the tag.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod; coach keeps its 128 Mi limit unless the post-tag measurement says otherwise (then a separate, checked infra PR).
- **No alerting (D34).** Stuck erases are read on demand (`identity admin erasures --open`); no ops check, no ping, no Flux Alert.
- **Logs carry ids only** on every erase path (event id, erase request id, account id, durable) — never names, emails or content.
- **Parallel sessions:** check peers' tags, PRs and worktrees (+ ListAgents) before claiming migration versions or an ADR number, and again right before the tag push.

## Deliverables

- `events.Terminal` + the terminal branch in `consumer.go` (with its unit tests); `internal/platform/erase` + the subject constants and fixtures; the four service consumers with `EraseTx`, tombstones and coverage tests.
- coach's outbox, relay, `XLEARN_COACH` publisher and `coach-erase` consumer.
- identity's `erase_request`, `erase_ack`, `released_username`, `event_dead_letter` (+ `outbox.account_id` if missing), `GET /internal/erasures/{id}`, the ack durables and close rule, `identity admin erasures`.
- Tombstone checks in review, assessment and the projection replay.
- `topology.go`, the re-rendered golden ACL, the extended `nats-acl` test, `internal/e2e/erase_test.go`, updated architecture docs.
- `../infra`: the ACL PR and coach's seed/env PR (+ `IDENTITY_BASE_URL` for practice and assessment), both merged before the xlearn squash-merge.
- Tag **v1.11.0**, verified on prod.

## Update status

- [`../sprints/sprint-l-01.md`](../sprints/sprint-l-01.md): task rows ✅ as they land (task 7 with both PR numbers and the NetworkPolicy evidence); _Overall_ ✅.
- [`../status.md`](../status.md):
  - **Sprint board** l-01 ✅; **Milestones** L: "L-E consumers live (v1.11.0)" (L stays 🔄);
  - **Tag → floor → snapshot:** v1.11.0 → floor unchanged (1.9.0) → no snapshot (not required: no erase path yet);
  - **NATS rows:** "N3 ≥ 24 h re-check ✅ <date>" (m3-07 and mi-11 gate on it); coach N2 via infra PR #…; ACL re-render PR #…;
  - **Callers added** (for [mi-11](../sprints/sprint-mi-11.md)'s egress matrix): practice, assessment, coach → identity :8081; coach → NATS :4222;
  - **Flag inventory:** none added;
  - **Decisions log:** the shared `internal/platform/erase` handler; forged/unverified events dead-lettered on the first delivery via `events.Terminal` (never retried; m3-05's judge consumer uses the same path); a dead-lettered real erase has no re-emit verb yet (for l-02's runbook); ack rows with `outbox.account_id = NULL`; `expected_acks` snapshotted per request; `IDENTITY_BASE_URL` required when NATS is on; `identity admin erasures` replaces the dropped "open > 24 h" check (D34); coach's ack over NATS (ADR-0035 §1).
- Note on the board that [l-02](../sprints/sprint-l-02.md) is unblocked (with ds-l-01's freeze and the first tester).
- An ADR only if you depart from ADR-0027/0033/0035 (check peers before numbering).

## Done when (acceptance)

- [ ] A synthetic erase event removes the account's rows (incl. outbox rows) in all 4 services, each acks, and identity closes the request; a replay changes nothing.
- [ ] A forged or unverifiable event deletes nothing and leaves a dead-letter row on its first delivery (terminal, never retried).
- [ ] Coverage, subject-registry, golden, budget, `nats-acl` and `e2e` tests green; async consumers and the replay skip tombstoned accounts.
- [ ] N3 ≥ 24 h re-check recorded; the ACL (live `LEGACY` stage, diff-checked) and coach-seed/env PRs merged before the xlearn squash-merge; NetworkPolicy check recorded.
- [ ] **coach connects with its nkey on prod**; v1.11.0 live and verified per the checklist; status.md updated.

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: tag v1.11.0** (the next free minor), in order:
1. the xlearn PR → CI green (**not merged yet**);
2. `../infra` PR 1 (ACL, rendered from the PR head with the live `LEGACY` stage, diff-checked) → merge → verify; PR 2 (coach seed + env) → merge → verify;
3. squash-merge the xlearn PR → tag at once → Flux deploys → verify live per the checklist;
4. `git checkout main && git pull` in xlearn and `../infra`.

Never leave an open PR or merged work unpulled.
