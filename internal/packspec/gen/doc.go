// Package gen is THE seeded perf-case generator registry (t4 §4.1): a closed, public,
// explicit list of six generators that turn a spec and a seed into one case input, the
// canonical fd-3 payload of internal/platform/harness (func-json@1 or class-ops@1).
//
// Public code, private data. An item's pack.json gen[] holds only specs
// ({"gen": "int_array@1", "params": {…}}) and seeds, never generator code. `packlint lock`
// (m3-02) materializes perf cases through WriteInput to compute their expected digests;
// judge (m3-06) imports this package and regenerates the same bytes at runtime, streaming
// each input (≤ MaxInput, 8 MiB, t4 §9.4) to the runner instead of shipping it in the pack.
//
// Determinism: the bytes depend only on (spec, seed). Each generator reads a fixed
// math/rand/v2 PCG stream (rand.NewPCG(seed, <per-generator constant>)) through Uint64
// alone, with this package's own unbiased bounded draws; there is no global rand, no time
// and no map iteration in any output. Every name@v is golden-hash tested (gen_test.go): a
// change that alters a generator's bytes must ship as a new name@v+1, never as an edit of
// @v. Version (recorded in the tests.lock header) is the sorted list of names.
//
// The registry is closed: an explicit list, no init() registration, no plug-ins. A
// generator a later item needs is a new name@1 added to the list here, never a second
// registry (m3-06 imports this package rather than creating internal/judge/gen; graph@1
// is the edge-list generator, there is no graph_edges@1).
//
// Params (strict: unknown keys, nulls, non-integers and duplicate keys are errors):
//
//	int_array@1    n, min, max; distinct (bool), sorted ("" | "asc" | "desc")   → array int[]
//	string@1       n; alphabet (distinct bytes 0x20..0x7e, default a..z)        → string string
//	permutation@1  n; base (0 | 1, default 1)                                    → array int[]
//	tree@1         n, min, max; shape (random | left | right | complete | bst)   → tree TreeNode
//	graph@1        n ≥ 1, m; directed, weighted, connected, self_loops, multi
//	               (bools); wmin, wmax (weighted only, default 1); base (0 | 1)  → n int, edges int[][]
//	op_sequence@1  n; ctor (constructor args); ops [{name, weight ≥ 1 (default 1),
//	               args [{min, max}, …] (int/int64 op params)}]                  → class-ops@1 input
//
// Function-mode generators (all but op_sequence@1) also take "bind" (output name →
// signature param) and "with" (signature param → literal value). Without bind, a
// single-output generator fills a single-param signature, otherwise outputs fill the
// params of the same name; every signature param must be filled exactly once, by an
// output or by with. An int output may fill an int64 param (int[] → int64[], int[][] →
// int64[][]); integers always stay within the filled param's range (int is 32-bit).
package gen
