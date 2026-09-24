# Sprint m1-02 — path_slug everywhere + v2 envelope consumers + identity M1a columns → v1.6.0

> **Milestone:** M1 — spine (**M1a expand**) · **Track:** product · **Order:** 11
> **Prereqs:** [m1-09](sprint-m1-09.md) (and through it [m1-01](sprint-m1-01.md)) · [mi-05](sprint-mi-05.md) (N0)
> **Unblocks:** [mi-06](sprint-mi-06.md) (N2 needs the N0 tag live) · [m1-03](sprint-m1-03.md) · [m1-10](sprint-m1-10.md)
> **Release action:** **tag `v1.6.0`** (indicative: the next free minor at tag time, [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §1.6) · plus one **infra PR** (identity `NATS_URL`), merged before the tag
> **Calendar:** week 2 (by ≈ 2026-10-09)
> **Execute with:** [`../prompts/prompt-m1-02.md`](../prompts/prompt-m1-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | practice / review / assessment / coach expand (+ contract prep) | X | ⬜ |
| 2 | identity expand (role, status, admission, `admin_audit`, visibility) | X | ⬜ |
| 3 | v2 envelope, consumers first (producers stay v1) | X | ⬜ |
| 4 | Contract-header migration lint (CI) | X | ⬜ |
| 5 | Verify (sqlc diff, tests, e2e golden = v1, compose replay) | X | ⬜ |
| 6 | identity `NATS_URL` infra PR (identity becomes a NATS client in this tag) | I | ⬜ |
| 7 | Tag `v1.6.0` (release checklist) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row + milestone + tag/floor rows).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m1-01](sprint-m1-01.md) and [m1-09](sprint-m1-09.md) merged (M1a curriculum: `internal/course`, `dsa/course.json`, frozen item schema, curriculum expand `00002`)
- [ ] [mi-05](sprint-mi-05.md) merged (N0 rides `v1.6.0`; [mi-06](sprint-mi-06.md) needs it). Its review / assessment / identity migrations are numbered **before** this sprint's
- [ ] MI-2a live ✅ (bounded ranges `>=1.0.0 <2.0.0`, `.release-line` = `1`)
- [ ] No open peer PR adds a goose migration to practice, review, assessment, coach or identity (parallel sessions: `gh pr list`, `git worktree list`, ListAgents) — or it is sequenced before/after this one by agreement

## Goal

Finish the **M1a expand** across every service with **no behaviour change**: `path_slug` on every per-user
row, `total`/`max_total` and `mock_session_item` beside the v1 mock columns, identity's role / status /
admission / visibility columns plus `admin_audit`, and a shared **v2 event envelope that every consumer
decodes** (v1 and v2, forever) while producers keep emitting v1. Prepare the M1c contract by making every
column on the M1c drop list write-optional, add the contract-header lint, and cut **`v1.6.0`** with N0
([mi-05](sprint-mi-05.md)) riding it. Floor after the tag: **none (expand only)**.

## Scope

**In**
- Expand migrations in practice, review, assessment, coach and identity (all nullable, constant-default or
  new tables; `CREATE INDEX CONCURRENTLY` under `-- +goose NO TRANSACTION`), with backfills.
- **Contract prep** ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3): every
  M1c-drop column becomes write-optional so `v1.7.0` can stop writing it while an R-b to `1.6.0` still reads
  its rows; `v1.6.0` readers read `COALESCE(new, old)` and writers **dual-write**.
- The v2 envelope type in `internal/platform/events`; review (practice + notifications consumers) and
  assessment (projection consumer) decode v1 **and** v2, with v2 fixtures; producers still emit v1.
