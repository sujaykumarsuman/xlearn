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
	store    store.Store
	verifier auth.Verifier
	// cipher opens (and dual-writes) the LEGACY unbound sealed pair, which v1.6.0 can
	// still read. Required: it is the only way to read a key the re-wrap job has not
	// reached yet (ADR-0034 §3 — the rollback floor stays 1.6.0).
	cipher *secrets.Cipher
	// keys is the KEK keyring for the AD-bound pair. New material is sealed under its
	// active entry; an existing pair opens under the entry it was stamped with.
	keys *secrets.Keyring
	// providers is the provider registry: the single list of providers coach serves.
	providers map[string]Provider
	catalog   *Catalog
	courses   *course.Registry
	log       *slog.Logger
	health    *health.Handler
}

// NewService wires the coach application. verifier checks gateway-minted JWTs; cipher
// performs legacy envelope encryption and keys is the AD keyring (a nil keys falls back
// to a one-entry keyring over cipher, which is what tests and a single-key deployment
// want); openai/anthropic are the provider clients; courses is the course registry
// (production: the embedded manifests, loaded at startup; tests: coursetest.Registry). A
// nil courses means the embedded registry.
func NewService(st store.Store, verifier auth.Verifier, cipher *secrets.Cipher, keys *secrets.Keyring, openai, anthropic Provider, courses *course.Registry, log *slog.Logger) *Service {
	if courses == nil {
		courses = course.Embedded()
	}
	providers := map[string]Provider{}
	if openai != nil {
		providers[store.ProviderOpenAI] = openai
	}
	if anthropic != nil {
		providers[store.ProviderAnthropic] = anthropic
	}
	return &Service{
		store:     st,
		verifier:  verifier,
		cipher:    cipher,
		keys:      keys,
		providers: providers,
		catalog:   NewCatalog(),
		courses:   courses,
		log:       log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
	}
}

// Catalog is the model catalog this service serves and validates against.
func (s *Service) Catalog() *Catalog { return s.catalog }

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

		// The server model catalog (m1-10). JWT-scoped like every other coach route: it
		// carries no learner data, but keeping it behind the same gate means the public
		// profile stays the ONLY unauthenticated /api route.
		{http.MethodGet, "/models", s.handleModels},

		{http.MethodPost, "/chat", s.handleChat},
		{http.MethodGet, "/threads", s.handleThread},
	}
}
