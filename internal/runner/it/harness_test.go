//go:build linux && runner_it

package it

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

const (
	cgRoot    = "/sys/fs/cgroup"
	addr      = "127.0.0.1:18090"
	baseURL   = "http://" + addr
	testToken = "runner-it-token-0123456789abcdef"
	// The job UID pool (spawner.uidPoolStart..+uidPoolSize).
	uidLo, uidHi = 10000, 60000
)

var env struct {
	dir, bin, goroot, seed, tokenFile, jailDir, corpus string
}

var (
	rtMu sync.Mutex
	rt   *runnerProc
)

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		fmt.Fprintln(os.Stderr, "runner-it setup:", err)
		os.Exit(1)
	}
	code := m.Run()
	rtMu.Lock()
	if rt != nil {
		rt.stop()
	}
	rtMu.Unlock()
	os.Exit(code)
}

func setup() error {
	if os.Geteuid() != 0 {
		return errors.New("runner-it runs as root with a delegated cgroup (CI's privileged container, or `sudo make runner-it` on a Linux VM)")
	}
	if err := evacuateRootCgroup(); err != nil {
		return fmt.Errorf("evacuate the cgroup root: %w", err)
	}
	var err error
	if env.dir, err = os.MkdirTemp("", "runner-it-"); err != nil {
		return err
	}
	if err := os.Chmod(env.dir, 0o755); err != nil {
		return err
	}
	root, err := goEnv("GOMOD")
	if err != nil {
		return err
	}
	root = filepath.Dir(root)
	env.corpus = filepath.Join(root, "internal", "runner", "testdata", "corpus")
	if env.goroot, err = goEnv("GOROOT"); err != nil {
		return err
	}
	env.bin = filepath.Join(env.dir, "runner")
	build := exec.Command("go", "build", "-tags", "runner_it", "-o", env.bin, "./cmd/runner")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("build the runner: %v\n%s", err, out)
	}
	if err := buildSeed(); err != nil {
		return fmt.Errorf("build the GOCACHE seed: %w", err)
	}
	env.tokenFile = filepath.Join(env.dir, "token")
	if err := os.WriteFile(env.tokenFile, []byte(testToken+"\n"), 0o400); err != nil {
		return err
	}
	env.jailDir = filepath.Join(env.dir, "jail")
	return nil
}

func goEnv(k string) (string, error) {
	out, err := exec.Command("go", "env", k).Output()
	return strings.TrimSpace(string(out)), err
}

// evacuateRootCgroup moves every process out of the container's cgroup root into a sibling
// leaf, so the runner can enable controllers there (the no-internal-process rule). In the pod
// the spawner is PID 1 and alone; CI's container also has the step shell and go test in it.
func evacuateRootCgroup() error {
	leaf := filepath.Join(cgRoot, "ci-init")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		b, err := os.ReadFile(filepath.Join(cgRoot, "cgroup.procs"))
		if err != nil {
			return err
		}
		pids := strings.Fields(string(b))
		if len(pids) == 0 {
			return nil
		}
		for _, p := range pids {
			_ = os.WriteFile(filepath.Join(leaf, "cgroup.procs"), []byte(p), 0)
		}
	}
	return errors.New("processes keep appearing in the cgroup root")
}

// buildSeed fills a GOCACHE with the corpus's std dependencies using the testgo profile's
// exact flags and env; the compile jail then uses it read-only in place (t3 §16.2 block 1).
func buildSeed() error {
	env.seed = filepath.Join(env.dir, "gocache-seed")
	work := filepath.Join(env.dir, "seed-work")
	ents, err := os.ReadDir(env.corpus)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.Name() == "badcompile" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(env.corpus, e.Name(), "main.go"))
		if err != nil {
			return err
		}
		dir := filepath.Join(work, e.Name())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), src, 0o644); err != nil {
			return err
		}
		cmd := exec.Command(filepath.Join(env.goroot, "bin", "go"), "build", "-trimpath", "-buildvcs=false", "-o", "/dev/null", ".")
		cmd.Dir = dir
		cmd.Env = []string{
			"GOROOT=" + env.goroot, "PATH=" + env.goroot + "/bin:/usr/bin:/bin", "HOME=" + work, "GOPATH=" + filepath.Join(work, "gopath"),
			"GOCACHE=" + env.seed, "TMPDIR=" + os.TempDir(), "GO111MODULE=off", "CGO_ENABLED=0", "GOTOOLCHAIN=local",
			"GOTELEMETRY=off", "GOENV=off", "GOFLAGS=", "GOPROXY=off", "TZ=UTC", "LANG=C.UTF-8",
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %v\n%s", e.Name(), err, out)
		}
	}
	return filepath.Walk(env.seed, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(p, 0o755)
		}
		return os.Chmod(p, 0o644)
	})
}

