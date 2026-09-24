// Package identity is the xLearn identity service: OAuth 2.0/OIDC sign-in with
// GitHub & Google, server-side sessions behind an opaque HttpOnly cookie, accounts
// and onboarding state, and the account_created transactional outbox (ADR-0006,
// ADR-0004/0005). It is a ClusterIP-only internal service; the gateway is the only
// caller. This file resolves its 12-factor configuration from the environment.
package identity

import (
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
	Signup SignupMode
}

// SignupMode is SIGNUP_MODE. Anything other than "open" (unset, a typo) resolves to
// closed, so a missing env fails safe. v2 adds "invite".
type SignupMode string

const (
	SignupOpen   SignupMode = "open"
	SignupClosed SignupMode = "closed"
)

func parseSignupMode(v string) SignupMode {
	if SignupMode(strings.ToLower(strings.TrimSpace(v))) == SignupOpen {
		return SignupOpen
	}
	return SignupClosed
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
			CookieSecure: envBool("COOKIE_SECURE", strings.HasPrefix(base, "https://")),
			SessionTTL:   envDuration("SESSION_TTL", 30*24*time.Hour),
			DevAuth:      envBool("DEV_AUTH", false),
			Signup:       parseSignupMode(os.Getenv("SIGNUP_MODE")),
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

func envDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
