# v1 — UI/UX feedback tracker

Post-1.0 feedback on the shipped v1 app, tracked to implementation. Each feedback
is a file `FNNN-<slug>.md` with a fixed shape: **Feedback** (what was asked) ·
**Decision** (how we resolved the open calls) · **Scope / changes** · **Status** ·
**Notes**. Newest batch at the top of the index.

Legend: ⬜ planned · 🔄 in progress · ✅ done (in local review) · 🚀 shipped to prod · ⛔ blocked.

## Working mode for this batch

- **Review before ship.** Unlike the standing land-and-sync directive, this batch is
  built and left running as a **local docker-compose** (app + local Postgres + NATS)
  for the owner to review. Land-and-sync (branch → PR → CI → merge → Flux → verify →
  sync `main`) happens **only after an explicit go-ahead**, across every repo touched
  (this repo + `sujaykumarsuman.github.io` for projects-hub).
- Local review login is an **env-gated dev login** (`DEV_AUTH=1`), off by default and
  never enabled in the prod images — see [F002](F002-start-path-enrollment.md) /
  [ADR-0022](../../adr/0022-path-enrollment-and-dev-login.md).

## Index

| ID | Feedback | Status |
|----|----------|--------|
| [F001](F001-shell-and-nav-restructure.md) | Shell & nav restructure: no left nav on the home/Catalog, curriculum-scoped nav, curriculum selector → top bar (replacing search), drop Settings/Progress from the nav | 🚀 shipped (`v1.1.0`) |
| [F002](F002-start-path-enrollment.md) | Stop DSA being pre-"Active": a per-user **Start path** action; once started show current day, streak, and today's scheduled item | 🚀 shipped (`v1.1.0`) |
| [F003](F003-rename-dsa-path.md) | Rename **DSA Interview Mastery** → **Data Structures & Algorithms** | 🚀 shipped (`v1.1.0`) |
| [F004](F004-projects-hub-entry.md) | Add xLearn to **projects-hub** (`projects.sujaykumar.dev`), as the first project | 🚀 shipped (`github.io#28`) |
| [F005](F005-curriculum-gating-and-problems-arena.md) | Rounds 2–3: enrollment gates solving, scheduled-only counting, the **Problems** arena (untimed study), Roadmap wired, top-bar overlap fix | 🚀 shipped (`v1.1.0`) — one dual-state nuance deferred |
| [F006](F006-settings-redesign-and-coach-model.md) | Round 4–5: **Settings redesign** (section rail + cards, pill/circle controls) + **multi-provider coach** — connect Anthropic *and* OpenAI, each with its model/name, one marked the **default**; a **header quick-switch** (provider pill + model dropdown) sets it from anywhere | 🔄 in local review |

**Shipped as `v1.1.0`** (2026-09-22): merged (xlearn#32) → `v1.1.0` tag → Flux deploy → verified live
(gateway `v1.1.0`; the arena/enrollment/progress routes session-gated; dev login 404 in prod).
projects-hub shipped separately (`sujaykumarsuman.github.io#28`). One F005 nuance (ahead/practice
attempts *marking* personal-solved + re-attempt-from-course-to-credit) is deferred pending a careful
practice-engine change.

## Forward context (not this batch)

The owner's larger direction is **v2**: automate the manual/self-managed loop with a real
**coding judge** (run submissions against test cases → structured AI feedback → auto-mark
outcomes / mistakes / revision), an **AI interviewer**, and first-class **storage** of
curricula / concepts / problems / solutions / test-cases / attempts (a mix of Postgres +
MinIO object storage, wired through `../infra` like cnpg/longhorn). Navigation, the solving
ground, and the judge are expected to **differ per curriculum** (e.g. System Design leans on
a structured questionnaire or an in-app Excalidraw canvas + AI evaluation rather than a code
judge). This batch only keeps the v1 design **expandable** toward that; none of it is built here.
