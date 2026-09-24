# Prompt — Sprint ds-m6a-01 · Design M6a part 1 + ADR-0032 → Accepted: Mock-v2, setup/consent/pre-flight, live HUD text (AB13, AB24, AB25)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m6a-01.md`](../sprints/sprint-ds-m6a-01.md)   ·   **Milestone:** M6a (design track)   ·   **Prereqs:** [spk-04](../sprints/sprint-spk-04.md), [ds-m1-01](../sprints/sprint-ds-m1-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions. Land-and-sync applies to **task 1's ADR PR only**; the board PR ends open and is never merged by you (BP3, design sprints).
- The plan: [`../sprints/sprint-ds-m6a-01.md`](../sprints/sprint-ds-m6a-01.md) — task 1's ADR edits and stop rule, the frame tables (AB13 F1–F12, AB24 F1–F10, AB25 F1–F15), file names, copy and "Decisions to confirm". Follow them exactly.
- **The S6 result:** [t6 §16](../research/t6-realtime-interviewer.md) (added by spk-04) and `docs/v2/research/t6-s6-fixtures/index.json`.
- Decisions: [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (all of it; you accept it), [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) (amended by 0032 §7), [ADR-0024](../../adr/0024-public-user-dashboards-and-usernames.md) (D31 note), [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (names), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L16, L19), [ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule), the [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D14, D17, D27, D28–D31, D34).
- Research detail: [t6](../research/t6-realtime-interviewer.md) §3 (routes, limits, onboarding advice), §4 (clock, classifier, idle, lease, coach lock), §5 (coding round, hints, rail), §6 (assessment; never-assessed list; distress), §7 (consent strings, retention, S1–S4, a11y), §8 (estimate copy, pre-flight), §11 (amendments), §13 (D28–D31); [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (A9 Mock-v2), [§3.7](../research/t4-judge-contract.md#37-touches-mocks-arena), [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock); [PRD](../../prd/xlearn-v2-prd.md) §5.6 R-MI1–R-MI8.
- [Rollout §9](../rollout-plan.md#9-artboards-by-milestone) (the M6a row and the freeze rule), [§3](../rollout-plan.md#3-milestone-map) (M6a entry), [§4 M6a/M6b](../rollout-plan.md#m6a--m6b-v21--57--45-sprints).
- [`../../../design-system/README.md`](../../../design-system/README.md) and [`../../../design-system/theme.css`](../../../design-system/theme.css) — reuse verbatim; `design-system/screens/v2/index.html` and `board.css` (from [ds-m1-01](../sprints/sprint-ds-m1-01.md)) for names and chrome.
- Neighbouring boards on `main` (read-only; match their vocabulary): `AB01-coach-states.html` (F5, F13), `AB06-public-profile-v2.html`, `AB07-workspace-code.html`, `AB08-results-dock.html`, `AB11-degradation-badges.html`, `AB19-invite-acceptance.html`, `AB20-privacy-notice.html`.
- v1 references (read-only, **not runnable**): `design-system/screens/Mock.dc.html`; live structure: `web/src/screens/Mock.tsx` (`PROBLEM_SETS`, `RAIL_REFERENCE`), `web/src/lib/mock.ts` (`RUBRIC_DIMENSIONS`).
- Consumers: [m6a-01](../sprints/sprint-m6a-01.md) … [m6a-05](../sprints/sprint-m6a-05.md), [mi-13](../sprints/sprint-mi-13.md) (gates on ADR-0032 Accepted), [ds-m6a-02](../sprints/sprint-ds-m6a-02.md), [ds-m6b-01](../sprints/sprint-ds-m6b-01.md).

## Context

M6a is the **text** mock interviewer with the owner's failsafes, on the learner's BYO `interview` key, living in coach
(`internal/coach/interview`). The owner decided (BP2) that ADR-0032 is accepted when S6 reports, before the M6a design freeze, and
that the acceptance is a task of the sprint that consumes S6's result — this one. So the session has two outputs: **first** a
small docs PR accepting ADR-0032 with the S6 result (merged, per land-and-sync), **then** the first three M6a boards on a design
branch, stopped for owner review. AB13 is the course's Mock page in v2 (format choice; classic mock with server-picked items;
evidence | sliders scoring, "Scores lock when saved"); AB24 is everything between "AI interviewer · text" and the first message
(opt-in, setup, per-session consent, pre-flight, the mandatory $ cap); AB25 is the live text HUD (rail, server clock, transcript,
editor and Run, `give_hint`, the snapshot indicator, Hold, Finish). Voice (M6b) appears only as labelled "from M6b" variants.
The rollout freeze rule: AB13 and AB24–AB28 are frozen before M6a; [m6a-01](../sprints/sprint-m6a-01.md) gates on it.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **S6 reported:** t6 §16 is on `main` with a decision inside the D28 rules as spk-04 task 6 applies them (a shell or "text only, revisited on <date>"; the M7 deploy shape; the browser list) and the fixtures exist. An escalated result is **not** a pass: stop.
- [ ] [ds-m1-01](../sprints/sprint-ds-m1-01.md) merged: `design-system/screens/v2/index.html` and `board.css` on `main`.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR edits ADR-0032/0007/0024 or adds `design-system/screens/v2/AB13-*`, `AB24-*`, `AB25-*`. ADR numbers aren't claimed here (0032 exists).

## Do this (in order)

1. **[X] Accept ADR-0032 (own docs PR).** Branch `docs/adr-0032-accept` off an up-to-date `origin/main`. Apply the plan's task 1:
   ADR-0032 Status → **Accepted (<date>)** with the S6 result folded into §2 (shell + measured reason; deploy shape from M7), §4
   (fixtures = spend-limit path; the probe arbitrates prepaid), §6 (browser list; measured cost; D29 cadence cost), Links (t6 §16,
   fixtures) — and, if both shells failed, P1 "deferred: voice revisited by <date>". ADR-0007: the ADR-0032 amendment → Accepted
   plus a short *Amendment (ADR-0032)* section (SDP brokering; `client_secrets` fallback only if S6 needed it; the key per live
   segment). ADR-0024: its ADR-0032 note → Accepted. `docs/adr/README.md` and the feasibility T6 rows → Accepted. `docs/v2/status.md`:
   ADR row, Decisions log, this sprint's board row 🔄; this sprint file: task 1 ✅, _Overall_ 🔄. Commit `docs(adr): accept ADR-0032
   with the S6 result` (attribution lines), push, PR, **merge once CI is green**, `git checkout main && git pull`.
   **Stop rule:** if the S6 note is escalated or outside the D28 rules, don't accept — leave it Proposed, log why, stop and report.
2. **[X] Branch** `design/ds-m6a-01` off the updated `main`.
3. **[X] Read the neighbours** on `main` (AB01, AB06, AB07, AB08, AB11, AB19, AB20); note component choices and strings you must
   reuse: locked-coach banner, keys panel names, editor chrome, Run strip, badges, consent tone, and AB20's host-image and
   pre-release-snapshot wording. xLearn keeps **no backups** (D12), so never write "backups". Don't copy AB01 F13's "(needs a
   realtime-capable model)": t6 §11 drops that rule. List it under "Decisions to confirm".
4. **[X] AB13** — `design-system/screens/v2/AB13-mock-v2.html`, frames F1–F12 with the plan's copy: mock home with the format
   cards (voice only as "from M6b"), the "only you see your scores" line; classic setup that **picks the problem for you** and
   skips open/due items; classic live (interview-mode editor, Run ≤ 1/10 s, coach locked, no Submit); multi-item tabs
   (illustrative); finishing; **evidence | sliders** with `evidence_hints` as a suggestion and "Scores lock when saved."; save
   confirm; saved/locked with `self · honor`; history with every status; limits and empty states; an N-dimension rubric; narrow.
5. **[X] AB24** — `AB24-mock-setup-preflight.html`, frames F1–F10: the account opt-in with the "what it evaluates" card; setup
   (format, length, multiplier with cost delta, difficulty, "Your AI coach (your key)" row, persona); the three unticked consent
   boxes + retention choice + the deletion-lag line in **AB20's wording** (host image up to about 7 days, pre-release snapshot up
   to 1 day; never "backups"); pre-flight checks; each classifier failure's copy (`ErrModelAccess`, `ErrAuth`,
   `ErrQuota`, custom model, no key); the estimate with "+ tax (India: 18% GST)", extras, the **mandatory cap** (85% wrap, 100%
   end), the negative-balance line and the key-hygiene tip; the L19 caps with *proposed* codes; starting; the **Anthropic variant
   listing what differs**: Text only; provider-named `ErrQuota` copy with Anthropic's `billing_url`; "billed by Anthropic"; no
   negative-balance line; the Anthropic tip. Then narrow.
6. **[X] AB25** — `AB25-live-hud-text.html`, frames F1–F15: rail + public phase goal + server clock + multiplier chip; the AI
   disclosure as the first message; the snapshot indicator ("Shared with the interviewer · 2 s ago", 64 KB meter); Run strip and
   cooldown; `give_hint` (recorded, ≤ 1 conceptual per phase); phase nudge; streaming and transient/rate errors; Hold; quiet during
   Code; the distress card (Tele-MANAS 14416); 85%/100% cap; finish confirm; the idle check-in (`interrupted(idle)`: "Still
   there? We stopped your clock while you were away. [I'm back]", free re-prime, no countdown); second tab; the AB01 F5 coach
   lock; narrow.
7. **[X] Behaviour notes** — every frame's aside names its driver (`deadline_at`, `active_ms`/`live_since`, `PUT …/code`,
   `POST …/run|hint|hold|finish|abandon|interrupt`, bounded SSE + `Last-Event-ID`, `interview_consent`, the probe, the error codes),
   its decision ref, a11y (`role="log"`, polite vs assertive, focus order, disabled-reason text, reduced motion) and the `< 1024 px`
   intent; codes this brief invents are marked *proposed*. Each board ends with a 390 px section.
8. **[X] Self-review** — the plan's task 6: D29, D27, D31, T6's no-auto-accept, L19, the never-assessed list, D34; the leak check
   (no pattern chip on the live item, aggregate evidence only, no hint text before release, public descriptors only, no score
   live or public); the neighbour copy check. Fix, then re-check.
9. **[X] Screenshots**: serve statically with `python3 -m http.server 5198 --directory design-system`, then capture each
   board **full-page** at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB13@1440.png` … `AB25@390.png`, each
   ≲ 500 KB. A viewport-sized capture is not enough, and the Browser pane can't write PNGs to disk. Use:
   `npx -y playwright screenshot --channel chrome --full-page --wait-for-timeout 1000 --viewport-size "1440, 900" http://localhost:5198/screens/v2/AB13-mock-v2.html design-system/screens/v2/shots/AB13@1440.png`
   (and `--viewport-size "390, 844"` for `@390`). **Fallback:** read the board's `document.documentElement.scrollHeight` at that
   width (Browser pane `resize_window` + `javascript_tool`), then run
   `"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --hide-scrollbars --window-size=1440,<scrollHeight> --screenshot=<file> <url>`.
   Open each PNG to check it holds the whole board.
10. **[X] Sprint file** — in the design branch set tasks 2–6 ✅, task 7 🔄 "PR #N open — awaiting owner review". **Don't edit `docs/v2/status.md`** in this PR.
11. **[X] Commit + PR, then STOP** — commit `docs(design): AB13 AB24 AB25 — M6a boards, part 1` (attribution lines); push; open
    `docs(design): AB13 AB24 AB25 — M6a boards (part 1)` with the frame lists, embedded screenshots
    (`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6a-01/design-system/screens/v2/shots/<file>?raw=true`), the link to the
    merged ADR PR and **"Decisions to confirm"** (the plan's list). **Don't merge. Don't enable auto-merge.** Report the PR link and stop.

## Constraints

- **ADR PR:** docs only (ADRs, ADR index, feasibility rows, `status.md`, this sprint file); no board files in it. BP2 is the owner's pre-authorisation to merge it; the stop rule overrides.
- **Preview-only boards:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`, `internal/`, `cmd/`, `deploy/`, `curriculum/` or `../infra` change.
- **`theme.css` verbatim:** link `../../theme.css` then `board.css`; never copy, fork or override `ds-*`/`xl-*`. Dark theme only; Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; board CSS uses `--ds-*` tokens only.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>`.
- **Touch only this sprint's files:** three boards + six screenshots in the board PR. Never `index.html`, `board.css` or another sprint's board.
- **No leaks, no scores:** no pattern chip on the live item, no hidden inputs, no hint text before `give_hint`, no reference solution, no private anchors; no score live, none public (D31).
- **Perception:** copy says the AI follows "your messages and your editor", never "sees you" (D29; the never-assessed list).
- **Money:** BYO interview estimates in dollars (the learner's own key, t6 §8) with "+ tax"; never the platform allowance (AB18's percent rule is untouched).
- **D34:** no copy implying anyone is alerted. GitOps, the memory-sum rule, consumers-before-producers, ACL-before-tag, goose/sqlc and outbox/inbox are n/a (nothing runs).
- **Parallel sessions:** re-check peers before pushing each PR; rebase the ADR branch onto a moved `main`.

## Deliverables

- A merged ADR PR: ADR-0032 **Accepted** with the S6 result; ADR-0007 and ADR-0024 updated; ADR index, feasibility rows and `status.md` updated.
- `design-system/screens/v2/AB13-mock-v2.html`, `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html`.
- `design-system/screens/v2/shots/AB{13,24,25}@{1440,390}.png`.
- An open board PR (not merged) with frame lists, screenshots and "Decisions to confirm".

## Update status

- ADR PR: `docs/v2/status.md` — ADR-0032 row Accepted (date, PR), a Decisions log line (shell, deploy shape, browsers), this sprint's Sprint-board row 🔄; this sprint file task 1 ✅, _Overall_ 🔄.
- Board PR: this sprint file only — tasks 2–6 ✅, task 7 🔄 (PR #). **Not** `status.md`: [m6a-01](../sprints/sprint-m6a-01.md) writes "frozen (PR #, date)" for AB13, AB24, AB25, sets this sprint ✅ and ticks `ev-freeze-ds-m6a-01` once the owner has merged.
- The ADR acceptance is the ADR record for this sprint; no new ADR. Real ambiguities in the boards go under "Decisions to confirm".

## Done when (acceptance)

- [ ] ADR-0032 is **Accepted** on `main` with the S6 result folded in, ADR-0007 and ADR-0024 updated, via its own merged docs PR — or the stop rule was applied and recorded.
- [ ] Every frame listed for AB13 (F1–F12), AB24 (F1–F10) and AB25 (F1–F15) is present with final copy, its state, its decision refs and a behaviour-notes aside.
- [ ] No board leaks withheld data, shows a score live or publicly, or implies the AI perceives the learner; interview costs appear in dollars only as BYO estimates.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied; only the three board files and their screenshots are added.
- [ ] Board PR open with 1440 px and 390 px screenshots and "Decisions to confirm"; **not merged by the agent**.

Shipping: **task 1's ADR PR follows AGENT.md land-and-sync** (merge once green, then sync `main`). **The board PR is a design PR
— open it and STOP for owner review.** For it, this overrides land-and-sync: don't merge, don't enable auto-merge, don't tag. The
owner's merge (or explicit approval in chat) is the freeze.
