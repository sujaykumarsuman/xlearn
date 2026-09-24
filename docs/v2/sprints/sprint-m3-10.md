# Sprint m3-10 — review + assessment on judge signals (mistake pre-fill, P7 checked, P8)

> **Milestone:** M3 — judge plus code grader (**M3-2** slice, rides `v1.14.0`)   ·   **Track:** product   ·   **Order:** 50
> **Prereqs:** [m3-08](sprint-m3-08.md) (the `problem_solved` v2 / `touch_concluded` M3 fields) · builds on [m2-05](sprint-m2-05.md) (D2 anchoring) and [m2-02](sprint-m2-02.md)/[m2-03](sprint-m2-03.md) (projections v2, `/public/stats`)
> **Unblocks:** [m3-13](sprint-m3-13.md) (AB12 source chip, concepts, judge-checked %) · [m4-03](sprint-m4-03.md) (the analyzer applies against `last_refreshed_by_attempt_id`)
> **Release action:** **merge only** (ships in `v1.14.0`, tagged by [m3-13](sprint-m3-13.md)) · infra ACL PR **only if** the NATS golden changes (expected: none)
> **Calendar:** mid-November (agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m3-10.md`](../prompts/prompt-m3-10.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | review migration: `mistake_entry.{category_source, category_suggested, concepts, concepts_source, last_refreshed_by_attempt_id}` + backfill | X | ⬜ |
| 2 | DSA manifest `mistakes.prefill` table (t4 §6.3 proposal) + golden update — M3-08 reads it | X | ⬜ |
| 3 | review pre-fill: precedence (learner > strong rule > analyzer > weak rule), concepts, `anchor_at` golden rows for judge resolutions | X | ⬜ |
| 4 | review + gateway mistakes API: source, suggestion, concepts; a learner PATCH locks; concepts through `withhold()` | X | ⬜ |
| 5 | assessment `proj_judge_stats` (P8): migration, consumer, read endpoint, replay list + runbook | X | ⬜ |
| 6 | assessment judge-checked % (P7 checked): real values, `null` before any judged conclusion, exclusions, replay golden + property tests | X | ⬜ |
| 7 | Public: `/public/stats` judge-checked % only; allowlist tests; AB06 renders the value | X | ⬜ |
| 8 | Gateway Progress composition: `judgeCheckedPct` + `judgeStats` on the course Progress aggregate (data for M3-13) | X | ⬜ |
| 9 | NATS ACL check → infra PR only if the golden changes | I | ⬜ |
| 10 | Docs: `events.md`, `data-model.md`, `api.md`/`openapi.yaml`, `projection-rebuild.md` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **[m3-08](sprint-m3-08.md) event fields merged:** `problem_solved` v2 carries `anchor_at, revisable, arena_prior, gave_up, resolution, mistake_hint{category, strength, signal}, concepts_hint[], language, counted_submits, first_submit_passed, trust` (with `graded_by=auto` on the judge path), and `touch_concluded` carries `trust, evaluation_ids[]` (plus `mistake_hint`, `concepts_hint[]` for judged re-solves); the shared v2 fixtures in `internal/platform/events/testdata/` are updated. (`mistake_hint` stays null until task 2 fills the DSA table; M3-08's tests use a fixture table.)
- [ ] **The M2 base is on `main`:** [m2-05](sprint-m2-05.md)'s D2 path (review anchors L1–L5 on every conclusion at `anchor_at`, below-clean opens/refreshes a mistake with a Day-1 `revisit_date`, `REVISION_ENTRY_RULE`), [m2-02](sprint-m2-02.md)'s projections v2 (`proj_outcome_mix_v2` with `graded_by`/`trust`), the `assessment admin replay-projections` command, and [m2-03](sprint-m2-03.md)'s `GET /public/stats` with `courses[].judgeCheckedPct`.
- [ ] [m1-06](sprint-m1-06.md)'s `withhold()` covers mistakes enrichment (pattern) — this sprint adds `concepts` to it.
- [ ] **Parallel sessions:** [m3-09](sprint-m3-09.md) runs beside this sprint and edits the gateway route table plus assessment **queries** (L16). This sprint touches `internal/gateway/{progress,mistakes,public}.go` and adds one migration each to review and assessment: take the next free goose version at rebase (CI fails on a duplicate) and re-run `sqlc generate` (`gh pr list`, `git worktree list`, ListAgents).

## Goal

Let downstream services **use judge evidence**, as [ADR-0029 §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) and [t4 §6.3–§6.5](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) specify:
- the DSA manifest gets its **`mistakes.prefill` table** (t4 §6.3), so practice's `mistake_hint` (M3-08) stops being null;
- **review** pre-fills mistakes from practice's `mistake_hint` by precedence (learner > strong rule > analyzer ≥ 0.6 > weak rule; a learner edit locks the field), stores up to 3 **concepts to revise**, remembers which attempt last refreshed the entry (M4's analyzer needs it), and keeps scheduling from `anchor_at`. Strong rules fill `category`, so the **weak area switches on in M3**.
- **assessment** gains `proj_judge_stats` (P8: counted submits, first-submit acceptance %, languages) and real **judge-checked %** (P7 checked: `graded_by=auto ∧ trust=checked`), both replay-equal.
- **Public stats respect the allowlist:** `/public/stats` gains a real judge-checked % and nothing else judge-related.

## Scope

**In**
- The DSA manifest's `mistakes.prefill` table in `curriculum/courses/dsa/course.json` ([t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)), which M3-08's matcher reads (it ships as `[]` from M1-01).
- review: one expand-only migration on `mistake_entry`; the `problem_solved` v2 and `touch_concluded` consumers apply `mistake_hint`/`concepts_hint`/`attempt_id`; the mistakes API exposes the new fields; the learner PATCH locks category and concepts.
- assessment: one expand-only migration (`proj_judge_stats`); the projection consumer; `GET /progress/judge-stats?path=`; judge-checked % made real on `/progress/outcome-mix` and `/public/stats`; replay golden, drop-and-replay and permutation tests; the replay command's table list.
- gateway: pass-through of the new mistake fields (with `withhold()` on concepts); `judgeCheckedPct` + `judgeStats` on the course Progress aggregate; the P10 allowlist test.
- web: AB06 (`UserDashboard.tsx`) shows the judge-checked % once it's non-null (the "—" state stays for `null`).

**Out**
- The analyzer and its suggestions (`evaluation_analyzed`, `review-analysis` durable, analyzer-sourced categories/concepts) → M4 ([m4-03](sprint-m4-03.md)).
- The authed UI for the source chip, concepts and judge-checked % (AB12: `Mistakes.tsx`, `ProgressViews.tsx`, `Week.tsx`) → [m3-13](sprint-m3-13.md).
- AI provenance (P7 AI) → [m4-06](sprint-m4-06.md).
- Judge stats on the **public** profile: [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas) ("What each course shows": counted submits, first-submit acceptance and languages) and t1 §9 put them on the public course card, but AB06 (frozen by [ds-m2-01](sprint-ds-m2-01.md)) has no slot for them, so they stay authed-only here. That is a **recorded departure from ADR-0027 §7**, pending an owner decision (see Risks and the DoD); P8's public half stays open.
- Practice's `mistake_hint` computation (the matcher over judge `signals[]` and attempt signals) is [m3-08](sprint-m3-08.md)'s; this sprint only fills the DSA table it reads (task 2).

## Tasks

### 1 · review migration [X]

Sources: [t4 §11.4](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag) #15, [t4 §6.3–§6.4](../research/t4-judge-contract.md#64-concepts-to-revise), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand rules).
- Goose migration in `internal/review/store/migrations/` (next free version), expand-only:

  | Column | Type | Rule |
  |---|---|---|
  | `category_source` | `text NULL` | `learner \| rule_strong \| rule_weak \| analyzer`, validated by a **closed Go enum, no CHECK** (M2-01's `resolution` precedent: M4 needs no constraint change) |
  | `category_suggested` | `text NULL` | the latest non-learner suggestion; shown as "suggested" while `category` is NULL |
  | `concepts` | `text[] NOT NULL DEFAULT '{}'` | ≤ 3 ordered refs `<course>:concept:<slug>`, checked in Go |
  | `concepts_source` | `text NULL` | `rule \| analyzer \| learner`, closed Go enum |
  | `last_refreshed_by_attempt_id` | `uuid NULL` | the `attempt_id` of the signal that last opened or refreshed the entry |

- **Backfill** (same migration, batched `UPDATE`): rows with a non-NULL `category` get `category_source='learner'` (every v1/M2 category was picked by the learner). Constant defaults and nullable columns keep it expand-safe; the contract-header lint stays silent.
- `sqlc generate`; commit the output; `sqlc diff` clean.

### 2 · DSA `mistakes.prefill` table [X]

Sources: [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (tier 2, the DSA proposal), [m1-01](sprint-m1-01.md) (manifest `mistakes.prefill[{signal, category, strength}]`, shipped as `[]`), [m3-08](sprint-m3-08.md) task 6 ("the DSA table is empty until m3-10 fills it").
- Fill the DSA course manifest's `mistakes.prefill` (m1-01's **`curriculum/courses/dsa/course.json`**; the register's shorthand `curriculum/dsa/course.json` means this path — don't create a file there) with t4 §6.3's DSA rows, in order (first match wins):

  | # | `signal` | `category` | `strength` |
  |---|---|---|---|
  | 1 | `tle_perf_only` | `complexity_misjudged` | strong |
  | 2 | `crit_unmet:complexity_stated` | `complexity_misjudged` | strong |
  | 3 | `crit_unmet:pattern_named_fast` | `wrong_pattern` | strong |
  | 4 | `wa_edge_only` | `off_by_one` | strong |
  | 5 | `late` | `time_management` | strong |
  | 6 | `sample_failed` | `misread` | weak |
  | 7 | `mle` | `complexity_misjudged` | weak |
  | 8 | `re_index` | `off_by_one` | weak |
  | 9 | `re_nil` | `off_by_one` | weak |
  | 10 | `ce_only` | `language_bug` | weak |
  | 11 | `hint_used` | `wrong_pattern` | weak |
  | 12 | `blank_draft` | `wrong_pattern` | weak |

  `wa`, `tle_small` and `re_other` get **no** row: they're the analyzer's (M4). The manifest format takes one signal per row, so t4's "`gave_up` + `blank_draft`" is row 12 alone (`blank_draft` only arises on a give-up or timeout close).
- Update `internal/course/golden_test.go` in the same PR, citing t4 §6.3 (m1-01's rule for deliberate DSA method changes); the content CI lint (every `prefill.category` declared, every `signal` in the closed vocabulary, [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) #13) stays green.
- Confirm with M3-08's matcher where the hint is computed: if self-path conclusions also carry `mistake_hint` (e.g. `late`, `hint_used`), the owner's self-path mistakes get pre-filled too — record it in the Decisions log either way.

### 3 · review pre-fill, concepts and anchoring [X]

Sources: [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (tiers, precedence, "weak rules show as suggested"), [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise), [t4 §6.5](../research/t4-judge-contract.md#65-unsolved-attempts-enter-revision-d2-with-the-revisable-guard), [ADR-0029 §3–§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop), [ADR-0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (the state machine stays one transaction).
- **Pure precedence function** `applyPrefill(current, hint) → next` in `internal/review/store/prefill.go`, table-tested:

  | Current `category_source` | Hint strength | Result |
  |---|---|---|
  | `learner` | any | unchanged (a learner edit **locks** the field); `category_suggested` updated for audit |
  | none / `rule_weak` / `rule_strong` / `analyzer` | **strong** | `category = hint`, `category_source = rule_strong`, `category_suggested = hint` |
  | none / `rule_weak` | **weak** | `category` stays NULL, `category_source = rule_weak`, `category_suggested = hint` |
  | `rule_strong` / `analyzer` | weak | unchanged (stronger source wins); `category_suggested` unchanged |
  | any | none (no hint, e.g. `wa`, `tle_small`: analyzer-only in M4) | unchanged |

  An unknown category (not in the course's manifest categories, m1-09's validator) is logged and dropped, never stored.
- **Where it applies** (inside the existing transaction of M2-05's `HandleProblemSolved`, after the inbox claim and the journal lock):
  - below-clean conclusion on a **revisable** item (I8) opens or refreshes the open mistake (M2-05) **and** applies `applyPrefill` + concepts + `last_refreshed_by_attempt_id = attempt_id`;
  - `touch_concluded` with `passed=false` (M2-01's open/re-open path) does the same when the signal carries `mistake_hint`/`concepts_hint` (tolerant decode: M3-08 adds them for judged re-solves only, so self-path touches carry none);
  - non-revisable items, clean conclusions, mock, arena and Run never open mistakes.
- **Concepts:** `concepts_hint[≤3]` → `concepts` with `concepts_source='rule'`, unless the source is `learner` (locked). Validate the ref shape and the count only; never call curriculum inside the transaction.
- **Anchoring (already D2 from M2-05; prove it for the judge path):** add golden rows to the review integration tests with v2 judge fixtures —
  - `resolution=timeout` (D15): the ladder and the mistake's Day-1 `revisit_date` date from `anchor_at` = the **deadline**, not the consume time;
  - give-up with an in-flight pass below `close_seq` → `grade=clean`, anchored at that pass's `submitted_at`, no mistake;
  - give-up with only failures → Miss at `gave_up_at`, mistake opened with the strong pre-fill;
  - `arena_prior=revealed` changes nothing here (D17: recorded, no cap).
- **Weak area:** unchanged code path (it counts non-NULL `category` on open entries), so strong-rule, learner and (M4) analyzer categories count and weak suggestions don't. Add a test that a `rule_weak`-only entry doesn't move the weekly snapshot.

### 4 · Mistakes API: review + gateway [X]

Sources: [PRD R-JG4](../../prd/xlearn-v2-prd.md) ("the learner can always change them"), [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise) (withheld while live).
- review `GET /mistakes` (`internal/review/mistakes.go`): `mistakeJSON` gains `categorySource`, `categorySuggested`, `concepts[]`, `conceptsSource`.
- review `PATCH /mistakes/{id}`: a `category` change sets `category_source='learner'` (locks); a `concepts` array (≤ 3, ref shape) sets `concepts_source='learner'`; 2 KiB prose caps unchanged.
- gateway (`internal/gateway/mistakes.go`): pass the new fields through; `concepts` goes through `withhold()` beside `pattern` (stripped while the item is live), and the route-enumeration sweep gains a concepts sentinel.
- The Mistakes UI (source chip, suggested label, concept chips) is [m3-13](sprint-m3-13.md)'s AB12 work; this sprint only ships the data.

### 5 · assessment `proj_judge_stats` (P8) [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P8, [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) (`proj_judge_stats(account, path, language)`), [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md) (idempotent, commutative upserts).
- Goose migration in `internal/assessment/store/migrations/` (next free version):
  `proj_judge_stats(account_id uuid, path_slug text, language text, judged_attempts int NOT NULL DEFAULT 0, counted_submits int NOT NULL DEFAULT 0, first_submit_passed int NOT NULL DEFAULT 0, PRIMARY KEY (account_id, path_slug, language))`.
- **Consumer** (`store.ApplyProjection`, same transaction as the inbox claim): for `problem_solved` v2 with `counted_submits ≥ 1` (a judged course attempt) and a non-empty `language`, upsert `+1 / +counted_submits / +(first_submit_passed ? 1 : 0)`. Additive only, so it's commutative; v1 and self-path events (no `counted_submits`) contribute nothing.
- **Read:** `GET /progress/judge-stats?path=` (JWT, `RequireRole("learner")`) → `{countedSubmits, judgedAttempts, firstSubmitAcceptancePct, languages[{language, attempts}]}`; omitting `path` aggregates the caller's courses.
- **Replay:** add `proj_judge_stats` to `assessment admin replay-projections`' truncate list and to [`docs/runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md)'s "what feeds what" table. **No prod replay is needed:** judged conclusions start only after M3-13 sets `JUDGE_BASE_URL`, so the table fills forward from zero.

