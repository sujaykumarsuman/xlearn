# Sprint ds-m1-01 — Design M1: coach states, course nav, revision v2 (AB01–AB03)

> **Milestone:** M1 — spine (the boards M1b builds against) · **Track:** design · **Order:** 2
> **Prereqs:** none (parallel with [mi-01](sprint-mi-01.md) and [m1-01](sprint-m1-01.md))
> **Unblocks:** [m1-03](sprint-m1-03.md) (AB02; the first M1b sprint, gated on this freeze) · [m1-10](sprint-m1-10.md) (AB01 key frames) · also consumed by [m1-06](sprint-m1-06.md) (AB03) and [m1-07](sprint-m1-07.md) (AB01 D27 / mode / cap frames) · [ds-p-01](sprint-ds-p-01.md) (its entry gate: AB02 frozen, plus the board index and `board.css`) · [ds-m6a-01](sprint-ds-m6a-01.md) and [ds-m6a-02](sprint-ds-m6a-02.md) (their entry gate: the board index and `board.css` on `main`)
> **Release action:** **PR, stop for owner review (design)** — the agent never merges; the owner's merge (or explicit approval) is the freeze
> **Calendar:** week 1 (2026-09-25 → 10-02) · owner event `ev-freeze-ds-m1-01` (review + merge) before m1-03 starts (week 2–3)
> **Execute with:** [`../prompts/prompt-ds-m1-01.md`](../prompts/prompt-ds-m1-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Board index for every v2 board (written once) + shared board chrome | X | ⬜ |
| 2 | Board scaffolding (conventions every board follows) | X | ⬜ |
| 3 | AB01 coach states | X | ⬜ |
| 4 | AB02 course nav | X | ⬜ |
| 5 | AB03 revision v2 | X | ⬜ |
| 6 | Self-review against the brief + screenshots (1440 px, 390 px) | X | ⬜ |
| 7 | Open the design PR and STOP (owner review = the freeze) | X · O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly. **Design exception:** this sprint's PR does **not** edit
> [`../status.md`](../status.md) (the PR may stay open for days). The PR sets tasks 1–6 ✅ and task 7 🔄
> ("PR #N open — awaiting owner review").
> **Close-out after the owner's merge** (four edits, each idempotent — skip any already done), made by the first
> consuming build sprint, [m1-03](sprint-m1-03.md), in its own status update:
> 1. this file: task 7 ✅ ("merged by the owner, PR #N, <date>") and _Overall_ ✅;
> 2. `../status.md` Sprint board: the ds-m1-01 row ✅;
> 3. `../status.md` Artboards: AB01, AB02, AB03 → "frozen (PR #N, <date>)";
> 4. `../status.md` owner events: `ev-freeze-ds-m1-01` ✅.
>
> Any later consumer ([m1-06](sprint-m1-06.md) for AB03, [m1-10](sprint-m1-10.md) / [m1-07](sprint-m1-07.md) for AB01) that
> finds one of these still unset applies it the same way. Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] none (no prerequisite beyond `depends_on`, which is empty)
- [ ] Parallel-sessions check: no open PR already adds `design-system/screens/v2/index.html` or `board.css`
      (`gh pr list --state open`, `git worktree list`, ListAgents) — this sprint writes both, once

## Goal

Draft the three M1 artboards — **AB01** coach states, **AB02** course nav, **AB03** revision v2 — as static
HTML on [`design-system/theme.css`](../../../design-system/theme.css) under `design-system/screens/v2/`, so
M1b's UI sprints ([m1-03](sprint-m1-03.md), [m1-06](sprint-m1-06.md), [m1-10](sprint-m1-10.md),
[m1-07](sprint-m1-07.md)) build against **owner-approved** designs. Every frame carries final copy, every
state, and a behaviour-notes aside citing the decision it implements. Because this is the **first** v2
design sprint, it also writes the **board index for all AB01–AB30** and the shared board chrome, once,
so the parallel design PRs ([ds-m2-01](sprint-ds-m2-01.md), [ds-l-01](sprint-ds-l-01.md), later ones)
never conflict. BP3 (owner decision 2026-09-24): agents draft every board; the owner only reviews and approves.

