# Prompt — Sprint m6a-05 · M6a UI part 1: Mock-v2 (AB13), setup/consent/pre-flight/$ cap (AB24), live HUD text (AB25)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-05.md`](../sprints/sprint-m6a-05.md)   ·   **Milestone:** M6a (text interviewer; ships dark in a `v2.0.x` patch)   ·   **Prereqs:** [m6a-03](../sprints/sprint-m6a-03.md), [m6a-04](../sprints/sprint-m6a-04.md), [ds-m6a-01](../sprints/sprint-ds-m6a-01.md) (AB13, AB24, AB25 frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, stack, land-and-sync.
- The plan: [`../sprints/sprint-m6a-05.md`](../sprints/sprint-m6a-05.md). Its route table (task 1), the AB24 step table (task 4) and the acceptance list are authoritative for this session.
- **The frozen boards** (every frame, copy and behaviour note): `design-system/screens/v2/AB13-mock-v2.html`, `design-system/screens/v2/AB24-mock-setup-preflight.html`, `design-system/screens/v2/AB25-live-hud-text.html`, via `design-system/screens/v2/index.html`. [`design-system/theme.css`](../../../design-system/theme.css) verbatim. The v1 [`Mock.dc.html`](../../../design-system/screens/Mock.dc.html) is a reference only (superseded by AB13).
- [t6](../research/t6-realtime-interviewer.md) §3 (routes, caps), §4 (clock, states, tab close), §5 (coding round), §6 (enforcement 4: the distress card), §7 (consent boxes, retention, accessibility, S1–S4), §8 (estimate copy, pre-flight), §13 (D29–D31 override the body). [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §4–§6. [PRD](../../prd/xlearn-v2-prd.md) §5.2 R-MK2; §5.6 R-MI1, R-MI2, R-AC6.
- [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (Mock-v2: server-picked items, N dimensions, evidence | sliders) and [§6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock).
- Peer plans: [m6a-01](../sprints/sprint-m6a-01.md) (routes, FSM states, cohort gate), [m6a-02](../sprints/sprint-m6a-02.md) (turns, probe, `give_hint`, the proactive cues the Hold flag silences), [m6a-03](../sprints/sprint-m6a-03.md) (mock deltas, scoring), [m6a-04](../sprints/sprint-m6a-04.md) (editor mode, `web/src/lib/interview/*` incl. `drafts.ts`, picker, the classic start, Run/drafts/evidence routes, streams), [m6a-06](../sprints/sprint-m6a-06.md) (what comes next on the same files), [m1-03](../sprints/sprint-m1-03.md) (`/:course/*`), [m3-11](../sprints/sprint-m3-11.md) (editor, drafts, idempotency helper, bundle check), [m1-07](../sprints/sprint-m1-07.md) (coach paused copy, AB01).
- Code: `web/src/router.tsx`, `web/src/screens/Mock.tsx` (v1 — keep), `web/src/lib/mock.ts`, `web/src/lib/interview/*`, `web/src/screens/workspace/CodeEditor.tsx`, `web/src/lib/judge.ts`, `web/src/lib/api.ts`, `web/src/components/Coach.tsx`, `web/src/components/AppShell.tsx`; for task 2's backend: `internal/coach/interview/` (handlers, m6a-02's cue guard), `internal/coach/catalog.go`, `internal/coach/store/`, the gateway's interview proxy, `docs/architecture/openapi.yaml`.

## Context

- [m6a-01](../sprints/sprint-m6a-01.md)…[m6a-04](../sprints/sprint-m6a-04.md) built the interviewer's server side and client libraries dark: FSM and caps, the text brain, proposal and scoring, and the coding round with bounded streams.
- **This sprint builds the first three screens on them**, exactly as the frozen boards draw them: the Mock-v2 hub and classic session (AB13), the interview setup with consent, pre-flight and the mandatory $ cap (AB24), and the live text HUD (AB25).
- After it, a **cohort account (owner/tester) can start and run a text mock in compose**; [m6a-06](../sprints/sprint-m6a-06.md) adds grace/pause/resume, debrief/proposal and accessibility, proves the M6a exit and tags the patch.
- **Owner-decided (never contradict):** the AI is disclosed; it watches the answer widgets, not the learner (D29); no AI call on the learner's key without their action; a mandatory $ cap; transcripts deleted 30 days after scoring unless kept; text mode is first-class and never penalised; the time multiplier is badged and never penalised; the public profile shows the mock count only (D31).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB13, AB24 and AB25 exist on `main` (the owner merged [ds-m6a-01](../sprints/sprint-ds-m6a-01.md)'s PR).
- [ ] [m6a-03](../sprints/sprint-m6a-03.md) merged (`status`/`format`/`scored_by`/`time_multiplier`/`caveats`, snapshot rubric, `scored`-only aggregates, proposal/accept routes).
- [ ] [m6a-04](../sprints/sprint-m6a-04.md) merged (editor interview mode, `web/src/lib/interview/{snapshots,events,turns,run,drafts}.ts`, picker, the classic start `POST /api/mocks {format:"classic", difficulty, language?}`, the Run / draft (`…/items/{ordinal}/draft` on interviews and mocks) / evidence routes, bounded streams).
- [ ] m6a-01/m6a-02 routes on `main`: `GET /api/interviews/active`, `POST /api/interviews`, `PUT …/optin`, `PUT /api/interviews/{id}/consent`, `POST /api/interviews/{id}/{start|attach|heartbeat|interrupt|pause|resume|finish|abandon}`, `GET /api/interviews/{id}`, and m6a-02's `turns`, `hint`, `probe`, `brief` (pre-flight runs inside `start`); the cohort gate (`interviewAudience`). **Write down the merged names** before coding.
- [ ] **Parallel sessions:** `gh pr list`, `git worktree list`, ListAgents — no open PR edits `router.tsx`, `Mock.tsx`, `web/src/screens/mock/**`, `web/src/lib/mock.ts`, `web/src/lib/interview/**` or m6a-02's cue code; a peer's coach goose migration means the next free version at rebase; m6a-06 is not running.

