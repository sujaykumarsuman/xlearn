# Prompt — Sprint m6a-04 · judge mock budgeting + CodeMirror interview mode + bounded SSE

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-04.md`](../sprints/sprint-m6a-04.md)   ·   **Milestone:** M6a (text interviewer, T6 P0; ships dark in a `v2.0.x` patch)   ·   **Prereqs:** [m6a-01](../sprints/sprint-m6a-01.md); on `main` since M3: [m3-06](../sprints/sprint-m3-06.md), [m3-14](../sprints/sprint-m3-14.md), [m3-09](../sprints/sprint-m3-09.md), [m3-11](../sprints/sprint-m3-11.md); [m1-06](../sprints/sprint-m1-06.md), [m1-07](../sprints/sprint-m1-07.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m6a-04.md`](../sprints/sprint-m6a-04.md). Its task-1 limits table, the route tables in tasks 3 and 6, and the acceptance list are authoritative for this session.
- [t6](../research/t6-realtime-interviewer.md): **§3** (routes, bounded SSE ≤ 50 s with `Last-Event-ID`, the 20 s/64 KiB/16 KiB sizes), **§4** (clock, "Coach lock", "Mock item selection", paused → item treated as live), **§5** (the coding round: interview mode, snapshots, Runs by the learner only, aggregate evidence at Finish), **§7** (S1–S4, logging), and **§13 D29** (2–3 s change cadence and every turn; it overrides §5's ≥ 20 s debounce).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §3–§6 (Accepted by [ds-m6a-01](../sprints/sprint-ds-m6a-01.md)); [ADR-0029 §2](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract) (contexts, gateway-resolved ids, mock silent); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L9–L13, L19, L24) and [§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (the NetworkPolicy standing rule).
- [t3 §7.4](../research/t3-sandbox.md#74-per-account-budgets-and-the-duty-cycle-breaker-t4-amendments) and [§10](../research/t3-sandbox.md#10-what-t3-constrains-downstream) "T6" (mock never shed, budget-counted, ≤ 1 / 10 s, Run p50 < 5 s); [t4](../research/t4-judge-contract.md) §2.1, §2.3, §2.4, §2.6, §3.7, §6.7, §9.2.
- The frozen board `design-system/screens/v2/AB25-live-hud-text.html` (editor interview-mode frame, snapshot indicator, read-only states, keyboard notes); [`design-system/theme.css`](../../../design-system/theme.css).
- Peer plans: [m6a-01](../sprints/sprint-m6a-01.md) (FSM and named sets, `interview_event`, `CodeSource`, cap constants, `GET /interviews/active`, cohort gate, placeholder pick), [m6a-02](../sprints/sprint-m6a-02.md) (turn handler, `ScreenSink`, `OnRun`, the JSON brief), [m6a-03](../sprints/sprint-m6a-03.md) (evidence consumer), [m3-14](../sprints/sprint-m3-14.md) (admission table), [m3-09](../sprints/sprint-m3-09.md) (BFF patterns), [m3-11](../sprints/sprint-m3-11.md) (editor, drafts, bundle check), [m1-06](../sprints/sprint-m1-06.md) (`withhold()`), [m1-07](../sprints/sprint-m1-07.md) (mode gate), [l-01](../sprints/sprint-l-01.md) (erase coverage test).
- Code: `internal/judge/admission/`, `internal/judge/dto/`, `internal/judge/` result tx; `internal/gateway/{bff.go,coach.go,withhold.go,assessment.go}` and m6a-01's interview proxy; `internal/assessment/{handlers.go,mock.go}` (`POST /mocks`, `GET /mocks/live`, L16 in `CreateMock`); `internal/platform/httpx/httpx.go` (`ReadBody`, `WriteTimeout` 60 s); `internal/coach/interview/`, `internal/coach/service.go`, `internal/coach/store/`; `web/src/screens/workspace/CodeEditor.tsx`, `web/src/lib/judge.ts`, `web/src/lib/settings.ts` (`streamCoachChat`, the SSE reader to copy); `internal/e2e/`; `docs/architecture/openapi.yaml`.

## Context

- M6a builds the **text interviewer** inside coach ([ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md)). [m6a-01](../sprints/sprint-m6a-01.md) built the durable core (tables, FSM, caps, cohort gate). [m6a-02](../sprints/sprint-m6a-02.md) and [m6a-03](../sprints/sprint-m6a-03.md) (brain; proposal and scoring) may be running beside you.
- **This sprint wires the coding round and the transport:** judge's `mock` Runs and finals (budget-counted, never shed, silent), server-picked items that skip live ones (interviews and the cohort's classic Mock-v2 start), mock-scoped draft routes, CodeMirror's interview mode, a snapshot publisher sharing the editor every 2–3 s and on every turn, and bounded SSE for the interview's event and turn streams. It also makes coach `locked` during in-session states.
- **Parallel by design:** this sprint runs beside m6a-02/m6a-03 as a deliberate exception to the register's serialization rule (shared coach migrations; assessment `POST /mocks` and the evidence seam with m6a-03). Next free goose version at rebase; adopt whatever seam merged first and delete your copy. This sprint owns `interview.judge_evidence` and the evidence intake.
- **Nothing renders yet:** you ship routes and library code; [m6a-05](../sprints/sprint-m6a-05.md) builds the screens on them. Everything is behind m6a-01's interview cohort gate (owner/tester) until v2.1.0.
- **Owner-decided (never contradict):** Runs are started by the learner only and the model has no run tool; the interviewer sees only what the learner sees (S1); hint text enters any context only through `give_hint` (S4); structured widget state, never screenshots (D29); no AI call on the learner's key without their action; no media, no alerting (D34).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **m6a-01 merged:** `internal/coach/interview` with the FSM (injectable `Clock`) and its named sets (`locked`, `editable`, `live`, …), the `interview_event` log + `events.Append`, `interview.items`, the `CodeSource` interface, the constants `SnapshotMinInterval` / `SnapshotMaxBytes` / `MockRunPace`, coach's local `/chat` lock, `GET /interviews/active`, the gateway's `/api/interviews/*` proxy + cohort gate (`interviewAudience`) with its placeholder pick, `coach admin interviews --live`.
- [ ] **v2.0.0 live, judge on:** `JUDGE_BASE_URL` set on prod; judge's `mock` context (m3-06); m3-14's admission rows + `job_telemetry`; m3-09's BFF answering `mock` with 422 `invalid_context` (its generic `/api/problems/{id}/…` routes keep that 422; `mock` opens only on this sprint's own routes); m3-11's lazy `CodeEditor.tsx`, `judge.ts` and drafts on `main`.
- [ ] **AB25 frozen:** `design-system/screens/v2/AB25-live-hud-text.html` on `main`.
- [ ] **Parallel sessions:** `gh pr list`, `git worktree list`, ListAgents — no open peer PR edits `internal/judge/admission/**`, `internal/judge/dto/**`, `internal/gateway/withhold.go`, `internal/gateway/sse.go` or `CodeEditor.tsx`; note which of m6a-02/m6a-03 have merged (it decides the seams in steps 2, 3, 5 and 6, incl. assessment's `POST /mocks` `items[]` and the context-id fallback). Take the next free coach goose version at rebase.

## Do this (in order)

1. **[X] judge mock admission** (plan task 1). Read m3-14's `admission.Admit` and m3-06's result tx first; implement only the gaps:
   - mock Run: P1; L9, L10, L11 apply; **L12 never blocks** and **L13 levels 1 and 2 never apply**; the **L19 pace** — ≥ 10 s since the account's previous mock Run, read from Postgres under the per-account advisory lock → 429 `too_fast{reason:"mock_pace", retry_after_ms}` + `Retry-After`; `MockRunPace` in `admission.Limits` with its `JUDGE_LIMITS_*` env override;
   - `job_telemetry` rows with `context_kind='mock'` for Runs and finals (they count in L12 sums and L13 busy %);
   - mock `final`: P0 close, parts may be empty (filled from drafts), row-6 exempt, replay-safe;
   - **no `evaluation_completed` outbox row** for any mock evaluation; mock DTO = aggregates only (hidden passed/total, perf, first-failure class, key aggregate unless concluded);
   - `judge admin status`: mock Runs/finals in the last 24 h and their busy share.

2. **[X] Picker + classic start + `InMock`** (plan task 2).
   - `internal/gateway/mockpick.go`: count from the manifest (DSA 1; clamp 1–3), the three-step pool, evaluable-first with a `runs:false` fallback, live items excluded via `g.itemStates` (fail closed → 503 `mock_state_unavailable`), a deterministic seed, pins with `contract_hash`, nothing echoed before start. Replace m6a-01's placeholder pick in `POST /api/interviews` (the picked `{ordinal, item_id, contract_hash, runs}` go into coach's `items`).
   - **Classic Mock-v2 start:** gateway `POST /api/mocks` with the v2 body `{format:"classic", difficulty, language?}` (cohort only; `Idempotency-Key` required) → `pickMockItems` → assessment `POST /mocks {format:"classic", difficulty, items[{ordinal,item_id,contract_hash}], client_ref}` → `201 {mockId, deadlineAt, count}` with no item in the response. Extend assessment's handler (no migration): `status='live'`, `deadline_at` from the manifest rail, `rubric_snapshot`, `mock_session_item` pins, all inside `CreateMock`'s transaction so **L16** holds (409 `mock_in_progress{mockId}`, 429 `mock_daily_limit`); `GET /mocks/live` returns `items` (additive). The gateway never writes assessment's tables. v1's `{setId, problemId}` body stays unchanged. If m6a-03 already extended `POST /mocks` with `items[]` for text/voice, fold classic into that path (else m6a-03 folds into yours).
   - `withhold.go`: `itemState.InMock` (like `OpenAttempt`), sourced from coach's `GET /interviews/active` (add an additive **`revealed`** flag: true once the interview first reached `live`) + assessment `GET /mocks/live`, cached 15 s per account, invalidated by the gateway's interview/mock routes, fail closed; true when **`revealed` and** the state is in m6a-01's `live` set **or `paused`** — never during `setup`, `preflight` or the first `connecting` (that would reveal the pick, S1); a classic mock's items while it is `live`.

