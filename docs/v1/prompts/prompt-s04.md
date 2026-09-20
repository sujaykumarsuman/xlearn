# Prompt — Sprint 04 · Week & Concept screens

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-04.md`](../sprints/sprint-04.md)   ·   **Milestone:** M2 — browse the full curriculum   ·   **Prereqs:** [S03](../sprints/sprint-03.md)

## Read first
- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions: Go services + PostgreSQL + React/TS SPA, `theme.css`, service boundaries, GitOps, ship at session end (AGENT.md land-and-sync).
- [`../sprints/sprint-04.md`](../sprints/sprint-04.md) — the plan this prompt executes (tasks, acceptance, DoD, risks).
- [`../build-plan.md`](../build-plan.md) — where S04 sits (D2 Content, M2) and its dependencies.
- [`../../architecture/services.md`](../../architecture/services.md) — curriculum (read-heavy) + gateway (BFF aggregation) responsibilities and the service<->screen matrix.
- [`../../architecture/api.md`](../../architecture/api.md) — the `GET /paths/{slug}/weeks/{n}` (**agg**) and `GET /concepts/{slug}` contracts + the error/auth envelope.
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — schema `curriculum`: `week`, `concept`, `week_concept`, `problem`.
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — why cross-screen data is aggregated in the gateway (no cross-schema joins).
- [`../../adr/0006-authn-authz.md`](../../adr/0006-authn-authz.md) — session cookie -> minted internal RS256 JWT for the fan-out.
- [`../../adr/0008-frontend-stack.md`](../../adr/0008-frontend-stack.md) — React/TS SPA embedded in and served by the gateway; `theme.css`.
- `design-system/screens/Week.dc.html` and `design-system/screens/Concept.dc.html` — layout/copy/interaction intent (design refs, **not runnable**; do not ship them).
- [`../../../design-system/README.md`](../../../design-system/README.md) — tokens/component classes, the Week/Concept routes, and curriculum mechanics.
- Sibling `../infra`: no new service this sprint — the gateway already ships as `infra/apps/xlearn-gateway.yaml`; its image auto-updates via Flux from `main`.

## Context
After S03, curriculum is seeded and the Catalog (`/xlearn`) + Roadmap (`/xlearn/dsa`) views render real
content through the gateway. This sprint adds the two remaining read views — **Week** and **Concept** —
and the BFF **week-aggregation** endpoint, completing the browse path and reaching **M2**. Per-user
practice/review state does not exist yet (S05/S06), so the aggregation returns a **stable placeholder
`userState`** that those services will later populate without a client change.

## Do this (in order)
> Steps 1-2 (the screens) consume the endpoints defined in step 3; build the UI against the `api.md`
> contract and implement the gateway handlers alongside it. Same four tasks as the plan, in the same order.

1. **Week screen** — Build a React/TS view for route `/xlearn/dsa/week/:n` from `Week.dc.html` (layout/copy
   only). Render: eyebrow (`xl-eyebrow`, "Week N of 16 · Phase X <theme>"), `h1` = `week.title`, the "What
   this week is really about" thesis card (`ds-card--teal`, from `week.thesis`), the **Concepts** grid
   (cards from the `week_concept` join, each -> `/xlearn/dsa/concept/:slug`), and the **Problems** list
   from `curriculum.problem` (filter `week_n`). Each problem row: difficulty token `xl-diff--easy|med|hard`
   (easy->green `--ds-ok`, med->amber `--ds-warn`, hard->red `--ds-err`), `xl-pat` pattern chip, a
   reinforcement chip when `is_reinforcement`, the five-touch dots (`xl-touch__d`), a status chip, and the
   `ds-seg` All/Guided/Reinforcement filter. Right rail: "Week N progress" meter + "Suggested for today".
   Dots/status/meter read from the placeholder `userState` (step 3) — honest empty/available state, 0/core,
   **never faked**. Reuse `theme.css` verbatim (no Tailwind).
2. **Concept screen** — Build `/xlearn/dsa/concept/:slug`, backed by `GET /concepts/{slug}`. Render
   `concept.title`, the lede, the "When should I think about this?" callout (`ds-card--teal`) from
   `when_to_use_md`, the body from `body_md` via a **sanitized markdown renderer** into the `.cn` article
   styles (h2/p/inline-`code` + the `ds-card--violet` "Watch out" callout), and the `xl-code` code block
   from `concept.code_template`. Right rail: "Reach for it when..." checklist, "Practice this pattern"
   links, and the "Practice now" primary button (-> Problem stub). `code_template` is a **single** field
   and v1 is Go-first: render Go and disable/stub the artboard's C++ tab — do not fabricate C++.
3. **BFF week aggregation** — Implement `GET /paths/{slug}/weeks/{n}` (**agg**) in the gateway. Server-side,
   read curriculum for the week (`week.thesis`, phase), its concepts (`week_concept`), and its problems;
   then merge practice + review per-user state. Since those services do not exist yet, assemble the
   curriculum data plus a placeholder `userState` with this **stable shape**: per problem
   `{ status: "available", lastOutcome: null, currentTouch: 0, touches: [5 x { level, dueDate: null,
   result: "none" }] }` (levels 1..5 = Day 1/3/7/21/45) and a week rollup
   `{ solved: 0, coreTotal, byDifficulty: { easy, med, hard }, populated: false }`. Also implement
   `GET /concepts/{slug}` as a straight curriculum passthrough. Both run behind the gateway cookie ->
   internal JWT ([ADR-0006](../../adr/0006-authn-authz.md)); aggregate in the gateway because services
   cannot cross-join ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). No new schema, no events.
4. **Navigation + polish** — Wire Roadmap -> Week -> Concept -> Problem(stub): Roadmap week rows ->
   `/xlearn/dsa/week/:n`, Week concept cards -> `/xlearn/dsa/concept/:slug`, problem rows + "Practice now"
   -> `/xlearn/dsa/problem/:id` (the S05 stub route). Correct breadcrumbs/eyebrows (Week: `xlearn / dsa /
   week/N`; Concept: `xlearn / dsa / week N / concept/slug`). Add loading skeletons + empty states per the
   DS. Confirm unauthenticated `/xlearn/api/*` -> `401` -> login redirect still holds (from S02).

## Constraints
- Reuse `design-system/theme.css` verbatim (tokens/components; difficulty green/amber/red) — **no Tailwind**, no restyling.
- The `.dc.html` artboards need their design-tool runtime — read them for layout/copy only; ship real React/TS and do **not** port the `sc-for`/`{{...}}` syntax.
- **Freeze** the week-aggregation response shape (per-problem `touches[5]` + week rollup, `populated: false`) so S05/S06 fill it without a client change; do **not** fake solve/revision state.
- Aggregation stays **server-side** in the gateway (services can't cross-join, [ADR-0005](../../adr/0005-data-ownership-and-migrations.md)); the SPA talks only to `/xlearn/api`.
- curriculum is read-only here: no new schema/migrations, no events, no outbox this sprint.
- Gateway fan-out uses the session cookie -> minted internal RS256 JWT ([ADR-0006](../../adr/0006-authn-authz.md)).
- Pull-based GitOps: the gateway ships as the existing `ghcr.io/sujaykumarsuman/xlearn-gateway` image; Flux image-automation deploys from `main`. Never `kubectl apply` by hand.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables
- Week view at `/xlearn/dsa/week/:n` and Concept view at `/xlearn/dsa/concept/:slug` (React/TS on `theme.css`), matching the artboards.
- Gateway endpoints `GET /paths/{slug}/weeks/{n}` (**agg**, with placeholder `userState`) and `GET /concepts/{slug}` (passthrough).
- Wired navigation Roadmap -> Week -> Concept -> Problem(stub) with correct breadcrumbs, loading skeletons, and empty states.
- No new service or schema; no infra changes beyond the gateway image auto-deploy.

## Update status
- As each task lands, set its row in [`../sprints/sprint-04.md`](../sprints/sprint-04.md) to ✅ (🔄 while in progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the **Milestones** table (M2 — browse the full curriculum). Add a **Decisions log** line for any notable call (e.g. the frozen placeholder-`userState` shape).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)
- [ ] A user can browse Catalog -> Roadmap -> Week -> Concept with real seeded content (**M2 — browse the full curriculum**).
- [ ] Week shows thesis + concepts + the filtered problem list, with placeholder five-touch/solve state (no faked data).
- [ ] Concept shows reading (`body_md`) + when-to-use (`when_to_use_md`) + code template.
- [ ] `GET /paths/{slug}/weeks/{n}` returns a stable shape with a placeholder `userState` object (per-problem `touches[5]` + week rollup, `populated: false`).
- [ ] The Problem link routes to the S05 stub; both screens match the Week/Concept artboards.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
