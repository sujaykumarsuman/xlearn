package practice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

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
	// CoachAssistAt (m1-07, D27): when the coach was first used on the OPEN attempt; null
	// when it never was or no attempt is open. The Problem HUD chip and the outcome step's
	// cap line read it.
	CoachAssistAt *string `json:"coachAssistAt"`
}

// openAttemptJSON is one entry of GET /attempts/open.
type openAttemptJSON struct {
	AttemptID     string  `json:"attemptId"`
	ProblemID     string  `json:"problemId"`
	PathSlug      string  `json:"pathSlug"`
	Purpose       string  `json:"purpose"`
	StartedAt     string  `json:"startedAt"`
	StageReached  string  `json:"stageReached"`
	CoachAssistAt *string `json:"coachAssistAt"`
}

// openProblemJSON is GET /attempts/open's `problem` slot (only with ?problem_id=): the
// item's solve state, which the gateway's coach mode gate reads instead of /state/{id}.
type openProblemJSON struct {
	ProblemID     string  `json:"problemId"`
	Status        string  `json:"status"`
	FirstSolvedAt *string `json:"firstSolvedAt"`
}

type penaltyJSON struct {
	OwedAttempt bool   `json:"owedAttempt"`
	DueInDays   int    `json:"dueInDays"`
	Message     string `json:"message"`
}

// --- handlers ---

// handleGetState: GET /state/{problemId} — one problem's state + active timer.
func (s *Service) handleGetState(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
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
	accountID := auth.ClaimsFrom(r.Context()).Subject
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

// handleStartAttempt: POST /problems/{id}/attempt/start?path=<slug> — create/resume
// the attempt and start the 15-min timer. The course is written when the problem
// state row is created; a resume keeps the row's own course.
func (s *Service) handleStartAttempt(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	pathSlug, ok := s.resolveCourse(w, r)
	if !ok {
		return
	}
	st, err := s.store.StartAttempt(r.Context(), accountID, problemID, pathSlug)
	if err != nil {
		s.mapErr(w, "start attempt", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": toStateJSON(st)})
}

// handleReveal: POST /problems/{id}/reveal?path=<slug> — unlock the next content
// stage; returns the penalty ack when the solution is revealed before the attempt timer
// elapses. A reveal never creates a problem state row, so `path` is only validated: the
// solution_revealed_early event carries the row's course.
func (s *Service) handleReveal(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	if _, ok := s.resolveCourse(w, r); !ok {
		return
	}
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

// handleOutcome: POST /problems/{id}/outcome?path=<slug> — log Clean/Rough/Assisted/
// Miss. Like a reveal it only validates `path`: problem_solved and attempt_logged carry
// the problem state row's course.
func (s *Service) handleOutcome(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	problemID := r.PathValue("id")
	if _, ok := s.resolveCourse(w, r); !ok {
		return
	}
	var body struct {
		Outcome string `json:"outcome"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	res, err := s.store.LogOutcome(r.Context(), accountID, problemID, body.Outcome)
	if err != nil {
		s.mapErr(w, "log outcome", err)
		return
	}
	out := map[string]any{"state": toStateJSON(res.State)}
	if res.CappedBy != "" {
		// D27: the self-reported clean/rough was recorded as assisted.
		out["cappedBy"] = res.CappedBy
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAssist: POST /attempts/{id}/assist — record the first coach use on the
// account's open attempt (D27; idempotent, the first time is kept). The gateway calls it
// BEFORE forwarding a chat about the attempt's problem, and sends nothing if it fails.
// 404 for an unknown attempt or another account's, 409 attempt_closed for a concluded one.
func (s *Service) handleAssist(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	attemptID := r.PathValue("id")
	at, err := s.store.MarkCoachAssist(r.Context(), accountID, attemptID)
	if err != nil {
		s.mapErr(w, "mark coach assist", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"attemptId": attemptID, "coachAssistAt": timePtr(at)})
}

// handleOpenAttempts: GET /attempts/open[?problem_id=] — the account's open attempts,
// newest first (purpose "course" until M2a, when open touches join the same list), plus
// with ?problem_id= the item's solve state. It is the coach mode gate's one practice read
// (m1-07): it replaces the coach path's /state/{id} call.
func (s *Service) handleOpenAttempts(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	problemID := strings.TrimSpace(r.URL.Query().Get("problem_id"))
	atts, err := s.store.ListOpenAttempts(r.Context(), accountID, problemID)
	if err != nil {
		s.mapErr(w, "list open attempts", err)
		return
	}
	list := make([]openAttemptJSON, 0, len(atts))
	for _, a := range atts {
		list = append(list, openAttemptJSON{
			AttemptID:     a.ID,
			ProblemID:     a.ProblemID,
			PathSlug:      a.PathSlug,
			Purpose:       a.Purpose,
			StartedAt:     a.StartedAt.UTC().Format(time.RFC3339),
			StageReached:  a.StageReached,
			CoachAssistAt: timePtr(a.CoachAssistAt),
		})
	}
	out := map[string]any{"attempts": list}
	if problemID != "" {
		st, err := s.store.GetState(r.Context(), accountID, problemID)
		if err != nil {
			s.mapErr(w, "open attempts: problem state", err)
			return
		}
		out["problem"] = openProblemJSON{ProblemID: problemID, Status: st.Status, FirstSolvedAt: timePtr(st.FirstSolvedAt)}
	}
	writeJSON(w, http.StatusOK, out)
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
	out.FirstSolvedAt = timePtr(st.FirstSolvedAt)
	out.CoachAssistAt = timePtr(st.CoachAssistAt)
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

// timePtr renders a time as RFC 3339 (UTC), or nil for the zero time.
func timePtr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	v := t.UTC().Format(time.RFC3339)
	return &v
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
	case errors.Is(err, store.ErrAttemptClosed):
		writeError(w, http.StatusConflict, "attempt_closed", "the attempt has concluded")
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