### 6 · Judge-checked % (P7 checked) [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P7, [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule) (trust labels), [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) (override excluded), [t4 §11.4](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag) #17.
- The numerator is `graded_by='auto' ∧ trust='checked'` over the first-attempt conclusions already in `proj_outcome_mix_v2` ([m2-02](sprint-m2-02.md)); the denominator is all first-attempt conclusions in the path. Honor (in-process profiles such as go-race, public-key probes), `override`, `claimed` and `review_flag` all carry `trust='honor'` and are excluded by construction.
- **`null` semantics:** return `judgeCheckedPct: null` while the path has **no** `graded_by='auto'` conclusion for this account (so "not measured" never reads as "0% checked"), else the rounded percentage. This replaces M2's "always 0 before M3" on `/progress/outcome-mix` and `/public/stats`.
- **Replay golden** (`internal/assessment/store/replay_golden_test.go`, `testdata/replay/`): add judge-path events — `auto/checked` clean and rough, `auto/honor` (go-race), a timeout Miss (`auto/checked`), an `override/honor` — plus `proj_judge_stats` rows; assert the read model, **drop-and-replay equal**, and **N random permutations equal** (commutativity, ADR-0018).

### 7 · Public: judge-checked % only [X]

Sources: [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas), [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P7/P10, [m2-03](sprint-m2-03.md).
- `/public/stats` (assessment, `public-read` only): `courses[].judgeCheckedPct` becomes real (`null` until judged). Nothing else judge-related is added: **no** `judgeStats`, counted submits, languages, arena or Run counts.
- **Allowlist tests:** extend the assessment-side shape test and the gateway P10 test (`internal/gateway/public_test.go`) — `judgeCheckedPct` allowed (number or null); sentinel-planted `judgeStats`, `countedSubmits`, `languages`, `arena*`, `run*` refused.
- **Web:** `UserDashboard.tsx` renders the AB06 "Judge-checked N%" when the value is non-null; `null` keeps AB06's F3 "—" with its tooltip. `UserDashboard.test.tsx` covers both.

### 8 · Gateway Progress composition [X]

- The course Progress aggregate (`internal/gateway/progress.go`, `GET /api/paths/{slug}/progress` and its DSA alias) fans out to `GET /progress/judge-stats?path=` and `/progress/outcome-mix?path=`, adding `judgeCheckedPct` and `judgeStats` to the response (additive; a failed section degrades to `null`, as the other fan-outs do). [m3-13](sprint-m3-13.md) renders them (AB12).
- OpenAPI updated (drift test).

### 9 · NATS ACL check [I]

- This sprint adds **no** stream, subject or durable: review and assessment already consume `xlearn.practice.*` and `touch_concluded` on their existing durables. Run `make nats-acl-render` and confirm the golden (`internal/platform/events/testdata/nats-authorization.golden.conf`) is unchanged.
- **Only if it changes** (e.g. a filter change): open the `../infra` PR pasting the re-rendered block into `infrastructure/messaging/release.yaml` (a reload, no restart), merged **before** the `v1.14.0` tag ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) standing rule). GitOps only.

