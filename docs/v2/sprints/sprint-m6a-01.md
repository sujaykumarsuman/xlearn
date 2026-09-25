# Sprint m6a-01 — Interview core: schema, state machine, failsafes, caps (L19)

> **Milestone:** M6a — text interviewer plus failsafes (v2.1 line; ships dark in a v2.0.x patch)   ·   **Track:** product (order 74)
> **Prereqs:** [ds-m6a-02](sprint-ds-m6a-02.md) (the M6a boards frozen; [ds-m6a-01](sprint-ds-m6a-01.md) before it accepted ADR-0032 with the S6 result) · [ga-02](sprint-ga-02.md) (v2.0.0 live)
> **Unblocks:** [m6a-02](sprint-m6a-02.md) (the text brain registers on this core) · [m6a-04](sprint-m6a-04.md) (judge mock budgeting, editor + snapshots, bounded SSE over this sprint's event log, the `InMock` withhold and the item picker)
> **Release action:** **merge only (ships dark in the next v2.0.x patch)** — normally [m6a-06](sprint-m6a-06.md)'s M6a patch; any earlier v2.0.x patch (e.g. a D6 content wave) carries it dark too. Plus task 7's infra PR **only if** coach gains a new in-cluster callee (the recommended design needs none).
> **Calendar:** Q1 2027, after GA. No owner involvement.
> **Execute with:** [`../prompts/prompt-m6a-01.md`](../prompts/prompt-m6a-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Coach schema: `interview`, event log, segment log, turns, checkpoints, pauses, consent, exposure, hint ledger, `admin_audit` | X | ⬜ |
| 2 | FSM + server clock + sweeper: grace, pause ≤ 24 h, resume, `incomplete`, checkpoints, lease/heartbeat, `pause_exposure` | X | ⬜ |
| 3 | Manifest: `mock.interviewer`, rubric `evidence` / `ai` / public descriptors, never-list lint | X | ⬜ |
| 4 | Caps (L19) + coach lock + gateway `/api/interviews/*` (cohort-gated, aud=coach) + tab-close beacon exemption | X | ⬜ |
| 5 | Retention (30 d after scoring / 12 m opt-in, per-turn and per-session delete) + erase coverage | X | ⬜ |
| 6 | `coach admin interviews --live` (+ `show`, `abandon`) for the release checklist; runbook | X | ⬜ |
| 7 | Policies for new internal callers: confirm none (gateway path) — or the infra PR, merged before the M6a patch tag | I | ⬜ |
| 8 | Docs: architecture (services, api, data-model, openapi), ADR-0032 dated update | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> M6a milestone row, the L19 limits row, the flag inventory, the artboard freeze rows). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **S6 done** ([spk-04](sprint-spk-04.md)): the results note (t6 §16) and the scrubbed fixtures (`docs/v2/research/t6-s6-fixtures/`) are merged.
- [ ] **ADR-0032 Accepted** with the S6 result folded in ([ds-m6a-01](sprint-ds-m6a-01.md) task 1; BP2: 0032 is accepted at S6, before the M6a design freeze).
- [ ] **AB13, AB24–AB28 frozen**: [ds-m6a-01](sprint-ds-m6a-01.md) and [ds-m6a-02](sprint-ds-m6a-02.md) merged (the merge is the freeze). Boards: `design-system/screens/v2/AB13-mock-v2.html`, `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html`, `AB26-grace-paused-resume.html`, `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html`.
- [ ] **M3 `mock` context live** (built in [m3-06](sprint-m3-06.md), shipped dark in v1.13.0; [m6a-04](sprint-m6a-04.md) only adds mock Run budgeting and the no-emission rule).
- [ ] **v2.0.0 live** ([ga-02](sprint-ga-02.md)). It carries everything this sprint builds on: `coach.key_default(feature='interview')` and the catalog `interview_brain` capability ([m1-10](sprint-m1-10.md)); the `coach_paused` mode gate ([m1-07](sprint-m1-07.md)); `withhold()` and its fail-closed resolver ([m1-06](sprint-m1-06.md)); coach's erase consumer and the per-schema erase **coverage test** ([l-01](sprint-l-01.md)); `internal/platform/llm` typed errors ([m4-01](sprint-m4-01.md)); the arena reveal/submit records ([m3-08](sprint-m3-08.md), [m3-14](sprint-m3-14.md), [m3-09](sprint-m3-09.md)); `account.role` and the T-3 cohort ([m1-04](sprint-m1-04.md)).
- [ ] **Parallel sessions:** no open peer PR adds a coach goose migration (if one does, take the next free version at rebase) or edits `internal/course` manifest types or `curriculum/courses/dsa/course.json` (serialize the golden update).

## Goal

Build the interviewer's **durable core** in coach ([ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec),
[t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)): the tables, the
**server-clocked state machine** with the owner's three failsafes (5-minute grace, pause ≤ 24 h from the first pause with ≤ 3
pauses, resume with a free deterministic state), `incomplete` after 24 h (D30), the L19 caps, retention and erase, and the
`coach admin interviews --live` check the release checklist needs "from M6". Nothing here calls a model: the brain, the
classifier and the paid brief are [m6a-02](sprint-m6a-02.md); scoring is [m6a-03](sprint-m6a-03.md); runs, snapshots and
streaming are [m6a-04](sprint-m6a-04.md). Everything ships **dark** behind the T-3 cohort gate.

## Scope

**In**
- New package `internal/coach/interview` (FSM, clock, event log, sweeper, caps, consent, retention, HTTP handlers) on coach's
  existing store (`internal/coach/store`: one migration + sqlc queries), expand-only.
- The FSM table as data with the **voice states already in the enum** (`connecting`, `held`, `bridging`) so M6b needs no
  migration; text mode never enters `held`/`bridging`.
- The per-interview **event log** (`interview_event`, monotonic `seq`) that m6a-04's bounded SSE streams; FSM transitions append
  `state` events in the same transaction.
