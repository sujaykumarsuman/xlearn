# Sprint ds-m3-01 — Design M3 part 1: Workspace-Code ★, results dock, degradation badges (AB07 ★, AB08, AB11)

> **Milestone:** M3 — judge plus code grader (the boards the M3 UI builds against) · **Track:** design · **Order:** 28
> **Prereqs:** none hard (the t4 §8 A1 doc-fix is a check, not a gate — see Entry gates); reuses [ds-m1-01](sprint-ds-m1-01.md)'s board index and conventions if merged
> **Unblocks:** [ds-m3-02](sprint-ds-m3-02.md) (the rest of the M3 set) · [m3-11](sprint-m3-11.md) (AB07; its gate is "AB07–AB12 frozen") · also consumed by [m3-12](sprint-m3-12.md) (AB08, AB11) and [m5-01](sprint-m5-01.md) (the AB07 evaluator-only variant is derived from the frozen AB07) · [ds-p-01](sprint-ds-p-01.md) (its entry gate: AB07★ and AB08 frozen)
> **Release action:** **PR, stop for owner review (design)** — the agent never merges; the owner's merge (or explicit approval) is the freeze
> **Calendar:** week 4 (2026-10-17 → 10-23) · owner event `ev-freeze-ds-m3-01` (review + merge) well before m3-11 (mid–late November)
> **Execute with:** [`../prompts/prompt-ds-m3-01.md`](../prompts/prompt-ds-m3-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board scaffolding (own three files only; never `index.html` / `board.css`) | X | ⬜ |
| 2 | AB07 ★ A1 Workspace-Code (hero) | X | ⬜ |
| 3 | AB08 A2 results dock | X | ⬜ |
| 4 | AB11 degradation badges | X | ⬜ |
| 5 | Self-review against the brief + screenshots (1440 px, 390 px) | X | ⬜ |
| 6 | Open the design PR and STOP | X | ⬜ |
| 7 | Freeze: owner reviews, approves and merges (`ev-freeze-ds-m3-01`) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Design exception:** this PR does **not** edit [`../status.md`](../status.md) (it may stay open
> for days). The PR sets tasks 1–5 ✅ and task 6 🔄 ("PR #N open — awaiting owner review"). **Who flips the rest, and when:**
> - If an agent merges on the owner's explicit approval in chat, it first pushes one last commit to the PR branch setting tasks 6–7
>   and _Overall_ ✅ ("frozen: PR #N, <date>") in this file, then merges.
> - If the owner merges in the GitHub UI, this file is left at 🔄. The **next session on `main`** that sees the PR merged (normally
>   [m3-11](sprint-m3-11.md), at its entry-gate check of "AB07–AB12 frozen") sets tasks 6–7 and _Overall_ ✅ here.
> - In both cases that next session writes the status.md side: the Artboards rows for AB07, AB08 and AB11 → "frozen (PR #, date)",
>   this sprint's Sprint-board row ✅, and owner event `ev-freeze-ds-m3-01` ticked.
>
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] Parallel sessions: no open PR adds `AB07-*`, `AB08-*` or `AB11-*` under `design-system/screens/v2/` (`gh pr list --state open`,
      `git worktree list`, ListAgents)

