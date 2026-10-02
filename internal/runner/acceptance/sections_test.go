//go:build runner_acceptance

package acceptance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// ---- A · contract ----

// sectionA: 401 without (or with a wrong) token; 400 for a bad profile, a harness the profile
// doesn't take, a file the profile doesn't take and a wrong media type; 503 + Retry-After +
// {"infra":{"kind":"saturated"}} for a third concurrent job; a client disconnect kills the job
// and sends nothing; pids.current is 0 afterwards.
func (s *suite) sectionA(t *testing.T, r *section) {
	noop := s.probeJob(t, "A", "go", "corpus", []string{"noop"}, runnerapi.ModeSubmit, limits{1000, 64})
	var body bytes.Buffer
	if err := runnerapi.EncodeJob(&body, noop.job, noop.inputs); err != nil {
		t.Fatal(err)
	}
	raw := func(ct, token string, b []byte) (int, []byte) {
		req, _ := http.NewRequest(http.MethodPost, cfg.url+"/v1/jobs", bytes.NewReader(b))
		req.Header.Set("Content-Type", ct)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := s.hc.Do(req)
		if err != nil {
			return 0, []byte(err.Error())
		}
		defer resp.Body.Close()
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		return resp.StatusCode, out.Bytes()
	}
	for name, tok := range map[string]string{"no token": "", "wrong token": "not-the-runner-token-0123456789"} {
		code, b := raw(runnerapi.ContentTypeJob, tok, body.Bytes())
		r.expect(t, code == http.StatusUnauthorized, "401 POST /v1/jobs, "+name, "%d %s", code, trunc(b, 120))
	}
	for _, path := range []string{"/v1/profiles", "/v1/stats"} {
		code, _, err := s.get(path, false)
		r.expect(t, code == http.StatusUnauthorized, "401 GET "+path+" without a token", "%d %v", code, err)
	}

	bad := map[string]func(j *runnerapi.Job){
		"unknown profile":                    func(j *runnerapi.Job) { j.Profile = "cobol@85" },
		"a harness the profile doesn't take": func(j *runnerapi.Job) { j.Harness = "raw@0" },
		"a file the profile doesn't take": func(j *runnerapi.Job) {
			j.Files = append(j.Files, runnerapi.File{Path: "evil_amd64.s", Data: []byte("TEXT ·x(SB),0,$0\n")})
		},
	}
	for name, mut := range bad {
		j := *noop.job
		j.Files = append([]runnerapi.File{}, noop.job.Files...)
		mut(&j)
		var b bytes.Buffer
		if err := runnerapi.EncodeJob(&b, &j, noop.inputs); err != nil {
			t.Fatal(err)
		}
		code, out := raw(runnerapi.ContentTypeJob, cfg.token, b.Bytes())
		r.expect(t, code == http.StatusBadRequest, "400 "+name, "%d %s", code, trunc(out, 160))
	}
	code, out := raw("application/json", cfg.token, body.Bytes())
	r.expect(t, code == http.StatusBadRequest, "400 wrong media type", "%d %s", code, trunc(out, 160))
	s.checkStable(t, "A 4xx probes")

	// 503 on a third concurrent job: two fillers hold both slots.
	var wg sync.WaitGroup
	fill := make([]*runnerapi.Result, 2)
	fillErr := make([]error, 2)
	for i := range fill {
		f := s.probeJob(t, "A", "go", "corpus", []string{"spin:2500", "spin:2500"}, runnerapi.ModeSubmit, limits{5000, 64})
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, _, b, res, err := s.post(context.Background(), f.job, f.inputs)
			if err == nil && code != http.StatusOK {
				err = fmt.Errorf("%d %s", code, trunc(b, 160))
			}
			fill[i], fillErr[i] = res, err
		}(i)
	}
	busy := s.waitBusy(2, 60*time.Second)
	if r.expect(t, busy, "two fillers hold both slots", "busy=%v", busy) {
		third := s.probeJob(t, "A", "go", "corpus", []string{"noop"}, runnerapi.ModeSubmit, limits{1000, 64})
		code, h, b, res, err := s.post(context.Background(), third.job, third.inputs)
		ok := err == nil && code == http.StatusServiceUnavailable && h.Get("Retry-After") != "" && res != nil &&
			res.Infra != nil && res.Infra.Kind == runnerapi.InfraSaturated && bytes.Contains(b, []byte(`"infra":{"kind":"saturated"}`))
		r.expect(t, ok, "503 saturated on a third concurrent job", "%d Retry-After=%q %s %v", code, h.Get("Retry-After"), trunc(b, 160), err)
	}
	wg.Wait()
	for i := range fill {
		ok := fillErr[i] == nil && fill[i] != nil && fill[i].Infra == nil && len(fill[i].Cases) == 2 &&
			fill[i].Cases[0].Term == runnerapi.TermOK && fill[i].Cases[1].Term == runnerapi.TermOK
		r.expect(t, ok, fmt.Sprintf("filler %d completes", i+1), "%v %+v", fillErr[i], fill[i])
	}
	s.report.Jobs += 2
	s.report.Cases += 4
	s.checkStable(t, "A fillers")

	// A client disconnect kills the job subtree and sends no Result.
	spin := s.probeJob(t, "A", "go", "corpus", []string{"spin:0"}, runnerapi.ModeSubmit, limits{10000, 64})
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, _, _, _, err := s.post(ctx, spin.job, spin.inputs)
		errc <- err
	}()
	if s.waitBusy(1, 60*time.Second) {
		time.Sleep(1500 * time.Millisecond)
	}
	cancel()
	err := <-errc
	r.expect(t, err != nil, "a disconnect gets no Result", "post error %v", err)
	st, idle := s.waitIdle(20 * time.Second)
	r.expect(t, idle, "the disconnected job is killed (slots idle, pids.current 0)", "%+v", st.Slots)
	s.report.Jobs++
	s.checkStable(t, "A disconnect")
	res := s.run(t, s.probeJob(t, "A", "go", "corpus", []string{"noop"}, runnerapi.ModeSubmit, limits{1000, 64}))
	r.expect(t, res.Cases[0].Term == runnerapi.TermOK, "a job runs after the disconnect", "%s", describe(res.Cases[0]))
	st, idle = s.waitIdle(10 * time.Second)
	r.expect(t, idle, "pids.current 0 afterwards", "%+v", st.Slots)
}

