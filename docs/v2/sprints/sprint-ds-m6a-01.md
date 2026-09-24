# Sprint ds-m6a-01 — Design M6a part 1 + ADR-0032 → Accepted: Mock-v2, setup/consent/pre-flight, live HUD text (AB13, AB24, AB25)

> **Milestone:** M6a — text interviewer plus failsafes (the boards M6a builds against) · **Track:** design · **Order:** 72
> **Prereqs:** [spk-04](sprint-spk-04.md) (the S6 result in t6 §16 and the fixtures) · [ds-m1-01](sprint-ds-m1-01.md) merged (board index, `board.css`)
> **Unblocks:** [ds-m6a-02](sprint-ds-m6a-02.md) (AB26–AB28 reuse this sprint's HUD, meter and evidence vocabulary) · [m6a-01](sprint-m6a-01.md) (entry gate "AB13, AB24–AB28 frozen", together with ds-m6a-02) · consumed by [m6a-05](sprint-m6a-05.md) (builds AB13, AB24, AB25) · [mi-13](sprint-mi-13.md) (entry gate "ADR-0032 Accepted") · referenced by [ds-m6b-01](sprint-ds-m6b-01.md) (AB29/AB30★ extend AB24/AB25)
> **Release action:** task 1 (ADR-0032 → Accepted) ships as **its own docs PR, merged per land-and-sync** (the owner pre-authorised the acceptance in BP2). The boards: **PR, stop for owner review (design)** — the agent never merges them; the owner's merge (or explicit approval in chat) **is the freeze**
> **Calendar:** Q1 2027 (after GA and after [spk-04](sprint-spk-04.md)) · owner event **`ev-freeze-ds-m6a-01`** (review + merge) before [m6a-01](sprint-m6a-01.md) starts
> **Execute with:** [`../prompts/prompt-ds-m6a-01.md`](../prompts/prompt-ds-m6a-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Accept ADR-0032 with the S6 result (own docs PR, merged) | X | ⬜ |
| 2 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 3 | AB13 A9 Mock-v2 (server-picked setup, multi-item, evidence \| sliders) | X | ⬜ |
| 4 | AB24 setup + consent + pre-flight + $ cap | X | ⬜ |
| 5 | AB25 live HUD (text) | X | ⬜ |
| 6 | Self-review against the brief + screenshots (1440 px, 390 px) | X | ⬜ |
| 7 | Open the design PR and STOP | X | ⬜ |
| 8 | Freeze: owner reviews, approves and merges (`ev-freeze-ds-m6a-01`) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Two PRs, two rules.** Task 1's ADR PR merges in-session and **does** update
> [`../status.md`](../status.md) (ADR row, Decisions log, this sprint's board row 🔄) and sets task 1 ✅ here. **Design exception:**
> the board PR does **not** edit `status.md` (it may stay open for days); it sets tasks 2–6 ✅, task 7 🔄 ("PR #N open — awaiting
> owner review") and _Overall_ 🔄. The first consuming build sprint, [m6a-01](sprint-m6a-01.md) (its entry gate is this freeze), sets
> tasks 7–8 and _Overall_ ✅, writes "frozen (PR #, date)" into the status.md Artboards rows for AB13, AB24 and AB25 and ticks
> `ev-freeze-ds-m6a-01`. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **S6 spike reported** ([spk-04](sprint-spk-04.md)): `docs/v2/research/t6-realtime-interviewer.md` §16 is on `main` with a decision inside the D28 rules as spk-04 task 6 applies them (including the owner's pre-run answer on the sole-passer case, and the rows t6 leaves unmeasured by design) — a shell (or "text only, voice revisited on <date>"), the M7 deploy shape and the browser gate list — and `docs/v2/research/t6-s6-fixtures/` holds the "spend-limit path" quota fixtures. An escalated result (outside the rules) is **not** a pass: stop.
- [ ] [ds-m1-01](sprint-ds-m1-01.md) merged: `design-system/screens/v2/index.html` (canonical file names) and `board.css` (board chrome) are on `main`. This sprint never edits either.
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR edits ADR-0032, ADR-0007 or ADR-0024, or adds `design-system/screens/v2/AB13-*`, `AB24-*` or `AB25-*`.

_Informational, not gates:_ where the neighbouring boards are frozen on `main`, reuse their vocabulary and copy verbatim —
AB01 (F5 coach-locked banner, F13 keys panel), AB06 (public mock count), AB07 (workspace, editor, language picker), AB08 (Run
results), AB11 (badges), AB19/AB20 (consent tone; the privacy notice's host-image and pre-release-snapshot wording, since xLearn
keeps no backups per D12). Where one disagrees with this brief, list it under "Decisions to confirm".

## Goal

First, accept [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) with the S6 result, as the owner decided (BP2: "0032 at
S6, before the M6a design freeze"), so everything downstream cites an Accepted decision. Then draft the first half of the M6a
boards — **AB13** A9 Mock-v2, **AB24** interview setup + consent + pre-flight + $ cap and **AB25** the live text HUD — as static,
preview-only HTML on [`theme.css`](../../../design-system/theme.css) under `design-system/screens/v2/`, with every frame, every
state and final copy, so [m6a-05](sprint-m6a-05.md) builds against owner-approved designs. BP3: agents draft every board; the
owner only reviews.

The boards encode decisions that are easy to get subtly wrong on screen: **D29** (the AI follows the answer widgets as
structured state, never the learner), **D27** (item selection skips live items; no start refusal; coach locked while live),
**D14 as amended by T6** (mock proposals need an explicit accept — the 24 h auto-submit doesn't apply), **D31** (the public profile
shows the mock count only), the **L19** caps, and the **never-assessed list** (voice, face, accent, emotion). One contrast to keep
straight: interview costs are shown **in dollars** because they're billed to the learner's own key
([t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)); ADR-0031 §5's "never dollars" covers only the platform
allowance (AB18).

## Scope

**In**
- ADR-0032 → **Accepted**, folding the S6 result into it and its amendments into ADR-0007 and ADR-0024 (task 1, own docs PR).
- `AB13-mock-v2.html`, `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html` under `design-system/screens/v2/` (the
  ds-m1-01 index names), every frame in tasks 3–5, each with a behaviour-notes aside citing its decision.
- PNG screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/`, and the design PR, stopped for review.

**Out**
- Any `web/` code → [m6a-05](sprint-m6a-05.md). Backend states behind the frames → [m6a-01](sprint-m6a-01.md) (FSM, clock, caps,
  consent rows, retention), [m6a-02](sprint-m6a-02.md) (brain, classifier, probe, `give_hint`), [m6a-03](sprint-m6a-03.md)
  (assessment deltas, proposal/accept, `ScoreMock` once), [m6a-04](sprint-m6a-04.md) (mock Run budgeting, CodeMirror interview mode, bounded SSE).
- Grace, paused, resume, debrief, proposal and accessibility settings → AB26–AB28 in [ds-m6a-02](sprint-ds-m6a-02.md).
- Voice pre-flight, browser and EU notices, the voice HUD → AB29, AB30★ in [ds-m6b-01](sprint-ds-m6b-01.md). Voice appears here
  only as a labelled "from M6b" variant.
- Editing `index.html` or `board.css` → [ds-m1-01](sprint-ds-m1-01.md) (written once). Redrawing AB01, AB06, AB07 or AB08 — reference them.
- Shipping boards (preview-only, never imported by `web/`) and owner design hours beyond review (BP3).

## Tasks

### 1 · Accept ADR-0032 with the S6 result [X] — own docs PR, merged

Branch `docs/adr-0032-accept` off an up-to-date `origin/main`. Sources: the S6 note ([t6 §16](../research/t6-realtime-interviewer.md),
added by spk-04), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (amendments), BP2.
- **`docs/adr/0032-realtime-ai-mock-interviewer.md`:** Status → **Accepted (<date>)**, "S6 ran <date> — see t6 §16". Fold the
  result into the sections it changes: §2 *Shell choice* gets the outcome and the measured reason (e.g. "GPT-Live-1: every hard gate
  passed; naturalness 4 vs 2"), and the deploy shape from M7 ("one coach Deployment" or "a `coach-interview` Deployment, same
  image, own ImagePolicy, in P1"); §4 *Error typing* notes that the S6 quota fixtures cover the **spend-limit path** only and the
  probe arbitrates prepaid exhaustion; §6 *Availability* carries the browser gate list and §6 *Cost* the measured figures and
  D29's cadence cost; *Links* adds t6 §16 and the fixtures directory. If S6 found **both shells failing**, the P1 row reads
  "deferred: voice revisited by <date + 3 months>" and the Consequences say so. If the D34 note isn't there yet, record that T6's
  opscheck interview counters are superseded (the release checklist's "no live interviews" check stays).
- **ADR-0007:** turn the header's "Amended by ADR-0032 (v2, Proposed)" into "Accepted <date>" and add a short *Amendment
  (ADR-0032)* section with its §7 text: SDP brokering only, each segment creation logged; the Realtime `client_secrets` fallback
  (30–60 s TTL) **only if S6 showed brokering failing**; the key held as a `[]byte` per live segment, dropped on `interrupted` or
  `paused`, never in panic values or logs.
- **ADR-0024:** the header's ADR-0032 note (mock count only, D31) → Accepted <date>. Leave the ADR-0033 part as it is.
- **`docs/adr/README.md`** index row and the [feasibility](../feasibility.md#t6--realtime-ai-mock-interviewer) topic-map row and T6 header → Accepted.
- **`docs/v2/status.md`:** the ADR row (0032 Accepted, date, PR), a Decisions log line ("ADR-0032 accepted: shell X, deploy shape Y,
  browsers Z"), this sprint's Sprint-board row 🔄. This sprint file: task 1 ✅, _Overall_ 🔄.
- Commit `docs(adr): accept ADR-0032 with the S6 result`; PR; **merge once CI is green** (land-and-sync; BP2 is the owner's
  pre-authorisation); `git checkout main && git pull`.
- **Stop rule:** if the S6 note is escalated or outside the D28 rules, don't accept. Leave ADR-0032 Proposed, add a Decisions log
  line saying why, and stop and report to the owner; the boards wait.

### 2 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named per the ds-m1-01 index: `AB13-mock-v2.html`,
`AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html` (if `index.html` links different names, use those). Branch
`design/ds-m6a-01` off `main` **after** task 1 merged.
- Each links `../../theme.css`, then `board.css`, plus the Google Fonts `<link>` (Inter, JetBrains Mono), and reuses the tokens and
  components **verbatim** (dark "landscape console"; [`design-system/README.md`](../../../design-system/README.md)): `ds-card`,
  `ds-badge--ok|warn|err|info|violet`, `ds-chip`, `ds-btn--primary|secondary|ghost`, `ds-seg`, `ds-toggle`, `ds-meter`, `ds-modal--sm|md`,
  `ds-field`/`ds-input`, `ds-spin`, `xl-panel`, `xl-code`, `xl-lock`, `xl-timer`, `xl-kbd`, `xl-coach`/`xl-fab`, `xl-topbar`/`xl-crumb`.
  Difficulty: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`. Mono for clocks, money, sizes and ids. Board-only layout in
  the file's `<style>`, tokens only, never overriding a `ds-*`/`xl-*` rule.
- **Frames:** full-screen frames stack; panel- and modal-sized states sit side by side in a `.bd-grid`. Each frame has a label
  `ABnn-Fk · <state>`, its decision refs and **final copy** (numbers are example data; no lorem ipsum, no "TBD"). Frames that ship
  later carry a **"from M6b"** tag.
- **Behaviour-notes aside** per frame: the server field or route that drives it, timers, gates, exact error codes (codes this brief
  marks *proposed* are flagged in the PR for m6a-01 to confirm), the decision it implements, a11y (focus order, contrast ≥ 4.5:1,
  keyboard, `aria-live` policy, reduced motion) and the `< 1024 px` layout. Every board ends with a `.bd-narrow` 390 px section.
- Static HTML only: no JS runtime, no `support.js`, no `.dc.html` canvas markup; v1 `Mock.dc.html` is a reference only.
- **Never leak withheld data** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) S1/S4): no pattern chip on a mock item
  while it's live; no hidden test inputs, only aggregate evidence; no hint text before `give_hint` releases it; no reference solution;
  no private rubric anchors (public descriptors only); **no score anywhere live**, and no mock score on any public surface (D31).
- **Touch only this sprint's three board files and their screenshots.**

### 3 · AB13 A9 Mock-v2 [X]

`AB13-mock-v2.html` — the course's Mock page in v2: the format choice, the **classic** mock with server-picked items and the
**evidence | sliders** scoring that the AI proposal review (AB27) also reuses. Replaces v1 `Mock.dc.html` / `Mock.tsx`
(`PROBLEM_SETS`, `Mock.tsx:19-22`). Consumed by [m6a-05](sprint-m6a-05.md) task 3 on top of [m6a-03](sprint-m6a-03.md)
(`mock_session.status ∈ {open, scored, incomplete, abandoned}`, `format ∈ {classic, text, voice}`, `scored_by`,
`time_multiplier`, `caveats[]`, the rubric snapshot, `mock_session_item`). Sources: [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)
(A9: "server-picked setup · multi-item · evidence + rubric"; "two-column scoring (evidence | sliders), 'Scores lock when saved'"),
[t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena) and [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock)
(evidence only; `evidence_hints` as suggested caps), [ADR-0026](../../adr/0026-per-course-extensibility-model.md) (rail, rubric,
targets, item pools per course; a course may have no mock), [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) and
§11 (T1 row: aggregates exclude everything but `scored`), D27, D31, L16 ([m3-09](sprint-m3-09.md) task 6).

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Mock home** (DSA), format choice | Eyebrow "Mock interview · DSA"; title **"Mock interview"**; cards: **Classic** — "45 minutes, one problem, phase-timed. You score yourself on 7 dimensions." [Set up]; **AI interviewer · text** — "An AI interviews you by chat and follows your code as you write. Runs on your own key (about $0.45)." [Set up →] (AB24). Side card "Scored on 7 dimensions" (the public descriptors) and **"Only you see your scores. Your public profile shows how many mocks you've done — never the scores."** Variant **F1b "from M6b"**: a third card **AI interviewer · voice** — "Talk it through out loud. Needs an OpenAI key and Chrome or Edge." In M6a the voice card is absent, not greyed | D31; R-MI1; ADR-0032 §1 |
| F2 | **Classic setup, server-picked** | "We pick the problem." `ds-seg` Difficulty [Easy] [Medium] [Hard] [Mixed]; "From the weeks you've unlocked. We skip problems you have open or due for review."; Time [1×] [1.25×] [1.5×] [2×] (default from AB28) with "Extra time is shown on your result and never lowers your score."; the rail preview scaled by the multiplier (0–5 Clarify the problem · 5–10 Brute force out loud · 10–18 Key observation → plan · 18–33 Code it · 33–40 Trace + edge cases · 40–45 Complexity + follow-ups); "You'll see the problem when the clock starts."; "Mocks don't change your revision schedule or open mistakes."; [Start 45-minute mock] | D27 (skip, no refusal); t0 (mock rail per course; mocks never schedule, open no mistakes); WCAG 2.2.1 |
| F3 | Classic live, one item (DSA) | v1 chrome parity: phase rail with the server clock **"21:40 / 45:00"** and the current phase goal; the public statement; the editor in **interview mode** (no autocomplete; Go · C++ · Python); [Run] "sample tests · 1 Run per 10 s"; notes; the coach FAB showing AB01 F5's locked banner; [Finish mock]. **No Submit**: the final check runs at Finish. No pattern chip | t6 §5 (interview mode, learner-started Runs); D27 (coach locked); L19 (Run ≤ 1 / 10 s) |
| F4 | Classic live, **multi-item** (illustrative course) | Item tabs "1 · Medium · edited" · "2 · Easy · not started", "Item 2 of 2", one shared rail and clock; label "Multi-item mocks come from the course manifest; DSA picks one problem." | ADR-0026 (ordered item list per session) |
| F5 | Finishing | Per item: "Item 1 · running hidden tests" (`ds-spin`); "Usually under a minute. Your clock has stopped."; judge-unavailable variant: "We couldn't check your code. You can still score this mock — the evidence shows 'not checked'." | t4 §6.7; AB11 vocabulary |
| F6 | **Scoring: evidence \| sliders** | Left **Evidence** per item: "Hidden tests · 37 / 40 passed" (aggregate only), "First failure · too slow on large inputs", "Runs · 4 · last: 7 / 9 sample passed", "Hints · none", phase times vs rail ("Code it · 17:10 of 15:00"), your notes; `evidence_hints` chip **"Suggested: Optimisation ≤ 2 — only performance cases failed"** (shown, not enforced). Right **Sliders**: the rubric snapshot's dimensions with public descriptors — Communication · Problem understanding · Brute force · Optimisation · Code quality · Edge cases · Complexity — 1–5 each; running total **"27 / 35"**; "What to fix next time (optional)"; banner **"Scores lock when saved."**; [Save scores] | t4 §8 (A9 scoring); t4 §6.7 (`evidence_hints` suggested only); ADR-0027 §1 (no hidden inputs) |
| F7 | Save confirm (`ds-modal--sm`) | "Save 27 / 35? Scores lock when saved — you can't change them later." [Save scores] [Keep editing] (default focus) | score-once (ADR-0029, `ScoreMock`) |
| F8 | Saved, locked | Sliders locked (`xl-lock`), total, chips `self` · `honor` (the AB27 variant reads `AI-proposed · accepted` · `honor`), "1.5× time" chip when used, the trend vs the course target ("Week-13 target: 24 / 35"); "Your public profile now shows 5 mocks." | D31; t6 §6 (`trust=honor` always shown) |
| F9 | **Mock history** | Rows: date, format chip (Classic · Text · Voice), item count, status — **Scored · 27 / 35** · **Scores waiting** (→ AB27) · **Paused — resume by Thu 14:05** (→ AB26) · **Incomplete · not scored** · **Abandoned**; multiplier chip; footer "Only scored mocks count in your trend." | t6 §11 (T1 row: aggregates exclude all but `scored`); D30 |
| F10 | Limits and empty states | `409 mock_in_progress` → "You have a mock in progress — taking you back to it."; `429 mock_daily_limit` → "You've started 3 mocks today. Try again after 05:30." (both L16, m3-09); `409 no_eligible_items` (*proposed*) → "Every problem we could pick is open or due for review right now. Finish those first, then start a mock."; a course with no mock → "This course has no mock interviews." | L16; D27; ADR-0026 |
| F11 | N-dimension rubric (illustrative non-DSA course) | Five dimensions from another manifest, total "/ 25": the dimensions and `max_total` are data | R-MK2 (generalised); ADR-0026 |
| F12 | < 1024 px / 390 px | Format cards stacked; live: Problem · Code · Notes as tabs, the rail condensed to "Phase 4 of 6 · Code it · 21:40"; scoring: evidence as an accordion above the sliders, the total sticky | — |

Behaviour notes must state: the classic clock is assessment's `deadline_at` (the HUD mirrors it; text/voice use coach's clock, AB25);
the rubric comes from the **session's snapshot**, never the live manifest; Save calls `ScoreMock` once (a second save is refused);
evidence is aggregate only and never includes hidden inputs; `evidence_hints` never clamp a slider; no mock opens a mistake or
schedules a revision; the public route shows the count of **scored** mocks only.

### 4 · AB24 setup + consent + pre-flight + $ cap [X]

`AB24-mock-setup-preflight.html` — everything between choosing "AI interviewer · text" and the first interviewer message. Consumed
by [m6a-05](sprint-m6a-05.md) task 4 on top of [m6a-01](sprint-m6a-01.md) (L19 caps, `interview_consent`, the manifest's
`mock.interviewer`), [m6a-02](sprint-m6a-02.md) (the classifier and probe) and [m1-10](sprint-m1-10.md) (the `interview` default key,
the catalog, `interview_brain`). Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (limits and
onboarding advice), [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (classifier:
`ErrModelAccess`, `ErrAuth`, `ErrQuota`, the probe), [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)
(consent strings, the multiplier), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (estimate copy, pre-flight
checks), [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits), [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes)
(names), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L19, the S6 note (measured costs).

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Account opt-in** (first use; the Settings twin is AB28 F1, same strings) | Unticked `ds-toggle` **"Let an AI interviewer run mock interviews with me"**; card "How it works and what it evaluates": "It interviews you by chat, follows your code as you write it and ends with improvement areas. It runs on your own AI key — your provider bills you directly. It evaluates what you say and the code you write. It never evaluates your voice, face, accent or appearance."; [Continue] disabled until on | t6 §7 (account-level opt-in); R-MI2 |
| F2 | **Setup** | Format [Text] (the "from M6b" variant adds [Voice] [Voice with push-to-talk]); Length [45 min] [60 min] (manifest `max_minutes` ≤ 60); Time [1×] [1.25×] [1.5×] [2×] with the cost delta; Difficulty as AB13 F2; key row **"Your AI coach (your key) · OpenAI · gpt-6-sol"** [Change in Settings]; persona line from the manifest ("Your interviewer runs a 45-minute DSA round like a senior engineer would.") | ADR-0031 §7; t6 §9 P0 (`mock.interviewer`); WCAG 2.2.1 |
| F3 | **Consent** (per session, all unticked, all required) | ☐ "My transcript and code are sent to my AI provider's text API for the interviewer, summaries and feedback." ☐ "The interviewer is an AI. It can't see you." ☐ "Assessment uses what I say and my code — never my voice, face, accent or appearance." Retention (radio, first is default): "Delete my transcript 30 days after I score" / "Keep it for 12 months"; footer "You can delete the whole transcript, or any single message, at any time. Once deleted, a weekly server image kept by our host may hold it for up to about 7 days, and a pre-release snapshot for up to 1 day." This is AB20 F1 §6's wording. Never call it "backups": D12, and AB20's erase card says "xLearn keeps no off-node backups". If AB20 is frozen with different words, copy AB20's. "consent v1" | t6 §7 (per-session consent, retention, backup lag disclosed); ADR-0032 §6; D12 (no off-node backups) as worded on AB20 ([ds-l-01](sprint-ds-l-01.md)) |
| F4 | **Pre-flight checks** (text) | Checklist, each row with state: "Interview key is on ✓"; "Model access · gpt-6-sol ✓ — checked with a tiny call (under $0.001)"; "Credit ✓"; "Text mode works in any modern browser ✓". Running state with `ds-spin`; all green → [Start interview] enabled | t6 §4 (probe: 16-token call on the brain model); §8 (checks) |
| F5 | **Pre-flight failures** | `ErrModelAccess`: "This key can't use gpt-6-sol. Pick another model in Settings." (no retry; the key stays on) · `ErrAuth`: "Your provider rejected this key, so we turned it off. Add or fix a key in Settings." · `ErrQuota`: "Your OpenAI credits look empty. [Top up ↗] and check again." [Check again] (the key is never turned off) · custom model id: "Custom models can run the interviewer, but can't propose scores — you'll score this one yourself." (allowed) · no interview key: "Set a key for mock interviews: Settings → Your AI coach (your key) → Mock interviews." | t6 §4 classifier (`ErrQuota` never disables; `ErrModelAccess` never retries); t6 §6 (custom ids: no proposal); m1-10 (key defaults) |
| F6 | **Estimate + mandatory $ cap** | "Estimated cost on your key: **typical $0.45 · plan for $0.60** · + tax where applicable (India: 18% GST) · billed by OpenAI to your key."; "1.5× time adds about $0.15."; "Extras, only if you use them: resume summary ~$0.04 · second opinion on scores ~$0.06."; **Spending cap** field `$ 0.90` (default = plan-for × 1.5), required: "At 85% of your cap the interviewer wraps up. At 100% the interview ends — you can still score it yourself."; "OpenAI deducts any negative balance first: keep at least $1.00 available, or turn on auto-recharge."; tip card "Use a separate OpenAI project for xLearn with a hard spend limit, a key limited to the APIs it needs, and an expiry date." | t6 §3 (mandatory cap; onboarding advice); §8 (estimate copy, "plan for"); ADR-0032 §6; S6 M9 (measured) |
| F7 | **Caps (L19)** | `409 interview_in_progress` (*proposed*): "You already have an interview in progress. Resume it or end it before starting another." [Go to interview] · `429 interview_daily_limit` (*proposed*): "You've started 2 AI interviews today. You can start another after 05:30." · `422 cost_cap_required` (*proposed*): "Set a spending cap to start." | L19 (1 non-terminal, ≤ 2 starts/day, mandatory cap) |
| F8 | **Starting** | "Picking your problem — we skip problems you have open or due for review." → "Your AI coach is paused while the interview is live." → AB25 F1 | D27; t6 §4 (coach locked from `connecting`) |
| F9 | Anthropic key (what differs from F2–F6) | Format shows Text only, with "Voice interviews need an OpenAI key." (from M6b). The key row reads "Your AI coach (your key) · Anthropic · <model>", and F4's model-access row names that model. F5 `ErrQuota` reads "Your Anthropic credits look empty. [Top up ↗] and check again.", with the link going to the catalog's Anthropic `billing_url`. F6's estimate reads "billed by Anthropic to your key", and **the OpenAI negative-balance line is dropped**. The tip card reads "Use an API key only xLearn uses, and set a spend limit for it in the Anthropic Console." The consent, the caps, the $ cap field and the starting frames are unchanged | t6 §2 (no Anthropic audio API); t6 §4 (Anthropic `ErrQuota` codes); ADR-0032 §6 |
| F10 | < 1024 px / 390 px | One column with steps "1 Setup · 2 Consent · 3 Checks · 4 Cost"; a sticky [Start interview] showing why it's disabled ("Tick all three boxes to start") | — |

Behaviour notes must state: the probe is one 16-token call (≈ $0.0001), at most 1 per 10 s; errors are classified by `error.code`,
never HTTP status; `ErrQuota` never disables a key, `ErrAuth` does (ADR-0007); the cap is stored per interview and enforced
server-side (wrap at 85%, `finished(cut_short=cap)` at 100%); the consent version and timestamp are stored in `interview_consent`;
item selection skips withheld-live items and never refuses the start (D27); the live meter is hidden during the interview (shown in
grace, pause, resume and the report — AB26/AB27); [Start interview] states its disabled reason in text, not only by greying.

### 5 · AB25 live HUD (text) [X]

`AB25-live-hud-text.html` — the text interview while it's live. Consumed by [m6a-05](sprint-m6a-05.md) task 5 on top of
[m6a-01](sprint-m6a-01.md) (FSM, server clock, rail, caps, lease), [m6a-02](sprint-m6a-02.md) (brain loop, `give_hint`, widget
snapshots) and [m6a-04](sprint-m6a-04.md) (mock Runs, CodeMirror interview mode, bounded SSE). Sources:
[t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (routes: `turns`, `code`, `run`, `hint`, `hold`, `pause`,
`finish`, `abandon`; bounded SSE ≤ 50 s with `Last-Event-ID`), [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)
(clock, idle, second tab, coach lock, 85% cap), [§5](../research/t6-realtime-interviewer.md#5-the-coding-round) (snapshots, Runs,
hints, rail, quiet during Code), [§6](../research/t6-realtime-interviewer.md#6-assessment) (never assessed; distress on words),
[§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (S1–S4, conduct, a11y),
[ADR-0032 §3–§4](../../adr/0032-realtime-ai-mock-interviewer.md#3-what-the-ai-perceives-a-second-interviewer-sharing-the-session-owner-d29) (D29), L19.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Live · Clarify** | Top: the 6-phase rail (ranges × multiplier), current **"Clarify the problem · 0–5"** with its public goal "Ask questions until the problem is unambiguous."; server clock **"03:12 / 45:00"** (mono); chips "Text interview" · "1.5× time" when on. Left: the public statement. Centre: the transcript; first message, labelled "Interviewer (AI)": **"Hi, I'm an AI interviewer. I can't see or hear you — I follow your messages and your editor. Let's start: how would you approach this problem?"**; input "Type your answer… Enter to send · Shift+Enter for a new line". Right: the editor (interview mode) + [Run]. Bottom bar: [Ask for a hint] [Hold] [Finish] and ⋯ → "Save & resume later" · "End interview" | ADR-0032 §6 (AI disclosure); R-MI2; t6 §5 (rail) |
| F2 | **Code phase, snapshot indicator** | Under the editor: **"Shared with the interviewer · 2 s ago"** with the tooltip "The interviewer sees your code as text — not your screen, not you."; a size meter "58 / 64 KB" from 56 KB; over the cap: "Your code is over 64 KB, so the interviewer still sees your last version under the limit." | D29 (structured state, 2–3 s on change); L19 (≤ 1 snapshot / 2 s); L6 (64 KiB) |
| F3 | Run result | Strip "Run · 7 / 9 sample tests passed · first failure: wrong answer on case 4"; "Next Run in 7 s"; "The interviewer sees the same result you do." | t6 §5 (learner-started Runs; the model has no run tool); S1 |
| F4 | **Hint** (`give_hint`) | Popover "Ask for a hint? Hints are recorded, and the review takes them into account." [Get hint] [Cancel]; hint card "Hint 1 · conceptual — What do you need to remember about the elements you've already seen?" chip `recorded`; the interviewer-offered variant "The interviewer offered a hint"; the limit: "No more hints in this phase." | t6 §5 (hints only via `give_hint`, recorded first, ≤ 1 conceptual per phase, never code); S4 |
| F5 | Phase change | Nudge "Next: Code it (18:00). Write a working solution and explain your choices as you go." | t6 §5 (server-timer cues); a11y: `aria-live="polite"` |
| F6 | AI reply streaming; transient errors | "Interviewer is typing…"; `ErrTransient`: "Reconnecting to your provider (try 2 of 2)…"; `ErrRate`: "Your provider is rate-limiting this key — retrying in 20 s." After the tries → AB26 (grace or interrupted) | t6 §4 classifier |
| F7 | **Hold** | Chip "Held"; "The interviewer stays quiet until you message it or press Talk. The clock keeps running." [Talk] | t6 §4 (`held`: clock runs); text-mode reading flagged in the PR |
| F8 | Quiet during Code | After 90 s × multiplier with no edits and no message: "Take your time. Want to talk through where you're stuck?" — the only unprompted kind of message in Code | t6 §5 (quiet during Code) |
| F9 | **Distress card** | "It sounds like things might be hard right now. You can pause the interview at any time. If you need support, call Tele-MANAS on 14416 (India) or your local emergency number." [Pause interview] [Keep going] | t6 §6 (triggered on words only) |
| F10 | $ cap wrap-up | At 85%: `--ds-warn` banner "You're close to the spending cap you set, so the interviewer is wrapping up." At 100%: "Your spending cap was reached, so the interview has ended. You can still score it." → AB27 | t6 §3 (cap); ADR-0032 §6 |
| F11 | Finish confirm (`ds-modal--sm`) | "Finish the interview? You'll get a short debrief (up to 3 minutes), then your code gets a final check." [Finish] [Keep going] (default focus) | t6 §4 (`wrapping`); §6 (debrief) |
| F12 | Idle check-in | After 3 min × multiplier with no message, no edit and the tab hidden, the interview goes to `interrupted(idle)`: the clock freezes at the last activity and the provider session is hung up. On return the learner sees "Still there? We stopped your clock while you were away. [I'm back]"; [I'm back] re-primes for free. No countdown and no "pauses in 60 s": t6 has no pre-warning. The 5-minute window and "Save & resume later" follow AB26 F4's pattern ("…until 14:10") | t6 §4 (idle in any phase → `interrupted(idle)`, hang up; free re-prime) |
| F13 | Live in another tab | "This interview is open in another tab." [Use this tab] | t6 §4 (client lease) |
| F14 | Coach locked | The app shell's coach panel shows AB01 F5: "The coach is paused while a revision touch or mock is live. It's back when you finish." | D27; t6 §4 (locked except `paused`) |
| F15 | < 1024 px / 390 px | Tabs Chat · Code · Problem; the rail collapsed to "Phase 4 / 6 · Code it · 21:40 / 45:00"; the bottom bar sticky; Run inside the Code tab | — |

Behaviour notes must state: coach is the single clock writer (`active_ms` + `live_since`; it runs only in `live`, `bridging`,
`wrapping`) and phases are active time against the rail × multiplier; the HUD follows a bounded SSE (≤ 50 s, reconnect with
`Last-Event-ID`); `PUT …/code {rev, text, cursor_line}` at most once per 2 s, ≤ 64 KiB; `POST …/run` at most once per 10 s;
`pagehide` sends `sendBeacon(…/interrupt)`; the interviewer's context holds only what the learner can see (S1) and hint text only
after `give_hint` (S4); no score or band appears live; the conduct rules (never asks about age, caste, religion, family, health,
disability, nationality or salary; never comments on appearance, voice, accent or emotion; never claims to be human) are listed
in the aside; the transcript is a `role="log"` region; phase changes are announced politely and the grace warning (AB26)
assertively; reduced motion shows replies whole instead of streaming.

### 6 · Self-review against the brief + screenshots [X]

- Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) M6a row
  ("AB13 A9 Mock-v2 · AB24 setup + consent + pre-flight + $ cap · AB25 live HUD (text)").
- **Decision checks:** D29 (the AI follows the widgets, not the learner; no screenshots), D27 (skip live items, no refusal; coach
  locked while live), D31 (no public score anywhere), T6's D14 amendment (nothing about mocks auto-accepts), L19 (caps and codes),
  the never-assessed list, D34 (no copy implying anyone is alerted).
- **Leak check:** no pattern chip on the live item; aggregate evidence only; no hint text before release; no reference solution;
  public descriptors only.
- **Copy check against neighbours** on `main`: AB01 F5 and F13, AB06, AB07, AB08, AB11, AB19/AB20. Mismatches → "Decisions to confirm".
  Known in advance, so don't copy them:
  - AB01 F13's "(needs a realtime-capable model)" is superseded by t6 §11 (T5 §9 row);
  - "backups" wording is superseded by AB20's host-image and snapshot wording (D12).
- Screenshot each board **full-page** at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB13@1440.png`,
  `AB13@390.png`, … `AB25@390.png` (each ≲ 500 KB). A plain headless `--screenshot --window-size=1440,900` captures only the
  viewport, and the Browser pane can't write files. Use a full-page capture: `npx playwright screenshot --full-page`, or a
  tall headless window sized to the board's `scrollHeight` (the prompt has the commands).

### 7 · Open the design PR and STOP [X]

Branch `design/ds-m6a-01`; conventional commit `docs(design): AB13 AB24 AB25 — M6a boards, part 1` ending with the attribution lines.
Open a PR titled `docs(design): AB13 AB24 AB25 — M6a boards (part 1)` with each board's frame list, the screenshots embedded
(`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m6a-01/design-system/screens/v2/shots/<file>?raw=true`), a link to the
merged ADR PR, and **"Decisions to confirm"**, at least:
- the *proposed* error codes (`interview_in_progress`, `interview_daily_limit`, `cost_cap_required`, `no_eligible_items`) for m6a-01;
- **Hold in text mode** = the interviewer stays quiet, clock running (t6 defines "hold" for voice);
- the voice card **absent** (not greyed) in M6a;
- the $ cap default (plan-for × 1.5) and the example estimate figures (t6 §8 vs the S6 measurement);
- the public count = scored mocks only;
- AB24 F3's deletion-lag line ("a weekly server image kept by our host … up to about 7 days … a pre-release snapshot for up to 1
  day") against AB20's frozen notice. Never "backups" (D12);
- AB01 F13's per-feature default "Interview: <model> (needs a realtime-capable model)" is superseded by
  [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 row: "the 'must be realtime-capable' rule is
  dropped"; text needs `interview_brain`, voice needs `voice_shell`). AB24 F2's key row doesn't carry the old rule. The AB01 string
  is flagged for its owner, not copied;
- the Anthropic-key differences in AB24 F9 (Text only, no negative-balance line, the Anthropic billing link and tip).

**Do not merge.** Stop for owner review (event `ev-freeze-ds-m6a-01`). Requested changes go on the same branch.

### 8 · Freeze [O]

The owner reviews the boards and approves in chat or merges the PR. **Merging is the freeze.** An agent may merge only on the
owner's explicit approval in chat. [m6a-01](sprint-m6a-01.md) can't start until this freeze and ds-m6a-02's have landed.

## Acceptance criteria

- [ ] ADR-0032 is **Accepted** on `main` with the S6 result folded in (shell, deploy shape, browser list, spend-limit fixtures), ADR-0007 and ADR-0024 updated, via its own merged docs PR — or the stop rule was applied and recorded.
- [ ] Every frame listed for AB13 (F1–F12), AB24 (F1–F10) and AB25 (F1–F15) is present with final copy, its state, its decision refs and a behaviour-notes aside.
- [ ] No board leaks withheld data, shows a score live or publicly, or implies the AI perceives the learner; interview costs appear in dollars only as BYO estimates.
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied or overridden; only this sprint's three board files and their screenshots are added.
- [ ] PR open with 1440 px and 390 px screenshots and "Decisions to confirm"; **not merged by the agent**.

## Release

Two outputs. **The ADR PR** (ADR-0032, ADR-0007, ADR-0024, the ADR index, feasibility and `status.md`) merges in-session per
land-and-sync; it ships in no tag. **The board PR: stop for owner review (design).** Nothing deploys: boards are preview-only
files under `design-system/screens/v2/`, never imported by `web/`. The owner's merge is the freeze (`ev-freeze-ds-m6a-01`); with
ds-m6a-02's it gates [m6a-01](sprint-m6a-01.md), and [m6a-05](sprint-m6a-05.md) implements it. No tag, no infra PR.

## Definition of Done

ADR PR merged (or the stop rule recorded) · board PR open with AB13, AB24, AB25 and screenshots · every acceptance box ticked ·
this sprint's Status table updated (task 1 ✅ via the ADR PR; tasks 2–6 ✅ and task 7 🔄 in the board PR) · no `web/`, `internal/`,
`index.html`, `board.css` change, and no `status.md` change in the board PR · the agent stops at the open board PR. (Done-done —
✅ Overall — is set by [m6a-01](sprint-m6a-01.md) once the owner has merged.)

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D15/D16/D17/D18/D27/D29/D31, T6's D14 amendment): cite the decision per frame; where two sources disagree, follow the owner decision and list it under "Decisions to confirm".
- **Accepting on a weak S6 result.** An escalated or out-of-rule note isn't a GO; the stop rule protects the freeze from resting on an unmeasured shell.
- **"Both shells failed."** ADR-0032 is still accepted (D28 covers it) with P1 deferred; the voice variants here stay labelled "from M6b", and the owner decides v2.1.0's content.
- **Implying the AI watches the learner.** Every perception line says "your messages and your editor"; nothing says "sees you" (D29, R-MI2, EU AI Act Art. 5(1)(f)).
- **Dollar confusion with AB18.** BYO interview costs are dollars on the learner's own key; platform-AI allowance stays percentages. Keep the two apart in copy.
- **Promising alerts or auto-saves.** D34: nobody is notified; T6: no mock score is ever saved without the learner's click.
- **Codes invented here drifting from m6a-01.** Mark them *proposed* in the aside and the PR; m6a-01 fixes the final strings and m6a-05 builds to those.
