//go:build linux

package spawner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/criyle/go-sandbox/pkg/rlimit"
	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/cgroup"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/jail"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

// Artifact limits (t3 §5.4 step 4).
const (
	artifactMax   = 64 << 20
	artifactTmpfs = "size=65m,nr_inodes=16,mode=0700,uid=0,gid=0"
	srcTmpfsFmt   = "size=2m,nr_inodes=64,mode=0700,uid=%d,gid=%d"
	compileInit   = "/.xl/runner"
)

// slot is one of the two job slots.
type slot struct {
	idx                     int
	rootDir, srcDir, artDir string

	mu          sync.Mutex
	job         *job
	closed      bool
	baseline    int64
	activeSince time.Time
	lastEnd     time.Time
}

// job is the slot's current job.
type job struct {
	seq        uint64
	prof       *profile.Profile
	ps         *profState
	compileUID int
	jobUID     int
	cg         string
	compileCPU time.Duration
	compileMem int64

	// guarded by slot.mu
	compileStarted bool
	compiled       bool
	running        bool
	caseN          int
	kill           chan measure.Kill
	aborted        bool

	abort chan struct{} // closed on an aborting JobEnd
	ops   sync.WaitGroup
}

// busy reports whether the slot ran a job at any time in [from, to].
func (sl *slot) busyDuring(from, to time.Time) bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if !sl.activeSince.IsZero() && sl.activeSince.Before(to) {
		return true
	}
	return sl.lastEnd.After(from)
}

func (s *Spawner) other(i int) *slot { return s.slots[1-i] }

// ---- JobBegin ----

func (s *Spawner) handleJobBegin(m *ipc.Msg) {
	b := m.Body.(*ipc.JobBegin)
	fd, code, err := s.beginJob(int(m.Slot), int(b.ProfileIdx), b.JobSeq,
		time.Duration(b.CompileCPUms)*time.Millisecond, int64(b.CompileMemMB)<<20, s.cfg.FrontUID)
	if err != nil {
		s.fail(m, code, err)
		return
	}
	s.reply(m, ipc.TJobBegun, &ipc.JobBegun{CanaryMedianUs: uint64(s.canaryMedian().Microseconds())}, fd)
	unix.Close(fd)
}

// beginJob sets a job up: its cgroup subtree and a fresh src tmpfs owned by srcOwner (the
// front's UID; 0 for the startup baseline). It returns an fd on the src directory.
func (s *Spawner) beginJob(slotIdx, pidx int, seq uint64, compileCPU time.Duration, compileMem int64, srcOwner int) (int, ipc.FailCode, error) {
	p, ok := profile.ByIndex(pidx)
	if !ok {
		return -1, ipc.FailBadRequest, fmt.Errorf("unknown profile index %d", pidx)
	}
	sl := s.slots[slotIdx]
	// Let a running idle canary finish first so it never shares the slot's CPU with a job.
	s.canaryMu.Lock()
	s.canaryMu.Unlock()

	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.closed {
		return -1, ipc.FailSlotClosed, fmt.Errorf("slot %d failed closed", slotIdx)
	}
	if sl.job != nil {
		return -1, ipc.FailBadRequest, fmt.Errorf("slot %d is busy", slotIdx)
	}
	j := &job{seq: seq, prof: p, ps: s.pstate[pidx], compileCPU: min(compileCPU, measure.CompileCap), compileMem: compileMem,
		cg: filepath.Join(s.cg.Slot(slotIdx), "job"), abort: make(chan struct{})}
	j.compileUID, j.jobUID = s.nextUIDs()
	if err := cgroup.MkInner(j.cg); err != nil {
		return -1, ipc.FailSetup, err
	}
	opts := fmt.Sprintf(srcTmpfsFmt, srcOwner, srcOwner)
	if err := unix.Mount("tmpfs", sl.srcDir, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, opts); err != nil {
		_ = s.cg.Wipe(j.cg)
		return -1, ipc.FailSetup, fmt.Errorf("mount the src tmpfs: %w", err)
	}
	fd, err := unix.Open(sl.srcDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		_ = unix.Unmount(sl.srcDir, unix.MNT_DETACH)
		_ = s.cg.Wipe(j.cg)
		return -1, ipc.FailSetup, fmt.Errorf("open the src dir: %w", err)
	}
	sl.job = j
	sl.activeSince = time.Now()
	return fd, 0, nil
}

