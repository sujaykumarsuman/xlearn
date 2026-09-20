# Sprint 04 — Week & Concept screens

> **Milestone:** M2 — browse the full curriculum   ·   **Design phase:** D2 Content
> **Prereqs:** [S03](sprint-03.md)   ·   **Unblocks:** [S05](sprint-05.md)
> **Execute with:** [`../prompts/prompt-s04.md`](../prompts/prompt-s04.md) — one prompt, one session.

## Status

_Overall:_ 🔄 Code-complete & green (build/test/lint + `sqlc diff` pass); shipping.

| # | Task | Status |
|---|------|--------|
| 1 | Week screen | ✅ |
| 2 | Concept screen | ✅ |
| 3 | BFF week aggregation | ✅ |
| 4 | Navigation + polish | ✅ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Complete the read-only browse path for the DSA curriculum. After S03 a learner can reach Catalog and
Roadmap; this sprint adds the last two content views — **Week** (thesis, concept links, the filtered
problem list) and **Concept** (pattern reading, when-to-use callouts, code template) — plus the gateway
**week-aggregation** endpoint that stitches curriculum content to per-user progress. Per-user practice
and review state does not exist yet (it lands in S05/S06), so this sprint ships a **stable placeholder
state** with the exact shape those services will later fill. Delivering all four screens reaches **M2 —
browse the full curriculum**.

## Scope

**In**
- Week view at SPA route `/xlearn/dsa/week/:n`, wired from `design-system/screens/Week.dc.html`: eyebrow
  + title, the "What this week is really about" thesis card, the Concepts grid, and the Problems list
  with difficulty tokens, pattern chip, reinforcement chip, five-touch dots, status chip, and the
  All/Guided/Reinforcement filter.
- Concept view at `/xlearn/dsa/concept/:slug`, wired from `design-system/screens/Concept.dc.html`:
  reading from `body_md`, the when-to-use callout from `when_to_use_md`, and the `code_template` block.
- Gateway BFF endpoint `GET /paths/{slug}/weeks/{n}` (**agg**) returning curriculum week + concepts +
  problems, plus an explicit placeholder `userState` object with a stable shape.
- Gateway passthrough `GET /concepts/{slug}` (pure curriculum read).
- Navigation Roadmap -> Week -> Concept -> Problem(stub), breadcrumbs/eyebrows, loading + empty states.

**Out (later sprints)**
- Real five-touch dots + solve status — populated by practice/review ([S05](sprint-05.md)/[S06](sprint-06.md)).
- The guided Problem workspace — [S05](sprint-05.md); this sprint only links to its stub route.
- Any mutation of curriculum, events, or new schema (curriculum stays read-only, seeded in S03).

## Tasks

