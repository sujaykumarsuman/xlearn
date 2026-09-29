// Package runnerapi is the judge↔runner contract (t4 §11.1 with the T3 amendments; t3 §5.3,
// §5.5, §5.9, §5.11): the Job and Result types, the POST /v1/jobs wire stream, the
// validators and the generic file-name rule. It is imported by the runner
// (internal/runner/...) and by judge (m3-06), so it stays a stdlib-only leaf package.
//
// Contract discipline (ADR-0034 §1.5): every type here is append-only from runner-v1.0.0.
// Removing, renaming or retyping a field, or narrowing an enum, is a runner major.
package runnerapi

// Wire constants for POST /v1/jobs.
const (
	// MediaTypeJob is the request body's media type; its "v" parameter is the stream version.
	MediaTypeJob = "application/vnd.xlearn.runner-job"
	// StreamVersion is the only stream version this package speaks.
	StreamVersion = "1"
	// ContentTypeJob is the full Content-Type header judge sends.
	ContentTypeJob = MediaTypeJob + "; v=" + StreamVersion
)

// Mode is the job mode: a learner Run (public samples + custom inputs, stdout/stderr kept) or a
// Submit (verdict-only; stdout/stderr dropped).
type Mode string

// Modes.
const (
	ModeRun    Mode = "run"
	ModeSubmit Mode = "submit"
)

// Group is a case's group. Cases arrive in group order: sample, edge, random, perf (t4 §2.6).
type Group string

// Groups, in their canonical order.
const (
	GroupSample Group = "sample"
	GroupEdge   Group = "edge"
	GroupRandom Group = "random"
	GroupPerf   Group = "perf"
)

// groupRank is the canonical order ValidateJob enforces.
var groupRank = map[Group]int{GroupSample: 0, GroupEdge: 1, GroupRandom: 2, GroupPerf: 3}

// StopRule is a StopGroupOn value.
//   - StopAnyFail: the first non-ok case in the group ends the test phase — every later case,
//     in this group and every later group, is not_run (t4 §2.6: a sample failure skips the
//     hidden cases).
//   - StopTLE: the first tle in the group marks the rest of that group not_run; later groups
//     still run (the perf block stops at its first TLE).
type StopRule string

// Stop rules.
const (
	StopAnyFail StopRule = "any_fail"
	StopTLE     StopRule = "tle"
)

// OutputMode says how the harness channel (fd 4) comes back: the bytes themselves, or only
// their SHA-256 and length (large perf outputs).
type OutputMode string

// Output modes.
const (
	OutputBytes  OutputMode = "bytes"
	OutputSHA256 OutputMode = "sha256"
)

// Term is a case's terminal state, measured outside the learner's process. Every value is
// learner-attributable; platform faults are InfraError, never a Term.
type Term string

// Terms (closed set).
const (
	TermOK          Term = "ok"
	TermTLE         Term = "tle"
	TermMLE         Term = "mle"
	TermOLE         Term = "ole"
	TermSignal      Term = "signal"
	TermExitNonzero Term = "exit_nonzero"
	TermNotRun      Term = "not_run"
)

// CompileLimit names the compile limit a failed compile hit (CompileResult.Limit).
type CompileLimit string

// Compile limits. A limit hit is a CE (L14), and its CPU counts in Telemetry.SlotMs.
const (
	CompileLimitCPU    CompileLimit = "cpu"
	CompileLimitWall   CompileLimit = "wall"
	CompileLimitMemory CompileLimit = "memory"
	CompileLimitPids   CompileLimit = "pids"
	CompileLimitOutput CompileLimit = "output"
)

