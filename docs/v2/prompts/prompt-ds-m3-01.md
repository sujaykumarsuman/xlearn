# Prompt — Sprint ds-m3-01 · Design M3 part 1: Workspace-Code ★, results dock, degradation badges (AB07 ★, AB08, AB11)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m3-01.md`](../sprints/sprint-ds-m3-01.md)   ·   **Milestone:** M3 (design track)   ·   **Prereqs:** none hard; [ds-m1-01](../sprints/sprint-ds-m1-01.md)'s index and conventions if merged; the t4 §8 A1 doc-fix is checked, not gated

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions. **This sprint overrides the
  land-and-sync directive:** it ends at an open PR and never merges (BP3, design sprints).
- The plan: [`../sprints/sprint-ds-m3-01.md`](../sprints/sprint-ds-m3-01.md) — the frame tables (AB07 F1–F14, AB08 F1–F18, AB11 F1–F7)
  with their exact copy, the scaffolding rules and the "Decisions to confirm" list are spelled out there. Follow them exactly.
- [`../rollout-plan.md`](../rollout-plan.md) [§9](../rollout-plan.md#9-artboards-by-milestone) (the M3 board list, superseded v1 boards,
  the freeze rule, **"Fix before briefing"**), [§5](../rollout-plan.md#5-m3-hard-entry-checklist) (AB07–AB12 frozen gates the M3 UI sprint).
- Decisions (they override the research body): D10, D15, D16, D17, D18, D20, D27, D31, D34 in the
  [feasibility log](../feasibility.md#decisions-log-newest-first); [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md)
  (Accepted) §2–§4; [t4 §13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict).
- [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (the shell, flows, **results-dock copy table**, A1/A2
  frame lists, changed v1 boards), [§2.3](../research/t4-judge-contract.md#23-admission) (typed 413/429/503),
  [§2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers) (release, `self_grade_pending`),
  [§2.6](../research/t4-judge-contract.md#26-evaluation) (the learner DTO allowlist), [§3.3–§3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution)
  (transitions, give-up, the self path keeps v1 semantics), [§6.3–§6.4](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)
  (pre-fill, concepts to revise).
- [ADR-0027 §1, §5](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule) (withhold, feedback allowlist);
  [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (presence by
  config, cohort); [ADR-0035 §3–§4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (no alerting; L6, L9–L13).
- [`../../../design-system/README.md`](../../../design-system/README.md) and [`../../../design-system/theme.css`](../../../design-system/theme.css) —
  reuse verbatim. If on `main`: `design-system/screens/v2/index.html` (file names, Conventions block) and `board.css` (ds-m1-01).
- v1 references (read-only, **not runnable**): `design-system/screens/Problem.dc.html` (superseded by AB07; `:104` pattern-chip leak,
  `:126` pause, `:289-302` dock, `:332` early-unlock hint line, `:340-346` re-implement CTA, `:349-383` picker).
- Live-app structure (read-only): `web/src/screens/Problem.tsx` (stage timers `:19`, outcome labels `:21-26`, the hint copy near `:566`),
  `web/src/styles/app.css` (breakpoints, the `.xl-app` fill override).

## Context

M3 turns DSA practice into execution-graded attempts for the owner/tester cohort. The UI sprints — [m3-11](../sprints/sprint-m3-11.md)
(Workspace + CodeMirror) and [m3-12](../sprints/sprint-m3-12.md) (dock, Problems, Arena, badges) — may start only once AB07–AB12 are
**frozen** (owner-approved and merged). This sprint drafts the first three boards; [ds-m3-02](../sprints/sprint-ds-m3-02.md) drafts AB09, AB10, AB12.
The owner decided (BP3, 2026-09-24) that agents draft **every** v2 board, heroes included, and the owner only reviews. AB07 is the hero:
the owner's D15/D16/D18 flow — a **45:00 hard limit** that never pauses, the **hint at 15:00 caps Assisted**, **Clean** = pass ≤ 20:00 with
no hint, no coach and ≤ 3 failed submits, **give-up or timeout = Miss**, and **a pass finishes the attempt — no re-implement**. t4 §8's A1
list originally showed a 15:00 cover and a Re-implement frame — exactly what rollout §9 says to fix before briefing (the build-plan
session's doc-fix); if `main` still shows them, the decisions win. There is no alerting in v2
(D34), so AB11's in-app badges are how the owner notices grading trouble. No `web/` code changes here; boards are preview-only static HTML.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR adds `AB07-*`, `AB08-*` or `AB11-*` under
      `design-system/screens/v2/`. If one does, stop and report.
- [ ] Check (not a stop): does t4 §8's A1 frame list on `main` show a 45:00 cover and no Re-implement frame (the build-plan
      session's doc-fix)? If not, brief per D15/D16/D18 anyway (the decisions win) and say so in the PR body.
- [ ] Check (not a stop): are `design-system/screens/v2/index.html` and `board.css` on `main`? If yes, use the index's file names and link
      `board.css`; if not, use `AB07-workspace-code.html`, `AB08-results-dock.html`, `AB11-degradation-badges.html` and inline the `bd-*`
      chrome in each board's `<style>` — never create the index or `board.css`.

## Do this (in order)

1. **[X] Branch** `design/ds-m3-01` off an up-to-date `origin/main`.
2. **[X] Scaffold** the three board files per the plan's task 1 (link `../../theme.css` then `board.css`, Google Fonts `<link>`, static
   HTML, frame labels `ABnn-Fk · <state>`, behaviour-notes asides, `bd-narrow` 390 px sections). Original sample content only: the title
   "#16 3Sum" with a freshly written statement and invented numbers.
3. **[X] AB07 ★** — frames F1–F14 from the plan with the exact copy: cover `[Start attempt · 45:00]` with the rules card and no statement
   or pattern chip; Solving + Run (Run `⌘↵`, Submit `⌘⇧↵`); WA and perf-TLE submits with the grade outlook; the four over-time/hinted
   variants (past 20:00, hint confirm at ≥ 15:00, coach used (D27), last-5-minutes ring); the Give-up · Miss modal (default focus
   **Keep going**, no re-implement line); Solution (read-only editor, no re-implement editor); Concluded Clean; Concluded Miss with the
   *suggested* pre-fill chip and concepts to revise; Resume ("18:20 left", outlook "Rough at best · past 20:00"; past 45:00 → Miss);
   `self_grade_pending` with the **capped** picker ("up to Rough (you finished after 20:00)" — the ceiling comes from stage and clock,
   never a pass) and one Retry; the self-path variant (v1 semantics incl. its re-implement — the only place re-implement appears, and
   listed under "Decisions to confirm" — and the only uncapped picker); the Go · C++ · Python picker; the < 1024 px Statement / Work /
   Results tabs.
4. **[X] AB08** — frames F1–F18: t4 §8's dock copy **verbatim** where it has a row (queued, running, settling, WA, perf TLE, hidden-API CE,
   inconclusive, `contract_changed`, `not_evaluated_here`, 413, "8 submits left today"); the drafted rows (accepted, own CE, RE/MLE, the
   429/503 grid for L9–L13 — budget and breaker lines say "course submits", plus the arena variants, because arena work is
   budget-counted (t3 §7.4) — History, Feedback with the greyed M4 placeholder).
5. **[X] AB11** — frames F1–F7: the four badges with popover copy, placements on Problems, Workspace and Today, transitions, the
   cohort/kill-switch note, narrow.
6. **[X] Self-review** — walk every frame against its cited decision and rollout §9's list (no re-implement on the judged path; clock
   and outlook copy agree, e.g. "N left" vs "past 20:00"); the leak check (no pattern chip before the hint
   stage or conclusion; no hidden inputs, case ids, ordinals, stderr or per-case timing); nothing locks the arena (D17) or shows a public
   mock average (D31); contrast, focus order and `aria-live` notes on every frame. Fix, then re-check.
7. **[X] Screenshots** — serve statically (e.g. `python3 -m http.server 5198 --directory design-system`) and capture each board full-page
   at **1440 px** and **390 px** (Browser pane `resize_window` + screenshot, or headless Chrome `--headless=new --screenshot=…
   --window-size=1440,900`) into `design-system/screens/v2/shots/AB07@1440.png`, `AB07@390.png`, and the same for AB08 and AB11 (each ≲ 500 KB).
8. **[X] Sprint file** — in `docs/v2/sprints/sprint-ds-m3-01.md` set tasks 1–5 ✅, task 6 🔄 "PR #N open — awaiting owner review",
   _Overall_ 🔄. **Do not edit `docs/v2/status.md`.**
9. **[X] Commit + PR, then STOP** — conventional commit `docs(design): AB07 AB08 AB11 — M3 workspace, results dock, badges` ending with the
   attribution lines; push; open the PR titled `docs(design): AB07★ AB08 AB11 — M3 boards (part 1)` with the screenshots embedded via
   `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m3-01/design-system/screens/v2/shots/<file>?raw=true`, a frame list per board
   with decision cites, and the plan's **"Decisions to confirm"** list. **Do not merge. Do not enable auto-merge.** Report the PR link and
   stop. If the owner requests changes later, apply them on the same branch.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`, `internal/`,
  `curriculum/`, `deploy/` or `../infra` change.
- **`theme.css` verbatim:** link it; never copy, fork or override `ds-*`/`xl-*` rules. Dark theme only. Difficulty tokens Easy=`--ds-ok`,
  Medium=`--ds-warn`, Hard=`--ds-err`; status badges keep a text label and never rely on colour alone.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>` only.
- **Touch only this sprint's files:** the three boards, their screenshots and this sprint's plan file. Never `index.html`, `board.css`,
  `theme.css` or `docs/v2/status.md`.
- **Decisions over research:** D15/D16/D18/D27 and ADR-0029 win over t4's body (no 15:00 cover, no Re-implement, no pause, no arena lock).
- **No leaks** (ADR-0027 §1, §5; t4 §2.6): counts, the perf bit, the first failure class and bucketed usage only; no pattern before the
  hint stage or conclusion.
- **Final copy:** real strings with the real error codes; no placeholder text. AI content appears only as labelled M4 placeholders.
- **Parallel sessions:** check peers' open PRs and worktrees before creating the board files.
- D34 (no alerting) shapes AB11's purpose; the memory-sum rule, GitOps, migrations and ACLs are not in play: nothing here runs in the cluster.

## Deliverables

- `design-system/screens/v2/AB07-workspace-code.html`, `AB08-results-dock.html`, `AB11-degradation-badges.html` (or the index's names).
- `design-system/screens/v2/shots/AB{07,08,11}@{1440,390}.png`.
- An open PR (not merged) with the screenshots, per-board frame lists and "Decisions to confirm".

## Update status

- `docs/v2/sprints/sprint-ds-m3-01.md` in the PR: tasks 1–5 ✅, task 6 🔄 (PR #), _Overall_ 🔄.
- **Not** `docs/v2/status.md`. After the owner's approval: if you are asked to merge, first push one last commit setting this file's
  tasks 6–7 and _Overall_ ✅ ("frozen: PR #N, <date>"), then merge. If the owner merges in the UI, the next session on `main`
  (normally [m3-11](../sprints/sprint-m3-11.md)) sets them. Either way that next session flips the status.md Artboards rows for AB07,
  AB08 and AB11 to "frozen (PR #, date)", sets this sprint ✅ on the Sprint board and ticks owner event `ev-freeze-ds-m3-01`.
- No ADR expected: resolve design ambiguities as "Decisions to confirm" in the PR for the owner rather than writing an ADR.

## Done when (acceptance)

- [ ] Every frame listed for AB07 (F1–F14), AB08 (F1–F18) and AB11 (F1–F7) is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] AB07 has a 45:00 cover and no re-implement on the judged path (re-implement only inside the F12(a) self-path variant, listed
      under "Decisions to confirm"); the uncapped grade picker appears only in the self-path variant.
- [ ] No board leaks withheld data; boards open with no JS runtime; `theme.css` is linked, not copied; `index.html`, `board.css` and `theme.css` untouched.
- [ ] PR open with 1440 px and 390 px screenshots of each board and a "Decisions to confirm" list (incl. the self-path re-implement);
      **not merged by the agent**.

Shipping: **design sprint — open the PR and STOP for owner review.** This overrides AGENT.md's end-of-session land-and-sync: do not merge,
do not enable auto-merge, do not tag. The owner's merge (or explicit approval in chat) is the freeze.
