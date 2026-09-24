# Prompt — Sprint ds-m3-02 · Design M3 part 2: Problems, Arena, Week/Mistakes/Progress deltas (AB09, AB10, AB12)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m3-02.md`](../sprints/sprint-ds-m3-02.md)   ·   **Milestone:** M3 (design track)   ·   **Prereqs:** [ds-m3-01](../sprints/sprint-ds-m3-01.md) (AB07/AB08/AB11 drafted)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions. **This sprint overrides the
  land-and-sync directive:** it ends at an open PR and never merges (BP3, design sprints).
- The plan: [`../sprints/sprint-ds-m3-02.md`](../sprints/sprint-ds-m3-02.md) — the frame tables (AB09 F1–F8, AB10 F1–F11,
  AB12 W1–W4 / M1–M5 / P1–P4) and the "Decisions to confirm" seed list are spelled out there. Follow them exactly.
- [`../rollout-plan.md`](../rollout-plan.md) [§9 Artboards by milestone](../rollout-plan.md#9-artboards-by-milestone) (board list,
  superseded v1 boards, freeze rule), [§5](../rollout-plan.md#5-m3-hard-entry-checklist) ("AB07–AB12 frozen" gates the M3 UI
  sprint), [§10](../rollout-plan.md#10-public-dashboard-tasks) (P7 provenance / judge-checked %, P8 judge stats).
- **The owner decisions that override T4** — [t4 §13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict):
  **D17** (arena unrestricted; reveal recorded, **no cap**), D15/D18 (45:00 hard limit, hint at 15, grades), D16 (no
  re-implement), D14 (AI suggestions — M4, flagged only). Then D10 in the
  [feasibility decisions log](../feasibility.md#decisions-log-newest-first).
- [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (arena flow, results-dock copy, changed v1
  boards — **ignore its arena locks and reveal cap**), [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)
  (arena mechanics, same caveat), [t4 §4.4](../research/t4-judge-contract.md#44-which-steps-run-in-which-context),
  [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (mistake sources + precedence),
  [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise) (concepts, withheld while live).
- ADRs: [0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule) (withhold while live),
  [0027 §4](../../adr/0027-content-evalpack-and-user-data-model.md#4-problems-arena) (arena, D10),
  [0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) (learner DTO allowlist) and
  [§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) (arena unrestricted, pre-fill precedence),
  [0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6 413, L9 arena quota).
- [PRD](../../prd/xlearn-v2-prd.md) §5.5 (R-AR1 arena, R-PP1 provenance) and §5.6 (R-JG1–R-JG4; R-AR1 amended).
- Boards to reuse (read-only; never edit): `design-system/screens/v2/AB07-workspace-code.html`, `AB08-results-dock.html`,
  `AB11-degradation-badges.html` (on `main`, or on the `design/ds-m3-01` branch if not merged), `AB06-*` (public profile,
  provenance-chip vocabulary), `AB04-*` (touch dots), `index.html` (canonical names + Conventions block) and `board.css`.
- [`../../../design-system/README.md`](../../../design-system/README.md) and [`../../../design-system/theme.css`](../../../design-system/theme.css) —
  reuse verbatim. v1 references (not runnable): `design-system/screens/{Problem,Week,Mistakes,Progress}.dc.html`.
- Live-app structure for parity (read-only): `web/src/screens/{Problems,Week,Mistakes,Progress}.tsx`,
  `web/src/components/{ProgressViews,States}.tsx`, `internal/gateway/aggregate.go` (the touch-dot placeholder at `:120`),
  `curriculum/dsa/problems.json` (or its m1-09 successor under `curriculum/courses/dsa/`) for realistic item data.

## Context

M3 puts the judge behind Run/Submit. ds-m3-01 drafted the course-attempt hero (AB07 ★), the results dock (AB08) and the
degradation badges (AB11). This sprint drafts the other three M3 boards: **AB09** Problems with its two independent done
markers (course vs arena), **AB10** the arena workspace, and **AB12** the Week / Mistakes / Progress deltas (wired touch
dots, grading states, mistake source chips and concepts, provenance and judge-checked %). The rollout requires AB07–AB12
**frozen before the M3 UI sprint** (m3-11). The owner decided (BP3, 2026-09-24) that agents draft every board and the
owner only reviews. The arena is where T4's research text is most out of date: **D17 removed every arena lock and the
reveal cap**, and D10 fixed the arena as never-counting practice with its own done marker and a manual timer.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] ds-m3-01's PR is open or merged: AB07, AB08 and AB11 exist on `main` or on `design/ds-m3-01`
      (`gh pr list --state all --search "AB07"`). If still open with requested changes, draft against its latest commit and say so in the PR.
- _Informational, not a gate:_ ds-m1-01's `design-system/screens/v2/index.html` and `board.css`. If they are on `main`, use
  the index's file names and link `board.css`; if not, **do not create them** — use the names below and put the `bd-*`
  chrome rules in each board's own `<style>`, noting it in the PR (ds-m3-01's fallback).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR already adds `AB09-*`,
      `AB10-*` or `AB12-*`.

## Do this (in order)

1. **[X] Branch** `design/ds-m3-02` off an up-to-date `origin/main`.
2. **[X] Read the conventions** in `index.html`'s Conventions block and the canonical file names (if the index is on `main`). Use exactly
   `AB09-problems.html`, `AB10-arena.html`, `AB12-week-mistakes-progress.html` unless the index says otherwise.
3. **[X] AB09** — `AB09-problems.html`, frames F1–F8: the marker legend ("Course — counted…" / "Arena — free practice:
   never counts…"), by-week rows with **course marker + arena marker + grading chip**, the filter bar (URL state; **no
   pattern filter**), the row-state gallery (a–l, including "Ahead · arena only" routing to the arena and "Needs your
   grade"), AB11 badge placement (copied, not redefined), empty / filtered-empty / not-enrolled, loading / error, the
   judge-off variant (**no arena marker at all** — `arena_progress` and every arena route are judge's and 404 while judge is
   off; the arena legend, filter and count hidden), and the narrow layout.
4. **[X] AB10** — `AB10-arena.html`, frames F1–F11, on a board-local copy of AB07's shell with **no server timer, no grade
   outlook, no hint gate, no give-up**: the "Nothing here counts toward your course." strip, the manual timer popover
   (off by default, "isn't recorded"), arena Submit results via AB08's dock copy with a `not counted` chip and the marker
   flip on Accepted, the read-only History drawer (Runs aren't kept; Copy to editor with a dirty-editor confirm; Diff), the
   diff view (side-by-side / unified, `+`/`−` gutters), the **spoiler confirm that records but does not cap** (D17), the
   live-item honour notice with **nothing locked**, low-fidelity study mode with Mark studied (judge on), the judge-off
   frame (**the v1 arena unchanged**: no Run/Submit/History, no arena marker, no Mark studied, plus "Code checking isn't on
   for your account yet."; variant: judge on, runner lane off), the limit and error copy (L9, `too_fast`, `queue_full`, L6
   413, `inconclusive` — **AB08's F11/F14/F15 strings verbatim**, e.g. "Wait a moment between submits." and "The grader is
   busy. Try again in 30 s." with the seconds from `Retry-After`), and the narrow tabs. The spoiler-confirm note names
   m3-09's `POST /api/problems/{id}/arena/reveal` → practice `arena-reveal`, never t4's pre-D17 `/api/arena/{item}/reveal`.
5. **[X] AB12** — `AB12-week-mistakes-progress.html`: Week W1–W4 (wired dots, row status chips, the M4-flagged
   provisional chip, the rollup with "judge-checked 3 of 4"), Mistakes M1–M5 (source chips You / Judge / Suggested /
   xLearn AI-from-M4 with the precedence tooltip — a **Judge** chip from a strong rule such as `tle_perf_only` →
   Complexity misjudged, ≤ 3 concept chips withheld while live, the edit flow with [Use suggestion] on a **weak** rule
   (`sample_failed` → Misread, t4 §6.3), the weak-area banner excluding suggested categories, the pre-judge state), Progress P1–P4 (grade mix
   segmented by provenance, Judge-checked % with its definition, judge stats P8, the pre-judge "—" state, honour labels).
   Provenance chips use AB06's wording.
6. **[X] Narrow sections** — every board ends with `.bd-narrow` 390 px frames of its key frames.
7. **[X] Self-review** — the plan's task 5: D17 check (no lock, no cap), D10 check (arena never moves a course number),
   leak check, consistency with AB06 / AB08 / AB11, contrast and focus-order notes on every frame. Fix, then re-check.
8. **[X] Screenshots** — serve statically (`python3 -m http.server 5198 --directory design-system`, or open from disk) and
   capture each board full-page at **1440 px** and **390 px** (Browser pane `resize_window` + screenshot, or headless
   Chrome `--headless=new --screenshot=… --window-size=1440,900`) into `design-system/screens/v2/shots/AB{09,10,12}@{1440,390}.png`, each ≲ 500 KB.
9. **[X] Sprint file** — in `docs/v2/sprints/sprint-ds-m3-02.md` set tasks 1–5 ✅, task 6 🔄 "PR #N open — awaiting owner
   review", _Overall_ 🔄. **Do not edit `docs/v2/status.md`.**
10. **[X] Commit + PR, then STOP** — conventional commit `docs(design): AB09 AB10 AB12 — M3 boards part 2` ending with the
    attribution lines; push; open the PR titled `docs(design): AB09 AB10 AB12 — M3 boards (part 2)` with each board's frame
    list, the screenshots embedded via
    `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m3-02/design-system/screens/v2/shots/<file>?raw=true`, and the
    plan's three seeded **"Decisions to confirm"** plus any ambiguity you resolved. **Do not merge. Do not enable
    auto-merge.** Report the PR link and stop. Requested changes later go on the same branch.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`,
  `internal/`, `curriculum/`, `deploy/` or `../infra` change.
- **Own files only:** add the three board files and their six screenshots; never edit `index.html`, `board.css`, AB07,
  AB08, AB11, AB06 or AB04 (copy markup into your board instead).
- **`theme.css` verbatim:** link it; never copy, fork or override `ds-*`/`xl-*` rules. Dark theme only. Easy=`--ds-ok`,
  Medium=`--ds-warn`, Hard=`--ds-err`. Extra layout in the board's own `<style>`, tokens only.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup; popovers, drawers and modals are drawn open
  in their own frames.
- **Owner decisions win over T4 text:** D17 (no arena locks, reveal recorded without a cap), D10 (never counts; own
  marker; manual timer off by default), D15/D16/D18 (grades, no re-implement).
- **No leaks:** no pattern or primary-concept chip on a live item; no hidden inputs, expected outputs, case ids, per-case
  timings or test names; hidden results = count + perf bit + class.
- **Final copy:** real strings, real error codes, realistic item data; no placeholders.
- **No status.md edit** from a design PR; m3-11 records the freeze (m3-12 / m3-13 as fallback).
- **Parallel sessions:** check peers' open PRs and worktrees before creating the board files.
- D34 (no alerting) and the memory-sum rule are not in play: nothing here runs in the cluster.

## Deliverables

- `design-system/screens/v2/AB09-problems.html`, `AB10-arena.html`, `AB12-week-mistakes-progress.html`.
- `design-system/screens/v2/shots/AB{09,10,12}@{1440,390}.png`.
- An open PR (not merged) with frame lists, screenshots and "Decisions to confirm".

## Update status

- `docs/v2/sprints/sprint-ds-m3-02.md` Status table in the PR: tasks 1–5 ✅, task 6 🔄 (PR #), _Overall_ 🔄.
- **Not** `docs/v2/status.md`. The plan's **close-out after the owner's merge** (four idempotent edits: this sprint's tasks
  6–7 and _Overall_ ✅; the status.md Sprint board row; the Artboards rows AB09, AB10, AB12 → "frozen (PR #, date)"; owner
  event `ev-freeze-ds-m3-02` ✅) is made by [m3-11](../sprints/sprint-m3-11.md), or — if it is still unset — by the fallback
  owner, [m3-12](../sprints/sprint-m3-12.md) (AB09/AB10) or [m3-13](../sprints/sprint-m3-13.md) (AB12).
- No ADR expected; design ambiguities go under "Decisions to confirm" in the PR for the owner.

## Done when (acceptance)

- [ ] Every frame listed for AB09 (F1–F8), AB10 (F1–F11) and AB12 (W1–W4, M1–M5, P1–P4) is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] No frame locks the arena or caps a grade on reveal (D17); no arena activity moves a course number (D10).
- [ ] No board leaks withheld data; boards open with no JS runtime; `theme.css` and `board.css` are linked, not copied; only this sprint's files are added.
- [ ] PR open with 1440 px and 390 px screenshots of each board; **not merged by the agent**.

Shipping: **design sprint — open the PR and STOP for owner review.** This overrides AGENT.md's end-of-session
land-and-sync: do not merge, do not enable auto-merge, do not tag. The owner's merge (or explicit approval in chat) is the freeze.
