# Prompt — Sprint m6a-02 · Text brain: loop, classifier, give_hint, resume brief, store:false, replay tests

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-02.md`](../sprints/sprint-m6a-02.md)   ·   **Milestone:** M6a (text interviewer; ships dark in a v2.0.x patch)   ·   **Prereqs:** [m6a-01](../sprints/sprint-m6a-01.md), [spk-04](../sprints/sprint-spk-04.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions and the land-and-sync directive.
- The plan: [`../sprints/sprint-m6a-02.md`](../sprints/sprint-m6a-02.md) — the classifier policy table, hint and brief rules, replay scenarios.
- The core you build on: [`../sprints/sprint-m6a-01.md`](../sprints/sprint-m6a-01.md) (tables, FSM, `Prober`/`OnPhase` hooks, `AddSpend`, the 503 guard).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) §2, §4, §5, §7 (key lifetime) and its dated updates.
- [t6 §3](../research/t6-realtime-interviewer.md#3-architecture--media-path) (director brain, key lifetime),
  [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (the classifier table, failsafes 1 and 3),
  [t6 §5](../research/t6-realtime-interviewer.md#5-the-coding-round) (code visibility, quiet during Code, hints S4),
  [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (brain-authored speech, distress, never-list),
  [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (S1–S4, conduct rules, disclosure),
  [t6 §9 P0](../research/t6-realtime-interviewer.md#9-phased-plan), [t6 §10](../research/t6-realtime-interviewer.md#10-the-smallest-spike) (fixture scrubbing),
  [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (the T5 amendment rows),
  [t6 §13 D29](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict).
- The S6 results: t6 §16 and the scrubbed fixtures `docs/v2/research/t6-s6-fixtures/` (`index.json`) from [spk-04](../sprints/sprint-spk-04.md),
  as folded into ADR-0032 by [ds-m6a-01](../sprints/sprint-ds-m6a-01.md).
- The sibling plan sharing your seams: [m6a-04](../sprints/sprint-m6a-04.md) (event stream over m6a-01's log, `ScreenSink`, `run-echo`, `relayBounded`).
- [ADR-0031 §7](../../adr/0031-platform-ai-and-two-tier-keys.md#7-coach-byo-changes) and [t5 §9](../research/t5-platform-ai.md#9-coach-and-byo-changes-in-v2) (typed errors, quota ≠ disable, `store:false`);
  [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) "Structured outputs: a portable subset" (no
  `maxLength`/`maxItems`/`minimum`/`maximum`/`pattern` in schemas — Anthropic rejects them) and [m4-02](../sprints/sprint-m4-02.md)
  (the judge schemas + Go validator pattern to copy).
- [m4-01](../sprints/sprint-m4-01.md) (`internal/platform/llm`: `llm.go`, `openai.go`, `anthropic.go`, `errors.go`, `catalog.go`, `prices.go`, `redact.go`) and
  [m1-10](../sprints/sprint-m1-10.md) (key crypto, `key_default`, and its `interview` default rule: a catalog `interview_brain` model or
  a valid custom id; a known catalog id without the capability → 422 `model_not_interview_capable`).
- Code: `internal/coach/interview/` (m6a-01), `internal/coach/{handlers.go,providers.go,service.go}`, `internal/platform/llm/`,
  `internal/coach/store/queries/`, `internal/gateway/{interview.go,coach.go}` (`relayStream`) and m6a-04's `internal/gateway/sse.go`
  (`relayBounded`) if merged, `docs/architecture/openapi.yaml`.
- The frozen boards for copy: `design-system/screens/v2/AB24-mock-setup-preflight.html` (pre-flight errors),
  `AB25-live-hud-text.html` (turns, hint, safety card), `AB26-grace-paused-resume.html` (grace, still no credit, resume brief).

## Context

[m6a-01](../sprints/sprint-m6a-01.md) merged the interviewer's durable core in coach: the FSM, clock, sweeper, caps, retention and
`/api/interviews/*` — with **no model call**; `start` returns 503 `interviewer_unavailable` until a `Brain` is registered. This sprint
registers the **text brain** on the learner's `interview` key: pre-flight probe, the voice-ready `Segment` abstraction (text impl),
turns and server cues (disclosure first, phase cues, check-ins, debrief), the classifier by `error.code` with the same-key probe as
arbiter (quota never disables a key), `give_hint` recorded first, the resume brief cached per pause, a `store:false` test on every
call, and replay regression tests from recorded, scrubbed fixtures. Scoring ([m6a-03](../sprints/sprint-m6a-03.md)) and judge runs /
bounded SSE ([m6a-04](../sprints/sprint-m6a-04.md), possibly in flight in parallel) are not here. Everything stays dark (cohort only).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m6a-01](../sprints/sprint-m6a-01.md) merged on `main` (`internal/coach/interview/` exists; `start` returns 503 without a brain).
- [ ] S6 fixtures merged under `docs/v2/research/t6-s6-fixtures/` (with `index.json`) and t6 §16 readable.
- [ ] `internal/platform/llm` with typed errors, catalog `caps.interview_brain`, `Cost` and `NoStore` is on `main`.
- [ ] Parallel sessions (`gh pr list`, `git worktree list`, ListAgents): if [m6a-04](../sprints/sprint-m6a-04.md) is open or merged,
      adopt its `ScreenSink`, `run-echo` → `OnRun` wiring and `relayBounded` rather than duplicating them; no peer PR edits
      `internal/platform/llm/{errors,openai}.go`.

## Do this (in order)

1. **[X] Branch** `feat/m6a-02-text-brain` from an up-to-date `main`.
2. **[X] Classifier** (plan task 2): extend `internal/platform/llm/errors.go` with the T6 codes (`credit_balance_exhausted`, the
   spend/usage-limit codes, realtime `response.done{failed}` extraction, Anthropic credit/billing, `SessionCap`) keeping "type/code,
   then status, then message"; `internal/coach/interview/classify.go` with the plan's policy table; table tests for both providers; the
   test that reads `docs/v2/research/t6-s6-fixtures/index.json` **in place** (relative path, no copy) and fails on an unmapped fixture.
3. **[X] Probe** (task 2): `probe.go` (16-token call, same key/model); `POST /interviews/{id}/probe` (≤ 1 / 10 s, ≤ 15 per grace);
   register it as the sweeper's `Prober`; success within the grace → `cleared` → re-prime from checkpoint + verbatim tail (no paid
   summary; ≤ 2 automatic re-primes per grace).
4. **[X] Segment + brain** (task 1): `segment.go` (`Segment`, `TextSegment`), `brain/` with `interviewer-frame@1`, the context
   builder (cached prefix, one-line state, current-screen item ≤ 16 KiB + diff fed by `ScreenSink.OnScreen` — adopt m6a-04's
   interface or define it for m6a-04 to adopt — transcript within 32 KiB), key decrypted per open
   segment and zeroed on close/interrupt/pause; pre-flight (`preflight_ok`/`preflight_fail`) re-applying m1-10's model rule against
   the catalog at pre-flight time (catalog `interview_brain` → ok; absent → ok, snapshot `model_is_custom = true`; known without the
   capability → `preflight_fail(model_access)`; test all three — no change to m1-10's key route); register the `Brain` (the 503 goes away).
5. **[X] Turns + cues** (task 1): `POST /interviews/{id}/turns` (≤ 8 KiB, ≤ 1 in flight, ≤ 20/min, streamed); gateway proxy on
   `relayBounded` if m6a-04 merged, else `relayStream`; every turn, cue, hint and notice also written with m6a-01's `events.Append`
   (S1-filtered) in the same tx; the fixed-text disclosure first; `OnPhase` cues,
   check-ins, `OnRun` hook (fed by m6a-04's echoes), wrap-up on `transcript_full`, the debrief on `wrapping` → `debrief_done`;
   usage → `llm.Cost` → `AddSpend` (custom ids at the conservative price); safety card on distress words; never-list scan →
   `review_flag`. S1/S4 lint test on every context.
6. **[X] Hints + brief** (task 3): `hints.go` (learner request or a server-policy `hint_offer` accepted; record `interview_hint` first,
   then the turn, then context; never code; ≤ 1 per phase); `brief.go` (`POST /interviews/{id}/brief`: probe first → 409
   `still_no_credit`; else `interview-brief@1` structured call, cached in `interview_pause.brief`; the schema in t5's portable subset
   with the 600-char / ≤ 5-item / ≤ 1.2k-token limits enforced in Go, plus a schema test that forbids the length/count keywords); resume re-prime (prefix + brief +
   state + last ≤ 4 turns + cue); BYO `Complete` with JSON schema + `store:false` in `llm/openai.go` (platform lint untouched).
7. **[X] `store:false`** (task 4): the `callKinds` registry; per kind and per provider, a recording `llm.Provider` asserting
   `llm.Req.NoStore == true` at the adapter boundary (this is what covers Anthropic keys), plus the fake OpenAI server asserting
   `"store": false` in each body; the coverage meta-test (both providers) and the no-unregistered-call-site test.
8. **[X] Replay suite** (task 5): `internal/coach/interview/replay/` with the plan's scenarios, golden with `-update`, fake clock and
   fake provider; the `-record` mode (env key, scrub before write) documented but **not run** in this sprint (the owner runs it, or a
   later prompt that specifies the run and its $ budget); the fixture scrub test. Replay a 45-minute interview in compose and record coach's peak RSS in the PR.
9. **[X] Docs** (task 6): `docs/architecture/{api,services}.md`, `openapi.yaml`, `docs/runbooks/interviewer.md` (live replay recipe,
   classifier table), ADR-0032 dated update (hint release rule, no code-triggered calls in text, custom-id cap pricing, BYO `Complete`).
10. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real PG via `XLEARN_TEST_DATABASE_URL`),
    `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, web `typecheck`/`lint`/`test`/`build` (then
    `git checkout -- web/dist/.gitkeep`); a compose run: cohort account → create → consent → start → turns → pause → brief → resume →
    finish (fake provider).
11. **[X] Ship:** see **Ship** below. No tag.

## Constraints

- **Service boundaries (ADR-0005):** all interview state stays in schema `coach`; the gateway only proxies/relays.
- **The raw key never leaves coach**; held as a `[]byte` per open segment and zeroed; never logged, never in a panic value (canary test).
- **C4 data:** never log prompts, completions, transcripts or code; provider error bodies are parsed with a size cap and never echoed.
- **`platform/llm` does no retries**; retries and backoff live in the interview policy. Quota never disables a key.
- **Model output is data, never a control channel.** No tool calls, no control tokens; hints, transitions and spend are server code.
- **No AI call on the learner's key without their action**, except inside an interview they started (turns, cues, debrief); the brief and
  probes on resume run only on a click.
- **Fixtures are scrubbed before commit** (the repo is public); no agent uses a real provider key in this sprint (a live run needs a
  prompt that specifies it and its $ budget, D40).
- **goose + sqlc** if a query or column is added (next free coach version); `sqlc diff` clean.
- **Outbox/NATS:** none added (no ACL PR). **GitOps:** no infra change; never `kubectl apply`. **D34:** no alerting.
- **Memory-sum rule:** no new pod; coach limits unchanged (record RSS). **Frontend:** none.
- **Parallel sessions:** check peers' PRs/worktrees before claiming a migration or ADR number.

## Deliverables

- `internal/coach/interview/{segment.go,classify.go,probe.go,hints.go,brief.go,safety.go}` and `brain/` (prompts `interviewer-frame@1`,
  `interview-brief@1`); the registered `Brain`.
- `internal/platform/llm/errors.go` classification additions; BYO `Complete` in `openai.go`.
- Coach + gateway routes: turns (streamed), hint, probe, brief; their events in m6a-01's log; OpenAPI entries.
- `store:false` registry + tests; the replay suite with S6 + synthetic fixtures and goldens; the fixture scrub test.
- Runbook and ADR-0032 dated update; coach RSS measurement in the PR.

## Update status

- [`../sprints/sprint-m6a-02.md`](../sprints/sprint-m6a-02.md): tasks 🔄 → ✅; _Overall_ ✅ when merged.
- [`../status.md`](../status.md): Sprint board row; **M6a** milestone "brain merged (dark)"; **Decisions log**: hint release rule, no
  code-triggered calls in text mode, custom-id cap pricing, BYO `Complete` for OpenAI, the coach RSS figure (for mi-13's sizing).

## Done when (acceptance)

- [ ] Replay suite green; the classifier table covers every S6 quota fixture.
- [ ] Quota never disables a key; Auth does; ModelAccess caught at pre-flight; the probe arbitrates Unknown.
- [ ] `store:false` asserted on every interview text call kind (`NoStore` at the adapter for both providers + the OpenAI body check).
- [ ] Custom ids follow m1-10's rule at pre-flight; the brief schema is portable with its limits enforced in Go.
- [ ] Hints recorded before their text reaches a context; the model cannot release a hint.
- [ ] The resume brief is charged at most once per pause; a dry key on resume charges nothing.
- [ ] Disclosure first (fixed text); the debrief never states a score; never-list scan sets `review_flag`.
- [ ] Key held only while a segment is open; canary log test green.
- [ ] A cohort account completes a full text interview through the API in compose; coach RSS recorded.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m6a-02-text-brain` with `feat(coach): …` commits; there is no infra change.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in the next `v2.0.x` patch):** Nothing deploys; it ships dark in the next `v2.0.x` patch, normally the M6a patch cut by [m6a-06](../sprints/sprint-m6a-06.md). Don't tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