// startOp marks an operation in flight on the slot's job.
func (s *Spawner) startOp(slotIdx int, check func(j *job) error) (*job, ipc.FailCode, error) {
	sl := s.slots[slotIdx]
	sl.mu.Lock()
	defer sl.mu.Unlock()
	j := sl.job
	if j == nil {
		return nil, ipc.FailBadRequest, fmt.Errorf("slot %d has no job", slotIdx)
	}
	if j.running || j.aborted {
		return nil, ipc.FailBadRequest, fmt.Errorf("slot %d is busy or aborting", slotIdx)
	}
	if err := check(j); err != nil {
		return nil, ipc.FailBadRequest, err
	}
	j.running = true
	j.ops.Add(1)
	return j, 0, nil
}

func (s *Spawner) endOp(slotIdx int, j *job) {
	sl := s.slots[slotIdx]
	sl.mu.Lock()
	j.running = false
	j.kill = nil
	sl.mu.Unlock()
	j.ops.Done()
}

// ---- Compile ----

// compileHooks lets the IPC path hand the output pipes to the front and the startup path
// drain them itself.
type compileHooks struct {
	started func(outR, errR *os.File)
}

func (s *Spawner) handleCompile(m *ipc.Msg) {
	slotIdx := int(m.Slot)
	done, code, err := s.compile(slotIdx, compileHooks{started: func(outR, errR *os.File) {
		s.reply(m, ipc.TCompileStarted, &ipc.CompileStarted{}, int(outR.Fd()), int(errR.Fd()))
	}})
	if err != nil {
		s.fail(m, code, err)
		return
	}
	s.reply(m, ipc.TCompileDone, done)
}

