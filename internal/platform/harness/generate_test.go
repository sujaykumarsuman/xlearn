package harness

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// These tests compile the generated Go harness with SYNTHETIC, hand-made learner files (our
// own test fixtures, never pack or learner code) and run each case as its own process with
// fd 3 / fd 4, as the runner and packlint's executor do.

var update = flag.Bool("update", false, "rewrite testdata/golden")

// allTypes is every Go-reachable registry type.
var allTypes = []string{
	"int", "int64", "float64", "bool", "string",
	"int[]", "int64[]", "float64[]", "bool[]", "string[]",
	"int[][]", "int64[][]", "float64[][]", "bool[][]", "string[][]",
	"ListNode", "ListNode[]", "TreeNode", "TreeNode[]", "GraphNode",
}

func opName(typ string) string {
	r := strings.NewReplacer("[]", "Arr", "64", "Sixtyfour")
	return "Echo" + r.Replace(strings.ToUpper(typ[:1])+typ[1:])
}

// echoSig is a class with one echo op per type, so a single build round-trips them all.
func echoSig(t *testing.T) *Sig {
	s := course.Signature{Mode: "class", Name: "Echo", Params: []course.Param{{Name: "tag", Type: "string"}}}
	for _, typ := range allTypes {
		s.Ops = append(s.Ops, course.Op{Name: opName(typ), Params: []course.Param{{Name: "v", Type: typ}}, Returns: typ})
	}
	s.Ops = append(s.Ops, course.Op{Name: "Touch"})
	return sig(t, ClassOps, s)
}

func echoSource(s *Sig) string {
	var b strings.Builder
	b.WriteString("package main\n\ntype Echo struct{ touched int }\n\nfunc Constructor(tag string) Echo { return Echo{} }\n\nfunc (e *Echo) Touch() { e.touched++ }\n")
	for _, o := range s.Ops {
		if o.Name == "Touch" {
			continue
		}
		fmt.Fprintf(&b, "\nfunc (e *Echo) %s(v %s) %s { return v }\n", o.Name, GoType(o.Params[0].Type), GoType(o.Returns))
	}
	return b.String()
}

// build compiles learner + harness in a temp dir and returns the binary.
func build(t *testing.T, s *Sig, learner string) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	files, err := Generate("go", s.Harness, s)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, GoLearnerFile), []byte(learner), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Path), f.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(dir, "bin")
	cmd := exec.Command("go", append(append([]string{"build"}, GoBuildFlags...), "-o", bin, GoLearnerFile, GoHarnessFile)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), GoBuildEnv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s\n--- harness ---\n%s", err, out, files[0].Data)
	}
	return bin
}

// runCase runs one case: the frame on fd 3, the result from fd 4.
func runCase(t *testing.T, bin string, input []byte) []byte {
	t.Helper()
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin)
	cmd.ExtraFiles = []*os.File{inR, outW}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	inR.Close()
	outW.Close()
	go func() { inW.Write(input); inW.Close() }()
	out, _ := io.ReadAll(outR)
	outR.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("case exited: %v\n%s", err, stderr.String())
	}
	return out
}

