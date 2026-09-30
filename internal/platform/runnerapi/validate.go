package runnerapi

import (
	"encoding/hex"
	"fmt"
	"regexp"
)

// Limit ranges ValidateJob enforces (t3 §6.2, §7.2; L14).
const (
	MinCaseCPUms    = 100
	MaxCaseCPUms    = 10_000
	MinCaseMemMB    = 16
	MaxCaseMemMB    = 1024 // a slot is 1.25 GiB; the profile baseline sits on top
	MinCaseOutputKB = 1
	MaxCaseOutputKB = 8192 // the harness channel's cap; fds 1/2 have their own 1 MiB cap
	MaxCompileCPUms = 15_000
	MaxCompileMemMB = 1024
	MaxCount        = 100
	MaxTests        = 256
)

// Result caps ValidateResult enforces.
const (
	KeptStdoutBytes = 8 << 10 // stdout kept in Run mode
	KeptStderrBytes = 2 << 10 // stderr kept in Run mode
	HardFdCapBytes  = 1 << 20 // fds 1 and 2: past 1 MiB the case is OLE
	MaxDiags        = 256
	MaxDiagMsgBytes = 1 << 10
)

var (
	idRE      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	profileRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}@[0-9A-Za-z.+-]{1,31}$`)
	harnessRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}@[0-9]{1,4}$`)
	testRE    = regexp.MustCompile(`^[A-Za-z0-9_/.-]{1,128}$`)
	signalRE  = regexp.MustCompile(`^SIG[A-Z0-9]{1,12}$`)
)

// ValidationError is a contract violation (a 400 from the runner; infra: setup in judge when
// ValidateResult fails).
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Reason }

func invalid(field, format string, a ...any) error {
	return &ValidationError{Field: field, Reason: fmt.Sprintf(format, a...)}
}

// ValidateJob checks a decoded Job: enums, limits in range, file names (names.go), unique
// case ids, canonical group order and the stop rules. The front returns 400 on an error.
// Whether the profile and harness exist is the runner's check, not the contract's.
func ValidateJob(job *Job) error {
	if job == nil {
		return invalid("job", "missing")
	}
	if !idRE.MatchString(job.ID) {
		return invalid("id", "want [A-Za-z0-9_-]{1,64}")
	}
	if !profileRE.MatchString(job.Profile) {
		return invalid("profile", "want <name>@<version>")
	}
	if !harnessRE.MatchString(job.Harness) {
		return invalid("harness", "want <name>@<major>")
	}
	if job.Mode != ModeRun && job.Mode != ModeSubmit {
		return invalid("mode", "want run or submit")
	}
	if job.OutputMode != OutputBytes && job.OutputMode != OutputSHA256 {
		return invalid("output_mode", "want bytes or sha256")
	}
	if len(job.Files) == 0 {
		return invalid("files", "at least one file")
	}
	if err := ValidateFiles(job.Files, job.HiddenFiles); err != nil {
		return invalid("files", "%v", err)
	}
	if err := validateLimits(job.Limits); err != nil {
		return err
	}
	if len(job.Cases) > MaxCases {
		return invalid("cases", "%d cases, cap %d", len(job.Cases), MaxCases)
	}
	ids := make(map[string]bool, len(job.Cases))
	last := -1
	var total int64
	for i, c := range job.Cases {
		if !idRE.MatchString(c.OpaqueID) {
			return invalid(fmt.Sprintf("cases[%d].opaque_id", i), "want [A-Za-z0-9_-]{1,64}")
		}
		if ids[c.OpaqueID] {
			return invalid(fmt.Sprintf("cases[%d].opaque_id", i), "duplicate %q", c.OpaqueID)
		}
		ids[c.OpaqueID] = true
		rank, ok := groupRank[c.Group]
		if !ok {
			return invalid(fmt.Sprintf("cases[%d].group", i), "unknown group %q", c.Group)
		}
		if rank < last {
			return invalid(fmt.Sprintf("cases[%d].group", i), "groups out of order (want sample, edge, random, perf)")
		}
		last = rank
		if c.Size < 0 || c.Size > MaxCaseInputBytes {
			return invalid(fmt.Sprintf("cases[%d].size", i), "out of range")
		}
		total += c.Size
	}
	if total > MaxInputsBytes {
		return invalid("cases", "inputs exceed %d bytes", MaxInputsBytes)
	}
	for g, rule := range job.StopGroupOn {
		if _, ok := groupRank[g]; !ok {
			return invalid("stop_group_on", "unknown group %q", g)
		}
		if rule != StopAnyFail && rule != StopTLE {
			return invalid("stop_group_on", "unknown rule %q", rule)
		}
	}
	if job.Count < 0 || job.Count > MaxCount {
		return invalid("count", "want 0..%d", MaxCount)
	}
	if len(job.Tests) > MaxTests {
		return invalid("tests", "%d tests, cap %d", len(job.Tests), MaxTests)
	}
	names := make(map[string]bool, len(job.Tests))
	for i, t := range job.Tests {
		if !testRE.MatchString(t.Name) {
			return invalid(fmt.Sprintf("tests[%d].name", i), "bad test name")
		}
		if names[t.Name] {
			return invalid(fmt.Sprintf("tests[%d].name", i), "duplicate %q", t.Name)
		}
		names[t.Name] = true
	}
	return nil
}

