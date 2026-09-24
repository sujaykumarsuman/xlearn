# Prompt — Sprint spk-04 · S6 voice-shell bake-off (owner present, throwaway)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root, **with the owner at the keyboard for the day**.
> **Plan:** [`../sprints/sprint-spk-04.md`](../sprints/sprint-spk-04.md)   ·   **Milestone:** M6a (spike S6, gates the M6a design freeze)   ·   **Prereqs:** none in `depends_on`; owner go-ahead O1 + presence (event `ev-s6`)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions and the land-and-sync directive (it applies only to the results docs PR here).
- The plan: [`../sprints/sprint-spk-04.md`](../sprints/sprint-spk-04.md) — the harness parts table, the per-step table, the M1–M17 thresholds, the decision rules, the scrub rules and the note's contents. Follow it exactly.
- [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) (the spike: question, environment, steps 1–10, M1–M15, rules), [§13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D28 and the D29 measurements it added), [§2](../research/t6-realtime-interviewer.md#2-model--pipeline-options) (the two shells), [§3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (SDP brokering, the sideband and its audio copies, the browser control surface, surviving deploys), [§4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (the classifier table and the probe), [§5](../research/t6-realtime-interviewer.md#5-the-coding-round) (director push), [§8](../research/t6-realtime-interviewer.md#8-cost-on-byo-keys) (the cost baseline for M9), [§14](../research/t6-realtime-interviewer.md#14-risks) and the Sources list (re-verify each page on the day).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) — **Proposed; don't edit it.** [ds-m6a-01](../sprints/sprint-ds-m6a-01.md) accepts it from your result (BP2).
- [Feasibility D28/D29](../feasibility.md#decisions-log-newest-first) and [T6](../feasibility.md#t6--realtime-ai-mock-interviewer); [rollout §4 M6a/M6b](../rollout-plan.md#m6a--m6b-v21--57--45-sprints) (gates) and [§6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar) (S6 before the M6a design freeze).
- What consumers need from you: [mi-13](../sprints/sprint-mi-13.md) (M8 and the CSP check, its task 3), [m1-10](../sprints/sprint-m1-10.md) task 2 (the provisional `voice_shell` catalog entries), [m6a-02](../sprints/sprint-m6a-02.md) (replay fixtures), [ds-m6b-01](../sprints/sprint-ds-m6b-01.md) (browser list), [m6b-01](../sprints/sprint-m6b-01.md)/[m6b-02](../sprints/sprint-m6b-02.md).
- Code to mirror, read-only:
  - `internal/platform/httpx/httpx.go` (`NewServer` timeouts; the 60 s WriteTimeout is why the SSE is bounded);
  - `internal/gateway/coach.go` (the 10 s coach client the 20 s SDP route will sit beside);
  - `internal/gateway/security.go` (the CSP string [m1-04](../sprints/sprint-m1-04.md) shipped, copied verbatim onto the harness page).

## Context

The M6 interviewer is a text **director brain** plus one thin **voice shell** on the learner's OpenAI key. The browser talks
WebRTC straight to OpenAI; coach brokers the SDP server-side (no credential in the browser) and holds an outbound sideband that
also receives audio copies, which it drops unparsed. Two shells are candidates: **GPT-Live-1** (flat $0.05/min, full duplex,
model-controlled turn-taking, push-to-talk only emulated) and **`gpt-realtime-2.1-mini`** (token-billed, native VAD and
push-to-talk, 60-minute cap). The owner decided (D28) that GPT-Live-1 wins only if it passes every hard gate **and** he rates it
≥ 1 point more natural; otherwise mini; both failing leaves text only. S6 is the ≤ 1-day bake-off that decides it, plus the
deploy shape (M7: does a coach restart drop the call?) and the quota fixtures. It runs on a localhost harness on the owner's Mac,
on two throwaway OpenAI projects with enforced hard limits: $10, and $1 for the quota test, raised in $1 steps to at most $3.
The harness stops itself at $8, with GPT-Live's legs held to $5.50 so mini's hard gates still get measured.
Nothing touches the VPS or the cluster. ADR-0032 stays Proposed until [ds-m6a-01](../sprints/sprint-ds-m6a-01.md) accepts it
from your note.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **The owner gives the explicit go-ahead O1 in this session.** Without a clear "go" from the owner here, stop: no agent uses the key.
- [ ] **The owner is present for the day** (event `ev-s6`) to create the projects, play the candidate and rate the shells.
- [ ] The owner's OpenAI org is **Tier 1+** (ask; GPT-Live has no free tier). 2FA on OpenAI (MI-1): ask and record; not a blocker.
- [ ] Chrome stable, Firefox and Safari installed; `go version` ≥ 1.26.
- [ ] **No command touches production:** no `ssh vps`, no `kubectl` (the laptop's context tunnels to production).
- [ ] Parallel sessions: `gh pr list --state open`, `git worktree list`, ListAgents — no open PR edits `docs/v2/research/t6-realtime-interviewer.md` or adds `docs/v2/research/t6-s6-fixtures/`.

## Do this (in order)

1. **[O] Accounts.** Ask the owner to create throwaway projects `xlearn-s6` (**$10 hard limit**) and `xlearn-s6-quota` (**$1
   hard limit**) with one project-scoped key each (realtime/live + Responses only where the UI allows; shortest expiry), then set
   them in the harness terminal with `read -s OPENAI_KEY_S6` / `read -s OPENAI_KEY_S6_QUOTA` (export after). You never see,
   print or store the keys. Record the tier and which key restrictions the UI offered. Agree the quota project's limit steps
   ($1 → **$2** after GPT-Live's quota leg → **$3** after mini's, never higher). Then **ask the plan's task 1 question**
   and record the answer: if GPT-Live passes every hard gate and mini fails one, does GPT-Live win even when M14 < mini + 1?
2. **[H] Re-verify the docs** in t6's Sources (GPT-Live model, WebRTC, live conversations, delegation, sideband/server controls,
   Realtime WebRTC and costs, spend limits, prepaid billing, pricing). Note every delta from t6 §2/§3/§8; use the prices of the day
   as the harness constants.
3. **[H] Build the harness (≤ 1.5 h)** in `<scratchpad>/s6/` (outside every repo; `go mod init s6harness`; `github.com/coder/websocket`):
   - the `127.0.0.1` server with the `httpx.NewServer` timeouts copied in. Mini's captions go over a **bounded SSE (≤ 50 s,
     reconnect with `Last-Event-ID`)**, because the 60 s WriteTimeout would cut an open stream;
   - `POST /segments?shell=live|mini` brokering the offer (≤ 16 KiB) under a 20 s budget: to `/v1/live/sessions` (JSON, with
     **`store:false`** in the body) or to `/v1/realtime/calls` (multipart);
   - the 15 s orphan reaper;
   - the sideband with a **type-first streaming decoder that drops audio events without unmarshalling them**;
   - the stub director on `gpt-6-sol` (Responses, **`store:false`**) with D29's 2–3 s on-change screen updates, one
     replaceable current-screen item, and an off switch for the soak;
   - PTT, hang-up, reseed and run-echo controls;
   - the JSONL log (**no audio, no SDP, no headers**);
   - the $ tally, per shell, with the **GPT-Live checkpoint at $5.50** and the **self-stop at $8**;
   - the static page, served with the gateway's **shipped CSP verbatim**: `grep -n Content-Security-Policy
     internal/gateway/security.go` on `main`, which is m1-04's full string (the plan quotes it). Add `Permissions-Policy:
     camera=(self), microphone=(self)`. No inline script or style. The page listens for `securitypolicyviolation`.

   Smoke 60 s on each shell. Then, with the plan's key-shaped `KEYRE` (never a bare `sk-`), run
   `rg -c "$KEYRE|v=0|a=candidate|a=fingerprint"` on the raw log. It must print nothing.
4. **[O · H] GPT-Live, steps 2–7** on `xlearn-s6`, per the plan's step table: the scripted 15-minute mock (owner as candidate:
   30 terms, 3 × 45 s silences, 2 min typing-while-thinking without headphones, 2 barge-ins, 3 min emulated PTT, 5 code questions,
   1 run echo, 3 latency probes × {push-only, client delegation, Responses delegation}, a 5-minute D29 typing segment); the mid-call
   restart (second sideband **before** killing the first, then a cold re-attach; `getStats()` packets keep rising); tamper from
   DevTools; the quota leg on `xlearn-s6-quota` (first failing events, new-session error, overshoot, **does the live session
   die?**, the owner raises the limit **to $2** and you time purchase → probe success; label it **"spend-limit path"**); the
   reseed (brief + last 4 turns, TTFA, owner rates 1–5). Last of all, the **65-minute unattended soak**, with the director
   off, a separate Chrome profile and `--use-fake-device-for-media-stream --use-file-for-fake-audio-capture=<s6>/prompt.wav`
   (a `say`-generated prompt converted with `afconvert`). **Spend:** keep to the plan's per-step connected-minute caps, hang
   up between segments, and print GPT-Live's running total after each leg. Start the soak only while ≥ $3.25 of the $5.50
   remains; a leg that would pass $5.50 is "not run (spend)".
5. **[O · H] mini, step 8** (from the ≥ $2.50 kept back): steps 2 (10 min, same script), 3, 4, 6, 7; native
   `turn_detection:null` PTT; `semantic_vad` low; the cached-token ratio before and after a 15-minute silence; no-data-channel
   offer accepted, `m=video` refused; `conversation.item.delete` + re-create for the current-screen item. The quota leg runs
   on `xlearn-s6-quota` at $2; the owner then raises it **to $3** for recovery timing. **mini's M5 has no soak** (t6 step 8):
   judge it on the documented 60-minute cap, re-read that day with the page and date, plus step 7's reseed TTFA ≤ 3 s.
6. **[O · H] Browsers and SDP, step 9**: Firefox and Safari 5 min each on the winning-so-far shell (both shells if time and
   spend allow; they're cut first if the box runs out); inspect the answer SDP in memory for ICE-TCP/TLS
   candidates (never save it); look for a sideband audio filter; confirm from DevTools Network that **the page never requested
   an `openai.com` origin** (media aside) and that no CSP violation fired.
7. **[X] Decide.** Fill M1–M17 per shell and apply the plan's task 6 rules:
   - GPT-Live if every H passes and M14 ≥ mini + 1;
   - else mini if every H passes;
   - GPT-Live as the sole passer (M14 < mini + 1) → per the owner's task 1 answer: yes → GPT-Live-1; no → "both fail";
     not asked → escalate, recommending GPT-Live-1;
   - both fail → text only, revisit in 3 months;
   - M7 fail → the `coach-interview` Deployment (`COACH_ROLE=interview`) in P1.

   Rows t6 §10 leaves unmeasured **by design** get the plan's labels, and they are a complete result: mini M5 is judged on the
   documented cap plus the reseed, mini M8 is "not run on this shell", and M15 covers the winning-so-far shell. Rows cut by
   time or spend never count as a pass. Anything outside the rules is **escalated to the owner in the note**, not decided.
8. **[X] Scrub and deliver.** On a `docs/spk-04-s6-results` branch off an up-to-date `origin/main`: generate the fixtures with a
   throwaway scrub script. It applies synthetic ids; removes all SDP, headers, org/project ids, IPs, emails and names; replaces
   transcripts with placeholders that keep `len`; and replaces **every request-body content field** (director `input` and
   `instructions`, message `content`, `session.instructions`, reseed briefs, screen items) with a placeholder that keeps
   `len`, keeping only keys, `model`, `store:false` and types. Commit the fixtures to `docs/v2/research/t6-s6-fixtures/`:
   `index.json` with, per file, `observed[]`, `expected_class` (m6a-02's class names) and `expected_action` from t6 §4's
   table, plus one JSONL per scenario (the plan's list). Run the plan's `rg` scrub check with `KEYRE`; it must be empty. Then
   have **the owner skim one fixture per scenario**. Append **`## 16. S6 results (spk-04, <date>)`**
   to `docs/v2/research/t6-realtime-interviewer.md` with the plan's contents (environment and deltas; the table; the decision; the
   sections for m6b-01, m6b-02, m6a-02, ds-m6a-01/AB24, ds-m6b-01/AB29, mi-13 and the catalog; spend; teardown; step 10 pending)
   and a pointer line in its top blockquote. Update `docs/v2/status.md` and this sprint's Status table.