func TestGoHarnessRoundTripsEveryType(t *testing.T) {
	s := echoSig(t)
	bin := build(t, s, echoSource(s))
	samples := map[string]string{
		"int": "-2147483648", "int64": "9007199254740993", "float64": "0.1", "bool": "true",
		"string": "\"q\\\"\\\\\\u0001é\U0001F600\"",
		"int[]":  "[1,-2,3]", "int64[]": "[]", "float64[]": "[1e+21,2.5]", "bool[]": "[false]", "string[]": `["",""]`,
		"int[][]": "[[],[1],[2,3]]", "int64[][]": "[[9223372036854775807]]", "float64[][]": "[[0]]", "bool[][]": "[[true,false]]",
		"string[][]": `[["a"],[]]`, "ListNode": "[3,1,2]", "ListNode[]": "[[],[1],[2,3]]",
		"TreeNode": "[5,3,8,null,4,null,9,7]", "TreeNode[]": "[[],[1,null,2]]", "GraphNode": "[[2,4],[1,3],[2,4],[1,3]]",
	}
	ops := []string{`"Echo"`}
	args := []string{`["t"]`}
	for _, typ := range allTypes {
		ops = append(ops, `"`+opName(typ)+`"`)
		args = append(args, "["+samples[typ]+"]")
	}
	ops = append(ops, `"Touch"`)
	args = append(args, "[]")
	raw := []byte(`{"ops":[` + strings.Join(ops, ",") + `],"args":[` + strings.Join(args, ",") + `]}`)
	in, err := DecodeInput(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := EncodeInput(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	out := runCase(t, bin, frame)
	v, err := DecodeOutputFor(s, in, out)
	if err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	// Every echo returns its argument; the constructor and Touch give null.
	want := ListV(Null())
	for i := 1; i < len(in.Ops)-1; i++ {
		want.List = append(want.List, in.Args[i].List[0])
	}
	want.List = append(want.List, Null())
	if !Equal(v, want) {
		t.Fatalf("round trip:\n got %s\nwant %s", v, want)
	}
	// Byte-identical to the Go codec's own frame.
	if !bytes.Equal(out, OKFrame(want)) {
		t.Fatalf("fd-4 bytes differ from OKFrame:\n got %q\nwant %q", out, OKFrame(want))
	}
}

const panicLearner = `package main

func boom(k int, xs []int, unused *ListNode) int {
	var m map[int]int
	var p *ListNode
	switch k {
	case 0:
		return xs[k+5]
	case 1:
		return len(xs[3:k])
	case 2:
		return p.Val
	case 3:
		return k / (k - 3)
	case 4:
		m[k] = 1
	case 5:
		panic("custom")
	case 6:
		return sum(xs)
	}
	return -1
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}
`

func TestGoHarnessPanicClasses(t *testing.T) {
	s := sig(t, FuncJSON, course.Signature{Mode: "function", Name: "boom",
		Params: []course.Param{{Name: "k", Type: "int"}, {Name: "xs", Type: "int[]"}, {Name: "unused", Type: "ListNode"}}, Returns: "int"})
	bin := build(t, s, panicLearner)
	want := []string{"index_out_of_range", "slice_bounds", "nil_dereference", "divide_by_zero", "nil_map_write", "other"}
	for k, cls := range want {
		frame, err := EncodeInput(s, []byte(fmt.Sprintf(`{"args":[%d,[1,2],[]]}`, k)))
		if err != nil {
			t.Fatal(err)
		}
		_, err = DecodeOutput(s, runCase(t, bin, frame))
		var pe *PanicError
		if !errors.As(err, &pe) || pe.Class != cls {
			t.Errorf("k=%d: %v, want panic %s", k, err, cls)
		}
	}
	frame, _ := EncodeInput(s, []byte(`{"args":[6,[1,2,3],[1]]}`))
	if v, err := DecodeOutput(s, runCase(t, bin, frame)); err != nil || !Equal(v, IntV(6)) {
		t.Errorf("ok case: %v %v", v, err)
	}
	// A frame that does not decode is the harness's bad_input, never a learner verdict.
	var he *HarnessError
	if _, err := DecodeOutput(s, runCase(t, bin, AppendFrame(nil, []byte(`{"args":[1]}`)))); !errors.As(err, &he) || he.Code != "bad_input" {
		t.Errorf("bad input: %v", err)
	}
}

const badResultLearner = `package main

import "math"

type Bad struct{ k int }

func Constructor() Bad { return Bad{} }

func (b *Bad) List() *ListNode {
	a := &ListNode{Val: 1}
	a.Next = &ListNode{Val: 2, Next: a}
	return a
}

func (b *Bad) Tree() *TreeNode {
	c := &TreeNode{Val: 2}
	return &TreeNode{Val: 1, Left: c, Right: c}
}

func (b *Bad) Graph() *Node {
	a := &Node{Val: 1}
	c := &Node{Val: 1}
	a.Neighbors = []*Node{c}
	return a
}

func (b *Bad) Float() float64 { return math.Inf(1) }

func (b *Bad) Text() string { return string([]byte{0xff}) }
`

func TestGoHarnessErrors(t *testing.T) {
	s := sig(t, ClassOps, course.Signature{Mode: "class", Name: "Bad", Ops: []course.Op{
		{Name: "List", Returns: "ListNode"}, {Name: "Tree", Returns: "TreeNode"}, {Name: "Graph", Returns: "GraphNode"},
		{Name: "Float", Returns: "float64"}, {Name: "Text", Returns: "string"},
	}})
	bin := build(t, s, badResultLearner)
	for op, code := range map[string]string{"List": "cycle", "Tree": "cycle", "Graph": "invalid_graph", "Float": "non_finite", "Text": "invalid_utf8"} {
		frame, err := EncodeInput(s, []byte(`{"ops":["Bad","`+op+`"],"args":[[],[]]}`))
		if err != nil {
			t.Fatal(err)
		}
		var he *HarnessError
		if _, err := DecodeOutput(s, runCase(t, bin, frame)); !errors.As(err, &he) || he.Code != code {
			t.Errorf("%s: %v, want error %s", op, err, code)
		}
	}
}

func TestGoHarnessMissingFunctionFailsToCompile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	s := funcSig(t)
	files, err := Generate("go", FuncJSON, s)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, GoLearnerFile), []byte("package main\n\nfunc pairSums() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, GoHarnessFile), files[0].Data, 0o644)
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "bin"), GoLearnerFile, GoHarnessFile)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), GoBuildEnv...)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), GoHarnessFile) {
		t.Fatalf("a learner file without the function must fail in the harness file: %v\n%s", err, out)
	}
}

func TestGenerateGolden(t *testing.T) {
	for name, s := range map[string]*Sig{"func-json": funcSig(t), "class-ops": classSig(t)} {
		files, err := Generate("go", s.Harness, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 1 || files[0].Path != GoHarnessFile {
			t.Fatalf("%s: files %v", name, files)
		}
		golden := filepath.Join("testdata", "golden", name+".go.golden")
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, files[0].Data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%v (run go test -update)", err)
		}
		if !bytes.Equal(files[0].Data, want) {
			t.Errorf("%s: generated harness differs from %s (a byte change on fd 3/fd 4 is @2; otherwise -update)", name, golden)
		}
	}
	if _, err := Generate("cpp", FuncJSON, funcSig(t)); !errors.Is(err, ErrLanguagePending) {
		t.Errorf("cpp: %v, want ErrLanguagePending", err)
	}
}
