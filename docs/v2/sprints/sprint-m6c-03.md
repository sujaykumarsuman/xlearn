# Sprint m6c-03 — Outline: Safari leg, voice-lite on demand, weekly canary; AB31 canvas outline (SD course, v2.2+)

> **Milestone:** M6c — interviewer extras (v2.2) · **Track:** product · **Order:** 89 · **Kind:** **plan-outline card** (BP1: no prompt until v2.2 planning)
> **Prereqs:** [m6b-04](sprint-m6b-04.md) (`v2.1.0` interviewer GA live). Builds on:
> - [spk-04](sprint-spk-04.md): the S6 Safari and Firefox legs (M15) and the browser gate list;
> - [m6b-03](sprint-m6b-03.md): the Chrome/Edge capability and user-agent gate, and AB29's browser notice;
> - [m6b-04](sprint-m6b-04.md): the fake-media run procedure the canary reuses;
> - [m6a-02](sprint-m6a-02.md) and [m6a-04](sprint-m6a-04.md): the text-interviewer path, including bounded SSE, which voice-lite reuses.
>
> **Unblocks:** nothing in v2.2. AB31 is parked for the SD course's own planning (v2.2+, not yet planned). Siblings: [m6c-01](sprint-m6c-01.md), [m6c-02](sprint-m6c-02.md).
> **Release action:** **outline only (v2.2).** This card merges nothing and tags nothing. Once expanded:
> - **Safari widening and voice-lite:** merge only, shipping dark **to the T-3 cohort only** in `v2.1.x` patches (a T-1 cohort-audience code default; no T-2 env, so no infra PR). Their default flips are labelled at the M6c GA minor, the next free minor after `v2.1.0` (`v2.2.0` indicative, or the next one if [m5-01](sprint-m5-01.md) has already taken it) ([ADR-0034 §1.1](../../adr/0034-v2-release-labelling-gating-and-rollback.md#11-scheme)). The dedicated M6c GA tag sprint that [m6c-01](sprint-m6c-01.md) task 1 adds cuts it.
> - **The canary:** a runbook plus a script, merged; nothing deploys.
> - **AB31:** "land-and-sync; the merge is the design freeze" (D40), but only in a design sprint at SD-course planning, **not** in M6c.
>
> **Calendar:** v2.2+, after `v2.1.0` (≈ Q1 2027) *(inferred)*. The Safari leg needs the owner present (≈ 30 min). The canary is a manual weekly check, read by looking.
>
> **Execute with:** no prompt yet. v2.2 planning expands this card into a full plan and writes `../prompts/prompt-m6c-03.md` (one prompt, one session). If 1a or 1b is kept, add a share of the M6c design sprint (`ds-m6c-01`, see the entry gates).

## Status

_Overall:_ ⬜ Not started. This is an **outline card**: expand it at v2.2 planning before doing any work.

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Expand this card into a full plan and prompt at v2.2 planning | X | ⬜ |
| 1a | Sketch: Safari. Re-run the S6 Safari leg and widen the browser gate if it passes (only if it failed in S6 and has since been fixed) | X · O | ⬜ |
| 1b | Sketch: voice-lite (Web Speech → brain → `speechSynthesis`), only on demand | X | ⬜ |
| 1c | Sketch: optional weekly 30 s voice canary on the owner's key, read by looking (D34) | X · O | ⬜ |
| 1d | Sketch: AB31 A6 Workspace-Canvas (Excalidraw) outline for the SD course (drafted at SD-course planning, not here) | X | ⬜ |

> Rows 1a–1d are sketches, not commitments. Each has its own trigger (entry gates), and the expansion drops any row whose trigger hasn't fired.
> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line to match, and mirror the sprint's state into [`../status.md`](../status.md):
> - the Sprint board row ("⬜ outline" until the expansion);
> - the M6c milestone row;
> - the AB31 artboard row, which stays "outline only" until its design sprint;
> - once the canary exists, a manual-checks row beside the evalpack PAT expiry.
>
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **M6b shipped:** `v2.1.0` (interviewer GA) is live and verified ([m6b-04](sprint-m6b-04.md)).
- [ ] **This card has been expanded** at v2.2 planning into a full plan plus `prompt-m6c-03.md` (BP1).
- [ ] **1a only if:** the S6 Safari leg **failed** (the [spk-04](sprint-spk-04.md) results note, M15) **and** an upstream fix is known (a WebKit or OpenAI release note, or a closed issue). If S6 passed Safari, it is already in the gate list and 1a is moot. If no fix is known, 1a is dropped.
- [ ] **1b only if:** an Anthropic-only user asked for voice, and the request is recorded in the decisions log. In v2 that means the owner or a tester (D35). Real demand arrives at the v3 opening, so this is likely later.
- [ ] **1c only if:** the owner opts in. It spends about $0.03 a week on his own key and needs his account. Each run also uses **one of his ≤ 2 interview starts that day** (1c).
- [ ] **Board variants are frozen before the expansion's first UI sprint** (BP3), for the rows kept. They are drafted in the shared M6c design sprint (e.g. `ds-m6c-01`, whose PR merges on CI green: the merge is the freeze, D40; see [m6c-01](sprint-m6c-01.md)'s entry gates):
  - **1a:** AB29's widened browser copy ("Voice runs on Chrome, Edge or Safari N+") and the Safari version-floor notice;
  - **1b:** the voice-lite "device voice" label, its mic control and any recognizer consent (AB25/AB29).

  If that design sprint merged without them, follow the [m5-01](sprint-m5-01.md) precedent instead: variant screenshots committed with the build PR, and the merge freezes the variant (D40; the owner may revise it later with a follow-up PR).
- [ ] **1d is not drafted in M6c.** AB31 needs:
  - the **SD course** scheduled by its own planning;
  - **M4** shipped (`ai_rubric`, the llm lane);
  - the canvas widget planned;
  - **calibration sets of ≥ 80 labelled per rubric** ([rollout §8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading), [t5 §10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream) SD-course row).

  It is drafted and frozen before that course's first UI sprint (BP3).
- [ ] **Parallel sessions:** before claiming an ADR number or tagging, check peers' open PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`, `git worktree list`, ListAgents).

## Goal

**Browser reach and cheap health checks** for the voice interviewer:
- open voice to Safari once it works;
- offer a device-side voice mode to keys with no audio API, but only when someone asks;
- give the owner a 30-second weekly check that the voice path still works, **read by looking**, never pushed (D34).

Also **park the canvas board (AB31)** with a brief that the SD course's planning can pick up without re-research.

## Scope

**In (outline)**
- **1a:** a cohort-only Safari admission, then the Safari leg, then the flip (web gate, AB29 copy, PRD R-MI1 wording, and a decisions-log line against ADR-0032 §6's "at launch").
- **1b:** voice-lite as a mode of the text interviewer: browser speech recognition in, `speechSynthesis` out, no new server path.
- **1c:** the voice canary as an owner-run fake-media script plus a runbook, with results recorded in status.md.
- **1d:** the AB31 A6 Workspace-Canvas brief: frames, constraints and where it will be drafted.

**Out**
- **Firefox.** Its WebRTC sessions to OpenAI Realtime drop after 1–2 turns (openai-agents-js #1353), and the WebSocket transport is ruled out ([t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options)). If it is ever fixed, 1a's procedure applies, in a later card.
- A Gemini adapter and a second shell → t6 §9 **P3**, only on its triggers.
- **Any alerting:** a push channel, healthchecks.io, an opscheck CronJob or a Flux Alert (D34, [ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34)). The D34 revisit is a v3 opening gate ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- **Building the SD course, the canvas widget or `ai_rubric` calibration.** That is its own v2.2+ planning ([rollout §8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading)).
- The photo, read-aloud and recording → [m6c-01](sprint-m6c-01.md). The fairness check and estimates → [m6c-02](sprint-m6c-02.md).

## Tasks

### 1 · Outline only: expand at v2.2 planning [X]

The v2.2 planning session (docs only) replaces this card with a full plan and prompt. It must:
- **Check each trigger** (entry gates) and keep only the rows whose trigger fired. 1d always moves out, to SD-course planning.
- **Re-read what exists by then:** the S6 results note (browser legs, SDP and ICE candidates), m6b-03's gate code and m6b-04's fake-media procedure.
- **Plan the Safari leg:** it needs the owner present. Book it as a short owner event, `ev-m6c-safari`, prepared by the expanded sprint; whichever prompt runs the leg lists the owner's presence under `## Before you launch (owner)` (D40). Order it **after** a patch that admits Safari for the cohort only (1a), because m6b-03's gate sends Safari to text on production today.
- **Record in the decisions log:** the Safari widening, whether voice-lite ships, and the canary's shape. None of these needs a new ADR unless it changes ADR-0032's §6 constraints beyond the "at launch" wording.

### 1a · Sketch: Safari leg → widen the voice gate [X · O]

Sources: [t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options) (browsers), [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) step 9 and M15, [ADR-0032 §6](../../adr/0032-realtime-ai-mock-interviewer.md#6-privacy-cost-and-limits) ("voice on Chrome/Edge only at launch"), PRD R-MI1.

**The order matters.** [m6b-03](sprint-m6b-03.md)'s capability and user-agent gate sends Safari to text mode on production, so the leg can't run there until Safari is admitted. The steps are:
1. **Admit Safari for the cohort only [X].** Widen the gate to Safari at or above a version floor for `role ∈ {owner, tester}` only. It is a T-1 cohort-audience code default, like [m6a-01](sprint-m6a-01.md)'s `interviewAudience`, with no T-2 env and so no infra PR. It ships dark in a `v2.1.x` patch, with gate-matrix tests. Everyone else still gets text.
2. **Run the leg [O + X]** on production (below).
3. **Flip at the M6c GA minor**, only if the leg passed (below).

If the leg fails, the next patch removes the cohort admission.

**The leg [O + X]** runs on production with the owner's account and key, and the owner present. Safari has no Chrome-style fake-media flags, so a real mic is needed. It takes about 15 minutes and covers the hard gates that can differ by browser:
- the SDP answer is accepted and ICE connects (M1);
- turn-taking through 45 s silences and push-to-talk (M3, M3b);
- barge-in (M4);
- captions;
- audio continuity across a rollover or re-attach (M7-like);
- a browser tamper attempt fails (M13).

The pass/fail note uses the S6 results format.

**If it passes [X]:**
- the M6c GA tag sprint flips the Safari admission from the cohort to everyone, keeping the version floor;
- AB29's browser copy ships as the variant frozen in the design sprint (entry gates). There is no copy change without that frozen variant;
- update PRD R-MI1 ("Chrome or Edge"). ADR-0032 §6 says Chrome/Edge "at launch", so the widening is a decisions-log line, not an in-place edit of the Accepted ADR (task 1);
- leave Permissions-Policy and CSP unchanged ([mi-13](sprint-mi-13.md)).

**If it fails:** record the result, remove the cohort admission, and Safari keeps the text interviewer with its explanation.

**Alternative (decide at expansion):** run the leg first in an S6-style local harness ([spk-04](sprint-spk-04.md)) against a local build that admits Safari. Production then gets only the cohort admission and the flip.

### 1b · Sketch: voice-lite, only on demand [X]

Sources: [t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options) option D, the voice-lite row in [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys), and the [t6 §14](../research/t6-realtime-interviewer.md#14-risks) risk "Anthropic-only learners get no voiced coach".

**Shape.**
- In: browser `SpeechRecognition`.
- **On-device only where possible** (`processLocally` plus a language pack). Otherwise the browser's cloud recognizer receives the audio. That must be disclosed with its own unticked consent, or refused. Decide at expansion.
- The recognized text is a turn on the **existing text-interviewer path** (`POST /api/interviews/{id}/turns`, bounded SSE), on any `interview` key, Anthropic included.
- Out: the reply is read by `speechSynthesis`. That's an OS voice, **not** the coach's voice, and the UI says so. Only voices with `SpeechSynthesisVoice.localService === true` are on-device. Chrome's default "Google …" voices are network voices that send the reply text to Google, so restrict to local voices, or disclose the network voice and take its own unticked consent. Decide at expansion, together with the recognizer.
- Gate: the T-3 cohort only until the M6c GA flip (a T-1 audience default, no T-2 env), listed in the flag inventory against the ≤ 6 budget ([m6c-01](sprint-m6c-01.md) task 1).
- Half duplex, ≈ 1.6–2.7 s per turn. Local VAD stops playback, and headphones are advised.

**Server:** nothing new. Open questions for the expansion:
- **`mock_session.format`:** `text`, or a new `voice_lite` value (an expand-only enum change in assessment)?
- **Communication:** it stays `self_only_voice`, because browser ASR is biased (see [m6c-02](sprint-m6c-02.md)).
- **Turn source:** do browser-transcribed turns count as `source=client`? That would force `scored_by=self` under [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) step 5.
- **EU gate:** no live speech model is involved, but browser ASR may still send audio to a cloud recognizer, and a network synthesis voice sends reply text to the browser vendor. Legal-review item.
- **Privacy notice:** any cloud recognizer or network voice joins [m6c-01](sprint-m6c-01.md)'s single batched notice-version bump at the M6c GA minor.

**Cost:** ≈ $0.5 / $0.7 per 45 / 60-minute rail. It's brain-only, on the learner's key.

### 1c · Sketch: optional weekly 30 s voice canary, read by looking [X · O]

Sources: [t6 §9](../research/t6-realtime-interviewer.md#9-phased-plan) P2 ("optional weekly 30 s canary on the owner's key (≈ $0.03)"), critic 1 #9 in t6 §15, and **D34** ([ADR-0035 §3](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#3-no-alerting-in-v2-owner-d34)).

**Purpose.** The voice APIs are young, and provider drift breaks the shell silently. A 30-second real session catches that before the owner's next real interview does.

**Recommended shape: owner-run, not scheduled.**
- `docs/v2/runbooks/voice-canary.md` plus a small script, placed by the expansion. It trims [m6b-04](sprint-m6b-04.md)'s manual fake-media run to 30 s: Chrome with `--use-fake-device-for-media-stream --use-file-for-fake-audio-capture` and the recorded clip, driven against production with the owner's account.
- **It drives only the AB29 pre-flight voice check**: [m6b-03](sprint-m6b-03.md)'s real 20–30 s segment in the FSM's `preflight` state (≈ $0.02). Then it **abandons the interview explicitly**. It never goes `live`.
- Within that check, it checks in order: the SDP broker round-trip, first audio, captions, the sideband transcript, a clean hang-up, and the segment's cost in the segment log.
- It prints PASS or FAIL. The owner writes the date and the result into status.md, like the evalpack PAT expiry check.

**What a run costs the owner's account.** A production voice check has to go through an interview, so each run:
- **uses 1 of his ≤ 2 starts that day** ([m6a-01](sprint-m6a-01.md)'s `setup → start` rule), and holds his one non-terminal slot until the explicit abandon. The runbook says: don't run it on a day with two real interviews planned, and check that the abandon went through;
- **creates one assessment mock row** at start ([m6a-03](sprint-m6a-03.md)), closed as `abandoned` by the abandon mirror. Abandoned rows never count in trends, aggregates or the public mock count (m6a-03's reader tests; D31), but they do show in his history, about 52 a year. The expansion decides whether that's acceptable, or whether to plan a **probe-only segment kind** that creates no interview and no mock;
- counts toward his L19 voice minutes. Cost ≈ $0.025 of shell time plus a probe, about $0.03 per run.

**Rejected: an in-cluster CronJob.**
- Coach has no WebRTC stack, and a server-side audio transport would put media on the node.
- It would add a pod under the memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#5-capacity-the-memory-sum-rule-triggers-and-ordered-responses)).
- It drifts toward the opscheck design D34 dropped.

Revisit this with alerting at the v3 opening. **No push, no alert, no Flux Alert** in any shape.

### 1d · Sketch: AB31 A6 Workspace-Canvas (Excalidraw) outline for the SD course [X]

Sources:
- [rollout §9](../rollout-plan.md#9-artboards-by-milestone) (the v2.2+ row) and [§8](../rollout-plan.md#8-what-ships-where-content-hours-the-d6-reading) (SD course = M4 + canvas AB31 + calibration ≥ 80 per rubric);
- [t4 §8](../research/t4-judge-contract.md#8-learner-flows-and-v2-artboards-needed) (A6 frames) and [t4 §4.2](../research/t4-judge-contract.md#42-archetype-b-structured-response-canvas--questionnaire--ai) (canvas storage, grading);
- **D19** (Excalidraw) and **D29** (the interviewer follows the canvas as a text export);
- PRD V4 and R-CV1.

**Frames to brief** (from t4 §8 A6):
- navigator;
- canvas;
- **Outline** view: the table editor for keyboard, screen-reader and mobile use;
- **"What the grader sees"** drawer: `canvas-export@1`, rendered client-side by the same algorithm judge uses;
- estimates: `blank.numeric` with units;
- pre-flight;
- progress;
- result: the key steps plus `ai_rubric` bands, with the provisional, dispute and re-grade patterns from frozen AB16.

For an SD **mock**, add a frame where the interviewer's "current screen" is the canvas text export (D29: structured state, never screenshots).

**Constraints the board must respect.**
- **Excalidraw** (D19): typed shapes via `customData`; freehand allowed but **ignored by grading**.
- **`canvas-graph@1`:** ≤ 512 KiB and ≤ 2,000 elements. The export is ≤ 32 KiB, and the canvas body cap is 640 KiB ([ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) L6).
- **Storage:** inline in Postgres (D11).
- **Loading:** a lazy-loaded chunk with the `vite:preloadError` reload ([ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service)).
- **Grading:** **never from a PNG**.
- **The primary action** is "Finish & submit for grading" after the pre-flight. There is no Run.

**Where.**
- File: `design-system/screens/v2/AB31-workspace-canvas.html`, on `theme.css` verbatim, following the board conventions of [ds-m1-01](sprint-ds-m1-01.md).
- It is drafted in a design sprint, e.g. `ds-sd-01`, at SD-course planning.
- The index (`design-system/screens/v2/index.html`) already marks AB31 "outline only", and ds-m1-01's rule says later design PRs **never edit** it. So that design sprint adds only the new board file. It links the board from **status.md's AB31 artboard row**, which also records the freeze (the merge on CI green is the freeze, D40). If SD-course planning wants the index row linked too, its own plan states an explicit exception to ds-m1-01's rule for that one row: by v2.2+ no parallel early design PRs remain.
- **Nothing is drafted in M6c.**

## Acceptance criteria

- [ ] n/a (outline). Nothing here is executable until the card is expanded.

Candidate acceptance to refine at expansion (not gates today):
- [ ] **Safari (if its trigger fired):** the cohort-only admission shipped before the leg; the leg's note is recorded. If it passed, the M6c GA flip admits Safari at or above the floor for everyone, the frozen AB29 copy variant and the PRD wording are updated, and the gate-matrix tests are green. If it failed, the cohort admission is removed. Firefox still gets text.
- [ ] **Voice-lite (if its trigger fired):**
  - works on an Anthropic-only key, with no new server route;
  - cloud-recognizer use is disclosed and consented (or refused);
  - only `localService` synthesis voices are used, or a network voice is disclosed and consented;
  - the "device voice" label is shown;
  - Communication stays `self_only_voice`.
- [ ] **Canary:** the runbook and script are merged. One run is recorded in status.md as PASS with its cost. It ran only the pre-flight voice check, and the interview was abandoned. Nothing is scheduled or pushed.
- [ ] **AB31:** the brief above is carried into the SD-course planning. No board is drafted in M6c.

## Release

**Outline only (v2.2).** This card merges nothing and cuts no tag.

Once expanded:
- **Safari widening and voice-lite:** "merge only (ships dark to the cohort in the next `v2.1.x` patch)", with the ADR-0034 §6 checklist on each patch, including **"from M6: no live interviews"**. Their default flips ride the **M6c GA minor**, cut by the dedicated M6c GA tag sprint that [m6c-01](sprint-m6c-01.md) task 1 adds (e.g. `m6c-04`). That tag sprint copies the [ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) release checklist plus the ADR-0035 §2 NetworkPolicy standing rule. No new in-cluster caller is expected.
- **The Safari leg:** an owner event with no merge. Its note lands through a docs PR.
- **The canary:** the runbook and script merge. Nothing deploys, and there's no tag.
- **AB31:** "land-and-sync; the merge is the design freeze" in the SD course's design sprint, outside M6c.
- **infra:** no infra PR, because no T-2 env is added.

## Definition of Done

**This card:**
- it exists with its sources linked;
- it is listed on the [`../status.md`](../status.md) sprint board as "⬜ outline";
- AB31 stays "outline only" in the artboard table;
- no code and no prompt.

**At expansion:**
- replaced by a full plan and `../prompts/prompt-m6c-03.md`, containing only the rows whose triggers fired;
- `ev-m6c-safari` booked if 1a is kept;
- the AB31 brief handed to SD-course planning.

## Risks / watch-outs

- **Drifting into alerting (D34).** A "weekly" canary invites a scheduler and a notification. Keep it owner-run and recorded by hand. Any scheduling or push reopens D34, which is a v3 opening decision ([rollout §11](../rollout-plan.md#11-opening-gates-v3)).
- **Voice-lite privacy.** Browser speech recognition may send audio to the browser vendor's cloud, and a network synthesis voice (`localService === false`, e.g. Chrome's "Google …" voices) sends the reply text there. Either one is a new data processor the privacy notice doesn't name. Require on-device recognition and local voices, or disclose them and take consent. Never assume.
- **The canary uses a real interview slot.** Each run takes one of the owner's two daily starts and leaves an abandoned mock row in his history. Keep it to the pre-flight check, abandon at once, and consider a probe-only segment kind at expansion.
- **Safari passes the leg but regresses later.** Keep the version floor and the text fallback. The canary runs on Chrome only, so a Safari regression shows up only when someone uses it.
- **Canary costs and caps.** Each run is billed to the owner's key and counts toward his L19 minutes. Once a week is plenty.
- **AB31 scope creep.** The canvas board belongs to the SD course, which needs M4, the canvas widget and ≥ 80 labelled items per rubric. Don't draft it early: the brief ages less than a board would.
