# Prompt — Sprint ds-m6b-01 · Design M6b: voice pre-flight/notices, voice live HUD ★ (AB29, AB30 ★)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-ds-m6b-01.md`](../sprints/sprint-ds-m6b-01.md)   ·   **Milestone:** M6b (design track)   ·   **Prereqs:** [ds-m6a-01](../sprints/sprint-ds-m6a-01.md), [ds-m6a-02](../sprints/sprint-ds-m6a-02.md), [spk-04](../sprints/sprint-spk-04.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions. **This sprint overrides the
  land-and-sync directive:** it ends at an open PR and never merges (BP3, design sprints).
- The plan: [`../sprints/sprint-ds-m6b-01.md`](../sprints/sprint-ds-m6b-01.md) — the frame tables (AB29 F1–F14 incl. F5a–F5d,
  AB30 F1–F19), file names, copy and the "Decisions to confirm" list are spelled out there. Follow them exactly.
- [`../rollout-plan.md`](../rollout-plan.md) [§9 Artboards by milestone](../rollout-plan.md#9-artboards-by-milestone) (M6b row, freeze
  rule), [§3](../rollout-plan.md#3-milestone-map) (M6b entry: AB29–AB30 frozen), [§4](../rollout-plan.md#4-per-milestone-detail) (M6a/M6b gates).
- Decisions: [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (Accepted with the S6 result) §3 (D29: widgets, no video to the
  AI), §4 (failsafes), §5 (brain-authored debrief, no score), §6 (Chrome/Edge, EU/EEA, Anthropic → text, mandatory cap, 85% wrap,
  audio in memory only, L19 caps); the [feasibility decisions log](../feasibility.md#decisions-log-newest-first) (D27–D31, D34).
- Research detail: [t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options) (browsers),
  [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (states, failure table, grace copy),
  [§5](../research/t6-realtime-interviewer.md#5-the-coding-round) (rail, quiet during Code, hold voice),
  [§6](../research/t6-realtime-interviewer.md#6-assessment) (never-assessed list, distress trigger),
  [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (consent boxes verbatim, EU/EEA, captions,
  modes, screen-reader preset, self-view), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (estimate copy, checks),
  [§13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict)
  (D28–D31 override the body); [PRD](../../prd/xlearn-v2-prd.md) §5.6 R-MI1–R-MI8.
- **The S6 results** ([spk-04](../sprints/sprint-spk-04.md); t6 **§16** "S6 results" and the spike-results row in
  [`../status.md`](../status.md)): the winning shell, browser list (M15), PTT/Patient native or emulated (M3b), captions path, cost (M9),
  voice-check latency.
- [`../../../design-system/README.md`](../../../design-system/README.md) (tokens, components) and
  [`../../../design-system/theme.css`](../../../design-system/theme.css) — reuse verbatim.
- Neighbouring boards on `main` (read-only; match their vocabulary and copy): `design-system/screens/v2/index.html` and `board.css`
  ([ds-m1-01](../sprints/sprint-ds-m1-01.md)), `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html`, `AB26-grace-paused-resume.html`,
  `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html`, `AB07-workspace-code.html`, `AB08-results-dock.html`,
  `AB19-invite-acceptance.html`.
- v1 reference (read-only, **not runnable**): `design-system/screens/Mock.dc.html`. Live-app structure (read-only): `web/src/screens/Mock.tsx`,
  `web/src/components/Coach.tsx`, `web/src/styles/app.css`.

## Context

M6b adds **voice** to the text interviewer M6a shipped dark: the browser talks WebRTC directly to OpenAI on the learner's own key,
coach brokers the SDP server-side (no credential in the browser) and holds a sideband that drops audio copies in memory. Voice runs
in **Chrome/Edge only**, is **not offered to EU/EEA accounts**, needs an **OpenAI** interview key (Anthropic-only learners get text),
and every interview carries a **mandatory $ cap** (wrap-up at 85%). The AI follows the on-screen widgets (D29) and never sees the
learner — self-view is local and off by default. The rollout freeze rule says AB29–AB30 are frozen before M6b, and
[m6b-01](../sprints/sprint-m6b-01.md)'s entry gate requires the freeze. The owner decided (BP3, 2026-09-24) that agents draft every
v2 board — the AB30 hero included — and the owner only reviews. No `web/` code here; boards are preview-only static HTML. AB29/AB30
draw only the **voice deltas**: text setup/consent is AB24, the text HUD AB25, grace/paused/resume AB26, debrief/proposal AB27.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] The S6 results note exists ([spk-04](../sprints/sprint-spk-04.md)): winning shell, browser gate list, M7, M3b, captions path, M9.
      If the S6 decision was "both fail" (voice deferred 3 months), **stop and report** — there is nothing to design yet.
- [ ] [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) reads **Accepted** ([ds-m6a-01](../sprints/sprint-ds-m6a-01.md)).
- [ ] AB24–AB28 are on `main` (frozen by the owner via [ds-m6a-01](../sprints/sprint-ds-m6a-01.md) / [ds-m6a-02](../sprints/sprint-ds-m6a-02.md)).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR adds or edits
      `design-system/screens/v2/AB29-*` or `AB30-*`. If one does, stop and report.
- _Informational:_ if `design-system/screens/v2/index.html` is on `main`, use the file names it links for AB29–AB30 and link `board.css`;
  if it is not, **do not create either file** — use the plan's names and a board-local `<style>`.

## Do this (in order)

1. **[X] Branch** `design/ds-m6b-01` off an up-to-date `origin/main`.
2. **[X] Read the S6 note and the neighbours.** Write down: the winning shell and its per-45/60-minute cost, whether push-to-talk and
   Patient are native or emulated, the captions path, the browser list. From AB24–AB28 note the consent strings, estimate layout,
   rail/clock/editor/hint components, grace and resume modals. Anything you must diverge from goes into "Decisions to confirm".
3. **[X] AB29** — `design-system/screens/v2/AB29-voice-preflight-notices.html`, frames F1–F14 (with F5a–F5d) from the plan with the exact
   copy: mode choice (Voice · Voice push-to-talk · Text; Standard/Patient); the **voice consent** with five unticked boxes (audio through
   xLearn's server **in memory only**) and the retention radio; mic permission (allow / denied / no device); mic check with the
   headphones toggle; the **real ≤ 30 s voice check** and its four failures (no model access — no retry; network; credits — key stays
   enabled; slow answer = `504 sdp_timeout`); the voice **estimate + mandatory $ cap** with the S6 winner's figures, the multiplier row
   and "Voice today: N of 75 minutes left" (at 1×, plus a 1.5× still state: 75 × the multiplier); the **browser**, **EU/EEA**, **no voice key**, **busy / daily limit** notices, each with
   text mode one click away; the screen-reader preset; ready-to-start; the resume-by-voice reaffirmation; narrow.
4. **[X] AB30 ★** — `AB30-voice-live-hud.html`, frames F1–F19: interviewer speaking with the **fixed voice disclosure** (F1's exact text) on
   **AB25's rail labels verbatim** ("Clarify the problem", "Brute force out loud", "Key observation → plan", "Code it", "Trace + edge
   cases", "Complexity + follow-ups"); candidate speaking ("approximate" captions); the Code phase with "the interviewer sees your code as
   you type · updated 2 s ago" and quiet-while-coding; push-to-talk (Ctrl + Space); voice on hold; reconnecting (rollover / network with
   the grace countdown / restored); mic lost (the pause and clock stop at **30 s**); "Still there? The voice connection ends in 60
   seconds." (no minute count) and the idle hang-up; the **credit grace countdown** (5:00, "Checked 20 s ago", Top up with the negative-balance line,
   Check now, Save & resume later, End, the ≥ 75% Finish/End-without-feedback variant; mic muted, camera stopped, editor read-only;
   spend vs cap shown here); the 85% and 100% cap states and the `409 cap_reached` "voice can't reconnect" variant; wrapping/debrief with
   **no score** (and its text variant when entered from hold); the captions log; **self-view** (local only,
   "isn't sent to the AI or recorded"); the distress card (Tele-MANAS 14416); live in another tab; screen-reader preset; **check your transcript** (voice self-edit before AB27's proposal,
   "labelled self-reviewed"); narrow; **F19** the live daily voice limit (3-minute wrap banner, then paused with "resume in text").
5. **[X] Behaviour notes** — every frame's aside names the server field, SSE event or error code that drives it (`voice.available`,
   `voice.reason`, `403 voice_unavailable_region`, `409 consent_required {kinds}`, `409 lease_lost`, `403 voice_model_access`, `409 provider_quota`,
   `504 sdp_timeout`, `409 cap_reached`, `409 voice_key_required`, `401 provider_auth`, `502 provider_unavailable`, `429 voice_capacity`,
   `429 voice_daily_limit`, the interrupt reasons incl. `mic` and `voice_limit`, the event-log kinds `state`, `phase`, `turn`, `caption`,
   `segment`, `cap`, `notice` — incl. `notice{idle_check}`),
   its decision ref, a11y (focus order, contrast, `aria-live` only for phase changes and the grace warning, icon + text indicators,
   reduced motion, shortcuts) and the `< 1024 px` intent. Every board ends with a 390 px section.
6. **[X] Self-review** — walk each frame against its decision (D27, D29, D30, D31, D34, ADR-0032 §3–§6) and the rollout §9 M6b row. Run the
   checks: **no video to the AI** anywhere; **no score, band or delivery metric on the HUD**; no comment on tone, accent or emotion; no
   pattern chip, reference solution, unreleased hint, hidden test or rubric anchor; every "no voice" notice offers text; the consent
   discloses in-memory audio through xLearn; no copy implying anyone was alerted. Fix, then re-check.
7. **[X] Screenshots** — serve statically (e.g. `python3 -m http.server 5198 --directory design-system`, or open from disk) and capture
   each board full-page at **1440 px** and **390 px** (Browser pane `resize_window` + screenshot, or headless Chrome
   `--headless=new --screenshot=… --window-size=1440,900`). Save to `design-system/screens/v2/shots/AB29@1440.png`, `AB29@390.png`,
   `AB30@1440.png`, `AB30@390.png`, each ≲ 500 KB.
8. **[X] Sprint file** — in `docs/v2/sprints/sprint-ds-m6b-01.md` set tasks 1–4 ✅, task 5 🔄 "PR #N open — awaiting owner review",
   _Overall_ 🔄. **Do not edit `docs/v2/status.md`.**
9. **[X] Commit + PR, then STOP** — conventional commit `docs(design): AB29 AB30 — M6b voice boards` ending with the attribution lines;
   push; open the PR titled `docs(design): AB29 AB30 ★ — M6b voice boards` with each board's frame list, the screenshots embedded via
   `https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6b-01/design-system/screens/v2/shots/<file>?raw=true`, and **"Decisions to
   confirm"** (at least: no live spend meter on the HUD per t6 §8 vs the brief's "cost meter vs $ cap"; all five consent boxes required;
   the shortcuts; push-to-talk switchable mid-call; voice on narrow screens; the region wording and where "Confirm your region" leads;
   the transcript self-edit step placed before AB27; the mic-loss 30 s timing vs t6's 10 s; the voice-only idle pre-warning vs AB25 F12;
   the `cap_reached` actions; the daily limit pausing mid-interview; any divergence from AB24–AB28/AB19).
   **Do not merge. Do not enable auto-merge.** Report the PR link and stop. Requested changes go on the same branch.

## Constraints

- **Preview-only:** nothing under `design-system/screens/v2/` is imported by `web/`, embedded or deployed. No `web/`, `internal/`,
  `cmd/`, `deploy/`, `curriculum/` or `../infra` change in this sprint.
- **`theme.css` verbatim:** link `../../theme.css`; never copy, fork or override `ds-*`/`xl-*` rules. Dark theme only. Difficulty tokens
  Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. Board layout CSS uses `--ds-*` tokens only.
- **Static HTML:** no JS runtime, no `support.js`, no `.dc.html` canvas markup. Fonts via the Google Fonts `<link>` only.
- **Touch only this sprint's files:** the two boards and their four screenshots. Never edit `index.html` or `board.css` (written once by
  ds-m1-01) or another sprint's board.
- **Decisions on screen:** D29 (no video to the AI; self-view local and off by default), ADR-0032 §5 (no score live; debrief content-only
  and brain-authored), §6 (Chrome/Edge, EU/EEA off, Anthropic → text, cap mandatory), D34 (no alert promises), D31 (nothing public).
- **One shell:** draw the S6 winner; the other shell is at most one aside line.
- **Final copy:** real strings with the real error codes; no placeholder text.
- **No `status.md` edit** from a design PR; [m6b-01](../sprints/sprint-m6b-01.md) records the freeze.
- **Parallel sessions:** check peers' open PRs and worktrees before creating the board files. The memory-sum rule, GitOps and the
  release checklist are not in play: nothing here runs in the cluster.

## Deliverables

- `design-system/screens/v2/AB29-voice-preflight-notices.html`, `AB30-voice-live-hud.html`.
- `design-system/screens/v2/shots/AB29@{1440,390}.png`, `AB30@{1440,390}.png`.
- An open PR (not merged) with frame lists, screenshots and "Decisions to confirm".

## Update status

- `docs/v2/sprints/sprint-ds-m6b-01.md` Status table in the PR: tasks 1–4 ✅, task 5 🔄 (PR #), _Overall_ 🔄.
- **Not** `docs/v2/status.md`: [m6b-01](../sprints/sprint-m6b-01.md) flips the AB29–AB30 Artboards rows to "frozen (PR #, date)", sets this
  sprint ✅ on the Sprint board and ticks owner event `ev-freeze-ds-m6b-01` once the owner has merged.
- No ADR expected. If you resolve a real ambiguity (e.g. the live spend meter), list it under "Decisions to confirm" in the PR for the
  owner rather than writing an ADR.

## Done when (acceptance)

- [ ] Every frame listed for AB29 (F1–F14, incl. F5a–F5d) and AB30 (F1–F19) is present with final copy, its state, its decision refs and
      a behaviour-notes aside.
- [ ] No frame shows video going to the AI, a score or delivery metric on the HUD, or withheld item data; every "no voice" notice keeps
      text one click away; the consent frame discloses in-memory audio through xLearn's server.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied; only the two board files and their screenshots are added.
- [ ] PR open with 1440 px and 390 px screenshots of each board and "Decisions to confirm"; **not merged by the agent**.

Shipping: **design sprint — open the PR and STOP for owner review.** This overrides AGENT.md's end-of-session land-and-sync: do not
merge, do not enable auto-merge, do not tag. The owner's merge (or explicit approval in chat) is the freeze.
