# Prompt — Sprint m2-05 · Producers on + `touch_scored` backfill + reader switch + replay + D2 (M2c) → v1.10.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m2-05.md`](../sprints/sprint-m2-05.md)   ·   **Milestone:** M2 (exit)   ·   **Prereqs:** [m2-02](../sprints/sprint-m2-02.md) (v1.9.0 live), [m2-03](../sprints/sprint-m2-03.md), [m2-04](../sprints/sprint-m2-04.md) merged

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] You'll stay off xLearn (no attempts, touches or mocks) while the session runs the post-tag prod replays (step 10), so D1 = D2 on the first pair. If an event lands between the runs anyway, the session runs once more; nothing waits on you.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and the land-and-sync rule. This sprint **tags**.
- The plan: [`../sprints/sprint-m2-05.md`](../sprints/sprint-m2-05.md). The tasks, the rehearsal table, the release checklist and the rollback are authoritative.
- **M2-02** ([`../sprints/sprint-m2-02.md`](../sprints/sprint-m2-02.md)): task 1 (the `touch_scored` payload and fixture you must marshal to), task 3 (the v2 read endpoints you switch readers onto), task 4 (the replay command you reuse), task 5 (the pending contract). **L-01** ([`../sprints/sprint-l-01.md`](../sprints/sprint-l-01.md)) relies on this sprint's replay lock.
- **D2:**
  - [ADR-0026 §4](../../adr/0026-per-course-extensibility-model.md#4-universal-vs-per-course-method) (revision entry);
  - [ADR-0029 §3–§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) (`anchor_at`, `revisable`, `ReanchorPendingTouches`);
  - [t4 §6.5](../research/t4-judge-contract.md#65-unsolved-attempts-enter-revision-d2-with-the-revisable-guard);
  - [PRD §5.2 R-SR1](../../prd/xlearn-v2-prd.md#52-amended-requirements).
- **Signals:**
  - [t0 §5](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist): (b) `touch_concluded` from practice; (e) `xlearn.review.touch_scored` from review;
  - [t1 §9 "Events and replay"](../research/t1-content-data-model.md#events-and-replay): backfill first, `occurred_at = scored_at`, then drop and replay.
- **Releases:**
  - [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): kill switches are permanent;
  - [§3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules): consumers before producers, expand rules;
  - [§4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md#44-reversibility-by-step): M2c reversibility;
  - [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist): the checklist;
  - [rollout §7](../rollout-plan.md#7-indicative-tag-timeline) (v1.10.0) and [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (the sanctioned `kubectl exec` admin path, logged in status.md).
- **NATS:** [ADR-0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first): `topology.go`, fine ACLs (no `CONSUMER.DELETE`), and "ACL PR before the consuming tag".
- **Replay:** [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md) and [`docs/runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md) (the v2 version from M2-02). The rehearsal kit from [m1-08](../sprints/sprint-m1-08.md): `deploy/local/compose.rehearse.yml` (`XLEARN_IMAGE_TAG`) and `internal/e2e/rehearse_test.go`.
- **Code:**
  - `internal/practice/store/store.go` (M2-01's `concludeTouch()`, `insertEvent`);
  - `internal/review/store/store.go` (`HandleProblemSolved`, `applyTouchOutcomeTx`, `insertEvent`, `marshalEnvelope`), `internal/review/consumers.go`, `internal/review/config.go`, `internal/review/store/queries/revision_item.sql`;
  - `internal/assessment/consumers.go`, `progress.go`, `store/projections.go`, `cmd/assessment` (M2-02's `admin replay-projections`);
  - `internal/platform/events/topology.go`, `testdata/touch_scored.v2.json`, the golden ACL;
  - `internal/gateway/touch.go` (the `touchesEnabled` guard), `progress.go`, `dashboard.go`, `review.go` (`/revision/due`);
  - `web/src/screens/Revision.tsx`, `Progress.tsx`, `web/src/components/ProgressViews.tsx`, `web/src/lib/progress.ts`;
  - `cmd/identity` (the `identity admin` pattern), `cmd/review`;
  - `internal/e2e/` (the in-process e2e and the rehearse driver).
- **Infra:** `../infra/apps/xlearn-review.yaml` (where the kill-switch env would go) and `../infra/infrastructure/messaging/` (the NATS `authorization` block, if N1 is live).

## Context

- **v1.9.0** (M2-02) bound every M2 consumer with the producers idle:
  - review's `touch_concluded` consumer;
  - assessment's `touch_scored` handling (payload pinned by `touch_scored.v2.json`);
  - the projections v2 (`proj_activity`, `proj_touch_stats`, `proj_outcome_mix_v2`), dual-written beside v1 and read by the new `/progress/activity|touch-stats|outcome-mix` endpoints, which nothing calls yet;
  - the replay command `assessment admin replay-projections --confirm` on the declared durable `assessment-replay`.
- **M2-03** merged `public-read`, `/public/stats`, the visibility toggles and profile v2. **M2-04** merged the Touch UI, Today in minutes and the agenda, with the touch start **dark behind the `touchesEnabled` guard (default false)**.
- **This sprint closes M2:**
  - turn the producers on (practice `touch_concluded`; review `touch_scored` with M2-02's exact payload and deterministic ids) and delete the guard;
  - switch the authed Progress and Dashboard readers to projections v2 (M2-02 handed this off: the switch and the replay land in the same tag);
  - backfill `touch_scored`;
  - switch DSA to D2 behind `REVISION_ENTRY_RULE`, with `anchor_rule` stamped;
  - add the replay lock (L-01's erase shares it) and the digest, make the replay re-runnable, rehearse in compose, then **tag v1.10.0**;
  - run backfill → replay on prod through the sanctioned admin CLIs and record the M2 exit.
- **The rollback floor after this tag is 1.9.0.**

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **v1.9.0 live**: read-only `ssh sujaykumar-vps 'k3s kubectl get deploy -n xlearn -o wide'` shows ≥ 1.9.0, and healthz agrees.
- [ ] **M2-03 and M2-04 merged** on `main`, with the `touchesEnabled` guard (default false) present in `internal/gateway/touch.go`.
- [ ] **The subjects match.** `xlearn.practice.touch_concluded` is handled by review. The subject assessment handles for touch scoring is **`xlearn.review.touch_scored`**: review emits it, and review's `xlearn.review.>` ACL covers it. Check `topology.go`, the subject registry and `internal/assessment/consumers.go`. **If M2-02 used another name or owner, stop and report**; a mismatched producer is acked silently and lost.
- [ ] **M2-02's contract is on `main`**: `internal/platform/events/testdata/touch_scored.v2.json` and the `events.md` "Flow 6 — touch scored → projections" section.
- [ ] **M2-02's v2 read endpoints are on `main`**: `GET /progress/activity`, `/progress/touch-stats`, `/progress/outcome-mix`.
- [ ] **M2-02's replay is on `main`**: `assessment admin replay-projections --confirm` exists; `assessment-replay` is in `topology.go`; if N1 fine ACLs are live, it's in the merged infra ACL too (M2-02 task 6). The replay golden test and the v2 runbook are there.
- [ ] **The NATS state is known**: are N1 fine ACLs live on prod ([mi-06](../sprints/sprint-mi-06.md) status row)? That decides whether step 7 could apply.
- [ ] **Parallel sessions**: `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents. Find the **next free minor** (use it instead of 1.10.0 everywhere if taken) and check that no peer PR touches review or assessment migrations. If one does, take the next free goose version at rebase.

## Do this (in order)

1. **[X] Producers.**
   - **practice:** M2-01's `concludeTouch()` already builds the typed payload. Add **the one outbox insert** of `xlearn.practice.touch_concluded` (v2 envelope) inside its transaction.
     - The payload matches `internal/platform/events/testdata/touch_concluded.v2.json` byte for byte.
     - Invert `TestNoTouchProducerYet`: one outbox row per conclusion, none for a voided touch.
     - `event_id = uuidv5(ns, "touch_concluded:"+attempt_id)`.
   - **practice:** if `problem_solved` v2 lacks `anchor_at`/`revisable`, add them (append-only; self path `anchor_at` = the conclusion time; `revisable=true` for DSA).
   - **review:** every scored touch writes `xlearn.review.touch_scored` in the same transaction as its `touch_result` row. Emit it once, in M2-01's shared `applyTouchOutcomeTx`, so both `HandleTouchConcluded` and the v1 `POST /revisions/{id}/score` (self/honor) emit it.
     - **The payload is exactly M2-02's**, in a v2 envelope with `path_slug`: `problem_id, revision_item_id, touch_level, passed, graded_by, trust, attempt_id, anchor_at, mock_mode`.
       - Consumer path: `attempt_id` = the touch attempt, `anchor_at` and `mock_mode` from `touch_concluded`.
       - Self path (and the backfill): `attempt_id: null`, `anchor_at` = `scored_at`, `mock_mode: false`.
     - **No `level` and no `touch_result_id` field**: assessment decodes `touch_level` and `anchor_at`. `touch_result.id` is only the id seed.
     - `occurred_at` = `touch_result.scored_at`; `event_id = uuidv5(ns, "touch_scored:"+touch_result.id)`.
     - Add `insertEventAt(…, eventID, occurredAt, …)`.
     - **Producer golden test:** the consumer-path event marshals byte for byte to `touch_scored.v2.json`. The self-path event marshals to a new sibling `touch_scored.v2.self.json`, and assessment's decoder test decodes that fixture too.
   - **Delete the guard:** the `touchesEnabled` var, the local-only `touchesdev` build-tag file, its 503 `touches_disabled` branch, the `touchesEnabled` field on the shared `/api/paths/{slug}/revision/due` handler (and so its DSA alias), and the v1-form fallback in `Revision.tsx`.
   - Keep the registry, golden and budget tests green. Update [`events.md`](../../architecture/events.md) (producer rows; Flow 6's producer is live).

2. **[X] Backfill CLI.** `cmd/review`: `review admin backfill-touch-scored [--batch 500] [--wait 2m]`.
   - It inserts `touch_scored` outbox rows for every `touch_result` with the **same deterministic id**, `ON CONFLICT (event_id) DO NOTHING`, and **M2-02's payload**: `attempt_id = touch_result.attempt_id` (NULL on every self/v1 row), `anchor_at = COALESCE(anchor_at, scored_at)`, `mock_mode = false`, `graded_by`/`trust` defaulting to self/honor, `occurred_at = scored_at`, and `path_slug` from the item.
   - It prints `scanned/inserted/already-present`, and `--wait` polls review's unsent outbox to 0.
   - Integration test: run twice → the second inserts 0; a backfilled row marshals to `touch_scored.v2.self.json` byte for byte.

3. **[X] D2 (M2c).**
   - **Migration (review, expand-only):** `revision_item.anchor_rule text NOT NULL DEFAULT 'v1' CHECK (anchor_rule IN ('v1','v2'))`. Take the next free goose version; the contract lint stays silent.
   - **sqlc:** `ScheduleTouch`/`ReanchorPendingTouches` take `anchor_rule`; `sqlc generate`, commit, `sqlc diff`.
   - **Config:** `REVISION_ENTRY_RULE`, case-insensitive `v1|v2`, **unset → v2**, anything else → a startup error; log it at startup.
   - **`HandleProblemSolved` under v2** (revisable; absent → true; any grade):
     - anchor = `anchor_at`, else `occurred_at`;
     - missing levels → `ScheduleTouch` (`anchor_rule='v2'`); pending levels → `ReanchorPendingTouches`; passed levels untouched;
     - emit `revision_scheduled` per level changed;
     - below clean → open or refresh the mistake with `revisit_date` = the Day-1 due.
   - **Under v1:** today's code, unchanged. Existing `v2` rows stay.
   - **Table tests:** grade × rule × ladder state; redelivery; the kill-switch flip; `anchor_at` precedence; config parsing.

4. **[X] Replay: reuse M2-02's command; add the replay lock, the digest and re-runnability.**
   - **Don't build a second replay.** M2-02's `assessment admin replay-projections --confirm` already truncates the **v1 and v2** projection tables plus the `inbox` in one transaction, rebuilds `proj_activity.mocks` from `mock_session`, and replays through `assessment-replay` with the live apply + inbox claim. Every write is commutative and inbox-deduped, so the replay is correct on its own.
   - **Replay lock (firm: [l-01](../sprints/sprint-l-01.md)'s `EraseTx` shares it, so a running replay can't resurrect an erased account's rows):** export `ReplayLockKey` and `TryReplayLockShared(ctx, tx)` (`pg_try_advisory_xact_lock_shared`) from `internal/assessment/store`. The CLI holds `pg_advisory_lock(ReplayLockKey)` exclusively on its own connection from before the truncate to the end of the replay. The live projection handler probes it at the start of each message's transaction and naks with backoff when blocked (no scale-to-0 PR). The replay's own applies run on other pool connections, so they skip the probe (an explicit apply option). Real-Postgres tests: naks while held, applies after release; the CLI waits for an in-flight shared holder.
   - **Digest:** a read-only `assessment admin projection-digest`, also printed at the end of `replay-projections`. sha256 over each table's **semantic columns only** (exclude `updated_at` and every `now()`-stamped column), ordered by the full primary key. Print one line per table, a v1 digest, a v2 digest and a total.
   - **Re-runnable durable:** lower `assessment-replay`'s `InactiveThreshold` to **2 min** (the ACL is per durable, so the golden doesn't change). Before truncating, the CLI checks `CONSUMER.INFO` on each stream. If the durable still exists, it polls until it's reaped (`--reap-wait 5m`), else it exits non-zero **without truncating**. It never binds a durable whose delivered sequence is > 0. Unit-test it with a fake JetStream client.
   - Update [`projection-rebuild.md`](../../runbooks/projection-rebuild.md): the v2 prod order (backfill → drain → replay → reap wait → replay, digest equal → reconcile → record), the digest's tables and columns, the replay lock (the live consumer naks for those seconds), and "owner idle between the two prod runs".

5. **[X] Authed readers switch to projections v2** (in the same PR as the producers).
   - **assessment** `GET /progress/summary`: same wire shape, but `streak` from `proj_activity` and `outcomeMix` from `proj_outcome_mix_v2`, honouring `?path=`. The course dashboard's streak tile follows. `/progress/heatmap` stays but is no longer called.
   - **gateway** `internal/gateway/progress.go` (`GET /api/paths/{slug}/progress` + the DSA alias): swap the `/progress/heatmap` leg for `/progress/activity?path=<slug>`, and add `/progress/touch-stats?path=` and `/progress/outcome-mix?path=` legs, each degrading to `null` on its own. The response gains `heatmap.days[{date, attempts, touches, mocks}]`, `touchStats` and the outcome mix's `provenance`. Update `openapi.yaml`.
   - **web:** `web/src/lib/progress.ts` types, the heatmap legend, touch stats, and the provenance chips M2-03 added to `ProgressViews.tsx`. The rest of AB12 is M3-13's.
   - **Tests:** assessment handler tests; gateway `progress_test.go` (new legs, independent degradation, never calls `/progress/heatmap`); web `Progress.test.tsx`.

6. **[X] Compose rehearsal + e2e.** `hack/rehearse-m2-replay.sh` on M1-08's kit (`deploy/local/compose.rehearse.yml`, `XLEARN_IMAGE_TAG`; the driver `internal/e2e/rehearse_test.go`, extended with the touch routes for the HEAD phases). Paste the output into the PR. Follow the plan's phase table:
   1. `v1.8.0` images: seed history **A** (clean and below-clean attempts, v1 self touches incl. a reset, scored mocks).
   2. `v1.9.0` images: seed history **B**. The v2 tables hold B only, which is the prod shape.
   3. HEAD: the touch loop with the real producer (move a touch's due date back in the compose DB); a below-clean conclusion; backfill twice (second = 0) with `--wait`; `projection-digest` → **L**.
   4. HEAD: `replay-projections --confirm` → **R1**; again → it waits for the reap → **R2**. Check **R1 = R2**, **R1's v1 part = L's v1 part**, and R1's v2 part matches the runbook §C reconcile queries against the compose DB, **history A included**; the authed `/api/paths/dsa/progress` shows A's days.
   5. HEAD: restart review with `REVISION_ENTRY_RULE=v1` → a new below-clean conclusion schedules nothing and the `v2` rows remain; `REVISION_ENTRY_RULE=bogus` → review refuses to start.

   Also extend the **in-process e2e** (`internal/e2e`, `-tags e2e`) with the real producer: the touch loop on practice's handlers → `touch_concluded` → review advance/reset → `touch_scored` → the projections; a below-clean conclusion → L1–L5 with `anchor_rule='v2'` + a mistake with a Day-1 revisit.

7. **[I] ACL PR (conditional; expected n/a).**
   - `assessment-replay` was declared and ACL'd by M2-02, and the threshold change doesn't touch the ACL. Record "not needed: declared and ACL'd in M2-02".
   - Only if you had to add or rename a durable **and** N1 fine ACLs are live on prod: open a `../infra` PR pasting the re-rendered golden `authorization` block into the NATS values (`infrastructure/messaging/`, MI-06's convention). Get CI green and merge it **before the tag**. It's a reload, so verify with `host-verify --cluster` (no permission-violation logs).
   - Publish ACLs need nothing (`xlearn.<svc>.>`).

8. **[X] Merge.** One xlearn PR (or a short, serialized series): branch `feat/m2-05-producers-d2-replay`, conventional commits with the attribution lines, CI green (`go test ./...`, `sqlc diff`, the contract lint, registry/golden/budget tests, web tests, e2e), squash-merge.

9. **[X] Tag v1.10.0.** Run the **release checklist** in the plan's *Release* section.
   - Before the tag: parallel-sessions check, next free minor with major = `.release-line`, ACL PR merged or n/a.
   - Tag `v1.10.0` on the merge commit; GitHub release title **`v1.10.0 — v2 build · M2b/M2c`**; release notes list the behaviour changes (D2, the Touch screen, Today in minutes/agenda, profile v2 + visibility, the heatmap and streak from `proj_activity` on the public and authed views plus touch stats and provenance on Progress, the kill switch).
   - After the tag, by looking: healthz version, `get deploy` images, ImagePolicy latest = tag, HelmReleases Ready, a smoke test (login, dashboard, coach, a Revision touch, the authed Progress page, the public profile), and the review logs showing `revision_entry_rule=v2`.

10. **[H] Post-tag prod steps** (the sanctioned `kubectl exec` admin path; log each use in status.md). **Run these yourself: launching approves them (D40).** Don't hand them off or wait. Only if `ssh sujaykumar-vps` fails here, set plan task 8 ⛔ with these exact commands in the manual-path log, for whoever next has access to run in order.
    1. `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-review -- review admin backfill-touch-scored --wait 2m'`, then again → `inserted 0`;
    2. with no attempts, touches or mocks between the runs (the before-launch item): `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-assessment -- assessment admin replay-projections --confirm'` → D1, then again (it waits for the reap) → D2; **D1 must equal D2**. If they differ and the per-stream counts changed, a new event landed: run once more;
    3. reconcile per the runbook §C (Progress incl. heatmap, touch stats and provenance; Dashboard; the public profile vs the source services), and confirm `event_dead_letter` is empty.

11. **[X] Record the M2 exit.** See *Update status*.

## Constraints

- **Consumers before producers** ([ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)). The consumers shipped in v1.9.0, so the producers ship here and not earlier. The producer subjects **and payload field names** must equal what the bound consumers decode, exactly (M2-02's fixture).
- **Outbox / inbox.** Every event is written in the same transaction as its domain row. Deterministic event ids make re-emission and the backfill idempotent, and consumers dedupe on `event_id`. Never trim an outbox.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).** practice is the only writer of `touch_concluded`; review of `touch_scored` and the ladder; assessment of the projections. No cross-schema reads. The admin CLIs touch only their own service's schema and streams. (The compose rehearsal script may query every schema: it's a test harness on its own volumes.)
- **goose + sqlc.** Expand-only (constant-default column), migrations embedded and run on startup under the advisory lock, generated code committed, `sqlc diff` clean. **Never run `Down` in prod.** Don't drop `proj_heatmap` or the v1 `proj_outcome_mix` here: that's the pending contract.
- **Kill switch.** `REVISION_ENTRY_RULE` is a permanent T-2 kill switch: default `v2` in code, never removed. Its R-a path is an infra PR on `apps/xlearn-review.yaml` env. **Don't** open that PR now, because D2 ships on.
- **NATS:** a new durable needs its ACL PR merged **before** the tag. Publish ACLs are per service (`xlearn.<svc>.>`). No stream `DELETE`/`PURGE`, and no `CONSUMER.DELETE` (the reap wait replaces it).
- **GitOps:** never `kubectl apply` and never hand-edit a HelmRelease. The only prod writes outside GitOps are the two admin CLIs via `kubectl exec` (sanctioned, logged). Never move or re-push a tag. Don't suspend the shared IUA.
- **Memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)):** no new pod. The CLIs run inside the existing review and assessment containers and their limits, so the sum is unchanged.
- **No new in-cluster caller** (the gateway already calls assessment), so there's no NetworkPolicy PR.
- **No alerting (D34).** Verification is by looking.
- **Parallel sessions:** check peers' tags, PRs and worktrees (+ ListAgents) before tagging and before claiming any ADR number, and again right before the tag push.

## Deliverables

- practice `touch_concluded` producer (+ `anchor_at`/`revisable` on `problem_solved` v2 if missing); review `touch_scored` producer on M2-02's payload, with the producer golden tests and `touch_scored.v2.self.json`; the guard deleted.
- `review admin backfill-touch-scored`.
- The authed reader switch: assessment summary on v2 tables, gateway Progress on the v2 endpoints, `web/src/lib/progress.ts` + views.
- On M2-02's `replay-projections`: the replay lock (`ReplayLockKey`, `TryReplayLockShared`, live-handler nak), the digest, `assessment admin projection-digest`, and the re-runnable durable (2 min threshold + reap wait).
- The review migration `anchor_rule`, `REVISION_ENTRY_RULE` config, D2 in `HandleProblemSolved`, and tests.
- `hack/rehearse-m2-replay.sh` + its output in the PR; the extended in-process e2e; the updated runbook and `events.md`.
- The `../infra` ACL PR (or a recorded n/a); the tag **v1.10.0** verified; prod backfill + replay run by the session and recorded.

## Update status

- In [`../sprints/sprint-m2-05.md`](../sprints/sprint-m2-05.md): set task rows ✅ (task 6: the PR # or n/a; task 8: ⛔ with the logged commands only if `ssh sujaykumar-vps` failed), and set _Overall_ ✅.
- In [`../status.md`](../status.md):
  - **Sprint board** M2-05 ✅; **Milestones** M2 ✅ (v1.9.0 → v1.10.0; exit: replay equal, below-clean ladders, `public-read`-only);
  - **Tag → floor → snapshot**: v1.10.0 → 1.9.0 → none (not required);
  - **Flag inventory**: `REVISION_ENTRY_RULE` in the permanent kill-switch list (default `v2`; R-a = infra PR on `xlearn-review` env; owner M2; never removed); M2-04's T-1 gate `touchesEnabled` → **removed in v1.10.0**;
  - **Pending contracts**: drop `proj_heatmap`, the v1 `proj_outcome_mix` and `/progress/heatmap`, earliest at the first contract tag after v1.10.0 (it raises the floor to ≥ 1.10.0);
  - the **manual-path log**: both admin-CLI runs on prod with counts, digests D1 = D2 and the reconcile result (or, if `ssh sujaykumar-vps` failed, the logged commands and the ⛔);
  - **NATS rows**: the replay-durable ACL PR or n/a;
  - **Decisions log**: deterministic event ids; the `touch_scored.v2.self.json` fixture; the digest (semantic columns only) and the re-runnable durable (2 min threshold + reap wait); the replay lock as actually built (exclusive in the CLI, shared-try + nak in the live handler, the apply option that skips the probe; L-01's `EraseTx` shares it); D2 forward-only (no retro-anchoring of historic below-clean items); `revisable=true` for DSA until P-02.
- L-01 is now unblocked: note it on the board.
- Record an ADR only if you depart from ADR-0026/0029/0034. Check peers before numbering.

## Done when (acceptance)

- [ ] M2 exit: **replay equal** (compose R1 = R2, the v1 part equals the live-built one, the v2 part matches the reconcile with history A; prod D1 = D2; reconcile clean); **below-clean items get a ladder** (`anchor_rule='v2'`, a mistake with a Day-1 revisit); **the public route is `public-read`-only**.
- [ ] **D2 on; the kill switch verified in compose** (`v1` stops new anchors, `v2` rows remain, a typo fails startup).
- [ ] Both producers emit M2-02's payloads with deterministic ids (golden tests green); the touch loop runs end to end with the real producer; the guard is gone.
- [ ] The authed Progress and Dashboard read projections v2, and nothing calls `/progress/heatmap`.
- [ ] The backfill is idempotent and ran before the replay on prod; a replay inside the reap window waits or fails without truncating.
- [ ] The ACL PR is merged before the tag, or n/a recorded. v1.10.0 is live and verified per the checklist, and status.md is updated.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m2-05-producers-d2-replay`, then conventional commits with the attribution lines, then push, then the PR (step 8). A `../infra` ACL PR only if step 7 applies: its own PR, merged **before** the tag and never folded into it.
2. Once CI is green (fix, then merge, on failure), squash-merge each. Never enable auto-merge. `../infra` has no CI: the rendered-block diff in the ACL PR's body, then `host-verify --cluster` after the reload (step 7), are its checks.
3. **Release action — tag `v1.10.0`** (the next free minor): walk the release checklist (ADR-0034 §6, in the plan), push the tag, let Flux deploy, then verify live by looking (step 9). Then run the post-tag backfill and replay on prod yourself (step 10: D1 = D2, reconcile clean).
4. Update status: the sprint file and `docs/v2/status.md` (the M2 exit, step 11), in the same PR or a follow-up docs PR merged the same way (the tag record and the admin-CLI log need the follow-up).
5. Run `git checkout main && git pull` in every repo touched (xlearn, and `../infra` if step 7 ran). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
