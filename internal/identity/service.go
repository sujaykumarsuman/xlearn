package identity

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// streamIdentity is identity's JetStream stream (events.md).
const streamIdentity = "XLEARN_IDENTITY"

// Service is the identity HTTP application: OAuth, sessions, accounts, onboarding,
// and the internal JWT-protected routes. It owns no signing key (the gateway
// mints + publishes JWKS, ADR-0006); it only verifies inbound JWTs.
type Service struct {
	cfg       Config
	store     store.Store
	verifier  auth.Verifier
	providers map[string]*oauthProvider
	httpc     *http.Client
	log       *slog.Logger
	health    *health.Handler
}

// NewService wires the identity application. verifier checks gateway-minted JWTs on
// the protected internal routes.
func NewService(cfg Config, st store.Store, verifier auth.Verifier, log *slog.Logger) *Service {
	return &Service{
		cfg:       cfg,
		store:     st,
		verifier:  verifier,
		providers: newProviders(cfg.Auth),
		httpc:     &http.Client{Timeout: 10 * time.Second},
		log:       log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
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

	// User-data routes: verify the gateway-minted JWT via JWKS + ownership.
	mux.Handle("GET /accounts/{id}", s.requireJWT(http.HandlerFunc(s.handleGetAccount)))
	mux.Handle("PATCH /accounts/{id}", s.requireJWT(http.HandlerFunc(s.handlePatchAccount)))
	mux.Handle("POST /onboarding/step", s.requireJWT(http.HandlerFunc(s.handleOnboardingStep)))
	mux.Handle("POST /paths/{slug}/start", s.requireJWT(http.HandlerFunc(s.handleStartEnrollment)))
	// Account & sign-in management (ADR-0023): set/change password + disconnect a provider.
	mux.Handle("POST /accounts/{id}/password", s.requireJWT(http.HandlerFunc(s.handleSetPassword)))
	mux.Handle("DELETE /accounts/{id}/oauth/{provider}", s.requireJWT(http.HandlerFunc(s.handleUnlinkOAuth)))

	return mux
}

// NewOutboxRelay builds the identity outbox relay with the placeholder log
// publisher (NATS JetStream lands in S05/S06). Call Run in a goroutine.
func (s *Service) NewOutboxRelay() *events.Relay {
	pub := events.NewLogPublisher(s.log, streamIdentity)
	return events.NewRelay(outboxSource{s.store}, pub, s.log)
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
