//go:build linux && runner_it

package it

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/runnerapi"
)

func rawPost(t *testing.T, ct, token string, body []byte) (int, http.Header, []byte) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/jobs", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", ct)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	return resp.StatusCode, resp.Header, b.Bytes()
}

func encode(t *testing.T, job *runnerapi.Job, in [][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := runnerapi.EncodeJob(&b, job, in); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestContract401And400(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "echo", "testgo@0", "x")
	body := encode(t, job, in)
	for name, tok := range map[string]string{"no token": "", "bad token": "not-the-token-0123456789"} {
		if code, _, b := rawPost(t, runnerapi.ContentTypeJob, tok, body); code != http.StatusUnauthorized {
			t.Errorf("%s: %d %s", name, code, b)
		}
	}
	for name, path := range map[string]string{"profiles": "/v1/profiles", "stats": "/v1/stats"} {
		if code, _ := get(t, path, false); code != http.StatusUnauthorized {
			t.Errorf("%s without a token: %d", name, code)
		}
	}

	bad := map[string]struct {
		ct   string
		body []byte
	}{
		"wrong content type": {"application/json", body},
		"stream v2":          {runnerapi.MediaTypeJob + "; v=2", body},
		"unknown field":      {runnerapi.ContentTypeJob, frame([]byte(`{"id":"x","cost":1}`))},
		"9 MiB case refused from its prefix": {runnerapi.ContentTypeJob, func() []byte {
			j, _ := newJob(t, "echo", "testgo@0")
			j.Cases = []runnerapi.CaseInput{{OpaqueID: "big", Group: runnerapi.GroupPerf, Size: 9 << 20}}
			b := frame(mustJSON(t, j))
			var h [4]byte
			binary.BigEndian.PutUint32(h[:], 9<<20)
			return append(b, h[:]...)
		}()},
	}
	for _, mut := range []struct {
		name string
		f    func(j *runnerapi.Job)
	}{
		{"unknown profile", func(j *runnerapi.Job) { j.Profile = "cobol@85" }},
		{"harness the profile doesn't take", func(j *runnerapi.Job) { j.Harness = "func-json@1" }},
		{"declared tests before p-01", func(j *runnerapi.Job) { j.Tests = []runnerapi.TestSpec{{Name: "TestX"}} }},
		{"bad file name", func(j *runnerapi.Job) { j.Files[0].Path = "../main.go" }},
		{"limits out of range", func(j *runnerapi.Job) { j.Limits.Case.CPUms = 60_000 }},
	} {
		j, in := newJob(t, "echo", "testgo@0", "x")
		mut.f(j)
		b := frame(mustJSON(t, j))
		for _, x := range in {
			b = append(b, frame(x)...)
		}
		bad[mut.name] = struct {
			ct   string
			body []byte
		}{runnerapi.ContentTypeJob, b}
	}
	for name, c := range bad {
		t.Run(name, func(t *testing.T) {
			code, _, b := rawPost(t, c.ct, testToken, c.body)
			if code != http.StatusBadRequest {
				t.Errorf("want 400, got %d %s", code, b)
			}
		})
	}
	assertClean(t)
}

func frame(b []byte) []byte {
	out := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(out, uint32(len(b)))
	copy(out[4:], b)
	return out
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jsonEncode(&b, v); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(b.Bytes())
}

// longJob spins n cases for ms milliseconds each (TL above it).
func longJob(t *testing.T, n, ms int) (*runnerapi.Job, [][]byte) {
	job, in := newJob(t, "spin", "testgo@0", repeat(itoa(ms), n)...)
	job.Limits.Case.CPUms = int64(ms + 2000)
	return job, in
}

// waitBusy waits until n slots report busy.
func waitBusy(t *testing.T, n int) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		st := getJSON[runnerapi.StatsResponse](t, "/v1/stats")
		busy := 0
		for _, s := range st.Slots {
			if s.Busy {
				busy++
			}
		}
		if busy >= n {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%d slots never became busy", n)
}

// TestThirdJobIs503: two slots; a third concurrent job gets 503 + Retry-After +
// {"infra":{"kind":"saturated"}} — never a verdict.
func TestThirdJobIs503(t *testing.T) {
	ensure(t)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, in := longJob(t, 2, 2500)
			res := run(t, job, in)
			for _, c := range res.Cases {
				if c.Term != runnerapi.TermOK {
					t.Errorf("filler: %+v", c)
				}
			}
		}()
	}
	waitBusy(t, 2)
	job, in := newJob(t, "echo", "testgo@0", "x")
	r, err := post(context.Background(), t, job, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.code != http.StatusServiceUnavailable || r.header.Get("Retry-After") == "" ||
		r.res == nil || r.res.Infra == nil || r.res.Infra.Kind != runnerapi.InfraSaturated {
		t.Errorf("third job: %d %v %s", r.code, r.header, r.body)
	}
	if !bytes.Contains(r.body, []byte(`"infra":{"kind":"saturated"}`)) {
		t.Errorf("saturated body shape: %s", r.body)
	}
	wg.Wait()
	assertClean(t)
}