## Do this (in order)

1. **[X] Routes + selection** (plan task 1): `MockRoute` for `/:course/mock` — `GET /api/interviews/active` 404 → v1 `Mock.tsx` untouched, 200 → `MockHub`; `mock/setup`, `mock/live/:id` (state redirects), `mock/s/:sessionId`, and a `mock/:id/review` placeholder for m6a-06; the board's URLs win if they name any; router-ranking tests.

2. **[X] Client** (plan task 2): `web/src/lib/interview/api.ts` on the merged routes, an idempotency key per user action, the typed-error checklist (`interview_active`, `interview_daily_cap`, `consent_required`, `cap_required`, `illegal_transition`, `lease_lost`, `resume_expired`, `interviewer_unavailable`, `preflight_fail{reason}`, m6a-04's codes, and L16's `mock_in_progress` / `mock_daily_limit` on the classic start); `attach` on mount + `heartbeat` every 10 s. **Planned backend (no upstream sprint builds it):** (a) the estimate read — coach `GET /interviews/estimate` + gateway cohort pass-through `GET /api/interviews/estimate` + `openapi.yaml` (t6 §8 figures; default cap = plan-for × 1.5; no model call); (b) the **text-mode Hold** flag — an expand-only coach migration `coach.interview.quiet`, `POST /interviews/{id}/hold|talk` in the `editable` set (+ gateway pass-throughs), m6a-02's proactive cues skip while set, a candidate turn clears it. Small, tested, recorded — the only backend changes this sprint makes (adopt a peer's version instead if one merged meanwhile).

