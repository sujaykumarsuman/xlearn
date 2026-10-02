//go:build linux && runner_it

package profile_test

// The launch profiles through the real jail (m3-04 task 6; build tags linux && runner_it, the
// runner-it lane). Every item, case and solution is SYNTHETIC (internal/runner/testdata/items,
// m3-02's internal/judge/testdata fixtures and the lint's evasion fixtures); nothing comes from
// xlearn-evalpack.
//
//   - references pass every sample and hidden case in Go, C++ and Python; outputs are compared
//     through internal/platform/checker (no second comparator) and, for every non-float item,
//     the C++ and Python fd-4 frames are byte-identical to Go's;
//   - wrong solutions are classified WA / TLE / RE / CE / MLE / REJECTED by a test-only term →
//     class mapper (t4 §4.1; m3-06 owns the production one and reuses these fixtures);
//   - the lint-evasion fixtures are stopped by the jail (SIGSYS → RE, counted, rotation);
//   - the artifacts run as the job UID with their ArtifactMode; GET /v1/profiles serves the
//     profiles; the Go compile uses the read-only GOCACHE seed in place and rebuilds no std;
//   - provisional TL multipliers are computed from the perf cases (RUNNER-IT-METRIC lines).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/checker"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi/lint"
	"github.com/sujaykumarsuman/xlearn/internal/runner/it/itrt"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	goprofile "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile/proftest"
)

func TestMain(m *testing.M) { os.Exit(itrt.Main(m)) }

var (
	langs = []string{"go", "cpp", "python"}
	exts  = map[string]string{"go": "go", "cpp": "cpp", "python": "py"}
	// refTL is each language's CPU TL for the references (generous; the multipliers are
	// measured, not enforced here). Wrong solutions run at 1 s.
	refTL = map[string]int64{"go": 2000, "cpp": 2000, "python": 10000}
)

func itemsDir() string {
	return filepath.Join(itrt.Env.Root, "internal", "runner", "testdata", "items")
}

// ---- items and cases ----

type tcase struct {
	id    string
	group runnerapi.Group
	in    *harness.Input
	frame []byte
	want  *harness.Value // nil: the Go reference's output is the expected one
}

type titem struct {
	slug  string
	dir   string
	sig   *harness.Sig
	cfg   course.PartConfig
	check checker.Checker
	cases []tcase
}

