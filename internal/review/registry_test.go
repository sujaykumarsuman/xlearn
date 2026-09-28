package review

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// review's stream and durable constants resolve in topology.go (the single source of
// truth), with the filters and owner the code uses.
func TestTopologyDeclaresReview(t *testing.T) {
	s := events.MustStream(StreamReview)
	if s.Owner != ServiceName || !slices.Equal(StreamSubjects, s.Subjects) {
		t.Fatalf("XLEARN_REVIEW owner %q subjects %v; code says %q %v", s.Owner, s.Subjects, ServiceName, StreamSubjects)
	}
	for _, c := range []struct{ stream, durable, filter string }{
		{StreamPractice, DurableName, PracticeSubjectFilter},
		{StreamReview, NotificationsDurable, RevisionDueSubjectFilter},
	} {
		d, ok := events.LookupDurable(c.stream, c.durable)
		if !ok || d.Filter != c.filter || d.Service != ServiceName {
			t.Errorf("durable %s/%s: declared=%v filter=%q service=%q; code filter %q", c.stream, c.durable, ok, d.Filter, d.Service, c.filter)
		}
	}
}

// Every subject constant review produces is in its stream's Emits, and every
// practice subject it consumes is one practice emits.
func TestReviewSubjectsInRegistry(t *testing.T) {
	own := events.MustStream(StreamReview).Emits
	for _, s := range []string{store.SubjectRevisionScheduled, store.SubjectRevisionDue, store.SubjectMistakeOpened, store.SubjectMistakeClosed} {
		if !slices.Contains(own, s) {
			t.Errorf("review emits %s, but XLEARN_REVIEW's Emits %v lacks it", s, own)
		}
	}
	practice := events.MustStream(StreamPractice).Emits
	for _, s := range []string{store.SubjectProblemSolved, store.SubjectSolutionRevealedEarly} {
		if !slices.Contains(practice, s) {
			t.Errorf("review consumes %s, which practice doesn't emit (%v)", s, practice)
		}
	}
}

// The practice consumer: every Handles subject reaches a store method (never the
// default branch), every Ignores subject is acked quietly, and an unlisted subject is
// acked with an ERROR (ADR-0035 §1.1 — no silent ack).
func TestPracticeConsumerSubjectRegistry(t *testing.T) {
	d, _ := events.LookupDurable(StreamPractice, DurableName)
	var calls []string
	st := &fakeStore{
		problemSolved: func(_ context.Context, _, _, _, _, _ string, _ bool, _ time.Time) (int, error) {
			calls = append(calls, store.SubjectProblemSolved)
			return 0, nil
		},
		revealedEarly: func(_ context.Context, _, _, _, _ string, _ time.Time) (int, error) {
			calls = append(calls, store.SubjectSolutionRevealedEarly)
			return 0, nil
		},
	}
	var logs bytes.Buffer
	h := &practiceHandler{store: st, log: slog.New(slog.NewJSONHandler(&logs, nil))}
	drive := func(subject string) {
		t.Helper()
		data := envJSON(t, "evt-"+subject, subject, "acct-1", "2026-09-21T12:00:00Z", map[string]any{"problem_id": "7", "outcome": "clean", "first_solve": true})
		if err := h.Handle(context.Background(), events.Event{Subject: subject, Data: data}); err != nil {
			t.Fatalf("handle %s: %v", subject, err)
		}
	}

	for _, s := range d.Handles {
		drive(s)
	}
	if !slices.Equal(calls, d.Handles) {
		t.Fatalf("handled subjects reached %v, want every Handles subject %v", calls, d.Handles)
	}
	for _, s := range d.Ignores {
		drive(s)
	}
	if len(calls) != len(d.Handles) || strings.Contains(logs.String(), "unlisted subject") {
		t.Fatalf("an Ignores subject reached the store or logged: calls %v logs %s", calls, logs.String())
	}

	drive("xlearn.practice.brand_new")
	if !strings.Contains(logs.String(), `"level":"ERROR"`) || !strings.Contains(logs.String(), "unlisted subject") ||
		!strings.Contains(logs.String(), "xlearn.practice.brand_new") {
		t.Fatalf("an unlisted subject must log ERROR: %s", logs.String())
	}
}

// The service binds both durables with the dead-letter sink.
func TestReviewSubscriptionsPassDeadLetterSink(t *testing.T) {
	st := &fakeStore{}
	svc := NewService(st, &fakeVerifier{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	rec := &recordingConsumer{}
	if _, err := svc.StartConsumers(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartNotifications(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.configs) != 2 {
		t.Fatalf("%d subscriptions", len(rec.configs))
	}
	for i, sc := range rec.configs {
		if sc.DeadLetter != st {
			t.Errorf("subscription %d (%s) has no dead-letter sink", i, rec.durables[i])
		}
	}
}

// recordingConsumer captures Subscribe calls and their resolved options.
type recordingConsumer struct {
	durables []string
	configs  []events.SubscribeConfig
}

func (r *recordingConsumer) Subscribe(_ context.Context, durable, _ string, _ events.Handler, opts ...events.SubscribeOption) (events.Subscription, error) {
	var sc events.SubscribeConfig
	for _, o := range opts {
		o(&sc)
	}
	r.durables = append(r.durables, durable)
	r.configs = append(r.configs, sc)
	return nopSub{}, nil
}

type nopSub struct{}

func (nopSub) Stop() {}
