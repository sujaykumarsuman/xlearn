package front

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

const token = "front-test-token-0123456789"

func init() {
	profile.Register(&profile.Profile{
		Name: "fake@0", Toolchain: "fake1", Harnesses: []string{"raw@0"}, ArtifactName: "bin", ArtifactMode: 0o111,
		BaselineFiles: []runnerapi.File{{Path: "main.go"}},
		Compile:       profile.Compile{Argv: []string{"/bin/true"}, OutputPath: "/w/out", Pids: 1},
		Exec:          profile.Exec{Argv: []string{"/job/bin"}, Pids: 1},
		TestOnly:      true,
	})
}

// fakeBackend plays the spawner. caseFn decides each case's evidence and output.
type fakeBackend struct {
	t         *testing.T
	mu        sync.Mutex
	caseFn    func(idx int, quiet bool) (done ipc.CaseDone, stdout, result []byte)
	compileFn func() ipc.CompileDone
	quiet     QuietStatus
	written   map[string][]byte
	ended     []bool
	gate      chan struct{} // if set, RunCase blocks until it is closed
	drain     chan struct{}
	broken    chan struct{}
}

func newFake(t *testing.T) *fakeBackend {
	return &fakeBackend{t: t, drain: make(chan struct{}), broken: make(chan struct{}), written: map[string][]byte{},
		caseFn: func(int, bool) (ipc.CaseDone, []byte, []byte) {
			return ipc.CaseDone{Exited: 1, CPUus: 1000, WallUs: 2000, PeakBytes: 4 << 20, BaselineBytes: 2 << 20}, []byte("ok\n"), []byte("out")
		},
		compileFn: func() ipc.CompileDone {
			return ipc.CompileDone{Exited: 1, CPUus: 5000, Export: uint8(ipc.ExportOK)}
		},
	}
}

func (f *fakeBackend) Ping(context.Context) error { return nil }
func (f *fakeBackend) BeginJob(ctx context.Context, slot, pidx int, seq uint64, cpu, mem int64) (Begun, error) {
	dir := f.t.TempDir()
	f.t.Cleanup(func() { os.Chmod(dir, 0o755) }) // the front made it 0555; TempDir must remove it
	d, err := os.Open(dir)
	if err != nil {
		return Begun{}, err
	}
	f.mu.Lock()
	f.written["dir"] = []byte(dir)
	f.mu.Unlock()
	return Begun{SrcDir: d, CanaryMedianUs: 100000}, nil
}
func (f *fakeBackend) Compile(ctx context.Context, slot int) (*CompileRun, error) {
	or, ow, _ := os.Pipe()
	er, ew, _ := os.Pipe()
	ew.Write([]byte("./main.go:3:5: undefined: x\n# noise\n"))
	ow.Close()
	ew.Close()
	done := make(chan CompileOutcome, 1)
	d := f.compileFn()
	done <- CompileOutcome{Done: &d}
	return &CompileRun{Stdout: or, Stderr: er, Done: done}, nil
}
func (f *fakeBackend) RunCase(ctx context.Context, slot int, req ipc.CaseRun) (*CaseRun, error) {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	inR, inW, _ := os.Pipe()
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	resR, resW, _ := os.Pipe()
	d, stdout, result := f.caseFn(int(req.CaseIdx), req.Quiet == 1)
	done := make(chan CaseOutcome, 1)
	go func() {
		io.Copy(io.Discard, inR)
		inR.Close()
		outW.Write(stdout)
		resW.Write(result)
		outW.Close()
		errW.Close()
		resW.Close()
		done <- CaseOutcome{Done: &d}
	}()
	return &CaseRun{Input: inW, Stdout: outR, Stderr: errR, Result: resR, Done: done, Kill: func() {}}, nil
}
func (f *fakeBackend) EndJob(ctx context.Context, slot int, abort bool) (ipc.Teardown, error) {
	f.mu.Lock()
	f.ended = append(f.ended, abort)
	f.mu.Unlock()
	return ipc.TeardownOK, nil
}
func (f *fakeBackend) QuietCheck(context.Context, int) (QuietStatus, error) { return f.quiet, nil }
func (f *fakeBackend) Stats(context.Context) (ipc.StatsReply, error) {
	return ipc.StatsReply{CanaryMedianUs: 100000}, nil
}
func (f *fakeBackend) Probe(context.Context) (ipc.ProbeReply, error) {
	return ipc.ProbeReply{Ran: ipc.ProbeAll}, nil
}
func (f *fakeBackend) DrainRequested() <-chan struct{} { return f.drain }
func (f *fakeBackend) Broken() <-chan struct{}         { return f.broken }

