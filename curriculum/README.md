# curriculum/ — public course content

Everything here is **public** and is embedded into the curriculum service binary
(`embed.go`: `//go:embed paths.json ids.lock.json all:courses _schema/*.json`). The curriculum
service seeds it into schema `curriculum` on startup, in one transaction
([ADR-0026](../docs/adr/0026-per-course-extensibility-model.md), [ADR-0027 §2](../docs/adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships)).
Nothing private ever lives here: hidden tests, answer keys, rubric anchors and exemplars live in the
private eval pack. **Never copy content from `../xlearn-evalpack` into this repo.**

## Layout

```
curriculum/
  embed.go          the go:embed directive (data only, no logic)
  paths.json        the catalog: one row per course (Catalog card: title, status, summary, totals, order)
  ids.lock.json     every item id ever published, with its course and status (append-only)
  embed.allowlist   the exact list of embedded files; CI fails on any other embedded file
  _schema/          JSON Schemas for every JSON file below (item.schema.json is FROZEN, additive only)
  courses/<slug>/
    course.json                              the course manifest (settings only)
    phases.json  weeks.json  concepts.json   rows; the course is the directory (no path_slug field)
    concepts/<slug>.md                       concept body (Markdown)
    concepts/<slug>.when.md                  "when to use" (Markdown)
    concepts/_code/<slug>.<lang>.snip        code template fragment (language = extension before .snip)
    items/<id>/item.json                     one item (frozen item schema v1)
    items/<id>/sections/<stage>/<NN>-<kind>.md   prose section: stage attempt|hint|solution, NN = order (01..99)
    items/<id>/_code/<stage>-<NN>.<lang>.snip    code section fragment at that stage and order
    items/<id>/_code/solution.{go,cpp,py}         the public reference: a whole, compilable file (M3)
```

**Authoring graded items and their private eval packs:** see [`docs/v2/authoring.md`](../docs/v2/authoring.md)
(the two halves, hashes, stamps, packlint and the pre-push hook).

## Rules

- **Ids are forever.** An item's directory name is its `id`. DSA ids are bare numbers (`^[1-9][0-9]{0,2}$`);
  every other course uses `<id_prefix>-NNN`. An id is never reused, never moved to another course and
  never deleted: set `"status": "retired"` (hidden from the index and counts, still resolvable by id) or
  `"withdrawn"` (a takedown: no sections served). Every item is in `ids.lock.json` with its course and
  status; the lock only grows.
- **Course slugs** match `^[a-z0-9]+(-[a-z0-9]+)*$` and are not a reserved SPA segment
  (`internal/course/reserved.go`, the only list; `web/src/lib/reservedSegments.json` is generated from it).
- **Sidecars are byte-exact.** A section or concept file is served exactly as stored: no trailing newline is
  added or stripped. An absent sidecar is the empty string.
- **Code lives only under `_`-prefixed directories.** Display fragments (no `package` clause) are `*.snip`,
  never `*.go`: `gofmt -l .` walks `_` directories. A `.go` file anywhere under `curriculum/` must be a
  complete, gofmt-clean Go file.
- **Markdown profile:** CommonMark + GFM tables and fenced code; links `https://` only; images only as
  `asset:` refs; no raw HTML.
- **Filenames never look private:** `*.ans`, `hidden*`, `secret*`, `expected*`, `anchors*`, `exemplar*`,
  `calibration*`, `submissions/`, `wrong/` and the pack's own names (`pack.json`, `tests.lock`, `cases.jsonl*`,
  `*.jsonl.zst`, `edge.jsonl`, `invalid/`, `gen-hidden*`, `instances.json`, `timing.json`) are rejected; public
  `keys/*.json` alias tables stay allowed. Dotfiles are ignored by the loader and rejected by the embed allowlist.
- **Hints and editorial need their stamp.** An added or changed `sections/hint/*` or `sections/solution/*` file
  (or `_code/hint-NN.*.snip`, `_code/solution-NN.*.snip`) fails CI unless the item's `review.hints` /
  `review.editorial` is set. v1's converted sections are grandfathered until they change.
- **Labels on key-graded parts and probes** keep their meaning: a changed label under the same id fails CI unless
  the PR body says `label-edit-ok: <item>/<part>/<id>` (a typo); otherwise mint a new id.
- **Provenance** is required on every item (`origin` original | adapted | licensed; `inspired_by` is an
  idea reference such as `leetcode:two-sum`, never copied text). LeetCode stays an outbound link.

## Checks

`make contentlint` runs the public content checks locally (schema + strict decode, id and slug guards
against the lock and the previous release tag, the t4 §5.6 structure lints, the stamp gate and the
label-edit flag against the PR base, the Markdown profile, filename rules, the repo-wide pack-artefact
pass, the embed allowlists). After adding or removing a file, refresh the allowlist with
`go run ./cmd/contentlint -write-allowlist` and review the diff. CI's `content` job runs the same
checks plus the seeded-row snapshot test (`TestSeedMatchesV1Snapshot`) on Postgres 18.
