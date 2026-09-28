package events

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

// runJetStream starts an in-process NATS server with JetStream (the module's 2.14
// server) and returns its client URL. mod tweaks the options (auth).
func runJetStream(t *testing.T, mod func(*server.Options)) string {
	t.Helper()
	opts := &server.Options{
		Host:               "127.0.0.1",
		Port:               -1,
		JetStream:          true,
		StoreDir:           t.TempDir(),
		JetStreamMaxStore:  5 * GiB, // prod max_file_store (../infra messaging release)
		JetStreamMaxMemory: 64 * MiB,
		NoLog:              true,
		NoSigs:             true,
	}
	if mod != nil {
		mod(opts)
	}
	s, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("nats server: %v", err)
	}
	go s.Start()
	if !s.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(func() { s.Shutdown(); s.WaitForShutdown() })
	return s.ClientURL()
}

func clearClientEnv(t *testing.T) {
	t.Setenv(EnvNkeySeedFile, "")
	t.Setenv(EnvInboxPrefix, "")
}

// The v1.6.0 rollout path: the live v1 streams exist with NO limits; the owning
// service's ensureStream (CreateOrUpdateStream) must apply the table's limits.
func TestEnsureStreamAppliesLimitsToLiveV1Stream(t *testing.T) {
	clearClientEnv(t)
	url := runJetStream(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, _ := jetstream.New(nc)
	// v1's hard-coded config (nats.go before mi-05): unlimited.
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name: StreamPractice, Subjects: []string{"xlearn.practice.*"},
		Storage: jetstream.FileStorage, Duplicates: dedupeWindow,
	}); err != nil {
		t.Fatalf("create v1 stream: %v", err)
	}
	if _, err := js.Publish(ctx, "xlearn.practice.problem_solved", []byte(`{"event_id":"old"}`)); err != nil {
		t.Fatal(err)
	}

	log, _ := bufLogger()
	pub, err := NewNatsPublisher(ctx, "practice", url, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer pub.Close()
	if err := pub.ensureStream(ctx); err != nil {
		t.Fatalf("ensureStream: %v", err)
	}

	info := streamInfo(ctx, t, js, StreamPractice)
	if info.Config.MaxBytes != 1*GiB || info.Config.Discard != jetstream.DiscardNew || info.Config.MaxAge != 0 {
		t.Fatalf("limits not applied: max_bytes=%d discard=%s max_age=%s",
			info.Config.MaxBytes, info.Config.Discard, info.Config.MaxAge)
	}
	if info.State.Msgs != 1 {
		t.Fatalf("existing messages = %d, the update must keep them", info.State.Msgs)
	}

	// A fresh stream with MaxAge + Discard=Old (XLEARN_JUDGE) is created from the table.
	jpub, err := NewNatsPublisher(ctx, "judge", url, StreamJudge, log)
	if err != nil {
		t.Fatal(err)
	}
	defer jpub.Close()
	if err := jpub.ensureStream(ctx); err != nil {
		t.Fatal(err)
	}
	ji := streamInfo(ctx, t, js, StreamJudge)
	if ji.Config.MaxBytes != 512*MiB || ji.Config.MaxAge != 14*24*time.Hour || ji.Config.Discard != jetstream.DiscardOld {
		t.Fatalf("judge stream config = %+v", ji.Config)
	}
}

