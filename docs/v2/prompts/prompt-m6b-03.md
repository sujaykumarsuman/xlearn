# Prompt — Sprint m6b-03 · Voice UI: pre-flight/notices (AB29), live HUD★ (AB30), consent, EU gate, self-view → v2.0.x patch (M6b dark)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root. No owner time is needed in-session (D40): the owner's 5-minute pre-flight look is a post-ship pending-smoke note (plan task 7).
> **Plan:** [`../sprints/sprint-m6b-03.md`](../sprints/sprint-m6b-03.md)   ·   **Milestone:** M6b (voice, one shell)   ·   **Prereqs:** [m6b-02](../sprints/sprint-m6b-02.md) (after [m6b-01](../sprints/sprint-m6b-01.md)), [ds-m6b-01](../sprints/sprint-ds-m6b-01.md), [mi-13](../sprints/sprint-mi-13.md), [m6a-06](../sprints/sprint-m6a-06.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): conventions, GitOps, land-and-sync.
- **The plan:** [`../sprints/sprint-m6b-03.md`](../sprints/sprint-m6b-03.md). The gate table, the pre-flight order, the voice-client contract, the HUD state list, the media-audit output and the release checklist are spelled out there.
- **The frozen boards:** `design-system/screens/v2/AB29-*.html` (pre-flight, browser and EU notices) and `AB30-*.html` (voice live HUD★), including their behaviour notes. Also M6a's AB24–AB28 for the pieces this sprint reuses. Use [`../../../design-system/theme.css`](../../../design-system/theme.css) verbatim; [`../../../design-system/README.md`](../../../design-system/README.md) lists the tokens and components.
- **The M6a sprints this UI builds on** (and their merged PRs, which win where they differ):
  - [m6a-01](../sprints/sprint-m6a-01.md): the FSM table (`setup —start→ preflight` guards: consents, `cap_micros > 0`, ≤ 2 starts a day), the interrupt reasons, the client lease (`attach`, `heartbeat`), the `interviewAudience = "cohort"` gate, and `qa_slice` (owner-only, never scored);
  - [m6a-03](../sprints/sprint-m6a-03.md): `ScoreMock` for `format=voice`, `self_only_voice`, 409 `qa_session_unscored`;
  - [m6a-05](../sprints/sprint-m6a-05.md): the routes (`mock/setup`, `mock/live/:id`, `mock/:id/review`), `web/src/screens/mock/*`, `web/src/lib/interview/api.ts`, and the `pagehide` beacon (`interruptBeacon`, `text/plain`).
