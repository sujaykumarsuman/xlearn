// Package profile is the runner's language-profile type and registry. m3-03 defines the shape
// and a test-only profile (testgo@0, build tag runner_it); m3-04 fills in go@1.26, cpp and
// python. Profiles register from init() in their own packages, which cmd/runner imports; a
// release build imports no test profile (guarded by internal/runner/imports_test.go).
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// Bind is a read-only bind mount into a jail. Target is an absolute path inside the jail; an
// empty Target means "the same path as Source" (toolchains keep their paths so compile caches
// and GOROOT line up). Every bind is MS_BIND|MS_REC|MS_NOSUID|MS_NODEV|MS_RDONLY without
// MS_PRIVATE (the AppArmor remount rule, t3 §16.2 block 3).
type Bind struct {
	Source string
	Target string
}

// SeccompDefault names a filter's default action.
type SeccompDefault string

// Defaults. Kill is the exec default on every arch (m3-04 retired m3-03's dev-only arm64 LOG
// switch); ENOSYS is the compile default; Allow exists only for runner_it test profiles that
// exercise the namespace layer on its own.
const (
	DefaultKill   SeccompDefault = "kill"
	DefaultENOSYS SeccompDefault = "enosys"
	DefaultAllow  SeccompDefault = "allow"
)

// Seccomp describes one filter by names (internal/runner/seccomp assembles it).
type Seccomp struct {
	Default         SeccompDefault
	Allow           []string
	Kill            []string
	Clone3ENOSYS    bool
	CloneThreadOnly bool
	PrctlSetVMAOnly bool
}

// Policy turns the description into an assembler policy.
func (s Seccomp) Policy() seccomp.Policy {
	var def seccomp.Action
	switch s.Default {
	case DefaultENOSYS:
		def = seccomp.CompileDefault()
	case DefaultAllow:
		def = seccomp.ActAllow
	default:
		def = seccomp.ExecDefault()
	}
	return seccomp.Policy{
		Default: def, Allow: s.Allow, Kill: s.Kill, Clone3ENOSYS: s.Clone3ENOSYS,
		CloneThreadOnly: s.CloneThreadOnly, PrctlSetVMAOnly: s.PrctlSetVMAOnly,
	}
}

// Compile is the compile jail's spec. Argv runs with /src (the job's read-only files) as its
// working directory, under the runner's in-jail compile init, and must leave the artifact at
// OutputPath (on the /w tmpfs). HiddenFiles are written next to Files.
type Compile struct {
	Argv       []string
	Env        []string
	Binds      []Bind
	OutputPath string
	// CPUms and MemMB are the defaults when the job leaves them 0; CPU is capped at 15 s (L14).
	CPUms int64
	MemMB int64
	Pids  int64
	// WMB and WInodes size the compile's /w tmpfs.
	WMB     int64
	WInodes int64
	Seccomp Seccomp
}

// Exec is the per-case jail's spec. Argv runs with the artifact directory bound read-only at
// /job (the artifact is /job/<ArtifactName>).
type Exec struct {
	Argv  []string
	Env   []string
	Binds []Bind
	Pids  int64
	// Rlimits (0 = unset). RLIMIT_CORE is always 0 and RLIMIT_CPU is always TL + 2 s.
	FSizeBytes uint64
	NoFile     uint64
	StackBytes uint64
	// StackFromMemory sets RLIMIT_STACK to the case's memory limit (mem_mb + the baseline), so
	// deep recursion is bounded by memory, not by an 8 MiB stack (C++, Python).
	StackFromMemory bool
	// WMB and WInodes size the per-case /w tmpfs (default 64 MiB, 4k inodes).
	WMB     int64
	WInodes int64
	// Proc mounts a fresh procfs with hidepid=invisible,subset=pid (go-race later). Default off.
	Proc    bool
	Seccomp Seccomp
}

