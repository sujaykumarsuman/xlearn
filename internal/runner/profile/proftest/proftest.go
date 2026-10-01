// Package proftest holds the invariant checks every language profile's tests run (m3-04 task
// 5). Test-only: only _test.go files import it.
package proftest

import (
	"runtime"
	"slices"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Lists are a profile's syscall lists for the running arch, and (amd64) t3 §16.2's published
// list with the recorded additions.
type Lists struct {
	Exec, Compile        []string
	Published, Additions []string
	// PublishedLen is the size t3 §16.2 gives the published list (27 go, 19 cpp, 39 python).
	PublishedLen int
}

// Invariants checks the lists and the profile:
//   - every list sorted and unique;
//   - the dangerous set (seccomp.Dangerous, incl. splice, vmsplice, tee) in no exec list and
//     in no compile allowlist;
//   - on amd64, the exec list is t3 §16.2's list verbatim plus the recorded additions;
//   - on Linux, every name of both filters resolves on this arch (seccomp.Assemble);
//   - exec filters are KILL-default, compile filters ENOSYS-default.
func Invariants(t *testing.T, p *profile.Profile, l Lists) {
	t.Helper()
	for name, list := range map[string][]string{"exec": l.Exec, "compile": l.Compile, "published": l.Published} {
		if !slices.IsSorted(list) {
			t.Errorf("%s %s list is not sorted: %v", p.Name, name, list)
		}
		if len(slices.Compact(slices.Clone(list))) != len(list) {
			t.Errorf("%s %s list has duplicates", p.Name, name)
		}
	}
	for _, d := range seccomp.Dangerous {
		if slices.Contains(l.Exec, d) || slices.Contains(p.Exec.Seccomp.Allow, d) {
			t.Errorf("%s: the dangerous %s is in the exec allowlist", p.Name, d)
		}
		if slices.Contains(p.Compile.Seccomp.Allow, d) {
			t.Errorf("%s: the dangerous %s is in the compile allowlist", p.Name, d)
		}
	}
	if !slices.Equal(p.Exec.Seccomp.Kill, seccomp.Dangerous) || !slices.Equal(p.Compile.Seccomp.Kill, seccomp.Dangerous) {
		t.Errorf("%s: both filters kill exactly the dangerous set", p.Name)
	}
	if runtime.GOARCH == "amd64" {
		if len(l.Published) != l.PublishedLen {
			t.Errorf("%s: t3 §16.2 publishes %d names, the copy has %d", p.Name, l.PublishedLen, len(l.Published))
		}
		if want := seccomp.Union(l.Published, l.Additions); !slices.Equal(l.Exec, want) {
			t.Errorf("%s: exec list = t3 §16.2 + additions?\n got %v\nwant %v", p.Name, l.Exec, want)
		}
	}
	if p.Exec.Seccomp.Default != profile.DefaultKill {
		t.Errorf("%s: exec filters are KILL-default", p.Name)
	}
	if p.Compile.Seccomp.Default != profile.DefaultENOSYS {
		t.Errorf("%s: compile filters are ENOSYS-default", p.Name)
	}
	if runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		for name, pol := range map[string]seccomp.Policy{"exec": p.Exec.Seccomp.Policy(), "compile": p.Compile.Seccomp.Policy()} {
			_, unresolved, err := seccomp.Assemble(pol, seccomp.Native())
			if err != nil || len(unresolved) > 0 {
				t.Errorf("%s %s filter on %s: unresolved %v (%v)", p.Name, name, runtime.GOARCH, unresolved, err)
			}
		}
	}
}
