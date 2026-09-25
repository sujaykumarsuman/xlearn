# Prompt — Sprint m1-06 · withhold() on every surface + Markdown renderer + revision v2 (AB03)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-06.md`](../sprints/sprint-m1-06.md)   ·   **Milestone:** M1 (M1b slice, ships in `v1.7.0`)   ·   **Prereqs:** [m1-05](../sprints/sprint-m1-05.md), [ds-m1-01](../sprints/sprint-ds-m1-01.md) (AB03 frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — repo conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m1-06.md`](../sprints/sprint-m1-06.md) — the `withhold()` rule table is the spec.
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1 (the public/private rule: "public, withheld while live").
- [t1 §3.1](../research/t1-content-data-model.md#31-visibility-tiers), [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes)
  (the stage-gating row lists every surface; the Markdown row), [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation) (Markdown profile).
- [t0 §7](../research/t0-extensibility-frame.md#7-screens--routing) (arena and honesty gating), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)
  (pattern chip only from the hint stage; §6.4 concepts withheld; §6.6 touch formats), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (attempt mode includes "never solved").
- [ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) — why an XSS on this origin matters (soft MI-5b gate).
- [rollout §4 M1](../rollout-plan.md#4-per-milestone-detail) (scope T1/T5/T6) and [§9](../rollout-plan.md#9-artboards-by-milestone) (AB03, freeze rule).
- AB03 — the frozen board `design-system/screens/v2/AB03-revision-v2.html`; v1 reference [`Revision.dc.html`](../../../design-system/screens/Revision.dc.html)
  and [`Problem.dc.html`](../../../design-system/screens/Problem.dc.html) (its line 104 chip is the leak); [`theme.css`](../../../design-system/theme.css).
- Code: `internal/gateway/{bff,aggregate,dashboard,review,mistakes,progress,coach,practice,cache}.go`, the gateway tests
  (`bff_test.go` harness, `problem_test.go`, `openapi_drift_test.go`), `internal/curriculum/handlers.go`,
  `web/src/components/Markdown.tsx`, `web/src/screens/{Revision,Problem,Concept}.tsx`, `web/src/lib/revision.ts`, `web/package.json`.

## Context

v1 gates hint/solution **sections** by unlocked stage in the problem aggregate only. Everything else leaks: `pattern` is
ungated in list, bulk and detail, on the Revision card (the recall answer), in the arena (`?practice=1` returns every
stage), in Today and mistakes enrichment, and in the coach context (`gateway/coach.go:313`). v2's rule (ADR-0027 §1): the
pattern/topic, concepts, hint and solution stages and solution facts are **withheld while an item is live** — an open
counted attempt or a due/live touch — by **one gateway function on every surface**, proven by a route-enumeration test.
v2 content also needs a real Markdown renderer (tables, code, images, links) that is safe on an origin an XSS could turn
into cluster-admin. This sprint is last on the serialized gateway router (m1-03 → m1-04 → m1-05 → **m1-06**);
[m1-07](../sprints/sprint-m1-07.md) then builds the coach mode gate on your predicate and cuts `v1.7.0`.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB03 frozen: [ds-m1-01](../sprints/sprint-ds-m1-01.md) merged (the merge is the freeze, D40; the board exists on `main`)
- [ ] [m1-05](../sprints/sprint-m1-05.md) merged (router serialization; `httpx.ReadBody`)
- [ ] [m1-03](../sprints/sprint-m1-03.md) merged (course-scoped routes + aliases, `useCourse()` → `{slug, view, status}`) and `v1.6.0` live
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no peer PR edits `internal/gateway/**`
      (except m1-10's additive coach change: new `coach_models.go`, one route row, OpenAPI path, new test files),
      `Markdown.tsx` or `Revision.tsx`
- [ ] Soft (note, don't stop): is MI-5b ([mi-04](../sprints/sprint-mi-04.md)) live? Record the answer in the PR

## Do this (in order)

1. **[X] Branch** `feat/m1-06-withhold` off an up-to-date `main`. For the golden diff, capture the **before** JSON of
   Week, Today, revision due, mistakes and problem (course + arena). Take it from the `internal/gateway/bff_test.go`
   harness fixtures (gateway composites over fake upstreams) or from `docker compose up` with dev login. The e2e
   harness (`internal/e2e/coreloop_test.go`) drives only the practice, review and assessment HTTP surfaces and never
   runs the gateway.
2. **[X] `withhold.go` (task 1)** — `itemState`, `live()`, the surface enum and the rule table from the plan as a pure,
   table-tested function; the JSON field stripper reusing `aggregate.go`'s stage filter.
3. **[X] Item states (task 1)** — `g.itemStates(ctx, account, ids)`: parallel practice `GET /state?ids=` + review
   `GET /revisions/due`, `aggCallTimeout`, reuse handler-fetched state, **fail closed** (treat as live) with one WARN.
4. **[X] Apply (task 1)** — list/week/Today/revision due/mistakes (curriculum join and review's `mistake_entry.pattern`)/
   concept item chips/problem workspace (stage-gated while unsolved or in an open attempt, so the chip comes from the hint
   stage; a solved, not-live item stays as v1, including solved blind; a due touch hides hint+solution+pattern)/arena GET
   (live → attempt stage only)/coach context (live or never solved → stripped; `coach.go:287-313` only). Cover the
   course-scoped routes and their DSA aliases. Web: `Problem.tsx:336` and `Revision.tsx:235` render the chip only when
   `pattern` is present.
5. **[X] Route-enumeration test (task 2)** — `withholdPolicy` on `apiRoute`; set it on every row (m1-10's
   `/api/coach/models` → exempt "model catalog; no item data" if present); `withhold_routes_test.go` static check +
   behavioural sentinel sweep (items 1 open attempt, 2 due touch, 3 solved **blind** not due: its pattern must be on
   lists **and** its workspace GET) + coach request assertions.
6. **[X] `solution_facts` (task 5)** — curriculum serves it only inside the solution-stage block (never top-level or in
   list/bulk); fixture test even if m1-09 doesn't persist it yet; gateway test that it appears only after the solution
   stage unlocks and never while live.
7. **[X] Markdown renderer (task 3)** — decide the implementation (recommended: `react-markdown` + `remark-gfm` with
   `skipHtml`, `urlTransform`, `allowedElements`; `lowlight` with go/cpp/python/sql/json/bash/text → React nodes,
   lazy-loaded). Keep `InlineMD`, the callout, `.cn`. `resolveAsset` prop; https-only links with
   `rel="noopener noreferrer nofollow"`; no raw HTML; `<img>` only. Write `Markdown.test.tsx` with the XSS corpus and
   the v1 parity snapshot over `curriculum/`. Report the bundle delta. Write the ADR (next free number ≥ 0036 after the
   parallel-sessions check), MADR-style, Accepted, linking t1 §7.2/§10.
8. **[X] Revision v2 (task 4)** — AB03 **F1–F4, F6, F9** only.
   - Badges from the manifest band (`useCourse().view`; add the band's display fields to curriculum's learner-safe
     manifest view if missing, never criteria keys): "Re-solve · 20:00" for DSA, the mock marker at L4–5, and the
     "Recall" variant only from a band (fixture).
   - F1: an `xl-lock` "Pattern hidden while due" instead of the chip.
   - F2: the re-solve with the pattern hidden.
   - F3/F4: the pattern revealed in the scored result.
   - F6: the empty state. Show the "Next review" date only if the due payload carries one; else log the gap.
   - F9: narrow.
   - **Defer by name:** F5 → m2-04/p-02; F7 overdue (R-SR5) → m2-04; F8 all-courses → p-02.
   - `revision.ts` types; `Revision.test.tsx`; screenshots 1440/390 px beside AB03.
9. **[X] Verify** — `gofmt`, `go vet`, `go test -race ./...`, `sqlc diff`, e2e (`go test -tags e2e -race ./internal/e2e/...`);
   the **after** JSON diff vs step 1 shows only withheld fields on live items (explain each in the PR); measure
   Week/Today latency on compose before/after; `npm --prefix web run` `typecheck`, `lint`, `test`, `build`; manual smoke
   on compose (Problem, arena, Revision, Concept pages) with the CSP from m1-04 active — no console CSP violations.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the gateway composes and withholds;
  it owns no schema and never reads another service's tables. curriculum stays stateless about users (it gates facts by
  stage placement, not by account). No migration expected; if one appears, additive only (ADR-0034 §3) with `sqlc diff` clean.
- **No events, no NATS** here — no ACL PR, no consumers-before-producers step.
- **Withholding is a nudge:** don't build secrecy machinery; do make every surface consistent and fail closed.
- **Serialized router:** you are last in m1-03 → m1-06; touch only what the plan lists; keep `coach.go` edits to the
  context composition (m1-07 owns the modes).
- **Frontend:** `theme.css` tokens/components verbatim, dark theme, match AB03; no `dangerouslySetInnerHTML`, no inline
  `<style>`/scripts (CSP). Pin new npm dependencies; no CDN loads.
- **GitOps / D34:** no `kubectl apply`; nothing deploys until m1-07's tag; no alerting.
- **Memory-sum rule:** no new pod or container.
- **Parallel sessions:** check peers' PRs, tags and ADR numbers before numbering the renderer ADR.

## Deliverables

- `internal/gateway/withhold.go` + item-state resolver + application on every item surface and the coach context.
- `withholdPolicy` on every route; `internal/gateway/withhold_routes_test.go`.
- curriculum `solution_facts` placement + tests.
- `web/src/components/Markdown.tsx` v2 + `Markdown.test.tsx` (XSS corpus, parity); the renderer ADR.
- `web/src/screens/Revision.tsx` v2 per AB03 (+ `revision.ts`, tests); `Problem.tsx` chip rule.
- PR with the golden diff, latency numbers, bundle delta and AB03 screenshots.

## Update status

- Set each task in [`../sprints/sprint-m1-06.md`](../sprints/sprint-m1-06.md) 🔄 → ✅; _Overall_ ✅ when all five are.
- [`../status.md`](../status.md): **Sprint board** row for m1-06; **Milestones** M1 stays 🔄; the AB03 artboard row →
  "frozen (PR #, date)" if not already set.
- **Decisions log:**
  - the renderer choice (ADR number);
  - fail-closed withholding;
  - the list-vs-workspace rule difference: lists hide while live, the workspace follows the hint stage while unsolved
    or in an open attempt, and a solved, not-live item keeps v1's chip;
  - the AB03 frames deferred (F5/F7 → m2-04, F8 → p-02) and any F6 date gap;
  - MI-5b status at merge time.

## Done when (acceptance)

- [ ] Route-enumeration test green; no route leaks pattern/concepts/facts during a live attempt or due touch. A solved
      (incl. solved-blind), not-due item keeps its pattern on lists and in the workspace.
- [ ] Pattern chip only from the hint stage while unsolved or in an open attempt; due-touch and live-arena views
      withhold hint/solution/pattern.
- [ ] Coach context for live or never-solved problems carries no pattern/concepts/facts.
- [ ] Markdown XSS corpus blocked; tables/code/`asset:` images/https links render; v1 parity; ADR recorded.
- [ ] Revision matches AB03 F1–F4, F6, F9 (F5/F7/F8 deferred by name).
- [ ] `solution_facts` only with the solution stage.
- [ ] Every v1 e2e green; golden = v1 except withheld fields on live items; CI green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) (`feat(gateway): …`, `feat(web): …`, `docs(adr): …`) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. The renderer ADR merges with the PR (approved by the launch, D40).
3. **Release action — merge only:** nothing deploys (`main` is build-only), so there is no live verification in this session. It ships in **`v1.7.0`**, which [m1-07](../sprints/sprint-m1-07.md) cuts. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
