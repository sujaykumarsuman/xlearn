// Package measure turns evidence read outside the learner's process into verdict inputs
// (t3 §5.5, §7.2): cgroup file parsing, steal, the kill policy and the case classification.
// Everything here is a pure function, table-tested on every platform; the spawner does the
// reading and the killing.
package measure

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// ParseFlatKeyed parses a cgroup "flat keyed" file ("key value" per line: cpu.stat,
// memory.events, pids.events, cgroup.stat, cgroup.events).
func ParseFlatKeyed(b []byte) (map[string]int64, error) {
	out := map[string]int64{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("flat-keyed line %q", line)
		}
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("flat-keyed %q: %w", k, err)
		}
		out[k] = n
	}
	return out, sc.Err()
}

// ParseSingle parses a single-value cgroup file (memory.peak, memory.current, pids.current).
// "max" parses as -1.
func ParseSingle(b []byte) (int64, error) {
	s := strings.TrimSpace(string(b))
	if s == "max" {
		return -1, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// CPUTimes is the aggregate "cpu" line of /proc/stat, in USER_HZ ticks.
type CPUTimes struct {
	Total uint64
	Steal uint64
}

// ParseProcStat reads the aggregate cpu line: user nice system idle iowait irq softirq steal
// guest guest_nice. guest and guest_nice are already inside user/nice, so Total excludes them.
func ParseProcStat(b []byte) (CPUTimes, error) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 || f[0] != "cpu" {
			continue
		}
		if len(f) < 9 {
			return CPUTimes{}, fmt.Errorf("/proc/stat cpu line has %d fields", len(f))
		}
		var t CPUTimes
		for i := 1; i <= 8; i++ {
			v, err := strconv.ParseUint(f[i], 10, 64)
			if err != nil {
				return CPUTimes{}, fmt.Errorf("/proc/stat field %d: %w", i, err)
			}
			t.Total += v
			if i == 8 {
				t.Steal = v
			}
		}
		return t, nil
	}
	return CPUTimes{}, fmt.Errorf("/proc/stat: no aggregate cpu line")
}

// StealFraction is the steal share of all CPU time between two samples (s1). Zero when no
// time passed.
func StealFraction(a, b CPUTimes) float64 {
	if b.Total <= a.Total || b.Steal < a.Steal {
		return 0
	}
	return float64(b.Steal-a.Steal) / float64(b.Total-a.Total)
}

// CPUKillAt is the CPU kill point: TL + min(0.5 s, TL/2).
func CPUKillAt(tl time.Duration) time.Duration {
	return tl + min(CPUKillMarginMax, tl/2)
}

// WallLimit is 1.5·TL + 0.5 s.
func WallLimit(tl time.Duration) time.Duration {
	return time.Duration(float64(tl)*WallFactor) + WallSlack
}

// RlimitCPUSeconds is the RLIMIT_CPU backstop, TL + 2 s, rounded up to whole seconds.
func RlimitCPUSeconds(tl time.Duration) uint64 {
	d := tl + RlimitCPUSlack
	return uint64((d + time.Second - 1) / time.Second)
}

// Kill is why the spawner killed a process (or KillNone).
type Kill uint8

// Kill reasons. The numeric values travel over the spawner⇄front IPC.
const (
	KillNone  Kill = 0
	KillCPU   Kill = 1 // CPU > TL + margin
	KillWall  Kill = 2 // wall > 1.5·TL + 0.5 s with a CPU rate ≥ 5%
	KillIdle  Kill = 3 // CPU rate < 5% for ≥ 1 s
	KillCap   Kill = 4 // the L14 tests cap (or the compile cap) ran out
	KillOLE   Kill = 5 // the front saw an fd pass its cap
	KillAbort Kill = 6 // the job was aborted (client gone, drain)
	KillMem   Kill = 7 // compile only: memory (reported by the cgroup, not a kill we issue)
)

// Valid reports whether k is a known reason.
func (k Kill) Valid() bool { return k <= KillMem }

// Watch applies the kill policy to CPU samples taken every PollInterval. It keeps the last
// IdleWindow of samples to compute the CPU rate.
type Watch struct {
	// CPUKill is the CPU kill point (0 = no CPU kill; the compile uses a flat cap).
	CPUKill time.Duration
	// Wall is the wall limit (0 = none).
	Wall time.Duration
	// Cap is the remaining tests budget for this case, or the compile cap (0 = none).
	Cap time.Duration
	// Idle enables the idle kill.
	Idle bool

	samples []sample
}

type sample struct {
	at, cpu time.Duration
}

// Observe records a sample (wall since the process started, CPU consumed so far) and returns
// the kill decision.
func (w *Watch) Observe(at, cpu time.Duration) Kill {
	w.samples = append(w.samples, sample{at, cpu})
	// Drop samples older than the idle window, keeping one at or before its start.
	cut := 0
	for i := range w.samples {
		if w.samples[i].at <= at-IdleWindow {
			cut = i
		}
	}
	w.samples = w.samples[cut:]

	if w.CPUKill > 0 && cpu > w.CPUKill {
		return KillCPU
	}
	if w.Idle && at >= IdleWindow && w.rate(at, cpu) < IdleRate {
		return KillIdle
	}
	if w.Wall > 0 && at >= w.Wall {
		if w.rate(at, cpu) < IdleRate {
			return KillIdle
		}
		return KillWall
	}
	if w.Cap > 0 && at >= w.Cap {
		return KillCap
	}
	return KillNone
}

