# Prompt — Sprint ds-m4-01 · Design M4: AI suggestion/dispute ★, pointer notes, allowance + consents (AB16 ★, AB17, AB18)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m4-01.md`](../sprints/sprint-ds-m4-01.md)   ·   **Milestone:** M4 (design track)   ·   **Prereqs:** none
> **Design sprint: the board PR merges on CI green, and the merge is the freeze ([D40](../feasibility.md#decisions-log-newest-first)). Nothing waits on the owner.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, including land-and-sync,
  which this sprint follows (D40): see **Ship** at the end.
- The plan: [`../sprints/sprint-ds-m4-01.md`](../sprints/sprint-ds-m4-01.md) — the frame tables (AB16 F1–F16, AB17 F1–F10,
  AB18 F1–F11), file names, copy and the "Decisions to confirm" list are spelled out there. Follow them exactly.
- [`../rollout-plan.md`](../rollout-plan.md) [§9 Artboards by milestone](../rollout-plan.md#9-artboards-by-milestone) (M4 row, freeze
  rule), [§3](../rollout-plan.md#3-milestone-map) (M4 entry: AB16–AB18 frozen), [§4 M4](../rollout-plan.md#4-per-milestone-detail),
  [§10](../rollout-plan.md#10-public-dashboard-tasks) (P7 provenance, P10 public shape).
- Decisions: [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D14 AI
  suggestions, pass review), [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §4 (consents), §5 (percentage, degrade order),
  §6 (D26), §7 (UI names), the [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D14, D16, D18, D26, D27, D31, D34).
- Research detail: [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade),
  [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)
  ("Provisional and claim"), [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only),
  [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls)
  (learner view), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) items 2 and 5,
  [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (consent wording, C3 notes), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2),
  [t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream) (the artboard list); [PRD](../../prd/xlearn-v2-prd.md) §5.6 R-AI1–R-AI6.
- [`../../../design-system/README.md`](../../../design-system/README.md) (tokens, components) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) — reuse verbatim.
- Neighbouring boards, if on `main` (read-only; match their vocabulary and copy): `design-system/screens/v2/index.html` and
  `board.css` ([ds-m1-01](../sprints/sprint-ds-m1-01.md)), `AB01-coach-states.html` (F13 keys panel, F14 onboarding),
  `AB04-touch.html` (F10 fail result), `AB05-catalog-agenda.html` (F8 grades-waiting slot), `AB08-results-dock.html`,
  `AB11-degradation-badges.html`, `AB12-week-mistakes-progress.html` (provenance chips), `AB19-invite-acceptance.html` (consents).
- v1 references (read-only, **not runnable**): `design-system/screens/{Settings,Revision,Problem,Mistakes,Dashboard}.dc.html`.
- Live-app structure (read-only): `web/src/screens/{Settings,Revision,Problem,Mistakes,Dashboard}.tsx`, `web/src/styles/app.css`.

## Context

M4 turns platform AI on for the owner cohort: an analyzer that suggests mistake categories, concepts and pointer notes; AI
results as **suggestions** the learner accepts, edits within the deterministic ceilings, or disputes once (D14); a per-account
allowance shown as a **percentage**; and three withdrawable consents. The rollout freeze rule says AB16–AB18 are frozen before
the M4 UI sprint, and M4's entry gate ([m4-01](../sprints/sprint-m4-01.md)) requires the freeze. The owner decided (BP3,
2026-09-24) that agents draft every v2 board — the AB16 hero included — and (D40, 2026-09-25) that launching this prompt is the
owner's approval: the board PR merges on CI green and the merge is the freeze; the owner may review afterwards (a change to a
frozen board is a follow-up design PR). No `web/` code here;
boards are preview-only static HTML. In v2.0 no course item carries an `ai_rubric` step, so AB16's rubric frames are drawn on an
illustrative item and labelled (F1, F5–F12 — flagged, AI-unavailable and AI-off manual entry need `Score`, which no v2.0 item
calls); the frames the owner meets in v2.0 are the analyzer suggestion on a deterministic grade (F2), the honor-probe claim on a
touch (F3–F4), accepted/auto-accepted (F13), grades waiting (F14) and the provenance legend (F15).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] None beyond `depends_on` (empty).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR adds or edits
      `design-system/screens/v2/AB16-*`, `AB17-*` or `AB18-*`. If one does, stop and report.
- _Informational:_ if `design-system/screens/v2/index.html` is on `main`, use the file names it links for AB16–AB18 and link
  `board.css`; if it is not, **do not create either file** — use the plan's names and a board-local `<style>`.

## Do this (in order)

