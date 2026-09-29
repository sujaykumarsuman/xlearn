package measure

import "time"

// Every timing and throttling threshold lives in this one file (plan task 6), so mi-10 can
// retune them on production as a runner patch.
//
// Sources: t3 §5.7 (throttled, the quiet re-run), §7.2 (time-limit policy), L14 (ADR-0035 §4).
// The steal thresholds come from the S0 sample (t3 §15: `vmstat` steal p50 3%, p95 8%, max 20%
// over 1,305 minutes, 2026-09-24/25) and sar's 9-day p95 of 5.2% (docs/v2/status.md "Capacity
// reads"): 10% sits above the normal p95, 5% is the quiet bar. Cgroup CPU time includes steal
// (PARAVIRT_TIME_ACCOUNTING is off, t3 §15), which is why s1 and s2 exist at all.
const (
	// PollInterval is the spawner's kill-policy poll (t3 §5.4 step 5).
	PollInterval = 10 * time.Millisecond

	// CPUKillMarginMax caps the CPU kill margin: kill at TL + min(0.5 s, TL/2).
	CPUKillMarginMax = 500 * time.Millisecond
	// WallFactor and WallSlack give the wall limit 1.5·TL + 0.5 s.
	WallFactor = 1.5
	WallSlack  = 500 * time.Millisecond
	// RlimitCPUSlack puts the RLIMIT_CPU backstop at TL + 2 s.
	RlimitCPUSlack = 2 * time.Second
	// IdleRate and IdleWindow: a CPU rate under 5% for ≥ 1 s is TLE (idle).
	IdleRate   = 0.05
	IdleWindow = time.Second

	// StealSuspect (s1) and CanarySuspect (s2): a failing time verdict is suspect when steal over
	// the case window is ≥ 10%, or the on-demand canary is ≥ 1.2 × its median, or the other slot
	// was busy.
	StealSuspect  = 0.10
	CanarySuspect = 1.20
	// QuietSteal, QuietStealWindow and QuietCanary are the quiet re-run's entry bar: steal < 5%
	// over 10 s and a pre-canary ≤ 1.1 × median.
	QuietSteal       = 0.05
	QuietStealWindow = 10 * time.Second
	QuietCanary      = 1.10
	// QuietWait bounds the wait for quiet conditions (the other slot draining included).
	QuietWait = 60 * time.Second
	// CanaryWindow is how many canary runs the rolling median keeps (1 h at the 5 min period).
	CanaryWindow = 12

	// CompileCap is L14's compile cap (CPU and wall). A hit is CE.
	CompileCap = 15 * time.Second
	// TestsCap is L14's cap on the whole test phase (every case, from the first case start;
	// compile and the quiet re-run excluded). The quiet re-run gets its own TestsCap.
	TestsCap = 45 * time.Second
	// JobSlack and JobDeadline: job_timeout = compile 15 s + tests 45 s + the quiet re-run
	// allowance (≤ 60 s wait + ≤ 45 s re-run) + 5 s slack = 170 s, a runner-fault backstop only.
	JobSlack    = 5 * time.Second
	JobDeadline = CompileCap + TestsCap + QuietWait + TestsCap + JobSlack

	// BaselineTolerance is the teardown assertion: a slot's memory.current back to its
	// baseline ±5 MiB (t3 §5.10).
	BaselineTolerance = 5 << 20
)
