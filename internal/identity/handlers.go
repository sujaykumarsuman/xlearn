package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// claimsCtxKey carries verified JWT claims from requireJWT to the handler.
type claimsCtxKey struct{}

// --- OAuth: start + callback ---

func (s *Service) handleStart(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	p, ok := s.providers[provider]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown_provider", "unknown OAuth provider")
		return
	}
	if !p.client.Configured() {
		writeError(w, http.StatusServiceUnavailable, "provider_unavailable", "OAuth provider is not configured")
		return
	}

	state := newState()
	verifier, challenge := newPKCE()
	setOAuthTxCookie(w, oauthTx{Provider: provider, State: state, Verifier: verifier}, s.cfg.Auth.CookieSecure)

	authURL := p.authCodeURL(s.callbackURL(provider), state, challenge)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Service) handleCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	p, ok := s.providers[provider]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown_provider", "unknown OAuth provider")
		return
	}

	tx, haveTx := readOAuthTxCookie(r)
	clearOAuthTxCookie(w, s.cfg.Auth.CookieSecure)

	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		s.log.Warn("oauth callback: provider returned error", "provider", provider, "error", e)
		s.redirectToAuth(w, r, "denied")
		return
	}
	if !haveTx || tx.Provider != provider || tx.State == "" || q.Get("state") != tx.State {
		s.log.Warn("oauth callback: state mismatch", "provider", provider)
		s.redirectToAuth(w, r, "oauth_state")
		return
	}
	code := q.Get("code")
	if code == "" {
		s.redirectToAuth(w, r, "oauth_code")
		return
	}

	ctx := r.Context()
	token, err := p.exchange(ctx, s.httpc, code, s.callbackURL(provider), tx.Verifier)
	if err != nil {
		s.log.Error("oauth callback: code exchange failed", "provider", provider, "err", err)
		s.redirectToAuth(w, r, "oauth_exchange")
		return
	}
	prof, err := p.userProfile(ctx, s.httpc, token)
	if err != nil {
		s.log.Error("oauth callback: userinfo failed", "provider", provider, "err", err)
		s.redirectToAuth(w, r, "oauth_profile")
		return
	}

	acct, created, err := s.store.FindOrCreateAccount(ctx, store.OAuthUpsert{
		Provider:       provider,
		ProviderUserID: prof.ProviderUserID,
		DisplayName:    prof.DisplayName,
		Email:          prof.Email,
	})
	if err != nil {
		s.log.Error("oauth callback: find-or-create account failed", "provider", provider, "err", err)
		s.redirectToAuth(w, r, "account")
		return
	}

	sid := newSessionID()
	if _, err := s.store.CreateSession(ctx, sid, acct.ID, time.Now().Add(s.cfg.Auth.SessionTTL)); err != nil {
		s.log.Error("oauth callback: create session failed", "err", err)
		s.redirectToAuth(w, r, "session")
		return
	}
	setSessionCookie(w, sid, s.cfg.Auth.CookieSecure, s.cfg.Auth.SessionTTL)
	s.log.Info("oauth login", "provider", provider, "account_id", acct.ID, "created", created)

	http.Redirect(w, r, s.cfg.Auth.PublicBaseURL+"/auth", http.StatusFound)
}

func (s *Service) callbackURL(provider string) string {
	return s.cfg.Auth.PublicBaseURL + "/api/auth/" + provider + "/callback"
}

// redirectToAuth sends the browser back to the SPA auth screen with an error hint.
func (s *Service) redirectToAuth(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, s.cfg.Auth.PublicBaseURL+"/auth?error="+reason, http.StatusFound)
}

// --- Session trust endpoints (gateway → identity) ---

