# xLearn — Product Requirements Document

> **Status:** Draft v1 · **Owner:** @sujaykumarsuman · **Last updated:** 2026-09-20
> **Scope of this doc:** the product. Architecture lives in [`../architecture/`](../architecture/),
> decisions in [`../adr/`](../adr/), the v1 build plan in [`../v1/`](../v1/).

## 1. Problem

Most interview-prep resources are **content libraries** — a pile of problems, videos, and
editorials. They leave the *method* to the learner: what to solve next, when to revisit it,
how to know a pattern is actually internalised, and how to simulate the interview. Motivated
learners still fail interviews because they:

- solve breadth-first and never revisit, so patterns fade (no spaced repetition);
- read editorials too early and mistake *recognition* for *recall*;
- never track *why* they got a problem wrong, so the same mistake recurs;
- practise untimed, then freeze under a 45-minute clock;
- have no honest signal of readiness until the real interview.

## 2. Product thesis

**xLearn enforces the learning *method*, not just the content.** The curriculum is a rail;
the app gates progress, schedules revisions, scores re-solves, journals mistakes, runs timed
mocks, and coaches — so that finishing the path *means* interview-ready. The first path is a
16-week Go-first DSA interview course; the platform is **multi-path** by design.

## 3. Goals & non-goals

### Goals (v1)

| # | Goal | Success signal |
|---|------|----------------|
| G1 | Enforce a sequential, gated curriculum (path → phase → week → problem). | A problem cannot be "solved" without passing through attempt → (hint) → solution → re-implement → outcome. |
| G2 | Make spaced repetition automatic and unavoidable. | Every solved problem is scheduled at Day 1·3·7·21·45; due reviews take priority over new work. |
| G3 | Turn mistakes into a tracked, closing loop. | Every below-Clean outcome opens a Mistake Journal entry; entries close after 2 clean revisits. |
| G4 | Simulate the interview honestly. | 45-min timed mock, 7-dimension rubric, trend over time against phase targets. |
| G5 | Give a context-aware AI coach, using the learner's own API key. | Coach reads current-page context; is Socratic during attempts, reviewer post-solve. |
| G6 | Ship on the existing `projects.sujaykumar.dev` k3s/GitOps platform. | Live at `projects.sujaykumar.dev/xlearn`, deployed by Flux (no hand `kubectl`). |

### Non-goals (v1)

