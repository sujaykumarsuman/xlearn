# Sprint ds-m3-02 — Design M3 part 2: Problems, Arena, Week/Mistakes/Progress deltas (AB09, AB10, AB12)

> **Milestone:** M3 — judge plus code grader (the boards the M3 UI sprints build against)   ·   **Track:** design (parallel; order 35)
> **Prereqs:** [ds-m3-01](sprint-ds-m3-01.md) (AB07 ★ workspace shell, AB08 results dock and AB11 badges drafted: AB10 reuses the shell and the dock, AB09 places the badges)
> **Unblocks:** [m3-11](sprint-m3-11.md) (M3 hard checklist item "AB07–AB12 frozen", rollout §5) · consumed by [m3-12](sprint-m3-12.md) (AB09, AB10) and [m3-13](sprint-m3-13.md) (AB12)
> **Release action:** **PR, stop for owner review (design).** The agent never merges; the owner's merge (or explicit approval in chat) **is the freeze**.
> **Calendar:** late October (≈ week 5, 2026-10-24 → 10-30, right after ds-m3-01) · owner event `ev-freeze-ds-m3-02` (review + merge) before [m3-11](sprint-m3-11.md) starts (mid–late November)
> **Execute with:** [`../prompts/prompt-ds-m3-02.md`](../prompts/prompt-ds-m3-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB09 A4 Problems (dual markers, filters) | X | ⬜ |
| 3 | AB10 A5 Arena (never counts, manual timer, History + Diff, spoiler confirm, study mode) | X | ⬜ |
| 4 | AB12 Week / Mistakes / Progress deltas (provisional + touch dots, source chip + concepts, provenance + judge-checked %) | X | ⬜ |
| 5 | Self-review against the brief + screenshots (1440 px, 390 px) | X | ⬜ |
| 6 | Open the design PR and STOP | X | ⬜ |
| 7 | Freeze: owner reviews, approves and merges | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Design exception:** this sprint's PR does **not** edit [`../status.md`](../status.md)
> (the PR may stay open for days). The PR sets tasks 1–5 ✅ and task 6 🔄 ("PR #N open — awaiting owner review").
> **Close-out after the owner's merge** (four edits, each idempotent — skip any already done), made by the first build
> sprint gated on the freeze, [m3-11](sprint-m3-11.md) (its entry gate "AB07–AB12 frozen"), in its own status update:
> 1. this file: tasks 6 and 7 ✅ ("merged by the owner, PR #N, <date>") and _Overall_ ✅;
> 2. `../status.md` Sprint board: the ds-m3-02 row ✅;
> 3. `../status.md` Artboards: AB09, AB10, AB12 → "frozen (PR #N, <date>)";
> 4. `../status.md` owner events: `ev-freeze-ds-m3-02` ✅.
>
> **Fallback owner:** any later consumer that finds one of these still unset — [m3-12](sprint-m3-12.md) for AB09/AB10,
> [m3-13](sprint-m3-13.md) for AB12 — applies it the same way. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [ds-m3-01](sprint-ds-m3-01.md)'s PR is **open or merged**: `AB07-workspace-code.html`, `AB08-results-dock.html` and
      `AB11-degradation-badges.html` exist on `main` or on the `design/ds-m3-01` branch. AB10 reuses AB07's workspace shell
      and AB08's dock states; AB09 places AB11's badges. If ds-m3-01 is still open with requested changes, draft against the
      latest commit on its branch and say so in the PR.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR already adds
      `AB09-*`, `AB10-*` or `AB12-*` under `design-system/screens/v2/`.

_Informational, not a gate:_ `design-system/screens/v2/index.html` (canonical file names) and `board.css` (board chrome) are
written once by [ds-m1-01](sprint-ds-m1-01.md). If they are on `main`, use the file names the index links and link
`board.css`. If not, **do not create them**: use the names in Scope below and put the same `bd-*` chrome rules in each
board's own `<style>`, noting it in the PR (ds-m3-01's fallback). This sprint never edits either file.

_Not a gate here:_ the M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) gates the M3 UI
sprint; this sprint's freeze **supplies** its "AB07–AB12 frozen" item together with ds-m3-01's.

