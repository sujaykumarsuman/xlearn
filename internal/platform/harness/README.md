# `internal/platform/harness` — the harness wire format (`func-json@1`, `class-ops@1`)

The public, versioned harness codec: the closed type registry, canonical JSON, the fd-3/fd-4 frame protocol and
the per-language harness generators. **This file is the `@1` spec.** m3-02 created the package with the Go half;
m3-04 adds C++ and Python without changing a byte on fd 3 or fd 4. `harness@v` is part of `contract_hash`, so a
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
- m3-04 adds `templates/cpp`: `zz_xl_harness.cpp` + `xl_prelude.hpp`, where the learner calls a method of
  `class Solution`.
- m3-04 also adds `templates/python`: `__main__.py` + `xl_prelude.py`, also through `class Solution`.

## Tests

- `codec_test.go`: the type registry, canonical goldens, strict rejections, inputs and outputs.
- `generate_test.go` builds the generated Go harness with synthetic learner files and runs each case as its own
  process with fd 3 / fd 4. It checks that every type round-trips byte-identically to `OKFrame`, that panic
  classes and harness errors come back, and that a missing function fails in the harness file.
