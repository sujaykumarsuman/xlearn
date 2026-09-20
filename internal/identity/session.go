package identity

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
)

// Cookie names + paths. The session cookie is scoped to /xlearn (ADR-0006); the
// short-lived transaction cookie carries the in-flight OAuth state + PKCE verifier
// between /start and /callback.
const (
	oauthTxCookieName = "xl_oauthtx"
	cookiePath        = "/xlearn"
	oauthTxTTL        = 10 * time.Minute
)

// newSessionID returns a 256-bit opaque session token (the cookie value + the
// session row's primary key).
func newSessionID() string { return randToken(32) }

// oauthTx is the in-flight OAuth state persisted (HttpOnly) between start/callback.
type oauthTx struct {
	Provider string `json:"p"`
	State    string `json:"s"`
	Verifier string `json:"v"`
}

// setSessionCookie writes the opaque session cookie (HttpOnly, Secure, SameSite=Lax,
// path /xlearn), expiring with the session.
func setSessionCookie(w http.ResponseWriter, value string, secure bool, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    value,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

// clearSessionCookie expires the session cookie.
func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// setOAuthTxCookie persists the in-flight OAuth transaction (base64 JSON). It is
// HttpOnly + SameSite=Lax so it survives the top-level GET redirect back from the
// provider but is invisible to JS.
func setOAuthTxCookie(w http.ResponseWriter, tx oauthTx, secure bool) {
	raw, _ := json.Marshal(tx)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthTxCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(raw),
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthTxTTL.Seconds()),
	})
}

// readOAuthTxCookie decodes and returns the in-flight OAuth transaction.
func readOAuthTxCookie(r *http.Request) (oauthTx, bool) {
	c, err := r.Cookie(oauthTxCookieName)
	if err != nil {
		return oauthTx{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return oauthTx{}, false
	}
	var tx oauthTx
	if err := json.Unmarshal(raw, &tx); err != nil {
		return oauthTx{}, false
	}
	return tx, true
}

// clearOAuthTxCookie expires the transaction cookie.
func clearOAuthTxCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthTxCookieName,
		Value:    "",
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
