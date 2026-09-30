//go:build linux && runner_it

package it

import (
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// TestThousandCaseCgroups is the cleanup release gate (t3 §5.10): after 1,000 case cgroups there
// are no leftover cgroups, mounts or processes; runner/ memory is back to its baseline ±5 MiB;
// nr_dying_descendants settles near 0 within 60 s. It also measures the per-case jail overhead
// (spawn + teardown) that the L14 arithmetic must absorb: 45 s cap − Σ TL ≤ 40 CPU-s = 5 s for
// ≤ 512 cases.
func TestThousandCaseCgroups(t *testing.T) {
	ensure(t)
	// Warm-up at full size, so the front's heap has seen a 500-case Result before the baseline.
	job, in := newJob(t, "noop", "testgo@0", repeat("", 500)...)
	t0 := time.Now()
	res := run(t, job, in)
	elapsed := time.Since(t0)
	var sumWall int64
	for _, c := range res.Cases {
		if c.Term != runnerapi.TermOK {
			t.Fatalf("noop: %+v", c)
		}
		sumWall += c.WallMs
	}
	perCase := float64(elapsed.Milliseconds()) / 500
	metric(t, "per_case_total_ms", perCase)
	metric(t, "per_case_jail_overhead_ms", (float64(elapsed.Milliseconds())-float64(sumWall))/500)
	metric(t, "per_case_x512_s", perCase*512/1000)
	assertClean(t)
	settleDying(t)
	base := getJSON[runnerapi.StatsResponse](t, "/v1/stats")

	for j := 0; j < 2; j++ {
		job, in := newJob(t, "noop", "testgo@0", repeat("", 500)...)
		for i, c := range run(t, job, in).Cases {
			if c.Term != runnerapi.TermOK {
				t.Fatalf("job %d case %d: %+v", j, i, c)
			}
		}
	}
	assertClean(t)
	took := settleDying(t)
	metric(t, "nr_dying_settle_ms", took.Milliseconds())
	after := getJSON[runnerapi.StatsResponse](t, "/v1/stats")
	drift := after.RunnerMemoryCurrent - base.RunnerMemoryCurrent
	metric(t, "runner_memory_drift_kb", drift/1024)
	if drift > 5<<20 || drift < -(5<<20) {
		t.Errorf("runner/ memory drifted %d KiB over 1,000 case cgroups (±5 MiB allowed)", drift/1024)
	}
	for i, s := range after.Slots {
		if s.PidsCurrent != 0 {
			t.Errorf("slot %d pids.current %d", i, s.PidsCurrent)
		}
		metric(t, "slot_memory_current_kb_s"+itoa(i), s.MemoryCurrent/1024)
	}
}

// settleDying waits for both slots' nr_dying_descendants to reach ~0 (≤ 2) within 60 s.
func settleDying(t *testing.T) time.Duration {
	t.Helper()
	t0 := time.Now()
	for {
		st := getJSON[runnerapi.StatsResponse](t, "/v1/stats")
		if st.Slots[0].NrDyingDescendants <= 2 && st.Slots[1].NrDyingDescendants <= 2 {
			return time.Since(t0)
		}
		if time.Since(t0) > 60*time.Second {
			t.Fatalf("nr_dying_descendants still %d / %d after 60 s", st.Slots[0].NrDyingDescendants, st.Slots[1].NrDyingDescendants)
		}
		time.Sleep(250 * time.Millisecond)
	}
}
