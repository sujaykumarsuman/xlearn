package packspec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/packspec/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// The pipeline over the SYNTHETIC fixture item fx-001 (internal/judge/testdata) with a fake
// executor: programs are Go functions here, so these tests need no docker. The real
// executor runs in the pack-fixture CI job.

func specOf(name string) *gen.Spec { return &gen.Spec{Gen: name, Params: json.RawMessage(`{}`)} }

const (
	fixtureContent = "../judge/testdata/content"
	fixturePackSrc = "../judge/testdata/packsrc"
	fixturePack    = "../judge/testdata/pack"
)

// fakeProg is one program's behaviour.
type fakeProg struct {
	// call answers one harness case (canSplit's result, or killed / crashed).
	call      func(nums []int64) (ok bool, killed bool, exit int)
	stdio     func(args []string, stdin []byte) ([]byte, int)
	noCompile bool
	cpu       int64
	peak      int64
}

type fakeExec struct {
	t     *testing.T
	sig   *harness.Sig
	progs map[string]fakeProg // by the label's last word (a path)
	calls []string
}

func (f *fakeExec) Images() map[string]string { return map[string]string{"go": "golang@sha256:fake"} }

func (f *fakeExec) SyntaxCheck(_ context.Context, lang string, files []SourceFile) (bool, string, error) {
	return true, "", nil
}

func (f *fakeExec) Execute(_ context.Context, p Program, runs []Run, lim Limits) (*ExecResult, error) {
	f.calls = append(f.calls, p.Label)
	key := p.Label[strings.LastIndex(p.Label, " ")+1:]
	prog, ok := f.progs[key]
	if !ok {
		f.t.Fatalf("unexpected program %q", p.Label)
	}
	res := &ExecResult{CompileOK: !prog.noCompile, BaselineKB: 1000}
	if prog.noCompile {
		return res, nil
	}
	killed := false
	for _, r := range runs {
		if killed && lim.StopAfterKill {
			res.Runs = append(res.Runs, RunResult{Skipped: true})
			continue
		}
		rr := RunResult{CPUms: max(prog.cpu, 1), PeakKB: max(prog.peak, 2000)}
		switch p.Mode {
		case ModeStdio:
			rr.Output, rr.Exit = prog.stdio(r.Args, r.Input)
		default:
			payload, err := harness.SplitFrame(r.Input)
			if err != nil {
				f.t.Fatal(err)
			}
			in, err := harness.DecodeInput(f.sig, payload)
			if err != nil {
				f.t.Fatal(err)
			}
			var nums []int64
			for _, v := range in.Args[0].List {
				nums = append(nums, v.Int)
			}
			ans, k, exit := prog.call(nums)
			rr.Killed, rr.Exit = k, exit
			if k {
				rr.CPUms = lim.CPUms + 200
			}
			if !k && exit == 0 {
				rr.Output = harness.OKFrame(harness.BoolV(ans))
			}
		}
		killed = killed || rr.Killed
		res.Runs = append(res.Runs, rr)
	}
	return res, nil
}

func canSplit(nums []int64) bool {
	var total, left int64
	for _, x := range nums {
		total += x
	}
	for i := 0; i < len(nums)-1; i++ {
		left += nums[i]
		if 2*left == total {
			return true
		}
	}
	return false
}

func newFake(t *testing.T, sig *harness.Sig) *fakeExec {
	return &fakeExec{t: t, sig: sig, progs: map[string]fakeProg{
		"gen/random.go": {stdio: func(args []string, _ []byte) ([]byte, int) {
			seed, _ := strconv.ParseUint(args[1], 10, 64)
			a, b := int64(seed%7)-3, int64(seed%5)
			return []byte(fmt.Sprintf(`{"args":[[%d,%d,%d]]}`+"\n", a, b, a+b)), 0
		}},
		"_code/solution.go":               {call: func(n []int64) (bool, bool, int) { return canSplit(n), false, 0 }},
		"submissions/brute.go":            {call: func(n []int64) (bool, bool, int) { return canSplit(n), false, 0 }},
		"submissions/wrong/empty-part.go": {call: func(n []int64) (bool, bool, int) { return canSplit(n) || len(n) == 2 && n[0] == -n[1], false, 0 }},
		"submissions/wrong/quadratic.go":  {call: func(n []int64) (bool, bool, int) { return canSplit(n), len(n) > 1000, 0 }},
		"submissions/brute.cpp":           {call: func(n []int64) (bool, bool, int) { return canSplit(n), false, 0 }},
		"submissions/wrong/no-compile.go": {noCompile: true},
		"submissions/wrong/crash.go":      {call: func(n []int64) (bool, bool, int) { return false, false, 2 }},
	}}
}

