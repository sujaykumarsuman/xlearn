# xLearn — design system & screen reference

High-fidelity design reference for **xLearn**, a guided-learning platform. This folder is the
source of truth for *how xLearn should look and behave* so the app can be built in phases.

- **Live prototype (Claude Design canvas):** https://claude.ai/artifact/D8LqvCzerZmDK4WT6i1FdH
  (private to the owner; 12 clickable screens — open in Play mode to click through).
- **Brand / design system origin:** the *sujaykumar.dev "landscape console"* look, extracted from
  `github.com/sujaykumarsuman/sujaykumar-design-system` (React + TS component lib). Tokens and
  component styles below are ported from that repo's `src/tokens.ts` + `src/styles.css`.

## What's in here

| Path | What it is | Reuse |
|------|-----------|-------|
| `theme.css` | The full design system: `--ds-*` tokens + `ds-*` components + `xl-*` app shell & signature components. | **Directly reusable** — drop into the real app, or translate the tokens into your framework's theme. |
| `screens/*.dc.html` | The 12 screens as self-contained artboards. | **Design reference** for layout, structure, copy, and interaction. See note below. |
| `canvas.json` | Canvas index — board list, positions, titles, launch. | Reference for the screen inventory. |

> **Note on `.dc.html`:** these are Claude Design "artboards" — HTML + a small `<script type="text/x-dc">`
> logic block that only runs inside the Design canvas runtime (they reference `./support.js`). They are
> **not standalone-runnable** in a browser. Read them for markup/styling/interaction intent; don't ship
> them as-is. `theme.css` *is* plain CSS and ships directly.

## Design tokens (from the sujaykumar design system)

Dark-first, one committed theme. All defined on `:root` in `theme.css` as `--ds-*`.

**Surfaces:** `--ds-bg #0b0d10` · `--ds-panel #12151c` · `--ds-panel-2 #171b23` · `--ds-line #232a35` · `--ds-line-2 #2d3542` · inset `#0f131a` · input `#0e1219`
**Text:** `--ds-text #e6e9ef` · `--ds-dim #9aa4b2` · `--ds-muted #616b7a`
**Accents:** `--ds-teal #35d0c0` (brand/primary) · `--ds-violet #9b8cf0` (secondary / patterns) · `--ds-info #6aa6ff`
**Semantic = difficulty:** `--ds-ok #57d39a` = **Easy** · `--ds-warn #f0b429` = **Medium** · `--ds-err #f26d6d` = **Hard**
**Radii:** `--ds-radius-sm 6px` · `--ds-radius 9px` · `--ds-radius-lg 12px`
**Type:** `--ds-font-sans` = Inter → system fallback · `--ds-font-mono` = JetBrains Mono → system fallback (mono is used for identifiers, timers, numbers, metrics). Fonts load per-screen via a Google Fonts `<link>`.

## Component classes

- **From the DS (`ds-*`):** `ds-btn` (`--primary`/`--secondary`/`--ghost`, `--sm`/`--lg`/`--block`), `ds-card` (`--teal`/`--violet`/`--interactive`), `ds-chip`, `ds-badge` (`--ok`/`--warn`/`--err`/`--info`/`--violet`), `ds-dot` (StatusDot), `ds-tabs`/`ds-tab`, `ds-seg`/`ds-seg__btn`, `ds-meter`, `ds-input`/`ds-field`, `ds-toggle`, `ds-avatar`, `ds-modal`, `ds-mono`.
- **App shell (`xl-*`):** `xl-app` (1440×1024 frame), `xl-side` (collapsible sidebar; pure-CSS `.xl-collapse-cb:checked` toggle → 66px icon-only rail), `xl-pathsw` (path switcher), `xl-nav`/`xl-nav__item`, `xl-topbar`/`xl-crumb`/`xl-cmdk` (pill ⌘K search)/`xl-topbar__avatar`, `xl-content`, `xl-panel`, `xl-stat`, `xl-page-h`, `xl-eyebrow`.
- **Signature components:** `xl-touch`/`xl-touch__d` (five-touch dots: `--pass` filled ✓ / `--due` ring / `--fail` crossed / `--mock` dashed violet), `xl-diff` (`--easy`/`--med`/`--hard`), `xl-pat` (pattern chip), `xl-streak`, `xl-timer`, `xl-coach`/`xl-fab` (AI coach), `xl-kbd`, `xl-lock`, `xl-table`, `xl-code` (syntax-tinted code block).

## Screens & routes

Hosted at `projects.sujaykumar.dev/xlearn`. ★ = hero screens (build first).