// rate is the CPU rate over the last IdleWindow (1.0 = one full CPU).
func (w *Watch) rate(at, cpu time.Duration) float64 {
	if len(w.samples) == 0 {
		return 1
	}
	first := w.samples[0]
	span := at - first.at
	if span < IdleWindow {
		// Not enough history: treat the process as busy.
		return 1
	}
	return float64(cpu-first.cpu) / float64(span)
}

// Evidence is everything the spawner and front know about one finished case.
type Evidence struct {
	Kill       Kill
	Exited     bool // the process exited normally (ExitCode valid)
	ExitCode   int
	Signal     int // the terminating signal (Linux numbering), 0 if none
	OOMKills   int64
	PidsMax    int64
	CPU        time.Duration
	Wall       time.Duration
	PeakBytes  int64
	Baseline   int64 // the profile's calibrated baseline, bytes
	TL         time.Duration
	FrontOLE   bool // the front saw an fd pass its cap (it also asked for KillOLE)
	OtherBusy  bool
	Steal      float64
	CanaryRate float64 // on-demand canary ÷ median; 0 = not run
}

// Outcome is the classified case.
type Outcome struct {
	Term      runnerapi.Term
	Signal    string
	ExitCode  int
	IdleKill  bool
	ForkLimit bool
	// TimeFail marks a failing time verdict (CPU or wall kill): the kind t3 §5.7 re-runs when
	// it's suspect. Idle kills and the tests cap are not re-run.
	TimeFail bool
	// CapHit marks the L14 tests cap.
	CapHit bool
	// Suspect marks a TimeFail under s1, s2 or a busy other slot (or a non-idle wall kill
	// under TL): the job needs the quiet re-run.
	Suspect bool
	PeakKB  int64
}

// Linux signal numbers (identical on amd64 and arm64).
const (
	SIGKILL = 9
	SIGSEGV = 11
	SIGXCPU = 24
	SIGSYS  = 31
)

var signalNames = map[int]string{
	1: "SIGHUP", 2: "SIGINT", 3: "SIGQUIT", 4: "SIGILL", 5: "SIGTRAP", 6: "SIGABRT", 7: "SIGBUS",
	8: "SIGFPE", 9: "SIGKILL", 10: "SIGUSR1", 11: "SIGSEGV", 12: "SIGUSR2", 13: "SIGPIPE",
	14: "SIGALRM", 15: "SIGTERM", 16: "SIGSTKFLT", 17: "SIGCHLD", 18: "SIGCONT", 19: "SIGSTOP",
	20: "SIGTSTP", 21: "SIGTTIN", 22: "SIGTTOU", 23: "SIGURG", 24: "SIGXCPU", 25: "SIGXFSZ",
	26: "SIGVTALRM", 27: "SIGPROF", 28: "SIGWINCH", 29: "SIGIO", 30: "SIGPWR", 31: "SIGSYS",
}

// SignalName is the Linux name of sig ("SIG<n>" for real-time signals).
func SignalName(sig int) string {
	if n, ok := signalNames[sig]; ok {
		return n
	}
	return "SIG" + strconv.Itoa(sig)
}

// Classify maps evidence to a case outcome. Precedence: the kernel's OOM kill (MLE), then our
// own kills (CPU/wall/idle/cap → TLE, OLE), then RLIMIT_CPU's SIGXCPU (TLE), then any other
// signal (SIGSYS → signal, which judge maps to RE, counted), then the exit code.
func Classify(e Evidence) Outcome {
	o := Outcome{ForkLimit: e.PidsMax > 0, PeakKB: max(0, e.PeakBytes-e.Baseline) / 1024}
	switch {
	case e.OOMKills > 0:
		o.Term = runnerapi.TermMLE
	case e.Kill == KillCPU:
		o.Term, o.TimeFail = runnerapi.TermTLE, true
	case e.Kill == KillIdle:
		o.Term, o.IdleKill = runnerapi.TermTLE, true
	case e.Kill == KillWall:
		o.Term, o.TimeFail = runnerapi.TermTLE, true
	case e.Kill == KillCap:
		o.Term, o.CapHit = runnerapi.TermTLE, true
	case e.Kill == KillOLE || e.FrontOLE:
		o.Term = runnerapi.TermOLE
	case e.Signal == SIGXCPU:
		o.Term, o.TimeFail = runnerapi.TermTLE, true
	case e.Signal != 0:
		o.Term, o.Signal = runnerapi.TermSignal, SignalName(e.Signal)
	case e.Exited && e.ExitCode != 0:
		o.Term, o.ExitCode = runnerapi.TermExitNonzero, e.ExitCode
	default:
		o.Term = runnerapi.TermOK
	}
	if o.TimeFail {
		o.Suspect = e.OtherBusy || e.Steal >= StealSuspect || (e.CanaryRate > 0 && e.CanaryRate >= CanarySuspect) ||
			(e.Kill == KillWall && e.CPU < e.TL)
	}
	return o
}

// Median returns the median of xs (0 for none). It does not modify xs.
func Median(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]int64(nil), xs...)
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}

// Rolling is a fixed-size window of canary results with a median.
type Rolling struct {
	N    int
	vals []int64
}

// Add appends v, dropping the oldest past N.
func (r *Rolling) Add(v int64) {
	r.vals = append(r.vals, v)
	if n := r.N; n > 0 && len(r.vals) > n {
		r.vals = r.vals[len(r.vals)-n:]
	}
}

// Median is the window's median.
func (r *Rolling) Median() int64 { return Median(r.vals) }

// Len is the number of values held.
func (r *Rolling) Len() int { return len(r.vals) }
