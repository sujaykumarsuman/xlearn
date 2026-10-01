//go:build linux && runner_it

// Package itrt is the runner-it lane's shared harness (m3-03's, made importable in m3-04): it
// builds the real runner binary (runner_it tags), prepares the toolchains layout
// (RUNNER_TOOLCHAINS_DIR: the Go toolchain and its GOCACHE seed, built once per machine with
// `runner seed-gocache`), runs the runner as root like the pod's PID 1 and talks to it over
// HTTP. internal/runner/it, internal/runner/profile and internal/platform/harness's jail tests
// use it; `make runner-it` runs those packages one at a time (-p 1), each with its own runner.
package itrt

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

// Fixed test settings.
const (
	CgRoot    = "/sys/fs/cgroup"
	Addr      = "127.0.0.1:18090"
	BaseURL   = "http://" + Addr
	TestToken = "runner-it-token-0123456789abcdef"
	// UIDLo..UIDHi is the job UID pool (spawner.uidPoolStart..+uidPoolSize).
	UIDLo, UIDHi = 10000, 60000
)

// Env is the prepared environment.
var Env struct {
	// Dir is this package run's temp dir; Root the module root.
	Dir, Root string
	// Bin is the runner binary (runner_it tags).
	Bin string
	// GOROOT is the host Go toolchain; Seed the testgo@0 corpus seed (RUNNER_TESTGO_GOCACHE).
	GOROOT, Seed string
	// Toolchains is RUNNER_TOOLCHAINS_DIR (go → GOROOT, gocache = the go@1.26 seed) and
	// GoSeedHash the seed's tree hash as `runner seed-gocache` printed it.
	Toolchains, GoSeedHash     string
	TokenFile, JailDir, Corpus string
}

var (
	mu sync.Mutex
	rt *Runner
)

// Setup prepares the environment (root, a delegated cgroup, the runner binary, the seeds).
func Setup() error {
	if os.Geteuid() != 0 {
		return errors.New("runner-it runs as root with a delegated cgroup (CI's privileged container, or `sudo make runner-it` on a Linux VM)")
	}
	if err := evacuateRootCgroup(); err != nil {
		return fmt.Errorf("evacuate the cgroup root: %w", err)
	}
	var err error
	if Env.Dir, err = os.MkdirTemp("", "runner-it-"); err != nil {
		return err
	}
	if err := os.Chmod(Env.Dir, 0o755); err != nil {
		return err
	}
	root, err := goEnv("GOMOD")
	if err != nil {
		return err
	}
	Env.Root = filepath.Dir(root)
	Env.Corpus = filepath.Join(Env.Root, "internal", "runner", "testdata", "corpus")
	if Env.GOROOT, err = goEnv("GOROOT"); err != nil {
		return err
	}
	Env.Bin = filepath.Join(Env.Dir, "runner")
	build := exec.Command("go", "build", "-tags", "runner_it", "-buildvcs=false", "-o", Env.Bin, "./cmd/runner")
	build.Dir = Env.Root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("build the runner: %v\n%s", err, out)
	}
	if err := buildTestgoSeed(); err != nil {
		return fmt.Errorf("build the testgo GOCACHE seed: %w", err)
	}
	if err := toolchains(); err != nil {
		return fmt.Errorf("the toolchains layout: %w", err)
	}
	Env.TokenFile = filepath.Join(Env.Dir, "token")
	if err := os.WriteFile(Env.TokenFile, []byte(TestToken+"\n"), 0o400); err != nil {
		return err
	}
	Env.JailDir = filepath.Join(Env.Dir, "jail")
	return nil
}

// Main is a package's TestMain body: Setup, the tests, then stop the runner.
func Main(m *testing.M) int {
	if err := Setup(); err != nil {
		fmt.Fprintln(os.Stderr, "runner-it setup:", err)
		return 1
	}
	code := m.Run()
	Teardown()
	return code
}

// Teardown stops the shared runner.
func Teardown() {
	mu.Lock()
	defer mu.Unlock()
	if rt != nil {
		rt.Stop()
	}
}