### 10 · Docs [X]

- [`docs/architecture/events.md`](../../architecture/events.md): the M3 fields review and assessment read, and the precedence rule.
- [`docs/architecture/data-model.md`](../../architecture/data-model.md): the `mistake_entry` columns and `proj_judge_stats`.
- [`api.md`](../../architecture/api.md) + `openapi.yaml`: the mistakes fields, `PATCH` locking, `GET /progress/judge-stats`, the Progress aggregate fields, `judgeCheckedPct: null`.
- [`docs/runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md): the v2 table list gains `proj_judge_stats`.

## Acceptance criteria

- [ ] **A below-clean judged attempt opens a pre-filled mistake** (compose e2e: a submission failing only edge-tagged hidden cases — add such a synthetic item to the public fixture pack `internal/judge/testdata/pack` if missing — → give-up → Miss with `mistake_hint{off_by_one, strong}` → review's open entry has `category=off_by_one`, `category_source=rule_strong`, concepts, `last_refreshed_by_attempt_id`). This leg needs the real runner: [m3-06](sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner`, build tags `e2e,runner`), green in CI's Linux **`judge-runner-e2e`** job; on macOS run it against the fake runner (`internal/judge/runnerfake`, scripted to return the edge-only failure) or the spike's multipass VM.
- [ ] The DSA `mistakes.prefill` table matches t4 §6.3 (golden updated, content lint green).
- [ ] The precedence table passes, a learner PATCH locks category and concepts, and weak suggestions never move the weak area.
- [ ] Judge-path anchoring golden rows pass (timeout at the deadline; in-flight pass before give-up; give-up Miss).
- [ ] **The judge stats projection replays equal** (drop-and-replay and N permutations), and judge-checked % counts only `auto ∧ checked`, `null` before any judged conclusion.
- [ ] `/public/stats` shows judge-checked % only; the gateway P10 and assessment shape tests refuse judge stats, arena and Run counts; AB06 renders the value or "—".
- [ ] The Progress aggregate carries `judgeCheckedPct` and `judgeStats`; mistake `concepts` are withheld while the item is live.
- [ ] NATS golden unchanged, or its infra PR merged before `v1.14.0`.
- [ ] CI green: `go test ./...`, `sqlc diff`, web tests, OpenAPI drift, route enumeration, e2e.

