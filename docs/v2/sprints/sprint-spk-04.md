# Sprint spk-04 — S6 voice-shell bake-off (owner present, throwaway)

> **Milestone:** M6a — spike **S6**, which gates the M6a design freeze · **Track:** spike · **Kind:** spike · **Order:** 71
> **Prereqs:** none in `depends_on` · launching this prompt is the owner's go-ahead **O1** (D40); the owner's presence for the day (event `ev-s6`), the two OpenAI projects and their keys are before-launch items · soft: [m1-10](sprint-m1-10.md) (the catalog's provisional `voice_shell` entries this spike confirms or corrects)
> **Unblocks:** [ds-m6a-01](sprint-ds-m6a-01.md) (accepts [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) from this result, then drafts AB13/AB24/AB25) · [m6a-02](sprint-m6a-02.md) (the scrubbed replay fixtures) · [ds-m6b-01](sprint-ds-m6b-01.md) (browser gate list, chosen shell, M7 deploy shape) · [mi-13](sprint-mi-13.md) (shell, M8 coach memory, the CSP/Permissions-Policy check) · consumed later by [m6b-01](sprint-m6b-01.md) (adapter, SDP shapes, sideband event types) and [m6b-02](sprint-m6b-02.md) (rollover, M7, cost events, push-to-talk)
> **Release action:** **no merge (spike, throwaway).** The harness, the raw logs, the fake-capture audio and the OpenAI projects are never committed and are deleted at the end. Only the results note, the **scrubbed** fixtures and the status rows land, through one docs PR squash-merged on CI green with no owner review stop (D40)
> **Calendar:** event **`ev-s6`**: any day the owner is present, **before the M6a design freeze**. Recommended after M3 and within ~4 weeks before [ds-m6a-01](sprint-ds-m6a-01.md) (≈ Dec 2026 – Jan 2027), because the voice APIs were weeks old at planning time and the result ages. **Time box:** 1 working day, ≤ 8 h hands-on plus a 65-minute unattended soak. **Spend:** `xlearn-s6` has a $10 hard limit (enforced); the harness stops itself at $8, with per-leg checkpoints (GPT-Live legs ≤ $5.50, leaving ≥ $2.50 for mini and the browsers). `xlearn-s6-quota` starts at $1 and is raised in $1 steps to at most $3. Worst case across both projects: $11. t6 expects ≈ $6–7
> **Execute with:** [`../prompts/prompt-spk-04.md`](../prompts/prompt-spk-04.md) — one prompt, one session.

## D41 changes (read first; they override the text below where they conflict)

> **D41 (owner, 2026-09-25): spikes first.** All four spikes run **before any build sprint**, so every design yes/no is answered before M1 starts: spk-01 and spk-02 on Fri 2026-09-25 (agent-only, after the MI-0 reboot), spk-03 and spk-04 on Sat 2026-09-26 (spk-03 after the owner's Console step; spk-04 with the owner present). For this sprint, overriding the text below:
> - **Run on Sat 2026-09-26, with the owner present** (`ev-s6`). The "recommended after M3" timing is superseded; the owner accepted that results age.
> - **A ≤ 1 h recheck runs before ds-m6a-01** (event `ev-s6-recheck`, non-blocking for everything before M6a): the same shells, the hard gates and M14 only, and a note if the winner or its price changed.
> - **Sole-passer question, answered by the owner on 2026-09-25: yes.** If GPT-Live passes every hard gate and mini fails one, GPT-Live wins even when its M14 isn't a full point higher (a shell that fails a hard gate isn't viable).

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Accounts: two throwaway OpenAI projects ($10 and $1 hard limits), project-scoped keys | O (before launch) | ⬜ |
| 2 | Harness (1.5 h): SDP brokers, sideband decoder, stub director, JSONL log, proxy, static page | H | ⬜ |
| 3 | GPT-Live runs: scripted mock, mid-call restart, tamper, 65-min soak, quota, reseed (t6 §10 steps 2–7) | O · H | ⬜ |
| 4 | mini runs: steps 2 (10 min), 3, 4, 6, 7, plus native push-to-talk, `semantic_vad` low, cache ratio (step 8) | O · H | ⬜ |
| 5 | Browser legs, answer-SDP inspection, sideband filter, CSP/Permissions-Policy check (step 9) | O · H | ⬜ |
| 6 | Decide: apply the D28 rules; fix the deploy shape (M7) and the browser gate list | X | ⬜ |
| 7 | Deliver: scrub the fixtures, results note in t6 §16, status rows (docs PR) | X | ⬜ |
| 8 | Teardown: projects and keys deleted, raw logs and harness directory deleted | O · H | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (say why).
> A failed hard gate is still ✅ once it's recorded with numbers; the verdict goes in the results. Update the _Overall_ line to
> match, and mirror the sprint's state into [`../status.md`](../status.md) (the Sprint board row, the spike-results row and the
> `ev-s6` event). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] **The go-ahead O1 is this launch on the day** (D40). D28 approved S6 as specified; launching the prompt is the go-ahead the day still needs, and **no agent uses a key the owner didn't set before launch** ([t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike)).
- [ ] **The owner is present for the day** (a before-launch item, event `ev-s6`, ≤ 1 working day): he plays the candidate, rates naturalness and continuity, reads the usage page, raises the quota project's limit at task 1's steps and deletes the projects at the end. The projects and keys exist before launch (task 1).
- [ ] **The owner's OpenAI organisation is usage Tier 1 or higher** (GPT-Live has no free tier) and has 2FA on (MI-1, event `ev-mi1`), both stated in the launch message. Record the tier; 2FA missing is noted, not a blocker.
- [ ] **The owner's Mac:** Chrome stable (Edge optional), Firefox and Safari installed; Go ≥ 1.26 (the repo's `go.mod` line); a home network in India (record wired/Wi-Fi). Headphones are available but **not** used for M3.
- [ ] **Nothing touches the VPS or the cluster.** No `ssh sujaykumar-vps`, no `kubectl`: the laptop's kube context tunnels to production.
- [ ] **Parallel sessions:** no open peer PR edits [`../research/t6-realtime-interviewer.md`](../research/t6-realtime-interviewer.md) or adds `docs/v2/research/t6-s6-fixtures/` (`gh pr list --state open`, `git worktree list`, ListAgents).

