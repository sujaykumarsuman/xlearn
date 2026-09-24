# Sprint m2-02 — M2b consumers + projections v2 → v1.9.0

> **Milestone:** M2 — attempt engine and projections (M2b consumers)   ·   **Track:** product (order 34)
> **Prereqs:** [m2-01](sprint-m2-01.md)   ·   **Unblocks:** [m2-03](sprint-m2-03.md) · [m2-04](sprint-m2-04.md) · [m2-05](sprint-m2-05.md)
> **Release action:** **tag v1.9.0** (indicative: the next free minor at tag time; [ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)) + the infra ACL PR, merged before the tag (only if NATS N1 has landed; else the golden ships with N1).
> **Calendar:** late October (right after m2-01; no owner action needed).
> **Execute with:** [`../prompts/prompt-m2-02.md`](../prompts/prompt-m2-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | `xlearn.review.touch_scored` schema, registry, fixture + the assessment consumer | X | ⬜ |
| 2 | Projections v2 (`proj_activity`, `proj_touch_stats`, provenance outcome mix, touch-aware mastery), dual-written | X | ⬜ |
| 3 | v2 read endpoints + drop `problemTotal=151` | X | ⬜ |
| 4 | Replay: golden + property tests, the replay command, runbook v2 | X | ⬜ |
| 5 | ADR "Projections v2" | X | ⬜ |
| 6 | Infra ACL PR (re-rendered golden: the replay durable), merged before the tag | I | ⬜ |
| 7 | Tag v1.9.0 (release checklist) | X | ⬜ |
| 8 | Verify after the tag (by looking, D34): consumers bound and idle | H | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> M2 milestone row, milestone → tag → floor → snapshot, flags). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m2-01](sprint-m2-01.md) merged (M2a engine + the `touch_concluded` consumer on `main`; no `touch_concluded` producer)
- [ ] m2-01's infra PR (practice `REVIEW_BASE_URL`, NetworkPolicy check) merged in `../infra` — required before the tag
- [ ] v1.8.0 is the live release and no peer tag is pending (`git ls-remote --tags origin`, `gh pr list`)
- [ ] NATS N1 state known ([mi-06](sprint-mi-06.md)): `../infra/infrastructure/messaging/release.yaml` carries the golden
      `authorization` block. If it does, Task 6 opens the infra ACL PR. If N1 hasn't landed, the xlearn golden update is
      enough (N1 pastes the then-current golden, `assessment-replay` included); record Task 6 as "n/a — ACL ships with N1"

## Goal