// fixtureSource copies fx-001's pack source to a temp pack and resolves it.
func fixtureSource(t *testing.T, edit func(dir string)) (*ItemSource, *fakeExec) {
	t.Helper()
	pack := t.TempDir()
	src := filepath.Join(fixturePackSrc, "courses", "fixture", "items", "fx-001")
	dst := filepath.Join(pack, "courses", "fixture", "items", "fx-001")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(dst, TimingFile))
	if edit != nil {
		edit(dst)
	}
	c, err := curriculum.LoadContent(os.DirFS(fixtureContent))
	if err != nil {
		t.Fatal(err)
	}
	var pub = &c.Courses[0].Items[0]
	if pub.Item.ID != "fx-001" {
		t.Fatalf("fixture order: %s", pub.Item.ID)
	}
	s, err := LoadItemSource(pack, "fixture", "fx-001", pub)
	if err != nil {
		t.Fatal(err)
	}
	return s, newFake(t, s.Sig)
}

func editJSON(t *testing.T, file string, fn func(m map[string]any)) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	fn(m)
	out, _ := json.Marshal(m)
	if err := os.WriteFile(file, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineGates(t *testing.T) {
	s, ex := fixtureSource(t, nil)
	ctx := context.Background()
	m, err := s.Materialize(ctx, ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Cases) != 18 || m.Cases[0].Group() != GroupEdge || m.Cases[17].Group() != GroupPerf {
		t.Fatalf("cases: %d", len(m.Cases))
	}
	// The perf cases' expected is the digest of the canonical frame (exact + bool).
	var digest struct {
		Bytes  int    `json:"bytes"`
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(m.Cases[17].Expected, &digest); err != nil || digest.Bytes == 0 || len(digest.SHA256) != 64 {
		t.Fatalf("perf expected %s", m.Cases[17].Expected)
	}
	// Deterministic: a second materialization gives the same lock entries.
	e1, _ := m.Entries()
	m2, _ := s.Materialize(ctx, ex)
	e2, _ := m2.Entries()
	if d := Diff(e1, e2); !d.Empty() {
		t.Fatalf("materialization is not deterministic: %s", d)
	}

	if rep, err := s.Validate(ctx, ex, m.Cases); err != nil || rep.Invalid != 4 {
		t.Fatalf("Validate: %+v %v", rep, err)
	}
	if g := m.Oracle(ctx, ex); len(g) != 1 || g[0].Status != Pass {
		t.Fatalf("oracle: %+v", g)
	}
	wrong := m.WrongSolutions(ctx, ex)
	for _, w := range wrong {
		if w.Gate.Status != Pass {
			t.Errorf("wrong %s: %+v", w.Wrong.File, w.Gate)
		}
	}
	if w := wrong[1]; w.Outcome.Verdict != TLE || !w.Outcome.PerfTLE {
		t.Errorf("quadratic: %+v", w.Outcome)
	}
	g, timing := m.TimeLimits(wrong, "golang@sha256:fake")
	if g.Status != Pass || !timing.Provisional || timing.TLFloorMS != 1000 || timing.SumTLms != 18000 {
		t.Fatalf("tl: %+v %+v", g, timing)
	}
	if err := m.WriteTiming(timing); err != nil {
		t.Fatal(err)
	}

	// Build → manifest → listing: only allowed files, and the hashes verify.
	out := filepath.Join(t.TempDir(), "build")
	man, err := Build([]BuiltItem{{Materialized: m, GraderKinds: []string{"code"}}}, BuildOptions{Out: out, Version: "0.1.0", ValidatedAgainst: strings.Repeat("0", 40), Dockerfile: true})
	if err != nil {
		t.Fatal(err)
	}
	if bad := man.VerifyFiles(os.DirFS(out)); len(bad) != 0 {
		t.Fatalf("VerifyFiles: %v", bad)
	}
	l, _ := ListDir(out)
	if errs := CheckListing(l); len(errs) != 0 {
		t.Fatalf("listing: %v", errs)
	}
	if _, err := Build(nil, BuildOptions{Out: out, Version: "0.1.0", ValidatedAgainst: strings.Repeat("0", 40)}); err == nil {
		t.Fatal("Build into a non-empty --out")
	}
}

func TestWrongExpectFails(t *testing.T) {
	s, ex := fixtureSource(t, func(dir string) {
		editJSON(t, filepath.Join(dir, ItemFile), func(m map[string]any) {
			w := m["wrong"].([]any)
			w[0].(map[string]any)["expect"] = "TLE" // empty-part.go actually gets WA
			m["wrong"] = append(w,
				map[string]any{"file": "submissions/wrong/no-compile.go", "expect": "CE"},
				map[string]any{"file": "submissions/wrong/crash.go", "expect": "RE"},
				map[string]any{"file": "submissions/wrong/x.cpp", "expect": "WA"},
				map[string]any{"file": "submissions/wrong/lint.go", "expect": "REJECTED"})
		})
		for _, f := range []string{"no-compile.go", "crash.go", "x.cpp", "lint.go"} {
			os.WriteFile(filepath.Join(dir, "submissions", "wrong", f), []byte("package main\n"), 0o644)
		}
	})
	ctx := context.Background()
	m, err := s.Materialize(ctx, ex)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, w := range m.WrongSolutions(ctx, ex) {
		got[w.Wrong.File] = w.Gate.Status + " " + w.Gate.Detail
	}
	want := map[string]string{
		"submissions/wrong/empty-part.go": "fail got WA, declared TLE",
		"submissions/wrong/no-compile.go": "pass CE",
		"submissions/wrong/crash.go":      "pass RE",
		"submissions/wrong/x.cpp":         "pending",
		"submissions/wrong/lint.go":       "pending",
	}
	for f, w := range want {
		if !strings.HasPrefix(got[f], w) {
			t.Errorf("%s: %q, want prefix %q", f, got[f], w)
		}
	}
}

func TestTimeLimitRules(t *testing.T) {
	s, ex := fixtureSource(t, nil)
	ex.progs["_code/solution.go"] = fakeProg{call: func(n []int64) (bool, bool, int) { return canSplit(n), false, 0 }, cpu: 400, peak: 40_000}
	m, err := s.Materialize(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	g, timing := m.TimeLimits(nil, "img")
	if g.Status != Fail || timing.TLFloorMS != 1200 || !strings.Contains(g.Detail, "time_ms 1000 < floor 1200") || !strings.Contains(g.Detail, "memory_mb 64 < floor") {
		t.Fatalf("tl: %+v %+v", g, timing)
	}
	// A TLE wrong solution that never exceeds the TL on a perf case.
	wrong := []WrongResult{{Wrong: Wrong{File: "submissions/wrong/q.go", Expect: TLE}, Gate: GateResult{Status: Pass}, Outcome: &Outcome{Verdict: TLE}}}
	if g, _ := m.TimeLimits(wrong, "img"); !strings.Contains(g.Detail, "never exceeds the TL on a perf case") {
		t.Fatalf("tl perf rule: %+v", g)
	}
}

func TestInputErrors(t *testing.T) {
	ctx := context.Background()
	// An invalid line every validator accepts.
	s, ex := fixtureSource(t, func(dir string) {
		os.WriteFile(filepath.Join(dir, "invalid", "accepted.jsonl"), []byte(`{"args": [[1, 1]]}`+"\n"), 0o644)
	})
	cases, err := s.Inputs(ctx, ex)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Validate(ctx, ex, cases); err == nil || !strings.Contains(err.Error(), "accepted by every validator") {
		t.Fatalf("Validate: %v", err)
	}
	// A constraint-breaking edge case.
	s, ex = fixtureSource(t, func(dir string) {
		os.WriteFile(filepath.Join(dir, EdgeFile), []byte(`{"args": [[1, 5000]]}`+"\n"), 0o644)
	})
	cases, _ = s.Inputs(ctx, ex)
	if _, err := s.Validate(ctx, ex, cases); err == nil || !strings.Contains(err.Error(), "nums[*] range") {
		t.Fatalf("Validate: %v", err)
	}
	// Expected outputs are never hand-written; duplicates and oversize literals fail.
	for name, line := range map[string]string{
		"expected":  `{"args": [[1, 1]], "expected": true}`,
		"duplicate": `{"args": [[1, 1]]}` + "\n" + `{"args": [[1,1]]}`,
		"oversize":  `{"args": [[` + strings.Repeat("1,", 140_000) + `1]]}`,
		"group":     `{"args": [[1, 1]], "tags": ["perf"]}`,
	} {
		s, ex := fixtureSource(t, func(dir string) {
			os.WriteFile(filepath.Join(dir, EdgeFile), []byte(line+"\n"), 0o644)
		})
		if _, err := s.Inputs(ctx, ex); err == nil {
			t.Errorf("%s: Inputs accepted it", name)
		}
	}
	// No public Go reference: lock fails "no public Go reference".
	c, err := curriculum.LoadContent(os.DirFS(fixtureContent))
	if err != nil {
		t.Fatal(err)
	}
	pub := c.Courses[0].Items[0]
	delete(pub.References, "go")
	pub.References = map[string]string{"py": "x"}
	if _, err := LoadItemSource(fixturePackSrc, "fixture", "fx-001", &pub); err == nil || !strings.Contains(err.Error(), "no public Go reference") {
		t.Fatalf("LoadItemSource: %v", err)
	}
}

// A gate line never has the "file.go: message" shape CI problem matchers turn into errors.
func TestGateLineShape(t *testing.T) {
	g := GateResult{Item: "fixture/fx-001", Gate: "oracle", Target: "submissions/brute.go", Status: Pass, Detail: "16 small cases agree"}
	if s := g.String(); strings.Contains(s, ".go:") || s != "fixture/fx-001 oracle submissions/brute.go — pass (16 small cases agree)" {
		t.Fatalf("gate line %q", s)
	}
}
