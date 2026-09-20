// Package config resolves a service's runtime settings from the environment
// (12-factor, per ADR-0009). Every knob has a documented default so the binary
// runs with no configuration in local dev; the k8s deployment sets what differs
// via env. A resolved Config is an immutable snapshot.
package config

import (
	"os"
	"strings"
)

// Defaults for the gateway. Each is overridable by the matching env var.
const (
	defaultPort     = "8080"
	defaultLogLevel = "info"
	defaultBasePath = "/xlearn"
)

// Config is the resolved gateway configuration.
type Config struct {
	// Port is the TCP port the HTTP server listens on (env PORT).
	Port string
	// LogLevel is the slog level: debug|info|warn|error (env LOG_LEVEL).
	LogLevel string
	// BasePath is the URL prefix the SPA + API are mounted under, e.g. /xlearn
	// (env BASE_PATH). Always a leading slash and no trailing slash.
	BasePath string
}

// Load reads the environment and returns a validated, normalised Config.
func Load() Config {
	return Config{
		Port:     env("PORT", defaultPort),
		LogLevel: env("LOG_LEVEL", defaultLogLevel),
		BasePath: normalizeBasePath(env("BASE_PATH", defaultBasePath)),
	}
}

// Addr is the listen address for http.Server (":8080").
func (c Config) Addr() string { return ":" + c.Port }

// normalizeBasePath guarantees a leading slash and strips any trailing slash so
// path prefixing is unambiguous. "" and "/" both normalise to "" (mounted at
// root), which the gateway treats as "no prefix".
func normalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