func goEnv(k string) (string, error) {
	out, err := exec.Command("go", "env", k).Output()
	return strings.TrimSpace(string(out)), err
}

// evacuateRootCgroup moves every process out of the container's cgroup root into a sibling
// leaf, so the runner can enable controllers there (the no-internal-process rule). In the pod
// the spawner is PID 1 and alone; CI's container also has the step shell and go test in it.
func evacuateRootCgroup() error {
	leaf := filepath.Join(CgRoot, "ci-init")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		b, err := os.ReadFile(filepath.Join(CgRoot, "cgroup.procs"))
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

// buildTestgoSeed fills a GOCACHE with the corpus's std dependencies using the testgo
// profile's exact flags and env; the compile jail then uses it read-only in place (t3 §16.2
// block 1).
func buildTestgoSeed() error {
	Env.Seed = filepath.Join(Env.Dir, "gocache-seed")
	work := filepath.Join(Env.Dir, "seed-work")
	ents, err := os.ReadDir(Env.Corpus)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.Name() == "badcompile" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(Env.Corpus, e.Name(), "main.go"))
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
		cmd := exec.Command(filepath.Join(Env.GOROOT, "bin", "go"), "build", "-trimpath", "-buildvcs=false", "-o", "/dev/null", ".")
		cmd.Dir = dir
		cmd.Env = []string{
			"GOROOT=" + Env.GOROOT, "PATH=" + Env.GOROOT + "/bin:/usr/bin:/bin", "HOME=" + work, "GOPATH=" + filepath.Join(work, "gopath"),
			"GOCACHE=" + Env.Seed, "TMPDIR=" + os.TempDir(), "GO111MODULE=off", "CGO_ENABLED=0", "GOTOOLCHAIN=local",
			"GOTELEMETRY=off", "GOENV=off", "GOFLAGS=", "GOPROXY=off", "TZ=UTC", "LANG=C.UTF-8",
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %v\n%s", e.Name(), err, out)
		}
	}
	return filepath.Walk(Env.Seed, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(p, 0o755)
		}
		return os.Chmod(p, 0o644)
	})
}

// toolchains lays out RUNNER_TOOLCHAINS_DIR once per machine and Go version: go → the host
// GOROOT, and gocache = the go@1.26 seed from `runner seed-gocache` (m3-15's recipe; ~a
// minute of `go build std`, then reused by every package's run).
func toolchains() error {
	ver, err := goEnv("GOVERSION")
	if err != nil {
		return err
	}
	Env.Toolchains = filepath.Join(os.TempDir(), "xl-runner-it-toolchains-"+ver)
	if err := os.MkdirAll(Env.Toolchains, 0o755); err != nil {
		return err
	}
	link := filepath.Join(Env.Toolchains, "go")
	if _, err := os.Lstat(link); err != nil {
		if err := os.Symlink(Env.GOROOT, link); err != nil {
			return err
		}
	}
	seed := filepath.Join(Env.Toolchains, "gocache")
	hashFile := filepath.Join(Env.Toolchains, "gocache.sha256")
	if b, err := os.ReadFile(hashFile); err == nil {
		Env.GoSeedHash = strings.TrimSpace(string(b))
		return nil
	}
	_ = os.RemoveAll(seed)
	t0 := time.Now()
	cmd := exec.Command(Env.Bin, "seed-gocache", "-out", seed)
	cmd.Env = append(os.Environ(), "RUNNER_TOOLCHAINS_DIR="+Env.Toolchains)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("runner seed-gocache: %v\n%s", err, stderr.Bytes())
	}
	Env.GoSeedHash = strings.TrimSpace(string(out))
	if len(Env.GoSeedHash) != 64 {
		return fmt.Errorf("runner seed-gocache printed %q, want a tree hash", out)
	}
	fmt.Fprintf(os.Stderr, "RUNNER-IT-METRIC go_seed_build_s=%.1f go_seed_sha256=%s\n", time.Since(t0).Seconds(), Env.GoSeedHash)
	return os.WriteFile(hashFile, []byte(Env.GoSeedHash+"\n"), 0o644)
}

