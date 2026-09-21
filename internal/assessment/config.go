// Package assessment is the xLearn assessment service: the timed mock-interview
// aggregate (a 45-minute, server-authoritative phase-rail session scored on the
// 7-dimension rubric /35) plus the consume seam for the S09 progress projections. It
// emits mock_completed via a transactional outbox to NATS JetStream (XLEARN_ASSESSMENT)
// and subscribes durable pull consumers to XLEARN_PRACTICE / XLEARN_REVIEW whose
// projection handler bodies are no-op stubs this sprint (ADR-0004/0005/0006). It is a
// ClusterIP-only internal service; the gateway is the only HTTP caller. This file
// resolves its 12-factor configuration.
package assessment

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// Config is the resolved assessment configuration.
type Config struct {
	Port     string
	LogLevel string

	DB   DBConfig
	JWT  JWTVerifyConfig
	NATS NATSConfig
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-assessment-db secret; ADR-0005/0009). The role is xlearn_assessment, scoped
// to schema assessment.
type DBConfig struct {
	Host       string
	Port       string
	Database   string
	User       string
	Password   string
	SSLMode    string
	SearchPath string
}

// JWTVerifyConfig verifies the gateway-minted JWT on the user routes via the
// gateway's JWKS (ADR-0006).
type JWTVerifyConfig struct {
	JWKSURL  string
	Audience string // this service's expected aud ("assessment")
	Issuer   string // "" to skip the issuer check
}

// NATSConfig points both the outbox relay (publish to XLEARN_ASSESSMENT) and the
// durable consumers (subscribe to XLEARN_PRACTICE / XLEARN_REVIEW) at the JetStream
// backbone (ADR-0004). An empty URL disables NATS (local dev): the relay falls back to
// the log publisher and no consumer runs, so the service still starts with no broker.
type NATSConfig struct {
	URL string
}

// LoadConfig reads the environment and returns assessment's configuration. Defaults
// suit local dev (gateway :8080 … review :8084, assessment :8085); k8s overrides via env.
func LoadConfig() Config {
	return Config{
		Port:     env("PORT", "8085"),
		LogLevel: env("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_assessment"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "assessment"),
		},
		JWT: JWTVerifyConfig{
			JWKSURL:  env("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
			Audience: env("JWT_AUDIENCE", "assessment"),
			Issuer:   env("JWT_ISSUER", "xlearn-gateway"),
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
