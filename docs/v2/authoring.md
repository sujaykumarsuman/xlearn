# Authoring items and eval packs

For the owner and for agents drafting pack material. Tooling from [m3-01](sprints/sprint-m3-01.md); the pack
pipeline (cases, `tests.lock`, generators, `make packcheck`) arrives with [m3-02](sprints/sprint-m3-02.md).
Decisions: [ADR-0027](../adr/0027-content-evalpack-and-user-data-model.md); research:
[t1 §3.3–3.4, §7–8, §10](research/t1-content-data-model.md).

> **Never copy content from `../xlearn-evalpack` into this repo** — cases, expected outputs, keys, anchors,
> exemplars, generators, wrong solutions, timing data — not in code, tests, fixtures, docs, commit messages,
> PR bodies or issues. A slip into the public repo is permanent.

## 1. Two halves

| | Public item (this repo) | Private pack (`xlearn-evalpack`, a sibling checkout) |
|---|---|---|
| Where | `curriculum/courses/<slug>/items/<id>/` | `courses/<slug>/items/<id>/` in the private repo |
| Holds | statement, constraints, samples, signature, reference code, hints, editorial, solution facts, probes | hidden cases, keys not derivable from public text, anchors, exemplars, generators, wrong solutions |
| Read by | every service (embedded) | judge only (a read-only image volume) |

