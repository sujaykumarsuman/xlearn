# Prompt — Sprint m4-06 · M4 UI: AI suggestion/dispute (AB16★), pointer notes (AB17), allowance + consents (AB18)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-06.md`](../sprints/sprint-m4-06.md)   ·   **Milestone:** M4 (the M4 UI sprint)   ·   **Prereqs:** [ds-m4-01](../sprints/sprint-ds-m4-01.md) (boards frozen), [m4-03](../sprints/sprint-m4-03.md), [m4-04](../sprints/sprint-m4-04.md), [m4-05](../sprints/sprint-m4-05.md) merged

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions and land-and-sync.
- The plan: [`../sprints/sprint-m4-06.md`](../sprints/sprint-m4-06.md). The frame lists, the gap rule and the e2e reachability table are there, and they're authoritative.
- **The frozen boards** (authoritative for layout and copy; preview-only, never imported):
  - `design-system/screens/v2/AB16-ai-suggestion-dispute.html` (F1–F16);
  - `design-system/screens/v2/AB17-pointer-notes.html` (F1–F10);
  - `design-system/screens/v2/AB18-ai-allowance-consents.html` (F1–F11);
  - neighbours for shared vocabulary: AB04 (touch result), AB05 (Today), AB08 (results dock), AB11 (badges), AB12 (provenance), AB19 (acceptance consents, if frozen).
  
  Tokens and components: [`design-system/README.md`](../../../design-system/README.md), `design-system/theme.css`.
