package front

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
)

// disposition says how handleJob answers.
type disposition int

const (
	dispOK     disposition = iota // 200 + Result (possibly carrying setup or job_timeout)
	dispKilled                    // 503 + Retry-After + {"infra":{"kind":"killed"}}
	dispNone                      // the client went away: no Result
)

// Compile output kept for diagnostics.
const compileKeep = 64 << 10

// jobRun is one job's orchestration state.
type jobRun struct {
	s      *Server
	slot   int
	job    *runnerapi.Job
	inputs [][]byte
	pe     profileEntry
	res    *runnerapi.Result

	stealSum, wallSum float64
}

// runJob drives one job through the spawner: sources, compile, the case loop (stop rules, the
// L14 tests cap, the quiet re-run) and teardown.
func (s *Server) runJob(ctx context.Context, slot int, seq uint64, job *runnerapi.Job, inputs [][]byte, pe profileEntry) (*runnerapi.Result, disposition) {
	t0 := time.Now()
	res := &runnerapi.Result{Versions: s.baseVersions(), Cases: make([]runnerapi.CaseResult, len(job.Cases))}
	res.Versions.Profile, res.Versions.Toolchain, res.Versions.ProfileSHA = job.Profile, pe.man.Toolchain, pe.man.ProfileSHA256
	for i, c := range job.Cases {
		res.Cases[i] = runnerapi.CaseResult{OpaqueID: c.OpaqueID, Term: runnerapi.TermNotRun}
	}
	jr := &jobRun{s: s, slot: slot, job: job, inputs: inputs, pe: pe, res: res}
	defer func() { res.Telemetry.BusyMs = time.Since(t0).Milliseconds() }()

	compileCPU := firstPositive(job.Limits.Compile.CPUms, pe.prof.Compile.CPUms, measure.CompileCap.Milliseconds())
	compileCPU = min(compileCPU, measure.CompileCap.Milliseconds())
	compileMem := firstPositive(job.Limits.Compile.MemMB, pe.prof.Compile.MemMB, 768)
	begun, err := s.backend.BeginJob(ctx, slot, pe.idx, seq, compileCPU, compileMem)
	if err != nil {
		return jr.setupFailure("begin", err, false)
	}
	res.Versions.CanaryMedian = begun.CanaryMedianUs
	if err := writeSources(begun.SrcDir, job.Files, job.HiddenFiles); err != nil {
		return jr.setupFailure("write sources", err, true)
	}
	if ctx.Err() != nil {
		return jr.abort(ctx)
	}
	if err := jr.compile(ctx); err != nil {
		if ctx.Err() != nil {
			return jr.abort(ctx)
		}
		return jr.setupFailure("compile", err, true)
	}
	if res.Compile.OK {
		if err := jr.runCases(ctx); err != nil {
			if ctx.Err() != nil {
				return jr.abort(ctx)
			}
			return jr.setupFailure("case", err, true)
		}
	}
	td, err := s.backend.EndJob(context.Background(), slot, false)
	if err != nil || td != ipc.TeardownOK {
		s.log.Error("teardown assertion failed", "slot", slot, "outcome", td, "err", err)
		s.markSlotClosed(slot)
		return s.infraResult(runnerapi.InfraSetup, fmt.Sprintf("teardown outcome %d", td)), dispOK
	}
	if jr.wallSum > 0 {
		res.Telemetry.StealPct = 100 * jr.stealSum / jr.wallSum
	}
	return res, dispOK
}

func (s *Server) infraResult(kind runnerapi.InfraKind, detail string) *runnerapi.Result {
	return &runnerapi.Result{Infra: runnerapi.NewInfra(kind, detail), Versions: s.baseVersions()}
}

// setupFailure answers infra: setup (a runner fault before or around learner code) after
// aborting the job if it began.
func (jr *jobRun) setupFailure(stage string, err error, began bool) (*runnerapi.Result, disposition) {
	jr.s.log.Error("job setup failed", "stage", stage, "slot", jr.slot, "err", err)
	if began {
		if td, eerr := jr.s.backend.EndJob(context.Background(), jr.slot, true); eerr != nil || td != ipc.TeardownOK {
			jr.s.markSlotClosed(jr.slot)
		}
	}
	var fe *FailError
	if errors.As(err, &fe) && fe.Code == ipc.FailSlotClosed {
		jr.s.markSlotClosed(jr.slot)
	}
	return jr.s.infraResult(runnerapi.InfraSetup, stage), dispOK
}