// ---- the runner process ----

type runnerProc struct {
	cmd     *exec.Cmd
	logPath string
	done    chan struct{}
	code    int
}

// startRunner starts the runner binary (as root, like the pod's PID 1) and waits until it is
// healthy and ready.
func startRunner(t testing.TB, extraEnv ...string) *runnerProc {
	t.Helper()
	logf, err := os.CreateTemp(env.dir, "runner-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(env.bin)
	cmd.Env = append(os.Environ(),
		"RUNNER_MODE=dev", "RUNNER_ADDR="+addr, "RUNNER_TOKEN_FILE="+env.tokenFile, "RUNNER_JAIL_DIR="+env.jailDir,
		"RUNNER_TESTGO_GOROOT="+env.goroot, "RUNNER_TESTGO_GOCACHE="+env.seed, "RUNNER_DRAIN_TIMEOUT=5s", "LOG_LEVEL=info")
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.Stdout, cmd.Stderr = logf, logf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &runnerProc{cmd: cmd, logPath: logf.Name(), done: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		logf.Close()
		p.code = cmd.ProcessState.ExitCode()
		_ = err
		close(p.done)
	}()
	deadline := time.Now().Add(180 * time.Second)
	for {
		select {
		case <-p.done:
			t.Fatalf("the runner exited during startup (code %d):\n%s", p.code, p.tail(80))
		default:
		}
		if code, _ := get(t, "/readyz", false); code == http.StatusOK {
			return p
		}
		if time.Now().After(deadline) {
			p.stop()
			t.Fatalf("the runner never became ready:\n%s", p.tail(80))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (p *runnerProc) alive() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// waitExit waits for the runner to exit by itself (a drain or rotation).
func (p *runnerProc) waitExit(t testing.TB, within time.Duration) int {
	t.Helper()
	select {
	case <-p.done:
		return p.code
	case <-time.After(within):
		t.Fatalf("the runner did not exit within %v:\n%s", within, p.tail(60))
		return -1
	}
}

func (p *runnerProc) stop() {
	if !p.alive() {
		return
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
	case <-time.After(30 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

func (p *runnerProc) tail(n int) string {
	b, _ := os.ReadFile(p.logPath)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// ensure returns a healthy shared runner, restarting it after a rotation.
func ensure(t testing.TB) *runnerProc {
	t.Helper()
	rtMu.Lock()
	defer rtMu.Unlock()
	if rt != nil && rt.alive() {
		if code, body := get(t, "/readyz", false); code == http.StatusOK && !bytes.Contains(body, []byte("draining")) {
			return rt
		}
		// Draining (a rotation): let it finish.
		select {
		case <-rt.done:
		case <-time.After(60 * time.Second):
			rt.stop()
		}
	}
	rt = startRunner(t)
	t.Cleanup(func() {
		if t.Failed() {
			rtMu.Lock()
			if rt != nil {
				t.Logf("runner log tail:\n%s", rt.tail(60))
			}
			rtMu.Unlock()
		}
	})
	return rt
}

// restart replaces the shared runner with one started with extra env.
func restart(t testing.TB, extraEnv ...string) *runnerProc {
	t.Helper()
	rtMu.Lock()
	defer rtMu.Unlock()
	if rt != nil {
		rt.stop()
	}
	rt = startRunner(t, extraEnv...)
	return rt
}

// ---- HTTP ----

func get(t testing.TB, path string, auth bool) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if auth {
		req.Header.Set("Authorization", "Bearer "+testToken)
	}
	c := http.Client{Timeout: 10 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func getJSON[T any](t testing.TB, path string) T {
	t.Helper()
	code, b := get(t, path, true)
	if code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, code, b)
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return v
}

type response struct {
	code   int
	header http.Header
	body   []byte
	res    *runnerapi.Result
}

// post sends a job and decodes the reply strictly. A 200 Result must pass judge's own check.
func post(ctx context.Context, t testing.TB, job *runnerapi.Job, inputs [][]byte) (*response, error) {
	var body bytes.Buffer
	if err := runnerapi.EncodeJob(&body, job, inputs); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/jobs", &body)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", runnerapi.ContentTypeJob)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := &response{code: resp.StatusCode, header: resp.Header, body: b}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		var res runnerapi.Result
		if err := dec.Decode(&res); err != nil {
			return nil, fmt.Errorf("decode the Result: %v: %s", err, b)
		}
		r.res = &res
		if resp.StatusCode == http.StatusOK {
			if err := runnerapi.ValidateResult(job, &res); err != nil {
				return nil, fmt.Errorf("judge's ValidateResult rejects the Result: %v\n%s", err, b)
			}
		}
	}
	return r, nil
}

// run posts a job that must answer 200 with no Infra.
func run(t testing.TB, job *runnerapi.Job, inputs [][]byte) *runnerapi.Result {
	t.Helper()
	r, err := post(context.Background(), t, job, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if r.code != http.StatusOK || r.res == nil {
		t.Fatalf("POST /v1/jobs: %d %s", r.code, r.body)
	}
	if r.res.Infra != nil {
		t.Fatalf("infra error %+v", r.res.Infra)
	}
	return r.res
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

// jobUIDProcs lists processes whose UID is in the job pool (there must be none between jobs).
func jobUIDProcs() []string {
	ents, _ := os.ReadDir("/proc")
	var out []string
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		f, err := os.Open(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if uid, ok := strings.CutPrefix(sc.Text(), "Uid:"); ok {
				fs := strings.Fields(uid)
				if len(fs) > 0 {
					if n, _ := strconv.Atoi(fs[0]); n >= uidLo && n < uidHi {
						out = append(out, e.Name()+":"+fs[0])
					}
				}
			}
		}
		f.Close()
	}
	return out
}

// leftoverCgroups lists cgroups under the slots other than s0/s1 and their idle canary.
func leftoverCgroups() []string {
	var out []string
	for _, s := range []string{"s0", "s1"} {
		ents, _ := os.ReadDir(filepath.Join(cgRoot, "slots", s))
		for _, e := range ents {
			if e.IsDir() && e.Name() != "canary" {
				out = append(out, s+"/"+e.Name())
			}
		}
	}
	return out
}

// jailMounts lists mounts under the jail dir in this (the container's) mount namespace.
func jailMounts() []string {
	b, _ := os.ReadFile("/proc/self/mountinfo")
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) > 4 && strings.HasPrefix(f[4], env.jailDir) {
			out = append(out, f[4])
		}
	}
	return out
}

func readKeyed(path string) map[string]int64 {
	b, _ := os.ReadFile(path)
	m := map[string]int64{}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, " ")
		if ok {
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			m[k] = n
		}
	}
	return m
}

// containerOOMs is the container's own (non-hierarchical) OOM count: INV-14 keeps it at 0.
func containerOOMs() int64 {
	return readKeyed(filepath.Join(cgRoot, "memory.events.local"))["oom_kill"]
}

// assertClean checks the cleanup invariants between jobs.
func assertClean(t testing.TB) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		procs, cgs, mounts := jobUIDProcs(), leftoverCgroups(), jailMounts()
		if len(procs) == 0 && len(cgs) == 0 && len(mounts) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("not clean after the job: survivors %v, cgroups %v, mounts %v", procs, cgs, mounts)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func jsonEncode(w io.Writer, v any) error { return json.NewEncoder(w).Encode(v) }

func itoa(n int) string { return strconv.Itoa(n) }

func metric(t testing.TB, name string, v any) {
	t.Helper()
	t.Logf("RUNNER-IT-METRIC %s=%v", name, v)
}
