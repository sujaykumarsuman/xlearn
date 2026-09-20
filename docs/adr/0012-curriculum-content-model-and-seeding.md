# ADR-0012 — Curriculum content model & seeding

- **Status:** Accepted
- **Date:** 2026-09-21
- **Deciders:** @sujaykumarsuman
- **Related:** [0003](0003-service-decomposition.md), [0005](0005-data-ownership-and-migrations.md), [0009](0009-deployment-and-gitops.md), [PRD Q2](../prd/xlearn-prd.md#10-open-questions)

## Context

S03 builds the `curriculum` service — the read-heavy content rail (paths, phases, weeks,
concepts, problems, sections). The domain model is fixed by
[data-model.md](../architecture/data-model.md#schema-curriculum) and the read API by
[api.md](../architecture/api.md#curriculum-read), but several concrete build questions were open:
where the content lives, how it is applied, exactly which columns the tables carry, and whether the
read endpoints are public. This ADR records those calls so later content sprints stay consistent.

## Decision

### Content lives as versioned seed files, embedded, applied idempotently on startup

- The DSA content is authored as JSON under **`curriculum/`** at the repo root (PRD Q2: seed files,
  not an in-app CMS): `curriculum/paths.json` (all path rows) + `curriculum/dsa/{phases,weeks,concepts,problems}.json`.
- A tiny **`package curriculum` (`curriculum/embed.go`)** `//go:embed`s them. It is data-only (no
  logic, no store dependency) and lives at the repo root because `go:embed` cannot reach a parent
  directory — mirroring how `embed.go` embeds `web/dist` for the gateway. Keeping the embed in its
  own package means the curriculum binary carries only the seed, never the SPA.
- A **seed loader** (`internal/curriculum.Seed`) parses the files (strict `DisallowUnknownFields`)
  and calls `store.SeedAll`, which upserts everything in **one transaction** with
  `ON CONFLICT ... DO UPDATE` on natural keys (path slug, `(path_slug,"order")`, `(path_slug,n)`,
  concept slug, problem id, `(problem_id,stage,"order")`). Re-running on every boot — and concurrent
  replicas — never duplicates rows (verified: identical counts across three boots + an integration
  test). Seeding runs after migrations; the service **refuses to serve** if either fails.
- Seeded coverage vs the 151-problem target is logged per path. S03 seeds the documented sample set
  (**14 of 151**); the seed expands in later sprints without a schema change.

### Table columns: the data-model set, plus a few content-support columns

Modeled exactly per data-model.md, with small, documented additions (its column lists are the *key*
columns, not exhaustive):

- `path`: adds `summary` (card blurb) and `sort_order` so **Catalog is fully API-driven** (which
  paths exist, their copy, and their order all come from the API, not hard-coded in the SPA).
- `phase`: adds `name` (display name, e.g. "Fundamentals") distinct from `theme` (the subtitle line).
- `"order"` is a reserved word and is always **quoted** in DDL/queries.
- Curriculum is **static seeded content**, so its tables carry **no `created_at`/`updated_at`** audit
  columns and **no outbox** (it emits/consumes nothing — [services.md](../architecture/services.md)).

### Read API is content-only this sprint; the gateway session-gates the proxy

- Endpoints return **content only** (no per-user state): `GET /paths`, `/paths/{slug}`,
  `/paths/{slug}/weeks/{n}`, `/problems/{id}` (sections keyed by `stage`), `/concepts/{slug}`. The
  five-touch/gating `agg` variants are S04/S05.
- The gateway **proxies** these under `/xlearn/api` (no aggregation yet). It **requires a valid
  session** (consistent with the rest of `/api`) but forwards **no user JWT** — curriculum has no
  user-scoped logic, so it stays ignorant of users and trusts the ClusterIP boundary.
- `problem_section` rows carry `stage`; the payload shape is already what S05 will filter to
  "only unlocked stages", so S05 only adds the filter.

### Roadmap difficulty mix is computed from seeded data

`GET /paths/{slug}` returns per-week easy/med/hard counts computed (via a `FILTER` aggregate) from
the **actually seeded** problems — honest real data rather than the artboard's decorative estimates.
Difficulty tokens: Easy = `--ds-ok` (green), Medium = `--ds-warn` (amber), Hard = `--ds-err` (red).
The overall ring and phase-progress meters render **0 / placeholder** until `practice` state exists.

## Consequences

- ✅ Content is versioned in-repo, reproducible, and safe to re-apply on every boot / rollout.
- ✅ Catalog + Roadmap are driven entirely by the API; adding a path or expanding the seed is a data
  change, no code change.
- ✅ The read shape is already S05-ready (stage-keyed sections), minimising later churn.
- ⚠️ The `curriculum/` root package is a slight repo-layout wrinkle (a data-only Go package beside the
  services) — the price of `go:embed`'s no-parent rule; documented here so it isn't mistaken for a
  service.
- ⚠️ Content browsing depends on identity being up (session validation). Acceptable: the app is
  unusable without login anyway, and identity is a core service.
- ⚠️ Two supporting columns (`path.summary`/`sort_order`, `phase.name`) extend the data-model's key
  set; recorded here to keep data-model.md the reference and this the rationale.

## Alternatives considered

| Option | Why not |
|--------|---------|
| **Seed as a big SQL migration** | Couples content edits to migrations; JSON seed files read as content, not schema, and re-apply idempotently on every boot. |
| **Embed seed under `internal/curriculum/`** | Would decouple cleanly, but the repo convention (PRD/AGENT) is versioned seed at `curriculum/`; honored via a root data package. |
| **Public (unauthenticated) content endpoints** | Simpler + decoupled from identity, but inconsistent with the app's post-login boundary; chose session-gated for a uniform security posture. |
| **Frontend hard-codes the coming-soon cards** | The artboard lists them, but hard-coding defeats "API-driven"; a `path` row + `summary` per stub keeps Catalog data-driven. |
| **No per-week difficulty counts (decorative only)** | Would miss demonstrating the difficulty tokens on real data; computed counts are honest and cheap. |
