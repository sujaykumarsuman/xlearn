package front

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
	"github.com/sujaykumarsuman/xlearn/internal/runner"
	"github.com/sujaykumarsuman/xlearn/internal/runner/ipc"
)

// The /readyz canaries (t3 §5.3), re-run every readyEvery; /readyz is 200 only when every
// canary passed within readyFresh. Egress is probed from the front (it shares the pod's
// network namespace); the rest runs in the spawner, the process that holds capabilities.
const (
	readyEvery   = 30 * time.Second
	readyFresh   = 60 * time.Second
	egressTimout = time.Second
)

// EgressTargets must all be unreachable: kube-dns, the apiserver VIP and the internet.
var EgressTargets = []string{"10.43.0.10:53", "10.43.0.1:443", "1.1.1.1:443"}

var probeNames = []struct {
	bit      uint32
	name     string
	security bool
}{
	{ipc.ProbeAppArmor, "apparmor_label", true},
	{ipc.ProbeUIDMap, "uid_map_not_identity", true},
	{ipc.ProbeCgroupfs, "cgroupfs_writable", false},
	{ipc.ProbeUserNS, "userns_denied", true},
	{ipc.ProbeFsopen, "fsopen_denied", true},
	{ipc.ProbeSCTP, "sctp_denied", true},
	{ipc.ProbeCorePattern, "core_pattern_not_pipe", true},
}

type readiness struct {
	mu        sync.Mutex
	at        time.Time
	reachable []string // egress targets that answered (a security failure)
	probe     ipc.ProbeReply
	probeErr  error
	targets   []string
}

func (r *readiness) loop(ctx context.Context, s *Server) {
	r.run(ctx, s.backend)
	t := time.NewTicker(readyEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.run(ctx, s.backend)
		}
	}
}

func (r *readiness) run(ctx context.Context, b Backend) {
	targets := r.targets
	if targets == nil {
		targets = EgressTargets
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var reachable []string
	for _, t := range targets {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			d := net.Dialer{Timeout: egressTimout}
			c, err := d.DialContext(ctx, "tcp", addr)
			if err == nil {
				c.Close()
				mu.Lock()
				reachable = append(reachable, addr)
				mu.Unlock()
			}
		}(t)
	}
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	pr, perr := b.Probe(pctx)
	cancel()
	wg.Wait()
	r.mu.Lock()
	r.at, r.reachable, r.probe, r.probeErr = time.Now(), reachable, pr, perr
	r.mu.Unlock()
}

// report is /readyz's view. Functional failures (a stale or failed probe, cgroupfs) always
// fail; security failures fail in prod and only warn in dev (RUNNER_MODE).
func (r *readiness) report(mode runner.Mode, now time.Time) (bool, map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	checks := map[string]string{}
	ok := true
	if r.at.IsZero() || now.Sub(r.at) > readyFresh {
		checks["canaries"] = "stale"
		return false, checks
	}
	if r.probeErr != nil {
		checks["spawner_probe"] = "error"
		return false, checks
	}
	secFail := func(name string) {
		if mode == runner.ModeDev {
			checks[name] = "warn"
		} else {
			checks[name] = "fail"
			ok = false
		}
	}
	for _, addr := range r.reachable {
		secFail("egress_blocked:" + addr)
	}
	for _, p := range probeNames {
		switch {
		case r.probe.Ran&p.bit == 0:
			checks[p.name] = "not_run"
			ok = false
		case r.probe.Failed&p.bit == 0:
			checks[p.name] = "pass"
		case p.security:
			secFail(p.name)
		default:
			checks[p.name] = "fail"
			ok = false
		}
	}
	return ok, checks
}

// jobGate is the prod rule: while any canary fails (or none has run), POST /v1/jobs answers
// infra: setup. dev only warns.
func (r *readiness) jobGate(mode runner.Mode) (runnerapi.InfraKind, string) {
	ok, _ := r.report(mode, time.Now())
	if ok {
		return "", ""
	}
	if mode == runner.ModeDev {
		// dev refuses only on functional failures.
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.at.IsZero() || r.probeErr != nil || r.probe.Failed&ipc.ProbeCgroupfs != 0 {
			return runnerapi.InfraSetup, "readiness canaries not passing"
		}
		return "", ""
	}
	return runnerapi.InfraSetup, "readiness canaries not passing"
}
