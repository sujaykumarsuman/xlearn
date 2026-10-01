# `internal/platform/harness` — the harness wire format (`func-json@1`, `class-ops@1`)

The public, versioned harness codec: the closed type registry, canonical JSON, the fd-3/fd-4 frame protocol and
the per-language harness generators. **This file is the `@1` spec.** m3-02 created the package with the Go half;
m3-04 added C++ and Python without changing a byte on fd 3 or fd 4. `harness@v` is part of `contract_hash`, so a
change that alters frame bytes is a new major (`@2`), never an edit of `@1`.

Users: packlint (m3-02, every pack gate), the runner profiles (m3-04), judge (m3-06: `Generate`, `EncodeInput`,
`DecodeOutput`, `CanonicalizableOutput`). A leaf library: the stdlib, `internal/course` and
`internal/platform/runnerapi` only (`leaf_test.go`).

## Types (closed; t1 §7.1)

| Type | Go | JSON |
|---|---|---|
| `int` | `int` (values in the 32-bit range) | integer |
| `int64` | `int64` | integer |
| `float64` | `float64` | number (finite) |
| `bool` / `string` | `bool` / `string` (valid UTF-8) | `true`/`false` / string |
| `T[]`, `T[][]` (scalars and `string`) | `[]T`, `[][]T` | arrays |
| `ListNode`, `ListNode[]` | `*ListNode{Val int; Next *ListNode}` | the values in order: `[1,2,3]` |
| `TreeNode`, `TreeNode[]` | `*TreeNode{Val int; Left, Right *TreeNode}` | LeetCode level order, `null` holes, trailing `null`s trimmed: `[1,null,2]` |
| `GraphNode` | `*Node{Val int; Neighbors []*Node}` | 1-indexed adjacency lists (node `i+1`'s neighbours): `[[2],[1]]` |
| `void` | — | `null` (a class op with no `returns`) |

Anything else is an error (`ParseType`); a new type is code plus a release. Decoding is strict: integer ranges,
finite floats, valid UTF-8, nesting ≤ 64, ≤ 10⁶ nodes, a well-formed level order and adjacency.

## Canonical JSON

No whitespace; integers in decimal; strings escape only `"`, `\` and the controls `< 0x20` (`\"`, `\\`,
`\u00xx` with lowercase hex), everything else raw UTF-8; object keys sorted by their UTF-8 bytes; the node shapes
above. **Floats are never canonical**: their bytes-mode spelling is Go's shortest round trip (`strconv` `'g'`,
`-1`: `0.1`, `3`, `1e+21`; `-0` is `0`). An output holding a float is compared with a float checker
(`CanonicalizableOutput` is false), never by sha256.

## Frames

One process per case. The harness reads exactly one frame on **fd 3** and writes exactly one frame on **fd 4**.
A frame is a `u32` big-endian length followed by that many bytes of canonical JSON.

```
fd 3  func-json@1   {"args":[a0,a1,…]}
      class-ops@1   {"args":[[ctor args],[op1 args],…],"ops":["Class","op1",…]}      (constructor first; keys sorted)
fd 4  {"ok":<value>}          the result; class-ops@1: one entry per op, null for the constructor and void ops
      {"panic":"<class>"}     a caught runtime error from the closed per-language set (below)
      {"error":"<code>"}      bad_input | cycle | too_many_nodes | invalid_utf8 | non_finite | invalid_graph
```

- Nothing else is ever written to fd 4; stdout and stderr stay the program's (Run mode only).
- After an `ok`, `panic` or `error` frame the process exits 0. A crash the harness cannot catch (a fatal error,
  a signal, a stack overflow) leaves fd 4 empty, with a non-zero exit.
- `error` codes: `bad_input` means the input frame did not decode, which is a judge-side fault and never a
  learner verdict. Every other code means the learner's result could not be encoded, which judge treats as WA.
- Panic classes:
  - Go: `index_out_of_range`, `nil_dereference`, `divide_by_zero`, `slice_bounds`, `nil_map_write`, `other`.
  - C++ (m3-04): `out_of_range`, `bad_alloc`, `length_error`, `logic_error`, `runtime_error`, `other`.
  - Python (m3-04): `IndexError`, `KeyError`, `ValueError`, `TypeError`, `ZeroDivisionError`, `RecursionError`,
    `AttributeError`, `AssertionError`, `other`.
- A perf case's expected digest (packspec) is `{"bytes": len, "sha256": hex}` of the whole fd-4 frame (`OKFrame`),
  which is exactly what the runner's `OutputSHA256` / `OutputBytes` measure.

## Generated files (Go, m3-02)

| Learner file | Harness file | Build |
|---|---|---|
| `solution.go` (`package main`): the function (`func-json@1`), or the class with `func Constructor(<params>) <Class>` (or `*<Class>`) and one method per op (`class-ops@1`) | `zz_xl_harness.go` (`package main`): `func main`, the node types the signature uses, a tiny JSON reader/writer, `recover()` → panic class; no reflection | `go build -trimpath -buildvcs=false -o bin solution.go zz_xl_harness.go` with `GoBuildEnv` |

