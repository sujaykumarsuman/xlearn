package review

import (
	"errors"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// This file is the mistake-journal / weak-area / reminders HTTP surface of the review
// service (S07). All routes verify the gateway-minted JWT and derive the account from
// the token subject (ADR-0006); the gateway maps its external /mistakes · /weak-area ·
// /dashboard surface onto these internal routes.

// --- JSON shapes (api.md conventions) ---

// mistakeJSON is one journal row. problemId + pattern are review-owned; the gateway
// enriches with the curriculum problem title/difficulty (like the due queue).
type mistakeJSON struct {
	ID           string  `json:"id"`
	ProblemID    string  `json:"problemId"`
	Pattern      string  `json:"pattern"`
	Mistake      string  `json:"mistake"`
	RootCause    string  `json:"rootCause"`
	Insight      string  `json:"insight"`
	Category     string  `json:"category"` // "" = uncategorised
	Status       string  `json:"status"`   // open | closed
	RevisitCount int     `json:"revisitCount"`
	RevisitDate  *string `json:"revisitDate"`
	CreatedAt    string  `json:"createdAt"`
}

func toMistakeJSON(m store.Mistake) mistakeJSON {
	j := mistakeJSON{
		ID:           m.ID,
		ProblemID:    m.ProblemID,
		Pattern:      m.Pattern,
		Mistake:      m.Mistake,
		RootCause:    m.RootCause,
		Insight:      m.Insight,
		Category:     m.Category,
		Status:       m.Status,
		RevisitCount: m.RevisitCount,
		CreatedAt:    m.CreatedAt.UTC().Format(time.RFC3339),
	}
	if !m.RevisitDate.IsZero() {
		d := m.RevisitDate.UTC().Format(time.RFC3339)
		j.RevisitDate = &d
	}
	return j
}

// --- handlers ---

// handleListMistakes: GET /mistakes?status=open|closed — the journal. The list honours
// the optional status filter; openCount/closedCount are always over the full journal so
// the screen header + segmented filter read consistently. closeThreshold surfaces the
// "n/2" denominator without hardcoding it client-side.
func (s *Service) handleListMistakes(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	status := r.URL.Query().Get("status")
	if status != "" && status != store.MistakeOpen && status != store.MistakeClosed {
		writeError(w, http.StatusBadRequest, "bad_request", "status must be open or closed")
		return
	}
	all, err := s.store.ListMistakes(r.Context(), accountID, "")
	if err != nil {
		s.mapErr(w, "list mistakes", err)
		return
	}
	openCount, closedCount := 0, 0
	for _, m := range all {
		if m.Status == store.MistakeOpen {
			openCount++
		} else {
			closedCount++
		}
	}
	out := make([]mistakeJSON, 0, len(all))
	for _, m := range all {
		if status != "" && m.Status != status {
			continue
		}
		out = append(out, toMistakeJSON(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mistakes":       out,
		"openCount":      openCount,
		"closedCount":    closedCount,
		"closeThreshold": store.MistakeCloseThreshold,
		"categories":     store.MistakeCategories,
	})
}

// handleCreateMistake: POST /mistakes — manually create an entry.
func (s *Service) handleCreateMistake(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	var body struct {
		ProblemID string `json:"problemId"`
		Pattern   string `json:"pattern"`
		Mistake   string `json:"mistake"`
		RootCause string `json:"rootCause"`
		Insight   string `json:"insight"`
		Category  string `json:"category"`
		Status    string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	if body.ProblemID == "" {
		writeError(w, http.StatusUnprocessableEntity, "invalid_mistake", "problemId is required")
		return
	}
	m, err := s.store.CreateMistake(r.Context(), accountID, store.MistakeInput{
		ProblemID: body.ProblemID,
		Pattern:   body.Pattern,
		Mistake:   body.Mistake,
		RootCause: body.RootCause,
		Insight:   body.Insight,
		Category:  body.Category,
		Status:    body.Status,
	})
	if err != nil {
		s.mapErr(w, "create mistake", err)
		return
	}
	writeJSON(w, http.StatusCreated, toMistakeJSON(m))
}

// handlePatchMistake: PATCH /mistakes/{id} — edit root cause / insight / category /
// status / revisit. Absent fields are unchanged (pointer presence).
func (s *Service) handlePatchMistake(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	id := r.PathValue("id")
	var body struct {
		Pattern      *string `json:"pattern"`
		Mistake      *string `json:"mistake"`
		RootCause    *string `json:"rootCause"`
		Insight      *string `json:"insight"`
		Category     *string `json:"category"`
		Status       *string `json:"status"`
		RevisitCount *int    `json:"revisitCount"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	m, err := s.store.UpdateMistake(r.Context(), accountID, id, store.MistakePatch{
		Pattern:      body.Pattern,
		Mistake:      body.Mistake,
		RootCause:    body.RootCause,
		Insight:      body.Insight,
		Category:     body.Category,
		Status:       body.Status,
		RevisitCount: body.RevisitCount,
	})
	if err != nil {
		s.mapErr(w, "update mistake", err)
		return
	}
	writeJSON(w, http.StatusOK, toMistakeJSON(m))
}

// handleWeakArea: GET /weak-area/current — the weekly weak-area banner (R-MJ3). When
// no snapshot exists yet, topCategory is "" and the client hides the banner.
func (s *Service) handleWeakArea(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	wa, found, err := s.store.WeakAreaCurrent(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "weak area", err)
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{
			"topCategory": "",
			"topCount":    0,
			"counts":      map[string]int{},
			"entries":     []mistakeJSON{},
		})
		return
	}
	entries := make([]mistakeJSON, 0, len(wa.Entries))
	for _, m := range wa.Entries {
		entries = append(entries, toMistakeJSON(m))
	}
	resp := map[string]any{
		"topCategory": wa.TopCategory,
		"topCount":    wa.TopCount,
		"counts":      wa.Counts,
		"entries":     entries,
	}
	if !wa.WeekOf.IsZero() {
		resp["weekOf"] = wa.WeekOf.UTC().Format("2006-01-02")
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleReminders: GET /reminders — the account's due in-app reminders, for the
// gateway's Dashboard aggregation.
func (s *Service) handleReminders(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	rows, err := s.store.ListDueReminders(r.Context(), accountID, reminderLimit)
	if err != nil {
		s.mapErr(w, "list reminders", err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, rem := range rows {
		out = append(out, map[string]any{
			"id":    rem.ID,
			"kind":  rem.Kind,
			"dueAt": rem.DueAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"reminders": out})
}

// reminderLimit caps the reminders returned to the Dashboard.
const reminderLimit = 50

// mapMistakeErr is folded into the shared mapErr (below in handlers.go), extended here
// for the S07 error classes.
func mapMistakeErr(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "an open mistake already exists for this problem")
		return true
	case errors.Is(err, store.ErrInvalidCategory):
		writeError(w, http.StatusUnprocessableEntity, "invalid_category", "invalid category or status")
		return true
	default:
		return false
	}
}
