package gen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// Every signature, spec and seed here is SYNTHETIC and hand-made for these tests.

func mustSig(t testing.TB, h string, s course.Signature) *harness.Sig {
	t.Helper()
	sig, err := harness.ParseSig(h, &s)
	if err != nil {
		t.Fatalf("ParseSig: %v", err)
	}
	return sig
}

func fn(params ...string) course.Signature {
	s := course.Signature{Mode: "function", Name: "solve", Returns: "int"}
	for i := 0; i < len(params); i += 2 {
		s.Params = append(s.Params, course.Param{Name: params[i], Type: params[i+1]})
	}
	return s
}

// The synthetic signatures.
var (
	sigNums     = fn("nums", "int[]")
	sigNums64K  = fn("nums", "int64[]", "k", "int")
	sigStr      = fn("s", "string")
	sigPerm     = fn("perm", "int[]")
	sigTree     = fn("root", "TreeNode")
	sigGraph    = fn("n", "int", "edges", "int[][]")
	sigGraphRev = fn("edges", "int64[][]", "n", "int64")
	sigEdges    = fn("edges", "int[][]")
	sigLedger   = course.Signature{
		Mode: "class", Name: "Ledger",
		Params: []course.Param{{Name: "limit", Type: "int"}},
		Ops: []course.Op{
			{Name: "deposit", Params: []course.Param{{Name: "amount", Type: "int"}}},
			{Name: "withdraw", Params: []course.Param{{Name: "amount", Type: "int64"}}, Returns: "bool"},
			{Name: "balance", Returns: "int64"},
			{Name: "transfer", Params: []course.Param{{Name: "to", Type: "int"}, {Name: "amount", Type: "int"}}, Returns: "bool"},
			{Name: "rename", Params: []course.Param{{Name: "tag", Type: "string"}}},
		},
	}
)

type tcase struct {
	name string
	sig  course.Signature
	gen  string
	par  string
	seed uint64
	want string // golden sha256 of the WriteInput bytes
}

func (c tcase) mode() string {
	if c.sig.Mode == "class" {
		return harness.ClassOps
	}
	return harness.FuncJSON
}

func (c tcase) spec() Spec { return Spec{Gen: c.gen, Params: json.RawMessage(c.par)} }

func run(t testing.TB, sig *harness.Sig, s Spec, seed uint64) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := WriteInput(&b, sig, s, seed); err != nil {
		t.Fatalf("WriteInput(%s %s seed %d): %v", s.Gen, s.Params, seed, err)
	}
	return b.Bytes()
}

// golden pins every generator's bytes: 2–3 fixed (params, seed) per gen@v and the sha256
// of what WriteInput writes. A failure here means a generator's output changed. That is
// NEVER fixed by updating the hash: published perf cases (tests.lock, judge's runtime
// regeneration) depend on name@v producing the same bytes forever. A behaviour change
// ships as a new name@v+1 next to the old one (with its own golden rows); only a change
// that keeps every byte (a refactor) may touch @v's code.
var golden = []tcase{
	{"int_array/plain", sigNums, "int_array@1", `{"n":20,"min":-50,"max":50}`, 1,
		"57d6251dd507a6b7d3a90517df900a591ea504c7bff2ef379ad59a1e5645cb46"},
	{"int_array/distinct-asc", sigNums, "int_array@1", `{"n":16,"min":0,"max":20,"distinct":true,"sorted":"asc"}`, 7,
		"057442323ea33753172573a174679c3dc0839e419b7b3d49eb05cdd01621679e"},
	{"int_array/int64-bind-with", sigNums64K, "int_array@1",
		`{"n":12,"min":-1000000000000,"max":1000000000000,"sorted":"desc","bind":{"array":"nums"},"with":{"k":3}}`, 42,
		"303e3f75bd727947dd7779f70e0dd6cad83f2e7a61b13d4f99c867c751e432f7"},
	{"string/default", sigStr, "string@1", `{"n":40}`, 1, "f1d071f64936db0c03c18c54e83ba6084a6a4afcf221ea1fdf053d22e4ad9508"},
	{"string/escapes", sigStr, "string@1", `{"n":30,"alphabet":"ab\"\\ ~"}`, 9, "93ea48081ddf36c862b9431083c4a1c5836cfe226d81631511a63a30eaef0c83"},
	{"permutation/base1", sigPerm, "permutation@1", `{"n":15}`, 1, "addced94ddc26a3ceea7f2bea4c4e55ee52d12a9460619631495c190118cab05"},
	{"permutation/base0", sigPerm, "permutation@1", `{"n":10,"base":0}`, 2, "b9644c7bf6445e2a604c58ed8e36ccb8edd6b4827aa724f5f7e75052efbbcb64"},
	{"tree/random", sigTree, "tree@1", `{"n":15,"min":-9,"max":9}`, 1, "3d222c38194c0d67037c00c0cd104ab02b0386be034ff0928c06f1415cdf1529"},
	{"tree/bst", sigTree, "tree@1", `{"n":12,"min":1,"max":100,"shape":"bst"}`, 3, "f7c3bc31b68e8076d865ab782b79a74d1a7fb408a5d760f7c6712cddc4c9ae6e"},
	{"tree/left", sigTree, "tree@1", `{"n":6,"min":0,"max":5,"shape":"left"}`, 5, "56a93dc70c4551dfcdf7083dca5b4754cca6e3af1acfafdbce52944c388572e0"},
	{"graph/connected", sigGraph, "graph@1", `{"n":8,"m":10,"connected":true}`, 1, "a80c9edb230e57ba6ed6e5c482e6e0de9057a884708f2742a826311453f41cba"},
	{"graph/directed-weighted", sigGraph, "graph@1", `{"n":6,"m":9,"directed":true,"weighted":true,"wmin":1,"wmax":20,"base":1}`, 2, "5f899ecd3833ec9b27756f9157b5ce1e3efa84af219394fba2f2c75b52429cb3"},
	{"graph/dense-rev", sigGraphRev, "graph@1", `{"n":5,"m":10,"connected":true}`, 3, "d3085ddaba312b2b581c373d9a26deeb079d83e8e459fdbe9e8150b97c838a33"},
	{"op_sequence/weighted", sigLedger, "op_sequence@1",
		`{"n":10,"ctor":[100],"ops":[{"name":"deposit","weight":3,"args":[{"min":1,"max":50}]},{"name":"withdraw","args":[{"min":1,"max":80}]},{"name":"balance","weight":2}]}`, 1,
		"c94c15f52d9e0becf95b158e069a60638d45ae165e47e49cf2ea11f5c1ba5516"},
	{"op_sequence/two-args", sigLedger, "op_sequence@1",
		`{"n":6,"ctor":[-5],"ops":[{"name":"transfer","args":[{"min":0,"max":3},{"min":-10,"max":10}]},{"name":"balance"}]}`, 11,
		"3e9adaf961681201e9ac6b26e1e6c7fbacf5dfa5fc7a763cd4f2c53437dfb1ce"},
}