// abort kills the job subtree and answers per the context's cause: the drain's kill → killed
// (503), the 170 s backstop → job_timeout, a client disconnect → no Result.
func (jr *jobRun) abort(ctx context.Context) (*runnerapi.Result, disposition) {
	if td, err := jr.s.backend.EndJob(context.Background(), jr.slot, true); err != nil || td != ipc.TeardownOK {
		jr.s.log.Error("teardown after an abort failed", "slot", jr.slot, "outcome", td, "err", err)
		jr.s.markSlotClosed(jr.slot)
	}
	switch cause := context.Cause(ctx); {
	case errors.Is(cause, errKilled):
		return jr.s.infraResult(runnerapi.InfraKilled, "draining"), dispKilled
	case errors.Is(cause, errJobTimeout):
		jr.s.log.Error("job_timeout: the runner's own deadline passed", "slot", jr.slot)
		return jr.s.infraResult(runnerapi.InfraJobTimeout, ""), dispOK
	default:
		return nil, dispNone
	}
}

func firstPositive(vs ...int64) int64 {
	for _, v := range vs {
		if v > 0 {
			return v
		}
	}
	return 0
}

// writeSources re-checks the names (t3 §2.4 A5) and writes Files and HiddenFiles 0444 through
// the directory fd, then makes the directory 0555 so the compile UID can read and nobody can
// write. The front owns the directory; the spawner never reads or chmods it (R-FS).
func writeSources(dir *os.File, files, hidden []runnerapi.File) error {
	defer dir.Close()
	if err := runnerapi.ValidateFiles(files, hidden); err != nil {
		return err
	}
	dfd := int(dir.Fd())
	for _, set := range [][]runnerapi.File{files, hidden} {
		for _, f := range set {
			fd, err := unix.Openat(dfd, f.Path, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o444)
			if err != nil {
				return fmt.Errorf("create %s: %w", f.Path, err)
			}
			w := os.NewFile(uintptr(fd), f.Path)
			_, err = w.Write(f.Data)
			cerr := w.Close()
			if err != nil || cerr != nil {
				return fmt.Errorf("write %s: %v %v", f.Path, err, cerr)
			}
		}
	}
	return unix.Fchmod(dfd, 0o555)
}

// ---- compile ----

func (jr *jobRun) compile(ctx context.Context) error {
	run, err := jr.s.backend.Compile(ctx, jr.slot)
	if err != nil {
		return err
	}
	stdout := &keeper{keep: compileKeep}
	stderr := &keeper{keep: compileKeep}
	wait := drainAll(map[*os.File]io.Writer{run.Stdout: stdout, run.Stderr: stderr})
	var o CompileOutcome
	select {
	case o = <-run.Done:
	case <-ctx.Done():
		wait(0)
		return ctx.Err()
	}
	wait(5 * time.Second)
	if o.Err != nil {
		return o.Err
	}
	d := o.Done
	jr.res.Telemetry.SlotMs += int64(d.CPUus / 1000)
	switch {
	case d.Export == uint8(ipc.ExportFailed):
		return errors.New("artifact export failed")
	case d.Exited == 1 && d.ExitCode == 0 && d.Kill == 0 && d.Export == uint8(ipc.ExportOK):
		jr.res.Compile.OK = true
	default:
		jr.res.Compile.Limit = compileLimit(d)
		names := map[string]bool{}
		for _, f := range jr.job.Files {
			names[f.Path] = true
		}
		for _, f := range jr.job.HiddenFiles {
			names[f.Path] = true
		}
		jr.res.Compile.Diags = parseDiags(append(stderr.kept, stdout.kept...), names)
	}
	return nil
}

func compileLimit(d *ipc.CompileDone) runnerapi.CompileLimit {
	switch {
	case d.OOMKills > 0:
		return runnerapi.CompileLimitMemory
	case measure.Kill(d.Kill) == measure.KillCPU || d.Signal == measure.SIGXCPU:
		return runnerapi.CompileLimitCPU
	case measure.Kill(d.Kill) == measure.KillCap:
		return runnerapi.CompileLimitWall
	case d.Export == uint8(ipc.ExportTooBig):
		return runnerapi.CompileLimitOutput
	case d.PidsMax > 0:
		return runnerapi.CompileLimitPids
	}
	return ""
}

