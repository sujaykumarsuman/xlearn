# Prompt — Sprint m1-09 · Curriculum spine part 2: converter, loader + guards, curriculum expand migration, content CI (M1a)

> **One self-contained prompt = one sprint = one session.** Paste into a fresh coding session at the repo root.
> **Plan:** [`../sprints/sprint-m1-09.md`](../sprints/sprint-m1-09.md)   ·   **Milestone:** M1 (M1a expand)   ·   **Prereqs:** [m1-01](../sprints/sprint-m1-01.md)

## Read first

- [`../../../CLAUDE.md`](../../../CLAUDE.md) / [`../../../AGENT.md`](../../../AGENT.md) — conventions, land-and-sync.
- The plan: [`../sprints/sprint-m1-09.md`](../sprints/sprint-m1-09.md) — the target layout, the migration table, the seed
  semantics and the six content-job checks are spelled out there. Follow them.
- [`../rollout-plan.md`](../rollout-plan.md) §4 M1 (scope T0/T1), [§2.2](../rollout-plan.md#22-operating-rules-every-mi-step-and-every-tag) (operating rules).
- [ADR-0026](../../adr/0026-per-course-extensibility-model.md) §2 (items: global ids, never re-parented, retired not deleted), §6.
- [ADR-0027](../../adr/0027-content-evalpack-and-user-data-model.md) [§1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule),
  [§2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships) (layout, `all:courses`, hashes),
  [§8](../../adr/0027-content-evalpack-and-user-data-model.md#8-authoring-and-rights) (rights: the two copied examples).
- [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules) (expand only;
  `CREATE INDEX CONCURRENTLY` under `-- +goose NO TRANSACTION`; floor), [ADR-0005](../../adr/0005-data-ownership-and-migrations.md),
  [ADR-0012](../../adr/0012-curriculum-content-model-and-seeding.md) (v1 seeding), [ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a) (`privacy` joins the slug guard).
- Research: [t0 §3](../research/t0-extensibility-frame.md#3-plug-in-boundaries), [t0 §9](../research/t0-extensibility-frame.md#9-how-live-dsa-migrates);
  [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery), [§3.4](../research/t1-content-data-model.md#34-versioning-and-pinning),
  [§4 curriculum + Expand → backfill → contract](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual),
  [§7.2](../research/t1-content-data-model.md#72-ci-validation), [§8](../research/t1-content-data-model.md#8-content-rights-stance).
- Code: `curriculum/{embed.go,paths.json,dsa/*.json}`, `curriculum/courses/*/course.json` and `curriculum/_schema/` (m1-01),
  `internal/course/` (m1-01), `internal/curriculum/{seed.go,handlers.go,service.go}`,
  `internal/curriculum/store/{migrations/00001_init.sql,queries/*.sql,seed.go,store.go,store_integration_test.go}`,
  `sqlc.yaml`, `.github/workflows/ci.yml`, `Makefile`, `web/src/lib/`, `internal/e2e/`.

## Context

m1-01 added the typed course manifest, the frozen item schema and prod-parity compose. This sprint finishes the M1a
curriculum slice so [m1-02](../sprints/sprint-m1-02.md) can cut `v1.6.0` (expand only, floor none). v1 seeds from a
fixed list of `curriculum/dsa/*.json` files with upserts that silently **re-parent** rows between courses; ids have no
guard; retired items would linger; sections never get deleted; two statements copy LeetCode's Example 1. After this
sprint: DSA lives in `curriculum/courses/dsa/` (one directory per item, Markdown sidecars, code under `_code/`), a glob
loader enforces id/slug/re-parenting guards and retire semantics, the curriculum schema has its v2 columns **beside**
the v1 ones (dual-written, old ones write-optional), a public `content` CI job guards every content PR, and the two
examples are original. The v1 API and every e2e stay byte-identical. v2 is owner-only (D35); production has 14 seeded
items and one account.

## Entry gates — verify first (stop and report if any is unmet)

- [ ] [m1-01](../sprints/sprint-m1-01.md) merged (`internal/course/`, `curriculum/courses/dsa/course.json`,
      `curriculum/_schema/item.schema.json`, the freeze guard on `main`).
- [ ] No open peer PR adds a curriculum goose migration (`gh pr list`, `git worktree list`, ListAgents). You own
      curriculum `00002` + `00003`; [m3-01](../sprints/sprint-m3-01.md) numbers after you.
- [ ] Note (not a gate): check whether `internal/course/canon` exists on `main` (m3-01) — it decides task 3's hash source.

## Do this (in order)

1. **[X] Branch** `feat/m1-curriculum-loader` off an up-to-date `origin/main`.
2. **[X] Snapshot (commit 1)** — add the `-update` helper test that seeds a fresh PG 18 schema with today's loader and
   writes `internal/curriculum/testdata/v1-seed-snapshot.json` (all seeded rows, uuids stripped, natural-key order). Commit it alone.
   The PR is squash-merged, so the commit order (snapshot → converter → Example-1s) is evidence **on the PR branch only**:
   keep them as separate commits, never squash locally, and check them on the PR yourself before the merge (they stay visible on the PR afterwards).
3. **[X] Migrations** — `internal/curriculum/store/migrations/00002_v2_expand.sql` (the plan's column table with backfills
   and the `DROP NOT NULL`s) and `00003_v2_expand_indexes.sql` (`-- +goose NO TRANSACTION`; three
   `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS`). Keep every old column and the old uniques on `concept.slug` and
   `problem_section(problem_id, stage, "order")`. Add a migration test that asserts all indexes are valid (`pg_index.indisvalid`).
4. **[X] Converter (commit 2)** — `hack/convert-v1-seed/main.go`; run it; commit the converter **with** its output layout
   (`courses/dsa/…`, `ids.lock.json`, `curriculum/README.md`); delete the converter and `curriculum/dsa/*.json` in a later
   commit of the same PR, and record commit 2's SHA in the PR body and the Decisions log. Items get `role`, `status: live`,
   `links`, `provenance` (`inspired_by` from the LeetCode link, `authored_by: ai-assisted` per git history). Code only
   under `_code/`, and v1's code fragments (no `package` clause, 4-space indents) are written **byte-exact as `*.go.snip`**
   (`items/<id>/_code/<stage>-<NN>.go.snip`, `concepts/_code/<slug>.go.snip`; language = the extension before `.snip`) —
   never `.go`: the repo-wide `gofmt -l .` walks `_` directories and would fail on a fragment.
5. **[X] Loader + guards** — `internal/curriculum/load.go` (globs via `course.Load` and `course.Item`), the id guard, the
   `:execrows` re-parenting guard (abort unless 1 row), concept upsert on `(path_slug, slug)`, retire/withdrawn semantics
   (retired hidden from `ListProblemsByWeek`, `ListProblemsByPath`, `CountProblemsByPath` **and `ListWeeksWithCounts`**;
   still resolvable by id), DB-but-missing ⇒ retired (WARN), the course-slug guard from `internal/course/reserved.go`
   (the **only** list) plus the Go test that generates `web/src/lib/reservedSegments.json` (`-update`, fails on drift) —
   m1-03's `courseSlugGuard.ts` imports that JSON, so there is no second hand-kept list; `maxBulkProblemIDs` → 1024. Switch `curriculum/embed.go` to
   `//go:embed paths.json ids.lock.json all:courses _schema/*.json`. Remove the old fixed list in `seed.go`.
6. **[X] Seed semantics + sqlc** — `content_hash` from `internal/course/canon` (create `canon.ContentHash` here — m3-01 runs after this sprint and reuses it; one definition), written but **not** used as a skip key: every seed deletes and re-inserts every
   item's sections (`1.5.2`, the R-b target, rewrites section bodies without touching `content_hash`, so a hash skip would
   keep its text after a roll-forward), delete-missing per course for
   phase/week/week_concept/concept, dual-write of `is_reinforcement`/`leetcode_url`/`neetcode_url`/`code_template`, v1
   response shape kept. Update `queries/*.sql`, `sqlc generate`, `sqlc diff` clean.
7. **[X] Content CI** — `cmd/contentlint` (schema + strict decode, id/slug guards vs the lock and the previous tag,
   Markdown profile with goldmark + GFM, filename denylist + "every `curriculum/**/*.go` is a complete Go file",
   embedded-file allowlist vs `curriculum/embed.allowlist`),
   `make contentlint`, and a `content` job in `ci.yml` (`fetch-depth: 0`, `postgres:18` service) that runs contentlint and
   `TestSeedMatchesV1Snapshot`. Confirm contentlint's deps never reach a service binary (`go list -deps ./cmd/<svc>`).
8. **[X] Example-1s (commit 3)** — original examples for items `3` and `16`, drafted **from the brief only** (never look up
   LeetCode); outputs computed by running each item's Go reference in a scratch test; update exactly those two bodies in the
   snapshot fixture in the same commit. **Shape rule:** inputs differ in length, shape and value range from any well-known
   example — ≥ 6 elements; Two Sum with negative numbers and a negative or zero target; 3Sum with duplicates and more than
   one resulting triplet; never a short run of small positive integers. Leave both items' `review.statement` **unset**.
   In the PR body add an "Example-1 rewrites — unverified against source" block quoting old → new, for the owner's later
   look. The drafts land as written (D40; `ev-m1-statements` is automatic): neither this merge nor m1-02's `v1.6.0` tag
   waits on a review, and any owner edit is a later content PR.
9. **[X] Verify** — the plan's task 6: snapshot equal (two bodies aside), new-column table, re-seed cases (idempotent,
   one-item change, retire on removal incl. `ListWeeksWithCounts`, abort on re-parent and bad slug, **a 1.5.2-style
   writer (v1's `UpsertProblem`/`UpsertSection` SQL with v1's bodies) then the new seed → snapshot, new Example-1s
   restored**), upgrade path (v1-seeded DB → migrate → new seed), **R-b smoke then roll forward** (compose
   `ghcr.io/sujaykumarsuman/xlearn-curriculum:1.5.2` against the expanded schema: it seeds and serves; then this branch's
   image again: row snapshot equal, new Example-1s back), `gofmt -l .` empty (no `.go` fragments under `curriculum/`) /
   `go vet`/`go test -race ./...`/`sqlc diff`/web tests/e2e on PG 18, and a before/after capture of the curriculum
   responses (`/paths`, `/paths/dsa`, `/paths/dsa/problems`, `/paths/dsa/weeks/{n}`, `/problems/16`, `/concepts/hashing`,
   the bulk read — identical apart from the two example bodies).
10. **[X] Update status** (below), then conventional commit(s) (e.g. `feat(curriculum): per-course layout, glob loader +
    guards, v2 expand migration, content CI`) with the attribution lines; push; open the PR, and ship it (see Ship).

## Constraints

- **No behaviour change:** v1 response shapes, routes and events unchanged; readers switch in m1-03, writers stop in m1-03.
- **Expand only (ADR-0034 §3):** nullable or constant-default columns, backfills, `CREATE INDEX CONCURRENTLY` in a
  `NO TRANSACTION` file; no `DROP COLUMN/CONSTRAINT`, no `SET NOT NULL`. Floor after `v1.6.0` is none — the `1.5.2` image
  must still run. Never run `Down` in production.
- **Service boundaries (ADR-0005):** only schema `curriculum`; goose migrations embedded and run on startup under the
  advisory lock; commit sqlc output (`sqlc diff` in CI).
- **One hash definition:** `content_hash` comes from `internal/course/canon`; never a second implementation.
- **Rights (R-CT1):** original examples from the brief only, following the shape rule; LeetCode stays an outbound link;
  never fetch it. The drafts land as written, marked "unverified against source" (`review.statement` unset) until any
  later owner review sets the stamp (D40).
- **Answer secrecy:** the filename denylist and allowlist exist so nothing private is ever embedded; never copy content
  from `../xlearn-evalpack` into this repo.
- **Dependencies:** goldmark (and any JSON Schema validator) only in `cmd/contentlint` and tests.
- **GitOps / infra:** no `../infra` change; never `kubectl apply`. D34: no alerting work. No new pod (memory-sum rule
  untouched). No events, so consumers-before-producers and the ACL rule do not apply.
- **Parallel sessions:** before merging, check peers' PRs, tags and worktrees (`gh pr list`, `git ls-remote --tags origin`,
  `git worktree list`, ListAgents), especially m3-01 (canon, curriculum migrations) and mi-05 (`ci.yml`); rebase onto a moved `main`.

## Deliverables

- `curriculum/courses/dsa/**` (code fragments as `*.go.snip`), `curriculum/{ids.lock.json,embed.allowlist,README.md,embed.go}`;
  `curriculum/dsa/` removed.
- `internal/curriculum/{load.go,seed.go (fixed list removed),handlers.go}`; `internal/course/reserved.go` (+ `canon/` if absent);
  `web/src/lib/reservedSegments.json`.
- `internal/curriculum/store/migrations/{00002_v2_expand.sql,00003_v2_expand_indexes.sql}`, updated queries + sqlc output.
- `cmd/contentlint`, `make contentlint`, the `content` job in `.github/workflows/ci.yml`.
- `internal/curriculum/testdata/v1-seed-snapshot.json` and the loader/migration tests.

## Update status

- [`../sprints/sprint-m1-09.md`](../sprints/sprint-m1-09.md): each task 🔄 → ✅; _Overall_ ✅ once all are.
- [`../status.md`](../status.md): Sprint board row; **M1** milestone row; **content status** ("DSA converted to
  `curriculum/courses/dsa/`; `ids.lock.json`: 14 items"; "Example-1 statements rewritten 2/2, landed as drafted"); **owner
  events** `ev-m1-statements` ✅ automatic at the merge (D40; it gates nothing); **Decisions log**: curriculum `00002` + `00003` taken (m3-01 takes the
  next), the old `problem_section` unique **confirmed on [m1-08](../sprints/sprint-m1-08.md)'s M1c drop list** (or flagged
  there if missing), sections rewritten on every seed (no `content_hash` skip while `1.5.2` is an R-b target), fragments as
  `*.go.snip`, the converter's commit SHA, the open item "block new counted attempts on a retired item — no owner yet",
  the deferred content gates (→ m3-01/M3), the `canon.ContentHash` owner, contentlint's dependencies.
- No ADR: this implements ADR-0026/0027 as decided (layout per t1 §3.2). Record any real deviation as an ADR amendment.

## Done when (acceptance)

- [ ] v1 seeded-row snapshot equal (the two rewritten example bodies aside); snapshot → converter → Example-1s are separate,
      ordered commits in the PR (checked before the squash); the converter's SHA recorded.
- [ ] Every v1 e2e green; no API behaviour change.
- [ ] CI `content` job green, including the row-snapshot test.
- [ ] Re-parenting aborts the seed; retire/withdrawn and delete-missing behave as specified; re-seed is idempotent.
- [ ] `sqlc diff` clean; the `1.5.2` curriculum image runs against the expanded schema, and rolling forward restores the
      snapshot (new Example-1s back).
- [ ] Items `3` and `16` carry original Example-1s that follow the shape rule; `review.statement` unset; the drafts land as
      written (`ev-m1-statements` automatic, D40).

## Ship (land-and-sync — owner approval pre-granted)

> Launching this prompt is the owner's approval for every change it makes (D40); don't stop for review.

1. Branch `feat/m1-curriculum-loader`, then the ordered conventional commits (snapshot → converter → Example-1s, then the rest) with the attribution lines, then push, then the PR. This repo only: no `../infra` PR.
2. Once CI is green (fix, then merge, on failure), squash-merge. Never enable auto-merge. The Example-1 drafts merge as written; nothing waits on an owner review.
3. **Release action — merge only:** nothing deploys (`main` is build-only). It ships in **`v1.6.0`**, which [m1-02](../sprints/sprint-m1-02.md) tags; that tag doesn't wait on a statement review. No tag here.
4. Update status: the sprint file and `docs/v2/status.md`, in the same PR or a follow-up docs PR merged the same way (record the converter commit's SHA).
5. Run `git checkout main && git pull`. If a clean peer worktree holds `main`, use `git -C <worktree> merge --ff-only origin/main` and then `git switch --detach main`.
