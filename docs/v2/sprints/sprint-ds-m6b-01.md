# Sprint ds-m6b-01 — Design M6b: voice pre-flight/notices, voice live HUD ★ (AB29, AB30 ★)

> **Milestone:** M6b — voice, one shell (the boards M6b builds against) · **Track:** design (parallel; order 80)
> **Prereqs (gates):** [ds-m6a-01](sprint-ds-m6a-01.md) (AB24 setup + consent + pre-flight + $ cap and AB25 live HUD text frozen; ADR-0032 → Accepted) · [ds-m6a-02](sprint-ds-m6a-02.md) (AB26–AB28 frozen: the grace/paused/resume and accessibility vocabulary AB29/AB30 extend) · [spk-04](sprint-spk-04.md) (S6: the winning shell, the browser gate list, M7) · references, not gates: [ds-m3-01](sprint-ds-m3-01.md) (AB07 editor + Run, AB08 results dock), [ds-l-01](sprint-ds-l-01.md) (AB19 region), [ds-m1-01](sprint-ds-m1-01.md) (board index, `board.css`)
> **Unblocks:** [m6b-01](sprint-m6b-01.md) (M6b entry gate "AB29–AB30 frozen", [rollout §3](../rollout-plan.md#3-milestone-map)) · consumed by [m6b-03](sprint-m6b-03.md) (builds every frame here); error codes and copy reused by [m6b-01](sprint-m6b-01.md) (broker preconditions) and [m6b-02](sprint-m6b-02.md) (caps, cap wrap, idle, grace)
> **Release action:** **land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag, nothing deploys (launching the prompt is the owner's approval, [D40](../feasibility.md#decisions-log-newest-first); the owner may review after the merge, and any change to a frozen board is a follow-up design PR)
> **Calendar:** Q1 2027, after the S6 results and the M6a freeze · no owner event: the freeze happens at this sprint's merge (`ev-freeze-ds-m6b-01` is automatic and needs no tick; the Artboards rows record the freeze), before [m6b-01](sprint-m6b-01.md) starts
> **Execute with:** [`../prompts/prompt-ds-m6b-01.md`](../prompts/prompt-ds-m6b-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB29 voice pre-flight + browser/EU notices + voice consent | X | ⬜ |
| 3 | AB30 ★ voice live HUD (hero) | X | ⬜ |
| 4 | Self-review checklist (run before merging) + screenshots (1440 px, 390 px) | X | ⬜ |
| 5 | Open the design PR (screenshots, frame lists, the ticked self-review checklist, "Decisions to confirm") | X | ⬜ |
| 6 | Freeze: squash-merge on CI green (the merge is the freeze) → status → sync `main` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Land and sync ([D40](../feasibility.md#decisions-log-newest-first)):** the board PR
> merges on CI green and the merge is the freeze, so this sprint records its own close-out. Once the PR number is known, a last
> commit on the PR branch sets tasks 1–6 and _Overall_ ✅ here (task 6: "frozen: merged in PR #N, <date>") and, in
> [`../status.md`](../status.md), this sprint's Sprint-board row ✅, the Artboards rows AB29 and AB30 → ✅ "frozen (merged, PR #N,
> <date>)", and the Snapshot's artboard count (`ev-freeze-ds-m6b-01` is automatic: no tick). If the merge
> slips to another day or fails after that commit, correct the rows in a follow-up docs PR merged the same way.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **S6 results note available** ([spk-04](sprint-spk-04.md)): the **chosen shell** (GPT-Live-1 or `gpt-realtime-2.1-mini`), the
      **browser gate list** (Chrome/Edge; Firefox and Safari legs, M15), the **M7 deploy shape**, and the facts the frames depend on:
      push-to-talk and Patient native or emulated (M3b), captions path (GPT-Live `oai-events` data channel vs the sideband over SSE),
      measured cost per 45/60 minutes (M9) and the pre-flight voice-check latency
- [ ] **ADR-0032 Accepted** with the S6 result folded in ([ds-m6a-01](sprint-ds-m6a-01.md) task 1)
- [ ] **AB24–AB28 frozen on `main`** ([ds-m6a-01](sprint-ds-m6a-01.md), [ds-m6a-02](sprint-ds-m6a-02.md) merged), so AB29 extends
      AB24's setup/consent/pre-flight and AB30 extends AB25's HUD and AB26's grace/paused/resume rather than redrawing them
- [ ] **Parallel sessions:** no open peer PR adds or edits `design-system/screens/v2/AB29-*` or `AB30-*`
      (`gh pr list --state open`, `git worktree list`, ListAgents)

_Informational, not gates:_ `design-system/screens/v2/index.html` and `board.css` are written once by
[ds-m1-01](sprint-ds-m1-01.md) and already link `AB29-voice-preflight-notices.html` and `AB30-voice-live-hud.html`. If they are on
`main`, link `board.css` and use exactly those names; if they are not, **do not create them** — use the names in task 1 and a
board-local `<style>` for chrome.

## Goal

Draft the two M6b boards — **AB29** (voice pre-flight: mode choice, voice consent, mic check, the real voice check, the voice
estimate and $ cap, and every "no voice here" notice) and the **AB30 ★ voice live HUD** hero — as static, preview-only HTML on
[`theme.css`](../../../design-system/theme.css) under `design-system/screens/v2/`, with every frame, every state and **final copy**,
so [m6b-03](sprint-m6b-03.md) builds against frozen boards and [m6b-01](sprint-m6b-01.md) can open M6b. Per **BP3** (owner decision
2026-09-24) the agent drafts every board, the AB30 hero included; per [D40](../feasibility.md#decisions-log-newest-first) the board
PR merges on CI green and the merge is the freeze — the owner may review afterwards, and any change to a frozen board is a
follow-up design PR. The rollout's "owner designs the heroes in Claude Design" ([rollout §9](../rollout-plan.md#9-artboards-by-milestone))
is superseded by BP3.

The boards encode decisions that are easy to get subtly wrong on screen: **D29** (the AI "shares the session like another
interviewer": it follows the on-screen answer widgets as structured state, **no video goes to the AI**, the camera is an optional
local self-view, off by default), **ADR-0032 §6** (voice on **Chrome/Edge only**; **no voice for EU/EEA** accounts until a legal
review; Anthropic-only keys get text; a **mandatory, editable $ cap** with wrap-up at 85%; audio **never stored** and passing
through xLearn's server **in memory only** — disclosed), **ADR-0032 §3/§5** (what is said and coded is assessed — never tone,
accent, voice, face or appearance; evaluative speech is brain-authored and the interviewer **never states a score**),
**ADR-0032 §4** (grace, pause, resume — voice adds the countdown and the hang-up) and **D34** (no copy may imply anyone was alerted).

## Scope

**In**
- `AB29-voice-preflight-notices.html` and `AB30-voice-live-hud.html` under `design-system/screens/v2/`, one file per board, every
  frame in tasks 2–3.
- A behaviour-notes aside per frame citing its decision (D-number / ADR § / t6 §), with the exact server field or error code,
  timers (× the time multiplier where t6 says they scale), gates and a11y notes; a `< 1024 px` intent per frame and a 390 px
  section per board.
- PNG screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/` and in the PR body.
- The design PR, merged on CI green (the freeze), and this sprint's rows in `docs/v2/status.md` (Status note).

**Out**
- Any `web/` code → [m6b-03](sprint-m6b-03.md) (both boards). The states behind the frames → [m6b-01](sprint-m6b-01.md) (SDP broker,
  preconditions and their typed errors, sideband captions, second-tab supersede, current-screen updates) and
  [m6b-02](sprint-m6b-02.md) (lease/re-attach, drain, cost and cap, voice caps, idle hang-up, modes, hold, rollover).
- Editing `design-system/screens/v2/index.html` or `board.css` → owned by [ds-m1-01](sprint-ds-m1-01.md) (written once).
- The text setup, account-level interviewer opt-in, text consent and text pre-flight → **AB24**; the text HUD → **AB25**; the
  simple grace modal, paused banner, resume modal and `incomplete` → **AB26**; debrief + proposal (incl. F7's self-scored voice
  Communication variant) → **AB27**; accessibility settings (captions, multiplier, reduced motion) →
  **AB28** — all [ds-m6a-01](sprint-ds-m6a-01.md) / [ds-m6a-02](sprint-ds-m6a-02.md). AB29/AB30 draw only the **voice deltas** and
  reference those boards by frame id.
- M6c extras (photo "show your work", read-aloud, local recording download, Safari if it failed S6, voice-lite) →
  [m6c-01](sprint-m6c-01.md), [m6c-03](sprint-m6c-03.md) (outline).
- Shipping boards (preview-only, never imported by `web/`) and owner design hours (BP3; any owner review happens after the merge).

## Tasks

### 1 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named per the ds-m1-01 index:
`AB29-voice-preflight-notices.html`, `AB30-voice-live-hud.html` (if `index.html` on `main` links different names, use those).
- Each file links `../../theme.css` (then `board.css` if it is on `main`) and reuses its tokens/components **verbatim** — dark
  "landscape console"; see [`design-system/README.md`](../../../design-system/README.md): `ds-card`, `ds-badge--ok|warn|err|info|violet`,
  `ds-chip`, `ds-btn--primary|secondary|ghost`, `ds-seg`/`ds-seg__btn--on`, `ds-toggle`, `ds-meter`/`ds-meter__fill`,
  `ds-modal--sm|md`, `ds-field`/`ds-input`, `ds-spin`, `ds-dot--ok|progressing|failed`, `xl-panel`, `xl-code`, `xl-lock`, `xl-timer`,
  `xl-kbd`, `xl-sect`, `xl-mut`, `xl-eyebrow`. Difficulty tokens: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`.
  Inter + JetBrains Mono via the Google Fonts `<link>`; mono for clocks, countdowns, amounts, minutes and shortcuts. Board-only
  layout CSS lives in the file's own `<style>` and uses `--ds-*` tokens only; it never overrides a `ds-*`/`xl-*` rule.
- **Frames:** full-screen frames stack; card- and modal-sized states sit side by side in a grid. Each frame carries a label
  `ABnn-Fk · <state>`, its decision refs (e.g. `ADR-0032 §6 · t6 §7`) and **final copy** (no lorem ipsum, no "TBD").
- **Behaviour-notes aside** per frame: the server field or SSE event that drives it, timers, gates, exact error codes, a11y (focus
  order, contrast ≥ 4.5:1 on text, keyboard and shortcuts, `aria-live` limited to phase changes and the grace warning,
  reduced motion; indicators are icon + text, never colour alone) and the `< 1024 px` layout.
- Static HTML only: no JS runtime, no `support.js`, no `.dc.html` canvas markup; boards open from disk or any static server.
  Audio levels, captions and countdowns are drawn as still states.
- **Shell-specific copy.** Use the **S6 winner's** figures and behaviour (cost from the catalog row / S6 M9, native vs emulated
  push-to-talk, captions path). Where the other shell would differ, one line in the aside is enough — P1 ships exactly one shell.
- **Never leak withheld data** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) S1/S4): no pattern chip, reference
  solution, hint ladder, hidden test inputs or expected outputs, rubric anchors or exemplars anywhere on the HUD; only released
  hints (`give_hint`) and the learner-visible Run DTO (passed/total, failure class).
- **Touch only this sprint's own board files** (and their screenshots, plus this sprint's own rows in `docs/v2/status.md`).
  `index.html` already has a row and link for AB29–AB30.

### 2 · AB29 voice pre-flight + notices [X]

`AB29-voice-preflight-notices.html` — everything between "I want a voice interview" and the first live second, plus every
reason voice isn't offered. Consumed by [m6b-03](sprint-m6b-03.md) (task 1) on top of [m6b-01](sprint-m6b-01.md) (preconditions,
`voice{available, reason}` in the pre-flight response) and [m6b-02](sprint-m6b-02.md) (estimate, caps). Sources:
[t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (consent boxes, EU/EEA, accessibility),
[t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (estimate copy, checks),
[t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options) (browsers), [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)
(`ErrModelAccess`), [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits),
[ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md) (region is an attestation), [PRD](../../prd/xlearn-v2-prd.md) §5.6 R-MI1, R-MI2, R-MI8.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Mode choice** (the AB24 setup row, voice added; entered from AB13 F1b's "AI interviewer · voice" card) | `ds-seg` **[Voice] [Voice · push-to-talk] [Text]**; under Voice a second `ds-seg` **[Standard] [Patient]** with helper "Patient waits longer before replying — useful if you think out loud."; one line "Voice runs in Chrome or Edge with your OpenAI key. Text works with any key and is scored the same way." No badge or penalty on Text | R-MI1; t6 §7 modes; ADR-0032 §6 |
| F2 | **Voice consent** (per session, all boxes unticked; extends AB24 F3) | Card "Before a voice interview": AB24 F3's three boxes **verbatim** (**☐** "My transcript and code are sent to my AI provider's text API for the interviewer, summaries and feedback." · **☐** "The interviewer is an AI. It can't see you." · **☐** "Assessment uses what I say and my code — never my voice, face, accent or appearance.") plus two voice boxes: **☐** "Stream my microphone to OpenAI using my key. My voice also passes through xLearn's server **in memory only**; it is never written to disk or logs." · **☐** "I'll take this somewhere private: nearby voices are sent and transcribed too."; AB24 F3's retention radio and backup footer unchanged; one added line "In voice interviews you score Communication yourself — the AI gives you quotes." (AB27 F7); **[Continue]** disabled until every box is ticked; "consent v1" | t6 §7 consent; ADR-0032 §6; R-MI8; [m6b-01](sprint-m6b-01.md) voice consent kinds |
| F3 | Microphone permission | Pre-prompt "xLearn needs your microphone for a voice interview. Chrome will ask you next." **[Allow microphone]**; **denied**: "Microphone blocked. Click the icon at the left of the address bar → Site settings → Microphone → Allow, then try again." **[Try again]** **[Use text mode]**; **no device**: "No microphone found. Plug one in, or use text mode." The camera is never requested here | t6 §7 (camera only on self-view) |
| F4 | Mic check | Input `ds-field` select ("MacBook Pro Microphone"); a still `ds-meter` level "Say something — the bar should move."; toggle **"I'm wearing headphones"** + helper "Without headphones the interviewer can hear itself through your speakers. Headphones make interruptions work better." | t6 §10 M3 (no-headphones gate); t6 §7 |
| F5 | **Real voice check** (≈ $0.02) | Header "Quick voice check · about 20 seconds · uses your key (≈ $0.02)"; states side by side: **connecting** "Connecting to OpenAI…" (`ds-spin`) → **greeting** caption "Hi, I'm an AI interviewer. Could you say one sentence so I can check I can hear you?" → **listening** → **passed** "Voice check passed · replied in 1.4 s · captions working" (`ds-badge--ok`) | t6 §8 checks; ADR-0032 §6 (AI disclosure) |
| F5a | Voice check: no model access | "Voice needs a paid OpenAI account (Tier 1 or higher). This key can't use the voice model." **[Use text mode]** **[Open Settings]** — no retry button | t6 §4 `ErrModelAccess` (never retry, never disable) |
| F5b | Voice check: network | "Couldn't connect the audio. Your network may block the connection voice needs." **[Try again]** **[Use text mode]** | t6 §14 (UDP-blocked networks) |
| F5c | Voice check: credits | "Your OpenAI credits ran out." **[Top up ↗]** **[Use text mode]** — the key stays enabled | t6 §4 `ErrQuota` (never disable) |
| F5d | Voice check: slow answer | "OpenAI took too long to answer. Nothing was started." **[Try again]** **[Use text mode]** | [mi-13](sprint-mi-13.md) `504 sdp_timeout` |
| F6 | **Estimate + $ cap** (voice variant of AB24's) | (GPT-Live figures shown; use the S6 winner's) "Typical **$3.1** · plan for **$3.5** · + tax where applicable (India: 18% GST) · billed by OpenAI to your key. OpenAI deducts any negative balance first: keep at least **$4.5** available, or turn on auto-recharge."; multiplier row "Time multiplier **1×**"; `ds-field` **"Stop the interview at $ [5.25]"** (default plan-for × 1.5) + helper "At 85% the interviewer wraps up. At 100% the interview ends. Required."; tip "Holding voice while you code lowers the cost."; line **"Voice today: 62 of 75 minutes left"**; a second still state at **1.5×**: "Time multiplier **1.5×** → typical **$4.3**" and "Voice today: 62 of **112** minutes left" (the daily allowance is 75 × the multiplier) | t6 §8; ADR-0032 §6 (mandatory cap, 85% wrap); L19 ([m6b-02](sprint-m6b-02.md): 75 min × multiplier) |
| F7 | **Browser not supported** | "Voice interviews run in Chrome or Edge for now." / "In Firefox the voice connection drops after a turn or two, so we don't offer it yet. Text interviews work in any browser and are scored the same way." **[Continue in text]** **[Copy link]** ("Open this page in Chrome or Edge"). Safari variant only if Safari failed its S6 leg | t6 §2 browsers; S6 M15 |
| F8 | **EU/EEA notice** | "Voice interviews aren't available for accounts in the EU or EEA yet." / "We're waiting for a legal review of live voice AI there. Text interviews are available and scored the same way." / muted line "Region on your account: Germany" **[Continue in text]**; **no-region** variant (fails closed): "Voice needs the region on your account, and it isn't set." **[Confirm your region]** **[Continue in text]** | ADR-0032 §6; t6 §7 EU/EEA; ADR-0033 §6 |
| F9 | **No voice key** | "Voice needs an OpenAI key as your interview key." / "Your interview key is Anthropic, which has no voice. Add an OpenAI key in Settings, or continue in text." **[Open Settings]** **[Continue in text]** | t6 §1 (Anthropic-only → text); [m1-10](sprint-m1-10.md) `key_default(interview)` |
| F10 | **Voice not available right now** | two variants: **busy** "Voice is busy right now. Try again in a minute, or start in text." (`429 voice_capacity`); **daily limit** "You've used today's 75 voice minutes. Voice is back at 05:30 tomorrow — text is available now." (`429 voice_daily_limit`, the reset rendered in the account timezone; "75" is 75 × the multiplier, so "112" at 1.5×) **[Continue in text]** | L19; ADR-0032 §6 caps |
| F11 | Screen-reader preset | "Using a screen reader? We'll switch to push-to-talk and ask you to wear headphones, so the reader's speech doesn't interrupt the interviewer. Captions stay on in a log you can move through." **[Use this preset]** · link "Accessibility settings" (AB28) | t6 §7 accessibility |
| F12 | **Ready to start** | Checklist with icon + text: "Chrome ✓ · Microphone ✓ · Voice check ✓ · Consent ✓ · Stops at $5.25 · Time multiplier 1×"; "The interviewer will tell you it's an AI when it starts."; **[Start interview]** (primary; this click is the audio/mic gesture) **[Back]** | t6 §4 (gesture); ADR-0032 §6 (AI disclosure) |
| F13 | **Resume by voice** (the compact reaffirmation inside AB26's resume modal) | "Resuming by voice: your mic streams to OpenAI on your key, and audio passes through xLearn's server in memory only — never stored." **[Resume interview]** **[Resume in text]** | t6 §7 ("a compact reaffirmation on resume") |
| F14 | < 1024 px / 390 px | Steps stacked as one column with a step counter "2 of 5"; the notices (F7–F10) as full-width cards with the text-mode button first | — |

Behaviour notes must state: `voice.available` and `voice.reason ∈ {region, no_voice_key, model_access, capacity, daily_limit,
disabled}` come from the server's pre-flight response (never inferred client-side, except the browser check, which is a
capability + user-agent test per the S6 list); the region is read from the account (an attestation) and the server refuses voice
anyway (`403 voice_unavailable_region`); the voice check is a real ≤ 30 s provider session that counts toward the day's voice minutes
and the cost; `ErrModelAccess` never offers retry; consent is per session, versioned, stored with a timestamp; every notice keeps text
mode one click away; nothing on AB29 claims a score or evaluates the voice.

### 3 · AB30 ★ voice live HUD (hero) [X]

`AB30-voice-live-hud.html` — the live voice interview, from the first "I'm an AI interviewer" to the debrief. It **extends AB25**
(rail, server clock, editor + Run, `give_hint`, snapshot indicator) and reuses AB26's grace and paused vocabulary. Consumed by
[m6b-03](sprint-m6b-03.md) (task 2) on top of [m6b-01](sprint-m6b-01.md) and [m6b-02](sprint-m6b-02.md). Sources:
[t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (states, failure table, grace),
[t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (rail, quiet during Code, hold voice, hints),
[t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (captions, modes, shortcuts, self-view, screen-reader preset),
[t6 §6](../research/t6-realtime-interviewer.md#6-assessment) item 4 (distress trigger on words),
[ADR-0032 §3](../../adr/0032-realtime-ai-mock-interviewer.md#3-what-the-ai-perceives-a-second-interviewer-sharing-the-session-owner-d29) (D29),
[§4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec), [§5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring),
[PRD](../../prd/xlearn-v2-prd.md) §5.6 R-MI2–R-MI4.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Live · interviewer speaking** (Clarify the problem) | AB25's phase rail with its labels **verbatim** — "Clarify the problem 0–5 · Brute force out loud 5–10 · Key observation → plan 10–18 · Code it 18–33 · Trace + edge cases 33–40 · Complexity + follow-ups 40–45" — with "Clarify the problem" active; server clock **`03:12 / 45:00`** (mono, `xl-timer`); chips `Voice · Standard`, `Captions on`; indicator (icon + text) **"Interviewer speaking"**; captions panel, the fixed voice disclosure (never model-generated; [m6b-01](sprint-m6b-01.md) speaks exactly this): "Hi, I'm an AI interviewer. I can't see you — I follow what you say and your editor. I'll ask a few questions, then we'll code together." → "Before you start, what input size should we plan for?"; problem statement panel; control bar **[Mute]** (mic live) · **[Hold voice while I code]** · **[Ask to repeat]** · **Self-view off** · **[End interview]** | R-MI2 (AI disclosure); ADR-0032 §3; t6 §5 rail |
| F2 | **Candidate speaking** | Indicator **"You're speaking"** + still level; candidate caption tagged **"approximate"** ("So the array can have up to ten to the five elements…"); barge-in note in the aside ("talking over the interviewer stops it") | t6 §7 captions; S6 M4 |
| F3 | **Code phase** | Rail at **Code it** (`18–33`); CodeMirror in interview mode (no autocomplete) with **[Run]**; results line "Last run · 7/9 passed · wrong answer on a hidden case" (the learner-visible DTO only); indicator **"The interviewer sees your code as you type · updated 2 s ago"**; chip **"Quiet while you code — ask anything"**; **[Ask for a hint]** ("Hints are recorded"). No pattern chip, no reference solution | D29 (widgets as structured state, 2–3 s); t6 §5 (quiet during Code, `give_hint`) |
| F4 | **Push-to-talk** | Large **[Hold to talk]** button with `xl-kbd` **Ctrl + Space**; states **idle** "Hold Ctrl + Space to talk" / **talking** "Talking… release to send" / **waiting** "The interviewer is answering"; `ds-toggle` **"Push-to-talk"** in the control bar (switchable mid-call); aside: native (mini: `turn_detection:null`) or emulated (GPT-Live: mute/unmute) per S6 | t6 §7 modes; S6 M3b |
| F5 | **Voice on hold** | Banner "Voice is on hold. The clock is still running, and the interviewer can still see your code." **[Talk]** ("reconnects in about 2 seconds"); muted line "Holding voice while you code saves credit." Mic indicator off | t6 §5 ("hold voice while I code") |
| F6 | **Reconnecting** | variants: **rollover** "Starting the coding round… the interviewer will pick up where you left off." (`ds-spin`, clock running); **network** "Connection lost — reconnecting. The clock is paused." + grace countdown **`4:38`**; **restored** toast "Reconnected." Failure leads to AB26's paused state | t6 §4 (bridging, network drop, free re-prime); ADR-0032 §4 |
| F7 | Mic lost | "Your microphone disconnected. Reconnect it or choose another input." **[Choose input]**; after 10 s "Reconnect your mic — if it isn't back in 30 seconds, we'll pause the interview and stop the clock." | t6 §4 failure table; [m6b-03](sprint-m6b-03.md) posts `interrupt {reason: "mic"}` at 30 s, [m6b-02](sprint-m6b-02.md)'s `mic` reason (t6's "clock stops after 10 s" is simplified to the 30 s interrupt — Decisions to confirm) |
| F8 | **Still there?** | modal `ds-modal--sm` "Still there? The voice connection ends in 60 seconds." **[I'm here]** (no minute count: the check fires 60 s before the idle limit, which is 3 min × the multiplier); then **ended** "We ended the voice connection to save your credit. Your place is saved." **[Reconnect voice]** | t6 §4 idle (3 min × multiplier, hang-up); [m6b-02](sprint-m6b-02.md) `notice{kind:"idle_check"}` 60 s before `interrupt(idle)` |
| F9 | **Credit grace countdown** (the voice version of AB26's grace) | `ds-modal--md` "Your OpenAI credits ran out — interview paused at **23:41**."; countdown **`5:00`** (mono) + "Checked 20 s ago"; **[Top up ↗]** with "Top-ups can take a few minutes to register. OpenAI first deducts any negative balance, so add at least **$2.40**. If it doesn't register in time, we'll save your place, and resuming is quick."; **[Check now]** (disabled for 10 s after use) **[Save & resume later]** **[End interview]**; ≥ 75% variant adds **[Finish & get feedback]**, which reads **[End without feedback]** while the key is still dry; behind it: mic shown muted, camera stopped, editor read-only; "Spent so far $2.10 of your $5.25 cap" | t6 §4 failsafe 1; ADR-0032 §4; t6 §8 (meter shown in grace) |
| F10 | **Cap** | **85%** banner (`ds-badge--warn`) "You're close to your $5.25 cap, so the interviewer is wrapping up."; **100%** "The interview ended at your $5.25 cap. Your feedback is next." **[See feedback]** (→ AB27); **voice can't reconnect** (`409 cap_reached`: a Talk after hold, or a reconnect, with under a minute of voice left under the cap) "Voice can't reconnect — less than a minute of voice is left under your $5.25 cap. The interviewer can still see your code." **[Finish & get feedback]** (shown where the interview can finish: from hold, or ≥ 75% through a grace or resume) **[End interview]**; the live meter is **not** on the HUD otherwise (see Decisions to confirm) | ADR-0032 §6 (85% wrap); t6 §8 (meter hidden live); [m6b-02](sprint-m6b-02.md) (`cap_reached` at mint) |
| F11 | **Wrapping / debrief** | Rail at "Debrief · 3 min"; "Wrapping up — about 3 minutes of feedback."; captions of 2–3 improvement areas ("Say your plan out loud before you start coding."); line **"Scores come next — you'll review them before anything is saved."** — no number, no band, no "you did well/badly" on tone; variant **entered while voice is on hold** (rail end, finish or cap from `held`): the same debrief as text in the captions log, no reconnect ([m6b-02](sprint-m6b-02.md)) | ADR-0032 §5 (brain-authored, no score); D14 (no auto-accept for mocks) |
| F12 | **Captions log** | A scrollable `role="log"` transcript: speaker labels "Interviewer" / "You (approximate)", timestamps (mono), **[Ask to repeat]** and **[Rephrase that]**; "Captions" `ds-toggle` (on by default) | t6 §7 (captions on by default, navigable log) |
| F13 | **Self-view** | off: "Self-view off" + `ds-toggle` **"Show my camera on this screen"**; on: small mirrored tile with the label **"Only you can see this. It isn't sent to the AI or recorded."**; camera lost: the tile disappears quietly, nothing else changes | D29 (no video to the AI); t6 §7 (off by default) |
| F14 | **Distress card** | `ds-card` "It sounds like things may be hard right now. If you'd like to talk to someone: **Tele-MANAS 14416** (India, free, 24×7) or your local emergency number." **[Pause interview]** **[Keep going]** — raised by words in the transcript, never by the voice | t6 §6 item 4 |
| F15 | **Live in another tab** | "This interview is live in another tab." **[Use this tab]** ("The other tab will disconnect.") — taking over hangs up the other tab's voice connection | t6 §4 (refresh / second tab); [m6a-01](sprint-m6a-01.md) client lease (`409 lease_lost`, `notice`) |
| F16 | **Screen-reader preset** | F4's push-to-talk + a persistent "Headphones on" chip; only phase changes and the grace warning are announced (`assertive`); captions in F12's log | t6 §7 screen-reader preset |
| F17 | **Check your transcript** (after the call, before AB27's proposal review) | "Before we propose scores, fix anything the transcript misheard in **your** lines. The interviewer's lines can't be edited."; each candidate line with **[Edit]** (inline `ds-input`, ≤ 2 KiB) and an "edited" chip after saving; banner on the first edit: "Edited lines are fine — this mock's score will be labelled **self-reviewed**, and the AI's proposal stays a suggestion."; **[Continue to scores]** | t6 §6 step 4 (voice self-edit → `transcript_edited`); [m6b-02](sprint-m6b-02.md) `PATCH …/turns/{seq}`; AB27 F3/F7 |
| F18 | < 1024 px / 390 px | Rail collapses to AB25 F15's "Phase 4 / 6 · Code it · 21:40 / 45:00"; captions become a bottom sheet; editor full width; control bar sticky at the bottom with icon + text buttons | t6 §5 (editor first) |
| F19 | **Voice minutes running out** (the live daily limit) | At 3 voice minutes left today: banner (`ds-badge--warn`, icon + text) "3 voice minutes left today — the interviewer is wrapping up this part."; at the limit, AB26's paused banner with "You've used today's 75 voice minutes, so we paused the interview and saved your place. Voice is back at 05:30 tomorrow — or resume in text now." **[Resume in text]** (AB26's resume modal with AB29 F13's text choice) · "Resume by 14:10 tomorrow" (AB26's `resume_by` line) | L19; [m6b-02](sprint-m6b-02.md) task 4 (3-minute wrap cue, then `interrupt(voice_limit)` → `save_later` → `paused`; counts toward m6a-01's ≤ 3 pauses and its 24 h `resume_by`); 75 × the multiplier |

Behaviour notes must state: the clock renders from the server (`active_ms` + `live_since`), never a client timer, and runs only in
m6a-01's running set (`live`, `held`, `bridging`, `wrapping`); phase lengths and the idle/check-in timers scale with the time multiplier; the "sees your code"
indicator reads the last snapshot ack (D29: widget state on change every 2–3 s and on every turn — never screenshots, never the
camera); the event-log kinds on the bounded SSE drive the states (M6a's `state` — incl. `grace_until` and the interrupt reasons, M6b's
`mic` and `voice_limit` among them —, `phase`, `turn`, `cap`, `notice` — incl. `idle_check`, `probe`, "live in another tab" — plus M6b's
`caption` and `segment` — incl. `reprime`, `rollover`) from [m6b-01](sprint-m6b-01.md) / [m6b-02](sprint-m6b-02.md); the typed errors a
new voice connection can meet (`409 cap_reached`, `429 voice_capacity`, `429 voice_daily_limit`, `409 provider_quota`,
`401 provider_auth`, `502 provider_unavailable`, `504 sdp_timeout`) each map to a frame; interviewer captions come from the model's output transcript, candidate
captions are labelled approximate; **no score, band or delivery metric appears on the HUD**; nothing comments on tone, accent or
emotion; shortcuts (Ctrl + Space talk, Ctrl + Shift + U mute, Ctrl + Shift + E end → confirm) avoid CodeMirror's interview-mode and
Chrome/Edge defaults — verify both lists; `prefers-reduced-motion` stops the level and speaking animations; End asks for confirmation
with **[Keep going]** as the default button.

### 4 · Self-review checklist (run before merging) + screenshots [X]

The session's own gate before the merge (D40: nothing waits on the owner); its ticked result goes in the PR body. Fix what
fails, then re-check.
- **`theme.css` check:** linked (then `board.css` if on `main`), never copied or overridden; `--ds-*` tokens only, no new
  colours; difficulty Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; indicators icon + text; the index's file names.
- Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) M6b row ("AB29 voice
  pre-flight, browser and EU notices · AB30★ voice live HUD") and the register brief (AB30: rail, captions, mic/camera state with
  optional self-view, editor + Run, Hold/Talk and push-to-talk, "hold voice while I code", countdown grace, cost vs $ cap).
- **Decision checks:** D29 (no video to the AI; self-view off by default and local only; the AI follows widgets), ADR-0032 §6
  (Chrome/Edge; EU/EEA off; Anthropic-only → text; cap mandatory, 85% wrap; audio in memory only, disclosed), ADR-0032 §5 (no score
  on the HUD, brain-authored debrief), D27 (no start refusal because of an open attempt — item selection skips live items),
  D30 (pause → `incomplete` after 24 h: AB26's copy, not redrawn), D31 (nothing public), D34 (no "we've notified the team").
- **Leak check:** no pattern chip, reference solution, unreleased hint, hidden test, expected output, rubric anchor or exemplar on any
  frame; the Run line shows only passed/total and the failure class.
- **Copy check against neighbours** on `main`: AB24 (consent box strings, estimate layout, key/model pre-flight), AB25 (rail, clock,
  editor, hint), AB26 (grace, paused banner, resume modal), AB27 (debrief → proposal hand-off), AB28 (captions, multiplier),
  AB19 (region wording). Mismatches go into "Decisions to confirm".
- Screenshot every board full-page at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB29@1440.png`,
  `AB29@390.png`, `AB30@1440.png`, `AB30@390.png` (each ≲ 500 KB).

### 5 · Open the design PR [X]

Branch `design/ds-m6b-01`; conventional commit `docs(design): AB29 AB30 — M6b voice boards` ending with the attribution lines. Open
a PR titled `docs(design): AB29 AB30 ★ — M6b voice boards` with each board's frame list, the screenshots embedded
(`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6b-01/design-system/screens/v2/shots/<file>?raw=true`), the ticked
self-review checklist (task 4) and a **"Decisions to confirm"** list. The list doesn't block the merge: each item states the
default the merge freezes (the boards as drawn), and the owner may revisit any item after the merge through a follow-up design
PR. At least:
- **Live spend on the HUD.** [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) hides the live meter during the
  interview (shown in grace, pause, resume and the report); the register brief asks for "cost meter vs $ cap". The board follows
  t6: no meter live, an 85% wrap banner, spend in the grace modal. Alternative: an opt-in "Show spend" disclosure, off by default.
- **All five consent boxes required** for voice (retention is a choice, not a box).
- **Shortcuts** (Ctrl + Space, Ctrl + Shift + U, Ctrl + Shift + E) — checked against CodeMirror interview mode and Chrome/Edge.
- **Push-to-talk switchable mid-call** (F4), not only at setup.
- **Voice on narrow screens** — the boards draw a 390 px layout; whether voice should be offered on phones at all.
- **Region wording** (F8) — it names the account's region, an attestation; where "Confirm your region" (the no-region variant) leads —
  the acceptance step or Settings: draw one (the conservative reading), state it here, and the owner may revisit it after the merge.
- **Self-edit placement** (AB30 F17) — M6a's boards left voice transcript self-edit to M6b; drawn here as the step between the call and
  AB27's proposal review.
- **Mic loss timing** (AB30 F7) — t6 §4 says the clock stops 10 s after the mic is lost; the build interrupts (and stops the clock) at 30 s
  ([m6b-03](sprint-m6b-03.md) / [m6b-02](sprint-m6b-02.md)'s `mic` reason), and the copy says so. Alternative: a server-side clock rewind to
  the 10 s mark.
- **Idle pre-warning in voice only** (AB30 F8) — AB25 F12 (text) has no countdown; voice warns 60 s before the hang-up
  (`notice{idle_check}`), because an idle voice connection is billed.
- **Voice can't reconnect near the cap** (AB30 F10) — which actions `409 cap_reached` offers (finish where the FSM allows it, else end).
- **Daily voice limit mid-interview** (AB30 F19) — the interview pauses (counting toward the 3 pauses and the 24 h resume window) rather
  than ending; resuming in text stays available.

### 6 · Freeze: merge on CI green [X]

Once the PR number is known, push the status commit (the Status note). When CI is green (fix, then merge, on failure),
squash-merge: **the merge is the freeze** (D40). Never enable auto-merge. Then sync `main` (`git checkout main && git pull`).
[m6b-01](sprint-m6b-01.md) cannot start until it lands (its entry gate), and [m6b-03](sprint-m6b-03.md) builds against exactly
these files. The owner may review after the merge; any change is a follow-up design PR.

## Acceptance criteria

- [ ] Every frame listed for AB29 (F1–F14, incl. F5a–F5d) and AB30 (F1–F19) is present with final copy, its state, its decision refs
      and a behaviour-notes aside.
- [ ] No frame shows video going to the AI, a score or delivery metric on the HUD, or withheld item data; every "no voice" notice keeps
      text mode one click away; the consent frame discloses audio passing through xLearn's server in memory only.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied or overridden; only this sprint's two board files and their
      screenshots are added (no `index.html`/`board.css` edit).
- [ ] The self-review checklist passed and is ticked in the PR body, with the "Decisions to confirm" list (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots of each board; this file and `docs/v2/status.md` record ds-m6b-01 ✅ and AB29, AB30 "frozen (merged)".

## Release

**Land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag. Nothing deploys: boards are preview-only files
under `design-system/screens/v2/`, never imported by `web/`. Launching the prompt is the owner's approval
([D40](../feasibility.md#decisions-log-newest-first)), so nothing waits on the owner; the owner may review after the merge, and
any change to a frozen board is a follow-up design PR. The merge is the freeze that gates [m6b-01](sprint-m6b-01.md) (M6b entry)
and is what [m6b-03](sprint-m6b-03.md) implements. No infra PR; the only `status.md` edits are this sprint's rows.

## Definition of Done

PR merged on CI green (the freeze) with AB29–AB30 and screenshots · every acceptance box ticked · this file's Status all ✅ and
`docs/v2/status.md` updated (Sprint-board row ✅; Artboards AB29, AB30 "frozen (merged, PR #N, <date>)") · no `web/`, `internal/`, `index.html` or `board.css` change · local `main` synced.

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D27, D29, D30, D31, D34, ADR-0032 §3–§6): cite the decision per frame; where the
  register brief and t6 disagree (the live cost meter), follow the ADR/t6 and list it under "Decisions to confirm".
- **Drawing both shells.** P1 ships one shell; boards that hedge between GPT-Live and mini invite m6b-03 to build two paths. Use the
  S6 winner and keep the other to one aside line.
- **Implying the AI sees the learner.** Self-view is local; any copy like "the interviewer can see you" or a camera icon on the AI
  side is a D29 violation.
- **Evaluative voice.** No frame may show the interviewer praising or criticising delivery, tone or confidence — the debrief is
  content-only and brain-authored.
- **Countdown drift.** Grace, idle and phase timers are server deadlines; a board implying a client timer invites the bug m6b-03 lists.
- **Copy drift from AB24/AB26.** The voice consent and grace frames extend M6a's boards; if a string differs, it is flagged, not
  silently changed.
- **Parallel status edits:** other sessions may update `docs/v2/status.md` when they land; rebase on `origin/main` before the
  status commit, touch only this sprint's rows and keep theirs.
