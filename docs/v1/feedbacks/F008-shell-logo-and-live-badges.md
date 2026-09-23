# F008 — Shell polish: brand routes home + live Practice-loop badges

## Feedback

Two small shell issues raised while reviewing F007:

- The **xLearn logo** in the sidebar wasn't clickable — it should route to `/xlearn` (the Catalog
  home) from any page, or to the login screen when signed out.
- The **Practice-loop badge counts** in the sidebar (Revision · Mistakes) looked hard-coded — the
  sidebar said "4" while the Revision page said "1 due today." Explain the logic, or make it real.

## Decision

- **Brand → home link.** The plain-shell top bar already linked its brand to `/` (`BrandInline`); the
  gap was the **curriculum sidebar** brand, a bare `<div>`. It's now a `<Link to="/">`, so it lands on
  the Catalog home when authenticated and — via the existing `RequireAuth` gate — redirects to `/auth`
  when not. One anchor-reset rule in `app.css` (`theme.css` stays byte-identical to the design source).
- **Live badge counts.** They *were* hard-coded scaffold (`nav.ts`: `Revision "4"`, `Mistakes "3"` —
  "static decoration until the practice/review services back them"). The real counts already exist, so
  the sidebar now reads them:
  - **Revision** = `GET /revision/due → dueCount` (reviews due today, `useDueRevision`)
  - **Mistakes** = `GET /mistakes → openCount` (open journal entries, `useMistakes`)

  Each badge is **hidden until the count loads and only shows when > 0** (a "0" chip is noise), capped
  at "99+". Both are cached react-query reads shared with the Revision / Mistakes screens, so the
  sidebar adds no fetch once those screens have loaded. Revision keeps its teal "due" styling.

## Scope / changes

- **web**: `nav.ts` (`navForPath(slug, counts)` + `NavCounts`; badges derived from live counts, hidden
  at 0); `Sidebar.tsx` (brand `<Link>`; `useDueRevision` + `useMistakes` feed the counts);
  `app.css` (`a.xl-brand` anchor reset + hover). Tests: `AppShell.test.tsx` gains a live-badge-count
  case and a brand-routes-home case.

## Status

🔄 **In progress** — code-complete and green (web typecheck/lint + tests; no backend change). In local
review alongside [F007](F007-email-password-auth.md); ships in the same release (`v1.3.0`).

## Notes

- No backend or schema change — this is purely wiring the sidebar to endpoints that already exist.
- The badges reflect the enrolled path's counts (DSA only today); multi-path fan-out is future work.
