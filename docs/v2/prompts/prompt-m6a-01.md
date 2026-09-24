# Prompt — Sprint m6a-01 · Interview core: schema, state machine, failsafes, caps (L19)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m6a-01.md`](../sprints/sprint-m6a-01.md)   ·   **Milestone:** M6a (text interviewer; ships dark in a v2.0.x patch)   ·   **Prereqs:** [ds-m6a-02](../sprints/sprint-ds-m6a-02.md), [ga-02](../sprints/sprint-ga-02.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions and the land-and-sync directive.
- The plan: [`../sprints/sprint-m6a-01.md`](../sprints/sprint-m6a-01.md) — the table list, the FSM table and named sets, caps, the exposure design.
- The sibling plans whose seams you create: [m6a-02](../sprints/sprint-m6a-02.md) (brain: `Prober`, `OnPhase`, `AddSpend`, the hint
  ledger, the pause brief) and [m6a-04](../sprints/sprint-m6a-04.md) (snapshots via `CodeSource`, the event log it streams, `InMock` from
  your current-interview read, the item picker that replaces your placeholder, run echoes into `run_count`/`last_verdict_class`).
- [ADR-0032](../../adr/0032-realtime-ai-mock-interviewer.md) (Accepted with the S6 result by [ds-m6a-01](../sprints/sprint-ds-m6a-01.md)) §4 failsafes, §5, §6 limits.
- [t6 §4](../research/t6-realtime-interviewer.md#4-session-state-machine--the-owners-failsafes) (FSM diagram, failsafes 1–3,
  failure-mode table, coach lock list), [t6 §6](../research/t6-realtime-interviewer.md#6-assessment) (evidence enum, never-list, DSA
  dimensions), [t6 §7](../research/t6-realtime-interviewer.md#7-privacy-consent-retention--accessibility) (data inventory, consent,
  retention), [t6 §9 P0](../research/t6-realtime-interviewer.md#9-phased-plan), [t6 §11](../research/t6-realtime-interviewer.md#11-what-t6-constrains-downstream),
  [t6 §13](../research/t6-realtime-interviewer.md#13-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict)
  (D29, D30 override the body), and t6 §16 (the S6 results).
- Limits and releases: [ADR-0035 §4](../../adr/0035-v2-operations-nats-auth-limits-capacity.md#4-limits-inventory) (L6, L19),
  [ADR-0034 §2](../../adr/0034-v2-release-labelling-gating-and-rollback.md#2-feature-gating-three-tiers-no-flag-service) (T-1/T-3),
  [§6](../../adr/0034-v2-release-labelling-gating-and-rollback.md#6-release-checklist) ("from M6: no live interviews");
  [ADR-0033 §7–§8, §12](../../adr/0033-invite-only-admission-and-owner-admin.md#12-authz-deltas-across-v2) (roles, admin CLI pattern, row 14);
  [ADR-0026 §1](../../adr/0026-per-course-extensibility-model.md#1-a-course-is-data-a-capability-is-code) (manifest snapshot rule);
  [ADR-0005](../../adr/0005-data-ownership-and-migrations.md).
- The frozen boards (behaviour and copy only; never shipped): `design-system/screens/v2/AB13-mock-v2.html` (the source for the DSA
  rubric's public band descriptors, task 3), `AB24-mock-setup-preflight.html`, `AB25-live-hud-text.html`,
  `AB26-grace-paused-resume.html`, `AB27-debrief-proposal.html`, `AB28-accessibility-settings.html`.
- Sprints this builds on (all live in v2.0.0): [m1-06](../sprints/sprint-m1-06.md) (`withhold()` + resolver),
  [m1-07](../sprints/sprint-m1-07.md) (`coach_paused`), [m1-10](../sprints/sprint-m1-10.md) (`key_default`, catalog),
  [m1-04](../sprints/sprint-m1-04.md) (admin CLI pattern, roles), [l-01](../sprints/sprint-l-01.md) (coach erase handler + coverage
  test + tombstones), [m3-09](../sprints/sprint-m3-09.md)/[m3-14](../sprints/sprint-m3-14.md)/[m3-08](../sprints/sprint-m3-08.md) (arena routes and records).
- Code: `internal/coach/{service.go,handlers.go,config.go}`, `internal/coach/store/{store.go,migrations/,queries/,gen/}`, `cmd/coach/main.go`,
  `internal/course/{manifest.go,validate.go,golden_test.go}`, `curriculum/_schema/`, `curriculum/courses/dsa/course.json`,
  `internal/assessment/mock.go` (the v1 rail prompts), `internal/gateway/{coach.go,bff.go,security.go}` + the arena handlers,
  `docs/architecture/openapi.yaml`, `internal/gateway/openapi_drift_test.go`, `sqlc.yaml`.
  Infra (read-only): `../infra/apps/xlearn-coach.yaml`, `../infra/apps/xlearn-*.yaml` NetworkPolicy values.

## Context

v2.0.0 is live for the owner; the interviewer is v2.1 (D28) and ships **dark** in v2.0.x patches until
[m6b-04](../sprints/sprint-m6b-04.md) flips it on. S6 chose the voice shell and ADR-0032 is Accepted. This sprint builds the
**durable core** in coach — tables, an append-only event log, a server-clocked FSM with the owner's failsafes (5-minute grace, pause
≤ 24 h from the first pause and ≤ 3 pauses, a free resume state, `incomplete` after 24 h), the L19 caps that live in coach,
retention/erase and the `coach admin interviews --live` check — with **no model call**. [m6a-02](../sprints/sprint-m6a-02.md) adds the
brain; [m6a-03](../sprints/sprint-m6a-03.md) links the assessment `mock_session` and scoring; [m6a-04](../sprints/sprint-m6a-04.md) owns
snapshots, runs, the real item picker, `InMock` and the bounded SSE over your event log. Keep the voice states in the enum (unreachable
in text) so M6b needs no migration, and export small interfaces for every seam instead of building the siblings' parts.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [spk-04](../sprints/sprint-spk-04.md) (S6) reported: t6 §16 and `docs/v2/research/t6-s6-fixtures/` are on `main`.
- [ ] ADR-0032 status is **Accepted** (`docs/adr/0032-realtime-ai-mock-interviewer.md`).
- [ ] AB13, AB24–AB28 frozen: the ds-m6a-01 and ds-m6a-02 PRs are **merged** and the files exist under `design-system/screens/v2/`.
- [ ] judge's `mock` context exists on `main` ([m3-06](../sprints/sprint-m3-06.md)).
- [ ] `v2.0.0` (or a later v2.0.x) is live: `/xlearn/api/v1/healthz` version; `.release-line` = `2`.
- [ ] Parallel sessions: `gh pr list`, `git worktree list`, ListAgents — no open peer PR adds a coach goose migration or edits
      `internal/course` / `curriculum/courses/dsa/course.json` (else take the next free version at rebase / serialize the golden).

## Do this (in order)

1. **[X] Branch** `feat/m6a-01-interview-core` from an up-to-date `main`.
2. **[X] Schema** (plan task 1): one expand migration `internal/coach/store/migrations/0000N_interview.sql` (next free version) with
   `interview`, `interview_event` (append-only, monotonic `seq`), `interview_segment` (partial unique: ≤ 1 open), `interview_turn`,
   `interview_checkpoint`, `interview_pause`, `interview_consent`, `interview_exposure`, `interview_hint`, `admin_audit`; the
   one-non-terminal partial unique index; the sweeper indexes. Every table except `admin_audit` has `account_id` (it holds no
  account data; exempt in the erase list with a reason). `cap_micros` is nullable with no default (CHECKs `cap_micros IS NULL OR
  cap_micros > 0` and `state IN ('setup','abandoned') OR cap_micros IS NOT NULL`). sqlc queries in
   `internal/coach/store/queries/interview*.sql`; transitions as CAS `UPDATE … WHERE state = $expected` (`:execrows`). `sqlc generate`;
   commit `gen/`. Size caps in Go (turn ≤ 8 KiB, Σ ≤ 256 KiB, event data ≤ 4 KiB).
3. **[X] FSM + clock + events** (task 2): `internal/coach/interview/{fsm.go,clock.go,events.go}` — the 15 states, the transition table
   as data, guards, the exported **running / locked / editable / live / terminal** sets, an injectable `Clock`, rail = manifest rail ×
   multiplier (× 0.33 for an owner `qa_slice`), the 60-minute follow-up phase, the pause-limit end rule, `resume_by = first_paused_at +
   24 h` set once; `events.Append` and one `state` event per transition in the same tx. **Every non-terminal state has a timed exit**
   — use the plan's explicit **timed alphabet** (`setup_expired`, `preflight_timeout` → `setup`, `connect_timeout` →
   `interrupted(network)` keeping the grace when entered by `cleared`, `heartbeat_lost` in `live` and `resuming` (→ `paused`),
   `idle`, `rail_done`, `grace_expired`, `resume_expired` in `paused` **and** `resuming`, `debrief_done`, `stale_30d`) — plus
   `interrupted` → `end_without_feedback` → `finished(report_pending=quota)` (≥ 75 % dry; the ≥ 90 % quota default). `preflight_ok`
   sets `started_at`, which is what the ≤ 2 starts/day cap counts. Write `fsm_table_test.go`, the BFS **model check** (liveness =
   the timed-only graph is acyclic and ends only in terminals) and the seeded `math/rand/v2` **property test** (after the random
   prefix, 31 d of timed events alone ends terminal; in-memory, then a real-PG variant gated on `XLEARN_TEST_DATABASE_URL`) with
   exactly the plan's assertions.
4. **[X] Checkpoints, lease, sweeper** (task 2): deterministic checkpoints (phase, interrupt, pause, sigterm, finish) with the code
   fields read through a nil-safe `CodeSource`; `attach`/`heartbeat`/`interrupt` (beacon, `text/plain`); `sweeper.go` (5 s tick,
   `pg_try_advisory_lock`, `SKIP LOCKED`, CAS; one job per timed event: grace expiry with a nil-safe `Prober` (clears only with a
   fresh lease), heartbeat 30 s (`live`, `resuming`, `connecting`) measured from `max(last_heartbeat_at, state_since)`, idle 3 min ×
   multiplier, `preflight` > 2 min, `connecting` > 2 min, `resume_by` (`paused` and `resuming`), 30-day finished/proposed, setup 1 h,
   phase-boundary checkpoints with an `OnPhase` hook, rail end, debrief end, daily purge; skip `coach.erased_account`); a SIGTERM
   checkpoint in `cmd/coach/main.go` before the drain. Test two sweepers on one DB.
5. **[X] Handlers** (task 2): the coach routes in the plan with the typed errors; `start` returns 503 `interviewer_unavailable` while no
   `Brain` is registered (m6a-02 registers it); `GET /interviews/active` returns `{id, state, items, coach_locked, editable, resume_by}`;
   `GET …/resume-state` is free and deterministic; internal `POST /internal/interviews/{id}/exposure`. `AddSpend` applies the 85 %/100 %
   cap thresholds. Handler tests per route and error.
6. **[X] Manifest** (task 3): `mock.interviewer{…}`, `mock.rubric.dims[].evidence/ai/descriptors` in `internal/course` types, validation
   and `curriculum/_schema/`; DSA values from the plan; the exported **never-list** + lint (prove it fails on a planted term);
   regenerate the DSA golden (`-update`) and review the diff.
7. **[X] Caps, lock, gateway** (task 4): `caps.go` (409/429/422 + exported `SnapshotMinInterval`, `SnapshotMaxBytes`, `MockRunPace`);
   coach `POST /chat` → 409 `coach_paused {reason:"mock"}` in the locked set only, before any key decrypt; `internal/gateway/interview.go`
   (aud=coach, cohort gate `interviewAudience = "cohort"` → 404 for learners, the placeholder item pick, never echoed before `live`), the
   arena exposure push and the resume/finish backstop over existing gateway edges (the backstop sees only a **first-ever** reveal —
   `arena_revealed_at` is `COALESCE`d — so re-reveals rely on the push; say so in the PR hand-off to m6a-04); `apiRoute` withhold
   policies; `openapi.yaml` entries. Extend m1-04's JSON-check exemption in `internal/gateway/security.go` with
   `POST /api/interviews/{id}/interrupt` for `text/plain` only (the `sendBeacon` body), requiring `Sec-Fetch-Site` present and
   `same-origin` on that path. Tests: a learner gets 404; cohort flows; exposure push + backstop; the lock set; the beacon accepted
   same-origin and refused cross-site / without the header; text/plain elsewhere → 415; the daily cap counts at `preflight_ok`.
8. **[X] Retention + erase** (task 5): consent kinds/versions (`start` → 409 `consent_required`), `purge_at`, the purge job, per-turn and
   per-session delete (caveat `transcript_edited`), every table in coach's erase list (coverage test green), the canary log test
   extended to the new routes.
9. **[X] Admin CLI** (task 6): `coach admin interviews --live|--all`, `show <id>`, `abandon <id> --reason`; `admin_audit` per verb;
   `docs/runbooks/interviewer.md`.
10. **[I] Policies** (task 7): read `../infra/apps/xlearn-*.yaml` and confirm coach calls nobody new; record "no NetworkPolicy/ACL change"
    with the evidence in the PR. Only if coach must call practice/judge `/internal/*`: open the `../infra` PR (coach egress + callee
    ingress), GitOps only, and list it as a gate for [m6a-06](../sprints/sprint-m6a-06.md)'s tag.
11. **[X] Docs** (task 8): `docs/architecture/{services,api,data-model}.md`; ADR-0032 dated "Update — <date> (m6a-01)" section
    (pause-limit rule, exposure for the whole interview, the named sets, the timed alphabet and the `preflight`/`connecting`/`resuming`
    exits, `end_without_feedback`, starts counted at `preflight_ok`, the beacon exemption).
12. **[X] Verify:** `gofmt -l`, `go vet ./...`, `go test -race ./...` (real PG), `go test -tags e2e -race ./internal/e2e/...`, `sqlc diff`,
    the migration lint, web `typecheck`/`lint`/`test`/`build` (then `git checkout -- web/dist/.gitkeep`). In compose: as a cohort
    account, create → consent → start (503 until m6a-02) and drive the FSM through the store in a test; check `coach admin interviews --live`.
13. **[X] Merge** per land-and-sync (PR, CI green, squash). No tag.

## Constraints

- **Service boundaries (ADR-0005):** coach writes only schema `coach`; no cross-schema read; `mock_session_id` is a soft ref and stays
  NULL here (m6a-03 links it). The gateway composes and holds no state.
- **goose + sqlc:** expand-only migration, next free coach version; `sqlc diff` clean; no `Down` in prod.
- **Don't build the siblings' parts:** no snapshot route or timeline, no run echo, no `InMock`, no real picker, no SSE (m6a-04); no
  model call, no key decrypt (m6a-02). Export the interfaces they plug into.
- **C4 data:** never log transcripts, event data, code or consent text; admin output has no account id or content.
- **Dark:** cohort gate (T-1 default `cohort` + T-3 role); no T-2 env; nothing reachable by a `learner`.
- **Outbox/NATS:** no new subject, stream or consumer (so no ACL PR); if you find you need one, stop and re-plan (consumers before
  producers; the ACL PR merged before the consuming tag).
- **GitOps:** never `kubectl apply`; `ssh vps` read-only; `kubectl exec` only for admin CLIs. **D34:** no alerting, no counters job.
- **Memory-sum rule:** no new pod, no limit change (coach stays 250m / 128 Mi until mi-13).
- **Frontend:** none (UI is m6a-05/06); theme.css untouched.
- **Parallel sessions:** check peers' PRs/worktrees before claiming a migration number or an ADR number.

## Deliverables

- Coach migration + sqlc queries for the 10 tables; `internal/coach/interview/` (FSM, clock, events, sweeper, caps, consent, retention,
  handlers) with the table, model-check and property tests; the exported seams (`Clock`, `CodeSource`, `Prober`, `OnPhase`, `AddSpend`,
  the named sets, `events.Append`).
- Manifest fields, DSA values and the exported never-list + lint; regenerated golden.
- Gateway `/api/interviews/*` (cohort-gated, aud=coach, placeholder pick), arena exposure push + backstop; OpenAPI entries; coach's local lock.
- `coach admin interviews` + `coach.admin_audit`; `docs/runbooks/interviewer.md`.
- Architecture docs; the ADR-0032 dated update; task 7 recorded (or the infra PR).

## Update status

- [`../sprints/sprint-m6a-01.md`](../sprints/sprint-m6a-01.md): tasks 🔄 → ✅; _Overall_ ✅ when merged.
- [`../status.md`](../status.md): Sprint board row; **M6a** milestone "core merged (dark)"; the **L19** limits row (values, where
  enforced); the **flag inventory**: `interviewAudience = cohort` (T-1 + T-3, owner M6a, removal v2.1.0 / m6b-04); the artboard rows
  AB13, AB24–AB28 → "frozen (PR #, date)" if still unset; the release section: "from now on, run `coach admin interviews --live` before
  every tag"; **Decisions log**: the pause-limit end rule, exposure for the whole interview, the named sets, the timed alphabet, starts
  counted at `preflight_ok`, the beacon exemption, task 7's outcome.
- ADR-0032 dated update (no new ADR unless an Accepted decision is contradicted).

## Done when (acceptance)

- [ ] FSM property tests: every path terminates; `incomplete` after 24 h (from `paused` or `resuming`); the model check proves liveness
      over the timed alphabet (incl. `preflight`, `connecting`, `resuming`), ≤ 3 pauses, absorbing terminals.
- [ ] Caps enforced with typed errors (409/429/422; starts counted at `preflight_ok`); coach 409 `coach_paused` in the locked set, allowed in `paused`.
- [ ] The same-origin `text/plain` tab-close beacon is accepted; cross-site or header-less → 403.
- [ ] Checkpoints at every boundary/interruption/pause/SIGTERM; free resume state; sweeper transitions tested (incl. two sweepers).
- [ ] One `state` event per transition, `seq` strictly increasing.
- [ ] Arena access on an interview item sets `pause_exposure`.
- [ ] Manifest validated, golden regenerated, never-list lint green.
- [ ] Retention, per-turn/per-session delete and erase coverage green.
- [ ] `coach admin interviews --live` lists the live set with no PII; audited.
- [ ] Learners get 404 on `/api/interviews/*`; OpenAPI drift + route-enumeration green; task 7 recorded.

**Shipping:** per AGENT.md land-and-sync with **this sprint's release action — merge only (ships dark in the next v2.0.x
patch)**: branch → conventional commits (`feat(coach): …`) with the attribution lines → push → PR → CI green → squash-merge
(plus task 7's `../infra` PR only if it was needed) → `git checkout main && git pull` in every repo touched. **No tag** —
[m6a-06](../sprints/sprint-m6a-06.md) tags the M6a patch; any earlier v2.0.x patch carries this dark.
