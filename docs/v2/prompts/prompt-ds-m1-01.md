# Prompt — Sprint ds-m1-01 · Design M1: coach states, course nav, revision v2 (AB01–AB03)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m1-01.md`](../sprints/sprint-ds-m1-01.md)   ·   **Milestone:** M1 (design track)   ·   **Prereqs:** none

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions. **This sprint overrides the
  land-and-sync directive:** it ends at an open PR and never merges (BP3, design sprints).
- The plan: [`../sprints/sprint-ds-m1-01.md`](../sprints/sprint-ds-m1-01.md) — the frame tables (AB01 F1–F15, AB02 F1–F9,
  AB03 F1–F9), the canonical board file names and the index layout are spelled out there. Follow them exactly.
- [`../rollout-plan.md`](../rollout-plan.md) [§9 Artboards by milestone](../rollout-plan.md#9-artboards-by-milestone) (board list,
  superseded v1 boards, freeze rule), §4 M1 (scope), §6 (owner calendar).
- [`../../../design-system/README.md`](../../../design-system/README.md) (tokens, component classes, screens) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) — reuse verbatim.
- v1 references (read-only, **not runnable**): `design-system/screens/{Roadmap,Dashboard,Revision,Settings,Auth}.dc.html`.
- AB01: [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (D27, limits, UI names),
  [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (mode gate, D18 capture, catalog, usage, onboarding copy),
  [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D18 Assisted),
  [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D18, D27).
- AB02: [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today),
  [t0 §7 Screens & routing](../research/t0-extensibility-frame.md#7-screens--routing),
  [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (`privacy` joins the slug guard).
- AB03: [PRD §5.1–5.2](../../prd/xlearn-v2-prd.md#51-method-universal-vs-per-course) (R-SR2),
  [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule) (withhold while live),
  [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise), [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course).
- Live-app structure for parity (read-only): `web/src/nav.ts`, `web/src/components/{Sidebar,PathSwitcher,Topbar,Coach,CoachModelSwitcher}.tsx`,
  `web/src/screens/{Revision,NotFound,Catalog,Settings,Auth}.tsx`, `web/src/styles/app.css` (the `.xl-app` fill override, 900/720 px breakpoints).

## Context

v2 turns DSA into "a course like any other" (M1) before the judge and platform AI land. M1b's UI sprints — m1-03
(course nav), m1-06 (revision), m1-10 and m1-07 (coach) — must build against **owner-approved** boards, and the rollout
freeze rule says AB01–AB03 are frozen before M1b. The owner decided (BP3, 2026-09-24) that agents draft **every** v2
board and the owner only reviews. This is the first v2 design sprint, so it also writes the board index for all
AB01–AB30 and the shared `board.css`, once, so the other design PRs (ds-m2-01 and ds-l-01 run in the same weeks) never
conflict. No `web/` code changes here; boards are preview-only static HTML.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] None beyond `depends_on` (empty).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR already adds
      `design-system/screens/v2/index.html` or `board.css`. If one does, stop and report (the index is written once).

## Do this (in order)

1. **[X] Branch** `design/ds-m1-01` off an up-to-date `origin/main`.
2. **[X] Shared chrome** — write `design-system/screens/v2/board.css` (`bd-canvas`, `bd-section`, `bd-frame`,
   `bd-frame__label`, `bd-grid`, `bd-notes`, `bd-narrow`, and `.bd-frame .xl-app { width: 100%; height: auto; min-height: 720px; }`
   — analogous to, not the same as, `web/src/styles/app.css`'s viewport fill `height: 100vh; min-height: 0`).
   Tokens only; no `ds-*`/`xl-*` redefinitions.
3. **[X] Board index** — write `design-system/screens/v2/index.html`: one table per milestone with the plan's task 1 columns
   and **exactly** its file names for AB01–AB30 (AB23 row "dropped — admin is CLI-only (D33)", AB31 row "outline only,
   v2.2+"), sprint links as `../../../docs/v2/sprints/sprint-<id>.md`, the superseded-v1 table linking `../<Name>.dc.html`,
   and a **Conventions** block restating the plan's task 2 rules. No freeze state in the index.
4. **[X] AB01** — `AB01-coach-states.html`, frames F1–F15 from the plan with the exact copy and error codes
   (`409 assist_confirm_required`, `409 coach_paused`, `503 assist_unavailable` / `503 coach_state_unavailable` fail-closed,
   typed 429s `coach_rate_limited` (20/min) · `coach_busy` (2 streams) · `coach_daily_cap` (300 per **UTC** day — copy
   "It resets at 00:00 UTC (<local time> your time)", never "in your timezone"),
   cut-short marker, provider-limited, `ErrModelAccess`, server catalog with "cost unknown" custom ids and no Google /
   no base URL, Settings per-feature defaults + "This month on your keys" estimate, onboarding step 3/4 with a flagged
   from-M4 copy variant, no-key parity). Panel-sized states side by side in a `bd-grid`.
5. **[X] AB02** — `AB02-course-nav.html`, frames F1–F9: DSA Roadmap and Today chrome **pixel-parity** with the live app
   (manifest-driven nav groups identical to `navForPath("dsa")`), **v1 copy kept** (switcher footer "Browse all paths";
   NotFound button "Back to Today" → `/dsa/dashboard`) — any wording or target change you'd recommend is drawn as a
   labelled alternative and listed under "Decisions to confirm"; switcher with a cohort-only `preview` course and
   coming-soon rows, `coming_soon` teaser, start-path CTA, uniform NotFound for an unknown course,
   breadcrumbs matching `buildCrumbs`, one low-fidelity second-course nav, narrow rail + sheet. List the reserved segments
   `u, auth, settings, api, assets, healthz, readyz, privacy` in the notes.
6. **[X] AB03** — `AB03-revision-v2.html`, frames F1–F9: format badges ("Re-solve · 20:00", "Re-solve · mock conditions ·
   20:00", and a low-fidelity "Recall · ~5 min" non-DSA row — label from the band, minutes from
   `plan.est_minutes["touch_<format>"]`), **pattern withheld** (`xl-lock` "Pattern hidden while due")
   on due and active rows, revealed only after scoring, empty / overdue (R-SR5 per course) / all-courses states, narrow.
7. **[X] Narrow sections** — every board ends with `.bd-narrow` frames (390 px) for its key frames.
8. **[X] Self-review** — walk each frame against its cited decision (D27, D18, L18, ADR-0026 §5, ADR-0027 §1, R-SR2, D4) and
   the rollout §9 list; run the leak check (no pattern chip while live, no hidden inputs, no answers); check contrast and
   focus order notes exist on every frame. Fix, then re-check.
9. **[X] Screenshots** — serve the repo statically (e.g. `python3 -m http.server 5198 --directory design-system`, or open the
   files from disk) and capture each board full-page at **1440 px** and **390 px** (Browser pane `resize_window` +
   screenshot, or headless Chrome `--headless=new --screenshot=… --window-size=1440,900`). Save to
   `design-system/screens/v2/shots/AB0n@1440.png` / `AB0n@390.png`, each ≲ 500 KB.
10. **[X] Sprint file** — in `docs/v2/sprints/sprint-ds-m1-01.md` set tasks 1–6 ✅, task 7 🔄 "PR #N open — awaiting owner
    review", _Overall_ 🔄. **Do not edit `docs/v2/status.md`.**
11. **[X] Commit + PR, then STOP** — conventional commit `docs(design): v2 board index + AB01–AB03 (M1)` ending with the
    attribution lines; push; open the PR titled `docs(design): AB01 AB02 AB03 — M1 boards (+ v2 board index)` with each
    board's frame list and screenshots embedded via `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m1-01/design-system/screens/v2/shots/<file>?raw=true`,
    plus a "Decisions to confirm" list for any ambiguity you resolved **and every proposed visible change vs v1 for DSA**
    (each a yes/no for the owner). **Do not merge. Do not enable auto-merge.** Report
    the PR link and stop. If the owner requests changes later, apply them on the same branch.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded, or deployed. No `web/`,
  `internal/`, `curriculum/`, `deploy/` or `../infra` change in this sprint.
- **`theme.css` verbatim:** link `../../theme.css`; never copy, fork or override `ds-*`/`xl-*` rules. Dark theme only.
  Difficulty tokens Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>` only.
- **Touch only this sprint's files:** `index.html` and `board.css` are written **once** here; later design PRs never edit
  them. Board files use the index's names exactly.
- **No leaks:** boards never show a pattern while an item is live, never show hidden inputs or answers.
- **Final copy:** real strings with the real error codes; no placeholder text.
- **No status.md edit** from a design PR; the first consuming build sprint (m1-03) records the freeze.
- **Parallel sessions:** check peers' open PRs and worktrees before creating `index.html` / `board.css`.
- D34 (no alerting) and the memory-sum rule are not in play: nothing here runs in the cluster.

## Deliverables

- `design-system/screens/v2/index.html` (AB01–AB31 rows) and `design-system/screens/v2/board.css`.
- `design-system/screens/v2/AB01-coach-states.html`, `AB02-course-nav.html`, `AB03-revision-v2.html`.
- `design-system/screens/v2/shots/AB0{1,2,3}@{1440,390}.png`.
- An open PR (not merged) with screenshots, frame lists and "Decisions to confirm".

## Update status

- `docs/v2/sprints/sprint-ds-m1-01.md` Status table in the PR: tasks 1–6 ✅, task 7 🔄 (PR #), _Overall_ 🔄.
- **Not** `docs/v2/status.md`. After the owner's merge, the plan's Status-note close-out (four idempotent edits: this
  sprint's task 7 + _Overall_ ✅; the Sprint-board row; Artboards AB01–AB03 "frozen (PR #, date)"; owner event
  `ev-freeze-ds-m1-01` ✅) is made by m1-03, or by whichever consumer (m1-06, m1-10, m1-07) first finds it unset.
- No ADR expected; if you resolve a real design ambiguity (e.g. D27 confirm placement), list it under "Decisions to confirm"
  in the PR for the owner rather than writing an ADR.

## Done when (acceptance)

- [ ] `index.html` lists AB01–AB31 (AB23 dropped, AB31 outline) with the plan's file names and links; `board.css` defines only `bd-*` classes plus the `.xl-app` fill override.
- [ ] Every frame listed for AB01 (F1–F15), AB02 (F1–F9) and AB03 (F1–F9) is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] No board leaks withheld data; boards open with no JS runtime; `theme.css` is linked, not copied.
- [ ] PR open with 1440 px and 390 px screenshots of each board; **not merged by the agent**.

Shipping: **design sprint — open the PR and STOP for owner review.** This overrides AGENT.md's end-of-session
land-and-sync: do not merge, do not enable auto-merge, do not tag. The owner's merge (or explicit approval in chat) is the freeze.
