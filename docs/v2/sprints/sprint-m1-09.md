# Sprint m1-09 — Curriculum spine part 2: converter, loader + guards, curriculum expand migration, content CI (M1a)

> **Milestone:** M1 — spine (**M1a expand**) · **Track:** product · **Order:** 8
> **Prereqs:** [m1-01](sprint-m1-01.md) (`internal/course`, DSA manifest, frozen item schema)
> **Unblocks:** [m1-02](sprint-m1-02.md) (cuts `v1.6.0` with this inside) · [m3-01](sprint-m3-01.md) (runs next, order 9, and is hard-gated on this merge: it extends the resolved item type, `canon.ContentHash`, `cmd/contentlint` and the `content` CI job, and takes curriculum `00004` after this sprint's `00002`/`00003`)
> **Release action:** **merge only** (ships in `v1.6.0`, cut by [m1-02](sprint-m1-02.md))
> **Calendar:** week 2 (2026-10-05 → 10-09), before m1-02 · owner event `ev-m1-statements` (~1 h: review the two rewritten Example-1 statements; must be ✅ before m1-02 tags `v1.6.0`)
> **Execute with:** [`../prompts/prompt-m1-09.md`](../prompts/prompt-m1-09.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | v1 row snapshot, one-shot converter, new layout, `ids.lock.json`, old loader deleted | X | ⬜ |
| 2 | Glob loader + guards (id `:execrows`, retire/withdrawn, course-slug guard) | X | ⬜ |
| 3 | curriculum expand migrations (`00002`, `00003`) + seed semantics + sqlc | X | ⬜ |
| 4 | Public `content` CI job (`cmd/contentlint`) | X | ⬜ |
| 5 | Replace the two copied Example-1s — agent draft (X) + owner review `ev-m1-statements` (O) | X · O | ⬜ |
| 6 | Verify (row snapshot, re-seed cases, R-b smoke on `1.5.2`, e2e, no API change) | X | ⬜ |
| 7 | Record | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> the M1 milestone row, content status, owner events). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m1-01](sprint-m1-01.md) merged: `internal/course` (manifest + item types), `curriculum/courses/dsa/course.json`,
      `curriculum/_schema/*.schema.json`, the freeze guard
- [ ] No open peer PR adds a curriculum goose migration (`gh pr list`, `git worktree list`, ListAgents): this sprint owns
      curriculum `00002` and `00003`; [m3-01](sprint-m3-01.md)'s `contract_hash` migration takes the next free number after it
- [ ] Note (not a gate): whether [m3-01](sprint-m3-01.md) has merged `internal/course/canon` decides where `content_hash`
      comes from (task 3)

## Goal

Finish the M1a curriculum spine with **no behaviour change**: convert the v1 seed into the per-course layout with a
one-shot converter proven by a **row snapshot**, replace the fixed-file seed with a **glob loader** that guards ids,
slugs and re-parenting and knows retire/withdrawn, land the curriculum **expand migration** (new columns beside the
old, old ones made write-optional, dual-written), add the public **`content` CI job**, and replace the two
LeetCode-copied Example-1s with original ones.

## Scope

**In**
- The one-shot converter (`curriculum/dsa/*.json` → `curriculum/courses/dsa/…`), `curriculum/ids.lock.json`, the old
  loader deleted in the same PR.
- Glob loader; id guard (`:execrows`, abort unless 1 row); retire/withdrawn semantics; DB-but-missing ⇒ retired; the
  course-slug guard, shared with the SPA router.
- Migrations `00002` (columns, backfills) and `00003` (`-- +goose NO TRANSACTION` unique indexes): `problem.role`,
  `status` + `retired_at`, `links`, `content_hash`; `path.id_prefix`; concept `(path_slug, slug)` key and `templates`;
  `problem_section.language` and its new unique key; the M1c-drop columns made nullable; `maxBulkProblemIDs` 256 → 1024.
- Seed semantics: every item's sections rewritten on every seed (`content_hash` written, not used as a skip key),
  delete-missing per course, dual-write of the old columns.
- `cmd/contentlint` + the `content` job in `.github/workflows/ci.yml` (schema/strict decode, id/slug guards vs the lock
  and the previous tag, Markdown profile, filename denylist, embedded-file allowlist, the row-snapshot test).
