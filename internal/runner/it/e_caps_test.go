//go:build linux && runner_it

package it

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// capJob is many cases that each spin just under their TL: together they cross the L14 45 s
// tests cap.
func capJob(t *testing.T) (*runnerapi.Job, [][]byte) {
	job, in := newJob(t, "spin", "testgo@0", repeat("800", 70)...)
	job.Limits.Case.CPUms = 1000
	return job, in
}

// checkCapShape: the cases before the cap are ok, exactly one is the in-flight tle, the rest are
// not_run; TestsCapHit, no Infra.
func checkCapShape(t *testing.T, res *runnerapi.Result) int {
	t.Helper()
	if res.Infra != nil || !res.Telemetry.TestsCapHit {
		t.Fatalf("want TestsCapHit and no Infra: %+v", res.Telemetry)
	}
	ok, tle := 0, -1
	for i, c := range res.Cases {
		switch {
		case tle < 0 && c.Term == runnerapi.TermOK:
			ok++
		case tle < 0 && c.Term == runnerapi.TermTLE:
			tle = i
		case tle >= 0 && c.Term == runnerapi.TermNotRun:
		default:
			t.Errorf("case %d: unexpected %+v (ok so far %d, tle at %d)", i, c, ok, tle)
		}
	}
	if tle < 0 {
		t.Errorf("no in-flight tle")
	}
	return ok
}

// TestTestsCap45s is L14: the whole test phase gets 45 s of wall. When it fires the in-flight
// case is tle, every later case not_run, TestsCapHit=true — TLE, counted, never job_timeout.
func TestTestsCap45s(t *testing.T) {
	ensure(t)
	job, in := capJob(t)
	t0 := time.Now()
	res := run(t, job, in)
	took := time.Since(t0)
	ok := checkCapShape(t, res)
	if res.Throttled {
		t.Errorf("an unsuspect cap is a verdict, not throttled: %+v", res.Telemetry)
	}
	var cpu, wall int64
	for _, c := range res.Cases {
		cpu += c.CPUms
		wall += c.WallMs
	}
	metric(t, "tests_cap_job_s", took.Seconds())
	metric(t, "tests_cap_ok_cases", ok)
	metric(t, "tests_cap_per_case_overhead_ms", float64(45_000-wall)/float64(ok+1))
	if took > 60*time.Second || took < 44*time.Second {
		t.Errorf("the job took %v; the cap is 45 s of tests plus compile", took)
	}
	assertClean(t)
}

// TestTestsCapThrottledWhenOtherSlotBusy: the same cap while the other slot is busy is suspect,
// so the Result is Throttled (judge re-queues) instead of a verdict.
func TestTestsCapThrottledWhenOtherSlotBusy(t *testing.T) {
	ensure(t)
	var wg sync.WaitGroup
	results := make([]*runnerapi.Result, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			job, in := capJob(t)
			results[i] = run(t, job, in)
		}(i)
	}
	wg.Wait()
	for i, res := range results {
		if res == nil {
			continue
		}
		checkCapShape(t, res)
		if !res.Telemetry.OtherSlotBusy || !res.Throttled {
			t.Errorf("job %d: want Throttled with the other slot busy: throttled=%v %+v", i, res.Throttled, res.Telemetry)
		}
	}
	assertClean(t)
}

// TestQuietRerun: a CPU TLE while the other slot was busy is suspect, so the runner takes
// exclusivity (new jobs get 503), waits for the other slot to drain, checks steal and a
// pre-canary, and re-runs the case quietly: the quiet verdict is final (TLE, Quiet) — or, if
// quiet conditions can't be met, Throttled.
func TestQuietRerun(t *testing.T) {
	ensure(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		job, in := longJob(t, 8, 1000)
		run(t, job, in)
	}()
	waitBusy(t, 1)
	job, in := newJob(t, "spin", "testgo@0", "")
	job.Limits.Case.CPUms = 1000
	type out struct {
		r   *response
		err error
	}
	done := make(chan out, 1)
	go func() {
		r, err := post(context.Background(), t, job, in)
		done <- out{r, err}
	}()
	// Once the other job is done and the quiet check runs, a new job is refused (exclusivity).
	wg.Wait()
	time.Sleep(2 * time.Second)
	j3, in3 := newJob(t, "echo", "testgo@0", "x")
	if r, err := post(context.Background(), t, j3, in3); err != nil || r.code != http.StatusServiceUnavailable {
		code := 0
		if r != nil {
			code = r.code
		}
		t.Errorf("a job during the quiet re-run: %d %v", code, err)
	}
	o := <-done
	if o.err != nil || o.r.code != http.StatusOK || o.r.res.Infra != nil {
		t.Fatalf("victim: %v %+v", o.err, o.r)
	}
	res := o.r.res
	c := res.Cases[0]
	t.Logf("quiet re-run: throttled=%v case=%+v telemetry=%+v", res.Throttled, c, res.Telemetry)
	if !res.Telemetry.OtherSlotBusy {
		t.Errorf("the first TLE ran beside a busy slot: %+v", res.Telemetry)
	}
	if res.Throttled {
		// Quiet conditions not met within 60 s: a legitimate outcome, but it proves less.
		t.Logf("quiet conditions were not met: throttled (QuietReruns=%d)", res.Telemetry.QuietReruns)
	} else if res.Telemetry.QuietReruns != 1 || c.Term != runnerapi.TermTLE || !c.Quiet || c.IdleKill {
		t.Errorf("want one quiet re-run ending in a final quiet TLE: %+v %+v", c, res.Telemetry)
	}
	metric(t, "quiet_rerun_outcome", map[bool]string{true: "throttled", false: "quiet_tle"}[res.Throttled])
	assertClean(t)
}