- Authoring UI for curriculum (content is **seeded** from a versioned source, not edited in-app).
- Multi-tenant orgs / teams / classrooms; social features; leaderboards.
- Native mobile apps (responsive web only; desktop-first 1440×1024 design).
- Running/judging user code against real test suites (the editor is for *re-implementation
  practice*; correctness is self/auto-assessed, not sandbox-executed) — see [Open questions](#10-open-questions).
- Billing/subscriptions. xLearn is the owner's personal + portfolio project.
- Providing model inference: the coach uses the **user's own** provider key (BYO-key).

## 4. Personas

| Persona | Who | Needs |
|---------|-----|-------|
| **Primary — "The candidate"** | Working SWE (0–5 yrs) prepping for a job switch, 1–2 hrs/day for ~16 weeks. | Structure, a next action every day, honest readiness signal, spaced revision that fits a busy schedule. |
| **Secondary — "The returner"** | Experienced dev refreshing DSA before a specific interview loop. | Fast path to weak areas, mock reps, mistake patterns. |
| **Owner — "The operator"** | @sujaykumarsuman (solo). | Low-ops deploy on existing infra; content versioned in git; add new paths later without re-architecting. |

## 5. Scope — screens & routes (v1)

Twelve screens; see [`design-system/README.md`](../../design-system/README.md) for the source of
truth and [`design-system/screens/*.dc.html`](../../design-system/screens/) for layout intent.
★ = hero screens (built first).

| Screen | Route | Purpose |
|--------|-------|---------|
| Catalog | `/xlearn` | Multi-path hub. DSA active; others "Coming soon". |
| Roadmap | `/xlearn/dsa` | 16-week roadmap: phase/week rail + overall progress ring. |
| Dashboard ★ | `/xlearn/dsa/dashboard` | "Today" — default post-login. Daily plan, revisions due, weak-area, stats. |
| Week | `/xlearn/dsa/week/:n` | Week thesis, concept links, problem list with five-touch dots + filter. |
| Concept | `/xlearn/dsa/concept/:slug` | Pattern reading, "when to reach for it", code template. |
| Problem ★ | `/xlearn/dsa/problem/:id` | Guided 3-pane workspace: statement · editor · timer HUD, gated reveal, outcome logging. |
| Revision | `/xlearn/dsa/revision` | Spaced-repetition queue; re-solve auto-scores → advance or reset-to-Day-1. |
| Mistakes | `/xlearn/dsa/mistakes` | 8-category journal, weekly weak-area banner, open/closed. |
| Mock | `/xlearn/dsa/mock` | 45-min mock: setup → live phase rail → 7-dim rubric radar + trend. |
| Progress | `/xlearn/dsa/progress` | Coverage, revision heatmap, pattern mastery, rubric trend, outcome mix. |
| Settings | `/settings` | Profile, study budget, API keys (power the coach). |
| Auth | sign-up / log-in | OAuth + 3-step onboarding (path → budget → API key). |

## 6. Functional requirements — the mechanics spec

These are the **product logic** requirements. Each is testable and maps to a service in
[`../architecture/services.md`](../architecture/services.md).

### 6.1 Guided problem flow (gated stages)

The Problem screen advances through **gated stages**; later stages are locked until the learner
acts on the earlier one.

| Stage | Timer | Unlocked content | Locked |
|-------|-------|------------------|--------|
| **0 · Attempt** | 15 min | Tags, why-it-matters, summary, one example, LeetCode/NeetCode links. | Hints, solution. |
| **1 · Hint** | 10 min | Brute force, key observation, pattern box. | Full solution. |
| **2 · Solution** | — | Optimized approach, steps, dry run, Go/C++ reference, complexity, common mistakes, discussion, variations. | — |
| **3 · Re-implement** | — | Blank editor; write the solution from memory. | — |
| **4 · Log outcome** | — | Outcome picker (see 6.2). | — |

- **R-PF1** Stages unlock in order; a locked stage's content is not delivered to the client.
- **R-PF2** Revealing the **Solution** before the attempt timer elapses **owes another attempt in 3 days** — surfaced on the reveal button *before* the click, and it creates a follow-up revision touch.
- **R-PF3** The attempt and hint timers are server-authoritative (client HUD is a mirror); refresh/return resumes the same countdown.
- **R-PF4** Re-implementation uses a blank editor (no prefill); the reference solution is not shown side-by-side during stage 3.

### 6.2 Outcome logging

- **R-OL1** On completing a problem the learner logs one of: **Clean · Rough · Assisted · Miss**.
- **R-OL2** Any outcome **below Clean** opens a **Mistake Journal** entry with the pattern pre-filled, a category picker (see 6.4), and a root-cause field.
- **R-OL3** The outcome and reveal history feed the revision schedule (6.3) — e.g. a Miss shortens the next interval / resets.

### 6.3 Five-touch spaced repetition

- **R-SR1** Every solved problem is scheduled for review at **Day 1 · 3 · 7 · 21 · 45** from first clean solve.
- **R-SR2** Each review is a **re-solve from blank** on a **20-min** timer, **auto-scored**. It **passes** only if: pattern named in < 2 min **and** solved correctly within the timer **and** complexity stated.
- **R-SR3** A **fail resets the problem to Day 1** and auto-creates a Mistake entry.
- **R-SR4** **Day 21 and Day 45** reviews run under **mock conditions** (stricter, closer to interview).
- **R-SR5** **Due reviews take priority over new problems** — the Dashboard and Revision queue surface them first; the daily plan won't advance new work while reviews are overdue.
- **R-SR6** "Due today" is materialised reliably even if the learner is offline for days (a sweep catches up missed dates).

### 6.4 Mistake journal

- **R-MJ1** Columns: Problem · Pattern · Mistake · Root cause · Correct insight · Category · Revisit · Status.
- **R-MJ2** Eight categories: **Misread · Wrong pattern · Right pattern wrong state · Off-by-one/boundary · Language bug · Complexity misjudged · Communication · Time management**.
- **R-MJ3** A **weekly summary** flags the **top category** (the current weak area) and surfaces it on the Dashboard.
- **R-MJ4** An entry **closes after 2 successful revisits**; a later fail **re-opens** it.

### 6.5 Mock interview

- **R-MK1** 45-minute timed session with a **phase rail**: 0–5 clarify · 5–10 brute force · 10–18 observation→plan · 18–33 code · 33–40 trace+edges · 40–45 complexity + follow-ups.
- **R-MK2** Scored on **7 dimensions × 1–5 = /35**: Communication, Problem understanding, Brute force, Optimisation, Code quality, Edge cases, Complexity.
- **R-MK3** Readiness targets: **≥ 24 by W13, ≥ 28 by W15, ≥ 30 pre-interview**; the trend is charted on Progress.

### 6.6 AI coach (BYO key)

- **R-AC1** A persistent bubble opens a side panel on **every** screen.
- **R-AC2** The coach reads **current-page context** (shown to the user as a context chip) — the problem, stage, recent outcome, weak area.
- **R-AC3** Behaviour is **Socratic during an active attempt** (no spoilers) and a **reviewer after solve** (critiques the learner's code).
- **R-AC4** The coach requires the **user's own provider API key**; empty state routes to Settings. The key is encrypted at rest and never returned to the client (see [ADR-0007](../adr/0007-ai-coach-byo-key-and-secrets.md)).

### 6.7 Account, onboarding, settings

- **R-AN1** Sign-in via **OAuth (GitHub / Google)** — no passwords stored.
- **R-AN2** First-run onboarding is 3 steps: **choose path → set study budget → add API key** (key is skippable; coach shows an empty state until added).
- **R-AN3** Settings manages profile, study budget + timezone + reminders, and API key config (provider, masked key, default model, enabled).

## 7. Domain model (product view)

Canonical entities (implementation mapping in [`../architecture/data-model.md`](../architecture/data-model.md)).

```mermaid
erDiagram
    PATH ||--o{ PHASE : has
    PHASE ||--o{ WEEK : contains
    WEEK ||--o{ PROBLEM : lists
    ACCOUNT ||--o{ USER_PROBLEM_STATE : tracks
    PROBLEM ||--o{ USER_PROBLEM_STATE : "state per user"
    USER_PROBLEM_STATE ||--o{ REVISION_ITEM : schedules
    USER_PROBLEM_STATE ||--o{ MISTAKE_ENTRY : "may open"
    ACCOUNT ||--o{ MOCK_SESSION : runs
    ACCOUNT ||--|| API_KEY_CONFIG : "has (optional)"

    PATH { string slug; string title; string status; int totals }
    PHASE { int order; string theme; intRange weeks }
    WEEK { int n; string title; string thesis; list concepts }
    PROBLEM { string id; string title; enum difficulty; string pattern; int week; url leetcodeUrl; url neetcodeUrl; bool isReinforcement }
    USER_PROBLEM_STATE { enum status; date firstSolvedDate; json touches; int currentTouch; enum lastOutcome }
    REVISION_ITEM { string problemId; date dueDate; int touchLevel }
    MISTAKE_ENTRY { string problemId; string pattern; string mistake; string rootCause; string insight; enum category; date revisitDate; enum status; int revisitCount }
    MOCK_SESSION { string setId; date date; string problem; json rubric7; int total35; string notes }
    ACCOUNT { uuid id; json profile; json budget; string timezone; json reminders }
    API_KEY_CONFIG { string provider; string maskedKey; string defaultModel; bool enabled }
```

- **Difficulty tokens (UI):** Easy = green `--ds-ok`, Medium = amber `--ds-warn`, Hard = red `--ds-err`.
- **UserProblemState.status:** `locked → available → attempting → solved`.
- **touches:** the five-touch record — for each of Day 1/3/7/21/45: due date + result.

## 8. Success metrics

Personal / portfolio project; metrics are **method-adherence** signals, not growth vanity.

| Metric | Target | Why |
|--------|--------|-----|
| Method completion | The owner completes the 16-week path with the mechanics enforced. | Validates the core thesis (method > content). |
| Revision adherence | ≥ 90% of due reviews done within 24h of due. | The spaced-repetition loop is actually followed. |
| Mistake closure | Median mistake entry closes within its target revisit window. | The mistake loop closes rather than accumulating. |
| Mock trend | Hits ≥24 (W13) / ≥28 (W15) / ≥30 (pre-interview). | Honest readiness signal works. |
| Ops | Zero hand-`kubectl`; deploys land via Flux; p95 screen API < 300 ms on the VPS. | Fits the low-ops GitOps platform. |

## 9. Constraints

- **Platform:** single-node k3s on a VPS, Flux + Helm GitOps, shared CloudNativePG, Traefik, SOPS/age secrets, GHCR images (see [ADR-0009](../adr/0009-deployment-and-gitops.md)). Resource-frugal by necessity.
- **Architecture:** service-oriented, **not** a monolith (per repo guide). Balanced against a single small node → see [ADR-0003](../adr/0003-service-decomposition.md).
- **Stack:** Go backend, PostgreSQL, React + TypeScript frontend.
- **Design:** dark "landscape console" theme; reuse `design-system/theme.css` tokens/components.
- **Team:** solo developer → bias to low operational overhead everywhere.

## 10. Open questions

| # | Question | Current assumption (v1) |
|---|----------|-------------------------|
| Q1 | Is user code **executed/judged**, or self/auto-assessed? | **Self/auto-assessed** — the editor is for re-implementation; auto-scoring checks pattern-named/time/complexity, not test execution. Sandbox execution is a later path. |
| Q2 | Curriculum content source of truth? | **Versioned seed** (files in-repo → `curriculum` service migration/seed), not an in-app CMS. |
| Q3 | Which LLM providers for BYO-key at launch? | **OpenAI + Anthropic** message APIs via the coach gateway; more behind a provider interface. |
| Q4 | Staging environment? | **None** in v1 (single node) — trunk-based, build-semver auto-deploy; a `xlearn-staging` namespace can be added later. |