// Job is the header frame (job.json) of POST /v1/jobs. Case inputs follow as separate frames
// (stream.go), so they never sit inside the JSON.
//
// There is no Job.Cost (dropped, t3 §14) and no wall, idle or job deadline: the runner
// derives those from Limits (t3 §7.2, L14).
type Job struct {
	// ID is judge's job id (opaque; [A-Za-z0-9_-], ≤ 64 bytes).
	ID string `json:"id"`
	// Profile is the runner profile, "<name>@<version>" (GET /v1/profiles lists them).
	Profile string `json:"profile"`
	// Harness is the harness codec, "<name>@<major>" (e.g. func-json@1); the profile lists
	// the majors it accepts.
	Harness string `json:"harness"`
	// Mode is run or submit.
	Mode Mode `json:"mode"`
	// Files are the learner files and the public harness, written read-only into the job's
	// source directory (names.go rule).
	Files []File `json:"files"`
	// HiddenFiles exist only in the compile jail (honor profiles' hidden tests, later). They
	// never reach an exec jail and are never echoed back beyond a file name in a diagnostic.
	HiddenFiles []File `json:"hidden_files,omitempty"`
	// Cases are the case descriptors, in canonical group order; their inputs are the frames
	// that follow job.json, one per case, in this order.
	Cases []CaseInput `json:"cases"`
	// Tests are declared tests (go-race, gotest@1; p-01). Empty until then.
	Tests []TestSpec `json:"tests,omitempty"`
	// Limits are the per-case and compile limits.
	Limits Limits `json:"limits"`
	// StopGroupOn maps a group to its stop rule (t4 §2.6), e.g. {"sample":"any_fail","perf":"tle"}.
	StopGroupOn map[Group]StopRule `json:"stop_group_on,omitempty"`
	// Count is -test.count=N for declared tests (go-race; p-01). Zero until then.
	Count int `json:"count,omitempty"`
	// OutputMode is how fd 4 comes back: bytes, or sha256 + length only.
	OutputMode OutputMode `json:"output_mode"`
}

// File is one source file. Path is a flat file name (names.go), never a path with separators.
type File struct {
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// CaseInput describes one case. Its input bytes are the matching stream frame, whose length
// must equal Size. Expected outputs never enter the runner (judge compares).
type CaseInput struct {
	OpaqueID string `json:"opaque_id"`
	Group    Group  `json:"group"`
	Size     int64  `json:"size"`
}

// TestSpec is one declared test (p-01). Empty until the go-race profile exists.
type TestSpec struct {
	Name string `json:"name"`
}

// Limits are the job's limits. Wall, idle and job deadlines are derived by the runner.
type Limits struct {
	Case    CaseLimits    `json:"case"`
	Compile CompileLimits `json:"compile"`
}

// CaseLimits apply to every case: the CPU time limit (the verdict clock), the memory limit on
// top of the profile's calibrated baseline, and the harness-channel output cap.
type CaseLimits struct {
	CPUms    int64 `json:"cpu_ms"`
	MemMB    int64 `json:"mem_mb"`
	OutputKB int64 `json:"output_kb"`
}

// CompileLimits apply to the compile jail. Zero means the profile default. CPU is capped at
// 15 s (L14) whatever is asked.
type CompileLimits struct {
	CPUms int64 `json:"cpu_ms"`
	MemMB int64 `json:"mem_mb"`
}

// Result is the 200 response body of POST /v1/jobs, and also the body of the typed infra
// responses (only Infra and Versions set).
type Result struct {
	Compile CompileResult  `json:"compile"`
	Cases   []CaseResult   `json:"cases"`
	Tests   []TestEvent    `json:"tests,omitempty"`
	Race    bool           `json:"race,omitempty"`
	Dump    *GoroutineDump `json:"dump,omitempty"`
	// Throttled means no clean quiet re-run was possible (t3 §5.7): judge re-queues once, then
	// inconclusive(runner_throttled). The case verdicts of a throttled Result are not final.
	Throttled bool `json:"throttled"`
	// Infra is set when the runner could not produce a verdict; never a learner verdict.
	Infra     *InfraError `json:"infra,omitempty"`
	Versions  Versions    `json:"versions"`
	Telemetry Telemetry   `json:"telemetry"`
}

// CompileResult is the compile step's outcome. Diags are positioned diagnostics; judge drops
// the ones outside learner files.
type CompileResult struct {
	OK    bool   `json:"ok"`
	Diags []Diag `json:"diags,omitempty"`
	// Limit is set when the compile was stopped by a limit (a CE, L14).
	Limit CompileLimit `json:"limit,omitempty"`
}

// Diag is one compile diagnostic.
type Diag struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
	Msg  string `json:"msg"`
}

