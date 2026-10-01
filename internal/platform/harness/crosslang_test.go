package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The C++ and Python harnesses against the Go one (m3-04). Every learner file here is a
// SYNTHETIC, hand-made fixture. A language whose toolchain is missing is skipped (macOS has
// no libstdc++ bits/stdc++.h); CI's Linux `go` lane has g++ and python3, and the runner-it lane
// runs the same codecs through the real jail.

// program is one built harness + learner.
type program struct {
	lang string
	argv []string
	dir  string
}

var (
	toolOnce sync.Once
	haveCpp  bool
	havePy   bool
)

func detectTools() {
	toolOnce.Do(func() {
		if _, err := exec.LookPath("g++"); err == nil {
			cmd := exec.Command("g++", "-std=gnu++20", "-fsyntax-only", "-x", "c++", "-")
			cmd.Stdin = strings.NewReader("#include <bits/stdc++.h>\nint main() { char b[32]; std::to_chars(b, b + 32, 1.5); }\n")
			haveCpp = cmd.Run() == nil
		}
		if _, err := exec.LookPath("python3"); err == nil {
			havePy = exec.Command("python3", "-c", "import sys; assert sys.version_info >= (3, 10)").Run() == nil
		}
	})
}

// haveLang reports whether a language's toolchain is available.
func haveLang(lang string) bool {
	detectTools()
	switch lang {
	case "cpp":
		return haveCpp
	case "python":
		return havePy
	}
	_, err := exec.LookPath("go")
	return err == nil
}

func needLang(t *testing.T, lang string) {
	t.Helper()
	detectTools()
	switch lang {
	case "go":
		if _, err := exec.LookPath("go"); err != nil {
			t.Skip("no go toolchain on PATH")
		}
	case "cpp":
		if !haveCpp {
			t.Skip("no g++ with bits/stdc++.h and std::to_chars on PATH")
		}
	case "python":
		if !havePy {
			t.Skip("no python3 >= 3.10 on PATH")
		}
	}
}

// buildLang writes learner + generated harness files and builds them (C++ is compiled; Go
// is go-built; Python runs __main__.py from the directory). A build failure returns the
// tool's output as the error.
func buildLang(t *testing.T, lang string, s *Sig, learner string) (*program, error) {
	t.Helper()
	needLang(t, lang)
	files, err := Generate(lang, s.Harness, s)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LearnerFile(lang)), []byte(learner), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Path), f.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(dir, "bin")
	var cmd *exec.Cmd
	switch lang {
	case "go":
		cmd = exec.Command("go", append(append([]string{"build"}, GoBuildFlags...), "-o", bin, GoLearnerFile, GoHarnessFile)...)
		cmd.Env = append(os.Environ(), GoBuildEnv...)
	case "cpp":
		cmd = exec.Command("g++", "-std=gnu++20", "-O1", "-pipe", "-o", bin, CppHarnessFile)
	case "python":
		return &program{lang: lang, argv: []string{"python3", "-S", "-B", PythonMainFile}, dir: dir}, nil
	}
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("%s build: %v\n%s", lang, err, out)
	}
	return &program{lang: lang, argv: []string{bin}, dir: dir}, nil
}