Ship **every M2a/M2b consumer** — review's `touch_concluded` consumer (m2-01) and assessment's `touch_scored` consumer —
plus the **projections v2** code in **v1.9.0**, with every producer idle
([rollout §4 M2 tag order](../rollout-plan.md#4-per-milestone-detail), [§7](../rollout-plan.md#7-indicative-tag-timeline)).
v1.10.0 ([m2-05](sprint-m2-05.md)) then turns the producers on, backfills `touch_scored`, and drop-and-replays the
projections; because the consumers are already bound, no event is lost
([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)).

## Scope

**In**
- `xlearn.review.touch_scored`: the schema (subject, payload, fixture), the subject-registry entries, and assessment's consumer.
- Projections v2 in assessment ([t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas),
  [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas), rollout §10 **P6, P7 (self), P9**):
  `proj_activity` dated by `anchor_at` (fixes the first-solve inflation), `proj_touch_stats`, the outcome mix with
  `graded_by`/`trust` provenance, touch-aware mastery, `path_slug` on every projection — **dual-written** alongside v1.
- New v2 read endpoints; removal of `problemTotal=151` (`internal/assessment/progress.go:18`).
- Replay golden test, commutativity property tests, the replay command and the v2 runbook.
- Tag **v1.9.0**.

**Out**
- Producers: review emitting `touch_scored`, practice emitting `touch_concluded`, the `touch_scored` backfill, D2 →
  [m2-05](sprint-m2-05.md) (v1.10.0).
- The replay **on prod** (after v1.10.0 and the backfill) → [m2-05](sprint-m2-05.md).
- Switching readers to v2: the public header and `/public/stats` → [m2-03](sprint-m2-03.md); the authed
  Progress/Dashboard → with m2-05's replay step (the switch and the replay land in the same tag, so no reader sees
  partial history). v1 readers are untouched in v1.9.0.
- Judge-checked values (`auto ∧ checked`, P7 checked) and judge stats (P8) → [m3-10](sprint-m3-10.md); AI provenance → [m4-06](sprint-m4-06.md).

## Tasks

### 1 · `xlearn.review.touch_scored` + the assessment consumer [X]

- **Producer is review.** [t0 §5 (e)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist)
  and [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) define
  `xlearn.review.touch_scored`: review scores every touch — the v1 self endpoint and m2-01's `touch_concluded` consumer
  — and backfills from `review.touch_result`. Producer = review (`xlearn.review.touch_scored`, T0 §5 (e), T1 §9);
  [m2-05](sprint-m2-05.md) emits it from review.
- **Payload** (v2 envelope with `path_slug`; `occurred_at` = the scoring time, `scored_at` for the backfill):
  `problem_id, revision_item_id, touch_level, passed, graded_by, trust, attempt_id` (null for the v1 self path and the
  backfill), `anchor_at` (the touch's anchor; `scored_at` for the self path), `mock_mode`. Enums, numbers and refs only.
- Shared fixture `internal/platform/events/testdata/touch_scored.v2.json` (m2-05's producer test must marshal to it).
- Declare the subject constant in review (no emit this sprint); `topology.go` registry: producer review; **handled by
  assessment**; explicitly ignored by identity once its erase durable exists (review's notifications worker filters `xlearn.review.revision_due` only,
  so the registry records it as out of filter). The registry test stays green.
- assessment's projection consumer (`internal/assessment/consumers.go`) decodes the v2 fields (`path_slug`,
  `graded_by`, `trust`, `anchor_at`, `passed`, `touch_level`) with v1 defaults kept forever (v1 envelope → `dsa`,
  `self`, `honor`, `anchor_at = occurred_at`). It rides the existing durable on `XLEARN_REVIEW` (filter
  `xlearn.review.*`), so no new live durable.
- `docs/architecture/events.md`: catalogue row + "Flow 6 — touch scored → projections" (producer idle until v1.10.0).

### 2 · Projections v2, dual-written [X]

A goose migration in `internal/assessment/store/migrations/` (next free version), expand-only:

| Table / column | Grain and rule | Source |
|---|---|---|
| `proj_activity(account_id, path_slug, activity_date, attempts, touches, mocks)` PK `(account_id, path_slug, activity_date)` | UTC day of `anchor_at` (else `occurred_at`); additive counters | `attempts` ← `problem_solved`; `touches` ← `touch_scored`; `mocks` ← the `ScoreMock` transaction (assessment-owned `mock_session`, **not an event**) |
| `proj_touch_stats(account_id, path_slug, touch_level, trust, completed, passed)` PK `(account_id, path_slug, touch_level, trust)` | additive; **Day-7 pass rate** = Σ`passed`/Σ`completed` at level 3 (P9); `trust` kept so honor results stay labelled ([t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas)) | `touch_scored` |
| `proj_outcome_mix_v2(account_id, path_slug, graded_by, trust, outcome, cnt)` PK `(account_id, path_slug, graded_by, trust, outcome)` — a **new table**; the v1 `proj_outcome_mix(account_id, outcome, cnt)` has no provenance columns and stays untouched until the pending contract | first-attempt grade mix with provenance (P7 self); v1 events count as `self`/`honor` | `problem_solved` with `first_solve` |
| `proj_coverage.path_slug`, `proj_mastery.path_slug` | `DEFAULT 'dsa'`; set from the event | `problem_solved` |
| `proj_mastery.max_passed_level` | `GREATEST` over passed touches | `touch_scored` |

- **Redefinitions** (applied by the v2 read paths, Task 3): heatmap and streak from `proj_activity` (attempts + touches
  + mocks) instead of solves + `revision_scheduled` — this removes the +5 "reviews" spike per first clean solve;
  **mastery** = max(best first grade, touch-derived rank) with the proposed mapping *highest passed level ≥ 3 (Day 7) →
  clean-equivalent, levels 1–2 → rough-equivalent* (record it in the ADR; golden-test it). `revision_scheduled` keeps
  feeding `proj_coverage.level1_schedules` (Day-7 retention is unchanged).
- **One name downstream:** every later sprint reads provenance from **`proj_outcome_mix_v2`** (m2-03's `/public/stats`,
  m2-05's reader switch, m3-10's judge-checked %, m3-13's provenance chips); the ADR says so.
- **Concluded / passed per path** ([t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas)'s
  `proj_coverage(+path_slug, concluded, passed)`) are **derived at read time, not stored**: concluded = `proj_coverage.solved`
  (latches on the first `problem_solved`, any grade — every conclusion emits one), passed = `proj_mastery.best_rank ≥ 3`
  (rough or clean, [T0 §4](../research/t0-extensibility-frame.md#4-universal-vs-per-course-method)), grouped by `path_slug`.
  One store read, `CourseCounts(account, paths)`, returns `{path_slug, concluded, passed}`; m2-03 builds `/public/stats` on it
  (no ad-hoc derivation there), and a golden-test assertion covers it.
- **Dual-write:** `store.ApplyProjection` applies the v1 **and** v2 upserts in the same transaction as the inbox claim.
  v1 tables and v1 readers are unchanged in v1.9.0; the v2 tables start filling forward from deploy and are rebuilt
  from the stream start by m2-05's replay.
- **`mocks`:** `ScoreMock` increments `proj_activity.mocks` inside its existing score-once `FOR UPDATE` transaction (no
  event: it is assessment's own data; assessment does **not** consume its own `xlearn.assessment.mock_completed`, and
  no consumer for it is added). A replay rebuilds the column from `mock_session` (status `scored`) with one set-based
  step, `RebuildActivityMocks`. Live and rebuild date a mock identically — the UTC day of `mock_session.started_at`
  (immutable), `path_slug` from the session (`dsa` for v1 rows) — so drop-and-replay stays equal.
- Every write stays an idempotent, commutative upsert (`+`, `GREATEST`, `LEAST`) — [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md).
  No external read in the projection path; `path_slug` comes only from the envelope (or, for mocks, the session row).
  The projections are a pure function of **the event log plus assessment-owned `mock_session`**; the ADR records that.
- `sqlc generate`; commit the generated code.

### 3 · v2 read endpoints + drop `problemTotal=151` [X]

- assessment (JWT, learner role; `internal/assessment/progress.go` + `service.go`): `GET /progress/activity?path=`
  (days + current/longest streak from `proj_activity`, derived at read time), `GET /progress/touch-stats?path=`
  (completed per level, Day-7 pass rate), `GET /progress/outcome-mix?path=` (grade mix by `graded_by`/`trust`, plus
  `judge_checked_pct` = share with `graded_by=auto ∧ trust=checked`, 0 until M3). Omitting `path` aggregates all of the
  caller's courses. **Not called by the gateway in v1.9.0**; m2-03 builds `/public/stats` on the same store reads.
- **Remove `problemTotal`** (`progress.go:18`): `GET /progress/summary` returns `total: null`; the gateway's
  `overrideSolvedTotal` (`internal/gateway/progress.go`) keeps composing the per-course total from curriculum
  ([ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md)); the SPA's progress types accept `null` and render
  "—" if curriculum is unreachable (a one-line guard in `web/src/lib/progress.ts` and its view). Test: a non-DSA course
  never shows "/151".

### 4 · Replay: golden + property tests, the replay command, runbook v2 [X]

- **Golden test** `internal/assessment/store/replay_golden_test.go` (real Postgres, gated on `XLEARN_TEST_DATABASE_URL`)
  over `testdata/replay/m2b.jsonl`: v1 envelopes (the prod-like 19-event shape), v2 `problem_solved` whose `resolution`
  is `immediate`, **`timeout`** (D15 deadline Miss), **`accepted`** (M4 provisional accept) and **`override`**
  (`graded_by=override`, `trust=honor`), `revision_scheduled` and `touch_scored` (self path, consumer pass and fail,
  backfill shape). **Mocks are not events:** the test seeds scored `mock_session` rows (`testdata/replay/m2b.mocks.sql`,
  including one that starts just before a UTC midnight) and runs the same `RebuildActivityMocks` step as the replay
  command. Assert (a) the expected read model (`testdata/replay/m2b.golden.json`, `-update`), including `CourseCounts`,
  (b) **drop and replay → identical**, (c) **N random permutations → identical** (ADR-0018 commutativity property test).
- **Replay command** for m2-05's prod step: `assessment admin replay-projections --confirm` (run via `kubectl exec`, a
  sanctioned manual path; log each use in `docs/v2/status.md`). In **one transaction** it truncates the v1 + v2 projection
  tables and the inbox (so the live consumer and the replay dedupe on the same fresh inbox and can run concurrently), rebuilds `proj_activity.mocks` from `mock_session` (`RebuildActivityMocks`), and replays `xlearn.practice.*` + `xlearn.review.*` from the
  stream start on a **dedicated durable `assessment-replay`** (`DeliverAll`, `InactiveThreshold` 1 h so the server
  reaps it — no delete permission needed). Declare that durable in `topology.go` (the fine ACLs are per durable, so an
  undeclared replay consumer is denied after N3). Rehearse it in compose on seeded data.
- **Runbook** `docs/runbooks/projection-rebuild.md`: a v2 section — the "what feeds what" table for v2, the order
  (m2-05: review's `touch_scored` backfill first, **then** replay), the command, and the reconcile table with the v2
  numbers (touches completed vs `review.touch_result` rows; Day-7 pass rate; activity days). Mark the prod run as m2-05's.

### 5 · ADR "Projections v2" [X]

Record: the `proj_activity` grain and `anchor_at` dating; touches from `touch_scored` (review is the producer); the
touch-aware mastery mapping; the provenance outcome mix in the new `proj_outcome_mix_v2` (the name every later reader
uses); concluded/passed derived at read time (`CourseCounts`); purity = a pure function of the event log **plus
assessment-owned `mock_session`** (mocks dated by `started_at`); dual-write in v1.9.0 then the reader switch at the
replay (v1.10.0); the `assessment-replay` durable; `problemTotal` removal. List the **pending contract**: drop `proj_heatmap`
and the v1 `proj_outcome_mix` at least one release after the last reader switches (tracked in `docs/v2/status.md`).
Add an "Amended by" line to ADR-0018. Take the **next free ADR number** after checking peers' PRs and worktrees.

### 6 · Infra ACL PR [I]

`make nats-acl-render` → the golden changes for the `assessment-replay` durable (CONSUMER.CREATE/INFO/MSG.NEXT/ACK on
`XLEARN_PRACTICE` and `XLEARN_REVIEW`). Update `internal/platform/events/testdata/nats-authorization.golden.conf` in the
xlearn PR. **If N1 has landed** (entry gate), open the `../infra` PR pasting the re-rendered block into
`infrastructure/messaging/release.yaml` — a reload, no restart. If N1 hasn't landed, there is no block to update: record
"n/a — the golden ships with N1" and tell the mi-06 owner the golden changed. **Merged before the v1.9.0 tag** (standing rule, [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)).
GitOps only. If m2-01 also produced an ACL diff, fold both into this one PR.

### 7 · Tag v1.9.0 [X]

Once Tasks 1–6 are merged and the entry gates hold: run the release checklist (below), tag the next free minor
(indicative **v1.9.0**), GitHub release title **`v1.9.0 — v2 build · M2a/M2b consumers`**; gate state after:
**producers off**. Rollback floor after: **unchanged, 1.7.0** (expand-only tag). Snapshot: not required (not a
contract, erase or GA tag).

### 8 · Verify after the tag [H]

By looking (D34), read-only: the release checklist's "after the tag" items, plus **consumers bound and idle**:
- **Bound, 0 pending** — the NATS monitor through the API-server **pod** proxy (the path `host-verify --cluster` uses
  for `/varz`/`/connz`; `/jsz` is not part of host-verify yet):
  `ssh vps 'k3s kubectl get --raw "/api/v1/namespaces/messaging/pods/nats-0:8222/proxy/jsz?consumers=true"'` —
  `account_details[].stream_detail[].consumer_detail[]` lists `review` on `XLEARN_PRACTICE` and `assessment` on
  `XLEARN_PRACTICE` and `XLEARN_REVIEW`, each with `num_pending` 0 and `num_ack_pending` 0 (checked 2026-09-25: this read works).
- **No producer** — `/jsz` has no per-subject counts, so read the outboxes (kept, never pruned) instead:
  `ssh vps 'k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select subject, count(*) from practice.outbox group by 1 union all select subject, count(*) from review.outbox group by 1"'`
  lists neither `xlearn.practice.touch_concluded` nor `xlearn.review.touch_scored`. (A stream-side check needs the NATS
  break-glass `nats stream subjects`; if used, log it in `docs/v2/status.md`.)

## Acceptance criteria

- [ ] Consumers bound and **idle** in prod: review handles `touch_concluded`, assessment handles `touch_scored`; no producer emits either.
- [ ] **Replay golden equal**: drop-and-replay and N permutations of the fixture log produce the golden read model (v1 and v2 tables).
- [ ] Projections v2 dual-written; v1 readers unchanged; `problemTotal` gone (no "/151" for a non-DSA course).
- [ ] The replay command rehearsed in compose; the runbook's v2 section written; the ADR recorded.
- [ ] ACL PR merged before the tag (or n/a — N1 not landed, the golden ships with N1); **v1.9.0 verified** per the checklist.

## Release

**Tag v1.9.0** — the M2a `touch_concluded` consumer and the M2b consumers; producers idle ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline)).
It carries m2-01 (merged earlier) and this sprint. Title `v1.9.0 — v2 build · M2a/M2b consumers`; gate state **producers
off**; floor **1.7.0 (unchanged)**; no snapshot required; no flag change.

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), plus the ADR-0035 §2 standing rule):
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

For this tag: the new in-cluster caller is **practice → review** (m2-01's infra PR); no new service image or policy; not
a contract, erase or GA tag (the snapshot and `host-verify` items are n/a, recorded as such); M6 not started.

## Definition of Done

CI green (`go test -race ./...`, real-PG integration and replay golden tests, `sqlc diff`, the migration lint, the
subject-registry and ACL golden tests, web tests) · e2e green · merged via PR (squash) · infra ACL PR merged before the
tag (or n/a if N1 hasn't landed) · **v1.9.0 tagged, deployed by Flux and verified live** · statuses updated (this file + [`../status.md`](../status.md):
Sprint board, M2 milestone "v1.9.0 consumers live", milestone → tag → floor → snapshot row, pending contracts) · the ADR written.

## Risks / watch-outs

- **Commutativity broken by a new projection** — every v2 write must be `+`/`GREATEST`/`LEAST`; the ADR-0018 permutation
  property test is required, not optional. Watch "first attempt" logic: count only `first_solve=true`, never "the first
  event seen".
- **Dating by `anchor_at` across a day boundary** — use the UTC date of `anchor_at` when present, never the consume
  time; replay determinism depends on it.
- **Producer ownership drift** — `touch_scored` belongs to review (T0 §5 (e), T1 §9). If m2-05 plans a practice-side
  producer, stop and reconcile before v1.10.0: a subject on the wrong stream would miss assessment's `xlearn.review.*` filter.
- **Readers switched too early** — flipping Progress/Dashboard to v2 in v1.9.0 would show the owner an empty history
  until m2-05's replay; keep v1 readers until then.
- **Replay durable and ACLs** — after N3 an undeclared consumer is denied; the `assessment-replay` durable must be in
  `topology.go` and in the merged ACL before m2-05 runs the command.
- **Tag hygiene** — take the next free minor (a peer may have tagged); never move or re-push a tag; the `.release-line`
  guard refuses a wrong major.
- **Memory** — no new pod; the projection tables are tiny at one account (ADR-0035 memory-sum rule unaffected).