// CaseResult is one case's outcome. CPUms, WallMs and PeakKB are measured outside the learner's
// process (cgroup files, pidfd/wait status, the front's pipe counters).
type CaseResult struct {
	OpaqueID string `json:"opaque_id"`
	Term     Term   `json:"term"`
	// Signal is the terminating signal's name (e.g. SIGSYS) when Term is signal.
	Signal   string `json:"signal,omitempty"`
	ExitCode int    `json:"exit_code"`
	CPUms    int64  `json:"cpu_ms"`
	WallMs   int64  `json:"wall_ms"`
	// PeakKB is the case cgroup's memory.peak minus the profile's calibrated baseline.
	PeakKB int64 `json:"peak_kb"`
	// PanicClass is the harness-reported panic class (m3-04), if any.
	PanicClass string `json:"panic_class,omitempty"`
	// Output is the harness channel (fd 4) in bytes mode.
	Output []byte `json:"output,omitempty"`
	// OutputSHA256 is the hex SHA-256 of the fd 4 bytes (always set for a case that ran).
	OutputSHA256 string `json:"output_sha256,omitempty"`
	// OutputBytes is the fd 4 byte count.
	OutputBytes int64 `json:"output_bytes"`
	// Stdout (≤ 8 KiB) and Stderr (≤ 2 KiB) are kept in Run mode only.
	Stdout []byte `json:"stdout,omitempty"`
	Stderr []byte `json:"stderr,omitempty"`
	// Quiet marks a verdict from the in-runner quiet re-run (t3 §5.7); it is final.
	Quiet bool `json:"quiet,omitempty"`
	// IdleKill marks a TLE (idle): the CPU rate stayed under 5% for ≥ 1 s.
	IdleKill bool `json:"idle_kill,omitempty"`
	// ForkLimit marks that the case hit its pids cap (pids.events max > 0): RE (fork limit).
	ForkLimit bool `json:"fork_limit,omitempty"`
}

// TestEvent is one declared-test event (test2json shape; p-01). Output is dropped in Submit.
type TestEvent struct {
	Test    string  `json:"test"`
	Action  string  `json:"action"`
	Elapsed float64 `json:"elapsed,omitempty"`
	Output  string  `json:"output,omitempty"`
}

// GoroutineDump is the go-race per-test timeout dump (p-01): learner-package frames only, no
// argument words, no PC offsets (t3 §5.8).
type GoroutineDump struct {
	Goroutines []DumpGoroutine `json:"goroutines"`
}

// DumpGoroutine is one blocked goroutine.
type DumpGoroutine struct {
	State  string      `json:"state"`
	Frames []DumpFrame `json:"frames"`
}