_Check (not a stop):_ [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)'s A1 frames should read
**45:00 cover, no "Re-implement" frame** (the doc-fix owned by the build-plan session, [rollout §9](../rollout-plan.md#9-artboards-by-milestone)
"Fix before briefing"). If `main` still shows "Start attempt · 15:00" or a Re-implement frame, brief per D15/D16/D18 anyway (the
decisions win, t4 §13) and say so in the PR body. It is a check, not a gate, because the decisions already settle the brief.

_Informational, not a gate:_ `design-system/screens/v2/index.html` and `board.css` are written once by [ds-m1-01](sprint-ds-m1-01.md).
If they are on `main`, use the exact file names the index links and link `board.css`. If not, **do not create them**: use the names
below and put the same `bd-*` chrome rules in each board's own `<style>`, noting it in the PR.

### M3 hard entry checklist (rollout §5, verbatim) — context, not this sprint's gate

This checklist gates the M3 UI sprint ([m3-11](sprint-m3-11.md)). This sprint delivers half of the **AB07–AB12 frozen** line
([ds-m3-02](sprint-ds-m3-02.md) the other half).

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

## Goal

Draft the first three M3 boards — the **A1 Workspace-Code hero (AB07 ★)**, the **A2 results dock (AB08)** and the **degradation badges
(AB11)** — as static, preview-only HTML on [`theme.css`](../../../design-system/theme.css), with every frame, state and final copy, so
[m3-11](sprint-m3-11.md) (Workspace + editor) and [m3-12](sprint-m3-12.md) (dock, badges) build against owner-approved designs. The
flow is the owner's D15/D16/D18 ([ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer),
Accepted): a **45:00 hard limit**, the **hint at 15:00 caps Assisted**, **give-up = Miss**, **a pass finishes the attempt, no re-implement**.
Per **BP3** (owner decision 2026-09-24) the agent drafts every board, heroes included; the owner only reviews and approves — rollout §9's
"the owner designs the heroes" is superseded.

## Scope

