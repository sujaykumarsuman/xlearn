# Prompt — Sprint m3-08 · practice: judge consumer, reconciler, D15/D16/D18 grading strategy

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m3-08.md`](../sprints/sprint-m3-08.md)   ·   **Milestone:** M3 (M3-2, Run/Submit)   ·   **Prereqs:** [m3-07](../sprints/sprint-m3-07.md) (`v1.13.0` live) and, on `main`, [m3-06](../sprints/sprint-m3-06.md), [m2-01](../sprints/sprint-m2-01.md), [m1-07](../sprints/sprint-m1-07.md), [l-01](../sprints/sprint-l-01.md), [m3-01](../sprints/sprint-m3-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m3-08.md`](../sprints/sprint-m3-08.md) — columns, the D18 golden table, the consumer steps, the ticker jobs and the internal contract are spelled out there.
- [ADR-0029](../../adr/0029-judge-contract-and-learning-signal.md) §2 (submission, `attempt_seq`, counted), **§3 (conclusion: facts, gap-free lock, hard time limit, give-up, DSA defaults, a pass finishes)**, §4 (revision, mistake pre-fill, arena), [ADR-0026](../../adr/0026-per-course-extensibility-model.md) (judge produces evidence, practice decides).
- [t4](../research/t4-judge-contract.md) **§13 first (D14–D19 override the body)**, then §2.1, §2.4 (close/give-up), §2.8 (events, durables), §2.9 (INV-1…14), **§3.1–§3.7 (states, consumer + ticker, transitions, give-up, touches/arena)**, §5.4 (strategy interface), **§6 (invariants, strategies, pre-fill, concepts, D2)**, §10 (security), §11.4 (amendments), §14 (risks).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §2 (T-2 kill switches incl. the grading override, T-3 cohort, presence by config), §3 (append-only envelope, consumers before producers, ACL before tag).
- [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §1 (topology, dead-letter hook, registry test), §2 (fine ACLs, reload, standing rules), §3 (D34).
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §12 row 10 (practice → judge `/internal/*` behind MI-5a; role never in the JWT).
- [t1 §4](../research/t1-content-data-model.md) (practice M3 columns, inbox), [`../rollout-plan.md`](../rollout-plan.md) §4 M3, §7 (`v1.14.0`).
- Peer plans: [m2-01](../sprints/sprint-m2-01.md) (M2a columns, guarded `Conclude()`, `concludeTouch()`, `ticker.go`, `reviewclient.go`), [m1-07](../sprints/sprint-m1-07.md) (`coach_assist_at`, `assist{}`), [m1-01](../sprints/sprint-m1-01.md) (manifest `grading.params`, golden), [mi-05](../sprints/sprint-mi-05.md) (topology, dead letters, registry), [m3-06](../sprints/sprint-m3-06.md) (internal endpoints), [m3-09](../sprints/sprint-m3-09.md) (the gateway side of task 7).
- Code: `cmd/practice/main.go`, `internal/practice/{config.go,service.go,handlers.go,ticker.go,reviewclient.go}`, `internal/platform/judgeapi/` (m3-06: client, fixtures, practice-side fake), `internal/practice/store/{store.go,migrations/,queries/}`, `internal/platform/events/{topology.go,consumer.go,relay.go}`, `internal/course/` (manifest, `canon`), `curriculum/courses/dsa/course.json` (the DSA manifest; task 2 switches its `grading.strategy`), `internal/judge/handlers*.go` (the internal routes), `internal/e2e/`, `docker-compose.yml`, `docs/architecture/{api,events,data-model}.md`.
- Infra (via PR only): `../infra/infrastructure/messaging/release.yaml`.

## Context

`v1.13.0` put judge on production dark: it can evaluate, emits `xlearn.judge.evaluation_completed` for counted course
and touch evaluations, and serves read-only internal endpoints (`evaluations?seq_lt=T`, `watermark`, `evaluable`) — but
nothing reaches it until `v1.14.0` sets `JUDGE_BASE_URL`. **This sprint makes practice the single writer of learning
signals from that evidence** (ADR-0026/0029): practice's first judge consumer (always ack, inbox, gap-free
`attempt_seq`), a pull reconciler and deadline jobs on the M2a ticker, the synchronous close and give-up, and the owner's
D15/D16/D18 method for judged DSA attempts — **45:00 hard limit, hint at 15:00 caps Assisted, coach help caps Assisted,
Clean = pass ≤ 20:00 with ≤ 3 failed submits, a pass finishes the attempt (no re-implement), reveal = give-up = Miss,
timeout = Miss dated at the deadline**. The **self path stays exactly as today** (golden = v1). [m3-09](../sprints/sprint-m3-09.md)
builds the gateway side in parallel; [m3-10](../sprints/sprint-m3-10.md) consumes the new event fields; both ride
`v1.14.0` with this sprint. Production has one owner account (D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] `v1.13.0` live (m3-07): judge Ready dark; `XLEARN_JUDGE` exists (read-only `/jsz` via `host-verify --cluster` or the NATS monitor proxy)
- [ ] On `main`: judge's `/internal/contexts/{kind}/{id}/evaluations`, its watermark (preferred: `/internal/contexts/{kind}/{id}/watermark` for course + touch from m3-06; else touch-only and step 6 generalizes it), `/internal/evaluable`; the fixture pack
- [ ] On `main`: m2-01 (attempt states incl. `self_grade_pending`/`voided`, guarded `Conclude()`, `ticker.go`), m1-07 (`coach_assist_at`), l-01 (practice erase consumer, `event_dead_letter`), m3-01 (`internal/course/canon`), m1-01 (D18 params in the manifest)
- [ ] No open peer PR adds a practice migration (`gh pr list`, `git worktree list`, ListAgents)
- [ ] If m3-09 runs in parallel: agree task 7's internal contract with that session first (ListAgents / its PR)

