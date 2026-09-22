# F003 — Rename the DSA path

**Status:** ✅ done (local review) · **Opened:** 2026-09-22

## Feedback

Rename **"DSA Interview Mastery"** → simply **"Data Structures & Algorithms"**.

## Decision

- The path **title** becomes "Data Structures & Algorithms". The short code / eyebrow stays
  **"DSA"** (slug unchanged: `dsa`).
- Source of truth is the curriculum seed (`curriculum/paths.json`); the web hard-coded
  fallbacks and fixtures are updated to match so tests and the offline-fallback render agree.

## Scope / changes

- `curriculum/paths.json` — `title`.
- `internal/curriculum/handlers_test.go` — fixture title.
- web fallbacks/fixtures: `screens/Auth.tsx`, `screens/Roadmap.tsx`, `screens/Dashboard.tsx`,
  and the `*.test.tsx` fixtures (`Catalog`, `Roadmap`, `Week`, `Auth`).

## Changes done

- `curriculum/paths.json` — `title` → "Data Structures & Algorithms" (seed source of truth).
- `internal/curriculum/handlers_test.go` — fixture title.
- web: `screens/Auth.tsx`, `screens/Roadmap.tsx`, `screens/Dashboard.tsx` fallbacks +
  `Catalog/Roadmap/Week/Auth` test fixtures.

**Verified live** (docker-compose): `GET /paths` returns the new title; it renders on the Catalog
hero, the Roadmap header, the Dashboard eyebrow, and the onboarding path card.

## Notes

- Slug `dsa` and all routes are unchanged — pure display rename. The short code stays **DSA**
  (the top-bar selector badge + the onboarding icon).
