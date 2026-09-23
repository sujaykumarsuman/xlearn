package identity

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// --- Usernames (F009 / ADR-0024) ---

// handleSetUsername: POST /accounts/{id}/username (JWT) — claim or change the account's
// public handle. Validated + normalised + reserved-word checked before it hits the store;
// a case-insensitive collision is a 409. Mirrors handleSetPassword's ownership guard.
func (s *Service) handleSetUsername(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	id := r.PathValue("id")
	if claims.Subject != id {
		writeError(w, http.StatusForbidden, "forbidden", "account does not match token subject")
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	name := normalizeUsername(body.Username)
	if err := validateUsername(name); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_username", err.Error())
		return
	}
	acct, err := s.store.SetUsername(r.Context(), id, name)
	if err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			writeError(w, http.StatusConflict, "username_taken", "that username is already taken")
			return
		}
		s.mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "username": acct.Username})
}

// handleUsernameAvailable: GET /username/available?u=<name> (JWT) — the live availability
// check for the claim UI. Returns {available, reason?}. A malformed/reserved name is
// reported as unavailable with the reason (not an error status) so the UI can show it
// inline. The caller's own current username reads as available (re-saving is a no-op).
func (s *Service) handleUsernameAvailable(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	name := normalizeUsername(r.URL.Query().Get("u"))
	if err := validateUsername(name); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": err.Error()})
		return
	}
	acct, err := s.store.GetAccountByUsername(r.Context(), name)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusOK, map[string]any{"available": true})
	case err != nil:
		s.mapStoreErr(w, err)
	case acct.ID == claims.Subject:
		writeJSON(w, http.StatusOK, map[string]any{"available": true})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "that username is taken"})
	}
}

// handleInternalGetAccountByUsername: GET /internal/accounts/by-username/{username} — a
// service-to-service lookup (ClusterIP + NetworkPolicy; no user JWT) the gateway uses to
// resolve the PUBLIC profile at /xlearn/u/<username> → an account id. Like the other
// /internal/* endpoints (ADR-0016) it exposes ONLY non-PII public fields — never email,
// OAuth identities, timezone, budget or session data — so the public dashboard can never
// leak PII no matter what the gateway composes on top.
func (s *Service) handleInternalGetAccountByUsername(w http.ResponseWriter, r *http.Request) {
	name := normalizeUsername(r.PathValue("username"))
	if validateUsername(name) != nil {
		// A malformed handle can't be a real username — 404 (uniform with "no such user").
		writeError(w, http.StatusNotFound, "not_found", "no such user")
		return
	}
	acct, err := s.store.GetAccountByUsername(r.Context(), name)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":   acct.ID,
		"username":     acct.Username,
		"display_name": acct.DisplayName,
		"created_at":   acct.CreatedAt.UTC().Format(time.RFC3339),
		// A COARSE region (F009 review): only the account's UTC offset band, derived
		// server-side from the timezone — never the IANA zone/city (that stays PII).
		"region": utcOffsetLabel(acct.Timezone),
	})
}

// utcOffsetLabel renders an account's timezone as a coarse UTC-offset band (e.g.
// "UTC+05:30", "UTC") for the public profile — the low-identifying "Region" the owner chose
// (F009 review). It exposes only the offset, never the IANA zone name. "" when the zone is
// empty/unknown (the profile omits Region).
func utcOffsetLabel(tz string) string {
	if tz == "" {
		return ""
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return ""
	}
	_, off := time.Now().In(loc).Zone() // offset in seconds east of UTC
	if off == 0 {
		return "UTC"
	}
	sign := "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, off/3600, (off%3600)/60)
}