var diagRE = regexp.MustCompile(`(?m)^(?:\./|/src/)?([A-Za-z0-9_]+\.[A-Za-z0-9_]+):(\d+)(?::(\d+))?: (.+)$`)

// parseDiags extracts "file:line[:col]: msg" diagnostics for the job's own file names (a
// generic parser; m3-04 adds per-profile ones such as `go build -json`).
func parseDiags(out []byte, names map[string]bool) []runnerapi.Diag {
	var diags []runnerapi.Diag
	for _, m := range diagRE.FindAllSubmatch(out, -1) {
		file := string(m[1])
		if !names[file] {
			continue
		}
		line, _ := strconv.Atoi(string(m[2]))
		col, _ := strconv.Atoi(string(m[3]))
		msg := string(m[4])
		if len(msg) > runnerapi.MaxDiagMsgBytes {
			msg = msg[:runnerapi.MaxDiagMsgBytes]
		}
		diags = append(diags, runnerapi.Diag{File: file, Line: line, Col: col, Msg: msg})
		if len(diags) == runnerapi.MaxDiags {
			break
		}
	}
	return diags
}

// ---- cases ----

// runCases applies t4 §2.6's order and stop rules, the L14 tests cap and the quiet re-run.
func (jr *jobRun) runCases(ctx context.Context) error {
	job, res := jr.job, jr.res
	var (
		testsStart  = time.Now()
		quietTotal  time.Duration // excluded from the main cap
		quietStart  time.Time
		quietGroup  runnerapi.Group
		quietUsed   bool
		stopAll     bool
		stopGroup   = map[runnerapi.Group]bool{}
		leaveQuiet  = func() {}
		capCaseDone *ipc.CaseDone
	)
	defer func() { leaveQuiet() }()
	for i := 0; i < len(job.Cases); {
		c := job.Cases[i]
		if quietGroup != "" && c.Group != quietGroup {
			quietTotal += time.Since(quietStart)
			quietGroup = ""
			leaveQuiet()
			leaveQuiet = func() {}
		}
		if stopAll || stopGroup[c.Group] {
			i++
			continue
		}
		var left time.Duration
		if quietGroup != "" {
			left = measure.TestsCap - time.Since(quietStart)
		} else {
			left = measure.TestsCap - (time.Since(testsStart) - quietTotal)
		}
		if left <= 0 {
			res.Telemetry.TestsCapHit = true
			break
		}
		cr, oc, done, err := jr.runOne(ctx, i, quietGroup != "", left)
		if err != nil {
			return err
		}
		if done.OtherBusy == 1 {
			res.Telemetry.OtherSlotBusy = true
		}
		if done.CanaryPermille > 0 {
			res.Telemetry.CanaryRatio = float64(done.CanaryPermille) / 1000
		}
		if oc.CapHit {
			res.Cases[i] = cr
			res.Telemetry.TestsCapHit = true
			capCaseDone = done
			break
		}
		if oc.TimeFail && oc.Suspect && quietGroup == "" {
			if quietUsed {
				res.Throttled = true // one quiet re-run per job (the 170 s arithmetic assumes one)
				return nil
			}
			release, ok := jr.enterQuiet(ctx)
			if !ok {
				res.Throttled = true
				return nil
			}
			quietUsed, quietGroup, quietStart, leaveQuiet = true, c.Group, time.Now(), release
			res.Telemetry.QuietReruns++
			continue // re-run this case, quietly
		}
		if quietGroup != "" && oc.TimeFail &&
			(float64(done.StealPermille)/1000 >= measure.StealSuspect || float64(done.CanaryPermille)/1000 >= measure.CanarySuspect) {
			res.Throttled = true // s1/s2 fired during the quiet re-run
			return nil
		}
		res.Cases[i] = cr
		switch job.StopGroupOn[c.Group] {
		case runnerapi.StopAnyFail:
			if cr.Term != runnerapi.TermOK {
				stopAll = true
			}
		case runnerapi.StopTLE:
			if cr.Term == runnerapi.TermTLE {
				stopGroup[c.Group] = true
			}
		}
		i++
	}
	// The L14 cap is TLE, counted — unless the job was suspect while it ran: then no verdict.
	if res.Telemetry.TestsCapHit {
		suspect := res.Telemetry.OtherSlotBusy
		if capCaseDone != nil {
			suspect = suspect || capCaseDone.OtherBusy == 1 ||
				float64(capCaseDone.StealPermille)/1000 >= measure.StealSuspect ||
				float64(capCaseDone.CanaryPermille)/1000 >= measure.CanarySuspect
		}
		if suspect {
			res.Throttled = true
		}
	}
	return nil
}

