# xLearn — agent guide

Orientation for any AI coding agent or assistant working in this repository.
Tool- and model-agnostic.

xLearn is a guided-learning platform that enforces the learning **method** — sequential
unlocks, timed practice, five-touch spaced revision, a mistake journal, scored mock
interviews, and an AI coach — not just content. First curriculum: a 16-week DSA
interview-prep course (4 phases, 151 problems, Go-first); multi-path by design (more
"paths" drop in later). Hosted at `projects.sujaykumar.dev/xlearn`.

## Status

Design-complete, pre-code. The app is built in **phases** from the design reference.
Current build line: **v1**.

## Read first

- **`design-system/README.md`** — the design source of truth: tokens, component classes, all
  12 screens + routes, full product mechanics, domain model, and a suggested build order.
  **Always start here.**
- `design-system/screens/*.dc.html` — the 12 screens as **design-canvas artboards** (HTML that
  needs its design tool's runtime). Read for layout / interaction / copy intent only; they are
  **not runnable — do not ship them**.
- `design-system/theme.css` — reusable design-system CSS (dark "landscape console" look, ported
  from `github.com/sujaykumarsuman/sujaykumar-design-system`). Reuse this for the real UI.
- `docs/` — planning & architecture (layout below). When present, read `docs/v1/status.md` and
  `docs/v1/build-plan.md` before starting build work.

## Tech stack

- **Backend:** Go · **Database:** PostgreSQL · **Frontend:** React + TypeScript.
- **Architecture:** distributed / service-oriented (multiple services, not a monolith).
- **Deploy:** k3s cluster on a VPS via **GitOps**. Infra + GitOps config lives in the sibling
  repo `../infra` (`github.com/sujaykumarsuman/infra`). **Follow its conventions** (GitOps tool,
  Kustomize/Helm, image registry, ingress/TLS, secrets, namespacing, naming) — inspect it before
  adding any deploy config. Never `kubectl apply` by hand; changes go through the GitOps flow.
- Open choices (message broker, API gateway/BFF, auth, migrations, query layer, frontend data
  layer, CI) are decided in `docs/adr/`.

## Repo layout

- `design-system/` — design source of truth (committed; the reference for the build).
- `docs/`
  - `docs/prd/`, `docs/adr/`, `docs/architecture/` — **global**, evolve across versions
    (ADRs are numbered, MADR-style, append-only).
  - `docs/git-strategy.md` — repo-wide git & release strategy.
  - `docs/vX/` — **per-version build execution**: `build-plan.md`, `status.md`, and
    `sprints/<sprint-NN>/{plan.md, prompts/}`. Current version: **`docs/v1/`**; future builds add
    `docs/v2/`, etc.
- Service and web-app code is added during the phased build.

## Working conventions

- Match the design: use `design-system/theme.css` tokens/components; keep the dark theme.
  Difficulty tokens are Easy=green (`--ds-ok`), Medium=amber (`--ds-warn`), Hard=red (`--ds-err`).
- Respect service boundaries once defined in `docs/architecture/`; don't collapse back to a monolith.
- Git: `main` is the default branch. Follow `docs/git-strategy.md` once it exists. Commit messages
  are conventional (`feat:`, `fix:`, `docs:`, …). **Don't commit or push unless asked.**
- Record notable technical decisions as ADRs; keep docs concise and skimmable.
- Update `docs/v1/status.md` when you finish a chunk of build work.
