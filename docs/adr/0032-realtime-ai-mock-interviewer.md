# ADR-0032 — Realtime AI mock interviewer (M6 · v2.1)

- **Status:** Proposed. **Spike S6 is approved and pending**; it needs the owner present and gates the M6a design freeze. **Kept Proposed at the v2 build-plan sign-off (2026-09-24, BP2):** it is accepted with the **S6 result** (sprint [spk-04](../v2/sprints/sprint-spk-04.md)), as task 1 of design sprint [ds-m6a-01](../v2/sprints/sprint-ds-m6a-01.md), before the M6a design freeze. Its amendments to [ADR-0007](0007-ai-coach-byo-key-and-secrets.md) (§7) and [ADR-0024](0024-public-user-dashboards-and-usernames.md) (D31) are folded in then.
- **Date:** 2026-09-24
- **Deciders:** @sujaykumarsuman
- **Related:** [0031](0031-platform-ai-and-two-tier-keys.md) (BYO only, the `interview` key, derived credentials), [0029](0029-judge-contract-and-learning-signal.md) (mock evidence, `ScoreMock` once), [0027](0027-content-evalpack-and-user-data-model.md) (C4 data, erase), [0028](0028-object-storage-and-backups.md) (no object store), [0017](0017-mock-model-and-projection-consumer-scaffold.md), [0024](0024-public-user-dashboards-and-usernames.md) (public mock stats, amended here), [0007](0007-ai-coach-byo-key-and-secrets.md) (amended here).
- **Links:** v2 topic T6, [feasibility § T6](../v2/feasibility.md#t6--realtime-ai-mock-interviewer) and the full [research appendix](../v2/research/t6-realtime-interviewer.md).

## Context

**The owner's spec:**
- a real 45–60 minute interview with the AI as interviewer;
- two-way voice;
- speech and coding assessed, ending with improvement areas;
- the coach gets a voice;
- runs on the learner's BYO key;
- failsafes: a 5-minute top-up grace, a pause of up to 24 hours, and a resume summary generated from the transcript on the learner's tokens.

**Constraints:**
- The VPS has **no UDP**.
- Traefik runs `web` and `websecure` only, and the gateway handles only short request/response traffic.
- The raw BYO key never leaves coach.
- There is no object store.
- Transcripts are C4 data and must stay outside assessment.
- Anthropic has **no audio API**.
- The EU AI Act Art. 5(1)(f) prohibits emotion inference in education and hiring.

## Decision

### 1. Scope and milestone

T6 is **not in v2.0**. It is milestone **M6**, which comes after M3 and can run in parallel with M4/M5. The proposed release is **v2.1**.

| Phase | Contents |
|---|---|
| **P0 (M6a)** | A **text interviewer** that works with any `interview` catalog model, Anthropic or OpenAI, plus all the owner's failsafes. Kept permanently as the accessible mode. |
| **P1 (M6b)** | **Voice** through **one** OpenAI shell, chosen by S6 |
| **P2 (M6c)** | Extras and a fairness check |

### 2. Architecture

- **Owner.** The interviewer lives in **coach** as `internal/coach/interview`. No new service.
- **Two components:**
  - a text **director brain** on the learner's `interview` key, which writes all evaluative speech;
  - a thin **voice shell** that only listens and speaks.
- **Media path.** The browser connects to OpenAI over WebRTC. **Coach brokers the SDP server-side, so no credential reaches the browser.**
  - Coach also holds an outbound **sideband** for transcripts, director notes, quota detection and hang-up.
  - **Audio copies on the sideband are dropped by event type, unparsed.** They are never written to disk or logs.
- **No infra change:** no UDP, no Traefik WebSockets, no object store.
- **Small changes needed:**
  - coach limits go to 500m / 256 Mi;
  - a graceful drain and make-before-break rollout;
  - `coder/websocket`;
  - a 20 s gateway route for the SDP exchange;
  - Permissions-Policy and CSP gates.
- **If S6 M7 fails** (the sideband can't survive a rollout), add a `coach-interview` Deployment of the same image with its own ImagePolicy.
- **Shell choice (S6, owner-approved).** GPT-Live-1 is chosen only if it passes every hard gate and is rated at least 1 point more natural; otherwise `gpt-realtime-2.1-mini`. P1 ships **one** shell.

### 3. What the AI perceives: "a second interviewer sharing the session" (owner: D29)

- **Live two-way voice**, plus the **on-screen answer widgets**: editor, canvas, text fields and choices, as **structured state** (code text, `canvas-graph@1` export, values), **not** screenshots.
- **Update triggers:**
  - on change, sampled every **2–3 s**, debounced, and never re-sent when unchanged;
  - on every candidate turn or question.
- **Context handling.** The live model keeps one replaceable **"current screen"** item. The director gets diffs.
- **Storage.** A **snapshot timeline** is stored alongside the transcript. The final analysis uses the **whole transcript plus the on-screen state at each point**.
- **Never assessed:** tone, emotion, accent, pronunciation, fluency, filler words, pace, face or appearance. The rubric's evidence enum has no audio or video member, and a CI lint enforces that.
- **Communication in voice mode is self-scored**, with the AI supplying quotes, until a multi-speaker fairness check passes (P2).
- **No video goes to the AI.** A local self-view is optional and off by default.
- **Peer-to-peer interviews** between learners, with video, are a **future item**:
  - signalling reuses the SDP-relay pattern;
  - **TURN** (managed vs self-hosted coturn with UDP) is a deferred T7 decision;
  - nothing is built for it in v2.

### 4. Session state and failsafes (owner spec)

- **Clock.** Coach runs a server clock over the manifest's phase rail.
- **Segments.** Every provider session is a segment. **Rollover, recovery and resume** all prime a **new segment from our own text state**; nothing depends on provider-side state.
- **Error typing.** Errors are classified by `error.code`: `ErrQuota`, `ErrAuth`, `ErrModelAccess`, `ErrRate`, `ErrSessionCap` and `ErrTransient`. A cheap same-key probe breaks ties. Quota errors **never** disable a key.
- **5-minute top-up grace:**
  - freeze the clock;
  - take a free checkpoint;
  - **hang up** (voice is billed per connected second);
  - show a countdown modal with a top-up link, and probe every 30 s;
  - on success, resume from the checkpoint with no paid summary.
- **Pause:**
  - the resume deadline is fixed at **24 h** from the first pause, with at most 3 pauses;
  - coach unlocks while paused;
  - any arena exposure to the mock's item is recorded (`pause_exposure`).
- **After 24 h: `incomplete`, unscored** (owner: D30). A free partial view is available; paid improvement areas run only on a click.
- **Resume modal:**
  - the free deterministic state (phase rail, time, code, hints) shows **at once**;
  - **[Summarise & resume]** runs one call on the learner's key (≈ $0.04), cached for that pause;
  - the new segment is primed with the brief, the state and the last few turns.

### 5. Assessment and scoring

- **Debrief.** A ≤ 3-minute spoken debrief with 2–3 improvement areas, written by the brain. It never states a score.
- **Review call.** One call on the learner's key (`store:false`) produces an `xlearn.mock_review@1` proposal, held in coach. It uses **public rubric descriptors only** and never pack material.
- **Accept step.** The learner must **explicitly accept or edit** the proposal; there is no auto-accept for mocks. Then **`ScoreMock` is called once.**
- **Label.** `scored_by=ai-byo` only when the proposal was accepted unchanged and the session was clean; otherwise `self`. `trust=honor` always.
- **Public profile (owner: D31).** It shows the **mock count only**. No best or average, for any mock. This amends ADR-0024.

### 6. Privacy, cost and limits

- **Storage:**
  - **Audio and video are never stored.**
  - Transcript plus snapshot timeline: C4, inline in coach, **deleted 30 days after scoring** by default, 12 months opt-in, per-turn delete, covered by the erase consumer.
- **Consent.** Per session, with unticked boxes, including a disclosure that audio passes through xLearn's server in memory only.
- **AI disclosure.** The interviewer is spoken as an AI at the start.
- **Availability:**
  - voice on **Chrome/Edge** only at launch;
  - **no voice for EU/EEA learners** until a legal review;
  - Anthropic-only learners get the **text** interviewer.
- **Cost on the learner's key:**
  - text ≈ $0.45 (45 min);
  - voice ≈ $2.1–3.1 (45 min), plus GST;
  - shown up front with a **mandatory, editable $ cap**, with wrap-up at 85% of it;
  - the D29 screen-context cadence cost is measured in S6.
- **Caps:**
  - one non-terminal interview per account;
  - ≤ 2 starts a day;
  - ≤ 75 voice minutes a day;
  - ≤ 3 live voice interviews platform-wide.

### 7. ADR-0007 amendments

- **Realtime credential:** SDP brokering only, with each segment creation logged. Fallback: a Realtime `client_secrets` token with a 30–60 s TTL.
- **Key lifetime:** the BYO key is decrypted only while a segment is live, then dropped.

## Consequences

- ✅ It feels like a real interview: voice, plus an interviewer who watches the candidate's work. No media ever touches the node.
- ✅ The failsafes behave as specified, and resume is cheap and deterministic.
- ✅ Assessment is legal and fair: words and on-screen work only.
- ⚠️ **Voice APIs are weeks old.** S6 gates the shell choice, and the text mode is always the fallback.
- ⚠️ **Screen-context updates every 2–3 s cost the learner tokens.** The replaceable current-screen item and diffs bound the cost, and S6 measures it.
- ⚠️ **Sideband audio copies pass through coach memory.** This is disclosed.
- ⚠️ **Voice is OpenAI-only.** Anthropic-only learners get text.
- ⚠️ **Solo QA burden.** Mitigated by replay regression tests in CI and a fake-media end-to-end run.

## Alternatives considered

| Option | Why not |
|---|---|
| Self-hosted media (LiveKit or Pipecat) or our own WebSocket audio relay | Needs UDP or TURN; all audio would pass through a 128 Mi pod |
| Gemini Live | Google is not a v2 BYO provider; it needs a browser token; sessions are short |
| A hosted pipeline with third-party STT/TTS keys | New key types and new data processors |
| Streaming camera frames or scoring delivery (pace, fillers, tone) | Emotion inference is prohibited (EU AI Act Art. 5(1)(f)); speech recognition is biased; little signal |
| Screenshots of the screen as context | Structured widget state is cheaper, exact and assessable (T4: grade from text export) |
| Storing audio for playback | Needs a bucket (D11) and adds C4 erase burden |
| Auto-accepting AI mock proposals | Would publish unreviewed "AI" scores |
| Showing mock best/average publicly | The owner chose count only |
| Two voice shells in P1 | Double maintenance on young APIs |
