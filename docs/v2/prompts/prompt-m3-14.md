# Prompt — Sprint m3-14 · judge admission (L9–L15, L6), learner API + DTO allowlist, drafts, arena history/progress, telemetry, judge admin

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-14.md`](../sprints/sprint-m3-14.md)   ·   **Milestone:** M3 (M3-1, judge dark)   ·   **Prereqs:** [m3-06](../sprints/sprint-m3-06.md) (and through it [m3-05](../sprints/sprint-m3-05.md))

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-14.md`](../sprints/sprint-m3-14.md) (the admission table, routes, DTO rules and the migration are spelled out there).
- [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) §2 (contract, DTO allowlist, transport), §4 (arena, D10/D17).
- [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L9–L15), §3 (no alerting; Postgres rows over logs), §5 (TR-QUEUE, TR-STEAL).
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §4 (arena), §5 (leak controls), §6 (erase).
- [t4](../research/t4-judge-contract.md) §2.1–§2.7 (vocabulary, request, **admission**, close, job, **evaluation + DTO**, transport), §2.9 (invariants), §3.7 (arena), §9 (lanes, priorities, caps, sweeper), §11.3, §13 (D14–D19 override the body).
- [t3](../research/t3-sandbox.md) §5.7 (`throttled`, quiet re-run), §5.9 (typed infra errors), **§7.4 (budgets, CE cache, duty-cycle breaker)**, §10 (telemetry).
- [t1](../research/t1-content-data-model.md) §4 judge (tables, PK/FK-only bodies), §6.3 (payload caps and daily quotas).
- [`../rollout-plan.md`](../rollout-plan.md) §4 M3 (ops row: `judge admin` replaces opscheck J checks), §7 (`v1.13.0` row).
- [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (goose + sqlc), [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) (admin CLIs via `kubectl exec`, `admin_audit`).
- Code: `cmd/judge/main.go`, `internal/judge/**` (m3-05/m3-06: store, migrations, queue/lane, graders, contexts, handlers, erase consumer, IDOR scaffold), `internal/platform/httpx/body.go` (`ReadBody`, the typed-413 body helper, m1-05), `internal/identity/admin/` + `cmd/identity/main.go` (the admin-CLI pattern), `internal/judge/testdata/pack` (fixture pack), `internal/e2e/`, `docs/architecture/api.md`.

## Context

M3-1 puts judge on production **dark** in `v1.13.0`. [m3-05](../sprints/sprint-m3-05.md) stood judge up (schema, pack
loader that is Ready with zero evaluable items rather than crash-looping, `XLEARN_IDENTITY` erase consumer, `aud=judge`
auth, kill-switch env); [m3-06](../sprints/sprint-m3-06.md) built the evaluation core (Postgres `SKIP LOCKED` queue, one
runner lane, `code`/`key`/`composite@1` graders, the four contexts, `evaluation_completed`, the internal
`evaluations`/`watermark` endpoints) and tested it in-process. **This sprint adds everything a learner request passes
through**: admission control with typed errors, per-account idempotency, the learner routes behind a DTO allowlist, drafts,
arena history, the D10 arena done marker, runner telemetry and the `judge admin` CLI that replaces opscheck under D34.
The next sprint, [m3-07](../sprints/sprint-m3-07.md), tags `v1.13.0` and deploys judge dark; the gateway only reaches
these routes from `v1.14.0` ([m3-09](../sprints/sprint-m3-09.md), [m3-13](../sprints/sprint-m3-13.md)). Production has one
owner account (D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m3-06](../sprints/sprint-m3-06.md) merged (`git log origin/main`; `internal/judge` has the queue, runner client, graders, contexts and `/internal/contexts/*` routes)
- [ ] m3-05's schema, pack loader, `aud=judge` verification, IDOR scaffold and the `JUDGE_ADMISSION` kill switch (+ `requireAdmission` middleware) are on `main`; m3-06's `Service.Submit/Run/GetSubmission/GetRun`, typed errors and the `onArenaPass` hook too
- [ ] The typed-413 body helper `ReadBody` (`http.MaxBytesReader`) is in `internal/platform/httpx/body.go` ([m1-05](../sprints/sprint-m1-05.md))
- [ ] No open peer PR adds a judge migration (`gh pr list`, `git worktree list`, ListAgents) — judge migrations are serialized m3-05 → m3-06 → m3-14

## Do this (in order)

1. **[X] Branch** `feat/m3-judge-admission` from an up-to-date `main`.
2. **[X] Migration** (plan task 1), next free judge goose version. `submission.{idem_key, body_sha256}` and its unique
   already exist (m3-06) — don't re-add. Add `job.{idem_key, body_sha256}` with a partial unique on Runs (only if m3-06
   didn't), in a separate `-- +goose NO TRANSACTION` file (`CREATE UNIQUE INDEX CONCURRENTLY`); `job_telemetry`;
   `admission_reject` (aggregate, no account id); `breaker_override` (rows kept); `ce_cache` (per account); `admin_audit`. PK/FK only on tables that can hold code or compiler output; `(account_id, created_at)` indexes.
   Queries in `internal/judge/store/queries/{admission,telemetry,arena,admin}.sql`; `sqlc generate`; `sqlc diff` clean.
   Add `job_telemetry` and `ce_cache` to the erase consumer's delete list and its fixture test (`admission_reject` holds no account id).
3. **[X] Admission** (plan task 2): `internal/judge/admission` with `Admit(ctx, tx, req)` called first inside m3-06's
   `Service.Submit`/`Service.Run` transaction, rows 1 → 6 in the plan's order (kill switch/evaluable → idempotency → L6
   caps → close state → step skipping → L9/L10/L11/L12/L13), and the HTTP mapping of m3-06's typed errors
   (`ErrNotEvaluable` → 503, `ErrReplay` → original response, `ErrIdempotencyMismatch` → 422, `ErrContextClosed`/`ErrPartLocked`
   → 409, part caps → 413). m3-05's `requireAdmission` middleware (`JUDGE_ADMISSION`) on every new-work route. Per-account
   `pg_advisory_xact_lock`. Typed JSON errors with `Retry-After`; each rejection bumps `admission_reject` in its own
   statement after the rollback; a `quota{kind, remaining, limit, reset_at}` block on every 2xx (the only implementation:
   m3-09 mirrors it into headers, it adds none). Exempt closes, give-ups
   and re-grades from every row-6 limit. Limits in one `admission.Limits` struct with `JUDGE_LIMITS_*` env overrides and
   the plan's defaults. The kill switch refuses new work only; it never cancels jobs.