## Release

**Merge only: ships in `v1.14.0`**, cut by [m3-13](sprint-m3-13.md). Do **not** tag.
- **Consumers and producer in one tag:** these consumers ship in the same tag as M3-08's new `problem_solved` fields. That's safe because the fields are additive on existing subjects (decoders stay tolerant, the envelope is append-only) and judged conclusions only start after `JUDGE_BASE_URL` is set post-tag (T-2 dark launch, rollout §2.2). During the rollout an old review pod may consume a new event and open a mistake without the pre-fill; that's harmless (the learner picks the category).
- Infra: none expected (task 9). No new pod, no env, no NetworkPolicy change.

## Definition of Done

CI green (incl. `sqlc diff`, the replay golden and permutation tests, the allowlist tests) · squash-merged to `main` (no tag) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board row; M3 stays 🔄; the public-dashboard row **P7 (checked)** → done, ships in `v1.14.0`; **P8** → "authed data done (`v1.14.0`); public exposure pending owner item (ADR-0027 §7 vs AB06)" — **not** done) · Decisions-log lines for the precedence storage, the `null` semantics, the P8 grain, and "**departure from ADR-0027 §7**: judge stats stay authed-only in M3 because the frozen AB06 has no slot; pending the owner's choice between amending ADR-0027 §7 and revising AB06 + widening the P10 allowlist" · the owner open item recorded.

