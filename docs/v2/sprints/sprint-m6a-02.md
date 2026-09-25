# Sprint m6a-02 — Text brain: loop, classifier, give_hint, resume brief, store:false, replay tests

> **Milestone:** M6a — text interviewer plus failsafes (v2.1 line; ships dark in a v2.0.x patch)   ·   **Track:** product (order 75)
> **Prereqs:** [m6a-01](sprint-m6a-01.md) (the interview core) · [spk-04](sprint-spk-04.md) (S6: the scrubbed quota fixtures)
> **Unblocks:** [m6a-03](sprint-m6a-03.md) (the review call and scoring reuse this brain, its classifier and its `store:false` test)
> **Release action:** **merge only (ships dark in the next v2.0.x patch)** — normally [m6a-06](sprint-m6a-06.md)'s M6a patch. No infra change.
> **Calendar:** Q1 2027. No owner involvement; the manual live replay (≈ $0.5) is documented as a runbook recipe and isn't run in this sprint (the owner runs it, or a later prompt that specifies the run and its $ budget, D40).
> **Execute with:** [`../prompts/prompt-m6a-02.md`](../prompts/prompt-m6a-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Brain loop: `interview` key, pre-flight probe, `Segment` abstraction (text impl), turns, cues, debrief, safety | X | ⬜ |
| 2 | Classifier by `error.code` (T6 table) + same-key probe + grace/re-prime policy | X | ⬜ |
| 3 | `give_hint` (recorded first) + resume brief (cached per pause) + re-prime | X | ⬜ |
| 4 | `store:false` acceptance test on every interview text call | X | ⬜ |
| 5 | Replay regression suite (S6 + synthetic fixtures) + the manual live-replay recipe | X | ⬜ |
| 6 | Docs: api/openapi, runbook, ADR-0032 dated update, status | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the
> M6a milestone row, decisions log). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m6a-01](sprint-m6a-01.md) **merged**: the coach interview tables, FSM, sweeper (with its nil-safe `Prober` and `OnPhase`
      hooks), `AddSpend`, the 503 `interviewer_unavailable` guard and `/api/interviews/*` are on `main`.
- [ ] **S6 fixtures available** ([spk-04](sprint-spk-04.md)): the scrubbed quota fixtures (labelled "spend-limit path") are merged
      under `docs/v2/research/t6-s6-fixtures/` (with `index.json`) and the results note is in t6 §16.
- [ ] `internal/platform/llm` ([m4-01](sprint-m4-01.md)) is live: typed errors in `errors.go`, catalog `caps.interview_brain`, `Cost`,
      `NoStore`; coach's key crypto and `key_default(feature='interview')` ([m1-10](sprint-m1-10.md)).
- [ ] Parallel sessions: [m6a-04](sprint-m6a-04.md) may be in flight (it depends only on m6a-01). Coordinate on the shared seams:
      m6a-01's event log (m6a-04 streams it), the `ScreenSink` screen hook (m6a-04's snapshot intake feeds it), the run-echo trigger
      (m6a-04's `run-echo` → this sprint's `OnRun`) and the gateway relay (`relayBounded` once m6a-04 merges). Adopt whichever
      interface merged first; never duplicate a route. No peer PR edits `internal/platform/llm/{errors,openai}.go` (serialize).

## Goal

