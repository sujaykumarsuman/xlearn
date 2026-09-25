# Sprint m3-13 — Week/Mistakes/Progress deltas (AB12) + M3 exit tests → v1.14.0

> **Milestone:** M3 — judge plus code grader (**M3-2** Run/Submit close-out and the **M3 exit**) · **Track:** product · **Order:** 55
> **Prereqs:** [m3-12](sprint-m3-12.md) · [m3-10](sprint-m3-10.md) (and through them [m3-08](sprint-m3-08.md), [m3-09](sprint-m3-09.md), [m3-11](sprint-m3-11.md)) · TL baselines from [m3-15](sprint-m3-15.md) (`runner-v1.0.0`) · prod time-limit multipliers from [mi-10](sprint-mi-10.md)
> **Unblocks:** [p-01](sprint-p-01.md) · [m4-01](sprint-m4-01.md) · [ga-01](sprint-ga-01.md)
> **Release action:** **tag `v1.14.0`** (indicative: the next free minor at tag time, [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6; floor after: `1.13.0`) · **then the infra PR `JUDGE_BASE_URL`** (cohort-only; merged after the tag is verified) · any raised `limits.time_ms` rides `v1.14.0` (public content) · **evalpack `v1.0.x`** if the TL re-gate changes a pack (clearing the `timing.json` provisional flags, validated against the new public sha, counts) · a pre-tag NetworkPolicy/ACL infra PR only if task 5 finds a gap
> **Calendar:** late November · owner dogfood (~1 h) after the flip: a post-ship owner event (`ev-m3-dogfood`), non-blocking (D40)
> **Execute with:** [`../prompts/prompt-m3-13.md`](../prompts/prompt-m3-13.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | AB12 deltas: Week grading states + wired touch dots + rollup; Mistakes source chip + concepts; Progress provenance + judge-checked % + judge stats | X | ⬜ |
| 2 | M3 exit tests: cohort provenance, kill switch, denylist sweep | X | ⬜ |
| 3 | Outbox republish runbook + `judge admin outbox republish` (dry run by default) | X | ⬜ |
| 4 | Re-gate pack TLs against `runner-v1.0.0` baselines × prod multipliers: raised `limits.time_ms` in this sprint's xlearn PR (rides `v1.14.0`); provisional flags cleared → evalpack `v1.0.x` | E, X | ⬜ |
| 5 | Pre-tag infra check: ACL PRs merged, NetworkPolicy standing rule (a PR only on a gap) | I | ⬜ |
| 6 | Tag `v1.14.0` (release checklist) | X | ⬜ |
| 7 | `JUDGE_BASE_URL` infra PR after the tag (cohort-only), then live verify + telemetry baseline | I | ⬜ |
| 8 | Record the owner event `ev-m3-dogfood` (the owner's ~1 h dogfood after ship) in status.md; it gates nothing | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the M3 milestone + tag/floor, flag and content rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

**M3 hard entry checklist** ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist), verbatim; this sprint turns judge on for the cohort, so every item is re-read here):

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

*Key (not part of the verbatim list):* rollout step ids `MI-NN` are **not** sprint ids `mi-NN`. MI-4 = [mi-14](sprint-mi-14.md) · MI-5, MI-5a = [mi-03](sprint-mi-03.md) · MI-7 (N3) = [mi-06](sprint-mi-06.md) + the ≥ 24 h re-check in [l-01](sprint-l-01.md) · MI-8 = [mi-02](sprint-mi-02.md) · MI-9 = [mi-07](sprint-mi-07.md) + the evalpack ImagePolicy in [m3-07](sprint-m3-07.md) · MI-10 = [spk-01](sprint-spk-01.md) + [spk-02](sprint-spk-02.md) · MI-11 = [mi-09](sprint-mi-09.md) + the 2026-10-24 host window · MI-11a = [mi-08](sprint-mi-08.md) · MI-12 = [mi-10](sprint-mi-10.md) · MI-13 = [m3-07](sprint-m3-07.md) · T25/T26 = [m3-01](sprint-m3-01.md) · 14 packs = the owner's pack event (prepared by [m3-02](sprint-m3-02.md)) · `account.role` = [m1-02](sprint-m1-02.md) · AB07–AB12 = [ds-m3-01](sprint-ds-m3-01.md) and [ds-m3-02](sprint-ds-m3-02.md) merged (the merge is the freeze, D40).
*Key:* the MI-8 NATS item is read at the live stage — `host-verify --cluster --nats-stage=n3` (no `legacy` connection) until N4 lands, `--nats-stage=n4` (`auth_required: true`) once [mi-11](sprint-mi-11.md) has run. N4 is MI-15 hygiene, not an M3 gate ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §3 NATS row).
*Key:* the weekly-image date is read in hPanel by the owner **before launch** (the prompt's "Before you launch (owner)" list, D40); the session records the date from that attestation and never waits for it mid-run.

**Sprint gates:**

- [ ] Every sprint that rides `v1.14.0` is merged: [m3-08](sprint-m3-08.md), [m3-09](sprint-m3-09.md), [m3-10](sprint-m3-10.md), [m3-11](sprint-m3-11.md), [m3-12](sprint-m3-12.md) (`git log origin/main`).
- [ ] `v1.13.0` live ([m3-07](sprint-m3-07.md)): judge Ready, `judge admin status` shows the stamped items evaluable, `JUDGE_BASE_URL` still **unset** on the gateway.
- [ ] m3-08's infra ACL PR (practice's durable on `XLEARN_JUDGE`) is merged, and m3-10's too if its durables changed (task 5 re-checks before the tag).
- [ ] `runner-v1.0.0`'s per-profile TL baselines are published ([m3-15](sprint-m3-15.md)) and mi-10's per-language prod multipliers are recorded (task 4).
- [ ] Parallel sessions: no peer tag or open PR claims `v1.14.0`, and no open peer PR edits `internal/gateway/{aggregate,bff,progress,mistakes}.go`, `web/src/screens/{Week,Mistakes,Progress}.tsx` or `web/src/components/ProgressViews.tsx` (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents).

