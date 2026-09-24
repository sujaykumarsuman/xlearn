# Sprint m6a-04 — judge mock budgeting + CodeMirror interview mode + bounded SSE

> **Milestone:** M6a — text interviewer plus failsafes (T6 **P0**; v2.1 build, ships **dark** in a `v2.0.x` patch)   ·   **Track:** product   ·   **Order:** 77
> **Prereqs:** [m6a-01](sprint-m6a-01.md) (interview core: tables, FSM, L19 caps, the `/api/interviews/*` proxy and its cohort gate) · already on `main` since M3: [m3-06](sprint-m3-06.md) (judge's `mock` context), [m3-14](sprint-m3-14.md) (admission L9–L13, `job_telemetry`), [m3-09](sprint-m3-09.md) (judge BFF; `mock` refused until now), [m3-11](sprint-m3-11.md) (lazy `CodeEditor`, `judge.ts`, drafts autosave) · [m1-06](sprint-m1-06.md) (`withhold()`) · [m1-07](sprint-m1-07.md) (coach mode gate)
> **Unblocks:** [m6a-05](sprint-m6a-05.md) (the HUD and the classic session compose this sprint's editor mode, snapshot publisher, event stream, turn stream, drafts, classic start and Run routes) · [m6a-03](sprint-m6a-03.md)'s review consumes this sprint's judge evidence (this sprint **owns** `interview.judge_evidence` and the evidence intake; task 3)
> **Release action:** **merge only** (ships dark in the next `v2.0.x` patch, tagged by [m6a-06](sprint-m6a-06.md)) · no infra PR (no new in-cluster caller; task 7 records the check) · no new pod
> **Calendar:** Q1 2027, after v2.0.0; runs beside [m6a-02](sprint-m6a-02.md) / [m6a-03](sprint-m6a-03.md) as a **deliberate exception** to the register rule "siblings touching the same service are serialized by `depends_on`": all three add coach goose migrations, and m6a-03 and this sprint both edit assessment's `POST /mocks` handler and the evidence seam. Protocol: take the **next free goose version at rebase**; **adopt whatever seam merged first** and delete your own copy; never build a route or column twice. No owner time.
> **Execute with:** [`../prompts/prompt-m6a-04.md`](../prompts/prompt-m6a-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | judge: mock Runs paced (L19 ≤ 1 / 10 s), budget-counted but never blocked or shed; mock `final` as a P0 close; mock stays silent | X | ⬜ |
| 2 | Gateway mock item picker with `withhold()`; classic Mock-v2 start (gateway + assessment `POST /mocks`); `InMock` item state | X | ⬜ |
| 3 | Mock Run / drafts / final / evidence routes (interview + classic Mock-v2); run echo to coach | X | ⬜ |
| 4 | CodeMirror interview mode + snapshot publisher (web) | X | ⬜ |
| 5 | Snapshot route + coach snapshot store and timeline (C4) | X | ⬜ |
| 6 | Bounded SSE: gateway `relayBounded`, `events` + `turns` routes, coach event stream, web clients | X | ⬜ |
| 7 | Coach lock for interviews + NetworkPolicy check | X | ⬜ |
| 8 | Tests, e2e leg, docs, hand-offs | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> **M6a** milestone row, the Decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **m6a-01 merged** ([m6a-01](sprint-m6a-01.md)): `internal/coach/interview` with the FSM (injectable `Clock`) and its exported **named sets** (`running`, `locked`, `editable`, `live`, `terminal`); the `interview_event` log with `events.Append` (the SSE cursor); `interview.items` and the `CodeSource` checkpoint interface; the cap constants `SnapshotMinInterval`, `SnapshotMaxBytes`, `MockRunPace`; coach's local `/chat` lock; `GET /interviews/active` (`{id, state, items, coach_locked, editable, resume_by}`); the gateway's `/api/interviews/*` proxy with the **interview cohort gate** (`interviewAudience = "cohort"`, T-3) and its **placeholder item pick** in `POST /api/interviews`; `coach admin interviews --live`.
- [ ] **v2.0.0 live and judge on** ([ga-02](sprint-ga-02.md)): `JUDGE_BASE_URL` set on prod; judge's `mock` context ([m3-06](sprint-m3-06.md), dark since v1.13.0); m3-14's admission rows and `job_telemetry`; m3-09's BFF (which answers `mock` with 422 `invalid_context`; this sprint opens `mock` **only through its own interview- and mock-scoped routes** (task 3), and the generic `/api/problems/{id}/…` routes keep refusing `mock` with that 422); m3-11's `web/src/screens/workspace/CodeEditor.tsx` (lazy chunk), `web/src/lib/judge.ts` and drafts autosave.
- [ ] **AB25 frozen** ([ds-m6a-01](sprint-ds-m6a-01.md) merged): `design-system/screens/v2/AB25-live-hud-text.html` on `main` (the editor's interview-mode frame, the snapshot indicator and the read-only states come from it). m6a-01 already gated on AB13 and AB24–AB28.
- [ ] **Parallel sessions** (`gh pr list`, `git worktree list`, ListAgents): [m6a-02](sprint-m6a-02.md) and [m6a-03](sprint-m6a-03.md) may be open beside this sprint. No open peer PR edits `internal/judge/admission/**`, `internal/judge/dto/**`, `internal/gateway/withhold.go`, `internal/gateway/sse.go` or `web/src/screens/workspace/CodeEditor.tsx`. m6a-03 may have `internal/assessment/handlers.go` (`POST /mocks`) and the evidence seam open: follow the Calendar protocol (adopt what merged, delete your copy). Coach goose migrations: take the **next free version at rebase** (CI fails on a duplicate).

## Goal

Wire the **coding round** of a mock ([t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round)) and the transport the interviewer needs, all dark behind m6a-01's cohort gate:
- **judge:** mock Runs (started by the learner only, ≤ 1 per 10 s) and the mock `final` go through the one runner lane; they are **budget-counted but never budget-blocked or shed** by the breaker ([t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) "T6"), and mock evaluations stay **silent** (no `evaluation_completed`, [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract)).
- **Gateway:** server-picked mock items that skip anything `withhold()` marks live (no start refusal, D27), for interviews and for the cohort's classic Mock-v2 start; Run, drafts, final and **aggregate-only** evidence routes; the mock's items are withheld as live from the moment they are revealed while the interview runs or is paused.
- **Web + coach:** CodeMirror 6 in **interview mode** and a **snapshot publisher** that shares the editor with the interviewer on change every 2–3 s and on every turn (D29), bounded at 64 KiB and ≤ 1 per 2 s ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L6, L19), with a C4 snapshot timeline beside the transcript.
- **Bounded SSE:** interview streams through the gateway carry first-byte, idle, max-duration and size bounds; the event stream is ≤ 50 s with `Last-Event-ID` resume from a DB sequence ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)).
- **Coach lock:** coach is `locked` in the in-session interview states and unlocked while paused ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) "Coach lock").

