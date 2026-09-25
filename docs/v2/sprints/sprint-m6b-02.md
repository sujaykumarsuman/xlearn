# Sprint m6b-02 — Voice robustness: lease, re-attach, drain, cost + $ cap, PTT, rollover (+ coach-interview if M7 failed)

> **Milestone:** M6b — voice, one shell ([rollout §3](../rollout-plan.md#3-milestone-map)) · **Track:** product (coach; + infra and a patch tag only if S6 M7 failed) · **Order:** 83
> **Prereqs:** [m6b-01](sprint-m6b-01.md) (`VoiceShell`/`Sideband` with `Detach` ≠ `HangUp`, `voice.Segment`, the SDP broker, `voiceAdmission`, the reaper, captions)
> **Unblocks:** [m6b-03](sprint-m6b-03.md) (voice UI + the `v2.0.x` patch that ships M6b dark) → [m6b-04](sprint-m6b-04.md) (fake-media e2e, v2.1.0)
> **Release action:** **merge only** (ships dark in [m6b-03](sprint-m6b-03.md)'s `v2.0.x` patch). **Only if S6 M7 failed:** **infra PR A →
> tag a `v2.0.x` patch → infra PR B** — PR A (the peers' NetworkPolicy side: gateway egress, the `databases` ingress) merged **before** the
> tag; the tag (m6b-01 + m6b-02, dark) so the fallback `coach-interview` Deployment can run the role code — **image before
> HelmRelease/policy**; PR B (the `coach-interview` HelmRelease with its own chart-rendered NetworkPolicy, its pinned ImagePolicy and the
> routing switch) merged **after** it ([m6b-03](sprint-m6b-03.md)'s entry gate needs that Deployment live and routed). This conditional tag
> is a deliberate deviation from the register's "merge only (+ infra PR only if M7 failed)" — see Release
> **Calendar:** Q1 2027 · no owner event (an optional live restart check on the owner's key, ≈ $0.20: pre-approved by launching the prompt (D40), and run only if the owner saved his OpenAI key in the local compose stack before launch)
> **Execute with:** [`../prompts/prompt-m6b-02.md`](../prompts/prompt-m6b-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Sideband lease + make-before-break re-attach + cold re-attach; holder-only provider I/O; cross-pod wake-ups | X | ⬜ |
| 2 | SIGTERM drain inside the 60 s grace | X | ⬜ |
| 3 | Voice usage → `AddSpend`; $ cap hang-up and headroom check; the voice estimate | X | ⬜ |
| 4 | L19 voice caps (≤ 3 live platform-wide, ≤ 75 voice-min/day → typed 429) + voice idle ("Still there?") | X | ⬜ |
| 5 | Modes: Standard / Patient / push-to-talk; hold voice while I code; voice grace specifics | X | ⬜ |
| 6 | Rollover / reseed; AI notes checkpoints; transcript self-edit; client mirror | X | ⬜ |
| 7a | **Only if S6 M7 failed:** `COACH_ROLE` (interview role never on NATS) + gateway interview upstream for every coach `/interviews`, `/internal/interviews` call + lagging-schema test + runbook section (in this sprint's PR) — else n/a | X | ⬜ |
| 7b | **Only if S6 M7 failed:** infra **PR A** — peer-side NetworkPolicies (gateway egress → :8086, `databases` ingress), merged before the tag — else n/a | I | ⬜ |
| 7c | **Only if S6 M7 failed:** tag the `v2.0.N` patch (m6b-01 + m6b-02 dark; role code inert) — else n/a | X | ⬜ |
| 7d | **Only if S6 M7 failed:** infra **PR B** — `coach-interview` HelmRelease (own NetworkPolicy + `expected-netpol.tsv` row), pinned ImagePolicy, routing; merged after the tag — else n/a | I | ⬜ |
| 7e | **Only if S6 M7 failed:** status records + m6b-03 hand-off (small docs PR after PR B) — else n/a | X | ⬜ |
| 8 | Tests (restart in-process and in compose, cap, 76th minute), docs, record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why); rows 7a–7e are ✅
> "n/a — M7 passed (S6)" when the one-Deployment shape holds (7a merges with task 8's PR; 7b–7e follow that merge, in order). Update
> the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row; M6b stays 🔄;
> M7-failed only: the milestone → tag → floor row). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **[m6b-01](sprint-m6b-01.md) merged** (broker, sideband, `voice.Segment`, mint log, reaper, supersede, `caption`/`segment` events on `main`)
- [ ] **S6 facts at hand** (t6 §16 from [spk-04](sprint-spk-04.md), "for m6b-02"): **M7** (make-before-break / cold re-attach with audio
      uninterrupted — decides task 7), **M3b** (push-to-talk native or emulated), **M5** (session duration / `expired` close), `usage_ratio` at
      60 min, reseed TTFA, the cached-token ratio before/after a 15-minute silence, the **M9** usage-event shape used for cost
- [ ] **coach sized for voice** ([mi-13](sprint-mi-13.md), live): `terminationGracePeriodSeconds: 60`, `rollingUpdate` maxSurge 1 / maxUnavailable 0;
      the memory-sum margin mi-13 left is recorded in status.md (task 7 needs it)
- [ ] **M6a facts read:** [m6a-01](sprint-m6a-01.md)'s UTC day for "≤ 2 starts/day", its idle rule and client lease, its sweeper
      (`pg_try_advisory_lock`) and SIGTERM checkpoint; [m6a-04](sprint-m6a-04.md)'s event-stream notifier (in-process + 1 s poll)
- [ ] **Parallel sessions:** no open peer PR adds a coach migration or edits `internal/coach/interview/`, `cmd/coach/main.go`, the gateway
      interview proxy, `../infra/apps/xlearn-coach*.yaml`, `../infra/apps/image-automation.yaml` or the `xlearn` NetworkPolicies; the next free
      `v2.0.N` is known if task 7 applies (`gh pr list` in both repos, `git ls-remote --tags origin`, `git worktree list`, ListAgents)

## Goal

Make voice interviews **survive coach restarts and stay inside budget**. A **sideband lease** makes one coach pod the sole owner of each live
provider session; on a rollout the new pod attaches a **second sideband before** the old one detaches (**make-before-break**), a crashed holder
is replaced by **cold re-attach** by session id, and a **SIGTERM drain** fits the 60 s grace — audio never stops, because it runs browser ↔
OpenAI. Voice usage is **priced into M6a's `AddSpend`**, so the **mandatory $ cap** wraps up at 85% and ends — with a hang-up — at 100%;
**≤ 3 live voice** sessions run platform-wide and **≤ 75 voice-minutes per account per day** (× the time multiplier), else a typed 429; idle
calls get "Still there?" and are hung up. Add **Standard / Patient / push-to-talk**, **"hold voice while I code"**, planned **rollover**, AI
notes checkpoints and learner transcript self-edit. If S6 **M7 failed**, stand up the fallback: a `coach-interview` Deployment of the same
image with its own, manually bumped ImagePolicy, live and routed before [m6b-03](sprint-m6b-03.md).

## Scope

**In**
- `internal/coach/interview/voice/`: the sideband lease, handoff, cold re-attach, drain hooks, pricing into `AddSpend`, caps, the voice idle
  prompt, modes, hold, rollover; FSM table rows for `held` and `bridging` (the states exist since m6a-01 — no enum migration).
- `cmd/coach/main.go`: the drain sequence (shutdown budget 10 s → 50 s), a dedicated LISTEN connection, role wiring (task 7 only).
- Coach endpoints: `POST /interviews/{id}/ptt`, `POST /interviews/{id}/hold`, `POST /interviews/{id}/voice-mode`,
  `PATCH /interviews/{id}/turns/{seq}` (self-edit), `POST /interviews/{id}/turns/mirror`; the matching gateway interview-proxy rows;
  `openapi.yaml`/`api.md`.
- The `voice` block completed: `mode`, `capacity` / `daily_limit` reasons, `minutes_left_today`, `estimate{…}`.
- An expand migration on schema `coach`: `interview_sideband_lease`, `interview.voice_mode`.
- `coach admin interviews --live` gains the lease holder, epoch, open segment, shell and purpose.
- `mic` and `voice_limit` as new interrupt reasons; `held` taking `live`'s rail, finish and cap rows (task 5).
- **Only if S6 M7 failed:** `COACH_ROLE`, the gateway's `COACH_INTERVIEW_BASE_URL`, two infra PRs, a `v2.0.x` patch tag, and a
  "coach-interview (M7 fallback)" section in `docs/runbooks/interviewer.md` ([m6a-01](sprint-m6a-01.md)'s runbook).

**Out**
- Every UI surface (pre-flight, HUD, PTT button, hold banner, grace countdown, cap banners, "Still there?", self-edit screen) →
  [m6b-03](sprint-m6b-03.md) (AB29, AB30 F1–F19).
- The first manual bump of `coach-interview` to m6b-03's tag (M7-failed only) → [m6b-03](sprint-m6b-03.md), by this sprint's runbook.
- The fake-media e2e on prod and the v2.1.0 flip → [m6b-04](sprint-m6b-04.md); a mid-call restart on prod is out there too (no hand
  `kubectl rollout`), so this sprint's compose run is the evidence.
- History-calibrated estimates, the multi-speaker fairness check → M6c ([m6c-02](sprint-m6c-02.md)).
- Interview counters, alerts, opscheck (D34); stored audio (never).

## Tasks

### 1 · Sideband lease + make-before-break + cold re-attach [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) ("Surviving deploys"),
[t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (failure table: coach restart or deploy),
[t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) step 3 and M7. (Not to be confused with [m6a-01](sprint-m6a-01.md)'s
**client** lease, which picks the browser tab; this one picks the coach **pod**.)

- **Table** (expand): `coach.interview_sideband_lease(interview_id uuid PRIMARY KEY REFERENCES coach.interview ON DELETE CASCADE,
  segment_n int NOT NULL, holder text NOT NULL, epoch bigint NOT NULL, renewed_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
  handoff_requested_at timestamptz NULL)`. `holder` = pod hostname + a random boot nonce. Taken in the transaction that confirms a voice
  segment (m6b-01), renewed every **5 s**, TTL **15 s**, deleted when the segment closes.
- **Only the holder touches the provider:** the sideband, the segment's key buffer, director/brain voice pushes, current-screen pushes,
  PTT mute/unmute and hang-ups. A request that lands on another pod writes its intent to the DB and wakes the holder with
  `pg_notify('coach_interview', '{"interview_id","epoch","kind"}')` (kinds `snapshot`, `ptt`, `hold`, `mode`, `ask`, `close`, `handoff`,
  `taken` — never transcript, SDP or key material); the holder ignores stale epochs. Segment closes triggered by the FSM on a non-holder pod
  (e.g. m6a-01's sweeper firing `interrupt(network)`) are routed the same way, so `HangUp` always runs in the holder.
- **Event stream across pods:** [m6a-04](sprint-m6a-04.md)'s handler waits on an in-process notifier with a 1 s poll fallback, so it already
  works across two pods at ≤ 1 s; add a `pg_notify('coach_interview_events', interview_id)` wake-up after each `events.Append` so captions
  written by the holder reach a stream served by the other pod promptly.
- **A dedicated LISTEN connection** per pod (a `pgx.Conn` outside the pool): the pool pin stays `MaxConns` 4 (L21); two pods during a rollout
  use 2 × (4 + 1) = 10 connections, inside the role's `connectionLimit` 20 (MI-15). Reconnect with backoff; on reconnect, re-read leases.
- **Make-before-break** (rollout, maxSurge 1): the old holder, on SIGTERM (task 2), sets `handoff_requested_at` and NOTIFYs `handoff`. A ready
  peer claims it: decrypt the key → `AttachSideband(provider_session_id)` (a **second** sideband) → wait ≤ 5 s for its first event →
  `UPDATE interview_sideband_lease SET holder=$me, epoch=epoch+1, handoff_requested_at=NULL, expires_at=now()+interval '15 s'
  WHERE interview_id=$1 AND epoch=$e` → NOTIFY `taken`. The old pod, on `taken`, calls `Sideband.Detach()` (**never `HangUp`**) and zeroes its
  key buffer.
- **Cold re-attach:** a job in m6a-01's sweeper (tick 5 s, `pg_try_advisory_lock`) claims leases past `expires_at` for open voice segments
  (holder crashed, OOM-killed, or its drain ran out) and re-attaches by session id. If the provider refuses (session gone / `expired`) →
  `interrupt(provider)` → m6a-01's **free re-prime**: event `segment{event:"reprime", gap_ms}`, the SPA posts a new offer, the new segment is
  primed from our own text state (target **TTFA ≤ 3 s**). The gap is kept in that event (task 6 fills it from the client mirror where one exists).
- **Two-pod safety:** reaper, sweepers and state transitions keep M6a's conditional `UPDATE … WHERE state=$expected`; the lease is the only
  authority for provider I/O; a pod that sees the epoch move stops pushing at once.
- `coach admin interviews --live` shows `holder`, `epoch`, the open segment `n`, `shell`, `purpose` per live interview (the release checklist's
  "no live interviews" read is unchanged).

### 2 · SIGTERM drain [X]

`cmd/coach/main.go` (today `shutdownTimeout = 10 * time.Second`; m6a-01 added a `sigterm` checkpoint for every running interview). New
sequence, all inside the pod's **60 s** grace:
1. mark draining: new segment creation → 503 `draining` with `Retry-After: 2` (the SPA retries once and lands on the new pod — the gateway
   never retries), and `srv.SetKeepAlivesEnabled(false)` so the gateway's pooled keep-alive connections close after their next response
   and reconnect through the Service. **No readiness flip is relied on:** coach's HelmRelease probes `/healthz` for both readiness and
   liveness (chart 0.3.0's `probes.readinessPath` is a per-release opt-in — [mi-01](sprint-mi-01.md) — that [mi-13](sprint-mi-13.md) doesn't
   set for coach), and a terminating pod leaves the Service's ready endpoints on its own once the surge pod is Ready; `503 draining` plus
   one SPA retry covers the propagation lag. (If a later session wants `/readyz` as coach's readiness, that is its own infra PR.)
2. m6a-01's `sigterm` checkpoints (code snapshot, transcript tail, clock), then a handoff request for every lease this pod holds (task 1);
   wait up to **30 s** for `taken`;
3. leases still held after 30 s: set `expires_at = now()` and `Detach()` the sideband **without** hanging up (the cold path takes over);
4. stop the erase consumer subscription **before** `srv.Shutdown` (the consumer shutdown-order rule) and stop the sweeper;
5. `srv.Shutdown` with the remaining budget (bounded event streams end within their ≤ 50 s cap; clients reconnect with `Last-Event-ID`);
6. zero every key buffer; exit.
`shutdownTimeout` becomes **50 s** in total (handoff ≤ 30 s + shutdown ≤ 15 s + margin), below the 60 s grace.

### 3 · Voice usage → `AddSpend`, the cap, the estimate [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (caps table), [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys),
[ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits), [m6a-01](sprint-m6a-01.md) (`cap_micros`, `spent_micros`,
`AddSpend`: 85% → `wrapping`, 100% → `finished(cut_short=cap)`).

- **Pricing:** from the sideband's usage events — GPT-Live `session.usage.updated` (billed seconds, `usage_ratio`) × the catalog per-second
  price; Realtime `response.done` usage (text/audio input/output and cached tokens) × the catalog token prices — written into the open
  segment's `usage` (counts only, never content) and `cost_micros`, and passed to `AddSpend` as each event arrives. Pre-flight segments are
  priced too. Director, checkpoint, probe and brief calls already reach `AddSpend` through m6a-02.
- **The cap on voice:** `AddSpend`'s transitions already move the FSM; for voice, `cap_100` closes the segment, so `voice.Segment.Close`
  **hangs up at once**; the `cap` event (m6a-04's kind) carries `{pct, cap_micros}` for AB30 F10. At mint, remaining headroom below one
  minute's estimated cost → **409 `cap_reached`**. The post-finish review call and one re-propose (≈ $0.06 each) are outside the live cap but
  inside the "plan for" estimate — record this.
- **Estimate** in the `voice` block: `estimate{typical_micros, plan_for_micros, keep_available_micros, per_multiplier}` — duration-billed
  shells: minutes × rate × multiplier + the brain estimate; token-billed: a per-phase speaking profile at a cache-hit rate of 0.9
  (plan-for = the high end); "keep available" = plan-for + $1. A pure function (`voice/estimate.go`), table-tested against t6 §8 (±5%)
  and the S6 M9 figures.
- **Speaking profile — derived in coach, not a manifest field.** t6 §8 names a manifest `mock.voice_profile`, but no sprint adds it
  ([m6a-01](sprint-m6a-01.md)'s manifest additions are `mock.interviewer` and the rubric fields, and unknown fields stay rejected). So the
  profile is derived from the interview's `mock_snapshot` rail and t6 §8's assumptions — 45-min rail ≈ 48 min wall, ~20 min candidate
  speech, ~9 min AI speech, ~55 responses; 60-min ≈ 63 min, ~26 / ~12 min, ~72 responses — spread over the phases by their minutes and
  scaled by the multiplier, as named constants in `estimate.go`. No curriculum or manifest-schema change here; a per-course
  `mock.voice_profile` override is a later additive manifest field when a second course needs one. Record this in the Decisions log.

### 4 · L19 voice caps + voice idle [X]

Sources: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L19, [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits),
[t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) caps, [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (idle).
Implemented in m6b-01's `voiceAdmission` hook and the holder's meter — durable in Postgres, never in process (two pods overlap on every rollout).

| Cap | Rule | Refusal |
|---|---|---|
| **≤ 3 live voice platform-wide** | at mint, in a transaction holding `pg_advisory_xact_lock(hashtext('coach.voice_live'))`: count open `kind='voice'` segments (both purposes) | **429 `voice_capacity`**, `Retry-After: 60` |
| **≤ 75 voice-minutes per account per day × the interview's time multiplier** | billed seconds (or connected wall time while usage is pending) over the account's voice segments since 00:00 UTC — [m6a-01](sprint-m6a-01.md)'s day rule for starts | at mint: **429 `voice_daily_limit`**, `Retry-After` = seconds to 00:00 UTC; live: a 3-minute wrap cue, then at the limit `interrupt(voice_limit)` (a new m6a-01 reason; the segment closes and hangs up) followed by `save_later` → `paused` (m6a-01's `resume_by` and ≤ 3-pause rules unchanged) |
| **Idle** (extends m6a-01's rule: 3 min × multiplier, no candidate turn, no code change, heartbeat `visible=false`) | voice speech events (speech started / final candidate turns) update `last_input_at`; **60 s before** the idle interrupt, a `notice{kind:"idle_check"}` event ("Still there?", AB30 F8) | m6a-01's `interrupt(idle)` closes the segment, which **hangs up**; Talk / reconnect re-primes free |

The `voice` block gains `reason ∈ {capacity, daily_limit}` and `minutes_left_today`. At 1×, the **76th** voice-minute of the day is refused
with a typed 429.

### 5 · Modes, hold, voice grace [X]

Sources: [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (quiet during Code, hold voice), [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (modes),
[t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (failsafe 1), S6 M3/M3b.

- `interview.voice_mode text NULL CHECK (voice_mode IN ('standard','patient','ptt'))`, set at setup (AB29 F1) for `mode='voice'`;
  `POST /interviews/{id}/voice-mode {mode}` switches **standard ↔ ptt** mid-call (AB30 F4) via `Reconfigure`.
- **Per the S6 winner:** Realtime mini — Standard `semantic_vad` eagerness `medium`, Patient `low`, push-to-talk `turn_detection: null` with
  `input_audio_buffer.commit` + `response.create` on release; GPT-Live — Standard default, Patient through the instruction prefix, push-to-talk
  **emulated** with `session.input_audio.mute`/unmute on the sideband (the SPA also toggles `track.enabled`, m6b-03), inside S6 M3b's thresholds.
- `POST /interviews/{id}/ptt {"state":"down"|"up"}` → the holder mutes/unmutes (`up` triggers the reply); ≤ 10/s per interview.
- **Quiet during Code:** eagerness `low` during the Code phase (Realtime); m6a-02's director checks in only on a question, a Run worth probing,
  or ≥ 90 s × multiplier of silence with no edits.
- **Hold voice while I code:** FSM rows `live —hold→ held` and `held —talk→ connecting` (data-table additions; `held` is in m6a-01's running set,
  so the clock keeps running). `POST /interviews/{id}/hold` closes the segment (**hang-up**, key zeroed); **Talk** = the SPA posts a new offer to
  `/segments` from `held` (m6b-01's broker gains `held` as a source state; the offer fires `talk` first, so a creation error takes m6b-01's `connecting` effects) → `connecting` → a free
  reseed. Because the clock runs in `held`, `held` also takes `live`'s rail, finish and cap rows — `rail_done` \| `finish` \| `cap_85` →
  `wrapping`, `cap_100` → `finished(cut_short=cap)`, plus `abandon` — so a held interview still wraps and the cap still ends it; no segment
  is minted in `wrapping`, so a debrief entered from `held` arrives as text turns (AB30 F11). The table test and model check cover the rows.
- **Mic loss** ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) failure table): `mic` joins
  m6a-01's interrupt reasons next to `voice_limit` (task 4), and is **client-postable** like `network` and `tab_closed`: [m6b-03](sprint-m6b-03.md)'s
  SPA shows "Reconnect mic" after 10 s and posts `interrupt {reason: "mic"}` after 30 s → the segment closes (hang-up), m6a-01's grace runs,
  and a new offer from `interrupted` once the mic is back re-primes free. The clock stops **at that interrupt**: t6's "stops after 10 s" is
  simplified to the 30 s interrupt, and AB30 F7's copy says so.
- **Voice grace specifics:** on `ErrQuota` the segment closes at once (GPT-Live bills connected seconds) — m6a-01 already marks the cut-off
  interviewer turn `truncated` and sets `grace_until`; probes decrypt per call; each probe writes `notice{kind:"probe", checked_at, result}` so
  AB30 F9 can show "Checked 20 s ago". M6a's probe limits (≤ 1 per 10 s, ≤ 15 per grace, the sweeper's final probe) are unchanged.

### 6 · Rollover, checkpoints, self-edit, mirror [X]

- **Rollover (`bridging`):** FSM rows `live —rollover→ bridging` and `bridging —segment_open→ live`. Realtime — at the start of Code when the
  segment is over 40 min or its context over ~24k tokens; **forced** for any 60-minute rail or multiplier > 1× (the 60-minute hard cap).
  GPT-Live — only if S6 found a duration limit shorter than the interview, or when `usage_ratio` passes **0.8** (at the next turn boundary).
  Mechanism: the brain speaks a bridge line; event `segment{event:"rollover"}`; the old segment closes (**hang-up**) and the SPA posts a new offer
  (same `client_id`, `purpose:'live'`) — m6a-01's index allows one open segment, so the swap is close-then-open at a phase boundary (≈ 1–3 s
  of silence behind the bridge line); the new segment is primed with m6a-02's `Prime` (stable prefix + brief/phase state + last ≤ 4 turns +
  "continue with ⟨next_step⟩", ≤ 8,192 tokens). `ErrSessionCap` (`expired`) takes the same path as a free reseed. A creation error on
  the new offer takes m6b-01's `connecting` effects through one more voice-only row, `bridging —interrupt(reason)→ interrupted`.
- **AI notes checkpoints:** at phase boundaries the director also emits `{evidence[≤ 3 per public dimension, with turn refs], covered[],
  pending_followups[], next_step}` into m6a-01's checkpoints (C4; same retention and erase); its cost is in the estimate.
- **Learner transcript self-edit (voice):** `PATCH /interviews/{id}/turns/{seq} {"text"}` — candidate turns only, only in `finished` /
  `proposed`, ≤ 2 KiB → `edited=true` and the `transcript_edited` caveat, which makes the saved score `scored_by=self`
  ([m6a-03](sprint-m6a-03.md)'s `ai-byo` conditions; AB30 F17). m6a-01's per-turn delete stays.
- **Client mirror:** `POST /interviews/{id}/turns/mirror` accepts ≤ 20 turns of ≤ 2 KiB each, only inside a recorded re-prime gap, stored with
  `source='client'` (again no `ai-byo`); outside a gap → 409 `no_gap`. m6b-03 decides whether the SPA mirrors (it takes captions only from the
  event stream, so the mirror is a later option; the endpoint is cheap and bounded).

### 7 · Conditional deploy shape — **only if S6 M7 failed** [X · I · X · I · X]

If S6 recorded **M7 pass**, mark rows 7a–7e ✅ "n/a — M7 passed" and skip this task. Otherwise ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)
"If M7 fails"; [ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture)), in this order — each row its own PR or tag.
xlearn's NetworkPolicies are **chart-rendered per release** from `apps/xlearn-*.yaml` ([mi-03](sprint-mi-03.md) MI-5a,
[mi-11](sprint-mi-11.md) MI-15), and chart `networkPolicy.enabled` defaults to `false`: so `coach-interview`'s **own** ingress/egress can
only be born with its HelmRelease (PR B), while every **peer's** side changes first (PR A). The signed-in smoke checks in 7b–7d use an
already-signed-in browser session if the session has one (the agent never enters credentials); otherwise record "owner login smoke
pending" as a pending-smoke note in status.md and carry on.

**7a · X — role code (merged with this sprint's PR)**
- `cmd/coach`: `COACH_ROLE ∈ {all (default, today's behaviour), core, interview}`. `interview` serves `/interviews/*` and
  `/internal/interviews/*` and runs the sweeper (lease, reaper, grace/pause jobs, retention purge); `core` serves keys/chat/threads/models,
  the outbox relay and the erase consumer, and answers `/interviews/*` and `/internal/interviews/*` with 404. Both run migrations under the
  advisory lock; the **lagging** `interview` role must start against a newer schema (test: `store.Migrate` with the embedded FS against a DB
  one version further returns nil).
- **`interview` never connects to NATS:** no publisher, no outbox relay, no consumer, even if `NATS_URL` is set — a `LogPublisher` relay
  there would mark `core`'s unsent erase acks as sent — and interview code writes no outbox rows. So `coach-interview` needs no nkey, no
  messaging ingress and no ACL change. Test: `COACH_ROLE=interview` starts with no NATS env and never starts the relay or a consumer.
- **Gateway:** `COACH_INTERVIEW_BASE_URL` (unset → the coach URL), chosen by the **coach upstream path**, not the public route: every gateway
  → coach call to `/interviews…` or `/internal/interviews…` uses it — the `/api/interviews/*` proxy rows (`/segments`, the event stream, the
  new m6b routes) **and** the gateway's own internal calls: the arena handlers' `POST /internal/interviews/{id}/exposure` push and the
  resume/finish backstop ([m6a-01](sprint-m6a-01.md)'s `pause_exposure`), and the `GET /interviews/active` reads behind the exposure push,
  the coach-lock derivation and [m6a-03](sprint-m6a-03.md)'s submit orchestration. A test enumerates the gateway's coach call sites and
  asserts each interview path resolves to the interview upstream — otherwise `core`'s 404 drops `pause_exposure` with only a WARN.
- **Runbook** — `docs/runbooks/interviewer.md` § "coach-interview (M7 fallback)": the **manual bump** (a range PR to the new exact tag, only
  when `coach admin interviews --live` is empty — [m6b-03](sprint-m6b-03.md) does the first one right after its tag), the **rollback**
  (revert PR B: the gateway falls back to coach, `COACH_ROLE` back to `all`), the **contract floor** (a coach contract migration waits until
  the pin is at or above its floor), and the policy shape below.

**7b · I — PR A: peer-side NetworkPolicies, merged before the tag** (ADR-0035 §2 standing rule). Harmless before any pod exists:
- `apps/xlearn-gateway.yaml`: its `networkPolicy.egress` admits `app.kubernetes.io/instance: xlearn-coach-interview` on TCP 8086 (next to
  the `xlearn-coach` :8086 rule);
- `infrastructure/database/cluster/` — MI-5's `databases/projects-pgstore-ingress`: `xlearn-coach-interview` added to the instance list on
  5432 (a missing entry blocks it silently);
- **verify, no change expected:** identity :8081 and the gateway's JWKS :8080 admit `podSelector: {}` (same namespace) since MI-5a, so
  `coach-interview` reaches them; paste the rendered rules into the PR. No messaging change (`interview` never connects to NATS);
- `helm template` diff (only the gateway's egress changes) + the policy diff; `--dry-run=server`; merge once CI is green; Flux reconciles;
  smoke login and a coach chat.

**7c · X — tag the next free `v2.0.N`** (release checklist below; title `v2.0.N — v2.1 build · M6b backend (dark) + coach-interview roles`).
It carries m6b-01 + m6b-02 dark; `COACH_ROLE` unset = `all` and `COACH_INTERVIEW_BASE_URL` unset = coach, so nothing changes yet.

**7d · I — PR B: the Deployment, merged after the tag**, only when `coach admin interviews --live` is empty:
- `apps/xlearn-coach-interview.yaml`: a HelmRelease of `charts/project` that copies `apps/xlearn-coach.yaml`'s values (env, `envFrom` DB and
  master-key secrets, probes, security contexts, `automountServiceAccountToken`, `GOMEMLIMIT`, grace 60 s, rollingUpdate 1/0) and changes
  only: `component: coach-interview`, `replicaCount: 1`, `COACH_ROLE=interview`, **no `NATS_URL` or nkey env/secret**, limits 500m / 256 Mi,
  `route.enabled: false`, `tag: 2.0.N` with the setter marker `{"$imagepolicy": "flux-system:xlearn-coach-interview:tag"}`;
- **its own NetworkPolicy, mirroring coach's minus NATS:** `networkPolicy.enabled: true`, `from: null`,
  `extraIngress: [{from: [{podSelector: {}}], ports: [{port: 8086, protocol: TCP}]}]` (same namespace, as coach's MI-5a rule — the gateway is
  the only caller), and `networkPolicy.egress`: DNS (`kube-system` kube-dns, UDP + TCP 53), PG (`databases`, `cnpg.io/cluster:
  projects-pgstore`, 5432), identity :8081, gateway :8080 (JWKS), TCP 443 to `0.0.0.0/0` with [mi-11](sprint-mi-11.md)'s except-list (node
  IP `/32` included). Helm creates the NetworkPolicy before the Deployment, so the pod never runs unpoliced;
- `hack/expected-netpol.tsv`: row `xlearn<TAB>xlearn-coach-interview<TAB>m6b-02` and the byte-identical embedded copy in
  `hack/host-verify.sh` ([mi-02](sprint-mi-02.md)'s presence check); `hack/host-lint.sh` clean;
- `apps/image-automation.yaml`: ImagePolicy `xlearn-coach-interview` on the existing ImageRepository `xlearn-coach`, `semver.range` pinned
  to **exactly `2.0.N`** (it exists: image before policy). Routine release tags never move it; only a PR does, with no interview live;
- `apps/xlearn-coach.yaml`: `COACH_ROLE=core`; `apps/xlearn-gateway.yaml`: `COACH_INTERVIEW_BASE_URL=http://xlearn-coach-interview.xlearn.svc.cluster.local:8086`;
- **memory sum** before merging: +256 Mi limits (+500m CPU) steady; `coach-interview` sits outside the fleet rollout (its bumps are
  solo), so the surge term is unchanged. Check against mi-13's recorded margin with `host-verify --cluster`; if it won't fit, ⛔ "R0 trim
  first" ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
  PG connections: `core` + `interview` + a surge pod ≤ 3 × (4 + 1) = 15, inside the `xlearn_coach` role's `connectionLimit` 20 (MI-15);
- `helm template` render diff (one new release — Deployment, Service, NetworkPolicy — plus two env changes); `--dry-run=server`.
- **Verify:** Flux reconciles; `k3s kubectl get deploy -n xlearn` shows `xlearn-coach-interview` Ready on `2.0.N` and
  `get networkpolicy -n xlearn xlearn-coach-interview` exists; as the owner, `GET /xlearn/api/v1/interviews/active` is answered (routed
  to `coach-interview`) and an arena page loads with no exposure WARN in the gateway log (its `GET /interviews/active` read now reaches
  `coach-interview`); coach chat still works on `xlearn-coach` (`core`); `host-verify --cluster` green, NetworkPolicy presence included.

**7e · X — record** (a small `docs(status)` PR after PR B; this sprint's main PR has merged by then): status.md milestone → tag → floor
row (`v2.0.N`, floor unchanged, no snapshot), PR A/B numbers, the pinned version, the memory sum before/after, config notes
(`COACH_ROLE`, `COACH_INTERVIEW_BASE_URL`), the `coach-interview` contract floor, and the hand-off line for m6b-03 ("after your tag: bump
`coach-interview` per `docs/runbooks/interviewer.md`").

## Acceptance criteria

- [ ] **Restart during a call keeps audio** (make-before-break: a second sideband attaches before the first detaches; the provider sees no
      hang-up), **or reseeds within TTFA ≤ 3 s** (cold path) — in-process integration and the two-container compose run green; the optional
      live check's result, or "S6 M7 + fakes", recorded
- [ ] SIGTERM drain completes inside 50 s (< the 60 s grace) with no provider hang-up; the old pod's key buffers zeroed
- [ ] **The cap stops the session with a clear message:** 85% → `wrapping` + `cap` event; 100% → hang-up + `finished(cut_short=cap)` + `cap`
      event; mint refused (409 `cap_reached`) below one minute of headroom
- [ ] **The 76th voice-minute of the day → typed 429** `voice_daily_limit` (at 1×) with `Retry-After`; a 4th concurrent live voice → 429 `voice_capacity`
- [ ] Voice idle: `notice{idle_check}` 60 s before m6a-01's `interrupt(idle)`, which hangs up
- [ ] Standard / Patient / push-to-talk behave per the S6 winner; hold closes the segment with the clock running; Talk reseeds free; a held
      interview still wraps on rail end, finish or cap; `interrupt(mic)` and `interrupt(voice_limit)` close the segment and hang up
- [ ] Rollover primes ≤ 8,192 tokens and swaps close-then-open at a phase boundary; self-edit and mirror turns block `ai-byo`
- [ ] (M7 failed only) PR A (peer side) merged before `v2.0.N`; `v2.0.N` tagged and verified; PR B (with `coach-interview`'s own
      NetworkPolicy and `expected-netpol.tsv` row) merged after it with the memory sum inside the rule; `/api/interviews/*` **and** the
      gateway's `/internal/interviews/*` calls served by `coach-interview`; the `interview` role opens no NATS connection; runbook section +
      hand-off recorded
- [ ] CI green (`go test ./...`, `go vet`, lint, `sqlc diff`, OpenAPI drift); merged

## Release

**Merge only (ships dark in [m6b-03](sprint-m6b-03.md)'s `v2.0.x` patch)**; like m6b-01 it rides any earlier peer patch, still dark
(cohort-gated API, no UI). No NATS change (no ACL PR). No new pod in the M7-pass shape.

**Only if S6 M7 failed — tag a `v2.0.x` patch + infra PRs** (task 7): the xlearn merge (7a) and PR A (peer-side NetworkPolicies, 7b), in
either order → **tag `v2.0.N`** (7c) → PR B (HelmRelease with its own NetworkPolicy + pinned ImagePolicy + routing + `COACH_ROLE=core`, 7d)
→ the records PR (7e). It is a **patch**: from `v2.0.0` on the minor moves only at a GA flip
([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)).

**Deviation from the register, on purpose.** The register gives this sprint "merge only (ships dark in m6b-03's v2.0.x patch) (+ infra PR
only if M7 failed)" and the M6b tag chain `v2.0.x (mi-13) → v2.0.x (m6b-03) → v2.1.0`. In the M7-failed branch that can't hold: PR B's
HelmRelease and ImagePolicy need an image that already carries the role code (**image before HelmRelease/policy**,
[rollout §2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag)), and t6 §10 needs `coach-interview` live before P1's UI —
[m6b-03](sprint-m6b-03.md)'s entry gate. So the M7-failed chain is `v2.0.x (mi-13) → v2.0.N (m6b-02, conditional) → v2.0.x (m6b-03) →
v2.1.0`; in the M7-pass branch the register's chain is unchanged.

Release checklist
([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag:** ACL PRs — n/a (no stream or consumer); new service — n/a (same coach image; `coach-interview`'s policy is pinned to this
tag **after** it exists); contract/erase/GA — n/a (expand-only migrations), so no snapshot; **no live interviews** —
`ssh sujaykumar-vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` empty; standing rule — PR A (the peers'
side: gateway egress, `databases` ingress) merged first, and `coach-interview`'s own chart-rendered policy is born with its release in PR B;
flags — none (`COACH_ROLE` and `COACH_INTERVIEW_BASE_URL` are deploy-shape config, recorded in status.md's config notes, not the flag
inventory). `v2.0.N` → floor unchanged → no snapshot. Rollback: revert PR B (routing back to coach), then R-c for code.

## Definition of Done

CI green · merged via PR (squash) · M7 pass: no tag · M7 failed: PR A, `v2.0.N`, PR B, the records PR in that order, deployed by Flux and verified live ·
acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md)) · local `main` synced in xlearn (and `../infra` if touched).

## Risks / watch-outs

- **Rolling update mid-call.** `maxUnavailable: 0` + the drain + make-before-break are what keep audio up; a drain longer than the grace turns
  a handoff into a cold re-attach — keep the 50 s budget and test it with three sessions.
- **Two holders.** Any provider I/O outside the lease (a sweeper hanging up, a handler pushing a note) races the new holder; epoch-check every
  push and route FSM-driven closes to the holder.
- **Two leases, two meanings.** m6a-01's client lease (which tab) and this sideband lease (which pod) must never be conflated in code or docs.
- **Billing ≠ our meter.** Usage events can lag; the cap uses our estimate and errs early (wrap at 85%). A quota error still arrives through
  the classifier and the grace.
- **The daily-minute clock** is UTC midnight like m6a-01's starts/day; AB29 shows the reset in the account timezone — keep them consistent.
- **Rollover silence.** Close-then-open costs 1–3 s at a phase boundary; the bridge line covers it. Relaxing m6a-01's one-open-segment index to
  overlap two sessions would double-bill for the overlap — don't.
- **Lagging `coach-interview` role** (M7 failed): a coach contract migration can break it — the floor rule and the runbook guard it; the pin moves
  only when no interview is live. Its erase coverage comes from `core` (same schema).
- **Split-shape silent failures** (M7 failed): a gateway interview call left on the coach URL gets `core`'s 404 (the exposure push only
  WARNs), a missing `databases` entry blocks `coach-interview` without an error anywhere, and an `interview` pod with a NATS relay would drain
  `core`'s outbox into logs — the call-site test, PR A and the no-NATS test are the guards.
