# Sprint 03 — Curriculum service + Catalog & Roadmap

> **Milestone:** none (mid-phase)   ·   **Design phase:** D2 Content
> **Prereqs:** [S01](sprint-01.md), [S02](sprint-02.md)   ·   **Unblocks:** [S04](sprint-04.md), [S05](sprint-05.md)
> **Execute with:** [`../prompts/prompt-s03.md`](../prompts/prompt-s03.md) — one prompt, one session.

## Status

_Overall:_ 🔄 Code complete + verified locally (real Postgres, idempotent seed, adversarial review clean); **deploy to prod pending commit/merge**

| # | Task | Status |
|---|------|--------|
| 1 | curriculum service + schema | ✅ |
| 2 | DSA curriculum seed (versioned) | ✅ |
| 3 | read API | ✅ |
| 4 | Catalog + Roadmap screens | ✅ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row; this
> sprint carries no milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up the `curriculum` service — the read-heavy content rail — and make the DSA path browsable
end to end. This sprint delivers the seeded content model (paths, phases, weeks, concepts, problems,
sections), the read API behind the gateway, and the first two content screens (Catalog, Roadmap)
wired to real data. It is the foundation the guided flow builds on: [S04](sprint-04.md) adds the
Week/Concept detail views with user state, and [S05](sprint-05.md) layers per-user gating and the
Problem engine on top of this content.

## Scope

**In**
- New `curriculum` service (`cmd/curriculum` + `internal/curriculum`), schema `curriculum` via
  `goose` (embedded, startup + advisory lock) and `sqlc`/`pgx` ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).