## Goal

Answer, on the owner's own OpenAI account and inside our architecture (coach-style SDP brokering, a coach-held sideband and
a text director brain on the same key), the seven questions [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike)
asks of each candidate shell, **GPT-Live-1** and **`gpt-realtime-2.1-mini`**:
1. Does it hold 60 minutes, or reseed cleanly?
2. Does it stay quiet through think-aloud, typing and 45-second silences on laptop speakers, and support push-to-talk?
3. Does it answer substantive turns fast enough, without bluffing about code?
4. Does it surface a classifiable quota signal?
5. Does it survive a server restart without dropping audio (M7, which decides the deploy shape)?
6. Does its cost land within ±25% of [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys)?
7. Can the browser alter the session?

Plus the two measurements D28 added from D29 ([t6 §13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict)):
the cost and latency of **screen-context updates at a 2–3 s change cadence**, and whether the **"current screen" item can be
replaced in place** on each shell.

Then apply the owner's decision rule (D28): **GPT-Live-1 only if it passes every hard gate and the owner rates it ≥ 1 point more
natural than mini; otherwise mini; both fail → text only, voice revisited in 3 months**. The result is the input
[ds-m6a-01](sprint-ds-m6a-01.md) needs to accept ADR-0032 and freeze the M6a boards.

## Scope

**In**
- Steps 1–10 of [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike), in its time box, on a localhost harness.
- Measurements **M1–M15** from t6 §10, plus **M16–M17** (D29 screen context), with the decision rules.
- A CSP/Permissions-Policy leg for [mi-13](sprint-mi-13.md): the harness page runs under the gateway's shipped CSP string, verbatim from [m1-04](sprint-m1-04.md), plus the `Permissions-Policy` that mi-13 will add.
- Re-verifying the model pages, prices, limits and the sideband docs **on the day**, and recording every delta from t6 §2/§3/§8.
- Recording the results in `t6-realtime-interviewer.md` §16 and committing **scrubbed** fixtures to `docs/v2/research/t6-s6-fixtures/`.

