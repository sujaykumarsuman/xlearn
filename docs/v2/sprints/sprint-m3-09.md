# Sprint m3-09 — gateway judge BFF, DTO allow/deny lists, typed 413, L16, degradation status

> **Milestone:** M3 — judge plus code grader (**M3-2** slice, rides `v1.14.0`)   ·   **Track:** product   ·   **Order:** 49
> **Prereqs:** [m3-07](sprint-m3-07.md) (`v1.13.0` live: judge dark on prod) · [m3-14](sprint-m3-14.md) (judge learner API, admission, drafts, arena history/progress) · [m3-08](sprint-m3-08.md) (practice close/give-up, `GET /state/{problemId}` M3 fields, gating)
> **Unblocks:** [m3-11](sprint-m3-11.md) (Workspace-Code UI) · consumed by [m3-12](sprint-m3-12.md) (dock, Problems, Arena, badges) and [m3-13](sprint-m3-13.md) (exit tests, `JUDGE_BASE_URL`)
> **Release action:** **merge only** (ships dark in `v1.14.0`, tagged by [m3-13](sprint-m3-13.md)) · no infra PR · no new pod
> **Calendar:** mid-November (agent work, no owner time)
> **Execute with:** [`../prompts/prompt-m3-09.md`](../prompts/prompt-m3-09.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Judge client, config and presence gate (`JUDGE_BASE_URL` + cohort) + the problem aggregate's `judge` block | X | ⬜ |
| 2 | Submission and Run routes: enqueue (`aud=judge`), context resolution, poll with `poll_after_ms` + ETag, practice composition | X | ⬜ |
| 3 | Close and give-up orchestration (gateway → judge → practice, sync) + attempt-start gating | X | ⬜ |
| 4 | Drafts, arena history/diff, arena progress, Mark studied, arena reveal record (D10, D17) | X | ⬜ |
| 5 | Learner DTO allowlist at the BFF + per-endpoint, per-state denylist test | X | ⬜ |
| 6 | Limits: typed 413 (64 KiB code), quota fields, typed 429 pass-through, L16 classic-mock caps | X | ⬜ |
| 7 | Degradation status `GET /api/judge/status` (saturated · breaker · evaluable=0 · per-item mismatch) | X | ⬜ |
| 8 | `withhold()` extension reconciled with D17 + route-enumeration rows | X | ⬜ |
| 9 | IDOR + presence tests, compose e2e legs, docs (`api.md`, `openapi.yaml`, `services.md`) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + any
> milestone). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **`v1.13.0` live** ([m3-07](sprint-m3-07.md)): `xlearn-judge` Ready on prod with the evalpack mounted, [`../status.md`](../status.md) carries m3-07's record (MI-13 ✅ and the evalpack-stream row with the item count, its task 10; its task-9 verify read N evaluable) — don't re-run `judge admin status` for this gate: every `judge admin` verb, reads included, writes an `admin_audit` row, [m3-14](sprint-m3-14.md) task 5 — and the gateway's `JUDGE_BASE_URL` is still **unset** (judge dark).
- [ ] [m3-14](sprint-m3-14.md) merged (it ships in `v1.13.0`): judge's learner API (`POST /submissions`, `GET /submissions/{id}`, `POST /runs`, `GET /runs/{id}`, drafts, arena history and parts, `GET /arena/progress`, `POST /arena/{item}/studied`), its internal `GET /internal/status?path=` (no JWT, ClusterIP-only), admission L9–L15 with typed 429/503, `quota` on every 2xx, per-account idempotency, and judge's own DTO allowlist.
- [ ] [m3-08](sprint-m3-08.md) merged (the build order puts it right before this sprint), with the internal contract its task 7 lists: `POST /problems/{id}/attempt/start` + `{judge}`, `GET /state/{problemId}` with the M3 fields, `POST /attempts/{id}/close`, the `hint_locked` / `use_give_up` / `evaluated_item` 409s, and `POST /problems/{id}/arena-reveal`. If a peer is still on M3-08, build against its PR's contract and rebase before merging.
- [ ] The M1b gateway floor is on `main`: [m1-04](sprint-m1-04.md)'s `authSession` + `inCohort()` (`internal/gateway/cohort.go`), [m1-05](sprint-m1-05.md)'s `httpx.ReadBody` + L5 per-account buckets + typed 429/413 envelopes, [m1-06](sprint-m1-06.md)'s `withhold()` + route-enumeration test, [m1-03](sprint-m1-03.md)'s course resolution.
- [ ] **Parallel sessions:** no open peer PR edits the `apiRoutes()` table in `internal/gateway/bff.go` or `internal/gateway/withhold.go`. [m3-10](sprint-m3-10.md) runs beside this sprint and touches `internal/gateway/{progress,mistakes,public}.go` plus assessment migrations; this sprint touches assessment **queries only** (L16), so re-run `sqlc generate` after rebasing on it (`gh pr list`, `git worktree list`, ListAgents).

