# xLearn v1 — status

Cross-sprint living tracker. Updated per the
[status protocol](sprints/README.md#status-protocol-way-of-working) whenever a task or sprint changes
state (it's baked into every `prompt-sNN.md`). Per-task detail lives in each
[`sprints/sprint-NN.md`](sprints/); newest decisions at the top of the log.

- **Build line:** v1 · **Phase:** S02 identity/auth **code + infra complete & verified locally**; prod deploy (M1) pending merge + OAuth app registration
- **Deploy mode:** build-semver auto-deploy from `main` ([ADR-0009](../adr/0009-deployment-and-gitops.md)) — proven end-to-end (`xlearn-gateway:0.1.3`)
- **Last updated:** 2026-09-20

## Snapshot

| Area | State |
|------|-------|
| PRD | ✅ [drafted](../prd/xlearn-prd.md) |
| ADRs 0001-0011 | ✅ accepted |
| Architecture (overview/services/data/api/events/diagrams) | ✅ drafted |
| Git strategy | ✅ drafted |
| v1 build plan | ✅ done |
| Sprint plans + prompts (S01-S12) | ✅ all scaffolded |
| Application code | 🔄 S01 gateway + web shell **live**; S02 identity service + auth (OAuth, sessions, JWT/JWKS, outbox, Auth screen) **built & verified locally**, not yet deployed |
| `infra` xlearn wiring | 🔄 gateway **live**; S02 adds `xlearn-identity` HelmRelease + gateway JWT env, `xlearndb` 4-step DB wiring (managed roles + `Database` CR), SOPS `xlearn-db`/`xlearn-oauth`/`xlearn-jwt`, identity image-automation — **staged, not merged** |

## Sprint board

| Sprint | Focus | State |
|--------|-------|-------|
| [S01](sprints/sprint-01.md) | Bootstrap + app shell (→ M0 live at `/xlearn`) | ✅ Done — live at `/xlearn` |
| [S02](sprints/sprint-02.md) | identity / auth (→ M1) | 🔄 Code + infra complete, verified locally; deploy pending merge + OAuth apps |
| [S03](sprints/sprint-03.md) | curriculum + Catalog/Roadmap | ⬜ Planned |
| [S04](sprints/sprint-04.md) | Week + Concept (→ M2) | ⬜ Planned |
| [S05](sprints/sprint-05.md) | practice / Problem ★ (→ M3) | ⬜ Planned |
| [S06](sprints/sprint-06.md) | review scheduler / Revision | ⬜ Planned |
| [S07](sprints/sprint-07.md) | mistakes + notifications (→ M4) | ⬜ Planned |
| [S08](sprints/sprint-08.md) | assessment / Mock | ⬜ Planned |
| [S09](sprints/sprint-09.md) | progress + Dashboard ★ (→ M5) | ⬜ Planned |
| [S10](sprints/sprint-10.md) | Settings | ⬜ Planned |
| [S11](sprints/sprint-11.md) | coach (→ M6) | ⬜ Planned |
| [S12](sprints/sprint-12.md) | hardening + 1.0 (→ M7) | ⬜ Planned |

Legend: ✅ done · 🔄 in progress · ⬜ planned/not started · ⛔ blocked. Update per the
[status protocol](sprints/README.md#status-protocol-way-of-working).

## Milestones

| ID | Target | State |
|----|--------|-------|
| M0 live at `/xlearn` | end S01 | ✅ **live** at `projects.sujaykumar.dev/xlearn` (`xlearn-gateway:0.1.3`, Flux-deployed 2026-09-20) |
| M1 login | end S02 | 🔄 login works end-to-end **locally** (real Postgres + browser); prod deploy pending merge + GitHub/Google OAuth app registration |
| M2 browse curriculum | end S04 | ⬜ |
| M3 guided problem | end S05 | ⬜ |
| M4 repetition+mistakes | end S07 | ⬜ |
| M5 mock+analytics | end S09 | ⬜ |
| M6 feature-complete | end S11 | ⬜ |
| M7 1.0 | S12 | ⬜ |

## Decisions log

Notable calls not (yet) worth a full ADR, newest first. Promote to an ADR if they harden.

| Date | Decision | Notes |
|------|----------|-------|
| 2026-09-20 | **Ship GitHub-only OAuth for the initial S02 cut; Google deferred** until its OAuth app is registered. | The provider abstraction stays generic — re-adding Google is one `newProviders` entry + a `googleProfile` branch + the secret keys. ADR-0006 (target design) unchanged. |
| 2026-09-20 | **S02 code passed a multi-agent adversarial review** (4 dimensions → verify): 6 confirmed defects fixed — DSN URL-encoding (high), onboarding route gate, data-first Auth render, logout-failure handling, JWKS rotation-overlap publication, and a `/me`-error retry. | Regression tests added for each. |
| 2026-09-20 | **JWKS is published by the gateway** (the JWT issuer), not identity — refines `services.md`. | [ADR-0011](../adr/0011-jwks-publication-and-shared-db-schema-ownership.md). Keeps single key custody per ADR-0006. |
| 2026-09-20 | **Shared-DB schema ownership via CNPG `Database` CR `spec.schemas[].owner`** (needs CNPG ≥1.25; cluster runs v1.30). Bootstrap `xlearn` owns `xlearndb`; each service role owns only its schema. | [ADR-0011](../adr/0011-jwks-publication-and-shared-db-schema-ownership.md). Migrations create tables only; validated against a real Postgres. |
| 2026-09-20 | **Minimal deps:** add only `pgx/v5` + `goose/v3`; RS256 JWT, JWKS, OAuth2+PKCE, and OIDC userinfo are **stdlib-only**. | Keeps the S01 stdlib-first ethos; full control over kid rotation + clock-skew leeway. |
| 2026-09-20 | **Outbox emission finalized; relay uses a placeholder log-publisher** (marks rows sent) until NATS JetStream lands in S05/S06. | Transactional write (domain + outbox row in one tx) is fully implemented; `internal/platform/events` gains a reusable `Relay`. |
| 2026-09-20 | **Session TTL 30d** (cookie/session lifetime not specified in ADR-0006, which only sets the ~5-min JWT TTL); opaque 256-bit session id; JWT verify leeway 30s, JWKS cache 1h with kid-overlap refetch. | Configurable via env (`SESSION_TTL`, etc.). |
| 2026-09-20 | **M0 reached** — skeleton live at `/xlearn` via Flux (`xlearn-gateway:0.1.3`); full GitOps loop proven (merge → CI → image-automation tag bump → reconcile). | S01 done. |
| 2026-09-20 | New `xlearn-*` service `deploy.yml` must grant `packages: write` on the calling job and omit `secrets: inherit` (reusable `build-push.yml` declares no secrets; new-repo token is read-only by default). | Learned from S01's two `startup_failure`s; codified in the gateway's `deploy.yml`. |
| 2026-09-20 | **Base-path model** ([ADR-0010](../adr/0010-spa-base-path-and-gateway-serving.md)): Traefik `stripPrefix: true`, Vite `base:/xlearn/`, Router `basename:/xlearn`, base-path-tolerant gateway. | One binary serves both local (`/xlearn`) and behind Traefik (`/`). |
| 2026-09-20 | S01 skeleton built: `cmd/gateway` + `internal/platform/*` + embedded React shell; `xlearn-gateway` CI/deploy + `infra` HelmRelease. | Zero external Go deps (stdlib only). Not yet pushed. |
| 2026-09-20 | Docs/architecture scaffold created; ADRs 0001-0009 accepted. | This scaffold. See open assumptions below. |
| 2026-09-20 | `review` groups scheduler + mistakes; `assessment` groups mock + progress. | Node-frugal; split later if needed ([ADR-0003](../adr/0003-service-decomposition.md)). |
| 2026-09-20 | Build-semver auto-deploy from `main` for v1; switch to release tags at 1.0. | Matches `projects-hub`/`landscape`. |

## Open assumptions (confirm / correct)

Tracked from the PRD [open questions](../prd/xlearn-prd.md#10-open-questions) and scaffold assumptions:

- [ ] **Code execution:** re-implementation is **self/auto-assessed**, not sandbox-executed in v1.
- [ ] **Curriculum source:** versioned **seed in-repo** (`curriculum/`), not an in-app CMS.
- [ ] **LLM providers at launch:** **OpenAI + Anthropic**.
- [ ] **Staging:** **none** in v1 (single node; trunk-based auto-deploy).
- [ ] **NATS placement:** its own `messaging` namespace vs inside `xlearn` — default `xlearn`.
- [ ] **Data isolation:** schema-per-service in one `xlearndb` (not DB-per-service) for v1.

## Blocked / needs input

_(none — proceeding on the assumptions above until corrected.)_