func mustBuild(t *testing.T, lang string, s *Sig, learner string) *program {
	t.Helper()
	p, err := buildLang(t, lang, s, learner)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// run runs one case: the frame on fd 3, the result from fd 4. A non-zero exit is an error.
func (p *program) run(t *testing.T, input []byte) []byte {
	t.Helper()
	out, err := p.try(input)
	if err != nil {
		t.Fatalf("%s case: %v", p.lang, err)
	}
	return out
}

func (p *program) try(input []byte) ([]byte, error) {
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(p.argv[0], p.argv[1:]...)
	cmd.Dir = p.dir
	cmd.ExtraFiles = []*os.File{inR, outW}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	inR.Close()
	outW.Close()
	go func() { inW.Write(input); inW.Close() }()
	out, _ := io.ReadAll(outR)
	outR.Close()
	if err := cmd.Wait(); err != nil {
		return out, fmt.Errorf("%v\n%s", err, stderr.String())
	}
	return out, nil
}

// ---- learner sources for the echo class (one echo op per type) ----

func echoCpp(s *Sig) string {
	var b strings.Builder
	b.WriteString("class Echo {\npublic:\n    Echo(string tag) {}\n    void Touch() {}\n")
	for _, o := range s.Ops {
		if o.Name == "Touch" {
			continue
		}
		ty := CppType(o.Returns)
		fmt.Fprintf(&b, "    %s %s(%s v) { return v; }\n", ty, o.Name, ty)
	}
	b.WriteString("};\n")
	return b.String()
}

func echoPython(s *Sig) string {
	var b strings.Builder
	b.WriteString("class Echo:\n    def __init__(self, tag):\n        self.tag = tag\n\n    def Touch(self):\n        pass\n")
	for _, o := range s.Ops {
		if o.Name != "Touch" {
			fmt.Fprintf(&b, "\n    def %s(self, v):\n        return v\n", o.Name)
		}
	}
	return b.String()
}

func echoLearner(lang string, s *Sig) string {
	switch lang {
	case "go":
		return echoSource(s)
	case "cpp":
		return echoCpp(s)
	}
	return echoPython(s)
}

// echoInput is one class-ops@1 case calling every echo op once.
func echoInput(t *testing.T, s *Sig, samples map[string]string) (*Input, []byte) {
	t.Helper()
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
	return in, frame
}

var echoSamples = []map[string]string{
	{
		"int": "-2147483648", "int64": "9007199254740993", "float64": "0.1", "bool": "true",
		"string": "\"q\\\"\\\\\\u0001é\U0001F600\u007f\"",
		"int[]":  "[1,-2,3]", "int64[]": "[]", "float64[]": "[1e+21,2.5,-0,100000,1e+06,1.234567e+06,0.0001,1e-05]",
		"bool[]": "[false]", "string[]": `["",""]`,
		"int[][]": "[[],[1],[2,3]]", "int64[][]": "[[9223372036854775807,-9223372036854775808]]", "float64[][]": "[[0],[],[3,-7.25]]",
		"bool[][]": "[[true,false],[]]", "string[][]": `[["a"],[]]`, "ListNode": "[3,1,2]", "ListNode[]": "[[],[1],[2,3]]",
		"TreeNode": "[5,3,8,null,4,null,9,7]", "TreeNode[]": "[[],[1,null,2]]", "GraphNode": "[[2,4],[1,3],[2,4],[1,3]]",
	},
	{
		"int": "2147483647", "int64": "-1", "float64": "-1.5e-300", "bool": "false",
		"string": `"tab\there\nnew \u001f end / \u2028"`,
		"int[]":  "[]", "int64[]": "[0,1]", "float64[]": "[]", "bool[]": "[]", "string[]": `["\"","\\"]`,
		"int[][]": "[]", "int64[][]": "[[]]", "float64[][]": "[]", "bool[][]": "[[]]", "string[][]": `[[]]`,
		"ListNode": "[]", "ListNode[]": "[]", "TreeNode": "[]", "TreeNode[]": "[[1,2,3,4,null,null,5]]",
		"GraphNode": "[[1,2],[1]]",
	},
	{
		"int": "0", "int64": "0", "float64": "123456.7", "bool": "true", "string": `""`,
		"int[]": "[0]", "int64[]": "[]", "float64[]": "[5e-324,1.7976931348623157e+308]", "bool[]": "[true,true]",
		"string[]": `["日本語"]`, "int[][]": "[[0]]", "int64[][]": "[]", "float64[][]": "[[]]", "bool[][]": "[]",
		"string[][]": "[]", "ListNode": "[7]", "ListNode[]": "[[]]", "TreeNode": "[1,2,null,3,null,4,null,5]",
		"TreeNode[]": "[]", "GraphNode": "[]",
	},
}

// TestCrossLanguageEchoIsByteIdentical: every registry type round-trips through each
// language's harness, and every fd-4 frame equals the Go harness's bytes (m3-02's @1) and
// the codec's own OKFrame.
func TestCrossLanguageEchoIsByteIdentical(t *testing.T) {
	s := echoSig(t)
	progs := map[string]*program{}
	for _, lang := range Languages {
		if haveLang(lang) {
			progs[lang] = mustBuild(t, lang, s, echoLearner(lang, s))
		}
	}
	if progs["go"] == nil {
		t.Skip("no go toolchain")
	}
	for i, samples := range echoSamples {
		in, frame := echoInput(t, s, samples)
		goOut := progs["go"].run(t, frame)
		v, err := DecodeOutputFor(s, in, goOut)
		if err != nil {
			t.Fatalf("sample %d: go frame %q: %v", i, goOut, err)
		}
		if !bytes.Equal(goOut, OKFrame(v)) {
			t.Fatalf("sample %d: go frame is not OKFrame:\n got %q\nwant %q", i, goOut, OKFrame(v))
		}
		for _, lang := range []string{"cpp", "python"} {
			p := progs[lang]
			if p == nil {
				continue
			}
			if out := p.run(t, frame); !bytes.Equal(out, goOut) {
				t.Errorf("sample %d: %s fd-4 bytes differ from go:\n%s %q\ngo  %q", i, lang, lang, out, goOut)
			}
		}
	}
}

// TestCrossLanguageFloatSpelling: float64 outputs (bytes mode) are spelled exactly as Go's
// strconv.FormatFloat(v, 'g', -1, 64) in every language, over hand-picked and random values.
func TestCrossLanguageFloatSpelling(t *testing.T) {
	s := sig(t, FuncJSON, course.Signature{Mode: "function", Name: "same",
		Params: []course.Param{{Name: "xs", Type: "float64[]"}}, Returns: "float64[]"})
	learners := map[string]string{
		"go":     "package main\n\nfunc same(xs []float64) []float64 { return xs }\n",
		"cpp":    "class Solution {\npublic:\n    vector<double> same(vector<double>& xs) { return xs; }\n};\n",
		"python": "class Solution:\n    def same(self, xs):\n        return xs\n",
	}
	vals := []float64{1, -1, 0.1, 0.2, 0.3, 1.0 / 3, 2.0 / 3, 1e21, 1e20, 1e22, 1e-7, 1e-4, 1.5e-4, 9.999e-5, 123456.7,
		999999, 999999.5, 1e6, 1234567, 12345678.9, 5e-324, math.MaxFloat64, math.SmallestNonzeroFloat64 * 3,
		100, 1e15, 1e16, 1e17, 4.35, 0.000123456789, 6.02214076e23, -2.5e-8}
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 400; i++ {
		vals = append(vals, math.Float64frombits(r.Uint64()&^(0x7ff<<52)|uint64(r.IntN(2046)+1)<<52))
		vals = append(vals, float64(r.IntN(2_000_000)-1_000_000)/float64(r.IntN(999)+1))
	}
	var items []string
	for _, v := range vals {
		items = append(items, FormatFloat(v))
	}
	raw := []byte(`{"args":[[` + strings.Join(items, ",") + `]]}`)
	frame, err := EncodeInput(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	var goOut []byte
	for _, lang := range Languages {
		t.Run(lang, func(t *testing.T) {
			out := mustBuild(t, lang, s, learners[lang]).run(t, frame)
			if lang == "go" {
				goOut = out
				return
			}
			if goOut == nil {
				t.Skip("no go reference")
			}
			if !bytes.Equal(out, goOut) {
				a, b := string(out), string(goOut)
				for i := 0; i < len(a) && i < len(b); i++ {
					if a[i] != b[i] {
						lo := max(0, i-40)
						t.Fatalf("%s float spelling differs at byte %d:\n%s …%s\ngo  …%s", lang, i, lang, a[lo:min(len(a), i+40)], b[lo:min(len(b), i+40)])
					}
				}
				t.Fatalf("%s float spelling differs in length: %d vs %d", lang, len(a), len(b))
			}
		})
	}
}

// fixtureItems are m3-02's synthetic fixture items: their references in every language must
// give the same fd-4 bytes on every public sample (the cross-language compatibility golden).
func fixtureParts(t *testing.T) []fixturePart {
	t.Helper()
	root := filepath.Join("..", "..", "judge", "testdata", "content", "courses", "fixture", "items")
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var out []fixturePart
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(root, e.Name(), "item.json"))
		if err != nil {
			t.Fatal(err)
		}
		var item struct {
			Parts []struct {
				Type   string            `json:"type"`
				Config course.PartConfig `json:"config"`
			} `json:"parts"`
		}
		if err := json.Unmarshal(b, &item); err != nil {
			t.Fatal(err)
		}
		for _, p := range item.Parts {
			if p.Type != "code" {
				continue
			}
			s, err := ParseSig(p.Config.Harness, p.Config.Signature)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, fixturePart{dir: filepath.Join(root, e.Name()), sig: s, cfg: p.Config})
		}
	}
	return out
}

