# Prompt — Sprint ds-p-01 · Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-p-01.md`](../sprints/sprint-ds-p-01.md)   ·   **Milestone:** P (design track)   ·   **Prereqs:** [ds-m1-01](../sprints/sprint-ds-m1-01.md), [ds-m2-01](../sprints/sprint-ds-m2-01.md), [ds-m3-01](../sprints/sprint-ds-m3-01.md) merged; [spk-02](../sprints/sprint-spk-02.md)'s P3 result recorded
> **This is a design sprint: it starts by asking the owner PRD Q5, and it ends with an open board PR and a STOP for owner review. You never merge the board PR.**

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions. The land-and-sync
  directive does **not** apply in this session (see the last line). The board PR is never merged by you. The small Q5
  record PR is merged only if the owner explicitly says yes to merging it.
- The plan: [`../sprints/sprint-ds-p-01.md`](../sprints/sprint-ds-p-01.md) — the frame tables in tasks 3–5 are the brief.
- Conventions and file names: `design-system/screens/v2/index.html` (its Conventions block) and `board.css`, both from
  [ds-m1-01](../sprints/sprint-ds-m1-01.md); the frozen boards you extend or build on: `AB02-course-nav.html`,
  `AB05-catalog-agenda.html`, `AB04-touch.html`, `AB07-workspace-code.html`, `AB08-results-dock.html`.
- Design system: [`../../../design-system/README.md`](../../../design-system/README.md) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) (verbatim). v1 references (not runnable):
  [`Problem.dc.html`](../../../design-system/screens/Problem.dc.html), [`Catalog.dc.html`](../../../design-system/screens/Catalog.dc.html).
