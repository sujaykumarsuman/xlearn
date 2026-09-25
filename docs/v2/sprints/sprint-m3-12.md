# Sprint m3-12 — Results dock, Problems, Arena, degradation badges (AB08–AB11)

> **Milestone:** M3 — judge plus code grader (**M3-2** Run/Submit, UI slice 2 of 3) · **Track:** product · **Order:** 54
> **Prereqs:** [m3-11](sprint-m3-11.md) (Workspace shell, lazy CodeMirror, `web/src/lib/judge.ts`) · [ds-m3-02](sprint-ds-m3-02.md) (AB09, AB10 frozen) · [ds-m3-01](sprint-ds-m3-01.md) (AB08, AB11 frozen) · BFF routes from [m3-09](sprint-m3-09.md) · `arena_progress` from [m3-14](sprint-m3-14.md)
> **Unblocks:** [m3-13](sprint-m3-13.md) (AB12 deltas, M3 exit tests, the `v1.14.0` tag)
> **Release action:** **merge only** (ships dark in `v1.14.0`, tagged by [m3-13](sprint-m3-13.md)). Nothing is visible until m3-13's `JUDGE_BASE_URL` infra PR, and then only to the owner and tester cohort (T-3). No infra PR, no flag, no new pod.
> **Calendar:** late November (agent work; no owner time)
> **Execute with:** [`../prompts/prompt-m3-12.md`](../prompts/prompt-m3-12.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Results dock (AB08): Samples · Submission · History · Feedback, every state | X | ⬜ |
| 2 | Problems (AB09): course vs arena done markers, filters | X | ⬜ |
| 3 | Arena (AB10): "Nothing here counts", manual timer, History drawer, spoiler confirm, study mode | X | ⬜ |
| 4 | Degradation badges (AB11) from `GET /api/judge/status` | X | ⬜ |
| 5 | Tests: every AB08 state reachable in compose; arena never alters course progress | X | ⬜ |
| 6 | Visual check against the frozen boards + docs (`api.md`, route list) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + artboard rows; M3 stays 🔄).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB07–AB12 frozen**: [ds-m3-01](sprint-ds-m3-01.md) and [ds-m3-02](sprint-ds-m3-02.md) merged (the merge is the freeze). This sprint builds from `design-system/screens/v2/AB08-results-dock.html`, `design-system/screens/v2/AB09-problems.html`, `design-system/screens/v2/AB10-arena.html` and `design-system/screens/v2/AB11-degradation-badges.html`.
- [ ] **[m3-11](sprint-m3-11.md) merged**: `web/src/screens/Workspace.tsx` (with the dock slot it left empty), the lazy CodeMirror chunk and its `vite:preloadError` reload, drafts autosave, and `web/src/lib/judge.ts` (submit, run, poll, drafts; TanStack Query; 25 s fetch timeout).
- [ ] **[m3-09](sprint-m3-09.md) merged** (BFF): submissions/Runs with `pollAfterMs` + `ETag`, drafts, arena history and diff, the arena progress read and `POST /api/problems/{id}/arena/studied`, the arena reveal route, `GET /api/judge/status`, typed 413 (64 KiB code) and 429s with quota fields, `withhold()` on arena surfaces, presence by config (404 unless `JUDGE_BASE_URL` is set and the cohort passes).
- [ ] **[m3-14](sprint-m3-14.md) on `main`** (shipped in `v1.13.0`): the learner DTO allowlist, arena history, `arena_progress` with `source auto|manual`.
- [ ] **The M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) was recorded green at [m3-11](sprint-m3-11.md)'s start**, and nothing has regressed since (re-run `host-verify --cluster` only if a host change happened in between). The verbatim list lives in m3-11 and [m3-13](sprint-m3-13.md).
- [ ] **No open peer PR** edits `web/src/screens/{Workspace,Problems}.tsx`, `web/src/lib/judge.ts` or `web/src/router.tsx` (parallel sessions: `gh pr list`, `git worktree list`, ListAgents), or the order is agreed.

## Goal

