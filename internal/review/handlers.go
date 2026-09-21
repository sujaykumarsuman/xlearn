package review

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// claimsCtxKey carries verified JWT claims from requireJWT to the handler.
type claimsCtxKey struct{}

// dueQueueLimit caps the due queue returned to the gateway/screen.
const dueQueueLimit = 100

// --- JSON response shapes (api.md conventions) ---

// dueItemJSON is one entry in the prioritised revision queue. dayLabel ("Day 1"…)
// and touchLevel let the screen group by touch day (D1/D3/D7/D21/D45); `due`
// distinguishes overdue reviews from the "coming up" tail.
type dueItemJSON struct {
	ItemID     string `json:"itemId"`
	ProblemID  string `json:"problemId"`
	TouchLevel int    `json:"touchLevel"`
	DayLabel   string `json:"dayLabel"`
	DueDate    string `json:"dueDate"`
	Due        bool   `json:"due"`
	MockMode   bool   `json:"mockMode"`
	Status     string `json:"status"`
}

type scoreResultJSON struct {
	ItemID         string  `json:"itemId"`
	ProblemID      string  `json:"problemId"`
	TouchLevel     int     `json:"touchLevel"`
	AutoPass       bool    `json:"autoPass"`
	Status         string  `json:"status"` // passed | failed
	MockMode       bool    `json:"mockMode"`
	Reset          bool    `json:"reset"`
	NextTouchLevel int     `json:"nextTouchLevel"` // 0 = ladder complete
	NextDayLabel   string  `json:"nextDayLabel"`
	NextDueDate    *string `json:"nextDueDate"`
}

// --- handlers ---

// handleDueQueue: GET /revisions/due — the account's prioritised revision queue
// (reviews before new work, R-SR5), grouped-ready by touch day.
func (s *Service) handleDueQueue(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	items, err := s.store.DueQueue(r.Context(), accountID, dueQueueLimit)
	if err != nil {
		s.mapErr(w, "due queue", err)
		return
	}
	out := make([]dueItemJSON, 0, len(items))
	dueCount := 0
	for _, it := range items {
		if it.Due {
			dueCount++
		}
		out = append(out, dueItemJSON{
			ItemID:     it.ItemID,
			ProblemID:  it.ProblemID,
			TouchLevel: it.TouchLevel,
			DayLabel:   dayLabel(it.TouchLevel),
			DueDate:    it.DueDate.UTC().Format(time.RFC3339),
			Due:        it.Due,
			MockMode:   it.MockMode,
			Status:     it.Status,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "dueCount": dueCount})
}

// handleScore: POST /revisions/{id}/score — auto-score a re-solve (R-SR2) and
// advance the touch or reset the ladder to Day 1 (R-SR3).
func (s *Service) handleScore(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	itemID := r.PathValue("id")
	var body struct {
		NamedPatternSecs int  `json:"namedPatternSecs"`
		SolvedInTimer    bool `json:"solvedInTimer"`
		StatedComplexity bool `json:"statedComplexity"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	if body.NamedPatternSecs < 0 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_score", "namedPatternSecs must be non-negative")
		return
	}
	res, err := s.store.Score(r.Context(), accountID, itemID, store.ScoreInput{
		NamedPatternSecs: body.NamedPatternSecs,
		SolvedInTimer:    body.SolvedInTimer,
		StatedComplexity: body.StatedComplexity,
	})
	if err != nil {
		s.mapErr(w, "score", err)
		return
	}
	out := scoreResultJSON{
		ItemID:         res.ItemID,
		ProblemID:      res.ProblemID,
		TouchLevel:     res.TouchLevel,
		AutoPass:       res.AutoPass,
		Status:         statusFor(res.AutoPass),
		MockMode:       res.MockMode,
		Reset:          res.Reset,
		NextTouchLevel: res.NextTouchLevel,
	}
	if res.NextTouchLevel > 0 {
		out.NextDayLabel = dayLabel(res.NextTouchLevel)
		nd := res.NextDueDate.UTC().Format(time.RFC3339)
		out.NextDueDate = &nd
	}
	writeJSON(w, http.StatusOK, out)
}

// --- helpers ---

// dayLabel renders a touch level as its human day ("Day 1" … "Day 45").
func dayLabel(level int) string {
	switch level {
	case 1:
		return "Day 1"
	case 2:
		return "Day 3"
	case 3:
		return "Day 7"
	case 4:
		return "Day 21"
	case 5:
		return "Day 45"
	default:
		return ""
	}
}

func statusFor(pass bool) string {
	if pass {
		return store.StatusPassed
	}
	return store.StatusFailed
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
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case mapMistakeErr(w, err):
		// Handled (409 conflict / 422 invalid category|status).
	default:
		s.log.Error("review store error", "op", what, "err", err)
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