func (s *Spawner) compile(slotIdx int, hooks compileHooks) (*ipc.CompileDone, ipc.FailCode, error) {
	j, code, err := s.startOp(slotIdx, func(j *job) error {
		if j.compileStarted {
			return errors.New("compile already ran")
		}
		j.compileStarted = true
		return nil
	})
	if err != nil {
		return nil, code, err
	}
	defer s.endOp(slotIdx, j)
	sl := s.slots[slotIdx]
	p := j.prof

	// The artifact tmpfs and file exist before the compile starts; the compile jail never sees
	// them. The spawner writes the export pipe's raw bytes here, then sets the modes.
	if err := unix.Mount("tmpfs", sl.artDir, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, artifactTmpfs); err != nil {
		return nil, ipc.FailSetup, fmt.Errorf("mount the artifact tmpfs: %w", err)
	}
	artPath := filepath.Join(sl.artDir, p.ArtifactName)
	afd, err := unix.Open(artPath, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, ipc.FailSetup, fmt.Errorf("create the artifact: %w", err)
	}
	art := os.NewFile(uintptr(afd), "artifact")
	defer art.Close()

	leaf := filepath.Join(j.cg, "compile")
	if err := cgroup.MkLeaf(leaf, cgroup.Limits{MemoryMax: j.compileMem, PidsMax: p.Compile.Pids}); err != nil {
		return nil, ipc.FailSetup, err
	}
	defer func() { _ = s.cg.Remove(leaf) }()

	pipes, err := newPipes(3)
	if err != nil {
		return nil, ipc.FailSetup, err
	}
	defer pipes.closeAll()
	outR, outW := pipes.pair(0)
	errR, errW := pipes.pair(1)
	expR, expW := pipes.pair(2)

	argv := append([]string{compileInit, "compile-init", "-out", p.Compile.OutputPath, "-max", strconv.Itoa(artifactMax), "--"}, p.Compile.Argv...)
	binds := []jail.Bind{{Source: s.exe, Target: compileInit}, {Source: sl.srcDir, Target: "/src"}}
	binds = append(binds, toJailBinds(p.Compile.Binds)...)
	spec := jail.Spec{
		Argv: argv, Env: p.Compile.Env, WorkDir: "/src", UID: j.compileUID, RootDir: sl.rootDir, Binds: binds,
		WTmpfs: fmt.Sprintf("size=%dm,nr_inodes=%d,mode=0700", orDefault(p.Compile.WMB, 256), orDefault(p.Compile.WInodes, 16384)),
		RLimits: []rlimit.RLimit{
			{Res: unix.RLIMIT_CPU, Rlim: syscall.Rlimit{Cur: measure.RlimitCPUSeconds(j.compileCPU), Max: measure.RlimitCPUSeconds(j.compileCPU)}},
			{Res: unix.RLIMIT_FSIZE, Rlim: syscall.Rlimit{Cur: artifactMax + 1<<20, Max: artifactMax + 1<<20}},
		},
		Seccomp: j.ps.compileProg,
		Files:   []uintptr{s.devnull.Fd(), outW.Fd(), errW.Fd(), expW.Fd()},
	}
	pid, closeCg, err := s.startJail(&spec, leaf)
	if err != nil {
		return nil, ipc.FailSetup, err
	}
	closeCg()
	pipes.closeChildEnds(outW, errW, expW)
	hooks.started(outR, errR)
	pipes.closeFrontEnds(outR, errR)

	// Copy the export pipe's raw bytes into the artifact file (no parsing), capped.
	type copyRes struct {
		n   int64
		err error
	}
	copied := make(chan copyRes, 1)
	go func() {
		n, err := io.Copy(art, io.LimitReader(expR, artifactMax+1))
		copied <- copyRes{n, err}
	}()

	ex := s.watch(pid, leaf, &measure.Watch{CPUKill: j.compileCPU, Cap: measure.CompileCap}, nil, j.abort)
	ev, evErr := s.leafEvidence(leaf)
	// Every writer of the export pipe is dead now (the leaf is empty and our end is closed), so
	// the copy reaches EOF; the timeout only guards against a runner bug.
	var cr copyRes
	select {
	case cr = <-copied:
	case <-time.After(5 * time.Second):
		pipes.close(expR)
		cr = <-copied
		cr.err = errors.New("the export pipe never reached EOF")
	}
	if evErr != nil {
		return nil, ipc.FailSetup, evErr
	}

	done := &ipc.CompileDone{
		Kill: uint8(ex.kill), CPUus: uint64(ev.CPU.Microseconds()), WallUs: uint64(ex.wall.Microseconds()),
		PeakBytes: uint64(max(ev.PeakBytes, 0)), OOMKills: uint32(ev.OOMKills), PidsMax: uint32(ev.PidsMax),
		Export: uint8(ipc.ExportSkipped),
	}
	if ex.exited {
		done.Exited, done.ExitCode = 1, int32(ex.code)
	} else {
		done.Signal = uint16(ex.signal)
	}
	switch {
	case !ex.exited || ex.kill != measure.KillNone:
	case ex.code == 0 && cr.err == nil && cr.n > 0 && cr.n <= artifactMax:
		if err := art.Chmod(p.ArtifactMode); err != nil {
			return nil, ipc.FailSetup, err
		}
		if err := os.Chmod(sl.artDir, 0o111); err != nil {
			return nil, ipc.FailSetup, err
		}
		done.Export, done.ArtifactBytes = uint8(ipc.ExportOK), uint64(cr.n)
		s.slots[slotIdx].mu.Lock()
		j.compiled = true
		s.slots[slotIdx].mu.Unlock()
	case ex.code == jail.CompileExitMissing || (ex.code == 0 && cr.n == 0):
		done.Export = uint8(ipc.ExportMissing)
	case ex.code == jail.CompileExitTooBig || cr.n > artifactMax:
		done.Export = uint8(ipc.ExportTooBig)
	case ex.code == jail.CompileExitExport || cr.err != nil:
		done.Export = uint8(ipc.ExportFailed)
	}
	return done, 0, nil
}