| Screen file | Route | Notes |
|-------------|-------|-------|
| `Catalog` | `/xlearn` | Multi-path hub. DSA active; others "Coming soon". |
| `Roadmap` | `/xlearn/dsa` | 16-week roadmap: vertical phase/week rail, overall ring. |
| `Dashboard` ★ | `/xlearn/dsa/dashboard` | "Today" — default post-login. Daily plan, revisions due, weak-area, stats. |
| `Week` | `/xlearn/dsa/week/:n` | Week thesis, concept links, problem list w/ five-touch dots + filter. |
| `Concept` | `/xlearn/dsa/concept/:slug` | Pattern reading, "when to reach for it" callouts, code template. |
| `Problem` ★ | `/xlearn/dsa/problem/:id` | Guided 3-pane IDE: statement · editor · timer HUD + gated reveal + outcome logging. |
| `Revision` | `/xlearn/dsa/revision` | Spaced-repetition queue; re-solve auto-scores → advance or reset-to-Day-1. |
| `Mistakes` | `/xlearn/dsa/mistakes` | 8-category journal, weekly weak-area banner, open/closed. |
| `Mock` | `/xlearn/dsa/mock` | 45-min mock: setup → live phase rail → 7-dim rubric radar + trend. |
| `Progress` | `/xlearn/dsa/progress` | Coverage, revision heatmap, pattern mastery, rubric trend, outcome mix. |
| `Settings` | `/settings` | Profile, study budget, API keys (power the coach). |
| `Auth` | sign-up / log-in | OAuth + 3-step onboarding (path → budget → API key). |

## Core mechanics (the product logic to implement)

- **Guided problem flow** — gated stages: **0 Attempt** (15-min timer; only tags, why-it-matters, summary, example, LC/NC links; hints/solution locked) → **1 Hint** (10-min; brute force, key observation, pattern box) → **2 Solution** (optimized approach, steps, dry run, Go/C++, complexity, mistakes, discussion, variations) → **Re-implement from memory** (blank editor) → **log outcome**. Revealing the solution early "owes another attempt in 3 days" (surface on the reveal button).
- **Outcome logging** — Clean / Rough / Assisted / Miss. Anything below Clean opens a Mistake Journal entry (pattern pre-filled, category picker, root-cause) and affects the revision schedule.
- **Five-touch spaced repetition** — every solved problem reviewed on **Day 1 · 3 · 7 · 21 · 45**. Each review is a re-solve from blank on a 20-min timer, **auto-scored**: passes only if pattern named < 2 min, correct within timer, complexity stated. A fail resets to Day 1 + auto mistake entry. Day 21 & 45 are mock conditions. Reviews take priority over new problems.
- **Mistake journal** — columns: Problem · Pattern · Mistake · Root cause · Correct insight · Category · Revisit · Status. 8 categories: Misread · Wrong pattern · Right pattern wrong state · Off-by-one/boundary · Language bug · Complexity misjudged · Communication · Time management. Weekly summary flags the top category. Entry closes after 2 successful revisits (re-opens on a later fail).
- **Mock interview** — 45-min timed, phase rail (0–5 clarify · 5–10 brute force · 10–18 observation→plan · 18–33 code · 33–40 trace+edges · 40–45 complexity+follow-ups). Scored 7 dims × 1–5 = /35 (Communication, Problem understanding, Brute force, Optimisation, Code quality, Edge cases, Complexity). Targets ≥24 by W13, ≥28 by W15, ≥30 pre-interview.
- **AI coach** — persistent bubble → side panel on every screen; reads current-page context (shown as a context chip); Socratic during attempts, reviews code post-solve. Requires the user's own API key (empty state → Settings).

## Domain model (entities)

`Path` (slug, title, status, totals) · `Phase` (weeks, theme) · `Week` (n, title, thesis, concepts) ·
`Problem` (id, title, difficulty, pattern, week, leetcodeUrl, neetcodeUrl, isReinforcement, content sections) ·
`UserProblemState` (status locked/available/attempting/solved, firstSolvedDate, touches[Day1/3/7/21/45] w/ due+result, currentTouch, lastOutcome) ·
`RevisionItem` (problemId, dueDate, touchLevel) · `MistakeEntry` (problemId, pattern, mistake, rootCause, insight, category, revisitDate, status, revisitCount) ·
`MockSession` (setId, date, problem, rubric[7], total/35, notes) · `Account` (profile, budget, timezone, reminders) ·
`ApiKeyConfig` (provider, maskedKey, defaultModel, enabled).

## Sample data used in the prototype

Path "DSA Interview Mastery" · 16 weeks · 4 phases (Fundamentals W1–3 · Core Data Structures W4–8 ·
Advanced Patterns W9–12 · Interview Mastery W13–16) · 151 problems · Go-first. Learner in Week 2, 12-day
streak, 28/151 solved, 4 revisions due, weakest category Off-by-one, mock trend 20→22→24. Hero problem:
`#16 3Sum` (Medium · Two Pointers). Week 1: Contains Duplicate, Valid Anagram, Two Sum, Group Anagrams,
Top K Frequent, Subarray Sum Equals K. Week 2: 3Sum, Longest Substring Without Repeating, Minimum Window
Substring. Later signatures: LRU Cache (W4), Course Schedule (W8), Cheapest Flights Within K Stops (W9),
Coin Change (W10), Largest Rectangle in Histogram (W3).

## Suggested build order (phases)

1. **Foundation** — port `theme.css` tokens/components into the app's styling system; app shell (sidebar + collapse, top bar + avatar, coach scaffold); routing.
2. **Content model + data** — curriculum seed (paths/phases/weeks/problems), then Catalog → Roadmap → Week → Concept (mostly read views).
3. **Guided problem engine** — the Problem screen: editor, stage gating, timer, reveal + penalty, outcome logging → mistake entry.
4. **Spaced repetition** — Revision queue + the five-touch scheduler + auto-scoring; Mistakes journal.
5. **Mock + analytics** — Mock mode + rubric; Progress stats.
6. **Account + coach** — Settings (budget, API keys), then wire the AI coach with the user's key.
