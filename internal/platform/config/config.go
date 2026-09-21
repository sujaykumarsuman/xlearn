// Package config resolves a service's runtime settings from the environment
// (12-factor, per ADR-0009). Every knob has a documented default so the binary
// runs with no configuration in local dev; the k8s deployment sets what differs
// via env. A resolved Config is an immutable snapshot.
package config

import (
	"os"
	"strings"
	"time"
)

// Defaults for the gateway. Each is overridable by the matching env var.
const (
	defaultPort               = "8080"
	defaultLogLevel           = "info"
	defaultBasePath           = "/xlearn"
	defaultIdentityBaseURL    = "http://localhost:8081"
	defaultCurriculumBaseURL  = "http://localhost:8082"
	defaultPracticeBaseURL    = "http://localhost:8083"
	defaultReviewBaseURL      = "http://localhost:8084"
	defaultAssessmentBaseURL  = "http://localhost:8085"
	defaultJWTIssuer          = "xlearn-gateway"
	defaultJWTAudIdentity     = "identity"
	defaultJWTAudPractice     = "practice"
	defaultJWTAudReview       = "review"
	defaultJWTAudAssessment   = "assessment"
	defaultJWTTTL             = 5 * time.Minute
	defaultSessionCookieScope = "/xlearn"
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
	// IdentityBaseURL is the internal ClusterIP URL of the identity service
	// (env IDENTITY_BASE_URL), e.g. http://xlearn-identity.xlearn.svc.cluster.local:8081.
	IdentityBaseURL string
	// CurriculumBaseURL is the internal ClusterIP URL of the curriculum service
	// (env CURRICULUM_BASE_URL), e.g. http://xlearn-curriculum.xlearn.svc.cluster.local:8082.
	CurriculumBaseURL string
	// PracticeBaseURL is the internal ClusterIP URL of the practice service
	// (env PRACTICE_BASE_URL), e.g. http://xlearn-practice.xlearn.svc.cluster.local:8083.
	PracticeBaseURL string
	// ReviewBaseURL is the internal ClusterIP URL of the review service
	// (env REVIEW_BASE_URL), e.g. http://xlearn-review.xlearn.svc.cluster.local:8084.
	ReviewBaseURL string
	// AssessmentBaseURL is the internal ClusterIP URL of the assessment service
	// (env ASSESSMENT_BASE_URL), e.g. http://xlearn-assessment.xlearn.svc.cluster.local:8085.
	AssessmentBaseURL string
	// JWT holds the RS256 signing config for gateway-minted internal JWTs (ADR-0006).
	JWT JWTConfig
}

// JWTConfig configures the gateway's JWT minting + JWKS publication.
type JWTConfig struct {
	// PrivateKeyPEM is the RSA signing key PEM (env JWT_PRIVATE_KEY). Empty when
	// the key is supplied via PrivateKeyFile or generated ephemerally for dev.
	PrivateKeyPEM string
	// PrivateKeyFile is a path to the RSA signing key PEM (env JWT_PRIVATE_KEY_FILE).
	PrivateKeyFile string
	// Issuer is the "iss" claim on minted tokens (env JWT_ISSUER).
	Issuer string
	// AudienceIdentity is the "aud" for tokens forwarded to identity (env JWT_AUD_IDENTITY).
	AudienceIdentity string
	// AudiencePractice is the "aud" for tokens forwarded to practice (env JWT_AUD_PRACTICE).
	AudiencePractice string
	// AudienceReview is the "aud" for tokens forwarded to review (env JWT_AUD_REVIEW).
	AudienceReview string
	// AudienceAssessment is the "aud" for tokens forwarded to assessment (env JWT_AUD_ASSESSMENT).
	AudienceAssessment string
	// TTL is the token lifetime (env JWT_TTL, e.g. "5m").
	TTL time.Duration
	// AdditionalPublicKeysPEM holds extra RSA public keys (PEM, one or more blocks;
	// env JWT_ADDITIONAL_PUBLIC_KEYS) published in JWKS but not used for minting —
	// the outgoing/incoming keys during a rotation overlap window (ADR-0011).
	AdditionalPublicKeysPEM string
}

// Load reads the environment and returns a validated, normalised Config.
func Load() Config {
	return Config{
		Port:              env("PORT", defaultPort),
		LogLevel:          env("LOG_LEVEL", defaultLogLevel),
		BasePath:          normalizeBasePath(env("BASE_PATH", defaultBasePath)),
		IdentityBaseURL:   strings.TrimRight(env("IDENTITY_BASE_URL", defaultIdentityBaseURL), "/"),
		CurriculumBaseURL: strings.TrimRight(env("CURRICULUM_BASE_URL", defaultCurriculumBaseURL), "/"),
		PracticeBaseURL:   strings.TrimRight(env("PRACTICE_BASE_URL", defaultPracticeBaseURL), "/"),
		ReviewBaseURL:     strings.TrimRight(env("REVIEW_BASE_URL", defaultReviewBaseURL), "/"),
		AssessmentBaseURL: strings.TrimRight(env("ASSESSMENT_BASE_URL", defaultAssessmentBaseURL), "/"),
		JWT: JWTConfig{
			PrivateKeyPEM:           os.Getenv("JWT_PRIVATE_KEY"),
			PrivateKeyFile:          strings.TrimSpace(os.Getenv("JWT_PRIVATE_KEY_FILE")),
			Issuer:                  env("JWT_ISSUER", defaultJWTIssuer),
			AudienceIdentity:        env("JWT_AUD_IDENTITY", defaultJWTAudIdentity),
			AudiencePractice:        env("JWT_AUD_PRACTICE", defaultJWTAudPractice),
			AudienceReview:          env("JWT_AUD_REVIEW", defaultJWTAudReview),
			AudienceAssessment:      env("JWT_AUD_ASSESSMENT", defaultJWTAudAssessment),
			TTL:                     envDuration("JWT_TTL", defaultJWTTTL),
			AdditionalPublicKeysPEM: os.Getenv("JWT_ADDITIONAL_PUBLIC_KEYS"),
		},
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

func envDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
