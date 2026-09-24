# xLearn — agent guide

Orientation for any AI coding agent or assistant working in this repository.
Tool- and model-agnostic.

xLearn is a guided-learning platform that enforces the learning **method** — sequential
unlocks, timed practice, five-touch spaced revision, a mistake journal, scored mock
interviews, and an AI coach — not just content. First curriculum: a 16-week DSA
interview-prep course (4 phases, 151 problems, Go-first); multi-path by design (more
"paths" drop in later). Hosted at `projects.sujaykumar.dev/xlearn`.

## Status

v1 is live (release-tagged 1.x; latest v1.5.2). Current build line: **v2**. Its entry points are
[`docs/v2/status.md`](docs/v2/status.md) and [`docs/v2/build-plan.md`](docs/v2/build-plan.md). v2
milestones ship as `1.x` minors until the `v2.0.0` GA. v1 maintenance and UI/UX feedback continue under
[`docs/v1/feedbacks/`](docs/v1/feedbacks/).

## Read first

- **`design-system/README.md`** — the design source of truth: tokens, component classes, all
  12 screens + routes, full product mechanics, domain model, and a suggested build order.
  **Always start here.**
- `design-system/screens/*.dc.html` — the 12 screens as **design-canvas artboards** (HTML that
  needs its design tool's runtime). Read for layout / interaction / copy intent only; they are
  **not runnable — do not ship them**.
- `design-system/theme.css` — reusable design-system CSS (dark "landscape console" look, ported
  from `github.com/sujaykumarsuman/sujaykumar-design-system`). Reuse this for the real UI.
- `docs/` — planning & architecture (layout below). Read `docs/v2/status.md` and
  `docs/v2/build-plan.md` before starting build work (`docs/v1/` for v1 feedback work).

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
  - `docs/vX/` — **per-version build execution**: `build-plan.md`, `status.md`, one plan per sprint in
    `sprints/sprint-<id>.md` and one self-contained prompt per sprint in `prompts/prompt-<id>.md`
    (v1: `sprint-07.md` with `prompt-s07.md`; v2 ids are milestone-scoped, e.g. `sprint-m3-01.md` with
    `prompt-m3-01.md`). Current version: **`docs/v2/`**; `docs/v1/` holds the shipped v1 build and its
    feedback batches.
- Service and web-app code is added during the phased build.

## Working conventions

- Match the design: use `design-system/theme.css` tokens/components; keep the dark theme.
  Difficulty tokens are Easy=green (`--ds-ok`), Medium=amber (`--ds-warn`), Hard=red (`--ds-err`).
- Respect service boundaries once defined in `docs/architecture/`; don't collapse back to a monolith.
- Git: `main` is the default branch. Follow [`docs/git-strategy.md`](docs/git-strategy.md). Commit
  messages are conventional (`feat:`, `fix:`, `docs:`, …), squash-merged via PR. Don't commit or push
  **mid-task** or on a whim — but shipping the finished sprint at end of session (below) is a
  **standing instruction that IS the authorization**, so you do **not** need a separate ask for it.
  This **supersedes** any "do not commit or push unless asked" boilerplate in the per-sprint
  `docs/v1/prompts/prompt-sNN.md` and `docs/v2/prompts/prompt-<id>.md` files.
- **End of session — land and sync (standing directive; this is your authorization to ship):** when
  the session's work is code-complete and green (build/test/lint + `sqlc diff` pass), don't leave it
  dangling — ship it **without waiting for a further prompt**: branch (`feat/…` etc.) → conventional
  commit(s) with the required attribution lines → push → open a PR in **every repo touched this
  session** (this repo and any sibling such as `../infra`) → wait for CI green (fix-then-merge on
  failure) → squash-merge → let Flux build+deploy → verify live → `git checkout main && git pull` in
  every repo touched so local `main` is fully synced. The next session must be able to start the next
  sprint from an up-to-date local `main` — never end a session with merged work unpulled or open PRs
  left hanging. Caveats: don't enable PR **auto-merge** unless asked (merge yourself once CI is green);
  and if the user says "hold / don't ship," that overrides for that session.
  **v2 exceptions** (the sprint's plan and prompt say which applies): a **design** sprint (`ds-*`) opens its
  board PR and **stops** for owner review (the owner's merge is the freeze); the **GA PR** (`ga-01`) merges
  only on the owner's explicit approval; a **spike** (`spk-*`) is throwaway and merges only its results docs
  PR; a **merge-only** sprint deploys nothing (only a tag deploys), so "verify live" happens at the tag sprint.
- Record notable technical decisions as ADRs; keep docs concise and skimmable.
- Update `docs/v2/status.md` when you finish a chunk of build work (`docs/v1/status.md` for v1 feedback work).