func TestGolden(t *testing.T) {
	covered := map[string]int{}
	for _, c := range golden {
		t.Run(c.name, func(t *testing.T) {
			sig := mustSig(t, c.mode(), c.sig)
			out := run(t, sig, c.spec(), c.seed)
			sum := sha256.Sum256(out)
			if got := hex.EncodeToString(sum[:]); got != c.want {
				t.Errorf("%s seed %d: sha256 %s, golden %s\n  bytes: %s", c.gen, c.seed, got, c.want, out)
			}
		})
		covered[c.gen]++
	}
	for _, g := range Registry() {
		if n := covered[g.Name()]; n < 2 {
			t.Errorf("%s has %d golden rows, want ≥ 2", g.Name(), n)
		}
	}
}

func TestRegistry(t *testing.T) {
	want := []string{"graph@1", "int_array@1", "op_sequence@1", "permutation@1", "string@1", "tree@1"}
	var names []string
	for _, g := range Registry() {
		names = append(names, g.Name())
	}
	if !slices.Equal(names, want) {
		t.Fatalf("registry %v, want exactly %v (sorted)", names, want)
	}
	if v := Version(); v != strings.Join(want, ",") {
		t.Errorf("Version %q", v)
	}
	streams := map[uint64]string{}
	for _, g := range Registry() {
		if other, dup := streams[g.stream]; dup {
			t.Errorf("%s and %s share a PCG stream", g.Name(), other)
		}
		streams[g.stream] = g.Name()
	}
	g, err := Lookup("graph@1")
	if err != nil {
		t.Fatal(err)
	}
	if o := g.Outputs(); len(o) != 2 || o[0] != (Output{"n", "int"}) || o[1] != (Output{"edges", "int[][]"}) {
		t.Errorf("graph@1 outputs %v", o)
	}
	if g, _ := Lookup("op_sequence@1"); g.Outputs() != nil {
		t.Errorf("op_sequence@1 outputs %v, want nil", g.Outputs())
	}
	if _, err := Lookup("graph_edges@1"); !errors.Is(err, ErrSpec) {
		t.Errorf("Lookup(graph_edges@1) = %v", err)
	}
}

func TestDeterministicAndCanonical(t *testing.T) {
	for _, c := range golden {
		t.Run(c.name, func(t *testing.T) {
			sig := mustSig(t, c.mode(), c.sig)
			a := run(t, sig, c.spec(), c.seed)
			if b := run(t, sig, c.spec(), c.seed); !bytes.Equal(a, b) {
				t.Fatalf("same seed, different bytes:\n%s\n%s", a, b)
			}
			canon, err := harness.CanonicalInput(sig, a)
			if err != nil {
				t.Fatalf("CanonicalInput: %v\n%s", err, a)
			}
			if !bytes.Equal(canon, a) {
				t.Fatalf("not canonical:\n got %s\nwant %s", a, canon)
			}
			// Key order or whitespace in params never changes the bytes.
			var v any
			if err := json.Unmarshal([]byte(c.par), &v); err != nil {
				t.Fatal(err)
			}
			pretty, _ := json.MarshalIndent(v, " ", "  ")
			if b := run(t, sig, Spec{Gen: c.gen, Params: pretty}, c.seed); !bytes.Equal(a, b) {
				t.Errorf("re-spelled params changed the bytes")
			}
		})
	}
}

