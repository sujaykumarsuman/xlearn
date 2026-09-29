package gateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/gateway/limit"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
)

// The gateway's in-process request limits (ADR-0035 §4; sprint m1-05). They live here, not
// in Traefik, because Traefik can't key by the account behind the session cookie. Every
// denial is a typed 429 with Retry-After (limit.WriteTooMany), never a silent drop.
//
// Values are ADR-0035's start values. Bursts were checked against the SPA's measured
// cold-load fan-out (m1-05, browser network log on the compose stack): the worst screen
// (Roadmap) issues 7 /api calls within ~50 ms, the core screens 6, and a public profile
// load 1 call to /api/u/{username}. Every burst below is ≥ 2× that.
//
// All of this state is process-local: with the single gateway replica that is correct,
// and it is why L24 lists it as a scale-out blocker (docs/architecture/services.md).
//
// Exempt by construction: /healthz, /readyz, /api/healthz, /.well-known/jwks.json and the
// SPA's static assets never pass through any limit below.
var (
	// L1 login: 10/min per IP (burst 5) …
	loginIPRate  = limit.PerMinute(10)
	loginIPBurst = 5
	// … and 5 failures / 15 min per identifier (email or username, F009).
	loginMaxFailures   = 5
	loginFailureWindow = 15 * time.Minute
	// L2 signup and OAuth start: 5/min per IP. (The invite check joins in L-A, l-03.)
	signupIPRate  = limit.PerMinute(5)
	signupIPBurst = 5
	// L4 public profile: 60/min per IP (burst 20).
	publicIPRate  = limit.PerMinute(60)
	publicIPBurst = 20
	// L5 per account: every /api request 20/s (burst 40); mutating methods also 5/s (burst 10).
	accountRate       = limit.PerSecond(20)
	accountBurst      = 40
	accountWriteRate  = limit.PerSecond(5)
	accountWriteBurst = 10
)

// loginBodyLimit caps the login body the gateway buffers to read the identifier (L1); a
// real {"email","password"} body is well under 1 KiB.
const loginBodyLimit int64 = 8 << 10

// gatewayLimits is the gateway's limiter state (one per Gateway).
type gatewayLimits struct {
	loginIP      *limit.Limiter
	loginFails   *limit.FailureWindow
	signupIP     *limit.Limiter
	publicIP     *limit.Limiter
	account      *limit.Limiter
	accountWrite *limit.Limiter
}

func newGatewayLimits(now limit.Clock) *gatewayLimits {
	clk := limit.WithClock(now)
	return &gatewayLimits{
		loginIP:      limit.New(loginIPRate, loginIPBurst, clk),
		loginFails:   limit.NewFailureWindow(loginMaxFailures, loginFailureWindow, clk),
		signupIP:     limit.New(signupIPRate, signupIPBurst, clk),
		publicIP:     limit.New(publicIPRate, publicIPBurst, clk),
		account:      limit.New(accountRate, accountBurst, clk),
		accountWrite: limit.New(accountWriteRate, accountWriteBurst, clk),
	}
}

// allowIP takes a token for the request's client IP (limit.ClientIP: X-Real-Ip, else the
// RemoteAddr host) under key prefix route, or writes the typed 429 and returns false.
func allowIP(w http.ResponseWriter, r *http.Request, l *limit.Limiter, route string) bool {
	ok, wait := l.Allow(route + "|" + limit.ClientIP(r))
	if !ok {
		limit.WriteTooMany(w, limit.CodeRateLimited, wait)
	}
	return ok
}

// allowAccount is L5, applied once per request at authSession's post-validation point, so
// every authed route is covered without per-route wiring (the coach SSE chat counts once,
// at request start). A mutating request needs a token from both buckets.
func (g *Gateway) allowAccount(w http.ResponseWriter, r *http.Request, accountID string) bool {
	if isMutatingMethod(r.Method) {
		if ok, wait := g.limits.accountWrite.Allow(accountID); !ok {
			limit.WriteTooMany(w, limit.CodeRateLimited, wait)
			return false
		}
	}
	if ok, wait := g.limits.account.Allow(accountID); !ok {
		limit.WriteTooMany(w, limit.CodeRateLimited, wait)
		return false
	}
	return true
}

// loginIdentifierKey is the L1 failure-window key for a login body: the SHA-256 of the
// normalised `email` field (email or username; trimmed and lowercased as identity does).
// The raw identifier is never kept in memory or logged. "" when there is no identifier
// (identity answers that with a 400, which the window doesn't count).
func loginIdentifierKey(body []byte) string {
	var b struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &b) != nil {
		return ""
	}
	id := strings.ToLower(strings.TrimSpace(b.Email))
	if id == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

// loginWithLimits is L1 around the login proxy: the per-IP bucket, then the per-identifier
// failure window (at loginMaxFailures failures the gateway answers 429 without calling
// identity until the oldest failure ages out), then the forward; identity's 401 counts a
// failure and a 2xx clears the identifier. identity's uniform 401 and L3's timing are
// untouched.
func (g *Gateway) loginWithLimits(w http.ResponseWriter, r *http.Request) {
	if !allowIP(w, r, g.limits.loginIP, "login") {
		return
	}
	body, ok := httpx.ReadBody(w, r, loginBodyLimit)
	if !ok {
		return
	}
	key := loginIdentifierKey(body)
	if key != "" {
		if blocked, wait := g.limits.loginFails.Blocked(key); blocked {
			limit.WriteTooMany(w, limit.CodeRateLimited, wait)
			return
		}
	}
	fr := r.Clone(r.Context())
	fr.Body = io.NopCloser(bytes.NewReader(body))
	fr.ContentLength = int64(len(body))
	rec := &statusRecorder{ResponseWriter: w}
	g.identity.forward(rec, fr, "/auth/login")
	if key == "" {
		return
	}
	switch {
	case rec.status == http.StatusUnauthorized:
		g.limits.loginFails.Fail(key)
	case rec.status >= 200 && rec.status < 300:
		g.limits.loginFails.Reset(key)
	}
}

// statusRecorder remembers the status a handler wrote (L1 counts identity's 401s).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