## Scope

**In**
- `internal/judge/admission`: the mock-Run branch (L19 pace; L12/L13 exemptions; telemetry counted), mock `final` as a P0 close; `internal/judge/dto` mock cases; the silence assertion.
- `internal/gateway`: `mockpick.go` (picker), the classic Mock-v2 start on `POST /api/mocks` (v2 body, cohort only), the `InMock` item state in `withhold.go`, mock Run / drafts / final / evidence routes, the run echo, `sse.go` (`relayBounded` + per-account stream caps), the `events` and `turns` interview routes on the bounded relay (and a bounded client timeout on m6a-02's JSON `brief` proxy), the snapshot route, the coach-lock derivation for text/voice mocks.
- `internal/assessment` (no migration): `POST /mocks` accepts `format: "classic"` with picked `items[]` (task 2); `GET /mocks/live` returns the session's items (additive).
- `internal/coach/interview`: snapshot intake (pace, rev, unchanged), the director hand-off (`ScreenSink`), the `CodeSource` implementation, `interview_snapshot` timeline, `GET /interviews/{id}/events` over m6a-01's `interview_event` log, run echo and evidence intake (`interview.judge_evidence`, owned here), an additive `revealed` flag on `GET /interviews/active`.
- `web/`: `CodeEditor.tsx` interview mode; `web/src/lib/interview/{snapshots,events,turns,run,drafts}.ts`.
- Docs: `docs/architecture/openapi.yaml`, `api.md`, `services.md`, `data-model.md`, `events.md`; a dated amendment line in [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) (per-item mock context ids, task 3).

**Out**
- The screens: Mock-v2 hub, setup/consent/pre-flight/$ cap and the live HUD → [m6a-05](sprint-m6a-05.md); grace/paused/resume and debrief/proposal → [m6a-06](sprint-m6a-06.md). This sprint ships **library code and routes only**; nothing new renders outside tests.
- The brain loop, turn handler, classifier, probe, `give_hint`, resume brief, `store:false` → [m6a-02](sprint-m6a-02.md) (this sprint relays its streams with bounds and feeds its screen hook).
- Proposal, accept, `ScoreMock` once, twin gate, `mock_session.status/format` → [m6a-03](sprint-m6a-03.md) (this sprint supplies judge evidence).
- The FSM, `pause_exposure`, retention sweeper, erase coverage mechanics, caps other than judge's → [m6a-01](sprint-m6a-01.md) (this sprint's new tables join its retention and erase).
- Turning mock `evaluation_completed` on — not needed ([t4 §11.3](../research/t4-judge-contract.md#113-t6-and-t7): only if a consumer needs it; none does).
- Voice: the SDP route ([mi-13](sprint-mi-13.md)), captions over the event stream ([m6b-01](sprint-m6b-01.md)), "hold voice", leases and make-before-break ([m6b-02](sprint-m6b-02.md)).
- Bounding the existing `/api/coach/chat` relay (it keeps `relayStream`; a follow-up note, task 8).

## Tasks

### 1 · judge: mock Runs and finals [X]

Sources: [t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments) ("counted submits, mock and P0 are never budget-blocked"), [t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) (mock never shed, budget-counted, ≤ 1 per 10 s; Run p50 < 5 s), [t4 §2.3](../research/t4-judge-contract.md#23-admission) row 6, [§9.2](../research/t4-judge-contract.md#92-priorities-and-caps), [§3.7](../research/t4-judge-contract.md#37-touches-mocks-arena), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L9–L13, L19.

**Read what merged first.** [m3-14](sprint-m3-14.md)'s admission table already names mock in rows 6d ("never budget-blocked") and 6e ("level 2: only P0 closes and P1 counted submits and mock run"), but no learner route could exercise it. Keep what is right, add what is missing, and pin it with tests. The target behaviour:

| Limit ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)) | Course / touch / arena Run | **Mock Run** | **Mock `final`** (a close) |
|---|---|---|---|
| Priority ([t4 §9.2](../research/t4-judge-contract.md#92-priorities-and-caps)) | P1 | **P1** | **P0** |
| L9 daily quota (200 Runs) | applies | **applies** (t3: "T4 quotas … bound them") | exempt (closes are exempt from row 6) |
| L10 lane cap (32 queued) | 429 `queue_full` | applies (a physical cap, not shedding) | exempt |
| L11 pending (≤ 2 queued Runs across contexts; a new Run cancels queued Runs in its context) | applies | applies | exempt |
| L12 runner budget (10 / 60 slot-min) | 429 `budget_exhausted` | **never blocks**; its `busy_ms` **counts** toward the account's sums | never blocks; counts |
| L13 level 1 (Runs ≤ 1 / 30 s) | 429 `too_fast{reason: breaker}` | **exempt** (the L19 pace applies instead) | exempt |
| L13 level 2 (shed) | 429 `shed{reason: breaker}` | **never shed** | never shed |
| **L19 mock Run pace** | — | ≥ 10 s since the account's previous mock Run (any `mock` context) → **429 `too_fast{reason: "mock_pace", retry_after_ms}`** + `Retry-After` | — |

- Implement the pace as a Postgres read under m3-14's per-account advisory lock (the newest `job` or `job_telemetry` row with `context_kind = 'mock'` and kind `run` for the account; no in-process counter — judge may restart). Add `MockRunPace` (default 10 s) to m3-14's `admission.Limits` with its env override (`JUDGE_LIMITS_MOCK_RUN_PACE`, following the naming that merged).
- **"Budget-counted":** mock Runs and finals write `job_telemetry` rows with `context_kind = 'mock'` like any runner job, so they count in the account's L12 sums and in the L13 busy %. A course Run after a heavy mock can therefore meet `budget_exhausted` — intended ([t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream)).
- **Mock `final`:** `action = final`, `closes_context = true`, P0; **parts may be empty** — judge fills every missing part from the context's drafts (the close semantics of [t4 §2.1](../research/t4-judge-contract.md#21-vocabulary)); the gateway's idempotency key replays it ([t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers), "200 with the existing close"). Row 5 already keeps `ai_rubric` out of mock and stores the submission as the mock answer even when every step is skipped.
- **Silence:** the result transaction writes **no** `xlearn.judge.evaluation_completed` outbox row for `context_kind = 'mock'` ([ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract); m3-06's rule). Add the explicit assertion (task 8); `topology.go` and the subject registry are unchanged.
- **DTO** (`internal/judge/dto`, m3-14's allowlist): a mock Run returns the ordinary Run DTO (samples and custom input, the learner's own output only). A mock `final` evaluation returns **aggregates only**: `hidden{passed,total}`, `perf`, `first_failure.class`, and for `key` steps an aggregate score unless the learner has already concluded that item ([t4 §2.6](../research/t4-judge-contract.md#26-evaluation), [§3.7](../research/t4-judge-contract.md#37-touches-mocks-arena)). No case ids, stderr, timings or `answer_reveal`.
- **On-demand read (D34):** `judge admin status` gains "mock Runs / finals, last 24 h" and the mock share of busy % — a read, not an alert.

### 2 · Mock item picker, classic Mock-v2 start + `InMock` [X]

Sources: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) ("Mock item selection: skip items that `withhold()` marks live … no start refusal (D27)"; "while paused … `withhold()` treats the mock's item as live"), [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) ("server-picked items, replacing `PROBLEM_SETS`"), [t4 §2.1](../research/t4-judge-contract.md#21-vocabulary) (the mock context holds the item; `contract_hash` from the `mock_session_item` pin), [m1-06](sprint-m1-06.md) task 1.

**`internal/gateway/mockpick.go`** — `g.pickMockItems(ctx, accountID, course, req) ([]pickedItem, error)`:
- **Request:** `{format: classic|text|voice, difficulty: easy|med|hard|mixed, language?}`. **Count** from the course manifest's `mock` block if it carries one (e.g. `mock.selection.count`), else **1** for DSA's 45-minute rail; clamp 1–3. If m6a-01's manifest work added no selection field and you need one, add `mock.selection{count, pool}` additively (manifest golden updated) and record it.
- **Pool, in order** (deterministic; a seeded shuffle with seed = sha256(account ‖ interview id, or the classic start's `Idempotency-Key`) so tests are stable): (1) live items of the chosen difficulty at or below the learner's frontier week that weren't solved in the last 14 days; (2) any live item at or below the frontier; (3) any live, non-retired item of the course. `preview` courses only inside the cohort ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)).
- **Evaluable first:** for text/voice and for classic Mock-v2, prefer items judge reports `evaluable` for the chosen language (m3-09's cached `GET /internal/evaluable?path=`). If judge is absent (kill switch or unset) or nothing is evaluable, pick from the rest and mark the item `runs: false` — the HUD hides Run and the review works from transcript and code.
- **Exclude live items** with m1-06's per-request `g.itemStates` (open counted attempt, due or live touch, and `InMock` below). If item states can't be read, **fail closed**: 503 `mock_state_unavailable` (the SPA retries) — never pick blind.
- **Pin:** each pick carries `contract_hash` = the pack's accepted hash at pick time (the same pin rule practice uses at attempt start). Interviews keep the pins in coach's `interview.items` (m6a-03 copies them into assessment's `mock_session_item` at start); classic sessions pass them to assessment's `POST /mocks`, which writes `mock_session_item(session_id, ordinal, item_id, path_slug, contract_hash)` ([m1-02](sprint-m1-02.md)). The gateway never writes assessment's tables ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).
- **S1:** the pick is **never echoed to the client before the mock starts** (classic) or the interview is `live` (text/voice); setup views show count, difficulty and duration only.
- **Wiring (interviews):** replace m6a-01's **placeholder pick** in `POST /api/interviews` (the first evaluable item of the difficulty) with `pickMockItems`; the picked `{ordinal, item_id, contract_hash, runs}` go into coach's create call as `items` (additive fields on m6a-01's `items jsonb`), and [m6a-03](sprint-m6a-03.md) copies them into `mock_session_item` when it links the `format=text` session at start.

**Classic Mock-v2 start** (the route [m6a-05](sprint-m6a-05.md)'s `ClassicSession` calls):
- **Gateway** `POST /api/mocks` gains a **v2 body** `{format: "classic", difficulty, language?}` for cohort accounts (outside the cohort a v2 body → 404 `apiNotFound`, like every gated route). `Idempotency-Key` required (UUIDv7, m3-11's helper). Flow: `pickMockItems` → assessment `POST /mocks {format: "classic", difficulty, items[{ordinal, item_id, contract_hash}], client_ref}` (`client_ref` = the `Idempotency-Key` once m6a-03's column exists) → `201 {mockId, deadlineAt, count}`. **No item ids or statements in the response** (the client reads them from `GET /api/mocks/{id}` after Start, S1). The per-item `runs` flag is not stored: `GET /api/mocks/{id}` and the classic Run route derive it from judge's cached `GET /internal/evaluable` read (m3-09). v1's body `{setId, problemId}` is **unchanged** for every account (v1 `Mock.tsx`).
- **Assessment** (`internal/assessment/{handlers.go,mock.go}`, no migration): `POST /mocks` accepts `format: "classic"` with `items[]` → `status = 'live'` (m6a-03 keeps `live` for classic), `deadline_at = started_at + the manifest's mock rail minutes` (server-authoritative for classic, as v1), `rubric_snapshot` from the manifest, one `mock_session_item` per pick with its pin; the v1 `set_id` / `problem_id` columns follow m1-02's convention for itemised sessions (read what merged). It runs inside `CreateMock`'s transaction, so **L16 applies unchanged** ([m3-09](sprint-m3-09.md) task 6: ≤ 1 live classic mock per course → 409 `mock_in_progress{mockId}`, ≤ 3 starts per UTC day → 429 `mock_daily_limit{resetAt}` + `Retry-After`); a retried start without m6a-03's `client_ref` meets the 409 and the SPA resumes that session. `GET /mocks/live` returns the live session's `items[{ordinal, item_id}]` (additive) for `InMock`.
- **Seam with m6a-03:** m6a-03 extends the same handler with `items[]` for `format: text|voice` and keeps the classic path unchanged. Whichever merges **second** folds both into one `items[]` path (one decoder, one `mock_session_item` writer) and removes the other's copy.

**`InMock`** in `internal/gateway/withhold.go`:
- `itemState` gains `InMock bool`; `live(s)` becomes `OpenAttempt || DueTouch || LiveTouch || InMock`. In the rule table `InMock` behaves exactly like `OpenAttempt` (lists, course-workspace stage gating, coach context strip). The arena is unchanged (D17; arena exposure is recorded by m6a-01's `pause_exposure` path).
- **Source:** coach's `GET /interviews/active` (`items`, `state`, and a new additive **`revealed`** flag) and the account's live classic mock (assessment `GET /mocks/live` with its `items`, [m1-07](sprint-m1-07.md)), read once per request in the existing fan-out and cached per account in process for 15 s (like `aggCache`; single replica, L24), invalidated by the gateway's own interview and mock routes. Fail closed as m1-06 does.
- **When:** only **once the item has been revealed** and until `finished`: `revealed` (true once the interview has reached `live`, i.e. its first `segment_open`; the same predicate gates `items` on `GET /interviews/{id}`) **and** the state is in m6a-01's **`live` set or `paused`**. In practice: `live`, `held`, `bridging`, `interrupted`, `paused`, `resuming`, `wrapping`, and a re-entry `connecting` after a resume. **Never** in `setup`, `preflight` or the first `connecting`: withholding there would reveal which item was picked (S1). This is narrower than m6a-01's exposure window (which may start earlier because recording exposure reveals nothing) and matches t6 §4 ("while paused, `withhold()` treats the mock's item as live"). A classic mock's items are `InMock` while it is `live` (the pick is revealed at Start).

### 3 · Mock Run, drafts, final and evidence routes [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (route list), [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (Runs by the learner only; the model has no run tool; coach gets a compact echo; only aggregate evidence reaches the proposal), [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers), [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock), [m3-09](sprint-m3-09.md) tasks 2–3 (the patterns to copy).

**Context id.** [t4 §2.1](../research/t4-judge-contract.md#21-vocabulary) and [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) name **`mock_session_id`** as the mock context id, and judge allows **one unreleased close per `(context_kind, context_id)`** ([t4 INV-13, §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers)).
- **Base = `mock_session_id`** for interviews and classic Mock-v2 alike. An interview's is linked by [m6a-03](sprint-m6a-03.md)'s start orchestration (assessment `POST /mocks` → coach `start {mock_session_id}`) **before** `preflight`, so every `editable` state has it; Runs, drafts and finals only happen there.
- **Pre-link fallback (build order only):** while m6a-03 is not on `main`, `mock_session_id` is NULL and the base falls back to the **interview id** so this sprint's tests and e2e run. Whichever of the two sprints merges **second** removes the fallback, so on prod (both ship dark in the same patch) the base never changes mid-interview.
- **Per item:** a multi-item mock needs one close per item, so read m3-06's close guard. If it is per context → **per-item ids `uuidv5(base, "mock-item:" || ordinal)`**; if it is already keyed per (context, item) → the base itself.
- The id stays **gateway-resolved** (ADR-0029 §2), but the per-item rule changes what t4/t6 readers expect. Add a **dated amendment line to [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract)** ("mock `context_id` = `uuidv5(mock_session_id, "mock-item:"‖ordinal)` per item; m6a-04, <date>"), and record it in `api.md` and the Decisions log.

**Resolution (never from the body):** for an interview, coach's `GET /interviews/{id}` — owned by `sub`, state in m6a-01's **`editable`** set (else **409 `interview_not_live{state}`**) — and its `items[ordinal]` with the picker's `contract_hash` and `runs`, plus its `mock_session_id`; for a classic session, assessment's `mock_session_item` (owned by `sub`; session `live` (m6a-03 keeps `live` for classic) and before `deadline_at`, else **409 `mock_not_live`**; item present) and its `contract_hash`.

| BFF route (cohort-gated; 404 = `apiNotFound` outside it) | Upstream | Notes |
|---|---|---|
| `POST /api/interviews/{id}/run` | coach state check → judge `POST /runs` (context `mock`) | body `{itemOrdinal, language, parts[], runInput?}`; `Idempotency-Key` required; `httpx.ReadBody` 1 MiB, code ≤ 64 KiB per part (413 `part_too_large`), run input ≤ 64 KiB; 202 `{runId, pollAfterMs, quota}`; typed 429s passed through, incl. `too_fast{reason:"mock_pace"}` |
| `GET /api/interviews/{id}/runs/{runId}` | judge `GET /runs/{id}` | ETag → 304; 800 ms then 400 ms ([t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8)); on the **first terminal** DTO, post the echo to coach (below) |
| `POST /api/mocks/{id}/items/{ordinal}/runs` | judge `POST /runs` (context `mock`) | classic Mock-v2 (AB13); no coach; poll with m3-09's `GET /api/runs/{id}` |
| `GET /api/interviews/{id}/items/{ordinal}/draft?language=` · `GET /api/mocks/{id}/items/{ordinal}/draft?language=` | judge `GET /drafts?context_kind=mock&context_id=<resolved>&item_id=` | the learner's own draft for that item and language; readable from reveal until terminal (interview) or while the classic session exists (so read-only frames show the last code) |
| `PUT /api/interviews/{id}/items/{ordinal}/draft` · `PUT /api/mocks/{id}/items/{ordinal}/draft` | judge `PUT /drafts/mock/{context_id}/{part_id}` | `{partId, language, payload}`; `httpx.ReadBody` 1 MiB, ≤ 64 KiB per part (413 `part_too_large`); no `Idempotency-Key` (last write wins, as m3-09); writes only in the `editable` set (else 409 `editor_read_only{state}`) or a `live` classic session (else 409 `mock_not_live`) |
| `GET /api/interviews/{id}/evidence` · `GET /api/mocks/{id}/evidence` | ensure finals → judge `GET /submissions/{id}` | 202 `{pollAfterMs}` while any final is pending; 200 `{items:[{ordinal, itemId, state: ready\|pending\|unavailable, reason?, hidden?{passed,total}, perf?, firstFailureClass?, keyAggregate?, hints[]}]}` |

The classic start (`POST /api/mocks`, v2 body) is task 2's. Every row: cohort-gated, owner-scoped (a foreign id → 404, the IDOR tests in task 8), and declares its `withhold` policy for m1-06's route-enumeration test (the draft routes carry only the learner's own code, no item content). The generic `/api/problems/{id}/drafts|runs|submissions` routes keep answering `mock` with **422 `invalid_context`**: the client never names a context or a context id.

- **`g.ensureMockFinals(ctx, account, base, items)`** — for each item: judge `POST /submissions {context mock, action final, closes_context: true, parts: []}` with `Idempotency-Key = uuidv5(base, "final:" || ordinal)` (replays return the existing close). Called from (a) the gateway's interview `finish` proxy after coach accepts the transition (m6a-01's route: add the call), (b) the two evidence routes, and (c) [m6a-03](sprint-m6a-03.md)'s review trigger. A rail-done or $-cap transition happens inside coach; the client sees `state: wrapping` on the event stream and fetches evidence, which ensures the finals. An item with no draft → `unavailable{reason:"no_code"}`.
- **Evidence** is the allowlisted aggregate only ([t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) "At Finish"): hidden passed/total, perf bit, first-failure class, key aggregate; `hints[]` are the manifest's `mock.evidence_hints` suggested caps when the course defines them ([t4 §6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock); DSA may define none). **No pack material, never.**
- **To coach:** as soon as every final is terminal (whichever path ensured them), the gateway posts `POST /interviews/{id}/evidence` (aud=coach); coach stores it in **`coach.interview.judge_evidence jsonb NULL`** (expand column; purged and erased with the interview) for [m6a-03](sprint-m6a-03.md)'s review, which runs after `ensureMockFinals`. **This sprint owns the column and the intake.** m6a-03 builds them only if it merges first (its conditional); in that case reuse them and add nothing.
- **Run echo:** `POST /interviews/{id}/run-echo {runId, itemOrdinal, compileOk, passed, total, firstFailureClass?}` — the learner-visible subset only (S1, [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)). Coach dedupes on `runId`, bumps m6a-01's `interview.run_count` / `last_verdict_class`, appends a `run` event (task 6), and calls [m6a-02](sprint-m6a-02.md)'s `OnRun` hook (the director decides whether the result is worth probing). A client that never polls to terminal leaves no echo — acceptable (the review uses the final's evidence).
- **Degradation:** judge absent or 503 `evaluation_unavailable{fallback}` → passed through; the picked item is `runs: false` and the HUD hides Run.

### 4 · CodeMirror interview mode + snapshot publisher [X]

Sources: [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (interview mode, no autocomplete; snapshots), [t6 §13 D29](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (on change sampled every 2–3 s, never re-sent unchanged, and on every turn; this overrides §5's ≥ 20 s debounce), [ADR-0032 §3](../../adr/0032-realtime-ai-mock-interviewer.md), AB25, [m3-11](sprint-m3-11.md) task 3.

**`web/src/screens/workspace/CodeEditor.tsx`** (stays in the lazy chunk; m3-11's bundle check must still pass):
- new props `mode: "course" | "interview"` (default `course`, behaviour unchanged), `readOnly?: boolean`, `onDocChange?(text: string, cursorLine: number)`;
- **interview mode:** no completion popups or snippets; bracket closing and other assists as AB25's behaviour notes say (default off); lint diagnostics only from Run results; keymap `Mod-Enter` → Run and **no Submit binding**; m3-11's Esc-then-Tab escape kept;
- `readOnly` through `EditorState.readOnly` + `EditorView.editable` compartments, switchable without remounting (the grace and paused frames show a read-only editor, t6 §4).

**`web/src/lib/interview/snapshots.ts`** — `useSnapshotPublisher({interviewId, itemOrdinal, language, getDoc, enabled})`:
- a **2.5 s sampler** that sends only when the text's sha-256 changed since the last accepted snapshot;
- `flush(reason)` for `turn` (called before each candidate turn is sent), `run`, `phase`, `finish` and `hidden` (`visibilitychange → hidden`, which also pauses sampling until visible);
- **coalescing:** never two requests in flight and ≤ 1 per 2 s — the newest pending text replaces an older queued one;
- a per-interview monotonic `rev` (seeded from the server's `revCurrent`);
- pre-check the encoded body ≤ 64 KiB → expose `tooLarge` (AB25's "too large to share" state) instead of sending; 429 → honour `Retry-After`; 409 `editor_read_only` → stop until re-enabled;
- expose `{state: shared|sharing|stale|tooLarge|off, lastSharedAt}` for m6a-05's snapshot indicator.

**Canonical draft:** the judge `mock`-context draft stays canonical ([t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round)). **`web/src/lib/interview/drafts.ts`** — `getMockDraft` / `putMockDraft` for both scopes (`{kind: "interview" | "classic", id, ordinal}`) on task 3's draft routes; the client sends no context and no context id. Reuse m3-11's autosave logic (15 s debounce, the "Saved · 12:03" indicator, last write wins) with this transport, flushed on Run, turn, phase change, blur, hidden and Finish; a 409 `editor_read_only` / `mock_not_live` stops autosave until re-enabled. The final's parts come from these drafts (task 3).

### 5 · Snapshot route + coach snapshot store and timeline [X]

Sources: [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (`PUT …/code {rev, text, cursor_line}`, ≤ 1 / 2 s, ≤ 64 KiB, `MaxBytesReader`; coach keeps the latest in memory and persists every 30 s, at each phase boundary and at SIGTERM), D29 (a snapshot timeline stored with the transcript, C4, same retention and erase; the final analysis uses the on-screen state at each point), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L6 (interview snapshot 64 KiB), L19.

**Gateway** `PUT /api/interviews/{id}/code` `{rev, itemOrdinal, language, text, cursorLine, reason}`: `httpx.ReadBody(w, r, 64<<10)` → typed **413 `body_too_large{max: 65536}`**; forward to coach. Nothing is logged but ids and sizes.

**Coach** `PUT /interviews/{id}/code` (aud=coach, owner-scoped):
- state check: edits only in m6a-01's **`editable`** set (`live`; `held`/`bridging` in voice) → else **409 `editor_read_only{state}`**;
- **pace** `SnapshotMinInterval` (≤ 1 / 2 s per interview; the last-accepted time in memory, rev persisted) → **429 `too_fast{retry_after_ms}`**; `SnapshotMaxBytes` (64 KiB) re-checked on the decoded text;
- stale `rev` → 200 `{accepted: false, revCurrent}`; same sha as the latest → 200 `{accepted: true, unchanged: true}` (not stored, not re-sent to the director);
- else keep it as the interview's in-memory latest and hand the diff to the director through **`interview.ScreenSink.OnScreen(ctx, id, ScreenUpdate{rev, itemOrdinal, diff, cursorLine, reason})`** (m6a-02 defines it if it merged first — adopt that definition; otherwise define it here and m6a-02 adopts it). A `snapshot` event (rev and sha only, never text; task 6) is appended **only when a timeline row persists** (below), in the same transaction, so a 45-minute interview writes ≈ 150 snapshot events (turns, phases, 30 s ticks) instead of up to ~1,350 (one per 2 s); the live indicator comes from the PUT responses, not the event stream.
- **`CodeSource`:** implement m6a-01's checkpoint interface from the latest snapshot (code SHA + ≤ 16 KiB head), so phase, interrupt, pause, SIGTERM and finish checkpoints carry the code instead of NULL.

**Timeline** (next free coach goose version; sqlc in `internal/coach/store/queries/interview_snapshot.sql`; `sqlc diff` clean):
`coach.interview_snapshot(interview_id uuid REFERENCES coach.interview ON DELETE CASCADE, account_id uuid NOT NULL, seq int, rev int, item_ordinal int, reason text CHECK (reason IN ('turn','phase','periodic','interrupt','finish','shutdown')), turn_seq int NULL, phase text, kind text CHECK (kind IN ('full','diff')), body text, sha256 bytea, bytes int, created_at timestamptz DEFAULT now(), PRIMARY KEY (interview_id, seq))`.
- **Persist** (not every 2–3 s tick): the snapshot each candidate turn saw (`turn`, with `turn_seq`), every phase boundary and interruption (`phase`/`interrupt`, **full**), every 30 s if changed (`periodic`, diff), at Finish and in the SIGTERM drain (`finish`/`shutdown`).
- **Diffs** are line diffs against the previous persisted row (a small pure helper `internal/coach/interview/textdiff`), with a full copy at every phase boundary, so any point reconstructs from ≤ 1 full row + diffs. `SnapshotAt(interviewID, turnSeq)` returns the on-screen code at a turn — [m6a-03](sprint-m6a-03.md)'s review uses it (D29).
- **Cap:** ≤ 1 MiB of timeline per interview; past it only phase-boundary full copies are stored and the interview records `snapshot_capped` (a review caveat).
- **Retention and erase:** the table joins m6a-01's transcript retention (deleted with the transcript: 30 days after scoring by default, 12 months opt-in, per-session delete) and coach's erase transaction; l-01's `information_schema` coverage test turns red otherwise. Never in logs (the T5 canary covers `/api/interviews/*` bodies).

### 6 · Bounded SSE [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) ("`GET /api/interviews/{id}/events` bounded SSE ≤ 50 s, `Last-Event-ID` (DB seq)"; text turns use the chat relay pattern), `internal/gateway/coach.go` (`relayStream` clears the write deadline and has no maximum), `internal/platform/httpx/httpx.go` (`WriteTimeout` 60 s), [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L24 (single gateway replica).

**`internal/gateway/sse.go`:**
```go
type sseBounds struct {
    FirstByte, Idle, MaxDuration, Heartbeat time.Duration
    MaxBytes int64
    OnMax    string // "reconnect" (events) or "error" (turns)
}
func (g *Gateway) relayBounded(w http.ResponseWriter, r *http.Request,
    open func(ctx context.Context) (*http.Response, error), b sseBounds)
```
- the upstream context has deadline `MaxDuration` and is cancelled when the client goes away;
- the write deadline becomes `now + MaxDuration + 5 s` — **never cleared** (unlike `relayStream`);
- no upstream byte within `FirstByte`: before headers → **504 `upstream_timeout`** envelope; after headers → an `event: error` frame `{"error":"upstream_timeout"}`, then close;
- no upstream byte for `Idle` mid-stream → `event: error {"error":"upstream_idle"}`, close; a `: hb` comment every `Heartbeat` while waiting;
- `MaxDuration` reached → `OnMax=reconnect`: `retry: 1000` + `event: reconnect`; `OnMax=error`: `event: error {"error":"stream_too_long"}`;
- over `MaxBytes` → `event: error {"error":"stream_too_large"}`, close;
- `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`, flush per chunk;
- **per-account caps** (in-process, L24): ≤ 2 open `events` streams and ≤ 1 `turns` stream → **429 `too_many_streams`** + `Retry-After: 5`. Add the limiter's state to the scale-out-blocker list beside the cache epoch.
- **Injectable bounds:** each route's `sseBounds` is a field on the gateway (production values below as defaults), and coach's own 50 s stream end reads m6a-01's injectable `Clock`, so tests and the e2e cut a stream in about a second instead of waiting 50 s.

| Route | Upstream | Bounds |
|---|---|---|
| `GET /api/interviews/{id}/events` | coach `GET /interviews/{id}/events?after=<seq>` (`Last-Event-ID` header or `?after=`) | FirstByte 5 s · **Idle 35 s** (above coach's 15 s heartbeat, so a silent coach is cut) · Heartbeat 15 s · **MaxDuration 50 s** (inside the 60 s `WriteTimeout`) · 1 MiB · reconnect |
| `POST /api/interviews/{id}/turns` | coach's turn handler ([m6a-02](sprint-m6a-02.md); text ≤ 8 KiB, ≤ 1 in flight, ≤ 20 / min) | body ≤ 16 KiB (413) · FirstByte 30 s · Idle 30 s · MaxDuration 120 s · 256 KiB · error |

- If m6a-02 registered `turns` on `relayStream`, switch it to `relayBounded`. **No interview route clears the write deadline.** `/api/coach/chat` is untouched.
- m6a-02's **`brief`** is a structured JSON call, not a stream: give its proxy a **per-route 60 s client timeout** (a bounded `http.Client`, not the 10 s default and never unbounded). If it streams by the time you arrive, put it on `relayBounded` (FirstByte 30 s · Idle 30 s · MaxDuration 90 s · 64 KiB · error).

**Coach event stream** `GET /interviews/{id}/events?after=<seq>` (aud=coach, owner-scoped):
- streams **m6a-01's `interview_event` log** (monotonic `seq` per interview, `data` ≤ 4 KiB, S1-filtered, already in retention and erase);
- **kinds** (m6a-01's enum): `state` (FSM transitions, m6a-01), `phase` (rail cues), `turn` (turn id, role, final text — text the learner already sees; m6a-02), `hint` (only after `give_hint` recorded it, S4), `run` (the echo, this sprint), `snapshot` (rev, sha; this sprint), `cap` (85 % / 100 % of the $ cap), `notice` (the safety card, `lease_lost` → "live in another tab"). Writers use `events.Append(ctx, tx, id, kind, data)` inside their own transaction;
- **S1 filter:** payloads carry only what the learner may see now — never unreleased hint text, prompts, provider ids, key material or pack data; a test enumerates kinds against an allowlist;
- the handler replays `seq > after`, then waits on an in-process notifier (single replica) with a 1 s poll fallback; `id: <seq>` on every frame; `: hb` every 15 s; ends at 50 s (coach-side bound too, so a gateway bug can't hold it open).

**Web clients** (`web/src/lib/interview/`):
- `events.ts` — `useInterviewEvents(id, handlers)`: `EventSource` (same-origin cookie). It **tracks `lastSeq`** (the highest `id:` handled). The browser's native `Last-Event-ID` applies only to its own automatic retry of the same `EventSource`; every **manual** reconnect (the `reconnect` frame, handled at once; error back-off 1 → 2 → 5 → 10 s; the 35 s no-bytes watchdog; after a 429) closes it and opens a new one on **`?after=<lastSeq>`**, so a 50 s cycle never replays the log from the start. Frames are de-duplicated by seq; the hook closes on a terminal state and exposes `connection: live|reconnecting|offline` for AB25's banner.
- `turns.ts` — `streamTurn(id, body, onDelta, signal)`: fetch + reader like `streamCoachChat` (`web/src/lib/settings.ts`); typed errors (`too_many_streams`, `stream_too_long`, `upstream_timeout`, `upstream_idle`, `interview_not_live`, 413). The final turn text also arrives as a `turn` event, so a dropped stream loses nothing.
- `run.ts` — `runMockItem()` + `useMockRunPoll()` on the task-3 routes, reusing m3-11's poll helpers (800 → 400 ms, ETag, typed 429 with `Retry-After`).

### 7 · Coach lock + NetworkPolicy check [X]

Sources: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) "Coach lock" (locked in `connecting`, `live`, `held`, `bridging`, `interrupted`, `resuming`, `wrapping`; unlocked in `paused`), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 locked-mode amendment), [m1-07](sprint-m1-07.md) task 2, [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes).

- **One source of truth:** coach computes `coach_locked` from m6a-01's **`locked`** set (`connecting`, `live`, `held`, `bridging`, `interrupted`, `resuming`, `wrapping`), and m6a-01 already makes coach's `POST /chat` refuse **409 `coach_paused{reason:"mock"}`** while locked, before any key decrypt. This sprint adds the **gateway-side derivation** m6a-01 left here.
- **Gateway:** m1-07's mode gate reads assessment `GET /mocks/live` for `format = classic` only; for text/voice it uses `coach_locked` from `GET /interviews/active` (assessment's `deadline_at` isn't authoritative for them, [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) "The clock"). Locked → 409 `coach_paused{reason:"mock"}` without calling coach (AB01's "paused during mocks" copy, unchanged). `setup`, `preflight`, `paused`, `finished`/`proposed` and terminal states are unlocked; while `paused` the item is still `InMock`, so attempt-mode stripping applies (task 2).
- **NetworkPolicy standing rule** ([ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)): this sprint adds **no new in-cluster caller** — gateway → judge, gateway → coach and gateway → assessment already exist, and coach calls nobody new. Record "no NetworkPolicy PR needed" with that evidence in the Decisions log. (m6a-01 task 7 records its own NetworkPolicy outcome: none expected, since `pause_exposure` goes through the gateway.)

### 8 · Tests, e2e leg, docs, hand-offs [X]

- **judge** (real Postgres): every row of the task-1 table — seed `job_telemetry` to 45 % and 65 % busy: a mock Run is admitted at both levels while a course Run gets `too_fast{breaker}` / `shed`; a budget-exhausted account's mock Run is admitted and its `busy_ms` raises the account's sum; a second mock Run within 10 s → 429 `too_fast{mock_pace}` with `Retry-After`; a mock `final` with empty parts closes from drafts, is P0 and exempt from row 6; a replayed final returns the same close; **zero** `evaluation_completed` outbox rows after mock Runs and finals (passed and failed); the mock DTO denylist (no case ids, stderr, timings, `answer_reveal`; key aggregate only for unconcluded items).
- **gateway:** picker (difficulty, frontier, evaluable-first, `runs:false` fallback, live items excluded, fail-closed 503, deterministic seed, pin written, nothing echoed pre-start); the classic start (v2 body cohort-only, v1 body unchanged, pins reach assessment, no item in the response, L16's 409/429 passed through); `InMock` added to m1-06's route-enumeration sentinel sweep (an `InMock` item's sentinels appear on no applied route; **not** during `setup`/`preflight`/first `connecting`) and every new route declares its withhold policy; run/drafts/evidence routes (context resolution incl. the per-item id and the pre-link fallback, 409 `interview_not_live` / `editor_read_only` / `mock_not_live`, 413 at 64 KiB, cohort 404, **IDOR: another account's interview or mock id → 404**, typed pass-through, echo once, the generic `/api/problems/{id}/…` routes still 422 on `mock`); `relayBounded` with a fake upstream (first byte before/after headers, idle, max duration → reconnect frame, max bytes, client disconnect cancels upstream, heartbeat cadence, write deadline never cleared); stream caps; the snapshot route's 413; the lock derivation (classic vs text, paused unlocked).
- **assessment** (real Postgres): `POST /mocks {format: "classic", items}` writes the pins and `rubric_snapshot`, sets `deadline_at` from the rail, and meets L16 inside the same transaction; `GET /mocks/live` returns the items; a v1-shaped body still works.
- **coach** (real Postgres): snapshot pace, stale rev, unchanged no-op, persist triggers (turn, phase, 30 s, interrupt, finish, SIGTERM drain), a `snapshot` event only with a persisted row, the 1 MiB cap and `snapshot_capped`, `SnapshotAt` reconstruction; event-stream ordering, `after=` replay, 50 s end (injected clock), S1 kind allowlist; `revealed` on `GET /interviews/active`; `CodeSource` fills checkpoints; erase and retention include the new tables (coverage test).
- **web** (Vitest, fake timers): `CodeEditor` interview mode (no completions, `Mod-Enter` → Run only, read-only toggle); `useSnapshotPublisher` (2.5 s sampler, unchanged not sent, flush on turn, coalescing, 429, 64 KiB pre-check, hidden pause); `useInterviewEvents` (mock `EventSource`: reconnect frame, back-off, watchdog, **manual reconnects reopen on `?after=<lastSeq>`**, dedupe by seq, terminal close); `drafts.ts` (both scopes, no context id sent, 409 stops autosave); `streamTurn` errors; m3-11's bundle check still passes.
- **e2e** (`internal/e2e`, `//go:build e2e`, the core-loop harness: real Postgres, in-process services, fake provider, m3-06's fake runner): cohort start → the pick skips a live item → snapshot PUTs (pace, unchanged) → draft PUT → mock Run → the echo arrives on the event stream, resumed after a forced **MaxDuration** cut (bounds injected at ~1 s, never a real 50 s wait) without loss or duplicates → finish → the final fills from the draft → evidence aggregate only → no `evaluation_completed` row; plus a classic Mock-v2 start → draft → Run → evidence. Add the real-runner compose lane measurement: 20 mock Runs, **p50 < 5 s** ([t3 §10](../research/t3-sandbox.md#10-what-t3-constrains-downstream)).
- **Docs:** `docs/architecture/openapi.yaml` (drift test) and `api.md` (routes incl. the classic start and the draft routes, bodies, error codes, bounds, the context-id rule); the dated amendment line in ADR-0029 §2 (task 3); `services.md` (coach's event stream, echo and evidence intake; assessment's classic `items[]`; no new caller); `data-model.md` (`interview_snapshot`, `interview.judge_evidence`); `events.md` (mock stays silent).
- **Hand-offs** (Decisions log + the PR description): to [m6a-05](sprint-m6a-05.md) — the `web/src/lib/interview/*` APIs (incl. `drafts.ts`), the classic start body and the editor props; to [m6a-03](sprint-m6a-03.md) — the evidence seam and `SnapshotAt`; to [m6b-01](sprint-m6b-01.md) — captions ride the same event log (new kinds, same bounds); a follow-up note that `/api/coach/chat` could adopt `relayBounded`.

## Acceptance criteria

- [ ] **A mock Run returns within the runner SLO:** p50 < 5 s over 20 Runs on the real-runner compose lane; a mock Run is admitted at breaker levels 1 and 2 and with the account's budget exhausted while a course Run is refused; a second mock Run within 10 s gets 429 `too_fast{reason:"mock_pace"}`.
- [ ] Mock Runs and finals are **budget-counted** (their `busy_ms` is in the account's sums) and **never shed**; the mock `final` is a P0 close filled from drafts; **no mock evaluation emits `evaluation_completed`**.
- [ ] **Snapshots bounded:** > 64 KiB → typed 413; > 1 per 2 s → 429; unchanged text is neither stored nor re-sent; the timeline persists only on turn / phase / 30 s / interrupt / finish / shutdown, stays ≤ 1 MiB per interview, and is covered by retention and erase.
- [ ] The picker never picks a live item, pins `contract_hash`, and reveals nothing before start; `InMock` items are withheld on every applied route from reveal on, and never before it (sentinel sweep green).
- [ ] **Classic Mock-v2 startable:** a cohort `POST /api/mocks {format:"classic", difficulty}` creates an assessment session with picked, pinned items under L16; the v1 body is unchanged.
- [ ] **Mock drafts round-trip** on the interview and classic draft routes (gateway-resolved context, 64 KiB/413, writes refused outside the editable window, foreign ids 404), and a `final` with `parts: []` fills from them; the generic `/api/problems/{id}/…` routes still refuse `mock`.
- [ ] **Streams bounded:** events end at ≤ 50 s (Idle 35 s) and resume from `lastSeq` without loss or duplication; turns end at ≤ 120 s; per-account stream caps hold; no interview route clears the write deadline.
- [ ] Coach is locked in the in-session states and unlocked while paused (gateway and coach agree).
- [ ] Evidence reaching coach is aggregate only (DTO denylist green); the run echo carries only learner-visible fields.
- [ ] CI green: `go test ./...`, `sqlc diff`, the OpenAPI drift and route-enumeration tests, web typecheck/lint/tests/build, the bundle check, the e2e lane.

## Release

**Merge only.** It ships dark in the next `v2.0.x` patch, which [m6a-06](sprint-m6a-06.md) tags once the M6a UI is in ([rollout §7](../rollout-plan.md#7-indicative-tag-timeline): "v2.0.x … M6a/M6b dark"). The judge and assessment changes ride the same fleet tag (judge is `deploy.yml`'s 8th job). No infra PR, no new flag (the cohort gate is m6a-01's T-1 constant), no new pod, no event subject. Every new route is behind m6a-01's interview cohort gate until [m6b-04](sprint-m6b-04.md)'s v2.1.0 flip.

## Definition of Done

CI green (as above) · no infra change (GitOps untouched; the NetworkPolicy check recorded) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board, M6a row, Decisions log) · docs updated · local `main` synced after the squash-merge.

## Risks / watch-outs

- **Runner slots are shared with course work.** Mock is never shed, others are: a busy mock can push the breaker and shed arena and course Runs. Bounds: the ≤ 1 / 10 s pace, ≤ 1 non-terminal interview per account and ≤ 2 starts per day (L19). Watch the mock share in `judge admin status`; R0 tuning is an infra PR on `JUDGE_LIMITS_*` ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- **One close per context.** Multi-item mocks break on a per-context close guard; resolve the context-id question (task 3) before writing the final path.
- **Stream storms.** 2–3 s snapshots, a 50 s reconnect cycle and L5's per-account bucket (mutating 5/s) coexist; the publisher coalesces and the client backs off. A reconnect loop at 1 s would eat L5 — the back-off is tested.
- **Proxies buffering or cutting SSE.** 15 s heartbeats and `X-Accel-Buffering: no`; 50 s stays inside the 60 s `WriteTimeout` and Traefik's defaults. Verify the `events` stream through Traefik in the post-tag check ([m6a-06](sprint-m6a-06.md)).
- **Leaks through the event stream.** Every kind is S1-filtered and allowlisted by test; hint text appears only after `give_hint` records it.
- **Snapshot volume.** Only turn/phase/30 s rows persist, as diffs, capped at 1 MiB; the 2–3 s cadence lives in memory. Coach memory: one latest ≤ 64 KiB per live interview (≤ 1 per account); measure after the M6a tag.
- **Parallel sprints on the same seams** (the Calendar's deliberate exception to the register's serialization rule). m6a-02 (screen hook, turn handler), m6a-03 (assessment `POST /mocks` `items[]`, the pre-link fallback's removal) and this sprint may land in any order: next free goose version at rebase, adopt the interface that merged first, delete stubs, and never duplicate a route or column. `interview.judge_evidence` and its intake are this sprint's.
- **In-process state (L24).** Stream caps, the `InMock` cache and the event notifier assume one gateway and one coach replica; they join the scale-out-blocker list (the coach split trigger of [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) would revisit them). The notifier is only a wake-up (the stream always reads the log by `seq`), so [m6b-02](sprint-m6b-02.md) can swap it for Postgres `NOTIFY` when make-before-break runs two coach pods at once.