**Out**
- **The VPS or the cluster** in any form. Nothing here is deployed; there is no infra PR.
- **Committing the harness**, the raw JSONL logs, any audio file or any SDP. The repo is public.
- **Accepting ADR-0032** → [ds-m6a-01](sprint-ds-m6a-01.md) task 1 (BP2). This sprint leaves every ADR untouched.
- **The catalog code** (`internal/coach/catalog.go`): this sprint only records the facts; [m6b-01](sprint-m6b-01.md) confirms the `voice_shell` entries in code.
- **The gateway CSP, Permissions-Policy and 20 s SDP route** → [mi-13](sprint-mi-13.md); **the shell adapter, SDP broker and sideband** → [m6b-01](sprint-m6b-01.md); **rollover, lease and drain** → [m6b-02](sprint-m6b-02.md).
- **The twin fairness gate** → [m6a-03](sprint-m6a-03.md); **the multi-speaker fairness check** → [m6c-02](sprint-m6c-02.md); **a Safari fix** → [m6c-03](sprint-m6c-03.md).
- Voice-lite, Gemini Live and `gpt-realtime-2.1` (catalog-only): not measured ([t6 §2](../research/t6-realtime-interviewer.md#2-model--pipeline-options)).

## Tasks

**Shared setup.**
- **Work directory:** `<scratchpad>/s6/`, **outside every git repo**, never committed. It holds the Go harness (its own
  throwaway module), the static page, the fake-capture WAV, the Chrome test profile and the raw logs. Task 8 deletes it.
- **The key** lives only in environment variables the owner sets before launch, in the shell the session starts from
  (`read -s OPENAI_KEY_S6`, `read -s OPENAI_KEY_S6_QUOTA`, then `export`; no echo, not in shell history), so the harness
  inherits them. The agent never prints, reads back, logs or writes it. The harness never logs request headers.
- **Prices:** the harness tallies spend on `xlearn-s6` from usage events, at the prices on the model pages **that day** (entered
  as constants). It hangs up every session and refuses new ones at **$8**.
- **Leg budget on `xlearn-s6`.** The harness tallies per shell and enforces two checkpoints:
  - **GPT-Live legs ≤ $5.50** (task 3 steps 2–5 and 7, director included). At $0.05/min that is ≈ 100 connected minutes plus
    the director, and the soak alone is 65 of them. Task 3 gives the per-step caps. Keep the director **off during the soak**.
  - **≥ $2.50 stays for mini and the browsers.** t6 §8 puts mini at ≈ $2.1 per 45 minutes with the director. Mini's legs run
    before the Firefox/Safari legs.

  At a checkpoint, that shell's remaining legs stop and their rows read "not run (spend)". Print the running per-leg total
  after every leg.
- **Key-shaped pattern.** Every key check below uses `KEYRE='\bsk-(proj-|svcacct-|admin-)?[A-Za-z0-9_-]{20,}'` with `rg`, never a
  bare `sk-`: the repo already holds 60+ harmless `sk-` strings (test fakes, v1 board placeholders, and words like `risk-`).

### 1 · Accounts [O, before launch]

- **Before launch** (D40: dashboard work is owner-only), the owner creates, in the OpenAI dashboard, a **throwaway project
  `xlearn-s6`** with a **project hard spend limit of $10** (enforced, not an alert) and a second throwaway project
  **`xlearn-s6-quota`** with a **$1 hard limit** for step 6.
- **The quota project's limit, step by step** (it's the only brake on that project, since the harness self-stop covers `xlearn-s6`):
  - $1 for GPT-Live's quota leg, then raised to **$2** to time recovery;
  - mini's quota leg drives it to failure at $2, then it is raised to **$3** to time mini's recovery;
  - never higher. The owner reads the project's spend before each raise, and the note records each figure.
- One **project-scoped key** per project, restricted to the realtime/live and Responses APIs where the UI allows, with the
  shortest expiry offered. The owner sets them as environment variables before launch; nothing else holds them.
- Record only non-secret facts, from the launch message: the org's usage tier, which key restrictions the UI offered (they
  feed AB24's key-hygiene copy), and the project names. The agent never logs into the dashboard.
- **The owner answers one question in the launch message, and the session records the answer in the note.** "If GPT-Live
  passes every hard gate and mini fails one, does GPT-Live win even when M14 < mini + 1? That is, does M14 only decide between
  two passers?" D28 reads literally "otherwise mini" and doesn't cover this case. An answer given before the runs puts the
  case inside the rules (task 6); without one, the case follows task 6's "not answered" path. The session doesn't ask
  mid-run.

### 2 · Harness (1.5 h) [H]

A localhost Go program (`go mod init s6harness`; `github.com/coder/websocket`, ISC, the library ADR-0032 names) plus one static
page, mirroring the planned coach/gateway path ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path)):

