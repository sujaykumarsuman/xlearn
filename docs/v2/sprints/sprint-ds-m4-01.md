# Sprint ds-m4-01 — Design M4: AI suggestion/dispute ★, pointer notes, allowance + consents (AB16 ★, AB17, AB18)

> **Milestone:** M4 — platform AI (the boards M4 builds against) · **Track:** design (parallel; order 51)
> **Prereqs:** none (no prerequisite beyond `depends_on`, which is empty) · references, not gates: [ds-m1-01](sprint-ds-m1-01.md) (board index, `board.css`, AB01), [ds-m2-01](sprint-ds-m2-01.md) (AB04 touch result, AB05 "grades waiting" slot), [ds-m3-01](sprint-ds-m3-01.md) (AB08 results dock, AB11 degradation badges), [ds-m3-02](sprint-ds-m3-02.md) (AB12 provenance chips), [ds-l-01](sprint-ds-l-01.md) (AB19 acceptance consents)
> **Unblocks:** [m4-01](sprint-m4-01.md) (M4 entry gate: "AB16–AB18 frozen", [rollout §3](../rollout-plan.md#3-milestone-map)) · consumed by [m4-06](sprint-m4-06.md) (the M4 UI sprint builds every frame here); copy reused by [m4-04](sprint-m4-04.md) (error codes, states) and [m4-05](sprint-m4-05.md) (allowance shape, consent strings)
> **Release action:** **land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag, nothing deploys (launching the prompt is the owner's approval, [D40](../feasibility.md#decisions-log-newest-first); the owner may review after the merge, and any change to a frozen board is a follow-up design PR)
> **Calendar:** November 2026 (parallel with M3's UI sprints and P) · no owner event: the freeze happens at this sprint's merge (`ev-freeze-ds-m4-01` is automatic and needs no tick; the Artboards rows record the freeze), before [m4-01](sprint-m4-01.md) starts (December)
> **Execute with:** [`../prompts/prompt-ds-m4-01.md`](../prompts/prompt-ds-m4-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB16 ★ A3 AI suggestion / dispute (hero) | X | ⬜ |
| 3 | AB17 pointer notes + "correct, with improvements" | X | ⬜ |
| 4 | AB18 AI allowance meter + Settings AI consents | X | ⬜ |
| 5 | Self-review checklist (run before merging) + screenshots (1440 px, 390 px) | X | ⬜ |
| 6 | Open the design PR (screenshots, frame lists, the ticked self-review checklist, "Decisions to confirm") | X | ⬜ |
| 7 | Freeze: squash-merge on CI green (the merge is the freeze) → status → sync `main` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Land and sync ([D40](../feasibility.md#decisions-log-newest-first)):** the board PR
> merges on CI green and the merge is the freeze, so this sprint records its own close-out. Once the PR number is known, a last
> commit on the PR branch sets tasks 1–7 and _Overall_ ✅ here (task 7: "frozen: merged in PR #N, <date>") and, in
> [`../status.md`](../status.md), this sprint's Sprint-board row ✅, the Artboards rows AB16, AB17 and AB18 → ✅ "frozen (merged,
> PR #N, <date>)", and the Snapshot's artboard count (`ev-freeze-ds-m4-01` is automatic: no tick). If the
> merge slips to another day or fails after that commit, correct the rows in a follow-up docs PR merged the same way.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] none (no prerequisite beyond `depends_on`, which is empty)
- [ ] Parallel sessions: no open peer PR adds or edits `design-system/screens/v2/AB16-*`, `AB17-*` or `AB18-*`
      (`gh pr list --state open`, `git worktree list`, ListAgents)

_Informational, not gates:_ `design-system/screens/v2/index.html` and `board.css` are written once by
[ds-m1-01](sprint-ds-m1-01.md). If they are on `main`, link `board.css` and use the exact file names the index links; if they
are not, **do not create them** — use the file names in task 1 and a board-local `<style>` for chrome. Where the neighbouring
boards are already frozen on `main` (AB04, AB05, AB08, AB11, AB12, AB19), reuse their component vocabulary and copy
verbatim so M4 extends them rather than redrawing them; where they are not, draft conservatively and list the overlap under
"Decisions to confirm" in the PR.

## Goal

Draft the three M4 boards — the **A3 AI suggestion / dispute hero (AB16 ★)**, **pointer notes + "correct, with
improvements" (AB17)** and the **AI allowance meter + Settings AI consents (AB18)** — as static, preview-only HTML on
[`theme.css`](../../../design-system/theme.css) under `design-system/screens/v2/`, with every frame, every state and final
copy, so M4's build sprints ([m4-04](sprint-m4-04.md) … [m4-06](sprint-m4-06.md)) implement **frozen** designs and
[m4-01](sprint-m4-01.md) can open M4. Per **BP3** (owner decision 2026-09-24) the agent drafts every board, heroes included;
per [D40](../feasibility.md#decisions-log-newest-first) the board PR merges on CI green and the merge is the freeze — the owner
may review afterwards, and any change to a frozen board is a follow-up design PR. The rollout's "owner designs the heroes in
Claude Design" ([rollout §9](../rollout-plan.md#9-artboards-by-milestone)) is superseded by BP3.

The boards encode four decisions that are easy to get subtly wrong on screen: **D14** (AI results are suggestions: accept,
edit within the deterministic ceilings, one blind re-grade, 24 h auto-accept, manual always available), **D26** (pass review
= course passes + materially different touch passes, advisory, never touches grade or ladder), **ADR-0031 §4–§5** (two unticked
consents + a behavioral opt-in, withdrawable; the allowance is a **percentage, never dollars**) and **D34** (no alert
anywhere — degradation is an in-app badge the learner sees and the owner reads on demand).

## Scope

**In**
- `AB16-ai-suggestion-dispute.html`, `AB17-pointer-notes.html`, `AB18-ai-allowance-consents.html` under
  `design-system/screens/v2/`, one file per board, every frame in tasks 2–4.
- A behaviour-notes aside per frame citing its decision (D-number / ADR § / research §), with exact error codes, timers,
  gates and a11y notes; a `< 1024 px` intent per frame and a 390 px narrow section per board.
- PNG screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/` and in the PR body.
- The design PR, merged on CI green (the freeze), and this sprint's rows in `docs/v2/status.md` (Status note).

**Out**
- Any `web/` code → [m4-06](sprint-m4-06.md) (all three boards). Backend states behind the frames → [m4-02](sprint-m4-02.md)
  (caps, breaker, `ai_disabled`), [m4-03](sprint-m4-03.md) (analyzer, pointer notes), [m4-04](sprint-m4-04.md) (provisional,
  dispute, re-grade, claims), [m4-05](sprint-m4-05.md) (allowance, consents API).
- Editing `design-system/screens/v2/index.html` or `board.css` → owned by [ds-m1-01](sprint-ds-m1-01.md) (written once).
- The **acceptance-step** consents (onboarding step 0: 18+, notice, region, the two unticked AI consents) → AB19 ★ in
  [ds-l-01](sprint-ds-l-01.md); AB18 designs only their **Settings** twins and must use the same strings.
- The privacy notice page → AB20 in [ds-l-01](sprint-ds-l-01.md) (AB18 links to it).
- The BYO coach panel and key settings ("Your AI coach (your key)") → AB01 in [ds-m1-01](sprint-ds-m1-01.md); AB18 only shows the
  two names side by side.
- Results-dock and degradation-badge vocabulary → AB08/AB11 in [ds-m3-01](sprint-ds-m3-01.md); provenance chips on
  Week/Mistakes/Progress → AB12 in [ds-m3-02](sprint-ds-m3-02.md). AB16/AB18 add only the `ai` variants.
- Mock AI proposal (`ai-byo`, ScoreMock) → AB27 in [ds-m6a-02](sprint-ds-m6a-02.md) (v2.1).
- Shipping boards (preview-only, never imported by `web/`) and owner design hours (BP3; any owner review happens after the merge).

## Tasks

### 1 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named per the ds-m1-01 index:
`AB16-ai-suggestion-dispute.html`, `AB17-pointer-notes.html`, `AB18-ai-allowance-consents.html` (if `index.html` on `main` links
different names, use those).
- Each file links `../../theme.css` (then `board.css` if it is on `main`) and reuses its tokens/components **verbatim** — dark
  "landscape console"; see [`design-system/README.md`](../../../design-system/README.md): `ds-card`, `ds-badge--ok|warn|err|info|violet`,
  `ds-chip`, `ds-btn--primary|secondary|ghost`, `ds-seg`, `ds-toggle`, `ds-meter`/`ds-meter__fill`, `ds-modal--sm|md`,
  `ds-field`/`ds-input`, `ds-tabs`, `xl-panel`, `xl-code`, `xl-lock`, `xl-touch`, `xl-timer`, `xl-sect`, `xl-mut`. Difficulty tokens:
  Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`. Inter + JetBrains Mono via the Google Fonts `<link>`; mono for
  countdowns, percentages, line ranges and ids. Board-only layout CSS lives in the file's own `<style>` and uses `--ds-*`
  tokens only; it never overrides a `ds-*`/`xl-*` rule.
- **Frames:** full-screen frames stack; card- and modal-sized states sit side by side in a grid. Each frame carries a label
  `ABnn · Fk · <state>`, its decision refs (e.g. `D14 · t4 §3.5`) and **final copy** (no lorem ipsum, no "TBD").
- **Behaviour-notes aside** per frame: the server field that drives it, timers, gates, exact error codes, a11y (focus order,
  contrast ≥ 4.5:1 on text, keyboard, `aria-live` for countdowns and state changes, reduced motion) and the `< 1024 px` layout.
- Static HTML only: no JS runtime, no `support.js`, no `.dc.html` canvas markup; boards open from disk or any static server.
- **Never leak withheld data** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise)): no pattern chip, concepts or pointer notes while an item
  is live (open attempt, due or live touch); no hidden test inputs, expected outputs, rubric anchors, must-cover lists or
  exemplars; accepted probe answers appear **only after** the touch concluded.
- **Touch only this sprint's own board files** (and their screenshots, plus this sprint's own rows in `docs/v2/status.md`).
  `index.html` already has a row and link for AB16–AB18.

### 2 · AB16 ★ A3 AI suggestion / dispute (hero) [X]

`AB16-ai-suggestion-dispute.html` — how an AI-produced result becomes a **suggestion** the learner accepts, edits, disputes or
claims, and what happens when AI is unavailable. Consumed by [m4-06](sprint-m4-06.md) (task 1) on top of [m4-04](sprint-m4-04.md)
(practice `provisional`, 24 h auto-accept, bounded edits, re-grade, claims, `self_grade_pending`). Sources:
[ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D14),
[t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) (who may be
provisional, dispute, override, claim, timeout), [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers)
step 8 as amended by [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets) (gateway → practice
authorizes → judge re-grade), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) ("Provisional and
claim" copy), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 5 (independent
re-grade; no auto-accept when flagged), [t5 §2](../research/t5-platform-ai.md#2-two-tiers-when-each-key-is-used) ("Wait for xLearn AI"
vs "Grade it myself"), [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (precedence),
[PRD §5.6](../../prd/xlearn-v2-prd.md) R-AI1/R-AI2, D18 (ceilings).

**What is live in v2.0.** No v2.0 course item carries an `ai_rubric` step (DSA and go-concurrency are code/key graded; the
first rubric course is post-v2.0), so the rubric frames (**F1, F5–F12**) are drawn on an **illustrative rubric item** and
labelled "rubric items — contract built in M4, first used by a rubric course". F10–F12 belong there too: flagged
(`low_confidence`/`review_flag`), `inconclusive(budget_exhausted|ai_unavailable)` on a final and the AI-off `self_grade_pending`
header all come from `Score` finals ([m4-02](sprint-m4-02.md) task 3), and no v2.0 item calls `Score`. (M3's own manual picker
for a runner-inconclusive attempt stays m3-11's and doesn't carry F12's "xLearn AI is off" header.) The frames the owner meets in
v2.0 are **F2** (analyzer suggestion on a deterministic grade), **F3–F4** (honor-probe claim on a touch), **F13** (accepted /
auto-accepted), **F14** (grades waiting) and **F15** (provenance legend) — the set [m4-06](sprint-m4-06.md) task 1 builds.
Label each frame accordingly.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | **Provisional suggestion card** (rubric item) | Chip row **"Rough · xLearn AI · provisional"**; countdown **"auto-accepts in 23:41"** (mono, `xl-timer` style); per-criterion rows: band 1–5 as dots, the **verified learner quote** it rests on ("You wrote: '…'"), no rubric anchors; mistake category suggestion; ≤ 3 concepts; one-paragraph reasoning (≤ 480 chars); buttons **[Accept]** (primary) **[Edit…]** **[Dispute…]**; footnote "Accepting starts your revision schedule and opens a mistake entry if one applies." | D14; R-AI1; t4 §7 (evidence before band) |
| F2 | **Analyzer suggestion on a deterministic grade** (DSA course conclusion, below Clean) — **live in v2.0** | Grade row **"Rough · judge-checked"** with `xl-lock` "Code verdicts aren't editable"; block **"xLearn AI suggests"**: category chip **"Off-by-one / boundary"** + confidence word ("medium"), concepts `two-pointers` · `sorting`, a one-line summary and a next step ("Re-read: two pointers on sorted input"); **[Use this]** **[Change…]**; source note "Your choice always wins over a suggestion." No countdown (the grade is already final) | D14 ("mistake and notes always editable"); t4 §6.3 precedence learner > strong rule > analyzer ≥ medium > weak rule |
| F3 | **Touch result, honor-probe miss → provisional** (DSA re-solve) — **live in v2.0** | Criteria rows: "Pattern named · 'slidng window' ✗ (checked on your honour)", "Solved in time · 13:20 ✓ judge-checked", "Complexity · O(n) / O(1) ✓"; headline **"Not passed — unless you meant it"**; claim card: "Accepted answers: **Sliding window** · Two pointers (variable window)" **[I meant this]** (once per attempt) **[Keep as not passed]**; banner "Provisional · auto-accepts as not passed in 23:41" | t4 §3.5 item 4 (claim), item 1 (touch provisional iff flipping the honor criteria would pass it); AB04 F10 |
| F4 | Claim applied | **"Passed — advances to Day 21"**; chips `claimed` · `honor`; line "Counted on your honour — not included in your judge-checked %." | t4 §3.5 item 4; P7 ([rollout §10](../rollout-plan.md#10-public-dashboard-tasks)) |
| F5 | **Edit (bounded)** | `ds-seg` grade picker **[Miss] [Assisted] [Rough]** with **Clean** disabled + reason "Your time caps this attempt at Rough (submitted at 27:40, after the 20-minute Clean window)" — no hint, no coach; a second variant after a hint: **Clean** and **Rough** disabled, reason "Your hint caps this attempt at Assisted (hint opened at 17:02)" (coach help on that problem reads "Coach help on this problem caps this attempt at Assisted"); deterministic steps listed with `xl-lock` ("Hidden tests · 40/40 · locked"); editable mistake category (the 8 categories), notes (≤ 240), concepts; **[Save]** — a grade change goes through F9 | D14 (edits bounded by deterministic ceilings, labelled self-set); D18 ceilings (late or > 3 failed submits → Rough; hint → Assisted); D27 (coach → Assisted) |
| F6 | **Dispute modal** (`ds-modal--md`) | Title **"Ask for a re-grade"**; "One re-grade per attempt. An independent reviewer grades your answer again from scratch — the grade can go up or down."; tick the misjudged criteria (reason codes: "Missed evidence I gave" · "Misread my answer" · "Unfair to my approach" · "Other"); deterministic steps shown **locked**; optional text ≤ 500 chars with counter and helper **"Shown to the course author for calibration; the re-grade doesn't read it."**; **[Request re-grade]** **[Cancel]** (default focus) | t4 §3.5 item 2; t5 §7 item 5 |
| F7 | Re-grade running | Card state **"Re-grading · independent reviewer · usually under 2 minutes"** (`ds-spin`); countdown hidden with "Your 24 h window restarts when the re-grade lands."; polling note in the aside (`poll_after_ms`) | t4 §3.5 item 2 (new 24 h window) |
| F8 | **Re-grade compare** | Two columns **"First grade · Rough"** / **"Re-grade · Assisted"**, per-criterion bands side by side with changes marked; new countdown "auto-accepts in 23:59"; **[Accept re-grade]** (primary) **[Set my own grade…]**; "Dispute used" chip | t4 §3.5 items 2–3; t4 §8 |
| F9 | **Override confirm** (`ds-modal--sm`) | "Set your own grade? It will be labelled **self-set** everywhere and **won't count toward your judge-checked %**. You can choose between Miss and Rough — your time caps this attempt at Rough." **[Set grade]** **[Keep the AI grade]** (default focus) | D14; t4 §3.5 item 3; P7; D18 (same ceiling as F5) |
| F10 | **Flagged — no auto-accept** | Two variants. `low_confidence`: banner "The reviewer wasn't sure about this one, so it won't be accepted automatically. You can accept it, edit it or ask for a re-grade." `review_flag`: chip `honor` + "We couldn't verify this grade automatically, so it counts on your honour (not judge-checked) and won't be accepted automatically." — never names the heuristic; no countdown in either | t5 §7 item 5; t4 §7 and §11.4 item 17 (`review_flag` → `trust=honor`, not `self-set`) |
| F11 | **AI unavailable on a final** | "xLearn AI can't grade this right now." Two choices: **[Wait for xLearn AI]** ("resets Dec 1" for `budget_exhausted` · "usually back within the hour" for `ai_unavailable`) and **[Grade it myself]**; touch variant: "This review falls back to self-scoring." | D14; t5 §2 (Score row); R-AI2 |
| F12 | **Manual entry** (`self_grade_pending`) | The bounded grade picker + mistake picker + notes, headed "Graded by you — xLearn AI is off for this one"; label `self-set`; no countdown | D14; t4 §2.4 step 6 |
| F13 | Accepted / auto-accepted | "Rough · **xLearn AI** · accepted" and the variant "accepted automatically after 24 h"; follow-ups: "Mistake entry opened" link, "Day 1 review: Thu, Dec 10" | D14 ("submitting triggers the downstream actions"); t4 §3.5 item 6 (`anchor_at`) |
| F14 | **Today "grades waiting"** (full fidelity of AB05 F8) | Card "**2 grades waiting** · the first auto-accepts in 3:12:40" with rows (item, grade, countdown) and **[Review]** | D14; AB05 F8 |
| F15 | Provenance legend | The chip set used in P7 views and results: `judge-checked` · `ai` · `self-set` · `claimed` · `honor` — one line each on what it means and whether it counts toward judge-checked % | P7 (M4 adds `ai`); t4 §11.4 item 17 |
| F16 | < 1024 px / 390 px | F1 stacked (criteria as an accordion); F6 and F9 as full-screen sheets; F8 as two tabs "First grade / Re-grade" | t4 §8 shell |

Behaviour notes must state: the countdown renders from the **server deadline** (`accept_deadline_at`), never a client clock;
`aria-live="polite"` announces it at 1 h and 5 min left only; one dispute per attempt (`409 dispute_used`), dispute only while
provisional (`409 not_provisional`), a re-grade that can't run (`503 ai_unavailable`) offers F11/F12; the dispute text is stored
for calibration and never sent to the model; **no auto-accept when flagged** (F10); code verdicts are never editable; edits
never exceed the ceiling; closes/re-grades are exempt from quotas; Esc closes modals; destructive or labelled choices are never
the default button.

### 3 · AB17 pointer notes + "correct, with improvements" [X]

`AB17-pointer-notes.html` — the D16/D26 pass review. Consumed by [m4-06](sprint-m4-06.md) (task 2) on top of
[m4-03](sprint-m4-03.md) (analyzer on passes, `evaluation_analyzed`, notes stored per (account, item) in judge). Sources:
[ADR-0031 §6](../../adr/0031-platform-ai-and-two-tier-keys.md#6-scope-of-ai-review-of-passing-solutions-owner-d26-refines-d16) (D26),
[ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (pass review, optional
revisit), [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 2
(`pass_review{pointer_notes[≤3]{note, lines|null}, revisit{suggested, reason|null}}`), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)
(notes are C3, learner-private, withheld during live touches), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2)
(pointer notes collapsed during a live touch), [PRD §5.6](../../prd/xlearn-v2-prd.md) R-AI3 (refined).

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | Course pass, notes **collapsed** (default) | Concluded card "**Clean** · judge-checked"; chip **"correct, with improvements"**; a collapsed row "xLearn AI · **2 improvement notes**" (chevron, `aria-expanded="false"`) | D26; t5 §9 (collapsed by default) |
| F2 | Notes expanded | ≤ 3 notes, each ≤ 240 chars with a line-range tag **"L12–18"** that highlights those lines in a read-only excerpt of **the learner's own submission** (`xl-code`, ≤ 8 lines shown); "Optional revisit suggested: try it with O(1) extra space." **[Add optional revisit]** **[Not now]**; footnote **"Notes never change your grade or your revision schedule."** | D16/D26; R-AI3 |
| F3 | Optional revisit added | "Optional revisit · Thu, Dec 17" listed **apart from** the five-touch dots, chip `optional`; "Remove" link | D16 (optional, off-ladder) |
| F4 | Revision queue row + touch result, **materially different** touch pass | AB03-style row with chip "correct, with improvements"; the touch result shows the notes collapsed (after conclusion) with "From this review's solution" | D26 (materially different fingerprint) |
| F5 | **Live touch or open attempt** | The notes slot shows `xl-lock` **"Notes are hidden during this review — they come back when you finish."**; not expandable | t5 §8 (withheld during live touches); ADR-0027 §1 |
| F6 | Near-identical resubmission | "Same approach as your Dec 3 solution — its notes still apply." (notes from the last reviewed solution, not re-reviewed) | D26 |
| F7 | Reviewed, nothing to improve | Quiet line "Reviewed by xLearn AI · no improvement notes" (an optimal solution legitimately gets none) | t5 §4 (acceptance set: "no pointer is correct") |
| F8 | Notes unavailable | Three muted variants, no blame: passing-solution consent off → "Improvement notes are off · **Settings**"; advisory work shed or allowance paused → "Improvement notes are paused for now."; AI off → nothing but the provenance chip | ADR-0031 §5 (advisory shed first); ADR-0031 §4 (consent) |
| F9 | Where notes appear | Low-fidelity strip: the problem page after conclusion, the arena study view and the coach's review mode ("includes your xLearn AI notes", reference AB01) — and an explicit "**Never** on your public profile" callout (P10 shape) | t5 §8 (C3, private); [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) P10 |
| F10 | < 1024 px / 390 px | Notes as a full-width accordion under the result; the code excerpt scrolls horizontally inside its own box (no page scroll) | — |

Behaviour notes must state: notes are per (account, item), learner-private, erased with the account; at most 3; line ranges
always fall inside the submission (validator-enforced); notes and the optional revisit never touch pass/fail, grade or ladder;
collapsed during live touches and never shown while the item is live.

### 4 · AB18 AI allowance meter + Settings AI consents [X]

`AB18-ai-allowance-consents.html` — how much xLearn AI the learner has left and the three switches that govern it. Consumed by
[m4-06](sprint-m4-06.md) (task 3) on top of [m4-05](sprint-m4-05.md) (`GET /api/me/ai-allowance`, `PATCH /api/me/consents`) and
[m4-02](sprint-m4-02.md) (caps, breaker, per-account disable). Sources:
[ADR-0031 §4](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24) (consents),
[§5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25) (percentage, never dollars; degrade order),
[§7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (UI names),
[t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (learner view `{used_pct, resets_at, state:
ok|low|paused, global_paused}`), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (consent wording, withdrawal),
[ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (the acceptance
twins), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L17, [PRD §5.6](../../prd/xlearn-v2-prd.md) R-AI4–R-AI6.

| Frame | State | Must show (final copy) | Decisions |
|---|---|---|---|
| F1 | Settings → **"xLearn AI (included)"**, allowance ok | `ds-meter` "**This month: 38% used** · resets Jan 1"; "Covers AI suggestions on your graded work and improvement notes on passing solutions. No key needed." — **no dollar figure anywhere** | ADR-0031 §5 (percentage, never dollars); R-AI4/R-AI5 |
| F2 | Low (≥ 80%) | Meter in `--ds-warn`; "86% used — improvement notes pause first; suggestions on graded work continue until 100%." | ADR-0031 §5 degrade order |
| F3 | Paused for this account (month) | Meter full in `--ds-err`; "**xLearn AI is paused for your account until Jan 1.** You'll pick mistakes and grades yourself until then — nothing else changes." | R-AI5; D14 manual path |
| F4 | Daily limit reached | "Today's xLearn AI reviews are used up — they're back at 05:30 tomorrow." (the UTC reset rendered in the account timezone) | L17 ($1/day, ≤ 20 analyses/day, ≤ 6 AI finals/day) |
| F5 | **Paused for everyone** (breaker or app cap) | Degradation badge in the AB11 style "**xLearn AI paused**" + detail "xLearn AI is paused for everyone right now. Grading still works — you'll fill in mistakes yourself. It resumes on its own." — no reason, no provider name, no dollars | ADR-0031 §5 breaker; D34 (in-app badge, no alert) |
| F6 | Off for this account (owner or auto) | "xLearn AI is turned off for your account for now. You can keep practising — grading suggestions are manual." | t5 §6 (flag-based throttle → `ai_disabled` pending owner review) |
| F7 | **Settings AI consents** | Three `ds-toggle`s, **unticked** for an account that never chose. The first two **labels are AB19 F9's, verbatim** ([ds-l-01](sprint-ds-l-01.md); = [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)): **"xLearn AI reviews my graded work"** and **"…and also reviews my passing solutions for improvement notes"** (disabled until the first is on). Settings-only helper lines under them (these may differ from AB19's inline text): "Mistake suggestions, concepts and AI grades on your submissions. Off: you pick them yourself." / "Improvement notes on solutions that already pass. Never changes your grade." New, Settings-only: the third toggle **"xLearn AI may review my behavioral answers"** — "A separate choice. There is no behavioral course yet." Footer reuses AB19's retention sentence: "If on, your work is sent to Anthropic, processed outside India, kept by them for up to 30 days and never used for training. **Privacy notice →**"; Settings-only line "Last changed Dec 3, 2026 · consent v1" | ADR-0031 §4; t5 §8; R-AI6; AB19 F9 labels and retention sentence |
| F8 | Just switched off | Immediate, no confirm dialog; toast "xLearn AI reviews are off. You'll pick mistakes and grades yourself." (and turning the first off also turns the second off) | t5 §8 ("free and easy to withdraw") |
| F9 | **The two names** | Settings sections side by side: **"xLearn AI (included)"** (allowance + consents) and **"Your AI coach (your key)"** (reference AB01 F13, not redrawn) with one line each: "Included — reviews your submissions." / "Your own key — chat with a coach, billed by your provider." | ADR-0031 §7 |
| F10 | AI-off badges elsewhere | Small chips where suggestions would appear: results dock "xLearn AI off · manual", Today card "xLearn AI paused", mistake entry "pick a category" (no AI chip) | m4-06 task 3 ("badges when AI is off") |
| F11 | < 1024 px / 390 px | Meter and toggles stacked full width; helper text under each toggle; the badge collapses to an icon + label | — |

Behaviour notes must state: the meter reads `used_pct` and `resets_at` only (the API never returns money to the browser); `state`
drives F1–F3, `global_paused` drives F5, the per-account disable drives F6; consent changes are effective for new work at once
(nothing already sent is recalled — provider copies expire on the provider's schedule); toggles are real `<button role="switch">`
with `aria-checked`; the acceptance step (AB19) and Settings use **identical** consent labels and the same retention sentence
(helper lines, the behavioral toggle and the last-changed line are Settings-only).

### 5 · Self-review checklist (run before merging) + screenshots [X]

The session's own gate before the merge (D40: nothing waits on the owner); its ticked result goes in the PR body. Fix what
fails, then re-check.
- **`theme.css` check:** linked (then `board.css` if on `main`), never copied or overridden; `--ds-*` tokens only, no new
  colours; difficulty Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; the index's file names.
- Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) M4 row
  ("accept, edit, re-grade, compare, override, honor claim"; "pointer notes + 'correct, with improvements'"; "allowance meter +
  Settings AI consents").
- **Decision checks:** D14 (edits bounded, labelled self-set; manual always reachable from every AI state), D26 (no grade/ladder
  effect anywhere), D18/D27 ceilings (Clean disabled when late or > 3 failed submits; Clean **and** Rough disabled after a hint or
  coach help on that problem — the cap is Assisted), ADR-0031 §5 (**no dollars** on any learner surface), D31 (no mock best/average
  on any public surface), D34 (no "we've alerted the team" copy — nothing alerts).
- **Leak check:** no pattern chip, concepts or notes while an item is live; no hidden inputs, anchors, must-cover lists or
  exemplars; accepted probe answers only after conclusion; the review_flag copy never names the heuristic.
- **Copy check against neighbours** (where frozen on `main`): AB04 F10 (accepted answers), AB05 F8 (grades waiting), AB11 (badge),
  AB12 (provenance chips), AB19 (consent strings). Mismatches go into "Decisions to confirm".
- Screenshot every board full-page at **1440 px** and **390 px** into `design-system/screens/v2/shots/AB16@1440.png`,
  `AB16@390.png`, … (each ≲ 500 KB).

### 6 · Open the design PR [X]

Branch `design/ds-m4-01`; conventional commit `docs(design): AB16 AB17 AB18 — M4 platform-AI boards` ending with the attribution
lines. Open a PR titled `docs(design): AB16 ★ AB17 AB18 — M4 boards` with each board's frame list, the screenshots embedded
(`https://github.com/sujaykumarsuman/xlearn/blob/design/ds-m4-01/design-system/screens/v2/shots/<file>?raw=true`), the ticked
self-review checklist (task 5) and a **"Decisions to confirm"** list. The list doesn't block the merge: each item states the
default the merge freezes (the boards as drawn), and the owner may revisit any item after the merge through a follow-up design
PR. At least:
- **Edit before re-grade.** D14 lets the learner edit (bounded, self-set) straight from the card; [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade)
  item 3 had override only after the re-grade. The boards follow **D14** (owner decisions override the research body).
- **Allowance in percent only.** ADR-0031 §5 says never dollars; [m4-05](sprint-m4-05.md)'s register wording ("remaining $")
  is read as an internal field that never reaches the browser.
- **Daily reset** shown in the account timezone although caps reset at 00:00 UTC.
- The `review_flag` copy (F10) and the per-account-disable copy (AB18 F6).

### 7 · Freeze: merge on CI green [X]

Once the PR number is known, push the status commit (the Status note). When CI is green (fix, then merge, on failure),
squash-merge: **the merge is the freeze** (D40). Never enable auto-merge. Then sync `main` (`git checkout main && git pull`).
[m4-01](sprint-m4-01.md) cannot start until it lands (its entry gate), and [m4-06](sprint-m4-06.md) builds against exactly
these files. The owner may review after the merge; any change is a follow-up design PR.

## Acceptance criteria

- [ ] Every frame listed for AB16 (F1–F16), AB17 (F1–F10) and AB18 (F1–F11) is present with final copy, its state, its decision refs
      and a behaviour-notes aside.
- [ ] No board shows dollars to the learner, leaks withheld data, or names the injection heuristic; manual entry is reachable from
      every AI state (F10–F12 of AB16; F3, F5, F6 of AB18).
- [ ] Boards open with no JS runtime; `theme.css` is linked, not copied or overridden; only this sprint's three board files and
      their screenshots are added (no `index.html`/`board.css` edit).
- [ ] The self-review checklist passed and is ticked in the PR body, with the "Decisions to confirm" list (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze) with 1440 px and 390 px screenshots of each board; this file and `docs/v2/status.md` record ds-m4-01 ✅ and AB16–AB18 "frozen (merged)".

## Release

**Land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag. Nothing deploys: boards are preview-only files
under `design-system/screens/v2/`, never imported by `web/`. Launching the prompt is the owner's approval
([D40](../feasibility.md#decisions-log-newest-first)), so nothing waits on the owner; the owner may review after the merge, and
any change to a frozen board is a follow-up design PR. The merge is the freeze that gates [m4-01](sprint-m4-01.md) (M4 entry)
and is what [m4-06](sprint-m4-06.md) implements. No infra PR; the only `status.md` edits are this sprint's rows.

## Definition of Done

PR merged on CI green (the freeze) with AB16–AB18 and screenshots · every acceptance box ticked · this file's Status all ✅ and
`docs/v2/status.md` updated (Sprint-board row ✅; Artboards AB16–AB18 "frozen (merged, PR #N, <date>)") · no `web/`, `internal/`, `index.html` or `board.css` change · local `main` synced.

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D14/D15/D16/D17/D18/D26/D27/D31): cite the decision per frame; where two sources
  disagree (D14 vs t4 §3.5 on edit-before-re-grade), follow the owner decision and list it under "Decisions to confirm".
- **Drawing v2.0 as if rubric items existed.** Only F2–F4 and F13–F15 of AB16 are live in v2.0 (F10–F12 need `Score`); label the rubric frames so
  m4-06 does not build a rubric UI nobody can reach, and so a post-merge reader (m4-06, the owner) sees the live frames first.
- **Dollar leakage.** The allowance must read as a percentage everywhere, including the paused and daily states; a "$" in a
  frame is a bug (ADR-0031 §5).
- **Consent-string drift from AB19.** The acceptance step and Settings must say the same thing; if ds-l-01 is not frozen yet,
  use the ADR-0031 §4 / t5 §8 strings exactly and flag it.
- **Countdown drift.** Every countdown is a server deadline; a board that implies a client timer invites the bug m4-06 lists.
- **Promising alerts.** D34: no copy may imply that anyone was notified; degraded states say what still works.
- **Parallel status edits:** other sessions may update `docs/v2/status.md` when they land; rebase on `origin/main` before the
  status commit, touch only this sprint's rows and keep theirs.