// Profile is one language profile.
type Profile struct {
	// Name is "<name>@<version>", the Job.Profile value.
	Name string
	// Toolchain is the version string reported in Versions.Toolchain.
	Toolchain string
	// Harnesses are the harness majors the profile accepts ("func-json@1").
	Harnesses []string
	// ArtifactName is the artifact's file name in the artifact tmpfs (and under /job).
	ArtifactName string
	// ArtifactMode is the artifact's file mode: 0111 for a static ELF (exec needs no read
	// bit), 0444 for an interpreted artifact the interpreter must read.
	ArtifactMode fs.FileMode
	// ToolchainPaths are pre-read at startup (so their page cache is charged to runner/, not to
	// a case cgroup) and hashed into ProfileSHA.
	ToolchainPaths []string
	// BaselineFiles is a no-op program: startup compiles it and runs it to measure the
	// profile's memory baseline (median memory.peak).
	BaselineFiles []runnerapi.File
	Compile       Compile
	Exec          Exec
	// TestOnly marks runner_it profiles; a release build must not register one.
	TestOnly bool

	// Language is the item `languages[]` value the profile serves ("go", "cpp", "python"; ""
	// for a test profile). ForLanguage is the one table; GET /v1/profiles serves it.
	Language string
	// Baseline is the TL baseline profile: judge's pack TLs are measured on it, and judge
	// multiplies them by TLMultiplier for this profile (ADR-0030 §3).
	Baseline string
	// TLMultiplier and Calibrated are served by GET /v1/profiles: provisional (false) from the
	// runner-it lane until mi-10 calibrates on production. The runner enforces only the CPU TL
	// judge sends.
	TLMultiplier float64
	Calibrated   bool
	// Assets are the public harness templates and preludes the profile's jobs carry (path →
	// bytes); they are part of ProfileSHA, so a one-byte template change moves it.
	Assets map[string][]byte
	// Detect reports the toolchain version at spawner startup (go version, g++
	// -dumpfullversion, python3 -VV); "" keeps Toolchain.
	Detect func() (string, error) `json:"-"`
	// Diagnostics parses compile output into positioned diagnostics for the job's own file
	// names; nil uses the front's generic file:line:col parser.
	Diagnostics func(out []byte, names map[string]bool) []runnerapi.Diag `json:"-"`
}

// ForLanguage returns the registered profile serving an item language ("go", "cpp", "python").
func ForLanguage(lang string) (*Profile, bool) {
	for _, p := range All() {
		if !p.TestOnly && p.Language == lang && lang != "" {
			return p, true
		}
	}
	return nil, false
}

// Fingerprint is ProfileSHA: a SHA-256 over the profile's description (commands, env, binds,
// limits, the seccomp lists, the harness templates and preludes, the TL multiplier), the
// detected toolchain version, and the tree hash of every toolchain path (the toolchain itself
// and, for Go, the GOCACHE seed). tree returns a path's tree hash.
func Fingerprint(p *Profile, toolchain string, tree func(path string) (string, error)) (string, error) {
	h := sha256.New()
	desc, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	h.Write(desc)
	fmt.Fprintf(h, "\ntoolchain %s", toolchain)
	for _, path := range p.ToolchainPaths {
		th, err := tree(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "\n%s %s", path, th)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Accepts reports whether the profile takes harness h.
func (p *Profile) Accepts(h string) bool {
	for _, x := range p.Harnesses {
		if x == h {
			return true
		}
	}
	return false
}

var (
	mu       sync.RWMutex
	registry = map[string]*Profile{}
)

// Register adds p. It panics on a duplicate or an invalid profile (a programming error caught
// at init).
func Register(p *Profile) {
	if err := p.validate(); err != nil {
		panic(fmt.Sprintf("profile %q: %v", p.Name, err))
	}
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[p.Name]; dup {
		panic(fmt.Sprintf("profile %q registered twice", p.Name))
	}
	registry[p.Name] = p
}

func (p *Profile) validate() error {
	if p.Name == "" || p.ArtifactName == "" || len(p.Exec.Argv) == 0 || len(p.Compile.Argv) == 0 {
		return fmt.Errorf("incomplete")
	}
	if strings.ContainsAny(p.ArtifactName, "/\x00") || p.ArtifactName == "." || p.ArtifactName == ".." {
		return fmt.Errorf("artifact name %q", p.ArtifactName)
	}
	switch p.ArtifactMode {
	case 0o111, 0o444, 0o555:
	default:
		return fmt.Errorf("artifact mode %o", p.ArtifactMode)
	}
	if p.Compile.OutputPath == "" || p.Compile.Pids <= 0 || p.Exec.Pids <= 0 {
		return fmt.Errorf("compile output and pids caps are required")
	}
	if p.Exec.Seccomp.Default == DefaultAllow && !p.TestOnly {
		return fmt.Errorf("an allow-default exec filter is test-only")
	}
	if len(p.BaselineFiles) == 0 {
		return fmt.Errorf("a baseline program is required")
	}
	return nil
}

// All returns the registered profiles sorted by name; an index into it is the profile's IPC id
// (the spawner and front are the same binary, so the order agrees).
func All() []*Profile {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]*Profile, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup finds a profile by name and returns its index in All().
func Lookup(name string) (*Profile, int, bool) {
	for i, p := range All() {
		if p.Name == name {
			return p, i, true
		}
	}
	return nil, 0, false
}

// ByIndex returns All()[i].
func ByIndex(i int) (*Profile, bool) {
	all := All()
	if i < 0 || i >= len(all) {
		return nil, false
	}
	return all[i], true
}