| Part | What it does |
|---|---|
| **Server** | `http.Server` with the timeouts of `internal/platform/httpx.NewServer` copied in (ReadHeader 10 s, Read 30 s, Write 60 s, Idle 120 s), listening on `127.0.0.1` only. The 60 s WriteTimeout would cut a long SSE stream, so the caption stream is a **bounded SSE of ≤ 50 s** that the page reconnects with `Last-Event-ID`. That is the shape the interview HUD will use ([t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path), [m6a-04](sprint-m6a-04.md)) |
| **SDP brokers** | `POST /segments?shell=live\|mini` takes the browser's offer (≤ 16 KiB) and posts it upstream with the key, under its own **20 s** budget (the route [mi-13](sprint-mi-13.md) ships). GPT-Live → `POST /v1/live/sessions` (JSON `session` + `transport{webrtc, sdp}`, with **`store:false`** in the body, per [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility)). mini → `POST /v1/realtime/calls` (multipart). Returns the answer SDP. Records round-trip ms (M1). **The SDP is never logged** (its ICE candidates carry IP addresses) |
| **Orphan reaper** | hangs up any session whose browser hasn't confirmed within 15 s |
| **Sideband** | one outbound WSS per session with the standard key. A **streaming type-filtering decoder** reads each event's `type` and **discards audio events without unmarshalling their payload** (`session.input_audio.append`, `session.output_audio.delta` and the Realtime equivalents); only their count and byte size are logged. Records bytes/s, harness RSS and decoder CPU (M8) |
| **Stub director** | `gpt-6-sol` via the Responses API with **`store:false`** on the same key. Triggers: a substantive candidate turn, a code change (sampled every 2–3 s, only when changed, per D29), a run echo, a phase boundary. Emits a ≤ 500-token note: GPT-Live `session.thinking.append` (context) or `session.commentary.append` (speak); mini `conversation.item.create`. Maintains **one replaceable "current screen" item** (M17). A switch turns it **off for the soak** (spend) |
| **Controls** | push-to-talk (`session.input_audio.mute`/unmute on GPT-Live; `turn_detection:null` + manual commit on mini), hang-up, reseed (a new session primed with a brief plus the last 4 turns, ≤ 8,192 tokens), a synthetic run echo |
| **JSONL log** | per event: relative `t_ms`, channel (`http`, `sideband`, `datachannel`), direction, `type`, sizes, `error.code`/`error.type`, usage numbers, `usage_ratio`, close reasons. **Audio dropped, SDP never written, headers never written.** Transcript text is allowed in the raw local log only (task 7 replaces it) |
| **$ tally** | GPT-Live: connected seconds × the per-minute price; mini: usage tokens × token prices; plus director tokens. Tallied per shell: the GPT-Live checkpoint is **$5.50**, the self-stop **$8** |
| **Static page** | served by the harness with the gateway's **shipped CSP, verbatim**: read it from `internal/gateway/security.go` on `main` (m1-04's string: `default-src 'self'; script-src 'self'; style-src 'self' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self' https://github.com`; if `main` differs, use `main`'s and record the delta). Add **`Permissions-Policy: camera=(self), microphone=(self)`**. No inline `<script>`, `<style>` or `style=""`: JS and CSS are served as files. `getUserMedia({audio})`, an `RTCPeerConnection` (audio only; GPT-Live with its `oai-events` data channel, mini **without** a data channel), the offer posted to the harness, captions (data channel on GPT-Live; harness SSE on mini), a PTT button, a code textarea whose changes post to the harness at most every 2 s, a "run" button, `getStats()` inbound-RTP counters for M7 |

**Pass:** both shells connect through the broker from the page under that CSP, and a 60 s smoke on each shows captions, a
director note and a clean hang-up. Check the raw log with `rg -c "$KEYRE|v=0|a=candidate|a=fingerprint"`. The expected result is
no output. Use counts only, never print matches.

### 3 · GPT-Live runs [O · H]

Steps 2–7 of [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike), the owner as candidate, on the `xlearn-s6` project:

| Step | What | Feeds |
|---|---|---|
| 2 · Scripted 15-min DSA mock (1 h) | 30 DSA terms; 3 × 45 s silences; **2 min of typing while thinking aloud without headphones**; 2 barge-ins; 3 min of **emulated push-to-talk** (`track.enabled` + sideband mute/unmute); 5 code questions after director pushes; 1 run echo; 3 latency probes in each of **push-only**, **client delegation** and **Responses delegation**; a 5-minute typing segment with the D29 2–3 s screen updates on | M2, M3, M3b, M4, M11, M12, M14, M16, M17 |
| 3 · Mid-call restart (15 min) | attach a **second sideband before** killing the first harness process (make-before-break), then a **cold re-attach** by session id; the owner says whether audio continued; `getStats()` packets keep rising | **M7** |
| 4 · Tamper (10 min) | from DevTools, send `session.update` and instruction events over the GPT-Live data channel; record what the server accepts | M13 |
| 5 · 65-min soak (unattended) | a separate Chrome profile with `--use-fake-device-for-media-stream --use-file-for-fake-audio-capture=<s6>/prompt.wav` looping a synthetic prompt every 2 min (`say -o prompt.aiff …` then `afconvert` to 16-bit WAV: no one's real voice); **director off** (the director path is measured in step 2); record the close reason and time, `usage_ratio` at 60 min, sideband bytes/s, harness memory and CPU | M5, M8 |
| 6 · Quota (30 min) | on `xlearn-s6-quota` ($1): drive a live session to failure; capture the first failing events on the sideband and the data channel, the error for creating a new session, and the overshoot; **record whether the active WebRTC session is killed or keeps running**; the owner raises the limit **to $2** (task 1) and the harness times purchase → probe success; **label the fixtures "spend-limit path"** | **M6**, m6a-02's classifier |
| 7 · Reseed (20 min) | a new session seeded with the brief plus the last 4 turns (≤ 8,192 tokens); measure TTFA; the owner rates continuity 1–5 | M5 (reseed arm), M10 |

**Spend checkpoint: GPT-Live ≤ $5.50 on `xlearn-s6`.** The soak alone costs ≈ $3.25 (65 min × $0.05). That leaves ≈ $2.25 for
everything else: about 37 connected minutes plus the director. So:
- hang up between script segments;
- cap connected time at about 15 minutes for step 2, 8 for step 3, 5 for step 4 and 8 for step 7;
- run the soak **last** of GPT-Live's `xlearn-s6` legs (over lunch), and only while ≥ $3.25 of the budget remains.

Record the running total after each leg. A leg that would take the total past the checkpoint becomes "not run (spend)". Step 6
bills `xlearn-s6-quota` and doesn't count here.

### 4 · mini runs [O · H]

Step 8 of t6 §10 on `gpt-realtime-2.1-mini` (1.5 h, from the ≥ $2.50 kept back): steps 2 (10 minutes, same script), 3, 4, 6 and 7, plus:
- **native push-to-talk** with `turn_detection:null` (M3b), and `semantic_vad` eagerness `low` for the silence and typing legs (M3);
- the **cached-token ratio** before and after a 15-minute silence (cost volatility, [t6 §8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys));
- tamper (M13): an offer with **no data channel** is accepted; an `m=video` section is refused;
- the D29 screen-context leg: replacing the current-screen item with `conversation.item.delete` + re-create (M17), cost per minute (M16);
- the quota leg (step 6) on `xlearn-s6-quota`, now limited at $2: drive to failure, then the owner raises it **to $3** and the
  harness times recovery.

**mini's M5 is judged without a soak**, because t6 §10 step 8 deliberately leaves the soak out:
- **The 60-minute arm:** the documented 60-minute Realtime cap and its `expired` close, re-read on the model page that day and
  recorded with the page and date.
- **The reseed arm:** step 7's measured TTFA ≤ 3 s.

mini passes M5 when the documented cap is ≥ 60 minutes and step 7's reseed TTFA is ≤ 3 s. Rule 2 already plans a rollover at the
start of Code, so production never relies on reaching the cap. Record the row as "cap documented (page, date), not soaked by
design + reseed TTFA <n> s". It is a measured result, not an unmeasured gate. If the API offers a shorter session lifetime and the
box and budget allow, one short forced-expiry session showing the actual close event is a bonus, not a requirement.

### 5 · Browser legs, SDP, CSP [O · H]

Step 9 of t6 §10 (20 min):
- **Firefox and Safari, 5 minutes each**, on the winning-so-far shell (both shells if time allows). Record whether each holds
  past two turns (openai-agents-js #1353). M15 decides the browser gate list.
- **Answer SDP:** inspect (in memory, never saved) for ICE-TCP / TLS candidates, which matter for UDP-blocked learner networks.
  Record yes/no per shell.
- **Sideband filter:** look for any documented or accepted option that stops audio copies reaching the sideband; use it if it exists.
- **CSP/Permissions-Policy check for [mi-13](sprint-mi-13.md):** the page ran every leg under the task 2 headers. Record any
  `securitypolicyviolation` event (the page listens and posts them to the harness) and confirm that **the page never requested a
  provider origin** (DevTools Network, filtered on `openai.com`: only the WebRTC media, no fetch/XHR/WebSocket). A page-side
  provider request contradicts [ADR-0032 §2](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture): record it as a blocker.

### 6 · Decide [X]

Fill the measurement table, then apply the rules from [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) (D28):

| Measurement | GO threshold | Hard? |
|---|---|---|
| M1 SDP broker round-trip p95 | ≤ 3 s | H |
| M2 substantive-answer latency p95, push-only | ≤ 2.0 s (delegated ≤ 3.5 s is soft; if it fails, delegation isn't used) | H |
| M3 false interjections during silences and typing-while-thinking-aloud, no headphones | ≤ 1 per 10 min | H |
| M3b push-to-talk: speech while muted; response after release | 0; ≤ 2.5 s | H |
| M4 barge-in stop time | ≤ 500 ms | S |
| M5 session duration | ≥ 60 min, or a clean `expired` close plus a reseed with TTFA ≤ 3 s | H |
| M6 quota signal (spend-limit path) | a classifiable code at the harness within 5 s; session closable; fixtures captured | H |
| M7 make-before-break or cold re-attach, audio uninterrupted | pass → one coach Deployment; fail → a `coach-interview` Deployment in P1 | H (deploy shape) |
| M8 sideband load | ≤ 150 KB/s per session; harness ≤ 100 MiB; decoder CPU recorded | S |
| M9 cost from usage events | within ±25% of t6 §8 | H |
| M10 reseed continuity | owner rating ≥ 4/5 | S |
| M11 DSA-term ASR fidelity | ≥ 90% (below 80% → timeline evidence dropped in voice mode) | S |
| M12 code-question bluffing | 0 wrong claims in 5 probes | H |
| M13 browser can't alter the session | GPT-Live rejects browser `session.update`; mini works with no data channel, or re-applying the config works | H |
| M14 naturalness | owner rating 1–5 per shell, same script | S (decides between two passers) |
| M15 browsers | Chrome/Edge pass; Firefox and Safari results decide the gate list | S |
| **M16** screen-context cost and latency at a 2–3 s change cadence (D29) | recorded: $/min added, effect on M2 | S |
| **M17** "current screen" item replaceable in place (D29) | recorded per shell: yes / no / workaround | S |

**Rows t6 §10 leaves unmeasured by design.** These rows are judged as stated here. They are not "unmeasured hard gates", and
recording them this way is a complete result:
- **mini M5:** judged on the documented 60-minute cap plus step 7's reseed TTFA (task 4); there is no mini soak.
- **mini M8:** not run on this shell (it's soft, and the soak is GPT-Live only).
- **M15 on the shell that isn't winning so far:** not run unless time allowed (it's soft).

Rows cut by the **time box or spend** ("not run (time box)", "not run (spend)") are different: they never count as a pass.

**Rules:**
- **GPT-Live passes every H and M14 ≥ mini + 1** → GPT-Live-1 is the default shell.
- **Otherwise, mini passes every H** → `gpt-realtime-2.1-mini` is the default, with a planned rollover at the start of Code.
- **GPT-Live is the sole passer** (every H passes, mini fails an H, but M14 < mini + 1): decided by the owner's answer to the
  task 1 question in the launch message:
  - he said yes (M14 only decides between two passers) → GPT-Live-1 is the default;
  - he said no → no shell qualifies, and the "both fail" rule applies;
  - he didn't answer → outside the rules: record the options with the recommendation "GPT-Live-1, the only shell that passed
    every hard gate", mark ds-m6a-01's ADR-0032 gate ⛔ "needs owner decision" in `status.md`, and still land the results.
- **Both fail** → P0 text only; voice is revisited in 3 months (record the date). M6b and its sprints go ⛔ until then; the owner decides what v2.1.0 carries (⛔ "needs owner decision").
- **M7 fails** (for the chosen shell) → [m6b-02](sprint-m6b-02.md) adds the `coach-interview` Deployment (same image, `COACH_ROLE=interview`, own ImagePolicy).
- **The browser gate list:** Chrome/Edge, plus Safari only if its leg passed; Firefox gets text mode with an explanation.

A hard gate cut by the time box or spend **disqualifies GPT-Live** under rule 1: D28 needs it to pass every hard gate. The
following results are **not decided here** (they fall outside every pre-decided path): record the options and a
recommendation in the note, mark ds-m6a-01's ADR-0032 gate ⛔ "needs owner decision" in `status.md`, and still land the
results PR (D40):
- a cut hard gate that leaves mini's pass, or "both fail", resting on an unmeasured row;
- the sole-passer case without an owner answer in the launch message;
- the spend over $10, or the quota project over $3;
- a broken architecture assumption, such as a page-side provider request;
- SDP brokering impossible on both shells.

ds-m6a-01 won't accept ADR-0032 on an escalated result.

### 7 · Deliver [X] — docs PR `docs(v2): S6 voice-shell bake-off results (spk-04)`

**Scrub first** (the repo is public; t6 §10 "scrubbed fixtures before anything is committed"). In the harness directory, a
throwaway script turns the raw JSONL into fixtures:
- every provider id (session, call, item, response, event and request ids) → deterministic synthetic ids (`s6-live-sess-1`, `s6-evt-000123`);
- **no SDP at all**; no headers; no org or project ids; no IP addresses, emails, names or places; error messages scrubbed of ids;
- every transcript text or delta → a placeholder (`<candidate turn 7>`, `<interviewer turn 7>`) with its length kept as `len`;
- **every request-body content field** → a placeholder with its `len`: the director's `input` and `instructions`, message
  `content`, the live-session `session.instructions`, reseed briefs and screen-context items. These carry the owner's words and
  code. Keep only the keys, `model`, `store:false` and the event/item types;
- kept: `type`, relative `t_ms`, channel, direction, `error.code`/`error.type`, status, usage numbers, `usage_ratio`, close reasons.

Commit the fixtures under **`docs/v2/research/t6-s6-fixtures/`**:
- **`index.json`**, one entry per file. Each entry has shell, scenario, path label, date, model ids and API version notes, plus the
  classifier handoff [m6a-02](sprint-m6a-02.md)'s test reads:
  - `observed[]`: channel, `type`, `error.code`/`error.type` and status of each failing or closing event;
  - `expected_class`: m6a-02's classes `Quota`, `Auth`, `ModelAccess`, `RateLimited`, `SessionCap`, `Unavailable`, `Unknown`,
    or `none` for a fixture with no error;
  - `expected_action`: from m6a-02's policy table, e.g. `interrupt(quota)`, `free reseed`, or `none`.

  Fill these from [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes)'s classifier
  table applied to the observed codes. m6a-02 owns the final mapping and may correct an entry, saying so in its PR.
- **One JSONL per scenario**, at least `live-quota-spend-limit.jsonl`, `mini-quota-spend-limit.jsonl`,
  `live-session-lifecycle.jsonl`, `mini-session-lifecycle.jsonl`, `live-soak-close.jsonl`, `live-tamper.jsonl`,
  `mini-no-datachannel.jsonl`, `probe.jsonl` and `director-store-false.jsonl` (the director's request shape showing
  `store:false`, content fields scrubbed as above), plus a rate-limit or `expired` fixture if one was seen.

Before staging, run
`rg -n "$KEYRE"'|sess_|rtc_|org-|proj_|v=0|a=candidate|a=fingerprint|ice-ufrag|([0-9]{1,3}\.){3}[0-9]{1,3}|@' docs/v2/research/t6-s6-fixtures/`.
It must return nothing, and **the agent skims one fixture per scenario** before staging. The merge doesn't wait on an owner
review (D40); the owner may look afterwards.

**Results note:** append **`## 16. S6 results (spk-04, <date>)`** to [`../research/t6-realtime-interviewer.md`](../research/t6-realtime-interviewer.md)
(one line in its top status blockquote points at it). About one page:
- **environment:** date, Mac and Chrome versions, network, the org tier, the model ids and prices on the day with `as_of`, and every delta from t6 §2/§3/§8;
- **the measurement table** (M1–M17 per shell: number or task 6 label, threshold, pass/fail);
- **the decision:** shell, deploy shape (M7), browser gate list, and whether client delegation is used, with the owner's
  launch-message answer to the task 1 sole-passer question (or "not answered");
- **for [m6b-01](sprint-m6b-01.md):** the SDP request/response shapes (endpoint and field names only), whether GPT-Live needs its data channel and mini works without one, whether brokering worked on both (else the `client_secrets` fallback is needed), the exact audio event types the decoder drops, and whether a sideband filter exists;
- **for [m6b-02](sprint-m6b-02.md):** the rollover rule (GPT-Live's duration limit if found, `usage_ratio` at 60 min), reseed TTFA, push-to-talk mechanics (native vs emulated), the cache ratio after silence;
- **for [m6a-02](sprint-m6a-02.md):** the quota path (codes, where they arrived, detection latency, whether the live session died, the overshoot, time to recover after raising the limit), the probe's behaviour, and the fixture index;
- **for [ds-m6a-01](sprint-ds-m6a-01.md) / AB24 and [ds-m6b-01](sprint-ds-m6b-01.md) / AB29:** measured cost per 45/60-minute interview on the winner, the D29 cadence cost, the text-brain cost, the key restrictions the dashboard offers, and ICE-TCP/TLS presence;
- **for [mi-13](sprint-mi-13.md):** M8 (sideband KB/s, harness RSS, decoder CPU → coach 500m / 256 Mi confirmed or revised) and the CSP/Permissions-Policy result (no violations; no page-side provider request);
- **for the catalog** (`voice_shell` entries from [m1-10](sprint-m1-10.md)): model ids, `billing_shape`, `session_cap_s`, `min_tier`, voices (is `marin` available?), `billing_url`;
- **spend:** per leg and per project (`xlearn-s6` against the $8 self-stop, the $5.50 GPT-Live checkpoint and the $10 limit;
  `xlearn-s6-quota` at each $1 step); **teardown date**; step 10 (passive prepaid timing) marked *pending: the owner appends the purchase → probe time after his next real top-up*.

**Leave ADR-0032 untouched** (it stays Proposed; [ds-m6a-01](sprint-ds-m6a-01.md) task 1 accepts it). **`docs/v2/status.md`:** the
Sprint board row, the spike-results row (spk-04: shell, deploy shape, date, link to t6 §16), `ev-s6` ✅ with the date, and
Decisions log lines (the shell; the deploy shape; the browser list; any ⛔ "needs owner decision" with its options).

### 8 · Teardown [O · H]

- The owner, present for the day, deletes both OpenAI projects and their keys (this also deletes their logs). Don't hold the
  docs PR for his confirmation: if it hasn't come by the time you ship, record it ⛔ in `status.md` → Open owner items.
- **Key check, before the harness directory is deleted.** It uses the key-shaped `KEYRE` and is scoped to what S6 could have touched; a
  repo-wide `git grep 'sk-'` is noise, with 60+ harmless hits on `main`.
  - **What the docs PR adds:** `git diff origin/main...HEAD | rg -n "^\+.*$KEYRE"` → nothing.
  - **The harness directory and the shell history:** `rg -c "$KEYRE" <scratchpad>/s6 ~/.zsh_history`. Counts only, never print
    matches: the history may hold unrelated keys. A non-zero count goes to `status.md` → Open owner items for the owner to
    inspect and clean (don't wait).
- Then the agent deletes `<scratchpad>/s6/` (harness, raw logs, WAV, Chrome profile).

## Acceptance criteria

- [ ] Every M1–M17 row is recorded **per shell**. Each entry has a number, or one of the task 6 labels for rows t6 §10 leaves unmeasured by design: mini M5 judged on the documented cap plus the reseed TTFA, mini M8 "not run on this shell", and M15 on the shell that wasn't winning so far unless time allowed. Rows cut by the box or spend are marked "not run (time box / spend)". The decision (shell, deploy shape, browser gate list) follows the task 6 rules or is explicitly escalated.
- [ ] `xlearn-s6` stayed ≤ $8 (the harness self-stop; hard limit $10). The GPT-Live checkpoint of $5.50 held, or its cut legs are marked. `xlearn-s6-quota` stayed ≤ $3, raised only in the task 1 steps. Per-project and per-leg totals are recorded.
- [ ] The harness page ran under the gateway's shipped CSP (m1-04's string verbatim, including `connect-src 'self'`) and `camera=(self), microphone=(self)` with no violation and no page-side provider request, recorded for mi-13.
- [ ] Only the scrubbed fixtures (`index.json` with `observed`/`expected_class`/`expected_action`), the t6 §16 note and the status rows are committed, through one merged docs PR. The scrub `rg` is empty and the agent skimmed one fixture per scenario.
- [ ] Both OpenAI projects and keys are deleted by the owner, or the deletion is recorded ⛔ as his follow-up. The scoped key checks came back clean: the PR diff and the `rg -c` of the harness directory and shell history. The harness directory, raw logs and audio are gone.

## Release

**No merge: a throwaway spike.** The harness, logs, audio and projects are never committed and are deleted. The only merge is
the **docs PR** (t6 §16, `docs/v2/research/t6-s6-fixtures/`, `status.md`), squash-merged on CI green with no owner review
stop (D40); it ships in no tag. An escalated result still lands, marked ⛔ "needs owner decision". No infra PR; nothing
touches production.

## Definition of Done

Results recorded and the docs PR merged · no production change (no `ssh sujaykumar-vps`, no `kubectl`) · statuses updated here and in
[`../status.md`](../status.md) · raw logs and the harness gone; projects and keys deleted by the owner, or recorded as his
follow-up · the time box and the $8 self-stop respected; any unfinished row marked "not run (time box)" or "not run (spend)".

## Risks / watch-outs

- **Spend-limit enforcement isn't instantaneous** (OpenAI: "not instantaneous"). The harness's own $8 self-stop is the real
  brake, and the stepped quota limits bound the quota legs ($1 → $2 → $3). Record the overshoot you see.
- **GPT-Live's connected minutes eat the budget.** The soak is ≈ $3.25 of the $5.50 checkpoint. Hang up between segments and
  keep to the per-step caps (task 3); otherwise mini's hard gates end up "not run (spend)" and the result escalates.
- **The APIs drift.** They were weeks old at planning. Re-read the model, WebRTC, sideband and spend-limit pages on the day and
  record every delta; if a documented endpoint or event name changed, follow the docs and say so in the note.
- **Key hygiene.** One `echo $OPENAI_KEY_S6` in a transcript is a leak. Env vars via `read -s` only; no headers in logs; the
  scoped `KEYRE` checks in task 8 (count-only outside the diff). If a key leaks, the owner deletes it at once and the run
  continues on a new one.
- **Personal data.** The raw log holds the owner's words; fixtures carry placeholders only, and the raw log is deleted.
  The fake-capture prompt is synthetic speech.
- **Subjective gates.** M14 and M10 are the owner's ratings; keep the script identical across shells so the ratings compare.
- **Owner fatigue.** It's a long day. Run GPT-Live first (it's the preferred candidate), start the soak unattended over lunch,
  and cut the Firefox/Safari legs before the mini legs if the box runs out.
- **The result ages.** If [m6b-01](sprint-m6b-01.md) starts more than ~8 weeks after this run, it re-checks the fixtures and
  model pages first (its risk).
- **A "both fail" outcome changes v2.1.** Don't soften it: record it with ⛔ "needs owner decision", let the owner decide what
  v2.1.0 carries, and still land the results (D40).
