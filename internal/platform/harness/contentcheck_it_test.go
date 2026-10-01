//go:build linux && runner_it

package harness_test

// The public content check (m3-04 task 8, m3-01's hand-off; t1 §7.2: "starters are generated
// and compiled", "the reference passes the samples ... with --network none"), in the runner-it
// lane: for every code part of every public item and each language it lists,
//
//   - the starter (`_starter/solution.<ext>`, else harness.Starter) compiles with the
//     generated harness through the real profile and does NOT pass the samples;
//   - every public solution-stage reference (`_code/solution.<ext>`) passes every public sample
//     through the jail (an empty network namespace), compared with the item's checker
//     (internal/platform/checker).
//
// Public data only (curriculum/, never xlearn-evalpack). With no code item on main yet the
// curriculum pass is vacuous and says so; m3-02's synthetic fixture content runs the same check
// as its self-test. m3-15 re-runs it inside the runner image.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/checker"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/it/itrt"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/cpp"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	_ "github.com/sujaykumarsuman/xlearn/internal/runner/profile/python"
)

func TestMain(m *testing.M) { os.Exit(itrt.Main(m)) }

var refExt = map[string]string{"go": "go", "cpp": "cpp", "python": "py"}

func TestPublicContentCheck(t *testing.T) {
	n := contentCheck(t, filepath.Join(itrt.Env.Root, "curriculum", "courses"))
	if n == 0 {
		t.Log("curriculum: no code item on main yet; the content check passes vacuously")
	}
	itrt.Metric(t, "content_check_curriculum_code_parts", n)
}

// TestContentCheckSelfTest runs the same check over m3-02's synthetic fixture content (public,
// hand-made), so the check itself is exercised while the curriculum has no code item.
func TestContentCheckSelfTest(t *testing.T) {
	if n := contentCheck(t, filepath.Join(itrt.Env.Root, "internal", "judge", "testdata", "content", "courses")); n == 0 {
		t.Fatal("the fixture content has code parts; the self-test found none")
	}
}

// contentCheck checks every code part under root/<course>/items/<id>/item.json and returns
// how many it checked.
func contentCheck(t *testing.T, root string) int {
	t.Helper()
	items, err := filepath.Glob(filepath.Join(root, "*", "items", "*", "item.json"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, path := range items {
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
				continue // gotest@1 modules are p-01's
			}
			n++
			for _, lang := range part.Config.Languages {
				t.Run(fmt.Sprintf("%s/%s/%s", item.ID, part.ID, lang), func(t *testing.T) {
					checkPart(t, dir, part.Config, lang)
				})
			}
		}
	}
	return n
}

func checkPart(t *testing.T, dir string, cfg course.PartConfig, lang string) {
	sig, err := harness.ParseSig(cfg.Harness, cfg.Signature)
	if err != nil {
		t.Fatal(err)
	}
	chk, err := checker.Lookup(cfg.Checker)
	if err != nil {
		t.Fatal(err)
	}
	type sample struct {
		in    *harness.Input
		frame []byte
		want  harness.Value
	}
	var samples []sample
	for _, s := range cfg.Samples {
		m := map[string]any{"args": s.Args}
		if sig.Class() {
			m["ops"] = s.Ops
		}
		raw, _ := json.Marshal(m)
		in, err := harness.DecodeInput(sig, raw)
		if err != nil {
			t.Fatalf("sample %s: %v", s.ID, err)
		}
		frame, _ := harness.EncodeInput(sig, raw)
		want, err := harness.DecodeExpected(sig, in, s.Expected)
		if err != nil {
			t.Fatalf("sample %s expected: %v", s.ID, err)
		}
		samples = append(samples, sample{in, frame, want})
	}
	// passed runs learner against the samples through the jail and counts the passes; a CE
	// fails the test.
	passed := func(what string, learner []byte) int {
		p, ok := profile.ForLanguage(lang)
		if !ok {
			t.Fatalf("no profile serves %s", lang)
		}
		gen, err := harness.Generate(lang, cfg.Harness, sig)
		if err != nil {
			t.Fatal(err)
		}
		job := &runnerapi.Job{
			ID: "content-" + lang, Profile: p.Name, Harness: cfg.Harness, Mode: runnerapi.ModeSubmit,
			Files:      append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: learner}}, gen...),
			Limits:     runnerapi.Limits{Case: runnerapi.CaseLimits{CPUms: 10000, MemMB: 256, OutputKB: 8192}},
			OutputMode: runnerapi.OutputBytes,
		}
		var inputs [][]byte
		for i, s := range samples {
			job.Cases = append(job.Cases, runnerapi.CaseInput{OpaqueID: fmt.Sprint("s", i), Group: runnerapi.GroupSample, Size: int64(len(s.frame))})
			inputs = append(inputs, s.frame)
		}
		itrt.Ensure(t)
		res := itrt.Run(t, job, inputs)
		if !res.Compile.OK {
			t.Fatalf("the %s does not compile with the harness: %+v", what, res.Compile)
		}
		npass := 0
		for i, s := range samples {
			cr := res.Cases[i]
			if cr.Term != runnerapi.TermOK {
				continue
			}
			if got, err := harness.DecodeOutputFor(sig, s.in, cr.Output); err == nil && chk.Check(got, s.want).OK {
				npass++
			}
		}
		return npass
	}

	starter, err := os.ReadFile(filepath.Join(dir, "_starter", harness.LearnerFile(lang)))
	if err != nil {
		st, err := harness.Starter(lang, cfg.Harness, sig)
		if err != nil {
			t.Fatal(err)
		}
		starter = st.Data
	}
	if len(samples) > 0 && passed("starter", starter) == len(samples) {
		t.Errorf("the starter passes every sample (it must not be a solution)")
	}
	ref, err := os.ReadFile(filepath.Join(dir, "_code", "solution."+refExt[lang]))
	if err != nil {
		return // no public reference in this language
	}
	if got := passed("reference", ref); got != len(samples) {
		t.Errorf("the public reference passes %d of %d samples", got, len(samples))
	}
}
