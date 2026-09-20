// Package curriculum is the xLearn curriculum service: the read-heavy content rail
// — paths, phases, weeks, concepts, problems and their stage-scoped sections
// (ADR-0005). It is a ClusterIP-only internal service; the gateway is the only
// caller. Content is seeded idempotently on startup from the versioned files under
// curriculum/. Curriculum emits/consumes no events (no outbox). This file resolves
// its 12-factor configuration from the environment.
package curriculum

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// Config is the resolved curriculum configuration.
type Config struct {
	Port     string
	LogLevel string
	DB       DBConfig
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-curriculum-db secret; ADR-0005/0009). The role is xlearn_curriculum,
// scoped to schema curriculum.
type DBConfig struct {
	Host       string
	Port       string
	Database   string
	User       string
	Password   string
	SSLMode    string
	SearchPath string
}

// LoadConfig reads the environment and returns curriculum's configuration.
// Defaults suit local dev (gateway :8080, identity :8081, curriculum :8082); k8s
// overrides via env.
func LoadConfig() Config {
	return Config{
		Port:     env("PORT", "8082"),
		LogLevel: env("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_curriculum"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "curriculum"),
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
