package assessment

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// JetStream topology (events.md / ADR-0014). assessment OWNS XLEARN_ASSESSMENT for its
// own emissions (mock_completed) and SUBSCRIBES to XLEARN_PRACTICE (xlearn.practice.*)
// and XLEARN_REVIEW (xlearn.review.*) as durable pull consumers named DurableName —
// the progress-projection consumers that upsert the coverage / mastery / heatmap /
// outcome-mix read model (S09, ADR-0018). The streams' subjects and limits and both
// durables are declared in internal/platform/events/topology.go (the single source of
// truth, ADR-0035 §1); these names must resolve there (Subscribe refuses otherwise).
const (
	// ServiceName is assessment's identity on NATS (topology owner, connection names,
	// nkey user in the rendered ACL).
	ServiceName = "assessment"

	// StreamAssessment is assessment's own stream; the relay publishes mock_completed
	// here. main passes it to the NATS publisher.
	StreamAssessment = events.StreamAssessment

	// StreamPractice / StreamReview are the streams assessment consumes for projections.
	StreamPractice = events.StreamPractice
	StreamReview   = events.StreamReview

	// PracticeSubjectFilter / ReviewSubjectFilter are the wildcards the durable
	// consumers bind to.
	PracticeSubjectFilter = "xlearn.practice.*"
	ReviewSubjectFilter   = "xlearn.review.*"

	// DurableName is assessment's durable-consumer name on each consumed stream (one
	// durable per subscribing service, per ADR-0004; distinct per stream).
	DurableName = "assessment"
)

// StreamSubjects are the subjects the XLEARN_ASSESSMENT stream captures (from the table).
var StreamSubjects = events.MustStream(StreamAssessment).Subjects

// Service is the assessment HTTP application: the mock lifecycle + rubric endpoints
// plus the k8s probes. It verifies the gateway-minted JWT on every user route
// (ADR-0006) and derives the account id from the token subject.
type Service struct {
	store    store.Store
	verifier auth.Verifier
	log      *slog.Logger
	health   *health.Handler

	// courses resolves the internal `?path=<slug>` param (m1-03): the manifests
	// compiled into the binary unless a test injects coursetest.Registry.
	courses *course.Registry
}

// NewService wires the assessment application. verifier checks gateway-minted JWTs. The
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

// Handler builds assessment's HTTP routes (Go 1.22+ method+pattern mux). Every user
// route (userRoutes) verifies the gateway-minted JWT and requires the learner role; the
// account is the token subject. The external gateway surface /xlearn/api/mocks/* maps
// onto the internal /mocks/* routes.
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

// userRoutes is assessment's per-user route table. The static /mocks/trend pattern is
// registered alongside /mocks/{id}: Go's mux prefers the more specific literal, so
// "trend" never collides with an {id}.
func (s *Service) userRoutes() []userRoute {
	return []userRoute{
		// POST /mocks, the trend and the progress reads take the optional internal
		// `?path=<slug>` (m1-03, see resolveCourse); a session by id doesn't.
		{http.MethodPost, "/mocks", s.handleStartMock},
		{http.MethodGet, "/mocks/trend", s.handleTrend},
		{http.MethodGet, "/mocks/{id}", s.handleGetMock},
		{http.MethodPost, "/mocks/{id}/score", s.handleScoreMock},

		// Progress read model (S09): the four tiles + outcome mix (summary), the
		// revision-activity heatmap, and per-problem solve quality (mastery) the gateway
		// rolls up by curriculum pattern + phase. All read the assessment projections only.
		{http.MethodGet, "/progress/summary", s.handleProgressSummary},
		{http.MethodGet, "/progress/heatmap", s.handleProgressHeatmap},
		{http.MethodGet, "/progress/mastery", s.handleProgressMastery},
	}
}

// NewOutboxRelay builds the assessment outbox relay over pub (a JetStream publisher in
// prod, the log publisher in local dev). Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher, opts ...events.RelayOption) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log, opts...)
}

// StartProjectionConsumer binds a durable pull consumer for the progress projections on
// a consumed stream (XLEARN_PRACTICE or XLEARN_REVIEW), filtered to filterSubject. The
// handler dedupes on event_id via the inbox and applies the coverage / mastery /
// heatmap / outcome-mix upserts in the same transaction (ADR-0018). It returns the
// running subscription (Stop it on shutdown).
//
// DeliverNew: both consumed streams already hold prior sprints' history (S05 practice,
// S06/S07 review). Because the projection bodies are no-op stubs this sprint, replaying
// that history has no value and would only flood the inbox — so the durable starts at
// the stream head. A later restart still resumes from its committed offset (offline
// catch-up); S09 backfills pre-existing history via a separate replay (events.md).
//
// A message that fails its last delivery is recorded in assessment.event_dead_letter
// (ADR-0035 §1.2).
func (s *Service) StartProjectionConsumer(ctx context.Context, consumer events.Consumer, filterSubject string) (events.Subscription, error) {
	h := &projectionHandler{store: s.store, log: s.log}
	return consumer.Subscribe(ctx, DurableName, filterSubject, h,
		events.WithDeliverNew(), events.WithDeadLetter(s.store))
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
