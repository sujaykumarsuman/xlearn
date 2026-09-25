# Prompt — Sprint m6a-03 · Assessment deltas + proposal/accept + ScoreMock once + twin fairness gate

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-03.md`](../sprints/sprint-m6a-03.md)   ·   **Milestone:** M6a (text interviewer; ships dark in a v2.0.x patch)   ·   **Prereqs:** [m6a-02](../sprints/sprint-m6a-02.md) (and [m6a-01](../sprints/sprint-m6a-01.md))

## Before you launch (owner)

Launching this prompt attests these are done (D40). If one turns out to be missing, land everything that doesn't depend on it and record the gap as ⛔ in `status.md`; don't wait.

- [ ] Your own provider keys for the catalog `interview_brain` models are set as env vars where this session runs, with at least $20 of spend headroom. Only step 8 (the live twin-gate run, ≤ $20, which launching this prompt approves) needs them; without them 3b goes ⛔ and m6a-06's tag waits for a re-run of that step.

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions and the land-and-sync directive.
- The plan: [`../sprints/sprint-m6a-03.md`](../sprints/sprint-m6a-03.md) — the migration, the `ai-byo` conditions, the submit
  sequence, the twin-gate rules.
- What you build on: [m6a-01](../sprints/sprint-m6a-01.md) (FSM `finished → proposed → scored`, `mock_session_id`, caveats, retention,
  the exported never-list), [m6a-02](../sprints/sprint-m6a-02.md) (brain, probe, `callKinds`, `ScreenSink`), and — if merged —
  [m6a-04](../sprints/sprint-m6a-04.md) (`ensureMockFinals`, the evidence route, `SnapshotAt`, the picker's `contract_hash` pins).
- [ADR-0032 §5](../../adr/0032-realtime-ai-mock-interviewer.md#5-assessment-and-scoring) and its dated updates;
  [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (inputs, `mock_review@1`, validation, review screen, `ai-byo` conditions,
  no auto-accept, fairness gates), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream) (T1/T4 amendment rows),
  [t6 §13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) (D30, D31).
- [t4 §7](../research/t4-judge-contract.md#7-structured-feedback-shape-for-t5) (quote verification, injection heuristic),
  [t4 §6.7](../research/t4-judge-contract.md#67-mock-evidence--scoremock); [m4-02](../sprints/sprint-m4-02.md) (where the verifier landed).
- [ADR-0017](../../adr/0017-mock-model-and-projection-consumer-scaffold.md) (score-once), [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)
  (expand/contract, pending contracts), [rollout §10](../rollout-plan.md#10-public-dashboard-tasks) (P1, P10), [m2-03](../sprints/sprint-m2-03.md) (`/public/stats`).
- The rules your migrations and inserts must satisfy: [m1-02](../sprints/sprint-m1-02.md) task 4 (the migration lint: `DROP CONSTRAINT`
  is contract unless marked `-- xlearn:relax <reason>`; the `NO TRANSACTION` concurrent-index second file), [m1-08](../sprints/sprint-m1-08.md)
  (`mock_session.path_slug` has no default; confirm constraint names with `\d+`), [m3-09](../sprints/sprint-m3-09.md) task 6 (L16's
  counts in assessment `CreateMock`); [t5 §4](../research/t5-platform-ai.md#4-provider-and-model-choice-per-task) "Structured outputs: a
  portable subset" and [m4-02](../sprints/sprint-m4-02.md)'s Go validator (limits in Go, not in the schema); the evidence hand-off in
  [m6a-04](../sprints/sprint-m6a-04.md) task 3 (`ensureMockFinals` callers).
- The frozen boards: `design-system/screens/v2/AB27-debrief-proposal.html` (accept / edit / re-propose once / self, `incomplete`
  free partial view, caveat copy) and `AB13-mock-v2.html` (N dimensions, evidence | sliders).
- Code: `internal/assessment/{mock.go,handlers.go,progress.go,service.go}`, `internal/assessment/store/{store.go,migrations/,queries/mock_session.sql}`,
  `internal/coach/interview/`, `internal/coach/store/{migrations/,queries/}`, `internal/platform/llm/{catalog.go,…}`, the M4 verifier
  (`internal/judge/ai/…` or `internal/platform/llm/…`), `internal/gateway/{interview.go,assessment.go,public.go,public_test.go}`,
  `docs/architecture/openapi.yaml`.

## Context

After [m6a-01](../sprints/sprint-m6a-01.md) and [m6a-02](../sprints/sprint-m6a-02.md), a cohort account can run a text interview to
`finished` through the API, but nothing scores it and assessment doesn't know the new session shapes. This sprint makes it **one honest
score**: assessment gains `open/incomplete/abandoned`, `format`, `time_multiplier`, `caveats` (expand only) and keeps counting only
`scored`; the gateway links the interview to a `format=text` session at start and mirrors terminal states; coach runs one review call on
the learner's key using **public descriptors only**, validates quotes, and holds the proposal until the learner **explicitly** accepts,
edits, re-proposes once or scores it themselves; the gateway then calls **`ScoreMock` once** with `scored_by ∈ {self, ai-byo}`. The
**twin fairness gate** must run live per catalog model before M6a ships (m6a-06's tag). D31 keeps the public profile at a mock count.
[m6a-04](../sprints/sprint-m6a-04.md) runs in parallel and supplies judge evidence; whichever of you merges second wires the seam.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m6a-02](../sprints/sprint-m6a-02.md) merged: a cohort interview reaches `finished` in compose; `callKinds` and the never-list exist.
- [ ] ADR-0032 Accepted; the M4 quote verifier and `llm` catalog are on `main`.
- [ ] [m6a-04](../sprints/sprint-m6a-04.md) status known (`gh pr list`): merged → consume its evidence route, `SnapshotAt`,
      `ensureMockFinals`, pins; open/not started → define `interview.judge_evidence` + `POST /interviews/{id}/evidence` for it to adopt.
- [ ] Parallel sessions (`gh pr list`, `git worktree list`, ListAgents): no open peer PR adds an assessment or coach goose migration or edits
      `ScoreMock` / `internal/gateway/public.go` (else take the next free version at rebase / serialize).

## Do this (in order)

1. **[X] Branch** `feat/m6a-03-mock-scoring` from an up-to-date `main`.
2. **[X] assessment migrations + store** (plan task 1): `0000N_mock_formats.sql` (expand: the new status CHECK added `NOT VALID` +
   `VALIDATE`, then the v1 `mock_session_status_check` dropped on a line marked `-- xlearn:relax widened CHECK` — confirm the name with
   `\d+` first; `format`, `time_multiplier`, `caveats`, `client_ref uuid NULL` with no inline UNIQUE) and `0000(N+1)_mock_client_ref_idx.sql`
   (`-- +goose NO TRANSACTION`, `CREATE UNIQUE INDEX CONCURRENTLY … WHERE client_ref IS NOT NULL`); `POST /mocks` for text/voice (body
   carries `path_slug` and `difficulty`; `set_id = 'interview'`; status `open`, items → `mock_session_item`, snapshot rubric,
   non-authoritative `deadline_at`, `client_ref` idempotency); `POST /mocks/{id}/close`; `ScoreMock` for text/voice (`open` → score,
   `scored` → the stored result as today, `incomplete`/`abandoned` → 409 `mock_not_open`; N dims from `rubric_snapshot`, `scored_by`,
   caveat enum, notes ≤ 2 KiB; classic forced `self`); L16's counts in `CreateMock` restricted to `format='classic'` (text/voice skip
   L16); every aggregate `scored` only; `GET /mocks/live` classic only; `sqlc generate`. Real-PG tests per the plan; the migration lint.
3. **[X] Gateway orchestration** (task 1): start (assessment create with `path_slug`, `difficulty`, `client_ref` → coach
   `start {mock_session_id}`), abandon mirror, lazy `incomplete` mirror on reads, classic `/api/mocks/{id}/score` → 409
   `use_interview_review` for text/voice and `scored_by='self'` always. Tests incl. the partial-failure retry and the mirror lag.
4. **[X] Coach migration** (task 2): `interview_proposal`, `interview.submission`, `interview.review_started_at`, and — only if m6a-04
   hasn't — `interview.judge_evidence` + `POST /interviews/{id}/evidence`; add them to the retention purge and the erase list (coverage
   test green).
5. **[X] Review call + validation** (task 2): `internal/coach/interview/review/` — `mock-review@1` prompt and schema (t5's portable
   subset: `band` as `anyOf:[{enum:[1..5]},null]`, no length/count keywords; the 200-char / ≤ 3 / 8 KiB limits in `validate.go`; a
   schema test). **Trigger exactly as the plan says:** coach starts the review (CAS on `review_started_at IS NULL`) on the **second** of
   `finished` and evidence stored — on `finished` if evidence is already there or every item is `runs:false`, else on the evidence
   intake; the gateway (the only `ensureMockFinals` caller) posts evidence from its finish proxy, the evidence routes and a lazy check
   on `GET /api/interviews/{id}`, `…/active` and `…/proposal` when coach reports `finished` + `evidence_pending` (empty evidence until
   m6a-04 merges). Test: sweeper-driven finish → later read → exactly one review, with evidence. Inputs per the plan (public descriptors, transcript, `SnapshotAt`, aggregate evidence, timeline,
   hint ledger; **no pack material** — pack-canary test); validation with M4's verifier (lift the pure function to `internal/platform/llm`
   if it's judge-internal, no behaviour change), null bands, caps, injection heuristic, deny-lexicon, drill-ref check, unlogged hints;
   per-dimension `ai` modes; custom/un-gated models → quotes only. Add `review`, `repropose`, `partial` to `callKinds` (`store:false`).
6. **[X] Review API + submit** (task 2): `GET /interviews/{id}/proposal`, `…/proposal/repropose` (once, blind), `…/partial-feedback`
   (incomplete only, click, probe first), `POST /interviews/{id}/proposal` (retry after `report_pending=quota`); gateway
   `POST /api/interviews/{id}/submit` with the 3-step sequence (freeze → `ScoreMock` → `scored`), the `ai-byo` rule computed in coach,
   `pending_since` on the active read, `qa_slice` → 409; a `scored` replay from `ScoreMock` counts as step-2 success. Score-once tests:
   double, concurrent, crash window.
7. **[X] Twin gate** (task 3a): synthetic twin fixtures (3 items × base + ASR-noisy + non-native + injected), `twingate.go` comparator
   (median of 3, ±1 band, null mismatch = fail, injection → `review_flag`), CI replay with a planted failure, the catalog
   `mock_review_gate` field and the "un-gated → quotes only" rule with its prompt-bump test; live flags `-models`, `-items`,
   `-budget-usd` with a dry-run estimate, a per-call `llm.Cost` tally and a stop before the crossing call (unit-tested). A model that
   runs and fails is demoted to `self_only`; "red" (the register's ship blocker) means "not run live or the harness failing" (t6 §6).
8. **[X] Twin gate live** (task 3b), pre-approved by launching this prompt (D40): ≤ $20 on the owner's own keys — ≈ 5 models × 12
   transcripts (3 items × 4) × 3 samples ≈ 180 calls ≈ $11–18. First verify the before-launch keys are in the environment (never read
   from coach's store); if they're missing, don't wait: mark 3b ⛔ "owner keys missing" in status.md, record "twin gate run live" as an
   entry gate for [m6a-06](../sprints/sprint-m6a-06.md)'s tag, and land the rest. Otherwise print the harness's dry-run estimate (it
   must fit $20), run `go test -tags twinlive … -run TestTwinGateLive -budget-usd=20`, commit the results (bands and flags only), and
   set each model's gate; a model the budget stop left unfinished stays un-gated and is listed as pending. If the owner asks in-session
   for a smaller budget (≤ $6), run `-items=1` (60 calls) and say so in the ADR-0032 tolerance note.
9. **[X] Public** (task 4): `header.mocks` counts `scored` only (test); extend `public_test.go`'s allowlist with the new fields; authed trend
   and mock views gain `format`, `scored_by`, `caveats` for the learner's own sessions.
10. **[X] Docs** (task 5): `docs/architecture/{data-model,api,events}.md`, `openapi.yaml`; ADR-0032 dated update (tolerance and the
    live run's item count/budget, gate rule, `ai-byo` as built, lazy mirror, the review trigger: `finished` + evidence); ADR-0017 update line.
11. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real PG via `XLEARN_TEST_DATABASE_URL`),
    `go test -tags e2e -race ./internal/e2e/...` (add the leg: cohort interview → finished → proposal → accept → exactly one `ScoreMock`
    and one `mock_completed`), `sqlc diff`, the migration lint, web `typecheck`/`lint`/`test`/`build` (then `git checkout -- web/dist/.gitkeep`).
12. **[X] Ship:** see **Ship** below. No tag.

## Constraints

- **Service boundaries (ADR-0005):** assessment owns `mock_session` and stays the **only writer of the mock signal** (`ScoreMock`);
  transcripts and proposals stay in coach and never reach assessment. The gateway orchestrates; no cross-schema read.
- **Score once:** every step idempotent; the `FOR UPDATE` score-once path in assessment is the backstop.
- **Public descriptors only** in any BYO call; **no pack material**; `store:false` on every call; never log prompts, completions,
  transcripts or quotes.
- **No auto-accept**; no AI call on the learner's key for an `incomplete` interview without a click.
- **goose + sqlc:** expand only (keep `live`; its contract is a later release, recorded as pending); `sqlc diff` clean.
- **Owner keys:** only in step 8's live run (pre-approved by launching this prompt, D40), capped by `-budget-usd=20`; results carry
  bands and flags only (synthetic transcripts).
- **Outbox/NATS:** `mock_completed` unchanged; no new subject (no ACL PR). **GitOps:** no infra change. **D34:** no alerting, no reminder job.
- **Memory-sum rule:** no new pod. **Frontend:** none beyond types (UI is m6a-05/06).
- **Parallel sessions:** check peers' PRs/worktrees before claiming migration or ADR numbers.

## Deliverables

- assessment migration, store and handlers (text/voice create, close, ScoreMock, aggregates, `/mocks/live` classic only).
- Gateway start orchestration, terminal mirror, submit sequence, hardened classic score route.
- Coach `interview_proposal` + `submission` (+ evidence storage if needed), `review/` (prompt, schema, validation), review API.
- Twin-gate harness, fixtures, CI replay, catalog gate rule; live results (or the ⛔ carry-over).
- Public allowlist test extension; docs; ADR-0032 and ADR-0017 updates.

## Update status

- [`../sprints/sprint-m6a-03.md`](../sprints/sprint-m6a-03.md): tasks 🔄 → ✅ (3b ⛔ "owner keys missing" only if the before-launch keys weren't there); _Overall_ ✅ when merged.
- [`../status.md`](../status.md): Sprint board row; **M6a** milestone "scoring merged (dark)"; **pending contracts**: `mock_session.status`
  `live → open`; the twin-gate results per model (or "live run pending — m6a-06 gate"); **Decisions log**: tolerance, the gate rule, the
  `ai-byo` conditions, the lazy mirror, the automatic review.

## Done when (acceptance)

- [ ] A mock scores exactly once (double / concurrent / crash-window tests; a `scored` replay returns the stored result); the classic
      route refuses text/voice and never sends `ai-byo`.
- [ ] The review runs once, with evidence, on the second of `finished` / evidence stored (sweeper-driven finish covered).
- [ ] Both assessment migrations pass the lint; L16 counts classic only; text/voice creates carry `path_slug`.
- [ ] Twin gate green in CI; live results recorded per catalog `interview_brain` model (or carried to m6a-06); un-gated models propose no bands.
- [ ] `ai-byo` only when every condition holds; no auto-accept; re-propose at most once.
- [ ] Public descriptors only (pack-canary test); unverified quotes dropped; missing evidence → `null`; injection/deny-lexicon → `review_flag`.
- [ ] Every aggregate counts only `scored`; `/mocks/live` classic only; the public allowlist covers the new fields.
- [ ] Start idempotent via `client_ref`; abandon mirrored at once, `incomplete` lazily.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m6a-03-mock-scoring` with `feat(assessment): …` and `feat(coach): …` commits (step 8's twin-gate results included); there is no infra change.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in the next `v2.0.x` patch):** Nothing deploys; it ships dark in the next `v2.0.x` patch, normally the M6a patch cut by [m6a-06](../sprints/sprint-m6a-06.md) once the twin gate is green. Don't tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