- Q5 inputs: [PRD §7 Q5](../../prd/xlearn-v2-prd.md#7-open-questions-routed-to-topics); [rollout §4 P](../rollout-plan.md#4-per-milestone-detail),
  [§9](../rollout-plan.md#9-artboards-by-milestone) (P row, freeze rule), [§13](../rollout-plan.md#13-open-owner-items);
  [t3 §16.2](../research/t3-sandbox.md) (spk-02's P3 amd64 replay: the TSAN verdict and ASLR policy);
  [t1 §7.5](../research/t1-content-data-model.md#75-what-comes-first); [t0 §6](../research/t0-extensibility-frame.md#6-six-course-fit).
- Quiz: [t4 §4.3](../research/t4-judge-contract.md#43-archetype-c-quiz--key), [t4 §6.2](../research/t4-judge-contract.md#62-strategies-a-closed-go-set-the-manifest-picks-one-and-sets-thresholds),
  [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only), [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course).
- go-concurrency: [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (go-concurrency and SQL rows),
  [t3 §5.8](../research/t3-sandbox.md#58-goroutine-dump-go-race), [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them),
  [t3 §6.3](../research/t3-sandbox.md#63-the-sql-sandbox-sql-pg18-pilot) (SQL fallback), [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (per-course table).
- Flows and shell: [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) and the owner overrides in
  [t4 §13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D14–D18).
- Decisions: D2, D4, D10, D15–D18 in the [feasibility decisions log](../feasibility.md#decisions-log-newest-first);
  [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today);
  [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule);
  [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (`preview`);
  [PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses); [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P7).

## Context

Milestone **P** ships the second course as **manifest + content with widget/profile code only**: go-concurrency
(recommended; PRD Q5 stays open until the owner confirms it at P entry), as a `preview` course visible only to the
owner/tester cohort until GA. It needs the **go-race** runner profile ([p-01](../sprints/sprint-p-01.md), `runner-v1.1.0`),
the multi-course catalog/agenda/nav and a multi-file workspace ([p-02](../sprints/sprint-p-02.md)), and the A7 quiz widget
plus the race verdict views ([p-03](../sprints/sprint-p-03.md), `v1.15.0`). The rollout freeze rule says **AB14–AB15 are
frozen before P**, and AB02/AB05 come to full fidelity here. Per **BP3** (owner, 2026-09-24) agents draft every board and
the owner only reviews. This session: ask Q5, record it, draft four boards' worth of frames, open the PR, stop.

go-concurrency results are **honor-grade**: the hidden tests run inside the learner's program (`gotest@1`), so results are
labelled honor and excluded from judge-checked %. The runner returns RACE / DEADLOCK / LEAK / WA / TLE / RE / CE, a
goroutine dump on timeouts reduced to learner-package function names and file:line, and no test output in Submit.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **PRD Q5 confirmed by the owner** (go-concurrency, or the SQL fallback) — you ask it in step 2, **before AB15 is drafted** (`ev-q5`)
- [ ] spk-02's **P3 result is recorded** in t3 §16.2 (`git fetch && git show origin/main:docs/v2/research/t3-sandbox.md | grep -n '16.2'`)
- [ ] **AB02 and AB05 frozen** (ds-m1-01 and ds-m2-01 merged): `git ls-tree origin/main design-system/screens/v2/`
- [ ] **AB07★ and AB08 frozen** (ds-m3-01 merged)

If a board you extend is missing from `main`, stop and report. If `index.html` links different file names than the plan,
use the index's names and say so in the PR.

## Do this (in order)

1. **[X] Sync and check peers.** `git checkout main && git pull`; `gh pr list`, `git worktree list`, ListAgents — other
   design or build PRs may be open; you will touch only your files.
2. **[O] Ask PRD Q5 in chat.** Give the owner three lines: the P3 TSAN verdict from t3 §16.2; the recommendation
   (go-concurrency: reuses Go + the code widget + the quiz, honor-grade; SQL: cheapest *checked* course, heaviest runner
   profile); the consequence (SQL → AB15 becomes the SQL explorer and p-01 is re-planned to `sql-pg` before it starts). If
   P3 failed for go-race, say go-concurrency needs a redesign or SQL. In the **same message**, ask: "OK to merge the Q5
   record PR (PRD §7, rollout §13, status.md) once CI is green?" **Wait for the answer.** If none comes in-session, do
   steps 4–5 (course-agnostic) and stop before step 6, reporting why.
3. **[X] Record Q5 in its own docs PR.** Branch `docs/ds-p-01-q5`:
   - `docs/prd/xlearn-v2-prd.md` §7: strike Q5 and add "**Resolved (owner, <date>):** <course>";
   - `docs/v2/rollout-plan.md` §13: the PRD Q5 row marked resolved;
   - `docs/v2/status.md`: owner event `ev-q5` ✅ (date), a Decisions-log line (course, the P3 input, fallback status), and
     only if SQL: a flag on the p-01 board row "re-plan to `sql-pg` before start".
   Commit `docs(v2): PRD Q5 resolved — pilot course <slug> (ev-q5)` with the attribution lines; push; PR.
   - **Merge only if the owner explicitly said yes** to the merge question in step 2 (it edits the PRD and the rollout
     plan; a Q5 answer alone isn't approval, per BP3): on green CI, squash-merge, then `git checkout main && git pull`.
   - Otherwise leave it open, note its URL for the board PR body and your report, and go on from the current `main`.
     p-01's entry gate waits for the owner's merge, and p-01's Record task sets `ev-q5` ✅ if this PR didn't.
4. **[X] Branch** `design/ds-p-01` from the updated `main`. Scaffold per the index Conventions: `../../theme.css` then
   `board.css`, the Google Fonts link, `bd-*` chrome, tokens only in the file's `<style>`, frame labels
   `AB14-F3 · <state>` / `AB02-P1 · <state>`, final copy, decision refs and a behaviour-notes aside per frame (timers,
   gates, exact error copy, focus order, contrast, keyboard, `aria-live`, < 1024 px), a closing `.bd-narrow` section. No JS runtime.
5. **[X] AB14** `AB14-workspace-quiz.html` — frames F1–F14 exactly as the plan's task 3 table: cover; answering with
   "4 / 6 answered"; one card per widget mode (single, multi, order with Alt+↑/↓, text, numeric + unit, big-O,
   predict-output); submit confirm with the unanswered warning; grading; concluded pass and miss with **keys and rationale
   only now**, `weighted_gate@1` bands and "Judge · checked" provenance. On the miss (F7), show the misconception label.
   "Mistake entry opened" appears **only for a revisable item**, and the pilot drill variant (`gc-008`) shows none (I8:
   drills are non-revisable, `revision.drills false`). Then: give up; deadline; the honor-key variant; arena study mode per
   **D17** (reveal behind a spoiler confirm, recorded, uncapped); the go-concurrency **recall touch** on a revisable code
   item in the AB04 shell (lock-ins, no correctness until the result; its "Not passed" result carries the
   `misconception:*` → category mistake pre-fill); couldn't grade; mobile.
6. **[X] AB15** `AB15-go-concurrency-race.html` — frames F1–F16 of task 4: the multi-file workspace (editable vs read-only
   files, the exported API in the statement, "honor · tests run in-process"); Run with visible tests and your own output;
   a race in a visible test; Submit running (20×, up to 45 s); RACE, DEADLOCK with the grouped goroutine dump, TLE,
   LEAK, WA, CE (and hidden-API CE), REJECTED, inconclusive; concluded Clean with the honor tooltip; the L4–5 re-solve
   touch; the race → Data race pre-fill; mobile. Show **only learner-editable files** in any frame list. **If Q5 = SQL**,
   draw the SQL explorer set from the plan instead (same file name, title "AB15 · SQL explorer (PRD Q5 → SQL)").
7. **[X] AB02 / AB05 full fidelity** — append a `bd-section` "P · full fidelity (ds-p-01)" to `AB02-course-nav.html`
   (P1–P7) and `AB05-catalog-agenda.html` (P1–P7) with DSA + Go Concurrency Patterns (`preview`, 10 exercises, 2 weeks):
   manifest nav without Mock, the Preview badge, item crumbs, the non-cohort NotFound and switcher, the preview CTA; the
   catalog for cohort vs non-cohort; the two-course agenda in minutes, "nothing fits", R-SR5 per course, the 7-day agenda.
   Label the older low-fidelity second-course frames "superseded by P·n"; change nothing else in the frozen frames.
8. **[X] Self-review.** Walk every frame against its decision (D10, D15–D18, P7, I8) and the rollout §9 P row. Leak check: no
   key or rationale before a counted conclusion; no hidden test name, source, line or output; no hidden-test frame in a
   dump or race report; no topic chip on a live item; the pilot absent from every non-cohort frame. Decision check (I8):
   no mistake entry and no revision line on a non-revisable drill. Pre-fill appears only on revisable items (AB14 F12,
   AB15 F15). Contrast ≥ 4.5:1.
   Full-page screenshots at **1440** and **390** px into `design-system/screens/v2/shots/` (`AB14@1440.png`,
   `AB15@1440.png`, `AB02-p@1440.png`, `AB05-p@1440.png`, and the `@390` pairs; each ≲ 500 KB), e.g.
   `npx playwright screenshot --full-page --viewport-size=1440,1024 file://$PWD/design-system/screens/v2/AB14-workspace-quiz.html design-system/screens/v2/shots/AB14@1440.png`.
9. **[X] Commit and open the PR.** Commit `docs(design): P boards AB14 AB15 + AB02/AB05 full fidelity` with the
    attribution lines; push; `gh pr create` titled `docs(design): AB14 AB15 (+ AB02/AB05 full fidelity) — P boards`
    (append "· AB15 = SQL explorer" if Q5 flipped). Body: embedded screenshots, a frame list per board with decision
    cites, the self-review checklist, open questions (at least: D17's arena reveal vs t4 §4.3's "never in the arena"),
    and the Q5 record PR's link and state (merged, or open for the owner).
10. **[X] Update the plan's Status, then STOP.** Now that the PR number is known, set the plan's Status on the same
    branch: tasks 1–7 ✅, task 8 ⬜, _Overall_ 🔄 "PR #N open, awaiting owner review", with the real number. Commit it
    (`docs(design): ds-p-01 status — PR #N open`, attribution lines) and push to the same PR. **Do not merge. Do not
    enable auto-merge.** Report the PR URL (and the Q5 record PR's URL if it's still open) and stop.

## Constraints

- **Preview-only boards.** Static HTML on `theme.css` verbatim plus `board.css`; no new colours or tokens; never imported
  by `web/`, never embedded, never shipped; v1 `.dc.html` files are references only.
- **Touch only your files** in the board PR: `AB14-workspace-quiz.html`, `AB15-go-concurrency-race.html`, the appended
  sections of `AB02-course-nav.html` / `AB05-catalog-agenda.html`, their `shots/`, and this sprint's plan file. Never
  `index.html`, `board.css`, `theme.css` or `docs/v2/status.md` (the Q5 record PR is the only `status.md` edit).
- **Decisions win.** Cite the decision per frame; where t4's body and an owner decision differ, draw the decision and flag it.
- **Withholding** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule)): keys and
  rationale only after a counted conclusion; hidden tests never named or shown; dump/race frames filtered to learner files.
- **P = widget/profile code only.** Don't design pilot-only service behaviour (a new grade rule, a mock, a new dock state
  outside AB08's set); reuse the AB04/AB07/AB08 mechanics.
- **No `web/`, service, infra or deploy changes; no `kubectl` of any kind; no alerting surface (D34).** Service boundaries
  ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), goose/sqlc, outbox/inbox, the memory-sum rule,
  consumers-before-producers and ACL-before-consumer don't apply: this session changes no code.
- **Parallel sessions:** other design PRs may be open; rebase on `main` before pushing and resolve nothing outside your files.

## Deliverables

- A docs PR recording PRD Q5 (PRD §7, rollout §13, `status.md` `ev-q5`): merged if the owner explicitly approved the merge, otherwise open for the owner.
- `design-system/screens/v2/AB14-workspace-quiz.html`, `AB15-go-concurrency-race.html` (or the SQL explorer), and the
  "P · full fidelity" sections in `AB02-course-nav.html` / `AB05-catalog-agenda.html`, with screenshots in `shots/`.
- An open board PR with per-board frame lists, the self-review checklist and open questions.

## Update status

- [`../sprints/sprint-ds-p-01.md`](../sprints/sprint-ds-p-01.md) on the design branch: tasks 1–7 ✅ as they land, _Overall_
  🔄 "PR #N open, awaiting owner review"; leave task 8 ⬜.
- [`../status.md`](../status.md): **only** through the Q5 record PR (`ev-q5` ✅, Decisions log, the p-01 flag if SQL).
  After the owner merges the board PR, [p-01](../sprints/sprint-p-01.md) sets AB14, AB15, AB02 (full) and AB05 (full) to
  "frozen (PR #N, date)", marks task 8 ✅ and this sprint ✅.
- No ADR is expected. If the owner's review changes a decision (e.g. the arena reveal for quizzes), the build sprint that
  implements it records the ADR.

## Done when (acceptance)

- [ ] PRD Q5 is answered by the owner and recorded before AB15 was drafted.
- [ ] Every frame of AB14 (F1–F14), AB15 (F1–F16 or the SQL explorer set), AB02-P1–P7 and AB05-P1–P7 is present with final copy, its state and a cited behaviour-notes aside.
- [ ] No board leaks withheld data, and no non-cohort frame shows the pilot.
- [ ] Boards use `../../theme.css` + `board.css` verbatim and open with no JS runtime; `index.html`, `board.css`, `theme.css` and `status.md` are untouched by the board PR.
- [ ] Board PR open with 1440 px and 390 px screenshots; **not merged by the agent**.

**Shipping:** this is a **design sprint** — the AGENT.md land-and-sync directive is replaced by this sprint's release
action: **open the board PR and STOP for owner review.** Do not merge it, do not enable auto-merge, do not tag. The
owner's approval + merge is the freeze. (The small Q5 record PR in step 3 is docs-only. It is merged on green only
if the owner explicitly said yes to merging it; otherwise it stays open for the owner.)
