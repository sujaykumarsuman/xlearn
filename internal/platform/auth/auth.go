// Package auth is the identity-propagation seam for xLearn services (ADR-0006): the
// gateway mints a short-TTL RS256 JWT from the validated session cookie and
// publishes its JWKS; downstream services verify the JWT against that JWKS. It is
// implemented with the Go standard library only (crypto/rsa, crypto/sha256,
// encoding/base64/json) — no third-party JWT dependency.
//
// Files: jwt.go (compact JWS encode/decode + Signer), jwks.go (JWK set + the
// fetch/cache verifier), keys.go (RSA PEM loading + RFC-7638 kid).
package auth

import (
	"context"
	"errors"
)

// SessionCookieName is the opaque server-session cookie the gateway reads to
// validate a request and the identity service sets on login (ADR-0006). Shared so
// the two services agree on the name without importing each other.
const SessionCookieName = "xl_session"

// Common verification errors. Callers map ErrUnauthenticated to the 401 envelope
// { "error": { "code": "unauthenticated" } } (api.md).
var (
	ErrUnauthenticated = errors.New("auth: unauthenticated")
	ErrTokenExpired    = errors.New("auth: token expired")
	ErrBadAudience     = errors.New("auth: wrong audience")
	ErrBadIssuer       = errors.New("auth: wrong issuer")
	ErrUnknownKey      = errors.New("auth: unknown signing key (kid)")
	ErrBadSignature    = errors.New("auth: bad signature")
	ErrMalformedToken  = errors.New("auth: malformed token")
)

// Claims is the verified identity carried by a gateway-minted JWT.
type Claims struct {
	// Subject is the account id ("sub").
	Subject string
	// Roles is the caller's role set (v1: a single "learner").
	Roles []string
	// Audience is the intended downstream service ("aud").
	Audience string
}

// Verifier verifies a gateway-minted JWT and returns its claims. Downstream
// services depend on this seam; the gateway provides the concrete JWKS-backed
// implementation via JWKSVerifier.
type Verifier interface {
	// Verify parses and cryptographically verifies token, returning its claims
	// or an error if the token is missing, malformed, expired, or untrusted.
	Verify(ctx context.Context, token string) (Claims, error)
}

// Minter exchanges a validated session for a short-lived downstream JWT. Only the
// gateway holds the signing key; implemented by Signer.
type Minter interface {
	// Mint issues a signed JWT for subject, scoped to audience.
	Mint(ctx context.Context, subject, audience string, roles []string) (token string, err error)
}
