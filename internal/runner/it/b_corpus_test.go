//go:build linux && runner_it

package it

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// TestBalloonMLE100 is the INV-14 gate (t3 §9 P2): 100 one-GiB balloons against a 256 MiB case
// limit are MLE 100/100, and every OOM lands in a case cgroup — the container's own
// (non-hierarchical) oom_kill count doesn't move.
func TestBalloonMLE100(t *testing.T) {
	ensure(t)
	local0 := containerOOMs()
	hier0 := readKeyed(filepath.Join(cgRoot, "memory.events"))["oom_kill"]
	mle := 0
	for j := 0; j < 4; j++ {
		job, in := newJob(t, "balloon", "testgo@0", repeat("", 25)...)
		res := run(t, job, in)
		for i, c := range res.Cases {
			if c.Term == runnerapi.TermMLE {
				mle++
			} else {
				t.Errorf("job %d case %d: %+v", j, i, c)
			}
		}
		assertClean(t)
	}
	local1 := containerOOMs()
	hier1 := readKeyed(filepath.Join(cgRoot, "memory.events"))["oom_kill"]
	metric(t, "balloon_mle", fmt.Sprintf("%d/100", mle))
	metric(t, "container_oom_kill_local_delta", local1-local0)
	metric(t, "container_oom_kill_hierarchical_delta", hier1-hier0)
	if mle != 100 {
		t.Errorf("MLE %d/100", mle)
	}
	if local1 != local0 {
		t.Errorf("container-level OOM kills: %d → %d (INV-14 broken)", local0, local1)
	}
	st := getJSON[runnerapi.StatsResponse](t, "/v1/stats")
	if st.ContainerOOMKillsLocal != local1 {
		t.Errorf("/v1/stats container_oom_kill_local %d, the cgroup says %d", st.ContainerOOMKillsLocal, local1)
	}
}

// TestCorpusGoProfile runs the P2 rows under spk-02's amd64 go allowlist (testgo@0).
func TestCorpusGoProfile(t *testing.T) {
	ensure(t)
	cases := []struct {
		prog  string
		check func(t *testing.T, c runnerapi.CaseResult)
	}{
		{"threadbomb", func(t *testing.T, c runnerapi.CaseResult) {
			// RE (fork limit): the Go runtime dies at the pids cap.
			if !c.ForkLimit || (c.Term != runnerapi.TermExitNonzero && c.Term != runnerapi.TermSignal) {
				t.Errorf("threadbomb: want RE with fork_limit, got %+v", c)
			}
		}},
		{"stdoutflood", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermOLE {
				t.Errorf("stdoutflood: want ole, got %+v", c)
			}
		}},
		{"spin", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermTLE || c.IdleKill || c.CPUms < 1000 {
				t.Errorf("spin: want a CPU TLE, got %+v", c)
			}
		}},
		{"sleep", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermTLE || !c.IdleKill || c.WallMs > 2500 {
				t.Errorf("sleep: want TLE (idle) within ~1 s, got %+v", c)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.prog, func(t *testing.T) {
			job, in := newJob(t, tc.prog, "testgo@0", "")
			res := run(t, job, in)
			tc.check(t, res.Cases[0])
			if res.Throttled && tc.prog != "spin" {
				t.Errorf("throttled: %+v", res.Telemetry)
			}
			assertClean(t)
		})
	}
}

