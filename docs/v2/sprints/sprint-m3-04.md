# Sprint m3-04 — Runner profiles Go/C++/Python + harness codecs + amd64 allowlists

> **Milestone:** M3 — judge plus code grader (runner track; code for rollout step **MI-12**) · **Track:** product · **Order:** 31
> **Prereqs:** [m3-03](sprint-m3-03.md) (runner core, the `Profile` interface, the seccomp builder, the `runner-it` CI job) · [m3-02](sprint-m3-02.md) (order 14: the shared harness package with the Go codec and its README wire format, and the closed checker registry) · [spk-02](sprint-spk-02.md) (the amd64 per-profile allowlists in t3 §16.2) · [m1-01](sprint-m1-01.md) (the frozen item schema: `signature`, `harness`, `languages[]`)
> **Unblocks:** [m3-15](sprint-m3-15.md) (image, acceptance suite, `runner-v1.0.0`). Also feeds [m3-06](sprint-m3-06.md) (judge generates harness files and decodes outputs with the package extended here) and, through packlint at `validated_against`, [m3-13](sprint-m3-13.md)'s evalpack re-gate (the C++/Python gates [m3-02](sprint-m3-02.md) left `pending(harness)`)
> **Release action:** **merge only** (ships in `runner-v1.0.0`, cut in [m3-15](sprint-m3-15.md)). The shared `internal/platform/harness` (created by [m3-02](sprint-m3-02.md), extended here) and `runnerapi/lint` packages are also compiled into judge from `v1.13.0`. No tag, no infra PR.
> **Calendar:** late October, right after m3-03
> **Execute with:** [`../prompts/prompt-m3-04.md`](../prompts/prompt-m3-04.md) — one prompt, one session.

## Status

_Overall:_ ⬜ Not started

