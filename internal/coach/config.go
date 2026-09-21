// Package coach is the xLearn AI-coach service (ADR-0007): it stores each user's own
// provider API key ENVELOPE-ENCRYPTED at rest, builds page-context prompts server-side
// (Socratic + spoiler-free during an active attempt, code reviewer post-solve), and
// fans the chat out to the user's LLM provider (OpenAI / Anthropic), streaming the reply
// back over SSE. It never returns the raw key. It is a ClusterIP-only internal service
// (route.enabled: false); the gateway is the only HTTP caller. coach emits no events
// (no outbox) and consumes none via NATS (page context arrives on the request). This
// file resolves its 12-factor configuration.
package coach

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// Config is the resolved coach configuration.
type Config struct {
	Port     string
	LogLevel string

	// MasterKey is the raw 32-byte envelope master key (decoded from MasterKeyRaw). It is
	// never logged. Empty is fatal in prod (the service refuses to serve).
	MasterKeyRaw string

	DB        DBConfig
	JWT       JWTVerifyConfig
	Providers ProviderConfig
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-coach-db secret; ADR-0005/0009). The role is xlearn_coach, scoped to schema
// coach.
type DBConfig struct {
	Host       string
	Port       string
	Database   string
	User       string
	Password   string
	SSLMode    string
	SearchPath string
}

// JWTVerifyConfig verifies the gateway-minted JWT on the user routes via the gateway's
// JWKS (ADR-0006).
type JWTVerifyConfig struct {
	JWKSURL  string
	Audience string // this service's expected aud ("coach")
	Issuer   string // "" to skip the issuer check
}

// ProviderConfig points the provider clients at their base URLs. Defaults are the real
// provider endpoints; tests override them to a local httptest server so no unit/
// integration test ever hits a real provider or needs a real key.
type ProviderConfig struct {
	OpenAIBaseURL    string
	AnthropicBaseURL string
}

// LoadConfig reads the environment and returns coach's configuration. Defaults suit
// local dev (gateway :8080 … assessment :8085, coach :8086); k8s overrides via env.
func LoadConfig() Config {
	return Config{
		Port:         env("PORT", "8086"),
		LogLevel:     env("LOG_LEVEL", "info"),
		MasterKeyRaw: os.Getenv("COACH_MASTER_KEY"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_coach"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "coach"),
		},
		JWT: JWTVerifyConfig{
			JWKSURL:  env("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
			Audience: env("JWT_AUDIENCE", "coach"),
			Issuer:   env("JWT_ISSUER", "xlearn-gateway"),
		},
		Providers: ProviderConfig{
			OpenAIBaseURL:    strings.TrimRight(env("OPENAI_BASE_URL", "https://api.openai.com"), "/"),
			AnthropicBaseURL: strings.TrimRight(env("ANTHROPIC_BASE_URL", "https://api.anthropic.com"), "/"),
		},
	}
}

// MasterKeyFile returns the optional path to a mounted master-key file (COACH_MASTER_KEY_FILE).
func MasterKeyFile() string { return strings.TrimSpace(os.Getenv("COACH_MASTER_KEY_FILE")) }

// Addr is the listen address for http.Server.
func (c Config) Addr() string { return ":" + c.Port }

// DSN builds the Postgres connection URL. User + password are percent-encoded via
// net/url so a role password with URL-reserved bytes can't corrupt the authority.
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