func loadItem(t *testing.T, dir string, hidden bool) *titem {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "item.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ID    string `json:"id"`
		Parts []struct {
			Type   string            `json:"type"`
			Config course.PartConfig `json:"config"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	it := &titem{slug: doc.ID, dir: dir}
	for _, p := range doc.Parts {
		if p.Type == "code" {
			it.cfg = p.Config
		}
	}
	if it.sig, err = harness.ParseSig(it.cfg.Harness, it.cfg.Signature); err != nil {
		t.Fatal(err)
	}
	if it.check, err = checker.Lookup(it.cfg.Checker); err != nil {
		t.Fatal(err)
	}
	for _, s := range it.cfg.Samples {
		it.add(t, s.ID, runnerapi.GroupSample, s.Ops, s.Args, s.Expected)
	}
	if !hidden {
		return it
	}
	f, err := os.Open(filepath.Join(dir, "cases.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var c struct {
			ID       string            `json:"id"`
			Group    runnerapi.Group   `json:"group"`
			Ops      []string          `json:"ops"`
			Args     []json.RawMessage `json:"args"`
			Expected json.RawMessage   `json:"expected"`
		}
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatalf("%s cases.jsonl: %v", it.slug, err)
		}
		it.add(t, c.ID, c.Group, c.Ops, c.Args, c.Expected)
	}
	for i, raw := range proftest.PerfCases(it.slug) {
		it.addRaw(t, "p"+strconv.Itoa(i+1), runnerapi.GroupPerf, raw, nil)
	}
	return it
}

func (it *titem) add(t *testing.T, id string, g runnerapi.Group, ops []string, args []json.RawMessage, expected json.RawMessage) {
	t.Helper()
	m := map[string]any{"args": args}
	if it.sig.Class() {
		m["ops"] = ops
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	it.addRaw(t, id, g, raw, expected)
}

func (it *titem) addRaw(t *testing.T, id string, g runnerapi.Group, raw, expected []byte) {
	t.Helper()
	in, err := harness.DecodeInput(it.sig, raw)
	if err != nil {
		t.Fatalf("%s %s: %v", it.slug, id, err)
	}
	frame, err := harness.EncodeInput(it.sig, raw)
	if err != nil {
		t.Fatal(err)
	}
	c := tcase{id: id, group: g, in: in, frame: frame}
	if len(expected) > 0 {
		v, err := harness.DecodeExpected(it.sig, in, expected)
		if err != nil {
			t.Fatalf("%s %s expected: %v", it.slug, id, err)
		}
		c.want = &v
	}
	it.cases = append(it.cases, c)
}

func syntheticItems(t *testing.T) []*titem {
	t.Helper()
	var out []*titem
	for _, slug := range []string{"pair-sum", "group-words", "reverse-list", "level-order", "clone-graph", "running-median", "min-stack"} {
		out = append(out, loadItem(t, filepath.Join(itemsDir(), slug), true))
	}
	return out
}

// fixtureItems are m3-02's synthetic fixture items (samples only; their references are in
// _code/): reused here so the pack fixture's three languages also run through the jail.
func fixtureItems(t *testing.T) []*titem {
	t.Helper()
	root := filepath.Join(itrt.Env.Root, "internal", "judge", "testdata", "content", "courses", "fixture", "items")
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var out []*titem
	for _, e := range ents {
		out = append(out, loadItem(t, filepath.Join(root, e.Name()), false))
	}
	return out
}

// ---- jobs ----

var jobN int

func newJob(t *testing.T, it *titem, lang string, learner []byte, cpuMs int64, stop map[runnerapi.Group]runnerapi.StopRule) (*runnerapi.Job, [][]byte) {
	t.Helper()
	p, ok := profile.ForLanguage(lang)
	if !ok {
		t.Fatalf("no profile for %s", lang)
	}
	gen, err := harness.Generate(lang, it.sig.Harness, it.sig)
	if err != nil {
		t.Fatal(err)
	}
	jobN++
	job := &runnerapi.Job{
		ID: fmt.Sprintf("refs-%s-%s-%d", it.slug, lang, jobN), Profile: p.Name, Harness: it.sig.Harness, Mode: runnerapi.ModeSubmit,
		Files:       append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: learner}}, gen...),
		Limits:      runnerapi.Limits{Case: runnerapi.CaseLimits{CPUms: cpuMs, MemMB: 256, OutputKB: 8192}},
		OutputMode:  runnerapi.OutputBytes,
		StopGroupOn: stop,
	}
	var inputs [][]byte
	for _, c := range it.cases {
		job.Cases = append(job.Cases, runnerapi.CaseInput{OpaqueID: c.id, Group: c.group, Size: int64(len(c.frame))})
		inputs = append(inputs, c.frame)
	}
	return job, inputs
}

// classify is the TEST-ONLY term → class mapper (t4 §4.1's verdict table, precedence TLE >
// MLE > OLE > RE > WA): the runner's terms first, then the fd-4 frame ({"panic"} → RE, a
// harness error → WA), then the item's checker against want.
func classify(t *testing.T, it *titem, c tcase, cr runnerapi.CaseResult, want *harness.Value) (string, harness.Value) {
	t.Helper()
	switch cr.Term {
	case runnerapi.TermTLE:
		return "TLE", harness.Value{}
	case runnerapi.TermMLE:
		return "MLE", harness.Value{}
	case runnerapi.TermOLE:
		return "OLE", harness.Value{}
	case runnerapi.TermSignal, runnerapi.TermExitNonzero:
		return "RE", harness.Value{}
	case runnerapi.TermNotRun:
		return "NOT_RUN", harness.Value{}
	}
	v, err := harness.DecodeOutputFor(it.sig, c.in, cr.Output)
	var pe *harness.PanicError
	var he *harness.HarnessError
	switch {
	case errors.As(err, &pe):
		if cr.PanicClass != pe.Class {
			t.Errorf("%s %s: CaseResult.PanicClass %q, frame %q", it.slug, c.id, cr.PanicClass, pe.Class)
		}
		return "RE", v
	case errors.As(err, &he) && he.Code == "bad_input":
		t.Fatalf("%s %s: bad_input is a judge-side fault", it.slug, c.id)
	case err != nil:
		return "WA", v
	}
	if want != nil && !it.check.Check(v, *want).OK {
		return "WA", v
	}
	return "AC", v
}

// ---- references ----

func TestReferencesPassInEveryLanguage(t *testing.T) {
	items := append(syntheticItems(t), fixtureItems(t)...)
	cpu := map[string]map[string]int64{} // lang → item/case → CPU ms (perf cases)
	for _, it := range items {
		t.Run(it.slug, func(t *testing.T) {
			goOut := map[string][]byte{}
			goVal := map[string]harness.Value{}
			for _, lang := range langs {
				src := filepath.Join(it.dir, "refs", "solution."+exts[lang])
				if _, err := os.Stat(src); err != nil {
					src = filepath.Join(it.dir, "_code", "solution."+exts[lang]) // m3-02's fixtures
				}
				learner, err := os.ReadFile(src)
				if err != nil {
					t.Fatal(err)
				}
				if vs := lint.Check(lint.Language(lang), append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: learner}}, mustGen(t, it, lang)...)); len(vs) > 0 {
					t.Errorf("%s reference is REJECTED by the lint: %v", lang, vs)
				}
				itrt.Ensure(t)
				job, in := newJob(t, it, lang, learner, refTL[lang], nil)
				res := itrt.Run(t, job, in)
				if !res.Compile.OK {
					t.Fatalf("%s reference: CE %+v", lang, res.Compile)
				}
				if res.Versions.Profile != job.Profile || len(res.Versions.ProfileSHA) != 64 {
					t.Errorf("%s versions %+v", lang, res.Versions)
				}
				for i, c := range it.cases {
					cr := res.Cases[i]
					want := c.want
					if want == nil && lang != "go" {
						w := goVal[c.id]
						want = &w
					}
					cls, v := classify(t, it, c, cr, want)
					if cls != "AC" {
						t.Errorf("%s %s (%s): %s (term %s %s, %d ms, stderr %q)", lang, c.id, c.group, cls, cr.Term, cr.Signal, cr.CPUms, cr.Stderr)
						continue
					}
					if lang == "go" {
						goOut[c.id], goVal[c.id] = cr.Output, v
					} else if harness.CanonicalizableOutput(it.sig) && !bytes.Equal(cr.Output, goOut[c.id]) {
						t.Errorf("%s %s: fd-4 bytes differ from Go's (%d vs %d bytes)", lang, c.id, len(cr.Output), len(goOut[c.id]))
					}
					if c.group == runnerapi.GroupPerf {
						if cpu[lang] == nil {
							cpu[lang] = map[string]int64{}
						}
						cpu[lang][it.slug+"/"+c.id] = cr.CPUms
						itrt.Metric(t, fmt.Sprintf("perf_cpu_ms_%s_%s_%s", lang, it.slug, c.id), cr.CPUms)
					}
				}
				itrt.AssertClean(t)
			}
		})
	}
	// Provisional TL multipliers (task 7): per perf case CPU(lang) / CPU(go), the max rounded
	// up to the next 0.5, floor 1.0. The Go CPU is floored at 10 ms so a millisecond-level Go
	// case can't make the ratio meaningless.
	for _, lang := range []string{"cpp", "python"} {
		maxRatio := 0.0
		for k, ms := range cpu[lang] {
			g := max(cpu["go"][k], 10)
			r := float64(ms) / float64(g)
			itrt.Metric(t, fmt.Sprintf("tl_ratio_%s_%s", lang, strings.ReplaceAll(k, "/", "_")), fmt.Sprintf("%.2f", r))
			maxRatio = max(maxRatio, r)
		}
		mult := max(1.0, math.Ceil(maxRatio*2)/2)
		itrt.Metric(t, "tl_multiplier_"+lang, mult)
	}
}

func mustGen(t *testing.T, it *titem, lang string) []runnerapi.File {
	t.Helper()
	gen, err := harness.Generate(lang, it.sig.Harness, it.sig)
	if err != nil {
		t.Fatal(err)
	}
	return gen
}

// ---- wrong solutions ----

var wantRE = regexp.MustCompile(`^(?://|#) want: ([A-Z]+)(?: line=(\d+))?(?: (file=harness))?`)

var harnessFile = map[string]string{"go": harness.GoHarnessFile, "cpp": harness.CppHarnessFile, "python": harness.PythonMainFile}

var wrongStop = map[runnerapi.Group]runnerapi.StopRule{
	runnerapi.GroupSample: runnerapi.StopAnyFail, runnerapi.GroupEdge: runnerapi.StopAnyFail,
	runnerapi.GroupRandom: runnerapi.StopAnyFail, runnerapi.GroupPerf: runnerapi.StopTLE,
}

func TestWrongSolutionsAreClassified(t *testing.T) {
	seen := map[string]map[string]bool{}
	for _, slug := range []string{"pair-sum", "reverse-list"} {
		it := loadItem(t, filepath.Join(itemsDir(), slug), true)
		// The Go reference defines the expected outputs of the generated cases.
		ref, _ := os.ReadFile(filepath.Join(it.dir, "refs", "solution.go"))
		itrt.Ensure(t)
		job, in := newJob(t, it, "go", ref, refTL["go"], nil)
		res := itrt.Run(t, job, in)
		for i, c := range it.cases {
			if c.want == nil {
				_, v := classify(t, it, c, res.Cases[i], nil)
				it.cases[i].want = &v
			}
		}
		paths, _ := filepath.Glob(filepath.Join(it.dir, "wrong", "*"))
		for _, p := range paths {
			lang := map[string]string{".go": "go", ".cpp": "cpp", ".py": "python"}[filepath.Ext(p)]
			t.Run(slug+"/"+filepath.Base(p), func(t *testing.T) {
				src, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				m := wantRE.FindStringSubmatch(string(src))
				if m == nil {
					t.Fatalf("no want: header")
				}
				want := m[1]
				if seen[want] == nil {
					seen[want] = map[string]bool{}
				}
				seen[want][lang] = true
				files := append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: src}}, mustGen(t, it, lang)...)
				vs := lint.Check(lint.Language(lang), files)
				if want == "REJECTED" {
					if len(vs) == 0 {
						t.Error("a lint violation must be REJECTED before enqueue")
					}
					return
				}
				if len(vs) > 0 {
					t.Fatalf("unexpected lint violations: %v", vs)
				}
				itrt.Ensure(t)
				job, in := newJob(t, it, lang, src, 1000, wrongStop)
				res := itrt.Run(t, job, in)
				if want == "CE" {
					if res.Compile.OK || len(res.Compile.Diags) == 0 {
						t.Fatalf("want a CE with diagnostics, got %+v", res.Compile)
					}
					d := res.Compile.Diags[0]
					wantFile := harness.LearnerFile(lang)
					if m[3] != "" {
						wantFile = harnessFile[lang]
					}
					if d.File != wantFile || (m[2] != "" && strconv.Itoa(d.Line) != m[2]) {
						t.Errorf("diagnostic %+v, want %s line %s", d, wantFile, m[2])
					}
					itrt.Metric(t, "ce_diag_"+lang+"_"+filepath.Base(p), fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Col))
					return
				}
				if !res.Compile.OK {
					t.Fatalf("compile failed: %+v", res.Compile)
				}
				got := ""
				for i, c := range it.cases {
					cls, _ := classify(t, it, c, res.Cases[i], c.want)
					if cls != "AC" && cls != "NOT_RUN" {
						got = cls
						t.Logf("first failing case %s (%s): %s term=%s signal=%s cpu=%dms panic=%q", c.id, c.group, cls, res.Cases[i].Term, res.Cases[i].Signal, res.Cases[i].CPUms, res.Cases[i].PanicClass)
						break
					}
				}
				if got != want {
					t.Errorf("verdict %q, want %q", got, want)
				}
				itrt.AssertClean(t)
			})
		}
	}
	for _, cls := range []string{"WA", "TLE", "RE", "CE", "MLE", "REJECTED"} {
		for _, lang := range langs {
			if !seen[cls][lang] {
				t.Errorf("no %s wrong solution in %s", cls, lang)
			}
		}
	}
}