9. **[O · H] Teardown.** The owner deletes both projects and keys and confirms. Then run the scoped key checks:
   - `git diff origin/main...HEAD | rg -n "^\+.*$KEYRE"` → nothing. Don't use a repo-wide `git grep 'sk-'`: `main` already has
     60+ harmless hits.
   - `rg -c "$KEYRE" <scratchpad>/s6 ~/.zsh_history` → counts only; never print matches. A non-zero count goes to the owner.

   Then delete `<scratchpad>/s6/` entirely.
10. **[X] Ship the docs PR** (see the last line).

## Constraints

- **Throwaway:** the harness, raw logs, audio, Chrome profile and SDP never enter any repo. Only scrubbed fixtures, the t6 §16 note and status rows are committed. The repo is public.
- **Key hygiene:** keys only in env vars set by the owner with `read -s`; never echoed, logged, written or pasted; the harness logs no headers. Key checks use the key-shaped `KEYRE`, scoped to the PR diff, and count-only on local files. If one leaks, the owner revokes it at once.
- **Spend:**
  - `xlearn-s6`: hard limit $10 (enforced); harness self-stop at $8; GPT-Live checkpoint $5.50, which leaves ≥ $2.50 for mini and the browsers.
  - `xlearn-s6-quota`: $1 → $2 → $3, never higher.
  - Worst case $11 across both; t6 expects ≈ $6–7. Never use the owner's normal project or a platform key.