- **Research, t6:**
  - [§3](../research/t6-realtime-interviewer.md#3-architecture--media-path): the media path; the browser control surface (data channel, never `session.update`); the SDP never logged.
  - [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes): the FSM, failure modes (mic loss → `interrupted(mic)`), grace, pause and resume.
  - [§5](../research/t6-realtime-interviewer.md#5-the-coding-round): the editor, "hold voice while I code", the slice rail.
  - [§7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility): consent lines, the EU/EEA rule, accessibility.
  - [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys): the pre-flight estimate; the live meter hidden.
  - [§9](../research/t6-realtime-interviewer.md#9-phased-plan): the P1 row.
  - [§11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream): privacy-notice additions.
- **ADRs:**
  - [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §2, §3, §6;
  - [ADR-0033 §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2), rows 14 and 15;
  - [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L19);
  - [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.1, §2 and **§6 (release checklist)**.
- **Rollout:** [`../rollout-plan.md`](../rollout-plan.md) §3 (the M6b row), [§4 M6a/M6b](../rollout-plan.md#m6a--m6b-v21--57--45-sprints), [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v2.0.x`: M6a/M6b dark), [§9](../rollout-plan.md#9-artboards-by-milestone) (AB29, AB30★, the freeze rule). The PRD: [`xlearn-v2-prd.md`](../../prd/xlearn-v2-prd.md) §5.6, R-MI1–R-MI8.
- **The S6 results note** from [spk-04](../sprints/sprint-spk-04.md): the browser list, the shell, the data channel, PTT native or emulated. **The PRs** of [m6b-01](../sprints/sprint-m6b-01.md), [m6b-02](../sprints/sprint-m6b-02.md) and [mi-13](../sprints/sprint-mi-13.md): the `voice` block, the broker body and errors, confirm, ask, voice-mode, the turn self-edit, caption and segment events, and the fake-provider endpoints m6b-01's PR description specifies for this compose walk. Also [l-05](../sprints/sprint-l-05.md) for the privacy notice's shape.
- **Code:**
  - `web/src/lib/api.ts` (`apiFetch` and its **15 s** `REQUEST_TIMEOUT_MS`), `web/src/router.tsx`;
  - `web/src/screens/mock/*`, `web/src/lib/interview/*` (m6a-05/06), `web/src/test/` (stubs);
  - `internal/gateway` (the interview proxy and its cohort gate);
  - `internal/coach/interview` (the FSM table, the broker's checks 4–6, `consent.go`) and coach's `admin` CLI (m6a-01's `interviews --live`), `internal/coach/store` (schema `coach`, the `api_key_config` key envelope);
  - `docs/architecture/api.md` + `openapi.yaml`.

## Context

- **What is live.** v2.0.0 (owner-facing GA) is live. M6a, the text interviewer, shipped **dark** to the T-3 cohort in [m6a-06](../sprints/sprint-m6a-06.md)'s `v2.0.x` patch. MI-16 is live too ([mi-13](../sprints/sprint-mi-13.md)): the SPA's `Permissions-Policy` and the 20 s SDP route `POST /api/interviews/{id}/segments`.
- **Backend voice is merged, not yet shipped.** [m6b-01](../sprints/sprint-m6b-01.md) merged the `VoiceShell` adapter for the S6-winning shell, the coach-brokered SDP with its preconditions (client lease, FSM state, voice consent, **the EU/EEA region gate in coach**, the key), confirm, ask, and the audio-dropping sideband. [m6b-02](../sprints/sprint-m6b-02.md) merged the sideband lease, re-attach, drain, per-segment cost with a mandatory $ cap, the L19 voice caps, PTT modes and the mid-call mode switch, hold, rollover and the transcript self-edit. Neither has shipped yet.
- **This sprint builds what the learner sees:**
  - the **AB29** pre-flight and notices: the Chrome/Edge gate and the server's `voice.reason` come before any mic prompt or spend, and consent and the cap come before `start` and the paid voice check;
  - **voice consent**;
  - the **AB30★** live HUD, through the post-call "Check your transcript" screen;
  - an optional **local self-view** that never enters the PeerConnection.
- **It also prepares [m6b-04](../sprints/sprint-m6b-04.md)'s fake-media run on prod:** a read-only `coach admin interviews media-audit` verb, m6a-01's owner-only `qa_slice` reachable from the setup UI, and the runbook.
- **Then it ships everything as a `v2.0.x` patch, still dark:** cohort only, interviewer defaults unchanged. The GA flip and `v2.1.0` belong to m6b-04.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] AB29–AB30 frozen: [ds-m6b-01](../sprints/sprint-ds-m6b-01.md) merged (the merge is the freeze, D40); the boards are on `main`. If status.md's AB29/AB30 rows don't read "frozen (PR #, date)" yet, set them (idempotent)
- [ ] [m6b-01](../sprints/sprint-m6b-01.md) and [m6b-02](../sprints/sprint-m6b-02.md) merged. List what this UI consumes: the `voice` block fields and its `reason` enum; `attach`/`heartbeat`; the `/segments` body, response and typed errors; confirm; `ask`, `ptt`, `hold`, `voice-mode`, `PATCH …/turns/{seq}`; the caption, segment and notice events
- [ ] [m6a-06](../sprints/sprint-m6a-06.md)'s patch is live (M6a screens + the interviewer cohort gate); [mi-13](../sprints/sprint-mi-13.md)'s patch is live (`curl -sI …/xlearn/` shows `permissions-policy: camera=(self), microphone=(self)`; the `/segments` route exists)
- [ ] `/api/me` returns `acceptance.region` ([l-05](../sprints/sprint-l-05.md)), and every account has passed acceptance
- [ ] The S6 results note is at hand; ADR-0032 is Accepted
- [ ] Only if S6 M7 failed: `coach-interview` is live and routed
- [ ] `.release-line` = `2`; ranges `>=1.0.0 <3.0.0`
- [ ] No open peer PR on `web/src/screens/mock/**`, `web/src/lib/interview/**`, `router.tsx`, `api.ts`, `package.json`, the gateway interview proxy, `internal/coach/interview` or coach's admin CLI (`gh pr list`, `git worktree list`, ListAgents)

## Do this (in order)

1. **(X) Branch** `feat/m6b-03-voice-ui`. Read the boards' behaviour notes and the m6b-01/02 PRs, and write the frame → component map into the PR description.
2. **(X) Server gates** (plan task 1, "Server gates"; verify each in coach, add only what's missing):
   - a) EU/EEA is **coach's** (m6b-01 check 5): 403 `voice_unavailable_region` on a `mode=voice` create and on every `/segments` purpose, for EEA codes (EU-27 + `IS`, `LI`, `NO`; `GR` for Greece) and NULL. Fill any missing table-test cases in m6b-01's tests. **No gateway 403 and no second EEA list**; only if m6b-01 left no EEA set at all, add it where check 5 reads it.
   - b) Voice consent: m6b-01 check 4 (409 `consent_required`).
   - c) Key: m6b-01 check 6 (409 `voice_key_required`).
   - d) The `mic` interrupt reason: add it to m6a-01's FSM data table (grace like `network`) with a table test if nothing added it yet; widen a `state_reason` CHECK by an expand migration only if one exists.
   - If a DTO or route changes, update `openapi.yaml`, `api.md` and the drift test.
3. **(X) Voice client** `web/src/lib/voice/` (plan task 2), as a lazy chunk on m6a-05's `lib/interview/api.ts`:
   - m6a-01's client lease: `attach {client_id}` on mount, `heartbeat {client_id, visible, last_input_at}` ≤ 1 / 10 s. m6b-02's sideband lease is server-only; never conflate the two;
   - mic capture only on a click; `sendrecv` audio; **no video track**;
   - the `oai-events` data channel only if `voice.data_channel` asks, and never `session.update`;
   - **one** `/segments` POST with body **`{sdp, purpose, client_id}`** (`preflight` for the AB29 check; `live` from `connecting`, `interrupted`, `held`, `bridging`, and on `reprime`/`rollover` events), its own ≈ 22 s signal (not `apiFetch`'s 15 s), no retry except one after 503 `draining`, and text mode on 504/5xx/abort;
   - the 201 `{segment_n, sdp, shell, confirm_by}`; the answer and the audio element; `POST …/segments/{segment_n}/confirm` on `connected`, before `confirm_by`;
   - the typed errors mapped to frames (plan task 2's table);
   - captions from the SSE; no client mirror;
   - ICE/offline → `interrupt(network)`; mic loss → "Reconnect mic" at 10 s, `interrupt(mic)` at 30 s;
   - PTT (`track.enabled` + `ptt`), the mid-call toggle (`voice-mode`), Hold/Talk;
   - teardown everywhere, plus **m6a-05's `interruptBeacon`** on `pagehide` (one mechanism for both HUDs);
   - never log an SDP.
4. **(X) AB29** (plan task 1), **in this order**, because m6a-01's `start` needs consent and the cap and the voice check needs `preflight`:
   - the gate frames first (cohort → key → browser → region/null → L19 429s), keyed to `voice.reason ∈ {region, no_voice_key, model_access, capacity, daily_limit, disabled}`;
   - F1 mode (create with `mode: "voice"` and `voice_mode`) → F2 **unticked** voice-consent boxes in AB29's final copy, recorded at the current version, [Continue] disabled until every box is ticked → F3 mic explainer, denied and no-device states → F4 device picker, level meter, headphones toggle → F6 estimate with the **mandatory, editable $ cap** (default plan-for × 1.5) and the multiplier → `start` (`setup → preflight`) → F5 the real voice check ("uses your key (≈ $0.02)", `purpose: "preflight"`) with F5a–F5d → F11 screen-reader preset → F12 Ready and **Start** as the gesture;
   - F13 the resume reaffirmation; F14 the narrow layout.
   - A Vitest proves no `/segments` POST happens before the consent is recorded, the cap is set and `start` returned `preflight`.

   Check the privacy notice for the t6 §11 disclosures. If it has a gap, draft the update as its own **privacy-notice PR** (l-05's shape: `web/src/content/privacy-notice.md` with "What changed", `NOTICE_VERSION` and identity's `NoticeVersion` bumped together) and merge it on CI green before the tag. It lands as drafted (D40): the owner may revise it later with a content PR, and his review of the notice is a v3 opening gate. Every account, the owner included, re-accepts once it ships in this patch.
5. **(X) AB30★** (plan task 3):
   - the rail and server clock; the captions log (`role="log"`, `aria-live="off"`; candidate captions "approximate"); [Ask to repeat] / [Rephrase that] → `POST …/ask`;
   - mic and camera state (icon + text); the editor + Run from m6a-04;
   - PTT with the F4 mid-call toggle; Hold voice while I code / Talk; the mode indicator;
   - the cap notices per the frozen board (default: no running $ while live);
   - the voice grace variant (5:00 countdown, Check now ≤ 1 / 10 s, 30 s probes, the ≥ 75% buttons, mic shown muted, self-view stopped);
   - the F14 distress card (M6a's safety card, AB30's copy);
   - bridging, held, interrupted (network/mic/provider/idle, with F8 "Still there?" on `notice{idle_check}`), "Live in another tab";
   - paused/resume with the reaffirmation; wrapping and the spoken debrief;
   - F17 **Check your transcript** in `finished`/`proposed`: candidate lines only, ≤ 2 KiB, `PATCH …/turns/{seq}`, the "edited" chip and the self-reviewed banner; then AB27.
   - Accessibility: `aria-live` only on phase changes and the grace warning; the board's shortcuts, not clashing with the editor; reduced motion; the < 1024 px layout.
6. **(X) Self-view** (plan task 4): off by default, a separate video stream, mirrored `<video muted>`, never in the PC. It stops on toggle-off, grace, pause, finish and unmount; a lost camera is silent.
7. **(X) Prepare m6b-04** (plan task 6):
   - `coach admin interviews media-audit <id>` (read-only JSON; the binary-column allowlist = the key envelope; per-table rows and max size; SDP markers; ≥ 4 KiB base64 runs; the segment count, voice minutes and µUSD; non-zero exit unless `clean`), with real-Postgres tests (clean / planted SDP / planted base64);
   - m6a-01's owner-only **`qa_slice`** reachable from the owner's setup UI: add a "QA slice (0.33×)" toggle only if none exists (and accept `qa_slice` on create, owner-only, only if m6a-01's route doesn't yet). **No new `rail` field, no new error code**; the QA session is never scored (409 `qa_session_unscored`);
   - `docs/runbooks/voice-fake-media-e2e.md` (a ≈ 3-minute looped clip via `say` → `afconvert` WAV 48 kHz mono, never committed; the Chrome flags with `--remote-debugging-port` and a throwaway profile, webrtc-internals in a separate window; the expected ≈ $0.7–1.2 spend and a cap ≈ 2.5× that; m6a-01's start rule (one retry a day); owner vs agent roles; the steps, ending at `proposed` → refused → abandoned; the no-media checks; the cost reconciliation; cleanup).
8. **(X) Tests and the compose walk** (plan task 5):
   - Vitest for gates, the pre-flight order, consent, the client (`purpose` per state, `client_id`, one POST, one retry on `draining`, 22 s, confirm, no video sender, no `m=video`, no `session.update`, teardown, the beacon, no SDP in console), the HUD (ask, PTT and toggle, idle, distress, F17) and self-view.
   - Go tests for any gate you added (the `mic` reason) and for the verb.
   - Compose: extend the scratchpad fake-LLM provider (the local-review memory note) with the two endpoints m6b-01's PR description specifies, and add a **dev-only `RTCPeerConnection` stub** (`import.meta.env.DEV` plus a flag; a build test greps `dist/` for its marker) so the HUD reaches `connected`. Walk every AB29 frame and the AB30 frames the fake sideband drives, a `DE` account (notice + a `curl` 403 `voice_unavailable_region` from coach through the proxy), a null region, a 429 and a 504. Record which frames were left to Vitest and to m6b-04.
   - Screenshots at 1440/390 next to the boards.
   - `npm run build`: the voice code is its own chunk, the entry grows by < 5 KiB.
   - CI: `go test ./...`, `go vet`, lint, the web suite, `sqlc diff`, OpenAPI drift.
9. **(X) PR and merge** once CI is green (**Ship** steps 1–2 below), with step 4's privacy-notice PR if one was needed. If S6 M7 failed, confirm `coach-interview` is live first.
10. **(X) Tag.** Run the release checklist in the plan's **Release** section: parallel-sessions check, next free `v2.0.N`, major = `.release-line`, and **`coach admin interviews --live` empty** right before the tag. Tag `v2.0.N` with the release title `v2.0.N — v2.1 build · M6b dark (voice)`.
11. **(H) Verify on prod** (plan task 7):
    - the healthz version, the images, the ImagePolicies' latest (except `xlearn-coach-interview` if M7 failed, which lags until step 12), HelmReleases Ready;
    - a smoke test of login, dashboard and coach, with an already-signed-in browser session if you have one (never enter credentials); otherwise it joins the pending-smoke note below;
    - an anonymous `/segments` POST → 401; the headers unchanged; the voice chunk served;
    - `media-audit` runs; `--live` empty; `host-verify --cluster` green.
    - **The owner's look (≈ 5 min, post-ship; not a gate, D40):** add it to status.md's pending-smoke notes (for `v2.0.N`, run by the owner): he opens Mock-v2 → Voice through the gates, consent, mic permission and the mic check, stops before `start`, and abandons that `setup` interview. You never sign in and never start a paid voice check, and you don't wait: task 7 is ✅ on your checks, and [m6b-04](../sprints/sprint-m6b-04.md)'s owner-present run closes the note at the latest.
12. **(I) Only if S6 M7 failed:** after the tag, with `--live` empty, open **its own `../infra` PR** moving `xlearn-coach-interview`'s exact pin to `2.0.N` (m6b-02's runbook), merge it once green, and re-check that policy and the Deployment (plan task 8).
13. **(X) Record** (plan task 9) in a small docs PR if anything is left after the tag.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):**
  - the SPA talks only to the gateway;
  - coach owns interviews, the key, every provider call and the voice gates (region, consent, key);
  - the gateway only proxies the region refusal; it doesn't enforce the region and holds no EEA list;
  - no new service and no new in-cluster caller (the ADR-0035 §2 standing rule then needs no infra PR).
- **No credential or media on the page's origin:**
  - the browser gets only an SDP answer; coach holds the key;
  - the CSP stays `connect-src 'self'`; don't touch the gateway's security headers (mi-13's);
  - the camera never enters the PeerConnection; the SDP is never logged or persisted;
  - the dev-only PeerConnection stub never reaches a production build.
- **Consent is the learner's own act:** the boxes start unticked and the server checks the version. No `/segments` POST before consent and the cap. Privacy-notice changes **land as drafted** (D40), merged on CI green before the tag; the owner may revise them later with a content PR (his review of the notice is a v3 opening gate).
- **UI:** `theme.css` verbatim, dark theme, no Tailwind; match the frozen boards. The difficulty tokens where shown: Easy=`--ds-ok`, Medium=`--ds-warn`, Hard=`--ds-err`. The boards are preview-only and never imported.
- **goose + sqlc:** no migration is expected; only a `state_reason` CHECK widening for `mic` if one exists (expand). The verb is read-only; if you add queries, commit the `sqlc generate` output, and `sqlc diff` must be clean. No outbox or event change.
- **Release discipline:**
  - a **patch**, never a minor (ADR-0034 §1.1); the interviewer defaults stay unchanged (cohort only);
  - **no live interviews** at tag time;
  - the parallel-sessions check (peers' tags, PRs, worktrees, ListAgents) before tagging or claiming an ADR number;
  - never move or re-push a tag;
  - the conditional `coach-interview` bump is its own infra PR, never folded into the tag.
- **GitOps only:** never `kubectl apply`/`edit`/`rollout` by hand. `ssh vps` is for reads and `host-verify.sh`; `kubectl exec` only runs the admin CLIs (`coach admin`).
- **D34:** no alerting, opscheck, interview counters, Flux Alert or push channel. The release checklist's live-interview check is the M6 ops step.
- **Memory-sum rule:** no new pod here. If M7 failed, `coach-interview` was sized and checked in m6b-02; don't resize anything.
- **Consumers before producers / ACL before the consuming tag:** n/a (no streams or consumers).
- **Spend:** never start a paid voice session on anyone's key yourself. Compose uses the fake provider; the real leg is m6b-04, with the owner present.

## Deliverables

- `web/src/lib/voice/` (the client, lazy) and the AB29 screens (gates, the FSM-ordered pre-flight, notices, consent, reaffirmation).
- The AB30★ HUD with the voice grace variant, the F4 toggle, ask/rephrase, the distress card and the F17 transcript check, and `web/src/components/SelfView.tsx`, all with Vitest suites.
- Server gates only where missing: m6b-01's coach-side region, consent and key checks verified (test cases filled in), the `mic` interrupt reason, and `openapi.yaml` + `api.md` if anything changed.
- `coach admin interviews media-audit` with tests; the owner's `qa_slice` toggle (if missing); `docs/runbooks/voice-fake-media-e2e.md`.
- The PR with the 1440/390 screenshots, the compose/Vitest/m6b-04 frame split and bundle sizes. A separate privacy-notice PR, only if needed, merged before the tag (it lands as drafted, D40).
- Tag `v2.0.N` (M6b dark), verified on prod by your checks; the owner's pre-flight look as a pending-smoke note. The conditional `coach-interview` ImagePolicy PR only if M7 failed.

## Update status

- This sprint's Status table in [`../sprints/sprint-m6b-03.md`](../sprints/sprint-m6b-03.md) (🔄 / ✅ / ⛔ per task; _Overall_).
- [`../status.md`](../status.md):
  - the **Sprint board** row;
  - the **Milestones** row M6b → "dark in `v2.0.N`";
  - **milestone → tag → floor → snapshot:** `M6b dark → v2.0.N → floor unchanged → no snapshot`;
  - the **artboards** AB29/AB30 → implemented (PR #);
  - the **flag inventory:** the interviewer T-3 cohort-gate row (`interviewAudience`; removal `v2.1.0`, m6b-04), noting that voice reuses it, and `qa_slice` as an owner-only QA tool, not a flag;
  - the privacy-notice outcome ("covered", or the merged PR # and the new notice version);
  - the **pending-smoke notes:** the owner's pre-flight look (and the login smoke, if no signed-in session covered it), for `v2.0.N`, run by the owner;
  - the **decisions log:** the data channel, captions path and no client mirror; the pre-flight order; the cost-display call; the PTT shortcut; the EU gate in coach (m6b-01), verified; the `mic` reason; the `pagehide` beacon for both HUDs; the ICE-gathering choice; the `disabled` frame; the media-audit verb; the `qa_slice` toggle; the runbook.
- No ADR expected. If one becomes necessary, claim its number only after the parallel-sessions check.

## Done when (acceptance)

- [ ] Every AB29 and AB30 frame is implemented as frozen (1440/390 px screenshots in the PR), including the voice variant of the grace countdown, the F4 mid-call toggle, the F14 distress card and the F17 transcript check
- [ ] The pre-flight follows the FSM: consent and the cap come before `start`, and no `/segments` POST happens before `start` returns `preflight` (tested)
- [ ] The gates hold, and each refusal offers text mode:
  - an unsupported browser gets the notice;
  - an **EU/EEA account sees the notice and no voice**, and coach answers 403 `voice_unavailable_region` through the proxy on a voice create and on `/segments`;
  - a null region gets "Confirm your region";
  - no OpenAI `voice_shell` key gets the key notice;
  - the L19 429s get their frames
- [ ] Voice consent: the boxes start unticked, [Continue] (and so Start) stays disabled until every box is ticked, the version is stored with the interview's consent row, and resume shows the reaffirmation. The privacy notice is covered, or its update merged before the tag (as drafted, D40)
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
- [ ] **M6b dark patch live on prod (cohort only):** `v2.0.N` is tagged with the release checklist and verified; the owner's pre-flight look is recorded as a pending-smoke note (not a gate); the interviewer defaults are unchanged; nothing changed for anonymous visitors

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). The xlearn branch is `feat/m6b-03-voice-ui`; the privacy-notice PR, if step 4 needed one, is its own xlearn PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (`../infra` has no CI: paste the local checks into the bump PR's body and merge on them.)
3. **Release action — tag a `v2.0.x` patch (M6b dark, cohort):** with the xlearn PR and any privacy-notice PR merged, walk the release checklist (ADR-0034 §6; the plan's Release section; steps 10–11 above), including "from M6: no live interviews" (`coach admin interviews --live` empty right before the tag). Push the tag `v2.0.N` (the next free patch; title `v2.0.N — v2.1 build · M6b dark (voice)`), let Flux deploy, then verify live by looking (step 11). No snapshot: a patch with no contract, erase or GA flip. Only if S6 M7 failed: after the tag, the `coach-interview` ImagePolicy bump as its own `../infra` PR (step 12). The owner's pre-flight look goes into status.md as a pending-smoke note; don't wait for it.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn, and `../infra` if M7 failed). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