type fixturePart struct {
	dir string
	sig *Sig
	cfg course.PartConfig
}

// sampleInput is a public sample's case input (args, plus ops in class mode).
func sampleInput(t *testing.T, s *Sig, smp course.Sample) []byte {
	t.Helper()
	m := map[string]any{"args": smp.Args}
	if s.Class() {
		m["ops"] = smp.Ops
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := EncodeInput(s, raw)
	if err != nil {
		t.Fatal(err)
	}
	return frame
}

func TestCrossLanguageFixtureReferences(t *testing.T) {
	for _, fp := range fixtureParts(t) {
		t.Run(filepath.Base(fp.dir), func(t *testing.T) {
			outs := map[string][][]byte{}
			for _, lang := range Languages {
				if !haveLang(lang) {
					continue
				}
				src, err := os.ReadFile(filepath.Join(fp.dir, "_code", LearnerFile(lang)))
				if err != nil {
					t.Fatal(err)
				}
				p, err := buildLang(t, lang, fp.sig, string(src))
				if err != nil {
					t.Fatal(err)
				}
				for _, smp := range fp.cfg.Samples {
					outs[lang] = append(outs[lang], p.run(t, sampleInput(t, fp.sig, smp)))
				}
			}
			for i, smp := range fp.cfg.Samples {
				goOut := outs["go"][i]
				in, _ := DecodeInput(fp.sig, mustInputJSON(t, fp.sig, smp))
				want, err := DecodeExpected(fp.sig, in, smp.Expected)
				if err != nil {
					t.Fatal(err)
				}
				got, err := DecodeOutputFor(fp.sig, in, goOut)
				if err != nil {
					t.Fatalf("sample %s: %v", smp.ID, err)
				}
				if fp.cfg.Checker.Name == "exact" && !Equal(got, want) {
					t.Errorf("sample %s: go reference %s, expected %s", smp.ID, got, want)
				}
				for _, lang := range []string{"cpp", "python"} {
					if o := outs[lang]; o != nil && !bytes.Equal(o[i], goOut) {
						t.Errorf("sample %s: %s %q != go %q", smp.ID, lang, o[i], goOut)
					}
				}
			}
		})
	}
}

// TestCompatibilityGolden pins m3-02's `@1` bytes for its fixture items: every sample's fd-3
// payload and expected fd-4 payload (OKFrame). Extending the package (m3-04) must not move
// them; packs are gated against these bytes (tests.lock, contract_hash).
func TestCompatibilityGolden(t *testing.T) {
	var b strings.Builder
	for _, fp := range fixtureParts(t) {
		for _, smp := range fp.cfg.Samples {
			raw := mustInputJSON(t, fp.sig, smp)
			in3, err := CanonicalInput(fp.sig, raw)
			if err != nil {
				t.Fatal(err)
			}
			in, _ := DecodeInput(fp.sig, raw)
			want, err := DecodeExpected(fp.sig, in, smp.Expected)
			if err != nil {
				t.Fatal(err)
			}
			out4, err := SplitFrame(OKFrame(want))
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&b, "%s %s fd3 %s\n%s %s fd4 %s\n", filepath.Base(fp.dir), smp.ID, in3, filepath.Base(fp.dir), smp.ID, out4)
		}
	}
	checkGolden(t, filepath.Join("testdata", "golden", "compat-m3-02-fixtures.golden"), []byte(b.String()))
}

