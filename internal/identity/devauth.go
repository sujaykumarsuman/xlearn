package identity

import (
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// Dev login is a LOCAL-ONLY sign-in that mints a session for a fixed account without
// OAuth, so a reviewer can drive the app from docker-compose (F002 / ADR-0022). Every
// handler here is a 404 no-op unless cfg.Auth.DevAuth is set — which must never be true
// in a prod image. It adds no trust surface in prod: the routes exist but refuse.
const (
	// Reuse an allowed provider enum value (the oauth_identity CHECK is github|google);
	// the fixed provider_user_id marks the row as the local dev account.
	devProvider    = "github"
	devProviderUID = "dev-local"
	devDisplayName = "Dev User"
	devEmail       = "dev@localhost"
)

// handleDevEnabled reports whether dev login is available, so the SPA shows the button
// only when it is. Disabled → 404, indistinguishable from "no such route".
func (s *Service) handleDevEnabled(w http.ResponseWriter, _ *http.Request) {
	if !s.cfg.Auth.DevAuth {
		writeError(w, http.StatusNotFound, "not_found", "dev login is disabled")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true})
}

// handleDevLogin finds/creates the fixed dev account, completes its onboarding (so the
// reviewer lands straight in the app rather than the first-run flow), mints a session
// and sets the cookie — mirroring the OAuth callback's session issuance. Disabled → 404.
func (s *Service) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Auth.DevAuth {
		writeError(w, http.StatusNotFound, "not_found", "dev login is disabled")
		return
	}
	ctx := r.Context()
	acct, _, err := s.store.FindOrCreateAccount(ctx, store.OAuthUpsert{
		Provider:       devProvider,
		ProviderUserID: devProviderUID,
		DisplayName:    devDisplayName,
		Email:          devEmail,
	})
	if err != nil {
		s.log.Error("dev login: account failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "dev login failed")
		return
	}
	if _, err := s.store.CompleteOnboarding(ctx, acct.ID); err != nil {
		s.log.Error("dev login: complete onboarding failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "dev login failed")
		return
	}
	sid := newSessionID()
	if _, err := s.store.CreateSession(ctx, sid, acct.ID, time.Now().Add(s.cfg.Auth.SessionTTL)); err != nil {
		s.log.Error("dev login: create session failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "dev login failed")
		return
	}
	setSessionCookie(w, sid, s.cfg.Auth.CookieSecure, s.cfg.Auth.SessionTTL)
	s.log.Info("dev login", "account_id", acct.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