func newServer(t *testing.T, b Backend) (*Server, *httptest.Server) {
	cfg := runner.Config{Mode: runner.ModeProd, MaxJobs: 500, MaxAge: time.Hour, DrainTimeout: 2 * time.Second, ImageDigest: "unknown"}
	man := &runner.Manifest{Token: token, Version: "test", BootEpoch: "h/1/00", CPUModel: "cpu",
		Profiles: []runner.ProfileManifest{{Name: "fake@0", Toolchain: "fake1", ProfileSHA256: "ab", BaselineBytes: 2 << 20}}}
	s := NewServer(cfg, man, b, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.ready.targets = []string{} // no egress probes in unit tests
	s.quietWait, s.quietCheck = 300*time.Millisecond, 50*time.Millisecond
	s.ready.run(context.Background(), b)
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return s, srv
}

func testJob(n int) (*runnerapi.Job, [][]byte) {
	job := &runnerapi.Job{ID: "j1", Profile: "fake@0", Harness: "raw@0", Mode: runnerapi.ModeRun, OutputMode: runnerapi.OutputBytes,
		Files:  []runnerapi.File{{Path: "main.go", Data: []byte("package main")}},
		Limits: runnerapi.Limits{Case: runnerapi.CaseLimits{CPUms: 1000, MemMB: 64, OutputKB: 4}}}
	var in [][]byte
	for i := 0; i < n; i++ {
		job.Cases = append(job.Cases, runnerapi.CaseInput{OpaqueID: "c" + strconv.Itoa(i), Group: runnerapi.GroupRandom, Size: 1})
		in = append(in, []byte("x"))
	}
	return job, in
}

func postJob(t *testing.T, url string, job *runnerapi.Job, in [][]byte, tok string) (int, http.Header, *runnerapi.Result, []byte) {
	t.Helper()
	var b bytes.Buffer
	if err := runnerapi.EncodeJob(&b, job, in); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, url+"/v1/jobs", &b)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", runnerapi.ContentTypeJob)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var res runnerapi.Result
	_ = json.Unmarshal(body, &res)
	return resp.StatusCode, resp.Header, &res, body
}

func TestAuthAndContract(t *testing.T) {
	_, srv := newServer(t, newFake(t))
	job, in := testJob(1)
	if code, _, _, _ := postJob(t, srv.URL, job, in, "wrong-token-0000000000"); code != http.StatusUnauthorized {
		t.Errorf("bad token: %d", code)
	}
	job.Profile = "nope@1"
	if code, _, _, _ := postJob(t, srv.URL, job, in, token); code != http.StatusBadRequest {
		t.Errorf("unknown profile: %d", code)
	}
	job, in = testJob(1)
	code, _, res, body := postJob(t, srv.URL, job, in, token)
	if code != http.StatusOK || res.Cases[0].Term != runnerapi.TermOK || string(res.Cases[0].Output) != "out" {
		t.Fatalf("happy path: %d %s", code, body)
	}
	if err := runnerapi.ValidateResult(job, res); err != nil {
		t.Errorf("ValidateResult: %v", err)
	}
	if res.Versions.BootEpoch != "h/1/00" || res.Versions.ProfileSHA != "ab" || res.Versions.CanaryMedian != 100000 {
		t.Errorf("versions: %+v", res.Versions)
	}
}