4. **[X] Learner routes** (plan task 3): `POST /submissions`, `GET /submissions/{id}` (poll backoff + ETag/304),
   `POST /runs`, `GET /runs/{id}`, drafts `GET`/`PUT`, `GET /arena/{item}/history`, `GET /arena/{item}/submissions/{id}/parts/{part_id}`,
   `GET /arena/progress`, `POST /arena/{item}/studied`, `GET /internal/status?path=` (no JWT; field `admission: open|closed`,
   no `kill_switch` field). `sub`-scoped reads (another account → 404); arena
   context id derived as `uuidv5(sub, item_id)`, client values ignored. Update m3-06's route-table test (learner routes
   behind `requireJWT`). Document routes, `quota{}`, the `/internal/status` shape and error codes in
   `docs/architecture/api.md` (judge section) — **the single source m3-09 reads**; where a peer plan paraphrases it
   differently, `api.md` as merged here wins.
5. **[X] DTO allowlist** (plan task 3): typed structs in `internal/judge/dto/` built by explicit mapping from internal
   records (no embedding, no maps); the CE-in-hidden-file fixed text; generic check keys; key marks/`answer_reveal` only
   with `concluded=true`. Add the reflection golden over every response type and the denylist fixture test.
6. **[X] Arena progress** (plan task 4): fill m3-06's `onArenaPass` hook in the fenced result transaction —
   `first_passed_at` set once, `source=auto`, `submits` = arena submits up to that first pass; Mark studied → `manual`
   only while no pass exists; the progress read (bulk or one item); no outbox row for arena, ever.
7. **[X] Ops surface** (plan task 5): a `job_telemetry` row per runner job from the runner `Result` and the claim timings,
   with the plan's explicit field → column map (`Telemetry.SlotMs` → `busy_ms`, the only L12/L13 input; `BusyMs` →
   `runner_busy_ms`; `StealPct`, `CanaryRatio`, `QuietReruns` → `steal_pct`, `canary_ratio`, `quiet_reruns`; `Versions.*`);
   the per-account 24 h CE cache consulted before dispatch; `judge admin status|breaker|review-flags` dispatched from
   `cmd/judge/main.go` before `run()`, in `internal/judge/admin/`, each verb writing `admin_audit`, `--json` supported,
   no learner code or hidden data in output; `status` reports all three TR-QUEUE legs (counted-submit wait p95, counted
   `queue_full` from `admission_reject`, hours at breaker level ≥ 2 over 7 d).