## Risks / watch-outs

- **Commutativity of new projections.** `proj_judge_stats` must stay additive: no `SET x = EXCLUDED.x`, no read-then-write. The permutation property test is the guard.
- **Double-applying a pre-fill on redelivery.** Everything runs inside the inbox-claimed transaction; a same-`attempt_id` refresh under a new `event_id` is idempotent because `applyPrefill` is a pure function of the stored state and the hint.
- **Clobbering the learner.** The `learner` source locks; test it for category **and** concepts, including a re-open after close.
- **"0%" read as a claim.** `null` until the first judged conclusion; AB06 keeps "—".
- **Public scope vs ADR-0027 §7.** ADR-0027 §7 ("What each course shows") / t1 §9 put counted submits, first-submit acceptance and languages on the public course card; AB06 has no slot for them, so this plan keeps them authed-only in M3. That **departs from an accepted ADR**, so record it as such in the Decisions log and raise the owner open item (amend ADR-0027 §7, or revise AB06 and widen the allowlist); don't mark P8 done and don't ship the widening silently. If the owner decides during the session, follow the decision (an ADR-0027 §7 amendment note, or a follow-up task for the AB06 revision + allowlist edit).
- **Goose version collisions with M3-09/peers.** Take the next free version at rebase in both services.
- **Manifest categories:** an unknown pre-fill category must be dropped, not stored, or a later manifest change leaves orphans in the weak area.