func streamInfo(ctx context.Context, t *testing.T, js jetstream.JetStream, name string) *jetstream.StreamInfo {
	t.Helper()
	s, err := js.Stream(ctx, name)
	if err != nil {
		t.Fatalf("stream %s: %v", name, err)
	}
	info, err := s.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestConstructorsRefuseUndeclared(t *testing.T) {
	clearClientEnv(t)
	url := runJetStream(t, nil)
	ctx := context.Background()
	log, _ := bufLogger()

	if _, err := NewNatsPublisher(ctx, "practice", url, "XLEARN_NOPE", log); err == nil {
		t.Error("publisher on an undeclared stream must fail")
	}
	if _, err := NewNatsPublisher(ctx, "review", url, StreamPractice, log); err == nil {
		t.Error("a non-owner publisher must fail")
	}
	if _, err := NewNatsConsumer(ctx, "review", url, "XLEARN_NOPE", log); err == nil {
		t.Error("consumer on an undeclared stream must fail")
	}
	cons, err := NewNatsConsumer(ctx, "review", url, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer cons.Close()
	h := handlerFunc(func(context.Context, Event) error { return nil })
	for _, c := range []struct{ durable, filter string }{
		{"judge", "xlearn.practice.*"},      // not declared yet
		{"review", "xlearn.practice.>"},     // declared, wrong filter
		{"assessment", "xlearn.practice.*"}, // declared, another service's
	} {
		if _, err := cons.Subscribe(ctx, c.durable, c.filter, h); err == nil {
			t.Errorf("Subscribe(%s, %s) must be refused", c.durable, c.filter)
		}
	}
}

// Dead letter over a real JetStream: a handler that always fails is delivered
// MaxDeliver times, then recorded through the sink and terminated (nothing pending).
func TestDeadLetterOverJetStream(t *testing.T) {
	clearClientEnv(t)
	url := runJetStream(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	log, _ := bufLogger()

	pub, err := NewNatsPublisher(ctx, "practice", url, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer pub.Close()
	data, _ := json.Marshal(map[string]any{"event_id": "evt-poison", "subject": "xlearn.practice.problem_solved"})
	if err := pub.Publish(ctx, Event{ID: "evt-poison", Subject: "xlearn.practice.problem_solved", Data: data}); err != nil {
		t.Fatal(err)
	}

	cons, err := NewNatsConsumer(ctx, "review", url, StreamPractice, log)
	if err != nil {
		t.Fatal(err)
	}
	defer cons.Close()
	var calls atomic.Int32
	sink := &fakeSink{}
	sub, err := cons.Subscribe(ctx, "review", "xlearn.practice.*",
		handlerFunc(func(context.Context, Event) error { calls.Add(1); return context.DeadlineExceeded }),
		WithDeadLetter(sink), WithMaxDeliver(2))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Stop()

	waitFor(t, 10*time.Second, func() bool { return len(sink.letters()) == 1 })
	dl := sink.letters()[0]
	if dl.EventID != "evt-poison" || dl.Durable != "review" || dl.ErrClass != ErrClassTimeout || dl.StreamSeq != 1 {
		t.Fatalf("dead letter = %+v", dl)
	}
	time.Sleep(3 * testBackoff) // no further delivery after Term
	if n := calls.Load(); n != 2 {
		t.Fatalf("handler called %d times, want MaxDeliver=2", n)
	}
	c, err := cons.js.Consumer(ctx, StreamPractice, "review")
	if err != nil {
		t.Fatal(err)
	}
	ci, err := c.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ci.NumAckPending != 0 || ci.NumPending != 0 {
		t.Fatalf("after Term: ack pending %d, pending %d", ci.NumAckPending, ci.NumPending)
	}
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Dial's client options: connection names, the nkey seed, the default inbox prefix
// under a seed, and the ErrorHandler logging a permission violation at ERROR.
func TestDialClientOptions(t *testing.T) {
	kp, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	pk, _ := kp.PublicKey()
	seed, _ := kp.Seed()
	seedFile := filepath.Join(t.TempDir(), "seed")
	if err := os.WriteFile(seedFile, seed, 0o600); err != nil {
		t.Fatal(err)
	}
	url := runJetStream(t, func(o *server.Options) {
		o.Nkeys = []*server.NkeyUser{{
			Nkey: pk,
			Permissions: &server.Permissions{
				Publish:   &server.SubjectPermission{Allow: []string{"ok.>"}},
				Subscribe: &server.SubjectPermission{Allow: []string{"_INBOX_practice.>"}},
			},
		}}
	})
	ctx := context.Background()

	t.Run("missing seed fails closed", func(t *testing.T) {
		t.Setenv(EnvNkeySeedFile, filepath.Join(t.TempDir(), "absent"))
		t.Setenv(EnvInboxPrefix, "")
		log, _ := bufLogger()
		if _, err := Dial(ctx, "practice", url, log); err == nil {
			t.Fatal("a missing seed must be an error")
		}
		if _, err := NewNatsPublisher(ctx, "practice", url, StreamPractice, log); err == nil {
			t.Fatal("the publisher must surface the seed error")
		}
		if _, err := NewNatsConsumer(ctx, "review", url, StreamPractice, log); err == nil {
			t.Fatal("the consumer must surface the seed error")
		}
	})

	t.Run("bad prefix fails closed", func(t *testing.T) {
		t.Setenv(EnvNkeySeedFile, "")
		t.Setenv(EnvInboxPrefix, "bad.")
		log, _ := bufLogger()
		if _, err := Dial(ctx, "practice", url, log); err == nil {
			t.Fatal("an invalid inbox prefix must be an error")
		}
	})

	t.Run("seed, default prefix, name, error handler", func(t *testing.T) {
		t.Setenv(EnvNkeySeedFile, seedFile)
		t.Setenv(EnvInboxPrefix, "")
		log, buf := bufLogger()
		nc, err := Dial(ctx, "practice", url, log, nats.Name(connName("practice", StreamPractice, "pub")))
		if err != nil {
			t.Fatal(err)
		}
		defer nc.Close()
		waitFor(t, 5*time.Second, nc.IsConnected)
		if nc.Opts.InboxPrefix != "_INBOX_practice" {
			t.Fatalf("inbox prefix %q, want _INBOX_practice", nc.Opts.InboxPrefix)
		}
		if nc.Opts.Name != "xlearn-practice:XLEARN_PRACTICE:pub" {
			t.Fatalf("connection name %q", nc.Opts.Name)
		}
		if !strings.HasPrefix(nc.NewInbox(), "_INBOX_practice.") {
			t.Fatalf("inbox %q", nc.NewInbox())
		}
		// Allowed publish + request-reply on the own inbox work.
		if err := nc.Publish("ok.x", nil); err != nil {
			t.Fatal(err)
		}
		// A denied publish fires the ErrorHandler → ERROR with the subject.
		if err := nc.Publish("denied.x", nil); err != nil {
			t.Fatal(err)
		}
		_ = nc.Flush()
		waitFor(t, 5*time.Second, func() bool { return strings.Contains(buf.String(), "nats permission violation") })
		out := buf.String()
		if !strings.Contains(out, `"level":"ERROR"`) || !strings.Contains(out, `"subject":"denied.x"`) {
			t.Fatalf("permission violation log = %s", out)
		}
	})

	t.Run("no env is the v1 anonymous connection", func(t *testing.T) {
		clearClientEnv(t)
		url := runJetStream(t, nil)
		log, _ := bufLogger()
		nc, err := Dial(ctx, "review", url, log)
		if err != nil {
			t.Fatal(err)
		}
		defer nc.Close()
		waitFor(t, 5*time.Second, nc.IsConnected)
		if nc.Opts.InboxPrefix != "" || nc.Opts.Nkey != "" {
			t.Fatalf("no env must leave prefix/nkey unset: %q %q", nc.Opts.InboxPrefix, nc.Opts.Nkey)
		}
		if nc.Opts.Name != "xlearn-review" {
			t.Fatalf("default name %q", nc.Opts.Name)
		}
	})
}
