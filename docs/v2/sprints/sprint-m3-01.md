# Sprint m3-01 — Authoring tooling T25/T26: canonical hashes, contract_hash, packlint, pre-push fingerprint hook

> **Milestone:** M3 — judge plus code grader (content track: the "T25/T26 tooling" line of the M3 hard checklist) · **Track:** content · **Order:** 9 — runs **right after** [m1-09](sprint-m1-09.md) (order 8)
> **Prereqs:** [m1-01](sprint-m1-01.md) (item schema frozen, `internal/course`) · [m1-09](sprint-m1-09.md) (the whole sprint builds on it: the resolved item type and `canon.ContentHash`, `cmd/contentlint` + the `content` CI job, `curriculum/README.md`, curriculum `00002`/`00003`, `problem.content_hash`)
> **Unblocks:** [m3-02](sprint-m3-02.md) (the evalpack pipeline) · owner event `ev-packs-14` (author the 14 pilot packs with this tooling) · also the **single** hash and lint definitions that [m2-01](sprint-m2-01.md) (`canon.PolicyVersion` on attempts), [m3-05](sprint-m3-05.md) (judge's contract check; t4 §5.6 re-runs the lints at judge start) and [m3-08](sprint-m3-08.md) (contract pinning) import — never re-implemented
> **Release action:** **merge only** — ships dark in the next app tag (`v1.6.0` if merged before [m1-02](sprint-m1-02.md) cuts it, else `v1.7.0`). packlint, contentlint and the hook are dev tools and never enter an image.
> **Calendar:** ≈ 2026-10-06 → 10-07, right after `ev-schema-freeze` and m1-09's merge (week 2, before m1-02 tags) · owner installs the hook (~5 min)
> **Execute with:** [`../prompts/prompt-m3-01.md`](../prompts/prompt-m3-01.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Canonical hash package `internal/course/canon` (+ field classification, golden vectors) | X | ⬜ |
| 2 | `internal/packspec` source types + `cmd/packlint` (`check`, `hash`) | X | ⬜ |
| 3 | Public content gates deferred from m1-09: stamp gate, label-edit flag, t4 §5.6 structure lints | X | ⬜ |
| 4 | Pre-push fingerprint hook (`packlint fingerprint`) + `make install-hooks` | X | ⬜ |
| 5 | Repo hygiene: `.gitignore` / `.dockerignore`, AGENT.md rule, leak-lint extensions | X | ⬜ |
| 6 | Authoring guide `docs/v2/authoring.md` | X | ⬜ |
| 7 | curriculum `contract_hash` + `grading_summary` (migration `00004`, seed, `GET /problems/{id}`) | X | ⬜ |
| 8 | Verify + record | X | ⬜ |
| 9 | Owner installs the hook on the authoring machine | O | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row,
> the M3 milestone row and its T25/T26 checklist line, content status, owner events). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] Item schema frozen: [m1-01](sprint-m1-01.md) merged (`internal/course` item + manifest types, `curriculum/_schema/*.schema.json`,
      the freeze guard) and `ev-schema-freeze` recorded in [`../status.md`](../status.md)
- [ ] [m1-09](sprint-m1-09.md) merged — **a gate for the whole sprint**, not only task 7: it creates the loader's resolved item type
      and `canon.ContentHash` (task 1 extends them), `cmd/contentlint` and the `content` CI job (tasks 3, 5 and 8 extend them),
      `curriculum/README.md` (task 6 links from it) and curriculum `00002` + `00003` with `problem.content_hash` (task 7 takes the
      next free number, `00004`). Never take `00002`/`00003`.