**In**
- `AB07-workspace-code.html`, `AB08-results-dock.html`, `AB11-degradation-badges.html` under `design-system/screens/v2/`.
- Every frame in tasks 2–4 with final copy, a frame label citing its decision(s), and a behaviour-notes aside.
- Screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/` (ds-m1-01's convention).
- The design PR, stopped for owner review.

**Out**
- Any `web/` code — the builds are [m3-11](sprint-m3-11.md) (AB07) and [m3-12](sprint-m3-12.md) (AB08, AB11).
- AB09 Problems, AB10 Arena, AB12 Week/Mistakes/Progress deltas → [ds-m3-02](sprint-ds-m3-02.md).
- AI suggestion / dispute, pointer notes, AI allowance (the real M4 content behind the "AI" placeholders here) → [ds-m4-01](sprint-ds-m4-01.md).
- The evaluator-only AB07 variant (no self picker) → [m5-01](sprint-m5-01.md), derived from the frozen AB07.
- Editing `index.html`, `board.css` or `theme.css`; `docs/v2/status.md` edits (see the Status note).
- Shipping boards: preview-only, never imported by `web/`, never embedded, never deployed. Owner design hours beyond review (BP3).

## Tasks

### 1 · Board scaffolding [X]

Follow ds-m1-01's conventions (its plan's task 2, restated in the index's Conventions block):
- One file per board, linking `../../theme.css` then `board.css` and the Google Fonts `<link>` (Inter, JetBrains Mono). Static HTML
  only: no JS runtime, no `support.js`, no `.dc.html` canvas markup.
- `theme.css` tokens and components **verbatim** (dark "landscape console"; `ds-btn`, `ds-card`, `ds-badge`, `ds-modal`, `ds-seg`,
  `ds-tabs`, `ds-meter`, `ds-spin`, `xl-topbar`/`xl-crumb`, `xl-diff`, `xl-pat`, `xl-lock`, `xl-timer`, `xl-touch`, `xl-code`, `xl-kbd`,
  `xl-coach`/`xl-fab` …). Difficulty: Easy = `--ds-ok`, Medium = `--ds-warn`, Hard = `--ds-err`. Board-only layout lives in the board's
  `<style>` and never overrides a `ds-*`/`xl-*` rule.
- Full-screen frames stack vertically; panel-sized states sit side by side in a `bd-grid`. Each frame is labelled `ABnn-Fk · <state>`
  with final copy (no lorem ipsum, no "TBD"), plus a **behaviour-notes aside**: timers, gates, error codes (exact strings), the decision
  it implements, a11y (focus order, contrast ≥ 4.5:1, keyboard shortcuts, `aria-live` for status changes, reduced motion) and the
  **< 1024 px** intent. Each board ends with `bd-narrow` 390 px frames.
- v1 references (read-only, not runnable): `design-system/screens/Problem.dc.html` — superseded by AB07; drop its `:104` pattern chip,
  `:126` pause button, `:332` early-unlock hint line ("Unlocks when the timer ends — or tap below."; v2's hint unlocks at 15:00, D18)
  and `:340-346` re-implement CTA ("Re-implement from memory"); its `:349-383` outcome picker survives only on the self path
  ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) "Changed v1 artboards").
- **Sample content is original:** use the title "#16 3Sum" and a statement written fresh (titles are fine, statements are not —
  [t1 §8](../research/t1-content-data-model.md#8-content-rights-stance)); sample numbers invented.
- **Never leak withheld data** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule), [§5](../../adr/0027-content-evalpack-and-user-data-model.md#5-leak-and-integrity-controls-preconditions-for-m3)):
  no pattern chip before the hint stage or conclusion; no hidden inputs, expected outputs, case ids, ordinals ("failed test 38"),
  stderr or per-case timing for hidden tests — only the passed count, the perf bit, the first failure class and bucketed usage.

### 2 · AB07 ★ A1 Workspace-Code (hero) [X]

`AB07-workspace-code.html`. One shell ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)): **HUD** (crumb
`xlearn / dsa / problems / #16`, title, `xl-diff--med`, role chip "Core", stage tabs, a **server timer ring with no pause**, a provenance
chip "Original · inspired by LeetCode 15 ↗" + "Not affiliated with LeetCode"); **left** statement + part navigator; **centre** editor +
results dock (AB08); **right** stage stepper `Attempt · Hint (15:00) · Solution` (no Re-implement), a live **grade outlook**, the action
card and the conclusion card. Sources: D10, D15, D16, D18, D20, D27 ([feasibility log](../feasibility.md#decisions-log-newest-first)),
[ADR-0029 §3–§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop), [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers),
[§3.3–§3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution), [§6.3–§6.4](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only).

| Frame | State | Must show |
|---|---|---|
| F1 | **Cover** (before Start) | HUD without timer running; **no statement, no pattern chip** (D10: the statement shows on Start); a rules card: "45:00 · the timer starts when you start and doesn't pause" · "The hint unlocks at 15:00 — opening it caps this attempt at **Assisted**" · "**Clean:** pass within 20:00, no hint, no coach, ≤ 3 failed submits" · "No pass by 45:00, or giving up, is a **Miss** — it comes back as a Day-1 review"; language picker (Go default · C++ · Python, only the languages the item lists); "Auto-graded · hidden tests" chip; primary **[Start attempt · 45:00]** |
| F2 | **Solving + Run** | Statement, samples, constraints and signature (left); editor with the language `ds-seg` and toolbar **Run** `⌘↵` / **Submit** `⌘⇧↵` (Ctrl on Windows/Linux); dock on **Samples**: s1 ✓, s2 ✗ with got vs expected (public samples only); custom input "Custom input — you'll see only your own output; there's no reference to compare against." (≤ 64 KiB); timer ring "32:10 left"; outlook "**Clean possible** · failed 0/3"; hint card locked "Hint unlocks at 15:00"; draft status "Saved · 10 s ago" |
| F3 | **Submit → WA** | Dock: "**Wrong answer:** 37/40 hidden passed. Hidden inputs aren't shown; try edge cases with a custom Run." Outlook "Clean possible · failed 1/3" |
| F4 | **Submit → perf TLE** | Dock: "**Too slow on large inputs:** correctness 40/40, performance failed. Check your complexity against n ≤ 1e5." Outlook "Clean possible · failed 2/3" (TLE counts; CE and REJECTED are free) |
| F5 | **Over time / hinted** (four side-by-side variants) | (a) past 20:00 → "**Rough at best** · past 20:00"; (b) at ≥ 15:00 the hint confirm `ds-modal`: "Open the hint? This caps this attempt at **Assisted**." **[Open hint]** **[Not yet]** (default focus), then the hint and pattern chip visible and "**Assisted at best** · hint used"; (c) coach used on this problem (D27) → "Assisted at best · coach used on this problem"; (d) ≤ 5:00 left: ring `--ds-warn`, ≤ 1:00 `--ds-err`, announced "5 minutes left" |
| F6 | **Give up · Miss modal** | `ds-modal`: "Give up and see the solution?" · "Your draft is sent for feedback." · "The solution unlocks now." · "This attempt is graded **Miss** — unless a submit that's still running passes." · "It comes back as a Day-1 review tomorrow, and a mistake entry opens." **[Give up · Miss]** (danger) **[Keep going]** (primary, default focus). No re-implement line (D16) |
| F7 | **Solution** (after give-up or timeout) | Solution stage unlocked: editorial, the reference in Go / C++ / Python tabs (`xl-code`), solution facts "O(n²) time · O(1) extra space", pattern chip now visible; the editor read-only with "Attempt ended — study the solution. You'll re-solve it from memory on Day 1." **No re-implement editor** (D16) |
| F8 | **Concluded Clean** | Conclusion card "**Clean**" (`--ds-ok`); provenance "Auto-graded · judge-checked · Go"; facts "Passed 40/40 hidden · performance ✓ · 12:48 · 1 failed submit"; five-touch dots with L1 due "Day 1 review · Thu, Oct 22"; "Compare with the reference" (solution unlocked for study); **[Next problem]** |
| F9 | **Concluded Miss + pre-fill** | "**Miss** · time limit reached (45:00)" (variant "Miss · gave up at 31:20"); best submit "40/40 correct · too slow"; mistake block with a **pre-filled** chip "Complexity misjudged — suggested from your result (correct, but too slow)" labelled *suggested*, **[Change]** opens the category picker (the learner wins), root-cause and insight fields; "Concepts to revise" (≤ 3 chips, post-conclusion only); "Day-1 review tomorrow" · "Solution unlocked" |
| F10 | **Resume** | "Welcome back — 18:20 left. The timer kept running while you were away." · "Draft restored (Go, saved 3 min ago)"; outlook recomputed "Rough at best · past 20:00" (26:40 elapsed). Variant: back after 45:00 → F9 with "Time ran out while you were away." (D15: no resume past the deadline) |
| F11 | **`self_grade_pending`** | "We couldn't grade this attempt automatically." + the cause: `contract_changed` "This problem was updated while you worked." / third release "The grader couldn't get a reliable result three times." / kill switch "Auto-grading is paused." Then "Pick your grade — up to **Rough** (you finished after 20:00)." The ceiling is the server's `self@1{cap:stage}` from the stage and the clock — hint or coach → Assisted, past 20:00 → Rough, else Clean — never from a pass, since `self_grade_pending` has no authoritative result ([t4 §3.3](../research/t4-judge-contract.md#33-transitions)). A **capped** picker (grades above the ceiling disabled, with the reason) and **[Retry grading]** once ("Retry used" after) |
| F12 | **Self-path variant** | (a) *not auto-graded yet* (no pack, grading pending — AB11 badge): the v2 shell, Run on samples when available, **no Submit**, v1 semantics per [t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution) (attempt 15:00 → hint 10:00 → solution; an early reveal "owes another attempt in 3 days"; re-implement from memory — kept on this **unjudged** path only, as [m3-08](sprint-m3-08.md) keeps `stages.reimplement` for self items; listed under "Decisions to confirm") and the four-grade picker (Clean "no help, in time" · Rough "solved, ugly" · Assisted "needed a hint" · Miss "didn't get it") — the **only** place the uncapped picker appears; (b) judge off for this account (kill switch, or not in the owner/tester cohort before GA): identical to v1's Problem page, no Run |
| F13 | **Language picker** | `ds-seg` Go · C++ · Python in the editor header; switching shows "Switched to C++ — your Go draft is kept"; the starter comes from the signature; a language the item doesn't list is absent; a learner-file compile error "Compile error (not counted) · line 12: 'seen' was not declared in this scope" (D20) |
| F14 | **< 1024 px** | Statement / Work / Results tabs (`ds-tabs`); timer + outlook in a sticky HUD row; Run/Submit in a bottom bar; 390 px frames of F1, F2, F6 and F9 |

Behaviour notes must state: the grade outlook rules (Clean ≤ 20:00, no hint, no coach, ≤ 3 failed · Rough ≤ 45:00 without hint/coach ·
Assisted after the hint or coach · Miss on timeout or give-up — D18); timer announcements via `aria-live="polite"` at 15:00 (hint
unlocked), 20:00 (Clean no longer possible), 40:00 and 44:00; the ring's `role="timer"` label; **Tab indents, Esc then Tab leaves the
editor** (stated in a visible hint); the Give-up modal's default focus on **Keep going**; judge features appear only for the owner/tester
cohort before GA ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)).