func TestWithLiterals(t *testing.T) {
	sig := mustSig(t, harness.FuncJSON, fn("eps", "float64", "nums", "int[]", "label", "string", "root", "TreeNode", "grid", "bool[][]"))
	s := Spec{Gen: "int_array@1", Params: json.RawMessage(`{"n":3,"min":0,"max":9,"bind":{"array":"nums"},` +
		`"with":{"eps":0.25,"label":"a\"b\\cé","root":[1,null,2,null,null],"grid":[[true],[]]}}`)}
	out := run(t, sig, s, 5)
	canon, err := harness.CanonicalInput(sig, out)
	if err != nil || !bytes.Equal(canon, out) {
		t.Fatalf("not canonical (%v):\n%s\n%s", err, out, canon)
	}
	want := `{"args":[0.25,[`
	if !bytes.HasPrefix(out, []byte(want)) || !bytes.HasSuffix(out, []byte(`],"a\"b\\cé",[1,null,2],[[true],[]]]}`)) {
		t.Fatalf("literals not written canonically: %s", out)
	}
}

func TestSeedsDiffer(t *testing.T) {
	big := []tcase{
		{sig: sigNums, gen: "int_array@1", par: `{"n":200,"min":0,"max":1000000}`},
		{sig: sigStr, gen: "string@1", par: `{"n":200}`},
		{sig: sigPerm, gen: "permutation@1", par: `{"n":200}`},
		{sig: sigTree, gen: "tree@1", par: `{"n":200,"min":0,"max":1000}`},
		{sig: sigGraph, gen: "graph@1", par: `{"n":100,"m":300}`},
		{sig: sigLedger, gen: "op_sequence@1", par: `{"n":200,"ctor":[1],"ops":[{"name":"deposit","args":[{"min":0,"max":99}]},{"name":"balance"}]}`},
	}
	for _, c := range big {
		sig := mustSig(t, c.mode(), c.sig)
		seen := map[string]uint64{}
		for seed := range uint64(8) {
			out := string(run(t, sig, c.spec(), seed))
			if prev, dup := seen[out]; dup {
				t.Errorf("%s: seeds %d and %d give the same bytes", c.gen, prev, seed)
			}
			seen[out] = seed
		}
	}
}

// decode runs a spec and decodes the result through the harness codec (which validates it).
func decode(t *testing.T, c tcase) (*harness.Sig, *harness.Input) {
	t.Helper()
	sig := mustSig(t, c.mode(), c.sig)
	out := run(t, sig, c.spec(), c.seed)
	in, err := harness.DecodeInput(sig, out)
	if err != nil {
		t.Fatalf("DecodeInput: %v", err)
	}
	if canon, err := harness.CanonicalInput(sig, out); err != nil || !bytes.Equal(canon, out) {
		t.Fatalf("not canonical (%v)", err)
	}
	return sig, in
}

func intsOf(v harness.Value) []int64 {
	a := make([]int64, v.Len())
	for i, e := range v.List {
		a[i] = e.Int
	}
	return a
}

func TestIntArrayValid(t *testing.T) {
	for seed := range uint64(20) {
		// Sparse distinct (rejection path), dense distinct (partial Fisher–Yates path), plain.
		for _, par := range []string{
			`{"n":300,"min":-1000000,"max":1000000,"distinct":true}`,
			`{"n":300,"min":-150,"max":200,"distinct":true,"sorted":"desc"}`,
			`{"n":300,"min":1,"max":300,"distinct":true}`,
			`{"n":300,"min":-3,"max":3,"sorted":"asc"}`,
			`{"n":300,"min":-2147483648,"max":2147483647}`,
		} {
			var p struct {
				N        int    `json:"n"`
				Min      int64  `json:"min"`
				Max      int64  `json:"max"`
				Distinct bool   `json:"distinct"`
				Sorted   string `json:"sorted"`
			}
			_ = json.Unmarshal([]byte(par), &p)
			_, in := decode(t, tcase{sig: sigNums, gen: "int_array@1", par: par, seed: seed})
			a := intsOf(in.Args[0])
			if len(a) != p.N {
				t.Fatalf("%s: len %d", par, len(a))
			}
			seen := map[int64]bool{}
			for i, v := range a {
				if v < p.Min || v > p.Max {
					t.Fatalf("%s: %d out of range", par, v)
				}
				if p.Distinct && seen[v] {
					t.Fatalf("%s: %d repeated", par, v)
				}
				seen[v] = true
				if i > 0 && p.Sorted == "asc" && a[i-1] > v || i > 0 && p.Sorted == "desc" && a[i-1] < v {
					t.Fatalf("%s: not %s at %d", par, p.Sorted, i)
				}
			}
		}
	}
	// int → int64[] keeps 64-bit values; with fills k.
	_, in := decode(t, tcase{sig: sigNums64K, gen: "int_array@1", seed: 3,
		par: `{"n":50,"min":-9000000000000000000,"max":9000000000000000000,"bind":{"array":"nums"},"with":{"k":-7}}`})
	if in.Args[1].Int != -7 || in.Args[0].Len() != 50 {
		t.Fatalf("args %v", in.Args)
	}
	big := false
	for _, v := range intsOf(in.Args[0]) {
		big = big || v > math.MaxInt32 || v < math.MinInt32
	}
	if !big {
		t.Error("no value outside int32 in 50 draws over ±9e18")
	}
}

