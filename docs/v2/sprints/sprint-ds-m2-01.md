# Sprint ds-m2-01 — Design M2: Touch ★, catalog/agenda, public profile v2, visibility (AB04 ★, AB05, AB06, AB22)

> **Milestone:** M2 — attempt engine and projections   ·   **Track:** design (parallel; order 6)
> **Prereqs:** none (no prerequisite beyond `depends_on`)   ·   **Unblocks:** [m2-01](sprint-m2-01.md) (freeze gate: AB04–AB06 and AB22 frozen before M2a) · consumed by [m2-03](sprint-m2-03.md) (AB06, AB22) and [m2-04](sprint-m2-04.md) (AB04, AB05) · [ds-p-01](sprint-ds-p-01.md) (its entry gate: AB05 frozen)
> **Release action:** **land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag, nothing deploys (launching the prompt is the owner's approval, [D40](../feasibility.md#decisions-log-newest-first); the owner may review after the merge, and any change to a frozen board is a follow-up design PR)
> **Calendar:** week 1–2 (2026-09-25 → 10-09); the freeze lands with this session's merge, well before M2a (late October, after v1.8.0). No owner event (`ev-freeze-ds-m2-01` is automatic at the merge and needs no tick).
> **Execute with:** [`../prompts/prompt-ds-m2-01.md`](../prompts/prompt-ds-m2-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own files only; never `index.html` or `board.css`) | X | ⬜ |
| 2 | AB04 ★ A8 Touch (hero) | X | ⬜ |
| 3 | AB05 catalog + cross-course agenda + Today in minutes | X | ⬜ |
| 4 | AB06 public profile v2 (P12) | X | ⬜ |
| 5 | AB22 visibility toggles | X | ⬜ |
| 6 | Self-review checklist (run before merging) + screenshots | X | ⬜ |
| 7 | Open the design PR (screenshots, frame lists, the ticked self-review checklist, "Decisions to confirm") | X | ⬜ |
| 8 | Freeze: squash-merge on CI green (the merge is the freeze) → status → sync `main` | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Land and sync ([D40](../feasibility.md#decisions-log-newest-first)):** the board PR
> merges on CI green and the merge is the freeze, so this sprint records its own close-out. Once the PR number is known, a last
> commit on the PR branch sets tasks 1–8 and _Overall_ ✅ here (task 8: "frozen: merged in PR #N, <date>") and, in
> [`../status.md`](../status.md), this sprint's Sprint-board row ✅, the Artboards rows AB04, AB05, AB06 and AB22 → ✅ "frozen
> (merged, PR #N, <date>)", the Snapshot's artboard count (`ev-freeze-ds-m2-01` is automatic: no tick),
> and a Decisions-log line for "Decisions to confirm" #1 (the honor-key policy the merge froze). If the merge slips to another
> day or fails after that commit, correct the rows in a follow-up docs PR merged the same way.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] none (no prerequisite beyond `depends_on`)

_Informational, not a gate:_ `design-system/screens/v2/index.html` and `board.css` are written once by [ds-m1-01](sprint-ds-m1-01.md)
(its [task 1 table](sprint-ds-m1-01.md#1--board-index--shared-chrome-once-for-every-design-sprint-x) is the naming source of truth).
If they are on `main`, link `board.css`. If ds-m1-01's PR is still open, **do not create or edit either file**: put the same `bd-*`
chrome in each board's own `<style>` block. Either way use the Task 1 file names — they are the names the index links.

## Goal

Draft the four M2 boards — including the **A8 Touch hero** — as static, preview-only HTML on
[`theme.css`](../../../design-system/theme.css), with every frame, state and final copy, so M2's build sprints
([m2-01](sprint-m2-01.md) engine, [m2-03](sprint-m2-03.md) profile/visibility, [m2-04](sprint-m2-04.md) Touch UI + Today)
are built against frozen designs. Per **BP3** (owner decision 2026-09-24) the agent drafts all boards, heroes
included; per [D40](../feasibility.md#decisions-log-newest-first) the board PR merges on CI green and the merge is the freeze —
the owner may review afterwards, and any change to a frozen board is a follow-up design PR. The rollout's "owner designs the
heroes in Claude Design" ([rollout §9](../rollout-plan.md#9-artboards-by-milestone)) is superseded by BP3.

## Scope

**In**
- AB04 ★, AB05, AB06, AB22 as static HTML under `design-system/screens/v2/`, one file per board.
- Every frame/state listed in Tasks 2–5, with final copy, a frame label that cites its decision(s), and a behaviour-notes aside.
- 1440 px and 390 px screenshots of every board, committed under `design-system/screens/v2/shots/` (ds-m1-01's convention)
  and embedded in the PR description.
- Freeze = the board PR merged on CI green (D40), before M2a ([m2-01](sprint-m2-01.md)); plus this sprint's own rows in
  `docs/v2/status.md` (Status note).

**Out**
- Any `web/` code → [m2-03](sprint-m2-03.md) (AB06, AB22) and [m2-04](sprint-m2-04.md) (AB04, AB05).
- Editing `design-system/screens/v2/index.html` or `board.css` → owned by [ds-m1-01](sprint-ds-m1-01.md) (written once).
- AB02/AB05 at **full** fidelity for a second course → [ds-p-01](sprint-ds-p-01.md) (pilot).
- Judge-graded re-solve (Run/Submit, results dock) → AB07/AB08 in [ds-m3-01](sprint-ds-m3-01.md); AB04 only shows a labelled "judge path (M3)" variant.
- Honor-probe **claim** and AI provisional/dispute → AB16 in [ds-m4-01](sprint-ds-m4-01.md).
- Coach states (the `coach_paused` banner itself) → AB01 in [ds-m1-01](sprint-ds-m1-01.md); AB04 references it.
- Shipping boards (preview-only; never imported by `web/`) and owner design hours (BP3; any owner review happens after the merge).

## Tasks

### 1 · Board scaffolding [X]

Static HTML boards under `design-system/screens/v2/`, one file per board, named exactly as the
[ds-m1-01 index table](sprint-ds-m1-01.md#1--board-index--shared-chrome-once-for-every-design-sprint-x) links them:
`AB04-touch.html`, `AB05-catalog-agenda.html`, `AB06-public-profile-v2.html`, `AB22-visibility-toggles.html`. Follow
ds-m1-01's [board conventions](sprint-ds-m1-01.md#2--board-scaffolding-the-conventions-x) (written once for every design sprint):
- Each file links `../../theme.css`, then `board.css` when it is on `main` (else the same `bd-*` chrome inline in the
  board's `<style>`), plus the Google Fonts `<link>` the v1 boards use (Inter, JetBrains Mono); mono for timers, numbers
  and identifiers. It reuses the tokens/components **verbatim** (dark "landscape console"; see
  [`design-system/README.md`](../../../design-system/README.md): `ds-*` components, `xl-*` shell and signature components
  such as `xl-touch`/`xl-touch__d--pass|--due|--fail|--mock`, `xl-timer`, `xl-pat`, `xl-lock`, `ds-badge--violet`).
  Difficulty tokens: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`. Board-only layout CSS lives in the
  file's `<style>`, uses `--ds-*` tokens only (no new colours) and never overrides a `ds-*`/`xl-*` rule.
- Frames: full-screen frames stack vertically; panel-sized states sit side by side in a `.bd-grid`. Each frame has a
  label `ABnn-Fk · <state>` (e.g. `AB04-F4 · Name the pattern`), the decision refs it implements (e.g. `D15 · T4 §3.6`),
  final copy (no lorem ipsum, no "TBD"), and a **behaviour notes** aside: timers, gates, error codes, a11y (focus
  order, contrast ≥ 4.5:1 on text, keyboard shortcuts, `aria-live` for clocks), and the < 1024 px layout. Each board
  ends with a 390 px `.bd-narrow` section for its key frames (the "< 1024 px" frames below live there).
- **Preview-only:** no script beyond trivial static toggles, never imported by `web/`, never shipped. v1 `.dc.html`
  artboards ([`design-system/screens/`](../../../design-system/screens/)) are references only.
- **Touch only this sprint's own board files and their screenshots** (plus this sprint's own rows in `docs/v2/status.md`).
  `index.html` and `board.css` are written once by ds-m1-01 (the index has a row and link for every board AB01–AB30; missing
  files 404 in preview), so parallel design PRs (ds-m1-01, ds-m2-01, ds-l-01) never conflict.

### 2 · AB04 ★ A8 Touch (hero) [X]

The revision touch as a **server-timed attempt** (practice `purpose=touch`, built in [m2-01](sprint-m2-01.md), UI in
[m2-04](sprint-m2-04.md)). Sources: [t4 §8 Touches](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed),
[t4 §3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts) (as overridden by D15),
[t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course) (DSA criteria),
[ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer), D15 and D27
in the [decisions log](../feasibility.md#decisions-log-newest-first).

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Queue card | Revision queue card (evolves AB03): `#16 3Sum` · Medium · **no pattern chip** (withheld while due) · touch dots at L3 · format badge **"Re-solve · 20:00"**; at L4–5 also **"Mock conditions"** (`ds-badge--violet`); CTA **[Start review]** | `withhold()` ([m1-06](sprint-m1-06.md)), R-SR4 |
| F2 Start cover | "Starting begins a 20:00 server clock. It keeps running if you leave. The coach is paused during reviews. At the end, unmet criteria count as a fail." **[Start review]** · **[Not now]** | D15, D27 |
| F3 Mock-conditions cover (L4–5) | Same as F2 plus "Mock conditions: statement only · coach off · talk it through aloud, no notes." | R-SR4, v1 `isMockTouch` |
| F4 Step 1 · Name the pattern | Statement panel; probe prompt "Name the pattern."; free-text field; **pattern clock** (mono, counts up from start, target mark at 2:00: "Counts if locked in under 2:00"); main clock 20:00 in the HUD (`xl-timer`); **[Lock in]** (Enter). After lock-in: "Locked at 0:48" — **no correctness shown** | T4 §3.7 (lock-ins, correctness withheld), R-SR2 |
| F5 Step 2 · Re-solve (M2 self path) | Blank editor (`xl-code` styling, placeholder "// re-solve from memory — a blank editor, not a re-read"); **[I solved it]** · **[I couldn't solve it]**; helper "Self-attested: counted on your honour and labelled *self* in your stats. Code checking arrives with the judge." After either: "Recorded at 13:20" | D15, T4 §6.6 (`correct_in_timer` is self-attested until M3) |
| F6 Step 2 · Re-solve (judge path, M3 preview) | Same step with Run/Submit and a compact results line ("Passed hidden tests at 13:20"); frame label **"M3 preview — final in AB07/AB08"** | ADR-0029 §2 |
| F7 Step 3 · State the complexity | Two blanks **Time** / **Space** (mono inputs, hint "e.g. O(n log n)"); **[Lock in]**; after: "Locked" (no correctness) | T4 §6.6 (`blank.big_o`) |
| F8 End review confirm | `ds-modal--sm`: "End review now? Unmet criteria count as a fail, and this problem goes back to Day 1." **[Keep going]** (default focus) · **[End review]** | T4 §8, D15 |
| F9 Result · pass | Server criteria rows: "Pattern named · 0:48 ✓" · "Solved in time · 13:20 ✓ (self)" · "Complexity · O(n²) / O(1) ✓"; headline **"Passed — advances to Day 21"**; touch dots updated; provenance chip "self · honour" | R-SR2, T4 §8 |
| F10 Result · fail | Unmet rows marked, with the accepted answer shown **only now** ("Accepted: Two Pointers"); headline **"Not passed — resets to Day 1 (due tomorrow)"**; "Mistake entry opened" link to Mistakes | R-SR3; a failed re-solve opens a mistake (v1 flow 3, R-MJ4) |
| F11 Result · abandoned | Deadline passed after the probes were shown, no evidence: **"Time ran out — counted as a fail."** "resets to Day 1" | D15 (shown → fail), T4 §3.6 |
| F12 Deadline on page | Clock hits 0:00 → "Time's up · recording your result…" (server concludes at deadline + 5 s) → F9/F10/F11 | D15 |
| F13 Resume | Returning within the timer: "Resuming · 12:41 left"; locked steps stay locked (read-only "Locked at 0:48") | D15 (server clock never pauses) |
| F14 Couldn't verify (M3+) | **"Couldn't verify, not counted. [Retry]"** — the touch is voided, stays due; label "appears only when the judge is unavailable" | D15 (infra-inconclusive → void), T4 §8 |
| F15 Coach paused | Coach bubble in its paused state during the touch (reference AB01's `coach_paused` frame; do not redesign it) | D27 |
| F16 < 1024 px | Steps stacked; HUD pinned with both clocks; Statement / Work tabs | T4 §8 shell |

Behaviour notes must state: one live touch per problem; correctness hidden until conclusion; the touch concludes the
moment every criterion is decided; a never-shown touch is voided silently (no frame — the learner never saw it);
keyboard: Enter locks in, Esc closes modals, the End-review button is never the default action; clocks announce
at 5:00 and 1:00 left via `aria-live="polite"`.

### 3 · AB05 catalog + cross-course agenda + Today in minutes [X]

Sources: [ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today) (D4),
[PRD §5.4](../../prd/xlearn-v2-prd.md#54-today-and-budget-across-courses), [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)
(`preview` hidden outside the cohort). Non-DSA content is **low fidelity** here (placeholder course "Go concurrency");
full fidelity comes with [ds-p-01](sprint-ds-p-01.md).

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Catalog | Course cards: DSA (enrolled, active: progress ring, "≈ 35 min/day of your plan"), Go concurrency (**Preview** badge, "owner & testers only" — visible only to the cohort), System design (**Coming soon**, disabled), an active not-enrolled course with **[Enroll]**; retired courses never listed | D4, ADR-0034 §2 |
| F2 Today · budget header | **"Today · 55 min of 60"** (planned of budget — the exact string [m2-04](sprint-m2-04.md) builds as "Today · N min of M", on the agenda and the course dashboard) with a segmented `ds-meter` (reviews / new work per course) | D4 |
| F3 Today · Reviews first | Cross-course due touches, oldest first: each row = course chip · `#id` title · format badge ("Re-solve · 20:00", "Mock conditions") · "≈ 20 min"; header "Reviews first · 3 due · ≈ 45 min" | D4, R-SR5 |
| F4 Today · per-course block | "New DSA work is paused until today's DSA reviews are done." (R-SR5 applies **per course**); other courses still get their share | D4 |
| F5 Today · new work split | Remainder split across active enrollments: "DSA · 10 min · next: #18 …", "Go concurrency · 5 min" | D4 |
| F6 Over budget | "Reviews alone need 75 min today — new work is paused. Reviews still come first." | D4 |
| F7 All caught up | "Nothing due today. Your next review is Thu (2 due)." | — |
| F8 Grades waiting (slot) | Placeholder card "1 grade waiting · auto-accepts in 23:41 [Review]" labelled **"M4 — slot only"** (low fidelity) | D14 |
| F9 Cross-course agenda | Next 7 days: day rows (Today, Tomorrow, Thu …) with due touches per course, minute totals per day; toggle "All courses / This course" | D4, ADR-0026 §5 |
| F10 < 1024 px | Stacked cards; the agenda as an accordion | — |

Behaviour notes: the planner is a deterministic pure function in the gateway (same inputs → same plan); minutes come
from each manifest's `est_minutes` per format; reviews are ordered oldest-due first across courses.

### 4 · AB06 public profile v2 (P12) [X]

Sources: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P1–P12),
[ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas),
[ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas),
[t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas), D7 and **D31** ([ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md)).
Evolves today's `web/src/screens/UserDashboard.tsx` layout (20:80 profile, completion-by-phase bar).

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Header | Display name, joined month, region band; **header totals from visible courses only**: solved, current/longest streak, activity heatmap (from `proj_activity`: attempts + reviews + mocks per UTC day), **"Mock interviews · 6" (count only)** | D7, P6, D31 |
| F2 Course row (expanded, DSA) — **M3+ state** | "Concluded 48 · Passed 41 · of 151 core"; difficulty split; topic mastery bars; **first-attempt grade mix with provenance** chips (Self / Judge / AI); **"Judge-checked 62%"** (a populated value, frame label **"M3+"**); touch stats "Reviews completed 42 · Day-7 pass rate 78%"; mocks count only | P7, P9, D31 |
| F3 Course row — **M2 state** (judge-checked pre-M3) | The same row as [m2-03](sprint-m2-03.md) ships it: judge-checked shown as "—" (never a 0% presented as a claim) with the tooltip "Judge-checked results arrive with the code judge."; provenance chips Self only | P7 (hidden / "—" until M3) |
| F4 Second course row (collapsed) | Low-fidelity placeholder course row | P2 |
| F5 Provenance tooltip | "Self = you reported it. Judge = hidden tests passed. AI = suggested, then accepted. Judge-checked counts only judge results." | P7 |
| F6 No visible courses | Profile card + "No public course stats yet." | D7 |
| F7 Private / unknown / suspended / erased (L-E) | One **identical** frame: "No profile for @name" (same copy, same 404 for all four; erased applies from L-E) | P5, P11 |
| F8 < 1024 px | Header card, then totals, then course rows | — |

Behaviour notes must list what is **never public** (P10 allowlist): item ids, per-item pattern, sub-day timestamps,
arena or Run counts, prose, mistakes, `scored_by`, mock best/average, transcripts. The board must not show any of them.

### 5 · AB22 visibility toggles [X]

Sources: P5 ([rollout §10](../rollout-plan.md#10-public-dashboard-tasks)), D7, [ADR-0033 §13](../../adr/0033-invite-only-admission-and-owner-admin.md#13-public-dashboard-authz-deltas);
built in [m2-03](sprint-m2-03.md) (identity `PATCH /accounts/{id}/visibility`, synchronous cache-epoch bump).
Evolves the Settings profile section in `web/src/screens/Settings.tsx`.

| Frame | Content and final copy | Decisions |
|---|---|---|
| F1 Section | "Public profile" card: profile toggle **Public / Private** with the URL `projects.sujaykumar.dev/xlearn/u/<username>`; per-course toggles for enrolled courses with "Default: visible (set by the course)" / "Default: hidden" hints | D7, P5 |
| F2 Go private — confirm | "Make your profile private? Anyone opening your link sees 'No profile for @name' — the same as an unknown user. Applies immediately." **[Cancel]** · **[Make private]** | P5 |
| F3 Hide a course — confirm | "Hide DSA from your profile? Its stats disappear, and your header totals (solved, streak, activity, mocks) are recomputed from visible courses only. Applies immediately." | D7, P6 |
| F4 Private state | Course toggles disabled with "Course visibility applies while your profile is public." | P5 |
| F5 Saved / error | Inline "Saved" confirmation; on failure the toggle reverts with "Couldn't save — try again." | — |
| F6 < 1024 px | Stacked toggles | — |

### 6 · Self-review checklist (run before merging) [X]

The session's own gate before the merge (D40: nothing waits on the owner); its ticked result goes in the PR body.
- **Brief:** walk every frame against its cited decisions and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone)
  board list; every frame in tasks 2–5 is present with final copy, a frame label and a behaviour-notes aside.
- **`theme.css`:** linked (+ `board.css` when on `main`), never copied or overridden; `--ds-*` tokens only, no new colours;
  difficulty Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`; canonical index file names.
- **Leak check:** **no pattern chip during a due or live touch** (only the post-conclusion "Accepted" line), correctness
  never shown before conclusion, no hidden inputs, nothing from the P10 never-public list.
- **Decisions:** no "Re-implement" step (D16), no pause on the touch clock (D15), coach paused during the touch (D27),
  mock **count only** on the profile (D31); M3/M4 behaviour only in labelled preview frames.
- **Contrast** on every text colour pair used.
- **Screenshots:** every board full-page at **1440 px** and **390 px** wide (headless Chrome/Playwright) into
  `design-system/screens/v2/shots/AB04@1440.png`, `AB04@390.png` (and AB05, AB06, AB22), each ≲ 500 KB (ds-m1-01's
  convention: the shots become part of the frozen reference the build sprints compare against).

Fix what fails, then re-check.

### 7 · Open the design PR [X]

Branch `design/ds-m2-01`, conventional commit `docs(design): M2 boards AB04★ AB05 AB06 AB22`, PR titled
"design: AB04★ AB05 AB06 AB22 (M2)". The body: the screenshots embedded from the branch
(`…/blob/design/ds-m2-01/design-system/screens/v2/shots/<file>?raw=true`), a frame list per board with decision cites,
the ticked self-review checklist (task 6), and **"Decisions to confirm"**. The list doesn't block the merge: each item states
the default the merge freezes (the boards as drawn), and the owner may revisit any item after the merge through a follow-up
design PR:
1. **Honor-key strictness in M2** (AB04 F4, F7, F9, F10). From v1.10.0 ([m2-05](sprint-m2-05.md) turns the producers on) until
   [m4-04](sprint-m4-04.md)'s "I meant this" claim ships in v1.16.0, a pattern or big-O answer missing from the alias table or
   the normaliser fails its criterion and resets the ladder — the configuration
   [t4 §12](../research/t4-judge-contract.md#12-alternatives-considered) rejects ("worse than v1's self-affirm"). The options:
   **(A)** accept strict matching until M4, mitigated by generous aliases, the normaliser corpus and the accepted answer shown on a
   fail (the boards as drawn); or **(B)** v1-parity until the claim: a `public:*` probe's criterion is met when it is locked in
   within its limit (the lock-in is the v1 self-affirmation — correctness is hidden until conclusion, so an after-the-miss affirm
   would need M4's provisional state), recorded `source=attest`, `trust=honor`, with the key match kept for alias growth; m4-04
   then switches to strict + claim. If (B), F9/F10 show the probe rows as "✓ (self)" with "Key: Two Pointers" as a note.
   **The merge freezes (A)** (the boards as drawn), and [m2-01](sprint-m2-01.md) builds it and records the policy. If the owner
   picks (B) after the merge, a follow-up design PR redraws F9/F10 per the (B) note and m2-01's one-constant switch flips;
   nothing waits on the answer.
2. The AB05 Today header "Today · 55 min of 60" (one string for the agenda and the course dashboard).
3. AB06 F2 (M3+, populated judge-checked %) vs F3 (the M2 state, "—").

### 8 · Freeze: merge on CI green [X]

Once the PR number is known, push the status commit (the Status note). When CI is green (fix, then merge, on failure),
squash-merge: **the merge is the freeze** (D40), and it gates [m2-01](sprint-m2-01.md). Never enable auto-merge. Then sync
`main` (`git checkout main && git pull`). The owner may review after the merge; any change is a follow-up design PR.

## Acceptance criteria

- [ ] Every frame listed for AB04, AB05, AB06, AB22 is present with final copy and states, and each frame cites its decision(s).
- [ ] Boards carry the canonical index names, link `../../theme.css` (+ `board.css` when on `main`) and use only its tokens/components; no new colours; preview-only.
- [ ] No board leaks withheld data (pattern chip, pre-conclusion correctness, never-public profile facts).
- [ ] Screenshots at 1440 px and 390 px committed under `design-system/screens/v2/shots/` and embedded in the PR, with the ticked self-review checklist and the three "Decisions to confirm" (each item's frozen default stated).
- [ ] PR **merged on CI green** (the freeze); `index.html`, `board.css` and `theme.css` untouched; `docs/v2/status.md` changed only in this sprint's rows (ds-m2-01 ✅, AB04/AB05/AB06/AB22 "frozen (merged)", the #1 Decisions-log line).

## Release

**Land-and-sync; the merge is the design freeze** — the board PR squash-merges on CI green; no tag. Nothing ships: boards are preview-only and
never imported by `web/`. Launching the prompt is the owner's approval ([D40](../feasibility.md#decisions-log-newest-first)), so
nothing waits on the owner; the owner may review after the merge, and any change to a frozen board is a follow-up design PR.
The merge is the freeze that gates [m2-01](sprint-m2-01.md) (rollout §9 freeze rule: AB04–AB06 and AB22 before M2a).

## Definition of Done

Four board files merged on `main` (CI green; the freeze) · every frame present with final copy and decision cites · eight
screenshots under `shots/`, embedded in the PR · the ticked self-review checklist and "Decisions to confirm" in the PR
description · this file's Status all ✅ and this sprint's `docs/v2/status.md` rows updated · no edits outside the four board
files, their screenshots, this sprint file and those status rows · local `main` synced.

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D15 / D16 / D18 / D17 / D27 / D31) — cite the decision per frame;
  e.g. no "Re-implement" step anywhere (D16), no pause on the touch clock (D15), mock **count only** (D31).
- **Designing M3/M4 behaviour as if it were M2.** Judge Run/Submit, the honor-probe claim ("I meant this") and AI
  provisional are later; label any preview frame with its milestone and keep it out of M2's flow.
- **Withholding leaks** are the most likely self-review finding: the v1 Revision card shows the pattern chip
  (`Revision.tsx:235-238`) — AB04 F1 must not.
- **File-name drift with `index.html`** — ds-m1-01 and this sprint are parallel PRs, so this one usually runs before the
  index is on `main`. The Task 1 names are the index's canonical names (`AB06-public-profile-v2.html`,
  `AB22-visibility-toggles.html`, …); any other name breaks the index link and m2-03/m2-04's board lookup.
- **Parallel design PRs** (ds-m1-01, ds-l-01) — touch only your four files and their shots; never `theme.css`,
  `index.html` or `board.css`. They also edit `docs/v2/status.md` when they land: rebase on `origin/main` before the status
  commit, touch only this sprint's rows and keep theirs.
- **Strict honor keys before the claim** — the merge freezes (A), and m2-01 builds it unless the owner has picked (B) after
  the merge (then a follow-up design PR redraws F9/F10 and m2-01's switch flips). Nothing waits on the answer; the Decisions-log
  line records which policy was frozen.
