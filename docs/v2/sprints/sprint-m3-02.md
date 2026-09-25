# Sprint m3-02 — Evalpack pipeline: private CI gates, generic validator, image build (E) + compose fixture pack

> **Milestone:** M3 — judge plus code grader (content track) · **Track:** content · **Order:** 14
> **Prereqs:** [mi-07](sprint-mi-07.md) (MI-9: private repo, probe, pull secret, `v0.1.0`) · [m3-01](sprint-m3-01.md) (`canon` hashes, `packspec`, `packlint check|hash|fingerprint`, the hook)
> **Unblocks:** [m3-05](sprint-m3-05.md) (the fixture pack + the `packspec` manifest reader for its loader) · [m3-07](sprint-m3-07.md) (evalpack `v1.0.0` is built by this pipeline) · [m3-04](sprint-m3-04.md) (its entry gate: the harness package, wire format and checker registry created here) · owner event `ev-packs-14` (author and stamp the 14 pilot packs) · also **creates** three shared packages that later sprints extend, never duplicate: `internal/platform/harness` (Go half; [m3-04](sprint-m3-04.md) adds C++/Python), `internal/platform/checker` ([m3-06](sprint-m3-06.md)'s `code@1` imports it) and `internal/packspec/gen` (m3-06's perf cases import it) · [m3-13](sprint-m3-13.md) (TL re-gate), [spk-02](sprint-spk-02.md) (multi-arch pack image)
> **Release action:** **merge only + an optional evalpack `v0.2.0` (below every range)** — `xlearn-evalpack` PR merged to `main`; `v0.2.0` only as the build → push → probe proof; **no `>=1.0.0` tag** (that is [m3-07](sprint-m3-07.md)) · plus an xlearn PR (packspec, packlint pipeline, shared packages, fixture, compose anchor) that ships in the next app tag with no runtime change
> **Calendar:** October, before pack authoring ramps — ≈ 2026-10-12 → 10-16 (beside the owner's spike week; no owner time needed), after mi-07 lands by Fri 2026-10-09 · then the owner's `ev-packs-14` (author and stamp the 14 pilot packs with this pipeline) continues to mid-November
> **Execute with:** [`../prompts/prompt-m3-02.md`](../prompts/prompt-m3-02.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Pack format v1 + `make packcheck` | E | ⬜ |
| 2 | `internal/packspec` pipeline + `packlint` `lock` / `validate` / `exec` / `build` / `listing` | X | ⬜ |
| 3 | Seeded perf generator registry (`gen@v`) | X | ⬜ |
| 4 | Authoring executor (`--network none`) + `internal/platform/checker` + Go half of `internal/platform/harness` | X | ⬜ |
| 5 | Private CI gates (`packcheck.yml`) incl. the fixture self-test | E | ⬜ |
| 6 | Image build: `FROM scratch`, one layer per course, `/manifest.json`, listing test, multi-arch | E | ⬜ |
| 7 | Synthetic fixture pack + public `pack-fixture` CI job + compose anchor | X | ⬜ |
| 8 | Authoring rules (pack README) | E | ⬜ |
| 9 | Verify + record | X · E | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the M3
> row, content status, the evalpack stream row, owner events). Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] MI-9 done ([mi-07](sprint-mi-07.md)): `sujaykumarsuman/xlearn-evalpack` is **private**, not a fork or template, checked out at
      `../xlearn-evalpack`; its probe is green on every push; `0.1.0` pushed; pull secrets + ImageRepository merged (no ImagePolicy)
- [ ] [m3-01](sprint-m3-01.md) merged: `internal/course/canon`, `internal/packspec` source types, `cmd/packlint check|hash|fingerprint`;
      the pre-push hook active in this clone (`git config core.hooksPath` → `hack/git-hooks`; if it's unset, run `make install-hooks`
      first: a per-clone setting the session applies itself, not an owner step)
- [ ] Parallel sessions: no open PR (in either repo) touches `cmd/packlint`, `internal/packspec`, `internal/platform/checker`,
      `internal/platform/harness`, `docker-compose.yml` or the evalpack workflows in a conflicting way (`gh pr list` in both repos,
      `git worktree list`, ListAgents); no peer session is editing `../xlearn-evalpack`

_Informational, not gates:_ in the global order this sprint runs long before [m3-04](sprint-m3-04.md) and [m3-06](sprint-m3-06.md), so it
normally **creates** `internal/platform/harness`, `internal/platform/checker` and `internal/packspec/gen`. If a peer has already created
one, extend it instead (one definition). If spk-02 asked for a multi-arch pack (its hand-off note), task 6 covers it.

### M3 hard entry checklist (rollout §5, verbatim) — context, not this sprint's gate

This checklist gates the M3 UI sprint ([m3-11](sprint-m3-11.md)) and the M3-2 tag ([m3-13](sprint-m3-13.md)). This sprint prepares the
**14 pilot packs** line (the owner authors and stamps them in `ev-packs-14` with this pipeline).

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