func validateLimits(l Limits) error {
	if l.Case.CPUms < MinCaseCPUms || l.Case.CPUms > MaxCaseCPUms {
		return invalid("limits.case.cpu_ms", "want %d..%d", MinCaseCPUms, MaxCaseCPUms)
	}
	if l.Case.MemMB < MinCaseMemMB || l.Case.MemMB > MaxCaseMemMB {
		return invalid("limits.case.mem_mb", "want %d..%d", MinCaseMemMB, MaxCaseMemMB)
	}
	if l.Case.OutputKB < MinCaseOutputKB || l.Case.OutputKB > MaxCaseOutputKB {
		return invalid("limits.case.output_kb", "want %d..%d", MinCaseOutputKB, MaxCaseOutputKB)
	}
	if l.Compile.CPUms < 0 || l.Compile.CPUms > MaxCompileCPUms {
		return invalid("limits.compile.cpu_ms", "want 0..%d (0 = profile default)", MaxCompileCPUms)
	}
	if l.Compile.MemMB < 0 || l.Compile.MemMB > MaxCompileMemMB {
		return invalid("limits.compile.mem_mb", "want 0..%d (0 = profile default)", MaxCompileMemMB)
	}
	return nil
}

var terms = map[Term]bool{
	TermOK: true, TermTLE: true, TermMLE: true, TermOLE: true, TermSignal: true, TermExitNonzero: true, TermNotRun: true,
}

// Valid reports whether t is in the closed set.
func (t Term) Valid() bool { return terms[t] }

var compileLimits = map[CompileLimit]bool{
	CompileLimitCPU: true, CompileLimitWall: true, CompileLimitMemory: true, CompileLimitPids: true, CompileLimitOutput: true,
}

// ValidateResult is judge's check of a runner Result against the Job it sent (t3 §5.3): case
// ids and counts match, enums are closed, size caps hold, Versions.Profile is the job's
// profile and BootEpoch is present. A mismatch is infra: setup in judge.
//
// A Result carrying Infra only needs a valid kind and a BootEpoch: it has no verdicts.
func ValidateResult(job *Job, res *Result) error {
	if job == nil || res == nil {
		return invalid("result", "missing job or result")
	}
	if res.Versions.BootEpoch == "" {
		return invalid("versions.boot_epoch", "missing")
	}
	if res.Infra != nil {
		if !res.Infra.Kind.Valid() {
			return invalid("infra.kind", "unknown kind %q", res.Infra.Kind)
		}
		return nil
	}
	if res.Versions.Profile != job.Profile {
		return invalid("versions.profile", "%q, job asked %q", res.Versions.Profile, job.Profile)
	}
	if len(res.Cases) != len(job.Cases) {
		return invalid("cases", "%d results for %d cases", len(res.Cases), len(job.Cases))
	}
	if res.Compile.Limit != "" && !compileLimits[res.Compile.Limit] {
		return invalid("compile.limit", "unknown limit %q", res.Compile.Limit)
	}
	if res.Compile.OK && res.Compile.Limit != "" {
		return invalid("compile.limit", "set on a successful compile")
	}
	if len(res.Compile.Diags) > MaxDiags {
		return invalid("compile.diags", "%d diagnostics, cap %d", len(res.Compile.Diags), MaxDiags)
	}
	for i, d := range res.Compile.Diags {
		if len(d.Msg) > MaxDiagMsgBytes || len(d.File) > MaxFileNameBytes || d.Line < 0 || d.Col < 0 {
			return invalid(fmt.Sprintf("compile.diags[%d]", i), "out of range")
		}
	}
	outCap := job.Limits.Case.OutputKB << 10
	for i, c := range res.Cases {
		field := fmt.Sprintf("cases[%d]", i)
		if c.OpaqueID != job.Cases[i].OpaqueID {
			return invalid(field+".opaque_id", "%q, want %q", c.OpaqueID, job.Cases[i].OpaqueID)
		}
		if !c.Term.Valid() {
			return invalid(field+".term", "unknown term %q", c.Term)
		}
		if !res.Compile.OK && c.Term != TermNotRun {
			return invalid(field+".term", "a case ran after a failed compile")
		}
		if c.Term == TermSignal && !signalRE.MatchString(c.Signal) {
			return invalid(field+".signal", "want a signal name")
		}
		if c.Term != TermSignal && c.Signal != "" {
			return invalid(field+".signal", "set without term signal")
		}
		if c.CPUms < 0 || c.WallMs < 0 || c.PeakKB < 0 || c.OutputBytes < 0 {
			return invalid(field, "negative measurement")
		}
		if int64(len(c.Output)) > outCap {
			return invalid(field+".output", "%d bytes, cap %d", len(c.Output), outCap)
		}
		if job.OutputMode == OutputSHA256 && len(c.Output) != 0 {
			return invalid(field+".output", "bytes present in sha256 mode")
		}
		if c.OutputSHA256 != "" {
			if b, err := hex.DecodeString(c.OutputSHA256); err != nil || len(b) != 32 {
				return invalid(field+".output_sha256", "want 64 hex chars")
			}
		}
		if job.OutputMode == OutputSHA256 && c.Term != TermNotRun && c.OutputSHA256 == "" {
			return invalid(field+".output_sha256", "missing in sha256 mode")
		}
		if len(c.Stdout) > KeptStdoutBytes || len(c.Stderr) > KeptStderrBytes {
			return invalid(field, "stdout/stderr over the kept caps")
		}
		if job.Mode == ModeSubmit && (len(c.Stdout) != 0 || len(c.Stderr) != 0) {
			return invalid(field, "stdout/stderr present in submit mode")
		}
		if c.IdleKill && c.Term != TermTLE {
			return invalid(field+".idle_kill", "set without term tle")
		}
	}
	return nil
}
