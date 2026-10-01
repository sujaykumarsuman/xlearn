//go:build linux

package spawner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner/cgroup"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/jail"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
	"github.com/sujaykumarsuman/xlearn/internal/runner/seccomp"
)

// profState is what startup computed for one profile.
type profState struct {
	sha         string
	toolchain   string
	baseline    int64
	execProg    *syscall.SockFprog
	compileProg *syscall.SockFprog
}

// Baseline measurement: the no-op program runs this many times; the median memory.peak is the
// profile's baseline (PeakKB subtracts it; mem_mb is enforced on top of it).
const baselineRuns = 5

// loadProfiles assembles each profile's filters, detects its toolchain version, and pre-reads
// and hashes its toolchain (profile.TreeHash), so the toolchain's page cache is charged to
// runner/ (not to a case cgroup that would then linger as a dying memcg) and ProfileSHA
// (profile.Fingerprint: the description incl. lists and harness templates, the version, the
// trees incl. the Go GOCACHE seed) covers it.
func (s *Spawner) loadProfiles() error {
	s.profiles = profile.All()
	s.pstate = make([]*profState, len(s.profiles))
	trees := map[string]string{}
	for i, p := range s.profiles {
		ps := &profState{toolchain: p.Toolchain}
		var err error
		if ps.execProg, err = s.assemble(p.Name+" exec", p.Exec.Seccomp.Policy()); err != nil {
			return err
		}
		if ps.compileProg, err = s.assemble(p.Name+" compile", p.Compile.Seccomp.Policy()); err != nil {
			return err
		}
		if p.Detect != nil {
			v, err := p.Detect()
			if err != nil {
				return fmt.Errorf("profile %s: detect the toolchain: %w", p.Name, err)
			}
			if v != "" {
				ps.toolchain = v
			}
		}
		ps.sha, err = profile.Fingerprint(p, ps.toolchain, func(path string) (string, error) {
			if th, ok := trees[path]; ok {
				return th, nil
			}
			t0 := time.Now()
			th, err := profile.TreeHash(path)
			if err != nil {
				return "", fmt.Errorf("profile %s: pre-read %s: %w", p.Name, path, err)
			}
			trees[path] = th
			s.log.Info("toolchain pre-read", "profile", p.Name, "path", path, "took_ms", time.Since(t0).Milliseconds())
			return th, nil
		})
		if err != nil {
			return err
		}
		s.log.Info("profile loaded", "profile", p.Name, "toolchain", ps.toolchain, "profile_sha256", ps.sha)
		s.pstate[i] = ps
	}
	return nil
}

// assemble builds one filter. Every name must resolve on this arch: amd64 runs t3 §16.2's
// lists, an arm64 dev VM its own lists (seccomp_arm64.go); nothing is skipped.
func (s *Spawner) assemble(what string, pol seccomp.Policy) (*syscall.SockFprog, error) {
	prog, unresolved, err := seccomp.Assemble(pol, seccomp.Native())
	if err != nil {
		return nil, fmt.Errorf("%s filter: %w", what, err)
	}
	if len(unresolved) > 0 {
		return nil, fmt.Errorf("%s filter: unknown syscall names on %s: %s", what, runtime.GOARCH, strings.Join(unresolved, " "))
	}
	return seccomp.SockFprog(prog), nil
}

// measureBaselines compiles each profile's no-op program and runs it baselineRuns times in
// slot 0 (t3 §5.4 step 7). This also proves the whole jail path before the runner serves.
func (s *Spawner) measureBaselines() error {
	for _, sl := range s.slots {
		b, err := cgroup.ReadInt(s.cg.Slot(sl.idx), "memory.current")
		if err != nil {
			return err
		}
		sl.baseline = b
	}
	for i, p := range s.profiles {
		t0 := time.Now()
		b, err := s.baselineOf(i, p)
		if err != nil {
			return fmt.Errorf("profile %s: %w", p.Name, err)
		}
		s.pstate[i].baseline = b
		s.log.Info("profile baseline", "profile", p.Name, "baseline_kb", b/1024, "took_ms", time.Since(t0).Milliseconds())
	}
	return nil
}