## Do this (in order)

1. **[X] Branch** `feat/m3-practice-judge` from an up-to-date `main`.
2. **[X] Schema** (plan task 1): the attempt columns (`grading_mode`, `contract_hash`, `language`, `within_timer`,
   `failed_submits`, `counted_submits`, `first_submit_passed`, `grade_locked_at`, `close_*`, `gave_up_at`,
   `released_closes`, `retry_used`, `facts_through_seq`, `facts_gap_since`, `ceiling`),
   `user_problem_state.arena_revealed_at`, `attempt_evaluation`, `inbox` (and `event_dead_letter` only if missing). No
   re-implement columns (D16), no provisional columns (M4). Erase list + fixture extended. Queries; `sqlc generate`;
   `sqlc diff` clean.
3. **[X] Strategies** (plan task 2): `internal/practice/grading/` — the pure `Strategy` interface, `verdict_timer@1`
   with the manifest's D18 params, `weighted_gate@1` (fixtures), `self@1{cap:none|stage}`; the **19-row D18 golden
   table**; `self@1{cap:none}` rows copied from today's store tests; invariant property tests I1–I5, I7, I8 with **no
   arena cap** (D17). Switch DSA's `grading.strategy` to `verdict_timer@1` and update `internal/course/golden_test.go`
   citing D18.
4. **[X] Start, hint gate, close, INV-9, arena reveal** (plan task 3): the judged start (`{judge: true}` from the gateway +
   `JUDGE_BASE_URL` + no override + judge `evaluable` ok for practice's own `contract_hash`; one 45:00 timer; D18
   `stage_params`); `reveal` → 409 `hint_locked{available_at}` before 15:00 and 409 `use_give_up` for the solution;
   `POST /attempts/{id}/close` (sync, idempotent on `submission_id`; give-up unlocks the solution at once and writes
   `solution_revealed_early`; released `final` handling; one retry from `self_grade_pending`); `outcome` → 409
   `evaluated_item` unless self / `self_grade_pending` (pick ≤ `ceiling`); `POST /problems/{id}/arena-reveal` (recorded,
   never caps). No re-implement stage on the judged path.
5. **[X] Consumer** (plan task 4): `internal/practice/judgeconsumer.go` — durable on `XLEARN_JUDGE`,
   `xlearn.judge.evaluation_completed`, `DeliverAll`, dead-letter sink; declared in `topology.go` with its registry entry;
   started in the background. The seven steps in one transaction; **always ack** except on a DB error. Touch facts turn
   `correct_in_timer` into checked evidence for packed items. `RELAY_INTERVAL` (default 1 s) for practice's relay.