**The rule** ([ADR-0027 §1](../adr/0027-content-evalpack-and-user-data-model.md#1-the-publicprivate-rule)): if the
problem statement would say it, it is public; if it encodes an answer published nowhere else, it is private.

**Trust:** a probe keyed from a public field (`key_source: public:<field>`, e.g. DSA pattern and complexity) is
`honor`; everything judge checks against the pack is `checked`. Stats show "judge-checked %" without honor grades.

## 2. Formats

**Public `item.json`** — the frozen item schema v1 ([`curriculum/_schema/item.schema.json`](../../curriculum/_schema/item.schema.json),
Go types in `internal/course/item.go`). The grading-relevant fields: `parts[]` (`id`, `type`, `cadence`,
`grading`, `config`), `grader[]` steps (`step`, `kind`, `inputs`, `required`, `rubric`), `revision.probes[]`
(`key_source` names where a key lives, never the key), `solution_facts`, `assets`. Layout and sidecars:
[`curriculum/README.md`](../../curriculum/README.md).

**Code-part types** (`config.signature` params, `returns` and op returns; the closed registry,
[`internal/platform/harness`](../../internal/platform/harness/README.md#types-closed-t1-71), m3-04). Anything else is
refused (a new type is code plus a release):

| Type | Go | C++ | Python | JSON in cases and samples |
|---|---|---|---|---|
| `int` (32-bit range) / `int64` | `int` / `int64` | `int` / `long long` | `int` | integer |
| `float64` | `float64` | `double` | `float` | number; **outputs need a float checker** (`float_abs`/`float_rel`), never `exact` by digest |
| `bool` / `string` (UTF-8) | `bool` / `string` | `bool` / `string` | `bool` / `str` | `true`/`false` / string |
| `T[]`, `T[][]` over the above | `[]T`, `[][]T` | `vector<T>`, `vector<vector<T>>` | `List[T]` | arrays |
| `ListNode`, `ListNode[]` | `*ListNode{Val, Next}` | `ListNode*` (`val`, `next`) | `ListNode(val, next)` | `[1,2,3]` |
| `TreeNode`, `TreeNode[]` | `*TreeNode{Val, Left, Right}` | `TreeNode*` | `TreeNode(val, left, right)` | level order, `null` holes: `[1,null,2]` |
| `GraphNode` | `*Node{Val, Neighbors}` | `Node*` | `Node(val, neighbors)` | 1-indexed adjacency: `[[2],[1]]` (only nodes reachable from node 1 come back) |
| `void` (an op with no `returns`) | — | `void` | `None` | `null` |

Go calls the function (`func-json@1`) or `Constructor` and the methods (`class-ops@1`); C++ and Python call a
method of `class Solution`, or the class itself. Without a `_starter/solution.<ext>`, `harness.Starter` generates
the skeleton. `int` is 32-bit in every language (C++'s `int`); use `int64` when values can exceed it. Python's
multiplier makes its TL longer: keep an item's Σ TL × the served `tl_multiplier` under the 45 s tests cap.

**Public reference files:** `curriculum/courses/<slug>/items/<id>/_code/solution.go` (and `solution.cpp`,
`solution.py`): whole, compilable files (a `.go` must be gofmt-clean). They are **not** the converted section
fragments `_code/<stage>-<NN>.<lang>.snip`, which are never references. `packlint lock` (m3-02) computes
expected outputs from `solution.go` only.

**Private pack** (`internal/packspec`, format 1 since m3-02):

```
pack.json                                    {"format_major": 1, "version": "X.Y.Z"}
tests.lock                                   written by `packlint lock --write`; never by hand
drafts/                                      unstamped hint/editorial prose; never built
courses/<slug>/items/<id>/
  pack.json      {"item", "accepts_contract_hashes": [1..2],
                  "gen"?: [{"cmd": "gen/<name>.go", "args"?: [...], "count", "tags"?}
                          | {"spec": {"gen": "int_array@1", "params": {...}}, "count", "tags"?}],
                  "wrong": [{"file", "expect", "category"?}],
                  "keys"?: {"<part or probe id>": "keys/<file>"}, "review"?: {"tests": "YYYY-MM-DD"},
                  "large_case_exception"?: {"max_bytes": ≤ 2097152, "stamped": "YYYY-MM-DD"}}
  tests/edge.jsonl   hand-picked INPUTS, one per line: {"args": [...]} or {"ops": [...], "args": [...]}, "tags"?
                     — never an expected output (packlint computes it from the reference)
  gen/<name>.go      a correctness-case generator: `--seed <u64>` + args → ONE input line on stdout
  validate/<name>.go a custom structural invariant: the canonical input on stdin; exit 0 = valid
  invalid/<name>.jsonl inputs that must be rejected (by the codec, the constraints or a custom validator)
  submissions/brute.go  the oracle: a different algorithm, run on the small (edge + random) cases
  submissions/wrong/*   declared in pack.json `wrong[]`
  keys/ anchors/ exemplars/                  only where the answer is not public-derivable
  timing.json        written by the TL gate (provisional until m3-13)
```

`expect` ∈ `WA TLE RE MLE CE REJECTED`; `category` is a mistake category of the course. Nothing else may sit
beside `pack.json` and `timing.json`; no dotfiles. A class-mode input has the public sample shape: `ops` starts
with the class name (the constructor), and `args[0]` holds the constructor's arguments.

## 3. Hashes

`packlint hash --item <id>` prints `<id> <content_hash> <contract_hash>` (canon@1, `internal/course/canon`).

- **`content_hash`** — the whole public item, sidecars and reference files inlined. Any real edit moves it;
  whitespace, key order, number spelling (`1e-6` ≡ `0.000001`) and absent-vs-empty never do.
- **`contract_hash`** — only what hidden data depends on ([`canon/classify.go`](../../internal/course/canon/classify.go)
  is the complete list). `""` for a self-path item (no graded part, no pack-keyed probe).

| Moves `contract_hash` (and `content_hash`) | Moves `content_hash` only |
|---|---|
| a part id or type; a signature's mode, name, param **types**, return type, class ops | statements, hints, editorial, reference files |
| the harness `name@v`; the checker name or a checker param (`eps`) | samples, `limits`, `languages`, **`constraints[]`** |
| a choice's mode or option **ids**; blank/text field **ids**; a blank field's kind | option and field **labels**, param **names** |
| grader steps (`step`, `kind`, `inputs`, `rubric@v`); a canvas palette `@v` | cadence, required, weights, provenance, links, stamps, concepts |
| probe ids, types and `key_source`, and their option/field ids | probe prompts, bands, criteria, timers |
| asset `id@v` refs | solution facts and all other metadata |

**A contract change needs new hidden data anyway:** author it together with the pack change, and list **both**
hashes in `accepts_contract_hashes` (at most 2) so either repo can ship first. The worst case is a short
self-graded window for that item. A content-only edit to limits, samples, languages, constraints or prompts
keeps the pack valid but may break its expectations: `packlint check --since <ref>` warns "re-run
`make packcheck`" (TLE expectations, the validator).

### The pipeline (m3-02)

`packlint` turns the source into the built pack. Every program runs in a network-less container through
packlint's executor, never on the host. The Go programs are the reference, the brute oracle, wrong solutions,
generators and validators; they run in `golang:1.26.8` pinned by digest. C++ and Python references are
syntax-checked only, and their execution gates report `pending(harness)` until m3-04.

- **Cases.** A case is a canonical JSON line `{args | ops+args, expected, id, tags}` or, for a perf spec,
  `{expected, gen, id, params, seed, tags}`.
  - `id = sha256("xlearn.case@1\n" + canonical input)` hashes the input only, so fixing a case makes a new one.
  - Cases are ordered edge → random → perf. `perf` cases come from `spec` entries of the public generator
    registry (`internal/packspec/gen`: `int_array@1`, `string@1`, `permutation@1`, `tree@1`, `graph@1`,
    `op_sequence@1`), or from a `cmd` entry tagged `perf`.
  - A literal input is ≤ 256 KiB, or ≤ 2 MiB with a stamped `large_case_exception`.
- **Seeds** are derived, never written: the first 8 bytes of
  `sha256(item ‖ 0 ‖ cmd ‖ 0 ‖ args ‖ 0 ‖ index)`. For a spec, `cmd` is the spec's canonical JSON and `args`
  is empty.
- **Expected outputs** come from `_code/solution.go` only, run through the generated harness.
  - A perf case under the `exact` checker with a float-free output stores `{"bytes", "sha256"}` of the
    reference's fd-4 frame.
  - Every other expected output is the literal result.
- **Gates.**

  | Command | Gate |
  |---|---|
  | `lock --verify` | regenerated cases equal `tests.lock` byte for byte |
  | `validate` | the generic validator from `constraints[]`, plus `validate/*`; every `invalid/*` line is rejected |
  | `exec --gate oracle` | the reference agrees with `submissions/brute.go` on the small cases |
  | `exec --gate wrong` | every wrong solution gets exactly its declared verdict |
  | `exec --gate tl --provisional` | see the TL rules below |

  The TL gate checks that `time_ms ≥ max(3 × reference max CPU, 1000)`, that Σ TL ≤ 40 CPU-s, and that
  `memory_mb ≥ 2 × reference peak + baseline`. It also checks that every `expect: TLE` solution exceeds the
  TL on a perf case.
- **Build.** `build` writes `manifest.json`, `courses/<slug>/items/<id>/cases.jsonl.zst` and
  keys/anchors/exemplars for stamped items only. Unstamped items fall back to self.
- **Listing.** `listing` checks the image allowlist.
- **The public fixture.** `internal/judge/testdata` holds the synthetic `fixture` course (`fx-001…003`) with
  its pack source and built pack. It runs every gate in public CI (`pack-fixture`) and in the private repo's
  `selftest`.

## 4. Stamps

| Stamp | Where | Gate |
|---|---|---|
| `review.statement` | public `item.json` | set when you have reviewed the statement |
| `review.hints` | public `item.json` | CI fails an added or changed `sections/hint/*` (or `_code/hint-NN.*.snip`) without it |
| `review.editorial` | public `item.json` | CI fails an added or changed `sections/solution/*` (or `_code/solution-NN.*.snip`) without it |
| `review.tests` | pack `pack.json` | packlint: a stamped pack for an auto code part needs ≥ 2 wrong solutions |

v1's converted hint and editorial sections are **grandfathered**: unchanged files pass. `make contentlint`
counts them; `go run ./cmd/contentlint -report-unstamped` lists them — stamp them during `ev-packs-14`. Keep
unreviewed drafts local or in the private repo's `drafts/`.

**Label edits on key-graded parts and probes:** changing an option or field label under the same id fails the
public `content` job, because the pack's key sits beside the id. Either mint a new id (a contract change) or,
for a pure typo, put `label-edit-ok: <item>/<part>/<id>` in the PR body and re-run the job.

## 5. What AI may draft ([t1 §7.3](research/t1-content-data-model.md#73-where-ai-may-help))

- Pack material is drafted **only on API or no-training plans**.
- **Expected outputs: never AI-written** — they are computed from the reference and checked against the oracle.
- Keys: AI proposes, **the owner confirms**.
- Inputs, generators, validators, the Go reference, the brute oracle (a *different* algorithm) and wrong
  solutions: AI may draft; the pipeline checks them.
- Statements, constraints and samples: **from the brief only**, never fetched from LeetCode
  ([t1 §8](research/t1-content-data-model.md#8-content-rights-stance)); write original expression, record
  `provenance.authored_by`.
- Hints and editorial: AI may draft; the owner reviews teaching quality and originality after their own attempt.

## 6. Per-item workflow

1. **Brief** → AI draft of the statement, constraints and samples (public item).
2. **Owner review before the owner's own attempt** → stamp `review.statement`.
3. The owner **attempts** the item.
4. Hints and editorial drafted → owner review → stamp `review.hints` / `review.editorial`.
5. **Pack:** edge cases, generators, brute oracle, ≥ 2 wrong solutions, keys where needed.
6. `packlint hash --item <id>` → paste the contract hash into the pack's `accepts_contract_hashes`.
7. `packlint check --public . --pack ../xlearn-evalpack --item <id>` → clean.
8. `make packcheck ITEM=<id>` in `../xlearn-evalpack`: lock, validator, oracle, wrong solutions, time limits.
   Then `make lock ITEM=<id>` (rewrites `tests.lock`) once the cases are right.
9. Stamp `review.tests`; open PRs in both repos (either order).

## 7. Tools

| Command | What |
|---|---|
| `packlint check [--item <id>] [--since <ref>] [--json] [--strict]` | the nine pack rules; exit 1 on an error (or a warning with `--strict`) |
| `packlint hash [--item <id>]` | `<id> <content_hash> <contract_hash>` |
| `packlint lock (--write \| --verify) [--item <id>]` | materialize the cases (expected from the Go reference) and write or verify `tests.lock` |
| `packlint validate [--item <id>]` | the validators gate |
| `packlint exec --gate oracle\|wrong\|tl\|syntax\|all [--provisional] [--item <id>]` | the execution gates (`--provisional` for `tl`); `exec --warm-cache` builds the Go std-cache volume once |
| `packlint build --out build/ --version <v> --validated-against <sha>` | the built pack (stamped items) + its Dockerfile |
| `packlint listing <image \| archive.tar \| dir>` | the image allowlist, one layer per course, the manifest verifies |
| `make packlint` | `packlint check` over `PACK` (default `$XLEARN_EVALPACK_DIR` or `../xlearn-evalpack`) |
| `make contentlint` | the public content checks (the CI `content` job) |
| `make install-hooks` | per clone: `core.hooksPath` → `hack/git-hooks` (the pre-push fingerprint hook) |

**The pre-push hook** scans every outgoing commit's added lines and messages for pack payloads (≥ 24 bytes,
whitespace-stripped) and for anchor/exemplar paragraphs. A block prints
`BLOCKED: <file>:<line> contains private eval-pack data (matches <pack path>)` — never the data. Remove the data
from **every** outgoing commit (amend or rewrite the branch), then push again. **Never `git push --no-verify`**
without the owner's explicit decision; the private CI's tree scan is a backstop, not a substitute. Without the
sibling checkout the hook is a no-op.

**Dry run (after installing):** point `XLEARN_EVALPACK_DIR` at a scratch dir holding a synthetic
`courses/dsa/items/900/tests/edge.jsonl`, commit one of its values on a throwaway branch, and push to a local
bare remote (`git init --bare <scratch>/remote.git`): the push is blocked and the value is not printed.

**If something leaks** ([t1 §10](research/t1-content-data-model.md#10-security-and-threat-notes)): treat anything
that reached a remote as disclosed; author new cases and keys; ask GitHub Support to purge cached views and PR
refs.

## 8. Budget

The 14 pilot packs (`ev-packs-14`, Go/C++/Python references) are about **28–41 owner hours** at ~10 h/week.
Measure the AI speed-up on the first items ([t1 §7.4](research/t1-content-data-model.md#74-effort-v20-scope-inferred-bottom-up-with-ai-40):
METR found experienced developers 19% slower with AI on their own repos) and re-plan the rest from it.
