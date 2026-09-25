# Prompt — Sprint ds-p-01 · Design P: Workspace-Quiz, go-concurrency multi-file + race, AB02/AB05 full fidelity (AB14, AB15)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-p-01.md`](../sprints/sprint-ds-p-01.md)   ·   **Milestone:** P (design track)   ·   **Prereqs:** [ds-m1-01](../sprints/sprint-ds-m1-01.md), [ds-m2-01](../sprints/sprint-ds-m2-01.md), [ds-m3-01](../sprints/sprint-ds-m3-01.md) merged; [spk-02](../sprints/sprint-spk-02.md)'s P3 result recorded
> **Design sprint: PRD Q5 is confirmed by the owner before launch (below); the Q5 record PR and the board PR both merge on CI green, and the board merge is the freeze ([D40](../feasibility.md#decisions-log-newest-first)). Nothing waits on the owner mid-session.**

## Before you launch (owner)

- [ ] **PRD Q5 — the pilot course (`ev-q5`).** Default: **go-concurrency** (recommended; SQL is the fallback). Launching this
      prompt confirms the default; to choose SQL instead, say so in the launch message. The input is spk-02's P3 verdict in
      [t3 §16.2](../research/t3-sandbox.md): if go-race failed even with a per-process `setarch -R` launcher, the session
      applies the SQL fallback on its own (step 2).

Launching attests the item above is done (D40); the session never asks for it again.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, including land-and-sync,
  which this sprint follows (D40): see **Ship** at the end.
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
(recommended; PRD Q5 is confirmed by the owner before launch — see **Before you launch** — and recorded in step 3), as a
`preview` course visible only to the
owner/tester cohort until GA. It needs the **go-race** runner profile ([p-01](../sprints/sprint-p-01.md), `runner-v1.1.0`),
the multi-course catalog/agenda/nav and a multi-file workspace ([p-02](../sprints/sprint-p-02.md)), and the A7 quiz widget
plus the race verdict views ([p-03](../sprints/sprint-p-03.md), `v1.15.0`). The rollout freeze rule says **AB14–AB15 are
frozen before P**, and AB02/AB05 come to full fidelity here. Per **BP3** (owner, 2026-09-24) agents draft every board; per
**D40** (owner, 2026-09-25) launching this prompt is the owner's approval for every change it makes. This session: land the Q5
record, draft four boards' worth of frames, and land the board PR — its merge is the freeze. The owner may review
afterwards; a change to a frozen board is a follow-up design PR.

go-concurrency results are **honor-grade**: the hidden tests run inside the learner's program (`gotest@1`), so results are
labelled honor and excluded from judge-checked %. The runner returns RACE / DEADLOCK / LEAK / WA / TLE / RE / CE, a
goroutine dump on timeouts reduced to learner-package function names and file:line, and no test output in Submit.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **PRD Q5 confirmed before launch** (`ev-q5`, the block above; launching attests it): go-concurrency unless the launch message names SQL. Step 2 applies the fallback rule if P3 contradicts it, **before AB15 is drafted**; not a wait
- [ ] spk-02's **P3 result is recorded** in t3 §16.2 (`git fetch && git show origin/main:docs/v2/research/t3-sandbox.md | grep -n '16.2'`)
- [ ] **AB02 and AB05 frozen** (ds-m1-01 and ds-m2-01 merged): `git ls-tree origin/main design-system/screens/v2/`
- [ ] **AB07★ and AB08 frozen** (ds-m3-01 merged)

If a board you extend is missing from `main`, stop and report. If `index.html` links different file names than the plan,
use the index's names and say so in the PR.

## Do this (in order)

1. **[X] Sync and check peers.** `git checkout main && git pull`; `gh pr list`, `git worktree list`, ListAgents — other
   design or build PRs may be open; you will touch only your files.
2. **[X] Settle PRD Q5 — don't ask and wait** (D40). Take the course the owner confirmed before launch (the launch message's,
   else go-concurrency) and check it against the P3 verdict in t3 §16.2 (the plan's task 1 rule):
   - SQL named in the launch message → **SQL** (cheapest *checked* course, heaviest runner profile): AB15 becomes the SQL
     explorer and p-01 is re-planned to `sql-pg` before it starts;
   - go-concurrency, and go-race works (default policy or a per-process `setarch -R` launcher) → **go-concurrency** (reuses Go
     + the code widget + the quiz, honor-grade);
   - go-concurrency, but go-race failed even with the per-process launcher → the pre-decided **SQL** fallback, as above; note
     in the Q5 PR that redesigning go-concurrency without go-race was set aside, so the owner can revisit it after the merge;
   - a P3 result that fits neither → keep go-concurrency for AB15, record the options and your recommendation in the Q5 PR,
     and mark p-01's Sprint-board row ⛔ "needs owner decision (Q5: P3 inconclusive for go-race)" in `status.md`. Still land
     both PRs.
3. **[X] Record Q5 in its own docs PR.** Branch `docs/ds-p-01-q5`:
   - `docs/prd/xlearn-v2-prd.md` §7: strike Q5 and add the resolved line (wording below);
   - `docs/v2/rollout-plan.md` §13: the PRD Q5 row marked resolved;
   - `docs/v2/status.md`: owner event `ev-q5` ✅ (date; "confirmed before launch; recorded by ds-p-01"), the "Open owner items" PRD Q5 row
     resolved, a Decisions-log line (course, the P3 input, fallback status), and only if SQL: a flag on the p-01 board row
     "re-plan to `sql-pg` before start" (or the ⛔ from step 2's last case);
   - this sprint's plan file: task 1 ✅.
   The PRD §7 line reads "**Resolved (<date>):** <course> — confirmed by the owner before launch of ds-p-01 (D40)", plus "; the
   SQL fallback applied (P3: <verdict>)" when step 2's fallback did. Commit `docs(v2): PRD Q5 resolved — pilot course <slug> (ev-q5)`
   with the attribution lines; push; PR. **Merge it on CI green** (fix, then merge, on failure; never enable auto-merge) —
   launching this prompt approved it (D40) — then `git checkout main && git pull`.
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
8. **[X] Self-review checklist (before merging; nothing waits on the owner).** Check `theme.css` is linked verbatim (then
   `board.css`; tokens only, no new colours, no `ds-*`/`xl-*` override; difficulty tokens). Walk every frame against its
   decision (D10, D15–D18, P7, I8) and the rollout §9 P row. Leak check: no
   key or rationale before a counted conclusion; no hidden test name, source, line or output; no hidden-test frame in a
   dump or race report; no topic chip on a live item; the pilot absent from every non-cohort frame. Decision check (I8):
   no mistake entry and no revision line on a non-revisable drill. Pre-fill appears only on revisable items (AB14 F12,
   AB15 F15). Contrast ≥ 4.5:1. Fix, then re-check; the ticked checklist goes in the PR body.
   Full-page screenshots at **1440** and **390** px into `design-system/screens/v2/shots/` (`AB14@1440.png`,
   `AB15@1440.png`, `AB02-p@1440.png`, `AB05-p@1440.png`, and the `@390` pairs; each ≲ 500 KB), e.g.
   `npx playwright screenshot --full-page --viewport-size=1440,1024 file://$PWD/design-system/screens/v2/AB14-workspace-quiz.html design-system/screens/v2/shots/AB14@1440.png`.
9. **[X] PR body** (for **Ship** step 1) — embedded screenshots, a frame list per board with decision cites, the ticked
    self-review checklist, **"Decisions to confirm"** (at least: D17's arena reveal vs t4 §4.3's "never in the arena" —
    drawn and frozen: D17), and the merged Q5 record PR's link. Each item states the default the merge freezes; the list
    never blocks the merge, and the owner may revisit any item afterwards through a follow-up design PR.
10. **[X] Land it** — run **Ship** below: board PR, merge on CI green (the freeze), status, sync.

## Constraints

- **Preview-only boards.** Static HTML on `theme.css` verbatim plus `board.css`; no new colours or tokens; never imported
  by `web/`, never embedded, never shipped; v1 `.dc.html` files are references only.
- **Touch only your files** in the board PR: `AB14-workspace-quiz.html`, `AB15-go-concurrency-race.html`, the appended
  sections of `AB02-course-nav.html` / `AB05-catalog-agenda.html`, their `shots/`, this sprint's plan file and this sprint's
  own `docs/v2/status.md` rows (Update status). Never `index.html`, `board.css` or `theme.css`.
- **Decisions win.** Cite the decision per frame; where t4's body and an owner decision differ, draw the decision and flag it.
- **Withholding** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule)): keys and
  rationale only after a counted conclusion; hidden tests never named or shown; dump/race frames filtered to learner files.
- **P = widget/profile code only.** Don't design pilot-only service behaviour (a new grade rule, a mock, a new dock state
  outside AB08's set); reuse the AB04/AB07/AB08 mechanics.
- **No `web/`, service, infra or deploy changes; no `kubectl` of any kind; no alerting surface (D34).** Service boundaries
  ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)), goose/sqlc, outbox/inbox, the memory-sum rule,
  consumers-before-producers and ACL-before-consumer don't apply: this session changes no code.
