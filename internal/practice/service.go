package practice

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// ServiceName is practice's identity on NATS: its stream owner name in topology.go,
// its connection names and its nkey user in the rendered ACL.
const ServiceName = "practice"

// StreamPractice is practice's JetStream stream (events.md). Its subjects and limits
// live in internal/platform/events/topology.go, the single source of truth; main
// passes the name to the NATS publisher, which provisions it from the table.
const StreamPractice = events.StreamPractice

// StreamSubjects are the subjects the XLEARN_PRACTICE stream captures (from the table).
var StreamSubjects = events.MustStream(StreamPractice).Subjects

// Service is the practice HTTP application: the guided-flow endpoints (state, start,
// reveal, outcome) plus the k8s probes. It verifies the gateway-minted JWT on every
// user route (ADR-0006) and derives the account id from the token subject.
type Service struct {
	store    store.Store
	verifier auth.Verifier
	log      *slog.Logger
	health   *health.Handler

	// courses resolves the internal `?path=<slug>` param (m1-03): the manifests
	// compiled into the binary unless a test injects coursetest.Registry.
	courses *course.Registry
}

// NewService wires the practice application. verifier checks gateway-minted JWTs. The
// course registry defaults to the embedded manifests (course.Embedded); main loads them
// first with course.LoadEmbedded so a bad manifest fails the boot, not a request.
func NewService(st store.Store, verifier auth.Verifier, log *slog.Logger) *Service {
	return &Service{
		store:    st,
		verifier: verifier,
		log:      log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
		courses: course.Embedded(),
	}
}

// WithCourses replaces the course registry `?path=` resolves against (tests inject
// coursetest.Registry for the fixture courses). Returns the service for chaining.
func (s *Service) WithCourses(r *course.Registry) *Service {
	s.courses = r
	return s
}

// Handler builds practice's HTTP routes (Go 1.22+ method+pattern mux). Every user
// route (userRoutes) verifies the gateway-minted JWT and requires the learner role;
// the account is the token subject.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	requireLearner := auth.RequireRole(s.verifier, auth.RoleLearner, auth.WithLogger(s.log))
	for _, rt := range s.userRoutes() {
		mux.Handle(rt.Method+" "+rt.Pattern, requireLearner(rt.Handler))
	}

	return mux
}

// userRoute is one per-user (JWT) route. Handler registers every userRoutes entry behind
// auth.RequireRole(learner) (ADR-0033 §12 row 5), so a route added to the table cannot
// skip the role check.
type userRoute struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

// userRoutes is practice's per-user route table (proxied through the gateway BFF under
// /xlearn/api). Problems are global ids, so the reads take no course; the three writes
// accept the optional `?path=<slug>` (m1-03, see resolveCourse).
func (s *Service) userRoutes() []userRoute {
	return []userRoute{
		{http.MethodGet, "/state", s.handleListStates},
		{http.MethodGet, "/state/{problemId}", s.handleGetState},
		{http.MethodPost, "/problems/{id}/attempt/start", s.handleStartAttempt},
		{http.MethodPost, "/problems/{id}/reveal", s.handleReveal},
		{http.MethodPost, "/problems/{id}/outcome", s.handleOutcome},
	}
}

// NewOutboxRelay builds the practice outbox relay over pub (a JetStream publisher in
// prod, the log publisher in local dev). Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher, opts ...events.RelayOption) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log, opts...)
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
