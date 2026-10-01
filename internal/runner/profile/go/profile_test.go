package goprofile

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile/proftest"
)

func TestInvariants(t *testing.T) {
	p := Profile()
	proftest.Invariants(t, p, proftest.Lists{Exec: execAllow, Compile: compileAllow, Published: spk02Exec, Additions: execAdditions, PublishedLen: 27})
	if !p.Exec.Seccomp.CloneThreadOnly || !p.Exec.Seccomp.PrctlSetVMAOnly || !p.Exec.Seccomp.Clone3ENOSYS {
		t.Error("go exec keeps spk-02's fixed rules: clone CLONE_THREAD only, prctl PR_SET_VMA only, clone3 ENOSYS")
	}
	if p.ArtifactMode != 0o111 || p.ArtifactName != "bin" {
		t.Errorf("go artifact %s %o, want bin 0111 (a static ELF needs no read bit)", p.ArtifactName, p.ArtifactMode)
	}
	if p.Compile.CPUms != 15000 || p.Compile.MemMB != 768 || p.Compile.Pids != 256 || p.Exec.Pids != 32 {
		t.Errorf("go limits: compile %d ms %d MiB pids %d, exec pids %d", p.Compile.CPUms, p.Compile.MemMB, p.Compile.Pids, p.Exec.Pids)
	}
	if p.TLMultiplier != 1 || p.Calibrated || p.Baseline != Name || p.Language != "go" {
		t.Errorf("go is the TL baseline: multiplier %v calibrated %v baseline %s", p.TLMultiplier, p.Calibrated, p.Baseline)
	}
	for _, e := range p.Exec.Env {
		if strings.HasPrefix(e, "GOMEMLIMIT=") {
			t.Error("GOMEMLIMIT stays unset")
		}
	}
}

// TestCompileEnv: GOROOT is set (no /proc in the jail), TMPDIR and GOPATH are on the compile's /w, HOME is unset,
// GOCACHE is the seed in place when one is installed (read-only bind, no overlay, no copy).
func TestCompileEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "go", "VERSION"), []byte("go1.26.8\ntime 2026-01-01\n"), 0o644)
	t.Setenv("RUNNER_TOOLCHAINS_DIR", dir)
	env := func(p *profile.Profile) map[string]string {
		m := map[string]string{}
		for _, e := range p.Compile.Env {
			k, v, _ := strings.Cut(e, "=")
			m[k] = v
		}
		return m
	}
	goroot := profile.Resolve(filepath.Join(dir, "go"))
	p := Profile()
	e := env(p)
	if e["GOROOT"] != goroot || e["TMPDIR"] != "/w" || e["GOPATH"] != "/w/gopath" || e["HOME"] != "" || e["GOCACHE"] != "/w/gocache" || e["CGO_ENABLED"] != "0" {
		t.Errorf("cold compile env %v", e)
	}
	if p.Toolchain != "go1.26.8" {
		t.Errorf("toolchain %q", p.Toolchain)
	}
	seed := filepath.Join(dir, "gocache")
	os.Mkdir(seed, 0o755)
	p = Profile()
	if env(p)["GOCACHE"] != profile.Resolve(seed) {
		t.Errorf("GOCACHE with a seed = %s", env(p)["GOCACHE"])
	}
	if !slices.Contains(p.ToolchainPaths, profile.Resolve(seed)) || !slices.ContainsFunc(p.Compile.Binds, func(b profile.Bind) bool { return b.Source == profile.Resolve(seed) && b.Target == "" }) {
		t.Errorf("the seed is bound read-only in place and hashed: binds %v paths %v", p.Compile.Binds, p.ToolchainPaths)
	}
	if slices.Contains(p.Compile.Argv, "-vet=off") || !slices.Contains(p.Compile.Argv, "-json") || !slices.Contains(p.Compile.Argv, "-trimpath") {
		t.Errorf("argv %v", p.Compile.Argv)
	}
}

func TestDiagnostics(t *testing.T) {
	out := []byte(`{"ImportPath":"command-line-arguments","Action":"build-output","Output":"# command-line-arguments\n"}
{"ImportPath":"command-line-arguments","Action":"build-output","Output":"./solution.go:4:9: undefined: y\n"}
{"ImportPath":"command-line-arguments","Action":"build-output","Output":"./zz_xl_harness.go:20:7: undefined: pairSum\n"}
{"ImportPath":"command-line-arguments","Action":"build-fail"}
solution.go:3:8: package os/exec is not in std
`)
	names := map[string]bool{"solution.go": true, "zz_xl_harness.go": true}
	d := Diagnostics(out, names)
	if len(d) != 3 || d[0].File != "solution.go" || d[0].Line != 4 || d[0].Col != 9 || d[1].File != "zz_xl_harness.go" || d[2].Line != 3 {
		t.Errorf("diags %+v", d)
	}
}

// TestFixSeed: every file gets SeedTime, the trim stamp and the index entries' times are fixed
// (two builds of one toolchain give the same tree), modes are world-readable.
func TestFixSeed(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "ab"), 0o700)
	entry := "v1 " + strings.Repeat("a", 64) + " " + strings.Repeat("b", 64) + " " + "                1234" + " " + "         17592186044" + "\n"
	if len(entry) != entrySize {
		t.Fatalf("entry is %d bytes, want %d", len(entry), entrySize)
	}
	os.WriteFile(filepath.Join(dir, "ab", "x-a"), []byte(entry), 0o600)
	os.WriteFile(filepath.Join(dir, "ab", "x-d"), []byte("object"), 0o600)
	if err := fixSeed(dir); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "ab", "x-a"))
	if !strings.HasSuffix(string(b), strconv.FormatInt(SeedTime.UnixNano(), 10)+"\n") || len(b) != entrySize {
		t.Errorf("entry time not fixed: %q", b)
	}
	trim, _ := os.ReadFile(filepath.Join(dir, "trim.txt"))
	if string(trim) != strconv.FormatInt(SeedTime.Unix(), 10) {
		t.Errorf("trim.txt %q", trim)
	}
	filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if !fi.ModTime().Equal(SeedTime) {
			t.Errorf("%s mtime %v", p, fi.ModTime())
		}
		if want := os.FileMode(0o644); !fi.IsDir() && fi.Mode().Perm() != want {
			t.Errorf("%s mode %o", p, fi.Mode().Perm())
		}
		return nil
	})
	if !SeedTime.After(time.Now().Add(50 * 365 * 24 * time.Hour)) {
		t.Error("SeedTime must be far in the future")
	}
}

func TestBuildSeedRefusesNonEmpty(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x"), nil, 0o644)
	if _, err := BuildSeed("/nonexistent", dir); err == nil {
		t.Error("a non-empty -out must be refused")
	}
	out := filepath.Join(t.TempDir(), "seed")
	if _, err := BuildSeed("/nonexistent-goroot", out); err == nil {
		t.Error("a failing build must fail")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("a failed seed leaves nothing at -out")
	}
	ents, _ := os.ReadDir(filepath.Dir(out))
	if len(ents) != 0 {
		t.Errorf("a failed seed leaves no partial directory: %v", ents)
	}
}