### 3 · AB08 A2 results dock [X]

`AB08-results-dock.html`. The dock evolves `Problem.dc.html:289-302` with tabs **Samples · Submission · History · Feedback**. Copy is
**exactly** [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)'s results-dock table where it has a row; the
rest is drafted here and listed in the PR for confirmation. Sources: [t4 §2.3](../research/t4-judge-contract.md#23-admission) (typed
responses), [§2.6](../research/t4-judge-contract.md#26-evaluation) (the DTO allowlist), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L9–L13).

| Frame | State | Copy / must show |
|---|---|---|
| F1 | Samples (Run) | per sample ✓/✗ with args, got and expected (public); stdout ≤ 8 KiB, stderr ≤ 2 KiB with "Output truncated"; compile diagnostics at learner-file lines only; custom-input result "Your output: …" |
| F2 | queued | "Queued · 1 ahead of you" (`ds-spin`) |
| F3 | running | "Running hidden tests…" |
| F4 | settling | "Recording your result…" |
| F5 | accepted | "All hidden tests passed · 40/40 · performance ✓"; bucketed usage "Time ≤ 0.5 s · Memory ≤ 64 MB" |
| F6 | WA | "**Wrong answer:** 37/40 hidden passed. Hidden inputs aren't shown; try edge cases with a custom Run." |
| F7 | perf TLE | "**Too slow on large inputs:** correctness 40/40, performance failed. Check your complexity against n ≤ 1e5." |
| F8 | CE, own code | "Compile error in your code — not counted." + diagnostics at your lines |
| F9 | CE, hidden API | "Hidden tests don't compile against your code. Check the required signature. (Not counted.)" |
| F10 | RE / MLE (drafted) | "Runtime error on hidden tests: 12/40 passed." / "Memory limit exceeded on hidden tests: 39/40 passed." — class only, no stack or stderr for hidden tests |
| F11 | `inconclusive` / close released | "We couldn't get a reliable result. Not counted. **[Submit again]**" |
| F12 | `contract_changed` | "This problem was updated while you worked. Not graded." + **[Reload problem]** "your draft is kept" |
| F13 | `not_evaluated_here` | "Checked only in a course attempt or review." |
| F14 | 413 | "Too large to submit (code is capped at 64 KiB)" |
| F15 | quota and limits (grid) | "8 submits left today" (shown at ≤ 10); `quota_exceeded` "You've used today's 100 submits. More at 00:00 your time."; Runs "You've used today's 200 Runs. Submits still work."; runner budget (L12) "You've used this hour's run time. Course submits still work."; `queue_full` "The grader is busy. Try again in 30 s." (Retry-After); `too_many_pending` "Wait for your earlier submissions to finish."; `too_fast` "Wait a moment between submits."; breaker level 1 (L13 > 40%) "Runs are limited to one every 30 s while the grader is busy. Course submits aren't affected."; level 2 (> 60%) "Runs are paused while the grader is busy. Course submits still work."; **arena variants** of the same dock (arena Runs *and* submits are budget-counted, [t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments)): L12 "You've used this hour's run time. Arena runs and submits resume next hour; course submits still work."; level 1 "Arena submits are delayed a few minutes while the grader is busy."; level 2 "Arena runs and submits are paused while the grader is busy. Course submits still work."; 503 `evaluation_unavailable{fallback: self}` "Auto-grading isn't available for this problem right now — you'll pick your own grade when you finish." |
| F16 | History | this attempt's counted submits "#3 · 12:48 · Go · Wrong answer · 37/40"; read-only code view; **[Copy to editor]**; never case detail |
| F17 | Feedback | before conclusion "Feedback appears when your attempt ends."; after: "Samples ✓ · hidden 37/40 · first failure: Wrong answer" + a link to the mistake suggestion; a greyed **"xLearn AI review — arrives with M4"** placeholder, labelled *not in M3* (content owned by AB16/AB17) |
| F18 | Narrow | the dock as the Results tab at 390 px |

Behaviour notes: status lines are `aria-live="polite"`; verdicts carry icon + text, never colour alone; polling cadence is the server's
`poll_after_ms`; closes, give-ups and the re-grade are exempt from the quota rows (t4 §2.3 row 6); counted (course, touch, mock)
submits are never budget-blocked, while arena work is budget-counted and shed first by the breaker (t3 §7.4) — hence "course submits"
in every budget/breaker line and the arena variants.

### 4 · AB11 degradation badges [X]

`AB11-degradation-badges.html`. With no alerting in v2 (D34, [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34)),
these badges are the in-app signal. Data: `GET /api/judge/status` ([m3-09](sprint-m3-09.md)) → saturated · breaker level · `evaluable = 0` ·
per-item contract mismatch. Shown only when judge is configured **and** the viewer is in the owner/tester cohort (presence by config,
[ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)). They never expose
internals (no pack version, PAT, error string or host detail).

| Frame | State | Must show |
|---|---|---|
| F1 | The badge set | icon + text, each with its popover copy: **"Grading pending · self-report"** (`ds-badge--info`) — "This problem isn't auto-graded right now, so you'll pick your own grade when you finish. Nothing you've done is lost." · **"Grader busy"** (`--warn`) — "Lots of code is running. Runs may be slower or limited; submits still count and are graded in order." · **"Auto-grading off"** (`--err`) — "Auto-grading is unavailable. You can still solve problems and pick your own grade." · **"xLearn AI paused"** (muted, labelled *M4 placeholder*) — "xLearn AI is paused. You can grade manually — nothing is blocked." |
| F2 | Problems placement | a row chip "Self-report" beside the difficulty for an item with a contract mismatch or no pack; a page banner when `evaluable = 0` |
| F3 | Workspace placement | a HUD chip and a cover note (AB07 F1/F12); "Grader busy" in the dock header (AB08 F15) |
| F4 | Dashboard (Today) placement | a banner "Auto-grading is off right now — problems use self-report until it's back." when `evaluable = 0`; "Grader busy" inline on the Resume card |
| F5 | Transitions | busy → normal: the chip clears and `aria-live="polite"` announces "The grader is back to normal"; no toasts that steal focus |
| F6 | Cohort and kill switch | a non-cohort account (before GA) sees no judge badges; with the kill switch (`JUDGE_BASE_URL` unset) judge is absent, so there is **no badge** — results just read "Self-graded" (flagged for the owner) |
| F7 | Narrow | badges collapse to icon + short label; the popover becomes a bottom sheet |

Behaviour notes: badge text keeps contrast ≥ 4.5:1 and never sits next to a same-coloured difficulty chip without its label; status
badges use `role="status"`.

### 5 · Self-review against the brief + screenshots [X]

Walk every frame against its cited decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone) list: 45:00 cover, Run/Submit,
language picker, hint at 15 caps Assisted, give-up = Miss, pre-fill, resume, self-path variant, **no re-implement on the judged path**
(AB07; the self-path variant keeps v1's, pending confirmation); queued/running,
WA, TLE, CE, inconclusive, `contract_changed`, 413/quota (AB08); the badges (AB11). Check against D15/D16/D18/D27 and against D17 and D31
(nothing here may lock the arena or show a public mock average). Leak check per task 1. Screenshot every board full-page at **1440 px** and
**390 px** into `design-system/screens/v2/shots/AB07@1440.png` / `AB07@390.png` (and AB08, AB11), each ≲ 500 KB.

### 6 · Open the design PR and STOP [X]

Branch `design/ds-m3-01`; conventional commit `docs(design): AB07 AB08 AB11 — M3 workspace, results dock, badges` ending with the
attribution lines; PR titled `docs(design): AB07★ AB08 AB11 — M3 boards (part 1)` with the screenshots embedded, a frame list per board
with decision cites, and a **"Decisions to confirm"** list. **Do not merge, do not enable auto-merge.** Requested changes land on the same
branch. The PR doesn't edit `docs/v2/status.md`.

"Decisions to confirm" must include at least: the self-path timing (v1 semantics kept per t4 §3.4 until M5, vs the 45:00 flow);
**the self-path re-implement** (drafted: kept, v1 semantics, as m3-08 keeps `stages.reimplement` for self items, vs dropped
everywhere because D16's "no mandatory re-implement" isn't limited to judged items — m3-11's acceptance says "no re-implement stage
anywhere", so the owner's answer settles both); the hint copy (t4's "Hint available any time" conflicts with D18's hint at 15:00 —
drafted as "unlocks at 15:00"); the drafted RE/MLE and quota copy, incl. the arena budget/breaker variants; the kill-switch "no badge"
reading; whether the solution is offered after a Clean pass (drafted: yes, for study).

### 7 · Freeze [O]

The owner reviews the PR (`ev-freeze-ds-m3-01`) and approves and merges it, or requests changes. The merge — by the owner, or by an agent
only on the owner's explicit approval in chat — **is the freeze**. [m3-11](sprint-m3-11.md) cannot start until AB07–AB12 are frozen.

## Acceptance criteria

- [ ] Every frame listed for AB07 (F1–F14), AB08 (F1–F18) and AB11 (F1–F7) is present with final copy, its state and a behaviour-notes aside citing its decision.
- [ ] AB07 has a 45:00 cover and **no re-implement on the judged path** (no Re-implement frame, stage, CTA or modal line); re-implement
      appears only inside the F12(a) self-path variant, listed under "Decisions to confirm"; the uncapped grade picker appears only in
      the self-path variant.
- [ ] No board leaks withheld data (no pattern chip before the hint stage or conclusion; no hidden inputs, case ids, ordinals or per-case timing).
- [ ] Boards open from disk or a static server with no JS runtime; `theme.css` is linked, not copied or overridden; `index.html`, `board.css` and `theme.css` untouched.
- [ ] PR open with 1440 px and 390 px screenshots of each board and a "Decisions to confirm" list; **not merged by the agent**.

## Release

**PR, stop for owner review (design).** Nothing deploys: boards are preview-only files. The owner's merge is the freeze
(`ev-freeze-ds-m3-01`); with [ds-m3-02](sprint-ds-m3-02.md)'s it satisfies the "AB07–AB12 frozen" line of the M3 checklist.

## Definition of Done

PR open with the three boards and screenshots · every acceptance box ticked · this sprint's Status table updated in the PR (tasks 1–5 ✅,
task 6 🔄 awaiting review) · no `web/` or `docs/v2/status.md` change · the agent stops at the open PR. (Done-done — ✅ Overall — is set
per the Status note: by the approving agent's last commit, or by the next session on `main`, normally [m3-11](sprint-m3-11.md).)

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D15/D16/D18/D17/D27/D31): cite the decision per frame; when one is ambiguous, draft the
  conservative reading and list it under "Decisions to confirm".
- **t4 §8's stale A1 frames** (15:00 cover, Re-implement): the decisions override the research body (t4 §13) — never draft them.
- **Leaking hidden-test detail through "helpful" copy** ("failed on test 38", a stack trace): the DTO allowlist forbids it; keep counts and classes only.
- **Promising M4 early:** AI appears only as clearly labelled placeholders; the real content is ds-m4-01's.
- **File-name drift** with `index.html`: use its names if merged; otherwise the ones above. Never edit the index or `board.css`.
- **Screenshots in git:** keep each PNG ≲ 500 KB; they become the frozen reference m3-11/m3-12 compare against.