// TestCorpusNamespaceLayer runs the rows the go allowlist would stop first with SIGSYS under
// testgo-open@0, to prove the namespace, tmpfs and pids layers on their own.
func TestCorpusNamespaceLayer(t *testing.T) {
	ensure(t)
	cases := []struct {
		prog  string
		check func(t *testing.T, c runnerapi.CaseResult)
	}{
		{"tmpfsfill", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermExitNonzero || c.ExitCode != 3 || !strings.Contains(string(c.Stdout), "no space left") {
				t.Errorf("tmpfsfill: want the program's own ENOSPC exit, got %+v stdout %q", c, c.Stdout)
			}
		}},
		{"inodefill", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermExitNonzero || c.ExitCode != 3 || !strings.Contains(string(c.Stdout), "no space left") {
				t.Errorf("inodefill: want the program's own ENOSPC exit, got %+v stdout %q", c, c.Stdout)
			}
		}},
		{"forkbomb", func(t *testing.T, c runnerapi.CaseResult) {
			if !c.ForkLimit || c.Term != runnerapi.TermExitNonzero {
				t.Errorf("forkbomb: want RE (fork limit), got %+v stdout %q", c, c.Stdout)
			}
		}},
		{"netprobe", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermOK || strings.Contains(string(c.Stdout), "REACHED") {
				t.Errorf("netprobe: every connect must fail, got %+v\n%s", c, c.Stdout)
			}
			t.Logf("netprobe:\n%s", c.Stdout)
		}},
		{"orphan", func(t *testing.T, c runnerapi.CaseResult) {
			if c.Term != runnerapi.TermOK || !strings.Contains(string(c.Stdout), "parent done") {
				t.Errorf("orphan: %+v stdout %q", c, c.Stdout)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.prog, func(t *testing.T) {
			job, in := newJob(t, tc.prog, "testgo-open@0", "")
			job.Mode = runnerapi.ModeRun
			res := run(t, job, in)
			tc.check(t, res.Cases[0])
			assertClean(t) // orphan: 0 surviving processes after teardown
		})
	}
}

// TestCrossJobMarkers: markers job N leaves in /w, /tmp, /dev/shm, SysV shm, a POSIX mq, an
// abstract socket and the user keyring are invisible to job N+1 (t3 §5.10).
func TestCrossJobMarkers(t *testing.T) {
	ensure(t)
	token := fmt.Sprintf("m%d", time.Now().UnixNano())
	job, in := newJob(t, "markers", "testgo-open@0", "write "+token)
	job.Mode = runnerapi.ModeRun
	w := run(t, job, in)
	t.Logf("writer:\n%s", w.Cases[0].Stdout)
	if w.Cases[0].Term != runnerapi.TermOK {
		t.Fatalf("writer: %+v", w.Cases[0])
	}
	job, in = newJob(t, "markers", "testgo-open@0", "read "+token, "read "+token)
	job.Mode = runnerapi.ModeRun
	r := run(t, job, in)
	for i, c := range r.Cases {
		if c.Term != runnerapi.TermOK || len(c.Output) != 0 {
			t.Errorf("reader case %d sees markers %q: %+v", i, c.Output, c)
		}
	}
	assertClean(t)
}

// TestSigsysRotates: a disallowed syscall is SIGSYS → Term=signal(SIGSYS) (judge: RE, counted),
// and the runner rotates right after the job (a new BootEpoch).
func TestSigsysRotates(t *testing.T) {
	p := ensure(t)
	before := getJSON[runnerapi.StatsResponse](t, "/v1/stats").BootEpoch
	job, in := newJob(t, "sigsys", "testgo@0", "bpf", "io_uring_setup", "unshare", "mount", "ptrace", "keyctl")
	res := run(t, job, in)
	for i, c := range res.Cases {
		if c.Term != runnerapi.TermSignal || c.Signal != "SIGSYS" {
			t.Errorf("probe %q: want signal SIGSYS, got %+v (output %q)", in[i], c, c.Output)
		}
	}
	if code := p.waitExit(t, 30*time.Second); code != 0 {
		t.Errorf("rotation exit code %d", code)
	}
	ensure(t)
	after := getJSON[runnerapi.StatsResponse](t, "/v1/stats").BootEpoch
	if after == before || after == "" {
		t.Errorf("BootEpoch did not change across the rotation: %q → %q", before, after)
	}
	if code, _ := get(t, "/healthz", false); code != http.StatusOK {
		t.Errorf("the restarted runner is not healthy")
	}
}
