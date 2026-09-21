// Package review is the xLearn review service: the five-touch spaced-repetition
// scheduler and the Revision queue — the first async CONSUMER in the system. It
// reacts to practice events (problem_solved / solution_revealed_early) on NATS
// JetStream to schedule the Day 1/3/7/21/45 ladder, auto-scores each re-solve
// (advance or reset-to-Day-1), and materialises "due today" with a periodic sweep,
// emitting revision_scheduled / revision_due via a transactional outbox
// (ADR-0004/0005/0006). It is a ClusterIP-only internal service; the gateway is the
// only HTTP caller. This file resolves its 12-factor configuration.
package review

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved review configuration.
type Config struct {
	Port     string
	LogLevel string

	DB    DBConfig
	JWT   JWTVerifyConfig
	NATS  NATSConfig
	Sweep SweepConfig

	// CurriculumBaseURL is the curriculum service's internal URL. Used by the mistake
	// journal to pre-fill a problem's pattern (a soft reference; ADR-0005). Empty →
	// patterns are left blank (best-effort).
	CurriculumBaseURL string
	// IdentityBaseURL is the identity service's internal URL. Used by the background
	// workers (weak-area recompute + notifications) to resolve account timezone +
	// study-budget without a user JWT (ADR-0016). Empty → UTC / immediate reminders.
	IdentityBaseURL string
}

// DBConfig is the Postgres connection (coords as plain env, creds from the SOPS
// xlearn-review-db secret; ADR-0005/0009). The role is xlearn_review, scoped to
// schema review.
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
	Audience string // this service's expected aud ("review")
	Issuer   string // "" to skip the issuer check
}

// NATSConfig points both the outbox relay (publish to XLEARN_REVIEW) and the durable
// consumers (subscribe to XLEARN_PRACTICE) at the JetStream backbone (ADR-0004). An
// empty URL disables NATS (local dev): the relay falls back to the log publisher and
// no consumer runs, so the service still starts with no broker.
type NATSConfig struct {
	URL string
}

// SweepConfig tunes the periodic due-sweep (flow 4).
type SweepConfig struct {
	Interval time.Duration
	Batch    int
}

// LoadConfig reads the environment and returns review's configuration. Defaults suit
// local dev (gateway :8080 … practice :8083, review :8084); k8s overrides via env.
func LoadConfig() Config {
	return Config{
		Port:     env("PORT", "8084"),
		LogLevel: env("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:       env("PGHOST", "localhost"),
			Port:       env("PGPORT", "5432"),
			Database:   env("PGDATABASE", "xlearndb"),
			User:       env("PGUSER", "xlearn_review"),
			Password:   os.Getenv("PGPASSWORD"),
			SSLMode:    env("PGSSLMODE", "prefer"),
			SearchPath: env("PGSEARCHPATH", "review"),
		},
		JWT: JWTVerifyConfig{
			JWKSURL:  env("JWKS_URL", "http://localhost:8080/.well-known/jwks.json"),
			Audience: env("JWT_AUDIENCE", "review"),
			Issuer:   env("JWT_ISSUER", "xlearn-gateway"),
		},
		NATS: NATSConfig{
			URL: strings.TrimSpace(os.Getenv("NATS_URL")),
		},
		Sweep: SweepConfig{
			Interval: envDuration("REVIEW_SWEEP_INTERVAL", 15*time.Minute),
			Batch:    envInt("REVIEW_SWEEP_BATCH", 500),
		},
		CurriculumBaseURL: strings.TrimSpace(os.Getenv("CURRICULUM_BASE_URL")),
		IdentityBaseURL:   strings.TrimSpace(os.Getenv("IDENTITY_BASE_URL")),
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

func envDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
