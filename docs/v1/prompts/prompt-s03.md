# Prompt — Sprint 03 · Curriculum service + Catalog & Roadmap

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-03.md`](../sprints/sprint-03.md)   ·   **Milestone:** none (mid-phase)   ·   **Prereqs:** [S01](../sprints/sprint-01.md), [S02](../sprints/sprint-02.md)

## Read first
- [`../../../CLAUDE.md`](../../../CLAUDE.md) + [`../../../AGENT.md`](../../../AGENT.md) — repo conventions (theme, service boundaries, "ship at session end (AGENT.md land-and-sync)").
- [`../../adr/0005-data-ownership-and-migrations.md`](../../adr/0005-data-ownership-and-migrations.md) — schema-per-service, `goose` (embedded, startup + advisory lock), `sqlc`/`pgx`, per-schema role.
- [`../../adr/0009-deployment-and-gitops.md`](../../adr/0009-deployment-and-gitops.md) — GHCR image per service, Flux image-automation, SOPS secrets, ClusterIP-only for non-gateway services, pull-based (never `kubectl apply`).
- [`../../architecture/services.md`](../../architecture/services.md) — the `curriculum` responsibility + API + "emits/consumes nothing".
- [`../../architecture/data-model.md`](../../architecture/data-model.md) — the exact `curriculum` tables and columns.
- [`../../architecture/api.md`](../../architecture/api.md) — the external `/xlearn/api` surface and JSON/error envelope for the read endpoints.
- [`../../../design-system/README.md`](../../../design-system/README.md) — the **Sample data** section (what to seed) + difficulty tokens (green/amber/red); reuse `design-system/theme.css` verbatim.
- `design-system/screens/Catalog.dc.html` and `design-system/screens/Roadmap.dc.html` — layout/copy/interaction intent (design references only; not runnable, do not ship).
- Sibling infra: read `infra/apps/airlift.yaml` (HelmRelease template to copy), `infra/apps/image-automation.yaml`, `infra/infrastructure/database/README.md` (the DB 4-step pattern), and whatever `S02` added for `identity` (`infra/apps/xlearn-identity.yaml`, `infra/apps/secrets/*`) — mirror it for `curriculum`.

## Context
[S01](../sprints/sprint-01.md) put the app shell + all routes live at `/xlearn` (M0) and stood up the gateway BFF and `internal/platform/` helpers. [S02](../sprints/sprint-02.md) added `identity`, OAuth login, sessions, and the gateway-minted JWT (M1). Now build the **content rail**: a new read-heavy `curriculum` service, seeded from versioned files, plus the first two content screens. This sprint is read-only and has **no user state** yet (gating/progress arrive with `practice` in [S05](../sprints/sprint-05.md)).

## Do this (in order)
1. **curriculum service + schema** — Scaffold `cmd/curriculum/main.go` + `internal/curriculum/` (handler/service/store), reusing `internal/platform/` for config, `slog` JSON logging, the HTTP server, and the `pgx` pool. Create schema `curriculum` with `goose` SQL migrations under `internal/curriculum/store/migrations/`, `//go:embed`-ed and applied **on startup inside a Postgres advisory lock** (refuse to serve on failure). Model the seven tables exactly per `data-model.md`: `path`(slug PK, title, status `active`/`coming_soon`, problem_total, week_total), `phase`(path_slug, order, theme, week_from, week_to), `week`(path_slug, n, title, thesis, `unique(path_slug,n)`), `concept`(path_slug, slug, title, body_md, when_to_use_md, code_template), `week_concept`(week_id, concept_id), `problem`(natural id, path_slug, week_n, title, difficulty `easy`/`med`/`hard`, pattern, leetcode_url, neetcode_url, is_reinforcement), `problem_section`(problem_id, stage `attempt`/`hint`/`solution`, kind, order, body_md/code). Hand-write SQL, generate Go with `sqlc` (commit output; `sqlc diff` in CI). Set `search_path` to `curriculum`; no cross-schema access. Add `/healthz` + `/readyz`, `deploy/curriculum.Dockerfile` (multi-stage, non-root, read-only rootfs), and a CI path filter for `cmd/curriculum`/`internal/curriculum`. **No outbox** (curriculum emits nothing). In sibling `../infra` (mirror S02's identity wiring): add `infra/apps/secrets/pg-xlearn-curriculum.enc.yaml` (SOPS role password), the CNPG managed role `xlearn_curriculum` via the 4-step pattern + schema grant, the app-credentials secret, `infra/apps/xlearn-curriculum.yaml` (HelmRelease from `charts/project`, copied from `infra/apps/airlift.yaml`, `route.enabled: false`), and an `infra/apps/image-automation.yaml` entry for `ghcr.io/sujaykumarsuman/xlearn-curriculum` (semver `>=0.1.0`, setter `# {"$imagepolicy": "flux-system:xlearn-curriculum:tag"}`).
2. **DSA curriculum seed (versioned)** — Author the DSA curriculum as files under `curriculum/` (seed files, not a CMS). DSA `path`: slug `dsa`, title `DSA Interview Mastery`, status `active`, problem_total 151, week_total 16. 4 phases: Fundamentals (W1-3), Core Data Structures (W4-8), Advanced Patterns (W9-12), Interview Mastery (W13-16) with the Roadmap-artboard themes. All 16 weeks (title + thesis). Problems: seed **at minimum** the documented sample set — W1 Contains Duplicate / Valid Anagram / Two Sum / Group Anagrams / Top K Frequent / Subarray Sum Equals K; W2 3Sum / Longest Substring Without Repeating / Minimum Window Substring; signatures LRU Cache (W4), Largest Rectangle in Histogram (W3), Course Schedule (W8), Cheapest Flights Within K Stops (W9), Coin Change (W10) — each with difficulty (`easy`/`med`/`hard`), pattern, leetcode_url, neetcode_url, is_reinforcement (`#16 3Sum`, Medium/Two Pointers, is the hero). Add concept rows for the week patterns + `week_concept` links. Write a **seed loader** (startup, or a seed migration) that applies the files **idempotently** — `ON CONFLICT ... DO UPDATE` keyed on natural ids, so re-boot never duplicates. Log seeded count vs the 151 target. Also seed `coming_soon` path stubs (System Design, Go Concurrency, LLD/OOD, SQL & Data Modeling, Behavioral) so Catalog is API-driven.
3. **read API** — Expose these from curriculum and proxy them through the gateway under `/xlearn/api` (standard JSON success + `{ "error": { code, message } }` envelope): `GET /paths` (all paths + status), `GET /paths/{slug}` (phases, weeks, totals), `GET /paths/{slug}/weeks/{n}` (thesis + concepts + problem list, **content only** — the stateful `agg` is [S04](../sprints/sprint-04.md)), `GET /problems/{id}` (problem + `problem_section`s keyed by `stage`; return content sections now, keep the shape ready for S05's "only unlocked stages" filter), `GET /concepts/{slug}` (reading + code template). Read-only, no user state; the gateway just proxies (no aggregation this sprint).
4. **Catalog + Roadmap screens** — Port `Catalog.dc.html` -> `/xlearn` and `Roadmap.dc.html` -> `/xlearn/dsa` into React/TS under `web/`, reusing `theme.css` verbatim (no Tailwind). Catalog: active DSA card from `GET /paths` + coming-soon cards from the `coming_soon` paths. Roadmap: the 4-phase / 16-week rail + summary column from `GET /paths/dsa`; the overall ring and phase-progress meters render **0 / placeholder** until `practice` state exists. Difficulty tokens: Easy=`--ds-ok` green, Medium=`--ds-warn` amber, Hard=`--ds-err` red; mono font for numbers. Fetch `/xlearn/api` with `credentials: include`.

## Constraints
- Reuse `design-system/theme.css` **verbatim** (tokens + `ds-*`/`xl-*` classes); **no Tailwind**.
- **Schema-per-service:** `curriculum` touches only its own schema; `goose` embedded migrations applied on startup under an advisory lock; `sqlc` + `pgx`, no ORM; commit `sqlc` output (`sqlc diff` in CI).
- **Read-only, no user state** this sprint; do not reach into `practice`/`review` (they do not exist yet).
- **Seed idempotently** — `ON CONFLICT DO UPDATE` on natural keys; re-running on every boot must not duplicate.
- Curriculum **emits/consumes no events** — no outbox/inbox.
- Match infra conventions: copy `infra/apps/airlift.yaml` as the HelmRelease template; ClusterIP only (`route.enabled: false`); GHCR image + Flux image-automation entry; SOPS secrets under `infra/apps/secrets/`; pod hardening (non-root, read-only rootfs, dropped caps) from chart defaults.
- **Pull-based GitOps:** never `kubectl apply` by hand — all deploy config lands in `../infra` for Flux to reconcile.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables
- `cmd/curriculum/` + `internal/curriculum/` (handler/service/store, `goose` migrations, `sqlc` queries + generated Go), `/healthz` + `/readyz`.
- `deploy/curriculum.Dockerfile` + a CI path filter for the service.
- `curriculum/` versioned seed files (DSA path, 4 phases, 16 weeks, the sample-set problems + concepts, coming-soon stubs) + the idempotent seed loader.
- Read endpoints `GET /paths`, `/paths/{slug}`, `/paths/{slug}/weeks/{n}`, `/problems/{id}`, `/concepts/{slug}` served through the gateway.
- `web/` Catalog (`/xlearn`) + Roadmap (`/xlearn/dsa`) screens wired to the real API.
- Infra additions in `../infra`: `apps/xlearn-curriculum.yaml`, the `apps/image-automation.yaml` entry, and the `xlearn_curriculum` DB role/schema/SOPS secret (4-step pattern).

## Update status
- As each task lands, set its row in [`../sprints/sprint-03.md`](../sprints/sprint-03.md) to ✅ (🔄 while in progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row (S03). Add a **Decisions log** line for any notable call (e.g. what was seeded vs the 151 target).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)
- [ ] `curriculum` is deployed to prod (ClusterIP) via Flux; its schema is migrated + seeded **idempotently** on startup (re-boot does not duplicate rows).
- [ ] `GET /paths` and `GET /paths/dsa` return real seeded content through the gateway.
- [ ] Catalog renders the DSA path (active) + coming-soon others; Roadmap renders 4 phases / 16 weeks from the API (ring/percentages may be placeholder).
- [ ] Difficulty tokens are correct (green/amber/red); Catalog + Roadmap match the artboards.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