// ---- the runner process ----

// Runner is one runner process.
type Runner struct {
	cmd     *exec.Cmd
	logPath string
	done    chan struct{}
	code    int
}

// Start starts the runner binary (as root, like the pod's PID 1) and waits until it is
// healthy and ready.
func Start(t testing.TB, extraEnv ...string) *Runner {
	t.Helper()
	logf, err := os.CreateTemp(Env.Dir, "runner-*.log")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(Env.Bin)
	cmd.Env = append(os.Environ(),
		"RUNNER_MODE=dev", "RUNNER_ADDR="+Addr, "RUNNER_TOKEN_FILE="+Env.TokenFile, "RUNNER_JAIL_DIR="+Env.JailDir,
		"RUNNER_TESTGO_GOROOT="+Env.GOROOT, "RUNNER_TESTGO_GOCACHE="+Env.Seed, "RUNNER_TOOLCHAINS_DIR="+Env.Toolchains,
		"RUNNER_DRAIN_TIMEOUT=5s", "LOG_LEVEL=info")
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.Stdout, cmd.Stderr = logf, logf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &Runner{cmd: cmd, logPath: logf.Name(), done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		logf.Close()
		p.code = cmd.ProcessState.ExitCode()
		close(p.done)
	}()
	deadline := time.Now().Add(300 * time.Second)
	for {
		select {
		case <-p.done:
			t.Fatalf("the runner exited during startup (code %d):\n%s", p.code, p.Tail(80))
		default:
		}
		if code, _ := Get(t, "/readyz", false); code == http.StatusOK {
			return p
		}
		if time.Now().After(deadline) {
			p.Stop()
			t.Fatalf("the runner never became ready:\n%s", p.Tail(80))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Signal sends sig to the runner process.
func (p *Runner) Signal(sig os.Signal) error { return p.cmd.Process.Signal(sig) }

// Alive reports whether the process still runs.
func (p *Runner) Alive() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// WaitExit waits for the runner to exit by itself (a drain or rotation).
func (p *Runner) WaitExit(t testing.TB, within time.Duration) int {
	t.Helper()
	select {
	case <-p.done:
		return p.code
	case <-time.After(within):
		t.Fatalf("the runner did not exit within %v:\n%s", within, p.Tail(60))
		return -1
	}
}

// Stop sends SIGTERM and waits (SIGKILL after 30 s).
func (p *Runner) Stop() {
	if !p.Alive() {
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

// Tail is the last n log lines.
func (p *Runner) Tail(n int) string {
	b, _ := os.ReadFile(p.logPath)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// Log is the whole runner log.
func (p *Runner) Log() string {
	b, _ := os.ReadFile(p.logPath)
	return string(b)
}

// Ensure returns a healthy shared runner, restarting it after a rotation.
func Ensure(t testing.TB) *Runner {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if rt != nil && rt.Alive() {
		if code, body := Get(t, "/readyz", false); code == http.StatusOK && !bytes.Contains(body, []byte("draining")) {
			return rt
		}
		// Draining (a rotation): let it finish.
		select {
		case <-rt.done:
		case <-time.After(60 * time.Second):
			rt.Stop()
		}
	}
	rt = Start(t)
	t.Cleanup(func() {
		if t.Failed() {
			mu.Lock()
			if rt != nil {
				t.Logf("runner log tail:\n%s", rt.Tail(60))
			}
			mu.Unlock()
		}
	})
	return rt
}

// Restart replaces the shared runner with one started with extra env.
func Restart(t testing.TB, extraEnv ...string) *Runner {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if rt != nil {
		rt.Stop()
	}
	rt = Start(t, extraEnv...)
	return rt
}

// ---- HTTP ----

// Get is a GET (with the bearer token when auth).
func Get(t testing.TB, path string, auth bool) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, BaseURL+path, nil)
	if auth {
		req.Header.Set("Authorization", "Bearer "+TestToken)
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

// GetJSON is an authed GET decoded into T (200 required).
func GetJSON[T any](t testing.TB, path string) T {
	t.Helper()
	code, b := Get(t, path, true)
	if code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, code, b)
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return v
}

// Response is one POST /v1/jobs reply.
type Response struct {
	Code   int
	Header http.Header
	Body   []byte
	Res    *runnerapi.Result
}

// Post sends a job and decodes the reply strictly. A 200 Result must pass judge's own check.
func Post(ctx context.Context, t testing.TB, job *runnerapi.Job, inputs [][]byte) (*Response, error) {
	var body bytes.Buffer
	if err := runnerapi.EncodeJob(&body, job, inputs); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+"/v1/jobs", &body)
	req.Header.Set("Authorization", "Bearer "+TestToken)
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
	r := &Response{Code: resp.StatusCode, Header: resp.Header, Body: b}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		var res runnerapi.Result
		if err := dec.Decode(&res); err != nil {
			return nil, fmt.Errorf("decode the Result: %v: %s", err, b)
		}
		r.Res = &res
		if resp.StatusCode == http.StatusOK {
			if err := runnerapi.ValidateResult(job, &res); err != nil {
				return nil, fmt.Errorf("judge's ValidateResult rejects the Result: %v\n%s", err, b)
			}
		}
	}
	return r, nil
}

// Run posts a job that must answer 200 with no Infra.
func Run(t testing.TB, job *runnerapi.Job, inputs [][]byte) *runnerapi.Result {
	t.Helper()
	r, err := Post(context.Background(), t, job, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != http.StatusOK || r.Res == nil {
		t.Fatalf("POST /v1/jobs: %d %s", r.Code, r.Body)
	}
	if r.Res.Infra != nil {
		t.Fatalf("infra error %+v", r.Res.Infra)
	}
	return r.Res
}

// ---- host-side invariants ----

// JobUIDProcs lists processes whose UID is in the job pool (there must be none between jobs).
func JobUIDProcs() []string {
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
					if n, _ := strconv.Atoi(fs[0]); n >= UIDLo && n < UIDHi {
						out = append(out, e.Name()+":"+fs[0])
					}
				}
			}
		}
		f.Close()
	}
	return out
}