func TestStringValid(t *testing.T) {
	for seed := range uint64(10) {
		_, in := decode(t, tcase{sig: sigStr, gen: "string@1", par: `{"n":500,"alphabet":"xy\"\\"}`, seed: seed})
		s := in.Args[0].Str
		if len(s) != 500 || strings.Trim(s, "xy\"\\") != "" {
			t.Fatalf("string %q", s)
		}
		_, in = decode(t, tcase{sig: sigStr, gen: "string@1", par: `{"n":2000}`, seed: seed})
		s = in.Args[0].Str
		if len(s) != 2000 || strings.Trim(s, "abcdefghijklmnopqrstuvwxyz") != "" {
			t.Fatalf("string %q", s)
		}
		for c := 'a'; c <= 'z'; c++ {
			if !strings.ContainsRune(s, c) {
				t.Fatalf("%c never drawn in 2000", c)
			}
		}
	}
	_, in := decode(t, tcase{sig: sigStr, gen: "string@1", par: `{"n":0}`})
	if in.Args[0].Str != "" {
		t.Fatal("n=0 not empty")
	}
}

func TestPermutationValid(t *testing.T) {
	for seed := range uint64(10) {
		for _, base := range []int64{0, 1} {
			par := `{"n":500,"base":` + itoa(int(base)) + `}`
			_, in := decode(t, tcase{sig: sigPerm, gen: "permutation@1", par: par, seed: seed})
			a := intsOf(in.Args[0])
			sorted := slices.Sorted(slices.Values(a))
			for i, v := range sorted {
				if v != base+int64(i) {
					t.Fatalf("base %d: not a permutation (%d at %d)", base, v, i)
				}
			}
			if slices.Equal(a, sorted) {
				t.Fatalf("seed %d: identity permutation of 500", seed)
			}
		}
	}
}

// rebuild turns a level order back into a tree (node 0 the root).
func rebuild(lv []harness.Value) (val []int64, left, right []int) {
	if len(lv) == 0 {
		return
	}
	add := func(v int64) int {
		val, left, right = append(val, v), append(left, -1), append(right, -1)
		return len(val) - 1
	}
	add(lv[0].Int)
	queue, i := []int{0}, 1
	for h := 0; h < len(queue) && i < len(lv); h++ {
		x := queue[h]
		for side := 0; side < 2 && i < len(lv); side++ {
			e := lv[i]
			i++
			if e.IsNull() {
				continue
			}
			id := add(e.Int)
			if side == 0 {
				left[x] = id
			} else {
				right[x] = id
			}
			queue = append(queue, id)
		}
	}
	return
}

