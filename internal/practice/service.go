package practice

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// StreamPractice is practice's JetStream stream; StreamSubjects is the wildcard it
// captures (events.md). main uses these to provision the stream on the NATS
// publisher.
const StreamPractice = "XLEARN_PRACTICE"

// StreamSubjects are the subjects the XLEARN_PRACTICE stream captures.
var StreamSubjects = []string{"xlearn.practice.*"}

// Service is the practice HTTP application: the guided-flow endpoints (state, start,
// reveal, outcome) plus the k8s probes. It verifies the gateway-minted JWT on every
// user route (ADR-0006) and derives the account id from the token subject.
type Service struct {
	store    store.Store
	verifier auth.Verifier
	log      *slog.Logger
	health   *health.Handler
}

// NewService wires the practice application. verifier checks gateway-minted JWTs.
func NewService(st store.Store, verifier auth.Verifier, log *slog.Logger) *Service {
	return &Service{
		store:    st,
		verifier: verifier,
		log:      log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
	}
}

// Handler builds practice's HTTP routes (Go 1.22+ method+pattern mux). All user
// routes verify the gateway-minted JWT; the account is the token subject.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	// Per-user guided-flow routes (proxied through the gateway BFF under /xlearn/api).
	mux.Handle("GET /state", s.requireJWT(http.HandlerFunc(s.handleListStates)))
	mux.Handle("GET /state/{problemId}", s.requireJWT(http.HandlerFunc(s.handleGetState)))
	mux.Handle("POST /problems/{id}/attempt/start", s.requireJWT(http.HandlerFunc(s.handleStartAttempt)))
	mux.Handle("POST /problems/{id}/reveal", s.requireJWT(http.HandlerFunc(s.handleReveal)))
	mux.Handle("POST /problems/{id}/outcome", s.requireJWT(http.HandlerFunc(s.handleOutcome)))

	return mux
}

// NewOutboxRelay builds the practice outbox relay over pub (a JetStream publisher in
// prod, the log publisher in local dev). Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log)
}

// outboxSource adapts the practice store to events.OutboxSource.
type outboxSource struct{ st store.Store }

func (o outboxSource) ListUnsent(ctx context.Context, limit int32) ([]events.Event, error) {
	rows, err := o.st.ListUnsentOutbox(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]events.Event, 0, len(rows))
	for _, r := range rows {
		out = append(out, events.Event{ID: r.EventID, Subject: r.Subject, Data: r.Payload})
	}
	return out, nil
}

func (o outboxSource) MarkSent(ctx context.Context, eventID string) error {
	return o.st.MarkOutboxSent(ctx, eventID)
}
