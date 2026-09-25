# Prompt — Sprint m3-10 · review + assessment on judge signals (mistake pre-fill, P7 checked, P8)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-10.md`](../sprints/sprint-m3-10.md)   ·   **Milestone:** M3 (M3-2 slice, ships in `v1.14.0`)   ·   **Prereqs:** [m3-08](../sprints/sprint-m3-08.md) (event fields merged)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, stack, and the land-and-sync rule.
- The plan: [`../sprints/sprint-m3-10.md`](../sprints/sprint-m3-10.md). Its precedence table, column list, grain and acceptance are authoritative for this session.
- [ADR-0029 §3–§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop): the signal fields, revision at `anchor_at`, mistake pre-fill precedence, concepts to revise, D15/D16/D17.
- [t4 §6.1 (I8), §6.3, §6.4, §6.5](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only), [§2.8](../research/t4-judge-contract.md#28-events-payloads-carry-only-enums-numbers-and-refs) (event fields), [§3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) (override excluded from judge-checked %), [§11.4](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag) #13, #15, #17, and §13 (D14–D19 override the body).
- [Rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P7, P8, P10). [ADR-0027 §1 and §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas) (trust labels, public deltas). [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas). [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas).
- [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md) (idempotent, commutative upserts; drop-and-replay), [ADR-0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (the mistake state machine), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand rules; append-only envelope).
- Peer plans this builds on: [m2-01](../sprints/sprint-m2-01.md) (review `touch_concluded` consumer, closed-enum precedent), [m2-02](../sprints/sprint-m2-02.md) (projections v2, replay command), [m2-03](../sprints/sprint-m2-03.md) (`/public/stats`), [m2-05](../sprints/sprint-m2-05.md) (D2 in `HandleProblemSolved`), [m1-06](../sprints/sprint-m1-06.md) (`withhold()`), [m3-08](../sprints/sprint-m3-08.md) (the producer fields).
- Code:
  - `internal/review/store/store.go` (`HandleProblemSolved`, `openMistakeTx`, the touch-concluded path), `store/migrations/`, `store/queries/`, `internal/review/consumers.go`, `mistakes.go`, `weakarea.go`;
  - `internal/assessment/store/` (`ApplyProjection`, `projections.go`, `migrations/`, `queries/`, `replay_golden_test.go`, `testdata/replay/`), `internal/assessment/progress.go`, the public stats handler, `cmd/assessment` (replay command);
  - `internal/gateway/mistakes.go`, `progress.go`, `public_test.go`, `withhold.go`;
  - `internal/platform/events/testdata/` (v2 fixtures, the NATS golden);
  - `web/src/screens/UserDashboard.tsx` (+ test); `docs/runbooks/projection-rebuild.md`.

## Context

- **M3-08** made practice conclude from judge evidence and emit `problem_solved` v2 with `mistake_hint`, `concepts_hint[]`, `language`, `counted_submits`, `first_submit_passed`, `trust` and `graded_by=auto`; judged conclusions are dated by `anchor_at` (a timeout at the deadline, a give-up at `gave_up_at` unless an earlier pass wins).
- **M2** already anchors the ladder on every conclusion at `anchor_at` and opens below-clean mistakes with a Day-1 revisit (M2-05), and keeps a provenance outcome mix (M2-02) whose judge-checked % was always 0.
- **This sprint turns that evidence into learner value:** strong-rule pre-filled mistake categories (so the weak area finally has data), up to 3 concepts to revise, the attempt that last refreshed an entry (M4's analyzer matches on it), judge stats (P8) and a real judge-checked % (P7).
- The consumers ride the same tag as the producer fields (`v1.14.0`). That's safe: additive fields on existing subjects, and judge conclusions only begin after [m3-13](../sprints/sprint-m3-13.md) sets `JUDGE_BASE_URL`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **M3-08 merged:** `internal/platform/events/testdata/` has the updated `problem_solved` v2 (and `touch_concluded`) fixtures with the M3 fields; `gh pr list --state merged --search "m3-08"`.
- [ ] **The M2 base is on `main`:** D2 in review's `HandleProblemSolved` (`anchor_rule`, `REVISION_ENTRY_RULE`), `proj_outcome_mix_v2`, `assessment admin replay-projections`, and `/public/stats` with `judgeCheckedPct`.
- [ ] M1-06's `withhold()` covers mistakes enrichment.
- [ ] **Parallel sessions:** `gh pr list`, `git worktree list`, ListAgents. [m3-09](../sprints/sprint-m3-09.md) may be open (gateway route table, assessment L16 queries): coordinate; take the next free goose version in review and assessment at rebase and re-run `sqlc generate`.

## Do this (in order)

1. **[X] review migration** (next free version, expand-only): `mistake_entry` + `category_source text`, `category_suggested text`, `concepts text[] NOT NULL DEFAULT '{}'`, `concepts_source text`, `last_refreshed_by_attempt_id uuid`; no CHECKs (closed Go enums, M2-01's precedent); backfill `category_source='learner'` where `category IS NOT NULL`. `sqlc generate`.

2. **[X] DSA `mistakes.prefill` table.** Fill the DSA manifest's `mistakes.prefill` (m1-01's **`curriculum/courses/dsa/course.json`**, shipped as `[]`; the register's shorthand `curriculum/dsa/course.json` means this path — don't create a file there) with the plan's 12 rows from t4 §6.3, in order (strong: `tle_perf_only`, `crit_unmet:complexity_stated` → `complexity_misjudged`; `crit_unmet:pattern_named_fast` → `wrong_pattern`; `wa_edge_only` → `off_by_one`; `late` → `time_management`. Weak: `sample_failed` → `misread`; `mle` → `complexity_misjudged`; `re_index`, `re_nil` → `off_by_one`; `ce_only` → `language_bug`; `hint_used`, `blank_draft` → `wrong_pattern`). No rows for `wa`, `tle_small`, `re_other` (analyzer, M4). Update `internal/course/golden_test.go` citing t4 §6.3; the content lint stays green. Check with M3-08's matcher whether self-path conclusions carry the hint and record it.

3. **[X] Precedence function.** `internal/review/store/prefill.go`: pure `applyPrefill(current, hint)` per the plan's table (learner locks; strong fills `category` and overrides weak/strong/analyzer; weak only sets `category_suggested` with `category` NULL; no hint → unchanged; unknown categories dropped). Table tests for every row.

4. **[X] Consumers.** In `HandleProblemSolved`'s existing transaction (after the inbox claim and journal lock), for a below-clean conclusion on a revisable item: open/refresh the mistake (M2-05), apply `applyPrefill`, set `concepts` (≤ 3, ref shape, `concepts_source='rule'` unless learner-locked) and `last_refreshed_by_attempt_id`. Do the same on the failed `touch_concluded` path when the fields are present (tolerant decode). No HTTP calls inside the transaction.

5. **[X] Anchoring golden rows** (real Postgres): timeout Miss anchored at the deadline; give-up with an in-flight earlier pass → clean at that pass, no mistake; give-up with only failures → Miss at `gave_up_at` with a strong pre-fill; `arena_prior=revealed` changes nothing. Plus: a `rule_weak`-only entry doesn't move the weak-area snapshot.

6. **[X] Mistakes API.** review `GET /mistakes` adds `categorySource`, `categorySuggested`, `concepts`, `conceptsSource`; `PATCH /mistakes/{id}` sets `category_source='learner'` on a category change and `concepts_source='learner'` on a concepts change (≤ 3). Gateway `mistakes.go` passes them through, with `concepts` under `withhold()` beside `pattern`; add a concepts sentinel to the route-enumeration sweep.

7. **[X] `proj_judge_stats` (P8).** Assessment migration (next free version): PK `(account_id, path_slug, language)`, counters `judged_attempts`, `counted_submits`, `first_submit_passed`. In `ApplyProjection` (same tx as the inbox claim): additive upsert for `problem_solved` v2 with `counted_submits ≥ 1` and a language. `GET /progress/judge-stats?path=` (learner role). Add the table to the replay command's truncate list. No prod replay (fills forward from zero).

8. **[X] Judge-checked % (P7).** `auto ∧ checked` ÷ first-attempt conclusions, from `proj_outcome_mix_v2`; `null` until the path has any `graded_by='auto'` conclusion. Apply to `/progress/outcome-mix` and `/public/stats`. Extend the replay golden (`testdata/replay/`) with `auto/checked`, `auto/honor`, timeout Miss and `override/honor` events plus judge stats; assert the read model, drop-and-replay equal, and N random permutations equal.

9. **[X] Public.** `/public/stats` gains nothing but the real `judgeCheckedPct`. Extend the assessment shape test and the gateway P10 test (`internal/gateway/public_test.go`): `judgeCheckedPct` allowed (number or null); planted `judgeStats`, `countedSubmits`, `languages`, arena and Run fields refused. `UserDashboard.tsx`: render "Judge-checked N%" when non-null, keep AB06's "—" + tooltip for null; tests for both.

10. **[X] Gateway Progress.** `internal/gateway/progress.go`: the course Progress aggregate (and its DSA alias) adds `judgeCheckedPct` and `judgeStats` from the two assessment reads; a failed section degrades to `null`. Update `openapi.yaml`.

11. **[I] ACL check.** `make nats-acl-render`; the golden must be unchanged (no new stream, subject or durable). Only if it changes: open the `../infra` PR with the re-rendered block in `infrastructure/messaging/release.yaml`, merged **before** `v1.14.0`.

12. **[X] Compose e2e + docs.** In `internal/e2e` with judge + runner + the fixture pack: a submission failing only edge-tagged hidden cases (add a synthetic item to `internal/judge/testdata/pack` if none exists) → give-up → review's entry has `category_source=rule_strong`, concepts and `last_refreshed_by_attempt_id`; a clean judged pass → `proj_judge_stats` row and judge-checked % 100. These legs need the real runner: [m3-06](../sprints/sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner`, build tags `e2e,runner`), which is the acceptance in CI's Linux **`judge-runner-e2e`** job; the runner can't run on macOS, so locally script the fake runner (`internal/judge/runnerfake`) to return the edge-only failure, or use the spike's multipass VM. Update [`events.md`](../../architecture/events.md), [`data-model.md`](../../architecture/data-model.md), [`api.md`](../../architecture/api.md) + `openapi.yaml`, and [`projection-rebuild.md`](../../runbooks/projection-rebuild.md). Then `go test ./...`, `sqlc diff`, `npm test` in `web/`, the e2e lane.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).** review and assessment read only their own schemas and the event payloads; the manifest categories come from the compiled `internal/course` package; no cross-schema reads and no HTTP inside a consumer transaction.
- **goose + sqlc:** expand-only migrations (nullable or constant-default columns, backfill by `UPDATE`, no drops), embedded and run under the advisory lock; commit the sqlc output; `sqlc diff` clean; the contract-header lint silent.
- **Outbox/inbox:** every consumer write stays in the inbox-claimed transaction; redelivery is a no-op; projections stay additive and commutative (ADR-0018).
- **Practice is the single writer of learning signals** (ADR-0026 §3): review and assessment react; they never compute grades.
- **Frontend:** `theme.css` verbatim, match the frozen AB06; no other screen changes here (AB12 is M3-13's).
- **GitOps:** never `kubectl apply`; the ACL PR (if any) is its own task, merged before the consuming tag. **No alerting (D34).** No new pod (memory sum unchanged).
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before merging and before claiming an ADR number.
- This sprint **does not tag**.

## Deliverables

- The DSA manifest's `mistakes.prefill` table + golden update.
- review: the `mistake_entry` migration, `prefill.go` + tests, the consumer changes, the mistakes API fields and PATCH locking, the anchoring golden rows.
- assessment: the `proj_judge_stats` migration, consumer, `GET /progress/judge-stats`, real judge-checked % with `null` semantics, the replay golden/permutation tests, the replay command list.
- gateway: mistakes pass-through (concepts withheld), the Progress aggregate fields, the extended P10 test; `UserDashboard.tsx` renders the %.
- Docs (`events.md`, `data-model.md`, `api.md`, `openapi.yaml`, `projection-rebuild.md`); an infra ACL PR only if the golden changed.

## Update status

- In [`../sprints/sprint-m3-10.md`](../sprints/sprint-m3-10.md): set each task 🔄 → ✅ (⛔ with a reason), and set _Overall_.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for M3-10 (M3 stays 🔄);
  - the public-dashboard rows: **P7 (checked)** → done (ships in `v1.14.0`); **P8** → "authed data done (`v1.14.0`); public exposure pending owner item (ADR-0027 §7 vs AB06)" — **not** done;
  - **Decisions log** lines: how precedence is stored (`category` vs `category_suggested`), `judgeCheckedPct = null` before any judged conclusion, the P8 grain (course attempts with ≥ 1 counted submit, per language), "**departure from ADR-0027 §7**: judge stats stay authed-only in M3 because the frozen AB06 has no slot", and whether an ACL PR was needed;
  - an **owner open item** (non-blocking, D40: land without waiting for an answer; the answer becomes a follow-up PR): "ADR-0027 §7 puts judge stats (counted submits, first-submit acceptance, languages) on the public course card, but AB06 has no slot and M3 keeps them authed-only. Amend ADR-0027 §7, or revise AB06 and widen the P10 allowlist?" An answer given in-session is followed like any owner instruction; record the outcome.
- No ADR is expected (ADR-0029/0027/0018 cover it). If you depart from them, run the parallel-sessions check before numbering one.

## Done when (acceptance)

- [ ] The DSA `mistakes.prefill` table matches t4 §6.3 (golden updated).
- [ ] A below-clean judged attempt opens a pre-filled mistake (`rule_strong`, concepts, `last_refreshed_by_attempt_id`), in compose.
- [ ] The precedence table passes; a learner PATCH locks category and concepts; weak suggestions never move the weak area.
- [ ] Judge-path anchoring golden rows pass.
- [ ] The judge stats projection replays equal (drop-and-replay and permutations); judge-checked % counts only `auto ∧ checked` and is `null` before any judged conclusion.
- [ ] `/public/stats` shows judge-checked % only (allowlist tests green); AB06 renders the value or "—".
- [ ] The Progress aggregate carries `judgeCheckedPct` + `judgeStats`; mistake concepts are withheld while live.
- [ ] NATS golden unchanged, or its infra PR merged before `v1.14.0`.
- [ ] CI green (`go test ./...`, `sqlc diff`, web tests, OpenAPI drift, route enumeration, e2e).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Branch `feat/m3-10-judge-signals-review-assessment`; plus the `../infra` ACL PR only if step 11's golden changed (infra has no CI: paste the `make nats-acl-render` output into its PR body and merge on it).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships in `v1.14.0`):** Nothing deploys; it ships in `v1.14.0` (cut by [m3-13](../sprints/sprint-m3-13.md)). Don't tag. An ACL PR, if one was needed, is merged on its own, before the `v1.14.0` tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, and `../infra` if you opened the ACL PR). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
