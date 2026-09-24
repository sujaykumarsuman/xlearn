# Sprint ds-m6a-02 — Design M6a part 2: grace/paused/resume, debrief + proposal, accessibility (AB26, AB27, AB28)

> **Milestone:** M6a — text interviewer plus failsafes (the boards M6a builds against) · **Track:** design · **Order:** 73
> **Prereqs:** [ds-m6a-01](sprint-ds-m6a-01.md) (ADR-0032 Accepted on `main`; AB24/AB25 drafted — AB26 reuses the HUD and the estimate, AB27 reuses AB13's evidence \| sliders) · [ds-m1-01](sprint-ds-m1-01.md) merged (board index, `board.css`)
> **Unblocks:** [m6a-01](sprint-m6a-01.md) (entry gate "AB13, AB24–AB28 frozen", together with ds-m6a-01) · consumed by [m6a-06](sprint-m6a-06.md) (builds AB26, AB27, AB28 and proves the M6a exit) · [ds-m6b-01](sprint-ds-m6b-01.md) (AB30★'s countdown grace extends AB26 F1; AB29 uses AB28's voice preferences)
> **Release action:** **PR, stop for owner review (design).** The agent never merges; the owner's merge (or explicit approval in chat) **is the freeze**
> **Calendar:** Q1 2027 (right after ds-m6a-01) · owner event **`ev-freeze-ds-m6a-02`** (review + merge) before [m6a-01](sprint-m6a-01.md) starts
> **Execute with:** [`../prompts/prompt-ds-m6a-02.md`](../prompts/prompt-ds-m6a-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB26 grace, paused, resume (+ the `incomplete` expiry) | X | ⬜ |
| 3 | AB27 debrief + proposal (explicit accept), `incomplete` free partial view | X | ⬜ |
| 4 | AB28 accessibility settings | X | ⬜ |
| 5 | Self-review against the brief + screenshots (1440 px, 390 px) | X | ⬜ |
| 6 | Open the design PR and STOP | X | ⬜ |
| 7 | Freeze: owner reviews, approves and merges (`ev-freeze-ds-m6a-02`) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Design exception:** this sprint's PR does **not** edit [`../status.md`](../status.md)
> (the PR may stay open for days). The PR sets tasks 1–5 ✅, task 6 🔄 ("PR #N open — awaiting owner review") and _Overall_ 🔄.
> The first consuming build sprint, [m6a-01](sprint-m6a-01.md) (its entry gate is this freeze), sets tasks 6–7 and _Overall_ ✅,
> writes "frozen (PR #, date)" into the status.md Artboards rows for AB26, AB27 and AB28 and ticks `ev-freeze-ds-m6a-02`.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] none beyond `depends_on`: [ds-m6a-01](sprint-ds-m6a-01.md) — its **ADR PR merged** (ADR-0032 **Accepted** on `main`) and its **board PR open or merged** (`AB13-mock-v2.html`, `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html` on `main` or on `design/ds-m6a-01`). If that PR has requested changes, draft against the latest commit on its branch and say so in the PR.
- [ ] [ds-m1-01](sprint-ds-m1-01.md) merged: `design-system/screens/v2/index.html` and `board.css` on `main`. This sprint never edits either.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR adds `design-system/screens/v2/AB26-*`, `AB27-*` or `AB28-*`.

## Goal

Draft the second half of the M6a boards — **AB26** grace, paused and resume (the owner's three failsafes), **AB27** debrief and
the AI proposal with an **explicit accept**, plus the `incomplete` free partial view, and **AB28** accessibility settings — as
static, preview-only HTML on [`theme.css`](../../../design-system/theme.css) under `design-system/screens/v2/`, with every frame,
every state and final copy, so [m6a-06](sprint-m6a-06.md) builds against owner-approved designs and M6a can open. BP3: agents
draft every board; the owner only reviews.

These boards carry the failsafes the owner specified ([ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec)):
a 5-minute top-up grace, a pause of at most 24 hours from the first pause, a resume modal that shows the free state at once and
charges the key only on a click — and **D30**: a pause not resumed in 24 h becomes `incomplete`, unscored and out of trends. The
scoring board encodes T6's amendment to **D14**: a mock proposal is **never** saved automatically; the learner must accept or edit
it, and `ScoreMock` runs once.

## Scope

**In**
- `AB26-grace-paused-resume.html`, `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html` under `design-system/screens/v2/`
  (the ds-m1-01 index names), every frame in tasks 2–4, each with a behaviour-notes aside citing its decision.
- PNG screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/`, and the design PR, stopped for review.

**Out**
- Any `web/` code → [m6a-06](sprint-m6a-06.md). Backend states → [m6a-01](sprint-m6a-01.md) (FSM, sweeper, `pause_exposure`,
  retention, erase), [m6a-02](sprint-m6a-02.md) (probe, resume brief, classifier), [m6a-03](sprint-m6a-03.md) (review → proposal →
  accept → `ScoreMock` once, re-propose, twin gate).
- Mock home, setup, consent, pre-flight and the live HUD → AB13, AB24, AB25 in [ds-m6a-01](sprint-ds-m6a-01.md) (reference, don't redraw).
- The **countdown** grace UI, voice consent, captions, mic state and the voice HUD → AB29, AB30★ in [ds-m6b-01](sprint-ds-m6b-01.md)
  (P1 adds the countdown; P0's grace is a simple modal, [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)).
  Voice-only rows here carry a **"from M6b"** tag.
- Learner transcript self-edit for voice ASR and AI-notes checkpoints → P1 ([m6b-02](sprint-m6b-02.md), [m6b-03](sprint-m6b-03.md)).
- Platform-AI suggestions and the 24 h auto-accept → AB16 in [ds-m4-01](sprint-ds-m4-01.md): mocks deliberately differ.
- Editing `index.html` or `board.css` → [ds-m1-01](sprint-ds-m1-01.md). Shipping boards (preview-only) and owner design hours beyond review (BP3).

## Tasks

### 1 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named per the ds-m1-01 index:
`AB26-grace-paused-resume.html`, `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html` (if `index.html` links different
names, use those). Branch `design/ds-m6a-02` off an up-to-date `main`.
- Each links `../../theme.css`, then `board.css`, plus the Google Fonts `<link>` (Inter, JetBrains Mono), and reuses the tokens and
  components **verbatim** (dark "landscape console"; [`design-system/README.md`](../../../design-system/README.md)): `ds-card`,
  `ds-badge`, `ds-chip`, `ds-btn`, `ds-seg`, `ds-toggle`, `ds-meter`, `ds-modal--sm|md`, `ds-field`/`ds-input`, `ds-spin`, `ds-dot`,
  `xl-panel`, `xl-code`, `xl-lock`, `xl-timer`, `xl-kbd`, `xl-table`. Difficulty: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard =
  `--ds-err`. Mono for times, money and sizes. Board-only layout in the file's `<style>`, tokens only, never overriding `ds-*`/`xl-*`.
- **Reuse ds-m6a-01's vocabulary:** AB25's HUD chrome (rail, clock, bottom bar) behind the AB26 modals; AB24's estimate and cap
  wording in the meters; AB13 F6's evidence | sliders layout for AB27's proposal review; AB13 F8's chips (`self`, `AI-proposed ·
  accepted`, `honor`, multiplier).
- **Frames:** full-screen frames stack; modal and banner states sit side by side in a `.bd-grid`. Each frame has a label
  `ABnn-Fk · <state>`, its decision refs and **final copy** (numbers are example data; no lorem ipsum, no "TBD").
- **Behaviour-notes aside** per frame: the state and field that drive it, timers, gates, error codes (*proposed* ones flagged for
  m6a-01/m6a-03), the decision, a11y (focus order, contrast ≥ 4.5:1, keyboard, `aria-live` policy, reduced motion) and the
  `< 1024 px` layout. Every board ends with a `.bd-narrow` 390 px section.
- Static HTML only: no JS runtime, no `support.js`, no `.dc.html` canvas markup.
- **Never leak withheld data or scores** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) S1–S4): the resume state shows the
  learner's **own** code only; no pattern chip, reference solution or unreleased hint; the debrief never states a score; no mock
  score on any public surface (D31).
- **Touch only this sprint's three board files and their screenshots.**

### 2 · AB26 grace, paused, resume [X]

`AB26-grace-paused-resume.html` — every way a live interview stops without finishing, and how it comes back. Consumed by
[m6a-06](sprint-m6a-06.md) task 1 on top of [m6a-01](sprint-m6a-01.md) (FSM, sweeper, `pause_exposure`) and [m6a-02](sprint-m6a-02.md)
(probe, brief). Sources: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (the state
machine, failsafes 1–3, checkpoints, other failure modes), [ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec),
D30, D17 (the arena stays open; exposure is recorded), D27 (coach unlocked while paused), [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)
(the meter shows in grace, pause and resume), [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)
(compact consent reaffirmation on resume), PRD R-MI3–R-MI5.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Grace modal** (text, simple) over a read-only HUD | Title **"Your OpenAI credits ran out — interview paused at 23:41."**; "Top-ups can take a few minutes to register. OpenAI first deducts any negative balance, so add at least **$1.25**. If it doesn't register in time, we'll save your place — resuming is quick."; [Top up ↗] (the catalog `billing_url`, new tab); status line "We'll keep checking until 14:10 · checked 20 s ago"; meter "Spent so far: $0.31 of your $0.90 cap"; buttons [Check now] · [Save & resume later] · [End interview]. Variant tagged **"from M6b"**: the 5:00 countdown ring (AB30) | Failsafe 1; R-MI3; t6 §9 (P0 simple modal, P1 countdown) |
| F1b | **Grace modal, Anthropic key** (text interviews run on Anthropic keys too) | Title **"Your Anthropic credits ran out — interview paused at 23:41."**; "Top-ups can take a few minutes to register. If it doesn't register in time, we'll save your place — resuming is quick." **No negative-balance line and no "$1.25" minimum** (that is OpenAI's rule); [Top up ↗] goes to the catalog's Anthropic `billing_url` (the Console's billing page); status line, meter and buttons as F1. Every later provider-named string (F7, AB27 F10) follows the key's provider the same way | ADR-0032 §1 (P0 text works with Anthropic keys); t6 §4 (Anthropic `ErrQuota`: 400 "credit balance is too low", 402 `billing_error`) |
| F2 | Grace at ≥ 75% of the rail | Adds [Finish & get feedback]; while the key is still dry it reads [End without feedback] with "Feedback needs credit on your key. You can still score it yourself." | t6 §4 failsafe 1 step 3; D30 (early finish ≥ 75%) |
| F3 | Credits found | "Credits found — picking up where you left off." then the interviewer's re-entry "Welcome back. We were in Code it, and you were handling duplicates." — **no paid summary**; the clock restarts on that first message | Failsafe 1 step 5 |
| F4 | Other interruptions (inline, same 5-minute window) | Offline: "You're offline. We'll reconnect for free when you're back — until 14:10. Your clock has stopped." · provider outage: "Your provider isn't responding. We'll keep trying until 14:10 — your clock has stopped." · rate limit: "Your provider is rate-limiting this key." · **key rejected** (`ErrAuth`): straight to paused — "Your provider rejected your key, so we turned it off and paused the interview. Resume with another interview key." | t6 §4 (other failure modes; classifier) |
| F5 | **Paused banner** (every page, every device) | Bar **"Interview paused — resume by Thu 14:05"** [Resume]; detail "Pause 1 of 3 · 23:41 done"; "Your coach is available again, but won't discuss this interview's problem until you finish."; "If you open this problem in the arena while paused, it's noted on your result." | Failsafe 2 (absolute 24 h, ≤ 3 pauses); D27 (coach unlocked, item withheld); D17 (arena open, exposure recorded) |
| F6 | **Resume modal — free state, instant** | Title "Resume your interview"; per-phase rail "Clarify the problem 4:10 / 5:00 · Brute force out loud 6:02 / 5:00 · Key observation → plan 7:40 / 8:00 · **Code it 5:29 / 15:00 (current)**"; "22:47 left"; "Hints given: 1"; your last code (read-only `xl-code`, ≤ 12 lines, "saved 14:02"); "Runs: 3 · last: 7 / 9 sample passed"; pauses "Pause 1 · credits ran out · Wed 14:05"; "About $0.25 to finish on your key"; meter "Spent so far: $0.31 of your $0.90 cap"; reaffirmation "Same terms as before: your transcript and code go to your provider's text API, and the interviewer is an AI."; [Summarise & resume — uses your key (~$0.04)] (primary) · [Finish now & get feedback] (≥ 75% only) · [Abandon] | Failsafe 3; R-MI4; t6 §7 (reaffirmation) |
| F7 | Still no credit | "Still no credit on your key — nothing was charged. [Top up ↗] and try again." | Failsafe 3 step 2 (probe first) |
| F8 | **Brief ready** | Card: summary (≤ 600 chars) "You clarified the bounds, rejected the O(n²) scan and started a single pass with a hashmap…"; progress (≤ 5 bullets); "Next: finish the loop and handle duplicates."; [Resume interview] (primary); "If resuming fails again, we reuse this summary — you won't pay for it twice." | Failsafe 3 steps 3–5 (brief cached per pause) |
| F9 | Pause exposure | Chip **"Viewed the solution during the pause"**; "You opened this problem in the arena while paused. You can finish the interview, but it will be self-scored." | t6 §4 (`pause_exposure` → no `ai-byo`) |
| F10 | Last pause; expiry | Variant "This is your last pause (3 of 3)." · **Expired (D30):** "This interview expired — it wasn't resumed by Thu 14:05. It isn't scored and doesn't count in your trends." [See what you did (free)] (→ AB27 F9) · [Start a new mock] | D30; R-MI5 |
| F11 | Credit ran out at ≥ 90% | "Your credits ran out near the end. Finish without the rest? You'll get feedback once your key has credit, or you can score it yourself." [Finish without the rest] (default) · [Save & resume later] | t6 §4 (`finished(report_pending=quota)`) |
| F12 | < 1024 px / 390 px | Modals as full-screen sheets; the resume rail as a vertical list; the banner as a compact bar with [Resume] | — |

Behaviour notes must state: at the first `ErrQuota` the clock freezes at the last good moment, `grace_until = now + 5 min`, a
**deterministic checkpoint** is written (no AI call), the provider session is hung up and a cut-off interviewer sentence is marked
`truncated` and asked again; probes are browser-driven every 30 s, the server allows ≤ 1 per 10 s and ≤ 15 per grace, and the
sweeper probes once more at `grace_until`; a probe that passes needs 60 s stable before it counts, with ≤ 2 automatic re-primes per
grace; `resume_by = first_paused_at + 24 h` is absolute and at most 3 pauses are allowed; the sweeper moves an expired pause to
`incomplete`; **no AI call runs on the key without a click**; the brief is cached per pause (a failed resume doesn't pay twice);
times render in the account timezone with the weekday; the grace modal is `role="alertdialog"` and its warning is the **only**
`aria-live="assertive"` announcement; [End interview] and [Abandon] are never the default button.

### 3 · AB27 debrief + proposal [X]

`AB27-debrief-proposal.html` — from the last minutes of the interview to a saved score, and the free view of an `incomplete`
one. Consumed by [m6a-06](sprint-m6a-06.md) task 2 on top of [m6a-03](sprint-m6a-03.md) (review → proposal → explicit accept →
`ScoreMock` once, re-propose, twin gate) and [m6a-02](sprint-m6a-02.md). Sources: [t6 §6](../research/t6-realtime-interviewer.md#6-assessment)
(flow steps 1–7, the `ai-byo` conditions, the never-assessed list), [ADR-0032 §5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring),
[t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (D14 amendment; `ScoreMock` only from `finished`;
null bands), D30, D31, [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (retention,
per-turn delete), PRD R-MI6–R-MI8.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Debrief** (`wrapping`, in the AB25 HUD) | Chip "Debrief · up to 3 minutes"; the interviewer: "Thanks — that's time. Two things went well: you asked about duplicates before coding, and you traced your loop on the second example. Two things to work on: say the complexity before you start coding, and test the empty input out loud." **No score, no band.** [End debrief] | t6 §6 flow 1 (brain-authored; never states a score) |
| F2 | Preparing the review | "Reviewing your interview — one call on your key (about $0.06). Usually under a minute." (`ds-spin`); "Your final code is checked too: hidden tests 37 / 40." | t6 §6 flow 2 (`store:false`); t4 §6.7 (aggregate evidence) |
| F3 | **Proposal review** (`proposed`) | Header **"AI-proposed scores — nothing is saved until you accept."**; per dimension (the 7 DSA dimensions): band dots 1–5 or **"No evidence"** with its reason ("You didn't discuss complexity"), one verified quote "You wrote: 'I'll sort first so two pointers work' · message 14" (links into the transcript), a one-line why, and a slider to change it; the evidence column from AB13 F6; strengths (≤ 3); improvements (≤ 3), each an observable behaviour + "Practise: Two pointers on sorted input →"; the summary; caveats ("1 hint given", "Paused once"); footer [Accept all] (primary) · [Re-propose once — uses your key (~$0.06)] · [Score it myself]; banner "Scores lock when saved."; provenance "Proposed by gpt-6-sol · prompt v1 · from your messages and code only — never your voice or appearance." | T6's D14 amendment (explicit accept); R-MI6; t6 §6 (null bands, verified quotes, public descriptors only) |
| F4 | Edited | The changed row tagged `edited`; banner "You changed a score, so this mock will be saved as self-scored." | t6 §6 flow 5 (`ai-byo` only if every band is unchanged) |
| F5 | Second proposal | Two columns "First proposal · 26 / 35" and "Second proposal · 24 / 35", differences marked; [Accept the second proposal] (primary) · [Accept the first] · [Score it myself]; chip "Re-propose used"; "The second reviewer didn't see the first proposal." | t6 §6 flow 4 (re-propose once, blind, fresh sample) |
| F6 | **Saved** | "Saved · 26 / 35"; chips `AI-proposed · accepted` · `honor` (or `self` · `honor`); "Your improvement areas are in the mock's notes."; [Talk it over with your coach →] (coach review mode); "Your public profile shows 6 mocks — never scores." | t6 §6 flow 5 (`trust=honor` always; notes pre-filled); D31 |
| F7 | **Self-scored only** (no AI proposal) | One variant per reason, none naming a heuristic: custom model — "Custom models can't propose scores. Here are the quotes the AI found — score it yourself."; model not cleared to propose — "This model isn't set up to propose scores yet, so you'll score this one yourself."; `review_flag` — "We couldn't verify this review automatically, so this mock is self-scored."; pause exposure — "You viewed the solution during a pause, so this mock is self-scored."; tagged **"from M6b"**: "In voice interviews you score Communication yourself — the AI gives you quotes." Evidence and quotes shown; sliders start empty | t6 §6 (custom ids, twin gate `self_only`, `review_flag`, `pause_exposure`, voice Communication `self_only`) |
| F8 | **Scores waiting** | Mock history / Today card "Scores waiting for you · proposed Tue"; "Proposals are never saved automatically."; in-app reminder copy at 24 h and 7 days; "Score it by Jan 12, or it expires unscored." | T6's D14 amendment (no auto-accept; 30 days → `incomplete`) |
| F9 | **Incomplete — free partial view** | "Incomplete · not scored · not in your trends"; free, from the checkpoints: phase times against the rail, hints given, your last code, Runs and the last result, pauses and reasons; [Get improvement areas — uses your key (~$0.06)] only on a click; the result card when clicked (≤ 3 improvements, **no scores**) | D30; R-MI5 |
| F10 | Review failed; cut short | "We couldn't get the AI review: your OpenAI credits ran out." ("your Anthropic credits" on an Anthropic key) [Try again] · [Score it myself]; cut short by the cap: caveat "Ended at your spending cap"; cut short by credit: "Feedback is waiting for credit on your key." | t6 §4 (`report_pending=quota`, `cut_short=cap`) |
| F11 | **Transcript and retention** | The transcript with a delete control per message ("Delete this message? It's removed from the transcript. Scores you already saved don't change." [Delete] [Cancel]); "Your transcript is deleted automatically on Jan 14 (30 days after scoring)." [Keep for 12 months] · [Delete transcript now]; note "Delivery stats are for voice interviews only, for your information — never scored and never public." | t6 §7 (30 days after scoring, 12 months opt-in, per-turn delete); R-MI8 |
| F12 | < 1024 px / 390 px | Dimension rows as cards (band, quote, slider); evidence collapsed above; footer buttons stacked with [Accept all] first | — |

Behaviour notes must state: the review runs once at `finished` on the learner's `interview` key with `store:false` and produces
`xlearn.mock_review@1`, held in coach with `model` and `prompt@v`; quotes are checked against the transcript (normalise, then
substring) and a missing quote drops the claim; missing evidence gives a **null band** (no "clamp ≤ 2" for mocks); `scored_by=ai-byo`
only when the model is a non-custom catalog model that passed the twin gate, every band was accepted unchanged, no turn is `edited`
or `source=client`, `pause_exposure` and `review_flag` are unset, and the learner clicked Accept — otherwise `self`; `ScoreMock` runs
once, only from `finished` (*proposed*: `409 already_scored`, `409 repropose_used`); an untouched proposal stays private and becomes
`incomplete` after 30 days; improvement areas pre-fill `mock_session.notes` (≤ 2 KiB, no quotes); the public route shows the count
only; the never-assessed list (tone, emotion, accent, fluency, pace, face, appearance) appears in the aside.

### 4 · AB28 accessibility settings [X]

`AB28-accessibility-settings.html` — the Settings section for mock interviews and the accessibility choices the HUD honours.
Consumed by [m6a-06](sprint-m6a-06.md) task 3. Sources: [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)
(*Accessibility*: modes, screen-reader preset, captions, multiplier, camera off, shortcuts, reduced motion), [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)
(idle and check-in timers scale), [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (multiplier cost shown),
[ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits), WCAG 2.2.1 (timing adjustable).

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Settings → "Mock interviews"** | The account opt-in `ds-toggle` **"Let an AI interviewer run mock interviews with me"** (AB24 F1's strings exactly; switching off is immediate, no confirm); "Default mode" `ds-seg` [Text] (tagged "from M6b": [Voice] [Voice with push-to-talk]); **"Extra time"** [1×] [1.25×] [1.5×] [2×] — "Adds time to every phase and to the check-in timers. Shown on your results; never lowers your score. No reason needed."; **"Reduced motion"** [Follow my system] [On] [Off]; **"Screen reader announcements"** [Every interviewer message] [Only phase changes and warnings] | t6 §7 (modes; multiplier badged, never penalised, no diagnosis); WCAG 2.2.1 |
| F2 | Extra-time cost | "1.5× adds about $0.15 to a text interview on your key." (tagged "from M6b": "…and about $1.20 to a voice interview") | t6 §7 ("its added cost is shown"); §8 |
| F3 | Voice preferences (tagged "from M6b") | "Captions" on by default ("Interviewer captions come from the AI's own words; your captions are approximate."); "Screen-reader preset: push-to-talk, a headphones reminder and announcements limited to phase changes and warnings"; "Camera self-view: off by default — only you see it; it's never sent." | t6 §7 (captions on by default; screen-reader preset; camera off) |
| F4 | Text mode in the HUD with these settings | The AB25 HUD with: replies shown whole (no streaming animation) under reduced motion; "Jump to latest" in the transcript log; a visible focus ring on the log, input, editor and bottom bar in tab order | t6 §7; `prefers-reduced-motion` |
| F5 | **Keyboard shortcuts** (`?` opens the sheet) | Actions: send · new line · move between chat, editor and problem · Run · ask for a hint · Hold/Talk · (from M6b) mute and push-to-talk. Finish and End have **no** shortcut. This departs from t6 §7, which lists one for "end", and is listed under "Decisions to confirm". The alternative is a chord that only opens the confirm modal (AB25 F11), never ends directly. Each chord shown with `xl-kbd` | t6 §7 (shortcuts for mute, push-to-talk and end that don't clash with the editor) |
| F6 | Result badges | Chips on results and history: "1.5× time" (neutral `ds-chip`, never warn or err), "Text interview" | t6 §7 (badged, never penalised) |
| F7 | < 1024 px / 390 px | Settings rows stacked full width, helper text under each control; the shortcut sheet as a full-screen list | — |

Behaviour notes must state: the multiplier is chosen before a session starts (AB24 F2 defaults from here) and is fixed for that
session (`mock_session.time_multiplier`); it scales the rail, the idle check-in (3 min × multiplier) and the quiet-during-Code
threshold (90 s × multiplier); no diagnosis or reason is ever collected; indicators use icon + text, never colour alone; the default
announcement policy is "only phase changes and warnings" (polite; the grace warning assertive); **the proposed chords are checked
against CodeMirror's `defaultKeymap` (e.g. `Mod-Enter` is `insertBlankLine`), AB07's shortcuts, macOS Option-character input and
common screen-reader keys — an action with no safe chord gets none**; toggles are `<button role="switch">` with `aria-checked`.

### 5 · Self-review against the brief + screenshots [X]

- Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) M6a row
  ("AB26 grace, paused, resume · AB27 debrief + proposal (explicit accept), `incomplete` · AB28 accessibility settings").
- **Decision checks:** the failsafes as specified (5-minute grace, 24 h absolute, ≤ 3 pauses, free state first, paid brief only on a
  click and cached); D30 (`incomplete`: unscored, out of trends, free partial view, paid improvements only on a click); T6's D14
  amendment (no auto-accept anywhere); D31 (no public score); D17 and D27 (arena open during a pause, exposure recorded, coach
  unlocked but the item withheld); the never-assessed list; D34 (no copy implying anyone is alerted).
- **Leak check:** only the learner's own code in resume and partial views; no pattern chip, reference solution or unreleased hint;
  the debrief never states a score.
- **Copy check against ds-m6a-01** (AB24's opt-in and cap strings, AB25's HUD, AB13's chips and evidence). Mismatches → "Decisions to confirm".
- Screenshot each board **full-page** at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB26@1440.png` …
  `AB28@390.png` (each ≲ 500 KB). A plain headless `--screenshot --window-size=1440,900` captures only the viewport, and the
  Browser pane can't write files. Use `npx playwright screenshot --full-page`, or a tall headless window sized to the board's
  `scrollHeight` (the prompt has the commands).

### 6 · Open the design PR and STOP [X]

Branch `design/ds-m6a-02`; conventional commit `docs(design): AB26 AB27 AB28 — M6a boards, part 2` ending with the attribution lines.
Open a PR titled `docs(design): AB26 AB27 AB28 — M6a boards (part 2)` with each board's frame list, the screenshots embedded
(`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6a-02/design-system/screens/v2/shots/<file>?raw=true`), a link to
ds-m6a-01's PR, and **"Decisions to confirm"**, at least:
- the **simple** grace modal in M6a (no countdown ring until AB30), with "We'll keep checking until 14:10" instead;
- after a re-propose, the learner may accept **either** proposal unchanged (t6 says "re-propose once" but not which one stands);
- the *proposed* codes (`already_scored`, `repropose_used`) for m6a-03;
- the paid partial improvements on an `incomplete` interview priced like a review (~$0.06);
- the default screen-reader announcement policy and the shortcut chords;
- **End has no shortcut**, although t6 §7 lists shortcuts "for mute, push-to-talk and end". The alternative is a chord that only
  opens the confirm modal;
- the provider-named grace and review copy: AB26 F1 is OpenAI and F1b is Anthropic, and only OpenAI shows the negative-balance line;
- any divergence from AB24/AB25/AB13 while ds-m6a-01 is still open.

**Do not merge.** Stop for owner review (event `ev-freeze-ds-m6a-02`). Requested changes go on the same branch.

### 7 · Freeze [O]

The owner reviews the boards and approves in chat or merges the PR. **Merging is the freeze.** An agent may merge only on the
owner's explicit approval in chat. [m6a-01](sprint-m6a-01.md) can't start until this freeze and ds-m6a-01's have landed;
[m6a-06](sprint-m6a-06.md) builds exactly these files.

## Acceptance criteria

- [ ] Every frame listed for AB26 (F1–F12 + F1b), AB27 (F1–F12) and AB28 (F1–F7) is present with final copy, its state, its decision refs and a behaviour-notes aside.
- [ ] The failsafes read exactly as specified (5-minute grace, 24 h absolute resume deadline, ≤ 3 pauses, free state first, paid brief only on a click and cached), `incomplete` is unscored and out of trends, and no frame auto-saves a mock score.
- [ ] No board leaks withheld data, states a score in the debrief, or shows a mock score publicly.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied or overridden; only this sprint's three board files and their screenshots are added.
- [ ] PR open with 1440 px and 390 px screenshots and "Decisions to confirm"; **not merged by the agent**.

## Release

**PR, stop for owner review (design).** Nothing deploys: boards are preview-only files under `design-system/screens/v2/`, never
imported by `web/`. The owner's merge is the freeze (`ev-freeze-ds-m6a-02`); with ds-m6a-01's it gates [m6a-01](sprint-m6a-01.md),
and [m6a-06](sprint-m6a-06.md) implements it. No tag, no infra PR, no `status.md` edit in this PR.

## Definition of Done

PR open with AB26–AB28 and screenshots · every acceptance box ticked · this sprint's Status table updated in the PR (tasks 1–5 ✅,
task 6 🔄 awaiting review) · no `web/`, `internal/`, `index.html`, `board.css` or `docs/v2/status.md` change · the agent stops at
the open PR. (Done-done — ✅ Overall — is set by [m6a-01](sprint-m6a-01.md) once the owner has merged.)

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D14 as amended by T6, D17, D27, D30, D31; ADR-0032 §4–§5): cite the decision per frame; where sources disagree, follow the owner decision and list it under "Decisions to confirm".
- **Charging without a click.** Every paid action names its price on the button ("uses your key (~$0.04)"); nothing on these boards spends the learner's key on its own.
- **Countdown drift.** Deadlines come from the server (`grace_until`, `resume_by`) in the account timezone; a board implying a client timer invites the bug m6a-06 would build.
- **Auto-accept creeping back in** from AB16's platform-AI pattern: mocks never auto-save, and an untouched proposal expires to `incomplete`.
- **Blame-free failure copy.** Quota, outage and rejected-key states say what happened and what still works; never "you did something wrong", never an alert promise (D34).
- **Accessibility settings that promise voice features early:** keep every voice row tagged "from M6b".