// TestEvasionStoppedByJail: each language's lint-evasion fixture (the lint's own testdata)
// runs and the jail stops it: socket() (or, in Python, the socket module's own setup) is
// outside the allowlist and KILLed by the exec filter → signal(SIGSYS) → RE, counted, and the
// runner rotates. Go has no lint-clean path to a syscall, so its fixture is posted past the
// lint.
func TestEvasionStoppedByJail(t *testing.T) {
	it := loadItem(t, filepath.Join(itemsDir(), "pair-sum"), false)
	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(itrt.Env.Root, "internal", "platform", "runnerapi", "lint", "testdata", lang, "evasion."+exts[lang]))
			if err != nil {
				t.Fatal(err)
			}
			files := append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: src}}, mustGen(t, it, lang)...)
			if vs := lint.Check(lint.Language(lang), files); (lang == "go") != (len(vs) > 0) {
				t.Errorf("lint on the %s evasion fixture: %v", lang, vs)
			}
			p := itrt.Ensure(t)
			job, in := newJob(t, it, lang, src, 1000, wrongStop)
			res := itrt.Run(t, job, in)
			if !res.Compile.OK {
				t.Fatalf("compile: %+v", res.Compile)
			}
			cr := res.Cases[0]
			if cr.Term != runnerapi.TermSignal || cr.Signal != "SIGSYS" {
				t.Fatalf("the jail must stop it with SIGSYS: %+v", cr)
			}
			if cls, _ := classify(t, it, it.cases[0], cr, it.cases[0].want); cls != "RE" {
				t.Errorf("class %s, want RE", cls)
			}
			if code := p.WaitExit(t, 60*time.Second); code != 0 {
				t.Errorf("the runner rotates after a SIGSYS: exit %d", code)
			}
		})
	}
}

