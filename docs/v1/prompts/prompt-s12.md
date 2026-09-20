# Prompt — Sprint 12 · Hardening & 1.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-12.md`](../sprints/sprint-12.md)   ·   **Milestone:** M7 — 1.0 tagged   ·   **Prereqs:** [S09](../sprints/sprint-09.md), [S11](../sprints/sprint-11.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions; do not commit/push unless asked; keep the dark theme.
- [`../../adr/0009-deployment-and-gitops.md`](../../adr/0009-deployment-and-gitops.md) — the 1.0 image-policy switch (`>=0.1.0` -> `>=1.0.0`), infra file map, single-node constraints.
- [`../../git-strategy.md`](../../git-strategy.md) — release train; build-semver -> release-tag mode; what "cut 1.0" means.
- [`../../architecture/api.md`](../../architecture/api.md) — the gateway surface, the `agg` endpoints to profile, the OpenAPI note, and `/xlearn/api/v1` versioning.
- [`../../architecture/events.md`](../../architecture/events.md) — the core-loop flows (1-4) the e2e must exercise over real NATS.
- [`../../architecture/services.md`](../../architecture/services.md) — which service/schema backs each screen; projection read paths in `assessment`.
- [`../../adr/0006-authn-authz.md`](../../adr/0006-authn-authz.md) + [`../../adr/0007-ai-coach-byo-key-and-secrets.md`](../../adr/0007-ai-coach-byo-key-and-secrets.md) — the security posture not to regress.
- [`../../prd/xlearn-prd.md`](../../prd/xlearn-prd.md) — success metrics (p95 < 300ms; method-adherence), the 12 screens/routes.
- `design-system/screens/*.dc.html` — polish parity (states/copy/layout intent; reference only, not runnable) + [`../../../design-system/README.md`](../../../design-system/README.md) for tokens.
- Sibling `../infra` repo: `infra/apps/image-automation.yaml` (ImagePolicy ranges), `infra/apps/airlift.yaml` (release-tag template) — inspect before touching deploy config; never `kubectl apply`.

## Context

v1 is feature-complete (M6): S05-S11 delivered the guided Problem flow, five-touch review + mistakes,
mock + progress projections (S09), and the BYO-key coach panel across all screens (S11). Nothing new is
added here. This sprint hardens performance, error/empty/loading states, and accessibility; locks the
API contract with a generated OpenAPI spec + a real-event core-loop e2e; then cuts **1.0** by switching
the deploy mode to release-semver tags and tagging `v1.0.0` (M7).

## Do this (in order)

1. **Performance pass** — Profile the hot screen endpoints in `api.md`: the `agg` fan-outs
   `GET /dashboard`, `GET /paths/{slug}/weeks/{n}`, `GET /progress`, `GET /problems/{id}`. Add missing DB
   indexes in the owning schemas via new `goose` migrations (e.g. `practice.user_problem_state(account_id, week)`,
   `review.revision_item(account_id, due_date)`, the `assessment.proj_*` tables by their read keys) and
   remove N+1s in the `sqlc` query layer. Add safe **per-account, event-invalidated** caching of the
   gateway Dashboard/Week aggregations (the gateway owns no schema, so cache in the BFF). Confirm the
   `assessment` projections stay projection-served (no live cross-joins). Size NATS JetStream + CNPG
   resource requests/limits for the single node. Target: **p95 screen API < 300ms on the VPS**.
2. **States + accessibility polish** — Complete **error / empty / loading** states for all 12 screens
   (Catalog, Roadmap, Dashboard, Week, Concept, Problem, Revision, Mistakes, Mock, Progress, Settings,
   Auth), matching the `*.dc.html` artboards for copy/layout. Run an a11y pass against `theme.css`:
   focus order + visible focus, keyboard operability (Problem reveal gates / timer HUD, Mock phase rail),
   ARIA on the coach panel + dialogs, and contrast (including difficulty tokens Easy `--ds-ok` / Medium
   `--ds-warn` / Hard `--ds-err`). Check responsive sanity below 1440px. Reuse `theme.css` verbatim.