Make `xlearn-evalpack` a real pipeline ([ADR-0027 §2, §8](../../adr/0027-content-evalpack-and-user-data-model.md#8-authoring-and-rights),
[t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack), [§7.2](../research/t1-content-data-model.md#72-ci-validation)):
reproducible cases, **one generic validator derived from the public `constraints[]`**, oracle and wrong-solution gates, a time-limit
gate, the review-stamp gate, and the `FROM scratch` data image — plus a **synthetic fixture pack** so public CI and compose exercise the
format with no secrets. The principle: **public code, private data.** The format, case codec, lock, validator, executor, generator
registry and image builder are public Go in this repo (shared with judge, no secrets in them); only the pack's contents are private.

## Scope

**In**
- Pack format v1: item `pack.json` (+ `gen[]`, `large_case_exception`), `tests/edge.jsonl`, `gen/`, `validate/`, `invalid/`,
  `submissions/{brute.*, wrong/*}`, `tests.lock`, `timing.json`, `drafts/`; `make packcheck`.
- `internal/packspec`: cases (canonical JSONL, case ids, `cases.jsonl.zst`), lock, generic validator, built manifest, builder, listing check.
- The closed public **seeded perf generator registry** `internal/packspec/gen` ([t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)).
- The **Go half** of `internal/platform/harness` (type registry, canonical JSON, fd-3/fd-4 frames, the Go `func-json@1` /
  `class-ops@1` templates) built to [m3-04](sprint-m3-04.md)'s spec, and the closed checker registry `internal/platform/checker`.
- `packlint lock|validate|exec|build|listing`; the executor runs every author/AI program in a container with `--network none`.
- Private CI gates on changed items (t1 §7.2 table + t3 pack lints) and a self-test on the public fixture.
- Image build: `FROM scratch`, one layer per course, `/manifest.json`, `cases.jsonl.zst`, listing test, multi-arch index.
- `internal/judge/testdata/{content,packsrc,pack}` (3 synthetic items in a `fixture` course, with Go, C++ and Python references), a
  public `pack-fixture` CI job, the compose `x-evalpack-mount` anchor.
- The AI-drafting / authoring rules in the pack README.

**Out**
- Tagging evalpack `v1.0.0` and its ImagePolicy → [m3-07](sprint-m3-07.md).
- Pack content: the 14 pilot packs → owner event `ev-packs-14`. The agent writes **no** real item's pack and never stamps.
- judge's loader (hash verification at start, statuses, lazy reads, `GET /internal/evaluable`) and the compose judge service —
  including the dev-only content overlay that makes the fixture's items resolvable in compose → [m3-05](sprint-m3-05.md) (hand-off in task 7).
- Runtime generation and streaming of perf inputs, the `code` grader → [m3-06](sprint-m3-06.md) (imports `internal/packspec/gen` and
  `internal/platform/checker`).
- The C++/Python halves of `internal/platform/harness`, the cross-language golden test, in-jail diagnostics, seccomp →
  [m3-04](sprint-m3-04.md) (extends the package created here); runner TL baselines → [m3-15](sprint-m3-15.md);
  the definitive TL re-gate (baselines × prod multipliers) → [m3-13](sprint-m3-13.md).
- go-race, SQL and `ai_rubric` calibration gates → P ([p-01](sprint-p-01.md)) and later.

## Tasks

### 1 · Pack format v1 + `make packcheck` [E]

```
pack.json                      {"format_major": 1, "version": "<semver>"}       (mi-07's scaffold had 0)
tests.lock                     {"format": 1, "tools": {packlint, harness, gen, image digests}, "cases": {"<course>/<id>": {"<case_id>": "sha256:…"}}}
courses/<slug>/items/<id>/
  pack.json                    item · accepts_contract_hashes[1..2] · gen[] · wrong[] · keys? · review{tests} · large_case_exception?
  tests/edge.jsonl             hand-picked inputs {args | ctor+ops+args, tags}; NO expected (CI computes it)
  gen/<name>.go                correctness-case generators, CI-only (`--seed <u64>` + params → JSONL on stdout)
  validate/<name>.go           custom structural invariants only (BST, connected graph …); exit 0 = valid
  invalid/<name>.jsonl         inputs every validator must reject
  submissions/brute.go         the oracle: a *different* algorithm, small cases only
  submissions/wrong/<name>.go  declared in pack.json `wrong[]` with `expect`
  keys/ anchors/ exemplars/    only where the answer isn't derivable from public text
  timing.json                  written by the TL gate (provisional flag)
drafts/                        unstamped hint/editorial drafts — never built
Makefile  README.md  .github/workflows/{build.yml (mi-07), packcheck.yml}
```

Item `pack.json` (extends m3-01's type, strict):

```jsonc
{ "item": "16", "accepts_contract_hashes": ["sha256:…"],
  "gen":   [{"cmd": "gen/random.go", "args": ["--n", "50"], "count": 30, "tags": ["random"]},
            {"spec": {"gen": "int_array@1", "params": {"n": 3000, "min": -100000, "max": 100000}}, "count": 3, "tags": ["perf"]}],
  "wrong": [{"file": "submissions/wrong/no-dedupe.go", "expect": "WA", "category": "right_pattern_wrong_state"},
            {"file": "submissions/wrong/cubic.go", "expect": "TLE", "category": "complexity_misjudged"}],
  "review": {"tests": "2026-10-20"} }
```

- **Seeds** are `hash(item_id, cmd)` ([t1 §7.2](../research/t1-content-data-model.md#72-ci-validation)): the first 8 bytes of
  `sha256(item_id ‖ 0 ‖ cmd ‖ 0 ‖ args ‖ 0 ‖ index)` as a `uint64` (for a `spec` entry, its canonical JSON stands in for `cmd` and
  `args`), so regenerating is byte-for-byte reproducible.
- **`Makefile`:** `make packcheck [ITEM=<id>] [COURSE=dsa]` (every gate locally; needs docker), `make lock ITEM=…` (rewrite
  `tests.lock` after an intended change), `make build`, `make image` (local, never pushed), `make selftest` (the public fixture).
  packlint comes from the sibling public checkout (`go -C ../xlearn run ./cmd/packlint`), so local and CI run the same code.
- Replace mi-07's `hack/build.sh` with `packlint build`; keep its `version == tag` check.

### 2 · `internal/packspec` pipeline + packlint subcommands [X]

| File | Content |
|---|---|
| `cases.go` | Canonical case line `{"id", "args" \| "ctor"+"ops"+"args" \| "gen"+"params"+"seed", "expected", "tags"}`; **case id = `sha256("xlearn.case@1\n" + canonical input)`** — input only, so a fix makes a new case ([t1 §7.1](../research/t1-content-data-model.md#71-package-format)); order edge → random → `perf` last ([t4 §2.6](../research/t4-judge-contract.md#26-evaluation)); literal cap **256 KiB**, ≤ 2 MiB only with a stamped `large_case_exception` ([t4 §9.4](../research/t4-judge-contract.md#94-the-per-case-input-cap-t1-flag-resolved)); `cases.jsonl.zst` writer/reader (`github.com/klauspost/compress/zstd`, single-threaded, fixed level, checksums → deterministic bytes) |
| `lock.go` | Materialize: edge lines + generator output (run through task 4's executor) + perf specs (task 3); **`expected` computed by running the public Go reference** — the whole file `curriculum/courses/<slug>/items/<id>/_code/solution.go`, the name m1-09 reserves for M3 references and m3-01's authoring guide documents (the converted `_code/<stage>-<NN>.go.snip` section fragments are never references; no `solution.go` → `lock` fails "no public Go reference") — never AI-written ([t1 §7.3](../research/t1-content-data-model.md#73-where-ai-may-help)); perf specs store `expected: {sha256, bytes}` only for canonicalizable checkers, else a literal ([t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)); a lock entry = sha256 of the canonical line; `--verify` recomputes and diffs |
| `validate.go` | **One generic validator derived from the public `constraints[]`** — the grammar m1-01 froze (`{arg, len}`, `{arg: "nums[*]", range}`, …; an unknown kind is an error "extend the validator") — applied to every edge, random and materialized perf input; custom `validate/` programs run through the executor; every `invalid/*` line must be rejected by at least one validator |
| `manifest.go` | Built `/manifest.json` = exactly `{format_major, version, validated_against, items{id: {item_hash, accepts_contract_hashes, grader_kinds, files{path: sha256}}}}` ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack)'s shape — **no extra fields**, since judge may decode it strictly; the course is already in every `files` path `courses/<slug>/…`); `item_hash = sha256("xlearn.item@1\n" + sorted "path sha256\n" lines)`; `ReadManifest` (strict) + `VerifyFiles(fs.FS)` (**streaming** sha256 per file, never a whole file in memory — judge runs in 256 Mi) — the **pure** reader and verifier offered to [m3-05](sprint-m3-05.md)'s loader so the writer and reader share one type (its statuses and index stay m3-05's) |
| `build.go` | `build/` = `manifest.json` + `courses/<slug>/items/<id>/{cases.jsonl.zst, keys/**, anchors/**, exemplars/**}` + a generated `Dockerfile` (task 6). **Only items with `review.tests`** are built; the rest are listed in the summary and fall back to self (stamp gate) |
| `listing.go` | The image allowlist check (task 6) |

```
packlint lock      --public . --pack ../xlearn-evalpack [--item <id>]… (--write | --verify)
packlint validate  --public . --pack … [--item …]
packlint exec      --public . --pack … [--item …] --gate oracle|wrong|tl|all [--provisional]
packlint build     --public . --pack … --out build/ --version <semver> --validated-against <public sha>
packlint listing   <image ref | exported tar>
```

m3-01's `fingerprint` now also reads materialized cases under the pack's `build/`, so the hook covers generated cases too.
Unit tests per file; end-to-end tests on the fixture (task 7).

### 3 · Seeded perf generator registry [X]

[t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) decided a **closed, public Go registry** — `int_array@1`, `string@1`,
`permutation@1`, `tree@1`, `graph@1`, `op_sequence@1` (~400 LOC) — and that the pack holds specs and seeds, **never generator code**
([t4 §11.4 #7](../research/t4-judge-contract.md#114-amendments-to-settled-docs-every-flag)). It lands here, first, because `packlint lock`
needs it to materialize perf cases, and it is **the** registry: one path, `internal/packspec/gen`, with exactly t4 §4.1's six generators.
[m3-06](sprint-m3-06.md) imports it at runtime rather than creating `internal/judge/gen` (its plan names that path and a `graph_edges@1`;
`graph@1` here **is** the edge-list generator, one name). A generator the pilot items need later is a new `name@1` added to this
package, never a second registry.
- `internal/packspec/gen`: an explicit list (no `init()` registration); `Generate(w io.Writer, params, seed) error` writes the
  canonical `args` JSON **streaming** (judge streams ≤ 8 MiB per case to the runner, L14), deterministic from `(params, seed)`
  (`math/rand/v2` PCG, no global rand). `graph@1` emits an edge list (`n`, `edges[[u,v(,w)]]`, directed/weighted/connected flags);
  `tree@1` a level-order array with `null` holes (m3-04's `TreeNode` encoding).
- Params are validated against the item's public `constraints[]` (a perf spec outside them fails `validate`).
- **Golden hash per `gen@v`:** a behaviour change must bump `@v`; the lock header records the registry version.

### 4 · Authoring executor + checkers + Go harness codec [X]

- **Every author- or AI-written program** (reference, brute, wrong solutions, generators, custom validators) runs **only** in a container
  with `--network none` ([t1 §7.2–7.3](../research/t1-content-data-model.md#73-where-ai-may-help)), **one container per program**:
  `docker run --rm --network none --read-only --tmpfs /w:rw,exec,size=512m --cpus 1 --memory 1g --pids-limit 128 --user 65534:65534
  -v <src>:/src:ro -v <gocache volume>:/seed:ro <image@sha256>`. Images pinned by digest: `golang:1.26.8-bookworm` (the runner's Go,
  [t3 §6.1](../research/t3-sandbox.md#61-image-and-release)), `python:3.13-slim` and `gcc:14` for syntax checks. Our public driver compiles
  once and spawns **one process per case** (as the runner does), measuring CPU time and peak RSS per case via `rusage`.
- **Go build environment** (m3-04's compile env, adapted to docker): `HOME=/w`, `GOCACHE=/w/gocache`, `GOFLAGS=-mod=readonly`,
  `GOTOOLCHAIN=local`, `GOPROXY=off`, `CGO_ENABLED=0`, `GOTELEMETRY=off`, `GOENV=off`, `-trimpath -buildvcs=false`. The golang image
  compiles std into `GOCACHE` on the first build, so an unseeded read-only container would fail (no writable HOME) or overflow a small
  tmpfs and burn minutes per program. Instead: **one read-only std-cache volume per image digest**
  (`xlearn-packlint-gocache-<digest12>`), built once by `packlint exec --warm-cache` (compiles the Go harness template and a probe
  that imports the std packages references use), is copied into the tmpfs at container start (`cp -R /seed/. /w/gocache`, well under
  a second). The seed is mounted read-only, so no program can poison it. CI caches the volume's tarball keyed by the digest
  (`actions/cache`), so a warm run compiles only the learner-shaped files.
- **Checkers (one definition):** outputs are compared with the item's public checker from the closed registry (`exact`, `unordered`,
  `unordered_deep`, `float_abs|rel(eps)`, `set_equal`, `any_of`; typed verdicts on `harness.Value`, never formatted values).
  Create **`internal/platform/checker`** (a library: stdlib + `internal/platform/harness`, no judge imports, so packlint can use it);
  [m3-06](sprint-m3-06.md)'s `code@1` grader in `internal/judge/grader` imports it instead of defining checkers inline.
- **Harness codec (one definition):** submissions are function- or class-mode, so the executor wraps them with `func-json@1` /
  `class-ops@1` over the closed type registry ([t1 §7.1](../research/t1-content-data-model.md#71-package-format)). The package is
  **`internal/platform/harness`** — the path [m3-04](sprint-m3-04.md) owns and extends — and this sprint builds its **Go half to m3-04's
  plan (tasks 1–2) as the spec**, so nothing is rewritten later:
  - `types.go` (`ParseType` over the closed set, the Go column of the type mapping), `canon.go` (canonical JSON, `TreeNode` level order,
    `GraphNode` 1-indexed adjacency, floats never canonical, `CanonicalizableOutput`), `EncodeInput`, `DecodeOutput`, `Canonical`;
  - the **frame protocol**: `u32` big-endian length + canonical JSON, one frame in on **fd 3**, one frame out on **fd 4**
    (`{"ok": …}` · `{"panic": "<class>"}` with Go's closed class set · `{"error": "<harness_error>"}`); stdout/stderr stay the program's;
  - `templates/go/{func-json,class-ops}.tmpl` → `zz_xl_harness.go` (`package main`, node types, `recover()` → panic class), no reflection;
  - `Generate` returns the package's own `File{Name string; Data []byte}` because `internal/platform/runnerapi` ([m3-03](sprint-m3-03.md))
    doesn't exist yet; m3-04 adapts it to `runnerapi.File` without changing a byte on fd 3/fd 4;
  - golden tests: every Go-reachable type round-trips; malformed inputs are rejected with a typed error; the same level-order and
    graph edge cases m3-04 lists.

  m3-04 then adds the C++ and Python templates and preludes, the cross-language golden test and the `ProfileSHA` inputs. `harness@v`
  is part of `contract_hash`, so the wire format is fixed once — here, to m3-04's spec. The executor's driver feeds fd 3 and reads
  fd 4 (`os/exec` `ExtraFiles`), exactly as the runner will.
- **Languages before m3-04:** Go runs every gate; C++/Python references are syntax-checked only and their execution gates report
  `pending(harness)` (not a failure). [m3-13](sprint-m3-13.md) re-gates everything.
- **Gates** (`packlint exec --gate …`):
  - `oracle` — the reference agrees with `submissions/brute.*` on the small cases (checker-aware).
  - `wrong` — each `wrong[]` gets its declared verdict on the full case set (WA via the checker, TLE via the TL, RE via exit/panic, CE via compile).
  - `tl` — `limits.time_ms ≥ max(3 × reference max CPU ms × scale, 1000)` ([t3 §7.2](../research/t3-sandbox.md#72-time-limit-policy)),
    with `scale = 1.0` on the pinned image and `provisional: true` written to `timing.json` (image digest, reference max, TL floor);
    also **Σ TL over the hidden cases ≤ 40 CPU-s** and **`memory_mb ≥ 2 × reference peak + baseline`** (t3 pack lints), and every
    `expect: TLE` wrong solution exceeds the TL on at least one perf case. The provisional flag clears in m3-13 (runner-v1.0.0
    baselines from m3-15 × mi-10 prod multipliers).

### 5 · Private CI gates [E]

`.github/workflows/packcheck.yml` on `pull_request`, `push` to `main` and `workflow_dispatch`:
- Checkout the pack; checkout public `sujaykumarsuman/xlearn` at `main` or at `public-ref: <sha>` from the PR body (a public repo —
  no credentials); build `packlint` from that checkout; record the public SHA as `validated_against`.
- **Changed items only:** item directories touched vs the merge base, plus items whose public `contract_hash` is no longer accepted
  (`packlint check`). **Full run** when the recorded tool versions in the `tests.lock` header change (packlint, harness, `gen@v`,
  executor image digests), on the `full-run` label, or on a dispatch input.

| Gate | Command | Rule |
|---|---|---|
| Lint | `packlint check` | m3-01's rules (stale hash, declarations, files) |
| Reproducibility | `packlint lock --verify` | regenerated cases equal `tests.lock` (seed = `hash(item_id, cmd)`) |
| Validators | `packlint validate` | generic validator from public `constraints[]` + custom structural invariants; every `invalid/*` rejected |
| Correctness | `packlint exec --gate oracle` | the reference agrees with the brute oracle on small cases |
| Wrong solutions | `packlint exec --gate wrong` | each gets its declared verdict |
| Time limits | `packlint exec --gate tl --provisional` | TL ≥ 3 × reference max (floor 1 s) on the pinned profile; Σ TL ≤ 40 CPU-s; memory rule; provisional until m3-13 |
| Cross-repo fingerprint | `packlint fingerprint --tree $PUBLIC --pack .` | no pack payload anywhere in the public tree (the hook's backstop) |
| Review stamp | `packlint build` | unstamped items are left out of the built pack (self fallback) and listed |
| Probe | mi-07's `probe` job | anonymous GET 401/403 on every push; add a weekly `schedule` ([t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes)) — a CI check, not an alert (D34) |

- **`selftest` job:** every gate + build + listing on the **public synthetic fixture** (`--public $PUBLIC/internal/judge/testdata/content
  --pack $PUBLIC/internal/judge/testdata/packsrc`), so the pipeline is proven before any real item is stamped.
- **Job summary:** per item ✓/✗ per gate, provisional flags, excluded (unstamped) items — ids and counts only, **never payloads**.
- **Budget:** changed items only, and `make packcheck` runs the same gates locally, keeping the private repo well under 2,000 Actions min/month.

### 6 · Image build [E]

- mi-07's `build.yml` build job (on `v*` tags) runs `packlint build --version ${TAG#v} --validated-against <public sha>` → `build/`, whose
  generated Dockerfile is:

  ```Dockerfile
  FROM scratch
  LABEL org.opencontainers.image.source=https://github.com/sujaykumarsuman/xlearn-evalpack
  COPY courses/dsa/ /courses/dsa/   # one COPY = one layer per course, courses sorted
  COPY manifest.json /manifest.json
  ```
- `docker buildx build --platform linux/amd64,linux/arm64` (data only, identical layers; lets the arm64 spike VM and Apple-silicon compose
  pull it — the spk-02 hand-off), `provenance: false`, `sbom: false`, `rewrite-timestamp=true` with `SOURCE_DATE_EPOCH` = the tag commit
  time, so identical content gives an identical digest.
- **Listing test** (`packlint listing`, before the push, and on PRs against a local build): the image may contain only `/manifest.json`
  and `/courses/<slug>/items/<id>/{cases.jsonl.zst, keys/**, anchors/**, exemplars/**}`. Anything else — `gen/`, `validate/`, `invalid/`,
  `submissions/`, `tests/`, `pack.json`, `tests.lock`, `timing.json`, `drafts/`, dotfiles — fails the build ([t1 §3.3](../research/t1-content-data-model.md#33-private-eval-pack):
  no generators, oracles, wrong solutions or calibration data).
- Then mi-07's probe (existence 200, anonymous 401/403, negative control 200).
- **Optional:** tag `v0.2.0` to prove build → push → probe with format 1 (it builds only stamped items, so it may be empty — judge would be
  Ready with 0 evaluable). It sits below every `>=1.0.0 <2.0.0` range. **Never tag `>=1.0.0` here.**

### 7 · Synthetic fixture pack + `pack-fixture` CI job + compose anchor [X]

Under `internal/judge/testdata/` — **synthetic, hand-made, original; never derived from `../xlearn-evalpack`**. The tree's root carries
m3-01's marker file `internal/judge/testdata/SYNTHETIC.md` (first line exactly `SYNTHETIC — hand-made test data, never derived from
xlearn-evalpack.`, then what the fixture is for). That marker is what lets the repo-wide pack-artefact lint accept `tests.lock` and
`cases.jsonl.zst` here and nowhere unmarked.

**A separate `fixture` course, not real DSA items** (the milestone plan first said "2–3 synthetic DSA items"; this is a deliberate
deviation, logged): the v1 DSA items are self-path (no graded parts, `contract_hash = ""`), so a fixture pack for real DSA ids would need the
agent to author their public graded half (signature, samples, constraints, reference) ahead of the owner's `ev-packs-14`, against the
"owner reviews before the owner's own attempt" rule. It would also couple public CI to every future contract edit of those items and
put hidden-style cases for real items in the public repo, where they could collide with the owner's real cases in the fingerprint scan.

```
content/paths.json                       one row {slug "fixture", title "Fixture (dev only)", status "active", …} — m1-01's manifest ⇔ paths.json check
content/ids.lock.json                    {"items": {"fx-001": {"course": "fixture", "status": "live"}, …}, "assets": {}}
content/courses/fixture/course.json      minimal valid manifest, id_prefix "fx", status "active" (v1's path.status CHECK allows only
                                         active|coming_soon until p-02; the fixture is never embedded, so never seeded in prod)
content/courses/fixture/{phases,weeks,concepts}.json   minimal, so m1-09's loader and guards accept the root unchanged
content/courses/fixture/items/fx-001/…    function mode, `exact` (a boolean answer)
content/courses/fixture/items/fx-002/…    function mode, `unordered_deep` (a list of triples) — wrong/: one WA (no dedupe), one TLE (cubic)
content/courses/fixture/items/fx-003/…    class mode `class-ops@1` (a tiny key-value cache)
packsrc/…                                 the source pack for the three items (pack.json, edge.jsonl, gen/, invalid/, submissions/)
pack/                                     the built pack (manifest.json, cases.jsonl.zst) — what compose mounts
```

- No `_schema/` copy (m1-01's freeze guard watches one schema): packlint validates the fixture root against the checkout's
  `curriculum/_schema/`. `packlint check --public internal/judge/testdata/content` exits 0; that is the first `pack-fixture` step.
- Statements are one original line each. **References:** each item has `_code/solution.go`, `_code/solution.cpp` and
  `_code/solution.py` (the reserved whole-file names, m3-01's authoring guide). Expected outputs come from `solution.go` only; the
  C++ and Python references are syntax-checked here, and their execution gates report `pending(harness)` until
  [m3-04](sprint-m3-04.md) runs all three through the jail (its task 6 reuses these items). [m3-06](sprint-m3-06.md)'s synthetic
  **learner** solutions (`internal/judge/testdata/solutions/<item>/`) are its own. If m1-09's loader treats every `_code/` file as a
  section fragment, teach it here to skip the reserved `solution.*` names (with a test); converted DSA items are unaffected.
- **Two fixture tests, split by what they need:**
  - `internal/packspec/fixture_test.go` (plain `go test`, **no docker**): the committed `pack/` is internally consistent and
    deterministic. Every line of every `cases.jsonl.zst` hashes to its `tests.lock` entry, re-compressing the decoded lines gives the
    same bytes, and `manifest.json` (file hashes, `item_hash`) recomputes identically (`-update` regenerates `pack/` from a fresh
    build).
  - The **source → pack rebuild** runs the Go reference to compute `expected`, so it needs docker. It lives in the `pack-fixture` CI
    job: `lock --verify`, then `build --out $RUNNER_TEMP/pack` and `diff -r $RUNNER_TEMP/pack internal/judge/testdata/pack`.
- **Public CI** — a new `pack-fixture` job in `.github/workflows/ci.yml` (docker is available on `ubuntu-24.04`; **no secrets**):
  `packlint check`, `lock --verify`, `validate`, `exec --gate all`, `build` + the diff above, `listing` on the fixture. The planted WA/TLE
  solutions fail as declared here. The std-cache volume (task 4) is restored from `actions/cache` keyed by the golang image digest.
- **Compose** — a top-level anchor, in the style of `x-pg-env`:

  ```yaml
  # The eval pack judge mounts at /evalpack (t1 §3.7). Default: the synthetic fixture. For the real pack,
  # set EVALPACK_DIR=../xlearn-evalpack/build after `make build` there (local only; never committed).
  x-evalpack-mount: &evalpack-mount
    type: bind
    source: ${EVALPACK_DIR:-./internal/judge/testdata/pack}
    target: /evalpack
    read_only: true
  ```
  [m3-05](sprint-m3-05.md)'s judge service attaches it (`volumes: [*evalpack-mount]`); `docker compose config` stays valid.
  `deploy/local/README.md` gains an "Eval pack in compose" section.
- **Hand-off: the dev-only content overlay** (recorded in the Decisions log and in `deploy/local/README.md`). This sprint does not
  build it, because judge doesn't exist yet. Compose's services embed the real curriculum, which has no `fixture` course, so with only
  the default mount judge starts **Ready with 0 evaluable**: every pack item is `invalid` (no public item), which is t1 §3.7's "with
  no pack" behaviour. For the fixture items to be evaluable in compose (m3-05's "N evaluable" acceptance; the m3-06, m3-11 and m3-12
  compose e2e), [m3-05](sprint-m3-05.md) builds the overlay from the **contract** below and those e2e set it; its compose check
  reads `path=fixture`, not `path=dsa`.
  - `FIXTURE_CONTENT_DIR=<dir>` loads that content root **in addition to** the embedded curriculum. It is honoured only when `DEV_AUTH`
    is set (the ADR-0033 §3 guard pattern); otherwise it is ignored, logged at ERROR, and the service still starts.
  - The same env var is read by judge (m3-05), curriculum (seeds the `fixture` path, so the SPA lists it) and practice (resolves the
    items for attempts). Compose mounts `./internal/judge/testdata/content` read-only and sets it on all three.
  - Ids can't collide: `fx-` prefix, the fixture's own `ids.lock.json`, and the loader's cross-course uniqueness guard.
  - With the overlay, `GET /internal/evaluable?path=fixture` lists `fx-001…003` as `ok`.

### 8 · Authoring rules (pack README) [E]

In `xlearn-evalpack/README.md` ([t1 §7.3](../research/t1-content-data-model.md#73-where-ai-may-help), [ADR-0027 §8](../../adr/0027-content-evalpack-and-user-data-model.md#8-authoring-and-rights)).
The rules land as drafted (D40; the owner may revise them later with a content PR). The owner steps they name belong to pack
authoring (`ev-packs-14`), not to this session:
- Pack material is drafted with AI **only on API or no-training plans**.
- **Expected outputs are never AI-written:** `packlint lock` computes them from the reference and checks them against the oracle.
- Private keys: AI proposes, the **owner confirms**.
- AI may draft inputs, generators and custom validators (owner reviews the edge list), the Go reference (owner approves the idiom), the
  brute oracle (must be a different algorithm) and wrong solutions (must fail as declared).
- Every author- or AI-written program runs only through `packlint exec` (`--network none`), never directly on the host.
- Never copy pack content into the public repo, issues, PR bodies, commit messages, or chats on training plans (the hook and the CI scan enforce it).
- The repo and the package are never made public, forked or templated.
- `review.tests` is stamped by the owner only, after reviewing the edge list with every gate green.
- Contract changes are authored with the pack change; list both hashes (≤ 2) while either repo ships first.
- Unstamped prose lives in `drafts/`. Leak runbook: [t1 §10](../research/t1-content-data-model.md#10-security-and-threat-notes).

### 9 · Verify + record [X · E]

- **X:** `gofmt -l .`, `go vet ./...`, `go test -race ./...`, `sqlc diff` (no SQL here), web checks, e2e unchanged, the `pack-fixture` job
  green, `docker compose config` valid.
- **E:** `packcheck` green on the fixture self-test (and on any owner item already present); a PR that declares the wrong `expect` for a
  wrong solution fails; a planted `gen/` file in the build fails the listing test (then removed); the build job produces a multi-arch
  image if `v0.2.0` is tagged, and the probe stays green.
- [`../status.md`](../status.md): Sprint board; M3 row; **content status** ("pack pipeline live — format 1; items stamped 0/14; TLs
  provisional until m3-13; C++/Python gates pending m3-04"); **evalpack stream** row (`v0.2.0` digest if tagged); **owner events**
  (`ev-packs-14`: pipeline ready — the owner authors and stamps the 14 pilot packs with it, 28–41 owner hours, to mid-November; owner
  content, not a task here, and it doesn't gate this sprint's _Overall_ ✅); **Decisions log** (public code / private data; one path
  per shared package — `internal/platform/harness` (Go half here, m3-04 extends), `internal/platform/checker` (m3-06 imports), `internal/packspec/gen` with
  t4 §4.1's six generators (m3-06 imports; `graph@1` = the edge-list generator); the `/manifest.json` shape kept exactly to t1 §3.3;
  the separate `fixture` course instead of DSA items and the `FIXTURE_CONTENT_DIR` overlay contract handed to m3-05; the std-cache
  seed volume; the provisional TL scale; multi-arch; the `public-ref:` PR token).

## Acceptance criteria

- [ ] A sample item passes every gate (the synthetic fixture, in public CI and in the private `selftest` job); a planted wrong solution fails as declared, and a wrong `expect` fails the gate.
- [ ] `packlint lock --verify` regenerates `tests.lock` byte-for-byte; every `invalid/*` is rejected by the constraint-derived validator or a custom one.
- [ ] The image contains only allowed files (listing test), one layer per course, and a `/manifest.json` whose file hashes verify.
- [ ] Probe green on every push (and on the weekly schedule).
- [ ] `docker-compose.yml` carries the `x-evalpack-mount` anchor; public CI needs no secrets; the `pack-fixture` job rebuilds `pack/`
      from source identically and `fixture_test.go` passes without docker.
- [ ] `internal/platform/harness` (Go half, m3-04's frame protocol), `internal/platform/checker` and `internal/packspec/gen` exist once,
      at those paths, with golden tests.
- [ ] The pack README carries the authoring rules; `docs/v2/status.md` is updated.

## Release

**Merge only + an optional evalpack `v0.2.0` (below every range).** In `xlearn-evalpack`: the PR merges to `main`; `v0.2.0` is tagged
only if you want the build → push → probe proof with format 1, and skipping it changes nothing downstream. **No `>=1.0.0` tag** — the first
real pack is cut in [m3-07](sprint-m3-07.md), after which its ImagePolicy merges (image before policy). A `v0.2.0` never deploys (no
evalpack ImagePolicy exists yet, and it sits below `>=1.0.0`), so ADR-0034 §6's app-tag checklist doesn't apply — but check peers'
evalpack tags first (`git ls-remote --tags` in `../xlearn-evalpack`). In xlearn: merge only — packlint, testdata and a compose anchor,
so the next app tag carries no runtime change. No infra PR.

## Definition of Done

CI green in both repos · both PRs merged (squash, conventional commits) · acceptance criteria met · statuses updated (this file +
[`../status.md`](../status.md)) · decisions logged · local `main` synced in xlearn and `../xlearn-evalpack`. No new ADR (ADR-0027 is
Accepted); a change to a decided shape needs a new ADR (check peers for the next free number first).

## Risks / watch-outs

- **GitHub Actions minutes** (private repo): changed items only; heavy iteration locally with `make packcheck`.
- **TLs calibrated on non-production hardware:** every TL is `provisional` until m3-13 re-gates against the runner baselines × prod multipliers.
- **Duplicate definitions** of the codec, checkers or generators with m3-04/m3-06: one path each, fixed here
  (`internal/platform/harness`, `internal/platform/checker`, `internal/packspec/gen`). m3-04 and m3-06 extend or import them; where
  their plans name another path (`internal/judge/gen`, checkers inline in `internal/judge/grader`), the paths fixed here win.
- **The largest content sprint — cut line.** It absorbs the Go harness half, the checkers, the generator registry and the executor
  on top of the pipeline itself. The **xlearn PR (tasks 2, 3, 4, 7, and task 9's X half) is self-contained**: land it first. If the
  session can't also finish the evalpack half, **stop at the cut line**. Leave `../xlearn-evalpack` `main` untouched (never
  half-wired CI there). Set tasks 1, 5, 6 and 8 ⛔ "carried: evalpack half" and _Overall_ 🔄. The next session runs prompt steps
  9–13 only (they need nothing but the merged xlearn PR). `ev-packs-14` can start on the xlearn half alone: `make packcheck` is local.
- **The fixture mistaken for real data**, or real data leaking into it: synthetic only, the `SYNTHETIC.md` marker, and m3-01's
  repo-wide pack-artefact lint, which accepts artefacts only in marked `internal/**/testdata/` trees.
- **The fixture unreachable in compose** until m3-05 builds the `FIXTURE_CONTENT_DIR` overlay: compose judge is Ready with 0 evaluable,
  which is harmless; the contract is in task 7 and the Decisions log.
- **A generator change silently rewrites cases:** golden hash per `gen@v`, the tool versions in the lock header, `lock --verify`.
- **A public change breaks pack CI** (stale hash) — the intended signal. Pin a PR with `public-ref:` while both sides land.
- **`--network none` must cover generators and validators too**, not only submissions — the executor is the only way programs run.
- **Private package made public** (irreversible): don't touch visibility settings; the probe runs on every push and weekly.