- The design brief behind the boards: [`../sprints/sprint-ds-m4-01.md`](../sprints/sprint-ds-m4-01.md). It covers which frames are live in v2.0, the behaviour notes and the "Decisions to confirm" list, whose drafted defaults froze with the merge (D40; a later change is a follow-up design PR).
- **Decisions:**
  - [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D14, pass review);
  - [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md): §4 consents, §5 percent-never-dollars and degrade order, §6 D26, §7 UI names;
  - [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (presence by config, T-3 cohort);
  - [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (the acceptance twins of the consents).
- **Research:**
  - [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) (provisional, dispute, override, claim), [§6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only) (precedence), [§6.4](../research/t4-judge-contract.md#64-concepts-to-revise) (withheld concepts), [§8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) ("Provisional and claim"), [§13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D14 overrides §3.5's override-after-re-grade);
  - [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (learner view), [§7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) (confidence enum, `pass_review`), [§8](../research/t5-platform-ai.md#8-privacy-and-residency) (consents, notes are C3), [§12](../research/t5-platform-ai.md#12-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D26).
- [PRD §5.6](../../prd/xlearn-v2-prd.md) R-AI1–R-AI6; [rollout §9](../rollout-plan.md#9-artboards-by-milestone) (M4 row), [§10](../rollout-plan.md#10-public-dashboard-tasks) (P7, P10).
- **Code:**
  - screens: `web/src/screens/{Workspace,Touch,Settings,Dashboard,Revision,Week,Mistakes,Progress,UserDashboard,Arena}.tsx`;
  - components: `web/src/components/{ProgressViews.tsx,judge/ResultsDock.tsx,judge/JudgeBadges.tsx}`;
  - lib: `web/src/lib/{judge,api,settings,dashboard,mistakes,progress}.ts`;
  - gateway: `internal/gateway/` (BFF routes, `withhold()`, `public_test.go`);
  - API docs: `docs/architecture/{api.md,openapi.yaml}` (m4-03/m4-04/m4-05's routes and error codes);
  - test data and harness: `internal/judge/testdata/pack` (m4-04's synthetic rubric item), `internal/e2e/` (the in-process `-tags e2e` harness; judge on m4-02's `httptest` fake provider, never a committed compose service), m4-04's seeded calibration-row fixture.

## Context

M4 (platform AI) ships in **`v1.16.0`**:
- [m4-01](../sprints/sprint-m4-01.md): `platform/llm` and WIF;
- [m4-02](../sprints/sprint-m4-02.md): Scorer, ledger, caps, breaker;
- [m4-03](../sprints/sprint-m4-03.md): analyzer, pointer notes, `evaluation_analyzed`;
- [m4-04](../sprints/sprint-m4-04.md): provisional, dispute, re-grade, claims;
- [m4-05](../sprints/sprint-m4-05.md): allowance, consents;
- **this sprint**: the UI;
- [m4-07](../sprints/sprint-m4-07.md): the canary test, cap sizing, the tag, and the `LLM_PLATFORM_ENABLED=true` infra PR for the owner/tester cohort.

This sprint is **web code** plus tests. It adds thin gateway passthroughs only under the plan's gap rule.

What the owner actually meets in v2.0 is AB16 F2–F4 and F10–F14 (analyzer suggestions, honor-probe claims, budget fallbacks, grades waiting), AB17 and AB18. No v2.0 item has an `ai_rubric` step, so F1 and F5–F9 are the generic D14 components, proven in the e2e harness on a synthetic rubric item.

v2 is owner-only use (D35), and nothing alerts (D34): degraded states are in-app badges.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB16–AB18 frozen: the ds-m4-01 PR is merged, and the merge is the freeze (`git log origin/main -- design-system/screens/v2/AB1[678]-*`); status.md's artboard rows read "frozen (PR #, date)" (recorded by the ds-m4-01 session or m4-01's close-out).
- [ ] [m4-03](../sprints/sprint-m4-03.md) merged: the analyzer (passes and failures, D26 touch passes), review's `category_source=analyzer` + concepts + "correct, with improvements", pointer notes per (account, item), and its gateway routes: `GET /api/problems/{id}/pointer-notes`, `GET /api/attempts/{id}/analysis`, `improvements` + `optional[]` on `GET /api/revision/due`, `POST`/`DELETE /api/problems/{id}/optional-revisit`.
- [ ] [m4-04](../sprints/sprint-m4-04.md) merged: `provisional` with `accept_deadline_at` / `server_now` / `ceiling` / flags; `POST /api/attempts/{id}/accept {grade?, category?}` (edit and override = accept with `grade ≠ candidate_grade`), `POST /api/evaluations/{id}/dispute`, `POST /api/attempts/{id}/claim` and their error codes; `self_grade_pending` reasons; `grades_waiting` on `GET /api/dashboard`; `PRACTICE_SETTLE_WINDOW` (with `DEV_AUTH`).
- [ ] [m4-05](../sprints/sprint-m4-05.md) merged: `GET /api/me/ai-allowance`, `PATCH /api/me/consents` (two AI kinds + behavioral), and a consent read with version and timestamp.
- [ ] The e2e harness runs platform AI offline: `internal/e2e` (`-tags e2e`, in-process) with judge on m4-02's `httptest` fake provider. No fake provider is a committed compose service (m4-01's convention). If m4-04's synthetic rubric fixture item or its calibration-row fixture is missing, you'll add it as test data in step 7.
- [ ] No open peer PR edits the screens, components or libs listed in the plan's gate (`gh pr list`, `git worktree list`, ListAgents).

## Do this (in order)

1. **[X] Branch** `feat/m4-ai-ui` from an up-to-date `main`. Read `openapi.yaml` for the exact M4 route names, DTO fields and error codes; use them verbatim.
2. **[X] Shared pieces.**
   - `web/src/lib/grading.ts`: accept (`POST /api/attempts/{id}/accept {grade?, category?}`), dispute, claim, wait / grade-myself. **Edit and override are accept with `grade ≠ candidate_grade`** (m4-04 has no override route): map `422 above_ceiling` and `409 grade_not_editable`. Each call sends an `Idempotency-Key`; responses and errors are typed.
   - `web/src/lib/ai.ts`:
     - `useAiAllowance()` (staleTime 60 s, refetch on focus). **Absent** on a 404 or on a 200 with `state: off` and `reason ∈ {platform_disabled, not_cohort}`; any other `off` (`no_consent`, `ai_disabled`) renders, so the owner can reach the consents after the flip;
     - `useAiConsents()` / `useSetAiConsent()` (optimistic with rollback; m4-05's kind constants);
     - the pointer-notes hook (`GET /api/problems/{id}/pointer-notes`), disabled while the item is live, and the analysis hook (`GET /api/attempts/{id}/analysis`).
   - `web/src/components/ai/Countdown.tsx`:
     - renders from `accept_deadline_at` plus the server clock offset (m4-04's `server_now`, else the `Date` header);
     - `aria-live` only at 1 h and 5 min;
     - refetches at 0 and never concludes locally;
     - renders nothing for a null deadline.
3. **[X] AB16 frames live in v2.0** (plan task 1):
   - **F2** in the Workspace concluded card and on Mistakes, from `GET /api/attempts/{id}/analysis`: code verdict locked; "xLearn AI suggests" (category + confidence word, ≤ 3 concepts, summary, next step); **[Use this]** → `PATCH /mistakes/{id}`; **[Change…]** → the 8-category picker; a bounded pending state while there's no row or `status=queued` (refetch that route 5 s → 15 s, stop at 5 min), then the muted variant its `status`/`reason` names.
   - **F3–F4** claim card in `Touch.tsx` and the concluded card: accepted answers only from the post-lock DTO; **[I meant this]** once (`409 claim_used`).
   - **F13** accepted and auto-accepted, with the mistake link and the Day-1 date from `anchor_at`.
   - **F14** Today "grades waiting" from `grades_waiting` on `GET /api/dashboard`: rows with per-row countdowns and **[Review]**; link m3-13's Week "Provisional" chip to it.
   - **F15** `ProvenanceLegend`.
4. **[X] AB16 provisional contract** (plan task 2):
   - **F1** card: verified quotes only, never anchors or exemplars.
   - **F5** bounded edit from the DTO's `ceiling`, with the server's reasons; a grade change goes through F9. D14: edit straight from the card, unless the server refuses (mirror it). **Save in two calls, in order:** the accept with `{grade?, category?}` first; then, once the conclusion has opened the mistake entry, the notes and concepts via `PATCH /mistakes/{id}`. No prose ever goes in the accept body.
   - **F6** dispute modal: reason codes, locked deterministic steps, ≤ 500 chars + the "re-grade doesn't read it" helper, default focus **[Cancel]**, the 409/503 mapping.
   - **F7–F8** re-grade running and compare.
   - **F9** override confirm naming **self-set** and the **judge-checked exclusion** (default focus on keep).
   - **F10** flagged: no countdown and no auto copy; never name the heuristic.
   - **F11** wait / grade myself; **F12** manual entry on m3-11's picker.
   - **F16** narrow layouts.
   - Never mount any of these in a mock context.
5. **[X] AB17** (plan task 3):
   - Data from m4-03's routes (already under `withhold()` and the public denylist): `GET /api/problems/{id}/pointer-notes` (`{state, notes[≤3]{note, lines}, revisit_suggested, revisit_reason, updated_at}`), `GET /api/attempts/{id}/analysis` (`status`, `reason`, `improvements`, `baseline_reviewed_at`), `improvements` + `optional[]` on `GET /api/revision/due`, and `POST`/`DELETE /api/problems/{id}/optional-revisit`.
   - `PointerNotes.tsx`: collapsed by default; ≤ 3 notes as **plain text** (never HTML or Markdown); line-range highlight in a `<pre class="xl-code">` excerpt of the learner's own submission; the optional revisit (**[Add]** → `POST`, **[Not now]**, F3's **Remove** → `DELETE`); the "never changes your grade or schedule" footnote.
   - **F5** lock while live (no fetch; the BFF answers `{state: "withheld"}` anyway). **F6** near-identical (`fingerprint_same`), **F7** nothing to improve, **F8** unavailable variants (`no_consent`, `budget_exhausted`, AI off).
   - The "correct, with improvements" chip on `Revision.tsx` rows, the Week row and the touch result.
   - Placements: the Workspace after conclusion and the arena study view. **Never** on the public profile.
   - **Gap rule, fallback only:** if one of those routes turns out missing from `openapi.yaml`, add a thin additive passthrough when the owning service exposes the capability (plus the `withhold()` route list, the denylist and `openapi.yaml`). Otherwise render the read-only variant and report the gap.
6. **[X] AB18 + badges + provenance** (plan tasks 4–5):
   - Settings "xLearn AI (included)" with the meter states F1–F6 (percent only, `state`-driven, the daily reset in the account timezone).
   - **F7** three `role="switch"` consents: unticked by default, the second gated on the first, the behavioral separate, the board's footer text, no notice link (l-05 adds it). **F8** instant off plus a toast. **F9** the two names side by side.
   - Export `AiConsentToggles` for l-05.
   - Badges: AB11 "AI off" live, plus AB18 F10's chips; absent whenever the allowance is absent.
   - The results-dock Feedback tab renders only the AI fields `GET /api/attempts/{id}/analysis` carries.
   - The `ai` chip in `ProgressViews` and `UserDashboard` (aggregate only); judge-checked % unchanged; Mistakes' "AI" source chip live.
7. **[X] Tests.**
   - **Vitest** for every frame from JSON fixtures in `web/src/test/fixtures/ai/`:
     - the countdown under a ±10 min clock skew;
     - flagged → no countdown;
     - no grade above the ceiling;
     - dispute limits;
     - claims once;
     - notes as literal text (`<img>` / Markdown link) and not fetched while live;
     - **no `$` in the DOM** for any allowance fixture;
     - consent defaults and gating;
     - `ai` provenance and judge-checked %.
   - **e2e.** Write `internal/e2e/m4_ui_test.go` (`-tags e2e`) on the in-process harness, with `LLM_PLATFORM_ENABLED=true` in the harness config and judge on m4-02's `httptest` fake provider (never a committed compose service). Reach every frame per the plan's table:
     - **setup first:** seed passed `llm_calibration` rows for `analyze` and m4-04's `score` + `score_regrade` configurations (via `judge admin calibration record --from -` in-process, or m4-04's SQL fixture). Without them the analyzer makes no call (`no_acceptance`) and the rubric item answers 503. Also set `PRACTICE_SETTLE_WINDOW` (with `DEV_AUTH`) to a few seconds for F13's auto-accept;
     - the synthetic rubric item (m4-04's; add it under `internal/judge/testdata/pack` if missing: public, synthetic, never from `../xlearn-evalpack`);
     - F11 twice: `budget_exhausted` via `judge admin llm-limit <owner> --day-usd 0` ("resets <date>"), and `ai_unavailable` via `judge admin breaker set --scope llm` ("usually back within the hour");
     - `judge admin llm-limit` / `breaker set --scope llm` / `ai-disable` for AB18;
     - the `learner` account for presence by config: the presence-gated routes (allowance, analysis, pointer notes, optional revisit, dispute) 404 and nothing renders; `GET /api/me/consents` answers 200 (not cohort-gated) with its section hidden; the public profile is unchanged.
   - Save each response as the vitest golden.
8. **[X] Verify.**
   - `npm --prefix web run typecheck && npm --prefix web run lint && npm --prefix web run test && npm --prefix web run build`, then `git checkout -- web/dist/.gitkeep`.
   - `go test -race ./...`, `sqlc diff` (untouched), the OpenAPI drift test, and `XLEARN_TEST_DATABASE_URL=… go test -tags e2e -race ./internal/e2e/...` (CI's e2e lane). A manual `docker compose up --build` look is optional; any fake-provider override stays in your scratchpad, never committed.
9. **[X] Visual check + docs.**
   - Use a temporary `web/vite.mock.config.ts` (never committed) to compare every frame with AB16–AB18 at 1440 px and 390 px; attach the screenshots to the PR.
   - Update `docs/architecture/api.md` (screens → AI routes, any passthrough).
10. **[X] Ship:** see **Ship** below (commit e.g. `feat(web): AI suggestion/dispute, pointer notes, allowance + consents (AB16–AB18)`). **No tag.**

## Constraints

- **Web code plus tests.** Gateway only for thin, additive passthroughs under the gap rule: JWT-scoped, presence by config, `withhold()` on live items, denylist + OpenAPI updated. Never domain logic, tables or migrations. **Service boundaries** ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)): practice decides, judge produces evidence, and review owns mistakes and revision items. The SPA renders DTOs and never infers withheld data.
- **D14:** every AI output is a suggestion, and manual entry is reachable from every AI state. **No auto-accept when flagged.** Countdowns come from the server deadline only.
- **ADR-0031 §5:** the allowance is a **percentage, never dollars**, on every surface.
- **ADR-0031 §4:** consents are unticked by default and withdrawable at once, using m4-05's constants. The strings are identical to AB19, via one exported component.
- **D26:** notes and the optional revisit never touch grade, pass/fail or ladder. Notes are learner-private (C3): never public, never in events.
- **Presence by config ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)):** a 404, or `off` with `platform_disabled`/`not_cohort`, means absent. Render v1.15's UI with no badge and no error state.
- **`theme.css` verbatim** (no Tailwind); dark theme; difficulty tokens Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. Boards are never imported.
- **Model output is untrusted text:** plain text rendering only, never `dangerouslySetInnerHTML`.
- **goose + sqlc:** no migration; `sqlc diff` must pass untouched. No outbox, inbox, stream or subject change, so no ACL PR, and consumers-before-producers is untouched.
- **GitOps / D34:** nothing to deploy here; no alerting, opscheck, healthchecks.io, Flux Alert or push channel anywhere. The memory sum is untouched (no pod).
- **Content secrecy:** fixtures are synthetic and public; never copy from `../xlearn-evalpack`.
- **Parallel sessions:** check peers' PRs and worktrees before rebasing onto a moved `main`. Check peers' ADR numbers before claiming one (none expected).

## Deliverables

- `web/src/components/ai/*` (Countdown, AnalyzerSuggestion, SuggestionCard, GradeEditor, DisputeModal, RegradeCompare, OverrideConfirm, ClaimCard, ManualEntry, PointerNotes, AllowanceMeter, AiConsentToggles, AiBadge, ProvenanceLegend) + tests; `web/src/lib/{ai,grading}.ts`.
- Updated `Workspace`, `Touch`, `Settings`, `Dashboard`, `Revision`, `Week`, `Mistakes`, `Progress`, `UserDashboard`, `Arena`, `ResultsDock`, `JudgeBadges`, `ProgressViews`.
- `web/src/test/fixtures/ai/*.json` (recorded); `internal/e2e/m4_ui_test.go`; the synthetic rubric fixture if it was missing.
- Any gap-rule passthrough (+ denylist, `withhold()` list, OpenAPI); updated `docs/architecture/api.md`.

## Update status

- [`../sprints/sprint-m4-06.md`](../sprints/sprint-m4-06.md): each task 🔄 → ✅ (⛔ with the reason for a gap reported under the gap rule); _Overall_ ✅.
- [`../status.md`](../status.md):
  - Sprint board row m4-06 ✅;
  - artboard rows AB16–AB18 → "implemented (m4-06, PR #)";
  - public-dashboard row **P7** → AI stage done;
  - M4 stays 🔄; no tag, floor or flag change.
- **Decisions log:** the countdown clock-offset source; plain-text notes; the exported consent component for l-05; each gap-rule passthrough or reported gap (with its owning sprint). No ADR expected; if one becomes necessary, check peers' ADR numbers first.

## Done when (acceptance)

- [ ] Every AB16 (F1–F16), AB17 (F1–F10) and AB18 (F1–F11) frame is implemented and reached in the `-tags e2e` harness; any gap-rule frame is listed with its owner.
- [ ] No auto-submit when flagged; countdowns come from the server deadline only (skew test green).
- [ ] Edits never exceed the ceiling; the override is labelled self-set and excluded from judge-checked %; one dispute per attempt.
- [ ] The allowance is percent-only (no `$`); consents are unticked by default, the second gated on the first, and off is immediate.
- [ ] Notes are collapsed by default, hidden and unfetched while live, plain text, and never public.
- [ ] `ai` provenance shows in Progress and the public profile; judge-checked % is unchanged.
- [ ] Outside the cohort or with LLM off, the UI is v1.15's; boards matched at 1440/390 px; all checks and CI green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m4-ai-ui`, with the 1440/390 px screenshots in the PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in `v1.16.0`):** Nothing deploys; it ships in `v1.16.0` (cut by [m4-07](../sprints/sprint-m4-07.md)). Don't tag. No infra PR.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
