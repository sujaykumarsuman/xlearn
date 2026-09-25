# Prompt — Sprint ds-m6a-02 · Design M6a part 2: grace/paused/resume, debrief + proposal, accessibility (AB26, AB27, AB28)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m6a-02.md`](../sprints/sprint-ds-m6a-02.md)   ·   **Milestone:** M6a (design track)   ·   **Prereqs:** [ds-m6a-01](../sprints/sprint-ds-m6a-01.md), [ds-m1-01](../sprints/sprint-ds-m1-01.md)
> **Design sprint: the board PR merges on CI green, and the merge is the freeze ([D40](../feasibility.md#decisions-log-newest-first)). Nothing waits on the owner.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, including land-and-sync, which this sprint follows (D40): see **Ship** at the end.
- The plan: [`../sprints/sprint-ds-m6a-02.md`](../sprints/sprint-ds-m6a-02.md) — the frame tables (AB26 F1–F12 + F1b, AB27 F1–F12, AB28 F1–F7), file names, copy and "Decisions to confirm". Follow them exactly.
- Decisions: [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (**Accepted** by ds-m6a-01; §4 failsafes, §5 scoring, §6 privacy and limits), the [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D14, D17, D27, D30, D31, D34), [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule).
- Research detail: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (the state machine, failsafes 1–3 with their copy, checkpoints, other failure modes), [§6](../research/t6-realtime-interviewer.md#6-assessment) (debrief, review, proposal, `ai-byo` conditions, never-assessed list), [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (retention, per-turn delete, accessibility), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (meters in grace/pause/resume), [§9](../research/t6-realtime-interviewer.md#9-phased-plan) (P0 simple grace modal, P1 countdown), [§11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (D14 amendment, null bands), [§13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D30, D31); the S6 note (t6 §16) for measured costs; [PRD](../../prd/xlearn-v2-prd.md) §5.6 R-MI3–R-MI8.
- [Rollout §9](../rollout-plan.md#9-artboards-by-milestone) (the M6a row and the freeze rule) and [§4 M6a/M6b](../rollout-plan.md#m6a--m6b-v21--57--45-sprints).
- [`../../../design-system/README.md`](../../../design-system/README.md) and [`../../../design-system/theme.css`](../../../design-system/theme.css) — reuse verbatim; `design-system/screens/v2/index.html` and `board.css` from [ds-m1-01](../sprints/sprint-ds-m1-01.md).
- **ds-m6a-01's boards** (on `main` or on `design/ds-m6a-01`): `AB13-mock-v2.html` (F6 evidence | sliders, F8 chips, F9 history), `AB24-mock-setup-preflight.html` (opt-in strings, estimate and cap wording), `AB25-live-hud-text.html` (the HUD chrome). Also `AB01-coach-states.html`, `AB07-workspace-code.html` (shortcuts) if on `main`.
- Consumers: [m6a-01](../sprints/sprint-m6a-01.md) (gates on the freeze), [m6a-02](../sprints/sprint-m6a-02.md), [m6a-03](../sprints/sprint-m6a-03.md), [m6a-06](../sprints/sprint-m6a-06.md) (builds these boards), [ds-m6b-01](../sprints/sprint-ds-m6b-01.md) (extends AB26 F1 with the countdown).

## Context

M6a ships the text interviewer with the three failsafes the owner specified: a **5-minute top-up grace** when the learner's
credits run out (clock frozen, free checkpoint, hang up, probe every 30 s), a **pause of at most 24 hours** from the first pause
(≤ 3 pauses; the coach is unlocked but the item stays withheld; arena exposure is recorded), and a **resume modal** that shows the
free deterministic state at once and runs one paid summary (~$0.04) only on a click, cached for that pause. **D30:** a pause not
resumed within 24 h becomes `incomplete` — unscored, out of trends, with a free partial view and paid improvement areas only on a
click. Scoring follows T6's amendment to **D14**: the review produces an `ai-byo` proposal the learner must **explicitly** accept or
edit (no 24 h auto-submit for mocks), with one blind re-propose, then `ScoreMock` once. AB28 holds the accessibility choices: the
time multiplier (badged, never penalised, cost shown), reduced motion, screen-reader announcements and shortcuts, with voice rows
tagged "from M6b". In M6a the grace modal is the **simple** one; the countdown ring is AB30's (M6b). ADR-0032 is Accepted
(ds-m6a-01). No `web/` code here: boards are preview-only static HTML. Per D40 (owner, 2026-09-25) launching this prompt is the
owner's approval: the board PR merges on CI green and the merge is the freeze; the owner may review afterwards (a change to a
frozen board is a follow-up design PR).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [ds-m6a-01](../sprints/sprint-ds-m6a-01.md): its ADR PR is merged (**ADR-0032 Accepted** on `main`) and its board PR is **open or merged** (AB13, AB24, AB25 exist on `main` or `design/ds-m6a-01`). If it is still open (its session hasn't landed it yet), draft against its latest commit and say so in the PR.
- [ ] [ds-m1-01](../sprints/sprint-ds-m1-01.md) merged: `design-system/screens/v2/index.html` and `board.css` on `main`.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR adds `design-system/screens/v2/AB26-*`, `AB27-*` or `AB28-*`.

## Do this (in order)

1. **[X] Branch** `design/ds-m6a-02` off an up-to-date `origin/main`.
2. **[X] Read ds-m6a-01's three boards** and note the strings and components to reuse: the HUD chrome (AB25), the opt-in toggle,
   estimate and cap wording (AB24), the evidence | sliders layout and the result chips (AB13).
3. **[X] AB26** — `design-system/screens/v2/AB26-grace-paused-resume.html`, frames F1–F12 + F1b with the plan's copy: the **simple**
   grace modal ("Your OpenAI credits ran out — interview paused at 23:41.", add at least ⟨remaining + $1⟩, [Top up ↗], "We'll keep
   checking until 14:10 · checked 20 s ago", the meter, [Check now] [Save & resume later] [End interview]) with a "from M6b" countdown
   variant; the **Anthropic-key variant F1b** ("Your Anthropic credits ran out…", Anthropic's `billing_url`, no negative-balance
   line or $1.25 minimum); the ≥ 75% buttons; credits found (no paid summary); the other interruptions and `ErrAuth` → paused; the paused banner
   (absolute deadline, pause n of 3, coach available but not for this problem, arena exposure noted); the resume modal's **instant free
   state**, [Summarise & resume — uses your key (~$0.04)], [Finish now & get feedback] (≥ 75%), [Abandon]; still no credit; the brief
   (cached per pause); pause exposure; last pause and **expiry → `incomplete`**; credit out at ≥ 90%; narrow.
4. **[X] AB27** — `AB27-debrief-proposal.html`, frames F1–F12: the brain-authored debrief (**never a score**); preparing the review
   (~$0.06, hidden tests aggregate); the **proposal review** (bands or "No evidence" with a reason, one verified quote each, why,
   sliders, strengths, improvements with practice links, caveats, provenance "from your messages and code only"; [Accept all]
   [Re-propose once — uses your key (~$0.06)] [Score it myself]; "nothing is saved until you accept"); edited → self-scored;
   the two-proposal compare; saved (`AI-proposed · accepted` or `self`, always `honor`; public shows the count only); the
   self-scored-only reasons (custom model, not cleared, `review_flag` without naming the heuristic, pause exposure, "from M6b" voice
   Communication); scores waiting (never auto-saved; reminders; 30-day expiry); the **`incomplete` free partial view** with paid
   improvements only on a click; review failed / cut short; transcript per-message delete and retention; narrow.
5. **[X] AB28** — `AB28-accessibility-settings.html`, frames F1–F7: Settings → Mock interviews (the opt-in with AB24's exact strings,
   default mode, **Extra time** 1×–2× "never lowers your score, no reason needed", reduced motion, screen-reader announcements); the
   extra-time cost line; voice preferences tagged "from M6b" (captions on by default, screen-reader preset, camera self-view off);
   the text HUD under these settings; the shortcut sheet (actions listed; Finish/End have none, which departs from t6 §7's "end"
   shortcut, so list it under "Decisions to confirm"); result badges; narrow.
6. **[X] Behaviour notes** — every frame's aside names its driver (`grace_until`, `resume_by`, the probe limits, the checkpoint,
   `pause_exposure`, `report_pending`, `cut_short`, `xlearn.mock_review@1`, the `ai-byo` conditions, `ScoreMock` from `finished`,
   `mock_session.time_multiplier`), its decision ref, a11y (`role="alertdialog"`; the grace warning is the only assertive
   announcement; focus order; destructive buttons never default; switches with `aria-checked`) and the `< 1024 px` intent; codes you
   invent are marked *proposed*. Check the proposed chords against CodeMirror's `defaultKeymap` (`Mod-Enter` = `insertBlankLine`),
   AB07's shortcuts, macOS Option input and screen-reader keys; an action with no safe chord gets none. Each board ends with a 390 px section.
7. **[X] Self-review checklist (before merging; nothing waits on the owner)** — the plan's task 5: `theme.css` linked verbatim
   (tokens only, no new colours, no `ds-*`/`xl-*` override; difficulty tokens); the failsafes exactly as specified, D30, no
   auto-accept anywhere, D31, D17/D27 during a pause, the never-assessed list, D34; the leak check (own code only; no pattern
   chip, reference solution or unreleased hint; no score in the debrief); the copy check against AB13/AB24/AB25. Fix, then
   re-check. The ticked checklist goes in the PR body.
8. **[X] Screenshots**: serve statically with `python3 -m http.server 5198 --directory design-system`, then capture each
   board **full-page** at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB26@1440.png` … `AB28@390.png`, each
   ≲ 500 KB. A viewport-sized capture is not enough, and the Browser pane can't write PNGs to disk. Use:
   `npx -y playwright screenshot --channel chrome --full-page --wait-for-timeout 1000 --viewport-size "1440, 900" http://localhost:5198/screens/v2/AB26-grace-paused-resume.html design-system/screens/v2/shots/AB26@1440.png`
   (and `--viewport-size "390, 844"` for `@390`). **Fallback:** read the board's `document.documentElement.scrollHeight` at that
   width (Browser pane `resize_window` + `javascript_tool`), then run
   `"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --hide-scrollbars --window-size=1440,<scrollHeight> --screenshot=<file> <url>`.
   Open each PNG to check it holds the whole board.
9. **[X] PR body** (for **Ship** step 1) — the frame lists, embedded screenshots
   (`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6a-02/design-system/screens/v2/shots/<file>?raw=true`), a link to
   ds-m6a-01's PR, the ticked self-review checklist and **"Decisions to confirm"** (the plan's list), each item stating the
   default the merge freezes (the boards as drawn). The list never blocks the merge; the owner may revisit any item afterwards
   through a follow-up design PR.
10. **[X] Land it** — run **Ship** below: PR, merge on CI green (the freeze), status, sync.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`, `internal/`, `cmd/`, `deploy/`, `curriculum/` or `../infra` change.
- **`theme.css` verbatim:** link `../../theme.css` then `board.css`; never copy, fork or override `ds-*`/`xl-*`. Dark theme only; Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; board CSS uses `--ds-*` tokens only.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>`.
- **Touch only this sprint's files:** three boards + six screenshots, plus this sprint's plan file and its own `status.md` rows (Update status). Never `index.html`, `board.css`, ds-m6a-01's boards or any other sprint's board.
- **Money on buttons:** every paid action states its price and runs only on a click; nothing spends the key by itself. BYO amounts are dollars on the learner's own key (t6 §8); the platform allowance's percent rule (AB18) is untouched.
- **No auto-accept, no public score:** mocks are saved only on an explicit accept (T6's D14 amendment); the public profile shows the count only (D31).
- **No leaks:** own code only in resume and partial views; no pattern chip, reference solution or unreleased hint; the debrief never states a score; `review_flag` copy never names the heuristic.
- **D34:** no copy implying anyone is alerted. GitOps, the memory-sum rule, consumers-before-producers, ACL-before-tag, goose/sqlc, outbox/inbox and service boundaries are n/a (nothing runs).
- **Parallel sessions:** check peers' open PRs and worktrees before creating the board files and again before pushing; rebase before the status commit and keep other sessions' `status.md` rows.

## Deliverables

- `design-system/screens/v2/AB26-grace-paused-resume.html`, `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html`.
- `design-system/screens/v2/shots/AB2{6,7,8}@{1440,390}.png`.
- The PR, **merged on CI green** (the freeze), with frame lists, screenshots, the ticked self-review checklist and "Decisions to confirm".

## Update status

In the board PR — a last commit once the PR number is known, before the merge — or in a follow-up docs PR merged the same way:
- `docs/v2/sprints/sprint-ds-m6a-02.md`: tasks 1–7 ✅ (task 7: "frozen: merged in PR #N, <date>"), _Overall_ ✅.
- `docs/v2/status.md`: mark the sprint ✅ (ds-m6a-02's Sprint-board row) and the boards ✅ **"frozen (merged, PR #N, <date>)"** (Artboards rows AB26, AB27, AB28); update the Snapshot's artboard count (`ev-freeze-ds-m6a-02` is automatic: no tick).
- No ADR expected. Real ambiguities (which proposal may be accepted after a re-propose; the simple grace modal) go under "Decisions to confirm", with the default you froze.

## Done when (acceptance)

- [ ] Every frame listed for AB26 (F1–F12 + F1b), AB27 (F1–F12) and AB28 (F1–F7) is present with final copy, its state, its decision refs and a behaviour-notes aside.
- [ ] The failsafes read exactly as specified (5-minute grace, 24 h absolute resume deadline, ≤ 3 pauses, free state first, paid brief only on a click and cached), `incomplete` is unscored and out of trends, and no frame auto-saves a mock score.
- [ ] No board leaks withheld data, states a score in the debrief, or shows a mock score publicly.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied; only the three board files and their screenshots are added.
- [ ] The self-review checklist passed and is ticked in the PR body, with "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots; the sprint file and `docs/v2/status.md` show ds-m6a-02 ✅ and AB26, AB27, AB28 "frozen (merged)"; local `main` synced.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. **Branch, commit, push, PR** — this repo only (a design sprint touches no `../infra`). On `design/ds-m6a-02` (step 1): commit
   `docs(design): AB26 AB27 AB28 — M6a boards, part 2` with the attribution lines; push; open `docs(design): AB26 AB27 AB28 — M6a
   boards (part 2)` with the body from step 9.
2. **Merge on green** — once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — design: the merge is the freeze; no tag.** Nothing deploys (boards are preview-only). The owner may review
   after the merge; any change to a frozen board is a follow-up design PR.
4. **Update status** — as in "Update status" above (sprint ✅, AB26/AB27/AB28 "frozen (merged)"), in the same PR (a last commit
   before step 2's merge) or a follow-up docs PR merged the same way.
5. **Sync** — `git checkout main && git pull`. If a clean peer worktree holds `main`, use
   `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
