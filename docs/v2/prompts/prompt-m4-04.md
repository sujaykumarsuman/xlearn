# Prompt — Sprint m4-04 · Provisional grades, dispute, re-grade, honor claims (D14)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m4-04.md`](../sprints/sprint-m4-04.md)   ·   **Milestone:** M4 (platform AI)   ·   **Prereqs:** [m4-03](../sprints/sprint-m4-03.md) (analyzer) · [m3-08](../sprints/sprint-m3-08.md) (practice attempt states)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m4-04.md`](../sprints/sprint-m4-04.md) — the lock → provisional table, columns, the dispute
  sequence, the fallback table and the test list live there.
- Decisions: [ADR-0029 §3](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer)
  (**D14**: AI results are suggestions; accept, edit, one blind re-grade; 24 h auto-submit; manual fallback) and §5;
  [ADR-0031 §5](../../adr/0031-platform-ai-and-two-tier-keys.md#5-spend-owner-d25) (degrade order, nothing blocks the loop);
  PRD [§5.6](../../prd/xlearn-v2-prd.md) R-AI1, R-AI2.
- Research: [t4 §2.4](../research/t4-judge-contract.md#24-close-give-up-and-dispute-fixes-both-critics-blockers) (close and
  dispute), [§2.7](../research/t4-judge-contract.md#27-transport) (poll fields),
  [§3](../research/t4-judge-contract.md#3-attempt-and-conclusion-lifecycle) (states, ticker, **§3.5** provisional / claim,
  §3.6), [§6.1](../research/t4-judge-contract.md#61-universal-invariants-practice-enforced-property-tested) (invariants),
  [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course) (DSA honor probes); **t4 §13
  (D14–D18) overrides the body.** [t5 §3](../research/t5-platform-ai.md#3-where-the-platform-key-lives-and-secrets)
  (re-grade through gateway → practice → judge), [§7](../research/t5-platform-ai.md#7-quality-calibration-regression-and-injection-defences)
  (item 5: independent re-grade, no auto-submit when flagged), [§10](../research/t5-platform-ai.md#10-what-t5-constrains-downstream)
  (the independent configuration comes with the SD course).
- Rules: [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules),
  [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md) (replay), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md).
- The frozen board `design-system/screens/v2/AB16-ai-suggestion-dispute.html` (its frames define what the routes return).
- Code: `internal/practice/{handlers.go,service.go,ticker.go,judgeconsumer.go,store/store.go,store/queries/attempt.sql,store/migrations/}`
  (the M3 strategies, `Conclude()`, `concludeTouch()`; `judgeconsumer.go` is [m3-08](../sprints/sprint-m3-08.md)'s
  `evaluation_completed` consumer, extended here for `regrade_of` facts), `internal/course/keys` (honor keys), `internal/judge/{handlers.go,store/,ai/,admin/}`,
  `internal/gateway/{bff.go,judge.go,practice.go,dashboard.go,openapi_drift_test.go}`, `docs/architecture/openapi.yaml`,
  `internal/judge/testdata/pack` (the synthetic rubric fixture), `internal/e2e/`, `docker-compose.yml`.

## Context

M3 made practice the single writer of learning signals from judge evidence (seq + gap-free lock + pull reconciler, the
D15 hard timer, `self_grade_pending`). M4 so far added the LLM layer, the Scorer and ledger (m4-01/02) and the analyzer
(m4-03). This sprint makes non-authoritative grades **suggestions** (D14): locks an AI step contributed to, and touches
that fail only on honor-grade public keys, go `provisional`; the learner accepts, edits within the ceiling (`override`,
`trust=honor`), requests **one blind re-grade** (AI) or makes **one claim** (honor keys); untouched suggestions auto-accept
at 24 h unless flagged; AI failures fall back to manual entry, and the loop never blocks. Know the v2.0 reality: **no live
item has an `ai_rubric` step and no independent re-grade configuration is calibrated**, so the AI-provisional and re-grade
paths are proven in the e2e with a fixture rubric item and stay dormant on prod; the **DSA honor-key claim is the live
path**. Everything ships in **v1.16.0** (m4-07 tags).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] m4-03 merged on `main` (`gh pr list --state merged --search "m4-03"`).
- [ ] practice has the M3 lifecycle: `provisional` and `self_grade_pending` in the `attempt.state` CHECK, `attempt_evaluation`,
      the strategies, the guarded `Conclude()`, the ticker and `retry_used` (read `internal/practice/store/migrations/`);
      v1.14.0+ live (`curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz`).
- [ ] m4-02's `ai_rubric` grader is registered on the llm lane and covered by its fixture-item tests, and `llm_calibration`
      rows gate configurations (read `internal/judge/grader` and `internal/judge/ai`; run its unit tests). The end-to-end
      proof is this sprint's own e2e (step 10), not a gate.
- [ ] AB16★ frozen: ds-m4-01 merged (the merge is the freeze), so `design-system/screens/v2/AB16-ai-suggestion-dispute.html` is on `main`.
- [ ] Parallel sessions: `gh pr list`, `git worktree list` and ListAgents show no peer editing practice or judge migrations
      or the gateway route table right now.

## Do this (in order)

1. **[X] Branch** `feat/m4-04-provisional` from an up-to-date `main`.
2. **[X] practice migration** (plan task 1): the columns m3-08 deferred — `candidate_grade`, `settle_deadline` (exposed as
   `accept_deadline_at`), `dispute_used`, `claim_used` — plus `provisional_reason`, `suggestion`, `dispute_authorized_at`,
   `dispute_evaluation_id`, `regrade_evaluation_id`. (`ceiling`, `grade_locked_at`, `retry_used` and
   `attempt_evaluation.regrade_of` already exist; so does judge's `evaluation.regrade_of UNIQUE`.) Next free goose version;
   `sqlc generate`.
3. **[X] Lock → provisional** (task 1), in the one strategy-apply path and in `concludeTouch()`:
   - AI contributed → provisional, unless a deterministic gate fixed the grade;
   - an honor-key-only touch fail that a flip would pass → provisional, **cohort only until GA**. The gateway's touch-start
     proxy sends `{"honor_claims": true}` only when `!judgeCohortOnly || inCohort(info)` (m3-09's constant, deleted by
     ga-01; no new flag). practice snapshots it into `stage_params`, and without it M2's final fail holds. Test: a
     non-cohort touch;
   - mocks and arena never;
   - `settle_deadline` NULL when flagged or low-confidence; `anchor_at` frozen.
4. **[X] Accept / edit + auto-accept** (task 2): `POST /attempts/{id}/accept` (409 `not_provisional`, 422 `above_ceiling`,
   409 `grade_not_editable`; notes/concepts go through review's mistakes PATCH after conclusion, not here); the ticker's
   settle job with **`resolution=auto_accepted`** (m3-08 already uses `timeout` for the D15 deadline Miss); the dev-only
   `PRACTICE_SETTLE_WINDOW` (honoured only with `DEV_AUTH`); the `-race` test.
5. **[X] Claims** (task 3): `POST /attempts/{id}/claim {probe_ids[]}`, once per attempt (409 `claim_used`), missed
   honor-key probes only (422 `not_claimable`); `resolution=claimed`, `trust=honor`; confirm the touch view shows accepted
   answers and `claim_allowed`.
6. **[X] judge re-grade** (task 4): `POST /evaluations/{id}/regrade` (aud=judge, owner by `sub`) with the **synchronous
   pre-check**: kill switch, breaker, a passed `llm_calibration` row with **`task = score_regrade`** for the `rubric@v`
   (the independent configuration is its own row; this sprint's judge migration widens m4-02's `task` enum
   `analyze|score|feedback` with `score_regrade`, additively, and `judge admin calibration record` accepts it), and
   admission. Otherwise it answers 503 `ai_unavailable` / `budget_exhausted{resets_at}` and writes nothing. Then
   **create** `evaluation_dispute` (moved here from m4-02; reason codes
   `missed_evidence | misread_answer | unfair_to_approach | other`) and `judge admin disputes export`; the P0 `regrade` job
   calling m4-02's `Score(purpose = score_regrade)` (independent configuration, k = 5), its result guarded by m3-05's
   `UNIQUE(regrade_of)` and reusing the original's seq; the outbox event; the canary test proving the dispute text never
   reaches a request, log, ledger row or event; erase coverage.
7. **[X] practice dispute side** (task 4): `POST /attempts/{id}/dispute` (authorize: 409 `not_provisional` /
   `not_disputable` / `dispute_used`; idempotent replay; pause the window) and `DELETE /attempts/{id}/dispute` (release an
   authorization with no re-grade fact); the consumer handling `regrade_of` facts (gap check ignores them; new candidate;
   new 24 h window); the 30-minute ticker backstop.
8. **[X] Fallbacks** (task 5): `inconclusive(budget_exhausted | ai_unavailable)` → course `self_grade_pending` with
   `resets_at` in the DTO; touch AI criterion → attestation; `GET /attempts/waiting`; the "AI off" whole-loop test.
9. **[X] Gateway** (task 6): `internal/gateway/grades.go`:
   - accept, claim, dispute (practice authorize → judge re-grade; on judge's 503, release the authorization and pass the
     503 on);
   - the M4 fields **in camelCase** (`acceptDeadlineAt`, `serverNow`, `disputeAllowed`, `claimAllowed`, …, mapped from
     practice's snake_case) on the submission poll and the touch view `GET /api/touches/{attemptId}`;
   - `gradesWaiting` in the dashboard agg;
   - the `{"honor_claims": true}` cohort bit on touch start;
   - presence by config, cache-epoch invalidation, `openapi.yaml`, and the denylist test over the new responses, plus a
     no-snake_case-key test.
10. **[X] Events + tests** (task 7): the new `resolution` values `accepted | regraded | override | claimed | auto_accepted`
    (Go enum); replay golden rows; lifecycle goldens; property tests; the `-tags e2e` run with judge on an `httptest` fake
    provider (m4-01's convention; never a committed compose service), the synthetic rubric fixture item (one `text` part,
    one `ai_rubric` step) under `internal/judge/testdata/pack` (reuse m4-02's if present), seeded `llm_calibration` rows
    with `task = score` and `task = score_regrade`, and AI consent rows.
11. **[X] Docs + ADR** (task 8): `docs/architecture/{events,api,data-model,services}.md`; the ADR "Provisional grades,
    disputes and claims (M4 realization)" — check peers' PRs/worktrees for ADR numbers first, then the next free number.
12. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG integration tests for practice and judge via
    `XLEARN_TEST_DATABASE_URL`), `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, the migration lint, the
    subject-registry test, the openapi drift test, web tests (unchanged). For a manual look, run the compose stack with a
    scratchpad `-f` override pointing judge at a fake provider and exercise the routes with a dev-login token (ADR-0022).
13. **[X] Ship:** see **Ship** below.

## Constraints

- **Service boundaries (ADR-0005):** practice alone decides grades and conclusions (INV-2); judge only produces the
  re-grade evidence; the gateway orders the dispute (practice authorizes first). practice → judge stays read-only
  (`/internal/*` never enqueues work, and **no `/internal/*` route may enqueue llm work** — m4-02's test stays green).
- **goose + sqlc:** next free version, expand only, generated code committed, `sqlc diff` clean; the `resolution` and
  `provisional_reason` enums are Go enums (no CHECK changes); never run `Down` in production.
- **One conclusion:** every path concludes through the guarded `Conclude()` / `concludeTouch()` (`WHERE concluded_at IS NULL
  RETURNING` + `UNIQUE(outcome.attempt_id)`); `anchor_at` is the lock time and never moves.
- **Outbox / inbox:** judge writes the re-grade evaluation and its outbox row in the fenced result transaction; practice
  stores facts in its inbox transaction; no HTTP inside a transaction.
- **Mocks excluded** from every provisional path and the 24 h auto-submit.
- **Money and privacy:** the re-grade goes through `Reserve` (tier `final`); the dispute text is stored for calibration only
  and never reaches the scorer; no prose in events or logs.
- **No infra change** in this sprint (no new subject, durable or caller); if you find you need one, it is its own `../infra`
  PR merged before v1.16.0 (ADR-0035 §2 standing rule) — never `kubectl apply`.
- **D34:** no alerting; the backstop and "grades waiting" are in-app.
- **Memory-sum rule:** no new pod.
- **Parallel sessions:** check peers' PRs, tags and worktrees before the ADR number and before merging migrations.

## Deliverables

- practice: the migration; lock → provisional rules; accept/edit; claims; dispute authorization; re-grade fact handling;
  the ticker's settle job and dispute backstop; `GET /attempts/waiting`; tests (goldens, property, `-race`, real-PG).
- judge: `POST /evaluations/{id}/regrade` with the pre-check; the `evaluation_dispute` table and `judge admin disputes export`;
  the `regrade` job on `Score(purpose = score_regrade)`; the canary test; erase coverage.
- gateway: `internal/gateway/grades.go` routes; camelCase poll/touch-view fields; `gradesWaiting`; the `honor_claims` cohort
  bit on touch start; `openapi.yaml`; denylist coverage.
- The synthetic rubric fixture item (`internal/judge/testdata/pack`), seeded calibration rows and the e2e cases; replay golden rows.
- Docs (`events.md`, `api.md`, `data-model.md`, `services.md`) and the new ADR.

## Update status

- [`../sprints/sprint-m4-04.md`](../sprints/sprint-m4-04.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M4** milestone row stays 🔄 (provisional/dispute merged, ships
  in v1.16.0); the flag inventory gains no new flag, but its m3-09 `judgeCohortOnly` row notes that it also gates the
  honor-key hold (removal GA, ga-01). Add a **Decisions log** line each for:
  - D14 applied to every AI-contributing lock (vs t4's could-gain);
  - one claim naming every missed honor probe;
  - the honor-key hold cohort-only until GA through `judgeCohortOnly` (`stage_params.honor_claims`);
  - flagged → no auto-accept;
  - re-grade facts reuse the seq;
  - the independent configuration as its own `llm_calibration` row (`task = score_regrade`);
  - the synchronous 503 pre-check that releases the dispute (no independent configuration in v2.0);
  - `auto_accepted` vs m3-08's `timeout`;
  - camelCase M4 poll/dashboard fields;
  - the touch AI fallback = attestation;
  - the ADR number.

  Note the two v1.16.0 behaviour changes (cohort-only) for m4-07's release notes.

## Done when (acceptance)

- [ ] **Auto-accept at 24 h** (untouched, unflagged; `resolution=auto_accepted`), anchored at the lock; none when flagged or low-confidence.
- [ ] **One re-grade max**; the re-grade never sees the appeal text (canary test); with no passed independent configuration judge answers 503, the dispute isn't spent, and the learner can accept or edit.
- [ ] Edits bounded by the ceiling and **labelled `override`, `trust=honor`**; deterministic verdicts aren't editable.
- [ ] One honor claim per touch attempt flips only missed honor-key criteria (`resolution=claimed`); the honor-key hold is cohort-only until GA.
- [ ] **The loop never blocks when AI is off** (flag off, breaker, budget, no consent); claims still work.
- [ ] Mocks never enter the provisional path; exactly one conclusion per attempt.
- [ ] `gradesWaiting` in `GET /api/dashboard`; the M4 poll/touch-view fields in camelCase; routes in `openapi.yaml`.
- [ ] Replay goldens equal for the new resolutions; e2e green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here: xlearn only, on `feat/m4-04-provisional`.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships in `v1.16.0`):** Nothing deploys; it ships in `v1.16.0` (cut by [m4-07](../sprints/sprint-m4-07.md)). Don't tag. No infra PR.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
