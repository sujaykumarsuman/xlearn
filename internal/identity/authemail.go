package identity

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// --- Email/password auth (ADR-0023) ---

// handleSignup: POST /auth/signup — create an email/password account + session. No email
// verification yet (deferred), so the account is usable immediately. With SIGNUP_MODE
// closed it is a 403 before the body is even read, so it can't reveal registered emails.
func (s *Service) handleSignup(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Auth.Signup != SignupOpen {
		writeError(w, http.StatusForbidden, "signup_closed", "xLearn is invite-only right now")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	email := normalizeEmail(body.Email)
	if !validEmail(email) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_email", "enter a valid email address")
		return
	}
	if err := validatePassword(body.Password); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "weak_password", err.Error())
		return
	}
	hash, err := s.pw.hash(body.Password)
	if err != nil {
		if s.bcryptBusy(w, err) {
			return
		}
		s.log.Error("signup: hash password failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	acct, err := s.store.CreateEmailAccount(r.Context(), email, hash, displayNameFromEmail(email))
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email_taken", "an account with this email already exists")
			return
		}
		s.log.Error("signup: create account failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not create account")
		return
	}
	s.startSession(w, r, acct.ID, "email_signup")
}

// handleLogin: POST /auth/login — sign in with email OR username + password (F009). The
// `email` field carries either identifier (kept as the JSON key for back-compat); an '@'
// resolves it as an email, otherwise as a username. A uniform 401 on any failure so the
// response never reveals whether an identifier is registered — uniform in time too (L3):
// an unknown identifier, a password-less account and a suspended one each run exactly one
// bcrypt compare, against the startup dummy hash. When both bcrypt slots are busy the
// answer is 429 whatever the identifier.
func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	acct, err := s.resolveLoginIdentifier(r, body.Email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.log.Error("login: account lookup failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not sign in")
		return
	}
	// hash is "" (→ the dummy compare) unless there is a real, active password account.
	hash := ""
	if err == nil && acct.Status != store.StatusSuspended {
		hash = acct.PasswordHash
	}
	ok, cerr := s.pw.check(hash, body.Password)
	if cerr != nil {
		if s.bcryptBusy(w, cerr) {
			return
		}
		s.log.Error("login: password check failed", "err", cerr)
		writeError(w, http.StatusInternalServerError, "internal", "could not sign in")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect email/username or password")
		return
	}
	s.startSession(w, r, acct.ID, "email_login")
}

// bcryptBusy writes L3's 429 when err is errBcryptBusy and reports whether it did.
func (s *Service) bcryptBusy(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, errBcryptBusy) {
		return false
	}
	w.Header().Set("Retry-After", "1")
	writeError(w, http.StatusTooManyRequests, "too_many_requests", "too many sign-in attempts right now; try again in a moment")
	return true
}

// resolveLoginIdentifier looks up the account for an email-or-username login identifier
// (F009): an '@' means email (case-insensitive), otherwise username (case-insensitive).
func (s *Service) resolveLoginIdentifier(r *http.Request, identifier string) (store.Account, error) {
	if strings.Contains(identifier, "@") {
		return s.store.GetAccountByEmail(r.Context(), normalizeEmail(identifier))
	}
	return s.store.GetAccountByUsername(r.Context(), normalizeUsername(identifier))
}

// startSession mints a server session + cookie and returns 200 {ok:true}. The email flows
// are fetch-based (not a browser redirect), so the SPA re-fetches /me and routes in.
func (s *Service) startSession(w http.ResponseWriter, r *http.Request, accountID, kind string) {
	sid := newSessionID()
	if _, err := s.store.CreateSession(r.Context(), sid, accountID, time.Now().Add(s.cfg.Auth.SessionTTL)); err != nil {
		s.log.Error("auth: create session failed", "kind", kind, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not start session")
		return
	}
	setSessionCookie(w, sid, s.cfg.Auth.CookieSecure, s.cfg.Auth.SessionTTL)
	s.log.Info("auth login", "kind", kind, "account_id", accountID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleSetPassword: POST /accounts/{id}/password (JWT) — set or change the password. An
// account that already has one must supply the correct current password; an OAuth-only
// account (no hash) sets it for the first time without one. On success every session of
// the account is revoked, the caller's included (ADR-0033 §12 row 4), and the answer is
// {ok:true, reauth:true}: the SPA sends the learner to sign in again.
func (s *Service) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFrom(r.Context())
	id := r.PathValue("id")
	if claims.Subject != id {
		writeError(w, http.StatusForbidden, "forbidden", "account does not match token subject")
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	if err := validatePassword(body.NewPassword); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "weak_password", err.Error())
		return
	}
	acct, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	if acct.PasswordHash != "" {
		ok, err := s.pw.check(acct.PasswordHash, body.CurrentPassword)
		if err != nil {
			if s.bcryptBusy(w, err) {
				return
			}
			s.log.Error("set password: check failed", "err", err)
			writeError(w, http.StatusInternalServerError, "internal", "could not set password")
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "wrong_password", "your current password is incorrect")
			return
		}
	}
	hash, err := s.pw.hash(body.NewPassword)
	if err != nil {
		if s.bcryptBusy(w, err) {
			return
		}
		s.log.Error("set password: hash failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not set password")
		return
	}
	_, revoked, err := s.store.SetAccountPassword(r.Context(), id, hash)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	s.log.Info("password set; sessions revoked", "account_id", id, "revoked_sessions", revoked)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "reauth": true})
}

// handleUnlinkOAuth: DELETE /accounts/{id}/oauth/{provider} (JWT) — disconnect a provider,
// but never remove the account's LAST sign-in method (which would lock the user out). An
// account may unlink only if it keeps a password or another linked provider.
func (s *Service) handleUnlinkOAuth(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFrom(r.Context())
	id := r.PathValue("id")
	provider := r.PathValue("provider")
	if claims.Subject != id {
		writeError(w, http.StatusForbidden, "forbidden", "account does not match token subject")
		return
	}
	acct, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	providers, err := s.store.ListOAuthProviders(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	remaining := 0
	for _, p := range providers {
		if p != provider {
			remaining++
		}
	}
	if acct.PasswordHash == "" && remaining == 0 {
		writeError(w, http.StatusConflict, "last_login_method", "set a password before disconnecting your only sign-in method")
		return
	}
	removed, err := s.store.UnlinkOAuth(r.Context(), id, provider)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	if !removed {
		writeError(w, http.StatusNotFound, "not_linked", "that provider isn't connected")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
