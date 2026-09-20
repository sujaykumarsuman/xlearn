package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
)

// ParseRSAPrivateKeyPEM loads an RSA private key from PEM (PKCS#1 "RSA PRIVATE
// KEY" or PKCS#8 "PRIVATE KEY"). The gateway loads its signing key this way from
// the SOPS-managed xlearn-jwt secret (ADR-0006).
func ParseRSAPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("auth: no PEM block in key data")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("auth: parse pkcs8: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("auth: pkcs8 key is %T, want *rsa.PrivateKey", key)
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("auth: unsupported PEM block type %q", block.Type)
	}
}

// ParseRSAPublicKeysPEM loads one or more RSA public keys from PEM (PKIX "PUBLIC
// KEY" or PKCS#1 "RSA PUBLIC KEY"); several keys may be concatenated. Used to give
// the gateway extra verification keys during a signing-key rotation (ADR-0011).
func ParseRSAPublicKeysPEM(pemBytes []byte) ([]*rsa.PublicKey, error) {
	var keys []*rsa.PublicKey
	rest := pemBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "PUBLIC KEY":
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("auth: parse pkix public key: %w", err)
			}
			rsaPub, ok := pub.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("auth: public key is %T, want *rsa.PublicKey", pub)
			}
			keys = append(keys, rsaPub)
		case "RSA PUBLIC KEY":
			rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("auth: parse pkcs1 public key: %w", err)
			}
			keys = append(keys, rsaPub)
		default:
			return nil, fmt.Errorf("auth: unsupported PEM block type %q", block.Type)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("auth: no public keys in PEM data")
	}
	return keys, nil
}

// JWKSHandler serves a signer's JWK Set at /.well-known/jwks.json. The public key
// is not secret; a short cache is safe and reduces refetch load on services.
func JWKSHandler(s *Signer) http.HandlerFunc {
	body, err := s.JWKS()
	return func(w http.ResponseWriter, r *http.Request) {
		if err != nil {
			http.Error(w, `{"error":{"code":"internal"}}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(body)
	}
}
