# Sprint 01 — Bootstrap & app shell

> **Milestone:** M0 — skeleton live at /xlearn via Flux   ·   **Design phase:** D1 Foundation
> **Prereqs:** none   ·   **Unblocks:** everything (S02-S12)
> **Execute with:** [`../prompts/prompt-s01.md`](../prompts/prompt-s01.md) — one prompt, one session.

## Status

_Overall:_ 🔄 Code + wiring complete and locally verified; **M0 (Flux deploy) pending merge to `main` + the `infra` PR** (not pushed — repo policy is "don't commit/push unless asked").

| # | Task | Status |
|---|------|--------|
| 1 | Monorepo, tooling & platform primitives (gateway health skeleton) | ✅ |
| 2 | Web app: port theme.css, app shell, routing, 12 stub screens | ✅ |
| 3 | Gateway embeds & serves the SPA under /xlearn | ✅ |
| 4 | CI + infra wiring -> deploy (reach M0) | 🔄 files written & validated (helm template, docker build, YAML); Flux deploy pending push |

**Verified locally (2026-09-20):** `make build` embeds `web/dist` into `bin/gateway`; `make test` +
`make lint` green (Go race + Vitest + eslint/tsc); the shell renders at `/xlearn` with all 12 routes,
SPA deep-link fallback, sidebar collapse + coach FAB working; `/xlearn/api/healthz`, `/readyz`,
`/healthz` return 200; hashed assets `immutable`, `index.html` `no-cache`; base-path tolerance works
both stripped (`/api/healthz`) and un-stripped (`/xlearn/api/healthz`). The `deploy/gateway.Dockerfile`
image builds (15.5 MB distroless) and runs under `--read-only` rootfs. `infra` renders via
`helm template` + `helm lint` (IngressRoute `/xlearn` stripPrefix, `readOnlyRootFilesystem: true`).

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + the
> M0 milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Goal

Stand up the empty monorepo into a deployable skeleton and prove the entire GitOps path end to end before any
feature work. This sprint delivers the Go module + platform primitives, the ported design system and app shell
(sidebar, topbar, coach scaffold) with all 12 routes as stubs, a gateway that embeds and serves the SPA under
`/xlearn`, and the CI + `../infra` wiring that lets Flux deploy it. Reaching **M0** means the skeleton is live at
`projects.sujaykumar.dev/xlearn` from a merge to `main`, so every later sprint ships on a road already paved.

## Scope

**In**
- One Go module `github.com/sujaykumarsuman/xlearn` (Go 1.26); `cmd/gateway/main.go` boot chain (config -> logger -> HTTP server -> health, graceful shutdown).
- `internal/platform/{config,slogx,httpx,health}` primitives; **stub-only** interfaces (no impl) for `internal/platform/{auth,events}`.
- `web/` Vite + React + TS (strict) app: `theme.css` ported verbatim, Inter + JetBrains Mono loaded once, the `xl-*` app shell, React Router data router (basename `/xlearn`) with all 12 routes rendering real chrome + placeholder bodies, a typed API-client seam + TanStack Query provider (no endpoints called).
- Gateway serves the embedded `web/dist` under `/xlearn/*` (SPA fallback) plus `/xlearn/api/healthz` and k8s `/healthz`,`/readyz`; one documented base-path model consistent with the Traefik route.
- `deploy/gateway.Dockerfile`, `.github/workflows/{ci,deploy}.yml`, and the `../infra` additions (xlearn namespace, `apps/xlearn-gateway.yaml` HelmRelease, image-automation entry).

**Out (later sprints)**
- Any auth / sessions / JWT-JWKS -> **S02** (identity). Screens' Auth + onboarding bodies stay stubs.
- Curriculum, Catalog/Roadmap data -> **S03**; Week + Concept -> **S04**.
- practice / Problem, timers, outcomes, NATS + outbox -> **S05**; review scheduler -> **S06**; mistakes/notifications -> **S07**.
- Mock -> **S08**; Progress + Dashboard aggregation -> **S09**; Settings -> **S10**; coach LLM wiring -> **S11**.
- Any DB / NATS / secrets in `../infra`: none this sprint (the gateway is stateless).

## Tasks

### 1 · Monorepo, tooling & platform primitives (gateway health skeleton)