- Original Example-1s for items `3` (Two Sum) and `16` (3Sum).

**Out**
- Readers switching to the new columns, writers stopping the old ones, course-scoped routes → [m1-03](sprint-m1-03.md) (`v1.7.0`).
- The item-level retire/withdraw effects outside curriculum ([t1 §4](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual)
  "Retire semantics"), each where it really lands:
  - a withdrawn item's due touches — touch start refuses `409 item_withdrawn` in [m2-01](sprint-m2-01.md) (passed through by
    [m2-04](sprint-m2-04.md));
  - mock pools exclude retired items — v1 has no server-side pool (`mock.pools []`); the first pickers are
    [m6a-01](sprint-m6a-01.md) (placeholder pick: first evaluable, non-retired item) and [m6a-04](sprint-m6a-04.md) (live,
    non-retired pool);
  - **blocking new counted attempts on a retired item** — no scaffolded sprint owns it yet: record it in the Decisions log
    as an open item (production retires nothing before then; a retired item is reachable only by a direct URL).
- Dropping `is_reinforcement`, `leetcode_url`/`neetcode_url`, `code_template`, `UNIQUE(concept.slug)`, the old
  `problem_section` unique → [m1-08](sprint-m1-08.md) (M1c contract, `v1.8.0`).
- `contract_hash`, `grading_summary`, packlint, the pre-push hook → [m3-01](sprint-m3-01.md); `problem.spec` → [m2-01](sprint-m2-01.md).
- Widening the `path.status` CHECK for `preview`/`retired` (a CHECK swap is contract-lint territory) → the first sprint that
  seeds such a course ([p-02](sprint-p-02.md)).
- The content-CI gates that need stamps, specs or the runner — the hints/editorial **stamp gate**, compiled starters,
  reference-passes-samples, the label-edit flag ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation)) → [m3-01](sprint-m3-01.md) / M3.

## Tasks

### 1 · v1 row snapshot, converter, new layout [X]

**a. Snapshot first (its own commit).** A helper test (`-update` flag) runs today's `Seed` against a fresh PG 18
schema and writes `internal/curriculum/testdata/v1-seed-snapshot.json`: every seeded row of `path`, `phase`, `week`,
`concept`, `week_concept` (as `(week n, concept slug)` pairs), `problem`, `problem_section`, uuids stripped, ordered by
natural keys. Committing it **before** the converter proves the fixture predates the change.

> **Commit order is PR-branch evidence.** AGENT.md squash-merges every PR, so these separate commits (snapshot →
> converter + output → Example-1s) exist only on the PR branch (GitHub keeps them under the PR's Commits tab and
> `refs/pull/<N>/head`), not on `main`. The reviewer checks the ordered commits **on the PR before the squash**, and the
> PR body plus the Decisions log record the SHA of the commit that contains the converter's final source.

**b. Convert.** `hack/convert-v1-seed/main.go` runs once; its output is committed and the converter is deleted in the
same PR (its final source stays reachable at the SHA recorded in the PR body and the Decisions log). Layout per [ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships)
and [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery):

```
curriculum/
  embed.go            //go:embed paths.json ids.lock.json all:courses _schema/*.json
  paths.json          (unchanged)
  ids.lock.json       {"items": {"<id>": {"course": "dsa", "status": "live"}}, "assets": {}}  (append-only)
  embed.allowlist     sorted list of embedded files (task 4)
  README.md           layout, file roles, id rules, sidecar naming (short)
  _schema/            (m1-01)
  courses/dsa/
    course.json                              (m1-01)
    phases.json  weeks.json  concepts.json   (concepts keep `weeks[]`; body fields move out)
    concepts/<slug>.md                        body_md
    concepts/<slug>.when.md                   when_to_use_md
    concepts/_code/<slug>.go.snip             code_template → templates.go
    items/<id>/item.json                      frozen schema (m1-01); role from is_reinforcement, links from the urls
    items/<id>/sections/<stage>/<NN>-<kind>.md prose sections (NN = v1 `order`)
    items/<id>/_code/<stage>-<NN>.go.snip     kind=code sections (language = the extension before `.snip`)
```

