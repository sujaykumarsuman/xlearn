// Package cpp registers cpp@g++14 (D20; t3 §6.2/§6.4; m3-04 task 3):
//
//   - compile: `g++ -std=gnu++20 -O2 -static -pipe -fdiagnostics-format=json -fmax-errors=20
//     -ftemplate-depth=512 -fconstexpr-depth=512 -o /w/bin zz_xl_harness.cpp` (the harness
//     includes xl_prelude.hpp and the learner's solution.cpp, so diagnostics keep
//     solution.cpp:line positions), with Debian trixie's g++ 14 and /usr bound read-only;
//     15 s, 1 GiB, pids 64 (cc1plus, as, ld): the compile-bomb limits, artifact ≤ 64 MiB;
//   - exec: the static /job/bin under the amd64 `cpp` allowlist (t3 §16.2), KILL-default, no
//     clone at all; RLIMIT_STACK = the case's memory limit (deep recursion); pids 4.
//
// m3-15 decides whether the image ships a precompiled xl_prelude.hpp.
package cpp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	goprofile "github.com/sujaykumarsuman/xlearn/internal/runner/profile/go"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Name is the profile id.
const Name = "cpp@g++14"

// Compiler is Debian trixie's g++ (14) in the image.
const Compiler = "/usr/bin/g++"

const artifact = "/w/bin"

// Flags are the compile flags (t3 §6.2 + D20).
var Flags = []string{
	"-std=gnu++20", "-O2", "-static", "-pipe", "-fdiagnostics-format=json", "-fmax-errors=20",
	"-ftemplate-depth=512", "-fconstexpr-depth=512",
}

// ExecAllow and CompileAllow are this arch's lists (seccomp_<arch>.go).
func ExecAllow() []string    { return append([]string{}, execAllow...) }
func CompileAllow() []string { return append([]string{}, compileAllow...) }

// Profile builds cpp@g++14.
func Profile() *profile.Profile {
	tri := profile.Triplet()
	argv := append(append([]string{Compiler}, Flags...), "-o", artifact, harness.CppHarnessFile)
	return &profile.Profile{
		Name:         Name,
		Language:     "cpp",
		Baseline:     goprofile.Name,
		TLMultiplier: TLMultiplier,
		Toolchain:    "g++",
		Harnesses:    append([]string{}, harness.Harnesses...),
		ArtifactName: "bin",
		ArtifactMode: 0o111,
		// Everything a compile reads, so its page cache is charged to runner/ at startup (never
		// to a slot, whose teardown checks memory is back to baseline) and ProfileSHA covers it:
		// the driver, cc1plus and its libraries, the headers, binutils, the static libc and
		// libstdc++ (not the rest of /usr).
		ToolchainPaths: profile.ExistingPaths(profile.Resolve(Compiler), "/usr/include", "/usr/lib/gcc/"+tri+"/14",
			"/usr/libexec/gcc/"+tri+"/14", "/usr/lib/"+tri, profile.Resolve("/usr/bin/"+tri+"-as"),
			profile.Resolve("/usr/bin/"+tri+"-ld.bfd")),
		Assets:        harness.Sources("cpp"),
		BaselineFiles: []runnerapi.File{{Path: harness.CppHarnessFile, Data: []byte("int main() { return 0; }\n")}},
		Compile: profile.Compile{
			Argv:       argv,
			Env:        []string{"PATH=/usr/bin:/bin", "HOME=/w", "TMPDIR=/w", "TZ=UTC", "LANG=C.UTF-8"},
			Binds:      profile.SystemBinds(),
			OutputPath: artifact,
			CPUms:      15000,
			MemMB:      1024,
			Pids:       64,
			WMB:        128,
			WInodes:    1024,
			Seccomp: profile.Seccomp{
				Default: profile.DefaultENOSYS,
				// g++'s own set, plus the go build set and the init's extras for the runner's
				// compile init (a Go program).
				Allow:        seccomp.Union(compileAllow, goprofile.CompileAllow(), seccomp.CompileInitExtra),
				Kill:         seccomp.Dangerous,
				Clone3ENOSYS: true,
			},
		},
		Exec: profile.Exec{
			Argv:            []string{"/job/bin"},
			Env:             []string{"TZ=UTC", "LANG=C.UTF-8"},
			Pids:            4,
			FSizeBytes:      1 << 20,
			NoFile:          64,
			StackFromMemory: true,
			WMB:             64,
			WInodes:         4096,
			Seccomp: profile.Seccomp{
				Default:      profile.DefaultKill,
				Allow:        execAllow,
				Kill:         seccomp.Dangerous,
				Clone3ENOSYS: true,
			},
		},
		Detect:      detect,
		Diagnostics: Diagnostics,
	}
}

// TLMultiplier is the provisional cpp multiplier over the Go TL (task 7: the runner-it lane's
// max CPU ratio vs the Go references on the perf cases, rounded up to the next 0.5, floor
// 1.0); mi-10 calibrates it on production.
const TLMultiplier = 1.0

func init() { profile.Register(Profile()) }

func detect() (string, error) {
	out, err := exec.Command(Compiler, "-dumpfullversion").Output()
	if err != nil {
		return "", fmt.Errorf("g++ -dumpfullversion: %w", err)
	}
	return "g++ " + strings.TrimSpace(string(out)), nil
}

// gccDiag is one entry of GCC's -fdiagnostics-format=json array.
type gccDiag struct {
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Locations []struct {
		Caret struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Column int    `json:"column"`
		} `json:"caret"`
	} `json:"locations"`
}

// Diagnostics parses GCC's JSON diagnostics (stderr; notes are dropped) and falls back to
// "file:line: msg" text for everything else (the linker's errors).
func Diagnostics(out []byte, names map[string]bool) []runnerapi.Diag {
	var diags []runnerapi.Diag
	var rest bytes.Buffer
	for len(out) > 0 {
		line, tail, _ := bytes.Cut(out, []byte("\n"))
		if len(line) > 0 && line[0] == '[' {
			dec := json.NewDecoder(bytes.NewReader(out))
			var arr []gccDiag
			if err := dec.Decode(&arr); err == nil {
				for _, g := range arr {
					if g.Kind == "note" || len(g.Locations) == 0 {
						continue
					}
					c := g.Locations[0].Caret
					if d, ok := profile.NewDiag(c.File, c.Line, c.Column, g.Kind+": "+g.Message, names); ok && len(diags) < runnerapi.MaxDiags {
						diags = append(diags, d)
					}
				}
				out = out[dec.InputOffset():]
				continue
			}
		}
		rest.Write(line)
		rest.WriteByte('\n')
		out = tail
	}
	for _, d := range profile.TextDiags(rest.Bytes(), names) {
		if len(diags) < runnerapi.MaxDiags {
			diags = append(diags, d)
		}
	}
	return diags
}