func (s *Spawner) baselineOf(pidx int, p *profile.Profile) (int64, error) {
	mem := orDefault(p.Compile.MemMB, 768) << 20
	fd, _, err := s.beginJob(0, pidx, 0, measure.CompileCap, mem, 0)
	if err != nil {
		return 0, err
	}
	endOnce := func() error {
		out, _, err := s.endJob(0, false)
		if err != nil {
			return err
		}
		if out != ipc.TeardownOK {
			return fmt.Errorf("baseline teardown: outcome %d", out)
		}
		return nil
	}
	if err := writeFiles(fd, p.BaselineFiles); err != nil {
		unix.Close(fd)
		_ = endOnce()
		return 0, err
	}
	unix.Close(fd)
	var stderr bytes.Buffer
	done, _, err := s.compile(0, compileHooks{started: func(outR, errR *os.File) {
		drain(outR, io.Discard)
		drain(errR, &limited{w: &stderr, n: 16 << 10})
	}})
	if err != nil {
		_ = endOnce()
		return 0, err
	}
	if done.Export != uint8(ipc.ExportOK) {
		_ = endOnce()
		return 0, fmt.Errorf("baseline compile failed (exit %d, kill %d, export %d): %s", done.ExitCode, done.Kill, done.Export, stderr.String())
	}
	var peaks []int64
	for k := 0; k < baselineRuns; k++ {
		cd, _, err := s.runCase(0, &ipc.CaseRun{CaseIdx: uint32(k), CPUms: 2000, MemMB: 256, WallCapMs: 10000},
			caseHooks{started: func(inW, outR, errR, resR *os.File) {
				drain(outR, io.Discard)
				drain(errR, io.Discard)
				drain(resR, io.Discard)
			}})
		if err != nil {
			_ = endOnce()
			return 0, err
		}
		if cd.Exited != 1 || cd.ExitCode != 0 || cd.Kill != 0 {
			_ = endOnce()
			return 0, fmt.Errorf("baseline run %d: exited=%d code=%d signal=%d kill=%d", k, cd.Exited, cd.ExitCode, cd.Signal, cd.Kill)
		}
		peaks = append(peaks, int64(cd.PeakBytes))
	}
	if err := endOnce(); err != nil {
		return 0, err
	}
	return measure.Median(peaks), nil
}

// writeFiles writes trusted startup files (the profile's own no-op program) through a dir fd.
func writeFiles(dirfd int, files []runnerapi.File) error {
	for _, f := range files {
		fd, err := unix.Openat(dirfd, f.Path, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o444)
		if err != nil {
			return err
		}
		_, werr := unix.Write(fd, f.Data)
		unix.Close(fd)
		if werr != nil {
			return werr
		}
	}
	return unix.Fchmod(dirfd, 0o555)
}

// drain copies a dup of f to w in the background (the caller closes f right after the hook).
func drain(f *os.File, w io.Writer) {
	fd, err := unix.FcntlInt(f.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return
	}
	d := os.NewFile(uintptr(fd), f.Name())
	go func() {
		_, _ = io.Copy(w, d)
		d.Close()
	}()
}

type limited struct {
	w io.Writer
	n int
}

func (l *limited) Write(p []byte) (int, error) {
	if l.n > 0 {
		k := min(len(p), l.n)
		_, _ = l.w.Write(p[:k])
		l.n -= k
	}
	return len(p), nil
}

// ---- canary ----

func (s *Spawner) canaryMedian() time.Duration {
	s.medMu.Lock()
	defer s.medMu.Unlock()
	return time.Duration(s.median.Median()) * time.Microsecond
}

func (s *Spawner) seedCanary() error {
	for i := 0; i < 3; i++ {
		d, err := s.runCanary(0)
		if err != nil {
			return err
		}
		s.addCanary(d)
	}
	return nil
}

func (s *Spawner) addCanary(d time.Duration) {
	s.medMu.Lock()
	s.median.Add(d.Microseconds())
	s.medMu.Unlock()
}

// canaryRatio runs the canary in the slot and returns its CPU time over the median (s2).
func (s *Spawner) canaryRatio(slotIdx int) (float64, error) {
	d, err := s.runCanary(slotIdx)
	if err != nil {
		return 0, err
	}
	med := s.canaryMedian()
	if med <= 0 {
		return 0, fmt.Errorf("no canary median")
	}
	return float64(d) / float64(med), nil
}

// runCanary runs the canary twice back to back and returns the faster run. A CPU that just sat
// idle (the quiet check's 10 s steal window, the 5-minute idle loop) runs its first pass ~30%
// slow while its clock ramps (measured on the dev VM); the second pass is what the median
// compares against.
func (s *Spawner) runCanary(slotIdx int) (time.Duration, error) {
	s.canaryMu.Lock()
	defer s.canaryMu.Unlock()
	a, err := s.canaryOnce(slotIdx)
	if err != nil {
		return 0, err
	}
	b, err := s.canaryOnce(slotIdx)
	if err != nil {
		return 0, err
	}
	return min(a, b), nil
}

// canaryOnce runs `runner canary` capless in <slot>/canary and returns its cgroup CPU time.
func (s *Spawner) canaryOnce(slotIdx int) (time.Duration, error) {
	leaf := filepath.Join(s.cg.Slot(slotIdx), "canary")
	_ = s.cg.Wipe(leaf)
	if err := cgroup.MkLeaf(leaf, cgroup.Limits{MemoryMax: 256 << 20, PidsMax: 16}); err != nil {
		return 0, err
	}
	defer func() { _ = s.cg.Remove(leaf) }()
	fd, err := cgroup.OpenDir(leaf)
	if err != nil {
		return 0, err
	}
	pid, err := jail.StartProcess(jail.Process{
		Argv: []string{s.exe, "canary"}, Env: []string{"GOMAXPROCS=1", "GOGC=off"},
		Files: []uintptr{s.devnull.Fd(), s.devnull.Fd(), s.devnull.Fd()}, UID: canaryUID, CgroupFd: fd,
	})
	unix.Close(fd)
	if err != nil {
		return 0, err
	}
	if code := waitPid(pid); code != 0 {
		return 0, fmt.Errorf("canary exited %d", code)
	}
	return cgroup.CPUUsage(leaf)
}

