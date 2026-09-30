// Package front is the runner's capability-less half (t3 §5.2): a fixed non-zero UID with an
// empty bounding set, 0 capabilities and NO_NEW_PRIVS, set before its first instruction. It
// serves HTTP on :8090 (judge is the only caller), checks the bearer token, spools inputs in
// memory, writes the job's source files, streams inputs into fd 3, decodes fds 1/2/4 and the
// compile diagnostics, and assembles the Result. It never holds a capability and never touches
// cgroupfs: everything privileged is a fixed-schema request to the spawner (Backend).
package front

import (
	"context"
	"fmt"
	"os"

	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
)

// Backend is the spawner as the front sees it. The Linux implementation speaks the ipc
// protocol; tests use a fake.
type Backend interface {
	Ping(ctx context.Context) error
	// BeginJob sets a job up in the slot; the caller writes the sources through SrcDir and
	// closes it.
	BeginJob(ctx context.Context, slot, profileIdx int, seq uint64, compileCPUms, compileMemMB int64) (Begun, error)
	// Compile runs the compile jail; the caller drains Stdout/Stderr and closes them.
	Compile(ctx context.Context, slot int) (*CompileRun, error)
	// RunCase runs one case; the caller writes Input, drains the three outputs and closes all.
	RunCase(ctx context.Context, slot int, req ipc.CaseRun) (*CaseRun, error)
	// EndJob tears the job down (abort kills anything still running).
	EndJob(ctx context.Context, slot int, abort bool) (ipc.Teardown, error)
	QuietCheck(ctx context.Context, slot int) (QuietStatus, error)
	Stats(ctx context.Context) (ipc.StatsReply, error)
	Probe(ctx context.Context) (ipc.ProbeReply, error)
	// DrainRequested is closed when the spawner asks the front to drain (SIGTERM).
	DrainRequested() <-chan struct{}
	// Broken is closed when the pair is gone; the front exits at once.
	Broken() <-chan struct{}
}

// Begun is a started job.
type Begun struct {
	SrcDir         *os.File
	CanaryMedianUs int64
}

// CompileRun is a running compile.
type CompileRun struct {
	Stdout, Stderr *os.File
	Done           <-chan CompileOutcome
}

// CompileOutcome is the compile's evidence, or the error that ended the request.
type CompileOutcome struct {
	Done *ipc.CompileDone
	Err  error
}

// CaseRun is a running case.
type CaseRun struct {
	Input                  *os.File // fd 3's write end
	Stdout, Stderr, Result *os.File // fds 1, 2 and 4's read ends
	Done                   <-chan CaseOutcome
	// Kill asks the spawner to kill the case (the front saw OLE).
	Kill func()
}

// CaseOutcome is a case's evidence, or the error that ended the request.
type CaseOutcome struct {
	Done *ipc.CaseDone
	Err  error
}

// QuietStatus is the quiet re-run's entry measurement.
type QuietStatus struct {
	Steal  float64
	Canary float64
}

// FailError is a TFail reply.
type FailError struct {
	Code  ipc.FailCode
	Errno uint32
}

func (e *FailError) Error() string {
	return fmt.Sprintf("spawner refused the request (code %d, errno %d)", e.Code, e.Errno)
}