- **No production contact:** no VPS, no cluster, no infra PR, no deploy. GitOps, the memory-sum rule, consumers-before-producers and ACL-before-tag are n/a (nothing runs in the cluster).
- **No ADR edits:** ADR-0032 stays Proposed; [ds-m6a-01](../sprints/sprint-ds-m6a-01.md) accepts it (BP2). No code in `internal/`, `web/` or `cmd/`.
- **D34:** add no alerting of any kind; S6's M-numbers are one-off measurements, not monitors.
- **Parallel sessions:** before pushing the docs PR, re-check peers (`gh pr list --state all`, `git worktree list`, ListAgents): t6 and `status.md` may be open elsewhere; rebase onto a moved `main`.
- n/a here: service boundaries (ADR-0005), goose + sqlc, outbox/inbox, `theme.css`.
- **Time box:** 1 working day hands-on (≤ 8 h) plus the soak. At the box, stop and record what's known. Rows cut by the box or spend stay explicit ("not run (time box / spend)") and never count as a pass. Rows t6 §10 leaves unmeasured **by design** are the exception and are judged as the plan's task 6 says: mini M5 on the documented cap plus the reseed, mini M8 not run, M15 on the winning-so-far shell.

## Deliverables

- `docs/v2/research/t6-realtime-interviewer.md` §16 — the one-page S6 results note.
- `docs/v2/research/t6-s6-fixtures/` — `index.json` + scrubbed JSONL fixtures (quota "spend-limit path", lifecycle, soak close, tamper, no-data-channel, probe, director `store:false`).
- The decision: shell, deploy shape (M7), browser gate list, delegation used or not — or an explicit escalation.
- `docs/v2/status.md` rows; owner confirmation that both projects and keys are deleted.

