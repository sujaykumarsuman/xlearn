package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DefaultTTL is the JWT lifetime (ADR-0006: ~5 min short-TTL).
const DefaultTTL = 5 * time.Minute

// header is the JWS protected header (RS256 only).
type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// claims is the JWT payload: registered claims plus the xLearn roles claim. aud is
// a single string (one JWT per downstream audience, ADR-0006).
type claims struct {
	Sub   string   `json:"sub"`
	Aud   string   `json:"aud"`
	Iss   string   `json:"iss,omitempty"`
	Iat   int64    `json:"iat"`
	Nbf   int64    `json:"nbf"`
	Exp   int64    `json:"exp"`
	Roles []string `json:"roles,omitempty"`
}

// Signer mints RS256 JWTs. Only the gateway constructs one (it holds the private
// key). It implements Minter.
type Signer struct {
	priv      *rsa.PrivateKey
	kid       string
	issuer    string
	ttl       time.Duration
	now       func() time.Time
	extraPubs []*rsa.PublicKey // additional public keys published in JWKS (rotation)
}

// NewSigner builds a Signer from an RSA private key. The kid is derived from the
// public key's RFC-7638 thumbprint so a rotated key gets a new kid automatically.
func NewSigner(priv *rsa.PrivateKey, issuer string, ttl time.Duration) *Signer {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Signer{
		priv:   priv,
		kid:    thumbprint(&priv.PublicKey),
		issuer: issuer,
		ttl:    ttl,
		now:    time.Now,
	}
}

// KID returns the signing key id (its JWK thumbprint).
func (s *Signer) KID() string { return s.kid }

// Public returns the corresponding public key (for JWKS publication).
func (s *Signer) Public() *rsa.PublicKey { return &s.priv.PublicKey }

// Mint issues a signed JWT for subject scoped to audience.
func (s *Signer) Mint(_ context.Context, subject, audience string, roles []string) (string, error) {
	now := s.now()
	c := claims{
		Sub:   subject,
		Aud:   audience,
		Iss:   s.issuer,
		Iat:   now.Unix(),
		Nbf:   now.Unix(),
		Exp:   now.Add(s.ttl).Unix(),
		Roles: roles,
	}
	return sign(s.priv, s.kid, c)
}

// AddVerificationKeys registers additional public keys to publish in JWKS without
// minting with them. During a key rotation the gateway publishes the outgoing and
// incoming public keys together (the overlap window) so downstream verifiers accept
// tokens signed by either kid while replicas roll (ADR-0006, ADR-0011).
func (s *Signer) AddVerificationKeys(pubs ...*rsa.PublicKey) {
	for _, p := range pubs {
		if p != nil {
			s.extraPubs = append(s.extraPubs, p)
		}
	}
}

// JWKS returns the JWK Set for this signer's public key plus any additional
// verification keys, deduped by kid.
func (s *Signer) JWKS() ([]byte, error) {
	keys := []publicKey{{kid: s.kid, key: &s.priv.PublicKey}}
	seen := map[string]bool{s.kid: true}
	for _, p := range s.extraPubs {
		kid := thumbprint(p)
		if seen[kid] {
			continue
		}
		seen[kid] = true
		keys = append(keys, publicKey{kid: kid, key: p})
	}
	return marshalJWKS(keys)
}

var _ Minter = (*Signer)(nil)

// sign builds the compact JWS: base64url(header).base64url(payload).base64url(sig).
func sign(priv *rsa.PrivateKey, kid string, c claims) (string, error) {
	h := header{Alg: "RS256", Typ: "JWT", Kid: kid}
	hb, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}
	pb, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	signingInput := b64(hb) + "." + b64(pb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	return signingInput + "." + b64(sig), nil
}

// parse splits and decodes a compact JWS, returning the header, claims and the
// raw signing input + signature for verification. It performs no crypto.
func parse(token string) (header, claims, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	hb, err := unb64(parts[0])
	if err != nil {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	var h header
	if err := json.Unmarshal(hb, &h); err != nil {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	pb, err := unb64(parts[1])
	if err != nil {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	var c claims
	if err := json.Unmarshal(pb, &c); err != nil {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	sig, err := unb64(parts[2])
	if err != nil {
		return header{}, claims{}, "", nil, ErrMalformedToken
	}
	return h, c, parts[0] + "." + parts[1], sig, nil
}

// verifyClaims checks aud/iss/exp/nbf with clock-skew leeway.
func verifyClaims(c claims, wantAud, wantIss string, now time.Time, leeway time.Duration) error {
	if wantAud != "" && c.Aud != wantAud {
		return ErrBadAudience
	}
	if wantIss != "" && c.Iss != wantIss {
		return ErrBadIssuer
	}
	if c.Exp > 0 && now.After(time.Unix(c.Exp, 0).Add(leeway)) {
		return ErrTokenExpired
	}
	if c.Nbf > 0 && now.Add(leeway).Before(time.Unix(c.Nbf, 0)) {
		return ErrUnauthenticated
	}
	return nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

// thumbprint is the RFC-7638 JWK thumbprint of an RSA public key, used as the kid.
func thumbprint(pub *rsa.PublicKey) string {
	// Canonical members, lexicographically ordered: e, kty, n.
	jwk := fmt.Sprintf(`{"e":"%s","kty":"RSA","n":"%s"}`, encodeExponent(pub.E), encodeModulus(pub.N))
	sum := sha256.Sum256([]byte(jwk))
	return b64(sum[:])
}
