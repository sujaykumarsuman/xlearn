package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// claimsCtxKey carries verified JWT claims from requireJWT to the handler.
type claimsCtxKey struct{}

// --- JSON response shapes (api.md conventions) ---

type timerJSON struct {
	Kind             string `json:"kind"`
	DeadlineAt       string `json:"deadlineAt"`
	RemainingSeconds int    `json:"remainingSeconds"`
	Expired          bool   `json:"expired"`
}

type stateJSON struct {
	ProblemID      string     `json:"problemId"`
	Status         string     `json:"status"`
	StageReached   string     `json:"stageReached,omitempty"`
	UnlockedStages []string   `json:"unlockedStages,omitempty"`
	CurrentTouch   int        `json:"currentTouch"`
	LastOutcome    *string    `json:"lastOutcome"`
	FirstSolvedAt  *string    `json:"firstSolvedAt"`
	RevealedEarly  bool       `json:"revealedEarly"`
	Timer          *timerJSON `json:"timer"`
}

type penaltyJSON struct {
	OwedAttempt bool   `json:"owedAttempt"`
	DueInDays   int    `json:"dueInDays"`
	Message     string `json:"message"`
}

// --- handlers ---

// handleGetState: GET /state/{problemId} — one problem's state + active timer.
func (s *Service) handleGetState(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	problemID := r.PathValue("problemId")
	st, err := s.store.GetState(r.Context(), accountID, problemID)
	if err != nil {
		s.mapErr(w, "get state", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": toStateJSON(st)})
}

// handleListStates: GET /state?week=&ids= — per-problem states for a set of problem
// ids (the Week five-touch dots + daily plan). practice is week-agnostic, so the
// gateway supplies the week's problem ids via ?ids=; `week` is accepted for logging.
func (s *Service) handleListStates(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	ids := parseIDs(r.URL.Query().Get("ids"))
	states, err := s.store.ListStates(r.Context(), accountID, ids)
	if err != nil {
		s.mapErr(w, "list states", err)
		return
	}
	out := make(map[string]stateJSON, len(states))
	for id, st := range states {
		out[id] = toStateJSON(st)
	}
	writeJSON(w, http.StatusOK, map[string]any{"states": out})
}

// handleStartAttempt: POST /problems/{id}/attempt/start — create/resume the attempt
// and start the 15-min timer.
func (s *Service) handleStartAttempt(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	st, err := s.store.StartAttempt(r.Context(), accountID, problemID)
	if err != nil {
		s.mapErr(w, "start attempt", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": toStateJSON(st)})
}

// handleReveal: POST /problems/{id}/reveal — unlock the next content stage; returns
// the penalty ack when the solution is revealed before the attempt timer elapses.
func (s *Service) handleReveal(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	res, err := s.store.Reveal(r.Context(), accountID, problemID)
	if err != nil {
		s.mapErr(w, "reveal", err)
		return
	}
	body := map[string]any{
		"revealed": res.Revealed,
		"state":    toStateJSON(res.State),
		"penalty":  nil,
	}
	if res.Penalty != nil {
		body["penalty"] = penaltyJSON{
			OwedAttempt: res.Penalty.OwedAttempt,
			DueInDays:   res.Penalty.DueInDays,
			Message: fmt.Sprintf("Revealing the full solution now means you owe #%s another attempt in %d days.",
				problemID, res.Penalty.DueInDays),
		}
	}
	writeJSON(w, http.StatusOK, body)
}

// handleOutcome: POST /problems/{id}/outcome — log Clean/Rough/Assisted/Miss.
func (s *Service) handleOutcome(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	var body struct {
		Outcome string `json:"outcome"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	st, err := s.store.LogOutcome(r.Context(), accountID, problemID, body.Outcome)
	if err != nil {
		s.mapErr(w, "log outcome", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": toStateJSON(st)})
}

// --- conversions + helpers ---

func toStateJSON(st store.State) stateJSON {
	out := stateJSON{
		ProblemID:      st.ProblemID,
		Status:         st.Status,
		StageReached:   st.StageReached,
		UnlockedStages: st.UnlockedStages,
		CurrentTouch:   st.CurrentTouch,
		RevealedEarly:  st.RevealedEarly,
	}
	if st.LastOutcome != "" {
		v := st.LastOutcome
		out.LastOutcome = &v
	}
	if !st.FirstSolvedAt.IsZero() {
		v := st.FirstSolvedAt.UTC().Format(time.RFC3339)
		out.FirstSolvedAt = &v
	}
	if st.Timer != nil {
		remaining := int(time.Until(st.Timer.DeadlineAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		out.Timer = &timerJSON{
			Kind:             st.Timer.Kind,
			DeadlineAt:       st.Timer.DeadlineAt.UTC().Format(time.RFC3339),
			RemainingSeconds: remaining,
			Expired:          st.Timer.Expired,
		}
	}
	return out
}

// parseIDs splits a comma-separated id list, trimming blanks and de-duplicating.
func parseIDs(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(csv, ",") {
		id := strings.TrimSpace(part)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// requireJWT verifies the gateway-minted JWT (Authorization: Bearer) via JWKS and
// stores its claims in the request context (ADR-0006).
func (s *Service) requireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			writeUnauthenticated(w)
			return
		}
		claims, err := s.verifier.Verify(r.Context(), token)
		if err != nil {
			s.log.Warn("jwt verify failed", "err", err)
			writeUnauthenticated(w)
			return
		}
		ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFrom(ctx context.Context) auth.Claims {
	c, _ := ctx.Value(claimsCtxKey{}).(auth.Claims)
	return c
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// mapErr maps a store error to the right HTTP status + error envelope.
func (s *Service) mapErr(w http.ResponseWriter, what string, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrNoAttempt):
		writeError(w, http.StatusConflict, "no_attempt", "start an attempt first")
	case errors.Is(err, store.ErrAlreadySolved):
		writeError(w, http.StatusConflict, "already_solved", "this problem is already solved")
	case errors.Is(err, store.ErrNothingToReveal):
		writeError(w, http.StatusConflict, "nothing_to_reveal", "the solution is already revealed")
	case errors.Is(err, store.ErrInvalidOutcome):
		writeError(w, http.StatusUnprocessableEntity, "invalid_outcome", "outcome must be one of clean, rough, assisted, miss")
	default:
		s.log.Error("practice store error", "op", what, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}

// writeUnauthenticated emits exactly the api.md 401 envelope the SPA redirects on.
func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
}