func mustInputJSON(t *testing.T, s *Sig, smp course.Sample) []byte {
	t.Helper()
	m := map[string]any{"args": smp.Args}
	if s.Class() {
		m["ops"] = smp.Ops
	}
	raw, _ := json.Marshal(m)
	return raw
}

// ---- panics, harness errors, bad input ----

const cppPanicLearner = `class Solution {
public:
    int boom(int k, vector<int>& xs) {
        switch (k) {
        case 0: return xs.at(10);
        case 1: { std::string s; s.reserve(s.max_size() + 1); return 0; }
        case 2: throw std::invalid_argument("x");
        case 3: throw std::bad_alloc();
        case 4: throw std::overflow_error("x");
        case 5: throw 42;
        }
        return xs.size();
    }
};
`

const pyPanicLearner = `class Solution:
    def boom(self, k, xs):
        if k == 0:
            return xs[10]
        if k == 1:
            return {}[k]
        if k == 2:
            return int("x")
        if k == 3:
            return len(5)
        if k == 4:
            return k // 0
        if k == 5:
            return self.boom(k, xs)
        if k == 6:
            return None.x
        if k == 7:
            assert False
        if k == 8:
            raise RuntimeError("x")
        return len(xs)
`

func TestCppAndPythonPanicClasses(t *testing.T) {
	s := sig(t, FuncJSON, course.Signature{Mode: "function", Name: "boom",
		Params: []course.Param{{Name: "k", Type: "int"}, {Name: "xs", Type: "int[]"}}, Returns: "int"})
	cases := map[string]struct {
		src  string
		want []string
	}{
		"cpp":    {cppPanicLearner, []string{"out_of_range", "length_error", "logic_error", "bad_alloc", "runtime_error", "other"}},
		"python": {pyPanicLearner, []string{"IndexError", "KeyError", "ValueError", "TypeError", "ZeroDivisionError", "RecursionError", "AttributeError", "AssertionError", "other"}},
	}
	for lang, c := range cases {
		t.Run(lang, func(t *testing.T) {
			p := mustBuild(t, lang, s, c.src)
			for k, cls := range c.want {
				frame, _ := EncodeInput(s, []byte(fmt.Sprintf(`{"args":[%d,[1,2]]}`, k)))
				_, err := DecodeOutput(s, p.run(t, frame))
				var pe *PanicError
				if !errors.As(err, &pe) || pe.Class != cls {
					t.Errorf("k=%d: %v, want panic %s", k, err, cls)
				}
			}
			frame, _ := EncodeInput(s, []byte(`{"args":[99,[1,2,3]]}`))
			if v, err := DecodeOutput(s, p.run(t, frame)); err != nil || !Equal(v, IntV(3)) {
				t.Errorf("ok case: %v %v", v, err)
			}
			var he *HarnessError
			if _, err := DecodeOutput(s, p.run(t, AppendFrame(nil, []byte(`{"args":[1]}`)))); !errors.As(err, &he) || he.Code != "bad_input" {
				t.Errorf("bad input: %v", err)
			}
		})
	}
}

