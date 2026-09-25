# Sprint m6b-01 — VoiceShell adapter + SDP broker + sideband

> **Milestone:** M6b — voice, one shell ([rollout §3](../rollout-plan.md#3-milestone-map)) · **Track:** product (coach; small gateway/identity touches) · **Order:** 82
> **Prereqs:** [mi-13](sprint-mi-13.md) (MI-16 live: the 20 s `POST /api/interviews/{id}/segments` route, `Permissions-Policy`, coach 500m / 256 Mi, grace 60 s, rollingUpdate 1/0, WSS egress) · [ds-m6b-01](sprint-ds-m6b-01.md) (AB29–AB30 frozen) · [m6a-06](sprint-m6a-06.md) (M6a shipped dark: interview core, brain, bounded SSE, segment log)
> **Unblocks:** [m6b-02](sprint-m6b-02.md) (lease, drain, cost, caps, modes, rollover) → [m6b-03](sprint-m6b-03.md) (voice UI + the v2.0.x patch)
> **Release action:** **merge only** (ships dark in [m6b-03](sprint-m6b-03.md)'s `v2.0.x` patch; if a peer cuts an earlier `v2.0.x` patch it rides that one, still dark: cohort-gated API, no UI)
> **Calendar:** Q1 2027 · no owner event (an optional ≤ 2-minute live check on the owner's key, ≈ $0.10: pre-approved by launching the prompt (D40), and run only if the owner saved his OpenAI key in the local compose stack before launch, else deferred to [m6b-04](sprint-m6b-04.md))
> **Execute with:** [`../prompts/prompt-m6b-01.md`](../prompts/prompt-m6b-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Reconcile inputs: S6 facts, route name, fixtures, ADR-0007 amendment, record the AB29–AB30 freeze (if ds-m6b-01 didn't) | X | ⬜ |
| 2 | Catalog `voice_shell` row final + `VoiceShell` transport + `voice.Segment` (m6a-02's `Segment`) + the S6-winner adapter | X | ⬜ |
| 3 | Segment key lifetime (ADR-0007 amendment) extended to the sideband, reaper and supersede paths | X | ⬜ |
| 4 | SDP broker `POST /interviews/{id}/segments`: preconditions, mint log, typed errors, pre-flight purpose | X | ⬜ |
| 5 | Confirm, 15 s orphan reaper, second-tab supersede (m6a-01's client lease) | X | ⬜ |
| 6 | Sideband: `coder/websocket`, audio-drop decoder, tamper re-apply | X | ⬜ |
| 7 | Director push, D29 current-screen item, captions + "ask to repeat" on the event log | X | ⬜ |
| 8 | Tests: canary + no-audio-persisted, fixture contract, fake-shell integration; optional live check | X | ⬜ |
| 9 | Docs + record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row; the
> **M6b** milestone row → 🔄; the Artboards rows AB29–AB30). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **MI-16 done and its `v2.0.x` patch live** ([mi-13](sprint-mi-13.md)): `POST /xlearn/api/v1/interviews/{id}/segments` answers on prod
      (cohort-only, 20 KiB cap → typed 413, **20 s** budget → typed 504 `sdp_timeout`, no retry, body never logged); coach runs
      500m / 256 Mi, grace 60 s, rollingUpdate 1/0; coach egress admits TCP 443 (recorded in status.md)
- [ ] **AB29–AB30 frozen:** [ds-m6b-01](sprint-ds-m6b-01.md) merged (the merge is the freeze, D40)
- [ ] **S6 result** ([spk-04](sprint-spk-04.md)): the winning shell; **not** "both fail" (then voice waits 3 months — stop); M7 (deploy shape,
      consumed by m6b-02); M13 (GPT-Live rejects browser `session.update`? Realtime accepted an offer with **no data channel**?); whether a
      sideband **event filter** exists; whether the D29 **current-screen item is replaceable in place** on the winner; the exact sideband
      URL and audio event type names; the scrubbed fixtures' path. [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) reads **Accepted**
      ([ds-m6a-01](sprint-ds-m6a-01.md))
- [ ] **`account.region` live** (M1a column, [m1-02](sprint-m1-02.md)) and set by the acceptance step ([l-05](sprint-l-05.md), v1.17.0)
- [ ] **M6a shipped dark** ([m6a-06](sprint-m6a-06.md) patch live): [m6a-01](sprint-m6a-01.md)'s FSM (voice states `connecting`, `held`,
      `bridging` already in the enum), clock, sweeper, client lease (`attach`/`heartbeat`, 409 `lease_lost`), `interview_segment` (the mint log,
      ≤ 1 open segment), `interview_consent`, `AddSpend`; [m6a-02](sprint-m6a-02.md)'s `Segment` interface + `TextSegment`, segment-owned key,
      director, classifier and probe; [m6a-04](sprint-m6a-04.md)'s snapshots, event log (`events.Append`) and bounded SSE; the gateway interview
      proxy + cohort gate; `coach admin interviews --live`
- [ ] **Parallel sessions:** no open peer PR adds a coach goose migration or edits `internal/coach/interview/`, `internal/coach/catalog.go`,
      the gateway interview proxy or identity's internal account handler (`gh pr list`, `git worktree list`, ListAgents); if one adds a
      coach migration, take the next free version at rebase

## Goal

Add voice with **no media on the node**: the S6-winning OpenAI shell behind a `VoiceShell` interface, **coach-brokered SDP** on the
learner's BYO key (no credential of any kind reaches the browser), an outbound **sideband** whose decoder **drops audio copies by event
type without parsing them**, the director brain pushing notes and the D29 "current screen" into the live model, and captions flowing into
the stored transcript and the interview SSE. The key lives as a `[]byte` only while a segment is live. Everything ships dark behind the
M6a cohort gate; the UI is [m6b-03](sprint-m6b-03.md)'s.

## Scope

**In**
- `internal/coach/catalog.go`: the S6 winner's `voice_shell` row final (billing shape, voices, tier, session cap, `as_of`); the losing
  shell's provisional row removed.
- `internal/coach/interview/voice/` (new package): `VoiceShell` + `Sideband` transport interfaces, `voice.Segment` implementing
  [m6a-02](sprint-m6a-02.md)'s `Segment`, the **winner's adapter only**, an in-process fake shell for tests, the audio-drop decoder.
- coach `POST /interviews/{id}/segments` (SDP broker) with its preconditions and typed errors, `POST /interviews/{id}/segments/{n}/confirm`,
  the 15 s orphan reaper, second-tab supersede through m6a-01's client lease, the pre-flight voice-check purpose (≤ 30 s),
  `POST /interviews/{id}/ask` (repeat / rephrase).
- Sideband via `github.com/coder/websocket` (ISC); director push; the D29 current-screen item; captions on m6a-04's event log.
- The voice consent kinds (`internal/coach/interview/consent.go`) and the EU/EEA gate on voice creates and segments.
- One voice-only FSM data-table row (`connecting —interrupt→ interrupted`, task 4) and the voice-mode fixed disclosure turn (task 7).
- An expand migration on schema `coach`: three columns on `interview_segment`, and `interview_event.kind` widened with `caption` and
  `segment` (plus [m6a-04](sprint-m6a-04.md)'s S1 kind allowlist and its test).
- identity: `region` in the internal account response if missing (coach reads it for the EU/EEA gate).
- gateway: the confirm and ask routes in the interview proxy; the pre-flight `voice` block passthrough; `openapi.yaml` / `api.md`.
- Tests: canary, no-audio-persisted, fixture contract, fake-shell integration; docs.

**Out**
- Lease, make-before-break re-attach, cold re-attach, SIGTERM drain; per-segment usage and cost, the $ cap wrap/cut, the L19 voice caps
  (≤ 3 live platform-wide, ≤ 75 voice-min/day), idle hang-up; Standard/Patient/push-to-talk, hold voice, rollover/reseed, AI notes
  checkpoints, transcript self-edit, the `coach-interview` Deployment if M7 failed → [m6b-02](sprint-m6b-02.md). This sprint leaves one
  admission hook (`voiceAdmission`) for m6b-02's caps and uses **Standard** turn-taking only.
- Voice UI (AB29 pre-flight + notices, AB30 HUD, consent screen, self-view, browser gate) → [m6b-03](sprint-m6b-03.md).
- The fake-media e2e on prod and the interviewer GA flip (v2.1.0) → [m6b-04](sprint-m6b-04.md).
- Gateway route, CSP, `Permissions-Policy`, sibling deny, coach sizing and egress → done in [mi-13](sprint-mi-13.md); nothing here widens
  `connect-src`.
- The Realtime `client_secrets` fallback — accepted in [ADR-0032 §7](../../adr/0032-realtime-ai-mock-interviewer.md#7-adr-0007-amendments)
  but **conditional** (Realtime only, only if S6 showed brokering failing) and **not built here** (see task 4). A second shell — never in P1
  ([ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture)).
- Alerts, metrics, opscheck — dropped by D34 (the decoder's per-segment dropped-frame count is one log line at close, task 6).

## Tasks

### 1 · Reconcile the inputs [X]

- **S6 facts.** From the S6 results (**t6 §16**, "S6 results (spk-04, <date>)", appended to [t6](../research/t6-realtime-interviewer.md) by
  [spk-04](sprint-spk-04.md); the scrubbed fixtures under `docs/v2/research/t6-s6-fixtures/` with their `index.json`; the spike-results row in
  [`../status.md`](../status.md)) copy into the PR description a short table: winning shell + model id; endpoint and body shape (GPT-Live
  `POST /v1/live/sessions` JSON `{session, transport:{webrtc, sdp}}`; Realtime `POST /v1/realtime/calls` multipart); sideband URL and auth;
  audio event names; data-channel rule (GPT-Live `oai-events` required, browser may send only `session.close`; Realtime no data channel);
  ICE servers the harness page used (expected none); event filter yes/no; current-screen replaceable yes/no; M1 SDP p95; M8 sideband KB/s.
- **Age of the result.** If S6 ran more than ~8 weeks ago, re-read the model and WebRTC doc pages and re-check the fixtures' shapes first
  (spk-04's risk); record any delta.
- **Route name.** [mi-13](sprint-mi-13.md) shipped `POST /api/interviews/{id}/segments` ("the SDP route") → coach `POST /interviews/{id}/segments`.
  If the live route differs, the live route wins; note it.
- **ADR-0007.** Per BP2, ADR-0032's amendments are folded into the ADRs they amend when 0032 is accepted
  ([ds-m6a-01](sprint-ds-m6a-01.md) task 1). If [ADR-0007](../../adr/0007-ai-coach-byo-key-and-secrets.md) still lacks its
  *Amendment (ADR-0032)* section, add it in this PR with **ds-m6a-01's wording** of
  [ADR-0032 §7](../../adr/0032-realtime-ai-mock-interviewer.md#7-adr-0007-amendments): SDP brokering only, each segment creation logged; the
  Realtime `client_secrets` fallback (30–60 s TTL) **only if S6 showed brokering failing**; the key held as a `[]byte` per live segment,
  dropped on `interrupted` or `paused`, never in panic values or logs. If ds-m6a-01 already folded it, leave it as it is.
- **Record the freeze, if ds-m6b-01's session didn't** ([ds-m6b-01](sprint-ds-m6b-01.md)'s PR merged on CI green and the merge is the
  freeze, D40; idempotent, so skip any edit already done): its Status rows and _Overall_ ✅; status.md Artboards AB29–AB30 "frozen (PR #,
  date)"; `ev-freeze-ds-m6b-01` ✅ ("automatic at the merge"), if status.md still lists it.

### 2 · Catalog + `VoiceShell` + `voice.Segment` + the winner's adapter [X]

Sources: [t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options), [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path),
[t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) ("T5 §9 key and catalog"), [m1-10](sprint-m1-10.md) (catalog).

- **Catalog** (`internal/coach/catalog.go`): the winner's `voice_shell` row loses "provisional" and gains (where M6a hasn't added them):
  `billing_shape` (`per_second` for `gpt-live-1` at $0.05/min, not rounded up; `tokens` for `gpt-realtime-2.1-mini` with audio/text in/out
  and cached prices), `session_cap_s` (mini 3600; GPT-Live the S6 M5 value or 0 = unpublished), `ctx_window`, `voices` (incl. `marin`),
  `min_tier: 1` (no free tier), `billing_url`, `probe_model`, `data_use`, `as_of` = the day the prices were read (never guessed). Remove the
  losing shell's row; keep the 32k-context `gpt-realtime` excluded.
- **Transport interfaces** (`internal/coach/interview/voice/shell.go`) — the shell-specific wire, kept apart from the interview logic:

  ```go
  type VoiceShell interface {
      ID() string // catalog model id of the S6 winner
      Validate(offer SDPOffer) error // one m=audio, no m=video, data-channel rule per shell, ≤ 16 KiB
      CreateSession(ctx context.Context, key []byte, offer SDPOffer, cfg SessionConfig) (Created, error) // {ProviderSessionID, AnswerSDP}
      AttachSideband(ctx context.Context, key []byte, providerSessionID string) (Sideband, error)
  }
  type Sideband interface {
      Events() <-chan Event                      // decoded, non-audio events only
      Push(ctx context.Context, n Note) error    // director note: context or speak
      ReplaceScreen(ctx context.Context, s Screen) error // D29 single current-screen item
      Reconfigure(ctx context.Context) error     // re-apply coach's session config (tamper)
      HangUp(ctx context.Context) error          // end the provider session
      Detach() error                             // close the sideband WITHOUT hanging up (m6b-02 make-before-break)
  }
  ```

  `SessionConfig` = model, pinned voice, instructions (m6a-02's `Prime`: the stable prefix `interviewer-frame@v`, persona, phase goals, the
  public statement, mode rules — **all instruction text is treated as learner-visible**, S4), input transcription on, turn-taking **Standard**
  (m6b-02 adds the others), `store:false`/no provider-side state wherever the endpoint takes it.
- **`voice.Segment`** (`segment.go`) implements [m6a-02](sprint-m6a-02.md)'s `Segment` interface — `Open(ctx, Prime)` (with the offer
  carried alongside `Prime`) creates the provider session and attaches the sideband; `Cue(ctx, Cue)` → `Sideband.Push` (speak);
  `Turn(ctx, CandidateInput, sink)` → a typed candidate input as a conversation item (typed questions, "ask to repeat"); `Close(reason)` →
  `HangUp` and the m6a-02 usage/`cost_micros` write. Every FSM path that closes a segment (interrupt, pause, finish, abandon, cap) therefore
  hangs up the voice session with no voice-specific code in the FSM.
- **Adapter:** `gptlive.go` **or** `realtime.go` — the winner only — built from the S6 fixtures' request/response shapes. Both calls use
  coach's configured `OPENAI_BASE_URL` (the sideband derives `wss://`/`ws://` from it), so the local fake provider can stand in. Provider
  errors go through [m6a-02](sprint-m6a-02.md)'s classifier by `error.code` (`ErrQuota`, `ErrAuth`, `ErrModelAccess`, `ErrRate`,
  `ErrSessionCap`, `ErrTransient`), never by HTTP status.
- **Fakes:** test-only helpers (`fake_test.go`): an `httptest` server that answers session creation with a canned SDP and serves a sideband
  WebSocket replaying scripted frames (transcripts, usage, audio with a canary payload, errors, `session.updated`). For
  [m6b-03](sprint-m6b-03.md)'s compose walk, the PR also describes the two endpoints the scratchpad fake-LLM provider must add (session
  create → canned answer; sideband → scripted captions/usage) — the fake itself stays out of the repo, per the local-review convention.
- **Pinned voice:** the interview's `mock_snapshot.interviewer.voice_default` (snapshotted at create by [m6a-01](sprint-m6a-01.md); default
  `marin`) is validated against the catalog row's `voices` when a voice interview starts (422 `voice_unavailable` otherwise) and used for every
  segment of that interview — rollover, resume, re-prime.
- **Catalog-only shell:** the shell's model id always comes from the catalog row, never from the key, the request or a custom id. The
  brain keeps M6a's rules: a custom brain id may run the interview but [m6a-03](sprint-m6a-03.md) never gives it an AI proposal — add a
  voice-mode test for that.
- **Availability:** extend M6a's setup/pre-flight DTO (the one AB24 reads) with `voice{available, reason, shell, voice, data_channel}`;
  `reason ∈ {region, no_voice_key, model_access, disabled}` (m6b-02 adds `capacity`, `daily_limit`, `minutes_left_today`, the mode and the
  estimate); `data_channel` tells the SPA whether to open `oai-events` (GPT-Live) or not (Realtime). `model_access` is the last recorded
  `ErrModelAccess` for that key.

### 3 · Segment key lifetime (ADR-0007 amendment) [X]

[m6a-02](sprint-m6a-02.md) already holds the decrypted key as a `[]byte` owned by the open segment, zeroed on close / `interrupted` /
`paused`. This task extends that to the voice paths:
- the **sideband**, the director's voice pushes and the probe for a live voice segment use the segment's buffer (no second decrypt);
- the orphan reaper, supersede and `Detach` (m6b-02) zero the buffer on every path that ends the segment in this pod; `held` (m6b-02) too;
- the key type implements `String`, `GoString`, `Format` and `slog.LogValuer` → `[redacted]` (add them if m6a-02 didn't); it is never
  wrapped into an `error`; sideband goroutines `recover()` and log only the panic's type; no `%v` of structs that hold it.
- Tests: after the reaper, a supersede and a sideband-initiated close the backing array is all zeros (test hook); a forced panic in a sideband
  goroutine prints no key bytes; the canary key never appears in captured slog output.

### 4 · SDP broker [X]

Coach `POST /interviews/{id}/segments` behind `requireJWT` (`aud=coach`; the account is the token subject and must own the interview).
Body, decoded with `DisallowUnknownFields` under a 20 KiB `MaxBytesReader`: `{"sdp": "<offer>", "purpose": "live"|"preflight",
"client_id": "<the tab's client id>"}`. Checks, in this order (error names follow [m6a-01](sprint-m6a-01.md)'s where they exist):

| # | Check | Typed error |
|---|---|---|
| 1 | interview exists, owned by the caller, `mode='voice'` | 404 `not_found` |
| 2 | `client_id` holds the interview's client lease ([m6a-01](sprint-m6a-01.md) `attach`) | 409 `lease_lost` |
| 3 | FSM allows a segment: `purpose=preflight` in `preflight`; `purpose=live` in `connecting`, or in `interrupted` for a non-quota reason (the offer fires `cleared` **once `CreateSession` has succeeded**, so a failed re-prime leaves the interview `interrupted` with its grace unchanged — m6a-01 resets the grace on each new `interrupted`; its 60-second-stable rule and ≤ 2 automatic re-primes per grace apply); m6b-02 adds `held` (Talk) and `bridging` (rollover) | 409 `illegal_transition {state, event}` |
| 4 | the voice consent kinds at the current version are recorded for this interview (`interview_consent`; kinds added to `consent.go`, task 4 below) | 409 `consent_required {kinds}` |
| 5 | account region not EU/EEA (identity `GET /internal/accounts/{id}` → `region`, cached 60 s; EEA = EU-27 + IS, LI, NO; NULL → refuse) — the same check refuses `POST /interviews` with `mode=voice` | 403 `voice_unavailable_region` |
| 6 | the `interview` default key ([m1-10](sprint-m1-10.md) `key_default`) is an enabled OpenAI key and the catalog has a `voice_shell` | 409 `voice_key_required` |
| 7 | offer valid for the shell (≤ 16 KiB, exactly one `m=audio`, **no `m=video`**, `m=application` only if the shell requires it — Realtime: absent) | 422 `sdp_invalid` |
| 8 | `voiceAdmission` hook (m6b-02: ≤ 3 live voice platform-wide, ≤ 75 voice-min/day, cap headroom) — a no-op here | 429 / 409 (m6b-02) |
| 9 | an open segment of this interview (m6a-01's partial unique index allows one): an unconfirmed one from the same client is closed first (`close_reason='replaced'`, hung up) | — |

Then: decrypt the key into the segment's buffer → `CreateSession` under a **15 s** context (inside the gateway's 20 s) → insert the **mint**
row in `interview_segment` (m6a-01's columns — `kind='voice'`, `model`, `provider_session_id`, `opened_at` — plus `purpose`, `client_id`)
+ one slog line (interview id, segment `n`, model, purpose — **never** SDP, key, IP or transcript) → attach the sideband (task 6) →
`201 {"segment_n", "sdp": "<answer>", "shell", "confirm_by"}` with `Cache-Control: no-store`. Creation errors keep one HTTP code each, but
the FSM effect depends on the **source state** — m6a-01's `interrupt` fires only from `live`, and pre-flight failures go
`preflight_fail(reason)` → `setup` ([m6a-02](sprint-m6a-02.md)'s reasons `model_access | auth | quota`):

| Classifier | HTTP | `purpose=preflight` (from `preflight`) | `purpose=live` from `connecting` | `purpose=live` from `interrupted` |
|---|---|---|---|---|
| `ErrModelAccess` | 403 `voice_model_access` | `preflight_fail(model_access)` → `setup` | `interrupt(model_access)` → grace | stays `interrupted`; counts as a failed re-prime |
| `ErrQuota` | 409 `provider_quota` | `preflight_fail(quota)` → `setup` | `interrupt(quota)` → m6a-01's grace | stays `interrupted`, grace unchanged (m6a-02's probe arbitrates) |
| `ErrAuth` | 401 `provider_auth` | key disabled (ADR-0007); `preflight_fail(auth)` → `setup` | key disabled; m6a-02's auth path → `paused` | key disabled; `auth` → `paused` |
| `ErrRate` / `ErrTransient` | 502 `provider_unavailable` | state unchanged | state unchanged | stays `interrupted`; counts as a failed re-prime |
| context expiry | 504 `sdp_timeout` | state unchanged | state unchanged | stays `interrupted`; counts as a failed re-prime |

The `connecting` column needs one **voice-only** data-table row, `connecting —interrupt(reason)→ interrupted` (same effects as from `live`:
checkpoint, `grace_until`; nothing to truncate). Add it to m6a-01's table, its model check (unreachable in `mode=text`, whose `connecting`
is immediate) and the table test; m6b-02 maps `held` and `bridging` like `connecting`. `ErrModelAccess` is never retried and, like
`ErrQuota`, **never** disables the key. **No retry anywhere** — a retry opens a second billed session; "state unchanged" means the
learner may try again with a click (AB29 F5b/F5d).

- **Pre-flight purpose:** a ≤ 30 s segment for AB29's voice check; coach speaks AB29 F5's **fixed** greeting ("Hi, I'm an AI interviewer.
  Could you say one sentence so I can check I can hear you?", never model-generated) and hangs up at 30 s; it isn't on the interview clock,
  but it is logged as a mint (m6b-02 counts its seconds and cost).
- **SDP bodies** are never logged, stored or echoed in errors (ICE candidates carry the learner's IP addresses).
- **`client_secrets` fallback** ([ADR-0032 §7](../../adr/0032-realtime-ai-mock-interviewer.md#7-adr-0007-amendments),
  [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)): **accepted but conditional** — Realtime only, a 30–60 s TTL, one
  per segment, logged, and only if S6 recorded that brokering failed on the winner (GPT-Live has no fallback: brokering is its only path).
  It is **not built in this session.** If S6 recorded that brokering failed, **stop and report** (a gate failure, not a review: ⛔ in
  status.md): the fallback puts a short-lived token in the browser and needs [mi-13](sprint-mi-13.md)'s CSP `connect-src` widened to the
  provider origin — its own gateway PR (mi-13's CSP test changed with it) in a follow-up sprint, not a new ADR. Otherwise record "brokering
  works on the winner; fallback not needed".
- **Voice consent kinds** (`internal/coach/interview/consent.go`, versioned like M6a's text kinds — [m6a-01](sprint-m6a-01.md) says "voice
  kinds are M6b's"): `voice_audio_stream` ("Stream my microphone to OpenAI using my key. My voice also passes through xLearn's server in
  memory only; it is never written to disk or logs."), `voice_private_place` ("nearby voices are sent and transcribed too"), and the voice
  reading of M6a's AI-disclosure and assessment kinds, with the exact AB29 F2 strings; `resume_reaffirm` gains its voice text (AB29 F13).
- **Migration** (expand, next free coach version):
  - `interview_segment` + `purpose text NOT NULL DEFAULT 'live' CHECK (purpose IN ('live','preflight'))`, `client_id uuid NULL`,
    `confirmed_at timestamptz NULL`;
  - **widen `interview_event.kind`** with `caption` (task 7) and `segment` (task 5). m6a-01's set (`state|phase|turn|hint|run|snapshot|cap|notice`)
    is closed: drop and re-add its CHECK constraint with the two values added (widening is expand-safe — older code never writes them); if
    m6a-01 enforced the set only in Go, extend the Go enum instead. In the same PR extend [m6a-04](sprint-m6a-04.md)'s **S1 kind allowlist
    and its test** with both kinds and their payload rules: `caption{item_id, role, text, final, approximate}` (text the learner is
    already hearing) and `segment{event, n, purpose, reason}` (never `provider_session_id`, model, SDP or key material).
  - **`interview_turn.source` is unchanged:** voice transcripts reach coach server-side over the sideband, so they are written as
    `source='server'` (the voice origin is the open segment's `kind='voice'`); `client` stays [m6b-02](sprint-m6b-02.md)'s mirror value.
  - Everything else exists (m6a-01: `kind`, `model`, `provider_session_id`, `opened_at`, `closed_at`, `close_reason`, `usage`, `cost_micros`;
    the voice FSM states are already in the enum). **No SDP column, ever.** sqlc queries in `internal/coach/store/queries/`;
    `sqlc generate`; `sqlc diff` clean.

### 5 · Confirm, orphan reaper, supersede [X]

- **Confirm:** `POST /interviews/{id}/segments/{n}/confirm` (gateway: one row in the interview proxy with the default budget;
  `openapi.yaml` + `api.md`) — the SPA calls it when `RTCPeerConnection.connectionState === 'connected'`. Alternatively the first sideband
  event proving media (speech started / input transcription) confirms. First one wins: `confirmed_at`, then m6a-01's `segment_open` →
  `live` (the clock runs from `live`); a pre-flight segment's confirm doesn't touch the clock.
- **Orphan reaper** (a job in m6a-01's sweeper — tick 5 s, `pg_try_advisory_lock`, injectable clock): close segments with
  `confirmed_at IS NULL AND closed_at IS NULL AND opened_at < now() - interval '15 seconds'` (`close_reason='unconfirmed'`, CAS) → `HangUp`
  each (the attached sideband, else the shell's hang-up call) and zero its key; the interview stays in `connecting` (a new offer may follow).
  The same sweep runs once at startup. (Until m6b-02's lease, coach is one pod; m6b-02 routes the hang-up to the holder.)
- **`segment` event kind** (added to [m6a-04](sprint-m6a-04.md)'s event log by task 4's migration; same bounds and S1 allowlist):
  `segment{event: opened|confirmed|closed, n, purpose, reason}` in the same transaction as each segment change, so the HUD and the pre-flight (m6b-03) follow the voice connection; m6b-02 adds
  `reprime` and `rollover`.
- **Supersede** (refresh or second tab, [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)): when
  another tab takes m6a-01's client lease (`attach`), coach closes the open voice segment (`close_reason='superseded'`, `HangUp`) and fires
  `interrupt(tab_closed)`; the old tab's calls get 409 `lease_lost` and its `notice` event (AB30 F15); the new tab's offer from `interrupted`
  fires `cleared` once the provider session exists (check 3) and opens a fresh segment. m6a-01's partial unique index keeps ≤ 1 open segment per interview.
- `pagehide` → the keepalive `…/interrupt {reason: "tab_closed"}` (m6a-01) closes the segment, which hangs up the voice session.

### 6 · Sideband [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) ("The sideband and audio copies", "The browser control surface").

- `go get github.com/coder/websocket` (ISC — note the licence in the PR); `internal/coach/interview/voice/sideband.go` dials the shell's
  sideband URL for the provider session id with the segment key (standard `Authorization: Bearer`), a 10 s dial timeout, compression
  explicitly disabled, and reads frames with `conn.Reader(ctx)` (streaming — never `conn.Read`, which buffers the whole frame).
- **Read limit.** coder/websocket caps every message at **32,768 bytes by default — on `Reader` as well as `Read`** — and closes the
  connection with `StatusMessageTooBig` past it. Real frames exceed that: `session.created` / `session.updated` echo the instructions
  (`Prime` ≤ 8,192 tokens ≈ 32 KB+), and audio frames can be larger. Call `conn.SetReadLimit(n)` right after the dial, with `n` = the
  larger of **1 MiB** and 2× the largest frame in the S6 M8 fixtures; allocations stay bounded by the 512-byte prefix and the per-type
  limits below, not by `n`. State `n` and where it came from in the PR.
- **Audio-drop decoder** (`decode.go`): per frame, read at most a **512-byte prefix** into a fixed buffer and extract `"type":"…"` with a small
  byte scanner (no `json.Unmarshal` of the frame):
  - type ∈ **audio set** (GPT-Live `session.input_audio.append`, `session.output_audio.delta`; the Realtime equivalents; the exact names from
    the S6 fixtures) → `io.Copy(io.Discard, r)`: the payload is never buffered beyond the prefix, never parsed, never logged;
  - type ∈ **allowlist** (input/output transcripts, speech started/stopped, usage, errors, session created/updated/closed, delegation) →
    read the rest through `io.LimitReader(256 KiB)` and unmarshal into a typed struct; a frame longer than that is drained with
    `io.Copy(io.Discard, r)` and dropped, never half-parsed;
  - anything else (unknown type, or no `type` in the prefix) → discard.
  Every frame is read to its end before the next `conn.Reader` call. Dropped unknown or oversized frames are tallied **per segment** and
  written as one slog line when the segment closes (segment `n` + the count; no content) — a debugging aid, not a metric: nothing is
  exported or alerted on (D34). If S6 found a provider-side event filter, request it at attach **and** keep the decoder.
- **Events out** to the interview FSM: `Transcript{role, item_id, text, final}`, `Speech{started|stopped}`, `Usage{…}` (priced by m6b-02),
  `ProviderError{code}` → classifier, `SessionUpdated{ours bool}`, `Closed{reason}`.
- **Tamper** (S6 M13): on a `session.updated` coach didn't send, `Reconfigure` (re-apply coach's config); on the **second**, `HangUp` →
  `interrupted(provider)` and add caveat `session_tampered` (the review treats it like `review_flag`: no `ai-byo`). Realtime runs with no
  data channel, so the browser has nothing to send; the rule still applies.
- **Load:** a benchmark (`BenchmarkDecodeAudioFrame`, 32 KiB and 512 KiB base64 frames) shows allocations bounded by the prefix buffer;
  record decoded KB/s against S6 M8 (≤ 150 KB/s per session) in the PR.

### 7 · Director push, current screen, captions [X]

Sources: [ADR-0032 §3](../../adr/0032-realtime-ai-mock-interviewer.md#3-what-the-ai-perceives-a-second-interviewer-sharing-the-session-owner-d29) (D29),
[t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round), [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (enforcement), [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) S1–S4.

- **Director sink:** [m6a-02](sprint-m6a-02.md)'s brain loop gains a voice sink: notes (≤ 500 tokens) go through `Sideband.Push` — GPT-Live
  `session.thinking.append` (context) / `session.commentary.append` (speak), Realtime `conversation.item.create` (+ a fixed-text response to
  speak). **All evaluative speech is brain-authored** (S2): phase cues, check-ins, the debrief — the voice model never composes feedback.
  **Voice disclosure:** m6a-02's fixed first turn is text-mode copy (AB25 F1: "…I can't see or hear you…"), wrong in voice. Add a
  **voice-mode fixed disclosure turn** with AB30 F1's text (never model-generated), written as the first `interview_turn`
  (`kind='disclosure'`) and **spoken** as the opener of the first live segment; a test asserts a voice interview's first turn is the voice
  string and never contains the text disclosure's "can't see or hear".
- **Ask to repeat / rephrase:** `POST /interviews/{id}/ask {"kind": "repeat"|"rephrase"}` (gateway interview-proxy row; ≤ 6 / min) → a speak
  cue asking the interviewer to repeat or rephrase its last question (AB30 F12).
- **D29 current screen:** on every [m6a-04](sprint-m6a-04.md) snapshot change (sampled every 2–3 s; unchanged state is never re-sent) and on
  every final candidate turn, replace the single "current screen" item (Realtime `conversation.item.delete` + re-create; GPT-Live per the S6
  finding). If S6 found in-place replacement impossible on the winner, send diffs through director notes only and record that. The director
  itself gets diffs.
- **Hints and runs:** `give_hint(n)` is recorded first (M6a), then its text is pushed as a note; the gateway's run echo
  `{passed, total, first_failure_class}` becomes a note. The model has no run tool (S1).
- **Transcript:** final transcripts become `interview_turn` rows (`source='server'` — task 4; role, text; `truncated=true` when a barge-in or hang-up
  cuts an interviewer line) with m6a-02's `turn` event in the same transaction; each final candidate turn also updates m6a-01's
  `last_input_at`. m6a-02's distress **text** check runs on each final candidate turn (it raises the card; it never triggers on voice); the
  never-list scan covers interviewer turns.
- **Captions — one path for both shells** ([m6b-03](sprint-m6b-03.md) reads only this; the data channel is not a caption source): partial
  transcripts become `caption{item_id, role, text, final:false, approximate}` events on [m6a-04](sprint-m6a-04.md)'s event log (the kind
  task 4's migration adds; same bounds and S1 allowlist: coalesced ≥ 300 ms per item, ≤ 4 KiB, `approximate=true` on candidate lines), and the final text arrives as the `turn` event.
  Caption events carry transcript text, so they join the transcript's C4 retention and erase. Latency over SSE: +0.2–0.5 s.

### 8 · Tests + optional live check [X]

- **Unit:** the offer table (valid audio-only; `m=video` → 422; data channel present on Realtime → 422; 16 KiB + 1 → 422); the precondition
  order (one test per row of task 4) and the per-source-state error effects (task 4's second table, incl. the voice-only
  `connecting —interrupt→ interrupted` row in the FSM table test and model check); decoder: an audio frame whose payload is garbage after
  the prefix decodes without error and without allocation beyond the prefix; unknown, prefix-less and oversized allowlisted frames are
  dropped and the per-segment tally is logged once at close; the tamper rule (first → reconfigure, second → hang-up + caveat); m6a-04's S1
  kind allowlist test covers `caption` and `segment`; the voice disclosure turn (never the text one's "can't see or hear").
- **Large frames through a real socket:** over the fake shell's real coder/websocket pair (not a byte-slice decoder test), a **> 32 KiB
  `session.updated`** and a **> 32 KiB audio frame** (and one at the chosen read limit) are read without `StatusMessageTooBig`; the
  sideband stays attached and the next transcript frame still arrives.
- **Integration** (real Postgres per the service workflow; the fake shell over `httptest` + WebSocket): broker 201 with the answer SDP; each
  typed error; the mint row has no SDP and no IP; the reaper hangs up an unconfirmed segment at 15 s (fake clock); a second tab's `attach`
  hangs up the open voice segment and the old tab gets 409 `lease_lost`; an offer from `interrupted(network)` fires `cleared` after `CreateSession` succeeds, and a failed one leaves `grace_until` unchanged; every FSM exit
  from `live` (interrupt, pause, finish, abandon, `cap_100`) hangs up the fake session; key zeroed on each path; captions appear as `caption`
  then `turn` events in order.
- **No audio persisted (the acceptance test):** a scripted fake call whose audio frames carry a canary base64 string; afterwards scan every
  text/jsonb/bytea column of every `coach` table, the captured slog output and any panic output for the canary → **0 hits**; the transcript
  canary appears only in `interview_turn`, never in logs.
- **Canary** ([t5](../research/t5-platform-ai.md) pattern, extended per [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)
  "Logging"): `/api/interviews/*` and the sideband decoder never log transcripts, audio, SDP, prompts or completions.
- **Contract:** the adapter's request (minus secrets) equals the S6 fixture's shape; the fixture's answer and sideband frames parse.
- **Gateway:** the confirm and ask routes (cohort gate, `aud=coach`); the OpenAPI drift test.
- **Optional live check** — pre-approved by launching the prompt (D40; ≈ $0.10, ≤ 2 minutes), and run only if the owner's before-launch
  item is in place: **his own OpenAI key**, entered by him in the local compose Settings and set as the `interview` default (the agent never
  types a key). The session records the voice consent for its test interview (compose test data; the audio is a fake clip); Chrome
  with `--use-fake-ui-for-media-stream --use-fake-device-for-media-stream --use-file-for-fake-audio-capture=<clip.wav>`; from DevTools:
  `getUserMedia({audio:true})` → `new RTCPeerConnection()` (+ `createDataChannel('oai-events')` only on GPT-Live) → `createOffer` →
  `fetch('/xlearn/api/v1/interviews/<id>/segments', {method:'POST', …, body: JSON.stringify({sdp, purpose:'preflight', client_id})})` →
  `setRemoteDescription(answer)` → `connected` → the greeting caption arrives on SSE → hang up; then `coach admin interviews --live` is
  empty and the no-audio scan is repeated. Without that key, record "live leg deferred to [m6b-04](sprint-m6b-04.md) task 2" (not ⛔: the
  check is optional).

### 9 · Docs + record [X]

- `docs/architecture/services.md` (coach: the voice module, sideband, key lifetime), `api.md` + `openapi.yaml` (segments, confirm, ask, the
  `voice` block, error codes), `data-model.md` (new columns, the widened `interview_event.kind`), `events.md` (the `caption` and `segment` event-log kinds; still no NATS subject).
- `docs/v2/status.md`: Sprint board row; M6b 🔄; Artboards AB29–AB30 frozen (task 1); decisions log — the winner and its catalog row
  `as_of`, the precondition order and error codes (and their per-source-state FSM effects), confirm = browser call or first media event,
  supersede through the client lease, the current-screen mechanism, the `caption` and `segment` kinds (and the widened `kind` set), voice
  turns as `source='server'`, the sideband read limit, the `session_tampered` caveat, "brokering works; `client_secrets` fallback not
  needed", whether the live check ran; a hand-off
  line for [m6b-03](sprint-m6b-03.md): the `voice` block fields, the routes, the event kinds and the fake-provider endpoints for its compose walk.

## Acceptance criteria

- [ ] Browser ↔ provider WebRTC with the SDP brokered via coach: the fixture contract and fake-shell integration pass, and the live check
      connected (or is recorded as deferred to m6b-04 because the owner's key wasn't in local compose at launch); no response to the browser
      ever carries a credential
- [ ] **No audio bytes persisted or logged** (canary scan over every coach table, logs and panics = 0 hits); audio frames are discarded unparsed
- [ ] Every segment creation is logged as a mint row with no SDP and no IP; SDP never appears in logs
- [ ] An unconfirmed session is hung up at 15 s; a second tab's lease takeover hangs up the first tab's session; ≤ 1 open segment per interview;
      every FSM exit from `live` hangs up the provider session
- [ ] The key is held as a `[]byte` per live segment and zeroed on every path that ends it (test); never in logs or panic output
- [ ] Server-side preconditions hold: client lease, FSM state, voice consent (409 `consent_required`), EU/EEA region (403
      `voice_unavailable_region`, NULL refused, voice creates too), OpenAI interview key with a catalog shell, audio-only offer
- [ ] Captions reach the event log as `caption` → `turn` for the winning shell (one path)
- [ ] Tamper: an unsent `session.updated` is re-applied, a second hangs up
- [ ] The sideband survives > 32 KiB frames (`SetReadLimit` at attach; real-socket test); `caption`/`segment` are in the widened `kind` set and the S1 allowlist
- [ ] CI green (`go test ./...`, `go vet`, lint, `sqlc diff`, OpenAPI drift, web suite untouched); merged; nothing tagged

## Release

**Merge only (ships dark in [m6b-03](sprint-m6b-03.md)'s `v2.0.x` patch).** The API is reachable only by the T-3 cohort through M6a's
interview proxy, and no UI calls it until m6b-03. If a peer session cuts another `v2.0.x` patch first, this code rides it, still dark.
No infra PR: coach's TCP 443 egress was confirmed by [mi-13](sprint-mi-13.md) and coach → identity `:8081` exists since [l-01](sprint-l-01.md)
(verify in the rendered NetworkPolicy; if coach → identity isn't there, add it in its own infra PR — ADR-0035 §2 standing rule). No NATS
subject, stream or consumer changes, so no ACL PR. No new always-on pod (memory-sum rule unaffected).

## Definition of Done

CI green · merged via PR (squash) · no tag · acceptance criteria met · ADR-0007 carries the ADR-0032 §7 amendment · statuses updated (this file,
[ds-m6b-01](sprint-ds-m6b-01.md)'s freeze if its session didn't record it, [`../status.md`](../status.md)) · local `main` synced.

## Risks / watch-outs

- **Provider API drift since S6.** GPT-Live and Realtime-2.1 are weeks old at S6: re-run the scrubbed fixtures against the current docs; if a
  shape changed, fix the adapter and note it; if the change breaks brokering itself, stop and report (a gate failure: ⛔ in status.md) —
  ADR-0032 §7's `client_secrets` fallback is conditional (Realtime only) and first needs mi-13's `connect-src` widened in its own gateway
  PR, in a follow-up sprint.
- **Frame size.** coder/websocket's default 32 KiB read limit closes the sideband on the first large `session.updated` or audio frame —
  and tamper detection with it; `SetReadLimit` at attach and the > 32 KiB real-socket test are the guard.
- **Parsing audio by accident.** `json.Unmarshal` on a whole frame, `conn.Read`, or logging a frame on error all break the "unparsed" promise the
  consent makes; the prefix scanner plus the allocation benchmark are the guard.
- **Double-billed sessions.** Any retry — gateway, coach or a naïve SPA — opens a second provider session on the learner's key; the broker never
  retries and `client_id` supersede hangs up the old one.
- **Timeouts.** A provider call finishing after coach's 15 s context may leave a session coach never learned about; it was never connected, and the
  S6 M1 p95 (≤ 3 s) makes this rare — record any sighting.
- **Region is an attestation** (ADR-0033): the gate refuses NULL and EU/EEA codes; it isn't geolocation.
- **Key leaks in panics.** A struct holding the key printed by a recover handler is the classic leak; the `LogValuer`/`Format` methods and the
  forced-panic test cover it.
- **Coach restarts cut calls until m6b-02.** This sprint's code has no lease or drain yet. That only matters once a UI can start a call:
  even if a peer's `v2.0.x` patch carries this code, the API is cohort-gated with no caller, and the UI ([m6b-03](sprint-m6b-03.md)) comes
  after m6b-02.