// ---- CaseRun ----

type caseHooks struct {
	started func(inW, outR, errR, resR *os.File)
}

func (s *Spawner) handleCaseRun(m *ipc.Msg) {
	b := m.Body.(*ipc.CaseRun)
	done, code, err := s.runCase(int(m.Slot), b, caseHooks{started: func(inW, outR, errR, resR *os.File) {
		s.reply(m, ipc.TCaseStarted, &ipc.CaseStarted{}, int(inW.Fd()), int(outR.Fd()), int(errR.Fd()), int(resR.Fd()))
	}})
	if err != nil {
		s.fail(m, code, err)
		return
	}
	s.reply(m, ipc.TCaseDone, done)
}

func (s *Spawner) runCase(slotIdx int, req *ipc.CaseRun, hooks caseHooks) (*ipc.CaseDone, ipc.FailCode, error) {
	var n int
	j, code, err := s.startOp(slotIdx, func(j *job) error {
		if !j.compiled {
			return errors.New("no artifact")
		}
		j.caseN++
		n = j.caseN
		j.kill = make(chan measure.Kill, 1)
		return nil
	})
	if err != nil {
		return nil, code, err
	}
	defer s.endOp(slotIdx, j)
	sl := s.slots[slotIdx]
	p := j.prof
	tl := time.Duration(req.CPUms) * time.Millisecond

	leaf := filepath.Join(j.cg, fmt.Sprintf("case-%d", n))
	if err := cgroup.MkLeaf(leaf, cgroup.Limits{MemoryMax: int64(req.MemMB)<<20 + j.ps.baseline, PidsMax: p.Exec.Pids}); err != nil {
		return nil, ipc.FailSetup, err
	}
	defer func() { _ = s.cg.Remove(leaf) }()

	pipes, err := newPipes(4)
	if err != nil {
		return nil, ipc.FailSetup, err
	}
	defer pipes.closeAll()
	inR, inW := pipes.pair(0)
	outR, outW := pipes.pair(1)
	errR, errW := pipes.pair(2)
	resR, resW := pipes.pair(3)

	rl := []rlimit.RLimit{{Res: unix.RLIMIT_CPU, Rlim: syscall.Rlimit{Cur: measure.RlimitCPUSeconds(tl), Max: measure.RlimitCPUSeconds(tl)}}}
	stack := p.Exec.StackBytes
	if p.Exec.StackFromMemory {
		// Deep recursion is bounded by the case's memory limit, not by an 8 MiB stack (C++,
		// Python; the cgroup still caps the total).
		stack = uint64(req.MemMB)<<20 + uint64(max(j.ps.baseline, 0))
	}
	for _, r := range []struct {
		res int
		v   uint64
	}{{unix.RLIMIT_FSIZE, p.Exec.FSizeBytes}, {unix.RLIMIT_NOFILE, p.Exec.NoFile}, {unix.RLIMIT_STACK, stack}} {
		if r.v > 0 {
			rl = append(rl, rlimit.RLimit{Res: r.res, Rlim: syscall.Rlimit{Cur: r.v, Max: r.v}})
		}
	}
	spec := jail.Spec{
		Argv: p.Exec.Argv, Env: p.Exec.Env, WorkDir: "/w", UID: j.jobUID, RootDir: sl.rootDir,
		Binds:   append([]jail.Bind{{Source: sl.artDir, Target: "/job"}}, toJailBinds(p.Exec.Binds)...),
		WTmpfs:  fmt.Sprintf("size=%dm,nr_inodes=%d,mode=0700", orDefault(p.Exec.WMB, 64), orDefault(p.Exec.WInodes, 4096)),
		Proc:    p.Exec.Proc,
		RLimits: rl,
		Seccomp: j.ps.execProg,
		Files:   []uintptr{s.devnull.Fd(), outW.Fd(), errW.Fd(), inR.Fd(), resW.Fd()},
	}
	st0, _ := readProcStat()
	t0 := time.Now()
	pid, closeCg, err := s.startJail(&spec, leaf)
	if err != nil {
		return nil, ipc.FailSetup, err
	}
	closeCg()
	pipes.closeChildEnds(inR, outW, errW, resW)
	hooks.started(inW, outR, errR, resR)
	pipes.closeFrontEnds(inW, outR, errR, resR)
	cpu0, _ := cgroup.CPUUsage(leaf)

	w := &measure.Watch{CPUKill: measure.CPUKillAt(tl), Wall: measure.WallLimit(tl), Idle: true,
		Cap: time.Duration(req.WallCapMs) * time.Millisecond}
	ex := s.watch(pid, leaf, w, j.kill, j.abort)
	ev, err := s.leafEvidence(leaf)
	if err != nil {
		return nil, ipc.FailSetup, err
	}
	st1, _ := readProcStat()
	done := &ipc.CaseDone{
		Kill: uint8(ex.kill), CPUus: uint64(max(ev.CPU-cpu0, 0).Microseconds()), WallUs: uint64(ex.wall.Microseconds()),
		RusageCPUus: uint64(ex.rusage.Microseconds()), PeakBytes: uint64(max(ev.PeakBytes, 0)), BaselineBytes: uint64(j.ps.baseline),
		OOMKills: uint32(ev.OOMKills), PidsMax: uint32(ev.PidsMax),
		StealPermille: uint16(min(1000, measure.StealFraction(st0, st1)*1000)),
	}
	if ex.exited {
		done.Exited, done.ExitCode = 1, int32(ex.code)
	} else {
		done.Signal = uint16(ex.signal)
	}
	if s.other(slotIdx).busyDuring(t0, time.Now()) {
		done.OtherBusy = 1
	}
	if ex.signal == measure.SIGSYS {
		s.log.Error("SIGSYS in a jail; the front rotates after this job", "slot", slotIdx, "job_seq", j.seq)
	}
	// On-demand canary after a failing time verdict, once no process of the job is alive (the
	// case cgroup is empty now): s2 for the suspect rules.
	timeKill := ex.kill == measure.KillCPU || ex.kill == measure.KillWall || ex.kill == measure.KillCap || ex.signal == measure.SIGXCPU
	if timeKill && ev.OOMKills == 0 {
		if ratio, err := s.canaryRatio(slotIdx); err == nil {
			done.CanaryPermille = uint32(ratio * 1000)
		} else {
			s.log.Error("on-demand canary", "err", err)
		}
	}
	return done, 0, nil
}