func TestSourcesWrittenReadOnly(t *testing.T) {
	f := newFake(t)
	_, srv := newServer(t, f)
	job, in := testJob(1)
	job.HiddenFiles = []runnerapi.File{{Path: "hidden_test.go", Data: []byte("h")}}
	if code, _, _, body := postJob(t, srv.URL, job, in, token); code != http.StatusOK {
		t.Fatalf("%d %s", code, body)
	}
	dir := string(f.written["dir"])
	for _, name := range []string{"main.go", "hidden_test.go"} {
		fi, err := os.Stat(dir + "/" + name)
		if err != nil || fi.Mode().Perm() != 0o444 {
			t.Errorf("%s: %v %v", name, fi.Mode(), err)
		}
	}
	if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o555 {
		t.Errorf("src dir mode %v, want 0555", fi.Mode().Perm())
	}
}

func TestCompileErrorDiagnostics(t *testing.T) {
	f := newFake(t)
	f.compileFn = func() ipc.CompileDone {
		return ipc.CompileDone{Exited: 1, ExitCode: 1, Export: uint8(ipc.ExportSkipped)}
	}
	_, srv := newServer(t, f)
	job, in := testJob(2)
	_, _, res, _ := postJob(t, srv.URL, job, in, token)
	if res.Compile.OK || len(res.Compile.Diags) != 1 || res.Compile.Diags[0].Line != 3 || res.Compile.Diags[0].File != "main.go" {
		t.Errorf("CE: %+v", res.Compile)
	}
	if res.Cases[0].Term != runnerapi.TermNotRun {
		t.Errorf("cases after a CE: %+v", res.Cases)
	}
}

func TestStopRules(t *testing.T) {
	f := newFake(t)
	fail := map[int]bool{1: true}
	f.caseFn = func(idx int, quiet bool) (ipc.CaseDone, []byte, []byte) {
		if fail[idx] {
			return ipc.CaseDone{Exited: 1, ExitCode: 1, CPUus: 1000}, nil, nil
		}
		return ipc.CaseDone{Exited: 1, CPUus: 1000}, nil, nil
	}
	_, srv := newServer(t, f)
	job, in := testJob(4)
	job.Cases[0].Group, job.Cases[1].Group = runnerapi.GroupSample, runnerapi.GroupSample
	job.StopGroupOn = map[runnerapi.Group]runnerapi.StopRule{runnerapi.GroupSample: runnerapi.StopAnyFail}
	_, _, res, _ := postJob(t, srv.URL, job, in, token)
	want := []runnerapi.Term{runnerapi.TermOK, runnerapi.TermExitNonzero, runnerapi.TermNotRun, runnerapi.TermNotRun}
	for i, w := range want {
		if res.Cases[i].Term != w {
			t.Errorf("sample any_fail: case %d = %s, want %s", i, res.Cases[i].Term, w)
		}
	}

	// perf: stop at the first TLE, the rest of the group not_run.
	f.caseFn = func(idx int, quiet bool) (ipc.CaseDone, []byte, []byte) {
		if idx == 1 {
			return ipc.CaseDone{Signal: 9, Kill: uint8(measure.KillCPU), CPUus: 1_600_000}, nil, nil
		}
		return ipc.CaseDone{Exited: 1, CPUus: 1000}, nil, nil
	}
	job, in = testJob(3)
	for i := range job.Cases {
		job.Cases[i].Group = runnerapi.GroupPerf
	}
	job.StopGroupOn = map[runnerapi.Group]runnerapi.StopRule{runnerapi.GroupPerf: runnerapi.StopTLE}
	_, _, res, _ = postJob(t, srv.URL, job, in, token)
	want = []runnerapi.Term{runnerapi.TermOK, runnerapi.TermTLE, runnerapi.TermNotRun}
	for i, w := range want {
		if res.Cases[i].Term != w {
			t.Errorf("perf tle: case %d = %s, want %s", i, res.Cases[i].Term, w)
		}
	}
}