func TestTreeValid(t *testing.T) {
	for seed := range uint64(10) {
		for _, shape := range []string{"random", "left", "right", "complete", "bst"} {
			for _, n := range []int{0, 1, 2, 7, 300} {
				par := `{"n":` + itoa(n) + `,"min":-500,"max":500,"shape":"` + shape + `"}`
				_, in := decode(t, tcase{sig: sigTree, gen: "tree@1", par: par, seed: seed})
				lv := in.Args[0].List
				val, left, right := rebuild(lv)
				if len(val) != n {
					t.Fatalf("%s n=%d: %d nodes", shape, n, len(val))
				}
				if n > 0 && lv[len(lv)-1].IsNull() {
					t.Fatalf("%s: trailing null", shape)
				}
				for _, v := range val {
					if v < -500 || v > 500 {
						t.Fatalf("%s: value %d out of range", shape, v)
					}
				}
				switch shape {
				case "left", "right":
					for i := range n {
						l, r := left[i], right[i]
						if shape == "right" {
							l, r = r, l
						}
						if r != -1 || (i < n-1 && l != i+1) || (i == n-1 && l != -1) {
							t.Fatalf("%s: node %d is not a chain link", shape, i)
						}
					}
				case "complete":
					for _, e := range lv {
						if e.IsNull() {
							t.Fatalf("complete tree has a hole")
						}
					}
				case "bst":
					// In-order is strictly increasing (valid BST, distinct values).
					var inorder []int64
					var stack []int
					for cur := 0; n > 0 && (cur >= 0 || len(stack) > 0); {
						for ; cur >= 0; cur = left[cur] {
							stack = append(stack, cur)
						}
						cur, stack = stack[len(stack)-1], stack[:len(stack)-1]
						inorder = append(inorder, val[cur])
						cur = right[cur]
					}
					for i := 1; i < len(inorder); i++ {
						if inorder[i-1] >= inorder[i] {
							t.Fatalf("bst: in-order not strictly increasing at %d", i)
						}
					}
					if len(inorder) != n {
						t.Fatalf("bst: in-order visited %d of %d", len(inorder), n)
					}
				}
			}
		}
	}
	// A bst over exactly n values uses all of them.
	_, in := decode(t, tcase{sig: sigTree, gen: "tree@1", par: `{"n":64,"min":1,"max":64,"shape":"bst"}`, seed: 4})
	val, _, _ := rebuild(in.Args[0].List)
	if s := slices.Sorted(slices.Values(val)); s[0] != 1 || s[63] != 64 {
		t.Fatalf("bst over 1..64: %v", s)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

type graphParams struct {
	N, M                                 int
	Directed, Weighted, Connected, Multi bool
	SelfLoops                            bool `json:"self_loops"`
	Wmin, Wmax, Base                     int64
}

func TestGraphValid(t *testing.T) {
	specs := []string{
		`{"n":50,"m":49,"connected":true}`,
		`{"n":50,"m":120,"connected":true,"weighted":true,"wmin":-5,"wmax":5,"base":1}`,
		`{"n":30,"m":200,"directed":true}`,
		`{"n":10,"m":45,"connected":true}`,                     // complete K10 (dense path)
		`{"n":8,"m":64,"directed":true,"self_loops":true}`,     // every directed edge and loop
		`{"n":12,"m":100,"directed":true,"connected":true}`,    // dense directed with a tree
		`{"n":4,"m":40,"multi":true,"self_loops":true}`,        // repeats allowed
		`{"n":1,"m":0,"connected":true}`,                       // a single node
		`{"n":200,"m":600,"connected":true,"weighted":true}`,   // default weights 1..1
		`{"n":3,"m":30,"multi":true,"directed":true,"base":1}`, // multi, no loops
	}
	for seed := range uint64(10) {
		for _, par := range specs {
			var p graphParams
			if err := json.Unmarshal([]byte(par), &p); err != nil {
				t.Fatal(err)
			}
			if !p.Weighted {
				p.Wmin, p.Wmax = 0, 0
			} else if p.Wmin == 0 && p.Wmax == 0 {
				p.Wmin, p.Wmax = 1, 1
			}
			_, in := decode(t, tcase{sig: sigGraph, gen: "graph@1", par: par, seed: seed})
			if in.Args[0].Int != int64(p.N) {
				t.Fatalf("%s: n %d", par, in.Args[0].Int)
			}
			edges := in.Args[1].List
			if len(edges) != p.M {
				t.Fatalf("%s: %d edges", par, len(edges))
			}
			parent := make([]int, p.N)
			for i := range parent {
				parent[i] = i
			}
			var find func(int) int
			find = func(x int) int {
				for parent[x] != x {
					parent[x] = parent[parent[x]]
					x = parent[x]
				}
				return x
			}
			seen := map[[2]int64]bool{}
			for _, e := range edges {
				want := 2
				if p.Weighted {
					want = 3
				}
				if e.Len() != want {
					t.Fatalf("%s: edge %v", par, e)
				}
				u, v := e.List[0].Int, e.List[1].Int
				if u < p.Base || u >= p.Base+int64(p.N) || v < p.Base || v >= p.Base+int64(p.N) {
					t.Fatalf("%s: node out of range in %v", par, e)
				}
				if u == v && !p.SelfLoops {
					t.Fatalf("%s: self loop %v", par, e)
				}
				if p.Weighted && (e.List[2].Int < p.Wmin || e.List[2].Int > p.Wmax) {
					t.Fatalf("%s: weight out of range in %v", par, e)
				}
				k := [2]int64{u, v}
				if !p.Directed && u > v {
					k = [2]int64{v, u}
				}
				if seen[k] && !p.Multi {
					t.Fatalf("%s: duplicate edge %v", par, e)
				}
				seen[k] = true
				parent[find(int(u-p.Base))] = find(int(v - p.Base))
			}
			if p.Connected {
				for i := range p.N {
					if find(i) != find(0) {
						t.Fatalf("%s seed %d: not connected", par, seed)
					}
				}
			}
		}
	}
	// Output order follows the signature, with int64 params.
	_, in := decode(t, tcase{sig: sigGraphRev, gen: "graph@1", par: `{"n":7,"m":6,"connected":true}`, seed: 1})
	if in.Args[1].Int != 7 || in.Args[0].Len() != 6 {
		t.Fatalf("reversed signature args %v", in.Args)
	}
	// An edges-only signature drops n with an explicit bind.
	_, in = decode(t, tcase{sig: sigEdges, gen: "graph@1", par: `{"n":7,"m":6,"bind":{"edges":"edges"}}`, seed: 1})
	if in.Args[0].Len() != 6 {
		t.Fatalf("edges-only args %v", in.Args)
	}
}

func TestOpSequenceValid(t *testing.T) {
	par := `{"n":2000,"ctor":[25],"ops":[` +
		`{"name":"deposit","weight":5,"args":[{"min":1,"max":9}]},` +
		`{"name":"withdraw","args":[{"min":-9000000000,"max":9000000000}]},` +
		`{"name":"transfer","weight":2,"args":[{"min":0,"max":0},{"min":-3,"max":3}]},` +
		`{"name":"balance","weight":2}]}`
	for seed := range uint64(5) {
		_, in := decode(t, tcase{sig: sigLedger, gen: "op_sequence@1", par: par, seed: seed})
		if len(in.Ops) != 2001 || in.Ops[0] != "Ledger" || in.Args[0].Len() != 1 || in.Args[0].List[0].Int != 25 {
			t.Fatalf("ops %d, first %q %v", len(in.Ops), in.Ops[0], in.Args[0])
		}
		count := map[string]int{}
		for i, op := range in.Ops[1:] {
			a := in.Args[i+1].List
			count[op]++
			switch op {
			case "deposit":
				if a[0].Int < 1 || a[0].Int > 9 {
					t.Fatalf("deposit %v", a)
				}
			case "withdraw":
				if a[0].Int < -9000000000 || a[0].Int > 9000000000 {
					t.Fatalf("withdraw %v", a)
				}
			case "transfer":
				if a[0].Int != 0 || a[1].Int < -3 || a[1].Int > 3 {
					t.Fatalf("transfer %v", a)
				}
			case "balance":
				if len(a) != 0 {
					t.Fatalf("balance %v", a)
				}
			default:
				t.Fatalf("op %q was not listed", op)
			}
		}
		// Weights 5:1:2:2 over 2000 draws: deposit ≈ 1000, withdraw ≈ 200.
		if count["deposit"] < 850 || count["deposit"] > 1150 || count["withdraw"] < 120 || count["withdraw"] > 280 {
			t.Fatalf("weights ignored: %v", count)
		}
	}
	_, in := decode(t, tcase{sig: sigLedger, gen: "op_sequence@1", par: `{"n":0,"ctor":[1],"ops":[{"name":"balance"}]}`})
	if len(in.Ops) != 1 {
		t.Fatalf("n=0: ops %v", in.Ops)
	}
}

func TestCheckRejects(t *testing.T) {
	ledger := `"ctor":[1],"ops":[{"name":"balance"}]`
	cases := []struct {
		name, want string
		sig        course.Signature
		gen, par   string
	}{
		{"unknown generator", "unknown generator", sigNums, "int_array@2", `{"n":1,"min":0,"max":1}`},
		{"unknown param key", `unknown param "bogus"`, sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"bogus":1}`},
		{"key case matters", `unknown param "N"`, sigNums, "int_array@1", `{"N":1,"n":1,"min":0,"max":1}`},
		{"missing n", "n is required", sigNums, "int_array@1", `{"min":0,"max":1}`},
		{"null value", "must not be null", sigNums, "int_array@1", `{"n":1,"min":0,"max":null}`},
		{"non-integer n", "64-bit integer", sigNums, "int_array@1", `{"n":1.5,"min":0,"max":1}`},
		{"string n", "64-bit integer", sigNums, "int_array@1", `{"n":"3","min":0,"max":1}`},
		{"duplicate key", "duplicate", sigNums, "int_array@1", `{"n":1,"n":2,"min":0,"max":1}`},
		{"params not an object", "want a JSON object", sigNums, "int_array@1", `[1]`},
		{"min > max", "min 5 > max 3", sigNums, "int_array@1", `{"n":1,"min":5,"max":3}`},
		{"negative n", "out of [0,", sigNums, "int_array@1", `{"n":-1,"min":0,"max":1}`},
		{"n over cap", "out of [0, 1000000]", sigNums, "int_array@1", `{"n":1000001,"min":0,"max":1}`},
		{"string n over cap", "out of [0, 4000000]", sigStr, "string@1", `{"n":4000001}`},
		{"permutation n over cap", "out of [0, 1000000]", sigPerm, "permutation@1", `{"n":1000001}`},
		{"tree n over cap", "out of [0, 200000]", sigTree, "tree@1", `{"n":200001,"min":0,"max":1}`},
		{"graph n over cap", "out of [1, 200000]", sigGraph, "graph@1", `{"n":200001,"m":0}`},
		{"graph m over cap", "out of [0, 500000]", sigGraph, "graph@1", `{"n":200000,"m":500001,"multi":true}`},
		{"op_sequence n over cap", "out of [0, 200000]", sigLedger, "op_sequence@1", `{"n":200001,` + ledger + `}`},
		{"distinct impossible", "distinct", sigNums, "int_array@1", `{"n":10,"min":0,"max":5,"distinct":true}`},
		{"int range for int param", "leaves the int range", sigNums, "int_array@1", `{"n":1,"min":0,"max":2147483648}`},
		{"bad sorted", "sorted", sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"sorted":"up"}`},
		{"missing binding", "param k is not bound", sigNums64K, "int_array@1", `{"n":1,"min":0,"max":1,"bind":{"array":"nums"}}`},
		{"no output bound", "no output fills", sigNums64K, "int_array@1", `{"n":1,"min":0,"max":1}`},
		{"double binding via with", "bound twice", sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"with":{"nums":[1]}}`},
		{"double binding via bind", "bound twice", sigGraph, "graph@1", `{"n":2,"m":1,"bind":{"n":"n","edges":"n"}}`},
		{"bind unknown output", `output "arr"`, sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"bind":{"arr":"nums"}}`},
		{"bind unknown param", "not a param", sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"bind":{"array":"xs"}}`},
		{"with unknown param", "not a param", sigNums, "int_array@1", `{"n":1,"min":0,"max":1,"with":{"xs":1}}`},
		{"type mismatch", "cannot fill param nums (int[])", sigNums, "string@1", `{"n":1}`},
		{"int cannot fill int[]", "cannot fill", fn("nums", "int[]"), "graph@1", `{"n":2,"m":1,"bind":{"n":"nums"}}`},
		{"op_sequence on a function", "is a function", sigNums, "op_sequence@1", `{"n":1,` + ledger + `}`},
		{"int_array on a class", "is a class", sigLedger, "int_array@1", `{"n":1,"min":0,"max":1}`},
		{"unknown op", `"nope" is not an op`, sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"nope"}]}`},
		{"constructor as op", "is the constructor", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"Ledger"}]}`},
		{"op listed twice", "listed twice", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"balance"},{"name":"balance"}]}`},
		{"non-int op param", "int and int64 arguments only", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"rename","args":[{"min":0,"max":1}]}]}`},
		{"op arg count", "takes 1 argument", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"deposit"}]}`},
		{"op arg range for int", "leaves the int range", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"deposit","args":[{"min":0,"max":3000000000}]}]}`},
		{"op unknown key", `unknown param "wieght"`, sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"balance","wieght":2}]}`},
		{"op zero weight", "weight 0", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[{"name":"balance","weight":0}]}`},
		{"no ops", "at least one op", sigLedger, "op_sequence@1", `{"n":1,"ctor":[1],"ops":[]}`},
		{"ctor missing", "ctor is required", sigLedger, "op_sequence@1", `{"n":1,"ops":[{"name":"balance"}]}`},
		{"ctor wrong type", "ctor limit", sigLedger, "op_sequence@1", `{"n":1,"ctor":["x"],"ops":[{"name":"balance"}]}`},
		{"bind on op_sequence", `unknown param "bind"`, sigLedger, "op_sequence@1", `{"n":1,` + ledger + `,"bind":{}}`},
		{"with literal of the wrong type", "with k", sigNums64K, "int_array@1", `{"n":1,"min":0,"max":1,"bind":{"array":"nums"},"with":{"k":"x"}}`},
		{"with literal out of int range", "with k", sigNums64K, "int_array@1", `{"n":1,"min":0,"max":1,"bind":{"array":"nums"},"with":{"k":2147483648}}`},
		{"bad alphabet byte", "printable ASCII", sigStr, "string@1", `{"n":1,"alphabet":"a\tb"}`},
		{"repeated alphabet", "repeats", sigStr, "string@1", `{"n":1,"alphabet":"aba"}`},
		{"empty alphabet", "alphabet is empty", sigStr, "string@1", `{"n":1,"alphabet":""}`},
		{"permutation base", "base 2", sigPerm, "permutation@1", `{"n":1,"base":2}`},
		{"tree shape", "shape", sigTree, "tree@1", `{"n":1,"min":0,"max":1,"shape":"zigzag"}`},
		{"tree values are 32-bit", "leaves the TreeNode range", sigTree, "tree@1", `{"n":1,"min":0,"max":4294967296}`},
		{"bst impossible", "bst", sigTree, "tree@1", `{"n":5,"min":1,"max":4,"shape":"bst"}`},
		{"graph n zero", "out of [1,", sigGraph, "graph@1", `{"n":0,"m":0}`},
		{"graph too many edges", "exceeds the 6 distinct edges", sigGraph, "graph@1", `{"n":4,"m":7}`},
		{"graph connected needs edges", "connected needs m ≥ n-1", sigGraph, "graph@1", `{"n":5,"m":3,"connected":true}`},
		{"graph one node no loops", "need self_loops", sigGraph, "graph@1", `{"n":1,"m":1,"multi":true}`},
		{"graph weights unweighted", "need weighted", sigGraph, "graph@1", `{"n":2,"m":1,"wmax":5}`},
		{"graph wmin > wmax", "wmin 5 > wmax 1", sigGraph, "graph@1", `{"n":2,"m":1,"weighted":true,"wmin":5}`},
		{"graph weight int range", "leaves the int range", sigGraph, "graph@1", `{"n":2,"m":1,"weighted":true,"wmax":3000000000}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := harness.FuncJSON
			if c.sig.Mode == "class" {
				h = harness.ClassOps
			}
			sig := mustSig(t, h, c.sig)
			s := Spec{Gen: c.gen, Params: json.RawMessage(c.par)}
			err := Check(sig, s)
			if err == nil || !errors.Is(err, ErrSpec) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Check = %v, want an ErrSpec containing %q", err, c.want)
			}
			if werr := WriteInput(io.Discard, sig, s, 1); werr == nil || werr.Error() != err.Error() {
				t.Fatalf("WriteInput = %v, want Check's error", werr)
			}
		})
	}
	if err := Check(nil, Spec{Gen: "int_array@1"}); !errors.Is(err, ErrSpec) {
		t.Errorf("nil sig: %v", err)
	}
	// The int64 range is allowed for an int64 param.
	if err := Check(mustSig(t, harness.FuncJSON, sigNums64K), Spec{Gen: "int_array@1",
		Params: json.RawMessage(`{"n":1,"min":-9223372036854775808,"max":9223372036854775807,"bind":{"array":"nums"},"with":{"k":0}}`)}); err != nil {
		t.Errorf("full int64 range: %v", err)
	}
}

