# Prompt — Sprint m3-09 · gateway judge BFF, DTO allow/deny lists, typed 413, L16, degradation status

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-09.md`](../sprints/sprint-m3-09.md)   ·   **Milestone:** M3 (M3-2 slice, ships dark in `v1.14.0`)   ·   **Prereqs:** [m3-07](../sprints/sprint-m3-07.md) (`v1.13.0` live), [m3-14](../sprints/sprint-m3-14.md), [m3-08](../sprints/sprint-m3-08.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md): repo conventions, stack, and the land-and-sync rule.
- The plan: [`../sprints/sprint-m3-09.md`](../sprints/sprint-m3-09.md). Its route tables, DTO shapes, error codes and acceptance are authoritative for this session.
- [ADR-0029 §2–§4](../../adr/0029-judge-contract-and-learning-signal.md#2-the-contract): the contract (idempotency, server-set context/pin/seq, the learner DTO allowlist, polling, `MaxBytesReader`), conclusion (D15, D16) and the arena (D10, **D17: unrestricted, reveal recorded, no cap**).
- [ADR-0033 §7 and §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) rows 9–11: `aud=judge`, every read scoped to `sub`, IDOR tests, the pack never through the BFF; roles never in the JWT.
- [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service): presence by config, T-1/T-2/T-3, `JUDGE_BASE_URL` as a permanent kill switch.
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory): L6, L9–L11, L13, L15, L16, L24.
- [t4](../research/t4-judge-contract.md): §2.2 (submission request), §2.3 (admission and typed errors), §2.4 (close/give-up), §2.6 (learner DTO + denylist), §2.7 (transport: `poll_after_ms`, ETag, `facts_through_seq`, cache epoch), §3.4, §3.7, §6.4 (concepts withheld), §8 (dock copy: 413/quota), §10 (security), §11.4 #20, and **§13 (D14–D19 override the body)**. [t3 §7.6](../research/t3-sandbox.md#76-throughput-and-latency-inferred-measured-at-a8) (Run poll cadence). [t1 §6.3](../research/t1-content-data-model.md#63-caps-and-quotas-enforced-by-judge-t3t4-own-rate-limiting) (caps).
- [Rollout §4 M3](../rollout-plan.md#4-per-milestone-detail) and [§7](../rollout-plan.md#7-indicative-tag-timeline) (`v1.14.0`).
- Peer plans this builds on: [m1-04](../sprints/sprint-m1-04.md) (`authSession`, `inCohort`), [m1-05](../sprints/sprint-m1-05.md) (`httpx.ReadBody`, typed 413/429, L5), [m1-06](../sprints/sprint-m1-06.md) (`withhold()`, route enumeration), [m3-05](../sprints/sprint-m3-05.md)/[m3-06](../sprints/sprint-m3-06.md)/[m3-14](../sprints/sprint-m3-14.md) (judge), [m3-08](../sprints/sprint-m3-08.md) (practice).
- Code:
  - `internal/gateway/bff.go` (`apiRoutes`, `mintFor`, `proxyPracticeWrite`, `invalidateAgg`, `handleGetProblem`), `gateway.go` (`Options`), `cohort.go`, `withhold.go`, `withhold_routes_test.go`, `openapi_drift_test.go`, `cache.go`;
  - `internal/platform/config/config.go`, `internal/platform/httpx/` (body registry), `cmd/gateway/main.go`, `docker-compose.yml`;
  - `internal/judge/handlers.go` (learner + internal routes), `internal/practice/handlers.go` (close, state, arena reveal);
  - `internal/assessment/handlers.go` (`handleStartMock`), `internal/assessment/store/store.go` (`CreateMock`) + `queries/`;
  - `web/src/lib/mock.ts`, `web/src/screens/Mock.tsx`; `internal/e2e/`.

## Context

- **`v1.13.0`** put judge on prod **dark**: judge with the evalpack mounted, its learner API and admission ([m3-14](../sprints/sprint-m3-14.md)), its erase consumer, and nothing calling it from the SPA. `JUDGE_BASE_URL` is unset on the gateway.
- [m3-08](../sprints/sprint-m3-08.md) made practice the single writer from judge evidence: its consumer and reconciler, the sync close/give-up, the D15/D16/D18 strategies, `arena_revealed_at`.
- **This sprint is the only door between the SPA and judge.** It mints `aud=judge`, resolves every context id server-side, allowlists every response, proves the pack never leaks, adds typed limits and the status the badges need.
- Everything is **present only by config**: `JUDGE_BASE_URL` set **and** the account in the owner/tester cohort (T-3), with the T-1 default `judgeCohortOnly = true` that [ga-01](../sprints/sprint-ga-01.md) flips. Otherwise each route answers the unknown-route 404.
- It merges dark; [m3-11](../sprints/sprint-m3-11.md) and [m3-12](../sprints/sprint-m3-12.md) build the UI on it, and [m3-13](../sprints/sprint-m3-13.md) tags `v1.14.0` and sets `JUDGE_BASE_URL` for the cohort.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] **`v1.13.0` live:** `ssh sujaykumar-vps 'k3s kubectl get deploy -n xlearn -o wide'` shows `xlearn-judge` Ready at `1.13.0`+; `/xlearn/api/v1/healthz` reports `1.13.0`+; [`../status.md`](../status.md) carries m3-07's record (MI-13 ✅ and the evalpack-stream row with the item count; its verify read N evaluable) (**don't** run `judge admin status` for this gate: every `judge admin` verb, reads included, writes an `admin_audit` row on prod, [m3-14](../sprints/sprint-m3-14.md) task 5); the gateway has **no** `JUDGE_BASE_URL` (`ssh sujaykumar-vps 'k3s kubectl get deploy -n xlearn xlearn-gateway -o yaml' | grep JUDGE` is empty).
- [ ] **M3-14 merged:** judge's `POST /submissions`, `GET /submissions/{id}`, `POST /runs`, `GET /runs/{id}` (their 2xx carry `quota`), drafts, `GET /arena/{item}/history`, `GET /arena/{item}/submissions/{id}/parts/{part_id}`, `GET /arena/progress`, `POST /arena/{item}/studied`, the internal `GET /internal/status?path=` (no JWT), admission and its DTO allowlist exist on `main` (`grep -n "HandleFunc\|Handle(" internal/judge/*.go`).
- [ ] **M3-08 merged** (or its PR's contract available to build against), with its task-7 internal contract: `POST /problems/{id}/attempt/start` + `{judge}`, `GET /state/{problemId}` with the M3 fields, `POST /attempts/{id}/close`, the `hint_locked` / `use_give_up` / `evaluated_item` 409s, `POST /problems/{id}/arena-reveal`.
- [ ] **The M1b gateway floor is on `main`:** `authSession`/`inCohort`, `httpx.ReadBody` + `BodyLimit*`, the typed 429/413 envelopes, `withhold()` with the `withholdPolicy` field on `apiRoute`, course resolution.
- [ ] **Parallel sessions:** `gh pr list`, `git worktree list`, ListAgents. No open PR edits `apiRoutes()` or `withhold.go`. [m3-10](../sprints/sprint-m3-10.md) may be open on `internal/gateway/{progress,mistakes,public}.go` and assessment migrations: coordinate, and re-run `sqlc generate` after rebasing on it.

## Do this (in order)

1. **[X] Config and client.** Add `JudgeBaseURL` (env `JUDGE_BASE_URL`, default empty) and `JWT.AudienceJudge` (env `JWT_AUD_JUDGE`, default `judge`) to `internal/platform/config`; wire them through `gateway.Options` and `cmd/gateway/main.go`. Add `judgeClient` (10 s timeout; forwards `Idempotency-Key`, `If-None-Match`, `Retry-After`; `LimitReader` on response bodies only). Set `JUDGE_BASE_URL=http://judge:8087` on the compose gateway.

2. **[X] Presence gate.** In `internal/gateway/judge.go`:
   - `const judgeCohortOnly = true` with a comment naming GA-01 as the flip;
   - `g.judgeFor(w, r)` = `authSession` → `g.judge != nil && (!judgeCohortOnly || inCohort(info))`, else write **exactly** `apiNotFound`'s 404;
   - use it in every judge handler; the rows stay in `apiRoutes()`.

3. **[X] Problem aggregate.** On the course view of `GET /api/problems/{id}`, when present, add `judge{enabled, evaluable, runAvailable, gradingMode, languages[], stageParams{timeLimitS, hintAtS, cleanWithinS, maxFailedForClean}, limits{codeBytes, runInputBytes}}`:
   - `evaluable` (`ok ∧ submit_available`) and `runAvailable` from judge `/internal/evaluable?path=` (no JWT; decode with [m3-05](../sprints/sprint-m3-05.md)'s `internal/platform/judgeapi` wire types; cached 30 s). It carries no languages;
   - `languages[]` from the item's public code part `parts[].config.languages[]` (m1-01's item schema, embedded `internal/course` content); if the aggregate doesn't already carry the public part config, add `parts[{id, type, signature, samples, constraints}]` (the SPA builds starters from `signature`);
   - `gradingMode` from practice's `GET /state/{problemId}` once an attempt exists;
   - `stageParams` from the attempt's pinned `stage_params`, **else** the manifest's `grading.params["verdict_timer@1"]` via `resolveCourse(slug)`, so the AB07 cover renders before Start.
   Absent when not present.

4. **[X] Submission and Run routes** per the plan's task-2 table:
   - `POST /api/problems/{id}/submissions` → judge `POST /submissions`, `POST /api/problems/{id}/runs` → `POST /runs`, `GET /api/submissions/{id}`, `GET /api/runs/{id}`;
   - require `Idempotency-Key` on the POSTs (400 `idempotency_key_required`);
   - the submissions route takes **`action: submit` only**; `final` or `give_up` there → 422 `use_close_route` (every close goes through step 5's route);
   - resolve contexts server-side: `course` → practice's `GET /state/{problemId}` (course attempt only): the open course attempt (409 `no_open_attempt`, 403 `not_enrolled`; a counted submit after `deadline_at + 2 s` → 409 `attempt_expired`); `touch` → practice's `GET /attempts/open?problem_id=` (m1-07), the `attempts[]` entry with `purpose: "touch"` (m2-01) — its `attemptId`, `band` and `deadlineAt` (add `band`/`deadlineAt` there as additive read fields if missing; don't use `GET /touches/{id}`, whose first call sets `probes_shown_at`), refused after `deadline + 5 s` (409 `touch_expired`); `arena` → kind only (judge derives `uuidv5(sub, item)`); `mock` → 422 `invalid_context`; ignore any body `context.id`; send judge `item_id`, `context.id`, `band` (touch) and the pinned `contract_hash`;
   - `httpx.ReadBody(…, BodyLimitDefault)`, then decode `parts[]` as raw JSON values;
   - compose the practice state on counted polls (if `GET /state/{problemId}` lacks `failed_submits`, the hint-opened time, the coach-assist flag, `resolution`, `mistake_hint`, `concepts_hint`, `retry_used` or `pending_reason` (`contract_changed | unreliable | grading_off | other`, derived from stored facts per the plan), add them there as additive read fields); ask judge for `key` marks (`concluded=true`) only when practice says concluded; ETag (sha256 of the composed body) → 304; pass `pollAfterMs` through;
   - `invalidateAgg` only on a 200 poll showing lock/conclusion;
   - pass judge's typed 409/422/429/503 through (incl. `idempotency_key_reused`, `nothing_to_evaluate`, `budget_exhausted`, `shed`) with code, `details` and `Retry-After`, decoded into typed error structs.

5. **[X] Close and give-up.** `POST /api/attempts/{id}/close {action: final|give_up, parts[]}` + `Idempotency-Key`: practice state read (owner/open-or-`self_grade_pending`/pin/purpose/mode) → judge `POST /submissions` with `closes_context=true` → practice `POST /attempts/{id}/close {action, submission_id, close_seq, submitted_at}` (sync) → the composed view + invalidate. A failure after judge's step → 502; a retry with the same key replays. **Retry grading** (AB07 F11) is this route with `{action: final}` from `self_grade_pending` while `retryUsed = false` (practice accepts it once and sets `retry_used`, m3-08 task 3); when `retryUsed` is already true → 409 `retry_used` before judge is called. Pass practice's 409s through: `hint_locked{availableAt}` and `use_give_up` on `…/reveal`, `evaluated_item` on `…/outcome`. On attempt start for a judged item, send `{judge: true}` only after the cohort check, and 409 `ahead_of_schedule` for items past the frontier.

6. **[X] Drafts and arena routes** per the plan's task-4 table: drafts GET/PUT (judge `GET /drafts?…`, `PUT /drafts/{kind}/{id}/{part}`; same caps), arena history list (`?cursor=`) + own-code detail, the batched `GET /api/paths/{slug}/arena/progress`, `POST …/arena/studied`, `POST …/arena/reveal` (practice `POST /problems/{id}/arena-reveal`, then return the solution sections). In the arena view, withhold solution stages for cohort judged items with no concluded course attempt, so the spoiler confirm is the only way in. **No arena locks (D17).**

7. **[X] Allowlist + denylist.** `internal/gateway/judge_dto.go`: decode every judge/practice body into gateway-owned structs and re-encode (never `passthrough()` a judge body, errors included); the practice view struct has exactly the plan's composition fields, incl. `retryUsed` and `pendingReason`. `judge_denylist_test.go`: a poisoned judge fake returning every forbidden key (`expected, args, input, case_id, anchors, key, aliases, seed, gen, test_name, per_test, signals, versions, hidden_files, pack_path`, per-case timings/ordinals, `/evalpack/`, `EVALPACK_DIR`, `HIDDEN-SENTINEL`); walk every judge route × state; `sampleExpected` only in Run/sample views, `answerReveal` only when concluded.

8. **[X] Limits.**
   - `internal/platform/httpx`: `CodePartLimit = 64<<10`, `MultiFilePartLimit = 128<<10` (≤ 16 files), `RunInputLimit = 64<<10`;
   - 413 `part_too_large{partId, max}` / `run_input_too_large` before calling judge (drafts too);
   - `quota{kind, remaining, limit, resetAt}` on every 2xx + `X-Quota-Remaining` / `X-Quota-Reset`: mirror the `quota{kind, remaining, limit, reset_at}` that [m3-14](../sprints/sprint-m3-14.md) already puts on every 2xx of judge's `POST /submissions` and `POST /runs` into the gateway DTO and headers — **no judge edit**;
   - **L16** in assessment `CreateMock` (advisory xact lock per account): live mock for the same `path_slug` with `deadline_at > now()` → 409 `mock_in_progress{mockId}`; > 3 starts in the UTC day → 429 `mock_daily_limit{resetAt}` + `Retry-After`. sqlc queries only, no migration. `web/src/lib/mock.ts` + `Mock.tsx`: resume on 409, error copy on 429.

9. **[X] Degradation status.** `GET /api/judge/status[?path=]` → `{state: ok|saturated|breaker|off, breakerLevel?, evaluable, items?{<id>: evaluable|contract_mismatch|unsupported|invalid|no_pack}}` mapped from judge's internal **`GET /internal/status?path=`** (no JWT, ClusterIP-only — call it without minting a token): `admission: closed` or judge unreachable → `off`, `breaker{level}`, `saturated`, `evaluable`, per-item `contract_mismatch`; plus `/internal/evaluable` for the per-item rest (m3-05's `ok` → `evaluable`, `spec_mismatch` → `contract_mismatch`, `unsupported`/`invalid`/`no_pack` unchanged); 15 s cache; 404 when absent. Not an alert (D34).

10. **[X] `withhold()` + route enumeration.** Concept chips (`conceptsHint`, `concepts`) through `withhold()`; arena Submit/History/Copy rows `applied` (item fields withheld) but never locked — test a Submit and a History read during an open course attempt and a due touch. Every new route row sets its `withholdPolicy` (exempt ones with a reason); extend the behavioural sweep with judge fakes.

11. **[X] Tests.** IDOR (another account's submission/Run/draft/history/close → 404 identical to unknown); presence (unset / `learner` role → 404 everywhere and no `judge` block; `owner`/`tester` → live); close/give-up with an in-flight pass and a replayed key; `final`/`give_up` on the submissions route → 422 `use_close_route`; Retry grading once, then 409 `retry_used`; limits (64 KiB + 1 → 413; each 429 code + `Retry-After`); L16 on real Postgres (concurrent starts; the 4th start). **Compose e2e** in `internal/e2e` with judge + runner + [m3-02](../sprints/sprint-m3-02.md)'s fixture pack: start → Run → Submit WA → Submit pass → `factsThroughSeq ≥ attemptSeq` → concluded; give-up; the denylist walk against the real judge. The real-runner legs use [m3-06](../sprints/sprint-m3-06.md)'s compose profile **`runner`** (`docker compose --profile runner`, build tags `e2e,runner`) and are the acceptance in CI's Linux **`judge-runner-e2e`** job; the runner can't run on macOS, so locally run the legs against the fake runner (`internal/judge/runnerfake`) or the spike's multipass VM. Then `go test ./...`, `sqlc diff`, `npm test` in `web/`, the e2e lane.

12. **[X] Docs.** [`api.md`](../../architecture/api.md) + `openapi.yaml` (routes, error codes, `quota`, ETag/`pollAfterMs`; the drift test must pass); [`services.md`](../../architecture/services.md) (gateway → judge, `aud=judge`, presence by config; the evaluable/status caches join the L24 single-replica list).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)).** The gateway owns no schema and composes only; judge and practice are reached over HTTP with audience-scoped JWTs. No cross-schema reads. judge never computes grades (INV-2); practice stays the single writer.
- **goose + sqlc:** this sprint adds **no migration**. New assessment queries → `sqlc generate`, commit the output; CI runs `sqlc diff`. The only upstream edits allowed are additive **practice** read fields (missing `GET /state/{problemId}` fields incl. `retry_used`/`pending_reason`; the touch entry's `band`/`deadlineAt` on `GET /attempts/open`), with practice's DTO golden updated. **No judge edit** (`quota` and `/internal/status` already exist). Anything bigger is a gap in M3-08/M3-14 — stop and report.
- **Outbox/inbox:** no new event and no new durable, so no ACL PR. If you find you need one, stop: it's a plan change.
- **Allowlist, never pass-through**, for every judge-derived body. The SPA never sends a context id, pin, seq or `submitted_at`.
- **Presence by config** everywhere; `JUDGE_BASE_URL` stays unset on prod (M3-13 sets it after the tag). No SSE.
- **GitOps:** never `kubectl apply`; `ssh sujaykumar-vps` read-only for the entry gates only (`get deploy`, healthz, the gateway env grep — no `judge admin` verb, since each writes an audit row). **No alerting of any kind (D34)**: the status endpoint feeds an in-app badge only.
- **No new always-on pod**, so the memory-sum rule ([ADR-0035 §5](../../adr/0035-v2-operations-nats-auth-limits-capacity.md)) is unchanged; the new gateway caches are small and bounded.
- **Parallel sessions:** check peers' PRs, tags and worktrees (and ListAgents) before merging and before claiming an ADR number.
- This sprint **does not tag**.

## Deliverables

- `internal/gateway/judge.go`, `judge_dto.go`, `arena.go`, `judge_status.go` + tests (`judge_test.go`, `judge_denylist_test.go`, `judge_idor_test.go`); new `apiRoutes()` rows with withhold policies; the `judge` block on the problem aggregate.
- Config (`JUDGE_BASE_URL`, `JWT_AUD_JUDGE`), compose env, the httpx cap registry entries.
- L16 in assessment (queries + handler) and the Mock SPA error mapping.
- Any small additive practice read field the upstream sprints didn't ship (named in the PR).
- The compose e2e legs; `api.md`, `openapi.yaml`, `services.md`.

## Update status

- In [`../sprints/sprint-m3-09.md`](../sprints/sprint-m3-09.md): set each task 🔄 → ✅ (⛔ with a reason), and set _Overall_.
- In [`../status.md`](../status.md):
  - the **Sprint board** row for M3-09 (M3 stays 🔄);
  - the **flag inventory**: add the judge cohort gate (T-3; owner M3; removal GA via GA-01) — `JUDGE_BASE_URL` is already listed as a permanent kill switch;
  - **Decisions log** lines: the L16 definitions (live = unscored and before its deadline; per-account UTC day), the arena-reveal rule (withheld until the spoiler confirm for unconcluded items; recorded, no cap), the ahead-of-schedule, `attempt_expired`, `use_close_route` and `retry_used` codes, the status mapping (`admission: closed` → `off`; `spec_mismatch` → `contract_mismatch`), the `pendingReason` derivation, and every additive practice read field (`GET /state/{problemId}`, `GET /attempts/open`).
- No ADR is expected (the design is ADR-0029/0033/0034/0035). If you depart from them, run the parallel-sessions check before numbering one.

## Done when (acceptance)

- [ ] Denylist test green for every judge route and state (poisoned fake **and** the compose fixture pack).
- [ ] IDOR: another account's submission, Run, draft, history or close → 404, identical to unknown.
- [ ] Typed 413 (`part_too_large`, max 65536) and 429s (`quota_exceeded`, `queue_full`, `too_many_pending`, `too_fast`, `budget_exhausted`, `shed`) reach the SPA with `Retry-After`; `quota.remaining` on every 2xx.
- [ ] L16 enforced race-safely (409 `mock_in_progress`, 429 `mock_daily_limit`).
- [ ] Unset `JUDGE_BASE_URL` or a non-cohort account → every judge route 404 and v1 unchanged.
- [ ] Close/give-up synchronous, idempotent and the only close path (`use_close_route` on the submissions route); Retry grading works once from `self_grade_pending`; the poll view carries `retryUsed` + `pendingReason`; attempt start sends `{judge: true}` only for the cohort and 409s ahead items; a counted submit after `deadline_at + 2 s` → 409 `attempt_expired`.
- [ ] `/api/judge/status` reports `off` (`admission: closed` or unreachable), `breaker`, `saturated`, `evaluable: 0` and per-item `contract_mismatch` / `unsupported` / `invalid` / `no_pack`, from `/internal/status` + `/internal/evaluable`.
- [ ] Arena Submit/History/Copy work during a live attempt and a due touch; the reveal is recorded once and never caps.
- [ ] CI green (`go test ./...`, `sqlc diff`, web tests, OpenAPI drift, route enumeration, compose e2e).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Branch `feat/m3-09-judge-bff`; this repo only (no infra PR).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in `v1.14.0`):** Nothing deploys; it ships in `v1.14.0` (cut by [m3-13](../sprints/sprint-m3-13.md), which also sets `JUDGE_BASE_URL` in its infra PR after the tag). Don't tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
