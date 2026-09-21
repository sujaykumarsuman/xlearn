package assessment

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// JetStream topology (events.md / ADR-0014). assessment OWNS XLEARN_ASSESSMENT for its
// own emissions (mock_completed) and SUBSCRIBES to XLEARN_PRACTICE (xlearn.practice.*)
// and XLEARN_REVIEW (xlearn.review.*) as durable pull consumers named DurableName —
// the S09 progress-projection consume seam (no-op handler bodies this sprint).
const (
	// StreamAssessment is assessment's own stream; the relay publishes mock_completed
	// here. main uses StreamAssessment + StreamSubjects to provision it.
	StreamAssessment = "XLEARN_ASSESSMENT"

	// StreamPractice / StreamReview are the streams assessment consumes for projections.
	StreamPractice = "XLEARN_PRACTICE"
	StreamReview   = "XLEARN_REVIEW"

	// PracticeSubjectFilter / ReviewSubjectFilter are the wildcards the durable
	// consumers bind to.
	PracticeSubjectFilter = "xlearn.practice.*"
	ReviewSubjectFilter   = "xlearn.review.*"

	// DurableName is assessment's durable-consumer name on each consumed stream (one
	// durable per subscribing service, per ADR-0004; distinct per stream).
	DurableName = "assessment"
)

// StreamSubjects are the subjects the XLEARN_ASSESSMENT stream captures.
var StreamSubjects = []string{"xlearn.assessment.*"}

// Service is the assessment HTTP application: the mock lifecycle + rubric endpoints
// plus the k8s probes. It verifies the gateway-minted JWT on every user route
// (ADR-0006) and derives the account id from the token subject.
type Service struct {
	store    store.Store
	verifier auth.Verifier
	log      *slog.Logger
	health   *health.Handler
}

// NewService wires the assessment application. verifier checks gateway-minted JWTs.
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

// Handler builds assessment's HTTP routes (Go 1.22+ method+pattern mux). The mock
// routes verify the gateway-minted JWT; the account is the token subject. The external
// gateway surface /xlearn/api/mocks/* maps onto these internal /mocks/* routes. The
// static /mocks/trend pattern is registered alongside /mocks/{id} — Go's mux prefers
// the more specific literal, so "trend" never collides with an {id}.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	mux.Handle("POST /mocks", s.requireJWT(http.HandlerFunc(s.handleStartMock)))
	mux.Handle("GET /mocks/trend", s.requireJWT(http.HandlerFunc(s.handleTrend)))
	mux.Handle("GET /mocks/{id}", s.requireJWT(http.HandlerFunc(s.handleGetMock)))
	mux.Handle("POST /mocks/{id}/score", s.requireJWT(http.HandlerFunc(s.handleScoreMock)))

	return mux
}

// NewOutboxRelay builds the assessment outbox relay over pub (a JetStream publisher in
// prod, the log publisher in local dev). Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log)
}

// StartProjectionConsumer binds a durable pull consumer for the S09 progress
// projections on a consumed stream (XLEARN_PRACTICE or XLEARN_REVIEW), filtered to
// filterSubject. The handler dedupes on event_id via the inbox and no-ops the
// projection body this sprint (S09 fills it). It returns the running subscription
// (Stop it on shutdown).
//
// DeliverNew: both consumed streams already hold prior sprints' history (S05 practice,
// S06/S07 review). Because the projection bodies are no-op stubs this sprint, replaying
// that history has no value and would only flood the inbox — so the durable starts at
// the stream head. A later restart still resumes from its committed offset (offline
// catch-up); S09 backfills pre-existing history via a separate replay (events.md).
func (s *Service) StartProjectionConsumer(ctx context.Context, consumer events.Consumer, filterSubject string) (events.Subscription, error) {
	h := &projectionHandler{store: s.store, log: s.log}
	return consumer.Subscribe(ctx, DurableName, filterSubject, h, events.WithDeliverNew())
}

// outboxSource adapts the assessment store to events.OutboxSource.
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
