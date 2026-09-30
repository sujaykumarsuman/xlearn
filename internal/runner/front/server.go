package front

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
	"github.com/sujaykumarsuman/xlearn/internal/runner/measure"
	"github.com/sujaykumarsuman/xlearn/internal/runner/profile"
)

// Retry-After values (seconds) for the 503s. Judge re-queues with run_after; a 503 is never a
// verdict.
const (
	retrySaturated = 5
	retryQuiet     = 30
	retryDraining  = 30
)

// Cancel causes for a job's context.
var (
	errKilled     = errors.New("runner draining: job killed")
	errJobTimeout = errors.New("runner job deadline passed")
)

// Server is the front's HTTP API and job orchestration.
type Server struct {
	cfg     runner.Config
	man     *runner.Manifest
	backend Backend
	log     *slog.Logger
	started time.Time
	// profiles maps a profile name to its registry index and manifest facts.
	profiles map[string]profileEntry

	mu       sync.Mutex
	busy     [ipc.Slots]bool
	closed   [ipc.Slots]bool
	draining bool
	quiet    int // the slot holding quiet-re-run exclusivity, or -1
	// quietWait bounds the quiet re-run's wait (60 s); quietCheck is one QuietCheck's steal window.
	quietWait, quietCheck time.Duration
	running               map[int]context.CancelCauseFunc
	seq                   uint64
	inflight              sync.WaitGroup

	served atomic.Int64
	sigsys atomic.Int64

	ready readiness

	drainOnce sync.Once
	drained   chan struct{}
	http      *http.Server
}

type profileEntry struct {
	prof *profile.Profile
	idx  int
	man  runner.ProfileManifest
}

// NewServer builds the front. The manifest came from the spawner at spawn time.
func NewServer(cfg runner.Config, man *runner.Manifest, b Backend, log *slog.Logger) *Server {
	s := &Server{cfg: cfg, man: man, backend: b, log: log, started: time.Now(), quiet: -1,
		quietWait: measure.QuietWait, quietCheck: measure.QuietStealWindow,
		running: map[int]context.CancelCauseFunc{}, drained: make(chan struct{}), profiles: map[string]profileEntry{}}
	for _, pm := range man.Profiles {
		if p, idx, ok := profile.Lookup(pm.Name); ok {
			s.profiles[pm.Name] = profileEntry{prof: p, idx: idx, man: pm}
		}
	}
	return s
}

// Handler is the front's routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/jobs", s.authed(s.handleJob))
	mux.HandleFunc("GET /v1/profiles", s.authed(s.handleProfiles))
	mux.HandleFunc("GET /v1/stats", s.authed(s.handleStats))
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	return mux
}

// authed checks the bearer token in constant time. The token is never logged.
func (s *Server) authed(h http.HandlerFunc) http.HandlerFunc {
	want := []byte(s.man.Token)
	return func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized"}})
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) badRequest(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "contract", "detail": err.Error()}})
}

// infraResponse writes a typed infra error. saturated and killed are 503 + Retry-After; setup
// and job_timeout are 200 with Result.Infra (the job reached the runner and failed there).
func (s *Server) infraResponse(w http.ResponseWriter, kind runnerapi.InfraKind, detail string, retry int) {
	res := &runnerapi.Result{Infra: runnerapi.NewInfra(kind, detail), Versions: s.baseVersions()}
	code := http.StatusOK
	if kind == runnerapi.InfraSaturated || kind == runnerapi.InfraKilled {
		code = http.StatusServiceUnavailable
		w.Header().Set("Retry-After", strconv.Itoa(retry))
	}
	writeJSON(w, code, res)
}

func (s *Server) baseVersions() runnerapi.Versions {
	return runnerapi.Versions{Runner: s.man.Version, ImageDigest: s.cfg.ImageDigest, CPUModel: s.man.CPUModel, BootEpoch: s.man.BootEpoch}
}