var badResultLearners = map[string]string{
	"cpp": `class Bad {
public:
    ListNode* List() { auto a = new ListNode(1); a->next = new ListNode(2, a); return a; }
    TreeNode* Tree() { auto c = new TreeNode(2); return new TreeNode(1, c, c); }
    Node* Graph() { auto a = new Node(1); auto c = new Node(1); a->neighbors = {c}; return a; }
    double Float() { return std::numeric_limits<double>::infinity(); }
    string Text() { return string("\xff"); }
    ListNode* Many() { ListNode* h = nullptr; for (int i = 0; i <= 1000000; i++) h = new ListNode(i, h); return h; }
};
`,
	"python": `class Bad:
    def List(self):
        a = ListNode(1)
        a.next = ListNode(2, a)
        return a

    def Tree(self):
        c = TreeNode(2)
        return TreeNode(1, c, c)

    def Graph(self):
        a = Node(1)
        a.neighbors = [Node(1)]
        return a

    def Float(self):
        return float("inf")

    def Text(self):
        return "\ud800"

    def Many(self):
        h = None
        for i in range(1000001):
            h = ListNode(i, h)
        return h
`,
	"go": badResultLearner + `
func (b *Bad) Many() *ListNode {
	var h *ListNode
	for i := 0; i <= 1000000; i++ {
		h = &ListNode{Val: i, Next: h}
	}
	return h
}
`,
}