// countWriter counts what reaches the underlying writer.
type countWriter struct{ n int }

func (w *countWriter) Write(p []byte) (int, error) { w.n += len(p); return len(p), nil }

func TestInputCap(t *testing.T) {
	sig := mustSig(t, harness.FuncJSON, fn("nums", "int64[]"))
	// 1e6 values of ~19 digits ≈ 20 MB: passes Check, fails the 8 MiB stream cap.
	s := Spec{Gen: "int_array@1", Params: json.RawMessage(`{"n":1000000,"min":1000000000000000000,"max":9000000000000000000}`)}
	if err := Check(sig, s); err != nil {
		t.Fatal(err)
	}
	var w countWriter
	err := WriteInput(&w, sig, s, 1)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("WriteInput = %v, want ErrTooLarge", err)
	}
	if w.n > MaxInput {
		t.Fatalf("%d bytes reached the writer, cap %d", w.n, MaxInput)
	}
	// The largest permutation (≈ 6.9 MB) fits.
	w = countWriter{}
	if err := WriteInput(&w, mustSig(t, harness.FuncJSON, sigPerm), Spec{Gen: "permutation@1", Params: json.RawMessage(`{"n":1000000}`)}, 1); err != nil {
		t.Fatalf("permutation n=1e6: %v", err)
	}
	if w.n < 6_000_000 || w.n > MaxInput {
		t.Fatalf("permutation n=1e6 wrote %d bytes", w.n)
	}
}

