package curriculum

import (
	"log/slog"
	"net/http"

	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/health"
)

// Service is the curriculum HTTP application: the read-only content endpoints plus
// the k8s probes. It owns no auth (content is read-only, no user state — the gateway
// enforces the session boundary) and emits no events.
type Service struct {
	store  store.Store
	log    *slog.Logger
	health *health.Handler
}

// NewService wires the curriculum application.
func NewService(st store.Store, log *slog.Logger) *Service {
	return &Service{
		store: st,
		log:   log,
		health: health.New(health.Named{
			Name:  "postgres",
			Check: st.Ping,
		}),
	}
}

// Handler builds curriculum's HTTP routes (Go 1.22+ method+pattern mux). All
// content routes are read-only; there is no user state this sprint (ADR-0005).
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health.Live)
	mux.HandleFunc("GET /readyz", s.health.Ready)

	// Content reads (proxied through the gateway BFF under /xlearn/api).
	mux.HandleFunc("GET /paths", s.handleListPaths)
	mux.HandleFunc("GET /paths/{slug}", s.handleGetPath)
	mux.HandleFunc("GET /paths/{slug}/problems", s.handleListPathProblems)
	mux.HandleFunc("GET /paths/{slug}/weeks/{n}", s.handleGetWeek)
	mux.HandleFunc("GET /problems/{id}", s.handleGetProblem)
	mux.HandleFunc("GET /concepts/{slug}", s.handleGetConcept)

	return mux
}
