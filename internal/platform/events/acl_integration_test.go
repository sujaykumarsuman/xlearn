//go:build natsacl

package events

// The NATS-auth integration test (mi-05, ADR-0035 §2 "Verification before any seed is
// mounted"): boot NATS 2.14 — the compose image, the prod minor — with the authorization
// block rendered from topology.go for throwaway nkeys, then drive the REAL client code
// (Dial, NewNatsPublisher, NewNatsConsumer, Subscribe) through every allowed and denied
// case, for all six services plus ops, and through the three legacy stages (N1 allow,
// N3 deny, N4 none). Run it with `make nats-acl-test` (needs docker); CI runs it as the
// `nats-acl` job. It must pass before mi-06 mounts any seed (N2).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const aclComposeFile = "../../../deploy/local/nats-acl.compose.yml"

// aclServerBlock is the non-auth part of the test server's conf: JetStream on a tmpfs
// with prod's max_file_store (5 Gi), so the topology.go budget reserves exactly as on
// the cluster.
const aclServerBlock = `# natsacl test server (deploy/local/nats-acl.compose.yml)
server_name: "xlearn-natsacl"
listen: "0.0.0.0:4222"
http: "0.0.0.0:8222"
jetstream {
  store_dir: "/data"
  max_file_store: 5368709120
  max_memory_store: 67108864
}
`

type aclHarness struct {
	dir, confPath       string
	pub                 map[string]string
	seedPath            map[string]string
	clientURL, monitor  string
	composeArgs, runEnv []string
}