3. **[X] AB13 Mock-v2** (plan task 3):
   - `MockHub.tsx`: the format chooser (Classic · AI interviewer · text; the voice card is absent in M6a, AB13 F1), in-progress card, history and a trend of `scored` only.
   - `ClassicSession.tsx`: Start → `POST /api/mocks {format:"classic", difficulty, language?}` (Idempotency-Key; server-picked items, L16's 409 → open that session, 429 → reset time; nothing shown before Start), the server rail and clock, per-item statement tabs, the lazy editor in `mode: "interview"` with Run on `POST /api/mocks/{id}/items/{ordinal}/runs` and judge `mock` drafts via `drafts.ts` on `GET|PUT /api/mocks/{id}/items/{ordinal}/draft`.
   - Finish & score → evidence (poll 202) → **evidence | sliders** over the session's `rubric_snapshot` (N dimensions, 1–5, descriptors) → "Scores lock when saved" (one score).
   - v2 types in `web/src/lib/mock.ts`; `RUBRIC_DIMENSIONS` kept for v1.

4. **[X] AB24 setup** (plan task 4): format, duration, difficulty/language, the time multiplier (1× · 1.25× · 1.5× · 2×, badged), the account opt-in and per-session consent (unticked; text subset; retention choice; AI disclosure; content-only assessment), the key/model checks and the pre-flight inside `start` on the learner's click (ok, or `preflight_fail{model_access|auth|quota|region|key_required}` back to setup; the custom-model notice), the **mandatory $ cap** with the server's estimate and "+ tax where applicable (India: 18% GST)" copy. Start is disabled without consent and a cap. The item is never shown here.

5. **[X] AB25 live HUD** (plan task 5):
   - rail + phase goals, the server clock (`role="timer"`), the statement once `live`;
   - the turn log (`role="log"`) + composer (flush the snapshot publisher with `turn` **before** each send; ≤ 8 KiB of text);
   - the lazy editor + Run (⌘↵) and its compact result; drafts via `drafts.ts` on `GET|PUT /api/interviews/{id}/items/{ordinal}/draft`; `give_hint` with the board's confirm, the text shown from the `hint` event; the snapshot indicator; Hold / Talk (AB25 F7) on the text-mode Hold flag, never an FSM transition; Finish → `wrapping` in the HUD, then `finished` → the review placeholder;
   - connection / other-tab (`lease_lost`) / 85 % cap banners; the safety card when m6a-02's `safety_card` turn arrives, if AB25 draws it;
   - `pagehide` → `sendBeacon` interrupt `{reason:"tab_closed"}` (m6a-01 accepts `text/plain`; check cookies reach the gateway), `visibilitychange` flush; read-only editor and placeholder cards on `interrupted` (incl. grace) / `paused` / `resuming`; `wrapping` stays in the HUD;
   - coach paused; < 1024 px per the board; reduced motion; icon + text states.

6. **[X] Tests + check** (plan task 6): Vitest per frame of AB13/AB24/AB25 plus `MockRoute`, the typed errors, gating (consent/cap), nothing-before-live, flush-before-turn, read-only switch, a11y roles, narrow layout. A compose check as the owner with a scratchpad fake-provider override (never committed): a text interview setup → live → two turns, a Run, a hint → Finish, and a classic Mock-v2 to a saved score; a non-cohort account sees v1. Screenshots at 1440/390 beside each board in the PR. Go tests for the estimate read and the Hold flag (cues skipped while set, cleared by a candidate turn, refused outside `editable`), `sqlc diff`, the OpenAPI drift and route-enumeration tests. Then `npm run typecheck`, `npm run lint`, `npm test`, `npm run build` (bundle check), and `go test ./...`.

## Constraints

- **Server-authoritative:** clock, phase, state transitions, caps, prices, picks, evidence and scores come from the server; the client renders, streams and polls. Never fake a missing route in the client — render the board's disabled state and note the gap for m6a-06's P0 audit.
- **Privacy / S1–S4:** no item details before `live`/Start; hint text only from `hint` events; no pattern chip in a mock; never log turn or code text in the browser console.
- **Frontend:** `theme.css` tokens and components verbatim (no Tailwind), dark theme, the frozen copy and layout; difficulty tokens Easy `--ds-ok`, Medium `--ds-warn`, Hard `--ds-err`; CodeMirror stays lazy (bundle check).
- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** web, plus task 2's two planned backend pieces (coach owns both; the gateway passes them through). The Hold flag is an expand-only coach migration: next free goose version at rebase, `sqlc generate` committed, `sqlc diff` clean.
- **GitOps:** no infra change; never `kubectl apply`. **No alerting (D34).** No new pod.
- **Parallel sessions:** check peers' PRs, tags, worktrees and ListAgents before merging and before claiming an ADR number (none expected).
- This sprint **does not tag**.

## Deliverables

- `web/src/router.tsx` (`MockRoute`, the mock subtree), `web/src/screens/mock/{MockHub,ClassicSession,InterviewSetup,LiveHud}.tsx` + `hud/*`, `web/src/lib/interview/api.ts`, v2 types in `web/src/lib/mock.ts`.
- The estimate read and the text-mode Hold flag in coach (one expand-only migration) + gateway pass-throughs + `openapi.yaml`, with tests.
- Vitest suites, the compose-check notes and board screenshots in the PR.

## Update status

- [`../sprints/sprint-m6a-05.md`](../sprints/sprint-m6a-05.md): each task 🔄 → ✅ (⛔ with a reason); set _Overall_.
- [`../status.md`](../status.md): the **Sprint board** row (M6a stays 🔄); the **Artboards** rows AB13, AB24, AB25 → "frozen (PR #, date) · consumed by m6a-05" if the design sprint's rows weren't set yet; **Decisions log** lines for the route URLs, the estimate read and the text-mode Hold added here (routes, the `quiet` column), the beacon approach, and every server gap you found (with the route or behaviour missing) so m6a-06's P0 audit picks them up.
- No ADR expected.

## Done when (acceptance)

- [ ] Every AB13, AB24 and AB25 frame matches the frozen board at 1440 px and 390 px (screenshots in the PR).
- [ ] A text mock is startable by the cohort in compose (setup → live → turns → Run → hint → Finish); the item is hidden until `live`.
- [ ] Classic Mock-v2: started on `POST /api/mocks {format:"classic"}` with server-picked items, mock-mode editor + Run + drafts on m6a-04's mock routes, evidence | sliders over N dimensions, one locked score.
- [ ] The estimate read and the text-mode Hold exist and are tested; Hold never moves the FSM in text mode.
- [ ] Outside the cohort `/:course/mock` is v1's screen, unchanged.
- [ ] No client-side price, clock, grade or transition.
- [ ] CI green.

**Ship at session end** per AGENT.md land-and-sync with **this sprint's release action: merge only (ships dark in the next `v2.0.x` patch)**: branch `feat/m6a-05-mock-ui-part-1`, conventional commits with the attribution lines, a PR with the board screenshots and the compose-check notes, CI green, squash-merge, then `git checkout main && git pull`. **Do not tag** — [m6a-06](../sprints/sprint-m6a-06.md) cuts the patch. No infra PR.