- `hack/lint-migrations.sh` wired into `ci.yml`.
- identity becomes a NATS client in this tag (mi-05's `XLEARN_IDENTITY` relay): its `NATS_URL` infra PR.
- Tag `v1.6.0` — GitHub release title `v1.6.0 — v2 build · M1a`.

**Out**
- Producers emitting the v2 envelope, readers switching to the new columns, writers dropping old columns,
  coach `page_context` prefixes → [m1-03](sprint-m1-03.md) (`v1.7.0`).
- coach keys reading `key_default` only / stop writing `is_default` → [m1-10](sprint-m1-10.md).
- Anything that **reads** `role` / `status` (sessions, `RequireRole`, CLI) → [m1-04](sprint-m1-04.md).
- `visible_courses[]` and the public filter (P2) → [m1-05](sprint-m1-05.md); toggles → [m2-03](sprint-m2-03.md).
- Any drop, CHECK removal or `SET NOT NULL` (M1c) → [m1-08](sprint-m1-08.md).
- NATS server auth (N1–N3) → [mi-06](sprint-mi-06.md).
- curriculum and gateway: no schema change here — m1-09 owns curriculum `00002`; the gateway has no DB.

## Tasks

### 1 · practice / review / assessment / coach expand [X]

Sources: [t1 §4](../research/t1-content-data-model.md) (practice, review, assessment, coach, *Expand → backfill → contract*),
[ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) (data model),
[ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3,
[ADR-0026](../../adr/0026-per-course-extensibility-model.md) §3.
**Numbering:** each service takes its **next free goose version at rebase** (mi-05 already took one in review,
assessment and identity; m1-09 owns curriculum `00002`); CI fails on a duplicate. Every file is additive only
and carries **no** `xlearn:contract` marker (task 4 lints it). Write sqlc queries for every new column and
commit `sqlc generate` output.

**practice** — `internal/practice/store/migrations/0000N_m1a_path_slug.sql`
- `user_problem_state.path_slug text NOT NULL DEFAULT 'dsa'`.
- `attempt.account_id uuid`, `attempt.path_slug text`, `attempt.problem_id text` (nullable, denormalized),
  backfilled from `user_problem_state` by `user_problem_state_id`; writers set all three on insert.
- `outbox.account_id uuid` backfilled from `payload_json->>'account_id'`; the outbox writer sets it on insert
  (erase prep, [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) §6).

**review** — `internal/review/store/migrations/0000N_m1a_path_slug.sql` + `0000N+1_m1a_weak_area_unique.sql`
- `path_slug text NOT NULL DEFAULT 'dsa'` on `revision_item`, `mistake_entry`, `weak_area_snapshot`;
  nullable `reminder.path_slug`; `outbox.account_id` (backfilled as practice).
- The second file is `-- +goose NO TRANSACTION` and only
  `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS weak_area_snapshot_account_path_week_uq ON review.weak_area_snapshot (account_id, path_slug, week_of)`.
  The v1 `UNIQUE (account_id, week_of)` **stays** (dropping it is contract work — see Risks).

**assessment** — `internal/assessment/store/migrations/0000N_m1a_totals_items.sql`
- `mock_session`: `path_slug text NOT NULL DEFAULT 'dsa'`, `rubric_id text` (backfilled with the id m1-01's
  `curriculum/dsa/course.json` gives the v1 7×5 rubric — read it, don't invent one), `rubric_snapshot jsonb`, `total integer`, `max_total integer`,
  `scored_by text CHECK (scored_by IN ('self','ai-byo'))`; `difficulty` → `DROP NOT NULL`.
  Backfill `total = total_35, max_total = 35, scored_by = 'self'` for scored rows and the snapshot from the
  manifest rubric.
- **Relax, don't drop, the status/total coupling:** the v1 table CHECK `(status = 'scored') = (total_35 IS NOT NULL)`
  (unnamed, `mock_session_check` — confirm with `\d`) would reject a `v1.7.0` scored row that no longer writes
  `total_35`. Add `mock_session_scored_total_check CHECK ((status = 'scored') = (COALESCE(total, total_35) IS NOT NULL)) NOT VALID`,
  `VALIDATE` it, then drop the old CHECK on a line marked `-- xlearn:relax replaced by a weaker CHECK` (task 4).
  `total_35` is already nullable.
- New `mock_session_item(session_id uuid REFERENCES mock_session ON DELETE CASCADE, ordinal int, item_id text NULL,
  path_slug text NOT NULL DEFAULT 'dsa', contract_hash text NULL, PRIMARY KEY (session_id, ordinal))`, one backfilled
  row per existing session (t1 §4: ordinal 1, `item_id` = the session's `problem_id`). **Rule for `problem_id = ''`:
  the row is still written, with `item_id NULL`** ("no pinned item"). v1 never picks an item: `problemId` is optional
  setup input and `''` means a mixed set (`00001_init.sql`); 1 of prod's 2 sessions is `live` with `''` (read-only
  check, 2026-09-25). The dual-write site is `handleStartMock` (`internal/assessment/handlers.go`) →
  `store.CreateMock` (`internal/assessment/store/store.go`), which inserts the session and its ordinal-1 item row in
  **one transaction**, under the same `NULL` rule. There is no `StartMock` store function.
- `outbox.account_id` (backfilled as practice; [t1 §4 assessment](../research/t1-content-data-model.md)).
- `rubric_score.dimension` CHECK stays (dropped at M1c).

**coach** — `internal/coach/store/migrations/0000N_m1a_key_default.sql`
- Nullable `path_slug` on `coach_thread` **and** `coach_message` (m1-03 writes them; m1-07 then adds only
  `prompt_v` and `attempt_id`).
- New `coach.key_default(account_id uuid, feature text NOT NULL DEFAULT 'coach' CHECK (feature IN ('coach','interview')),
  key_id uuid NOT NULL REFERENCES coach.api_key_config(id) ON DELETE CASCADE, model text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (account_id, feature))`, backfilled from the
  `is_default = true` rows (`model` from `default_model`). Every writer of `is_default` (first key, set-default,
  promote-on-delete — `internal/coach/store/queries/api_key_config.sql` + `store.go`) **dual-writes** `key_default(feature='coach')`; readers
  prefer `key_default` and fall back to `is_default`. [m1-10](sprint-m1-10.md) owns the semantics beyond this.
- `api_key_config.is_default` → `DROP NOT NULL` (keeps `DEFAULT false`).

**Readers and writers in `v1.6.0`:** assessment readers use `COALESCE(total, total_35)` / `COALESCE(max_total, 35)`;
every writer sets `path_slug = 'dsa'` explicitly and `total`/`max_total` alongside `total_35`. The public API
shape is unchanged (`total35` stays in `/api/mocks/*` JSON).

### 2 · identity expand [X]

Sources: [ADR-0033](../../adr/0033-invite-only-admission-and-owner-admin.md) §4, §7, §13; [t1 §4 identity](../research/t1-content-data-model.md); D7.
`internal/identity/store/migrations/0000N_m1a_roles_admission.sql`:
- `account.role text NOT NULL DEFAULT 'learner' CHECK (role IN ('learner','tester','owner'))`.
- `account.status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended'))`.
- `account.admitted_via text CHECK (admitted_via IN ('grandfathered','invite','cli','dev'))` — backfill
  **every existing row** to `grandfathered`, **the owner's included** (ADR-0033 §2: "Accounts created before v1.5.2
  … M1a marks them `grandfathered`"; prod has exactly 1 account, the owner's). The migration can't tell the owner
  apart, and it doesn't need to: the owner really did sign up before v1.5.2, so `grandfathered` is accurate
  provenance and stays. `set-role` (m1-04) changes `role` only, never `admitted_via`, and nothing gates on
  `admitted_via`. New rows: dev login and `open`-mode signup (compose only) write `dev`; `cli` arrives with
  [m1-04](sprint-m1-04.md), `invite` with [l-03](sprint-l-03.md).
- `account.invite_id uuid` (no FK until L-A creates `invite`), `account.accepted_at timestamptz`, `account.region text`.
- `account.profile_visibility text NOT NULL DEFAULT 'public' CHECK (profile_visibility IN ('public','private'))`.
- `path_enrollment.public_visible boolean NOT NULL DEFAULT true`; `StartEnrollment` writes the manifest's
  per-course default (`internal/course`, D7; DSA = `true`).
- New `admin_audit(id uuid PK DEFAULT gen_random_uuid(), at timestamptz NOT NULL DEFAULT now(), verb text NOT NULL,
  target text NOT NULL, detail jsonb NOT NULL DEFAULT '{}')` + index on `(at DESC)` — written by m1-04's CLI.
- **Nothing reads these columns in `v1.6.0`** (sessions, `RequireRole`, CLI and filters are M1b).

### 3 · v2 envelope, consumers first [X]

Sources: [t0 §5](../research/t0-extensibility-frame.md) (envelope rule), [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §3,
[ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3 (append-only, decoders forever), [`../../architecture/events.md`](../../architecture/events.md).
- `internal/platform/events/envelope.go` (next to `events.go`): the append-only `Envelope{EventID, Subject,
  OccurredAt, Version, AccountID, PathSlug, Data}` with `json:"path_slug,omitempty"`, a `NewEnvelope(version, …)`
  marshaller (producers adopt it in m1-03, replacing `marshalEnvelope` in practice/review/assessment and identity's
  `marshalAccountCreated`), and
  `DecodeEnvelope([]byte) (Envelope, error)`:
  - version 1 (or missing): `PathSlug = "dsa"` — **v1 envelopes only**;
  - version ≥ 2: known fields read, unknown fields ignored; a **course-scoped** subject (practice.*, review.*,
    `assessment.mock_completed`) with an empty `path_slug` is an error → the consumer dead-letters it through
    mi-05's hook (`event_dead_letter` + ERROR log; D34, no alert). Account-scoped subjects (`identity.*`) carry
    no `path_slug`. Record the course-scoped flag beside each subject in mi-05's subject registry (`topology.go`).
- Consumers switch to `DecodeEnvelope`: `internal/review/consumers.go` (practice handler), the notifications
  worker (`internal/review/notifications/worker.go`, `revision_due`), `internal/assessment/consumers.go` (projection
  handler). They **carry** `PathSlug` into the store calls (writing the new `path_slug` columns explicitly) —
  still `dsa` for every event today.
- Tests: table tests per consumer with a v1 fixture and the **same event as a v2 fixture** → identical rows /
  projections; v2-without-path → dead-lettered; unknown version 3 with extra fields → processed.
  Fixtures under `internal/platform/events/testdata/envelope/`.
- **Producers keep emitting v1** (practice, review, assessment, identity). Emitting v2 is m1-03, one tag later
  (consumers before producers, [rollout §2.2](../rollout-plan.md)).
- Document the v2 envelope and the decode rule in [`../../architecture/events.md`](../../architecture/events.md) (Conventions).

### 4 · Contract-header lint [X]

Sources: [ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §3; [`../../git-strategy.md`](../../git-strategy.md) (follow-up, code side).
- New `hack/lint-migrations.sh` (POSIX sh + grep/awk; no new toolchain), run from the `go` job in
  `.github/workflows/ci.yml` and `make lint`.
- Scope: every `internal/*/store/migrations/*.sql` **not present at `v1.5.2`** (a baseline list checked in as
  `hack/migrations-baseline.txt`, so CI needs no tag history). Pre-v2 files (e.g. coach `00003`'s `DROP CONSTRAINT`) are exempt.
- A file is **contract** if its Up section has `DROP COLUMN`, `DROP TABLE`, `DROP CONSTRAINT`, **any** `DROP INDEX`,
  `SET NOT NULL`, or **any** `ALTER COLUMN … TYPE` — unless that statement's line carries `-- xlearn:relax <reason>`
  (a reviewed relaxation, e.g. task 1's CHECK swap; also check m1-09's and mi-05's merged files and annotate any
  genuine relaxation there). The classifier is deliberately conservative: a grep can't tell whether a dropped index
  was unique or whether a type change narrows, so every `DROP INDEX` and every type change counts until a reviewer
  marks it `xlearn:relax`.
- **Marker grammar (one definition for m1-08, p-02 and ga-01):** `-- xlearn:contract floor=vX.Y.Z` (ADR-0034 §3) and
  `-- xlearn:relax <reason>`, which ADR-0034 §3 doesn't define — this sprint introduces it. Write both, with the
  statement list above, in the script's header comment and in [`../../architecture/data-model.md`](../../architecture/data-model.md)
  (Ownership rules), and record the `xlearn:relax` addition in the decisions log.
- A contract file **must** carry `-- xlearn:contract floor=vX.Y.Z` (semver); a file **without** a contract
  statement **must not** carry it. The `Down` section is ignored (never run in prod).
- Self-test: `hack/testdata/migrations/{expand,contract_ok,contract_missing,relax,marker_on_expand,drop_index_unmarked}.sql` with
  expected exit codes, run by the same CI step.

### 5 · Verify [X]

- `sqlc generate` + **`sqlc diff`** clean; `go test -race ./...`; `npm --prefix web run test`; `go vet`; gofmt.
- e2e (`go test -tags e2e ./internal/e2e/...`): **golden = v1** — every existing assertion unchanged.
- **Compose replay of 19 prod-like events:** `internal/e2e/testdata/v1-events.jsonl` — 19 **synthetic** v1
  envelopes shaped like prod's log (1 account; `problem_solved`, `attempt_logged`, `solution_revealed_early`,
  review schedules/mistakes — never copied from prod: prod data stays on the node, [ADR-0034 §4.3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#43-snapshot-rule));
  replay them through review + assessment on a fresh
  schema and compare projections to a golden; replay the same 19 re-encoded as v2 → identical. This is also
  the fixture [m1-08](sprint-m1-08.md)'s rehearsal reuses.
- Migrations on a v1.5.2-shaped database: bring compose up on `v1.5.2` images, seed via the UI, then start this
  branch's images → every service migrates, backfills are correct (`total = total_35`, `admitted_via = grandfathered`,
  `key_default` rows = `is_default` rows, one `mock_session_item` per session with `item_id NULL` for a `''` session),
  the SPA works unchanged.
- Rollback check (R-b): after the upgrade, start the **v1.5.2** images against the expanded schema → they boot
  and pass the smoke (goose ignores unknown versions; nothing they write violates the new CHECKs).

### 6 · identity `NATS_URL` infra PR [I]

mi-05 made identity publish `XLEARN_IDENTITY` through its outbox; with no `NATS_URL` identity falls back to the
log publisher and marks events sent. One PR in `../infra`, **its own task, merged before the tag**:
- `apps/xlearn-identity.yaml` env: `NATS_URL: nats://nats.messaging.svc.cluster.local:4222` (as on review).
  Harmless on `v1.5.2` (identity doesn't read it yet).
- **This PR owns identity's prod `NATS_URL`.** [mi-06](sprint-mi-06.md)'s identity N2 PR adds only the seed and
  inbox prefix, and it waits for identity's `v1.6.0` connection (as `legacy`) on `/connz`, which needs this line.
  Record this ownership in the decisions log.
- **NetworkPolicy standing rule** ([ADR-0035](../../adr/0035-v2-operations-nats-auth-limits-capacity.md) §2): identity is a new NATS caller.
  - **If MI-5's `messaging` policy ([mi-03](sprint-mi-03.md)) is live:** check that it admits identity on 4222
    (ADR-0035 §2 forward-declares it) with `ssh vps 'k3s kubectl get networkpolicy -n messaging -o yaml'`. If it
    doesn't, add identity to it in this PR.
  - **If MI-5 isn't live:** there's no policy to change. MI-5 forward-declares identity when it lands.
- Memory: no new pod; identity's limit (128 Mi) is unchanged — note the post-tag RSS in status.md.

### 7 · Tag `v1.6.0` [X]

Run the release checklist below. GitHub release title **`v1.6.0 — v2 build · M1a`**; notes: "no behaviour change;
M1a expand; consumers accept the v2 envelope; N0 (mi-05) rides this tag". In [`../status.md`](../status.md):
milestone M1a → tag `v1.6.0` → floor **none (expand only)** → snapshot **n/a**; record **N0 rode v1.6.0** (mi-06's
N2 gate); no flag change.

## Acceptance criteria

- [ ] No behaviour change: every v1 e2e green; the SPA is pixel-identical on compose.
- [ ] Every consumer (review practice + notifications, assessment projections) accepts a v2-envelope fixture and
      produces the same rows as its v1 twin; v2-without-path dead-letters.
- [ ] Backfills correct on a v1.5.2-shaped DB (totals, `grandfathered`, `key_default`, `mock_session_item`, `outbox.account_id`).
- [ ] `v1.5.2` images run against the expanded schema (R-b safe).
- [ ] `hack/lint-migrations.sh` in CI, self-tests green; no new migration carries a contract statement.
- [ ] `sqlc diff` clean; CI green.
- [ ] `v1.6.0` live and verified (healthz, images, ImagePolicies, HelmReleases Ready, smoke); identity connected
      to NATS and `XLEARN_IDENTITY` exists.

## Release

**Tag `v1.6.0`** (ADR-0034 §1.6: M1a expand; gate state after: no behaviour change; rollback floor after: none).
Release checklist ([ADR-0034](../../adr/0034-v2-release-labelling-gating-and-rollback.md) §6, verbatim, plus the ADR-0035 §2 standing rule):

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

**For this tag:** ACL PRs — n/a (NATS is either still anonymous or, once [mi-06](sprint-mi-06.md)'s N1 is live,
identity connects as `legacy`, which allows `>`; mi-05's golden already lists `XLEARN_IDENTITY`);
new service — n/a; contract / erase / GA — n/a (expand only, no snapshot required); M6 — n/a; the only new NATS
caller is identity → task 6. Extra after-tag reads: identity logs show the relay connected (no log-publisher warning),
and `XLEARN_IDENTITY` exists (read-only: the NATS monitor `/jsz?streams=true` as `host-verify --cluster` reads it).

## Definition of Done

CI green (incl. `sqlc diff` and the migration lint) · infra PR merged before the tag · `v1.6.0` tagged, deployed by
Flux (no hand `kubectl`) and verified by the checklist · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md): board, M1 milestone, tag → floor, N0 note) · notable calls in the decisions log.

## Risks / watch-outs

- **Backfill cost:** trivial on prod (1 account, 19 events); still write each backfill as one set-based `UPDATE`,
  never a row loop, and keep them idempotent (`WHERE … IS NULL`).
- **Consumers must accept v2 before any producer emits it** — producers flip in m1-03 (`v1.7.0`), never in this tag.
- **The mock CHECK swap** is the one constraint change in M1a: it only widens what is accepted (1.5.2 writes both
  columns), so R-b stays safe; the lint's `xlearn:relax` marker records the review.
- **`weak_area_snapshot`'s v1 `UNIQUE (account_id, week_of)`** would block a second course's snapshot in the same
  week. Harmless while DSA is the only course; it must be dropped before [P](sprint-p-02.md) writes snapshots —
  record it in the decisions log as an addition to [m1-08](sprint-m1-08.md)'s contract list.
- **Migration numbering races** with peers (mi-05, m1-09, m3-01): rebase last, take the next free version, let CI's
  duplicate check catch misses.
- A consumer that treats a v2 event as "unknown subject" would ack it silently — the registry test (mi-05) and the
  v2 fixtures are the guard.
