# Prompt — Sprint m2-01 · M2a touch-attempt engine + `touch_concluded` consumer

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m2-01.md`](../sprints/sprint-m2-01.md)   ·   **Milestone:** M2 (M2a)   ·   **Prereqs:** [m1-08](../sprints/sprint-m1-08.md) (v1.8.0 live) · [ds-m2-01](../sprints/sprint-ds-m2-01.md) (boards frozen)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, service boundaries, land-and-sync.
- The plan: [`../sprints/sprint-m2-01.md`](../sprints/sprint-m2-01.md) — task detail, tables and endpoint contracts live there.
- Touch lifecycle: [t4 §3.1–3.3](../research/t4-judge-contract.md#3-attempt-and-conclusion-lifecycle),
  [§3.6](../research/t4-judge-contract.md#36-parked-q2-resolved-abandoned-attempts) and [§3.7](../research/t4-judge-contract.md#37-touches-mocks-arena),
  DSA criteria [§6.6](../research/t4-judge-contract.md#66-touch-formats-and-pass-criteria-per-course), D2 queries
  [§6.5](../research/t4-judge-contract.md#65-unsolved-attempts-enter-revision-d2-with-the-revisable-guard); **t4 §13's owner
  decisions (D14–D18) override the body** — D15 replaces "resumable forever".
- Signal contract: [t0 §5](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist) (`touch_concluded`
  fields), [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates) (M2a/M2b/M2c).
- Schema deltas: [t1 §4](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual) (practice, review, curriculum
  M2a rows), item format + probes [t1 §7.1](../research/t1-content-data-model.md#71-package-format), content delivery
  [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery).
- ADRs: [0026](../../adr/0026-per-course-extensibility-model.md) (practice is the single signal writer),
  [0029 §3–§4](../../adr/0029-judge-contract-and-learning-signal.md#3-conclusion-practice-the-single-writer),
  [0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand
  rules, consumers before producers), [0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#1-event-pipeline-no-new-component-five-code-changes)
  (topology, registry test, ACL standing rule), [0005](../../adr/0005-data-ownership-and-migrations.md),
  [0016](../../adr/0016-mistake-journal-and-worker-service-auth.md) (internal endpoints, no HTTP in a tx),
  [0015](../../adr/0015-five-touch-scheduler-model.md), [0018](../../adr/0018-progress-projection-grain-and-rebuild.md).
- Rollout: [§4 M2](../rollout-plan.md#4-per-milestone-detail) (tag order), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (operating rules).
- Mechanisms this sprint reuses, not re-invents: m1-02's migration lint and its `-- xlearn:relax <reason>` marker
  ([m1-02 task 4](../sprints/sprint-m1-02.md#4--contract-header-lint-x)); mi-05's dead-letter sink
  ([mi-05 task 3](../sprints/sprint-mi-05.md#3--dead-letter-hook-x)); the honor-key claim that later replaces the M2
  policy ([m4-04](../sprints/sprint-m4-04.md), [t4 §3.5](../research/t4-judge-contract.md#35-parked-q1-resolved-who-finalizes-a-non-authoritative-grade)).
- The frozen board `design-system/screens/v2/AB04-touch.html` (frames F2–F14 define what the endpoints must support).
- Code: `internal/practice/{handlers.go,service.go,store/store.go,store/queries/*.sql,store/migrations/}`,
  `cmd/practice/main.go`, `internal/review/{consumers.go,handlers.go,service.go,clients.go,store/store.go,store/queries/*.sql}`,
  `internal/curriculum/{handlers.go,seed.go,store/}`, the curriculum content package and DSA item files (m1-09 layout),
  `internal/course` (m1-01), `internal/platform/events/{topology.go,consumer.go,events.go}`, `internal/e2e/coreloop_test.go`,
  `docker-compose.yml`; infra: `../infra/apps/xlearn-practice.yaml`, `../infra/apps/xlearn-review.yaml`.

## Context

M1 made DSA "a course like any other" with no behaviour change (v1.8.0, floor 1.7.0 hard). M2 turns revision touches
into real attempts. This sprint is **M2a**: practice gains `purpose=touch` attempts with a server clock, answer-free
probes and lock-ins graded against public (honor-grade) keys, a D15 deadline ticker and one guarded conclusion; review
gains the `touch_concluded` consumer, scoring through the same ladder code as v1's self endpoint. It ships in **v1.9.0**
(tagged by m2-02) with **producers idle**: the `touch_concluded` outbox insert is m2-05's (v1.10.0), because review acks
unknown subjects silently and a producer shipping with its consumer loses events. No gateway route reaches the touch
endpoints until m2-04. There is no judge yet, so the DSA re-solve criterion is self-attested (`graded_by=self`).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] M1 shipped: **v1.8.0 live** — `curl -s https://projects.sujaykumar.dev/xlearn/api/v1/healthz` reports ≥ 1.8.0 and `docs/v2/status.md` records floor 1.7.0.
- [ ] AB04–AB06 and AB22 **frozen**: the ds-m2-01 PR is merged (`gh pr list --state merged --search "AB04"`) and `design-system/screens/v2/AB04-touch.html`, `AB05-catalog-agenda.html`, `AB06-public-profile-v2.html` and `AB22-visibility-toggles.html` are on `main` (`git ls-tree origin/main design-system/screens/v2/`).
- [ ] On `main`: `internal/course` with `revision.bands[*]` (criteria, `mock_mode`, band timer); item schema with `revision.probes[]` + `solution_facts`; the m1-09 DSA item layout (`curriculum/courses/dsa/items/<id>/…`); `internal/platform/events/topology.go` + the subject-registry test; v2 envelope decoders; practice `GET /attempts/open`; `hack/lint-migrations.sh` with m1-02's `-- xlearn:relax <reason>` marker; review's `event_dead_letter` sink (mi-05).

Then read (information, not a gate) the ds-m2-01 PR's **"Decisions to confirm" #1** — honor-key strictness until M4's
claim: (A) strict or (B) v1-parity (plan Task 3). The ds-m2-01 merge froze (A), its stated default (D40). If status.md
records that the owner has since picked (B), build that; otherwise build (A) and keep it as an open owner item in
`docs/v2/status.md`. Don't ask and don't wait.
- [ ] Parallel sessions: `gh pr list`, `git worktree list` and ListAgents show no peer editing practice/review migrations right now (goose versions are sequential per service).

## Do this (in order)

1. **[X] Branch** `feat/m2-01-touch-engine` from an up-to-date `main`.
2. **[X] practice M2a expand** (plan Task 1): the migration (next free goose version; the `CONCURRENTLY` indexes in their
   own `-- +goose NO TRANSACTION` file), the idempotent backfill (also run at startup as `HealEndedCourseAttempts`, so an
   R-b rollback to v1.8.0 heals on roll-forward; "open course attempt" stays `ended_at IS NULL`), the widened
   `timer.kind` CHECK (the `DROP CONSTRAINT` line carries m1-02's `-- xlearn:relax widen practice.timer.kind for touch`;
   **don't edit the lint**; a store test inserts every pre-existing value), the `purpose = 'course'` filter on every v1
   attempt query, and `LogOutcome` as a guarded conclude. Pre-check prod for duplicate outcomes read-only (plan Task 1)
   before relying on the unique index; stop and report if any exist. `sqlc generate`.
3. **[X] `problem.spec` + content** (Task 2), in m1-09's layout: curriculum migration + seed + answer-free API field; add
   the two templated DSA probes to every `curriculum/courses/dsa/items/<id>/item.json` (one-shot script, commit the
   output); fill `solution_facts.complexity` there for every seeded item — 5 of the 14 v1 items state it in
   `curriculum/courses/dsa/items/{1,3,16,40,104}/sections/solution/*-complexity.md`, 9 are derived from the reference
   code under `_code/`; add any item authored since; include the learner-facing variants for multi-/named-variable facts
   (ids 7, 40, 104; plan Task 2) and list every fact in the PR; write `curriculum/courses/dsa/keys/patterns.json`; extend
   the content CI (both probes present, every pattern aliased, every fact parses under the normaliser).
4. **[X] `internal/course/keys`** (Task 3): text alias matcher, big-O normaliser (any identifier is an atom; commutative
   products/sums; no variable renaming), `key_source` resolution, honor flag, and `honorProbeMet()` with the policy
   ((A) as frozen, or (B) if status.md records the owner's pick) as a code constant stamped into `policy_version`; `key_match` kept in `criteria[]`; table tests over
   every seeded item plus the variant corpus (ids 7, 40, 104 included).
5. **[X] review side first** (Task 6) — so the practice client has something to call: the `touch_result` migration,
   the reanchor queries (`ReanchorTouch` with an anchor, new `ReanchorPendingTouches`), `applyTouchOutcomeTx` extracted
   from `Score` (v1 self path unchanged — its existing tests must stay green), `HandleTouchConcluded` + the consumer
   switch case (consumer rows set `auto_pass = touch_passed` and `mock_mode`, v1 DSA columns NULL; a path-less or
   malformed v2 envelope → `event_dead_letter` row with `err_class=decode` through the mi-05 sink, `Term()`, ERROR log —
   tested), and `GET /internal/revisions/{id}`.
6. **[X] practice touch endpoints** (Task 4): the review client (`REVIEW_BASE_URL`, 5 s timeout, nil-safe → 503),
   the embedded item index, start / view (`probes_shown_at`) / lock-in / attest / end, `concludeTouch()` with the typed
   `touch_concluded` payload built but **not** written to the outbox, `TestNoTouchProducerYet`, and `GET /attempts/open`
   reporting `purpose: "touch"`.
7. **[X] Ticker** (Task 5): 30 s, `FOR UPDATE OF attempt SKIP LOCKED`, void / abandoned / deadline; started in
   `cmd/practice/main.go`, stopped before the pool closes; the two-ticker `-race` test.
8. **[X] Registry, fixture, e2e, docs** (Task 7): register `xlearn.practice.touch_concluded` (review handles; assessment
   and identity ignore); `internal/platform/events/testdata/touch_concluded.v2.json` shared by the practice golden test,
   review's consumer tests and the e2e; the e2e idempotency case; update `docs/architecture/{events,data-model,services,api}.md`;
   `docker-compose.yml` sets `REVIEW_BASE_URL` for practice.
9. **[X] ADR** "Touch attempts in practice (M2a realization)" — check peers' PRs/worktrees for ADR numbers first, then
   take the next free number (MADR, append-only).
10. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG integration tests via
    `XLEARN_TEST_DATABASE_URL` for practice, review, curriculum), `go test -tags e2e -race ./internal/e2e/...`,
    `sqlc diff`, the migration lint, the content job, web tests (unchanged). Run the compose stack and exercise the touch
    endpoints directly (a token from the compose dev login, ADR-0022) through start → view → lock-ins → attest → conclusion, plus a
    deadline expiry and a void.
11. **[I] Infra PR — practice → review caller** (Task 8) in `../infra`: `REVIEW_BASE_URL` on `apps/xlearn-practice.yaml`;
    confirm from `apps/xlearn-review.yaml` that MI-5a already admits the `xlearn` namespace to review's internal routes
    (state it in the PR); add practice → review :8084 egress only if the mi-11 egress matrix has merged. Its own PR,
    merged in this session (harmless on v1.8.0), so it's in before v1.9.0.
12. **[I] ACL render check** (Task 9): `make nats-acl-render`, diff against the golden. Expected no change; if it changes,
    open the infra ACL PR (re-rendered block) and merge it in this session, before v1.9.0.
13. **[X] Ship** (see Ship, below).

## Constraints

- **Service boundaries (ADR-0005):** practice writes only schema `practice`, review only `review`, curriculum only
  `curriculum`. practice reads review through `GET /internal/revisions/{id}` (a soft reference), never its tables.
  Downstream stays course-blind: review never interprets parts or criteria keys (it stores `criteria` opaquely).
- **goose + sqlc:** migrations embedded, run on startup under the advisory lock; the next free version per service;
  commit generated code; `sqlc diff` clean. **Expand only** (ADR-0034 §3): no drops, no `SET NOT NULL`, no contract
  marker (this is not a contract; the floor stays 1.7.0). Never run `Down` in production.
- **Outbox / inbox:** review's consumer claims the inbox and writes `touch_result` in one tx; no HTTP inside a tx
  (resolve the pattern and call review before `pool.Begin`); `NakWithDelay` backoff; stop the consumer and the ticker
  before `srv.Shutdown()`. **practice writes no `touch_concluded` outbox row** in this sprint (consumers before
  producers; the emit is m2-05's).
- **No gateway or web change.** No BFF route to the touch endpoints (m2-04 owns them).
- **GitOps:** infra changes only through `../infra` PRs; never `kubectl apply`. Infra PRs are their own tasks, never
  folded into a tag. **ACL PR before the consuming tag** if durables change; the NetworkPolicy change for a new in-cluster
  caller in its own PR before the tag (ADR-0035 §2 standing rule).
- **D34:** no alerting of any kind (no Flux Alert, no push, no opscheck); log at ERROR where the design says "alert".
- **Memory-sum rule (ADR-0035 §5):** no new pod; practice keeps its existing memory limit — check the working set
  read-only after v1.9.0.
- **Parallel sessions:** check peers' PRs, tags and worktrees before claiming the ADR number and before merging
  migrations; rebase and take the next free goose version if a peer merged first.

## Deliverables

- practice: the M2a migration(s) + backfill; guarded `LogOutcome`; touch endpoints; the item index; `concludeTouch()`
  + typed payload; the ticker; the review client; tests (unit, real-PG integration, `-race`).
- curriculum: `problem.spec` migration, seed, answer-free API field; DSA probes, `solution_facts`, `keys/patterns.json`;
  content CI checks.
- `internal/course/keys` with table tests.
- review: `touch_result` v2 migration; reanchor queries; `applyTouchOutcomeTx`; `HandleTouchConcluded` + consumer case;
  `GET /internal/revisions/{id}`; tests.
- `topology.go` registry entry; `internal/platform/events/testdata/touch_concluded.v2.json`; the e2e case.
- Docs (`events.md`, `data-model.md`, `services.md`, `api.md`) and the new ADR.
- `../infra` PR: practice `REVIEW_BASE_URL` (+ egress only if mi-11 merged); an ACL PR only if the render changed.

## Update status

- [`../sprints/sprint-m2-01.md`](../sprints/sprint-m2-01.md): each task 🔄 → ✅ (⛔ with a reason); _Overall_ ✅ when all are.
- [`../status.md`](../status.md): the Sprint board row; the **M2** milestone row → 🔄 (M2a merged, ships in v1.9.0);
  **confirm the artboard rows AB04, AB05, AB06, AB22 read "frozen (PR #N, date)"** and
  [ds-m2-01](../sprints/sprint-ds-m2-01.md) reads ✅ (its own merge records them under D40; repair if missing; this is the
  first build sprint consuming them); the infra PR numbers; a **Decisions log** line each for: interim practice-side key
  evaluation, the M2 attestation, touch `graded_by`/`trust` derivation, the honor-probe policy ((A) as frozen, or (B) if
  the owner picked it — plus, if he hasn't, an **open owner item** "strict honor keys (A) until m4-04; a switch to (B) is
  a one-constant PR", which nothing waits on), the `timer.kind` widening under `xlearn:relax`, the rollback-safe
  course-attempt heal, the practice → review internal read, and the ADR number.
- Content status table: DSA probes + solution facts present for the seeded items (count).

## Done when (acceptance)

- [ ] Touch attempts start **server-timed**, resume idempotently, allow one live touch per (account, problem), and conclude per **D15** (immediate when every criterion is decided, End review, deadline + 5 s via the ticker; never-shown → voided; shown-and-silent → abandoned fail).
- [ ] Lock-ins are evaluated server-side against public keys, stamped once, never reveal correctness before conclusion; `pattern_named_fast` enforces < 120 s.
- [ ] review processes a `touch_concluded` fixture **idempotently** (same `event_id`; same `attempt_id` under a new `event_id`), advancing on pass and resetting to Day 1 from `anchor_at` on fail; the v1 self endpoint behaves exactly as before.
- [ ] **No producer emits the new subject yet** (`TestNoTouchProducerYet` green; no BFF route).
- [ ] v1 course flows unchanged (every v1 e2e green); `LogOutcome` double-submit → one outcome, one event.
- [ ] Subject registry and ACL golden green; the practice → review infra PR merged in this session.

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch, then conventional commits with the attribution lines, then push, then a PR in every repo touched: the xlearn PR and the `../infra` practice → review caller PR (step 11), plus an ACL PR only if the render changed (step 12). Infra PRs are their own PRs, never folded into a tag.
2. Once CI is green (fix, then merge, on failure), squash-merge each. Never enable auto-merge. `../infra` has no CI: the env diff and the NetworkPolicy/ACL evidence in each PR body are its checks.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships in **`v1.9.0`**, which [m2-02](../sprints/sprint-m2-02.md) cuts with every M2a/M2b consumer and the producers idle; the infra PRs are already merged by then. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way.
5. Run `git checkout main && git pull` in xlearn and `../infra`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