The M3 hard entry checklist ([rollout §5](../rollout-plan.md#5-m3-hard-entry-checklist)) gates the first M3 **UI** sprint ([m3-11](sprint-m3-11.md)) and the `v1.14.0` tag ([m3-13](sprint-m3-13.md)), not this sprint: everything here ships dark behind `JUDGE_BASE_URL`.

## Goal

Expose the judge to the SPA **safely**, through the gateway only
([ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract),
[ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) rows 9–11,
[t4 §2](../research/t4-judge-contract.md#2-the-common-contract)):
- the gateway mints **`aud=judge`** JWTs and resolves every context id itself (the SPA never sends one);
- submissions, Runs, polling, drafts, arena history/progress and close/give-up get BFF routes;
- a **learner DTO allowlist** at the BFF plus a **denylist test** prove the pack never crosses it;
- limits fail loudly: a typed **413** at 64 KiB of code, quota fields ("8 submits left today"), the judge's typed 429s, and **L16** classic-mock caps;
- `GET /api/judge/status` feeds the degradation badges ([m3-12](sprint-m3-12.md), AB11).

Every route is **present only by config** ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)): with `JUDGE_BASE_URL` unset, or for a non-cohort account, each one answers the same 404 as an unknown `/api` route and v1 behaviour is unchanged.

## Scope

**In**
- gateway: `internal/gateway/judge.go` (client, presence gate, handlers), `judge_dto.go` (allowlist types), `arena.go`, `judge_status.go`; new rows in `apiRoutes()`; the `judge` block on `GET /api/problems/{id}`; `internal/platform/config` (`JUDGE_BASE_URL`, `JWT_AUD_JUDGE`); `cmd/gateway/main.go`; `docker-compose.yml` gateway env (local only).
- Contexts `course`, `touch`, `arena` and the `run` action. `mock` is refused at the BFF (422 `invalid_context`) until M6a.
- The close/give-up orchestration of [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers), `final` included (the P quiz uses it in [p-03](sprint-p-03.md)).
- L16 in assessment (`POST /mocks`, a count query, no migration) plus the SPA error mapping on `Mock.tsx`.
- Small, additive **practice** read fields, **only where M3-08 lacks them**: any `GET /state/{problemId}` field the poll view needs (task 2, incl. `retry_used` and `pending_reason`) and the touch fields on `GET /attempts/open` (task 2). **No judge edit:** [m3-14](sprint-m3-14.md)'s 2xx already carries `quota` (task 6). No migration anywhere in this sprint.

**Out**
- Every judge UI: the Workspace, editor and client lib → [m3-11](sprint-m3-11.md); the results dock, Problems markers, Arena and badges → [m3-12](sprint-m3-12.md).
- SSE: deferred by [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) ("Transport"); polling is the only transport in v2.0. Bounded SSE for interviews → [m6a-04](sprint-m6a-04.md).
- The `mock` context's Runs/finals → [m6a-04](sprint-m6a-04.md). Dispute, re-grade, claims and AI allowance routes → M4 ([m4-04](sprint-m4-04.md), [m4-05](sprint-m4-05.md)).
- Setting `JUDGE_BASE_URL` on prod → [m3-13](sprint-m3-13.md)'s infra PR after the `v1.14.0` tag.
- Judge-side admission and quotas (already [m3-14](sprint-m3-14.md)); practice's strategies (already [m3-08](sprint-m3-08.md)).

## Tasks

### 1 · Judge client, config and presence gate [X]

Sources: [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (presence by config, T-1/T-2/T-3), [ADR-0033 §7](../../adr/0033-invite-only-admission-and-owner-admin.md) (roles never in the JWT), [t4 §2.2](../research/t4-judge-contract.md#22-submission-request-post-apiproblemsidsubmissions--judge-post-submissions).
- **Config** (`internal/platform/config/config.go`): `JudgeBaseURL` (env `JUDGE_BASE_URL`, **default empty**, like `COACH_BASE_URL`) and `JWT.AudienceJudge` (env `JWT_AUD_JUDGE`, default `judge`). Wire both through `gateway.Options` in `cmd/gateway/main.go`. Compose sets `JUDGE_BASE_URL=http://judge:8087` for the gateway (local only; prod stays unset until M3-13).
- **Client** `judgeClient{get, post, put}` beside the other upstream clients: 10 s timeout (enqueue is milliseconds, [t4 §2.7](../research/t4-judge-contract.md#27-transport)), `LimitReader` on **response** bodies only, forwards `Idempotency-Key`, `If-None-Match` and `Retry-After`. Mint with `g.mintFor(w, accountID, g.audJudge)`; roles stay `["learner"]`.
- **Presence gate** — one helper, used by every judge route:
  - `const judgeCohortOnly = true` — the T-1 code default; [ga-01](sprint-ga-01.md) flips it and removes the cohort gate.
  - `g.judgeFor(w, r) (accountID string, info sessionInfo, ok bool)` = `authSession` → `g.judge != nil && (!judgeCohortOnly || inCohort(info))`; otherwise it writes **exactly** `apiNotFound`'s 404 (same status, body, headers).
  - The routes stay registered in `apiRoutes()` (the OpenAPI drift and route-enumeration tests see them); only the handler answers 404.
- **Problem aggregate** (`handleGetProblem`, course view): when present, add `judge{enabled, evaluable, runAvailable, gradingMode, languages[], stageParams{timeLimitS, hintAtS, cleanWithinS, maxFailedForClean}, limits{codeBytes, runInputBytes}}`:
  - `evaluable` (`status = ok ∧ submit_available`) and `runAvailable` (`run_available`) from judge `GET /internal/evaluable?path=` ([m3-05](sprint-m3-05.md) task 3: `{admission, pack, items[{item_id, status, contract_hash, grader_kinds, run_available, submit_available}]}`; decode with m3-05's wire types in **`internal/platform/judgeapi`** (`Evaluable`, `EvaluableItem`), never a local copy), cached per path for 30 s in-process (single replica, L24). That response carries **no** languages;
  - `languages[]` from the item's **public** code part, `parts[].config.languages[]` ([m1-01](sprint-m1-01.md)'s frozen item schema, in the embedded `internal/course` content — the same `all:courses` loader judge uses, m3-05). The SPA generates each language's starter from the same part's public `signature` (AB07 F13), so if the aggregate doesn't already carry the public part config, add `parts[{id, type, signature, samples, constraints}]` to the `judge` block (public fields only);
  - `gradingMode` from practice's `GET /state/{problemId}` ([m3-08](sprint-m3-08.md) task 7) once an attempt exists;
  - `stageParams` (limit 45, hint at 15, clean 20, max failed 3, [D18](../feasibility.md#decisions-log-newest-first)): the attempt's pinned `stage_params` from practice once an attempt exists, **else** the course manifest's `grading.params["verdict_timer@1"]` ([m1-01](sprint-m1-01.md); read through [m1-03](sprint-m1-03.md)'s `resolveCourse(slug)` over the compiled `internal/course` manifests), so the AB07 cover (45:00, hint at 15, Clean ≤ 20:00 / ≤ 3 failed) renders **before** Start;
  - **absent** when not present, so the SPA keeps the v1 `Problem.tsx` (the kill switch = v1 behaviour).

### 2 · Submission and Run routes [X]

Sources: [t4 §2.2–§2.3, §2.7](../research/t4-judge-contract.md#27-transport), [t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8) (Run poll cadence).

| BFF route | Upstream | Notes |
|---|---|---|
| `POST /api/problems/{id}/submissions` | judge `POST /submissions` ([m3-14](sprint-m3-14.md) task 3) | body `{context: course\|touch\|arena, action: submit, parts[]}` — **`submit` only**: a `final` or `give_up` here → **422 `use_close_route`**, because every close goes through `POST /api/attempts/{id}/close` (task 3), where practice records it synchronously ([t4 §2.1, §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers)); **`Idempotency-Key` required** (400 `idempotency_key_required`); the gateway adds `item_id`, `context.id` (+ `band` for touch) and the pinned `contract_hash` from practice; 200 inline `{evaluation, quota}` / 202 `{submissionId, attemptSeq?, state, pollAfterMs, queuePosition, quota}` |
| `POST /api/problems/{id}/runs` | judge `POST /runs` | body `{context, language, parts[], runInput?}`; never counted, no seq; a new Run cancels the account's queued Runs in that context (judge) |
| `GET /api/submissions/{id}` | judge `GET /submissions/{id}` (+ practice state) | `ETag` → 304; `pollAfterMs`; see composition below |
| `GET /api/runs/{id}` | judge `GET /runs/{id}` | `ETag` → 304; cadence 800 ms, then 400 ms ([t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8)) |

- **Context resolution (never from the body):**
  - `course` → from practice's `GET /state/{problemId}` ([m3-08](sprint-m3-08.md) task 7), which covers the **course** attempt only: the open course attempt for `(sub, item)`: owned, open, not concluded; else 409 `no_open_attempt`; enrolment checked (403 `not_enrolled`). **Refuse a counted submit when `now > deadline_at + 2 s`** → 409 `attempt_expired` (M3-08 leaves this gateway check to this sprint; practice's ticker concludes the Miss).
  - `touch` → from practice's `GET /attempts/open?problem_id=` ([m1-07](sprint-m1-07.md)): [m2-01](sprint-m2-01.md) reports the live touch as an `attempts[]` entry with `purpose: "touch"` (no separate `touches` slot) → its `attemptId` is the context id. `band` and `deadlineAt` come from the same entry: if it lacks them, add them there as **additive read fields** (practice already stores `attempt.band` and the touch `timer.deadline_at`, m2-01; no migration). Don't resolve through `GET /touches/{attempt_id}`: its first call sets `probes_shown_at` (m2-01). No live touch → 409 `no_open_attempt`; refuse when `now > deadline + 5 s` (409 `touch_expired`, [t4 §2.3](../research/t4-judge-contract.md#23-admission)).
  - `arena` → kind only; judge derives `uuidv5(sub, item)` itself and ignores any client value ([m3-14](sprint-m3-14.md)). Arena Submit is **never locked** (D17, task 8).
  - `mock` → 422 `invalid_context` (M6a).
- **Body handling:** `httpx.ReadBody(w, r, httpx.BodyLimitDefault)` (1 MiB → 413 `body_too_large`), then decode `parts[]` as **raw JSON values** (not escaped strings) and check caps before calling judge (task 6).
- **Poll composition** for counted contexts (course, touch): judge's DTO plus practice's state, both allowlisted (task 5). From M3-08's `GET /state/{problemId}`: `gradingMode, state, deadlineAt, hintAvailableAt, grade?, gradedBy?, trust?, factsThroughSeq, concluded, ceiling?, arenaRevealed`. The Workspace outlook and conclusion also need `failedSubmits`, the hint-opened time, the coach-assist flag (M1-07's `assist`), `resolution` and, after conclusion, `mistakeHint{category, strength}` / `conceptsHint[]`; the `self_grade_pending` frame (AB07 F11) needs **`retryUsed`** (practice's `attempt.retry_used`, [m3-08](sprint-m3-08.md) task 1 — not in its task-7 field list) and **`pendingReason: contract_changed | unreliable | grading_off | other`**, derived by practice from what it already stores (a `contract_changed` fact → `contract_changed`; `released_closes = 3` → `unreliable`; moved by the kill-switch sweep, i.e. neither → `grading_off`; anything else, e.g. `budget_exhausted` → `other`, rendered with F11's generic line). If M3-08's state response lacks any of these, add them there as **additive read fields** (practice already stores the inputs; no migration). The SPA keeps polling until `factsThroughSeq ≥ attemptSeq` ([t4 §2.7](../research/t4-judge-contract.md#27-transport)); the gateway only reports both numbers. Judge's `key` marks/`answer_reveal` are requested (`concluded=true`) only when practice's state says the context is concluded.
- **ETag:** sha256 of the composed body; `If-None-Match` equal → 304 with no body. `pollAfterMs` passes judge's backoff (500 ms → 1 s → 2 s → 5 s) through.
- **Cache epoch:** call `g.invalidateAgg` on a **200** (not 304) poll whose practice view is locked or concluded, and on an arena-progress change (task 4) — never on judge's terminal result alone ([t4 §2.7](../research/t4-judge-contract.md#27-transport) "Cache epoch").
- **Typed pass-through** ([m3-14](sprint-m3-14.md) task 2's table): 409 `context_closed` / `part_locked`; 422 `invalid_part` / `idempotency_key_reused` / `nothing_to_evaluate`; 429 `quota_exceeded{kind, resetAt}` / `queue_full` / `too_many_pending` / `too_fast{reason?}` / `budget_exhausted{window, resetAt}` / `shed{reason}` (all with `Retry-After`); 503 `evaluation_unavailable{fallback}` — each keeps its code and `details` in the gateway envelope (decoded into typed error structs, never a raw body).

### 3 · Close, give-up and attempt start [X]

Sources: [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers), [§3.4](../research/t4-judge-contract.md#34-give-up-which-for-evaluated-items-includes-reveal-full-solution), [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer) (D15, D16).
- **`POST /api/attempts/{id}/close`** `{action: final|give_up, parts[]}` with a required `Idempotency-Key` — the **only** close route (the submissions route refuses `final`/`give_up`, task 2):
  1. read practice's attempt (owner = `sub`, open **or** `self_grade_pending`, pin, purpose, `grading_mode`); a foreign or unknown id → 404. In `self_grade_pending` only `action: final` with `retryUsed = false` is accepted (else 409 `retry_used`, before judge is called): that call **is AB07 F11's "Retry grading"** — practice accepts a close from `self_grade_pending` once and sets `retry_used` ([m3-08](sprint-m3-08.md) task 3);
  2. judge `POST /submissions` with `closes_context=true` → `{submissionId, seq, submittedAt}` (a close exists → judge replays it);
  3. practice `POST /attempts/{id}/close {action, submission_id, close_seq, submitted_at}` — **synchronous and authoritative**, idempotent on `submission_id`;
  4. respond with the composed practice view (give-up: the solution unlocks now; the grade locks at the earliest pass below `close_seq`, else Miss) and invalidate the account's aggregates.
  - If step 3 fails after step 2, return 502 `upstream`; practice's consumer backstop records the close from the fact ([m3-08](sprint-m3-08.md)), and a client retry with the same key replays step 2 and re-runs step 3.
- **Hint and solution for judged attempts:** the hint stays the existing `POST /api/problems/{id}/reveal` (practice records the `HintAt` fact; before 15:00 it answers 409 `hint_locked{availableAt}`, [PRD R-PF1](../../prd/xlearn-v2-prd.md)); revealing the solution **is** give-up (D15): practice answers 409 `use_give_up`, the gateway passes it through, and the SPA's "Reveal solution" opens the give-up modal (the close route above, which carries the current parts). `POST /api/problems/{id}/outcome` on a judged attempt passes practice's 409 `evaluated_item` through (INV-9); in `self_grade_pending` it's the capped pick.
- **Attempt start** (`POST /api/problems/{id}/attempt/start`) for a judged item:
  - sends `{"judge": true}` to practice ([m3-08](sprint-m3-08.md) task 3) **only** after the T-3 cohort check passes, because only the gateway knows `role`; practice then pins `grading_mode` from judge's `/internal/evaluable`;
  - an item ahead of the learner's frontier → 409 `ahead_of_schedule{scheduledWeek, currentWeek}` instead of v1's `counted:false` ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) behaviour change; the SPA offers the arena).

### 4 · Drafts and arena routes [X]

Sources: [ADR-0027 §4](../../adr/0027-content-evalpack-and-user-data-model.md#4-problems-arena) (D10), [ADR-0029 §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) (D17), [t4 §3.7](../research/t4-judge-contract.md#37-touches-mocks-arena) (as overridden by D17).

| BFF route | Upstream | Notes |
|---|---|---|
| `GET /api/problems/{id}/drafts?context=course\|touch\|arena&language=` | judge `GET /drafts?context_kind=&context_id=&item_id=` | the learner's own draft for the resolved context, part and language |
| `PUT /api/problems/{id}/drafts` | judge `PUT /drafts/{context_kind}/{context_id}/{part_id}` | `{context, partId, language, payload}`; same caps as a submission (413); no `Idempotency-Key` (last write wins) |
| `GET /api/problems/{id}/arena/history?cursor=` | judge `GET /arena/{item}/history?cursor=` | rows `{submissionId, submittedAt, language, status, class?, hidden{passed,total}?}`; newest first |
| `GET /api/problems/{id}/arena/history/{submissionId}` | judge `GET /arena/{item}/submissions/{id}/parts/{part_id}` | the learner's **own** submitted code for Copy-to-editor and Diff (Diff is client-side) |
| `GET /api/paths/{slug}/arena/progress` | judge `GET /arena/progress?path=` | `{<itemId>: {done, source: auto\|manual, firstPassedAt, submits}}` in **one** batched read; feeds the Problems dual markers and the Arena's marker (m3-12) |
| `POST /api/problems/{id}/arena/studied` | judge `POST /arena/{item}/studied` | "Mark studied" → `source=manual`; invalidates aggregates |
| `POST /api/problems/{id}/arena/reveal` | practice `POST /problems/{id}/arena-reveal` ([m3-08](sprint-m3-08.md) task 3) | the spoiler confirm: `arena_revealed_at = COALESCE(arena_revealed_at, now())`, then the gateway returns the solution sections; **recorded, never a cap (D17)** |

- **Arena reveal:** the arena view (M1-06) withholds solution stages for a live item; in M3 (cohort, judged items) it also withholds them for an item with **no concluded course attempt** yet, so the spoiler confirm is the only way in and is always recorded. A concluded item's arena view shows the solution directly.
- Arena Submit, History and Copy-to-editor are **not locked** during a live attempt or a due touch (D17 supersedes t4 §3.7's locks and §11.4 #20's lock half).

### 5 · Learner DTO allowlist + denylist test [X]

Sources: [t4 §2.6](../research/t4-judge-contract.md#26-evaluation) (learner DTO, denylist), [t4 §10](../research/t4-judge-contract.md#10-security-and-integrity-notes), [ADR-0027 §1, §5](../../adr/0027-content-evalpack-and-user-data-model.md#5-leak-and-integrity-controls-preconditions-for-m3), [PRD R-CT2](../../prd/xlearn-v2-prd.md).
- **Allowlist at the BFF** (`internal/gateway/judge_dto.go`): every judge and practice response is decoded into gateway-owned structs and re-encoded; unknown upstream fields are dropped, so a new judge field reaches the SPA only through a reviewed struct edit. The shapes:
  - all: `submissionId, status, reason, action, contextKind, submittedAt, completedAt, attemptSeq, checks[{key,label,required,met}], score, trust, evaluatedWith` (a public label, never digests);
  - Run: `compile{ok, diagnostics}` (≤ 8 KiB, learner files only) and per sample/custom input `{id, status, class, got, sampleExpected, stdout, stderr}` within judge's caps;
  - Submit: samples in full on failure; hidden only `{passed, total}`, `perf: passed|failed|not_run`, `firstFailure.class`, bucketed usage;
  - `key` marks and `answerReveal` only when the context is concluded (P needs them; DSA code has none);
  - practice view (poll and close responses): exactly the task-2 composition fields, incl. `retryUsed` and `pendingReason`.
- **Denylist test** `internal/gateway/judge_denylist_test.go`, per endpoint **and** per state (queued, running, terminal pass/fail/inconclusive, concluded vs not, Run vs Submit vs arena vs touch, drafts, history, status):
  - a **poisoned judge fake** returns every forbidden key with sentinel values: `expected, args, input, case_id, anchors, key, aliases, seed, gen, test_name, per_test, signals, versions, hidden_files, pack_path` plus per-case timings/ordinals and strings like `/evalpack/`, `EVALPACK_DIR`, `HIDDEN-SENTINEL`;
  - the walk fails on any forbidden key or sentinel anywhere in any body; `sampleExpected` is allowed only in Run and sample views, `answerReveal` only when concluded;
  - a **compose leg** (task 9) runs the same walk against the real judge with [m3-02](sprint-m3-02.md)'s fixture pack, whose hidden cases carry sentinel strings.
- This is the ADR-0033 §12 row 11 proof: "the pack never goes through the BFF".

### 6 · Limits: typed 413, quota, L16 [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L9–L11, L16), [t1 §6.3](../research/t1-content-data-model.md#63-caps-and-quotas-enforced-by-judge-t3t4-own-rate-limiting) (payload caps), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (413/quota copy).
- **Registry** (`internal/platform/httpx`, beside M1-05's `BodyLimit*`): `CodePartLimit = 64 << 10` (decoded source bytes), `MultiFilePartLimit = 128 << 10` (≤ 16 files), `RunInputLimit = 64 << 10`.
- **Typed 413** before calling judge: a code part whose decoded file contents exceed the cap → `413 {"error":{"code":"part_too_large","details":{"partId":…,"max":65536}}}`; custom Run input → `run_input_too_large`. The same code judge uses ([t4 §2.3](../research/t4-judge-contract.md#23-admission) row 3), so the SPA handles one shape. Judge still enforces its own caps (defence in depth). Drafts use the same check.
- **Quota fields:** every 2xx submission/Run response carries `quota{kind: runs|counted|arena, remaining, limit, resetAt}` (L9), mirrored in `X-Quota-Remaining` / `X-Quota-Reset` headers; the SPA shows "8 submits left today" per AB08. **Judge already sends it:** [m3-14](sprint-m3-14.md) task 2 puts `quota{kind, remaining, limit, reset_at}` on every 2xx of `POST /submissions` and `POST /runs` (and task 3's 200/202 shapes carry it), so the gateway only decodes it into its allowlisted DTO (`reset_at` → `resetAt`) and sets the headers — **no judge edit**. A 429 keeps judge's `Retry-After`.
- **L16 classic mocks** (assessment, `internal/assessment/store` + `handlers.go`, no migration):
  - in `CreateMock`'s transaction, take `pg_advisory_xact_lock(hashtext('mock-start:'||account))`, then count;
  - **≤ 1 live per course**: a `mock_session` with `status='live' AND deadline_at > now()` for the same `path_slug` → **409 `mock_in_progress{mockId}`** (the SPA resumes it);
  - **≤ 3 started per UTC day** per account → **429 `mock_daily_limit{resetAt}`** + `Retry-After`;
  - an unscored session past its deadline no longer blocks (it can still be scored); record both definitions in the Decisions log;
  - `web/src/lib/mock.ts` / `Mock.tsx`: 409 → navigate to the live session; 429 → the existing error state with "You've started 3 mocks today. Try again after <time>."
- New sqlc queries in assessment only; `sqlc generate` after rebasing on M3-10's assessment migration.

### 7 · Degradation status [X]

Sources: rollout [§4 M3](../rollout-plan.md#4-per-milestone-detail) ("degradation badges", "judge telemetry rows, badges and `judge admin` reads replace [opscheck]"), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L13, L15), D34.
- `GET /api/judge/status[?path=<slug>]` → `{state: ok|saturated|breaker|off, breakerLevel?, evaluable, items?{<id>: evaluable|contract_mismatch|unsupported|invalid|no_pack}}`:
  - `off`: judge's `admission: closed` (the L15 `JUDGE_ADMISSION` kill switch, [m3-05](sprint-m3-05.md)) **or** judge unreachable / timed out;
  - `breaker`: the L13 duty-cycle breaker at level 1 or 2 (`breakerLevel` from `breaker{level}`);
  - `saturated`: judge's flag (runner queue ≥ 75 % of the L10 cap, or the runner lane unreachable);
  - `evaluable: 0` → every item falls back to self-report ("grading pending");
  - per item (with `?path=`), mapped from m3-05's `/internal/evaluable` status vocabulary (`ok | spec_mismatch | unsupported | invalid | no_pack`, `internal/platform/judgeapi`): `ok` → `evaluable`; `spec_mismatch` → **`contract_mismatch`** (the same condition — the live public `contract_hash` isn't in the pack's accepted hashes — which `/internal/status` reports per item under that name; the learner-facing name wins); `unsupported`, `invalid` and `no_pack` pass through unchanged (AB11 renders all three as "Grading pending · self-report"). A mismatch shows on that item only.
- Upstream: judge **`GET /internal/status?path=`** ([m3-14](sprint-m3-14.md) task 3: `saturated`, `breaker{level}`, `evaluable` count, per-item `contract_mismatch`, `admission: open|closed`) plus `GET /internal/evaluable?path=` for the per-item rest. Both are **internal routes with no JWT** (ClusterIP-only, the ADR-0016 internal model behind MI-5a and judge's ingress policy), so the gateway calls them without minting a token, exactly like task 1's evaluable read; no account data crosses them.
- Cached in-process for 15 s; present-by-config like every judge route (404 when absent). **Not an alert** (D34): an in-app badge only, rendered by [m3-12](sprint-m3-12.md) (AB11).

### 8 · `withhold()` extension (reconciled with D17) [X]

Sources: [t4 §6.4](../research/t4-judge-contract.md#64-concepts-to-revise), [t4 §11.4](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag) #20, D17 in [ADR-0029 §4](../../adr/0029-judge-contract-and-learning-signal.md#4-learning-loop) (it overrides t4 §3.7/§8's arena locks), [m1-06](sprint-m1-06.md).
- **Concept chips:** `conceptsHint[]` and any `concepts` field on the new responses go through `withhold()`; the primary concept is stripped while the item is live (open counted attempt, due or live touch).
- **Arena Submit, History, Copy-to-editor:** registered as arena surfaces in the route-enumeration test with policy `applied` (their item fields — pattern, concepts, solution stages — go through `withhold()`), but **never locked** (D17). A test asserts an arena Submit and a History read succeed while a course attempt is open and while a touch is due.
- **Arena reveal:** recorded once through practice (task 4) → becomes `arena_prior=revealed` on the later `problem_solved`; **no cap** (M3-08's strategies must not cap; a test here asserts the reveal response and the practice write, not the grade).
- Every new `apiRoutes()` row declares its `withholdPolicy` (exempt rows carry a reason: e.g. "evaluation DTO; allowlisted by the denylist test"), and the behavioural sweep gains the judge fakes.

### 9 · Tests and docs [X]

- **IDOR** (`internal/gateway/judge_idor_test.go` + compose): account B reading account A's submission, Run, draft, arena history row, or closing A's attempt → **404**, byte-identical to an unknown id. Judge scopes every read by `sub` ([ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 9); the gateway never forwards an id without its own `sub` token.
- **Presence:** `JUDGE_BASE_URL` unset → every judge route 404 byte-identical to `/api/nope`, and `GET /api/problems/{id}` has no `judge` block; a `learner`-role account with judge configured → the same; `owner`/`tester` → routes live.
- **Close/give-up:** give-up with an in-flight pass below `close_seq` (practice fake) → the composed view shows the pass; same `Idempotency-Key` twice → one judge submission, same response; `action: final` or `give_up` on `POST /api/problems/{id}/submissions` → 422 `use_close_route` and **no** judge call; Retry grading (`close {final}` from `self_grade_pending` with `retryUsed=false`) → judge close + practice close; a second one → 409 `retry_used` with no judge call.
- **Limits:** 64 KiB + 1 byte of code → 413 `part_too_large`; 1 MiB + 1 → `body_too_large`; each judge 429 passes code, `details` and `Retry-After`; L16 integration tests on real Postgres (two concurrent starts → one wins; the 4th start in a UTC day → 429).
- **Compose e2e** (`internal/e2e`, judge + runner + fixture pack): start → Run → Submit WA → Submit pass → the poll reaches `factsThroughSeq ≥ attemptSeq` and `concluded`; give-up path; the denylist walk against the real judge.
  - **Where it runs:** the real-runner legs use [m3-06](sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner up`; build tags `e2e,runner`) and run on Linux in CI's **`judge-runner-e2e`** job (m3-06). The runner can't run on macOS: locally, run the same legs against the fake runner (`internal/judge/runnerfake`, the default `e2e` lane), or use the spike's multipass VM, and let CI's Linux job be the acceptance.
- **Docs:** [`docs/architecture/api.md`](../../architecture/api.md) and `openapi.yaml` (every new route, the error codes, `quota`, the ETag/`pollAfterMs` contract; the drift test enforces it); [`services.md`](../../architecture/services.md) (gateway → judge, `aud=judge`, presence by config; the status/evaluable caches join the L24 single-replica list).

## Acceptance criteria

- [ ] **Denylist test green**: no hidden input, expected output, per-case timing/ordinal, test source, `signals`, `versions` or pack path in any BFF response, for every judge route and state (poisoned fake **and** the compose fixture pack).
- [ ] **IDOR tests:** another account's submission, Run, draft, arena history or attempt close → 404, identical to an unknown id.
- [ ] **Typed 413/429 surfaced to the SPA:** `part_too_large{max: 65536}`, `body_too_large`, `quota_exceeded`, `queue_full`, `too_many_pending`, `too_fast`, `budget_exhausted`, `shed` with `Retry-After`; `quota.remaining` on every 2xx.
- [ ] L16: ≤ 1 live classic mock per course (409 `mock_in_progress`) and ≤ 3 starts per UTC day (429 `mock_daily_limit`), race-safe.
- [ ] Presence by config: unset `JUDGE_BASE_URL` or a non-cohort account → every judge route 404 and the v1 Problem flow unchanged.
- [ ] Close/give-up is gateway → judge → practice, synchronous and idempotent, and is the **only** close path (`final`/`give_up` on the submissions route → 422 `use_close_route`); Retry grading from `self_grade_pending` works once; the poll view carries `retryUsed` and `pendingReason`; the attempt start sends `{judge: true}` only for the cohort and 409s ahead-of-schedule items; a counted submit after `deadline_at + 2 s` → 409 `attempt_expired`.
- [ ] `GET /api/judge/status` reports `off` (`admission: closed` or judge unreachable), `breaker`, `saturated`, `evaluable: 0` and per-item `contract_mismatch` / `unsupported` / `invalid` / `no_pack` from judge's `/internal/status` + `/internal/evaluable` (fakes + compose with `JUDGE_ADMISSION=closed`).
- [ ] Arena Submit/History/Copy work during a live attempt and a due touch (D17); the arena reveal is recorded once and never caps.
- [ ] CI green: `go test ./...`, `sqlc diff`, web tests, OpenAPI drift, route enumeration, the compose e2e (real-runner legs in the Linux `judge-runner-e2e` job, `--profile runner`).

## Release

**Merge only: ships dark in `v1.14.0`**, cut by [m3-13](sprint-m3-13.md) after M3-11 and M3-12 merge. Do **not** tag.
- **No infra PR.** judge's ingress from the gateway is [m3-07](sprint-m3-07.md)'s policy; the gateway's egress rule to judge :8087 is forward-declared by [mi-11](sprint-mi-11.md) (MI-15); `JUDGE_BASE_URL` on `apps/xlearn-gateway.yaml` is [m3-13](sprint-m3-13.md)'s infra PR **after** the tag (T-2). No new pod, so the memory sum is unchanged.
- While `JUDGE_BASE_URL` is unset on prod, every route here is a 404 and nothing changes for anyone.

## Definition of Done

CI green (incl. `sqlc diff`, OpenAPI drift, route enumeration, the denylist and IDOR tests) · squash-merged to `main` (no tag) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board row; M3 stays 🔄; the **flag inventory** gains the judge cohort gate (T-3, owner M3, removal GA via [ga-01](sprint-ga-01.md)) beside the permanent `JUDGE_BASE_URL` kill switch) · Decisions-log lines for the L16 definitions, the arena-reveal rule, the ahead-of-schedule, `attempt_expired`, `use_close_route` and `retry_used` codes, the status mapping (`admission: closed` → `off`; `spec_mismatch` → `contract_mismatch`), the `pendingReason` derivation, and every additive practice read field (`GET /state/{problemId}`, `GET /attempts/open`).

## Risks / watch-outs

- **SSE through Traefik buffering.** Don't add SSE here; polling with ETag is the default and the only transport in v2.0.
- **A judge field leaking through a pass-through.** Never `passthrough()` a judge body: always decode into the allowlist structs. The poisoned-fake test must cover every route, including errors (judge's 4xx `details`).
- **Context ids from the client.** The SPA never sends one; a body `context.id` is ignored (test it). Arena ids are account-scoped.
- **The close's two writes.** Judge's partial unique index plus practice's idempotency on `submission_id` make a retried close safe; don't add a gateway-side lock.
- **D17 vs t4's arena locks.** t4 §3.7, §8 and §11.4 #20 still describe locks; D17 wins (arena unrestricted, reveal recorded, no cap). Don't port the locks.
- **L16 and the owner's own mocks.** An abandoned unscored session blocks only until its deadline; confirm in the Decisions log.
- **Cache churn from polling.** Invalidate only on a 200 poll showing lock/conclusion, not on every poll.
- **Rebase churn with M3-10** (assessment sqlc output, `internal/gateway/*.go`): rebase late, re-run `sqlc generate`, keep the diff focused.
- **Touching practice from a gateway sprint.** Only additive **practice** read fields are allowed (missing `GET /state/{problemId}` fields incl. `retry_used`/`pending_reason`, and the touch entry's `band`/`deadlineAt` on `GET /attempts/open`); each lands with practice's DTO golden updated. **Judge needs no edit** (`quota` is already on its 2xx, `/internal/status` already exists). Anything bigger is a gap in M3-08/M3-14: stop and report.
