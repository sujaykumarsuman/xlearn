//go:build runner_it

// Package testgo registers the test-only profiles testgo@0 and testgo-open@0 (build tag
// runner_it). No release build contains them: cmd/runner imports this package only under the
// tag, and internal/runner/imports_test.go guards it.
//
//   - testgo@0: compile = `go build` with the host Go toolchain bound read-only at its own path,
//     GOROOT set (the jail has no /proc, t3 §16.2) and TMPDIR=/w; the Go compile cache is a
//     read-only seed used in place when RUNNER_TESTGO_GOCACHE names one (t3 §16.2 block 1), else
//     a cold cache on /w. exec = the artifact under the real go@1.26 profile's exec allowlist
//     (goprofile.ExecAllow, t3 §16.2 block 2), KILL-default, with its fixed rules.
//   - testgo-open@0: the same, but its exec filter allows everything except the dangerous set
//     (minus socket and the keyring calls). It exists to test the namespace layer on its own —
//     cross-job markers, fork bombs against pids.max, orphan double-forks, network probes in an
//     empty netns — which the go allowlist would otherwise stop first with SIGSYS.
package testgo

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	goprofile "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Harness is the raw test harness: the program reads its case input on fd 3 and writes its
// result on fd 4 itself.
const Harness = "raw@0"

func init() {
	goroot := os.Getenv("RUNNER_TESTGO_GOROOT")
	if goroot == "" {
		goroot = "/usr/local/go"
	}
	seed := os.Getenv("RUNNER_TESTGO_GOCACHE")
	toolchain := "unknown"
	if b, err := os.ReadFile(filepath.Join(goroot, "VERSION")); err == nil {
		toolchain = strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
	}
	gocache := "/w/gocache"
	binds := []profile.Bind{{Source: goroot}}
	paths := []string{goroot}
	if seed != "" {
		gocache = seed
		binds = append(binds, profile.Bind{Source: seed})
		paths = append(paths, seed)
	}
	compile := profile.Compile{
		Argv: []string{goroot + "/bin/go", "build", "-trimpath", "-buildvcs=false", "-o", "/w/out", "."},
		Env: []string{
			"GOROOT=" + goroot, "PATH=" + goroot + "/bin", "GOPATH=/w/gopath", "GOCACHE=" + gocache,
			"TMPDIR=/w", "GOTMPDIR=/w", "GO111MODULE=off", "CGO_ENABLED=0", "GOTOOLCHAIN=local", "GOTELEMETRY=off",
			"GOENV=off", "GOFLAGS=", "GOPROXY=off", "TZ=UTC", "LANG=C.UTF-8",
		},
		Binds:      binds,
		OutputPath: "/w/out",
		CPUms:      15000,
		MemMB:      768,
		Pids:       256,
		WMB:        256,
		WInodes:    16384,
		Seccomp: profile.Seccomp{
			Default:      profile.DefaultENOSYS,
			Allow:        append(append([]string{}, goprofile.CompileAllow()...), seccomp.CompileInitExtra...),
			Kill:         seccomp.Dangerous,
			Clone3ENOSYS: true,
		},
	}
	exec := profile.Exec{
		Argv:       []string{"/job/bin"},
		Env:        []string{"GOMAXPROCS=1", "TZ=UTC", "LANG=C.UTF-8", "GOTRACEBACK=single"},
		Pids:       32,
		FSizeBytes: 1 << 20,
		NoFile:     64,
		WMB:        64,
		WInodes:    4096,
		Seccomp: profile.Seccomp{
			Default:         profile.DefaultKill,
			Allow:           seccomp.Without(goprofile.ExecAllow(), "clone", "prctl"),
			Kill:            seccomp.Dangerous,
			Clone3ENOSYS:    true,
			CloneThreadOnly: true,
			PrctlSetVMAOnly: true,
		},
	}
	base := []runnerapi.File{{Path: "main.go", Data: []byte("package main\n\nfunc main() {}\n")}}

	profile.Register(&profile.Profile{
		Name: "testgo@0", Toolchain: toolchain, Harnesses: []string{Harness}, ArtifactName: "bin", ArtifactMode: 0o111,
		ToolchainPaths: paths, BaselineFiles: base, Compile: compile, Exec: exec, TestOnly: true,
	})

	open := exec
	// A toolchain-style read-only bind, so runner_it can prove binds refuse writes (EROFS).
	open.Binds = []profile.Bind{{Source: goroot}}
	open.Seccomp = profile.Seccomp{
		Default:      profile.DefaultAllow,
		Kill:         seccomp.Without(seccomp.Dangerous, "socket", "keyctl", "add_key", "request_key"),
		Clone3ENOSYS: true,
	}
	profile.Register(&profile.Profile{
		Name: "testgo-open@0", Toolchain: toolchain, Harnesses: []string{Harness}, ArtifactName: "bin", ArtifactMode: 0o111,
		ToolchainPaths: paths, BaselineFiles: base, Compile: compile, Exec: open, TestOnly: true,
	})
}
