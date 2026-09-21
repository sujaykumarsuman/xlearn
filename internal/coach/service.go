package coach

import (
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
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
// only for a provider call and never logged or returned.
type Service struct {
	store     store.Store
	verifier  auth.Verifier
	cipher    *secrets.Cipher
	openai    Provider
	anthropic Provider
	log       *slog.Logger
	health    *health.Handler
}

// NewService wires the coach application. verifier checks gateway-minted JWTs; cipher
// performs envelope encryption; openai/anthropic are the provider clients.
func NewService(st store.Store, verifier auth.Verifier, cipher *secrets.Cipher, openai, anthropic Provider, log *slog.Logger) *Service {
	return &Service{
		store:     st,
		verifier:  verifier,
		cipher:    cipher,
		openai:    openai,
		anthropic: anthropic,
		log:       log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
	}
}

// Handler builds coach's HTTP routes (Go 1.22+ method+pattern mux). The key + chat +
// thread routes verify the gateway-minted JWT; the account is the token subject. The
// external gateway surface (/coach/key, /coach/chat, /coach/thread) maps onto these
// internal /keys, /chat, /threads routes (services.md).
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	mux.Handle("GET /keys", s.requireJWT(http.HandlerFunc(s.handleGetKey)))
	mux.Handle("PUT /keys", s.requireJWT(http.HandlerFunc(s.handlePutKey)))
	mux.Handle("DELETE /keys", s.requireJWT(http.HandlerFunc(s.handleDeleteKey)))

	mux.Handle("POST /chat", s.requireJWT(http.HandlerFunc(s.handleChat)))
	mux.Handle("GET /threads", s.requireJWT(http.HandlerFunc(s.handleThread)))

	return mux
}