// ---- artifacts, profiles, names ----

// TestArtifactModes: each profile's artifact runs as the job UID with its ArtifactMode (0111
// for the Go and C++ static ELF, 0444 for Python's zipapp), here with the generated starter,
// which compiles with the harness and fails the samples.
func TestArtifactModes(t *testing.T) {
	it := loadItem(t, filepath.Join(itemsDir(), "pair-sum"), false)
	for lang, mode := range map[string]fs.FileMode{"go": 0o111, "cpp": 0o111, "python": 0o444} {
		t.Run(lang, func(t *testing.T) {
			p, _ := profile.ForLanguage(lang)
			if p.ArtifactMode != mode {
				t.Fatalf("%s ArtifactMode %o, want %o", p.Name, p.ArtifactMode, mode)
			}
			st, err := harness.Starter(lang, it.sig.Harness, it.sig)
			if err != nil {
				t.Fatal(err)
			}
			itrt.Ensure(t)
			job, in := newJob(t, it, lang, st.Data, 1000, nil)
			res := itrt.Run(t, job, in)
			if !res.Compile.OK {
				t.Fatalf("the starter must compile: %+v", res.Compile)
			}
			failed := false
			for i, c := range it.cases {
				if res.Cases[i].Term != runnerapi.TermOK {
					t.Errorf("%s: the artifact must run as the job UID: %+v", c.id, res.Cases[i])
				}
				if cls, _ := classify(t, it, c, res.Cases[i], c.want); cls != "AC" {
					failed = true
				}
			}
			if !failed {
				t.Error("the starter passes every sample")
			}
		})
	}
}

