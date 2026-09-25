# Sprint m4-06 — M4 UI: AI suggestion/dispute (AB16★), pointer notes (AB17), allowance + consents (AB18)

> **Milestone:** M4 — platform AI (the one M4 UI sprint) · **Track:** product · **Order:** 65
> **Prereqs:** [ds-m4-01](sprint-ds-m4-01.md) (AB16–AB18 frozen) · [m4-03](sprint-m4-03.md) (analyzer, `evaluation_analyzed`, pointer notes) · [m4-04](sprint-m4-04.md) (provisional, dispute, re-grade, claims, `self_grade_pending`, "grades waiting") · [m4-05](sprint-m4-05.md) (`/api/me/ai-allowance`, `PATCH /api/me/consents`) · builds on [m3-11](sprint-m3-11.md) (Workspace concluded card), [m3-12](sprint-m3-12.md) (results dock, AB11 badges), [m3-13](sprint-m3-13.md) (Week chips, Mistakes source chip, Progress provenance), [m2-04](sprint-m2-04.md) (Touch result, Today slot), [m2-03](sprint-m2-03.md) (`ProgressViews`, public provenance), [m1-06](sprint-m1-06.md) (`withhold()`, Revision v2), [m1-10](sprint-m1-10.md) ("Your AI coach (your key)")
> **Unblocks:** [m4-07](sprint-m4-07.md) (canary test, caps, the `v1.16.0` tag, the cohort flip) · via `v1.16.0`: [l-05](sprint-l-05.md) (reuses this sprint's consent component and strings at the acceptance step)
> **Release action:** **merge only** (ships dark in `v1.16.0`, tagged by [m4-07](sprint-m4-07.md)). Every AI surface is present only while platform AI is in use for the account: the presence-gated routes 404 outside the T-3 cohort, and with `LLM_PLATFORM_ENABLED=false` the allowance reads `off` (`platform_disabled`), which the UI treats as absent (task 4). So nothing is visible until m4-07's infra PR, and then only to the owner and testers. No infra PR, no flag, no new pod, no migration.
> **Calendar:** December 2026 (agent work; no owner time)
> **Execute with:** [`../prompts/prompt-m4-06.md`](../prompts/prompt-m4-06.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | AB16 frames live in v2.0: analyzer suggestion on a deterministic grade (F2), honor-probe claim (F3–F4), accepted (F13), Today "grades waiting" (F14), provenance legend (F15), server countdown | X | ⬜ |
| 2 | AB16 provisional contract: suggestion card (F1), bounded edit (F5), dispute (F6), re-grade running/compare (F7–F8), override confirm (F9), flagged (F10), AI unavailable (F11), manual entry (F12), narrow (F16) | X | ⬜ |
| 3 | AB17 pointer notes + "correct, with improvements" + optional revisit (F1–F10) | X | ⬜ |
| 4 | AB18 allowance meter, paused states, Settings AI consents, the two names (F1–F9, F11) | X | ⬜ |
| 5 | AI-off badges (AB18 F10, AB11 "AI off") + `ai` provenance in P7 views (Progress, public profile, Mistakes) | X | ⬜ |
| 6 | Tests: vitest per frame; the `-tags e2e` harness reaching every frame (judge on m4-02's `httptest` fake provider) | X | ⬜ |
| 7 | Visual check against the frozen boards + docs + status rows | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, artboard rows AB16–AB18, public-dashboard row P7; M4 stays 🔄).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB16–AB18 frozen**: the [ds-m4-01](sprint-ds-m4-01.md) PR is merged — the merge is the freeze (D40), so its drafted copy and its "Decisions to confirm" defaults are final as merged (a later change is a follow-up design PR) — and status.md's artboard rows read "frozen (PR #, date)" (recorded by the ds-m4-01 session, or by [m4-01](sprint-m4-01.md)'s close-out). This sprint builds from `design-system/screens/v2/AB16-ai-suggestion-dispute.html`, `design-system/screens/v2/AB17-pointer-notes.html` and `design-system/screens/v2/AB18-ai-allowance-consents.html`.
- [ ] **[m4-03](sprint-m4-03.md) merged**: the analyzer runs on course passes and failures and on materially different touch passes (D26); review stores `category_source=analyzer`, the suggested category and concepts, and marks revision items "correct, with improvements"; judge stores pointer notes per (account, item). The gateway serves `GET /api/problems/{id}/pointer-notes`, `GET /api/attempts/{id}/analysis`, `improvements` + `optional[]` on `GET /api/revision/due`, and `POST`/`DELETE /api/problems/{id}/optional-revisit`.
- [ ] **[m4-04](sprint-m4-04.md) merged**: practice `provisional` with `accept_deadline_at`, `server_now`, `ceiling`, the candidate and the flags (`low_confidence`, `review_flag`) on `GET /api/submissions/{id}` and the touch view. The routes: `POST /api/attempts/{id}/accept {grade?, category?}` (an edit or override is an accept with `grade ≠ candidate_grade`: `422 above_ceiling`, `409 grade_not_editable`), `POST /api/evaluations/{id}/dispute` (→ re-grade; `409 dispute_used`, `503 ai_unavailable` / `budget_exhausted{resets_at}`), `POST /api/attempts/{id}/claim` (`409 claim_used`), and `409 not_provisional`. Also `self_grade_pending` with its reason (`budget_exhausted` / `ai_unavailable`), `grades_waiting` on `GET /api/dashboard`, and `PRACTICE_SETTLE_WINDOW` (honoured only with `DEV_AUTH`).
- [ ] **[m4-05](sprint-m4-05.md) merged**: `GET /api/me/ai-allowance` (`used_pct`, `resets_at`, `state`, `global_paused`, plus the daily-vs-monthly and per-account-disable fields), `PATCH /api/me/consents` with the two AI kinds and the behavioral opt-in, and a consent read with version and timestamp.
- [ ] **The e2e harness runs platform AI offline**: `internal/e2e` (`-tags e2e`; real Postgres and JetStream, services in-process, as m4-03/m4-04/m4-05 extended it) with judge on m4-02's `httptest` fake provider. That is m4-01's convention: a fake provider is never a committed compose service. Also m4-04's **synthetic rubric fixture item** under `internal/judge/testdata/pack` and its seeded calibration-row fixture. If either fixture is absent, task 6 adds it as test data only.
- [ ] **Parallel sessions:** no open peer PR edits `web/src/screens/{Workspace,Touch,Settings,Dashboard,Revision,Mistakes,Progress,UserDashboard,Arena}.tsx`, `web/src/components/{ProgressViews.tsx,judge/*}` or `web/src/lib/{judge,settings,dashboard}.ts` (`gh pr list`, `git worktree list`, ListAgents), or the order is agreed.

## Goal

Build the **platform-AI surfaces** so that when [m4-07](sprint-m4-07.md) switches `LLM_PLATFORM_ENABLED` on for the cohort, every AI state the owner can reach renders the frozen board and keeps four promises:
- **AI output is a suggestion** (D14): the learner accepts, edits within the deterministic ceilings, asks for one blind re-grade, or claims an honor-key answer. Manual entry is reachable from every AI state, and an untouched suggestion auto-accepts after 24 h **unless it is flagged**;
- **pass review never changes a grade or the ladder** (D16/D26): pointer notes are collapsed, private and hidden while the item is live;
- **the allowance is a percentage, never dollars**, and consents are unticked by default and withdrawable at once ([ADR-0031 §4–§5](../../adr/0031-platform-ai-and-two-tier-keys.md#4-retention-and-privacy-owner-d24));
- **degraded states say what still works** (D34: an in-app badge, never an alert).

Sources: [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D14, pass review), [ADR-0031](../../adr/0031-platform-ai-and-two-tier-keys.md) §4–§7, [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade) (who is provisional, dispute, override, claim), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) ("Provisional and claim"), [t4 §13](../research/t4-judge-contract.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D14 overrides §3.5 on edit-before-re-grade), [t5 §6–§8](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (learner view, consents, notes privacy), [t5 §12](../research/t5-platform-ai.md#12-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D26), [PRD §5.6](../../prd/xlearn-v2-prd.md) R-AI1–R-AI6, [rollout §9](../rollout-plan.md#9-artboards-by-milestone) (M4 row), [§10](../rollout-plan.md#10-public-dashboard-tasks) P7 (AI stage).

## Scope

**In**
- web: `web/src/components/ai/` (suggestion, claim, dispute, re-grade compare, override confirm, bounded editor, manual entry, countdown, pointer notes, allowance meter, consent toggles, AI badge, provenance legend), `web/src/lib/ai.ts` (allowance, consents, notes) and `web/src/lib/grading.ts` (provisional actions), mounted in the Workspace concluded area, the results dock's Feedback tab, the Touch result, Revision rows, Week chips, Today, Settings, Mistakes, Progress and the public profile.
- **What is live in v2.0** ([ds-m4-01](sprint-ds-m4-01.md) task 2): no v2.0 item has an `ai_rubric` step, so the owner meets AB16 **F2–F4 and F10–F14** on production. **F1 and F5–F9** are the generic D14 components that m4-04's contract already serves; they are built and proven in the e2e harness on the synthetic rubric item, and nothing more (no rubric workspace).
- Outside `web/`: tests and test data (`internal/e2e/m4_ui_test.go`, synthetic fixtures), and **thin gateway passthroughs only under the gap rule** below.
- **Gap rule.** If a board needs a read or write that the **owning service already exposes** but the gateway doesn't route, add the thin, additive passthrough in `internal/gateway` (JWT-scoped, presence by config, `withhold()` where the item can be live, a row in the per-endpoint denylist test, `openapi.yaml` updated). If the owning service **lacks the capability**, don't build it here: render the frame's read-only variant, and report the gap against [m4-03](sprint-m4-03.md) / [m4-04](sprint-m4-04.md) / [m4-05](sprint-m4-05.md) in the decisions log. m4-03, m4-04 and m4-05 already route everything these boards read or write (named per task below), so the gap rule is only a fallback for drift from those plans.

**Out**
- The backend states behind the frames: caps, breaker, `ai_disabled` → [m4-02](sprint-m4-02.md); analyzer and notes → [m4-03](sprint-m4-03.md); provisional, dispute, claims → [m4-04](sprint-m4-04.md); allowance and consents API → [m4-05](sprint-m4-05.md).
- The canary log test, cap sizing, the `v1.16.0` tag and the `LLM_PLATFORM_ENABLED` flip → [m4-07](sprint-m4-07.md).
- The consents **at the acceptance step** (AB19★), the privacy-notice page (AB20) and its link from Settings → [l-05](sprint-l-05.md). This sprint exports the consent component and strings it reuses.
- The BYO coach panel (AB01) → [m1-10](sprint-m1-10.md) / [m1-07](sprint-m1-07.md); only the two names side by side here. Pointer notes in the coach's review-mode context are server-side and already built by [m4-03](sprint-m4-03.md) (`internal/gateway/coach.go`, [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2)); this sprint shows AB01's review-mode line only if the board has one (task 3, F9).
- A rubric workspace, the canvas and the SD course → v2.2+ (AB31, [m6c-03](sprint-m6c-03.md)). The mock AI proposal (`ai-byo`, AB27) → [m6a-06](sprint-m6a-06.md).

## Tasks

### 1 · AB16 frames live in v2.0 [X]

Board `design-system/screens/v2/AB16-ai-suggestion-dispute.html` frames F2, F3, F4, F13, F14, F15. The frozen board wins over any copy quoted here.

- **Countdown** — `web/src/components/ai/Countdown.tsx`, shared by every provisional frame.
  - It renders from the server's `accept_deadline_at` only, corrected by the server clock offset (m4-04's `server_now` field, falling back to the response `Date` header, measured once per fetch). The client clock is never trusted.
  - Mono digits in the `xl-timer` style. `aria-live="polite"` announces only at 1 h and 5 min left.
  - At 0 it **refetches** the attempt and never concludes anything locally. With a null deadline (flagged, F10) it renders nothing.
- **F2 · analyzer suggestion on a deterministic grade** (DSA course conclusion below Clean), in the Workspace concluded card ([m3-11](sprint-m3-11.md)) and on the Mistakes entry ([m3-13](sprint-m3-13.md)):
  - the grade row is locked (`xl-lock` "Code verdicts aren't editable");
  - the data is m4-03's `GET /api/attempts/{id}/analysis` (`status` `queued|done|skipped|unavailable`, `reason`, `category`, `confidence`, `concept_keys[]`, `summary`, `next_step`);
  - an **"xLearn AI suggests"** block shows the category chip with its confidence word (`low|medium|high`, [t5 §7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences) item 2), ≤ 3 concept chips, the one-line summary and the next step;
  - **[Use this]** sends the existing `PATCH /mistakes/{id}` with the suggested category, which locks the field (precedence learner > strong rule > analyzer ≥ medium > weak rule, [t4 §6.3](../research/t4-judge-contract.md#63-mistake-pre-fill-two-tiers-data-only)). **[Change…]** opens the 8-category picker. The note reads "Your choice always wins over a suggestion.";
  - the analyzer lands **after** conclusion, so while the analysis route has no row yet or `status=queued`, show a bounded **pending** state ("xLearn AI is reviewing…"): refetch that route every 5 s, then 15 s, and stop after 5 min. Then show the muted variant its `status`/`reason` names: `unavailable` (breaker, `no_acceptance`), `skipped(too_large)`, consent off (`no_consent`) and paused (`budget_exhausted`, AB18 F10);
  - concepts come only from the post-conclusion DTO; a withheld concept is **absent**, never greyed ([t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise)).
- **F3–F4 · honor-probe claim** in the Touch result (`web/src/screens/Touch.tsx`, [m2-04](sprint-m2-04.md), AB04 F10), and in the concluded card when a `public:*` key criterion missed on a course attempt:
  - criteria rows with their trust labels, the headline, and a claim card with the **accepted answers**. Those exist only in the post-lock DTO, so the UI never asks for them earlier;
  - **[I meant this]** (once per attempt; `409 claim_used` disables it with the board's copy) and **[Keep as not passed]**, plus the provisional banner and countdown;
  - F4 after a claim: "Passed — advances to Day 21" with the `claimed` and `honor` chips and "not included in your judge-checked %".
- **F13 · accepted / auto-accepted**: the grade with the "xLearn AI · accepted" chip (or "accepted automatically after 24 h"), and the follow-ups: a link to the mistake entry and the Day-1 date from `anchor_at`.
- **F14 · Today "grades waiting"**: wire AB05's slot ([m2-04](sprint-m2-04.md); it may be a placeholder) to m4-04's `grades_waiting{count, first_deadline_at, items[≤5]}` on `GET /api/dashboard`. Each row shows the item, the grade and a countdown from its own `accept_deadline_at`, with **[Review]** opening the attempt. m3-13's Week "Provisional" chip links to the same place.
- **F15 · provenance legend** (`ProvenanceLegend.tsx`, a popover used by the concluded cards and Progress): `judge-checked` · `ai` · `self-set` · `claimed` · `honor`, with one line each on what counts toward judge-checked %.
- **Actions** go through `web/src/lib/grading.ts` with an `Idempotency-Key` per mutation (m3-11's convention). Use the route names and error codes in `docs/architecture/openapi.yaml` as m4-04 merged them:
  - **accept** = `POST /api/attempts/{id}/accept {grade?, category?}`. **Edit and override are the same route** with `grade ≠ candidate_grade`; m4-04 has no separate override route. Map `422 above_ceiling` and `409 grade_not_editable`;
  - **dispute** = `POST /api/evaluations/{id}/dispute`; **claim** = `POST /api/attempts/{id}/claim`;
  - **grade it myself** = m3-11's bounded picker through the v1 outcome route (INV-9), plus its single **Retry grading**.

  The server decides eligibility, the ceiling and the deadline; the client only renders them.

### 2 · AB16 provisional contract (F1, F5–F12, F16) [X]

These are the generic D14 components. On production in v2.0 only F10–F12 are reachable, through the honor-probe and budget paths. The rest are reachable in the e2e harness on the synthetic rubric item (task 6). Mount them wherever an attempt is `provisional` or `self_grade_pending`: the Workspace conclusion area and the Touch result. **Never in a mock context:** m4-04 keeps the path course/touch-only.

- **F1 · suggestion card**: the chip row "Rough · xLearn AI · provisional", the countdown, and per-criterion rows (band dots plus the **verified learner quote** "You wrote: '…'"; never rubric anchors, must-cover lists or exemplars). Then the mistake suggestion, ≤ 3 concepts, reasoning ≤ 480 chars, and **[Accept]** (primary) **[Edit…]** **[Dispute…]**.
- **F5 · bounded edit**: a `ds-seg` grade picker built from the DTO's `ceiling` and allowed grades. Grades above the ceiling are disabled with the server's reason ("hint opened at 17:02"). Deterministic steps are locked; mistake category, notes (≤ 240) and concepts are editable. A **grade** change goes through F9. D14 allows editing straight from the card (t4 §3.5's "override only after re-grade" is overridden, t4 §13). If m4-04's server refuses it before a re-grade, the UI mirrors the server.
  - **Saving is two calls, in order** (m4-04 task 2). First `POST /api/attempts/{id}/accept {grade?, category?}`: the category rides the accept, and a `grade ≠ candidate_grade` is the override (`422 above_ceiling`, `409 grade_not_editable`). Then, once the conclusion has opened the mistake entry (refetch until it exists), save the notes and concepts with review's `PATCH /mistakes/{id}`. **No prose goes in the accept body**, so none ever rides an event.
- **F6 · dispute modal** (`ds-modal--md`, Esc closes, **[Cancel]** has default focus):
  - tick the misjudged criteria with m4-04's reason codes; deterministic steps are shown locked;
  - optional text ≤ 500 chars with a counter and the helper "Shown to the course author for calibration; the re-grade doesn't read it.";
  - errors: `409 dispute_used` → the "Dispute used" state; `409 not_provisional` → refetch; `503 ai_unavailable` → F11/F12.
- **F7–F8 · re-grade running and compare**: poll with `poll_after_ms`; the countdown is hidden with "Your 24 h window restarts when the re-grade lands". Then two columns (first grade / re-grade, per-criterion bands with changes marked), a new countdown, **[Accept re-grade]** **[Set my own grade…]** and the "Dispute used" chip.
- **F9 · override confirm** (`ds-modal--sm`, default focus on **[Keep the AI grade]**): it names the **self-set** label, says the grade **won't count toward judge-checked %**, and gives the allowed range from the ceiling.
- **F10 · flagged, no auto-accept**: `low_confidence` and `review_flag` variants with the board's copy. No countdown and no "auto-accepts" text. The `review_flag` copy never names the heuristic.
- **F11 · AI unavailable on a final**: **[Wait for xLearn AI]** ("resets <date>" from the DTO's `resets_at` for `budget_exhausted`, "usually back within the hour" for `ai_unavailable`) and **[Grade it myself]**; the touch variant says "falls back to self-scoring".
- **F12 · manual entry** (`self_grade_pending`): m3-11's bounded picker gets the AB16 header "Graded by you — xLearn AI is off for this one" and the `self-set` label. It keeps m3-11's single **Retry grading**.
- **F16 · < 1024 px / 390 px**: F1's criteria become an accordion; F6 and F9 become full-screen sheets; F8 becomes two tabs.

### 3 · AB17 pointer notes + "correct, with improvements" [X]

Board `design-system/screens/v2/AB17-pointer-notes.html` F1–F10; [ADR-0031 §6](../../adr/0031-platform-ai-and-two-tier-keys.md#6-scope-of-ai-review-of-passing-solutions-owner-d26-refines-d16), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (C3, learner-private).

- **Data** — [m4-03](sprint-m4-03.md)'s routes; its `withhold()` coverage and public denylist already include them (add a test case here only if one is missing):
  - `GET /api/problems/{id}/pointer-notes` → `{state: available|none|withheld, notes[≤3]{note, lines}, revisit_suggested, revisit_reason, updated_at}` (F1–F3, F5, F7, F9);
  - `GET /api/attempts/{id}/analysis` → `status`, `reason`, `improvements`, `baseline_reviewed_at` (F4, F6, F8);
  - `improvements` per item and the `optional[]` list on `GET /api/revision/due` (F3, F4);
  - `POST` / `DELETE /api/problems/{id}/optional-revisit` (F2–F3, in `internal/gateway/review.go`).

  If a route is missing from `openapi.yaml` after all, apply the gap rule.
- **`PointerNotes.tsx`**:
  - **F1** is collapsed by default ("xLearn AI · 2 improvement notes", `aria-expanded`).
  - **F2** expanded shows ≤ 3 notes rendered as **plain text**: no Markdown, no HTML, never `dangerouslySetInnerHTML`, because the prose is model output. A line-range tag "L12–18" highlights those lines in a read-only excerpt of the learner's **own** submission (≤ 8 lines, `xl-code` `<pre>`, no editor chunk). Below: "Optional revisit suggested: …" with **[Add optional revisit]** **[Not now]**, and the footnote "Notes never change your grade or your revision schedule."
- **F3 · optional revisit added**: **[Add optional revisit]** sends `POST /api/problems/{id}/optional-revisit` (idempotent; the server sets `due_on`). The revisit is listed apart from the five touch dots, from `optional[]` on `GET /api/revision/due`, with the `optional` chip and **Remove** (`DELETE`).
- **F4 · "correct, with improvements"**: the chip on Revision queue rows (`web/src/screens/Revision.tsx`, AB03), on the Week row, and on the touch result after conclusion ("From this review's solution").
- **F5 · live item**: during a live touch or an open counted attempt, the slot shows `xl-lock` "Notes are hidden during this review — they come back when you finish." It can't expand, and the component **doesn't fetch** notes while the attempt/touch state says live. The BFF answers `{state: "withheld"}` anyway.
- **F6 · near-identical resubmission** (`reason=fingerprint_same` + `baseline_reviewed_at`: "Same approach as your Dec 3 solution — its notes still apply."), **F7 · nothing to improve** (a done review, notes `state=none`: "Reviewed by xLearn AI · no improvement notes"), and **F8 · unavailable** in three variants: pass-review consent off (`no_consent`) → "Improvement notes are off · Settings"; shed or paused (`budget_exhausted`) → "paused for now"; AI off → only the provenance chip.
- **F9 · placements**: the Workspace after conclusion, the arena study view (`web/src/screens/Arena.tsx`, [m3-12](sprint-m3-12.md); after the course conclusion, never while live) and the coach panel's review-mode line only if AB01 shows one. **Never on the public profile** (test).
- **F10**: < 1024 px accordion; the excerpt scrolls inside its own box, so the page never scrolls sideways.

### 4 · AB18 allowance + consents [X]

Board `design-system/screens/v2/AB18-ai-allowance-consents.html` F1–F9, F11; [ADR-0031 §4, §5, §7](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25), [t5 §6](../research/t5-platform-ai.md#6-budgets-metering-and-abuse-controls) (learner view), [t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency) (consent wording).

- **`web/src/lib/ai.ts`**:
  - `useAiAllowance()`: `GET /api/me/ai-allowance`, `staleTime` 60 s, refetch on focus. **Absent** on a 404 (outside the cohort: m4-05's gateway presence rule) **and** on a 200 with `state: off` and `reason ∈ {platform_disabled, not_cohort}` (`LLM_PLATFORM_ENABLED=false`; m4-05 task 4). Any other `off` (`no_consent`, `ai_disabled`) **renders**: right after m4-07's flip the owner's state is `off`/`no_consent`, and Settings must still show the consents.
  - `useAiConsents()` / `useSetAiConsent()`: `PATCH /api/me/consents`, optimistic with rollback.
  - The TS types admit only the percentage fields (`used_pct`, `resets_at`, `state`, `global_paused`, the pause scope, the per-account disable). If the DTO carries money, the type drops it and the decisions log records the drift against m4-05. ADR-0031 §5 wins.
  - Consent kinds are **m4-05's constants** (from `openapi.yaml`), never strings invented here.
- **Settings → "xLearn AI (included)"** (`web/src/screens/Settings.tsx`; the section renders unless the allowance is absent):
  - F1 ok: `ds-meter` "This month: 38% used · resets Jan 1";
  - F2 low: `--ds-warn`, from the server `state`, never a client threshold;
  - F3 paused for the month: `--ds-err`;
  - F4 daily limit reached: the UTC reset shown in the account timezone;
  - F5 paused for everyone (`global_paused`): the AB11-style badge + detail;
  - F6 off for this account.
  - **No dollar figure in any state.**
- **F7 · consents**: three real `<button role="switch" aria-checked>` in the `ds-toggle` style:
  - **unticked** for an account that never chose;
  - the second ("…and also reviews my passing solutions") is disabled until the first is on;
  - the behavioral opt-in is separate.
  - Helper lines and the footer (Anthropic, outside India, ≤ 30 days, no training) use the board's text. Show "Last changed <date> · consent v<n>" from the consent read.
  - The **privacy-notice link** is added by [l-05](sprint-l-05.md) with the AB20 page; until then the footer has no link.
- **F8 · switch off**: immediate, no confirm dialog, toast. Turning the first off also turns the second off, mirroring the server. It is effective for new work at once; nothing already sent is recalled.
- **F9 · the two names** side by side: "xLearn AI (included)" and "Your AI coach (your key)", with one line each. Rename m1-10's coach-key section if it isn't already named that way.
- **Export** `AiConsentToggles` (a `variant: 'settings' | 'acceptance'` prop) so [l-05](sprint-l-05.md) mounts it in the acceptance step with **identical strings** (AB19).
- **F11**: stacked full width below 1024 px.

### 5 · AI-off badges + `ai` provenance (P7 AI stage) [X]

- **Badges** (AB18 F10; AB11's "AI off", typed by [m3-12](sprint-m3-12.md) in `web/src/components/judge/JudgeBadges.tsx`):
  - results dock "xLearn AI off · manual", Today "xLearn AI paused", and the mistake entry "pick a category" (no AI chip);
  - they're driven by the allowance (`state`, `global_paused`, per-account disable) and the consents. **Absent entirely** when the allowance is absent (task 4: a 404, or `off` with `platform_disabled`/`not_cohort`; presence by config, [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service));
  - informational, never modal, theme contrast tokens.
- **Results dock Feedback tab** ([m3-12](sprint-m3-12.md)): after a counted conclusion, render the xLearn AI review fields from m4-03's `GET /api/attempts/{id}/analysis` (summary, findings with line ranges, suggested category, concepts, next step) and the pointer notes (task 3), labelled "xLearn AI (included) · suggestion". Show only fields present in the DTO; pending, unavailable and off states per AB16/AB18, from its `status`/`reason`.
- **Progress** (`web/src/components/ProgressViews.tsx`, `web/src/screens/Progress.tsx`):
  - the provenance chips gain **`ai`** (the field has existed since [m2-03](sprint-m2-03.md): `provenance{self,auto,ai,override,honor}`);
  - **judge-checked % is unchanged**: `auto ∧ checked` only; `ai`, `self-set`, `claimed` and `honor` are excluded, and the F15 legend says so.
- **Public profile** (`web/src/screens/UserDashboard.tsx`): the `ai` chip in the aggregate provenance row only. No notes, suggestions or prose; the P10 allowlist test stays green, plus a case proving pointer notes never reach `/public/stats`.
- **Mistakes**: m3-13's "AI" source chip (`category_source=analyzer`) and the analyzer concept source go live.
- *Inferred:* in v2.0 content no **grade** is AI-decided (DSA and go-concurrency are code- or key-graded), so the `ai` grade bucket stays 0 on production. The chip must still render correctly from fixtures.

### 6 · Tests [X]

- **Vitest** (`web/src/**/*.test.tsx`), with fixtures in `web/src/test/fixtures/ai/*.json` recorded from the e2e below:
  - **countdown**: a client clock skewed ±10 min shows the same remaining time; at 0 it refetches and changes no local state; announcements only at 1 h and 5 min; no countdown with a null deadline;
  - **flagged**: `low_confidence` and `review_flag` show no countdown and no "auto-accepts" text; the `review_flag` copy contains neither "injection" nor the heuristic;
  - **edit and override**: no grade above `ceiling` is selectable; the override copy names self-set and the judge-checked exclusion;
  - **dispute**: the 500-char counter; the 409/503 mapping; disabled after use;
  - **claim**: the answers render only when the DTO carries them; once per attempt;
  - **notes**: collapsed by default; locked and **not fetched** while live; a note containing `<img src=x>` or `[x](http://…)` renders as literal text; the line-range highlight;
  - **allowance**: every state; **no `$` anywhere in the rendered DOM** for any fixture; a 404 or `off`/`platform_disabled` → the section and badges are absent; `off`/`no_consent` → the section renders with the consents;
  - **consents**: unticked default; the second disabled until the first; first off → second off; `role=switch` / `aria-checked`; the PATCH body uses m4-05's kinds;
  - **provenance**: the `ai` chip; judge-checked % excludes `ai`; the public page shows no prose.
- **e2e** — `internal/e2e/m4_ui_test.go` (`-tags e2e`), on the in-process harness m4-03/m4-04/m4-05 extended (real Postgres and JetStream, services and the gateway in-process). Judge runs with `LLM_PLATFORM_ENABLED=true` in the harness config only, on m4-02's `httptest` fake provider scripted per case. That is m4-01's convention: **no fake provider is committed as a compose service**. CI's e2e lane runs it.
  - **Setup:**
    - owner, tester and `learner` accounts, seeded as the harness seeds accounts (roles per m1-04); the cohort consents are set through `PATCH /api/me/consents`;
    - **passed `llm_calibration` rows** for the synthetic configurations, seeded through `judge admin calibration record --from -` (m3-14's dispatcher called in-process) or m4-04's SQL fixture. One is `analyze`: without it the analyzer makes no call (`no_acceptance`), so AB16 F2 and AB17 are unreachable. The others are m4-04's `score` and `score_regrade` configurations: without them F1 and F5–F9 answer 503. Keep the `score_regrade` row's config tuple for the independent re-grade assertion;
    - `PRACTICE_SETTLE_WINDOW` set to a few seconds, with `DEV_AUTH`, in the harness's practice config (m4-04 honours it only with `DEV_AUTH`).
  - Every frame is reached through the real BFF handlers, and each response is saved as the vitest golden. The `judge admin` verbs below run through m3-14's dispatcher, in-process, against the harness DB.

  | Frames | How the e2e reaches them |
  |---|---|
  | AB16 F2 | a below-Clean DSA fixture conclusion (WA, then a late pass); the fake analyzer returns a category, concepts and a next step; `GET /api/attempts/{id}/analysis` goes `queued` → `done` |
  | AB16 F3–F4 | a fixture touch with a misspelled pattern answer → provisional → claim, and a second claim → `409 claim_used` |
  | AB16 F1, F5–F9 | the **synthetic rubric item** (m4-04's; add it under `internal/judge/testdata/pack` if missing) with scripted bands: accept; a bounded edit (accept with `grade ≠ candidate_grade`, then `PATCH /mistakes/{id}` for notes); an edit above the ceiling → `422 above_ceiling`; dispute → re-grade → compare → override |
  | AB16 F10 | a scripted band spread ≥ 2 (`low_confidence`) and the injection-heuristic phrase (`review_flag`): `accept_deadline_at` is null |
  | AB16 F11 "resets <date>" + F12 | `judge admin llm-limit <owner> --day-usd 0` → the rubric final is `self_grade_pending(budget_exhausted)` with `resets_at`; then **[Grade it myself]**; then `llm-limit <owner> --clear` |
  | AB16 F11 "usually back within the hour" | `judge admin breaker set --scope llm` → the next rubric final is `inconclusive(ai_unavailable)` (m4-02); then `breaker clear --scope llm` |
  | AB16 F13 | accept; and an untouched provisional attempt auto-accepts once the shortened `PRACTICE_SETTLE_WINDOW` passes (`resolution=auto_accepted`) |
  | AB16 F14 | two provisional attempts → `grades_waiting.count = 2` on `GET /api/dashboard` |
  | AB17 F1–F8 | a clean pass → 2 notes + a revisit (then `POST` / `DELETE /api/problems/{id}/optional-revisit`); a near-identical touch pass → F6 (`fingerprint_same`); an optimal pass → F7; a live touch → `{state: "withheld"}` (F5); the pass-review consent off → F8 (`no_consent`) |
  | AB18 F1–F6 | seeded `llm_call` rows plus `judge admin llm-limit <user> --month-usd … / --day-usd …` to reach 38% / 86% / 100% and the daily cap; `judge admin breaker set --scope llm` → `global_paused`; `judge admin ai-disable <tester>` → F6 |
  | Presence | the `learner` account: the presence-gated routes (`/api/me/ai-allowance`, `/api/attempts/{id}/analysis`, `/api/problems/{id}/pointer-notes`, the optional-revisit routes, dispute) 404 and no AI surface renders; accept and claim refuse a non-provisional attempt; `GET /api/me/consents` answers 200 (m4-05: not cohort-gated) but the UI hides its section because the allowance is absent; the public profile is unchanged. Then judge with `LLM_PLATFORM_ENABLED=false`: the owner's allowance is `off`/`platform_disabled` (the golden that vitest renders as absent) |

  A manual `docker compose up --build` check is optional: point judge's `LLM_ANTHROPIC_BASE_URL` at a fake through a `-f docker-compose.yml -f <override>` kept in the session scratchpad, never committed (m4-01's convention).
- **Gates:** `npm --prefix web run {typecheck,lint,test,build}` (then `git checkout -- web/dist/.gitkeep`); `go test -race ./...`; `XLEARN_TEST_DATABASE_URL=… go test -tags e2e -race ./internal/e2e/...` (CI's e2e lane); `sqlc diff` clean (no migration); the OpenAPI drift test if a passthrough was added. No new heavy dependency: notes use `<pre>`, not the editor chunk.

### 7 · Visual check + docs + status [X]

- Compare every frame with the frozen boards at **1440 px and 390 px** using a temporary `web/vite.mock.config.ts` that stubs `/xlearn/api/*` from the recorded fixtures (**delete it before committing**). Attach the screenshots to the PR.
- `docs/architecture/api.md`: which screen reads which AI route, and any passthrough added under the gap rule.
- [`../status.md`](../status.md): artboard rows AB16–AB18 → "implemented (m4-06, PR #)"; public-dashboard row **P7 → AI stage done**; the sprint board row; decisions log.

## Acceptance criteria

- [ ] Every AB16 (F1–F16), AB17 (F1–F10) and AB18 (F1–F11) frame is implemented and reached in the `-tags e2e` harness (the e2e table passes, in CI's e2e lane). Any frame left read-only by the gap rule is listed with its owning sprint.
- [ ] **No auto-submit when flagged**: `low_confidence` / `review_flag` attempts show no countdown and no auto-accept copy.
- [ ] Countdowns render only from the server deadline (the clock-skew test); reaching 0 refetches and never concludes client-side.
- [ ] Edits never exceed the ceiling; an override is labelled self-set and excluded from judge-checked %; the dispute is once per attempt and its text is labelled as never read by the re-grade.
- [ ] The allowance shows a **percentage only** (no `$` in any state); consents are unticked by default, the second is gated on the first, and switching off is immediate.
- [ ] Pointer notes are collapsed by default, hidden and unfetched while the item is live, rendered as plain text, and **never** on the public profile.
- [ ] `ai` provenance shows in Progress and the public profile; judge-checked % is unchanged.
- [ ] Presence by config: outside the cohort, or with `LLM_PLATFORM_ENABLED=false`, the UI is exactly v1.15's (no AI surface, no badge).
- [ ] Screens match AB16–AB18 at 1440 px and 390 px; web checks, `go test -race`, the e2e and CI are green.

## Release

**Merge only — ships dark in `v1.16.0`**, tagged by [m4-07](sprint-m4-07.md). Do **not** tag. No infra PR, no flag, no new pod, no migration: gateway passthroughs added under the gap rule are additive routes behind presence by config.
- **Dark until the flip:** `v1.16.0` ships with `LLM_PLATFORM_ENABLED=false` (set by [mi-12](sprint-mi-12.md)). For the owner the allowance reads `off` (`platform_disabled`) and the presence-gated routes produce nothing new; for anyone outside the cohort they 404. Either way the UI is v1.15's. `GET /api/me/consents` answers for every account, but its section stays hidden. m4-07's infra PR turns it on, for the T-3 cohort only.
- **Rollback:** R-a (`LLM_PLATFORM_ENABLED=false` → the surfaces vanish) or R-c ([ADR-0034 §4.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#41-mechanisms-fastest-first)).

## Definition of Done

CI green · PR squash-merged to `main` (no tag) · acceptance criteria met · screenshots against AB16–AB18 in the PR · statuses updated (this file + [`../status.md`](../status.md): board row, artboard rows AB16–AB18, P7; M4 stays 🔄) · decisions log: the countdown clock-offset source, plain-text notes, the shared consent component, any gap-rule passthrough or reported gap · local `main` synced.

## Risks / watch-outs

- **Countdown drift.** A client timer that "auto-accepts" locally is wrong twice: clocks skew, and the server decides. Render from `accept_deadline_at` with the server offset, and refetch at 0.
- **Building rubric UI nobody can reach.** F1 and F5–F9 are the D14 components m4-04's contract already serves. Build them generically and prove them in the e2e harness, and build no rubric workspace ([ds-m4-01](sprint-ds-m4-01.md) risk). Review F2–F4 and F10–F14 first: those are what the owner meets.
- **Dollar leakage** (ADR-0031 §5). The types drop money fields; the DOM test greps for `$`.
- **Leaking withheld data.** Notes and concepts on live items, and accepted answers before the lock. The component never asks for what the DTO withholds, and the BFF's `withhold()` is the backstop.
- **Model output as markup.** Pointer notes and summaries are untrusted text: plain text only, never HTML.
- **Consent-string drift from AB19.** One exported component and m4-05's constants, so l-05 can't diverge.
- **Scope creep into the backend.** The gap rule allows a passthrough, never domain logic, tables or migrations. A missing capability is reported, not built.
- **Async analyzer.** The suggestion arrives after conclusion. The pending state is bounded (5 min), then quietly becomes "no suggestion".
- **Mocks.** None of these surfaces render in a mock context; the D14 path is course/touch only (m4-04).
