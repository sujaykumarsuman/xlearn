// Package python registers python@3.13 (D20; t3 §6.2; m3-04 task 3):
//
//   - compile: a syntax check (`compile()` of every source; SyntaxError → CE), the harness's
//     requirement check (solution.py defines the class and methods __main__.py calls, or a CE
//     in __main__.py, like Go's and C++'s "fails in the harness file"), then a deterministic
//     zipapp /w/app.pyz (__main__.py, solution.py, xl_prelude.py; stored, fixed timestamps);
//     15 s, 256 MiB, pids 16;
//   - exec: `python3 -s -P -S -B /job/app.pyz` under the amd64 `python` allowlist (t3 §16.2),
//     KILL-default, clone only with CLONE_THREAD; PYTHONHASHSEED=0; the harness raises
//     sys.setrecursionlimit and RLIMIT_STACK is the case's memory limit; pids 4.
//
// The interpreter is Debian trixie's CPython 3.13 at its system path; the jails see only it,
// its stdlib, the multiarch shared libraries and the dynamic loader (read-only binds).
package python

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	goprofile "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Name is the profile id.
const Name = "python@3.13"

// Interpreter is the python3 the jails run (Debian's /usr/bin/python3 → python3.13).
const Interpreter = "/usr/bin/python3"

const artifact = "/w/app.pyz"

// ExecArgs are the interpreter flags. -I would also imply -E, which ignores PYTHONHASHSEED;
// the jail's environment is entirely the profile's, so the run keeps -I's other two halves
// (-s, -P) and drops -E so the hash seed applies (decisions log, m3-04). -S: no site; -B: no
// .pyc writes.
var ExecArgs = []string{"-s", "-P", "-S", "-B"}

// ExecAllow and CompileAllow are this arch's lists (seccomp_<arch>.go).
func ExecAllow() []string    { return append([]string{}, execAllow...) }
func CompileAllow() []string { return append([]string{}, compileAllow...) }

// stdlib returns the resolved interpreter binary and its stdlib directory.
func stdlib() (bin, lib string) {
	bin = profile.Resolve(Interpreter)
	base := filepath.Base(bin) // python3.13
	if !strings.HasPrefix(base, "python3.") {
		base = "python3.13"
	}
	return bin, "/usr/lib/" + base
}

// Binds are the jails' read-only view of the interpreter: the binary at /usr/bin/python3, its
// stdlib, the multiarch shared libraries and the dynamic loader. Nothing else of /usr.
func Binds() []profile.Bind {
	bin, lib := stdlib()
	return profile.Existing(
		profile.Bind{Source: bin, Target: Interpreter},
		profile.Bind{Source: lib},
		profile.Bind{Source: "/usr/lib/" + profile.Triplet()},
		profile.Bind{Source: profile.Loader()},
	)
}

// Profile builds python@3.13.
func Profile() *profile.Profile {
	bin, lib := stdlib()
	return &profile.Profile{
		Name:         Name,
		Language:     "python",
		Baseline:     goprofile.Name,
		TLMultiplier: TLMultiplier,
		Toolchain:    "python3",
		Harnesses:    append([]string{}, harness.Harnesses...),
		ArtifactName: "app.pyz",
		ArtifactMode: 0o444,
		// Everything the jails read (the interpreter, its stdlib, the shared libraries its
		// extension modules load), so the page cache is charged to runner/ at startup.
		ToolchainPaths: profile.ExistingPaths(bin, lib, "/usr/lib/"+profile.Triplet()),
		Assets:         harness.Sources("python"),
		BaselineFiles: []runnerapi.File{
			{Path: harness.PythonMainFile, Data: []byte("pass\n")},
			{Path: harness.PythonLearnerFile, Data: []byte("")},
			{Path: harness.PythonPreludeFile, Data: []byte("")},
		},
		Compile: profile.Compile{
			Argv:       []string{Interpreter, "-I", "-S", "-B", "-c", CompileDriver, artifact},
			Env:        []string{"HOME=/w", "TMPDIR=/w", "TZ=UTC", "LANG=C.UTF-8"},
			Binds:      Binds(),
			OutputPath: artifact,
			CPUms:      15000,
			MemMB:      256,
			Pids:       16,
			WMB:        64,
			WInodes:    256,
			Seccomp: profile.Seccomp{
				Default: profile.DefaultENOSYS,
				// py_compile's set, plus the go build set and the init's extras for the runner's
				// compile init (a Go program).
				Allow:        seccomp.Union(compileAllow, goprofile.CompileAllow(), seccomp.CompileInitExtra),
				Kill:         seccomp.Dangerous,
				Clone3ENOSYS: true,
			},
		},
		Exec: profile.Exec{
			Argv: append(append([]string{Interpreter}, ExecArgs...), "/job/app.pyz"),
			Env: []string{"PYTHONHASHSEED=0", "PYTHONDONTWRITEBYTECODE=1", "PYTHONIOENCODING=utf-8",
				"TZ=UTC", "LANG=C.UTF-8"},
			Binds:           Binds(),
			Pids:            4,
			FSizeBytes:      1 << 20,
			NoFile:          64,
			StackFromMemory: true,
			WMB:             64,
			WInodes:         4096,
			Seccomp: profile.Seccomp{
				Default:         profile.DefaultKill,
				Allow:           seccomp.Without(execAllow, "clone"),
				Kill:            seccomp.Dangerous,
				Clone3ENOSYS:    true,
				CloneThreadOnly: true,
			},
		},
		Detect:      detect,
		Diagnostics: Diagnostics,
	}
}

// TLMultiplier is the provisional python multiplier over the Go TL (task 7: the runner-it
// lane's max CPU ratio vs the Go references on the perf cases, rounded up to the next 0.5,
// floor 1.0); mi-10 calibrates it on production.
const TLMultiplier = 1.0

func init() { profile.Register(Profile()) }

func detect() (string, error) {
	out, err := exec.Command(Interpreter, "-VV").Output()
	if err != nil {
		return "", fmt.Errorf("python3 -VV: %w", err)
	}
	v, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return v, nil
}

// Diagnostics reads the compile driver's JSON diagnostic lines ({"file","line","col","msg"},
// one per SyntaxError or missing requirement) and falls back to text.
func Diagnostics(out []byte, names map[string]bool) []runnerapi.Diag {
	var diags []runnerapi.Diag
	var rest []byte
	for _, line := range strings.Split(string(out), "\n") {
		var d struct {
			File string `json:"file"`
			Line int    `json:"line"`
			Col  int    `json:"col"`
			Msg  string `json:"msg"`
		}
		if strings.HasPrefix(line, `{"file"`) && json.Unmarshal([]byte(line), &d) == nil {
			if dd, ok := profile.NewDiag(d.File, d.Line, d.Col, d.Msg, names); ok && len(diags) < runnerapi.MaxDiags {
				diags = append(diags, dd)
			}
			continue
		}
		rest = append(append(rest, line...), '\n')
	}
	for _, d := range profile.TextDiags(rest, names) {
		if len(diags) < runnerapi.MaxDiags {
			diags = append(diags, d)
		}
	}
	return diags
}