// admit takes a free slot, or says why not.
func (s *Server) admit() (int, runnerapi.InfraKind, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.draining:
		return -1, runnerapi.InfraSaturated, retryDraining
	case s.quiet >= 0:
		return -1, runnerapi.InfraSaturated, retryQuiet
	}
	for i := range s.busy {
		if !s.busy[i] && !s.closed[i] {
			s.busy[i] = true
			s.inflight.Add(1)
			return i, "", 0
		}
	}
	return -1, runnerapi.InfraSaturated, retrySaturated
}

func (s *Server) release(slot int) {
	s.mu.Lock()
	s.busy[slot] = false
	delete(s.running, slot)
	s.mu.Unlock()
	s.inflight.Done()
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	if err := runnerapi.CheckContentType(r.Header.Get("Content-Type")); err != nil {
		s.badRequest(w, err)
		return
	}
	if kind, detail := s.ready.jobGate(s.cfg.Mode); kind != "" {
		s.infraResponse(w, kind, detail, 0)
		return
	}
	slot, kind, retry := s.admit()
	if slot < 0 {
		s.infraResponse(w, kind, "", retry)
		return
	}
	defer s.release(slot)

	job, inputs, err := runnerapi.DecodeJob(http.MaxBytesReader(w, r.Body, runnerapi.MaxBodyBytes), runnerapi.DefaultCaps())
	if err != nil {
		s.badRequest(w, err)
		return
	}
	if err := runnerapi.ValidateJob(job); err != nil {
		s.badRequest(w, err)
		return
	}
	pe, ok := s.profiles[job.Profile]
	switch {
	case !ok:
		s.badRequest(w, errors.New("unknown profile "+job.Profile))
		return
	case !pe.prof.Accepts(job.Harness):
		s.badRequest(w, errors.New("profile "+job.Profile+" does not take harness "+job.Harness))
		return
	case len(job.Tests) > 0 || job.Count > 0:
		s.badRequest(w, errors.New("declared tests need the go-race/gotest@1 profiles (p-01)"))
		return
	}

	// The job's context: the client's connection (a disconnect kills the job and sends no
	// Result), the runner's own 170 s backstop, and the drain's kill.
	ctx, cancel := context.WithCancelCause(r.Context())
	defer cancel(nil)
	ctx, stop := context.WithDeadlineCause(ctx, time.Now().Add(measure.JobDeadline), errJobTimeout)
	defer stop()
	s.mu.Lock()
	s.seq++
	seq := s.seq
	s.running[slot] = cancel
	s.mu.Unlock()

	res, disp := s.runJob(ctx, slot, seq, job, inputs, pe)
	s.served.Add(1)
	s.checkRotation()

	switch disp {
	case dispKilled:
		w.Header().Set("Retry-After", strconv.Itoa(retryDraining))
		writeJSON(w, http.StatusServiceUnavailable, res)
	case dispNone:
		// The client went away: the job subtree was killed and no Result is sent.
		s.log.Info("client disconnected; job killed", "slot", slot, "seq", seq)
	default:
		writeJSON(w, http.StatusOK, res)
	}
}

// checkRotation drains the front after RUNNER_MAX_JOBS jobs, after RUNNER_MAX_AGE, or at once
// after a SIGSYS (t3 §5.2); kubelet then restarts the container with a new BootEpoch.
func (s *Server) checkRotation() {
	switch {
	case s.sigsys.Load() > 0:
		s.StartDrain("sigsys")
	case s.served.Load() >= int64(s.cfg.MaxJobs):
		s.StartDrain("max_jobs")
	case time.Since(s.started) >= s.cfg.MaxAge:
		s.StartDrain("max_age")
	}
}

// markSlotClosed records a slot that failed closed and rotates the runner.
func (s *Server) markSlotClosed(slot int) {
	s.mu.Lock()
	s.closed[slot] = true
	s.mu.Unlock()
	s.StartDrain("slot_failed_closed")
}

