> **T6 research appendix.** Method: four parallel research slices (realtime model options; architecture and media path; failsafes and BYO billing; assessment, privacy and ethics) were synthesized into one draft. The draft faced two adversarial critiques: feasibility, latency, cost and solo ops (6.5/10) and privacy, ethics and security (7/10). This is the revised final. Model names, prices and limits were verified on the web on 2026-09-24; **the voice APIs are weeks old, so S6 re-checks them.** Decided in [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (Proposed).
>
> **Status:** settled with the owner 2026-09-24 (**D28–D31**, §13). §13 overrides the body where they conflict, notably:
> - the AI follows the **on-screen answer widgets at a 2–3 s change cadence and on every turn**, and "shares the session like another interviewer" (D29);
> - no video to the AI; peer video interviews are a future item with TURN deferred;
> - the public profile shows the **mock count only** (D31).

# T6: Realtime AI mock interviewer (voice + video)

This is research and design only. No repo files were changed, the VPS and cluster were not touched, no service was signed up for, and nothing was spent. The contested facts were re-checked on the web on 2026-09-24; sources are at the end. "(inferred)" marks my own reasoning.

Cost scripts:
- the original cross-check: `scratchpad/t6synth/cost.py`
- this revision, which adds the director brain on both shells and the time multipliers: `scratchpad/t6synth/cost2.py`

---

## 1. Recommendation in one paragraph

**T6 is not in v2.0.** It is milestone **M6**, which starts after M3 (the judge `mock` context and CodeMirror) and can run alongside M4/M5. The proposed labels are **v2.1 = P0 + P1** and **P2 in v2.2+**, and T6 never gates the v2.0 tag.

The interviewer is **a text-model "director" brain plus one thin voice shell**. It lives in a new `internal/coach/interview` module in coach, which already holds the raw BYO key, so no ninth service is needed.

**P0 is a typed interviewer.** It runs over the existing SSE path on any `interview` catalog model, Anthropic or OpenAI, and ships the owner's three failsafes:
- a 5-minute top-up grace;
- a pause of at most 24 hours from the first pause;
- a resume modal with the deterministic interview state plus an AI summary generated on the learner's key.

The transcript is stored inline in coach's own schema. After the session, coach makes an `ai-byo` proposal, and **the learner must explicitly accept it** before `ScoreMock` is called, once.

**P1 adds voice.**
- The browser talks WebRTC directly to OpenAI. Coach brokers the connection setup (SDP) server-side, so **no credential of any kind reaches the browser**.
- A coach-held sideband carries transcripts, director notes, quota detection and hang-up.
- No RTP touches the node, and no change to UDP, Traefik WebSockets or object storage is needed.
- The sideband *does* deliver transient base64 copies of the audio into coach memory. Coach drops them unparsed, and the consent text says so.

**The shell is chosen by a spike of at most one day on the owner's key (S6), run before M6a's design freeze.**
- `gpt-live-1` costs a flat $0.05/min and is full duplex: about **$3.1 per 45-minute interview and $4.0 per 60-minute one**, brain included. It is chosen only if it passes every hard gate, including interjections during think-aloud, push-to-talk emulation and code-question honesty, **and** the owner rates it at least 1 point more natural than `gpt-realtime-2.1-mini`.
- `gpt-realtime-2.1-mini` costs about **$2.1 / $2.8** typically and $8–11 in bad cases, and has native Patient and push-to-talk modes.
- P1 ships **exactly one shell**.

**The coach's voice** is one pinned catalog voice per course persona (default `marin`). It is identical across rollovers and resumes.

**What the AI judges:** what was said and coded, **never how the candidate sounded or looked**.
- Video is an optional local self-view, off by default, and never sent or stored.
- Audio is never stored.
- In voice mode, Communication is self-scored (the AI supplies verified quotes) until a multi-speaker ASR fairness check passes.

**Launch limits:**
- voice runs on Chrome/Edge only;
- EU/EEA residents get no voice until a legal review;
- Anthropic-only learners get the text interviewer.

| At a glance | Decision |
|---|---|
| Milestone | M6 (post-M3). Proposed v2.1 = P0 text + P1 voice; P2 in v2.2+ |
| Owner service | coach (`internal/coach/interview`); split into a second Deployment of the same image only if S6 M7 fails |
| Media path | Browser↔OpenAI WebRTC. Coach brokers the SDP and holds the sideband. Nothing on UDP, Traefik WebSockets or object storage |
| Shell | S6 bake-off: GPT-Live-1 (preferred on naturalness and flat price) vs Realtime-2.1-mini. One ships |
| Brain | Text model on the same `interview` key. It pushes director notes off the critical path and writes all evaluative speech |
| Scoring | Transcript, code, judge aggregate evidence and a server timeline → proposal kept in coach → explicit learner accept → `ScoreMock` once |
| Stored | Transcript (C4, inline, deleted 30 days after scoring by default). No audio and no video, ever |

## 2. Model & pipeline options

TTFA = time to first audio **token**, as measured by Artificial Analysis. That token can be an acknowledgement rather than content, so TTFA is not our substantive-answer latency. Costs include the director brain and ≈ $0.09 of review and brief (§8), on the learner's key.

| Option | Providers / keys | Latency | Barge-in / turn-taking | Video | Session limit / resume | Cost 45 / 60 min | BYO-key fit | Verdict |
|---|---|---|---|---|---|---|---|---|
| **A. GPT-Live-1 shell + coach director brain** | OpenAI key (Tier 1+; **free tier unsupported**) | AA lists "GPT-Live-1 (Sol, low)" at 1.24 s TTFA, with the backend configuration undisclosed. **That is not our path.** Our director pushes notes, so most turns need no round trip. Client delegation adds Mumbai↔US hops plus a non-streamed append of ≤ 500 tokens: est. 2.5–5 s (inferred). Measured in S6 M2 | **Model-controlled** ("listens while it speaks and decides when to stop"). No VAD or eagerness settings. Patient = prompting. Push-to-talk = emulated through `session.input_audio.mute`/unmute plus `track.enabled` (S6 M3b) | None (audio and text only) | Duration limit **unpublished** (`expired` close reason). 128k context; above 90% a replacement engine keeps the instructions plus ≤ 8,192 tokens. Seeding ≤ 128 msgs / 8,192 tok. We reseed from our own text state | **≈ $3.1 (2.9–3.3) / $4.0 (3.8–4.2)**. Voice is flat at $2.40 / $3.15, billed per second with no round-up | Good: server-side session creation only; per-second usage events; Tier-1 limit of 25 concurrent sessions | **Preferred if it wins S6** |
| **B. `gpt-realtime-2.1-mini` + the same director** | OpenAI key | 0.85 s (minimal) / 4.28 s (high). With director notes, minimal effort should suffice (inferred) | Native: `semantic_vad` eagerness low or medium; `turn_detection:null` = real push-to-talk; `interrupt_response`; WebRTC truncates unplayed audio | Image input only | **60-minute hard cap.** Planned rollover at the start of Code | **≈ $2.1 (1.6–2.9) / $2.8 (2.1–3.9)** with a rollover. **No-cache worst case $8.3 / $11.2** | Good. Tier-1 40k TPM is tight late in long sessions. The data channel can probably be omitted (verify in S6), which removes browser tampering | **Co-candidate. It ships if A fails a hard gate or the naturalness margin** |
| C. `gpt-realtime-2.1` | OpenAI | 0.97 / 1.21 s | as B | as B | 60 min | ≈ $4.0 / $5.0 with a rollover; worst $24 / $28+ | Costly and volatile | Catalog-only entry **if B's adapter ships** (no extra code); otherwise not planned |
| D. Voice-lite (Web Speech recognition → brain over SSE → `speechSynthesis`) | Any key, including Anthropic | ≈ 1.6–2.7 s, half duplex (inferred) | Local VAD stops playback; headphones advised | n/a | Our own segments | ≈ $0.5 / $0.7 | Audio stays on the device **only** with Chrome's on-device recognition (`processLocally` plus a language pack). Otherwise the browser's cloud recognizer receives the audio. OS voices ≠ the coach's voice | P2, **only if Anthropic-only learners ask for it** |
| E. Hosted pipeline with third-party speech keys | 2 extra key types | ≈ 1–1.5 s | Good | n/a | ours | ≈ $1.3 / $1.7 | New key types and new data processors | Rejected |
| F. Gemini Live | Google key (Google is not a v2 BYO provider, T5 §9) | 1.18 s | Good | ≤ 1 fps | 2 min audio+video without compression; connection ≈ 10 min (from the slice) | ~$5–8 (from the slice) | A token sits in the browser | Rejected; revisit only if Google returns to BYO |
| **G. Text interviewer** | Any catalog chat model | ~1–2 s to first token | Typed | n/a | Our own segments | **≈ $0.45 / $0.65** | Perfect | **Adopt as P0 and as a permanent accessible mode** |
| H. Self-hosted media (LiveKit/Pipecat) or our own WebSocket audio relay | any | +VPS hop | — | — | — | Node cost | Needs UDP or TURN; all audio through a 128 Mi pod | Rejected |

**Notes:**
- **Anthropic:** Claude takes text and images in and produces text out. There is no audio API, so there is no Claude voice shell.
- **Browsers:** Firefox WebRTC sessions to OpenAI Realtime disconnect after 1–2 turns (openai-agents-js #1353, open since 2026-05-21; the only workaround is the WebSocket transport, which we rule out). **Voice therefore launches on Chrome/Edge only, gated by capability and user agent.** Firefox gets text mode with an explanation. Safari is enabled only if its S6 leg passes.
- **"One brain" is not a differentiator between A and B.** The director pattern works on both. The choice rests on measured turn-taking, naturalness and the flat price.

## 3. Architecture & media path

**Owner: coach, module `internal/coach/interview`.**

| Question | Answer |
|---|---|
| Why coach | The raw key must stay in coach (T5 §10), and SDP brokering and the sideband start there. "Coach is locked during a mock" becomes a local check. Coach already has an erase consumer |
| Why not assessment | T1, T2 and T4 §6.7 keep transcripts and AI proposals out of it. Assessment keeps `mock_session` and stays the **only** writer of the mock signal (`ScoreMock`) |
| Why not a ninth service | It would still need coach on the key path: an extra hop with no extra isolation. Split-out triggers: open signup (D13 lifted), more than 3 concurrent interviews, or coach above 70% of its memory limit |

```
Browser SPA /xlearn/<course>/mock/live        (voice: Chrome/Edge; other browsers → text mode)
 │ mic ═══ WebRTC ICE/DTLS/RTP  browser ↔ OpenAI only — never via the VPS ═══════════►  OpenAI  gpt-live-1 session | realtime call
 │ captions ◄═ oai-events data channel (GPT-Live: required; browser may send only        ▲
 │             session.close per docs — S6 M13)   (Realtime: no data channel — S6)       │ sideband WSS, OUTBOUND from coach,
 │ camera ─► <video muted> local self-view; OFF by default; never in the PeerConnection  │ raw BYO key (standard auth).
 │                                                                                       │  in : transcripts, usage, errors, delegation,
 ├─ POST /api/interviews/{id}/segments  (SDP offer ≤16 KiB) ─► Traefik ─► gateway        │       + transient base64 audio copies
 │        (route timeout 20 s) ─► coach ─► POST /v1/live/sessions | /v1/realtime/calls  │       → dropped by type, payload never parsed
 │     ◄── SDP answer. Coach hangs up any session the browser hasn't confirmed in 15 s   │  out: director notes, phase cues, run echo,
 ├─ GET  /api/interviews/{id}/events     bounded SSE ≤50 s, Last-Event-ID (DB seq)       │       mute/unmute (PTT), hang-up
 ├─ POST /api/interviews/{id}/turns      text mode (chat relayStream pattern)            ┘
 ├─ PUT  /api/interviews/{id}/code       ≤64 KiB, ≥2 s debounce
 ├─ POST /api/interviews/{id}/run        gateway ─► judge(context=mock) ─► runner;  gateway ─► coach: {passed,total,first_failure_class}
 ├─ POST /api/interviews/{id}/{probe|ptt|hint|hold|interrupt|pause|resume|brief|finish|abandon}
 └─ Finish ─► judge final(mock) aggregate evidence ─► coach review call (BYO text, store:false) ─► proposal (coach)
            ─► learner explicit Accept / Edit / Re-propose once / Self ─► gateway ─► assessment ScoreMock ONCE

Director brain (off the critical path). Triggers: a substantive candidate turn, a code change (≥20 s debounce),
a run echo or a phase boundary → one text call on the same key → a note of ≤500 tokens:
    GPT-Live: session.thinking.append (context) or session.commentary.append (speak)
    Realtime: conversation.item.create
Client delegation is used only when the live model asks. The brain writes all evaluative speech.
```

**Credential model** (it satisfies T5's "derived credentials … or streams proxied by coach"):
- **SDP brokering only.** The browser posts its SDP offer. Coach posts it to OpenAI with the decrypted key:
  - GPT-Live: `/v1/live/sessions` with JSON `session` + `transport{webrtc,sdp}`; this is the only path GPT-Live offers;
  - Realtime: `/v1/realtime/calls`, multipart; this is the "unified interface" OpenAI recommends.
- **Every segment creation is logged as the "mint":** `interview_segment(provider_session_id, model, created_at, account HMAC, est_cost)`. The SDP is **never logged**, because its ICE candidates carry the learner's IP addresses.
- **Fallback, Realtime only, and only if brokering fails S6:** a `client_secrets` token with a 30–60 s TTL, one per segment, logged.
- **Key lifetime (ADR-0007 amendment):**
  - the key is decrypted into a `[]byte` held only while a segment is live, shared by the sideband, probes and brain calls;
  - it is dropped on `interrupted` or `paused` and decrypted again on resume;
  - it never appears in panic values or logs, and the canary test covers this.

**The sideband and audio copies.** OpenAI documents no filter and no transcript-only attach: the sideband "also receives copies of subsequent input and output audio". So:
- coach uses a streaming frame decoder that reads each event's `type` and **discards audio events without unmarshalling their payload** (`session.input_audio.append`, `session.output_audio.delta`, and the Realtime equivalents);
- nothing is ever written to disk or logs, and the T5 canary test is extended to the sideband;
- S6 checks whether a filter option exists and uses it if so;
- §1, §7 and the consent text state this truthfully.

**The browser control surface:**

| Shell | Browser can… | Mitigation |
|---|---|---|
| GPT-Live | The docs list only `session.close` as a browser-sent event | S6 M13 tries `session.update` and instruction events from DevTools. If they are accepted, coach re-applies its configuration and hangs up on a second attempt |
| Realtime | Change instructions with `session.update` when a data channel exists (demonstrated in the 2025-05 community thread) | **Open the connection with no data channel.** Captions come from the sideband over our SSE (+0.2–0.5 s, inferred). If S6 shows the channel is mandatory, coach re-applies its configuration on any `session.updated` it didn't send and hangs up on the second |
| Both | — | The score never comes from the realtime model; it comes from coach's own transcript and review. All instruction text is treated as learner-visible (S4, §7) |

**Surviving deploys.** Flux rolls coach on every release tag (the ImagePolicy is `>=1.0.0` with a 1-minute poll; `replicaCount: 1`, RollingUpdate), and AGENT.md mandates a ship at the end of every session.

| Mechanism | Detail |
|---|---|
| Graceful drain | On SIGTERM, coach flushes the code snapshot and transcript tail, releases the lease and closes its sideband. `terminationGracePeriodSeconds: 60` |
| Make-before-break | `maxSurge: 1, maxUnavailable: 0`: the new pod takes the lease and attaches a second sideband before the old one detaches. Audio never stops, because it runs browser↔OpenAI. **S6 M7 is hard** |
| If M7 fails | Run the interview module as a **second Deployment of the same coach image** (`ROLE=interview`) with its **own ImagePolicy, bumped manually** when no interviews are live. The gateway routes `/api/interviews/*` to it. It is still one service and one schema; expand/contract migrations keep the lagging role compatible |
| Ops | opscheck lists live interviews. The release runbook checks that count before tagging |

**Infra needs vs. today:**

| Need | Today | T6 | Change |
|---|---|---|---|
| UDP / TURN | ufw allows TCP 22/80/443 only | none (ICE is browser↔OpenAI) | **none** |
| WebSocket through Traefik | none | none (the sideband is outbound) | **none** |
| Long-lived browser stream | chat SSE | bounded SSE (≤ 50 s, `Last-Event-ID`) | none |
| SDP round-trip | gateway `httpc` timeout 10 s (`gateway/coach.go:43`) | one **20 s** route timeout, plus coach reaping orphans after 15 s | gateway code |
| Request sizes | 1 MiB body | SDP ≤ 16 KiB, code ≤ 64 KiB | none |
| Coach resources | 250m / 128 Mi limit | **500m / 256 Mi**, `GOMEMLIMIT` ≈ 200 MiB (S6 M8 sizes this) | HelmRelease values |
| Coach rollout | defaults | `terminationGracePeriodSeconds: 60`, `maxSurge: 1`, `maxUnavailable: 0` | HelmRelease / chart values |
| WebSocket client | none | `coder/websocket` (ISC) | go.mod |
| Permissions-Policy | none | SPA: `camera=(self), microphone=(self)` | gateway |
| Shared origin `projects.sujaykumar.dev` (landscape, airlift, kubescope, projects-hub) | — | **P1 gate:** `camera=(), microphone=()` on every sibling router | Traefik headers middleware in `../infra` |
| CSP | none | **P1 gate:** `script-src 'self'; connect-src 'self'` (the page never calls api.openai.com; `connect-src` doesn't govern WebRTC) | gateway |
| Coach egress | 443 works | plus WSS on 443 | allow it if T7 adds a default-deny egress policy |
| Object store | none (D11) | none | **none** |

**Limits and abuse caps.** A same-origin script could start sessions on the learner's key. That is no worse than today's same-origin risk to any xLearn API, but it costs more, so these caps bound the damage:

| Cap | Value |
|---|---|
| Non-terminal interviews per account | 1 |
| Live voice interviews, platform-wide | ≤ 3 (until a split-out trigger) |
| Starts per account per day | ≤ 2 |
| Voice minutes per account per day | **≤ 75 × the time multiplier** |
| Per-interview $ cap | **mandatory**; default = "plan for" × 1.5, learner-editable. At 85% the interviewer wraps up; at 100% the session goes to `finished(cut_short=cap)` |
| Onboarding advice (copy) | a dedicated OpenAI project for xLearn with a **hard spend limit**, a key restricted to realtime/live/responses, and a key expiry date |

## 4. Session state machine & the owner's failsafes

**The clock.**
- **Coach is the single writer**: `active_ms` plus `live_since`, and the clock runs only in `live`, `bridging` and `wrapping`.
- Phase = active time measured against the manifest rail × the learner's time multiplier.
- Assessment's `deadline_at` is not authoritative for `text` or `voice` mocks. The v1 `classic` mock is unchanged.

**Segments.**
- Each provider session (or text run) is a segment. A partial unique index allows at most one live segment per interview.
- Transitions use `UPDATE … WHERE state=$expected`, the same score-once pattern as `ScoreMock`.
- **Rollover, grace recovery, resume, network recovery, "hold voice" and idle reconnects are one mechanism: a new segment primed from our own text state.** No provider-side state is relied on.

```
setup ─► preflight (20–30 s real voice check) ─► connecting ─► live ◄──► bridging (planned rollover at a phase boundary)
                                                  │   └─ held (learner "Hold voice while I code": segment closed, clock runs; Talk ─► connecting)
                                                  ├─► interrupted(reason)   [clock frozen at t_last_good; grace_until = +5 min]
                                                  │      ├─ cause cleared ─► connecting ─► live   (free re-prime; cleared only after 60 s stable; ≤ 2 auto re-primes per grace)
                                                  │      ├─ "Finish & get feedback" (≥ 75% done, key has credit) ─► wrapping
                                                  │      └─ grace expired | "Save & resume later" | ErrAuth ─► paused
                                                  │   paused  [resume_by = first_paused_at + 24 h; ≤ 3 pauses]
                                                  │      ├─ resuming (probe → paid AI brief on click → modal) ─► connecting ─► live
                                                  │      └─ sweeper: now > resume_by ─► incomplete (terminal, unscored)
                                                  ├─► wrapping (rail done | Finish | 85% $ cap) ─ ≤ 3-min debrief (brain-authored) ─► finished
                                                  │      finished ─► review call ─► proposed ─► explicit Accept/Edit/Re-propose once/Self ─► ScoreMock ─► scored
                                                  │      finished, not scored in 30 days ─► incomplete
                                                  └─► abandoned (explicit; terminal)
```

**Classifier.** Errors are shared and typed; T5's P0 fix is a prerequisite. OpenAI errors are classified by `error.code`, never by HTTP status, because a 429 can mean either rate limiting or no credit.

| Class | OpenAI codes | Anthropic codes (text) | Action |
|---|---|---|---|
| `ErrQuota` | `insufficient_quota`, `credit_balance_exhausted`, `project_spend_limit_exceeded`, `organization_spend_limit_exceeded`, `organization_usage_limit_exceeded`, whether in an `error` event, `response.done{status:failed}` or a failed session creation | 400 "credit balance is too low", 402 `billing_error`, spend limit | grace; **never disable the key** |
| `ErrAuth` | 401 `invalid_api_key` | 401 | `paused`; key disabled (ADR-0007) |
| **`ErrModelAccess`** (new) | `model_not_found`, permission or tier codes at session creation (GPT-Live has no free tier) | 404 model | **never retry, never disable.** "Voice needs a paid OpenAI account (Tier 1+)", with an offer of text mode. Usually caught at pre-flight |
| `ErrRate` | `rate_limit_exceeded` | 429 `rate_limit_error` | ≤ 3 × 20 s, then `interrupted(rate)` and a rollover to a small context |
| `ErrSessionCap` | `expired` / `session_expired` | — | free reseed |
| `ErrTransient` | 5xx, `server_error`, WebSocket 1011, ICE `failed` while online | 5xx / 529 | 2 new-segment tries in 30 s, then `interrupted(provider)` |
| Unknown | — | — | ≤ 2 retries, then the **same-key probe** decides |

The probe is a 16-token text call on the brain model (≈ $0.0001): quota if it fails with quota, otherwise transient. How dry credit shows up in the middle of a session is **undocumented**. S6 captures fixtures for the **spend-limit** path only, and the probe decides for prepaid exhaustion until real fixtures exist.

**Failsafe 1 — the 5-minute top-up grace.** P0 ships it in text form with a simple modal; P1 adds the countdown UI.
1. **At the first `ErrQuota`:**
   - freeze `active_ms`;
   - set `grace_until = now + 5 min`;
   - write a **deterministic checkpoint** (phase, per-phase ms, code SHA plus a ≤ 16 KiB copy, `upto_turn_seq`, hints given). There is no AI call;
   - **hang up the provider session** (GPT-Live bills connected seconds);
   - mark a cut-off interviewer sentence `truncated`, so it is asked again.
2. **One immediate probe.** If it passes, the error was a false positive: go to `connecting`, with the 60-second-stable rule.
3. **The modal** (read-only editor; mic shown muted; camera stopped):
   - "Your OpenAI credits ran out — interview paused at 23:41."
   - **[Top up ↗]** from the catalog `billing_url`, plus: "Top-ups can take a few minutes to register. OpenAI first deducts any negative balance, so add at least **⟨remaining estimate + $1⟩**. If it doesn't register in time, we'll save your place, and resuming is quick."
   - A 5:00 countdown and "Checked 20 s ago".
   - Buttons: **[Check now]** (at most once per 10 s), **[Save & resume later]** and **[End interview]**. At ≥ 75% of the rail there is also **[Finish & get feedback]**; while the key is still dry it becomes **[End without feedback]**.
4. **Probes** are driven by the browser every 30 s. The server allows at most 1 per 10 s and at most 15 per grace, and the sweeper runs a final probe at `grace_until`.
5. **If the probe succeeds within the grace:** a new segment is primed from the checkpoint plus the **verbatim transcript tail**. **There is no paid summary.** The clock restarts on the interviewer's first audio.
6. **When the grace expires:** `paused`. This is expected to be the usual outcome, and it is cheap.

**Failsafe 2 — pause for at most 24 hours.**
- `resume_by = first_paused_at + 24 h` is absolute, and at most 3 pauses are allowed.
- A banner shows on every device: "Interview paused — resume by Thu 14:05".
- **While paused, coach is unlocked**, and `withhold()` treats the mock's item as live.
  - `withhold()` gates only coach, and the arena is unrestricted (D17), so the gateway records any arena **view, reveal or submit on the mock's item** during the pause. On resume, a hit sets `pause_exposure`.
  - `pause_exposure` shows as "Viewed the solution during the pause" and makes the session ineligible for `ai-byo`.
- **After 24 hours → `incomplete`** (O3a):
  - unscored and left out of trends;
  - a free partial view from the deterministic checkpoints;
  - paid partial improvement areas **only on a click**;
  - **no AI call ever runs on the learner's key without their action.**

**Failsafe 3 — the resume modal (the owner's spec).**
1. Opening a paused interview shows the **free deterministic state at once**:
   - the phase rail, with elapsed time against budget per phase and the total remaining;
   - hints given;
   - the last code snapshot, the Run count and the last verdict class;
   - the pauses and their reasons;
   - an **estimate to finish**.
2. **[Summarise & resume — uses your key (~$0.04)]** runs the probe first. If the key is still dry, it shows "Still no credit" and nothing is charged.
3. **The AI brief** is one BYO text call (`store:false`) streamed over SSE.
   - Input: the full transcript so far plus the deterministic state.
   - Output: `{candidate_summary ≤600 chars, progress[≤5], next_step, interviewer_brief ≤1.2k tok}`.
   - **The brief is cached for this pause.** If the resume fails again, for example on a negative balance, it is not charged a second time.
4. **Buttons:** **[Resume interview]** (this click also satisfies the autoplay and `getUserMedia` gesture requirement), **[Finish now & get feedback]** (≥ 75% done), **[Abandon]**.
5. **Priming the new segment:**
   - a stable instruction prefix (interviewer frame at its version, persona, problem, public descriptors, mode rules);
   - the brief and the phase state;
   - the last ≤ 4 turns as text;
   - the cue "welcome back in one sentence, recap in ≤ 2, continue with ⟨next_step⟩".

   This fits GPT-Live's seeding limit of ≤ 128 messages / 8,192 tokens.

**Checkpoints:**
- **Deterministic** ones are free (P0): at every phase boundary and on every interruption.
- **AI notes** (P1) are a side output of the director at phase boundaries, disclosed in the estimate. They hold ≤ 3 evidence bullets per public dimension with turn references, `covered[]`, `pending_followups[]` and `next_step`.

**Planned rollover:**
- **Realtime:** at the start of Code when the segment is over 40 minutes or its context is over ~24k tokens. It is forced by any 60-minute rail or any multiplier above 1×.
- **GPT-Live:** only if S6 finds a duration limit shorter than the interview, or when `usage_ratio` passes 0.8, which pre-empts the provider's 8,192-token replacement.

**Other failure modes:**

| Failure | Detection | Clock | Path |
|---|---|---|---|
| Network drop or laptop sleep | browser `offline`; ICE `disconnected` > 5 s; heartbeat missing 30 s | stops | `interrupted(network)` → free re-prime within the grace → else `paused` |
| Refresh or second tab | stale client lease | stops | the new segment **hangs up the orphaned session**; the other tab shows "live in another tab" |
| SDP answer never confirmed | no data-channel or first event within 15 s | — | coach hangs up the orphaned provider session |
| Tab close | `pagehide` → `sendBeacon(/interrupt)` | stops | grace → `paused` |
| Provider outage | 5xx / 1011 / `server_error` | stops | 2 tries → `interrupted(provider)`; the probe separates an outage from quota |
| Mic lost | track `ended` / permission change | stops after 10 s | "Reconnect mic"; after 30 s `interrupted(mic)` |
| Camera lost | track `ended` | runs | silent (video is never used) |
| Key revoked | `ErrAuth` | stops | `paused`, key disabled; resume with any enabled `interview` key |
| No model access / free tier | `ErrModelAccess` | — | blocked at pre-flight; offer text mode |
| Top-up too small (negative balance deducted) | `ErrQuota` again after resume | stops | back to `paused`; cached brief, no new charge |
| Coach restart or deploy | sideband drops; audio keeps flowing | runs | make-before-break re-attach (M7). If that fails, `interrupted(provider)` → free re-prime; the transcript gap is filled from the browser mirror (`source=client`) |
| Credit runs out at ≥ 90% | `ErrQuota` | stops | default "Finish without the rest" → `finished(report_pending=quota)`; self-score always available |
| **Idle in any phase** | 3 min × multiplier with no speech, no edits and `document.hidden` | stops | "Still there?" → `interrupted(idle)` and **hang up** (OpenAI advises closing idle sessions) |

**Coach lock:** `locked` in `connecting`, `live`, `held`, `bridging`, `interrupted`, `resuming` and `wrapping`; **unlocked in `paused`**.

**Mock item selection:** skip items that `withhold()` marks live (an open counted attempt, or a due or live touch). **There is no start refusal (D27).**

## 5. The coding round

- **Editor:** CodeMirror 6 (the T4 decision) in "interview mode" with no autocomplete, per the artboard. No canvas in DSA mocks.
- **Snapshots:** `PUT …/code {rev, text, cursor_line}`, at most one per 2 s, ≤ 64 KiB, enforced with `MaxBytesReader`. Coach keeps the latest in memory and persists it every 30 s, at each phase boundary and at SIGTERM. The canonical draft is judge's `mock`-context draft.
- **How the interviewer sees the code (both shells, push model):**
  - On a code change (≥ 20 s debounce) the director gets a stable prefix, then the **numbered diff since the last push** plus the current function. It does **not** get the full 64 KiB file each time.
  - The director pushes a ≤ 500-token observation, for example "[editor] L12–18 now use a hashmap; possible off-by-one at L15; the last run passed 7/9".
  - The live model is instructed to **say "let me look" rather than guess** about code it hasn't been told about. S6 M12 measures the bluff rate.
  - This is generalised per T0's part registry as `view_work(part)`. A future canvas course pushes its `canvas-graph@1` text export.
- **Runs are started by the learner only:**
  - `POST …/run` → gateway → judge (`context=mock`, `context_id=mock_session_id`), at most 1 per 10 s (T3 §10). Mocks are never shed, and no media reaches the runner.
  - The gateway returns the learner-visible DTO and sends coach a compact echo.
  - **The model has no run tool.** The interviewer sees only what the learner sees (S1).
- **Hints (S4):**
  - The hint ladder never sits in any model context.
  - A hint is released only through a server-side `give_hint(n)` action, which the learner requests or the director triggers per policy (at most one conceptual hint per phase, never code). The server **records it first**, then appends its text.
  - The review also flags unlogged hints it finds in the assistant transcript.
- **Phase rail:**
  - The manifest's `mock.rail` (DSA: 0–5 Clarify, 5–10 Brute force, 10–18 Observation→plan, 18–33 Code, 33–40 Trace+edges, 40–45 Complexity+follow-ups) × the multiplier.
  - A 60-minute option adds a follow-up phase, followed by a 3-minute debrief.
  - Phase changes are server-timer cues to the director and HUD nudges. A 0.33× "slice" rail exists for QA only.
- **Quiet during Code**, except when the candidate asks something, on a Run result worth probing, or after ≥ 90 s × multiplier of silence with no edits.
  - Realtime: `semantic_vad` eagerness `low`.
  - GPT-Live: prompting only. S6 M3 is a hard gate on interjections while typing and thinking aloud.
  - **"Hold voice while I code"** (P1, learner-initiated) closes the segment and **Talk** reopens it (a free reseed). It saves cost on long or multiplied Code phases.
- **At Finish:** the gateway submits `final` (`mock`) to judge. **Only aggregate evidence** reaches the proposal: hidden passed/total, first-failure class and `evidence_hints` caps. Mock `evaluation_completed` stays off.

## 6. Assessment

**Inputs are text and deterministic data only:** the transcript, code, judge evidence, a server timeline and the hint ledger.

| Dimension (DSA 7, /35) | Evidence (closed enum: `transcript \| code \| judge \| timeline \| hints`) | Voice-mode default |
|---|---|---|
| Communication (**content only**: explains before coding, narrates decisions, checks assumptions, answers what was asked) | transcript, timeline | **`self_only`**: the AI gives verified quotes and the learner sets the band, until the multi-speaker check passes |
| Problem understanding | transcript; timeline (first clarifying question before the first edit) | AI proposes. Timeline evidence is **advisory** in voice mode, because ASR can lose a question |
| Brute force | transcript | AI proposes |
| Optimisation | transcript, judge, code | AI proposes |
| Code quality | code | AI proposes |
| Edge cases | judge, transcript, code | AI proposes |
| Complexity | transcript, code | AI proposes |

**Never assessed:**
- tone, confidence, nervousness, enthusiasm or any emotion;
- accent, pronunciation, fluency, filler rate, words per minute;
- face, eye contact, expression, posture, appearance, background;
- personality, culture fit, honesty, and any demographic inference.

**Why:**
- **EU AI Act Art. 5(1)(f)** has applied since 2025-02-02. The Commission guidelines put *voice*-based inference in scope, read "education" broadly (including training and distance learning), count hiring as workplace, and limit the safety exception to protecting life and health. Inference from written text is out of scope.
- OpenAI's usage policies prohibit the same, and the risk would land on *the learner's* account.
- ASR is uneven across speaker groups, and video adds almost no signal.

**Enforcement, now extended to the live voice model:**
1. The evidence enum has no `audio`, `video`, `prosody` or `face` member. A CI lint checks dimension ids, labels and descriptors against the never-list.
2. **All evaluative speech is brain-authored.** The debrief and any feedback line are written by the text brain from content-only evidence and spoken through `session.commentary.append` or a fixed-text response. The voice model never composes an assessment.
3. Interviewer instructions forbid adapting to, or remarking on, perceived tone or emotion.
4. **Distress and self-harm handling triggers on words**, through a server-side text check on the candidate transcript that raises a UI card (Tele-MANAS **14416** plus "your local emergency number") and offers a pause. It never triggers on voice.
5. The assistant output transcript is scanned against the never-list afterwards, and a hit sets `review_flag`.
6. A deny-lexicon runs on the written review: an offending sentence is dropped and `review_flag` is set.

Delivery timing (talk share, longest silence) is shown for information only. It is hideable, never scored and never public.

**Flow:**
1. **`wrapping`:** a debrief of at most 3 minutes with 2–3 improvement areas, **brain-authored**. It never states a score.
2. **`finished` → one review call** by coach on the brain model (`store:false`).
   - Inputs: public descriptors, the stored transcript, final code, judge aggregate evidence, the timeline and the hint ledger, as delimited data. **No pack material is included.**
   - Output: `xlearn.mock_review@1` (≤ 8 KiB): per-dimension `band` 1–5 or `null` with a `null_reason`, verified quotes, `why`, ≤ 3 strengths, ≤ 3 improvements (each an observable behaviour plus a drill link to a validated course reference), a summary and caveats.
   - It is stored in coach's `interview_proposal` together with **`model` and `prompt@v`**, outside assessment.
3. **Validation:**
   - quotes are checked against the transcript with T4's normalise-then-substring check;
   - `evidence_hints` act as caps;
   - T4's injection heuristic applies;
   - **missing evidence → `band:null`** (T4's "clamp ≤ 2" does not apply to mocks).
   - **Custom model ids never get an AI proposal**; the session is self-scored with the AI's quotes only.
4. **Review screen:** Accept all · Edit per dimension · **Re-propose once** (blind, fresh sample; D14 parity) · Score it myself. In voice mode (P1), the learner may first correct **their own** transcript turns, which sets `transcript_edited`.
5. **Submit → gateway → `ScoreMock` once.**
   - **`scored_by=ai-byo` only if all of these hold; otherwise `self`:**
     - the model is a non-custom catalog model that passed the twin gate;
     - every band was accepted unchanged;
     - there are zero `edited` or `source=client` turns;
     - `pause_exposure` is not set;
     - `review_flag` is not set;
     - the learner explicitly clicked Accept.
   - **No auto-accept for mock proposals** (D14's 24-hour auto-submit does not extend to mocks). An untouched proposal stays private in `proposed`, with reminders at 24 h and 7 days. A finished session not scored within 30 days becomes `incomplete`.
   - `trust=honor` is always shown. "AI proposed 24 · you scored 27" stays in coach.
   - The improvement areas pre-fill `mock_session.notes` (≤ 2 KiB, no quotes). Afterwards the coach in `review` mode can discuss them, as the artboard's post-mock coach panel shows.
6. **Public route:** pending **O4**. The recommendation is to keep count/best/average labelled **"practice mock scores (self-reviewed)"**, with no AI wording and no `scored_by` split. This avoids an "AI-assessed score" reaching recruiters, which would drift toward EU AI Act Annex III 4(a).
7. **Fairness gates:**
   - **Twin test, per catalog model that can act as reviewer, before M6a ships:** `mock_review@1` × 3 samples on matched transcripts (clean, ASR-noisy, non-native grammar, and one with an injected instruction). Every non-injection twin must stay within ±1 band on every dimension, and the injection twin must get `review_flag`. A model that fails gets `ai: self_only`.
   - **Multi-speaker check, P2, before voice Communication becomes AI-proposed:** at least 10 consenting speakers across accents, including at least one disfluent speaker, read scripted answers through the real shell, and the result must show ±1-band parity.

## 7. Privacy, consent, retention & accessibility

**Data inventory:**

| Data | Leaves the browser | Stored by xLearn | Class | Retention / erase |
|---|---|---|---|---|
| Mic audio | Voice mode: to the **learner's own** OpenAI org | **Never** | C4 | OpenAI: 30-day abuse logs, no training by default. `/v1/live/sessions` keeps no state unless `store:true` (we send `false`). Eligible for zero data retention |
| **Audio copies on the sideband** | OpenAI → coach | **Never**: transient in coach memory, dropped by event type unparsed, never on disk or in logs | C4 | the life of the event frame |
| Camera video | **No** (off by default; local self-view only) | Never | C4 | — |
| Transcript and code in **text-model calls** (director, review, brief) | To the learner's provider: OpenAI Responses with **`store:false`**, or Anthropic Messages | — | C4 | the provider's 30-day abuse logs, no training by default |
| Transcript (both sides, with `source`, `truncated` and `edited` flags) | — | coach `interview_turn`, inline, ≤ 256 KiB (60 min ≈ 55 KB) | C4 | **Default: deleted 30 days after scoring** (or 30 days after finish or incomplete). Opt-in keep for 12 months. Per-session and **per-turn delete** (for bystander speech). Erase consumer: delete plus tombstones |
| Checkpoints, brief, proposal, delivery stats | via BYO calls | coach | C4 | deleted with the transcript |
| Timeline (phases, pauses, VAD ms, runs, `pause_exposure`) | — | coach | C3 | same as the transcript |
| Mock Runs / final | judge `mock` | judge | C3 | learning record |
| Score, notes, `scored_by`, `format`, caveats | — | assessment | C2/C3 | learning record. The public route shows only aggregates (O4) |
| Segment log (session ids, model, usage, µUSD) | — | coach | C2 | 90 days; **included in the erase consumer** (tombstone) |
| Consent records | — | coach | C1/C2 | life of the account, then a tombstone (erase consumer) |

Hostinger's weekly images (D12) can hold erased or expired rows for up to one image cycle. **The privacy notice discloses this.**

**Consent (DPDP-grade; the core duties apply from 2027-05-13; built now):**
- **Account level:** an unticked "AI mock interviewer" opt-in, with a "how it works and what it evaluates" card.
- **Per session, before starting, with unticked boxes:**
  - "Stream my microphone to OpenAI using my key. My voice also passes through xLearn's server **in memory only**; it is never written to disk or logs."
  - "My transcript and code are sent to my AI provider's text API for the interviewer, summaries and feedback."
  - Transcript retention: **delete 30 days after I score** (default), or keep 12 months.
  - "The interviewer is an AI. It can't see you."
  - "Assessment uses what you say and your code, never your voice, face, accent or appearance."
  - "Take this somewhere private: nearby voices are sent and transcribed too."
- A compact reaffirmation on resume. The consent version and timestamp are stored.
- **AI disclosure:** "I'm an AI interviewer" is spoken at the start. This covers EU AI Act Art. 50 (from 2026-08-02) and OpenAI's requirement to disclose AI voices.
- **EU/EEA:** voice mode is **not offered to EU/EEA-resident learners** until a legal review. The invite records the region (T7), and text mode remains available. The design keeps emotion inference out, but the live speech model still perceives prosody. *Not legal advice.*
- **India:** there is no statutory rule on recording consent, and covert recording is treated as a privacy violation (Art. 21). The only party is the consenting learner and **no audio is stored**. The remaining risk is **bystanders**, handled by the notice. 18+ is reaffirmed.

**Logging.** T5 §8 applies: never log transcripts, audio, SDP, prompts or completions. The canary test covers `/api/interviews/*` **and the sideband decoder**.

**Safety invariants:**
- **S1.** The interviewer's context contains only what the learner may see right now: the public statement, released hints, public descriptors and the learner-visible judge DTO. A reference solution is included only once the item has concluded.
- **S2.** The interviewer never scores and never composes evaluative speech; the brain does.
- **S3.** Transcripts never reach another user, a platform-key call, a calibration set or the public route.
- **S4.** Hint text enters a model context only through `give_hint(n)`, recorded first. **All instruction text is treated as learner-visible**, and S1 is linted against it.

**Conduct rules** (in the prompt and in QA):
- never ask about age, caste, religion, family, health, disability, nationality or salary history;
- never comment on appearance, voice, accent or emotion;
- never claim to be human.

**Voices:** stock provider voices only; no cloning.

**Accessibility:**
- **Modes: Voice · Voice with push-to-talk · Text.** Text is P0, first-class and never penalised (`mode=text`; delivery stats not applicable). Push-to-talk and Patient are native on Realtime and text, and emulated on GPT-Live, gated by S6 M3b.
- **Screen-reader preset:** push-to-talk, a headphones prompt, and `aria-live` limited to phase changes and the grace warning (`assertive`), so screen-reader speech doesn't trigger barge-in. Captions stay in a navigable log.
- **Captions on by default.** Interviewer captions come from the model's own output transcript; candidate captions are labelled "approximate". An "Ask to repeat / rephrase" button sends a text event.
- **Time multiplier 1× / 1.25× / 1.5× / 2×**, chosen before starting (WCAG 2.2.1).
  - It is badged and never penalised, and no diagnosis is collected.
  - **Its added cost is shown at pre-flight (§8).**
  - **Idle and check-in timers scale with it.**
  - "Hold voice while I code" reduces the cost.
- **Camera off by default.** `getUserMedia({video})` is called only when the learner turns self-view on. There is no "camera required" gate.
- Shortcuts for mute, push-to-talk and end that don't clash with the editor; `prefers-reduced-motion` respected; indicators use icon plus text, not colour alone.

## 8. Cost on BYO keys

**Assumptions:**
- A 45-minute rail = 48 minutes of wall time, with ~20 min of candidate speech, ~9 min of AI speech and ~55 responses. A 60-minute rail = 63 minutes, ~26 / 12 min and ~72 responses.
- The director brain on `gpt-6-sol` ($2 / $0.20 / $10): 40 / 52 calls of ≈ 10k context, 60–80% cached, 500–1,000 output tokens including reasoning.
- Realtime re-reads the whole conversation on every response.
- The recomputed figures are from `cost2.py`.

| Option | 45 min: typical (range) | 60 min: typical (range) | 1.5× / 2× on a 45-min rail | Worst case | Notes |
|---|---|---|---|---|---|
| **GPT-Live-1 + director** | **$3.1** (2.9–3.3) | **$4.0** (3.8–4.2) | $4.3 / $5.6 (60-min rail: $5.7 / $7.3) | idle connected time is billed → idle hang-up and "hold voice" | Voice $2.40 / $3.15 flat; brain $0.4–0.8 / $0.55–1.0 |
| **Realtime-2.1-mini + director** | **$2.1** (1.6–2.9) | **$2.8** (2.1–3.9) with a rollover | $2.4 / $2.8 (60-min rail: $3.3 / $4.0) | **$8.3 / $11.2** with no cache | Silence isn't billed; cost depends on cache hits; Tier-1 TPM |
| Realtime-2.1 (catalog only) | ≈ $4.5 | ≈ $5.5 with a rollover | — | $24+ / $28+ | not planned unless B ships |
| Voice-lite (P2, on demand) | ≈ $0.5 | ≈ $0.7 | — | — | brain-only |
| **Text interviewer (P0)** | **≈ $0.45** | **≈ $0.65** | ≈ +20–40% | — | any chat model |
| Extras on the same key | review ≈ $0.06 (+ $0.06 for a re-propose); resume brief ≈ $0.03–0.05 (cached per pause); AI checkpoints ≈ $0.06–0.12 (P1); pre-flight voice check ≈ $0.02; probe ≈ $0.0001 | same | — | — | all shown in the estimate |

**What the learner's card pays:** add tax and forex. OpenAI charges **18% GST** to Indian buyers without a GSTIN (community report), plus the card's forex fee. For example, $3.1 → ≈ $3.8.

**Pre-flight screen (before Start):**
- **Estimate:** "Typical **$3.1** · plan for **$3.5** · + tax where applicable (India: 18% GST) · billed by OpenAI to your key. OpenAI deducts any negative balance first: keep at least **$4.5** available, or turn on auto-recharge."
  - Duration-billed shells: minutes × rate × multiplier, plus the brain estimate.
  - Token-billed shells: the sum over the manifest's `mock.voice_profile` phases at a cache hit rate of 0.9. "Plan for" is the high end of the range.
  - Once the learner has at least 2 sessions, the p50 / p90 of their own `segment.cost_micros` per active minute replace the defaults (P2).
- **Checks:**
  - the key is enabled;
  - **a 20–30 s real voice check (≈ $0.02).** The interviewer says hello and asks for a sentence, which proves tier and model access, the ICE path, the voice, mic levels and captions. A failure offers text mode;
  - the browser gate (Chrome/Edge) and the EU gate;
  - the mandatory $ cap is shown and editable;
  - consent.
- **Live meter:** hidden during the interview. Shown in the grace, pause and resume screens and in the report.

## 9. Phased plan

| Step | Scope | Needs from T7 / infra | Milestone |
|---|---|---|---|
| **Spike S6** (§10) | ≤ 1 day, owner's key, enforced $10 hard limit. Decides the shell, the deploy shape (M7) and the quota fixtures | owner approval (**O1**) | **Before the M6a design freeze.** Can run any time; independent of M1–M3 |
| **P0 — text interviewer and the failsafes (M6a)** | <ul><li>`internal/coach/interview`: brain loop, voice-ready segment abstraction, server clock and rail</li><li>manifest `mock.interviewer{persona, voice_default, phase_goals[] (public), max_minutes ≤ 60, debrief_minutes 3, camera:"self_view_optional"}`, `mock.rubric.dims[].evidence` and `ai: propose \| self_only \| self_only_voice`</li><li>the classifier, including `ErrModelAccess`, and the probe</li><li>**grace (simple modal), pause ≤ 24 h, resume modal with AI brief (cached per pause), sweeper, `incomplete`**; deterministic checkpoints; `pause_exposure`</li><li>`give_hint`; review → proposal → explicit accept → `ScoreMock` once; re-propose once; twin gate</li><li>`interview_*` tables, consent, retention (30 days after scoring), erase (including the segment log and consent rows)</li><li>**`store:false` acceptance test**; CodeMirror interview mode; judge `mock` Runs and final; assessment deltas</li><li>text-mode accessibility; replay-based brain regression tests (CI uses recorded fixtures; a manual live replay costs ≈ $0.5)</li></ul> | **M1:** coach P0 fixes (typed errors, quota ≠ disable), the `interview` default key and catalog. **M3:** judge `mock` context and the editor. Bounded SSE. **No infra change** | **M6a** after M3, alongside M4/M5. Proposed **v2.1** |
| **P1 — voice, one shell (M6b)** | <ul><li>the S6-winning shell adapter behind a `VoiceShell` interface</li><li>SDP broker (20 s route, 15 s orphan reaper); sideband with the audio-drop decoder; director push; captions</li><li>Standard / Patient / push-to-talk (native or emulated); "hold voice while I code"</li><li>countdown grace UI; AI notes checkpoints; learner transcript self-edit; rollover</li><li>lease, make-before-break re-attach and SIGTERM drain; per-segment usage and cost; **mandatory $ cap**; pre-flight voice check; idle hang-up</li><li>optional self-view camera; voice consent (including audio through the server); Chrome/Edge gate; EU gate; pinned coach voice</li><li>manual Chrome fake-media end-to-end run (`--use-fake-device-for-media-stream --use-file-for-fake-audio-capture`, recorded clips, 0.33× slice rail, ≈ $0.3 per run on the owner's key)</li></ul> | <ul><li>coach 500m / 256 Mi</li><li>`terminationGracePeriodSeconds: 60`, `maxSurge: 1`</li><li>`coder/websocket`</li><li>**P1 gates:** SPA Permissions-Policy, sibling-app camera/mic deny middleware (`../infra`), xLearn CSP</li><li>gateway 20 s SDP route</li><li>egress 443 if a policy lands</li><li>opscheck counters (live, paused, incomplete, sweeper lag, stuck `live`)</li><li>release-runbook live-interview check</li><li>invite region field</li><li>**if M7 failed:** a `coach-interview` Deployment with its own ImagePolicy</li></ul> | **M6b**, **v2.1** (with P0) or v2.1.x |
| **P2 — extras (M6c)** | <ul><li>opt-in "show your work" photo sent to the brain, not stored (O2b)</li><li>chat-coach read-aloud in the interviewer's voice</li><li>opt-in local recording download (never uploaded)</li><li>history-calibrated estimates</li><li>**multi-speaker fairness check** → AI-proposed voice Communication</li><li>Safari, if it failed in S6 and has since been fixed</li><li>voice-lite only on demand</li><li>optional weekly 30 s canary on the owner's key (≈ $0.03)</li></ul> | none server-side | **M6c**, v2.2+ |
| **P3 — only on triggers** | stored audio playback (T2's B2 design); a Gemini adapter if Google returns to BYO; a second shell only if the first is deprecated; splitting out an `interview` service | media bucket; a new service | later |

## 10. The smallest spike

**S6: a voice-shell bake-off between GPT-Live-1 and `gpt-realtime-2.1-mini` in our architecture.**

**The exact question.** Run on the owner's Mac with Chrome stable, on his home network in India, with coach-style SDP brokering, a sideband and a director brain (`gpt-6-sol`) on the same key. For each shell:
1. Does it hold 60 minutes, or reseed cleanly?
2. Does it stay quiet through think-aloud, typing and 45-second silences on laptop speakers without headphones, and support push-to-talk?
3. Does it answer substantive turns fast enough without bluffing about code?
4. Does it surface a classifiable quota signal?
5. Does it survive a server restart without dropping audio?
6. Does its cost land within ±25% of §8?
7. Can the browser alter the session?

**Environment:**
- **The harness:**
  - a throwaway directory in the scratchpad, **not the repo**;
  - a localhost Go program: SDP brokers for both endpoints, a sideband client with the type-filtering decoder, a stub director, and a JSONL log with audio dropped and no SDP;
  - a small proxy using our `http.Server` timeouts;
  - one static page, plus short Firefox and Safari legs.
- **The OpenAI account:**
  - a new throwaway project, with a **project hard spend limit of $10** (enforced, not an alert). OpenAI says enforcement "is not instantaneous", so the harness stops itself at $8 from its own usage tally;
  - a project-scoped key, restricted where the UI allows, held only in an environment variable;
  - the project and all logs are deleted afterwards.
- Nothing touches the VPS or the cluster.
- **It runs only with the owner's explicit go-ahead (O1). No agent uses the key without it.**

**Steps (time box: 1 working day, ≤ 8 h hands-on, plus an unattended soak):**
1. **Build the harness** (1.5 h).
2. **A scripted 15-minute DSA mock on GPT-Live** (1 h). The owner is the candidate. The script includes:
   - 30 DSA terms (for M11);
   - 3 silences of 45 s;
   - **2 minutes of typing while thinking aloud** without headphones;
   - 2 barge-ins;
   - 3 minutes of **emulated push-to-talk** (`track.enabled` plus sideband `session.input_audio.mute`/unmute);
   - 5 code questions after director pushes;
   - 1 run echo;
   - 3 latency probes in each mode: **push-only**, **client delegation** and **Responses delegation**.
3. **Restart the harness mid-call** (15 min): attach a second sideband *before* killing the first (make-before-break), then a cold re-attach by session id. Note whether audio continued.
4. **Tamper** (10 min): from DevTools, send `session.update` and instruction events over the GPT-Live data channel. On Realtime, check that an offer with **no data channel** is accepted, and that `m=video` is refused.
5. **A 65-minute soak** (unattended; Chrome fake-capture loops a recorded prompt every 2 minutes). Record the close reason and time, `usage_ratio` at 60 minutes, sideband bytes per second, and harness memory and CPU.
6. **Quota** (30 min), on a second throwaway project with a **$1 hard limit**:
   - drive a live session to failure;
   - capture the first failing events (sideband and data channel), the error code for creating a new session, and the overshoot;
   - record **whether the active WebRTC session is killed or keeps running**;
   - raise the limit and time how long until the probe succeeds;
   - **label the fixtures "spend-limit path"**.
7. **Reseed** (20 min): a new session seeded with the brief plus the last 4 turns (≤ 8,192 tokens). Measure TTFA; the owner rates continuity 1–5.
8. **Repeat on mini** (1.5 h): steps 2 (10 minutes), 3, 4, 6 and 7, plus native `turn_detection:null` push-to-talk, `semantic_vad` low, and the cached-token ratio before and after a 15-minute silence.
9. **Browser legs and SDP** (20 min): Firefox and Safari for 5 minutes each; inspect the answer SDP for ICE-TCP / TLS candidates; look for a sideband event filter.
10. **Passive prepaid timing (no extra spend):** at the owner's next real top-up, time purchase → probe success on his normal key.

**Measurements** (H = hard, S = soft):

| # | Measurement | GO threshold |
|---|---|---|
| M1 (H) | SDP broker round-trip p95 | ≤ 3 s |
| M2 (H) | **Substantive-answer latency p95** (end of candidate speech → first content word), push-only | ≤ 2.0 s (delegated ≤ 3.5 s is soft; if it fails, delegation isn't used) |
| M3 (H) | False interjections during silences and typing-while-thinking-aloud, no headphones | ≤ 1 per 10 min |
| M3b (H) | Push-to-talk: model speech that starts while input is muted; response after release | 0; ≤ 2.5 s |
| M4 (S) | Barge-in: time until the interviewer stops | ≤ 500 ms (inferred) |
| M5 (H) | Session duration | ≥ 60 min, or a clean `expired` close plus a reseed with TTFA ≤ 3 s |
| M6 (H) | Quota signal (spend-limit path) | a classifiable code at coach within 5 s; session closable; fixtures captured |
| M7 (H — **decides the deploy shape**) | Make-before-break or cold re-attach with audio uninterrupted | pass → one coach Deployment; fail → a `coach-interview` Deployment before P1 |
| M8 (S) | Sideband load | ≤ 150 KB/s per session; harness ≤ 100 MiB; decoder CPU recorded (sizes T7's coach limits) |
| M9 (H) | Cost from usage events | within ±25% of §8 |
| M10 (S) | Reseed continuity | owner rating ≥ 4/5 |
| M11 (S) | DSA-term ASR fidelity | ≥ 90% (below 80% → timeline evidence dropped in voice mode) |
| M12 (H) | Code-question bluffing | 0 wrong claims in 5 probes |
| M13 (H) | Browser can't alter the session | GPT-Live rejects browser `session.update`; Realtime works with no data channel, or re-applying the configuration works |
| M14 (S) | Naturalness | owner rating 1–5 per shell, same script |
| M15 (S) | Browsers | Chrome/Edge pass; Firefox and Safari results decide the gate list |

**Decision rules:**
- **GPT-Live passes every H and M14 ≥ mini + 1:** GPT-Live becomes the default.
- **Otherwise, mini passes every H:** mini becomes the default, with rollover at the start of Code.
- **Both fail:** P0 text only; revisit voice in 3 months.
- **M7 fails:** add the `coach-interview` Deployment to P1.
- **Expected spend:** ≈ $6–7. The hard limit is $10 (enforced), and the harness stops at $8.
- **Deliverables:** a one-page results note, and **scrubbed** fixtures (synthetic ids, no SDP, placeholder transcripts) before anything is committed. The repo is public.

## 11. What T6 constrains downstream

**T7 (build, rollout, operations):**
- **Milestone:** M6 = M6a (P0), M6b (P1), M6c (P2). It comes after M3, can run alongside M4/M5, and **is not in v2.0**. Proposed labels: v2.1 = M6a + M6b.
- **Gates:**
  - S6 before the M6a design freeze;
  - M1's coach P0 fixes (typed errors including the full quota list and `ErrModelAccess`, and the probe);
  - the `interview` default key and catalog;
  - M3's judge `mock` context and CodeMirror;
  - the twin gate before M6a ships.
- **Infra (P1, all small):**
  - coach 500m / 256 Mi, `terminationGracePeriodSeconds: 60`, `maxSurge: 1` / `maxUnavailable: 0`;
  - `coder/websocket`;
  - the gateway's 20 s SDP route;
  - **P1 gates:** SPA Permissions-Policy, the sibling-app camera/mic deny middleware, and the xLearn CSP;
  - egress 443;
  - the `coach-interview` Deployment only if M7 fails.

  **No UDP, Traefik, ufw, namespace or object-store change.**
- **Operations:**
  - opscheck counters: live, paused, incomplete, sweeper lag, stuck `live`;
  - the release runbook checks for live interviews before tagging;
  - privacy-notice additions (sideband audio in memory, text-API processing, backup lag);
  - the consent version;
  - an invite `region` field for the EU gate;
  - a recommendation to consider a dedicated xLearn subdomain before D13 is lifted.
- **QA:** replay regression in CI; a manual fake-media end-to-end run; the slice rail.
- **Artboards (none exist yet):**
  - setup, consent and pre-flight estimate;
  - the live HUD (rail, captions, mic and camera state, editor, Run, Hold / Talk);
  - the grace modal;
  - the paused banner;
  - the resume modal;
  - the debrief and proposal review;
  - text mode;
  - accessibility settings;
  - the browser-unsupported and EU notices.

**Amendments to settled documents:**

| Doc | Amendment |
|---|---|
| ADR-0007 | Two amendments. (1) Realtime credentials: **SDP brokering, so no credential leaves coach**, and each segment creation is logged; fallback: Realtime `client_secrets` with a 30–60 s TTL, one per segment. (2) Key lifetime: the key is held as a `[]byte` **per live segment**, dropped on `interrupted` / `paused`, never in panic values or logs |
| T5 §9 key and catalog | `interview` default = `{provider key, brain model, voice shell?}` on **one** key. The "must be `realtime`-capable" rule is dropped: text needs `interview_brain`, and voice needs `voice_shell`. Catalog fields: `billing_shape`, `session_cap_s`, `ctx_window`, `voices`, `probe_model`, `billing_url`, `data_use`, `min_tier`. **Custom model ids: text interviewer allowed, but never an AI proposal.** Exclude the 32k-context `gpt-realtime` |
| T5 §9 P0 typed errors | Classify by `error.code`, including `credit_balance_exhausted`, the spend-limit codes and realtime `response.done{failed}`; Anthropic's 400 "credit balance is too low"; **`ErrModelAccess`**; the probe as arbiter |
| T5 P1 `store:false` | Becomes an **M6a acceptance test** for every interview text call |
| T5 §2 row 42 / §9 (stale "refused while a counted attempt is open") | Superseded by D27. T6 applies `withhold()` in **item selection** |
| T5 §9 locked mode | Locked in every non-terminal state **except `paused`**, where the item is withheld and arena exposure is recorded |
| D14 | The 24-hour auto-submit **does not apply to mock proposals**; they need an explicit Accept |
| T4 §6.7 / §3.7 | `scored_by ∈ {self, ai-byo}`, with the `ai-byo` conditions of §6 step 5. `ScoreMock` only from `finished`. **No "clamp ≤ 2" for mocks (null band).** Mock `evaluation_completed` is not needed |
| T3 §10 | Unchanged. Runs are started by the learner only; the model has no run tool |
| T2 §4 / ADR-0028 | **Audio is not stored** in v2.x (a transient in-memory sideband copy is not storage). The B2 media bucket is parked until P3 |
| T1 | Transcript owner = **coach** (`interview_turn`, inline, C4), **deleted 30 days after scoring by default**, 12 months opt-in, per-turn delete. Audio row: "not stored". `mock_session` gains `status ∈ {open, scored, incomplete, abandoned}`, `format ∈ {classic, text, voice}`, `scored_by`, `time_multiplier` and `caveats[]`. Rubric validation moves to the snapshot (the exactly-7 check at `store.go:85-88` goes). Aggregates exclude everything except `scored` |
| Public route (ADR-0024; `gateway/public.go:56-59`) | Pending **O4**. Recommended: relabel the mock aggregates "practice mock scores (self-reviewed)", with no AI wording and no `scored_by` split |
| T0 | `t0:321` ("`scored_by=ai`, one dispute") becomes an `ai-byo` proposal accepted before `ScoreMock`. The manifest gains `mock.interviewer`, a rubric `evidence` enum and `ai: propose \| self_only \| self_only_voice` |
| practice / gateway | Expose the arena view, reveal and submit on an item since a timestamp (for `pause_exposure`) |
| PRD V6 | "1-way candidate video" = an optional local self-view (**O2**). "Assessing speech" = *content*. "Coach needs a voice" = the pinned interviewer voice, with read-aloud in P2 |
| Dependency matrix | Rows for OpenAI voice (learner's org, $0 on the node), the coach WebSocket client, Permissions-Policy and CSP, and the coach 256 Mi limit. "Object store for audio: declined." |

## 12. Alternatives considered

| Alternative | Why not |
|---|---|
| Choosing GPT-Live because it has "one brain" | Withdrawn as a reason. The director pattern works on both shells (Realtime can take appended items or function calls), so the choice rests on the S6 turn-taking, naturalness and price results |
| `gpt-realtime-2.1-mini` as the default without a bake-off | About $1 cheaper per interview and has native Patient / push-to-talk, but GPT-Live may be clearly more natural. **It is the co-candidate in S6** |
| `gpt-realtime-2.1` as the default | ≈ $4.5 typical, $24+ with no cache |
| OpenAI-hosted Responses delegation as the brain | Lower latency for delegated turns, but the policy and rubric state would live in OpenAI-side context. It is measured as an S6 arm and used only if push-only fails M2 |
| Shipping two shells in P1 | Doubles solo maintenance on 2-week-old APIs, and conversational quality can't be judged by an agent |
| Ephemeral client secret with the browser posting the SDP | Puts a token in the browser; the ephemeral-key sideband bug of 2026-09-09 to -15; GPT-Live doesn't offer it |
| A single-use start nonce against same-origin abuse | A same-origin script can fetch the nonce too. We rely on Permissions-Policy on the sibling apps, CSP, caps and a subdomain later |
| Our own WebSocket audio relay; self-hosted LiveKit / Pipecat | All audio through a 128 Mi pod; UDP or TURN on a TCP-only node |
| Hosted voice platforms; third-party speech keys | New key types and data processors, which conflicts with BYO-only |
| Gemini Live | Not a v2 BYO provider; a token in the browser; 2-minute audio+video sessions |
| Camera frames sent to the model | Not mainly a cost problem: frames sent to the brain would cost ≈ $0.1–0.3 (inferred). Rejected on **policy and value**: the only extra signal is affect (Art. 5(1)(f)), and face data would go to a third party |
| Storing audio for playback | Needs a bucket (D11), is C4, and carries an erase burden. P2 offers a local-only download instead |
| Scoring delivery (pace, fillers, fluency, tone) | Unreliable and biased ASR; legal and policy risk on the learner's own key |
| Auto-accepting mock proposals after 24 h (D14 parity) | Would publish unreviewed "AI" scores; mocks need an explicit Accept |
| A "late grace" that resumes without the paid brief | The brief costs ≈ $0.04, is cached per pause and is the owner's spec |
| Running the interview in a separate Deployment by default | Only needed if make-before-break re-attach fails (M7) |
| Refusing a mock start while an attempt is open | Contradicts D27; item selection skips live items instead |
| Platform key for interviews | Settled no (T4 §11.2, T5) |

## 13. Owner decisions — resolved 2026-09-24 (they OVERRIDE the body where they conflict)

| # | Question | Decision |
|---|----------|----------|
| **D28** | O1: spike S6 and the shell rule | **S6 is approved as specified** (≤ 1 day, owner's key, throwaway OpenAI project with a **$10 hard limit**, harness stops at $8). GPT-Live-1 is chosen only if it passes every hard gate **and** is rated ≥ 1 point more natural than `gpt-realtime-2.1-mini`; otherwise mini.<br>**S6 needs the owner present:** the owner creates the project and key and plays the candidate. It is **scheduled before the M6a design freeze** (T7), not run in the planning session.<br>Added measurements (from D29): the cost and latency of screen-context updates at a 2–3 s change cadence, and whether the "current screen" item can be replaced in place (Realtime `conversation.item.delete`/re-create; GPT-Live `thinking.append` semantics). |
| **D29** | O2: what the AI perceives | **"The AI shares the interview session like another interviewer would."**<br>• **Live two-way voice conversation** (P1).<br>• The AI **follows the on-screen answer widgets** (code editor, canvas, text fields, choices) and **drives the interview from them**. It uses **structured widget state**, not screenshots: code text, the `canvas-graph@1`/`canvas-export@1` text export, field values, selections.<br>• Updates fire **when content changes, sampled every 2–3 s** (debounced; unchanged state is never re-sent), **and on every candidate turn or question**. This replaces §5's ≥ 20 s code-diff debounce.<br>• The live model keeps **one replaceable "current screen" item**, not an accumulating history, to bound cost. The director brain gets diffs.<br>• A **snapshot timeline** (diffs, per turn and per phase) is stored with the transcript (C4, same retention and erase).<br>• The **final analysis** uses the **entire transcript plus the on-screen state at each point**.<br>• **No video goes to the AI.** The camera is not part of the AI interview; an optional local self-view may stay, off by default.<br>• **Future (not designed now):** **peer-to-peer mock interviews between learners with video.** Keep the architecture compatible: signalling reuses the SDP-relay pattern via our API. The one missing piece of infra, **TURN** (managed TURN vs self-hosted coturn with UDP ports on ufw), is a **deferred T7 decision**. Nothing is built for it in v2. |
| **D30** | O3: paused > 24 h | **`incomplete`, unscored, left out of trends.** A free partial view from checkpoints; paid partial improvement areas only on a click; early "Finish & get feedback" allowed at ≥ 75% done. |
| **D31** | O4: public profile | **Mock count only.** No scores (best/average) on the public profile, whether AI-proposed or self-scored. This amends ADR-0024's `mock{count,best,average}` public composition to `mock{count}` for all mocks. |

## 14. Risks

| Risk | Handling |
|---|---|
| GPT-Live is 2 weeks old (API launched 2026-09-10); its duration limit and mid-session quota behaviour are undocumented | S6 hard gates; mini co-candidate; the `VoiceShell` interface; P1 ships one shell |
| The dry-credit signal differs from the fixtures, or arrives late; prepaid access "may not stop immediately" and balances can go negative | Fixtures labelled by path; the probe arbitrates; grace and pause cover late detection; negative-balance copy; cached brief |
| Top-ups take longer than 5 minutes (sync delay, Indian card friction) | Expectation-setting copy; pausing is the normal, cheap outcome; auto-recharge suggested at pre-flight |
| Delegated turns feel slow | Director push keeps the brain off the critical path; M2 on push-only; delegation only for rare deep questions |
| GPT-Live interjects during think-aloud; push-to-talk emulation is weak | Hard gates M3 and M3b; otherwise mini, which has native settings |
| Routine releases cut live interviews | SIGTERM drain, make-before-break (M7 hard); fallback `coach-interview` Deployment with its own ImagePolicy; runbook check |
| Audio copies on the sideband load coach or leak through logs | Type-filtered decoder that never parses payloads; canary test; M8; 256 Mi limit |
| Same-origin script abuse of the SDP endpoint | Permissions-Policy on the sibling apps, CSP (P1 gates), mandatory $ cap, 75 voice-min/day, dedicated project advice, subdomain later |
| Browser tampering (`session.update`) or synthetic audio | M13; no data channel on Realtime; re-apply and hang up; honor tier; the score comes from coach's transcript review |
| ASR errors for Indian English and accents bias the review | Voice Communication `self_only`; null bands; learner transcript edits; twin gate per model; multi-speaker check before change |
| The live voice model reacts to emotion (Art. 5(1)(f)) | Brain-authored evaluative speech; instructions; text-based distress trigger; post-hoc scan; EU voice-off pending legal review |
| The learner steers `ai-byo` (custom model, edits, re-propose, pause lookup) | Strict `ai-byo` conditions; no auto-accept; `model` and `prompt@v` stored; public relabel (O4) |
| Firefox / Safari WebRTC interop | Chrome/Edge only at launch; text fallback; S6 legs |
| Accommodation cost (multiplier on a per-minute shell) | Cost shown at pre-flight; "hold voice while I code"; timers scale; text mode free of the premium |
| Prompt injection through speech or code comments | Delimiting; injection heuristic; `review_flag` blocks `ai-byo`; only affects the learner's own honor score |
| Anthropic-only learners get no voiced coach | Text interviewer; voice-lite in P2 on demand; say so in the settings |
| Solo QA burden (every prompt change needs a manual voice session) | Replay regression in CI; fake-media end-to-end run on the slice rail (≈ $0.3 per run) |
| UDP-blocked learner networks | S6 checks ICE-TCP / TLS candidates; the pre-flight voice check fails fast to text |
| Backups keep erased transcripts | Disclosed; D12 erase = delete plus tombstones |

## 15. Critic findings addressed

**Critic 1 (feasibility, latency, cost, solo ops):**

| # | Finding (severity) | Disposition | Where |
|---|---|---|---|
| 1 | GPT-Live's 1.24 s TTFA misattributed; delegated turns take 2.5–5 s (major) | **Fixed.** Relabelled as "benchmark, not our path". Director push keeps most turns off the round trip. S6 has 3 latency arms (push-only, client delegation, Responses delegation) and measures substantive-answer latency separately | §2 A, §3, §10 M2 |
| 2 | GPT-Live turn-taking isn't configurable (major) | **Fixed.** Modes depend on the shell (native on mini and text; emulated via the documented `session.input_audio.mute` on GPT-Live). Hard gates M3 and M3b; mini wins if emulation fails | §2, §5, §7, §10 |
| 3 | The spike tests the wrong failure; the $10 cap isn't hard (major) | **Fixed.** An enforced project hard limit plus the harness stopping at $8; a separate $1-limit project for the quota test; fixtures labelled spend-limit only; the probe arbitrates for prepaid; passive purchase→probe timing; optional S6b (O1b); records whether the active session is killed | §10, §13 O1 |
| 4 | Top-ups rarely finish within 5 minutes (major) | **Partly adopted.** The owner's 5-minute countdown stays (his spec), but the copy sets expectations, warns about negative-balance deduction ("add ≥ remaining + $1"), suggests auto-recharge at pre-flight and adds "+ tax". A late grace without the brief is not adopted: the brief costs ≈ $0.04, is cached per pause and is the owner's spec | §4, §8 |
| 5 | No model-access class; the probe can flap (major) | **Fixed.** `ErrModelAccess`; a real 20–30 s pre-flight voice check; the 60-second-stable rule; ≤ 2 automatic re-primes per grace. The "15 s minimum billing" claim was **not adopted**: the model page says duration is "not rounded up" | §4, §8 |
| 6 | Coach deploys cut every live interview (major) | **Fixed.** M7 is hard and decides the deploy shape; SIGTERM drain; `terminationGracePeriodSeconds: 60`; `maxSurge: 1` make-before-break; fallback `coach-interview` Deployment with its own ImagePolicy; runbook check | §3, §10, §11 |
| 7 | "One brain" overstated (major) | **Fixed.** Removed as a reason. Director push works on both shells; hints come from the `give_hint` ledger plus the transcript; bluff metric M12 | §1, §2, §5, §12 |
| 8 | S6 runs after P0 is built; P0 is overloaded (major) | **Fixed.** S6 runs before the M6a design freeze. P0 slimmed: the countdown UI, AI checkpoints and transcript editing move to P1. Re-propose stays for D14 parity (it is the same call) | §9 |
| 9 | Solo maintenance of 4 voice paths (major) | **Fixed.** P1 ships exactly one shell; 2.1 is catalog-only if it rides the same adapter; voice-lite only on demand; replay regression, fake-media end-to-end and slice rail; optional canary | §2, §9 |
| 10 | No browser matrix; Firefox breaks (major) | **Fixed.** Chrome/Edge only at launch; Firefox → text; Safari depends on its S6 leg | §2, §7, §10 |
| 11 | Multiplier cost and caps hidden (minor) | **Fixed.** Cost rows, pre-flight disclosure, "hold voice", timers scale, `usage_ratio` rollover at 0.8 | §4, §7, §8 |
| 12 | ±10% too tight; tax and forex (minor) | **Fixed.** Ranges widened; "+ tax (India 18% GST)"; the brain gets diffs, not the full file | §5, §8 |
| 13 | Idle in the Code phase is billed (minor) | **Fixed.** The idle rule applies in any phase and hangs up | §4 |
| 14 | SDP broker timeout leaves orphan sessions (minor) | **Fixed.** 20 s route timeout; unconfirmed sessions hung up after 15 s | §3, §4 |
| 15 | Voice-lite "no audio leaves the device" (minor) | **Fixed.** On-device mode required or the cloud fallback disclosed; the voice differs | §2 D |
| V | Critic 1's list of claims verified wrong (latency, cap, turn-taking, quota path, "second implementation", voice-lite, "+$7 frames", Firefox) | All corrected as above. The frames claim is corrected in §12 and O2c | — |

**Critic 2 (privacy, ethics, security, integrity):**

| # | Finding (severity) | Disposition | Where |
|---|---|---|---|
| 1 | The learner can steer `ai-byo`, and it is shown publicly (major) | **Fixed.** Strict `ai-byo` conditions (catalog model, no edits or client turns, no `pause_exposure`, explicit Accept); no auto-accept; `model` and `prompt@v` stored; custom ids get no proposal; public relabel (O4) | §6, §11, §13 |
| 2 | Sideband audio passes through coach, so "no media" was wrong (major) | **Fixed.** §1, §3 and §7 corrected; the type-filtered decoder drops audio unparsed; canary test; consent and notice disclose it; inventory row; S6 looks for a filter | §1, §3, §7, §10 |
| 3 | Art. 5(1)(f) exposure through the live voice model (major) | **Fixed.** Brain-authored evaluative speech; instructions forbid tone reactions; text-based distress trigger; post-hoc scan; EU voice-off until a legal review | §6, §7 |
| 4 | SDP endpoint abuse by a same-origin script; Realtime tampering (major) | **Mostly fixed.** CSP, sibling camera/mic deny and Permissions-Policy are P1 gates; 75 voice-min/day; mandatory $ cap; Realtime without a data channel, or re-apply and hang up; onboarding key hygiene. **Nonce not adopted**, because a same-origin script can fetch it; a subdomain is a T7 recommendation | §3, §11, §12 |
| 5 | Fairness rests on one speaker (major) | **Fixed.** Voice Communication `self_only` by default; timeline evidence advisory; twin gate per catalog model; multi-speaker check (≥ 10 speakers) before changing; custom ids get no proposal | §6 |
| 6 | Pause lets the learner look up the item in the arena (minor) | **Fixed.** Arena view, reveal and submit recorded → `pause_exposure` caveat, no `ai-byo` | §4, §6 |
| 7 | Hint ladder can leak through the context (minor) | **Fixed.** `give_hint(n)` records first; unreleased hints never enter a context; instructions treated as learner-visible (S4) | §5, §7 |
| 8 | Transcript sent to text APIs; `store:false` (minor) | **Fixed.** M6a acceptance test; inventory and consent rows | §7, §9 |
| 9 | 12-month retention; backups; segment log not erased (minor) | **Fixed.** 30 days after scoring by default, 12 months opt-in; backup lag disclosed; segment log and consent rows erased; per-turn delete | §7 |
| 10 | Accessibility gaps (screen-reader echo, multiplier cost, fixed timers, camera grant) (minor) | **Fixed.** Screen-reader preset; cost shown; timers scale; camera off by default | §7 |
| 11 | Spike cap; fixtures hold owner data (minor) | **Fixed.** Enforced hard limit plus harness stop; scrubbing before any commit; logs deleted with the project | §10 |
| 12 | Key held decrypted for the whole session (minor) | **Fixed.** ADR-0007 amendment: a `[]byte` per live segment, dropped on interrupt or pause | §3, §11 |
| V | Critic 2's list of claims verified wrong ("no media", Art. 5 overstated, "every proposal reviewed" vs auto-accept, $10 cap, `withhold()` doesn't cover the arena, GPT-Live browser `session.update`) | All corrected. GPT-Live's WebRTC guide documents only `session.close` as browser-sent; S6 M13 verifies | — |

**Strengths kept from both critiques:**
- no media on the node's network path;
- SDP brokering, and the unified new-segment mechanism;
- text P0 as a permanent mode;
- the classifier by `error.code`, with the probe as arbiter and `ErrQuota` never disabling a key;
- deliberate hang-ups;
- verified prices;
- free deterministic checkpoints, and no key spend without a click;
- content-only assessment with null bands;
- bounded SSE;
- a throwaway, owner-approved spike.

---

### Sources (all seen 2026-09-24)

**Re-verified for this final version:**
- GPT-Live-1 model page ($0.05/min billed per second, "not rounded up"; audio and text only; free tier unsupported; Tier 1 limit of 25 concurrent sessions; `v1/live/sessions` only): https://developers.openai.com/api/docs/models/gpt-live-1
- GPT-Live delegation (`session.delegation.created` "does not contain the user's utterance"; appends limited to 500 tokens; "Live speech and delegated work continue independently"): https://developers.openai.com/api/docs/guides/live-delegation
- GPT-Live session management (`expired`; 128k context; replacement engine above 90% keeping 8,192 tokens; seeding ≤ 128 / 8,192; `session.instructions.append`; `session.input_audio.mute`; close idle sessions; `session.usage.updated` with `usage_ratio`): https://developers.openai.com/api/docs/guides/live-conversations
- GPT-Live WebRTC (server-side `POST /v1/live/sessions` only; `oai-events` data channel; only `session.close` documented as browser-sent): https://developers.openai.com/api/docs/guides/voice-webrtc?api=live
- Sideband / server controls ("also receives copies of subsequent input and output audio", base64 PCM16LE 24 kHz; no filter documented; standard-key auth; frontend model and audio configuration fixed at startup): https://developers.openai.com/api/docs/guides/realtime-server-controls
- Spend limits (spend alerts don't cap; `project_/organization_spend_limit_exceeded`; "Enforcement is not instantaneous"; `credit_balance_exhausted`): https://developers.openai.com/api/docs/guides/spend-limits
- OpenAI prepaid billing (access may not stop immediately; negative balance deducted from the next purchase): https://help.openai.com/en/articles/8264644-setting-up-and-managing-prepaid-api-billing
- LiveKit GPT-Live plugin ("The model controls turn detection and barge-in"; voice and mode fixed after start): https://docs.livekit.io/agents/models/realtime/plugins/gpt-live/
- Artificial Analysis speech-to-speech (GPT-Live-1 Sol low 1.24 s; realtime-2.1 0.97 / 1.21 s; mini 0.85 / 4.28 s; Gemini 3.8 Live 1.18 s; the backend for "Sol" is undisclosed): https://artificialanalysis.ai/speech-to-speech
- Firefox ↔ OpenAI Realtime WebRTC disconnects (open since 2026-05-21): https://github.com/openai/openai-agents-js/issues/1353
- Browser `session.update` over the Realtime data channel (2025-05-05): https://community.openai.com/t/realtime-api-webrtc-how-to-avoid-end-users-updating-instructions-of-the-model/1251817
- FPF on the Commission's Art. 5(1)(f) guidelines (voice in scope; education read broadly; hiring counts as workplace; narrow safety exception; text out of scope): https://fpf.org/blog/red-lines-under-eu-ai-act-unpacking-the-prohibition-of-emotion-recognition-in-the-workplace-and-education-institutions/
- OpenAI GST in India (18% without a GSTIN): https://community.openai.com/t/gst-no-longer-being-charged-after-adding-gst-number-in-india/998331

**Carried from the draft (verified there on 2026-09-24):**
- Realtime WebRTC unified interface: https://developers.openai.com/api/docs/guides/realtime-webrtc
- Realtime conversations (60-minute cap): https://developers.openai.com/api/docs/guides/realtime-conversations
- Realtime costs: https://developers.openai.com/api/docs/guides/realtime-costs
- `gpt-realtime-2.1`: https://developers.openai.com/api/docs/models/gpt-realtime-2.1
- `gpt-realtime-2.1-mini`: https://developers.openai.com/api/docs/models/gpt-realtime-2.1-mini
- OpenAI pricing: https://developers.openai.com/api/docs/pricing
- OpenAI data controls: https://developers.openai.com/api/docs/guides/your-data
- OpenAI text-to-speech (voices; AI-voice disclosure): https://developers.openai.com/api/docs/guides/text-to-speech
- Ephemeral-key sideband bug, fixed 2026-09-15: https://community.openai.com/t/ephemeral-keys-trigger-realtime-sideband-404-errors/1396002
- Anthropic models overview: https://platform.claude.com/docs/en/about-claude/models/overview
- EU AI Act Art. 5: https://artificialintelligenceact.eu/article/5/
- Digital Omnibus: https://www.gibsondunn.com/eu-ai-act-omnibus-agreement-postponed-high-risk-deadlines-and-other-key-changes/
- Gemini Live (from the slice): https://ai.google.dev/gemini-api/docs/live-session

**Repo facts checked:**
- `internal/gateway/public.go:56-59` (public mock count/best/average)
- `internal/gateway/coach.go:43` (10 s `httpc`) and `:249` (WriteTimeout cleared for streams)
- `docs/adr/0007-ai-coach-byo-key-and-secrets.md:27` (key zeroed after each call)
- `docs/v2/research/t5-platform-ai.md:42,497-510,532,542-547,609`
- `docs/v2/feasibility.md:14-41,242,429-431`
- `../infra/apps/image-automation.yaml:11,22-23` (semver `>=1.0.0`, 1-minute interval)
- `../infra/charts/project/values.yaml:32-35` (`replicaCount: 1`, RollingUpdate)

**Cost scripts:**
- `scratchpad/t6synth/cost.py`
- `scratchpad/t6synth/cost2.py`