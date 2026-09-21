package review

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
	"github.com/sujaykumarsuman/xlearn/internal/review/notifications"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// JetStream topology (events.md / ADR-0014). review OWNS XLEARN_REVIEW for its own
// emissions and SUBSCRIBES to XLEARN_PRACTICE (subjects xlearn.practice.*) as a
// durable pull consumer named DurableName.
const (
	// StreamReview is review's own stream; the relay publishes revision_scheduled /
	// revision_due here. main uses StreamReview + StreamSubjects to provision it.
	StreamReview = "XLEARN_REVIEW"

	// StreamPractice is the practice stream review consumes.
	StreamPractice = "XLEARN_PRACTICE"

	// PracticeSubjectFilter is the wildcard the durable consumer binds to.
	PracticeSubjectFilter = "xlearn.practice.*"

	// DurableName is review's durable-consumer name on XLEARN_PRACTICE (one durable
	// per subscribing service, per ADR-0004).
	DurableName = "review"

	// NotificationsDurable is the in-app reminder worker's durable-consumer name on
	// review's OWN stream (XLEARN_REVIEW): a cleanly separable second consumer that
	// reads back revision_due to write reminders (ADR-0016). A distinct durable name
	// keeps it lift-and-shift ready for a future standalone notifications service.
	NotificationsDurable = "notifications"

	// RevisionDueSubjectFilter is the subject the notifications consumer binds to.
	RevisionDueSubjectFilter = "xlearn.review.revision_due"
)

// StreamSubjects are the subjects the XLEARN_REVIEW stream captures.
var StreamSubjects = []string{"xlearn.review.*"}

// Service is the review HTTP application: the Revision queue endpoints (due queue +
// auto-score) plus the k8s probes. It verifies the gateway-minted JWT on every user
// route (ADR-0006) and derives the account id from the token subject.
type Service struct {
	store    store.Store
	verifier auth.Verifier
	log      *slog.Logger
	health   *health.Handler

	// accounts resolves account timezone/study-budget for the background workers (the
	// weak-area recompute + notifications). nil in local dev / tests → UTC fallback.
	accounts accountResolver
	// weakArea recomputes the weekly weak-area on the sweep tick (always built; uses
	// accounts when set, else a UTC week window).
	weakArea *weakAreaComputer
}

// NewService wires the review application. verifier checks gateway-minted JWTs.
func NewService(st store.Store, verifier auth.Verifier, log *slog.Logger) *Service {
	return &Service{
		store:    st,
		verifier: verifier,
		log:      log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
		weakArea: &weakAreaComputer{store: st, log: log},
	}
}

// WithAccountResolver injects the identity client the background workers use to resolve
// account timezone + study-budget (ADR-0016). Returns the service for chaining. Omit it
// in local dev / tests: the weak-area recompute uses a UTC week and reminders schedule
// immediately.
func (s *Service) WithAccountResolver(r accountResolver) *Service {
	s.accounts = r
	s.weakArea.accounts = r
	return s
}

// Handler builds review's HTTP routes (Go 1.22+ method+pattern mux). The Revision
// routes verify the gateway-minted JWT; the account is the token subject. The
// external gateway surface /xlearn/api/revision/* maps to these internal
// /revisions/* routes (api.md: external `revision`, internal `revisions`).
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	mux.Handle("GET /revisions/due", s.requireJWT(http.HandlerFunc(s.handleDueQueue)))
	mux.Handle("POST /revisions/{id}/score", s.requireJWT(http.HandlerFunc(s.handleScore)))

	// Mistake journal + weekly weak-area + reminders (S07). The gateway maps external
	// /mistakes · /weak-area · /dashboard onto these internal routes.
	mux.Handle("GET /mistakes", s.requireJWT(http.HandlerFunc(s.handleListMistakes)))
	mux.Handle("POST /mistakes", s.requireJWT(http.HandlerFunc(s.handleCreateMistake)))
	mux.Handle("PATCH /mistakes/{id}", s.requireJWT(http.HandlerFunc(s.handlePatchMistake)))
	mux.Handle("GET /weak-area/current", s.requireJWT(http.HandlerFunc(s.handleWeakArea)))
	mux.Handle("GET /reminders", s.requireJWT(http.HandlerFunc(s.handleReminders)))

	return mux
}

// NewOutboxRelay builds the review outbox relay over pub (a JetStream publisher in
// prod, the log publisher in local dev). Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log)
}

// StartConsumers binds the durable pull consumer on XLEARN_PRACTICE and routes each
// practice event to the right handler. It returns the running subscription (Stop it
// on shutdown). Handlers dedupe on event_id via the inbox (idempotent).
func (s *Service) StartConsumers(ctx context.Context, consumer events.Consumer) (events.Subscription, error) {
	h := &practiceHandler{store: s.store, log: s.log}
	return consumer.Subscribe(ctx, DurableName, PracticeSubjectFilter, h)
}

// StartNotifications binds the in-app reminder worker as a durable consumer on review's
// OWN stream (XLEARN_REVIEW), filtered to revision_due. It is a separate consumer from
// the practice one (own durable, own package) so a later split into a standalone
// notifications service is mechanical (ADR-0016). The store dedupes on event_id.
func (s *Service) StartNotifications(ctx context.Context, consumer events.Consumer) (events.Subscription, error) {
	h := notifications.NewHandler(s.store, s.accounts, s.log)
	// DeliverNew: XLEARN_REVIEW already holds S06's revision_due history, and a stale
	// due-review is not worth an in-app reminder — the notifications durable must start
	// at the stream head on first deploy, not replay every past event (ADR-0016). A
	// later restart still resumes from its committed offset (offline catch-up).
	return consumer.Subscribe(ctx, NotificationsDurable, RevisionDueSubjectFilter, h, events.WithDeliverNew())
}

// outboxSource adapts the review store to events.OutboxSource.
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