8. **[X] Tests + verify** (plan task 6): admission table tests (every row, every L-limit, exemptions, idempotency trio);
   kill-switch drain; breaker levels + override; IDOR on every new `{id}` route; arena fence/once test; allowlist +
   denylist; erase fixture; a compose e2e through `POST /submissions` with an Idempotency-Key and a forced typed 429.
   Then `gofmt`, `go vet`, `go test -race ./...`, `npm --prefix web run test` (unchanged), `-tags e2e`, the migration lint,
   `sqlc diff`.
9. **[X] Ship:** see **Ship** below.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** judge touches only schema `judge`.
  It never grades (INV-2), never writes review/assessment, never calls practice here. Arena never alters course state (D10).
- **goose + sqlc:** embedded migrations under the advisory lock, expand only (no `xlearn:contract`), `CONCURRENTLY` in
  its own `NO TRANSACTION` file; commit generated code; **`sqlc diff` must pass**.
- **Outbox/inbox:** the arena upsert and any counted-context outbox row stay in m3-06's single result transaction; no
  new subject, no new durable → **no NATS ACL change** in this sprint.
- **Durable state, not logs (ADR-0035 §3):** telemetry, overrides and audit are Postgres rows; pod logs don't survive a rollout.
- **D34:** no alert, no ping, no opscheck, no healthchecks.io anywhere. `judge admin` is read on demand via `kubectl exec`.
- **Leak rules (ADR-0027 §5, INV-7):** DTOs are an allowlist; nothing from `EVALPACK_DIR`, no hidden inputs, expected
  outputs, per-case timing, ordinals, test names or pack paths ever leaves judge; no handler serves the pack.
- **GitOps:** no infra change here; never `kubectl apply`. Read-only `ssh vps` only if you need to check something live.
- **Memory-sum rule:** no new pod here (judge's pod and limits are m3-07's); keep judge's working set small (no in-memory
  counters, stream large inputs).
- **Parallel sessions:** check peers' PRs, tags and worktrees before numbering migrations; never tag here.

## Deliverables

- judge migration + sqlc queries/gen; erase list and fixture extended.
- `internal/judge/admission/` (L6, L9–L13, L15, idempotency) with table tests.
- Learner routes + `internal/judge/dto/` + allowlist/denylist tests; judge section in `docs/architecture/api.md`.
- `arena_progress` upsert in the result transaction + Mark studied + progress read.
- `job_telemetry` writes, per-account CE cache, `judge admin status|breaker|review-flags` + `admin_audit`.
- Compose e2e through the learner API.

## Update status

- This plan's Status table ([`../sprints/sprint-m3-14.md`](../sprints/sprint-m3-14.md)): each task 🔄 → ✅; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row m3-14 ✅ (merged, ships dark in `v1.13.0`); M3 milestone stays 🔄.
- Decisions log: the provisional L12/L13 thresholds and their env overrides; the per-account CE cache (erase-exact);
  idempotency storage (`submission`/`job` columns, Run keys die with the job); `judge admin` as the D34 replacement for
  opscheck J1, J3–J5 (J2 stays the evalpack CI probe + the PAT-expiry row). Record an ADR only if a call goes beyond
  ADR-0029/0035 (check peers' ADR numbers first).

## Done when (acceptance)

- [ ] Quota and cap tests return **typed 429s** with `Retry-After`; closes and give-ups pass every row-6 limit.
- [ ] The **kill switch** answers new work with **503** while the queue **drains** to terminal evaluations.
- [ ] Typed 413s for body and part caps; idempotent replay returns the original result; a reused key with another body → 422.
- [ ] A passing arena submit upserts `arena_progress` **once**; **Mark studied** sets `manual`; arena writes no outbox row.
- [ ] Allowlist/denylist and IDOR tests green; `judge admin` verbs work and write `admin_audit`; telemetry rows written.
- [ ] Erase covers the new tables; `sqlc diff` clean; CI green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Branch `feat/m3-judge-admission` (step 1); commit(s) `feat(judge): admission, learner API, arena progress, telemetry, admin CLI`; this repo only (no infra PR).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge.
3. **Release action — merge only (ships dark in `v1.13.0`):** Nothing deploys; it ships in `v1.13.0` (cut by [m3-07](../sprints/sprint-m3-07.md)). Don't tag.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in every repo touched (xlearn). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