- [ ] Parallel sessions: no open PR creates `internal/course/canon` (beyond m1-09's `ContentHash`), `internal/packspec`, `cmd/packlint`,
      `hack/git-hooks/` or a curriculum migration (`gh pr list`, `git worktree list`, ListAgents) — one definition.

_Informational, not a gate:_ `internal/course/canon` normally exists on `main` (m1-09 created `ContentHash`), so task 1 **extends** it.
If m1-09 kept a local hash helper instead, task 1 creates the package and moves that helper into it (same bytes).

### M3 hard entry checklist (rollout §5, verbatim) — context, not this sprint's gate

This checklist gates the M3 UI sprint ([m3-11](sprint-m3-11.md)) and the M3-2 tag ([m3-13](sprint-m3-13.md)). This sprint
delivers the **T25/T26** line.

- [ ] Spike P0–P3 **GO** and the image-volume spike **GO** (MI-10)
- [ ] MI-4, MI-5, **MI-5a**, **MI-7 (N3)**, MI-9, MI-11, **MI-11a**, MI-12 and MI-13 done
- [ ] **MI-8: `host-verify --cluster` (extended) green.** The memory sum, *with the runner's 3 GiB counted*, is ≤ capacity − 0.5 GiB; no OOMKills; PVCs < 60%; NATS `auth_required`; the NetworkPolicies are present
- [ ] T25/T26 tooling (pre-push fingerprint hook, packlint, `contract_hash`)
- [ ] **14 pilot packs stamped** (Go, C++ and Python references; about 28–41 owner hours)
- [ ] `account.role` live (M1a), so the owner and tester cohort gates judge features
- [ ] TR-STEAL not firing (sar p95 read by `host-verify`), or R2 planned
- [ ] AB07–AB12 frozen
- [ ] The last Hostinger weekly image is ≤ 7 days old

## Goal

Give the owner the tooling to author eval packs **safely** right after the M1a schema freeze
([rollout §6](../rollout-plan.md#6-critical-path-parallel-tracks-owner-calendar): the owner-hours chain starts at the freeze):
- **one canonical-hash package** shared by the curriculum seed, practice, judge and packlint, defining `policy_version`,
  `content_hash` and the narrow `contract_hash` ([ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships),
  [t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning));
- **`packlint`**, the linter that sees both halves (public item + private pack);
- the **pre-push fingerprint hook** — the only check that sees both repos **before** anything is public
  ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation));
- the repo guards and an authoring guide, plus `problem.contract_hash` and a derived `grading_summary` in curriculum.

ADR-0027 is **Accepted** (BP2, 2026-09-24): this sprint implements it as decided.

## Scope

**In**
- `internal/course/canon`: canonical re-marshal + sha256, `PolicyVersion`, `ContentHash`, `ContractHash`, the field
  classification and golden vectors.
- `internal/packspec` (private-pack **format** types — never data) and `cmd/packlint` (`check`, `hash`, `fingerprint`).
- Public content gates [m1-09](sprint-m1-09.md) deferred here: the hints/editorial **stamp gate**, the **label-edit flag** on
  `key`-graded parts, and the public-registry half of the [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) structure lints.
- `hack/git-hooks/pre-push` + `make install-hooks`.
- `.gitignore` / `.dockerignore` entries, the AGENT.md never-copy rule, leak-lint extensions.
- `docs/v2/authoring.md`.
- curriculum `problem.contract_hash` + `grading_summary` (M3 expand, migration `00004`) and the prefix on `GET /problems/{id}`.

**Out**
- Private CI gates, pack format v1 (cases, `tests.lock`, generators), the generic validator, the image build, the synthetic
  fixture pack and the compose mount → [m3-02](sprint-m3-02.md).
- Pack content: the 14 pilot packs → owner event `ev-packs-14`.
- judge's use of the hashes (evaluable index, `spec_mismatch`, `inconclusive(contract_changed)`) → [m3-05](sprint-m3-05.md) /
  [m3-06](sprint-m3-06.md); practice pinning `contract_hash` on attempts → [m3-08](sprint-m3-08.md); `policy_version` on
  attempts → [m2-01](sprint-m2-01.md) (it must call `canon.PolicyVersion`, never hash the manifest itself).
- Compiled starters and "the reference passes the samples" in public CI → [m3-04](sprint-m3-04.md) (they need the harness
  codecs); the `Caps` half of t4 §5.6 #3 and lints #10, #11, #15 → [m3-06](sprint-m3-06.md), [m3-09](sprint-m3-09.md), [p-01](sprint-p-01.md).
- `problem.spec` → [m2-01](sprint-m2-01.md). Exposing `grading_summary` on list routes (Problems markers) → [m3-09](sprint-m3-09.md) / [m3-12](sprint-m3-12.md).

## Tasks

### 1 · Canonical hash package `internal/course/canon` [X]

**Extend** the package [m1-09](sprint-m1-09.md) created with `ContentHash` (see the informational note under Entry gates if it
didn't). Stdlib only; it imports `internal/course` and nothing else, so every service can use it without cycles.

```go
package canon
func Bytes(v any) ([]byte, error)                       // canonical JSON of a typed value (canon@1)
func PolicyVersion(m *course.Manifest) (string, error)  // "sha256:<64 hex>"
func ContentHash(it *course.ResolvedItem) (string, error) // the whole public item, sidecars inlined — m1-09's function, kept
func ContractHash(it *course.Item) (string, error)      // "" for an item with no graded part (self path)
func Prefix(h string) string                            // first 12 hex chars, for humans
```

`course.ResolvedItem` stands for **the resolved item type m1-09's loader already defines in `internal/course`** (the typed item
with its sidecars inlined) under whatever name m1-09 gave it. Reuse that type; never add a second resolved type.

**Canonical rules (canon@1)** — written down in the package doc and pinned by golden vectors:

| Rule | Why |
|---|---|
| Re-marshal **typed structs** with `encoding/json`: struct fields in declaration order, map keys sorted, `SetEscapeHTML(false)`, no trailing newline | fixed order with no JCS dependency ([t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning)) |
| Every optional field in a hashed view is `omitempty`, so absent ≡ empty (`[]`, `{}`, `""`, 0) | an author's whitespace, key order, number spelling or absent-vs-empty choice never moves a hash — that is the whole "formatting" claim; any other edit is a real edit |
| Open JSON values (sample `args`/`expected`, checker `params`) are decoded with `UseNumber` and re-encoded key-sorted, with **numbers normalized**: an integer literal within int64 keeps its decimal digits (`-0` → `0`, no `+`, no leading zeros); any other number is parsed as float64 and written with `strconv.FormatFloat(f, 'g', -1, 64)`, and an integral value within int64 is written as an integer — so `1e-6` ≡ `0.000001` and `1e3` ≡ `1000`; NaN/Inf and integers outside int64 are invalid | int64 safety without float rounding; no raw author bytes pass through, so a checker `eps` respelled doesn't move `contract_hash` |
| Set-valued fields are sorted before hashing (part ids, option/field ids, probe ids, class ops, step inputs, asset/rubric refs); positional lists (param types) keep their order | order is presentation, not contract |
| **Domain separation:** `sha256("xlearn.<kind>@1\n" + bytes)`, kind ∈ {`policy`, `content`, `contract`}, encoded `sha256:<64 hex>` | hashes of different kinds never coincide; a future canon@2 is explicit |
| No `map[string]any` / `json.RawMessage` field is hashed as-is | raw bytes would leak formatting |

**Contract view** (`contract.go`): exactly what hidden data depends on ([ADR-0027 §2](../../adr/0027-content-evalpack-and-user-data-model.md#2-where-content-lives-and-how-it-ships),
[t1 §3.4](../research/t1-content-data-model.md#34-versioning-and-pinning), [t4 §5.6 #9](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start)):

| In `contract_hash` | Content-only (`content_hash` alone) |
|---|---|
| every part's `{id, type}` | prompts, statements, section bodies, labels (option/field **labels**, param **names**) |
| grader steps `{step, kind, inputs (sorted)}` | `samples[]`, `limits{time_ms, memory_mb}`, `languages[]` |
| code: `signature{mode, name, params[].type (ordered), returns, ops[]{name, params[].type, returns}}`, `harness` (`name@v`), `checker{name, params}` | `constraints[]` — packlint re-validates cases on the next pack PR (advisory, task 2 check 8). **A new classification (t1 §3.4 lists it on neither side): owner to confirm** — see below |
| choice option **ids**, blank field **ids** (sorted) | provenance, links, review stamps, `concepts[]`, `solution_facts`, metadata |
| probe `{id, type}` (sorted); asset, rubric (and later palette) `id@v` (sorted) | `revision.probes[].prompt_md`, `timer_s`, `criterion` |

**Two deliberate gaps against [t4 §5.6 #9](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start)** ("covers ids,
kinds, inputs, harness, checker, generator, rubric and palette `@v`"), both logged in the Decisions log:
- **Generator `@v` is not in `contract_hash`.** Generator specs live in the **pack** (item `pack.json` `gen[]`, [m3-02](sprint-m3-02.md)),
  not in the public item, so there is nothing public to hash. A `gen@v` bump is a pack-side change: it is recorded in `tests.lock`'s
  tool header and on every perf case line, and the private CI re-validates the item. The public contract is unaffected.
- **`constraints[]` is content-only.** Tightening a constraint can make existing hidden cases invalid, and loosening one can make
  the pack incomplete. This sprint does not move the hash for either; packlint's advisory re-check (rule 8) plus the private CI's
  generic validator ([m3-02](sprint-m3-02.md)) catch both on the next pack PR. Because this decides **which public edits invalidate
  hidden cases**, the Decisions log line is marked **"owner to confirm"**. If the owner rejects it, `constraints[]` moves to the
  contract column before any pack is stamped (no packs exist yet, so the switch costs nothing).

**Tests** (`internal/course/canon/*_test.go`):
- **Classification test:** reflect over `course.Item`'s JSON paths; every path is in `contractPaths` or `contentOnlyPaths`
  (`classify.go`). A new (additive) schema field that nobody classified fails CI with "classify the new field in canon/classify.go".
- **Golden vectors:** inputs = m1-01's original fixtures (`internal/course/testdata/valid-{self,code,probes}.json`) + a class-mode
  fixture + the DSA manifest; expected hashes in `testdata/vectors.golden.json` (`-update` regenerates; CI never updates).
- **Mutation table:** for each content-only path (a statement prompt, a sample, `limits.time_ms`, `languages`, a label, a param
  name, a constraint, a section body) `content_hash` changes and `contract_hash` doesn't; for each contract path (a part id, a
  param type, the harness version, a checker param, an option id, a probe id) both change. **Non-edits** move nothing: the same
  item re-serialized with other whitespace and key order, or with a checker `eps` spelled `1e-6` instead of `0.000001`.
- Round trip: decode → `Bytes` → decode → `Bytes` is identical; a self-path item's `ContractHash` is `""`.

m1-09 defined `ContentHash`: keep its bytes unless they break a rule above. If you must change them (e.g. to add the domain
prefix or the number normalization), update m1-09's content-hash golden in the same PR and note the one-time section rewrite at the
next seed (harmless: children are re-inserted once).

### 2 · `internal/packspec` source types + `cmd/packlint` [X]

**Naming:** `internal/packspec`, never `internal/evalpack` — task 5's `.gitignore` entry `evalpack/` matches a directory of
that name at any depth.

`internal/packspec/source.go` — the private repo's **format** ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack),
[§7.1](../research/t1-content-data-model.md#71-package-format)), strict decoding, no data:
- root `pack.json` `{format_major, version}` (mi-07's scaffold is `format_major: 0`; [m3-02](sprint-m3-02.md) moves it to `1`);
- item `courses/<slug>/items/<id>/pack.json`: `{item, accepts_contract_hashes[1..2], wrong[{file, expect, category?}], keys?, review{tests?}}`
  — `expect ∈ {WA, TLE, RE, MLE, CE, REJECTED}`; m3-02 extends the type with `gen[]` and `large_case_exception`;
- allowed item subdirectories: `tests/`, `gen/`, `validate/`, `invalid/`, `submissions/`, `keys/`, `anchors/`, `exemplars/`.

**`packlint`** (`cmd/packlint`; `--public` accepts the repo root or any content root holding `courses/`):

```
packlint check --public . --pack ../xlearn-evalpack [--item <id>]… [--since <public ref>] [--json] [--strict]   (default)
packlint hash  --public . [--item <id>]…     # id · content_hash · contract_hash (the author pastes the contract hash)
packlint fingerprint …                        # task 4
```

`check` rules (ERROR unless marked):
1. Root `pack.json` decodes; `format_major` is one this packlint supports.
2. Every item `pack.json` decodes strictly; `item` equals its directory; the public item exists and is `live`
   (`retired`/`withdrawn` → WARN "packed item is not live").
3. `accepts_contract_hashes` has 1–2 entries and **contains the live `contract_hash`** — else
   `stale contract hash for <id>: live sha256:ab12cd34ef56, pack accepts [sha256:…]` (the acceptance case).
4. Files on disk ⇔ declarations: every `wrong[].file` and key file exists; no undeclared file outside the allowed subdirectories; no dotfiles.
5. A public item with an `auto` code part and a pack claiming `review.tests` declares ≥ 2 `wrong[]` with an `expect`
   ([t1 §7.4](../research/t1-content-data-model.md#74-effort-v20-scope-inferred-bottom-up-with-ai-40) full-pack tier); WARN when unstamped.
6. `key`-graded parts are `cadence: final` ([t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) #4);
   every `key_source: pack` probe or part has a key file.
7. WARN: label edits on `key`-graded parts since `--since` (task 3 holds the blocking public check).
8. WARN (advisory, [t1 §3.4 rule 3](../research/t1-content-data-model.md#34-versioning-and-pinning)): content-only edits since
   `--since` — `limits`, `samples`, `languages`, `constraints`, prompts — "re-run `make packcheck ITEM=<id>`" (TL/TLE expectations, validator).
9. INFO: public items with an `auto` code part and no pack yet (self-graded until packed).

Exit 0 clean · 1 errors · 2 usage; `--strict` also fails on WARN. Output never includes pack payloads — only ids, paths and hashes.

**Tests:** `cmd/packlint/check_test.go` with a temp pack under `testdata/` built from m1-01's **original** fixture item (never real
pack data): stale hash → exit 1 naming both hashes; missing wrong file → 1; undeclared file → 1; clean → 0; `hash` output
matches `canon`.

### 3 · Public content gates deferred from m1-09 [X]

Extend `cmd/contentlint` and the `content` CI job ([m1-09](sprint-m1-09.md) task 4, on `main` per the entry gate); put the
rules in a pure `internal/course/lint` package (stdlib + `internal/course` only) so packlint and judge — [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start)
runs these lints "again at judge start" ([m3-05](sprint-m3-05.md)) — import the same rules instead of re-implementing them.

- **Stamp gate** ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation)): a `sections/hint/*` or editorial
  (`sections/solution/*`) file **added or changed** in the PR fails unless the item's `review.hints` / `review.editorial`
  stamp is set. v1's converted sections are **grandfathered** (unchanged files pass); `contentlint --report-unstamped` lists
  them so the owner stamps them during `ev-packs-14` (count recorded in status.md content status).
- **Label-edit flag** ([t1 §3.4 rule 4](../research/t1-content-data-model.md#34-versioning-and-pinning)): for each `key`-graded
  part, diff option/field **labels** against the PR base. A changed label with the same id fails unless the PR body carries
  `label-edit-ok: <item>/<part>/<id>` (the owner's "typo" confirmation; CI reads `github.event.pull_request.body`). A new id is
  a contract change: the hash moves and the pack must list the new hash.
- **t4 §5.6 structure lints** (public-registry half): #1 part types from the closed set, configs decode strictly; #2 step kinds
  from the closed set; #3 `inputs ⊆` part ids (the `Caps` type match waits for judge's grader registry, m3-06); #4 `key`/`ai_rubric`
  inputs are `final`, `code` inputs `iterate`; #5 ≤ 1 `ai_rubric` step; #6 ≥ 1 required step when parts exist; #7 every part feeds
  a step or has `grading: none`; #8 `public:*` key sources resolve; #12 `concepts[]` resolve to an existing `<course>:concept:<slug>`;
  #13 manifest `mistakes.prefill` signals and categories are valid; #14 every `revision.bands[*].criteria[]` key maps to a step check
  or probe criterion (items with parts only).
- Compiled starters and reference-passes-samples stay with [m3-04](sprint-m3-04.md) (they need the harness codecs).

### 4 · Pre-push fingerprint hook [X]

- **`hack/git-hooks/pre-push`** (POSIX sh, executable). Resolve the pack as `${XLEARN_EVALPACK_DIR:-<repo>/../xlearn-evalpack}`.
  If it doesn't exist → exit 0 silently (CI, other machines, clones without the sibling). Otherwise run
  `packlint fingerprint --pre-push --pack <dir>` on the hook's stdin (`<local ref> <local sha> <remote ref> <remote sha>` lines),
  using `bin/packlint` rebuilt when `cmd/packlint` or `internal/packspec` is newer, else `go run ./cmd/packlint`.
- **`Makefile`:** `install-hooks` (`git config core.hooksPath hack/git-hooks`, per clone), `uninstall-hooks`, `packlint`
  (`contentlint` already exists from m1-09).
- **`packlint fingerprint`** ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation), [§10](../research/t1-content-data-model.md#10-security-and-threat-notes)):
  - **Corpus**, read locally and never written anywhere: every `tests/edge.jsonl` value (`args`, `ctor`/`ops`, any `expected`),
    key-file values and — once [m3-02](sprint-m3-02.md) builds packs — materialized cases under the pack's `build/`; each
    whitespace-stripped canonical JSON value of ≥ `--min-len` bytes (default 24) is a payload. Anchors and exemplars become
    normalized 8-word shingles.
  - **Scan target:** added lines of the outgoing diff per pushed ref (`<remote sha>..<local sha>`; a new branch diffs from
    `git merge-base <local sha> origin/main`; a deleted ref is skipped) **and** the pushed commits' messages, normalized the
    same way. A payload substring (Aho–Corasick or a length-bucketed set) or ≥ 3 consecutive shingles of one anchor/exemplar is a hit.
  - **On a hit:** print `BLOCKED: <file>:<line> contains private eval-pack data (matches <pack-relative path>)` — **never the
    payload** — and exit 1. Point to the leak runbook in [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes)
    (treat anything already pushed as disclosed; author new cases).
  - Modes: `--pre-push` (stdin), `--diff <range>`, `--tree <dir>` (m3-02's private-CI backstop over a public checkout).
  - It never prints, logs or uploads corpus hashes or payloads; nothing is written to the public repo; public CI never runs it.
- `git push --no-verify` is documented as forbidden without the owner's explicit decision; m3-02's private-CI tree scan is the backstop.
- **Tests** (`cmd/packlint/fingerprint_test.go`, temp git repo + temp synthetic pack): a planted payload in a commit → exit 1
  naming the file, payload absent from output; a payload only in a commit message → exit 1; a payload shorter than min-len →
  pass; an exemplar paragraph pasted → exit 1; the shell hook with no pack dir → exit 0.

### 5 · Repo hygiene [X]

- **`.gitignore`:** `evalpack/` and `xlearn-evalpack/` (any depth — [t1 §3.2](../research/t1-content-data-model.md#32-public-content-layout-and-delivery)
  verified a nested clone lands in the build context).
- **`.dockerignore`:** `**/evalpack` and `**/xlearn-evalpack` (Docker patterns are not recursive without `**`).
- **AGENT.md** (Working conventions; CLAUDE.md imports it): *"**Never copy content from `../xlearn-evalpack` into this repo** —
  cases, expected outputs, keys, anchors, exemplars, generators, wrong solutions, timing data — not in code, tests, fixtures, docs,
  commit messages, PR bodies or issues. Synthetic test packs under `internal/**/testdata/` (each marked by a `SYNTHETIC.md`) are
  hand-made and unrelated. Install the hook (`make install-hooks`) on any machine with the sibling checkout."*
- **Leak lints** (extend m1-09's contentlint): the filename denylist under `curriculum/` gains `pack.json`, `tests.lock`,
  `cases.jsonl*`, `*.jsonl.zst`, `edge.jsonl`, `invalid/`, `gen-hidden*`, `instances.json`, `timing.json` (public `keys/*.json`
  alias tables stay allowed). A **repo-wide pass** over tracked files (`git ls-files`) fails on `tests.lock`, `cases.jsonl*` and `*.jsonl.zst` anywhere, **except** inside
  an `internal/**/testdata/` tree whose nearest `testdata/` directory holds a **`SYNTHETIC.md` marker** (first line exactly
  `SYNTHETIC — hand-made test data, never derived from xlearn-evalpack.`). Pack artefacts in a `testdata/` tree without the marker, or
  anywhere outside `internal/`, fail. The prefix rule covers every planned fixture tree without further edits: m3-02's
  `internal/judge/testdata/{content,packsrc,pack}`, m3-04's `internal/runner/testdata/items/*/cases.jsonl`, and m3-05's and m3-12's
  pack variants (`internal/judge/pack/testdata/`, `internal/judge/testdata/pack-broken`, …). Each of those sprints only adds the marker
  file to its `testdata/` root. The embedded-file allowlist check runs over **every** package with `//go:embed`
  (`go list -json ./...` → `EmbedFiles`), one allowlist per package, not only `./curriculum`.

### 6 · Authoring guide `docs/v2/authoring.md` [X]

Short, skimmable, for the owner and for agents drafting pack material:
1. **Two halves:** the public item vs the private pack; the public/private rule ([ADR-0027 §1](../../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule));
   `honor` vs `checked`.
2. **Formats:** the grading-relevant `item.json` fields (link m1-01's schema) and the pack layout / `pack.json` (m3-02 adds
   cases, `tests.lock`, `gen`). **Public reference files:** `curriculum/courses/<slug>/items/<id>/_code/solution.go` (and
   `solution.cpp`, `solution.py`) — whole, compilable files, the names m1-09 reserves for M3 references. They are distinct from the
   converted section fragments `_code/<stage>-<NN>.<lang>.snip`, which are never references. `packlint lock` (m3-02) computes
   expected outputs from `solution.go` only.
3. **Hashes:** which edits change `contract_hash` (task 1's table) and what to do — author the contract change with the pack
   change and list both hashes (≤ 2) while either repo ships first.
4. **Stamps:** `review.statement|hints|editorial` (public) and `review.tests` (pack); the stamp gate; grandfathered v1 sections.
5. **What AI may draft** ([t1 §7.3](../research/t1-content-data-model.md#73-where-ai-may-help)): pack material only on API /
   no-training plans; expected outputs **never** AI-written; keys owner-confirmed; statements from the brief only, never fetched
   from LeetCode ([t1 §8](../research/t1-content-data-model.md#8-content-rights-stance)).
6. **Per-item workflow:** brief → AI draft (statement, constraints, samples) → owner review **before** the owner's own attempt →
   attempt → hints/editorial stamped after → pack (edges, generators, brute, wrong solutions) → `packlint check` →
   `make packcheck` (m3-02) → stamp `review.tests` → PRs in both repos (either order).
7. **Tools:** `packlint check|hash`, the hook (what a block means, never `--no-verify`), the leak runbook pointer.
8. **Budget:** 14 pilot packs ≈ 28–41 owner hours; measure the AI speed-up on the first items ([t1 §7.4](../research/t1-content-data-model.md#74-effort-v20-scope-inferred-bottom-up-with-ai-40)).

Add a one-line pointer from `curriculum/README.md` (m1-09) to the guide.

### 7 · curriculum `contract_hash` + `grading_summary` [X]

- **Migration** `internal/curriculum/store/migrations/00004_contract_hash.sql` (the next free number after m1-09's `00002`/`00003`;
  re-take the next free one at rebase if a peer landed first): `ALTER TABLE curriculum.problem ADD COLUMN IF NOT EXISTS
  contract_hash text NOT NULL DEFAULT ''`, `ADD COLUMN IF NOT EXISTS grading_summary jsonb NOT NULL DEFAULT '{}'::jsonb`.
  Expand-only (constant defaults, [ADR-0034 §3](../../adr/0034-v2-release-labelling-gating-and-rollback.md#3-migration-and-event-compatibility-rules)); no index.
- **`grading_summary`** — derived at seed, answer-free, public: `{"mode": "self"|"auto"|"mixed", "parts": [{id, type, grading, cadence}],
  "grader_kinds": [...], "languages": [...]}`. A self-path item gets `{"mode": "self"}` and `contract_hash = ''`.
- **Seed** (`SeedAll`, one transaction): `content_hash` via `canon.ContentHash` (switch m1-09's helper if it used a local one),
  `contract_hash` via `canon.ContractHash`, `grading_summary` for every item. A contract change is also a content change, so m1-09's
  child-rewrite path already fires.
- **Queries:** `UpsertProblem` gains both columns; `GetProblem` / `GetProblemsByIDs` select them; `sqlc generate`; `sqlc diff` clean.
- **API:** `GET /problems/{id}` adds `contract_hash_prefix` (12 hex, `""` for self-path items — "humans see a prefix", t1 §3.4) and
  `grading_summary`. Additive; both answer-free and not stage-gated. If [m1-06](sprint-m1-06.md) turned the gateway's problem DTO
  into an allowlist, add the two fields there; its route-enumeration test must still pass. List and bulk routes are unchanged.
- **Tests:** store integration test on fresh PG 18 (migrate + seed → stored hashes equal `canon` on the loaded items; re-seed
  idempotent; a content-only edit moves only `content_hash`); handler test for the fields; m1-09's row snapshot still passes (add
  the two columns to its "new columns" table).

### 8 · Verify + record [X]

- `gofmt -l .` empty · `go vet ./...` · `go test -race ./...` · `sqlc diff` · web typecheck/lint/test/build · e2e (`-tags e2e`, PG 18)
  · `make contentlint` · every v1 e2e green.
- Manual hook check: `make install-hooks`; a scratch `xlearn-evalpack`-shaped dir with one **synthetic** edge case
  (`XLEARN_EVALPACK_DIR=<scratch>`); plant it in a commit on a throwaway branch and push to a **local bare remote**
  (`git init --bare <scratch>/remote.git`) → blocked, payload not printed; remove it → push passes. Never push the plant to `origin`.
- [`../status.md`](../status.md): Sprint board; M3 row 🔄 (content track started); the M3 checklist line **T25/T26** ✅ on merge;
  **content status** ("authoring tooling live: packlint, hook, hashes; grandfathered unstamped hint/editorial items: N");
  **owner events** (`ev-packs-14` can use the tooling); **Decisions log**: canon@1 rules, number normalization and domain
  separation; the contract/content field classification (limits content-only with the advisory re-check; **`constraints[]`
  content-only — owner to confirm**); generator `@v` kept out of `contract_hash` (specs live in the pack; t4 §5.6 #9 deviation); the
  `packspec` name; migration `00004`; stamp-gate grandfathering; the `label-edit-ok:` PR-body token; the `SYNTHETIC.md` marker rule
  for `internal/**/testdata/` pack artefacts; the public reference file names (`_code/solution.{go,cpp,py}`).

### 9 · Owner installs the hook [O]

About 5 minutes on the authoring machine: `git pull`, `make install-hooks`, check `git config core.hooksPath` → `hack/git-hooks`,
then the dry run from the authoring guide (a planted synthetic string pushed to a local bare remote is blocked). Recorded in the
owner-events table.

## Acceptance criteria

- [ ] Hash golden vectors stable (committed vectors + round-trip test); the classification test covers every `course.Item` field.
- [ ] A prompt edit changes `content_hash` only; a signature, harness, checker-param or option-id edit changes `contract_hash` (mutation table); reformatting or respelling a number (`1e-6` ↔ `0.000001`) moves neither.
- [ ] `packlint check` flags a stale contract hash (exit 1, naming the live and the accepted hashes).
- [ ] The hook blocks a planted hidden-case string (test + manual push to a local bare remote) without printing it; with no pack dir it is a no-op.
- [ ] The stamp gate, the label-edit flag and the t4 §5.6 structure lints run in the public `content` job.
- [ ] `.gitignore`/`.dockerignore` list both pack directory names; AGENT.md carries the never-copy rule; `docs/v2/authoring.md` exists.
- [ ] The repo-wide pack-artefact pass fails on a `tests.lock` / `cases.jsonl*` / `*.jsonl.zst` outside a `SYNTHETIC.md`-marked `internal/**/testdata/` tree and passes inside one (test).
- [ ] `problem.contract_hash` + `grading_summary` are seeded (migration `00004`) and `GET /problems/{id}` shows the prefix; `sqlc diff` clean; every v1 e2e green.

## Release

**Merge only — ships dark in the next app tag:** `v1.6.0` if merged before [m1-02](sprint-m1-02.md) cuts it, otherwise `v1.7.0`
([m1-07](sprint-m1-07.md)). The migration is expand-only with constant defaults, so the rollback floor is unchanged; the API gains two
additive fields. No infra PR, no evalpack tag. packlint, contentlint and the hook never enter an image.

## Definition of Done

CI green · merged to `main` (squash, conventional commit) · acceptance criteria met · statuses updated (this file + [`../status.md`](../status.md)) · the owner has installed the hook (task 9) · decisions logged. No new
ADR: this implements ADR-0027 as decided; changing a decided shape needs a new ADR (check peers for the next free number first).

## Risks / watch-outs

- **A hash that includes too much desyncs packs on typos.** The classification and mutation tests keep `contract_hash` to what hidden
  data depends on; constraints and limits stay content-only with packlint's advisory re-check.
- **canon drift after packs exist.** Changing canon@1 bytes invalidates every `accepts_contract_hashes`. Only via canon@2, with packs
  listing both hashes (≤ 2) during the switch; the golden vectors make an accidental change fail CI.
- **Two hash definitions** (the m1-09 overlap): m1-09 merges first and owns `ContentHash` and the resolved item type; this sprint
  extends that package and type, never a second implementation. Consumers (m2-01, m3-05, m3-08) import `canon` and never re-hash.
- **Hook false positives:** short, common inputs (`[]`, `[1,2,3]`) sit under min-len; a hit names the pack file so the owner can judge.
  **False negatives:** reformatting can evade substring matching — whitespace-stripped matching catches most, the private-CI tree scan
  is the backstop, and the hook is one of ADR-0027's four layers.
- **`.gitignore` `evalpack/` matches any directory of that name** — hence `internal/packspec`; never name a package `evalpack`.
- **Stamp gate on legacy content:** without grandfathering, the converted v1 hint/solution sections would fail CI on day one.
- **Migration numbering races** with peers (m1-09's `00002`/`00003` are already on `main`): take the next free number at rebase; CI
  fails on a duplicate goose version.
- **A fixture tree without its `SYNTHETIC.md` marker** fails the repo-wide pack-artefact pass. That is intended; the fix is the marker,
  never a wider exclusion.
- **API shape:** the two new fields must not break m1-09's one-time before/after capture or m1-06's route test — update goldens deliberately.