## Update status

- Set each task in [`../sprints/sprint-spk-04.md`](../sprints/sprint-spk-04.md) 🔄 / ✅ / ⛔ as you go; _Overall_ ✅ when all eight are ✅.
- [`../status.md`](../status.md): the **Sprint board** row; the **spike-results** row (spk-04: shell, deploy shape, browsers, date, link to t6 §16); owner event **`ev-s6`** ✅ with the date; **Decisions log** lines (shell; deploy shape; browser list; any escalation). ADR-0032 stays Proposed there.
- No ADR is written or accepted here. If the result contradicts ADR-0032 (e.g. both shells fail, or brokering fails), say so in the Decisions log so ds-m6a-01 amends it at acceptance or escalates.

## Done when (acceptance)

- [ ] Every M1–M17 row is recorded **per shell**, with a number or one of the plan's by-design labels (mini M5 on the documented cap plus the reseed TTFA; mini M8 "not run on this shell"; M15 on the winning-so-far shell), or marked "not run (time box / spend)". The decision follows the task 6 rules or is explicitly escalated.
- [ ] `xlearn-s6` ≤ $8 (self-stop; hard limit $10), and the $5.50 GPT-Live checkpoint held or its cut legs are marked. `xlearn-s6-quota` ≤ $3. Per-leg and per-project totals are recorded.
- [ ] The page ran under the gateway's shipped CSP (m1-04's string verbatim, `connect-src 'self'`) and `camera=(self), microphone=(self)` with no violation and no page-side provider request, recorded for mi-13.
- [ ] Only scrubbed fixtures (`index.json` with `observed`/`expected_class`/`expected_action`), the t6 §16 note and status rows are committed, through one merged docs PR. The scrub `rg` is empty and the owner confirmed.
- [ ] Both OpenAI projects and keys deleted (owner confirmed). The scoped key checks (PR diff; count-only on the harness directory and shell history) came back clean. The harness directory, raw logs and audio are gone.

**Shipping:** spike — the harness, logs, audio and projects are **throwaway and never committed**. Per AGENT.md land-and-sync,
ship **only the results docs PR** (branch `docs/spk-04-s6-results` → conventional commit `docs(v2): S6 voice-shell bake-off results (spk-04)`
with the attribution lines → push → PR → merge once CI is green → `git checkout main && git pull`). No tag, no infra PR, no deploy.