// idleCanaryLoop runs the canary every CanaryEvery while both slots are idle.
func (s *Spawner) idleCanaryLoop(ctx context.Context) {
	t := time.NewTicker(s.cfg.CanaryEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			idle := true
			for _, sl := range s.slots {
				sl.mu.Lock()
				if sl.job != nil {
					idle = false
				}
				sl.mu.Unlock()
			}
			if !idle {
				continue
			}
			if d, err := s.runCanary(0); err == nil {
				s.addCanary(d)
			} else {
				s.log.Error("idle canary", "err", err)
			}
		}
	}
}

// ---- quiet check, stats, probes ----

func readProcStat() (measure.CPUTimes, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return measure.CPUTimes{}, err
	}
	return measure.ParseProcStat(b)
}

// handleQuietCheck measures steal over QuietStealWindow, then runs a pre-canary (t3 §5.7).
func (s *Spawner) handleQuietCheck(m *ipc.Msg) {
	st0, err := readProcStat()
	if err != nil {
		s.fail(m, ipc.FailSetup, err)
		return
	}
	time.Sleep(measure.QuietStealWindow)
	st1, err := readProcStat()
	if err != nil {
		s.fail(m, ipc.FailSetup, err)
		return
	}
	ratio, err := s.canaryRatio(int(m.Slot))
	if err != nil {
		s.fail(m, ipc.FailSetup, err)
		return
	}
	s.reply(m, ipc.TQuietStatus, &ipc.QuietStatus{
		StealPermille:  uint16(min(1000, measure.StealFraction(st0, st1)*1000)),
		CanaryPermille: uint32(ratio * 1000),
	})
}

func (s *Spawner) handleStats(m *ipc.Msg) {
	var r ipc.StatsReply
	for i := range s.slots {
		dir := s.cg.Slot(i)
		mem, _ := cgroup.ReadInt(dir, "memory.current")
		pids, _ := cgroup.ReadInt(dir, "pids.current")
		st, _ := cgroup.ReadKeyed(dir, "cgroup.stat")
		r.MemoryCurrent[i], r.PidsCurrent[i], r.NrDying[i] = uint64(max(mem, 0)), uint32(max(pids, 0)), uint32(max(st["nr_dying_descendants"], 0))
	}
	rm, _ := cgroup.ReadInt(s.cg.Runner(), "memory.current")
	r.RunnerMemory = uint64(max(rm, 0))
	if ev, err := cgroup.ReadKeyed(s.cg.Root, "memory.events"); err == nil {
		r.OOMKills = uint64(max(ev["oom_kill"], 0))
	}
	if ev, err := cgroup.ReadKeyed(s.cg.Root, "memory.events.local"); err == nil {
		r.OOMKillsLocal = uint64(max(ev["oom_kill"], 0))
	}
	r.CanaryMedianUs = uint64(s.canaryMedian().Microseconds())
	s.reply(m, ipc.TStatsReply, &r)
}

// handleProbe runs the privileged readiness canaries (t3 §5.3 /readyz) from the spawner, the
// process that holds capabilities: every "denied" must hold even for it.
func (s *Spawner) handleProbe(m *ipc.Msg) {
	var r ipc.ProbeReply
	check := func(bit uint32, pass bool) {
		r.Ran |= bit
		if !pass {
			r.Failed |= bit
		}
	}
	check(ipc.ProbeAppArmor, strings.HasPrefix(apparmorLabel(), "xlearn-runner"))
	check(ipc.ProbeUIDMap, !identityUIDMap())
	check(ipc.ProbeCgroupfs, s.cg.Probe() == nil)
	check(ipc.ProbeUserNS, jail.ProbeUserNS(s.exe, s.devnull.Fd()))
	fsfd, err := unix.Fsopen("tmpfs", unix.FSOPEN_CLOEXEC)
	if err == nil {
		unix.Close(fsfd)
	}
	check(ipc.ProbeFsopen, err != nil)
	sfd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_SCTP)
	if err == nil {
		unix.Close(sfd)
	}
	check(ipc.ProbeSCTP, err != nil)
	cp, err := os.ReadFile("/proc/sys/kernel/core_pattern")
	check(ipc.ProbeCorePattern, err == nil && !strings.HasPrefix(string(cp), "|"))
	s.reply(m, ipc.TProbeReply, &r)
}

func apparmorLabel() string {
	for _, p := range []string{"/proc/self/attr/apparmor/current", "/proc/self/attr/current"} {
		if b, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// identityUIDMap reports whether /proc/self/uid_map is the host identity map (no user namespace).
func identityUIDMap() bool {
	b, err := os.ReadFile("/proc/self/uid_map")
	if err != nil {
		return true
	}
	f := strings.Fields(string(b))
	return len(f) >= 3 && f[0] == "0" && f[1] == "0" && f[2] == "4294967295"
}