6. **[X] Ticker** (plan task 5): m3-06's `internal/platform/judgeapi.Client` behind a nil-safe 5 s wrapper (tests use its
   exported practice-side fake and `evaluation_completed.v2.json`) and the jobs — gap reconciler
   (`source=pull`, invalid → non-passing filler + ERROR), touch watermark drain, **course deadline → drain, then Miss at
   the deadline**, kill-switch sweep (→ `self_grade_pending` / voided touches, never Miss), WARN-only stale reads plus
   `docs/v2/runbooks/practice-judge-lifecycle.md` with the three SQL reads. Select → call → lock → re-check; no HTTP under
   a row lock. The course deadline drain needs `/internal/contexts/{kind}/{id}/watermark`: if m3-06 shipped it for both
   kinds, just consume `judgeapi.Client.Watermark(kind, id)`; if it shipped touch only, generalize it here (read-only, no
   migration) in the judge files the plan names (`internal/judge/handlers.go`, the watermark query if kind-filtered,
   `internal/platform/judgeapi` + `testdata/watermark.json` + the practice-side fake, m3-06's contract test), after
   checking no peer PR touches them, and record that this session touched judge.
7. **[X] Events** (plan task 6): `problem_solved` v2 additions (skip fields m2-05/m1-07 already added), `mistake_hint`
   via `mistakes.prefill` (fixture table in tests; DSA rows come in m3-10), `concepts_hint[≤3]`, `touch_concluded`
   `evaluation_ids[]` (+ hints), `attempt_logged` course-only with `path_slug` + `anchor_at`. Update `docs/architecture/events.md`.
8. **[X] Gating + contract** (plan task 7): `JUDGE_BASE_URL` and `GRADING_OVERRIDE` in `internal/practice/config.go`
   (use the override's name if a peer already recorded one in status.md); the `GET /state` additions matching m3-09's poll
   composition allowlist (`started_at`, `deadline_at`, `hint_available_at`, `hint_opened_at`, `coach_used_at`, `failed_submits`,
   `resolution`, `mistake_hint`, `concepts_hint`, …); document
   every internal route in `docs/architecture/api.md`; set practice's `JUDGE_BASE_URL` in `docker-compose.yml`.
9. **[X] Tests + verify** (plan task 9): strategy goldens + properties; lifecycle goldens (pass, give-up × in-flight pass
   both orderings, give-up → Miss, timeout → Miss after drain, released ×3, stale contract, bounded pick, INV-9, sweep);
   replay golden through review + assessment in compose; consumer edge cases; **reconciler heal in compose** (`-tags e2e`,
   a hook drops one event); touch drain; `-race` exactly-once; **self path = v1**. Then `gofmt`, `go vet`,
   `go test -race ./...`, web tests, `-tags e2e`, the migration lint, `sqlc diff`, topology/registry/golden-ACL tests,
   the NATS-auth integration test.
10. **[X] PR** → conventional commit(s) `feat(practice): judge consumer, reconciler, D15/D16/D18 strategies` with the
    attribution lines → CI green → squash-merge. **No tag** (m3-13 tags `v1.14.0`).
11. **[I] ACL PR** (plan task 8): `make nats-acl-render` on `main` after the merge; the diff vs the live `messaging` block
    must be only practice's new `XLEARN_JUDGE` durable grants; paste; no annotation bump (reload); merge; verify the
    reload, no `legacy`, no permission-violation logs, `host-verify --cluster --nats-stage=n3` (or `n4`). If mi-11's
    `xlearn` egress policies are live, confirm practice → `xlearn-judge:8087` is allowed (else a separate infra PR before `v1.14.0`).

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** practice touches only schema
  `practice`; judge data arrives by event or by judge's internal API — never a cross-schema read. judge never grades
  (INV-2); practice never evaluates code.
- **goose + sqlc:** embedded, expand-only migrations (no `xlearn:contract`); commit generated code; **`sqlc diff` must pass**.
- **Outbox/inbox:** the fact, the lock, `Conclude()` and the outbox rows in **one** transaction; the inbox dedupes on
  `event_id`; always ack; poison → dead-letter row (mi-05), never a silent drop.
- **Exactly one conclusion** (INV-9): the guarded `UPDATE … WHERE concluded_at IS NULL RETURNING` + `UNIQUE(outcome.attempt_id)`.
- **Timers apply to `submitted_at`** (INV-6); queue latency never penalises the learner.
- **Envelope append-only; decoders forever**; consumers before producers (judge stays dark until `v1.14.0` + `JUDGE_BASE_URL`).
- **ACL PR before the tag** (ADR-0035 §2): merge it in this sprint; it's its own infra PR, never folded into a tag.
- **GitOps:** never `kubectl apply`; read-only `ssh sujaykumar-vps` for verification.
- **D34:** no alert, no ping, no opscheck — WARN/ERROR logs and Postgres rows read on demand.
- **Memory-sum rule:** no new pod; practice's limits unchanged (note its RSS after the change in status.md if it moves).
- **Parallel sessions:** check peers' PRs, tags and worktrees before numbering the migration; coordinate the internal
  contract with m3-09; never tag here.

## Deliverables

- practice migration + sqlc; `internal/practice/grading/` with goldens and property tests; the DSA manifest strategy switch + golden update.
- Judged start, hint gate, `POST /attempts/{id}/close`, INV-9, arena-reveal record; `GET /state` additions.
- `internal/practice/judgeconsumer.go` (+ topology/registry entry), the `judgeapi` wrapper, the extended `ticker.go`.
- Event field additions; `docs/architecture/{api,events,data-model}.md` updated; `docs/v2/runbooks/practice-judge-lifecycle.md`.
- Compose/e2e tests incl. the reconciler heal; judge's watermark generalized to `{kind}` (course + touch) — only if m3-06 didn't ship it.
- infra PR: practice's `XLEARN_JUDGE` durable in the NATS ACL.

## Update status

- This plan's Status table ([`../sprints/sprint-m3-08.md`](../sprints/sprint-m3-08.md)): each task 🔄 → ✅; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row m3-08 ✅ (merged, ships in `v1.14.0`); M3 milestone stays 🔄;
  **flag inventory**: `GRADING_OVERRIDE` (permanent kill switch, practice) and practice's `JUDGE_BASE_URL` (kill switch,
  set with the gateway's in m3-13); NATS rows: the practice-durable ACL PR and its reload date.
- Decisions log: deadline Miss applies to judged attempts only (self path = v1 until M5); kill switch → `self_grade_pending`
  / voided touches, never Miss; the course watermark generalization (m3-06 or here); the start-time `{judge: true}` cohort hand-over
  from the gateway; **hand-offs to m3-13** (set `JUDGE_BASE_URL` on practice and the gateway; kill-switch test covers
  `GRADING_OVERRIDE`). ADR only for a call beyond ADR-0029/0034 (check peers' ADR numbers first).

## Done when (acceptance)

- [ ] Replay/golden tests for timeout, accept and override conclusions (M3: timeout Miss, immediate accept, bounded `self_grade_pending` pick) are green.
- [ ] The reconciler heals a dropped event in compose, against m3-06's internal endpoints.
- [ ] The touch deadline drains to the watermark; expired judged course attempts conclude as Miss at the deadline.
- [ ] The strategy table equals v1 on the self path; the D18 golden table and the invariant properties are green.
- [ ] Give-up is synchronous: solution unlocked at once; lock = earliest pass below `close_seq`, else Miss.
- [ ] New event fields present; subject-registry and golden-ACL tests green; the ACL PR merged; `sqlc diff` clean; CI green.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commit(s) with the attribution lines, then push, then a PR in every repo touched (`../infra` PRs first where the order requires it; infra PRs are never folded into a tag). Here the xlearn PR goes first (step 10 above), then the `../infra` ACL PR rendered from `main` after that merge (step 11 above; plus the egress PR only if step 11 needs it).
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. (infra has no CI: the `make nats-acl-render` diff in the PR body is the check; verify the reload after the merge.)
3. **Release action — merge only (ships in `v1.14.0`) + the ACL PR:** nothing deploys from the xlearn PR; it ships in `v1.14.0`, cut by [m3-13](../sprints/sprint-m3-13.md). Don't tag. The NATS ACL infra PR (plan task 8) is merged in this sprint, on its own and before `v1.14.0`, then verified (reload, no `legacy`, no permission-violation logs, `host-verify --cluster --nats-stage=n3` or `n4`).
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (the ACL PR and its reload date go in the follow-up, since that PR merges after the xlearn one).
5. Run `git checkout main && git pull` in every repo touched (xlearn and `../infra`). If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
