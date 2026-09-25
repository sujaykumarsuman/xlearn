# Prompt — Sprint ds-m1-01 · Design M1: coach states, course nav, revision v2 (AB01–AB03)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m1-01.md`](../sprints/sprint-ds-m1-01.md)   ·   **Milestone:** M1 (design track)   ·   **Prereqs:** none
> **Design sprint: the board PR merges on CI green, and the merge is the freeze ([D40](../feasibility.md#decisions-log-newest-first)). Nothing waits on the owner.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, including land-and-sync,
  which this sprint follows (D40): see **Ship** at the end.
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
(course nav), m1-06 (revision), m1-10 and m1-07 (coach) — must build against **frozen** boards, and the rollout
freeze rule says AB01–AB03 are frozen before M1b. The owner decided (BP3, 2026-09-24) that agents draft **every** v2
board, and (D40, 2026-09-25) that launching this prompt is the owner's approval for all of it: the board PR merges on CI
green, the merge is the freeze, and the owner may review afterwards (a change to a frozen board is a follow-up design
PR). This is the first v2 design sprint, so it also writes the board index for all
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
8. **[X] Self-review checklist (before merging; nothing waits on the owner)** — walk each frame against its cited decision
   (D27, D18, L18, ADR-0026 §5, ADR-0027 §1, R-SR2, D4) and the rollout §9 list; check `theme.css` is linked verbatim (tokens
   only, no new colours, no `ds-*`/`xl-*` override; difficulty tokens) and `board.css` defines only `bd-*` plus the `.xl-app`
   override; run the leak check (no pattern chip while live, no hidden inputs, no answers); check AB02's parity frames keep
   v1's copy; check contrast and focus order notes exist on every frame. Fix, then re-check. The ticked checklist goes in the PR body.
9. **[X] Screenshots** — serve the repo statically (e.g. `python3 -m http.server 5198 --directory design-system`, or open the
   files from disk) and capture each board full-page at **1440 px** and **390 px** (Browser pane `resize_window` +
   screenshot, or headless Chrome `--headless=new --screenshot=… --window-size=1440,900`). Save to
   `design-system/screens/v2/shots/AB0n@1440.png` / `AB0n@390.png`, each ≲ 500 KB.
10. **[X] PR body** (for **Ship** step 1) — each board's frame list; the screenshots embedded via
    `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m1-01/design-system/screens/v2/shots/<file>?raw=true`; the ticked
    self-review checklist (step 8); and a **"Decisions to confirm"** list for any ambiguity you resolved **and every proposed
    visible change vs v1 for DSA**. The list doesn't block the merge: each item states the default the merge freezes (for
    DSA, always the parity frame), and the owner may revisit any item after the merge through a follow-up design PR.
11. **[X] Land it** — run **Ship** below: PR, merge on CI green (the freeze), status, sync.

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
- **`status.md`: own rows only** — this sprint's Sprint-board row, the AB01–AB03 Artboards rows and the Snapshot's artboard
  count (`ev-freeze-ds-m1-01` is automatic: no tick).
- **Parallel sessions:** check peers' open PRs and worktrees before creating `index.html` / `board.css`; ds-m2-01 and ds-l-01
  also edit `status.md` when they land, so rebase on `origin/main` before the status commit and keep their rows.
- D34 (no alerting) and the memory-sum rule are not in play: nothing here runs in the cluster.

## Deliverables

- `design-system/screens/v2/index.html` (AB01–AB31 rows) and `design-system/screens/v2/board.css`.
- `design-system/screens/v2/AB01-coach-states.html`, `AB02-course-nav.html`, `AB03-revision-v2.html`.
- `design-system/screens/v2/shots/AB0{1,2,3}@{1440,390}.png`.
- The PR, **merged on CI green** (the freeze), with screenshots, frame lists, the ticked self-review checklist and "Decisions to confirm".

## Update status

In the board PR — a last commit once the PR number is known, before the merge — or in a follow-up docs PR merged the same way:
- `docs/v2/sprints/sprint-ds-m1-01.md`: tasks 1–7 ✅ (task 7: "frozen: merged in PR #N, <date>"), _Overall_ ✅.
- `docs/v2/status.md`: mark the sprint ✅ (ds-m1-01's Sprint-board row) and the boards ✅ **"frozen (merged, PR #N, <date>)"**
  (Artboards rows AB01, AB02, AB03); update the Snapshot's artboard count (`ev-freeze-ds-m1-01` is automatic: no tick).
- No ADR expected. If you resolve a real design ambiguity (e.g. D27 confirm placement), list it under "Decisions to confirm"
  with the default you froze, and add a Decisions-log line in `status.md` if a build sprint depends on it.

## Done when (acceptance)

- [ ] `index.html` lists AB01–AB31 (AB23 dropped, AB31 outline) with the plan's file names and links; `board.css` defines only `bd-*` classes plus the `.xl-app` fill override.
- [ ] Every frame listed for AB01 (F1–F15), AB02 (F1–F9) and AB03 (F1–F9) is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] No board leaks withheld data; boards open with no JS runtime; `theme.css` is linked, not copied.
- [ ] The self-review checklist passed and is ticked in the PR body, with "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots of each board; the sprint file and `docs/v2/status.md` show ds-m1-01 ✅ and AB01–AB03 "frozen (merged)"; local `main` synced.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. **Branch, commit, push, PR** — this repo only (a design sprint touches no `../infra`). On `design/ds-m1-01`: conventional
   commit `docs(design): v2 board index + AB01–AB03 (M1)` with the attribution lines; push; open the PR titled
   `docs(design): AB01 AB02 AB03 — M1 boards (+ v2 board index)` with the body from step 10.
2. **Merge on green** — once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — design: the merge is the freeze; no tag.** Nothing deploys (boards are preview-only). The owner may
   review after the merge; any change to a frozen board is a follow-up design PR.
4. **Update status** — as in "Update status" above (sprint ✅, AB01–AB03 "frozen (merged)"), in the same PR (a last commit
   before step 2's merge) or a follow-up docs PR merged the same way.
5. **Sync** — `git checkout main && git pull`. If a clean peer worktree holds `main`, use
   `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
