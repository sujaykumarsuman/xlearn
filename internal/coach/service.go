package coach

import (
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// historyLimit caps how many prior thread turns are replayed to the provider (bounds
// token spend on the user's key). The full history is still returned by GET /threads.
const historyLimit = 20

// Service is the coach HTTP application: the key-config CRUD, the SSE chat stream, and
// the thread history endpoint, plus the k8s probes. It verifies the gateway-minted JWT
// on every user route (ADR-0006) and derives the account id from the token subject. The
// cipher seals/opens the provider key (ADR-0007); the raw key is decrypted in memory
// only for a provider call and never logged or returned. courses resolves the course of
// a page context (sprint m1-03: the thread key and path_slug).
type Service struct {
	store     store.Store
	verifier  auth.Verifier
	cipher    *secrets.Cipher
	openai    Provider
	anthropic Provider
	courses   *course.Registry
	log       *slog.Logger
	health    *health.Handler
}

// NewService wires the coach application. verifier checks gateway-minted JWTs; cipher
// performs envelope encryption; openai/anthropic are the provider clients; courses is
// the course registry (production: the embedded manifests, loaded at startup; tests:
// coursetest.Registry). A nil courses means the embedded registry.
func NewService(st store.Store, verifier auth.Verifier, cipher *secrets.Cipher, openai, anthropic Provider, courses *course.Registry, log *slog.Logger) *Service {
	if courses == nil {
		courses = course.Embedded()
	}
	return &Service{
		store:     st,
		verifier:  verifier,
		cipher:    cipher,
		openai:    openai,
		anthropic: anthropic,
		courses:   courses,
		log:       log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
	}
}

// Handler builds coach's HTTP routes (Go 1.22+ method+pattern mux). Every user route
// (userRoutes) verifies the gateway-minted JWT and requires the learner role; the
// account is the token subject. The external gateway surface (/coach/key, /coach/chat,
// /coach/thread) maps onto the internal /keys, /chat, /threads routes (services.md).
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

// userRoutes is coach's per-user route table: the key config, the chat stream and the
// thread history.
func (s *Service) userRoutes() []userRoute {
	return []userRoute{
		{http.MethodGet, "/keys", s.handleGetKey},
		{http.MethodPut, "/keys", s.handlePutKey},
		{http.MethodDelete, "/keys", s.handleDeleteKey},

		{http.MethodPost, "/chat", s.handleChat},
		{http.MethodGet, "/threads", s.handleThread},
	}
}
