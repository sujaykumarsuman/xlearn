//go:build linux && runner_it

package it

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/it/itrt"
)

// The shared runner harness lives in internal/runner/it/itrt (m3-04 made it importable for the
// profiles' and the harness's jail tests); these are this package's names for it.

const (
	cgRoot    = itrt.CgRoot
	baseURL   = itrt.BaseURL
	testToken = itrt.TestToken
	uidLo     = itrt.UIDLo
	uidHi     = itrt.UIDHi
)

var env struct {
	dir, bin, goroot, corpus string
}

func TestMain(m *testing.M) {
	if err := itrt.Setup(); err != nil {
		fmt.Fprintln(os.Stderr, "runner-it setup:", err)
		os.Exit(1)
	}
	env.dir, env.bin, env.goroot, env.corpus = itrt.Env.Dir, itrt.Env.Bin, itrt.Env.GOROOT, itrt.Env.Corpus
	code := m.Run()
	itrt.Teardown()
	os.Exit(code)
}

type runnerProc struct{ *itrt.Runner }

func (p runnerProc) waitExit(t testing.TB, within time.Duration) int { return p.WaitExit(t, within) }

func ensure(t testing.TB) runnerProc { return runnerProc{itrt.Ensure(t)} }

func restart(t testing.TB, extraEnv ...string) runnerProc {
	return runnerProc{itrt.Restart(t, extraEnv...)}
}

func get(t testing.TB, path string, auth bool) (int, []byte) { return itrt.Get(t, path, auth) }

func getJSON[T any](t testing.TB, path string) T { return itrt.GetJSON[T](t, path) }

type response struct {
	code   int
	header http.Header
	body   []byte
	res    *runnerapi.Result
}

func post(ctx context.Context, t testing.TB, job *runnerapi.Job, inputs [][]byte) (*response, error) {
	r, err := itrt.Post(ctx, t, job, inputs)
	if r == nil {
		return nil, err
	}
	return &response{code: r.Code, header: r.Header, body: r.Body, res: r.Res}, err
}

func run(t testing.TB, job *runnerapi.Job, inputs [][]byte) *runnerapi.Result {
	t.Helper()
	return itrt.Run(t, job, inputs)
}

// ---- jobs ----

var jobN int

func source(t testing.TB, prog string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(env.corpus, prog, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newJob builds a job running corpus program prog under profile once per input.
func newJob(t testing.TB, prog, profile string, inputs ...string) (*runnerapi.Job, [][]byte) {
	t.Helper()
	jobN++
	job := &runnerapi.Job{
		ID: fmt.Sprintf("it-%s-%d", prog, jobN), Profile: profile, Harness: "raw@0", Mode: runnerapi.ModeSubmit,
		Files:      []runnerapi.File{{Path: "main.go", Data: source(t, prog)}},
		Limits:     runnerapi.Limits{Case: runnerapi.CaseLimits{CPUms: 1000, MemMB: 256, OutputKB: 64}},
		OutputMode: runnerapi.OutputBytes,
	}
	var ins [][]byte
	for i, in := range inputs {
		job.Cases = append(job.Cases, runnerapi.CaseInput{OpaqueID: "c" + strconv.Itoa(i), Group: runnerapi.GroupRandom, Size: int64(len(in))})
		ins = append(ins, []byte(in))
	}
	return job, ins
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// ---- host-side invariants ----

func jobUIDProcs() []string                  { return itrt.JobUIDProcs() }
func readKeyed(path string) map[string]int64 { return itrt.ReadKeyed(path) }
func containerOOMs() int64                   { return itrt.ContainerOOMs() }
func assertClean(t testing.TB)               { t.Helper(); itrt.AssertClean(t) }
func jsonEncode(w io.Writer, v any) error    { return json.NewEncoder(w).Encode(v) }
func itoa(n int) string                      { return strconv.Itoa(n) }
func metric(t testing.TB, name string, v any) {
	t.Helper()
	itrt.Metric(t, name, v)
}
