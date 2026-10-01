package packspec

import "context"

// The executor seam (t1 §7.2–7.3): EVERY author- or AI-written program — the reference,
// the brute oracle, wrong solutions, generators, custom validators — runs only through an
// Executor, never on the host. The production executor (internal/packspec/executor) runs
// one `docker run --network none --read-only …` container per program with digest-pinned
// images: our public driver compiles the program once and spawns one process per case,
// measuring CPU time and peak RSS per case. Tests use a fake.

// Mode is how a program receives a case.
type Mode string

// Modes.
const (
	// ModeHarness: the case's fd-3 frame in, the fd-4 frame out (reference, brute, wrong).
	ModeHarness Mode = "harness"
	// ModeStdio: argv and stdin in, stdout out, exit 0 = success (generators, validators).
	ModeStdio Mode = "stdio"
)

// SourceFile is one program file (a flat name).
type SourceFile struct {
	Name string
	Data []byte
}

// Program is one program to build once and run per case.
type Program struct {
	// Label names it in messages ("fx-002 wrong submissions/wrong/cubic.go"): ids and paths only.
	Label string
	// Lang is the program's language; only "go" executes before m3-04.
	Lang  string
	Files []SourceFile
	Mode  Mode
}

// Run is one case's process.
type Run struct {
	// Input is the fd-3 frame (harness) or stdin (stdio).
	Input []byte
	// Args follow the program on its command line (stdio).
	Args []string
}

// Limits apply to every run of one Execute.
type Limits struct {
	// CPUms is the per-case CPU limit; the wall kill is 1.5·CPUms + 500 ms (t3 §7.2).
	CPUms int64
	// StopAfterKill skips every run after the first killed one (a TLE ends a perf block).
	StopAfterKill bool
	// OutputCap bounds fd 4 / stdout per run (default 64 MiB).
	OutputCap int64
}

// RunResult is one case's outcome, measured outside the program.
type RunResult struct {
	Exit   int    `json:"exit"`
	Signal string `json:"signal,omitempty"`
	CPUms  int64  `json:"cpu_ms"`
	WallMs int64  `json:"wall_ms"`
	// PeakKB is the process's peak RSS (rusage maxrss).
	PeakKB int64 `json:"peak_kb"`
	// Killed means the wall/CPU limit stopped it.
	Killed bool `json:"killed,omitempty"`
	// Skipped means it never ran (StopAfterKill).
	Skipped bool `json:"skipped,omitempty"`
	// Output is fd 4 (harness) or stdout (stdio).
	Output       []byte `json:"output,omitempty"`
	OutputCapped bool   `json:"output_capped,omitempty"`
}

// ExecResult is one program's build and runs.
type ExecResult struct {
	CompileOK bool `json:"compile_ok"`
	// CompileDiag is the compiler's output; packlint prints it only with --verbose (it may
	// quote pack source).
	CompileDiag string `json:"compile_diag,omitempty"`
	// BaselineKB is a no-op program's peak RSS in the same container (the memory rule's
	// baseline).
	BaselineKB int64       `json:"baseline_kb"`
	Runs       []RunResult `json:"runs"`
}

// Executor runs programs in isolation.
type Executor interface {
	// Execute builds p once and runs every Run as its own process, in order.
	Execute(ctx context.Context, p Program, runs []Run, lim Limits) (*ExecResult, error)
	// SyntaxCheck compiles (C++) or byte-compiles (Python) files without running them: the
	// C++/Python references before m3-04's harnesses exist.
	SyntaxCheck(ctx context.Context, lang string, files []SourceFile) (ok bool, diag string, err error)
	// Images are the pinned executor images by language (go, cpp, python), for the lock header.
	Images() map[string]string
}
