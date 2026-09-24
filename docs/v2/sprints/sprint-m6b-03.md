# Sprint m6b-03 — Voice UI: pre-flight/notices (AB29), live HUD★ (AB30), consent, EU gate, self-view → v2.0.x patch (M6b dark)

> **Milestone:** M6b — voice, one shell (**the only M6b UI sprint**; it ships M6b **dark** to the T-3 cohort) · **Track:** product · **Order:** 84
> **Prereqs:** [m6b-02](sprint-m6b-02.md) (lease, re-attach, drain, cost + $ cap, PTT modes, rollover, self-edit), which follows [m6b-01](sprint-m6b-01.md) (`VoiceShell`, SDP broker, the coach-side voice gates, sideband, captions) · [ds-m6b-01](sprint-ds-m6b-01.md) (AB29–AB30 frozen) · [mi-13](sprint-mi-13.md) (MI-16 live: Permissions-Policy, CSP, the 20 s SDP route) · [m6a-06](sprint-m6a-06.md) (the M6a screens and their `v2.0.x` patch), built on [m6a-01](sprint-m6a-01.md) (FSM, client lease, `qa_slice`, cohort gate) and [m6a-05](sprint-m6a-05.md) (routes, setup, text HUD, `pagehide` beacon)
> **Unblocks:** [m6b-04](sprint-m6b-04.md): the fake-media run needs this patch on prod, plus the media-audit verb, m6a-01's owner-only `qa_slice` reachable from the setup UI, and the runbook from task 6
> **Release action:** **tag a `v2.0.x` patch** (M6b dark, cohort only): it carries m6b-01, m6b-02 and this sprint, and it is the next free patch ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)). **Interviewer defaults unchanged.** No infra PR is expected. The one exception: if S6 M7 failed, m6b-02's `coach-interview` Deployment gets its own ImagePolicy bump after the tag, as **its own infra PR** (task 8), never folded into the tag.
> **Artboards:** AB29 voice pre-flight, browser and EU notices (`design-system/screens/v2/AB29-*.html`) · **AB30★** voice live HUD (`design-system/screens/v2/AB30-*.html`) · the voice variant of M6a's AB26 grace modal, as AB30's behaviour notes draw it
> **Calendar:** Q1 2027, after [m6b-02](sprint-m6b-02.md). Agent work, plus **5 minutes of the owner right after the tag** (task 7): he opens the voice pre-flight on prod up to the free mic check. The agent never signs in as anyone.
> **Execute with:** [`../prompts/prompt-m6b-03.md`](../prompts/prompt-m6b-03.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Voice gates, pre-flight, notices and voice consent (AB29), in the FSM's order; m6b-01's server gates verified, the `mic` interrupt reason added if missing | X | ⬜ |
| 2 | Browser voice client `web/src/lib/voice/` (mic, PeerConnection, m6b-01's segment contract, PTT, teardown) | X | ⬜ |
| 3 | Live HUD (AB30★): rail, captions, ask/rephrase, mic and camera state, editor + Run, Hold/Talk, PTT and its mid-call toggle, cap notices, countdown grace, distress card, failure states, transcript check, a11y | X | ⬜ |
| 4 | Optional local self-view (never sent) | X | ⬜ |
| 5 | Tests, a compose walk of the AB29/AB30 frames, screenshots, bundle check | X | ⬜ |
| 6 | Prepare m6b-04: `coach admin interviews media-audit`, m6a-01's `qa_slice` in the owner's setup UI, and the runbook `docs/runbooks/voice-fake-media-e2e.md` | X | ⬜ |
| 7 | Tag the `v2.0.x` patch (M6b dark) and verify on prod, incl. the owner's 5-minute pre-flight look | X + H + O | ⬜ |
| 8 | **Only if S6 M7 failed:** the `coach-interview` ImagePolicy bump (its own PR in `../infra`) | I | ⬜ |
| 9 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line to match. Mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, the M6b milestone row, the milestone → tag → floor → snapshot row, and the artboard rows AB29–AB30.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **AB29–AB30 frozen:** the owner merged [ds-m6b-01](sprint-ds-m6b-01.md)'s PR, so the boards are on `main` under `design-system/screens/v2/`. In status.md, rows AB29 and AB30 read "frozen (PR #, date)"; if [m6b-01](sprint-m6b-01.md) didn't set them, set them here.
- [ ] **[m6b-02](sprint-m6b-02.md) merged, and [m6b-01](sprint-m6b-01.md) before it.** Read their PRs (and m6b-01's hand-off line in status.md) and write down what this UI consumes:
  - the interview DTO's `voice` block: `available`, `reason ∈ {region, no_voice_key, model_access, disabled}` (m6b-01) plus `capacity` and `daily_limit` (m6b-02), `shell`, `voice`, `data_channel`, `mode`, `minutes_left_today` and `estimate{typical_micros, plan_for_micros, keep_available_micros, per_multiplier}`;
  - m6a-01's client lease: `POST …/attach {client_id}` and `POST …/heartbeat {client_id, visible, last_input_at}` (≤ 1 / 10 s);
  - segment creation through mi-13's `POST /api/interviews/{id}/segments` with m6b-01's body `{sdp, purpose, client_id}` and its 201 `{segment_n, sdp, shell, confirm_by}`, and the confirmation `POST …/segments/{n}/confirm` that beats coach's 15 s orphan reaper;
  - the `ask` (m6b-01), `ptt`, `hold`, `voice-mode`, `PATCH …/turns/{seq}` and `turns/mirror` (m6b-02) routes, and M6a's `interrupt`, `probe`, `pause`, `resume`, `finish` and `abandon`;
  - the `caption` and `segment` event kinds on the bounded SSE, and m6b-02's `notice{kind: "idle_check" | "probe"}`;
  - the typed errors: m6b-01's broker table, m6b-02's 409 `cap_reached`, 503 `draining` and the L19 429s (`voice_capacity`: ≤ 3 live voice platform-wide; `voice_daily_limit`: ≤ 75 voice-minutes per account per day).
- [ ] **M6a UI live** ([m6a-06](sprint-m6a-06.md)'s `v2.0.x` patch): Mock-v2, setup/consent/pre-flight, the text HUD, grace/paused/resume, debrief/proposal and accessibility. The interviewer's T-3 cohort gate (m6a-01's `interviewAudience = "cohort"`) is in place.
- [ ] **MI-16 live** ([mi-13](sprint-mi-13.md)'s `v2.0.x` patch):
  - `Permissions-Policy: camera=(self), microphone=(self)` on the SPA;
  - the CSP stays `connect-src 'self'`;
  - `POST /api/interviews/{id}/segments` has a 20 s budget, a 20 KiB cap and no retry.
- [ ] **`account.region` reaches coach and the SPA:** coach reads it through identity's internal account read (m6b-01's check 5); the SPA reads `/api/me` → `acceptance.region` (ISO 3166-1 alpha-2, [l-05](sprint-l-05.md), live in `v1.17.0`). Every account, the owner's included, has passed the acceptance step.
- [ ] **The S6 results note is at hand** ([spk-04](sprint-spk-04.md)) and [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) is Accepted ([ds-m6a-01](sprint-ds-m6a-01.md)). The note gives the browser gate list, the chosen shell (GPT-Live-1 or `gpt-realtime-2.1-mini`), whether that shell needs the `oai-events` data channel, and whether push-to-talk is native or emulated.
- [ ] **Only if S6 M7 failed:** m6b-02's `coach-interview` Deployment is live, and the gateway routes `/api/interviews/*` to it.
- [ ] **Release line:** `.release-line` = `2`, and every `xlearn-*` ImagePolicy is `>=1.0.0 <3.0.0` ([ga-02](sprint-ga-02.md)).
- [ ] **Parallel sessions:** no open peer PR touches `web/src/router.tsx`, `web/src/screens/mock/**`, `web/src/lib/interview/**`, `web/src/lib/api.ts`, `web/package.json`, the gateway's interview proxy, `internal/coach/interview` or coach's admin CLI. Check with `gh pr list`, `git worktree list` and ListAgents.

## Goal

Ship the **voice surfaces** exactly as the frozen boards draw them:
- **AB29:** the pre-flight, the browser and EU/EEA notices, and voice consent;
- **AB30★:** the voice live HUD, through the post-call transcript check;
- the optional local self-view.

The **browser gate** is the SPA's (an interop gate). The **EU/EEA gate** is coach's: [m6b-01](sprint-m6b-01.md) reads `account.region` ([ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) row 14) and refuses voice; the SPA only renders the server's answer. Every refusal offers text mode. The browser talks WebRTC **only to OpenAI**; coach brokers the SDP, so no credential or media reaches the page's own origin. The camera never enters the PeerConnection ([ADR-0032 §2, §3, §6](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture)).

Then cut a **`v2.0.x` patch** that carries M6b **dark**: m6b-01 + m6b-02 + this sprint, behind the interviewer's existing T-3 cohort gate, with the interviewer defaults unchanged. The patch also carries the three things [m6b-04](sprint-m6b-04.md)'s fake-media run needs on prod: a read-only media-audit verb, m6a-01's owner-only `qa_slice` reachable from the setup UI, and a runbook.

## Scope

**In**
- **AB29 (X):** the gates (cohort → key → browser → region → caps), then the pre-flight **in the order AB29 draws it and m6a-01's FSM allows** (task 1): mode, voice consent, mic permission and check, the voice estimate with the mandatory, editable $ cap, `start`, the real 20–30 s voice check, Ready. Also the compact reaffirmation on resume.
- **Server gates (X):** verify [m6b-01](sprint-m6b-01.md)'s coach-side gates (EU/EEA region, voice consent, the OpenAI `voice_shell` key); add only what is missing. Add the `mic` interrupt reason to m6a-01's FSM table if nothing added it yet.
- **Voice client (X):** `web/src/lib/voice/`, loaded as a lazy chunk, on m6b-01's segment contract and m6a-01's client lease.
- **AB30★ (X):** the HUD (F1–F18), including the voice variant of AB26's grace modal (the countdown), the mid-call push-to-talk toggle, ask/rephrase, the distress card and the post-call "Check your transcript" screen.
- **Self-view (X):** optional, local only.
- **Preparing m6b-04 (X):**
  - `coach admin interviews media-audit <id>`;
  - m6a-01's owner-only `qa_slice` reachable from the owner's setup UI (a toggle, only if m6a-05/06 didn't add one);
  - the runbook `docs/runbooks/voice-fake-media-e2e.md`.
- **The patch (X + H + O):** tag the `v2.0.x` patch and verify it on prod; the owner's 5-minute pre-flight look.
- **Only if S6 M7 failed (I):** the `coach-interview` ImagePolicy bump, its own PR.

**Out**
- **To [m6b-04](sprint-m6b-04.md):** the fake-media end-to-end run on prod against this patch, the interviewer GA flip (T-1) and `v2.1.0`.
- **Already done in [m6b-01](sprint-m6b-01.md) / [m6b-02](sprint-m6b-02.md):** backend voice behaviour (the broker and its preconditions, sideband, audio-drop decoder, director push, lease, re-attach, drain, cost, caps, rollover, self-edit). This sprint only verifies the server gates, and adds any that are missing.
- **Already done in [mi-13](sprint-mi-13.md):** the gateway headers, the SDP route and coach sizing.
- **To M6c** ([m6c-01](sprint-m6c-01.md), [m6c-02](sprint-m6c-02.md), [m6c-03](sprint-m6c-03.md); outline, v2.2):
  - the "show your work" photo, chat read-aloud and a local recording download;
  - the multi-speaker fairness check and AI-proposed voice Communication;
  - history-calibrated estimates;
  - Safari, voice-lite and a weekly canary.
- **Owner-approved, never merged by the agent:** the privacy-notice PR (the `/xlearn/privacy` page and its notice version). It is opened here only if task 1 finds a gap, and must be live before m6b-04's flip.
- **Dropped by D34:** interview counters, alerts and opscheck.

## Tasks

### 1 · Voice gates, pre-flight, notices and voice consent (AB29) [X]

**Where.** Next to the M6a interview screens that [m6a-05](sprint-m6a-05.md)/[m6a-06](sprint-m6a-06.md) built: `web/src/screens/mock/` (`InterviewSetup.tsx`, `LiveHud.tsx`, `hud/*`) and `web/src/lib/interview/api.ts`. For example `web/src/screens/mock/voice/VoicePreflight.tsx` and `VoiceNotice.tsx`, reached from m6a-05's `mock/setup` and `mock/live/:id` routes (t6 §3 names `/xlearn/<course>/mock/live`). Voice is a **format** of the same interview, not a new route family. If m6a's merged layout differs, follow it.

**Gate order.** Evaluate every gate **before any mic prompt and before any spend**. Each failure has its own AB29 frame, and every one offers **"Use text mode"**, which keeps the setup already entered.

| # | Gate | Source | Fail frame (AB29) | Server enforcement |
|---|---|---|---|---|
| 1 | Cohort (T-3) | m6a-01's `interviewAudience` | none in AB29: outside the cohort, `/api/interviews/*` is the unknown-route 404 and `MockRoute` shows v1 (m6a-05). There is no `cohort` reason | the gateway's existing gate (m6a-01; mi-13 on `/segments`) |
| 2 | An OpenAI `interview` default key with the `voice_shell` catalog capability ([m1-10](sprint-m1-10.md)'s catalog) | `voice.reason = no_voice_key` | F9 "Voice needs an OpenAI key as your interview key", [Open Settings] | coach 409 `voice_key_required` (m6b-01 check 6) |
| 3 | Browser | the S6 list, default **desktop Chrome and Edge**, via `navigator.userAgentData.brands` ("Google Chrome", "Microsoft Edge"; `mobile: false`). The UA string is the fallback. Also `isSecureContext`, `RTCPeerConnection`, `mediaDevices.getUserMedia`, and `document.featurePolicy?.allowsFeature('microphone')` | F7 "Voice interviews run in Chrome or Edge for now". Firefox gets the reason (its WebRTC sessions drop after 1–2 turns, [t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options)). If `allowsFeature` is false, the frame says the page's policy blocks the mic, which means mi-13 regressed | none: an interop gate, not a security gate |
| 4 | **EU/EEA region** | `voice.reason = region`, computed by coach from `account.region` (a null region fails closed with the same reason) | F8, the EU/EEA notice, with the region name via `Intl.DisplayNames`; when `/api/me`'s `acceptance.region` is null, F8's no-region variant ("Confirm your region") | coach **403 `voice_unavailable_region`** on a `mode=voice` create and on every `/segments` purpose (m6b-01 check 5), passed through the gateway proxy |
| 5 | L19 caps | `voice.reason ∈ {capacity, daily_limit}` and `minutes_left_today`; the 429s at mint | F10 busy (`429 voice_capacity`, ≤ 3 live platform-wide); F10 daily limit (`429 voice_daily_limit`, 75 min/day, `Retry-After`, reset shown in the account timezone) | coach (m6b-02) |
| 6 | Model access | `voice.reason = model_access` (the last recorded `ErrModelAccess` for the key) | F5a "Voice needs a paid OpenAI account (Tier 1 or higher)", no retry | coach 403 `voice_model_access` at mint |
| 7 | Voice switched off | `voice.reason = disabled` | the frame AB29's behaviour notes assign (record the mapping) | coach |

- **The SPA carries no EU list and infers no eligibility.** It renders `voice.available` / `voice.reason` from the enum above and `data_channel`; the browser check is its only client-side test. Choosing F8's no-region variant from `/api/me` is a display choice, not a decision.
- **Frames and Vitest cases are keyed to the server's codes** (`region`, `no_voice_key`, `model_access`, `capacity`, `daily_limit`, `disabled`), never to invented ones.

**Server gates.** Check each in the m6b-01/02 code; add only what's missing, with a Go test:
- a) **EU/EEA** (m6b-01 check 5, in coach): 403 `voice_unavailable_region` on `POST /interviews` with `mode=voice` and on `/segments` (the pre-flight voice check included), for any code in the EEA set (EU-27 plus `IS`, `LI`, `NO`; Greece is `GR`) and for NULL. Confirm its table tests cover all 30 codes, a non-member (`IN`, `GB`, `CH`) and NULL; add missing cases to m6b-01's test file. **No gateway 403 and no second EEA list.** Only if m6b-01 left no EEA set at all, add it where check 5 reads it.
- b) **Voice consent** (m6b-01 check 4): a voice segment without the current voice consent kinds → 409 `consent_required {kinds}`.
- c) **Key** (m6b-01 check 6): no enabled OpenAI `interview` key with a catalog `voice_shell` → 409 `voice_key_required`.
- d) **The `mic` interrupt reason.** The client posts `interrupt {reason: "mic"}` on mic loss ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes): `interrupted(mic)`). m6a-01's reasons are `quota`, `rate`, `provider`, `network`, `idle`, `tab_closed`, `model_access`, and m6b-02 adds `voice_limit`. If `mic` is still missing, add it to m6a-01's FSM data table (`live —interrupt(mic)→ interrupted`, with grace like `network`), plus the table test and a handler test. If `state_reason` carries a CHECK, widen it in an expand migration (next free coach version; `sqlc diff` clean).

If a DTO or route changes, update `openapi.yaml` + `api.md` and run the drift test. `sqlc diff` stays clean.

**Pre-flight frames**, in AB29's order where m6a-01's FSM allows it. The FSM forces one move: `setup —start→ preflight` needs the session consents at the current version and `cap_micros > 0` ([m6a-01](sprint-m6a-01.md) task 2), and the voice check is a `purpose: "preflight"` segment that m6b-01 accepts only in `preflight`. So the estimate and cap (F6) come **before** the voice check (F5), and no `/segments` POST happens until consent is recorded and `start` has returned `preflight`.
1. **F1 Mode** (the AB24 setup row with voice; entered from AB13 F1b): [Voice] [Voice · push-to-talk] [Text], and [Standard] [Patient] under Voice. This creates the interview (`POST /api/interviews` with `mode: "voice"` and m6b-02's `voice_mode`); m6b-01's check 5 refuses an EU/EEA or null-region create with 403 → F8.
2. **F2 Voice consent:** unticked boxes, each required, in AB29's final copy. They carry AB24 F3's three boxes verbatim plus the voice lines of [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility):
   - the mic is streamed to OpenAI on the learner's own key, and the voice passes through xLearn's server **in memory only**, never written to disk or logs;
   - "The interviewer is an AI. It can't see you.";
   - "I'll take this somewhere private: nearby voices are sent and transcribed too."

   The retention choice and the text-mode lines from M6a stay (the transcript goes to the text API, retention is 30 days after scoring, and assessment uses content only). **[Continue] stays disabled until every box is ticked**; it records the M6a kinds and m6b-01's voice kinds at the current version through `PUT /api/interviews/{id}/consent`.
3. **F3 Mic permission:** explain before asking (one line on why the mic is needed), then `getUserMedia({audio: …})` on a click; denied and no-device states with per-browser steps and "Use text mode". The camera is never requested here. This is free and local, so it comes **before** `start`: a denied mic doesn't use one of the day's 2 starts.
4. **F4 Mic check:** `enumerateDevices()`, a Web Audio `AnalyserNode` on an `AudioContext` created in the same gesture, and the "I'm wearing headphones" toggle (barge-in, [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)).
5. **F6 Estimate and cap** (drawn after F5 on the board; moved here for the FSM): "Typical $X · plan for $Y · + tax where applicable (India: 18% GST) · billed by OpenAI to your key", plus the negative-balance line with the "keep at least $Z available" figure, all from the `voice.estimate` block (m6b-02's cost model × the time multiplier), and "Voice today: N of 75 minutes left". The **mandatory $ cap** defaults to plan-for × 1.5, stays learner-editable, and a cap of 0 or empty is refused. The time multiplier (AB28) and its added cost sit here too.
6. **Start the pre-flight:** `POST …/start` (`setup → preflight`; m6a-01's guards; it counts toward the ≤ 2 starts a day, and m6a-02's key probe runs inside it). A `preflight_fail{reason}` returns to setup with the matching frame.
7. **F5 Real voice check** (and F5a–F5d): labelled "uses your key (≈ $0.02)"; a `purpose: "preflight"` segment (≤ 30 s, m6b-01) opened on a click. The interviewer says hello and asks for a sentence; captions appear. Pass or fail, the segment is closed. F5b/F5d's [Try again] is a new click, not a retry.
8. **F11 Screen-reader preset**, where AB29 places it.
9. **F12 Ready → [Start interview]**, as a gesture: it unlocks autoplay and mic capture. It leaves `preflight` through the call m6b-01 merged (`preflight_ok → connecting`), then the client posts the `purpose: "live"` offer. If m6b-01 left a voice interview no way out of `preflight`, that is a backend gap: add the data-table row with a test and record it.
10. **F13 Resume by voice:** the compact reaffirmation (one line, one tick) inside AB26's resume modal, before **Resume interview**.
11. **F14 < 1024 px / 390 px:** one column with the step counter; the notices (F7–F10) with the text-mode button first.

**Privacy-notice check** ([t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream), Operations):
- Read the live notice ([l-05](sprint-l-05.md)'s `privacy-notice@N`, the `/xlearn/privacy` page). It must disclose:
  - sideband audio passing through coach's memory;
  - transcript and code processing by the learner's text API;
  - the backup-lag window.
- If it does, record "covered (`privacy-notice@N`)".
- If it doesn't, open an **owner-approval privacy-notice PR**. It is a code PR in l-05's shape: `web/src/content/privacy-notice.md` with its "What changed" block, `web/src/lib/notice.ts`'s `NOTICE_VERSION` and identity's `NoticeVersion` bumped together (the parity test). Title it for owner approval and **don't merge it**. The bump makes every account, the owner included, re-accept (AB19 F12, renotice). It goes live with the next patch tag after the owner merges it: this sprint's tag if he merges before it, otherwise a patch [m6b-04](sprint-m6b-04.md) tags (its task 1) before the fake-media run. Record the PR in status.md's open items. This sprint's tag doesn't wait for it.

### 2 · Browser voice client `web/src/lib/voice/` [X]

A **lazy chunk** (a dynamic `import()` from the voice screens), so the main bundle doesn't grow. The `vite:preloadError` reload handler from M3 covers a deploy that lands while a tab is open. The client builds on m6a-05's `web/src/lib/interview/api.ts` (the typed errors, the `Idempotency-Key` helper, `attach`, `heartbeat`, `interruptBeacon`) and invents no route.

| Concern | Rule |
|---|---|
| Client lease | On mount, `attach {client_id}` with m6a-05's per-tab UUID, then m6a-01's `heartbeat {client_id, visible, last_input_at}` at most once per 10 s. That is the **client** lease (which tab). m6b-02's **sideband** lease (which coach pod) is server-only: the SPA never sees it, and code and docs never call either one just "the lease". 409 `lease_lost` → AB30 F15 |
| Mic | `getUserMedia({audio: {echoCancellation: true, noiseSuppression: true, autoGainControl: true}})`, only inside a click (Test / Start / Resume / Talk). Keep one mic stream while a segment is open |
| PeerConnection | `new RTCPeerConnection()`, with ICE servers only if m6b-01 says so; ICE runs browser↔OpenAI. The mic track is added as `sendrecv` audio. **Never a video track**: self-view (task 4) is a separate stream |
| Data channel | Opened as `oai-events` **only** when `voice.data_channel` says the shell needs it (GPT-Live). The client sends nothing on it except what m6b-01 allows (`session.close` at hang-up); **never `session.update`** ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path), M13) |
| Offer | Gather ICE the way the S6 harness page did (record the choice). `POST /api/interviews/{id}/segments` with **`{sdp, purpose, client_id}`**. `purpose: "preflight"` is AB29's voice check (FSM `preflight`). `purpose: "live"` covers Start (`connecting`), a reconnect from `interrupted` (non-quota), Talk from `held`, a free re-prime on `segment{event:"reprime"}` and a rollover on `segment{event:"rollover"}` (same `client_id`) (m6b-01 check 3, m6b-02). Sent **once**: **no retry**, because a retry opens a second paid session on the learner's key. The single exception is m6b-02's **503 `draining`** (`Retry-After: 2`): coach refused before any provider call, so the client retries exactly once after the delay |
| Timeout | `apiFetch`'s 15 s `REQUEST_TIMEOUT_MS` is **shorter than the 20 s route budget**. Pass the SDP call its own ≈ 22 s `AbortSignal`. On 504 `sdp_timeout`, any other 5xx or an abort, offer text mode |
| Response and errors | 201 `{segment_n, sdp, shell, confirm_by}`. Typed errors → frames: 409 `lease_lost` (F15), 409 `illegal_transition`, 409 `consent_required` (back to F2), 403 `voice_unavailable_region` (F8), 409 `voice_key_required` (F9), 403 `voice_model_access` (F5a), 409 `provider_quota` (F5c, or the grace when live), 401 `provider_auth`, 422 `sdp_invalid` (a client bug: text mode, report it), 429 `voice_capacity` / `voice_daily_limit` (F10), 409 `cap_reached`, 502 `provider_unavailable` / 504 `sdp_timeout` (F5b / F5d) |
| Answer and confirm | `setRemoteDescription({type: 'answer', sdp})`. The remote track plays in an `<audio autoplay>` element created in the gesture. When `connectionState === 'connected'`, `POST /api/interviews/{id}/segments/{segment_n}/confirm` before `confirm_by` (m6b-01; the first media event on the sideband may confirm first, and either wins). Otherwise coach's reaper hangs up at 15 s. The `segment` events (`opened`, `confirmed`, `closed`, `reprime`, `rollover`) drive the pre-flight and the HUD |
| Captions | Taken from the bounded SSE's `caption` events, then the final `turn` event: **one path for both shells** (m6b-01). The data channel isn't a caption source. The client doesn't use m6b-02's `turns/mirror` (record the decision) |
| Health | `connectionState` / `iceConnectionState` `disconnected` for > 5 s, or the `offline` event → `POST …/interrupt {reason: "network"}`. `failed` → the server's transient path. m6a-01's heartbeat (row 1) keeps the client lease; a missing heartbeat for 30 s becomes the sweeper's `interrupt(network)` |
| Mic loss | Track `ended`, or a `permissions.query({name: 'microphone'})` change → "Reconnect mic" after 10 s → `interrupt {reason: "mic"}` after 30 s ([t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes); the reason exists after task 1 d). Show AB30 F7's copy as frozen |
| PTT | Holding the key or the on-screen button (pointer events, touch) → `track.enabled = true` + `POST …/ptt {state: "down"}`. Release → `false` + `"up"`. The server side (native `turn_detection:null` or emulated mute) is m6b-02's. The mid-call toggle (AB30 F4) → `POST …/voice-mode {mode: "ptt" | "standard"}` |
| Hold / Talk | **Hold voice while I code** → `POST …/hold`, then tear down the PeerConnection and stop mic capture; the clock keeps running. **Talk** → a new `live` segment from `held` (a free reseed) |
| Teardown | On hold, grace, pause, finish, abandon, unmount and `pagehide`: stop every track and close the PC. On `pagehide`, call m6a-05's **`interruptBeacon({reason: "tab_closed"})`**: the `navigator.sendBeacon` path that m6a-01's interrupt route accepts as `text/plain`, with m6a-05's gateway exemption. **One tab-close mechanism for the text and voice HUDs**; record it in the decisions log |
| Privacy | Never `console.*` an SDP, an ICE candidate or a stats report. SDP carries the learner's IP addresses |

### 3 · Live HUD (AB30★) [X]

A voice mode of m6a-05's `web/src/screens/mock/LiveHud.tsx` (or `web/src/screens/mock/voice/VoiceHud.tsx` beside it; follow the merged layout). It reuses M6a's rail, clock, editor, Run, hint and safety-card pieces, and adds every AB30 frame:
- **Rail and clock (F1):** the phase rail with its goals and the server clock from the SSE. Nothing is computed client-side.
- **Captions (F2, F12):** interviewer captions come from the model's output transcript; candidate captions are labelled **"approximate"**. The log is navigable, `role="log"` with **`aria-live="off"`**, so a screen reader doesn't speak over the interviewer and trigger barge-in. **[Ask to repeat]** and **[Rephrase that]** → `POST …/ask {kind: "repeat" | "rephrase"}` (m6b-01; ≤ 6 / min, a 429 disables them with the reason).
- **Mic state:** live, muted, PTT held or released, lost (F7 "Reconnect mic"). **Camera state (F13):** off (the default), self-view on, lost (silent, [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)). Both use icon + text, never colour alone.
- **Editor and Run (F3):** m6a-04's CodeMirror interview mode and its snapshots (≤ 1 / 2 s, ≤ 64 KiB), unchanged. The Run echo is visible, and the model has no run tool.
- **Push-to-talk (F4):** the [Hold to talk] button and shortcut, and the control bar's **"Push-to-talk" toggle, switchable mid-call** → `POST …/voice-mode` (standard ↔ ptt, m6b-02). The mode indicator (Standard / Patient / PTT).
- **Hold voice while I code / Talk (F5).**
- **Money (F10):** follow the frozen AB30. Where AB30 is silent, the default is [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys): **no running $ figure while live**. Show only the cap state: an 85% "wrapping up to stay inside your $X cap" notice, and at 100% `finished(cut_short=cap)` with a clear message. Record which one the board chose.
- **Grace (F9, the voice variant of AB26):**
  - a 5:00 countdown and "Checked 20 s ago" (from m6b-02's `notice{kind:"probe"}`);
  - **[Check now]** at most once per 10 s, with probes every 30 s driven by the browser;
  - **[Save & resume later]** and **[End interview]**;
  - at ≥ 75% of the rail, **[Finish & get feedback]**, which becomes **[End without feedback]** while the key is dry;
  - the mic is shown muted and self-view is stopped.
- **Distress card (F14):** M6a's safety card (m6a-05, on m6a-02's `safety_card` system turn) in AB30 F14's copy, with [Pause interview] and [Keep going]. It is raised by words in the transcript, never by the voice.
- **Other states:**
  - `bridging` (F6, a planned rollover): "Reconnecting voice…";
  - `held` (F5): "Voice on hold · clock running · [Talk]";
  - `interrupted`, per reason: network, mic, provider, and idle. F8 "Still there?" appears on m6b-02's `notice{kind:"idle_check"}`, 60 s before m6a-01's `interrupt(idle)`; [I'm here] sends a heartbeat with a fresh `last_input_at`;
  - "Live in another tab" (F15);
  - the M6a paused banner and resume modal, plus task 1's reaffirmation (AB29 F13);
  - wrapping and the spoken debrief (F11, ≤ 3 min), with captions.
- **Check your transcript (F17), after the call:** in `finished` / `proposed`, before AB27's proposal review (at m6a-05's `mock/:id/review`, ahead of m6a-06's review screen). Candidate lines are editable inline (≤ 2 KiB) through `PATCH /api/interviews/{id}/turns/{seq} {text}` (m6b-02); interviewer lines are read-only. An "edited" chip follows a save, and the first edit shows the **self-reviewed** banner (m6b-02's `transcript_edited` caveat makes the saved score `scored_by=self`). **[Continue to scores]** leads to AB27, where voice-mode Communication is `self_only_voice` (m6a-03) and shows the AI's quotes.
- **Always visible:** the "AI interviewer" label. The spoken AI disclosure is the brain's (m6b-01).
- **Accessibility** ([t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)):
  - captions are on by default;
  - `aria-live` is used only for phase changes (polite) and the grace warning (assertive);
  - the screen-reader preset (F16: PTT + a headphones prompt) comes from AB28's settings;
  - the mute, PTT and end shortcuts are the ones AB30 names, and they must not clash with the editor's bindings;
  - `prefers-reduced-motion` is honoured;
  - the focus order and the < 1024 px layout (F18) follow AB30's behaviour notes.

### 4 · Optional local self-view [X]

`web/src/components/SelfView.tsx`:
- **Off by default.** Turning it on calls `getUserMedia({video: {facingMode: 'user'}, audio: false})` as a separate stream, shown in a mirrored `<video muted playsInline autoPlay>`.
- **Never** added to the PeerConnection, uploaded or recorded. There is no "camera required" gate.
- Turning it off, the grace, pause, finish, or unmount **stops every video track**.
- A denied permission shows an inline note. A lost camera is silent: the state indicator changes and the clock runs.
- Tests: after self-view is on, `pc.getSenders()` holds no video track, the SDP offer has **no `m=video`**, and the tracks are stopped on toggle-off and unmount.

### 5 · Tests, compose walk, screenshots [X]

**Vitest + RTL.** `RTCPeerConnection`, `mediaDevices` and `permissions` are stubbed in `web/src/test/`. Cover:
- **Gates:** Chrome / Edge / Firefox / Safari / mobile brands; `allowsFeature` false; `voice.reason` ∈ {`region`, `no_voice_key`, `model_access`, `capacity`, `daily_limit`, `disabled`} → the right frame, with text mode offered each time; `region` with `/api/me` `acceptance.region` null → the no-region variant.
- **Order and consent:** boxes unticked by default; [Continue] (and so Start) disabled until every box is ticked; the version is sent; **no `/segments` POST happens until the consent PUT succeeded, the cap is set and `start` returned `preflight`**; a mic denial happens before any `start` call; the reaffirmation shows on resume.
- **The client:**
  - `purpose` per FSM state (`preflight` in `preflight`; `live` in `connecting`, `interrupted`, `held`, `bridging`) and `client_id` in every POST;
  - exactly **one** `/segments` POST on 504, 502 or abort, with the text-mode offer; exactly one retry on 503 `draining`;
  - the ≈ 22 s signal, not 15 s;
  - confirm posted on `connected`, with the returned `segment_n`;
  - the data channel opened only when the DTO asks, and `session.update` never sent;
  - no video sender;
  - teardown on every exit, plus m6a-05's beacon interrupt on `pagehide`;
  - a console spy sees no `v=0` or `a=candidate`.
- **The HUD:** caption roles and the `aria-live` placement; ask/rephrase posting `ask`; PTT toggling `track.enabled` and posting `ptt`; the F4 toggle posting `voice-mode`; Hold/Talk; the grace countdown and the Check-now throttle; each interrupted reason; the idle check; the cap notices; the distress card; F17 (candidate lines only, ≤ 2 KiB, only in `finished` / `proposed`, the chip, the self-reviewed banner).

**Go:** any gate task 1 added (the `mic` reason at least, if missing) and the task 6 verb (a real-Postgres integration test, the way coach's store tests run).

**Compose walk.** Run coach against the scratchpad fake-LLM provider (the local-review note in the agent memory: the fake-LLM compose override, never committed). Extend it with the two endpoints **m6b-01's PR description** specifies: session create → a canned answer SDP, and a sideband WebSocket replaying scripted captions and usage. A canned answer can't complete ICE in a real browser, so add a **dev-only `RTCPeerConnection` stub** in `web/src/lib/voice/` that reports `connected` without media. It is honoured only when `import.meta.env.DEV` and an explicit dev flag are both set, it is compiled out of the production build (a build test greps `dist/` for its marker), and confirmation comes from the fake sideband's first event (m6b-01's "first media event wins"). Log in as the owner (dev login) and walk:
- **In compose:** every AB29 frame (F5 with the stub); an EU account (a compose account whose region is `DE`, set through the acceptance step) → the notice, no voice UI, and a `curl` on `/segments` → **403 `voice_unavailable_region` from coach through the gateway proxy**; a null region; a 429 busy; a 504 on the SDP route; and the AB30 frames the fake sideband can drive (captions, the rail, Hold/Talk, PTT and its toggle, the grace, idle, cap, distress, "Live in another tab", F17).
- **Left to Vitest:** mic loss, audio playback and anything needing real media.
- **Left to [m6b-04](sprint-m6b-04.md):** the real provider leg.

Record the split in the PR.

**Screenshots.** Every frame at **1440 px and 390 px**, side by side with the frozen boards, in the PR.

**Bundle and CI.** `npm run build`: the voice code sits in its own chunk and the main entry chunk grows by < 5 KiB (record the sizes). CI must be green: `go test ./...`, `go vet`, lint, the web suite, `sqlc diff` and the OpenAPI drift test.

### 6 · Prepare m6b-04 [X]

These must ship in this patch, because [m6b-04](sprint-m6b-04.md) runs against it on prod.

**a) `coach admin interviews media-audit <interview-id>`.** A read-only verb next to m6a-01's `coach admin interviews --live`, run the same sanctioned way: `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews media-audit <id>'`. It prints JSON and exits non-zero unless the verdict is `clean`. It reports:
- `binary_columns`: every `bytea` / `oid` column in schema `coach` outside an allowlist. The allowlist is the key envelope: `api_key_config.enc_key`, `enc_data_key`, and any keyring columns [m1-10](sprint-m1-10.md) added. It must be `[]`.
- per interview table (`interview`, `interview_turn`, the segment log, checkpoints, the snapshot timeline, consent): the rows for this interview and the maximum `pg_column_size`;
- `sdp_markers`: text values holding `a=candidate`, `a=fingerprint` or a line `v=0` → must be 0;
- `base64_runs`: text values with a run of ≥ 4 KiB matching `[A-Za-z0-9+/=]` → must be 0;
- the interview's segment count, voice minutes and the µUSD sum from the segment log, for m6b-04's cost check.

Use pgx directly for the catalog reads (`information_schema.columns`) if sqlc can't parse them; `sqlc diff` stays clean either way. Integration tests on real Postgres: a seeded clean interview → `clean`; a planted SDP turn → `dirty`; a planted base64 run → `dirty`.

**b) The QA slice, from the owner's setup UI.** [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) keeps a **0.33× "slice" rail for QA only**, and [m6a-01](sprint-m6a-01.md) already defines it: `qa_slice = true` (rail × 0.33), accepted only for `role = owner`; such an interview reaches `finished` and a proposal but never `ScoreMock` ([m6a-03](sprint-m6a-03.md) answers submit with 409 `qa_session_unscored`; the owner abandons it). Check that the owner can set it from the setup UI. If no toggle exists, add one: "QA slice (0.33×)", shown only when `/api/me` says the role is `owner`, sending m6a-01's `qa_slice` on create. If m6a-01's create route doesn't take `qa_slice` from the body yet, accept it there, owner-only, with m6a-01's refusal for anyone else. **No new `rail` field and no new error code.** It isn't a cohort flag and it never flips.

**c) The runbook `docs/runbooks/voice-fake-media-e2e.md`**, written for m6b-04 and for re-runs after a prompt or shell change ([t6 §14](../research/t6-realtime-interviewer.md#14-risks), solo QA). It covers:
- **Prerequisites:** the owner is present; his key is in a dedicated OpenAI project with a hard limit; no live interviews; **no starts today (UTC)**. m6a-01 counts a start at `setup → preflight`, so a failed voice check that returns the interview to `setup` needs a second start: one retry a day at most.
- **Expected spend:** about 0.33 × [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)'s 45-minute figure (GPT-Live ≈ $1.0, mini ≈ $0.7), plus a ≤ 3-minute debrief and the ≈ $0.02 voice check: **≈ $0.7–1.2**. The $ cap should be about 2.5× that, so the 85% wrap can't fire before the rail ends.
- **Clips.** Synthetic candidate audio from macOS `say`, with `[[slnc N]]` silences, then `afconvert -f WAVE -d LEI16@48000 -c 1`. Make **one short cycle (≈ 3 min) that Chrome loops**, holding generic DSA utterances (clarify, brute force, plan, one 45 s silence, edge cases, complexity). Every `getUserMedia` (the mic check, the voice check, Start, Talk after Hold) restarts the fake file, so a track paced to the rail can't hold its pacing. The clips are kept in the session scratchpad and **never committed**.
- **Chrome.** Launch it with a throwaway `--user-data-dir`, `--remote-debugging-port=<port>` (Chrome binds it to 127.0.0.1), `--use-fake-device-for-media-stream` and `--use-file-for-fake-audio-capture=<cycle>.wav` (looping, so no `%noloop`). Open `chrome://webrtc-internals` in a **separate window** beside the interview window, never over it: the interview tab must stay visible (`document.hidden` false), or silence plus a hidden tab reaches the idle interrupt.
- **Roles:**
  - the **owner** signs in, ticks consent, confirms the $ cap and clicks Start;
  - the **agent** drives the rest and records.
- **The step list and what to record** at each step, ending at `proposed`: the QA session is refused a score (409 `qa_session_unscored`) and the owner abandons it.
- **The no-media-on-node checks:** the media-audit verb, log greps, coach's pod volumes, and no UDP (the `chrome://webrtc-internals` selected pair).
- **The cost reconciliation** against the provider's usage page.
- **Cleanup:** delete the profile, the clips and the webrtc dump; never commit them.

### 7 · Tag the `v2.0.x` patch (M6b dark) and verify on prod [X + H + O]

1. Merge the xlearn PR (tasks 1–6, CI green). If S6 M7 failed, m6b-02's `coach-interview` Deployment must already be live.
2. Run the release checklist below, including **no live interviews**.
3. Tag the **next free `v2.0.N`** (`git ls-remote --tags origin 'refs/tags/v2.0.*'`). The GitHub release title is `v2.0.N — v2.1 build · M6b dark (voice)`.

Checks on prod, by the agent, without the owner's session:
- `/xlearn/api/v1/healthz` reports `v2.0.N`.
- Anonymous `POST /xlearn/api/v1/interviews/<id>/segments` → 401, and anonymous `/xlearn/` → unchanged.
- `curl -sI https://projects.sujaykumar.dev/xlearn/` → still `permissions-policy: camera=(self), microphone=(self)` and the unchanged CSP.
- The new chunk is served: fetch the asset named in the built manifest.
- `media-audit` runs (against the most recent text interview, or it prints "not found" cleanly).
- `coach admin interviews --live` is empty.
- `host-verify --cluster` is green.
- Every `xlearn-*` ImagePolicy's latest equals the tag, **except `xlearn-coach-interview`** (only if S6 M7 failed): it is pinned to an exact version (m6b-02) and lags until task 8's PR merges. Re-check it then.

**The owner's look (O, required, about 5 minutes).** The register asks for proof on prod that a cohort account can open the voice pre-flight, and the agent never signs in. The owner opens Mock-v2 → Voice and goes through the free steps only: the gates, mode, consent, mic permission and the mic check. He stops before the estimate's Start, so no paid voice check runs and no start is used; he then abandons the `setup` interview F1 created (or m6a-01's sweeper abandons it after 1 h, which isn't a start). He also confirms that a text interview's setup looks unchanged. Record what he saw. If he isn't available right after the tag, leave task 7 🔄 with this one item open; [m6b-04](sprint-m6b-04.md)'s entry gate re-checks it.

### 8 · Only if S6 M7 failed: the `coach-interview` ImagePolicy bump [I]

If S6 recorded M7 pass, mark this ✅ "n/a — M7 passed". Otherwise, after the tag and with `coach admin interviews --live` empty, open **its own PR in `../infra`** that moves `xlearn-coach-interview`'s exact pin to `2.0.N`, per m6b-02's runbook (`docs/runbooks/interviewer.md` § "coach-interview (M7 fallback)"). Merge it when its checks are green. Flux rolls `xlearn-coach-interview`; verify with `k3s kubectl get deploy -n xlearn` and re-check that policy's latest. GitOps only: never `kubectl apply`.

### 9 · Record [X]

`docs/v2/status.md`:
- the Sprint board row;
- the **M6b** milestone row: "dark in `v2.0.N`";
- **milestone → tag → floor → snapshot:** `M6b dark → v2.0.N → floor unchanged` (m6b-01/02 migrations are expand-only; if either added a contract, stop, because that was never planned) `→ no snapshot` (a patch with no contract, erase or GA flip);
- the artboard rows AB29/AB30 → "implemented (PR #)";
- the **flag inventory:** the interviewer's T-3 cohort-gate row (`interviewAudience`; owner M6a/M6b; removal `v2.1.0`, [m6b-04](sprint-m6b-04.md)), noting that voice reuses it, and `qa_slice` listed as an owner-only QA tool (not a flag);
- the privacy-notice outcome from task 1 ("covered", or the open PR);
- the owner's pre-flight look (task 7) and, if M7 failed, the task 8 infra PR;
- the **decisions log:**
  - the data-channel choice, the captions path, and no client mirror;
  - the pre-flight order (F6 before F5, as the FSM requires);
  - the cost-display call (the AB30 board vs t6 §8);
  - the PTT shortcut;
  - the EU/EEA gate: coach (m6b-01 check 5), verified here;
  - the `mic` interrupt reason (added here or already present);
  - the `pagehide` mechanism: m6a-05's beacon for both HUDs;
  - the ICE-gathering choice;
  - the `disabled` reason's frame;
  - the media-audit verb, the `qa_slice` toggle and the runbook link.

## Acceptance criteria

- [ ] Every AB29 and AB30 frame is implemented as frozen (1440/390 px screenshots in the PR), including the voice variant of the grace countdown, the F4 mid-call toggle, the F14 distress card and the F17 transcript check
- [ ] The pre-flight follows the FSM: consent and the cap come before `start`, and no `/segments` POST happens before `start` returns `preflight` (tested)
- [ ] The gates hold, and each refusal offers text mode:
  - an unsupported browser gets the notice;
  - an **EU/EEA account sees the notice and no voice**, and coach answers 403 `voice_unavailable_region` through the proxy on a voice create and on `/segments`;
  - a null region gets "Confirm your region";
  - no OpenAI `voice_shell` key gets the key notice;
  - the L19 429s get their frames
- [ ] Voice consent: the boxes start unticked, [Continue] (and so Start) stays disabled until every box is ticked, the version is stored with the interview's consent row, and resume shows the reaffirmation. The privacy notice is covered, or an owner-approval PR is open
- [ ] The voice client:
  - holds m6a-01's client lease (`attach` + heartbeat) and posts m6b-01's `{sdp, purpose, client_id}`;
  - makes exactly one SDP POST (no retry, except one after 503 `draining`) with a ≈ 22 s client budget, and confirms on `connected`;
  - falls back to text mode on 504;
  - never has a video sender and never an `m=video`;
  - never sends `session.update`;
  - never logs an SDP;
  - tears down on hold, grace, pause and `pagehide` (m6a-05's beacon interrupt)
- [ ] The HUD: captions log (`aria-live="off"`), ask/rephrase, mic and camera state, PTT and its mid-call toggle, Hold/Talk, the cap notices, every interrupted reason (including `mic`), the idle check, the distress card, "Live in another tab", the transcript check, and the AB30 shortcuts don't clash with the editor
- [ ] Self-view is local only (tested)
- [ ] `coach admin interviews media-audit` is merged with its tests; the owner can set m6a-01's `qa_slice` from the setup UI; the runbook is merged
- [ ] **M6b dark patch live on prod (cohort only):** `v2.0.N` is tagged with the release checklist and verified; the owner opened the voice pre-flight up to the free mic check; the interviewer defaults are unchanged; nothing changed for anonymous visitors

## Release

**Tag a `v2.0.x` patch (M6b dark, cohort).** It carries [m6b-01](sprint-m6b-01.md), [m6b-02](sprint-m6b-02.md) and this sprint. It is a **patch**: from `v2.0.0` on, the minor moves only at a GA flip ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)), and `v2.1.0` belongs to [m6b-04](sprint-m6b-04.md).

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist), verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag:**
- **ACL PRs:** n/a (no stream or consumer).
- **New service:** n/a. If S6 M7 failed, `coach-interview` reuses coach's image and its policy came in m6b-02; its exact pin moves in task 8's own infra PR after the tag. Until then, the "every policy's latest equals the tag" line excludes `xlearn-coach-interview`.
- **Contract, erase or GA:** n/a, so no snapshot.
- **No live interviews:** `ssh vps 'sudo k3s kubectl exec -n xlearn deploy/xlearn-coach -- coach admin interviews --live'` is empty.
- **Major:** equals `.release-line` = `2`.
- **Standing rule:** no new caller. gateway → coach, gateway → identity and coach → identity exist; coach → OpenAI 443 is mi-13's.
- **Flags:** none new. Voice sits behind the interviewer's existing T-3 cohort gate.

