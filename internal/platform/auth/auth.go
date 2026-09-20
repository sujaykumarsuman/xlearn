// Package auth is the identity-propagation seam for xLearn services: the gateway
// mints a short-TTL RS256 JWT from the validated session cookie, and downstream
// services verify it against the gateway's JWKS (ADR-0006). This sprint ships
// only the interfaces + types so callers can be written against a stable seam.
//
// TODO(S02): implement session-cookie validation (gateway), JWT minting from a
// SOPS-managed RSA key, JWKS publication, and RS256/JWKS verification
// (downstream). No implementation lives here yet — do not wire it into request
// handling until S02.
package auth

import "context"

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
// implementation in S02.
type Verifier interface {
	// Verify parses and cryptographically verifies token, returning its claims
	// or an error if the token is missing, malformed, expired, or untrusted.
	Verify(ctx context.Context, token string) (Claims, error)
}

// Minter exchanges a validated session for a short-lived downstream JWT. Only
// the gateway holds the signing key; implemented in S02.
type Minter interface {
	// Mint issues a signed JWT for subject, scoped to audience.
	Mint(ctx context.Context, subject, audience string, roles []string) (token string, err error)
}
