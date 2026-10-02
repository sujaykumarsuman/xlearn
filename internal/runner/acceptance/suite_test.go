//go:build runner_acceptance

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

// suite is the run's state: the HTTP client, the runner epoch being tracked, and the report.
type suite struct {
	hc     *http.Client
	report *Report
	// profiles maps an item language (go, cpp, python) to the runner's profile for it.
	profiles map[string]runnerapi.ProfileInfo
	// epoch is the BootEpoch every job must run under until an expected rotation.
	epoch string
	// epochJobs counts the jobs this suite sent in the current epoch (RUNNER_MAX_JOBS = 500).
	epochJobs int
	// oomLocal and oomHier are the container's oom_kill counters at the start of the epoch.
	oomLocal, oomHier int64
	canary            []int64
	aborted           string
	jobN              int
}

func newSuite(t *testing.T) *suite {
	return &suite{
		hc: &http.Client{Timeout: 210 * time.Second}, // the runner's own job backstop is 170 s
		report: &Report{Schema: "runner-acceptance@1", Started: time.Now().UTC(), Subset: cfg.subset,
			RequireProd: cfg.requireProd, Calibrate: cfg.calibrate, Target: targetOf(cfg.url)},
		profiles: map[string]runnerapi.ProfileInfo{},
	}
}

// abort records a stop condition and ends the current section; every later section is skipped.
func (s *suite) abort(t *testing.T, format string, args ...any) {
	t.Helper()
	s.aborted = fmt.Sprintf(format, args...)
	s.report.Aborted = s.aborted
	t.Fatalf("ABORT: %s", s.aborted)
}

// ---- HTTP ----

func (s *suite) do(ctx context.Context, method, path string, auth bool, ct string, body []byte) (int, http.Header, []byte, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, cfg.url+path, rd)
	if err != nil {
		return 0, nil, nil, err
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+cfg.token)
	}
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	resp, err := s.hc.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, b, err
}

func (s *suite) get(path string, auth bool) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	code, _, b, err := s.do(ctx, http.MethodGet, path, auth, "", nil)
	return code, b, err
}

func getJSON[T any](s *suite, path string) (T, error) {
	var v T
	code, b, err := s.get(path, true)
	if err != nil {
		return v, err
	}
	if code != http.StatusOK {
		return v, fmt.Errorf("GET %s: %d %s", path, code, trunc(b, 200))
	}
	return v, json.Unmarshal(b, &v)
}

func trunc(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}

type readyz struct {
	Status string            `json:"status"`
	Mode   string            `json:"mode"`
	Checks map[string]string `json:"checks"`
}

func (s *suite) readyz() (int, readyz, error) {
	var r readyz
	code, b, err := s.get("/readyz", false)
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &r)
	}
	return code, r, err
}

// ---- the runner's state ----

// start connects, records the runner's identity and the first epoch's baselines.
func (s *suite) start(t *testing.T) bool {
	t.Helper()
	r := s.report.newSection("0", "connect")
	defer r.done()
	p, err := s.waitReady("", *flagRotation)
	if err != nil {
		r.add("runner reachable", false, err.Error())
		s.report.Aborted = "runner not reachable: " + err.Error()
		s.aborted = s.report.Aborted
		t.Errorf("runner not reachable at %s: %v", s.report.Target, err)
		return false
	}
	s.report.Runner, s.report.Mode = p.Runner, p.Mode
	s.report.Profiles = p.Profiles
	for _, pi := range p.Profiles {
		if pi.Language != "" {
			s.profiles[pi.Language] = pi
		}
		if s.report.ImageDigest == "" {
			s.report.ImageDigest = pi.ImageDigest
		}
	}
	for _, lang := range langs {
		if _, ok := s.profiles[lang]; !ok {
			r.add("profiles", false, "no profile serves "+lang)
			s.aborted, s.report.Aborted = "no profile serves "+lang, "no profile serves "+lang
			t.Errorf("GET /v1/profiles: no profile serves %s", lang)
			return false
		}
	}
	r.add("runner reachable", true, fmt.Sprintf("%s mode=%s epoch=%s", p.Runner, p.Mode, p.BootEpoch))
	s.newEpoch(t, p.BootEpoch)
	return true
}

// newEpoch starts tracking epoch: its oom_kill baselines come from /v1/stats.
func (s *suite) newEpoch(t *testing.T, epoch string) {
	t.Helper()
	st, err := s.statsRetry(10 * time.Second)
	if err != nil {
		s.abort(t, "/v1/stats on the new epoch %s: %v", epoch, err)
	}
	if st.BootEpoch != epoch {
		s.abort(t, "/v1/stats reports epoch %s right after /v1/profiles reported %s: an unexplained rotation", st.BootEpoch, epoch)
	}
	s.epoch, s.epochJobs = epoch, 0
	s.oomLocal, s.oomHier = st.ContainerOOMKillsLocal, st.ContainerOOMKills
}