func TestHarnessErrorsEveryLanguage(t *testing.T) {
	s := sig(t, ClassOps, course.Signature{Mode: "class", Name: "Bad", Ops: []course.Op{
		{Name: "List", Returns: "ListNode"}, {Name: "Tree", Returns: "TreeNode"}, {Name: "Graph", Returns: "GraphNode"},
		{Name: "Float", Returns: "float64"}, {Name: "Text", Returns: "string"}, {Name: "Many", Returns: "ListNode"},
	}})
	want := map[string]string{"List": "cycle", "Tree": "cycle", "Graph": "invalid_graph", "Float": "non_finite", "Text": "invalid_utf8", "Many": "too_many_nodes"}
	for _, lang := range Languages {
		t.Run(lang, func(t *testing.T) {
			p := mustBuild(t, lang, s, badResultLearners[lang])
			for op, code := range want {
				frame, err := EncodeInput(s, []byte(`{"ops":["Bad","`+op+`"],"args":[[],[]]}`))
				if err != nil {
					t.Fatal(err)
				}
				var he *HarnessError
				if _, err := DecodeOutput(s, p.run(t, frame)); !errors.As(err, &he) || he.Code != code {
					t.Errorf("%s: %v, want error %s", op, err, code)
				}
			}
		})
	}
}

// TestCppMissingFunctionFailsInHarnessFile: a learner file without the method, or with a
// wrong signature, fails to compile in zz_xl_harness.cpp (judge replaces that diagnostic
// with its fixed CE text, m3-06).
func TestCppMissingFunctionFailsInHarnessFile(t *testing.T) {
	s := funcSig(t)
	for name, src := range map[string]string{
		"missing":   "class Solution {\npublic:\n    vector<int> pairSums(vector<int>& n, int t) { return {}; }\n};\n",
		"wrongType": "class Solution {\npublic:\n    vector<int> pairSum(vector<string>& n, int t) { return {}; }\n};\n",
	} {
		_, err := buildLang(t, "cpp", s, src)
		if err == nil || !strings.Contains(err.Error(), CppHarnessFile) {
			t.Errorf("%s: want a compile error in %s, got %v", name, CppHarnessFile, err)
		}
	}
}

// ---- starters ----