Give the interviewer its **text brain** on the learner's `interview` key ([ADR-0032 §2, §4](../../adr/0032-realtime-ai-mock-interviewer.md#2-architecture),
[t6 §9 P0](../research/t6-realtime-interviewer.md#9-phased-plan)): a turn loop that follows the on-screen code (D29), a
**voice-ready segment abstraction**, robust **provider-error classification** with the same-key probe as arbiter (quota never
disables a key), `give_hint` recorded before any hint reaches a context, the **resume brief** cached per pause, a `store:false`
acceptance test and **replay-based regression tests** from recorded fixtures. After this sprint a cohort account can run a full
text interview through the API (the UI is [m6a-05](sprint-m6a-05.md)/[m6a-06](sprint-m6a-06.md)); scoring stays with m6a-03.

## Scope

**In**
- `internal/coach/interview/brain/` (director, prompts, context builder), `segment.go` (`Segment` interface + `TextSegment`),
  `classify.go`, `probe.go`, `hints.go`, `brief.go`, `safety.go`; registration into m6a-01's service (lifts the 503 guard).
- Pre-flight: key/model check with a real 16-token probe; `ErrModelAccess` caught here.
- Coach routes `POST /interviews/{id}/turns` (streamed), `POST /interviews/{id}/{hint|probe|brief}` and their gateway proxies
  (streamed ones on m6a-04's `relayBounded` if it merged, else `relayStream` — m6a-04 switches them); every turn, cue, hint and
  notice also appended to m6a-01's event log.
- Classifier table incl. realtime `response.done{failed}` codes (for M6b reuse), the probe, grace/re-prime policy.
- `llm/openai.go`: BYO structured `Complete` (JSON schema, `store:false`) for the brief — no platform OpenAI use.
- `store:false` test; replay suite; manual live-replay recipe; coach memory measured in compose.

**Out (later sprints)**
- The review call, proposal validation, `ai-byo`, `ScoreMock`, the twin gate → [m6a-03](sprint-m6a-03.md) (it adds its call kind to this sprint's `store:false` test).
- Run/final through judge, the echo **caller**, CodeMirror snapshot publisher, bounded SSE `GET …/events` → [m6a-04](sprint-m6a-04.md).
- Voice shell, sideband, director *push* notes into a live voice model, countdown grace UI, AI-notes checkpoints, transcript
  self-edit → [m6b-01](sprint-m6b-01.md), [m6b-02](sprint-m6b-02.md).
- UI → [m6a-05](sprint-m6a-05.md), [m6a-06](sprint-m6a-06.md).

## Tasks

### 1 · Brain loop [X]

Sources: [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (director brain, key lifetime),
[t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (how the interviewer sees code; quiet during Code),
[t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (never-list, brain-authored evaluative speech, distress),
[t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (S1–S4, conduct rules, AI disclosure),
[t6 §13 D29](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict),
[t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 key/catalog row: text needs `interview_brain`; custom ids allowed for text, never proposals),
[ADR-0032 §7](../../adr/0032-realtime-ai-mock-interviewer.md#7-adr-0007-amendments) (key lifetime).

- **Key and model:** `coach.key_default(account, 'interview')` → key + model. No interview default → pre-flight fails
  `interview_key_required` (AB24 links to Settings). [m1-10](sprint-m1-10.md)'s `PUT /keys` (feature=`interview`) already
  accepts either a catalog model with `caps.interview_brain` **or** a valid custom id (its id regex; "custom" = absent from the
  catalog at read time, not a stored flag), and refuses a **known** catalog id without `interview_brain` with 422
  `model_not_interview_capable` — no key-route change here. Pre-flight re-applies that rule against the catalog **at pre-flight
  time** and snapshots the outcome: catalog + `interview_brain` → ok; absent from the catalog → ok with
  `interview.model_is_custom = true` (so m6a-03 never proposes AI bands for this interview, even if the catalog later adds the id);
  in the catalog but without `interview_brain` (the catalog changed since the key was saved) → `preflight_fail(model_access)`.
  A test covers all three.
  The key is decrypted into a `[]byte` owned by the open segment, **zeroed on close / `interrupted` / `paused`**, never in a
  panic value or log (extend the canary test).
- **Pre-flight** (`setup → preflight`): enabled key, model allowed, then a real **16-token probe** on that key/model → `preflight_ok`,
  or `preflight_fail(reason ∈ {model_access, auth, quota, region, key_required})` back to `setup` with AB24's copy. Register the
  `Brain` with m6a-01's service so `start` stops returning 503.
- **`Segment` abstraction** (`segment.go`): `Open(ctx, Prime)`, `Turn(ctx, CandidateInput, sink)`, `Cue(ctx, Cue)`, `Close(reason)`;
  `TextSegment` implements it over `llm.Provider.Stream`; M6b's `VoiceShell` implements the same interface. `Prime` = stable
  instruction prefix + phase state (+ brief on resume) + the last ≤ 4 turns + a cue. Each open writes the `interview_segment` row
  (m6a-01), each close records usage and `cost_micros`.
- **Instruction prefix** `interviewer-frame@1` (a versioned file under `brain/prompts/`, recorded as `prompt_v` on every turn):
  the frame, conduct rules (never ask about age, caste, religion, family, health, disability, nationality or salary history; never
  comment on appearance, voice, accent or emotion; never claim to be human; never state or hint at a score), persona and
  `phase_goals` from the manifest snapshot, the item's **public statement only** (S1), text-mode rules ("say *let me look* rather
  than guess about code you haven't been shown"). **All instruction text is treated as learner-visible** (S4): a lint test asserts
  the prefix and every context contain no pack material, hidden-test data, unreleased hint text or solution facts.
- **Turns:** `POST /interviews/{id}/turns {text ≤ 8 KiB}` — ≤ 1 in flight per interview, ≤ 20 / min → store the candidate turn →
  safety check → brain call (streamed through the gateway: `relayBounded` with m6a-04's turn bounds if it merged, else the chat
  `relayStream` pattern, which m6a-04 replaces) → store the interviewer turn (`truncated` on `max_tokens`/cut stream) → usage → `llm.Cost` → m6a-01's
  `AddSpend` (85 % → wrap-up, 100 % → `finished(cut_short=cap)`). Custom ids are priced at the most expensive catalog
  `interview_brain` price for cap accounting, labelled "estimated".
- **Context per call:** cached stable prefix (Anthropic `Cache: true` on the system block), a one-line state (phase, time left,
  hints released), the **current-screen item** — the latest code snapshot (≤ 16 KiB around the cursor) plus the diff since the
  previous call (D29: one replaceable item, never an accumulating history), fed by `interview.ScreenSink.OnScreen` from m6a-04's
  snapshot intake (define the interface here if m6a-04 hasn't merged; adopt its definition if it has) — and the transcript (whole while ≤ 32 KiB; beyond,
  the opening turns plus the most recent within budget).
- **Server cues** (each one brain call, stored as turns and appended as events): the AI disclosure is the **first turn, fixed text**
  ("I'm an AI interviewer…", never model-generated); a `phase_cue` from m6a-01's `OnPhase` hook; **quiet during Code** except a
  candidate question, a run echo worth probing (`OnRun`, fed by m6a-04's echoes) or a `checkin` after ≥ 90 s × multiplier with no
  turn and no edit; `transcript_full` → wrap-up; entering `wrapping` → the **debrief** (≤ 3 min, 2–3 improvement areas,
  brain-authored, **never a score**) → `debrief_done`. No brain call is triggered by code changes alone in text mode.
- **Delivery:** every turn, cue, hint, hint offer and safety notice is written with `events.Append` (m6a-01) in the same
  transaction as its row (kinds `turn`, `phase`, `hint`, `notice`; S1-filtered payloads — never unreleased hint text or prompts);
  [m6a-04](sprint-m6a-04.md)'s bounded SSE streams them by `seq`. The candidate's own turn gets the reply on its `POST …/turns`
  stream; proactive cues reach the client only through the event stream (until m6a-04 merges, tests read the log).
- **Safety:** a server-side text check on candidate turns (curated distress/self-harm word list) inserts a `safety_card` system turn
  (Tele-MANAS **14416** plus "your local emergency number") and offers a pause; never sent to the model as a judgment, never voice.
  A post-hoc **never-list scan** of interviewer turns (the list m6a-01 exported for the manifest lint) sets
  `interview.review_flag` + a caveat (m6a-03 reads it).

### 2 · Classifier + probe [X]

Sources: [t6 §4 Classifier table](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes),
[t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 §9 P0 typed-errors row),
[ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) (quota never disables), [m4-01](sprint-m4-01.md) (`llm/errors.go`).

- Extend `internal/platform/llm/errors.go` classification (type/code first, status second, message last): OpenAI
  `insufficient_quota`, **`credit_balance_exhausted`**, `project_spend_limit_exceeded`, `organization_spend_limit_exceeded`,
  `organization_usage_limit_exceeded` (in an `error` event, a realtime `response.done{status:failed}` or a failed session creation)
  → `Quota`; Anthropic 400 "credit balance is too low", 402 `billing_error`, spend limit → `Quota`; `model_not_found` / tier /
  permission codes, Anthropic 404 model → `ModelAccess`; add `SessionCap` (`expired`/`session_expired`); `Unavailable` is the
  transient class (5xx, `server_error`, 529, WebSocket 1011). The function takes `(provider, source, status, type, code, message)`
  so M6b's sideband events use it unchanged.
- **Policy** (`internal/coach/interview/classify.go`; `platform/llm` still never retries):

  | Class | Action |
  |---|---|
  | Quota | `interrupt(quota)`: checkpoint, close the segment, grace 5 min; **never disable the key** |
  | Auth | `paused`; key disabled (the existing ADR-0007 path) |
  | ModelAccess / Region | never retry, never disable; at pre-flight → blocked with copy; mid-session → `interrupt(model_access)` → `paused` (resume with another enabled `interview` key) |
  | RateLimited | ≤ 3 retries × 20 s, then `interrupt(rate)` and re-prime with a small context |
  | SessionCap | free re-seed (a new segment primed from our state) |
  | Unavailable | 2 new-segment tries within 30 s, then `interrupt(provider)` |
  | Unknown | ≤ 2 retries, then the **probe** decides |
- **Probe** (`probe.go`): a 16-token text call on the brain model with the same key (≈ $0.0001): quota → `Quota`, success → clear.
  `POST /interviews/{id}/probe` (the grace modal's *Check now*) ≤ 1 / 10 s and ≤ 15 per grace (m6a-01's counters); registered as
  the sweeper's `Prober` for the final probe at `grace_until`. A successful probe within the grace → `cleared` → a new segment
  primed from the **checkpoint plus the verbatim transcript tail** — **no paid summary** (failsafe 1 step 5); ≤ 2 automatic
  re-primes per grace.
- **Tests:** a table test per class for both providers from recorded bodies; a test that reads
  `docs/v2/research/t6-s6-fixtures/index.json` **in place** (a relative path from the package, as the OpenAPI drift test does — no
  copy) and fails if any fixture has no expected class and action; `Quota` never flips `enabled`.

### 3 · `give_hint` + resume brief [X]

Sources: [t6 §5 Hints (S4)](../research/t6-realtime-interviewer.md#5-the-coding-round), [t6 §4 Failsafe 3](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes),
[ADR-0032 §4](../../adr/0032-realtime-ai-mock-interviewer.md#4-session-state-and-failsafes-owner-spec).

- **`give_hint(n)`** (`hints.go`): the item's public, stamped hint ladder. Release paths: the learner's `POST /interviews/{id}/hint`,
  or a **server-policy offer** (a `hint_offer` turn when the current phase has run past its budget × 1.25; at most one offer per
  phase) that the learner accepts. Policy: ≤ 1 conceptual hint per phase, next level only, **never code**. Order: **record the
  `interview_hint` row first**, then insert the `hint` turn, then the text may enter a context. Model output is **never** a control
  channel (no tool calls, no control tokens) — the "director triggers per policy" of t6 §5 is server code. The unreleased ladder
  never appears in any context (test).
- **Resume brief** (`brief.go`; the resume modal's "Summarise & resume — uses your key (~$0.04)"): `POST /interviews/{id}/brief` →
  **probe first**; dry → 409 `still_no_credit`, no charge (`still_dry` → `paused`, same pause). Otherwise one BYO call
  (`store:false`) with schema `interview-brief@1` → `{candidate_summary ≤ 600 chars, progress[≤ 5], next_step, interviewer_brief
  ≤ 1.2k tokens}` from the full transcript so far plus the deterministic state; **cached in `interview_pause.brief`** — a second
  resume attempt in the same pause reuses it and is never charged again. Returned as one JSON (or one SSE event).
  **Schema in the portable subset** ([t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) "Structured
  outputs"; the pattern [m4-02](sprint-m4-02.md)'s judge schemas and Go validator use): Anthropic's `output_config.format` rejects
  `minLength`/`maxLength`, `maxItems`, `minimum`/`maximum` and `pattern`, and the interviewer runs on Anthropic keys too. So
  `interview-brief@1` carries only types, `enum`, `anyOf:[T,null]`, every field required and `additionalProperties:false`; the
  600-char, ≤ 5-item and ≤ 1.2k-token limits are enforced **in Go** after decoding (truncate strings at a rune boundary, drop
  items beyond 5; an over-long `interviewer_brief` is cut to budget). A schema test fails if any of those keywords appears.
- **Re-prime on resume** (`resume_confirmed`): stable prefix + brief + phase state + last ≤ 4 turns + the cue "welcome back in one
  sentence, recap in ≤ 2, continue with ⟨next_step⟩" (within GPT-Live's 128 msg / 8,192-token seeding limit, so M6b reuses it).
- **OpenAI structured output for BYO:** `internal/platform/llm/openai.go` `Complete` currently returns `ErrUnsupported`
  (m4-01, no platform OpenAI use). Add a BYO `Complete` with JSON-schema output and `store:false`; the CI lint that confines
  **platform** credentials to `internal/judge/ai` stays unchanged and green. Anthropic uses `output_config.format`.

### 4 · `store:false` acceptance test [X]

Sources: [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T5 P1 row: an M6a acceptance test),
[t6 §7 inventory](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility).

- A registry `callKinds` = {`preflight_probe`, `turn`, `cue`, `checkin`, `debrief`, `grace_probe`, `brief`} (m6a-03 adds `review`,
  `repropose`, `partial`). Two layers, both per kind and **per provider**: (1) at the adapter boundary, a recording `llm.Provider`
  asserts `llm.Req.NoStore == true` on every request of every kind, whichever provider the key is for (Anthropic has no `store`
  field, so this is the check that covers Anthropic interview calls); (2) on top, the fake OpenAI server asserts `"store": false` in
  every request body. A meta-test fails if a kind in the registry has no covering case for **both** providers, and a grep-style test
  fails if interview code calls a provider outside the registry. Also assert no tools / MCP / container fields (the `llm.Req` shape
  has none).

### 5 · Replay regression suite [X]

Sources: [t6 §9 P0](../research/t6-realtime-interviewer.md#9-phased-plan) ("CI uses recorded fixtures; a manual live replay costs ≈ $0.5"),
[t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) (scrubbed fixtures, public repo), [t6 §14](../research/t6-realtime-interviewer.md#14-risks) (solo QA burden).

- `internal/coach/interview/replay/`: a **scripted synthetic interview** (candidate turns, code snapshots, run echoes, pauses) driven
  through the real brain loop with a fake clock and a fake provider that serves **recorded, scrubbed** responses keyed by call kind
  and order. Golden (`-update`): the turn log (role, kind, seq, truncated), hint ledger, checkpoints, FSM transitions, classifier
  decisions, spend. Scenarios: happy 45-minute path; quota mid-Code → grace → probe success → re-prime with the verbatim tail;
  quota → grace expiry → `paused` → brief → resume; still dry on resume (cached brief, no second charge); rate limit → retries →
  `interrupt(rate)`; transient → 2 tries → `interrupt(provider)`; auth → `paused` + key disabled; `ErrModelAccess` at pre-flight;
  `transcript_full` → wrap-up; distress word → safety card; every S6 quota fixture through the classifier API.
- Runs in `go test ./...` (no network). The **S6 fixtures** are read in place from `docs/v2/research/t6-s6-fixtures/` (one source;
  spk-04 scrubbed them before they were committed); synthetic fixtures for the text path live in `internal/coach/interview/replay/testdata/`.
- **Manual live replay** (documented, not run by the agent): a `-record` mode that runs the script against a real key from an env var
  (≈ $0.5), then scrubs before anything is written; the recipe goes in `docs/runbooks/interviewer.md`. This sprint doesn't run it: the
  owner runs it, or a later prompt that specifies the run and its $ budget does (launching that prompt approves it, D40); no agent uses
  a provider key otherwise.
- **Memory:** replay a 45-minute text interview in compose and record coach's peak RSS (`docker stats`) in the PR; it must sit well
  inside today's 128 Mi limit (flag anything > 96 Mi for [mi-13](sprint-mi-13.md)).

### 6 · Docs [X]

- `docs/architecture/api.md` + `openapi.yaml` (the new `/api/interviews/*` routes; drift test), `services.md` (brain, classifier, probe).
- `docs/runbooks/interviewer.md`: the manual live replay, the classifier table, what the grace/pause copy means.
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) dated update: hint release = learner click or server-policy offer (no
  model control channel); text mode makes no code-triggered calls; custom-id cap pricing; the OpenAI BYO `Complete`.

## Acceptance criteria

- [ ] **Replay suite green**; the **classifier table covers every S6 quota fixture** (the enumerating test fails on an unmapped fixture).
- [ ] `Quota` never disables a key; `Auth` does; `ModelAccess` is caught at pre-flight; the probe arbitrates `Unknown`.
- [ ] **`store:false` asserted on every interview text call kind** — `NoStore` at the adapter boundary for both providers, plus
      `"store": false` in every OpenAI body; no call site outside the registry.
- [ ] Custom model ids work per m1-10's rule (pre-flight: catalog `interview_brain` ok, absent → `model_is_custom`, known without the
      capability → `preflight_fail(model_access)`); the brief schema is in the portable subset with its limits enforced in Go.
- [ ] Hints are recorded before their text reaches any context; the unreleased ladder never appears in a context; the model cannot release a hint.
- [ ] The resume brief is charged at most once per pause (cached); a dry key on resume charges nothing; re-prime uses brief + state + ≤ 4 turns.
- [ ] The AI disclosure is the first turn (fixed text); the debrief never states a score; the never-list scan sets `review_flag`.
- [ ] The key is held only while a segment is open (zeroed on close/interrupt/pause); canary log test green on the new routes.
- [ ] A cohort account completes a full text interview through the API in compose (fake provider); coach RSS recorded.

## Release

**Merge only — ships dark in the next v2.0.x patch** (normally [m6a-06](sprint-m6a-06.md)'s). After this merge, `start` works for the
T-3 cohort through the API (no UI yet), spending only on the cohort member's own key. No new NATS subject, no infra change, no new pod
(memory-sum rule unaffected; coach stays 128 Mi until [mi-13](sprint-mi-13.md)). The release checklist's `coach admin interviews
--live` check applies to every tag that follows.

## Definition of Done

CI green (`go test -race ./...` with the replay suite and classifier/fixture tests, real-PG integration, `sqlc diff` if a query was added,
the canary log test, the OpenAPI drift and route-enumeration tests, the platform-credential lint) · e2e lane green · merged via PR (squash)
· statuses updated (this file + [`../status.md`](../status.md): Sprint board, M6a row "brain merged (dark)", decisions log) · ADR-0032 dated
update · runbook updated.

## Risks / watch-outs

- **Custom model ids:** allowed for the text interviewer (m1-10 accepts them for the `interview` default) but **never** for AI
  proposals — snapshot `model_is_custom` at pre-flight; m6a-03 enforces.
- **Structured-output portability** — a length/count keyword in a schema works on OpenAI and fails on Anthropic; keep limits in Go
  (schema test).
- **Prompt injection** through candidate text or code comments: delimit learner data, treat instructions as learner-visible, never let
  model output drive a server action (hints, transitions, scores).
- **Dry-credit signal differs from the fixtures** (S6 captured only the spend-limit path; prepaid exhaustion is undocumented) — the probe
  arbitrates; keep the Anthropic message matcher per variant with a fixture each.
- **Fixture scrubbing** — the repo is public; a scrub test greps fixtures for key-like strings, `v=0`/`a=candidate` SDP lines and org/project ids.
- **Cost drift** — usage is priced with `llm.Cost`; an unknown price is never 0 (custom ids use the conservative price).
- **Enabling OpenAI `Complete`** must stay BYO-only; the platform-credential lint and the catalog's `platform_allowed` are untouched.
- **Parallel m6a-04** — the event log, `ScreenSink`, the run-echo trigger and the relay are shared seams; whoever merges second adopts
  the first's interface and deletes any stub.
