package ipc

import (
	"fmt"

	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
)

// Bodies. Every field is a fixed-size number: no strings, no slices, no learner bytes.

// Ping checks the pair answers (GET /healthz).
type Ping struct{ Nonce uint64 }

// Pong echoes Ping.Nonce.
type Pong struct{ Nonce uint64 }

// JobBegin asks the spawner to set a job up in Msg.Slot: its cgroup subtree and a fresh src
// tmpfs owned by the front's UID. ProfileIdx indexes profile.All() (the same binary on both
// sides). JobSeq numbers the job within this boot (it picks the job's UIDs).
type JobBegin struct {
	ProfileIdx   uint16
	JobSeq       uint64
	CompileCPUms uint32
	CompileMemMB uint32
}

func (b *JobBegin) validate() error {
	if b.CompileCPUms == 0 || b.CompileCPUms > uint32(measure.CompileCap.Milliseconds()) {
		return fmt.Errorf("compile cpu %d ms out of range", b.CompileCPUms)
	}
	if b.CompileMemMB == 0 || b.CompileMemMB > 1024 {
		return fmt.Errorf("compile mem %d MiB out of range", b.CompileMemMB)
	}
	return nil
}

// JobBegun carries the src directory fd (the front writes the job's files through it) and the
// canary's current rolling median (Versions.CanaryMedian).
type JobBegun struct{ CanaryMedianUs uint64 }

// Compile runs the compile jail on the (now read-only) src directory.
type Compile struct{}

// CompileStarted carries the compiler's stdout and stderr read ends.
type CompileStarted struct{}

// Export is the in-jail exporter's outcome.
type Export uint8

// Export outcomes.
const (
	ExportOK      Export = 0
	ExportMissing Export = 1 // the compile succeeded but produced no regular file
	ExportTooBig  Export = 2 // over the 64 MiB artifact cap
	ExportFailed  Export = 3 // the pipe or the artifact tmpfs failed (setup)
	ExportSkipped Export = 4 // the compile failed; nothing to export
)

// CompileDone is the compile's evidence.
type CompileDone struct {
	Exited        uint8
	ExitCode      int32
	Signal        uint16
	Kill          uint8
	CPUus         uint64
	WallUs        uint64
	PeakBytes     uint64
	OOMKills      uint32
	PidsMax       uint32
	ArtifactBytes uint64
	Export        uint8
}

func (b *CompileDone) validate() error {
	if err := boolByte("exited", b.Exited); err != nil {
		return err
	}
	if !measure.Kill(b.Kill).Valid() {
		return fmt.Errorf("kill reason %d", b.Kill)
	}
	if b.Export > uint8(ExportSkipped) {
		return fmt.Errorf("export %d", b.Export)
	}
	return nil
}

// CaseRun runs case CaseIdx. WallCapMs is what is left of the tests cap (L14) for this case;
// the spawner kills at the earliest of the CPU, wall, idle and cap limits.
type CaseRun struct {
	CaseIdx   uint32
	CPUms     uint32
	MemMB     uint32
	WallCapMs uint32
	Quiet     uint8
}

func (b *CaseRun) validate() error {
	if b.CPUms < 100 || b.CPUms > 10_000 {
		return fmt.Errorf("case cpu %d ms out of range", b.CPUms)
	}
	if b.MemMB < 16 || b.MemMB > 1024 {
		return fmt.Errorf("case mem %d MiB out of range", b.MemMB)
	}
	if b.WallCapMs == 0 || b.WallCapMs > uint32(measure.TestsCap.Milliseconds()) {
		return fmt.Errorf("wall cap %d ms out of range", b.WallCapMs)
	}
	if b.CaseIdx >= 512 {
		return fmt.Errorf("case index %d", b.CaseIdx)
	}
	return boolByte("quiet", b.Quiet)
}

// CaseStarted carries fd 3's write end and the read ends of fds 1, 2 and 4, in that order.
type CaseStarted struct{}

// CaseDone is one case's evidence, all read outside the learner's process.
type CaseDone struct {
	Exited         uint8
	ExitCode       int32
	Signal         uint16
	Kill           uint8
	CPUus          uint64
	WallUs         uint64
	RusageCPUus    uint64
	PeakBytes      uint64
	BaselineBytes  uint64
	OOMKills       uint32
	PidsMax        uint32
	StealPermille  uint16
	CanaryPermille uint32 // on-demand canary ÷ median × 1000; 0 = not run
	OtherBusy      uint8
}

