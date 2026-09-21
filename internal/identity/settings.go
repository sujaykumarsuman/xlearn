package identity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// This file implements the S10 account surface on identity: the partial PATCH /me
// (profile / study budget / timezone / reminders) and the shared validators reused by
// the onboarding budget step. The handler is JWT-scoped to the caller's own account;
// email is owned by the OAuth identity and is never writable here (server-authoritative).

// Study-budget + reminder value bounds. The weekday-minutes range matches the
// Settings stepper (30..240); the weekend band + reminder-time shapes match the
// segmented control and time input in Settings.dc.html / Auth.dc.html.
const (
	minWeekdayMinutes = 30
	maxWeekdayMinutes = 240
	maxDisplayName    = 120
	defaultReminderAt = "20:00"
)

// hhmm matches a 24-hour HH:MM local time (00:00..23:59), zero-padded.
var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// handlePatchAccount applies a partial update to the caller's own account (PATCH /me).
// Only fields present in the body are changed; email is read-only and ignored. The
// study-budget / reminders blobs are validated + canonicalised before they are stored.
func (s *Service) handlePatchAccount(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	id := r.PathValue("id")
	if claims.Subject != id {
		writeError(w, http.StatusForbidden, "forbidden", "account does not match token subject")
		return
	}

	var body struct {
		DisplayName *string         `json:"display_name"`
		Timezone    *string         `json:"timezone"`
		StudyBudget json.RawMessage `json:"study_budget"`
		Reminders   json.RawMessage `json:"reminders"`
		// The server-owned account fields (id/email/created_at) are accepted-but-ignored
		// so a client can PATCH back the whole /me account object without a 400 on
		// DisallowUnknownFields — only the four editable fields above are applied.
		ID        *string `json:"id"`
		Email     *string `json:"email"`
		CreatedAt *string `json:"created_at"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}

	update := store.AccountUpdate{}
	if body.DisplayName != nil {
		name, err := validateDisplayName(*body.DisplayName)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_display_name", err.Error())
			return
		}
		update.DisplayName = &name
	}
	if body.Timezone != nil {
		tz, err := validateTimezone(*body.Timezone)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_timezone", err.Error())
			return
		}
		update.Timezone = &tz
	}
	if len(body.StudyBudget) > 0 {
		budget, err := validateStudyBudget(body.StudyBudget)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_budget", err.Error())
			return
		}
		update.StudyBudget = budget
	}
	if len(body.Reminders) > 0 {
		reminders, err := validateReminders(body.Reminders)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_reminders", err.Error())
			return
		}
		update.Reminders = reminders
	}

	if _, err := s.store.UpdateAccount(r.Context(), id, update); err != nil {
		s.mapStoreErr(w, err)
		return
	}
	// Return the same {account, onboarding} shape as GET /me so the SPA can use the
	// response directly (and so study_budget / reminders round-trip to the form).
	s.writeAccount(w, r, id)
}

// validateDisplayName trims and bounds a profile name.
func validateDisplayName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("display_name must not be empty")
	}
	// Count runes, not bytes — a multibyte name (CJK, emoji) is well within the
	// character limit even when its byte length exceeds it.
	if utf8.RuneCountInString(name) > maxDisplayName {
		return "", fmt.Errorf("display_name must be at most %d characters", maxDisplayName)
	}
	return name, nil
}

// validateTimezone requires a resolvable IANA zone name (e.g. Asia/Kolkata). "" and
// the machine-local alias "Local" are rejected so reminder times are always computed
// against a real, portable zone.
func validateTimezone(raw string) (string, error) {
	tz := strings.TrimSpace(raw)
	if tz == "" || tz == "Local" {
		return "", errors.New("timezone must be an IANA name (e.g. Asia/Kolkata)")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "", errors.New("unknown IANA timezone")
	}
	return tz, nil
}

// validateStudyBudget validates + canonicalises study_budget_json:
// { weekday_minutes (30..240), weekend_band ("2"|"3-4"|"5") }.
func validateStudyBudget(raw json.RawMessage) ([]byte, error) {
	var b struct {
		WeekdayMinutes *int   `json:"weekday_minutes"`
		WeekendBand    string `json:"weekend_band"`
	}
	if err := strictUnmarshal(raw, &b); err != nil {
		return nil, errors.New("study_budget must be an object of {weekday_minutes, weekend_band}")
	}
	if b.WeekdayMinutes == nil {
		return nil, errors.New("weekday_minutes is required")
	}
	if *b.WeekdayMinutes < minWeekdayMinutes || *b.WeekdayMinutes > maxWeekdayMinutes {
		return nil, fmt.Errorf("weekday_minutes must be between %d and %d", minWeekdayMinutes, maxWeekdayMinutes)
	}
	switch b.WeekendBand {
	case "2", "3-4", "5":
	default:
		return nil, errors.New(`weekend_band must be one of "2", "3-4", "5"`)
	}
	return json.Marshal(map[string]any{
		"weekday_minutes": *b.WeekdayMinutes,
		"weekend_band":    b.WeekendBand,
	})
}

// validateReminders validates + canonicalises reminders_json:
// { daily_reminder_on, daily_reminder_time ("HH:MM" local), revision_due_alerts_on }.
func validateReminders(raw json.RawMessage) ([]byte, error) {
	var rm struct {
		DailyReminderOn     *bool  `json:"daily_reminder_on"`
		DailyReminderTime   string `json:"daily_reminder_time"`
		RevisionDueAlertsOn *bool  `json:"revision_due_alerts_on"`
	}
	if err := strictUnmarshal(raw, &rm); err != nil {
		return nil, errors.New("reminders must be an object")
	}
	if rm.DailyReminderOn == nil || rm.RevisionDueAlertsOn == nil {
		return nil, errors.New("daily_reminder_on and revision_due_alerts_on are required")
	}
	t := strings.TrimSpace(rm.DailyReminderTime)
	if t == "" {
		t = defaultReminderAt
	}
	if !hhmm.MatchString(t) {
		return nil, errors.New("daily_reminder_time must be HH:MM (24-hour)")
	}
	return json.Marshal(map[string]any{
		"daily_reminder_on":      *rm.DailyReminderOn,
		"daily_reminder_time":    t,
		"revision_due_alerts_on": *rm.RevisionDueAlertsOn,
	})
}

// strictUnmarshal decodes JSON with unknown-field rejection so a typo'd or extra key
// is a validation error, not a silently-dropped field.
func strictUnmarshal(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