Finish the **judged surfaces** that sit around m3-11's Workspace, so that when m3-13 turns judge on for the cohort, every state a learner can reach renders the frozen board:
- the **results dock** (A2 / AB08) shows every judge outcome: queued, running and settling; accepted; WA; perf TLE; CE on the learner's own code and against the hidden API; RE / MLE; `inconclusive`; `contract_changed`; `not_evaluated_here`; 413, quota and the 429/503 limit family (incl. the L12 budget and L13 breaker). It never shows anything the learner DTO doesn't carry;
- **Problems** (A4 / AB09) carries **two done markers**, course and arena (D10), with filters;
- the **Arena** (A5 / AB10) is untimed study that **never counts** (D10) and is **unrestricted** (D17): a manual timer that's off by default, a read-only History drawer with Copy to editor and Diff, a spoiler confirm that **records the reveal and doesn't cap** the course grade, and study mode with **Mark studied**;
- **degradation badges** (AB11) make judge degradation visible in the product, the D34 substitute for alerts.

Sources: [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (results-dock copy, the arena frames), [t4 §2.6–§2.7](../research/t4-judge-contract.md) (learner DTO, polling), [t4 §3.7](../research/t4-judge-contract.md) (arena mechanics; **its locks and reveal cap are overridden by D17**, t4 §13), [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) §2 and §4 (D10, D17), [PRD R-AR1 (amended)](../../prd/xlearn-v2-prd.md), [rollout §9](../rollout-plan.md#9-artboards-by-milestone) (AB08–AB11).

## Scope

**In**
- web: `ResultsDock` in m3-11's Workspace, `Problems.tsx` (dual markers, filters), the arena screen on the Workspace shell, the badges, `lib/judge.ts` additions (arena, status), and the route list. Outside `web/` only tests and test data: `internal/e2e/judge_dock_test.go`, synthetic learner solutions in `internal/judge/testdata/solutions/<item>/`, and synthetic fixture additions in `internal/judge/testdata/{content,packsrc}` with the built `internal/judge/testdata/pack/` **regenerated** (`internal/packspec/fixture_test.go -update`; never hand-edited), plus adding the e2e to CI's `judge-runner-e2e` job (`.github/workflows/ci.yml`).
- The Problems row routing for the cohort: a course-current packed item opens the course attempt (Workspace); **any** item can open the arena; ahead-of-schedule items go to the arena instead of v1's `counted:false` attempt (t4 §8 behaviour change, cohort only).
- A compose e2e table proving every AB08 state is reachable through the real BFF on the fixture pack.

**Out**
- Week provisional state and touch dots, the Mistakes source chip and concepts, Progress provenance and judge-checked % (AB12) → [m3-13](sprint-m3-13.md).
- The M3 exit tests (kill switch, denylist sweep, cohort provenance), the `v1.14.0` tag and the `JUDGE_BASE_URL` flip → [m3-13](sprint-m3-13.md).
- Workspace frames (cover, Run/Submit, timer ring, give-up modal, concluded cards, self-path variant) → [m3-11](sprint-m3-11.md), already merged.
- AI feedback in the Feedback tab, the "xLearn AI paused" badge made live, provisional/dispute UI → M4 ([m4-06](sprint-m4-06.md)).
- Quiz study mode and race verdicts (A7, go-race) → P ([p-03](sprint-p-03.md)).
- Any BFF, judge or practice change. If a DTO field the boards need is missing, stop and report (it belongs to [m3-09](sprint-m3-09.md) / [m3-14](sprint-m3-14.md)); don't widen the DTO here.

## Tasks

### 1 · Results dock (AB08) [X]

`web/src/components/judge/ResultsDock.tsx` (+ `ResultsDock.test.tsx`), mounted in the dock slot m3-11 left in `Workspace.tsx`. Four tabs, **Samples · Submission · History · Feedback**, evolving the v1 dock (`Problem.dc.html:289-302`). Below 1024 px it is the **Results** tab of m3-11's Statement / Work / Results layout.

**Field names.** The SPA reads [m3-09](sprint-m3-09.md)'s BFF, which re-encodes every judge and practice response in **camelCase** (`internal/gateway/judge_dto.go`, `docs/architecture/openapi.yaml`). t4's research contract is snake_case; don't copy its names. The component's TS types mirror `openapi.yaml`.

**Data.** Only through `web/src/lib/judge.ts` (m3-11): add `useSubmission(id)` and `useRun(id)` polling hooks if m3-11 didn't. Polling follows [t4 §2.7](../research/t4-judge-contract.md):
- `pollAfterMs` backs off 500 ms → 1 s → 2 s → 5 s, sending `If-None-Match` (304 keeps the cached body);
- stop after 5 min with "still grading, we'll show it on Today";
- for **counted** contexts keep polling until `factsThroughSeq ≥ attemptSeq` (up to 60 s after the evaluation is terminal). That window is the **settling** state; only then render the conclusion;
- no SSE (deferred; Traefik buffering, m3-09 risk).

**States and copy** ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) results-dock table; **the frozen AB08 board wins** where the two differ; frame numbers are AB08's from [ds-m3-01](sprint-ds-m3-01.md)):

| State | Trigger in the BFF DTO | Copy |
|---|---|---|
| queued / running / settling (F2–F4) | `status` queued (with `queuePosition`) / running / terminal but `factsThroughSeq < attemptSeq` | "Queued · 1 ahead of you" / "Running hidden tests…" / "Recording your result…" |
| accepted (F5) | `status=passed`, `hidden.passed = hidden.total`, `perf=passed` | "All hidden tests passed · 40/40 · performance ✓" + the bucketed usage line |
| WA (F6) | `status=failed`, `firstFailure.class=WA`, `hidden{passed,total}` | "**Wrong answer:** 37/40 hidden passed. Hidden inputs aren't shown; try edge cases with a custom Run." |
| perf TLE (F7) | `hidden.passed = total`, `perf=failed` | "**Too slow on large inputs:** correctness 40/40, performance failed. Check your complexity against n ≤ 1e5." (the `n` bound comes from the item's public `constraints[]`) |
| CE, own code (F8) | `class=CE` with learner-file `compile.diagnostics` | "Compile error in your code — not counted." + diagnostics at the learner's lines |
| CE, hidden API (F9) | `class=CE` with the fixed hidden-API reason | "Hidden tests don't compile against your code. Check the required signature. (Not counted.)" |
| RE / MLE (F10) | `firstFailure.class` RE / MLE, `hidden{passed,total}` | "Runtime error on hidden tests: 12/40 passed." / "Memory limit exceeded on hidden tests: 39/40 passed." (class only; no stack or stderr for hidden tests) |
| `inconclusive` / close released (F11) | `status=inconclusive` (any platform reason), or practice state back to `attempting` after a released close | "We couldn't get a reliable result. Not counted. [Submit again]" — the button re-submits with a **new** Idempotency-Key |
| `contract_changed` (F12) | `reason=contract_changed` | "This problem was updated while you worked. Not graded." + **[Reload problem]** ("your draft is kept") |
| `not_evaluated_here` (F13) | a step `skipped(not_evaluated_here)` (key steps in Run or arena) | "Checked only in a course attempt or review." |
| 413 (F14) | gateway typed 413 `part_too_large` (64 KiB code) | "Too large to submit (code is capped at 64 KiB)" |
| quota and 429s (F15) | `quota{kind, remaining, limit, resetAt}` on every 2xx; 429 `quota_exceeded`, `queue_full`, `too_many_pending`, `too_fast`, `budget_exhausted` (L12), `too_fast{reason: breaker}` (L13 level 1), `shed{reason: breaker}` (L13 level 2), each with `Retry-After` | AB08 F15's grid: "8 submits left today" (shown at ≤ 10), the reset time for quota, and for L12 and both breaker levels the lines that say **submits still work** (the breaker throttles or sheds only Runs and arena work; counted submits always run) |
| 503 unavailable (F15) | a submit's 503 `evaluation_unavailable{fallback: self}` | "Auto-grading isn't available for this problem right now — you'll pick your own grade when you finish." The Workspace keeps following practice's state (task 4); it never switches the open attempt to a picker by itself |

**Tabs.**
- **Samples:** each Run result per sample or custom input: `status`, `class`, `got`, `sampleExpected` (public), `stdout`, `stderr` (DTO caps: 8 KiB / 2 KiB), compile diagnostics **at learner-file positions only** (hand them to m3-11's editor as lint markers if it exposes the hook).
- **Submission:** the latest counted evaluation: `checks[{key,label,required,met}]`, `hidden{passed,total}`, `perf`, `firstFailure.class`, `trust`, `evaluatedWith`. Nothing else: no case ids, ordinals, timings or stderr exist in the DTO, and the component's TS types admit only the allowlisted fields ([ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) §2, INV-7).
- **History:** this context's evaluation-bearing submissions (seq, time, status, class); Runs are never listed (INV-12).
- **Feedback:** after a counted conclusion, `key` marks and `answerReveal` if the DTO carries them (`Visibility.Concluded`); otherwise the AB08 empty state. AI feedback is M4.

**A11y:** the status line is `aria-live="polite"`; tabs are a proper `tablist` with arrow-key navigation; focus stays in the editor on ⌘↵ / ⌘⇧↵ (m3-11's shortcuts).

### 2 · Problems: dual markers + filters (AB09) [X]

`web/src/screens/Problems.tsx` (today the v1 "practice arena" list, `?practice=1` links).
- **Course-resolved:** use m1-03's `useCourse()` / `/:course/*` routes; no hard-coded `dsa`.
- **Two markers per row (D10):** **course done** (practice state: `status=solved` and the grade chip from `lastOutcome`) and **arena done** (`firstPassedAt` from m3-09's batched `GET /api/paths/{slug}/arena/progress`, `{<itemId>: {done, source, firstPassedAt, submits}}`, with `source=manual` shown as "Studied"). One **batched** arena-progress read for the list, never N calls.
- **Filters** exactly as AB09 F2: a status segment **All · Not started · Course done · Arena done · Review due**; difficulty chips Easy / Medium / Hard; grading chips **Judge-graded / Self-report**; a week select; search by title or `#`. Filter state lives in the URL query (`?status=arena&diff=med`) so a reload or Back restores it. **There is no pattern filter:** filtering by pattern would reveal the pattern of live items, a withhold leak ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule)). A row's pattern chip follows the BFF: `withhold()` removes it for an item with an open attempt or a due or live touch (t4 §8: the chip appears only from the hint stage).
- **Row actions (cohort, judge present):** "Start" / "Resume" on course-current packed items → the Workspace course attempt; "Arena" on every item → task 3. Ahead-of-schedule items offer the arena only (t4 §8 behaviour change); v1's `counted:false` practice link is not shown to the cohort.
- **Presence by config:** when the arena-progress route answers 404 (judge absent, or the account is outside the owner/tester cohort), render **exactly today's v1 list** (no arena marker, no arena link, no badges). Accounts outside the owner/tester cohort therefore see v1 until GA. This departs from AB09 F7 (see Risks).
- Replace the v1 "Ahead · free practice" copy with AB09's.

### 3 · Arena (AB10) [X]

`web/src/screens/Arena.tsx` on m3-11's Workspace shell (same statement column, editor chunk and language picker Go/C++/Python), route **`/:course/arena/:id`** in `router.tsx` (inside `CurriculumShell`; follow m1-03's course routing). The server resolves `context=arena` (`context_id = uuidv5(account, item)`); the SPA never sends an id.
- **Banner:** "Nothing here counts toward your course." (D10). No course timer ring, no grade outlook, no give-up.
- **Unrestricted (D17):** there is **no lock state**. t4 §8 / §3.7's "Finish your review first" frame, the disabled Submit and the withheld Solution/History are **overridden** (t4 §13, PRD R-AR1 amended). The only thing that can be absent is what `withhold()` withholds on every surface (e.g. the primary concept chip during a live touch, t4 §6.4); it renders as absent, never as a lock.
- **Manual timer popover:** client-side only, **off by default**, start / pause / reset, kept in `sessionStorage` per item (wrapped in try/catch); never sent to the server, no events.
- **Submit and Run:** through `lib/judge.ts` with `context: arena`; results in the same `ResultsDock` (key steps show `not_evaluated_here`).
- **History drawer:** read-only list of **arena submits** (persisted; Runs are not, D10), newest first, with status and language. **Copy to editor** (confirm if the editor is dirty; the draft autosave then records it) and **Diff** against the current editor, using `@codemirror/merge` inside the same lazy editor chunk (don't grow the main bundle).
- **Spoiler confirm** on "Show solution": the reveal is **recorded** (`arena_revealed_at` via m3-09's reveal route → `arena_prior` on the later course attempt) and **doesn't cap** the course grade (D17). Copy exactly as AB10; never "caps your course attempt at Assisted" (t4's pre-D17 wording).
- **Study mode + Mark studied:** an arena item with no evaluator for this account (not packed, or a B/C archetype) opens in study mode: editable parts, a revealable reference, and **Mark studied** → `POST /api/problems/{id}/arena/studied` (`source=manual`). A later passing arena submit keeps `firstPassedAt` (m3-14's upsert rule). This needs judge present for the account: the studied and arena-progress routes are judge-backed and 404 without it, so AB10 F9 (judge off → study mode with Mark studied) can't be built. With judge absent the arena screen isn't linked, and a direct visit falls back to v1's practice view (see Risks).
- **Never counts:** no practice call, no attempt, no timer, no `problem_solved`; the arena done marker is the only state it changes (task 5 proves it).

### 4 · Degradation badges (AB11) [X]

`web/src/components/judge/JudgeBadges.tsx` + `useJudgeStatus()` in `lib/judge.ts` reading `GET /api/judge/status` (m3-09: `{state: ok|saturated|breaker|off, breakerLevel?, evaluable, items?{<id>: evaluable|contract_mismatch|unsupported|no_pack}}`). `staleTime` 30 s, refetch on window focus; a 404 means judge is absent for this account → **no badges** (presence by config, AB11 F6).

The mapping follows AB11 F1–F4 (the frozen board wins on copy and placement):

| Badge (copy per AB11 F1) | Shown when | Effect |
|---|---|---|
| **Grading pending · self-report** | `evaluable = 0` (the page banner of AB11 F2/F4), or this item is `contract_mismatch` / `unsupported` / `no_pack` (the row chip of AB11 F2) | informational. An item that is **not evaluable when the attempt starts** is pinned `self` by practice and opens in m3-11's self-path variant (AB07 F12, the only uncapped picker). Nothing switches mid-attempt |
| **Grader busy** | `state=saturated`, `state=breaker` (level 1 or 2), or a `queue_full` 429 | results may take longer; Runs and arena work may be throttled (level 1) or paused (level 2), with AB08 F15's copy in the dock header; **submits still work and nothing switches to self** (the L13 breaker never touches counted submits, [m3-14](sprint-m3-14.md)) |
| **Auto-grading off** | `state=off` (judge's L15 kill switch `JUDGE_ADMISSION=closed`, or judge unreachable) | informational; a submit gets 503 `evaluation_unavailable{fallback: self}` → AB08 F15's 503 copy in the dock |
| **xLearn AI paused** | never in M3 | typed and placed now as AB11's M4 placeholder; [m4-06](sprint-m4-06.md) turns it on |

**The Workspace follows practice, never the badge.** An open judged attempt keeps its judged frames whatever the status says. After a 503 on submit the dock shows the F15 line, and the Workspace moves to AB07 F11 (the `self_grade_pending` picker, **capped** at `ceiling`) only once practice's state reports `self_grade_pending`. It never renders AB07 F12's uncapped picker for an attempt that started judged. Kill switches that remove judge (`JUDGE_BASE_URL` unset, non-cohort) show no badge (AB11 F6).

Placement per AB11 F2–F4: the Problems header banner (plus the per-row chip for a mismatch or no pack), the Workspace HUD and cover note (with "Grader busy" in the dock header), and the course Dashboard (Today) banner / Resume-card chip. Badges are informational, never modal, announce changes with `aria-live="polite"` (AB11 F5) and meet the theme's contrast tokens.

### 5 · Tests [X]

- **Vitest** (`web/src/**/*.test.tsx`), from DTO fixtures in `web/src/test/fixtures/judge/*.json`:
  - one test per AB08 state (the copy above, the tab contents, `aria-live`), plus "a fixture carrying a forbidden key (`expected`, `input`, `case_id`, `test_name`) renders nothing from it";
  - Problems: both markers, `source=manual` label, filters from the URL, the v1 fallback on a 404;
  - Arena: timer off by default, History drawer (Copy to editor, Diff), spoiler confirm posts the reveal and its copy says it doesn't cap, Mark studied, **no lock frame even with a due touch in the fixture**;
  - badges: each `state` / `breakerLevel` / per-item status → the AB11 badge in the table above (breaker → "Grader busy", `off` → "Auto-grading off"), 404 → none; an open judged attempt plus a 503 on submit keeps the judged frames until the practice fixture says `self_grade_pending` (then the capped F11 picker, never F12).
- **Every AB08 state reachable in compose.** `internal/e2e/judge_dock_test.go` is tagged **`-tags e2e,runner`**: it needs [m3-06](sprint-m3-06.md)'s real-runner lane, meaning the privileged compose profile `runner` (local only) plus gateway, practice and judge with the dev fixture-content overlay. It runs in CI in m3-06's Linux **`judge-runner-e2e`** job; extend that job to start the services the BFF path needs if it only starts the runner. CI's plain `e2e` job has no runner. On macOS, run it in the spike's multipass VM or rely on CI. The test drives the real BFF on the fixture pack and asserts the DTO state, and it records each response as the vitest fixture (golden JSON) so the UI and backend agree.
  - **Test data:** learner solutions are m3-06's synthetic ones in `internal/judge/testdata/solutions/<item>/` (Go, C++ and Python). Add the missing ones there: a hidden-API CE (compiles alone but not against the harness, e.g. a wrong signature), an own-code CE, an RE and an MLE. They are **not** the pack's `packsrc/…/submissions/wrong/*`, which are packlint inputs. Any fixture change goes into `internal/judge/testdata/{content,packsrc}`, and then `internal/packspec/fixture_test.go -update` regenerates `pack/` byte for byte. Never hand-edit `pack/`. **Don't edit `fx-001…` items:** a new part changes their `contract_hash` and breaks the m3-06/m3-09/m3-11 goldens. Add a new synthetic item under the next free `fx-NNN` instead, and update any test that counts the fixture's evaluable items in the same PR.

  | State | How compose reaches it |
  |---|---|
  | queued / running / settling | two quick submits (1 running per lane) with a slow-but-correct solution from `testdata/solutions/` |
  | accepted / WA / perf TLE / CE own code / CE hidden API / RE / MLE | the matching synthetic solutions in `internal/judge/testdata/solutions/<item>/`, across Go, C++ and Python |
  | `inconclusive` | stop the compose runner service mid-job (`docker compose stop <runner service>`) → `infra_exhausted` after the retries |
  | `contract_changed` | start a course attempt (practice pins the current `contract_hash`), then **restart judge only** with a variant of the dev content overlay in which one fixture item's contract differs (e.g. a param type), plus a pack variant, regenerated the same way, whose `accepts_contract_hashes` contains the variant's live hash. Without that, packlint check 3 and judge mark the item `spec_mismatch`, which is not evaluable and returns 503. The next submit carries the old pin → judge evaluates it inline as `inconclusive(contract_changed)` ([m3-06](sprint-m3-06.md)) |
  | `not_evaluated_here` | a Run (or arena submit) on the new synthetic `fx-NNN` item, which has a code part plus a `key` part |
  | 413 | a > 64 KiB paste |
  | quota / `too_fast` / `too_many_pending` | L9: a compose override lowering the quota (if m3-14 made it env-configurable), else a seeded counter row; two submits < 2 s apart in one context; three quick Runs |
  | `budget_exhausted` (L12) | seed `judge.job_telemetry` busy-ms rows for the test account in the trailing hour (test-only SQL in compose), or a compose override if m3-14 made L12 env-configurable; then a Run → 429 |
  | breaker level 1 / level 2 (L13) | `judge admin breaker set --level 1 --for 1h` (the `breaker_override` pin) → a second Run within 30 s → 429 `too_fast{reason: breaker}`; `--level 2` → Run and arena → 429 `shed{reason: breaker}`, while a counted submit still succeeds; `GET /api/judge/status` reports `state=breaker`; `judge admin breaker clear` after |
  | 503 `evaluation_unavailable` | restart judge with `JUDGE_ADMISSION=closed` → a submit gets 503 `{fallback: self}` and `GET /api/judge/status` reports `state=off` |
  | `queue_full` | only if L10 is env-configurable; otherwise vitest-only, recorded in the decisions log |
- **Arena never alters course progress** (acceptance), in the same e2e. Before the arena actions, snapshot `GET /api/progress`, the week payload for the item's week, `GET /api/revision/due`, practice's `GET /state/{problemId}` for the item, and the practice outbox row count. Then do an arena pass, a reveal and a Mark studied, and split the assertion:
  - **The reveal is recorded:** `arena_revealed` (`arenaRevealed` in the BFF view) flips false → true **exactly once** (a second reveal leaves `arena_revealed_at` unchanged, D17);
  - **Everything else is byte-equal:** grade, `ceiling`, outcome and every other practice state field, `/api/progress`, the week payload, `/api/revision/due`, and the practice outbox count (no `xlearn.practice.*` event). The arena marker is the only list-level change;
  - **No cap later:** start and pass a course attempt on the same item, then assert its `problem_solved` carries `arena_prior=revealed` and its grade is **not** capped (e.g. Clean on a fast, hint-free pass).

  Don't "fix" a failing snapshot by not recording the reveal; D17 requires the record.
- `npm --prefix web run {typecheck,lint,test,build}`; `go test -race ./...` (unchanged Go code must stay green); `go test -tags e2e,runner ./internal/e2e/...` with the compose `runner` profile up; `git checkout -- web/dist/.gitkeep` after the build.

### 6 · Visual check + docs [X]

- Eyeball Problems, the Workspace dock (each state from the fixtures) and the Arena against the frozen boards at **1440 px and 390 px** (a temporary `web/vite.mock.config.ts` stubbing `/xlearn/api/*` works without the backend; **delete it before committing**). Attach the screenshots to the PR.
- `docs/architecture/api.md`: the SPA routes (`/:course/arena/:id`) and which BFF routes each screen reads. SPA routes are documented there only. The `withhold()` route-enumeration test ([m1-06](sprint-m1-06.md)) walks the BFF's `apiRoutes()`, not SPA routes. **Verify** that m3-09's arena BFF routes are in it with policy `applied`; don't add anything there (no BFF change in this sprint).
- `docs/v2/status.md` artboard rows AB08–AB11 → "implemented (m3-12, PR #)"; first mark AB08–AB11 "frozen (merged, PR #N, date)" only if the ds-m3-01/ds-m3-02 sessions or m3-11 didn't already (skip any edit already done).

## Acceptance criteria

- [ ] Every AB08 state is reachable in compose (the `-tags e2e,runner` table passes in `judge-runner-e2e`; anything vitest-only is named in the decisions log) and renders the AB08 copy (vitest on the recorded fixtures).
- [ ] The dock renders only allowlisted DTO fields; hidden inputs, expected outputs, case ids, ordinals and timings never appear.
- [ ] Problems shows course and arena done markers side by side with AB09 F2's filters (no pattern filter), and falls back to the v1 list when judge is absent for the account.
- [ ] The arena has no lock state (D17), the timer is off by default, History offers Copy to editor and Diff, the spoiler confirm records the reveal without a cap, and Mark studied sets `source=manual`.
- [ ] **Arena never alters course progress** (e2e: `arena_revealed` flips once, everything else byte-equal, no practice event, and a later course pass carries `arena_prior=revealed` uncapped).
- [ ] Badges follow `GET /api/judge/status` per AB11 (breaker/saturated → "Grader busy", `off` → "Auto-grading off", not evaluable → "Grading pending · self-report"); none render when the route 404s; an open judged attempt never switches to the uncapped picker.
- [ ] Screens match AB08–AB11 at 1440 px and 390 px; web typecheck, lint, test and build green; CI green.

## Release

**Merge only — ships dark in `v1.14.0`**, tagged by [m3-13](sprint-m3-13.md). Do **not** tag. No infra PR, no flag, no new pod, no migration. Every surface here is behind m3-09's presence-by-config gate, so `v1.14.0` changes nothing for anyone until m3-13's `JUDGE_BASE_URL` infra PR, and then only for the owner and tester cohort (T-3). Rollback is R-a (unset `JUDGE_BASE_URL`) or R-c.

## Definition of Done

CI green · PR squash-merged to `main` (no tag) · acceptance criteria met · screenshots against AB08–AB11 in the PR · statuses updated (this file + [`../status.md`](../status.md): board row, artboard rows AB08–AB11; M3 stays 🔄) · notable calls (route shape, the diff library, any DTO gap reported to m3-09, the AB09 F7 / AB10 F9 judge-off departure flagged to the owner, any AB08 state left vitest-only) in the decisions log.

## Risks / watch-outs

- **t4 §8 and §3.7 still describe arena locks and a reveal cap.** D17 overrides both (t4 §13, ADR-0029 §4, PRD R-AR1 amended): the arena is unrestricted. A reviewer quoting t4 is reading the superseded text.
- **Leaking through the UI:** the dock must never infer what the DTO withholds (no "case 38 failed", no per-case timing from polling timestamps). The types are the guard; the server denylist test (m3-09) is the backstop.
- **Settling vs done:** rendering a conclusion as soon as the evaluation is terminal races practice's lock. Keep polling until `factsThroughSeq ≥ attemptSeq`.
- **Bundle size:** `@codemirror/merge` must stay inside the lazy editor chunk; check the build report.
- **Behaviour change for the cohort:** ahead-of-schedule items now open the arena, not a `counted:false` attempt (t4 §8). Non-cohort accounts keep v1; list it in m3-13's release notes.
- **`contract_changed` in compose** needs a content-overlay variant and a matching pack variant (task 5), both regenerated from `content`/`packsrc`. A pack variant alone gives `spec_mismatch` (503), not `contract_changed`. Keep both synthetic and public (never copy anything from `../xlearn-evalpack`, m3-01's AGENT.md rule).
- **Real-runner lane:** most dock states need real execution, so the e2e is `-tags e2e,runner` and runs in the Linux `judge-runner-e2e` job. A green plain-`e2e` job proves nothing about it.
- **Judge-off frames vs presence by config.** AB09 F7 and AB10 F9 draw judge-off / non-cohort accounts with an arena "Studied" marker, Mark studied and "Code checking isn't on for your account yet." That can't be built: the arena-progress and studied routes are judge-backed and 404 without judge ([m3-09](sprint-m3-09.md)). The v1 fallback therefore wins, with no arena marker, no arena link and no badges. Record the departure in the decisions log and **flag it to the owner** in this PR's body (AB09/AB10 are already frozen: ds-m3-02's merge is an entry gate). The flag doesn't block the merge; a change to AB09/AB10 would be a follow-up design PR.
- **Breaker ≠ self-report.** The L13 breaker only throttles or sheds Runs and arena work, so a badge that switched an open attempt to a picker would mis-grade learners. The Workspace follows practice's state (task 4).