1. **[X] Branch** `design/ds-m4-01` off an up-to-date `origin/main`.
2. **[X] Read the neighbours** that are on `main` (AB01, AB04, AB05, AB08, AB11, AB12, AB19) and note their component choices and
   strings (accepted-answer line, grades-waiting card, badge style, provenance chips, consent strings). Anything you must diverge
   from goes into "Decisions to confirm".
3. **[X] AB16 ★** — `design-system/screens/v2/AB16-ai-suggestion-dispute.html`, frames F1–F16 from the plan with the exact copy:
   provisional card with the server-deadline countdown and [Accept] [Edit…] [Dispute…]; the **analyzer suggestion on a
   deterministic grade** (grade locked, category + concepts + summary, [Use this] [Change…]); the touch honor-probe claim
   ([I meant this] once) and its applied state; the bounded edit per D18/D27 (a late pass: Clean disabled, "caps this attempt at
   Rough"; the hint variant: Clean **and** Rough disabled, "Your hint caps this attempt at Assisted"; deterministic steps locked;
   F9 names the same ceiling as F5); the dispute modal (reason codes, ≤ 500 chars, "the re-grade doesn't read it"); re-grade running; compare; override
   confirm naming **self-set** and the **judge-checked %** exclusion; flagged (no auto-accept, heuristic never named); AI
   unavailable (Wait vs Grade it myself); manual entry; accepted/auto-accepted; Today "grades waiting"; provenance legend; narrow.
   Label rubric frames (F1, F5–F12) "rubric items — contract built in M4, first used by a rubric course" and the live-in-v2.0
   frames (F2–F4, F13–F15) as such. The `review_flag` variant of F10 uses the `honor` chip ("counts on your honour (not
   judge-checked)"), never `self-set` — that label is for a learner override.
4. **[X] AB17** — `AB17-pointer-notes.html`, frames F1–F10: collapsed by default; expanded with line-range tags highlighting the
   learner's own code (≤ 8 lines shown); optional revisit off the ladder; the "correct, with improvements" revision row;
   **hidden during a live touch/open attempt** (`xl-lock`); near-identical resubmission; nothing to improve; unavailable
   variants (consent off / paused / AI off); where notes appear and "never on your public profile"; narrow.
5. **[X] AB18** — `AB18-ai-allowance-consents.html`, frames F1–F11: the meter in **percent only** (ok, low, paused for the account,
   daily limit, paused for everyone as an AB11-style badge, off for the account); the three Settings consent toggles (unticked
   by default, the second disabled until the first is on, the behavioral opt-in separate) — the first two **labels copied
   verbatim from AB19 F9** ("xLearn AI reviews my graded work" / "…and also reviews my passing solutions for improvement notes";
   helper lines may be Settings-specific) — with AB19's retention sentence (Anthropic / outside India / kept by them for up to
   30 days / never used for training) and the privacy-notice link; switch-off with no confirm; the two names side by side; AI-off chips
   elsewhere; narrow.
6. **[X] Behaviour notes** — every frame's aside names the server field or error code that drives it (`accept_deadline_at`,
   `409 dispute_used`, `409 not_provisional`, `503 ai_unavailable`, `inconclusive(budget_exhausted|ai_unavailable)`,
   `self_grade_pending`, `used_pct`, `resets_at`, `state`, `global_paused`), its decision ref, a11y (focus order, contrast,
   `aria-live` only at 1 h and 5 min, `role="switch"`), and the `< 1024 px` intent. Every board ends with a 390 px section.
7. **[X] Self-review checklist (before merging; nothing waits on the owner)** — check `theme.css` is linked verbatim (then
   `board.css`; `--ds-*` tokens only, no new colours, no `ds-*`/`xl-*` override; difficulty tokens); walk each frame against
   its decision (D14; D18/D27 ceilings — Clean disabled when late or > 3 failed submits, Clean **and** Rough disabled after a
   hint or coach help on that problem; D26; ADR-0031 §4–§7; D31 — no mock best/average on any public surface; D34) and the
   rollout §9 M4 row. Run the checks: **no "$" on any learner surface**; no pattern/concepts/notes while live; no hidden
   inputs, anchors, must-cover lists or exemplars; accepted probe answers only after conclusion; manual entry reachable from
   every AI state; no copy implying anyone was alerted. Fix, then re-check. The ticked checklist goes in the PR body.
8. **[X] Screenshots** — serve statically (e.g. `python3 -m http.server 5198 --directory design-system`, or open from disk) and
   capture each board full-page at **1440 px** and **390 px** (Browser pane `resize_window` + screenshot, or headless Chrome
   `--headless=new --screenshot=… --window-size=1440,900`). Save to `design-system/screens/v2/shots/AB16@1440.png`, `AB16@390.png`,
   … `AB18@390.png`, each ≲ 500 KB.
