//go:build runner_acceptance

package acceptance

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// The probes and kernels (testdata keeps these learner-style files out of `go build ./...`).
//
//go:embed testdata/probes testdata/kernels
var testdata embed.FS

var (
	langs = []string{"go", "cpp", "python"}
	exts  = map[string]string{"go": "go", "cpp": "cpp", "python": "py"}
)

// The two signatures the suite's own programs use (func-json@1, the normal profile path).
var (
	probeSig  = mustSig("probe", "arg", "string", "string")
	kernelSig = mustSig("kernel", "n", "int", "int64")
)

func mustSig(name, param, ptype, ret string) *harness.Sig {
	s, err := harness.ParseSig(harness.FuncJSON, &course.Signature{Mode: "function", Name: name,
		Params: []course.Param{{Name: param, Type: ptype}}, Returns: ret})
	if err != nil {
		panic(err)
	}
	return s
}

func source(t *testing.T, kind, lang, name string) []byte {
	t.Helper()
	b, err := testdata.ReadFile(fmt.Sprintf("testdata/%s/%s/%s.%s", kind, lang, name, exts[lang]))
	if err != nil {
		t.Fatalf("testdata: %v", err)
	}
	return b
}

// limits are a job's case limits.
type limits struct {
	cpuMs, memMB int64
}

// newJob builds a func-json@1 job of learner under lang's profile, one case per raw input.
func (s *suite) newJob(t *testing.T, sec, lang, label string, sig *harness.Sig, learner []byte, raws [][]byte,
	mode runnerapi.Mode, lim limits, stop map[runnerapi.Group]runnerapi.StopRule) job {
	t.Helper()
	p := s.profiles[lang]
	gen, err := harness.Generate(lang, sig.Harness, sig)
	if err != nil {
		t.Fatal(err)
	}
	s.jobN++
	id := fmt.Sprintf("acc-%s-%s-%d", strings.ToLower(sec), lang, s.jobN)
	j := &runnerapi.Job{
		ID: id, Profile: p.Profile, Harness: sig.Harness, Mode: mode,
		Files:       append([]runnerapi.File{{Path: harness.LearnerFile(lang), Data: learner}}, gen...),
		Limits:      runnerapi.Limits{Case: runnerapi.CaseLimits{CPUms: lim.cpuMs, MemMB: lim.memMB, OutputKB: 8192}},
		OutputMode:  runnerapi.OutputBytes,
		StopGroupOn: stop,
	}
	var inputs [][]byte
	for i, raw := range raws {
		frame, err := harness.EncodeInput(sig, raw)
		if err != nil {
			t.Fatalf("%s: input %s: %v", label, raw, err)
		}
		j.Cases = append(j.Cases, runnerapi.CaseInput{OpaqueID: fmt.Sprintf("c%d", i), Group: runnerapi.GroupRandom, Size: int64(len(frame))})
		inputs = append(inputs, frame)
	}
	return job{section: sec, label: fmt.Sprintf("%s %s/%s", sec, lang, label), job: j, inputs: inputs}
}

func argsJSON(v any) []byte {
	b, _ := json.Marshal(map[string]any{"args": []any{v}})
	return b
}

// probeJob runs testdata/probes/<lang>/<file> once per arg.
func (s *suite) probeJob(t *testing.T, sec, lang, file string, args []string, mode runnerapi.Mode, lim limits) job {
	t.Helper()
	raws := make([][]byte, len(args))
	for i, a := range args {
		raws[i] = argsJSON(a)
	}
	j := s.newJob(t, sec, lang, file+"["+strings.Join(trimArgs(args), ",")+"]", probeSig, source(t, "probes", lang, file), raws, mode, lim, nil)
	return j
}

func trimArgs(args []string) []string {
	if len(args) <= 6 {
		return args
	}
	return append(append([]string{}, args[:3]...), fmt.Sprintf("…×%d", len(args)))
}

// probeOut decodes a probe case's string result (fd 4).
func probeOut(c runnerapi.CaseResult) (string, error) {
	if len(c.Output) == 0 {
		return "", errNoOutput
	}
	in, err := harness.DecodeInput(probeSig, argsJSON("x"))
	if err != nil {
		return "", err
	}
	v, err := harness.DecodeOutputFor(probeSig, in, c.Output)
	if err != nil {
		return "", err
	}
	return v.Str, nil
}

// describe is a one-line view of a case for check details.
func describe(c runnerapi.CaseResult) string {
	out, err := probeOut(c)
	d := fmt.Sprintf("term=%s", c.Term)
	if c.Signal != "" {
		d += " signal=" + c.Signal
	}
	if c.Term == runnerapi.TermExitNonzero {
		d += fmt.Sprintf(" exit=%d", c.ExitCode)
	}
	if c.ForkLimit {
		d += " fork_limit"
	}
	if c.IdleKill {
		d += " idle"
	}
	if c.PanicClass != "" {
		d += " panic=" + c.PanicClass
	}
	d += fmt.Sprintf(" cpu=%dms wall=%dms", c.CPUms, c.WallMs)
	if err == nil {
		d += fmt.Sprintf(" out=%q", trunc([]byte(out), 120))
	}
	if len(c.Stderr) > 0 {
		d += fmt.Sprintf(" stderr=%q", trunc(c.Stderr, 160))
	}
	return d
}

func isSIGSYS(c runnerapi.CaseResult) bool {
	return c.Term == runnerapi.TermSignal && c.Signal == "SIGSYS"
}

// isRE is t4 §4.1's runtime-error class seen from the runner's side: a signal, a non-zero exit
// or a caught panic frame.
func isRE(c runnerapi.CaseResult) bool {
	return c.Term == runnerapi.TermSignal || c.Term == runnerapi.TermExitNonzero ||
		(c.Term == runnerapi.TermOK && c.PanicClass != "")
}
