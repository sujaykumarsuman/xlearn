# Prompt — Sprint m1-02 · path_slug everywhere + v2 envelope consumers + identity M1a columns → v1.6.0

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-02.md`](../sprints/sprint-m1-02.md)   ·   **Milestone:** M1 (M1a expand)   ·   **Prereqs:** [m1-09](../sprints/sprint-m1-09.md), [mi-05](../sprints/sprint-mi-05.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m1-02.md`](../sprints/sprint-m1-02.md) (the DDL per service is spelled out there).
- [`../rollout-plan.md`](../rollout-plan.md) §4 M1 (scope T0/T7), §2.2 (operating rules), §7 (tag timeline: `v1.6.0` row).
- [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6 (tags), **§3 (expand/contract, envelope, consumers before producers)**, §6 (release checklist).
- [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §2, **§4 (identity schema)**, §7, §13.
- [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §3 (every practice/review event gains `path_slug`), §6 (migration).
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (per-user data model), [ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §1–§2 (N0, NetworkPolicy standing rule).
- [t1 §4 Schema deltas](../research/t1-content-data-model.md) (practice, review, assessment, identity, coach, *Expand → backfill → contract*); [t0 §5 envelope rule, §9](../research/t0-extensibility-frame.md).
- [`../../architecture/events.md`](../../architecture/events.md), [`../../architecture/data-model.md`](../../architecture/data-model.md), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md) (goose + advisory lock + sqlc).
- Code: `internal/{practice,review,assessment,coach,identity}/store/migrations/`, `internal/*/store/queries/`,
  `internal/platform/events/{events.go,consumer.go,topology.go}` (topology from mi-05), `internal/review/consumers.go`,
  `internal/review/notifications/worker.go`, `internal/assessment/consumers.go`, `internal/assessment/store/store.go`
  (`marshalEnvelope`, `CreateMock`, `ScoreMock`), `internal/assessment/handlers.go` (`handleStartMock`),
  `internal/assessment/store/migrations/00001_init.sql` (the unnamed status/total CHECK; `problem_id` `''` = mixed set),
  `internal/coach/store/{store.go,queries/api_key_config.sql}` (`is_default` writers), `internal/identity/store/`,
  `internal/course/` (m1-01), `.github/workflows/ci.yml`, `Makefile`, `internal/e2e/coreloop_test.go`.
- Infra (read-only until task 6): `../infra/apps/xlearn-identity.yaml`, `../infra/apps/xlearn-review.yaml` (the `NATS_URL` precedent).

## Context

M1 turns DSA into "a course like any other" with **no behaviour change**, in three tags: **M1a expand (`v1.6.0`, this
sprint)** → M1b (`v1.7.0`, m1-03…m1-07/m1-10) → M1c contract (`v1.8.0`, m1-08). m1-01/m1-09 already landed the course
manifest, the frozen item schema and the curriculum expand; mi-05 landed N0 (NATS topology, dead-letter hook, identity
outbox relay, client options, pool pins) dark. This sprint adds the per-user `path_slug` and the v2 mock/identity columns
everywhere else, makes every consumer accept the **v2 envelope** (producers keep emitting v1 — they flip one tag later),
makes every M1c-drop column write-optional (so `v1.7.0` can stop writing it while R-b to `1.6.0` still works), adds the
contract-header lint, and cuts `v1.6.0`. Production has 1 account and 19 events; the owner is the only user (D35).

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m1-01](../sprints/sprint-m1-01.md) and [m1-09](../sprints/sprint-m1-09.md) merged (`git log origin/main`; `internal/course` and curriculum `00002` exist)
- [ ] [mi-05](../sprints/sprint-mi-05.md) merged (dead-letter tables + identity relay on `main`; note its migration numbers)
- [ ] MI-2a live ✅ (`.release-line` = `1`; ranges `<2.0.0`)
- [ ] No open peer PR adds migrations to practice/review/assessment/coach/identity (`gh pr list`, `git worktree list`, ListAgents) — else agree an order

## Do this (in order)

1. **[X] Branch** `feat/m1a-expand` from an up-to-date `main`.
2. **[X] Expand migrations** (plan task 1–2), one file per service at the **next free goose version**:
   - practice: `user_problem_state.path_slug` (NOT NULL DEFAULT `'dsa'`); `attempt.account_id/path_slug/problem_id`
     (nullable, backfilled via `user_problem_state_id`); `outbox.account_id` (backfilled from `payload_json->>'account_id'`).
   - review: `path_slug` on `revision_item`, `mistake_entry`, `weak_area_snapshot` (NOT NULL DEFAULT `'dsa'`), nullable
     `reminder.path_slug`, `outbox.account_id`; a **separate** `-- +goose NO TRANSACTION` file with
     `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS … (account_id, path_slug, week_of)` (keep the v1 unique).
   - assessment: `mock_session.{path_slug, rubric_id, rubric_snapshot, total, max_total, scored_by}`, `difficulty DROP NOT NULL`,
     backfill `total = total_35, max_total = 35, scored_by = 'self'`, `rubric_id`/snapshot from `curriculum/dsa/course.json`;
     swap the status/total CHECK for `COALESCE(total, total_35)` (add `NOT VALID` → `VALIDATE` → drop the old one on a line
     marked `-- xlearn:relax …`); `mock_session_item` + one backfilled row per session (`item_id NULL` when
     `problem_id = ''`, a mixed set; prod has one such `live` session); `outbox.account_id`.
   - coach: nullable `path_slug` on `coach_thread` and `coach_message`; `coach.key_default(account_id, feature, key_id, model, updated_at)`
     backfilled from `is_default`; `is_default DROP NOT NULL`.
   - identity: `account.{role, status, admitted_via, invite_id, accepted_at, region, profile_visibility}` (CHECKs + constant
     defaults per the plan), backfill `admitted_via = 'grandfathered'` for every existing row; `path_enrollment.public_visible`
     (DEFAULT true); `admin_audit`.
   - Queries + `sqlc generate`; `sqlc diff` clean.
3. **[X] Dual-write + COALESCE readers** (no behaviour change): writers set `path_slug='dsa'` explicitly, `total`/`max_total`
   beside `total_35`, `scored_by='self'`, `mock_session_item` in `store.CreateMock` (called by `handleStartMock`; session +
   item row in one tx, same `NULL` rule), `attempt.*` denormalized ids, `outbox.account_id`,
   `key_default` wherever `is_default` is written, `admitted_via='dev'` on dev-login / `open` signup, `public_visible` from the
   manifest on enrollment. Readers: `COALESCE(total, total_35)`, `COALESCE(max_total, 35)`, `key_default` first then `is_default`.
   API JSON unchanged (`total35` stays).
4. **[X] v2 envelope** (plan task 3): `internal/platform/events/envelope.go` with `Envelope`, `NewEnvelope`, `DecodeEnvelope`
   (v1 → `path_slug="dsa"`; v2 course-scoped subject without `path_slug` → error → mi-05 dead-letter hook; unknown fields
   ignored). Mark course-scoped vs account-scoped subjects in mi-05's subject registry. Switch review's practice consumer,
   the notifications worker and assessment's projection consumer to it and pass `PathSlug` into the store. **Do not** change
   any producer's emitted version. Fixtures in `internal/platform/events/testdata/envelope/`; per-consumer v1-vs-v2 twin tests.
   Update [`../../architecture/events.md`](../../architecture/events.md) (envelope v2 + decode rule) and
   [`../../architecture/data-model.md`](../../architecture/data-model.md) (new columns/tables).
5. **[X] Migration lint** (plan task 4): `hack/lint-migrations.sh` + `hack/migrations-baseline.txt` (files present at `v1.5.2`)
   + `hack/testdata/migrations/*` self-tests; run it in `ci.yml` (`go` job) and `make lint`. Classifier: `DROP COLUMN|TABLE|CONSTRAINT`,
   **every** `DROP INDEX`, `SET NOT NULL`, **every** `ALTER COLUMN … TYPE` is contract unless its line carries `-- xlearn:relax <reason>`.
   Document both markers (`xlearn:contract floor=…`, `xlearn:relax`) in the script header and `data-model.md`. Annotate any
   genuine relaxation in m1-09's/mi-05's merged files with `-- xlearn:relax <reason>` if the lint flags it.
6. **[X] Verify** (plan task 5): `gofmt`, `go vet`, `go test -race ./...`, web tests, `-tags e2e` (golden = v1, unchanged);
   the 19-event synthetic v1 fixture replay (+ its v2 twin) against golden projections; compose upgrade from `v1.5.2`
   images with seeded data → backfills correct; then **`v1.5.2` images on the expanded schema** boot and pass the smoke (R-b).
7. **[X] PR** → conventional commit(s) `feat(m1a): …` with the attribution lines → CI green → squash-merge.
8. **[I] Infra PR** in `../infra` (its own PR, never folded into the tag): add `NATS_URL` to `apps/xlearn-identity.yaml`.
   This PR owns identity's prod `NATS_URL`; mi-06's identity N2 PR adds only the seed and inbox prefix. If MI-5 is live
   and its `messaging` NetworkPolicy doesn't admit identity on 4222, add identity in this PR; if MI-5 isn't live, there's
   nothing to change (MI-5 forward-declares identity). Merge before the tag.
9. **[X] Tag `v1.6.0`** — run every release-checklist line in the plan (parallel-sessions check first:
   `git ls-remote --tags origin`, `gh pr list`, `git worktree list`, ListAgents; next free minor; major = `.release-line`).
   Release title `v1.6.0 — v2 build · M1a`. After the tag, verify by looking (D34): healthz version, `k3s kubectl get deploy -n xlearn`
   images, ImagePolicies' latest, HelmReleases Ready, smoke login/dashboard/coach, identity relay connected, `XLEARN_IDENTITY` exists.

## Constraints

- **Service boundaries ([ADR-0005](../../adr/0005-data-ownership-and-migrations.md)):** each service migrates only its own schema;
  soft references by id; no cross-schema reads (backfills use only the service's own tables).
- **goose + sqlc:** embedded migrations under the advisory lock; commit generated code; **`sqlc diff` must pass**. Never run `Down` in prod.
- **Expand only:** nullable / constant-default columns, new tables, `CONCURRENTLY` under `-- +goose NO TRANSACTION`. No
  `xlearn:contract` marker anywhere in this sprint; the single CHECK swap is a marked `xlearn:relax`.
- **Envelope is append-only; decoders stay forever** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3).
  **Consumers before producers:** no producer emits v2 in this tag.
- **Outbox/inbox:** domain row + outbox row in one transaction; consumers dedupe on `event_id`.
- **GitOps only:** the infra change is a PR in `../infra`; never `kubectl apply`. Read-only `ssh vps` is fine for verification.
- **D34:** no alerting, no Flux Alert, no opscheck — failures are logs + dead-letter rows read on demand.
- **Memory-sum rule** ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §5): no new pod; identity keeps its limit.
- **NetworkPolicy standing rule:** identity is a new NATS caller in this tag → its policy is checked/changed in its own infra PR before the tag.
- **Parallel sessions:** check peers' PRs, tags and worktrees before numbering migrations and before tagging; never move or re-push a tag.

## Deliverables

- Expand migrations + sqlc queries/gen for practice, review, assessment, coach, identity (backfills included).
- Dual-writes and `COALESCE` readers; `key_default` dual-write in coach.
- `internal/platform/events/envelope.go` + fixtures; review/assessment consumers on `DecodeEnvelope`; registry flags.
- `hack/lint-migrations.sh`, `hack/migrations-baseline.txt`, self-tests, CI step.
- `internal/e2e/testdata/v1-events.jsonl` + replay test (v1 and v2 twin).
- Updated `docs/architecture/events.md` and `data-model.md`.
- Infra PR: identity `NATS_URL` (+ policy if needed).
- Tag `v1.6.0`, verified.

## Update status

- This plan's Status table ([`../sprints/sprint-m1-02.md`](../sprints/sprint-m1-02.md)): each task 🔄 → ✅; _Overall_ ✅ at the end.
- [`../status.md`](../status.md): Sprint board row (m1-02 ✅); **Milestones** (M1: M1a shipped); **milestone → tag → floor → snapshot**
  row `M1a → v1.6.0 → none (expand only) → n/a`; note **N0 (mi-05) rode v1.6.0** (mi-06's N2 gate); flag inventory unchanged;
  MI row MI-6 → shipped in `v1.6.0`.
- Decisions log: the mock CHECK relax and the new `xlearn:relax` marker (beyond ADR-0034 §3's `xlearn:contract`), the
  `weak_area_snapshot` v1 unique to be added to m1-08's drop list, the identity `admitted_via` backfill rule (every existing
  row, owner included), `mock_session_item.item_id NULL` for mixed-set sessions, identity's prod `NATS_URL` owned by this
  sprint's infra PR (not mi-06 N2), the envelope course-scoped flag. Record an ADR only if a call goes beyond ADR-0033/0034
  (check peers' ADR numbers first).

## Done when (acceptance)

- [ ] No behaviour change: every v1 e2e green; the SPA is unchanged on compose.
- [ ] Every consumer accepts a v2-envelope fixture with the same result as its v1 twin; v2-without-path dead-letters.
- [ ] Backfills correct on a v1.5.2-shaped DB; `v1.5.2` images run on the expanded schema.
- [ ] Migration lint in CI with green self-tests; `sqlc diff` clean; CI green.
- [ ] Infra `NATS_URL` PR merged before the tag.
- [ ] `v1.6.0` live and verified (healthz, images, policies, HelmReleases, smoke; identity on NATS).

Ship per AGENT.md land-and-sync with **this sprint's release action: tag `v1.6.0`** (plus the identity infra PR merged
before it) — then `git checkout main && git pull` in both repos.
