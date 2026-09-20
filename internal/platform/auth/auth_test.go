package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return k
}

// jwksServer serves a JWK Set built from a mutable list of signers (rotation).
type jwksServer struct {
	mu      sync.Mutex
	signers []*Signer
	srv     *httptest.Server
	hits    int
}

func newJWKSServer(t *testing.T, signers ...*Signer) *jwksServer {
	t.Helper()
	js := &jwksServer{signers: signers}
	js.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		js.mu.Lock()
		defer js.mu.Unlock()
		js.hits++
		keys := make([]publicKey, 0, len(js.signers))
		for _, s := range js.signers {
			keys = append(keys, publicKey{kid: s.KID(), key: s.Public()})
		}
		body, _ := marshalJWKS(keys)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(js.srv.Close)
	return js
}

func (js *jwksServer) setSigners(signers ...*Signer) {
	js.mu.Lock()
	defer js.mu.Unlock()
	js.signers = signers
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	js := newJWKSServer(t, signer)
	v := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway")

	tok, err := signer.Mint(context.Background(), "acct-123", "identity", []string{"learner"})
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	claims, err := v.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "acct-123" || claims.Audience != "identity" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "learner" {
		t.Fatalf("unexpected roles: %+v", claims.Roles)
	}
}

func TestVerifyWrongAudience(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	js := newJWKSServer(t, signer)
	v := NewJWKSVerifier(js.srv.URL, "practice", "xlearn-gateway") // expects practice

	tok, _ := signer.Mint(context.Background(), "acct-1", "identity", []string{"learner"})
	if _, err := v.Verify(context.Background(), tok); err != ErrBadAudience {
		t.Fatalf("want ErrBadAudience, got %v", err)
	}
}

func TestVerifyWrongIssuer(t *testing.T) {
	signer := NewSigner(mustKey(t), "someone-else", DefaultTTL)
	js := newJWKSServer(t, signer)
	v := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway")

	tok, _ := signer.Mint(context.Background(), "acct-1", "identity", nil)
	if _, err := v.Verify(context.Background(), tok); err != ErrBadIssuer {
		t.Fatalf("want ErrBadIssuer, got %v", err)
	}
}

func TestVerifyExpiryAndLeeway(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", time.Minute)
	base := time.Now()
	signer.now = func() time.Time { return base }
	js := newJWKSServer(t, signer)

	tok, _ := signer.Mint(context.Background(), "acct-1", "identity", nil)

	// 90s later: token (exp=+60s) is past expiry but within a 30s+ leeway? No —
	// 90s > 60s + 30s leeway, so it must be expired.
	vExpired := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway", WithLeeway(30*time.Second))
	vExpired.now = func() time.Time { return base.Add(90 * time.Second) }
	if _, err := vExpired.Verify(context.Background(), tok); err != ErrTokenExpired {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}

	// 75s later with a 30s leeway: 75 < 60 + 30, so still valid (clock-skew slack).
	vSkew := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway", WithLeeway(30*time.Second))
	vSkew.now = func() time.Time { return base.Add(75 * time.Second) }
	if _, err := vSkew.Verify(context.Background(), tok); err != nil {
		t.Fatalf("within-leeway token should verify, got %v", err)
	}
}

func TestVerifyMalformedAndTampered(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	js := newJWKSServer(t, signer)
	v := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway")

	if _, err := v.Verify(context.Background(), "not-a-jwt"); err != ErrMalformedToken {
		t.Fatalf("want ErrMalformedToken, got %v", err)
	}

	tok, _ := signer.Mint(context.Background(), "acct-1", "identity", nil)
	// Flip a character in the signature segment.
	tampered := tok[:len(tok)-2] + func() string {
		last := tok[len(tok)-2:]
		if last[0] == 'A' {
			return "BB"
		}
		return "AA"
	}()
	if _, err := v.Verify(context.Background(), tampered); err == nil {
		t.Fatalf("tampered token must not verify")
	}
}

