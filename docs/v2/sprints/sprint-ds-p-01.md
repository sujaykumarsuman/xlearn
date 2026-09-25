# Sprint ds-p-01 — Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15)

> **Milestone:** P — pilot course (go-concurrency)   ·   **Track:** design (parallel; order 45)
> **Prereqs:** [ds-m1-01](sprint-ds-m1-01.md) (board index, `board.css`, AB02) · [ds-m2-01](sprint-ds-m2-01.md) (AB05) · [ds-m3-01](sprint-ds-m3-01.md) (AB07★ shell, AB08 dock) · [spk-02](sprint-spk-02.md) (the P3 TSAN verdict task 1 checks Q5 against)
> **Unblocks:** [p-01](sprint-p-01.md) (freeze gate: AB14–AB15 frozen before P) · consumed by [p-02](sprint-p-02.md) (AB02/AB05 full fidelity, the multi-file workspace frames of AB15) and [p-03](sprint-p-03.md) (AB14, AB15 verdicts)
> **Release action:** **land-and-sync; the merge is the design freeze** — the Q5 record docs PR (task 1) and the board PR both squash-merge on CI green; the board merge is the freeze; no tag, nothing deploys (launching the prompt is the owner's approval, [D40](../feasibility.md#decisions-log-newest-first); the owner may review after the merge, and any change to a frozen board is a follow-up design PR)
> **Calendar:** November (order 45, alongside M3-1). PRD Q5 (`ev-q5`) is confirmed by the owner **before launch** (the prompt's `## Before you launch (owner)` block; launching attests it) and recorded in task 1 — nothing waits mid-session; the freeze is this session's board merge (`ev-freeze-ds-p-01` is automatic and needs no tick; the Artboards rows record the freeze), before p-01 (December).
> **Execute with:** [`../prompts/prompt-ds-p-01.md`](../prompts/prompt-ds-p-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | PRD Q5 (`ev-q5`): the owner confirms it before launch; the session records it (small docs PR, merged on CI green) | O (before launch) + X | ⬜ |
| 2 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 3 | AB14 A7 Workspace-Quiz | X | ⬜ |
| 4 | AB15 go-concurrency multi-file + race verdicts (or the SQL explorer if Q5 flips) | X | ⬜ |
| 5 | AB02 / AB05 full fidelity (two real courses) | X | ⬜ |
| 6 | Self-review checklist (run before merging) + screenshots | X | ⬜ |
| 7 | Open the design PR (screenshots, frame lists, the ticked self-review checklist, "Decisions to confirm") | X | ⬜ |
| 8 | Freeze: squash-merge on CI green (the merge is the freeze) → status → sync `main` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Two PRs, both merged on CI green ([D40](../feasibility.md#decisions-log-newest-first)).**
> Task 1's Q5 record PR merges first; it updates [`../status.md`](../status.md) (`ev-q5`, the Decisions-log line, the PRD Q5
> open-owner-item row, the p-01 flag if SQL) and sets task 1 ✅ here. The board PR records the rest: once its number is known, a
> last commit sets tasks 2–8 and _Overall_ ✅ here (task 8: "frozen: merged in PR #N, <date>") and, in status.md, this sprint's
> Sprint-board row ✅, the Artboards rows AB14 and AB15 → ✅ "frozen (merged, PR #N, <date>)" and AB02 and AB05 → ✅ "full
> fidelity frozen (merged, PR #N, <date>)", and the Snapshot's artboard count (`ev-freeze-ds-p-01` is automatic: no tick). If
> the merge slips to another day or fails after that commit, correct the rows in a follow-up docs PR merged the same way. Full
> rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **PRD Q5 confirmed before launch** (`ev-q5`: the prompt's `## Before you launch (owner)` block; launching attests it) — go-concurrency unless the launch message names the SQL fallback. Task 1 records it **before AB15 is drafted**; if spk-02's P3 verdict contradicts the confirmed course, task 1's fallback rule applies. Nothing waits.
- [ ] spk-02's **P3 result is recorded** in [t3 §16.2](../research/t3-sandbox.md) (the TSAN / go-race verdict under `mmap_rnd_bits=32`); task 1 checks the confirmed Q5 against it
- [ ] **AB02 and AB05 frozen** ([ds-m1-01](sprint-ds-m1-01.md), [ds-m2-01](sprint-ds-m2-01.md) merged): this sprint appends a full-fidelity section to those two files
- [ ] **AB07★ and AB08 frozen** ([ds-m3-01](sprint-ds-m3-01.md) merged): AB15 draws only the go-concurrency deltas on the Workspace-Code shell and results dock

_Informational, not a gate:_ `design-system/screens/v2/index.html` and `board.css` (from ds-m1-01) are the naming and
convention source of truth. Use exactly the file names the index links: `AB14-workspace-quiz.html`,
`AB15-go-concurrency-race.html`, `AB02-course-nav.html`, `AB05-catalog-agenda.html`.

## Goal

Record the pilot course first (PRD Q5: go-concurrency recommended, SQL the fallback — confirmed by the owner before launch,
[D40](../feasibility.md#decisions-log-newest-first), so nothing waits on an answer mid-session), then draft the pilot's boards as static, preview-only HTML on [`theme.css`](../../../design-system/theme.css):
the **A7 Workspace-Quiz** (AB14), the **go-concurrency multi-file workspace with race / deadlock / leak verdicts and the
goroutine-dump view** (AB15), and **AB02 / AB05 at full fidelity** with two real courses (DSA + the `preview` pilot). The board
PR's merge on CI green freezes them before [p-01](sprint-p-01.md), as the [rollout §9](../rollout-plan.md#9-artboards-by-milestone)
freeze rule requires ("AB14–AB15 before P"). Per **BP3** (owner, 2026-09-24) agents draft every board; per D40 the owner may
review after the merge, and any change to a frozen board is a follow-up design PR.

## Scope

**In**
- Task 1: record PRD Q5 as the owner confirmed it before launch (go-concurrency unless the launch message names SQL; the
  fallback rule covers a contradicting P3 verdict) in a separate docs PR (PRD §7 Q5 row, rollout §13, `docs/v2/status.md`),
  merged on CI green.
- AB14 and AB15 as new static HTML files under `design-system/screens/v2/`, every frame with final copy and decision cites.
- AB02 and AB05: a clearly separated **"P · full fidelity (ds-p-01)"** section appended to the existing files; the frozen
  M1/M2 frames above it are not edited, except that their low-fidelity second-course frames get a "superseded by P·n" label.
- 1440 px and 390 px screenshots of each board in the PR.
- Freeze = the board PR merged on CI green (D40), before [p-01](sprint-p-01.md); plus this sprint's own rows in
  `docs/v2/status.md` (Status note).

**Out**
- Any `web/` code → [p-02](sprint-p-02.md) (AB02/AB05, multi-file editor), [p-03](sprint-p-03.md) (AB14 quiz widget, AB15 verdict views).
- Runner and judge behaviour (the go-race profile, `gotest@1`) → [p-01](sprint-p-01.md); AB15 draws what p-01 will return.
- Editing `index.html`, `board.css`, `theme.css` or any v1 `.dc.html` → never (ds-m1-01 owns the index; theme is verbatim).
- The Workspace-Code shell itself (cover, timer ring, hint at 15, give-up) → AB07★ ([ds-m3-01](sprint-ds-m3-01.md)); AB15 reuses it.
- AI provisional, dispute, honor claim, pointer notes → AB16★/AB17 ([ds-m4-01](sprint-ds-m4-01.md)).
- Mock for the pilot: the pilot manifest has **no mock** (AB02 shows no "Mock interview" entry); mock-v2 is AB13 (M6a).
- Shipping boards (preview-only, never imported by `web/`) and owner design hours (BP3; any owner review happens after the merge).

## Tasks

### 1 · Record PRD Q5 (`ev-q5`) [O before launch + X]

Sources: [PRD §7 Q5](../../prd/xlearn-v2-prd.md#7-open-questions-routed-to-topics), [rollout §4 P](../rollout-plan.md#4-per-milestone-detail)
and [§13](../rollout-plan.md#13-open-owner-items), [t1 §7.5](../research/t1-content-data-model.md#75-what-comes-first),
[t0 §6](../research/t0-extensibility-frame.md#6-six-course-fit).

- **The owner confirms Q5 before launch** (the prompt's `## Before you launch (owner)` block;
  [D40](../feasibility.md#decisions-log-newest-first): launching attests it, and the session never asks and waits). The
  confirmed course is the one the launch message names, else the default **go-concurrency**. Before any AB15 work, check it
  against spk-02's P3 result in t3 §16.2:
  - **the launch message names SQL** → **SQL** (the cheapest *checked* course, but the heaviest runner profile: a fresh PG per
    job). AB15 becomes the SQL explorer, and [p-01](sprint-p-01.md) is re-planned (`sql-pg` profile) before it starts;
  - **go-concurrency, and go-race works** (TSAN runs under `mmap_rnd_bits=32`, with the default policy or a per-process
    `setarch -R` launcher) → **go-concurrency** (reuses the Go toolchain, the code widget and the quiz; honor-grade because its
    tests run in-process);
  - **go-concurrency, but go-race failed even with the per-process launcher** → the pre-decided fallback, **SQL**, as above.
    Note in the Q5 PR that the other way out — redesigning go-concurrency without go-race — was set aside, so the owner can
    revisit it after the merge;
  - **a P3 result that fits neither case** (e.g. recorded, but not conclusive for go-race): record the options and your
    recommendation in the Q5 PR, keep the recommendation (go-concurrency) for AB15, and mark p-01's Sprint-board row ⛔
    "needs owner decision (Q5: P3 inconclusive for go-race)" in `status.md`. Still land both PRs.

  Nothing before M3 changes either way.
- **Record it (X)** in a small docs PR of its own, separate from the board PR:
  branch `docs/ds-p-01-q5`, commit `docs(v2): PRD Q5 resolved — pilot course <slug> (ev-q5)`:
  - [`../../prd/xlearn-v2-prd.md`](../../prd/xlearn-v2-prd.md) §7: Q5 row struck through and "**Resolved (<date>):** <course> —
    confirmed by the owner before launch of ds-p-01 (D40)", plus "; the SQL fallback applied (P3: <verdict>)" when it did;
  - [`../rollout-plan.md`](../rollout-plan.md) §13: the PRD Q5 row marked resolved;
  - [`../status.md`](../status.md): owner event `ev-q5` ✅ with the date ("confirmed before launch; recorded by ds-p-01"), the "Open owner items"
    PRD Q5 row resolved, a Decisions-log line (course, the P3 input, the fallback status), and — only if SQL — a flag on the
    [p-01](sprint-p-01.md) board row: "re-plan to `sql-pg` before start"; this file: task 1 ✅.
  - Like every change this prompt makes, it **merges on CI green** (fix, then merge, on failure; never auto-merge), then
    `git checkout main && git pull` before the board branch is cut.

### 2 · Board scaffolding [X]

Follow the **Conventions** block in `design-system/screens/v2/index.html` ([ds-m1-01 task 2](sprint-ds-m1-01.md)):
- One file per board; `<link rel="stylesheet" href="../../theme.css">` then `board.css`, plus the Google Fonts `<link>`
  (Inter, JetBrains Mono). `ds-*` / `xl-*` components **verbatim** (`ds-btn`, `ds-card`, `ds-badge`, `ds-modal`, `ds-seg`,
  `ds-tabs`, `ds-chip`, `ds-meter`, `xl-code`, `xl-timer`, `xl-lock`, `xl-kbd`, `xl-touch`, `xl-pathsw`, `xl-nav`, `xl-crumb`…);
  board chrome from `board.css` only (`bd-canvas`, `bd-section`, `bd-frame`, `bd-grid`, `bd-notes`, `bd-narrow`).
  Difficulty: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. Board-only layout in the file's own `<style>`, tokens only.
- Frame label `AB14-F3 · <state>`; the full-fidelity additions in AB02/AB05 are labelled `AB02-P1 · …`, `AB05-P1 · …`.
  Every frame carries final copy (no lorem, no "TBD"), its decision refs, and a **behaviour-notes** aside: timers, gates,
  exact error codes, a11y (focus order, contrast ≥ 4.5:1 on text, keyboard, `aria-live` for status changes, reduced
  motion) and the < 1024 px intent. Each board ends with a `.bd-narrow` section (390 px).
- Static HTML only: no JS runtime, no `.dc.html` runtime. Preview-only: never imported by `web/`, never embedded, never shipped.
- **Touch only this sprint's files:** the two new boards, the appended sections of AB02/AB05, their screenshots, this plan file
  and this sprint's own rows in `docs/v2/status.md` (plus the Q5 record PR's files, task 1).

### 3 · AB14 A7 Workspace-Quiz [X]

`AB14-workspace-quiz.html`. Consumed by [p-03](sprint-p-03.md) (widget + result view). Sources:
[t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key) (modes, reveal, diagnosis, widgets),
[t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (A7: list, lock-in, results with rationale),
[t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course) (go-concurrency L1–3 recall quiz),
[t4 §6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds) (`weighted_gate@1`: quiz `clean_pct 1.0`, `pass_pct 0.8`),
[t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (`misconception:*` → category, strong),
[ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule), D10, D15, D17, D18.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | Cover (course attempt on a quiz item, e.g. `gc-008 Spot the race`) | "6 questions · graded when you submit your answers · 45:00" ; "Starting begins a server clock. It keeps running if you leave." **[Start]** | D15, D18 |
| F2 | Answering | Question cards in a part navigator; header counter **"4 / 6 answered"** (`aria-live="polite"`); unanswered cards marked; "Saved" draft chip; server timer ring (no pause); no correctness anywhere | t4 §4.3, §8 |
| F3 | Widget modes (one card each) | `choice.single` (radio `fieldset`/`legend`), `choice.multi` (checkboxes, "Select all that apply"), `choice.order` (move-up/down buttons + Alt+↑/↓), `blank.text`, `blank.numeric` (`inputmode=decimal` + unit `<select>`), `blank.big_o` (hint "e.g. O(n log n)"), `blank.predict_output` (read-only `xl-code` snippet + monospace answer box). Options are shown in a per-attempt shuffled order | t4 §4.3 |
| F4 | Submit confirm | `ds-modal--sm`: "Submit your answers? 2 questions are unanswered — they count as wrong. You can't change answers after submitting." **[Keep answering]** (default focus) · **[Submit answers]** | t4 §4.3 (`final` cadence) |
| F5 | Grading | "Recording your result…" (the `key` grader is inline; this is the practice settling phase) | t4 §5.5 (`settling`) |
| F6 | Concluded · pass | Per-question marks ✓/✗; for each: **the key and its rationale, shown only now**; score "5 / 6 · 83%"; grade chip **Rough** (≥ 80% but < 100%) with provenance **"Judge · checked"** (pack key); a revision line **only for a revisable item** ("Day 1 review tomorrow"); a pilot drill shows "Drill · not scheduled for revision" (`revision.drills false`) | `weighted_gate@1`, t4 §4.3 reveal rule, D2, I8 |
| F7 | Concluded · miss | Same marks and rationale; "Not passed · 3 / 6"; the chosen distractor's misconception **label** on its question card ("Wrong primitive"). The mistake line appears **only for a revisable item** ("Mistake entry opened · Wrong primitive", strong pre-fill from `misconception:wrong_primitive`). Draw the pilot drill variant (`gc-008`) **without** a mistake entry, showing "Drill · not scheduled for revision" as in F6: drills are non-revisable under I8 (`revision.drills false`), and mistakes open only on revisable items | t4 §6.3, I8 |
| F8 | Give up · Miss | Confirm: "Give up? This attempt is graded Miss and the answers and rationale unlock now." (+ "It enters revision tomorrow." only for a revisable item) | D15, D16, D2 |
| F9 | Deadline on page | "Time's up · recording your result…" → F6/F7 with unanswered = wrong | D15 |
| F10 | Honor-key variant | A question whose key is public-derived shows provenance **"honor"** and the note "This answer is in the public course text, so it counts on your honour" | t1 §3.1, t4 §4.0 |
| F11 | Arena / study mode | "Nothing here counts toward your course." **[Mark studied]**; answers behind a spoiler confirm "Show answers? It's recorded, and doesn't cap your course grade." | D10, **D17** (see note) |
| F12 | Recall touch (go-concurrency L1–3, on a revisable **code** item, e.g. `gc-002`) | Inside the AB04 Touch shell: badge **"Recall · ~5 min"**; 2–3 probe questions, each **[Lock in]** (Enter); after lock-in "Locked" — no correctness; result frame "Passed — advances to Day 7" / "Not passed — resets to Day 1", keys + rationale shown only now. In the "Not passed" variant, a chosen distractor's misconception gives **"Mistake entry opened · Wrong primitive"** (strong pre-fill `misconception:wrong_primitive` → `wrong_primitive`). This is where the quiz pre-fill is drawn, because the item is revisable | t4 §6.6, §6.3, I8, D15, AB04 |
| F13 | Couldn't grade | "We couldn't record your answers. Not counted. [Try again]" (infra error; the attempt stays open while the clock runs) | t4 §3, INV-14 |
| F14 | < 1024 px | One question per screen with Prev/Next and the counter pinned; Submit in a sticky footer | — |

Behaviour notes must state: keys and rationale appear **only after a counted conclusion** (course or touch) and never in
Run; mistake entries and revision lines appear only for **revisable** items (I8; the pilot's quiz drills get neither);
MCQ options are shuffled per attempt; recall bands prefer blanks over MCQ (guessing); quiz items are never
provisional (only honor keys could be claimed, and that claim UI is AB16's). **Decision to confirm:** t4 §4.3 says the key
is never shown in the arena before conclusion; **D17** (the arena is unrestricted, reveals recorded but uncapped) is drawn
here because owner decisions override the t4 body. List it under "Decisions to confirm": D17's reading is what the merge
freezes, and the owner may revisit it after the merge.

### 4 · AB15 go-concurrency multi-file + race verdicts [X]

`AB15-go-concurrency-race.html` (keep the index's file name even if Q5 flips to SQL). Consumed by [p-02](sprint-p-02.md)
(F1–F2 multi-file editor) and [p-03](sprint-p-03.md) (F3–F15 verdict views). Draw only the deltas on the frozen AB07★
shell and AB08 dock. Sources: [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (go-concurrency variant:
RACE / DEADLOCK / LEAK, "a race in any run fails", dump on timeout), [t3 §5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race)
(learner-package function names and file:line only; Submit drops test output),
[t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them) (Run `-count=1` ≤ 20 s; job ≤ 45 s;
verdict precedence RACE > DEADLOCK > TLE > LEAK > RE > WA), [t1 §7.1](../research/t1-content-data-model.md#71-package-format)
(starter module, editable list, `visible_test.go`; honor trust), [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
[rollout §10 P7](../rollout-plan.md#10-public-dashboard-tasks) (judge-checked % excludes honor),
[PRD §5.5 R-CT2](../../prd/xlearn-v2-prd.md#55-content-arena-profile-privacy-from-t1--adr-0027) (hidden feedback = passed count + first failure class).

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | Multi-file workspace · solving | File tabs + a small file tree: `pool.go`, `worker.go` (editable), `visible_test.go` and `go.mod` (read-only, `xl-lock` + "read-only"); the AB07 HUD (45:00 ring, stage tabs) with provenance chip **"honor · tests run in-process"**; actions **Run visible tests** (⌘↵) · **Submit** (⌘⇧↵); statement lists the exported API the hidden tests use (`NewPool`, `(*Pool).Submit`, `(*Pool).Wait`) | t1 §7.1, D18 |
| F2 | Run · visible tests | Per-test rows (`TestPoolRunsAll ✓ 0.12 s`, `TestPoolBounded ✗`) with **your own** assertion output (capped); note "Ran once with the race detector (-count=1). Submit runs every test many times." | t3 §6.2 |
| F3 | Run · race in a visible test | "Data race" card: two access sites in your code (`pool.go:42` write in `(*Pool).Submit` ↔ `pool.go:57` read in `(*Pool).Wait`) and the goroutine's creation site; runtime frames elided ("+4 frames outside your code") | t3 §5.8 |
| F4 | Submit · running | "Running hidden tests 20× under the race detector… (up to 45 s)" → "Recording your result…" | t3 §6.2, AB08 |
| F5 | Submit · RACE | Headline **"Data race · not passed"**; "Hidden tests: 6 / 9 passed · first failure: data race"; location in **your** files only (`(*Cache).Get` cache.go:31 ↔ `(*Cache).Set` cache.go:48); "A race in any run fails, even if the test passed on other runs." No hidden test names, lines or output | t4 §4.1, R-CT2 |
| F6 | Submit · DEADLOCK + goroutine dump | "Deadlock · a test timed out with goroutines stuck in your code"; dump panel grouped by state: "2 goroutines · chan send · 10 s" → `pipeline.stage` pipeline.go:23; "1 goroutine · sync.Mutex.Lock" → `(*Bank).Transfer` bank.go:18; stdlib/runtime/hidden-test frames collapsed ("+3 frames outside your code") | t3 §5.8 |
| F7 | Submit · TLE (not a deadlock) | "Too slow · a test hit its deadline, but no goroutine was stuck in your code. Look for busy loops or unbounded work." | t3 §5.8 |
| F8 | Submit · LEAK | "Goroutine leak · goroutines were still running after the test finished"; creation site in your code (`merge` merge.go:12, "created by fanIn") | t4 §4.1 (goleak) |
| F9 | Submit · WA | "Wrong result · 7 / 9 hidden tests passed · first failure: assertion." No test output in Submit | t3 §5.8, R-CT2 |
| F10 | CE | Your-file diagnostics positioned (`pool.go:14:2: undefined: sync.WaitGrop`) — **counted as not failed**; hidden-API variant: "Hidden tests don't compile against your code. Check the exported names in the statement. (Not counted.)" | t4 §5.6 #15, AB08 |
| F11 | REJECTED (lint, uncounted) | "Not allowed in this course: importing `unsafe` / defining `TestMain` / adding files. Nothing ran; not counted." | t3 §10 (A5 rules) |
| F12 | Inconclusive | "We couldn't get a reliable result (the server was busy). Not counted. [Submit again]" (`runner_throttled`) | INV-14, t3 §5.7 |
| F13 | Concluded · Clean | "Passed 20× under the race detector at 18:40"; grade chip **Clean** + provenance **"Judge · honor"** with tooltip "The tests run inside your program, so this counts on your honour. Judge-checked % leaves it out." | D18, P7 |
| F14 | Re-solve touch (L4–5) | AB04 Touch shell with the multi-file workspace; badge **"Re-solve · 25:00 · Mock conditions"**; criterion row "Tests pass in time (honor)"; result "advances to Day 45" / "resets to Day 1" | t4 §6.6, D15 |
| F15 | Mistake pre-fill | After a race-concluded Miss: "Mistake entry · **Data race** (from your result)" chip (strong rule `race → data_race`; `deadlock`, `leak` likewise) | t4 §6.3 |
| F16 | < 1024 px | Statement / Files / Results tabs; the file tabs become a `<select>`; dump groups as an accordion | — |

**If PRD Q5 = SQL**, draw instead the **SQL explorer** in the same file (title "AB15 · SQL explorer (PRD Q5 → SQL)"): schema
panel from the public `schema.sql`; query editor; Run on the sample dataset (grid ≤ 100 rows); Submit verdicts WA (result
set differs — no hidden rows shown), TLE (`statement_timeout`), RE (SQL error), **REJECTED** (a write in a query item,
uncounted), required `explain:*` checks; provenance **"Judge · checked"** ([t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)
SQL row, [t3 §6.3](../research/t3-sandbox.md#63-the-sql-sandbox-sql-pg18-pilot)). Say so in the PR title.

Behaviour notes must state: Submit never shows hidden test names, sources, lines or output; dump and race frames are
filtered to **learner-editable files** (a frame in a hidden `_test.go` is collapsed, never named); Run reads only
public files; honor results are labelled and excluded from judge-checked %; CE and REJECTED are uncounted.

### 5 · AB02 / AB05 at full fidelity [X]

Append a `bd-section` "**P · full fidelity (ds-p-01)**" to `AB02-course-nav.html` and `AB05-catalog-agenda.html`, drawn
with two real courses: DSA and **Go Concurrency Patterns** (`preview`, cohort only; pilot = 10 exercises over 2 weeks).
Label the earlier low-fidelity second-course frames (AB02-F8, AB05's go-concurrency placeholders) "superseded by P·n"; do
not otherwise edit frozen frames. Sources: [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today),
[ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
(`preview` hidden everywhere outside the cohort), [PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses) (D4).

| Frame | State | Must show |
|---|---|---|
| AB02-P1 | go-concurrency Roadmap `/xlearn/go-concurrency` (cohort) | Sidebar **from its manifest**: Today · Roadmap · Exercises · Progress / cap "Practice loop": Revision · Mistakes — **no Mock interview**; topbar switcher "Go Concurrency Patterns" + `ds-badge--violet` **Preview**; crumbs `xlearn / go-concurrency`; weeks 1–2 with exercise rows (difficulty chip, a code or quiz glyph, **no topic chip while an item is live**) |
| AB02-P2 | Switcher open (cohort, both enrolled) | DSA ✓, Go Concurrency Patterns · Preview; coming-soon courses locked; "Browse all courses →" |
| AB02-P3 | Item crumbs | `xlearn / go-concurrency / exercises / gc-003` (item noun from the manifest) |
| AB02-P4 | Non-cohort account, direct URL | `/xlearn/go-concurrency` renders the **same NotFound** as an unknown course (AB02-F6) — no hint the course exists |
| AB02-P5 | Non-cohort switcher | DSA + coming-soon only; no gap where the pilot would be |
| AB02-P6 | Cohort, not enrolled | Preview course CTA: "Preview · owner & testers. Tests run inside your program, so results count on your honour." **[Start course]** |
| AB02-P7 | < 1024 px | Icon rail; switcher as a sheet with both courses |
| AB05-P1 | Catalog (cohort) | DSA (enrolled: ring, "≈ 35 min/day"), Go Concurrency Patterns (**Preview**, "Pilot · 10 exercises · 2 weeks", enrolled or **[Enroll]**), four coming-soon cards |
| AB05-P2 | Catalog (non-cohort) | DSA + four coming-soon; **no trace** of the pilot |
| AB05-P3 | Today agenda, both courses | "Today · 60 min budget · 55 min planned"; **Reviews first**: DSA "Re-solve · 20:00", Go "Recall · ~5 min", Go "Re-solve · 25:00 · Mock conditions" (oldest first, course chips); then per-course new work in minutes |
| AB05-P4 | Doesn't fit | "Go concurrency: nothing fits in today's remaining 15 min" (an item that doesn't fit isn't shown) |
| AB05-P5 | R-SR5 per course | Go overdue → "Reviews first — new work paused in Go Concurrency Patterns"; DSA new work continues |
| AB05-P6 | 7-day agenda | Both courses per day with minute totals; toggle "All courses / This course" |
| AB05-P7 | < 1024 px | Stacked cards; agenda accordion |

Behaviour notes: the planner is the gateway's deterministic pure function; minutes come from each manifest's
`est_minutes`; `preview` is filtered server-side (catalog, course routes, item routes by id, enrollment, agenda, public
profile, `/public/stats`) — the SPA never has to hide it. The public repo holds the content anyway: "no trace" means
product surfaces, not secrecy.

### 6 · Self-review checklist (run before merging) [X]

The session's own gate before the merge (D40: nothing waits on the owner); its ticked result goes in the PR body. Fix what
fails, then re-check. `theme.css` check: linked (then `board.css`), never copied or overridden; tokens only, no new colours;
difficulty Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; `bd-*` chrome from `board.css` only; the index's file names.
Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) P row.
Leak check: no quiz key or rationale before a counted conclusion; no hidden test name, source, line or output; no
hidden-test frame in any dump or race report; no topic chip on a live item; the pilot absent from every non-cohort frame.
Check copy against D15 (no pause), D16 (no re-implement), D17 (arena), D18 (45:00, hint at 15), P7 (honor excluded from
judge-checked %) and **I8**: no mistake entry and no revision line on a non-revisable drill (every pilot quiz is a
`drill` with `revision.drills false`). Mistake pre-fill appears only on revisable items: AB14 F12's recall result, and
AB15 F15 on a core code item. Check contrast on every text colour pair. Screenshot each board at **1440 px** and **390 px** (full page)
into `design-system/screens/v2/shots/` per the index conventions: `AB14@1440.png`, `AB14@390.png`, `AB15@…`, and for the
appended sections `AB02-p@1440.png`, `AB05-p@1440.png` (+ `@390`) so the M1/M2 shots stay intact. Keep each ≲ 500 KB.

### 7 · Open the design PR [X]

Branch `design/ds-p-01`; conventional commit `docs(design): P boards AB14 AB15 + AB02/AB05 full fidelity` ending with the
attribution lines. PR titled `docs(design): AB14 AB15 (+ AB02/AB05 full fidelity) — P boards` (append "· AB15 = SQL
explorer" if Q5 flipped), with the screenshots embedded, a frame list per board with decision cites, the ticked self-review
checklist (task 6), **"Decisions to confirm"** (at least the D17-vs-t4 §4.3 arena reveal), and the merged Q5 record PR's link.
The list doesn't block the merge: each item states the default the merge freezes (the boards as drawn), and the owner may
revisit any item after the merge through a follow-up design PR.

### 8 · Freeze: merge on CI green [X]

Once the PR number is known, push the status commit (the Status note). When CI is green (fix, then merge, on failure),
squash-merge: **the merge is the freeze** (D40); it gates [p-01](sprint-p-01.md), and [p-02](sprint-p-02.md) /
[p-03](sprint-p-03.md) build against it. Never enable auto-merge. Then sync `main` (`git checkout main && git pull`). The
owner may review after the merge; any change is a follow-up design PR.

## Acceptance criteria

- [ ] PRD Q5, as confirmed before launch (or by the fallback rule when P3 contradicts it), is recorded before AB15 was drafted, in its own docs PR (PRD §7, rollout §13, `status.md` `ev-q5` ✅), **merged on CI green**.
- [ ] Every frame listed for AB14 (F1–F14), AB15 (F1–F16, or the SQL explorer set), AB02-P1–P7 and AB05-P1–P7 is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] No board leaks withheld data (keys before conclusion, hidden tests, hidden-test frames, topic chips on live items) and no non-cohort frame shows the pilot.
- [ ] Boards link `../../theme.css` + `board.css`, use tokens/components verbatim, and open from disk with no JS runtime.
- [ ] The self-review checklist passed and is ticked in the PR body, with "Decisions to confirm" (each item's frozen default stated).
- [ ] Board PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots of each board; `index.html`, `board.css` and `theme.css` untouched; the board PR changes `docs/v2/status.md` only in this sprint's rows (ds-p-01 ✅; AB14, AB15 frozen; AB02/AB05 full fidelity frozen).

## Release

**Land-and-sync; the merge is the design freeze** — the Q5 record docs PR and the board PR both squash-merge on CI green; the board merge is the freeze; no tag. Nothing
deploys: boards are preview-only. Launching the prompt is the owner's approval ([D40](../feasibility.md#decisions-log-newest-first)),
so nothing waits on the owner mid-session: Q5 is confirmed before launch, and the freeze is the merge. The owner may review after the merge: a change to a
frozen board is a follow-up design PR, and a different Q5 answer is a follow-up docs PR (plus p-01's re-plan). The board merge
is the freeze that gates [p-01](sprint-p-01.md) (rollout §9: AB14–AB15 frozen before P) and that [p-02](sprint-p-02.md) /
[p-03](sprint-p-03.md) build against. The Q5 record PR (task 1) is docs-only and ships in no tag.

## Definition of Done

Q5 recorded (docs PR merged on CI green) · the board PR merged on CI green (the freeze) with AB14, AB15 and the AB02/AB05
full-fidelity sections plus screenshots · every acceptance box ticked · this file's Status all ✅ and this sprint's
`docs/v2/status.md` rows updated · no `web/`, service or infra change · local `main` synced.

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D10/D15/D16/D17/D18): cite the decision per frame; where t4's body and an
  owner decision differ (the arena reveal), draw the decision and flag the conflict.
- **Leaking hidden tests through the dump or race report.** The runner's frames include hidden `_test.go` goroutines;
  AB15 must show only learner-editable files, or p-03 will copy the leak.
- **Honor labelling.** go-concurrency results come from in-process tests: every result frame carries the honor provenance,
  and nothing implies they count toward judge-checked %.
- **Designing the pilot as more than manifest + content.** The P exit criterion is "widget/profile code only": boards must
  not invent pilot-only service behaviour (a new grade rule, a mock) — reuse AB04/AB07/AB08 mechanics.
- **Q5 flipping late.** If the owner changes Q5 after the merge (e.g. picks SQL), a follow-up design PR
  redraws AB15 as the SQL explorer, a follow-up docs PR updates the Q5 record, and p-01 is re-planned.
- **Parallel design PRs:** touch only your files; never `index.html`, `board.css` or `theme.css`. Other sessions also edit
  `docs/v2/status.md` when they land: rebase on `origin/main` before each status commit, touch only this sprint's rows and
  keep theirs.
