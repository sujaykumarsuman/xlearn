package cpp

import (
	"slices"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/runner/profile/proftest"
)

func TestInvariants(t *testing.T) {
	p := Profile()
	proftest.Invariants(t, p, proftest.Lists{Exec: execAllow, Compile: compileAllow, Published: spk02Exec, Additions: execAdditions, PublishedLen: 19})
	if slices.Contains(p.Exec.Seccomp.Allow, "clone") || p.Exec.Seccomp.CloneThreadOnly {
		t.Error("a static single-threaded C++ program gets no clone at all")
	}
	if p.ArtifactMode != 0o111 || p.ArtifactName != "bin" {
		t.Errorf("cpp artifact %s %o, want bin 0111", p.ArtifactName, p.ArtifactMode)
	}
	if p.Compile.CPUms != 15000 || p.Compile.MemMB != 1024 || p.Compile.Pids != 64 || p.Exec.Pids != 4 || !p.Exec.StackFromMemory {
		t.Errorf("cpp limits: compile %d ms %d MiB pids %d; exec pids %d stack=mem %v", p.Compile.CPUms, p.Compile.MemMB, p.Compile.Pids, p.Exec.Pids, p.Exec.StackFromMemory)
	}
	for _, f := range []string{"-std=gnu++20", "-O2", "-static", "-pipe", "-fdiagnostics-format=json", "-fmax-errors=20", "-ftemplate-depth=512", "-fconstexpr-depth=512"} {
		if !slices.Contains(p.Compile.Argv, f) {
			t.Errorf("compile argv lacks %s", f)
		}
	}
	if p.Language != "cpp" || p.Baseline != "go@1.26" || p.TLMultiplier < 1 || p.Calibrated {
		t.Errorf("cpp: language %s baseline %s multiplier %v calibrated %v", p.Language, p.Baseline, p.TLMultiplier, p.Calibrated)
	}
}

func TestDiagnostics(t *testing.T) {
	out := []byte(`[{"kind": "error", "message": "expected ';' before '}' token", "children": [], "column-origin": 1, "locations": [{"caret": {"file": "solution.cpp", "line": 4, "display-column": 21, "byte-column": 21, "column": 21}}], "escape-source": false}, {"kind": "note", "message": "x", "locations": [{"caret": {"file": "solution.cpp", "line": 1, "column": 1}}]}, {"kind": "error", "message": "'class Solution' has no member named 'pairSum'", "locations": [{"caret": {"file": "zz_xl_harness.cpp", "line": 30, "column": 40}}]}, {"kind": "error", "message": "in prelude", "locations": [{"caret": {"file": "/usr/include/c++/14/bits/stl_vector.h", "line": 9, "column": 1}}]}]
/usr/bin/ld: /tmp/cc.o: in function ` + "`main'" + `:
solution.cpp:7: undefined reference to ` + "`helper()'" + `
collect2: error: ld returned 1 exit status
`)
	d := Diagnostics(out, map[string]bool{"solution.cpp": true, "zz_xl_harness.cpp": true, "xl_prelude.hpp": true})
	if len(d) != 3 {
		t.Fatalf("diags %+v", d)
	}
	if d[0].File != "solution.cpp" || d[0].Line != 4 || d[0].Col != 21 || d[1].File != "zz_xl_harness.cpp" || d[2].File != "solution.cpp" || d[2].Line != 7 {
		t.Errorf("diags %+v", d)
	}
}