// StartDrain stops admitting (503), lets in-flight jobs finish within DrainTimeout, answers
// killed for anything unfinished, then shuts the server down; Drained is closed at the end.
func (s *Server) StartDrain(reason string) {
	s.drainOnce.Do(func() {
		s.mu.Lock()
		s.draining = true
		s.mu.Unlock()
		s.log.Info("draining", "reason", reason, "timeout", s.cfg.DrainTimeout.String())
		go func() {
			done := make(chan struct{})
			go func() { s.inflight.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(s.cfg.DrainTimeout):
				s.mu.Lock()
				for slot, cancel := range s.running {
					s.log.Info("drain timeout; killing the job", "slot", slot)
					cancel(errKilled)
				}
				s.mu.Unlock()
				<-done
			}
			if s.http != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = s.http.Shutdown(ctx)
				cancel()
			}
			close(s.drained)
		}()
	})
}

// Drained is closed when a drain has finished.
func (s *Server) Drained() <-chan struct{} { return s.drained }

// Draining reports whether a drain started.
func (s *Server) Draining() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.draining
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	resp := runnerapi.ProfilesResponse{Profiles: []runnerapi.ProfileInfo{}, Runner: s.man.Version, BootEpoch: s.man.BootEpoch, Mode: string(s.cfg.Mode)}
	if st, err := s.backend.Stats(r.Context()); err == nil {
		resp.CanaryMedian = int64(st.CanaryMedianUs)
	}
	for _, pm := range s.man.Profiles {
		pe, ok := s.profiles[pm.Name]
		if !ok {
			continue
		}
		resp.Profiles = append(resp.Profiles, runnerapi.ProfileInfo{
			Profile: pm.Name, Toolchain: pm.Toolchain, ProfileSHA256: pm.ProfileSHA256, ImageDigest: s.cfg.ImageDigest,
			Harnesses: append([]string{}, pe.prof.Harnesses...), TLMultiplier: 1, MemBaselineKB: pm.BaselineBytes / 1024,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.backend.Stats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{"code": "spawner"}})
		return
	}
	s.mu.Lock()
	busy := s.busy
	draining := s.draining
	s.mu.Unlock()
	resp := runnerapi.StatsResponse{
		RunnerMemoryCurrent: int64(st.RunnerMemory), ContainerOOMKills: int64(st.OOMKills), ContainerOOMKillsLocal: int64(st.OOMKillsLocal),
		JobsServed: s.served.Load(), SigsysCount: s.sigsys.Load(), BootEpoch: s.man.BootEpoch, Mode: string(s.cfg.Mode), Draining: draining,
	}
	for i := 0; i < ipc.Slots; i++ {
		resp.Slots = append(resp.Slots, runnerapi.SlotStats{Slot: i, Busy: busy[i], MemoryCurrent: int64(st.MemoryCurrent[i]),
			PidsCurrent: int64(st.PidsCurrent[i]), NrDyingDescendants: int64(st.NrDying[i])})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := s.backend.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "ipc_unanswered"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ok, report := s.ready.report(s.cfg.Mode, time.Now())
	if s.Draining() {
		ok = false
		report["draining"] = "true"
	}
	code := http.StatusOK
	status := "ok"
	if !ok {
		code, status = http.StatusServiceUnavailable, "unready"
	}
	writeJSON(w, code, map[string]any{"status": status, "mode": s.cfg.Mode, "checks": report})
}

// Serve runs the HTTP server on cfg.Addr until a drain shuts it down.
func (s *Server) Serve(ctx context.Context) error {
	s.http = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// Bodies are ≤ ~17 MiB on the cluster network; a job answers within 170 s.
		ReadTimeout:  60 * time.Second,
		WriteTimeout: measure.JobDeadline + 30*time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go s.ready.loop(ctx, s)
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
