//go:build runner_acceptance

package acceptance

// Section G (references, SUBSET=full only): m3-04's items through the image's own toolchains (t1
// §7.2's "in the public production runner image"). Every item, case and solution is SYNTHETIC or
// public (internal/runner/testdata/items, m3-02's fixture content, curriculum/, the lint's
// evasion fixtures); nothing comes from xlearn-evalpack.
//
//   - G1 references pass every sample and hidden case in Go, C++ and Python (the item's checker;
//     for a canonicalizable output, C++'s and Python's fd-4 bytes equal Go's);
//   - G2 wrong solutions are classified WA / TLE / RE / CE / MLE as their `want:` header says
//     (REJECTED ones never reach the runner: judge's lint rejects them, checked here too);
//   - G3 the public content check (m3-04 task 8): starters compile with the harness and don't
//     pass every sample; public `_code` references pass every sample;
//   - G4 the lint-evasion fixtures die of SIGSYS (one job per language, last: one rotation each).

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/checker"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi/lint"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile/proftest"
)

// refTL is each language's CPU TL for the references (generous: the multipliers are calibrated,
// not enforced here). Wrong solutions run at 1 s, as in m3-04's runner-it lane.
var refTL = map[string]int64{"go": 2000, "cpp": 2000, "python": 10000}

type gcase struct {
	id    string
	group runnerapi.Group
	in    *harness.Input
	raw   []byte
	want  *harness.Value // nil: the Go reference's output is the expected one
}

type gitem struct {
	slug  string
	dir   string
	cfg   course.PartConfig
	sig   *harness.Sig
	check checker.Checker
	cases []gcase
}

func repoPath(parts ...string) string { return filepath.Join(append([]string{cfg.repo}, parts...)...) }

