# Prompt — Sprint m3-11 · Workspace-Code UI (AB07★) + CodeMirror lazy load

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-11.md`](../sprints/sprint-m3-11.md)   ·   **Milestone:** M3 (M3-2 slice, ships in `v1.14.0`; the first M3 UI sprint)   ·   **Prereqs:** [m3-09](../sprints/sprint-m3-09.md), [ds-m3-01](../sprints/sprint-ds-m3-01.md) + [ds-m3-02](../sprints/sprint-ds-m3-02.md) (AB07–AB12 frozen), [mi-10](../sprints/sprint-mi-10.md) (runner dark, MI-12)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, stack, and the land-and-sync rule.
- The plan: [`../sprints/sprint-m3-11.md`](../sprints/sprint-m3-11.md). Its frame table, screen-selection table and acceptance are authoritative for this session.
- **The frozen board:** `design-system/screens/v2/AB07-workspace-code.html` (every frame, copy and behaviour note), via `design-system/screens/v2/index.html`. Also AB08 (dock copy you render in AB07's frames), AB11 (badge placement, for M3-12) and AB04 F6 (the judged touch re-solve). Reuse [`design-system/theme.css`](../../../design-system/theme.css) verbatim. The v1 [`Problem.dc.html`](../../../design-system/screens/Problem.dc.html) is a reference only (superseded by AB07).
- [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D15 hard limit, D16 no re-implement, D18 45:00 / hint at 15 / Clean ≤ 20:00 and ≤ 3 failed) and [§4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) (D17). [PRD §5.6](../../prd/xlearn-v2-prd.md) (R-PF1…R-PF5, R-JG1–R-JG5, R-AC6).
- [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): presence by config and **"Lazy chunks"** (`vite:preloadError` → reload with the first lazy import).
- [t4](../research/t4-judge-contract.md): §4.1 (CodeMirror 6, not Monaco), §5.5 (widget and result-view contracts, draft debounce/timeout), §2.7 (`poll_after_ms`, ETag, 5-min stop, `facts_through_seq`), §8 (the shell, the dock copy, A1 frames), and **§13 (D14–D19 override the body; ignore §8's "15:00" cover and "Re-implement" frame)**. [t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8) (Run poll 800 → 400 ms).
- [Rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist) (the checklist below) and [§9](../rollout-plan.md#9-artboards-by-milestone) (AB07; "fix before briefing").
- Peer plans: [m3-09](../sprints/sprint-m3-09.md) (every route, DTO and error code you call), [m1-05](../sprints/sprint-m1-05.md) (typed 429/413 in `api.ts`), [m1-06](../sprints/sprint-m1-06.md) (withheld fields; a possibly-lazy highlighter), [m1-07](../sprints/sprint-m1-07.md) (coach D27 confirm), [m2-04](../sprints/sprint-m2-04.md) (Touch, Today; what it deferred to you), [m3-12](../sprints/sprint-m3-12.md) (what comes next on the same files).
- Code: `web/src/router.tsx`, `web/src/screens/Problem.tsx` (v1, keep), `Touch.tsx`, `Dashboard.tsx`, `web/src/lib/api.ts`, `curriculum.ts`, `queryClient.ts`, `web/src/main.tsx`, `web/src/components/Coach.tsx`, `web/package.json`, `web/vite.config.ts`, `internal/gateway/gateway.go` (`serveStatic`), `internal/e2e/`.

## Context

- [m3-09](../sprints/sprint-m3-09.md) shipped the judge BFF dark: submissions, Runs, polling with ETag, close/give-up, drafts, arena routes, allowlisted DTOs, typed 413/429, `quota`, `/api/judge/status`, and a `judge` block on `GET /api/problems/{id}` present only when `JUDGE_BASE_URL` is set **and** the account is in the owner/tester cohort.
- **This sprint builds the hero screen on it** — the A1 Workspace-Code, exactly as the frozen AB07 draws it — plus the lazy CodeMirror editor, the client library every judged surface reuses, and the deploy-safe chunk reload.
- **Owner-decided behaviour (never contradict):** 45:00 hard limit that never pauses; the statement appears on Start; hint available at 15:00 and opening it caps the attempt at Assisted; give-up (or revealing the solution) = Miss; a pass finishes a **judged** attempt (no re-implement on the judged path, D16); Clean = pass ≤ 20:00, no hint, no coach, ≤ 3 failed submits. The **self path** (not auto-graded) keeps v1's semantics, re-implement from memory included (AB07 F12(a), [t4 §3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution)).
- M3-11 is **the first M3 UI sprint**, so the M3 hard entry checklist gates it. It merges into `v1.14.0`; [m3-12](../sprints/sprint-m3-12.md) adds the rest of the dock, Problems, Arena and badges, and [m3-13](../sprints/sprint-m3-13.md) tags and turns judge on for the cohort.

## Entry gates — verify first (stop and report if any is unmet)

**M3 hard entry checklist** (verbatim from [rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist); a red item blocks this sprint):

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

How to check (read-only): the status.md MI rows and Decisions log for the spike GO, MI-4…MI-13 and the pack count; `ssh vps 'bash -s -- --cluster --nats-stage=n3' < ../infra/hack/host-verify.sh` (or `--nats-stage=n4` once [mi-11](../sprints/sprint-mi-11.md) has run; read-only, as [mi-02](../sprints/sprint-mi-02.md) documents) for MI-8, the memory sum and TR-STEAL; `gh pr list --state merged` for the AB07–AB12 design PRs; the Hostinger panel date (ask the owner if it isn't recorded). Rollout step ids `MI-NN` are **not** sprint ids `mi-NN` — the plan's entry-gate key maps them.

**Sprint gates:**

- [ ] [m3-09](../sprints/sprint-m3-09.md) merged (routes, `judge` block incl. `stageParams`/`languages`/`runAvailable`, DTOs incl. `retryUsed`/`pendingReason`, the close route's Retry grading, error codes).
- [ ] `design-system/screens/v2/AB07-workspace-code.html` … `AB12-week-mistakes-progress.html` exist on `main`.
- [ ] **Parallel sessions:** `gh pr list`, `git worktree list`, ListAgents. No open PR edits `router.tsx`, `Problem.tsx`, `api.ts`, `main.tsx`, `web/package.json` or `internal/gateway/gateway.go`; [m3-12](../sprints/sprint-m3-12.md) must not start in parallel.

## Do this (in order)

1. **[X] Screen selection.** `ProblemRoute` for `/:course/problem/:id` per the plan's table: no `judge` block → v1 `Problem.tsx` untouched; `judge.enabled && evaluable` → `Workspace.tsx`; not evaluable or `gradingMode=self` → the self-path variant; `self_grade_pending` → that frame; `?practice=1` unchanged (M3-12). The client never decides grading.

2. **[X] Client library.**
   - `web/src/lib/api.ts`: a per-call `timeoutMs` (default 15 s unchanged).
   - `web/src/lib/judge.ts` (TanStack Query): `submit` (always `action: "submit"`; closes never go there — m3-09 answers 422 `use_close_route`), `run`, `closeAttempt`, `retryGrading(attemptId, parts)` (= `POST /api/attempts/{id}/close {action: "final", parts}` from `self_grade_pending`), `getDraft`, `putDraft`; one UUIDv7 `Idempotency-Key` per user action (reused on that action's retries only); 25 s timeout on submissions and draft PUTs; `serialize()` pre-checks decoded code ≤ 64 KiB and the encoded request ≤ 1 MiB.
   - `useSubmission(id)` (the name M3-12 expects): `refetchInterval` = the last `pollAfterMs`; `If-None-Match` + 304 keeps data; stop at 5 min with a drafted "Still grading — we'll show the result on Today." (no board has it: list it per step 5's "Copy not on a frozen board"); after a terminal evaluation keep polling (`settling`) until `factsThroughSeq ≥ attemptSeq`, ≤ 60 s. `useRun(id)`: 800 ms, then 400 ms. No auto-retry on 4xx; 429 honours `Retry-After`.
   - Drafts: per context + language; debounce 15 s; flush on Run, Submit, blur, `visibilitychange → hidden` and route leave; "Saved · hh:mm" indicator; open = draft, else the starter generated from the item's public `signature` (AB07 F13; the `judge` block carries it).

3. **[X] Lazy editor.**
   - Add pinned MIT deps: `@codemirror/state`, `view`, `commands`, `language`, `autocomplete`, `lint`, `lang-go`, `lang-cpp`, `lang-python`.
   - `web/src/screens/workspace/CodeEditor.tsx`, loaded **only** through `React.lazy(() => import("./CodeEditor"))`; theme from `theme.css` custom properties; `Mod-Enter` → Run, `Mod-Shift-Enter` → Submit; diagnostics via `@codemirror/lint` at learner-file positions; Tab behaviour and the Esc-then-Tab escape announced via `aria-describedby`.
   - Verify under the M1-04 CSP in Chrome, Firefox and Safari; if styles are blocked, use `EditorView.cspNonce` or the narrowest CSP change and record it.

4. **[X] `vite:preloadError`.** In `web/src/main.tsx`, before any lazy import (grep first: M1-06 may already lazy-load the highlighter): `preventDefault()` then `location.reload()`; if a reload for this reason happened < 10 s ago (`sessionStorage`, in `try/catch`), show a "A new version is available · Reload" notice instead (drafted copy — AB07 doesn't draw it; list it per step 5). **Gateway:** `serveStatic` returns 404 + `Cache-Control: no-store` for a missing `assets/*` file (client routes still get the shell); test it in `internal/gateway/gateway_test.go`.

5. **[X] `Workspace.tsx` + `web/src/screens/workspace/*`** — every AB07 frame in the plan's task-2 table, in AB07's layout and copy:
   - Cover ("Start attempt · 45:00"; no statement before Start; rules from `judge.stageParams`, which m3-09 fills from the manifest before Start) → Start → full statement + editor;
   - language picker (Go/C++/Python from `judge.languages`), Run on samples + custom input, Submit;
   - the server timer ring (`role="timer"`, minute-granular label; markers at 15:00 and 20:00), never paused;
   - the grade outlook from the server's `ceiling`, `failedSubmits`, `hintOpenedAt`, `coachUsedAt`;
   - the hint button enabled from 15:00 with the confirm "caps this attempt at Assisted" (AB07 F5(b)) → `POST /api/problems/{id}/reveal`. In the judged Workspace this takes the place of v1's "I'm stuck — show a hint" / "Reveal full solution" buttons and the "Revealing the full solution now means you owe … another attempt in 3 days" warning; those strings stay in `Problem.tsx` and the self-path variant;
   - Give up · Miss modal (AB07 copy) → `POST /api/attempts/{id}/close {give_up}` with the current parts → the Solution frame;
   - Concluded Clean (provenance, facts, next touch; **no re-implement on a judged attempt**) and Concluded Miss with AB07 F9's mistake block: the *suggested* pre-filled chip, **[Change]** (category picker) and root-cause / insight fields on this problem's open entry from the existing `GET /api/mistakes?status=open` (match `problemId`), saved with the existing `PATCH /api/mistakes/{id}`; until review has opened the entry (it's async), show the chip read-only from `mistakeHint`, refetch for ≤ 30 s, then link to Mistakes; ≤ 3 concept chips only when the payload carries them;
   - Resume ("Resuming · mm:ss left · best grade now …"); time's up at 45:00 with AB04 F12's frozen "Time's up · recording your result…" (AB07 has no separate frame), polling until practice concludes the Miss, then F9's "Miss · time limit reached (45:00)"; never conclude client-side;
   - `self_grade_pending` (AB07 F11): the cause line per `pendingReason` (`contract_changed` / `unreliable` / `grading_off`; `other` → headline only), the picker capped at `ceiling` (`POST …/outcome`), **[Retry grading]** once via `retryGrading`, then "Retry used" (disabled when `retryUsed`);
   - the **self-path variant** (AB07 F12(a)): the v2 shell with v1 semantics — attempt 15:00 → hint 10:00 → solution, the early-reveal "owes another attempt in 3 days", the **re-implement from memory** stage (practice's self path still serves `stages.reimplement`), **Run on samples when `judge.runAvailable`**, **no Submit**, and the uncapped four-grade picker (the only place it appears);
   - 409 `ahead_of_schedule` on Start (only reachable by direct URL: AB09 F3(b) sends ahead rows to the arena) → a one-line notice + an arena link; typed 413/429 → the action card's error state (the dock's AB08 413/quota copy is M3-12's; expose `quota` from `judge.ts`);
   - < 1024 px: Statement / Work / Results tabs; the coach panel in attempt mode (M1-07's confirm);
   - **Copy not on a frozen board:** the preloadError notice (step 4), the 5-minute "still grading" stop (step 2), the ahead-of-schedule notice, the time's-up line if AB04 F12's wording doesn't fit, the judged touch re-solve's deltas from AB04 F6 and Today's resume row (step 7). Draft each in AB07's voice, list them in a "Copy not on a frozen board" section of the PR body for the owner's sign-off, and add a Decisions-log line per string. They don't block the merge (nothing is visible on prod until m3-13 sets `JUDGE_BASE_URL`); owner edits land before the `v1.14.0` tag.

6. **[X] Dock slot + `DockStatus`.** `Workspace.tsx` renders a dock slot (the Results tab below 1024 px) passing t4 §5.5's `ResultProps` (`{evaluation, practice, phase, context, parts, onFocusPart}`), so [m3-12](../sprints/sprint-m3-12.md) can mount its `web/src/components/judge/ResultsDock.tsx` there unchanged. Until then the slot holds a minimal `web/src/screens/workspace/DockStatus.tsx`: the phase line, AB07's one-line headlines (running, "37/40 hidden passed", "Too slow on large inputs", compile errors → marked lines) and Run's per-sample pass/fail. `CodeEditor` exposes a `diagnostics` prop (learner-file positions → lint marks). No AB08-only states here.

7. **[X] Judged touch re-solve + Today resume.** In `Touch.tsx`, when the touch's grading mode is judge, the re-solve step uses the lazy editor with Run/Submit on `context: "touch"` and AB04 F6's compact results line, built as AB04 F6 draws it (it's labelled "final in AB07/AB08", but AB07 has no touch frame; list any delta per step 5). On Today (`Dashboard.tsx`), an "Unfinished · #id · mm:ss left [Resume]" row (no board draws it: base it on AB09 F3(c)'s "In progress · 31:12 left" and AB07 F10's resume copy, listed per step 5) for an open judged course attempt (add an additive `openAttempts[{problemId, deadlineAt}]` to the gateway's Today composition only if the aggregate lacks it); it disappears after the deadline.

8. **[X] Tests.**
   - Vitest: `Workspace.test.tsx` (one test per AB07 frame, the language switch, hint confirm, give-up POST body, ahead-of-schedule, typed 413/429 on the action card, < 1024 px tabs; plus the **self-path variant** — v1 timers, early-reveal copy, the re-implement stage present, Run only when `runAvailable`, no Submit, the uncapped picker — with a judged fixture showing no re-implement stage; and **`self_grade_pending`** — cause per `pendingReason`, grades above `ceiling` disabled, Retry grading's POST body, disabled when `retryUsed`), `ProblemRoute.test.tsx` (the selection table), `judge.test.ts` (backoff, 304, 5-min stop, settling ≤ 60 s, one key per action, 25 s timeout, typed errors, draft debounce/flush), a `preloadError` test (one reload; the guard; `sessionStorage` throwing), `Touch.test.tsx`, `Dashboard.test.tsx`.
   - Bundle: `web/scripts/check-bundle.mjs` in CI's web job asserts the entry chunk has no `@codemirror` module; report both chunk sizes in the PR.
   - Compose e2e on [m3-02](../sprints/sprint-m3-02.md)'s fixture pack: extend `internal/e2e` to drive the gateway routes in the client's order (start → Run → Submit WA → Submit pass → settled Clean; give-up → Miss with `mistakeHint`). The real-runner legs use [m3-06](../sprints/sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner`, build tags `e2e,runner`) and are the acceptance in CI's Linux **`judge-runner-e2e`** job; the runner can't run on macOS, so locally use the fake runner (`internal/judge/runnerfake`) or the spike's multipass VM.
   - Deploy check: build twice (different chunk hashes), serve A, open a judged cover, swap to B, press Start → exactly one reload, then the editor loads. Write the steps and result in the PR.
   - Then `npm run typecheck`, `npm run lint`, `npm test`, `npm run build` in `web/`, `go test ./...`, the e2e lane.

9. **Eyeball against AB07** in compose as the owner (cohort), at 1440 px and 390 px (the fake runner is enough for the screens), or with the temporary vite mock config used for v1 visual checks (delete it before committing). Put side-by-side screenshots in the PR.

## Constraints

- **Server-authoritative:** timers, deadlines, ceilings, grades, unlocks and context ids come from the server; the client renders and polls. No client-side grading, no client timer state beyond display.
- **Presence by config:** without the `judge` block the app is exactly v1 (`Problem.tsx` untouched apart from the router split). `JUDGE_BASE_URL` stays unset on prod until M3-13.
- **Frontend:** `theme.css` tokens and components verbatim (no Tailwind), dark theme, the frozen AB07 copy and layout; difficulty tokens Easy `--ds-ok`, Medium `--ds-warn`, Hard `--ds-err`; keyboard and screen-reader behaviour per AB07's notes.
- **Bundle:** CodeMirror (and its language packs) only in the lazy chunk; no Monaco; pinned MIT dependencies.
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** the only backend changes are the gateway asset 404 and, if needed, one additive Today field; no schema change (so no goose/sqlc work unless you touch a query — then `sqlc diff` must stay clean).
- **GitOps:** never `kubectl apply`; `ssh vps` read-only for the checklist. **No alerting (D34).** No new pod (memory sum unchanged).
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before merging and before claiming an ADR number.
- This sprint **does not tag**.

## Deliverables

- `web/src/screens/Workspace.tsx`, `web/src/screens/workspace/*` (incl. the lazy `CodeEditor.tsx`), `ProblemRoute` in `router.tsx`.
- `web/src/lib/judge.ts` (`useSubmission`, `useRun`, …), `api.ts` `timeoutMs`, drafts autosave, the dock slot + `DockStatus.tsx`, the editor's `diagnostics` prop.
- The `vite:preloadError` handler with its loop guard; the gateway's asset 404.
- The judged touch re-solve step; Today's resume row.
- Tests, `web/scripts/check-bundle.mjs` wired into CI, the compose e2e leg, AB07 screenshots and the deploy check in the PR.

## Update status

- In [`../sprints/sprint-m3-11.md`](../sprints/sprint-m3-11.md): set each task 🔄 → ✅ (⛔ with a reason), and set _Overall_.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for M3-11 (M3 stays 🔄);
  - the **Artboards** rows AB07–AB12 → "frozen (PR #, date)" if the design sprints didn't set them, and AB07 → consumed by M3-11;
  - a line recording **the M3 hard entry checklist green**, with the date and the `host-verify --cluster` summary (memory sum, steal p95, NATS stage);
  - **Decisions log** lines: the CSP outcome for CodeMirror, the preloadError loop guard, the asset-404 change, the editor chunk size, any additive Today field, and each drafted string not on a frozen board (pending owner sign-off).
- No ADR is expected (ADR-0029/0034 cover it). If you depart from them, run the parallel-sessions check before numbering one.

## Done when (acceptance)

- [ ] All AB07 frames implemented and matching the frozen board at 1440 px and 390 px (screenshots in the PR).
- [ ] A tab open across a deploy reloads instead of 404ing the chunk (once per 10 s at most); a missing asset is a real 404.
- [ ] The editor is a lazy chunk (the bundle check passes); non-judged screens never load it.
- [ ] No re-implement on a **judged** attempt; the self-path variant keeps v1's semantics incl. re-implement, Run on samples when available and no Submit (AB07 F12(a)); hint from 15:00 with the Assisted-cap confirm; give-up = Miss, then the solution.
- [ ] `self_grade_pending` shows the cause, the capped picker and Retry grading once.
- [ ] Every string not on a frozen board is listed in the PR and the Decisions log.
- [ ] No `judge` block (unset `JUDGE_BASE_URL` or non-cohort) → the v1 `Problem.tsx` flow, unchanged.
- [ ] Drafts survive a reload and a language switch; one idempotency key per action; 25 s timeouts.
- [ ] The judged touch re-solve and Today's resume row work in compose.
- [ ] CI green (web typecheck/lint/tests/build, bundle check, `go test ./...`, compose e2e incl. the Linux `judge-runner-e2e` job).

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: merge only (ships in `v1.14.0`)**. That means branch `feat/m3-11-workspace-code`, conventional commits with the attribution lines, a PR (with the AB07 screenshots and the deploy check), CI green, and a squash-merge, then `git checkout main && git pull`. **Do not tag**; [m3-13](../sprints/sprint-m3-13.md) cuts `v1.14.0` after [m3-12](../sprints/sprint-m3-12.md). There's no infra PR.