Code lives only under `_`-prefixed directories (a `.go` file elsewhere under `curriculum/` breaks `go vet ./...`), and the
embed must be `all:courses` (plain `courses` silently drops `_` directories). **v1's code is display fragments, so it is
written as `*.go.snip`, never `*.go`:** the 4 code sections and 9 concept templates have no `package` clause and use
4-space indents, and the repo-wide `gofmt -l .` (CI `go` job, `make go-lint`) walks `_` directories too — a fragment
saved as `.go` fails with "expected package", and "fixing" it would break the byte-exact row snapshot. A `.go` file under
`curriculum/` is reserved for a **complete, gofmt-clean** Go file (M3's `_code/solution.go` references); contentlint
(task 4) rejects any `curriculum/**/*.go` that does not parse as a whole file. The loader derives `language` from the
extension before `.snip` (`go`). Each item gets `provenance`
(`origin: original` after task 5, `inspired_by` from its LeetCode link, `authored_by` from git history — the v1 seed was
written in build session S03, so `ai-assisted`; the owner confirms at `ev-m1-statements`).

**c. Delete the old loader** (`internal/curriculum/seed.go`'s fixed file list at `:18-24`) and the `curriculum/dsa/*.json`
files in the same PR.

### 2 · Glob loader + guards [X]

`internal/curriculum/load.go` replaces the fixed list ([t0 §3](../research/t0-extensibility-frame.md#3-plug-in-boundaries)):
glob `courses/*/course.json` (via `course.Load`), `courses/<slug>/{phases,weeks,concepts}.json`, `courses/<slug>/items/*/item.json`
(decoded as `course.Item` + `Validate`) and their sidecars; dotfiles ignored; strict decoding everywhere.

- **Id guard:** DSA ids `^[1-9][0-9]{0,2}$`, other courses `^<id_prefix>-[0-9]{3}$`; the `items/<id>/` directory equals
  `item.json`'s `id`; ids unique across courses; every id is in `ids.lock.json` with its course.
- **Re-parenting guard** ([t1 §4](../research/t1-content-data-model.md#4-schema-deltas-vs-v1-conceptual)): `UpsertProblem …
  ON CONFLICT (id) DO UPDATE SET … WHERE curriculum.problem.path_slug = EXCLUDED.path_slug` becomes `:execrows`; the seed
  aborts unless exactly 1 row is affected (fixes `queries/problem.sql:40-49`). Concepts upsert on `(path_slug, slug)`.
- **Retire semantics:** *retired* — hidden from the index (`ListProblemsByWeek`, `ListProblemsByPath`) and counts
  (`CountProblemsByPath`, and `ListWeeksWithCounts` — the Roadmap's per-week difficulty mix behind `GET /paths/{slug}`, so
  the counts agree with the index), still resolvable by id (`GetProblem`, `GetProblemsByIDs`); *withdrawn* — a takedown:
  prose blanked (no sections served; title kept). An item in the DB but missing from the seed is set `retired`,
  `retired_at = now()` defensively and logged at WARN (the lock should make this impossible). The effects outside
  curriculum (touches, mock pools, counted attempts) land where Scope → Out says.
- **Course-slug guard:** `^[a-z0-9]+(-[a-z0-9]+)*$` and not in `{u, auth, settings, api, assets, healthz, readyz, privacy}`
  ([ADR-0033 §6](../../adr/0033-invite-only-admission-and-owner-admin.md#6-the-acceptance-step-onboarding-step-0-l-a):
  new static SPA segments join it). **One source, one generated copy — no hand-kept second list:**
  `internal/course/reserved.go` (`ReservedSegments`) plus a Go test that writes `web/src/lib/reservedSegments.json` under
  `-update` and fails on any drift. [m1-03](sprint-m1-03.md)'s `web/src/lib/courseSlugGuard.ts` **imports that JSON** and
  re-exports it (so its "parity with the Go guard" holds by construction; the Go drift test is the parity test), and
  [l-05](sprint-l-05.md)'s check that `privacy` is reserved reads the same two files. A new reserved segment is added in
  `reserved.go` only, then `-update`. Checked at seed time and in the content job.
- `maxBulkProblemIDs` 256 → 1024 (`internal/curriculum/handlers.go:238`).

### 3 · curriculum expand migrations + seed semantics [X]

**`internal/curriculum/store/migrations/00002_v2_expand.sql`** (transactional; additive per
[ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)):

| Table | Change | Backfill |
|---|---|---|
| `problem` | `role text NOT NULL DEFAULT 'core' CHECK (role IN ('core','reinforcement','drill'))` | `'reinforcement'` where `is_reinforcement` |
| `problem` | `status text NOT NULL DEFAULT 'live' CHECK (status IN ('live','retired','withdrawn'))`, `retired_at timestamptz` | — |
| `problem` | `links jsonb NOT NULL DEFAULT '[]'` | from `leetcode_url` / `neetcode_url` (`[{kind, url}]`) |
| `problem` | `content_hash text NOT NULL DEFAULT ''` | written by the seed |
| `problem` | `is_reinforcement`, `leetcode_url`, `neetcode_url` **`DROP NOT NULL`** (write-optional for M1b) | — |
| `path` | `id_prefix text` (nullable; unique index in `00003`) | written by the seed from the manifests |
| `concept` | `templates jsonb NOT NULL DEFAULT '{}'`; `code_template` **`DROP NOT NULL`** | `{"go": code_template}` where non-empty |
| `problem_section` | `language text NOT NULL DEFAULT ''` | `'go'` where `kind = 'code'` |

**`00003_v2_expand_indexes.sql`** (`-- +goose NO TRANSACTION`): `CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS` on
`path (id_prefix)`, `concept (path_slug, slug)` and `problem_section (problem_id, stage, "order", language)`.

**Kept until M1c** (so an R-b to `1.5.2` keeps working — floor after `v1.6.0` is *none*): the old `UNIQUE(concept.slug)`
and `UNIQUE(problem_section(problem_id, stage, "order"))` that v1.5.2's upserts target, and every old column. The old
`problem_section` unique must be dropped in M1c before a second language's sections can exist: **check that it is on
[m1-08](sprint-m1-08.md)'s contract drop list** (plan and prompt). If it is missing, add it to the Decisions log as an
M1c drop-list item and say so in the PR body.

**Seed semantics** (`internal/curriculum/store` `SeedAll`, one transaction):
- `content_hash` = sha256 over the canonical re-marshal of the resolved typed item, sidecars inlined
  ([t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning)). If [m3-01](sprint-m3-01.md) has merged, use
  `internal/course/canon`; if not, create `canon.ContentHash` here (typed struct → `encoding/json`, fixed field order and
  sorted map keys → sha256) and m3-01 extends the same package. **One definition, never two.**
- **Every seed rewrites every item's sections:** `DELETE FROM problem_section WHERE problem_id = $1` then a full
  re-insert, inside the one transaction (fixes v1's lingering rows; 14 items, so the cost is negligible). `content_hash` is
  still written (it is the item's version for pinning) but is **not** used to skip children: `v1.5.2` — the R-b target
  while the floor is *none* — rewrites `problem_section` bodies through its own `UpsertSection … ON CONFLICT (problem_id,
  stage, "order") DO UPDATE` (restoring the copied Example-1s) and never touches `content_hash`, so after an R-b to
  `1.5.2` and a roll-forward an equal stored hash would keep 1.5.2's text. A skip may come back later only if it is decided
  from the **stored children** (a digest of the DB `problem_section` rows compared with the desired rows), never from
  `problem.content_hash` alone. Parent rows (`problem`, `concept`, `week`, …) are always upserted.
- Delete-missing **per course** for `phase`, `week`, `week_concept`, `concept` (content-only tables). Problems are never
  deleted — they retire.
- **Dual-write:** the seed keeps writing `is_reinforcement`, `leetcode_url`, `neetcode_url`, `code_template` from the new
  fields, so v1.5.2 reads correct values and no NULL exists before M1c.
- **Readers:** the curriculum API keeps its v1 response shape. Where a query reads a moved field it may read the new
  column with a fallback to the old (equal under dual-write); the full switch and the stop-writing step are m1-03's.
- Update `queries/{problem,section,concept,path}.sql`; `sqlc generate`; `sqlc diff` clean.

### 4 · Public `content` CI job [X]

`cmd/contentlint` (a dev/CI tool, never linked into a service binary) and a `content` job in `.github/workflows/ci.yml`
(`fetch-depth: 0` for tags; a `postgres:18` service for the snapshot test). Per [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation):

1. **Schema + strict decode** of every JSON file under `curriculum/` against `_schema/` and the Go types.
2. **Id / slug guards** vs `ids.lock.json` **and the previous release tag** (`git show <prev-tag>:curriculum/ids.lock.json`;
   skipped until a tag carries the file): no disappearing id, no course change, no `id@v` byte change; every item dir in
   the lock; course slugs pass the guard.
3. **Markdown profile** for every `.md` under `curriculum/courses/`: CommonMark + GFM tables + fenced code; images only as
   `asset:` refs; links `https://` only; **no raw HTML**; an SVG lint (no `<script>`, no `on*=`, no external refs). Parser:
   `github.com/yuin/goldmark` + its GFM extension.
4. **Filename rules** under `curriculum/`: the denylist `*.ans`, `hidden*`, `secret*`, `expected*`, `anchors*`, `exemplar*`,
   `calibration*`, `submissions/`, `wrong/`; and every `*.go` parses as a **complete** Go file (`go/parser`, package
   clause present) — fragments must be `*.go.snip` (task 1b), so the repo-wide `gofmt -l .` never meets a fragment.
5. **Embedded-file allowlist:** `go list -json ./curriculum` → `EmbedFiles`, diffed against `curriculum/embed.allowlist`;
   any extra embedded file (a dotfile, a stray) fails.
6. **The M1a row-snapshot test** (`go test ./internal/curriculum/... -run TestSeedMatchesV1Snapshot` with `XLEARN_TEST_DATABASE_URL`).

`make contentlint` runs 1–5 locally.

### 5 · Replace the two copied Example-1s [X · O]

v1 copies LeetCode's Example 1 for **item `3`** Two Sum (`curriculum/dsa/problems.json:88`) and **item `16`** 3Sum (`:249`)
([t1 §8](../research/t1-content-data-model.md#8-content-rights-stance), [ADR-0027 §8](../../adr/0027-content-evalpack-and-user-data-model.md#8-authoring-and-rights), PRD R-CT1).
- **[X]** Draft original examples **from the brief only** (never fetch or look up LeetCode): new input values in the same
  one-line style ("… → … because …"); compute the outputs by **running the item's Go reference** in a scratch test, never by hand.
- **Shape rule** (LLMs memorise popular problems, and the agent can't compare against the source — [t1 §8](../research/t1-content-data-model.md#8-content-rights-stance)):
  the new inputs must differ in **length, shape and value range** from any well-known published example — at least 6
  elements; for Two Sum include negative numbers and a negative or zero target; for 3Sum include duplicates and more than
  one resulting triplet; no input that is a short run of small positive integers.
- **Flag as unverified until reviewed:** leave both items' `review.statement` stamp **unset**, and label the PR block
  "unverified against source — owner review required". The owner's confirmation sets the stamp (a one-line follow-up).
- Land it as its own commit after the converter; that commit also updates exactly those two `problem_section` bodies in the
  snapshot fixture, so the fixture diff shows the only intended content change.
- **[O]** Owner review (~1 h, `ev-m1-statements`): the PR carries an "Owner review requested" block quoting old → new. The
  **merge** does not wait on it (the drafts replace text that is certainly copied, and the shape rule keeps them far from
  the published examples), but the **`v1.6.0` tag does**: [m1-02](sprint-m1-02.md) must not tag while `ev-m1-statements`
  is ⬜. Owner edits land as a follow-up content PR before that tag.

### 6 · Verify [X]

- `TestSeedMatchesV1Snapshot`: migrations + the new loader on a fresh PG 18 schema reproduce the snapshot exactly (the two
  task-5 bodies aside); a second table asserts the new columns (`role`, `status`, `links`, `content_hash`, `language`,
  `templates`, `id_prefix`).
- Re-seed cases (temp `fstest.MapFS`): seeding twice changes nothing (row snapshot, uuids stripped); editing one item
  changes only that item's rows; removing an item retires it (gone from the index and every count, `ListWeeksWithCounts`
  included; still resolvable by id); moving an item to another course aborts the seed; a reserved
  or malformed course slug aborts; **a 1.5.2-style writer in between** (the test runs v1's `UpsertProblem`/`UpsertSection`
  SQL with v1's bodies, leaving `content_hash` untouched) followed by the new seed → the snapshot again, new Example-1s
  restored.
- **Upgrade path:** a DB seeded by v1 code, then `00002`/`00003` + the new seed → the same rows.
- **R-b smoke (floor *none*), then roll forward:** in compose, run the published
  `ghcr.io/sujaykumarsuman/xlearn-curriculum:1.5.2` image against the expanded schema; it must migrate-noop, seed and serve.
  Then start this branch's image again: the row snapshot equals the post-task-5 fixture (the new Example-1s restored,
  `content_hash` and the new columns intact).
- `gofmt`, `go vet`, `go test -race ./...`, `sqlc diff`, web tests, e2e (`-tags e2e`, PG 18): every v1 e2e green.
- **No API behaviour change:** capture curriculum responses before and after (`/paths`, `/paths/dsa`,
  `/paths/dsa/problems`, `/paths/dsa/weeks/{n}`, `/problems/16`, `/concepts/hashing`, the bulk read) — identical apart
  from the two example bodies.

### 7 · Record [X]

[`../status.md`](../status.md): the Sprint board row; M1 🔄; content status ("DSA converted to `curriculum/courses/dsa/`;
`ids.lock.json` — 14 items"); owner event `ev-m1-statements` (⬜ until the owner confirms — it gates m1-02's `v1.6.0`
tag); Decisions log: curriculum `00002`+`00003` taken (m3-01 takes the next), the old `problem_section` unique confirmed
on (or added to) the M1c drop list, sections rewritten on every seed (no `content_hash` skip while `1.5.2` is a valid R-b
target), code fragments as `*.go.snip`, the converter's commit SHA, the open item "block new counted attempts on a
retired item — no owner yet", the deferred content gates (→ m3-01/M3), `canon.ContentHash` ownership, the contentlint
dependencies (goldmark).

## Acceptance criteria

- [ ] v1 seeded-row snapshot equal (the two rewritten example bodies aside), with snapshot → converter → Example-1s as
      separate, ordered commits in the PR (checked before the squash); the converter's commit SHA recorded in the PR body
      and the Decisions log.
- [ ] Every v1 e2e green; no API behaviour change (before/after response capture identical).
- [ ] CI `content` job green, including the row-snapshot test.
- [ ] Re-parenting aborts the seed; retire/withdrawn and delete-missing behave as specified; re-seed is idempotent.
- [ ] `sqlc diff` clean; the `1.5.2` curriculum image runs against the expanded schema (R-b smoke), and rolling forward
      afterwards restores the snapshot exactly (the new Example-1s back, not 1.5.2's text).
- [ ] Items `3` and `16` carry original Example-1s that follow the shape rule; `review.statement` stays unset and the owner
      review is requested (`ev-m1-statements`, which gates the `v1.6.0` tag, not this merge).

## Release

**Merge only — ships in `v1.6.0`**, which [m1-02](sprint-m1-02.md) tags (M1a expand; rollback floor **none**: the old
columns and uniques stay, and the seed dual-writes, so R-b to `1.5.2` works). No infra PR.

## Definition of Done

CI green (including the new `content` job) · merged to `main` · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md)) · the old `problem_section` unique confirmed on [m1-08](sprint-m1-08.md)'s drop list (or
flagged) · decisions logged.

## Risks / watch-outs

- **Converter drift silently changes a statement** — the row snapshot, committed before the converter, catches it.
- **`all:courses` embeds everything** (dotfiles, stray files) — the allowlist diff fails CI on anything unexpected.
- **A `.go` file outside `_code/`** under `curriculum/` breaks `go vet ./...`; the converter writes code only under `_`-dirs.
- **`gofmt -l .` walks `_` directories** (unlike `go vet ./...`): a v1 fragment saved as `.go` fails with "expected
  package", and an unformatted full file is listed, failing the CI `go` job and `make go-lint`. Fragments are `*.go.snip`
  (never reformatted — the snapshot is byte-exact); contentlint rejects any `curriculum/**/*.go` that is not a whole file.
- **Global concept slugs until M1c:** the old `UNIQUE(concept.slug)` still blocks a second course from reusing a DSA slug until `v1.8.0` (P comes later, so no clash).
- **A failed `CREATE INDEX CONCURRENTLY`** leaves an INVALID index that `IF NOT EXISTS` would then skip — the migration test must check `pg_index.indisvalid`.
- **m3-01 overlap** on `internal/course/canon` and the curriculum migrations — whichever merges second rebases; never two hash definitions.
- **A ~100-file diff:** review it through the fixture diff and the loader tests, not by eye.
