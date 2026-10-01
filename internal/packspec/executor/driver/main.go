// Command driver is packlint's in-container program runner (sprint m3-02). It is public,
// stdlib-only code: packlint cross-compiles it for linux and mounts it read-only into every
// executor container (`docker run --network none --read-only …`), where it
//
//  1. copies the read-only Go std-cache seed (/seed) into the tmpfs (/w/gocache);
//  2. compiles the program in /src once (go build -trimpath -buildvcs=false, the harness
//     build environment);
//  3. measures a no-op program's peak RSS (the memory rule's baseline);
//  4. runs every case as its own process — harness mode: the case frame on fd 3, the result
//     frame read from fd 4; stdio mode: stdin and argv in, stdout out — with GOMAXPROCS=1,
//     a wall kill at 1.5·CPU + 500 ms, and CPU time and peak RSS from rusage;
//  5. prints one JSON result on stdout. Program output never reaches packlint's own
//     output: packlint judges it and prints ids and counts only.
//
// Usage (inside the container): driver /job/job.json
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type jobRun struct {
	Input string   `json:"input,omitempty"`
	Args  []string `json:"args,omitempty"`
}

type job struct {
	Mode          string   `json:"mode"`
	Files         []string `json:"files"`
	Runs          []jobRun `json:"runs"`
	CPUms         int64    `json:"cpu_ms"`
	StopAfterKill bool     `json:"stop_after_kill"`
	OutputCap     int64    `json:"output_cap"`
}

type runResult struct {
	Exit         int    `json:"exit"`
	Signal       string `json:"signal,omitempty"`
	CPUms        int64  `json:"cpu_ms"`
	WallMs       int64  `json:"wall_ms"`
	PeakKB       int64  `json:"peak_kb"`
	Killed       bool   `json:"killed,omitempty"`
	Skipped      bool   `json:"skipped,omitempty"`
	Output       []byte `json:"output,omitempty"`
	OutputCapped bool   `json:"output_capped,omitempty"`
}

type result struct {
	CompileOK   bool        `json:"compile_ok"`
	CompileDiag string      `json:"compile_diag,omitempty"`
	BaselineKB  int64       `json:"baseline_kb"`
	Runs        []runResult `json:"runs"`
	Error       string      `json:"error,omitempty"`
}

const (
	work    = "/w"
	compile = 5 * time.Minute
)

// goEnv is the harness build environment (internal/platform/harness.GoBuildEnv) plus the
// container's writable paths.
var goEnv = []string{
	"CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOTELEMETRY=off", "GOENV=off",
	"HOME=/w", "GOCACHE=/w/gocache", "GOPATH=/w/gopath", "TMPDIR=/w/tmp",
	"PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin",
}

func main() {
	res := run()
	_ = json.NewEncoder(os.Stdout).Encode(res)
}

func run() (res result) {
	if len(os.Args) != 2 {
		res.Error = "usage: driver <job.json>"
		return
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		res.Error = err.Error()
		return
	}
	var j job
	if err := json.Unmarshal(b, &j); err != nil {
		res.Error = "job: " + err.Error()
		return
	}
	if j.OutputCap <= 0 {
		j.OutputCap = 64 << 20
	}
	for _, d := range []string{"gocache", "gopath", "tmp", "build", "run"} {
		if err := os.MkdirAll(filepath.Join(work, d), 0o755); err != nil {
			res.Error = err.Error()
			return
		}
	}
	if ents, err := os.ReadDir("/seed"); err == nil && len(ents) > 0 {
		if out, err := exec.Command("cp", "-R", "/seed/.", "/w/gocache").CombinedOutput(); err != nil {
			res.Error = "seed copy: " + err.Error() + ": " + string(out)
			return
		}
	}
	var files []string
	for _, f := range j.Files {
		data, err := os.ReadFile(filepath.Join("/src", f))
		if err != nil {
			res.Error = err.Error()
			return
		}
		if err := os.WriteFile(filepath.Join(work, "build", f), data, 0o644); err != nil {
			res.Error = err.Error()
			return
		}
		files = append(files, f)
	}
	ok, diag := build(filepath.Join(work, "build"), "/w/bin", files)
	res.CompileOK, res.CompileDiag = ok, diag
	if !ok {
		return
	}
	res.BaselineKB = baseline()
	limit := time.Duration(j.CPUms) * time.Millisecond
	wall := limit*3/2 + 500*time.Millisecond
	killed := false
	for _, r := range j.Runs {
		if killed && j.StopAfterKill {
			res.Runs = append(res.Runs, runResult{Skipped: true})
			continue
		}
		rr := runOne(j.Mode, r, wall, j.OutputCap)
		killed = killed || rr.Killed
		res.Runs = append(res.Runs, rr)
	}
	return
}

