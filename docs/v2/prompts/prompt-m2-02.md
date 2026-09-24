# Prompt — Sprint m2-02 · M2b consumers + projections v2 → v1.9.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m2-02.md`](../sprints/sprint-m2-02.md)   ·   **Milestone:** M2 (M2b consumers; tag v1.9.0)   ·   **Prereqs:** [m2-01](../sprints/sprint-m2-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions and the land-and-sync directive.
- The plan: [`../sprints/sprint-m2-02.md`](../sprints/sprint-m2-02.md) — tables, endpoints and the release checklist.
- The M2a work this tag carries: [`../sprints/sprint-m2-01.md`](../sprints/sprint-m2-01.md).
- Projections: [ADR-0018](../../adr/0018-progress-projection-grain-and-rebuild.md) (event grain, commutative upserts,
  drop-and-replay) and [`../../runbooks/projection-rebuild.md`](../../runbooks/projection-rebuild.md);
  [t1 §9](../research/t1-content-data-model.md#9-public-dashboard-data-deltas) (what the v2 projections hold and the
  backfill-then-replay order); [ADR-0027 §7](../../adr/0027-content-evalpack-and-user-data-model.md#7-public-profile-deltas);
  [t0 §5 (e)](../research/t0-extensibility-frame.md#5-the-learning-signal-contract-narrow-waist) (`xlearn.review.touch_scored`,
  review is the producer).
- Public-dashboard tasks P6, P7, P9: [rollout §10](../rollout-plan.md#10-public-dashboard-tasks).
- Tag order and timeline: [rollout §4 M2](../rollout-plan.md#4-per-milestone-detail), [rollout §7](../rollout-plan.md#7-indicative-tag-timeline);
  [ADR-0034 §1.6, §3, §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) (timeline, consumers
  before producers, release checklist); [ADR-0035 §1–§2](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#2-nats-auth-nkey-users-fine-acls-server-first)
  (topology, registry test, per-durable ACLs, the ACL-before-tag and NetworkPolicy standing rules).
- Code: `internal/assessment/{consumers.go,progress.go,mock.go,service.go,store/projections.go,store/queries/projections.sql,store/migrations/}`,
  `cmd/assessment/main.go`, `internal/review/store/store.go` (subject constants), `internal/platform/events/{topology.go,testdata/}`,
  `internal/gateway/progress.go` (`overrideSolvedTotal`), `web/src/lib/progress.ts`, `.github/workflows/deploy.yml`, `.release-line`;
  infra: `../infra/infrastructure/messaging/release.yaml`.

## Context

m2-01 merged the M2a engine and review's `touch_concluded` consumer with **no producer**. This sprint adds the M2b
consumer side — assessment's `touch_scored` consumer and the **projections v2** (`proj_activity` dated by `anchor_at`,
`proj_touch_stats`, the provenance outcome mix, touch-aware mastery, `path_slug` everywhere) — **dual-written** next to
the v1 projections, then tags **v1.9.0** with every producer idle. v1.10.0 (m2-05) turns the producers on, runs review's
`touch_scored` backfill and the prod drop-and-replay, and switches readers; because both consumers are already bound,
nothing is lost. Producer = review (`xlearn.review.touch_scored`, T0 §5(e), T1 §9). The floor stays 1.7.0; this is an
expand-only tag, so no snapshot is needed.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m2-01](../sprints/sprint-m2-01.md) merged on `main` (touch engine + review `touch_concluded` consumer; `TestNoTouchProducerYet` green).
- [ ] m2-01's `../infra` PR (practice `REVIEW_BASE_URL`, NetworkPolicy check) merged — required before the tag.
- [ ] v1.8.0 is live (`/xlearn/api/v1/healthz`) and no peer tag or release PR is in flight (`git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents).
- [ ] NATS N1 state known ([mi-06](../sprints/sprint-mi-06.md)): `grep -n authorization ../infra/infrastructure/messaging/release.yaml`. If the golden block is there, step 9 opens the ACL PR; if N1 hasn't landed, the xlearn golden update is enough and step 9 is recorded "n/a — ACL ships with N1".

## Do this (in order)

1. **[X] Branch** `feat/m2-02-projections-v2` from an up-to-date `main`.
2. **[X] `touch_scored` schema** (plan Task 1): the subject constant in review (no emit), the payload shape in
   `docs/architecture/events.md`, the fixture `internal/platform/events/testdata/touch_scored.v2.json`, the `topology.go`
   registry entries (review produces; assessment handles; identity ignores; review/notifications is out of filter). Keep the
   registry test green.
3. **[X] assessment migration** (Task 2): `proj_activity`, `proj_touch_stats`, the **new** `proj_outcome_mix_v2` (the v1
   `proj_outcome_mix` has no provenance and stays untouched; every later reader uses `_v2`), `path_slug` on
   `proj_coverage`/`proj_mastery`, `proj_mastery.max_passed_level` (next free goose version; expand only). `sqlc generate`.
4. **[X] Consumer + dual-write** (Tasks 1–2): decode the v2 fields with v1 defaults forever; `ApplyProjection` writes v1
   and v2 in the same tx as the inbox claim; `touch_scored` feeds `touches`, `proj_touch_stats` and `max_passed_level`;
   `ScoreMock` increments `proj_activity.mocks` inside its existing score-once tx, dated by the UTC day of
   `mock_session.started_at` — **no** `mock_completed` consumer; `RebuildActivityMocks` (set-based, from scored
   `mock_session` rows, same dating) for the replay. The `CourseCounts(account, paths)` store read (concluded =
   `proj_coverage.solved`, passed = `proj_mastery.best_rank ≥ 3`, per `path_slug`) that m2-03 builds on. Every write
   `+`/`GREATEST`/`LEAST`.
5. **[X] v2 read endpoints** (Task 3): `GET /progress/activity`, `/progress/touch-stats`, `/progress/outcome-mix`
   (`?path=`, JWT learner), with the mastery mapping and `judge_checked_pct` (0 until M3). Remove `problemTotal`
   (summary `total: null`); confirm the gateway's `overrideSolvedTotal` still fills it from curriculum; add the SPA's
   null guard ("—"). Do **not** switch any existing reader.
6. **[X] Replay** (Task 4): the golden test over `testdata/replay/m2b.jsonl` (v1 + v2 envelopes; `resolution` =
   immediate / timeout / accepted / override; touch_scored variants) **plus seeded scored `mock_session` rows**
   (`testdata/replay/m2b.mocks.sql` — mocks are not events; never put mock events in the jsonl) run through the same
   `RebuildActivityMocks` step as the command, with the expected-model golden (incl. `CourseCounts`), the drop-and-replay
   equality and the N-permutation property test; the `assessment admin replay-projections --confirm` command on the
   dedicated `assessment-replay` durable (declared in `topology.go`, `DeliverAll`, `InactiveThreshold` 1 h; truncate
   projections + inbox in one tx; `RebuildActivityMocks`); rehearse it in compose on seeded data; write the runbook's v2 section.
7. **[X] ADR "Projections v2"** (Task 5) — check peers' PRs/worktrees, take the next free number; add "Amended by" to ADR-0018.
8. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real-PG via `XLEARN_TEST_DATABASE_URL`),
   `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`, the migration lint, web `typecheck`/`lint`/`test`/`build`
   (then `git checkout -- web/dist/.gitkeep`), the ACL golden (`make nats-acl-render` → update the golden file).
9. **[I] ACL PR** (Task 6) — **only if N1 has landed** (entry gate): paste the re-rendered `authorization` block (the
   `assessment-replay` durable, plus any m2-01 diff) into `../infra/infrastructure/messaging/release.yaml`; reload only.
   Merge it **before** the tag; confirm Flux applied it (read-only). If N1 hasn't landed: record "n/a — the golden ships
   with N1" and note in status.md that the golden changed for mi-06.
10. **[X] Merge** the xlearn PR (CI green, squash) per land-and-sync; `git checkout main && git pull`.
11. **[X] Tag** (Task 7): walk the release checklist below; take the **next free minor** (indicative `v1.9.0`; its
    major must equal `.release-line`); `git tag v1.9.0 && git push origin v1.9.0`; GitHub release titled
    **`v1.9.0 — v2 build · M2a/M2b consumers`** noting "producers off", floor 1.7.0 (unchanged), no snapshot (expand-only).
12. **[H] Verify after the tag** (Task 8, by looking, D34): healthz version, images, ImagePolicies + HelmReleases, the
    smoke test, and **consumers bound and idle**, read-only:
    - `ssh vps 'k3s kubectl get --raw "/api/v1/namespaces/messaging/pods/nats-0:8222/proxy/jsz?consumers=true"'` (the
      API-server **pod** proxy host-verify uses for `/varz`/`/connz`): `review` on `XLEARN_PRACTICE` and `assessment` on
      `XLEARN_PRACTICE` + `XLEARN_REVIEW` present, each `num_pending` 0 and `num_ack_pending` 0;
    - no producer (`/jsz` has no per-subject counts):
      `ssh vps 'k3s kubectl exec -n databases projects-pgstore-1 -c postgres -- psql -d xlearndb -Atc "select subject, count(*) from practice.outbox group by 1 union all select subject, count(*) from review.outbox group by 1"'`
      lists neither `xlearn.practice.touch_concluded` nor `xlearn.review.touch_scored`.

Release checklist ([ADR-0034 §6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) + the ADR-0035 §2 standing rule):
- [ ] Before the tag: peers' tags and PRs are checked (parallel sessions; `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents)
- [ ] Before the tag: it is the next free version, and its major equals `.release-line`
- [ ] Before the tag: ACL PRs for new streams and consumers are merged
- [ ] Before the tag: a new service's image comes before its policy
- [ ] Before the tag: for a contract: rehearsed in compose, floor marked
- [ ] Before the tag: for a contract, erase or GA tag: `host-verify --cluster` is green (ADR-0035), the host has settled, and the snapshot is taken
- [ ] Before the tag: from M6: no live interviews
- [ ] After the tag (by looking, D34): `/xlearn/api/v1/healthz` reports the version
- [ ] After the tag: `k3s kubectl get deploy -n xlearn` shows the new images
- [ ] After the tag: every `xlearn-*` ImagePolicy's latest equals the tag, and the HelmReleases are Ready
- [ ] After the tag: smoke-test login, the dashboard and coach
- [ ] Record milestone → tag → floor → snapshot and any flag changes in `docs/v2/status.md`
- [ ] (ADR-0035 §2 standing rule, not part of ADR-0034 §6) Every new in-cluster HTTP or NATS caller this tag introduces has its NetworkPolicy (ingress and egress) change in its own infra PR, merged before the tag

## Constraints

- **Consumers before producers:** no code path emits `touch_concluded` or `touch_scored` in this tag; the registry test
  and m2-01's guard test must stay green. The producers, the backfill and D2 are m2-05's.
- **Projections are a pure function of the event log plus assessment-owned `mock_session`** (ADR-0018, amended by
  this sprint's ADR): no external reads in the projection path, no wall-clock branching, `path_slug` from the envelope
  (or the mock session) only, every write commutative, inbox claim in the same tx. Readers derive streaks, rates and
  concluded/passed at read time.
- **Do not switch readers** in v1.9.0 (no gateway/SPA change beyond the `total: null` guard). The v2 tables fill forward
  now and are rebuilt by m2-05's replay.
- **Service boundaries (ADR-0005):** assessment writes only schema `assessment`; `mock_session` → `proj_activity.mocks`
  is assessment-owned data, not a cross-schema read.
- **goose + sqlc:** next free version per service; expand only (no drops — `proj_heatmap` and the v1
  `proj_outcome_mix` are dropped by a later contract, tracked as pending in status.md); `sqlc diff` clean.
- **GitOps + ACLs:** the ACL PR (per-durable fine ACLs) merged before the tag; infra PRs are their own tasks, never
  folded into the tag; never `kubectl apply`. The replay command is run via `kubectl exec` only in m2-05 (sanctioned
  path, logged in status.md) — not in this sprint.
- **D34:** no alerting; post-tag verification is by looking. **Memory-sum rule:** no new pod or container.
- **Tag hygiene:** next free minor; major = `.release-line`; never move or re-push a tag; check peers first.

## Deliverables

- The `touch_scored` subject constant, fixture, registry entries and the events.md flow.
- assessment: the projections v2 migration; dual-write in `ApplyProjection`; `ScoreMock` → `proj_activity.mocks` and
  `RebuildActivityMocks`; the `CourseCounts` store read; the three v2 read endpoints; `problemTotal` removed (+ the SPA
  null guard).
- `replay_golden_test.go` + `testdata/replay/{m2b.jsonl,m2b.mocks.sql,m2b.golden.json}`; `assessment admin
  replay-projections`; the `assessment-replay` durable in `topology.go`; the runbook's v2 section.
- The ADR "Projections v2" (+ "Amended by" in ADR-0018).
- `../infra` ACL PR (merged before the tag).
- Tag **v1.9.0** (release notes: M2a/M2b consumers, producers off, floor 1.7.0).

## Update status

- [`../sprints/sprint-m2-02.md`](../sprints/sprint-m2-02.md): tasks 🔄 → ✅; _Overall_ ✅ once v1.9.0 is verified.
- [`../status.md`](../status.md): the Sprint board row; **M2** milestone → "v1.9.0 consumers live, producers off"; the
  **milestone → tag → floor → snapshot** row: `M2a/M2b consumers → v1.9.0 → floor 1.7.0 (unchanged) → snapshot n/a
  (expand-only)`; gate state "producers off"; flag inventory unchanged; **pending contracts**: `proj_heatmap`, v1
  `proj_outcome_mix`; the ACL PR number under NATS; a **Decisions log** line for the touch-aware mastery mapping, the
  dual-write/switch-at-replay call, the replay durable, `proj_outcome_mix_v2` as the one provenance table, the
  log-plus-`mock_session` purity rule, and `touch_scored` producer = review (`xlearn.review.touch_scored`).
- Record the ADR number.

## Done when (acceptance)

- [ ] Consumers bound and **idle** in prod: review handles `touch_concluded`, assessment handles `touch_scored`; no producer emits either.
- [ ] **Replay golden equal**: drop-and-replay and N permutations of the fixture log produce the golden read model.
- [ ] Projections v2 dual-written; v1 readers unchanged; `problemTotal` gone.
- [ ] Replay command rehearsed in compose; runbook v2 written; ADR recorded.
- [ ] ACL PR merged before the tag (or n/a — N1 not landed); **v1.9.0 verified** per the checklist and recorded in status.md.

**Shipping:** per AGENT.md land-and-sync with **this sprint's release action — tag v1.9.0**: branch → conventional
commits with the attribution lines → push → PR → CI green → squash-merge (and the `../infra` ACL PR, merged first) →
walk the release checklist → push the tag → let Flux build and deploy → verify live (by looking) → record in
status.md (a small docs PR, merged the same way) → `git checkout main && git pull` in every repo touched.