// TestDisconnectKills: when judge drops the connection, the job subtree is cgroup.kill'ed and no
// Result is sent; the runner stays healthy.
func TestDisconnectKills(t *testing.T) {
	ensure(t)
	job, in := newJob(t, "spin", "testgo@0", "")
	job.Limits.Case.CPUms = 10_000
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := post(ctx, t, job, in)
		errc <- err
	}()
	waitBusy(t, 1)
	time.Sleep(time.Second)
	if len(jobUIDProcs()) == 0 {
		t.Fatal("the spin case isn't running")
	}
	cancel()
	if err := <-errc; err == nil {
		t.Error("the client got a response after disconnecting")
	}
	assertClean(t)
	deadline := time.Now().Add(10 * time.Second)
	for {
		st := getJSON[runnerapi.StatsResponse](t, "/v1/stats")
		if !st.Slots[0].Busy && !st.Slots[1].Busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("slots still busy after the disconnect: %+v", st.Slots)
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, in = newJob(t, "echo", "testgo@0", "after")
	if res := run(t, job, in); res.Cases[0].Term != runnerapi.TermOK {
		t.Errorf("runner unhealthy after a disconnect: %+v", res.Cases[0])
	}
}

// TestDrainKilled: SIGTERM stops admissions, gives in-flight jobs the drain timeout (5 s here,
// 60 s in the pod), answers killed (503) for the rest, and exits 0.
func TestDrainKilled(t *testing.T) {
	p := ensure(t)
	job, in := longJob(t, 6, 8000)
	type out struct {
		r   *response
		err error
	}
	done := make(chan out, 1)
	go func() {
		r, err := post(context.Background(), t, job, in)
		done <- out{r, err}
	}()
	waitBusy(t, 1)
	time.Sleep(time.Second)
	t0 := time.Now()
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	// A new job during the drain is refused.
	time.Sleep(200 * time.Millisecond)
	j2, in2 := newJob(t, "echo", "testgo@0", "x")
	if r, err := post(context.Background(), t, j2, in2); err == nil && r.code != http.StatusServiceUnavailable {
		t.Errorf("admitted during the drain: %d %s", r.code, r.body)
	}
	o := <-done
	if o.err != nil {
		t.Fatal(o.err)
	}
	if o.r.code != http.StatusServiceUnavailable || o.r.res == nil || o.r.res.Infra == nil || o.r.res.Infra.Kind != runnerapi.InfraKilled {
		t.Errorf("drained job: %d %s", o.r.code, o.r.body)
	}
	metric(t, "drain_to_killed_ms", time.Since(t0).Milliseconds())
	if code := p.waitExit(t, 30*time.Second); code != 0 {
		t.Errorf("exit code after the drain: %d", code)
	}
	assertClean(t)
}

// TestSpawnPathCgroupProcs: the fallback spawn path (cgroup.procs write + sync pipe) works too
// (t3 §16.1: both work on 6.8; CLONE_INTO_CGROUP is the default).
func TestSpawnPathCgroupProcs(t *testing.T) {
	restart(t, "RUNNER_SPAWN_PATH=cgroup_procs")
	defer restart(t)
	job, in := newJob(t, "echo", "testgo@0", "via-procs")
	if res := run(t, job, in); res.Cases[0].Term != runnerapi.TermOK || string(res.Cases[0].Output) != "via-procs" {
		t.Errorf("echo: %+v", res.Cases[0])
	}
	job, in = newJob(t, "balloon", "testgo@0", "", "")
	for _, c := range run(t, job, in).Cases {
		if c.Term != runnerapi.TermMLE {
			t.Errorf("balloon via cgroup.procs: %+v", c)
		}
	}
	job, in = newJob(t, "whoami", "testgo-open@0", "/job")
	if c := run(t, job, in).Cases[0]; c.Term != runnerapi.TermOK || !strings.Contains(string(c.Output), `"cap_eff":0`) {
		t.Errorf("whoami via cgroup.procs: %+v %s", c, c.Output)
	}
	assertClean(t)
}