func loadGItem(t *testing.T, dir string, hidden bool) *gitem {
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
		t.Fatalf("%s: %v", dir, err)
	}
	it := &gitem{slug: doc.ID, dir: dir}
	for _, p := range doc.Parts {
		if p.Type == "code" {
			it.cfg = p.Config
		}
	}
	if it.sig, err = harness.ParseSig(it.cfg.Harness, it.cfg.Signature); err != nil {
		t.Fatalf("%s: %v", it.slug, err)
	}
	if it.check, err = checker.Lookup(it.cfg.Checker); err != nil {
		t.Fatalf("%s: %v", it.slug, err)
	}
	for _, sm := range it.cfg.Samples {
		it.add(t, sm.ID, runnerapi.GroupSample, sm.Ops, sm.Args, sm.Expected)
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

func (it *gitem) add(t *testing.T, id string, g runnerapi.Group, ops []string, args []json.RawMessage, expected json.RawMessage) {
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

func (it *gitem) addRaw(t *testing.T, id string, g runnerapi.Group, raw, expected []byte) {
	t.Helper()
	in, err := harness.DecodeInput(it.sig, raw)
	if err != nil {
		t.Fatalf("%s %s: %v", it.slug, id, err)
	}
	c := gcase{id: id, group: g, in: in, raw: raw}
	if len(expected) > 0 {
		v, err := harness.DecodeExpected(it.sig, in, expected)
		if err != nil {
			t.Fatalf("%s %s expected: %v", it.slug, id, err)
		}
		c.want = &v
	}
	it.cases = append(it.cases, c)
}

func (s *suite) itemJob(t *testing.T, it *gitem, lang, label string, learner []byte, cpuMs int64, stop map[runnerapi.Group]runnerapi.StopRule) job {
	t.Helper()
	raws := make([][]byte, len(it.cases))
	groups := make([]runnerapi.Group, len(it.cases))
	for i, c := range it.cases {
		raws[i], groups[i] = c.raw, c.group
	}
	j := s.newJob(t, "G", lang, it.slug+" "+label, it.sig, learner, raws, runnerapi.ModeSubmit, limits{cpuMs, 256}, stop)
	for i := range j.job.Cases {
		j.job.Cases[i].OpaqueID, j.job.Cases[i].Group = it.cases[i].id, groups[i]
	}
	return j
}

// classify is the TEST-ONLY term → class mapper (t4 §4.1's verdict table, precedence TLE > MLE >
// OLE > RE > WA; m3-06 owns the production one): the runner's terms, then the fd-4 frame, then
// the item's checker against want.
func classify(it *gitem, c gcase, cr runnerapi.CaseResult, want *harness.Value) (string, harness.Value) {
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
		return "RE", v
	case errors.As(err, &he) && he.Code == "bad_input":
		return "BAD_INPUT", v
	case err != nil:
		return "WA", v
	}
	if want != nil && !it.check.Check(v, *want).OK {
		return "WA", v
	}
	return "AC", v
}

func (s *suite) sectionG(t *testing.T, r *section) {
	var items []*gitem
	for _, slug := range []string{"pair-sum", "group-words", "reverse-list", "level-order", "clone-graph", "running-median", "min-stack"} {
		items = append(items, loadGItem(t, repoPath("internal", "runner", "testdata", "items", slug), true))
	}
	fixtures, _ := filepath.Glob(repoPath("internal", "judge", "testdata", "content", "courses", "fixture", "items", "*"))
	for _, d := range fixtures {
		items = append(items, loadGItem(t, d, false))
	}

	// G1 references.
	refs := 0
	for _, it := range items {
		goOut := map[string][]byte{}
		goVal := map[string]harness.Value{}
		for _, lang := range langs {
			src := filepath.Join(it.dir, "refs", "solution."+exts[lang])
			if _, err := os.Stat(src); err != nil {
				src = filepath.Join(it.dir, "_code", "solution."+exts[lang])
			}
			learner, err := os.ReadFile(src)
			if err != nil {
				r.expect(t, false, fmt.Sprintf("G1 %s %s reference exists", it.slug, lang), "%v", err)
				continue
			}
			res := s.run(t, s.itemJob(t, it, lang, "ref", learner, refTL[lang], nil))
			if !r.expect(t, res.Compile.OK, fmt.Sprintf("G1 %s %s reference compiles", it.slug, lang), "%+v", res.Compile) {
				continue
			}
			bad := 0
			for i, c := range it.cases {
				cr := res.Cases[i]
				want := c.want
				if want == nil && lang != "go" {
					w := goVal[c.id]
					want = &w
				}
				cls, v := classify(it, c, cr, want)
				if cls != "AC" {
					bad++
					r.add(fmt.Sprintf("G1 %s %s %s", it.slug, lang, c.id), false, fmt.Sprintf("%s: %s", cls, describe(cr)))
					t.Errorf("G1 %s %s %s: %s %s", it.slug, lang, c.id, cls, describe(cr))
					continue
				}
				if lang == "go" {
					goOut[c.id], goVal[c.id] = cr.Output, v
				} else if harness.CanonicalizableOutput(it.sig) && !bytes.Equal(cr.Output, goOut[c.id]) {
					bad++
					r.add(fmt.Sprintf("G1 %s %s %s fd-4 bytes equal Go's", it.slug, lang, c.id), false,
						fmt.Sprintf("%d vs %d bytes", len(cr.Output), len(goOut[c.id])))
					t.Errorf("G1 %s %s %s: fd-4 bytes differ from Go's", it.slug, lang, c.id)
				}
			}
			if bad == 0 {
				refs++
			}
		}
	}
	r.add("G1 references pass in every language", refs == 3*len(items), fmt.Sprintf("%d/%d item×language", refs, 3*len(items)))
	r.metric("g1_reference_runs", refs)

	// G2 wrong solutions.
	wantRE := regexp.MustCompile(`^(?://|#) want: ([A-Z]+)(?: line=(\d+))?(?: (file=harness))?`)
	stop := map[runnerapi.Group]runnerapi.StopRule{
		runnerapi.GroupSample: runnerapi.StopAnyFail, runnerapi.GroupEdge: runnerapi.StopAnyFail,
		runnerapi.GroupRandom: runnerapi.StopAnyFail, runnerapi.GroupPerf: runnerapi.StopTLE,
	}
	classified := 0
	for _, slug := range []string{"pair-sum", "reverse-list"} {
		it := loadGItem(t, repoPath("internal", "runner", "testdata", "items", slug), true)
		ref, _ := os.ReadFile(filepath.Join(it.dir, "refs", "solution.go"))
		res := s.run(t, s.itemJob(t, it, "go", "ref (expected outputs)", ref, refTL["go"], nil))
		for i, c := range it.cases {
			if c.want == nil {
				_, v := classify(it, c, res.Cases[i], nil)
				it.cases[i].want = &v
			}
		}
		paths, _ := filepath.Glob(filepath.Join(it.dir, "wrong", "*"))
		for _, p := range paths {
			lang := map[string]string{".go": "go", ".cpp": "cpp", ".py": "python"}[filepath.Ext(p)]
			name := fmt.Sprintf("G2 %s %s", slug, filepath.Base(p))
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			m := wantRE.FindStringSubmatch(string(src))
			if m == nil {
				r.expect(t, false, name, "no want: header")
				continue
			}
			want := m[1]
			gen, err := harness.Generate(lang, it.sig.Harness, it.sig)
			if err != nil {
				t.Fatal(err)
			}
			vs := lint.Check(lint.Language(lang), append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: src}}, gen...))
			if want == "REJECTED" {
				if r.expect(t, len(vs) > 0, name+" is REJECTED by judge's lint (never reaches the runner)", "%v", vs) {
					classified++
				}
				continue
			}
			res := s.run(t, s.itemJob(t, it, lang, filepath.Base(p), src, 1000, stop))
			if want == "CE" {
				ok := !res.Compile.OK && len(res.Compile.Diags) > 0
				if ok {
					d := res.Compile.Diags[0]
					wantFile := harness.LearnerFile(lang)
					if m[3] != "" {
						wantFile = map[string]string{"go": harness.GoHarnessFile, "cpp": harness.CppHarnessFile, "python": harness.PythonMainFile}[lang]
					}
					ok = d.File == wantFile && (m[2] == "" || strconv.Itoa(d.Line) == m[2])
				}
				if r.expect(t, ok, name+" is CE", "%+v", res.Compile) {
					classified++
				}
				continue
			}
			if !r.expect(t, res.Compile.OK, name+" compiles", "%+v", res.Compile) {
				continue
			}
			got, first := "", ""
			for i, c := range it.cases {
				cls, _ := classify(it, c, res.Cases[i], c.want)
				if cls != "AC" && cls != "NOT_RUN" {
					got, first = cls, c.id+": "+describe(res.Cases[i])
					break
				}
			}
			if r.expect(t, got == want, name+" is "+want, "got %q (%s)", got, first) {
				classified++
			}
		}
	}
	r.metric("g2_wrong_classified", classified)

	// G3 the public content check: curriculum/ (vacuous until a code item lands) and m3-02's
	// fixture content as its self-test.
	n := s.contentCheck(t, r, repoPath("curriculum", "courses"))
	r.metric("g3_curriculum_code_parts", n)
	nf := s.contentCheck(t, r, repoPath("internal", "judge", "testdata", "content", "courses"))
	r.expect(t, nf > 0, "G3 the fixture content's code parts are checked", "%d", nf)

	// G4 the lint-evasion fixtures: the jail stops them (SIGSYS → RE, counted, one rotation each).
	ps := loadGItem(t, repoPath("internal", "runner", "testdata", "items", "pair-sum"), false)
	for _, lang := range langs {
		src, err := os.ReadFile(repoPath("internal", "platform", "runnerapi", "lint", "testdata", lang, "evasion."+exts[lang]))
		if err != nil {
			t.Fatal(err)
		}
		j := s.itemJob(t, ps, lang, "evasion", src, 1000, stop)
		j.planSigsys = true
		res := s.run(t, j)
		r.expect(t, res.Compile.OK && len(res.Cases) > 0 && isSIGSYS(res.Cases[0]), fmt.Sprintf("G4 %s evasion fixture dies of SIGSYS", lang),
			"%+v %s", res.Compile, func() string {
				if len(res.Cases) > 0 {
					return describe(res.Cases[0])
				}
				return ""
			}())
	}
}

