package identity

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// --- Email/password auth (ADR-0023) ---

// handleSignup: POST /auth/signup — create an email/password account + session. No email
// verification yet (deferred), so the account is usable immediately.
func (s *Service) handleSignup(w http.ResponseWriter, r *http.Request) {
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
	hash, err := hashPassword(body.Password)
	if err != nil {
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
// response never reveals whether an identifier is registered.
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
	if err != nil || acct.PasswordHash == "" || !checkPassword(acct.PasswordHash, body.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect email/username or password")
		return
	}
	s.startSession(w, r, acct.ID, "email_login")
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
// account (no hash) sets it for the first time without one.
func (s *Service) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
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
	if acct.PasswordHash != "" && !checkPassword(acct.PasswordHash, body.CurrentPassword) {
		writeError(w, http.StatusUnauthorized, "wrong_password", "your current password is incorrect")
		return
	}
	hash, err := hashPassword(body.NewPassword)
	if err != nil {
		s.log.Error("set password: hash failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not set password")
		return
	}
	if _, err := s.store.SetAccountPassword(r.Context(), id, hash); err != nil {
		s.mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleUnlinkOAuth: DELETE /accounts/{id}/oauth/{provider} (JWT) — disconnect a provider,
// but never remove the account's LAST sign-in method (which would lock the user out). An
// account may unlink only if it keeps a password or another linked provider.
func (s *Service) handleUnlinkOAuth(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
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