3. **[X] Run / drafts / final / evidence** (plan task 3).
   - **Context id first:** the base is **`mock_session_id`** (t4 §2.1, t6 §5) for interviews and classic sessions; an interview's is linked by m6a-03's start before `preflight`. Only while m6a-03 is not on `main`, fall back to the interview id; whichever of the two merges second removes the fallback. Read m3-06's close guard: per `(context_kind, context_id)` (INV-13) → per-item ids `uuidv5(base, "mock-item:" || ordinal)`; per (context, item) → the base. Write the rule into `api.md`, the Decisions log and a **dated amendment line in ADR-0029 §2**.
   - Routes: `POST /api/interviews/{id}/run`, `GET /api/interviews/{id}/runs/{runId}` (echo to coach on the first terminal DTO, deduped on `runId` by coach), `POST /api/mocks/{id}/items/{ordinal}/runs` (classic Mock-v2), **`GET|PUT /api/interviews/{id}/items/{ordinal}/draft`** and **`GET|PUT /api/mocks/{id}/items/{ordinal}/draft`** (→ judge `GET /drafts?context_kind=mock…` / `PUT /drafts/mock/{context_id}/{part_id}`; `{partId, language, payload}`; ≤ 64 KiB per part → 413; last write wins; writes only in the `editable` set → else 409 `editor_read_only`, or a `live` classic session → else 409 `mock_not_live`), `GET /api/interviews/{id}/evidence` and `GET /api/mocks/{id}/evidence` (202 while pending). Resolve the context server-side (interview: coach's `items` + `mock_session_id` + the `editable` set → else 409 `interview_not_live{state}`; classic: assessment's `mock_session_item` + pin, `live` and before `deadline_at`); `Idempotency-Key` required on Runs; 1 MiB / 64 KiB caps with typed 413; typed 429/503 pass-through; cohort 404 outside the gate; a foreign id → 404 (IDOR tests); every route declares its withhold policy. The generic `/api/problems/{id}/…` routes keep 422 `invalid_context` on `mock`.
   - `g.ensureMockFinals` (idempotency key `uuidv5(base, "final:" || ordinal)`, `parts: []`, filled from the drafts above) called from the interview `finish` proxy, the evidence routes and m6a-03's review trigger.
   - Coach intake: `POST /interviews/{id}/run-echo` (learner-visible subset only; bumps `run_count` / `last_verdict_class`, appends a `run` event, calls m6a-02's `OnRun`), `POST /interviews/{id}/evidence` (aggregate only) into `interview.judge_evidence` — **this sprint owns both**; if m6a-03 merged first with its conditional copy, adopt it and remove your stub.

4. **[X] Editor + publisher** (plan task 4).
   - `CodeEditor.tsx`: `mode`, `readOnly` (compartments, no remount), `onDocChange`; interview mode per AB25 (no completions/snippets, assists per the board, `Mod-Enter` → Run only).
   - `web/src/lib/interview/snapshots.ts`: 2.5 s sampler on sha change, `flush(turn|run|phase|finish|hidden)`, coalescing (one in flight, ≤ 1 / 2 s), monotonic `rev`, 64 KiB pre-check, 429/409 handling, indicator state.
   - `web/src/lib/interview/drafts.ts`: `getMockDraft` / `putMockDraft` on the step-3 draft routes (interview and classic scopes; no context id sent), driving m3-11's autosave logic (15 s debounce; flush on Run, turn, phase, blur, hidden, Finish; a 409 stops it).

5. **[X] Snapshot route + store** (plan task 5).
   - Gateway `PUT /api/interviews/{id}/code` with `httpx.ReadBody(w, r, 64<<10)` → typed 413.
   - Coach `PUT /interviews/{id}/code`: outside the `editable` set → 409 `editor_read_only`, `SnapshotMinInterval` → 429, stale rev, unchanged no-op, in-memory latest, `ScreenSink.OnScreen` (adopt m6a-02's definition if it merged first, else define it), a `snapshot` event (rev + sha only) **only when a timeline row persists**; implement m6a-01's `CodeSource` from the latest snapshot.
   - Migration `coach.interview_snapshot` (next free version; sqlc; `sqlc diff`): persist on turn / phase (full) / 30 s if changed (diff) / interrupt / finish / SIGTERM drain; `textdiff` helper; `SnapshotAt`; the 1 MiB cap with `snapshot_capped`; join m6a-01's retention and coach's erase transaction (the coverage test must stay green).

6. **[X] Bounded SSE** (plan task 6).
   - `internal/gateway/sse.go`: `relayBounded` with `sseBounds` (first byte, idle, max duration, heartbeat, max bytes, on-max behaviour; **injectable per route** so tests cut in ~1 s), a write deadline set but **never cleared**, per-account stream caps → 429 `too_many_streams`.
   - Routes: `GET /api/interviews/{id}/events` (first byte 5 s / **idle 35 s** / 15 s hb / **50 s** / 1 MiB / reconnect), `POST /api/interviews/{id}/turns` (16 KiB body; 30 s / 30 s / 120 s / 256 KiB). Move `turns` off `relayStream` if m6a-02 put it there; give m6a-02's JSON `brief` proxy a bounded 60 s client (or `relayBounded` 30 s / 30 s / 90 s / 64 KiB if it streams); leave `/api/coach/chat` alone.
   - Coach `GET /interviews/{id}/events?after=` over m6a-01's `interview_event` log (writers already use `events.Append`; this sprint adds the `run` and `snapshot` writers); the kind allowlist with the S1 filter; replay then in-process notify with a 1 s poll fallback; `id:` = seq; heartbeat 15 s; 50 s end (on m6a-01's injectable clock).
   - Web: `events.ts` (`EventSource`; track `lastSeq`; native `Last-Event-ID` covers only the browser's own retry, so **every manual reconnect** — reconnect frame, back-off 1/2/5/10 s, 35 s watchdog, 429 — reopens on `?after=<lastSeq>`; dedupe by seq, terminal close, connection state), `turns.ts` (`streamTurn` + typed errors), `run.ts` (`runMockItem` + poll).

7. **[X] Coach lock** (plan task 7). Coach's local `/chat` refusal while `coach_locked` is m6a-01's; this sprint adds the gateway side: the mode gate keeps assessment `GET /mocks/live` for classic and uses coach's `coach_locked` for text/voice; paused is unlocked but `InMock`. Record "no NetworkPolicy PR needed" with the evidence (no new in-cluster caller; m6a-01 records its own outcome — none expected, `pause_exposure` goes through the gateway).

8. **[X] Tests + docs** (plan task 8): the judge, gateway, assessment, coach, web and e2e lists in the plan, including the real-runner measurement (20 mock Runs, p50 < 5 s), the classic start under L16, the draft routes (IDOR, 413, 409) and the resume from `lastSeq` across a forced MaxDuration cut (bounds injected, no real 50 s wait). Update `openapi.yaml`, `api.md`, `services.md`, `data-model.md`, `events.md` and the ADR-0029 §2 amendment line. Then `go test ./...`, `sqlc diff`, the e2e lane, and in `web/`: `npm run typecheck`, `npm run lint`, `npm test`, `npm run build` (bundle check).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** judge owns runs, drafts and evaluations; coach owns interview state, snapshots, events and evidence copies; assessment owns `mock_session(_item)` (the gateway starts a classic session only through assessment's `POST /mocks`); the gateway composes. No cross-schema reads; coach calls nobody new (no coach → judge path).
- **goose + sqlc:** expand-only migrations (new tables, nullable columns); next free version per service at rebase; `sqlc generate` committed; `sqlc diff` clean.
- **Limits fail loudly:** typed 413/429 (`Retry-After`)/503 envelopes; limiter state is Postgres in judge, in-process only where L24 allows (gateway, coach, single replica — note it on the scale-out-blocker list).
- **Privacy:** never log snapshot text, transcripts, prompts or SDP; the event stream and the echo carry learner-visible data only (S1); hint text only after `give_hint` (S4); evidence is aggregate only — no pack material.
- **Frontend:** `theme.css` verbatim, the frozen AB25 behaviour notes; CodeMirror stays in the lazy chunk; pinned MIT deps only.
- **GitOps:** no infra change; never `kubectl apply`; **no alerting (D34)** — `judge admin status` is the on-demand read. No new pod (memory sum unchanged). No NATS subject or ACL change (mock stays silent).
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before merging and before claiming an ADR number (none is expected).
- This sprint **does not tag**.

## Deliverables

- judge: the mock-Run admission branch, `MockRunPace`, mock telemetry, the P0 final path, the mock DTO cases, the silence test, the `judge admin status` lines.
- gateway: `mockpick.go`, the classic Mock-v2 start (v2 body on `POST /api/mocks`), `InMock`, the Run/poll/drafts/evidence routes, `ensureMockFinals`, the run echo, `sse.go`, the bounded `events`/`turns` routes and the bounded `brief` proxy, the snapshot route, the lock derivation.
- assessment: `POST /mocks` with `format:"classic"` + `items[]` (L16 unchanged), `GET /mocks/live` items; no migration.
- coach: snapshot intake + `interview_snapshot` + `textdiff` + `SnapshotAt` + `CodeSource`, the event stream, echo/evidence intake (`interview.judge_evidence`), `revealed` on `GET /interviews/active`; migrations + sqlc.
- web: `CodeEditor` interview mode; `web/src/lib/interview/{snapshots,events,turns,run,drafts}.ts` with tests.
- e2e leg + real-runner measurement; architecture docs.

## Update status

- [`../sprints/sprint-m6a-04.md`](../sprints/sprint-m6a-04.md): each task 🔄 → ✅ (⛔ with a reason); set _Overall_.
- [`../status.md`](../status.md): the **Sprint board** row (M6a stays 🔄); **Decisions log** lines for the mock context-id rule (base `mock_session_id`, per-item ids, the pre-link fallback and who removed it), the classic Mock-v2 start body and the mock draft routes, the parallel-run exception with m6a-02/m6a-03, the L9-applies/L12-L13-exempt reading for mock Runs, the stream bounds, the `InMock` cache TTL, the snapshot timeline cap, "no NetworkPolicy PR needed", and the measured mock Run p50. The flag inventory is unchanged (no env flag added; `JUDGE_LIMITS_MOCK_RUN_PACE` is a limit override, not a feature flag — note it beside the other `JUDGE_LIMITS_*`).
- No new ADR: the per-item mock context id is a **dated amendment line in ADR-0029 §2**. If you depart further from ADR-0029/0032/0035, run the parallel-sessions check before numbering one.

## Done when (acceptance)

- [ ] A mock Run returns within the runner SLO (p50 < 5 s over 20 Runs, real-runner lane); admitted at breaker levels 1 and 2 and with an exhausted budget while a course Run is refused; a second mock Run within 10 s → 429 `too_fast{reason:"mock_pace"}`.
- [ ] Mock Runs and finals are budget-counted and never shed; the final is a P0 close filled from drafts; no mock `evaluation_completed`.
- [ ] Snapshots bounded (64 KiB → 413; ≤ 1 / 2 s → 429; unchanged not stored or re-sent); the timeline persists only on its triggers, ≤ 1 MiB, with retention and erase.
- [ ] The picker skips live items, pins `contract_hash`, reveals nothing before start; `InMock` items withheld everywhere from reveal on, never before (sentinel sweep).
- [ ] A cohort classic Mock-v2 starts on `POST /api/mocks {format:"classic"}` with picked, pinned items under L16; v1's body unchanged.
- [ ] Mock drafts round-trip on the interview and classic draft routes (server-resolved context, 413, 409 outside the editable window, foreign ids 404) and the empty-parts `final` fills from them; the generic routes still refuse `mock`.
- [ ] Events ≤ 50 s (idle 35 s) with lossless resume from `lastSeq`; turns ≤ 120 s; stream caps; no write deadline cleared on interview routes.
- [ ] Coach locked in-session, unlocked while paused; evidence and echo carry aggregate / learner-visible data only.
- [ ] CI green (Go, `sqlc diff`, drift + route-enumeration tests, web, bundle check, e2e).

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: merge only (ships dark in the next `v2.0.x` patch)**: branch `feat/m6a-04-mock-runs-editor-sse`, conventional commits with the attribution lines, a PR, CI green, squash-merge, then `git checkout main && git pull`. **Do not tag** — [m6a-06](../sprints/sprint-m6a-06.md) cuts the patch. There is no infra PR.
