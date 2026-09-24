# Sprint m1-07 — Coach D27 assist capture, mode gate, L18 caps + AB01 → v1.7.0

> **Milestone:** M1 — spine (**M1b**, the tag that ships all of it) · **Track:** product · **Order:** 27
> **Prereqs:** [m1-03](sprint-m1-03.md) · [m1-04](sprint-m1-04.md) · [m1-05](sprint-m1-05.md) · [m1-06](sprint-m1-06.md) · [m1-10](sprint-m1-10.md) · design [ds-m1-01](sprint-ds-m1-01.md) (AB01 frozen) · soft: [mi-04](sprint-mi-04.md) (MI-5b)
> **Unblocks:** [m1-08](sprint-m1-08.md) (the M1c contract needs `v1.7.0` live, with no reader or writer of a dropped column)
> **Release action:** **tag `v1.7.0`** (indicative: the next free minor at tag time, [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6) — rollback floor after: **1.6.0** (v2 envelopes in the log)
> **Calendar:** week 4 (2026-10-17 → 10-23) — tag before the Sat 2026-10-24 host window, never during it · owner event after the tag: `ev-owner-role` (~2 min)
> **Execute with:** [`../prompts/prompt-m1-07.md`](../prompts/prompt-m1-07.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | D27 per-problem assist capture (409 confirm, practice assist endpoint, Assisted ceiling, event field) | X | ⬜ |
| 2 | Mode gate `locked`/`attempt`/`review`/`general` + `coach-prompt@2` + per-course persona | X | ⬜ |
| 3 | L18 BYO caps (20/min, 2 streams, 300/day, history 20 turns / 32 KiB) | X | ⬜ |
| 4 | Coach UI states (AB01) + outcome-step cap line | X | ⬜ |
| 5 | M1b exit + M1c readiness (golden = v1; no reader/writer of a drop-list column) | X | ⬜ |
| 6 | Tag `v1.7.0` (release checklist) | X | ⬜ |
| 7 | Owner role set once (`ev-owner-role`) | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + M1 milestone + tag → floor row + flag inventory).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB01 frozen** — [ds-m1-01](sprint-ds-m1-01.md)'s PR merged by the owner (the board lives at `design-system/screens/v2/AB01-*.html`); the artboard row in [`../status.md`](../status.md) reads "frozen (PR #, date)".
- [ ] **All of M1b merged on `main`:** [m1-03](sprint-m1-03.md) (course resolution, producers emit v2, writers stop writing the drop-list columns), [m1-04](sprint-m1-04.md) (roles, sessions, admin CLI, L7 guard, L3, CSP), [m1-05](sprint-m1-05.md) (limits, public floor), [m1-06](sprint-m1-06.md) (`withhold()`, Markdown renderer, revision v2), [m1-10](sprint-m1-10.md) (keys, catalog, AEAD, keyring, `store:false`, usage; `is_default` no longer written).
- [ ] **`v1.6.0` live** ([m1-02](sprint-m1-02.md)) — the floor this tag records; every consumer decodes the v2 envelope.
- [ ] **NATS topology unchanged** since `v1.6.0` (`topology.go` golden / subject-registry test shows no new stream, consumer or subject) — or, if [mi-06](sprint-mi-06.md) N1 is live and something changed, the re-rendered ACL PR is merged in `../infra` first.
- [ ] Soft (recommended, not blocking): **MI-5b live** ([mi-04](sprint-mi-04.md)). [ADR-0033 §11](../../adr/0033-invite-only-admission-and-owner-admin.md#11-admin-console-isolation-mi-5b) recommends it before the M1 Markdown renderer (m1-06) ships in this tag. If it isn't live, say so in the decisions log.

## Goal

Finish the coach's v2 behaviour and cut **M1b**. A coach chat about a problem whose **counted attempt is open**
needs an explicit confirmation, is recorded on the attempt **before** anything reaches the provider (fail-closed),
and caps that attempt at **Assisted** (D27, per problem). The coach's behaviour mode becomes a server-side gate
with four modes — `locked` during a live mock (touches from M2a), `attempt` with no pattern / concepts / solution
facts, `review`, `general` — under a versioned prompt `coach-prompt@2` with the course persona. BYO use gets the L18
caps with typed 429s. The panel shows the AB01 states. Then verify M1b end to end (golden = v1 except the
expected-change list in task 5; no reader or writer of any M1c-drop column) and tag **`v1.7.0`**.

## Scope

**In**
- **D27 capture** ([t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) as overridden by t5 §12 D27; [feasibility D27](../feasibility.md#decisions-log-newest-first)): gateway 409 `assist_confirm_required` (skipped once the attempt already carries `coach_assist_at`), a non-consuming coach L18 admission probe, then practice `POST /attempts/{id}/assist` (idempotent) called **before** forwarding, 503 fail-closed, the practice Assisted ceiling, `problem_solved` v2 `assist{hint, coach}`, the honesty copy.
- **Mode gate** ([ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes)): practice `GET /attempts/open` (replaces the coach path's `/state/{id}` call), assessment `GET /mocks/live` (the mock lock), m1-06's `withhold()` account state for due touches / never solved; `coach-prompt@2` (frame → persona → mode rules → context); `coach_message.prompt_v`, `attempt_id`; the mode (and the open attempt's assist state) **exposed to the SPA** on the chat response and on `GET /api/coach/thread`, so AB01's mode chip and capped note can render.
- **L18** ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory)): 20 msg/min, 2 concurrent streams, 300 messages/day per account, history 20 turns / 32 KiB; typed 429 with `Retry-After`; a read-only `GET /coach/admission` probe for the gateway.
- **Coach UI (AB01 F1–F10):** mode chip, assist confirm, capped note + Problem HUD chip, fail-closed notice, paused during mocks (touches later), review / general modes, the three L18 notices, cut-short marker with [Continue], provider-limited notice, "Your AI coach (your key)" naming; plus one text line on the Problem outcome step when the grade was capped.
- **M1b exit + M1c readiness** checks; **tag `v1.7.0`**; `docs/v2/status.md` tag → floor row and the **flag inventory** start (`SIGNUP_MODE` as an operating mode).

**Out**
- Keys, catalog, AEAD, keyring, `store:false`, usage/cost, Settings key UI → [m1-10](sprint-m1-10.md) (merged before this sprint).
- The interviewer, derived realtime credentials, the `interview` default key in use → M6a ([m6a-01](sprint-m6a-01.md), [m6a-02](sprint-m6a-02.md)).
- Platform AI and its "xLearn AI (included)" naming → M4 ([m4-06](sprint-m4-06.md)); only the BYO name ships here.
- `review`-mode context beyond the pattern (t5 §9 table): **pointer notes** → [m4-03](sprint-m4-03.md) (its "Coach review mode" bullet). The **final submission (≤ 16 KiB)** and the **xLearn AI feedback** have **no owning sprint yet** — deferred, and flagged for the register (proposed: [m4-03](sprint-m4-03.md), alongside the pointer notes, since the gateway already calls judge there). The `review` slots stay empty until then.
- Live **touches** as a `locked` source → M2a ([m2-01](sprint-m2-01.md) engine, [m2-04](sprint-m2-04.md) UI). `/attempts/open` already carries `purpose` on each attempt (`"course"` until M2a); m2-01 reports an open touch as `purpose: "touch"` in the same list.
- The v1.5.1 P0 coach fixes (typed quota errors, `max_tokens`, onboarding key save) — already shipped.
- Dropping any column or CHECK → [m1-08](sprint-m1-08.md) (M1c contract, `v1.8.0`).

## Tasks

### 1 · D27 per-problem assist capture [X]

Sources: [t5 §9 "D18 capture"](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) with **§12 D27 overriding it** (per problem only; the "no mock or realtime while any
counted attempt is open" rule is **dropped**), [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes),
[ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) grade table (Assisted = "pass after the hint, or with AI-coach help during the attempt").
ADR-0031 stays *Proposed* until the WIF spike (accepted in [mi-12](sprint-mi-12.md)); its §7 coach behaviour rests on
the settled owner decision D27, so build it now.

**practice** (`internal/practice/`)
- Migration `internal/practice/store/migrations/0000N_m1b_coach_assist.sql` (next free version at rebase; expand
  only, no contract marker): `attempt.coach_assist_at timestamptz NULL`.
- `POST /attempts/{id}/assist` (JWT-scoped, `requireJWT`): the attempt must belong to `sub` and be open
  (`ended_at IS NULL`), else 404 / 409 `attempt_closed`. One statement,
  `UPDATE practice.attempt SET coach_assist_at = COALESCE(coach_assist_at, now()) WHERE id = $1 AND account_id = $2 AND ended_at IS NULL RETURNING coach_assist_at`
  — idempotent; returns `{attemptId, coachAssistAt}`.
- `LogOutcome` (`internal/practice/store/store.go`): the lock point is the outcome's conclusion (`lock_at` =
  the outcome row's `logged_at`). If `coach_assist_at IS NOT NULL AND coach_assist_at <= lock_at`, a self-reported
  `clean` or `rough` is recorded as **`assisted`** (`miss` stays `miss`); `below_clean` is computed from the
  effective value. The response carries `cappedBy: "coach"` when it applied. v1 has one counted attempt per
  problem (a solved problem refuses a new outcome), so the clamp is exact.
- `problem_solved` data gains `"assist": {"hint": <stage_reached ≠ attempt>, "coach": <coach_assist_at set>}`.
  It is **additive** on an existing subject (no new subject, no ACL change); m1-02's `DecodeEnvelope` ignores
  unknown fields, so `v1.6.0` consumers (this tag's R-b floor) accept it. Document it in
  [`../../architecture/events.md`](../../architecture/events.md).
- `GET /state/{problemId}` adds `coachAssistAt` so the Problem outcome step can show the cap (task 4).

**gateway** (`internal/gateway/coach.go`; new `internal/gateway/coach_gate.go` + `coach_gate_test.go`)
- In `handleCoachChat`, after the mode/state lookup (task 2) and before any call to coach, for a
  `problem:<id>` context with an open counted attempt `A` (`purpose: "course"`) on **that** problem:
  - `A.coachAssistAt` **already set** → no confirmation (the attempt is already capped; nothing is left to
    protect, and a reload must not re-prompt): forward with `X-Coach-Attempt: A.id`, whatever `assist_ack` says;
  - `A.coachAssistAt` unset and body `assist_ack` ≠ `A.id` → **409** `{"error":{"code":"assist_confirm_required","attemptId":A.id,"problemId":…}}`; nothing sent, nothing persisted;
  - `A.coachAssistAt` unset and `assist_ack` = `A.id` → first coach **`GET /coach/admission`** (task 3, read-only
    L18 probe). A 429 is relayed as is (typed code, `Retry-After`) and the attempt is **not** capped. Then practice
    `POST /attempts/{A.id}/assist`; any error or non-2xx → **503** `assist_unavailable`, nothing forwarded
    (**fails closed**);
  - then forward with header `X-Coach-Attempt: A.id`; strip `assist_ack` from the forwarded body.
- Chats in **another** problem's or page's context are untouched — the accepted honor-based bypass (D27).
- A **provider** failure after the assist record still leaves the attempt capped (t5 §9: "the learner chose to
  ask"). t5 §9 does **not** cover xLearn's own limiter, hence the probe. The one residual case is a race: the
  probe passes, then a concurrent send takes the last slot and the chat gets a 429 after the record. It happens at
  most once per attempt (the record is idempotent and later chats skip the confirm). Accept it and record it in the
  decisions log as **this sprint's decision**, not as t5's.

**Tests:** practice store tests (idempotent assist; clamp matrix clean/rough/assisted/miss × assisted/not;
`assist` in the payload); gateway table tests on the existing httptest harness (`newCoachHarness` in
`internal/gateway/coach_test.go`, with fake practice / review / assessment / coach servers): no ack → 409 and coach
never called; ack + practice 500 → 503 and coach never called; ack → admission probe, then practice called exactly
once, both **before** the chat is forwarded; admission 429 → 429 relayed with `Retry-After` and practice never
called; `coachAssistAt` already set → no 409, practice not called, forwarded with `X-Coach-Attempt`;
other-problem context → no 409; wrong `attempt_id` in the ack → 409.

### 2 · Mode gate + `coach-prompt@2` + per-course persona [X]

Sources: [t5 §9 table](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2), [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes),
[t1 §4 coach + §10 withhold](../research/t1-content-data-model.md), [ADR-0026](../../adr/0026-per-course-extensibility-model.md) (manifest persona).

**Inputs (one lookup per chat, run in parallel with short per-call timeouts):**
- practice **`GET /attempts/open[?problem_id=]`** (new, JWT-scoped): `{attempts: [{attemptId, problemId, pathSlug, purpose, startedAt, stageReached, coachAssistAt}], problem?: {problemId, status, firstSolvedAt}}`.
  `purpose` is the constant `"course"` until M2a adds the `attempt.purpose` column
  ([t1 §2](../research/t1-content-data-model.md): `course | touch`); [m2-01](sprint-m2-01.md) then reports open touches in
  the same list as `purpose: "touch"` (no separate `touches` slot; this is the shape m2-01 and m2-04 test against).
  The coach path stops calling `/state/{id}` (`coach.go:294-302`); the route stays for its other callers.
- assessment **`GET /mocks/live`** (new, JWT-scoped; `internal/assessment/service.go`, `mock.go`, query in
  `store/queries/mock_session.sql`): the account's `status = 'live'` session with `deadline_at > now()`, or `{"live": null}`.
- m1-06's exported `itemState` / `live(s)` predicate (`internal/gateway/withhold.go`). m1-06 already strips
  pattern, concepts and facts from the coach context when `live(s) || !s.Solved` (its fix for `coach.go:313`). Build
  the context item's `itemState` from `/attempts/open` (`OpenAttempt`, `Solved`) plus review's due queue
  (`DueTouch`, as m1-06 does), and derive the mode from the **same** predicate, so withholding and mode can never
  disagree.

**Modes** (server-authoritative; `X-Coach-Mode` as today):

| Mode | When | Gateway | Prompt gets |
|---|---|---|---|
| `locked` | a live mock (assessment) or, from M2a, an open attempt with `purpose: "touch"` on the account; the course manifest's `coach.off_during` lists the kind | **409** `coach_paused` `{reason: "mock" \| "touch"}`; nothing sent or persisted | — |
| `attempt` | `live(s) \|\| !s.Solved` — open counted attempt, due touch, or never solved | no `pattern`, concepts or solution facts in the forwarded body (m1-06's strip, kept) | title, stage, recent outcome only (coach-side guard: fixes `internal/coach/prompt.go:84-86`) |
| `review` | concluded, no touch due, no open attempt | as today | + pattern (submission / AI feedback slots arrive in M3 / M4) |
| `general` | not an item (week, concept, dashboard, …) | adds `live_items: [{id, title}]` (problems with open attempts) to the body | a guard line naming those items off-limits |

- **Fail closed:** if practice, review or assessment can't answer, a chat returns **503** `coach_state_unavailable`.
  m1-06's "treat as live" fallback is right for withholding, but not enough here, because the D27 check needs the
  attempt id.
- **Mode to the client** (AB01 F1, F3, F6, F7 need it before and after a send; today `X-Coach-Mode` only goes
  gateway → coach, `coach.go:96`):
  - the gateway sets `X-Coach-Mode` on the relayed chat response (SSE headers, set before streaming);
  - `GET /api/coach/thread?context=` (`handleCoachThread`) gains
    `gate: {mode, reason?, attempt?: {attemptId, coachAssistAt}}`, from the **same** state lookup run read-only
    (no assist write, never a 409). If a lookup fails, `gate` is omitted and the thread still loads: the field is
    display-only, and the chat itself stays fail-closed;
  - `web/src/lib/settings.ts` types both. Document them in `openapi.yaml` / `api.md` (the drift test covers them).
- **Course:** the gateway resolves `path_slug` server-side (m1-03's resolution), applies the manifest's
  `coach.off_during` for `locked`, and sends `X-Coach-Course`. Coach reads `coach.persona` (≤ 600 chars) from the
  compiled manifest (`internal/course`, m1-01). DSA's persona is the course-specific lines of v1's
  `prompt.go:56-57` (m1-01), so the DSA prompt stays golden.
- **Prompt** (`internal/coach/prompt.go`): `const PromptVersion = "coach-prompt@2"`; order **frame → persona →
  mode rules (hard constraints) → context**; `pageContext.Pattern` is ignored unless the mode is `review`, whatever
  the body says (defence in depth).
- **Store** (`internal/coach/store/migrations/0000N_m1b_prompt_v.sql`, expand): `coach_message.prompt_v text NULL`,
  `coach_message.attempt_id uuid NULL` (m1-02 already added `path_slug`); `AppendMessage` writes all three for
  user and assistant rows.
- **Tests:** golden prompt snapshots per mode in `internal/coach/testdata/prompt/*.golden`; a test that an
  `attempt` prompt never contains the pattern even when the body carries it; gateway tests for each mode
  (live mock → 409 `coach_paused` and coach never called; solved + no due touch → `review`; concept page →
  `general` with the live-item guard); `X-Coach-Mode` on the chat response; `gate` on the thread `GET` per mode,
  with `attempt.coachAssistAt`, and omitted (thread still 200) when a lookup fails.

### 3 · L18 BYO caps [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L18 and L24; [t5 §9 "Usage display and limits"](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2).
Enforced **in coach** (`internal/coach/limits.go` + `limits_test.go`), checked in this order **before** the user
turn is persisted and before the key is decrypted, so a rejected request consumes nothing:

| Limit | Mechanism | Rejection |
|---|---|---|
| 2 concurrent streams per account | in-process counter, released in a `defer` when the stream ends or the client goes | 429 `coach_busy`, `Retry-After: 5` |
| 20 messages / minute per account | in-process token bucket (capacity 20, refill 20/min) | 429 `coach_rate_limited`, `Retry-After` = seconds to the next token |
| 300 messages / day per account (UTC day) | new `coach.message_quota_day(account_id uuid, day date, n int NOT NULL, PRIMARY KEY (account_id, day))` (expand migration); atomic `INSERT … ON CONFLICT (account_id, day) DO UPDATE SET n = message_quota_day.n + 1 WHERE message_quota_day.n < 300 RETURNING n` — no row ⇒ cap reached | 429 `coach_daily_cap`, `Retry-After` = seconds to the next UTC midnight |
| History | `ThreadHistory` reads the last 20 messages; `buildTurns` drops the oldest until ≤ 32 KiB, always keeping the current turn | — |

- **Admission probe** `GET /coach/admission` (JWT-scoped, read-only): runs the same three checks without
  consuming anything (streams < 2, the bucket holds ≥ 1 token, today's `n` < 300) and returns 204 or the same
  typed 429 with `Retry-After`. The gateway calls it only right before a D27 assist record (task 1), at most once
  per attempt. It is not an admission reservation, so the chat request still runs the real checks.
- Values are code constants (not env flags, so nothing joins the flag inventory). The v1.5.1 `provider_limited`
  429 stays distinct.
- **The day is the UTC day** (one clock, no identity lookup). AB01 F8's copy says the cap "resets at 00:00 in your
  timezone", which would be false for most learners. The SPA shows the reset as a **local clock time** computed from
  `Retry-After` ("It resets at 05:30 your time"). Flag that copy delta to the owner in the PR (the board is frozen).
  A per-account-timezone day would need the gateway to pass the account's timezone to coach: a follow-up, not this sprint.
- `relayStream` (`internal/gateway/coach.go:366`) copies `Retry-After` from coach's response (test).
- **Erase hand-off.** `message_quota_day` is per-user data. [l-01](sprint-l-01.md)'s coach erase list (today
  `api_key_config`, `coach_thread`, `outbox`) doesn't name it, nor m1-02's `coach.key_default`. Note both in
  [`../../architecture/data-model.md`](../../architecture/data-model.md) as erase-by-`account_id`, and add an explicit
  hand-off line to the `status.md` decisions log: "l-01 coach erase must also delete `coach.message_quota_day` and
  `coach.key_default` (verify against the schema)".
- **L24:** coach runs one replica and now holds per-minute and concurrency state in process — add coach to the
  scale-out-blocker list m1-05 started in [`../../architecture/services.md`](../../architecture/services.md).
- **Tests:** 21st message in a minute → 429 with `Retry-After`; 3rd concurrent stream → 429; 301st in a UTC day →
  429 (concurrent requests can't overshoot: race test on the atomic upsert); a rejected request writes no
  `coach_message` row; history trimmed at 20 messages and at 32 KiB; `/coach/admission` returns the matching 429
  in each exhausted state and consumes nothing (the bucket and `n` are unchanged after it).

### 4 · Coach UI states (AB01) [X]

Board: `design-system/screens/v2/AB01-*.html` (frozen in [ds-m1-01](sprint-ds-m1-01.md)); this sprint owns frames
**F1–F10** (m1-10 built F11–F15). Tokens and components from [`theme.css`](../../../design-system/theme.css) verbatim;
v1 reference [`Problem.dc.html`](../../../design-system/screens/Problem.dc.html) (coach panel). **The board's copy wins**
over any string quoted in this plan; the only sanctioned delta is F8's reset wording (task 3), flagged to the owner.
- `web/src/lib/settings.ts`: `CoachChatBody.assist_ack?: string`; `CoachChatError` gains `status`, `retryAfter`
  (from the header), `attemptId` and `reason`; the thread response's `gate` and the chat response's `X-Coach-Mode` (task 2).
- `web/src/components/Coach.tsx`, per AB01's frames:
  - **F1 / F6 / F7 mode chip** from `gate.mode` on thread load, updated from `X-Coach-Mode` after each send
    (Attempt / Review / General copy per the board); no chip when `gate` is absent. F7's off-limits note is display
    only (the guard itself is server-side).
  - **F2 assist confirm** on 409 `assist_confirm_required`: the board's card and buttons; on confirm resend with
    `assist_ack`. No client-side memory is needed: once recorded, the gateway skips the confirm (task 1), so a reload
    doesn't re-prompt.
  - **F3 assist recorded:** the panel note when `gate.attempt.coachAssistAt` is set or right after an acked send,
    and the **Problem HUD chip** (`TimerHUD` in `web/src/screens/Problem.tsx`, from `/state`'s `coachAssistAt`). After
    an acked send, the panel invalidates the `["problem", id]` query so the HUD refreshes.
  - **F4** on 503 `assist_unavailable` / `coach_state_unavailable`: the board's fail-closed copy; nothing was sent.
  - **F5 paused** on 409 `coach_paused`: banner, input disabled, reason (mock; touch from M2a).
  - **F8 typed 429s:** the three variants. `coach_rate_limited` counts down from `Retry-After`; `coach_busy` waits for the
    current reply; `coach_daily_cap` shows the reset as a local clock time (task 3). Reuse the frame's component; don't
    invent layout AB01 lacks.
  - **F9 cut-short marker** from the SSE `done.truncated` flag (the persisted note text stays for history), with
    **[Continue]**: it sends a fixed user turn ("Continue from where you stopped.") through the normal send path, so it
    counts toward L18 and goes through the same gate.
  - **F10 provider limited:** the board's notice, driven by m1-10's typed `reason`; the key toggle stays on.
  - **Naming:** "Your AI coach (your key)" in the panel header and FAB label (the Settings heading is m1-10's).
- `web/src/screens/Problem.tsx` outcome step: when `coachAssistAt` is set, Clean/Rough are shown as capped with the
  same honesty line, and a `cappedBy: "coach"` response is reflected in the recorded grade (text-only delta, no new layout).
- CSP (m1-04) is strict: no inline styles or scripts in the new states.
- **Tests:** `Coach.test.tsx` for each frame (mode chip from `gate` and from `X-Coach-Mode`; 409 confirm → resend with
  ack; recorded note; 503; paused; the three 429s incl. the local reset time; truncated marker + [Continue] sends one
  turn; provider limited; naming); `Problem.test.tsx` for the HUD chip and the capped outcome. Screenshots at 1440 px
  and 390 px beside AB01 F1–F10 in the PR.

### 5 · M1b exit + M1c readiness [X]

Sources: [rollout §3 M1 exit](../rollout-plan.md#3-milestone-map), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules), [t1 §4 *Expand → backfill → contract*](../research/t1-content-data-model.md).
- **Full verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...`, `go test -tags e2e ./internal/e2e/...`
  (PG 18, compose parity; set `XLEARN_TEST_DATABASE_URL`, or the e2e tests **silently skip**, per the gate in
  `internal/e2e/coreloop_test.go`), `sqlc generate` + `sqlc diff`, web typecheck / lint / test / build,
  `hack/lint-migrations.sh`, the subject-registry / topology golden, m1-06's route-enumeration test, m1-05's
  public-shape allowlist test, the OpenAPI drift test (`internal/gateway/openapi_drift_test.go` — new 409/429/503
  responses documented in [`../../architecture/openapi.yaml`](../../architecture/openapi.yaml) and [`api.md`](../../architecture/api.md)).
- **Golden = v1:** walk compose against v1 (m1-03's DSA parity screenshots). The **expected changes** are the
  D27 / ADR-0031 §7 family plus D31, and none of them is a regression to fix:
  - the D27 assist confirm, the capped panel note, HUD chip and outcome line;
  - `coach_paused` during a live mock (v1 mounts `<Coach/>` on every page, `web/src/components/AppShell.tsx`, so the
    coach worked during a mock; D27 keeps it locked);
  - the fail-closed 503 notices (v1 fell back to `attempt`);
  - the mode chip, the "Your AI coach (your key)" naming, the F8 L18 notices and the F9 [Continue] marker;
  - D31 (public mock count only).

  Flag any delta **outside** this list, then fix it or record it.
- **M1c readiness** (m1-08's entry gate; record results in the PR and the decisions log):
  - No SQL in HEAD reads or writes an M1c-drop column — `problem.is_reinforcement`, `leetcode_url`, `neetcode_url`,
    `concept.code_template`, `mock_session.total_35`, `api_key_config.is_default`. Run m1-03's
    `hack/lint-dropped-columns.sh`, with coach `is_default` **removed from its allowlist** now that m1-10 stopped
    using it, and m1-10's `internal/coach/contract_test.go`. By hand, also check what that script doesn't scan: the
    sqlc output `internal/*/store/gen/*.go` (sqlc expands `*` and `RETURNING *` into column lists), raw SQL in
    `internal/*/store/*.go` and the curriculum loader (m1-08 folds these surfaces into the script). Event-payload and
    JSON API names (`total_35` in the `mock_completed` payload, `total35` in `/api/mocks/*`) are not DB columns and stay.
  - **No v1 conflict target left** — m1-08 drops all three v1 uniques:
    - `ON CONFLICT (account_id, week_of)` on `review.weak_area_snapshot`: [m1-03](sprint-m1-03.md) task 3 switched
      it to `(account_id, path_slug, week_of)`;
    - `ON CONFLICT (slug)` on `curriculum.concept`: [m1-09](sprint-m1-09.md) switched it to `(path_slug, slug)`;
    - `ON CONFLICT (problem_id, stage, "order")` on `curriculum.problem_section` (v1:
      `internal/curriculum/store/queries/section.sql:15`): nothing has switched it yet. It must target m1-09's
      `(problem_id, stage, "order", language)` unique, or become a plain insert after m1-09's delete-then-reinsert.
      Otherwise the old unique blocks every multi-language code section (Go / C++ / Python at the same stage and
      order, M3).

    If any of the three is still present in HEAD, **switch it here** with a test (weak-area upsert with an explicit
    `path_slug`; concept and section seed re-run idempotent). Don't push the drop to a later tag: removing a unique
    later needs its own snapshot-first contract.
  - m1-02's CHECK swap (`mock_session_scored_total_check` on `COALESCE(total, total_35)`) is live, so scoring
    without `total_35` passes; no writer relies on a `DEFAULT 'dsa'`.
  - Go-side validation exists for what m1-08 un-CHECKs: mistake `category` ∈ the course manifest's categories
    (review `POST`/`PATCH /mistakes` and the consumer's pre-fill) and rubric `dimension` ∈ the session's
    `rubric_snapshot` (assessment `ScoreMock`); tests reject a bad value with 422. If missing, add it here — after
    m1-08 the rollback floor is 1.7.0, so this tag must be safe on the contracted schema.
  - No migration new since `v1.6.0` carries `-- xlearn:contract` (if one does, this is a contract tag: the contract
    lines of the checklist apply, including rehearsal, `host-verify --cluster` and a snapshot).

### 6 · Tag `v1.7.0` [X]

Run the release checklist (§Release). GitHub release title **`v1.7.0 — v2 build · M1b`**; notes: producers emit the
v2 envelope; course resolution; `withhold()`; limits; `RequireRole`; `identity admin`; D27 capture; mode gate; L18.
In [`../status.md`](../status.md): milestone M1b → tag `v1.7.0` → floor **1.6.0** (v2 envelopes in the log) →
snapshot n/a; **start the flag inventory** with `SIGNUP_MODE` under *operating modes* (permanent, [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service));
record that "the first path-aware release (the consumer floor for non-DSA events) is `v1.7.0`" (ADR-0034 §3).

### 7 · Owner role set once (`ev-owner-role`) [O]

After `v1.7.0` is verified, the owner runs (runbook `docs/runbooks/identity-admin.md` from m1-04):
`ssh vps 'k3s kubectl exec -n xlearn deploy/xlearn-identity -- identity admin account set-role <owner> owner'`.
The agent then confirms with `… identity admin account list --role owner` (read-only), checks `admin_audit` got the
row, and logs the CLI use in `docs/v2/status.md` (sanctioned manual path, [rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)).

## Acceptance criteria

- [ ] During an open counted attempt on problem P, a chat in P's context without `assist_ack` → 409
      `assist_confirm_required` (with `attemptId`); with it, the L18 admission probe passes and practice records
      `coach_assist_at`, both **before** coach is called; an admission 429 leaves the attempt uncapped; practice down →
      503 and nothing sent; once recorded, later chats (and reloads) skip the confirm (tests).
- [ ] An attempt with `coach_assist_at` logs Clean/Rough as **Assisted**; `problem_solved` carries `assist{hint, coach}`;
      v1- and v2-envelope consumers accept it; chats about another problem or page are unaffected.
- [ ] Live mock → 409 `coach_paused`, nothing sent; `attempt` prompts carry no pattern / concepts / solution facts
      (golden snapshots); `review` only when concluded with no due touch; `general` names live items off-limits;
      `coach_message.prompt_v = coach-prompt@2`; the mode reaches the SPA (`X-Coach-Mode` on the chat response,
      `gate` on the thread `GET`).
- [ ] L18: typed 429s with `Retry-After` for per-minute, concurrency and the 300/day cap; history ≤ 20 turns / 32 KiB.
- [ ] The coach panel matches AB01 F1–F10 (mode chip, confirm, recorded note + HUD chip, fail-closed, paused,
      review / general, the three 429s, cut-short + [Continue], provider limited, naming) at 1440 px and 390 px;
      the F8 reset-copy delta is flagged to the owner.
- [ ] **M1b live:** every visible change vs v1 is on task 5's expected-change list (D27 / ADR-0031 §7 family, D31);
      nothing else differs; every v1 e2e green.
- [ ] **No reader or writer of an M1c-drop column in `v1.7.0`** (grep + tests; task 5 record).
- [ ] `v1.7.0` verified (healthz version, images, ImagePolicies, HelmReleases Ready, smoke); floor 1.6.0 and the
      flag inventory recorded; owner role set.

## Release

**Tag `v1.7.0`** ([ADR-0034 §1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline): M1b — producers emit the v2 envelope; course resolution;
`withhold()`; limits; `RequireRole`; `identity admin` verbs; gate state after: golden = v1 with the D27 confirm, D31
and 429s aside — spelled out in full as task 5's expected-change list; rollback floor after: **1.6.0**). Take the next free minor at tag time.
Release checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6, verbatim, plus the ADR-0035 §2 standing rule):

- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

**For this tag:** ACL PRs — none expected (`assist` is a field on an existing subject; entry gate 4 confirms the
topology is unchanged); new service — n/a; contract / erase / GA — n/a unless task 5 finds a contract marker since
`v1.6.0`; M6 — n/a. New in-cluster callers — **none**: gateway → practice, gateway → assessment and gateway → coach
(the admission probe) already exist (list the `*_BASE_URL` clients in the diff to confirm); no new pod, so the memory-sum rule
([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)) is unaffected.
Extra smoke after the tag (owner account): a coach chat on an unsolved problem with an open attempt shows the
confirm; after confirming, the reply streams.

## Definition of Done

CI green (incl. `sqlc diff`, the migration lint and the OpenAPI drift test) · PR squash-merged · `v1.7.0` tagged,
deployed by Flux (no hand `kubectl`) and verified by the checklist · coach panel matches AB01 · acceptance criteria
met · statuses updated (this file + [`../status.md`](../status.md): board, M1 milestone, tag → floor, flag inventory,
owner-role CLI log) · notable calls in the decisions log (an ADR only for a genuinely new decision; check the next
free ADR number with peers first).

## Risks / watch-outs

- **Fail-closed coach.** With practice or assessment down, chats return 503 (intended: D27 capture fails closed,
  and the mode can't be known). Watch it in the smoke; don't "fix" it by defaulting to `attempt` and forwarding.
- **v1 attempts never auto-close.** An abandoned attempt stays open until an outcome is logged
  (`LogOutcome` is the only path that ends one), so its problem's chat asks for the confirm once and stays capped
  after that. That is D27 as written (the counted attempt is still open); the M2a/M3 tickers bring deadlines later.
- **Order of checks:** state lookup (503) → `locked` (409) → D27 confirm (409, skipped once recorded) → L18
  admission probe (429, attempt not capped) → assist record (503 on failure) → coach's own L18 checks → provider.
  A provider error after the record still caps the attempt (t5 §9). A 429 after the record happens only in the
  probe → chat race: this sprint's accepted decision, logged as such (t5 §9 doesn't cover it).
- **Session size.** This is the largest M1 session: four services (practice, assessment, gateway, coach), three
  expand migrations, web states, the full M1b verification, a tag and an owner event. If it runs long, stop once
  the PR is squash-merged (merging ships dark; 1.x deploys only on a tag), and run tasks 5 (the compose walk and the
  readiness record, re-run on `main`), 6 and 7 in a continuation session from prompt step 9. Never tag at the end of
  a rushed run.
- **Event compatibility:** `assist` must stay optional and additive; `v1.6.0` (the R-b floor) ignores unknown fields.
  Never rename or repurpose an existing payload field (envelope append-only, ADR-0034 §3).
- **Hidden M1c trap:** if anything in `v1.7.0` still names a drop-list column or targets a v1 unique (a sqlc
  `SELECT *` expansion, an `ON CONFLICT (slug)` or `(problem_id, stage, "order")`, a writer leaning on
  `DEFAULT 'dsa'`), m1-08 can't contract it. Task 5 is the gate; fix it
  here rather than moving the contract.
- **In-process limiter state** (coach per-minute and streams) resets on a coach restart and blocks scale-out (L24);
  the durable daily cap doesn't.
- **ADR-0031 is Proposed.** Don't wait for it: §7 rests on D27. Don't touch its status (mi-12 accepts it).
- **Calendar:** tag in week 4. Don't tag during the Sat 2026-10-24 host window, and leave the weekend for m1-08's
  settle-then-snapshot sequence.
