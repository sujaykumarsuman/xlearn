# Prompt — Sprint 01 · Bootstrap & app shell

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-01.md`](../sprints/sprint-01.md)   ·   **Milestone:** M0 (skeleton live at /xlearn via Flux)   ·   **Prereqs:** none

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, tech stack, and working rules (match the design, respect service boundaries, ship at session end (AGENT.md land-and-sync)).
- [`../sprints/sprint-01.md`](../sprints/sprint-01.md) — the plan this prompt executes (scope, tasks, acceptance, DoD, risks).
- [ADR-0002](../../adr/0002-monorepo-vs-multi-repo.md) — single Go module monorepo, `cmd/<svc>/` + `internal/`, one image per service.
- [ADR-0008](../../adr/0008-frontend-stack.md) — Vite + React + TS (strict), React Router data router, TanStack Query, port `theme.css` (no Tailwind), SPA embedded in the gateway.
- [ADR-0009](../../adr/0009-deployment-and-gitops.md) — Flux/Helm GitOps, `charts/project`, GHCR images, image-automation setter, build-semver `>=0.1.0`, pull-based (never `kubectl apply`).
- [`../../architecture/overview.md`](../../architecture/overview.md) — the repo layout tree and cross-cutting concerns (logging, health, config).
- [`../../architecture/services.md`](../../architecture/services.md) — the gateway is the only `edge` service; it embeds the SPA and owns no schema.
- [`../../architecture/api.md`](../../architecture/api.md) — external surface at `/xlearn/api`, error envelope, `credentials: include` (nothing called yet, but the client seam matches it).
- [`../../../design-system/README.md`](../../../design-system/README.md) — tokens, `xl-*`/`ds-*` component classes, the 12 screens + routes, difficulty tokens (Easy=green/Medium=amber/Hard=red).
- `design-system/theme.css` — the CSS to port **verbatim**; `design-system/screens/Dashboard.dc.html` + `Roadmap.dc.html` — the app-shell layout/copy reference (design-only, not runnable).
- Sibling repos (inspect, do not modify except where task 4 says): `../airlift/Makefile`, `../airlift/.github/workflows/{ci,deploy}.yml`, `../airlift/embed.go` (the shape to mirror); `../landscape` and `../infra/apps/landscape.yaml` (the HelmRelease template); `../infra/apps/airlift.yaml` and `../infra/apps/image-automation.yaml`.

## Context

First build sprint. The repo is empty apart from `docs/` and `design-system/`. Nothing has been deployed yet.
This sprint delivers a deployable skeleton -- Go module + platform primitives, the ported design system and app
shell with all 12 routes as stubs, a gateway that embeds and serves the SPA under `/xlearn`, and the CI + `../infra`
wiring -- and proves the whole GitOps path (merge to `main` -> CI image -> Flux -> live) before any feature work.

## Do this (in order)

1. **Monorepo, tooling & platform primitives (gateway health skeleton)** — Init `go.mod` (module `github.com/sujaykumarsuman/xlearn`, Go 1.26), `.gitignore`, `.dockerignore`, `LICENSE`. Write `cmd/gateway/main.go` with the boot chain config -> `slog` logger -> `http.Server` -> health routes -> graceful shutdown (`SIGINT`/`SIGTERM` + context). Build `internal/platform/`: `config` (env `PORT`, `LOG_LEVEL`, `BASE_PATH=/xlearn`), `slogx` (JSON `slog` to stdout + request-id field), `httpx` (server builder + middleware chain: request-id, panic-recovery, access log), `health` (`/healthz` liveness, `/readyz` readiness -- trivially OK, no DB this sprint). Add **stub-only** seams (interface + doc comment, no impl): `internal/platform/auth` (`// TODO S02`, JWT/JWKS per ADR-0006) and `internal/platform/events` (`// TODO S05/S06`, outbox + NATS per ADR-0004). Add a `Makefile` mirroring `../airlift` (`build`/`test`/`lint`/`run`/`clean`); web + embed wiring is completed in steps 2-3.

2. **Web app: port theme.css, app shell, routing, 12 stub screens** — Scaffold `web/` (Vite + React + TS strict, ESLint/Prettier, Vitest + Testing Library) with npm scripts `dev`/`build`/`typecheck`/`lint`/`test` matching `../airlift`'s web app. Copy `design-system/theme.css` **verbatim** into `web/src/styles/`; load Inter + JetBrains Mono once via a Google Fonts `<link>` in `index.html`; verify difficulty tokens (Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`). Build the shell from the Dashboard/Roadmap artboards: `xl-app` frame; `xl-side` sidebar with `xl-brand`, `xl-pathsw`, `xl-nav` + `xl-nav__cap`, and the pure-CSS `.xl-collapse-cb:checked` collapse to a 66px icon rail; `xl-topbar` with `xl-crumb`, `xl-cmdk` (cmd-K pill), bell iconbtn, `xl-topbar__avatar` menu; `xl-content`; and a coach FAB scaffold (`xl-fab` opens an empty `xl-coach` panel with a context chip -- **no LLM**). Configure a React Router data router (`basename` `/xlearn`) with **all 12 routes** rendering placeholder screens (real chrome, stub body, "coming in Sprint NN" note): Catalog `/` (S03), Roadmap `/dsa` (S03), Dashboard ★ `/dsa/dashboard` (S09 fills it), Week `/dsa/week/:n` (S04), Concept `/dsa/concept/:slug` (S04), Problem ★ `/dsa/problem/:id` (S05), Revision `/dsa/revision` (S06), Mistakes `/dsa/mistakes` (S07), Mock `/dsa/mock` (S08), Progress `/dsa/progress` (S09), Settings `/settings` (S10), Auth (S02). Add a typed API-client seam at `/xlearn/api` (`credentials: include`, error-envelope type from `api.md`) and a TanStack Query provider -- wired, calling nothing.