9. **[X] PR body** (for **Ship** step 1) — each board's frame list, the screenshots embedded via
   `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m4-01/design-system/screens/v2/shots/<file>?raw=true`, the ticked
   self-review checklist, and **"Decisions to confirm"** (at least: edit-before-re-grade follows D14 over t4 §3.5; percent-only
   allowance vs m4-05's wording; daily reset shown in local time; the `review_flag` and per-account-disable copy; any divergence
   from AB04/AB05/AB11/AB12/AB19), each stating the default the merge freezes (the boards as drawn). The list never blocks the
   merge; the owner may revisit any item afterwards through a follow-up design PR.
10. **[X] Land it** — run **Ship** below: PR, merge on CI green (the freeze), status, sync.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`, `internal/`,
  `cmd/`, `deploy/`, `curriculum/` or `../infra` change in this sprint.
- **`theme.css` verbatim:** link `../../theme.css`; never copy, fork or override `ds-*`/`xl-*` rules. Dark theme only. Difficulty
  tokens Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. Board layout CSS uses `--ds-*` tokens only.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>` only.
- **Touch only this sprint's files:** the three boards, their six screenshots, this sprint's plan file and this sprint's own
  rows in `docs/v2/status.md` (Update status). Never edit `index.html` or `board.css` (written once by ds-m1-01) or another
  sprint's board.
- **No leaks:** boards never show a pattern, concepts or notes while an item is live; never hidden inputs, rubric anchors,
  must-cover lists or exemplars; accepted probe answers only after a touch concluded.
- **No dollars** on any learner surface (ADR-0031 §5); **no alert promises** (D34); **no mock best/average on any public
  surface** (D31).
- **Final copy:** real strings with the real error codes; no placeholder text.
- **Parallel sessions:** check peers' open PRs and worktrees before creating the board files; rebase on `origin/main` before the
  status commit and keep other sessions' `status.md` rows. The memory-sum rule and GitOps are not in play: nothing here runs in
  the cluster.

## Deliverables

- `design-system/screens/v2/AB16-ai-suggestion-dispute.html`, `AB17-pointer-notes.html`, `AB18-ai-allowance-consents.html`.
- `design-system/screens/v2/shots/AB1{6,7,8}@{1440,390}.png`.
- The PR, **merged on CI green** (the freeze), with frame lists, screenshots, the ticked self-review checklist and "Decisions to confirm".

## Update status

In the board PR — a last commit once the PR number is known, before the merge — or in a follow-up docs PR merged the same way:
- `docs/v2/sprints/sprint-ds-m4-01.md`: tasks 1–7 ✅ (task 7: "frozen: merged in PR #N, <date>"), _Overall_ ✅.
- `docs/v2/status.md`: mark the sprint ✅ (ds-m4-01's Sprint-board row) and the boards ✅ **"frozen (merged, PR #N, <date>)"**
  (Artboards rows AB16, AB17, AB18); update the Snapshot's artboard count (`ev-freeze-ds-m4-01` is automatic: no tick).
- No ADR expected. If you resolve a real ambiguity (e.g. edit-before-re-grade), list it under "Decisions to confirm" with the
  default you froze, rather than writing an ADR.

## Done when (acceptance)

- [ ] Every frame listed for AB16 (F1–F16), AB17 (F1–F10) and AB18 (F1–F11) is present with final copy, its state, its decision refs
      and a behaviour-notes aside.
- [ ] No board shows dollars to the learner, leaks withheld data or names the injection heuristic; manual entry is reachable from
      every AI state.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied; only the three board files and their screenshots are added.
- [ ] The self-review checklist passed and is ticked in the PR body, with "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots of each board; the sprint file and
      `docs/v2/status.md` show ds-m4-01 ✅ and AB16–AB18 "frozen (merged)"; local `main` synced.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. **Branch, commit, push, PR** — this repo only (a design sprint touches no `../infra`). On `design/ds-m4-01` (step 1):
   conventional commit `docs(design): AB16 AB17 AB18 — M4 platform-AI boards` ending with the attribution lines; push; open the
   PR titled `docs(design): AB16 ★ AB17 AB18 — M4 boards` with the body from step 9.
2. **Merge on green** — once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — design: the merge is the freeze; no tag.** Nothing deploys (boards are preview-only). The owner may review
   after the merge; any change to a frozen board is a follow-up design PR.
4. **Update status** — as in "Update status" above (sprint ✅, AB16–AB18 "frozen (merged)"), in the same PR (a last commit before
   step 2's merge) or a follow-up docs PR merged the same way.
5. **Sync** — `git checkout main && git pull`. If a clean peer worktree holds `main`, use
   `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