// contentCheck is m3-04 task 8's public content check over root/<course>/items/<id>/item.json
// and returns how many code parts it checked.
func (s *suite) contentCheck(t *testing.T, r *section, root string) int {
	paths, _ := filepath.Glob(filepath.Join(root, "*", "items", "*", "item.json"))
	n := 0
	for _, path := range paths {
		dir := filepath.Dir(path)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var item struct {
			ID    string `json:"id"`
			Parts []struct {
				ID     string            `json:"id"`
				Type   string            `json:"type"`
				Config course.PartConfig `json:"config"`
			} `json:"parts"`
		}
		if err := json.Unmarshal(b, &item); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, part := range item.Parts {
			if part.Type != "code" || part.Config.Harness == "gotest@1" {
				continue
			}
			n++
			it := &gitem{slug: item.ID, dir: dir, cfg: part.Config}
			if it.sig, err = harness.ParseSig(part.Config.Harness, part.Config.Signature); err != nil {
				r.expect(t, false, "G3 "+item.ID+" signature", "%v", err)
				continue
			}
			if it.check, err = checker.Lookup(part.Config.Checker); err != nil {
				r.expect(t, false, "G3 "+item.ID+" checker", "%v", err)
				continue
			}
			for _, sm := range part.Config.Samples {
				it.add(t, sm.ID, runnerapi.GroupSample, sm.Ops, sm.Args, sm.Expected)
			}
			for _, lang := range part.Config.Languages {
				starter, err := os.ReadFile(filepath.Join(dir, "_starter", harness.LearnerFile(lang)))
				if err != nil {
					st, err := harness.Starter(lang, part.Config.Harness, it.sig)
					if err != nil {
						t.Fatal(err)
					}
					starter = st.Data
				}
				pass, ok := s.samplesPassed(t, it, lang, starter)
				name := fmt.Sprintf("G3 %s/%s %s", item.ID, part.ID, lang)
				r.expect(t, ok, name+" starter compiles", "")
				if len(it.cases) > 0 {
					r.expect(t, pass < len(it.cases), name+" starter does not pass every sample", "%d/%d", pass, len(it.cases))
				}
				ref, err := os.ReadFile(filepath.Join(dir, "_code", "solution."+exts[lang]))
				if err != nil {
					continue
				}
				pass, ok = s.samplesPassed(t, it, lang, ref)
				r.expect(t, ok && pass == len(it.cases), name+" public reference passes every sample", "%d/%d compiled=%v", pass, len(it.cases), ok)
			}
		}
	}
	return n
}

func (s *suite) samplesPassed(t *testing.T, it *gitem, lang string, learner []byte) (int, bool) {
	res := s.run(t, s.itemJob(t, it, lang, "content", learner, 10000, nil))
	if !res.Compile.OK {
		return 0, false
	}
	n := 0
	for i, c := range it.cases {
		if cls, _ := classify(it, c, res.Cases[i], c.want); cls == "AC" {
			n++
		}
	}
	return n, true
}
