# Sprint m6c-01 — Outline: "show your work" photo + chat read-aloud + local recording download

> **Milestone:** M6c — interviewer extras (v2.2) · **Track:** product · **Order:** 87 · **Kind:** **plan-outline card** (BP1: no prompt until v2.2 planning)
> **Prereqs:** [m6b-04](sprint-m6b-04.md) (`v2.1.0` interviewer GA live) · builds on [m6b-01](sprint-m6b-01.md) (`VoiceShell`, SDP broker, sideband), [m6b-02](sprint-m6b-02.md) (L19 voice caps, per-segment cost, $ cap), [m6b-03](sprint-m6b-03.md) (voice consent, the Chrome/Edge and EU gates), [m6a-01](sprint-m6a-01.md) (interview tables, consent rows, erase) and [m6a-02](sprint-m6a-02.md) (director brain, `store:false`)
> **Unblocks:** nothing gates on it. Siblings: [m6c-02](sprint-m6c-02.md), [m6c-03](sprint-m6c-03.md).
> **Sources:** [rollout §4](../rollout-plan.md#m5-later-2x-and-m6c-v22) (the M6c row) and [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading) (the v2.2+ row); [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P2.
> **Release action:** **outline only (v2.2).** This card merges nothing and tags nothing. Once expanded, each build sprint is "merge only", shipping dark to the **T-3 cohort only** in `v2.1.x` patches (a T-1 cohort-audience code default, like [m6a-01](sprint-m6a-01.md)'s `interviewAudience`; no T-2 env). The label is the M6c GA flip at the **next free minor after `v2.1.0`**: `v2.2.0` is indicative, or the next minor if [m5-01](sprint-m5-01.md) has already taken it ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme), [§1.6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#16-indicative-tag-timeline)). A dedicated M6c GA tag sprint cuts it; task 1 has the v2.2 planning add it. No infra PR is expected: t6 lists P2 as "none server-side".
> **Calendar:** v2.2+, after `v2.1.0` (≈ Q1 2027), so ≈ Q2 2027 or later *(inferred)*. No owner event is booked. The only owner input is the photo ruling, settled at v2.2 planning before any prompt exists (entry gates); the design sprint's board PR merges on CI green, and the merge is the freeze (D40).
>
> **Execute with:** no prompt yet. v2.2 planning expands this card into a full plan and writes `../prompts/prompt-m6c-01.md` (one prompt, one session). Expect about two sessions: the board-variant design sprint `ds-m6c-01`, which all three M6c cards share, and one build sprint.

## Status

_Overall:_ ⬜ Not started. This is an **outline card**: expand it at v2.2 planning before doing any work.

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Expand this card into a full plan and prompt at v2.2 planning | X | ⬜ |
| 1a | Sketch: "show your work" photo sent to the brain, never stored (O2b). Needs an owner ruling against D29, settled at v2.2 planning | X · O | ⬜ |
| 1b | Sketch: chat-coach read-aloud in the interviewer's voice | X | ⬜ |
| 1c | Sketch: opt-in local recording download (never uploaded) | X | ⬜ |

> Rows 1a–1c are sketches, not commitments. The expansion replaces them with real tasks, and it may defer or drop any of them.
> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md): the Sprint board row, which reads "⬜ outline" until the expansion, and the M6c milestone row.
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **M6b shipped:** `v2.1.0` (interviewer GA) is live and verified ([m6b-04](sprint-m6b-04.md)). [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) is Accepted with the S6 results folded in ([ds-m6a-01](sprint-ds-m6a-01.md)).
- [ ] **This card has been expanded** at v2.2 planning into a full plan plus `prompt-m6c-01.md` (BP1). Nothing here can be executed before that.
- [ ] **The owner's photo ruling (1a) is settled at v2.2 planning**, before any prompt exists (a planning input, not a session stop), and the ruling is in the decisions log. The options are keep, narrow (file or phone photo only, no webcam) or drop. The reason: [D29](../feasibility.md#decisions-log-newest-first) (t6 [§13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict), which overrides the body) says "**no video goes to the AI**" and "the camera is not part of the AI interview". The O2b photo comes from the older body text ([t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P2).
- [ ] **Board variants are frozen before the expansion's first UI sprint** (BP3). The expansion schedules one design sprint for all of M6c (e.g. `ds-m6c-01`, whose board PR merges on CI green: the merge is the freeze, D40). It drafts variant frames on the frozen boards, only for the rows each card's expansion keeps:
  - this card: AB01 coach states (the read-aloud control and its consent); AB24 and AB29 pre-flight and consent (the photo and recording opt-ins); AB25 and AB30 live HUDs (the photo action and the recording indicator);
  - [m6c-02](sprint-m6c-02.md): the AB27 voice-Communication proposal row (a proposed band with verified quotes) and the AB24/AB29 "Based on your last N sessions" estimate line;
  - [m6c-03](sprint-m6c-03.md): AB29's widened browser copy (Safari) and the voice-lite "device voice" label and consent (AB25/AB29).
- [ ] **Parallel sessions:** before claiming an ADR number or tagging, check peers' open PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents).

## Goal

Add **opt-in extras that need no server-side storage** ([t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P2). With them, the learner can:
- show paper working to the interviewer, and xLearn never keeps it;
- hear chat-coach replies in the interviewer's pinned voice;
- keep a recording of their own interview **on their own device**.

Every extra keeps the v2.1 invariants:
- **nothing is stored** (PRD R-MI8, [ADR-0028 §1](../../adr/0028-object-storage-and-backups.md#1-no-object-store-in-v20), D11);
- **no new media path through the node**, beyond the sideband copies M6b already discloses ([ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture));
- **content-only assessment** ([ADR-0032 §3](../../adr/0032-realtime-ai-mock-interviewer.md#3-what-the-ai-perceives-a-second-interviewer-sharing-the-session-owner-d29), [§5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring));
- **BYO key only**;
- **explicit, unticked consent** for each extra.

## Scope

**In (outline)**
- **1a:** an opt-in "show your work" **still photo** of written working. It goes to the director brain on the learner's key and is never stored (O2b), **subject to the owner ruling**.
- **1b:** chat-coach **read-aloud** in the pinned interviewer voice ([t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream), the PRD V6 amendment: "coach needs a voice").
- **1c:** an opt-in **local recording download** of a voice interview, never uploaded ([t6 §12](../research/t6-realtime-interviewer.md#12-alternatives-considered): "P2 offers a local-only download instead" of stored playback).

**Out**
- The multi-speaker fairness check, AI-proposed voice Communication and history-calibrated estimates → [m6c-02](sprint-m6c-02.md).
- Safari, voice-lite, the weekly canary and AB31 → [m6c-03](sprint-m6c-03.md).
- Stored audio playback, a media bucket, a Gemini adapter, a second shell and splitting out an `interview` service → t6 §9 **P3**, only on its triggers ([ADR-0028](../../adr/0028-object-storage-and-backups.md) stays: no object store).
- **Video to the AI** in any form. Peer-to-peer video interviews and TURN are a D29 future item; nothing is designed in v2.
- A new always-on pod or any `../infra` change. None is expected. If the expansion finds one, it becomes its own infra task, checked against the memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).

## Tasks

### 1 · Outline only: expand at v2.2 planning [X]

The v2.2 planning session is a docs-only session, like the v2 build-plan session. It replaces this card with a full plan in the v2 sprint format and writes the prompt. It must:
- **Re-read what exists by then:** [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P2, the Accepted ADR-0032, the S6 results note ([spk-04](sprint-spk-04.md)) and the shipped M6b code (`internal/coach/interview/voice`, and the AB29/AB30 components under `web/src`). The sketches below predate all of it.
- **Split the work into sessions:** most likely `ds-m6c-01` (board variants for all three M6c cards; the PR merges on CI green and the merge is the freeze, D40) plus one or two build sprints. Name the design sprint in the Prereqs of every M6c UI sprint.
- **Gate each extra:** each ships **dark to the T-3 cohort only**, through a T-1 cohort-audience code default (the [m6a-01](sprint-m6a-01.md) `interviewAudience` pattern). **No T-2 env is added**, which keeps "no infra PR" true. Its default flips at the labelled minor ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)). The flags go into status.md's flag inventory with M6c as both the owning and the removal milestone. **Budget the count:** this card's three extras plus m6c-03's Safari widening and voice-lite could add up to five non-kill flags, and the inventory must stay at **≤ 6 live non-kill flags**. If there is no room, use one shared M6c audience constant. m6c-02's `VOICE_COMM_AI` is a permanent kill switch and doesn't count toward the limit.
- **Add the M6c GA tag sprint:** a dedicated sprint (e.g. `m6c-04`, "tag the next free minor after `v2.1.0`") that flips the defaults of the extras that have merged and been dogfooded. It copies the [ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) release checklist plus the ADR-0035 §2 NetworkPolicy standing rule, and it carries the single batched privacy-notice version bump (1a Privacy). m6c-02's voice-Communication flip is **not** a precondition for it.
- **Record the calls:**
  - **Photo kept or narrowed:** both shapes let the AI perceive an image, which ADR-0032 §3 rules out ("structured state … not screenshots"; D29 "no video goes to the AI"). Either one needs a **new ADR that amends ADR-0032 §3**: the AI may receive a learner-chosen still of written work, and it is never scored. The same ADR records the photo cap as an **L19 extension that amends ADR-0035 §4**, and read-aloud's counting toward L19 (1b). Accepted ADRs are never edited in place. Take the next free number after the parallel-sessions check.
  - **Photo dropped:** a decisions-log line is enough. Read-aloud's L19 counting then goes into a decisions-log line too.

### 1a · Sketch: "show your work" photo → brain, not stored (O2b) [X · O]

**Owner ruling first [O]**, settled at v2.2 planning before any prompt exists (a planning input, not a session stop). D29 keeps the camera out of the AI interview. Three shapes are on the table:
- **keep:** a learner-initiated still from the webcam or a file;
- **narrow (recommended):** a file or phone photo only, with the webcam never opened by this feature;
- **drop.**

In every shape it is still an image, not structured widget state. So it can feed the conversation, but it must never become scored evidence (t4: the LLM grades text exports, never images).

**If kept (sketch) [X]:**

**web.** A "Show your work" action in the text and voice HUDs (AB25, AB30):
- `<input type="file" accept="image/*" capture="environment">`, with no `getUserMedia` stream;
- client-side re-encode through `<canvas>` to a JPEG of **≤ 700 KiB**, which strips EXIF and GPS. Base64 inflates it by about 4/3, so the JSON body stays under the 1 MiB body limit;
- the learner previews the photo and confirms before it is sent.

The unticked per-session consent line goes in AB24/AB29: *"A photo I choose to share goes to my AI provider on my key for this conversation only. xLearn never stores it. Photograph your work, not yourself."*

**gateway.** `POST /api/interviews/{id}/photo` → coach, in `internal/gateway/interview.go` (the [m6a-01](sprint-m6a-01.md) `/api/interviews/*` proxy):
- **the body is JSON**: `{"image_b64": "…", "mime": "image/jpeg"}`. [m1-04](sprint-m1-04.md)'s rule for mutating `/api/*` applies unchanged: `Content-Type: application/json`, else 415 `unsupported_media_type`, plus the `Sec-Fetch-Site` check. There is **no carve-out** for `image/*` or `multipart/*`;
- aud=coach, and the cohort gate until the flip;
- `httpx.ReadBody` at the 1 MiB default, giving the typed 413 `body_too_large` ([m1-05](sprint-m1-05.md); [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L6). A decoded image over 700 KiB, or one that isn't a JPEG, is refused with a typed 4xx;
- a per-session cap, e.g. ≤ 5 photos. It is an L19 extension, recorded in the amending ADR (task 1);
- the body is never logged.

**coach.** `internal/coach/interview` sends the image as image input in **one** director-brain call on the `interview` key:
- `store:false` on OpenAI;
- the bytes are held in memory for that call only;
- the brain returns a text note of ≤ 500 tokens ("the sketch shows two pointers …");
- **only that text note** may be stored, as an `interview_turn` (e.g. `source=photo_note`), never the bytes.

The catalog gains an image-input capability field next to the `interview_brain` and `voice_shell` capabilities ([t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream), amending [T5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2)), and models without it hide the action.

**Privacy.**
- The canary log test ([t5 §8](../research/t5-platform-ai.md#8-privacy-and-residency)) is extended to the photo route and the brain call: no bytes and no base64 in logs, panic values or rows.
- The ADR-0032 / [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) data inventory gains a "photo: to your provider, never stored" row.
- **The privacy notice** (`web/src/content/privacy-notice.md`, with [l-05](sprint-l-05.md)'s notice-version parity test) gains the same row. A notice change **bumps the notice version**. Every account, the owner and testers included, then gets l-05's `403 acceptance_required` on every non-onboarding API until it re-accepts the notice. So **batch all M6c notice changes** into **one** version bump at the M6c GA minor: the photo, read-aloud (1b) and any voice-lite cloud recognizer or network voice ([m6c-03](sprint-m6c-03.md)). The notice text lands as drafted (D40), as l-05's does; the owner may revise it later with a content PR. Until that bump, each extra's unticked consent carries its own disclosure for the cohort.

**Assessment stays content-only.**
- The never-list applies to images too: no face, appearance or background ([t6 §6](../research/t6-realtime-interviewer.md#6-assessment)).
- The brain is told to describe only the written work.
- The post-hoc never-list scan covers `photo_note` turns.
- **Open question:** can a session that contains a photo note still be `ai-byo`? The safe default treats a photo note like a `source=client` turn, which makes the session `scored_by=self`.

**Other open questions:**
- webcam still vs file only;
- the per-session cap;
- image cost per call (≈ 1–2k input tokens *(inferred)*), shown before sending;
- Anthropic vs OpenAI image size limits.

### 1b · Sketch: chat-coach read-aloud in the interviewer's voice [X]

Sources: [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (the PRD V6 row: "coach needs a voice" = the pinned interviewer voice, with read-aloud in P2) and [ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture). The voice is one pinned catalog voice per course persona (default `marin`), identical across rollovers and resumes.

**Mechanism (recommended; decide at expansion).** Reuse M6b's `VoiceShell`:
- coach brokers a short **output-only segment** over a **new read-aloud SDP route**, e.g. `POST /api/coach/read-aloud/segments {thread_id, message_id, sdp}` in `internal/gateway/coach.go`. [mi-13](sprint-mi-13.md)'s `POST /api/interviews/{id}/segments` can't carry it: that route is interview-scoped, cohort-gated and served by coach's interview segment handler, and the chat coach has no interview id. The new route copies its properties: a 20 s budget, a 20 KiB JSON body with a typed 413, aud=coach, no retry, no SDP logging, and the same EU and browser gates;
- coach looks up the already-generated reply in the stored thread (it never takes the text from the request) and sends it as **fixed text** over the sideband (GPT-Live `session.commentary.append`, or a Realtime fixed-text response);
- the audio flows **OpenAI → browser** over WebRTC, so nothing new touches the node.

This needs S6/M6b to confirm that a `recvonly` offer (no mic track) is accepted. If it isn't, fall back to a muted mic track with `track.enabled=false`.

**Rejected:** a server-side TTS REST call that relays audio bytes through coach and the gateway. That puts media on the node's path and contradicts ADR-0032's "no media ever touches the node".

**Free fallback:** the browser's `speechSynthesis`, labelled "device voice (not the interviewer's)". It serves:
- Anthropic-only keys (Anthropic has no audio API);
- Firefox and Safari while they are gated;
- EU accounts, if the gate below applies.

It is only "on-device" for voices with `SpeechSynthesisVoice.localService === true`. Chrome's default "Google …" voices are **network voices** (`localService === false`): the reply text goes to Google, a processor the notice doesn't name. So either restrict the fallback to local voices and hide it when there are none, or disclose the network voice and take its own unticked consent. Decide at expansion.

**web.** A "Read aloud" control on each reply in `web/src/components/Coach.tsx` (an AB01 variant):
- stop and skip controls;
- the cost on the control ("≈ $0.05/min on your key" on GPT-Live, or whatever the shipped shell costs);
- no captions needed, because the text is already on screen.

**coach** (`internal/coach/handlers.go` plus the voice package):
- a read-aloud segment kind, logged in the segment log with its cost ([m6b-02](sprint-m6b-02.md)). It has no interview, so the segment row needs a non-interview owner (a nullable `interview_id` or its own table; decide at expansion);
- it **counts toward L19**: ≤ 3 live voice sessions platform-wide and ≤ 75 voice-minutes per account per day. m6b-02's counters sum only interview segments today, so they must be widened to include these **non-interview segments** (recorded as in task 1);
- a per-click cap of about 3 minutes;
- idle hang-up as soon as playback ends.

**Consent.** Read-aloud gets its **own** unticked consent, once per account, in AB01's variant. M6b's disclosure sits in the per-interview voice consent, which never runs for the chat coach. It says: the audio is generated on the learner's OpenAI key; coach joins the session's **sideband** to send the text and read usage, as M6b discloses for interviews; no audio passes through xLearn.

**Gates and locks.**
- The Chrome/Edge gate is the same as for voice.
- **EU gate (open question):** the EU block exists because the live model perceives the learner's speech ([ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits)). Output-only read-aloud has no mic, but keep the gate unless the legal review clears it.
- Read-aloud exists only where the chat coach does. The coach stays locked during touches and mocks (D27), and the post-mock `review`-mode coach is the only mock-adjacent surface.

**Other open questions:**
- Is `recvonly` accepted?
- What is the per-reply length cap?
- Is reading the spoken debrief (AB27) back in scope?
- Device-voice fallback: local voices only, or disclosed network voices with consent?

### 1c · Sketch: opt-in local recording download (never uploaded) [X]

Sources: [t6 §12](../research/t6-realtime-interviewer.md#12-alternatives-considered) (stored playback rejected: it needs a bucket (D11), it's C4 data and it carries an erase burden) and PRD R-MI8.

**web only.**
- An unticked per-session opt-in in the AB29 pre-flight: *"Save a recording of this interview to this device. It is never uploaded."*
- `MediaRecorder` over a **WebAudio mix** of the local mic track and the remote interviewer track (the `RTCPeerConnection` receiver), recorded as Opus/WebM in timesliced chunks held in memory.
- The mix node **persists across segments**: rollover, grace recovery and hold/talk each create a new peer connection, and the remote track is re-wired.
- Download happens at finish or pause through an object URL, one file per sitting (a pause of up to 24 h splits sittings). If the tab closes before the download, the recording is lost, and the copy says so.
- Text mode can offer a plain transcript download instead (optional).

**No server change.** Invariant tests:
- web: recorder blobs never reach `fetch`, `XMLHttpRequest` or `sendBeacon`;
- gateway: [m1-04](sprint-m1-04.md)'s existing rule (every mutating `/api/*` needs `Content-Type: application/json`, else 415 `unsupported_media_type`) already refuses `audio/*` and `multipart/*`. 1a's photo route takes base64 JSON, so it needs no carve-out. A test pins the 415 on `/api/interviews/{id}/segments`, the photo route and the read-aloud route;
- CSP stays `connect-src 'self'` ([mi-13](sprint-mi-13.md)).

**Copy.**
- Remind the learner about bystanders.
- The file holds the learner's voice and the AI's, and it's the learner's to keep safe.
- The AI-disclosure line stays.

**Size.** 60 minutes of Opus at ~32 kb/s is ≈ 14 MB in browser memory *(inferred)*.

**Open questions:**
- Should the recording include the local self-view video? Recommended: no, audio only, so the camera stays out of the feature.
- File naming.
- Should recording be allowed while self-view is on?

## Acceptance criteria

- [ ] n/a (outline). Nothing here is executable until the card is expanded.

Candidate acceptance to refine at expansion (not gates today):
- [ ] No photo bytes, read-aloud audio or recording blob ends up in any Postgres row, log line, panic value or volume. The canary log test is extended, and prod is inspected the way [m6b-04](sprint-m6b-04.md) checks for no media.
- [ ] Photo route: JSON base64 body; typed 413 above the 1 MiB body (an image over ≈ 700 KiB is refused); a non-JSON body gets 415; EXIF stripped client-side; the action is hidden for models without image input; consent unticked by default; a photo never becomes scored evidence.
- [ ] Read-aloud: its own route and its own unticked consent; audio flows only between the browser and OpenAI; its non-interview segments count toward the L19 voice minutes and the live-voice count; it stops at its cap; the device-voice fallback is labelled and uses only local voices (or a disclosed, consented network voice).
- [ ] Recording: downloads locally across a rollover and a pause; no request carries audio; m1-04's 415 is pinned for `audio/*` and `multipart/*` on the interview and read-aloud routes.
- [ ] Each extra is visible only to the cohort (a T-1 audience default, no T-2 env) until the M6c GA flip. Each is listed in the status.md flag inventory, which stays at ≤ 6 live non-kill flags.
- [ ] The M6c notice rows ship as one notice-version bump at the M6c GA minor, with the re-acceptance tested.

## Release

**Outline only (v2.2).** This card merges nothing and cuts no tag.

Once expanded:
- **design sprint:** "land-and-sync; the merge is the design freeze" (the board PR merges on CI green, D40);
- **build sprints:** "merge only (ships dark to the cohort in the next `v2.1.x` patch)". Every patch runs the ADR-0034 §6 checklist, **including "from M6: no live interviews"** (`coach admin interviews --live` must be empty);
- **the M6c GA flip:** the dedicated M6c GA tag sprint that task 1 adds (e.g. `m6c-04`). It tags the next free minor after `v2.1.0` and flips the defaults of the extras that are ready. It copies the [ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) release checklist plus the ADR-0035 §2 NetworkPolicy standing rule, and carries the batched notice-version bump. No NetworkPolicy change is expected, because no new in-cluster caller is added;
- **infra:** no infra PR, because no T-2 env is added.

## Definition of Done

**This card:**
- it exists with its sources linked;
- it is listed on the [`../status.md`](../status.md) sprint board as "⬜ outline";
- no code and no prompt.

**At expansion:**
- this file is replaced by a full plan in the v2 sprint format, and `../prompts/prompt-m6c-01.md` is written;
- the O2b ruling is recorded, with the amending ADR if the photo is kept or narrowed;
- the shared board-variant design sprint (`ds-m6c-01`) is scheduled;
- the M6c GA tag sprint is added;
- the flags are entered in status.md, with the ≤ 6 count checked.

## Risks / watch-outs

- **Conflict with D29 (photo).** The O2b photo predates D29's "camera is not part of the AI interview". Building it without an explicit owner ruling would contradict a settled decision. Get the ruling at v2.2 planning, before any prompt exists. If the answer is keep or narrow, record it as a new ADR amending ADR-0032 §3, and ADR-0035 §4 for the photo cap.
- **"No media on the node" creep.** A server-side TTS call for read-aloud would relay audio through coach and the gateway. Keep the audio on the browser↔OpenAI path, or use the device voice.
- **The never-list through images** (EU AI Act Art. 5(1)(f); [t6 §6](../research/t6-realtime-interviewer.md#6-assessment)). A photo can include a face or a room. Mitigations: brain instructions, the consent copy, the post-hoc scan of photo notes, and **no scoring from images**.
- **Cost surprises on the learner's key.** GPT-Live bills read-aloud per connected second. Show the cost on the control, cap each read, and count it in L19.
- **Provider drift since S6/M6b.** Re-run the scrubbed fixtures at expansion. [m6c-03](sprint-m6c-03.md)'s optional canary helps.
- **Owner-only v2 (D35).** Real demand for extras only appears at the v3 opening. The expansion should build only what the owner or testers will actually use, and may defer the rest.