### 1 · Week screen
Build a real React/TS view for `/xlearn/dsa/week/:n` from the `Week.dc.html` artboard (layout/copy only —
the artboard is not runnable). Render, top to bottom: the eyebrow (`xl-eyebrow`, "Week N of 16 · Phase X
<theme>"), the `h1` from `week.title`, the "What this week is really about" thesis card
(`ds-card--teal`, from `week.thesis`), the **Concepts** grid (cards from the `week_concept` join in
schema `curriculum`, each linking to `/xlearn/dsa/concept/:slug`), and the **Problems** list. Each
problem row comes from `curriculum.problem` filtered by `week_n` and renders the difficulty token
(`xl-diff--easy|med|hard` mapping `difficulty` easy/med/hard to green `--ds-ok` / amber `--ds-warn` /
red `--ds-err`), the `xl-pat` pattern chip, a reinforcement chip when `is_reinforcement`, the five-touch
dots (`xl-touch__d`), a status chip, and the `ds-seg` All/Guided/Reinforcement filter. The right rail
holds the "Week N progress" meter and the "Suggested for today" panel. Five-touch dots and solve status
read from the aggregation's placeholder `userState` (Task 3): all dots in the neutral/empty state, status
"Available", progress meter 0/core — **render an honest empty/available state, never faked data**. Reuse
`theme.css` verbatim; see [`../../architecture/data-model.md`](../../architecture/data-model.md) for the
`curriculum` tables.

### 2 · Concept screen
Build `/xlearn/dsa/concept/:slug`, backed by `GET /concepts/{slug}` (a pure curriculum read). From the
`Concept.dc.html` artboard, render `concept.title`, the lede, the "When should I think about this?"
callout (`ds-card--teal`) from `when_to_use_md`, the body reading from `body_md` (a sanitized markdown
renderer -> the `.cn` article styles: h2/p/inline-`code`, plus the `ds-card--violet` "Watch out"
callout), and the reusable code template (`xl-code`) from `concept.code_template`. The right rail carries
the "Reach for it when..." checklist, the "Practice this pattern" problem links, and the "Practice now"
primary button (-> Problem stub). Note: `code_template` is a **single** field and v1 is Go-first —
render the Go template and either disable or stub the artboard's C++ tab; do not fabricate a C++
template. Endpoints and fields per [`../../architecture/api.md`](../../architecture/api.md) and
[`../../architecture/data-model.md`](../../architecture/data-model.md).

### 3 · BFF week aggregation
Implement `GET /paths/{slug}/weeks/{n}` (**agg**) in the gateway. Server-side, fan out to curriculum for
the week (`week.thesis`, its phase), its concepts (`week_concept`), and its problems; then merge per-user
practice state and review five-touch state. Because practice/review are not built until S05/S06, the
gateway assembles the curriculum data plus an explicit placeholder `userState` with a **stable shape**:
per problem `{ status: "available", lastOutcome: null, currentTouch: 0, touches: [5 x { level, dueDate:
null, result: "none" }] }` (touch levels 1..5 = Day 1/3/7/21/45), and a week rollup `{ solved: 0,
coreTotal, byDifficulty: { easy, med, hard }, populated: false }`. The `populated: false` flag marks the
block as un-sourced so S05/S06 fill the same fields without a client change. `GET /concepts/{slug}` is a
straight passthrough to curriculum. Both run behind the gateway session cookie -> minted internal RS256
JWT ([ADR-0006](../../adr/0006-authn-authz.md)); aggregation lives in the gateway precisely because
services cannot cross-join ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). No new schema,
no events. Contract table: [`../../architecture/api.md`](../../architecture/api.md); service roles:
[`../../architecture/services.md`](../../architecture/services.md).

### 4 · Navigation + polish
Wire the whole read path Roadmap -> Week -> Concept -> Problem(stub): Roadmap week rows link to
`/xlearn/dsa/week/:n`, Week concept cards to `/xlearn/dsa/concept/:slug`, and every problem row plus
"Practice now" to `/xlearn/dsa/problem/:id` (the S05 stub route — a placeholder view, not the workspace).
Breadcrumbs/eyebrows correct on both screens (Week: `xlearn / dsa / week/N`; Concept: `xlearn / dsa /
week N / concept/slug`). Add loading skeletons and empty states per the DS (the artboards' `sc-for`
placeholder counts show the intended skeleton). Confirm the S02 behaviour still holds: unauthenticated
`/xlearn/api/*` -> `401` -> SPA redirects to login.

## Acceptance criteria
- [ ] A user can browse Catalog -> Roadmap -> Week -> Concept with real seeded content (**M2 — browse the full curriculum**).
- [ ] Week shows thesis + concepts + the filtered problem list, with placeholder five-touch/solve state (no faked data).
- [ ] Concept shows reading (`body_md`) + when-to-use (`when_to_use_md`) + code template.
- [ ] `GET /paths/{slug}/weeks/{n}` returns a stable shape with a placeholder `userState` object (per-problem `touches[5]` + week rollup, `populated: false`).
- [ ] The Problem link routes to the S05 stub; both screens match the Week/Concept artboards.

## Definition of Done
CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs
- **Freeze the aggregation shape now.** Field names, the 5-entry `touches` array, and the week rollup must
  stay stable so S05/S06 populate them without a frontend rewrite; renaming keys later is the expensive mistake.
- **Do not fake solve/revision state.** Render honest empty/available states — a green dot or a "solved"
  chip here would be a lie until practice/review exist.
- **`code_template` is one field and v1 is Go-first.** Render Go; stub or disable the artboard's C++ tab
  rather than inventing a C++ template.
- **The `.dc.html` artboards are not runnable.** Read them for layout/copy/interaction only; ship real
  React against `theme.css` and do not port the `sc-for`/`{{...}}` template syntax.
- **Keep aggregation server-side.** Services cannot cross-join ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)); do not leak curriculum/practice internals to the client beyond the documented shape.
