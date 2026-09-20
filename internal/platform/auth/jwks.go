package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// jwk is one RSA verification key in a JWK Set (RFC 7517).
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use,omitempty"`
	Alg string `json:"alg,omitempty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwkSet is the JWKS document served at /.well-known/jwks.json.
type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type publicKey struct {
	kid string
	key *rsa.PublicKey
}

// marshalJWKS renders public keys as a JWK Set. Multiple keys support rotation
// (old + new published together during the overlap window).
func marshalJWKS(keys []publicKey) ([]byte, error) {
	set := jwkSet{Keys: make([]jwk, 0, len(keys))}
	for _, k := range keys {
		set.Keys = append(set.Keys, jwk{
			Kty: "RSA",
			Kid: k.kid,
			Use: "sig",
			Alg: "RS256",
			N:   encodeModulus(k.key.N),
			E:   encodeExponent(k.key.E),
		})
	}
	return json.Marshal(set)
}

func encodeModulus(n *big.Int) string { return b64(n.Bytes()) }

func encodeExponent(e int) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(e))
	i := 0
	for i < len(buf)-1 && buf[i] == 0 {
		i++
	}
	return b64(buf[i:])
}

func (k jwk) publicKey() (*rsa.PublicKey, error) {
	if k.Kty != "RSA" {
		return nil, fmt.Errorf("auth: unsupported key type %q", k.Kty)
	}
	nb, err := unb64(k.N)
	if err != nil {
		return nil, fmt.Errorf("auth: bad modulus: %w", err)
	}
	eb, err := unb64(k.E)
	if err != nil {
		return nil, fmt.Errorf("auth: bad exponent: %w", err)
	}
	e := new(big.Int).SetBytes(eb)
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(e.Int64())}, nil
}

// JWKSVerifier verifies RS256 JWTs against a remote JWKS, caching keys by kid. It
// tolerates key rotation: a set may hold several keys (old + new), and an unknown
// kid triggers a rate-limited refetch. Implements Verifier.
type JWKSVerifier struct {
	url      string
	aud      string
	iss      string
	leeway   time.Duration
	cacheTTL time.Duration
	minFetch time.Duration
	httpc    *http.Client
	now      func() time.Time

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	lastTry   time.Time
}

// VerifierOption configures a JWKSVerifier.
type VerifierOption func(*JWKSVerifier)

// WithLeeway sets the clock-skew allowance (ADR-0006). Default 30s.
func WithLeeway(d time.Duration) VerifierOption { return func(v *JWKSVerifier) { v.leeway = d } }

// WithHTTPClient overrides the HTTP client used to fetch the JWKS.
func WithHTTPClient(c *http.Client) VerifierOption { return func(v *JWKSVerifier) { v.httpc = c } }

// WithCacheTTL sets how long a fetched JWKS is trusted before a proactive refetch.
func WithCacheTTL(d time.Duration) VerifierOption { return func(v *JWKSVerifier) { v.cacheTTL = d } }

// NewJWKSVerifier builds a verifier for the JWKS at url, requiring the given
// audience (and issuer, when non-empty).
func NewJWKSVerifier(url, audience, issuer string, opts ...VerifierOption) *JWKSVerifier {
	v := &JWKSVerifier{
		url:      url,
		aud:      audience,
		iss:      issuer,
		leeway:   30 * time.Second,
		cacheTTL: time.Hour,
		minFetch: 30 * time.Second,
		httpc:    &http.Client{Timeout: 5 * time.Second},
		now:      time.Now,
		keys:     map[string]*rsa.PublicKey{},
	}
	for _, o := range opts {
		o(v)
	}
	return v
}

var _ Verifier = (*JWKSVerifier)(nil)

// Verify parses, cryptographically verifies, and validates the claims of token.
func (v *JWKSVerifier) Verify(ctx context.Context, token string) (Claims, error) {
	h, c, signingInput, sig, err := parse(token)
	if err != nil {
		return Claims{}, err
	}
	if h.Alg != "RS256" {
		return Claims{}, ErrMalformedToken
	}
	key, err := v.keyFor(ctx, h.Kid)
	if err != nil {
		return Claims{}, err
	}
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return Claims{}, ErrBadSignature
	}
	if err := verifyClaims(c, v.aud, v.iss, v.now(), v.leeway); err != nil {
		return Claims{}, err
	}
	return Claims{Subject: c.Sub, Roles: c.Roles, Audience: c.Aud}, nil
}

// keyFor returns the cached key for kid, refetching the JWKS when the cache is
// cold/stale or the kid is unknown (rotation), rate-limited by minFetch.
func (v *JWKSVerifier) keyFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := v.now().Sub(v.fetchedAt) < v.cacheTTL
	v.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}

	if err := v.refresh(ctx, ok); err != nil && !ok {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	if key, ok := v.keys[kid]; ok {
		return key, nil
	}
	return nil, ErrUnknownKey
}

// refresh refetches the JWKS. When haveStale is true (we still hold a usable key)
// the refetch is rate-limited and best-effort; otherwise it is forced.
func (v *JWKSVerifier) refresh(ctx context.Context, haveStale bool) error {
	v.mu.Lock()
	if haveStale && v.now().Sub(v.lastTry) < v.minFetch {
		v.mu.Unlock()
		return nil
	}
	v.lastTry = v.now()
	v.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.url, nil)
	if err != nil {
		return fmt.Errorf("auth: build jwks request: %w", err)
	}
	resp, err := v.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("auth: fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: jwks status %d", resp.StatusCode)
	}
	var set jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("auth: decode jwks: %w", err)
	}
	next := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		pk, err := k.publicKey()
		if err != nil {
			continue // skip unusable keys rather than failing the whole set
		}
		next[k.Kid] = pk
	}
	v.mu.Lock()
	v.keys = next
	v.fetchedAt = v.now()
	v.mu.Unlock()
	return nil
}