func TestSuspectTLEQuietRerun(t *testing.T) {
	f := newFake(t)
	f.quiet = QuietStatus{Steal: 0.01, Canary: 1.0}
	f.caseFn = func(idx int, quiet bool) (ipc.CaseDone, []byte, []byte) {
		d := ipc.CaseDone{Signal: 9, Kill: uint8(measure.KillCPU), CPUus: 1_600_000}
		if !quiet {
			d.OtherBusy = 1
		}
		return d, nil, nil
	}
	_, srv := newServer(t, f)
	job, in := testJob(1)
	_, _, res, body := postJob(t, srv.URL, job, in, token)
	if res.Throttled || res.Telemetry.QuietReruns != 1 || !res.Cases[0].Quiet || res.Cases[0].Term != runnerapi.TermTLE {
		t.Errorf("quiet re-run: %s", body)
	}

	// Quiet conditions never met → Throttled.
	f.quiet = QuietStatus{Steal: 0.2, Canary: 1.5}
	job, in = testJob(1)
	f.quiet = QuietStatus{Steal: 0.2}
	_, _, res, _ = postJob(t, srv.URL, job, in, token)
	if !res.Throttled {
		t.Errorf("want throttled when quiet conditions fail")
	}
}

func TestSaturatedAndDrain(t *testing.T) {
	f := newFake(t)
	f.gate = make(chan struct{})
	s, srv := newServer(t, f)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	bodies := make([][]byte, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			job, in := testJob(1)
			codes[i], _, _, bodies[i] = postJob(t, srv.URL, job, in, token)
		}(i)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		s.mu.Lock()
		busy := s.busy[0] && s.busy[1]
		s.mu.Unlock()
		if busy || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, in := testJob(1)
	code, hdr, res, body := postJob(t, srv.URL, job, in, token)
	if code != http.StatusServiceUnavailable || hdr.Get("Retry-After") == "" || res.Infra == nil || res.Infra.Kind != runnerapi.InfraSaturated {
		t.Errorf("third job: %d %s", code, body)
	}
	// Drain with both jobs stuck: after the drain timeout they answer killed (503).
	s.StartDrain("test")
	wg.Wait()
	for i := range codes {
		var r runnerapi.Result
		_ = json.Unmarshal(bodies[i], &r)
		if codes[i] != http.StatusServiceUnavailable || r.Infra == nil || r.Infra.Kind != runnerapi.InfraKilled {
			t.Errorf("job %d after the drain: %d %s", i, codes[i], bodies[i])
		}
	}
	select {
	case <-s.Drained():
	case <-time.After(10 * time.Second):
		t.Fatal("drain never finished")
	}
}

func TestReadyzProdRefusesOnSecurityCanary(t *testing.T) {
	f := newFake(t)
	s, srv := newServer(t, f)
	s.ready.probe = ipc.ProbeReply{Ran: ipc.ProbeAll, Failed: ipc.ProbeUserNS}
	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("prod readyz with a failing security canary: %d", resp.StatusCode)
	}
	job, in := testJob(1)
	code, _, res, body := postJob(t, srv.URL, job, in, token)
	if code != http.StatusOK || res.Infra == nil || res.Infra.Kind != runnerapi.InfraSetup {
		t.Errorf("prod job with a failing security canary: %d %s", code, body)
	}
	// dev only warns.
	s.cfg.Mode = runner.ModeDev
	if ok, checks := s.ready.report(runner.ModeDev, time.Now()); !ok || checks["userns_denied"] != "warn" {
		t.Errorf("dev: %v %v", ok, checks)
	}
}

func TestKeeper(t *testing.T) {
	over := 0
	k := &keeper{keep: 4, hard: 8, over: func() { over++ }}
	k.Write([]byte("abcdef"))
	k.Write([]byte("ghij"))
	if string(k.kept) != "abcd" || k.n != 10 || over != 1 {
		t.Errorf("kept %q n %d over %d", k.kept, k.n, over)
	}
}