func TestUniform(t *testing.T) {
	src := rand.NewPCG(1, 2)
	hits := make([]int, 7)
	for range 7000 {
		v := uniform(src, -3, 3)
		if v < -3 || v > 3 {
			t.Fatalf("uniform(-3,3) = %d", v)
		}
		hits[v+3]++
	}
	for i, h := range hits {
		if h < 800 || h > 1200 {
			t.Errorf("value %d drawn %d/7000 times", i-3, h)
		}
	}
	for range 1000 {
		if v := uniform(src, math.MaxInt64-1, math.MaxInt64); v < math.MaxInt64-1 {
			t.Fatalf("uniform near MaxInt64 = %d", v)
		}
		if v := uniform(src, math.MinInt64, math.MinInt64+1); v > math.MinInt64+1 {
			t.Fatalf("uniform near MinInt64 = %d", v)
		}
		if v := uniform(src, 5, 5); v != 5 {
			t.Fatalf("uniform(5,5) = %d", v)
		}
		_ = uniform(src, math.MinInt64, math.MaxInt64)
	}
	// below never reaches n, across a non-power-of-two bound that forces rejection.
	n := uint64(1)<<63 + 1
	for range 1000 {
		if v := below(src, n); v >= n {
			t.Fatalf("below(%d) = %d", n, v)
		}
	}
}

func BenchmarkLargest(b *testing.B) {
	cases := []tcase{
		{sig: sigNums, gen: "int_array@1", par: `{"n":1000000,"min":-1000,"max":1000}`},
		{sig: sigNums, gen: "int_array@1", par: `{"n":500000,"min":0,"max":1000000000,"distinct":true}`},
		{sig: sigStr, gen: "string@1", par: `{"n":4000000}`},
		{sig: sigTree, gen: "tree@1", par: `{"n":200000,"min":-1000000,"max":1000000,"shape":"bst"}`},
		{sig: sigGraph, gen: "graph@1", par: `{"n":200000,"m":400000,"connected":true,"weighted":true,"wmax":9}`},
		{sig: sigLedger, gen: "op_sequence@1", par: `{"n":200000,"ctor":[1],"ops":[{"name":"deposit","args":[{"min":0,"max":999}]},{"name":"balance"}]}`},
	}
	for _, c := range cases {
		sig := mustSig(b, c.mode(), c.sig)
		b.Run(c.gen, func(b *testing.B) {
			for b.Loop() {
				if err := WriteInput(io.Discard, sig, c.spec(), 1); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