- Tables `path`, `phase`, `week`, `concept`, `week_concept`, `problem`, `problem_section`
  (columns per [data-model.md](../../architecture/data-model.md#schema-curriculum)).
- Per-service DB role `xlearn_curriculum` (infra 4-step) + SOPS secret; `apps/xlearn-curriculum.yaml`
  HelmRelease (ClusterIP, `route.enabled: false`) + an image-automation entry; `/healthz` + `/readyz`.
- Versioned DSA seed under `curriculum/` (files, not an in-app CMS) applied idempotently on startup.
- Read endpoints (via the gateway BFF): `GET /paths`, `GET /paths/{slug}`,
  `GET /paths/{slug}/weeks/{n}`, `GET /problems/{id}`, `GET /concepts/{slug}`.
- Catalog (`/xlearn`) and Roadmap (`/xlearn/dsa`) screens ported to React/TS against the real API.

**Out (later sprints)**
- Week + Concept **detail** screens and the week aggregation **with the user's five-touch state** —
  [S04](sprint-04.md) (the `agg` version of `/paths/{slug}/weeks/{n}`).
- Per-user gating/progress (locked stages, unlocked sections, solved counts, the Roadmap ring's real
  percentages) — `practice` in [S05](sprint-05.md).
- Any event emission/consumption — `curriculum` emits and consumes **nothing**
  ([services.md](../../architecture/services.md)), so no outbox/inbox here.

## Tasks

### 1 · curriculum service + schema

Scaffold `cmd/curriculum/main.go` and `internal/curriculum/` (handler / service / store layers, reusing
the `internal/platform/` helpers from [S01](sprint-01.md) for config, `slog` JSON logging, HTTP server,
and the `pgx` pool). Create schema `curriculum` with `goose` SQL migrations under
`internal/curriculum/store/migrations/`, **embedded** with `//go:embed` and applied on **startup inside
a Postgres advisory lock** — the service refuses to serve if migration fails
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). Model the seven tables exactly as
[data-model.md](../../architecture/data-model.md#schema-curriculum) specifies: `path` (slug PK, title,
status `active`/`coming_soon`, `problem_total`, `week_total`), `phase` (path_slug, order, theme,
week_from, week_to), `week` (path_slug, n, title, thesis, `unique(path_slug,n)`), `concept`
(path_slug, slug, title, body_md, when_to_use_md, code_template), `week_concept` (week_id, concept_id
M:N), `problem` (natural id, path_slug, week_n, title, `difficulty` in `easy`/`med`/`hard`, pattern,
leetcode_url, neetcode_url, is_reinforcement), and `problem_section` (problem_id, `stage` in
`attempt`/`hint`/`solution`, kind, order, body_md/code). Write hand-authored SQL queries and generate
type-safe Go with `sqlc` (commit the output; CI runs `sqlc diff`). The role sets `search_path` to its
own schema; **no** cross-schema access. Add `/healthz` + `/readyz`, a `deploy/curriculum.Dockerfile`
(multi-stage, non-root, read-only rootfs), and a CI path filter for `cmd/curriculum`/`internal/curriculum`.
Infra (sibling `../infra`, mirroring what [S02](sprint-02.md) added for identity): a role-password SOPS
secret `infra/apps/secrets/pg-xlearn-curriculum.enc.yaml`, the CNPG managed role `xlearn_curriculum`
(4-step pattern) with a schema grant, an app-credentials secret, `infra/apps/xlearn-curriculum.yaml`
(HelmRelease rendering `charts/project`, copied from `infra/apps/airlift.yaml`, `route.enabled: false`),
and an `infra/apps/image-automation.yaml` ImageRepository/ImagePolicy for
`ghcr.io/sujaykumarsuman/xlearn-curriculum` (semver `>=0.1.0`, setter
`# {"$imagepolicy": "flux-system:xlearn-curriculum:tag"}`). Curriculum emits no events, so **no outbox**.

### 2 · DSA curriculum seed (versioned)

Author the DSA curriculum as versioned files under `curriculum/` (the PRD Q2 decision: seed files, not
an in-app CMS). Model the DSA `path` (slug `dsa`, title `DSA Interview Mastery`, status `active`,
`problem_total` 151, `week_total` 16), its 4 phases (Fundamentals W1-3, Core Data Structures W4-8,
Advanced Patterns W9-12, Interview Mastery W13-16 — themes from the Roadmap artboard), all 16 weeks
(title + thesis), plus problem metadata. Seed **at minimum** the documented sample set from
[design-system/README.md (Sample data)](../../../design-system/README.md#sample-data-used-in-the-prototype):
W1 Contains Duplicate / Valid Anagram / Two Sum / Group Anagrams / Top K Frequent / Subarray Sum Equals
K; W2 3Sum / Longest Substring Without Repeating / Minimum Window Substring; and the signature problems
LRU Cache (W4), Largest Rectangle in Histogram (W3), Course Schedule (W8), Cheapest Flights Within K
Stops (W9), Coin Change (W10). Each problem carries `difficulty` (`easy`/`med`/`hard`), `pattern`,
`leetcode_url`, `neetcode_url`, and `is_reinforcement`; `#16 3Sum` (Medium, Two Pointers) is the hero.
Author concept rows for the week patterns and link them via `week_concept`. A **seed loader** (invoked
at startup, or a dedicated seed migration) applies the files **idempotently** — upsert by natural key
(`ON CONFLICT ... DO UPDATE`) so re-running on every boot never duplicates. Log seeded vs the 151 target.
Also seed the `coming_soon` path stubs (System Design, Go Concurrency, LLD/OOD, SQL & Data Modeling,
Behavioral) so Catalog is fully API-driven rather than hard-coding the artboard's cards.

### 3 · read API

Expose the read-only content endpoints from the curriculum service and proxy them through the gateway
under `/xlearn/api` ([api.md](../../architecture/api.md#curriculum-read)): `GET /paths` (Catalog: all
paths + status), `GET /paths/{slug}` (Roadmap: phases, weeks, totals), `GET /paths/{slug}/weeks/{n}`
(week thesis + concepts + problem list — **content only**; the `agg` version that folds in the user's
five-touch state is [S04](sprint-04.md), since `practice`/`review` do not exist yet), `GET /problems/{id}`
(problem + its `problem_section`s keyed by `stage` — return content sections for now; true "only unlocked
stages" gating lands with `practice` in [S05](sprint-05.md), so keep the response shape ready with the
`stage` field), and `GET /concepts/{slug}` (concept reading + code template; the Concept screen itself is
[S04](sprint-04.md)). All read-only, no user state; use the standard JSON success + `{ "error": { code,
message } }` envelope from [api.md](../../architecture/api.md#conventions). The gateway simply proxies
(no aggregation this sprint).

### 4 · Catalog + Roadmap screens

Port `design-system/screens/Catalog.dc.html` and `Roadmap.dc.html` into real React/TS components under
`web/`, reusing `design-system/theme.css` **verbatim** (tokens + `ds-*`/`xl-*` classes; no Tailwind).
**Catalog** (`/xlearn`): render the active DSA card from `GET /paths` (the `active` entry) with its
totals, and the coming-soon cards from the `coming_soon` paths. **Roadmap** (`/xlearn/dsa`): render the
vertical phase/week rail (4 phases, 16 week cards with difficulty mix) and the summary column from
`GET /paths/dsa`; the overall ring and phase-progress meters show **0 / placeholder** until `practice`
state exists ([S05](sprint-05.md)). Apply the difficulty tokens correctly: Easy = `--ds-ok` (green),
Medium = `--ds-warn` (amber), Hard = `--ds-err` (red); use the mono font for numbers/metrics. Fetch from
`/xlearn/api` with `credentials: include`.

## Acceptance criteria
- [~] `curriculum` deployed to prod (ClusterIP) via Flux; its schema is migrated + seeded
      **idempotently** on startup (re-boot does not duplicate rows). — _migrate + idempotent seed
      **verified locally** (identical row counts across 3 boots + integration test + the shipping
      Docker image); infra written; **prod deploy pending commit/merge**._
- [x] `GET /paths` and `GET /paths/dsa` return real seeded content through the gateway. — _verified
      locally through the gateway BFF (session-gated proxy)._
- [x] Catalog renders the DSA path (active) + coming-soon others; Roadmap renders 4 phases / 16 weeks
      from the API (ring/percentages may be placeholder). — _verified live in the browser._
- [x] Difficulty tokens are correct (green/amber/red); Catalog + Roadmap match the artboards.

## Definition of Done
CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs
- **Seed idempotency:** the loader runs on every startup, so it must **upsert** (`ON CONFLICT DO
  UPDATE`) keyed on natural ids (path slug, `(path_slug,n)`, problem id, concept slug), never blind
  `INSERT` — otherwise multi-replica or restart re-seeds duplicate rows.
- **Stage scoping is content-only here:** `problem_section.stage` shapes the payload, but true gating
  (locked sections not delivered) belongs to `practice` in [S05](sprint-05.md). Keep the
  `GET /problems/{id}` response shape ready (sections carry `stage`) so S05 only has to filter.
- **151 is the target, not a blocker:** it is fine to seed the documented sample set now and expand
  the seed later; record what was seeded vs the 151 target (seed log + a Decisions-log line).
- **No cross-schema reads:** `/paths/{slug}/weeks/{n}` must return content only this sprint; do not
  reach for `practice`/`review` (they do not exist yet). The stateful `agg` is [S04](sprint-04.md).