- Deterministic checkpoints, lease + heartbeat + idle/network interruption, `pause_exposure` (gateway push + backstop).
- Manifest `mock.interviewer`, `mock.rubric.dims[].evidence` / `.ai` / public `descriptors`, and the never-list lint
  ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment)).
- L19 caps ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)) that live in coach: 1
  non-terminal per account, ≤ 2 starts/day, mandatory $ cap (stored + threshold rule); the shared constants for the snapshot
  (≤ 1 / 2 s, ≤ 64 KiB, L6) and mock Run (≤ 1 / 10 s) caps; aud=coach; coach's local `locked` check.
- The gateway `/api/interviews/*` lifecycle proxy (cohort-gated) with a placeholder item pick, and the arena exposure hook.
- Retention (30 d after scoring by default, 12 months opt-in, per-turn and per-session delete) and erase coverage.
- `coach admin interviews --live|show|abandon` + `coach.admin_audit`; runbook `docs/runbooks/interviewer.md`.

**Out (later sprints)**
- Brain loop, turns, director cues, `give_hint` release, classifier + probe, resume brief, `store:false` test, replay tests → [m6a-02](sprint-m6a-02.md).
- assessment deltas (`format`, `status`, `time_multiplier`, `caveats`), linking the interview to a `format=text` `mock_session`
  at start, proposal → explicit accept → `ScoreMock` once, twin gate → [m6a-03](sprint-m6a-03.md). Until then
  `interview.mock_session_id` stays NULL (the interviewer is API-only and dark).