// caseKill passes the front's OLE kill to the running case (non-blocking).
func (s *Spawner) caseKill(slotIdx int) {
	sl := s.slots[slotIdx]
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.job != nil && sl.job.kill != nil {
		select {
		case sl.job.kill <- measure.KillOLE:
		default:
		}
	}
}

// ---- JobEnd ----

func (s *Spawner) handleJobEnd(m *ipc.Msg) {
	b := m.Body.(*ipc.JobEnd)
	out, code, err := s.endJob(int(m.Slot), b.Abort == 1)
	if err != nil {
		s.fail(m, code, err)
		return
	}
	s.reply(m, ipc.TJobEnded, &ipc.JobEnded{Outcome: uint16(out)})
}

// endJob tears the job down and asserts the cleanup invariants (t3 §5.4 step 6): cgroup.kill,
// populated 0, rmdir; unmount the artifact and src tmpfs; the slot's memory.current back to
// its baseline ±5 MiB and pids.current 0 — else the slot fails closed and the front rotates.
func (s *Spawner) endJob(slotIdx int, abort bool) (ipc.Teardown, ipc.FailCode, error) {
	sl := s.slots[slotIdx]
	sl.mu.Lock()
	j := sl.job
	if j == nil {
		sl.mu.Unlock()
		return 0, ipc.FailBadRequest, fmt.Errorf("slot %d has no job", slotIdx)
	}
	if abort && !j.aborted {
		j.aborted = true
		close(j.abort)
	}
	sl.mu.Unlock()
	if abort {
		_ = cgroup.Kill(j.cg)
	}
	j.ops.Wait()

	out := ipc.TeardownOK
	if err := s.cg.Wipe(j.cg); err != nil {
		s.log.Error("teardown: job cgroup", "slot", slotIdx, "err", err)
		out = ipc.TeardownCgroup
	}
	for _, d := range []string{sl.artDir, sl.srcDir} {
		if err := unmountAll(d); err != nil {
			s.log.Error("teardown: unmount", "dir", d, "err", err)
			out = ipc.TeardownMount
		}
	}
	if out == ipc.TeardownOK {
		out = s.assertSlotClean(sl)
	}

	sl.mu.Lock()
	sl.job = nil
	sl.activeSince = time.Time{}
	sl.lastEnd = time.Now()
	if out != ipc.TeardownOK {
		sl.closed = true
	}
	sl.mu.Unlock()
	if out != ipc.TeardownOK {
		s.log.Error("teardown assertion failed; the slot failed closed and the runner rotates", "slot", slotIdx, "outcome", out)
		s.requestDrain()
	}
	return out, 0, nil
}