func TestStartersCompileAndFailSamples(t *testing.T) {
	for _, fp := range fixtureParts(t) {
		for _, lang := range Languages {
			t.Run(filepath.Base(fp.dir)+"/"+lang, func(t *testing.T) {
				st, err := Starter(lang, fp.sig.Harness, fp.sig)
				if err != nil {
					t.Fatal(err)
				}
				if st.Path != LearnerFile(lang) {
					t.Fatalf("starter path %s", st.Path)
				}
				p, err := buildLang(t, lang, fp.sig, string(st.Data))
				if err != nil {
					t.Fatalf("the starter must compile with the harness: %v\n%s", err, st.Data)
				}
				passed := 0
				for _, smp := range fp.cfg.Samples {
					in, _ := DecodeInput(fp.sig, mustInputJSON(t, fp.sig, smp))
					want, _ := DecodeExpected(fp.sig, in, smp.Expected)
					if got, err := DecodeOutputFor(fp.sig, in, p.run(t, sampleInput(t, fp.sig, smp))); err == nil && Equal(got, want) {
						passed++
					}
				}
				if passed == len(fp.cfg.Samples) {
					t.Errorf("the starter passes every sample:\n%s", st.Data)
				}
			})
		}
	}
}

func TestStarterGolden(t *testing.T) {
	for name, s := range map[string]*Sig{"func-json": funcSig(t), "class-ops": classSig(t)} {
		for _, lang := range Languages {
			st, err := Starter(lang, s.Harness, s)
			if err != nil {
				t.Fatal(err)
			}
			checkGolden(t, filepath.Join("testdata", "golden", "starter-"+name+"."+lang+".golden"), st.Data)
		}
	}
}

// TestGenerateGoldenCppPython pins the generated C++ and Python harness sources (a source
// change that keeps the frame bytes refreshes them with -update; a byte change is @2).
func TestGenerateGoldenCppPython(t *testing.T) {
	for name, s := range map[string]*Sig{"func-json": funcSig(t), "class-ops": classSig(t), "nodes": nodeSig(t)} {
		for _, lang := range []string{"cpp", "python"} {
			files, err := Generate(lang, s.Harness, s)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 2 {
				t.Fatalf("%s %s: files %v", name, lang, files)
			}
			checkGolden(t, filepath.Join("testdata", "golden", name+"."+lang+".golden"), files[0].Data)
			if want := Sources(lang)[lang+"/"+files[1].Path]; !bytes.Equal(files[1].Data, want) {
				t.Errorf("%s %s: the prelude is not the static template file", name, lang)
			}
		}
	}
}

func nodeSig(t *testing.T) *Sig {
	return sig(t, FuncJSON, course.Signature{Mode: "function", Name: "mix",
		Params:  []course.Param{{Name: "head", Type: "ListNode"}, {Name: "root", Type: "TreeNode"}, {Name: "g", Type: "GraphNode"}},
		Returns: "TreeNode[]"})
}

func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test -update)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs (a byte change on fd 3/fd 4 is @2; otherwise go test -update)", path)
	}
}

func TestCheckNames(t *testing.T) {
	bad := []struct {
		lang string
		s    course.Signature
	}{
		{"cpp", course.Signature{Mode: "function", Name: "delete", Returns: "int"}},
		{"python", course.Signature{Mode: "function", Name: "lambda", Returns: "int"}},
		{"go", course.Signature{Mode: "function", Name: "main", Returns: "int"}},
		{"go", course.Signature{Mode: "function", Name: "xlRun", Returns: "int"}},
		{"python", course.Signature{Mode: "function", Name: "f", Params: []course.Param{{Name: "self", Type: "int"}}, Returns: "int"}},
		{"cpp", course.Signature{Mode: "class", Name: "Node", Ops: []course.Op{{Name: "get", Returns: "int"}}}},
	}
	for _, c := range bad {
		h := FuncJSON
		if c.s.Mode == "class" {
			h = ClassOps
		}
		s := sig(t, h, c.s)
		if _, err := Generate(c.lang, h, s); err == nil {
			t.Errorf("%s %s: want a refusal", c.lang, c.s.Name)
		}
	}
	if _, err := Generate("python", FuncJSON, sig(t, FuncJSON, course.Signature{Mode: "function", Name: "xlen", Returns: "int"})); err == nil {
		t.Error("xlen starts with the xl prefix")
	}
	_ = strconv.Itoa
}
