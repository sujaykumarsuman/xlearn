# Sprint 12 — Hardening & 1.0

> **Milestone:** M7 — 1.0 tagged   ·   **Design phase:** Hardening
> **Prereqs:** [S09](sprint-09.md), [S11](sprint-11.md)   ·   **Unblocks:** —
> **Execute with:** [`../prompts/prompt-s12.md`](../prompts/prompt-s12.md) — one prompt, one session.

## Status

_Overall:_ ✅ Done — merged & deployed to prod; **`v1.0.0` tagged, M7 reached**

| # | Task | Status |
|---|------|--------|
| 1 | Performance pass (p95 screen API < 300ms) | ✅ |
| 2 | States + accessibility polish (all 12 screens) | ✅ |
| 3 | Core-loop e2e + OpenAPI (generated, CI drift check) | ✅ |
| 4 | Cut 1.0 (release-semver switch, tag v1.0.0) | ✅ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M7 milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

v1 is feature-complete (M6): the whole method loop, mock, analytics, and coach are wired. This sprint
does not add features — it **hardens** what exists (performance, error/empty/loading states,
accessibility), locks the contract with a **generated OpenAPI** spec and a **core-loop e2e** that runs
the real NATS event flow, then **cuts 1.0**: switch the deploy mode from build-semver auto-deploy to
release-semver tags ([ADR-0009](../../adr/0009-deployment-and-gitops.md)) and tag `v1.0.0`. Landing this
is **M7 — 1.0 tagged**.

## Scope