// enterQuiet takes the runner's exclusivity for a quiet re-run (t3 §5.7): stop admitting,
// wait ≤ 60 s for the other slot to drain, then require steal < 5% over 10 s and a pre-canary
// ≤ 1.1 × median. It returns a release func and whether quiet conditions were met.
func (jr *jobRun) enterQuiet(ctx context.Context) (func(), bool) {
	s := jr.s
	s.mu.Lock()
	if s.quiet >= 0 {
		s.mu.Unlock()
		return func() {}, false // the other slot holds exclusivity
	}
	s.quiet = jr.slot
	s.mu.Unlock()
	release := func() {
		s.mu.Lock()
		if s.quiet == jr.slot {
			s.quiet = -1
		}
		s.mu.Unlock()
	}
	deadline := time.Now().Add(s.quietWait)
	for {
		s.mu.Lock()
		otherBusy := s.busy[1-jr.slot]
		s.mu.Unlock()
		if !otherBusy {
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			release()
			return func() {}, false
		}
		time.Sleep(50 * time.Millisecond)
	}
	for time.Until(deadline) > s.quietCheck {
		qs, err := s.backend.QuietCheck(ctx, jr.slot)
		if err != nil {
			break
		}
		if qs.Steal < measure.QuietSteal && qs.Canary <= measure.QuietCanary {
			return release, true
		}
		s.log.Info("quiet conditions not met yet", "slot", jr.slot, "steal", qs.Steal, "canary", qs.Canary)
	}
	release()
	return func() {}, false
}