3. **Gateway embeds & serves the SPA under /xlearn** — Add `//go:embed web/dist` in the gateway and make `make build` build `web/dist` first, then the Go binary (mirror `../airlift/embed.go`). Serve `GET /xlearn/*` from the embedded FS with SPA fallback to `index.html`; keep `/xlearn/api/healthz` (app) and the k8s `/healthz`,`/readyz`. Set content-types, `immutable` long-cache for hashed assets, `no-cache` for `index.html`. Serve entirely from the embedded FS -- **no runtime disk writes** (read-only rootfs). Decide + document **one** base-path model consistent with step 4's Traefik route (recommended: Traefik `stripPrefix: true` -> pod sees `/`; Vite `base: /xlearn/` + Router `basename` `/xlearn`). No auth/DB/business endpoints.

4. **CI + infra wiring -> deploy (reach M0)** — Write `deploy/gateway.Dockerfile` (multi-stage: node builds `web/dist` -> Go static build -> distroless/nonroot, `EXPOSE 8080`). Add `.github/workflows/ci.yml` (go: `gofmt -l`, `go vet`, `go test -race`; web: `npm ci`, `typecheck`, `lint`, `test`, `build`) on PR + push `main`, and `.github/workflows/deploy.yml` (push `main`) calling `sujaykumarsuman/.github/.github/workflows/build-push.yml@main` with image `ghcr.io/sujaykumarsuman/xlearn-gateway` + `dockerfile: deploy/gateway.Dockerfile`. In sibling `../infra` (separate PR): add the `xlearn` namespace; `apps/xlearn-gateway.yaml` HelmRelease off `charts/project` (copy `apps/airlift.yaml`/`apps/landscape.yaml`) with the image + the `# {"$imagepolicy": "flux-system:xlearn-gateway:tag"}` setter marker and route `pathPrefix: /xlearn`, `stripPrefix: true`, `redirectSlash: true`, `priority: 100`; and the image-automation entry in `apps/image-automation.yaml` (ImageRepository + ImagePolicy semver `>=0.1.0`). No DB/NATS/secrets (gateway is stateless). Never `kubectl apply` -- let Flux reconcile.

## Constraints

- Port `design-system/theme.css` **verbatim** -- CSS variables + `ds-*`/`xl-*` classes, **no Tailwind**; CSS Modules only for screen-local styles (ADR-0008).
- Keep the dark landscape-console look and the difficulty tokens (Easy=green `--ds-ok` / Medium=amber `--ds-warn` / Hard=red `--ds-err`).
- One Go module, `cmd/gateway/` + shared `internal/platform/`; the SPA is embedded in and served by the gateway (ADR-0002/0008).
- Base path: pick **one** model and set Vite `base`, Router `basename`, and the Traefik route consistently.
- Read-only rootfs: the gateway serves only from the embedded FS; no runtime disk writes.
- Match `../infra` conventions exactly: copy the `airlift`/`landscape` HelmRelease shape, GHCR image naming `xlearn-<svc>`, the image-automation setter marker, semver range `>=0.1.0`. Pull-based GitOps -- never `kubectl apply` by hand.
- `auth` and `events` are stub-only seams this sprint (no impl). No auth/DB/NATS/business logic yet.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).

## Deliverables

- `go.mod`, `Makefile`, `.gitignore`, `.dockerignore`, `LICENSE`.
- `cmd/gateway/main.go`; `internal/platform/{config,slogx,httpx,health}`; stub-only `internal/platform/{auth,events}`.
- `web/` Vite + React + TS app: ported `theme.css`, the `xl-*` app shell (sidebar/topbar/coach scaffold), React Router with all 12 stub routes, the typed `/xlearn/api` client seam + TanStack Query provider.
- Gateway embedding + serving `web/dist` under `/xlearn/*` with SPA fallback; `/xlearn/api/healthz` + k8s `/healthz`,`/readyz`.
- `deploy/gateway.Dockerfile`, `.github/workflows/ci.yml`, `.github/workflows/deploy.yml`.
- `../infra` (separate PR): `xlearn` namespace, `apps/xlearn-gateway.yaml` HelmRelease, image-automation entry.

## Update status

- As each task lands, set its row in [`../sprints/sprint-01.md`](../sprints/sprint-01.md) to ✅ (🔄 while in progress); set _Overall_ when all four tasks are ✅.
- Mirror the sprint's state into [`../status.md`](../status.md): the **Sprint board** row, and the **Milestones** table (M0). Add a **Decisions log** line for any notable call (e.g. the base-path model chosen).
- Record notable technical decisions as ADRs under `docs/adr/` (append-only, MADR-style).

## Done when (acceptance)

- [ ] `make build` produces a `gateway` binary embedding `web/dist`; `make test` and `make lint` are green.
- [ ] Locally the shell renders at `/xlearn`, all 12 routes render (real sidebar/topbar + placeholder bodies), deep links work via SPA fallback, and `/xlearn/api/healthz` + `/readyz` return 200.
- [ ] The SPA uses the ported `theme.css` (dark landscape-console look; difficulty tokens present); sidebar collapse + coach FAB scaffold work.
- [ ] `ci.yml` + `deploy.yml` are present; `deploy.yml` builds + pushes `xlearn-gateway` on merge to `main`.
- [ ] `../infra` has the `xlearn` namespace + `apps/xlearn-gateway.yaml` + the image-automation entry; **Flux deploys it and the skeleton is reachable at `projects.sujaykumar.dev/xlearn` (M0)**.
- Ship at session end per AGENT.md land-and-sync (standing directive; no separate ask needed).
