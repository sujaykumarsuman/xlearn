// Package executor is packlint's production packspec.Executor (sprint m3-02; t1 §7.2–7.3):
// every author- or AI-written program runs ONLY here, one container per program:
//
//	docker run --rm --network none --read-only --tmpfs /w:rw,exec,size=512m --cpus 1 \
//	  --memory 1g --pids-limit 128 --user 65534:65534 --security-opt no-new-privileges \
//	  -v <src>:/src:ro -v <job>:/job:ro -v <driver>:/xl/driver:ro \
//	  -v xlearn-packlint-gocache-<digest12>:/seed:ro <image@sha256:…> /xl/driver /job/job.json
//
// The images are pinned by digest: golang:1.26.8-bookworm (the runner's Go, t3 §6.1),
// gcc:14 and python:3.13-slim (syntax checks only, until m3-04's harnesses). The driver
// (./driver, public and stdlib-only) is cross-compiled for the docker host's architecture
// and compiles the program once, then spawns one process per case. The Go std library is
// pre-compiled once into a read-only volume per image digest (`packlint exec
// --warm-cache`) and copied into the tmpfs at container start, so a program compiles in
// about a second and no program can poison the seed.
package executor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sujaykumarsuman/xlearn/internal/packspec"
)

// Pinned images (index digests: multi-arch, so CI's amd64 and an Apple-silicon Mac pull the
// same pin). A bump changes tests.lock's header, which triggers the private CI's full run.
const (
	GoImage     = "golang:1.26.8-bookworm@sha256:a688600ca24f8a4d3ca77f95b0dd40704a9fc787c826660eb7ba0b641b8b175d"
	CppImage    = "gcc:14@sha256:9188ac751ca24431dc43dbd142a223c98ea74f01d2858e84d30ba342a0d67844"
	PythonImage = "python:3.13-slim@sha256:7c61056e61ac89e852de05f3dc6fa51a6dd2181797bceed46aa725dd7cb2cd3b"
)

// SeedRecipe versions the std-cache seed's content (bump it when warmProbe changes).
const SeedRecipe = "seed@1"

//go:embed driver/main.go
var driverSource []byte

// Docker is the docker-backed executor.
type Docker struct {
	// Stderr receives compile diagnostics when Verbose (they may quote pack source, so the
	// default is off and CI never sets it).
	Stderr  io.Writer
	Verbose bool
	// CacheDir holds the cross-compiled driver (default: the user cache dir).
	CacheDir string

	once      sync.Once
	driver    string
	goarch    string
	onceErr   error
	seedReady bool
}

// Images implements packspec.Executor.
func (d *Docker) Images() map[string]string {
	return map[string]string{"cpp": CppImage, "go": GoImage, "python": PythonImage}
}

// SeedVolume is the std-cache volume of the Go image.
func SeedVolume() string {
	_, digest, _ := strings.Cut(GoImage, "@sha256:")
	return "xlearn-packlint-gocache-" + digest[:12]
}

func (d *Docker) cacheDir() string {
	if d.CacheDir != "" {
		return d.CacheDir
	}
	if c, err := os.UserCacheDir(); err == nil {
		return filepath.Join(c, "xlearn-packlint")
	}
	return filepath.Join(os.TempDir(), "xlearn-packlint")
}

// docker runs the docker CLI and returns stdout.
func docker(ctx context.Context, stdin io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("docker %s: %v: %s", args[0], err, strings.TrimSpace(lastLines(stderr.String(), 5)))
	}
	return out, nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// prepare finds the docker host's architecture and cross-compiles the driver for it.