## Goal

**Close M3.** Put judge evidence where the learner reads their progress (AB12: provenance and judge-checked % on Week, Mistakes and Progress), prove the M3 exit in compose, finalize the pack time limits against the real runner, cut **`v1.14.0`**, and then **turn judge on for the owner and tester cohort** with the `JUDGE_BASE_URL` infra PR and verify it live. The owner dogfoods on production afterwards (a post-ship owner event, non-blocking).

**M3 exit** ([rollout §3](../rollout-plan.md)): packed items execution-graded with provenance (owner and tester cohort); kill switch tested; denylist test green.

Sources: [rollout §3 (M3 exit), §4 M3, §5, §7 (`v1.14.0`)](../rollout-plan.md#7-indicative-tag-timeline); [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (T-2/T-3, kill switches), §4.4 (M3 row: "the kill switch is an M3 exit test; nothing is re-graded"), §6; [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2 (ACL + NetworkPolicy standing rules), §5 (TR-STEAL, TR-QUEUE); [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (changed v1 boards: Week, Mistakes, Progress), [t4 §6.3–§6.4](../research/t4-judge-contract.md) (source precedence, concepts), §14 (outbox republish); [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P7 (judge-checked %).

## Scope

**In**
- AB12 on Week, Mistakes and Progress, including the gateway aggregation behind them (additive read fields only; no migration).
- The M3 exit e2e suite in compose (real-runner lane, `-tags e2e,runner`).
- `docs/runbooks/judge-outbox-republish.md` and the audited `judge admin outbox republish` verb it drives.
- The pack TL re-gate: measured in `xlearn-evalpack` (E); any raised `limits.time_ms` in the **public** item files of this repo (X, content-only, rides `v1.14.0`); then the evalpack PR clearing the provisional flags → an evalpack `v1.0.x` patch.
- The pre-tag ACL/NetworkPolicy check, the `v1.14.0` tag, the `JUDGE_BASE_URL` infra PR and its live verify; recording the owner dogfood as a post-ship owner event (`ev-m3-dogfood`, task 8).

**Out**
- The pilot course and go-race → P ([p-01](sprint-p-01.md), [p-02](sprint-p-02.md), [p-03](sprint-p-03.md)).
- Platform AI, provisional/dispute UI, the "xLearn AI paused" badge made live → M4 ([m4-01](sprint-m4-01.md) … [m4-07](sprint-m4-07.md)).
- Judge for every account (the T-1 default flip) → GA ([ga-01](sprint-ga-01.md), [ga-02](sprint-ga-02.md)).
- N4 and the `xlearn` egress policies themselves → [mi-11](sprint-mi-11.md) (this sprint only checks them).
- The public profile's judge-checked % (already allowlisted by [m3-10](sprint-m3-10.md) / [m2-03](sprint-m2-03.md)); only verified here.

## Tasks

### 1 · AB12 deltas [X]

Boards: `design-system/screens/v2/AB12-week-mistakes-progress.html` (frozen; authoritative for copy and layout), superseding v1's [`Week.dc.html`](../../../design-system/screens/Week.dc.html), [`Mistakes.dc.html`](../../../design-system/screens/Mistakes.dc.html) and [`Progress.dc.html`](../../../design-system/screens/Progress.dc.html) for these parts (t4 §8 "Changed v1 artboards").

**Week — wired touch dots + provisional state.**
- **Touch dots** are a placeholder since v1: `buildUserState` in `internal/gateway/aggregate.go:120` sets `emptyTouches()`. Fill them **in place** ([ADR-0013](../../adr/0013-bff-week-aggregation-userstate-contract.md): same fields, no client contract change) from review: per problem, the five levels' `dueDate` and `result` (`none | pass | fail | due | mock`) and `currentTouch`.
  - Source: a batched review read for the week's problem ids. If `main` has none (check what m2-04/m2-05 added first), add `GET /revisions/ladders?ids=…` to review (JWT, account-scoped, read-only sqlc query on `revision_item` + `touch_result`, course-scoped by `path_slug`) and call it from `handleGetWeek` beside `weekPracticeStates`, in parallel, each with its own timeout.
  - Degrade honestly: if review fails, keep neutral dots and don't cache that response (the existing `populated` rule, extended).
- **Provisional state:** two **additive** keys per problem in `userState.problems[id]` (ADR-0013 freezes the existing keys; new keys are additive): `attemptState` (`awaiting_evaluation | provisional | self_grade_pending | null`) and `provenance` (`{gradedBy: self|auto|ai|override, trust: checked|honor}` for `lastOutcome`), from practice's `GET /state?ids=` (extend its response additively if m3-08 didn't).
- `web/src/screens/Week.tsx`: `TouchDots` render the real states with an `aria-label` that reads them ("2 of 5 passed; next due 12 Dec") plus AB12 W1's tooltip ("Day 7 · passed Oct 9") and legend. The row chip uses AB12 W2's copy: "Grading…" (`awaiting_evaluation`), and **"Needs your grade"** (`self_grade_pending`, `ds-badge--warn`, linking to the Workspace). The provenance mini-chip sits beside the grade chip, in AB06's vocabulary ("judge", "self · honour"). W3's "Provisional" chip is flagged **from M4**: type it and render it from the key, but M3 never produces it.
- **Week rollup (AB12 W4):** "Core 5 · concluded 4 · passed 3 · judge-checked 3 of 4" + the difficulty split. *Passed* means grade ≠ Miss (t4 §11.4 #4). *Judge-checked* counts concluded rows whose `provenance` is `gradedBy=auto ∧ trust=checked`. Compute it from the same `userState` keys; if the aggregation already sums per week, use an additive week-level field instead.
- `web/src/lib/curriculum.ts`: types for the new keys; `docs/architecture/api.md` + `openapi.yaml` (the drift test fails otherwise).

**Mistakes — source chip + concepts** (`web/src/screens/Mistakes.tsx`, `web/src/lib/mistakes.ts`). Copy comes from AB12 M1–M5; **the board wins** over this summary.
- Per entry, a **source chip** from m3-10's `categorySource` with AB12 M1's labels:
  - **You** (`learner`; locks the field);
  - **Judge** (`rule_strong`);
  - **Suggested** (`rule_weak`; a ghost / dashed chip). The learner accepts it through the existing `PATCH /mistakes/{id}`, which flips it to **You**;
  - **xLearn AI** (`analyzer`; M4, typed now, flagged "from M4").

  The M1 tooltip states the precedence (learner > strong rule > analyzer ≥ 0.6 > weak rule, [t4 §6.3](../research/t4-judge-contract.md)). The M3 edit affordance reads "Suggested by the judge: … [Use suggestion]", from `categorySuggested`. M5: before any judge signal, v1 parity, all **You**, no empty chips.
- **Concepts to revise:** ≤ 3 chips per entry (`concepts`, `conceptsSource`), primary first, linking to `/:course/concept/:slug`.
- **Withheld concepts vs AB12 M2's lock:** the BFF strips the primary concept of an item with a live attempt or a due or live touch (t4 §6.4 via `withhold()`), and the SPA must never infer it. To render M2's `xl-lock` "Concepts hidden while this problem's review is due", add an **additive boolean** `conceptsWithheld` in `internal/gateway/mistakes.go`, set when `withhold()` stripped a concept. It reveals only that the learner's own item is live, which Today and Revision already show, and never the concept. The lock renders only from that marker; without it, the concept is simply absent. Test it: sentinel concepts never appear, and the marker is present exactly on live items. Record the marker in the decisions log; it lands with this sprint's PR (D40: no review stop). If the owner rejects it later, a follow-up PR drops the marker and renders nothing, and the M2 frame is flagged for a follow-up design PR.
- m3-10 already exposes `categorySource`, `categorySuggested`, `concepts` and `conceptsSource` through review's `GET /mistakes` and the gateway passthrough; verify them, and add them additively only if one is missing.
- The weekly weak-area banner keeps its layout; its counting rule (weak rules don't count) is server-side (m3-10). Add AB12 M4's footnote ("2 suggested categories aren't counted — confirm them to include them.").

**Progress — provenance, judge-checked % and judge stats** (`web/src/components/ProgressViews.tsx`, `web/src/screens/Progress.tsx`, `web/src/lib/progress.ts`).
- The grade mix and phase completion carry **provenance** per AB12 P1: segments judge · self · honour · AI · override, with a legend; AI and override are M4 and typed now. The source is m2-02's **`proj_outcome_mix_v2`** (`graded_by`, `trust`), which m2-05's `outcomeMix` already reads. If m2-05 already renders provenance chips on Progress, extend them; don't duplicate.
- A **Judge-checked %** tile from m3-10's **`judgeCheckedPct`** (`auto ∧ trust=checked` over first-attempt conclusions, P7), with AB12 P1's ⓘ copy. While it is `null`, the tile shows "—" with AB06 F3's tooltip wording (AB12 P3). Honour results are labelled (P4).
- A **Judge stats** tile (AB12 P2, P8) from m3-10's **`judgeStats`**: counted submits, first-submit acceptance, the language split, and the note "Arena submits and Runs aren't counted here." It is **hidden before any judge-graded result** (P3).
- The course Progress aggregate (`internal/gateway/progress.go`, `GET /api/paths/{slug}/progress` and its DSA alias) already carries `judgeCheckedPct` and `judgeStats` from [m3-10](sprint-m3-10.md) task 8. Verify and consume them, and **don't add a second judge-checked field**. Add only what AB12 still lacks (e.g. a provenance breakdown for phase completion), as additive fields.
- `ProgressViews` is shared with the public `UserDashboard.tsx`: keep the new props optional so the public page renders only what `/public/stats` allows (judge-checked % only; no judge stats, arena or Run counts — P10 allowlist test stays green).

**Tests:** gateway unit tests (touches filled / degraded, additive keys, the `conceptsWithheld` marker, progress fields consumed), review handler + store test for the ladder read, vitest for each AB12 frame (W1–W4, M1–M5, P1–P4), the OpenAPI drift test; `sqlc generate` + **`sqlc diff`** clean.

### 2 · M3 exit tests [X]

`internal/e2e/m3_exit_test.go`, tagged **`-tags e2e,runner`**. The cohort solves need real execution, so it uses [m3-06](sprint-m3-06.md)'s real-runner lane: compose with the privileged `runner` profile (local only), gateway, practice and judge with the dev fixture-content overlay, the fixture pack (`internal/judge/testdata/pack`), and `JUDGE_BASE_URL` set on gateway and practice. It runs in CI in m3-06's Linux **`judge-runner-e2e`** job, next to m3-12's `judge_dock_test.go`; CI's plain `e2e` job has no runner. On macOS, use the spike's multipass VM or rely on CI. Accounts come from m1-04's CLI in compose: `owner`, `tester`, `learner`.

1. **Cohort, execution-graded with provenance.** Owner and tester each solve a packed item in **Go, C++ and Python** (one language per item is enough): Submit → runner → evaluation → practice conclusion `graded_by=auto, trust=checked`; `problem_solved` v2 carries `trust`, `anchor_at`, `language`; Week shows `provenance`, Progress's judge-checked % moves, a below-clean pass opens a pre-filled mistake with a source chip (m3-10). The **learner** account on the same item gets the self path (picker) and every judge route answers 404 (T-3, presence by config).
2. **Kill switch; nothing re-graded** ([ADR-0034 §4.4](../../adr/0034-v2-release-labelling-gating-and-rollback.md) M3 row). With judge-graded history present, snapshot `practice.outcome`, the `problem_solved` / `touch_concluded` outbox rows, judge's `evaluation` count and `GET /api/progress`. Also leave **one judged course attempt open** (started, one submit, not concluded). Then:
   - restart gateway and practice with `JUDGE_BASE_URL` **unset** → packed items show the self path, judge routes 404, badges gone (AB11 F6); snapshots unchanged; a new self-path attempt concludes as v1 (`self@1`). **The open judged attempt** goes to `self_grade_pending` through m3-08's kill-switch sweep, and its picker is **capped at `ceiling`** (the capped pick concludes `graded_by=self`, `resolution=self`). It is never an automatic Miss, whether from the sweep or at its deadline;
   - set it again → nothing re-graded (no new evaluation for old attempts, no new signal); snapshots still equal apart from the new attempts;
   - the **grading override** `GRADING_OVERRIDE=self` on practice (m3-08's T-2 env; ADR-0034 §2) → the same assertions, including a fresh open judged attempt → `self_grade_pending` (capped);
   - judge's own **L15 kill switch**: restart judge with **`JUDGE_ADMISSION=closed`** (m3-05/m3-14) → `503 evaluation_unavailable{fallback: self}` for new work while queued jobs drain to terminal evaluations. `GET /api/judge/status` reports `state=off`, which the SPA renders as AB11's **"Auto-grading off"** (m3-12's vitest covers the render). Assert what [m3-08](sprint-m3-08.md) specifies for an attempt left open under L15. It must never be an automatic Miss for a learner who couldn't submit. m3-08's sweep names only practice's `JUDGE_BASE_URL` and `GRADING_OVERRIDE`; if L15 isn't covered, **stop and report it against m3-08** instead of patching practice here.
3. **Denylist green.** m3-09's per-endpoint, per-state denylist test is in CI and green. The e2e adds a sweep. It covers every judge-related BFF response captured in (1) and (2): submissions, Runs, drafts, arena history/diff, `judge/status`, week, mistakes and progress. No response may contain any of `expected, args, input, case_id, anchors, key, aliases, seed, gen, test_name` (outside the allowed Run/sample views). None may contain the fixture's hidden-case **sentinel strings**, which m3-09's compose leg already expects the fixture's hidden cases to carry. Reuse them. If they're missing, plant one unique token per synthetic hidden case in `internal/judge/testdata/packsrc`, then regenerate `pack/` with `internal/packspec/fixture_test.go -update`. Never hand-edit `pack/`.
4. Arena-never-counts is m3-12's test; re-run it here as part of the suite.

Record the run (date, commit, pass counts) in the sprint file; the prod side of the exit is task 7's live verify after the flip (the owner's dogfood follows as a post-ship owner event, which task 8 records).

### 3 · Outbox republish runbook [X]

**Why:** `XLEARN_JUDGE` is the only stream with an age limit (512 MiB · **14 d · Discard Old**, [ADR-0035 §1](../../adr/0035-v2-operations-nats-auth-limits-capacity.md)). If practice (or its durable) is down longer than 14 d, unconsumed `evaluation_completed` messages age out.
- **The primary heal is automatic:** practice's **pull reconciler** reads judge's tables directly. [m3-08](sprint-m3-08.md) also added the **course and touch watermark drain**: before any deadline conclusion, practice drains to `max_seq`, so a lost message (even the **last** evaluation of an attempt) becomes a visible gap. The pass is then pulled and graded rather than concluded as a wrong Miss. t4 §14 lists the reconciler first.
- **Republish is the secondary path.** It speeds convergence for open attempts that would otherwise wait for their deadline drain. It also covers a stuck or deleted durable, a stream that hit `MaxBytes` (Discard Old), and any future `XLEARN_JUDGE` consumer that has no pull path.

- **Verb** (the sanctioned manual path is an admin CLI via `kubectl exec`, D33 — never hand SQL writes): `judge admin outbox republish --subject xlearn.judge.evaluation_completed --since <RFC3339> [--until <RFC3339>] [--apply]` in m3-14's `judge admin` (`internal/judge/admin`).
  - **Dry run by default:** it prints the row count, the byte estimate vs `MaxBytes`, and the oldest/newest ids. It changes nothing.
  - `--apply` resets `sent_at = NULL` in batches of 500 (the relay then re-publishes at `RELAY_INTERVAL` 1 s) and writes an `admin_audit` row.
  - Tests: a unit test, plus a compose test that consumes, stops practice, republishes and restarts → practice's inbox dedupes on `event_id` (a republish of consumed rows is a no-op).
  - **Scope note:** the register asks for the runbook only, so this mutating verb goes beyond it. Record that in the decisions log with the reason (D33: no hand SQL writes on prod), and keep it dry-run by default.
- **Runbook** `docs/runbooks/judge-outbox-republish.md` (format of [`projection-rebuild.md`](../../runbooks/projection-rebuild.md)):
  1. **When:** practice or its `XLEARN_JUDGE` durable was down, deleted or stuck longer than the 14 d MaxAge, or the stream hit `MaxBytes` (Discard Old). **First let the pull reconciler and the watermark drain heal** (they need no action). Republish only if attempts stay `awaiting_evaluation` or gapped after practice is back, or if a consumer without a pull path is affected. Practice's own streams need no republish: they are `Discard=New` with no age limit, so unsent rows stay in the outbox.
  2. **Detect (read-only):** the durable's delivered stream seq below the stream's `first_seq` (NATS `/jsz?consumers=true`, read node-locally the way `host-verify --cluster` reads it); practice attempts `awaiting_evaluation` > 10 min or with `facts_gap_since` set; `judge admin status`.
  3. **Order:** if practice is still down, republish **before** it returns, so its consumer finds the messages first; if it's already back, republish at once.
  4. **Run:** `ssh sujaykumar-vps 'k3s kubectl exec -n xlearn deploy/xlearn-judge -- judge admin outbox republish --subject xlearn.judge.evaluation_completed --since <RFC3339>'` (dry run), then the same with `--apply` (m3-14's invocation pattern).
  5. **Verify:** consumer pending → 0; no `awaiting_evaluation` or gap older than 30 s; no new `event_dead_letter` rows.
  6. **Late facts (rare with the drain):** an attempt already concluded (e.g. by the kill-switch sweep, or a drain that ran while judge's rows were incomplete) stays concluded (INV-9); practice stores the late fact and ignores it. List them with a **read-only** query (`BEGIN READ ONLY`), record them in `docs/v2/status.md`; the Day-1 revision is the re-attempt (D15).
  7. **Never:** purge or delete stream messages, use the offline ops seed (the relay publishes with judge's own nkey), or edit rows by hand.

### 4 · Re-gate pack TLs [E, X]

The time limit lives in **two places**. Each rides its own release:
- **The limit itself is public:** `limits.time_ms` in the item's `item.json` in **this repo** (`curriculum/courses/dsa/items/<id>/`). Judge and practice both embed it. It is **content-only**, outside `contract_hash` ([m3-01](sprint-m3-01.md) canon), so changing it moves `content_hash` but causes no `contract_changed` for live attempts. A raised TL is therefore an xlearn change and **rides `v1.14.0`**.
- **The pack holds only the measurement:** `timing.json` in `xlearn-evalpack` (the reference max, the image digest and the `provisional` flag m3-02 set). It never enters the built image; only the manifest's `validated_against` does.

Order: the re-gate runs **before** this sprint's xlearn PR, so any TL change makes the tag.
1. **Measure [E]:** in an `xlearn-evalpack` checkout, re-run m3-02's TL gate (`packlint exec --gate tl`) for **every stamped item and every reference language** (Go, C++, Python), with `--public` pointing at this sprint's branch. The rule is TL ≥ 3 × the reference's max, measured against the **`runner-v1.0.0` profile baseline** ([m3-15](sprint-m3-15.md) docs) × **mi-10's per-language prod multiplier** (floor 1 s). packlint's advisory "TL change vs TLE expectations" must stay green (each wrong solution still gets its declared verdict).
2. **Raise TLs [X]:** for any item that fails, raise `limits.time_ms` in its public `item.json` **on this sprint's branch**, re-run the gate until green, and let it ride this sprint's xlearn PR → `v1.14.0`. Record each change (item, old → new) in the decisions log. **Pack payloads never cross into this repo**: only the public limit changes here.
3. **Finalize the pack [E], after the xlearn PR merges:** an evalpack PR clears every `provisional` flag in `timing.json` and is built `--validated-against` the merged public sha. Tag **evalpack `v1.0.x`** (patch). The existing `>=1.0.0 <2.0.0` ImagePolicy picks it up and judge restarts on the image-volume bump; wait for `judge admin status` to show every stamped item evaluable again. Do this **before the flip** (task 7), so the dogfood runs on final limits.
- `docs/v2/status.md` content-status rows: items stamped per tier per week, TL status "final (runner-v1.0.0 × prod)", any raised TLs (shipped in `v1.14.0`), evalpack version + digest.

### 5 · Pre-tag infra check [I]

Read-only first; a PR only if something is missing (each gap is its own infra PR, merged **before** the tag).
- **ACL** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) standing rule): the `authorization` block in `../infra/infrastructure/messaging/release.yaml` equals the golden rendered from `topology.go` at the commit you will tag — practice's durable on `XLEARN_JUDGE` (m3-08), and m3-10's durables if they changed. If a re-render differs, the owning sprint's ACL PR is missing: open/merge it now.
- **NetworkPolicy** (standing rule): `v1.14.0` introduces two in-cluster HTTP callers: **gateway → judge :8087** (m3-09's BFF) and **practice → judge :8087** (`/internal/contexts/*`, m3-08). Read `ssh sujaykumar-vps 'k3s kubectl get networkpolicy -n xlearn -o yaml'`: judge's ingress (m3-07) must admit both; if [mi-11](sprint-mi-11.md)'s `xlearn` egress policies are live, their matrix must include both. No new NATS caller (practice is already a client with its nkey).
- **Memory:** no new pod in `v1.14.0`; the memory sum is unchanged (still read in the entry gates).

### 6 · Tag `v1.14.0` [X]

Run the release checklist (below). GitHub release title **`v1.14.0 — v2 build · M3-2 Run/Submit`**. Release notes: "judge Run/Submit, arena, the D15/D16/D18 flow; **dark until the `JUDGE_BASE_URL` infra PR, then owner and tester cohort only**", plus the cohort behaviour-change list ([ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) Consequences): full statement at start; hard 45-minute limit with the hint at 15; revealing the solution before a pass = Miss; no re-implement stage; coach use during the attempt caps at Assisted; ahead-of-schedule items open the arena; the pattern chip is hidden during a live attempt. Non-cohort accounts keep v1 behaviour until GA.

### 7 · `JUDGE_BASE_URL` infra PR (after the tag), live verify, telemetry baseline [I]

One PR in `../infra`, **its own task, never folded into the tag**, opened only after task 6's verification:
- `apps/xlearn-gateway.yaml` env: `JUDGE_BASE_URL: http://xlearn-judge.xlearn.svc.cluster.local:8087` (the `*_BASE_URL` pattern), plus `JWT_AUD_JUDGE: judge` if m3-09's config reads the audience from env with no default (the `JWT_AUD_*` pattern).
- `apps/xlearn-practice.yaml` env: the same `JUDGE_BASE_URL` — practice gates its judge path on it and calls judge's internal context endpoints (m3-08 task 7). Confirm every reader with `grep -rn JUDGE_BASE_URL internal/ cmd/` before writing the PR.
- **Consumers before producers:** judge emits `evaluation_completed` only once submissions reach it, i.e. after this PR (the dark-producer-behind-a-T-2-switch pattern, [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md)). Merge only once practice's durable on `XLEARN_JUDGE` is bound and review/assessment have consumed a v1.14.0 `problem_solved` without dead letters.
- **After the merge (live verify; with task 2's exit suite, the evidence for M3 ✅):** gateway and practice pods restarted with the env (read-only `ssh sujaykumar-vps`: both Deployments carry `JUDGE_BASE_URL` and have rolled out); for the owner `GET /api/judge/status` → 200 and a packed item shows Run/Submit, checked through an already-signed-in browser session if the session has one (an agent never enters credentials), else recorded as "owner login smoke pending" in status.md's pending-smoke notes; a non-cohort account (if one exists) still gets the self path.
- **Telemetry baseline:** right after the verify, read `judge admin status` and the telemetry rows (steal, quiet re-runs, queue waits) for TR-STEAL / TR-QUEUE and record them. A re-read after the owner's dogfood is on demand (D34).
- **Rollback (R-a):** revert this PR — the permanent kill switch; nothing is re-graded (task 2 proved it).

### 8 · Record the owner dogfood event `ev-m3-dogfood` [X]

The session adds `ev-m3-dogfood` to status.md's owner events (when: after the flip; prepared by m3-13) as part of its status update. The dogfood itself is a **post-ship owner event**, not a session step (D40): the session doesn't ask for it or wait for it, and it **gates nothing**: _Overall_ ✅ and M3 ✅ rest on the session's own evidence (task 2's exit suite, task 6's checklist, task 7's live verify). Anything the owner finds becomes a follow-up PR.

About 1 h on production, after the flip. The owner:
- solves 3–5 pilot-pack items end to end, at least one each in Go, C++ and Python; includes one WA → pass, one give-up (Miss, solution unlocks, pre-filled mistake with a source chip) and one hinted solve (Assisted);
- does an arena pass on a packed item and a Mark studied on an unpacked one, and checks the course state didn't move;
- checks Week (provenance, real touch dots, the rollup), Mistakes (source chip, concepts), Progress (judge-checked %, judge stats) and the badges;
- files issues (label `m3-dogfood`); blockers are fixed forward in a `v1.14.x` patch (R-c), never by moving the tag.

If a CLI-minted tester exists (only after MI-5b, ADR-0033 §11), the tester repeats one item; otherwise the tester leg of the M3 exit rests on task 2's compose run. The TR-STEAL / TR-QUEUE baseline is task 7's; any later session re-reads `judge admin status` and the telemetry rows on demand after the dogfood (D34).

## Acceptance criteria

- [ ] AB12 implemented (W1–W4, M1–M5, P1–P4; the board's copy): Week shows real touch dots, the grading states ("Grading…", "Needs your grade") and the judge-checked rollup; Mistakes shows the You/Judge/Suggested/xLearn AI source chip and concepts (the lock only from the server's `conceptsWithheld` marker); Progress shows provenance, judge-checked % (`judgeCheckedPct`) and judge stats (hidden before any judge-graded result). `sqlc diff` clean; the OpenAPI drift test green.
- [ ] **M3 exit** (`-tags e2e,runner`, green in `judge-runner-e2e`): packed items execution-graded with provenance for the owner and tester cohort (compose e2e for both; on prod, task 7's live verify for the owner); **kill switch tested** (unset `JUDGE_BASE_URL` / `GRADING_OVERRIDE=self` → self path, nothing re-graded, an open judged attempt → `self_grade_pending` capped at `ceiling`, never an automatic Miss; `JUDGE_ADMISSION=closed` → 503 + drain, `state=off`); **denylist test green** (m3-09's test + the sentinel sweep).
- [ ] Every stamped item TL-gated against `runner-v1.0.0` × prod multipliers; any raised `limits.time_ms` merged in this sprint's xlearn PR and shipped in `v1.14.0`; no provisional TL flag left (evalpack PR validated against the merged public sha, `v1.0.x` tagged) before the flip.
- [ ] The republish runbook is merged and its dry run and `--apply` exercised in compose.
- [ ] `v1.14.0` live and verified (checklist below); the `JUDGE_BASE_URL` PR merged after it; judge on for the cohort only.
- [ ] Task 7's live verify recorded (the owner checks through an already-signed-in browser session, or "owner login smoke pending" in status.md), the telemetry baseline read and recorded, and `ev-m3-dogfood` added to status.md's owner events (post-ship; the dogfood itself doesn't gate this sprint).

## Release

**Tag `v1.14.0`** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6: M3-2 Run/Submit, arena, the D15/D16/D18 flow; gate state after: `JUDGE_BASE_URL` set, judge features cohort-only; rollback floor after: **`1.13.0`**), followed by the `JUDGE_BASE_URL` infra PR (task 7). Raised `limits.time_ms` values (task 4) ride this tag as public content; the evalpack `v1.0.x` that clears the provisional flags lands after the xlearn PR merges and before the flip.
Release checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) §6, verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag:** ACL PRs — practice's `XLEARN_JUDGE` durable (m3-08) and any m3-10 durable (task 5). New service — n/a (judge's image and policy landed with `v1.13.0`). Contract / erase / GA — n/a: expand-only migrations, so no snapshot is required (the M3 checklist's weekly-image item, from the owner's before-launch attestation, and its `host-verify` item were read in the entry gates). M6 — n/a. Login smoke (the after-tag line, and task 7's owner checks): through an already-signed-in browser session if the session has one; an agent never enters credentials, so otherwise run the credential-free checks and add an "owner login smoke pending" row to status.md's pending-smoke notes. New callers — gateway → judge and practice → judge (task 5). Memory — no new pod. Extra after-tag reads: judge routes still 404 for the owner (presence by config until task 7); practice's durable bound on `XLEARN_JUDGE`; no new `event_dead_letter` rows in review/assessment; `judge admin status`. Right before pushing the tag, **re-check for peer and owner messages** (a hold overrides the ship).
**Record** in [`../status.md`](../status.md): `M3-2 → v1.14.0 → floor 1.13.0 → snapshot n/a`; flags: `JUDGE_BASE_URL` **set** on gateway + practice (permanent kill switch, task 7's PR #), the grading override (permanent kill switch, unset; name per m3-08), the judge T-3 cohort gate (non-kill; owner M3; removal at GA, [ga-01](sprint-ga-01.md)); the evalpack version/digest (task 4) and any TLs raised in `v1.14.0`.

## Definition of Done

CI green (incl. `sqlc diff`, the OpenAPI drift and denylist tests) · the M3 exit suite green in compose · pre-tag infra gaps (if any) merged · `v1.14.0` tagged, deployed by Flux (no hand `kubectl`) and verified by the checklist · `JUDGE_BASE_URL` infra PR merged after the tag · evalpack patch tagged (task 4) before the flip · task 7's live verify and telemetry baseline recorded · statuses updated (this file + [`../status.md`](../status.md): board row, **M3 → ✅** on the session's own evidence, tag/floor row, flag inventory, content-status rows, the telemetry baseline, owner event `ev-m3-dogfood` (post-ship, non-blocking) and any "owner login smoke pending" note) · local `main` synced in every repo touched · notable calls in the decisions log (the ladder read endpoint, the additive `userState` keys, the `conceptsWithheld` marker, the republish verb as beyond the register's runbook-only scope, every raised TL).

## Risks / watch-outs

- **Steal spikes skew timings** (TR-STEAL): judge's telemetry rows carry steal, canary medians and quiet re-runs. Read them after the flip (task 7's baseline) and on demand after the dogfood; if TR-STEAL fires (quiet re-runs > 5% a day, 24 h steal p95 > 10% on ≥ 3 of 7 days), R0 first (lower the breaker), then R2 ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md)).
- **Flipping before consumers are bound** would leave `evaluation_completed` unconsumed (it survives 14 d, then ages out). Task 7 checks the durable first; the runbook covers the long-outage case.
- **TL changes are public content, not pack content.** A raised `limits.time_ms` goes into this repo's `item.json` and must be on the branch **before** the xlearn PR merges, or it misses `v1.14.0`. The evalpack bump restarts judge, so do it before the flip, not during the dogfood.
- **Real-runner lane:** the exit suite is `-tags e2e,runner`. A green plain-`e2e` job doesn't cover it; `judge-runner-e2e` must run it.
- **L15 vs open attempts:** m3-08's kill-switch sweep is keyed to practice's `JUDGE_BASE_URL` / `GRADING_OVERRIDE`, not judge's `JUDGE_ADMISSION`. If an attempt left open under L15 could be concluded Miss at its deadline, that is an m3-08 gap: report it, don't patch it here.
- **The touch-dot fan-out** adds a review call to every Week load: parallel, per-call timeout, honest neutral dots and no caching on failure (ADR-0013's placeholder rule).
- **ADR-0013 freezes `userState`:** the new keys are additive; renaming or retyping an existing key is a contract break.
- **The kill-switch drill runs in compose, not on prod.** Production before task 7 is the "off" state; an R-a on prod is one revert PR if ever needed.
- **Tester leg on prod** needs MI-5b and an existing tester (ADR-0033 §11); otherwise compose carries it — say so in status.md, don't mint a tester just for this.
- **Tag hygiene:** never move or re-push `v1.14.0`; a dogfood blocker is a `v1.14.x` patch.
