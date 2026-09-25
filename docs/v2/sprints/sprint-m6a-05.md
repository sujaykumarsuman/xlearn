# Sprint m6a-05 — M6a UI part 1: Mock-v2 (AB13), setup/consent/pre-flight/$ cap (AB24), live HUD text (AB25)

> **Milestone:** M6a — text interviewer plus failsafes (T6 **P0**; ships **dark** in a `v2.0.x` patch)   ·   **Track:** product   ·   **Order:** 78
> **Prereqs:** [m6a-03](sprint-m6a-03.md) (mock deltas, proposal/accept, `ScoreMock` once) · [m6a-04](sprint-m6a-04.md) (editor interview mode, snapshot publisher, bounded streams, Run/evidence routes, picker) · [ds-m6a-01](sprint-ds-m6a-01.md) (AB13, AB24, AB25 frozen) · transitively [m6a-01](sprint-m6a-01.md), [m6a-02](sprint-m6a-02.md)
> **Unblocks:** [m6a-06](sprint-m6a-06.md) (grace/paused/resume, debrief/proposal, accessibility, the M6a exit and the patch tag)
> **Release action:** **merge only** (ships dark in the next `v2.0.x` patch, tagged by [m6a-06](sprint-m6a-06.md)) · no infra PR · no new pod
> **Services:** web, plus two small **planned** backend pieces in coach + gateway (task 2: the interview estimate read and the text-mode Hold flag, one expand-only coach migration) — no upstream sprint builds them
> **Artboards:** **AB13** A9 Mock-v2 (`design-system/screens/v2/AB13-mock-v2.html`) · **AB24** setup + consent + pre-flight + $ cap (`AB24-mock-setup-preflight.html`) · **AB25** live HUD, text (`AB25-live-hud-text.html`)
> **Calendar:** Q1 2027. No owner time (the boards were frozen at ds-m6a-01's merge).
> **Execute with:** [`../prompts/prompt-m6a-05.md`](../prompts/prompt-m6a-05.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Screen selection + routes: `MockRoute` (cohort → Mock-v2, else v1 `Mock.tsx`), `setup`, `live/:id`, classic session, m6a-06 placeholders | X | ⬜ |
| 2 | Interview client `web/src/lib/interview/api.ts`; backend: the estimate read and the text-mode Hold (coach + gateway) | X | ⬜ |
| 3 | AB13 Mock-v2: hub, server-picked classic session, mock-mode editor + Run, N dimensions, evidence \| sliders, "Scores lock when saved" | X | ⬜ |
| 4 | AB24 setup: format, duration, multiplier, opt-in + per-session consent, key/model pre-flight, mandatory $ cap | X | ⬜ |
| 5 | AB25 live HUD (text): rail, server clock, turns + composer, editor + Run, `give_hint`, snapshot indicator, Hold, Finish, banners, lifecycle | X | ⬜ |
| 6 | Tests, compose check, board screenshots | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> **M6a** row, the Artboards rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB13, AB24, AB25 frozen:** [ds-m6a-01](sprint-ds-m6a-01.md) merged (the merge is the freeze); `design-system/screens/v2/AB13-mock-v2.html`, `AB24-mock-setup-preflight.html` and `AB25-live-hud-text.html` exist on `main`.
- [ ] **m6a-03 merged:** `mock_session.status ∈ {open, scored, incomplete, abandoned}`, `format ∈ {classic, text, voice}`, `scored_by`, `time_multiplier`, `caveats[]`; rubric validation from the session's `rubric_snapshot` (no exactly-7 check); trends and aggregates count `scored` only; the proposal/accept routes (m6a-06 renders them).
- [ ] **m6a-04 merged:** `CodeEditor` interview mode; `web/src/lib/interview/{snapshots,events,turns,run,drafts}.ts`; the picker and the classic Mock-v2 start (`POST /api/mocks {format:"classic", difficulty, language?}`); the Run, poll, **draft** (`GET|PUT /api/interviews/{id}/items/{ordinal}/draft`, `GET|PUT /api/mocks/{id}/items/{ordinal}/draft`) and evidence routes; the bounded `events`/`turns` streams.
- [ ] **m6a-01 and m6a-02 on `main`:** m6a-01's `/api/interviews/*` — `GET /api/interviews/active`, `POST /api/interviews` (create in `setup`), `PUT …/optin`, `PUT /api/interviews/{id}/consent`, `POST /api/interviews/{id}/{start|attach|heartbeat|interrupt|pause|resume|finish|abandon}`, `GET /api/interviews/{id}` — with its cohort gate (`interviewAudience`); m6a-02's `turns`, `hint`, `probe`, `brief` and the pre-flight inside `start`. Write down the names that actually merged — this sprint calls them, it doesn't invent them.
- [ ] **Parallel sessions** (`gh pr list`, `git worktree list`, ListAgents): no open peer PR edits `web/src/router.tsx`, `web/src/screens/Mock.tsx`, `web/src/screens/mock/**`, `web/src/lib/mock.ts`, `web/src/lib/interview/**` or m6a-02's cue code in `internal/coach/interview/`; a peer's coach goose migration means taking the next free version at rebase. [m6a-06](sprint-m6a-06.md) follows on the same files; don't start it in parallel.

## Goal

Ship the **first half of the interviewer UI** exactly as the frozen boards draw it:
- **AB13 A9 Mock-v2** — the v2 mock hub and the classic, self-scored mock rebuilt for v2: server-picked items (replacing `PROBLEM_SETS`), a mock-mode editor with Run, **N rubric dimensions** from the session's snapshot, the two-column **evidence | sliders** scoring and "Scores lock when saved" ([t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed), [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock); PRD R-MK2).
- **AB24** — interview setup: text format, duration, time multiplier, opt-in and per-session consent, the key/model pre-flight and the **mandatory $ cap** estimate ([t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys); R-MI1).
- **AB25** — the live text HUD: the phase rail and goals, the server clock, the AI turns and composer, the editor with Run, `give_hint`, the snapshot indicator, Hold and Finish ([t6 §3–§5](../research/t6-realtime-interviewer.md#5-the-coding-round); D29; R-MI2).

After this sprint a **cohort account can start and run a text mock interview** in compose from setup to Finish, and run a classic Mock-v2 to a saved score. Grace, pause, resume, debrief and the proposal screens are [m6a-06](sprint-m6a-06.md)'s. Accounts outside the cohort keep v1's Mock screen, unchanged.

## Scope

**In**
- `web/src/router.tsx` (`MockRoute` and the mock subtree under m1-03's `:course`), `web/src/screens/mock/*` (hub, classic session, setup, live HUD and `hud/*` parts), `web/src/lib/interview/api.ts`, `web/src/lib/mock.ts` (v2 types beside the v1 ones).
- **Planned backend** (task 2; no upstream sprint builds either): the interview cost-estimate read, and AB25 F7's **text-mode Hold** (m6a-01 keeps the `held` state voice-only) — coach handlers, one expand-only coach migration (`coach.interview.quiet`), a tweak to m6a-02's proactive-cue guard, and gateway cohort pass-throughs.
- Vitest per frame, Go tests for the backend pieces, a compose check, screenshots.

**Out**
- Grace modal, paused banner, resume modal, `interrupted` and "Still there?" states, debrief, proposal review, `incomplete` view, accessibility settings (AB26–AB28) → [m6a-06](sprint-m6a-06.md). This sprint routes those states to placeholder cards.
- Any other server behaviour (FSM, brain, caps, scoring, picker, classic start, drafts, streams): [m6a-01](sprint-m6a-01.md)–[m6a-04](sprint-m6a-04.md). Apart from task 2's two planned pieces, a missing route is a gap to report, not something to fake in the client.
- Voice: pre-flight, notices, the voice HUD (AB29, AB30★) → [m6b-03](sprint-m6b-03.md).
- The public profile (mock count only, D31) — unchanged.

## Tasks

### 1 · Screen selection + routes [X]

Sources: [m1-03](sprint-m1-03.md) task 4 (`/:course/*`, `useCourse()`), [m3-11](sprint-m3-11.md) task 1 (the selection pattern), [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (presence by config, T-3 cohort).

| Route (under `:course`) | Screen | Notes |
|---|---|---|
| `mock` | `MockRoute` → `MockHub` (AB13) **or** v1 `Mock.tsx` | presence = m6a-01's `GET /api/interviews/active`: `apiNotFound` 404 (outside the cohort) → v1 `Mock.tsx` untouched; 200 → Mock-v2 |
| `mock/setup` | `InterviewSetup` (AB24) | cohort only |
| `mock/live/:id` | `LiveHud` (AB25) | renders by m6a-01's state: `setup` → setup; `preflight`/`connecting` → AB24's "Starting" frame; `live` → the HUD; `interrupted` (grace is `interrupted` + `grace_until`), `paused`, `resuming` → m6a-06's views (placeholder cards until then); `wrapping` stays in the HUD (the debrief, AB27 F1, is m6a-06's); `finished`/`proposed` → `mock/:id/review` (placeholder); terminal → hub |
| `mock/s/:sessionId` | `ClassicSession` (AB13 live + scoring) | classic Mock-v2 |
| `mock/:id/review` | placeholder | [m6a-06](sprint-m6a-06.md) fills it (AB27) |

- Use the board's URLs if AB13/AB24/AB25 name them; otherwise these. Router-ranking tests: none of these shadows `settings`, `auth` or `u/…` (m1-03's rule).
- The client **never** decides eligibility, grading or state: an unknown or foreign id is the server's 404 → NotFound.

### 2 · Interview client [X]

- `web/src/lib/interview/api.ts` (TanStack Query) on the routes m6a-01/m6a-02 merged: `useCurrentInterview()` (`GET /api/interviews/active`), `useInterview(id)` (state, rail, phase goals, items once live, `revCurrent`, cap usage band), `createInterview` (course, difficulty, minutes, multiplier, and the cap wherever m6a-01 takes it — create or start body), `saveOptIn`, `saveConsent`, `startInterview` (runs the pre-flight), `attach`, `heartbeat`, `requestHint`, `hold`/`talk`, `pause`, `finish`, `abandon`, `interruptBeacon`. Every state-changing call carries a UUIDv7 `Idempotency-Key` per user action (m3-11's helper).
- **Typed errors** (the codes m6a-01/m6a-02 define; confirm against what merged): 409 `interview_active{id}` (L19 → link to it), 429 `interview_daily_cap` + `Retry-After` (→ reset time), 409 `consent_required{kinds}`, 422 `cap_required`, 409 `illegal_transition{state, event}`, 409 `lease_lost` (→ "live in another tab"), 410 `resume_expired`, 503 `interviewer_unavailable`, the pre-flight's `preflight_fail{reason ∈ model_access, auth, quota, region, key_required}` (→ back to setup with AB24's copy), m6a-04's `interview_not_live`, `mock_not_live`, `mock_state_unavailable`, `too_many_streams`, `editor_read_only`, `too_fast{mock_pace}`, L16's `mock_in_progress{mockId}` / `mock_daily_limit{resetAt}` (classic start), and typed 413/429 with `Retry-After`.
- **Lease and liveness** (m6a-01): the HUD calls `attach {client_id}` on mount (a per-tab UUID in `sessionStorage`, in `try/catch`) and `heartbeat {client_id, visible, last_input_at}` every 10 s; the server's sweeper turns missing heartbeats into `interrupt(network)` and idle-and-hidden into `interrupt(idle)` — the client never decides either.
- **Estimate (backend, planned here) — the client computes no price.** m6a-01 stores the cap and m6a-02 prices calls, but no upstream sprint exposes an estimate, so this sprint adds one small additive read: coach `GET /interviews/estimate?format=&minutes=&multiplier=&model=` (aud=coach) behind a gateway cohort pass-through `GET /api/interviews/estimate`, plus `openapi.yaml`. It computes from the catalog model and [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)'s assumptions: text ≈ $0.45 (45 min) / ≈ $0.65 (60 min), +20–40 % for multipliers, plus the extras on the same key (review ≈ $0.06, a re-propose ≈ $0.06, a resume brief ≈ $0.03–0.05 per pause). Response `{typicalUsd, planForUsd, defaultCapUsd (= plan-for × 1.5, t6 §3), minCapUsd, maxCapUsd, taxNote}`. No model call.
- **Text-mode Hold (backend, planned here).** AB25 F7 reads Hold in text as "The interviewer stays quiet until you message it or press Talk. The clock keeps running." m6a-01 keeps the FSM's `held` state voice-only, so the text reading is a **flag, not a state**: an expand-only coach migration (next free goose version at rebase; `sqlc generate`; `sqlc diff` clean) adding `coach.interview.quiet boolean NOT NULL DEFAULT false`; coach `POST /interviews/{id}/hold` and `/talk` (only in the `editable` set, else 409 `illegal_transition`; a `notice{kind:"hold"}` event); m6a-02's proactive cues (phase cue, check-in, run-echo probe) skip while it is set; any candidate turn clears it. Gateway cohort pass-throughs `POST /api/interviews/{id}/hold|talk` (each declares its withhold policy) + `openapi.yaml`.
- If a peer sprint shipped either piece after this plan was written, adopt it instead of building a second one. These two are the only backend changes this sprint makes (small, in coach and the gateway, with tests); record each in the Decisions log.

### 3 · AB13 Mock-v2 [X]

Sources: [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) ("server-picked items (replacing `PROBLEM_SETS`); N rubric dimensions; widgets in mock mode; two-column scoring (evidence | sliders), 'Scores lock when saved'"), [t4 §6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock), [PRD](../../prd/xlearn-v2-prd.md) R-MK2 (N dimensions × 1–5, total and max), D31, [m6a-03](sprint-m6a-03.md), [m6a-04](sprint-m6a-04.md) tasks 2–3.

**`web/src/screens/mock/MockHub.tsx`** — every AB13 hub frame:
- the format chooser (AB13 F1): **Classic** (timed, self-scored) · **AI interviewer · text** (on your key, about $0.45); the voice card is **absent** in M6a, not greyed (it arrives with M6b's variant F1b);
- the in-progress card (an interview or classic session that isn't terminal → Continue; a paused interview → the resume entry m6a-06 fills);
- history and trend: **`scored` sessions only** in the trend and averages (m6a-03's aggregates); `incomplete` and `abandoned` rows listed with their label, never charted; the readiness targets as v1;
- empty state.

**`web/src/screens/mock/ClassicSession.tsx`** — the classic Mock-v2 session:
- **Start:** difficulty (and language) → **`POST /api/mocks {format: "classic", difficulty, language?}`** with an `Idempotency-Key` ([m6a-04](sprint-m6a-04.md) task 2: the server picks 1–3 items, assessment creates the session under L16) → `201 {mockId, deadlineAt, count}` → `mock/s/:sessionId`; **nothing about the items shows before Start**. Errors: 409 `mock_in_progress{mockId}` → open that session; 429 `mock_daily_limit{resetAt}` → the reset time; 503 `mock_state_unavailable` → retry;
- the server rail and clock from assessment (as v1: server-authoritative, polled), per-item statement tabs, and the lazy `CodeEditor` in `mode: "interview"` (mock mode: no hints, no coach, no Submit) with **Run** on `POST /api/mocks/{id}/items/{ordinal}/runs` and polling via `GET /api/runs/{id}`; judge `mock` drafts through m6a-04's `drafts.ts` on `GET|PUT /api/mocks/{id}/items/{ordinal}/draft` (m3-11's autosave logic; flush on Run, blur, hidden, Finish; the client sends no context id); `runs: false` items hide Run with AB13's copy; typed 429 `mock_pace` → "Run again in Ns";
- **Finish & score** → `GET /api/mocks/{id}/evidence` (poll the 202) → **two columns**: left, the evidence per item (hidden passed/total, perf, first-failure class, key aggregate; the manifest's evidence hints as *suggested* caps, shown only); right, **one slider per dimension of the session's `rubric_snapshot`** (N dimensions, 1–5, public band descriptors as helper text), notes ≤ 2 KiB;
- **"Scores lock when saved"** → `POST /api/mocks/{id}/score` **once** (m6a-03 accepts it for `format=classic` only and always records `scored_by=self`); the button disables on submit and a replay shows the saved score; the result view's radar and trend adapt to N dimensions and `total`/`max_total`;
- `web/src/lib/mock.ts`: v2 types (`format`, `status`, `scoredBy`, `rubricSnapshot`, `items[]`, `total`/`maxTotal`); `RUBRIC_DIMENSIONS` stays for v1 `Mock.tsx` only.
- The coach panel shows AB01's "paused during mocks" while a mock is live (the server's 409 `coach_paused`).

### 4 · AB24 setup, consent, pre-flight, $ cap [X]

Sources: [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (consent boxes, retention choice, AI disclosure, multiplier), [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (estimate copy, pre-flight checks), [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (caps: mandatory, default plan-for × 1.5, wrap at 85 %, cut at 100 %), [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits), [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (the `interview` key), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (custom model ids: text allowed, never an AI proposal).

**`web/src/screens/mock/InterviewSetup.tsx`** — AB24's frames, in its order and copy. What the server needs from them:

| Step | Content | Server |
|---|---|---|
| Format | **Text** only in M6a (AB24's voice buttons are its "from M6b" variant) | create |
| Duration and item | 45 min (DSA default) or 60 min where the manifest allows (`max_minutes ≤ 60`); difficulty; language from the course | create → the picker (item not shown) |
| Time multiplier | 1× · 1.25× · 1.5× · 2×, badged "never penalised · no reason asked" (WCAG 2.2.1); its cost appears in the estimate; default 1× (m6a-06 adds the AB28 preference) | create (`time_multiplier`) |
| Opt-in (once per account) | unticked "AI mock interviewer" + the "how it works and what it evaluates" card | `PUT …/optin` (`interviewer_optin`) |
| Per-session consent | unticked boxes, text-mode subset: transcript and code go to your provider's text API for the interviewer, summaries and feedback · retention **delete 30 days after I score** (default) or keep 12 months · "The interviewer is an AI. It can't see you." · "Assessment uses what you say and your code, never your voice, face, accent or appearance." Consent version and timestamp from the server | `PUT /api/interviews/{id}/consent` (m6a-01's kinds `text_processing`, `retention`, `ai_disclosure_ack`, `assessment_scope_ack`) |
| Key / model pre-flight | the `interview` default key exists and is enabled (else → Settings AI keys); the model has `interview_brain`; a custom model id → "Custom model: the interview works; its scores are self-reviewed (no AI proposal)". The **probe runs inside `start`** (m6a-02: `setup → preflight`, a 16-token call on the learner's click) → ok, or `preflight_fail{reason}` back to setup with AB24's per-reason copy | `start` |
| **$ cap (mandatory)** | "Typical $0.45 · plan for $X · + tax where applicable (India: 18% GST) · billed by ⟨provider⟩ to your key"; the default cap filled in (plan-for × 1.5), editable within the server's bounds; "At 85 % the interviewer wraps up; at 100 % the interview ends." Start stays disabled without a cap | cap |
| Start | create (if needed) → opt-in → consent → cap → `start` (pre-flight) → AB24's "Starting" frame → `mock/live/:id` | create, consent, `start` |

- Errors: `interview_active` → its link; `interview_daily_cap` → reset time; `consent_required` / `cap_required` → the missing step; cohort 404 → v1. No AI call runs without a click (the probe runs on the learner's Start).
- The item is **never** shown here (S1): the pick is revealed when the interview goes `live`.

### 5 · AB25 live HUD, text [X]

Sources: [t6 §3–§5](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes), [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (accessibility, S1–S4), D29, [m6a-04](sprint-m6a-04.md) tasks 4 and 6 (the libraries).

**`web/src/screens/mock/LiveHud.tsx` + `web/src/screens/mock/hud/*`** — every AB25 frame:
- **Rail and phase goals:** the manifest's public `phase_goals[]` from the server, the current phase highlighted; phase changes arrive as `phase` events.
- **Server clock:** `role="timer"` with a minute-granular accessible label; the value comes from the server (active ms scaled by the multiplier); the client only interpolates between events and snaps to every `state`/`phase` event.
- **Statement panel:** the picked item's public statement, shown once `live`.
- **Turns:** a navigable `role="log"` list with `aria-live="off"` by default (AB28's default announcement policy is "only phase changes and warnings"; m6a-06 adds the "every interviewer message" preference); interviewer turns stream through `streamTurn` and reconcile with the final `turn` event (a dropped stream loses nothing); candidate turns labelled. **Composer:** Enter sends, Shift+Enter newline, ≤ 8 KiB of text (coach's cap), disabled while a turn streams; **before sending**, `flush("turn")` on the snapshot publisher (D29: the interviewer sees the screen at every turn).
- **Editor + Run:** m6a-04's lazy `CodeEditor` (`mode: "interview"`) and **Run** (⌘↵) → `runMockItem` → the compact result line (e.g. "Run: 7/9 samples passed · WA on sample 3"); `runs: false` → Run hidden with AB25's copy; 429 `mock_pace` → "Run again in Ns"; judge `mock` drafts autosaved through `drafts.ts` on `GET|PUT /api/interviews/{id}/items/{ordinal}/draft` (flushed before each turn too, beside the snapshot).
- **`give_hint`:** "Ask for a hint" with AB25's confirm (at most one conceptual hint per phase, never code; the server records it first) → m6a-02's route; the text shows when the `hint` event arrives; disabled with the reason once the phase's hint is used.
- **Snapshot indicator:** from `useSnapshotPublisher` ("Shared with the interviewer · 2 s ago" · sharing · "Too large to share" · off).
- **Hold / Talk** (AB25 F7): "The interviewer stays quiet until you message it or press Talk. The clock keeps running." — on the text-mode Hold flag (task 2), never an FSM transition in text mode.
- **Finish:** "Finish & get feedback" with a confirm → `finish` → `wrapping` (the HUD stays; m6a-06 renders the debrief) → `finished` → `mock/:id/review` (placeholder until m6a-06).
- **Banners and notices:** the connection state from `useInterviewEvents` (live · reconnecting · offline); "Live in another tab"; the 85 % **cap** warning (`cap` event); the **safety card** when m6a-02's `safety_card` system turn arrives (Tele-MANAS **14416** + "your local emergency number" + a Pause offer; [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) enforcement 4) if AB25 draws it.
- **Lifecycle:** `attach` on mount and `heartbeat` every 10 s (task 2); `pagehide` → `navigator.sendBeacon` to m6a-01's interrupt route with `{reason: "tab_closed"}` (it accepts `text/plain`; [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) "Tab close"; verify the same-origin cookie reaches the gateway in Chrome and Firefox); `visibilitychange → hidden` flushes the publisher and the next heartbeat reports `visible: false`; `lease_lost` → the "Live in another tab" frame; `interrupted` (incl. grace) / `paused` / `resuming` → m6a-06's views (placeholder cards until then), with the editor switched read-only.
- **Coach:** hidden or paused (the server's 409 `coach_paused`).
- **Keyboard and motion:** Run ⌘↵, send Enter, the editor's Esc-then-Tab; nothing clashes with the editor (AB25's notes); `prefers-reduced-motion` respected; state shown with icon + text, never colour alone. **< 1024 px:** AB25's layout (e.g. tabs Interviewer · Code · Problem).

### 6 · Tests, compose check, screenshots [X]

- **Vitest:** `MockRoute.test.tsx` (200 → v2, 404 → v1 unchanged); `MockHub.test.tsx`, `ClassicSession.test.tsx`, `InterviewSetup.test.tsx`, `LiveHud.test.tsx` — one test per frame of AB13/AB24/AB25, with m6a-04's libraries mocked; the typed-error mapping (`interview/api.test.ts`); Start disabled without consent or cap; nothing about the item before `live`/Start; flush-before-turn ordering; the read-only switch on `interrupted`; `role="timer"` / `role="log"` labels; the < 1024 px layout (`matchMedia` mocked).
- **Compose check** as the owner (cohort), with a fake provider override kept in the scratchpad (never committed): a text interview from setup through live (two turns, a Run, a hint, Finish → the placeholder), and a classic Mock-v2 through a saved score; a non-cohort account sees v1 `Mock.tsx`. Record the result in the PR.
- **Screenshots** at 1440 px and 390 px beside each board in the PR; m3-11's bundle check still passes (the editor stays lazy).
- The backend pieces: Go handler and store tests (estimate: the t6 §8 figures per duration and multiplier, default cap = plan-for × 1.5, cohort 404 outside the gate; Hold: cues skipped while set, cleared by a candidate turn, refused outside `editable`, the `notice` event), `sqlc diff`, `openapi.yaml` (drift test), m1-06's route-enumeration test (new routes declare a withhold policy).
- Then `npm run typecheck`, `npm run lint`, `npm test`, `npm run build` in `web/`; `go test ./...`.

## Acceptance criteria

- [ ] **Frames match the boards:** every AB13, AB24 and AB25 frame implemented, compared at 1440 px and 390 px (screenshots in the PR).
- [ ] **A text mock is startable by the cohort** in compose: setup (opt-in, consent, pre-flight, cap) → live → streamed turns → Run → hint → Finish; nothing about the item appears before `live`.
- [ ] Classic Mock-v2: started on `POST /api/mocks {format:"classic"}` with server-picked items, mock-mode editor + Run + drafts on m6a-04's mock routes, evidence | sliders over N dimensions from the snapshot, "Scores lock when saved" (one score).
- [ ] The estimate read and the text-mode Hold exist, tested, behind the cohort gate; Hold never moves the FSM in text mode.
- [ ] Outside the cohort, `/:course/mock` is v1's Mock screen, unchanged.
- [ ] The client computes no price, clock, grade or transition; every one comes from the server.
- [ ] CI green (web typecheck/lint/tests/build, bundle check; `go test ./...`, `sqlc diff`, the OpenAPI drift and route-enumeration tests).

## Release

**Merge only.** It ships dark in the next `v2.0.x` patch, which [m6a-06](sprint-m6a-06.md) tags ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): after GA every non-GA release is a patch). Behind m6a-01's interview cohort gate until [m6b-04](sprint-m6b-04.md)'s v2.1.0 flip. The coach and gateway changes (task 2) ride the same fleet tag; the coach migration is expand-only (no contract, floor unchanged). No infra PR, no flag, no new pod.

## Definition of Done

CI green · screens match the frozen boards · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md): Sprint board, M6a row, Artboards rows AB13/AB24/AB25 → "· consumed by m6a-05" appended, plus "frozen (PR #, date)" only if the ds-m6a-01 session didn't already record it) · local `main` synced after the squash-merge.

## Risks / watch-outs

- **Many server seams, three sprints deep.** Route and error names come from m6a-01/02/03/04; read what merged. Never paper over a missing route with client logic — render the board's disabled state and log the gap for m6a-06's P0 audit.
- **S1 in the UI.** The item stays hidden until `live`; hints appear only from `hint` events; pattern chips never render in a mock.
- **Clock drift.** Interpolate for display only and snap to the server on every event; a long-idle tab resyncs on `visibilitychange`.
- **`sendBeacon`** sends cookies but no custom headers — m6a-01's interrupt route accepts `text/plain` for this; if the gateway proxy demands an idempotency or CSRF header, exempt that one route (idempotent by state) rather than dropping the beacon.
- **Screen-reader noise.** `role="log"` is implicitly `aria-live="polite"` — set it to `off` by default so every interviewer message isn't read out; phase changes are announced politely and (from m6a-06) the grace warning is the only assertive announcement (t6 §7, AB25/AB28 notes).
- **v1 regression.** Non-cohort accounts keep v1 `Mock.tsx`; `MockRoute.test.tsx` pins it.