func (s *suite) statsRetry(within time.Duration) (runnerapi.StatsResponse, error) {
	deadline := time.Now().Add(within)
	for {
		st, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats")
		if err == nil || time.Now().After(deadline) {
			return st, err
		}
		time.Sleep(time.Second)
	}
}

// waitReady polls /readyz and /v1/profiles until the runner is ready on an epoch other than
// prev (any epoch when prev is ""). It tolerates connection errors and 503s: the old container
// draining, the restart back-off (Docker's --restart=always, kubelet's CrashLoopBackOff up to
// 5 min) and the port-forward's retry loop.
func (s *suite) waitReady(prev string, within time.Duration) (runnerapi.ProfilesResponse, error) {
	deadline := time.Now().Add(within)
	var last string
	for {
		code, rz, err := s.readyz()
		switch {
		case err != nil:
			last = err.Error()
		case code != http.StatusOK:
			last = fmt.Sprintf("/readyz %d %v", code, rz.Checks)
		default:
			p, err := getJSON[runnerapi.ProfilesResponse](s, "/v1/profiles")
			switch {
			case err != nil:
				last = err.Error()
			case p.BootEpoch == "" || p.BootEpoch == prev:
				last = "still on epoch " + p.BootEpoch
			default:
				return p, nil
			}
		}
		if time.Now().After(deadline) {
			return runnerapi.ProfilesResponse{}, fmt.Errorf("not ready on a new epoch within %v (last: %s)", within, last)
		}
		time.Sleep(time.Second)
	}
}

// rotate waits across one expected rotation and starts tracking the new epoch.
func (s *suite) rotate(t *testing.T, sec, job, reason string) {
	t.Helper()
	prev, t0 := s.epoch, time.Now()
	p, err := s.waitReady(prev, *flagRotation)
	if err != nil {
		s.abort(t, "after %s (%s): %v", job, reason, err)
	}
	s.report.Rotations.Observed++
	s.report.Rotations.Events = append(s.report.Rotations.Events, rotationEvent{Section: sec, Job: job, Reason: reason,
		From: prev, To: p.BootEpoch, WaitS: round(time.Since(t0).Seconds(), 1)})
	t.Logf("rotation after %s (%s): %s → %s in %.1f s", job, reason, prev, p.BootEpoch, time.Since(t0).Seconds())
	s.newEpoch(t, p.BootEpoch)
}

// checkStable is the between-jobs stop-condition check outside an expected rotation: /v1/stats
// answers, on the same epoch, with no container OOM.
func (s *suite) checkStable(t *testing.T, job string) runnerapi.StatsResponse {
	t.Helper()
	st, err := s.statsRetry(10 * time.Second)
	if err != nil {
		s.abort(t, "/v1/stats stopped answering after %s, outside an expected rotation: %v", job, err)
	}
	s.accountOOM(st)
	switch {
	case st.BootEpoch != s.epoch:
		s.abort(t, "unexplained BootEpoch change after %s: %s → %s (a possible container OOM)", job, s.epoch, st.BootEpoch)
	case st.ContainerOOMKillsLocal > s.oomLocal:
		s.abort(t, "container oom_kill (memory.events.local) rose %d → %d after %s: INV-14 broken", s.oomLocal, st.ContainerOOMKillsLocal, job)
	case st.Draining:
		if s.epochJobs >= 500 {
			s.report.Rotations.Expected++
			s.rotate(t, "", job, "max_jobs")
			return st
		}
		s.abort(t, "the runner is draining after %s, outside an expected rotation", job)
	}
	s.sampleCanary()
	return st
}

// accountOOM folds the epoch's counters into the report (they restart with every epoch).
func (s *suite) accountOOM(st runnerapi.StatsResponse) {
	if st.BootEpoch != s.epoch {
		return
	}
	s.report.OOM.ContainerLocalDelta += st.ContainerOOMKillsLocal - s.oomLocal
	s.report.OOM.ContainerHierDelta += st.ContainerOOMKills - s.oomHier
	s.oomLocal, s.oomHier = st.ContainerOOMKillsLocal, st.ContainerOOMKills
}

func (s *suite) sampleCanary() {
	if p, err := getJSON[runnerapi.ProfilesResponse](s, "/v1/profiles"); err == nil && p.CanaryMedian > 0 {
		s.canary = append(s.canary, p.CanaryMedian)
	}
}