Initialise the single-module monorepo per [ADR-0002](../../adr/0002-monorepo-vs-multi-repo.md) and the
[repo layout](../../architecture/overview.md#repository-layout): `go.mod` (module `github.com/sujaykumarsuman/xlearn`,
Go 1.26), `.gitignore`, `.dockerignore`, `LICENSE`. Create `cmd/gateway/main.go` with the boot chain
config -> `slog` logger -> `http.Server` -> health routes -> graceful shutdown (context + `SIGINT`/`SIGTERM`).
Build `internal/platform/`:

- `config` — 12-factor env load: `PORT`, `LOG_LEVEL`, `BASE_PATH` (default `/xlearn`).
- `slogx` — JSON `slog` handler to stdout with a request-id field (per the [logging concern](../../architecture/overview.md#cross-cutting-concerns)).
- `httpx` — `http.Server` builder + a middleware chain (request-id, panic-recovery, access log).
- `health` — `/healthz` (liveness) and `/readyz` (readiness) handlers per the health concern; `readyz` is trivially OK this sprint (no DB).

Add **stub-only** platform seams (interfaces + doc comment, no implementation): `internal/platform/auth` (JWT/JWKS
verify seam, `// TODO S02` per [ADR-0006](../../adr/0006-authn-authz.md)) and `internal/platform/events`
(outbox + NATS publish/consume seam, `// TODO S05/S06` per [ADR-0004](../../adr/0004-inter-service-comms-and-events.md)).
Add a `Makefile` mirroring the sibling `airlift` targets (`build`, `test`, `lint`, `run`, `clean`); the `web`
targets are added in task 2 and the `web build -> Go build` order is wired in task 3.

### 2 · Web app: port theme.css, app shell, routing, 12 stub screens

Scaffold `web/` with Vite + React + TypeScript (strict), ESLint/Prettier, and Vitest + Testing Library, with npm
scripts `dev`/`build`/`typecheck`/`lint`/`test` matching the sibling `airlift` web app
([ADR-0008](../../adr/0008-frontend-stack.md)). Copy `design-system/theme.css`
([design-system](../../../design-system/README.md)) **verbatim** into `web/src/styles/` and load Inter +
JetBrains Mono once via a Google Fonts `<link>` in `index.html` (centralised, not per-screen). Confirm the difficulty
tokens render: Easy=`--ds-ok` (green), Medium=`--ds-warn` (amber), Hard=`--ds-err` (red).

Build the app shell from the `Dashboard.dc.html` / `Roadmap.dc.html` artboards (reference only, not runnable):
the `xl-app` frame; the collapsible `xl-side` sidebar (`xl-brand`, `xl-pathsw` path switcher, `xl-nav` items +
`xl-nav__cap` captions, collapse via the pure-CSS `.xl-collapse-cb:checked` toggle -> 66px icon rail); the `xl-topbar`
(`xl-crumb`, `xl-cmdk` cmd-K search pill, bell iconbtn, `xl-topbar__avatar` account menu); `xl-content`; and the coach
FAB scaffold (`xl-fab` opens an empty `xl-coach` panel with a context chip -- **no LLM**, wired in S11). Configure a
React Router data router with `basename` `/xlearn` and **all 12 routes** rendering placeholder screens (real
sidebar/topbar chrome, stub bodies, each with a "coming in Sprint NN" note): Catalog `/` (S03), Roadmap `/dsa` (S03),
Dashboard ★ `/dsa/dashboard` (skeleton now; aggregation S09), Week `/dsa/week/:n` (S04), Concept `/dsa/concept/:slug`
(S04), Problem ★ `/dsa/problem/:id` (S05), Revision `/dsa/revision` (S06), Mistakes `/dsa/mistakes` (S07), Mock
`/dsa/mock` (S08), Progress `/dsa/progress` (S09), Settings `/settings` (S10), Auth (S02). Add a typed API-client seam
at `/xlearn/api` (`credentials: include`, error-envelope type per [`api.md`](../../architecture/api.md)) and a
TanStack Query provider -- wired but calling no endpoints yet.

### 3 · Gateway embeds & serves the SPA under /xlearn

Add `//go:embed web/dist` in the gateway and make `make build` build the web bundle first, then the Go binary.
Serve `GET /xlearn/*` from the embedded FS with SPA fallback to `index.html`; also expose `/xlearn/api/healthz`
(app-level) and the k8s `/healthz`,`/readyz` from task 1. Set correct content-types, `immutable` long-cache for
hashed assets and `no-cache` for `index.html`. **No runtime disk writes** (the pod runs read-only rootfs per the
chart defaults, [ADR-0009](../../adr/0009-deployment-and-gitops.md)). Decide and document **one** base-path model
consistent with the Traefik route in task 4 -- recommended: Traefik `stripPrefix: true` so the pod sees `/`, with
Vite `base: /xlearn/` and Router `basename` `/xlearn`. No auth / DB / business endpoints yet (external surface per
[`api.md`](../../architecture/api.md); the gateway is the only edge service, [services](../../architecture/services.md)).

### 4 · CI + infra wiring -> deploy (reach M0)

In the xlearn repo: write `deploy/gateway.Dockerfile` (multi-stage -- node builds `web/dist` -> Go static build ->
distroless/nonroot, `EXPOSE 8080`). Add `.github/workflows/ci.yml` (go job: `gofmt -l`, `go vet`, `go test -race`;
web job: `npm ci`, `typecheck`, `lint`, `test`, `build`) on PR + push to `main`, and `.github/workflows/deploy.yml`
on push to `main` calling the reusable `sujaykumarsuman/.github/.github/workflows/build-push.yml@main` with image
`ghcr.io/sujaykumarsuman/xlearn-gateway` and `dockerfile: deploy/gateway.Dockerfile`. In the sibling `../infra` repo
(separate PR, per [ADR-0009](../../adr/0009-deployment-and-gitops.md)): add the `xlearn` namespace; `apps/xlearn-gateway.yaml`
as a HelmRelease off `charts/project` (copy `apps/airlift.yaml`/`apps/landscape.yaml` as the template) with the GHCR
image + the `# {"$imagepolicy": "flux-system:xlearn-gateway:tag"}` setter marker and route `pathPrefix: /xlearn`,
`stripPrefix: true`, `redirectSlash: true`, `priority: 100` (matching the task-3 base-path model); and the
image-automation entry in `apps/image-automation.yaml` (ImageRepository + ImagePolicy, semver range `>=0.1.0`).
**No DB/NATS/secrets** this sprint (the gateway is stateless). Never `kubectl apply` by hand -- Flux reconciles.

## Acceptance criteria

- [ ] `make build` produces a `gateway` binary embedding `web/dist`; `make test` and `make lint` are green.
- [ ] Locally the shell renders at `/xlearn`, all 12 routes render (real sidebar/topbar + placeholder bodies), deep links work via SPA fallback, and `/xlearn/api/healthz` + `/readyz` return 200.
- [ ] The SPA uses the ported `theme.css` (dark landscape-console look; difficulty tokens Easy=green/Medium=amber/Hard=red present); sidebar collapse and the coach FAB scaffold work.
- [ ] `ci.yml` + `deploy.yml` are present; `deploy.yml` builds and pushes `xlearn-gateway` on merge to `main`.
- [ ] `../infra` has the `xlearn` namespace + `apps/xlearn-gateway.yaml` + the image-automation entry; **Flux deploys it and the skeleton is reachable at `projects.sujaykumar.dev/xlearn` (M0)**.

## Definition of Done

CI green · deployed to prod via Flux (no hand `kubectl`) · screens match the artboards · acceptance
criteria met · statuses updated (this file + [`../status.md`](../status.md)) · notable decisions recorded as ADRs.

## Risks / watch-outs

- **Fonts:** the `.dc.html` artboards load Inter + JetBrains Mono via a per-screen Google Fonts `<link>`. Centralise this into `index.html` once so screens do not each refetch.
- **Base path:** the SPA/router is rooted at `/xlearn` while Traefik uses `stripPrefix`. Pick one model and set Vite `base` + Router `basename` + the route consistently, or deep links and asset URLs break.
- **Artboards are not runnable:** `.dc.html` files need their design-tool runtime (`support.js`, `<script type="text/x-dc">`). Port markup/CSS to real React + `theme.css`; do not ship the artboards.
- **Read-only rootfs:** the gateway pod runs a read-only filesystem (chart default). The gateway must serve entirely from the embedded FS and never write to disk at runtime.
- **One repo, many images:** this is a small deviation from the infra one-image-per-repo norm; keep the `apps/` side clean by copying the `airlift`/`landscape` HelmRelease shape exactly.
