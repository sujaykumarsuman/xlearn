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

**Public reference files:** `curriculum/courses/<slug>/items/<id>/_code/solution.go` (and `solution.cpp`,
`solution.py`): whole, compilable files (a `.go` must be gofmt-clean). They are **not** the converted section
fragments `_code/<stage>-<NN>.<lang>.snip`, which are never references. `packlint lock` (m3-02) computes
expected outputs from `solution.go` only.

**Private pack** (`internal/packspec`):

```
pack.json                                    {"format_major": 0|1, "version": "X.Y.Z"}
courses/<slug>/items/<id>/
  pack.json      {"item", "accepts_contract_hashes": [1..2], "wrong": [{"file", "expect", "category"?}],
                  "keys"?: {"<part or probe id>": "keys/<file>"}, "review"?: {"tests": "YYYY-MM-DD"}}
  tests/edge.jsonl   hand-picked cases: {args | ctor+ops+args, expected, tags}
  keys/ anchors/ exemplars/                  only where the answer is not public-derivable
  gen/ validate/ invalid/ submissions/{brute.*, wrong/*}
```

`expect` ∈ `WA TLE RE MLE CE REJECTED`; `category` is a mistake category of the course. Nothing else may sit
beside `pack.json`; no dotfiles. m3-02 adds cases, `tests.lock`, `gen[]` and `format_major: 1`.

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
8. `make packcheck ITEM=<id>` (m3-02: lock, validator, oracle, wrong solutions, time limits).
9. Stamp `review.tests`; open PRs in both repos (either order).

## 7. Tools

| Command | What |
|---|---|
| `packlint check [--item <id>] [--since <ref>] [--json] [--strict]` | the nine pack rules; exit 1 on an error (or a warning with `--strict`) |
| `packlint hash [--item <id>]` | `<id> <content_hash> <contract_hash>` |
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