// waitIdle waits until no slot is busy and every slot's pids.current is 0.
func (s *suite) waitIdle(within time.Duration) (runnerapi.StatsResponse, bool) {
	deadline := time.Now().Add(within)
	for {
		st, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats")
		if err == nil {
			idle := true
			for _, sl := range st.Slots {
				if sl.Busy || sl.PidsCurrent != 0 {
					idle = false
				}
			}
			if idle {
				return st, true
			}
		}
		if time.Now().After(deadline) {
			return st, false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// ---- jobs ----

// job is one POST /v1/jobs with what the suite expects of it.
type job struct {
	section string
	label   string
	job     *runnerapi.Job
	inputs  [][]byte
	// planSigsys marks a job whose cases are expected (or allowed) to die of SIGSYS: it counts
	// toward the planned rotations.
	planSigsys bool
}

// post sends a job and returns the raw response (no state checks).
func (s *suite) post(ctx context.Context, j *runnerapi.Job, inputs [][]byte) (int, http.Header, []byte, *runnerapi.Result, error) {
	var body bytes.Buffer
	if err := runnerapi.EncodeJob(&body, j, inputs); err != nil {
		return 0, nil, nil, nil, err
	}
	code, h, b, err := s.do(ctx, http.MethodPost, "/v1/jobs", true, runnerapi.ContentTypeJob, body.Bytes())
	if err != nil {
		return 0, nil, nil, nil, err
	}
	var res *runnerapi.Result
	if code == http.StatusOK || code == http.StatusServiceUnavailable {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		var r runnerapi.Result
		if err := dec.Decode(&r); err != nil {
			return code, h, b, nil, fmt.Errorf("decode the Result: %v: %s", err, trunc(b, 300))
		}
		res = &r
	}
	return code, h, b, res, nil
}

// run sends one job through the rotation-aware path: a 503 saturated is retried, any other
// non-200 or an infra error aborts; a Result with a SIGSYS case is followed by a wait across the
// rotation it causes; any other job is followed by the stop-condition check.
func (s *suite) run(t *testing.T, j job) *runnerapi.Result {
	t.Helper()
	if j.planSigsys {
		s.report.Rotations.Planned++
	}
	s.report.Jobs++
	s.report.Cases += len(j.job.Cases)
	deadline := time.Now().Add(3 * time.Minute)
	var res *runnerapi.Result
	for {
		code, h, b, r, err := s.post(context.Background(), j.job, j.inputs)
		if err != nil {
			if _, serr := s.statsRetry(10 * time.Second); serr != nil {
				s.abort(t, "%s: POST /v1/jobs: %v; /v1/stats: %v (the runner went away outside an expected rotation)", j.label, err, serr)
			}
			s.abort(t, "%s: POST /v1/jobs: %v", j.label, err)
		}
		if code == http.StatusServiceUnavailable && r != nil && r.Infra != nil && r.Infra.Kind == runnerapi.InfraSaturated && time.Now().Before(deadline) {
			wait, _ := strconv.Atoi(h.Get("Retry-After"))
			time.Sleep(time.Duration(min(max(wait, 1), 10)) * time.Second)
			continue
		}
		if code != http.StatusOK || r == nil {
			s.abort(t, "%s: POST /v1/jobs: %d %s", j.label, code, trunc(b, 300))
		}
		if r.Infra != nil {
			s.abort(t, "%s: infra error %s (%s)", j.label, r.Infra.Kind, r.Infra.Detail)
		}
		if err := runnerapi.ValidateResult(j.job, r); err != nil {
			s.abort(t, "%s: judge's ValidateResult rejects the Result: %v", j.label, err)
		}
		res = r
		break
	}
	if res.Versions.BootEpoch != s.epoch {
		s.abort(t, "%s ran on epoch %s, the suite tracked %s: an unexplained rotation", j.label, res.Versions.BootEpoch, s.epoch)
	}
	if s.report.CPUModel == "" {
		s.report.CPUModel = res.Versions.CPUModel
	}
	s.epochJobs++
	if n := sigsysCases(res); n > 0 {
		s.report.Rotations.Expected++
		s.rotate(t, j.section, j.label, fmt.Sprintf("sigsys ×%d", n))
	} else {
		s.checkStable(t, j.label)
	}
	return res
}

func sigsysCases(r *runnerapi.Result) int {
	n := 0
	for _, c := range r.Cases {
		if c.Term == runnerapi.TermSignal && c.Signal == "SIGSYS" {
			n++
		}
	}
	return n
}

// ---- finish ----

func (s *suite) finish(t *testing.T) {
	s.report.Finished = time.Now().UTC()
	if s.aborted == "" && s.epoch != "" {
		if st, err := getJSON[runnerapi.StatsResponse](s, "/v1/stats"); err == nil {
			s.accountOOM(st)
		}
	}
	s.report.Pass = s.aborted == "" && !t.Failed()
	for _, sec := range s.report.Sections {
		if !sec.Pass {
			s.report.Pass = false
		}
	}
	if s.report.Rotations.Expected != s.report.Rotations.Observed {
		s.report.Pass = false
		t.Errorf("rotations: expected %d, observed %d", s.report.Rotations.Expected, s.report.Rotations.Observed)
	}
	md, err := s.report.write()
	if err != nil {
		t.Errorf("write the report: %v", err)
	}
	fmt.Printf("\n%s\nJSON report: %s\n", md, cfg.report)
}

var errNoOutput = errors.New("no harness output")

func lower(s string) string { return strings.ToLower(s) }
