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

// runnerMayImport are the module packages outside internal/runner the runner may import: the
// judge↔runner contract, its logger, and the shared leaf libraries (the harness templates the
// profiles hash and the front's panic-frame check; the front's name re-check).
var runnerMayImport = map[string]bool{
	module + "/internal/platform/runnerapi":      true,
	module + "/internal/platform/runnerapi/lint": true,
	module + "/internal/platform/harness":        true,
	module + "/internal/platform/slogx":          true,
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
					!strings.HasPrefix(imp, runnerPkgs) && !runnerMayImport[imp] {
					t.Errorf("[tags=%q] %s imports %s: the runner imports no other service (ADR-0005)", tags, pkg, imp)
				}
				if pkg == module+"/internal/platform/runnerapi" && strings.Contains(strings.SplitN(imp, "/", 2)[0], ".") {
					t.Errorf("runnerapi imports %s: it is stdlib-only", imp)
				}
			}
		}
		// The shared leaf libraries judge and packlint link (m3-04): nothing they pull in may
		// reach the jail, cgroups, go-sandbox or any runner package.
		for _, leaf := range []string{module + "/internal/platform/harness", module + "/internal/platform/runnerapi/lint"} {
			for pkg := range deps(t, tags, "./"+strings.TrimPrefix(leaf, module+"/")) {
				if strings.HasPrefix(pkg, runnerPkgs) || strings.HasPrefix(pkg, goSandbox) {
					t.Errorf("[tags=%q] %s pulls in %s: a shared leaf library never links runner code", tags, leaf, pkg)
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

// TestArm64ListsNeverInAmd64Builds: each profile's seccomp_arm64.go (the dev-VM lists, never in
// a release image) is linked only into arm64 builds, and its seccomp_amd64.go (t3 §16.2) only
// into amd64 builds; a release build of cmd/runner links all three launch profiles.
func TestArm64ListsNeverInAmd64Builds(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	files := func(arch, pkg string) string {
		cmd := exec.Command("go", "list", "-f", "{{join .GoFiles \" \"}}", pkg)
		cmd.Dir = filepath.Join("..", "..")
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list %s (%s): %v", pkg, arch, err)
		}
		return " " + strings.TrimSpace(string(out)) + " "
	}
	for _, p := range []string{"go", "cpp", "python"} {
		pkg := "./internal/runner/profile/" + p
		amd, arm := files("amd64", pkg), files("arm64", pkg)
		if strings.Contains(amd, " seccomp_arm64.go ") || !strings.Contains(amd, " seccomp_amd64.go ") {
			t.Errorf("%s amd64 build files: %s", pkg, amd)
		}
		if !strings.Contains(arm, " seccomp_arm64.go ") || strings.Contains(arm, " seccomp_amd64.go ") {
			t.Errorf("%s arm64 build files: %s", pkg, arm)
		}
		if _, ok := deps(t, "", "./cmd/runner")[runnerPkgs+"/profile/"+p]; !ok {
			t.Errorf("a release build of cmd/runner does not link the %s profile", p)
		}
	}
}