func (s *suite) waitBusy(n int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if st, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats"); err == nil {
			busy := 0
			for _, sl := range st.Slots {
				if sl.Busy {
					busy++
				}
			}
			if busy >= n {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// ---- B · network ----

var netProbes = []string{
	"tcp:1.1.1.1:443",   // the internet
	"tcp:10.43.0.1:443", // the apiserver VIP
	"udp:10.43.0.10:53", // kube-dns, UDP (an answer would count)
	"tcp:10.43.0.10:53", // kube-dns, TCP
	"dns:kubernetes.default.svc.cluster.local",
	"bind:127.0.0.1:0", // loopback is down
}

// sectionB: per language, one job: every network attempt fails (signal SIGSYS from the exec
// filter, or an in-process error); one rotation per job.
func (s *suite) sectionB(t *testing.T, r *section) {
	for _, lang := range langs {
		j := s.probeJob(t, "B", lang, "net", netProbes, runnerapi.ModeRun, limits{2000, 128})
		j.planSigsys = true
		res := s.run(t, j)
		r.expect(t, res.Compile.OK, lang+" compiles", "%+v", res.Compile)
		sig := 0
		for i, c := range res.Cases {
			out, _ := probeOut(c)
			ok := isSIGSYS(c) || (c.Term == runnerapi.TermOK && strings.HasPrefix(out, "blocked")) ||
				(c.Term == runnerapi.TermTLE && c.IdleKill)
			if isSIGSYS(c) {
				sig++
			}
			r.expect(t, ok, fmt.Sprintf("%s %s fails", lang, netProbes[i]), "%s", describe(c))
		}
		r.metric(lang+"_sigsys", fmt.Sprintf("%d/%d", sig, len(res.Cases)))
	}
}

// ---- C · syscalls ----

var syscallProbes = []string{
	"io_uring_setup", "bpf", "perf_event_open", "userfaultfd", "keyctl", "add_key", "ptrace",
	"process_vm_readv", "mount", "unshare_newuser", "setns", "clone_newns", "socket_netlink",
}

// sectionC: per language, one job: each dangerous call is signal(SIGSYS) (the image's exec
// filters are KILL-default on amd64 and on arm64); one rotation per job; /readyz still reports
// core_pattern not a pipe.
func (s *suite) sectionC(t *testing.T, r *section) {
	for _, lang := range langs {
		j := s.probeJob(t, "C", lang, "syscalls", syscallProbes, runnerapi.ModeRun, limits{2000, 128})
		j.planSigsys = true
		res := s.run(t, j)
		r.expect(t, res.Compile.OK, lang+" compiles", "%+v", res.Compile)
		atCall := 0
		for i, c := range res.Cases {
			r.expect(t, isSIGSYS(c), fmt.Sprintf("%s %s is SIGSYS", lang, syscallProbes[i]), "%s", describe(c))
			if bytes.Contains(c.Stderr, []byte("calling ")) {
				atCall++
			}
		}
		// Where the kill came: at the probed call itself, or earlier (Python: ctypes' own setup).
		r.metric(lang+"_killed_at_the_call", fmt.Sprintf("%d/%d", atCall, len(res.Cases)))
	}
	code, rz, err := s.readyz()
	cp := rz.Checks["core_pattern_not_pipe"]
	if s.report.Mode == "prod" || cfg.requireProd {
		r.expect(t, code == http.StatusOK && cp == "pass", "/readyz: core_pattern is not a pipe", "%d %s %v", code, cp, err)
	} else {
		r.add("/readyz: core_pattern is not a pipe (dev: reported only)", true, fmt.Sprintf("%d core_pattern_not_pipe=%s", code, cp))
	}
}

// ---- D · P2 corpus ----

type dCheck struct {
	prog  string
	want  string
	check func(c runnerapi.CaseResult, out string) bool
}

var (
	wantMLE  = func(c runnerapi.CaseResult, _ string) bool { return c.Term == runnerapi.TermMLE }
	wantOLE  = func(c runnerapi.CaseResult, _ string) bool { return c.Term == runnerapi.TermOLE }
	wantSpin = func(c runnerapi.CaseResult, _ string) bool { return c.Term == runnerapi.TermTLE && !c.IdleKill }
	wantIdle = func(c runnerapi.CaseResult, _ string) bool { return c.Term == runnerapi.TermTLE && c.IdleKill }
	wantFill = func(c runnerapi.CaseResult, out string) bool {
		return isSIGSYS(c) || (c.Term == runnerapi.TermOK && strings.Contains(lower(out), "no space left"))
	}
	wantBomb   = func(c runnerapi.CaseResult, _ string) bool { return isRE(c) && (c.ForkLimit || isSIGSYS(c)) }
	wantOrphan = func(c runnerapi.CaseResult, out string) bool {
		return isSIGSYS(c) || (c.Term == runnerapi.TermOK && out == "parent done")
	}
	dChecks = map[string]dCheck{
		"balloon":     {"balloon", "MLE", wantMLE},
		"tmpfsfill":   {"tmpfsfill", "ENOSPC or SIGSYS", wantFill},
		"inodefill":   {"inodefill", "ENOSPC or SIGSYS", wantFill},
		"stdoutflood": {"stdoutflood", "OLE", wantOLE},
		"spin:0":      {"spin", "TLE (CPU)", wantSpin},
		"sleep":       {"sleep", "TLE (idle)", wantIdle},
		"threadbomb":  {"threadbomb", "RE (fork limit or SIGSYS)", wantBomb},
		"forkbomb":    {"forkbomb", "RE (fork limit or SIGSYS)", wantBomb},
		"orphan":      {"orphan", "SIGSYS, or the parent returns", wantOrphan},
	}
	// dPlan is each language's corpus: the contained programs in one job, the SIGSYS-expected ones
	// (the profile's exec filter denies the fork, the file open or the thread) in one job, last.
	dPlan = map[string]struct{ contained, sigsys []string }{
		"go":     {[]string{"tmpfsfill", "inodefill", "stdoutflood", "spin:0", "sleep", "threadbomb"}, []string{"forkbomb", "orphan"}},
		"cpp":    {[]string{"balloon", "stdoutflood", "spin:0", "sleep"}, []string{"tmpfsfill", "inodefill", "forkbomb", "threadbomb", "orphan"}},
		"python": {[]string{"balloon", "tmpfsfill", "inodefill", "stdoutflood", "spin:0", "sleep", "threadbomb"}, []string{"forkbomb", "orphan"}},
	}
)

// sectionD: the 1 GiB balloon × 100 (Go) is MLE 100/100 with no container OOM; every other
// corpus program once per language gives its specified terminal state; 0 survivors (pids.current
// 0 in every slot afterwards).
func (s *suite) sectionD(t *testing.T, r *section) {
	mle := 0
	for b := 0; b < 4; b++ {
		args := make([]string, 25)
		for i := range args {
			args[i] = "balloon"
		}
		res := s.run(t, s.probeJob(t, "D", "go", "corpus", args, runnerapi.ModeSubmit, limits{2000, 256}))
		for i, c := range res.Cases {
			if c.Term == runnerapi.TermMLE {
				mle++
			} else {
				r.add(fmt.Sprintf("go balloon job %d case %d is MLE", b+1, i), false, describe(c))
				t.Errorf("balloon: %s", describe(c))
			}
		}
	}
	r.expect(t, mle == 100, "balloon × 100 is MLE 100/100", "%d/100", mle)
	r.metric("balloon_mle", fmt.Sprintf("%d/100", mle))
	for _, lang := range langs {
		plan := dPlan[lang]
		for _, part := range []struct {
			progs  []string
			sigsys bool
		}{{plan.contained, false}, {plan.sigsys, true}} {
			j := s.probeJob(t, "D", lang, "corpus", part.progs, runnerapi.ModeSubmit, limits{1000, 256})
			j.planSigsys = part.sigsys
			res := s.run(t, j)
			if !r.expect(t, res.Compile.OK, lang+" corpus compiles", "%+v", res.Compile) {
				continue
			}
			for i, c := range res.Cases {
				dc := dChecks[part.progs[i]]
				out, _ := probeOut(c)
				r.expect(t, dc.check(c, out), fmt.Sprintf("%s %s is %s", lang, dc.prog, dc.want), "%s", describe(c))
			}
		}
	}
	st, idle := s.waitIdle(15 * time.Second)
	r.expect(t, idle, "0 survivors (pids.current 0 in every slot)", "%+v", st.Slots)
	r.expect(t, s.report.OOM.ContainerLocalDelta == 0, "container oom_kill delta 0", "%d", s.report.OOM.ContainerLocalDelta)
}

// ---- E · cross-job markers ----

var markerChannels = []string{"w", "tmp", "devshm", "sysvshm", "posixmq", "abstract", "keyring"}

func token() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// sectionE: per language, one job tries every channel (the denied ones die of SIGSYS: one
// rotation); then the channels that were writable are written by job N and looked for by job
// N+1 under the same BootEpoch (asserted). Nothing may be visible.
func (s *suite) sectionE(t *testing.T, r *section) {
	for _, lang := range langs {
		tok := token()
		args := make([]string, len(markerChannels))
		for i, ch := range markerChannels {
			args[i] = "write:" + ch + ":" + tok
		}
		j := s.probeJob(t, "E", lang, "markers", args, runnerapi.ModeRun, limits{2000, 128})
		j.planSigsys = true
		res := s.run(t, j)
		if !r.expect(t, res.Compile.OK, lang+" markers compile", "%+v", res.Compile) {
			continue
		}
		var allowed, denied []string
		for i, c := range res.Cases {
			out, _ := probeOut(c)
			switch {
			case c.Term == runnerapi.TermOK && out == "written":
				allowed = append(allowed, markerChannels[i])
			case isSIGSYS(c) || (c.Term == runnerapi.TermOK && strings.HasPrefix(out, "denied")):
				denied = append(denied, markerChannels[i])
			default:
				r.expect(t, false, fmt.Sprintf("%s %s write is written or denied", lang, markerChannels[i]), "%s", describe(c))
			}
		}
		r.metric(lang+"_allowed", allowed)
		r.metric(lang+"_denied", denied)
		if len(allowed) == 0 {
			r.add(lang+": no channel is writable", true, "every marker was denied (nothing can cross)")
			continue
		}
		tok = token()
		w := make([]string, len(allowed))
		rd := make([]string, len(allowed))
		for i, ch := range allowed {
			w[i], rd[i] = "write:"+ch+":"+tok, "read:"+ch+":"+tok
		}
		wres := s.run(t, s.probeJob(t, "E", lang, "markers", w, runnerapi.ModeRun, limits{2000, 128}))
		epoch := wres.Versions.BootEpoch
		for i, c := range wres.Cases {
			out, _ := probeOut(c)
			r.expect(t, c.Term == runnerapi.TermOK && out == "written", fmt.Sprintf("%s job N writes %s", lang, allowed[i]), "%s", describe(c))
		}
		rres := s.run(t, s.probeJob(t, "E", lang, "markers", rd, runnerapi.ModeRun, limits{2000, 128}))
		r.expect(t, rres.Versions.BootEpoch == epoch, lang+" job N+1 runs under job N's BootEpoch", "%s vs %s", rres.Versions.BootEpoch, epoch)
		for i, c := range rres.Cases {
			out, _ := probeOut(c)
			r.expect(t, c.Term == runnerapi.TermOK && strings.HasPrefix(out, "absent"), fmt.Sprintf("%s job N+1 can't see %s", lang, allowed[i]), "%s", describe(c))
		}
	}
}

// ---- F · cleanup ----

// sectionF: ≥ 1,000 case cgroups after a warm-up, then /v1/stats: every slot back at its
// baseline ±5 MiB with pids.current 0, nr_dying_descendants ~0 within 60 s, runner/ at its
// baseline ±5 MiB.
func (s *suite) sectionF(t *testing.T, r *section) {
	noops := func() job {
		args := make([]string, 500)
		for i := range args {
			args[i] = "noop"
		}
		return s.probeJob(t, "F", "go", "corpus", args, runnerapi.ModeSubmit, limits{1000, 64})
	}
	t0 := time.Now()
	res := s.run(t, noops()) // warm-up at full size: the front's heap has seen a 500-case Result
	el := time.Since(t0)
	var wall int64
	for _, c := range res.Cases {
		wall += c.WallMs
	}
	r.metric("per_case_ms", round(float64(el.Milliseconds())/500, 2))
	r.metric("per_case_jail_overhead_ms", round(float64(el.Milliseconds()-wall)/500, 2))
	_, idle := s.waitIdle(15 * time.Second)
	r.expect(t, idle, "idle after the warm-up", "")
	if d, ok := s.settleDying(60 * time.Second); !ok {
		r.expect(t, false, "nr_dying_descendants settles after the warm-up", "%v", d)
	}
	base, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats")
	if err != nil {
		s.abort(t, "/v1/stats: %v", err)
	}
	cases := 0
	for j := 0; j < 2; j++ {
		res := s.run(t, noops())
		for i, c := range res.Cases {
			if c.Term != runnerapi.TermOK {
				r.expect(t, false, fmt.Sprintf("noop job %d case %d", j+1, i), "%s", describe(c))
				break
			}
		}
		cases += len(res.Cases)
	}
	r.metric("case_cgroups_after_baseline", cases)
	_, idle = s.waitIdle(15 * time.Second)
	took, settled := s.settleDying(60 * time.Second)
	r.expect(t, settled, "nr_dying_descendants → ~0 within 60 s", "took %v", took)
	r.metric("nr_dying_settle_s", round(took.Seconds(), 1))
	after, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats")
	if err != nil {
		s.abort(t, "/v1/stats: %v", err)
	}
	const tol = 5 << 20
	drift := after.RunnerMemoryCurrent - base.RunnerMemoryCurrent
	r.metric("runner_memory_drift_kib", drift/1024)
	r.expect(t, drift <= tol && drift >= -tol, "runner/ at its baseline ±5 MiB", "drift %d KiB", drift/1024)
	for i := range after.Slots {
		d := after.Slots[i].MemoryCurrent - base.Slots[i].MemoryCurrent
		r.metric(fmt.Sprintf("slot%d_memory_drift_kib", i), d/1024)
		r.expect(t, d <= tol && d >= -tol, fmt.Sprintf("slot %d memory at its baseline ±5 MiB", i), "drift %d KiB", d/1024)
		r.expect(t, after.Slots[i].PidsCurrent == 0, fmt.Sprintf("slot %d pids.current 0", i), "%d", after.Slots[i].PidsCurrent)
	}
	r.expect(t, cases >= 1000, "≥ 1,000 case cgroups", "%d", cases)
}

func (s *suite) settleDying(within time.Duration) (time.Duration, bool) {
	t0 := time.Now()
	for {
		st, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats")
		if err == nil {
			ok := true
			for _, sl := range st.Slots {
				if sl.NrDyingDescendants > 2 {
					ok = false
				}
			}
			if ok {
				return time.Since(t0), true
			}
		}
		if time.Since(t0) > within {
			return time.Since(t0), false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// ---- H · canary ----

// sectionH reports canary_median_us over the run (sampled from /v1/profiles after every job) and
// its coefficient of variation. The thresholds are mi-10's (from S0); nothing is gated here.
func (s *suite) sectionH(t *testing.T, r *section) {
	s.sampleCanary()
	xs := make([]float64, len(s.canary))
	for i, v := range s.canary {
		xs[i] = float64(v)
	}
	med, cv, lo, hi := stats(xs)
	s.report.Canary = canaryReport{Samples: len(xs), MedianUs: med, MinUs: int64(lo), MaxUs: int64(hi), CV: round(cv, 4),
		Threshold: "reported only; mi-10 sets thresholds from S0"}
	r.add("canary median reported", len(xs) > 0, fmt.Sprintf("median %.0f µs, CV %.4f, %d samples", med, cv, len(xs)))
}

// ---- I · prod-only ----

var prodChecks = []string{"apparmor_label", "uid_map_not_identity", "userns_denied", "fsopen_denied", "sctp_denied",
	"core_pattern_not_pipe", "cgroupfs_writable"}

func (s *suite) prodChecks(t *testing.T, r *section) {
	p, err := getJSON[runnerapi.ProfilesResponse](s, "/v1/profiles")
	if err != nil {
		s.abort(t, "/v1/profiles: %v", err)
	}
	if !r.expect(t, p.Mode == "prod", "/v1/profiles mode=prod", "mode=%s", p.Mode) {
		s.abort(t, "REQUIRE_PROD=1 and the runner is in mode=%s: a dev runner fails the run", p.Mode)
	}
	code, rz, err := s.readyz()
	r.expect(t, code == http.StatusOK && rz.Status == "ok", "/readyz 200", "%d %v %v", code, rz.Checks, err)
	for _, c := range prodChecks {
		r.expect(t, rz.Checks[c] == "pass", "/readyz "+c, "%q", rz.Checks[c])
	}
	var extra []string
	for k, v := range rz.Checks {
		if strings.HasPrefix(k, "egress_blocked:") || (v != "pass" && !contains(prodChecks, k)) {
			extra = append(extra, k+"="+v)
		}
	}
	sort.Strings(extra)
	r.expect(t, len(extra) == 0, "/readyz: every egress target blocked, no other failing check", "%v", extra)
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// sectionIPre fails a dev-mode runner before any job runs (REQUIRE_PROD=1).
func (s *suite) sectionIPre(t *testing.T, r *section) { s.prodChecks(t, r) }

// sectionI re-checks the prod canaries at the end of the run, on the last epoch.
func (s *suite) sectionI(t *testing.T, r *section) { s.prodChecks(t, r) }
