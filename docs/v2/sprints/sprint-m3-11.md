# Sprint m3-11 — Workspace-Code UI (AB07★) + CodeMirror lazy load

> **Milestone:** M3 — judge plus code grader (**M3-2** slice, rides `v1.14.0`; **the first M3 UI sprint**)   ·   **Track:** product   ·   **Order:** 53
> **Prereqs:** [m3-09](sprint-m3-09.md) (judge BFF) · [ds-m3-01](sprint-ds-m3-01.md) + [ds-m3-02](sprint-ds-m3-02.md) (AB07–AB12 frozen) · [mi-10](sprint-mi-10.md) (runner dark on prod, MI-12)
> **Unblocks:** [m3-12](sprint-m3-12.md) (results dock, Problems, Arena, badges on this shell and client) · later [m5-01](sprint-m5-01.md) (the AB07 evaluator-only variant) and [m6a-04](sprint-m6a-04.md) (CodeMirror interview mode)
> **Release action:** **merge only** (ships in `v1.14.0`, tagged by [m3-13](sprint-m3-13.md)) · no infra PR · no new pod
> **Artboards:** **AB07★** A1 Workspace-Code (`design-system/screens/v2/AB07-workspace-code.html`); AB04 F6 (judged touch re-solve, drawn by [ds-m2-01](sprint-ds-m2-01.md) as an "M3 preview — final in AB07/AB08", but AB07 has no touch frame, so it's built as drawn with AB08's dock lines); a few strings no frozen board draws are drafted here and land as drafted (task 2, "Copy not on a frozen board"; D40: the owner may revise them later with a content PR)
> **Calendar:** mid–late November (agent work; owner time only for the M3 checklist items already booked)
> **Execute with:** [`../prompts/prompt-m3-11.md`](../prompts/prompt-m3-11.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Screen selection: judged → `Workspace.tsx`, self-path variant, v1 `Problem.tsx` when judge is absent | X | ⬜ |
| 2 | `Workspace.tsx`: every AB07 frame (cover 45:00, Run/Submit, language picker, timer ring, outlook, hint at 15, give-up = Miss, concluded, resume, `self_grade_pending`, self-path, < 1024 px tabs) | X | ⬜ |
| 3 | CodeMirror 6 as a lazy chunk + `vite:preloadError` → reload (loop-guarded) + a real 404 for missing assets | X | ⬜ |
| 4 | Judge client `web/src/lib/judge.ts` (TanStack Query; submit/run/close/poll/drafts; 25 s timeout; idempotency keys; typed errors) + drafts autosave | X | ⬜ |
| 5 | Dock slot + a minimal `DockStatus` (the lines AB07's frames need); M3-12 mounts the full `ResultsDock` there | X | ⬜ |
| 6 | Judged touch re-solve (AB04 F6) + Today's "Unfinished · Resume" row | X | ⬜ |
| 7 | Tests: Vitest per frame, client, preloadError; compose e2e on the fixture pack; bundle check; AB07 screenshots | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

**M3 hard entry checklist** — copied verbatim from [rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist). A red item blocks this sprint.

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

*Key (not part of the verbatim list):* rollout step ids `MI-NN` are **not** sprint ids `mi-NN`. MI-4 = [mi-14](sprint-mi-14.md) · MI-5, MI-5a = [mi-03](sprint-mi-03.md) · MI-7 (N3) = [mi-06](sprint-mi-06.md) + the ≥ 24 h re-check in [l-01](sprint-l-01.md) · MI-8 = [mi-02](sprint-mi-02.md) · MI-9 = [mi-07](sprint-mi-07.md) + the evalpack ImagePolicy in [m3-07](sprint-m3-07.md) · MI-10 = [spk-01](sprint-spk-01.md) + [spk-02](sprint-spk-02.md) · MI-11 = [mi-09](sprint-mi-09.md) + the 2026-10-24 host window · MI-11a = [mi-08](sprint-mi-08.md) · MI-12 = [mi-10](sprint-mi-10.md) · MI-13 = [m3-07](sprint-m3-07.md) · T25/T26 = [m3-01](sprint-m3-01.md) · 14 packs = the owner's pack-stamping event (prepared by [m3-02](sprint-m3-02.md)) · `account.role` = [m1-02](sprint-m1-02.md) · AB07–AB12 = [ds-m3-01](sprint-ds-m3-01.md) and [ds-m3-02](sprint-ds-m3-02.md) merged (the merge is the freeze, D40).

*Key:* the MI-8 NATS item is read at the live stage — `host-verify --cluster --nats-stage=n3` (no `legacy` connection) until N4 lands, `--nats-stage=n4` (`auth_required: true`) once [mi-11](sprint-mi-11.md) has run. N4 is MI-15 hygiene (rollout: unblocks "hygiene"), not an M3 gate ([ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34) NATS row).

*Key:* the weekly-image date is read in hPanel by the owner **before launch** (the prompt's "Before you launch (owner)" list, D40); the session records the date from that attestation and never waits for it mid-run.

**Sprint gates**

- [ ] [m3-09](sprint-m3-09.md) merged: the judge BFF routes, the `judge` block on `GET /api/problems/{id}` (incl. `stageParams`, `languages`, `runAvailable`), the allowlisted DTOs (the composed view incl. `retryUsed` and `pendingReason`), the close route accepting Retry grading from `self_grade_pending`, typed 413/429, `quota`, `/api/judge/status`.
- [ ] [ds-m3-01](sprint-ds-m3-01.md) and [ds-m3-02](sprint-ds-m3-02.md) merged (the merge is the freeze): the AB07–AB12 boards are on `main`: `design-system/screens/v2/AB07-workspace-code.html` … `AB12-week-mistakes-progress.html`.
- [ ] **Parallel sessions:** no open peer PR edits `web/src/router.tsx`, `web/src/screens/Problem.tsx`, `web/src/lib/api.ts`, `web/src/main.tsx`, `web/package.json` or `internal/gateway/gateway.go` (`gh pr list`, `git worktree list`, ListAgents). [m3-12](sprint-m3-12.md) follows this sprint on the same files; don't start it in parallel.

## Goal

Ship the **A1 Workspace-Code hero** exactly as the frozen AB07 board draws it, per D15/D16/D18
([ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer), [PRD §5.6](../../prd/xlearn-v2-prd.md)):
a 45:00 cover, the full statement on Start, Run (⌘↵) and Submit (⌘⇧↵) in Go, C++ or Python, a server timer ring that never pauses, a live grade outlook, the hint at 15:00 that caps the attempt at Assisted, give-up = Miss, **no re-implement on a judged attempt** (D16: a pass finishes it), the concluded grade with its pre-filled mistake, resume, `self_grade_pending` and the self-path variant (which keeps v1's semantics, re-implement included: AB07 F12(a), [t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution)).
The editor is **CodeMirror 6 in a lazy chunk** that loads only on judged items, and a tab left open across a deploy **reloads instead of 404ing the chunk** ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service), "Lazy chunks").

## Scope

**In**
- `web/src/screens/Workspace.tsx` + `web/src/screens/workspace/*` (Cover, Hud/TimerRing, Statement, EditorPane, ActionCard, GradeOutlook, HintConfirm, GiveUpModal, ConcludedCard, ResumeBanner, SelfPathPicker, MobileTabs); `ProblemRoute` screen selection in `web/src/router.tsx`.
- `web/src/screens/workspace/CodeEditor.tsx` (the lazy chunk), a theme from `theme.css` tokens, diagnostics at learner-file positions.
- `web/src/lib/judge.ts` (client + hooks), drafts autosave, `web/src/lib/api.ts` (a per-call `timeoutMs`), `web/src/main.tsx` (`vite:preloadError`).
- The dock **slot** in `Workspace.tsx` (the Results tab below 1024 px) and a minimal `web/src/screens/workspace/DockStatus.tsx` for AB07's frames; the full dock (`web/src/components/judge/ResultsDock.tsx`) is [m3-12](sprint-m3-12.md)'s.
- The judged re-solve step in `web/src/screens/Touch.tsx` (deferred here by [m2-04](sprint-m2-04.md)) and Today's "Unfinished · Resume" row (also deferred by M2-04).
- gateway: `serveStatic` answers 404 for a missing `assets/*` file instead of the SPA shell.

**Out**
- Every AB08 dock state beyond AB07's frames (settling detail, inconclusive, `contract_changed`, `not_evaluated_here`, the 413/quota copy, History, Feedback), Problems markers (AB09), the Arena workspace (AB10) and badges (AB11) → [m3-12](sprint-m3-12.md). The v1 `PracticeWorkspace` stays for `?practice=1` until then.
- Week/Mistakes/Progress deltas (AB12) → [m3-13](sprint-m3-13.md).
- AI suggestion, dispute, pointer notes (AB16/AB17) → M4 ([m4-06](sprint-m4-06.md)).
- The evaluator-only AB07 variant → [m5-01](sprint-m5-01.md). The quiz workspace (AB14) → [p-03](sprint-p-03.md).
- Setting `JUDGE_BASE_URL` on prod → [m3-13](sprint-m3-13.md).

## Tasks

### 1 · Screen selection [X]

Sources: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (presence by config; the kill switch = v1), [t4 §3.3](../research/t4-judge-contract.md#33-transitions) (`self_grade_pending`), [t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution) (the self path keeps v1's reveal).
- `/:course/problem/:id` renders a `ProblemRoute` that reads `GET /api/problems/{id}` (M3-09's `judge` block) and picks:

  | Condition (server data only) | Screen |
  |---|---|
  | no `judge` block (`JUDGE_BASE_URL` unset, or not in the cohort) | v1 `Problem.tsx`, **unchanged** (the R-a kill switch returns everyone here) |
  | `judge.enabled` and `judge.evaluable` | `Workspace.tsx`, judged |
  | `judge.enabled`, item not evaluable, or `gradingMode = self` | `Workspace.tsx`, **self-path variant** (AB07 F12(a): v1 semantics) |
  | attempt `state = self_grade_pending` | `Workspace.tsx`, the `self_grade_pending` frame |
  | `?practice=1` | unchanged until [m3-12](sprint-m3-12.md) |
- The client never decides grading; every branch mirrors a server field.

### 2 · `Workspace.tsx` — the AB07 frames [X]

Sources: the frozen **AB07** board, [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (one shell for every archetype; as fixed for D15/D16/D18), [PRD R-PF1…R-PF5, R-JG1–R-JG5, R-AC6](../../prd/xlearn-v2-prd.md), D27 (coach assist).
Layout per [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed): HUD (crumb, difficulty, role, timer ring, provenance chip) · left statement (+ hint/solution once unlocked) · centre editor + results dock · right action card, grade outlook, conclusion card. `theme.css` verbatim; difficulty tokens Easy `--ds-ok`, Medium `--ds-warn`, Hard `--ds-err`.

| Frame (AB07) | Behaviour | Data (server) |
|---|---|---|
| Cover | "Start attempt · 45:00"; limit, hint-at-15 and Clean rule copy; statement not rendered before Start | `judge.stageParams` ([m3-09](sprint-m3-09.md) task 1: the attempt's pinned params once one exists, else the manifest's `verdict_timer@1` defaults, so it's there before Start) |
| Solving + Run | full statement on Start; language picker (Go/C++/Python from `judge.languages`; each language's starter generated from the item's public `signature`, AB07 F13); Run ⌘↵ (`Mod-Enter`) on samples + custom input | practice timer; `POST …/runs` |
| Submit WA · perf TLE | Submit ⌘⇧↵ (`Mod-Shift-Enter`); the dock headline per AB07 ("37/40 hidden passed", "Too slow on large inputs"); CE diagnostics as editor marks (learner files only) | `POST …/submissions` → poll |
| Over-time / hinted | the outlook drops ("Clean possible · failed 1/3" → "Rough at best" after 20:00 → "Assisted at best" after the hint); the hint button enables at 15:00 with a confirm that it **caps this attempt at Assisted** (AB07 F5(b); in the judged Workspace this takes the place of v1's "I'm stuck — show a hint" / "Reveal full solution" buttons and the "Revealing the full solution now means you owe … another attempt in 3 days" warning, which stay in `Problem.tsx` and the self-path variant); a 409 `hint_locked{availableAt}` from practice ([m3-08](sprint-m3-08.md)) keeps it disabled | `ceiling`, `failedSubmits`, `hintAvailableAt`, `hintOpenedAt`, `coachUsedAt` |
| Give-up = Miss modal | the "Give up" and "Reveal solution" controls both open it (practice answers `use_give_up` to a direct solution reveal); AB07 copy (draft sent for feedback; the solution unlocks now; Miss unless a submit already in flight passes; enters revision tomorrow; opens a mistake entry) → `POST /api/attempts/{id}/close {give_up}` with the current parts | M3-09 close route |
| Solution | the solution stage renders after give-up or conclusion only (server-unlocked sections) | problem aggregate refetch |
| Concluded Clean | grade + provenance ("Judge · checked"), facts, next touch; **no re-implement** (D16) | composed practice view |
| Concluded Miss + pre-fill | grade, facts, and AB07 F9's **mistake block**: the pre-filled chip labelled *suggested* ("… suggested from your result"), **[Change]** (the category picker; the learner wins) and the root-cause / insight fields, bound to this problem's open entry from the **existing** `GET /api/mistakes?status=open` (match `problemId`; [m3-10](sprint-m3-10.md) adds `categorySource`, `categorySuggested`, `concepts`) and saved through the existing `PATCH /api/mistakes/{id}` (a learner category change locks it, m3-10). Review opens the entry asynchronously from `problem_solved`, so until it appears render the chip read-only from `mistakeHint` and refetch (≤ 30 s), then fall back to a link to Mistakes. ≤ 3 concept chips only when present (`withhold()` decides) | `mistakeHint`, `conceptsHint`; `GET`/`PATCH /api/mistakes` |
| Resume | "Resuming · 12:41 left · best grade now Rough"; the server timer kept running | open attempt state |
| `self_grade_pending` | AB07 F11: "We couldn't grade this attempt automatically." + the cause per `pendingReason` (`contract_changed` "This problem was updated while you worked." · `unreliable` "The grader couldn't get a reliable result three times." · `grading_off` "Auto-grading is paused." · `other` → the headline only); the picker capped at `ceiling` (grades above it disabled with the reason; the pick is `POST /api/problems/{id}/outcome`, practice bounds it); **[Retry grading]** once → `judge.ts` `retryGrading(attemptId, parts)` = `POST /api/attempts/{id}/close {action: "final", parts}` ([m3-09](sprint-m3-09.md) task 3), then "Retry used" | `ceiling`, `retryUsed`, `pendingReason` (m3-09's composed view) |
| Self-path variant (F12(a)) | the v2 shell with **v1 semantics** ([t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution)): attempt 15:00 → hint 10:00 → solution; an early reveal "owes another attempt in 3 days"; the **re-implement from memory** stage, which practice's self path still serves from `stages.reimplement` ([m3-08](sprint-m3-08.md)); **Run on samples** when `judge.runAvailable`; **no Submit**; the uncapped four-grade picker (Clean "no help, in time" · Rough "solved, ugly" · Assisted "needed a hint" · Miss "didn't get it"), the **only** place it appears. F12(b) (judge off for the account) is task 1's v1 `Problem.tsx` row | `gradingMode = self` or not evaluable; v1 stage/timer fields on the aggregate; `judge.runAvailable` |
| < 1024 px | Statement / Work / Results tabs | — |

- **Time's up:** the SPA never concludes. AB07 has no separate time's-up frame (only F10's "Time ran out while you were away" variant), so at 45:00 reuse the frozen AB04 F12 wording "Time's up · recording your result…" (AB08 F4's settling line for the dock), poll until practice's ticker has concluded the Miss (D15, [m3-08](sprint-m3-08.md)), then show F9's "Miss · time limit reached (45:00)".
- **Ahead of schedule:** AB09 F3(b) already routes an ahead-of-schedule row to the **arena** ("Ahead · arena only"), so a 409 `ahead_of_schedule{scheduledWeek, currentWeek}` on Start is reachable only from a direct URL: show a one-line notice with a link to the item's arena (drafted copy, below).
- **Copy not on a frozen board.** The AB07 spec ([ds-m3-01](sprint-ds-m3-01.md)) doesn't draw these. Draft each in AB07's voice, list them in a **"Copy not on a frozen board"** section of the PR body, and add one Decisions-log line per string. They **land as drafted** (D40: launching the prompt approves them) and don't block the merge (nothing is visible on prod until [m3-13](sprint-m3-13.md) sets `JUDGE_BASE_URL`); the owner may revise any of them later with a content PR.
  - the `vite:preloadError` loop-guard notice ("A new version is available · Reload", task 3);
  - the 5-minute poll stop ("Still grading — we'll show the result on Today.", task 4);
  - the ahead-of-schedule notice on a Start 409 (AB09 F3(b) has only the row label);
  - the time's-up interim line, if AB04 F12's wording doesn't fit the Workspace;
  - the judged touch re-solve (task 6): AB04 F6 is labelled "final in AB07/AB08", but AB07 has no touch frame, so build F6 as drawn with AB08's dock lines and list any delta;
  - Today's "Unfinished · #id · 12:41 left [Resume]" row (task 6): no board draws it (AB11 F4 only mentions a Resume card), so base it on AB09 F3(c)'s "In progress · 31:12 left" and AB07 F10's resume copy.
- **Coach:** the existing coach panel in attempt mode; its D27 confirm is [m1-07](sprint-m1-07.md)'s — the outlook reflects `coachUsedAt`.
- **Typed errors** from M1-05/M3-09 reach the action card's existing error state (the client's pre-check stops an over-64 KiB submit before sending); the dock's exact AB08 copy for 413 and quota ("Too large to submit (code is capped at 64 KiB)", "8 submits left today") is [m3-12](sprint-m3-12.md)'s. `judge.ts` exposes `quota` from every 2xx for it.
- **Accessibility:** focus order and keyboard per AB07's notes; the dock result region is `aria-live="polite"`; the timer ring is `role="timer"` with a minute-granular label; the editor's Tab behaviour and its escape (Esc then Tab) are announced via `aria-describedby`.

### 3 · CodeMirror 6 lazy chunk + `vite:preloadError` [X]

Sources: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) ("Lazy chunks"), [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (CodeMirror 6, not Monaco), [t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (`PartWidget.load()`).
- **Dependencies** (MIT, pinned, added to `web/package.json`): `@codemirror/state`, `view`, `commands`, `language`, `autocomplete` (bracket closing only), `lint` (diagnostics), `lang-go`, `lang-cpp`, `lang-python`. No Monaco.
- `web/src/screens/workspace/CodeEditor.tsx` is imported **only** via `React.lazy(() => import("./CodeEditor"))` from the judged workspace and the judged touch step, so non-judged screens never download it.
- **Editor theme** from `theme.css` custom properties (`EditorView.theme` referencing `var(--ds-*)`), dark only; keymaps `Mod-Enter` → Run, `Mod-Shift-Enter` → Submit; diagnostics via `@codemirror/lint` `setDiagnostics` at learner-file positions.
- **CSP check:** verify the editor's styles load under [m1-04](sprint-m1-04.md)'s CSP in Chrome, Firefox and Safari (CodeMirror uses constructable stylesheets where available). If a browser blocks them, fix it with `EditorView.cspNonce` or the narrowest CSP change, and record which in the Decisions log.
- **`vite:preloadError` → reload** in `web/src/main.tsx`, registered before any lazy import (check whether [m1-06](sprint-m1-06.md) already lazy-loads the highlighter; the handler then covers both):
  - `event.preventDefault()`, then `window.location.reload()`;
  - **loop guard:** if a reload for this reason happened < 10 s ago (a `sessionStorage` timestamp, read and written in `try/catch`), don't reload again — show a "A new version is available · Reload" notice instead (drafted copy: not on AB07, listed per task 2).
- **Gateway** (`internal/gateway/gateway.go` `serveStatic`): a missing file under `assets/` → **404** with `Cache-Control: no-store`, never the SPA shell (today a stale chunk URL gets `index.html` with 200). Client routes still fall back to the shell. Test in `gateway_test.go`.

### 4 · Judge client + drafts autosave [X]

Sources: [t4 §2.2, §2.7](../research/t4-judge-contract.md#27-transport) (`poll_after_ms`, ETag, stop at 5 min, `facts_through_seq`), [t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (draft debounce/timeout), [t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8) (Run poll 800 ms → 400 ms).
- `web/src/lib/api.ts`: `apiFetch` accepts `timeoutMs` (default stays 15 s); judge submissions and draft PUTs use **25 s** (≤ the server's 30 s `ReadTimeout`).
- `web/src/lib/judge.ts` (TanStack Query):
  - `submit(problemId, {context, parts})` (always `action: "submit"`: [m3-09](sprint-m3-09.md) answers `final`/`give_up` there with 422 `use_close_route`), `run(problemId, {context, language, parts, runInput})`, `closeAttempt(attemptId, {action, parts})`, `retryGrading(attemptId, parts)` (= `closeAttempt(attemptId, {action: "final", parts})` from `self_grade_pending`, with its own key), `getDraft`, `putDraft`;
  - **one UUIDv7 `Idempotency-Key` per user action** (a click), reused on that action's retries, never across actions;
  - `serialize()` checks the decoded code bytes (≤ 64 KiB) and the encoded request (≤ 1 MiB) **before** sending (UX only; the server is authoritative);
  - `useSubmission(id)` (the name M3-12 expects): `refetchInterval` from the last `pollAfterMs`; `If-None-Match` with the last ETag, and a 304 keeps the cached data; stop at 5 min ("Still grading — we'll show the result on Today.", drafted copy per task 2); after a terminal evaluation keep polling (phase `settling`) until `factsThroughSeq ≥ attemptSeq`, up to 60 s;
  - `useRun(id)`: 800 ms, then 400 ms;
  - typed errors (M1-05's `ApiError` codes plus M3-09's): no automatic retry on 4xx; 429 waits for `Retry-After`.
- **Drafts autosave:** keyed by context + language; debounce 15 s (t4 §5.5 for code), flushed on Run, Submit, blur, `visibilitychange → hidden` and route leave; last write wins; a "Saved · 12:03" indicator per AB07. On open: the draft for (context, language), else the starter.

### 5 · Dock slot + `DockStatus` [X]

Sources: [t4 §5.5](../research/t4-judge-contract.md#55-widget-and-result-view-contracts-ts) (`ResultProps`), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (dock tabs and copy), [m3-12](sprint-m3-12.md) task 1 (it mounts `web/src/components/judge/ResultsDock.tsx` "in the dock slot m3-11 left in `Workspace.tsx`").
- `Workspace.tsx` renders a **dock slot** under the editor (the Results tab below 1024 px) that receives `{evaluation, practice, phase: idle|queued|running|settling|done, context, parts, onFocusPart}` — t4 §5.5's `ResultProps`, so M3-12's component drops in without touching the workspace.
- Until M3-12, the slot holds a minimal `web/src/screens/workspace/DockStatus.tsx`: the phase line and the one-line headlines AB07's frames show ("Running hidden tests…", "37/40 hidden passed", "Too slow on large inputs", compile errors "see the marked lines"), and Run's per-sample pass/fail list. No History, Feedback, inconclusive or quota states here: those are AB08's, in M3-12.
- `CodeEditor` exposes a `diagnostics` prop (learner-file positions → `@codemirror/lint` marks), which both `DockStatus` and M3-12's dock feed.

### 6 · Judged touch re-solve + Today resume [X]

Sources: [m2-04](sprint-m2-04.md) Out ("Judge-graded re-solve: M3-08, M3-11"; "Unfinished: resume or give up … M3"), AB04 F6, [t4 §6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course).
- `web/src/screens/Touch.tsx`: when the touch attempt's grading mode is judge, the re-solve step renders the lazy `CodeEditor` with Run/Submit on `context: "touch"` and AB04 F6's compact results line ("Passed hidden tests at 13:20"), built as AB04 F6 draws it (AB07 has no touch frame; list any delta per task 2); probes and the self path are unchanged.
- Today (`Dashboard.tsx`, the AB05 agenda): an "Unfinished · #id · 12:41 left [Resume]" row (no board draws it: drafted copy per task 2) for an open judged course attempt, from the dashboard aggregate's practice state (if the aggregate lacks the open attempt's deadline, add an additive `openAttempts[{problemId, deadlineAt}]` field to the gateway's Today composition). Past its deadline the row disappears (the ticker concludes the Miss).

### 7 · Tests and checks [X]

- **Vitest** (`web/src/screens/Workspace.test.tsx`, the editor mocked): one test per AB07 frame in the task-2 table, plus the language switch (per-language draft), the hint confirm copy, the give-up POST body, the ahead-of-schedule copy, a typed 413/429 reaching the action card's error state, and the < 1024 px tabs (`matchMedia` mocked). Two cases spelled out:
  - **self-path variant** (`gradingMode = self`): v1 timers (15:00 attempt → 10:00 hint), the early-reveal "another attempt in 3 days" copy, the **re-implement stage present**, Run shown only when `judge.runAvailable`, **no Submit**, the uncapped four-grade picker — and on a judged fixture, no re-implement stage;
  - **`self_grade_pending`**: the cause line per `pendingReason`, grades above `ceiling` disabled, [Retry grading] posting `{action: "final", parts}` to `/api/attempts/{id}/close`, and disabled ("Retry used") when `retryUsed`.
- `ProblemRoute.test.tsx`: the screen-selection table, incl. "no `judge` block → v1 `Problem.tsx`".
- `web/src/lib/judge.test.ts`: `pollAfterMs` backoff, 304 handling, the 5-min stop, settling until `factsThroughSeq ≥ attemptSeq` (≤ 60 s), one idempotency key per action, the 25 s timeout, typed 413/429, draft debounce and flush.
- `web/src/main.test.ts` (or `preloadError.test.ts`): the event → `preventDefault` + one reload; the loop guard shows the notice; `sessionStorage` throwing still reloads once.
- `Touch.test.tsx` (judged re-solve) and `Dashboard.test.tsx` (resume row).
- **Gateway:** a missing `assets/*.js` → 404 `no-store`; a client route → the shell.
- **Compose e2e** on [m3-02](sprint-m3-02.md)'s fixture pack: extend `internal/e2e` to drive the gateway routes in the client's order (start → Run → Submit WA → Submit pass → settled Clean; give-up → Miss with `mistakeHint`). The real-runner legs use [m3-06](sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner`, build tags `e2e,runner`) and are the acceptance in CI's Linux **`judge-runner-e2e`** job; the runner can't run on macOS, so locally use the fake runner (`internal/judge/runnerfake`, the default `e2e` lane) or the spike's multipass VM. Then a manual browser walkthrough in compose as the owner (cohort) at 1440 px and 390 px (the fake runner is enough for the screens), screenshots beside AB07 in the PR.
- **Bundle check:** `vite build`, then assert the entry chunk contains no `@codemirror` module (a small `web/scripts/check-bundle.mjs` run in CI's web job); report the entry-chunk delta and the editor chunk size in the PR.
- **Deploy check (acceptance):** build twice with different chunk hashes, serve build A, open a judged problem's cover, swap to build B, press Start → the page reloads once and the editor loads (record the steps in the PR).

## Acceptance criteria

- [ ] **All AB07 frames implemented** (the task-2 table), matching the frozen board at 1440 px and 390 px (screenshots in the PR).
- [ ] **A tab open across a deploy reloads instead of 404ing the chunk**, at most once per 10 s; a missing asset is a real 404, never the SPA shell.
- [ ] The editor is a lazy chunk: non-judged screens don't download it; the entry chunk has no CodeMirror module.
- [ ] No re-implement stage on a **judged** attempt (D16); the self-path variant keeps v1's semantics (timers, early-reveal debt, re-implement from memory, Run on samples when available, no Submit, the uncapped picker: AB07 F12(a)); the hint enables at 15:00 with the Assisted-cap confirm; give-up = Miss with the solution unlocked after it.
- [ ] `self_grade_pending` shows the cause, the capped picker and Retry grading once (the m3-09 close route).
- [ ] Every string not on a frozen board is listed in the PR's "Copy not on a frozen board" section and in the Decisions log.
- [ ] Unset `JUDGE_BASE_URL` (or a non-cohort account) → the v1 `Problem.tsx` flow, unchanged.
- [ ] Drafts survive a reload and a language switch; Run/Submit use one idempotency key per action; judge calls time out at 25 s.
- [ ] The judged touch re-solve and Today's resume row work in compose.
- [ ] CI green: web tests, typecheck, lint, bundle check, `go test ./...`, the compose e2e (real-runner legs in the Linux `judge-runner-e2e` job).

## Release

**Merge only: ships in `v1.14.0`**, cut by [m3-13](sprint-m3-13.md) after [m3-12](sprint-m3-12.md) merges. Do **not** tag.
- Nothing is visible on prod until M3-13 sets `JUDGE_BASE_URL`, and then only for the owner/tester cohort (T-3); everyone else keeps the v1 `Problem.tsx`.
- No infra PR, no env, no new pod. The gateway change (asset 404) ships in the same image.

## Definition of Done

CI green · squash-merged to `main` (no tag) · the Workspace matches AB07 · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board row; M3 stays 🔄; the **Artboards** rows AB07–AB12 set to "frozen (merged, PR #N, date)" only if the ds-m3-01/ds-m3-02 sessions didn't already record them (skip any edit already done), and AB07 → consumed by M3-11; a line recording the M3 hard entry checklist green with the `host-verify --cluster` summary and date, the weekly-image date taken from the owner's before-launch attestation) · Decisions-log lines for the CSP outcome, the preloadError loop guard, the asset-404 change and each drafted string not on a frozen board (landed as drafted; the owner may revise later with a content PR).

## Risks / watch-outs

- **Editor bundle size** — lazy only on judged items; the bundle check fails CI if CodeMirror leaks into the entry chunk. Keep the language packs inside the lazy chunk too.
- **CSP vs CodeMirror styles** — test all three browsers before merging; don't loosen `script-src`.
- **Reload loops** — a broken deploy (chunk missing in the new build too) must not reload forever: the 10 s guard shows the notice instead.
- **Client-side grading logic creeping in** — the outlook, ceilings and deadlines come from the server's view; the client only formats them. Tests assert rendering from fixtures, not computed grades.
- **Polling load** — honour `pollAfterMs` and 304s; stop at 5 min; never poll a Run faster than 400 ms.
- **The v1 path must stay byte-for-byte** for non-cohort accounts: `Problem.tsx` untouched except for the router split.
- **M3-12 builds on the same files** (the dock slot, `judge.ts`, `CodeEditor`, the router): merge promptly and keep the slot's props, the hook names (`useSubmission`, `useRun`) and the `diagnostics` prop stable.