// LeftoverCgroups lists cgroups under the slots other than s0/s1 and their idle canary.
func LeftoverCgroups() []string {
	var out []string
	for _, s := range []string{"s0", "s1"} {
		ents, _ := os.ReadDir(filepath.Join(CgRoot, "slots", s))
		for _, e := range ents {
			if e.IsDir() && e.Name() != "canary" {
				out = append(out, s+"/"+e.Name())
			}
		}
	}
	return out
}

// JailMounts lists mounts under the jail dir in this (the container's) mount namespace.
func JailMounts() []string {
	b, _ := os.ReadFile("/proc/self/mountinfo")
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) > 4 && strings.HasPrefix(f[4], Env.JailDir) {
			out = append(out, f[4])
		}
	}
	return out
}

// ReadKeyed reads a cgroup "key value" file.
func ReadKeyed(path string) map[string]int64 {
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

// ContainerOOMs is the container's own (non-hierarchical) OOM count: INV-14 keeps it at 0.
func ContainerOOMs() int64 {
	return ReadKeyed(filepath.Join(CgRoot, "memory.events.local"))["oom_kill"]
}

// AssertClean checks the cleanup invariants between jobs.
func AssertClean(t testing.TB) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		procs, cgs, mounts := JobUIDProcs(), LeftoverCgroups(), JailMounts()
		if len(procs) == 0 && len(cgs) == 0 && len(mounts) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("not clean after the job: survivors %v, cgroups %v, mounts %v", procs, cgs, mounts)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Metric logs a RUNNER-IT-METRIC line (the lane's numbers for the PR and the decisions log).
func Metric(t testing.TB, name string, v any) {
	t.Helper()
	t.Logf("RUNNER-IT-METRIC %s=%v", name, v)
}