- The `PUT …/code` snapshot route and the `interview_snapshot` timeline, the run echo and evidence intake, judge mock Runs/finals,
  the real item picker (`mockpick.go`) with `withhold()`, the `InMock` item state, the gateway's coach-lock derivation for
  text/voice, the bounded SSE `GET …/events` → [m6a-04](sprint-m6a-04.md) (its new tables join this sprint's retention and erase).
- All UI (AB13, AB24–AB28) → [m6a-05](sprint-m6a-05.md), [m6a-06](sprint-m6a-06.md).
- Voice behaviour, ≤ 3 live voice platform-wide, ≤ 75 voice-min/day → [m6b-01](sprint-m6b-01.md), [m6b-02](sprint-m6b-02.md);
  coach 256 Mi / grace 60 s / maxSurge → [mi-13](sprint-mi-13.md).

## Tasks

### 1 · Coach schema [X]

Sources: [t6 §7 data inventory](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility),
[t6 §3–§4](../research/t6-realtime-interviewer.md#3-architecture--media-path), [ADR-0032 §3, §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits),
[ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (schema-per-service, goose on startup, sqlc).

One expand migration `internal/coach/store/migrations/0000N_interview.sql` (next free version; new tables only). Every table
**except `admin_audit`** carries `account_id uuid NOT NULL` (the l-01 erase coverage test requires it; `admin_audit` holds no
account data and is exempted with a reason, task 5). Every table is schema-qualified `coach.`. Soft references only (no
cross-schema FK).

| Table | Key columns | Class |
|---|---|---|
| `interview` | `id`, `account_id`, `path_slug`, `mode` (`text`\|`voice`, default `text`), `state` (15-value CHECK, task 2), `state_reason`, `state_since`; `items jsonb` (ordered `[{ordinal, item_id}]`), `manifest_hash`, `mock_snapshot jsonb` (rail, rubric `id@v`, `interviewer` block at create — the ADR-0026 snapshot rule); `minutes` (45\|60), `time_multiplier numeric(3,2)` ∈ {1, 1.25, 1.5, 2}, `qa_slice bool`; clock `active_ms bigint`, `live_since`, `phase_index`; grace `grace_until`, `grace_probes`, `grace_reprimes`; pause `first_paused_at`, `resume_by`, `pause_count` (≤ 3); money `cap_micros bigint NULL` (no default; set by `start`; CHECKs `cap_micros IS NULL OR cap_micros > 0` and `state IN ('setup','abandoned') OR cap_micros IS NOT NULL`, so `setup → abandoned` never violates them), `spent_micros`, `cut_short` (`cap`\|`pause_limit`\|`quota`\|NULL), `report_pending`; key `key_id`, `model`, `model_is_custom`; `mock_session_id uuid NULL UNIQUE` (m6a-03 sets it); `run_count`, `last_verdict_class` (maintained by m6a-04's echo intake); lease `client_lease uuid`, `last_heartbeat_at`, `last_input_at`, `visible`; `pause_exposure bool`, `review_flag bool` (m6a-02's never-list scan, m6a-03's review checks), `caveats text[]`; retention `retention` (`30d`\|`12m`), `purge_at`, `purged_at`; `created_at`, `started_at` (set at `preflight_ok`; what the ≤ 2 starts/day cap counts), `finished_at`, `scored_at`, `ended_at` | C3 (+ C4 via children) |
| `interview_event` | `(interview_id, seq bigint)`, `kind` (`state`\|`phase`\|`turn`\|`hint`\|`run`\|`snapshot`\|`cap`\|`notice`), `data jsonb` (≤ 4 KiB, S1-filtered), `created_at` — the **append-only log whose `seq` is the SSE cursor** ([m6a-04](sprint-m6a-04.md) streams it with `Last-Event-ID`). One helper `events.Append(ctx, tx, id, kind, data)`; this sprint writes `state` events | C3 |
| `interview_segment` | `(interview_id, n)`, `kind` (`text`\|`voice`), `model`, `provider_session_id NULL`, `opened_at`, `closed_at`, `close_reason`, `usage jsonb`, `cost_micros`; **partial unique** `(interview_id) WHERE closed_at IS NULL` (≤ 1 live segment) — the "mint" log; SDP is never stored | C2, 90 d |
| `interview_turn` | `(interview_id, seq)` (transcript order), `role` (`interviewer`\|`candidate`\|`system`), `kind` (`speech`\|`disclosure`\|`phase_cue`\|`checkin`\|`hint`\|`hint_offer`\|`safety_card`\|`debrief`), `source` (`server`\|`client`), `text` (≤ 8 KiB), `prompt_v`, `truncated`, `edited`, `deleted`, `phase_index`, `at_active_ms`, `created_at` (m6a-02 writes it) | C4 |
| `interview_checkpoint` | `(interview_id, n)`, `kind` (`phase`\|`interrupt`\|`pause`\|`sigterm`\|`finish`), `phase_index`, `per_phase_ms jsonb`, `code_sha`, `code_head` (≤ 16 KiB), `upto_turn_seq`, `hints_given`, `run_count`, `last_verdict_class` | C4 |
| `interview_pause` | `(interview_id, n)` n ≤ 3, `paused_at`, `reason`, `resumed_at`, `reaffirmed_at`, `brief jsonb NULL`, `brief_model`, `brief_at` (the brief is m6a-02's; cached here per pause) | C4 |
| `interview_consent` | `id`, `interview_id NULL` (NULL = account-level opt-in), `kind`, `version`, `value text`, `at` | C1/C2, life of account |
| `interview_exposure` | `(interview_id, kind)` UNIQUE, `kind` (`view`\|`reveal`\|`submit`), `first_at`, `source` (`gateway`\|`backstop`) | C3 |
| `interview_hint` | `(interview_id, n)`, `level`, `phase_index`, `trigger` (`learner`\|`offer_accepted`), `released_at` — the ledger m6a-02 writes **before** any hint text enters a context (S4) | C3 |
| `admin_audit` | `id`, `verb`, `target`, `detail jsonb` (no C4), `at` — coach's CLI trail (identity's shape) | C2 |

- **Partial unique index** `interview_one_open_uq ON coach.interview (account_id) WHERE state NOT IN ('scored','incomplete','abandoned')` (L19).
- Sweeper indexes: `(state, grace_until)`, `(state, resume_by)`, `(state, last_heartbeat_at)`, `(state, state_since)`,
  `(purge_at) WHERE purged_at IS NULL`.
- sqlc queries in `internal/coach/store/queries/interview.sql` (+ `interview_admin.sql`); `sqlc generate`, commit `gen/`.
  Transitions are **compare-and-set**: `UPDATE coach.interview SET … WHERE id = $1 AND account_id = $2 AND state = $3`
  (`:execrows`; 0 rows → `ErrIllegalTransition` or a concurrent winner) — the score-once pattern `ScoreMock` uses.
- Size caps enforced in Go (not fallible CHECKs that would log content): turn ≤ 8 KiB, Σ turns ≤ 256 KiB per interview
  (beyond → `transcript_full`, which m6a-02 turns into a wrap-up), event `data` ≤ 4 KiB.

### 2 · FSM + server clock + sweeper [X]

Sources: [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (FSM diagram, failsafes 1–3,
failure-mode table, lock list), [t6 §13 D29/D30](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict),
[ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec), PRD R-MI3–R-MI5 ([`../../prd/xlearn-v2-prd.md`](../../prd/xlearn-v2-prd.md)).

**States** (`internal/coach/interview/fsm.go`): `setup`, `preflight`, `connecting`, `live`, `held`\*, `bridging`\*, `interrupted`,
`paused`, `resuming`, `wrapping`, `finished`, `proposed`, and the absorbing terminals `scored`, `incomplete`, `abandoned`
(\* voice only; unreachable in `mode=text`). The table is **data** (one `map[State]map[Event]rule`), the single source for the
handlers, the sweeper, the model check and the docs table. The clock is injectable (`Clock` interface) for tests and for m6a-04.

| From | Event → To | Notes |
|---|---|---|
| `setup` | `start` → `preflight` | guards: account opt-in + session consents at the current version, caps (task 4: fewer than 2 interviews with `started_at` in today's UTC day), `cap_micros > 0`, an enabled `interview` key; `start` itself is **not** counted |
| `setup` | sweeper `setup_expired` (`state_since` > 1 h) → `abandoned(setup_expired)` | never started: not counted as a start |
| `preflight` | `preflight_ok` → `connecting`; `preflight_fail(reason)` → `setup`; sweeper `preflight_timeout` (`state_since` > 2 min, e.g. a coach restart mid-probe) → `setup` (`preflight_fail(timeout)`) | `preflight_ok` sets `started_at` — **this is what counts** toward ≤ 2 starts/day; a failed or timed-out pre-flight costs no start. m6a-02 supplies the key/model probe; until a `Brain` is registered, `start` returns 503 `interviewer_unavailable` (a code guard, not a flag) |
| `connecting` | `segment_open` → `live`; sweeper `connect_timeout` (`state_since` > 2 min, or no heartbeat for 30 s) → `interrupted(network)` | opens an `interview_segment`; text: immediate. A `connecting` entered by `cleared` (a re-prime inside a grace) re-enters `interrupted` **keeping** `grace_until` and the re-prime count — no new grace; any other entry starts a normal grace |
| `live` | `interrupt(reason)` → `interrupted`; sweeper `heartbeat_lost` (30 s) → `interrupt(network)`; sweeper `idle` → `interrupt(idle)` | reasons `quota`, `rate`, `provider`, `network`, `idle`, `tab_closed`, `model_access`; **deterministic checkpoint**, segment closed, `grace_until = now + 5 min`, the in-flight interviewer turn marked `truncated` |
| `live` | `rail_done` (sweeper) \| `finish` (learner) \| `cap_85` → `wrapping`; `cap_100` → `finished(cut_short=cap)` | `cap_*` from `spent_micros / cap_micros` via `AddSpend` (m6a-02 adds spend) |
| `interrupted` | `cleared` → `connecting` (≤ 2 automatic re-primes per grace); `finish_now` (≥ 75 % of the rail, key has credit) → `wrapping`; `end_without_feedback` (≥ 75 %, the last probe still dry) → `finished(report_pending=quota)`; `grace_expired` (sweeper) \| `save_later` \| `auth` → `paused` | grace counters reset on each new `interrupted` from `live` (not on a re-prime timeout, above). `end_without_feedback` is AB26's **[End without feedback]** and, at ≥ 90 % with reason `quota`, the default **[Finish without the rest]** (t6 §4 failure table) — no debrief call on a dry key. The sweeper's final probe at `grace_until` may fire `cleared` only while the lease is fresh (heartbeat ≤ 30 s); otherwise `grace_expired` |
| `paused` | `resume` → `resuming`; sweeper `resume_expired` (`now > resume_by`) → `incomplete` (D30) | `resume_by = first_paused_at + 24 h`, **absolute**, set once |
| `resuming` | `resume_confirmed` → `connecting`; `still_dry` \| sweeper `heartbeat_lost` (30 s, e.g. the tab closed on the resume modal) → `paused` (same pause, no new count); `finish_now` (≥ 75 %) → `wrapping`; sweeper `resume_expired` (`now > resume_by`) → `incomplete` | the paid brief between these is m6a-02's |
| `wrapping` | `debrief_done` (sweeper, after `debrief_minutes` of active time, or the brain's signal) → `finished`; `interrupt` → `finished(report_pending=<reason>)` | no grace for the debrief; the debrief is server-side, so a lost heartbeat doesn't interrupt it |
| `finished` | `proposal_ready` → `proposed`; `submitted` → `scored`; sweeper `stale_30d` → `incomplete` | `proposal_ready`/`submitted` are driven by m6a-03 |
| `proposed` | `submitted` → `scored`; sweeper `stale_30d` → `incomplete` | |
| any non-terminal | `abandon` → `abandoned` | explicit; terminal |

- **Timed alphabet** (the events the sweeper fires from the clock alone; the model check's liveness runs over exactly these):
  `setup_expired`, `preflight_timeout`, `connect_timeout`, `heartbeat_lost` (`live`, `resuming`), `idle`, `rail_done`, `grace_expired`
  (incl. the final probe, which can't clear without a fresh lease), `resume_expired` (`paused`, `resuming`), `debrief_done`,
  `stale_30d`. Heartbeat age is measured from `max(last_heartbeat_at, state_since)`, so entering a state never times out on an old
  heartbeat. Everything else (`start`, `preflight_ok/fail`, `segment_open`, `interrupt`, `cleared`, `finish*`, `end_without_feedback`,
  `save_later`, `auth`, `resume*`, `still_dry`, `cap_*`, `proposal_ready`, `submitted`, `abandon`) is a learner, brain or provider event.
- **Voice rows** (`held`, `bridging`, the voice `connecting` offers) are [m6b-01](sprint-m6b-01.md)/[m6b-02](sprint-m6b-02.md)'s,
  with their own timed exits; they extend the model check to `mode=voice`. This sprint's table has no edge into either.

- **Named sets** (exported, used by m6a-02/03/04 and the admin CLI; pinned by the table test):
  **running** {`live`, `held`, `bridging`, `wrapping`} (the clock runs); **locked** {`connecting`, `live`, `held`, `bridging`,
  `interrupted`, `resuming`, `wrapping`} (coach chat refused); **editable** {`live`, `held`, `bridging`} (m6a-04's snapshot and
  Run routes accept edits); **live** {`preflight`, `connecting`, `live`, `held`, `bridging`, `interrupted`, `resuming`, `wrapping`}
  (the release check); **terminal** {`scored`, `incomplete`, `abandoned`}.
- **Pause limit:** a transition that would be the **4th** pause ends the interview: `finished(cut_short=pause_limit)` when ≥ 75 %
  of the rail is done (it can still be scored), else `incomplete`. (T6 doesn't say — record it in the ADR-0032 update, task 8.)
- **Clock** (`clock.go`): coach is the single writer. `active_ms` accumulates only in the running set; `live_since` is set on entry
  and folded into `active_ms` on exit. Rail = manifest `mock.rail` × `time_multiplier` (× 0.33 for an owner-only `qa_slice`, task 4);
  `minutes = 60` appends a follow-up phase of (60 − rail total) minutes whose goal is the extra `phase_goals` entry. Phase =
  f(active time, rail). assessment's `deadline_at` is **not** authoritative for text/voice mocks.
- **Deterministic checkpoints** (free, no AI call) at every phase boundary, interruption, pause, SIGTERM and finish: phase,
  per-phase ms, `upto_turn_seq`, hints given, `run_count` + `last_verdict_class`, and the code SHA + ≤ 16 KiB head read through a
  `CodeSource` interface (nil → NULL until m6a-04's snapshot store implements it).
- **Lease + liveness:** `POST …/attach {client_id}` takes the lease (a second tab takes it over; the old tab's calls get 409
  `lease_lost` → "live in another tab", a `notice` event). `POST …/heartbeat {client_id, visible, last_input_at}` ≤ 1 / 10 s.
  Sweeper: no heartbeat for 30 s → `interrupt(network)` in `live`, `connect_timeout` in `connecting`, back to `paused` in `resuming`;
  no candidate turn and no code change for **3 min × multiplier** with `visible=false` → `interrupt(idle)`;
  `POST …/interrupt {reason: tab_closed}` is the `sendBeacon` target (accepts `text/plain`; the gateway exemption is task 4).
- **Sweeper** (`sweeper.go`): one coach goroutine, tick 5 s, guarded by `pg_try_advisory_lock` so two pods during a rollout never both
  run; batches with `FOR UPDATE SKIP LOCKED`, every transition CAS. Jobs — one per timed event above: grace expiry (calls the
  optional `Prober` once at `grace_until` — nil until m6a-02; a pass clears only with a fresh lease), heartbeat (`live`,
  `resuming`) and idle, `preflight` > 2 min → `setup`, `connecting` > 2 min or heartbeat lost → `interrupted(network)`,
  `resume_by` → `incomplete` (**`paused` and `resuming`**), `finished`/`proposed` > 30 d → `incomplete`, `setup` > 1 h →
  `abandoned`, phase-boundary checkpoints (+ an `OnPhase` hook m6a-02 uses for cues), rail end → `wrapping`, debrief end →
  `finished`, and (daily) the retention purge (task 5). **Skips tombstoned accounts** (`coach.erased_account`, l-01).
- **SIGTERM:** coach's shutdown writes a `sigterm` checkpoint for every interview in the running set before the HTTP server drains
  (text: the interview stays `live`; the clock is in Postgres, so a 5–10 s restart costs nothing).
- **`pause_exposure`** ([t6 §4 failsafe 2](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes),
  [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) practice/gateway row) — **through the gateway,
  no new in-cluster caller:**
  1. **Push:** the gateway's arena handlers (m3-09: arena GET, reveal, submit, history) check the account's current interview
     (coach `GET /interviews/active`, cached per request); when the item is in its `items` and the state is in the **live** set or
     `paused` — the arena is never blocked (D17), so any access while the interview runs counts, not only in `paused`; after
     `finished` the transcript is frozen and access no longer matters — they call
     coach `POST /internal/interviews/{id}/exposure {kind}` (JWT aud=coach, account from the token; `ON CONFLICT DO NOTHING`). If
     that call fails the arena action still proceeds (D17) and the gateway logs WARN; the backstop covers reveal/submit.
  2. **Backstop at resume and at finish:** the gateway reads practice's `user_problem_state.arena_revealed_at` (m3-08) and judge's
     arena submissions for the item (m3-14 history) since `started_at`, over edges it already has (gateway → practice, gateway →
     judge), and forwards any hit as `source=backstop`. **Limit:** `arena_revealed_at` records the **first** reveal ever
     (`COALESCE(arena_revealed_at, now())`, m3-09), so the backstop catches a reveal only for an item never revealed before the
     interview; a re-reveal of an already-revealed item is seen only by the push. Submissions carry their own timestamps, so the
     backstop covers every in-interview submit. (Whether the picker should skip already-revealed items is [m6a-04](sprint-m6a-04.md)'s
     call; note the gap in the PR hand-off.)
  3. Coach sets `interview.pause_exposure = true` and the caveat when any exposure row exists; the resume state shows "Viewed the
     solution during the interview". (m6a-03 makes it block `ai-byo`.)
- **Handlers** (`internal/coach/interview/handlers.go`, all `requireJWT`, account-scoped): `POST /interviews` (create in `setup` with
  gateway-supplied `items`), `GET /interviews/active` (`{id, state, items, coach_locked, editable, resume_by}` or `{active:null}` — the
  "current-interview read" m6a-04's `InMock` and lock derivation consume), `GET /interviews/{id}` (state, clock, rail, phase,
  `grace_until`, `resume_by`, pauses, caveats, consents missing), `PUT /interviews/{id}/consent`, `PUT /interviews/optin`,
  `POST /interviews/{id}/{start|attach|heartbeat|interrupt|pause|resume|finish|abandon}`, `GET /interviews/{id}/resume-state` (the free
  deterministic state: rail with elapsed vs budget per phase and total remaining, hints, last code head, run count and last verdict
  class, pauses and reasons, an estimate to finish), `DELETE /interviews/{id}/turns/{seq}`, `DELETE /interviews/{id}`,
  `PATCH /interviews/{id}/retention`, internal `POST /internal/interviews/{id}/exposure`. Typed errors: 409 `interview_active` (with
  id), 409 `illegal_transition {state, event}`, 409 `consent_required {kinds}`, 409 `lease_lost`, 410 `resume_expired`, 422
  `cap_required`, 429 `interview_daily_cap` with `Retry-After`, 503 `interviewer_unavailable`.

**Tests.** `fsm_table_test.go` (every row, guards, illegal pairs → typed error, the named sets); **model check**
`fsm_modelcheck_test.go`: BFS over the abstract state × (`pause_count`, grace flags incl. the re-prime count and "entered by
`cleared`", lease fresh/stale, `mode`) generated from the table, asserting (a) every text state reachable in `mode=text` and
`held`/`bridging` unreachable in text (m6b-02 adds the voice rows and the `mode=voice` check), (b) terminals absorbing, (c)
**liveness over the timed alphabet listed above**: the graph restricted to those events is acyclic over the abstract state and its
only sinks are terminals — so from every non-terminal state the timed events alone reach a terminal (setup → `abandoned` in 1 h;
`preflight` → `setup` → `abandoned`; `connecting`/`live`/`resuming`/`interrupted` → `paused` → `incomplete` by `resume_by`, at most one
grace plus 24 h after the first pause; `wrapping` → `finished`; finished/proposed → `incomplete` in 30 d), (d) ≤ 3 pauses on every
path, (e) the running and locked sets as specified, (f) both `end_without_feedback` edges (the ≥ 75 % and the ≥ 90 % quota default)
reach `finished(report_pending=quota)` without a `wrapping` step;
**property test** `fsm_property_test.go`: seeded `math/rand/v2` random walks (10k × 200 steps; no new dependency) of learner,
provider and timer events against the real transition function with a fake clock — invariants after every step: `resume_by` never
moves once set, `active_ms` is monotone and never grows outside the running set, ≤ 1 open segment, one `state` event per transition
with strictly increasing `seq`; and every walk, once its random prefix ends, driven for 31 d of fake time by **timed events alone**
(no learner, brain or provider event) ends terminal; a real-Postgres variant
(`XLEARN_TEST_DATABASE_URL`, 200 walks) drives the store's CAS updates and the sweeper; two sweepers on one DB; handler tests per
route and error.

### 3 · Manifest [X]

Sources: [t6 §9 P0 row](../research/t6-realtime-interviewer.md#9-phased-plan), [t6 §6](../research/t6-realtime-interviewer.md#6-assessment)
(evidence enum, never-list, DSA dimension table), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T0 row),
[ADR-0026 §1](../../adr/0026-per-course-extensibility-model.md#1-a-course-is-data-a-capability-is-code), [t1 §2](../research/t1-content-data-model.md#2-entity-map) (rubric definition: public band descriptors).

- `internal/course/manifest.go` + `validate.go` + the JSON schema under `curriculum/_schema/` gain (additive; unknown fields stay
  rejected):
  - `mock.interviewer{persona (≤ 600 chars), voice_default, phase_goals[] (public, one per rail phase + an optional follow-up),
    max_minutes (≤ 60), debrief_minutes (3), camera ("self_view_optional")}`;
  - `mock.rubric.dims[].evidence[]` ⊆ the closed enum `transcript | code | judge | timeline | hints`;
  - `mock.rubric.dims[].ai ∈ {propose, self_only, self_only_voice}` (per dimension; `self_only_voice` = AI proposes in text, the
    learner scores it in voice);
  - `mock.rubric.dims[].descriptors` — five public band descriptors (1–5), unless [m1-01](sprint-m1-01.md) already added them.
- **DSA values** (`curriculum/courses/dsa/course.json`), from the t6 §6 table: communication `[transcript, timeline]`
  `self_only_voice`; problem_understanding `[transcript, timeline]`; brute_force `[transcript]`; optimisation `[transcript, judge, code]`;
  code_quality `[code]`; edge_cases `[judge, transcript, code]`; complexity `[transcript, code]` (all `propose` except
  communication); persona from coach's DSA persona; `phase_goals` from the v1 rail prompts (`internal/assessment/mock.go`);
  `max_minutes 60`, `debrief_minutes 3`, `voice_default "marin"`. Descriptors are drafted from the v1 Mock copy and the frozen
  AB13 board — flag them in the PR description as public content (no pack material).
- **Never-list lint** (`internal/course/neverlist.go` + test; runs in the public CI and at load; the list is **exported** for
  m6a-02's output scan and m6a-03's deny-lexicon): dimension ids, labels, descriptors and `phase_goals` must not match the
  never-list (tone, confidence, nervousness, enthusiasm, emotion, accent, pronunciation, fluency, filler, pace / words per minute,
  face, eye contact, expression, posture, appearance, background, personality, culture fit, honesty, demographics); the evidence
  enum has no `audio`/`video`/`prosody`/`face` member (a test proves the schema rejects them).
- Golden test: the DSA golden is regenerated (`-update`) with only the additive `mock` fields changed; the diff is reviewed in the PR.
  Item selection fields (e.g. `mock.selection`) are [m6a-04](sprint-m6a-04.md)'s if its picker needs them.

### 4 · Caps (L19), coach lock, gateway routes [X]

Sources: [ADR-0035 §4 L19, L6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory), [ADR-0033 §12 row 14](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2)
(aud=coach), [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (T-1/T-3),
[t6 §4 coach lock](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 locked-mode row).

- **Caps** (`internal/coach/interview/caps.go`; constants exported and mirrored in the docs):

  | Cap | Enforcement |
  |---|---|
  | 1 non-terminal interview per account | partial unique index; `POST /interviews` while one exists → 409 `interview_active {id}` — except a `setup` row, which is returned (idempotent) |
  | ≤ 2 starts per account per UTC day (L18's day rule) | a start **counts at `preflight_ok`** (`started_at` set); `start` checks the count and returns 429 `interview_daily_cap` + `Retry-After` to the next UTC midnight when 2 already started today. A `preflight_fail` or `preflight_timeout` (bad key, wrong model, dry credit) costs no start — the cap test pins this (two failed pre-flights, then two real starts, then 429) |
  | mandatory $ cap | `cap_micros` required at `start` (422 `cap_required`; NULL until then, see task 1's CHECKs); thresholds 85 % → `wrapping`, 100 % → `finished(cut_short=cap)` applied by `AddSpend` (m6a-02 calls it) |
  | ≤ 1 snapshot / 2 s, ≤ 64 KiB (L6) | constants `SnapshotMinInterval`, `SnapshotMaxBytes` here; enforced by [m6a-04](sprint-m6a-04.md)'s snapshot route (gateway typed 413, coach 429) |
  | mock Run ≤ 1 / 10 s | constant `MockRunPace` here; enforced judge-side by m6a-04 (L19 "coach (+ judge)") |
- **Coach lock (local check):** coach's `POST /chat` returns m1-07's 409 `coach_paused {reason: "mock"}` — before any key decrypt
  or provider call — while the account has an interview in the **locked** set; unlocked in `setup`, `preflight` (the item isn't
  shown yet), `paused` (owner spec: the item stays withheld through m6a-04's `InMock` and arena access is recorded), `finished`,
  `proposed` and terminals. The register's "locked in every non-terminal state except `paused`" is read as the in-interview states
  of t6 §4; the table test pins the set. The gateway-side derivation for text/voice (`coach_locked` from `GET /interviews/active`)
  is m6a-04's.
- **Gateway routes** (`internal/gateway/interview.go`): `/api/interviews/*` proxies to coach with an **aud=coach** mint
  (`mintForCoachW`) — `/api/interviews/active`, create, the lifecycle routes of task 2, the turn delete, retention and consent;
  JSON routes on the 10 s client, none streams yet. `POST /api/interviews {course, difficulty, minutes, time_multiplier}` does a
  **placeholder server-side pick** (the first evaluable, non-retired item of the requested difficulty) that
  [m6a-04](sprint-m6a-04.md)'s `mockpick.go` replaces; the pick is never echoed before `live` (S1). Every route declares its
  `withhold` policy in `apiRoute` (m1-06's route-enumeration test) and is added to `docs/architecture/openapi.yaml` (the drift test).
- **Beacon exemption** (`internal/gateway/security.go`, [m1-04](sprint-m1-04.md) task 6): `navigator.sendBeacon` can't set headers,
  so it sends `text/plain`, which m1-04's mutating-`/api/*` check refuses with 415 `unsupported_media_type`. Extend m1-04's
  exemption list (today only the form POST `/api/auth/{provider}/start`) with **`POST /api/interviews/{id}/interrupt`**, and only for
  `Content-Type: text/plain` (the body is still parsed as JSON, ≤ 1 KiB, `reason` must be `tab_closed`). text/plain is a CORS-simple
  request, so on this route the `Sec-Fetch-Site` check stays and tightens: the header must be **present** and `same-origin` (every
  browser that has `sendBeacon` sends it); absent or `same-site`/`cross-site` → 403 `cross_site_request`. Every other route is
  unchanged. Tests: a same-origin text/plain beacon → 200 and `interrupted(tab_closed)`; the same body with `Sec-Fetch-Site:
  cross-site` or no header → 403; text/plain on any other interview route → 415.
- **Cohort gate (dark):** all `/api/interviews/*` return **404** unless `role ∈ {owner, tester}` (T-3 from session-validate,
  ADR-0033 §7) — a **T-1 code default** `interviewAudience = "cohort"` that [m6b-04](sprint-m6b-04.md) flips to `"all"` at v2.1.0.
  No T-2 env is added. List it in the status.md flag inventory (owner M6a, removal v2.1.0).
- **QA slice rail:** `qa_slice = true` (rail × 0.33) is accepted only for `role = owner`; such interviews can reach `finished` and a
  proposal but never `ScoreMock` (m6a-03 refuses; the owner abandons them) — used by m6b-04's fake-media run.

### 5 · Retention + erase [X]

Sources: [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (inventory, consent),
PRD R-MI8, [ADR-0027 §6](../../adr/0027-content-evalpack-and-user-data-model.md#6-account-erase-v20), [l-01](sprint-l-01.md) (coach erase handler, coverage test, tombstones).

- **Consent** (P0 text kinds, versioned constants in `internal/coach/interview/consent.go`): account-level `interviewer_optin`
  (unticked by default, with "how it works and what it evaluates"); per session `text_processing` ("My transcript and code are sent
  to my AI provider's text API for the interviewer, summaries and feedback"), `retention` (`30d` default \| `12m`),
  `ai_disclosure_ack` ("The interviewer is an AI. It can't see you."), `assessment_scope_ack` ("Assessment uses what you say and
  your code, never your voice, face, accent or appearance."); `resume_reaffirm` on each resume. Voice kinds are M6b's. `start`
  without all of them at the current version → 409 `consent_required {kinds}`.
- **Retention:** on `scored`, set `purge_at = scored_at + 30 d` (or + 12 months when the learner chose `12m`); `incomplete`,
  `abandoned` and an unscored `finished` → `ended_at + 30 d`. The daily purge deletes turns, events, checkpoints, pause briefs,
  exposure, hints, m6a-03's proposals and **every table m6a-04 adds** (snapshot timeline, run echoes), and sets `purged_at`; the
  `interview` header row (ids, state, timestamps, money) stays until erase. Segment rows are purged at 90 d. Consent rows live as long
  as the account.
- **Per-turn delete** (`DELETE /interviews/{id}/turns/{seq}`, candidate turns only, any state before purge): blanks `text`, sets
  `deleted`, keeps `seq`; adds the caveat `transcript_edited` (m6a-03: blocks `ai-byo`). **Per-session delete**
  (`DELETE /interviews/{id}`, terminal only): purge now.
- **Erase:** add every new table to coach's erase list in the l-01 handler (by `account_id`; `admin_audit` exempt with a reason — it
  holds no account data) — the **coverage test** turns CI red otherwise. The sweeper and every async path skip tombstoned accounts.
  Test: seed two accounts with a full interview each, erase one, assert zero rows for it and untouched rows for the other.
- **Logging:** no transcript, code, event data or consent text in any log line (access logs keep method, route, status, ids); extend
  M4's canary log test ([m4-07](sprint-m4-07.md)) to the new coach and gateway routes.

### 6 · `coach admin interviews` [X]

Sources: [ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) ("from M6: no live interviews"),
[ADR-0035 §6](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#6-amendments) (opscheck counters dropped; the runbook check stays), m1-04's admin CLI pattern.

- `cmd/coach/main.go` dispatches `coach admin …` before `run()` (like `-version`); code in `internal/coach/admin/`, distroless-safe
  (static Go, `flag`, no shell); DB from the pod's env.
- `coach admin interviews --live` prints one line per interview in the **live** set — id, state, since, mode — and a final `live=N`;
  **no account id, no content**. `--all` adds `paused`/`finished`/`proposed` with `resume_by`. `show <id>` prints header fields only.
  `abandon <id> --reason <text>` (break-glass for a stuck interview before a tag) runs the CAS transition. Every verb writes
  `coach.admin_audit` in the same transaction; output goes to the exec'd terminal.
- Runbook `docs/runbooks/interviewer.md`: the call (`ssh sujaykumar-vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin
  interviews --live'`), the release-checklist use (**empty before every tag from the first patch that carries this sprint**), stuck
  interview handling, the sweeper's jobs, retention and erase.

### 7 · Policies for new internal callers [I]

Sources: [ADR-0035 §2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first) (standing rule),
[mi-11](sprint-mi-11.md) (the `xlearn` egress matrix), [mi-03](sprint-mi-03.md) (MI-5a: `/internal/*` admits the `xlearn` namespace).

- With task 2's design, the new edges are **gateway → coach** (`/interviews/*`, `/internal/interviews/*`) and gateway → practice /
  judge reads that already exist in mi-11's matrix; **coach calls nobody new**. Verify against the live `../infra/apps/xlearn-*.yaml`
  policies (read-only) and record "no NetworkPolicy change" with that evidence in the PR and the status.md decisions log. No NATS
  subject, stream or consumer is added, so no ACL PR.
- **Only if** the implementation ends up with coach calling practice or judge `/internal/*` (e.g. for the exposure backstop): open a
  `../infra` PR extending coach's egress and the callee's ingress (chart 0.3.0 knobs in `apps/xlearn-*.yaml`), GitOps only,
  **merged before the M6a patch tag** ([m6a-06](sprint-m6a-06.md)); smoke the path after the merge.

### 8 · Docs [X]

- `docs/architecture/services.md` (coach owns the interviewer; the sweeper; the named sets), `api.md` + `openapi.yaml`
  (`/api/interviews/*`, cohort-gated), `data-model.md` (the coach tables with class and retention).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md): a dated **"Update — <date> (m6a-01)"** section (append-only) for the
  build-time calls: the pause-limit end rule, exposure recorded for the whole interview, the locked/editable/live sets, the timed
  exits of `preflight`/`connecting`/`resuming` (the timed alphabet), `end_without_feedback`, a start counted at `preflight_ok`, and the
  tab-close beacon exemption. A new ADR only
  if a call contradicts an Accepted decision (check peers' PRs and worktrees for the next free number first).

## Acceptance criteria

- [ ] **FSM property tests: every path terminates**; the model check proves liveness over the timed alphabet from every non-terminal
      state (incl. `preflight`, `connecting` and `resuming`), ≤ 3 pauses, absorbing terminals and the running/locked sets;
      **`incomplete` after 24 h** from the first pause (from `paused` or `resuming`), with a fixed `resume_by`.
- [ ] **Caps enforced with typed errors**: 1 non-terminal (409 `interview_active`), ≤ 2 starts/day counted at `preflight_ok` (429 + `Retry-After`; a failed pre-flight costs none), mandatory $
      cap (422); coach chat 409 `coach_paused {reason:"mock"}` in the locked set and allowed in `paused`; snapshot and Run constants exported.
- [ ] Deterministic checkpoints at every phase boundary, interruption, pause and SIGTERM; `GET …/resume-state` returns the free state with
      no model call; grace expiry → `paused`; `resume_by` → `incomplete` via the sweeper (incl. two sweepers on one DB).
- [ ] Every FSM transition appends one `state` event with a strictly increasing `seq`.
- [ ] Arena view/reveal/submit on an interview item sets `pause_exposure` (push and backstop tests).
- [ ] Manifest fields validated; DSA values in the golden; never-list lint green and proven to fail on a planted term.
- [ ] Retention purge and per-turn/per-session delete tested; erase coverage test green with every new table.
- [ ] `coach admin interviews --live` lists exactly the live set with no PII; audit rows written.
- [ ] The same-origin `text/plain` tab-close beacon reaches `interrupted(tab_closed)`; a cross-site or header-less one gets 403.
- [ ] All `/api/interviews/*` return 404 for a `learner`; OpenAPI drift and route-enumeration tests green; no new NetworkPolicy or NATS
      ACL needed (recorded) — or task 7's infra PR is merged (its own PR, before the M6a patch tag).

## Release

**Merge only — ships dark in the next v2.0.x patch** (normally [m6a-06](sprint-m6a-06.md)'s M6a patch;
[ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme): after `v2.0.0` everything but a GA flip is a
patch). Dark-safe by construction: cohort-gated routes (404 for everyone else), an expand-only coach migration, a sweeper idle on
empty tables, the 503 `interviewer_unavailable` guard until m6a-02, no new NATS subject, no new pod or limit change (coach stays
250m / 128 Mi until [mi-13](sprint-mi-13.md)). **From the first patch that carries this sprint, every release checklist's "from M6:
no live interviews" line is run with `coach admin interviews --live`.** No snapshot (not a contract, erase or GA tag); rollback floor
unchanged.

## Definition of Done

CI green (`go test -race ./...` incl. the model check and property tests, real-PG integration via `XLEARN_TEST_DATABASE_URL`,
`sqlc diff`, the migration lint, the erase coverage test, the manifest golden + never-list lint, the OpenAPI drift and
route-enumeration tests, web build unchanged) · e2e lane green · merged via PR (squash) · task 7 recorded (or its infra PR merged on
its own, before m6a-06's tag) · statuses updated (this file + [`../status.md`](../status.md): Sprint board, M6a row "core merged (dark)",
L19 row, flag inventory `interviewAudience=cohort`, the AB13/AB24–AB28 rows "frozen (PR #, date)" if not yet set, the release section's
live-interview check, decisions log) · ADR-0032 update written · runbook written.

## Risks / watch-outs

- **State explosion** — keep one table as the only source and model-check it; don't add booleans that duplicate a state (grace is
  `interrupted` + `grace_until`, not a flag). Voice-only states stay unreachable in text by construction.
- **Two sweepers during a rollout** — the advisory lock plus CAS transitions make a double run harmless; tested.
- **Clock drift across restarts** — the clock lives in Postgres (`active_ms`, `live_since`), never in process memory.
- **`resume_by` must be absolute** (first pause + 24 h); a later pause never extends it (property-tested).
- **Exposure false negatives** if the gateway push fails — the backstop covers every submit and the **first-ever** reveal from durable
  records; views and re-reveals of an item revealed before the interview rely on the push (WARN logged). Accepted: it only affects the
  learner's own honor label.
- **Seams shared with m6a-02/m6a-04** (event log, `CodeSource`, `OnPhase`, `Prober`, the named sets, the current-interview read) —
  export small interfaces, document them in the PR, and let the later sprint adopt rather than duplicate.
- **Manifest golden churn** — a peer editing the DSA manifest at the same time forces a regenerate; serialize (entry gate).
- **Memory** — no new pod; ≤ 256 KiB of transcript per interview; coach's 128 Mi limit is unchanged here
  ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- **Dark code in a content-wave patch** — if a peer tags a v2.0.x patch after this merges, it ships this sprint dark; the cohort gate
  and the 503 guard (no brain yet) keep it inert.
