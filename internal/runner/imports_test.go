package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	module     = "github.com/sujaykumarsuman/xlearn"
	goSandbox  = "github.com/criyle/go-sandbox"
	testgoPkg  = module + "/internal/runner/profile/testgo"
	runnerPkgs = module + "/internal/runner"
)

// allowedSandbox are the only go-sandbox packages the runner may use (ADR-0030 §1: pinned in
// go.mod, import-restricted). Never go-sandbox/container: it forces a user namespace and keeps
// ambient SYS_ADMIN (t3 §4).
var allowedSandbox = map[string]bool{
	goSandbox + "/pkg/forkexec":       true,
	goSandbox + "/pkg/forkexec/vfork": true,
	goSandbox + "/pkg/mount":          true,
	goSandbox + "/pkg/rlimit":         true,
	goSandbox + "/pkg/seccomp":        true,
	goSandbox + "/runner":             true, // pkg/rlimit's size type
}

// deps lists "importpath imports…" for every package in pkgs and their dependencies.
func deps(t *testing.T, tags string, pkgs ...string) map[string][]string {
	t.Helper()
	// -e: the module root embeds web/dist, which a lane without the SPA build lacks.
	args := []string{"list", "-e", "-deps", "-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	cmd := exec.Command("go", append(args, pkgs...)...)
	cmd.Dir = filepath.Join("..", "..")
	// The jail is Linux-only: judge the Linux/amd64 build graph whatever the host.
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	m := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(line)
		m[f[0]] = f[1:]
	}
	return m
}

// TestImportGuards: nothing imports go-sandbox/container; only internal/runner/... imports
// go-sandbox at all, and only the allowed packages; internal/runner imports no other service;
// runnerapi stays a leaf; release builds never link the test profiles.
func TestImportGuards(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list over the module")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	for _, tags := range []string{"", "runner_it"} {
		all := deps(t, tags, "./...")
		for pkg, imps := range all {
			if strings.HasPrefix(pkg, goSandbox+"/container") {
				t.Errorf("[tags=%q] %s is in the build graph", tags, pkg)
			}
			for _, imp := range imps {
				if strings.HasPrefix(imp, goSandbox) && !strings.HasPrefix(pkg, goSandbox) {
					if !strings.HasPrefix(pkg, runnerPkgs+"/") && pkg != runnerPkgs {
						t.Errorf("[tags=%q] %s imports %s: only internal/runner/... may use go-sandbox", tags, pkg, imp)
					}
					if !allowedSandbox[imp] {
						t.Errorf("[tags=%q] %s imports %s: not an allowed go-sandbox package", tags, pkg, imp)
					}
				}
				if strings.HasPrefix(pkg, runnerPkgs) && strings.HasPrefix(imp, module+"/internal/") &&
					!strings.HasPrefix(imp, runnerPkgs) && imp != module+"/internal/platform/runnerapi" &&
					imp != module+"/internal/platform/slogx" {
					t.Errorf("[tags=%q] %s imports %s: the runner imports no other service (ADR-0005)", tags, pkg, imp)
				}
				if pkg == module+"/internal/platform/runnerapi" && strings.Contains(strings.SplitN(imp, "/", 2)[0], ".") {
					t.Errorf("runnerapi imports %s: it is stdlib-only", imp)
				}
			}
		}
	}
	// No release build contains testgo@0; the runner_it build does (the guard is live).
	if _, ok := deps(t, "", "./cmd/runner")[testgoPkg]; ok {
		t.Errorf("a release build of cmd/runner links %s", testgoPkg)
	}
	if _, ok := deps(t, "runner_it", "./cmd/runner")[testgoPkg]; !ok {
		t.Errorf("the runner_it build of cmd/runner should link %s", testgoPkg)
	}
}