3. **Core-loop e2e + OpenAPI** — Write an e2e test that runs `events.md` Flows 1-4 over the **real**
   NATS JetStream + transactional outbox (idempotent consumers; do not mock the broker):
   attempt -> `problem_solved` -> `review` schedules Day 1/3/7/21/45 -> `POST /revision/{id}/score`
   auto-scores (advance or reset) -> a below-clean path opens a `mistake_entry` ->
   `POST /mocks` + `/score` -> `assessment` projections update Progress. Generate
   `docs/architecture/openapi.yaml` (OpenAPI 3.1) from the gateway handlers, and add a **drift check** to
   `ci.yml` that fails when the committed spec is stale versus the handlers.
4. **Cut 1.0** — In `../infra`: switch every `xlearn-<svc>` `ImagePolicy` range in
   `infra/apps/image-automation.yaml` from `>=0.1.0` to `>=1.0.0` and adopt tag-based deploy
   (`vX.Y.Z` -> `X.Y.Z`, mirroring `infra/apps/airlift.yaml`). Introduce **`/xlearn/api/v1`** versioning
   on the gateway. Update `docs/git-strategy.md` (versioning status) and `docs/v1/status.md` (M7: deploy
   mode -> release-semver, 1.0 tagged). Coordinate the infra range change with the `git tag v1.0.0` so
   auto-deploy does not stall. (Tag/commit/push only when the user asks.)

## Constraints

- Reuse `design-system/theme.css` tokens/components **verbatim** (no Tailwind, no ad-hoc CSS); keep the dark theme.
- Match infra conventions exactly: copy `infra/apps/airlift.yaml`/`landscape.yaml` as the HelmRelease template; GHCR `ghcr.io/sujaykumarsuman/xlearn-<svc>`; **pull-based GitOps — never `kubectl apply` by hand**.
- Keep the data model intact: schema-per-service in `xlearndb`, `goose` migrations (embedded, startup + advisory lock), `sqlc`/pgx; new indexes ship as migrations.
- Async stays idempotent: transactional outbox + dedupe-on-`event_id` consumers; the e2e must use the real NATS path.
- Do not regress security: BYO-key encrypted at rest; gateway-minted RS256 JWT / JWKS intact; caches are per-account and never cross-user.
- Inherit chart pod hardening (non-root, read-only rootfs, dropped caps).
- **Do not commit or push unless asked** — including the `v1.0.0` tag and any infra change.

## Deliverables

- Index-adding `goose` migrations + N+1 fixes in the hot `sqlc` query paths; per-account, event-invalidated caching of the gateway Dashboard/Week aggregations; verified NATS/CNPG resource limits.
- Complete error/empty/loading states across all 12 screens + accessibility fixes, all on `theme.css`.
- A core-loop e2e (real NATS/JetStream + outbox) green in CI; generated `docs/architecture/openapi.yaml` + a CI drift check in `ci.yml`.
- `/xlearn/api/v1` versioning live; infra `ImagePolicy` ranges at `>=1.0.0` (release-tag deploy); `v1.0.0` tag; updated `docs/git-strategy.md` + `docs/v1/status.md` (M7).

## Update status

- As each task lands, set its row in [`../sprints/sprint-12.md`](../sprints/sprint-12.md) to ✅ (🔄 while in progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the **Milestones** table (M7). Add a **Decisions log** line for the deploy-mode switch to release-semver.
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] p95 screen API < 300ms on the VPS; no obvious N+1 on the hot screens (Dashboard, Week, Problem, Progress).
- [ ] All 12 screens have complete error / empty / loading states and pass the a11y pass.
- [ ] Core-loop e2e is green in CI and runs the real NATS event flow; `docs/architecture/openapi.yaml` is generated and drift-checked in CI.
- [ ] **M7 gate:** 1.0 tagged (`v1.0.0`); infra image-automation switched to `>=1.0.0`; `/xlearn/api/v1` live; docs updated (`git-strategy.md` + `status.md`, M7).
- Do not commit or push unless asked.
