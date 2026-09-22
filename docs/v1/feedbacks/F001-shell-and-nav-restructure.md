# F001 — Shell & navigation restructure

**Status:** ✅ done (local review) · **Opened:** 2026-09-22

## Feedback

- Remove the (disabled ⌘K) search; move the **curriculum selector** into that top-bar spot.
- The home page `projects.sujaykumar.dev/xlearn` should **only list paths to select** and
  need **no left nav** — the left nav belongs **inside a curriculum**, because each
  curriculum may expose different options.
- **Settings** and **Progress** are already in the avatar menu — drop them from the left nav.

Rationale (owner): a System Design path won't follow the same revision pattern as DSA, though
some items stay common (Today, Roadmap, Progress, Mock interview). The solving ground / judge
may differ too. Keep the design **expandable** for that; don't build the per-curriculum
variants yet.

## Decision

- Split the app frame into two shells that share the top bar + coach:
  - **Plain shell** (no sidebar): `/` (Catalog) and `/settings` — account/hub level.
  - **Curriculum shell** (with sidebar): everything under `/dsa/*`.
- The left nav becomes **curriculum-scoped** via `navForPath(slug)` (a clear extension point;
  DSA is the only populated path today). Shared-across-curricula items (Today, Roadmap, Mock)
  live there; the per-curriculum "Practice loop" section can vary later.
- The curriculum selector (`xl-pathsw`) moves from the sidebar into the top bar: **hidden on
  the home/Catalog page**, shown inside a curriculum (switch path / browse all paths).
- Nav no longer lists **Settings** or **Progress** (both remain reachable from the avatar menu;
  Progress stays routed at `/dsa/progress`).

## Scope / changes

- `web/src/router.tsx` — two layout routes (plain vs curriculum).
- `web/src/components/AppShell.tsx` → shared frame + `CurriculumShell` / `PlainShell`.
- `web/src/components/Sidebar.tsx` — drop the in-sidebar path switcher; render `navForPath`.
- `web/src/components/Topbar.tsx` — remove search; mount the path switcher.
- `web/src/components/PathSwitcher.tsx` (new) — the top-bar curriculum selector.
- `web/src/nav.ts` — `navForPath(slug)`; drop Settings/Progress.

## Changes done

- `web/src/components/AppShell.tsx` — replaced the single shell with `PlainShell`
  (no sidebar) + `CurriculumShell` (sidebar), sharing the top bar + coach.
- `web/src/router.tsx` — two layout routes under the auth gate: PlainShell wraps
  `/` (Catalog) + `/settings` + the `*` 404; CurriculumShell wraps `/dsa/*`.
- `web/src/components/RequireAuth.tsx` — the gate now renders `<Outlet/>` (not the shell).
- `web/src/components/PathSwitcher.tsx` (new) + `Topbar.tsx` — search removed; the
  selector sits in the top bar inside a curriculum, the brand shows on hub pages.
- `web/src/components/Sidebar.tsx` — path switcher removed; renders `navForPath(slug)`.
- `web/src/nav.ts` — `NAV` → `navForPath(slug)`; Settings + Progress dropped.
- `web/src/styles/app.css` — top-bar switcher/brand/spacer rules (theme.css untouched).
- Tests: `AppShell.test.tsx`, `nav.test.ts`, `Auth.test.tsx` updated to the split shells.

**Verified live** (docker-compose): home has no sidebar + no selector; inside `/dsa/*`
the sidebar returns, the selector opens in the top bar, and Settings/Progress are only in
the avatar menu.

## Notes

- Coach stays mounted on every screen (incl. home), per the design.