func (d *Docker) prepare(ctx context.Context) error {
	d.once.Do(func() {
		out, err := docker(ctx, nil, "version", "--format", "{{.Server.Arch}}")
		if err != nil {
			d.onceErr = fmt.Errorf("docker is required for packlint's executor: %w", err)
			return
		}
		d.goarch = strings.TrimSpace(string(out))
		switch d.goarch {
		case "amd64", "arm64":
		default:
			d.onceErr = fmt.Errorf("unsupported docker architecture %q", d.goarch)
			return
		}
		dir := d.cacheDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			d.onceErr = err
			return
		}
		src, err := os.MkdirTemp(dir, "driver-src-")
		if err != nil {
			d.onceErr = err
			return
		}
		defer os.RemoveAll(src)
		if err := os.WriteFile(filepath.Join(src, "main.go"), driverSource, 0o644); err != nil {
			d.onceErr = err
			return
		}
		bin := filepath.Join(dir, "driver-linux-"+d.goarch)
		cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", bin, "main.go")
		cmd.Dir = src
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+d.goarch, "CGO_ENABLED=0", "GOFLAGS=", "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			d.onceErr = fmt.Errorf("build the executor driver: %v: %s", err, out)
			return
		}
		d.driver = bin
		_, err = docker(ctx, nil, "volume", "inspect", SeedVolume())
		d.seedReady = err == nil
	})
	return d.onceErr
}

// tempDir is a fresh directory the container's user (65534) can read: os.MkdirTemp's 0700
// would hide a bind mount from it on Linux.
func tempDir(parent, pattern string) (string, error) {
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	return dir, os.Chmod(dir, 0o755)
}

// runArgs is the hardened `docker run` prefix.
func runArgs(user string) []string {
	return []string{"run", "--rm", "-i", "--network", "none", "--read-only",
		"--tmpfs", "/w:rw,exec,size=512m", "--tmpfs", "/tmp:rw,size=16m",
		"--cpus", "1", "--memory", "1g", "--memory-swap", "1g", "--pids-limit", "128",
		"--user", user, "--security-opt", "no-new-privileges", "--cap-drop", "ALL"}
}

type driverJob struct {
	Mode          string         `json:"mode"`
	Files         []string       `json:"files"`
	Runs          []driverJobRun `json:"runs"`
	CPUms         int64          `json:"cpu_ms"`
	StopAfterKill bool           `json:"stop_after_kill"`
	OutputCap     int64          `json:"output_cap"`
}

type driverJobRun struct {
	Input string   `json:"input,omitempty"`
	Args  []string `json:"args,omitempty"`
}

type driverResult struct {
	packspec.ExecResult
	Error string `json:"error,omitempty"`
}

