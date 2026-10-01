// Package goprofile registers go@1.26, the reference language profile (t3 §6.2's go@1.26
// column; D20; m3-04 task 3):
//
//   - compile: `go build -json -trimpath -buildvcs=false -o /w/bin solution.go zz_xl_harness.go`
//     with the pinned go1.26 tarball bound read-only at its own path, GOROOT set (the jail has
//     no /proc), TMPDIR and GOPATH on the compile's /w (HOME unset), and GOCACHE = the
//     read-only GOCACHE seed IN PLACE (t3 §16.2 block 1: no overlay, no per-case copy; the go
//     command reads the dependencies' packages from the seed and ignores its failed writes for
//     the learner's); 15 s, 768 MiB, pids 256;
//   - exec: the static /job/bin under the amd64 `go` allowlist (t3 §16.2), KILL-default,
//     clone only with CLONE_THREAD, prctl only PR_SET_VMA, clone3 ENOSYS; GOMAXPROCS=1,
//     GOMEMLIMIT unset; pids 32.
//
// The toolchain path is $RUNNER_TOOLCHAINS_DIR/go (default /opt/xl/go, m3-15's image); the
// version pin is m3-15's Dockerfile.
package goprofile

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Name is the profile id.
const Name = "go@1.26"

// Paths inside the compile jail.
const (
	artifact = "/w/bin"
	work     = "/w"
)

// GOROOT is the profile's Go toolchain ($RUNNER_TOOLCHAINS_DIR/go, symlinks resolved).
func GOROOT() string { return profile.Resolve(filepath.Join(profile.ToolchainsDir(), "go")) }

// SeedDir is where the GOCACHE seed lives ($RUNNER_TOOLCHAINS_DIR/gocache).
func SeedDir() string { return profile.Resolve(filepath.Join(profile.ToolchainsDir(), "gocache")) }

// BuildFlags are the compile's `go build` flags that decide cache hits (the seed only hits
// when it was built with the same toolchain, flags and env).
var BuildFlags = []string{"-trimpath", "-buildvcs=false"}

// BuildEnv is the compile environment (and the seed recipe's): harness.GoBuildEnv plus the
// jail's paths. gocache is the read-only seed in place, or a cold cache on /w; TMPDIR and
// GOPATH sit on work. HOME stays unset (m3-03's hand-off): with a HOME the go command keeps
// telemetry state under $HOME/.config even with GOTELEMETRY=off, and its sidecar wants /proc.
func BuildEnv(goroot, gocache, work string) []string {
	return append(append([]string{}, harness.GoBuildEnv...),
		"GOROOT="+goroot, "GOCACHE="+gocache, "TMPDIR="+work, "GOPATH="+work+"/gopath",
		"PATH="+goroot+"/bin", "TZ=UTC", "LANG=C.UTF-8")
}

// ExecAllow and CompileAllow are this arch's exec and compile syscall lists
// (seccomp_<arch>.go).
func ExecAllow() []string    { return append([]string{}, execAllow...) }
func CompileAllow() []string { return append([]string{}, compileAllow...) }

// Profile builds go@1.26 for the current environment.
func Profile() *profile.Profile {
	goroot, seed := GOROOT(), SeedDir()
	gocache := work + "/gocache" // cold, when no seed is installed
	binds := []profile.Bind{{Source: goroot}}
	paths := []string{goroot}
	if fi, err := os.Stat(seed); err == nil && fi.IsDir() {
		gocache = seed
		binds = append(binds, profile.Bind{Source: seed})
		paths = append(paths, seed)
	}
	argv := append(append([]string{goroot + "/bin/go", "build", "-json"}, BuildFlags...),
		"-o", artifact, harness.GoLearnerFile, harness.GoHarnessFile)
	return &profile.Profile{
		Name:           Name,
		Language:       "go",
		Baseline:       Name,
		TLMultiplier:   1,
		Toolchain:      version(goroot),
		Harnesses:      append([]string{}, harness.Harnesses...),
		ArtifactName:   "bin",
		ArtifactMode:   0o111,
		ToolchainPaths: paths,
		Assets:         harness.Sources("go"),
		BaselineFiles: []runnerapi.File{
			{Path: harness.GoLearnerFile, Data: []byte("package main\n")},
			{Path: harness.GoHarnessFile, Data: []byte("package main\n\nfunc main() {}\n")},
		},
		Compile: profile.Compile{
			Argv:       argv,
			Env:        BuildEnv(goroot, gocache, work),
			Binds:      binds,
			OutputPath: artifact,
			CPUms:      15000,
			MemMB:      768,
			Pids:       256,
			WMB:        256,
			WInodes:    16384,
			Seccomp: profile.Seccomp{
				Default:      profile.DefaultENOSYS,
				Allow:        seccomp.Union(compileAllow, seccomp.CompileInitExtra),
				Kill:         seccomp.Dangerous,
				Clone3ENOSYS: true,
			},
		},
		Exec: profile.Exec{
			Argv:       []string{"/job/bin"},
			Env:        []string{"GOMAXPROCS=1", "TZ=UTC", "LANG=C.UTF-8", "GOTRACEBACK=single"},
			Pids:       32,
			FSizeBytes: 1 << 20,
			NoFile:     64,
			WMB:        64,
			WInodes:    4096,
			Seccomp: profile.Seccomp{
				Default:         profile.DefaultKill,
				Allow:           seccomp.Without(execAllow, "clone", "prctl"),
				Kill:            seccomp.Dangerous,
				Clone3ENOSYS:    true,
				CloneThreadOnly: true,
				PrctlSetVMAOnly: true,
			},
		},
		Detect:      func() (string, error) { return version(goroot), nil },
		Diagnostics: Diagnostics,
	}
}

func init() { profile.Register(Profile()) }

// version is the toolchain's VERSION file's first line ("go1.26.8"), or "unknown".
func version(goroot string) string {
	b, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return "unknown"
	}
	v, _, _ := strings.Cut(string(b), "\n")
	return strings.TrimSpace(v)
}

// Diagnostics parses `go build -json` output: each build-output event's text is gc's
// "file:line:col: msg" lines; plain-text lines (the go command's own load errors) are parsed
// the same way.
func Diagnostics(out []byte, names map[string]bool) []runnerapi.Diag {
	var text bytes.Buffer
	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) > 0 && line[0] == '{' {
			var ev struct{ Action, Output string }
			if json.Unmarshal(line, &ev) == nil {
				if ev.Action == "build-output" {
					text.WriteString(ev.Output)
				}
				continue
			}
		}
		text.Write(line)
		text.WriteByte('\n')
	}
	return profile.TextDiags(text.Bytes(), names)
}
