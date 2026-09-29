package measure

import (
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

func TestParseFlatKeyed(t *testing.T) {
	m, err := ParseFlatKeyed([]byte("usage_usec 123456\nuser_usec 100000\nsystem_usec 23456\nnr_periods 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m["usage_usec"] != 123456 || m["system_usec"] != 23456 {
		t.Errorf("got %v", m)
	}
	ev, err := ParseFlatKeyed([]byte("low 0\nhigh 0\nmax 3\noom 1\noom_kill 1\noom_group_kill 1\n"))
	if err != nil || ev["oom_kill"] != 1 || ev["max"] != 3 {
		t.Errorf("memory.events: %v %v", ev, err)
	}
	if _, err := ParseFlatKeyed([]byte("garbage\n")); err == nil {
		t.Error("want an error for a line without a value")
	}
	if _, err := ParseFlatKeyed([]byte("k notanumber\n")); err == nil {
		t.Error("want an error for a non-numeric value")
	}
}

func TestParseSingle(t *testing.T) {
	for in, want := range map[string]int64{"4096\n": 4096, "max\n": -1, "0": 0} {
		got, err := ParseSingle([]byte(in))
		if err != nil || got != want {
			t.Errorf("ParseSingle(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
}

func TestProcStatSteal(t *testing.T) {
	a, err := ParseProcStat([]byte("cpu  100 0 50 800 0 0 0 50 0 0\ncpu0 1 2 3 4 5 6 7 8 9 10\nintr 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if a.Total != 1000 || a.Steal != 50 {
		t.Fatalf("got %+v", a)
	}
	b, _ := ParseProcStat([]byte("cpu  200 0 100 1500 0 0 0 200 0 0\n"))
	if got := StealFraction(a, b); got < 0.149 || got > 0.151 {
		t.Errorf("steal = %v, want 0.15", got)
	}
	if StealFraction(b, a) != 0 {
		t.Error("a backwards window is 0")
	}
	if _, err := ParseProcStat([]byte("intr 1\n")); err == nil {
		t.Error("want an error without a cpu line")
	}
}

func TestLimitsArithmetic(t *testing.T) {
	cases := []struct {
		tl              time.Duration
		cpuKill, wall   time.Duration
		rlimitCPUSecond uint64
	}{
		{time.Second, 1500 * time.Millisecond, 2 * time.Second, 3},
		{600 * time.Millisecond, 900 * time.Millisecond, 1400 * time.Millisecond, 3},
		{4 * time.Second, 4500 * time.Millisecond, 6500 * time.Millisecond, 6},
	}
	for _, c := range cases {
		if got := CPUKillAt(c.tl); got != c.cpuKill {
			t.Errorf("CPUKillAt(%v) = %v, want %v", c.tl, got, c.cpuKill)
		}
		if got := WallLimit(c.tl); got != c.wall {
			t.Errorf("WallLimit(%v) = %v, want %v", c.tl, got, c.wall)
		}
		if got := RlimitCPUSeconds(c.tl); got != c.rlimitCPUSecond {
			t.Errorf("RlimitCPUSeconds(%v) = %v, want %v", c.tl, got, c.rlimitCPUSecond)
		}
	}
	if JobDeadline != 170*time.Second {
		t.Errorf("job_timeout = %v, want 170 s (15 + 45 + 60 + 45 + 5)", JobDeadline)
	}
}

// run feeds a synthetic CPU curve to a Watch and returns the first kill and when it fired.
func run(w *Watch, cpuAt func(at time.Duration) time.Duration, until time.Duration) (Kill, time.Duration) {
	w.Observe(0, 0)
	for at := PollInterval; at <= until; at += PollInterval {
		if k := w.Observe(at, cpuAt(at)); k != KillNone {
			return k, at
		}
	}
	return KillNone, until
}

func TestWatchKillPolicy(t *testing.T) {
	tl := time.Second
	newWatch := func() *Watch { return &Watch{CPUKill: CPUKillAt(tl), Wall: WallLimit(tl), Idle: true} }

	t.Run("spin → CPU kill just past TL + 0.5 s", func(t *testing.T) {
		k, at := run(newWatch(), func(at time.Duration) time.Duration { return at }, 10*time.Second)
		if k != KillCPU || at < 1500*time.Millisecond || at > 1520*time.Millisecond {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("sleep → idle kill after 1 s", func(t *testing.T) {
		k, at := run(newWatch(), func(time.Duration) time.Duration { return 2 * time.Millisecond }, 10*time.Second)
		if k != KillIdle || at < time.Second || at > 1100*time.Millisecond {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("starved at 30% → wall kill (suspect material)", func(t *testing.T) {
		k, at := run(newWatch(), func(at time.Duration) time.Duration { return at * 3 / 10 }, 10*time.Second)
		if k != KillWall || at < 2*time.Second || at > 2020*time.Millisecond {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("busy then idle → idle", func(t *testing.T) {
		cpu := func(at time.Duration) time.Duration {
			if at < 500*time.Millisecond {
				return at
			}
			return 500 * time.Millisecond
		}
		k, at := run(newWatch(), cpu, 10*time.Second)
		if k != KillIdle || at < 1400*time.Millisecond || at > 1600*time.Millisecond {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("tests cap fires before the case limits", func(t *testing.T) {
		w := newWatch()
		w.Cap = 700 * time.Millisecond
		k, at := run(w, func(at time.Duration) time.Duration { return at }, 10*time.Second)
		if k != KillCap || at != 700*time.Millisecond {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("compile: flat CPU and wall caps, no idle kill", func(t *testing.T) {
		w := &Watch{CPUKill: CompileCap, Cap: CompileCap}
		k, _ := run(w, func(time.Duration) time.Duration { return 0 }, 14*time.Second)
		if k != KillNone {
			t.Errorf("a quiet compile under the cap was killed: %v", k)
		}
		k, at := run(&Watch{CPUKill: CompileCap, Cap: CompileCap}, func(time.Duration) time.Duration { return 0 }, 20*time.Second)
		if k != KillCap || at != CompileCap {
			t.Errorf("got %v at %v", k, at)
		}
	})
	t.Run("a fast program is never killed", func(t *testing.T) {
		k, _ := run(newWatch(), func(at time.Duration) time.Duration { return at }, 900*time.Millisecond)
		if k != KillNone {
			t.Errorf("got %v", k)
		}
	})
}

func TestClassify(t *testing.T) {
	tl := time.Second
	cases := []struct {
		name string
		e    Evidence
		want Outcome
	}{
		{"ok", Evidence{Exited: true, PeakBytes: 3 << 20, Baseline: 1 << 20},
			Outcome{Term: runnerapi.TermOK, PeakKB: 2048}},
		{"exit 3", Evidence{Exited: true, ExitCode: 3},
			Outcome{Term: runnerapi.TermExitNonzero, ExitCode: 3}},
		{"oom beats the SIGKILL it caused", Evidence{Signal: SIGKILL, OOMKills: 1},
			Outcome{Term: runnerapi.TermMLE}},
		{"oom beats a racing CPU kill", Evidence{Kill: KillCPU, Signal: SIGKILL, OOMKills: 2},
			Outcome{Term: runnerapi.TermMLE}},
		{"cpu kill", Evidence{Kill: KillCPU, Signal: SIGKILL, CPU: 1500 * time.Millisecond, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true}},
		{"cpu kill, other slot busy → suspect", Evidence{Kill: KillCPU, Signal: SIGKILL, OtherBusy: true, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true, Suspect: true}},
		{"cpu kill, steal 12% → suspect", Evidence{Kill: KillCPU, Signal: SIGKILL, Steal: 0.12, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true, Suspect: true}},
		{"cpu kill, canary 1.25× → suspect", Evidence{Kill: KillCPU, Signal: SIGKILL, CanaryRate: 1.25, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true, Suspect: true}},
		{"cpu kill, canary 1.1× → not suspect", Evidence{Kill: KillCPU, Signal: SIGKILL, CanaryRate: 1.1, Steal: 0.05, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true}},
		{"wall kill under TL → suspect", Evidence{Kill: KillWall, Signal: SIGKILL, CPU: 600 * time.Millisecond, TL: tl},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true, Suspect: true}},
		{"idle", Evidence{Kill: KillIdle, Signal: SIGKILL, OtherBusy: true},
			Outcome{Term: runnerapi.TermTLE, IdleKill: true}},
		{"tests cap", Evidence{Kill: KillCap, Signal: SIGKILL},
			Outcome{Term: runnerapi.TermTLE, CapHit: true}},
		{"front OLE", Evidence{Kill: KillOLE, Signal: SIGKILL, FrontOLE: true},
			Outcome{Term: runnerapi.TermOLE}},
		{"RLIMIT_CPU backstop", Evidence{Signal: SIGXCPU},
			Outcome{Term: runnerapi.TermTLE, TimeFail: true}},
		{"SIGSYS", Evidence{Signal: SIGSYS},
			Outcome{Term: runnerapi.TermSignal, Signal: "SIGSYS"}},
		{"segv", Evidence{Signal: SIGSEGV},
			Outcome{Term: runnerapi.TermSignal, Signal: "SIGSEGV"}},
		{"thread bomb: fatal exit at the pids cap", Evidence{Exited: true, ExitCode: 2, PidsMax: 4},
			Outcome{Term: runnerapi.TermExitNonzero, ExitCode: 2, ForkLimit: true}},
		{"peak below baseline clamps at 0", Evidence{Exited: true, PeakBytes: 1, Baseline: 1 << 20},
			Outcome{Term: runnerapi.TermOK}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Classify(c.e); got != c.want {
				t.Errorf("got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestSignalName(t *testing.T) {
	if SignalName(31) != "SIGSYS" || SignalName(9) != "SIGKILL" || SignalName(40) != "SIG40" {
		t.Error("signal names")
	}
}

func TestRollingMedian(t *testing.T) {
	r := Rolling{N: 3}
	if r.Median() != 0 {
		t.Error("empty median is 0")
	}
	for _, v := range []int64{10, 30, 20, 1000} {
		r.Add(v)
	}
	// Window keeps 30, 20, 1000 → 30.
	if r.Median() != 30 || r.Len() != 3 {
		t.Errorf("median %d len %d", r.Median(), r.Len())
	}
	if Median([]int64{4, 1, 3, 2}) != 2 {
		t.Error("even median")
	}
}