// Execute implements packspec.Executor (Go programs).
func (d *Docker) Execute(ctx context.Context, p packspec.Program, runs []packspec.Run, lim packspec.Limits) (*packspec.ExecResult, error) {
	if p.Lang != "go" {
		return nil, fmt.Errorf("%s: only Go programs execute before m3-04 (pending(harness))", p.Label)
	}
	if err := d.prepare(ctx); err != nil {
		return nil, err
	}
	tmp, err := tempDir(d.cacheDir(), "job-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	srcDir, jobDir := filepath.Join(tmp, "src"), filepath.Join(tmp, "job")
	for _, dir := range []string{srcDir, jobDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	j := driverJob{Mode: string(p.Mode), CPUms: lim.CPUms, StopAfterKill: lim.StopAfterKill, OutputCap: lim.OutputCap}
	for _, f := range p.Files {
		if strings.ContainsAny(f.Name, `/\`) || strings.HasPrefix(f.Name, ".") {
			return nil, fmt.Errorf("%s: bad file name %q", p.Label, f.Name)
		}
		if err := os.WriteFile(filepath.Join(srcDir, f.Name), f.Data, 0o644); err != nil {
			return nil, err
		}
		j.Files = append(j.Files, f.Name)
	}
	for i, r := range runs {
		jr := driverJobRun{Args: r.Args}
		if r.Input != nil {
			jr.Input = fmt.Sprintf("in-%05d", i)
			if err := os.WriteFile(filepath.Join(jobDir, jr.Input), r.Input, 0o644); err != nil {
				return nil, err
			}
		}
		j.Runs = append(j.Runs, jr)
	}
	jb, _ := json.Marshal(j)
	if err := os.WriteFile(filepath.Join(jobDir, "job.json"), jb, 0o644); err != nil {
		return nil, err
	}
	args := append(runArgs("65534:65534"),
		"-v", srcDir+":/src:ro", "-v", jobDir+":/job:ro", "-v", d.driver+":/xl/driver:ro")
	if d.seedReady {
		args = append(args, "-v", SeedVolume()+":/seed:ro")
	}
	args = append(args, GoImage, "/xl/driver", "/job/job.json")
	out, err := docker(ctx, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p.Label, err)
	}
	var res driverResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("%s: driver output: %w", p.Label, err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("%s: driver: %s", p.Label, res.Error)
	}
	if res.CompileOK && len(res.Runs) != len(runs) {
		return nil, fmt.Errorf("%s: driver returned %d runs for %d", p.Label, len(res.Runs), len(runs))
	}
	if !res.CompileOK && d.Verbose && d.Stderr != nil {
		fmt.Fprintf(d.Stderr, "--- %s: compile output (--verbose) ---\n%s\n", p.Label, res.CompileDiag)
	}
	return &res.ExecResult, nil
}

// cppPrelude stands in for m3-04's xl_prelude.hpp in the syntax check only: the standard
// library and the node types the references may use.
const cppPrelude = `#include <bits/stdc++.h>
using namespace std;
struct ListNode { int val; ListNode *next; ListNode() : val(0), next(nullptr) {} ListNode(int x) : val(x), next(nullptr) {} ListNode(int x, ListNode *n) : val(x), next(n) {} };
struct TreeNode { int val; TreeNode *left; TreeNode *right; TreeNode() : val(0), left(nullptr), right(nullptr) {} TreeNode(int x) : val(x), left(nullptr), right(nullptr) {} TreeNode(int x, TreeNode *l, TreeNode *r) : val(x), left(l), right(r) {} };
class Node { public: int val; vector<Node*> neighbors; Node() : val(0) {} Node(int v) : val(v) {} Node(int v, vector<Node*> n) : val(v), neighbors(n) {} };
`

// SyntaxCheck implements packspec.Executor: g++ -fsyntax-only (with a stand-in prelude) or
// Python's compile(), in the pinned image, network-less and read-only.
func (d *Docker) SyntaxCheck(ctx context.Context, lang string, files []packspec.SourceFile) (bool, string, error) {
	if err := d.prepare(ctx); err != nil {
		return false, "", err
	}
	tmp, err := tempDir(d.cacheDir(), "syntax-")
	if err != nil {
		return false, "", err
	}
	defer os.RemoveAll(tmp)
	var image string
	var script strings.Builder
	switch lang {
	case "cpp":
		image = CppImage
		for i, f := range files {
			wrapper := fmt.Sprintf("check-%d.cpp", i)
			if err := os.WriteFile(filepath.Join(tmp, f.Name), f.Data, 0o644); err != nil {
				return false, "", err
			}
			if err := os.WriteFile(filepath.Join(tmp, wrapper), []byte(cppPrelude+"#include \"/src/"+f.Name+"\"\n"), 0o644); err != nil {
				return false, "", err
			}
			fmt.Fprintf(&script, "g++ -std=gnu++20 -fsyntax-only /src/%s || exit 1\n", wrapper)
		}
	case "python":
		image = PythonImage
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(tmp, f.Name), f.Data, 0o644); err != nil {
				return false, "", err
			}
			fmt.Fprintf(&script, "python3 -I -S -c 'import sys; compile(open(sys.argv[1], encoding=\"utf-8\").read(), sys.argv[1], \"exec\")' /src/%s || exit 1\n", f.Name)
		}
	default:
		return false, "", fmt.Errorf("no syntax check for %q", lang)
	}
	args := append(runArgs("65534:65534"), "-e", "HOME=/w", "-v", tmp+":/src:ro", image, "sh", "-c", script.String())
	out, err := docker(ctx, nil, args...)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(errors.Unwrap(err), &ee) || strings.Contains(err.Error(), "exit status") {
			if d.Verbose && d.Stderr != nil {
				fmt.Fprintf(d.Stderr, "--- %s syntax check (--verbose) ---\n%v\n", lang, err)
			}
			return false, string(out), nil
		}
		return false, "", err
	}
	return true, "", nil
}

// warmProbe imports the std packages the harness, references (the runner's lint allowlist,
// t3 §6.2), generators and validators use, so the seed holds them compiled.
const warmProbe = `package main

import (
	_ "bufio"
	_ "bytes"
	_ "cmp"
	_ "container/heap"
	_ "container/list"
	_ "container/ring"
	_ "encoding/binary"
	_ "encoding/json"
	_ "errors"
	_ "flag"
	_ "fmt"
	_ "io"
	_ "iter"
	_ "maps"
	_ "math"
	_ "math/bits"
	_ "math/rand/v2"
	_ "os"
	_ "slices"
	_ "sort"
	_ "strconv"
	_ "strings"
	_ "unicode"
	_ "unicode/utf16"
	_ "unicode/utf8"
)

func main() {}
`

// WarmCache builds the read-only std-cache volume for the pinned Go image (once per
// digest), or restores it from cacheDir/<volume>.tar (CI's actions/cache); after a fresh
// build it saves that tarball when cacheDir is set. It runs only public code (the probe
// above) as root in a network-less container, so the volume's files are root-owned and
// read-only to every program container.
func (d *Docker) WarmCache(ctx context.Context, cacheDir string, out io.Writer) error {
	vol := SeedVolume()
	marker := SeedRecipe + " " + GoImage
	if got, err := docker(ctx, nil, "run", "--rm", "--network", "none", "-v", vol+":/seed:ro", GoImage,
		"cat", "/seed/.xlearn-seed"); err == nil && strings.TrimSpace(string(got)) == marker {
		fmt.Fprintf(out, "packlint exec --warm-cache: %s is ready\n", vol)
		return nil
	}
	tarball := ""
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return err
		}
		abs, err := filepath.Abs(cacheDir)
		if err != nil {
			return err
		}
		cacheDir, tarball = abs, filepath.Join(abs, vol+".tar")
	}
	if _, err := docker(ctx, nil, "volume", "create", vol); err != nil {
		return err
	}
	if tarball != "" {
		if _, err := os.Stat(tarball); err == nil {
			script := "tar -xf /cache/" + vol + ".tar -C /seed && test \"$(cat /seed/.xlearn-seed)\" = \"" + marker + "\""
			if _, err := docker(ctx, nil, "run", "--rm", "--network", "none", "-v", vol+":/seed", "-v", cacheDir+":/cache:ro",
				GoImage, "sh", "-c", script); err == nil {
				fmt.Fprintf(out, "packlint exec --warm-cache: restored %s from %s\n", vol, filepath.Base(tarball))
				d.seedReady = true
				return nil
			}
			fmt.Fprintf(out, "packlint exec --warm-cache: %s is stale; rebuilding\n", filepath.Base(tarball))
		}
	}
	if err := os.MkdirAll(d.cacheDir(), 0o755); err != nil {
		return err
	}
	tmp, err := tempDir(d.cacheDir(), "warm-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(warmProbe), 0o644); err != nil {
		return err
	}
	env := []string{"-e", "CGO_ENABLED=0", "-e", "GOTOOLCHAIN=local", "-e", "GOPROXY=off", "-e", "GOFLAGS=-mod=readonly",
		"-e", "GOTELEMETRY=off", "-e", "GOENV=off", "-e", "HOME=/w", "-e", "GOCACHE=/seed", "-e", "GOPATH=/w/gopath", "-e", "TMPDIR=/w"}
	script := "rm -rf /seed/* /seed/.xlearn-seed; cp /probe/main.go /w/main.go && cd /w && " +
		"go build -trimpath -buildvcs=false -o /w/probe main.go && echo '" + marker + "' > /seed/.xlearn-seed && chmod -R a+rX /seed"
	args := []string{"run", "--rm", "--network", "none", "--read-only", "--tmpfs", "/w:rw,exec,size=512m",
		"--cpus", "2", "--memory", "2g", "--user", "0:0", "-v", vol + ":/seed", "-v", tmp + ":/probe:ro"}
	args = append(append(args, env...), GoImage, "sh", "-c", script)
	if _, err := docker(ctx, nil, args...); err != nil {
		return fmt.Errorf("warm the std cache: %w", err)
	}
	d.seedReady = true
	fmt.Fprintf(out, "packlint exec --warm-cache: built %s (%s)\n", vol, SeedRecipe)
	if tarball != "" {
		if _, err := docker(ctx, nil, "run", "--rm", "--network", "none", "-v", vol+":/seed:ro", "-v", cacheDir+":/cache",
			GoImage, "tar", "-cf", "/cache/"+vol+".tar", "-C", "/seed", "."); err != nil {
			return fmt.Errorf("save the std cache: %w", err)
		}
		fmt.Fprintf(out, "packlint exec --warm-cache: saved %s\n", filepath.Base(tarball))
	}
	return nil
}
