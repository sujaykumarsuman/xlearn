package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// claimsCtxKey carries verified JWT claims from requireJWT to the handler.
type claimsCtxKey struct{}

// Input caps: guard the opaque setup/notes fields so a hostile client can't stash
// unbounded text in a mock row (the ids are soft refs; the notes are free text).
const (
	maxSetupFieldLen = 128
	maxNotesLen      = 2000
)

// --- JSON response shapes (api.md conventions) ---

// dimScoreJSON is one scored dimension (key + display name + 1..5 score).
type dimScoreJSON struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// mockViewJSON is the mock session + server-computed phase-rail state + rubric. It is
// the shape returned by POST /mocks, GET /mocks/{id}, and POST /mocks/{id}/score, so
// the client renders setup/live/results off one contract keyed by `status`.
type mockViewJSON struct {
	ID         string         `json:"id"`
	Status     string         `json:"status"` // live | scored
	SetID      string         `json:"setId"`
	ProblemID  string         `json:"problemId"`
	Difficulty string         `json:"difficulty"`
	Date       string         `json:"date"`       // YYYY-MM-DD
	StartedAt  string         `json:"startedAt"`  // RFC3339 UTC
	DeadlineAt string         `json:"deadlineAt"` // RFC3339 UTC
	Total35    *int           `json:"total35"`    // null until scored
	Notes      string         `json:"notes"`
	Rail       RailState      `json:"rail"`       // server-authoritative timer + phases
	Dimensions []dimScoreJSON `json:"dimensions"` // empty until scored
	Targets    Targets        `json:"targets"`
}

// trendPointJSON is one scored mock in the trend series (R-MK3).
type trendPointJSON struct {
	MockID     string `json:"mockId"`
	SetID      string `json:"setId"`
	ProblemID  string `json:"problemId"`
	Difficulty string `json:"difficulty"`
	Date       string `json:"date"`
	StartedAt  string `json:"startedAt"`
	Total35    int    `json:"total35"`
}

// buildMockView assembles the response view, computing the server-authoritative rail
// state from the session's started_at against now (never a client clock).
func buildMockView(m store.MockSession, scores []store.RubricScore, now time.Time) mockViewJSON {
	dims := make([]dimScoreJSON, 0, len(scores))
	if len(scores) > 0 {
		byDim := make(map[string]int, len(scores))
		for _, sc := range scores {
			byDim[sc.Dimension] = sc.Score
		}
		// Emit in canonical PRD order (R-MK2), not storage order.
		for _, key := range store.Dimensions {
			dims = append(dims, dimScoreJSON{Key: key, Name: dimensionLabel(key), Score: byDim[key]})
		}
	}
	return mockViewJSON{
		ID:         m.ID,
		Status:     m.Status,
		SetID:      m.SetID,
		ProblemID:  m.ProblemID,
		Difficulty: m.Difficulty,
		Date:       m.Date.Format("2006-01-02"),
		StartedAt:  m.StartedAt.UTC().Format(time.RFC3339),
		DeadlineAt: m.DeadlineAt.UTC().Format(time.RFC3339),
		Total35:    m.Total35,
		Notes:      m.Notes,
		Rail:       computeRail(m.StartedAt, now),
		Dimensions: dims,
		Targets:    readinessTargets,
	}
}

// --- handlers ---

// handleStartMock: POST /mocks — accept the setup (problem set + difficulty), insert a
// live session with a server-authoritative 45-minute window (setup -> live), and
// return the live view (R-MK1).
func (s *Service) handleStartMock(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	var body struct {
		SetID      string `json:"setId"`
		ProblemID  string `json:"problemId"`
		Difficulty string `json:"difficulty"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	setID := strings.TrimSpace(body.SetID)
	problemID := strings.TrimSpace(body.ProblemID)
	if setID == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_setup", "setId is required")
		return
	}
	if len(setID) > maxSetupFieldLen || len(problemID) > maxSetupFieldLen {
		writeError(w, http.StatusUnprocessableEntity, "invalid_setup", "setId/problemId too long")
		return
	}
	difficulty := strings.TrimSpace(body.Difficulty)
	if difficulty == "" {
		difficulty = "med"
	}
	if !store.ValidDifficulty(difficulty) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_setup", "difficulty must be easy, med, or hard")
		return
	}
	now := time.Now()
	m, err := s.store.CreateMock(r.Context(), accountID, setID, problemID, difficulty, now, now.Add(store.MockDuration))
	if err != nil {
		s.mapErr(w, "create mock", err)
		return
	}
	writeJSON(w, http.StatusCreated, buildMockView(m, nil, time.Now()))
}

// handleGetMock: GET /mocks/{id} — the live session + server-computed phase-rail state
// and remaining time (refresh/return resumes the same countdown; elapsed clamps at
// 45:00). For a scored session it returns the rubric too (R-MK1/R-MK2).
func (s *Service) handleGetMock(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	m, scores, err := s.store.GetMock(r.Context(), accountID, r.PathValue("id"))
	if err != nil {
		s.mapErr(w, "get mock", err)
		return
	}
	writeJSON(w, http.StatusOK, buildMockView(m, scores, time.Now()))
}

// handleScoreMock: POST /mocks/{id}/score — record the seven-dimension rubric (each
// 1..5), compute /35 server-side, latch the session scored, and emit mock_completed
// via the outbox in the same transaction (R-MK2). Idempotent on re-submit.
func (s *Service) handleScoreMock(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	var body struct {
		Scores map[string]int `json:"scores"`
		Notes  string         `json:"notes"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	if err := store.ValidateRubric(body.Scores); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_rubric",
			"exactly the seven dimensions, each scored 1-5, are required")
		return
	}
	notes := strings.TrimSpace(body.Notes)
	if len(notes) > maxNotesLen {
		notes = notes[:maxNotesLen]
	}
	m, scores, err := s.store.ScoreMock(r.Context(), accountID, r.PathValue("id"), body.Scores, notes)
	if err != nil {
		s.mapErr(w, "score mock", err)
		return
	}
	writeJSON(w, http.StatusOK, buildMockView(m, scores, time.Now()))
}

// handleTrend: GET /mocks/trend — the account's scored /35 history against the R-MK3
// readiness targets (used to draw the W13/W15/pre target lines).
func (s *Service) handleTrend(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	points, err := s.store.Trend(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "trend", err)
		return
	}
	out := make([]trendPointJSON, 0, len(points))
	for _, p := range points {
		out = append(out, trendPointJSON{
			MockID:     p.MockID,
			SetID:      p.SetID,
			ProblemID:  p.ProblemID,
			Difficulty: p.Difficulty,
			Date:       p.Date.Format("2006-01-02"),
			StartedAt:  p.StartedAt.UTC().Format(time.RFC3339),
			Total35:    p.Total35,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": out, "targets": readinessTargets})
}

// --- helpers ---

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
	case errors.Is(err, store.ErrInvalidRubric):
		writeError(w, http.StatusUnprocessableEntity, "invalid_rubric",
			"exactly the seven dimensions, each scored 1-5, are required")
	default:
		s.log.Error("assessment store error", "op", what, "err", err)
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