func TestProfilesEndpoint(t *testing.T) {
	itrt.Ensure(t)
	resp := itrt.GetJSON[runnerapi.ProfilesResponse](t, "/v1/profiles")
	got := map[string]runnerapi.ProfileInfo{}
	for _, p := range resp.Profiles {
		if p.Language != "" {
			got[p.Language] = p
		}
	}
	for lang, name := range map[string]string{"go": "go@1.26", "cpp": "cpp@g++14", "python": "python@3.13"} {
		p, ok := got[lang]
		switch {
		case !ok || p.Profile != name:
			t.Errorf("%s: %+v", lang, p)
		case p.Toolchain == "" || p.Toolchain == "unknown" || len(p.ProfileSHA256) != 64 || p.MemBaselineKB <= 0:
			t.Errorf("%s: versions, ProfileSHA and baseline: %+v", name, p)
		case p.TLMultiplier < 1 || p.Calibrated || p.Baseline != "go@1.26":
			t.Errorf("%s: multiplier %v calibrated %v baseline %q", name, p.TLMultiplier, p.Calibrated, p.Baseline)
		case !slices.Equal(p.Harnesses, harness.Harnesses):
			t.Errorf("%s harnesses %v", name, p.Harnesses)
		}
		itrt.Metric(t, "mem_baseline_kb_"+lang, p.MemBaselineKB)
		itrt.Metric(t, "toolchain_"+lang, strconv.Quote(p.Toolchain))
		itrt.Metric(t, "tl_multiplier_served_"+lang, p.TLMultiplier)
	}
}

