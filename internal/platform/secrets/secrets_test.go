package secrets

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func newMaster(t *testing.T) []byte {
	t.Helper()
	m := make([]byte, MasterKeySize)
	if _, err := rand.Read(m); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return m
}

func TestSealOpenRoundTrip(t *testing.T) {
	c, err := NewCipher(newMaster(t))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	secret := []byte("sk-ant-api03-super-secret-value-3f2a")
	encKey, encDataKey, err := c.Seal(secret)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	// The ciphertext must not contain the plaintext (it is actually encrypted).
	if bytes.Contains(encKey, secret) {
		t.Fatal("enc_key contains the plaintext")
	}
	got, err := c.Open(encKey, encDataKey)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("round trip = %q, want %q", got, secret)
	}
}

func TestSealIsNondeterministic(t *testing.T) {
	c, _ := NewCipher(newMaster(t))
	secret := []byte("the same secret")
	a1, b1, _ := c.Seal(secret)
	a2, b2, _ := c.Seal(secret)
	// Fresh data key + fresh nonces → distinct ciphertexts each time.
	if bytes.Equal(a1, a2) || bytes.Equal(b1, b2) {
		t.Fatal("seal is deterministic; expected fresh data key + nonce per call")
	}
}

func TestOpenWithWrongMasterFails(t *testing.T) {
	c1, _ := NewCipher(newMaster(t))
	c2, _ := NewCipher(newMaster(t))
	encKey, encDataKey, err := c1.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := c2.Open(encKey, encDataKey); err != ErrDecrypt {
		t.Fatalf("open with wrong master = %v, want ErrDecrypt", err)
	}
}

func TestOpenDetectsTampering(t *testing.T) {
	c, _ := NewCipher(newMaster(t))
	encKey, encDataKey, _ := c.Seal([]byte("secret"))
	// Flip a byte in the sealed key: the AEAD tag must reject it.
	tampered := append([]byte(nil), encKey...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := c.Open(tampered, encDataKey); err != ErrDecrypt {
		t.Fatalf("tampered open = %v, want ErrDecrypt", err)
	}
	// A truncated blob (shorter than the nonce) must also fail cleanly, not panic.
	if _, err := c.Open([]byte{0x01}, encDataKey); err != ErrDecrypt {
		t.Fatalf("truncated open = %v, want ErrDecrypt", err)
	}
}

func TestNewCipherRejectsBadKey(t *testing.T) {
	if _, err := NewCipher(nil); err != ErrEmptyMasterKey {
		t.Fatalf("nil key = %v, want ErrEmptyMasterKey", err)
	}
	if _, err := NewCipher([]byte("short")); err != ErrBadMasterKey {
		t.Fatalf("short key = %v, want ErrBadMasterKey", err)
	}
}

func TestMask(t *testing.T) {
	cases := map[string]string{
		"":                                     "",
		"short":                                "...",
		"sk-12345":                             "...", // 8 chars → fully redacted
		"sk-ant-api03-abcdefghijklmnop-3f2a":   "sk-...3f2a",
		"sk-proj-abcdefghijklmnopqrstuvwx1234": "sk-...1234",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Fatalf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
	// The mask must never contain the full secret.
	raw := "sk-ant-abcdefghijklmnopqrstuv"
	if strings.Contains(Mask(raw), raw) {
		t.Fatal("mask leaked the full key")
	}
}

func TestParseMasterKey(t *testing.T) {
	raw := newMaster(t)
	// base64 std
	if got, err := ParseMasterKey(base64.StdEncoding.EncodeToString(raw)); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("base64 std: got %x err %v", got, err)
	}
	// base64 raw-url (no padding)
	if got, err := ParseMasterKey(base64.RawURLEncoding.EncodeToString(raw)); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("base64 raw-url: got %x err %v", got, err)
	}
	// hex
	hexed := make([]byte, 0)
	for _, b := range raw {
		hexed = append(hexed, "0123456789abcdef"[b>>4], "0123456789abcdef"[b&0x0f])
	}
	if got, err := ParseMasterKey(string(hexed)); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("hex: got %x err %v", got, err)
	}
	// whitespace tolerated
	if _, err := ParseMasterKey("  " + base64.StdEncoding.EncodeToString(raw) + "\n"); err != nil {
		t.Fatalf("whitespace: %v", err)
	}
	// wrong length
	if _, err := ParseMasterKey(base64.StdEncoding.EncodeToString([]byte("too short"))); err != ErrBadMasterKey {
		t.Fatalf("short = %v, want ErrBadMasterKey", err)
	}
	if _, err := ParseMasterKey(""); err != ErrEmptyMasterKey {
		t.Fatalf("empty = %v, want ErrEmptyMasterKey", err)
	}
}

func TestZero(t *testing.T) {
	b := []byte{1, 2, 3, 4}
	Zero(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte %d = %d, want 0", i, v)
		}
	}
	Zero(nil) // must not panic
}