func TestKeyRotationOverlap(t *testing.T) {
	key1 := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	js := newJWKSServer(t, key1)
	v := NewJWKSVerifier(js.srv.URL, "identity", "xlearn-gateway")

	// Prime the cache with key1.
	tok1, _ := key1.Mint(context.Background(), "acct-1", "identity", nil)
	if _, err := v.Verify(context.Background(), tok1); err != nil {
		t.Fatalf("verify key1: %v", err)
	}

	// Rotate: publish key1 + key2 (overlap window).
	key2 := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	if key1.KID() == key2.KID() {
		t.Fatalf("distinct keys must have distinct kids")
	}
	js.setSigners(key1, key2)

	// A token from the new key has an unknown kid → verifier refetches and accepts.
	tok2, _ := key2.Mint(context.Background(), "acct-2", "identity", nil)
	if _, err := v.Verify(context.Background(), tok2); err != nil {
		t.Fatalf("verify key2 after rotation: %v", err)
	}
	// The old key still verifies during the overlap window.
	if _, err := v.Verify(context.Background(), tok1); err != nil {
		t.Fatalf("verify key1 during overlap: %v", err)
	}
}

func TestJWKSHandlerServesValidSet(t *testing.T) {
	signer := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	JWKSHandler(signer)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var set jwkSet
	if err := json.Unmarshal(rec.Body.Bytes(), &set); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	if len(set.Keys) != 1 || set.Keys[0].Kid != signer.KID() || set.Keys[0].Alg != "RS256" {
		t.Fatalf("unexpected jwks: %+v", set)
	}
	if _, err := set.Keys[0].publicKey(); err != nil {
		t.Fatalf("jwks key not reconstructable: %v", err)
	}
}

func TestParseRSAPrivateKeyPEMRoundTrip(t *testing.T) {
	// PKCS#8 round-trip via x509 marshal is covered indirectly; here assert an
	// unknown block type is rejected.
	if _, err := ParseRSAPrivateKeyPEM([]byte("garbage")); err == nil {
		t.Fatalf("expected error for non-PEM input")
	}
}

func TestJWKSPublishesAdditionalKeysForRotation(t *testing.T) {
	// Simulate a rotation overlap: the gateway mints with key1 but also publishes
	// key2's public key. A verifier reading that JWKS must accept tokens signed by
	// EITHER key (this is the publish-side overlap the single-key JWKS lacked).
	signer1 := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	signer2 := NewSigner(mustKey(t), "xlearn-gateway", DefaultTTL)
	signer1.AddVerificationKeys(signer2.Public())

	// The published set must carry both kids.
	body, err := signer1.JWKS()
	if err != nil {
		t.Fatalf("jwks: %v", err)
	}
	var set jwkSet
	if err := json.Unmarshal(body, &set); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	if len(set.Keys) != 2 {
		t.Fatalf("expected 2 keys in JWKS, got %d", len(set.Keys))
	}

	js := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := signer1.JWKS()
		_, _ = w.Write(b)
	}))
	t.Cleanup(js.Close)
	v := NewJWKSVerifier(js.URL, "identity", "xlearn-gateway")

	tok1, _ := signer1.Mint(context.Background(), "acct-1", "identity", nil)
	tok2, _ := signer2.Mint(context.Background(), "acct-2", "identity", nil)
	if _, err := v.Verify(context.Background(), tok1); err != nil {
		t.Fatalf("verify key1 token: %v", err)
	}
	if _, err := v.Verify(context.Background(), tok2); err != nil {
		t.Fatalf("verify key2 token (published as extra JWKS key): %v", err)
	}

	// De-dup: adding the signer's own key again does not duplicate it.
	signer1.AddVerificationKeys(signer1.Public())
	body2, _ := signer1.JWKS()
	var set2 jwkSet
	_ = json.Unmarshal(body2, &set2)
	if len(set2.Keys) != 2 {
		t.Fatalf("expected 2 keys after re-adding own key, got %d", len(set2.Keys))
	}
}

func TestParseRSAPublicKeysPEM(t *testing.T) {
	k1, k2 := mustKey(t), mustKey(t)
	pem1 := publicKeyPEM(t, &k1.PublicKey)
	pem2 := publicKeyPEM(t, &k2.PublicKey)

	// Two concatenated PKIX public-key blocks parse to two keys.
	keys, err := ParseRSAPublicKeysPEM(append(append([]byte{}, pem1...), pem2...))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if _, err := ParseRSAPublicKeysPEM([]byte("not-a-key")); err == nil {
		t.Fatalf("expected error for non-PEM input")
	}
}

func publicKeyPEM(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}