- Identifiers starting with `xl` are reserved for the harness.
- A learner file without the function, or with the wrong signature, fails to compile **in the harness file**.
  Judge replaces that diagnostic with its fixed CE text (m3-06).
- Templates live in `templates/go/` (`runtime.tmpl`, `func-json.tmpl`, `class-ops.tmpl`). The goldens in
  `testdata/golden/` pin the generated source. A source change that keeps the frame bytes refreshes the goldens
  with `go test -update`; a byte change on fd 3/fd 4 is `@2`.
- `Generate` returns `[]runnerapi.File` (`runnerapi` exists since m3-03).

## Generated files (C++ and Python, m3-04)

The same bytes on fd 3 and fd 4; only the source changes.

| Language | Learner file | Harness files | Build (the runner profile) |
|---|---|---|---|
| C++ | `solution.cpp`: `class Solution` with the method (`func-json@1`), or the class with its constructor and one method per op (`class-ops@1`); no `main`, no includes needed | `zz_xl_harness.cpp`: includes `xl_prelude.hpp`, defines the node structs the signature uses (`ListNode{val,next}`, `TreeNode{val,left,right}`, `Node{val,neighbors}`, LeetCode shapes), `#include "solution.cpp"`, then `main`; `xl_prelude.hpp`: `<bits/stdc++.h>`, `using namespace std;`, the reader/writer and the exception → panic-class map in `namespace xlh` | `g++ -std=gnu++20 -O2 -static … zz_xl_harness.cpp`; diagnostics keep `solution.cpp:line` |
| Python | `solution.py`: `class Solution` with the method, or the class with `__init__` and one method per op | `__main__.py`: `XL_REQUIRES`, installs the node classes the signature uses into `builtins` (so `solution.py` sees `ListNode`, `TreeNode`, `Node`), imports `solution`, one case; `xl_prelude.py`: the node classes, the reader (`json`) and writer, the exception → panic-class map | a stored zipapp `app.pyz`, run with `python3 -s -P -S -B` |

- `xl_prelude.hpp` and `xl_prelude.py` are **one static file each** for every signature (a precompiled header can
  serve the C++ one, m3-15). The node structs sit in the generated `zz_xl_harness.cpp`, and only the ones the
  signature uses, so a learner's own `Node` helper doesn't clash.
- A learner file without the function or with the wrong signature fails in the harness file: C++ at compile
  (`zz_xl_harness.cpp`); Python at the profile's compile step, which checks `solution.py` against
  `XL_REQUIRES` (class, `(method, positional arity)` pairs) and reports a miss in `__main__.py`.
- A wrong-typed Python result (a `str` for `int`, `None` for a scalar, a `bool` for an `int`) raises `TypeError`
  in the encoder: `{"panic":"TypeError"}`. `None` for an array or a node type is `[]`, like Go's nil. An `int`
  outside its range is written as is (the decoder rejects it, as for Go).
- C++ catches `out_of_range`, `length_error`, `logic_error`, `bad_alloc`, `runtime_error`, then anything else
  (`other`); Python checks `RecursionError`, `ZeroDivisionError`, `IndexError`, `KeyError`, `ValueError`,
  `TypeError`, `AttributeError`, `AssertionError` by `isinstance`, then `other`. A learner `exit`, a signal or a stack
  overflow leaves fd 4 empty, as in Go. Python's harness raises `sys.setrecursionlimit` to 10⁶; the profile sets
  `RLIMIT_STACK` to the memory limit for C++ and Python.
- Floats: C++ and Python spell a float exactly as Go's `strconv.FormatFloat(v, 'g', -1, 64)` (shortest round-trip
  digits from `std::to_chars` / `repr`, `%e` when the exponent is < -4 or ≥ 6).
- Identifiers starting with `xl` are reserved; `Generate` refuses a name that is a keyword of the target language
  or that the generated code uses (`main`, the node types, Python's `self`).
- `Starter(lang, harness, sig)` returns the learner-file skeleton (zero-value returns, empty methods) used when an
  item has no `_starter/solution.<ext>`; it compiles with the harness and fails the samples.

## Tests

- `codec_test.go`: the type registry, canonical goldens, strict rejections, inputs and outputs.
- `generate_test.go` builds the generated Go harness with synthetic learner files and runs each case as its own
  process with fd 3 / fd 4. It checks that every type round-trips byte-identically to `OKFrame`, that panic
  classes and harness errors come back, and that a missing function fails in the harness file.
- `crosslang_test.go` (m3-04): the C++ and Python harnesses against Go's bytes — every type through an echo class,
  hundreds of floats, m3-02's fixture references, panic classes, harness errors in every language, a missing C++
  method in the harness file, starters, and goldens for the generated C++ and Python sources. A missing `g++`
  (with `bits/stdc++.h`) or `python3` ≥ 3.10 skips that language (macOS has no libstdc++); CI's Linux `go` lane has
  both.
- `contentcheck_it_test.go` (runner-it lane): the public content check through the real jail.