// runOne runs case i and classifies it.
func (jr *jobRun) runOne(ctx context.Context, i int, quiet bool, left time.Duration) (runnerapi.CaseResult, measure.Outcome, *ipc.CaseDone, error) {
	job := jr.job
	req := ipc.CaseRun{
		CaseIdx:   uint32(i),
		CPUms:     uint32(job.Limits.Case.CPUms),
		MemMB:     uint32(job.Limits.Case.MemMB),
		WallCapMs: uint32(min(max(left.Milliseconds(), 1), measure.TestsCap.Milliseconds())),
	}
	if quiet {
		req.Quiet = 1
	}
	run, err := jr.s.backend.RunCase(ctx, jr.slot, req)
	if err != nil {
		return runnerapi.CaseResult{}, measure.Outcome{}, nil, err
	}
	input := jr.inputs[i]
	go func() {
		_, _ = run.Input.Write(input)
		run.Input.Close()
	}()
	var ole atomic.Bool
	overCap := func() {
		if ole.CompareAndSwap(false, true) {
			run.Kill()
		}
	}
	keepOut, keepErr := 0, 0
	if job.Mode == runnerapi.ModeRun {
		keepOut, keepErr = runnerapi.KeptStdoutBytes, runnerapi.KeptStderrBytes
	}
	stdout := &keeper{keep: keepOut, hard: runnerapi.HardFdCapBytes, over: overCap}
	stderr := &keeper{keep: keepErr, hard: runnerapi.HardFdCapBytes, over: overCap}
	outCap := int(job.Limits.Case.OutputKB << 10)
	result := &keeper{keep: outCap, hard: outCap, over: overCap, h: sha256.New()}
	if job.OutputMode == runnerapi.OutputSHA256 {
		result.keep = 0
	}
	wait := drainAll(map[*os.File]io.Writer{run.Stdout: stdout, run.Stderr: stderr, run.Result: result})
	var o CaseOutcome
	select {
	case o = <-run.Done:
	case <-ctx.Done():
		wait(0)
		return runnerapi.CaseResult{}, measure.Outcome{}, nil, ctx.Err()
	}
	wait(5 * time.Second)
	if o.Err != nil {
		return runnerapi.CaseResult{}, measure.Outcome{}, nil, o.Err
	}
	d := o.Done
	tl := time.Duration(job.Limits.Case.CPUms) * time.Millisecond
	oc := measure.Classify(measure.Evidence{
		Kill: measure.Kill(d.Kill), Exited: d.Exited == 1, ExitCode: int(d.ExitCode), Signal: int(d.Signal),
		OOMKills: int64(d.OOMKills), PidsMax: int64(d.PidsMax),
		CPU: time.Duration(d.CPUus) * time.Microsecond, Wall: time.Duration(d.WallUs) * time.Microsecond,
		PeakBytes: int64(d.PeakBytes), Baseline: int64(d.BaselineBytes), TL: tl, FrontOLE: ole.Load(),
		OtherBusy: d.OtherBusy == 1, Steal: float64(d.StealPermille) / 1000, CanaryRate: float64(d.CanaryPermille) / 1000,
	})
	cr := runnerapi.CaseResult{
		OpaqueID: job.Cases[i].OpaqueID, Term: oc.Term, Signal: oc.Signal, ExitCode: oc.ExitCode,
		CPUms: int64(d.CPUus / 1000), WallMs: int64(d.WallUs / 1000), PeakKB: oc.PeakKB,
		OutputBytes: result.n, OutputSHA256: hex.EncodeToString(result.h.Sum(nil)),
		Quiet: quiet, IdleKill: oc.IdleKill, ForkLimit: oc.ForkLimit,
	}
	if job.OutputMode == runnerapi.OutputBytes && len(result.kept) > 0 {
		cr.Output = result.kept
	}
	if job.Mode == runnerapi.ModeRun {
		cr.Stdout, cr.Stderr = nonEmpty(stdout.kept), nonEmpty(stderr.kept)
	}
	if oc.Signal == "SIGSYS" {
		jr.s.sigsys.Add(1)
		jr.s.log.Error("SIGSYS from a jail (RE, counted); the runner rotates after this job", "slot", jr.slot, "case", i)
	}
	jr.res.Telemetry.SlotMs += cr.CPUms
	wall := float64(d.WallUs)
	jr.stealSum += float64(d.StealPermille) / 1000 * wall
	jr.wallSum += wall
	return cr, oc, d, nil
}

func nonEmpty(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

// keeper counts, keeps up to keep bytes, optionally hashes, and calls over once past hard.
type keeper struct {
	keep, hard int
	over       func()
	h          hash.Hash
	n          int64
	kept       []byte
}

func (k *keeper) Write(p []byte) (int, error) {
	prev := k.n
	k.n += int64(len(p))
	if k.hard > 0 && prev <= int64(k.hard) {
		// Hash and keep only the bytes inside the cap.
		inCap := p
		if k.n > int64(k.hard) {
			inCap = p[:int64(k.hard)-prev]
		}
		if k.h != nil {
			k.h.Write(inCap)
		}
		if room := k.keep - len(k.kept); room > 0 {
			k.kept = append(k.kept, inCap[:min(room, len(inCap))]...)
		}
	} else if k.hard == 0 {
		if room := k.keep - len(k.kept); room > 0 {
			k.kept = append(k.kept, p[:min(room, len(p))]...)
		}
	}
	if k.hard > 0 && k.n > int64(k.hard) && k.over != nil {
		k.over()
	}
	return len(p), nil
}

// drainAll copies each file into its writer in the background. The returned wait blocks up to
// timeout for EOF on all of them, then closes the files. The files are pollable (the backend
// sets them non-blocking), so Close unblocks a pending Read.
func drainAll(m map[*os.File]io.Writer) func(time.Duration) {
	var wg sync.WaitGroup
	files := make([]*os.File, 0, len(m))
	for f, w := range m {
		files = append(files, f)
		wg.Add(1)
		go func(f *os.File, w io.Writer) {
			defer wg.Done()
			_, _ = io.Copy(w, f)
		}(f, w)
	}
	return func(timeout time.Duration) {
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		if timeout > 0 {
			select {
			case <-done:
			case <-time.After(timeout):
			}
		}
		for _, f := range files {
			f.Close()
		}
		<-done
	}
}