func build(dir, out string, files []string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), compile)
	defer cancel()
	args := append([]string{"build", "-trimpath", "-buildvcs=false", "-o", out}, files...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = goEnv
	o, err := cmd.CombinedOutput()
	if err != nil {
		return false, truncate(string(o), 16<<10)
	}
	return true, ""
}

// baseline is a no-op Go program's peak RSS in this container.
func baseline() int64 {
	dir := filepath.Join(work, "noop")
	_ = os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		return 0
	}
	if ok, _ := build(dir, "/w/noop-bin", []string{"main.go"}); !ok {
		return 0
	}
	var best int64
	for i := 0; i < 3; i++ {
		cmd := exec.Command("/w/noop-bin")
		cmd.Env = runEnv()
		if err := cmd.Run(); err != nil {
			return 0
		}
		if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok && (best == 0 || ru.Maxrss < best) {
			best = ru.Maxrss
		}
	}
	return best
}

func runEnv() []string {
	return []string{"GOMAXPROCS=1", "TZ=UTC", "LANG=C.UTF-8", "HOME=/w/run", "PATH=/usr/bin:/bin"}
}

// capped is a bounded buffer.
type capped struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	max  int64
	over bool
}

func (c *capped) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if room := c.max - int64(c.buf.Len()); room < int64(len(p)) {
		if room > 0 {
			c.buf.Write(p[:room])
		}
		c.over = true
		return len(p), nil
	}
	return c.buf.Write(p)
}

func runOne(mode string, r jobRun, wall time.Duration, outCap int64) runResult {
	var rr runResult
	cmd := exec.Command("/w/bin", r.Args...)
	cmd.Dir = filepath.Join(work, "run")
	cmd.Env = runEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out := &capped{max: outCap}
	discard := &capped{max: 8 << 10}
	var input *os.File
	if r.Input != "" {
		f, err := os.Open(filepath.Join("/job", r.Input))
		if err != nil {
			rr.Exit, rr.Signal = -1, "input: "+err.Error()
			return rr
		}
		defer f.Close()
		input = f
	}
	var resR, resW *os.File
	switch mode {
	case "harness":
		var err error
		if resR, resW, err = os.Pipe(); err != nil {
			rr.Exit, rr.Signal = -1, err.Error()
			return rr
		}
		if input == nil {
			input, _ = os.Open(os.DevNull)
		}
		cmd.ExtraFiles = []*os.File{input, resW}
		cmd.Stdout, cmd.Stderr = discard, discard
	default:
		if input != nil {
			cmd.Stdin = input
		}
		cmd.Stdout, cmd.Stderr = out, discard
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		rr.Exit, rr.Signal = -1, "start: "+err.Error()
		return rr
	}
	var wg sync.WaitGroup
	if resW != nil {
		resW.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(out, resR)
			resR.Close()
		}()
	}
	var killedFlag atomic.Bool
	timer := time.AfterFunc(wall, func() {
		killedFlag.Store(true)
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	})
	err := cmd.Wait()
	timer.Stop()
	rr.Killed = killedFlag.Load()
	// Reap any stragglers of the case's process group.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	wg.Wait()
	rr.WallMs = time.Since(start).Milliseconds()
	if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		rr.CPUms = (ru.Utime.Sec+ru.Stime.Sec)*1000 + int64(ru.Utime.Usec+ru.Stime.Usec)/1000
		rr.PeakKB = ru.Maxrss
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		rr.Signal = ws.Signal().String()
		rr.Exit = -1
	} else {
		rr.Exit = cmd.ProcessState.ExitCode()
	}
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		rr.Signal = strings.TrimSpace(rr.Signal + " wait: " + err.Error())
	}
	rr.Output, rr.OutputCapped = out.buf.Bytes(), out.over
	return rr
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + fmt.Sprintf("\n… (%d more bytes)", len(s)-n)
}