// DumpFrame is one learner-package frame.
type DumpFrame struct {
	Func string `json:"func"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// Versions identifies what produced a Result. BootEpoch lets judge regrade every evaluation
// from a suspect runner epoch (t3 §2.4 residual 2).
type Versions struct {
	Runner      string `json:"runner"`
	ImageDigest string `json:"image_digest"`
	Profile     string `json:"profile"`
	Toolchain   string `json:"toolchain"`
	ProfileSHA  string `json:"profile_sha256"`
	CPUModel    string `json:"cpu_model"`
	// CanaryMedian is the rolling median of the runner canary's CPU time, in microseconds.
	CanaryMedian int64 `json:"canary_median_us"`
	// BootEpoch is "<hostname>/<container start, unix ns>/<8 random bytes hex>".
	BootEpoch string `json:"boot_epoch"`
}

// Telemetry is internal (never shown to a learner); judge persists it per job (t3 §10).
type Telemetry struct {
	// StealPct is the node steal fraction over the job's test phase, in percent (s1).
	StealPct float64 `json:"steal_pct"`
	// CanaryRatio is the last on-demand canary over the median (s2); 0 when none ran.
	CanaryRatio float64 `json:"canary_ratio"`
	// BusyMs is the wall time the job held its slot (compile, cases, quiet re-runs).
	BusyMs int64 `json:"busy_ms"`
	// OtherSlotBusy is true if the other slot ran a job at any time during this one.
	OtherSlotBusy bool `json:"other_slot_busy"`
	// QuietReruns counts the in-runner quiet re-runs this job took.
	QuietReruns int `json:"quiet_reruns"`
	// SlotMs is the CPU time the job consumed (compile, CE included, plus every case run);
	// judge charges it to the learner's runner budget (L12).
	SlotMs int64 `json:"slot_ms"`
	// TestsCapHit is true when the L14 45 s tests cap fired.
	TestsCapHit bool `json:"tests_cap_hit"`
}

// ProfilesResponse is the body of GET /v1/profiles.
type ProfilesResponse struct {
	Profiles []ProfileInfo `json:"profiles"`
	Runner   string        `json:"runner"`
	// BootEpoch is the serving runner's epoch.
	BootEpoch string `json:"boot_epoch"`
	// CanaryMedian is the canary's rolling median, in microseconds.
	CanaryMedian int64 `json:"canary_median_us"`
	// Mode is "prod" or "dev". Judge refuses a dev runner in production (m3-06).
	Mode string `json:"mode"`
}

// ProfileInfo describes one profile. m3-04 fills the calibration fields (A8 measures them).
type ProfileInfo struct {
	Profile       string   `json:"profile"`
	Toolchain     string   `json:"toolchain"`
	ProfileSHA256 string   `json:"profile_sha256"`
	ImageDigest   string   `json:"image_digest"`
	Harnesses     []string `json:"harnesses"`
	Baseline      string   `json:"baseline"`
	TLMultiplier  float64  `json:"tl_multiplier"`
	Calibrated    bool     `json:"calibrated"`
	MemBaselineKB int64    `json:"mem_baseline_kb"`
}

// StatsResponse is the body of GET /v1/stats: read-only counters, no learner data. The
// acceptance suite (m3-15) and `judge admin` read it on demand; nothing alerts (D34).
type StatsResponse struct {
	Slots []SlotStats `json:"slots"`
	// RunnerMemoryCurrent is runner/'s memory.current (spawner + front).
	RunnerMemoryCurrent int64 `json:"runner_memory_current"`
	// ContainerOOMKills is the container cgroup's hierarchical memory.events oom_kill (every
	// case OOM counts here too).
	ContainerOOMKills int64 `json:"container_oom_kill"`
	// ContainerOOMKillsLocal is memory.events.local oom_kill: OOMs charged to the container
	// itself, which INV-14 says stays 0.
	ContainerOOMKillsLocal int64  `json:"container_oom_kill_local"`
	JobsServed             int64  `json:"jobs_served"`
	SigsysCount            int64  `json:"sigsys_count"`
	BootEpoch              string `json:"boot_epoch"`
	Mode                   string `json:"mode"`
	Draining               bool   `json:"draining"`
}

// SlotStats are one slot's cgroup numbers.
type SlotStats struct {
	Slot               int   `json:"slot"`
	Busy               bool  `json:"busy"`
	MemoryCurrent      int64 `json:"memory_current"`
	PidsCurrent        int64 `json:"pids_current"`
	NrDyingDescendants int64 `json:"nr_dying_descendants"`
}