func newACLHarness(t *testing.T) *aclHarness {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("make nats-acl-test needs docker: %v", err)
	}
	compose, err := filepath.Abs(aclComposeFile)
	if err != nil {
		t.Fatal(err)
	}
	clientPort := envOr("NATSACL_CLIENT_PORT", "14222")
	monitorPort := envOr("NATSACL_MONITOR_PORT", "18222")
	h := &aclHarness{
		dir:         t.TempDir(),
		seedPath:    map[string]string{},
		clientURL:   "nats://127.0.0.1:" + clientPort,
		monitor:     "http://127.0.0.1:" + monitorPort,
		composeArgs: []string{"compose", "-p", "xlearn-natsacl", "-f", compose},
	}
	h.confPath = filepath.Join(h.dir, "nats.conf")
	h.runEnv = append(os.Environ(), "NATSACL_CONF="+h.confPath,
		"NATSACL_CLIENT_PORT="+clientPort, "NATSACL_MONITOR_PORT="+monitorPort)

	pub, seeds := throwawayKeys(t)
	h.pub = pub
	for id, seed := range seeds {
		p := filepath.Join(h.dir, id+".nk")
		if err := os.WriteFile(p, seed, 0o600); err != nil {
			t.Fatal(err)
		}
		h.seedPath[id] = p
	}
	t.Cleanup(func() { _, _ = h.docker("down", "-v", "--remove-orphans") })
	return h
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (h *aclHarness) docker(args ...string) (string, error) {
	cmd := exec.Command("docker", append(append([]string{}, h.composeArgs...), args...)...)
	cmd.Env = h.runEnv
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// boot (re)starts the server with the render for the legacy stage: a fresh container
// and a fresh tmpfs store each time, like the one N1 restart plus the N3/N4 reloads.
func (h *aclHarness) boot(t *testing.T, mode LegacyMode) {
	t.Helper()
	a, err := RenderAuthorization(h.pub, mode)
	if err != nil {
		t.Fatal(err)
	}
	pw := make([]byte, 16)
	_, _ = rand.Read(pw)
	a.SetLegacyPassword(hex.EncodeToString(pw))
	if err := os.WriteFile(h.confPath, []byte(aclServerBlock+a.Conf()), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := h.docker("up", "-d", "--force-recreate", "--wait", "--wait-timeout", "90"); err != nil {
		logs, _ := h.docker("logs", "--no-color", "--tail", "50")
		t.Fatalf("boot nats (%s): %v\n%s\n%s", mode, err, out, logs)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(h.monitor + "/healthz?js-enabled-only=true")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("nats (%s) not healthy: %v", mode, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// as sets the client env to identity id (its seed; ops also takes the default
// _INBOX prefix, which is its only subscribe grant) — or to anonymous when id is "".
func (h *aclHarness) as(t *testing.T, id string) {
	t.Helper()
	t.Setenv(EnvNkeySeedFile, h.seedPath[id])
	t.Setenv(EnvInboxPrefix, "")
	if id == OpsIdentity {
		t.Setenv(EnvInboxPrefix, "_INBOX")
	}
}

func (h *aclHarness) monitorJSON(t *testing.T, path string, v any) {
	t.Helper()
	resp, err := http.Get(h.monitor + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

// recLog records log lines so the test can assert the Dial ErrorHandler fired.
type recLog struct {
	mu   sync.Mutex
	recs []logRec
}

type logRec struct {
	level slog.Level
	msg   string
	attrs map[string]string
}

func (r *recLog) Enabled(context.Context, slog.Level) bool { return true }
func (r *recLog) WithAttrs([]slog.Attr) slog.Handler       { return r }
func (r *recLog) WithGroup(string) slog.Handler            { return r }
func (r *recLog) Handle(_ context.Context, rec slog.Record) error {
	lr := logRec{level: rec.Level, msg: rec.Message, attrs: map[string]string{}}
	rec.Attrs(func(a slog.Attr) bool { lr.attrs[a.Key] = a.Value.String(); return true })
	r.mu.Lock()
	r.recs = append(r.recs, lr)
	r.mu.Unlock()
	return nil
}

// violations lists the subjects of every permission violation the ErrorHandler logged.
func (r *recLog) violations() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, x := range r.recs {
		if x.msg == "nats permission violation" && x.level == slog.LevelError {
			out = append(out, x.attrs["subject"])
		}
	}
	return out
}

// expectViolation waits for the ErrorHandler to log a permission violation whose
// subject has the given prefix.
func (r *recLog) expectViolation(t *testing.T, who, prefix string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		for _, s := range r.violations() {
			if strings.HasPrefix(s, prefix) {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Errorf("%s: no ErrPermissionViolation for %q (saw %v)", who, prefix, r.violations())
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func newRecLog() (*recLog, *slog.Logger) {
	r := &recLog{}
	return r, slog.New(r)
}

func ownStream(svc string) string {
	for _, s := range Streams() {
		if s.Owner == svc {
			return s.Name
		}
	}
	return ""
}

func probe(subject string) Event {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	data, _ := json.Marshal(map[string]string{"event_id": id, "subject": subject})
	return Event{ID: id, Subject: subject, Data: data}
}

func TestNATSACL(t *testing.T) {
	h := newACLHarness(t)

	t.Run("N1 allow: services, ops, legacy", func(t *testing.T) {
		h.boot(t, LegacyAllow)
		pubs, logs := testServicesAllowed(t, h)
		testServicesDenied(t, h, pubs, logs)
		testOps(t, h)
		testLegacyAllow(t, h)
	})
	t.Run("N3 deny: legacy connects, can do nothing", func(t *testing.T) {
		h.boot(t, LegacyDeny)
		testLegacyDeny(t, h)
		testSeededStillWorks(t, h)
	})
	t.Run("N4 none: anonymous refused", func(t *testing.T) {
		h.boot(t, LegacyNone)
		testLegacyNone(t, h)
		testSeededStillWorks(t, h)
	})
}

// testServicesAllowed: every service ensures its own stream and publishes its own
// subject; every declared durable is created (filtered) by its service, fetches, naks
// with delay, re-fetches and acks — through the real Subscribe/dispatch path — and
// nobody logs a single permission violation.
func testServicesAllowed(t *testing.T, h *aclHarness) (map[string]*NatsPublisher, map[string]*recLog) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pubs, logs := map[string]*NatsPublisher{}, map[string]*recLog{}
	loggers := map[string]*slog.Logger{}
	for _, svc := range ACLServices() {
		logs[svc], loggers[svc] = newRecLog()
		h.as(t, svc)
		p, err := NewNatsPublisher(ctx, svc, h.clientURL, ownStream(svc), loggers[svc])
		if err != nil {
			t.Fatalf("%s publisher: %v", svc, err)
		}
		t.Cleanup(p.Close)
		if err := p.ensureStream(ctx); err != nil {
			t.Fatalf("%s ensureStream(%s): %v", svc, ownStream(svc), err)
		}
		if err := p.Publish(ctx, probe("xlearn."+svc+".probe")); err != nil {
			t.Fatalf("%s publish own subject: %v", svc, err)
		}
		if p.Conn().Opts.InboxPrefix != "_INBOX_"+svc {
			t.Fatalf("%s inbox prefix %q", svc, p.Conn().Opts.InboxPrefix)
		}
		pubs[svc] = p
	}

	for _, d := range Durables() {
		h.as(t, d.Service)
		c, err := NewNatsConsumer(ctx, d.Service, h.clientURL, d.Stream, loggers[d.Service])
		if err != nil {
			t.Fatalf("%s consumer: %v", d.Service, err)
		}
		owner := MustStream(d.Stream).Owner
		target := probe(d.Handles[0])
		var calls, acked atomic.Int32
		handler := handlerFunc(func(_ context.Context, e Event) error {
			if e.ID != target.ID {
				return nil
			}
			if calls.Add(1) == 1 {
				return errors.New("first delivery fails: nak with delay")
			}
			acked.Add(1)
			return nil
		})
		sub, err := c.Subscribe(ctx, d.Name, d.Filter, handler, WithMaxDeliver(5))
		if err != nil {
			t.Fatalf("%s/%s subscribe: %v", d.Stream, d.Name, err)
		}
		if err := pubs[owner].Publish(ctx, target); err != nil {
			t.Fatalf("%s publish %s: %v", owner, target.Subject, err)
		}
		waitFor(t, 15*time.Second, func() bool { return acked.Load() == 1 })
		cons, err := c.js.Consumer(ctx, d.Stream, d.Name) // CONSUMER.INFO
		if err != nil {
			t.Fatalf("%s/%s consumer info: %v", d.Stream, d.Name, err)
		}
		waitFor(t, 10*time.Second, func() bool {
			ci, ierr := cons.Info(ctx)
			return ierr == nil && ci.NumAckPending == 0
		})
		sub.Stop()
		c.Close()
		t.Logf("allowed: %s created %s/%s (filter %s), fetched, nak'd with delay, acked", d.Service, d.Stream, d.Name, d.Filter)
	}

	for svc, l := range logs {
		if v := l.violations(); len(v) != 0 {
			t.Errorf("%s logged permission violations on allowed operations: %v", svc, v)
		}
	}
	return pubs, logs
}

// testServicesDenied: each service is denied (and its ErrorHandler logs
// ErrPermissionViolation) for another service's subject, another service's stream,
// purging or deleting its own stream, a durable it doesn't own, and any inbox but its own.
func testServicesDenied(t *testing.T, h *aclHarness, pubs map[string]*NatsPublisher, logs map[string]*recLog) {
	services := ACLServices()
	short := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 1500*time.Millisecond)
	}
	for i, svc := range services {
		other := services[(i+1)%len(services)]
		own, otherStream := ownStream(svc), ownStream(other)
		p, l := pubs[svc], logs[svc]
		nc, js := p.Conn(), p.js

		// Another service's subject.
		_ = nc.Publish("xlearn."+other+".probe", []byte("{}"))
		_ = nc.Flush()
		l.expectViolation(t, svc, "xlearn."+other+".probe")

		// Another service's stream.
		ctx, cancel := short()
		_, _ = js.UpdateStream(ctx, MustStream(otherStream).Config())
		cancel()
		l.expectViolation(t, svc, "$JS.API.STREAM.UPDATE."+otherStream)

		// Purge and delete its OWN stream.
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		s, err := js.Stream(ctx, own) // STREAM.INFO own: allowed
		cancel()
		if err != nil {
			t.Fatalf("%s stream info %s: %v", svc, own, err)
		}
		ctx, cancel = short()
		_ = s.Purge(ctx)
		cancel()
		l.expectViolation(t, svc, "$JS.API.STREAM.PURGE."+own)
		ctx, cancel = short()
		_ = js.DeleteStream(ctx, own)
		cancel()
		l.expectViolation(t, svc, "$JS.API.STREAM.DELETE."+own)

		// A durable it doesn't own: create it (filtered) and fetch from it.
		var foreign Durable
		for _, d := range Durables() {
			if d.Service != svc {
				foreign = d
				break
			}
		}
		ctx, cancel = short()
		_, _ = js.CreateOrUpdateConsumer(ctx, foreign.Stream, jetstream.ConsumerConfig{
			Durable: foreign.Name, FilterSubject: foreign.Filter, AckPolicy: jetstream.AckExplicitPolicy,
		})
		cancel()
		l.expectViolation(t, svc, "$JS.API.CONSUMER.CREATE."+foreign.Stream+"."+foreign.Name)
		next := "$JS.API.CONSUMER.MSG.NEXT." + foreign.Stream + "." + foreign.Name
		_ = nc.PublishRequest(next, nc.NewInbox(), []byte(`{"batch":1,"expires":500000000}`))
		_ = nc.Flush()
		l.expectViolation(t, svc, next)

		// Another service's inbox, and the shared default inbox.
		for _, subj := range []string{"_INBOX_" + other + ".>", "_INBOX.>"} {
			sub, err := nc.SubscribeSync(subj)
			if err == nil {
				_ = nc.Flush()
				l.expectViolation(t, svc, subj)
				_ = sub.Unsubscribe()
			}
		}

		// The connection survives every denial (permission errors are not fatal).
		if !nc.IsConnected() {
			t.Errorf("%s connection dropped after denials", svc)
		}
		t.Logf("denied: %s on xlearn.%s.>, UPDATE %s, PURGE/DELETE %s, %s/%s create+fetch, foreign inboxes",
			svc, other, otherStream, own, foreign.Stream, foreign.Name)
	}
}

// testOps: the break-glass identity can create and purge a scratch stream.
func testOps(t *testing.T, h *aclHarness) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l, log := newRecLog()
	h.as(t, OpsIdentity)
	nc, err := Dial(ctx, OpsIdentity, h.clientURL, log)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, _ := jetstream.New(nc)
	s, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name: "XLEARN_SCRATCH", Subjects: []string{"scratch.>"}, Storage: jetstream.FileStorage, MaxBytes: 1 * MiB,
	})
	if err != nil {
		t.Fatalf("ops create scratch: %v", err)
	}
	if err := s.Purge(ctx); err != nil {
		t.Fatalf("ops purge scratch: %v", err)
	}
	if err := js.DeleteStream(ctx, "XLEARN_SCRATCH"); err != nil {
		t.Fatalf("ops delete scratch: %v", err)
	}
	if v := l.violations(); len(v) != 0 {
		t.Fatalf("ops violations: %v", v)
	}
}

// testLegacyAllow (N1): a seedless (v1.5.2-era) client connects as `legacy` through
// no_auth_user and works end to end; /connz shows it as legacy and a seeded client as
// its own nkey.
func testLegacyAllow(t *testing.T, h *aclHarness) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	l, log := newRecLog()
	h.as(t, "")
	p, err := NewNatsPublisher(ctx, "practice", h.clientURL, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	c, err := NewNatsConsumer(ctx, "review", h.clientURL, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	target := probe("xlearn.practice.problem_solved")
	var got atomic.Bool
	sub, err := c.Subscribe(ctx, "review", "xlearn.practice.*", handlerFunc(func(_ context.Context, e Event) error {
		if e.ID == target.ID {
			got.Store(true)
		}
		return nil
	}), WithMaxDeliver(5))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Stop()
	if err := p.Publish(ctx, target); err != nil {
		t.Fatalf("anonymous publish under N1: %v", err)
	}
	waitFor(t, 15*time.Second, got.Load)
	if v := l.violations(); len(v) != 0 {
		t.Fatalf("anonymous client violations under N1: %v", v)
	}

	var connz struct {
		Connections []struct {
			Name           string `json:"name"`
			AuthorizedUser string `json:"authorized_user"`
		} `json:"connections"`
	}
	h.monitorJSON(t, "/connz?auth=true&limit=1024", &connz)
	users := map[string]string{}
	for _, cn := range connz.Connections {
		users[cn.Name] = cn.AuthorizedUser
	}
	if u := users["xlearn-practice:XLEARN_PRACTICE:pub"]; u != LegacyUser {
		t.Errorf("/connz: the seedless practice publisher is %q, want %q (N1 verify)", u, LegacyUser)
	}
	t.Logf("/connz?auth=true: %v", users)
}

// testLegacyDeny (N3): a seedless client still connects (as legacy) but every publish
// is denied.
func testLegacyDeny(t *testing.T, h *aclHarness) {
	l, log := newRecLog()
	h.as(t, "")
	nc, err := Dial(context.Background(), "practice", h.clientURL, log)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	waitFor(t, 10*time.Second, nc.IsConnected)
	for _, subj := range []string{"xlearn.practice.probe", "$JS.API.STREAM.INFO.XLEARN_PRACTICE"} {
		_ = nc.Publish(subj, []byte("{}"))
		_ = nc.Flush()
		l.expectViolation(t, "legacy", subj)
	}
	if _, err := nc.SubscribeSync("xlearn.>"); err == nil {
		_ = nc.Flush()
		l.expectViolation(t, "legacy", "xlearn.>")
	}
}

// testLegacyNone (N4): an anonymous connect is refused and /varz says auth_required.
func testLegacyNone(t *testing.T, h *aclHarness) {
	nc, err := nats.Connect(h.clientURL, nats.Timeout(3*time.Second))
	if err == nil {
		nc.Close()
		t.Fatal("N4: an anonymous connect must be refused")
	}
	if !errors.Is(err, nats.ErrAuthorization) && !strings.Contains(strings.ToLower(err.Error()), "authorization") {
		t.Fatalf("N4: anonymous connect failed with %v, want an authorization violation", err)
	}
	var varz struct {
		AuthRequired bool `json:"auth_required"`
	}
	h.monitorJSON(t, "/varz", &varz)
	if !varz.AuthRequired {
		t.Fatal("N4: /varz auth_required is false")
	}
}

// testSeededStillWorks: a seeded service is unaffected by the legacy stage.
func testSeededStillWorks(t *testing.T, h *aclHarness) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l, log := newRecLog()
	h.as(t, "practice")
	p, err := NewNatsPublisher(ctx, "practice", h.clientURL, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Publish(ctx, probe("xlearn.practice.probe")); err != nil {
		t.Fatalf("seeded practice publish: %v", err)
	}
	if v := l.violations(); len(v) != 0 {
		t.Fatalf("seeded practice violations: %v", v)
	}
}