**Rollback:**
- **R-c** (the default): a revert plus the next patch.
- **R-b:** narrow to the previous `v2.0.x`. m6b-01/02 migrations are expand-only, and so is task 1 d's CHECK widening if one was needed, so the old coach runs on the new schema.
- **R-a:** not available, since M6b has no kill switch. The cohort gate already limits exposure to the owner and testers.

## Definition of Done

- CI is green (web suite, Go, `sqlc diff`, OpenAPI drift).
- The screens match the frozen AB29/AB30 (screenshots in the PR).
- `v2.0.N` is tagged, deployed by Flux (no hand `kubectl`) and verified live, including the owner's pre-flight look.
- If S6 M7 failed, task 8's infra PR is merged and `xlearn-coach-interview` runs `v2.0.N`.
- The acceptance criteria are met.
- The media-audit verb, the `qa_slice` toggle and the runbook are live for [m6b-04](sprint-m6b-04.md).
- Statuses are updated (this file and [`../status.md`](../status.md)).
- Local `main` is synced in every repo touched.

## Risks / watch-outs

- **A Permissions-Policy misconfiguration blocks the mic.** mi-13's header tests cover it. The browser gate's `featurePolicy.allowsFeature('microphone')` check turns a regression into a clear frame instead of a silent failure.
- **`apiFetch`'s 15 s timeout would abort the SDP call** before the 20 s route budget, and a retry would double-bill the learner. The call gets its own signal and is sent exactly once; only a 503 `draining` (no provider call made) earns one retry.
- **The pre-flight order.** Drawing F6 after F5 literally would put a paid voice check, and mic audio sent to a third party, before consent and the cap. The FSM and m6b-01 refuse it (409), so follow task 1's order and test it.
- **The EU gate only in the UI is not a gate.** Coach's refusal (m6b-01 check 5, verified in task 1 a) is what counts; the SPA only renders the server's decision. A null region fails closed. Don't add a second EEA list or a gateway 403.
- **Consent ticked on the learner's behalf, or pre-ticked, is invalid.** The boxes start unticked, and the server checks the version.
- **Autoplay and `getUserMedia` need a gesture.** Every place that (re)opens voice is a click: Test, Start, Resume, Talk.
- **Two tab-close paths for one interview** would drift. Reuse m6a-05's beacon (`text/plain`, its gateway exemption) for voice too.
- **Two leases.** m6a-01's client lease (which tab) and m6b-02's sideband lease (which pod) must never be conflated in the client, its tests or the docs.
- **Captions in a live region make a screen reader speak over the interviewer**, which triggers barge-in. Use `aria-live="off"` on the log.
- **The running $ figure vs the board:** t6 §8 hides it while live, and the register lists a cost meter. The frozen AB30 decides; record the call.
- **A video track in the offer would send the camera to a third party**, against ADR-0032 §3. Tests assert no video sender and no `m=video`.
- **SDP or stats in console logs leak IP addresses.** A console spy test guards against it.
- **The dev-only PeerConnection stub must never ship.** It is gated on `import.meta.env.DEV` plus a flag, and a build test greps `dist/` for its marker.
- **Shell drift since S6** (GPT-Live's data channel, event names). Re-read m6b-01's fixtures; the UI never depends on the provider's event names, only on our SSE.
- **Tagging mid-interview rolls coach.** Check `coach admin interviews --live` right before the tag.
- **The patch must not move the minor** (ADR-0034 §1.1).
- **This is a large sprint** (two boards, a WebRTC client, a CLI verb, a runbook, a tag). Work in task order. If the session can't finish, merge what is done and green (it ships dark behind the cohort gate), leave the tag undone, and report; don't tag a half-built HUD.
