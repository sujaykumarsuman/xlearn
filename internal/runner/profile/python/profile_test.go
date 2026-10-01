package python

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile/proftest"
)

func TestInvariants(t *testing.T) {
	p := Profile()
	proftest.Invariants(t, p, proftest.Lists{Exec: execAllow, Compile: compileAllow, Published: spk02Exec, Additions: execAdditions, PublishedLen: 39})
	if !p.Exec.Seccomp.CloneThreadOnly || slices.Contains(p.Exec.Seccomp.Allow, "clone") {
		t.Error("python: clone only with CLONE_THREAD (os.fork dies)")
	}
	if p.ArtifactMode != 0o444 || p.ArtifactName != "app.pyz" {
		t.Errorf("python artifact %s %o, want app.pyz 0444 (the interpreter reads it)", p.ArtifactName, p.ArtifactMode)
	}
	if p.Compile.CPUms != 15000 || p.Compile.MemMB != 256 || p.Compile.Pids != 16 || p.Exec.Pids != 4 || !p.Exec.StackFromMemory {
		t.Errorf("python limits: compile %d ms %d MiB pids %d; exec pids %d", p.Compile.CPUms, p.Compile.MemMB, p.Compile.Pids, p.Exec.Pids)
	}
	if !slices.Contains(p.Exec.Env, "PYTHONHASHSEED=0") || slices.Contains(p.Exec.Argv, "-I") || slices.Contains(p.Exec.Argv, "-E") {
		t.Errorf("python exec: PYTHONHASHSEED=0 must apply (no -I/-E): %v %v", p.Exec.Argv, p.Exec.Env)
	}
	for _, b := range p.Exec.Binds {
		if b.Source == "/usr" {
			t.Error("the python exec jail sees the interpreter, its stdlib and the shared libraries, never all of /usr")
		}
	}
}

func TestDiagnostics(t *testing.T) {
	out := []byte(`{"file": "solution.py", "line": 3, "col": 9, "msg": "SyntaxError: invalid syntax"}
{"file": "__main__.py", "line": 6, "col": 1, "msg": "solution.py does not define class Solution"}
{"file": "/usr/lib/python3.13/x.py", "line": 1, "col": 1, "msg": "not ours"}
`)
	d := Diagnostics(out, map[string]bool{"solution.py": true, "__main__.py": true, "xl_prelude.py": true})
	if len(d) != 2 || d[0].File != "solution.py" || d[0].Line != 3 || d[0].Col != 9 || d[1].File != "__main__.py" {
		t.Errorf("diags %+v", d)
	}
}

// runDriver runs the compile driver on files in a temp dir with the local python3 (skipped
// when there is none; the runner-it lane runs it in the jail).
func runDriver(t *testing.T, files map[string]string) (string, []byte, error) {
	t.Helper()
	if exec.Command("python3", "-c", "import sys; assert sys.version_info >= (3, 10)").Run() != nil {
		t.Skip("no python3 >= 3.10")
	}
	dir := t.TempDir()
	for n, s := range files {
		os.WriteFile(filepath.Join(dir, n), []byte(s), 0o644)
	}
	out := filepath.Join(dir, "app.pyz")
	cmd := exec.Command("python3", "-I", "-S", "-B", "-c", CompileDriver, out)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return out, stderr.Bytes(), err
}

func harnessFiles(t *testing.T, learner string) map[string]string {
	t.Helper()
	sig, err := harness.ParseSig(harness.FuncJSON, &course.Signature{Mode: "function", Name: "pairSum",
		Params: []course.Param{{Name: "nums", Type: "int[]"}, {Name: "target", Type: "int"}}, Returns: "int[]"})
	if err != nil {
		t.Fatal(err)
	}
	gen, err := harness.Generate("python", harness.FuncJSON, sig)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{harness.PythonLearnerFile: learner}
	for _, f := range gen {
		m[f.Path] = string(f.Data)
	}
	return m
}

func TestCompileDriver(t *testing.T) {
	ok := "class Solution:\n    def pairSum(self, nums, target=0):\n        return []\n"
	out, stderr, err := runDriver(t, harnessFiles(t, ok))
	if err != nil {
		t.Fatalf("driver: %v\n%s", err, stderr)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if f.Method != zip.Store || !f.Modified.IsZero() && f.Modified.Year() != 1980 {
			t.Errorf("%s: method %d time %v", f.Name, f.Method, f.Modified)
		}
	}
	zr.Close()
	if !slices.Equal(names, []string{"__main__.py", "solution.py", "xl_prelude.py"}) {
		t.Errorf("zipapp entries %v", names)
	}
	// Deterministic: a second build is byte-identical.
	out2, _, _ := runDriver(t, harnessFiles(t, ok))
	a, _ := os.ReadFile(out)
	b, _ := os.ReadFile(out2)
	if sha256.Sum256(a) != sha256.Sum256(b) {
		t.Error("the zipapp is not deterministic")
	}

	cases := map[string]struct{ src, file, msg string }{
		"syntax":    {"class Solution:\n    def pairSum(self, nums, target)\n        return []\n", "solution.py", "SyntaxError"},
		"indent":    {"class Solution:\ndef pairSum(self, nums, target):\n    return []\n", "solution.py", "IndentationError"},
		"no class":  {"def pairSum(nums, target):\n    return []\n", "__main__.py", "does not define class Solution"},
		"no method": {"class Solution:\n    def pairSums(self, nums, target):\n        return []\n", "__main__.py", "no method pairSum"},
		"arity":     {"class Solution:\n    def pairSum(self, nums):\n        return []\n", "__main__.py", "does not take 2"},
	}
	for name, c := range cases {
		_, stderr, err := runDriver(t, harnessFiles(t, c.src))
		d := Diagnostics(stderr, map[string]bool{"solution.py": true, "__main__.py": true, "xl_prelude.py": true})
		if err == nil || len(d) == 0 || d[0].File != c.file || !strings.Contains(d[0].Msg, c.msg) {
			t.Errorf("%s: err %v diags %+v\n%s", name, err, d, stderr)
		}
	}
	// Accepted unchecked: a base class (methods may be inherited), *args, a decorated method.
	for _, src := range []string{
		"class Base:\n    def pairSum(self, *a):\n        return []\n\n\nclass Solution(Base):\n    pass\n",
		"class Solution:\n    def pairSum(self, *args):\n        return []\n",
		"import functools\n\n\nclass Solution:\n    @functools.cache\n    def pairSum(self, a):\n        return []\n",
	} {
		if _, stderr, err := runDriver(t, harnessFiles(t, src)); err != nil {
			t.Errorf("accepted shape refused: %v\n%s\n%s", err, src, stderr)
		}
	}
}