// TestFrontRechecksNames: the front's lint.CheckNames answers a file the profile doesn't take
// with 400, never a verdict.
func TestFrontRechecksNames(t *testing.T) {
	it := loadItem(t, filepath.Join(itemsDir(), "pair-sum"), false)
	ref, _ := os.ReadFile(filepath.Join(it.dir, "refs", "solution.go"))
	itrt.Ensure(t)
	job, in := newJob(t, it, "go", ref, 1000, nil)
	job.Files = append(job.Files, runnerapi.File{Path: "evil_amd64.s", Data: []byte("TEXT ·x(SB),0,$0\n")})
	r, err := itrt.Post(context.Background(), t, job, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != http.StatusBadRequest || !bytes.Contains(r.Body, []byte("file_name")) {
		t.Errorf("an extra .s file: %d %s", r.Code, r.Body)
	}
}

// ---- the Go GOCACHE seed ----

// TestCompileWithReadOnlySeedRebuildsNoStd ("compile with a read-only seed rebuilds no std";
// t3 §16.2 block 1): `runner seed-gocache` built the seed (itrt), with fixed future mtimes and
// world-readable modes; a `go build -x` as an unprivileged user with GOCACHE = the seed (not
// writable for it), TMPDIR on its own scratch and GOROOT set compiles no std package; the
// jail's compile, with the seed bound read-only in place, is fast; and a second seed of the
// same toolchain has the same tree hash.
func TestCompileWithReadOnlySeedRebuildsNoStd(t *testing.T) {
	seed := filepath.Join(itrt.Env.Toolchains, "gocache")
	if h, err := profile.TreeHash(seed); err != nil || h != itrt.Env.GoSeedHash {
		t.Fatalf("seed tree hash %s (%v), seed-gocache printed %s", h, err, itrt.Env.GoSeedHash)
	}
	_ = filepath.WalkDir(seed, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, _ := d.Info()
		if !fi.ModTime().Equal(goprofile.SeedTime) {
			t.Fatalf("%s mtime %v, want the fixed future %v", p, fi.ModTime(), goprofile.SeedTime)
		}
		return nil
	})

	// go build -x as nobody: the seed is read-only for it (owned by root, 0755/0644).
	it := loadItem(t, filepath.Join(itemsDir(), "pair-sum"), false)
	dir, err := os.MkdirTemp("", "seedcheck-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	ref, _ := os.ReadFile(filepath.Join(it.dir, "refs", "solution.go"))
	os.WriteFile(filepath.Join(dir, harness.GoLearnerFile), ref, 0o644)
	os.WriteFile(filepath.Join(dir, harness.GoHarnessFile), mustGen(t, it, "go")[0].Data, 0o644)
	work := filepath.Join(dir, "w")
	os.Mkdir(work, 0o777)
	os.Chmod(dir, 0o777)
	os.Chmod(work, 0o777)
	goroot := profile.Resolve(filepath.Join(itrt.Env.Toolchains, "go"))
	cmd := exec.Command(filepath.Join(goroot, "bin", "go"), "build", "-x", "-trimpath", "-buildvcs=false", "-o", filepath.Join(work, "bin"),
		harness.GoLearnerFile, harness.GoHarnessFile)
	cmd.Dir = dir
	cmd.Env = goprofile.BuildEnv(goroot, seed, work)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build -x with the read-only seed: %v\n%s", err, out)
	}
	compiles := 0
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || !strings.HasSuffix(f[0], "/compile") {
			continue
		}
		if !strings.Contains(line, " -p main ") {
			t.Errorf("a std package was rebuilt: %s", line)
		}
		compiles++
	}
	if compiles == 0 {
		t.Errorf("go build -x shows no compile of the learner package:\n%s", out)
	}
	itrt.Metric(t, "seed_compile_actions", compiles)

	// Through the jail: the compile's CPU (SlotMs minus a trivial case) stays far under a cold
	// std build (seconds).
	itrt.Ensure(t)
	job, in := newJob(t, it, "go", ref, 1000, nil)
	res := itrt.Run(t, job, in)
	var caseMs int64
	for _, c := range res.Cases {
		caseMs += c.CPUms
	}
	compileMs := res.Telemetry.SlotMs - caseMs
	itrt.Metric(t, "go_compile_cpu_ms_with_seed", compileMs)
	if !res.Compile.OK || compileMs > 3000 {
		t.Errorf("the jail's Go compile took %d ms CPU with the seed (a cold std build takes seconds): %+v", compileMs, res.Compile)
	}

	// Reproducible: a second seed of the same toolchain has the same tree hash.
	again := filepath.Join(dir, "seed2")
	cmd = exec.Command(itrt.Env.Bin, "seed-gocache", "-out", again)
	cmd.Env = append(os.Environ(), "RUNNER_TOOLCHAINS_DIR="+itrt.Env.Toolchains)
	h2, err := cmd.Output()
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if strings.TrimSpace(string(h2)) != itrt.Env.GoSeedHash {
		t.Errorf("seed tree hash not reproducible: %s vs %s", h2, itrt.Env.GoSeedHash)
	}
	// And seed-gocache refuses a non-empty -out, leaving it untouched.
	cmd = exec.Command(itrt.Env.Bin, "seed-gocache", "-out", again)
	cmd.Env = append(os.Environ(), "RUNNER_TOOLCHAINS_DIR="+itrt.Env.Toolchains)
	if err := cmd.Run(); err == nil {
		t.Error("seed-gocache must refuse a non-empty -out")
	}
}
