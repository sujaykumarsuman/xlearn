# F005 — Curriculum gating, scheduled counting & the Problems arena

**Status:** 🔄 in progress (core shipped to local review; one nuance deferred) · **Opened:** 2026-09-22

## Feedback (review round 2)

- I can open a path via "explore roadmap" and **solve a problem without starting** — and the
  state isn't reflected consistently across screens.
- I should need to **start the curriculum before being able to solve** any question. Once
  started, I follow the **curriculum timeline**.
- I'm okay attempting **any** problem, but it **shouldn't count** toward curriculum activity
  **unless it's scheduled**.
- Simplification: make a separate **"Problems"** nav section (below Roadmap) — the course attempt
  can be locked to the schedule, but a user may attempt **any** problem in that section.
- A solved problem stays **marked solved** but **still open for attempt** in the course; you'd
  **attempt it again from the course** to make it count toward "course done".
- A card (the Roadmap's sticky summary) **overlaps** the top-bar curriculum selector.

## Decision

Two solved-states — this maps onto the existing services:
- **Personal "solved"** = practice `status` (the green dot; set by any completed attempt).
- **Course "done"** = the assessment projections (streak / progress % / the frontier), fed by
  `problem_solved` events — which are emitted only for **scheduled** solves.

Enforced at the **gateway** (the composition layer), so no risky event-stream rewrite:
- **Enrollment gate:** every practice write (`attempt`/`reveal`/`outcome`) requires the path to be
  started → `403 not_enrolled`; the Problem workspace shows a **"Start the path"** gate.
- **Scheduled counting:** for a NEW-problem `outcome`, "scheduled" = the problem's week ≤ the
  **frontier** (`currentWeek` — the lowest week with an unsolved core problem). An **ahead** outcome
  is acknowledged `{counted:false}` and **not** forwarded (no solve, no events, no revision). Due
  revisions ride their own already-due-gated path, so they're inherently scheduled.
- **Problems arena** (`/dsa/problems`, new nav item below Roadmap): a flat, by-week list to open +
  attempt any problem; ahead weeks are labelled "free practice". **Open without a course
  dependency, and fully decoupled from course state** — arena links carry `?practice=1`, so the
  workspace opens as an **untimed study view**: the agg delivers **all stages** for study, and the
  arena makes **no server calls** (every write is a no-op) — so opening a problem in the arena
  **creates zero practice state** (no attempt, no timer). This closes a leak where an arena attempt
  showed as course "in progress". Reveal is client-side; nothing is saved; course credit comes only
  from solving via the schedule. The COURSE flow (Today/Week → problem, no flag) keeps the
  enrollment gate + schedule check + the real timed guided engine.
- **Breadcrumb fix:** the top-bar crumb for `/dsa/problem/:id` linked the "problem" segment to the
  non-existent `/dsa/problem` (404); it now points at the arena `/dsa/problems`.
- **Consistency:** the Roadmap rail (overall ring, Current week, streak, revisions due, phase
  meters) is wired to `GET /progress` (previously hard-coded `0% / Not started`).
- **Overlap fix:** the top bar gets `position:relative; z-index:20` so its dropdowns paint above the
  Roadmap's `position:sticky` summary column (below the coach at z-index 45).

Recorded in [ADR-0022](../../adr/0022-path-enrollment-and-dev-login.md) (enrollment) + this doc.

## Deferred (needs a careful practice-engine change)

The **"mark solved without counting + re-attempt from the course to earn credit"** dual-state is
**not** built this round. Today an **ahead** outcome is acknowledged but **not recorded** as
personally-solved (it doesn't turn the dot green). To make an ahead attempt mark the dot green while
still not counting — and to re-open a solved problem for a counting course attempt — needs a practice
`counts` flag (emit-gating) + relaxing `ErrAlreadySolved`, a change to a heavily-tested service best
done on its own with tests. Scoped as the next increment (confirm before touching the practice engine).

## Changes done

- gateway: `internal/gateway/scheduling.go` (enrolledPaths/isEnrolled/requireEnrolled/pathFrontier/
  problemGateFor), the `attempt/reveal/outcome` gate in `proxyPracticeWrite`, the Problem-agg `gate`
  block, `GET /api/paths/{slug}/problems` proxy, and `enrolled`/`currentWeek`/`revisionsDue` on
  `/progress`. openapi + drift green. Tests: `TestBFFPracticeGatedByEnrollment` + harness enrollment.
- web: `lib/curriculum.ts` (`ProblemGate`, `usePathProblems`, `isAhead`/`OutcomeResponse`, the
  `practice` flag on the write calls), `lib/progress.ts` fields, `nav.ts` (Problems item +
  `/dsa/problem/:id` breadcrumb → arena), `screens/Problems.tsx` (arena, links `?practice=1`),
  `screens/Problem.tsx` (Start gate + practice/ahead banners + counted:false + practice mode),
  `screens/Roadmap.tsx` (rail wired), `styles/app.css` (top-bar z-index). Tests: `Problems.test.tsx`,
  Problem gate tests, `TestBFFPracticeArenaOpen`, breadcrumb test.

**Verified live** (docker-compose): not-enrolled `attempt` → 403; after Start, a Week-1 solve counts
(solved 1, streak 1); an ahead (Week-2) outcome → `{counted:false}` and progress stays 1; `/progress`
reports enrolled/currentWeek; the Problems arena lists all 14 seeded problems.