| # | Task | Repo | Status |
|---|------|------|--------|
| 1 | Closed type registry + canonical JSON (extend m3-02's `internal/platform/harness`) | X | ⬜ |
| 2 | Harness codecs `func-json@1` and `class-ops@1` for Go, C++ and Python | X | ⬜ |
| 3 | Profiles `go@1.26`, `cpp@g++14`, `python@3.13` | X | ⬜ |
| 4 | Per-profile source lint (`internal/platform/runnerapi/lint`) | X | ⬜ |
| 5 | amd64 allowlists: exec and compile filters per profile | X | ⬜ |
| 6 | Synthetic items + reference and wrong-solution tests (Linux CI) | X | ⬜ |
| 7 | Provisional TL multipliers, docs, hand-offs | X | ⬜ |
| 8 | Public content CI: compiled starters + reference passes samples (m3-01's hand-off) | X | ⬜ |

> **Keep this current.** Set a task 🔄 when you start it, ✅ when its acceptance bullet passes, ⛔ if blocked (note why).
> Update the _Overall_ line accordingly, and mirror the sprint's state into [`../status.md`](../status.md) (Sprint board row, the M3 milestone row, the MI-12 "runner code" part).
> Full rules: [status protocol](README.md#status-protocol-way-of-working).

## Entry gates

- [ ] [m3-03](sprint-m3-03.md) merged: `internal/runner/profile` (the `Profile` type and registry), `internal/runner/seccomp` (filters from name lists), the jail and the `runner-it` CI job are on `main`; ADR-0030 is **Accepted**.
- [ ] [t3 §16.2](../research/t3-sandbox.md) holds the amd64 allowlists for **`go`, `cpp` and `python`** as sorted syscall-name lists with **0 unexpected SIGSYS under KILL**, plus the compile-jail syscall sets.
- [ ] The item schema is frozen ([m1-01](sprint-m1-01.md)): `internal/course` exposes `Signature{Mode, Name, Params[{Name, Type}], Returns, Ops}`, `Harness`, `Languages`.
- [ ] [m3-02](sprint-m3-02.md) merged (it runs first, order 14): **`internal/platform/harness`** exists with the Go `func-json@1` / `class-ops@1` codec, its **README wire format** and goldens, and **`internal/platform/checker`** holds the closed checker registry; its synthetic fixture pack (`internal/judge/testdata/pack`) is on `main`. **The README's wire format must be t3 §6.2's** (one `u32`-BE-framed canonical JSON frame in on fd 3, one out on fd 4). If it chose another channel (stdin/stdout, no framing), **stop and report**: `harness@1` is inside `contract_hash`, so reconciling it is an owner-visible decision, not a silent rewrite. *(If m3-02 has not merged, follow its own rule in reverse — create `internal/platform/harness` here with the README, leave `internal/platform/checker` to m3-02 and use a test-only comparator until it lands — and record the order in the decisions log.)*
- [ ] No open peer PR touches `internal/runner/profile/`, `internal/platform/harness/`, `internal/platform/checker/` or `internal/platform/runnerapi/lint/` (`gh pr list`, `git worktree list`, ListAgents).

This sprint is not gated by the [M3 hard entry checklist](../rollout-plan.md#5-m3-hard-entry-checklist) (that gates [m3-11](sprint-m3-11.md)); it produces part of MI-12's code.

## Goal

Make the runner grade the **three launch languages (D20: Go, C++, Python)**:
- **versioned harness codecs** `func-json@1` and `class-ops@1` over a **closed type registry**, generated per item signature, with the learner's result travelling on a separate channel (fd 4) so there is no presentation error — **one package, `internal/platform/harness`**: m3-02 created it with the Go codec, this sprint extends it to C++ and Python and keeps its `@1` bytes;
- **three runner profiles** with pinned toolchains, a deterministic environment, a compile step, per-language limits and **provisional** time-limit multipliers (calibrated on production in [mi-10](sprint-mi-10.md));
- **per-profile amd64 seccomp allowlists** taken from the spike's P3 output ([spk-02](sprint-spk-02.md)), with a runtime SIGSYS classified as RE (counted);
- proof: **reference solutions pass in all three languages** and **wrong solutions are classified** (WA/TLE/RE/CE, plus MLE and REJECTED) through the real jail.

## Scope

**In**
- **Extending** `internal/platform/harness` (m3-02's package; the one path, also used by [p-01](sprint-p-01.md)'s `harness/gotest/`): the closed type registry, canonical JSON, the frame protocol, per-language generators and preludes, starters, input encoding and output decoding. Shared: packlint (m3-02) and judge ([m3-06](sprint-m3-06.md)) import it, so it stays a leaf library (stdlib, `internal/course`, `internal/platform/runnerapi` only; a guard test forbids any other `internal/runner/...` or go-sandbox import).
- Profiles `go@1.26`, `cpp@g++14`, `python@3.13` in `internal/runner/profile/{go,cpp,python}/`: toolchain paths, compile and exec specs, env, limits, rlimits, diagnostics parsing, panic classes, the Go compile-cache seed recipe.
- `internal/platform/runnerapi/lint`: per-profile file-type, directive and import rules (judge runs them before enqueue; the runner's front re-checks names and types).
- Exec and compile seccomp filters per profile (amd64 authoritative; an arm64 dev-VM delta, build-tagged).
- Synthetic items with references and wrong solutions in Go, C++ and Python; integration tests through the jail; the `runner-it` CI job gains `g++` and `python3`.
- Provisional TL multipliers and per-profile memory baselines, served by `GET /v1/profiles`.
- The public `content` CI checks m3-01 handed over: compiled starters and "the reference passes the samples" for Go, C++ and Python (task 8).

**Out**
- `deploy/runner.Dockerfile` (the toolchain pins, the image, reproducibility), `runner-release.yml`, the acceptance suite, the compose service, TL baselines doc, the tag → [m3-15](sprint-m3-15.md).
- `go-race` and `gotest@1` (goroutine dump, RACE/DEADLOCK/LEAK) → `runner-v1.1.0`, [p-01](sprint-p-01.md). `sql-pg` / `sql-result@1` → not in the pilot plan.
- The checker registry (`exact`, `unordered`, `unordered_deep`, `float_abs|rel`, `set_equal`, `any_of`) is m3-02's `internal/platform/checker`; tests here **use it** (no second comparator). The term → class mapping → [m3-06](sprint-m3-06.md).
- Calibrating the multipliers on production → [mi-10](sprint-mi-10.md). Re-gating pack TLs → [m3-13](sprint-m3-13.md).
- In-place (`void` + mutated argument) signatures: not expressible in the frozen schema; a later `func-json@2` if an item needs one.

## Tasks

### 1 · Closed type registry + canonical JSON (extend `internal/platform/harness`) [X]

Sources: [t1 §7.1](../research/t1-content-data-model.md#71-package-format) (format rules: the closed type registry), [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (harnesses are public and versioned; result channel separate from stdout), [t3 §6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them) ("frames in on fd 3, canonical JSON out on fd 4; no reflection"), [m3-02](sprint-m3-02.md) task 4 (the package, its README and the Go codec).

- **Start from m3-02's package, don't fork it.** Read `internal/platform/harness/README.md` and its goldens first; reuse whatever already exists (`ParseType`, canonical JSON, the Go generator, node helpers) and add only what is missing. **m3-02's README is the `@1` spec**: where a rule below differs from bytes m3-02 already fixed, the README wins, the rule below is corrected to match, and the difference is recorded as a `@2` candidate in the decisions log. packlint already gates packs with these bytes (`tests.lock`, `contract_hash`), so two formats both labelled `@1` must never exist.
- **Leaf-library guard:** `internal/platform/harness/...` imports only the stdlib, `internal/course` and `internal/platform/runnerapi`; extend m3-03's `internal/runner/imports_test.go` so judge and packlint importing it can never pull in the jail, cgroups or go-sandbox.
- `types.go` — `ParseType(string) (Type, error)` over the **closed** set: `int` (32-bit range), `int64`, `float64`, `bool`, `string` (UTF-8), `T[]` and `T[][]` over the scalars and `string`, `ListNode`, `ListNode[]`, `TreeNode` (level order), `TreeNode[]`, `GraphNode`, plus `void` as a class-op return. Anything else is an error (a new type is code plus a release, t1 §7.1). Per-language mapping table:

  | Type | Go | C++ | Python |
  |---|---|---|---|
  | `int` / `int64` | `int` / `int64` | `int` / `long long` | `int` |
  | `float64` / `bool` / `string` | `float64` / `bool` / `string` | `double` / `bool` / `std::string` | `float` / `bool` / `str` |
  | `T[]` / `T[][]` | `[]T` / `[][]T` | `std::vector<T>` / `std::vector<std::vector<T>>` | `list` |
  | `ListNode` | `*ListNode{Val int; Next *ListNode}` | `ListNode*` (`int val; ListNode* next`) | `ListNode(val, next)` |
  | `TreeNode` | `*TreeNode{Val int; Left, Right *TreeNode}` | `TreeNode*` | `TreeNode(val, left, right)` |
  | `GraphNode` | `*Node{Val int; Neighbors []*Node}` | `Node*` | `Node(val, neighbors)` |

  The node shapes follow the conventions learners know from LeetCode-style statements; the preludes (task 2) define them, so learners never do.
- `canon.go` — **canonical JSON**: no whitespace; integers in decimal; strings with only `"`, `\` and controls < 0x20 escaped (`\u00xx`, lowercase hex), everything else raw UTF-8; `ListNode` → array; `TreeNode` → LeetCode level order with `null` holes and trailing `null`s trimmed; `GraphNode` → 1-indexed adjacency lists. **Floats are never canonical**: any signature whose output contains `float64` must use `OutputMode: bytes` (judge compares with a float checker), and `CanonicalizableOutput(sig)` says so.
- `EncodeInput(sig, raw json.RawMessage) ([]byte, error)`: strict decode of a case's `args` (or `ctor`+`ops`+`args`) against the signature's types (ranges, UTF-8, depth ≤ 64, node count ≤ 10⁶), re-encoded canonically as the fd-3 frame.
- `DecodeOutput(sig, frame []byte) (Value, error)`: the fd-4 frame back to a typed `Value` for judge's checkers, plus `Canonical(Value)` for `OutputMode: sha256`.
- Golden tests: every type round-trips; malformed inputs are rejected with a typed error; level-order edge cases (`[]`, `[1,null,2]`, a right-only chain); graph with a self-loop and an isolated node. **Plus a compatibility golden:** m3-02's existing Go fixtures (its goldens and `internal/judge/testdata/pack` items) encode and decode to the **same bytes** before and after this sprint.

### 2 · Harness codecs `func-json@1` and `class-ops@1` [X]

Sources: t3 §6.2 "Harnesses", [t4 §2.6](../research/t4-judge-contract.md#26-evaluation) (learner-file diagnostics only; a compile error in a non-learner file becomes judge's fixed CE text), [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide) (verdict mapping).

- **Frame protocol** (both harnesses, all languages): `u32` big-endian length + canonical JSON. **One process per case** (t3 §6.2 "exec unit"): the harness reads exactly one frame from fd 3 and writes exactly one frame to fd 4:
  - `{"ok": <value>}` — the result;
  - `{"panic": "<class>"}` — a caught runtime error from a closed per-language class set (Go: `index_out_of_range`, `nil_dereference`, `divide_by_zero`, `slice_bounds`, `nil_map_write`, `other`; C++: `out_of_range`, `bad_alloc`, `length_error`, `logic_error`, `runtime_error`, `other`; Python: `IndexError`, `KeyError`, `ValueError`, `TypeError`, `ZeroDivisionError`, `RecursionError`, `AttributeError`, `AssertionError`, `other`) → `CaseResult.PanicClass`;
  - `{"error": "<harness_error>"}` — the learner's return value can't be encoded (a `ListNode` cycle, > 10⁶ nodes, invalid UTF-8, NaN/Inf where not allowed) → judge treats it as WA.
  - Nothing else is ever written to fd 4; stdout/stderr stay the learner's (Run-mode only).
- **`func-json@1`**: input `{"args":[…]}`; calls the signature's function (`mode: function`) or the method of a `Solution` class (C++/Python convention); output its return value.
- **`class-ops@1`**: input `{"ops":["MinStack","push","getMin",…],"args":[[],[-2],[],…]}` (constructor first); output the array of per-op results, `null` for `void`. Method names, parameter and return types come from `signature.ops`.
- **Generators** `Generate(lang, harness, sig) ([]runnerapi.File, error)` from `text/template` sources in `internal/platform/harness/templates/{go,cpp,python}/{func-json,class-ops}.tmpl`, generated per signature (**no reflection**). m3-02's Go generator moves into this template shape only if its output bytes are unchanged (golden); otherwise keep it as is and add C++/Python beside it:

  | | Learner file | Harness files | Notes |
  |---|---|---|---|
  | Go | `solution.go` (`package main`) | `zz_xl_harness.go` (`package main`, `func main`, node types, a tiny JSON reader/writer, `recover()` → panic class) | built together as one ad-hoc package |
  | C++ | `solution.cpp` | `zz_xl_harness.cpp` (includes `xl_prelude.hpp`, then `#include "solution.cpp"`, then `main`) + `xl_prelude.hpp` (`<bits/stdc++.h>`, `using namespace std;`, node structs, the JSON reader/writer) | diagnostics keep `solution.cpp:line` positions |
  | Python | `solution.py` | `__main__.py` (the harness; imports `solution` after injecting the prelude's node classes into its namespace) + `xl_prelude.py` | packaged as a zipapp by the compile step |

- A learner file missing the function or with the wrong signature fails to compile **in the harness file**; the Result keeps that file name, and judge replaces it with the fixed CE text ([m3-06](sprint-m3-06.md)). Test that shape here.
- **Starters:** `Starter(lang, harness, sig) runnerapi.File` — the learner-file skeleton (the signature with a zero-value body; for `class-ops@1` the class with empty methods), used when an item ships no `_starter/` (t1 §7.1). It must compile with the harness and fail the samples (task 8 checks it).
- **Cross-language golden test:** for every non-float fixture, the Go, C++ and Python references produce **byte-identical** fd-4 frames, equal to m3-02's Go bytes.
- Templates and preludes are **public**: they ship in the repo and the image, and the harness version (`func-json@1`) is part of `contract_hash` ([m3-01](sprint-m3-01.md)). Any change that alters bytes on fd 3/fd 4 is `@2`.

### 3 · Profiles `go@1.26`, `cpp@g++14`, `python@3.13` [X]

Sources: [t3 §6.1–§6.2](../research/t3-sandbox.md#62-profile-table-limits-are-proposed-a8-tunes-them) (the `go@1.26` column is authoritative; D20 in [t3 §12](../research/t3-sandbox.md#12-owner-decisions--resolved-2026-09-24-they-override-the-body-where-they-conflict) adds `cpp` and `python`), [ADR-0030 §2](../../adr/0030-runner-technology-and-host-hardening.md#2-languages-at-m3-owner).

Each profile implements m3-03's `Profile` (`internal/runner/profile/{go,cpp,python}/profile.go`). Toolchain paths are profile config (image paths by default, overridable in tests with `RUNNER_TOOLCHAINS_DIR`); the **version pins live in [m3-15](sprint-m3-15.md)'s Dockerfile**, and the profile reports the versions it detects at startup (`go version`, `g++ -dumpfullversion`, `python3 -VV`) in `Versions.Toolchain`.

| | `go@1.26` | `cpp@g++14` | `python@3.13` |
|---|---|---|---|
| Toolchain (target) | go1.26.8 from the checksummed tarball, `/opt/xl/go` | g++ 14 from Debian trixie | CPython 3.13 from Debian trixie |
| Compile | `go build -json -trimpath -buildvcs=false -vet=off -o /job/out/bin solution.go zz_xl_harness.go`; env `CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=off GOFLAGS=-mod=readonly GOTELEMETRY=off GOENV=off HOME=/w`; `GOCACHE` = an overlay (read-only seed lower + tmpfs upper) | `g++ -std=gnu++20 -O2 -static -pipe -fdiagnostics-format=json -fmax-errors=20 -ftemplate-depth=512 -fconstexpr-depth=512 -o /job/out/bin zz_xl_harness.cpp` (+ the prelude PCH if m3-15 ships one) | syntax check `compile()` of every source (SyntaxError → CE), then a deterministic zipapp `/job/out/app.pyz` (`__main__.py`, `solution.py`, `xl_prelude.py`; fixed zip timestamps) |
| Compile limits | 15 s, 768 MiB, pids 256 | 15 s, 1 GiB, pids 64 (cc1plus, as, ld) — **compile-bomb limits**; artifact ≤ 64 MiB | 15 s, 256 MiB, pids 16 |
| Exec | `/job/bin` | `/job/bin` | `python3 -I -S -B /job/app.pyz` |
| Exec env | `GOMAXPROCS=1`, **`GOMEMLIMIT` unset**, `TZ=UTC`, `LANG=C.UTF-8` | `TZ=UTC`, `LANG=C.UTF-8` | `PYTHONHASHSEED=0`, `PYTHONDONTWRITEBYTECODE=1`, `PYTHONIOENCODING=utf-8`, `TZ=UTC`, `LANG=C.UTF-8`; the harness raises `sys.setrecursionlimit` |
| Case limits | CPU TL (≥ 1 s) from judge; wall/idle per m3-03; mem `mem_mb` (default 256) + baseline; pids 32; `RLIMIT_FSIZE` 1 MiB; `NOFILE` 64; `/w` 64 MiB / 4k inodes; no `/proc` | same; **pids 4**; `RLIMIT_STACK` = the memory limit (deep recursion) | same; pids 4; `RLIMIT_STACK` = the memory limit |
| Diagnostics | `go build -json` events → `{File, Line, Col, Msg}` | GCC JSON diagnostics → the same | `SyntaxError` `lineno`/`offset` → the same |
| Seccomp | task 5 | task 5 | task 5 (the widest) |
| TL multiplier | 1.0 (the reference language) | provisional (task 7) | provisional (task 7) |

- **Go compile cache:** `profile/go/seed.go` exposes the seed recipe (`go build std` with the profile's exact flags into a `GOCACHE` whose files get a fixed future mtime, and a fresh `trim.txt` per job so nothing is copied up or trimmed, t3 §6.1). **Add the subcommand `runner seed-gocache`** to `cmd/runner/main.go` (beside m3-03's `runner`, `runner front`, `runner canary`, `-version`): flags `-out <dir>` (required, must be empty or absent) and `-goroot <dir>` (default the profile's toolchain path); it runs the recipe with the profile's exact flags and env, sets the fixed future mtime, prints the seed's tree hash on stdout, and exits non-zero on any failure, leaving no partial seed. [m3-15](sprint-m3-15.md)'s Dockerfile calls it in the image build; the runner hashes the seed into `ProfileSHA` at startup. A `runner_it` test seeds a cache, then compiles an item against the overlay and asserts no `std` package is rebuilt.
- **Artifact modes** (m3-03's `Profile.ArtifactMode`): `0111` for the Go and C++ `/job/bin` (static ELF), `0444` for Python's `/job/app.pyz` (the interpreter reads it). One `runner_it` test per language runs the artifact as the job UID.
- **Memory baselines**: m3-03's startup step measures each profile's no-op program (`memory.peak` median); `PeakKB` subtracts it and `mem_mb` is enforced on top of it. Record the three baselines in the PR.
- **Language ids:** the item's `languages[]` values `go | cpp | python` map to these profile ids in one table (`profile.ForLanguage`), which `GET /v1/profiles` serves.

### 4 · Per-profile source lint (`internal/platform/runnerapi/lint`) [X]

Sources: [t3 §2.4](../research/t3-sandbox.md#24-attacks-and-the-control-that-stops-each-one) A5 and A18, [t4 §5.6](../research/t4-judge-contract.md#56-ci-lints-public-ci-and-again-at-judge-start) #15. **Only rejections decided before any learner code runs are uncounted (REJECTED)**; the lint is attack-surface reduction, **not** the security boundary (the jail and seccomp are).

- **Go** (t3 A5 verbatim): only judge-named `*.go` files; reject `.s .S .c .h .cc .syso .swig .swigcxx .m .f`, `go.mod`, `go.work`, `vendor/`, `import "C"`, `//go:embed`, `//line`, `/*line*/`, `//export`, and every `//go:` directive except `//go:build`. Import allowlist (t3 §6.2): `fmt sort slices maps strings strconv math math/bits math/rand/v2 container/* unicode/* bytes errors cmp iter`; deny `os/* syscall unsafe net/* plugin embed runtime/* reflect C`. Parse with `go/parser` (imports and comments only).
- **C++:** only `solution.cpp`; `#include <…>` limited to the standard-library headers (plus `bits/stdc++.h`); reject `#include "…"`, `#include_next`, `#embed`, `asm` / `__asm__` / `__asm`, `#pragma` other than `GCC optimize|target`; reject a learner-defined `main`.
- **Python:** only `solution.py`; import allowlist `collections heapq bisect itertools functools math cmath typing dataclasses string re operator copy random fractions decimal statistics array enum abc numbers sys`; reject anything else found by an `ast` walk run in Go over a small tokenizer (or in the compile jail — decide and record). `__import__` / `importlib` evasion is expected and harmless: the jail has no network, no shell and a KILL-default filter.
- API: `lint.Check(profile, files) []Violation` (judge: pre-enqueue → REJECTED with a generic message); `lint.CheckNames(profile, files)` (the front's re-check of names and types; a violation there is `400`, never a verdict).
- Golden fixtures per rule; one "evasion" fixture per language that lint misses but the jail stops (asserted in task 6).

### 5 · amd64 allowlists: exec and compile filters per profile [X]

Sources: [t3 §16.2](../research/t3-sandbox.md) (spk-02: per-profile sorted lists, justified additions, compile sets), [t3 §2.4](../research/t3-sandbox.md#24-attacks-and-the-control-that-stops-each-one) A1.

- `internal/runner/profile/<p>/seccomp_amd64.go`: the **exec allowlist**, KILL by default, copied from §16.2 verbatim (sorted, unique, one comment per justified addition); `clone3` → ENOSYS; no socket family at all; `execve` allowed only as the jail's own first exec (whichever mechanism §16.2 validated).
- The **dangerous set** (one list in `internal/runner/seccomp`, t3 §2.4 A1): io_uring_*, bpf, perf_event_open, userfaultfd, keyctl/add_key/request_key, ptrace, process_vm_*, the mount family incl. fsopen/fsmount/move_mount/open_tree, unshare, setns, namespace flags on clone, socket, **`splice`, `vmsplice` and `tee`** (A1 denies splice/vmsplice; `tee` is the same pipe-buffer family).
- The **compile filter** per profile: ENOSYS by default (logged), **KILL** for the dangerous set; `fork`/`execve` allowed (compilers spawn helpers). If spk-02's compile sets (t3 §16.2) show a toolchain calling `splice`/`vmsplice`/`tee`, return ENOSYS for that call in that compile filter instead of KILL (Go and glibc fall back to read/write) and record it; it still never enters an exec list.
- **Invariant tests** (all platforms): every name resolves on amd64; lists are sorted and unique; the dangerous set (including `splice`, `vmsplice`, `tee`) appears in **no** exec allowlist; Python's list ⊇ Go's is **not** assumed.
- **Runtime SIGSYS → `Term=signal(SIGSYS)`**, which judge maps to RE (counted, t3 A18), and m3-03 rotates the front; add one probe per language to prove it.
- **arm64 (dev VM only):** `seccomp_arm64.go` (`//go:build arm64`) = the same names resolved for arm64 plus a delta **found** in LOG mode on a dev VM, documented "never in a release image" (the release image is amd64, m3-15). **The arm64 exec filters are KILL-default like amd64**: LOG was only how the delta was found, and this sprint retires m3-03's dev-only LOG switch for every profile that has an arm64 list. m3-15's arm64 VM rehearsal runs them in `prod` mode and expects SIGSYS on every syscall probe. A guard test fails if an amd64 build links the arm64 file.
- If C++ or Python references in task 6 hit a SIGSYS the spike didn't see (its corpus was Go-heavy), add the syscall **only if it's safe**, with a justification comment, and record it in the PR and the decisions log; never add anything from the dangerous set.

### 6 · Synthetic items + reference and wrong-solution tests (Linux CI) [X]

- Items under `internal/runner/testdata/items/<slug>/` — `item.json` (frozen-schema shape, public samples), `cases.jsonl` (synthetic hidden cases; expected outputs computed by the Go reference), `refs/solution.{go,cpp,py}`, `wrong/<class>.{go,cpp,py}`. **All synthetic; never copy anything from `../xlearn-evalpack`.** Coverage:

  | Item | Harness | Types exercised |
  |---|---|---|
  | `pair-sum` | func-json@1 | `int[]`, `int` → `int[]` |
  | `group-words` | func-json@1 | `string[]` → `string[][]` (unordered) |
  | `reverse-list` | func-json@1 | `ListNode` → `ListNode` |
  | `level-order` | func-json@1 | `TreeNode` → `int[][]` |
  | `clone-graph` | func-json@1 | `GraphNode` → `GraphNode` |
  | `running-median` | func-json@1 | `int[]` → `float64[]` (bytes mode) |
  | `min-stack` | class-ops@1 | ctor + `push/pop/top/getMin`, `void` returns |

  Reuse m3-02's fixture items where they cover a row.
- `internal/runner/profile/refs_it_test.go` (`//go:build linux && runner_it`), through the real jail, for each item × language:
  - the **reference passes** every sample and every synthetic hidden case;
  - **wrong solutions are classified**: WA (wrong value), TLE (quadratic on a perf case, and an infinite loop), RE (index out of range / null dereference / uncaught exception, and a runtime SIGSYS), CE (syntax error; and a missing function → the harness-file diagnostic shape), MLE (a 1 GiB allocation), REJECTED (a lint violation);
  - the runner terms are asserted directly; outputs are compared with **m3-02's `internal/platform/checker`** using each item's declared checker (`group-words` → `unordered_deep`, `running-median` → `float_abs`, the rest `exact`); a **test-only** mapper turns terms into WA/TLE/RE/CE per [t4 §4.1](../research/t4-judge-contract.md#41-archetype-a-code-ide)'s verdict table (m3-06 owns the production mapper and reuses these fixtures);
  - compile diagnostics point at `solution.<ext>` lines;
  - the lint-evasion fixtures run and are stopped by the jail (no network, SIGSYS or ENOENT), never by luck.
- CI: extend m3-03's `runner-it` job — inside the same `debian:trixie-slim` container, `apt-get install g++ python3` from trixie (the image base) next to the pinned Go tarball. Versions may differ by a patch from the image; this lane is functional, m3-15's in-image run is authoritative.

### 7 · Provisional TL multipliers, docs, hand-offs [X]

Sources: [ADR-0030 §2–§3](../../adr/0030-runner-technology-and-host-hardening.md#3-timing-on-a-noisy-4-vcpu-vm) (TL = max(3 × the reference on the production profile, 1 s); a Go reference × a per-language multiplier), [t3 §7.3](../research/t3-sandbox.md#73-calibration).

- **Provisional multipliers:** from the `runner-it` runs, `ratio = CPU(lang reference) / CPU(Go reference)` on each item's perf cases; the multiplier is the max ratio rounded up to the next 0.5, floor 1.0. Store it in the profile as `tl_multiplier` with `calibrated: false`; `GET /v1/profiles` serves both. **The runner enforces only the CPU TL judge sends**; judge multiplies the pack's Go-based TL by the served multiplier (hand-off to [m3-06](sprint-m3-06.md)). [mi-10](sprint-mi-10.md) calibrates on production; a changed value rides the next runner patch.
- **`ProfileSHA`** (with m3-03's startup hashing): toolchain versions + toolchain tree hash + flags + env + harness templates and preludes + seccomp lists + the Go cache seed hash. A one-byte template change must change it (test).
- **Docs:** extend `docs/architecture/runner.md` (m3-03) with "Profiles" (the task 3 table), "Harness protocol" (frames, `ok`/`panic`/`error`) and "Type registry" (the task 1 table). Add the type registry table to the author guide `docs/v2/authoring.md` ([m3-01](sprint-m3-01.md)) so pack authors pick valid types.
- **Hand-offs** (in `docs/v2/status.md`): [m3-06](sprint-m3-06.md) (`harness.Generate`/`EncodeInput`/`DecodeOutput`/`Starter`, `CanonicalizableOutput`, the lint API, `tl_multiplier`, the term→class mapping fixtures, the fixed CE text for harness-file diagnostics; `harness.Starter` also feeds whichever service serves an item's `templates` to [m3-11](sprint-m3-11.md)'s IDE); **[m3-13](sprint-m3-13.md) task 4** — m3-01 and m3-02 ran before this sprint, so the receiver is the E re-gate: packlint built from `main` at `validated_against` now executes the C++/Python gates m3-02 left `pending(harness)`, and should run `lint.Check` over every reference and wrong solution so a stamped reference is never REJECTED at judge; [m3-15](sprint-m3-15.md) (toolchain targets, `runner seed-gocache`, the C++ PCH decision, Python stdlib `.pyc` handling, and re-running task 8's content check inside the image).

### 8 · Public content CI: compiled starters + reference passes samples [X]

Sources: [t1 §7.2](../research/t1-content-data-model.md#72-ci-validation) ("starters are generated and compiled"; "the reference passes the samples in the public production runner image, with `--network none`"), handed here by [m3-01](sprint-m3-01.md) (which got it from [m1-09](sprint-m1-09.md)) because it needs the harness codecs.

- `internal/platform/harness/contentcheck_it_test.go` (`//go:build linux && runner_it`), run in the `runner-it` lane (path filter gains `curriculum/**`). It walks every code part of every `curriculum/courses/*/items/*/item.json` (public data only; never `../xlearn-evalpack`) and, for each language in the part's `languages[]`:
  - **starter compiles:** the item's `_starter/` file, else `harness.Starter`, compiles with the generated harness through the real profile (CE fails the check) and does **not** pass the samples (a sanity check that the starter isn't a solution);
  - **reference passes the samples:** every public solution-stage reference `_code/solution.<ext>` present passes every public sample through the jail (empty netns, the `--network none` equivalent), compared with the item's checker via `internal/platform/checker`.
- Items without code parts are skipped; with no code item on `main` yet the check passes vacuously and says so. `gotest@1` modules are [p-01](sprint-p-01.md)'s.
- The run "in the public production runner image" is [m3-15](sprint-m3-15.md)'s in-image lane, which re-runs this check (section G).

## Acceptance criteria

- [ ] **Reference solutions pass** samples and synthetic hidden cases **in all 3 languages** for every item, through the real jail in CI.
- [ ] **Wrong solutions are classified** WA / TLE / RE / CE (plus MLE and REJECTED) in all 3 languages; a runtime SIGSYS is `signal(SIGSYS)` → RE.
- [ ] `func-json@1` and `class-ops@1` round-trip every registry type; non-float outputs are byte-identical across Go, C++ and Python; malformed inputs and unencodable outputs give typed errors.
- [ ] Every profile's exec allowlist equals t3 §16.2 plus justified, recorded additions; the dangerous set is in no exec list; the arm64 delta never links into an amd64 build.
- [ ] Lint golden fixtures pass; each lint-evasion fixture is stopped by the jail.
- [ ] `GET /v1/profiles` serves the three profiles with toolchain versions, `ProfileSHA`, memory baselines and provisional multipliers (`calibrated: false`).
- [ ] **One harness package:** `internal/platform/harness` is extended, not duplicated; m3-02's Go fixtures give the same fd-3/fd-4 bytes before and after; the leaf-library import guard passes; tests compare through `internal/platform/checker`.
- [ ] Public content CI (task 8): every code item's starter compiles in each listed language, and every public `_code` reference passes its samples through the jail.
- [ ] `runner seed-gocache` builds a seed that a compile uses with no `std` rebuild; each profile's artifact runs with its `ArtifactMode`.
- [ ] CI green (macOS-safe lanes and `runner-it`); `sqlc diff` unchanged; docs and hand-offs recorded.

## Release

**Merge only.** The profiles ship inside the `runner-v1.0.0` image ([m3-15](sprint-m3-15.md)), deployed dark by [mi-10](sprint-mi-10.md). `internal/platform/harness` and `runnerapi/lint` are compiled into judge from `v1.13.0` ([m3-07](sprint-m3-07.md)); `deploy.yml` builds nothing runner-specific.

## Definition of Done

CI green (including `runner-it` with Go, C++ and Python) · acceptance criteria met · no host, cluster or infra change · statuses updated (this file + [`../status.md`](../status.md)) · allowlist additions and the multipliers in the decisions log.

## Risks / watch-outs

- **The spike's corpus was Go-heavy** (t3 §9). C++ and Python references may need syscalls P3 didn't log. Add only safe ones, justified; a missed one shows up as a wrongly counted RE, so run every reference and wrong solution under KILL before merging.
- **Python is slow and wide.** Its multiplier may be 3–5× or more; the pack's TL floor (1 s) × multiplier, summed over an item's hidden cases, must still fit m3-03's **45 s tests cap** (L14), i.e. m3-02's Σ TL ≤ 40 CPU-s lint applied per language. Flag any item whose Python Σ TL would exceed it to [m3-13](sprint-m3-13.md)'s re-gate rather than raising caps.
- **C++ compile bombs** (deep templates, `constexpr` loops, huge static arrays, `-static` link time) hit the 15 s / 1 GiB compile caps → CE with CPU counted. Watch the compile-memory headroom inside a 1.25 GiB slot.
- **`int` semantics differ** (Go `int` is 64-bit, C++ `int` 32-bit). The registry's `int` is 32-bit in range, `int64` is explicit; constraints must match.
- **Floats are never canonical.** A float output in `sha256` mode would make correct answers fail on formatting; `CanonicalizableOutput` must say no.
- **Level order and graphs** are where cross-language codecs drift; the byte-identical golden test is the guard, not code review.
- **CI toolchains ≠ image toolchains** by a patch; don't calibrate on CI numbers beyond "provisional". m3-15's in-image run and mi-10's prod run are authoritative.
- **Harness bytes are part of the contract** (`contract_hash`, t1): any change to what crosses fd 3 or fd 4 is a new harness major, not a patch. m3-02 fixed the Go bytes first and packs are gated against them; extending the package must not move them (the compatibility golden is the guard).