func (s *Service) handleValidateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.SessionID == "" {
		writeUnauthenticated(w)
		return
	}
	sess, err := s.store.GetValidSession(r.Context(), body.SessionID)
	if err != nil {
		writeUnauthenticated(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account_id": sess.AccountID,
		"expires_at": sess.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Service) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.SessionID == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if _, err := s.store.RevokeSession(r.Context(), body.SessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "could not revoke session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- User-data routes (JWT-protected) ---

func (s *Service) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	id := r.PathValue("id")
	if claims.Subject != id {
		writeError(w, http.StatusForbidden, "forbidden", "account does not match token subject")
		return
	}
	s.writeAccount(w, r, id)
}

// handleInternalGetAccount serves an account's scheduling preferences (timezone +
// study budget) to background workers over the ClusterIP (no user JWT; ADR-0016). It
// deliberately returns only what a worker needs — never OAuth identities, email or
// session data — and the raw jsonb blobs pass through unchanged.
func (s *Service) handleInternalGetAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	acct, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           acct.ID,
		"timezone":     acct.Timezone,
		"study_budget": rawJSONOrEmpty(acct.StudyBudget),
		"reminders":    rawJSONOrEmpty(acct.Reminders),
	})
}

// rawJSONOrEmpty passes a jsonb blob through as raw JSON (not a base64 string),
// defaulting an empty/nil blob to an empty object.
func rawJSONOrEmpty(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}

func (s *Service) handleOnboardingStep(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	var body struct {
		Step        string          `json:"step"`
		PathChosen  string          `json:"path_chosen"`
		StudyBudget json.RawMessage `json:"study_budget"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	switch body.Step {
	case "path":
		if body.PathChosen == "" {
			writeError(w, http.StatusUnprocessableEntity, "invalid_path", "path_chosen is required")
			return
		}
		ob, err := s.store.SetOnboardingPath(r.Context(), claims.Subject, body.PathChosen)
		if err != nil {
			s.mapStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"onboarding": toOnboardingJSON(ob)})
	case "budget":
		// Step 2: persist the study budget to the account and set onboarding.budget_set
		// (one transaction). Same validated shape as PATCH /me.
		budget, err := validateStudyBudget(body.StudyBudget)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid_budget", err.Error())
			return
		}
		ob, err := s.store.SetOnboardingBudget(r.Context(), claims.Subject, budget)
		if err != nil {
			s.mapStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"onboarding": toOnboardingJSON(ob)})
	case "finish":
		// Step 3 (Finish / Skip): stamp completed_at (idempotent). key_added stays false
		// until the coach key store works (S11) — we never fake it here.
		ob, err := s.store.CompleteOnboarding(r.Context(), claims.Subject)
		if err != nil {
			s.mapStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"onboarding": toOnboardingJSON(ob)})
	default:
		writeError(w, http.StatusBadRequest, "unsupported_step", "unknown onboarding step")
	}
}

func (s *Service) writeAccount(w http.ResponseWriter, r *http.Request, id string) {
	acct, err := s.store.GetAccount(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	ob, err := s.store.GetOnboarding(r.Context(), id)
	if err != nil {
		s.mapStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"account":    toAccountJSON(acct),
		"onboarding": toOnboardingJSON(ob),
	})
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

func (s *Service) mapStoreErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	s.log.Error("store error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal error")
}

// --- JSON response shapes + helpers ---

type accountJSON struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"display_name"`
	Email       string          `json:"email,omitempty"`
	Timezone    string          `json:"timezone"`
	StudyBudget json.RawMessage `json:"study_budget"`
	Reminders   json.RawMessage `json:"reminders"`
	CreatedAt   time.Time       `json:"created_at"`
}

type onboardingJSON struct {
	PathChosen *string `json:"path_chosen"`
	BudgetSet  bool    `json:"budget_set"`
	KeyAdded   bool    `json:"key_added"`
	Completed  bool    `json:"completed"`
}

func toAccountJSON(a store.Account) accountJSON {
	return accountJSON{
		ID:          a.ID,
		DisplayName: a.DisplayName,
		Email:       a.Email,
		Timezone:    a.Timezone,
		// The learner's own budget/reminder prefs, returned only to the owner via the
		// JWT-gated /me so the Settings form can load its current values (S10).
		StudyBudget: rawJSONOrEmpty(a.StudyBudget),
		Reminders:   rawJSONOrEmpty(a.Reminders),
		CreatedAt:   a.CreatedAt.UTC(),
	}
}

func toOnboardingJSON(o store.Onboarding) onboardingJSON {
	var path *string
	if o.PathChosen != "" {
		p := o.PathChosen
		path = &p
	}
	return onboardingJSON{
		PathChosen: path,
		BudgetSet:  o.BudgetSet,
		KeyAdded:   o.KeyAdded,
		Completed:  !o.CompletedAt.IsZero(),
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
