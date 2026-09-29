// Package identity is the xLearn identity service: OAuth 2.0/OIDC sign-in with
// GitHub & Google, server-side sessions behind an opaque HttpOnly cookie, accounts
// and onboarding state, and the account_created transactional outbox (ADR-0006,
// ADR-0004/0005). It is a ClusterIP-only internal service; the gateway is the only
// caller. This file resolves its 12-factor configuration from the environment.
package identity

import (
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved identity configuration.
type Config struct {
	Port     string
	LogLevel string

	DB   DBConfig
	JWT  JWTVerifyConfig
	Auth AuthConfig
	NATS NATSConfig
}

// NATSConfig points identity's outbox relay at JetStream (XLEARN_IDENTITY, mi-05). An
// empty URL keeps the log publisher: prod identity has no NATS_URL until mi-06's
// identity N2 PR adds it together with the nkey seed (ADR-0035 §2), so its first prod
// connection already uses its own nkey. docker-compose sets it.
type NATSConfig struct {
	URL string
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-db secret; ADR-0005/0009). The role is xlearn_identity, scoped to schema
// identity.
type DBConfig struct {
	Host       string
	Port       string
	Database   string
	User       string
	Password   string
	SSLMode    string
	SearchPath string
}

// JWTVerifyConfig verifies the gateway-minted JWT on protected internal routes via
// the gateway's JWKS (ADR-0006).
type JWTVerifyConfig struct {
	JWKSURL  string
	Audience string // this service's expected aud
	Issuer   string // "" to skip issuer check
}

// AuthConfig holds the OAuth providers and cookie/session policy. v1 ships GitHub
// only; Google (ADR-0006) is deferred until its OAuth app is registered — re-add it
// as one more provider entry (newProviders) + a Config field when ready.
type AuthConfig struct {
	// PublicBaseURL is the external base the SPA + API are served under, e.g.
	// https://projects.sujaykumar.dev/xlearn — used to build OAuth redirect URIs
	// (<base>/api/auth/{provider}/callback) and the post-login redirect (<base>/auth).
	PublicBaseURL string
	GitHub        OAuthClient
	CookieSecure  bool
	SessionTTL    time.Duration
	// DevAuth enables the local-only dev login (POST /auth/dev/login), which mints a
	// session for a fixed local account WITHOUT OAuth. Off unless DEV_AUTH is truthy;
	// it must never be set in a prod image (F002 / ADR-0022).
	DevAuth bool
	// Signup gates creating NEW accounts (ADR-0023 §2, amended 2026-09-24): email sign-up
	// and a first OAuth sign-in that matches no account. Existing accounts sign in as usual.
	// It is the EFFECTIVE mode (resolveSignupMode): `open` only with DEV_AUTH (L7).
	Signup SignupMode
	// SignupRequested is SIGNUP_MODE as set (trimmed, lower-cased), and SignupNote says
	// why the effective mode differs from it ("" when it doesn't). LogSignupMode logs it.
	SignupRequested string
	SignupNote      string
	// SeatCap is SEAT_CAP (ADR-0033 §3): active learners allowed. Only the admin CLI's
	// reactivate / set-role learner read it in M1b; invites (L-A) join. Default 15.
	SeatCap int
}

// DefaultSeatCap is SEAT_CAP's code default (ADR-0033 §3 initial value).
const DefaultSeatCap = 15

// SignupMode is the effective sign-up mode. v2 names three (ADR-0033 §3): closed,
// invite and open; identity runs only closed and open until L-A builds invite.
type SignupMode string

const (
	SignupOpen   SignupMode = "open"
	SignupClosed SignupMode = "closed"
	// SignupInvite is accepted as a value but runs as closed until L-A (l-03) builds
	// the invite flow.
	SignupInvite SignupMode = "invite"
)

// resolveSignupMode is the L7 guard (ADR-0033 §3, ADR-0035 §4): `open` is honoured only
// with DEV_AUTH set (compose); without it identity runs closed and says why (logged at
// ERROR — production at `open` would be D21's open-signup trigger, a code change behind
// a new ADR, never an env flip). `invite` runs closed until L-A. Unset or empty is
// closed with no note; anything else is closed with a note. The note never echoes more
// than the normalized value.
func resolveSignupMode(raw string, devAuth bool) (SignupMode, string) {
	switch SignupMode(normalizeSignupMode(raw)) {
	case SignupOpen:
		if devAuth {
			return SignupOpen, ""
		}
		return SignupClosed, "SIGNUP_MODE=open ignored without DEV_AUTH; running closed"
	case SignupInvite:
		return SignupClosed, "SIGNUP_MODE=invite runs closed until the invite flow ships (L-A)"
	case SignupClosed, "":
		return SignupClosed, ""
	default:
		return SignupClosed, "SIGNUP_MODE has an unknown value; running closed"
	}
}

func normalizeSignupMode(raw string) string { return strings.ToLower(strings.TrimSpace(raw)) }

// LogSignupMode logs the effective sign-up mode once at startup (D34: a log line, no
// alert): ERROR when `open` was asked for without DEV_AUTH (the L7 misconfiguration),
// INFO for any other note.
func (a AuthConfig) LogSignupMode(log *slog.Logger) {
	switch {
	case a.SignupNote == "":
		return
	case SignupMode(a.SignupRequested) == SignupOpen && a.Signup != SignupOpen:
		log.Error(a.SignupNote, "signup_mode", a.Signup, "requested", a.SignupRequested)
	default:
		log.Info(a.SignupNote, "signup_mode", a.Signup)
	}
}

// OAuthClient is a provider's registered app credentials.
type OAuthClient struct {
	ClientID     string
	ClientSecret string
}

// Configured reports whether the provider has credentials (so the SPA/handlers can
// 503 cleanly rather than sending an empty client_id to the provider).
func (c OAuthClient) Configured() bool { return c.ClientID != "" && c.ClientSecret != "" }

// LoadConfig reads the environment and returns identity's configuration. Defaults
// suit local dev (gateway on :8080, identity on :8081); k8s overrides via env.
func LoadConfig() Config {
	base := env("PUBLIC_BASE_URL", "http://localhost:8080/xlearn")
	devAuth := envBool("DEV_AUTH", false)
	signupRaw := os.Getenv("SIGNUP_MODE")
	signup, signupNote := resolveSignupMode(signupRaw, devAuth)
	return Config{
		Port:     env("PORT", "8081"),
		LogLevel: env("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_identity"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "identity"),
		},
		JWT: JWTVerifyConfig{
			JWKSURL:  env("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
			Audience: env("JWT_AUDIENCE", "identity"),
			Issuer:   env("JWT_ISSUER", "xlearn-gateway"),
		},
		Auth: AuthConfig{
			PublicBaseURL: strings.TrimRight(base, "/"),
			GitHub: OAuthClient{
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			},
			// Secure cookies whenever the public base is https (prod); plain http
			// (local dev) gets non-Secure so the browser will store the cookie.
			// Overridable with COOKIE_SECURE.
			CookieSecure:    envBool("COOKIE_SECURE", strings.HasPrefix(base, "https://")),
			SessionTTL:      envDuration("SESSION_TTL", 30*24*time.Hour),
			DevAuth:         devAuth,
			Signup:          signup,
			SignupRequested: normalizeSignupMode(signupRaw),
			SignupNote:      signupNote,
			SeatCap:         envPositiveInt("SEAT_CAP", DefaultSeatCap),
		},
		NATS: NATSConfig{
			URL: strings.TrimSpace(os.Getenv("NATS_URL")),
		},
	}
}

// Addr is the listen address for http.Server.
func (c Config) Addr() string { return ":" + c.Port }

// DSN builds the Postgres connection URL from the DB config. User and password are
// percent-encoded via net/url so a role password containing URL-reserved bytes
// (e.g. '/' or '@' from a random base64 secret) can't corrupt the authority — a
// raw-concatenated DSN would misparse and crashloop the service on such passwords.
func (d DBConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   "/" + d.Database,
	}
	if d.Password != "" {
		u.User = url.UserPassword(d.User, d.Password)
	} else {
		u.User = url.User(d.User)
	}
	q := url.Values{}
	q.Set("sslmode", d.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// envPositiveInt reads a positive integer, falling back to def when unset or invalid.
func envPositiveInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
