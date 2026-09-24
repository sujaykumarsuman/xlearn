# Sprint m2-05 — Producers on + `touch_scored` backfill + reader switch + replay + D2 (M2c) → v1.10.0

> **Milestone:** M2 — attempt engine and projections (M2a/M2b producers + M2c; **milestone exit**)   ·   **Track:** product
> **Prereqs:** [m2-02](sprint-m2-02.md) (v1.9.0: consumers bound), [m2-03](sprint-m2-03.md), [m2-04](sprint-m2-04.md)   ·   **Unblocks:** [l-01](sprint-l-01.md)
> **Release action:** **tag v1.10.0** (indicative: take the next free minor; floor 1.9.0), plus an infra ACL PR **only if** this sprint adds or renames a durable (expected n/a: M2-02 declared and ACL'd `assessment-replay`)   ·   **Calendar:** early November. Owner involvement is possible: if the session's VPS access is read-only, the owner runs the two admin-CLI commands of task 8 via `kubectl exec`.
> **Execute with:** [`../prompts/prompt-m2-05.md`](../prompts/prompt-m2-05.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Producers: practice `touch_concluded`, review `touch_scored` (M2-02's contract); remove M2-04's `touchesEnabled` guard | X | ⬜ |
| 2 | `touch_scored` backfill: `review admin backfill-touch-scored` (idempotent, deterministic ids) | X | ⬜ |
| 3 | D2 (M2c): anchor on every conclusion, `revision_item.anchor_rule`, `REVISION_ENTRY_RULE` kill switch | X | ⬜ |
| 4 | Replay: M2-02's command + replay lock + read-model digest + re-runnable durable; compose rehearsal; runbook | X | ⬜ |
| 5 | Authed readers switch to projections v2 (M2-02's hand-off), in the producers' PR | X | ⬜ |
| 6 | Infra ACL PR, only if a durable was added or renamed (expected n/a), merged before the tag | I | ⬜ |
| 7 | Tag v1.10.0 (release checklist) | X | ⬜ |
| 8 | Post-tag prod steps: backfill → drain → replay ×2 (digest equal) → reconcile → record (O if handed to the owner) | H | ⬜ |
| 9 | M2 exit recorded (milestone, tag → floor, flags, pending contract, decisions) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). If task 8 is handed to the owner, set it ⛔ "waiting on owner: admin-CLI run" and log the hand-off in the
> status.md manual-path log; set it ✅ when the owner's output is recorded. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **v1.9.0 live (consumers bound).** In prod:
  - review's `touch_concluded` consumer is bound;
  - assessment's `touch_scored` handling and the projections v2 are bound and idle (M2-02);
  - `/xlearn/api/v1/healthz` reports ≥ 1.9.0.
- [ ] **M2-03 and M2-04 merged** on `main`.
- [ ] **The subjects match the bound consumers.**
  - `xlearn.practice.touch_concluded` is in review's handler switch and in the subject registry.
  - The `touch_scored` subject v1.9.0's assessment consumer handles is **`xlearn.review.touch_scored`**: review emits it ([t0 §5(e)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist), [ADR-0026 §3](../../adr/0026-per-course-extensibility-model.md#3-the-narrow-waist-practice-is-the-only-writer-of-learning-signals)), and review's `xlearn.review.>` publish ACL covers it.
  - **If M2-02 registered a different name or owner, stop and report.** A producer that doesn't match the bound consumer is acked silently and lost ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).
- [ ] **M2-02's `touch_scored` contract is on `main`**: the shared fixture `internal/platform/events/testdata/touch_scored.v2.json` and the [`events.md`](../../architecture/events.md) "Flow 6 — touch scored → projections" section ([M2-02 task 1](sprint-m2-02.md#1--xlearnreviewtouch_scored--the-assessment-consumer-x)). Task 1's producer marshals to that fixture.
- [ ] **M2-02's v2 read endpoints are on `main`**: assessment `GET /progress/activity`, `/progress/touch-stats` and `/progress/outcome-mix` ([M2-02 task 3](sprint-m2-02.md#3--v2-read-endpoints--drop-problemtotal151-x)). Task 5 switches the readers onto them.
- [ ] **M2-02's replay command is on `main`**: `assessment admin replay-projections --confirm` exists, and the durable `assessment-replay` is declared in `internal/platform/events/topology.go`. If N1 fine ACLs are live on prod, it is also in the merged infra ACL ([M2-02 task 6](sprint-m2-02.md#6--infra-acl-pr-i)). The replay golden test and the v2 [`projection-rebuild` runbook](../../runbooks/projection-rebuild.md) are on `main` too.
- [ ] **The NATS state is known** ([MI-7 via MI-06](sprint-mi-06.md)): whether N1 fine ACLs are live on prod. That decides whether task 6 could ever be needed.
- [ ] **Parallel sessions**: the next free minor is known (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents). If `v1.10.0` is taken, use the next free minor everywhere below.

## Goal

**Close M2.**
- Turn the M2 producers on, one tag after their consumers (v1.9.0): practice emits `touch_concluded`, and review emits `touch_scored`.
- **Switch the authed Progress and Dashboard readers to projections v2** in the same tag as the replay, as M2-02 handed off, so no reader sees partial history.
- **Backfill** `touch_scored` for every existing touch result, then **drop and replay** the projections on production.
- Switch DSA to **"every concluded attempt anchors the ladder" (D2)** behind the permanent kill switch **`REVISION_ENTRY_RULE`**, with `revision_item.anchor_rule` stamped on every row.

The Touch UI (M2-04) and the public slice (M2-03) go live in the same tag. The M2 exit criteria are: replay equal; below-clean items get a ladder; the public route is `public-read`-only.

## Scope

**In**
- practice: the `touch_concluded` outbox insert in M2-01's `concludeTouch()`; `anchor_at`/`revisable` on `problem_solved` v2 if they're still missing ([ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer)).
- review: the `touch_scored` emission on every scored touch, exactly per M2-02's payload; a goose expand migration for `anchor_rule`; D2 in `HandleProblemSolved`; `REVISION_ENTRY_RULE` config; the `review admin backfill-touch-scored` subcommand.
- assessment: on M2-02's `replay-projections`, the replay advisory lock (which [l-01](sprint-l-01.md)'s erase shares), the read-model digest (also `assessment admin projection-digest`) and the re-runnable replay durable; `/progress/summary`'s streak and outcome mix read from the v2 tables.
- gateway and web: the authed Progress reader on the v2 endpoints (`web/src/lib/progress.ts` types, the heatmap, touch stats, provenance chips); delete M2-04's `touchesEnabled` guard and the Revision fallback branch.
- Tag v1.10.0; the post-tag backfill → replay on prod; the M2 exit record.

**Out (later sprints)**
- Judge signals and evaluator-graded touches: M3 ([m3-08](sprint-m3-08.md), [m3-10](sprint-m3-10.md)). The rest of the authed Progress deltas (AB12): [m3-13](sprint-m3-13.md).
- The pending contract that drops `proj_heatmap` and the v1 `proj_outcome_mix` (M2-02 task 5): a later contract tag. This sprint only records its earliest tag.
- The erase consumers and producer: [l-01](sprint-l-01.md) and [l-02](sprint-l-02.md), which start after this tag.
- Honor-probe claims and provisional touches: M4 ([m4-04](sprint-m4-04.md)).
- **Retro-anchoring historic below-clean items** (concluded before v1.10.0 with no ladder) is **not** done. D2 is forward-only: the kill switch stops *new* anchors, and replay determinism assumes anchors come only from events processed under the rule. If the owner wants them, it's a separate one-off (`review admin anchor-orphans`, anchored at `now()` with `anchor_rule='v2'`) in a later sprint. Record this in the Decisions log.

## Tasks

### 1 · Producers [X]

- **practice**: M2-01's guarded `concludeTouch()` already builds the typed `touch_concluded` payload, but doesn't insert it ([M2-01 task 4](sprint-m2-01.md#4--touch-endpoints--practice--review-read-x)). Add **the one outbox insert inside the same transaction**, as a v2 envelope.
  - Data per [t0 §5(b)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist) + [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer), as M2-01 built it: `path_slug, problem_id, revision_item_id, touch_level, attempt_id, touch_passed, resolution, criteria[{key, met, source, at_secs}], recall_secs, graded_by, trust, mock_mode, band, anchor_at, policy_version, evaluation_id: null`.
  - It must match the shared fixture `internal/platform/events/testdata/touch_concluded.v2.json` byte for byte. **Invert** M2-01's `TestNoTouchProducerYet`: exactly one outbox row per conclusion, none for a voided touch.
  - **`event_id = uuidv5(ns, "touch_concluded:" + attempt_id)`**, so a re-emit collides on the outbox PK.
  - Voided touches emit nothing ([t4 §3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts)).
- **practice**: if `problem_solved` v2 doesn't yet carry `anchor_at` and `revisable`, add them. The envelope is append-only.
  - On the self path, `anchor_at` = the conclusion time.
  - `revisable = true` for DSA, since every DSA item is `core` or `reinforcement`. The per-item role check ([I8](../research/t4-judge-contract.md#61-universal-invariants-practice-enforced-property-tested)) must exist before a course with drills goes live (see Risks).
- **review**: every scored touch writes `xlearn.review.touch_scored` **in the same transaction** as its `touch_result` row. Emit it once, in M2-01's shared `applyTouchOutcomeTx`, so both the `touch_concluded` consumer path (`HandleTouchConcluded`) and the v1 self endpoint `POST /revisions/{id}/score` (`graded_by=self`, `trust=honor`) emit it.
  - **The payload is exactly M2-02's contract** ([M2-02 task 1](sprint-m2-02.md#1--xlearnreviewtouch_scored--the-assessment-consumer-x)), in a v2 envelope carrying `path_slug` (from the item). Data:
    - `problem_id`, `revision_item_id`, `touch_level`, `passed`, `graded_by`, `trust`;
    - `attempt_id`: the touch attempt on the consumer path; `null` on the self path and in the backfill;
    - `anchor_at`: the touch's anchor (`touch_concluded.anchor_at`) on the consumer path; `scored_at` on the self path and in the backfill;
    - `mock_mode`: copied from `touch_concluded` on the consumer path; `false` on the self path and in the backfill.
  - There is **no** `level` and **no** `touch_result_id` field. Assessment's bound consumer decodes `touch_level` and `anchor_at`, and a producer emitting other names would key `proj_touch_stats` on a missing level. `touch_result.id` is only the seed of the event id.
  - `occurred_at` = the scoring time (`touch_result.scored_at`) on every path.
  - **`event_id = uuidv5(ns, "touch_scored:" + touch_result.id)`**. It's the same id the backfill uses, so live and backfill emissions can never double.
  - This needs a small `insertEventAt(subject, eventID, occurredAt, …)` variant of `internal/review/store/store.go` `insertEvent`, which today stamps `time.Now()` and a random v4 id.
  - **Producer golden test:** the consumer-path event marshals **byte for byte** to `touch_scored.v2.json`. The self-path event and a backfill row marshal to a sibling fixture `touch_scored.v2.self.json` (`attempt_id: null`, `mock_mode: false`, `anchor_at` = `scored_at`), and assessment's consumer decoder test decodes that fixture too, so both ends agree on both shapes.
- **gateway + web**: delete the `touchesEnabled` guard (the var, the local-only `touchesdev` build-tag file and the 503 `touches_disabled` branch in `internal/gateway/touch.go`), the `touchesEnabled` field on the shared `/api/paths/{slug}/revision/due` handler (and so its DSA alias), and the Revision v1-form fallback. The server-side v1 self endpoint stays ([t0 §5(b)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist)).
- The subject-registry, golden-ACL and budget tests (`internal/platform/events`) stay green. Add rows to [`docs/architecture/events.md`](../../architecture/events.md) for both producers, and mark Flow 6's producer live.

### 2 · `touch_scored` backfill [X]

- Add a `review admin backfill-touch-scored [--batch 500] [--wait 2m]` subcommand in `cmd/review`. It's distroless-safe and follows the `identity admin` pattern from M1-04.
  - It scans `review.touch_result` joined to `revision_item` and inserts one `touch_scored` outbox row per result with the **deterministic id** from task 1: `ON CONFLICT (event_id) DO NOTHING`.
  - Each row carries **M2-02's payload** (task 1): `attempt_id = touch_result.attempt_id` (`NULL` on every self and v1 row, which is all the backfill ever inserts, because consumer-path results were already emitted live), `anchor_at = COALESCE(touch_result.anchor_at, scored_at)` (M2-01 backfilled `anchor_at` from `scored_at`, so it equals `scored_at` on every self and v1 row), `mock_mode = false`, `graded_by = COALESCE(graded_by, 'self')`, `trust = COALESCE(trust, 'honor')`, `occurred_at = scored_at`, and `path_slug` from the item.
  - It prints `scanned / inserted / already-present`, then with `--wait` polls until review's unsent outbox count is 0.
  - **A second run inserts 0.**
- This is an integration test on real Postgres: seed v1-shape results (NULL `graded_by`/`trust`) and v2 results; run twice; assert the counts, ids and `occurred_at`; and assert a backfilled row marshals to `touch_scored.v2.self.json` byte for byte (task 1's golden test covers the shape).
- It runs **once after v1.10.0 is live** (task 8), **before** the replay ([t1 §9 "Events and replay"](../research/t1-content-data-model.md#events-and-replay)).

### 3 · D2 (M2c): anchor on every conclusion + `anchor_rule` + `REVISION_ENTRY_RULE` [X]

- **Migration (review, expand-only)**: `ALTER TABLE review.revision_item ADD COLUMN anchor_rule text NOT NULL DEFAULT 'v1' CHECK (anchor_rule IN ('v1','v2'))`.
  - The constant default makes it expand-safe ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)), and the contract-header lint must stay silent.
  - Take the next free review goose version at rebase. Regenerate sqlc (`ScheduleTouch` and `ReanchorPendingTouches` take `anchor_rule`), and keep `sqlc diff` clean.
- **Config** (`internal/review/config.go`): `REVISION_ENTRY_RULE`, case-insensitive `v1 | v2`.
  - **Unset → `v2`**: D2 ships to every account in this minor, per [ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md).
  - **Any other value → a startup error.** A typo in the kill-switch PR shows as a visible crash-loop, never a silent default.
  - Log `revision_entry_rule=<v>` at startup.
- **`HandleProblemSolved`** (`internal/review/store/store.go`, today's first-clean-only branch at `firstSolve && outcome == OutcomeClean`):
  - **Rule `v2`**, for a revisable item (signal `revisable`; absent → `true`), for **any grade** (clean, rough, assisted, miss, give-up):
    - anchor = the signal's `anchor_at`, else the envelope `occurred_at`;
    - for L1–L5, a missing level → `ScheduleTouch` (`DO NOTHING`, `anchor_rule='v2'`);
    - a **pending** level → `ReanchorPendingTouches` (M2-01) to anchor + Day[level];
    - **passed levels are untouched**;
    - emit `revision_scheduled` for each level inserted or re-anchored ([t4 §6.5](../research/t4-judge-contract.md#65-unsolved-attempts-enter-revision-d2-with-the-revisable-guard), [ADR-0029 §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop)).
  - Below clean also **opens or refreshes the mistake with `revisit_date` = the Day-1 due date**. This fixes the empty timestamp that v1 passes to `openMistakeTx`.
  - **Rule `v1`**: exactly today's behaviour (first clean solve only, rows stamped `v1`). Rows already stamped `v2` stay scheduled ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step)).
  - The inbox dedupe, the journal lock and the pattern pre-resolve (no HTTP call inside the transaction) are unchanged.
  - The R-PF2 owed Day-3 path (`HandleSolutionRevealedEarly`, `DO NOTHING`) is unchanged. Under `v2` the conclusion re-anchors that pending L2.
- **Tests** (real Postgres, table-driven):
  - grade × rule × ladder state (none, owed Day-3 only, partially passed, all passed);
  - redelivery is a no-op;
  - flipping `v2` → `v1` stops new below-clean anchors and leaves existing `v2` rows;
  - `anchor_at` beats `occurred_at`;
  - the mistake gets its Day-1 `revisit_date`;
  - config parsing (unset, `V2`, `v1`, a typo → error).

### 4 · Replay: M2-02's command + replay lock + digest + re-runnable durable; compose rehearsal; runbook [X]

- **Reuse M2-02's command; don't build a second one.** `assessment admin replay-projections --confirm` shipped in v1.9.0 ([M2-02 task 4](sprint-m2-02.md#4--replay-golden--property-tests-the-replay-command-runbook-v2-x)). As M2-02 built it:
  - in **one transaction** it truncates the **v1 and v2** projection tables plus assessment's `inbox`, and rebuilds `proj_activity.mocks` from `mock_session` (status scored);
  - it then replays `xlearn.practice.*` + `xlearn.review.*` from the stream start through the declared durable `assessment-replay` (`DeliverAll`, explicit ack), with the same projection function and inbox claim as the live path;
  - `mock_session`, `rubric_score` and `outbox` are never touched.

  The v1 tables are truncated too because `ApplyProjection` dual-writes: clearing the inbox alone would double-apply every v1 additive counter. Every write is commutative and deduped on the fresh inbox ([ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md)), so the replay itself would be correct with the live consumer running.
- **Add the replay lock** (a firm deliverable: [l-01](sprint-l-01.md)'s `EraseTx` takes this lock shared, "the same way the live projection handler does", so a running replay can't resurrect an erased account's rows mid-erase):
  - one exported key constant and helper in `internal/assessment/store` (e.g. `ReplayLockKey`, `TryReplayLockShared(ctx, tx) (bool, error)` over `pg_try_advisory_xact_lock_shared`);
  - `replay-projections` holds `pg_advisory_lock(ReplayLockKey)` exclusively on its own connection from before the truncate until the replay finishes, then releases it. Its own applies run on other pool connections, so they call the projection function **without** the lock probe (an explicit option on the apply path), or they would block on the CLI's own lock;
  - the live projection handler calls `TryReplayLockShared` at the start of each message's transaction and, when blocked, naks with backoff (`NakWithDelay`). The live durable pauses **without** a scale-to-0 infra PR, and its naked messages redeliver afterwards and dedupe on the inbox;
  - the pause lasts seconds at prod volume, far below the consumer's max-deliver window, so nothing reaches `event_dead_letter`;
  - tests (real Postgres): the handler naks while the lock is held and applies after release; a replay started while a handler transaction holds the shared lock waits for it.
- **Add a read-model digest.** `replay-projections` prints per-stream counts and the digest when it finishes. A new read-only `assessment admin projection-digest` (no `--confirm`) prints the same digest without replaying.
  - The digest is sha256 over each projection table's **semantic columns only**, ordered by the table's full primary key. It excludes `updated_at` and every other `now()`-stamped column, so two replays of the same log are byte-equal.
  - It prints one line per table (table, row count, sha256), a **v1** digest (`proj_coverage`, `proj_mastery`, `proj_heatmap`, `proj_outcome_mix`), a **v2** digest (`proj_activity`, `proj_touch_stats`, `proj_outcome_mix_v2`, plus the v2 columns `path_slug` and `max_passed_level` that M2-02 added to coverage and mastery), and a total.
  - The runbook lists the exact tables and columns in the digest.
- **Make the replay re-runnable.** A durable can't be rewound, and the fine ACL grants no `CONSUMER.DELETE` ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)). With M2-02's `InactiveThreshold` of 1 h, a second run inside the hour would re-bind `assessment-replay` at its ack floor, which is already past the stream end: it would deliver nothing after the truncate and leave near-empty tables. So:
  - Lower the durable's `InactiveThreshold` to **2 min** (in the command's consumer config, and in `topology.go` if it carries the setting). The ACL is per (stream, durable), so the golden ACL block doesn't change.
  - Before truncating, the CLI calls `CONSUMER.INFO` for `assessment-replay` on each stream. If the durable still exists, it polls until the server reaps it (`--reap-wait`, default 5 min). If the durable is still there after that, it exits non-zero ("previous replay durable not yet reaped; retry later") **without truncating**. It never binds an existing durable whose delivered stream sequence is > 0.
  - Test it: a unit test with a fake JetStream client (an existing durable → wait, then fail without truncating), and the compose rehearsal's back-to-back replays.
- **The compose rehearsal** (`hack/rehearse-m2-replay.sh`; paste its output into the PR). It reuses [m1-08](sprint-m1-08.md)'s tag-pinned kit: `deploy/local/compose.rehearse.yml` with `XLEARN_IMAGE_TAG` (compose project `xlearn-rehearse`, own volumes; HEAD builds from the branch), and M1-08's black-box driver `internal/e2e/rehearse_test.go` (`-tags rehearse`, v1 routes + DSA aliases) for the HTTP legs, extended with the touch routes for the HEAD phases only. It mirrors the prod shape, where the v2 tables lack all pre-v1.9.0 history and only the replay fills it:

  | Phase | Images | Do | Check |
  |---|---|---|---|
  | 0 | `XLEARN_IMAGE_TAG=v1.8.0` | fresh volumes; the driver seeds history **A**: clean and below-clean attempts, v1 self touches including a fail/reset, scored mocks | v1 tables fill; there are no v2 tables yet |
  | 1 | `v1.9.0` | the driver seeds history **B** (the same mix) | the v2 tables fill forward with B only |
  | 2 | HEAD | the touch loop with the real producer (the script makes a touch due by moving its due date back in the compose DB, which it owns); a below-clean conclusion (D2); `review admin backfill-touch-scored --wait 2m` twice (the second inserts 0); then `projection-digest` → **L** | the backfill counts; L's v1 part is the full live-built v1 read model |
  | 3 | HEAD | `replay-projections --confirm` → **R1**; run it again at once → it waits for the reap, then replays → **R2** | **R1 = R2** ("replay equal"); **R1's v1 part = L's v1 part** (the replay reproduces the live-built v1 tables); R1's v2 part matches the expected read model: the runbook §C reconcile queries against the compose DB (attempts per day = practice's `problem_solved` outbox rows, touches = `review.touch_result` rows, mocks = scored `mock_session` rows, per-level completed and passed from `touch_result`), **now including history A**; the authed `GET /api/paths/dsa/progress` shows A's days |
  | 4 | HEAD | restart review with `REVISION_ENTRY_RULE=v1` → a new below-clean conclusion; then `REVISION_ENTRY_RULE=bogus` | `v1`: nothing new is scheduled and the `v2` rows remain; `bogus`: review refuses to start |

- **The in-process e2e** (`internal/e2e`, CI `e2e` job, `-tags e2e`) closes the loop with the **real** producer: the touch loop on practice's handlers (start → conclude → `touch_concluded` → review advance/reset → `touch_scored` → `proj_touch_stats`/`proj_activity`); and a below-clean self conclusion → L1–L5 with `anchor_rule='v2'` plus a mistake with a Day-1 `revisit_date`.
- **The runbook.** Update [`docs/runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md) with the v2 prod order: backfill → drain → replay → wait for the reap → replay again (digest equal) → reconcile (§C) → record. Add the digest's table and column list, the replay lock (the live consumer naks for the seconds the replay runs; L-01's erase will wait on it too), the reap wait between runs (the CLI enforces it), and the note that D1 = D2 on prod needs the owner idle between the two runs. At prod volume each replay takes seconds.

### 5 · Authed readers switch to projections v2 [X]

M2-02 kept the v1 readers in v1.9.0 and handed this switch to this sprint: the switch and the replay land in the same tag, so no reader sees partial history ([M2-02 Scope](sprint-m2-02.md#scope)). It goes in the **same PR** as the producers.
- **assessment** (`internal/assessment/progress.go`): `GET /progress/summary` keeps its wire shape, but its `streak` comes from `proj_activity` (a day with any attempt, touch or mock) and its `outcomeMix` from `proj_outcome_mix_v2` (summed over `graded_by`/`trust`), both honouring `?path=`. The course dashboard's streak tile (`internal/gateway/dashboard.go` reads the summary's `streak`) follows with no gateway change. `GET /progress/heatmap` stays on `proj_heatmap` but is no longer called; the pending contract deletes it.
- **gateway** (`internal/gateway/progress.go`, `GET /api/paths/{slug}/progress` and its DSA alias `/api/progress`): replace the `/progress/heatmap` leg with `GET /progress/activity?path=<slug>`, and add `GET /progress/touch-stats?path=<slug>` and `GET /progress/outcome-mix?path=<slug>` legs. Each leg degrades to `null` on its own, as the existing fan-out does. The response gains `heatmap.days[{date, attempts, touches, mocks}]`, `touchStats{levels[{level, completed, passed}], day7PassRate}` and the outcome mix's `provenance`. Update `openapi.yaml` (drift test).
- **web**: the types in `web/src/lib/progress.ts` (`HeatmapDay` becomes `{date, attempts, touches, mocks}`; `touchStats`; `outcomeMix.provenance`), the heatmap legend, and the touch stats plus the provenance chips M2-03 added to `web/src/components/ProgressViews.tsx` (`web/src/screens/Progress.tsx` feeds them). The rest of AB12 stays [m3-13](sprint-m3-13.md)'s.
- **After this tag, no code path reads `proj_heatmap` or the v1 `proj_outcome_mix`** except the uncalled `/progress/heatmap`. A gateway test asserts the progress compose never calls `/progress/heatmap`.
- **Tests**: assessment handler tests (the summary streak from `proj_activity`, the outcome mix from `proj_outcome_mix_v2`); gateway `progress_test.go` (the three new legs, independent degradation); web `Progress.test.tsx` (the new heatmap shape, touch stats, provenance chips).
- The compose rehearsal (task 4) and the prod reconcile (task 8, runbook §C) check the authed Progress numbers against the source services.
- Record in status.md the **earliest tag for the pending contract** (drop `proj_heatmap`, the v1 `proj_outcome_mix` and `/progress/heatmap`, M2-02 task 5): the first contract tag after v1.10.0. That contract raises the floor to ≥ 1.10.0, because 1.9.0's readers still read `proj_heatmap`.

### 6 · Infra ACL PR (conditional; expected n/a) [I]

- **Expected n/a.** `assessment-replay` was declared in `topology.go` and ACL'd by M2-02 (task 6, merged before v1.9.0), and lowering its `InactiveThreshold` doesn't change the ACL.
- **Needed only if** this sprint adds or renames a durable (for example, if the re-runnability fix had to fall back to two alternating replay durables) **and** N1 fine ACLs are live on prod. Then open a `../infra` PR that pastes the re-rendered golden `authorization` block into the NATS values under `infrastructure/messaging/`, following [mi-06](sprint-mi-06.md)'s convention, and **merge it before v1.10.0** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) standing rule).
  - It's a reload with no restart. Verify it with `host-verify --cluster` (no permission-violation logs).
- **Publish permissions need no change**: `xlearn.practice.>` and `xlearn.review.>` already cover the new subjects.
- Record "ACL PR #… merged <date>" or "not needed: <reason>" in this file and in status.md.

### 7 · Tag v1.10.0 [X]

- Run the **release checklist** (below). GitHub release title: **`v1.10.0 — v2 build · M2b/M2c`**.
- The release notes list the behaviour changes:
  - D2: every concluded attempt anchors the ladder, and below-clean opens a mistake with a Day-1 revisit;
  - the Touch screen (AB04);
  - Today in minutes (D4) and the cross-course agenda (AB05);
  - public profile v2 + visibility toggles (AB06/AB22);
  - the heatmap and streak redefined from `proj_activity` on the public and authed views, plus touch stats and grade provenance on Progress;
  - the `REVISION_ENTRY_RULE` kill switch.
- **Rollback floor after: 1.9.0.** It isn't a contract, erase or GA tag, so there's no snapshot requirement. Run `host-verify --cluster` anyway only if a host change happened since the last tag.

### 8 · Post-tag prod steps [H]

These run over the **sanctioned admin-CLI path** (`kubectl exec`, [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)), the only prod writes outside GitOps. **Each use is logged in status.md.** If this session's VPS access is read-only, this becomes an owner action (O): hand the owner the exact commands below, set the task ⛔ "waiting on owner", log the hand-off, and record the owner's output when it comes back.
1. Verify v1.10.0 live (the checklist's after-tag items). The review logs show `revision_entry_rule=v2`.
2. `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-review -- review admin backfill-touch-scored --wait 2m'` → record the counts. Run it again → `inserted 0`.
3. With the owner idle (no attempts, touches or mocks between the two runs): `ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-assessment -- assessment admin replay-projections --confirm'` → digest D1. Run it again (the CLI waits for the previous durable's reap) → D2, and **D1 must equal D2** ("replay equal" on prod). If they differ, compare the per-stream counts: a changed count means a new event landed, so run once more with the owner idle.
4. **Reconcile** per [the runbook §C](../../runbooks/projection-rebuild.md): Progress (heatmap, touch stats, provenance), Dashboard and the public profile against the source services. Confirm `event_dead_letter` is empty.
5. Record the runs, counts, digests and reconcile result in status.md.

No touch attempt can predate v1.10.0 on prod: there was no BFF route before M2-04, and it was guarded until this tag. So there's no `touch_concluded` to back-emit.

### 9 · M2 exit recorded [X]

Set the **Milestones** row M2 to ✅ with **v1.9.0 → v1.10.0**, and the **Tag → floor → snapshot** row as v1.10.0 → floor 1.9.0 → no snapshot (not required). In the **Flag inventory**:
- list `REVISION_ENTRY_RULE` as a **permanent kill switch**: default `v2` in code, the R-a path is an infra PR on `apps/xlearn-review.yaml` env, owner milestone M2, never removed ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service));
- mark M2-04's T-1 gate `touchesEnabled` **removed** (v1.10.0).

Record the pending contract's earliest tag (task 5). Add Decisions-log lines. Note that [l-01](sprint-l-01.md) is unblocked.

## Acceptance criteria

- [ ] **M2 exit:**
  - **replay equal**: in compose R1 = R2, the replay's v1 part equals the live-built one, and the v2 part matches the reconcile (history A included); on prod D1 = D2 with the reconcile clean;
  - **below-clean items get a ladder**: L1–L5 with `anchor_rule='v2'` and a mistake with a Day-1 revisit, in integration tests and the e2e;
  - **the public route is `public-read`-only** (M2-03's tests green in the tag).
- [ ] **D2 is on, and the kill switch is verified in compose**: `REVISION_ENTRY_RULE=v1` stops new anchors, existing `v2` rows stay, and a typo fails startup.
- [ ] practice emits `touch_concluded` and review emits `touch_scored` with M2-02's payload (the producer golden tests pass against `touch_scored.v2.json` and the self fixture), with deterministic ids. The touch loop runs end to end with the real producer, and the `touchesEnabled` guard is gone.
- [ ] The authed Progress and Dashboard read projections v2 (`/progress/activity`, `/progress/touch-stats`, `/progress/outcome-mix`, the summary's streak and mix), and nothing calls `/progress/heatmap`.
- [ ] The backfill is idempotent (a second run inserts 0) and ran on prod **before** the replay. A second replay started inside the reap window waits, or fails without truncating. The replay lock is in place (the live handler naks while it's held; the exported helper is ready for L-01).
- [ ] The ACL PR is merged before the tag, or recorded as not needed. No new in-cluster caller means no NetworkPolicy PR.
- [ ] v1.10.0 is live and verified per the checklist, and status.md records the milestone → tag → floor, the flags, the pending contract's earliest tag and the admin-CLI runs.

## Release

**Tag `v1.10.0`** (indicative: the next free minor, with major = `.release-line` = 1). **Consumers first:** v1.9.0 bound them, and this tag turns the producers on ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline), [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)).

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
- The contract and snapshot lines are **n/a** (expand only; not a contract, erase or GA tag).
- "A new service's image" is **n/a**: no new service.
- The NetworkPolicy line is **n/a**: no new caller.
- The ACL line: n/a unless task 6 ran.
- Also smoke-test a Revision touch end to end, the authed Progress page and the public profile.

**Rollback** ([ADR-0034 §4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step)):
- **R-a:** `REVISION_ENTRY_RULE=v1` via an infra PR on `apps/xlearn-review.yaml` env. It stops new D2 anchors; D2-created items stay scheduled.
- **R-b:** narrow the ranges, never below **1.9.0**. The 1.9.0 readers still work: the v1 tables are dual-written, and the replay rebuilt them too.
- **R-c:** revert plus a patch tag (the default).
- Projections are always recoverable by drop and replay while outboxes are kept.

## Definition of Done

CI green · the tag cut and deployed by Flux (no hand `kubectl apply`) · the release checklist ticked · backfill and replay run on prod in order (by the session, or by the owner from the handed-over commands), with digests equal and the reconcile clean · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board, Milestones M2 ✅, tag → floor, flag inventory, the pending contract's earliest tag, the admin-CLI run log, Decisions log) · local `main` synced in every repo touched (xlearn, and `../infra` if task 6 ran).

## Risks / watch-outs

- **D2-created items stay scheduled after the kill switch** (only new anchors stop). Accepted: `anchor_rule` stamps make them identifiable, and there's no auto-unschedule.
- **Producer/consumer mismatch.** An unknown subject is acked silently ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)), and a known subject with the wrong field names is decoded to defaults. The entry-gate subject check, the byte-for-byte producer golden test against M2-02's fixture, the registry test and the e2e with the real producer guard against both.
- **Double counting.** The deterministic `event_id`s (per `touch_result`, per `attempt`) make live, backfill and re-run emissions collapse on the outbox PK. The assessment inbox dedupes the rest, and the replay truncates the v1 tables with the inbox so dual-written counters never double.
- **Order matters on prod**: backfill → drain → replay. Replaying before the backfill drains leaves `proj_touch_stats`/`proj_activity` short until the next replay. The runbook enforces the order, and the digest-twice check catches a live write racing the replay.
- **A second replay inside the reap window.** Without the reap wait it would truncate and then deliver nothing. The CLI checks `CONSUMER.INFO` first and refuses to truncate while the old durable exists.
- **The live consumer during the replay.** The replay lock makes it nak with backoff; its messages redeliver afterwards and dedupe on the inbox. At prod volume the replay takes seconds, far below max-deliver, so nothing should reach the dead-letter table. Check `event_dead_letter` is empty afterwards.
- **The replay's own applies must skip the lock probe.** They run on other pool connections than the lock holder; with the probe they'd nak forever against the CLI's own exclusive lock. The real-Postgres lock tests cover it.
- **`revisable` shortcut.** It's `true` for DSA only. Before any course with `role=drill` goes live ([p-02](sprint-p-02.md)), practice must compute `revisable` from the item role and `revision.drills` (I8). Leave a failing-if-drills test or a TODO that P-02's checklist picks up.
- **More review load.** Every conclusion now schedules five touches, and M2-04's minute budget absorbs and shows it ([ADR-0026](../../adr/0026-per-course-extensibility-model.md) Consequences).
- **The owner's history changes on the public and authed views** after the replay (heatmap and streak from `proj_activity`). That's expected and listed in the release notes.
- **Prod writes outside GitOps.** Only the two admin CLIs, which are the sanctioned path, logged in status.md. Never run `kubectl apply` or hand-edit a HelmRelease.