## Goal

Draft the remaining three M3 boards — **AB09** A4 Problems, **AB10** A5 Arena and **AB12** the Week / Mistakes / Progress
deltas — as static, preview-only HTML on [`design-system/theme.css`](../../../design-system/theme.css) under
`design-system/screens/v2/`, with every frame, every state and final copy, so [m3-12](sprint-m3-12.md) (Problems, Arena)
and [m3-13](sprint-m3-13.md) (Week, Mistakes, Progress) build against owner-approved designs. Together with
[ds-m3-01](sprint-ds-m3-01.md) this completes **AB07–AB12**, which must be frozen before the M3 UI sprint
([rollout §9 freeze rule](../rollout-plan.md#9-artboards-by-milestone)). Per **BP3** (owner decision 2026-09-24) the agent
drafts every board and the owner only reviews and approves.

Two owner decisions shape these boards more than anything else, and **both override the T4 research text**:
- **D10** (ADR-0027 §4): the arena keeps arena Submits as a history (Runs are not kept), has **its own done marker**
  separate from course done, has a **manual, client-side timer that is off by default**, and **never counts** toward course
  progress, grades or revision.
- **D17** (ADR-0029 §4): the arena is **unrestricted** — no locks during a live attempt or a due touch — and an early
  Solution reveal is **recorded but does not cap** the course grade. [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)
  still lists "Submit, Solution and History are locked ('Finish your review first')" and a spoiler confirm that "caps your
  course attempt at Assisted": **do not draw either.**

## Scope

**In**
- Three board files (names from the ds-m1-01 index): `AB09-problems.html`, `AB10-arena.html`,
  `AB12-week-mistakes-progress.html`. If the index on `main` names them differently, the index wins.
- Every frame in tasks 2–4, with final copy, a frame label citing its decision(s), and a behaviour-notes aside.
- A `.bd-narrow` (390 px) section per board, and PNG screenshots at 1440 px and 390 px under
  `design-system/screens/v2/shots/` (`AB09@1440.png`, `AB09@390.png`, …).
- The design PR, stopped for owner review.

**Out**
- Any `web/` code → [m3-12](sprint-m3-12.md) (AB09, AB10) and [m3-13](sprint-m3-13.md) (AB12).
- The course-attempt workspace (AB07 ★), the full results-dock state set (AB08) and the badge definitions (AB11) →
  [ds-m3-01](sprint-ds-m3-01.md). This sprint **reuses** them (a board-local copy of the markup) and never edits their files.
- AI provisional / dispute / claim cards (AB16 ★), pointer notes (AB17), AI allowance (AB18) →
  [ds-m4-01](sprint-ds-m4-01.md). AB12 shows only the **list-level** "provisional" chip, as a flagged "from M4" variant.
- Multi-course fidelity (a second course's Problems / Progress) → [ds-p-01](sprint-ds-p-01.md); AB10's study mode for
  quiz / canvas items is drawn at **low fidelity** here.
- The public profile (AB06, frozen in [ds-m2-01](sprint-ds-m2-01.md)); AB12's Progress frames reuse its provenance-chip vocabulary.
- `design-system/screens/v2/index.html`, `board.css` (ds-m1-01, written once) and `docs/v2/status.md` (see the Status note).
- Shipping boards (preview-only, never imported by `web/`) and owner design hours beyond review (BP3).

## Tasks

### 1 · Board scaffolding (own files only) [X]

Follow the conventions ds-m1-01 recorded in the index's **Conventions** block (and its [task 2](sprint-ds-m1-01.md#2--board-scaffolding-the-conventions-x)):
- One static HTML file per board, `<link rel="stylesheet" href="../../theme.css">` then `board.css`, plus the Google Fonts
  `<link>` the v1 boards use. No JS runtime, no `support.js`, no `.dc.html` canvas markup; the files open from disk.
- `theme.css` tokens and components **verbatim** (`ds-btn`, `ds-badge--{ok,warn,err,info,violet}`, `ds-chip`, `ds-seg`,
  `ds-modal`, `ds-meter`, `ds-tabs`, `ds-toggle`, `xl-panel`, `xl-page-h`, `xl-diff--{easy,med,hard}`, `xl-pat`, `xl-lock`,
  `xl-touch__d--{pass,fail,due,mock}`, `xl-timer`, `xl-code`, `xl-table`, `xl-kbd`, …). Difficulty: Easy=`--ds-ok`,
  Medium=`--ds-warn`, Hard=`--ds-err`. Layout the theme lacks (a right-hand drawer, a diff gutter, a stacked bar) goes in the
  board's own `<style>`, built from tokens only, and never overrides a `ds-*`/`xl-*` rule.
- Frames labelled `AB09-F3 · <state>`; full-screen frames stack vertically, panel-sized states sit side by side in a `bd-grid`.
- A **behaviour-notes aside** per frame: data source (endpoint and field), timers, gates, exact error codes and copy, the
  decision it implements, a11y (focus order, contrast ≥ 4.5:1 on text, keyboard shortcuts, `aria-live` for status changes,
  reduced motion) and the < 1024 px layout.
- **Leak rules** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise)): no pattern chip and no primary-concept chip on an item
  with an open counted attempt or a due or live touch (an `xl-lock` "Pattern hidden while live" instead); no hidden input,
  expected output, case id, per-case timing or test name anywhere; hidden results are **passed/total + one perf bit + the
  first failure class** only ([ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract)).
- Use realistic data from the 14 seeded DSA items (`curriculum/dsa/problems.json` or its m1-09 successor): ids, titles,
  difficulties, weeks. No lorem ipsum, no "TBD".

### 2 · AB09 A4 Problems [X]

`AB09-problems.html` — `/xlearn/dsa/problems`, the course's full item list with **two independent done markers**. Consumed
by [m3-12](sprint-m3-12.md). Supersedes v1 `web/src/screens/Problems.tsx` (no v1 board existed). Sources:
[t4 §8 Arena](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (dual markers; ahead-of-schedule
items go to the arena), D10 ([ADR-0027 §4](../../adr/0027-content-evalpack-and-user-data-model.md#4-problems-arena)),
D15/D18 grades ([ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer)),
PRD §5.5 R-AR1, AB11 for badges.

| Frame | State | Must show |
|---|---|---|
| F1 | Default list (owner, judge on) | Page head: eyebrow "Data Structures & Algorithms", **"Problems"**, summary "151 problems · 48 done in the course · 12 solved in the arena"; a two-line **marker legend**: "**Course** — counted attempts: graded, scheduled for revision" · "**Arena** — free practice: never counts toward your course"; by-week groups (v1 parity: "Week 2" + `Current` badge); rows `#16` · **3Sum** · `xl-diff--med` · **course marker** · **arena marker** · grading chip (`Judge` / `Self-report`) · chevron |
| F2 | Filters | `ds-seg` **All · Not started · Course done · Arena done · Review due**; difficulty chips Easy / Medium / Hard; grading chips "Judge-graded" / "Self-report"; week select; search "Search by title or #". State lives in the URL (`?status=arena&diff=med`) so Back restores it. **No pattern filter** (a pattern filter would reveal the pattern of live items — ADR-0027 §1); say so in the notes |
| F3 | Row-state gallery (panel-sized, side by side) | (a) current-week, not started → course marker "—", row opens the **AB07 cover**; (b) **ahead of schedule** → `xl-lock` "Ahead · arena only", row opens the **arena** (t4 §8 behaviour change: no more v1 `counted:false` attempts); (c) open attempt → "In progress · 31:12 left" (server timer, D15), pattern hidden; (d) `awaiting_evaluation` → "Grading…" (`ds-spin`); (e) `self_grade_pending` → `ds-badge--warn` "Needs your grade"; (f) concluded **Clean** + provenance "judge ✓"; (g) concluded **Miss** + "judge" (timeout or give-up); (h) **Rough** + "self · honour" (a self-report item); (i) due touch → `xl-touch__d--due` "Review due · Day 7", pattern hidden; (j) arena only → course "—", arena "Solved in arena · Oct 12"; (k) both markers; (l) arena "Studied" (manual marker) |
| F4 | Degradation placement (AB11 reference) | Page-level banners exactly as AB11 defines them (runner busy; judge off → "every problem uses self-report right now"); the per-row "grading pending: self-report" badge on a `spec_mismatch` item. Link AB11 in the notes; do **not** redefine the badges |
| F5 | Empty and edge states | No problems yet (v1 `EmptyState` parity); filters match nothing → "No problems match these filters." **[Clear filters]**; not enrolled → the AB02 F5 start-path call to action |
| F6 | Loading / error | Skeleton rows; "Couldn't load the problem list." **[Retry]** (`States.tsx` parity) |
| F7 | Judge off for this account (`JUDGE_BASE_URL` unset, or non-cohort) | No grading chips; course markers carry "self"; **no arena marker at all** — `arena_progress` is judge's and every judge route 404s while judge is off (m3-09 presence by config), so the arena is v1's; the legend's Arena line, the "Arena done" filter and the summary's "solved in the arena" count are hidden; a one-line note "Code checking isn't on for your account yet." |
| F8 | Narrow (390 px) | Row = title line + a second line with both markers; filters collapse to a **"Filters (2)"** sheet; the legend becomes an ⓘ popover |

Behaviour notes must state: markers come from practice's per-item state (course) and judge's `arena_progress`
(`first_passed_at` → "Solved in arena", `source=manual` → "Studied"), both via the BFF ([m3-09](sprint-m3-09.md)); with
the judge off there is no arena marker (F7); **arena activity never moves the course marker**, the week rollup or the
summary's "done in the course" count (D10).

### 3 · AB10 A5 Arena [X]

`AB10-arena.html` — the arena workspace for one item, `/xlearn/dsa/problem/:id?practice=1` (v1 arena URL; [m3-12](sprint-m3-12.md)
may rename it — the board does not fix the route). **Reuse AB07's workspace shell** (statement left, editor centre with the
AB08 dock, right rail) with an arena HUD: **no server timer ring, no grade outlook, no hint gate, no give-up**. Sources: D10,
D17, [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena) (as overridden by
[t4 §13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict)),
[t4 §4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context) (arena runs `code` steps only), PRD §5.5 R-AR1 and §5.6 R-AR1 (amended), L9 / L6 ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)).

| Frame | State | Must show |
|---|---|---|
| F1 | Arena default (item not live) | A persistent strip **"Arena · Nothing here counts toward your course."** (`ds-badge--info`); crumbs `xlearn / dsa / problems / #16 · arena`; difficulty chip; arena marker in the HUD ("Not solved in the arena yet"); **[Timer off]** button; language picker Go / C++ / Python; editor with the generated starter; **Run** (⌘↵) and **Submit** (⌘⇧↵) with the helper "Arena submits run the hidden tests but never count"; a **Solution** tab behind the spoiler confirm; **History (3)** |
| F2 | Manual timer popover (D10: manual, client-side, off by default) | Presets 15 / 25 / 45 min + custom; [Start] [Reset]; the note "Only you see this timer. It isn't recorded." Running: a small `xl-timer` in the HUD. At zero: "Time's up on your timer. Keep going — nothing is recorded." (no modal, no lock) |
| F3 | Arena Submit results (AB08 dock reuse) | "Arena · **Wrong answer** · 37/40 hidden passed" with a `not counted` chip; perf TLE ("correctness 40/40, performance failed"); CE ("Hidden tests don't compile against your code. Check the required signature. (Not counted.)"); **Accepted · 40/40 · performance passed** → the HUD marker flips to **"Solved in the arena · just now"** (`arena_progress` upsert, `source=auto`) |
| F4 | History drawer (read-only, R-AR1) | Right drawer "Arena history · 3 submits (Runs aren't kept)"; rows `Oct 12 · 14:02 · Go · Accepted · 40/40`, `Oct 12 · 13:55 · Go · Wrong answer · 37/40`, `Oct 11 · 21:40 · Python · Compile error · not counted`; the selected row in a read-only `xl-code` viewer; actions **[Copy to editor]** (dirty-editor confirm: "Replace the code in your editor with this submission?" [Replace] [Cancel]) and **[Diff with editor]** |
| F5 | Diff view | Side-by-side at ≥ 1024 px, unified at < 1024 px; added / removed lines tinted from `--ds-ok` / `--ds-err` tokens with `+`/`−` gutters (never colour alone); header "Submission Oct 12 · 13:55 ↔ your editor"; read-only; [Close] |
| F6 | **Spoiler confirm** (Solution before the course attempt has concluded) | `ds-modal--sm` "Reveal the solution?" — "You haven't finished this problem in the course yet. Revealing it here is noted on your course attempt as *seen in the arena*. It **doesn't cap your grade** — the arena works on the honour system." **[Reveal solution]** **[Cancel]**. Notes: `POST /api/problems/{id}/arena/reveal` ([m3-09](sprint-m3-09.md)) → practice `POST /problems/{id}/arena-reveal` records `arena_revealed_at` (`arena_prior=revealed`), **no cap** (D17 — replaces t4's "caps your course attempt at Assisted" and its pre-D17 `POST /api/arena/{item}/reveal`). Variant: course attempt already concluded → the tab opens with no confirm |
| F7 | **Item is live** (open course attempt, or a due or live touch) | **No lock** (D17). An honour-system notice: "You have a review due on this problem (Day 7). The arena stays open — practising here first makes that review less useful." Run, Submit, History and Solution (with F6's confirm) all enabled; the pattern chip stays withheld (withhold() nudge, ADR-0027 §1) until the Solution is revealed |
| F8 | Study mode (quiz / canvas items; low fidelity, "from P") | Editable parts, "Answers aren't checked in the arena" (INV-4: `key` and `ai_rubric` never run in arena), **[Reveal reference]** (F6 confirm variant), **[Mark studied]** → marker "Studied · Oct 12" (`source=manual`) |
| F9 | Judge off for this account (AB09 F7) | **The v1 arena, unchanged**: no Run, no Submit, no History, **no arena marker and no [Mark studied]** — every arena route is judge's and 404s while judge is off (m3-09 presence by config); the strip "Arena · Nothing here counts toward your course." plus the note "Code checking isn't on for your account yet." Variant (b), judge on but the runner lane off: Run and Submit hidden with AB11's badge; History, the arena marker and Solution stay |
| F10 | Arena limits and errors | L9 quota "You've used today's 100 arena submits. More at 00:00 your time."; `too_fast` "Wait a moment between submits."; `queue_full` "The grader is busy. Try again in 30 s." (the seconds come from `Retry-After`); L6 413 "Too large to submit (code is capped at 64 KiB)"; `inconclusive` "We couldn't get a reliable result. Not counted. [Submit again]" — every string is **AB08's F11 / F14 / F15 copy verbatim** (ds-m3-01), with "arena" added only to the quota line |
| F11 | Narrow | < 1024 px: Statement / Work / Results tabs (AB07 parity); History and Diff as full-screen sheets; 390 px frames of F1, F4, F6 |

Behaviour notes must state: arena context id = `uuidv5(account, item)` (server-side; the client never sends it); Runs are
not persisted (samples + custom input only, no reference oracle); arena events are never emitted; arena submits are
learner-private in v2.0 (D10) and never reach `/public/stats` (P10).

### 4 · AB12 Week / Mistakes / Progress deltas [X]

`AB12-week-mistakes-progress.html` — three sections, one board. Consumed by [m3-13](sprint-m3-13.md). Supersedes the v1
`Week`, `Mistakes` and `Progress` boards for the deltas only (layout otherwise v1 parity: `web/src/screens/{Week,Mistakes,Progress}.tsx`,
`web/src/components/ProgressViews.tsx`). Provenance chips reuse **AB06's vocabulary** (frozen in ds-m2-01): "self · honour",
"judge", "AI" (M4), "override · honour" (M4). Sources: [t4 §8 changed v1 artboards](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed),
[t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (sources, precedence),
[t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise) (concepts, withheld), [ADR-0029 §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop),
[rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P7 / P8, D2 (touch dots from `anchor_at`).

**Week** (`/xlearn/dsa/week/:n`)

| Frame | State | Must show |
|---|---|---|
| W1 | Wired touch dots | The five dots per row fed by review (today `aggregate.go:120` fills a placeholder): `xl-touch__d--{pass,fail,due,mock}` + neutral pending; tooltip "Day 7 · passed Oct 9" / "Day 21 · due Thu"; legend under the list |
| W2 | Row status chips | Not started · "In progress · 31:12 left" · "Grading…" · "Needs your grade" (`self_grade_pending`) · Clean / Rough / Assisted / Miss with a provenance mini-chip; pattern withheld on live rows |
| W3 | Provisional (flagged **"from M4"**) | "Rough · AI · provisional · auto-accepts in 23:41" as a list chip only; the card itself is AB16 |
| W4 | Week rollup | "Core 5 · concluded 4 · passed 3 · judge-checked 3 of 4" + the difficulty split; *passed* means grade ≠ Miss (T4 §11.4 #4) |

**Mistakes** (`/xlearn/dsa/mistakes`)

| Frame | State | Must show |
|---|---|---|
| M1 | Category **source chip** | On each entry's category: **You** (learner — locks the field), **Judge** (strong rule — e.g. "Judge · Complexity misjudged" from `tle_perf_only`, which pre-fills the category and counts toward the weak area), **Suggested** (weak rule; a ghost / dashed chip), **xLearn AI** (analyzer, flagged "from M4"). Tooltip: "Your choice always wins. Then judge rules, then xLearn AI; *suggested* categories don't count toward your weak area." |
| M2 | **Concepts** (≤ 3) | Chips linking to the Concept page, primary first (e.g. "Two pointers · Sorting · Dedup"), sourced from the item and the failed checks; on a live item the primary concept is withheld → `xl-lock` "Concepts hidden while this problem's review is due" |
| M3 | Entry edit | Category picker with a **weak-rule** suggestion — "Suggested by the judge: **Misread** (a sample test failed)" (`sample_failed` → misread, weak, [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)) — and **[Use suggestion]** (strong rules such as `tle_perf_only` pre-fill as a Judge chip instead, M1); editing flips the chip to **You**; concepts editable (R-JG4); the 2 KiB-per-field limit hint |
| M4 | Weak-area banner | Counts learner, judge-rule and xLearn AI categories only; footnote "2 suggested categories aren't counted — confirm them to include them." |
| M5 | Before any judge signal | v1 parity (all **You**); no empty source chips |

**Progress** (`/xlearn/dsa/progress`)

| Frame | State | Must show |
|---|---|---|
| P1 | Grade mix with provenance (P7) | Clean / Rough / Assisted / Miss bars, each segmented by provenance (judge · self · honour · AI · override) with a legend; **"Judge-checked 62%"** with ⓘ "Share of concluded attempts graded by the judge on checked tests. Self-reported, honour-probe, override and flagged grades don't count." |
| P2 | Judge stats (P8) | "Counted submits 214 · First-submit acceptance 38% · Languages Go 71% · Python 21% · C++ 8%"; note "Arena submits and Runs aren't counted here." |
| P3 | Before judge-graded results | Judge-checked "—" with the AB06 F3 tooltip wording; P2 hidden |
| P4 | Honour labels | Touch / probe results show "honour" where `trust=honor`; the authed Progress keeps mock best and average (D31 is public-only) |

Each section ends with its 390 px frames.

### 5 · Self-review against the brief + screenshots [X]

- Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) list
  (AB09 "dual markers"; AB10 "manual timer, history + diff"; AB12 "provenance, judge-checked %").
- **D17 check:** no frame locks the arena, and no copy says the reveal caps the grade. **D10 check:** nothing in the arena
  moves a course marker, week rollup or Progress number; no arena or Run count appears on Progress or the public profile.
- **Leak check:** no pattern or primary-concept chip on a live item; no hidden inputs, expected outputs, case ids, per-case
  timings or test names; hidden results are count + perf bit + class only.
- **Consistency:** dock copy identical to AB08; badges identical to AB11; provenance chips identical to AB06.
- Screenshot every board full-page at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB{09,10,12}@{1440,390}.png`
  (each ≲ 500 KB).

### 6 · Open the design PR and STOP [X]

Branch `design/ds-m3-02`; conventional commit `docs(design): AB09 AB10 AB12 — M3 boards part 2` ending with the
attribution lines. PR titled `docs(design): AB09 AB10 AB12 — M3 boards (part 2)` with each board's frame list, the
screenshots embedded (`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m3-02/design-system/screens/v2/shots/<file>?raw=true`)
and a **"Decisions to confirm"** list. Seed it with:
1. AB10 F7 keeps the **pattern chip withheld** in the arena while the item is live (ADR-0027 §1 nudge) even though D17 removes
   every lock — or show it?
2. [m3-09](sprint-m3-09.md) applies `withhold()` only to the pattern and concept chips (and solution stages) and never
   locks arena Submit / History / Copy-to-editor (D17); the boards follow the same rule. Confirm.
3. Whether code items (not only quiz / canvas) get **[Mark studied]** when the judge is on (the boards show it only in
   study mode; with the judge off there is no arena marker at all, AB09 F7 / AB10 F9).

**Do not merge. Do not enable auto-merge.** Report the PR link and stop.

### 7 · Freeze [O]

The owner reviews and approves; the owner's merge (or an agent's merge on the owner's explicit approval in chat) is the
freeze (`ev-freeze-ds-m3-02`). Requested changes are made on the same branch.

## Acceptance criteria

- [ ] Every frame listed for AB09 (F1–F8), AB10 (F1–F11) and AB12 (W1–W4, M1–M5, P1–P4) is present with final copy, its state, and a behaviour-notes aside citing its decision.
- [ ] No frame locks the arena or caps a grade on reveal (D17); no arena activity moves a course number (D10).
- [ ] No board leaks withheld data (pattern / primary concept on live items, hidden inputs, case ids, per-case timings).
- [ ] Boards open from disk with no JS runtime; `theme.css` and `board.css` are linked, not copied or overridden; only this sprint's three board files (and their screenshots) are added.
- [ ] PR open with 1440 px and 390 px screenshots of each board; **not merged by the agent**.

## Release

**PR, stop for owner review (design).** Nothing deploys: boards are preview-only files. The owner's merge is the freeze
(`ev-freeze-ds-m3-02`); with ds-m3-01's freeze it satisfies the M3 checklist item "AB07–AB12 frozen", which gates
[m3-11](sprint-m3-11.md).

## Definition of Done

PR open with AB09, AB10, AB12 and their screenshots · every acceptance box ticked · this sprint's Status table updated in
the PR (tasks 1–5 ✅, task 6 🔄 awaiting review) · no `web/`, `index.html`, `board.css` or `docs/v2/status.md` change · the
agent stops at the open PR. (Done-done — ✅ _Overall_ — is the Status note's close-out, made by [m3-11](sprint-m3-11.md)
once the owner has merged, or by the fallback owner.)

## Risks / watch-outs

- **Drawing T4's superseded arena locks.** t4 §3.7 / §8 / §10 still describe locked Submit / History / Solution and a
  reveal cap; D17 overrides all of it. Cite D17 on every AB10 frame that touches reveal or live items.
- **A board contradicting an Accepted decision** (D10, D15, D16, D17, D18, D27, D31): cite the decision per frame; when a
  decision is ambiguous, draw the conservative reading and list it under "Decisions to confirm".
- **Drift from ds-m3-01.** AB10 copies AB07's shell and AB08's dock; if the owner changes those in ds-m3-01's review, carry
  the change into AB10 on this branch before the freeze.
- **Promising M4 early.** The provisional chip (W3) and the xLearn AI source chip (M1) are flagged "from M4"; nothing else
  may show AI.
- **Filter leaks.** A pattern or concept filter on Problems would reveal withheld data for live items — keep it out.
- **Screenshots in git:** keep PNGs small; they become the frozen reference m3-12 / m3-13 compare against.
