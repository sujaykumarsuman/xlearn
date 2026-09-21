// Package practice is the xLearn practice service: the per-user problem lifecycle —
// stage gating (attempt→hint→solution→re-implement→log), server-authoritative
// timers (15/10 min), the reveal-penalty rule, and outcome logging
// (Clean/Rough/Assisted/Miss) — plus the first event producer, emitting
// problem_solved / attempt_logged / solution_revealed_early to NATS JetStream via a
// transactional outbox (ADR-0004/0005/0006). It is a ClusterIP-only internal
// service; the gateway is the only caller. This file resolves its 12-factor
// configuration from the environment.
package practice

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// Config is the resolved practice configuration.
type Config struct {
	Port     string
	LogLevel string

	DB   DBConfig
	JWT  JWTVerifyConfig
	NATS NATSConfig
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-practice-db secret; ADR-0005/0009). The role is xlearn_practice, scoped to
// schema practice.
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
	Audience string // this service's expected aud ("practice")
	Issuer   string // "" to skip the issuer check
}

// NATSConfig points the outbox relay at the JetStream backbone (ADR-0004). An empty
// URL disables the JetStream publisher and falls back to the log publisher (local
// dev), so the service runs with no broker; k8s sets NATS_URL in prod.
type NATSConfig struct {
	URL string
}

// LoadConfig reads the environment and returns practice's configuration. Defaults
// suit local dev (gateway :8080, identity :8081, curriculum :8082, practice :8083);
// k8s overrides via env.
func LoadConfig() Config {
	return Config{
		Port:     env("PORT", "8083"),
		LogLevel: env("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_practice"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "practice"),
		},
		JWT: JWTVerifyConfig{
			JWKSURL:  env("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
			Audience: env("JWT_AUDIENCE", "practice"),
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