**In**
- Performance: DB indexes, N+1 removal, and safe caching of the gateway `agg` aggregations
  (`GET /dashboard`, `GET /paths/{slug}/weeks/{n}`) so p95 screen API < 300ms on the VPS
  ([PRD success metrics](../../prd/xlearn-prd.md#8-success-metrics)); verify the `assessment` projection
  read paths (`proj_coverage`/`proj_heatmap`/`proj_mastery`/`proj_outcome_mix`) stay index-served.
- Fit-on-one-node check: NATS JetStream + CNPG resource requests/limits sized for the single VPS node.
- Complete **error / empty / loading** states across **all 12 screens**; a full accessibility pass
  (focus order, contrast, keyboard operability, ARIA) against `design-system/theme.css`; responsive
  sanity below the 1440x1024 design width.
- **Core-loop e2e** (attempt -> solve -> schedule -> revise -> mistake -> mock -> progress) exercising
  the real NATS/JetStream + outbox path; **generated** `docs/architecture/openapi.yaml` from the gateway
  handlers, wired into CI as a **drift check**.
- **Cut 1.0**: flip infra image-automation `ImagePolicy` ranges to `>=1.0.0`, adopt tag-based deploy
  (`git tag v1.0.0`), introduce `/xlearn/api/v1` versioning, update `docs/git-strategy.md` status and
  `docs/v1/status.md` (M7).

**Out (later sprints)**
- v2 paths / features (multi-path expansion, sandbox code execution, extra LLM providers) — explicitly
  deferred; not part of the 1.0 hardening line.
- Metrics/tracing stack (Prometheus/OTel) — recorded as a known gap in
  [ADR-0009](../../adr/0009-deployment-and-gitops.md); a later ADR + infra addition, not this sprint.

## Tasks

### 1 · Performance pass

Hit the **p95 screen API < 300ms on the VPS** target ([PRD Ops metric](../../prd/xlearn-prd.md#8-success-metrics)).
Profile the hot screen endpoints in [`api.md`](../../architecture/api.md): the `agg` endpoints
(`GET /dashboard`, `GET /paths/{slug}/weeks/{n}`, `GET /progress`, `GET /problems/{id}`) and their
downstream service calls. Add missing DB indexes per owning schema (e.g. `practice.user_problem_state`
by `(account_id, week)`, `review.revision_item` by `(account_id, due_date)`, the `assessment.proj_*`
projection tables by their read keys) and remove N+1 query patterns in the `sqlc` query layer
([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)). Cache the gateway's `agg` fan-outs
(Dashboard, Week) where correctness allows — short TTL / per-account, invalidated on the relevant events
— since the gateway owns no schema and reads through ([`services.md`](../../architecture/services.md)).
Confirm the `assessment` **projection read paths** stay projection-served (no live cross-joins). Verify
NATS JetStream + CloudNativePG resource requests/limits fit the single node
([ADR-0009](../../adr/0009-deployment-and-gitops.md)).

### 2 · States + accessibility polish

Complete the **error / empty / loading** states for every one of the 12 screens (Catalog, Roadmap,
Dashboard ★, Week, Concept, Problem ★, Revision, Mistakes, Mock, Progress, Settings, Auth) so no screen
shows a raw spinner-forever or an unstyled error. Match the artboards in `design-system/screens/*.dc.html`
for copy and layout intent (they are reference only, not runnable). Run an accessibility pass against
`design-system/theme.css` ([ADR-0008](../../adr/0008-frontend-stack.md)): visible focus rings and
logical focus order, keyboard operability of the Problem timer HUD / reveal gates and the Mock phase
rail, ARIA roles/labels on the coach panel and dialogs, and contrast — including the difficulty tokens
(Easy `--ds-ok` / Medium `--ds-warn` / Hard `--ds-err`). Check responsive sanity below the desktop-first
1440x1024 design width. Reuse `theme.css` tokens/components verbatim; no ad-hoc styling.

### 3 · Core-loop e2e + OpenAPI

Add an **end-to-end test** for the core loop that follows [`events.md`](../../architecture/events.md)
Flows 1-4: `POST /problems/{id}/outcome {clean}` -> `practice.problem_solved` -> `review` schedules
Day 1/3/7/21/45 -> `POST /revision/{id}/score` auto-scores (advance or reset) -> a below-clean path
opens a `mistake_entry` -> a `POST /mocks` + `POST /mocks/{id}/score` completes -> `assessment`
projections update Progress. It must exercise the **real NATS/JetStream + transactional outbox** path
with idempotent consumers ([ADR-0004](../../adr/0004-inter-service-comms-and-events.md)) — do **not**
mock the broker away. Generate `docs/architecture/openapi.yaml` (OpenAPI 3.1) from the gateway handlers
(the human contract table in [`api.md`](../../architecture/api.md) is the reference until then), and
wire a **drift check** into `ci.yml` that fails if the committed spec is stale versus the handlers.

### 4 · Cut 1.0

Perform the deliberate **ADR-0009 deploy-mode switch** ([ADR-0009](../../adr/0009-deployment-and-gitops.md),
[`git-strategy.md`](../../git-strategy.md)). In the sibling infra repo, change the image-automation
`ImagePolicy` ranges for every `xlearn-<svc>` from `>=0.1.0` to `>=1.0.0` in `infra/apps/image-automation.yaml`
and adopt tag-based deploy (image tag `vX.Y.Z` -> `X.Y.Z`), so a `git tag v1.0.0` is what ships prod.
Introduce **`/xlearn/api/v1`** versioning at this milestone (per the versioning note in
[`api.md`](../../architecture/api.md)). Update the `docs/git-strategy.md` versioning status and
`docs/v1/status.md` to record M7 (deploy mode -> release-semver, 1.0 tagged). Coordinate the infra range
change with the repo tag so auto-deploy does not stall on the transition.

## Acceptance criteria

- [ ] p95 screen API < 300ms on the VPS; no obvious N+1 on the hot screens (Dashboard, Week, Problem, Progress).
- [ ] All 12 screens have complete error / empty / loading states and pass the accessibility pass.
- [ ] Core-loop e2e is green in CI and runs the real NATS event flow; `docs/architecture/openapi.yaml`
      is generated and drift-checked in CI.
- [ ] **M7 gate:** 1.0 tagged (`v1.0.0`); infra image-automation switched to `>=1.0.0`; `/xlearn/api/v1`
      live; docs updated (`git-strategy.md` + `status.md`, M7).

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- The build-semver -> release-tag switch must be **coordinated in infra** (`ImagePolicy` ranges to
  `>=1.0.0`) at the same time as the `v1.0.0` tag, or auto-deploy stalls / re-deploys the wrong image.
- Do **not** regress the security posture during perf work: keep BYO-key handling encrypted-at-rest
  ([ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md)) and the gateway-minted RS256 JWT / JWKS
  path intact ([ADR-0006](../../adr/0006-authn-authz.md)); caching must stay per-account and never leak
  across users.
- The e2e must exercise the **real** event flow (NATS JetStream + outbox), not a mocked broker, or it
  will pass while the async path is broken.
- Caching the `agg` aggregations can serve stale "due today" / streak data — scope TTL and invalidation
  to the events that change them, and never cache below-clean/mistake state incorrectly.
