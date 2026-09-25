# Sprint m1-06 — withhold() on every surface + Markdown renderer + revision v2 (AB03)

> **Milestone:** M1 — spine (**M1b** slice, rides `v1.7.0`) · **Track:** product (gateway router serialized [m1-03](sprint-m1-03.md) → [m1-04](sprint-m1-04.md) → [m1-05](sprint-m1-05.md) → **m1-06**) · **Order:** 24
> **Prereqs:** [m1-05](sprint-m1-05.md) (gateway router) · [ds-m1-01](sprint-ds-m1-01.md) (AB03 frozen)
> **Unblocks:** [m1-07](sprint-m1-07.md) (its mode gate reuses this sprint's live-item predicate; `v1.7.0` tag)
> **Release action:** **merge only** (ships in `v1.7.0`, tagged by [m1-07](sprint-m1-07.md)) · no infra PR · no flag
> **Calendar:** week 3 (2026-10-12 → 10-16; agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m1-06.md`](../prompts/prompt-m1-06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Gateway `withhold()` + per-request item states, applied on every item surface (incl. arena GET and coach context) | X | ⬜ |
| 2 | Route-enumeration test (static policy on every route + behavioural sentinel sweep) | X | ⬜ |
| 3 | Sanitizing Markdown renderer (GFM tables, fenced code + highlighting, `asset:` images, https links) + XSS corpus + ADR | X | ⬜ |
| 4 | Revision v2 (AB03 F1–F4, F6, F9): format badges, withheld pattern, re-solve/result, empty state | X | ⬜ |
| 5 | `solution_facts` only with the solution stage (curriculum + BFF) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + milestone).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] AB03 frozen: [ds-m1-01](sprint-ds-m1-01.md) merged (the merge is the freeze, D40). This sprint builds frames **F1–F4, F6
      and F9**; F5, F7 and F8 are deferred by name (task 4).
- [ ] [m1-05](sprint-m1-05.md) merged (gateway router serialized m1-03 → m1-04 → m1-05 → m1-06; `httpx.ReadBody` in place)
- [ ] [m1-03](sprint-m1-03.md) merged (course-scoped routes and DSA aliases; `useCourse()` → `{slug, view, status}` with the
      learner-safe manifest view) and `v1.6.0` live (frozen item schema with `concepts[]` / `solution_facts`, [m1-01](sprint-m1-01.md))
- [ ] *Soft, recommended — not blocking:* MI-5b live ([mi-04](sprint-mi-04.md)) before the new Markdown renderer ships
      ([ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b)); m1-07
      carries the same soft gate for the tag
- [ ] Parallel sessions: no open peer PR edits `internal/gateway/**`, `web/src/components/Markdown.tsx` or
      `web/src/screens/Revision.tsx` (`gh pr list`, `git worktree list`, ListAgents). The one exception is
      [m1-10](sprint-m1-10.md)'s additive gateway change: the new `coach_models.go`, one coach route row, its
      OpenAPI path and new test files.

## Goal

Stop answer leaks **structurally**: one pure gateway **`withhold()`** decides, for an item that is *live* (an open
counted attempt, a due or live touch), which answer-bearing fields — pattern, concepts, solution facts, hint and
solution stages — every surface may show, and a **route-enumeration test** makes a new unguarded route fail CI
([ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1, [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes),
[t0 §7](../research/t0-extensibility-frame.md#7-screens--routing)). Ship the **sanitizing Markdown renderer** v2 content
needs, and the **revision v2** screen (AB03) whose pattern chip no longer gives the recall answer away
([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed)). Withholding is a nudge (the repo is
public), not secrecy.

## Scope

**In**
- `internal/gateway/withhold.go` (pure rules) + a per-request item-state resolver; applied on list, bulk, week, Today,
  the revision due queue, mistakes enrichment, the problem workspace, the arena GET (`?practice=1`), concept item chips,
  and coach context composition — including m1-03's course-scoped aliases.
- The problem workspace's pattern chip only from the hint stage while unsolved or during an open attempt (fixes the
  `Problem.dc.html:104` leak in `Problem.tsx:336`). A solved, not-live item keeps v1's chip.
- Route-enumeration test.
- `web/src/components/Markdown.tsx` v2: CommonMark + GFM tables, fenced code with highlighting, images only via `asset:`
  refs, https links, no raw HTML, SVG only through `<img>`; XSS corpus; ADR for the renderer choice.
- `web/src/screens/Revision.tsx` per AB03 **F1–F4, F6, F9**: format badges (manifest-driven; "Recall" only as a
  fixture until M2a), withheld pattern, re-solve and scored result, empty state, narrow layout.
- `solution_facts` served only inside the solution stage.

**Out**
- Arena Submit / History / Copy-to-editor / concept chips of the judge surfaces → [m3-09](sprint-m3-09.md) (M3).
- Coach **modes** (`locked` / `attempt` / `review` / `general`), `GET /attempts/open`, `coach-prompt@2`, D27 → [m1-07](sprint-m1-07.md);
  this sprint only strips withheld fields from the coach context and exports the predicate m1-07 builds on.
- Touch formats, touch attempts (`LiveTouch`), probes and the Touch UI → M2a ([m2-01](sprint-m2-01.md), [m2-04](sprint-m2-04.md)).
- AB03 **F5** (Recall row), **F7** (overdue banner, R-SR5 per course) → [m2-04](sprint-m2-04.md). AB03 **F8**
  (all-courses toggle) → [p-02](sprint-p-02.md). Details in task 4.
- Storing `problem.spec` / probes → [m2-01](sprint-m2-01.md); the curriculum media route for `asset:` images lands with
  the first content that needs it (the renderer only takes a resolver).
- The public profile shape → done in [m1-05](sprint-m1-05.md) (P10).

## Tasks

### 1 · `withhold()` [X]

Sources: [t1 §3.1](../research/t1-content-data-model.md#31-visibility-tiers) (the tier "public, withheld while an item is
live"), [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes) (stage-gating row: the surface list),
[t0 §7](../research/t0-extensibility-frame.md#7-screens--routing) (arena and honesty gating),
[t4 §6.4 / §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (concepts withheld; chip from the hint
stage), [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (attempt mode: "open counted attempt, due touch,
or never solved"), [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §1.
- **`internal/gateway/withhold.go`** — pure, table-tested:
  - `itemState{OpenAttempt, DueTouch, LiveTouch, Solved bool; Unlocked map[string]bool}` — `LiveTouch` is reserved for
    M2a touch attempts (always false today).
  - `live(s) = s.OpenAttempt || s.DueTouch || s.LiveTouch`.
  - `withhold(s itemState, sf surface) withheld{Pattern, Concepts, SolutionFacts, HintStage, SolutionStage bool}` with
    this rule table:

    | Surface | Withheld when |
    |---|---|
    | lists — `GET /api/paths/{slug}/problems`, week view, Today (`/api/dashboard`), revision due queue, mistakes enrichment (the curriculum join **and** review's own `mistake_entry.pattern`), concept pages' item chips | `live(s)` → pattern, concepts, facts |
    | problem workspace (course view) | `DueTouch‖LiveTouch` → pattern, concepts, facts, hint + solution stages. `OpenAttempt ‖ !Solved` → stage-gated: pattern + concepts only once `hint` is unlocked, facts + solution only once `solution` is unlocked (t4 §8: in the attempt shell the chip "appears only from the hint stage"). `Solved ∧ !live` → pattern + concepts shown **as in v1**, hint/solution/facts by unlocked stage as in v1. A problem solved blind keeps `unlockedStages=[attempt]` (`internal/practice/store/store_integration_test.go:186`) and must not lose its chip |
    | arena GET (`?practice=1`) | `live(s)` → attempt stage only, no pattern / concepts / facts (today it returns every stage, `bff.go:526-540`); not live → the full study view as today |
    | coach context (`gateway/coach.go:287-313`) | `live(s) ‖ !s.Solved` → pattern, concepts, facts stripped before the context goes to coach (fixes `coach.go:313`) |
    | progress / public aggregates | exempt — aggregates carry no per-item answer field (reason recorded, task 2) |
- **Item states per request:** `g.itemStates(ctx, account, ids)` — one parallel fan-out per request with
  `aggCallTimeout`: practice `GET /state?ids=` (status, `unlockedStages`, `timer` → `OpenAttempt`, `firstSolvedAt` →
  `Solved`) and review `GET /revisions/due` (→ `DueTouch`); reuse the state a handler already fetched (week and problem
  handlers do). **Fail closed:** if either upstream fails, treat every requested item as live for withholding (hide
  rather than leak) and log WARN once per request.
- **Apply** with one helper that strips JSON fields (`pattern`, `concepts`, `solution_facts`, stage-tagged sections)
  from the composed objects, reusing the unlocked-stage filter in `aggregate.go:151-184`; apply it to the course-scoped
  routes and their DSA aliases alike. Export the predicate for [m1-07](sprint-m1-07.md)'s mode gate (same package).
- **Caching:** the Week/Today `aggCache` entries are per account; attempt start/outcome writes already invalidate the
  account, and a touch that becomes due via review's sweep is picked up within the cache TTL (15 s) — acceptable for a
  nudge; note it in the code.
- **Web:** `Problem.tsx:336` and `Revision.tsx:235` render the pattern chip **only when the payload carries `pattern`**
  (the server decides); no client-side unlock logic.

### 2 · Route-enumeration test [X]

- `apiRoute` (`internal/gateway/bff.go`) gains `Withhold withholdPolicy{Mode: applied|exempt, Reason string}`; every row
  sets it (including [m1-10](sprint-m1-10.md)'s `/api/coach/models` if merged: exempt, "model catalog; no item data").
- `internal/gateway/withhold_routes_test.go`:
  - **Static:** every route in `apiRoutes()` declares a policy; every `exempt` row appears in a reviewed table inside
    the test with its reason (auth, account, coach key, mock, progress aggregate, public profile, healthz, JWKS, writes
    that return state only…); **a new unlisted route fails CI**.
  - **Behavioural sweep:** fakes where problem `1` has an open attempt, problem `2` has a due touch, and problem `3`
    is solved **blind** and not due (`unlockedStages=[attempt]`). Curriculum returns sentinels (`PATTERN-SENTINEL-1`,
    `CONCEPT-SENTINEL-1`, `FACTS-SENTINEL-1`, …) for each. Fetch every `applied` GET route, with path params from the
    fixtures, incl. `?practice=1` and the course-scoped variants. Assert that no sentinel of items 1–2 appears in any
    body, and that item 3's pattern **is** present on lists **and on its problem-workspace GET** (not
    over-withholding). Coach: a chat for problem 1 (live) and for a never-solved problem → the request the fake coach
    receives carries no pattern/concept/facts sentinel.
- Keep it next to the OpenAPI drift test (same route table, one source of truth).

### 3 · Markdown renderer [X]

Sources: [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation) (the Markdown profile m1-09's content CI enforces
on the author side), [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes) (public media and Markdown row),
[t0 §10](../research/t0-extensibility-frame.md#10-what-t0-constrains-downstream), [ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) (why an XSS here matters).
- Rewrite `web/src/components/Markdown.tsx` (today: a dependency-free subset — headings, lists, inline, the `> [!warning]`
  callout; used by `Concept.tsx` and `Problem.tsx:460`) to the v2 profile, **reader side**:
  - CommonMark + GFM **tables**, **fenced code** with syntax highlighting (go, cpp, python, sql, json, bash, text);
  - **images only via `asset:` refs**, resolved through a `resolveAsset(ref) → url | null` prop (unresolvable → alt text;
    `http(s):`, `data:`, relative and any other `src` dropped); rendered only as `<img>` (SVG never inline);
  - **links only `https:`**, with `rel="noopener noreferrer nofollow"` and `target="_blank"`; other schemes render as text;
  - **no raw HTML** (skipped, never passed through), never `dangerouslySetInnerHTML`;
  - keep the `> [!warning]` / "Watch out" callout, the `InlineMD` export and the `.cn` article typography; highlighting
    uses class names styled with `theme.css` tokens (no inline `<style>`, CSP-safe after [m1-04](sprint-m1-04.md)).
- **Recommended implementation** (decide and record as an ADR, next free number after the parallel-sessions check):
  `react-markdown` + `remark-gfm` (`skipHtml`, `urlTransform`, an element allowlist via `allowedElements`) and
  `lowlight` with only the languages above, converted to React nodes (no HTML strings); lazy-load the highlighter chunk;
  report the bundle delta in the PR. Keeping a hand-written parser is acceptable only if it passes the same corpus and
  covers the profile.
- **Tests** (`web/src/components/Markdown.test.tsx`): an **XSS corpus** — `<script>`, `<img onerror>`, `<svg onload>`,
  `<iframe>`/`<object>`/`<embed>`, HTML comments, `javascript:` / `data:` / `vbscript:` links incl. entity-, case- and
  whitespace-obfuscated forms, autolinks, reference-style links, image refs with `javascript:` and `asset:../../x`
  traversal, fence info strings carrying markup, table cells with HTML, and an emphasis/nesting bomb (must render in
  bounded time). Assert the DOM has no `script|iframe|object|embed|svg|style` element, no `on*` attribute, every `href`
  starts with `https://`, every `src` came from `resolveAsset`. Plus **parity**: every concept body and section in
  `curriculum/` renders the same block structure as v1 (snapshot).

### 4 · Revision v2 (AB03) [X]

Sources: AB03 (`design-system/screens/v2/AB03-revision-v2.html`, frozen by [ds-m1-01](sprint-ds-m1-01.md)), [rollout §9](../rollout-plan.md#9-artboards-by-milestone),
[t4 §6.6](../research/t4-judge-contract.md) (touch formats per band).
- `web/src/screens/Revision.tsx`:
  - **Format badges** from the course manifest band for the card's `touchLevel` (via m1-03's `useCourse().view`; if
    the learner-safe manifest view doesn't carry `revision.bands` yet, add only the display fields — format, timer,
    `est_minutes`, `mock_mode`, never criteria keys — to curriculum's manifest view): DSA today
    renders **"Re-solve · 20:00"** (timer from the band), with the mock-conditions marker at L4–5; the **"Recall · ~5 min"**
    variant renders only when a band says so (fixture test; real in M2a, [m2-04](sprint-m2-04.md)).
  - **Pattern withheld:** a due card shows no pattern chip (the server strips it); the pattern appears only where AB03
    shows it (the re-solve's hint stage and the result after scoring — the score response is composed after the touch
    concludes, so `withhold()` no longer applies).
  - **Frames in scope:**
    - **F1**, the due queue: badges, five-touch dots, and an `xl-lock` "Pattern hidden while due" in place of the chip.
    - **F2**, the active re-solve: timer from the band, v1's three criteria tiles, pattern hidden.
    - **F3 / F4**, scored pass / fail: pattern revealed; F4's "A mistake entry is open" link.
    - **F6**, empty / caught up. Render its "Next review" date only if the due payload already carries one;
      `GET /api/revision/due` has no upcoming-touch field today. Otherwise omit the date and log the gap for m2-04's
      agenda.
    - **F9**, narrow.

    Copy is verbatim from the board, and links use m1-03's course-scoped routes.
  - **Deferred by name:**
    - **F5** (a Recall row in the all-courses view) → [m2-04](sprint-m2-04.md), which makes the badges real data, and
      [p-02](sprint-p-02.md), which adds go-concurrency's Recall rows. This sprint has only the fixture-only Recall
      badge test.
    - **F7** (the overdue banner) → [m2-04](sprint-m2-04.md). It defines "overdue" and builds the per-course R-SR5
      block, and showing the banner before the block exists would be false copy.
    - **F8** (the all-courses toggle) → [p-02](sprint-p-02.md). It needs a cross-course due queue, and "revision
      all-courses" is in p-02's scope.
- `web/src/lib/revision.ts` types (pattern optional); `Revision.test.tsx` (badges per band, no chip on due items, result
  shows the pattern, empty state). Screenshots at 1440 px and 390 px beside AB03 in the PR.

### 5 · `solution_facts` gating [X]

Sources: [t1 §4 curriculum](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual) (API: "`solution_facts` is returned only with the solution stage").
- curriculum `GET /problems/{id}` (`internal/curriculum/handlers.go`): when an item carries `solution_facts` (frozen item
  schema, [m1-01](sprint-m1-01.md)), serve it **only inside the solution-stage payload** (a `stage: "solution"` block) —
  never top-level, never in `GET /problems?ids=` or `/paths/{slug}/problems`. If m1-09's loader does not persist the
  field yet, land the API rule and a fixture test anyway, so [m2-01](sprint-m2-01.md)'s `spec` column plugs in unchanged.
- The gateway's stage filter + `withhold()` drop it until the solution stage unlocks (and while the item is live); tests
  in `internal/curriculum/handlers_test.go` and `internal/gateway/problem_test.go`.

## Acceptance criteria

- [ ] **Route-enumeration test green:** every route declares a withhold policy (exempt ones with a reviewed reason); **no
      route leaks pattern, concepts or facts during a live attempt or a due touch**. A solved (incl. solved-blind), not-due
      item keeps its pattern on lists **and** in the workspace.
- [ ] Problem workspace: while unsolved or during an open attempt, the pattern chip appears only from the hint stage. A
      solved, not-live item shows it as in v1. A due-touch item shows no hint, solution or pattern, and the arena view
      of a live item shows the attempt stage only.
- [ ] Coach context for a live or never-solved problem carries no pattern, concepts or facts.
- [ ] **Markdown XSS corpus blocked**; tables, fenced code with highlighting, `asset:` images and https links render; v1
      content renders with parity; the renderer ADR is recorded.
- [ ] **Revision matches AB03 F1–F4, F6 and F9** (badges, withheld pattern, re-solve, scored result, empty state,
      narrow), with 1440/390 px screenshots in the PR. F5, F7 and F8 are deferred to m2-04 / p-02; the Recall badge
      exists only in a fixture test.
- [ ] `solution_facts` only with the solution stage (curriculum + gateway tests).
- [ ] Every v1 e2e green; Week/Today golden = v1 except the withheld pattern on live items; CI green (incl. `sqlc diff`).

## Release

**Merge only — ships in `v1.7.0`**, cut by [m1-07](sprint-m1-07.md) (M1b). Do **not** tag. No schema change, no infra PR,
no flag, no new pod. The renderer ships to production only with the tag; MI-5b before then is recommended (soft gate).

## Definition of Done

CI green · PR squash-merged to `main` (no tag) · acceptance criteria met · AB03 F1–F4, F6, F9 matched (F5/F7/F8 deferred) · renderer ADR merged with the
PR · statuses updated (this file + [`../status.md`](../status.md): board row; M1 stays 🔄; AB03 row "frozen (PR #, date)"
if not already set) · notable calls in the decisions log.

## Risks / watch-outs

- **`withhold()` on the dashboard aggregate changes a v1 view.** The golden must match v1 except the leak fix. Diff
  the Today/Week JSON before and after, and explain every difference in the PR. Capture it from the
  `internal/gateway/bff_test.go` harness fixtures or from compose with dev login. The e2e harness can't provide it:
  it drives the practice, review and assessment HTTP surfaces and never runs the gateway composites.
- **Extra fan-out** — a list route now needs review's due queue as well as practice state; fetch in parallel, reuse what
  the handler already has, keep `aggCallTimeout`; measure Week/Today latency on compose.
- **Fail-closed hides patterns during an upstream outage** — intended (a nudge, never a leak); log it once per request.
- **Over-withholding.** The sweep asserts that a solved-blind, not-due item still shows its pattern on lists and in
  the workspace. The Problems arena must stay a usable study view for items that aren't live.
- **Cache staleness** — a touch that turns due via review's sweep is withheld only after the 15 s cache TTL.
- **Renderer supply chain and size** — pin versions, allow only the listed languages, lazy-load highlighting; CSP from
  m1-04 must stay intact (no inline scripts or `<style>`).
- **m1-07 builds on this** (the exported predicate and `coach.go` edits) — keep the coach change minimal and named.