- **Parallel sessions:** other design PRs may be open; rebase on `main` before pushing and resolve nothing outside your files.
  Other sessions also edit `status.md` when they land: rebase before each status commit and keep their rows.

## Deliverables

- A docs PR recording PRD Q5 (PRD §7, rollout §13, `status.md` `ev-q5`), **merged on CI green**.
- `design-system/screens/v2/AB14-workspace-quiz.html`, `AB15-go-concurrency-race.html` (or the SQL explorer), and the
  "P · full fidelity" sections in `AB02-course-nav.html` / `AB05-catalog-agenda.html`, with screenshots in `shots/`.
- The board PR, **merged on CI green** (the freeze), with per-board frame lists, the ticked self-review checklist and
  "Decisions to confirm".

## Update status

- Q5 record PR (step 3): [`../status.md`](../status.md) — `ev-q5` ✅, the "Open owner items" PRD Q5 row resolved, the
  Decisions-log line, the p-01 flag if SQL (or the ⛔ of step 2's last case); this sprint file task 1 ✅.
- Board PR — a last commit once the PR number is known, before the merge — or a follow-up docs PR merged the same way:
  - [`../sprints/sprint-ds-p-01.md`](../sprints/sprint-ds-p-01.md): tasks 2–8 ✅ (task 8: "frozen: merged in PR #N, <date>"),
    _Overall_ ✅;
  - [`../status.md`](../status.md): mark the sprint ✅ (ds-p-01's Sprint-board row) and the boards ✅ **"frozen (merged, PR #N,
    <date>)"** (Artboards rows AB14, AB15; AB02 and AB05 as "full fidelity frozen (merged, PR #N, <date>)"); update the
    Snapshot's artboard count (`ev-freeze-ds-p-01` is automatic: no tick).
- No ADR is expected. If the owner later changes a decision (e.g. the arena reveal for quizzes), the follow-up design PR and
  the build sprint that implements it record the change.

## Done when (acceptance)

- [ ] PRD Q5, as confirmed before launch (or by the fallback rule when P3 contradicts it), is recorded — its docs PR merged on CI green — before AB15 was drafted.
- [ ] Every frame of AB14 (F1–F14), AB15 (F1–F16 or the SQL explorer set), AB02-P1–P7 and AB05-P1–P7 is present with final copy, its state and a cited behaviour-notes aside.
- [ ] No board leaks withheld data, and no non-cohort frame shows the pilot.
- [ ] Boards use `../../theme.css` + `board.css` verbatim and open with no JS runtime; `index.html`, `board.css` and `theme.css` are untouched.
- [ ] The self-review checklist passed and is ticked in the PR body, with "Decisions to confirm" (each item's frozen default stated).
- [ ] Board PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots; the sprint file and `docs/v2/status.md` show ds-p-01 ✅ and AB14, AB15 (and AB02/AB05 full fidelity) "frozen (merged)"; local `main` synced.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. **Branch, commit, push, PR** — this repo only (a design sprint touches no `../infra`). Two PRs: the Q5 record PR
   (`docs/ds-p-01-q5`, step 3) lands first; then, on `design/ds-p-01` (step 4), conventional commit `docs(design): P boards
   AB14 AB15 + AB02/AB05 full fidelity` with the attribution lines; push; `gh pr create` titled `docs(design): AB14 AB15
   (+ AB02/AB05 full fidelity) — P boards` (append "· AB15 = SQL explorer" if Q5 flipped) with the body from step 9.
2. **Merge on green** — once CI is green (fix, then merge, on failure), squash-merge each PR. Never enable auto-merge.
3. **Release action — design: the merge is the freeze; no tag.** Nothing deploys (boards are preview-only; the Q5 record is
   docs-only). The owner may review after the merge; any change to a frozen board is a follow-up design PR.
4. **Update status** — as in "Update status" above (sprint ✅, AB14/AB15 and AB02/AB05 full fidelity "frozen (merged)"), in the
   same PR (a last commit before step 2's merge) or a follow-up docs PR merged the same way.
5. **Sync** — `git checkout main && git pull`. If a clean peer worktree holds `main`, use
   `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