## Scope

**In**
- `design-system/screens/v2/index.html` — a static row and link for **every** board AB01–AB30 (AB23 dropped,
  AB31 outline), grouped by milestone; plus the superseded-v1 table ([rollout §9](../rollout-plan.md#9-artboards-by-milestone)).
- `design-system/screens/v2/board.css` — board chrome only (canvas, frame labels, notes aside, the `.xl-app`
  fill override). Never redefines a `ds-*`/`xl-*` class.
- Boards `AB01-coach-states.html`, `AB02-course-nav.html`, `AB03-revision-v2.html` with every frame in task 3–5.
- PNG screenshots of each board at 1440 px and 390 px under `design-system/screens/v2/shots/`.
- The design PR, stopped for owner review.

**Out**
- Any `web/` code — the builds are [m1-03](sprint-m1-03.md) (AB02), [m1-06](sprint-m1-06.md) (AB03),
  [m1-10](sprint-m1-10.md) + [m1-07](sprint-m1-07.md) (AB01).
- AB02 at **full** multi-course fidelity and AB05 → [ds-p-01](sprint-ds-p-01.md) (P). AB02 here is DSA-parity plus
  one low-fidelity second-course frame.
- Every other board (AB04–AB30) → its own design sprint ([ds-m2-01](sprint-ds-m2-01.md), [ds-l-01](sprint-ds-l-01.md),
  [ds-m3-01](sprint-ds-m3-01.md), [ds-m3-02](sprint-ds-m3-02.md), [ds-p-01](sprint-ds-p-01.md),
  [ds-m4-01](sprint-ds-m4-01.md), [ds-m6a-01](sprint-ds-m6a-01.md), [ds-m6a-02](sprint-ds-m6a-02.md),
  [ds-m6b-01](sprint-ds-m6b-01.md)). This sprint only lists them in the index.
- Shipping boards: preview-only, never imported by `web/`, never embedded, never deployed.
- `docs/v2/status.md` edits (see the Status note).
- Owner design hours beyond review (BP3).

## Tasks

### 1 · Board index + shared chrome (once, for every design sprint) [X]

**`design-system/screens/v2/index.html`** — static HTML linking `../../theme.css` and `board.css`. One table per
milestone; columns: **Board** (id, ★ for the 5 heroes) · **What** (one line) · **Drafted in** · **Consumed by** ·
**Frozen before** · **File** (relative link). Later design PRs **never edit this file**: a board whose file does
not exist yet simply 404s in preview. Freeze state is **not** kept here (it lives in `docs/v2/status.md`).

Canonical file names (the index is the naming source of truth — later design sprints use exactly these; they match the
names each drafting sprint falls back to when it runs before this PR merges, e.g. [ds-m2-01](sprint-ds-m2-01.md)'s
`AB06-public-profile-v2.html` / `AB22-visibility-toggles.html`, so no index link can 404 permanently):

| Board | File | Drafted in → consumed by | Frozen before |
|---|---|---|---|
| AB01 coach states | `AB01-coach-states.html` | ds-m1-01 → m1-10, m1-07 | M1b |
| AB02 course nav | `AB02-course-nav.html` | ds-m1-01 (+ ds-p-01 full fidelity) → m1-03, p-02 | M1b |
| AB03 revision v2 | `AB03-revision-v2.html` | ds-m1-01 → m1-06 | M1b |
| AB04★ A8 Touch | `AB04-touch.html` | ds-m2-01 → m2-01, m2-04 | M2a |
| AB05 catalog + agenda | `AB05-catalog-agenda.html` | ds-m2-01 (+ ds-p-01) → m2-04, p-02 | M2a |
| AB06 public profile v2 | `AB06-public-profile-v2.html` | ds-m2-01 → m2-03 | M2a |
| AB07★ A1 Workspace-Code | `AB07-workspace-code.html` | ds-m3-01 → m3-11, m5-01 | the M3 UI sprint |
| AB08 A2 results dock | `AB08-results-dock.html` | ds-m3-01 → m3-12 | the M3 UI sprint |
| AB09 A4 Problems | `AB09-problems.html` | ds-m3-02 → m3-12 | the M3 UI sprint |
| AB10 A5 Arena | `AB10-arena.html` | ds-m3-02 → m3-12 | the M3 UI sprint |
| AB11 degradation badges | `AB11-degradation-badges.html` | ds-m3-01 → m3-12 | the M3 UI sprint |
| AB12 Week/Mistakes/Progress deltas | `AB12-week-mistakes-progress.html` | ds-m3-02 → m3-13 | the M3 UI sprint |
| AB13 A9 Mock-v2 | `AB13-mock-v2.html` | ds-m6a-01 → m6a-05 | M6a |
| AB14 A7 Workspace-Quiz | `AB14-workspace-quiz.html` | ds-p-01 → p-03 | P |
| AB15 go-concurrency multi-file + race | `AB15-go-concurrency-race.html` | ds-p-01 → p-03 | P |
| AB16★ A3 AI suggestion / dispute | `AB16-ai-suggestion-dispute.html` | ds-m4-01 → m4-06 | the M4 UI sprint |
| AB17 pointer notes | `AB17-pointer-notes.html` | ds-m4-01 → m4-06 | the M4 UI sprint |
| AB18 AI allowance + consents | `AB18-ai-allowance-consents.html` | ds-m4-01 → m4-06 | the M4 UI sprint |
| AB19★ invite acceptance | `AB19-invite-acceptance.html` | ds-l-01 → l-05 | L-A |
| AB20 privacy notice / terms | `AB20-privacy-notice.html` | ds-l-01 → l-05 | L-A |
| AB21 erase account | `AB21-erase-account.html` | ds-l-01 → l-02 | **L-E** |
| AB22 visibility toggles | `AB22-visibility-toggles.html` | ds-m2-01 → m2-03 | M2a |
| ~~AB23~~ owner admin page | — (**dropped**: admin is CLI-only, D33) | — | — |
| AB24 setup + consent + pre-flight + $ cap | `AB24-mock-setup-preflight.html` | ds-m6a-01 → m6a-05 | M6a |
| AB25 live HUD (text) | `AB25-live-hud-text.html` | ds-m6a-01 → m6a-05 | M6a |
| AB26 grace / paused / resume | `AB26-grace-paused-resume.html` | ds-m6a-02 → m6a-06 | M6a |
| AB27 debrief + proposal | `AB27-debrief-proposal.html` | ds-m6a-02 → m6a-06 | M6a |
| AB28 accessibility settings | `AB28-accessibility-settings.html` | ds-m6a-02 → m6a-06 | M6a |
| AB29 voice pre-flight + notices | `AB29-voice-preflight-notices.html` | ds-m6b-01 → m6b-03 | M6b |
| AB30★ voice live HUD | `AB30-voice-live-hud.html` | ds-m6b-01 → m6b-03 | M6b |
| AB31 A6 Workspace-Canvas | — (**outline only**, v2.2+, [m6c-03](sprint-m6c-03.md)) | — | — |

Links in "Drafted in / Consumed by" point at `../../../docs/v2/sprints/sprint-<id>.md`. The index also carries:
the superseded-v1 table (Problem → AB07; Revision → AB03, AB04; Mock → AB13; Dashboard, Week → AB05, AB12;
Mistakes, Progress → AB12; Catalog → AB05; Settings → AB18, AB21, AB22; Auth → AB19) linking `../<Name>.dc.html`,
and a short **Conventions** block (task 2) so every later design sprint finds the rules in one place.

**`design-system/screens/v2/board.css`** — board chrome, prefixed `bd-` so it can never collide with `ds-*`/`xl-*`:
`.bd-canvas` (page padding, dark background from `--ds-bg`), `.bd-section` (milestone/section heading),
`.bd-frame` + `.bd-frame__label` (frame id `AB01-F3`, title, state), `.bd-grid` (wrapping grid for panel-sized
frames side by side), `.bd-notes` (behaviour-notes aside), `.bd-narrow` (a 390 px-wide frame container), and the
fill override `.bd-frame .xl-app { width: 100%; height: auto; min-height: 720px; }` (analogous to the real app's
override in `web/src/styles/app.css`, which uses `width: 100%; height: 100vh; min-height: 0` to fill the viewport; a
board frame instead grows with its content, so it reflows rather than being a fixed 1440×1024 artboard).
Tokens only — no raw hex except where `theme.css` itself uses one.

### 2 · Board scaffolding (the conventions) [X]

Every board, in this sprint and every later design sprint:
- One file per board under `design-system/screens/v2/`, named per the task 1 table; `<link rel="stylesheet" href="../../theme.css">`
  then `board.css`, plus the Google Fonts `<link>` the v1 boards use (Inter, JetBrains Mono).
- Reuses `theme.css` tokens and components **verbatim** (dark "landscape console"; `ds-btn`, `ds-card`, `ds-badge`,
  `ds-modal`, `ds-seg`, `ds-toggle`, `xl-side`, `xl-pathsw`, `xl-nav`, `xl-topbar`/`xl-crumb`, `xl-touch`, `xl-pat`,
  `xl-lock`, `xl-coach`/`xl-fab`, `xl-timer`, …). Difficulty: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`.
  Board-specific layout goes in the board's own `<style>`; it never overrides a `ds-*`/`xl-*` rule.
- **Frames:** full-screen frames stack vertically; panel-sized states (e.g. coach panel states) sit **side by side**
  in a `.bd-grid`. Each frame has a label `ABnn-Fk · <state>` and **final copy** (no lorem ipsum, no "TBD").
- **Behaviour-notes aside** per frame: timers, gates, error codes (exact strings, e.g. `409 assist_confirm_required`),
  the decision it implements (D-number / ADR §), a11y (focus order, contrast ≥ 4.5:1 on text, keyboard shortcuts,
  `aria-live` for status changes, reduced motion), and the **< 1024 px** layout intent.
- **Narrow frames:** each board ends with a `.bd-narrow` section showing the 390 px layout of its key frames.
- Static HTML only: no JS runtime, no `support.js`, no `<script type="text/x-dc">` (v1 `.dc.html` boards need a
  canvas runtime; these must open straight from disk or any static server). v1 `.dc.html` files are references only.
- **Never leak withheld data** ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule)):
  no pattern chip while an item is live (open attempt, due or live touch, never solved), no hidden inputs, no answers.
- Preview-only: never imported by `web/`, never `//go:embed`ded, never shipped.

### 3 · AB01 coach states [X]

`AB01-coach-states.html` — the BYO coach under the v2 contract. Consumed by [m1-10](sprint-m1-10.md) (F11–F15) and
[m1-07](sprint-m1-07.md) (F1–F10). Sources: [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes)
(D27), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2), [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D18 Assisted).

| Frame | State | Must show |
|---|---|---|
| F1 | Panel, `attempt` mode (open counted attempt on this problem) | Header **"Your AI coach (your key)"**; context chip `problem · #16 3Sum`; mode chip "Attempt · hints only — no pattern or solution"; input enabled |
| F2 | **D27 assist confirm** (first send while that attempt is open) | Server returns `409 assist_confirm_required`; confirm card: "Ask the coach about this problem? Using any AI during a counted attempt caps it at **Assisted**." Buttons **[Ask — cap at Assisted]** (primary) **[Cancel]**; re-send carries `assist_ack:<attempt_id>` |
| F3 | Assist recorded | Panel note "Coach used on this attempt · capped at Assisted"; the Problem HUD chip mirrors it (D18: coach help ⇒ Assisted) |
| F4 | Assist not recorded (503, fail closed) | "We couldn't record coach use on your attempt, so nothing was sent. Try again." Nothing is forwarded. Notes: `503 assist_unavailable` (the assist record failed) and `503 coach_state_unavailable` (the mode check couldn't reach practice/review/assessment) — both fail closed ([m1-07](sprint-m1-07.md)) |
| F5 | Locked (`409 coach_paused`) during a live touch or mock | Banner "The coach is paused while a revision touch or mock is live. It's back when you finish."; input disabled |
| F6 | `review` mode (concluded, no touch due) | Mode chip "Review"; normal chat |
| F7 | `general` mode (not an item page) | Mode chip "General"; note that live items stay off-limits (the guard line is server-side) |
| F8 | Typed 429s (L18) | Three variants: 20 msg/min ("You're sending quickly — try again in 12 s"), 2 concurrent streams ("One reply at a time — wait for the current reply"), **300/day** ("You've reached today's 300-message limit on your key. It resets at 00:00 UTC (5:30 AM your time)." — the day is a **UTC** day; the local time is computed from `Retry-After`). Notes list the exact codes [m1-07](sprint-m1-07.md) returns: `429 coach_rate_limited` (`Retry-After` = seconds to the next token), `429 coach_busy` (`Retry-After: 5`), `429 coach_daily_cap` (`Retry-After` = seconds to the next UTC midnight); all distinct from the v1.5.1 `provider_limited` 429 (F10) |
| F9 | Cut-short reply marker | Reply ends "Reply cut short (length limit)" + **[Continue]** (`stop_reason` = max tokens) |
| F10 | Provider limited — key stays enabled | Quota / billing / spend / rate limit: "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry." Key toggle stays on |
| F11 | Model access error (`ErrModelAccess`) | "This key can't use `<model>`. Pick another model." with the switcher open |
| F12 | Model switcher from the **server catalog** (`GET /api/coach/models`) | Anthropic / OpenAI only (no Google); per model: name, capability tags (Interview, Voice), price per MTok with "as of <date>", **Recommended** badge; "Custom model id…" row → "cost unknown" chip; **no base-URL field** |
| F13 | Settings → AI coach keys | Masked keys per provider; **per-feature defaults** "Coach: <model>" / "Interview: <model> (needs a realtime-capable model)"; "This month on your keys: 212 messages · 1.4 M tokens · ≈ $3.10 (estimate)" |
| F14 | Onboarding step 3/4 (optional BYO) | Anthropic / OpenAI only; **[Connect & finish] [Skip for now]**; pre-M4 copy "Optional: bring your own AI coach. Your key is stored encrypted; add it later in Settings." plus a flagged **from-M4** variant adding "xLearn AI already reviews your submissions — no key needed." |
| F15 | No usable key (parity) | v1 empty state under the new name, routing to Settings |

Naming reference strip: **"Your AI coach (your key)"** (BYO) vs **"xLearn AI (included)"** (platform AI, arrives in M4) — ADR-0031 §7.

### 4 · AB02 course nav [X]

`AB02-course-nav.html` — manifest-driven course navigation with **DSA pixel parity**. Consumed by
[m1-03](sprint-m1-03.md); full multi-course fidelity in [ds-p-01](sprint-ds-p-01.md). Sources:
[ADR-0026 §5](../../adr/0026-per-course-extensibility-model.md#5-screens-routing-today),
[t0 §7](../research/t0-extensibility-frame.md#7-screens--routing), ADR-0025 (`/u/` profiles),
[ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (new static segments join the slug guard).

| Frame | State | Must show |
|---|---|---|
| F1 | DSA Roadmap `/xlearn/dsa` | Sidebar from the manifest `nav` groups — Today · Roadmap · Problems · Progress; cap "Practice loop": Revision · Mistakes · Mock interview (live badges) — identical to v1 (`web/src/nav.ts`); topbar curriculum selector; crumbs `xlearn / dsa` |
| F2 | DSA Today `/xlearn/dsa/dashboard` | Same chrome; crumbs `xlearn / dsa / dashboard`; proves the chrome is course-agnostic |
| F3 | Course switcher open | Enrolled active courses (DSA ✓ current); a **`preview`** course with a "Preview" badge (`ds-badge--violet`), visible **only** to the owner/tester cohort; coming-soon courses muted, not selectable, lock icon + "<title> · Coming soon" (v1 `PathSwitcher.tsx` parity); v1's footer link **"Browse all paths"** to Catalog, unchanged (`PathSwitcher.tsx:91`) |
| F4 | `coming_soon` course `/xlearn/system-design` | Catalog teaser: title, summary, "Coming soon" badge, **[Back to catalog]**; no sidebar, no start button |
| F5 | Active, not enrolled | Start-path call to action (F002 parity) |
| F6 | Unknown course `/xlearn/nope` | The v1 NotFound page unchanged (eyebrow `404`, "Page not found", "That route doesn't exist in xLearn.", button **[Back to Today]** → the default course's dashboard, `/dsa/dashboard` as in v1); no hint whether the slug exists |
| F7 | Breadcrumbs | `/:course/week/3` → `xlearn / dsa / week/3`; concept with `?week=3` → `xlearn / dsa / week 3 / concept/two-pointers`; problem → `xlearn / dsa / problems / #16` (parity with `buildCrumbs`) |
| F8 | Second course (low fidelity, illustrative) | A go-concurrency nav rendered from its manifest: its own labels and item noun, no "Mock interview" when the manifest has no mock — shows the nav is data |
| F9 | Narrow | Collapsed 66 px icon rail (`.xl-collapse-cb`), the switcher as a full-width sheet |

Behaviour notes must list the reserved top-level segments that can never be a course slug:
`u, auth, settings, api, assets, healthz, readyz, privacy` (`.well-known` is excluded by the slug regex).

**DSA parity is the default; copy changes are proposals.** The M1 exit criterion allows only the D27 confirm, D31 and the
429s as visible changes ([rollout §3](../rollout-plan.md#3-milestone-map)), so F1–F3 and F6 keep v1's copy. Any visible
change you would still recommend (e.g. "Browse all courses →" instead of "Browse all paths", or NotFound's button
re-targeted to **[Back to catalog]**) is drawn as a clearly labelled **alternative** beside the parity frame and listed
under "Decisions to confirm" in the PR — it becomes an intended delta only if the owner approves it at the freeze. F4, F5
and F8 are new states (no v1 equivalent), not parity deltas.

### 5 · AB03 revision v2 [X]

`AB03-revision-v2.html` — the Revision queue with manifest-driven format badges and the **withheld pattern**.
Consumed by [m1-06](sprint-m1-06.md) (badges are placeholders until M2a's touch formats). Sources:
[PRD §5.1–5.2](../../prd/xlearn-v2-prd.md#51-method-universal-vs-per-course) (R-SR2), D3,
[t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise) (fixes the chip at `Revision.tsx:235-237`),
[t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course), D4 (R-SR5 per course).

| Frame | State | Must show |
|---|---|---|
| F1 | Due queue (DSA) | Rows with format badges **"Re-solve · 20:00"** (L1–3) and **"Re-solve · mock conditions · 20:00"** (L4–5, violet `mock` badge parity), five-touch dots, day label, difficulty chip; **no pattern chip** — an `xl-lock` "Pattern hidden while due" instead |
| F2 | Active re-solve | 20:00 timer; the three v1 criteria tiles (Named the pattern < 2 min · Solved in timer · Complexity stated); pattern still hidden (the learner names it) |
| F3 | Scored pass | Pattern revealed; "Advances to Day 7" |
| F4 | Scored fail | Reset to Day 1; pattern revealed; "A mistake entry is open" link |
| F5 | Recall badge (all-courses view, low fidelity) | A non-DSA row with **"Recall · ~5 min"** — the label comes from the band (`revision.bands[*].label`) and the minutes from `plan.est_minutes["touch_" + band.format]` (the manifest's single minutes source, [m1-01](sprint-m1-01.md)), not code |
| F6 | Empty / caught up | "Nothing due today. Next review: Thu, Oct 8." |
| F7 | Overdue | Banner "3 reviews overdue — new DSA problems wait until they're done." (R-SR5 blocks new work **per course**) |
| F8 | All-courses toggle | Course-scoped by default; "All courses" merges due touches, oldest first, each row tagged with its course |
| F9 | Narrow | Queue as cards; criteria tiles stacked |

### 6 · Self-review against the brief + screenshots [X]

Walk every frame against its cited ADR/PRD decision and the [rollout §9](../rollout-plan.md#9-artboards-by-milestone)
board list. Leak check: no pattern chip on a due/active touch or open attempt, no hidden inputs, no answers anywhere.
Check copy against D27 (per-problem cap, honour copy), D18 (Assisted), L18 (20/min, 2 streams, 300/day) and D31
(nothing mock-related here, but no board may show a public mock average). Screenshot every board at **1440 px** and
**390 px** (full page) into `design-system/screens/v2/shots/AB0n@1440.png` / `AB0n@390.png` (keep each ≲ 500 KB).

### 7 · Open the design PR and STOP [X · O]

Branch `design/ds-m1-01`; conventional commit `docs(design): v2 board index + AB01–AB03 (M1)` ending with the
attribution lines. Open a PR titled `docs(design): AB01 AB02 AB03 — M1 boards (+ v2 board index)` with the screenshots
embedded (`…/blob/design/ds-m1-01/design-system/screens/v2/shots/AB01@1440.png?raw=true`) and a per-board frame list.
The PR body also carries a **"Decisions to confirm"** list: every ambiguity you resolved (e.g. D27 confirm placement)
and every visible change against v1 you propose for DSA (AB02's labelled alternatives), each as an explicit yes/no for
the owner — without a yes, the parity frame is the frozen one.
**Do not merge.** Stop for owner review (owner event `ev-freeze-ds-m1-01`). Merging — by the owner, or by an agent only
on the owner's explicit approval in chat — **is the freeze**. Requested changes are made on the same branch.

## Acceptance criteria

- [ ] `design-system/screens/v2/index.html` lists AB01–AB31 (AB23 dropped, AB31 outline) with the task 1 file names and links; `board.css` exists and defines only `bd-*` classes plus the `.xl-app` fill override.
- [ ] Every frame listed for AB01 (F1–F15), AB02 (F1–F9) and AB03 (F1–F9) is present with final copy, its state, and a behaviour-notes aside citing its decision.
- [ ] No board leaks withheld data (pattern chip only once a touch/attempt is concluded or from the hint stage; no hidden inputs, no answers).
- [ ] Boards open from disk or a static server with no JS runtime; `theme.css` is linked, not copied or overridden.
- [ ] PR open with 1440 px and 390 px screenshots of each board; **not merged by the agent**.

## Release

**PR, stop for owner review (design).** Nothing deploys: boards are preview-only files. The owner's merge is the
freeze (`ev-freeze-ds-m1-01`); [m1-03](sprint-m1-03.md) cannot start until it lands (its entry gate).

## Definition of Done

PR open with the index, `board.css`, AB01–AB03 and screenshots · every acceptance box ticked · this sprint's Status
table updated in the PR (tasks 1–6 ✅, task 7 🔄 awaiting review) · no `web/` or `docs/v2/status.md` change · the
agent stops at the open PR. (Done-done — task 7 and _Overall_ ✅, the Sprint-board and Artboards rows, and
`ev-freeze-ds-m1-01` — is the Status note's four-edit close-out, done by m1-03 once the owner has merged.)

## Risks / watch-outs

- **A board contradicting an Accepted decision** (D15/D16/D17/D18/D27/D31): cite the decision per frame; when a
  decision is ambiguous, draft the conservative reading and flag it in the PR body for the owner.
- **Parity drift in AB02:** DSA must stay pixel-identical to v1 (M1 exit "golden = v1"). Draw F1/F2 from the live
  app's structure (`web/src/nav.ts`, `Sidebar.tsx`, `PathSwitcher.tsx`, `Topbar.tsx`, `NotFound.tsx`), not only the older
  `.dc.html`; copy included — a reworded label is a visible change and goes to "Decisions to confirm", not the parity frame.
- **Index conflicts:** later design PRs must not edit `index.html` or `board.css`; if a later board needs a different
  file name, that is a bug in that sprint, not a reason to edit the index.
- **Copy that promises M4 features early** (F14): keep the platform-AI line in a flagged "from M4" variant.
- **Screenshots in git:** keep PNGs small; they become part of the frozen reference build sprints compare against.
