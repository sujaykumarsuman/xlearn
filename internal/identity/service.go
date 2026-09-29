package identity

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// ServiceName is identity's identity on NATS (topology owner, connection names, nkey
// user in the rendered ACL).
const ServiceName = "identity"

// StreamIdentity is identity's JetStream stream (events.md); its subjects and limits
// live in internal/platform/events/topology.go. main passes it to the NATS publisher.
const StreamIdentity = events.StreamIdentity

// Service is the identity HTTP application: OAuth, sessions, accounts, onboarding,
// and the internal JWT-protected routes. It owns no signing key (the gateway
// mints + publishes JWKS, ADR-0006); it only verifies inbound JWTs. courses is the
// course registry enrollment validates against (sprint m1-03).
type Service struct {
	cfg       Config
	store     store.Store
	verifier  auth.Verifier
	courses   *course.Registry
	providers map[string]*oauthProvider
	httpc     *http.Client
	log       *slog.Logger
	health    *health.Handler
	// pw is the L3-gated bcrypt (process-wide; tests swap in their own).
	pw *passwords
}

// NewService wires the identity application. verifier checks gateway-minted JWTs on
// the protected internal routes; courses is the course registry (production: the
// embedded manifests, loaded at startup; tests: coursetest.Registry). A nil courses
// means the embedded registry.
func NewService(cfg Config, st store.Store, verifier auth.Verifier, courses *course.Registry, log *slog.Logger) *Service {
	if courses == nil {
		courses = course.Embedded()
	}
	pw := processPasswords()
	// L3: the dummy hash is made once, at startup, at the real bcrypt cost.
	if err := pw.warm(); err != nil {
		log.Error("bcrypt dummy hash failed; logins will 500", "err", err)
	}
	return &Service{
		cfg:       cfg,
		store:     st,
		verifier:  verifier,
		courses:   courses,
		providers: newProviders(cfg.Auth),
		httpc:     &http.Client{Timeout: 10 * time.Second},
		log:       log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
		pw: pw,
	}
}

// userRoute is one JWT route: every entry of userRoutes is registered behind
// auth.RequireRole(learner) (ADR-0033 §12 row 5), so a route can't skip the role check.
type userRoute struct {
	Method, Pattern string
	Handler         http.HandlerFunc
}

// userRoutes are identity's user-data routes: the gateway-minted JWT (verified via the
// JWKS) must carry the learner role, and handlers check ownership against its subject.
func (s *Service) userRoutes() []userRoute {
	return []userRoute{
		{"GET", "/accounts/{id}", s.handleGetAccount},
		{"PATCH", "/accounts/{id}", s.handlePatchAccount},
		{"POST", "/onboarding/step", s.handleOnboardingStep},
		{"POST", "/paths/{slug}/start", s.handleStartEnrollment},
		// Account & sign-in management (ADR-0023): set/change password + disconnect a provider.
		{"POST", "/accounts/{id}/password", s.handleSetPassword},
		{"DELETE", "/accounts/{id}/oauth/{provider}", s.handleUnlinkOAuth},
		// Username claim/change + availability check (F009). JWT-scoped to the caller.
		{"POST", "/accounts/{id}/username", s.handleSetUsername},
		{"GET", "/username/available", s.handleUsernameAvailable},
	}
}

// Handler builds identity's HTTP routes (Go 1.22+ method+pattern mux).
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	// Browser-facing OAuth (reached through the gateway proxy). `?link=1` on start attaches
	// the provider to the signed-in account (Settings → Connect GitHub) instead of signing in.
	mux.HandleFunc("POST /auth/{provider}/start", s.handleStart)
	mux.HandleFunc("GET /auth/{provider}/callback", s.handleCallback)

	// Email/password auth (ADR-0023), reached through the gateway proxy (fetch-based, so the
	// session cookie is set on the JSON response, not a redirect).
	mux.HandleFunc("POST /auth/signup", s.handleSignup)
	mux.HandleFunc("POST /auth/login", s.handleLogin)

	// Local-only dev login (F002 / ADR-0022): 404 unless DEV_AUTH is set. Never
	// enabled in a prod image. Also reached through the gateway proxy.
	mux.HandleFunc("GET /auth/dev/enabled", s.handleDevEnabled)
	mux.HandleFunc("POST /auth/dev/login", s.handleDevLogin)

	// Gateway trust endpoints (ClusterIP + NetworkPolicy; no user JWT — these
	// establish identity from the opaque session).
	mux.HandleFunc("POST /sessions/validate", s.handleValidateSession)
	mux.HandleFunc("POST /sessions/revoke", s.handleRevokeSession)

	// Service-to-service account lookup for BACKGROUND workers with no user request
	// context (the review weak-area recompute + notifications worker resolve account
	// timezone/study-budget here, ADR-0016). Same trust model as the session endpoints:
	// ClusterIP-only, no user JWT — cluster-internal isolation (ADR-0006). It exposes
	// only non-sensitive scheduling prefs (timezone, study budget), never OAuth data.
	mux.HandleFunc("GET /internal/accounts/{id}", s.handleInternalGetAccount)
	// Public-profile resolver (F009): the gateway resolves /xlearn/u/<username> → account id
	// here for the UNAUTHENTICATED public dashboard. Same ClusterIP trust model; returns
	// only non-PII public fields (id, username, display name, join date).
	mux.HandleFunc("GET /internal/accounts/by-username/{username}", s.handleInternalGetAccountByUsername)

	// User-data routes: the gateway-minted JWT with the learner role (RequireRole) +
	// ownership against its subject in each handler.
	requireLearner := auth.RequireRole(s.verifier, auth.RoleLearner, auth.WithLogger(s.log))
	for _, rt := range s.userRoutes() {
		mux.Handle(rt.Method+" "+rt.Pattern, requireLearner(rt.Handler))
	}

	return mux
}

// NewOutboxRelay builds the identity outbox relay over pub: a JetStream publisher on
// XLEARN_IDENTITY when NATS_URL is set (compose; prod from mi-06's identity N2 PR),
// else the log publisher. Call Run in a goroutine.
func (s *Service) NewOutboxRelay(pub events.Publisher, opts ...events.RelayOption) *events.Relay {
	return events.NewRelay(outboxSource{s.store}, pub, s.log, opts...)
}

// outboxSource adapts the identity store to events.OutboxSource.
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
