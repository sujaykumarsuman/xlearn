# Prompt — Sprint ds-m2-01 · Design M2: Touch ★, catalog/agenda, public profile v2, visibility (AB04 ★, AB05, AB06, AB22)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m2-01.md`](../sprints/sprint-ds-m2-01.md)   ·   **Milestone:** M2 (design track)   ·   **Prereqs:** none
> **Design sprint: the board PR merges on CI green, and the merge is the freeze ([D40](../feasibility.md#decisions-log-newest-first)). Nothing waits on the owner.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, including land-and-sync,
  which this sprint follows (D40): see **Ship** at the end.
- The plan: [`../sprints/sprint-ds-m2-01.md`](../sprints/sprint-ds-m2-01.md) — the frame tables in Tasks 2–5 are the brief.
- The board conventions every design sprint follows, written once by ds-m1-01: its
  [task 1 table](../sprints/sprint-ds-m1-01.md#1--board-index--shared-chrome-once-for-every-design-sprint-x) (canonical
  file names, `board.css` and its `bd-*` chrome) and [task 2](../sprints/sprint-ds-m1-01.md#2--board-scaffolding-the-conventions-x)
  (frame labels, behaviour notes, `.bd-narrow`, screenshots under `shots/`).
- Design system: [`../../../design-system/README.md`](../../../design-system/README.md) (tokens, `ds-*`/`xl-*` classes) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) (use verbatim).
- v1 references (not runnable; read for layout and copy): [`Revision.dc.html`](../../../design-system/screens/Revision.dc.html),
  [`Dashboard.dc.html`](../../../design-system/screens/Dashboard.dc.html), [`Catalog.dc.html`](../../../design-system/screens/Catalog.dc.html),
  [`Settings.dc.html`](../../../design-system/screens/Settings.dc.html); and today's shipped screens
  `web/src/screens/Revision.tsx`, `Dashboard.tsx`, `Catalog.tsx`, `UserDashboard.tsx`, `Settings.tsx`.
- Boards list and freeze rule: [rollout §9](../rollout-plan.md#9-artboards-by-milestone); public-dashboard tasks P1–P12:
  [rollout §10](../rollout-plan.md#10-public-dashboard-tasks).
- Touch behaviour: [t4 §3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts),
  [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena), [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course),
  [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed); the owner decisions in t4 §13 override the body.
- Decisions: D2, D4, D7, D14–D18, D27, D31 in the [feasibility decisions log](../feasibility.md#decisions-log-newest-first);
  [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today) (Today in minutes),
  [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas),
  [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md), [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (D31),
  [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas),
  [PRD §5.2](../../prd/xlearn-v2-prd.md#52-amended-requirements) and [§5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses).
- [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) — what becomes public, and what is never public.

## Context

M2 turns revision touches into real, server-timed attempts (practice `purpose=touch`, [m2-01](../sprints/sprint-m2-01.md)),
adds projections v2 and the `public-read` profile ([m2-02](../sprints/sprint-m2-02.md), [m2-03](../sprints/sprint-m2-03.md)),
and ships the Touch screen plus a cross-course Today budgeted in minutes ([m2-04](../sprints/sprint-m2-04.md)). The rollout
freeze rule says AB04–AB06 and AB22 are **frozen before M2a**. Per **BP3** (owner, 2026-09-24) agents draft every v2 board,
heroes included. Per **D40** (owner, 2026-09-25) launching this prompt is the owner's approval: you draft four boards, the PR
merges on CI green, and **the merge is the freeze**. The owner may review afterwards; any change is a follow-up design PR.

In M2 there is **no judge yet**: the DSA re-solve criterion (`correct_in_timer`) is self-attested and labelled *self*;
the pattern and complexity probes are server lock-ins graded against public keys (honor-grade), with correctness
hidden until the touch concludes. Judge Run/Submit (M3), the honor-probe claim and AI provisional (M4) appear only as
clearly labelled preview frames or not at all.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] none (no prerequisite beyond `depends_on`)

Then check (information, not a gate): `git fetch && git ls-tree origin/main design-system/screens/v2/`.
- If `index.html` and `board.css` exist (ds-m1-01 merged): link `board.css`, and confirm the index's hrefs for AB04/AB05/AB06/AB22
  equal the names below.
- If not: **do not create or edit them.** Inline the `bd-*` chrome from ds-m1-01's task 1–2 in each board's `<style>`; say so in the PR.
- Either way the file names are `AB04-touch.html`, `AB05-catalog-agenda.html`, `AB06-public-profile-v2.html`,
  `AB22-visibility-toggles.html` (the ds-m1-01 index's canonical names).

## Do this (in order)

1. **[X] Branch** `design/ds-m2-01` from an up-to-date `main`. Check for peers first (`gh pr list`, `git worktree list`,
   ListAgents): other design PRs may be open — you will touch only your four boards, their screenshots, this sprint's plan
   file and this sprint's own rows in `docs/v2/status.md`.
2. **[X] Scaffold** the four boards under `design-system/screens/v2/` — `AB04-touch.html`, `AB05-catalog-agenda.html`,
   `AB06-public-profile-v2.html`, `AB22-visibility-toggles.html`: `<link rel="stylesheet" href="../../theme.css">` (+ `board.css`
   when on `main`), the Google Fonts `<link>` for Inter + JetBrains Mono, a `<style>` block for board-only layout using `--ds-*`
   tokens only (never overriding a `ds-*`/`xl-*` rule). Full-screen frames stack vertically; panel-sized states sit side by
   side in a `.bd-grid`. Every frame has a label `ABnn-Fk · <state>` (e.g. `AB04-F4 · Name the pattern`), its decision refs,
   final copy, and a **behaviour notes** aside (timers, gates, error codes, focus order, contrast, keyboard, `aria-live` for
   clocks, the < 1024 px layout). Each board ends with a 390 px `.bd-narrow` section. No build step, no framework, nothing
   imported by `web/`.
3. **[X] AB04 ★ Touch** — draw frames F1–F16 exactly as the plan's Task 2 table lists (queue card without the pattern
   chip; start cover; mock-conditions cover; name-the-pattern lock-in with the 2:00 pattern clock; the M2 self-attested
   re-solve; the M3 judge-path preview frame; complexity lock-in; End-review confirm; pass / fail / abandoned results
   with "advances to Day N" / "resets to Day 1"; deadline-on-page; resume; "Couldn't verify, not counted. [Retry]";
   coach paused; < 1024 px). Correctness is never visible before a result frame. No re-implement step (D16). The clock
   never pauses (D15).
4. **[X] AB05 catalog + agenda + Today** — frames F1–F10 of Task 3: catalog with active / preview (cohort-only) /
   coming-soon / enroll states; Today budget in minutes (header exactly "Today · 55 min of 60" — m2-04's "Today · N min of
   M"); reviews first across courses (oldest first); R-SR5 per course;
   new-work split; over budget; all caught up; the "grades waiting" **slot** labelled M4; the 7-day cross-course
   agenda; mobile. Non-DSA content stays low fidelity.
5. **[X] AB06 public profile v2** — frames F1–F8 of Task 4: header totals from visible courses only; per-course rows with
   concluded / passed / total, difficulty split, mastery, first-attempt grade mix with provenance chips, judge-checked %
   (F2 a populated value labelled **M3+**; F3 the M2 state, "—", never a 0% presented as a claim), touch stats (reviews
   completed, Day-7 pass rate), **mock count only**; the one identical frame for private / unknown / suspended / erased
   (L-E); no visible courses; mobile. List the never-public facts in the behaviour notes and make sure none appears on the board.
6. **[X] AB22 visibility toggles** — frames F1–F6 of Task 5: profile Public/Private with the URL, per-course toggles with
   manifest-default hints, both confirmation modals (copy in the plan), the private state with disabled course toggles,
   saved / error, mobile.
7. **[X] Self-review checklist (before merging; nothing waits on the owner)** — walk every frame against its cited
   decisions and the rollout §9 list; confirm: `theme.css` linked verbatim (`--ds-*` tokens only, no new colours, no
   `ds-*`/`xl-*` override; difficulty tokens); no pattern chip during a due/live touch; no pre-conclusion correctness; no
   hidden inputs; nothing from the P10 never-public list; no re-implement step (D16), no clock pause (D15), mock count only
   (D31); text contrast ≥ 4.5:1. Fix, then re-check; the ticked checklist goes in the PR body. Take full-page screenshots of
   each board at **1440 px** and **390 px** wide (serve statically,
   e.g. `python3 -m http.server 5198 --directory design-system`, then
   `npx playwright screenshot --full-page --viewport-size=1440,900 http://localhost:5198/screens/v2/AB04-touch.html …`,
   or headless Chrome `--headless=new --screenshot=… --window-size=1440,900`) into
   `design-system/screens/v2/shots/AB04@1440.png`, `AB04@390.png` (and AB05, AB06, AB22), each ≲ 500 KB — committed
   (ds-m1-01's convention; they are part of the frozen reference).
8. **[X] PR body** (for **Ship** step 1) — the screenshots embedded from the branch
   (`…/blob/design/ds-m2-01/design-system/screens/v2/shots/<file>?raw=true`), a frame list per board with decision cites,
   the ticked self-review checklist, and the three **"Decisions to confirm"** from the plan's Task 7. #1 is the choice
   between strict honor-key matching until M4's claim (A) and the v1-parity interim (B): present both exactly as the plan
   words them, and state that **the merge freezes (A)** (the boards as drawn) — the owner may pick (B) after the merge (a
   follow-up design PR plus m2-01's one-constant switch). The list never blocks the merge.
9. **[X] Land it** — run **Ship** below: PR, merge on CI green (the freeze), status, sync.

## Constraints

- **Preview-only boards.** Static HTML on `theme.css` verbatim; no new colours or tokens; never imported by `web/`;
  never shipped. v1 `.dc.html` files are references only — do not edit them.
- **Touch only your files:** the four boards, their screenshots under `shots/`, this sprint's plan file and this sprint's
  own rows in `docs/v2/status.md` (Update status). Never `index.html` or `board.css` (ds-m1-01 owns them), never `theme.css`.
- **Decisions win.** Every frame cites the decisions it implements; if the plan and an ADR disagree, the ADR wins —
  note the discrepancy in the PR instead of inventing behaviour.
- **Withholding:** the pattern chip never shows during a due or live touch; answers appear only after conclusion;
  the public profile shows nothing on the P10 never-public list.
- **No `web/`, service, infra or deploy changes.** No `kubectl` of any kind. No alerting surfaces (D34).
- **Parallel sessions:** other design PRs (ds-m1-01, ds-l-01) may be open; rebase on `main` before pushing and resolve
  nothing outside your files. They also edit `status.md` when they land: rebase before the status commit and keep their rows.

## Deliverables

- `design-system/screens/v2/AB04-touch.html` (★), `AB05-catalog-agenda.html`, `AB06-public-profile-v2.html`,
  `AB22-visibility-toggles.html`.
- `design-system/screens/v2/shots/AB04@1440.png`, `AB04@390.png`, and the same pair for AB05, AB06, AB22.
- The PR, **merged on CI green** (the freeze), with the embedded screenshots, per-board frame lists, the ticked self-review
  checklist and "Decisions to confirm".
- This sprint's plan Status and its `status.md` rows updated.

## Update status

In the board PR — a last commit once the PR number is known, before the merge — or in a follow-up docs PR merged the same way:
- [`../sprints/sprint-ds-m2-01.md`](../sprints/sprint-ds-m2-01.md): tasks 1–8 ✅ (task 8: "frozen: merged in PR #N, <date>"),
  _Overall_ ✅.
- [`../status.md`](../status.md): mark the sprint ✅ (ds-m2-01's Sprint-board row) and the boards ✅ **"frozen (merged, PR #N,
  <date>)"** (Artboards rows AB04, AB05, AB06, AB22); update the Snapshot's artboard count (`ev-freeze-ds-m2-01` is
  automatic: no tick); add a Decisions-log line for "Decisions to confirm" #1 ("honor-key policy (A) frozen with ds-m2-01;
  the owner may switch to (B) after the merge").
- No ADR is expected. If the owner later changes a decision, the follow-up design PR and the build sprint that implements it
  record the change.

## Done when (acceptance)

- [ ] Every frame listed for AB04, AB05, AB06, AB22 is present with final copy and states, each citing its decision(s).
- [ ] Boards carry the canonical index names and use `../../theme.css` tokens/components only; preview-only; `index.html`, `board.css` and `theme.css` untouched.
- [ ] No board leaks withheld data (pattern chip, pre-conclusion correctness, never-public profile facts).
- [ ] The self-review checklist passed and is ticked in the PR body, with the three "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with the 1440 px and 390 px screenshots committed under `shots/` and embedded; the sprint file and `docs/v2/status.md` show ds-m2-01 ✅ and AB04/AB05/AB06/AB22 "frozen (merged)"; local `main` synced.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. **Branch, commit, push, PR** — this repo only (a design sprint touches no `../infra`). On `design/ds-m2-01` (step 1):
   conventional commit `docs(design): M2 boards AB04★ AB05 AB06 AB22` with the attribution lines from the session's system
   reminder; push; `gh pr create` titled "design: AB04★ AB05 AB06 AB22 (M2)" with the body from step 8.
2. **Merge on green** — once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — design: the merge is the freeze; no tag.** Nothing deploys (boards are preview-only). The owner may
   review after the merge; any change to a frozen board is a follow-up design PR.
4. **Update status** — as in "Update status" above (sprint ✅, AB04/AB05/AB06/AB22 "frozen (merged)"), in the same PR (a last
   commit before step 2's merge) or a follow-up docs PR merged the same way.
5. **Sync** — `git checkout main && git pull`. If a clean peer worktree holds `main`, use
   `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