func (b *CaseDone) validate() error {
	if err := boolByte("exited", b.Exited); err != nil {
		return err
	}
	if err := boolByte("other_busy", b.OtherBusy); err != nil {
		return err
	}
	if !measure.Kill(b.Kill).Valid() {
		return fmt.Errorf("kill reason %d", b.Kill)
	}
	if b.StealPermille > 1000 {
		return fmt.Errorf("steal %d‰", b.StealPermille)
	}
	return nil
}

// CaseKill asks the spawner to kill the running case (the front saw OLE).
type CaseKill struct{ Reason uint8 }

func (b *CaseKill) validate() error {
	if measure.Kill(b.Reason) != measure.KillOLE {
		return fmt.Errorf("case kill reason %d", b.Reason)
	}
	return nil
}

// JobEnd tears the job down. Abort kills whatever still runs (client gone, drain).
type JobEnd struct{ Abort uint8 }

func (b *JobEnd) validate() error { return boolByte("abort", b.Abort) }

// Teardown is JobEnded's outcome.
type Teardown uint16

// Teardown outcomes. Anything but TeardownOK fails the slot closed: the job answers
// infra: setup and the slot takes no more jobs this boot.
const (
	TeardownOK        Teardown = 0
	TeardownSurvivors Teardown = 1 // pids.current never reached 0
	TeardownMemory    Teardown = 2 // memory.current not back to baseline ±5 MiB
	TeardownCgroup    Teardown = 3 // a cgroup couldn't be killed or removed
	TeardownMount     Teardown = 4 // a per-job tmpfs couldn't be unmounted
)

// JobEnded reports the teardown assertions.
type JobEnded struct{ Outcome uint16 }

func (b *JobEnded) validate() error {
	if b.Outcome > uint16(TeardownMount) {
		return fmt.Errorf("teardown outcome %d", b.Outcome)
	}
	return nil
}

// QuietCheck asks for the quiet re-run's entry measurements: steal over 10 s and a
// pre-canary (t3 §5.7).
type QuietCheck struct{}

// QuietStatus is the quiet check's result.
type QuietStatus struct {
	StealPermille  uint16
	CanaryPermille uint32
}

// Stats asks for the cgroup counters behind GET /v1/stats.
type Stats struct{}

// StatsReply carries them.
type StatsReply struct {
	MemoryCurrent  [Slots]uint64
	PidsCurrent    [Slots]uint32
	NrDying        [Slots]uint32
	RunnerMemory   uint64
	OOMKills       uint64
	OOMKillsLocal  uint64
	CanaryMedianUs uint64
}

// Probe asks the spawner to run the privileged readiness canaries.
type Probe struct{}

// Probe bits (ProbeReply.Ran / Failed).
const (
	ProbeAppArmor    = 1 << 0 // the AppArmor label is xlearn-runner
	ProbeUIDMap      = 1 << 1 // the UID map is not the identity map
	ProbeCgroupfs    = 1 << 2 // cgroupfs is writable (functional, never downgraded)
	ProbeUserNS      = 1 << 3 // clone(CLONE_NEWUSER) is denied
	ProbeFsopen      = 1 << 4 // fsopen("tmpfs") is denied
	ProbeSCTP        = 1 << 5 // socket(AF_INET, SOCK_STREAM, IPPROTO_SCTP) is denied
	ProbeCorePattern = 1 << 6 // core_pattern doesn't start with '|'
	ProbeAll         = ProbeAppArmor | ProbeUIDMap | ProbeCgroupfs | ProbeUserNS | ProbeFsopen | ProbeSCTP | ProbeCorePattern
)

// ProbeReply lists which probes ran and which failed.
type ProbeReply struct {
	Ran    uint32
	Failed uint32
}

func (b *ProbeReply) validate() error {
	if b.Ran&^ProbeAll != 0 || b.Failed&^b.Ran != 0 {
		return fmt.Errorf("probe bits ran=%#x failed=%#x", b.Ran, b.Failed)
	}
	return nil
}

// Drain tells the front to drain and exit (SIGTERM reached the spawner).
type Drain struct{}

// FailCode is why a request failed.
type FailCode uint16

// Fail codes.
const (
	FailSetup      FailCode = 1 // a spawner syscall failed before learner code ran
	FailBadRequest FailCode = 2 // the request doesn't fit the slot's state
	FailSlotClosed FailCode = 3 // the slot failed closed earlier this boot
)

// Fail reports a failed request. Errno is the underlying errno, if any (for the ERROR log).
type Fail struct {
	Code  uint16
	Errno uint32
}

func (b *Fail) validate() error {
	if b.Code < uint16(FailSetup) || b.Code > uint16(FailSlotClosed) {
		return fmt.Errorf("fail code %d", b.Code)
	}
	return nil
}