func (s *Spawner) assertSlotClean(sl *slot) ipc.Teardown {
	dir := s.cg.Slot(sl.idx)
	deadline := time.Now().Add(2 * time.Second)
	for {
		pids, perr := cgroup.ReadInt(dir, "pids.current")
		mem, merr := cgroup.ReadInt(dir, "memory.current")
		switch {
		case perr != nil || merr != nil:
			return ipc.TeardownCgroup
		case pids == 0 && mem <= sl.baseline+measure.BaselineTolerance:
			return ipc.TeardownOK
		case time.Now().After(deadline):
			s.log.Error("slot not back to baseline", "slot", sl.idx, "pids", pids, "memory", mem, "baseline", sl.baseline)
			if pids != 0 {
				return ipc.TeardownSurvivors
			}
			return ipc.TeardownMemory
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func unmountAll(d string) error {
	var err error
	for i := 0; i < 8; i++ {
		err = unix.Unmount(d, 0)
		if errors.Is(err, unix.EINVAL) {
			return nil // nothing mounted here any more
		}
		if errors.Is(err, unix.EBUSY) {
			err = unix.Unmount(d, unix.MNT_DETACH)
		}
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("unmount %s: still mounted", d)
}

// ---- jail plumbing ----

// startJail places the jail in its leaf by the configured spawn path. closeCg releases the
// cgroup fd after the start.
func (s *Spawner) startJail(spec *jail.Spec, leaf string) (int, func(), error) {
	noop := func() {}
	if s.cfg.SpawnPath == runner.SpawnCgroupProcs {
		spec.CgroupProcs = filepath.Join(leaf, "cgroup.procs")
		pid, err := jail.Start(*spec)
		return pid, noop, wrapStart(err)
	}
	fd, err := cgroup.OpenDir(leaf)
	if err != nil {
		return 0, noop, err
	}
	spec.CgroupFd = fd
	pid, err := jail.Start(*spec)
	return pid, func() { unix.Close(fd) }, wrapStart(err)
}

func wrapStart(err error) error {
	if err == nil {
		return nil
	}
	if loc, ok := jail.ChildErrorLocation(err); ok {
		return fmt.Errorf("jail start failed at %s: %w", loc, err)
	}
	return fmt.Errorf("jail start: %w", err)
}

// exit is how a watched process ended.
type exit struct {
	exited bool
	code   int
	signal int
	kill   measure.Kill
	wall   time.Duration
	rusage time.Duration
}

// watch applies the kill policy every PollInterval until the jail's init exits, then kills
// the leaf (stragglers) and waits for it to empty.
func (s *Spawner) watch(pid int, leaf string, w *measure.Watch, frontKill <-chan measure.Kill, abort <-chan struct{}) exit {
	start := time.Now()
	type waited struct {
		ws unix.WaitStatus
		ru unix.Rusage
	}
	done := make(chan waited, 1)
	go func() {
		var r waited
		for {
			_, err := unix.Wait4(pid, &r.ws, 0, &r.ru)
			if !errors.Is(err, unix.EINTR) {
				break
			}
		}
		done <- r
	}()
	cpu0, _ := cgroup.CPUUsage(leaf)
	w.Observe(0, 0)
	tick := time.NewTicker(measure.PollInterval)
	defer tick.Stop()
	var ex exit
	var res waited
loop:
	for {
		select {
		case res = <-done:
			break loop
		case k := <-frontKill:
			ex.kill = k
			_ = cgroup.Kill(leaf)
			res = <-done
			break loop
		case <-abort:
			ex.kill = measure.KillAbort
			_ = cgroup.Kill(leaf)
			res = <-done
			break loop
		case <-tick.C:
			cpu, err := cgroup.CPUUsage(leaf)
			if err != nil {
				continue
			}
			if k := w.Observe(time.Since(start), cpu-cpu0); k != measure.KillNone {
				ex.kill = k
				_ = cgroup.Kill(leaf)
				res = <-done
				break loop
			}
		}
	}
	ex.wall = time.Since(start)
	_ = cgroup.Kill(leaf)
	if err := cgroup.WaitEmpty(leaf, 5*time.Second); err != nil {
		s.log.Error("leaf did not empty", "leaf", leaf, "err", err)
	}
	if res.ws.Exited() {
		ex.exited, ex.code = true, res.ws.ExitStatus()
	} else if res.ws.Signaled() {
		ex.signal = int(res.ws.Signal())
	}
	ex.rusage = time.Duration(res.ru.Utime.Nano() + res.ru.Stime.Nano())
	return ex
}

func (s *Spawner) leafEvidence(leaf string) (cgroup.Evidence, error) {
	ev, err := cgroup.ReadEvidence(leaf)
	if err != nil {
		return ev, fmt.Errorf("read evidence from %s: %w", leaf, err)
	}
	return ev, nil
}

func toJailBinds(bs []profile.Bind) []jail.Bind {
	out := make([]jail.Bind, 0, len(bs))
	for _, b := range bs {
		t := b.Target
		if t == "" {
			t = b.Source
		}
		out = append(out, jail.Bind{Source: b.Source, Target: t})
	}
	return out
}

func orDefault(v, def int64) int64 {
	if v > 0 {
		return v
	}
	return def
}

// pipeSet owns a set of pipes and closes whatever is still open.
type pipeSet struct {
	r, w []*os.File
}

func newPipes(n int) (*pipeSet, error) {
	ps := &pipeSet{}
	for i := 0; i < n; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			ps.closeAll()
			return nil, err
		}
		ps.r, ps.w = append(ps.r, r), append(ps.w, w)
	}
	return ps, nil
}

func (ps *pipeSet) pair(i int) (*os.File, *os.File) { return ps.r[i], ps.w[i] }

func (ps *pipeSet) close(fs ...*os.File) {
	for _, f := range fs {
		f.Close()
	}
}

func (ps *pipeSet) closeChildEnds(fs ...*os.File) { ps.close(fs...) }
func (ps *pipeSet) closeFrontEnds(fs ...*os.File) { ps.close(fs...) }

func (ps *pipeSet) closeAll() {
	for i := range ps.r {
		ps.r[i].Close()
		ps.w[i].Close()
	}
}
