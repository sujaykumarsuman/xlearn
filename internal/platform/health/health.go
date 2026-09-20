// Package health provides the standard liveness and readiness endpoints every
// xLearn service exposes for the chart's probes: /healthz (liveness — the
// process is up) and /readyz (readiness — dependencies are usable). Readiness
// runs a set of registered checks; with none registered it is trivially OK,
// which is all the stateless gateway needs this sprint (no DB yet).
package health

import (
	"context"
	"encoding/json"
	"net/http"
)

// Check reports whether a dependency is ready. It returns an error describing
// the problem when not ready.
type Check func(ctx context.Context) error

// Named pairs a readiness check with a short name for the readyz report.
type Named struct {
	Name  string
	Check Check
}

// Handler serves liveness and readiness.
type Handler struct {
	checks []Named
}

// New builds a Handler with the given readiness checks (may be none).
func New(checks ...Named) *Handler {
	return &Handler{checks: checks}
}

// Live is the liveness handler: 200 as long as the process can serve.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready is the readiness handler: 200 when every check passes, else 503 with the
// failing checks named.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	failures := map[string]string{}
	for _, c := range h.checks {
		if err := c.Check(r.Context()); err != nil {
			failures[c.Name] = err.Error()
		}
	}
	if len(failures) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unready", "checks": failures})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
